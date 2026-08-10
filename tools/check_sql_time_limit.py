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
#
"""Protect SQL correctness and planner regression gates from being weakened."""

from __future__ import annotations

import argparse
import ast
import math
import re
import subprocess
import sys
from collections import Counter
from pathlib import Path
from typing import Any

import yaml


EXPECTED_DEFAULT_MAX_TIME_SEC = 5.0
EXPECTED_DEFAULT_MAX_PLAN_SIZE = 200_000
MAPPING_BUDGETS = ("max_compile_phase_ms", "max_compile_metrics")
PROTECTED_EXPECTATIONS = (
    "rows",
    "data",
    "contains",
    "not_contains",
    "result_contains",
    "result_not_contains",
    "error",
)


class GuardFailure(RuntimeError):
    pass


def fail(message: str) -> None:
    print(f"SQL regression guard failed: {message}", file=sys.stderr)
    sys.exit(1)


def numeric_constant(node: ast.AST) -> float | None:
    if (
        isinstance(node, ast.Constant)
        and isinstance(node.value, (int, float))
        and not isinstance(node.value, bool)
    ):
        return float(node.value)
    return None


def assigned_numeric_constant(tree: ast.AST, name: str) -> float:
    values: list[ast.AST] = []
    for node in ast.walk(tree):
        if not isinstance(node, ast.Assign):
            continue
        for target in node.targets:
            if isinstance(target, ast.Name) and target.id == name:
                values.append(node.value)
    if len(values) != 1:
        raise GuardFailure(f"run_sql_tests.py must define {name} exactly once")
    value = numeric_constant(values[0])
    if value is None:
        raise GuardFailure(f"{name} must be a numeric literal")
    return value


def check_runner(root: Path) -> None:
    runner = root / "run_sql_tests.py"
    source = runner.read_text(encoding="utf-8")
    for environment_override in ("MEMCP_MAX_TIME", "MEMCP_MAX_PLAN_SIZE"):
        if environment_override in source:
            raise GuardFailure(
                f"run_sql_tests.py must not allow {environment_override} to override a fixed regression limit"
            )
    tree = ast.parse(source, filename=str(runner))
    actual_time = assigned_numeric_constant(tree, "DEFAULT_MAX_TIME_SEC")
    if actual_time != EXPECTED_DEFAULT_MAX_TIME_SEC:
        raise GuardFailure(
            f"DEFAULT_MAX_TIME_SEC must remain the literal {EXPECTED_DEFAULT_MAX_TIME_SEC}; "
            "optimize slow queries before tightening it, and do not raise it"
        )
    actual_plan_size = assigned_numeric_constant(tree, "DEFAULT_MAX_PLAN_SIZE")
    if actual_plan_size != EXPECTED_DEFAULT_MAX_PLAN_SIZE:
        raise GuardFailure(
            f"DEFAULT_MAX_PLAN_SIZE must remain the literal {EXPECTED_DEFAULT_MAX_PLAN_SIZE}"
        )


def load_suite(text: str, label: str) -> dict[str, Any]:
    try:
        value = yaml.safe_load(text) or {}
    except yaml.YAMLError as exc:
        raise GuardFailure(f"{label} is not valid YAML: {exc}") from exc
    if not isinstance(value, dict):
        raise GuardFailure(f"{label} must contain a YAML mapping")
    metadata = value.get("metadata", {})
    if metadata is None:
        metadata = {}
    if not isinstance(metadata, dict):
        raise GuardFailure(f"{label}: metadata must be a mapping")
    test_cases = value.get("test_cases", [])
    if test_cases is None:
        test_cases = []
    if not isinstance(test_cases, list) or any(
        not isinstance(case, dict) for case in test_cases
    ):
        raise GuardFailure(f"{label}: test_cases must be a list of mappings")
    value["metadata"] = metadata
    value["test_cases"] = test_cases
    return value


