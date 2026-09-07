#!/usr/bin/env python3
# Copyright (C) 2026 Carl-Philip Haensch
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

"""Destructive crash, recovery, restore, and soak drills for MemCP.

The drill owns every process and data directory it touches. It never attaches
to an existing server. Artifacts are retained so a failed run can be replayed.
"""

from __future__ import annotations

import argparse
import base64
import json
import os
from pathlib import Path
import random
import shutil
import signal
import socket
import subprocess
import sys
import threading
import time
import urllib.error
import urllib.parse
import urllib.request


ROOT = Path(__file__).resolve().parents[1]
DATABASE = "reliability_drill"


class DrillFailure(RuntimeError):
	pass


def free_port() -> int:
	with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
		sock.bind(("127.0.0.1", 0))
		return int(sock.getsockname()[1])


def make_artifact_dir(requested: Path | None) -> Path:
	if requested is None:
		stamp = time.strftime("%Y%m%d-%H%M%S")
		requested = Path("/tmp") / f"memcp-reliability-{stamp}-{os.getpid()}"
	requested = requested.expanduser().resolve()
	if requested.exists():
		raise DrillFailure(f"artifact directory already exists: {requested}")
	requested.mkdir(parents=True)
	return requested


class HttpClient:
	def __init__(self, port: int, timeout: float) -> None:
		self.port = port
		self.timeout = timeout
		auth = base64.b64encode(b"root:admin").decode("ascii")
		self.headers = {
			"Authorization": f"Basic {auth}",
			"Content-Type": "text/plain; charset=utf-8",
		}

	def request(self, route: str, body: str, session: str | None = None,
			timeout: float | None = None) -> str:
		headers = dict(self.headers)
		if session:
			headers["X-Session-Id"] = session
		request = urllib.request.Request(
			f"http://127.0.0.1:{self.port}{route}",
			data=body.encode("utf-8"), headers=headers, method="POST",
		)
		try:
			with urllib.request.urlopen(request, timeout=timeout or self.timeout) as response:
				return response.read().decode("utf-8")
		except urllib.error.HTTPError as error:
			payload = error.read().decode("utf-8", errors="replace")
			raise DrillFailure(f"request failed ({error.code}): {body}\n{payload}") from error

	def sql(self, statement: str, database: str = DATABASE,
			session: str | None = None, timeout: float | None = None) -> list[dict]:
		route = "/sql/" + urllib.parse.quote(database, safe="")
		payload = self.request(route, statement, session=session, timeout=timeout)
		rows = []
		for line in payload.splitlines():
			if line.strip():
				rows.append(json.loads(line))
		return rows

	def scm(self, expression: str, timeout: float | None = None) -> str:
		return self.request("/scm", expression, timeout=timeout)


