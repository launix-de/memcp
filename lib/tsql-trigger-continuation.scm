/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* Only these scalar language facts survive the old module's unwind. Native
transactions, row callbacks, temporary/cursor controllers and statement/session
forwarding closures never enter this source-sized payload. Diagnostic transport
is copied separately into a NEW facade, without delegating unknown keys. */
(define tsql_trigger_continuation_keys
	'("username" "__tsql_principal_id" "schema" "syntax" "__tsql_ansi_warnings" "__tsql_arithabort"
		"__tsql_xact_abort" "__tsql_concat_null_yields_null" "__tsql_datefirst" "__tsql_dateformat"
		"__tsql_language" "__tsql_nocount" "__tsql_fmtonly" "__tsql_quoted_identifier" "__tsql_ansi_nulls"
		"__tsql_ansi_nulls_locked" "__tsql_isolation_level" "__tsql_rowcount_limit" "__tsql_noexec"
		"__tsql_client_hostname" "__tsql_client_appname" "__tsql_identity" "__tsql_rowcount" "__tsql_last_error"
		"__tsql_error_number" "__tsql_error_message" "__tsql_error_state" "__tsql_error_severity"
		"__tsql_trigger_event" "__tsql_trigger_timing" "__tsql_trigger_columns" "__tsql_trigger_target_columns"
		"__tsql_trigger_depth" "__tsql_trigger_schema" "__tsql_trigger_name" "tsql_import_target" "tsql_import_source"))

/* The terminal native disposition and a real suffix failure are independent
facts. Deliver the suffix diagnostic first, retain its opaque datum, then stop
the outer batch with SQL Server's trigger-abort diagnostic. No rendered native
message is parsed or used to authorize recovery. */
(define tsql_raise_completed_trigger_abort (lambda (session cause) (begin
	(if (not (nil? cause)) (begin
		(define context (tsql_error_context cause))
		(define diagnostic (list "kind" "error" "number" (coalesceNil (context "number") 50000)
			"state" (coalesceNil (context "state") 1) "severity" (coalesceNil (context "severity") 16)
			"text" (context "message") "server" "MemCP" "procedure" "" "line" 1))
		(define deliver (session "__tsql_message"))
		(if deliver (deliver diagnostic) (print (context "message")))) true)
	(define text "The transaction ended in the trigger. The batch has been aborted.")
	(try (lambda ()
		(if (session "wire_metadata")
			(error (tds_diagnostic_error (list "number" 3609 "state" 1 "severity" 16
				"text" text "server" "MemCP" "procedure" "" "line" 1)))
			(tsql_raise_error 3609 text 1)))
		(lambda (terminal)
			(error (if (nil? cause) terminal
				(failure_annotate terminal "post-abort-failure" cause))))))))

/* A source-sized proof distinguishes a terminal rollback from one whose exact
next PC observes values. Labels/NOCOUNT cleanup do not require lexical payload
transfer. No predicate is evaluated, and no SQL leaf is compiled/executed here. */
(define tsql_abort_suffix_observes? (lambda (schema nodes pc seen)
	(if (nil? pc) false
		(has? seen pc) true
		(begin
			(define node (nodes (string pc)))
			(define command (car node))
			(if (or (equal? (car command) "label") (tsql_trigger_option_cleanup? schema command))
				(tsql_abort_suffix_observes? schema nodes (cadr node) (append seen pc))
				(expression_equal? command '("return" nil)) false
				true)))))
(define tsql_abort_scalar_lifetime? (lambda (command) (match command
	'("block" body) (reduce body (lambda (valid item) (and valid (tsql_abort_scalar_lifetime? item))) true)
	'("if" _condition yes no) (and (tsql_abort_scalar_lifetime? yes)
		(if no (tsql_abort_scalar_lifetime? no) true))
	'("while" _condition body) (tsql_abort_scalar_lifetime? body)
	_ (not (has? '("declare-table" "select-into" "cursor-declare" "cursor-assign-query"
		"cursor-open" "cursor-fetch" "cursor-close" "cursor-deallocate" "try") (car command))))))
