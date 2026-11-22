/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* SQL error disposition is frontend policy over the completed native
statement's original failure. The native owner alone attests restoration and
changes transaction state; language records never grant write/commit rights.
https://learn.microsoft.com/sql/t-sql/statements/set-xact-abort-transact-sql */
(define tsql_xact_abort (lambda (session)
	(coalesceNil (session "__tsql_xact_abort") false)))
(define tsql_failure_origin (lambda (failure) (if (failure_annotation failure "tsql-compile-scope") "compile"
	(match (failure_value failure)
		'("tsql-compile-error" _scope _cause) "compile"
		'("tsql-user-error" _cause) "user"
		(cons "tsql-error" _details) "runtime"
		'("cardinality-error" _message) "runtime"
		'("constraint-violation" "check" _name _message) "runtime"
		'("constraint-violation" "unique-declaration" _name _message) "runtime"
		'("constraint-violation" "not-null-declaration" _name _message) "runtime"
		'("column-declaration-conflict" "name" _table _column) "runtime"
		'("column-property-conflict" _table _column "tsql.rowguidcol" true _owner) "runtime"
		_ (if (failure_diagnostic failure) "runtime" "native")))))
(define tsql_arithmetic_batch_abort? (lambda (session failure)
	(and (tsql_arithabort session) (not (tsql_ansi_warnings session))
		(match (tsql_diagnostic_failure_value failure)
			'("tsql-error" number _message _state) (has? '(8115 8134) number)
			'("tsql-error" number _message _state _severity) (has? '(8115 8134) number)
			_ false))))
/* Provenance follows the actual failure, not the first failure ever handled
inside a borrowed statement. Deferred native diagnostics use the same scalar
profile at their actual producer, before crossing the completed owner. */
(define tsql_profile_failure (lambda (session failure) (begin
	(define existing (failure_annotation failure "tsql-execution-profile"))
	(define forced (session "__tsql_failure_origin"))
	(if (and existing (nil? forced)) failure
		(failure_annotate failure "tsql-execution-profile"
			(list (coalesceNil forced (tsql_failure_origin failure))
				(or (tsql_xact_abort session) (tsql_arithmetic_batch_abort? session failure))
				(coalesceNil (session "__tsql_handler_active") false)))))))
(define tsql_statement_call (lambda (owner session text recovery body) (begin
	(define record (lambda (failure) (error (tsql_profile_failure session failure))))
	(define policy (lambda (_failure completion) (begin
		(define profile (failure_annotation _failure "tsql-execution-profile"))
		(if (equal? completion "unrestored") "rollback"
			(transaction_rollback_only_failure? _failure) "preserve"
			(nil? profile) "rollback"
			(has? '("compile" "user") (car profile)) "preserve"
			(not (equal? (car profile) "runtime")) "rollback"
			(not (cadr profile)) "preserve"
			(nth profile 2) "rollback-only" "rollback"))))
	(owner text recovery (lambda (tx)
		(try (lambda () (begin
			(if (and (equal? (tx_state tx) "active")
				(equal? (session "__tsql_isolation_level") "SERIALIZABLE"))
				(tx_serializable tx) true)
			(body tx))) record)) policy))))

/* A lexical handler flag follows procedure/dynamic/trigger calls but is not
connection state. CATCH itself runs under the enclosing handler, if any. */
(define tsql_execution_fact_session (lambda (parent name value)
	(lambda args (match args
		'() (filter (parent) (lambda (key) (not (equal? key name))))
		'(key) (if (equal? key name) value (parent key))
		'(key _value) (if (equal? key name) (error "execution fact is immutable") (apply parent args))
		_ (apply parent args)))))
(define tsql_try_session (lambda (parent)
	(tsql_execution_fact_session parent "__tsql_handler_active" true)))

/* Compilation does not participate in XACT_ABORT. Annotation preserves both
the original native carrier and its originating lexical level across caches. */
(define tsql_compile_phase (lambda (session body)
	(try body (lambda (failure)
		(if (failure_annotation failure "tsql-compile-scope") (error failure)
			(or (list? (failure_value failure)) (string? (failure_value failure)))
			(error (failure_annotate failure "tsql-compile-scope" (tsql_noexec_scope_id session)))
			(error failure))))))

/* An outer request must not retain an uncommittable transaction after its
batch completes. Modules and dynamic batches borrow that request owner. */
(define tsql_finish_request (lambda (session body) (begin
	(define outcome (try (lambda () (list true (body)))
		(lambda (failure) (list false failure))))
	(define connection (tsql_connection_session session))
	(define doomed (and (connection "transaction")
		(equal? (tx_state (connection "__memcp_tx")) "rollback-only")))
	(if doomed (begin
		(tx_rollback connection)
		(tsql_transaction_clear connection)
		(connection "__tsql_transaction_end" 3)) true)
	(if (and (not (connection "transaction"))
		(equal? (tx_state (connection "__memcp_tx")) "aborted")) (begin
			(tsql_transaction_clear connection)
			(connection "__tsql_transaction_end" 3)) true)
	(if (not (car outcome)) (error (cadr outcome))
		doomed (tsql_raise_error 3998 "Uncommittable transaction detected at batch end; rolled back." 1)
		(cadr outcome)))))
