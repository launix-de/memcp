#!/usr/bin/env python3
"""
MemCP SQL Test Runner (Optimized)

Runs structured SQL/SPARQL tests from YAML files.
- Declarative tests
- Unified execution path
- Compact success logging, verbose failure logging

Requirements (pip): requests, PyYAML
If missing, create a venv and install, e.g.:
  python3 -m venv .venv && . .venv/bin/activate && pip install -U requests PyYAML
"""

import sys
import os

# Dependency checks with clear install hints
try:
    import yaml  # PyYAML
except Exception as e:
    print("Missing dependency: PyYAML (module 'yaml').")
    print("Install with: pip install -U PyYAML")
    sys.exit(2)

try:
    import requests
except Exception as e:
    print("Missing dependency: requests.")
    print("Install with: pip install -U requests")
    sys.exit(2)

import json
import subprocess
import time
import random
import multiprocessing
from pathlib import Path
from base64 import b64encode
from typing import Dict, List, Any, Optional, Tuple
from urllib.parse import quote

# CPU measurement helpers
NUM_CPUS = multiprocessing.cpu_count()

def find_memcp_pid() -> Optional[int]:
    """Find the PID of the memcp process."""
    try:
        result = subprocess.run(['pgrep', '-f', 'memcp'], capture_output=True, text=True, timeout=2)
        pids = result.stdout.strip().split('\n')
        for pid_str in pids:
            if pid_str.strip():
                return int(pid_str.strip())
    except:
        pass
    return None

def get_process_cpu_times(pid: int) -> Optional[Tuple[float, float]]:
    """Get user and system CPU times for a process from /proc/[pid]/stat.
    Returns (utime + cutime, stime + cstime) in seconds, or None if unavailable."""
    try:
        with open(f'/proc/{pid}/stat', 'r') as f:
            parts = f.read().split()
            # Fields: utime(14), stime(15), cutime(16), cstime(17) - 1-indexed in docs, 0-indexed here
            utime = int(parts[13])  # user time
            stime = int(parts[14])  # system time
            cutime = int(parts[15])  # children user time
            cstime = int(parts[16])  # children system time
            # Convert from clock ticks to seconds (typically 100 Hz = 100 ticks/sec)
            hz = os.sysconf(os.sysconf_names['SC_CLK_TCK'])
            total_time = (utime + stime + cutime + cstime) / hz
            return total_time
    except:
        return None

def measure_cpu_load(pid: int, start_cpu: float, end_cpu: float, elapsed_sec: float) -> Optional[float]:
    """Calculate CPU load as percentage of total CPU capacity.
    Returns percentage where 100% = one core fully utilized, NUM_CPUS*100% = all cores."""
    if start_cpu is None or end_cpu is None or elapsed_sec <= 0:
        return None
    cpu_used = end_cpu - start_cpu
    # CPU load as percentage of wall time (100% = 1 core, 200% = 2 cores, etc.)
    return (cpu_used / elapsed_sec) * 100

# Global flag for connect-only mode
is_connect_only_mode = False

# Performance test configuration
PERF_TEST_ENABLED = os.environ.get("PERF_TEST", "0") == "1"
PERF_CALIBRATE = os.environ.get("PERF_CALIBRATE", "0") == "1"  # reset baselines to current times
PERF_NORECALIBRATE = os.environ.get("PERF_NORECALIBRATE", "0") == "1"  # freeze row counts for bisecting
PERF_EXPLAIN = os.environ.get("PERF_EXPLAIN", "0") == "1"  # show query plans
PERF_BASELINE_FILE = ".perf_baseline.json"
PERF_THRESHOLD_FACTOR = 1.3  # 30% tolerance over baseline
PERF_TARGET_MIN_MS = 10000  # target minimum query time (10s)
PERF_TARGET_MAX_MS = 20000  # target maximum query time (20s)
PERF_SCALE_FACTOR = 1.3  # scale up/down by 30%
PERF_DEFAULT_ROWS = 10000  # default starting row count
PERF_MAX_ROWS = 10_000_000  # allow large datasets for proper calibration
PERF_MAX_RAM_FRACTION = 0.3  # max 30% of RAM for table data

