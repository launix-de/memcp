/*
Copyright (C) 2026  Carl-Philip Haensch

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

((lambda () (begin
	(try (lambda () (dropdatabase "memcp-tests")) (lambda (_e) nil))
	(createdatabase "memcp-tests" true)
	(define sqlx (lambda (query) (begin
		(define query-results (newsession))
		(query-results "rows" '())
		(define resultrow (lambda (row) (begin
			(query-results "rows" (append (query-results "rows") (list row)))
		)))
		(context (lambda () (begin
			(eval (parse_sql "memcp-tests" query (lambda (_schema _tblname _write) true)))
			(query-results "rows")
		)))
	)))
	(sqlx "CREATE TABLE dtef_users (ID INT PRIMARY KEY, name TEXT)")
	(sqlx "CREATE TABLE dtef_team (ID INT PRIMARY KEY, list_id INT, member INT)")
	(sqlx "CREATE TABLE dtef_items (ID INT PRIMARY KEY, team_id INT, val INT)")
	(sqlx "INSERT INTO dtef_users VALUES (1, 'Alice'), (2, 'Bob')")
	(sqlx "INSERT INTO dtef_team VALUES (1, 10, 1), (2, 20, 2)")
	(sqlx "INSERT INTO dtef_items VALUES (1, 10, 100), (2, 20, 200), (3, 30, 300)")
	(print "EXPLAIN-IR="
		(sqlx "EXPLAIN IR SELECT SUM(CASE WHEN EXISTS (SELECT TRUE FROM dtef_team tm WHERE tm.list_id = t.team_id AND (SELECT u.name FROM dtef_users u WHERE u.ID = tm.member LIMIT 1) IS NOT NULL LIMIT 1) THEN 1 ELSE 0 END) AS has_team FROM dtef_items t"))
	(print "QUERY="
		(sqlx "SELECT SUM(CASE WHEN EXISTS (SELECT TRUE FROM dtef_team tm WHERE tm.list_id = t.team_id AND (SELECT u.name FROM dtef_users u WHERE u.ID = tm.member LIMIT 1) IS NOT NULL LIMIT 1) THEN 1 ELSE 0 END) AS has_team FROM dtef_items t"))
	(dropdatabase "memcp-tests")
)))