class OwnedServer:
	def __init__(self, binary: Path, app: Path, data_dir: Path, artifact_dir: Path,
			timeout: float) -> None:
		self.binary = binary
		self.app = app
		self.data_dir = data_dir
		self.artifact_dir = artifact_dir
		self.timeout = timeout
		self.generation = 0
		self.process: subprocess.Popen[bytes] | None = None
		self.log_handle = None
		self.client: HttpClient | None = None

	def start(self) -> HttpClient:
		if self.process is not None:
			raise DrillFailure("refusing to start a second process under one server owner")
		self.generation += 1
		api_port = free_port()
		mysql_port = free_port()
		log_path = self.artifact_dir / f"memcp-generation-{self.generation}.log"
		self.log_handle = log_path.open("wb")
		command = [
			str(self.binary), "-data", str(self.data_dir),
			f"--api-port={api_port}", f"--mysql-port={mysql_port}",
			"--disable-mysql", "--no-repl", str(self.app),
		]
		self.process = subprocess.Popen(
			command, cwd=ROOT, stdin=subprocess.DEVNULL,
			stdout=self.log_handle, stderr=subprocess.STDOUT,
			start_new_session=True,
		)
		self.client = HttpClient(api_port, self.timeout)
		deadline = time.monotonic() + self.timeout
		while time.monotonic() < deadline:
			if self.process.poll() is not None:
				raise DrillFailure(
					f"MemCP exited during startup; inspect {log_path}"
				)
			try:
				self.client.sql("SELECT 1", database="system")
				return self.client
			except (OSError, DrillFailure, json.JSONDecodeError):
				time.sleep(0.05)
		raise DrillFailure(f"MemCP did not become ready; inspect {log_path}")

	def kill(self) -> None:
		process = self._take_process()
		if process.poll() is None:
			# Kill only the exact child created by this object. Never use pkill or a
			# port-pattern lookup in this destructive reliability tool.
			os.kill(process.pid, signal.SIGKILL)
		process.wait(timeout=self.timeout)
		self._close_log()

	def stop(self) -> None:
		process = self._take_process()
		if process.poll() is None:
			process.send_signal(signal.SIGTERM)
			try:
				process.wait(timeout=self.timeout)
			except subprocess.TimeoutExpired as error:
				os.kill(process.pid, signal.SIGKILL)
				process.wait(timeout=self.timeout)
				raise DrillFailure("graceful shutdown timed out and required SIGKILL") from error
		if process.returncode != 0:
			raise DrillFailure(f"graceful MemCP shutdown returned {process.returncode}")
		self._close_log()

	def cleanup(self) -> None:
		if self.process is None:
			return
		try:
			self.kill()
		except (OSError, subprocess.SubprocessError):
			self._close_log()

	def _take_process(self) -> subprocess.Popen[bytes]:
		if self.process is None:
			raise DrillFailure("no owned MemCP process is running")
		process = self.process
		self.process = None
		self.client = None
		return process

	def _close_log(self) -> None:
		if self.log_handle is not None:
			self.log_handle.close()
			self.log_handle = None


def scalar(client: HttpClient, statement: str) -> int:
	rows = client.sql(statement)
	if len(rows) != 1 or "value" not in rows[0]:
		raise DrillFailure(f"expected one scalar value for {statement}, got {rows}")
	return int(rows[0]["value"] or 0)


def atomic_signature(client: HttpClient) -> dict[str, int]:
	rows = client.sql(
		"SELECT COUNT(*) AS row_count, MIN(x) AS min_x, MAX(x) AS max_x, "
		"COALESCE(SUM(x), 0) AS x_sum FROM drill_atomic"
	)
	if len(rows) != 1:
		raise DrillFailure(f"expected one atomicity signature row, got {rows}")
	return {key: int(value or 0) for key, value in rows[0].items()}


def classify_atomic_signature(observed: dict[str, int], rows: int,
		committed_x: int) -> int:
	old = {
		"row_count": rows, "min_x": committed_x,
		"max_x": committed_x, "x_sum": rows * committed_x,
	}
	new = {
		"row_count": rows, "min_x": committed_x + 1,
		"max_x": committed_x + 1, "x_sum": rows * (committed_x + 1),
	}
	if observed == old:
		return committed_x
	if observed == new:
		return committed_x + 1
	raise DrillFailure(
		"crash recovery exposed a partial multi-shard commit: "
		f"got {observed}, expected exactly {old} or {new}"
	)


def signature(client: HttpClient) -> dict[str, int]:
	return {
		"count": scalar(client, "SELECT COUNT(*) AS value FROM drill_entry"),
		"id_sum": scalar(client, "SELECT COALESCE(SUM(id), 0) AS value FROM drill_entry"),
		"amount_sum": scalar(client, "SELECT COALESCE(SUM(amount), 0) AS value FROM drill_entry"),
		"pending": scalar(client, "SELECT COUNT(*) AS value FROM drill_entry WHERE state = 'pending'"),
		"audits": scalar(client, "SELECT COUNT(*) AS value FROM drill_audit"),
		"failed_sources": scalar(client, "SELECT COUNT(*) AS value FROM drill_trigger_source"),
		"failed_audits": scalar(client, "SELECT COUNT(*) AS value FROM drill_trigger_guard"),
		"rebuild_count": scalar(client, "SELECT COUNT(*) AS value FROM drill_rebuild"),
		"rebuild_sum": scalar(client, "SELECT COALESCE(SUM(payload), 0) AS value FROM drill_rebuild"),
	}


