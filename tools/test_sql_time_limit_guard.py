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
import json
import os
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from check_sql_time_limit import GuardFailure
from check_sql_time_limit import check_base_regressions
from check_sql_time_limit import read_head_suites
from check_sql_time_limit import check_added_planner_lines
from check_sql_time_limit import check_runner
from check_sql_time_limit import compare_suites
from check_sql_time_limit import load_suite
from check_sql_time_limit import implementation_gap_context, implementation_gap_case_contract, policy_digest, load_implementation_gap_migrations
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

    def test_unlisted_must_fail_cannot_become_a_correctness_test(self) -> None:
        old = suite({"name": "regression", "expect": {"error": True}})
        new = suite(
            {"name": "regression", "expect": {"rows": 1, "data": [{"value": 1}]}}
        )
        with self.assertRaisesRegex(GuardFailure, "reviewed implementation-gap"):
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


class NonYamlFixtureDiffTest(unittest.TestCase):
    def check_diff(self, before: dict[str, str], after: dict[str, str]) -> None:
        # Git hooks export repository paths. Temporary fixture repositories
        # must not inherit those paths and mutate the calling worktree.
        environment = {key: value for key, value in os.environ.items()
                       if not key.startswith("GIT_")}
        with patch.dict(os.environ, environment, clear=True):
            self.check_diff_isolated(before, after)

    def check_diff_isolated(self, before: dict[str, str], after: dict[str, str]) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)

            def git(*args: str) -> str:
                return subprocess.check_output(
                    ["git", "-c", "user.name=Guard test", "-c",
                     "user.email=guard@example.invalid", "-c",
                     "core.hooksPath=/dev/null", *args],
                    cwd=root, text=True, stderr=subprocess.DEVNULL,
                ).strip()

            def write(files: dict[str, str]) -> None:
                for name, content in files.items():
                    path = root / name
                    path.parent.mkdir(parents=True, exist_ok=True)
                    path.write_text(content, encoding="utf-8")

            git("init", "-q")
            write(before)
            git("add", ".")
            git("commit", "-qm", "baseline")
            base = git("rev-parse", "HEAD")
            for name in before:
                (root / name).unlink()
            write(after)
            git("add", "-A")
            git("commit", "-qm", "candidate")
            check_base_regressions(root, base, read_head_suites(root))

    def test_non_yaml_fixture_changes_do_not_parse_as_sql_suites(self) -> None:
        source = "<?php\n// SPDX-License-Identifier: GPL-3.0-or-later\necho 'fixture';\n"
        before = {"tests/php/integration.php": source}
        for after in [
            {"tests/php/integration.php": source + "echo 'updated';\n"},
            {},
            {"tests/php/renamed.php": source},
        ]:
            with self.subTest(paths=list(after)):
                self.check_diff(before, after)

    def test_yaml_suite_cannot_be_hidden_by_a_non_yaml_extension(self) -> None:
        source = json.dumps(suite())
        with self.assertRaisesRegex(GuardFailure, "must not disappear"):
            self.check_diff(
                {"tests/regression.yaml": source},
                {"tests/regression.php": source},
            )

    def test_yaml_limits_remain_protected_alongside_php_changes(self) -> None:
        old = suite({"name": "regression", "sql": "SELECT 1", "max_time": 1,
                     "expect": {"rows": 1}})
        new = suite({"name": "regression", "sql": "SELECT 1", "max_time": 2,
                     "expect": {"rows": 1}})
        with self.assertRaisesRegex(GuardFailure, "max_time"):
            self.check_diff(
                {"tests/regression.yaml": json.dumps(old), "tests/fixture.php": "<?php"},
                {"tests/regression.yaml": json.dumps(new), "tests/fixture.php": "<?php echo 1;"},
            )


