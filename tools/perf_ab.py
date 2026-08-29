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

"""Measure a trusted performance corpus against a base and candidate tree."""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
from collections import Counter
from pathlib import Path
from typing import Any

import yaml


# The legacy matrix benchmark treats {rows} as a matrix dimension and therefore
# performs cubic work. Its old local auto-calibrator converges near 100-300;
# starting it with the linear-test default of 10,000 exhausts CI memory before
# a base measurement exists. Keep this trusted bootstrap workload in the
# harness until the suite declares performance_rows itself.
BOOTSTRAP_ROWS = {
    "tests/performance/baseline.yaml::Perf: MATRIX MULT": 100,
}


class PerformanceFailure(RuntimeError):
    """A missing, invalid, or regressed performance measurement."""


def discover_suites(base_tree: Path) -> list[Path]:
    suites: list[Path] = []
    for path in sorted((base_tree / "tests" / "performance").rglob("*.yaml")):
        with path.open("r", encoding="utf-8") as handle:
            document = yaml.safe_load(handle) or {}
        metadata = document.get("metadata", {})
        if not isinstance(metadata, dict):
            raise PerformanceFailure(f"{path}: metadata must be a mapping")
        if metadata.get("ci", True) is not False:
            suites.append(path)
    if not suites:
        raise PerformanceFailure("trusted base contains no CI performance suites")
    return suites


def measurement_map(payload: dict[str, Any], suite: str) -> dict[str, dict[str, Any]]:
    if payload.get("schema_version") != 1:
        raise PerformanceFailure(f"{suite}: unsupported or missing result schema")
    measurements = payload.get("measurements")
    if not isinstance(measurements, list):
        raise PerformanceFailure(f"{suite}: measurements must be a list")

    occurrences: dict[str, int] = {}
    result: dict[str, dict[str, Any]] = {}
    for measurement in measurements:
        if not isinstance(measurement, dict) or not isinstance(measurement.get("name"), str):
            raise PerformanceFailure(f"{suite}: malformed measurement")
        name = measurement["name"]
        occurrences[name] = occurrences.get(name, 0) + 1
        identity = f"{suite}::{name}#{occurrences[name]}"
        result[identity] = measurement
    return result


def performance_cases(suite: Path, suite_id: str | None = None) -> list[dict[str, Any]]:
    with suite.open("r", encoding="utf-8") as handle:
        document = yaml.safe_load(handle) or {}
    metadata = document.get("metadata", {})
    default_regression = metadata.get("max_regression_pct", 20)
    cases: list[dict[str, Any]] = []
    for case in document.get("test_cases", []):
        expanded = [(case, case.get("name", "?"))]
        if "repeat" in case:
            amount = int(case["repeat"])
            expanded = [
                (inner, f"{inner.get('name', '?')} (iter {iteration + 1}/{amount})")
                for iteration in range(amount)
                for inner in case.get("tests", [])
            ]
        for measured, name in expanded:
            if "threshold_ms" not in measured:
                continue
            if "repetitions" in measured and "timing_samples" in measured:
                raise PerformanceFailure(
                    f"{suite}:{name}: use either repetitions or timing_samples, not both"
                )
            regression = measured.get("max_regression_pct", default_regression)
            if (
                isinstance(regression, bool)
                or not isinstance(regression, (int, float))
                or not 10 <= float(regression) <= 30
            ):
                raise PerformanceFailure(
                    f"{suite}:{name}: max_regression_pct must be from 10 through 30"
                )
            repetitions = measured.get(
                "repetitions", measured.get("timing_samples", 5)
            )
            if isinstance(repetitions, bool) or not isinstance(repetitions, int) or repetitions < 1:
                raise PerformanceFailure(
                    f"{suite}:{name}: repetitions must be a positive integer"
                )
            warmup = measured.get("warmup", 2)
            if warmup is True:
                warmup = 2
            elif warmup is False:
                warmup = 0
            if not isinstance(warmup, int) or warmup < 0:
                raise PerformanceFailure(
                    f"{suite}:{name}: warmup must be a non-negative integer or boolean"
                )
            cases.append({
                "name": name,
                "max_regression_pct": float(regression),
                "repetitions": repetitions,
                "warmup": warmup,
            })
            identity = f"{suite_id}::{name}" if suite_id else None
            cases[-1]["rows"] = int(
                measured.get(
                    "performance_rows",
                    BOOTSTRAP_ROWS.get(identity, 10_000),
                )
            )
            if cases[-1]["rows"] < 1:
                raise PerformanceFailure(f"{suite}:{name}: performance_rows must be positive")
    return cases


