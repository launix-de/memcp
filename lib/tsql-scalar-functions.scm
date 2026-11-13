/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */
/* Scalar modules share the native source catalog and its transaction lifetime.
Their compiled recipe has only typed lexical inputs. No caller session or
request transaction is retained in the callable shipped to scan workers. */
(define tsql_scalar_function_language? (lambda (language)
	(equal? (tsql_source_module_kind language) "FN")))
(define tsql_scalar_function_variable (lambda (name)
	(symbol (concat "__tsql_scalar_function_" name))))
(define tsql_scalar_function_declaration (lambda (contract) (begin
	(define effective (tsql_effective_parameter_contract contract))
	(if (has? '("BIT" "TINYINT" "SMALLINT" "INT" "INTEGER" "BIGINT" "VARCHAR" "NVARCHAR"
		"CHAR" "NCHAR" "DECIMAL" "NUMERIC" "FLOAT" "REAL" "MONEY" "SMALLMONEY"
		"DATE" "DATETIME" "DATETIME2" "SMALLDATETIME") (car effective)) effective
		(error "unsupported scalar function declaration: " (car contract))))))
(define tsql_scalar_function_type (lambda (contract)
	(if (empty_list? (cdr contract)) (car contract) (list (car contract) (cdr contract)))))
(define tsql_scalar_function_convert (lambda (info contract)
	(sql_type_formula '() (tsql_implicit_cast_expr
		(list 'sql_type_expr (sql_info_formula info) (sql_info_datatype info) (sql_info_collation info))
		(car contract) (cdr contract)))))
/* Hidden invocation operands are plain scalar context, separate from the
native declaration's SQL parameter header. Read session keys only at callsites;
module compilation neither reads nor retains a creation-time session. */
(define tsql_scalar_function_context_inputs (list
	(list '__tsql_scalar_dateformat "__tsql_dateformat" (tsql_dateformat_formula))
	(list '__tsql_scalar_language "__tsql_language" (tsql_language_formula))
	(list '__tsql_scalar_datefirst "__tsql_datefirst" (list 'coalesceNil (list 'session "__tsql_datefirst") 7))
	(list '__tsql_scalar_appname "__tsql_client_appname" (list 'session "__tsql_client_appname"))
	(list '__tsql_scalar_hostname "__tsql_client_hostname" (list 'session "__tsql_client_hostname"))
	(list '__tsql_scalar_username "username" (list 'session "username"))
	(list '__tsql_scalar_principal "__tsql_principal_id" (list 'session "__tsql_principal_id"))))
(define tsql_scalar_function_context_parameters (map tsql_scalar_function_context_inputs car))
(define tsql_scalar_function_context_arguments (lambda ()
	(map tsql_scalar_function_context_inputs (lambda (input) (nth input 2)))))
(define tsql_scalar_function_bind_context (lambda (formula) (match formula
	((symbol quote) _value) formula
	((symbol session) key) (begin
		(define input (find tsql_scalar_function_context_inputs (lambda (input) (equal? (cadr input) key)) nil))
		(if (nil? input) formula (car input)))
	(cons head arguments) (cons (tsql_scalar_function_bind_context head)
		(map arguments tsql_scalar_function_bind_context))
	_ formula)))
(define tsql_scalar_function_context_free? (lambda (formula) (match formula
	((symbol quote) _value) true
	(cons head tail)
	(and (tsql_scalar_function_context_free? head)
		(reduce tail (lambda (valid child)
			(and valid (tsql_scalar_function_context_free? child))) true))
	(symbol session) false
	(symbol session_globalvar) false
	(symbol tx) false
	(symbol resultrow) false
	(symbol resultfields) false
	_ true)))
(define tsql_scalar_function_validate_syntax (lambda (formula contracts) (match formula
	((symbol quote) _value) false
	((symbol tsql_batch_variable) name)
	(if (find contracts (lambda (contract) (equal? (car contract) name)) nil) false
		(error "undeclared T-SQL parameter: @" name))
	(cons head tail) (begin
		(define children (reduce tail (lambda (calls child)
			(or (tsql_scalar_function_validate_syntax child contracts) calls)) false))
		(define head_calls (tsql_scalar_function_validate_syntax head contracts))
		(or children head_calls (equal? head "scalar-function-call")))
	_ false)))
