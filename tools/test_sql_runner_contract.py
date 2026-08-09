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
import sys
import unittest


sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from run_sql_tests import is_error_response  # noqa: E402


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


if __name__ == "__main__":
    unittest.main()
