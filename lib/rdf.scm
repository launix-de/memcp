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

/* query plan cache for SPARQL */
(set sparql_queryplan_cache (newcachemap))

/*
this is how rdf works:
- every database may have a table rdf(s text, p text, o text)
- import formats are: xml, ttl
*/

(define rdf_sparql_json_term (lambda (value)
	(match value
		true (json_object "type" "literal" "value" "true"
			"datatype" "http://www.w3.org/2001/XMLSchema#boolean")
		false (json_object "type" "literal" "value" "false"
			"datatype" "http://www.w3.org/2001/XMLSchema#boolean")
		_ (if (rdf_is_blank value)
			(json_object "type" "bnode" "value"
				(replace (replace value "urn:uuid:" "") "_:" ""))
			(if (rdf_is_iri value)
				(json_object "type" "uri" "value" value)
				(if (int? value)
					(json_object "type" "literal" "value" (concat value)
						"datatype" "http://www.w3.org/2001/XMLSchema#integer")
				(if (number? value)
					(json_object "type" "literal" "value" (concat value)
						"datatype" "http://www.w3.org/2001/XMLSchema#decimal")
					(json_object "type" "literal" "value" value))))))
))
(define rdf_sparql_json_binding (lambda (row)
	(apply json_object (reduce_assoc row (lambda (acc key value)
		(if (nil? value) acc
			(append acc (substr (concat key) 1) (rdf_sparql_json_term value)))) '()))
))
(define rdf_sparql_query_vars (lambda (query)
	(match (ttl_header query)
		'("prefixes" definitions "rest" rest)
		(match (rdf_expand_select_star
			(rdf_resolve_prefixes (rdf_query (rdf_strip_leading_ws_comments rest)) definitions))
			'("select" cols "where" _conditions "group" _group "having" _having
				"order" _order "limit" _limit "offset" _offset "distinct" _distinct)
			(map (extract_assoc cols (lambda (title _expr) title)) (lambda (title)
				(substr (concat title) 1)))
			_ '())
		_ '())
))
(define rdf_sparql_results_json (lambda (rows ask_query vars)
	(if ask_query
		(json_object "head" (json_object) "boolean"
			(if (equal? rows '()) false (coalesceNil (get_assoc (car rows) "?ask") false)))
		(begin
			(json_object
				"head" (json_object "vars" (json_arrayagg_finalize vars))
				"results" (json_object "bindings"
					(json_arrayagg_finalize (map rows rdf_sparql_json_binding))))))
))
(define rdf_sparql_xml_escape (lambda (value)
	(htmlentities (concat value))
))
(define rdf_sparql_results_xml_term (lambda (value)
	(if (rdf_is_blank value)
		(concat "<bnode>" (rdf_sparql_xml_escape
			(replace (replace value "urn:uuid:" "") "_:" "")) "</bnode>")
		(if (rdf_is_iri value)
			(concat "<uri>" (rdf_sparql_xml_escape value) "</uri>")
			(concat "<literal"
				(if (or (int? value) (number? value))
					(concat " datatype=\"http://www.w3.org/2001/XMLSchema#"
						(if (int? value) "integer" "decimal") "\"")
					(if (or (equal? value true) (equal? value false))
						" datatype=\"http://www.w3.org/2001/XMLSchema#boolean\"" ""))
				">" (rdf_sparql_xml_escape value) "</literal>")))
))
(define rdf_sparql_results_xml (lambda (rows ask_query vars)
	(concat "<?xml version=\"1.0\" encoding=\"UTF-8\"?>"
		"<sparql xmlns=\"http://www.w3.org/2005/sparql-results#\"><head>"
		(reduce vars (lambda (acc var) (concat acc "<variable name=\""
			(rdf_sparql_xml_escape var) "\"/>")) "") "</head>"
		(if ask_query
			(concat "<boolean>" (if (equal? rows '()) "false"
				(if (coalesceNil (get_assoc (car rows) "?ask") false) "true" "false"))
				"</boolean>")
			(concat "<results>" (reduce rows (lambda (row_acc row)
				(concat row_acc "<result>" (reduce vars (lambda (binding_acc var)
					(begin
						(define value (get_assoc row (concat "?" var)))
						(if (nil? value) binding_acc
							(concat binding_acc "<binding name=\""
								(rdf_sparql_xml_escape var) "\">"
								(rdf_sparql_results_xml_term value) "</binding>")))) "")
					"</result>")) "") "</results>"))
		"</sparql>")
))
(define rdf_sparql_csv_field (lambda (value)
	(concat "\"" (replace (concat value) "\"" "\"\"") "\"")
))
(define rdf_sparql_delimited_results (lambda (rows vars separator variable_prefix)
	(concat
		(reduce vars (lambda (acc var) (concat acc
			(if (equal? acc "") "" separator) variable_prefix var)) "") "\n"
		(reduce rows (lambda (row_acc row) (concat row_acc
			(reduce vars (lambda (field_acc var) (begin
				(define value (get_assoc row (concat "?" var)))
				(concat field_acc (if (equal? field_acc "") "" separator)
					(if (nil? value) "" (rdf_sparql_csv_field value))))) "") "\n")) ""))
))

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
		(set pw (scan_lookup nil (table "system" "user") (list 372734710317056 (scan_boundary "equal" "username" 0 0 true true "" false) "password") (list (req "username"))))
		(if (and pw (equal? pw (password (req "password")))) (time (begin
			(define accept (coalesceNil (get_assoc (req "header") "Accept") ""))
			(define accept_lower (toLower accept))
			(define standard_json (rdf_contains accept_lower "application/sparql-results+json"))
			(define standard_xml (rdf_contains accept_lower "application/sparql-results+xml"))
			(define standard_csv (rdf_contains accept_lower "text/csv"))
			(define standard_tsv (rdf_contains accept_lower "text/tab-separated-values"))
			(define buffered (or standard_json (or standard_xml (or standard_csv standard_tsv))))
			((res "header") "Content-Type"
				(if standard_json "application/sparql-results+json; charset=utf-8"
				(if standard_xml "application/sparql-results+xml; charset=utf-8"
				(if standard_csv "text/csv; charset=utf-8"
				(if standard_tsv "text/tab-separated-values; charset=utf-8"
					"application/x-ndjson; charset=utf-8")))))
			((res "status") 200)
			(define row_store (if buffered (newsession) nil))
			(define resultrow (if buffered
				(lambda (row) (begin
					(row_store (concat "row:" (coalesceNil (row_store "count") 0)) row)
					(row_store "count" (+ (coalesceNil (row_store "count") 0) 1))))
				(res "jsonl")))
			(define session (req "__session"))
			(define session_state (req "__session_state"))
			(define query_seq (req "__query_seq"))
			(session "username" (req "username"))
			(session "schema" schema)
			/* Match SQL prepared-query semantics for URL parameters and execute the RDF
			plan through the same closed, session-bound query-plan infrastructure. */
			(extract_assoc (req "query") (lambda (k v) (session k v)))
			(with_autocommit session session_state query_seq query (lambda (tx) (begin
				(define formula (cached_parse sparql_queryplan_cache (list parse_sparql)
					schema query (lambda (_schema _table _write) true)
					(req "username") session false tx))
				(sql_execute_formula session tx formula resultrow (lambda (_fields) true))
			)))
			(if buffered
				(begin
					(define rows (map (produceN (coalesceNil (row_store "count") 0))
						(lambda (idx) (row_store (concat "row:" idx)))))
					(define ask_query (regexp_test (toUpper query) "(?:^|[\\s}])ASK(?:[\\s{]|$)"))
					(define vars (rdf_sparql_query_vars query))
					((res "print")
						(if standard_json (json_encode (rdf_sparql_results_json rows ask_query vars))
						(if standard_xml (rdf_sparql_results_xml rows ask_query vars)
						(if standard_csv (rdf_sparql_delimited_results rows vars "," "")
							(rdf_sparql_delimited_results rows vars "\t" "?"))))))
				nil)
		) query) (begin
				((res "header") "Content-Type" "text/plain")
				((res "header") "WWW-Authenticate" "Basic realm=\"authorization required\"")
				((res "status") 401)
				((res "print") "Unauthorized")
		))
	)))
	(define handle_ttl_load (lambda (req res schema ttl_data) (begin
		/* check for password */
		(set pw (scan_lookup nil (table "system" "user") (list 372734710317056 (scan_boundary "equal" "username" 0 0 true true "" false) "password") (list (req "username"))))
		(if (and pw (equal? pw (password (req "password")))) (begin
			((res "header") "Content-Type" "text/plain")
			((res "status") 200)
			/*(print "Loading TTL data into: " schema)*/
			/* ensure the shared RDF graph storage and its set-semantics key exist */
			(rdf_ensure_table schema)
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
			(regex "^/rdf/([^/]+)/load_ttl$" url schema) (begin
				(set ttl_data ((req "body")))
				(handle_ttl_load req res schema ttl_data)
			)
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
				((outer 1 old_handler) req res))
		)
	))
))
