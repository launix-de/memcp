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

"""Unit tests for the SQL and planner regression guard."""

from __future__ import annotations

from pathlib import Path
import shutil
import tempfile
import unittest

from check_sql_time_limit import GuardFailure
from check_sql_time_limit import check_added_planner_lines
from check_sql_time_limit import check_runner
from check_sql_time_limit import compare_suites
from check_sql_time_limit import load_suite
from check_sql_time_limit import validate_suite


def suite(case: dict | None = None, metadata: dict | None = None) -> dict:
    return {
        "metadata": metadata or {},
        "test_cases": [
            case or {"name": "regression", "sql": "SELECT 1", "expect": {"rows": 1}}
        ],
    }


class CurrentTreeValidationTest(unittest.TestCase):
    def runner_fixture(self) -> tuple[tempfile.TemporaryDirectory, Path]:
        temporary = tempfile.TemporaryDirectory()
        root = Path(temporary.name)
        repository = Path(__file__).resolve().parents[1]
        (root / "tools").mkdir()
        (root / ".github" / "workflows").mkdir(parents=True)
        for relative in (
            "run_sql_tests.py",
            "git-pre-commit",
            ".github/workflows/test.yml",
        ):
            target = root / relative
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(repository / relative, target)
        return temporary, root

    def test_current_runner_keeps_calibration_contract(self) -> None:
        check_runner(Path(__file__).resolve().parents[1])

    def test_runner_rejects_a_relaxed_machine_scale_cap(self) -> None:
        temporary, root = self.runner_fixture()
        with temporary:
            runner = root / "run_sql_tests.py"
            runner.write_text(
                runner.read_text(encoding="utf-8").replace(
                    "PERFORMANCE_SCALE_MAX = 16.0", "PERFORMANCE_SCALE_MAX = 160.0"
                ),
                encoding="utf-8",
            )
            with self.assertRaisesRegex(GuardFailure, "protected literal"):
                check_runner(root)

    def test_runner_rejects_calibration_work_that_can_be_tampered_with(self) -> None:
        temporary, root = self.runner_fixture()
        with temporary:
            runner = root / "run_sql_tests.py"
            runner.write_text(
                runner.read_text(encoding="utf-8").replace(
                    "hashlib.sha256(payload).digest()",
                    "time.sleep(0.1)",
                    1,
                ),
                encoding="utf-8",
            )
            with self.assertRaisesRegex(GuardFailure, "implementation changed"):
                check_runner(root)

    def test_runner_rejects_scaling_any_gate_other_than_max_time(self) -> None:
        temporary, root = self.runner_fixture()
        with temporary:
            runner = root / "run_sql_tests.py"
            runner.write_text(
                runner.read_text(encoding="utf-8").replace(
                    "plan_size_limit = int(test_case[\"max_plan_size\"])",
                    "plan_size_limit = scaled_wall_clock_limit_ms(1, self.performance_calibration)",
                    1,
                ),
                encoding="utf-8",
            )
            with self.assertRaisesRegex(GuardFailure, "exactly once"):
                check_runner(root)

    def test_structured_yaml_rejects_quoted_time_bypass(self) -> None:
        value = load_suite(
            "metadata: {}\ntest_cases:\n  - name: bypass\n    max_time: '6e0'\n    sql: SELECT 1\n",
            "tests/planner/bypass.yaml",
        )
        with self.assertRaisesRegex(GuardFailure, "at most 5.0"):
            validate_suite(value, "tests/planner/bypass.yaml")

    def test_zero_plan_size_cannot_disable_gate(self) -> None:
        with self.assertRaisesRegex(GuardFailure, "zero disables"):
            validate_suite(
                suite({"name": "bypass", "max_plan_size": 0}),
                "tests/planner/bypass.yaml",
            )

    def test_ci_opt_out_is_limited_to_manual_benchmarks(self) -> None:
        with self.assertRaisesRegex(
            GuardFailure, "reserved for manual performance benchmarks"
        ):
            validate_suite(
                suite(metadata={"ci": False}),
                "tests/planner/bypass.yaml",
            )

    def test_disabled_case_is_rejected(self) -> None:
        with self.assertRaisesRegex(GuardFailure, "disabled=true"):
            validate_suite(
                suite({"name": "bypass", "disabled": True}),
                "tests/planner/bypass.yaml",
            )


