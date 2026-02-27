/*
Copyright (C) 2024 - 2026  Carl-Philip Hänsch

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
	/*(print "request " req)*/
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
		(if (and pw (equal? pw (password (req "password")))) (time (begin
			((res "header") "Content-Type" "text/plain")
			((res "status") 200)
			/*(print "RDF query: " query)*/
			(define formula (parse_sparql schema query))
			(define resultrow (res "jsonl"))

			(eval formula)
		) query) (begin
				((res "header") "Content-Type" "text/plain")
				((res "header") "WWW-Authenticate" "Basic realm=\"authorization required\"")
				((res "status") 401)
				((res "print") "Unauthorized")
		))
	)))
	(define handle_ttl_load (lambda (req res schema ttl_data) (begin
		/* check for password */
		(set pw (scan "system" "user" '("username") (lambda (username) (equal? username (req "username"))) '("password") (lambda (password) password) (lambda (a b) b) nil))
		(if (and pw (equal? pw (password (req "password")))) (begin
			((res "header") "Content-Type" "text/plain")
			((res "status") 200)
			/*(print "Loading TTL data into: " schema)*/
			/* ensure rdf table exists */
			(eval (parse_sql schema "CREATE TABLE IF NOT EXISTS rdf (s TEXT, p TEXT, o TEXT)" (lambda (schema table write) true)))
			/* load the TTL data */
			(load_ttl schema ttl_data)
			((res "println") "TTL data loaded successfully")
		) (begin
				((res "header") "Content-Type" "text/plain")
				((res "header") "WWW-Authenticate" "Basic realm=\"authorization required\"")
				((res "status") 401)
				((res "print") "Unauthorized")
		))
	)))
	/* register RDF/SPARQL frontend in service registry */
	(if (not (nil? service_registry)) (begin
		(service_registry "RDF/SPARQL Frontend" (list (arg "api-port" (env "PORT" "4321")) "/rdf/[database]" "POST, SPARQL"))
	))
	old_handler old_handler /* workaround for optimizer bug */
	(lambda (req res) (begin
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
			(regex "^/rdf/([^/]+)/load_ttl$" url schema) (begin
				(set ttl_data ((req "body")))
				(handle_ttl_load req res schema ttl_data)
			)
			/* default */
			(!begin
				((outer old_handler) req res))
		)
	))
))
