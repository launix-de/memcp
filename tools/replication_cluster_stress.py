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
import contextlib
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
from types import SimpleNamespace

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
    def __init__(self, port: int, timeout: float, user="root", password=PASSWORD, database=DATABASE):
        super().__init__(port, timeout)
        self.database = database
        token = base64.b64encode(f"{user}:{password}".encode()).decode()
        self.headers["Authorization"] = "Basic " + token

    def sql(self, statement, database=None, session=None, timeout=None):
        return super().sql(statement, self.database if database is None else database, session, timeout)

    def number(self, statement, column="n", database=None):
        rows = self.sql(statement, database)
        if len(rows) != 1 or column not in rows[0]:
            raise DrillFailure(f"missing scalar {column}: {statement}: {rows}")
        return int(rows[0][column])

    def reject(self, statement, *, contains=None, status=None, database=None):
        started = time.monotonic()
        try:
            self.sql(statement, database)
        except DrillFailure as error:
            if contains and contains not in str(error):
                raise
            if status and f"({status})" not in str(error):
                raise
            if contains == NETWORK_ERROR and time.monotonic() - started > 5:
                raise DrillFailure(f"unavailable owner did not fail promptly: {statement}") from error
            return
        raise DrillFailure(f"unexpectedly accepted: {statement}")


class Node(OwnedServer):
    def __init__(self, args, artifacts, name, replication, local_password=PASSWORD, database=DATABASE):
        directory = artifacts / name
        directory.mkdir()
        data = directory / "data"
        data.mkdir()
        super().__init__(args.binary, ROOT / "lib/main.scm", data, directory, args.timeout)
        self.name = name
        self.database = database
        self.port, self.mysql_port = free_port(), free_port()
        self.replication = replication
        self.password = local_password
        self.executor = shlex.split(args.executor)
        self.startup_timeout = args.startup_timeout
        self.client = Client(self.port, self.timeout, database=self.database)

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
        self.client = Client(self.port, self.timeout, database=self.database)

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
        if list(self.data_dir.glob(".replication-state*")):
            raise DrillFailure(f"{self.name} wrote a private replication journal")
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

    def snapshot(self):
        # Inspect live RAM state; disk contents cannot grant MOSI rights.
        result = self.client.scm("(replication_status)").strip()
        if result.startswith('"'):
            result = json.loads(result)
        return json.loads(result)

    def records(self, table):
        schema = json.loads((self.data_dir / self.database / "schema.json").read_text())["tables"][table]
        shards = schema.get("PShards" if schema["ShardMode"] == "partition" else "Shards") or []
        active = {shard.get("mosi", shard["uuid"]) if isinstance(shard, dict) else shard
                  for shard in shards if shard is not None}
        records = [record for record in self.snapshot()["records"].values()
                   if record["resource"].get("shard") in active]
        expect(len(records), len(active), f"{self.name}: MOSI records for every active {table} shard")
        return records

    def record(self, table):
        records = self.records(table)
        expect(len(records), 1, f"{self.name}: one shard for {table}")
        return records[0]

    def database_records(self, database=None):
        name = self.database if database is None else database
        schema = json.loads((self.data_dir / name / "schema.json").read_text())
        return database_records(name, schema, self.snapshot())

    def assert_exclusive(self, database=None, owner=None, state="M"):
        identity = self.snapshot()["id"]
        records = self.database_records(database)
        for key, record in records.items():
            expect((record["state"], record.get("owner")), (state, identity if owner is None else owner),
                   f"{self.name}: exclusive database resource {key}")
        return set(records)

    def reclaim(self):
        expect(self.client.scm(f"(replication_reclaim {json.dumps(self.database)})").strip(),
               "true", f"{self.name}: full database M reclaim")
        return self.assert_exclusive()

    def statuses(self):
        return json.loads(json.loads(self.client.scm("(replication_slaves)")))

    def connected(self):
        statuses = self.statuses()
        return len(statuses) == 1 and statuses[0]["connected"] and statuses[0]["ready"]

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
                   "--user=root", self.database, "--execute=" + statement]
        result = subprocess.run(command, env={**os.environ, "MYSQL_PWD": PASSWORD},
                                capture_output=True, text=True, timeout=self.timeout)
        if result.returncode:
            raise DrillFailure(f"MySQL login/query failed: {result.stderr}")
        return result.stdout.strip()


