/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* These are native declaration/name facts, not transition rows. Operation
columns are snapshotted once at completion entry, even for a zero-row UPDATE;
an INSERT reports assignment to every declared target column. Never derive
this predicate from a value change, row count or later nested operation. */
(define tsql_trigger_column_updated (lambda (event assigned declared name) (begin
	(if (has? '("insert" "update" "delete") event) true
		(error "UPDATE(column) requires a statement trigger invocation"))
	(if (not (nil? (find declared (lambda (column) (equal?? column name)) nil))) true
		(error "unknown UPDATE(column) target: " name))
	(if (equal? event "insert") true
		(equal? event "update") (not (nil? (find assigned (lambda (column) (equal?? column name)) nil)))
		false))))

/* Recipe admission is frontend control analysis, never trigger execution.
The ordinary T-SQL parser has already validated every SQL leaf. Reuse that
parser's semantic recovery AST to distinguish ROLLBACK from COMMIT instead of
guessing from captured text or introducing a second recovery grammar. */
(define tsql_trigger_recovery_abort? (lambda (schema text)
	(if (expression_equal? (parse_tsql schema text nil nil nil)
		(list (quote tsql_rollback_transaction) (quote session))) true
		(error "statement triggers support only terminal ROLLBACK transaction control"))))

/* DROP is admitted only for the ordinary parser's local temporary target.
Persistent DDL remains outside native mutation callbacks. */
(define tsql_trigger_temporary_drop_formula? (lambda (formula)
	(match formula
		((symbol tsql_drop_temporary) _name _ifexists (symbol session) (symbol tx)) true
		(cons (symbol !begin) statements) (and (not (empty_list? statements))
			(reduce statements (lambda (valid statement)
				(and valid (tsql_trigger_temporary_drop_formula? statement))) true))
		_ false)))
(define tsql_trigger_temporary_drop? (lambda (schema text)
	(match (parse_tsql schema text nil nil nil)
		((symbol !begin) formula _completion) (tsql_trigger_temporary_drop_formula? formula)
		_ false)))

/* The first completion profile has no autonomous transaction after ROLLBACK.
An aborting branch cannot have an externally observable suffix, even PRINT.
Only the exact invocation-local option cleanup below may follow it. Validate
BOTH IF arms before combining their facts.
Loops containing an abort and TRY/CATCH are explicit unsupported profiles; this
does not replace the existing module interpreter or its ordinary SQL leaves. */
/* A static NOCOUNT setter is invocation-local and restored at module exit.
It may remain on the normal path after a possible abort only if no observing
leaf follows it. Keep the actual command; never discard arbitrary SET or SQL.
The proof consumes the ordinary parser's exact setter formula, not SQL text. */
(define tsql_trigger_option_cleanup? (lambda (schema command) (match command
	'("statement" text) (if (equal? (tsql_insert_exec_leaf_kind text) "SET") (begin
		(define formula (parse_tsql schema text nil nil nil))
		(or (expression_equal? formula (list (quote session) "__tsql_nocount" true))
			(expression_equal? formula (list (quote session) "__tsql_nocount" false)))) false)
	_ false)))
(define tsql_trigger_commands_abort? (lambda (schema commands in_loop module)
	(reduce commands (lambda (aborts command)
		(if aborts
			(if (tsql_trigger_option_cleanup? schema command) true
				(error "executable trigger suffix after ROLLBACK is not supported"))
			(tsql_trigger_command_abort? schema command in_loop module))) false)))
