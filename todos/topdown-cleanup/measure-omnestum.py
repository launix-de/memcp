#!/usr/bin/env python3
"""
Measure real Omnestum document dataview queries against a running MemCP server.

This script is intentionally kept under todos/ because it references local
Omnestum query files and data directories. Do not commit real query output or
production data.
"""

import argparse
import base64
import glob
import json
import os
import statistics
import time
from pathlib import Path
from urllib.parse import quote

import requests


DEFAULT_QUERIES = [
    "/tmp/omnestum-dv-t_ID_t_jahr_canView_-trace.sql",
    "/tmp/omnestum-dv-t_ID_t_standort_canView_.sql",
    "/tmp/omnestum-dv-t_ID_t_uploaded_at_.sql",
    "/tmp/omnestum-dv-t_.sql",
    "/tmp/omnestum-dv-nosorter.sql",
    "/tmp/omnestum-dv-idonly.sql",
]

PLAN_MARKERS = [
    ".mat:",
    "__mat:",
    "legacy_materialized",
    "legacy-fallback",
    "legacy-fallback-non-aggregate",
    "inner_select",
    "__union_distinct_seen",
    "__union_distinct_rows",
    "union_distinct",
    "make_keytable",
]


def auth_header(user: str, password: str) -> dict:
    token = base64.b64encode(f"{user}:{password}".encode()).decode()
    return {"Authorization": "Basic " + token, "Content-Type": "text/plain; charset=utf-8"}


def sql_request(base_url: str, db: str, query: str, headers: dict, session_id: str, timeout: int) -> requests.Response:
    url = f"{base_url.rstrip('/')}/sql/{quote(db, safe='')}"
    req_headers = dict(headers)
    if session_id:
        req_headers["X-Session-Id"] = session_id
    return requests.post(url, data=query.encode("utf-8"), headers=req_headers, timeout=timeout)


def run_sql(base_url: str, db: str, query: str, headers: dict, session_id: str, timeout: int) -> tuple[float, requests.Response]:
    start = time.perf_counter()
    response = sql_request(base_url, db, query, headers, session_id, timeout)
    return time.perf_counter() - start, response


def ensure_session(base_url: str, db: str, headers: dict, session_id: str, user_id: int, timeout: int) -> None:
    setup = [
        f"SET @fop_user := {int(user_id)}",
        "SET @fop_time := UNIX_TIMESTAMP()",
    ]
    for statement in setup:
        response = sql_request(base_url, db, statement, headers, session_id, timeout)
        if response.status_code != 200:
            raise RuntimeError(f"session setup failed for {statement}: {response.status_code} {response.text[:500]}")


def query_name(path: str) -> str:
    return Path(path).stem.replace("omnestum-dv-", "")


def classify_query(path: str, sql: str) -> dict:
    lower = sql.lower()
    return {
        "file": path,
        "name": query_name(path),
        "bytes": len(sql.encode("utf-8")),
        "select_count": lower.count("select "),
        "exists_count": lower.count("exists "),
        "derived_count": lower.count(" from (select"),
        "left_join_count": lower.count("left join"),
        "session_var_count": lower.count("@fop_"),
        "limit_count": lower.count(" limit "),
        "order_by_count": lower.count("order by"),
    }


def marker_counts(text: str) -> dict:
    return {marker: text.count(marker) for marker in PLAN_MARKERS}


