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


MODULE_PATH = Path(__file__).with_name("reliability_drill.py")
SPEC = importlib.util.spec_from_file_location("reliability_drill", MODULE_PATH)
assert SPEC is not None and SPEC.loader is not None
drill = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(drill)


class ReliabilityDrillTests(unittest.TestCase):
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


if __name__ == "__main__":
	unittest.main()
