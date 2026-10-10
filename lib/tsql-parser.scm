/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* T-SQL frontend. SELECT/UNION/window AST construction follows sql-parser.scm;
all relational work goes through the common logical and physical planner.
MemCP exposes one T-SQL schema, dbo, inside each MemCP database. */
(define tsql_identifier_unquoted (parser (not
	(regex "[a-zA-Z_][a-zA-Z0-9_$]*")
	(atom "SELECT" true) (atom "FROM" true) (atom "WHERE" true)
	(atom "GROUP" true) (atom "HAVING" true) (atom "ORDER" true)
	(atom "BY" true) (atom "AS" true) (atom "ON" true)
	(atom "JOIN" true) (atom "LEFT" true) (atom "RIGHT" true)
	(atom "INNER" true) (atom "OUTER" true) (atom "CROSS" true)
	(atom "FULL" true) (atom "APPLY" true) (atom "WITH" true)
	(atom "UNION" true) (atom "ALL" true) (atom "EXCEPT" true)
	(atom "INTERSECT" true) (atom "INSERT" true) (atom "VALUES" true)
	(atom "SET" true) (atom "TOP" true) (atom "PERCENT" true)
	(atom "TIES" true) (atom "OFFSET" true) (atom "FETCH" true)
	(atom "ROWS" true) (atom "ROW" true) (atom "ONLY" true)
	(atom "ASC" true) (atom "DESC" true) (atom "OVER" true)
	(atom "PARTITION" true) (atom "NOT" true) (atom "IN" true)
	(atom "BETWEEN" true) (atom "AND" true) (atom "OR" true)
	(atom "THEN" true) (atom "ELSE" true) (atom "END" true)
	(atom "OUTPUT" true) (atom "OPTION" true) (atom "FOR" true)
)))
(define tsql_identifier_quoted (parser (or
	(parser '("[" (define id (regex "(?:[^\\]]|\\]\\])+" false false)) "]") (replace id "]]" "]"))
	(parser '("\"" (define id (regex "(?:[^\"]|\"\")+" false false)) "\"") (replace id "\"\"" "\""))
)))
(define tsql_identifier (parser (or tsql_identifier_quoted tsql_identifier_unquoted)))
/* CREATE, CAST and parameter declarations share one type grammar. Names stay
frontend data; dimension and alias semantics are checked by the existing binder. */
(define tsql_type_specification (lambda (schema planning_session) (begin
	(define size nil) (define scale nil)
	(parser '((define names (+ tsql_identifier "."))
		(? "(" (define size (or (parser (atom "MAX" true) -1) sql_int)) (? "," (define scale sql_int)) ")"))
		(begin
			(define type (match names
				'(name) name
				'(owner name) (tsql_qualified_type owner name)
				'(database owner name)
				(if (or (equal?? database schema) (and planning_session (equal?? database (planning_session "tsql_import_source"))))
					(tsql_qualified_type owner name) (error "type declaration escapes the active database"))
				_ (error "unsupported qualified type declaration")))
			(list (toUpper type) size scale))))))
(define tsql_string_body (parser '((atom "'" false)
	(define value (regex "(?:''|[^'])*" false false)) (atom "'" false false))
	(replace value "''" "'")))