class BaseComparisonTest(unittest.TestCase):
    def test_time_limit_cannot_be_raised(self) -> None:
        old = suite({"name": "regression", "max_time": 1, "expect": {"rows": 1}})
        new = suite({"name": "regression", "max_time": 2, "expect": {"rows": 1}})
        with self.assertRaisesRegex(GuardFailure, "fix the regression"):
            compare_suites(old, new, "tests/planner/regression.yaml")

    def test_removing_specific_plan_limit_is_a_relaxation(self) -> None:
        old = suite(
            {"name": "regression", "max_plan_size": 30_000, "expect": {"rows": 1}}
        )
        new = suite({"name": "regression", "expect": {"rows": 1}})
        with self.assertRaisesRegex(GuardFailure, "max_plan_size was relaxed"):
            compare_suites(old, new, "tests/planner/regression.yaml")

    def test_existing_case_cannot_become_noncritical(self) -> None:
        old = suite()
        new = suite({"name": "regression", "noncritical": True, "expect": {"rows": 1}})
        with self.assertRaisesRegex(GuardFailure, "critical to noncritical"):
            compare_suites(old, new, "tests/planner/regression.yaml")

    def test_threshold_cannot_exempt_existing_correctness_test(self) -> None:
        old = suite()
        new = suite(
            {"name": "regression", "threshold_ms": 30_000, "expect": {"rows": 1}}
        )
        with self.assertRaisesRegex(
            GuardFailure, "exempt from hard time and plan-size"
        ):
            compare_suites(old, new, "tests/planner/regression.yaml")

    def test_compile_metric_cannot_be_removed(self) -> None:
        old = suite(
            {
                "name": "regression",
                "max_compile_metrics": {"logical_nodes": 1000},
                "expect": {"rows": 1},
            }
        )
        new = suite({"name": "regression", "expect": {"rows": 1}})
        with self.assertRaisesRegex(GuardFailure, "logical_nodes was relaxed"):
            compare_suites(old, new, "tests/planner/regression.yaml")

    def test_expectation_dimension_cannot_be_removed(self) -> None:
        old = suite(
            {"name": "regression", "expect": {"rows": 1, "data": [{"value": 1}]}}
        )
        new = suite({"name": "regression", "expect": {"rows": 1}})
        with self.assertRaisesRegex(GuardFailure, "removed expect.data"):
            compare_suites(old, new, "tests/planner/regression.yaml")

    def test_must_fail_can_become_a_correctness_test(self) -> None:
        old = suite({"name": "regression", "expect": {"error": True}})
        new = suite(
            {"name": "regression", "expect": {"rows": 1, "data": [{"value": 1}]}}
        )
        compare_suites(old, new, "tests/planner/regression.yaml")

    def test_tighter_budget_and_updated_expected_value_are_allowed(self) -> None:
        old = suite(
            {
                "name": "regression",
                "max_time": 2,
                "expect": {"rows": 1, "data": [{"value": 1}]},
            }
        )
        new = suite(
            {
                "name": "regression",
                "max_time": 1,
                "expect": {"rows": 1, "data": [{"value": 2}]},
            }
        )
        compare_suites(old, new, "tests/planner/regression.yaml")


class PlannerSourceDiffTest(unittest.TestCase):
    def test_retired_point_operator_cannot_return(self) -> None:
        with self.assertRaisesRegex(GuardFailure, "retired scan_order_point"):
            check_added_planner_lines(
                [(42, "(list (quote scan_order_point) table key)")]
            )

    def test_planner_cannot_emit_mutating_operator(self) -> None:
        with self.assertRaisesRegex(GuardFailure, "emit the functional API"):
            check_added_planner_lines(
                [(42, "(list (quote set_assoc_mut) acc key value)")]
            )

    def test_functional_operator_is_allowed(self) -> None:
        check_added_planner_lines([(42, "(list (quote set_assoc) acc key value)")])


if __name__ == "__main__":
    unittest.main()
