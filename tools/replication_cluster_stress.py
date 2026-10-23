#!/usr/bin/env python3
# Copyright (C) 2026 Carl-Philip Hänsch
# SPDX-License-Identifier: GPL-3.0-or-later

"""Own a replication cluster, workload threads, SIGKILLs and a write oracle.

Only disposable processes/data directories created by this runner are touched.
No Go test binary starts servers. Logs, seed and acknowledged/ambiguous writes
survive a failed run in the artifact directory.
"""

from __future__ import annotations

import argparse
import base64
import concurrent.futures
import http.client
import json
import os
from pathlib import Path
import queue
import random
import shlex
import shutil
import socket
import subprocess
import sys
import threading
import time
import traceback
import uuid

from reliability_drill import (
    ROOT, DrillFailure, HttpClient, OwnedServer, expect, free_port,
    make_artifact_dir, write_manifest,
)


DATABASE = "replica_test"
PASSWORD = "replication-test-password"
NETWORK_ERROR = "Cluster network error"


def eventually(check, context: str, timeout: float = 15):
    deadline = time.monotonic() + timeout
    while True:
        result = check()
        if result:
            return result
        if time.monotonic() >= deadline:
            raise DrillFailure(context)
        time.sleep(0.05)


class Client(HttpClient):
    def __init__(self, port: int, timeout: float, user="root", password=PASSWORD):
        super().__init__(port, timeout)
        token = base64.b64encode(f"{user}:{password}".encode()).decode()
        self.headers["Authorization"] = "Basic " + token

    def sql(self, statement, database=DATABASE, session=None, timeout=None):
        return super().sql(statement, database, session, timeout)

    def number(self, statement, column="n", database=DATABASE):
        rows = self.sql(statement, database)
        if len(rows) != 1 or column not in rows[0]:
            raise DrillFailure(f"missing scalar {column}: {statement}: {rows}")
        return int(rows[0][column])

    def reject(self, statement, *, contains=None, status=None, database=DATABASE):
        try:
            self.sql(statement, database)
        except DrillFailure as error:
            if contains and contains not in str(error):
                raise
            if status and f"({status})" not in str(error):
                raise
            return
        raise DrillFailure(f"unexpectedly accepted: {statement}")


