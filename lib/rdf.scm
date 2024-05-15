/*
Copyright (C) 2024  Carl-Philip Hänsch

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

(import "rdf-parser.scm")

/*
this is how rdf works:
 - every database may have a table rdf(s text, p text, o text)
 - import formats are: xml, ttl
*/

(define handler_404 (lambda (req res) (begin
	(print "request " req)
	((res "header") "Content-Type" "text/plain")
	((res "status") 404)
	((res "println") "404 not found")
)))

/* http hook for handling SparQL */
(define http_handler (begin
	(set old_handler (coalesce http_handler handler_404))
	(define handle_query (lambda (req res schema query) (begin
		/* check for password */
		(set pw (scan "system" "user" '("username") (lambda (username) (equal? username (req "username"))) '("password") (lambda (password) password) (lambda (a b) b) nil))
		(if (and pw (equal? pw (password (req "password")))) (begin
			((res "header") "Content-Type" "text/plain")
			((res "status") 200)
			(print "RDF query: " query)
			(define formula (parse_sparql schema query))
			(define resultrow (res "jsonl"))
			/*(define session (newsession))*/

			(eval formula)
		) (begin
			((res "header") "Content-Type" "text/plain")
			((res "header") "WWW-Authenticate" "Basic realm=\"authorization required\"")
			((res "status") 401)
			((res "print") "Unauthorized")
		))
	)))
	old_handler old_handler /* workaround for optimizer bug */
	(lambda (req res) (begin
			    (print "old handler" old_handler)
		/* hooked our additional paths to it */
		(match (req "path")
			(regex "^/rdf/([^/]+)$" url schema) (begin
				(set query ((req "body")))
				(handle_query req res schema query)
			)
			(regex "^/rdf/([^/]+)/(.*)$" url schema query_un) (begin
				(set query (urldecode query_un))
				(handle_query req res schema query)
			)
			/* default */
			(!begin
			    (print "handler" old_handler)
			((outer old_handler) req res))
			)
	))
))