def get_max_rows_for_ram(bytes_per_row: int = 100) -> int:
    """Calculate max rows based on available RAM (30% limit)."""
    try:
        with open('/proc/meminfo', 'r') as f:
            for line in f:
                if line.startswith('MemTotal:'):
                    total_kb = int(line.split()[1])
                    max_bytes = int(total_kb * 1024 * PERF_MAX_RAM_FRACTION)
                    return max_bytes // bytes_per_row
    except:
        pass
    return 100_000_000  # fallback: 100M rows

class SQLTestRunner:
    def __init__(self, base_url="http://localhost:4321", username="root", password="admin"):
        self.base_url = base_url
        self.username = username
        self.password = password
        self.auth_header = self._create_auth_header()
        self.test_count = 0
        self.test_passed = 0
        self.failed_tests = []  # list of (name, is_noncritical)
        self.failed_critical = 0
        self.failed_noncritical = 0
        self.noncritical_count = 0
        self.noncritical_passed = 0
        self.setup_operations = []
        self.current_database = None
        self._ensured_dbs = set()
        self.suite_metadata = {}
        self._restart_handler = None  # callable to restart memcp between tests
        self.suite_syntax = None
        self.perf_baselines = {}  # test_name -> {"time_ms": float, "rows": int}
        self.perf_results = {}  # test_name -> {"time_ms": float, "rows": int}

    def set_restart_handler(self, fn):
        """Install a restart handler callable that restarts MemCP (returns True on success)."""
        self._restart_handler = fn

    def load_perf_baselines(self):
        """Load performance baselines from config file."""
        try:
            with open(PERF_BASELINE_FILE, 'r') as f:
                self.perf_baselines = json.load(f)
        except (FileNotFoundError, json.JSONDecodeError):
            self.perf_baselines = {}

    def save_perf_baselines(self):
        """Save updated performance baselines to config file.

        All tests in a suite share the same row count (based on slowest test)
        to avoid issues with DELETE operations.
        """
        if not self.perf_results:
            return

        max_rows = get_max_rows_for_ram()

        # Find the slowest test to determine shared scaling
        slowest_time = max((r["time_ms"] for r in self.perf_results.values()), default=0)
        # Get current row count (should be same for all tests in suite)
        current_rows = max((r["rows"] for r in self.perf_results.values()), default=PERF_DEFAULT_ROWS)

        if PERF_NORECALIBRATE:
            # Just update times, keep existing rows
            for name, result in self.perf_results.items():
                if name in self.perf_baselines:
                    self.perf_baselines[name]["time_ms"] = round(result["time_ms"], 1)
                else:
                    self.perf_baselines[name] = {"time_ms": round(result["time_ms"], 1), "rows": current_rows}
        else:
            # Scale based on slowest test (all tests use same row count)
            if slowest_time < PERF_TARGET_MIN_MS:
                new_rows = int(current_rows * PERF_SCALE_FACTOR)
            elif slowest_time > PERF_TARGET_MAX_MS:
                new_rows = max(1000, int(current_rows / PERF_SCALE_FACTOR))
            else:
                new_rows = current_rows

            # Apply RAM limit and hard cap
            new_rows = min(new_rows, max_rows, PERF_MAX_ROWS)

            # Update all baselines with shared row count
            for name, result in self.perf_results.items():
                self.perf_baselines[name] = {
                    "time_ms": round(result["time_ms"], 1),
                    "rows": new_rows
                }

        with open(PERF_BASELINE_FILE, 'w') as f:
            json.dump(self.perf_baselines, f, indent=2)
        print(f"📏 Updated performance baselines in {PERF_BASELINE_FILE}")

    # ----------------------
    # SQL identifier quoting
    # ----------------------
    def _quote_ident(self, name: str) -> str:
        if name is None:
            return "``"
        # Escape backticks by doubling them
        return f"`{str(name).replace('`', '``')}`"

    # ----------------------
    # Helpers
    # ----------------------
    def _create_auth_header(self):
        credentials = f"{self.username}:{self.password}"
        encoded = b64encode(credentials.encode()).decode()
        return {"Authorization": f"Basic {encoded}"}

    def _record_success(self, name: str, is_noncritical: bool = False, elapsed_ms: float = None, threshold_ms: float = None, rows: int = None, heap_mb: float = None, cpu_pct: float = None):
        self.test_passed += 1
        if elapsed_ms is not None and threshold_ms is not None:
            rows_info = f", {rows:,} rows" if rows else ""
            # Calculate time per row in microseconds
            if rows and rows > 0:
                us_per_row = (elapsed_ms * 1000) / rows
                rate_info = f", {us_per_row:.2f}µs/row"
            else:
                rate_info = ""
            # Show heap memory if available
            mem_info = f", {heap_mb:.1f}MB heap" if heap_mb else ""
            # Show CPU load as percentage of total capacity (100%/Ncores = one core)
            cpu_info = f", {cpu_pct:.0f}%/{NUM_CPUS*100}% CPU" if cpu_pct is not None else ""
            print(f"✅ {name} ({elapsed_ms:.1f}ms / {threshold_ms:.0f}ms{rows_info}{rate_info}{mem_info}{cpu_info})")
        else:
            print(f"✅ {name}")
        if is_noncritical:
            self.noncritical_passed += 1
            print(f"   ⚠️  Passed but flagged noncritical — enable soon")

    def _record_fail(self, name: str, reason: str, query: str, response: Optional[requests.Response], expect, is_noncritical: bool = False, elapsed_ms: float = None, threshold_ms: float = None):
        self.failed_tests.append((name, is_noncritical))
        if is_noncritical:
            self.failed_noncritical += 1
        else:
            self.failed_critical += 1
        time_info = f" ({elapsed_ms:.1f}ms / {threshold_ms:.0f}ms)" if elapsed_ms is not None else ""
        print(f"❌ {name}{' (noncritical)' if is_noncritical else ''}{time_info}")
        print(f"    Reason: {reason}")
        if query:
            print(f"    Query: {query[:200]}{'...' if len(query) > 200 else ''}")
        if response is not None:
            print(f"    HTTP {response.status_code}: {response.text[:500]}{'...' if len(response.text) > 500 else ''}")
        if expect is not None:
            print(f"    Expected: {expect}\n")

    # ----------------------
    # Core execution
    # ----------------------
    def ensure_database(self, database: str) -> None:
        if not database or database == "system" or database in self._ensured_dbs:
            return
        try:
            url = f"{self.base_url}/sql/system"
            create_db_sql = f"CREATE DATABASE IF NOT EXISTS {self._quote_ident(database)}"
            requests.post(url, data=create_db_sql, headers=self.auth_header, timeout=10)
            # verify availability with a lightweight call
            check_url = f"{self.base_url}/sql/{quote(database, safe='')}"
            for _ in range(3):
                resp = requests.post(check_url, data="SHOW TABLES", headers=self.auth_header, timeout=10)
                if resp is not None and "database" not in resp.text.lower():
                    self._ensured_dbs.add(database)
                    break
                time.sleep(0.1)
        except Exception:
            pass

    def execute_sql(self, database: str, query: str, auth_header: Optional[Dict[str, str]] = None, syntax: Optional[str] = None, session_id: Optional[str] = None) -> Optional[requests.Response]:
        # proactively ensure database exists (works for connect-only too)
        self.ensure_database(database)
        encoded_db = quote(database, safe='')
        normalized = self._normalize_syntax(syntax)
        route = "psql" if normalized == "postgresql" else "sql"
        url = f"{self.base_url}/{route}/{encoded_db}"
        headers = dict(auth_header if auth_header is not None else self.auth_header)
        if session_id:
            headers["X-Session-Id"] = session_id
        # Try request; if connection fails, wait for memcp to be ready and retry a few times
        for attempt in range(5):
            try:
                return requests.post(url, data=query, headers=headers, timeout=10)
            except Exception:
                # parse port from base_url
                try:
                    port = int(self.base_url.rsplit(':', 1)[1])
                except Exception:
                    port = 4321
                wait_for_memcp(port, timeout=5)
        return None

    def _normalize_syntax(self, syntax: Optional[str]) -> Optional[str]:
        if not syntax:
            return None
        syntax_lower = str(syntax).strip().lower()
        if syntax_lower in ["mysql", "default"]:
            return None
        if syntax_lower in ["postgres", "postgresql", "psql"]:
            return "postgresql"
        return syntax_lower

    def execute_sparql(self, database: str, query: str, auth_header: Optional[Dict[str, str]] = None) -> Optional[requests.Response]:
        try:
            encoded_db = quote(database, safe='')
            url = f"{self.base_url}/rdf/{encoded_db}"
            headers = auth_header if auth_header is not None else self.auth_header
            return requests.post(url, data=query, headers=headers, timeout=10)
        except Exception as e:
            print(f"Error executing SPARQL: {e}")
            return None

    def load_ttl(self, database: str, ttl_data: str) -> bool:
        try:
            self.ensure_database(database)
            url = f"{self.base_url}/rdf/{quote(database, safe='')}/load_ttl"
            response = requests.post(url, data=ttl_data, headers=self.auth_header, timeout=10)
            return response is not None and response.status_code == 200
        except Exception as e:
            print(f"Error loading TTL data: {e}")
            return False

    def parse_jsonl_response(self, response: requests.Response) -> Optional[List[Dict]]:
        if response is None:
            return None
        text = response.text.strip()
        if not text:
            return []
        results = []
        for line in text.splitlines():
            try:
                results.append(json.loads(line))
            except:
                continue
        return results

    # ----------------------
    # Performance test helpers
    # ----------------------
    def execute_scm(self, code: str) -> Optional[requests.Response]:
        """Execute Scheme code via /scm endpoint."""
        try:
            url = f"{self.base_url}/scm"
            return requests.post(url, data=code, headers=self.auth_header, timeout=600)
        except Exception as e:
            print(f"SCM execution error: {e}")
            return None

    def generate_and_insert_parallel(self, database: str, spec: Dict) -> bool:
        """Generate and insert data using parallel Scheme workers.

        This runs directly in the database with parallel goroutines for maximum performance.
        """
        table = spec.get("table")
        columns = spec.get("columns", [])
        num_rows = spec.get("rows", 1000)

        # Build column generators for Scheme (deterministic, based on row index)
        col_names = [c.get("name") for c in columns]
        col_generators = []
        for idx, col in enumerate(columns):
            col_type = col.get("type", "int")
            if col_type == "sequential":
                col_generators.append("i")
            elif col_type == "int":
                # Simple deterministic value based on i (multiplied to spread values)
                col_generators.append(f"(* i {37 + idx * 17})")
            elif col_type == "float":
                col_generators.append(f"(/ (* i {41 + idx * 13}) 100.0)")
            elif col_type == "string":
                col_generators.append(f'(concat "str_" i)')
            elif col_type == "bool":
                col_generators.append("(> i 0)")
            else:
                col_generators.append("i")

        # Scheme code that generates and inserts data with parallel workers
        cols_scm = " ".join([f'"{c}"' for c in col_names])
        row_generator = f"(list {' '.join(col_generators)})"

        # Build column definitions for table recreation
        col_defs = []
        for col in columns:
            col_name = col.get("name")
            col_type = col.get("type", "int")
            if col_type == "sequential":
                col_defs.append(f"{col_name} INT")
            elif col_type == "int":
                col_defs.append(f"{col_name} INT")
            elif col_type == "float":
                col_defs.append(f"{col_name} FLOAT")
            elif col_type == "string":
                col_defs.append(f"{col_name} TEXT")
            elif col_type == "bool":
                col_defs.append(f"{col_name} BOOLEAN")
            else:
                col_defs.append(f"{col_name} INT")

        # Drop and recreate table (faster than DELETE for large tables)
        self.execute_sql(database, f"DROP TABLE IF EXISTS {self._quote_ident(table)}", syntax=self.suite_syntax)
        # Run rebuild to compact memory before measuring
        self.execute_scm("(rebuild)")
        create_sql = f"CREATE TABLE {self._quote_ident(table)} ({', '.join(col_defs)})"
        self.execute_sql(database, create_sql, syntax=self.suite_syntax)

        # Sequential insert, rebuild to shard data, then get memory stats
        # Note: rebuild with repartition will automatically create parallel shards
        # when data is large enough, even without partitioning hints
        scm_code = f'''
(begin
  (set rows (map (produceN {num_rows}) (lambda (i) {row_generator})))
  (set cnt (insert "{database}" "{table}" '({cols_scm}) rows))
  (rebuild)
  (set mem (memstats))
  (list cnt (mem "heap_alloc"))
)
'''
        resp = self.execute_scm(scm_code)
        if resp is None:
            print(f"SCM response: None")
            return {"success": False}
        if resp.status_code != 200:
            print(f"SCM error ({resp.status_code}): {resp.text[:500]}")
            return {"success": False}
        # Parse result - should be (count heap_alloc)
        try:
            # Response is JSON array [count, heap_bytes]
            result = json.loads(resp.text.strip())
            if isinstance(result, list) and len(result) >= 2:
                cnt, heap_bytes = result[0], result[1]
                if cnt != num_rows:
                    print(f"SCM insert count mismatch: expected {num_rows}, got {cnt}")
                return {"success": True, "heap_bytes": heap_bytes}
        except:
            print(f"SCM result: {resp.text[:200]}")
        return {"success": True, "heap_bytes": 0}

    # ----------------------
    # Test execution
    # ----------------------
    def run_test_case(self, test_case: Dict, database: str) -> bool:
        self.test_count += 1
        name = test_case.get("name", f"Test {self.test_count}")
        is_noncritical = bool(test_case.get("noncritical"))
        if is_noncritical:
            self.noncritical_count += 1

        # Performance test handling
        yaml_threshold_ms = test_case.get("threshold_ms")
        is_perf_test = yaml_threshold_ms is not None
        perf_rows = PERF_DEFAULT_ROWS  # default row count
        if is_perf_test:
            # Get baseline data if available
            baseline = self.perf_baselines.get(name, {})
            if isinstance(baseline, dict):
                baseline_time = baseline.get("time_ms")
                baseline_rows = baseline.get("rows", PERF_DEFAULT_ROWS)
            else:
                # Legacy format: just a number
                baseline_time = baseline
                baseline_rows = PERF_DEFAULT_ROWS

            # Use baseline time × 1.3 as threshold ONLY if in target range
            # During scaling, use YAML threshold (generous)
            if baseline_time and not PERF_CALIBRATE and baseline_time >= PERF_TARGET_MIN_MS:
                threshold_ms = baseline_time * PERF_THRESHOLD_FACTOR
            else:
                threshold_ms = yaml_threshold_ms

            # Use baseline rows (which may have been scaled)
            perf_rows = baseline_rows

            if not PERF_TEST_ENABLED:
                print(f"⏭️  {name} (skipped - set PERF_TEST=1 to run)")
                self.test_count -= 1  # don't count skipped perf tests
                return True

        # Data generation for performance tests using Scheme
        generate_data = test_case.get("generate_data")
        generate_elapsed_ms = 0
        heap_bytes = 0
        if generate_data:
            # Use this test's calibrated row count
            generate_data_with_rows = generate_data.copy()
            generate_data_with_rows["rows"] = perf_rows
            start_gen = time.monotonic()
            gen_result = self.generate_and_insert_parallel(database, generate_data_with_rows)
            if not gen_result.get("success", False):
                return self._record_fail(name, "Data generation failed", None, None, None, is_noncritical)
            generate_elapsed_ms = (time.monotonic() - start_gen) * 1000
            heap_bytes = gen_result.get("heap_bytes", 0)

        query = test_case.get("sql") or test_case.get("sparql")
        is_sparql = "sparql" in test_case
        # auth: allow per-test overrides, fallback to suite metadata, then runner defaults
        tc_user = test_case.get("username") or self.suite_metadata.get("username") or self.username
        tc_pass = test_case.get("password") or self.suite_metadata.get("password") or self.password
        creds = f"{tc_user}:{tc_pass}".encode()
        auth_header = {"Authorization": f"Basic {b64encode(creds).decode()}"}
        test_syntax = test_case.get("syntax")
        active_syntax = self._normalize_syntax(test_syntax) if test_syntax is not None else self.suite_syntax
        session_id = test_case.get("session_id")

        # TTL preload if SPARQL
        if is_sparql and "ttl_data" in test_case:
            if not self.load_ttl(database, test_case["ttl_data"]):
                return self._record_fail(name, "TTL load failed", query, None, None)

        # Special handling: SHUTDOWN command triggers graceful restart flow
        response: Optional[requests.Response]
        cpu_pct = None  # CPU load percentage, measured during query execution
        if query and query.strip().upper() == "SHUTDOWN":
            # Issue shutdown
            resp = self.execute_sql(database, query, auth_header, active_syntax)
            if resp is not None and resp.status_code >= 500:
                response = resp
            else:
                # Treat SHUTDOWN as successful regardless of response body, even if the connection closed.
                if self._restart_handler is not None:
                    self._restart_handler()
                self._record_success(name, is_noncritical)
                return True

            response = resp
        else:
            # Show query plan if PERF_EXPLAIN is enabled
            if is_perf_test and PERF_EXPLAIN and not is_sparql:
                explain_resp = self.execute_sql(database, f"DESCRIBE {query}", auth_header, active_syntax)
                if explain_resp and explain_resp.status_code == 200:
                    print(f"    📋 Query plan for {name}:")
                    for line in explain_resp.text.strip().split('\n')[:10]:
                        print(f"       {line[:120]}")

            # Warmup runs for performance tests (2 unmeasured runs before the measured one)
            if is_perf_test and test_case.get("warmup", True):
                for _ in range(2):
                    self.execute_sparql(database, query, auth_header) if is_sparql else self.execute_sql(database, query, auth_header, active_syntax)

            # Get memcp PID and start CPU measurement for perf tests
            memcp_pid = find_memcp_pid() if is_perf_test else None
            start_cpu = get_process_cpu_times(memcp_pid) if memcp_pid else None

            # Execute query (with timing for perf tests)
            start_time = time.monotonic()
            response = self.execute_sparql(database, query, auth_header) if is_sparql else self.execute_sql(database, query, auth_header, active_syntax, session_id=session_id)
            elapsed_ms = (time.monotonic() - start_time) * 1000
            elapsed_sec = elapsed_ms / 1000

            # End CPU measurement
            end_cpu = get_process_cpu_times(memcp_pid) if memcp_pid else None
            cpu_pct = measure_cpu_load(memcp_pid, start_cpu, end_cpu, elapsed_sec) if memcp_pid else None

        if response is None:
            return self._record_fail(name, "No response", query, None, None, is_noncritical)

        results = self.parse_jsonl_response(response)

        # Check performance threshold
        if is_perf_test and elapsed_ms > threshold_ms:
            return self._record_fail(name, f"Too slow: {elapsed_ms:.1f}ms > {threshold_ms:.0f}ms", query, response,
                                     test_case.get("expect"), is_noncritical, elapsed_ms, threshold_ms)

        if self.validate_expectation(test_case, response, results):
            if is_perf_test:
                heap_mb = heap_bytes / (1024 * 1024) if heap_bytes else None
                self._record_success(name, is_noncritical, elapsed_ms, threshold_ms, perf_rows, heap_mb, cpu_pct)
                # Store result for baseline update (time and row count)
                self.perf_results[name] = {"time_ms": elapsed_ms, "rows": perf_rows}
                # Cleanup: drop table after perf test to free memory
                if generate_data:
                    table = generate_data.get("table")
                    self.execute_sql(database, f"DROP TABLE IF EXISTS {self._quote_ident(table)}", syntax=self.suite_syntax)
            else:
                self._record_success(name, is_noncritical)
            return True
        else:
            return self._record_fail(name, "Expectation mismatch", query, response, test_case.get("expect"), is_noncritical)

    def validate_expectation(self, test_case: Dict, response: requests.Response, results: Optional[List[Dict]]) -> bool:
        expect = test_case.get("expect", {})

        if expect.get("error"):
            return response.status_code != 200 or "Error" in response.text

        if "Error" in response.text or response.status_code != 200:
            return False

        if "affected_rows" in expect:
            expected = expect["affected_rows"]
            if results and results and "affected_rows" in results[0]:
                return results[0]["affected_rows"] == expected
            return True

        if results is None:
            return False

        if expect.get("rows") is not None:
            if len(results) != expect["rows"]:
                return False

        if expect.get("data"):
            for i, row in enumerate(expect["data"]):
                if i >= len(results):
                    return False
                for k, v in row.items():
                    if isinstance(v, float) and isinstance(results[i].get(k), (int, float)):
                        if abs(results[i][k] - v) > 0.01:
                            return False
                    elif results[i].get(k) != v:
                        return False
        return True

    # ----------------------
    # Setup & Cleanup
    # ----------------------
    def run_setup(self, setup_steps: List[Dict], database: str) -> bool:
        self.setup_operations = []
        self.current_database = database
        for step in setup_steps:
            self.setup_operations.append(step)
            resp = self.execute_sql(database, step['sql'], syntax=self.suite_syntax)
            if resp is None or resp.status_code not in [200, 500]:
                return False
        return True

    def run_cleanup(self, cleanup_steps: List[Dict], database: str) -> None:
        for step in cleanup_steps:
            self.execute_sql(database, step['sql'], syntax=self.suite_syntax)

    def cleanup_test_database(self, database: str) -> None:
        try:
            global is_connect_only_mode
            if is_connect_only_mode:
                resp = self.execute_sql(database, "SHOW TABLES")
                if resp is not None and resp.status_code == 200:
                    try:
                        tables = resp.json().get('data', [])
                        for row in tables:
                            tbl = list(row.values())[0]
                            self.execute_sql(database, f"DROP TABLE IF EXISTS {self._quote_ident(tbl)}")
                    except:
                        pass
                return
            self.execute_sql("system", f"DROP DATABASE IF EXISTS {self._quote_ident(database)}")
            # drop ensures next ensure_database will recreate
            if database in self._ensured_dbs:
                try:
                    self._ensured_dbs.remove(database)
                except KeyError:
                    pass
        except:
            pass

    # ----------------------
    # Spec Runner
    # ----------------------
    def run_test_spec(self, spec_file: str) -> bool:
        with open(spec_file, 'r') as f:
            spec = yaml.safe_load(f)

        metadata = spec.get('metadata', {})
        self.suite_metadata = metadata or {}
        self.suite_syntax = self._normalize_syntax(self.suite_metadata.get("syntax"))
        database = 'memcp-tests'

        # Load performance baselines for this machine
        if PERF_TEST_ENABLED:
            self.load_perf_baselines()
            if PERF_CALIBRATE:
                print("🔧 Calibration mode: resetting baselines to current times")

        print(f"🎯 Running suite: {metadata.get('description', spec_file)}")
        print(f"💾 Database: {database}")

        # fresh DB for this suite; ensure exists after cleanup
        self.cleanup_test_database(database)
        self.ensure_database(database)

        if spec.get('setup') and not self.run_setup(spec['setup'], database):
            print("❌ Setup failed")
            return False

        for test in spec.get('test_cases', []):
            self.run_test_case(test, database)

        if spec.get('cleanup'):
            self.run_cleanup(spec['cleanup'], database)
        self.cleanup_test_database(database)

        print("="*60)
        total = self.test_count
        passed = self.test_passed
        failed_total = len(self.failed_tests)
        failed_noncrit = self.failed_noncritical
        failed_crit = self.failed_critical
        print(f"📊 Results: {passed}/{total} passed | Failures: {failed_total} | Noncritical failures: {failed_noncrit}")
        if self.failed_tests:
            print("❌ Failed:")
            for name, is_noncrit in self.failed_tests:
                suffix = " (noncritical)" if is_noncrit else ""
                print(f"   - {name}{suffix}")
        else:
            print("🎉 All tests passed!")
        print("="*60)

        # Update performance baselines on success (or in calibration mode)
        if PERF_TEST_ENABLED and self.perf_results and (failed_crit == 0 or PERF_CALIBRATE):
            self.save_perf_baselines()

        # Suite success is determined solely by critical tests
        return failed_crit == 0

