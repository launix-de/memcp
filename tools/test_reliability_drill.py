#!/usr/bin/env python3
# Copyright (C) 2026 Carl-Philip Haensch
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

"""Fast unit tests for reliability-drill safety and result handling."""

from __future__ import annotations

import importlib.util
from pathlib import Path
import tempfile
import unittest
from unittest import mock

import replication_cluster_stress as replication


MODULE_PATH = Path(__file__).with_name("reliability_drill.py")
SPEC = importlib.util.spec_from_file_location("reliability_drill", MODULE_PATH)
assert SPEC is not None and SPEC.loader is not None
drill = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(drill)


class ReliabilityDrillTests(unittest.TestCase):
	def test_ports_remain_distinct_before_nodes_start_listening(self) -> None:
		with mock.patch.object(drill, "_allocated_ports", set()), mock.patch.object(drill.socket, "socket") as factory:
			probe = factory.return_value.__enter__.return_value
			probe.getsockname.side_effect = [("127.0.0.1", 44603), ("127.0.0.1", 44603), ("127.0.0.1", 60783)]
			self.assertEqual(drill.free_port(), 44603)
			self.assertEqual(drill.free_port(), 60783)
			self.assertEqual(probe.bind.call_count, 3)

	def test_remote_database_options_are_emitted_as_sql_literals(self) -> None:
		class RecordingClient:
			def __init__(self) -> None:
				self.calls = []

			def sql(self, statement: str, database: str) -> None:
				self.calls.append((statement, database))

		client = RecordingClient()
		drill.create_database(client, {
			"backend": "s3", "force_path_style": True,
			"prefix": "ci's-prefix",
		})
		self.assertEqual(client.calls, [(
			"CREATE DATABASE IF NOT EXISTS reliability_drill SET "
			"backend='s3',force_path_style='true',prefix='ci''s-prefix'",
			"system",
		)])
		with self.assertRaises(drill.DrillFailure):
			drill.create_database(client, {"backend; DROP DATABASE system": "s3"})

	def test_artifact_directory_must_not_exist(self) -> None:
		with tempfile.TemporaryDirectory() as temp:
			with self.assertRaises(drill.DrillFailure):
				drill.make_artifact_dir(Path(temp))

	def test_artifact_directory_is_created_without_reusing_parent(self) -> None:
		with tempfile.TemporaryDirectory() as temp:
			path = Path(temp) / "new-run"
			self.assertEqual(drill.make_artifact_dir(path), path.resolve())
			self.assertTrue(path.is_dir())

	def test_expect_rejects_inexact_recovery(self) -> None:
		with self.assertRaises(drill.DrillFailure):
			drill.expect({"count": 2}, {"count": 3}, "recovery")

	def test_io_fault_environment_is_reproducible_and_scoped(self) -> None:
		environment = drill.io_fault_environment("log.write", "partial", 42)
		self.assertEqual(environment["MEMCP_IO_FAULT_PROBABILITY"], "1")
		self.assertEqual(environment["MEMCP_IO_FAULT_SEED"], "42")
		self.assertEqual(environment["MEMCP_IO_FAULT_OPERATIONS"], "log.write")
		self.assertEqual(environment["MEMCP_IO_FAULT_PHASE"], "partial")
		self.assertEqual(environment["MEMCP_IO_FAULT_LIMIT"], "1")
		self.assertEqual(environment["MEMCP_IO_FAULT_DATABASE"], drill.DATABASE)

	def test_atomicity_oracle_accepts_only_complete_generations(self) -> None:
		self.assertEqual(drill.classify_atomic_signature({
			"row_count": 100, "min_x": 4, "max_x": 4, "x_sum": 400,
		}, 100, 4), 4)
		self.assertEqual(drill.classify_atomic_signature({
			"row_count": 100, "min_x": 5, "max_x": 5, "x_sum": 500,
		}, 100, 4), 5)
		with self.assertRaises(drill.DrillFailure):
			drill.classify_atomic_signature({
				"row_count": 73, "min_x": 5, "max_x": 5, "x_sum": 365,
			}, 100, 4)

	def test_schema_race_mode_enables_fail_fast_race_reporting(self) -> None:
		self.assertEqual(
			drill.race_detector_environment(),
			{"GORACE": "halt_on_error=1 exitcode=66"},
		)
		with mock.patch("sys.argv", ["reliability_drill.py", "--mode", "schema-race"]):
			args = drill.parse_args()
		self.assertEqual(args.mode, "schema-race")
		self.assertGreater(args.schema_race_rows, 0)
		self.assertGreater(args.schema_race_rounds, 0)