def write_text(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")


def measure_one(args: argparse.Namespace, headers: dict, query_file: str, out_dir: Path) -> dict:
    sql = Path(query_file).read_text(encoding="utf-8").strip()
    meta = classify_query(query_file, sql)
    session_id = f"{args.session_prefix}-{meta['name']}"

    ensure_session(args.api, args.database, headers, session_id, args.fop_user, args.timeout)

    explain_text = ""
    explain_ir_text = ""
    if args.explain:
        for prefix, target in [("EXPLAIN ", "explain"), ("EXPLAIN IR ", "explain-ir")]:
            ensure_session(args.api, args.database, headers, session_id, args.fop_user, args.timeout)
            elapsed, response = run_sql(args.api, args.database, prefix + sql, headers, session_id, args.explain_timeout)
            text = response.text
            write_text(out_dir / f"{meta['name']}.{target}.txt", text)
            if target == "explain":
                explain_text = text
                meta["explain_elapsed_s"] = elapsed
                meta["explain_status"] = response.status_code
            else:
                explain_ir_text = text
                meta["explain_ir_elapsed_s"] = elapsed
                meta["explain_ir_status"] = response.status_code

    if args.no_execute:
        samples = []
    else:
        for _ in range(args.warmups):
            ensure_session(args.api, args.database, headers, session_id, args.fop_user, args.timeout)
            run_sql(args.api, args.database, sql, headers, session_id, args.timeout)

        samples = []
        for _ in range(args.repeats):
            ensure_session(args.api, args.database, headers, session_id, args.fop_user, args.timeout)
            elapsed, response = run_sql(args.api, args.database, sql, headers, session_id, args.timeout)
            sample = {
                "elapsed_s": elapsed,
                "status": response.status_code,
                "response_bytes": len(response.content),
                "response_head": response.text[:300],
            }
            samples.append(sample)
            if response.status_code != 200:
                break

    elapsed_values = [sample["elapsed_s"] for sample in samples if sample["status"] == 200]
    result = {
        **meta,
        "samples": samples,
        "sample_count": len(samples),
        "ok_count": len(elapsed_values),
        "markers_explain": marker_counts(explain_text),
        "markers_explain_ir": marker_counts(explain_ir_text),
        "explain_size": len(explain_text),
        "explain_ir_size": len(explain_ir_text),
    }
    if elapsed_values:
        result.update(
            {
                "min_s": min(elapsed_values),
                "median_s": statistics.median(elapsed_values),
                "max_s": max(elapsed_values),
            }
        )
    return result


def expand_queries(patterns: list[str]) -> list[str]:
    files: list[str] = []
    for pattern in patterns:
        matches = sorted(glob.glob(pattern))
        files.extend(matches if matches else [pattern])
    return [path for path in files if Path(path).exists()]


def write_summary(out_dir: Path, results: list[dict]) -> None:
    lines = [
        "# Omnestum Dataview Measurement",
        "",
        "| query | median | min | max | explain IR bytes | derived | exists | inner_select | legacy fallback | materialized |",
        "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |",
    ]
    for result in results:
        markers = result.get("markers_explain_ir", {})
        lines.append(
            "| {name} | {median} | {minv} | {maxv} | {ir_size} | {derived} | {exists} | {inner} | {legacy} | {mat} |".format(
                name=result["name"],
                median=f"{result['median_s']:.3f}s" if "median_s" in result else "n/a",
                minv=f"{result['min_s']:.3f}s" if "min_s" in result else "n/a",
                maxv=f"{result['max_s']:.3f}s" if "max_s" in result else "n/a",
                ir_size=result.get("explain_ir_size", 0),
                derived=result.get("derived_count", 0),
                exists=result.get("exists_count", 0),
                inner=markers.get("inner_select", 0),
                legacy=markers.get("legacy-fallback", 0),
                mat=markers.get(".mat:", 0) + markers.get("__mat:", 0),
            )
        )
    write_text(out_dir / "summary.md", "\n".join(lines) + "\n")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--api", default="http://127.0.0.1:4526")
    parser.add_argument("--database", default="omnestum")
    parser.add_argument("--user", default="root")
    parser.add_argument("--password", default="admin")
    parser.add_argument("--fop-user", type=int, default=105)
    parser.add_argument("--session-prefix", default="omnestum-measure")
    parser.add_argument("--query", action="append", default=[])
    parser.add_argument("--out", default="")
    parser.add_argument("--warmups", type=int, default=1)
    parser.add_argument("--repeats", type=int, default=5)
    parser.add_argument("--timeout", type=int, default=180)
    parser.add_argument("--explain-timeout", type=int, default=180)
    parser.add_argument("--no-explain", dest="explain", action="store_false")
    parser.add_argument("--no-execute", action="store_true")
    parser.set_defaults(explain=True)
    args = parser.parse_args()

    queries = expand_queries(args.query or DEFAULT_QUERIES)
    if not queries:
        raise SystemExit("no query files found")

    out_dir = Path(args.out or f"/tmp/omnestum-measure-{time.strftime('%Y%m%d-%H%M%S')}")
    out_dir.mkdir(parents=True, exist_ok=True)
    headers = auth_header(args.user, args.password)

    results = []
    for query_file in queries:
        print(f"measuring {query_file}", flush=True)
        result = measure_one(args, headers, query_file, out_dir)
        results.append(result)
        if "median_s" in result:
            print(f"  {result['name']}: median={result['median_s']:.3f}s min={result['min_s']:.3f}s max={result['max_s']:.3f}s", flush=True)
        else:
            print(f"  {result['name']}: no successful execution sample", flush=True)

    write_text(out_dir / "results.json", json.dumps(results, indent=2, ensure_ascii=False) + "\n")
    write_summary(out_dir, results)
    print(f"wrote {out_dir}")


if __name__ == "__main__":
    main()
