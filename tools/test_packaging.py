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
import os
from pathlib import Path
import stat
import subprocess
import tempfile
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
		self.assertIn("./usr/lib/systemd/system/memcp.service", listing)

	def test_rpm_artifact(self) -> None:
		self.assertTrue(self.rpm.is_file(), self.rpm)
		self.assertEqual(run("rpm", "-qp", "--qf", "%{NAME}", str(self.rpm)), "memcp")
		self.assertEqual(run("rpm", "-qp", "--qf", "%{VERSION}", str(self.rpm)), self.version)
		listing = run("rpm", "-qpl", str(self.rpm))
		self.assertIn("/usr/lib/memcp/initialize", listing)
		self.assertIn("/usr/lib/systemd/system/memcp.service", listing)

	def test_source_rpm_artifact(self) -> None:
		self.assertTrue(self.source_rpm.is_file(), self.source_rpm)
		self.assertEqual(run("rpm", "-qp", "--qf", "%{NAME}", str(self.source_rpm)), "memcp")


def main() -> int:
	parser = argparse.ArgumentParser()
	parser.add_argument("--artifacts", action="store_true", help="also inspect built DEB/RPM files")
	args, unittest_args = parser.parse_known_args()

	suite = unittest.defaultTestLoader.loadTestsFromTestCase(ReleaseSourceTests)
	if args.artifacts:
		suite.addTests(unittest.defaultTestLoader.loadTestsFromTestCase(BuiltArtifactTests))
	result = unittest.TextTestRunner(verbosity=2).run(suite)
	return 0 if result.wasSuccessful() else 1


if __name__ == "__main__":
	raise SystemExit(main())
