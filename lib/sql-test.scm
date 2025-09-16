/*
Copyright (C) 2025  Carl-Philip Hänsch

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

/* SQL Engine Test Suite - contained in its own environment */
((lambda () (begin
	(print "performing SQL engine tests ...")

	(set teststat (newsession))
	(teststat "count" 0)
	(teststat "success" 0)
	(define assert (lambda (val1 val2 errormsg) (begin
		(teststat "count" (+ (teststat "count") 1))
		(if (equal? val1 val2) (teststat "success" (+ (teststat "success") 1)) (print "failed test "(teststat "count")": " errormsg))
	)))

	/* Clean up any existing test database and create fresh one */
	(try (lambda () (dropdatabase "memcp-tests")) (lambda (e) nil))
	(createdatabase "memcp-tests" true)

	/* Helper function to execute SQL and return result rows */
	(define sql-test-exec (lambda (query) (begin
		(set query-results (newsession))
		(query-results "rows" '())
		(define resultrow (lambda (row) (begin
			(query-results "rows" (append (query-results "rows") (list row)))
		)))
		(eval (parse_sql "memcp-tests" query))
		(query-results "rows")
	)))

	/* Create test tables and run simple tests */
	(sql-test-exec "CREATE TABLE test_users (id INT PRIMARY KEY, name VARCHAR(50))")
	(sql-test-exec "INSERT INTO test_users (id, name) VALUES (1, 'Alice')")
	(sql-test-exec "INSERT INTO test_users (id, name) VALUES (2, 'Bob')")

	(define result1 (sql-test-exec "SELECT * FROM test_users"))
	(assert (equal? (count result1) 2) true "SELECT should return 2 rows")

	(define result2 (sql-test-exec "SELECT COUNT(*) FROM test_users"))
	(assert (equal? (count result2) 1) true "SELECT COUNT(*) should return 1 row")

	/* Basic parsing tests - just verify the SQL parses correctly without executing */
	(define allow (lambda (schema table write) true))
	(assert (list? (parse_sql "system" "SELECT 1" allow)) true "Simple SELECT should parse")
	(assert (list? (parse_sql "system" "SELECT * FROM user" allow)) true "SELECT * should parse")
	(assert (list? (parse_sql "system" "INSERT INTO user VALUES (1, 'test', 'pass')" allow)) true "INSERT should parse")
	(assert (list? (parse_sql "system" "UPDATE user SET username = 'newname' WHERE id = 1" allow)) true "UPDATE should parse")
	(assert (list? (parse_sql "system" "DELETE FROM user WHERE id = 1" allow)) true "DELETE should parse")

	(print "SQL parsing and execution tests completed successfully")

	/* Clean up test database */
	(dropdatabase "memcp-tests")

	(print "finished SQL engine tests")
	(print "test result: " (teststat "success") "/" (teststat "count"))
	(if (< (teststat "success") (teststat "count")) (begin
		(print "")
		(print "---- !!! some SQL test cases have failed !!! ----")
		(print "")
		(print " SQL engine may have issues")
		(error "SQL tests failed")
	) (print "all SQL tests succeeded."))
	(print "")
)))