class Node(OwnedServer):
    def __init__(self, args, artifacts, name, replication, local_password=PASSWORD):
        directory = artifacts / name
        directory.mkdir()
        data = directory / "data"
        data.mkdir()
        super().__init__(args.binary, ROOT / "lib/main.scm", data, directory, args.timeout)
        self.name = name
        self.port, self.mysql_port = free_port(), free_port()
        self.replication = replication
        self.password = local_password
        self.executor = shlex.split(args.executor)
        self.startup_timeout = args.startup_timeout
        self.client = Client(self.port, self.timeout)

    def start(self, *, retired=False, probe_sql=True):
        if self.process is not None:
            raise DrillFailure(f"{self.name} already has an owned process")
        self.generation += 1
        log = self.artifact_dir / f"memcp-generation-{self.generation}.log"
        self.log_handle = log.open("wb")
        command = self.executor + [
            str(self.binary), "-data", str(self.data_dir),
            f"--api-port={self.port}", f"--mysql-port={self.mysql_port}",
            "--mysql-socket=", "--no-repl", f"--root-password={self.password}",
            f"--replication={self.replication}", str(self.app),
        ]
        self.process = subprocess.Popen(
            command, cwd=ROOT, stdin=subprocess.DEVNULL, stdout=self.log_handle,
            stderr=subprocess.STDOUT, start_new_session=True,
        )
        print(f"  {self.name}: start pid={self.process.pid}, generation={self.generation}", flush=True)
        self.client = Client(self.port, self.timeout)

        def ready():
            if self.process.poll() is not None:
                if retired:
                    return True
                raise DrillFailure(f"{self.name} exited during startup; inspect {log}")
            if retired or not probe_sql:
                try:
                    with socket.create_connection(("127.0.0.1", self.port), timeout=0.1):
                        return True
                except OSError:
                    return False
            try:
                self.client.sql("SELECT 1", "system")
                return True
            except (OSError, DrillFailure):
                return False

        eventually(ready, f"{self.name} did not start; inspect {log}", self.startup_timeout)
        return self.client

    def kill(self):
        # Keep the fixed-address client available to workload threads while
        # OwnedServer kills and joins exactly its own child, never a port match.
        client = self.client
        try:
            if self.process is not None:
                action = "SIGKILL" if self.process.poll() is None else "reap"
                print(f"  {self.name}: {action} pid={self.process.pid}", flush=True)
            super().kill()
        finally:
            self.client = client

    def journal(self):
        return json.loads((self.data_dir / ".replication-state").read_text())

    def records(self, table):
        schema = json.loads((self.data_dir / DATABASE / "schema.json").read_text())["tables"][table]
        shards = schema.get("PShards" if schema["ShardMode"] == "partition" else "Shards") or []
        active = {shard.get("mosi", shard["uuid"]) if isinstance(shard, dict) else shard
                  for shard in shards if shard is not None}
        records = [record for record in self.journal()["records"].values()
                   if record["resource"].get("shard") in active]
        expect(len(records), len(active), f"{self.name}: MOSI records for every active {table} shard")
        return records

    def record(self, table):
        records = self.records(table)
        expect(len(records), 1, f"{self.name}: one shard for {table}")
        return records[0]

    def statuses(self):
        return json.loads(json.loads(self.client.scm("(replication_slaves)")))

    def connected(self):
        statuses = self.statuses()
        return len(statuses) == 1 and statuses[0]["connected"]

    def wait_offline(self):
        def offline():
            try:
                with socket.create_connection(("127.0.0.1", self.port), timeout=0.1):
                    return False
            except OSError:
                return True
        eventually(offline, f"{self.name} did not shut down", self.timeout)

    def mysql(self, statement):
        command = ["mysql", "--no-defaults", "--batch", "--skip-column-names",
                   "--protocol=TCP", "--host=127.0.0.1", f"--port={self.mysql_port}",
                   "--user=root", DATABASE, "--execute=" + statement]
        result = subprocess.run(command, env={**os.environ, "MYSQL_PWD": PASSWORD},
                                capture_output=True, text=True, timeout=self.timeout)
        if result.returncode:
            raise DrillFailure(f"MySQL login/query failed: {result.stderr}")
        return result.stdout.strip()


class Cluster:
    def __init__(self, args, artifacts, name):
        directory = artifacts / name
        directory.mkdir()
        port = free_port()
        self.master = Node(args, directory, "master", str(port))
        self.slave = Node(args, directory, "slave", f"127.0.0.1:{port}", "different-local-password")

    def __enter__(self):
        return self

    def __exit__(self, *_):
        self.slave.cleanup()
        self.master.cleanup()

    def initialize(self, statements):
        master = self.master.start()
        master.sql(f"CREATE DATABASE {DATABASE}", "system")
        for statement in statements:
            master.sql(statement)
        self.slave.start()


