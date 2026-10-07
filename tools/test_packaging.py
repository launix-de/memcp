#!/usr/bin/env python3
# Copyright (C) 2026 Carl-Philip Haensch
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

"""Fast release-chain checks that do not start Docker or the SQL test suite."""

from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import stat
import subprocess
import sys
import tempfile
import textwrap
import unittest


ROOT = Path(__file__).resolve().parents[1]


def run(*args: str, cwd: Path = ROOT, env: dict[str, str] | None = None) -> str:
	result = subprocess.run(
		args,
		cwd=cwd,
		env=env,
		text=True,
		stdout=subprocess.PIPE,
		stderr=subprocess.STDOUT,
		check=True,
	)
	return result.stdout


class ReleaseSourceTests(unittest.TestCase):
	def apt_setup(self, scenario: str, packages: str = "mariadb-client rpm"):
		# Execute the action's actual shell body against private source files and
		# a fake privileged package client. GNU timeout still runs for real.
		action = (ROOT / ".github/actions/setup-apt/action.yml").read_text()
		body = textwrap.dedent(action.split("      run: |\n", 1)[1])
		with tempfile.TemporaryDirectory(prefix="memcp-ci-apt-") as tmp:
			root = Path(tmp)
			sources = root / "sources"
			(sources / "sources.list.d").mkdir(parents=True)
			legacy = "deb http://azure.archive.ubuntu.com/ubuntu noble main\n"
			deb822 = ("URIs: https://azure.archive.ubuntu.com/ubuntu\n"
				"Suites: noble-updates\nComponents: main universe\n"
				"Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg\n")
			unrelated = "deb https://example.invalid/ubuntu noble main\n"
			(sources / "sources.list").write_text(legacy)
			(sources / "sources.list.d/ubuntu.sources").write_text(deb822)
			(sources / "sources.list.d/other.list").write_text(unrelated)
			fake = root / "sudo"
			fake.write_text(f"#!{sys.executable}\n" + textwrap.dedent('''\
				import json, os, pathlib, sys, time
				root = pathlib.Path(os.environ['FAKE_APT_ROOT'])
				args = sys.argv[1:]
				if args[0] != 'apt-get':
				    os.execvp(args[0], args)
				with (root / 'calls.jsonl').open('a') as log:
				    log.write(json.dumps(args) + '\\n')
				scenario = os.environ['FAKE_APT_SCENARIO']
				if 'update' in args:
				    count = sum('update' in json.loads(line) for line in (root / 'calls.jsonl').read_text().splitlines())
				    if scenario == 'hang' and count == 1:
				        time.sleep(10)
				    if scenario == 'persistent' or (scenario == 'transient' and count == 1):
				        sys.exit(100)
				elif scenario == 'install-hang':
				    time.sleep(10)
				elif scenario == 'install-fail':
				    sys.exit(100)
			'''))
			fake.chmod(0o755)
			env = os.environ.copy()
			env.update(PATH=str(root) + os.pathsep + env['PATH'],
				PACKAGES=packages, APT_SOURCES_DIRECTORY=str(sources),
				INDEX_TIMEOUT_SECONDS="0.5" if scenario == "hang" else "2",
				INSTALL_TIMEOUT_SECONDS="0.5" if scenario == "install-hang" else "2",
				FAKE_APT_ROOT=str(root), FAKE_APT_SCENARIO=scenario)
			result = subprocess.run(["bash", "-c", body], env=env, text=True,
				stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=5)
			log = root / "calls.jsonl"
			calls = [json.loads(line) for line in log.read_text().splitlines()] if log.exists() else []
			return result, calls, (sources / "sources.list").read_text(), \
				(sources / "sources.list.d/ubuntu.sources").read_text(), \
				(sources / "sources.list.d/other.list").read_text(), legacy, deb822, unrelated

	def test_apt_success_preserves_sources_and_installs_every_requested_package(self):
		result, calls, legacy, deb822, unrelated, old_legacy, old_deb822, old_unrelated = self.apt_setup("success")
		self.assertEqual(result.returncode, 0, result.stdout)
		self.assertEqual((legacy, deb822, unrelated), (old_legacy, old_deb822, old_unrelated))
		self.assertEqual(len(calls), 2)
		self.assertEqual(calls[-1][-4:], ["install", "-y", "mariadb-client", "rpm"])
		self.assertIn("--error-on=any", calls[0])
		for args in calls:
			self.assertIn("Acquire::Retries=3", args)
			self.assertIn("Acquire::http::Timeout=30", args)
			self.assertIn("Acquire::https::Timeout=30", args)

	def test_apt_recovers_once_without_changing_repository_trust(self):
		result, calls, legacy, deb822, unrelated, old_legacy, old_deb822, old_unrelated = self.apt_setup("transient")
		self.assertEqual(result.returncode, 0, result.stdout)
		self.assertEqual(len(calls), 3)
		self.assertEqual(legacy, old_legacy.replace("http://azure.archive.ubuntu.com", "https://archive.ubuntu.com"))
		self.assertEqual(deb822, old_deb822.replace("https://azure.archive.ubuntu.com", "https://archive.ubuntu.com"))
		self.assertEqual(unrelated, old_unrelated)
		self.assertIn("Signed-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg", deb822)

	def test_hung_apt_request_is_terminated_before_bounded_recovery(self):
		result, calls, *_ = self.apt_setup("hang")
		self.assertEqual(result.returncode, 0, result.stdout)
		self.assertEqual(len(calls), 3)
		self.assertIn("APT index download failed", result.stdout)

	def test_persistent_apt_failure_stops_before_installation(self):
		result, calls, *_ = self.apt_setup("persistent")
		self.assertNotEqual(result.returncode, 0)
		self.assertEqual(len(calls), 2)
		self.assertTrue(all("update" in call for call in calls))
		self.assertIn("INFRASTRUCTURE_FAILURE", result.stdout)

	def test_failed_apt_install_is_not_accepted_or_retried(self):
		result, calls, *_ = self.apt_setup("install-fail")
		self.assertNotEqual(result.returncode, 0)
		self.assertEqual(len(calls), 2)
		self.assertIn("required package installation failed", result.stdout)

	def test_empty_package_list_never_runs_apt(self):
		result, calls, *_ = self.apt_setup("success", packages="")
		self.assertNotEqual(result.returncode, 0)
		self.assertEqual(calls, [])

	def test_hung_apt_install_is_terminated_and_fails(self):
		result, calls, *_ = self.apt_setup("install-hang")
		self.assertNotEqual(result.returncode, 0)
		self.assertEqual(len(calls), 2)
		self.assertIn("required package installation failed", result.stdout)

	def test_multiline_package_list_installs_every_package(self):
		result, calls, *_ = self.apt_setup("success", packages="mariadb-client\nrpm")
		self.assertEqual(result.returncode, 0, result.stdout)
		self.assertEqual(calls[-1][-2:], ["mariadb-client", "rpm"])

	def test_package_input_cannot_disable_authentication(self):
		result, calls, *_ = self.apt_setup("success", packages="rpm --allow-unauthenticated")
		self.assertNotEqual(result.returncode, 0)
		self.assertEqual(calls, [])

	def test_shell_scripts_parse(self) -> None:
		scripts = [
			"debian/postinst",
			"debian/prerm",
			"debian/postrm",
			"packaging/initialize.sh",
			"packaging/docker-entrypoint.sh",
		]
		run("sh", "-n", *scripts)

	def test_removing_packages_never_removes_database_data(self) -> None:
		for name in ("debian/postrm", "memcp.spec"):
			contents = (ROOT / name).read_text(encoding="utf-8")
			self.assertNotRegex(contents, r"rm\s+-[^\n]*r[^\n]*/var/lib/memcp")

	def test_docker_context_is_allow_listed(self) -> None:
		dockerignore = (ROOT / ".dockerignore").read_text(encoding="utf-8")
		self.assertIn("**", dockerignore.splitlines())
		dockerfile = (ROOT / "Dockerfile").read_text(encoding="utf-8")
		self.assertNotRegex(dockerfile, r"(?m)^COPY\s+\.\s+\.")
		self.assertRegex(dockerfile, r"(?m)^USER\s+10001:10001$")

	def test_dependencies_are_not_vendored(self) -> None:
		self.assertFalse((ROOT / "third_party").exists())
		self.assertFalse((ROOT / "vendor").exists())
		for name in ("Dockerfile", ".dockerignore"):
			self.assertNotIn("third_party", (ROOT / name).read_text(encoding="utf-8"))

		go_mod = (ROOT / "go.mod").read_text(encoding="utf-8")
		for line in go_mod.splitlines():
			if "=>" not in line:
				continue
			target = line.split("=>", 1)[1].strip().split()[0]
			self.assertFalse(
				target.startswith((".", "/")),
				f"go.mod uses local dependency replacement: {line.strip()}",
			)

	def test_initializer_is_idempotent_and_keeps_credential(self) -> None:
		with tempfile.TemporaryDirectory(prefix="memcp-package-test-") as tmp:
			root = Path(tmp)
			data = root / "data"
			config = root / "memcp.conf"
			credential = root / "initial-root-password"
			binary = root / "fake-memcp"
			config.write_text(f"-data {data}\n", encoding="utf-8")
			binary.write_text(
				"#!/bin/sh\n"
				"data=\n"
				"while [ \"$#\" -gt 0 ]; do\n"
				"  if [ \"$1\" = -data ]; then shift; data=$1; fi\n"
				"  shift\n"
				"done\n"
				"mkdir -p \"$data/system\"\n"
				"exit 0\n",
				encoding="utf-8",
			)
			binary.chmod(0o755)
			env = os.environ.copy()
			env.update(
				{
					"MEMCP_CONFIG": str(config),
					"MEMCP_BINARY": str(binary),
					"MEMCP_INITIAL_PASSWORD_FILE": str(credential),
					"MEMCP_RUN_USER": "",
				}
			)
			run("sh", "packaging/initialize.sh", env=env)
			password = credential.read_text(encoding="utf-8").strip()
			self.assertRegex(password, r"^[0-9a-f]{48}$")
			self.assertEqual(stat.S_IMODE(credential.stat().st_mode), 0o600)
			self.assertTrue((data / "system").is_dir())
			run("sh", "packaging/initialize.sh", env=env)
			self.assertEqual(credential.read_text(encoding="utf-8").strip(), password)