def parse_runner_output(output: str, suite: Path, suite_id: str | None = None) -> dict[str, Any]:
    timing_pattern = re.compile(
        r"^QUERY_TIME median_ns=\d+ total_ns=(\d+) min_ns=\d+ max_ns=\d+ "
        r"n=(\d+) warmup=(\d+) name=(.*)$"
    )
    success_pattern = re.compile(
        r"^✅ .*? \([0-9.]+ms / [0-9.]+ms, ([0-9,]+) rows(?:,|\))"
    )
    observed_timings = [
        (int(match.group(1)), int(match.group(2)), int(match.group(3)), match.group(4))
        for line in output.splitlines()
        if (match := timing_pattern.match(line))
    ]
    rows = [
        int(match.group(1).replace(",", ""))
        for line in output.splitlines()
        if (match := success_pattern.match(line))
    ]
    expected = performance_cases(suite, suite_id)
    remaining = Counter(case["name"] for case in expected)
    timings: list[tuple[int, str]] = []
    for timing in observed_timings:
        name = timing[3]
        if remaining[name] > 0:
            timings.append(timing)
            remaining[name] -= 1
    if len(timings) != len(expected) or len(rows) != len(expected):
        raise PerformanceFailure(
            f"{suite.name}: expected {len(expected)} measurements, observed "
            f"{len(timings)} timings and {len(rows)} row counts"
        )
    measurements: list[dict[str, Any]] = []
    for expected_case, (total_ns, repetitions, warmup, actual_name), row_count in zip(expected, timings, rows):
        if actual_name != expected_case["name"]:
            raise PerformanceFailure(
                f"{suite.name}: expected measurement {expected_case['name']!r}, "
                f"observed {actual_name!r}"
            )
        measurements.append(
            {
                "name": actual_name,
                "total_ms": total_ns / 1_000_000,
                "repetitions": repetitions,
                "warmup": warmup,
                "rows": row_count,
                "max_regression_pct": expected_case["max_regression_pct"],
            }
        )
    return {"schema_version": 1, "measurements": measurements}


def write_workload_baseline(tree: Path, suites: list[Path], base_tree: Path) -> None:
    """Give both revisions the same trusted row counts without timing allowances."""
    baseline: dict[str, dict[str, int]] = {}
    for suite in suites:
        suite_id = suite.relative_to(base_tree).as_posix()
        for case in performance_cases(suite, suite_id):
            name = case["name"]
            if name in baseline:
                raise PerformanceFailure(
                    f"duplicate performance case name across suites: {name}"
                )
            baseline[name] = {"rows": case["rows"]}
    with (tree / ".perf_baseline.json").open("w", encoding="utf-8") as handle:
        json.dump(baseline, handle, indent=2, sort_keys=True)
        handle.write("\n")