def run_regressions(args, artifacts, journal):
    print("SCENARIO: alternating crashes, authentication, DDL, sharing and retirement", flush=True)
    with Cluster(args, artifacts, "regressions") as cluster:
        master, slave = cluster.master, cluster.slave
        m = master.start()
        m.sql(f"CREATE DATABASE {DATABASE}", "system")
        m.sql("CREATE TABLE items(id INT PRIMARY KEY, value INT) ENGINE=SAFE")
        m.sql("INSERT INTO items VALUES(1,10)")
        for statement in ("CREATE DATABASE private_test",
                          "CREATE USER replica_user IDENTIFIED BY 'user-password'",
                          f"GRANT ALL ON {DATABASE}.* TO replica_user"):
            m.sql(statement, "system")
        s = slave.start(probe_sql=False)
        def first_mysql_login():
            try:
                return slave.mysql("SELECT COUNT(*) FROM items")
            except DrillFailure:
                return False
        expect(eventually(first_mysql_login, "first MySQL login/transfer did not become ready"),
               "1", "first MySQL login/transfer")
        expect(s.number("SELECT COUNT(*) AS n FROM items"), 1, "initial HTTP transfer")
        identity = json.loads((slave.data_dir / "settings.json").read_text())["ReplicationUUID"]
        if not identity:
            raise DrillFailure("slave identity was not persisted")
        user = Client(slave.port, args.timeout, "replica_user", "user-password")
        expect(user.number("SELECT COUNT(*) AS n FROM items"), 1, "granted database")
        user.reject("SELECT 1", database="private_test")
        for statement in ("CREATE DATABASE slave_admin_test",
                          "CREATE USER slave_created_user IDENTIFIED BY 'slave-created-password'",
                          f"GRANT ALL ON {DATABASE}.* TO slave_created_user"):
            s.sql(statement, "system")
        expect(m.number("SELECT COUNT(*) AS n FROM user WHERE username='slave_created_user'",
                        database="system"), 1, "master access catalog")
        if not (master.data_dir / "slave_admin_test/schema.json").exists():
            raise DrillFailure("slave CREATE DATABASE did not reach master")
        for round_index in range(3):
            master.kill()
            if round_index == 0:
                s.sql("CREATE TABLE offline_created(id INT PRIMARY KEY, payload TEXT) ENGINE=SAFE")
                s.sql("INSERT INTO offline_created VALUES(1,'survived-offline-ddl')")
            for offset in range(5):
                row_id = round_index * 5 + offset + 2
                s.sql(f"INSERT INTO items VALUES({row_id},{row_id * 10})")
            s.sql("UPDATE items SET value=value+1 WHERE id=1")
            expect(slave.mysql("SELECT COUNT(*) FROM items"), str(6 + round_index * 5),
                   "offline MySQL authentication")
            master.start()
            eventually(master.connected, "slave did not reconnect to restarted master")
            eventually(lambda: (master.data_dir / DATABASE / "schema.json").read_bytes() ==
                       (slave.data_dir / DATABASE / "schema.json").read_bytes(), "backup schema did not catch up")
            slave.kill()
            m.reject("SELECT * FROM items", status=500, contains=NETWORK_ERROR)
            s = slave.start()
            expect(s.number("SELECT COUNT(*) AS n FROM items"), 6 + round_index * 5, "durable inserts")
            expect(s.number("SELECT value FROM items WHERE id=1", "value"), 11 + round_index, "durable update")
            expect(s.sql("SELECT payload FROM offline_created WHERE id=1")[0]["payload"],
                   "survived-offline-ddl", "offline DDL recovery")
            expect(json.loads((slave.data_dir / "settings.json").read_text())["ReplicationUUID"], identity,
                   "persistent reconnect identity")
        m.sql("ALTER TABLE items ADD COLUMN extra INT DEFAULT 7")
        expect(s.number("SELECT COUNT(*) AS n FROM items WHERE extra=7"), 16, "remote DDL/defaults")
        s.sql("UPDATE items SET value=value WHERE id=1")
        s.sql("ALTER TABLE items DROP COLUMN extra")
        m.sql("RENAME TABLE items TO renamed_items")
        expect(s.number("SELECT COUNT(*) AS n FROM renamed_items"), 16, "remote rename")
        s.sql("RENAME TABLE renamed_items TO items")
        s.sql("UPDATE items SET value=value WHERE id=1")
        m.sql("SELECT COUNT(*) AS n FROM items")
        expect((master.record("items")["state"], slave.record("items")["state"]), ("S", "O"), "shared read")
        epoch = master.record("items")["epoch"]
        m.sql("EXPLAIN SELECT COUNT(*) AS n FROM items")
        master.kill()
        expect(s.number("SELECT COUNT(*) AS n FROM items"), 16, "offline O read")
        s.reject("INSERT INTO items VALUES(9000,9000)", status=500, contains=NETWORK_ERROR)
        master.start()
        eventually(master.connected, "shared owner did not reconnect")
        def reader(index):
            client = s if index % 2 else m
            for _ in range(5):
                expect(client.number("SELECT COUNT(*) AS n FROM items"), 16, "parallel read")
        with concurrent.futures.ThreadPoolExecutor(max_workers=32) as threads:
            list(threads.map(reader, range(32)))
        expect((master.record("items")["state"], slave.record("items")["state"],
                master.record("items")["epoch"]), ("S", "O", epoch), "read-only ownership/epoch")
        for _ in range(2):
            m.sql("UPDATE items SET value=value+1 WHERE id=1")
            expect((master.record("items")["state"], slave.record("items")["state"]), ("M", "IO"), "write invalidation")
            s.sql("SELECT COUNT(*) AS n FROM items")
            expect((master.record("items")["state"], slave.record("items")["state"]), ("O", "S"), "reverse sharing")
        s.sql("UPDATE items SET value=value-2 WHERE id=1")
        session = uuid.uuid4().hex
        s.sql("START ACID TRANSACTION", session=session)
        s.sql("INSERT INTO items VALUES(1000,999)", session=session)
        with concurrent.futures.ThreadPoolExecutor(max_workers=1) as threads:
            waiting = threads.submit(m.number, "SELECT COUNT(*) AS n FROM items")
            try:
                waiting.result(timeout=0.1)
            except concurrent.futures.TimeoutError:
                pass
            else:
                raise DrillFailure("transfer crossed an uncommitted transaction")
            s.sql("COMMIT", session=session)
            expect(waiting.result(timeout=args.timeout), 17, "handover after transaction commit")
        expect(s.number("SELECT value FROM items WHERE id=1000", "value"), 999, "late handover")
        s.sql("DELETE FROM items WHERE id=1000")
        Client(slave.port, args.timeout, password="wrong-password").reject("SELECT 1", status=401)
        expect(master.statuses()[0]["id"], identity, "dashboard identity")
        s.sql("SHUTDOWN")
        slave.wait_offline()
        m.sql("SELECT 1", "system")
        slave.kill()
        s = slave.start()
        s.sql("SELECT COUNT(*) AS n FROM items")
        def backed_up():
            destination = master.journal()["records"]
            for key, source in slave.journal()["records"].items():
                if (source["resource"]["database"] == DATABASE and source["resource"].get("shard")
                        and source["state"] == "M"):
                    if not destination.get(key, {}).get("generation") or destination[key]["generation"] != source.get("generation"):
                        return False
            return True
        eventually(backed_up, "reconnected slave did not publish completed backups")
        m.scm(f"(replication_retire {json.dumps(identity)})")
        slave.wait_offline()
        expect(m.number("SELECT COUNT(*) AS n FROM items"), 16, "retirement backup")
        expect(m.sql("SELECT payload FROM offline_created WHERE id=1")[0]["payload"],
               "survived-offline-ddl", "offline backup schema/data")
        master.kill()
        m = master.start()
        expect(m.number("SELECT COUNT(*) AS n FROM items"), 16, "backup restart")
        slave.kill()
        slave.start(retired=True)
        try:
            slave.client.reject("SELECT * FROM items")
        except OSError:
            pass  # A retired slave may shut itself down before the HTTP request.
        journal.append({"scenario": "regressions", "crash_rounds": 3, "parallel_readers": 32})


