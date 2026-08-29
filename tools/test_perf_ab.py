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

from perf_ab import (
    PerformanceFailure,
    compare_measurements,
    parse_runner_output,
    performance_cases,
)


def measurement(
    total_ms: float, rows: int = 10_000, limit: float = 20,
    repetitions: int = 1, warmup: int = 2,
) -> dict:
    return {
        "total_ms": total_ms,
        "repetitions": repetitions,
        "warmup": warmup,
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

    def test_repetition_changes_are_normalized(self) -> None:
        comparisons, errors = compare_measurements(
            {"suite::query#1": measurement(500, repetitions=5)},
            {"suite::query#1": measurement(9_000, repetitions=100)},
        )
        self.assertEqual(errors, [])
        self.assertAlmostEqual(comparisons[0]["ratio"], 0.9)

    def test_disabling_warmup_adds_one_hundred_percentage_points(self) -> None:
        comparisons, errors = compare_measurements(
            {"suite::query#1": measurement(500, repetitions=5, warmup=2)},
            {"suite::query#1": measurement(1_100, repetitions=5, warmup=0)},
        )
        self.assertEqual(errors, [])
        self.assertEqual(comparisons[0]["warmup_bonus_pct"], 100)

        comparisons, errors = compare_measurements(
            {"suite::query#1": measurement(500, repetitions=5, warmup=2)},
            {"suite::query#1": measurement(1_105, repetitions=5, warmup=0)},
        )
        self.assertTrue(comparisons[0]["regressed"])
        self.assertRegex(errors[0], "121.0% slower")

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
                "QUERY_TIME median_ns=125000000 total_ns=625000000 min_ns=100000000 "
                "max_ns=150000000 n=5 warmup=2 name=measured query\n"
                "✅ measured query (125.0ms / 30000ms, 10,000 rows, 12.50µs/row)\n"
            )
            payload = parse_runner_output(output, suite)
        self.assertEqual(payload["measurements"][0]["total_ms"], 625)
        self.assertEqual(payload["measurements"][0]["repetitions"], 5)
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

    def test_cubic_matrix_uses_safe_bootstrap_dimension(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            suite = Path(directory) / "baseline.yaml"
            suite.write_text(
                "metadata: {description: test}\n"
                "test_cases:\n"
                "  - name: 'Perf: MATRIX MULT'\n"
                "    threshold_ms: 30000\n"
                "    sql: SELECT 1\n",
                encoding="utf-8",
            )
            cases = performance_cases(
                suite, "tests/performance/baseline.yaml"
            )
        self.assertEqual(cases[0]["rows"], 100)


if __name__ == "__main__":
    unittest.main()
