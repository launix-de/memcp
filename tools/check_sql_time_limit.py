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
"""Guard the SQL test runner's hard per-query time limit."""

from __future__ import annotations

import ast
import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
RUNNER = ROOT / "run_sql_tests.py"
TESTS = ROOT / "tests"
EXPECTED_DEFAULT_MAX_TIME_SEC = 5.0


def fail(message: str) -> None:
    print(f"SQL time-limit guard failed: {message}", file=sys.stderr)
    sys.exit(1)


def numeric_constant(node: ast.AST) -> float | None:
    if isinstance(node, ast.Constant) and isinstance(node.value, (int, float)):
        return float(node.value)
    return None


def check_runner() -> None:
    source = RUNNER.read_text(encoding="utf-8")
    if "MEMCP_MAX_TIME" in source:
        fail("run_sql_tests.py must not allow MEMCP_MAX_TIME to override the default limit")

    tree = ast.parse(source, filename=str(RUNNER))
    values: list[ast.AST] = []
    for node in ast.walk(tree):
        if not isinstance(node, ast.Assign):
            continue
        for target in node.targets:
            if isinstance(target, ast.Name) and target.id == "DEFAULT_MAX_TIME_SEC":
                values.append(node.value)

    if len(values) != 1:
        fail("run_sql_tests.py must define DEFAULT_MAX_TIME_SEC exactly once")

    value = numeric_constant(values[0])
    if value != EXPECTED_DEFAULT_MAX_TIME_SEC:
        fail(f"DEFAULT_MAX_TIME_SEC must be the literal {EXPECTED_DEFAULT_MAX_TIME_SEC}")


def check_yaml_overrides() -> None:
    pattern = re.compile(r"^\s*max_time\s*:\s*([0-9]+(?:\.[0-9]+)?)\s*(?:#.*)?$")
    for path in sorted(TESTS.rglob("*.yaml")):
        for lineno, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
            match = pattern.match(line)
            if match and float(match.group(1)) > EXPECTED_DEFAULT_MAX_TIME_SEC:
                fail(f"{path.relative_to(ROOT)}:{lineno} raises max_time above {EXPECTED_DEFAULT_MAX_TIME_SEC}")


def main() -> None:
    check_runner()
    check_yaml_overrides()
    print(f"SQL time-limit guard passed: DEFAULT_MAX_TIME_SEC is locked to {EXPECTED_DEFAULT_MAX_TIME_SEC}")


if __name__ == "__main__":
    main()
