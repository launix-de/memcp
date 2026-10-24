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
			"empty": {"PersistencyMode": "safe", "ShardMode": "free", "Shards": [{"uuid": "empty-generation", "mosi": "empty-lineage"}]},
			"partitioned": {"PersistencyMode": "safe", "ShardMode": "partition", "PShards": [None, "part-a", "part-b"]},
			".blobs": {"PersistencyMode": "safe", "ShardMode": "free", "Shards": ["blobs"]},
			".temporary": {"PersistencyMode": "cache", "ShardMode": "free", "Shards": ["excluded"]},
		}}
		records = {"db": {"resource": {"database": "db"}}}
		for table, shards in (("empty", ["empty-lineage"]), ("partitioned", ["part-a", "part-b"]), (".blobs", ["blobs"])):
			for shard in shards:
				records[f"db/{shard}"] = {"resource": {"database": "db", "table": table, "shard": shard}}
		records["db/retired"] = {"resource": {"database": "db", "table": "empty", "shard": "retired"}}
		return schema, {"records": records}

	def test_reclaim_checks_schema_empty_partitioned_and_internal_shards(self):
		schema, journal = self.fixture()
		self.assertEqual(set(replication.database_records("db", schema, journal)),
			{"db", "db/empty-lineage", "db/part-a", "db/part-b", "db/blobs"})

	def test_missing_empty_shard_cannot_be_declared_offline_ready(self):
		schema, journal = self.fixture()
		del journal["records"]["db/empty-lineage"]
		with self.assertRaises(replication.DrillFailure):
			replication.database_records("db", schema, journal)

	def test_wrong_table_identity_cannot_satisfy_reclaim(self):
		schema, journal = self.fixture()
		journal["records"]["db/part-a"]["resource"]["table"] = "wrong"
		with self.assertRaises(replication.DrillFailure):
			replication.database_records("db", schema, journal)


if __name__ == "__main__":
	unittest.main()