def database_records(name, schema, journal):
    """Use the current catalog, including empty/internal durable shards."""
    expected = {name: {"database": name}}
    for table_name, table in schema["tables"].items():
        if table_name.startswith(".") and table["PersistencyMode"] == "cache":
            continue
        shards = table.get("PShards" if table["ShardMode"] == "partition" else "Shards") or []
        for shard in shards:
            if shard is None:
                continue
            identity = shard.get("mosi", shard["uuid"]) if isinstance(shard, dict) else shard
            expected[f"{name}/{identity}"] = {"database": name, "table": table_name, "shard": identity}
    result = {}
    for key, resource in expected.items():
        if key not in journal["records"]:
            raise DrillFailure(f"missing MOSI resource: {key}")
        record = journal["records"][key]
        expect(record["resource"], resource, f"catalog/MOSI identity {key}")
        result[key] = record
    return result


class Cluster:
    def __init__(self, args, artifacts, name):
        directory = artifacts / name
        directory.mkdir()
        port = free_port() if args.dedicated_replication_port else None
        self.master = Node(args, directory, "master", str(port) if port else "listen")
        upstream = f"127.0.0.1:{port}" if port else f"http://127.0.0.1:{self.master.port}"
        self.slave = Node(args, directory, "slave", upstream, "different-local-password")

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
        identity = slave.snapshot()["id"]
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
            if round_index == 1:
                # After the slave itself restarted, recovered schema M must
                # still permit offline creation of brand-new data shards.
                s.sql("CREATE TABLE offline_restart_created(id INT PRIMARY KEY, payload TEXT) ENGINE=SAFE")
                s.sql("INSERT INTO offline_restart_created VALUES(1,'after-slave-restart')")
            for offset in range(5):
                row_id = round_index * 5 + offset + 2
                s.sql(f"INSERT INTO items VALUES({row_id},{row_id * 10})")
            s.sql("UPDATE items SET value=value+1 WHERE id=1")
            expect(slave.mysql("SELECT COUNT(*) FROM items"), str(6 + round_index * 5),
                   "offline MySQL authentication")
            master.start()
            eventually(master.connected, "slave did not reconnect to restarted master")
            def schema_backed_up():
                source = slave.snapshot()["records"].get(DATABASE, {}).get("generation")
                target = master.snapshot()["records"].get(DATABASE, {}).get("generation")
                return source and source == target
            eventually(schema_backed_up, "backup schema generation did not catch up")
            slave.kill()
            m.reject("SELECT * FROM items", status=500, contains=NETWORK_ERROR)
            s = slave.start()
            expect(s.number("SELECT COUNT(*) AS n FROM items"), 6 + round_index * 5, "durable inserts")
            expect(s.number("SELECT value FROM items WHERE id=1", "value"), 11 + round_index, "durable update")
            expect(s.sql("SELECT payload FROM offline_created WHERE id=1")[0]["payload"],
                   "survived-offline-ddl", "offline DDL recovery")
            expect(slave.snapshot()["id"], identity,
                   "persistent reconnect identity")
        # A master may become the modifier too. Its committed native WAL must
        # survive restart instead of recovering the slave's older surrendered
        # copy. Exercise the next slave write as well as reads on both nodes.
        m.sql("UPDATE items SET value=value+1000 WHERE id=1")
        expect((master.record("items")["state"], slave.record("items")["state"]),
               ("M", "IO"), "master owns committed write before crash")
        master.kill()
        m = master.start()
        eventually(master.connected, "master write owner did not recover")
        expect(m.number("SELECT value FROM items WHERE id=1", "value"), 1013,
               "master WAL after master crash")
        expect(s.number("SELECT value FROM items WHERE id=1", "value"), 1013,
               "slave receives latest master WAL after master crash")
        s.sql("UPDATE items SET value=value-1000 WHERE id=1")
        expect(s.number("SELECT value FROM items WHERE id=1", "value"), 13,
               "slave write preserves recovered master data")
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
        expect(s.number("SELECT COUNT(*) AS n FROM items"), 16, "failed write upgrade retains offline O read")
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
            # A closing handover may stop new writers, but the native
            # transaction already owning this shard must still finish.
            s.sql("INSERT INTO items VALUES(1001,998)", session=session)
            s.sql("COMMIT", session=session)
            expect(waiting.result(timeout=args.timeout), 18, "handover after transaction commit")
        expect(s.number("SELECT value FROM items WHERE id=1000", "value"), 999, "late handover")
        s.sql("DELETE FROM items WHERE id IN (1000,1001)")
        # Interrupt an actual ownership handover while its modifier still has
        # an open transaction. Losing the requester must reopen admission;
        # that transaction must finish locally, with both commit and rollback
        # surviving reconstruction at the restarted master.
        for outcome in ("COMMIT", "ROLLBACK"):
            slave.reclaim()
            session = uuid.uuid4().hex
            s.sql("START ACID TRANSACTION", session=session)
            s.sql("INSERT INTO items VALUES(1002,997)", session=session)
            with concurrent.futures.ThreadPoolExecutor(max_workers=1) as threads:
                waiting = threads.submit(m.number, "SELECT COUNT(*) AS n FROM items")
                try:
                    waiting.result(timeout=0.1)
                except concurrent.futures.TimeoutError:
                    pass
                else:
                    raise DrillFailure("interrupted handover crossed uncommitted rows")
                master.kill()
                try:
                    waiting.result(timeout=args.timeout)
                except (DrillFailure, OSError):
                    pass
                else:
                    raise DrillFailure("killed requester unexpectedly completed its query")
                s.sql("INSERT INTO items VALUES(1003,996)", session=session)
                s.sql(outcome, session=session)
            expected = 18 if outcome == "COMMIT" else 16
            expect(s.number("SELECT COUNT(*) AS n FROM items"), expected,
                   f"offline {outcome} after cancelled handover")
            m = master.start()
            eventually(master.connected, "interrupted handover did not reconnect")
            expect(m.number("SELECT COUNT(*) AS n FROM items"), expected,
                   f"reconstructed {outcome} after cancelled handover")
            expect(m.number("SELECT COUNT(*) AS n FROM items WHERE id IN (1002,1003)"),
                   2 if outcome == "COMMIT" else 0, f"{outcome} transaction visibility")
            slave.reclaim()
            s.sql("DELETE FROM items WHERE id IN (1002,1003)")
        Client(slave.port, args.timeout, password="wrong-password").reject("SELECT 1", status=401)
        expect(master.statuses()[0]["id"], identity, "dashboard identity")
        s.sql("SHUTDOWN")
        slave.wait_offline()
        m.sql("SELECT 1", "system")
        slave.kill()
        s = slave.start()
        s.sql("SELECT COUNT(*) AS n FROM items")
        def backed_up():
            destination = master.snapshot()["records"]
            for key, source in slave.snapshot()["records"].items():
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
        expect(m.sql("SELECT payload FROM offline_restart_created WHERE id=1")[0]["payload"],
               "after-slave-restart", "offline DDL after recovered slave restart")
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
        eventually(lambda: slave.snapshot().get("initialized", {}).get(DATABASE), "initial shard M claims")
        expect(m.number("SELECT value FROM items WHERE id=1", "value"), 10, "one-partition read")
        expect(sorted(record["state"] for record in master.records("items")), ["IO", "S"], "master partition rights")
        expect(sorted(record["state"] for record in slave.records("items")), ["M", "O"], "slave partition rights")
        shared_keys = {key for key, record in master.snapshot()["records"].items()
                       if record["state"] == "S"}
        master.kill()
        s.sql("UPDATE items SET value=value+1 WHERE id=9")
        s.reject("UPDATE items SET value=value+1 WHERE id=1", status=500, contains=NETWORK_ERROR)
        master.start()
        # A listening master has no recovered permissions yet. The surviving
        # slave must report S/O before we take that survivor down in turn.
        def shared_recovered():
            records = master.snapshot()["records"]
            return all(records.get(key, {}).get("state") == "S" for key in shared_keys)
        eventually(shared_recovered, "master did not reconstruct shared permissions from the live slave")
        slave.kill()
        expect(m.number("SELECT value FROM items WHERE id=1", "value"), 10, "live-recovered backend S")
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
        return actual

    def verify_pair(self, master, slave):
        """Call only at a mutation barrier; both nodes must agree exactly."""
        left, right = self.verify(master), self.verify(slave)
        expect(left, right, f"master/slave freshness for {master.database}")
        with self.lock:
            for row_id, allowed in self.rows.items():
                if len(allowed) > 1:
                    value = left.get(row_id)
                    self.rows[row_id] = {value}
                    self.log("resolved", row_id, value)


