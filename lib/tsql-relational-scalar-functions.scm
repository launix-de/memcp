/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */
/* A relational FN compiles to a planning factory, never a row-time SQL
interpreter. State and assignment packets are bounded tuples, not result bags.
Each state projection is a normal global aggregate: its exported slot remains
one logical producer when later statements read the same local value. */
(define tsql_relational_scalar_language? (lambda (language)
	(not (nil? (tsql_source_language_profile "tsql-relational-scalar-function" language)))))
/* Local table schemas are immutable source declarations. Their lexical native
handle is introduced only by the common invocation-scope physical owner. */
(define tsql_relational_table_source (lambda (database name alias planning) (begin
	(define declaration (find (coalesceNil (planning "__tsql_local_relations") '())
		(lambda (declaration) (equal? (car declaration) name)) nil))
	(if declaration true (error "undeclared local table variable: @" name))
	(list (coalesceNil alias name) (if (> (count declaration) 4) (nth declaration 4) database)
		(make_local_relation_reference (nth declaration 1) (nth declaration 3)) false nil
		(list "lifetime" "invocation" "columns" (nth declaration 3) "constraints" '("Unique" () "ForeignKeys" ()))))))
/* Parenthesized literals use the parser's deferred DEFAULT spelling too.
Recover only a syntax-proven literal; functions and session-dependent recipes
still require their own invocation expression owner and are never sampled. */
(define tsql_relational_literal_default (lambda (database attributes) (begin
	(define source (attributes "default_expression"))
	(if (and source (attributes "default_deferred")) (begin
		(define info (parse_tsql database source nil (tsql_default_planning_scope nil) nil))
		(define value (sql_info_formula info))
		(if (or (logical_sql_null_literal? value) (number? value) (string? value)
			(expression_equal? value true) (expression_equal? value false))
			(append (merge (extract_assoc attributes (lambda (key item)
				(if (has? '("default_expression" "default_deferred") key) '() (list key item)))))
				"default" value "__tsql_default_source" source)
			attributes)) attributes))))
(define tsql_relational_table_declaration (lambda (database name columns) (begin
	(define native (map columns (lambda (column) (begin
		(define attributes (tsql_relational_literal_default database (nth column 3)))
		/* Literal defaults, keys and nullability already have native column
		contracts. Normalize their SQL types through tsql_prepare_column and
		publish them with this private table; deferred defaults/identity need
		separate invocation expression owners. */
		(if (reduce (list "auto_increment" "default_expression")
			(lambda (found key) (or found (has_assoc? attributes key))) false)
			(error "local table identity/default declarations require lowering") true)
		/* Use the same schema-type normalization as CREATE TABLE. Delimited
		parser spellings are source names, not native storage declarations. */
		(define prepared (tsql_prepare_column database
			(list 'list "column" (car column) (cadr column) (nth column 2)
				(cons 'list attributes)) nil))
		(define options (tsql_publish_alias_attributes (nth prepared 3) (cdr (nth prepared 5))))
		(list "column" (nth prepared 2) (nth prepared 3) (cdr (nth prepared 4))
			(set_assoc options "null" (if (has_assoc? options "null") (options "null") true)))))))
	(define declarations (map native (lambda (column) (begin
		(define type (nth column 2))
		(define attributes (nth column 4))
		(list "Field" (nth column 1) "Type" type "RawType" type
			"Dimensions" (nth column 3) "Null" (attributes "null")
			"Default" (attributes "default") "DefaultExpression" (attributes "__tsql_default_source")
			"DefaultDeferred" false
			"Collation" (coalesceNil (attributes "collate")
				(if (has? '("CHAR" "VARCHAR" "TEXT" "NCHAR" "NVARCHAR" "NTEXT") type) "utf8mb4" "")))))))
	(if (equal? (count declarations) (count (merge_unique (list (map declarations (lambda (column) (toLower (column "Field"))))))))
		true (error "duplicate local table column"))
	(list name (symbol (concat "__fn_local_table_" name)) native declarations))))
(define tsql_relational_scalar_commands (lambda (source database quoted ansi) (begin
	(define commands (cadr ((tsql_parse_batch_context database source quoted true ansi) "syntax")))
	(if (equal? (count commands) 1) true (error "relational scalar function requires one BEGIN/END body"))
	(match (car commands) '("block" body) body
		_ (error "relational scalar function requires one BEGIN/END body")))))
/* Reuse the existing frontend keyword lexer (including comments). Statement
capture alone does not make INSERT/UPDATE/DELETE a relational read recipe. */
(define tsql_relational_scalar_command_reads? (lambda (command) (match command
	'("declare-table" _name _columns) true
	'("insert-local" _name _columns _query) true
	'("block" commands) (reduce commands (lambda (found child)
		(or (tsql_relational_scalar_command_reads? child) found)) false)
	'("while" _condition body) (tsql_relational_scalar_command_reads? body)
	'("assign" _name _operation expression)
	(reduce (tsql_context_tokens expression) (lambda (found token)
		(or found (and (equal? (car token) "word") (equal?? (cadr token) "SELECT")))) false)
	'("if" condition yes no) (or
		(reduce (tsql_context_tokens condition) (lambda (found token)
			(or found (and (equal? (car token) "word") (equal?? (cadr token) "SELECT")))) false)
		(tsql_relational_scalar_command_reads? yes)
		(if (nil? no) false (tsql_relational_scalar_command_reads? no)))
	(cons kind arguments) (if (has? '("statement" "statement-source") kind)
		(if (has? '("SELECT" "UPDATE") (tsql_insert_exec_leaf_kind (car arguments))) true
			(error "unsupported statement in scalar function body")) false)
	_ false)))
(define tsql_relational_scalar_source? (lambda (source database quoted ansi)
	(reduce (tsql_relational_scalar_commands source database quoted ansi) (lambda (found command)
		(or (tsql_relational_scalar_command_reads? command) found)) false)))
/* Parse scalar SQL as an expression projection, retaining nested queries for
the common binder/decorrelator. No expression is evaluated while compiling a
source module, and a SET scalar subquery retains its empty-result NULL. */
/* Parsing owns only lexical function syntax. Dependency guards and recursive
binding paths belong to the consuming plan, including calls inside a module. */
(define tsql_relational_query_scope (lambda (parent contracts quoted tables)
	(begin
		(define parent (coalesceNil parent (newsession)))
		(lambda args (match args
			'(key) (if
				(has? '("__tsql_logical_query" "__tsql_scalar_function_body") key) true
				(equal? key "__tsql_parameter_declarations") contracts
				(equal? key "__tsql_quoted_identifier") quoted
				(equal? key "__tsql_local_relations") tables
				/* A caller's captured SQL text and parser mode are not the
				callee's source. Only semantic dependency owners are inherited. */
				(has? '("__tsql_logical_update" "__tsql_lexical_context" "__tsql_script_syntax"
					"__tsql_expression_syntax" "__tsql_expression_only"
					"__tsql_expression_source" "__tsql_parameter_analysis"
					"__tsql_describe" "__tsql_static_module") key) nil
				(parent key))
			_ (apply parent args))))))
(define tsql_relational_deferred_call? (lambda (node) (match node
	((symbol quote) _data) false
	((symbol tsql_deferred_scalar_call) _database _name _arguments) true
	_ (begin
		(define found (newsession))
		(tsql_map_executable_children node (lambda (child) (begin
			(if (tsql_relational_deferred_call? child) (found "call" true) true)
			child)))
		(coalesceNil (found "call") false)))))
(define tsql_relational_requires_query? (lambda (node)
	(or (expr_contains_subquery? node) (tsql_relational_deferred_call? node))))
(define tsql_relational_scalar_value (lambda (database text contracts quoted policy planning tx tables) (begin
	(define query (parse_tsql database (concat "SELECT (" text ") AS __fn_value") policy
		(tsql_relational_query_scope planning contracts quoted tables) tx))
	(get_assoc (qb_fields query) "__fn_value"))))
(define tsql_relational_scalar_query (lambda (database text contracts quoted policy planning tx tables) (begin
	(define query (parse_tsql database text policy
		(tsql_relational_query_scope planning contracts quoted tables) tx))
	(define targets (qassoc_get (qb_facts query) 'tsql_select_assignments nil))
	(if (empty_list? targets) (error "relational scalar SELECT requires local assignment targets") true)
	(map targets (lambda (name)
		(if (find contracts (lambda (contract) (equal? (car contract) name)) nil) true
			(error "undeclared scalar SELECT target: @" name))))
	/* Explicit TOP owns its window. Unbounded ordered assignment needs an
	ordered final-tuple algebra; an unordered aggregate must not impersonate it. */
	(if (and (not (empty_list? (qb_order query))) (nil? (qb_limit query)))
		(error "ordered unbounded scalar assignment needs ordered final-tuple lowering") true)
	(list targets query))))
(define tsql_relational_insert_query (lambda (database text contracts quoted policy planning tx tables) (begin
	(define query (parse_tsql database text policy
		(tsql_relational_query_scope planning contracts quoted tables) tx))
	(if (query_block? query) query (error "local table INSERT currently requires a query block")))))
/* A local mutation retains ordinary UPDATE syntax until the shared planner
binds and decorrelates its complete relational input. No SQL is interpreted by
an execution callback, and only the declared private table is writable. */
(define tsql_relational_update_scope (lambda (parent contracts quoted tables) (begin
	(define context (tsql_relational_query_scope parent contracts quoted tables))
	(lambda args (match args
		'(key) (if (equal? key "__tsql_logical_query") false
			(equal? key "__tsql_logical_update") true (context key))
		_ (apply context args))))))
(define tsql_relational_update_query (lambda (database text contracts quoted policy planning tx tables) (begin
	(define query (parse_tsql database text policy
		(tsql_relational_update_scope planning contracts quoted tables) tx))
	(if (and (query_block? query) (qassoc_get (qb_facts query) 'local_update_target nil)) query
		(error "local UPDATE requires a declared relation target")))))
(define tsql_relational_statement_steps (lambda (database text contracts quoted tables planning) (begin
	(if (equal? (tsql_insert_exec_leaf_kind text) "UPDATE")
		(begin
			(tsql_relational_update_query database text contracts quoted nil planning nil tables)
			(list (list "update-local" text quoted)))
		(begin
			(define parsed (tsql_relational_scalar_query database text contracts quoted nil planning nil tables))
			(list (list "select" (car parsed) text quoted)))))))
(define tsql_relational_insert_input (lambda (database state_source query declaration columns bindings names) (begin
	(define alias (source_alias state_source))
	(define available (nth declaration 3))
	(define columns (coalesceNil columns (map available (lambda (column) (column "Field")))))
	(define targets (map columns (lambda (name) (begin
		(define column (find available (lambda (column) (equal?? (column "Field") name)) nil))
		(if column column (error "local INSERT column does not exist: " name))))))
	(if (equal? (count targets) (intdiv (count (qb_fields query)) 2)) true
		(error "local INSERT SELECT column count mismatch"))
	(if (equal? (count targets) (count (merge_unique (list (map targets (lambda (column) (column "Field")))))))
		true (error "duplicate local INSERT target"))
	(define packet_alias (tsql_relational_fresh_alias names (concat alias "_insert")))
	(define packet_fields (merge (list (list "__insert_present" true)
		(merge (mapIndex targets (lambda (index target)
			(list (concat "__insert_value_" index)
				(tsql_assignment_expr (nth (qb_fields query) (+ 1 (* index 2))) target))))))))
	(define packet (tsql_relational_substitute
		(make_query_block database (qb_sources query) packet_fields (qb_where query)
			(qb_group query) (qb_having query) (qb_order query) (qb_limit query) (qb_offset query)
			(qb_hidden query) (qb_stages query) (qb_facts query)) bindings))
	(define input (make_query_block database
		(list state_source
			(list packet_alias database packet true true nil))
		(merge (list (list "__fn_state" (list 'get_column alias false "__fn_state" false)
			"__insert_present" (list 'get_column packet_alias false "__insert_present" false))
			(merge (mapIndex targets (lambda (index target) (begin
				(define field (concat "__insert_value_" index))
				(list field (list 'get_column packet_alias false field false))))))))
		true nil nil nil nil nil nil nil nil))
	(list input (map targets (lambda (column) (column "Field")))))))
(define tsql_relational_scalar_steps (lambda (commands database state return_type quoted loop_depth)
	(merge (map commands (lambda (command) (match command
		'("block" body) (tsql_relational_scalar_steps body database state return_type quoted loop_depth)
		'("declare-table" name columns) (begin
			(if (or (find (state "contracts") (lambda (contract) (equal? (car contract) name)) nil)
				(find (state "tables") (lambda (table) (equal? (car table) name)) nil))
				(error "duplicate scalar function variable: @" name) true)
			(state "tables" (append (state "tables") (tsql_relational_table_declaration database name columns)))
			'())
		'("insert-local" name columns query) (begin
			(if (find (state "tables") (lambda (table) (equal? (car table) name)) nil) true
				(error "undeclared local table INSERT target: @" name))
			(define checked (tsql_relational_insert_query database query (state "contracts") quoted nil (tsql_ansi_planning_scope nil (state "ansi_nulls")) nil (state "tables")))
			(list (list "insert-local" name columns query quoted)))
		'("if" condition yes no) (begin
			(define expression (tsql_relational_scalar_value database condition
				(state "contracts") quoted nil (tsql_ansi_planning_scope nil (state "ansi_nulls")) nil (state "tables")))
			(define relational (tsql_relational_requires_query? expression))
			(if (and relational (not (tsql_relational_deferred_call? expression)))
				(if (equal? (sql_info_type (sql_expr_info '() expression)) "BOOLEAN") true
					(error "scalar function IF requires a Boolean predicate")) true)
			(define predicate (if relational condition
				(tsql_scalar_function_expression database condition (state "contracts") nil quoted (state "ansi_nulls"))))
			(define depth (coalesceNil (state "branch_depth") 0))
			(state "branch_depth" (+ depth 1))
			(define steps (list (list (if relational "if-query" "if") predicate
				(tsql_relational_scalar_steps (list yes) database state return_type quoted loop_depth)
				(if no (tsql_relational_scalar_steps (list no) database state return_type quoted loop_depth) '()) quoted)))
			(state "branch_depth" depth)
			steps)
		'("declare" declarations) (map declarations (lambda (declaration) (begin
			(define name (car declaration))
			(define contracts (state "contracts"))
			(if (or (find contracts (lambda (contract) (equal? (car contract) name)) nil)
				(find (state "tables") (lambda (table) (equal? (car table) name)) nil))
				(error "duplicate scalar function variable: @" name) true)
			(define type (tsql_scalar_function_declaration (cadr declaration)))
			(define value (if (nil? (nth declaration 2)) nil
				(tsql_scalar_function_expression database (nth declaration 2) contracts type quoted (state "ansi_nulls"))))
			(state "contracts" (append contracts (list name type)))
			(list "set" (list name) (list value)))))
		'("assign" name operation expression) (begin
			(if operation (error "relational scalar SET requires simple assignment") true)
			(define contract (find (state "contracts") (lambda (contract) (equal? (car contract) name)) nil))
			(if contract true (error "undeclared scalar function variable: @" name))
			(define value (tsql_relational_scalar_value database expression (state "contracts") quoted nil (tsql_ansi_planning_scope nil (state "ansi_nulls")) nil (state "tables")))
			(if (tsql_relational_requires_query? value)
				(list (list "set-query" (list name) expression quoted))
				(list (list "set" (list name) (list
					(tsql_scalar_function_expression database expression (state "contracts") (cadr contract) quoted (state "ansi_nulls")))))))
		'("statement" text)
		(tsql_relational_statement_steps database text (state "contracts") quoted (state "tables") (tsql_ansi_planning_scope nil (state "ansi_nulls")))
		'("statement-source" text saved_quoted)
		(tsql_relational_statement_steps database text (state "contracts") saved_quoted (state "tables") (tsql_ansi_planning_scope nil (state "ansi_nulls")))
		'("while" condition body) (begin
			(define predicate (tsql_scalar_function_expression database condition (state "contracts") nil quoted (state "ansi_nulls")))
			(list (list "while" predicate
				(tsql_relational_scalar_steps (list body) database state return_type quoted (+ loop_depth 1)))))
		'("return" expression) (begin
			(if (> loop_depth 0) (error "RETURN inside relational iteration requires a terminal branch") true)
			(if (> (coalesceNil (state "branch_depth") 0) 0)
				(error "RETURN inside relational choice requires a terminal branch") true)
			(if (expression_equal? return_type 'table-return)
				(if (nil? expression) (list (list "return-table"))
					(error "table-valued function RETURN cannot have a scalar expression"))
				(begin
					(define value (tsql_relational_scalar_value database expression (state "contracts") quoted nil (tsql_ansi_planning_scope nil (state "ansi_nulls")) nil (state "tables")))
					(if (tsql_relational_requires_query? value)
						(list (list "return-query" expression quoted))
						(list (list "return"
							(tsql_scalar_function_expression database expression (state "contracts") return_type quoted (state "ansi_nulls"))))))))
		_ (error "unsupported command in relational scalar function")))))))
(define tsql_relational_scalar_recipe (lambda (source context quoted) (begin
	(define database (context "schema"))
	(define signature (context "parameters"))
	(define parameters (tsql_unique_parameters (car signature)))
	(define return_type (tsql_scalar_function_declaration (cadr signature)))
	(define state (newsession))
	(state "tables" '())
	(state "ansi_nulls" (coalesceNil (context "__tsql_ansi_nulls") true))
	(state "contracts" (map parameters (lambda (parameter)
		(list (car parameter) (tsql_scalar_function_declaration (cadr parameter))))))
	(define commands (tsql_relational_scalar_commands source database quoted (state "ansi_nulls")))
	(match (car (reverse commands)) '("return" _expression) true
		_ (error "relational scalar function requires a terminal RETURN"))
	(define recipes (tsql_relational_scalar_steps commands database state return_type quoted 0))
	(list "identity" (list database (context "name"))
		"database" database "parameters" parameters "return" return_type "ansi_nulls" (state "ansi_nulls")
		"contracts" (state "contracts") "tables" (state "tables") "steps" recipes))))
(define tsql_relational_substitute (lambda (node bindings)
	(if (symbol? node) (get_assoc bindings (string node) node)
		(tsql_map_executable_children node (lambda (child) (tsql_relational_substitute child bindings))))))
/* Only aliases injected by this factory are eligible captures. Preserve SQL
scope shadowing while following dependencies through derived query boundaries;
a nested capture also makes its enclosing derived block dependent. */
(define tsql_relational_source_has_capture? (lambda (source aliases)
	(or (tsql_relational_has_capture? (source_relation source) aliases)
		(tsql_relational_has_capture? (source_join_expr source) aliases))))
(define tsql_relational_has_capture? (lambda (node aliases) (match node
	((symbol quote) _datum) false
	((symbol literal-rows) _columns _rows) false
	((symbol local-relation) _name _columns) false
	((symbol stage-output) _id) false
	((symbol sql_type_expr) expression _type _collation) (tsql_relational_has_capture? expression aliases)
	((symbol sql_compile_column) _compiler expression) (tsql_relational_has_capture? expression aliases)
	((symbol query-block) _database sources fields condition grouping having ordering limit offset hidden stages _facts)
	(begin
		(define visible (filter aliases (lambda (alias)
			(not (reduce (source_aliases sources) (lambda (shadowed local)
				(or shadowed (equal?? alias local))) false)))))
		(or (reduce sources (lambda (found source)
			(or found (tsql_relational_source_has_capture? source visible))) false)
			(tsql_relational_has_capture? (list fields condition grouping having ordering limit offset hidden stages) visible)))
	((symbol union-block) _mode branches ordering limit offset _facts)
	(tsql_relational_has_capture? (list branches ordering limit offset) aliases)
	((symbol group-stage) _id input domain keys aggregates having output ordering limit offset facts)
	(or (if (or (query_block? input) (union_block? input)) (tsql_relational_has_capture? input aliases)
		(tsql_relational_source_has_capture? input aliases))
		(tsql_relational_has_capture? (list domain keys aggregates having output ordering limit offset
			(qassoc_get facts 'iteration_predicate nil)) aliases))
	((symbol window-stage) _id source _column _sortcols _sortdirs _partitioncount _mapcols mapfn reducefn reduceinit _facts)
	(or (tsql_relational_source_has_capture? source aliases)
		(tsql_relational_has_capture? (list mapfn reducefn reduceinit) aliases))
	((symbol orc-stage) _id source _column _sortcols _sortdirs _partitioncount _mapcols mapfn reducefn reduceinit)
	(or (tsql_relational_source_has_capture? source aliases)
		(tsql_relational_has_capture? (list mapfn reducefn reduceinit) aliases))
	((symbol get_column) _alias _alias_ic _column _column_ic)
	(not (empty_list? (btw2025_expr_accessing_aliases node aliases)))
	(cons head tail) (or (tsql_relational_has_capture? head aliases)
		(reduce tail (lambda (found child)
			(or found (tsql_relational_has_capture? child aliases))) false))
	_ false)))
(define tsql_relational_mark_capture_scopes (lambda (node aliases) (begin
	(define mapped (tsql_map_executable_children node
		(lambda (child) (tsql_relational_mark_capture_scopes child aliases))))
	(if (and (query_block? node) (tsql_relational_has_capture? node aliases))
		(make_query_block (qb_schema mapped) (qb_sources mapped) (qb_fields mapped)
			(qb_where mapped) (qb_group mapped) (qb_having mapped) (qb_order mapped)
			(qb_limit mapped) (qb_offset mapped) (qb_hidden mapped) (qb_stages mapped)
			(qassoc_set (qb_facts mapped) 'lateral true)) mapped))))
(define tsql_relational_state_update (close_procedure (lambda (previous slots packet)
	(if (nil? packet) previous
		(mapIndex previous (lambda (index value) (begin
			(define position (find (produceN (count slots)) (lambda (i) (equal? (nth slots i) index)) nil))
			(if (nil? position) value (nth packet (+ position 1))))))))))
/* One invocation owns its generated names. Collect source strings only when
new syntax enters that invocation; never revisit the accumulated predecessor
program for each state/feedback name. Exact-string admission avoids duplicate
storage while collision checks retain the SQL Unicode comparator unchanged. */
(define tsql_relational_reserve_names (lambda (context inputs) (begin
	(define seen (context "seen"))
	(extract_assoc (binding_collect_string_catalog inputs '()) (lambda (name _present)
		(if (seen name) true (begin
			(seen name true)
			(context "names" (cons name (context "names"))) true))))
	true)))
(define tsql_relational_name_context (lambda (inputs) (begin
	(define context (newsession))
	(context "seen" (newsession))
	(context "next" (newsession))
	(context "names" '())
	(tsql_relational_reserve_names context inputs)
	context)))
(define tsql_relational_fresh_alias (lambda (context base) (begin
	(define next (context "next"))
	(define choose (lambda (choose index) (begin
		(define candidate (if (equal? index 0) base (concat base ":" index)))
		(if (find (context "names") (lambda (name) (equal?? name candidate)) nil)
			(choose choose (+ index 1)) (begin
				(next base (+ index 1))
				((context "seen") candidate true)
				(context "names" (cons candidate (context "names")))
				candidate)))))
	(choose choose (coalesceNil (next base) 0)))))
(define tsql_relational_source (lambda (database alias query)
	(list alias database query false true nil)))
(define tsql_relational_slots_from_payload (lambda (contracts payload)
	(merge (mapIndex contracts (lambda (index contract)
		(list (string (tsql_scalar_function_variable (car contract)))
			(list 'sql_type_expr (list 'nth payload index)
				(tsql_scalar_function_type (cadr contract)) nil)))))))
/* Pure recipes already completed SQL type selection. Their lexical reads
carry values only; relational expression parsing uses the typed slots above. */
(define tsql_relational_pure_slots_from_payload (lambda (contracts payload)
	(map_assoc (tsql_relational_slots_from_payload contracts payload)
		(lambda (_name expression) (sql_type_lower_annotations '() expression)))))
(define tsql_relational_slots (lambda (contracts alias)
	(tsql_relational_slots_from_payload contracts (list 'get_column alias false "__fn_state" false))))
(define tsql_relational_context_from_payload (lambda (contracts payload)
	(merge (mapIndex tsql_scalar_function_context_parameters (lambda (index name)
		(list (string name) (list 'nth payload (+ (count contracts) index))))))))
(define tsql_relational_context_slots (lambda (contracts alias)
	(tsql_relational_context_from_payload contracts (list 'get_column alias false "__fn_state" false))))
/* A factory state has one row: its scoped seed or the current step input.
Pure lexical steps compose on that payload instead of adding another producer.
The lambda argument evaluates the old operand once, even for repeated reads.
SELECT assignment remains the boundary owning row presence and LAST. */
(define tsql_relational_fuse_pure_step (lambda (query contracts slots expressions names) (begin
	(define payload (symbol (tsql_relational_fresh_alias names "__fn_previous_tuple")))
	(define bindings (merge (list (tsql_relational_pure_slots_from_payload contracts payload)
		(tsql_relational_context_from_payload contracts payload))))
	(define packet (cons 'list (cons true (map expressions (lambda (expression)
		(tsql_relational_substitute expression bindings))))))
	(define transition (list 'lambda (list payload)
		(list tsql_relational_state_update payload (list 'quote slots) packet)))
	(define aggregate (match (get_assoc (qb_fields query) "__fn_state")
		((symbol aggregate) operand reducer neutral)
		(list 'aggregate (list transition operand) reducer neutral)
		operand (list transition operand)))
	(make_query_block (qb_schema query) (qb_sources query) (list "__fn_state" aggregate)
		(qb_where query) (qb_group query) (qb_having query) (qb_order query)
		(qb_limit query) (qb_offset query) (qb_hidden query) (qb_stages query) (qb_facts query)))))
(define tsql_relational_argument_leaf? (lambda (expression) (match expression
	((symbol sql_type_expr) value _type _collation) (tsql_relational_argument_leaf? value)
	((symbol sql_compile_column) _compiler value) (tsql_relational_argument_leaf? value)
	((symbol get_column) _alias _alias_ic _column _column_ic) true
	/* A projection of an immutable bounded state tuple is the same lexical
	input as a column read. It introduces no volatile evaluation or query. */
	((symbol nth) payload index)
	(and (number? index) (tsql_relational_argument_leaf? payload))
	((symbol session) key) (and (string? key) (equal? (substr key 0 (strlen "__tsql_parameter:")) "__tsql_parameter:"))
	_ (or (logical_sql_null_literal? expression)
		(and (not (list? expression)) (not (symbol? expression)))))))
(define tsql_relational_authorize (lambda (node policy) (match node
	((symbol quote) _datum) true
	((symbol query-block) _database sources _fields _where _group _having _order _limit _offset _hidden _stages _facts)
	(begin
		(map sources (lambda (src) (begin
			(if (string? (source_relation src)) (policy (source_schema src) (source_relation src) false)
				(tsql_relational_authorize (source_relation src) policy))
			(tsql_relational_authorize (source_join_expr src) policy))))
		(tsql_relational_authorize (list _fields _where _group _having _order _hidden _stages) policy))
	_ (begin (tsql_map_executable_children node (lambda (child)
		(tsql_relational_authorize child policy))) true))))
/* One packet per actual input row. LAST retains a whole packet, including its
presence bit, so a real NULL never becomes an empty-stream fallback. This fold
is legal for unordered assignment; explicit TOP/order stays in its input QB. */
(define tsql_relational_assignment_packet (lambda (database query targets contracts bindings names) (begin
	(define values (map targets (lambda (target) (begin
		(define contract (find contracts (lambda (contract) (equal? (car contract) target)) nil))
		(tsql_implicit_cast_expr (get_assoc (qb_fields query) (concat "__tsql_assignment:" target))
			(car (cadr contract)) (cdr (cadr contract)))))))
	(define packet (cons 'list (cons true values)))
	(define input (make_query_block database (qb_sources query)
		(list "__fn_packet" packet)
		(qb_where query) (qb_group query) (qb_having query) (qb_order query) (qb_limit query) (qb_offset query)
		(qb_hidden query) (qb_stages query)
		(filter (qb_facts query) (lambda (fact) (not (equal? (car fact) 'tsql_select_assignments))))))
	(define input (tsql_relational_substitute input bindings))
	(if (and (not (nil? (qb_limit input))) (not (equal? (qb_limit input) 1)))
		(error "relational scalar TOP currently requires a single-row window") true)
	/* A FROM-less SELECT has at most one packet after WHERE/window selection.
	The scalar consumer preserves its empty-result NULL without another LAST.
	A global aggregate already supplies exactly one assignment packet,
	including COUNT=0 over empty input. An extra LAST over its derived source
	would introduce a second group boundary without a new semantic reduction. */
	(if (or (empty_list? (qb_sources input)) (equal? (qb_limit input) 1)
		(and (query_block_has_local_aggregates? input)
			(empty_list? (qb_group input)) (nil? (qb_having input))
			(nil? (qb_limit input)) (nil? (qb_offset input)))) input (begin
			(define alias (tsql_relational_fresh_alias names "__fn_packet_input"))
			(make_query_block database (list (tsql_relational_source database alias input))
				(list "__fn_packet" (list 'aggregate (list 'get_column alias false "__fn_packet" false)
					'relational_iteration_last_reduce nil)) true nil nil nil nil nil nil nil nil))))))
/* The native recipe stores pure syntax with lexical contracts. Bind it with
this expansion's snapshot guards before slot substitution or query fusion. */
(define tsql_relational_bind_pure_step (lambda (step database policy planning tx) (match step
	'("set" targets expressions)
	(list "set" targets (map expressions (lambda (expression)
		(if (nil? expression) nil (tsql_scalar_function_bind_expression expression database policy planning tx)))))
	'("return" expression)
	(list "return" (tsql_scalar_function_bind_expression expression database policy planning tx))
	'("while" predicate body)
	(list "while" (tsql_scalar_function_bind_expression predicate database policy planning tx) body)
	'("if" predicate yes no quoted)
	(list "if" (tsql_scalar_function_bind_expression predicate database policy planning tx) yes no quoted)
	_ step)))
(define tsql_relational_expand_steps (lambda (recipe steps initial outer_sources policy planning tx meta) (begin
	(define database (recipe "database"))
	(define contracts (recipe "contracts"))
	(define state (newsession))
	(state "query" initial)
	(define answer (newsession))
	(map steps (lambda (step) (if (answer "query") true (begin
		(define step (tsql_relational_bind_pure_step step database policy planning tx))
		(define previous (state "query"))
		(define parsed (if (equal? (car step) "if-query")
			(tsql_relational_scalar_value database (nth step 1) contracts (nth step 4) policy planning tx (recipe "tables"))
			(equal? (car step) "select")
			(tsql_relational_scalar_query database (nth step 2) contracts (nth step 3) policy planning tx (recipe "tables"))
			(equal? (car step) "set-query")
			(tsql_relational_scalar_value database (nth step 2) contracts (nth step 3) policy planning tx (recipe "tables"))
			(equal? (car step) "return-query")
			(tsql_relational_scalar_value database (nth step 1) contracts (nth step 2) policy planning tx (recipe "tables"))
			(equal? (car step) "update-local")
			(tsql_relational_update_query database (nth step 1) contracts (nth step 2) policy planning tx (recipe "tables"))
			(equal? (car step) "insert-local")
			(tsql_relational_insert_query database (nth step 3) contracts (nth step 4) policy planning tx (recipe "tables")) nil))
		(define names (meta "names"))
		(tsql_relational_reserve_names names (list step parsed))
		(define alias (tsql_relational_fresh_alias names (concat "__fn_state_" (meta "step"))))
		(meta "step" (+ (meta "step") 1))
		(define bindings (merge (list (tsql_relational_slots contracts alias)
			(tsql_relational_context_slots contracts alias))))
		(define source (tsql_relational_source database alias previous))
		(meta "captures" (append (meta "captures") source))
		(if (equal? (car step) "return-table")
			(answer "query" previous)
			(equal? (car step) "return") (begin
				/* RETURN consumes one actual predecessor tuple, just as pure SET
				fusion does. Repeating its SQL state-source read for each variable
				would re-run a fresh producer and its volatile arguments. Only
				already typed lexical scalar operations enter the lambda body. */
				(define payload (symbol (tsql_relational_fresh_alias names "__fn_return_tuple")))
				(define payload_bindings (merge (list
					(tsql_relational_pure_slots_from_payload contracts payload)
					(tsql_relational_context_from_payload contracts payload))))
				(define value (list (list 'lambda (list payload)
					(tsql_relational_substitute (cadr step) payload_bindings))
					(list 'get_column alias false "__fn_state" false)))
				(answer "query" (make_query_block database (list source)
					(list "__fn_result" (list 'sql_type_expr value
						(tsql_scalar_function_type (recipe "return")) nil)) true nil nil nil nil nil nil nil nil)))
			(equal? (car step) "return-query") (begin
				/* RETURN of a relational expression needs the same explicit
				predecessor binding as SET-query. Its callee sees immutable tuple
				slots, never SQL operands substituted into generated code heads. */
				(define state_symbol (symbol (tsql_relational_fresh_alias names (concat alias "_return_state"))))
				(define feedback_source (source_with_relation source
					(list 'parameter-rows '("__fn_state") (list (list "__fn_state" state_symbol)) 'all)))
				(meta "captures" (append (meta "captures") feedback_source))
				(define value (tsql_relational_substitute
					(tsql_implicit_cast_expr parsed (car (recipe "return")) (cdr (recipe "return"))) bindings))
				(define returned (make_query_block database (list feedback_source)
					(list "__fn_state" (list 'list value)) true nil nil nil nil nil nil nil nil))
				(define invocation (make_relational_sequence previous returned state_symbol outer_sources))
				(define return_alias (tsql_relational_fresh_alias names (concat alias "_return")))
				(answer "query" (make_query_block database
					(list (tsql_relational_source database return_alias invocation))
					(list "__fn_result" (list 'sql_type_expr
						(list 'nth (list 'get_column return_alias false "__fn_state" false) 0)
						(tsql_scalar_function_type (recipe "return")) nil))
					true nil nil nil nil nil nil nil nil)))
			(or (equal? (car step) "while") (has? '("if" "if-query") (car step))) (begin
				(define feedback_alias (tsql_relational_fresh_alias names (concat alias "_input")))
				(define state_symbol (symbol (tsql_relational_fresh_alias names (concat alias "_iteration"))))
				(define feedback_source (list feedback_alias database
					(list 'parameter-rows '("__fn_state")
						(list (list "__fn_state" state_symbol)) 'all) false nil nil))
				(meta "captures" (append (meta "captures") feedback_source))
				(define feedback (make_query_block database (list feedback_source)
					(list "__fn_state" (list 'get_column feedback_alias false "__fn_state" false))
					true nil nil nil nil nil nil nil nil))
				(define predicate (if (equal? (car step) "if-query")
					(make_query_block database (list feedback_source)
						(list "__fn_state" (list 'list (tsql_relational_substitute parsed
							(merge (list (tsql_relational_slots contracts feedback_alias)
								(tsql_relational_context_slots contracts feedback_alias))))))
						true nil nil nil nil nil nil nil nil)
					(tsql_relational_substitute (cadr step)
						(merge (list (tsql_relational_pure_slots_from_payload contracts state_symbol)
							(tsql_relational_context_from_payload contracts state_symbol))))))
				(define branch_predicate (if (equal? (car step) "if-query") predicate
					(make_query_block database (list feedback_source)
						(list "__fn_state" (list 'list predicate)) true nil nil nil nil nil nil nil nil)))
				(define next (tsql_relational_expand_steps recipe (nth step 2) feedback outer_sources policy planning tx meta))
				(state "query" (if (equal? (car step) "while")
					(make_relational_iteration previous next state_symbol outer_sources predicate)
					(make_relational_choice previous next
						(tsql_relational_expand_steps recipe (nth step 3) feedback outer_sources policy planning tx meta)
						state_symbol outer_sources branch_predicate))))
			(equal? (car step) "update-local") (begin
				/* The predecessor is one immutable state tuple, bound before the
				whole native mutation. Its row expressions remain ordinary SQL. */
				(define state_symbol (symbol (tsql_relational_fresh_alias names (concat alias "_update_state"))))
				(define payload_bindings (merge (list
					(tsql_relational_pure_slots_from_payload contracts state_symbol)
					(tsql_relational_context_from_payload contracts state_symbol))))
				(define input (tsql_relational_substitute parsed payload_bindings))
				(state "query" (make_relational_sequence previous
					(make_local_relation_update input state_symbol outer_sources) state_symbol outer_sources)))
			(equal? (car step) "insert-local") (begin
				(define declaration (find (recipe "tables") (lambda (table) (equal? (car table) (cadr step))) nil))
				/* The payload may be a FROM-less projection of local variables.
				Bind their predecessor once before entering that relational child. */
				(define state_symbol (symbol (tsql_relational_fresh_alias names (concat alias "_insert_state"))))
				(define feedback_source (source_with_relation source
					(list 'parameter-rows '("__fn_state")
						(list (list "__fn_state" state_symbol)) 'all)))
				(meta "captures" (append (meta "captures") feedback_source))
				(define payload_bindings (merge (list
					(tsql_relational_pure_slots_from_payload contracts state_symbol)
					(tsql_relational_context_from_payload contracts state_symbol))))
				(define input (tsql_relational_insert_input database feedback_source parsed declaration
					(nth step 2) payload_bindings names))
				(state "query" (make_relational_sequence previous
					(make_local_relation_insert (car input) (nth declaration 1) (cadr input) outer_sources)
					state_symbol outer_sources)))
			(begin
				(define targets (cadr step))
				(define slots (map targets (lambda (target)
					(find (produceN (count contracts)) (lambda (index) (equal? (car (nth contracts index)) target)) nil))))
				(if (equal? (car step) "set")
					(state "query" (tsql_relational_fuse_pure_step previous contracts slots (nth step 2) names))
					(begin
						/* Bind the predecessor once before entering the assignment
						query. Its locals are lexical scalar inputs to this child. */
						(define state_symbol (symbol (tsql_relational_fresh_alias names (concat alias "_sequence"))))
						(define feedback_source (source_with_relation source
							(list 'parameter-rows '("__fn_state")
								(list (list "__fn_state" state_symbol)) 'all)))
						(meta "captures" (append (meta "captures") feedback_source))
						(define packet (if (equal? (car step) "set-query") (begin
							(define contract (cadr (find contracts (lambda (item) (equal? (car item) (car targets))) nil)))
							(list 'list true (tsql_relational_substitute
								(tsql_implicit_cast_expr parsed (car contract) (cdr contract)) bindings)))
							(list 'inner_select (tsql_relational_assignment_packet
								database (cadr parsed) targets contracts bindings names))))
						(define update (list tsql_relational_state_update
							(list 'get_column alias false "__fn_state" false) (list 'quote slots) packet))
						/* The previous state and assignment packet each have one row.
						Project their transition; another LAST would introduce a
						group barrier without changing its cardinality. */
						(state "query" (make_relational_sequence previous
							(make_query_block database (list feedback_source)
								(list "__fn_state" update) true nil nil nil nil nil nil nil nil)
							state_symbol outer_sources))))))))))
	(coalesceNil (answer "query") (state "query")))))
(define tsql_relational_scalar_expand (lambda (recipe arguments outer_sources policy planning tx) (begin
	(define planning (tsql_ansi_planning_scope planning (recipe "ansi_nulls")))
	(define planning (tsql_scalar_function_binding_scope planning (recipe "identity")))
	(define database (recipe "database"))
	(define parameters (recipe "parameters"))
	(define contracts (recipe "contracts"))
	/* A bounded initial packet owns scalar arguments and their conversions.
	Each actual argument is evaluated once before any state/body statement;
	later reads consume its slot. Relational argument producers remain a
	separate logical domain and must not become recursive scalar callbacks. */
	(define values (mapIndex arguments (lambda (index argument) (begin
		(if (or (expr_contains_subquery? argument) (tsql_scalar_function_reference? argument))
			(error "relational function arguments require an explicit scalar input domain") true)
		(define target (tsql_scalar_function_declaration (cadr (nth parameters index))))
		(tsql_implicit_cast_expr argument (car target) (cdr target))))))
	(define local_count (- (count contracts) (count parameters)))
	(define meta (newsession))
	(meta "captures" outer_sources)
	(meta "step" 0)
	(define names (tsql_relational_name_context
		(list recipe arguments (map outer_sources binding_source_string_inputs))))
	(meta "names" names)
	(define argument_symbols (mapIndex values (lambda (index _value)
		(symbol (tsql_relational_fresh_alias names (concat "__fn_argument_" index))))))
	(define context_values (tsql_scalar_function_context_arguments))
	(define context_symbols (mapIndex context_values (lambda (index _value)
		(symbol (tsql_relational_fresh_alias names (concat "__fn_context_" index))))))
	/* SQL operands stay in the argument positions seen by normal name and
	column lowering. The scalar tuple constructor itself has lexical values
	only, including context captured by a nested caller. */
	(define initial (cons (list 'lambda (merge (list argument_symbols context_symbols))
		(cons 'list (merge (list argument_symbols
			(map (produceN local_count) (lambda (_index) nil)) context_symbols))))
		(merge (list values context_values))))
	/* This unconditional FROM-less projection owns exactly one state tuple.
	The scoped sequence/choice/iteration consumer binds its seed once; pure
	DECLARE/SET composition retains its single-argument lambda binding. */
	(define seed (make_query_block database '() (list "__fn_state" initial)
		true nil nil nil nil nil nil nil nil))
	(tsql_relational_reserve_names names seed)
	/* Recipe validation covered the complete source. Expansion follows the
	first RETURN and never constructs unreachable later query/state steps. */
	(define expanded (tsql_relational_expand_steps recipe (recipe "steps")
		seed outer_sources policy planning tx meta))
	(define expanded (if (empty_list? (recipe "tables")) expanded
		(begin
			/* The lifetime encloses the complete state program. RETURN consumes
			its bounded scalar tuple after teardown; native handles cannot escape
			through either a state slot or the scalar callable's public result. */
			(define source (car (qb_sources expanded)))
			(define scoped (make_local_relation_scope (source_relation source)
				(map (recipe "tables") (lambda (declaration)
					(list (nth declaration 1) database (nth declaration 2) (nth declaration 3)))) outer_sources))
			(make_query_block database (list (source_with_relation source scoped))
				(qb_fields expanded) (qb_where expanded) (qb_group expanded) (qb_having expanded)
				(qb_order expanded) (qb_limit expanded) (qb_offset expanded)
				(qb_hidden expanded) (qb_stages expanded) (qb_facts expanded)))))
	(define query (tsql_relational_mark_capture_scopes expanded (source_aliases (meta "captures"))))
	(tsql_relational_authorize query policy)
	(list 'inner_select (sql_bind_catalog_declarations
		(tsql_expand_relational_scalar_calls (sql_expand_views query policy planning tx)
			outer_sources policy planning tx) tx planning)))))
(define tsql_relational_scalar_compile (lambda (source context quoted) (begin
	(define recipe (tsql_relational_scalar_recipe source context quoted))
	/* The native source catalog receives a real callable. This planning factory
	captures immutable frontend syntax only; invocation resources are arguments. */
	(close_procedure ((lambda (recipe expand)
		(lambda (arguments outer_sources policy planning tx)
			(expand recipe arguments outer_sources policy planning tx))) recipe tsql_relational_scalar_expand)))))
(tsql_register_source_profiles "tsql-relational-scalar-function" tsql_relational_scalar_compile registerprocedurelanguage)
/* Qualify admitted argument leaves before adding any state/unit sources.
Keep lexical levels separate: an inner caller scope shadows its parent, while
native binding entries retain the real source aliases used by the query. */
(define tsql_relational_bind_arguments (lambda (arguments source_scopes) (begin
	(define path "relational-scalar-arguments")
	/* This qualification pass preserves each source identity: every entry is
	bound at the root path with no alpha-renaming outer scope. No fresh alias
	can be generated here, so collecting all strings from derived source bodies
	would traverse the same inlined views again for every function call. */
	(define all_strings (list '() '() path))
	(define scopes (map source_scopes (lambda (sources)
		(binding_scope_entries sources '() path all_strings 0))))
	(map arguments (lambda (argument) (bind_query_expr scopes argument path all_strings))))))
(define tsql_expand_relational_scalar_calls_scoped (lambda (node source_scopes policy planning tx) (match node
	((symbol quote) _datum) node
	((symbol tsql_relational_scalar_call) factory arguments)
	/* A factory introduces new logical sources after the enclosing query's
	view pass. Expand those sources through the same owner before binding;
	already expanded metadata catalogs retain their native-branch boundary. */
	(sql_expand_views
		(factory (tsql_relational_bind_arguments arguments source_scopes) (merge source_scopes) policy planning tx)
		policy planning tx)
	((symbol union-block) mode branches ordering limit offset facts)
	(begin
		(define expand (lambda (child)
			(tsql_expand_relational_scalar_calls_scoped child source_scopes policy planning tx)))
		(make_union_block mode (map branches expand)
			(map ordering (lambda (item) (list (expand (car item)) (cadr item)))) limit offset facts))
	((symbol query-block) database sources fields condition grouping having ordering limit offset hidden stages facts)
	(begin
		(define sources (map sources (lambda (src)
			(source_with_relation src (tsql_expand_relational_scalar_calls_scoped (source_relation src)
				(if (source_lateral? src)
					(cons (source_prefix_for_alias sources (source_alias src)) source_scopes) '()) policy planning tx)))))
		(define scope (cons sources source_scopes))
		(define expand (lambda (child)
			(tsql_expand_relational_scalar_calls_scoped child scope policy planning tx)))
		/* Containers are not call expressions: their first child is executable
		too. Facts retain opaque declarations and provenance, never callsites. */
		(define units (merge (list (qassoc_get facts 'nullable_join_scopes '())
			(qassoc_get facts 'join_operand_scopes '()))))
		(make_query_block database (map sources (lambda (src) (begin
			(define visible (append (source_prefix_for_alias sources (source_alias src)) src))
			(define aliases (join_predicate_scope_aliases (source_alias src) (source_aliases visible) units))
			(define scoped (filter sources (lambda (candidate) (has? aliases (source_alias candidate)))))
			(source_with_join_expr src (tsql_expand_relational_scalar_calls_scoped
				(source_join_expr src) (cons scoped source_scopes) policy planning tx)))))
			(map_assoc fields (lambda (_title expression) (expand expression)))
			(expand condition) (map grouping expand) (expand having)
			(map ordering (lambda (item) (list (expand (car item)) (cadr item))))
			limit offset (map_assoc hidden (lambda (_title expression) (expand expression)))
			(map stages expand) facts))
	_ (tsql_map_executable_children node (lambda (child)
		(tsql_expand_relational_scalar_calls_scoped child source_scopes policy planning tx))))))
(define tsql_expand_relational_scalar_calls (lambda (node outer_sources policy planning tx)
	(tsql_expand_relational_scalar_calls_scoped node
		(if (empty_list? outer_sources) '() (list outer_sources)) policy planning tx)))