(define tsql_validate_trigger_abort_lifetimes (lambda (schema commands) (begin
	(define flow (tsql_flow_commands commands nil nil nil (list nil (list) (list) 0 nil)))
	(define nodes (cadr flow))
	(define continuation (find (extract_assoc nodes (lambda (_key node) node))
		(lambda (node) (and (equal? (car (car node)) "recovery")
			(tsql_abort_suffix_observes? schema nodes (cadr node) '()))) nil))
	(if continuation
		(if (reduce commands (lambda (valid command) (and valid (tsql_abort_scalar_lifetime? command))) true)
			true (error "post-rollback trigger continuation requires scalar lexical lifetimes")) true))))

(define tsql_trigger_rollback_at (lambda (flow next statement session)
	(tsql_statement_call statement session "trigger ROLLBACK" true (lambda (tx) (begin
		(if (session "__tsql_post_abort")
			(tsql_raise_error 3903 "The ROLLBACK TRANSACTION request has no corresponding BEGIN TRANSACTION." 1) true)
		(if (nil? next) (request_transaction_abort tx)
			(request_transaction_abort tx (lambda () (begin
				/* Native terminal intent precedes ALL continuation preparation,
				including a frontend refusal to transfer a non-scalar setting. */
				(define observing (tsql_abort_suffix_observes? (session "schema") (cadr flow) next '()))
				(if (and observing (session "__tsql_identity_insert"))
					(error "post-rollback continuation requires scalar identity settings") true)
				(define contracts (if observing (session "__tsql_parameter_declarations") '()))
				(define payload (list flow next contracts
					(map contracts (lambda (declaration)
						(list (car declaration) (session (concat "__tsql_parameter:" (car declaration))))))
					(map tsql_trigger_continuation_keys (lambda (key) (list key (session key))))
					((tsql_identity_scope session) "__tsql_scope_identity")))
				(define fresh (newsession))
				/* These existing protocol callbacks carry transport, not a SQL
				executor or module scope. The fresh facade has no parent delegation. */
				(fresh "__tsql_message" (session "__tsql_message"))
				(fresh "wire_metadata" (session "wire_metadata"))
				(define continuation (close_procedure (lambda (execute OLD NEW payload)
					(tsql_resume_trigger_after_abort execute OLD NEW payload fresh))))
				(list continuation payload))))))))))

(define tsql_resume_trigger_after_abort (lambda (execute OLD NEW payload session) (begin
	(define flow (nth payload 0))
	(define next (nth payload 1))
	(define contracts (nth payload 2))
	(map (nth payload 4) (lambda (entry) (session (car entry) (cadr entry))))
	(if (not (nil? (session "__tsql_error_number")))
		(tsql_install_error_context session (list "number" (session "__tsql_error_number")
			"message" (session "__tsql_error_message") "state" (session "__tsql_error_state")
			"severity" (session "__tsql_error_severity"))) true)
	(session "__tsql_parameter_declarations" contracts)
	(map (nth payload 3) (lambda (entry) (session (concat "__tsql_parameter:" (car entry)) (cadr entry))))
	(session "__tsql_scope" (newsession))
	((tsql_identity_scope session) "__tsql_scope_identity" (nth payload 5))
	(session "__tsql_local_relations" '())
	(session "__tsql_table_variables" (newsession))
	(session "__tsql_post_abort" true)
	/* SQL names bind only these actual retained native relation identities.
	The new module scope owns its own scratch, never the transition payload. */
	(define temporary (tsql_temporary_scope session))
	(temporary "bindings" (list
		(list "deleted" (session "schema") (show OLD "name") "transition")
		(list "inserted" (session "schema") (show NEW "name") "transition")))
	(define statement (lambda (_text _recovery body _failure_policy)
		(execute session body)))
	(tsql_temporary_with_module session (lambda ()
		(tsql_with_table_variables session (lambda () (tsql_cursor_with_scope session (lambda ()
			(tsql_with_statement_diagnostic session (lambda (scoped)
				(tsql_run_bound_flow flow next statement scoped
					(lambda (_row) (error "result sets from statement triggers are not supported"))
					(lambda args (error "result sets from statement triggers are not supported"))))))))))))))