class ReplicationOracleTests(unittest.TestCase):
	def setUp(self) -> None:
		directory = tempfile.TemporaryDirectory()
		self.addCleanup(directory.cleanup)
		self.oracle = replication.Oracle(Path(directory.name) / "writes.jsonl")
		self.addCleanup(self.oracle.events.close)

	def check_rows(self, rows) -> None:
		client = mock.Mock()
		client.sql.return_value = [{"id": 0, "value": 0}] + rows
		self.oracle.verify(client)

	def test_acknowledged_insert_cannot_disappear(self) -> None:
		self.oracle.attempt(1, 10)
		self.oracle.acknowledge(1, 10)
		self.check_rows([{"id": 1, "value": 10}])
		with self.assertRaises(replication.DrillFailure):
			self.check_rows([])

	def test_unanswered_insert_allows_only_absence_or_complete_row(self) -> None:
		self.oracle.attempt(1, 10)
		self.oracle.uncertain(1, 10)
		self.check_rows([])
		self.check_rows([{"id": 1, "value": 10}])
		with self.assertRaises(replication.DrillFailure):
			self.check_rows([{"id": 1, "value": 999}])

	def test_unanswered_update_cannot_erase_acknowledged_insert(self) -> None:
		self.oracle.attempt(1, 10)
		self.oracle.acknowledge(1, 10)
		self.oracle.attempt(1, 11)
		self.oracle.uncertain(1, 11)
		for value in (10, 11):
			self.check_rows([{"id": 1, "value": value}])
		with self.assertRaises(replication.DrillFailure):
			self.check_rows([])
		self.oracle.acknowledge(1, 11)
		with self.assertRaises(replication.DrillFailure):
			self.check_rows([{"id": 1, "value": 10}])

	def test_acknowledged_delete_cannot_reappear(self) -> None:
		self.oracle.attempt(1, 10)
		self.oracle.acknowledge(1, 10)
		self.oracle.attempt(1, None)
		self.oracle.acknowledge(1, None)
		self.check_rows([])
		with self.assertRaises(replication.DrillFailure):
			self.check_rows([{"id": 1, "value": 10}])

	def test_extra_or_duplicate_rows_fail(self) -> None:
		with self.assertRaises(replication.DrillFailure):
			self.check_rows([{"id": 999, "value": 9990}])
		with self.assertRaises(replication.DrillFailure):
			self.check_rows([{"id": 0, "value": 0}])

	def pair(self, left, right):
		clients = [mock.Mock(database="replica_test") for _ in range(2)]
		for client, rows in zip(clients, (left, right)):
			client.sql.return_value = [{"id": 0, "value": 0}] + rows
		return clients

	def test_plausible_stale_master_value_fails(self) -> None:
		self.oracle.acknowledge(1, 11)
		with self.assertRaises(replication.DrillFailure):
			self.oracle.verify_pair(*self.pair([{"id": 1, "value": 10}], [{"id": 1, "value": 11}]))

	def test_nodes_cannot_disagree_on_unanswered_write(self) -> None:
		self.oracle.acknowledge(1, 10)
		self.oracle.attempt(1, 11)
		self.oracle.uncertain(1, 11)
		with self.assertRaises(replication.DrillFailure):
			self.oracle.verify_pair(*self.pair([{"id": 1, "value": 10}], [{"id": 1, "value": 11}]))

	def test_recovered_ambiguous_outcome_cannot_change_later(self) -> None:
		self.oracle.attempt(1, 10)
		self.oracle.uncertain(1, 10)
		self.oracle.verify_pair(*self.pair([], []))
		with self.assertRaises(replication.DrillFailure):
			self.check_rows([{"id": 1, "value": 10}])