(define tsql_string (parser '((? (atom "N" true false)) (define value tsql_string_body)) value))
(define tsql_number_node (lambda (text)
	(if (regexp_test text "[eE]") (simplify text)
		(list 'tsql_number_literal text (list 'quote (tsql_literal_spec text))))))
(define tsql_number (parser (or
	(parser (define value (regex "(?:(?:[0-9]+\\.[0-9]*|\\.[0-9]+)(?:[eE][+-]?[0-9]+)?|[0-9]+[eE][+-]?[0-9]+)")) (tsql_number_node value))
	(parser (define value (regex "[0-9]+")) (tsql_number_node value))
)))
(define tsql_column (parser (or
	(parser '((define alias tsql_identifier) "." (define name tsql_identifier)) '('get_column alias true name true))
	(parser (define name tsql_identifier) '('get_column nil true name true))
)))
(define tsql_object_name (lambda (database owner name)
	(if (equal? (toLower owner) "dbo") (list database name)
		(if (equal?? owner "sys") (list (concat "__tsql_catalog:" database) name)
			(error "T-SQL currently supports dbo and the read-only sys catalog")))))
(define tsql_has_subquery (lambda (expr) (match expr
	(cons (symbol inner_select) _) true
	(cons (symbol inner_select_in) _) true
	(cons (symbol inner_select_exists) _) true
	(cons head tail) (or (tsql_has_subquery head)
		(reduce tail (lambda (found part) (or found (tsql_has_subquery part))) false))
	_ false)))
(define tsql_has_clock (lambda (expr) (match expr
	(cons head tail) (or (equal? head 'tsql_clock_value)
		(reduce tail (lambda (found part) (or found (tsql_has_clock part))) false))
	_ false)))
(define tsql_guard_query (lambda (sql)
	(match sql (regex "(?i)^\\s*ROLLBACK(?:\\s+TRAN(?:SACTION)?)?\\s*;?\\s*$" _) "ROLLBACK" _ sql)))

(import "tsql-types.scm")
(import "tsql-catalog.scm")
(import "tsql-odbc.scm")

/* The registry is dialect-local: ISNULL and DATEDIFF must not overwrite the
existing dialect meanings in sql_builtins. Shared functions remain explicit. */
(define tsql_builtins (newsession))
(map '("ABS" "CEILING" "FLOOR" "ROUND" "SQRT" "RAND" "UPPER" "LOWER"
	"REPLACE" "SUBSTRING" "SOUNDEX")
	(lambda (name) (tsql_builtins name (sql_builtins name))))
(tsql_builtins "ABS" (lambda (value) (tsql_decimal_math "ABS" value)))
(tsql_builtins "FLOOR" (lambda (value) (tsql_decimal_math "FLOOR" value)))
(tsql_builtins "CEILING" (lambda (value) (tsql_decimal_math "CEILING" value)))
(tsql_builtins "ROUND" (lambda (value places truncate) (tsql_decimal_math "ROUND" value places (coalesceNil truncate 0))))
(tsql_builtins "REPLICATE" string_repeat)
(tsql_builtins "LEN" (lambda (value) (if (nil? value) nil (strlen (sql_rtrim value)))))
(tsql_builtins "NCHAR" tsql_nchar)
(tsql_builtins "GETDATE" (lambda () (tsql_clock_value "DATETIME")))
(tsql_builtins "SYSDATETIME" (lambda () (tsql_clock_value "DATETIME2")))
(tsql_builtins "LTRIM" sql_ltrim)
(tsql_builtins "RTRIM" sql_rtrim)
(tsql_builtins "TRIM" sql_trim)

(define tsql_add (lambda (a b)
	(if (or (nil? a) (nil? b)) nil
		(if (and (string? a) (string? b)) (concat a b)
			(+ (if (string? a) (tsql_cast_value a (if (int? b) '("BIGINT") '("FLOAT"))) a)
				(if (string? b) (tsql_cast_value b (if (int? a) '("BIGINT") '("FLOAT"))) b))))))
(define tsql_integer_type? (lambda (type) (has? '("INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT" "BIT" "BOOLEAN") type)))
(define tsql_divide (lambda (a b integer)
	(if (or (nil? a) (nil? b)) nil
		(if (equal? b 0) (error "T-SQL division by zero")
			(if integer (intdiv a b) (div_null a b))))))
(define tsql_avg_divide (lambda (sum count integer)
	(if (or (nil? sum) (equal? count 0)) nil (if integer (intdiv sum count) (/ sum count)))))

/* Resolve dialect arithmetic against catalog declarations before lowering.
Compressed numeric storage may return an integral FLOAT even for an INT column;
SQL division/AVG semantics must depend on SQL types, not the storage encoding. */
/* Character addition is concatenation, including nested expressions. Keep
its type separate from the numeric precision/scale calculation. */
(define tsql_concat_type (lambda (operator left_type right_type)
	(if (and (equal? operator "+")
		(has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR") left_type)
		(has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR") right_type))
		(if (or (has? '("NCHAR" "NVARCHAR") left_type)
			(has? '("NCHAR" "NVARCHAR") right_type)) "NVARCHAR" "VARCHAR") nil)))

/* Parameter declarations are invocation-local planning inputs. Reuse the
common SQL expression binder for every type decision; this walk merely adds
that declaration to parameter leaves and preserves AST data containers. */
(define tsql_parameter_expression (lambda (expr planning_session) (match expr
	((symbol quote) _) expr
	((symbol if) ((symbol session) bound_key) ((symbol session) key) ((symbol error) _)) (begin
		/* The failure branch cannot produce a value. Attach the declaration
		to the whole guarded parameter so its error branch does not dilute
		the type of a declared NULL or an exact coefficient. */
		(define spec (if (and planning_session (strlike key "tsql_param:%")
			(equal? bound_key (concat "tsql_bound:" (substr key 11))))
			(planner_literal_value (list 'session (concat "tsql_param_type:" (substr key 11))) planning_session) nil))
		(if spec (list 'tsql_parameter expr (list 'quote spec)) expr))
	((symbol session) key) (begin
		(define spec (if (and planning_session (strlike key "tsql_param:%"))
			(planner_literal_value (list 'session (concat "tsql_param_type:" (substr key 11))) planning_session) nil))
		(if spec (list 'tsql_parameter expr (list 'quote spec)) expr))
	(cons head tail) (cons (tsql_parameter_expression head planning_session)
		(map tail (lambda (part) (tsql_parameter_expression part planning_session))))
	_ expr)))
(define tsql_resolve_expr (lambda (sources expr planning_session)
	(begin
		(define annotated (tsql_parameter_expression expr planning_session))
		(match annotated
			(cons head tail) (if (symbol? head) (sql_info_formula (sql_expr_info sources annotated true))
				(map annotated (lambda (part) (tsql_resolve_expr sources part planning_session))))
			_ annotated))))
(define tsql_bind_assignment (lambda (schema table_name column_name expression sources planning_session) (begin
	(define column (find (tsql_get_schema schema table_name) (lambda (candidate) (equal?? (candidate "Field") column_name)) nil))
	(if (nil? column) (error "unknown assignment column") true)
	(define info (sql_expr_info sources (tsql_parameter_expression expression planning_session) true))
	(define target (tsql_column_spec column))
	(if target (list 'tsql_cast_bound (sql_info_formula info) (list 'quote (sql_info_spec info)) (list 'quote target)) (sql_info_formula info)))))
(define tsql_expr_type (lambda (sources expr planning_session)
	(sql_info_type (sql_expr_info sources (tsql_parameter_expression expr planning_session) true))))
(define tsql_binary_type? (lambda (type) (has? '("BINARY" "VARBINARY" "ROWVERSION" "TIMESTAMP") type)))
(define tsql_fold_additive (lambda (acc term) (match term
	'("+" value) '('tsql_add acc value)
	'("-" value) '('tsql_subtract acc value))))
(define tsql_fold_multiplicative (lambda (acc term) (match term
	'("*" value) '('tsql_multiply acc value)
	'("/" value) '('tsql_divide acc value)
	'("%" value) '('tsql_remainder acc value))))

(define tsql_window (lambda (name args spec)
	(if (and (has? '("COUNT" "SUM" "AVG" "MIN" "MAX") name) (not (empty_list? (cadr spec))))
		(error "ordered aggregate windows require default-frame support")
		'('window_func name (if (equal? name "SUM") (map args (lambda (value) (list (quote tsql_sum_value) value))) args) spec))))

/* Statement completion distinguishes an empty INSERT from a failed allocation.
The allocator reports only this invocation's range; no global counter is read. */
(define tsql_insert_identity_scope (lambda (session action)
	(tsql_insert_identity_values session action)))

/* Control-flow predicates have no relational input. Unsupported relational
shapes must not acquire a scalar execution path outside logical planning. */
(define tsql_conditional_relational? (lambda (expr) (match expr
	((symbol get_column) _ _ _ _) true
	(cons (symbol aggregate) _) true
	(cons (symbol count_distinct) _) true
	(cons (symbol group_concat_distinct) _) true
	(cons head tail) (or (subquery_head? head)
		(reduce tail (lambda (found part) (or found (tsql_conditional_relational? part))) false))
	_ false)))
(define tsql_conditional_predicate? (lambda (expr) (match expr
	(cons head _) (has? '(equal?? < <= > >= nil? not sql_not and or strlike sql_in) head)
	_ false)))

/* Bind existing query-block/union-block data after parsing, preserving the
logical phase boundary and invocation-local parameter declarations. */
(define tsql_statement_foreign_policy (lambda (definition policy) (match definition
	'("foreign" _ _ target _ _ _) (match target '(db parent) (if policy (policy db parent false) true)) _ true)))
(define tsql_resolve_sources (lambda (sources policy)
	(map sources (lambda (source)
		(if (string? (source_relation source)) (begin
			(define database (source_schema source)) (define name (source_relation source))
			(if policy (policy (coalesceNil (tsql_catalog_database database) database)
				(if (tsql_catalog_database database) true name) false) true)
			(source_with_relation source (tsql_resolve_table database name))) source)))))
(define tsql_resolve_subqueries (lambda (expression policy planning_session)
	(match expression
		((symbol quote) _value) expression
		((symbol inner_select) query) (list 'inner_select (tsql_resolve_query query policy planning_session))
		((symbol inner_select_exists) query) (list 'inner_select_exists (tsql_resolve_query query policy planning_session))
		((symbol inner_select_in) value query) (list 'inner_select_in
			(tsql_resolve_subqueries value policy planning_session) (tsql_resolve_query query policy planning_session))
		(cons head tail) (cons head (map tail (lambda (part) (tsql_resolve_subqueries part policy planning_session))))
		_ expression)))
(define tsql_resolve_query (lambda (query policy planning_session)
	(match query
		((symbol query-block) database sources fields condition group having order limit offset hidden stages facts) (begin
			(define sources (map (tsql_resolve_sources sources policy) (lambda (source)
				(if (string? (source_relation source)) source
					(source_with_relation source (tsql_resolve_query (source_relation source) policy planning_session))))))
			(define bind (lambda (expression) (tsql_resolve_expr sources
				(tsql_resolve_subqueries expression policy planning_session) planning_session)))
			(make_query_block database sources (map_assoc fields (lambda (_name expression) (bind expression)))
				(bind condition) (if group (map group bind) nil) (bind having)
				(if order (map order (lambda (item) (list (bind (car item)) (cadr item)))) nil)
				(bind limit) (bind offset) hidden stages facts))
		((symbol union-block) mode branches order limit offset facts)
		(make_union_block mode (map branches (lambda (branch) (tsql_resolve_query branch policy planning_session)))
			(tsql_parameter_expression order planning_session)
			(tsql_parameter_expression limit planning_session) (tsql_parameter_expression offset planning_session) facts)
		_ query)))

/* Statement ASTs stay frontend data. Only the selected control-flow branch
is lowered through the same compiler as an ordinary standalone statement. */
(define tsql_compile_statement (lambda (schema statement policy planning_session tx)
	(match statement
		'("insert-values" target columns rows) (match target '(db name) (begin
			(if policy (policy db name true) true)
			(define name (tsql_resolve_table db name))
			(define insert_columns (if columns (map columns (lambda (column) (tsql_resolve_column db name column))) (tsql_insertable_columns db name)))
			(if (reduce rows (lambda (valid row) (and valid (equal? (count row) (count insert_columns)))) true)
				true (error "INSERT column count does not match value count"))
			/* Subqueries must be decorrelated by the planner, never eval'd per cell. */
			(if (reduce rows (lambda (found row) (or found (tsql_has_subquery row))) false)
				(error "T-SQL subqueries in INSERT VALUES are not supported yet") true)
			(list 'tsql_insert_identity_scope 'session (list 'lambda (list 'report) (list '!begin (list 'tsql_prepare_rowversion_write db name (cons list insert_columns))
				(list 'insert '('table db name) (cons list insert_columns) (sql_insert_values_expr (map rows (lambda (row) (mapIndex row (lambda (ordinal cell) (tsql_bind_assignment db name (nth insert_columns ordinal) cell '() planning_session))))))
					'('list) nil false 'report 'tx true 'tsql_statement_values))))))
		'("insert-select" target columns query) (match target '(db name) (begin
			(if policy (policy db name true) true)
			(define name (tsql_resolve_table db name))
			(define insert_columns (if columns (map columns (lambda (column) (tsql_resolve_column db name column))) (tsql_insertable_columns db name)))
			(define query (tsql_bind_query_types (sql_expand_views (tsql_resolve_query query policy planning_session) policy)))
			(define source_descriptions (tsql_query_descriptions query planning_session))
			(if (equal? (count source_descriptions) (count insert_columns)) true (error "INSERT SELECT column count does not match"))
			(list 'tsql_insert_identity_scope 'session (list 'lambda (list 'report) (list '!begin
				(list 'tsql_prepare_rowversion_write db name (cons list insert_columns))
				/* Pass the DML sink as a lexical argument: a nested assignment can
				leave the compiled SELECT bound to the enclosing frontend sink. */
				(list 'tsql_count_insert_rows (list 'lambda (list 'record)
					(list (list 'lambda (list 'resultrow 'resultfields)
						(build_queryplan_term query planning_session tx))
						(list 'lambda (list 'item)
							(list 'record (list 'insert '('table db name) (cons list insert_columns)
								(cons list (list (cons list (map (produceN (count insert_columns)) (lambda (i) (begin
									(define target (tsql_column_spec (find (tsql_get_schema db name) (lambda (column) (equal?? (column "Field") (nth insert_columns i))) nil)))
									(define value '('nth 'item (+ (* i 2) 1)))
									(if target (list 'tsql_cast_bound value (list 'quote (tsql_description_spec (nth source_descriptions i))) (list 'quote target)) value)))))))
								'('list) nil false 'report 'tx true 'tsql_statement_values)))
						(list 'lambda (list 'titles) true)))))))))
		'("update" target assignments joined condition) (match target '(db name) (begin
			(define definitions (if joined (tsql_resolve_sources (merge joined) policy) (list (list name db (tsql_resolve_table db name) false nil))))
			(define target_def (reduce definitions (lambda (found definition)
				(match definition '(alias _ table _ _) (if (or (equal?? alias name) (equal?? table (tsql_resolve_table db name))) definition found))) nil))
			(if (nil? target_def) (error "UPDATE target not found in FROM") true)
			(match target_def '(alias target_db table _ _) (begin
				(if policy (policy target_db table true) true)
				(map assignments (lambda (assignment) (begin
					(define column (find (tsql_get_schema target_db table) (lambda (column) (equal?? (column "Field") (car assignment))) nil))
					(if (and column (equal? (column "Extra") "auto_increment")) (error "IDENTITY columns cannot be updated") true))))
				(list '!begin (list 'tsql_prepare_rowversion_write target_db table (cons list (map assignments car)))
					(build_dml_plan target_db table alias definitions (merge (extract_assoc (merge assignments) (lambda (col expr) (list (tsql_resolve_column target_db table col) (tsql_bind_assignment target_db table col expr definitions planning_session))))) (tsql_resolve_expr definitions (coalesceNil condition true) planning_session) nil nil nil planning_session tx))))))
		'("delete" target condition) (match target '(db name) (begin
			(if policy (policy db name true) true)
			(define name (tsql_resolve_table db name))
			(build_dml_plan db name nil (list (list name db name false nil)) nil (tsql_resolve_expr (list (list name db name false nil)) (coalesceNil condition true) planning_session) nil nil nil planning_session tx)))
		'("add-foreign" target validation definition) (match target '(db name) (begin
			(if (equal?? validation "NOCHECK") (error "WITH NOCHECK foreign keys are unsupported") true)
			(if policy (policy db name true) true)
			(tsql_statement_foreign_policy definition policy)
			(list 'tsql_create_foreign_key db name (list 'quote definition) 'tx)))
		'("create-table" target definitions) (match target '(db name) (begin
			(if policy (policy db name true) true)
			(map definitions (lambda (definition) (tsql_statement_foreign_policy definition policy)))
			'('tsql_create_table db name (cons list (tsql_table_definitions db name definitions)) '('list) false 'tx)))
		'("create-view" target aliases captured) (match target '(db name) (begin
			(if policy (policy db name true) true)
			(list 'tsql_create_view '('session "__memcp_tx") db name (car captured)
				(list 'quote (sql_apply_view_column_aliases (tsql_resolve_query (cadr captured) policy planning_session) aliases)))))
		'("metadata-procedure" name arguments) (begin
			(if (has? '("sp_tables" "sp_columns" "sp_datatype_info" "sp_datatype_info_100" "sp_describe_undeclared_parameters" "sp_pkeys" "sp_statistics" "sp_special_columns" "sp_fkeys") (toLower name)) true
				(error "unsupported metadata procedure"))
			(if policy (policy schema true false) true)
			(list 'tsql_metadata_emit schema (toLower name)
				(cons list (merge (mapIndex arguments (lambda (ordinal argument) (list (toLower (coalesceNil (car argument) (concat ordinal))) (tsql_resolve_expr '() (cadr argument) planning_session))))))
				'session 'resultrow 'resultfields))
		'("select" query) (begin
			(define expanded_query (tsql_bind_query_types (sql_expand_views (tsql_resolve_query query policy planning_session) policy)))
			(if (and planning_session (planning_session "tsql_describe_only"))
				(if (decorrelate_logical_query expanded_query)
					(list (quote quote) (tsql_query_descriptions expanded_query planning_session))
					(error "cannot describe this SELECT"))
				(list (quote !begin)
					(list (quote resultfields) (list (quote quote) (queryplan_result_titles expanded_query))
						(list (quote quote) (tsql_query_descriptions expanded_query planning_session)))
					(build_queryplan_term expanded_query planning_session tx))))
		'("explain-ir" query) (explain_queryplan_ir (tsql_bind_query_types (sql_expand_views (tsql_resolve_query query policy planning_session) policy)))
		'("explain" query) '('list (list 'quote (build_queryplan_term (tsql_bind_query_types (sql_expand_views (tsql_resolve_query query policy planning_session) policy)) planning_session tx)))
		'("create-type" target type nullable) (match target '(db name) (begin (if policy (policy db true true) true)
			(if (has? '("ROWVERSION" "TIMESTAMP") (car type)) (error "ROWVERSION cannot be an alias base type") true)
			(define declaration (tsql_native_type type))
			'('create_sql_type_alias db name (car type) (cons list (cadr declaration)) nullable)))
		'("drop-type" target exists) (match target '(db name) (begin (if policy (policy db true true) true)
			'('drop_sql_type_alias db name (not (nil? exists)))))
		'("add-default" target name value column) (match target '(db table_name) (begin
			(if policy (policy db table_name true) true)
			(if (tsql_has_subquery (cadr value)) (error "subqueries cannot be column defaults") true)
			(list 'tsql_create_default_constraint db table_name name column (list 'quote (cadr value)) (regexp_replace (car value) "^\\s+|\\s+$" ""))))
		'("add-unique" target name columns) (match target '(db table_name) (begin
			(if policy (policy db table_name true) true)
			(list 'tsql_create_unique_constraint db table_name name (list 'quote columns) 'tx)))
		'("add-column" target name type attributes) (match target '(db table_name) (begin (if policy (policy db table_name true) true)
			(match (tsql_column_definition db (list "column" name type attributes))
				'(_ _ column_name native dimensions options)
				'('tsql_add_column db table_name column_name native dimensions options))))
		'("drop-constraint" target name) (match target '(db table_name) (begin (if policy (policy db table_name true) true)
			(list 'tsql_drop_constraint db table_name name 'tx)))
		'("drop-column" target name) (match target '(db table_name) (begin (if policy (policy db table_name true) true)
			'('dropcolumn '('table db '('tsql_resolve_table db table_name)) '('tsql_resolve_column db '('tsql_resolve_table db table_name) name))))
		'("create-database" name) (if (and planning_session (planning_session "tsql_import_target")) '('tsql_import_database 'session name)
			(begin (if policy (policy "system" true true) true) '('createdatabase name)))
		'("drop-table" target exists) (match target '(db name) (begin (if policy (policy db name true) true) '('droptable db '('tsql_resolve_table db name) (not (nil? exists)))))
		'("drop-view" target exists) (match target '(db name) (begin (if policy (policy db name true) true) '('tsql_drop_view '('session "__memcp_tx") db name (not (nil? exists)))))
		'("truncate" target) (error "TRUNCATE TABLE is not supported in t-sql mode")
		'("use" db) (begin
			(if (and planning_session (planning_session "tsql_import_target")) '('tsql_import_database 'session db)
				(begin (if policy (policy db true false) true)
					'('if '('list? '('show db)) '('session "schema" db) '('error (concat "unknown database " db))))))
		'("begin-transaction") '('tx_begin_acid 'session 'tx)
		'("commit") '('tx_commit 'session)
		'("rollback") '('tx_rollback 'session)
		'("nocount" value) '('session "tsql_nocount" value)
		'("supported-option") true
		'("identity-insert" target) (if (and planning_session (planning_session "tsql_import_target"))
			(match target '(db name) (begin (if policy (policy db name true) true) true))
			(error "IDENTITY_INSERT currently requires load_tsql"))
		'("if" condition selected alternative) (begin
			(if (and planning_session (planning_session "tsql_describe_only"))
				(error "prepared result metadata currently requires SELECT") true)
			(if (or (tsql_conditional_relational? condition) (expr_contains_window? condition))
				(error "IF currently requires a scalar or catalog predicate") true)
			(if (tsql_conditional_predicate? condition) true (error "IF requires a boolean predicate"))
			(if policy (policy schema true false) true)
			(list 'if (tsql_resolve_expr '() condition planning_session)
				(list 'tsql_execute_statement_ast schema (list 'quote selected) policy 'session 'tx 'resultrow 'resultfields)
				(if alternative (list 'tsql_execute_statement_ast schema (list 'quote alternative) policy 'session 'tx 'resultrow 'resultfields) 0)))
		'("empty") 0
		_ (error "unsupported statement AST"))))

(define parse_tsql (lambda (schema s policy planning_session tx) (begin
	(define parse_started_ns (nanotime))
	(define object (parser (or
		(parser '((define db tsql_identifier) "." (define owner tsql_identifier) "." (define name tsql_identifier))
			(tsql_object_name (if (and planning_session (planning_session "tsql_import_target"))
				(if (or (equal?? db schema) (equal?? db (planning_session "tsql_import_source"))) schema
					(error "import reference escapes the target database")) db) owner name))
		(parser '((define owner tsql_identifier) "." (define name tsql_identifier)) (tsql_object_name schema owner name))
		(parser (define name tsql_identifier) (list schema name))
	)))
	(define type_spec (tsql_type_specification schema planning_session))
	(define expression (parser (or
		(parser '((define a expression1) (atom "OR" true) (define rest (+ expression1 (atom "OR" true)))) (cons 'or (cons a rest)))
		expression1)))
	(define expression1 (parser (or
		(parser '((define a expression2) (atom "AND" true) (define rest (+ expression2 (atom "AND" true)))) (cons 'and (cons a rest)))
		expression2)))
	(define expression2 (parser (or
		(parser '((atom "NOT" true) (define a expression2)) '('sql_not a))
		(parser '((define a expression3) (atom "NOT" true) (atom "IN" true) "(" (define query tsql_select) ")") '('sql_not '('inner_select_in a query)))
		(parser '((define a expression3) (atom "IN" true) "(" (define query tsql_select) ")") '('inner_select_in a query))
		(parser '((define a expression3) (define negate (? (atom "NOT" true))) (atom "IN" true) "(" (define values (+ expression ",")) ")")
			(if negate '('sql_not '('sql_in (cons list values) a)) '('sql_in (cons list values) a)))
		(parser '((define a expression3) (define negate (? (atom "NOT" true))) (atom "BETWEEN" true) (define lo expression3) (atom "AND" true) (define hi expression3))
			(begin (define result '('and '('>= a lo) '('<= a hi))) (if negate '('sql_not result) result)))
		(parser '((define a expression3) (define negate (? (atom "NOT" true))) (atom "LIKE" true) (define b expression3))
			(if negate '('sql_not '('strlike a b "utf8mb4_general_ci")) '('strlike a b "utf8mb4_general_ci")))
		(parser '((define a expression3) (atom "IS" true) (define negate (? (atom "NOT" true))) (atom "NULL" true))
			(if negate '('not '('nil? a)) '('nil? a)))
		(parser '((define a expression3) (define op (or (parser "<>" "ne") (parser "!=" "ne")
			(parser "<=" "le") (parser ">=" "ge") (parser "=" "eq") (parser "<" "lt") (parser ">" "gt"))) (define b expression3))
			(sql_comparison_expr op a b))
		expression3)))
	(define expression3 (parser '((define first expression4)
		(define rest (* (parser '((define op (or "+" "-")) (define value expression4)) (list op value)) empty true)))
		(reduce rest tsql_fold_additive first)))
	(define expression4 (parser '((define first expression5)
		(define rest (* (parser '((define op (or "*" "/" "%")) (define value expression5)) (list op value)) empty true)))
		(reduce rest tsql_fold_multiplicative first)))
	(define expression5 (parser (or
		(parser '("-" (define value expression5)) '('tsql_negate value))
		(parser '("+" (define value expression5)) value)
		expression6)))
	(define window_order (parser '((define value expression) (define direction (or (parser (atom "DESC" true) >) (parser (? (atom "ASC" true)) <)))) (list value direction)))
	(define window_spec (parser '(
		(? (atom "PARTITION" true) (atom "BY" true) (define partition (+ expression ",")))
		(? (atom "ORDER" true) (atom "BY" true) (define ordering (+ window_order ","))))
		(list (coalesce partition '()) (coalesce ordering '()))))
	(define expression6 (parser (or
		(parser '("(" (define query tsql_select) ")") '('inner_select query))
		(parser '("(" (define value expression) ")") value)
		(parser '((atom "EXISTS" true) "(" (define query tsql_select) ")") '('inner_select_exists query))
		(parser '((atom "CASE" true) (define cases (+ (parser '((atom "WHEN" true) (define condition expression) (atom "THEN" true) (define value expression)) (list condition value))))
			(? (atom "ELSE" true) (define otherwise expression)) (atom "END" true)) (cons 'if (merge (list (merge cases) (list otherwise)))))
		(parser '((atom "CASE" true) (define input expression) (define cases (+ (parser '((atom "WHEN" true) (define condition expression) (atom "THEN" true) (define value expression)) (list condition value))))
			(? (atom "ELSE" true) (define otherwise expression)) (atom "END" true)) (cons 'if (merge (list (merge (map cases (lambda (part) (list '('equal?? input (car part)) (cadr part))))) (list otherwise)))))
		(parser '((atom "COUNT" true) "(" "*" ")" (atom "OVER" true) "(" (define spec window_spec) ")") (tsql_window "COUNT" '() spec))
		(parser '((atom "AVG" true) "(" (define value expression) ")" (atom "OVER" true) "(" (define spec window_spec) ")")
			'('tsql_avg_value value (tsql_window "SUM" (list value) spec) (tsql_window "COUNT" (list value) spec)))
		(parser '((define name (or (atom "ROW_NUMBER" true) (atom "RANK" true) (atom "DENSE_RANK" true)
			(atom "NTILE" true) (atom "LAG" true) (atom "LEAD" true) (atom "FIRST_VALUE" true) (atom "LAST_VALUE" true)
			(atom "COUNT" true) (atom "SUM" true) (atom "AVG" true) (atom "MIN" true) (atom "MAX" true)))
			"(" (define args (* expression ",")) ")" (atom "OVER" true) "(" (define spec window_spec) ")") (tsql_window (toUpper name) args spec))
		(parser '((atom "COUNT" true) "(" "*" ")") '('aggregate 1 '+ 0))
		(parser '((atom "COUNT" true) "(" (atom "DISTINCT" true) (define value expression) ")") '('count_distinct value))
		(parser '((atom "COUNT" true) "(" (define value expression) ")") '('aggregate '('if '('nil? value) 0 1) '+ 0))
		(parser '((atom "AVG" true) "(" (define value expression) ")")
			(cons (quote tsql_avg_value) (cons value (cdr (sql_avg_expr (list (quote tsql_sum_value) value) (sql_aggregates "SUM") (sql_aggregates "COUNT"))))))
		(parser '((define name (or (atom "SUM" true) (atom "MIN" true) (atom "MAX" true))) "(" (define value expression) ")")
			(begin (define descriptor (sql_aggregates (toUpper name))) '('aggregate (if (equal? (toUpper name) "SUM") (list (quote tsql_sum_value) value) value) (car descriptor) (cadr descriptor))))
		(parser '((define name (or (atom "ABS" true) (atom "FLOOR" true) (atom "CEILING" true)))
			"(" (define value expression) ")") '('tsql_decimal_math (toUpper name) value))
		(parser '((atom "ROUND" true) "(" (define value expression) "," (define places expression)
			(? "," (define truncate expression)) ")") '('tsql_decimal_math "ROUND" value places (coalesceNil truncate (tsql_numeric_literal "0"))))
		(parser '((atom "NCHAR" true) "(" (define value expression) ")") '('tsql_nchar value))
		(parser '((atom "ISNULL" true) "(" (define value expression) "," (define replacement expression) ")") '('tsql_isnull value replacement))
		(parser '((atom "COALESCE" true) "(" (define args (+ expression ",")) ")") (cons 'coalesceNil args))
		(parser '((atom "CONCAT" true) "(" (define args (+ expression ",")) ")")
			(cons 'concat (map args (lambda (value) '('coalesceNil value "")))))
		(parser '((atom "CAST" true) "(" (define value expression) (atom "AS" true) (define type type_spec) ")") '('tsql_cast_value value (list 'quote type)))
		(parser '((atom "CONVERT" true) "(" (define type type_spec) "," (define value expression)
			(? "," (define style (parser (define style_value expression) (list style_value)))) ")")
			'('tsql_convert_value value (list 'quote type) (if style (car style) -1)))
		(parser '((atom "DATEADD" true) "(" (define unit tsql_identifier) "," (define amount expression) "," (define value expression) ")")
			'('tsql_dateadd unit amount value))
		(parser '((atom "DATEDIFF" true) "(" (define unit tsql_identifier) "," (define start expression) "," (define end expression) ")")
			'('tsql_datediff unit start end))
		(parser '((atom "DATEPART" true) "(" (define unit tsql_identifier) "," (define value expression) ")")
			'('tsql_datepart unit value))
		(parser '((define unit (or (atom "YEAR" true) (atom "MONTH" true) (atom "DAY" true))) "(" (define value expression) ")")
			'('tsql_datepart (toLower unit) value true))
		(parser '((atom "GETDATE" true) "(" ")") '('tsql_cast_value '('tsql_clock_value "DATETIME") (list 'quote '("DATETIME"))))
		(parser '((atom "SYSDATETIME" true) "(" ")") '('tsql_cast_value '('tsql_clock_value "DATETIME2") (list 'quote '("DATETIME2" 7))))
		(parser (atom "CURRENT_TIMESTAMP" true) '('tsql_cast_value '('tsql_clock_value "DATETIME") (list 'quote '("DATETIME"))))
		(parser '((atom "LEFT" true) "(" (define value expression) "," (define length expression) ")") '('sql_substr value 1 length))
		(parser '((atom "RIGHT" true) "(" (define value expression) "," (define length expression) ")") '('sql_substr value '('+ 1 '(- '('strlen value) length)) length))
		(parser '((atom "DB_NAME" true) "(" ")") '('session "schema"))
		(parser '((atom "SCOPE_IDENTITY" true) "(" ")") '('tsql_cast_value '('session "tsql_scope_identity") '('quote '("NUMERIC" 38 0))))
		(parser '((atom "@@" false) (define name tsql_identifier)) (match (toUpper name)
			"VERSION" "MemCP T-SQL compatibility frontend"
			"TRANCOUNT" '('if '('nil? '('session "transaction")) 0 1)
			"ROWCOUNT" '('coalesceNil '('session "tsql_rowcount") 0)
			"IDENTITY" '('tsql_cast_value '('session "tsql_last_identity") '('quote '("NUMERIC" 38 0)))
			_ (error (concat "unsupported T-SQL system variable " name))))
		(parser '((atom "@" false) (define name tsql_identifier))
			'('if '('session (concat "tsql_bound:" (toLower name)))
				'('session (concat "tsql_param:" (toLower name)))
				'('error (concat "unbound T-SQL parameter @" name))))
		(parser '((define name tsql_identifier) "(" (define args (* expression ",")) ")")
			(match (toUpper name)
				"TYPE_ID" (cons 'tsql_type_id (cons schema args))
				"OBJECT_ID" (cons 'tsql_object_id (cons schema (if (equal? (count args) 1) (append args nil) args)))
				"COL_LENGTH" (cons 'tsql_col_length (cons schema args))
				"SERVERPROPERTY" (cons 'tsql_server_property args)
				_ (cons (coalesce (tsql_builtins (toUpper name)) (error (concat "unknown T-SQL function " name))) args)))
		(parser (define value (regex "0[xX][a-zA-Z0-9_]*"))
			'('tsql_binary_literal (substr value 2)))
		(parser (atom "NULL" true) (sql_null_literal))
		(parser '((atom "N" true false) (define value tsql_string_body))
			'('tsql_cast_value value (list 'quote '("NVARCHAR" -1))))
		tsql_number tsql_string_body tsql_column
	)))

	(define tsql_rename_derived_fields (lambda (fields aliases) (match aliases
		(cons alias remaining_aliases) (match fields
			(cons _title (cons expr remaining_fields))
			(cons alias (cons expr (tsql_rename_derived_fields remaining_fields remaining_aliases)))
			_ (error "derived table has more column aliases than columns"))
		_ fields
	)))
	(define tsql_apply_derived_column_aliases (lambda (query aliases) (match query
		((symbol query-block) schema2 tables fields condition group having order limit offset hidden stages facts)
		(list (quote query-block) schema2 tables
			(tsql_rename_derived_fields fields aliases)
			condition group having order limit offset hidden stages facts)
		((symbol union-block) mode branches order limit offset facts)
		(list (quote union-block) mode
			(map branches (lambda (branch) (tsql_apply_derived_column_aliases branch aliases)))
			order limit offset facts)
		_ (error "derived table column aliases require a SELECT query")
	)))

	(define tsql_select_order (lambda (query) (match query
		((symbol query-block) schema tables fields condition group having order limit offset hidden stages facts) order
		'(schema tables fields condition group having order limit offset) order
		_ nil
	)))
	(define tsql_select_limit (lambda (query) (match query
		((symbol query-block) schema tables fields condition group having order limit offset hidden stages facts) limit
		'(schema tables fields condition group having order limit offset) limit
		_ nil
	)))
	(define tsql_select_offset (lambda (query) (match query
		((symbol query-block) schema tables fields condition group having order limit offset hidden stages facts) offset
		'(schema tables fields condition group having order limit offset) offset
		_ nil
	)))
	(define tsql_select_clear_stage (lambda (query) (match query
		((symbol query-block) schema tables fields condition group having order limit offset hidden stages facts)
		(list (quote query-block) schema tables fields condition group having nil
			(if (has? facts (list (quote tsql_top) true)) limit nil) nil hidden stages facts)
		'(schema tables fields condition group having order limit offset) (list schema tables fields condition group having nil nil nil)
		_ query
	)))
	(define tsql_union_all_parts (lambda (query)
		(match query
			((symbol union-block) (symbol all) branches order limit offset facts) (list branches order limit offset)
			((symbol union_all) branches order limit offset) (list branches order limit offset)
			_ nil)))
	(define tsql_union_distinct_parts (lambda (query)
		(match query
			((symbol union-block) (symbol distinct) branches order limit offset facts) (list branches order limit offset)
			((symbol union_distinct) branches order limit offset) (list branches order limit offset)
			_ nil)))
	(define tsql_union_limit (lambda (query) (match query
		((symbol query-block) _ _ _ _ _ _ _ limit _ _ _ facts)
		(if (has? facts (list (quote tsql_top) true)) nil limit)
		_ (tsql_select_limit query))))
	(define tsql_union_all_query (lambda (left right) (begin
		(define right_parts (tsql_union_all_parts right))
		(if (nil? right_parts)
			(list (quote union-block)
				(quote all)
				(list left (tsql_select_clear_stage right))
				(tsql_select_order right)
				(tsql_union_limit right)
				(tsql_select_offset right)
				'())
			(match right_parts '(branches order limit offset)
				(list (quote union-block) (quote all) (cons left branches) order limit offset '())))
	)))
	(define tsql_union_distinct_query (lambda (left right) (begin
		(define right_parts (tsql_union_distinct_parts right))
		(if (nil? right_parts)
			(list (quote union-block)
				(quote distinct)
				(list left (tsql_select_clear_stage right))
				(tsql_select_order right)
				(tsql_union_limit right)
				(tsql_select_offset right)
				'())
			(match right_parts '(branches order limit offset)
				(list (quote union-block) (quote distinct) (cons left branches) order limit offset '())))
	)))
	(define tabledefs (parser (or
		/* TODO: left [outer] join, right [outer] join recursive buildup */
		(parser '((define l tabledefs) (define x (or
			(parser '((atom "LEFT" true) (? (atom "OUTER" true)) (atom "JOIN" true) (define r tabledef) (atom "ON" true) (define e expression)) (match r '(id schema tbl _ nil) '('(id schema tbl true e))))
			(parser '((? (atom "INNER" true)) (atom "JOIN" true) (define r tabledef) (atom "ON" true) (define e expression)) (match r '(id schema tbl _ nil) '('(id schema tbl false e))))
			(parser '((? (atom "CROSS" true)) (atom "JOIN" true) (define r tabledefs)) r)
		))) (merge l x))
		/* Normalize RIGHT JOIN to the existing LEFT JOIN execution contract:
		the preserved row stream must come first, and the nullable side is marked
		as outer on the right. */
		(parser '((define l tabledef) (atom "RIGHT" true) (? (atom "OUTER" true)) (atom "JOIN" true) (define r tabledefs) (atom "ON" true) (define e expression))
			(match l '(id schema tbl _ nil)
				(merge r '('(id schema tbl true e)))))
		(parser (define t tabledef) '(t))
	)))
	(define tabledef (parser (or
		(parser '("(" (define query tsql_select) ")" (? (atom "AS" true)) (define alias tsql_identifier)
			"(" (define aliases (+ tsql_identifier ",")) ")") (list alias schema (tsql_apply_derived_column_aliases query aliases) false nil))
		(parser '("(" (define query tsql_select) ")" (? (atom "AS" true)) (define alias tsql_identifier)) (list alias schema query false nil))
		(parser '((define target object) (? (? (atom "AS" true)) (define alias tsql_identifier)))
			(match target '(db name) (list (coalesce alias name) db name false nil)))
	)))
	(define from nil) (define group nil) (define having nil)
	(define order nil) (define limit nil) (define offset nil)
	(define top nil)
	(define extract_title_or_sql (lambda (captured)
		(match (cadr captured)
			'('get_column _ _ name _) name
			_ (car captured))))

	(define tsql_select_core (parser '(
		(atom "SELECT" true)
		(define distinct (? (atom "DISTINCT" true)))
		(? (atom "TOP" true) (define top (or (parser '("(" (define n expression) ")") n) tsql_number)))
		(define cols (+ (or
			(parser "*" '("*" '((quote get_column) nil false "*" false)))
			(parser '((define tbl tsql_identifier_quoted) "." "*") '("*" '((quote get_column) tbl true "*" false)))
			(parser '((define tbl tsql_identifier_unquoted) "." "*") '("*" '((quote get_column) tbl true "*" false)))
			(parser '((define e expression) (atom "AS" true) (define title tsql_identifier)) '(title e))
			(parser '((define e expression) (atom "AS" true) (define title tsql_string)) '(title e))
			/* Bare select-list aliases also work without AS. */
			(parser '((define e expression) (define title tsql_identifier)) '(title e))
			/* capture expression to get raw SQL text for column naming */
			(parser (define captured (capture expression)) '((extract_title_or_sql captured) (car (cdr captured))))
		) ","))
		(?
			(atom "FROM" true)
			(define from (+ tabledefs ","))
		)
		(define condition (or (parser '(
			(atom "WHERE" true)
			(define condition2 expression)
		) condition2) (empty true)))
		/* GROUP BY + HAVING */
		(?
			(atom "GROUP" true)
			(atom "BY" true)
			(define group (+
				expression
				(atom "," true)
			))
		)
		(?
			(atom "HAVING" true)
			(define having expression)
		)
		/* ORDER BY + LIMIT */
		(?
			(atom "ORDER" true)
			(atom "BY" true)
			(define order (+
				(parser '(
					(define col expression)
					(? (atom "COLLATE" true) (define coll tsql_identifier))
					(define direction_desc (or
						(parser (atom "DESC" true) >)
						(parser (atom "ASC" true) <)
						(parser empty <)
					))
				) (list col (if coll (collate coll (equal? direction_desc >)) direction_desc)))

				(atom "," true)
			))
		)
		(? (atom "OFFSET" true) (define offset expression) (or (atom "ROW" true) (atom "ROWS" true))
			(? (atom "FETCH" true) (or (atom "FIRST" true) (atom "NEXT" true)) (define limit expression)
				(or (atom "ROW" true) (atom "ROWS" true)) (atom "ONLY" true)))
		(? (atom "FOR" true) (atom "BROWSE" true) (define browse (parser empty true)))
	) (begin
			(if (and (not (nil? top)) (not (nil? offset))) (error "TOP cannot be combined with OFFSET") true)
			(if (and (not (nil? offset)) (nil? order)) (error "OFFSET requires ORDER BY") true)
			(define limit (coalesceNil top limit))
			(define projected_exprs (extract_assoc (merge cols) (lambda (_title expr) expr)))
			(define sources (if (nil? from) '() (merge from)))
			/* GROUP BY already makes the result unique when every grouping key is
			projected. Preserve that explicit group instead of replacing it with the
			DISTINCT projection (which may contain aggregate expressions). */
			(define distinct_preserved_by_group (and distinct
				(and (not (nil? group))
					(and (not (empty_list? group))
						(reduce group (lambda (preserved expr)
							(and preserved (contains? projected_exprs expr))) true)))))
			(list (quote query-block) schema sources (merge cols) condition
				(if (and distinct (not distinct_preserved_by_group)) projected_exprs group)
				having order limit offset '() '()
				(merge (list (list (quote frontend) "tsql")) (if distinct (list (list (quote select_distinct) true)) '())
					(if browse (list (list (quote tsql_browse) true)) '())
					(if (nil? top) '() (list (list (quote tsql_top) true))))))))
	(define tsql_select (parser (or
		(parser '(
			(define left tsql_select_core)
			(atom "UNION" true)
			(atom "ALL" true)
			(define right tsql_select)
		) (tsql_union_all_query left right))
		(parser '(
			(define left tsql_select_core)
			(atom "UNION" true)
			(define right tsql_select)
		) (tsql_union_distinct_query left right))
		tsql_select_core
	)))

	(define tsql_insert (parser '((atom "INSERT" true) (? (atom "INTO" true))
		(define target object) (? "(" (define columns (+ tsql_identifier ",")) ")")
		(atom "VALUES" true) (define rows (+ (parser '("(" (define cells (+ expression ",")) ")") cells) ",")))
		(list "insert-values" target columns rows)))
	(define tsql_insert_select (parser '((atom "INSERT" true) (? (atom "INTO" true))
		(define target object) (? "(" (define columns (+ tsql_identifier ",")) ")") (define query tsql_select))
		(list "insert-select" target columns query)))
	(define tsql_update (parser '((atom "UPDATE" true) (define target object)
		(atom "SET" true) (define assignments (+ (parser '((define column tsql_identifier) "=" (define value expression)) (list column value)) ","))
		(? (atom "FROM" true) (define joined (+ tabledefs ",")))
		(? (atom "WHERE" true) (define condition expression)))
		(list "update" target assignments joined condition)))
	(define tsql_delete (parser '((atom "DELETE" true) (atom "FROM" true) (define target object)
		(? (atom "WHERE" true) (define condition expression)))
		(list "delete" target condition)))
	(define column_attributes (parser (define attributes (* (or
		(parser '((atom "NOT" true) (atom "NULL" true)) '("null" false))
		(parser (atom "NULL" true) '("null" true))
		(parser '((atom "PRIMARY" true) (atom "KEY" true) (? (or (atom "CLUSTERED" true) (atom "NONCLUSTERED" true)))) '("primary" true))
		(parser '((atom "UNIQUE" true) (? (or (atom "CLUSTERED" true) (atom "NONCLUSTERED" true)))) '("unique" true))
		(parser '((? (atom "CONSTRAINT" true) (define name tsql_identifier)) (atom "DEFAULT" true) (define value (capture expression)))
			(if (tsql_has_subquery (cadr value)) (error "subqueries cannot be column defaults")
				(merge (list "tsql_default_ast" (list 'quote (cadr value)) "tsql_default_sql" (regexp_replace (car value) "^\\s+|\\s+$" "")) (if name (list "tsql_default_name" name) '()))))
		(parser '((atom "IDENTITY" true) (? "(" (define seed sql_int) "," (define increment sql_int) ")"))
			(if (and (equal? (coalesceNil seed 1) 1) (equal? (coalesceNil increment 1) 1)) '("auto_increment" true)
				(error "only IDENTITY(1,1) is supported")))
	))) (begin
			(reduce attributes (lambda (present item)
				(if (and present (has_assoc? item "tsql_default_ast")) (error "duplicate column DEFAULT")
					(or present (has_assoc? item "tsql_default_ast")))) false)
			(merge attributes))))
	(define foreign_action (parser '((atom "ON" true) (define kind (or (atom "UPDATE" true) (atom "DELETE" true)))
		(define mode (or (parser '((atom "NO" true) (atom "ACTION" true)) "restrict")
			(parser (atom "CASCADE" true) "cascade")
			(parser '((atom "SET" true) (atom "NULL" true)) "set null")
			(parser '((atom "SET" true) (atom "DEFAULT" true)) (error "SET DEFAULT foreign keys are unsupported")))))
		(list (toLower kind) mode)))
	(define foreign_definition (parser '((? (atom "CONSTRAINT" true) (define name tsql_identifier))
		(atom "FOREIGN" true) (atom "KEY" true) "(" (define columns (+ tsql_identifier ",")) ")"
		(atom "REFERENCES" true) (define parent object) "(" (define parent_columns (+ tsql_identifier ",")) ")"
		(define actions (* foreign_action))) (begin
			(reduce actions (lambda (seen action) (if (has? seen (car action)) (error "duplicate foreign key action") (append seen (car action)))) '())
			(define modes (merge actions))
			(list "foreign" name columns parent parent_columns (coalesceNil (get_assoc modes "update") "restrict") (coalesceNil (get_assoc modes "delete") "restrict")))))
	(define tsql_add_foreign_key (parser '((atom "ALTER" true) (atom "TABLE" true) (define target object)
		(? (atom "WITH" true) (define validation (or (atom "CHECK" true) (atom "NOCHECK" true))))
		(atom "ADD" true) (define definition foreign_definition)) (list "add-foreign" target validation definition)))
	(define tsql_create_table (parser '((atom "CREATE" true) (atom "TABLE" true) (define target object)
		"(" (define definitions (+ (or
			foreign_definition
			(parser '((? (atom "CONSTRAINT" true) (define name tsql_identifier)) (atom "PRIMARY" true) (atom "KEY" true)
				(? (or (atom "CLUSTERED" true) (atom "NONCLUSTERED" true)))
				"(" (define columns (+ (parser '((define column tsql_identifier) (? (or (atom "ASC" true) (atom "DESC" true)))) column) ",")) ")") '('list "unique" "PRIMARY" (cons list columns)))
			(parser '((? (atom "CONSTRAINT" true) (define name tsql_identifier)) (atom "UNIQUE" true)
				(? (or (atom "CLUSTERED" true) (atom "NONCLUSTERED" true)))
				"(" (define columns (+ (parser '((define column tsql_identifier) (? (or (atom "ASC" true) (atom "DESC" true)))) column) ",")) ")") '('list "unique" name (cons list columns)))
			(parser '((define name tsql_identifier) (define type type_spec) (define attributes column_attributes))
				(list "column" name type attributes))
		) ",")) ")") (list "create-table" target definitions)))
	(define tsql_create_view (parser '((atom "CREATE" true) (atom "VIEW" true) (define target object)
		(? "(" (define aliases (+ tsql_identifier ",")) ")") (atom "AS" true) (define captured (capture tsql_select)))
		(list "create-view" target aliases captured)))
	(define metadata_procedure (parser '((? (atom "EXEC" true))
		(define name (or (atom "sp_tables" true) (atom "sp_columns" true) (atom "sp_datatype_info_100" true)
			(atom "sp_datatype_info" true) (atom "sp_describe_undeclared_parameters" true) (atom "sp_pkeys" true) (atom "sp_statistics" true)
			(atom "sp_special_columns" true) (atom "sp_fkeys" true)))
		(define arguments (* (parser '((? "@" (define key tsql_identifier) "=") (define value expression)) (list key value)) ","))) (list "metadata-procedure" name arguments)))
	/* Control flow carries statement AST data. Each branch is parsed by the
	same grammar; binding, policy and physical planning wait for selection. */
	(define conditional_statement (parser '((atom "IF" true) (define condition expression)
		(define selected statement) (? (atom "ELSE" true) (define alternative statement)))
		(list "if" condition selected alternative)))
	(define statement (parser (or
		(parser (define query tsql_select) (list "select" query))
		(parser '((atom "EXPLAIN" true) (atom "IR" true) (define query tsql_select)) (list "explain-ir" query))
		(parser '((atom "EXPLAIN" true) (define query tsql_select))
			(list "explain" query))
		tsql_insert tsql_insert_select tsql_update tsql_delete tsql_create_table tsql_create_view tsql_add_foreign_key metadata_procedure
		(parser '((atom "CREATE" true) (atom "TYPE" true) (define target object) (atom "FROM" true)
			(define type type_spec) (define nullable (or (parser '((atom "NOT" true) (atom "NULL" true)) false)
				(parser (? (atom "NULL" true)) true))))
			(list "create-type" target type nullable))
		(parser '((atom "DROP" true) (atom "TYPE" true) (define exists (? (atom "IF" true) (atom "EXISTS" true))) (define target object))
			(list "drop-type" target exists))
		(parser '((atom "ALTER" true) (atom "TABLE" true) (define target object) (atom "ADD" true)
			(? (atom "CONSTRAINT" true) (define name tsql_identifier)) (atom "DEFAULT" true) (define value (capture expression))
			(atom "FOR" true) (define column tsql_identifier)) (list "add-default" target name value column))
		(parser '((atom "ALTER" true) (atom "TABLE" true) (define target object) (atom "ADD" true)
			(? (atom "CONSTRAINT" true) (define name tsql_identifier)) (atom "UNIQUE" true)
			(? (or (atom "CLUSTERED" true) (atom "NONCLUSTERED" true)))
			"(" (define columns (+ tsql_identifier ",")) ")") (list "add-unique" target name columns))

		(parser '((atom "ALTER" true) (atom "TABLE" true) (define target object) (atom "ADD" true)
			(define name tsql_identifier) (define type type_spec) (define attributes column_attributes))
			(list "add-column" target name type attributes))
		(parser '((atom "ALTER" true) (atom "TABLE" true) (define target object) (atom "DROP" true) (atom "CONSTRAINT" true) (define name tsql_identifier))
			(list "drop-constraint" target name))
		(parser '((atom "ALTER" true) (atom "TABLE" true) (define target object) (atom "DROP" true) (atom "COLUMN" true) (define name tsql_identifier))
			(list "drop-column" target name))
		(parser '((atom "CREATE" true) (atom "DATABASE" true) (define name tsql_identifier))
			(list "create-database" name))
		(parser '((atom "DROP" true) (atom "TABLE" true) (define exists (? (atom "IF" true) (atom "EXISTS" true))) (define target object))
			(list "drop-table" target exists))
		(parser '((atom "DROP" true) (atom "VIEW" true) (define exists (? (atom "IF" true) (atom "EXISTS" true))) (define target object))
			(list "drop-view" target exists))
		(parser '((atom "TRUNCATE" true) (atom "TABLE" true) (define target object))
			(list "truncate" target))
		(parser '((atom "USE" true) (define db tsql_identifier)) (list "use" db))
		(parser '((atom "BEGIN" true) (or (atom "TRANSACTION" true) (atom "TRAN" true))) (list "begin-transaction"))
		(parser '((atom "COMMIT" true) (? (or (atom "TRANSACTION" true) (atom "TRAN" true)))) (list "commit"))
		(parser '((atom "ROLLBACK" true) (? (or (atom "TRANSACTION" true) (atom "TRAN" true)))) (list "rollback"))
		/* These switches describe supported defaults; reject opposite settings. */
		(parser '((atom "SET" true) (atom "NOCOUNT" true) (define value (or (parser (atom "ON" true) true) (parser (atom "OFF" true) false)))) (list "nocount" value))
		(parser '((atom "SET" true) (or (atom "QUOTED_IDENTIFIER" true) (atom "ANSI_NULLS" true) (atom "ANSI_PADDING" true)) (atom "ON" true)) (list "supported-option"))
		(parser '((atom "SET" true) (atom "IDENTITY_INSERT" true) (define target object) (or (atom "ON" true) (atom "OFF" true)))
			(list "identity-insert" target))
	)))
	(define p (parser (or conditional_statement statement (parser empty (list "empty")))))
	(if (and planning_session (planning_session "tsql_describe_ast"))
		((parser (define query tsql_select) (tsql_bind_query_types (sql_expand_views (tsql_resolve_query query policy planning_session) policy)) "^(?:/\\*(?s:.*?)\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+") s)
		((parser (define parsed p) (begin
			(define command (tsql_compile_statement schema parsed policy planning_session tx))
			(if (and planning_session (planning_session "tsql_describe_only"))
				(match command ((symbol quote) descriptions) command
					(cons (symbol tsql_insert_identity_scope) _insert)
					(if (planning_session "tsql_prepare_describe") (list (quote quote) '())
						(error "prepared result metadata currently requires SELECT"))
					_ (error "prepared result metadata currently requires SELECT"))
				(tsql_statement_formula command))) "^(?:/\\*(?s:.*?)\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+") s))
)))

/* Selection compiles the parsed statement AST against the current invocation,
then uses the ordinary execution bindings and its existing transaction. */
(define tsql_execute_statement_ast (lambda (schema statement policy session tx resultrow resultfields)
	(with_session session (lambda ()
		(sql_execute_formula session tx (sql_queryplan_compile_formula tx
			(sql_queryplan_bind_tx_calls (sql_queryplan_bind_execution_session
				(tsql_statement_formula (tsql_compile_statement schema statement policy session tx))))) resultrow resultfields)))))

/* Import aliases retarget object references, never literals. The policy wrapper
also rejects references to any database outside the fixed destination. */
(define tsql_import_database (lambda (session source) (begin
	(define previous (session "tsql_import_source"))
	(if (and previous (not (equal?? previous source))) (error "load_tsql accepts one source database per script") true)
	(session "tsql_import_source" source))))
(define load_tsql (lambda (schema source policy) (begin
	(if policy (policy schema true true) true)
	(createdatabase schema true)
	(define session (newsession))
	(session "schema" schema)
	(session "tsql_import_target" schema)
	(define import_policy (lambda (db name write)
		(if (equal?? db schema) (if policy (policy db name write) true)
			(error "load_tsql cannot access another database"))))
	(try (lambda () (begin
		(tsql_script_read source (lambda (command) (begin
			(with_autocommit session nil nil (tsql_guard_query command) (lambda (tx)
				(sql_execute_formula session tx (sql_queryplan_compile_formula tx
					(sql_queryplan_bind_execution_session (parse_tsql schema command import_policy session tx)))
					(lambda (row) true) (lambda columns true)))))) (lambda () (session "tsql_scope_identity" nil)))
		(if (not (nil? (session "transaction"))) (begin (tx_rollback session) (error "uncommitted transaction at end of import")) true)
		true))
		(lambda (failure) (begin (tx_rollback session) (error failure)))))))