def run_partitions(args, artifacts, journal):
    print("SCENARIO: independent partition ownership and backend authority", flush=True)
    with Cluster(args, artifacts, "partitions") as cluster:
        master, slave = cluster.master, cluster.slave
        m = master.start()
        m.sql(f"CREATE DATABASE {DATABASE}", "system")
        m.sql("CREATE TABLE items(id INT PRIMARY KEY, value INT) ENGINE=SAFE")
        m.sql("INSERT INTO items VALUES(1,10),(9,90)")
        expect(m.scm(f'(partitiontable (table "{DATABASE}" false "items" false) (list "id" (list 5)))').strip(),
               "true", "range partition setup")
        m.sql("CREATE TABLE volatile_items(id INT, value TEXT) ENGINE=MEMORY")
        m.sql("INSERT INTO volatile_items VALUES(1,'streamed-memory-generation')")
        s = slave.start()
        expect(s.sql("SELECT value FROM volatile_items WHERE id=1")[0]["value"],
               "streamed-memory-generation", "volatile generation transfer")
        expect(s.number("SELECT COUNT(*) AS n FROM items"), 2, "partition initial transfer")
        eventually(lambda: slave.journal().get("initialized", {}).get(DATABASE), "initial shard M claims")
        expect(m.number("SELECT value FROM items WHERE id=1", "value"), 10, "one-partition read")
        expect(sorted(record["state"] for record in master.records("items")), ["IO", "S"], "master partition rights")
        expect(sorted(record["state"] for record in slave.records("items")), ["M", "O"], "slave partition rights")
        master.kill()
        s.sql("UPDATE items SET value=value+1 WHERE id=9")
        s.reject("UPDATE items SET value=value+1 WHERE id=1", status=500, contains=NETWORK_ERROR)
        master.start()
        slave.kill()
        expect(m.number("SELECT value FROM items WHERE id=1", "value"), 10, "restart current backend S")
        m.reject("SELECT value FROM items WHERE id=9", status=500, contains=NETWORK_ERROR)
        s = slave.start()
        expect(s.number("SELECT value FROM items WHERE id=9", "value"), 91, "restart autonomous M write")
        journal.append({"scenario": "independent-partitions"})


