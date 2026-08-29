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

"""Prevent pull requests from weakening the independent performance gate."""

from __future__ import annotations

import argparse
import math
import sys
from collections import Counter
from pathlib import Path
from typing import Any

import yaml


PROTECTED_POLICY_PATHS = (
    ".github/workflows/performance.yml",
    ".github/workflows/performance-regression-policy.yml",
    "tools/perf_ab.py",
    "tools/check_performance_policy.py",
    "tools/test_perf_ab.py",
    "tools/test_performance_policy.py",
)
PROTECTED_CASE_FIELDS = (
    "sql",
    "sparql",
    "scm",
    "setup",
    "cleanup",
    "steps",
    "expect",
    "performance_rows",
)


class PolicyFailure(RuntimeError):
    """The candidate weakens a protected performance contract."""


def load_yaml(path: Path) -> dict[str, Any]:
    with path.open("r", encoding="utf-8") as handle:
        document = yaml.safe_load(handle) or {}
    if not isinstance(document, dict):
        raise PolicyFailure(f"{path}: suite must be a mapping")
    if not isinstance(document.get("metadata"), dict):
        raise PolicyFailure(f"{path}: metadata must be a mapping")
    if not isinstance(document.get("test_cases"), list):
        raise PolicyFailure(f"{path}: test_cases must be a list")
    return document


def numbered_cases(document: dict[str, Any]) -> dict[tuple[str, int], dict[str, Any]]:
    counts: Counter[str] = Counter()
    result: dict[tuple[str, int], dict[str, Any]] = {}
    for index, case in enumerate(document["test_cases"]):
        if not isinstance(case, dict):
            raise PolicyFailure(f"test case #{index + 1} must be a mapping")
        name = str(case.get("name", f"case #{index + 1}"))
        counts[name] += 1
        result[(name, counts[name])] = case
    return result


def finite_number(value: Any, label: str) -> float:
    if isinstance(value, bool) or not isinstance(value, (int, float)):
        raise PolicyFailure(f"{label} must be numeric")
    result = float(value)
    if not math.isfinite(result):
        raise PolicyFailure(f"{label} must be finite")
    return result


def regression_limit(case: dict[str, Any], metadata: dict[str, Any], label: str) -> float:
    value = case.get("max_regression_pct", metadata.get("max_regression_pct", 20))
    result = finite_number(value, f"{label}.max_regression_pct")
    if not 10 <= result <= 30:
        raise PolicyFailure(f"{label}.max_regression_pct must be from 10 through 30")
    return result


def validate_execution_counts(case: dict[str, Any], label: str) -> None:
    if "repetitions" in case and "timing_samples" in case:
        raise PolicyFailure(
            f"{label}: use either repetitions or timing_samples, not both"
        )
    repetitions = case.get("repetitions", case.get("timing_samples", 5))
    if isinstance(repetitions, bool) or not isinstance(repetitions, int) or repetitions < 1:
        raise PolicyFailure(f"{label}.repetitions must be a positive integer")
    warmup = case.get("warmup", 2)
    if isinstance(warmup, bool):
        return
    if not isinstance(warmup, int) or warmup < 0:
        raise PolicyFailure(f"{label}.warmup must be a non-negative integer or boolean")


def compare_suite(base: dict[str, Any], candidate: dict[str, Any], relative: str) -> None:
    base_metadata = base["metadata"]
    candidate_metadata = candidate["metadata"]
    if base_metadata.get("ci", True) is not False and candidate_metadata.get("ci") is False:
        raise PolicyFailure(f"{relative}: candidate disabled the CI suite")
    for field in ("setup", "cleanup"):
        if base.get(field) != candidate.get(field):
            raise PolicyFailure(f"{relative}: suite {field} is protected benchmark input")

    base_cases = numbered_cases(base)
    candidate_cases = numbered_cases(candidate)
    for identity, old in base_cases.items():
        name, occurrence = identity
        label = f"{relative}:{name}#{occurrence}"
        if identity not in candidate_cases:
            raise PolicyFailure(f"{label}: existing performance case was deleted or renamed")
        new = candidate_cases[identity]
        if old.get("disabled") is not True and new.get("disabled") is True:
            raise PolicyFailure(f"{label}: candidate disabled the case")
        for field in PROTECTED_CASE_FIELDS:
            if old.get(field) != new.get(field):
                raise PolicyFailure(f"{label}.{field} is protected benchmark input")
        if "threshold_ms" in old and "threshold_ms" not in new:
            raise PolicyFailure(f"{label}: candidate removed the performance measurement")
        if "threshold_ms" not in old:
            continue
        validate_execution_counts(old, label)
        validate_execution_counts(new, label)
        old_limit = regression_limit(old, base_metadata, label)
        new_limit = regression_limit(new, candidate_metadata, label)
        if new_limit > old_limit:
            raise PolicyFailure(
                f"{label}: max regression was relaxed from {old_limit:g}% to {new_limit:g}%"
            )


def check_policy(base_root: Path, candidate_root: Path) -> None:
    for relative in PROTECTED_POLICY_PATHS:
        base_path = base_root / relative
        candidate_path = candidate_root / relative
        if candidate_path.is_symlink() or not candidate_path.is_file():
            raise PolicyFailure(f"{relative}: protected policy file is missing or not regular")
        if base_path.read_bytes() != candidate_path.read_bytes():
            raise PolicyFailure(f"{relative}: protected performance policy was modified")

    performance_root = base_root / "tests" / "performance"
    for base_path in sorted(performance_root.rglob("*.yaml")):
        relative = base_path.relative_to(base_root).as_posix()
        candidate_path = candidate_root / relative
        if candidate_path.is_symlink() or not candidate_path.is_file():
            raise PolicyFailure(f"{relative}: existing performance suite was deleted")
        compare_suite(load_yaml(base_path), load_yaml(candidate_path), relative)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-root", type=Path, required=True)
    parser.add_argument("--candidate-root", type=Path, required=True)
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    try:
        check_policy(args.base_root.resolve(), args.candidate_root.resolve())
    except (OSError, PolicyFailure, yaml.YAMLError) as exc:
        print(f"Performance policy failed: {exc}", file=sys.stderr)
        return 1
    print("Performance policy passed: protected tests and A/B gate were not weakened")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
