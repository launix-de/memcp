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
	(insert "system" "user" '("id" 1 "username" "root" "password" (password "admin")))
))

/* http hook for handling SQL */
(define http_handler (begin
	(set old_handler http_handler)
	(lambda (req res) (begin
		/* hooked our additional paths to it */
		(match (req "path")
			(regex "^/sql/([^/]+)/(.*)$" url schema query) (begin
				/* check for password */
				(set pw (scan "system" "user" (lambda (username) (equal? username (req "username"))) (lambda (password) password) (lambda (a b) b) nil))
				(if (and pw (equal? pw (password (req "password")))) (begin
					((res "header") "Content-Type" "text/plain")
					((res "status") 200)
					(define formula (parse_sql schema query))
					(define resultrow (res "jsonl"))
					(print "received query: " query)
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
(mysql 3307
	(lambda (username_) (scan "system" "user" (lambda (username) (equal? username username_)) (lambda (password) password) (lambda (a b) b) nil)) /* auth: load pw hash from system.user */
	(lambda (schema) true) /* switch schema */
	(lambda (schema sql resultrow_sql) (begin /* sql */
		(print "received query: " sql)
		(define formula (parse_sql schema sql))
		(define resultrow resultrow_sql)
		(eval formula)
	))
)
(print "MySQL server listening on port 3307 (connect with `mysql -P 3307 -u root -p` using password 'admin')")
