# Copyright (C) 2026 Carl-Philip Haensch
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program. If not, see <https://www.gnu.org/licenses/>.

"""Reject mutable SQL fixture table names shared by multiple CI suites."""

from collections import defaultdict
from pathlib import Path
import re
import sys

import yaml


TABLE_MUTATION = re.compile(
    r"""(?ix)
    \b(?:CREATE|DROP|ALTER|TRUNCATE)\s+(?:TEMPORARY\s+)?TABLE\s+
    (?:IF\s+(?:NOT\s+)?EXISTS\s+)?
    (?:(?:`[^`]+`|[A-Za-z_][\w-]*)\s*\.\s*)?
    (?:`([^`]+)`|([A-Za-z_][\w$]*))
    """
)


def sql_fragments(value):
    if isinstance(value, dict):
        for key, child in value.items():
            if key in {"sql", "psql"} and isinstance(child, str):
                yield child
            else:
                yield from sql_fragments(child)
    elif isinstance(value, list):
        for child in value:
            yield from sql_fragments(child)


def main() -> int:
    owners = defaultdict(set)
    for path in sorted(Path("tests").rglob("*.yaml")):
        with path.open("r", encoding="utf-8") as handle:
            spec = yaml.safe_load(handle) or {}
        metadata = spec.get("metadata", {}) if isinstance(spec, dict) else {}
        if metadata.get("ci", True) is False or metadata.get("disabled"):
            continue
        for sql in sql_fragments(spec):
            for match in TABLE_MUTATION.finditer(sql):
                owners[(match.group(1) or match.group(2)).lower()].add(str(path))

    collisions = {name: paths for name, paths in owners.items() if len(paths) > 1}
    if not collisions:
        return 0

    print("Duplicate mutable test table names are unsafe in parallel CI:", file=sys.stderr)
    for name, paths in sorted(collisions.items()):
        print(f"  {name}:", file=sys.stderr)
        for path in sorted(paths):
            print(f"    {path}", file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