class BuiltArtifactTests(unittest.TestCase):
	@classmethod
	def setUpClass(cls) -> None:
		cls.version = run("make", "-s", "version").strip()
		cls.deb = ROOT / "dist" / f"memcp_{cls.version}_amd64.deb"
		cls.rpm = ROOT / "dist" / f"memcp_{cls.version}_x86_64.rpm"
		cls.source_rpm = ROOT / "dist" / f"memcp_{cls.version}.src.rpm"

	def test_debian_artifact(self) -> None:
		self.assertTrue(self.deb.is_file(), self.deb)
		self.assertEqual(run("dpkg-deb", "-f", str(self.deb), "Package").strip(), "memcp")
		self.assertEqual(run("dpkg-deb", "-f", str(self.deb), "Version").strip(), self.version)
		listing = run("dpkg-deb", "-c", str(self.deb))
		self.assertIn("./usr/lib/memcp/initialize", listing)
		self.assertIn("./usr/lib/memcp/php/libphp.so", listing)
		self.assertIn("./usr/lib/memcp/php/extensions/imagick.so", listing)
		self.assertIn("./usr/lib/memcp/php/php.ini", listing)
		self.assertIn("./usr/share/doc/memcp/php/LICENSE", listing)
		dependencies = run("dpkg-deb", "-f", str(self.deb), "Depends")
		self.assertIn("libc6", dependencies)
		self.assertIn("./usr/lib/systemd/system/memcp.service", listing)

	def test_rpm_artifact(self) -> None:
		self.assertTrue(self.rpm.is_file(), self.rpm)
		self.assertEqual(run("rpm", "-qp", "--qf", "%{NAME}", str(self.rpm)), "memcp")
		self.assertEqual(run("rpm", "-qp", "--qf", "%{VERSION}", str(self.rpm)), self.version)
		listing = run("rpm", "-qpl", str(self.rpm))
		self.assertIn("/usr/lib/memcp/initialize", listing)
		self.assertIn("/usr/lib/memcp/php/libphp.so", listing)
		self.assertIn("/usr/lib/memcp/php/extensions/imagick.so", listing)
		self.assertIn("/usr/lib/memcp/php/php.ini", listing)
		self.assertIn("/usr/share/doc/memcp/php/LICENSE", listing)
		self.assertIn("libc.so.6", run("rpm", "-qp", "--requires", str(self.rpm)))
		self.assertIn("/usr/lib/systemd/system/memcp.service", listing)

	def test_source_rpm_artifact(self) -> None:
		self.assertTrue(self.source_rpm.is_file(), self.source_rpm)
		self.assertEqual(run("rpm", "-qp", "--qf", "%{NAME}", str(self.source_rpm)), "memcp")


def main() -> int:
	parser = argparse.ArgumentParser()
	parser.add_argument("--artifacts", action="store_true", help="also inspect built DEB/RPM files")
	parser.add_argument("--format", choices=("all", "deb", "rpm"), default="all")
	args, unittest_args = parser.parse_known_args()

	suite = unittest.defaultTestLoader.loadTestsFromTestCase(ReleaseSourceTests)
	if args.artifacts:
		for name in unittest.defaultTestLoader.getTestCaseNames(BuiltArtifactTests):
			if args.format == "all" or (args.format == "deb") == (name == "test_debian_artifact"):
				suite.addTest(BuiltArtifactTests(name))
	result = unittest.TextTestRunner(verbosity=2).run(suite)
	return 0 if result.wasSuccessful() else 1


if __name__ == "__main__":
	raise SystemExit(main())