def expect(actual: object, wanted: object, context: str) -> None:
	if actual != wanted:
		raise DrillFailure(f"{context}: got {actual!r}, want {wanted!r}")


def initialize(client: HttpClient) -> None:
	client.sql(f"CREATE DATABASE IF NOT EXISTS {DATABASE}", database="system")
	client.scm('(settings "ShardSize" 10)')
	client.sql("CREATE TABLE drill_entry (id INT PRIMARY KEY, bucket INT NOT NULL, amount INT NOT NULL, state VARCHAR(16) NOT NULL) ENGINE=safe")
	client.sql("CREATE TABLE drill_audit (entry_id INT NOT NULL, amount INT NOT NULL) ENGINE=safe")
	client.sql("CREATE TRIGGER drill_entry_au AFTER UPDATE ON drill_entry FOR EACH ROW INSERT INTO drill_audit(entry_id, amount) VALUES (NEW.id, NEW.amount)")
	client.sql("CREATE TABLE drill_trigger_source (id INT PRIMARY KEY) ENGINE=safe")
	client.sql("CREATE TABLE drill_trigger_guard (marker INT UNIQUE) ENGINE=safe")
	client.sql("CREATE TRIGGER drill_trigger_source_ai AFTER INSERT ON drill_trigger_source FOR EACH ROW INSERT INTO drill_trigger_guard(marker) SELECT 1")
	values = ",".join(f"({i},{i % 4},{i * 10},'seed')" for i in range(1, 31))
	client.sql(f"INSERT INTO drill_entry VALUES {values}")
	client.scm(f'(rebuild (table "{DATABASE}" "drill_entry") true true)', timeout=120)
	# Give a rebuild enough work that SIGKILL can hit publication/catch-up rather
	# than merely killing an idle process after a tiny rebuild has completed.
	client.scm('(settings "ShardSize" 1000)')
	client.sql("CREATE TABLE drill_rebuild (id INT PRIMARY KEY, payload BIGINT NOT NULL) ENGINE=safe")
	client.scm(
		f'(insert (table "{DATABASE}" "drill_rebuild") \'("id" "payload") '
		'(map (produceN 100000) (lambda (i) (list (+ i 1) (+ i 1)))))',
		timeout=120,
	)
	client.scm(f'(rebuild (table "{DATABASE}" "drill_rebuild") true true)', timeout=120)


def initialize_atomicity(client: HttpClient, rows: int) -> None:
	client.sql(f"CREATE DATABASE IF NOT EXISTS {DATABASE}", database="system")
	client.scm('(settings "ShardSize" 100)')
	client.sql("CREATE TABLE drill_atomic (id INT PRIMARY KEY, x INT NOT NULL) ENGINE=safe")
	client.scm(
		f'(insert (table "{DATABASE}" "drill_atomic") \'("id" "x") '
		f'(map (produceN {rows}) (lambda (i) (list (+ i 1) 0))))',
		timeout=300,
	)
	client.scm(f'(rebuild (table "{DATABASE}" "drill_atomic") true true)', timeout=300)
	expect(
		atomic_signature(client),
		{"row_count": rows, "min_x": 0, "max_x": 0, "x_sum": 0},
		"atomicity fixture",
	)