def finite_number(value: Any, label: str) -> float:
    if isinstance(value, bool):
        raise GuardFailure(f"{label} must be numeric, not boolean")
    try:
        result = float(value)
    except (TypeError, ValueError) as exc:
        raise GuardFailure(f"{label} must be numeric") from exc
    if not math.isfinite(result):
        raise GuardFailure(f"{label} must be finite")
    return result


def validate_budget_mapping(value: Any, label: str) -> None:
    if not isinstance(value, dict):
        raise GuardFailure(f"{label} must be a mapping")
    for key, limit in value.items():
        if finite_number(limit, f"{label}.{key}") <= 0:
            raise GuardFailure(
                f"{label}.{key} must be greater than zero; zero disables the regression gate"
            )


def validate_budget_container(container: dict[str, Any], label: str) -> None:
    if "max_time" in container:
        limit = finite_number(container["max_time"], f"{label}.max_time")
        if limit <= 0 or limit > EXPECTED_DEFAULT_MAX_TIME_SEC:
            raise GuardFailure(
                f"{label}.max_time must be greater than zero and at most {EXPECTED_DEFAULT_MAX_TIME_SEC}"
            )
    for key in ("max_plan_size", "max_planner_time_ms", "threshold_ms"):
        if key in container and finite_number(container[key], f"{label}.{key}") <= 0:
            raise GuardFailure(
                f"{label}.{key} must be greater than zero; zero disables the regression gate"
            )
    for key in MAPPING_BUDGETS:
        if key in container:
            validate_budget_mapping(container[key], f"{label}.{key}")


def validate_suite(suite: dict[str, Any], path: str) -> None:
    metadata = suite["metadata"]
    validate_budget_container(metadata, f"{path}:metadata")
    if metadata.get("disabled") is True:
        raise GuardFailure(
            f"{path}: metadata.disabled=true would skip the complete suite"
        )
    if metadata.get("ci") is False and not path.startswith("tests/performance/"):
        raise GuardFailure(
            f"{path}: metadata.ci=false is reserved for manual performance benchmarks"
        )
    for index, case in enumerate(suite["test_cases"]):
        name = case.get("name", f"case #{index + 1}")
        label = f"{path}:{name}"
        validate_budget_container(case, label)
        if case.get("disabled") is True:
            raise GuardFailure(
                f"{label} uses disabled=true; remove the test or make its failure visible"
            )


def read_head_suites(root: Path) -> dict[str, dict[str, Any]]:
    result: dict[str, dict[str, Any]] = {}
    for path in sorted((root / "tests").rglob("*.yaml")):
        relative = path.relative_to(root).as_posix()
        suite = load_suite(path.read_text(encoding="utf-8"), relative)
        validate_suite(suite, relative)
        result[relative] = suite
    return result


def git(root: Path, *args: str, check: bool = True) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", *args],
        cwd=root,
        check=check,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )


def changed_test_paths(
    root: Path, base_ref: str
) -> list[tuple[str, str | None, str | None]]:
    output = git(
        root, "diff", "--name-status", "-M", f"{base_ref}...HEAD", "--", "tests"
    ).stdout
    changes: list[tuple[str, str | None, str | None]] = []
    for line in output.splitlines():
        parts = line.split("\t")
        status = parts[0]
        if status.startswith("R") and len(parts) == 3:
            changes.append(("R", parts[1], parts[2]))
        elif status == "A" and len(parts) == 2:
            changes.append(("A", None, parts[1]))
        elif status == "D" and len(parts) == 2:
            changes.append(("D", parts[1], None))
        elif len(parts) == 2:
            changes.append((status, parts[1], parts[1]))
    return changes


def load_base_suite(root: Path, base_ref: str, path: str) -> dict[str, Any]:
    result = git(root, "show", f"{base_ref}:{path}", check=False)
    if result.returncode != 0:
        raise GuardFailure(f"cannot read {path} from base commit {base_ref}")
    return load_suite(result.stdout, f"{base_ref}:{path}")