class WorkloadGate:
    """Pause new mutations for exact readback, without serializing writers."""
    def __init__(self, timeout):
        self.condition = threading.Condition()
        self.active = 0
        self.checking = False
        self.timeout = timeout

    @contextlib.contextmanager
    def mutation(self):
        with self.condition:
            if not self.condition.wait_for(lambda: not self.checking, self.timeout):
                raise DrillFailure("writer blocked at freshness checkpoint")
            self.active += 1
        try:
            yield
        finally:
            with self.condition:
                self.active -= 1
                self.condition.notify_all()

    @contextlib.contextmanager
    def checkpoint(self):
        with self.condition:
            if not self.condition.wait_for(lambda: not self.checking, self.timeout):
                raise DrillFailure("overlapping freshness checkpoints")
            self.checking = True
            if not self.condition.wait_for(lambda: self.active == 0, self.timeout):
                self.checking = False
                self.condition.notify_all()
                raise DrillFailure("in-flight writers did not reach freshness checkpoint")
        try:
            yield
        finally:
            with self.condition:
                self.checking = False
                self.condition.notify_all()


def workload(args, cluster, oracle, phase, *, analytics=False, crash_slave=False):
    stop = threading.Event()
    progressed = threading.Event()
    crashing = threading.Event()
    errors = queue.Queue()
    barrier = threading.Barrier(args.workers + (3 if analytics else 1))
    slave_client = cluster.slave.client
    master_client = Client(cluster.master.port, args.timeout, database=slave_client.database)
    gate = WorkloadGate(args.timeout)
    marker = 1000000000 + phase
    oracle.attempt(0, marker)
    slave_client.sql(f"UPDATE stress_items SET value={marker} WHERE id=0")
    oracle.acknowledge(0, marker)

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
                with gate.mutation():
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
                rows = master_client.sql("SELECT id,value FROM stress_items")
                sentinels = []
                for row in rows:
                    row_id, value = int(row["id"]), int(row["value"])
                    if row_id == 0:
                        sentinels.append(value)
                    elif value not in (row_id * 10, row_id * 10 + 1):
                        raise DrillFailure(f"analytic reader observed incomplete/corrupt row: {row}")
                expect(sentinels, [marker], "master must see acknowledged phase marker during concurrent writes")
                with gate.checkpoint():
                    oracle.verify_pair(master_client, slave_client)
                time.sleep(0.005)
        except Exception as error:
            errors.put(error)
            stop.set()

    def maintenance():
        try:
            barrier.wait(timeout=args.timeout)
            while not stop.is_set():
                cluster.master.client.scm(f'(blob_cleanup "{slave_client.database}")')
                cluster.slave.client.scm(f'(blob_cleanup "{slave_client.database}")')
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