class Oracle:
    """Track per-row legal outcomes; never replay an ambiguously answered write."""
    def __init__(self, path):
        self.lock = threading.Lock()
        self.rows = {0: {0}}
        self.acknowledged = 0
        self.ambiguous = 0
        self.events = path.open("w")

    def attempt(self, row_id, value):
        with self.lock:
            self.rows.setdefault(row_id, {None}).add(value)
            self.log("attempt", row_id, value)

    def acknowledge(self, row_id, value):
        with self.lock:
            self.rows[row_id] = {value}
            self.acknowledged += 1
            self.log("ack", row_id, value)

    def uncertain(self, row_id, value):
        with self.lock:
            self.ambiguous += 1
            self.log("ambiguous", row_id, value)

    def log(self, kind, row_id, value):
        self.events.write(json.dumps({"event": kind, "id": row_id, "value": value}) + "\n")
        self.events.flush()

    def verify(self, client):
        rows = client.sql("SELECT id,value FROM stress_items")
        actual = {int(row["id"]): int(row["value"]) for row in rows}
        expect(len(rows), len(actual), "no duplicate stress row IDs")
        with self.lock:
            for row_id in actual.keys() | self.rows.keys():
                if actual.get(row_id) not in self.rows.get(row_id, set()):
                    raise DrillFailure(f"lost/corrupted write: id={row_id}, actual={actual.get(row_id)}, allowed={self.rows.get(row_id)}")