def numbered_cases(suite: dict[str, Any]) -> dict[tuple[Any, int], dict[str, Any]]:
    seen: Counter[Any] = Counter()
    result: dict[tuple[Any, int], dict[str, Any]] = {}
    for index, case in enumerate(suite["test_cases"]):
        name = case.get("name", f"case #{index + 1}")
        seen[name] += 1
        result[(name, seen[name])] = case
    return result


def effective_limit(case: dict[str, Any], metadata: dict[str, Any], key: str) -> float:
    if key in case:
        return finite_number(case[key], key)
    if key in metadata:
        return finite_number(metadata[key], key)
    if key == "max_time":
        return EXPECTED_DEFAULT_MAX_TIME_SEC
    if key == "max_plan_size":
        return float(EXPECTED_DEFAULT_MAX_PLAN_SIZE)
    return 0.0


def assert_not_relaxed(old: float, new: float, label: str) -> None:
    if old > 0 and (new <= 0 or new > old):
        new_text = "disabled" if new <= 0 else str(new)
        raise GuardFailure(
            f"{label} was relaxed from {old} to {new_text}; fix the regression instead of weakening its gate"
        )


def compare_mapping_budget(old: Any, new: Any, label: str) -> None:
    if not isinstance(old, dict):
        return
    new_mapping = new if isinstance(new, dict) else {}
    for key, old_value in old.items():
        old_limit = finite_number(old_value, f"{label}.{key}")
        new_limit = (
            finite_number(new_mapping[key], f"{label}.{key}")
            if key in new_mapping
            else 0.0
        )
        assert_not_relaxed(old_limit, new_limit, f"{label}.{key}")


def compare_expectations(
    old_case: dict[str, Any], new_case: dict[str, Any], label: str
) -> None:
    old_expect = old_case.get("expect")
    new_expect = new_case.get("expect")
    if not isinstance(old_expect, dict):
        return
    if not isinstance(new_expect, dict):
        raise GuardFailure(f"{label} removed its expectation mapping")
    for key in PROTECTED_EXPECTATIONS:
        if key in old_expect and key not in new_expect:
            if key == "error" and any(
                candidate in new_expect
                for candidate in PROTECTED_EXPECTATIONS
                if candidate != "error"
            ):
                continue
            raise GuardFailure(
                f"{label} removed expect.{key}; retain an equally strong correctness assertion"
            )


def compare_suites(old: dict[str, Any], new: dict[str, Any], label: str) -> None:
    old_metadata = old["metadata"]
    new_metadata = new["metadata"]
    if old_metadata.get("ci", True) is not False and new_metadata.get("ci") is False:
        raise GuardFailure(
            f"{label} changed metadata.ci to false and would disappear from CI"
        )
    if (
        old_metadata.get("disabled") is not True
        and new_metadata.get("disabled") is True
    ):
        raise GuardFailure(f"{label} disabled an existing suite")
    for key in ("max_time", "max_plan_size"):
        old_limit = effective_limit({}, old_metadata, key)
        new_limit = effective_limit({}, new_metadata, key)
        assert_not_relaxed(old_limit, new_limit, f"{label}:metadata.{key}")

    old_cases = numbered_cases(old)
    new_cases = numbered_cases(new)
    for identity, old_case in old_cases.items():
        name, occurrence = identity
        case_label = f"{label}:{name}" + (
            f" occurrence {occurrence}" if occurrence > 1 else ""
        )
        if identity not in new_cases:
            raise GuardFailure(
                f"{case_label} was deleted; preserve the regression or move it unchanged"
            )
        new_case = new_cases[identity]
        if (
            old_case.get("noncritical") is not True
            and new_case.get("noncritical") is True
        ):
            raise GuardFailure(f"{case_label} was changed from critical to noncritical")
        if old_case.get("disabled") is not True and new_case.get("disabled") is True:
            raise GuardFailure(f"{case_label} was disabled")
        for key in ("max_time", "max_plan_size", "max_planner_time_ms"):
            old_limit = effective_limit(old_case, old_metadata, key)
            new_limit = effective_limit(new_case, new_metadata, key)
            assert_not_relaxed(old_limit, new_limit, f"{case_label}.{key}")
        old_threshold = effective_limit(old_case, old_metadata, "threshold_ms")
        new_threshold = effective_limit(new_case, new_metadata, "threshold_ms")
        if old_threshold <= 0 < new_threshold:
            raise GuardFailure(
                f"{case_label} added threshold_ms and would become exempt from hard time and plan-size gates"
            )
        assert_not_relaxed(old_threshold, new_threshold, f"{case_label}.threshold_ms")
        for key in MAPPING_BUDGETS:
            compare_mapping_budget(
                old_case.get(key), new_case.get(key), f"{case_label}.{key}"
            )
        compare_expectations(old_case, new_case, case_label)