def setup_stress_database(master, name):
    master.sql(f"CREATE DATABASE {name}", "system")
    for statement in ("CREATE TABLE stress_items(id INT PRIMARY KEY, value INT) ENGINE=SAFE",
                      "INSERT INTO stress_items VALUES(0,0)",
                      "CREATE TABLE stress_aux(id INT PRIMARY KEY, value INT) ENGINE=SAFE",
                      "INSERT INTO stress_aux VALUES(1,100),(9,900)",
                      "CREATE TABLE stress_empty(id INT PRIMARY KEY, value INT) ENGINE=SAFE"):
        master.sql(statement, name)
    expect(master.scm(f'(partitiontable (table "{name}" false "stress_aux" false) (list "id" (list 5)))').strip(),
           "true", "partitioned auxiliary table")


def table_values(client, table):
    rows = client.sql(f"SELECT id,value FROM `{table}`")
    result = {int(row["id"]): int(row["value"]) for row in rows}
    expect(len(rows), len(result), f"unique row IDs in {table}")
    return result


def verify_database_pair(master, slave, oracle, aux, extra=None):
    oracle.verify_pair(master, slave)
    for client in (master, slave):
        expect(table_values(client, "stress_aux"), aux, f"{client.database}: fresh partitioned data")
        expect(client.sql("SELECT * FROM stress_empty"), [], f"{client.database}: empty table")
        for table, expected in (extra or {}).items():
            expect(table_values(client, table), expected, f"{client.database}: offline schema/data {table}")


