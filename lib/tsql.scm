/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

(import "tsql-parser.scm")
(define tsql_queryplan_cache (newcachemap))
(define tsql_statement_text (lambda (text) (begin
	(define input (strtrim text))
	(define length (strlen input))
	(if (and (> length 0) (equal? (substr input (- length 1) 1) ";"))
		(substr input 0 (- length 1)) input))))

/* Bind at the request boundary, not in storage or row loops. NULL is a supplied
value; the wire descriptor and SQL declaration own the type independently. */
(define tds_bind_parameters (lambda (session declarations parameters) (begin
	(if (not (equal? (count declarations) (count parameters)))
		(error "T-SQL parameter count differs from declarations") true)
	(define names (map declarations car))
	(if (not (equal? (count names) (count (merge_unique (list names)))))
		(error "duplicate T-SQL parameter declaration") true)
	(define named (merge (map (produceN (count parameters)) (lambda (i) (begin
		(define parameter (nth parameters i))
		(define name (parameter "name"))
		(define name (if (equal? name "") (car (nth declarations i))
			(toLower (if (equal? (substr name 0 1) "@") (substr name 1) name))))
		(if (or (not (has? names name)) (not (equal? (parameter "flags") 0)))
			(error "unknown or unsupported output parameter: @" name) true)
		(list name parameter))))))
	(if (not (equal? (count parameters) (count (merge_unique (list (extract_assoc named (lambda (name _value) name)))))))
		(error "duplicate T-SQL parameter value") true)
	(define values (map declarations (lambda (declaration) (begin
		(define name (car declaration))
		(define contract (cadr declaration))
		(define parameter (named name))
		(define wire (parameter "type"))
		(if (not (has? '(31 38 104 109 167 175 231 239) (wire "kind")))
			(error "unsupported T-SQL parameter wire type: " (wire "kind")) true)
		(define converted (eval (sql_type_lower_annotations
			(tsql_cast_expr (list (quote quote) (parameter "value")) (car contract) (cdr contract)))))
		(list (concat "__tsql_parameter:" name) converted)))))
	/* All declarations and values are validated before installing this request. */
	(map values (lambda (item) (session (car item) (cadr item))))
	(session "__tsql_parameter_declarations" declarations)
	true)))

/* Description parses and resolves SQL contracts without running the compiled
statement. SELECT bypasses physical planning; mutations are refused here until
the parser exposes a pure metadata path for their result contract. */
(define tds_describe (lambda (schema text declarations session) (begin
	(define planning (newsession))
	(planning "__tsql_parameter_declarations" (tds_bind_declarations schema declarations))
	(planning "__tsql_describe" true)
	(match (parse_tsql schema (tsql_statement_text text) (sql_policy (coalesceNil (session "username") "root")) planning nil)
		((symbol quote) fields) fields
		_ (error "unsupported T-SQL describe shape")))))

/* A separate frontend entrypoint keeps dialect decisions out of existing SQL
execution paths. The cache, transaction owner and compiled evaluator are shared. */
(define tsql_execute_statement (lambda (tx schema text session resultrow resultfields) (begin
	(define input (tsql_statement_text text))
	(define username (coalesce (session "username") "root"))
	(define declarations (coalesceNil (session "__tsql_parameter_declarations") (list)))
	/* Cache identity includes the complete immutable declaration contract. Values
	remain invocation inputs; differently typed calls cannot reuse wrong metadata. */
	(define cache_scope (json_encode (list username declarations)))
	(define parser (lambda (database text policy planning tx) (begin
		(planning "__tsql_parameter_declarations" declarations)
		(parse_tsql database text policy planning tx))))
	(define formula (cached_parse tsql_queryplan_cache (list parser) schema input
		(list (quote sql-policy-for) username) cache_scope session false tx))
	(sql_execute_formula session tx formula resultrow resultfields))))

(define tsql_http_query (lambda (req res schema) (begin
	(define stored (mysql_auth (req "username")))
	(if (and stored (equal? stored (password (req "password"))))
		(begin
			(define text ((req "body")))
			(try (lambda () (time (begin
				((res "header") "Content-Type" "text/event-stream; charset=utf-8")
				(define session (req "__session"))
				(extract_assoc (req "query") (lambda (key value) (session key value)))
				/* Authentication and the route own identity and database selection;
				query options may supply values, never replace either authority. */
				(session "username" (req "username"))
				(session "schema" schema)
				(define output (newsession))
				(define result (with_autocommit session (req "__session_state") (req "__query_seq") text
					(lambda (tx) (tsql_execute_statement tx schema text session
						(lambda (values) (begin (output "rows" true) ((res "jsonl") values)))
						(lambda args true)))))
				(if (and (not (output "rows")) (number? result))
					((res "jsonl") (list "affected_rows" result)) true)
			) text)) (lambda (err) (begin
					(error_log (concat err) schema (req "username") text)
					((res "header") "Content-Type" "text/plain")
					((res "status") 500)
					((res "print") "SQL Error: " err)))))
		(begin
			((res "header") "Content-Type" "text/plain")
			((res "header") "WWW-Authenticate" "Basic realm=\"authorization required\"")
			((res "status") 401)
			((res "print") "Unauthorized"))))))
(define http_handler (begin
	(define previous http_handler)
	(lambda (req res) (match (req "path")
		(regex "^/tsql/([^/]+)$" _ schema) (tsql_http_query req res schema)
		_ (previous req res)))))
(service_registry "T-SQL Frontend" (list (arg "api-port" (env "PORT" "4321")) "/tsql/[database]" "POST, NDJSON"))

/* Network callbacks use the same accounts, transaction owner and SQL execution
path as the other frontends. Wire actions are interpreted here, never in scm/. */
(define tds_frontend (lambda (event session payload row fields state sequence) (match event
	"login" (begin
		(define user (payload "username"))
		(define database (payload "database"))
		(define stored (mysql_auth user))
		(if (and stored (equal? stored (password (payload "password"))) (mysql_schema user database))
			(begin
				((sql_policy user) database true false)
				(define session (newsession))
				(session "username" user)
				(session "schema" database)
				(session "syntax" "tsql")
				(session "wire_metadata" true)
				session) nil))
	"close" (tx_rollback session)
	"reset" (begin
		(tx_rollback session)
		(define replacement (newsession))
		(replacement "username" (session "username"))
		(replacement "schema" (session "schema"))
		(replacement "syntax" "tsql")
		(replacement "wire_metadata" true)
		replacement)
	"transaction" (begin
		(if (and (equal? (payload "action") 1) (not (has? '(0 2) (payload "isolation"))))
			(error "unsupported TDS transaction isolation") true)
		(define sql (match (payload "action") 1 "BEGIN" 2 "COMMIT" 3 "ROLLBACK" _ (error "unknown transaction action")))
		(with_autocommit session state sequence sql (lambda (tx)
			(tsql_execute_statement tx (session "schema") sql session (lambda (_row) true) (lambda args true))))
		(list "rows" 0 "transaction" (payload "action")))
	"execute" (begin
		(tds_bind_parameters session (tds_bind_declarations (payload "database") (payload "declarations"))
			(payload "parameters"))
		(define output (newsession))
		(define result (with_autocommit session state sequence (payload "text") (lambda (tx)
			(tsql_execute_statement tx (payload "database") (payload "text") session
				(lambda (values) (row (map (output "descriptors") (lambda (descriptor) (begin
					(define value (values (descriptor "name")))
					(if (nil? value) nil (if (equal? (descriptor "kind") 231) (string value) (if (equal? (descriptor "kind") 38) (intdiv value 1) value))))))))
				(lambda (_names descriptors) (begin
					(output "descriptors" descriptors)
					(fields descriptors)))))))
		(if (number? result) (list "rows" result) nil))
	"describe" (tds_describe (payload "database") (payload "text") (payload "declarations") session)
	"metadata" (error "T-SQL metadata procedures are not implemented yet")
	_ (error "unknown TDS frontend callback"))))
(if (arg "tds-port" nil) (begin
	(define port (arg "tds-port" nil))
	(tds port tds_frontend (arg "tds-database" "memcp-tests"))
	(service_registry "T-SQL protocol" (list port "" "TDS"))
	(print "TDS listener on port " port)))
