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

import copy
import unittest

from check_performance_policy import PolicyFailure, compare_suite


def suite() -> dict:
    return {
        "metadata": {"description": "performance"},
        "test_cases": [
            {
                "name": "protected query",
                "sql": "SELECT COUNT(*) FROM items",
                "threshold_ms": 30_000,
                "max_regression_pct": 15,
                "timing_samples": 7,
                "expect": {"rows": 1},
            }
        ],
    }


class PerformancePolicyTest(unittest.TestCase):
    def test_unchanged_suite_passes(self) -> None:
        document = suite()
        compare_suite(document, copy.deepcopy(document), "tests/performance/test.yaml")

    def test_case_deletion_is_rejected(self) -> None:
        base = suite()
        candidate = copy.deepcopy(base)
        candidate["test_cases"] = []
        with self.assertRaisesRegex(PolicyFailure, "deleted or renamed"):
            compare_suite(base, candidate, "tests/performance/test.yaml")

    def test_query_replacement_is_rejected(self) -> None:
        base = suite()
        candidate = copy.deepcopy(base)
        candidate["test_cases"][0]["sql"] = "SELECT 1"
        with self.assertRaisesRegex(PolicyFailure, "protected benchmark input"):
            compare_suite(base, candidate, "tests/performance/test.yaml")

    def test_regression_limit_cannot_be_relaxed(self) -> None:
        base = suite()
        candidate = copy.deepcopy(base)
        candidate["test_cases"][0]["max_regression_pct"] = 20
        with self.assertRaisesRegex(PolicyFailure, "relaxed from 15% to 20%"):
            compare_suite(base, candidate, "tests/performance/test.yaml")

    def test_sample_count_cannot_be_reduced(self) -> None:
        base = suite()
        candidate = copy.deepcopy(base)
        candidate["test_cases"][0]["timing_samples"] = 5
        with self.assertRaisesRegex(PolicyFailure, "reduced from 7 to 5"):
            compare_suite(base, candidate, "tests/performance/test.yaml")

    def test_tighter_limit_and_more_samples_are_allowed(self) -> None:
        base = suite()
        candidate = copy.deepcopy(base)
        candidate["test_cases"][0]["max_regression_pct"] = 10
        candidate["test_cases"][0]["timing_samples"] = 9
        compare_suite(base, candidate, "tests/performance/test.yaml")


if __name__ == "__main__":
    unittest.main()
