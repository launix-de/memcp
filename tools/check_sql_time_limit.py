#!/usr/bin/env python3
#
# Copyright (C) 2023-2026  Carl-Philip Hänsch
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
"""Guard the SQL test runner's hard per-query time limit.

The default hard limit documents compiler-performance regressions. It must stay
at one second and must not be disabled through environment variables or broad
YAML overrides.
"""

from __future__ import annotations

import ast
import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
RUNNER = ROOT / "run_sql_tests.py"
TESTS = ROOT / "tests"


def fail(message: str) -> None:
    print(f"SQL time-limit guard failed: {message}", file=sys.stderr)
    sys.exit(1)


def check_runner() -> None:
    source = RUNNER.read_text(encoding="utf-8")
    if "MEMCP_MAX_TIME" in source:
        fail("run_sql_tests.py must not allow MEMCP_MAX_TIME to override the default limit")

    tree = ast.parse(source, filename=str(RUNNER))
    values = []
    for node in ast.walk(tree):
        if not isinstance(node, ast.Assign):
            continue
        for target in node.targets:
            if isinstance(target, ast.Name) and target.id == "DEFAULT_MAX_TIME_SEC":
                values.append(node.value)

    if len(values) != 1:
        fail("run_sql_tests.py must define DEFAULT_MAX_TIME_SEC exactly once")

    value = values[0]
    if not isinstance(value, ast.Constant) or value.value != 1.0:
        fail("DEFAULT_MAX_TIME_SEC must be the literal 1.0")


def check_yaml_overrides() -> None:
    pattern = re.compile(r"^\s*max_time\s*:\s*([0-9]+(?:\.[0-9]+)?)\s*(?:#.*)?$")
    for path in sorted(TESTS.glob("*.yaml")):
        for lineno, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
            match = pattern.match(line)
            if match and float(match.group(1)) > 1.0:
                fail(f"{path.relative_to(ROOT)}:{lineno} raises max_time above 1.0")


def main() -> None:
    check_runner()
    check_yaml_overrides()
    print("SQL time-limit guard passed: DEFAULT_MAX_TIME_SEC is locked to 1.0")


if __name__ == "__main__":
    main()
