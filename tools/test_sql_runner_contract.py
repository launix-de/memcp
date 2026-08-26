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
    is_error_response,
    load_performance_scale,
    observe_atomic_json,
    performance_architecture,
    performance_scale_from_samples,
    planner_time_limit_with_tolerance_ms,
    publish_performance_scale,
    resolve_timing_samples,
    scaled_compile_time_limit_ms,
    scaled_wall_clock_limit_ms,
    suite_execution_mode,
)


class PerformanceScaleContractTest(unittest.TestCase):
    def test_timing_samples_default_and_explicit_median(self) -> None:
        self.assertEqual(resolve_timing_samples({}, False), 1)
        self.assertEqual(resolve_timing_samples({}, True), 5)
        self.assertEqual(resolve_timing_samples({"timing_samples": 3}, False), 3)

    def test_timing_samples_rejects_ambiguous_or_empty_samples(self) -> None:
        for invalid in (True, 0, 2, 2.5, "3"):
            with self.subTest(invalid=invalid):
                with self.assertRaisesRegex(ValueError, "positive odd integer"):
                    resolve_timing_samples({"timing_samples": invalid}, False)

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
    def test_isolated_restart_suite_owns_a_managed_server(self) -> None:
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
            self.assertEqual(suite_execution_mode(str(spec)), "managed_subprocess")

    def test_shared_restart_suite_keeps_the_direct_managed_server(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            spec = Path(tmp) / "restart.yaml"
            spec.write_text(
                "test_cases:\n"
                "  - name: restart\n"
                "    sql: SHUTDOWN\n",
                encoding="utf-8",
            )
            self.assertEqual(suite_execution_mode(str(spec)), "direct")

if __name__ == "__main__":
    unittest.main()