def added_planner_lines(root: Path, base_ref: str) -> list[tuple[int | None, str]]:
    output = git(
        root,
        "diff",
        "--unified=0",
        f"{base_ref}...HEAD",
        "--",
        "lib/queryplan.scm",
    ).stdout
    line_number: int | None = None
    result: list[tuple[int | None, str]] = []
    for line in output.splitlines():
        match = re.match(r"@@ -[^ ]+ \+(\d+)", line)
        if match:
            line_number = int(match.group(1))
            continue
        if line.startswith("+") and not line.startswith("+++"):
            result.append((line_number, line[1:]))
            if line_number is not None:
                line_number += 1
        elif line_number is not None and not line.startswith("-"):
            line_number += 1
    return result


def check_added_planner_lines(lines: list[tuple[int | None, str]]) -> None:
    mutating_emission = re.compile(
        r"(?:\(quote\s+|\(symbol\s+|['(])([A-Za-z0-9_-]+_mut)\b"
    )
    for line_number, line in lines:
        location = f"lib/queryplan.scm:{line_number or '?'}"
        if "scan_order_point" in line:
            raise GuardFailure(
                f"{location} reintroduces retired scan_order_point; optimize the existing scan_order implementation"
            )
        match = mutating_emission.search(line)
        if match:
            raise GuardFailure(
                f"{location} emits mutable operator {match.group(1)}; emit the functional API and let the optimizer choose mutation"
            )


def check_base_regressions(
    root: Path, base_ref: str, head_suites: dict[str, dict[str, Any]]
) -> None:
    git(root, "rev-parse", "--verify", f"{base_ref}^{{commit}}")
    for status, old_path, new_path in changed_test_paths(root, base_ref):
        if old_path is None:
            continue
        old_suite = load_base_suite(root, base_ref, old_path)
        if new_path is None or new_path not in head_suites:
            raise GuardFailure(
                f"{old_path} was deleted; CI regression suites must not disappear"
            )
        compare_suites(old_suite, head_suites[new_path], new_path)
    check_added_planner_lines(added_planner_lines(root, base_ref))


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--base-ref", help="compare regression gates with this PR base commit"
    )
    parser.add_argument(
        "--root", type=Path, default=Path(__file__).resolve().parents[1]
    )
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    root = args.root.resolve()
    try:
        check_runner(root)
        head_suites = read_head_suites(root)
        if args.base_ref:
            check_base_regressions(root, args.base_ref, head_suites)
    except (GuardFailure, subprocess.CalledProcessError) as exc:
        fail(str(exc))
    comparison = (
        f" and no gates were weakened from {args.base_ref}" if args.base_ref else ""
    )
    print(
        f"SQL regression guard passed: defaults are fixed, YAML gates are active{comparison}"
    )


if __name__ == "__main__":
    main()
