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

"""Regression tests for SQL runner response classification."""

from pathlib import Path
from types import SimpleNamespace
from unittest import mock
import os
import json
import io
from contextlib import redirect_stdout
import itertools
import requests
import signal
import sys
import tempfile
import threading
import time
import unittest


sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from run_sql_tests import (  # noqa: E402
    PERFORMANCE_ARCH_ENV,
    PERFORMANCE_CALIBRATION_ROUNDS,
    PERFORMANCE_MEASURED_NS_ENV,
    PERFORMANCE_REFERENCE_NS_ENV,
    PERFORMANCE_REFERENCE_NS_PER_MIB,
    PERFORMANCE_SCALE_ENV,
    PLANNER_TIME_TOLERANCE_FACTOR,
    SQLTestRunner,
    _load_runner_config,
    adaptive_measurement_complete,
    discover_performance_ci_suites,
    is_error_response,
    initialize_performance_recording,
    load_perf_regression_waivers,
    load_performance_scale,
    observe_atomic_json,
    parse_perf_regression_waivers,
    performance_ab_threshold_ms,
    performance_fixture_cases,
    run_performance_ab,
    summarize_performance_fixtures,
    validate_performance_fixture,
    performance_architecture,
    performance_case_fingerprint,
    performance_case_key,
    performance_regression_pct,
    performance_sample_ns,
    performance_scale_from_samples,
    planner_time_limit_with_tolerance_ms,
    publish_performance_scale,
    request_shared_supervisor_restart,
    revision_specific_scm,
    resolve_timing_samples,
    resolve_warmup_runs,
    run_test_specs,
    scaled_compile_time_limit_ms,
    scaled_wall_clock_limit_ms,
    sql_request_is_retry_safe,
    suite_execution_mode,
    wait_for_shared_supervisor_generation,
)
from tools.check_test_table_names import mutable_table_collisions  # noqa: E402