def run_commit_crash(server: OwnedServer, rows: int, rounds: int,
		rng: random.Random, journal: list[dict]) -> None:
	committed_x = 0
	for round_index in range(1, rounds + 1):
		client = server.client
		assert client is not None
		session = f"drill-commit-crash-{round_index}"
		client.sql("START ACID TRANSACTION", session=session)
		updated = client.sql("UPDATE drill_atomic SET x = x + 1", session=session, timeout=300)
		if not updated or int(updated[0].get("affected_rows", 0)) != rows:
			raise DrillFailure(f"commit-crash update affected unexpected rows: {updated}")

		commit_result: list[object] = []
		commit_started = threading.Event()

		def commit() -> None:
			commit_started.set()
			try:
				commit_result.append(client.sql("COMMIT", session=session, timeout=300))
			except Exception as error:
				commit_result.append(error)

		thread = threading.Thread(target=commit, name="reliability-commit", daemon=True)
		thread.start()
		commit_started.wait(timeout=server.timeout)
		delay = rng.uniform(0.0, 0.05)
		time.sleep(delay)
		journal.append({
			"scenario": "commit-crash",
			"round": round_index,
			"delay_seconds": delay,
			"expected_old_x": committed_x,
			"expected_new_x": committed_x + 1,
		})
		server.kill()
		thread.join(timeout=2)
		client = server.start()
		observed = atomic_signature(client)
		journal[-1]["observed"] = observed
		committed_x = classify_atomic_signature(observed, rows, committed_x)


def run_committed_crash(server: OwnedServer, journal: list[dict]) -> dict[str, int]:
	client = server.client
	assert client is not None
	session = "drill-committed"
	client.sql("START ACID TRANSACTION", session=session)
	client.sql("UPDATE drill_entry SET amount = amount + 5, state = 'committed' WHERE id IN (1,10,11,20,21,30)", session=session)
	client.sql("DELETE FROM drill_entry WHERE id IN (3,13,23)", session=session)
	client.sql("INSERT INTO drill_entry VALUES (31,3,310,'committed'),(32,0,320,'committed')", session=session)
	client.sql("COMMIT", session=session)
	wanted = {
		"count": 29, "id_sum": 489, "amount_sum": 4920,
		"pending": 0, "audits": 6,
		"failed_sources": 0, "failed_audits": 0,
		"rebuild_count": 100000, "rebuild_sum": 5000050000,
	}
	expect(signature(client), wanted, "committed state before crash")
	journal.append({"scenario": "committed-crash", "expected": wanted})
	server.kill()
	client = server.start()
	expect(signature(client), wanted, "committed state after WAL recovery")
	return wanted


def run_uncommitted_crash(server: OwnedServer, wanted: dict[str, int],
		journal: list[dict]) -> None:
	client = server.client
	assert client is not None
	session = "drill-uncommitted"
	client.sql("START ACID TRANSACTION", session=session)
	client.sql("UPDATE drill_entry SET amount = amount + 1000, state = 'pending' WHERE id IN (2,12,22)", session=session)
	client.sql("DELETE FROM drill_entry WHERE id IN (4,14,24)", session=session)
	client.sql("INSERT INTO drill_entry VALUES (33,1,330,'pending')", session=session)
	journal.append({"scenario": "uncommitted-crash", "expected": wanted})
	server.kill()
	client = server.start()
	expect(signature(client), wanted, "uncommitted state leaked through crash recovery")


def run_failed_trigger(client: HttpClient, wanted: dict[str, int],
		journal: list[dict]) -> None:
	try:
		client.sql("INSERT INTO drill_trigger_source VALUES (1),(2)")
	except DrillFailure:
		pass
	else:
		raise DrillFailure("trigger uniqueness violation unexpectedly succeeded")
	expect(signature(client), wanted, "failed trigger statement was not atomic")
	journal.append({"scenario": "failed-trigger", "expected": wanted})


def run_rebuild_crash(server: OwnedServer, wanted: dict[str, int], rng: random.Random,
		journal: list[dict], round_index: int) -> None:
	client = server.client
	assert client is not None
	result: list[object] = []

	def rebuild() -> None:
		try:
			result.append(client.scm(f'(rebuild (table "{DATABASE}" "drill_entry") true true)', timeout=120))
		except Exception as error:
			result.append(error)

	thread = threading.Thread(target=rebuild, name="reliability-rebuild", daemon=True)
	thread.start()
	delay = rng.uniform(0.0, 0.02)
	time.sleep(delay)
	if not thread.is_alive():
		raise DrillFailure(
			"rebuild completed before the crash window; increase the drill fixture"
		)
	journal.append({
		"scenario": "rebuild-crash", "round": round_index,
		"delay_seconds": delay, "expected": wanted,
	})
	server.kill()
	thread.join(timeout=2)
	client = server.start()
	expect(signature(client), wanted, "rebuild crash changed committed state")


