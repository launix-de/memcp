#!/usr/bin/env python3
#
# Copyright (C) 2026  Carl-Philip Hänsch
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

import tempfile
import unittest
from pathlib import Path

from perf_ab import PerformanceFailure, compare_measurements, parse_runner_output


def measurement(time_ms: float, rows: int = 10_000, limit: float = 20) -> dict:
    return {
        "time_ms": time_ms,
        "rows": rows,
        "max_regression_pct": limit,
    }


class PerformanceComparatorTest(unittest.TestCase):
    def test_one_regression_fails_the_comparison(self) -> None:
        comparisons, errors = compare_measurements(
            {"suite::query#1": measurement(100, limit=10)},
            {"suite::query#1": measurement(111, limit=10)},
        )
        self.assertTrue(comparisons[0]["regressed"])
        self.assertRegex(errors[0], "11.0% slower")

    def test_boundary_is_allowed(self) -> None:
        comparisons, errors = compare_measurements(
            {"suite::query#1": measurement(100)},
            {"suite::query#1": measurement(120)},
        )
        self.assertFalse(comparisons[0]["regressed"])
        self.assertEqual(errors, [])

    def test_missing_candidate_measurement_fails_closed(self) -> None:
        _comparisons, errors = compare_measurements(
            {"suite::query#1": measurement(100)}, {}
        )
        self.assertRegex(errors[0], "candidate measurement is missing")

    def test_workloads_must_match(self) -> None:
        _comparisons, errors = compare_measurements(
            {"suite::query#1": measurement(100, rows=10_000)},
            {"suite::query#1": measurement(100, rows=5_000)},
        )
        self.assertRegex(errors[0], "workload differs")

    def test_trusted_yaml_supplies_limit_and_output_supplies_time(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            suite = Path(directory) / "suite.yaml"
            suite.write_text(
                "metadata:\n"
                "  description: test\n"
                "test_cases:\n"
                "  - name: measured query\n"
                "    threshold_ms: 30000\n"
                "    max_regression_pct: 15\n"
                "    sql: SELECT 1\n",
                encoding="utf-8",
            )
            output = (
                "QUERY_TIME median_ns=9000000 min_ns=8000000 "
                "max_ns=10000000 n=1 name=correctness-only query\n"
                "QUERY_TIME median_ns=125000000 min_ns=100000000 "
                "max_ns=150000000 n=5 name=measured query\n"
                "✅ measured query (125.0ms / 30000ms, 10,000 rows, 12.50µs/row)\n"
            )
            payload = parse_runner_output(output, suite)
        self.assertEqual(payload["measurements"][0]["time_ms"], 125)
        self.assertEqual(payload["measurements"][0]["rows"], 10_000)
        self.assertEqual(payload["measurements"][0]["max_regression_pct"], 15)

    def test_partial_output_fails_closed(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            suite = Path(directory) / "suite.yaml"
            suite.write_text(
                "metadata: {description: test}\n"
                "test_cases:\n"
                "  - {name: measured query, threshold_ms: 30000, sql: SELECT 1}\n",
                encoding="utf-8",
            )
            with self.assertRaisesRegex(PerformanceFailure, "expected 1 measurements"):
                parse_runner_output("", suite)


if __name__ == "__main__":
    unittest.main()