def workload(args, cluster, oracle, phase, *, analytics=False, crash_slave=False):
    stop = threading.Event()
    progressed = threading.Event()
    crashing = threading.Event()
    errors = queue.Queue()
    barrier = threading.Barrier(args.workers + (3 if analytics else 1))
    slave_client = cluster.slave.client

    def writer(worker):
        rng = random.Random(args.seed + phase * 1000 + worker)
        try:
            barrier.wait(timeout=args.timeout)
            for offset in range(args.operations):
                if stop.is_set():
                    return
                row_id = 1 + (phase * args.workers + worker) * args.operations + offset
                statements = [(f"INSERT INTO stress_items VALUES({row_id},{row_id * 10})", row_id * 10),
                              (f"UPDATE stress_items SET value=value+1 WHERE id={row_id}", row_id * 10 + 1)]
                if rng.random() < 0.25:
                    statements.append((f"DELETE FROM stress_items WHERE id={row_id}", None))
                for statement, value in statements:
                    oracle.attempt(row_id, value)
                    try:
                        slave_client.sql(statement)
                    except (OSError, http.client.HTTPException, json.JSONDecodeError):
                        if not crashing.is_set():
                            raise
                        oracle.uncertain(row_id, value)
                        return
                    oracle.acknowledge(row_id, value)
                    progressed.set()
                time.sleep(rng.uniform(0.001, 0.01))
        except Exception as error:
            errors.put(error)
            stop.set()

    def reader():
        try:
            barrier.wait(timeout=args.timeout)
            while not stop.is_set():
                rows = cluster.master.client.sql("SELECT id,value FROM stress_items")
                for row in rows:
                    row_id, value = int(row["id"]), int(row["value"])
                    if value not in (row_id * 10, row_id * 10 + 1):
                        raise DrillFailure(f"analytic reader observed incomplete/corrupt row: {row}")
                time.sleep(0.005)
        except Exception as error:
            errors.put(error)
            stop.set()

    def maintenance():
        try:
            barrier.wait(timeout=args.timeout)
            while not stop.is_set():
                cluster.master.client.scm(f'(blob_cleanup "{DATABASE}")')
                cluster.slave.client.scm(f'(blob_cleanup "{DATABASE}")')
                time.sleep(0.01)
        except Exception as error:
            errors.put(error)
            stop.set()

    threads = [threading.Thread(target=writer, args=(worker,), name=f"writer-{phase}-{worker}", daemon=True)
               for worker in range(args.workers)]
    if analytics:
        threads.append(threading.Thread(target=reader, name=f"analytics-{phase}", daemon=True))
        threads.append(threading.Thread(target=maintenance, name=f"cleanup-{phase}", daemon=True))
    for thread in threads:
        thread.start()
    try:
        barrier.wait(timeout=args.timeout)
        if crash_slave:
            if not progressed.wait(args.timeout):
                raise DrillFailure("no acknowledged write before slave crash")
            time.sleep(random.Random(args.seed + phase).uniform(0.005, 0.02))
            crashing.set()
            # Master is alive throughout this phase; never crash both nodes.
            if cluster.master.process.poll() is not None:
                raise DrillFailure("master unexpectedly offline before slave crash")
            cluster.slave.kill()
        deadline = time.monotonic() + args.timeout * max(4, args.operations)
        for thread in threads[:args.workers]:
            thread.join(max(0, deadline - time.monotonic()))
            if thread.is_alive():
                raise DrillFailure(f"workload thread stuck: {thread.name}")
    finally:
        stop.set()
        for thread in threads:
            thread.join(args.timeout + 1)
        if any(thread.is_alive() for thread in threads):
            raise DrillFailure("workload threads did not stop")
    if not errors.empty():
        raise errors.get()