class ImplementationGapMigrationTest(unittest.TestCase):
    path = "tests/planner/regression.yaml"

    def fixtures(self):
        old = suite({"name": "regression", "sql": "SELECT id FROM source",
                     "expect": {"error": True, "contains": "not implemented"}})
        new = suite({"name": "regression", "sql": "SELECT id FROM source", "noncritical": True,
                     "expect": {"rows": 1, "data": [{"id": 1}]}})
        record = {"old_sql_sha256": policy_digest(old["test_cases"][0]["sql"]),
                  "new_sql_sha256": policy_digest(new["test_cases"][0]["sql"]),
                  "old_expect": old["test_cases"][0]["expect"],
                  "new_expect": new["test_cases"][0]["expect"], "syntax": "mysql",
                  "old_case_sha256": policy_digest(implementation_gap_case_contract(old["test_cases"][0])),
                  "new_case_sha256": policy_digest(implementation_gap_case_contract(new["test_cases"][0])),
                  "old_context_sha256": implementation_gap_context(old, 0),
                  "new_context_sha256": implementation_gap_context(new, 0)}
        return old, new, {(self.path, "regression", 1): record}

    def compare(self, old, new, records, path=None):
        with patch("check_sql_time_limit.load_implementation_gap_migrations", return_value=records):
            compare_suites(old, new, path or self.path)

    def test_exact_reviewed_gap_allows_positive_noncritical(self):
        old, new, records = self.fixtures()
        self.compare(old, new, records)

    def test_implementation_can_become_critical_without_weakening_its_oracle(self):
        old, new, records = self.fixtures()
        critical = json.loads(json.dumps(new))
        critical["test_cases"][0].pop("noncritical")
        self.compare(old, critical, records)
        self.compare(new, critical, records)

    def test_case_identity_query_and_dialect_are_exact(self):
        for change in ("name", "sql", "syntax", "path", "occurrence"):
            old, new, records = self.fixtures()
            if change == "path":
                target = "tests/planner/other.yaml"
            else:
                target = self.path
                if change == "occurrence":
                    records = {(self.path, "regression", 2): next(iter(records.values()))}
                else:
                    new["test_cases"][0][change] = {"name": "renamed", "sql": "SELECT 1", "syntax": "tsql"}[change]
            with self.subTest(change=change), self.assertRaises(GuardFailure):
                self.compare(old, new, records, target)

    def test_oracle_cannot_lose_values_rows_types_or_exact_cardinality(self):
        invalid = [{"rows": 1}, {"data": [{"id": 1}]}, {"rows": 0, "data": []},
                   {"rows": True, "data": [{"id": 1}]}, {"rows": 1.0, "data": [{"id": 1}]},
                   {"rows": 1, "data": [{}]}, {"rows": 1, "data": [{"id": True}]},
                   {"rows": 1, "data": [{"id": 2}]}, {"rows": 2, "data": [{"id": 1}]},
                   {"error": True, "rows": 1, "data": [{"id": 1}]}]
        for oracle in invalid:
            old, new, records = self.fixtures()
            new["test_cases"][0]["expect"] = oracle
            with self.subTest(oracle=oracle), self.assertRaises(GuardFailure):
                self.compare(old, new, records)

    def test_regular_errors_and_not_contains_remain_protected(self):
        for extra in ({"contains": "access denied"}, {"not_contains": "panic"}):
            old, new, records = self.fixtures()
            old["test_cases"][0]["expect"].update(extra)
            # The registry owns the original reviewed data, not the edited test.
            records = json.loads(json.dumps(list(records.values())))
            records[0]["old_expect"] = {"error": True, "contains": "not implemented"}
            registry = {(self.path, "regression", 1): records[0]}
            with self.subTest(extra=extra), self.assertRaises(GuardFailure):
                self.compare(old, new, registry)
        old, new, records = self.fixtures()
        with self.assertRaises(GuardFailure):
            self.compare(old, new, {})

    def test_positive_oracles_cannot_be_hidden_by_switching_back_to_error(self):
        _, positive, records = self.fixtures()
        invalid = json.loads(json.dumps(positive))
        invalid["test_cases"][0]["expect"]["error"] = True
        with self.assertRaisesRegex(GuardFailure, "bypass positive"):
            self.compare(positive, invalid, records)

    def test_budgets_disable_and_ci_controls_cannot_be_relaxed(self):
        for option in ({"max_time": 2}, {"max_plan_size": 200_000},
                       {"max_planner_time_ms": 2}, {"threshold_ms": 1}, {"disabled": True}):
            old, new, records = self.fixtures()
            old["test_cases"][0].update(max_time=1, max_plan_size=100_000, max_planner_time_ms=1)
            new["test_cases"][0].update(max_time=1, max_plan_size=100_000, max_planner_time_ms=1)
            new["test_cases"][0].update(option)
            with self.subTest(option=option), self.assertRaises(GuardFailure):
                self.compare(old, new, records)
        old, new, records = self.fixtures()
        new["metadata"]["ci"] = False
        with self.assertRaisesRegex(GuardFailure, "metadata.ci"):
            self.compare(old, new, records)

    def test_fixture_prefix_and_transport_cannot_be_substituted(self):
        for change in ("setup", "prefix", "scm", "steps", "tds_probe", "session_id", "metadata", "prefix_session"):
            old, new, records = self.fixtures()
            if change == "setup":
                new["setup"] = [{"sql": "DELETE FROM source"}]
            elif change == "prefix":
                new["test_cases"].insert(0, {"name": "changed fixture", "sql": "DELETE FROM source", "expect": {}})
            elif change == "metadata":
                new["metadata"]["database"] = "different-context"
            elif change == "prefix_session":
                earlier = {"name": "earlier", "sql": "SELECT 1", "expect": {}}
                old["test_cases"].insert(0, earlier)
                new["test_cases"].insert(0, {**earlier, "session_id": "other-owner"})
                record = next(iter(records.values()))
                record["old_context_sha256"] = implementation_gap_context(old, 1)
                record["new_context_sha256"] = implementation_gap_context(old, 1)
            else:
                new["test_cases"][0][change] = "different execution path"
            with self.subTest(change=change), self.assertRaises(GuardFailure):
                self.compare(old, new, records)

    def test_trusted_registry_has_only_well_formed_unique_exact_oracles(self):
        records = load_implementation_gap_migrations()
        self.assertGreater(len(records), 0)
        for record in records.values():
            self.assertEqual(record["new_expect"]["rows"], len(record["new_expect"]["data"]))
            self.assertNotIn("not_contains", record["old_expect"])
            self.assertNotIn("affected_rows", record["new_expect"])
        valid = {"version": 1, "migrations": [next(iter(records.values()))]}
        invalid = [[], {"version": True, "migrations": []}, {"version": 2, "migrations": []}]
        duplicate = json.loads(json.dumps(valid))
        duplicate["migrations"] *= 2
        invalid.append(duplicate)
        for oracle in ({"rows": True, "data": [{"id": 1}]}, {"rows": 1, "data": [{}]},
                       {"rows": 1, "data": []}, {"rows": 1, "data": [{"id": 1}], "affected_rows": 0}):
            document = json.loads(json.dumps(valid))
            document["migrations"][0]["new_expect"] = oracle
            invalid.append(document)
        try:
            for document in invalid:
                load_implementation_gap_migrations.cache_clear()
                with self.subTest(document=document), patch("check_sql_time_limit.Path.read_text", return_value=json.dumps(document)):
                    with self.assertRaises(GuardFailure):
                        load_implementation_gap_migrations()
        finally:
            load_implementation_gap_migrations.cache_clear()


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