class UpgradeSnapshotContractTest(unittest.TestCase):
    """Exercise the existing shell runner's inline exporter without a server."""

    def setUp(self):
        script = Path(__file__).resolve().parents[1] / "tests/storage/persistence/upgrade-validate.sh"
        code = script.read_text().split("<<'PYTHON'\n", 1)[1].split("\nPYTHON", 1)[0]
        definitions = code.rsplit("\ntry:\n", 1)[0]
        self.exporter = {}
        with mock.patch.object(sys, "argv", ["-", str(script), "1", "compare", "unused", ""]):
            exec(compile(definitions, str(script), "exec"), self.exporter)

    def fixture(self):
        import base64
        encode = lambda value: base64.b64encode(value).decode("ascii")
        columns = [{"name": "id", "encoding": "integer", "metadata": ["INT"]},
                   {"name": "value", "encoding": "bytes-base64", "metadata": ["BLOB"]}]
        return {"format": "memcp-upgrade-values-v1", "tables": {
            "up_string_nodict": {"columns": columns, "rows": [
                ["50", encode(b"previously unchecked row")],
                ["51", None], ["52", encode(b"NULL")],
                ["53", encode(b"a\x00b\tc\nd\r\xff")]]},
            "up_blob": {"columns": columns, "rows": [["1", encode(b"a" * 3000)]]},
            "up_json": {"columns": columns, "rows": [["1", encode(b'{"nested":{"v":42}}')]]},
            "up_float": {"columns": [{"name": "v", "encoding": "number", "metadata": ["DOUBLE"]}],
                         "rows": [["-0"], ["1.1557281258737144"]]}}}

    def test_every_value_and_inventory_difference_fails(self):
        import base64
        import copy

        def replace_bytes(tables, table, before, after):
            row = tables[table]["rows"][0]
            value = base64.b64decode(row[1]).replace(before, after, 1)
            row[1] = base64.b64encode(value).decode("ascii")

        changes = [
            lambda t: t["up_string_nodict"]["rows"][0].__setitem__(1, "Y29ycnVwdA=="),
            lambda t: replace_bytes(t, "up_blob", b"a" * 3000, b"a" * 1500 + b"b" + b"a" * 1499),
            lambda t: replace_bytes(t, "up_json", b"42", b"43"),
            lambda t: t["up_float"]["rows"][0].__setitem__(0, "0"),
            lambda t: t["up_float"]["rows"][1].__setitem__(0, "1.1557281258737146"),
            lambda t: t["up_string_nodict"]["rows"][1].__setitem__(1, "TlVMTA=="),
            lambda t: t["up_string_nodict"]["rows"].append(t["up_string_nodict"]["rows"][0]),
            lambda t: t["up_float"]["columns"][0].__setitem__("metadata", ["INT"]),
            lambda t: t.pop("up_json"),
        ]
        expected = self.fixture()
        for change in changes:
            with self.subTest(change=change):
                actual = copy.deepcopy(expected)
                change(actual["tables"])
                self.exporter["capture"] = lambda: actual
                with self.assertRaises(ValueError):
                    self.exporter["compare"](expected)

    def test_binary_null_and_record_boundaries_survive_capture(self):
        import base64
        raw = b"nul\x00tab\tline\ncarriage\rslash\\invalid\xff"
        token = base64.b64encode(raw)
        replies = [
            [[b"up_bytes"]],
            [[b"Type", b"Field", b"Collation", b"RawType", b"Dimensions", b"Null",
              b"Key", b"Default", b"DefaultExpression", b"Extra", b"Privileges", b"Comment",
              b"RowEstimate"],
             [b"INT", b"id"] + [b""] * 10 + [b"3"],
             [b"BLOB", b"content"] + [b""] * 10 + [b"3"]],
            [[b"0", b"1", b"0", token], [b"0", b"2", b"1", b"NULL"],
             [b"0", b"3", b"0", base64.b64encode(b"NULL")]],
        ]
        self.exporter["query"] = mock.Mock(side_effect=replies)
        captured = self.exporter["capture"]()["tables"]["up_bytes"]
        self.assertNotIn("RowEstimate", captured["columns"][0]["metadata"])
        self.assertEqual(captured["columns"][0]["name"], "id")
        rows = captured["rows"]
        self.assertEqual(base64.b64decode(rows[0][1]), raw)
        self.assertIsNone(rows[1][1])
        self.assertEqual(base64.b64decode(rows[2][1]), b"NULL")

    def test_sql_failure_is_fatal_even_with_empty_stdout(self):
        import subprocess
        with mock.patch.object(subprocess, "run", side_effect=subprocess.CalledProcessError(1, "mysql", b"", b"SQL failed")):
            with self.assertRaises(subprocess.CalledProcessError):
                self.exporter["capture"]()

    def test_post_dml_oracle_preserves_untouched_values(self):
        def table(names, rows):
            return {"columns": [{"name": n} for n in names], "rows": rows}
        expected = {"tables": {
            "up_float": table(["id", "val"], [["0", "1.25"], ["1", "2.75"]]),
            "up_const": table(["id", "status"], [["0", "YWN0aXZl"]]),
            "up_enum": table(["id", "grade"], [["98", "RA=="], ["99", "RA=="]]),
            "up_compute": table(["id", "val", "doubled"], [["1", "5", "10"], ["2", "7", "14"]]),
            "up_unrelated": table(["id", "bytes"], [["50", "AAkK/w=="]]),
        }}
        actual = self.exporter["changed_oracle"](expected)["tables"]
        self.assertEqual(actual["up_float"]["rows"], [["0", "3.14159"], ["1", "2.75"]])
        self.assertEqual(actual["up_const"]["rows"], [["0", "YWN0aXZl"], ["50", "YWN0aXZl"]])
        self.assertEqual(actual["up_enum"]["rows"], [["98", "RA=="]])
        self.assertEqual(actual["up_compute"]["rows"], [["1", "100", "200"], ["2", "7", "14"]])
        self.assertEqual(actual["up_unrelated"]["rows"], [["50", "AAkK/w=="]])

    def test_oracle_cannot_be_overwritten(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "old.json"
            self.exporter["write_new"](path, self.fixture())
            before = path.read_bytes()
            with self.assertRaises(FileExistsError):
                self.exporter["write_new"](path, {})
            self.assertEqual(path.read_bytes(), before)


class UpgradeWorkflowContractTest(unittest.TestCase):
    def setUp(self):
        import yaml
        workflow = Path(__file__).resolve().parents[1] / ".github/workflows/upgrade-compatibility.yml"
        self.jobs = yaml.safe_load(workflow.read_text())["jobs"]

    def test_required_gate_rejects_failed_cancelled_and_skipped_work(self):
        import subprocess
        gate = self.jobs["upgrade-compatibility"]
        self.assertEqual(gate["if"], "always()")
        self.assertCountEqual(gate["needs"], ["changes", "versions", "check-upgrade"])
        code = gate["steps"][0]["run"]
        statuses = ("success", "failure", "cancelled", "skipped")
        for relevant, event, changes, versions, upgrade in itertools.product(
                ("true", "false"), ("pull_request", "workflow_dispatch"), statuses, statuses, statuses):
            with self.subTest(relevant=relevant, event=event, changes=changes, versions=versions, upgrade=upgrade):
                env = dict(os.environ, CHANGE_RESULT=changes, RELEVANT=relevant,
                           VERSION_RESULT=versions, UPGRADE_RESULT=upgrade, EVENT_NAME=event)
                result = subprocess.run(["bash", "-c", code], env=env, capture_output=True)
                needed = relevant == "true" or event == "workflow_dispatch"
                expected = changes == "success" and (
                    versions == upgrade == ("success" if needed else "skipped"))
                self.assertEqual(result.returncode == 0, expected)

    def select_refs(self, requested, base=""):
        import subprocess
        step = self.jobs["versions"]["steps"][0]
        with tempfile.TemporaryDirectory() as tmp:
            output = Path(tmp) / "output"
            env = dict(os.environ, REQUESTED_REFS=requested, PR_BASE_REF=base,
                       GITHUB_OUTPUT=str(output))
            result = subprocess.run(["bash", "-c", step["run"]], env=env, capture_output=True)
            refs = None
            if output.exists():
                refs = json.loads(output.read_text().removeprefix("refs="))
            return result.returncode, refs

    def test_predecessor_defaults_and_deduplication(self):
        self.assertEqual(self.select_refs("[]", "abc123"), (0, ["abc123"]))
        self.assertEqual(self.select_refs("[]"), (0, ["master"]))
        self.assertEqual(self.select_refs('["v0.1", "master", "v0.1"]'), (0, ["v0.1", "master"]))
        self.assertEqual(self.select_refs(json.dumps(["master"] * 17)), (0, ["master"]))

    def test_predecessor_invalid_input_does_not_publish_matrix(self):
        for value in ("invalid JSON", '"master"', "null", "{}", "[1]", '[""]', '["  "]',
                      json.dumps(["ref" + str(i) for i in range(17)])):
            with self.subTest(value=value):
                status, refs = self.select_refs(value)
                self.assertNotEqual(status, 0)
                self.assertIsNone(refs)


class HookDiagnosticsContractTest(unittest.TestCase):
    def setUp(self):
        self.root = Path(__file__).resolve().parents[1]
        self.hook = (self.root / "git-pre-commit").read_text()

    def function(self, name):
        start = self.hook.index(name + "() {\n")
        end = self.hook.index("\n}\n", start) + 3
        return self.hook[start:end]

    def test_nonresponding_http_readiness_has_overall_deadline(self):
        import socketserver
        import subprocess
        release = threading.Event()
        connected = threading.Event()

        class SilentHandler(socketserver.BaseRequestHandler):
            def handle(self):
                connected.set()
                release.wait(5)

        try:
            server = socketserver.ThreadingTCPServer(("127.0.0.1", 0), SilentHandler)
        except PermissionError as error:
            self.skipTest("local sockets unavailable: " + str(error))
        server.daemon_threads = True
        worker = threading.Thread(target=server.serve_forever, daemon=True)
        worker.start()
        try:
            code = self.function("wait_for_sql_ready") + "\ntest_port=" + str(server.server_address[1]) + "\nwait_for_sql_ready 1 1\n"
            started = time.monotonic()
            result = subprocess.run(["bash", "-c", code], capture_output=True, timeout=4,
                                    env=dict(os.environ, NO_PROXY="localhost,127.0.0.1"))
            self.assertNotEqual(result.returncode, 0)
            self.assertTrue(connected.is_set(), "probe must reach the silent HTTP listener")
            self.assertLess(time.monotonic() - started, 3)
        finally:
            release.set()
            server.shutdown()
            server.server_close()
            worker.join(timeout=2)

    def test_failed_curl_cannot_pass_with_http_200_output(self):
        import subprocess
        code = self.function("wait_for_sql_ready") + "\ncurl() { printf 200; return 28; }\ntest_port=1\nwait_for_sql_ready 1 1\n"
        result = subprocess.run(["bash", "-c", code], capture_output=True, timeout=4)
        self.assertNotEqual(result.returncode, 0)

    def test_failure_and_signal_keep_partial_suite_and_server_logs(self):
        import subprocess
        traps = self.hook[self.hook.index("trap 'cleanup"):self.hook.index("\nif ! start_supervisor;")]
        for action, expected in (("exit 1", 1), ("kill -TERM $$", 143), ("kill -INT $$", 130)):
            with self.subTest(action=action), tempfile.TemporaryDirectory() as tmp:
                logs = Path(tmp) / "logs"
                logs.mkdir()
                (logs / "suite.out").write_bytes(b"partial test output\n")
                server = Path(tmp) / "server.log"
                server.write_bytes(b"server diagnostics\n")
                code = self.function("cleanup") + "\nstop_supervisor() { printf '%s' \"${1:-TERM}\" > \"$tmpdir/stop-signal\"; }\n"
                code += 'did_cleanup=0\nactive_pids=()\n' + traps + "\n" + action
                result = subprocess.run(["bash", "-c", code], capture_output=True, timeout=4,
                                        env=dict(os.environ, tmpdir=str(logs), memcp_log=str(server)))
                self.assertEqual(result.returncode, expected)
                self.assertEqual((logs / "suite.out").read_bytes(), b"partial test output\n")
                self.assertEqual((logs / "memcp.log").read_bytes(), b"server diagnostics\n")
                self.assertEqual((logs / "stop-signal").read_text(), "USR2")

    def test_supervisor_dumps_only_owned_child_and_does_not_restart(self):
        import subprocess
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            child = root / "memcp"
            child.write_text("#!/usr/bin/env python3\nimport signal,sys,time\n"
                             "def quit(sig, frame):\n print('OWNED STACK REQUEST', flush=True)\n sys.exit(0)\n"
                             "signal.signal(signal.SIGQUIT, quit)\nprint('CHILD READY', flush=True)\n"
                             "while True: time.sleep(0.1)\n")
            child.chmod(0o755)
            code = self.function("run_memcp_forever") + "\n" + self.function("stop_supervisor")
            code += '\npkill() { :; }\nenable_mysql=0\ntest_port=12345\ntest_data_dir=unused\n'
            code += 'run_memcp_forever &\nsupervisor_pid=$!\n'
            code += 'for i in {1..40}; do if grep -q "CHILD READY" "$memcp_log" 2>/dev/null; then break; fi; sleep 0.1; done\n'
            code += 'stop_supervisor USR2\n'
            result = subprocess.run(["bash", "-c", code], cwd=root, capture_output=True, timeout=8,
                                    env=dict(os.environ, memcp_log=str(root / "server.log"),
                                             supervisor_generation_file=str(root / "generation")))
            self.assertEqual(result.returncode, 0, result.stderr)
            log = (root / "server.log").read_text()
            self.assertEqual(log.count("CHILD READY"), 1)
            self.assertEqual(log.count("OWNED STACK REQUEST"), 1)

    def test_suite_start_is_visible_before_readiness_and_traps_cover_startup(self):
        code = self.function("run_one")
        self.assertLess(code.index("START suite"), code.index("wait_for_sql_ready"))
        self.assertLess(self.hook.index("trap 'cleanup"), self.hook.index("if ! start_supervisor;"))

    def test_ci_timeout_leaves_time_to_upload_logs_on_any_outcome(self):
        import yaml
        for name, job in (("test.yml", "test"), ("jit-test.yml", "jit-test")):
            with self.subTest(workflow=name):
                spec = yaml.safe_load((self.root / ".github/workflows" / name).read_text())["jobs"][job]
                run = next(step for step in spec["steps"] if "git-pre-commit" in step.get("run", ""))
                artifact = next(step for step in spec["steps"] if step.get("name") == "Upload test diagnostics")
                self.assertEqual(run["timeout-minutes"], 30)
                self.assertGreater(spec["timeout-minutes"], run["timeout-minutes"])
                self.assertEqual(artifact["if"], "always()")
                self.assertEqual(artifact["with"]["path"].rstrip("/"), run["env"]["MEMCP_TEST_LOGDIR"])


class PerformanceScaleContractTest(unittest.TestCase):
    def test_performance_discovery_honors_independent_ci_opt_in(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "enabled.yaml").write_text(
                "metadata: {ci: false, perf_ci: true}\n"
                "test_cases: [{name: measured, sql: SELECT 1, threshold_ms: 10}]\n"
            )
            (root / "manual.yaml").write_text(
                "metadata: {ci: false}\n"
                "test_cases: [{name: manual, sql: SELECT 1, threshold_ms: 10}]\n"
            )
            self.assertEqual(discover_performance_ci_suites(root), [str(root / "enabled.yaml")])

    def test_scm_performance_cannot_bypass_timing_gate(self) -> None:
        runner = SQLTestRunner("http://localhost:1")
        response = SimpleNamespace(status_code=200, text="true", headers={})
        clock = itertools.count(0, 1_000_000)
        with mock.patch("run_sql_tests.PERF_TEST_ENABLED", True), \
                mock.patch("run_sql_tests.PERF_AB_MODE", ""), \
                mock.patch("run_sql_tests.find_memcp_pid", return_value=None), \
                mock.patch("run_sql_tests.time.monotonic_ns", side_effect=lambda: next(clock)), \
                mock.patch("run_sql_tests.requests.post", return_value=response) as post:
            result = runner.run_test_case({
                "name": "SCM must be measured", "scm": "true", "threshold_ms": 0.01,
                "repetitions": 2, "warmup": 0, "expect": {"rows": 1},
            }, "memcp-tests")
        self.assertFalse(result)
        self.assertEqual(post.call_count, 2)

    def test_scm_scalar_data_expectations_are_validated(self) -> None:
        runner = SQLTestRunner("http://localhost:1")
        for expected, actual, wanted in [
            ("ok", "ok", True), ("ok", "wrong", False),
            (True, True, True), (True, 1, False),
            (20, 20, True), (20, 19, False),
            (None, None, True), ([1, 2], [1, 2], True),
            ({"n": 1}, "not a row", False),
        ]:
            with self.subTest(expected=expected, actual=actual):
                response = SimpleNamespace(status_code=200, text="value", headers={})
                self.assertEqual(runner.validate_expectation(
                    {"scm": "value", "expect": {"data": [expected]}},
                    response, [actual]), wanted)

    def test_ci_workload_seed_initializes_safe_rows(self) -> None:
        seed = Path(__file__).resolve().parents[1] / "tests/performance/ci-workloads.json"
        with tempfile.TemporaryDirectory() as tmp:
            baseline = Path(tmp) / "baseline.json"
            with mock.patch("run_sql_tests.PERF_BASELINE_SEED", str(seed)), \
                    mock.patch("run_sql_tests.PERF_BASELINE_FILE", str(baseline)):
                initialize_performance_recording()
                config = _load_runner_config()
        self.assertEqual(config["_ci_workload"]["default_rows"], 1000)
        self.assertEqual(
            config["tests/performance/baseline.yaml::Perf: MATRIX MULT"]["rows"],
            30,
        )

    def test_repetitions_default_and_explicit_count(self) -> None:
        self.assertEqual(resolve_timing_samples({}, False), 1)
        self.assertEqual(resolve_timing_samples({}, True), 5)
        self.assertEqual(resolve_timing_samples({"timing_samples": 3}, False), 3)
        self.assertEqual(resolve_timing_samples({"repetitions": 100}, True), 100)

    def test_repetitions_reject_ambiguous_or_empty_counts(self) -> None:
        for invalid in (True, 0, 2.5, "3"):
            with self.subTest(invalid=invalid):
                with self.assertRaisesRegex(ValueError, "positive integer"):
                    resolve_timing_samples({"timing_samples": invalid}, False)

    def test_repetitions_and_timing_samples_are_mutually_exclusive(self) -> None:
        with self.assertRaisesRegex(ValueError, "either repetitions or timing_samples"):
            resolve_timing_samples({"repetitions": 5, "timing_samples": 5}, True)

    def test_adaptive_measurement_requires_five_runs_and_time_budget(self) -> None:
        with mock.patch("run_sql_tests.PERF_MIN_MEASURE_MS", 250):
            self.assertFalse(adaptive_measurement_complete([250_000_000] * 4))
            self.assertFalse(adaptive_measurement_complete([2_499_999] * 100))
            self.assertTrue(adaptive_measurement_complete([50_000_000] * 5))

    def test_adaptive_measurement_ignores_outliers_when_selecting_duration(self) -> None:
        with mock.patch("run_sql_tests.PERF_MIN_MEASURE_MS", 1000):
            samples = [60_000_000] * 5 + [700_000_000]
            self.assertFalse(adaptive_measurement_complete(samples))
            self.assertTrue(adaptive_measurement_complete([60_000_000] * 17))

    def test_performance_sample_uses_median_to_ignore_one_scheduling_outlier(self) -> None:
        self.assertEqual(performance_sample_ns([78, 79, 80, 81, 200]), 80)
        self.assertEqual(performance_sample_ns([20, 10]), 15)
        with self.assertRaisesRegex(ValueError, "at least one sample"):
            performance_sample_ns([])

    def test_warmup_accepts_zero_and_counts(self) -> None:
        self.assertEqual(resolve_warmup_runs({}, True), 2)
        self.assertEqual(resolve_warmup_runs({"warmup": False}, True), 0)
        self.assertEqual(resolve_warmup_runs({"warmup": 7}, True), 7)

    def test_ab_metric_is_per_repetition_and_cold_candidate_gets_bonus(self) -> None:
        self.assertEqual(performance_ab_threshold_ms(100, 20, 2, 2), 120)
        self.assertAlmostEqual(performance_ab_threshold_ms(100, 20, 2, 0), 220)

    def test_ab_jitter_budget_is_amortized_over_repetitions(self) -> None:
        self.assertEqual(performance_ab_threshold_ms(100, 20, 2, 2, 1, 50), 170)
        self.assertEqual(performance_ab_threshold_ms(100, 20, 2, 2, 100, 50), 120.5)

    def test_regression_limit_is_bounded_and_case_overrides_suite(self) -> None:
        self.assertEqual(performance_regression_pct({}, {}), 20)
        self.assertEqual(
            performance_regression_pct({"max_regression_pct": 15}, {"max_regression_pct": 25}),
            15,
        )
        with self.assertRaisesRegex(ValueError, "10 through 30"):
            performance_regression_pct({"max_regression_pct": 31}, {})

    def test_performance_identity_is_stable_across_worktrees(self) -> None:
        self.assertEqual(
            performance_case_key("/tmp/base/tests/performance/a.yaml", "query"),
            "tests/performance/a.yaml::query",
        )

    def test_workload_fingerprint_allows_only_repetition_and_warmup_changes(self) -> None:
        base = {
            "name": "query", "sql": "SELECT 1", "threshold_ms": 30000,
            "repetitions": 5, "warmup": 2,
        }
        timing_change = dict(base, repetitions=100, warmup=0)
        query_change = dict(base, sql="SELECT 2")
        self.assertEqual(
            performance_case_fingerprint(base),
            performance_case_fingerprint(timing_change),
        )
        self.assertNotEqual(
            performance_case_fingerprint(base),
            performance_case_fingerprint(query_change),
        )
        self.assertNotEqual(
            performance_case_fingerprint(base, [{"sql": "INSERT INTO t VALUES (1)"}]),
            performance_case_fingerprint(base, [{"sql": "INSERT INTO t VALUES (2)"}]),
        )

    def test_cold_planner_budget_allows_bounded_measurement_jitter(self) -> None:
        self.assertEqual(PLANNER_TIME_TOLERANCE_FACTOR, 1.2)
        calibration = {"scale": 1.0}
        self.assertEqual(planner_time_limit_with_tolerance_ms(500, calibration), 600)

    def test_architecture_aliases_select_stable_profiles(self) -> None:
        self.assertEqual(performance_architecture("AMD64"), "x86_64")
        self.assertEqual(performance_architecture("arm64"), "aarch64")
        self.assertEqual(performance_architecture("mips64"), "other")

    def test_reference_machine_never_tightens_wall_clock_budget(self) -> None:
        reference_ns = (
            PERFORMANCE_REFERENCE_NS_PER_MIB
            * PERFORMANCE_CALIBRATION_ROUNDS["x86_64"]
        )
        calibration = performance_scale_from_samples(
            "x86_64", [reference_ns // 2, reference_ns // 2, reference_ns]
        )
        self.assertEqual(calibration["scale"], 1.0)
        self.assertEqual(scaled_wall_clock_limit_ms(0.02, calibration), 20.0)

    def test_slower_machine_scales_wall_clock_and_compile_budgets(self) -> None:
        reference_ns = (
            PERFORMANCE_REFERENCE_NS_PER_MIB
            * PERFORMANCE_CALIBRATION_ROUNDS["aarch64"]
        )
        calibration = performance_scale_from_samples(
            "aarch64", [reference_ns * 4] * 5
        )
        self.assertEqual(calibration["scale"], 4.0)
        self.assertEqual(scaled_wall_clock_limit_ms(0.02, calibration), 80.0)
        self.assertEqual(scaled_compile_time_limit_ms(300, calibration), 1200.0)
        self.assertEqual(planner_time_limit_with_tolerance_ms(300, calibration), 1440.0)

    def test_unreasonably_slow_calibration_fails_instead_of_disabling_gates(self) -> None:
        reference_ns = (
            PERFORMANCE_REFERENCE_NS_PER_MIB
            * PERFORMANCE_CALIBRATION_ROUNDS["armv7l"]
        )
        with self.assertRaisesRegex(ValueError, "exceeds supported maximum"):
            performance_scale_from_samples("armv7l", [reference_ns * 17])

    def test_inherited_scale_must_match_protected_measurements(self) -> None:
        profile = performance_architecture()
        reference_ns = (
            PERFORMANCE_REFERENCE_NS_PER_MIB
            * PERFORMANCE_CALIBRATION_ROUNDS[profile]
        )
        environment = {
            PERFORMANCE_ARCH_ENV: profile,
            PERFORMANCE_SCALE_ENV: "8",
            PERFORMANCE_MEASURED_NS_ENV: str(reference_ns),
            PERFORMANCE_REFERENCE_NS_ENV: str(reference_ns),
        }
        with mock.patch.dict("os.environ", environment, clear=False):
            with self.assertRaisesRegex(ValueError, "does not match its measurements"):
                load_performance_scale()

    def test_published_calibration_round_trips_without_recalibration(self) -> None:
        profile = performance_architecture()
        reference_ns = (
            PERFORMANCE_REFERENCE_NS_PER_MIB
            * PERFORMANCE_CALIBRATION_ROUNDS[profile]
        )
        calibration = {
            "architecture": profile,
            "scale": 2.0,
            "measured_ns": reference_ns * 2,
            "reference_ns": reference_ns,
        }
        with mock.patch.dict("os.environ", {}, clear=True):
            publish_performance_scale(calibration)
            self.assertEqual(load_performance_scale(), calibration)


class ErrorResponseContractTest(unittest.TestCase):
    def test_missing_response_is_not_a_sql_error(self) -> None:
        self.assertFalse(is_error_response(None))

    def test_success_response_is_not_a_sql_error(self) -> None:
        response = SimpleNamespace(status_code=200, text='{"value": 1}')
        self.assertFalse(is_error_response(response))

    def test_http_error_is_a_sql_error(self) -> None:
        response = SimpleNamespace(status_code=500, text="SQL Error: invalid query")
        self.assertTrue(is_error_response(response))

    def test_error_payload_is_a_sql_error(self) -> None:
        response = SimpleNamespace(status_code=200, text="Error: invalid query")
        self.assertTrue(is_error_response(response))


class InterruptedRequestContractTest(unittest.TestCase):
    def test_only_read_only_sql_is_safe_to_replay(self) -> None:
        for query in (
            "SELECT * FROM items",
            " -- inspect\nSHOW TABLES",
            "/* plan only */ EXPLAIN SELECT * FROM items",
            "DESCRIBE items",
        ):
            with self.subTest(query=query):
                self.assertTrue(sql_request_is_retry_safe(query))
        for query in (
            "INSERT INTO items VALUES (1)",
            "UPDATE items SET value = value + 1",
            "DELETE FROM items",
            "WITH source AS (SELECT 1) INSERT INTO items SELECT * FROM source",
            "SET @counter = 1",
        ):
            with self.subTest(query=query):
                self.assertFalse(sql_request_is_retry_safe(query))

    def test_mutation_is_not_retried_by_default_after_unknown_outcome(self) -> None:
        runner = SQLTestRunner("http://localhost:1")
        runner.ensure_database = lambda _database: None
        with mock.patch("run_sql_tests.requests.post", side_effect=TimeoutError) as post, \
                mock.patch("run_sql_tests.wait_for_memcp") as wait_for_memcp:
            response = runner.execute_sql(
                "memcp-tests", "INSERT INTO items VALUES (1)",
            )
        self.assertIsNone(response)
        post.assert_called_once()
        wait_for_memcp.assert_not_called()

    def test_read_only_request_may_retry_after_connection_loss(self) -> None:
        runner = SQLTestRunner("http://localhost:1")
        runner.ensure_database = lambda _database: None
        recovered = SimpleNamespace(status_code=200, text='{"value": 1}')
        with mock.patch(
            "run_sql_tests.requests.post", side_effect=[ConnectionError, recovered]
        ) as post, mock.patch("run_sql_tests.wait_for_memcp") as wait_for_memcp:
            response = runner.execute_sql("memcp-tests", "SELECT 1")
        self.assertIs(response, recovered)
        self.assertEqual(post.call_count, 2)
        wait_for_memcp.assert_called_once()

    def test_interrupted_mutation_is_not_retried_after_connection_loss(self) -> None:
        runner = SQLTestRunner("http://localhost:1")
        runner.ensure_database = lambda _database: None
        with mock.patch("run_sql_tests.requests.post", side_effect=ConnectionError), \
                mock.patch("run_sql_tests.wait_for_memcp") as wait_for_memcp:
            response = runner.execute_sql(
                "memcp-tests", "UPDATE items SET value = value + 1",
                retry_on_connection_failure=False,
            )
        self.assertIsNone(response)
        wait_for_memcp.assert_not_called()

    def test_completed_interrupted_request_is_exempt_from_normal_time_budget(self) -> None:
        runner = SQLTestRunner("http://localhost:1")
        response = SimpleNamespace(status_code=200, text='{"affected_rows": 999}')
        runner.ensure_database = lambda _database: None
        runner.execute_sql = mock.Mock(return_value=response)
        with mock.patch("run_sql_tests.time.monotonic_ns", side_effect=[0, 10_000_000_000]):
            passed = runner.run_test_case({
                "name": "mutation racing a crash",
                "sql": "UPDATE items SET value = value + 1",
                "expect": {"interrupted_ok": True},
            }, "memcp-tests")
        self.assertTrue(passed)
        self.assertFalse(runner.execute_sql.call_args.kwargs["retry_on_connection_failure"])

    def test_shutdown_does_not_wait_for_the_process_it_intentionally_stops(self) -> None:
        runner = SQLTestRunner("http://localhost:1")
        runner.ensure_database = lambda _database: None
        runner.execute_sql = mock.Mock(return_value=None)
        restart = mock.Mock(return_value=True)
        runner.set_restart_handler(restart)

        self.assertTrue(runner.run_test_case({
            "name": "managed restart",
            "sql": "SHUTDOWN",
        }, "memcp-tests"))
        self.assertFalse(runner.execute_sql.call_args.kwargs["retry_on_connection_failure"])
        restart.assert_called_once_with()

    def test_connect_only_shutdown_waits_for_the_shared_supervisor(self) -> None:
        runner = SQLTestRunner("http://localhost:23456")
        runner.ensure_database = lambda _database: None
        runner.execute_sql = mock.Mock(return_value=None)

        with mock.patch("run_sql_tests.shared_supervisor_generation", return_value="7"), \
                mock.patch("run_sql_tests.wait_for_shared_supervisor_generation", return_value=True) as generation_wait, \
                mock.patch("run_sql_tests.wait_for_sql_ready", return_value=True) as wait:
            self.assertTrue(runner.run_test_case({
                "name": "shared restart",
                "sql": "SHUTDOWN",
            }, "memcp-tests"))

        generation_wait.assert_called_once_with("7", 10)
        wait.assert_called_once_with(
            "http://localhost:23456",
            "root",
            "admin",
            "memcp-tests",
            timeout=120,
        )

    def test_generation_wait_does_not_accept_the_old_server(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            generation = Path(tmp) / "generation"
            generation.write_text("4\n", encoding="utf-8")
            with mock.patch.dict(os.environ, {
                "MEMCP_TEST_SUPERVISOR_GENERATION_FILE": str(generation),
            }):
                self.assertFalse(wait_for_shared_supervisor_generation("4", 0))
                generation.write_text("5\n", encoding="utf-8")
                self.assertTrue(wait_for_shared_supervisor_generation("4", 1))


class AtomicJSONObserverContractTest(unittest.TestCase):
    def test_accepts_complete_atomic_replacements(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "schema.json"
            path.write_text('{"generation": 0}', encoding="utf-8")

            def replace() -> None:
                for generation in range(1, 30):
                    candidate = path.with_suffix(".tmp")
                    candidate.write_text(f'{{"generation": {generation}}}', encoding="utf-8")
                    candidate.replace(path)

            writer = threading.Thread(target=replace)
            writer.start()
            reads, error = observe_atomic_json(path, 0.05)
            writer.join()
            self.assertGreater(reads, 0)
            self.assertIsNone(error)

    def test_rejects_partial_generation(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "schema.json"
            path.write_text('{"generation": 0}', encoding="utf-8")

            def truncate() -> None:
                time.sleep(0.01)
                path.write_text('{"generation":', encoding="utf-8")

            writer = threading.Thread(target=truncate)
            writer.start()
            reads, error = observe_atomic_json(path, 0.1)
            writer.join()
            self.assertGreater(reads, 0)
            self.assertIsNotNone(error)

    def test_rejects_rename_away_before_replacement(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "schema.json"
            backup = Path(tmp) / "schema.json.old"
            path.write_text('{"generation": 0}', encoding="utf-8")

            def replace_nonatomically() -> None:
                time.sleep(0.01)
                path.replace(backup)
                path.write_text('{"generation": 1}', encoding="utf-8")

            writer = threading.Thread(target=replace_nonatomically)
            writer.start()
            reads, error = observe_atomic_json(path, 0.1)
            writer.join()
            self.assertGreater(reads, 0)
            self.assertIsNotNone(error)


class FailFastParallelContractTest(unittest.TestCase):
    def test_ab_record_uses_revision_specific_physical_scm(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            spec = root / "tests" / "performance" / "physical.yaml"
            spec.parent.mkdir(parents=True)
            spec.write_text(
                "test_cases:\n"
                "  - name: physical probe\n"
                "    scm: '(old_scan_interface)'\n",
                encoding="utf-8",
            )
            case = {
                "name": "physical probe",
                "scm": "(new_scan_interface)",
                "revision_specific_scm": True,
            }
            with mock.patch("run_sql_tests.PERF_AB_MODE", "record"), \
                    mock.patch.dict(os.environ, {"MEMCP_TEST_WORKTREE": str(root)}):
                self.assertEqual(
                    revision_specific_scm(case, "tests/performance/physical.yaml"),
                    "(old_scan_interface)",
                )

    def test_normal_run_keeps_current_revision_physical_scm(self) -> None:
        case = {
            "name": "physical probe",
            "scm": "(new_scan_interface)",
            "revision_specific_scm": True,
        }
        with mock.patch("run_sql_tests.PERF_AB_MODE", ""):
            self.assertEqual(
                revision_specific_scm(case, "tests/performance/physical.yaml"),
                "(new_scan_interface)",
            )

    def test_ab_mode_skips_non_measurement_cases_in_performance_suite(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            spec = Path(tmp) / "performance.yaml"
            spec.write_text(
                "metadata: {description: A/B selection}\n"
                "test_cases:\n"
                "  - {name: correctness helper, sql: SELECT 1}\n"
                "  - {name: measured query, sql: SELECT 1, threshold_ms: 30000}\n",
                encoding="utf-8",
            )
            runner = SQLTestRunner("http://localhost:1")
            runner.ensure_database = lambda _database: None
            observed = []
            runner.run_test_case = lambda case, _database: observed.append(case["name"]) or True
            with mock.patch("run_sql_tests.PERF_AB_MODE", "record"):
                self.assertTrue(runner.run_test_spec(str(spec)))
        self.assertEqual(observed, ["measured query"])

    def test_performance_suite_setups_finish_before_measurement_phase(self) -> None:
        specs = ["tests/performance/a.yaml", "tests/performance/b.yaml"]
        barrier = threading.Barrier(2)
        events = []
        lock = threading.Lock()

        def prepare(_runner, spec_file):
            barrier.wait(timeout=1)
            with lock:
                events.append(("prepare", spec_file))
            return True

        def measure(_runner, spec_file, setup_done=False):
            self.assertTrue(setup_done)
            self.assertEqual(sum(kind == "prepare" for kind, _ in events), 2)
            events.append(("measure", spec_file))
            return True

        with mock.patch("run_sql_tests.PERF_TEST_ENABLED", True), \
                mock.patch("run_sql_tests.performance_measurement_count", return_value=1), \
                mock.patch.object(SQLTestRunner, "prepare_test_spec", prepare), \
                mock.patch.object(SQLTestRunner, "run_test_spec", measure):
            self.assertTrue(run_test_specs(specs, "http://localhost:1", 1, True, 2))

        self.assertCountEqual(events[2:], [("measure", specs[0]), ("measure", specs[1])])

    def test_performance_isolated_suites_run_as_separate_phases(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            specs = [Path(tmp) / "first.yaml", Path(tmp) / "second.yaml"]
            for spec in specs:
                spec.write_text(
                    "metadata: {isolated: true}\n"
                    "test_cases:\n"
                    "  - {name: measured, sql: SELECT 1, threshold_ms: 100}\n",
                    encoding="utf-8",
                )
            events = []

            def prepare(_runner, spec_file):
                events.append(("prepare", Path(spec_file).name))
                return True

            def measure(_runner, spec_file, setup_done=False):
                self.assertTrue(setup_done)
                events.append(("measure", Path(spec_file).name))
                return True

            with mock.patch("run_sql_tests.PERF_TEST_ENABLED", True), \
                    mock.patch("run_sql_tests.performance_measurement_count", return_value=1), \
                    mock.patch.object(SQLTestRunner, "prepare_test_spec", prepare), \
                    mock.patch.object(SQLTestRunner, "run_test_spec", measure):
                self.assertTrue(run_test_specs(
                    [str(spec) for spec in specs],
                    "http://localhost:1", 1, True, 2,
                ))

        self.assertEqual(events, [
            ("prepare", "first.yaml"),
            ("measure", "first.yaml"),
            ("prepare", "second.yaml"),
            ("measure", "second.yaml"),
        ])

    def test_performance_setup_restart_runs_once_before_measurements(self) -> None:
        specs = ["tests/performance/a.yaml", "tests/performance/b.yaml"]
        events = []

        def prepare(runner, spec_file):
            runner.suite_metadata = {
                "restart_after_setup": spec_file == specs[0],
            }
            events.append(("prepare", spec_file))
            return True

        def restart(_runner, _database):
            self.assertEqual(sum(kind == "prepare" for kind, _ in events), 2)
            events.append(("restart", "shared"))
            return True

        def measure(_runner, spec_file, setup_done=False):
            self.assertTrue(setup_done)
            self.assertEqual(sum(kind == "restart" for kind, _ in events), 1)
            events.append(("measure", spec_file))
            return True

        with mock.patch("run_sql_tests.PERF_TEST_ENABLED", True), \
                mock.patch("run_sql_tests.performance_measurement_count", return_value=1), \
                mock.patch.object(SQLTestRunner, "prepare_test_spec", prepare), \
                mock.patch.object(SQLTestRunner, "restart_server_after_setup", restart), \
                mock.patch.object(SQLTestRunner, "run_test_spec", measure):
            self.assertTrue(run_test_specs(specs, "http://localhost:1", 1, True, 2))

        self.assertEqual(sum(kind == "restart" for kind, _ in events), 1)

    def test_performance_rounds_finish_all_fills_before_serial_measurements(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            first = Path(tmp) / "first.yaml"
            second = Path(tmp) / "second.yaml"
            first.write_text(
                "metadata: {description: first}\n"
                "test_cases:\n"
                "  - {name: first-1, sql: SELECT 1, threshold_ms: 100}\n"
                "  - {name: first-2, sql: SELECT 1, threshold_ms: 100}\n",
                encoding="utf-8",
            )
            second.write_text(
                "metadata: {description: second}\n"
                "test_cases:\n"
                "  - {name: second-1, sql: SELECT 1, threshold_ms: 100}\n",
                encoding="utf-8",
            )
            events = []
            lock = threading.Lock()

            def prepare(runner, spec_file):
                runner.suite_metadata = {}
                runner.current_spec_file = spec_file
                return True

            def run_case(runner, test_case, _database):
                name = test_case["name"]
                with lock:
                    events.append(("fill", name))
                runner._perf_round_setup_barrier.wait(timeout=2)
                with lock:
                    events.append(("measure", name))
                return True

            def settled(_base_url):
                with lock:
                    events.append(("settled", "round"))
                return True

            with mock.patch("run_sql_tests.PERF_TEST_ENABLED", True), \
                    mock.patch("run_sql_tests.PERF_AB_MODE", "record"), \
                    mock.patch("run_sql_tests.wait_for_performance_setup_quiescence", settled), \
                    mock.patch.object(SQLTestRunner, "ensure_database"), \
                    mock.patch.object(SQLTestRunner, "prepare_test_spec", prepare), \
                    mock.patch.object(SQLTestRunner, "run_test_case", run_case):
                self.assertTrue(run_test_specs(
                    [str(first), str(second)], "http://localhost:1", 1, True, 2,
                ))

        first_round_done = max(
            events.index(("measure", "first-1")),
            events.index(("measure", "second-1")),
        )
        self.assertLess(events.index(("settled", "round")), first_round_done)
        self.assertGreater(events.index(("fill", "first-2")), first_round_done)
        self.assertEqual(sum(kind == "settled" for kind, _ in events), 2)

    def test_fail_fast_preserves_parallel_groups(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            spec = Path(tmp) / "parallel.yaml"
            spec.write_text(
                "metadata:\n"
                "  description: parallel contract\n"
                "test_cases:\n"
                "  - name: first\n"
                "    parallel: together\n"
                "  - name: second\n"
                "    parallel: together\n",
                encoding="utf-8",
            )
            runner = SQLTestRunner("http://localhost:1", fail_fast=True)
            runner.ensure_database = lambda _database: None
            barrier = threading.Barrier(2)
            completed = []

            def run_case(test_case, _database):
                barrier.wait(timeout=1)
                completed.append(test_case["name"])
                runner.test_count += 1
                runner.test_passed += 1
                return True

            runner.run_test_case = run_case
            self.assertTrue(runner.run_test_spec(str(spec)))
            self.assertCountEqual(completed, ["first", "second"])


class SuiteIsolationContractTest(unittest.TestCase):
    def test_exclusive_suites_may_share_fixture_names(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            for name in ("first", "second"):
                (root / f"{name}.yaml").write_text(
                    "metadata:\n"
                    "  isolated: true\n"
                    "test_cases:\n"
                    "  - name: reset shared diagnostics\n"
                    "    sql: TRUNCATE TABLE shared_diagnostics\n",
                    encoding="utf-8",
                )
            self.assertEqual(mutable_table_collisions(root), {})

    def test_parallel_suites_may_not_share_fixture_names(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            for name in ("first", "second"):
                (root / f"{name}.yaml").write_text(
                    "metadata:\n"
                    "  description: parallel fixture contract\n"
                    "test_cases:\n"
                    "  - name: create fixture\n"
                    "    sql: CREATE TABLE duplicate_fixture (id INT)\n",
                    encoding="utf-8",
                )
            self.assertEqual(
                set(mutable_table_collisions(root)),
                {"duplicate_fixture"},
            )

    def test_shared_restart_request_targets_only_the_declared_supervisor(self) -> None:
        with mock.patch.dict(os.environ, {"MEMCP_TEST_SUPERVISOR_PID": "12345"}):
            with mock.patch("run_sql_tests.os.kill") as kill:
                self.assertTrue(request_shared_supervisor_restart())
        kill.assert_called_once_with(12345, signal.SIGUSR1)

    def test_shared_restart_request_is_disabled_without_a_supervisor(self) -> None:
        with mock.patch.dict(os.environ, {}, clear=True):
            with mock.patch("run_sql_tests.os.kill") as kill:
                self.assertFalse(request_shared_supervisor_restart())
        kill.assert_not_called()

    def test_isolated_restart_suite_is_exclusive_on_the_shared_server(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            spec = Path(tmp) / "restart.yaml"
            spec.write_text(
                "metadata:\n"
                "  isolated: true\n"
                "test_cases:\n"
                "  - name: restart\n"
                "    sql: SHUTDOWN\n",
                encoding="utf-8",
            )
            self.assertEqual(suite_execution_mode(str(spec)), "exclusive")

    def test_restart_suite_is_exclusive_on_the_shared_server(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            spec = Path(tmp) / "restart.yaml"
            spec.write_text(
                "test_cases:\n"
                "  - name: restart\n"
                "    sql: SHUTDOWN\n",
                encoding="utf-8",
            )
            self.assertEqual(suite_execution_mode(str(spec)), "exclusive")

    def test_plain_isolated_suite_is_exclusive_on_the_shared_server(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            spec = Path(tmp) / "isolated.yaml"
            spec.write_text(
                "metadata:\n"
                "  isolated: true\n"
                "test_cases:\n"
                "  - name: select\n"
                "    sql: SELECT 1\n",
                encoding="utf-8",
            )
            self.assertEqual(suite_execution_mode(str(spec)), "exclusive")

    def test_precommit_hook_never_allocates_suite_local_data_directories(self) -> None:
        hook = (Path(__file__).resolve().parents[1] / "git-pre-commit").read_text(
            encoding="utf-8",
        )
        self.assertNotIn("managed_subprocess", hook)
        self.assertNotIn("managed_data_dir", hook)
        self.assertIn(
            'python3 -u run_sql_tests.py "$tf" $test_port --connect-only "${runner_args[@]}"',
            hook,
        )

    def test_precommit_hook_runs_safe_suites_in_parallel_by_default(self) -> None:
        hook = (Path(__file__).resolve().parents[1] / "git-pre-commit").read_text(
            encoding="utf-8",
        )
        self.assertIn('fail_fast_mode="${MEMCP_FAIL_FAST:-0}"', hook)

class PerfRegressionWaiverContractTest(unittest.TestCase):
    def test_parses_one_waiver_with_reason(self) -> None:
        waivers = parse_perf_regression_waivers(
            "Fix the thing\n\n"
            "Perf-Regression-Accepted: tests/performance/foo.yaml::Slow case | "
            "correlated plan is required for correctness here\n"
        )
        self.assertEqual(
            waivers,
            {"tests/performance/foo.yaml::Slow case": "correlated plan is required for correctness here"},
        )

    def test_parses_multiple_waivers_across_commits(self) -> None:
        # git log --format=%B concatenates one PR's commit bodies like this.
        waivers = parse_perf_regression_waivers(
            "commit one\n\nPerf-Regression-Accepted: a.yaml::A | reason a\n\n"
            "commit two\n\nPerf-Regression-Accepted: b.yaml::B | reason b\n"
        )
        self.assertEqual(waivers, {"a.yaml::A": "reason a", "b.yaml::B": "reason b"})

    def test_reason_is_optional(self) -> None:
        waivers = parse_perf_regression_waivers("Perf-Regression-Accepted: a.yaml::A\n")
        self.assertEqual(waivers, {"a.yaml::A": ""})

    def test_ignores_unrelated_commit_text(self) -> None:
        self.assertEqual(parse_perf_regression_waivers("just a normal commit message\n"), {})
        self.assertEqual(parse_perf_regression_waivers(""), {})
        self.assertEqual(parse_perf_regression_waivers(None), {})

    def test_key_matching_is_exact_no_prefix_or_glob(self) -> None:
        waivers = parse_perf_regression_waivers(
            "Perf-Regression-Accepted: a.yaml::Exact case | ok\n"
        )
        self.assertIn("a.yaml::Exact case", waivers)
        self.assertNotIn("a.yaml::Exact case (extended)", waivers)
        self.assertNotIn("a.yaml::Exact", waivers)

    def test_load_from_file_env_var(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            waivers_file = Path(tmp) / "waivers.txt"
            waivers_file.write_text("Perf-Regression-Accepted: a.yaml::A | ok\n")
            with mock.patch.dict(os.environ, {"PERF_REGRESSION_WAIVERS_FILE": str(waivers_file)}):
                self.assertEqual(load_perf_regression_waivers(), {"a.yaml::A": "ok"})

    def test_load_is_empty_without_env_var_or_missing_file(self) -> None:
        with mock.patch.dict(os.environ, {}, clear=False):
            os.environ.pop("PERF_REGRESSION_WAIVERS_FILE", None)
            self.assertEqual(load_perf_regression_waivers(), {})
        with mock.patch.dict(os.environ, {"PERF_REGRESSION_WAIVERS_FILE": "/nonexistent/path"}):
            self.assertEqual(load_perf_regression_waivers(), {})

    def test_waived_case_passes_through_and_is_recorded(self) -> None:
        runner = SQLTestRunner("http://localhost:1")
        runner.perf_regression_waivers = {"?::SCM must be measured": "known, accepted"}
        response = SimpleNamespace(status_code=200, text="true", headers={})
        clock = itertools.count(0, 1_000_000)
        with mock.patch("run_sql_tests.PERF_TEST_ENABLED", True), \
                mock.patch("run_sql_tests.PERF_AB_MODE", ""), \
                mock.patch("run_sql_tests.find_memcp_pid", return_value=None), \
                mock.patch("run_sql_tests.time.monotonic_ns", side_effect=lambda: next(clock)), \
                mock.patch("run_sql_tests.requests.post", return_value=response):
            result = runner.run_test_case({
                "name": "SCM must be measured", "scm": "true", "threshold_ms": 0.01,
                "repetitions": 2, "warmup": 0, "expect": {"rows": 1},
            }, "memcp-tests")
        self.assertTrue(result)
        self.assertEqual(len(runner.waived_regressions), 1)
        self.assertEqual(runner.waived_regressions[0][0], "SCM must be measured")
        self.assertEqual(runner.waived_regressions[0][2], "known, accepted")

    def test_unrelated_waiver_does_not_cover_a_different_case(self) -> None:
        runner = SQLTestRunner("http://localhost:1")
        runner.perf_regression_waivers = {"?::Some other case": "known, accepted"}
        response = SimpleNamespace(status_code=200, text="true", headers={})
        clock = itertools.count(0, 1_000_000)
        with mock.patch("run_sql_tests.PERF_TEST_ENABLED", True), \
                mock.patch("run_sql_tests.PERF_AB_MODE", ""), \
                mock.patch("run_sql_tests.find_memcp_pid", return_value=None), \
                mock.patch("run_sql_tests.time.monotonic_ns", side_effect=lambda: next(clock)), \
                mock.patch("run_sql_tests.requests.post", return_value=response):
            result = runner.run_test_case({
                "name": "SCM must be measured", "scm": "true", "threshold_ms": 0.01,
                "repetitions": 2, "warmup": 0, "expect": {"rows": 1},
            }, "memcp-tests")
        self.assertFalse(result)
        self.assertEqual(runner.waived_regressions, [])

    def test_waived_case_covers_a_query_killed_by_the_server(self) -> None:
        # A regression severe enough to trip the server's own long-running-query
        # guard never produces an HTTP response at all -- it never reaches the
        # elapsed_ms > threshold_ms check, it lands on the separate "No response"
        # path. A matching waiver must cover that path too.
        runner = SQLTestRunner("http://localhost:1")
        runner.perf_regression_waivers = {"?::SCM must be measured": "known, accepted"}
        clock = itertools.count(0, 1_000_000)
        with mock.patch("run_sql_tests.PERF_TEST_ENABLED", True), \
                mock.patch("run_sql_tests.PERF_AB_MODE", ""), \
                mock.patch("run_sql_tests.find_memcp_pid", return_value=None), \
                mock.patch("run_sql_tests.time.monotonic_ns", side_effect=lambda: next(clock)), \
                mock.patch("run_sql_tests.requests.post",
                           side_effect=requests.RequestException("simulated kill")):
            result = runner.run_test_case({
                "name": "SCM must be measured", "scm": "true", "threshold_ms": 0.01,
                "repetitions": 2, "warmup": 0, "expect": {"rows": 1},
            }, "memcp-tests")
        self.assertTrue(result)
        self.assertEqual(len(runner.waived_regressions), 1)
        self.assertEqual(runner.waived_regressions[0][0], "SCM must be measured")
        self.assertEqual(runner.waived_regressions[0][2], "known, accepted")

    def test_unwaived_no_response_still_fails(self) -> None:
        runner = SQLTestRunner("http://localhost:1")
        clock = itertools.count(0, 1_000_000)
        with mock.patch("run_sql_tests.PERF_TEST_ENABLED", True), \
                mock.patch("run_sql_tests.PERF_AB_MODE", ""), \
                mock.patch("run_sql_tests.find_memcp_pid", return_value=None), \
                mock.patch("run_sql_tests.time.monotonic_ns", side_effect=lambda: next(clock)), \
                mock.patch("run_sql_tests.requests.post",
                           side_effect=requests.RequestException("simulated kill")):
            result = runner.run_test_case({
                "name": "SCM must be measured", "scm": "true", "threshold_ms": 0.01,
                "repetitions": 2, "warmup": 0, "expect": {"rows": 1},
            }, "memcp-tests")
        self.assertFalse(result)
        self.assertEqual(runner.waived_regressions, [])


class PerformanceFixtureContractTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.base, self.candidate = self.root / "base", self.root / "candidate"
        for tree in (self.base, self.candidate):
            tree.mkdir()
            (tree / "memcp").write_bytes(b"fake binary")
        directory = self.candidate / "tests/performance"
        directory.mkdir(parents=True)
        self.suites = []
        for name in ("cold", "later"):
            path = directory / (name + ".yaml")
            path.write_text(json.dumps({
                "metadata": {"isolated": True, "restart_after_setup": True},
                "setup": ["CREATE TABLE fixture (id int)"],
                "test_cases": [{"name": name, "sql": "SELECT 1", "threshold_ms": 1000,
                                "warmup": 0, "timing_samples": 1, "expect": {"rows": 1}}],
            }))
            self.suites.append(str(path))
        self.seed = self.root / "seed.json"
        self.seed.write_text(json.dumps({"schema_version": 1, "default_rows": 1000, "rows": {}}))
        self.output = self.root / "result.json"
        self.calls = []
        self.durations = lambda suite, role, index: 100
        self.mutate = lambda result, index: None
        self.exit_code = 0

    def subprocess(self, command, *, cwd, env, stdout, stderr):
        suite = command[3]
        role = "A" if Path(env["MEMCP_TEST_WORKTREE"]) == self.base else "B"
        data = Path(env["MEMCP_TEST_DATA_DIR"])
        self.assertTrue(data.is_dir())
        self.assertEqual(list(data.iterdir()), [])
        self.assertNotIn(str(data), [call[2] for call in self.calls])
        (data / "mutated-fixture").write_text("each invocation changes its fixture")
        self.assertEqual(env["PERF_AB_MODE"], "record")
        self.assertEqual(env["PERF_FIXTURE_TRIAL"], "1")
        self.assertIn("--fail-fast", command)
        self.assertNotIn("--connect-only", command)
        spec = json.loads(Path(suite).read_text())
        self.assertTrue(spec["metadata"]["restart_after_setup"])
        self.assertEqual(spec["test_cases"][0]["warmup"], 0)
        self.assertEqual(spec["test_cases"][0]["timing_samples"], 1)
        self.calls.append((Path(suite).stem, role, str(data)))
        result = {"schema_version": 1}
        for key, case in performance_fixture_cases(suite).items():
            duration = self.durations(Path(suite).stem, role, len(self.calls))
            result[key] = {"time_ms": duration, "time_per_repetition_ms": duration,
                           "rows": 1000, "repetitions": 1, "warmup": 0,
                           "max_regression_pct": 20.0, "samples_ns": [int(duration * 1_000_000)],
                           "workload_sha256": performance_case_fingerprint(case, spec["setup"], None)}
        self.mutate(result, len(self.calls))
        Path(env["PERF_BASELINE_FILE"]).write_text(json.dumps(result))
        stdout.write("QUERY_TIME raw preserved\n")
        return SimpleNamespace(returncode=self.exit_code)

    def run_experiment(self):
        with mock.patch("run_sql_tests.PERF_BASELINE_FILE", str(self.output)), \
                mock.patch("run_sql_tests.PERF_BASELINE_SEED", str(self.seed)), \
                mock.patch("run_sql_tests.PERF_AB_JITTER_MS", 50), \
                mock.patch("run_sql_tests.subprocess.run", side_effect=self.subprocess), \
                redirect_stdout(io.StringIO()):
            return run_performance_ab(self.base, self.candidate, self.suites)

    def test_clean_initial_pass_does_not_repeat(self):
        self.assertTrue(self.run_experiment())
        self.assertEqual([(s, r) for s, r, _ in self.calls],
                         [("cold", "A"), ("cold", "B"), ("later", "A"), ("later", "B")])
        self.assertTrue(all(not Path(data).exists() for _, _, data in self.calls))

    def test_verification_is_after_all_suites_and_keeps_original_slow_sample(self):
        self.durations = lambda suite, role, index: 500 if index == 2 else 100
        self.assertTrue(self.run_experiment())
        self.assertEqual([(s, r) for s, r, _ in self.calls[:4]],
                         [("cold", "A"), ("cold", "B"), ("later", "A"), ("later", "B")])
        self.assertEqual("".join(r for _, r, _ in self.calls[4:]), "ABBA" * 3)
        result = json.loads(self.output.read_text())[performance_case_key(self.suites[0], "cold")]
        self.assertEqual(result["fixture_trials"], 7)
        self.assertEqual(result["b_samples_ms"], [500] + [100] * 6)
        self.assertAlmostEqual(result["threshold_ms"], 120 + 50 / 7)
        self.assertFalse(result["verification_pending"])

    def test_real_regression_fails_after_exactly_one_complete_verification(self):
        self.durations = lambda suite, role, index: 200 if suite == "cold" and role == "B" else 100
        self.assertFalse(self.run_experiment())
        self.assertEqual(len(self.calls), 16)

    def test_bad_measured_response_cannot_be_hidden_by_a_later_good_sample(self):
        for bad in (None, SimpleNamespace(status_code=200, text="Error: broken", headers={}),
                    SimpleNamespace(status_code=200, text="true\nfalse", headers={})):
            with self.subTest(bad=bad):
                runner = SQLTestRunner("http://localhost:1")
                good = SimpleNamespace(status_code=200, text="true", headers={})
                with mock.patch.dict(os.environ, {"PERF_FIXTURE_TRIAL": "1"}), \
                        mock.patch("run_sql_tests.PERF_TEST_ENABLED", True), \
                        mock.patch("run_sql_tests.PERF_AB_MODE", "record"), \
                        mock.patch("run_sql_tests.find_memcp_pid", return_value=None), \
                        mock.patch("run_sql_tests.requests.post", side_effect=[bad, good]) as post, \
                        redirect_stdout(io.StringIO()):
                    result = runner.run_test_case({
                        "name": "each sample must succeed", "scm": "true", "threshold_ms": 1,
                        "timing_samples": 2, "warmup": 0, "expect": {"rows": 1},
                        "noncritical": True,
                    }, "memcp-tests")
                self.assertFalse(result)
                self.assertEqual(post.call_count, 1)
                self.assertEqual(runner.failed_critical, 1)

    def test_execution_failure_never_enters_verification(self):
        self.exit_code = 1
        with self.assertRaisesRegex(RuntimeError, "fixture failed"):
            self.run_experiment()
        self.assertEqual(len(self.calls), 1)

    def test_missing_trusted_case_from_both_roles_is_rejected_before_execution(self):
        self.seed.write_text(json.dumps({"schema_version": 1, "rows": {
            performance_case_key(self.suites[0], "deleted"): 1000}}))
        with self.assertRaisesRegex(ValueError, "missing from both roles"):
            self.run_experiment()
        self.assertEqual(self.calls, [])

    def test_changed_rows_or_hash_or_sample_count_fail_without_verification(self):
        for field, bad in (("rows", 999), ("workload_sha256", "changed"), ("repetitions", 2)):
            with self.subTest(field=field), tempfile.TemporaryDirectory() as output_dir:
                self.output = Path(output_dir) / "result.json"
                self.calls = []
                def corrupt(result, index):
                    if index == 2:
                        result[performance_case_key(self.suites[0], "cold")][field] = bad
                self.mutate = corrupt
                with self.assertRaises(ValueError):
                    self.run_experiment()
                self.assertEqual(len(self.calls), 2)

    def test_verification_rechecks_initially_successful_sibling_cases(self):
        spec = json.loads(Path(self.suites[0]).read_text())
        spec["test_cases"].append(dict(spec["test_cases"][0], name="sibling"))
        Path(self.suites[0]).write_text(json.dumps(spec))
        self.durations = lambda suite, role, index: 500 if index == 2 else 100
        def regress_sibling(result, index):
            key = performance_case_key(self.suites[0], "sibling")
            if key in result:
                duration = 200 if index > 4 and self.calls[-1][1] == "B" else 100
                result[key].update(time_ms=duration, time_per_repetition_ms=duration,
                                   samples_ns=[duration * 1_000_000])
        self.mutate = regress_sibling
        self.assertFalse(self.run_experiment())
        summary = json.loads(self.output.read_text())
        self.assertTrue(summary[performance_case_key(self.suites[0], "cold")]["passed"])
        self.assertFalse(summary[performance_case_key(self.suites[0], "sibling")]["passed"])
        self.assertEqual(len(self.calls), 16)

    def test_missing_base_case_without_seed_override_is_rejected(self):
        path = self.base / "tests/performance/cold.yaml"
        path.parent.mkdir(parents=True)
        spec = json.loads(Path(self.suites[0]).read_text())
        spec["test_cases"].append(dict(spec["test_cases"][0], name="deleted"))
        path.write_text(json.dumps(spec))
        with self.assertRaisesRegex(ValueError, "base performance cases are missing"):
            self.run_experiment()
        self.assertEqual(self.calls, [])

    def test_sql_select_supports_declared_and_adaptive_sample_counts(self):
        path = Path(self.suites[0])
        spec = json.loads(path.read_text())
        case = spec["test_cases"][0]
        key = performance_case_key(str(path), "cold")
        for fixed, samples in ((True, 2), (False, 5)):
            with self.subTest(fixed=fixed):
                if fixed:
                    case["timing_samples"] = samples
                else:
                    case.pop("timing_samples", None)
                path.write_text(json.dumps(spec))
                result = {"schema_version": 1, key: {
                    "time_per_repetition_ms": 100, "rows": 1000,
                    "warmup": 0, "repetitions": samples, "max_regression_pct": 20,
                    "samples_ns": [100_000_000] * samples,
                    "workload_sha256": performance_case_fingerprint(case, spec["setup"], None),
                }}
                with mock.patch("run_sql_tests.PERF_REPEAT", 10):
                    self.assertEqual(validate_performance_fixture(result, str(path))[key]["repetitions"], samples)
                    for bad_count in (1, 11):
                        result[key]["repetitions"] = bad_count
                        result[key]["samples_ns"] = [100_000_000] * bad_count
                        with self.assertRaises(ValueError):
                            validate_performance_fixture(result, str(path))

    def test_wrong_warmup_result_stops_before_measuring(self):
        runner = SQLTestRunner("http://localhost:1")
        bad = SimpleNamespace(status_code=200, text="true\nfalse", headers={})
        with mock.patch.dict(os.environ, {"PERF_FIXTURE_TRIAL": "1"}), \
                mock.patch("run_sql_tests.PERF_TEST_ENABLED", True), \
                mock.patch("run_sql_tests.PERF_AB_MODE", "record"), \
                mock.patch("run_sql_tests.requests.post", return_value=bad) as post, \
                redirect_stdout(io.StringIO()):
            result = runner.run_test_case({
                "name": "warmup must be correct", "scm": "true", "threshold_ms": 1,
                "timing_samples": 2, "warmup": 1, "expect": {"rows": 1},
            }, "memcp-tests")
        self.assertFalse(result)
        self.assertEqual(post.call_count, 1)

    def test_incomplete_or_invalid_fixture_counts_are_rejected(self):
        for count in (True, 0, 2, 6, "7"):
            with self.subTest(count=count), self.assertRaises(ValueError):
                summarize_performance_fixtures([], count)
        with self.assertRaisesRegex(ValueError, "incomplete"):
            summarize_performance_fixtures([], 7)

    def test_existing_explicit_waiver_is_retained(self):
        key = performance_case_key(self.suites[0], "cold")
        self.durations = lambda suite, role, index: 200 if suite == "cold" and role == "B" else 100
        with mock.patch("run_sql_tests.load_perf_regression_waivers", return_value={key: "reviewed tradeoff"}):
            self.assertTrue(self.run_experiment())
        self.assertEqual(len(self.calls), 4)
        self.assertEqual(json.loads(self.output.read_text())[key]["waiver_reason"], "reviewed tradeoff")


if __name__ == "__main__":
    unittest.main()