def compare_measurements(
    base: dict[str, dict[str, Any]], candidate: dict[str, dict[str, Any]]
) -> tuple[list[dict[str, Any]], list[str]]:
    errors: list[str] = []
    if set(base) != set(candidate):
        for identity in sorted(set(base) - set(candidate)):
            errors.append(f"{identity}: candidate measurement is missing")
        for identity in sorted(set(candidate) - set(base)):
            errors.append(f"{identity}: unexpected candidate measurement")

    comparisons: list[dict[str, Any]] = []
    for identity in sorted(set(base) & set(candidate)):
        baseline = base[identity]
        current = candidate[identity]
        try:
            base_total_ms = float(baseline["total_ms"])
            candidate_total_ms = float(current["total_ms"])
            base_repetitions = int(baseline["repetitions"])
            candidate_repetitions = int(current["repetitions"])
            base_warmup = int(baseline["warmup"])
            candidate_warmup = int(current["warmup"])
            max_regression_pct = float(baseline["max_regression_pct"])
            base_rows = int(baseline["rows"])
            candidate_rows = int(current["rows"])
        except (KeyError, TypeError, ValueError) as exc:
            errors.append(f"{identity}: invalid numeric measurement: {exc}")
            continue
        if (base_total_ms <= 0 or candidate_total_ms <= 0
                or base_repetitions < 1 or candidate_repetitions < 1):
            errors.append(f"{identity}: timings must be greater than zero")
            continue
        if not 10 <= max_regression_pct <= 30:
            errors.append(f"{identity}: max_regression_pct must be from 10 through 30")
            continue
        if base_rows != candidate_rows:
            errors.append(
                f"{identity}: workload differs ({base_rows} base rows, "
                f"{candidate_rows} candidate rows)"
            )
            continue

        base_ms = base_total_ms / base_repetitions
        candidate_ms = candidate_total_ms / candidate_repetitions
        warmup_bonus_pct = (
            100.0 if base_warmup > 0 and candidate_warmup == 0 else 0.0
        )
        allowed_pct = max_regression_pct + warmup_bonus_pct
        ratio = candidate_ms / base_ms
        allowed_ratio = 1.0 + allowed_pct / 100.0
        regressed = ratio > allowed_ratio
        comparison = {
            "id": identity,
            "base_ms": base_ms,
            "candidate_ms": candidate_ms,
            "ratio": ratio,
            "max_regression_pct": max_regression_pct,
            "warmup_bonus_pct": warmup_bonus_pct,
            "base_repetitions": base_repetitions,
            "candidate_repetitions": candidate_repetitions,
            "base_warmup": base_warmup,
            "candidate_warmup": candidate_warmup,
            "regressed": regressed,
            "rows": base_rows,
        }
        comparisons.append(comparison)
        if regressed:
            errors.append(
                f"{identity}: {candidate_ms:.3f}ms is {(ratio - 1) * 100:.1f}% slower "
                f"than {base_ms:.3f}ms per repetition (allowed {allowed_pct:g}%)"
            )
    return comparisons, errors


def parse_corpus_output(
    output: str, suites: list[Path], base_tree: Path
) -> dict[str, dict[str, Any]]:
    blocks: dict[str, list[str]] = {}
    current: str | None = None
    for line in output.splitlines():
        if line.startswith("PERF_SUITE path="):
            current = line.removeprefix("PERF_SUITE path=")
            blocks[current] = []
        elif current is not None:
            blocks[current].append(line)

    measurements: dict[str, dict[str, Any]] = {}
    for suite in suites:
        relative = suite.relative_to(base_tree).as_posix()
        if relative not in blocks:
            raise PerformanceFailure(f"{relative}: runner emitted no suite output")
        payload = parse_runner_output("\n".join(blocks[relative]), suite, relative)
        measurements.update(measurement_map(payload, relative))
    return measurements