(define tsql_trigger_command_abort? (lambda (schema command in_loop module) (match command
	'("recovery" text) (if module
		(error "transaction control in a trigger-invoked module is not supported")
		(begin
			(define aborts (tsql_trigger_recovery_abort? schema text))
			(if (and in_loop aborts) (error "ROLLBACK inside a trigger loop is not supported") aborts)))
	'("block" commands) (tsql_trigger_commands_abort? schema commands in_loop module)
	'("if" _condition yes no) (begin
		(define yes_aborts (tsql_trigger_command_abort? schema yes in_loop module))
		(define no_aborts (if (nil? no) false (tsql_trigger_command_abort? schema no in_loop module)))
		/* Native abort transfer never resumes its selected branch. Only an IF
		whose alternatives both abort makes the enclosing suffix unreachable. */
		(and yes_aborts no_aborts))
	'("while" _condition body) (begin
		(tsql_trigger_command_abort? schema body true module)
		false)
	'("break") (if in_loop false (error "trigger BREAK requires a loop"))
	'("continue") (if in_loop false (error "trigger CONTINUE requires a loop"))
	/* The ordinary source parser validated label scopes and targets. These
	continuations belong to the existing tsql_flow control graph. */
	'("goto" _name) false
	'("label" _name) false
	'("return" value) (if (or module (nil? value)) false (error "trigger RETURN cannot supply a value"))
	'("statement" text) (if (has? '("SELECT" "INSERT" "UPDATE" "DELETE" "WITH" "SET")
		(tsql_insert_exec_leaf_kind text)) false
		(and (equal? (tsql_insert_exec_leaf_kind text) "DROP")
			(tsql_trigger_temporary_drop? schema text)) false
		(error "unsupported statement-trigger SQL leaf"))
	'("select-into" name _query _definitions _owner)
	(if (tsql_temporary_name? name) false (error "persistent SELECT INTO is unsupported inside a statement trigger"))
	'("cursor-declare" _name _options _text) false
	'("cursor-assign-query" _name _options _text) false
	'("cursor-open" _name _global) false
	'("cursor-fetch" _name _global _movement _targets) false
	'("cursor-close" _name _global) false
	'("cursor-deallocate" _name _global) false
	/* Runtime dynamic text is parsed and passed through THIS same admission
	before its first leaf executes, preserving DDL and terminal-abort limits. */
	'("execute-dynamic" _expression) false
	'("execute" _owner _name _arguments) false
	'("execute-status" _owner _name _arguments _target) false
	'("declare-table" _name _columns) false
	'("insert-local" _name _columns _query) false
	'("insert-local-source" _source) false
	'("declare" _declarations) false
	'("assign" _name _operation _text) false
	'("print" _text) false
	'("raiserror" _arguments) false
	_ (error "unsupported statement-trigger control recipe"))))
/* A called procedure/dynamic batch has its own RETURN value but cannot
hide ROLLBACK from the outer trigger continuation. Reject recovery throughout
that module BEFORE its first leaf; only direct terminal trigger abort is owned. */
(define tsql_validate_trigger_body (lambda (schema commands module) (begin
	(tsql_trigger_commands_abort? schema commands false module)
	commands)))

/* Keep error control explicit. ROLLBACK receives its diagnostic immediately
before native abort transfer. RETURN defers its error to the SQL statement owner
after normal native completion, so caller CATCH does not undo native work and
an uncaught error is sent once. Both retain the ordinary module interpreter. */
(define tsql_prepare_trigger_commands (lambda (schema commands)
	(map (produceN (count commands)) (lambda (index) (begin
		(define command (nth commands index))
		(match command
			'("raiserror" arguments) (begin
				(if (< (+ index 1) (count commands)) true
					(error "trigger RAISERROR currently requires an immediately following terminal ROLLBACK or RETURN"))
				(match (nth commands (+ index 1))
					'("recovery" text) (tsql_trigger_recovery_abort? schema text)
					'("return" nil) true
					_ (error "trigger RAISERROR currently requires an immediately following terminal ROLLBACK or RETURN"))
				(list "trigger-raiserror" arguments
					(match (nth commands (+ index 1)) '("return" nil) true _ false)))
			'("recovery" text) (begin (tsql_trigger_recovery_abort? schema text) (list "trigger-rollback"))
			'("block" body) (list "block" (tsql_prepare_trigger_commands schema body))
			'("if" condition yes no) (list "if" condition
				(car (tsql_prepare_trigger_commands schema (list yes)))
				(if (nil? no) nil (car (tsql_prepare_trigger_commands schema (list no)))))
			'("while" condition body) (list "while" condition
				(car (tsql_prepare_trigger_commands schema (list body))))
			_ command))))))
(define tsql_prepare_trigger_body (lambda (schema commands module)
	(tsql_prepare_trigger_commands schema (tsql_validate_trigger_body schema commands module))))