def run_soak(server: OwnedServer, workers: int, operations: int,
		journal: list[dict]) -> dict[str, int]:
	client = server.client
	assert client is not None
	start = threading.Barrier(workers + 2)
	failures: list[str] = []
	stop_rebuild = threading.Event()

	def writer(worker: int) -> None:
		try:
			start.wait()
			for offset in range(operations):
				row_id = 1000 + worker * operations + offset
				client.sql(f"INSERT INTO drill_entry VALUES ({row_id},{worker % 4},{row_id},'soak')")
			client.sql(f"UPDATE drill_entry SET amount = amount + 7 WHERE id >= {1000 + worker * operations} AND id < {1000 + (worker + 1) * operations}")
			delete_ids = [1000 + worker * operations + offset for offset in range(operations) if offset % 5 == 0]
			if delete_ids:
				client.sql("DELETE FROM drill_entry WHERE id IN (" + ",".join(map(str, delete_ids)) + ")")
		except Exception as error:
			failures.append(f"writer {worker}: {error}")

	def rebuilder() -> None:
		try:
			start.wait()
			while not stop_rebuild.wait(0.01):
				client.scm(f'(rebuild (table "{DATABASE}" "drill_entry") true false)', timeout=120)
		except Exception as error:
			failures.append(f"rebuilder: {error}")

	threads = [threading.Thread(target=writer, args=(worker,), daemon=True) for worker in range(workers)]
	threads.append(threading.Thread(target=rebuilder, daemon=True))
	for thread in threads:
		thread.start()
	start.wait()
	for thread in threads[:-1]:
		thread.join(timeout=max(30, operations * 2))
	stop_rebuild.set()
	threads[-1].join(timeout=120)
	if any(thread.is_alive() for thread in threads):
		raise DrillFailure("soak worker did not terminate")
	if failures:
		raise DrillFailure("; ".join(failures))

	expected_added_ids = [
		1000 + worker * operations + offset
		for worker in range(workers)
		for offset in range(operations)
		if offset % 5 != 0
	]
	before = signature(client)
	wanted = dict(before)
	wanted["count"] = 29 + len(expected_added_ids)
	wanted["id_sum"] = 489 + sum(expected_added_ids)
	wanted["amount_sum"] = 4920 + sum(row_id + 7 for row_id in expected_added_ids)
	wanted["pending"] = 0
	# The update trigger fires before subsequent deletes, once per inserted row.
	wanted["audits"] = 6 + workers * operations
	expect(before, wanted, "concurrent write/rebuild soak result")
	journal.append({"scenario": "soak", "workers": workers, "operations": operations, "expected": wanted})
	server.kill()
	client = server.start()
	expect(signature(client), wanted, "soak state after crash recovery")
	return wanted


def run_restore(server: OwnedServer, wanted: dict[str, int], artifact_dir: Path,
		binary: Path, app: Path, timeout: float, journal: list[dict]) -> OwnedServer:
	server.stop()
	backup_dir = artifact_dir / "backup-data"
	restored_dir = artifact_dir / "restored-data"
	shutil.copytree(server.data_dir, backup_dir)
	shutil.copytree(backup_dir, restored_dir)
	restored = OwnedServer(binary, app, restored_dir, artifact_dir / "restore-logs", timeout)
	restored.artifact_dir.mkdir()
	try:
		client = restored.start()
		expect(signature(client), wanted, "restored data-directory snapshot")
	except Exception:
		restored.cleanup()
		raise
	journal.append({"scenario": "restore", "backup": str(backup_dir), "expected": wanted})
	return restored


def write_manifest(path: Path, seed: int, status: str, journal: list[dict],
		error: str | None = None) -> None:
	payload = {
		"seed": seed,
		"status": status,
		"error": error,
		"scenarios": journal,
		"finished_at": time.strftime("%Y-%m-%dT%H:%M:%S%z"),
	}
	temporary = path.with_suffix(".tmp")
	temporary.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")
	temporary.replace(path)


