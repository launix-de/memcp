/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* A SQL leaf owns one bounded error cell outside the native statement call.
Its hidden immutable callback is inherited by synchronous trigger/module leaves,
never enumerated into a planner session or retained by a reusable plan. Native
completion must drain normally before this owner raises the SQL error; CATCH
can then retain the actual transaction without rolling back prior statements. */
(define tsql_statement_diagnostic_session (close_procedure ((lambda (tsql_connection_session)
	(lambda (parent capture)
		(lambda args (match args
			'() (filter (parent) (lambda (key) (not (equal? key "__tsql_statement_error"))))
			'(key) (if (equal? key "__tsql_statement_error") capture
				(equal? key "__tsql_connection_session") (tsql_connection_session parent) (parent key))
			'(key value) (if (equal? key "__tsql_statement_error")
				(error "statement error callback is immutable") (parent key value))
			_ (apply parent args))))) tsql_connection_session)))
(define tsql_with_statement_diagnostic (lambda (session callback)
	(if (session "__tsql_statement_error") (callback session) (begin
		(define pending (newsession))
		(define capture (lambda args (match args
			'() (pending "error")
			'(failure) (pending "get_or_compute_scoped" "tsql-diagnostic" "error"
				(lambda () (begin (pending "error" failure) failure))))))
		/* Consume the statement result before reading its diagnostic. Native
		mutation completion and the statement's decision have both returned. */
		((lambda (result) (begin
			(define failure (capture))
			(if failure (error failure) result)))
			(callback (tsql_statement_diagnostic_session session capture)))))))

/* Native completion supplies real OLD/NEW relation handles and one admitted
invocation. A module owns only SQL names/control and its own existing temporary
scope/cache. Transition data and lifetime remain native, never Scheme row bags. */
(define tsql_trigger_session (lambda (parent event timing columns declared depth schema name) (begin
	(define values (newsession))
	(values "__tsql_trigger_event" event)
	(values "__tsql_trigger_timing" timing)
	(values "__tsql_trigger_columns" columns)
	(values "__tsql_trigger_target_columns" declared)
	(values "__tsql_trigger_depth" depth)
	(values "__tsql_trigger_schema" schema)
	(values "__tsql_trigger_name" name)
	(define keys (values))
	(lambda args (match args
		'() (merge_unique (list keys (filter (parent) (lambda (key) (not (has? keys key))))))
		'(key) (if (has? keys key) (values key) (parent key))
		'(key value) (if (has? keys key) (error "statement trigger invocation facts are immutable") (parent key value))
		_ (apply parent args))))))

(define tsql_run_trigger_recipe (lambda (commands contracts schema name event timing quoted ansi OLD NEW caller tx) (begin
	/* The supported default disables direct AFTER recursion. A different
	intervening statement-trigger module remains an indirect nested invocation.
	RECURSIVE_TRIGGERS configuration is not offered by this initial profile. */
	(define direct (and (equal? timing "after_statement")
		(equal? (caller "__tsql_trigger_timing") "after_statement")
		(equal? (caller "__tsql_trigger_schema") schema) (equal? (caller "__tsql_trigger_name") name)))
	/* A prior trigger diagnostic already selected SQL error control. Remaining
	native completion programs still drain, but cannot execute a SQL suffix. */
	(define diagnostic (caller "__tsql_statement_error"))
	(if (or direct (if diagnostic (diagnostic) false)) 0 (begin
		(define depth (+ 1 (coalesceNil (caller "__tsql_trigger_depth") 0)))
		(if (> depth 32) (tsql_raise_error 217 "maximum trigger nesting level exceeded" 1) true)
		(tsql_require_database_access (tsql_session_actor caller) schema)
		/* The generic completion API attests the actual operation, not this SQL
		module's current child DML or the contents of its transition relations. */
		(define columns (mutation_columns tx))
		(define declared (map (show OLD) (lambda (column) (column "Field"))))
		(define scoped (tsql_scope_session caller contracts schema true))
		(scoped "__tsql_quoted_identifier" quoted)
		(scoped "__tsql_ansi_nulls" ansi)
		(scoped "__tsql_ansi_nulls_locked" true)
		(define session (tsql_trigger_session scoped event timing columns declared depth schema name))
		(define temporary (tsql_temporary_scope session))
		/* Borrowed lexical #temp names are readable, but their schema/write Undo
		is not part of this first mutation profile. Shadow only those real bindings;
		native query scratch keeps its separate existing relation-scope rights. */
		(define borrowed (map (filter (tsql_temporary_visible temporary tx) (lambda (binding)
			(tsql_temporary_name? (car binding)))) (lambda (binding)
				(list (car binding) (cadr binding) (nth binding 2) "trigger_temporary"))))
		(temporary "bindings" (merge (list (list
			(list "deleted" schema (show OLD "name") "transition")
			(list "inserted" schema (show NEW "name") "transition")) borrowed)))
		/* A SQL leaf borrows this admitted completion. Nested DML still opens its
		own native operation through the normal physical writer, never the request
		statement CAS or an implicit COMMIT/ROLLBACK in this callback. */
		(define statement (lambda (_text _recovery body) (begin (tx_check tx) (body tx))))
		(tsql_temporary_with_module session (lambda () (tsql_with_table_variables session (lambda () (tsql_cursor_with_scope session (lambda ()
			(tsql_run_commands commands statement session
				(lambda (_row) (error "result sets from statement triggers are not supported"))
				(lambda args (error "result sets from statement triggers are not supported"))))))))))))))

