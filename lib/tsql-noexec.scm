/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* NOEXEC owns a compilation phase, not a second execution engine. Receipts
contain only syntax, declared SQL contracts and unresolved object names. This
module never invokes a statement runner, physical planner, result sink or SQL
module callable. Its native input is the request's existing metadata visibility.
https://learn.microsoft.com/sql/t-sql/statements/set-noexec-transact-sql
SQL Server 2022: https://dbfiddle.uk/5db6xYD9 https://dbfiddle.uk/PEB2oT4l
https://dbfiddle.uk/vBlMoboV https://dbfiddle.uk/xOhsfkPU */
(define tsql_noexec_scope_id (lambda (session) (begin
	(define scope (tsql_identity_scope session))
	(coalesceNil (scope "__tsql_compile_scope_id") (begin
		(define id (uuid)) (scope "__tsql_compile_scope_id" id) id)))))
(define tsql_compile_planning_scope (lambda (session quoted tx) (begin
	(define planning (newsession))
	(map '("username" "__tsql_principal_id" "schema" "__tsql_parameter_declarations"
		"__tsql_local_relations" "__tsql_temporary_bindings" "__tsql_compile_relations"
		"tsql_import_target" "tsql_import_source" "__tsql_ansi_nulls" "__tsql_ansi_warnings"
		"__tsql_concat_null_yields_null" "__tsql_arithabort")
		(lambda (key) (planning key (session key))))
	(define temporary_scope (session "__tsql_temporary_scope"))
	(planning "__tsql_temporary_planning" true)
	(planning "__tsql_temporary_bindings" (if temporary_scope (tsql_temporary_visible temporary_scope tx)
		(coalesceNil (session "__tsql_temporary_bindings") '())))
	(planning "__tsql_quoted_identifier" quoted)
	(planning "__tsql_compile_only" true)
	planning)))
(define tsql_compile_declared_source (lambda (alias database name planning) (begin
	(define declaration (find (coalesceNil (planning "__tsql_compile_relations") '())
		(lambda (entry) (and (equal?? (car entry) database) (equal?? (cadr entry) name))) nil))
	(if declaration (begin
		(define columns (nth declaration 2))
		(list alias database (make_local_relation_reference (symbol (concat "__compile_transition_" name)) columns)
			false nil (list "columns" columns "constraints" '("Unique" () "ForeignKeys" ())))) nil))))
(define tsql_compile_statement (lambda (text quoted session tx) (begin
	(define planning (tsql_compile_planning_scope session quoted tx))
	(define result (tsql_compile_with_binding_diagnostics (lambda ()
		(parse_tsql (session "schema") text (tsql_policy (tsql_session_actor session)) planning tx))))
	(tsql_compile_require_parameters result)
	(match result
		((symbol tsql-command-binding) _kind _state _value _dependencies) result
		_ (tsql_command_binding 'statement (list text quoted) '())))))
/* Parse expressions through the ordinary frontend, then bind their logical
projection/condition through the same declared source and type owner as SELECT.
Neither literals nor initializers are evaluated, even when conversion would fail. */
(define tsql_compile_expression (lambda (text mode contract session tx) (if (nil? text)
	(tsql_command_binding 'expression nil '()) (begin
		(define planning (tsql_compile_planning_scope session (tsql_quoted_identifier session) tx))
		(planning "__tsql_expression_only" true)
		(planning "__tsql_expression_syntax" true)
		(define expression (parse_tsql (session "schema") text (tsql_policy (tsql_session_actor session)) planning tx))
		(tsql_compile_require_parameters expression)
		(define expression (if contract (begin
			(define effective (tsql_effective_parameter_contract contract))
			(tsql_implicit_cast_expr expression (car effective) (cdr effective))) expression))
		(tsql_compile_query_binding (make_query_block (session "schema") '()
			(list "value" (if (equal? mode 'predicate) 1 expression))
			(if (equal? mode 'predicate) expression true) nil nil nil nil nil nil nil nil)
			(tsql_policy (tsql_session_actor session)) planning tx)))))
(define tsql_compile_variable_contract (lambda (name session) (begin
	(define declaration (find (session "__tsql_parameter_declarations")
		(lambda (entry) (equal?? (car entry) name)) nil))
	(if declaration (cadr declaration)
		(tsql_raise_error 137 (concat "Must declare the scalar variable @" name) 2)))))
(define tsql_compile_arguments (lambda (arguments session tx)
	(map arguments (lambda (argument) (match (cadr argument)
		'("default") (tsql_command_binding 'default argument '())
		'("wire-parameter" parameter) (tsql_command_binding 'rpc-input (tds_parameter_source_type (parameter "type")) '())
		'("output" name) (tsql_command_binding 'output (tsql_compile_variable_contract name session) '())
		text (tsql_compile_expression text 'value nil session tx))))))
(define tsql_compile_commands (lambda (commands session tx)
	(map commands (lambda (command) (tsql_compile_command command session tx)))))
/* Module definitions get a lexical declaration environment. OFF in a module
being declared is ordinary body syntax and cannot alter its declaring session.
No source snapshot/compiler or stored callable is instantiated here. */
(define tsql_compile_module_body (lambda (database parameters source tables transitions session tx) (begin
	(define scoped (tsql_compile_planning_scope session (tsql_quoted_identifier session) tx))
	(scoped "schema" database)
	(scoped "__tsql_noexec" true)
	(scoped "__tsql_local_relations" '())
	(scoped "__tsql_compile_relations" transitions)
	(define commands (cadr ((tsql_parse_batch_context database source
		(tsql_quoted_identifier session) true (tsql_ansi_nulls session)) "syntax")))
	(define parameters (map parameters (lambda (parameter) (list (car parameter) (cadr parameter)))))
	(scoped "__tsql_parameter_declarations" (tsql_batch_contracts commands parameters))
	(tsql_bind_local_parameter_aliases database parameters scoped)
	(map tables (lambda (declaration)
		(tsql_register_table_variable (car declaration) (cadr declaration) scoped)))
	(tsql_compile_composite 'module (tsql_compile_commands commands scoped tx)))))
(define tsql_compile_trigger (lambda (database target source timing session tx) (begin
	(define handle (table database false target false tx))
	(define view (if handle nil (sql_find_view tx database target)))
	(if (or handle (and view (equal? timing "replace_statement"))) true
		(tsql_raise_error 8197 (concat "Trigger object does not exist or is invalid for this operation: " target) 1))
	(define columns (if handle
		(map (get_schema database target tx) (lambda (column) (tsql_effective_column column nil)))
		(tsql_metadata_view_columns database target (tsql_policy (tsql_session_actor session))
			(tsql_compile_planning_scope session (tsql_quoted_identifier session) tx) tx)))
	(tsql_compile_module_body database '() source '()
		(list (list database "inserted" columns) (list database "deleted" columns)) session tx))))
(define tsql_compile_command (lambda (command session tx) (match command
	'("statement" text) (tsql_compile_statement text (tsql_quoted_identifier session) session tx)
	'("statement-source" text quoted) (tsql_compile_statement text quoted session tx)
	'("recovery" text) (tsql_compile_statement text (tsql_quoted_identifier session) session tx)
	'("insert-local-source" text) (tsql_compile_statement text (tsql_quoted_identifier session) session tx)
	'("set-options" _options _enabled) (tsql_command_binding 'setting command '())
	'("declare" declarations) (begin
		(tsql_bind_local_parameter_aliases (session "schema") declarations session)
		(tsql_compile_composite 'declare (map declarations (lambda (declaration)
			(if (expression_equal? (cadr declaration) '("CURSOR"))
				(tsql_command_binding 'cursor-variable declaration '())
				(tsql_compile_expression (nth declaration 2) 'value
					(tsql_compile_variable_contract (car declaration) session) session tx))))))
	'("declare-table" name columns)
	(tsql_command_binding 'table-variable (tsql_register_table_variable name columns session) '())
	'("assign" name operation text) (begin
		(define contract (tsql_compile_variable_contract name session))
		(if (expression_equal? contract '("CURSOR"))
			/* Cursor identity/assignment existence is an execution contract. */
			(tsql_command_binding 'cursor-assignment command '())
			(tsql_compile_expression (if operation (concat "@" name " " operation " (" text "\n)") text)
				'value contract session tx)))
	'("select-into" _name query definitions _owner)
	(tsql_compile_statement (concat definitions query) (tsql_quoted_identifier session) session tx)
	'("insert-local" name columns query)
	(tsql_compile_statement (concat "INSERT INTO @" name
		(if columns (concat "(" (reduce (map columns (lambda (column)
			(concat "[" (replace column "]" "]]") "]")))
			(lambda (left right) (concat left "," right))) ")") "") " " query)
		(tsql_quoted_identifier session) session tx)
	'("insert-execute" owner name columns invocation) (begin
		(define target (tsql_insert_exec_target owner name session tx))
		(define database (car target))
		(define relation (cadr target))
		(define missing (if (local_relation_reference? relation) false (nil? (table database false relation false tx))))
		(tsql_command_binding 'insert-execute (list
			(if missing columns (tsql_insert_declarations database relation
				(coalesceNil columns (tsql_insert_target_columns database relation tx)) nil tx))
			(tsql_compile_command invocation session tx))
			(if missing (list (list 'relation database relation)) '())))
	'("execute" _owner _name arguments) (tsql_compile_composite 'execute (tsql_compile_arguments arguments session tx))
	'("execute-status" _owner _name arguments target) (begin
		(tsql_compile_variable_contract target session)
		(tsql_compile_composite 'execute (tsql_compile_arguments arguments session tx)))
	'("execute-dynamic" expression) (tsql_compile_expression expression 'value '("NVARCHAR" -1) session tx)
	'("cursor-declare" _name options query) (begin
		(tsql_cursor_validate_update_options options)
		(tsql_compile_statement query (tsql_quoted_identifier session) session tx))
	'("cursor-assign-query" name options query) (begin
		(tsql_compile_variable_contract name session)
		(tsql_cursor_validate_update_options options)
		(tsql_compile_statement query (tsql_quoted_identifier session) session tx))
	'("cursor-fetch" _name _global movement targets) (begin
		(map targets (lambda (name) (tsql_compile_variable_contract name session)))
		(if (cadr movement) (tsql_compile_expression (cadr movement) 'value '("INT") session tx) true)
		(tsql_command_binding 'fetch command '()))
	'("cursor-open" _name _global) (tsql_command_binding 'cursor command '())
	'("cursor-close" _name _global) (tsql_command_binding 'cursor command '())
	'("cursor-deallocate" _name _global) (tsql_command_binding 'cursor command '())
	'("block" body) (tsql_compile_composite 'block (tsql_compile_commands body session tx))
	'("if" condition yes no) (tsql_compile_composite 'if
		(cons (tsql_compile_expression condition 'predicate nil session tx)
			(tsql_compile_commands (if no (list yes no) (list yes)) session tx)))
	'("while" condition body) (tsql_compile_composite 'while
		(list (tsql_compile_expression condition 'predicate nil session tx)
			(tsql_compile_command body session tx)))
	'("try" body recovery) (tsql_compile_composite 'try
		(tsql_compile_commands (merge (list body recovery)) session tx))
	'("branch" condition _yes _no _preserve) (tsql_compile_expression condition 'predicate nil session tx)
	'("print" expression) (tsql_compile_expression expression 'value nil session tx)
	'("return" expression) (tsql_compile_expression expression 'value '("INT") session tx)
	'("raiserror" arguments) (tsql_compile_composite 'message
		(map arguments (lambda (expression) (tsql_compile_expression expression 'value nil session tx))))
	'("trigger-raiserror" arguments _returning) (tsql_compile_command (list "raiserror" arguments) session tx)
	'("throw" arguments) (tsql_compile_composite 'throw
		(map arguments (lambda (expression) (tsql_compile_expression expression 'value nil session tx))))
	'("create-procedure" database _name parameters source _replace)
	(tsql_compile_module_body database (tsql_procedure_parameters parameters) source '() '() session tx)
	'("alter-procedure" database _name parameters source)
	(tsql_compile_module_body database (tsql_procedure_parameters parameters) source '() '() session tx)
	'("create-scalar-function" _name parameters return_type source) (begin
		(tsql_effective_parameter_contract return_type)
		(tsql_compile_module_body (session "schema") parameters source '() '() session tx))
	'("alter-scalar-function" _name parameters return_type source) (begin
		(tsql_effective_parameter_contract return_type)
		(tsql_compile_module_body (session "schema") parameters source '() '() session tx))
	'("create-table-function" _name parameters return_name columns source)
	(tsql_compile_module_body (session "schema") parameters source (list (list return_name columns)) '() session tx)
	'("alter-table-function" _name parameters return_name columns source)
	(tsql_compile_module_body (session "schema") parameters source (list (list return_name columns)) '() session tx)
	'("create-trigger" _database _name database target _events source _timing _replace)
	(tsql_compile_trigger database target source _timing session tx)
	'("alter-trigger" _database _name database target _events source _timing)
	(tsql_compile_trigger database target source _timing session tx)
	(cons kind _arguments) (if (has? '("break" "continue" "goto" "label" "reset" "trigger-rollback"
		"drop-function" "drop-procedure" "drop-trigger") kind)
		(tsql_command_binding 'control command '()) (error "missing compile contract for command " kind))
	_ (error "invalid compile command"))))
/* Compilation borrows the real request capability solely for declaration
visibility and cancellation. It does not enter a native statement decision.
Compile failures retain their raw cause plus their lexical compilation level;
a same-level TRY cannot turn them into execution, while an outer EXEC caller
can observe the ordinary diagnostic. Native failures are never relabelled. */
(define tsql_noexec_compile_command (lambda (command session) (begin
	(define request_tx ((tsql_connection_session session) "__memcp_tx"))
	(tx_check request_tx)
	(define tx (if ((tsql_connection_session session) "transaction") request_tx nil))
	(define result (try (lambda () (tsql_compile_command command session tx)) (lambda (cause)
		/* Classify the actual language failure, never failure_value of a native
		completion envelope. The original value is retained, not converted to a
		success or a fabricated native restoration fact. */
		(if (or (list? cause) (string? cause))
			(error (list "tsql-compile-error" (tsql_noexec_scope_id session) cause))
			(error cause)))))
	(tx_check request_tx)
	result)))

(define tsql_noexec_same_level_failure? (lambda (cause session)
	(if (failure_annotation cause "tsql-compile-scope")
		(equal? (failure_annotation cause "tsql-compile-scope") (tsql_noexec_scope_id session))
		(match (failure_value cause)
			'("tsql-compile-error" scope _original) (equal? scope (tsql_noexec_scope_id session))
			_ false))))
/* A TRY/CATCH encountered while compiling has no active runtime handler.
SQL Server permits OFF there, executes its suffix, then reports 35101 at the
region's END. An actual enclosing TRY which executed before ON keeps its
ordinary handler. SQL Server 2022: https://dbfiddle.uk/2JNQbBCL */
(define tsql_noexec_compiled_region (lambda (commands statement session resultrow resultfields) (begin
	(define result (tsql_run_commands commands statement session resultrow resultfields))
	(if (session "__tsql_noexec") result
		(tsql_raise_error 35101 "NOEXEC changed from ON to OFF in a compiled TRY/CATCH region." 1)))))
(define tsql_noexec_compiled_try (lambda (body recovery statement session resultrow resultfields) (begin
	(tsql_noexec_compiled_region body statement session resultrow resultfields)
	(tsql_noexec_compiled_region recovery statement session resultrow resultfields))))
