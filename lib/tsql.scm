/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

(import "tsql-parser.scm")
(define tsql_queryplan_cache (newcachemap))

/* A separate frontend entrypoint keeps dialect decisions out of existing SQL
execution paths. The cache, transaction owner and compiled evaluator are shared. */
(define tsql_execute_statement (lambda (tx schema text session resultrow resultfields) (begin
	(define input (strtrim text))
	(define length (strlen input))
	(define input (if (and (> length 0) (equal? (substr input (- length 1) 1) ";"))
		(substr input 0 (- length 1)) input))
	(define username (coalesce (session "username") "root"))
	(define formula (cached_parse tsql_queryplan_cache (list parse_tsql) schema input
		(list (quote sql-policy-for) username) username session false tx))
	(sql_execute_formula session tx formula resultrow resultfields))))

(define tsql_http_query (lambda (req res schema) (begin
	(define stored (mysql_auth (req "username")))
	(if (and stored (equal? stored (password (req "password"))))
		(begin
			(define text ((req "body")))
			(try (lambda () (time (begin
				((res "header") "Content-Type" "text/event-stream; charset=utf-8")
				(define session (req "__session"))
				(session "username" (req "username"))
				(session "schema" schema)
				(extract_assoc (req "query") (lambda (key value) (session key value)))
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
		/* Named parameter declarations and exact wire domains require their own
		frontend binder. Reject them explicitly until that slice is implemented. */
		(if (or (not (empty_list? (payload "parameters"))) (not (equal? (payload "declarations") "")))
			(error "T-SQL parameter binding is not implemented yet") true)
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
	"describe" (error "pure T-SQL describe is not implemented yet")
	"metadata" (error "T-SQL metadata procedures are not implemented yet")
	_ (error "unknown TDS frontend callback"))))
(if (arg "tds-port" nil) (begin
	(define port (arg "tds-port" nil))
	(tds port tds_frontend (arg "tds-database" "memcp-tests"))
	(service_registry "T-SQL protocol" (list port "" "TDS"))
	(print "TDS listener on port " port)))
