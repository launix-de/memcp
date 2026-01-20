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
from pathlib import Path
from base64 import b64encode
from typing import Dict, List, Any, Optional
from urllib.parse import quote

# Global flag for connect-only mode
is_connect_only_mode = False

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

    def set_restart_handler(self, fn):
        """Install a restart handler callable that restarts MemCP (returns True on success)."""
        self._restart_handler = fn

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

    def _record_success(self, name: str, is_noncritical: bool = False):
        self.test_passed += 1
        print(f"✅ {name}")
        if is_noncritical:
            self.noncritical_passed += 1
            print(f"   ⚠️  Passed but flagged noncritical — enable soon")

    def _record_fail(self, name: str, reason: str, query: str, response: Optional[requests.Response], expect, is_noncritical: bool = False):
        self.failed_tests.append((name, is_noncritical))
        if is_noncritical:
            self.failed_noncritical += 1
        else:
            self.failed_critical += 1
        print(f"❌ {name}{' (noncritical)' if is_noncritical else ''}")
        print(f"    Reason: {reason}")
        if query:
            print(f"    Query: {query[:200]}{'...' if len(query) > 200 else ''}")
        if response:
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

    def execute_sql(self, database: str, query: str, auth_header: Optional[Dict[str, str]] = None, syntax: Optional[str] = None) -> Optional[requests.Response]:
        # proactively ensure database exists (works for connect-only too)
        self.ensure_database(database)
        encoded_db = quote(database, safe='')
        normalized = self._normalize_syntax(syntax)
        route = "psql" if normalized == "postgresql" else "sql"
        url = f"{self.base_url}/{route}/{encoded_db}"
        headers = auth_header if auth_header is not None else self.auth_header
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
    # Test execution
    # ----------------------
    def run_test_case(self, test_case: Dict, database: str) -> bool:
        self.test_count += 1
        name = test_case.get("name", f"Test {self.test_count}")
        is_noncritical = bool(test_case.get("noncritical"))
        if is_noncritical:
            self.noncritical_count += 1
        query = test_case.get("sql") or test_case.get("sparql")
        is_sparql = "sparql" in test_case
        # auth: allow per-test overrides, fallback to suite metadata, then runner defaults
        tc_user = test_case.get("username") or self.suite_metadata.get("username") or self.username
        tc_pass = test_case.get("password") or self.suite_metadata.get("password") or self.password
        creds = f"{tc_user}:{tc_pass}".encode()
        auth_header = {"Authorization": f"Basic {b64encode(creds).decode()}"}
        test_syntax = test_case.get("syntax")
        active_syntax = self._normalize_syntax(test_syntax) if test_syntax is not None else self.suite_syntax

        # TTL preload if SPARQL
        if is_sparql and "ttl_data" in test_case:
            if not self.load_ttl(database, test_case["ttl_data"]):
                return self._record_fail(name, "TTL load failed", query, None, None)

        # Special handling: SHUTDOWN command triggers graceful restart flow
        response: Optional[requests.Response]
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
            # Execute query
            response = self.execute_sparql(database, query, auth_header) if is_sparql else self.execute_sql(database, query, auth_header, active_syntax)
        if response is None:
            return self._record_fail(name, "No response", query, None, None)

        results = self.parse_jsonl_response(response)

        if self.validate_expectation(test_case, response, results):
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