/* Native source admission retains syntax and the lexical declaration at each
expression. Callees can still be staged in the creating transaction; their
declarations and SQL types are resolved only by the consuming SQL plan. */
(define tsql_scalar_function_expression (lambda (database text contracts target quoted ansi) (begin
	(define planning (newsession))
	(planning "__tsql_expression_only" true)
	(planning "__tsql_expression_syntax" true)
	(planning "__tsql_script_syntax" true)
	(planning "__tsql_scalar_function_body" true)
	(planning "__tsql_parameter_declarations" contracts)
	(planning "__tsql_quoted_identifier" quoted)
	(planning "__tsql_ansi_nulls" ansi)
	(define syntax (parse_tsql database text nil planning nil))
	(define calls (tsql_scalar_function_validate_syntax syntax contracts))
	(define expression (list 'tsql_deferred_scalar_expression syntax contracts target))
	/* Independent expressions keep their existing admission-time type checks.
	A declaration-dependent expression waits for the actual callee contracts. */
	(if calls true (tsql_scalar_function_bind_expression expression database nil planning nil))
	expression)))
(define tsql_scalar_function_bind_ast (lambda (syntax database contracts policy planning tx) (match syntax
	((symbol quote) _value) syntax
	((symbol tsql_batch_variable) name) (begin
		(define declaration (find contracts (lambda (contract) (equal? (car contract) name)) nil))
		(if (nil? declaration) (error "undeclared T-SQL parameter: @" name) true)
		(list 'sql_type_expr (tsql_scalar_function_variable name)
			(tsql_scalar_function_type (cadr declaration)) nil))
	'("scalar-function-call" name arguments)
	(tsql_scalar_function_call database name (map arguments (lambda (argument)
		(tsql_scalar_function_bind_ast argument database contracts policy planning tx))) policy planning tx true)
	(cons head tail)
	(cons (tsql_scalar_function_bind_ast head database contracts policy planning tx)
		(map tail (lambda (child) (tsql_scalar_function_bind_ast child database contracts policy planning tx))))
	_ syntax)))
(define tsql_scalar_function_bind_expression (lambda (expression database policy planning tx)
	(match expression
		((symbol tsql_deferred_scalar_expression) syntax contracts target) (begin
			(define info (sql_expr_info '()
				(tsql_scalar_function_bind_ast syntax database contracts policy planning tx)))
			(if (and (nil? target) (not (equal? (sql_info_type info) "BOOLEAN")))
				(error "scalar function IF requires a Boolean predicate") true)
			/* A read of our own typed lexical variable already satisfies that exact
			declaration: parameters and assignments performed its conversion. Keep
			arithmetic and other expressions on the ordinary checked conversion path. */
			(define declared_read (and (not (nil? target))
				(find contracts (lambda (contract)
					(and (expression_equal? (sql_info_formula info) (tsql_scalar_function_variable (car contract)))
						(expression_equal? (cadr contract) target))) nil)))
			(define formula (tsql_scalar_function_bind_context (if (or (nil? target) declared_read) (sql_info_formula info)
				(tsql_scalar_function_convert info target))))
			(if (and (tsql_scalar_expression_allowed? formula) (tsql_scalar_function_context_free? formula))
				formula (error "scalar function expression requires relational or session state")))
		_ (error "invalid deferred scalar expression"))))
/* Statement recipes retain scalar syntax with its lexical declarations.
Typed formulas are bound for the consuming plan; initializers still execute
only on their original control-flow path. */
(define tsql_scalar_function_recipes (lambda (commands database state return_type quoted)
	(merge (map commands (lambda (command) (match command
		'("block" statements)
		(tsql_scalar_function_recipes statements database state return_type quoted)
		'("declare" declarations) (map declarations (lambda (declaration) (begin
			(define name (car declaration))
			(define existing (state "contracts"))
			(if (find existing (lambda (item) (equal? (car item) name)) nil)
				(error "duplicate scalar function variable: @" name) true)
			(define contract (tsql_scalar_function_declaration (cadr declaration)))
			(define initializer (nth declaration 2))
			(define value (if (nil? initializer) nil
				(tsql_scalar_function_expression database initializer existing contract quoted (state "ansi_nulls"))))
			(state "contracts" (append existing (list name contract)))
			(list "bind" (tsql_scalar_function_variable name) value))))
		'("assign" name operation expression) (begin
			(if operation (error "scalar function requires simple SET assignment") true)
			(define existing (state "contracts"))
			(define contract (find existing (lambda (item) (equal? (car item) name)) nil))
			(if (nil? contract) (error "undeclared scalar function variable: @" name) true)
			(list (list "bind" (tsql_scalar_function_variable name)
				(tsql_scalar_function_expression database expression existing (cadr contract) quoted (state "ansi_nulls")))))
		'("if" condition yes no) (begin
			(define predicate (tsql_scalar_function_expression database condition (state "contracts") nil quoted (state "ansi_nulls")))
			(define yes_recipe (tsql_scalar_function_recipes (list yes) database state return_type quoted))
			(define no_recipe (if (nil? no) '()
				(tsql_scalar_function_recipes (list no) database state return_type quoted)))
			(list (list "if" predicate yes_recipe no_recipe)))
		'("while" condition body) (begin
			(define predicate (tsql_scalar_function_expression database condition (state "contracts") nil quoted (state "ansi_nulls")))
			(define depth (coalesceNil (state "loop_depth") 0))
			(state "loop_depth" (+ depth 1))
			(define body_recipe (tsql_scalar_function_recipes (list body) database state return_type quoted))
			(state "loop_depth" depth)
			(list (list "while" predicate body_recipe)))
		'("break") (begin
			(if (> (coalesceNil (state "loop_depth") 0) 0) true
				(error "BREAK requires an enclosing scalar function WHILE"))
			(list (list "break")))
		'("return" expression) (begin
			(if (nil? expression) (error "scalar function requires a return expression") true)
			(list (list "return"
				(tsql_scalar_function_expression database expression (state "contracts") return_type quoted (state "ansi_nulls")))))
		_ (error "unsupported statement in scalar function body")))))))