def assert_reclaimed(master, slave):
    keys = slave.reclaim()
    identity = slave.snapshot()["id"]
    expect(master.assert_exclusive(slave.database, owner=identity, state="IO"), keys,
           "same complete resource set, slave M/master IO")
    return len(keys)


def alternate_writers(master, slave, aux):
    # Equal row counts and plausible old values cannot satisfy these checks.
    for writer, reader in ((master, slave), (slave, master)):
        writer.sql("UPDATE stress_aux SET value=value+1")
        for row_id in aux:
            aux[row_id] += 1
        expect(table_values(reader, "stress_aux"), aux, "fresh values after remote write")
        expect(table_values(writer, "stress_aux"), aux, "fresh values at writer")


def run_stress(args, artifacts, journal):
    print(f"SCENARIO: threaded stress ({args.rounds} rounds, {args.workers} writers, seed {args.seed})", flush=True)
    oracle = Oracle(artifacts / "writes.jsonl")
    try:
        with Cluster(args, artifacts, "stress") as cluster:
            cluster.master.start()
            setup_stress_database(cluster.master.client, DATABASE)
            cluster.slave.start()
            cluster.slave.client.sql("SELECT * FROM stress_items")
            eventually(lambda: cluster.slave.snapshot().get("initialized", {}).get(DATABASE), "initial stress M claims")
            aux, extra = {1: 100, 9: 900}, {}
            master = cluster.master.client
            verify_database_pair(master, cluster.slave.client, oracle, aux)
            # A shared schema is sufficient to make offline reclaim impossible;
            # neither a timeout nor local backup files may manufacture M.
            cluster.master.kill()
            try:
                cluster.slave.client.scm(f'(replication_reclaim "{DATABASE}")')
            except DrillFailure as error:
                if NETWORK_ERROR not in str(error):
                    raise
            else:
                raise DrillFailure("reclaim succeeded with an unreachable S holder")
            expect(cluster.slave.snapshot()["records"][DATABASE]["state"], "O", "failed reclaim preserves durable shared authority")
            cluster.master.start()
            eventually(cluster.master.connected, "negative reclaim reconnect")
            for round_index in range(args.rounds):
                print(f"  round {round_index + 1}: online analytics/GC, master crash, slave crash", flush=True)
                phase = round_index * 3
                workload(args, cluster, oracle, phase, analytics=True)
                alternate_writers(master, cluster.slave.client, aux)
                verify_database_pair(master, cluster.slave.client, oracle, aux, extra)
                resource_count = assert_reclaimed(cluster.master, cluster.slave)
                cluster.master.kill()
                workload(args, cluster, oracle, phase + 1)
                table = f"offline_round_{round_index}"
                cluster.slave.client.sql(f"CREATE TABLE {table}(id INT PRIMARY KEY, value INT) ENGINE=SAFE")
                cluster.slave.client.sql(f"INSERT INTO {table} VALUES(1,{round_index})")
                extra[table] = {1: round_index}
                oracle.verify(cluster.slave.client)
                cluster.slave.assert_exclusive()
                cluster.master.start()
                eventually(cluster.master.connected, "stress slave did not reconnect")
                verify_database_pair(master, cluster.slave.client, oracle, aux, extra)
                assert_reclaimed(cluster.master, cluster.slave)
                workload(args, cluster, oracle, phase + 2, crash_slave=True)
                cluster.master.client.reject("SELECT * FROM stress_items", status=500, contains=NETWORK_ERROR)
                cluster.slave.start()
                verify_database_pair(master, cluster.slave.client, oracle, aux, extra)
                assert_reclaimed(cluster.master, cluster.slave)
                journal.append({"scenario": "threaded-stress", "round": round_index + 1,
                                "exclusive_resources": resource_count, "freshness": "both-nodes",
                                "acknowledged": oracle.acknowledged, "ambiguous": oracle.ambiguous})
            if oracle.acknowledged == 0:
                raise DrillFailure("stress completed without acknowledged writes")
    finally:
        oracle.events.close()


