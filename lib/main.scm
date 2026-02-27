/*
Copyright (C) 2023 - 2026  Carl-Philip Hänsch

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

(print "")
(print "Welcome to memcp")
(print "")
(import "test.scm")

(set static_files (serveStatic "../assets"))

/* this can be overhooked */
(define http_handler (lambda (req res) (begin
	(print "request " req)
	(if (equal? (req "path") "/") (begin
		((res "header") "Location" "/dashboard")
		((res "status") 301)
	) (static_files req res))
	/*
	((res "header") "Content-Type" "text/plain")
	((res "status") 404)
	((res "println") "404 not found")
	*/
)))

/* global service registry: each module registers itself as (service_registry name (list port route protocols)) */
(set service_registry (coalesce service_registry (newsession)))

(import "sql.scm")
(import "dashboard.scm")
(import "rdf.scm")

/* read ports from command line arguments or environment */
(if (not (arg "disable-api" false)) (begin
	(set port (arg "api-port" (env "PORT" "4321")))
	(serve port (lambda (req res) (http_handler req res)))
	(service_registry "HTTP Server" (list port "/" "HTTP"))
	(print "listening on http://localhost:" port)
))