/* Branches share their continuation instead of duplicating the rest of the
body. The continuation receives plain lexical values, including unchanged
variables and SQL NULL for a local declared only on an untaken path. */
(define tsql_scalar_function_lower (lambda (recipes symbols next counter break_target return_target)
	(reduce (reverse recipes) (lambda (next recipe) (match recipe
		'("bind" variable value) (list (list 'lambda (list variable) next) value)
		'("return" value) (if (nil? return_target) value (return_target value))
		'("if" predicate yes no) (begin
			(define id (coalesceNil (counter "id") 0))
			(counter "id" (+ id 1))
			(define continuation (symbol (concat "__tsql_scalar_branch_continuation_" id)))
			(define invoke (cons continuation symbols))
			(list (list 'lambda (list continuation)
				(list 'if predicate
					(tsql_scalar_function_lower yes symbols invoke counter break_target return_target)
					(tsql_scalar_function_lower no symbols invoke counter break_target return_target)))
				(list 'lambda symbols next)))
		'("break") break_target
		'("while" predicate body) (begin
			/* The lexical exit flag belongs to this loop alone. BREAK returns
			its current typed values without evaluating the remaining body. */
			(define id (coalesceNil (counter "id") 0))
			(counter "id" (+ id 1))
			(define running (symbol (concat "__tsql_scalar_loop_running_" id)))
			(define returned (symbol (concat "__tsql_scalar_loop_returned_" id)))
			(define result (symbol (concat "__tsql_scalar_loop_result_" id)))
			(define parameters (merge (list (list running returned result) symbols)))
			(define values (cons 'list (merge (list (list true false nil) symbols))))
			(define stopped (cons 'list (merge (list (list false false nil) symbols))))
			/* The terminal bit distinguishes RETURN NULL from ordinary exhaustion.
			An enclosing loop receives the same terminal result and current locals. */
			(define terminal (lambda (value)
				(cons 'list (merge (list (list false true value) symbols)))))
			(define iteration (list 'for values
				(list 'lambda parameters (list 'and running predicate))
				(list 'lambda parameters
					(tsql_scalar_function_lower body symbols values counter stopped terminal))))
			(list 'apply (list 'lambda parameters
				(list 'if returned (if (nil? return_target) result (return_target result)) next)) iteration))
		_ (error "unknown scalar function recipe"))) next)))
/* Parameter defaults are literal declaration syntax. They are admitted without
caller bindings and receive the ordinary parameter conversion at the call site. */
(define tsql_scalar_function_default_info (lambda (parameter) (begin
	(if (<= (count parameter) 2) (error "scalar function parameter has no default: @" (car parameter)) true)
	(define text (nth parameter 2))
	(if (string? text) true (error "scalar function default must be a literal"))
	(define literal ((parser '((define value tsql_parameter_default_literal) (regex "$")) value) text))
	(sql_expr_info '() literal))))
(define tsql_scalar_function_bind_recipes (lambda (recipes database policy planning tx)
	(map recipes (lambda (recipe) (match recipe
		'("bind" variable expression)
		(list "bind" variable (if (nil? expression) nil
			(tsql_scalar_function_bind_expression expression database policy planning tx)))
		'("return" expression)
		(list "return" (tsql_scalar_function_bind_expression expression database policy planning tx))
		'("if" predicate yes no)
		(list "if" (tsql_scalar_function_bind_expression predicate database policy planning tx)
			(tsql_scalar_function_bind_recipes yes database policy planning tx)
			(tsql_scalar_function_bind_recipes no database policy planning tx))
		'("while" predicate body)
		(list "while" (tsql_scalar_function_bind_expression predicate database policy planning tx)
			(tsql_scalar_function_bind_recipes body database policy planning tx))
		_ recipe)))))
/* Binding state is lexical to this expansion. Guard and authorization writes
still reach the consuming plan, while sibling calls retain independent paths. */
(define tsql_scalar_function_binding_scope (lambda (parent identity) (begin
	(define path (coalesceNil (parent "__tsql_scalar_function_binding_path") '()))
	(if (has? path identity) (error "recursive scalar function binding requires a recursion contract") true)
	(define path (cons identity path))
	(lambda args (match args
		'(key) (if (equal? key "__tsql_scalar_function_binding_path") path (parent key))
		_ (apply parent args))))))
(define tsql_scalar_function_bind (lambda (recipe policy planning tx) (begin
	(define planning (tsql_scalar_function_binding_scope planning (recipe "identity")))
	(define recipes (tsql_scalar_function_bind_recipes (recipe "steps") (car (recipe "identity")) policy planning tx))
	(define body (tsql_scalar_function_lower recipes (recipe "symbols") nil (newsession) nil nil))
	(define locals (recipe "locals"))
	(define body (if (empty_list? locals) body
		(cons (list 'lambda locals body) (map locals (lambda (_variable) nil)))))
	(close_procedure (eval (list 'lambda (recipe "arguments") body))))))
(define tsql_build_scalar_function_recipe (lambda (source context quoted) (begin
	(define database (context "schema"))
	(define signature (context "parameters"))
	(define parameters (tsql_unique_parameters (car signature)))
	(define return_type (tsql_scalar_function_declaration (cadr signature)))
	(define contracts (map parameters (lambda (parameter) (begin
		(define contract (tsql_scalar_function_declaration (cadr parameter)))
		(if (> (count parameter) 2)
			(tsql_scalar_function_convert (tsql_scalar_function_default_info parameter) contract) true)
		(list (car parameter) contract)))))
	(define batch (cadr ((tsql_parse_batch_context database source quoted true (coalesceNil (context "__tsql_ansi_nulls") true)) "syntax")))
	(if (equal? (count batch) 1) true (error "scalar function requires one BEGIN/END body"))
	(define commands (match (car batch)
		'("block" statements) statements
		_ (error "scalar function requires a BEGIN/END body")))
	(if (empty_list? commands) (error "scalar function requires a terminal RETURN") true)
	(match (car (reverse commands))
		'("return" expression) (if (nil? expression) (error "scalar function requires a return expression") true)
		_ (error "scalar function requires a terminal RETURN"))
	(define state (newsession))
	(state "contracts" contracts)
	(state "ansi_nulls" (coalesceNil (context "__tsql_ansi_nulls") true))
	(define recipes (tsql_scalar_function_recipes commands database state return_type quoted))
	(define symbols (map (state "contracts") (lambda (contract)
		(tsql_scalar_function_variable (car contract)))))
	(define arguments (merge (list (map parameters (lambda (parameter)
		(tsql_scalar_function_variable (car parameter)))) tsql_scalar_function_context_parameters)))
	(define locals (filter symbols (lambda (variable) (not (has? arguments variable)))))
	(list "identity" (list database (context "name")) "steps" recipes
		"symbols" symbols "arguments" arguments "locals" locals))))
/* Recipe reuse owns syntax only. Snapshot resolution, permissions and every
transitive source guard still belong to the consuming SQL compilation. Copy
only the portable source header: a native compiler may supply more context,
but neither that object nor its creation frame belongs in this cache. */
(define tsql_scalar_function_recipe (lambda (source context quoted) (begin
	(define header (list "schema" (context "schema") "name" (context "name")
		"parameters" (context "parameters")
		"__tsql_ansi_nulls" (coalesceNil (context "__tsql_ansi_nulls") true)))
	(define key (json_encode (list "scalar-function-recipe" source header quoted)))
	(define recipe (tsql_script_cache key))
	(if (not (nil? recipe)) recipe
		(tsql_script_cache key nil (close_procedure (lambda ()
			(tsql_build_scalar_function_recipe source header quoted))))))))

(define tsql_scalar_function_compile (lambda (source context quoted) (begin
	(define recipe (tsql_scalar_function_recipe source context quoted))
	(close_procedure ((lambda (recipe bind)
		(lambda (policy planning tx) (bind recipe policy planning tx))) recipe tsql_scalar_function_bind)))))
(tsql_register_source_profiles "tsql-scalar-function" tsql_scalar_function_compile registerprocedurelanguage)
(define tsql_scalar_function_definition (lambda (database name tx)
	(find (showprocedures database (toLower name) tx) (lambda (_definition) true) nil)))
(define tsql_scalar_function_reserved? (lambda (database name tx) (begin
	(define definition (tsql_scalar_function_definition database name tx))
	(if (nil? definition) false
		(has? '("FN" "TF") (tsql_source_module_kind (definition "language")))))))
(define tsql_define_scalar_function (lambda (name parameters return_type source statement session)
	(statement "scalar function definition" false (lambda (tx) (begin
		(if (or (session "__tsql_insert_exec") (session "__tsql_trigger_event"))
			(error "scalar function DDL is unsupported inside a native mutation") true)
		(define database (session "schema"))
		(tsql_require_database_access (tsql_session_actor session) database)
		(if (or (show (table database false name true tx) "name") (tsql_view_name database name))
			(error "a relation already owns this object name: " name) true)
		(define quoted (tsql_quoted_identifier session))
		(define relational (tsql_relational_scalar_source? source database quoted (tsql_ansi_nulls session)))
		(createprocedure database (toLower name)
			(tsql_source_language (if relational "tsql-relational-scalar-function" "tsql-scalar-function") session)
			source (list parameters return_type) false tx)
		0)))))
/* Functions share the native routine catalog, but DROP must retain its SQL
object kind. Validate every declared target before issuing native deletions. */
(define tsql_drop_scalar_functions (lambda (names ignore statement session)
	(statement "scalar function deletion" false (lambda (tx) (begin
		(if (or (session "__tsql_insert_exec") (session "__tsql_trigger_event"))
			(error "scalar function DDL is unsupported inside a native mutation") true)
		(define database (session "schema"))
		(tsql_require_database_access (tsql_session_actor session) database)
		(define definitions (map names (lambda (name) (begin
			(define definition (tsql_scalar_function_definition database name tx))
			(if (nil? definition)
				(if ignore nil (error "function does not exist: " name))
				(has? '("FN" "TF") (tsql_source_module_kind (definition "language"))) definition
				(error "object is not a scalar function: " name))))))
		(reduce definitions (lambda (_result definition)
			(if (nil? definition) false (dropprocedure database (definition "name") ignore tx))) false)
		0)))))
(define tsql_scalar_function_compiler (lambda (callable parameters return_type)
	(lambda (infos)
		(sql_info (cons callable (merge (list
			(mapIndex infos (lambda (index info)
				(tsql_scalar_function_convert info (tsql_scalar_function_declaration (cadr (nth parameters index))))))
			(tsql_scalar_function_context_arguments))))
			(tsql_scalar_function_type return_type) nil))))
(define tsql_scalar_function_reference? (lambda (formula) (match formula
	((symbol quote) _value) false
	((symbol tsql_relational_scalar_call) _factory _arguments) true
	(cons (symbol sql_compile_expr) (cons ((symbol tsql_scalar_function_compiler) _callable _params _return) _arguments)) true
	(cons head tail)
	(or (tsql_scalar_function_reference? head)
		(reduce tail (lambda (found child)
			(or found (tsql_scalar_function_reference? child))) false))
	_ false)))
/* Guard the same immutable source/declaration snapshot that supplied the
callable. A warm plan never resolves a different body or argument contract
inside a scan callback. The authorization guard is owned by the SQL planner. */
(define tsql_scalar_function_snapshot (lambda (database name policy planning tx) (begin
	(define name (toLower name))
	(policy database name false)
	(define snapshot (procedure database name true tx))
	(define definition (snapshot "declaration"))
	(if (tsql_scalar_function_language? (definition "language")) true
		(error "object is not a scalar function: " name))
	(planner_record_guard_condition
		(list 'expression_equal? (list 'showprocedures database name 'tx) (list 'quote (list definition))) planning)
	snapshot)))
(define tsql_scalar_function_source_context (lambda (database definition)
	(list "schema" database "name" (definition "name") "parameters" (definition "parameters")
		"__tsql_ansi_nulls" (cadr (tsql_source_language_settings (definition "language"))))))
(define tsql_scalar_function_source_quoted? (lambda (definition)
	(car (tsql_source_language_settings (definition "language")))))
/* These are parser-owned expression references, not a scan over SQL strings.
Every transitive snapshot contributes its ordinary permission/version guard. */
(define tsql_scalar_function_recipe_calls (lambda (node) (match node
	((symbol quote) _data) '()
	'("scalar-function-call" name arguments)
	(merge_unique (list (list name) (merge (map arguments tsql_scalar_function_recipe_calls))))
	(cons head tail) (merge_unique (list (tsql_scalar_function_recipe_calls head)
		(merge (map tail tsql_scalar_function_recipe_calls))))
	_ '())))
(define tsql_scalar_function_needs_relation? (lambda (database definition policy planning tx)
	(if (tsql_relational_scalar_language? (definition "language")) true (begin
		(define scope (tsql_scalar_function_binding_scope planning (list database (definition "name"))))
		(define recipe (tsql_scalar_function_recipe (definition "source")
			(tsql_scalar_function_source_context database definition) (tsql_scalar_function_source_quoted? definition)))
		(reduce (tsql_scalar_function_recipe_calls (recipe "steps")) (lambda (found name) (begin
			(define snapshot (tsql_scalar_function_snapshot database name policy scope tx))
			(or found (tsql_scalar_function_needs_relation? database (snapshot "declaration") policy scope tx)))) false)))))
(define tsql_scalar_function_call (lambda (database name arguments policy planning tx scalar_only) (begin
	(define name (toLower name))
	(define snapshot (tsql_scalar_function_snapshot database name policy planning tx))
	(define definition (snapshot "declaration"))
	(define relational (tsql_scalar_function_needs_relation? database definition policy planning tx))
	(if (and scalar_only relational) (error "scalar function expression requires a relational plan") true)
	(define signature (definition "parameters"))
	(define parameters (car signature))
	(define return_type (tsql_scalar_function_declaration (cadr signature)))
	(if (equal? (count arguments) (count parameters)) true (error "wrong number of scalar function arguments"))
	(define resolved (mapIndex arguments (lambda (index argument) (match argument
		'("scalar-parameter-default") (begin
			(define info (tsql_scalar_function_default_info (nth parameters index)))
			(list 'sql_type_expr (sql_info_formula info) (sql_info_datatype info) (sql_info_collation info)))
		_ argument))))
	(define callable (if relational
		(if (tsql_relational_scalar_language? (definition "language")) (snapshot "callable")
			/* A syntactically scalar wrapper can depend on a relational callee.
			Compile the same guarded immutable source through the relational
			factory; no SQL is parsed or dispatched by a row callback. */
			(tsql_relational_scalar_compile (definition "source")
				(tsql_scalar_function_source_context database definition)
				(tsql_scalar_function_source_quoted? definition)))
		((snapshot "callable") policy planning tx)))
	(define compiler (list 'tsql_scalar_function_compiler callable
		(list 'quote parameters) (list 'quote return_type)))
	(list 'sql_type_expr (if relational
		(list 'tsql_relational_scalar_call callable resolved)
		(cons 'sql_compile_expr (cons compiler resolved)))
		(tsql_scalar_function_type return_type) nil))))