def run_stress(args, artifacts, journal):
    print(f"SCENARIO: threaded stress ({args.rounds} rounds, {args.workers} writers, seed {args.seed})", flush=True)
    oracle = Oracle(artifacts / "writes.jsonl")
    try:
        with Cluster(args, artifacts, "stress") as cluster:
            cluster.initialize(["CREATE TABLE stress_items(id INT PRIMARY KEY, value INT) ENGINE=SAFE",
                                "INSERT INTO stress_items VALUES(0,0)"])
            cluster.slave.client.sql("SELECT * FROM stress_items")
            eventually(lambda: cluster.slave.journal().get("initialized", {}).get(DATABASE), "initial stress M claims")
            for round_index in range(args.rounds):
                print(f"  round {round_index + 1}: online analytics/GC, master crash, slave crash", flush=True)
                phase = round_index * 3
                workload(args, cluster, oracle, phase, analytics=True)
                oracle.verify(cluster.slave.client)
                # Public DDL obtains schema M and all table shard M, including
                # empty shards. Offline writes are tested only after full M;
                # this does not assume an unimplemented automatic reclamation.
                cluster.slave.client.sql("ALTER TABLE stress_items ADD COLUMN ownership_probe INT DEFAULT 0")
                cluster.slave.client.sql("ALTER TABLE stress_items DROP COLUMN ownership_probe")
                expect(set(record["state"] for record in cluster.slave.records("stress_items")), {"M"}, "pre-crash M rights")
                cluster.master.kill()
                workload(args, cluster, oracle, phase + 1)
                oracle.verify(cluster.slave.client)
                cluster.master.start()
                eventually(cluster.master.connected, "stress slave did not reconnect")
                workload(args, cluster, oracle, phase + 2, crash_slave=True)
                cluster.master.client.reject("SELECT * FROM stress_items", status=500, contains=NETWORK_ERROR)
                cluster.slave.start()
                oracle.verify(cluster.slave.client)
                journal.append({"scenario": "threaded-stress", "round": round_index + 1,
                                "acknowledged": oracle.acknowledged, "ambiguous": oracle.ambiguous})
            if oracle.acknowledged == 0:
                raise DrillFailure("stress completed without acknowledged writes")
    finally:
        oracle.events.close()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, default=ROOT / "memcp")
    parser.add_argument("--artifacts", type=Path)
    parser.add_argument("--rounds", type=int, default=5)
    parser.add_argument("--workers", type=int, default=4)
    parser.add_argument("--operations", type=int, default=25)
    parser.add_argument("--seed", type=int, default=random.SystemRandom().randrange(2**63))
    parser.add_argument("--timeout", type=float, default=20)
    parser.add_argument("--startup-timeout", type=float, default=45)
    parser.add_argument("--executor", default="", help="optional server executor, e.g. qemu-aarch64-static")
    args = parser.parse_args()
    if min(args.rounds, args.workers, args.operations, args.timeout, args.startup_timeout) <= 0:
        parser.error("rounds, workers, operations and timeouts must be positive")
    args.binary = args.binary.expanduser().resolve()
    if not args.binary.is_file() or not os.access(args.binary, os.X_OK):
        parser.error(f"build the MemCP binary first: {args.binary}")
    if not shutil.which("mysql"):
        parser.error("mysql client required (mariadb-client)")
    artifacts = make_artifact_dir(args.artifacts)
    journal = [{"scenario": "configuration", "rounds": args.rounds, "workers": args.workers,
                "operations": args.operations, "binary": str(args.binary), "executor": args.executor,
                "timeout": args.timeout, "startup_timeout": args.startup_timeout}]
    print(f"Replication cluster stress artifacts: {artifacts}\nSeed: {args.seed}", flush=True)
    try:
        run_regressions(args, artifacts, journal)
        run_partitions(args, artifacts, journal)
        run_stress(args, artifacts, journal)
        write_manifest(artifacts / "manifest.json", args.seed, "passed", journal)
        print("PASS: replication cluster stress", flush=True)
        return 0
    except Exception as error:
        write_manifest(artifacts / "manifest.json", args.seed, "failed", journal, str(error))
        traceback.print_exc()
        replay = [sys.executable, str(Path(__file__).resolve()), "--binary", str(args.binary),
                  "--seed", str(args.seed), "--rounds", str(args.rounds), "--workers", str(args.workers),
                  "--operations", str(args.operations), "--timeout", str(args.timeout),
                  "--startup-timeout", str(args.startup_timeout), "--executor", args.executor]
        print(f"FAIL: {error}\nReproduce: {shlex.join(replay)}\nArtifacts: {artifacts}", file=sys.stderr, flush=True)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
