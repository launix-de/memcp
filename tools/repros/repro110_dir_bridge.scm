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
	(sqlx "INSERT INTO era_box VALUES (1, 'alice', 0), (2, 'bob', 1)")
	(sqlx "INSERT INTO era_share VALUES (10, 1, 'carol')")
	(sqlx "INSERT INTO era_dir VALUES (100, 1), (200, 2)")
	(print "BOX_OK=" (sqlx "SELECT b.ID, NOT ((b.fopuser <> 'alice') AND (NOT (EXISTS (SELECT TRUE FROM era_share s WHERE s.mailbox = b.ID AND s.fopuser = 'alice' LIMIT 1)))) AS ok FROM era_box b ORDER BY b.ID"))
	(print "DIR_OK_SCALAR=" (sqlx "SELECT d.ID, (SELECT NOT ((b.fopuser <> 'alice') AND (NOT (EXISTS (SELECT TRUE FROM era_share s WHERE s.mailbox = b.ID AND s.fopuser = 'alice' LIMIT 1)))) FROM era_box b WHERE b.ID = d.mailbox LIMIT 1) AS ok FROM era_dir d ORDER BY d.ID"))
	(print "DIR_OK_COALESCE=" (sqlx "SELECT d.ID, COALESCE((SELECT NOT ((b.fopuser <> 'alice') AND (NOT (EXISTS (SELECT TRUE FROM era_share s WHERE s.mailbox = b.ID AND s.fopuser = 'alice' LIMIT 1)))) FROM era_box b WHERE b.ID = d.mailbox LIMIT 1), FALSE) AS ok FROM era_dir d ORDER BY d.ID"))
	(dropdatabase "memcp-tests")
)))