class ReplicationCatalogTests(unittest.TestCase):
	def fixture(self):
		schema = {"tables": {
			"empty": {"PersistencyMode": "safe", "ShardMode": "free", "Shards": [{"uuid": "empty-shard", "mosi": "ignored-legacy-id"}]},
			"partitioned": {"PersistencyMode": "safe", "ShardMode": "partition", "PShards": [None, "part-a", "part-b"]},
			".blobs": {"PersistencyMode": "safe", "ShardMode": "free", "Shards": ["blobs"]},
			".temporary": {"PersistencyMode": "cache", "ShardMode": "free", "Shards": ["excluded"]},
		}}
		records = {"db": {"resource": {"database": "db"}}}
		for table, shards in (("empty", ["empty-shard"]), ("partitioned", ["part-a", "part-b"]), (".blobs", ["blobs"])):
			for shard in shards:
				records[f"db/{shard}"] = {"resource": {"database": "db", "table": table, "shard": shard}}
		records["db/retired"] = {"resource": {"database": "db", "table": "empty", "shard": "retired"}}
		return schema, {"records": records}

	def test_reclaim_checks_schema_empty_partitioned_and_internal_shards(self):
		schema, journal = self.fixture()
		self.assertEqual(set(replication.database_records("db", schema, journal)),
			{"db", "db/empty-shard", "db/part-a", "db/part-b", "db/blobs"})

	def test_missing_empty_shard_cannot_be_declared_offline_ready(self):
		schema, journal = self.fixture()
		del journal["records"]["db/empty-shard"]
		with self.assertRaises(replication.DrillFailure):
			replication.database_records("db", schema, journal)

	def test_wrong_table_identity_cannot_satisfy_reclaim(self):
		schema, journal = self.fixture()
		journal["records"]["db/part-a"]["resource"]["table"] = "wrong"
		with self.assertRaises(replication.DrillFailure):
			replication.database_records("db", schema, journal)

	def test_reclaim_resolves_catalog_predecessor_through_live_authority(self):
		schema, snapshot = self.fixture()
		previous = snapshot["records"].pop("db/empty-shard")
		current = {**previous["resource"], "shard": "rebuilt"}
		snapshot["records"]["db/rebuilt"] = {"resource": current}
		snapshot["authority"] = {"db/empty-shard": {"resource": current}}
		self.assertEqual(set(replication.database_records("db", schema, snapshot)),
			{"db", "db/rebuilt", "db/part-a", "db/part-b", "db/blobs"})

	def test_forwarded_reclaim_still_requires_exact_current_resource(self):
		for invalid in ["missing-record", "wrong-table", "wrong-database"]:
			with self.subTest(invalid=invalid):
				schema, snapshot = self.fixture()
				del snapshot["records"]["db/empty-shard"]
				current = {"database": "db", "table": "empty", "shard": "rebuilt"}
				if invalid == "wrong-table":
					current["table"] = "partitioned"
				elif invalid == "wrong-database":
					current["database"] = "another"
				snapshot["authority"] = {"db/empty-shard": {"resource": current}}
				with self.assertRaises(replication.DrillFailure):
					replication.database_records("db", schema, snapshot)

	def test_two_active_catalog_slots_cannot_share_one_forwarded_resource(self):
		schema, snapshot = self.fixture()
		current = snapshot["records"]["db/part-a"]["resource"]
		snapshot["authority"] = {"db/part-b": {"resource": current}}
		with self.assertRaises(replication.DrillFailure):
			replication.database_records("db", schema, snapshot)

	def test_named_table_records_use_canonical_authority(self):
		schema, snapshot = self.fixture()
		record = snapshot["records"].pop("db/empty-shard")
		current = {**record["resource"], "shard": "rebuilt"}
		snapshot["records"]["db/rebuilt"] = {"resource": current}
		snapshot["authority"] = {"db/empty-shard": {"resource": current}}
		with tempfile.TemporaryDirectory() as directory:
			node = mock.Mock(spec=replication.Node)
			node.data_dir, node.database, node.name = Path(directory), "db", "named"
			node.snapshot.return_value = snapshot
			path = node.data_dir / "db"
			path.mkdir()
			(path / "schema.json").write_text(replication.json.dumps(schema))
			self.assertEqual(replication.Node.records(node, "empty"), [{"resource": current}])

class ClusterSQLContextRunnerTests(unittest.TestCase):
	def test_sql_failures_are_collected_instead_of_short_circuiting_or_xfail(self):
		client = mock.Mock()
		client.sql.side_effect = replication.DrillFailure("SQL contract failed")
		client.scm.return_value = "50"
		node = mock.Mock(name="standalone")
		node.name, node.client = "standalone", client
		reference = {"v0_sum": 10, "rows": 1, "partition_rows": 1, "first_value": 10,
			"first_partition_sum": 10, "first_two_partition_sum": 10, "nonlinear_sum": 53}
		journal = []
		with tempfile.TemporaryDirectory() as directory:
			report = Path(directory) / "contexts.json"
			with mock.patch("builtins.print"):
				cases = replication.run_cluster_sql_contexts([node], reference, journal, report)
			self.assertEqual(len(cases), 14)
			self.assertEqual(journal, cases)
			self.assertTrue(all(case["status"] == "failed" for case in cases))
			self.assertIn('"status": "failed"', report.read_text())
			self.assertTrue(all(step["sql"].split()[0] in {"SELECT", "SET", "START", "ROLLBACK"}
				for case in cases for step in case["steps"]))

	def test_correct_local_result_cannot_pass_a_forced_remote_contract(self):
		client = mock.Mock()
		client.sql.return_value = [{"total": 51}]
		client.scm.return_value = "50"
		node = mock.Mock()
		node.name, node.client = "coordinator", client
		reference = {"v0_sum": 10, "rows": 1, "partition_rows": 1, "first_value": 10,
			"first_partition_sum": 10, "first_two_partition_sum": 10, "nonlinear_sum": 53}
		with tempfile.TemporaryDirectory() as directory:
			with mock.patch("builtins.print"):
				cases = replication.run_cluster_sql_contexts([node], reference, [],
					Path(directory) / "contexts.json", lambda _: {"state": {"delegated": 0}})
		self.assertEqual(cases[0]["name"], "literal-capture")
		self.assertEqual(cases[0]["status"], "failed")
		self.assertIn("without executing a remote shard", cases[0]["error"])


if __name__ == "__main__":
	unittest.main()
