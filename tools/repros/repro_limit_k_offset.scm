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

/* Minimal repro for BTW2025 §4.4:
correlated scalar subqueries with ORDER BY + LIMIT k/OFFSET o must be evaluated
per outer binding, not by a global LIMIT. */

(define repro_limit_k_offset_setup (list
	"DROP TABLE IF EXISTS lko_emp"
	"DROP TABLE IF EXISTS lko_dept"
	"CREATE TABLE lko_dept (did INT PRIMARY KEY, dname TEXT)"
	"CREATE TABLE lko_emp (eid INT PRIMARY KEY, did INT, name TEXT, band TEXT, salary INT)"
	"INSERT INTO lko_dept VALUES (1, 'Engineering'), (2, 'Sales')"
	"INSERT INTO lko_emp VALUES (1, 1, 'Alice', 'A', 100), (2, 1, 'Bob', 'B', 90), (3, 1, 'Cara', 'A', 80), (4, 1, 'Dana', 'C', 70), (5, 1, 'Erin', 'B', 60), (6, 1, 'Fay', 'D', 50), (7, 2, 'Gabe', 'A', 95), (8, 2, 'Hank', 'B', 85), (9, 2, 'Iris', 'A', 75), (10, 2, 'Jill', 'C', 65), (11, 2, 'Karl', 'B', 55), (12, 2, 'Liam', 'D', 45)"
))

(define repro_limit_k_offset_queries (list
	"SELECT d.did, (SELECT e.name FROM lko_emp e WHERE e.did = d.did ORDER BY e.salary DESC LIMIT 3 OFFSET 1) AS shifted_earner FROM lko_dept d ORDER BY d.did"
	"SELECT d.did, (SELECT e.name FROM lko_emp e WHERE e.did = d.did ORDER BY e.salary DESC OFFSET 5) AS offset_only_earner FROM lko_dept d ORDER BY d.did"
	"SELECT d.did, (SELECT e.name FROM lko_emp e WHERE e.did = d.did ORDER BY e.salary DESC LIMIT 2) AS topk_earner FROM lko_dept d ORDER BY d.did"
	"SELECT d.did, (SELECT e.band FROM lko_emp e WHERE e.did = d.did GROUP BY e.band ORDER BY MAX(e.salary) DESC LIMIT 3) AS grouped_band FROM lko_dept d ORDER BY d.did"
	"SELECT d.did, (SELECT e.name FROM lko_emp e WHERE e.did = d.did LIMIT 5) AS synthesized_order_earner FROM lko_dept d ORDER BY d.did"
))
