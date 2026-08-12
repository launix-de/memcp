#!/usr/bin/env python3
#
# Copyright (C) 2026  Carl-Philip Haensch
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

"""Regression tests for SQL runner response classification."""

from pathlib import Path
from types import SimpleNamespace
import sys
import tempfile
import threading
import time
import unittest


sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from run_sql_tests import SQLTestRunner, is_error_response, observe_atomic_json  # noqa: E402


class ErrorResponseContractTest(unittest.TestCase):
    def test_missing_response_is_not_a_sql_error(self) -> None:
        self.assertFalse(is_error_response(None))

    def test_success_response_is_not_a_sql_error(self) -> None:
        response = SimpleNamespace(status_code=200, text='{"value": 1}')
        self.assertFalse(is_error_response(response))

    def test_http_error_is_a_sql_error(self) -> None:
        response = SimpleNamespace(status_code=500, text="SQL Error: invalid query")
        self.assertTrue(is_error_response(response))

    def test_error_payload_is_a_sql_error(self) -> None:
        response = SimpleNamespace(status_code=200, text="Error: invalid query")
        self.assertTrue(is_error_response(response))


class AtomicJSONObserverContractTest(unittest.TestCase):
    def test_accepts_complete_atomic_replacements(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "schema.json"
            path.write_text('{"generation": 0}', encoding="utf-8")

            def replace() -> None:
                for generation in range(1, 30):
                    candidate = path.with_suffix(".tmp")
                    candidate.write_text(f'{{"generation": {generation}}}', encoding="utf-8")
                    candidate.replace(path)

            writer = threading.Thread(target=replace)
            writer.start()
            reads, error = observe_atomic_json(path, 0.05)
            writer.join()
            self.assertGreater(reads, 0)
            self.assertIsNone(error)

    def test_rejects_partial_generation(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "schema.json"
            path.write_text('{"generation": 0}', encoding="utf-8")

            def truncate() -> None:
                time.sleep(0.01)
                path.write_text('{"generation":', encoding="utf-8")

            writer = threading.Thread(target=truncate)
            writer.start()
            reads, error = observe_atomic_json(path, 0.1)
            writer.join()
            self.assertGreater(reads, 0)
            self.assertIsNotNone(error)

    def test_rejects_rename_away_before_replacement(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "schema.json"
            backup = Path(tmp) / "schema.json.old"
            path.write_text('{"generation": 0}', encoding="utf-8")

            def replace_nonatomically() -> None:
                time.sleep(0.01)
                path.replace(backup)
                path.write_text('{"generation": 1}', encoding="utf-8")

            writer = threading.Thread(target=replace_nonatomically)
            writer.start()
            reads, error = observe_atomic_json(path, 0.1)
            writer.join()
            self.assertGreater(reads, 0)
            self.assertIsNotNone(error)


class FailFastParallelContractTest(unittest.TestCase):
    def test_fail_fast_preserves_parallel_groups(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            spec = Path(tmp) / "parallel.yaml"
            spec.write_text(
                "metadata:\n"
                "  description: parallel contract\n"
                "test_cases:\n"
                "  - name: first\n"
                "    parallel: together\n"
                "  - name: second\n"
                "    parallel: together\n",
                encoding="utf-8",
            )
            runner = SQLTestRunner("http://localhost:1", fail_fast=True)
            runner.ensure_database = lambda _database: None
            barrier = threading.Barrier(2)
            completed = []

            def run_case(test_case, _database):
                barrier.wait(timeout=1)
                completed.append(test_case["name"])
                runner.test_count += 1
                runner.test_passed += 1
                return True

            runner.run_test_case = run_case
            self.assertTrue(runner.run_test_spec(str(spec)))
            self.assertCountEqual(completed, ["first", "second"])

if __name__ == "__main__":
    unittest.main()
