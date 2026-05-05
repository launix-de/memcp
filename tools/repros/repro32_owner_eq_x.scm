/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

/* Minimal repro for correlated scalar subselects that reference a visible
derived-table alias column. The inner COUNT must correlate against the physical
outer source for `x`, not fail on the visible alias binding. */

(define repro32_owner_eq_x_setup (list
	"DROP TABLE IF EXISTS t2"
	"DROP TABLE IF EXISTS t3"
	"CREATE TABLE t2 (ID INT PRIMARY KEY, owner INT)"
	"CREATE TABLE t3 (ID INT PRIMARY KEY)"
	"INSERT INTO t3 VALUES (1), (2)"
	"INSERT INTO t2 VALUES (1, 1), (2, 1), (3, 2)"
))

(define repro32_owner_eq_x_queries (list
	"SELECT x, (SELECT COUNT(1) FROM t2 WHERE owner = x) AS cnt FROM (SELECT id AS x FROM t3) AS d ORDER BY x"
))
