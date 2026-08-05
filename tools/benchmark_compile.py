#!/usr/bin/env python3
#
# Copyright (C) 2026  Carl-Philip Haensch
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

"""Measure EXPLAIN COMPILE phase distributions against a running MemCP."""

from __future__ import annotations

import argparse
import base64
import json
import math
import time
import urllib.error
import urllib.request
from pathlib import Path


def percentile(values: list[int], fraction: float) -> int:
    ordered = sorted(values)
    return ordered[max(0, math.ceil(len(ordered) * fraction) - 1)]


def request_metrics(args: argparse.Namespace, query: str) -> dict[str, int]:
    endpoint = "psql" if args.syntax == "postgresql" else "sql"
    url = f"{args.url.rstrip('/')}/{endpoint}/{args.database}"
    credentials = base64.b64encode(f"{args.username}:{args.password}".encode()).decode()
    request = urllib.request.Request(
        url,
        data=("EXPLAIN COMPILE " + query).encode(),
        headers={"Authorization": f"Basic {credentials}"},
        method="POST",
    )
    started = time.perf_counter_ns()
    try:
        with urllib.request.urlopen(request, timeout=args.timeout) as response:
            rows = [json.loads(line) for line in response if line.strip()]
    except urllib.error.HTTPError as error:
        detail = error.read().decode(errors="replace")
        raise RuntimeError(f"EXPLAIN COMPILE failed with HTTP {error.code}: {detail}") from error
    wall_ns = time.perf_counter_ns() - started
    if len(rows) != 1:
        raise RuntimeError(f"expected one metrics row, got {len(rows)}")
    metrics = {key: value for key, value in rows[0].items() if isinstance(value, int)}
    metrics["wall_ns"] = wall_ns
    return metrics


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("query_file", type=Path)
    parser.add_argument("--url", default="http://127.0.0.1:4321")
    parser.add_argument("--database", default="memcp-tests")
    parser.add_argument("--username", default="root")
    parser.add_argument("--password", default="admin")
    parser.add_argument("--syntax", choices=("mysql", "postgresql"), default="mysql")
    parser.add_argument("--warmup", type=int, default=3)
    parser.add_argument("--samples", type=int, default=30)
    parser.add_argument("--timeout", type=float, default=30.0)
    parser.add_argument(
        "--cache-bust",
        action="store_true",
        help="append a unique comment for comparison with servers that cache EXPLAIN COMPILE",
    )
    args = parser.parse_args()
    if args.samples < 1 or args.warmup < 0:
        parser.error("--samples must be positive and --warmup must not be negative")

    query = args.query_file.read_text(encoding="utf-8").strip().rstrip(";")
    if query.upper().startswith("EXPLAIN COMPILE "):
        query = query[len("EXPLAIN COMPILE ") :]

    samples: list[dict[str, int]] = []
    run_id = time.time_ns()
    for index in range(args.warmup + args.samples):
        sample_query = query
        if args.cache_bust:
            sample_query += f" /* compile-sample:{run_id}:{index} */"
        metrics = request_metrics(args, sample_query)
        if index >= args.warmup:
            samples.append(metrics)

    metric_names = sorted(set.intersection(*(set(sample) for sample in samples)))
    summary = {}
    for name in metric_names:
        values = [sample[name] for sample in samples]
        summary[name] = {
            "p50": percentile(values, 0.50),
            "p95": percentile(values, 0.95),
            "max": max(values),
        }
    print(json.dumps({"samples": samples, "summary": summary}, indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
