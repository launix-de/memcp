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
	(sqlx "CREATE TABLE era_box (ID INT PRIMARY KEY, fopuser TEXT, zuweisungPublic INT)")
	(sqlx "CREATE TABLE era_share (ID INT PRIMARY KEY, mailbox INT, fopuser TEXT)")
	(sqlx "CREATE TABLE era_dir (ID INT PRIMARY KEY, mailbox INT)")
	(sqlx "CREATE TABLE era_email (ID INT PRIMARY KEY, directory INT, seen INT)")
	(sqlx "INSERT INTO era_box VALUES (1, 'alice', 0), (2, 'bob', 1)")
	(sqlx "INSERT INTO era_share VALUES (10, 1, 'carol')")
	(sqlx "INSERT INTO era_dir VALUES (100, 1), (200, 2)")
	(sqlx "INSERT INTO era_email VALUES (1000, 100, 0), (1001, 100, 1), (1002, 200, 0)")
	(print "COUNT*=" (sqlx "SELECT COUNT(*) AS v FROM era_email e WHERE NOT(e.seen) AND COALESCE((SELECT COALESCE((SELECT (CASE WHEN FALSE THEN TRUE ELSE NOT ((b.fopuser <> 'alice') AND (NOT (EXISTS (SELECT TRUE FROM era_share s WHERE s.mailbox = b.ID AND s.fopuser = 'alice' LIMIT 1)))) END) FROM era_box b WHERE b.ID = d.mailbox LIMIT 1), FALSE) FROM era_dir d WHERE d.ID = e.directory LIMIT 1), FALSE)"))
	(print "COUNT1=" (sqlx "SELECT COUNT(1) AS v FROM era_email e WHERE NOT(e.seen) AND COALESCE((SELECT COALESCE((SELECT (CASE WHEN FALSE THEN TRUE ELSE NOT ((b.fopuser <> 'alice') AND (NOT (EXISTS (SELECT TRUE FROM era_share s WHERE s.mailbox = b.ID AND s.fopuser = 'alice' LIMIT 1)))) END) FROM era_box b WHERE b.ID = d.mailbox LIMIT 1), FALSE) FROM era_dir d WHERE d.ID = e.directory LIMIT 1), FALSE)"))
	(print "SUM1=" (sqlx "SELECT SUM(1) AS v FROM era_email e WHERE NOT(e.seen) AND COALESCE((SELECT COALESCE((SELECT (CASE WHEN FALSE THEN TRUE ELSE NOT ((b.fopuser <> 'alice') AND (NOT (EXISTS (SELECT TRUE FROM era_share s WHERE s.mailbox = b.ID AND s.fopuser = 'alice' LIMIT 1)))) END) FROM era_box b WHERE b.ID = d.mailbox LIMIT 1), FALSE) FROM era_dir d WHERE d.ID = e.directory LIMIT 1), FALSE)"))
	(dropdatabase "memcp-tests")
)))
