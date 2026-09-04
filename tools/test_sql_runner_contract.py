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
    is_error_response,
    initialize_performance_recording,
    load_performance_scale,
    observe_atomic_json,
    performance_ab_threshold_ms,
    performance_sample_ns,
    performance_case_fingerprint,
    performance_case_key,
    performance_architecture,
    performance_regression_pct,
    performance_scale_from_samples,
    planner_time_limit_with_tolerance_ms,
    publish_performance_scale,
    request_shared_supervisor_restart,
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


class PerformanceScaleContractTest(unittest.TestCase):
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
            self.assertFalse(adaptive_measurement_complete(4, 1_000_000_000))
            self.assertFalse(adaptive_measurement_complete(100, 249_999_999))
            self.assertTrue(adaptive_measurement_complete(5, 250_000_000))

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

if __name__ == "__main__":
    unittest.main()