class MasterOnlyWorkload:
    """One persistent master SQL session; no slave may acquire this catalog."""
    def __init__(self, args, node, name):
        self.node, self.name, self.timeout = node, name, args.timeout
        self.client = Client(node.port, args.timeout, database=name)
        self.stop = threading.Event()
        self.condition = threading.Condition()
        self.cycles = 0
        self.error = None
        self.expected = {0: 0}
        self.finished = False
        self.thread = threading.Thread(target=self.run, name="master-only-ddl-and-data", daemon=True)

    def run(self):
        session = uuid.uuid4().hex
        query = lambda statement: self.client.sql(statement, session=session)
        try:
            sequence = 0
            while not self.stop.is_set():
                sequence += 1
                query(f"INSERT INTO master_items VALUES({sequence},{sequence * 10})")
                query(f"UPDATE master_items SET value=value+1 WHERE id={sequence}")
                self.expected[sequence] = sequence * 10 + 1
                if sequence % 3 == 0:
                    query(f"DELETE FROM master_items WHERE id={sequence - 1}")
                    del self.expected[sequence - 1]
                query(f"ALTER TABLE master_items ADD COLUMN cycle_marker INT DEFAULT {sequence}")
                rows = query("SELECT * FROM master_items")
                expect(len(rows), len(self.expected), "master-only row count during DDL")
                expect({int(row["id"]): row for row in rows},
                       {key: {"id": key, "value": value, "cycle_marker": sequence}
                        for key, value in self.expected.items()}, "master-only DDL/default/data freshness")
                query("ALTER TABLE master_items DROP COLUMN cycle_marker")
                rows = query("SELECT * FROM master_items")
                expect(len(rows), len(self.expected), "master-only row count after DROP")
                expect({int(row["id"]): row for row in rows},
                       {key: {"id": key, "value": value} for key, value in self.expected.items()},
                       "master-only schema after DROP and cached SELECT")
                self.node.assert_exclusive(self.name)
                with self.condition:
                    self.cycles += 1
                    self.condition.notify_all()
                self.stop.wait(0.01)
        except Exception as error:
            with self.condition:
                self.error = error
                self.condition.notify_all()

    def progress(self, previous):
        with self.condition:
            if not self.condition.wait_for(lambda: self.cycles > previous or self.error is not None, self.timeout):
                raise DrillFailure("master-only schema/data queries stalled while a slave was unavailable")
            if self.error is not None:
                raise self.error
            return self.cycles

    def finish(self):
        if self.finished:
            return
        self.finished = True
        self.stop.set()
        self.thread.join(self.timeout * 3)
        if self.thread.is_alive():
            raise DrillFailure("master-only thread did not stop")
        if self.error is not None:
            raise self.error
        expect(table_values(self.client, "master_items"), self.expected, "master-only final exact data")
        self.node.assert_exclusive(self.name)


