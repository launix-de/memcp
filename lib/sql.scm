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

/* helper functions to work with maps */
(define map_assoc (lambda (columns fn)
	(match columns
		(cons colid (cons colvalue rest)) (cons colid (cons (fn colvalue) (map_assoc rest fn)))
		'()
	)
))

(define extract_assoc (lambda (columns fn)
	(match columns
		(cons colid (cons colvalue rest)) (cons (fn colvalue) (extract_assoc rest fn))
		'()
	)
))

(import "sql-parser.scm")
(import "queryplan.scm")

/* TODO: session state handling -> which schema */
(createdatabase "test")
(createtable "test" "foo" '('("bar" "int" '() "")))
(insert "test" "foo" '("bar" 12))
(insert "test" "foo" '("bar" 44))
(set schema "test")

/* http hook for handling SQL */
(define http_handler (begin
	(set old_handler http_handler)
	(lambda (req res) (begin
		/* hooked our additional paths to it */
		(match (req "path")
			(concat "/sql/" rest) (begin
				((res "status") 200)
				((res "header") "Content-Type" "text/plain")
				(define formula (parse_sql rest))
				(define resultrow (res "println")) /* TODO: to JSON */
				(print "received query: " rest)
				(eval formula)
			)
			/* default */
			(old_handler req res))
	))
))

/* dedicated mysql protocol listening at port 3307 */
(mysql 3307
	(lambda (username) "TODO: return pwhash") /* auth */
	(lambda (schema) true) /* switch schema */
	(lambda (sql resultrow_sql) (begin /* sql */
		(print "received query: " sql)
		(define formula (parse_sql sql))
		(define resultrow resultrow_sql)
		(eval formula)
	))
)
(print "MySQL server listening on port 3307 (connect with mysql -P 3307 -u user -p)")