def run_corpus(
    tree: Path,
    runner: Path,
    suites: list[Path],
    base_tree: Path,
    samples: int,
    timeout: int,
) -> dict[str, dict[str, Any]]:
    environment = os.environ.copy()
    environment.update(
        {
            "PERF_TEST": "1",
            "PERF_NORECALIBRATE": "1",
            "PERF_REPEAT": str(samples),
        }
    )
    write_workload_baseline(tree, suites, base_tree)
    relative_suites = [suite.relative_to(base_tree).as_posix() for suite in suites]
    command = [
        sys.executable,
        str(runner),
        *relative_suites,
        "--jobs",
        os.environ.get("PERF_JOBS", "4"),
        "--log-times",
    ]
    completed = subprocess.run(
        command,
        cwd=tree,
        env=environment,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        timeout=timeout,
        check=False,
    )
    print(completed.stdout, end="" if completed.stdout.endswith("\n") else "\n")
    if completed.returncode != 0:
        raise PerformanceFailure(
            f"{tree.name}: performance corpus failed with exit code {completed.returncode}"
        )
    return parse_corpus_output(completed.stdout, suites, base_tree)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-tree", type=Path, required=True)
    parser.add_argument("--candidate-tree", type=Path, required=True)
    parser.add_argument("--runner", type=Path, required=True)
    parser.add_argument(
        "--suite",
        action="append",
        help="trusted suite path relative to the base tree; repeat to select several",
    )
    parser.add_argument("--samples", type=int, default=5)
    parser.add_argument("--suite-timeout", type=int, default=3600)
    parser.add_argument("--output", type=Path, required=True)
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    if args.samples < 1 or args.samples % 2 == 0:
        raise PerformanceFailure("--samples must be a positive odd integer")
    base_tree = args.base_tree.resolve()
    candidate_tree = args.candidate_tree.resolve()
    runner = args.runner.resolve()
    suites = discover_suites(base_tree)
    if args.suite:
        requested = set(args.suite)
        suites = [
            suite for suite in suites
            if suite.relative_to(base_tree).as_posix() in requested
        ]
        found = {suite.relative_to(base_tree).as_posix() for suite in suites}
        missing = requested - found
        if missing:
            raise PerformanceFailure(
                "unknown or non-CI performance suites: " + ", ".join(sorted(missing))
            )

    suites = [suite for suite in suites if performance_cases(
        suite, suite.relative_to(base_tree).as_posix()
    )]
    if not suites:
        raise PerformanceFailure("trusted performance corpus produced zero measurements")

    print("\n=== trusted base corpus ===", flush=True)
    all_base = run_corpus(
        base_tree, runner, suites, base_tree, args.samples, args.suite_timeout
    )
    print("\n=== merge candidate corpus ===", flush=True)
    all_candidate = run_corpus(
        candidate_tree, runner, suites, base_tree, args.samples, args.suite_timeout
    )

    if not all_base:
        raise PerformanceFailure("trusted performance corpus produced zero measurements")
    comparisons, errors = compare_measurements(all_base, all_candidate)
    output = {
        "schema_version": 1,
        "base_tree": str(base_tree),
        "candidate_tree": str(candidate_tree),
        "comparisons": comparisons,
        "errors": errors,
    }
    args.output.parent.mkdir(parents=True, exist_ok=True)
    with args.output.open("w", encoding="utf-8") as handle:
        json.dump(output, handle, indent=2, sort_keys=True)
        handle.write("\n")

    print("\nPerformance comparison:")
    for item in comparisons:
        marker = "FAIL" if item["regressed"] else "PASS"
        print(
            f"{marker} {item['id']}: {item['base_ms']:.3f}ms -> "
            f"{item['candidate_ms']:.3f}ms ({(item['ratio'] - 1) * 100:+.1f}%, "
            f"limit +{item['max_regression_pct'] + item['warmup_bonus_pct']:g}%, "
            f"{item['base_repetitions']} -> {item['candidate_repetitions']} repetitions)"
        )
    if errors:
        raise PerformanceFailure("\n".join(errors))
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (PerformanceFailure, subprocess.TimeoutExpired, json.JSONDecodeError) as exc:
        print(f"Performance A/B failed: {exc}", file=sys.stderr)
        raise SystemExit(1)