def wait_for_memcp(port=4321, timeout=30) -> bool:
    for _ in range(timeout):
        try:
            requests.get(f"http://localhost:{port}", timeout=2)
            return True
        except:
            time.sleep(1)
    return False

def start_memcp_process(port: int) -> subprocess.Popen | None:
    try:
        datadir = os.environ.get("MEMCP_TEST_DATADIR", f"/tmp/memcp-sql-tests-{port}")
        env = os.environ.copy()
        godebug = env.get("GODEBUG", "")
        if "invalidptr=" not in godebug:
            env["GODEBUG"] = f"{godebug},invalidptr=0" if godebug else "invalidptr=0"
        proc = subprocess.Popen([
            "./memcp", "-data", datadir,
            f"--api-port={port}", f"--mysql-port={port+1000}",
            "--disable-mysql", "lib/main.scm"
        ], cwd=os.path.dirname(os.path.abspath(__file__)),
           env=env, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        if not wait_for_memcp(port):
            return None
        return proc
    except Exception:
        return None

def stop_memcp_process(proc: subprocess.Popen) -> None:
    try:
        proc.stdin.close()
        proc.wait(timeout=10)
    except Exception:
        try:
            proc.kill()
        except Exception:
            pass

def kill_memcp_by_port(port: int) -> None:
    pattern = f"memcp.*--api-port={port}"
    try:
        subprocess.run(["pkill", "-f", pattern], check=False)
    except Exception:
        pass

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 run_sql_tests.py <test_spec.yaml> [port] [--connect-only]")
        sys.exit(1)

    spec_file = sys.argv[1]
    port = 4321
    connect_only = False

    for arg in sys.argv[2:]:
        if arg == "--connect-only":
            connect_only = True
        elif arg.isdigit():
            port = int(arg)

    base_url = f"http://localhost:{port}"
    global is_connect_only_mode
    is_connect_only_mode = connect_only

    memcp_process = None
    if connect_only:
        try:
            requests.get(base_url, timeout=2)
        except:
            print(f"❌ Cannot connect to MemCP on port {port}")
            sys.exit(1)
    else:
        try:
            requests.get(base_url, timeout=2)
        except:
            memcp_process = start_memcp_process(port)
            if not memcp_process:
                print("❌ Failed to start MemCP")
                sys.exit(1)

    runner = SQLTestRunner(base_url)
    if not connect_only:
        def restart_handler() -> bool:
            nonlocal memcp_process
            if memcp_process:
                stop_memcp_process(memcp_process)
                memcp_process = None
            memcp_process = start_memcp_process(port)
            return memcp_process is not None
        runner.set_restart_handler(restart_handler)
    success = runner.run_test_spec(spec_file)

    if not connect_only and memcp_process:
        stop_memcp_process(memcp_process)

    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()
