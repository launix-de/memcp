/*
Copyright (C) 2023  Carl-Philip Hänsch

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

(import "sql-parser.scm")
(import "queryplan.scm")

/* create user tables */
(print "Initializing SQL frontend")
(if (has? (show) "system") true (begin
	(print "creating database system")
	(createdatabase "system")
))
(if (has? (show "system") "user") true (begin
	(print "creating table system.user")
	(eval (parse_sql "system" "CREATE TABLE `user`(id int, username text, password text) ENGINE=SAFE"))
	(insert "system" "user" '("id" "username" "password") '('(1 "root" (password "admin"))))
))

(set globalvars '("lower_case_table_names" 0))

/* http hook for handling SQL */
(define http_handler (begin
	(set old_handler http_handler)
	(lambda (req res) (begin
		/* hooked our additional paths to it */
		(match (req "path")
			(regex "^/sql/([^/]+)/(.*)$" url schema query) (begin
				/* check for password */
				(set pw (scan "system" "user" '("username") (lambda (username) (equal? username (req "username"))) '("password") (lambda (password) password) (lambda (a b) b) nil))
				(if (and pw (equal? pw (password (req "password")))) (begin
					((res "header") "Content-Type" "text/plain")
					((res "status") 200)
					(define formula (parse_sql schema query))
					(define resultrow (res "jsonl"))
					(print "received query: " query)
					(define session (newsession))
					(eval formula)
				) (begin
					((res "header") "Content-Type" "text/plain")
					((res "header") "WWW-Authenticate" "Basic realm=\"authorization required\"")
					((res "status") 401)
					((res "print") "Unauthorized")
				))
			)
			/* default */
			(old_handler req res))
	))
))

/* dedicated mysql protocol listening at port 3307 */
(try (lambda () (begin
	(mysql 3307
		(lambda (username_) (scan "system" "user" '("username") (lambda (username) (equal? username username_)) '("password") (lambda (password) password) (lambda (a b) b) nil)) /* auth: load pw hash from system.user */
		(lambda (username schema) (list? (show schema))) /* switch schema (TODO check grants; in the moment, only the existence of the database is checked) */
		(lambda (schema sql resultrow_sql session) (begin /* sql */
			(print "received query: " sql)
			(define formula (parse_sql schema sql))
			(define resultrow resultrow_sql)
			(eval (source "SQL Query" 1 1 formula))
		))
	)
	(print "MySQL server listening on port 3307 (connect with `mysql -P 3307 -u root -p` using password 'admin')")
)) print)