def assert_separate_claims(master, slaves, master_only):
    for slave in slaves:
        forbidden = {master_only} | {other.database for other in slaves if other is not slave}
        for name in forbidden:
            if (slave.data_dir / name / "schema.json").exists():
                raise DrillFailure(f"{slave.name} unexpectedly replicated {name}")
        for record in slave.snapshot()["records"].values():
            if record["resource"]["database"] in forbidden:
                raise DrillFailure(f"{slave.name} has a cross-database MOSI claim: {record}")
    master.assert_exclusive(master_only)


def run_multiple_slaves(args, artifacts, journal):
    print("SCENARIO: two independent replication slaves plus concurrent master-only schema/data", flush=True)
    directory = artifacts / "multiple-slaves"
    directory.mkdir()
    port = free_port() if args.dedicated_replication_port else None
    master = Node(args, directory, "master", str(port) if port else "listen")
    upstream = f"127.0.0.1:{port}" if port else f"http://127.0.0.1:{master.port}"
    slaves = [Node(args, directory, "slave-a", upstream, "local-a", "replica_a"),
              Node(args, directory, "slave-b", upstream, "local-b", "replica_b")]
    master_only = "master_only"
    with contextlib.ExitStack() as owned:
        owned.callback(master.cleanup)
        for slave in slaves:
            owned.callback(slave.cleanup)
        master.start()
        for slave in slaves:
            setup_stress_database(master.client, slave.database)
            slave.start()
            slave.client.sql("SELECT * FROM stress_items")
            eventually(lambda s=slave: s.snapshot().get("initialized", {}).get(s.database), "per-slave initial M claims")
        master.client.sql(f"CREATE DATABASE {master_only}", "system")
        direct = Client(master.port, args.timeout, database=master_only)
        direct.sql("CREATE TABLE master_items(id INT PRIMARY KEY, value INT) ENGINE=SAFE")
        direct.sql("INSERT INTO master_items VALUES(0,0)")
        monitor = MasterOnlyWorkload(args, master, master_only)
        monitor.thread.start()
        owned.callback(monitor.finish)
        monitor.progress(0)
        oracles = [Oracle(directory / f"writes-{index}.jsonl") for index in range(2)]
        for oracle in oracles:
            owned.callback(oracle.events.close)
        pairs = [SimpleNamespace(master=master, slave=slave) for slave in slaves]
        for round_index in range(args.rounds):
            print(f"  multi-slave round {round_index + 1}: simultaneous independent databases", flush=True)
            with concurrent.futures.ThreadPoolExecutor(max_workers=2) as threads:
                futures = [threads.submit(workload, args, pair, oracle, round_index * 3, analytics=True)
                           for pair, oracle in zip(pairs, oracles)]
                for future in futures:
                    future.result()
            for slave, oracle in zip(slaves, oracles):
                reader = Client(master.port, args.timeout, database=slave.database)
                verify_database_pair(reader, slave.client, oracle, {1: 100, 9: 900})
                assert_reclaimed(master, slave)
            assert_separate_claims(master, slaves, master_only)
            for offline_index in range(2):
                offline = slaves[offline_index]
                live_index = 1 - offline_index
                live = slaves[live_index]
                offline.kill()
                Client(master.port, args.timeout, database=offline.database).reject(
                    "SELECT * FROM stress_items", status=500, contains=NETWORK_ERROR)
                before = monitor.cycles
                workload(args, pairs[live_index], oracles[live_index], round_index * 3 + offline_index + 1,
                         analytics=True)
                # Two completions ensure a full DDL/data cycle started after
                # the peer was already unavailable, even if one was in flight.
                monitor.progress(before + 1)
                verify_database_pair(Client(master.port, args.timeout, database=live.database), live.client,
                                     oracles[live_index], {1: 100, 9: 900})
                assert_reclaimed(master, live)
                offline.start()
                eventually(lambda: len([s for s in master.statuses() if s["connected"]]) == 2,
                           "both slaves did not reconnect")
                verify_database_pair(Client(master.port, args.timeout, database=offline.database), offline.client,
                                     oracles[offline_index], {1: 100, 9: 900})
                assert_reclaimed(master, offline)
                assert_separate_claims(master, slaves, master_only)
            journal.append({"scenario": "multiple-slaves", "round": round_index + 1,
                            "databases": [slave.database for slave in slaves],
                            "master_only_ddl_data_cycles": monitor.cycles})
        monitor.finish()
        for slave, oracle in zip(slaves, oracles):
            assert_reclaimed(master, slave)
        master.kill()
        with concurrent.futures.ThreadPoolExecutor(max_workers=2) as threads:
            futures = [threads.submit(workload, args, pair, oracle, args.rounds * 3 + index)
                       for index, (pair, oracle) in enumerate(zip(pairs, oracles))]
            for future in futures:
                future.result()
        for slave, oracle in zip(slaves, oracles):
            oracle.verify(slave.client)
            slave.assert_exclusive()
        master.start()
        eventually(lambda: len([s for s in master.statuses() if s["connected"]]) == 2,
                   "both slaves did not reconnect after master restart")
        expect(table_values(direct, "master_items"), monitor.expected, "master-only schema/data survived master crash")
        rows = direct.sql("SELECT * FROM master_items")
        expect(len(rows), len(monitor.expected), "master-only row count after master restart")
        expect({int(row["id"]): row for row in rows},
               {key: {"id": key, "value": value} for key, value in monitor.expected.items()},
               "master-only ADD/DROP schema survived master crash without resurrected columns")
        for index, (slave, oracle) in enumerate(zip(slaves, oracles)):
            reader = Client(master.port, args.timeout, database=slave.database)
            verify_database_pair(reader, slave.client, oracle, {1: 100, 9: 900})
            slave.kill()
            oracle.verify(reader)  # Its current S copy remains valid offline.
            slave.start()
            verify_database_pair(reader, slave.client, oracle, {1: 100, 9: 900})
            assert_reclaimed(master, slave)
        assert_separate_claims(master, slaves, master_only)
        journal.append({"scenario": "multiple-slave-master-restart", "freshness": "all-three-databases"})


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
    parser.add_argument("--dedicated-replication-port", action="store_true",
                        help="exercise the optional dedicated listener instead of the normal HTTP /cluster route")
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
                "timeout": args.timeout, "startup_timeout": args.startup_timeout,
                "dedicated_replication_port": args.dedicated_replication_port}]
    print(f"Replication cluster stress artifacts: {artifacts}\nSeed: {args.seed}", flush=True)
    try:
        run_regressions(args, artifacts, journal)
        run_partitions(args, artifacts, journal)
        run_stress(args, artifacts, journal)
        run_multiple_slaves(args, artifacts, journal)
        # Also catch a race which terminates a process after its last response
        # or during an expected shutdown; those exits cannot be inferred solely
        # from SQL response validation.
        for log in artifacts.rglob("memcp-generation-*.log"):
            with log.open("rb") as lines:
                if any(b"WARNING: DATA RACE" in line for line in lines):
                    raise DrillFailure(f"server race report: {log}")
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