def parse_args() -> argparse.Namespace:
	parser = argparse.ArgumentParser(description="Run destructive reliability drills on an owned disposable MemCP instance")
	parser.add_argument("--binary", type=Path, default=ROOT / "memcp")
	parser.add_argument("--app", type=Path, default=ROOT / "lib/main.scm")
	parser.add_argument("--artifacts", type=Path, help="new directory for data, logs, journal, and restore snapshot")
	parser.add_argument("--seed", type=int, default=random.SystemRandom().randrange(2**63))
	parser.add_argument("--mode", choices=("atomicity", "crash", "soak", "restore", "all"), default="all")
	parser.add_argument("--workers", type=int, default=4)
	parser.add_argument("--operations", type=int, default=25, help="rows written by each soak worker")
	parser.add_argument("--rebuild-crashes", type=int, default=5,
		help="number of randomized rebuild/kill/recovery rounds")
	parser.add_argument("--commit-crashes", type=int, default=25,
		help="number of ACID COMMIT/kill/recovery races in atomicity mode")
	parser.add_argument("--atomicity-rows", type=int, default=100000,
		help="rows updated across ShardSize=100 shards in atomicity mode")
	parser.add_argument("--timeout", type=float, default=30)
	return parser.parse_args()


def main() -> int:
	args = parse_args()
	if (args.workers < 1 or args.operations < 1 or args.rebuild_crashes < 1 or
			args.commit_crashes < 1 or args.atomicity_rows < 1 or args.timeout <= 0):
		raise DrillFailure("workers, operations, crash rounds, row counts, and timeout must be positive")
	binary = args.binary.expanduser().resolve()
	app = args.app.expanduser().resolve()
	if not binary.is_file() or not os.access(binary, os.X_OK):
		raise DrillFailure(f"MemCP binary is not executable: {binary}")
	if not app.is_file():
		raise DrillFailure(f"Scheme application does not exist: {app}")
	artifact_dir = make_artifact_dir(args.artifacts)
	data_dir = artifact_dir / "source-data"
	data_dir.mkdir()
	rng = random.Random(args.seed)
	journal: list[dict] = []
	server = OwnedServer(binary, app, data_dir, artifact_dir, args.timeout)
	active_server = server
	manifest = artifact_dir / "manifest.json"
	print(f"Reliability drill artifacts: {artifact_dir}")
	print(f"Seed: {args.seed}")
	try:
		client = server.start()
		if args.mode == "atomicity":
			initialize_atomicity(client, args.atomicity_rows)
			run_commit_crash(server, args.atomicity_rows, args.commit_crashes, rng, journal)
			write_manifest(manifest, args.seed, "passed", journal)
			print(f"PASS: {len(journal)} reliability scenarios; manifest: {manifest}")
			return 0
		initialize(client)
		wanted = run_committed_crash(server, journal)
		run_uncommitted_crash(server, wanted, journal)
		assert server.client is not None
		run_failed_trigger(server.client, wanted, journal)
		if args.mode in ("crash", "all"):
			for round_index in range(1, args.rebuild_crashes + 1):
				run_rebuild_crash(server, wanted, rng, journal, round_index)
		if args.mode in ("soak", "all"):
			wanted = run_soak(server, args.workers, args.operations, journal)
		if args.mode in ("restore", "all"):
			active_server = run_restore(server, wanted, artifact_dir, binary, app, args.timeout, journal)
		write_manifest(manifest, args.seed, "passed", journal)
		print(f"PASS: {len(journal)} reliability scenarios; manifest: {manifest}")
		return 0
	except Exception as error:
		write_manifest(manifest, args.seed, "failed", journal, str(error))
		print(f"FAIL: {error}", file=sys.stderr)
		print(f"Reproduce with --seed {args.seed}; artifacts retained at {artifact_dir}", file=sys.stderr)
		return 1
	finally:
		active_server.cleanup()


if __name__ == "__main__":
	raise SystemExit(main())
