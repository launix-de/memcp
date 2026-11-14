/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* Multi-statement table functions share the validated relational state
compiler. Only their result contract differs: initialization forwards bounded
state, while the declared return table exposes its complete native multiset. */
(define tsql_table_function_language? (lambda (language)
	(equal? (tsql_source_module_kind language) "TF")))
/* Query-once initialization requires an actual immutable invocation input.
Do not infer that property from absence of column references: feedback symbols,
iteration markers and opaque callable/source values are not literal inputs. */
(define tsql_table_function_argument_leaf? (lambda (expression) (match (expression_syntax expression)
	((symbol sql_type_expr) value _type _collation) (tsql_table_function_argument_leaf? value)
	((symbol sql_compile_column) _compiler value) (tsql_table_function_argument_leaf? value)
	((symbol session) key)
	(and (string? key) (equal? (substr key 0 (strlen "__tsql_parameter:")) "__tsql_parameter:"))
	((symbol nth) payload index) (and (number? index) (tsql_table_function_argument_leaf? payload))
	value (or (nil? value) (number? value) (string? value)
		(expression_equal? value true) (expression_equal? value false)
		(logical_sql_null_literal? value)))))
(define tsql_table_function_recipe (lambda (source context quoted) (begin
	(define database (context "schema"))
	(define signature (context "parameters"))
	(if (equal? (count signature) 4) true (error "invalid table function declaration contract"))
	(define parameters (tsql_unique_parameters (car signature)))
	(define return_name (nth signature 1))
	(define state (newsession))
	/* The native catalog stores normalized declaration data only. A lexical
	relation symbol belongs to this compiled recipe, never to persistence. */
	(state "tables" (list (list return_name (symbol (concat "__tf_return_table_" return_name))
		(nth signature 2) (nth signature 3))))
	(state "ansi_nulls" (coalesceNil (context "__tsql_ansi_nulls") true))
	(state "contracts" (map parameters (lambda (parameter) (begin
		(define type (tsql_scalar_function_declaration (cadr parameter)))
		(if (> (count parameter) 2)
			(tsql_scalar_function_convert (tsql_scalar_function_default_info parameter) type) true)
		(list (car parameter) type)))))
	(if (find parameters (lambda (parameter) (equal? (car parameter) return_name)) nil)
		(error "duplicate table function return variable") true)
	(define commands (tsql_relational_scalar_commands source database quoted (state "ansi_nulls")))
	(match (car (reverse commands)) '("return" nil) true
		_ (error "table-valued function requires a terminal bare RETURN"))
	(define steps (tsql_relational_scalar_steps commands database state 'table-return quoted 0))
	(list "identity" (list database (context "name")) "database" database
		"parameters" parameters "ansi_nulls" (state "ansi_nulls") "contracts" (state "contracts") "tables" (state "tables")
		"return_table" return_name "steps" steps))))

(define tsql_table_function_expand (lambda (recipe arguments alias policy planning tx) (begin
	(define planning (tsql_ansi_planning_scope planning (recipe "ansi_nulls")))
	(define planning (tsql_scalar_function_binding_scope planning (recipe "identity")))
	(define database (recipe "database"))
	(define parameters (recipe "parameters"))
	(define contracts (recipe "contracts"))
	(define values (mapIndex arguments (lambda (index argument) (begin
		/* Column references require a lateral D relation. This first profile
		admits immutable leaves and current invocation parameters only. */
		(if (tsql_table_function_argument_leaf? argument) true
			(error "correlated or compound table function arguments require a lateral invocation domain"))
		(define target (tsql_scalar_function_declaration (cadr (nth parameters index))))
		(tsql_implicit_cast_expr argument (car target) (cdr target))))))
	(define initial (cons 'list (merge (list values
		(map (produceN (- (count contracts) (count parameters))) (lambda (_index) nil))
		(tsql_scalar_function_context_arguments)))))
	(define seed (make_query_block database '() (list "__fn_state" initial)
		true nil nil nil nil nil nil nil nil))
	(define meta (newsession))
	(meta "captures" '())
	(meta "step" 0)
	(meta "names" (tsql_relational_name_context (list seed recipe alias arguments)))
	(define program (tsql_relational_expand_steps recipe (recipe "steps") seed '() policy planning tx meta))
	/* The one-row initializer is an ordinary existing sequence stage. Its
	identity step forwards state without touching the table's output rows. */
	(define state (symbol (tsql_relational_fresh_alias (meta "names") "__table_function_state")))
	(define final_state (make_query_block database '() (list "__fn_state" state)
		true nil nil nil nil nil nil nil nil))
	(define initialization (make_relational_sequence program final_state state '()))
	(define initializer (car (qb_stages initialization)))
	(define returned (find (recipe "tables") (lambda (declaration)
		(equal? (car declaration) (recipe "return_table"))) nil))
	(define columns (nth returned 3))
	(define source (list alias database (make_local_relation_reference (nth returned 1) columns)
		false nil (list "columns" columns "constraints" '("Unique" () "ForeignKeys" ()))))
	(define output (make_query_block database (list source)
		(merge (map columns (lambda (column)
			(list (column "Field") (list 'get_column alias false (column "Field") false)))))
		true nil nil nil nil nil nil nil nil))
	(define query (make_invocation_relation_query output
		(map (recipe "tables") (lambda (declaration)
			(list (nth declaration 1) database (nth declaration 2) (nth declaration 3)))) initializer))
	(define query (tsql_relational_mark_capture_scopes query (source_aliases (meta "captures"))))
	(tsql_relational_authorize query policy)
	(sql_bind_catalog_declarations
		(tsql_expand_relational_scalar_calls (sql_expand_views query policy planning tx) '() policy planning tx) tx planning))))

(define tsql_table_function_compile (lambda (source context quoted) (begin
	(define recipe (tsql_table_function_recipe source context quoted))
	(close_procedure ((lambda (recipe expand)
		(lambda (arguments alias policy planning tx) (expand recipe arguments alias policy planning tx)))
		recipe tsql_table_function_expand)))))
(tsql_register_source_profiles "tsql-table-function" tsql_table_function_compile registerprocedurelanguage)
(define tsql_define_table_function (lambda (name parameters return_name columns source statement session)
	(statement "table function definition" false (lambda (tx) (begin
		(if (or (session "__tsql_insert_exec") (session "__tsql_trigger_event"))
			(error "function DDL is unsupported inside a native mutation") true)
		(define database (session "schema"))
		(tsql_require_database_access (tsql_session_actor session) database)
		(if (or (show (table database false name true tx) "name") (tsql_view_name database name))
			(error "a relation already owns this object name: " name) true)
		(define declaration (tsql_relational_table_declaration database return_name columns))
		(createprocedure database (toLower name)
			(tsql_source_language "tsql-table-function" session)
			source (list parameters return_name (nth declaration 2) (nth declaration 3)) false tx)
		0)))))

(define tsql_table_function_call (lambda (database name arguments alias policy planning tx) (begin
	(if policy (policy database name false) (error "table function requires an authorized SQL plan"))
	(define name (toLower name))
	(define snapshot (procedure database name true tx))
	(define definition (snapshot "declaration"))
	(if (tsql_table_function_language? (definition "language")) true
		(error "object is not a table-valued function: " name))
	(planner_record_guard_condition
		(list 'expression_equal? (list 'showprocedures database name 'tx) (list 'quote (list definition))) planning)
	(define parameters (car (definition "parameters")))
	(if (equal? (count arguments) (count parameters)) true (error "wrong number of table function arguments"))
	(define values (mapIndex arguments (lambda (index argument) (match argument
		'("scalar-parameter-default") (begin
			(define info (tsql_scalar_function_default_info (nth parameters index)))
			(list 'sql_type_expr (sql_info_formula info) (sql_info_datatype info) (sql_info_collation info)))
		_ argument))))
	((snapshot "callable") values alias policy planning tx))))