/* Choose ERROR explicitly. Severity remains SQL RAISERROR policy here; the
generic transport callback receives only typed, bounded diagnostic data. */
(define tsql_run_trigger_raiserror (lambda (arguments statement session returning) (begin
	(define raised (tsql_bind_raiserror arguments statement session))
	(if (<= (nth raised 4) 10)
		(tsql_emit_message session (nth raised 1) (if (equal? (nth raised 4) 10) 0 (nth raised 4))
			(nth raised 3) (nth raised 2))
		returning (begin
			(define capture (session "__tsql_statement_error"))
			(if capture (capture raised)
				(error "trigger RETURN diagnostic requires its SQL statement owner"))
			0)
		(begin
			(define diagnostic (list "kind" "error" "number" (nth raised 1) "state" (nth raised 3)
				"severity" (nth raised 4) "text" (nth raised 2) "server" "MemCP"
				"procedure" (coalesceNil (session "__tsql_trigger_name") "") "line" 1))
			(define deliver (session "__tsql_message"))
			(if (and (session "wire_metadata") (nil? deliver))
				(error "statement trigger ERROR requires an active diagnostic sink") true)
			(if deliver (deliver diagnostic) (print (nth raised 2)))
			0)))))

/* Only the outer batch boundary translates the drained native transfer.
TRY/CATCH inside the batch must propagate it, so no later SQL statement runs.
Native storage has already completed whole-transaction Undo and cleared the
ordinary connection marker before this language-specific message is raised. */
(define tsql_trigger_abort_boundary (lambda (session callback)
	(try callback (lambda (failure)
		(if (transaction_abort_transfer? failure) (begin
			(session "__tsql_transaction_end" 3)
			(define text "The transaction ended in the trigger. The batch has been aborted.")
			(if (session "wire_metadata")
				(error (tds_diagnostic_error (list "number" 3609 "state" 1 "severity" 16
					"text" text "server" "MemCP" "procedure" "" "line" 1)))
				(tsql_raise_error 3609 text 1)))
			(match failure
				'("tsql-error" number text state severity)
				(if (session "wire_metadata")
					(error (tds_diagnostic_error (list "number" number "state" state "severity" severity
						"text" text "server" "MemCP" "procedure" "" "line" 1)))
					(error failure))
				_ (error failure)))))))

(define tsql_compile_statement_trigger (lambda (source context quoted) (begin
	(if (context "schema_only") (error "schema-only statement trigger sources are not supported") true)
	(define event (match (context "timing")
		"after_statement_insert" "insert"
		"after_statement_update" "update"
		"after_statement_delete" "delete"
		"replace_statement_delete" "delete"
		_ (error "unsupported statement trigger timing")))
	(define timing (if (equal? (context "timing") "replace_statement_delete") "replace_statement" "after_statement"))
	(define schema (context "schema"))
	/* Body parsing includes the ordinary label/loop validation. Definitions
	retain only recipes and contracts, never the creator's execution context. */
	(define commands (tsql_prepare_trigger_body schema
		(cadr ((tsql_parse_batch_context schema source quoted true (coalesceNil (context "__tsql_ansi_nulls") true)) "syntax")) false))
	(define contracts (tsql_batch_contracts commands '()))
	(eval (list 'lambda (list 'OLD 'NEW 'session 'tx)
		(list 'tsql_run_trigger_recipe (list 'quote commands) (list 'quote contracts)
			schema (context "name") event timing quoted (coalesceNil (context "__tsql_ansi_nulls") true) 'OLD 'NEW 'session 'tx))))))

(define tsql_define_statement_trigger (lambda (owner name target_owner target events source timing replace must_exist statement session)
	(statement "statement trigger definition" false (lambda (tx) (begin
		(if (or (session "__tsql_insert_exec") (session "__tsql_trigger_event"))
			(error "trigger DDL is unsupported inside native mutation execution") true)
		(if (session "tsql_import_target") (error "retargeted statement trigger definitions are not supported") true)
		(define database (tsql_qualifier_database owner (session "schema") session))
		(define target_database (tsql_qualifier_database target_owner (session "schema") session))
		(if (equal? database target_database) true (error "trigger and target must belong to the same database"))
		(if (tsql_temporary_name? target) (error "temporary statement trigger targets are not supported") true)
		(tsql_require_permission (tsql_session_actor session) database "ALTER")
		(define handle (tsql_native_relation_handle database target tx))
		(if handle true (error "trigger target does not exist: " target))
		(createstatementtrigger handle (toLower name) timing (tsql_statement_language session) source events replace must_exist tx)
		0)))))

(tsql_register_source_profiles "tsql-statement" tsql_compile_statement_trigger
	(lambda (name compiler) (registertriggerlanguage name compiler false false true)))
