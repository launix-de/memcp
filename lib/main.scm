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

/* Install the application below the SQL/dashboard/RDF wrappers. CLI paths
resolve against the process working directory, not this module's lib/ folder. */
(define cli_php_root (arg "serve" nil))
(define cli_php_handler (if (nil? cli_php_root) nil (begin
	(if (or (not (string? cli_php_root)) (equal? cli_php_root ""))
		(error "--serve requires a directory path"))
	(servePHP (if (equal? (substr cli_php_root 0 1) "/")
		cli_php_root (concat __CWD__ "/" cli_php_root)) "" "index.php")
)))

/* this can be overhooked */
(define http_handler (lambda (req res) (begin
	(print "request " req)
	(if (not (nil? cli_php_handler)) (cli_php_handler req res)
		(if (equal? (req "path") "/") (begin
			((res "header") "Location" "/dashboard")
			((res "status") 301)
		) (static_files req res)))
	/*
	((res "header") "Content-Type" "text/plain")
	((res "status") 404)
	((res "println") "404 not found")
	*/
)))

/* global service registry: each module registers itself as (service_registry name (list port route protocols)) */
(set service_registry (coalesce service_registry (newsession)))

/* Persistent trigger languages are process-local frontend hooks. Scheme is
available without loading an SQL module; other frontends can register their
own compiler under any language name. */
(registertriggerlanguage "scheme" (lambda (source context)
	(list (quote deferred_trigger)
		(scheme source (concat "trigger:" (context "schema") "." (context "table") ":" (context "name"))))))

(import "sql.scm")
(import "storage-failure-hooks.scm")
(import "dashboard.scm")
(import "rdf.scm")

/* read ports from command line arguments or environment */
(if (not (arg "disable-api" false)) (begin
	(set port (arg "api-port" (env "PORT" "4321")))
	(serve port (lambda (req res) (http_handler req res)))
	(service_registry "HTTP Server" (list port "/" "HTTP"))
	(print "listening on http://localhost:" port)
))
