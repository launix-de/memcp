/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

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
			(mysql_run_sql_statement tx (session "schema") sql session (lambda (_row) true) (lambda args true))))
		(list "rows" 0 "transaction" (payload "action")))
	"execute" (begin
		/* Named parameter declarations and exact wire domains require their own
		frontend binder. Reject them explicitly until that slice is implemented. */
		(if (or (not (empty_list? (payload "parameters"))) (not (equal? (payload "declarations") "")))
			(error "T-SQL parameter binding is not implemented yet") true)
		(define output (newsession))
		(define result (with_autocommit session state sequence (payload "text") (lambda (tx)
			(mysql_run_sql_statement tx (payload "database") (payload "text") session
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
