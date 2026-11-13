/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

(import "tsql-source-settings.scm")
(import "tsql-filesystem.scm")
(import "tsql-server-info.scm")
(import "tsql-permissions.scm")
(import "tsql-temporary.scm")
(import "tsql-databases.scm")
(import "tsql-row-guid.scm")

/* Independent T-SQL grammar. Statements emit the canonical query AST and
existing native DDL/DML operations, as the other SQL frontends do. */
/* A declaration completes successfully without an affected-row count. This
is frontend completion data, not a SQL scalar or a native operation result.
MS-TDS DONE_COUNT distinguishes absence from the valid DML count zero:
https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tds/3c06f110-98bd-4d5b-b836-b1ba66452cb7 */
(define tsql_no_count_completion (lambda () (list "tsql-completion" "no-count")))
(define tsql_completion_count_absent? (lambda (result) (match result
	'("tsql-completion" "no-count") true _ false)))
(define tsql_declaration_formula (lambda (formula)
	(list (quote !begin) formula (list (quote quote) (tsql_no_count_completion)))))

/* Integer tokens and integer text conversions use native int64 arithmetic.
The negative accumulator also represents the minimum int64 without overflow. */
/* Current-context form only; SID lookup and impersonation are separate features. */
(define tsql_security_context_name (lambda (name)
	(if (nil? name) nil
		(and (string? name) (<= (utf16_length name) 128)) name
		(error "login name is outside the supported NVARCHAR(128) domain"))))
/* Genuine native administrators map to dbo; ordinary database members retain
their authenticated name without an additional frontend owner role.
No creator identity is retained in a default expression or shared plan. */
(define tsql_database_principal_name (close_procedure ((lambda (account_id native_admin validate)
	(lambda (database username incarnation) (begin
		(if (nil? username) nil (begin
			(define identity (account_id username))
			(if (or (nil? identity) (and incarnation (not (equal? identity incarnation))))
				(error "database principal context changed") true)
			(validate (if (native_admin username) "dbo" username)))))))
	tsql_account_id tsql_native_admin? tsql_security_context_name)))
/* An unavailable transport claim is not the server's host or application. */
(define tsql_client_context_name (close_procedure ((lambda (validate)
	(lambda (value) (if (nil? value) (error "client connection metadata is unavailable")
		(validate value)))) tsql_security_context_name)))
/* Invocation-dependent expressions must not be sampled by optional folding.
Quoted data is not executable syntax. Relational expressions remain forbidden. */
(define tsql_default_dynamic? (lambda (expression) (match expression
	((symbol quote) _value) false
	(cons (symbol session) _args) true
	(cons (symbol now) _args) true
	(cons head tail) (or (tsql_default_dynamic? head)
		(reduce tail (lambda (found child) (or found (tsql_default_dynamic? child))) false))
	_ false)))
/* Persisted defaults cannot capture batch/prepared variables. Only the
context intrinsics' explicit invocation reads are meaningful in later scopes. */
(define tsql_default_expression_allowed? (lambda (expression)
	(and (tsql_scalar_expression_allowed? expression) (match expression
		((symbol quote) _value) true
		(cons (symbol tsql_batch_variable) _args) false
		(cons (symbol tsql_unbound_parameter) _args) false
		((symbol session) key)
		(has? '("username" "__tsql_principal_id" "schema" "__tsql_client_hostname" "__tsql_client_appname" "__tsql_language" "__tsql_dateformat" "__tsql_datefirst") key)
		(cons head tail) (and (tsql_default_expression_allowed? head)
			(reduce tail (lambda (allowed child) (and allowed (tsql_default_expression_allowed? child))) true))
		_ true))))
/* Parsing a stored scalar expression has its own syntax flags while schema,
source and permission guards still belong to the enclosing write plan. */
(define tsql_default_planning_scope (lambda (parent) (begin
	(define parent (coalesceNil parent (newsession)))
	(lambda args (match args
		'(key) (if (equal? key "__tsql_expression_only") true
			(has? '("__tsql_script_syntax" "__tsql_expression_syntax" "__tsql_logical_query"
				"__tsql_logical_update" "__tsql_scalar_function_body" "__tsql_parameter_analysis" "__tsql_describe"
				"__tsql_expression_source" "__tsql_lexical_context" "__tsql_parameter_declarations") key) nil
			(parent key))
		_ (apply parent args))))))
(define tsql_deferred_default_expr (lambda (database source policy planning tx) (begin
	(define options (tsql_default_planning_scope planning))
	(define info (parse_tsql database source policy options tx))
	(if (tsql_default_expression_allowed? (sql_info_formula info)) true
		(error "deferred default requires row- and parameter-free scalar SQL"))
	(list (quote sql_type_expr) (sql_info_formula info)
		(sql_info_datatype info) (sql_info_collation info)))))
(define tsql_raise_error (lambda (number message state)
	(error (list "tsql-error" number message state))))
(define tsql_parse_integer (close_procedure ((lambda (tsql_raise_error)
	(lambda (text) (begin
		(define trimmed (sql_trim text))
		(if (equal? trimmed "") 0 (begin
			(if (regexp_test trimmed "^[+-]?[0-9]+$") true (tsql_raise_error 245 "invalid integer text" 1))
			(define negative (equal? (substr trimmed 0 1) "-"))
			(define digits (regexp_replace trimmed "^[+-]?0*" ""))
			(define bound (if negative "9223372036854775808" "9223372036854775807"))
			(if (or (> (strlen digits) 19) (and (equal? (strlen digits) 19) (> digits bound)))
				(tsql_raise_error 8115 "integer conversion overflow" 1) true)
			(define coefficient (reduce (produceN (strlen digits)) (lambda (acc i)
				(- (* acc 10) (intdiv (simplify (substr digits i 1)) 1))) 0))
			(if negative coefficient (- 0 coefficient))))))) tsql_raise_error)))
(define tsql_int (parser (define value (regex "-?[0-9]+")) (tsql_parse_integer value)))
(define tsql_number (parser (define value (regex "-?(?:[0-9]+\\.?[0-9]*|\\.[0-9]+)(?:e-?[0-9]+)?" true))
	(begin
		(define integer (regexp_test value "^-?[0-9]+$"))
		(define number (if integer (tsql_parse_integer value) (simplify value)))
		/* Syntax owns the SQL contract, including numeric zero and one.
		The shared scalar equality considers these equal to Boolean values. */
		(list (quote sql_type_expr) number
			(if integer (if (and (>= number -2147483648) (<= number 2147483647)) "INT" "BIGINT")
				(if (regexp_test value "[eE]") "FLOAT" "DECIMAL")) nil))))
/* Regular identifiers use the Unicode identifier categories observed by
SQL Server: letters/Nl initially, then decimal digits, Mn/Mc/Pc/Cf as well.
The runtime's Unicode version supplies categories; this does not freeze its
character age to SQL Server's documented Unicode 3.2. Delimited names retain
their existing grammar. Supplementary regular characters are not admitted.
Reference: https://learn.microsoft.com/sql/relational-databases/databases/database-identifiers */
(define tsql_identifier_bmp (lambda (name)
	(if (regexp_test name "[\\x{10000}-\\x{10FFFF}]")
		(error "supplementary characters are not supported in regular identifiers") name)))
/* Both SQL tokens and runtime object addresses consume these same lexical
patterns. Only SQL parsing constructs native parser objects; portable address
validation closes over plain pattern strings. Statement keywords must not be
consumed as implicit aliases when a batch omits semicolons. */
/* OFFSET is a contextual ORDER BY clause word, not a reserved identifier.
The ordering suffix owns that clause; aliases, fields and object names may use
this spelling without delimiters. Reference:
https://learn.microsoft.com/sql/t-sql/language-elements/reserved-keywords-transact-sql */
(define tsql_regular_identifier_patterns (list "[\\p{L}\\p{Nl}_#][\\p{L}\\p{Nl}\\p{Nd}\\p{Mn}\\p{Mc}\\p{Pc}\\p{Cf}@$#]*"
	"(?:NOT|IN|BETWEEN|AS|ON|WHERE|GROUP|BY|VALUES|FROM|LEFT|RIGHT|INNER|OUTER|CROSS|JOIN|SELECT|UNION|ALL|INSERT|INTO|SET|ORDER|LIMIT|TRIM|LTRIM|RTRIM|TOP|FETCH|ROWS|ROW|ONLY|CASE|WHEN|THEN|ELSE|END|HAVING|WITH|OPTION|IF|BEGIN|DECLARE|CURSOR|OPEN|CLOSE|DEALLOCATE|EXEC|EXECUTE|CREATE|ALTER|DROP|UPDATE|DELETE|COMMIT|ROLLBACK|WHILE|BREAK|CONTINUE|GOTO|RETURN|THROW|PRINT|RAISERROR)(?:[^\\p{L}\\p{Nl}\\p{Nd}\\p{Mn}\\p{Mc}\\p{Pc}\\p{Cf}@$#]|$)"))
(define tsql_regular_identifier? (close_procedure ((lambda (regular keyword)
	(lambda (name) (and
		(regexp_test name (concat "^(?:" regular ")$"))
		(not (regexp_test name (concat "(?i)^(?:" keyword ")")))
		(not (regexp_test name "[\\x{10000}-\\x{10FFFF}]")))))
	(car tsql_regular_identifier_patterns) (cadr tsql_regular_identifier_patterns))))
/* Materialize syntax only at frontend bootstrap, before any runtime callable.
The declaration patterns remain the single policy source. */
(define tsql_identifier_unquoted
	(eval (list 'parser
		(list 'define 'name (list 'not
			(list 'regex (car tsql_regular_identifier_patterns))
			(list 'regex (cadr tsql_regular_identifier_patterns) true)))
		(list tsql_identifier_bmp 'name))))
/* One setting grammar serves direct leaves and metadata-only admission.
The probe returns a tagged value so OFF is distinct from an unrelated leaf. */
(define tsql_format_only_setting (parser '((atom "SET" true) (atom "FMTONLY" true)
	(define enabled (or (parser (atom "ON" true) true) (parser (atom "OFF" true) false))))
	(list "format-only" enabled)))
(define tsql_format_only_probe (parser (define result (or
	(parser '((define setting tsql_format_only_setting) (? ";") (regex "$")) setting)
	(parser (regex "(?s).*" false false) nil))) result
	"^(?:/\\*(?s:.*?)\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+"))

(define tsql_identifier_quoted (parser (or
	(parser '("[" (define id (regex "(?:[^\\]]|\\]\\])+" false false)) "]") (replace id "]]" "]"))
	(parser '("\"" (define id (regex "(?:[^\"]|\"\")+" false false)) "\"") (replace id "\"\"" "\"")))))
(define tsql_identifier (close_procedure ((lambda (tsql_identifier_quoted tsql_identifier_unquoted sql_private_relation?)
	(parser (define name (or tsql_identifier_quoted tsql_identifier_unquoted))
		(if (sql_private_relation? name) (error "private SQL relation names cannot be referenced directly") name))) tsql_identifier_quoted tsql_identifier_unquoted sql_private_relation?)))
/* Resolve spelling at the frontend boundary. Native metadata owns table and
column names; views remain definitions in the existing system.views table.
Only consumed name bindings enter the shared plan guards, never catalog epochs. */
(define tsql_view_name (lambda (database name)
	(scan nil (table "system" false "views" false nil)
		(list 369436443803648 (scan_boundary "equal" "database" 0 0 true true "" false)
			/* Identifier equality differs from the catalog column's binary order.
			Supply the same SimpleFold order as the residual, not a SQL value collation. */
			(scan_boundary "equal" "name" 1 1 true true "unicode-simple-fold" false
				-1 '() (collate "unicode-simple-fold" false) "unicode-simple-fold:asc" false))
		(list database name) '("name") (lambda (candidate) (equal?? candidate name)) '("name")
		(lambda (acc canonical) (if (nil? acc) canonical (error "ambiguous view name: " name)))
		nil (lambda (a b) (if (nil? a) b (nil? b) a (error "ambiguous view name: " name))))))
(define tsql_native_relation_handle (close_procedure ((lambda (sql_private_relation?)
	(lambda (database name tx)
		/* SQL spellings use the public folded catalog. An owned temporary binding
		already carries its exact native identity, which is hidden from that catalog. */
		(table database false name (not (sql_private_relation? name)) tx))) sql_private_relation?)))
(define tsql_relation_name (lambda (database name tx)
	(coalesceNil (show (tsql_native_relation_handle database name tx) "name") (tsql_view_name database name) name)))
(define tsql_table_name (lambda (database name planning_session tx) (begin
	(define resolved (tsql_resolve_relation database name planning_session tx))
	(define database (car resolved))
	(define name (cadr resolved))
	(define canonical (show (tsql_native_relation_handle database name tx) "name"))
	(planner_record_guard_condition
		(list (quote equal?) (list (quote show) (list (quote tsql_native_relation_handle) database name (quote tx)) "name") canonical)
		planning_session)
	(coalesceNil canonical name))))
(define tsql_create_relation (lambda (database name columns options ifnotexists tx) (begin
	(tsql_validate_row_guid_definitions columns)
	(if (tsql_scalar_function_reserved? database name tx) (error "a scalar function already owns this object name: " name) true)
	(if (tsql_view_name database name) (error "a view already owns this relation name: " name) true)
	(define publish (lambda () (begin
		(define published_columns (map columns (lambda (column) (match column
			'("column" column_name type dimensions attributes)
			(list "column" column_name type dimensions (tsql_publish_alias_attributes type attributes))
			_ column))))
		(sql_create_table database name published_columns (merge options '("name_ignorecase" true)) ifnotexists tx))))
	(if (> (count (filter columns (lambda (column) (match column
		'("column" _name type _dimensions _attributes)
		(has? '("ROWVERSION" "TIMESTAMP") (toUpper type))
		_ false)))) 1) (error "only one generated version column is allowed")
		(find columns (lambda (column) (match column
			'("column" _name _type _dimensions attributes) (has_assoc? attributes "__tsql_alias_id") _ false)))
		(tsql_type_ddl database tx publish) (publish)))))
(define tsql_create_view_definition (lambda (tx database name dialect text query mode) (begin
	(if (tsql_scalar_function_reserved? database name tx) (error "a scalar function already owns this object name: " name) true)
	(if (show (table database false name true tx) "name") (error "a table already owns this relation name: " name) true)
	(define existing (tsql_view_name database name))
	(if (and (nil? existing) (equal? mode "alter")) (error "view does not exist: " name) true)
	(create_sql_view tx database (coalesceNil existing name) dialect text query
		(if (equal? mode "alter") "replace" mode)))))

/* A relative default owner is parser metadata until its object context is
known. Resolve it before policy checks or emitting native relation metadata;
explicit database strings remain distinct even when they equal the request. */
(define tsql_qualifier_database (lambda (qualifier context_database options) (begin
	(define database (match qualifier
		(symbol tsql_default_owner) context_database
		_ (coalesceNil qualifier context_database)))
	(define target (options "tsql_import_target"))
	(if (nil? target) database
		(or (equal? database target) (equal? database (options "tsql_import_source"))) target
		(error "import reference outside destination and source alias: " database)))))

(import "tsql-object-metadata.scm")

/* The @ prefix makes this a variable token; SQL keywords need no identifier
escaping here. Consume it as one token so whitespace cannot split the name. */
(define tsql_variable_name (parser (define name (regex "@[\\p{L}\\p{Nl}\\p{Nd}\\p{Mn}\\p{Mc}\\p{Pc}\\p{Cf}@$#]+"))
	(toLower (substr (tsql_identifier_bmp name) 1))))
(define tsql_string_token (parser '((define national (or
	(parser (atom "N'" true) true) (parser (atom "'" false) false)))
	(define value (regex "(?:''|[^'])*" false false)) (atom "'" false false))
	(list (replace value "''" "'") national)))
(define tsql_string (parser (define token tsql_string_token) (car token)))
(define tsql_text_literal (parser (define token tsql_string_token)
	(list (quote sql_type_expr) (car token) (if (cadr token) "NVARCHAR" "VARCHAR") nil)))
/* A multipart identifier is syntax data until its declared FROM name is
bound. Quoted identifiers containing dots remain one token, never split text. */
(define tsql_qualified_column_ref (lambda (database namespace relation column)
	(list 'sql_compile_column 'tsql_compile_column
		(list 'get_column (list 'qualified-relation database (coalesceNil namespace "dbo") relation) true column true))))
(define tsql_qualified_star (parser (or
	(parser '((? (define database tsql_identifier)) "." (? (define namespace tsql_identifier)) "."
		(define relation tsql_identifier) "." "*")
		(tsql_qualified_column_ref database namespace relation "*"))
	(parser '((? (define namespace tsql_identifier)) "." (define relation tsql_identifier) "." "*")
		(tsql_qualified_column_ref nil namespace relation "*")))))
(define tsql_column (parser (or
	(parser '((? (define database tsql_identifier)) "." (? (define namespace tsql_identifier)) "."
		(define relation tsql_identifier) "." (define name tsql_identifier))
		(tsql_qualified_column_ref database namespace relation name))
	(parser '((? (define namespace tsql_identifier)) "." (define relation tsql_identifier) "." (define name tsql_identifier))
		(tsql_qualified_column_ref nil namespace relation name))
	(parser '((define alias tsql_identifier) "." (define name tsql_identifier))
		(list (quote sql_compile_column) (quote tsql_compile_column) '('get_column alias true name true)))
	(parser (define name tsql_identifier)
		(list (quote sql_compile_column) (quote tsql_compile_column) '('get_column nil true name true))))))
(define tsql_hex_literal (parser (define value sql_hex_literal)
	(list (quote sql_type_expr) value "VARBINARY" nil)))
(define tsql_literal (parser (or
	(parser (atom "NULL" true) (sql_null_literal)) tsql_hex_literal tsql_number tsql_text_literal)))
(define tsql_parameter_default_literal (parser (or tsql_literal
	(parser '("+" (define value tsql_number)) value))))

/* Binary semantics are chosen from declared SQL contracts at the shared
compile boundary. Byte equality reuses the native collation-aware operator;
no SQL type wrapper survives into execution or storage. */
(define tsql_character_type? (lambda (type)
	(has? '("NULL" "CHAR" "VARCHAR" "NCHAR" "NVARCHAR") type)))
(define tsql_binary_type? (lambda (type)
	(has? '("BINARY" "VARBINARY" "IMAGE" "ROWVERSION") type)))
/* Stored logical expressions carry global compiler names or factory syntax.
Only the consuming compilation creates callbacks; SQL operands stay separate. */
(define tsql_comparison_compiler (lambda (op) (lambda (infos) (begin
	(define left_info (car infos))
	(define right_info (cadr infos))
	(define left_type (sql_info_type left_info))
	(define right_type (sql_info_type right_info))
	(define head (match op "eq" (quote equal??) "ne" (quote equal??)
		"lt" (quote <) "gt" (quote >) "le" (quote <=) "ge" (quote >=)))
	(define binary (and
		(or (tsql_binary_type? left_type) (equal? left_type "NULL"))
		(or (tsql_binary_type? right_type) (equal? right_type "NULL"))
		(or (tsql_binary_type? left_type) (tsql_binary_type? right_type))))
	(define collation (if (and (sql_text_type? left_type) (sql_text_type? right_type))
		(sql_merge_collation (sql_info_collation left_info) (sql_info_collation right_info)) nil))
	/* Explicit COLLATE also governs constants and direct column comparisons.
	Do not lose its contract when a postfix annotation lowers to a literal. */
	(define explicit (and (not (nil? collation)) (equal? (cadr collation) 0)))
	/* Only character equality uses space padding in this slice. Keep binary
	payloads and range comparisons on their existing contracts. The opaque
	scalar residual cannot supply an incompatible raw column/index bound. */
	(define padded (and (has? '("eq" "ne") op)
		(tsql_character_type? left_type) (tsql_character_type? right_type)
		(not (and (equal? left_type "NULL") (equal? right_type "NULL")))))
	(define comparison (if padded
		(sql_info (list tsql_padded_text_equal (sql_info_formula left_info) (sql_info_formula right_info)
			(if (nil? collation) "bin" (car collation))) "BOOLEAN" collation)
		(or binary explicit)
		(sql_info (if (has? '("eq" "ne") op)
			(list (quote equal_collate) (sql_info_formula left_info) (sql_info_formula right_info)
				(if binary "bin" (car collation)))
			(list (quote sql_compare) (sql_info_formula left_info) (sql_info_formula right_info)
				(collate (if binary "bin" (car collation)) false) (string head))) "BOOLEAN" collation)
		(sql_comparison_info head left_info right_info)))
	(sql_info (if (equal? op "ne") (list (quote sql_not) (sql_info_formula comparison))
		(sql_info_formula comparison)) "BOOLEAN" (sql_info_collation comparison))))))
/* Predicate syntax supplies its Boolean contract before the shared logical
markers are decorrelated. No executable annotation survives SQL compilation. */
(define tsql_predicate_expr (lambda (formula)
	(list (quote sql_type_expr) formula "BOOLEAN" nil)))
(define tsql_comparison_expr (lambda (op left right)
	(list (quote sql_compile_expr) (list (quote tsql_comparison_compiler) op) left right)))
/* Preserve the syntactic NULL/variable boundary before binding and folding.
OFF never turns a column/column or compound/compound comparison null-safe. */
(define tsql_null_comparison_operand? (lambda (expression)
	(or (logical_sql_null_literal? expression) (match expression
		((symbol tsql_batch_variable) _name) true
		((symbol sql_type_expr) ((symbol session) key) _type _collation)
		(and (string? key) (strlike key "__tsql_parameter:%"))
		((symbol sql_type_expr) variable _type _collation) (symbol? variable)
		_ false))))
(define tsql_null_comparison_compiler (lambda (op) (lambda (infos) (begin
	(define left (car infos)) (define right (cadr infos))
	(define typed (lambda (info variable) (sql_info variable (sql_info_datatype info) (sql_info_collation info))))
	(define normal ((tsql_comparison_compiler op) (list (typed left '__ansi_left) (typed right '__ansi_right))))
	(define both (list 'and (list 'nil? '__ansi_left) (list 'nil? '__ansi_right)))
	(sql_info (list (list 'lambda (list '__ansi_left '__ansi_right)
		(list 'if (list 'or (list 'nil? '__ansi_left) (list 'nil? '__ansi_right))
			(if (equal? op "eq") both (list 'not both)) (sql_info_formula normal)))
		(sql_info_formula left) (sql_info_formula right)) "BOOLEAN" (sql_info_collation normal))))))
(define tsql_comparison_expr_using (lambda (ansi op left right)
	(if (and (not ansi) (has? '("eq" "ne") op)
		(or (tsql_null_comparison_operand? left) (tsql_null_comparison_operand? right)))
		(list 'sql_compile_expr (list 'tsql_null_comparison_compiler op) left right)
		(tsql_comparison_expr op left right))))
/* Literal lists have resolved element contracts. A numeric/mixed list keeps
native IN; dynamic arrays and subqueries never pass through this compiler. */
(define tsql_literal_in_compiler (lambda (infos) (begin
	(define probe (car infos))
	(define options (cdr infos))
	(define padded (and
		(reduce infos (lambda (valid info)
			(and valid (tsql_character_type? (sql_info_type info)))) true)
		(reduce infos (lambda (text info) (or text (not (equal? (sql_info_type info) "NULL")))) false)))
	(define values (cons list (map options sql_info_formula)))
	(if padded (begin
		(define collation (reduce infos (lambda (current info)
			(sql_merge_collation current (sql_info_collation info))) nil))
		(sql_info (list tsql_padded_text_in values (sql_info_formula probe)
			(if (nil? collation) "bin" (car collation))) "BOOLEAN" collation))
		(sql_info (list 'sql_in values (sql_info_formula probe)) "BOOLEAN" nil)))))
(define tsql_in_expr (lambda (values probe) (match values
	(cons head options) (if (identical? head list)
		(cons 'sql_compile_expr (cons 'tsql_literal_in_compiler (cons probe options)))
		(tsql_predicate_expr (list 'sql_in values probe)))
	_ (tsql_predicate_expr (list 'sql_in values probe)))))

(define tsql_column_type (parser (define name tsql_identifier) (toUpper name)))
(define tsql_storage_type (lambda (name)
	/* Text declarations remain native metadata. Storage chooses its value
	encoding independently; the frontend owns national-character width. */
	(match name "BIT" "BOOLEAN" "TIMESTAMP" "ROWVERSION" _ name)))
(define tsql_type_dimensions (parser (or
	(parser '("(" (atom "MAX" true) ")") (list -1))
	(parser '("(" (define precision tsql_int) "," (define scale tsql_int) ")") (list precision scale))
	(parser '("(" (define length tsql_int) ")") (list length))
	(parser empty (list)))))
/* The same type grammar is used for RPC declarations and expression casts. */
(define tsql_declared_type (parser '(
	(? (define namespace tsql_identifier) ".") (define type tsql_column_type)
) (if (nil? namespace) type
		(equal?? namespace "sys") (if (equal? type "SYSNAME") "[sys].[SYSNAME]" type)
		(error "unsupported type namespace: " namespace))))
/* Column declarations additionally admit persisted alias names. Cast/RPC
contracts keep their existing native-type grammar until alias contracts there
can be bound for the full procedure lifetime. */
(define tsql_schema_type (parser (define parts (+ tsql_identifier "."))
	/* Preserve token boundaries, including an unqualified quoted alias whose
	name itself contains dots, whitespace or closing brackets. */
	(reduce parts (lambda (prefix part)
		(concat prefix (if (equal? prefix "") "" ".") "[" (replace part "]" "]]") "]")) "")))
(define tsql_unique_parameters (lambda (parameters) (begin
	(define names (map parameters car))
	(if (equal? (count names) (count (merge_unique (list names)))) parameters
		(error "duplicate procedure parameter declaration")))))
/* Catalog values describe SQL declarations. COL_LENGTH uses native point
metadata and the frontend alias catalog; application values and compressed
widths never determine a declared SQL width. */
(define tsql_declared_byte_length (lambda (column) (begin
	(define type (toUpper (column "RawType")))
	(define dimensions (column "Dimensions"))
	(define width (if (empty_list? dimensions) 1 (car dimensions)))
	(match type
		"INT" 4 "INTEGER" 4 "BIGINT" 8 "SMALLINT" 2 "TINYINT" 1
		"BIT" 1 "BOOLEAN" 1 "REAL" 4 "DOUBLE" 8 "MONEY" 8 "SMALLMONEY" 4
		"DATE" 3 "DATETIME" 8 "SMALLDATETIME" 4 "UNIQUEIDENTIFIER" 16
		"ROWVERSION" 8 "TIMESTAMP" 8 "TEXT" 16 "NTEXT" 16 "IMAGE" 16
		_ (if (has? '("CHAR" "VARCHAR" "BINARY" "VARBINARY") type) width
			(has? '("NCHAR" "NVARCHAR") type) (if (equal? width -1) -1 (* width 2))
			(has? '("DECIMAL" "NUMERIC") type)
			(if (<= width 9) 5 (<= width 19) 9 (<= width 28) 13 17)
			(equal? type "FLOAT") (if (and (not (empty_list? dimensions)) (<= width 24)) 4 8)
			(has? '("TIME" "DATETIME2" "DATETIMEOFFSET") type) (begin
				(define precision (if (empty_list? dimensions) 7 width))
				(+ (if (<= precision 2) 3 (<= precision 4) 4 5)
					(match type "DATETIME2" 3 "DATETIMEOFFSET" 5 _ 0)))
			nil)))))
(define tsql_column_declaration_tuple (lambda (column)
	(list (column "RawType") (coalesceNil (column "Dimensions") '())
		(column "Null") (coalesceNil (column "Collation") ""))))
/* FK eligibility is SQL declaration policy. Retain exactly the native facts
consumed below so CREATEFOREIGNKEY can revalidate them under its DDL owner;
no row data or executable comparison policy enters the declaration. */
(define tsql_foreign_column_facts (lambda (column)
	(list (column "Field") (column "RawType") (column "Dimensions")
		(column "Collation") (column "Null") (column "Computed"))))
(define tsql_foreign_columns (lambda (handle names tx) (begin
	(define declarations ((show handle "declarations" tx) "columns"))
	(define columns (map names (lambda (name) (begin
		(define matches (filter declarations (lambda (column) (equal?? (column "Field") name))))
		(if (empty_list? matches) (error "unknown foreign key column: " name)
			(equal? (count matches) 1) (car matches)
			(error "ambiguous foreign key column: " name))))))
	(reduce columns (lambda (seen column)
		(if (has? seen (column "Field")) (error "duplicate foreign key column: " (column "Field"))
			(cons (column "Field") seen))) '())
	columns)))
/* Reuse the frontend's ordinary alias declaration and cache guard. The
original native columns remain the options precondition; effective SQL facts
are consumed only by this compiler, never stored as rewritten native types. */
(define tsql_foreign_effective_column (lambda (column planning_session) (begin
	(define effective (tsql_effective_column column planning_session))
	(if (effective "AliasId")
		(tsql_guard_type_alias (effective "DeclaredType") (effective "AliasId") planning_session) true)
	effective)))
(define tsql_validate_foreign_columns (lambda (child parent update_mode delete_mode planning_session) (begin
	(if (and (> (count child) 0) (equal? (count child) (count parent))) true
		(error "foreign key column count does not match reference"))
	(mapIndex child (lambda (i native_column) (begin
		(define column (tsql_foreign_effective_column native_column planning_session))
		(define target (tsql_foreign_effective_column (nth parent i) planning_session))
		(define type (toUpper (column "RawType")))
		(if (and (not (column "Computed")) (not (target "Computed"))) true
			(error "foreign keys on computed columns are not supported"))
		(if (and (equal? type (toUpper (target "RawType")))
			(expression_equal? (column "Dimensions") (target "Dimensions"))) true
			(error "foreign key SQL declarations do not match"))
		/* Temporal keys use the ordinary canonical numeric comparison.
		Text consumes the same declared collation as native UNIQUE/FK key
		comparison; mismatched endpoint contracts cannot borrow that policy. */
		(if (has? '("INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT" "BOOLEAN" "BIT" "UUID" "UNIQUEIDENTIFIER" "BINARY" "VARBINARY" "DATETIME" "SMALLDATETIME") type) true
			(has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR") type)
			(if (equal?? (column "Collation") (target "Collation")) true
				(error "foreign key text equality requires matching declared collations"))
			(error "foreign key SQL datatype is not supported: " type))
		(if (and (or (equal? update_mode "set null") (equal? delete_mode "set null"))
			(not (column "Null"))) (error "SET NULL requires nullable foreign key columns") true))))
	true)))
/* Historical conversion and tightening require native admission/validation.
The frontend accepts only unchanged contracts, text-width widening, same-scale
numeric precision widening, and relaxed NULL policy. No frontend row scans. */
(define tsql_compatible_column_declaration? (lambda (previous type dimensions nullable collation) (begin
	(define previous_type (toUpper (previous "RawType")))
	(define previous_dimensions (coalesceNil (previous "Dimensions") '()))
	(and (equal? type previous_type)
		(or nullable (not (previous "Null")))
		(equal? collation (coalesceNil (previous "Collation") ""))
		(if (expression_equal? dimensions previous_dimensions) true
			(and (has? '("VARCHAR" "NVARCHAR" "VARBINARY") type)
				(equal? (count dimensions) 1) (equal? (count previous_dimensions) 1))
			(or (equal? (car dimensions) -1)
				(and (not (equal? (car previous_dimensions) -1))
					(>= (car dimensions) (car previous_dimensions))))
			(and (has? '("DECIMAL" "NUMERIC") type)
				(equal? (count dimensions) 2) (equal? (count previous_dimensions) 2))
			(and (equal? (cadr dimensions) (cadr previous_dimensions))
				(>= (car dimensions) (car previous_dimensions)))
			false)))))
(define tds_bind_declarations (lambda (_schema text)
	(if (equal? (strtrim text) "") (list)
		((parser (define declarations (+ (parser '(
			(define name tsql_variable_name)
			(define type tsql_declared_type) (define dimensions tsql_type_dimensions)
			(define output (? (or (atom "OUTPUT" true) (atom "OUT" true))))
		) (if output (list (toLower name) (tsql_parameter_declaration _schema type dimensions) true)
				(list (toLower name) (tsql_parameter_declaration _schema type dimensions)))) ","))
			(tsql_unique_parameters declarations) "^(?:/\\*.*?\\*/|--[^\r\n]*(?:[\r\n]|$)|[\r\n\t ]+)+") text))))
/* Identity eligibility is a declaration-time dialect rule. Allocate values
with the existing native auto_increment option, without another allocator or
runtime SQL type checks. Numeric declarations may only have zero scale. */
(define tsql_identity_attributes (lambda (type dimensions attributes)
	(if (attributes "auto_increment") (begin
		(if (or (has? '("TINYINT" "SMALLINT" "INT" "INTEGER" "BIGINT") type)
			(and (has? '("DECIMAL" "NUMERIC") type) (equal? (cadr dimensions) 0)))
			true (error "IDENTITY requires an integer or zero-scale numeric declaration"))
		(if (attributes "null") (error "IDENTITY column was declared NULL") true)
		(if (find (extract_assoc attributes (lambda (key _value) key))
			(lambda (key) (has? '("default" "default_expression") key)))
			(error "IDENTITY cannot have a DEFAULT declaration") true)
		(append attributes "null" false)) attributes)))
/* This built-in alias has one immutable frontend declaration, not a catalog
row or another storage type. Source declarations keep their original name;
consumers resolve the existing parameter/column contract before compilation. */
(define tsql_builtin_type_declaration (lambda (type)
	(if (has? '("SYSNAME" "[SYS].[SYSNAME]") (toUpper type))
		(list "name" "sysname" "base_type" "NVARCHAR" "dimensions" '(128) "nullable" false) nil)))
(define tsql_parameter_declaration (lambda (database type dimensions) (begin
	(define dimensions (tsql_decl_dimensions type dimensions))
	(if (tsql_builtin_type_declaration type) (tsql_resolve_schema_type database type dimensions nil) true)
	(cons type dimensions))))
(define tsql_effective_parameter_contract (lambda (contract) (begin
	(define builtin (tsql_builtin_type_declaration (car contract)))
	(if builtin (cons (builtin "base_type") (builtin "dimensions")) contract))))
(define tsql_decl_dimensions (lambda (type dimensions) (begin
	(define count_dimensions (count dimensions))
	(if (tsql_builtin_type_declaration type)
		(if (empty_list? dimensions) dimensions (error "built-in alias dimensions cannot be overridden"))
		(has? '("VARCHAR" "CHAR" "NVARCHAR" "NCHAR" "BINARY" "VARBINARY") type)
		(begin
			(define length (if (empty_list? dimensions) 1 (car dimensions)))
			(if (and (<= count_dimensions 1)
				(or (and (equal? length -1) (has? '("VARCHAR" "NVARCHAR" "VARBINARY") type))
					(and (>= length 1) (<= length (if (has? '("NVARCHAR" "NCHAR") type) 4000 8000)))))
				(list length) (error "invalid length for " type)))
		(if (has? '("DECIMAL" "NUMERIC") type) (begin
			(define precision (if (empty_list? dimensions) 18 (car dimensions)))
			(define scale (if (< count_dimensions 2) 0 (cadr dimensions)))
			(if (and (<= count_dimensions 2) (>= precision 1) (<= precision 38) (>= scale 0) (<= scale precision))
				(list precision scale) (error "invalid precision or scale for " type)))
			(if (equal? type "FLOAT")
				(if (and (<= count_dimensions 1) (or (empty_list? dimensions) (and (>= (car dimensions) 1) (<= (car dimensions) 53))))
					dimensions (error "invalid FLOAT precision"))
				(if (has? '("TIME" "DATETIME2" "DATETIMEOFFSET") type)
					(if (and (<= count_dimensions 1) (or (empty_list? dimensions) (and (>= (car dimensions) 0) (<= (car dimensions) 7))))
						dimensions (error "invalid temporal precision"))
					(if (and (empty_list? dimensions) (has? '("INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT" "BIT" "REAL" "MONEY" "SMALLMONEY" "DATE" "DATETIME" "SMALLDATETIME" "ROWVERSION" "TIMESTAMP" "UNIQUEIDENTIFIER" "TEXT" "NTEXT" "IMAGE") type))
						dimensions (error "unsupported type or dimensions: " type)))))))))
/* These conversions are frontend scalar expressions. Stored values and the
shared planner continue to use the ordinary native numbers and booleans. */
/* TODO: Generic multi-precision integers would permit exact DECIMAL(38,s).
Represent decimals as integer coefficients and apply scale corrections through
the frontend's compile-time type system. Keep precision/scale out of the Scheme
runtime and storage hot paths; do not substitute float64 for exact casts. */
(define tsql_float_max (simplify "1.7976931348623157e308"))
(define tsql_numeric_input (close_procedure ((lambda (tsql_float_max tsql_raise_error)
	(lambda (value)
		(if (number? value) (if (and (>= value (- 0 tsql_float_max)) (<= value tsql_float_max)) value (tsql_raise_error 8115 "floating conversion overflow" 1))
			(if (string? value) (begin
				(define text (sql_trim value))
				(if (equal? text "") 0
					(if (regexp_test text "^[+-]?(?:[0-9]+\\.?[0-9]*|\\.[0-9]+)(?:[eE][+-]?[0-9]+)?$")
						(begin (define number (simplify text))
							(if (and (number? number) (>= number (- 0 tsql_float_max)) (<= number tsql_float_max)) number
								(tsql_raise_error 8115 "floating conversion overflow" 1)))
						(tsql_raise_error 8114 "invalid numeric text" 1))))
				(if (equal? value true) 1 (if (equal? value false) 0 (error "unsupported numeric conversion"))))))) tsql_float_max tsql_raise_error)))
(define tsql_cast_integer (close_procedure ((lambda (tsql_parse_integer tsql_numeric_input tsql_raise_error)
	(lambda (value minimum maximum)
		(if (nil? value) nil (begin
			(define integer (if (string? value) (tsql_parse_integer value)
				(if (int? value) value (intdiv (tsql_numeric_input value) 1))))
			(if (or (nil? integer) (< integer minimum) (> integer maximum))
				(tsql_raise_error 8115 "integer conversion overflow" 1) integer))))) tsql_parse_integer tsql_numeric_input tsql_raise_error)))
(define tsql_cast_float (close_procedure ((lambda (tsql_numeric_input)
	(lambda (value) (if (nil? value) nil (simplify (tsql_numeric_input value))))) tsql_numeric_input)))
(define tsql_cast_decimal (close_procedure ((lambda (tsql_parse_integer tsql_numeric_input tsql_cast_integer tsql_raise_error)
	(lambda (value precision scale) (if (nil? value) nil (begin
		(if (and (string? value) (equal? (sql_trim value) ""))
			(tsql_raise_error 8114 "invalid numeric text" 1) true)
		/* The declaration selects the integer carrier for scale zero. Preserve an
		already integral input rather than route it through floating-point round. */
		(define number (if (and (string? value) (regexp_test (sql_trim value) "^[+-]?[0-9]+$"))
			(tsql_parse_integer value) (tsql_numeric_input value)))
		(define converted (if (equal? scale 0)
			(tsql_cast_integer (if (int? number) number (round number))
				(tsql_parse_integer "-9223372036854775808") (tsql_parse_integer "9223372036854775807"))
			(if (<= scale 15) (sql_decimal_output number scale)
				(error "decimal scale exceeds native numeric precision"))))
		/* Scale-zero values already use int64. Keep their bounds integral too:
		float64 rounds the last valid 18-digit value up to its exclusive bound. */
		(define valid (if (equal? scale 0)
			(if (>= precision 19) true (begin
				(define bound (tsql_parse_integer (concat "1" (string_repeat "0" precision))))
				(and (> converted (- 0 bound)) (< converted bound))))
			(begin
				(define bound (simplify (concat "1e" (- precision scale))))
				(and (> converted (- 0 bound)) (< converted bound)))))
		(if valid converted (tsql_raise_error 8115 "decimal conversion overflow" 1)))))) tsql_parse_integer tsql_numeric_input tsql_cast_integer tsql_raise_error)))
/* Currency declarations have fixed scale four. Values remain ordinary native
numbers; only formatting and transport use an unscaled integer coefficient.
The float carrier cannot preserve every coefficient at the 64-bit endpoints.
Reject an out-of-range conversion before intdiv can wrap its sign. */
(define tsql_money_coefficient (close_procedure ((lambda (tsql_numeric_input tsql_raise_error)
	(lambda (value small) (if (nil? value) nil (begin
		(define scaled (round (* (sql_decimal_output (tsql_numeric_input value) 4) 10000)))
		(define limit (simplify "9223372036854775808.0"))
		/* At either 64-bit endpoint the float carrier cannot distinguish the
		last valid coefficient from an adjacent overflow. Refuse that ambiguity. */
		(if (or (<= scaled (- 0 limit)) (>= scaled limit))
			(tsql_raise_error 8115 "currency conversion overflow" 1) true)
		(define coefficient (intdiv scaled 1))
		(if (and small (or (< coefficient -2147483648) (> coefficient 2147483647)))
			(tsql_raise_error 8115 "small currency conversion overflow" 1) coefficient))))) tsql_numeric_input tsql_raise_error)))
(define tsql_cast_money (close_procedure ((lambda (tsql_money_coefficient)
	(lambda (value small)
		(if (nil? value) nil (/ (tsql_money_coefficient value small) 10000)))) tsql_money_coefficient)))
(define tsql_money_text_input (close_procedure ((lambda (tsql_numeric_input tsql_raise_error)
	(lambda (value) (if (nil? value) nil (begin
		(define text (sql_trim value))
		(if (or (equal? text "")
			(regexp_test text "^(?:[+-]?\\p{Sc}?|\\p{Sc}[+-]?)(?:[0-9][0-9,]*(?:\\.[0-9]*)?|\\.[0-9]+)$")) true
			(tsql_raise_error 235 "invalid currency text" 1))
		(tsql_numeric_input (regexp_replace (replace text "," "") "\\p{Sc}" "")))))) tsql_numeric_input tsql_raise_error)))
(define tsql_money_compiler (lambda (small) (lambda (infos) (begin
	(define input (car infos))
	(define type (sql_info_type input))
	(define character (has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR" "TEXT" "NTEXT") type))
	(if (or (sql_numeric_type? type) character (has? '("MONEY" "SMALLMONEY" "NULL") type)) true
		(error "currency conversion from " type " is not implemented"))
	(define formula (sql_info_formula input))
	(sql_info (list tsql_cast_money
		(if character (list tsql_money_text_input formula) formula) small)
		(if small "SMALLMONEY" "MONEY") nil)))))
(define tsql_cast_money_expr (lambda (value small)
	(list (quote sql_compile_expr) (list (quote tsql_money_compiler) small) value)))
/* These bounded strings describe a scale-four currency value, not a second
numeric representation. Split before taking absolute values so the signed
64-bit minimum never needs to be negated. */
(define tsql_money_group_digits (lambda (text)
	/* This bounded scalar string uses no recursive global callback binding. */
	(reduce_range (strlen text) (lambda (result index)
		(concat result (if (and (> index 0) (equal? (mod (- (strlen text) index) 3) 0)) "," "")
			(substr text index 1))) "")))
(define tsql_money_text (close_procedure ((lambda (tsql_money_coefficient tsql_money_group_digits)
	(lambda (value style) (if (nil? value) nil (begin
		(define coefficient (tsql_money_coefficient value false))
		(define four (has? '(2 126) style))
		(define units (if four coefficient (begin
			(define quotient (intdiv coefficient 100))
			(define remainder (mod coefficient 100))
			(+ quotient (if (>= remainder 50) 1 (<= remainder -50) -1 0)))))
		(define factor (if four 10000 100))
		(define whole (intdiv units factor))
		(define fraction (mod units factor))
		(define digits (string (if (< whole 0) (- 0 whole) whole)))
		(define tail (string (if (< fraction 0) (- 0 fraction) fraction)))
		(concat (if (< units 0) "-" "") (if (equal? style 1) (tsql_money_group_digits digits) digits)
			"." (string_repeat "0" (- (if four 4 2) (strlen tail))) tail))))) tsql_money_coefficient tsql_money_group_digits)))

(define tsql_cast_binary (lambda (value length fixed) (if (nil? value) nil (begin
	(if (string? value) true (error "numeric binary conversion is not implemented"))
	(define bytes (if (or (equal? length -1) (<= (strlen value) length)) value (substr value 0 length)))
	(if (and fixed (< (strlen bytes) length))
		(concat bytes (string_repeat (hex2bin "00") (- length (strlen bytes)))) bytes)))))
/* Integer binary contracts use at most sixteen hexadecimal digits. The SQL
compiler supplies widths; no runtime value tag chooses a conversion rule.
Complementing the nonnegative magnitude avoids negating the int64 minimum. */
(define tsql_integer_width (lambda (type) (match type
	"TINYINT" 1 "SMALLINT" 2 "INT" 4 "INTEGER" 4 "BIGINT" 8 _ nil)))
(define tsql_integer_hex (lambda (value digits negative)
	(if (equal? digits 0) "" (concat
		(tsql_integer_hex (intdiv value 16) (- digits 1) negative)
		(substr "0123456789abcdef" (if negative (- 15 (mod value 16)) (mod value 16)) 1)))))
(define tsql_integer_to_binary (lambda (value width length fixed) (if (nil? value) nil (begin
	(define negative (< value 0))
	(define magnitude (if negative (- 0 (+ value 1)) value))
	(define bytes (hex2bin (tsql_integer_hex magnitude (* 2 width) negative)))
	(define kept (if (or (equal? length -1) (>= length width)) bytes
		(substr bytes (- width length) length)))
	(if (and fixed (> length width))
		(concat (string_repeat (hex2bin "00") (- length width)) kept) kept)))))
(define tsql_hex_digit_integer (lambda (digit) (match digit
	"0" 0 "1" 1 "2" 2 "3" 3 "4" 4 "5" 5 "6" 6 "7" 7
	"8" 8 "9" 9 "a" 10 "b" 11 "c" 12 "d" 13 "e" 14 "f" 15
	_ (error "invalid binary hexadecimal digit"))))
(define tsql_binary_to_integer (close_procedure ((lambda (tsql_hex_digit_integer)
	(lambda (value width signed) (if (nil? value) nil (begin
		/* Oversized inputs need a separately verified source-kind truncation contract.
		Do not silently guess which bytes the original dialect would discard. */
		(define count (strlen value))
		(if (> count width) (error "overwide binary integer conversion is not implemented") true)
		(define hex (bin2hex value))
		/* Seed the sign in the first nibble, before multiplication can overflow.
		Short inputs are zero extended; a full signed width uses two's complement. */
		(reduce_range (strlen hex) (lambda (acc index) (begin
			(define digit (tsql_hex_digit_integer (substr hex index 1)))
			(if (and signed (equal? count width) (equal? index 0) (>= digit 8))
				(- digit 16) (+ (* acc 16) digit)))) 0))))) tsql_hex_digit_integer)))
(define tsql_integer_compiler (lambda (type minimum maximum) (lambda (infos) (begin
	(define input (car infos))
	(sql_info (if (tsql_binary_type? (sql_info_type input))
		(list tsql_binary_to_integer (sql_info_formula input) (tsql_integer_width type) (< minimum 0))
		(list tsql_cast_integer
			(if (tsql_temporal_numeric_target? (sql_info_type input))
				(list tsql_legacy_integer_days (tsql_calendar_formula input))
				(has? '("MONEY" "SMALLMONEY") (sql_info_type input))
				(list tsql_money_integer (sql_info_formula input)) (sql_info_formula input)) minimum maximum)) type nil)))))
(define tsql_money_integer (lambda (value) (if (nil? value) nil (round value))))
(define tsql_cast_integer_expr (lambda (value type minimum maximum)
	(list (quote sql_compile_expr) (list (quote tsql_integer_compiler) type minimum maximum) value)))
(define tsql_cast_bit (close_procedure ((lambda (tsql_numeric_input)
	(lambda (value)
		(if (nil? value) nil
			(if (and (string? value) (equal? (toUpper (sql_trim value)) "TRUE")) true
				(if (and (string? value) (equal? (toUpper (sql_trim value)) "FALSE")) false
					(not (equal? (tsql_numeric_input value) 0))))))) tsql_numeric_input)))
/* VARCHAR uses the existing TDS CP1252 profile unless a real SQL UTF8
collation selects UTF8 explicitly. Internal comparison labels do not change
that wire profile. Unknown code pages fail before choosing any row operator. */
(define tsql_character_encoding (lambda (info) (begin
	(define collation (sql_info_collation info))
	(define name (if collation (toLower (car collation)) ""))
	(if (equal? name "") "windows-1252"
		(regexp_test name "_utf8$") "utf-8"
		(regexp_test name "^(utf8mb[34]|utf8|binary|bin)(_|$)") "windows-1252"
		(regexp_test name "^latin1_general(_|$)") "windows-1252"
		(regexp_test name "^sql_latin1_general_cp1(_|$)") "windows-1252"
		(error "unsupported T-SQL character code page for collation: " name)))))
(define tsql_encode_character (lambda (value encoding limit)
	(if (nil? value) nil
		(equal? encoding "utf-8") (utf8_prefix value (if (equal? limit -1) (strlen value) limit))
		(single_byte_encode value encoding limit))))
(define tsql_decode_character (lambda (value encoding)
	(if (nil? value) nil
		(equal? encoding "utf-8") (utf8_prefix value (strlen value))
		(single_byte_decode value encoding))))

/* Character casts own their width semantics in the frontend. National text
counts UTF-16 units and never returns half of a supplementary character. The
fast path avoids splitting the ordinary short numeric/text conversions. */
(define tsql_cast_text (close_procedure ((lambda (tsql_encode_character tsql_decode_character) (lambda (value length national encoding) (if (nil? value) nil (begin
	(define text (string value))
	(if national
		(if (or (equal? length -1) (<= (strlen text) length)) text (utf16_prefix text length))
		(tsql_decode_character (tsql_encode_character text encoding length) encoding)))))) tsql_encode_character tsql_decode_character)))
(define tsql_cast_fixed_text (close_procedure ((lambda (tsql_cast_text tsql_encode_character) (lambda (value length national encoding) (if (nil? value) nil (begin
	(define text (tsql_cast_text value length national encoding))
	(define units (if national (utf16_length text) (strlen (tsql_encode_character text encoding -1))))
	(concat text (string_repeat " " (- length units))))))) tsql_cast_text tsql_encode_character)))

/* A cast keeps its syntax-declared SQL type until the shared expression
compiler consumes the (expr, type, collation) contract. Runtime values do not
select that type, and no arithmetic operation is needed merely to mark it. */

(define tsql_cast_money_text (close_procedure ((lambda (tsql_money_text tsql_raise_error tsql_cast_fixed_text tsql_cast_text)
	(lambda (value style length national fixed) (if (nil? value) nil (begin
		(define text (tsql_money_text value style))
		(if (and (not (equal? length -1)) (> (strlen text) length))
			(tsql_raise_error 8115 "currency character conversion overflow" 1) true)
		((if fixed tsql_cast_fixed_text tsql_cast_text) text length national "windows-1252"))))) tsql_money_text tsql_raise_error tsql_cast_fixed_text tsql_cast_text)))

/* SQL_VARIANT owns its intrinsic subtype. The frontend chooses that subtype
from declared virtual property metadata, never from a runtime value carrier. */
(define tsql_variant_scalar (lambda (value) (if (nil? value) nil (value "value"))))
(define tsql_variant_text (lambda (value) (if (nil? value) nil
	(if (has? '(167 175 231 239) ((value "variant") "kind")) (value "value")
		(error "conversion from this SQL_VARIANT subtype to text is not implemented")))))
(define tsql_variant_bit (close_procedure ((lambda (cast_bit) (lambda (value) (if (nil? value) nil
	(if (has? '(38 104) ((value "variant") "kind")) (cast_bit (value "value"))
		(error "conversion from this SQL_VARIANT subtype to BIT is not implemented"))))) tsql_cast_bit)))
(define tsql_bit_cast_compiler (lambda (infos) (begin
	(define info (car infos))
	(sql_info (list (if (equal? (sql_info_type info) "SQL_VARIANT") tsql_variant_bit tsql_cast_bit)
		(sql_info_formula info)) "BIT" nil))))
(define tsql_serverproperty ((lambda (location profile) (lambda (property) (if (nil? property) nil (begin
	(define property (toLower property))
	(define selected (profile property))
	(define type (if selected (selected "type") "NVARCHAR"))
	(define value (if selected (selected "value") (match property
		"edition" "MemCP"
		"instancedefaultdatapath" (location "data")
		"instancedefaultlogpath" (location "log")
		_ nil)))
	(if (nil? value) nil
		(list "variant" (if (equal? type "INT")
			(list "kind" 38 "size" 4 "precision" 0 "scale" 0)
			(list "kind" 231 "size" 256 "precision" 0 "scale" 0 "collation" (hex2bin "0904d00034")))
			"value" value)))))) tsql_filesystem_location tsql_server_profile_property))

(define tsql_cast_expr (lambda (value type dimensions) (begin
	(define name (toUpper type))
	(define builtin (tsql_builtin_type_declaration name))
	(tsql_decl_dimensions name dimensions)
	(define dimensions (if builtin (builtin "dimensions") dimensions))
	(define name (if builtin (builtin "base_type") name))
	(define formula (match name
		"INT" (tsql_cast_integer_expr value name -2147483648 2147483647)
		"INTEGER" (tsql_cast_integer_expr value name -2147483648 2147483647)
		"SMALLINT" (tsql_cast_integer_expr value name -32768 32767)
		"TINYINT" (tsql_cast_integer_expr value name 0 255)
		"BIGINT" (tsql_cast_integer_expr value name (tsql_parse_integer "-9223372036854775808") (tsql_parse_integer "9223372036854775807"))
		"FLOAT" (list tsql_cast_float value)
		/* Approximate types use native floating precision. Binary32 rounding and
		dialect precision boundaries remain noncritical compatibility work. */
		"REAL" (list tsql_cast_float value)
		"MONEY" (tsql_cast_money_expr value false)
		"SMALLMONEY" (tsql_cast_money_expr value true)
		"BIT" (list (quote sql_compile_expr) (quote tsql_bit_cast_compiler) value)
		"UNIQUEIDENTIFIER" (list (quote tsql_uuid) value)
		"DECIMAL" (tsql_cast_decimal_expr value name dimensions)
		"NUMERIC" (tsql_cast_decimal_expr value name dimensions)
		"BINARY" (tsql_cast_binary_expr value dimensions true 0)
		"VARBINARY" (tsql_cast_binary_expr value dimensions false 0)
		"VARCHAR" (tsql_cast_text_expr value dimensions false false)
		"NVARCHAR" (tsql_cast_text_expr value dimensions true false)
		"CHAR" (tsql_cast_text_expr value dimensions false true)
		"NCHAR" (tsql_cast_text_expr value dimensions true true)
		"DATE" (tsql_temporal_cast_expr value name 0 dimensions)
		"DATETIME" (tsql_temporal_cast_expr value name 0 dimensions)
		"DATETIME2" (tsql_temporal_cast_expr value name 0 dimensions)
		"SMALLDATETIME" (tsql_temporal_cast_expr value name 0 dimensions)
		"TIME" (tsql_temporal_cast_expr value name 0 dimensions)
		_ (error "unsupported conversion to " name)))
	(if (has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR" "BINARY" "VARBINARY" "DECIMAL" "NUMERIC") name) formula
		(list (quote sql_type_expr) formula
			(if (empty_list? dimensions) name (list name (tsql_decl_dimensions name dimensions))) nil)))))
/* Assignment conversion is distinct from explicit CAST/CONVERT. Inspect the
SQL source contract before using the ordinary width/scale conversion compiler;
the native value carrier cannot establish whether a legacy date is an integer.
Explicit day-number conversion is valid, but assigning that date to an integer
requires the SQL user to request it explicitly. */
(define tsql_implicit_cast_compiler (lambda (type dimensions) (lambda (infos) (begin
	(define input (car infos))
	(define target (car (tsql_effective_parameter_contract (cons (toUpper type) dimensions))))
	(if (and (tsql_temporal_numeric_target? (sql_info_type input))
		(has? '("INT" "INTEGER" "SMALLINT" "TINYINT" "BIGINT") target))
		(tsql_raise_error 257 (concat "unsupported implicit conversion from " (sql_info_type input) " to " target) 3) true)
	(sql_expr_info '() (tsql_cast_expr
		(list 'sql_type_expr (sql_info_formula input) (sql_info_datatype input) (sql_info_collation input))
		type dimensions))))))
(define tsql_implicit_cast_expr (lambda (value type dimensions)
	(list 'sql_compile_expr (list 'tsql_implicit_cast_compiler type (list 'quote dimensions)) value)))
(define tsql_decimal_compiler (lambda (type dims) (lambda (infos) (begin
	(define input (car infos))
	(define input_type (sql_info_type input))
	(if (or (sql_numeric_type? input_type)
		(has? '("NULL" "any" "MONEY" "SMALLMONEY" "CHAR" "VARCHAR" "NCHAR" "NVARCHAR" "TEXT" "NTEXT") input_type)) true
		(error "decimal conversion from " input_type " is not implemented"))
	(sql_info (list tsql_cast_decimal (sql_info_formula input) (car dims) (cadr dims)) (list type dims) nil)))))
(define tsql_cast_decimal_expr (lambda (value type dimensions) (begin
	(define dims (tsql_decl_dimensions type dimensions))
	(list (quote sql_compile_expr) (list (quote tsql_decimal_compiler) type (list (quote quote) dims)) value))))
/* Hex styles are frontend conversions, independent of the byte storage.
Validate the complete input before truncation, then decode only needed bytes. */
(define tsql_hex_to_binary (lambda (value prefix length) (if (nil? value) nil (begin
	(if (and prefix (or (< (strlen value) 2) (not (equal? (substr value 0 2) "0x"))))
		(tsql_raise_error 8114 "invalid hexadecimal prefix" 1) true)
	(define hex (if prefix (substr value 2) value))
	(if (or (not (equal? (mod (strlen hex) 2) 0)) (not (regexp_test hex "^[0-9A-Fa-f]*$")))
		(tsql_raise_error 8114 "invalid hexadecimal input" 1) true)
	(hex2bin (if (equal? length -1) hex (substr hex 0 (min (strlen hex) (* 2 length)))))))))
(define tsql_binary_to_hex (lambda (value prefix length) (if (nil? value) nil (begin
	/* Bound formatting before encoding a potentially large binary value.
	Odd text widths need one final byte; the ordinary text cast truncates it. */
	(define available (if (equal? length -1) -1 (max 0 (- length (if prefix 2 0)))))
	(define bytes (if (equal? available -1) value
		(substr value 0 (min (strlen value) (intdiv (+ available 1) 2)))))
	(concat (if prefix "0x" "") (toUpper (bin2hex bytes)))))))
(define tsql_binary_compiler (lambda (length fixed style) (lambda (infos) (begin
	(define input (car infos))
	(define source (sql_info_type input))
	(define width (tsql_integer_width source))
	(if (if (equal? style 0)
		(or width (tsql_binary_type? source) (has? '("NULL" "any") source))
		(has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR" "TEXT" "NTEXT" "NULL") source)) true
		(error "binary conversion from " source " is not implemented"))
	(define formula (if (equal? style 0) (sql_info_formula input)
		(list (quote tsql_hex_to_binary) (sql_info_formula input) (equal? style 1) length)))
	(sql_info (if (and width (equal? style 0))
		(list (quote tsql_integer_to_binary) formula width length fixed)
		(list (quote tsql_cast_binary) formula length fixed))
		(list (if fixed "BINARY" "VARBINARY") (list length)) nil)))))
(define tsql_cast_binary_expr (lambda (value dimensions fixed style) (begin
	(if (has? '(0 1 2) style) true (error "unsupported binary conversion style"))
	(define length (car (tsql_decl_dimensions (if fixed "BINARY" "VARBINARY")
		(if (empty_list? dimensions) '(30) dimensions))))
	(list (quote sql_compile_expr) (list (quote tsql_binary_compiler) length fixed style) value))))
/* Assignment contracts come from native column declarations at compile time.
Integer narrowing is an ordinary frontend cast. Existing native decimal scale
normalization remains in use; this guard rejects overflow, including carry from
rounding, before native insertion. Full precision-limit emulation is deferred. */
(define tsql_assignment_decimal (lambda (value threshold) (if (nil? value) nil (begin
	(define number (tsql_numeric_input value))
	(if (or (>= number threshold) (<= number (- 0 threshold))) (tsql_raise_error 8115 "decimal conversion overflow" 1) number)))))
(define tsql_assignment_expr (lambda (value declaration) (begin
	(define type (toUpper (declaration "RawType")))
	(if (equal? type "ROWVERSION")
		(error "generated version column cannot be assigned explicitly")
		(declaration "AliasId") (tsql_implicit_cast_expr value type (declaration "Dimensions"))
		(has? '("BIT" "BOOL" "BOOLEAN") type) (tsql_implicit_cast_expr value "BIT" '())
		(has? '("INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT") type)
		(tsql_implicit_cast_expr value type '())
		/* Minute rounding belongs to the declared assignment contract, before
		the existing sink receives the native calendar value. */
		(has? '("CHAR" "VARCHAR" "NVARCHAR" "NCHAR" "DATE" "DATETIME" "DATETIME2" "TIME" "SMALLDATETIME" "UNIQUEIDENTIFIER") type)
		(tsql_implicit_cast_expr (if (and (has? '("CHAR" "VARCHAR" "NVARCHAR" "NCHAR") type) (declaration "Collation"))
			(list (quote sql_compile_expr) (list (quote tsql_collation_compiler) (declaration "Collation")) value) value)
			type (declaration "Dimensions"))
		(has? '("MONEY" "SMALLMONEY") type) (tsql_implicit_cast_expr value type '())
		(has? '("DECIMAL" "NUMERIC") type) (begin
			(define dimensions (declaration "Dimensions"))
			(define precision (if (empty_list? dimensions) 18 (car dimensions)))
			(define scale (if (< (count dimensions) 2) 0 (cadr dimensions)))
			(if (equal? scale 0) (tsql_implicit_cast_expr value type (list precision scale)) (begin
				(define threshold (- (simplify (concat "1e" (- precision scale)))
					(* 0.5 (simplify (concat "1e-" scale)))))
				(list (quote sql_type_expr) (list (quote tsql_assignment_decimal) value threshold) type nil))))
		value))))
/* DEFAULT is frontend syntax for the existing column declaration. Share this
translation between explicit INSERT cells and UPDATE assignments. Omitted
deferred cells are bound below; other omitted defaults remain native. */
(define tsql_prepare_default (lambda (options target compile_value) (begin
	(define source (options "__tsql_default_source"))
	(define clean (merge (extract_assoc options (lambda (key value)
		(if (equal? key "__tsql_default_source") '() (list key value))))))
	(if (has_assoc? options "default") (begin
		(define info (sql_expr_info '() (options "default")))
		(if (and (tsql_temporal_type? target) (sql_text_type? (sql_info_type info)))
			(begin
				(if source true (error "temporal text DEFAULT requires its SQL source"))
				(append (merge (extract_assoc clean (lambda (key value)
					(if (equal? key "default") '() (list key value)))))
					"default_expression" source "default_deferred" true))
			(set_assoc clean "default" (compile_value (options "default"))))) clean))))
(define tsql_default_assignment_expr (lambda (declaration database policy planning tx)
	(if (declaration "DefaultDeferred")
		(tsql_deferred_default_expr database (declaration "DefaultExpression") policy planning tx)
		(has? '("NEWID" "UUIDV7" "UUID_V7" "UUID_V4") (toUpper (declaration "DefaultExpression")))
		(list (quote sql_type_expr)
			(list (if (has? '("NEWID" "UUID_V4") (toUpper (declaration "DefaultExpression"))) (quote uuid_v4) (quote uuid_v7)))
			"UNIQUEIDENTIFIER" nil)
		(equal?? (declaration "DefaultExpression") "CURRENT_TIMESTAMP")
		(list (quote sql_type_expr) (list (quote now)) "DATETIME" nil)
		(and (nil? (declaration "Default")) (not (declaration "Null")))
		(error "nonnullable column has no usable DEFAULT: " (declaration "Field"))
		(begin
			(define type (toUpper (declaration "RawType")))
			(define dimensions (coalesceNil (declaration "Dimensions") '()))
			(define collation (declaration "Collation"))
			/* Native Type is display text and Collation is a plain name. Build
			the same declared datatype and coercibility pair as column binding. */
			(list (quote sql_type_expr) (declaration "Default")
				(if (empty_list? dimensions) type (list type dimensions))
				(if (and (sql_text_type? type) (string? collation) (not (equal? collation "")))
					(list collation 2) nil))))))
(define tsql_assignment_value (lambda (expression declaration database policy planning tx)
	(if (equal?? expression (quote sql_default)) (tsql_default_assignment_expr declaration database policy planning tx) expression)))
/* Only declaration consumers register these O(1) guards. The common cache
recompiles a variant after DDL, without a schema epoch or runtime SQL type tag. */
(define tsql_declaration_contract (lambda (column)
	(list (column "RawType") (coalesceNil (column "Dimensions") '())
		(column "Null") (coalesceNil (column "Collation") "")
		(column "Default") (coalesceNil (column "DefaultExpression") "") (column "DefaultDeferred") (column "Extra") (column "Computed"))))
(define tsql_declaration_matches? (lambda (column contract)
	(if (nil? column) false (expression_equal? (tsql_declaration_contract column) contract))))
(define tsql_column_description (lambda (database name column ignore_case planning_session tx) (begin
	(if (local_relation_reference? name)
		(find (local_relation_columns name) (lambda (declaration)
			((if ignore_case equal?? equal?) (declaration "Field") column)) nil)
		(begin (define handle (table database false name false tx))
			(if (nil? handle) nil (tsql_effective_column (show handle "column" column ignore_case tx) planning_session)))))))
(define tsql_target_column_declaration (lambda (database name column planning_session tx) (begin
	(define declaration (tsql_column_description database name column true planning_session tx))
	(if (nil? declaration) nil (begin
		(if (declaration "Computed") (error "computed column is not writable: " (declaration "Field")) true)
		(if (and (not (local_relation_reference? name)) (declaration "AliasId"))
			(tsql_guard_type_alias (declaration "DeclaredType") (declaration "AliasId") planning_session) true)
		(if (local_relation_reference? name) true (planner_record_guard_condition
			(list (quote tsql_declaration_matches?)
				(list (quote tsql_column_description) database name (declaration "Field") false nil (quote tx))
				(list (quote quote) (tsql_declaration_contract declaration))) planning_session))
		declaration)))))
/* An omitted target list supplies only writable ordinary columns. Generated
identity/version declarations are frontend policy; their native storage and
allocation stay unchanged. Other dialects keep their own implicit-list rules. */
(define tsql_insert_target_columns (lambda (database name tx)
	(map (filter (if (local_relation_reference? name) (local_relation_columns name) (get_schema database name tx)) (lambda (column)
		(and (not (internal_column_name? (column "Field")))
			(not (equal? (column "Extra") "auto_increment"))
			(not (column "Computed"))
			(not (equal?? (column "RawType") "ROWVERSION")))))
		(lambda (column) (column "Field")))))
(define tsql_insert_declarations (lambda (database table_name columns planning_session tx) (begin	/* Resolve each target exactly once. Duplicate targets can overwrite an
	already validated identity/default value in the native column vector. */
	/* An empty vector has no column lookup that could reject an absent table. */
	(if (empty_list? columns)
		(if (if (local_relation_reference? table_name) true (table database false table_name false tx)) true
			(error "INSERT target table does not exist: " table_name)) true)
	(if (equal? (count columns) (count (merge_unique (list (map columns toLower)))))
		true (error "duplicate INSERT column"))
	(map columns (lambda (name) (begin
		(define declaration (tsql_target_column_declaration database table_name name planning_session tx))
		(if (nil? declaration) (error "unknown INSERT column: " name) declaration)))))))
/* Row versions consume a nontransactional durable database sequence. The
SQL spelling and binary representation belong to this frontend; native
reservation owns concurrency, persistence and rollback independence. */
(define tsql_next_rowversion (lambda (database tx)
	(tsql_integer_to_binary (car (sequence_reserve database "tsql.rowversion" 1 tx)) 8 8 true)))
(define tsql_current_rowversion (lambda (database tx)
	(tsql_integer_to_binary (sequence_current database "tsql.rowversion" tx) 8 8 true)))
(define tsql_rowversion_expr (lambda (database)
	(list (quote sql_type_expr) (list (quote tsql_next_rowversion) database (quote tx)) "ROWVERSION" nil)))
/* Bind both presence and absence from one native declaration publication.
A later ADD/DROP/ALTER default or generated column invalidates the write plan. */
(define tsql_write_declarations (lambda (database name planning_session tx) (begin
	(if (local_relation_reference? name) (local_relation_columns name) (begin
		(define declarations (sql_relation_local_declarations database name tx))
		(planner_record_guard_condition
			(list (quote sql_relation_local_declarations_match?) database name (quote tx)
				(list (quote quote) declarations)) planning_session)
		(declarations "columns"))))))
(define tsql_insert_deferred_columns (lambda (database name columns policy planning_session tx)
	(map (filter (tsql_write_declarations database name planning_session tx) (lambda (column)
		(and (not (column "Computed"))
			(or (column "DefaultDeferred") (equal?? (column "RawType") "ROWVERSION"))
			(not (has? columns (column "Field"))))))
		(lambda (column) (begin
			(define declared (tsql_target_column_declaration database name (column "Field") planning_session tx))
			(list (declared "Field") (sql_type_formula '()
				(if (equal?? (declared "RawType") "ROWVERSION") (tsql_rowversion_expr database)
					(tsql_assignment_expr (tsql_default_assignment_expr declared database policy planning_session tx) declared)))))))))
/* English legacy calendar text uses the existing Unix date primitives. The
bounded month directory is frontend syntax, not a second temporal carrier. */
(define tsql_calendar_month_names '("Jan" "Feb" "Mar" "Apr" "May" "Jun" "Jul" "Aug" "Sep" "Oct" "Nov" "Dec"))
(define tsql_calendar_month_output (lambda (value language) (if (nil? value) nil (begin
	(define hour (extract_date value "HOUR"))
	(define clock (if (equal? (mod hour 12) 0) 12 (mod hour 12)))
	(define day (extract_date value "DAY"))
	(concat (nth (tsql_language_months language true) (- (extract_date value "MONTH") 1)) " "
		(if (< day 10) " " "") day " " (format_date value "%Y") " "
		(if (< clock 10) " " "") clock ":" (format_date value "%i") (if (< hour 12) "AM" "PM"))))))
/* Numeric text width and Boolean spelling follow the declared input type.
Only the small integer family uses the ASCII overflow marker; BIGINT and
national text retain their conversion error. Runtime carriers select no type. */
(define tsql_cast_numeric_text (close_procedure ((lambda (tsql_raise_error tsql_cast_fixed_text tsql_cast_text)
	(lambda (value length national fixed asterisk boolean)
		(if (nil? value) nil (begin
			(define text (string (if boolean (if value 1 0) value)))
			(define text (if (and (not (equal? length -1)) (> (strlen text) length))
				(if asterisk "*" (tsql_raise_error 8115 "numeric character conversion overflow" 1)) text))
			((if fixed tsql_cast_fixed_text tsql_cast_text) text length national "windows-1252"))))) tsql_raise_error tsql_cast_fixed_text tsql_cast_text)))
/* Default numeric character conversion belongs to the declaration compiler.
Decimal scale survives integral carriers; FLOAT/REAL style zero uses at most
six significant digits, with a three-digit scientific exponent. */
(define tsql_decimal_text (close_procedure (lambda (value scale)
	(if (nil? value) nil (begin
		(define text (if (equal? value 0) "0" (string value)))
		/* Preserve exact integral text, including positive-scale padding. */
		(if (equal? scale 0) (if (regexp_test text "[eE]") (format_fixed value 0) text)
			(regexp_test text "^-?[0-9]+$") (concat text "." (string_repeat "0" scale))
			(format_fixed value scale)))))))
(define tsql_float_text (close_procedure (lambda (value)
	(if (nil? value) nil (equal? value 0) "0" (begin
		(define parts (split (format_general value 6) "e"))
		(if (equal? (count parts) 1) (car parts) (begin
			(define exponent (cadr parts))
			(define digits (substr exponent 1))
			(concat (car parts) "e" (substr exponent 0 1)
				(string_repeat "0" (max 0 (- 3 (strlen digits)))) digits))))))))
(define tsql_text_compiler (lambda (length national fixed) (lambda (infos) (begin
	(define info (car infos))
	(define source (sql_info_type info))
	(define formula (if (equal? source "SQL_VARIANT")
		(list tsql_variant_text (sql_info_formula info))
		(tsql_temporal_type? source)
		(if (has? '("DATETIME" "SMALLDATETIME") source)
			(list (quote tsql_calendar_month_output) (tsql_calendar_difference_formula info) (tsql_language_formula))
			(if (equal? source "DATE") (list format_date (tsql_calendar_difference_formula info) "%Y-%m-%d")
				(tsql_temporal_style_expr info 121 false)))
		(has? '("DECIMAL" "NUMERIC") source)
		(list tsql_decimal_text (sql_info_formula info) (cadr (tsql_decl_dimensions source (sql_info_dimensions info))))
		(has? '("FLOAT" "REAL") source) (list tsql_float_text (sql_info_formula info))
		(sql_info_formula info)))
	(define converted (if (has? '("MONEY" "SMALLMONEY") source)
		(list tsql_cast_money_text formula 0 length national fixed)
		(sql_numeric_type? source)
		(list tsql_cast_numeric_text formula length national fixed
			(and (not national) (has? '("TINYINT" "SMALLINT" "INT" "INTEGER") source))
			(has? '("BIT" "BOOL" "BOOLEAN") source))
		(list (if fixed tsql_cast_fixed_text tsql_cast_text) formula length national
			(if national "windows-1252" (tsql_character_encoding info)))))
	(sql_info converted
		(list (if (and national fixed) "NCHAR" national "NVARCHAR" fixed "CHAR" "VARCHAR") (list length))
		(sql_info_collation info))))))
(define tsql_cast_text_expr (lambda (value dimensions national fixed) (begin
	(define length (if (empty_list? dimensions) 30 (car dimensions)))
	(list (quote sql_compile_expr) (list (quote tsql_text_compiler) length national fixed) value))))
/* The default character contract is NCHAR(1), without a supplementary-character
collation. SQL conversion and range decisions stay here; the runtime operator
only encodes Unicode scalars. Isolated surrogate code units are unsupported. */
(define tsql_nchar (lambda (value round_input) (if (nil? value) nil (begin
	(define code (tsql_cast_integer (if round_input (round value) value) -2147483648 2147483647))
	(if (or (< code 0) (> code 65535)) nil (unicode_char code))))))
(define tsql_compile_nchar (lambda (infos) (begin
	(define info (car infos))
	(define type (tsql_addition_type (sql_info_type info)))
	/* Implicit INT conversion is a SQL decision. Money rounds; other numeric
	inputs truncate. Temporal carriers never become integer codes. */
	(if (has? '("NULL" "BIT" "TINYINT" "SMALLINT" "INT" "BIGINT" "REAL" "FLOAT"
		"DECIMAL" "MONEY" "SMALLMONEY" "CHAR" "VARCHAR" "NCHAR" "NVARCHAR") type) true
		(error "NCHAR requires an implicitly convertible integer expression"))
	(sql_info (list tsql_nchar (sql_info_formula info) (has? '("MONEY" "SMALLMONEY") type)) "NCHAR" nil))))
(define tsql_nchar_expr (lambda (value)
	(list (quote sql_compile_expr) (quote tsql_compile_nchar) value)))
(define tsql_export_datetime_seconds (lambda (value) (if (nil? value) nil (begin
	(define text (toUpper (sql_trim (string value))))
	(if (regexp_test text "^[0-9]{2}[.-][0-9]{2}[.-][0-9]{2}( [0-9]{2}:[0-9]{2}:[0-9]{2}( (AM|PM))?)?$")
		true (tsql_raise_error 241 "invalid day/month/year datetime text" 1))
	(define year (substr text 6 2))
	(define date (concat (if (< (tsql_parse_integer year) 50) "20" "19") year "-" (substr text 3 2) "-" (substr text 0 2)))
	(define clock (if (equal? (strlen text) 8) "00:00:00" (substr text 9 8)))
	(define clock (if (> (strlen text) 17) (begin
		(define hour (tsql_parse_integer (substr clock 0 2)))
		(if (> hour 12) (tsql_raise_error 241 "invalid twelve-hour datetime text" 1) true)
		(define hour (+ (mod hour 12) (if (equal? (substr text 18 2) "PM") 12 0)))
		(concat (if (< hour 10) "0" "") hour (substr clock 2 6))) clock))
	(define parsed (str_to_date (concat date " " clock) "%Y-%m-%d %H:%i:%s"))
	(if (nil? parsed) (tsql_raise_error 241 "invalid datetime value" 1) (unix_timestamp parsed))))))
/* Persisted view IR retains this pattern-based factory contract. New CONVERT
syntax selects a source-dependent style through the separate factory below. */
(define tsql_styled_text_compiler (lambda (pattern type dimensions) (lambda (infos) (begin
	(define input (car infos))
	(define formula (list format_date (tsql_calendar_difference_formula input) pattern))
	(sql_info (list (if (has? '("CHAR" "NCHAR") type) tsql_cast_fixed_text tsql_cast_text)
		formula (if (empty_list? dimensions) 30 (car dimensions)) (has? '("NVARCHAR" "NCHAR") type) "windows-1252") type nil)))))
(define tsql_convert_text_compiler (lambda (style type dimensions) (lambda (infos) (begin
	(define input (car infos))
	(define source (sql_info_type input))
	(define length (if (empty_list? dimensions) 30 (car dimensions)))
	(define national (has? '("NVARCHAR" "NCHAR") type))
	(define fixed (has? '("CHAR" "NCHAR") type))
	(if (has? '("MONEY" "SMALLMONEY") source)
		(sql_info (list tsql_cast_money_text (sql_info_formula input) style
			length national fixed) type nil)
		(tsql_temporal_type? source) (if (equal? style 100)
			(if (has? '("DATETIME" "SMALLDATETIME") source)
				((tsql_text_compiler length national fixed) infos)
				(error "month-name output style requires a legacy calendar declaration"))
			(and (equal? source "TIME") (has? '(102 103 105 110 111) style))
			(error "date-only output style requires a calendar date")
			(has? '(21 121 126) style) (sql_info
				(list (if fixed tsql_cast_fixed_text tsql_cast_text)
					(tsql_temporal_style_expr input style (equal? style 126)) length national "windows-1252") type nil)
			((tsql_styled_text_compiler (tsql_temporal_style_pattern style true) type dimensions) infos))
		(tsql_binary_type? source) (begin
			(if (has? '(1 2) style) true (error "unsupported binary conversion style"))
			(sql_info (list (if fixed tsql_cast_fixed_text tsql_cast_text)
				(list (quote tsql_binary_to_hex) (sql_info_formula input) (equal? style 1) length)
				length national "windows-1252") type nil))
		((tsql_text_compiler length national fixed) infos))))))
(define tsql_convert_expr (lambda (value type dimensions style) (begin
	(define builtin (tsql_builtin_type_declaration type))
	(tsql_decl_dimensions type dimensions)
	(define dimensions (if builtin (builtin "dimensions") dimensions))
	(define type (if builtin (builtin "base_type") type))
	(match style
		(symbol sql_null_style) (tsql_cast_expr (list (quote sql_type_expr) nil "NULL" nil) type dimensions)
		_ (if (or (nil? style) (equal? style 0)) (tsql_cast_expr value type dimensions)
			(tsql_temporal_type? type) (tsql_temporal_cast_expr value type style dimensions)
			(has? '("BINARY" "VARBINARY") type)
			(tsql_cast_binary_expr value dimensions (equal? type "BINARY") style)
			(has? '("VARCHAR" "NVARCHAR" "CHAR" "NCHAR") type)
			(list (quote sql_compile_expr) (list (quote tsql_convert_text_compiler) style type (list (quote quote) dimensions)) value)
			(error "unsupported CONVERT style for " type))))))
/* TRY converts only a validated conversion recipe. The operand is a strict
argument outside the catch closure, so its own errors are never conversion
failures. Unsupported fractional, high-scale and bounded-input contracts
remain explicit errors. Only deliberate frontend conversion errors are caught;
neither SQL declarations nor error classes are guessed from runtime carriers
or error-message text. */
(define tsql_try_cast_input (lambda (value national) (if (nil? value) true (begin
	(define limit (if national 4000 8000))
	(define units (if (and national (> (strlen value) limit))
		(reduce (split value "") (lambda (units character)
			(+ units (if (equal? (strlen character) 4) 2 1))) 0) (strlen value)))
	(if (> units limit) (error "TRY_CAST input length exceeds the supported limit") true)))))
/* Reuse the frontend's structured SQL errors. Unexpected helper failures,
cancellation and unsupported operations are never successful NULL casts. */
(define tsql_try_conversion_error (lambda (cause) (match cause
	'("tsql-error" number _message _state)
	(if (has? '(245 8114 8115 241 242) number) nil (error cause))
	_ (error cause))))
(define tsql_try_conversion_compiler (lambda (type dimensions style bounded_input) (lambda (infos) (begin
	(define input (car infos))
	(define source (sql_info_type input))
	(define numeric (sql_numeric_type? source))
	(define text (has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR" "TEXT" "NTEXT") source))
	(define absent (equal? source "NULL"))
	(define binary_text (and (tsql_binary_type? source)
		(or (has? '(1 2) style) (equal? style (quote sql_null_style)))))
	(if (and (sql_numeric_type? type) (not (or numeric text absent)))
		(error "conversion from " source " to " type " is not implemented") true)
	(if (and (has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR") type)
		(not (or numeric text absent (tsql_temporal_type? source) binary_text)))
		(error "conversion from " source " to " type " is not implemented") true)
	(if (and (not (nil? style)) (not (equal? style 0)) (not (equal? style (quote sql_null_style)))
		(has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR") type)
		(not (or absent (tsql_temporal_type? source) binary_text)))
		(error "styled text conversion requires a declared temporal input") true)
	/* Feed the existing compiler a declared local, not an executable SQL type
	wrapper. Only its expr slot enters the emitted invocation lambda. */
	(define operand (quote try_value))
	(define converted (sql_expr_info '() (tsql_convert_expr
		(list (quote sql_type_expr) operand source (sql_info_collation input)) type dimensions style)))
	/* Preserve the existing unsupported high-scale CAST contract. In
	particular its native precision error must not silently become NULL. */
	(define unsupported_scale (and (has? '("DECIMAL" "NUMERIC") type)
		(> (cadr (tsql_decl_dimensions type dimensions)) 15)))
	(define body (if unsupported_scale (sql_info_formula converted)
		(list (quote try) (list (quote lambda) '() (sql_info_formula converted))
			(list (quote lambda) (list (quote conversion_error))
				(list (quote tsql_try_conversion_error) (quote conversion_error))))))
	(define checks (if (and bounded_input text (not (equal? style (quote sql_null_style))))
		(list (list (quote tsql_try_cast_input) operand (has? '("NCHAR" "NVARCHAR" "NTEXT") source))) '()))
	(sql_info (list (list (quote lambda) (list operand)
		(cons (quote begin) (append checks body))) (sql_info_formula input))
		(sql_info_datatype converted)
		(if (and text (has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR") type))
			(sql_info_collation input) (sql_info_collation converted)))))))
(define tsql_try_convert_expr (lambda (value type dimensions style bounded_input) (begin
	(tsql_decl_dimensions type dimensions)
	(list (quote sql_compile_expr) (list (quote tsql_try_conversion_compiler)
		type (list (quote quote) dimensions) (list (quote quote) style) bounded_input) value))))
/* Aggregate contracts come from SQL syntax and the declared input type.
The native reducers do not determine the frontend's SQL result contract. */
(define tsql_aggregate_type (lambda (name input_type) (match name
	"COUNT" "INT"
	"ROW_NUMBER" "BIGINT" "RANK" "BIGINT" "DENSE_RANK" "BIGINT"
	"SUM" (tsql_numeric_aggregate_type input_type)
	"AVG" (tsql_numeric_aggregate_type input_type)
	"MIN" input_type "MAX" input_type "LAG" input_type "LEAD" input_type
	_ "any")))
(define tsql_numeric_aggregate_type (lambda (input_type) (match (tsql_addition_type input_type)
	"TINYINT" "INT" "SMALLINT" "INT" "INT" "INT"
	"BIGINT" "BIGINT" "REAL" "FLOAT" "FLOAT" "FLOAT"
	"DECIMAL" "DECIMAL" "MONEY" "MONEY" "SMALLMONEY" "MONEY"
	"NULL" "INT"
	_ (error "numeric aggregate requires a declared numeric input"))))
(define tsql_aggregate_result (lambda (name formula type collation)
	(sql_info (if (and (equal? name "AVG") (has? '("INT" "BIGINT") type))
		(sql_type_formula '() (tsql_cast_expr formula type '())) formula) type
		(if (sql_text_type? type) collation nil))))
(define tsql_aggregate_compiler (lambda (name) (lambda (infos) (begin
	/* Ordinary declared reducers already retain their operand contract.
	AVG/custom reducers, and persisted older expressions, carry a separate
	type witness. Neither form derives a SQL type from an SCM value tag. */
	(define input (match infos '(aggregate_info) aggregate_info
		'(aggregate_info input_info) input_info
		_ (error "aggregate compiler requires its declared input contract")))
	(tsql_aggregate_result name (sql_info_formula (car infos))
		(tsql_aggregate_type name (sql_info_type input))
		(sql_info_collation input))))))
(define tsql_aggregate_expr (lambda (name formula value) (begin
	(define retained (match formula
		((symbol aggregate) _operand reducer _neutral)
		(sql_aggregate_preserves_operand_contract? reducer)
		_ false))
	/* A second executable copy of the operand used only for typing would
	expand every relational function twice before ordinary name binding. */
	(cons (quote sql_compile_expr) (cons (list (quote tsql_aggregate_compiler) name)
		(if retained (list formula) (list formula value)))))))
/* Preserve the native window AST for logical stage construction. Its result
contract is compiled from resolved input triples, never from an empty scope. */
(define tsql_window_compiler (lambda (name has_input) (lambda (infos) (begin
	(define input_type (if (not has_input) "any" (sql_info_type (cadr infos))))
	(define type (tsql_aggregate_type name input_type))
	(define formula (match (sql_info_formula (car infos))
		((symbol window_func) function _arguments specification)
		(list (quote window_func) function (map (cdr infos) sql_info_formula) specification)
		_ (sql_info_formula (car infos))))
	(tsql_aggregate_result name formula type
		(if (not has_input) nil (sql_info_collation (cadr infos))))))))
(define tsql_window_expr (lambda (name args over)
	(cons (quote sql_compile_expr)
		(cons (list (quote tsql_window_compiler) name (not (empty_list? args)))
			(cons (list (quote window_func) name args over) args)))))
/* Calendar units are syntax, resolved once by the frontend. DATEDIFF counts
crossed boundaries rather than complete elapsed intervals. Native date/Unix
operators supply the values; the result SQL contract is always explicit INT. */
(define tsql_calendar_unit (lambda (unit) (match (toLower unit)
	"yyyy" "year" "yy" "year" "year" "year"
	"q" "quarter" "qq" "quarter" "quarter" "quarter"
	"m" "month" "mm" "month" "month" "month"
	"y" "dayofyear" "dy" "dayofyear" "dayofyear" "dayofyear"
	"d" "day" "dd" "day" "day" "day"
	"w" "weekday" "dw" "weekday" "weekday" "weekday"
	"wk" "week" "ww" "week" "week" "week"
	"isowk" "iso_week" "isoww" "iso_week" "iso_week" "iso_week"
	"hh" "hour" "hour" "hour"
	"n" "minute" "mi" "minute" "minute" "minute"
	"s" "second" "ss" "second" "second" "second"
	_ (error "unsupported calendar unit: " unit))))
(define tsql_calendar_month_index (lambda (value)
	(if (nil? value) nil (+ (* (extract_date value "YEAR") 12) (- (extract_date value "MONTH") 1)))))
(define tsql_calendar_quarter_index (lambda (value)
	(if (nil? value) nil (intdiv (tsql_calendar_month_index value) 3))))
/* Conversion is selected from the SQL declaration, never the runtime carrier.
Numeric dates count days from 1900; native DATE/DATETIME values are Unix values. */
/* Character conversion consumes the executing session, never parser state.
The compiler emits these reads as ordinary scalar AST dependencies. */
(define tsql_dateformat_formula (lambda ()
	(list (quote coalesceNil) (list (quote session) "__tsql_dateformat") "mdy")))
(define tsql_language_formula (lambda ()
	(list (quote coalesceNil) (list (quote session) "__tsql_language") "us_english")))
(define tsql_calendar_text_formula (lambda (value legacy)
	(list tsql_calendar_text value (tsql_dateformat_formula) (tsql_language_formula) legacy)))
(define tsql_calendar_days (close_procedure ((lambda (tsql_raise_error)
	(lambda (value) (if (nil? value) nil (begin
		(if (and (>= value -53690) (< value 2958464)) true (tsql_raise_error 242 "numeric datetime conversion overflow" 1))
		(define days (intdiv value 1))
		(define seconds (intdiv (* (- value days) 86400) 1))
		(date_add (date_add (parse_date "1900-01-01") days "DAY") seconds "SECOND"))))) tsql_raise_error)))
/* The common string family also includes binary byte carriers. Calendar text
is a T-SQL character declaration; bytes cannot silently select a date parser. */
(define tsql_calendar_text_type? (lambda (type)
	(has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR" "TEXT" "NTEXT") type)))
(define tsql_calendar_formula (lambda (info) (begin
	(define formula (sql_info_formula info))
	(define type (sql_info_type info))
	(if (equal? type "NULL") nil
		(has? '("DATE" "DATETIME" "DATETIME2" "SMALLDATETIME") type) (list parse_date formula)
		(sql_numeric_type? type) (list tsql_calendar_days formula)
		(tsql_calendar_text_type? type) (tsql_calendar_text_formula formula true)
		(error "calendar input requires a declared date, number or text type; got " type)))))
(define tsql_calendar_day_of_year (lambda (value)
	(if (nil? value) nil (+ 1 (timestampdiff "DAY" (format_date value "%Y-01-01") value)))))
/* Session input remains visible in the expression AST, so the common planner
owns its domain/cache dependency. Only week and weekday consume DATEFIRST. */
(define tsql_datefirst_expr (lambda ()
	(list (quote sql_type_expr) (list (quote coalesceNil)
		(list (quote session) "__tsql_datefirst") 7) "INT" nil)))
(define tsql_datefirst_value (lambda (value)
	(if (nil? value) (error "DATEFIRST requires an integer from 1 through 7")
		(tsql_cast_integer value 1 7))))
(define tsql_calendar_weekday (lambda (value first)
	(if (nil? value) nil (+ 1 (mod (+ (extract_date value "WEEKDAY") 8 (- 0 first)) 7)))))
(define tsql_calendar_week (lambda (value first) (if (nil? value) nil (begin
	/* January first belongs to week one; later weeks begin on DATEFIRST. */
	(define january (parse_date (format_date value "%Y-01-01")))
	(+ 1 (intdiv (+ (- (tsql_calendar_day_of_year value) 1)
		(- (tsql_calendar_weekday january first) 1)) 7))))))
(define tsql_calendar_iso_week (lambda (value) (extract_date value "WEEK")))
(define tsql_calendar_millisecond_index (lambda (value) (if (nil? value) nil (begin
	/* Native parts use floor seconds and nonnegative nanoseconds, including
	before 1970. Keep the boundary index exact across the SQL calendar range. */
	(define parts (date_parts value))
	(+ (* (car parts) 1000) (intdiv (cadr parts) 1000000))))))
(define tsql_boundary_calculator (lambda (unit) (match unit
	"year" (sql_builtins "YEAR") "month" tsql_calendar_month_index "quarter" tsql_calendar_quarter_index
	"millisecond" tsql_calendar_millisecond_index
	_ (begin
		(define seconds (match unit "hour" 3600 "minute" 60 "second" 1 "week" 604800 _ 86400))
		(define offset (if (equal? unit "week") 345600 0))
		/* Floor, including before the Unix epoch. Week boundaries are Sundays,
		independent of any session's first-day-of-week setting. */
		(lambda (value) (if (nil? value) nil
			(floor (/ (+ (unix_timestamp value) offset) seconds))))))))
/* Use native date arithmetic, correcting only this dialect's month-end rule.
No row-level datatype dispatch or alternate temporal representation is needed. */
(define tsql_calendar_shift_months (lambda (value amount) (if (or (nil? value) (nil? amount)) nil (begin
	(define source_day (extract_date value "DAY"))
	/* Moving from day one avoids native month overflow and retains the clock
	without formatting/parsing intermediate strings. */
	(define anchor (date_add value (intdiv (- 1 source_day) 1) "DAY"))
	(define target (date_add anchor amount "MONTH"))
	(define last (date_add (date_add target 1 "MONTH") -1 "DAY"))
	(define day (min source_day (extract_date last "DAY")))
	(date_add target (intdiv (- day 1) 1) "DAY")))))
(define tsql_calendar_shift_seconds (lambda (value amount) (if (or (nil? value) (nil? amount)) nil (begin
	/* Split large intervals so the native seconds operator cannot overflow
	its duration carrier before the frontend validates the calendar range. */
	(define days (intdiv amount 86400))
	(date_add (date_add value days "DAY") (intdiv (- amount (* days 86400)) 1) "SECOND")))))
(define tsql_calendar_range (close_procedure ((lambda (tsql_raise_error)
	(lambda (value minimum)
		(if (nil? value) nil (if (and (>= (unix_timestamp value) minimum) (<= (unix_timestamp value) 253402300799))
			value (tsql_raise_error 242 "calendar result overflow" 1))))) tsql_raise_error)))
/* Temporal contracts are compiled from SQL declarations. Calendar scalars
preserve their fractional clock; TIME retains its anchored calendar convention. */
(define tsql_temporal_types '("DATETIME2" "DATETIME" "SMALLDATETIME" "DATE" "TIME"))
(define tsql_temporal_type? (lambda (type) (has? tsql_temporal_types type)))
(define tsql_temporal_numeric_target? (lambda (type) (has? '("DATETIME" "SMALLDATETIME") type)))
(define tsql_calendar_time_only (lambda (value) (if (nil? value) nil
	(date_from_parts (+ -2208988800 (mod (+ (mod (unix_timestamp value) 86400) 86400) 86400))
		(cadr (date_parts value)) 0))))
(define tsql_calendar_small_datetime (close_procedure ((lambda (tsql_raise_error)
	(lambda (value) (if (nil? value) nil (begin
		(define parts (date_parts value))
		(define seconds (car parts))
		(define minute (- seconds (mod (+ (mod seconds 60) 60) 60)))
		(define clock (- seconds minute))
		(define rounded (date_from_parts (+ minute (if (or (> clock 29)
			(and (equal? clock 29) (>= (cadr parts) 999000000))) 60 0)) 0 0))
		(if (and (>= (unix_timestamp rounded) -2208988800) (<= (unix_timestamp rounded) 3453321540))
			rounded (tsql_raise_error 242 "small datetime conversion overflow" 1)))))) tsql_raise_error)))
(define tsql_temporal_style_pattern (lambda (style output) (match style
	1 (if output "%m/%d/%y" "%m/%d/%Y") 101 "%m/%d/%Y"
	102 "%Y.%m.%d" 103 "%d/%m/%Y" 105 "%d-%m-%Y"
	110 "%m-%d-%Y" 111 "%Y/%m/%d"
	4 (if output "%d.%m.%y" "%d.%m.%Y") 104 "%d.%m.%Y"
	12 (if output "%y%m%d" "%Y%m%d") 112 "%Y%m%d"
	8 "%H:%i:%s" 108 "%H:%i:%s"
	126 "%Y-%m-%dT%H:%i:%s"
	23 "%Y-%m-%d" 20 "%Y-%m-%d %H:%i:%s" 120 "%Y-%m-%d %H:%i:%s"
	21 (if output "%Y-%m-%d %H:%i:%s.000" "%Y-%m-%d %H:%i:%s")
	121 (if output "%Y-%m-%d %H:%i:%s.000" "%Y-%m-%d %H:%i:%s")
	_ (error "unsupported CONVERT style: " style))))
/* Numeric calendar formats have no language dependency. Keep their parser
closed independently of the language-sensitive style-100 profile. */
(define tsql_numeric_calendar_fields (close_procedure (lambda (text)
	/* Only a complete numeric date may discard separator padding. A clock,
	ISO T, zone, named month or split digit keeps its original text. */
	(if (regexp_test text "^ *[0-9]{1,4} *[-/.] *[0-9]{1,4} *[-/.] *[0-9]{1,4} *$")
		(replace text " " "") text))))
(define tsql_temporal_numeric_style_text (close_procedure ((lambda (style_pattern trim parse_integer raise_error normalize_fields)
	(lambda (value style) (if (nil? value) nil (begin
		(define text (trim value))
		(define pattern (style_pattern style false))
		/* The native format parser requires padded numeric fields. Normalize
		only separated calendar inputs whose declared year width already
		matches; keep invalid structure and dates for the same strict parser. */
		(define separated (regexp_test pattern "^%[Ymd][-/.]%[Ymd][-/.]%[Ymd]$"))
		(define text (if separated (normalize_fields text) text))
		(define year_first (equal? (substr pattern 0 2) "%Y"))
		(define calendar_shape (if year_first "^[0-9]{4}[-/.][0-9]{1,2}[-/.][0-9]{1,2}$"
			(equal? style 1) "^[0-9]{1,2}[-/.][0-9]{1,2}[-/.][0-9]{2}$"
			"^[0-9]{1,2}[-/.][0-9]{1,2}[-/.][0-9]{4}$"))
		(define text (if (and separated (regexp_test text calendar_shape))
			(regexp_replace
				(regexp_replace text "(^|[-/.])([0-9])([-/.]|$)" "${1}0${2}${3}")
				"(^|[-/.])([0-9])([-/.]|$)" "${1}0${2}${3}") text))
		(define text (if (has? '(1 12) style) (begin
			(define offset (if (equal? style 12) 0 6))
			(if (< (strlen text) (+ offset 2)) (raise_error 241 "invalid short calendar year" 1) true)
			(define year (substr text offset 2))
			(if (regexp_test year "^[0-9]{2}$") true (raise_error 241 "invalid short calendar year" 1))
			(concat (substr text 0 offset) (if (< (parse_integer year) 50) "20" "19")
				year (substr text (+ offset 2)))) text))
		(define value (if (has? '(21 121) style) (parse_date text) (str_to_date text pattern)))
		(if (nil? value) (raise_error 241 "invalid styled calendar input" 1) value)))))
	tsql_temporal_style_pattern sql_trim tsql_parse_integer tsql_raise_error tsql_numeric_calendar_fields)))

/* SQL precision is selected from the declaration; the generic calendar scalar
retains nanoseconds independently of this frontend's rounding policy. */
(define tsql_temporal_fraction_text (lambda (nanos digits) (begin
	(define padded (concat (string_repeat "0" (- 9 (strlen (string nanos)))) nanos))
	(if (equal? digits 0) "" (concat "." (substr padded 0 digits))))))
(define tsql_temporal_clock_text (close_procedure ((lambda (tsql_temporal_fraction_text)
	(lambda (value scale)
		(concat (format_date value "%H:%i:%s") (tsql_temporal_fraction_text (cadr (date_parts value)) scale)))) tsql_temporal_fraction_text)))
/* These factories consume SQL contracts at compile time. Only scalar units,
format patterns and range bounds reach the generated value expressions. */
(define tsql_temporal_factor (lambda (type dimensions)
	(if (equal? type "DATETIME") 300
		(nth '(1 10 100 1000 10000 100000 1000000 10000000)
			(if (empty_list? dimensions) 7 (car dimensions))))))
(define tsql_temporal_quantize (close_procedure ((lambda (tsql_calendar_range)
	(lambda (value factor minimum) (if (nil? value) nil (begin
		(define parts (date_parts value))
		(define ticks (intdiv (+ (* (cadr parts) factor) 500000000) 1000000000))
		(define carry (intdiv ticks factor))
		(define fraction (intdiv (+ (* (mod ticks factor) 1000000000) (intdiv factor 2)) factor))
		(tsql_calendar_range (date_from_parts (+ (car parts) carry) fraction (nth parts 2)) minimum))))) tsql_calendar_range)))
(define tsql_temporal_style_output (close_procedure ((lambda (tsql_temporal_fraction_text)
	(lambda (value pattern digits round_ms omit_zero) (if (nil? value) nil (begin
		(define nanos (cadr (date_parts value)))
		(define nanos (if round_ms (* 1000000 (intdiv (+ nanos 500000) 1000000)) nanos))
		(concat (format_date value pattern)
			(if (and omit_zero (equal? nanos 0)) "" (tsql_temporal_fraction_text nanos digits))))))) tsql_temporal_fraction_text)))
(define tsql_temporal_style_expr (lambda (info style omit_zero) (begin
	(define source (sql_info_type info))
	(define dimensions (sql_info_dimensions info))
	(define formula (tsql_calendar_difference_formula info))
	(define legacy (has? '("DATETIME" "SMALLDATETIME") source))
	(define formula (if (equal? source "DATETIME")
		(list tsql_temporal_quantize formula 300 -6847804800) formula))
	(define pattern (if (equal? source "DATE") "%Y-%m-%d" (equal? source "TIME") "%H:%i:%s"
		(equal? style 126) "%Y-%m-%dT%H:%i:%s" "%Y-%m-%d %H:%i:%s"))
	(define digits (if (equal? source "DATE") 0 legacy 3
		(if (empty_list? dimensions) 7 (car dimensions))))
	(list tsql_temporal_style_output formula pattern digits (equal? source "DATETIME") omit_zero))))
(define tsql_calendar_cast_text (close_procedure ((lambda (tsql_raise_error)
	(lambda (value) (if (nil? value) nil (begin
		(if (regexp_test value "[0-9]{2}:[0-9]{2}:[0-9]{2}\\.[0-9]{8}")
			(tsql_raise_error 241 "temporal text exceeds seven fractional digits" 1) true)
		value)))) tsql_raise_error)))
(define tsql_temporal_compiler (lambda (target style dimensions) (lambda (infos) (begin
	(tsql_decl_dimensions target dimensions)
	(define input (car infos))
	(define source (sql_info_type input))
	/* A shared Unix carrier never authorizes incompatible SQL contracts. */
	(if (or (and (equal? source "DATE") (equal? target "TIME"))
		(and (equal? source "TIME") (equal? target "DATE")))
		(error "incompatible temporal conversion from " source " to " target) true)
	(define input (if (and (sql_text_type? source) (not (equal? target "DATE")))
		(sql_info (list tsql_calendar_cast_text (sql_info_formula input)) source (sql_info_collation input)) input))
	(if (and (sql_numeric_type? source) (not (tsql_temporal_numeric_target? target)))
		(error "numeric input is not convertible to " target) true)
	(if (not (has? '(0 4 100) style))
		(tsql_temporal_style_pattern style false) true)
	(define formula (if (and (sql_text_type? source) (not (equal? style 0)))
		(if (equal? style 4) (list tsql_calendar_shift_seconds (parse_date "1970-01-01")
			(list tsql_export_datetime_seconds (sql_info_formula input)))
			(if (equal? style 100)
				(list tsql_calendar_text (sql_info_formula input) "mdy" (tsql_language_formula) true)
				(list tsql_temporal_numeric_style_text (sql_info_formula input) style)))
		(if (sql_text_type? source)
			(tsql_calendar_text_formula (sql_info_formula input) (has? '("DATETIME" "SMALLDATETIME") target))
			(tsql_calendar_difference_formula input))))
	(define converted (match target
		"DATE" (list tsql_calendar_range (list date_trunc_day formula) -62135596800)
		"TIME" (list tsql_calendar_time_only formula)
		"SMALLDATETIME" (list tsql_calendar_small_datetime formula)
		_ (list tsql_calendar_range formula (if (equal? target "DATETIME") -6847804800 -62135596800))))
	(define converted (if (has? '("DATETIME" "DATETIME2" "TIME") target)
		(list tsql_temporal_quantize converted (tsql_temporal_factor target dimensions)
			(if (equal? target "DATETIME") -6847804800 -62135596800)) converted))
	(sql_info converted (if (empty_list? dimensions) target (list target dimensions)) nil)))))
(define tsql_temporal_cast_expr (lambda (value target style dimensions)
	(list (quote sql_compile_expr)
		(list (quote tsql_temporal_compiler) target style (list (quote quote) dimensions)) value)))
(define tsql_dateadd_compiler (lambda (unit months seconds) (lambda (infos) (begin
	(define input (cadr infos))
	(define input_type (sql_info_type input))
	(define amount_info (car infos))
	(if (or (equal? (sql_info_type amount_info) "NULL") (sql_numeric_type? (sql_info_type amount_info))) true
		(error "DATEADD requires a declared numeric interval"))
	(if (and (equal? input_type "DATE") (has? '("hour" "minute" "second") unit))
		(error "clock interval is not supported for DATE") true)
	(if (and (equal? input_type "TIME") (not (has? '("hour" "minute" "second") unit)))
		(error "calendar interval is not supported for TIME") true)
	(define type (if (tsql_temporal_type? input_type) input_type "DATETIME"))
	(define minimum (if (equal? type "DATETIME") -6847804800 -62135596800))
	(define integer (list tsql_cast_integer (sql_info_formula amount_info) -2147483648 2147483647))
	(define formula (list (if months tsql_calendar_shift_months tsql_calendar_shift_seconds)
		(tsql_calendar_difference_formula input)
		(list (quote *) integer (coalesceNil months seconds))))
	(sql_info (match type
		"TIME" (list tsql_calendar_time_only formula)
		"SMALLDATETIME" (list tsql_calendar_small_datetime formula)
		_ (list tsql_calendar_range formula minimum))
		(if (and (equal? type input_type) (not (empty_list? (sql_info_dimensions input))))
			(sql_info_datatype input) type) nil)))))
(define tsql_dateadd_expr (lambda (unit amount value) (begin
	(define unit (tsql_calendar_unit unit))
	(if (equal? unit "iso_week") (error "DATEADD does not support ISO week") true)
	(define months (match unit "year" 12 "quarter" 3 "month" 1 _ nil))
	(define seconds (match unit "second" 1 "minute" 60 "hour" 3600 "week" 604800 _ 86400))
	(list (quote sql_compile_expr) (list (quote tsql_dateadd_compiler) unit months seconds) amount value))))
(define tsql_calendar_difference_formula (lambda (info)
	(if (equal? (sql_info_type info) "TIME") (tsql_calendar_text_formula (sql_info_formula info) true)
		(tsql_calendar_formula info))))
(define tsql_calendar_scalar_compiler (lambda (unit) (lambda (infos)
	(sql_info (list (match unit "year" (sql_builtins "YEAR") "month" (sql_builtins "MONTH") _ (sql_builtins "DAY"))
		(tsql_calendar_difference_formula (car infos))) "INT" nil))))
(define tsql_calendar_scalar_expr (lambda (unit value)
	(list (quote sql_compile_expr) (list (quote tsql_calendar_scalar_compiler) unit) value)))
(define tsql_calendar_unavailable_part (lambda (value)
	(if (nil? value) nil (error "calendar date part is not available for declared TIME"))))
(define tsql_calendar_part_formula (lambda (unit info)
	(if (equal? (sql_info_type info) "TIME")
		(if (has? '("hour" "minute" "second") unit) (tsql_calendar_difference_formula info)
			(list tsql_calendar_unavailable_part (sql_info_formula info)))
		(tsql_calendar_formula info))))
(define tsql_datediff_compiler (lambda (unit type) (begin
	(define boundary (tsql_boundary_calculator unit))
	(define minimum (if (equal? type "INT") -2147483648 (tsql_parse_integer "-9223372036854775808")))
	(define maximum (if (equal? type "INT") 2147483647 (tsql_parse_integer "9223372036854775807")))
	(lambda (infos) (begin
		/* DATEDIFF interprets character inputs as datetime2. Reuse its SQL
		precision/validation compiler; declared temporal inputs keep their own
		rounded values, including DATETIME's 300 ticks per second. */
		(define infos (map infos (lambda (info)
			(if (tsql_calendar_text_type? (sql_info_type info))
				((tsql_temporal_compiler "DATETIME2" 0 '()) (list info)) info))))
		(define difference (list (quote -)
			(list boundary (tsql_calendar_difference_formula (cadr infos)))
			(list boundary (tsql_calendar_difference_formula (car infos)))))
		(sql_info (list tsql_cast_integer difference minimum maximum) type nil))))))
(define tsql_datediff_expr (lambda (unit first last type) (begin
	/* Milliseconds are supported here only. Other calendar functions retain
	their own unit contracts rather than falling through a default day path. */
	(define unit (if (has? '("ms" "millisecond") (toLower unit)) "millisecond" (tsql_calendar_unit unit)))
	(if (equal? unit "iso_week") (error "DATEDIFF does not support ISO week") true)
	(list (quote sql_compile_expr) (list (quote tsql_datediff_compiler) unit type) first last))))
(define tsql_datepart_compiler (lambda (unit) (begin
	(define part (match unit
		"year" (sql_builtins "YEAR") "quarter" (sql_builtins "QUARTER") "month" (sql_builtins "MONTH")
		"day" (sql_builtins "DAY") "dayofyear" tsql_calendar_day_of_year "hour" (sql_builtins "HOUR")
		"minute" (sql_builtins "MINUTE") "second" (sql_builtins "SECOND")
		"weekday" tsql_calendar_weekday "week" tsql_calendar_week "iso_week" tsql_calendar_iso_week))
	(lambda (infos) (sql_info
		(merge (list (list part (tsql_calendar_part_formula unit (car infos)))
			(if (has? '("week" "weekday") unit)
				(list (list tsql_datefirst_value (sql_info_formula (cadr infos)))) '()))) "INT" nil)))))
(define tsql_datepart_expr (lambda (unit value) (begin
	(define unit (tsql_calendar_unit unit))
	(merge (list (list (quote sql_compile_expr) (list (quote tsql_datepart_compiler) unit) value)
		(if (has? '("week" "weekday") unit) (list (tsql_datefirst_expr)) '()))))))


/* SQL language profiles are bounded frontend data. Their scalar names and
orders are copied into sessions at login/SET, not looked up for every row. */
(define tsql_calendar_full_month_names '("January" "February" "March" "April" "May" "June" "July" "August" "September" "October" "November" "December"))
(define tsql_calendar_weekday_names '("Monday" "Tuesday" "Wednesday" "Thursday" "Friday" "Saturday" "Sunday"))
(define tsql_german_month_names '("Januar" "Februar" "März" "April" "Mai" "Juni" "Juli" "August" "September" "Oktober" "November" "Dezember"))
(define tsql_german_short_month_names '("Jan" "Feb" "Mär" "Apr" "Mai" "Jun" "Jul" "Aug" "Sep" "Okt" "Nov" "Dez"))
(define tsql_german_weekday_names '("Montag" "Dienstag" "Mittwoch" "Donnerstag" "Freitag" "Samstag" "Sonntag"))
(define tsql_language_profile (lambda (language) (match (toLower (sql_trim language))
	"us_english" '("us_english" "mdy" 7)
	"english" '("us_english" "mdy" 7)
	"deutsch" '("Deutsch" "dmy" 1)
	"german" '("Deutsch" "dmy" 1)
	_ (error "unsupported T-SQL language: " language))))
(define tsql_dateformat_value (lambda (value)
	(if (nil? value) (error "DATEFORMAT requires an order") (begin
		(define order (toLower (sql_trim value)))
		(if (has? '("mdy" "dmy" "ymd" "ydm" "myd" "dym") order) order
			(error "invalid DATEFORMAT: " value))))))
(define tsql_set_language (lambda (session language) (begin
	(define profile (tsql_language_profile language))
	(session "__tsql_language" (car profile))
	(session "__tsql_dateformat" (cadr profile))
	(session "__tsql_datefirst" (nth profile 2))
	true)))
(define tsql_language_months (close_procedure ((lambda (tsql_german_short_month_names tsql_german_month_names tsql_calendar_month_names tsql_calendar_full_month_names)
	(lambda (language short)
		(if (equal? language "Deutsch")
			(if short tsql_german_short_month_names tsql_german_month_names)
			(if short tsql_calendar_month_names tsql_calendar_full_month_names)))) tsql_german_short_month_names tsql_german_month_names tsql_calendar_month_names tsql_calendar_full_month_names)))
(define tsql_english_year (close_procedure ((lambda (tsql_parse_integer)
	(lambda (value)
		(if (regexp_test value "^[0-9]{4}$") value
			(regexp_test value "^[0-9]{2}$") (concat (if (< (tsql_parse_integer value) 50) "20" "19") value)
			nil))) tsql_parse_integer)))
(define tsql_calendar_pad (lambda (value)
	(if (equal? (strlen value) 1) (concat "0" value) value)))
(define tsql_english_date_parts (close_procedure ((lambda (tsql_english_year tsql_calendar_pad)
	(lambda (year month day) (begin
		(define year (tsql_english_year year))
		(if year (concat year "-" (tsql_calendar_pad month) "-" (tsql_calendar_pad day)) nil)))) tsql_english_year tsql_calendar_pad)))
(define tsql_named_date (close_procedure ((lambda (tsql_language_months tsql_english_date_parts)
	(lambda (parts language) (begin
		(define month_position (if (regexp_test (car parts) "^[^0-9]+$") 0 1))
		(define month (nth parts month_position))
		(define index (find (produceN 12) (lambda (index)
			(or (equal? month (toUpper (nth (tsql_language_months language true) index)))
				(equal? month (toUpper (nth (tsql_language_months language false) index))))) nil))
		(if (nil? index) nil (begin
			(define day (nth parts (if (equal? month_position 0) 1 0)))
			(if (regexp_test day "^[0-9]{1,2}$")
				(tsql_english_date_parts (nth parts 2) (string (+ index 1)) day) nil)))))) tsql_language_months tsql_english_date_parts)))
(define tsql_numeric_date (close_procedure ((lambda (tsql_english_date_parts)
	(lambda (parts order legacy) (begin
		/* Four-digit years are unambiguous. The remaining fields retain the
		language/dateformat day/month order; never guess DMY from a large day. */
		(define year_index (find '(0 1 2) (lambda (index) (equal? (strlen (nth parts index)) 4)) nil))
		(define year_index (if (nil? year_index) (match order "mdy" 2 "dmy" 2 "myd" 1 "dym" 1 _ 0) year_index))
		(define others (filter '(0 1 2) (lambda (index) (not (equal? index year_index)))))
		(define month_first (has? '("mdy" "myd" "ymd") order))
		(if (and (not legacy) (equal? order "ydm")) nil
			(tsql_english_date_parts (nth parts year_index)
				(nth parts (nth others (if month_first 0 1)))
				(nth parts (nth others (if month_first 1 0)))))))) tsql_english_date_parts)))
(define tsql_english_clock (close_procedure ((lambda (tsql_parse_integer tsql_calendar_pad)
	(lambda (value) (begin
		(define clock (replace (toUpper value) " " ""))
		(if (regexp_test clock "^[0-9]{1,2}:[0-9]{2}(:[0-9]{2}(\\.[0-9]+)?)?(AM|PM)?$") (begin
			(define meridian (if (regexp_test clock "(AM|PM)$") (substr clock (- (strlen clock) 2)) nil))
			(define clock (if meridian (substr clock 0 (- (strlen clock) 2)) clock))
			(define parts (split clock ":"))
			(define hour (tsql_parse_integer (car parts)))
			(if (and meridian (or (< hour 1) (> hour 12))) nil (begin
				(define hour (if meridian (+ (mod hour 12) (if (equal? meridian "PM") 12 0)) hour))
				(concat (tsql_calendar_pad (string hour)) ":" (cadr parts) ":"
					(if (equal? (count parts) 3) (nth parts 2) "00"))))) nil)))) tsql_parse_integer tsql_calendar_pad)))
(define tsql_datetime_text (close_procedure ((lambda (normalize_fields sql_trim tsql_named_date tsql_english_date_parts tsql_numeric_date tsql_english_clock)
	(lambda (value order language legacy) (if (nil? value) nil (begin
		(define text (sql_trim (regexp_replace (toUpper value) "\\s+" " ")))
		(define text (normalize_fields text))
		(define text (regexp_replace text ", *" " "))
		(define separated (regexp_matches text "^([0-9]{8}|[0-9]{1,4}[-/.][0-9]{1,4}[-/.][0-9]{1,4})($|[ T])"))
		(define named (if (empty_list? separated)
			(regexp_matches text "^([^0-9 :]+ [0-9]{1,2}|[0-9]{1,2} [^0-9 :]+) [0-9]{4}($| )") '()))
		(define prefix (if (not (empty_list? separated)) (sql_trim (car separated))
			(not (empty_list? named)) (sql_trim (car named)) nil))
		(define iso (and (not (nil? prefix)) (regexp_test prefix "T$")))
		(define prefix (if iso (substr prefix 0 (- (strlen prefix) 1)) prefix))
		(define date (if (nil? prefix) "1900-01-01"
			(not (empty_list? named)) (tsql_named_date (split prefix " ") language)
			(regexp_test prefix "^[0-9]{8}$") (concat (substr prefix 0 4) "-" (substr prefix 4 2) "-" (substr prefix 6 2))
			(begin
				(define parts (split (regexp_replace prefix "[-/.]" " ") " "))
				(if (or iso (and (not legacy) (regexp_test prefix "^[0-9]{4}-[0-9]{2}-[0-9]{2}$")))
					(tsql_english_date_parts (car parts) (cadr parts) (nth parts 2))
					(tsql_numeric_date parts order legacy)))))
		(define tail (if (nil? prefix) text (sql_trim (substr text (+ (strlen prefix) (if iso 1 0))))))
		(if (nil? date) nil
			(and (not (nil? prefix)) (equal? tail "")) date
			(begin
				(define clock (tsql_english_clock tail))
				(if clock (concat date " " clock) nil))))))) tsql_numeric_calendar_fields sql_trim tsql_named_date tsql_english_date_parts tsql_numeric_date tsql_english_clock)))
(define tsql_calendar_text (close_procedure ((lambda (tsql_datetime_text tsql_raise_error)
	(lambda (value dateformat language legacy) (if (nil? value) nil (begin
		(define text (tsql_datetime_text value dateformat language legacy))
		(define parsed (if text (parse_date text) nil))
		(if (nil? parsed) (tsql_raise_error 241 "invalid calendar input" 1) parsed))))) tsql_datetime_text tsql_raise_error)))
(define tsql_isdate_temporal (lambda (value)
	(if (nil? value) 0
		(and (>= (unix_timestamp value) -6847804800) (<= (unix_timestamp value) 253402300799)) 1 0)))
(define tsql_isdate_text (close_procedure ((lambda (tsql_datetime_text tsql_isdate_temporal) (lambda (value order language) (if (nil? value) 0 (begin
	(if (>= (utf16_length value) 4000) (error "ISDATE input must contain fewer than 4000 characters") true)
	(define normalized (tsql_datetime_text value order language true))
	(if (or (nil? normalized) (regexp_test normalized "\\.[0-9]{4,}")) 0 (begin
		(define parsed (parse_date normalized))
		(if (and parsed (equal? (unix_timestamp parsed) 253402300799)
			(regexp_test normalized "\\.999$")) 0 (tsql_isdate_temporal parsed)))))))) tsql_datetime_text tsql_isdate_temporal)))
(define tsql_isdate_other (lambda (value) (if (nil? value) 0 0)))
(define tsql_compile_isdate (lambda (infos) (begin
	(if (equal? (count infos) 1) true (error "ISDATE requires one argument"))
	(define input (car infos))
	(define type (sql_info_type input))
	(if (has? '("DATE" "TIME" "DATETIME2" "DATETIMEOFFSET") type)
		(tsql_raise_error 8116 (concat "Argument data type " type " is invalid for ISDATE") 1) true)
	/* Use the same declared conversion as CAST/CONVERT, never a runtime
	carrier test or a numeric day offset. Unsupported operands still evaluate. */
	(define numeric (or (sql_numeric_type? type) (has? '("BIT" "MONEY" "SMALLMONEY") type)))
	(define textual (or numeric (has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR") type)))
	(define input (if numeric ((tsql_text_compiler -1 false false) (list input)) input))
	(define formula (sql_info_formula input))
	(sql_info (if textual
		(list tsql_isdate_text formula (tsql_dateformat_formula) (tsql_language_formula))
		(list (if (has? '("DATETIME" "SMALLDATETIME") type) tsql_isdate_temporal tsql_isdate_other) formula)) "INT" nil))))
(define tsql_datename_text (lambda (value) (if (nil? value) nil (string value))))
(define tsql_datename_month (lambda (value language) (if (nil? value) nil
	(nth (tsql_language_months language false) (- (extract_date value "MONTH") 1)))))
(define tsql_datename_weekday (lambda (value language) (if (nil? value) nil
	(nth (if (equal? language "Deutsch") tsql_german_weekday_names tsql_calendar_weekday_names)
		(extract_date value "WEEKDAY")))))
(define tsql_datename_compiler (lambda (unit) (lambda (infos) (begin
	(define input (car infos))
	(define input (if (sql_text_type? (sql_info_type input))
		(sql_info (tsql_calendar_text_formula (sql_info_formula input) false) "DATETIME2" nil) input))
	(define formula (if (has? '("month" "weekday") unit)
		(list (if (equal? unit "month") (quote tsql_datename_month) (quote tsql_datename_weekday))
			(tsql_calendar_part_formula unit input) (tsql_language_formula))
		(list (quote tsql_datename_text) (sql_info_formula
			((tsql_datepart_compiler unit) (cons input (cdr infos)))))))
	(sql_info formula (list "NVARCHAR" '(4000)) '("utf8mb4_general_ci" 4))))))
(define tsql_datename_expr (lambda (unit value) (begin
	(define unit (tsql_calendar_unit unit))
	(merge (list (list (quote sql_compile_expr) (list (quote tsql_datename_compiler) unit) value)
		(if (equal? unit "week") (list (tsql_datefirst_expr)) '()))))))

/* Calendar constructors compile their input contracts once and use the same
native date value as columns and ordinary calendar arithmetic. */
(define tsql_calendar_integer_formula (lambda (info)
	(if (or (has? '("NULL" "BIT") (sql_info_type info)) (sql_numeric_type? (sql_info_type info))
		(sql_text_type? (sql_info_type info)))
		(list tsql_cast_integer (sql_info_formula info) -2147483648 2147483647)
		(error "calendar argument requires an integer-convertible SQL declaration"))))
(define tsql_calendar_from_parts (lambda (year month day)
	(if (or (nil? year) (nil? month) (nil? day)) nil (begin
		(if (and (>= year 1) (<= year 9999) (>= month 1) (<= month 12) (>= day 1) (<= day 31))
			true (error "invalid date constructor argument"))
		/* Native calendar shifts construct the value without a formatted string.
		Overflow into the following month diagnoses an impossible calendar day. */
		(define first (date_add (date_add (parse_date "0001-01-01") (intdiv (- year 1) 1) "YEAR")
			(intdiv (- month 1) 1) "MONTH"))
		(define value (date_add first (intdiv (- day 1) 1) "DAY"))
		(if (equal? (extract_date value "MONTH") month) value (error "invalid date constructor value"))))))
(define tsql_compile_datefromparts (lambda (infos)
	(sql_info (cons tsql_calendar_from_parts (map infos tsql_calendar_integer_formula)) "DATE" nil)))
(define tsql_datefromparts_expr (lambda (year month day)
	(list (quote sql_compile_expr) (quote tsql_compile_datefromparts) year month day)))
(define tsql_calendar_month_end (lambda (value amount)
	(if (or (nil? value) (nil? amount)) nil (begin
		(define shifted (tsql_calendar_shift_months value amount))
		(define first (date_add (date_trunc_day shifted) (intdiv (- 1 (extract_date shifted "DAY")) 1) "DAY"))
		(tsql_calendar_range (date_add (date_add first 1 "MONTH") -1 "DAY") -62135596800)))))
(define tsql_compile_eomonth (lambda (infos)
	(sql_info (list tsql_calendar_month_end (tsql_calendar_formula (car infos))
		(tsql_calendar_integer_formula (cadr infos))) "DATE" nil)))
(define tsql_eomonth_expr (lambda (value amount)
	(list (quote sql_compile_expr) (quote tsql_compile_eomonth) value amount)))

(define tsql_numeric_operand_type? (lambda (type)
	(or (has? '("NULL" "any" "MONEY" "SMALLMONEY") type)
		(and (sql_numeric_type? type) (not (has? '("BOOLEAN" "BOOL" "BIT") type))))))
/* SQL Server 2022 CU26 preserves these additional declared operands under
unary plus, including NULL, despite the numeric-only Microsoft syntax page.
Reproducible vendor results: https://dbfiddle.uk/DLSkEqAN
VARCHAR identity and concatenated CASE: https://dbfiddle.uk/giBimWpF
Retain the complete expression contract: no text/numeric/calendar conversion.
Other nonnumeric families remain unverified; unary minus has its own owner. */
(define tsql_compile_positive (lambda (infos) (begin
	(define info (car infos))
	(if (or (tsql_numeric_operand_type? (sql_info_type info))
		(has? '("DATETIME" "SMALLDATETIME" "DATETIME2" "NVARCHAR" "VARCHAR") (sql_info_type info))) info
		(error "unary plus requires a supported SQL operand declaration")))))
/* Bit-pattern widths come exclusively from the SQL declaration. The generic
integer operators know no SQL types; portable compiler slots disappear before
shared planning. Binary-string combinations need a verified alignment contract. */
(define tsql_bit_contract (lambda (type) (match type
	"TINYINT" '(8 255 0) "SMALLINT" '(16 65535 32768)
	"INT" '(32 4294967295 2147483648) "INTEGER" '(32 4294967295 2147483648)
	"BIGINT" '(64 -1 0) "BIT" '(1 1 0) "NULL" '(32 4294967295 2147483648)
	_ (error "bit operation requires declared integer operands"))))
(define tsql_bit_assignment_type (lambda (type)
	(if (equal? (tsql_addition_type type) "BIT") (error "bit compound assignment requires an integer target") true)))
(define tsql_bit_normalize (lambda (value mask sign)
	(if (nil? value) nil (- (bit_xor (bit_and value mask) sign) sign))))
(define tsql_bit_boolean (lambda (value) (if (nil? value) nil (if value 1 0))))
(define tsql_bit_shift (lambda (value amount width mask sign left)
	(if (or (nil? value) (nil? amount)) nil
		(or (>= amount width) (<= amount (- 0 width))) 0
		(begin
			(define reverse (< amount 0))
			(define distance (if reverse (- 0 amount) amount))
			(define unsigned (bit_and value mask))
			(define shifted (if (if reverse (not left) left)
				(bit_shift_left unsigned distance) (bit_shift_right unsigned distance)))
			(tsql_bit_normalize shifted mask sign)))))
(define tsql_bit_formula (lambda (info)
	(if (equal? (tsql_addition_type (sql_info_type info)) "BIT")
		(list (quote tsql_bit_boolean) (sql_info_formula info)) (sql_info_formula info))))
(define tsql_bit_compiler (lambda (operation) (lambda (infos) (begin
	(define types (map infos (lambda (info) (tsql_addition_type (sql_info_type info)))))
	(map types tsql_bit_contract)
	(define shift (has? '("left" "right") operation))
	(if (and shift (has? types "BIT")) (error "shift requires declared integer operands") true)
	(define type (if (or shift (equal? operation "not")) (car types)
		(coalesceNil (find '("BIGINT" "INT" "INTEGER" "SMALLINT" "TINYINT" "BIT")
			(lambda (candidate) (has? types candidate)) nil) "INT")))
	(define result_type (if (equal? type "NULL") "INT"
		(and (equal? type "BIT") (not (equal? operation "not"))) "TINYINT" type))
	(define contract (tsql_bit_contract result_type))
	(define mask (cadr contract))
	(define sign (nth contract 2))
	(define formulas (map infos tsql_bit_formula))
	(define formula (if shift
		(list (quote tsql_bit_shift) (car formulas) (cadr formulas) (car contract) mask sign (equal? operation "left"))
		(and (equal? operation "not") (equal? type "BIT")) (list (quote sql_not) (sql_info_formula (car infos)))
		(begin
			(define raw (cons (match operation "and" (quote bit_and) "or" (quote bit_or)
				"xor" (quote bit_xor) "not" (quote bit_not)) formulas))
			(if (equal? result_type "BIGINT") raw (list (quote tsql_bit_normalize) raw mask sign)))))
	(sql_info formula result_type nil)))))
(define tsql_bit_expr (lambda (operation arguments)
	(cons (quote sql_compile_expr) (cons (list (quote tsql_bit_compiler) operation) arguments))))
/* Text construction is selected from declared SQL contracts. Runtime helpers
only handle values; no SCM carrier selects a SQL type or character encoding. */
(define tsql_quote_name (lambda (value delimiter)
	(if (or (nil? value) (nil? delimiter)) nil
		/* At most three UTF8 bytes per UTF16 unit; refuse long inputs before decoding. */
		(> (strlen value) 384) nil (> (utf16_length value) 128) nil
		(begin
			(define pair (match delimiter "'" '("'" "'") "\"" '("\"" "\"")
				"[" '("[" "]") "]" '("[" "]") "(" '("(" ")") ")" '("(" ")")
				"{" '("{" "}") "}" '("{" "}") "<" '("<" ">") ">" '("<" ">")
				"`" '("`" "`") _ nil))
			(if (nil? pair) nil
				(concat (car pair) (replace value (cadr pair) (concat (cadr pair) (cadr pair))) (cadr pair)))))))
/* Identifier functions share declared text conversion, preserving raw text
handles until sysname bounds have been checked. */
(define tsql_identifier_text (lambda (info) (begin
	(define type (sql_info_type info))
	/* The shared text family includes byte strings. Identifier input must
	reject those before admitting character handles without conversion. */
	(if (tsql_binary_type? type) (error "text construction input conversion is not implemented for " type)
		(or (sql_text_type? type) (equal? type "NULL")) (sql_info_formula info)
		(sql_info_formula ((tsql_text_compiler -1 true false) (list info)))))))
(define tsql_compile_quotename (lambda (infos)
	(sql_info (cons (quote tsql_quote_name) (map infos tsql_identifier_text))
		"NVARCHAR" (sql_info_collation (car infos)))))
/* Multipart identifiers reuse the frontend identifier grammar and existing
UTF16 bounds. They do not resolve catalog objects or inspect runtime SQL types. */
(define tsql_sysname? (lambda (text)
	(and (<= (strlen text) 384) (<= (utf16_length text) 128))))
/* Reuse the SQL identifier unescaper for quoted parts. The scalar utility also
accepts numeric and keyword components, without treating them as SQL tokens.
An explicit full-input alternative returns NULL for malformed names; parser or
runtime errors are never caught and classified by their message text. */
(define tsql_name_parts (parser (or
	(parser '((define parts (+ (or tsql_identifier_quoted
		(regex "[\\p{L}\\p{N}_@$#]+") (parser empty nil)) "."))
		(regex "\\z" false false)) parts)
	(parser (regex "(?s).*" false false) nil))))
(define tsql_parsename (lambda (text piece)
	(if (or (nil? text) (nil? piece)) nil
		(or (< piece 1) (> piece 4)) nil
		(begin
			(define parts (tsql_name_parts text))
			(if (or (nil? parts) (> (count parts) 4) (> piece (count parts)) (nil? (car (reverse parts)))) nil
				(begin
					(define part (nth parts (- (count parts) piece)))
					(if (and (not (nil? part)) (not (equal? part "")) (tsql_sysname? part)) part nil)))))))
(define tsql_compile_parsename (lambda (infos) (begin
	(if (equal? (count infos) 2) true (error "PARSENAME has an invalid argument count"))
	(define text_info (car infos))
	(define piece_info (cadr infos))
	(define piece (sql_type_formula '() (tsql_cast_expr
		(list (quote sql_type_expr) (sql_info_formula piece_info) (sql_info_datatype piece_info) (sql_info_collation piece_info))
		"INT" '())))
	(sql_info (list (quote tsql_parsename) (tsql_identifier_text text_info) piece)
		"NVARCHAR" (sql_info_collation text_info)))))

(define tsql_space (lambda (amount)
	(if (or (nil? amount) (< amount 0)) nil (string_repeat " " (min amount 8000)))))
(define tsql_compile_space (lambda (infos) (begin
	(define info (car infos))
	(define type (sql_info_type info))
	(if (or (tsql_character_type? type)
		(has? '("TINYINT" "SMALLINT" "INT" "INTEGER" "BIGINT" "DECIMAL" "NUMERIC" "FLOAT" "REAL") type)) true
		(error "space count conversion is not implemented for " type))
	(define amount (sql_type_formula '() (tsql_cast_expr
		(list (quote sql_type_expr) (sql_info_formula info) type (sql_info_collation info)) "INT" '())))
	(sql_info (list (quote tsql_space) amount) "VARCHAR" '("utf8mb4_general_ci" 4)))))
(define tsql_positive_expr (lambda (value)
	(list (quote sql_compile_expr) (quote tsql_compile_positive) value)))
(define tsql_compile_negate (lambda (infos) (begin
	(define info (car infos))
	(define formula (sql_info_formula info))
	(define type (sql_info_type info))
	/* Diagnose constant conversion overflow using the declared range.
	Column-bearing operands are resolved by the shared compiler first. */
	(match formula
		'(conversion operand minimum maximum)
		(if (and (or (equal? conversion (quote tsql_cast_integer)) (equal? conversion tsql_cast_integer)) (plain_literal_expr? operand)
			(< minimum 0) (equal? (tsql_cast_integer operand minimum maximum) minimum))
			(error "unary integer conversion overflow") true)
		_ true)
	(if (tsql_numeric_operand_type? type) true
		(error "unary minus requires a numeric operand"))
	(define negated (list (quote -) 0 formula))
	(sql_info (if (has? '("MONEY" "SMALLMONEY") type)
		(list tsql_cast_money negated (equal? type "SMALLMONEY")) negated)
		(if (equal? type "TINYINT") "SMALLINT" type) nil))))
(define tsql_negate_expr (lambda (value)
	(list (quote sql_compile_expr) (quote tsql_compile_negate) value)))
(define tsql_key_placement (parser (? (or (atom "CLUSTERED" true) (atom "NONCLUSTERED" true)))))
(define tsql_key_columns (parser (define columns (+ (parser '(
	(define name tsql_identifier) (? (or (atom "ASC" true) (atom "DESC" true)))) name) ",")) columns))
/* Physical access preferences are accepted as syntax only. They do not
enter the AST or constrain the common planner, native isolation, or autoindexes.
Semantic requests to bypass constraints/triggers are deliberately excluded. */
(define tsql_hint_index (parser (or tsql_int tsql_identifier)))
(define tsql_table_hint (parser (or
	(parser '((atom "INDEX" true) (? "=") "(" (+ tsql_hint_index ",") ")") true)
	(parser '((atom "FORCESEEK" true) (? "(" tsql_hint_index "(" (+ tsql_identifier ",") ")" ")")) true)
	(parser (regex "(?:NOEXPAND|FORCESCAN|HOLDLOCK|NOLOCK|NOWAIT|PAGLOCK|READCOMMITTEDLOCK|READCOMMITTED|READPAST|READUNCOMMITTED|REPEATABLEREAD|ROWLOCK|SERIALIZABLE|SNAPSHOT|TABLOCKX|TABLOCK|UPDLOCK|XLOCK)\\b" true) true)
)))
(define tsql_table_hints (parser (or
	(parser '((atom "WITH" true) "(" (+ tsql_table_hint (? ",")) ")") true)
	/* The legacy parenthesized spelling permits a single simple hint. */
	(parser '("(" (regex "(?:NOLOCK|READUNCOMMITTED|READCOMMITTED|REPEATABLEREAD|SERIALIZABLE|HOLDLOCK|ROWLOCK|PAGLOCK|TABLOCKX|TABLOCK|UPDLOCK|XLOCK|NOWAIT)\\b" true) ")") true)
)))
(define tsql_query_hint (parser (or
	(parser '((regex "(?:HASH|LOOP|MERGE)\\b" true) (atom "JOIN" true)) true)
	(parser '((regex "(?:HASH|ORDER)\\b" true) (atom "GROUP" true)) true)
	(parser '((regex "(?:HASH|MERGE|CONCAT)\\b" true) (atom "UNION" true)) true)
	(parser '((atom "FORCE" true) (atom "ORDER" true)) true)
	(parser (regex "(?:RECOMPILE|ROBUST\\s+PLAN|KEEPFIXED\\s+PLAN|KEEP\\s+PLAN)\\b" true) true)
	(parser '((regex "(?:MAXDOP|MAXRECURSION|FAST)\\b" true) tsql_int) true)
	(parser '((atom "USE" true) (atom "HINT" true) "(" (+ tsql_string ",") ")") true)
	(parser '((atom "OPTIMIZE" true) (atom "FOR" true) (or
		(atom "UNKNOWN" true)
		'("(" (+ '("@" tsql_identifier (or (atom "UNKNOWN" true) '("=" tsql_literal))) ",") ")"))) true)
	(parser '((atom "TABLE" true) (atom "HINT" true) "(" tsql_identifier "," (+ tsql_table_hint ",") ")") true)
)))
(define tsql_query_hints (parser '((atom "OPTION" true) "(" (+ tsql_query_hint ",") ")") true))
/* Index layout options are compile-time spelling, not persistent metadata.
The two options which change uniqueness/admission remain explicit contracts. */
(define tsql_on_off (parser (or (atom "ON" true) (atom "OFF" true))))
(define tsql_index_wait (parser '("(" (atom "WAIT_AT_LOW_PRIORITY" true) "("
	(atom "MAX_DURATION" true) "=" tsql_int (atom "MINUTES" true) ","
	(atom "ABORT_AFTER_WAIT" true) "=" (regex "(?:NONE|SELF|BLOCKERS)\\b" true) ")" ")") true))
(define tsql_index_option (parser (or
	(parser '((define name (regex "(?:IGNORE_DUP_KEY|DROP_EXISTING)\\b" true)) "=" (define value tsql_on_off))
		(list (toUpper name) (equal? (toUpper value) "ON")))
	(parser '((regex "(?:PAD_INDEX|SORT_IN_TEMPDB|STATISTICS_NORECOMPUTE|STATISTICS_INCREMENTAL|ALLOW_ROW_LOCKS|ALLOW_PAGE_LOCKS|OPTIMIZE_FOR_SEQUENTIAL_KEY|RESUMABLE|LOB_COMPACTION)\\b" true) "=" tsql_on_off) '())
	(parser '((atom "ONLINE" true) "=" tsql_on_off (? tsql_index_wait)) '())
	(parser '((regex "(?:FILLFACTOR|MAXDOP|MAX_DURATION)\\b" true) "=" tsql_int (? (atom "MINUTES" true))) '())
	(parser '((atom "DATA_COMPRESSION" true) "=" (regex "(?:NONE|ROW|PAGE|COLUMNSTORE_ARCHIVE|COLUMNSTORE)\\b" true)) '())
)))
(define tsql_index_options (parser '((atom "WITH" true) "(" (define options (+ tsql_index_option ",")) ")") (merge options)))
(define tsql_index_location (parser '((atom "ON" true) tsql_identifier (? "(" tsql_identifier ")")) true))
(define tsql_check_index_options (lambda (options unique) (match options
	(cons _ _) (begin
		(if (and unique (options "IGNORE_DUP_KEY")) (error "IGNORE_DUP_KEY=ON requires native duplicate suppression") true)
		/* Even an ordinary acceleration declaration may replace a UNIQUE index.
		Reject replacement until the native key operation owns that transition. */
		(if (options "DROP_EXISTING") (error "DROP_EXISTING=ON requires native atomic key replacement") true)
		true)
	_ true)))
(define tsql_key_layout (parser '((? (define options tsql_index_options)) (? tsql_index_location))
	(tsql_check_index_options options true)))
(define tsql_default_literal (parser (or
	(parser '("(" (define value tsql_default_literal) ")") value)
	(parser (define value tsql_literal) value))))
(define tsql_clock_default (parser (or
	(parser '("(" tsql_clock_default ")") true)
	'((atom "GETDATE" true) "(" ")") '((atom "SYSDATETIME" true) "(" ")")
	(atom "CURRENT_TIMESTAMP" true))))
(define tsql_column_attributes (lambda (default_parser) (parser (define attrs (* (or
	(parser '((? (atom "CONSTRAINT" true) (define key tsql_identifier)) (atom "PRIMARY" true) (atom "KEY" true) tsql_key_placement tsql_key_layout)
		(if key (list "primary" true "null" false "constraint_name" key) '("primary" true "null" false)))
	(parser '((? (atom "CONSTRAINT" true) (define key tsql_identifier)) (atom "UNIQUE" true) tsql_key_placement tsql_key_layout)
		(if key (list "unique" true "nulls_not_distinct" true "constraint_name" key) '("unique" true "nulls_not_distinct" true)))
	(parser (atom "ROWGUIDCOL" true) '("rowguidcol" true))
	(parser '((atom "NOT" true) (atom "NULL" true)) '("null" false))
	(parser (atom "NULL" true) '("null" true))
	(parser '((atom "IDENTITY" true) (? "(" (define seed tsql_int) "," (define increment tsql_int) ")"))
		(if (and (or (nil? seed) (>= seed 1)) (or (nil? increment) (equal? increment 1)))
			(list "auto_increment" true "identity_seed" (coalesceNil seed 1))
			(error "IDENTITY currently requires a positive seed and increment 1")))
	(parser '((? (atom "CONSTRAINT" true) (define key tsql_identifier))
		(atom "DEFAULT" true) (define value default_parser))
		(merge value (if key (list "default_name" key) '())))
	(parser '((atom "COLLATE" true) (define collation tsql_identifier)) (list "collate" collation))
))) (begin
		(if (> (count (filter attrs (lambda (attr) (attr "rowguidcol")))) 1)
			(error "duplicate ROWGUIDCOL declaration") true)
		(if (> (count (filter attrs (lambda (attr)
			(or (has_assoc? attr "default") (has_assoc? attr "default_expression"))))) 1)
			(error "multiple DEFAULT declarations") true)
		(if (and (find attrs (lambda (attr) (attr "auto_increment")))
			(find attrs (lambda (attr) (attr "null"))))
			(error "IDENTITY column was declared NULL") true)
		(if (and (find attrs (lambda (attr) (attr "primary")))
			(find attrs (lambda (attr) (attr "null"))))
			(error "PRIMARY KEY column was declared NULL") true)
		(if (and (> (count (filter attrs (lambda (attr) (or (attr "primary") (attr "unique"))))) 1)
			(find attrs (lambda (attr) (not (nil? (attr "constraint_name"))))))
			(error "Multiple inline key declarations with constraint names are not supported") true)
		(merge attrs)))))
/* PRIMARY KEY nullability is a frontend decision expressed through native
column options; the engine does not infer SQL semantics from the key ID. */
(define tsql_table_primary_columns (lambda (definitions) (begin
	(define names (merge (map definitions (lambda (definition) (match definition
		'((symbol list) "column" name _type _dimensions _attributes) (list name)
		_ '())))))
	(if (equal? (count names) (count (merge_unique (list (map names toLower)))))
		true (error "duplicate column name"))
	(define canonical (lambda (name)
		(coalesceNil (find names (lambda (declared) (equal?? (toLower declared) (toLower name))))
			(error "key column does not exist: " name))))
	(define normalized (map definitions (lambda (definition) (match definition
		'((symbol list) "unique" id columns options)
		(list (quote list) "unique" id (cons (quote list) (map (cdr columns) canonical)) options)
		_ definition))))
	(define primary_columns (merge (map normalized (lambda (definition) (match definition
		'((symbol list) "unique" "PRIMARY" columns _) (cdr columns)
		_ '())))))
	(map normalized (lambda (definition) (match definition
		'((symbol list) "column" name type dimensions attributes)
		(if (has? primary_columns name) (begin
			(if ((cdr attributes) "null") (error "PRIMARY KEY column was declared NULL") true)
			(list (quote list) "column" name type dimensions (append attributes "null" false)))
			definition)
		_ definition))))))

/* The seed is parser data, consumed into the existing table option.
Only the native auto_increment flag reaches the column declaration. */
(define tsql_identity_table_options (lambda (definitions) (begin
	(define identities (filter definitions (lambda (definition) (match definition
		((symbol list) "column" _name _type _dimensions (cons (symbol list) attributes)) (attributes "auto_increment")
		_ false))))
	(if (> (count identities) 1) (error "only one IDENTITY column is allowed") true)
	(match identities
		'(((symbol list) "column" _name type dimensions (cons (symbol list) attributes))) (begin
			(define seed (coalesceNil (attributes "identity_seed") 1))
			(define maximum (tsql_identity_maximum type (cdr dimensions)))
			(if (> seed maximum) (error "IDENTITY seed exceeds the declared range") true)
			(if (equal? seed 1) '(list) (list (quote list) "auto_increment" seed)))
		_ '(list)))))
(define tsql_identity_native_column (lambda (definition) (match definition
	((symbol list) "column" name type dimensions (cons (symbol list) attributes))
	(if (has_assoc? attributes "identity_seed")
		(list (quote list) "column" name type dimensions
			(cons (quote list) (filter_assoc attributes (lambda (key _value) (not (equal? key "identity_seed"))))))
		definition)
	_ definition)))
/* SQL bounds come from declared metadata, not the native value tag. Wide
numeric identities retain the native integer carrier until generic precision
support exists; exact precision-limit emulation remains deferred. */
(define tsql_identity_maximum (lambda (type dimensions) (match (toUpper type)
	"TINYINT" 255 "SMALLINT" 32767 "INT" 2147483647 "INTEGER" 2147483647
	"BIGINT" (tsql_parse_integer "9223372036854775807")
	"DECIMAL" (tsql_identity_decimal_maximum dimensions)
	"NUMERIC" (tsql_identity_decimal_maximum dimensions)
	_ (error "unsupported IDENTITY declaration"))))
(define tsql_identity_decimal_maximum (lambda (dimensions) (begin
	(define precision (if (empty_list? dimensions) 18 (car dimensions)))
	(if (>= precision 19) (tsql_parse_integer "9223372036854775807")
		(tsql_parse_integer (reduce (produceN precision) (lambda (text _index) (concat text "9")) ""))))))
/* Scope ownership is supplied once by the batch/procedure runner. Direct SCM
callers without a batch owner retain their explicitly supplied invocation scope.
The shared connection owns @@IDENTITY; nested scope owners own SCOPE_IDENTITY.
Neither SQL function infers its declared NUMERIC contract from the carrier. */
(define tsql_identity_scope (lambda (session)
	(coalesceNil (session "__tsql_scope") session)))
(define tsql_read_identity (lambda (value)
	(match value
		'("unresolved" reason) (error "identity completion for " reason " is not supported yet")
		'("value" value) value
		_ nil)))
(define tsql_identity_expr (lambda (scoped)
	(list (quote sql_type_expr)
		(list (quote tsql_read_identity)
			(list (quote session) (if scoped "__tsql_scope_identity" "__tsql_identity")))
		"NUMERIC" nil)))
(define tsql_complete_identity (lambda (session value) (begin
	(define state (list "value" value))
	(session "__tsql_identity" state)
	((tsql_identity_scope session) "__tsql_scope_identity" state)
	true)))
/* Streaming native sinks execute in parallel. Until the common sink exports a
completion reduction, refuse identity reads rather than race a session write or
read another writer's global allocator. The INSERT itself remains unchanged. */
(define tsql_unresolved_identity (lambda (session reason) (begin
	(define state (list "unresolved" reason))
	(session "__tsql_identity" state)
	((tsql_identity_scope session) "__tsql_scope_identity" state)
	true)))
/* This setting owns no IDs or schema state. It retains the existing logical
handle so a replacement table cannot inherit authority from a reused name.
The current directory/name and column descriptors remain native metadata. */
(define tsql_identity_mode_live? (lambda (mode tx)
	(if (nil? mode) false (begin
		(define handle (cadr mode))
		(define current (table (car mode) false (show handle "name") false tx))
		(if (identical? current handle)
			(begin (define declaration (show handle "column" (nth mode 2) false tx))
				(if (nil? declaration) false (equal? (declaration "Extra") "auto_increment"))) false)))))
(define tsql_set_identity_insert (lambda (session database name enabled tx) (begin
	((tsql_temporary_policy (coalesceNil (session "username") "root") (tsql_temporary_bindings session)) database name true)
	(define handle (tsql_native_relation_handle database name tx))
	(if (nil? handle) (tsql_raise_error 208 "IDENTITY_INSERT target does not exist" 1) true)
	(define identity (find (show handle "columns" tx) (lambda (column) (equal? (column "Extra") "auto_increment")) nil))
	(if (nil? identity) (tsql_raise_error 8106 "IDENTITY_INSERT requires an IDENTITY column" 1) true)
	(define previous (session "__tsql_identity_insert"))
	(define same (if (nil? previous) false (identical? handle (cadr previous))))
	(if enabled
		(begin
			(if (if same false (tsql_identity_mode_live? previous tx))
				(tsql_raise_error 8107 "IDENTITY_INSERT is already ON for another table" 1) true)
			(session "__tsql_identity_insert" (list database handle (identity "Field"))))
		same (session "__tsql_identity_insert" nil) true)
	0)))
(define tsql_check_identity_insert (lambda (session handle explicit_columns supplied) (begin
	(define mode (session "__tsql_identity_insert"))
	(define enabled (if (nil? mode) false (identical? handle (cadr mode))))
	(if supplied
		(if (and explicit_columns enabled) true
			(tsql_raise_error 544 "explicit IDENTITY values require a column list and IDENTITY_INSERT ON" 1))
		enabled (tsql_raise_error 545 "IDENTITY_INSERT ON requires an explicit IDENTITY value" 1) true))))
/* Only compile-time declarations choose identity policy and value bounds.
The statement binds one native handle before either its source or allocator;
parallel SELECT sinks borrow it without session-setting probes per row. */
(define tsql_insert_identity (lambda (database name planning_session tx)
	(if (local_relation_reference? name) nil (begin
		(define identity (find (get_schema database name tx)
			(lambda (column) (equal? (column "Extra") "auto_increment")) nil))
		(if identity (tsql_target_column_declaration database name (identity "Field") planning_session tx)
			(begin
				/* Absence consumes a fact about every declaration, not only the
				provided targets. Enumerate their immutable metadata, never rows:
				DROP/CREATE or ADD may introduce identity beside unchanged targets. */
				(planner_record_guard_condition
					(list (quote nil?) (list (quote find) (list (quote get_schema) database name (quote tx))
						(list (quote lambda) (list (quote column))
							(list (quote equal?) (list (quote column) "Extra") "auto_increment")) nil)) planning_session)
				nil))))))
(define tsql_identity_insert_statement (lambda (database name identity explicit_columns columns body) (begin
	(define target (if (local_relation_reference? name) (local_relation_symbol name)
		(list 'table database false name false 'tx)))
	(physical_mutation_producer (list target) (list "insert" '()) (list (quote begin)
		(list (quote define) (quote __tsql_insert_target) target)
		(if identity
			(list (quote tsql_check_identity_insert) (quote session) (quote __tsql_insert_target)
				explicit_columns (has? columns (identity "Field"))) true)
		body)))))
/* Emit the existing native INSERT. VALUES and DEFAULT supply their bounded
syntax batch; the existing SELECT result sink supplies one emitted row. Never
scan/count/materialize SELECT input to reserve IDs, or rewind consumed blocks.
The frontend checks the complete reserved block before any row publication.
Explicit NULL must fail here: native auto_increment would otherwise allocate
an unchecked replacement. Completion uses only this invocation's reserved range. */
(define tsql_native_insert_expr (lambda (relation identity columns values row_count capture_identity deferred) (begin
	(define columns (merge (list columns (map deferred car))))
	(define values (if (empty_list? deferred) values
		(list (quote map) values (list (quote lambda) (list (quote __default_row))
			(list (quote merge) (list (quote list) (quote __default_row)
				(cons (quote list) (map deferred cadr))))))))
	(define native_insert (lambda (names rows callback)
		(list (quote insert) relation (list (quote quote) names) rows
			'(list) nil false callback (quote tx))))
	(define without_generated_identity (lambda (formula)
		(if capture_identity
			/* MATCH evaluates the native call before its completion mutation.
			A single-use DEFINE may be substituted into the later return expression. */
			(list (quote match) formula (quote __tsql_insert_result)
				(list (quote begin) (list (quote session) "__tsql_identity" nil)
					(quote __tsql_insert_result))) formula)))
	(define callback '('lambda '('id) '('session "last_insert_id" 'id)))
	/* Guarded identity absence proves the completion callback unreachable.
	Do not capture a caller session in delegated identity-free row sinks. */
	(if (nil? identity) (without_generated_identity (native_insert columns values nil))
		(equal? row_count 0) (without_generated_identity (native_insert columns values nil))
		(begin
			(define explicit (find (produceN (count columns))
				(lambda (index) (equal?? (nth columns index) (identity "Field"))) nil))
			(if (not (nil? explicit))
				'('begin
					'('define '__tsql_rows values)
					'('reduce '__tsql_rows
						'('lambda '('valid 'row)
							'('if '('nil? '('nth 'row explicit)) '('error "IDENTITY cannot be assigned NULL") 'valid)) true)
					(match (native_insert columns (quote __tsql_rows) callback) explicit_insert
						(if capture_identity
							'('match explicit_insert '__tsql_explicit_result
								'('begin '('tsql_unresolved_identity 'session "explicit IDENTITY insertion")
									'__tsql_explicit_result)) explicit_insert)))
				(begin
					(define maximum (tsql_identity_maximum (identity "RawType") (identity "Dimensions")))
					/* Batch extent comes from VALUES syntax or the existing sink's
					single-row contract, never a scan/count of SELECT input. */
					(define own_rows (if (equal? row_count 1)
						'('list '('cons '('car '__tsql_ids) '('car '__tsql_rows)))
						'('map '('produceN row_count)
							'('lambda '('index) '('cons '('+ '('car '__tsql_ids) 'index) '('nth '__tsql_rows 'index))))))
					'('begin
						'('define '__tsql_rows values)
						'('define '__tsql_ids '('reserve_ids relation row_count (quote tx)))
						'('if '('> '('cadr '__tsql_ids) maximum) '('error "IDENTITY exceeds the declared range") true)
						'('define '__tsql_inserted (native_insert (cons (identity "Field") columns) own_rows nil))
						/* Preserve completion only after successful native admission. */
						'('if '('> '__tsql_inserted 0) '('begin
							'('session "last_insert_id" '('car '__tsql_ids))
							(if capture_identity '('tsql_complete_identity 'session '('cadr '__tsql_ids)) true)) true)
						'__tsql_inserted))))))))
/* Bounds are invocation scalars, never per-row expressions. SQL conversion is
chosen from the frontend contract; only the count expression reaches the scan. */
(define tsql_row_count (lambda (value)
	(if (or (nil? value) (< value 0)) (error "TOP requires a nonnegative count") value)))
(define tsql_compile_row_count (lambda (infos) (begin
	(define info (tsql_convert_value_info (car infos) "BIGINT" nil))
	(sql_info (list (quote tsql_row_count) (sql_info_formula info)) "BIGINT" nil))))
(define tsql_select_suffix (parser (? (atom "OFFSET" true) (define offset tsql_int)
	(or (atom "ROW" true) (atom "ROWS" true))
	(? (atom "FETCH" true) (or (atom "FIRST" true) (atom "NEXT" true))
		(define limit tsql_int) (or (atom "ROW" true) (atom "ROWS" true)) (atom "ONLY" true)))
	(if (nil? offset) nil (if (or (< offset 0) (and limit (< limit 0)))
		(error "OFFSET and FETCH require nonnegative counts") (list limit offset)))))
/* Operator selection and implicit conversions use the SQL argument contracts.
Only the selected native operation survives the shared compile boundary. */
(define tsql_addition_numeric_types '("FLOAT" "REAL" "DECIMAL" "MONEY" "SMALLMONEY" "BIGINT" "INT" "SMALLINT" "TINYINT" "BIT"))
(define tsql_addition_text_types '("NVARCHAR" "NCHAR" "VARCHAR" "CHAR"))
/* Deprecated large text declarations retain their own higher CASE/COALESCE
precedence. Their implicit character conversion does not admit TEXT to +. */
(define tsql_common_text_types '("NTEXT" "TEXT" "NVARCHAR" "NCHAR" "VARCHAR" "CHAR"))
(define tsql_addition_type (lambda (type) (match type
	"INTEGER" "INT" "DOUBLE" "FLOAT" "NUMERIC" "DECIMAL"
	"BOOLEAN" "BIT" "BOOL" "BIT" _ type)))
(define tsql_arithmetic_temporal? (lambda (info)
	(or (tsql_temporal_type? (sql_info_type info)) (equal? (sql_info_type info) "DATETIMEOFFSET"))))
/* Numeric operands denote days, independent of the native Unix carrier.
Split whole days and fractional nanoseconds before the existing result-type
quantizer applies DATETIME ticks and the declaration's calendar range. */
(define tsql_shift_numeric_days (close_procedure ((lambda (raise_error)
	(lambda (value amount direction) (if (or (nil? value) (nil? amount)) nil (begin
		/* No valid legacy input can survive an offset this large. Validate it
		before constructing integer parts, including very large FLOATs. */
		(if (and (>= amount -6000000) (<= amount 6000000)) true
			(raise_error 242 "temporal day arithmetic overflow" 1))
		(define days (* amount direction))
		(define whole (intdiv days 1))
		(define fraction (round (* (- days whole) 86400000000000)))
		(define parts (date_parts value))
		(define nanos (+ (cadr parts) fraction))
		(define carry (floor (/ nanos 1000000000)))
		(date_from_parts (+ (car parts) (* whole 86400) carry)
			(- nanos (* carry 1000000000)) (nth parts 2)))))) tsql_raise_error)))
/* Legacy arithmetic uses days since 1900, while native calendar values use
Unix seconds. Combine their integer seconds and normalized subsecond parts;
never turn an absolute calendar into a floating-point day counter.
SQL Server 2022 reference: https://dbfiddle.uk/GmtTRq-M */
(define tsql_legacy_temporal_pair (close_procedure (lambda (left right direction)
	(if (or (nil? left) (nil? right)) nil (begin
		(define first (date_parts left))
		(define second (date_parts right))
		(define nanos (+ (cadr first) (* direction (cadr second))))
		(define carry (floor (/ nanos 1000000000)))
		(date_from_parts (+ (car first) (* direction (+ (car second) 2208988800)) carry)
			(- nanos (* carry 1000000000)) (nth first 2)))))))
/* DATETIME-to-integer rounds the signed day count, including a half day
before the epoch. Split by days so large calendar values never multiply an
absolute seconds count by a nanosecond scale. */
(define tsql_legacy_integer_days (close_procedure (lambda (value)
	(if (nil? value) nil (begin
		(define parts (date_parts value))
		(define seconds (+ (car parts) 2208988800))
		(define days (floor (/ seconds 86400)))
		(define clock (- seconds (* days 86400)))
		(define upward (if (< days 0)
			(or (> clock 43200) (and (equal? clock 43200) (> (cadr parts) 0)))
			(>= clock 43200)))
		(+ days (if upward 1 0)))))))
(define tsql_compile_legacy_temporal_pair (lambda (infos subtract) (begin
	(define type (if (find infos (lambda (info) (equal? (sql_info_type info) "DATETIME")) nil)
		"DATETIME" "SMALLDATETIME"))
	(define value (cons tsql_legacy_temporal_pair
		(append (map infos tsql_calendar_formula) (if subtract -1 1))))
	(sql_info (if (equal? type "SMALLDATETIME") (list tsql_calendar_small_datetime value)
		(list tsql_temporal_quantize value 300 -6847804800)) type nil))))
(define tsql_compile_temporal_day_arithmetic (lambda (infos subtract) (begin
	(define left (car infos))
	(define right (cadr infos))
	(define left_type (tsql_addition_type (sql_info_type left)))
	(define right_type (tsql_addition_type (sql_info_type right)))
	(if (reduce infos (lambda (valid info)
		(and valid (or (not (tsql_arithmetic_temporal? info))
			(tsql_temporal_numeric_target? (sql_info_type info))))) true) true
		(error "day arithmetic is only supported for DATETIME and SMALLDATETIME"))
	(define left_date (tsql_temporal_numeric_target? left_type))
	(define right_date (tsql_temporal_numeric_target? right_type))
	(if (and left_date right_date) (tsql_compile_legacy_temporal_pair infos subtract) (begin
		(if (and subtract right_date) (error "numeric minus temporal arithmetic is not implemented") true)
		(define temporal (if left_date left right))
		(define interval (if left_date right left))
		(define type (sql_info_type temporal))
		(define interval_type (tsql_addition_type (sql_info_type interval)))
		(if (or (equal? interval_type "NULL") (has? tsql_addition_numeric_types interval_type)) true
			(error "legacy temporal arithmetic requires a declared numeric day offset"))
		/* SMALLDATETIME's fractional arithmetic has a distinct rounding policy
		from its conversion operator. Keep that profile explicit until modeled. */
		(if (and (equal? type "SMALLDATETIME")
			(not (has? '("NULL" "BIT" "TINYINT" "SMALLINT" "INT" "BIGINT") interval_type)))
			(error "fractional SMALLDATETIME arithmetic is not implemented") true)
		(define amount (sql_info_formula (tsql_convert_value_info interval "FLOAT" nil)))
		(define shifted (list tsql_shift_numeric_days (tsql_calendar_formula temporal) amount (if subtract -1 1)))
		(sql_info (if (equal? type "SMALLDATETIME") (list tsql_calendar_small_datetime shifted)
			(list tsql_temporal_quantize shifted 300 -6847804800)) (sql_info_datatype temporal) nil))))))
(define tsql_compile_addition (lambda (infos) (begin
	(if (find infos tsql_arithmetic_temporal? nil) (tsql_compile_temporal_day_arithmetic infos false) (begin
		(define types (map infos (lambda (info) (tsql_addition_type (sql_info_type info)))))
		(define active (filter types (lambda (type) (not (equal? type "NULL")))))
		(define numeric (if (empty_list? active) "INT"
			(find tsql_addition_numeric_types (lambda (type) (has? active type)) nil)))
		(define text (find tsql_addition_text_types (lambda (type) (has? active type)) nil))
		(define target (if (empty_list? active) "INT" numeric numeric text text
			(error "unresolved or unsupported SQL type for +")))
		(if (or (equal? target "BIT")
			(not (reduce active (lambda (valid type) (and valid
				(or (has? tsql_addition_numeric_types type) (has? tsql_addition_text_types type)))) true)))
			(error "unsupported SQL operands for +: " types) true)
		(define formulas (map infos (lambda (info) (begin
			(define type (tsql_addition_type (sql_info_type info)))
			(define formula (sql_info_formula info))
			(if (and numeric (or (has? tsql_addition_text_types type) (equal? type "BIT")))
				(sql_type_formula '() (tsql_cast_expr formula
					(if (has? '("DECIMAL" "MONEY" "SMALLMONEY") target) "FLOAT" target) '()))
				formula)))))
		(sql_info (cons (if numeric (quote +) (quote concat)) formulas) target
			(if numeric nil (reduce infos (lambda (collation info)
				(sql_merge_collation collation (sql_info_collation info))) nil))))))))
(define tsql_add_expr (lambda (left right)
	(list (quote sql_compile_expr) (quote tsql_compile_addition) left right)))
/* Character/integer subtraction uses the INT operand's SQL precedence.
The generic dynamically typed arithmetic contract deliberately leaves mixed
text/numeric inputs as any. Convert both operands here, before MIN/derived
projection typing, and check the INT result range through the existing cast
owner. No input row value participates in result-type selection.
https://learn.microsoft.com/en-us/sql/t-sql/language-elements/subtract-transact-sql */
(define tsql_compile_text_integer_subtraction (lambda (infos) (begin
	(define operands (map infos (lambda (info) (tsql_convert_value_info info "INT" nil))))
	(define difference (cons (quote -) (map operands sql_info_formula)))
	/* Two checked INT operands subtract exactly in the native int64 carrier.
	The resulting value must still fit the declared SQL INT domain. */
	(tsql_convert_value_info (sql_info difference "BIGINT" nil) "INT" nil))))
/* Currency is a frontend declaration, absent from the common arithmetic type
taxonomy. Resolve its numeric precedence here, before a scalar consumer such as
ABS or a correlated projection compiles the difference. Native value tags do
not choose the result type. */
(define tsql_nullable_difference (close_procedure (lambda (left right)
	(if (or (nil? left) (nil? right)) nil (- left right)))))
(define tsql_compile_monetary_subtraction (lambda (infos) (begin
	(define types (map infos (lambda (info) (tsql_addition_type (sql_info_type info)))))
	(if (reduce types (lambda (valid type) (and valid
		(or (equal? type "NULL") (has? tsql_addition_text_types type)
			(has? tsql_addition_numeric_types type)))) true) true
		(error "unsupported SQL operands for -"))
	(define target (find tsql_addition_numeric_types (lambda (type) (has? types type)) nil))
	(define dimensions (if (equal? target "DECIMAL") (begin
		(define inputs (map infos (lambda (info)
			(if (equal? (sql_info_type info) "MONEY") '(19 4)
				(equal? (sql_info_type info) "SMALLMONEY") '(10 4)
				(if (empty_list? (sql_info_dimensions info)) '(18 0) (sql_info_dimensions info))))))
		(define scale (max (cadr (car inputs)) (cadr (cadr inputs))))
		(define precision (+ scale 1 (max (- (car (car inputs)) (cadr (car inputs)))
			(- (car (cadr inputs)) (cadr (cadr inputs))))))
		(list (min precision 38) (max 0 (- scale (max 0 (- precision 38)))))) '()))
	(define operands (map infos (lambda (info)
		(if (has? '("MONEY" "SMALLMONEY") target)
			(sql_type_lower_annotations '() (tsql_cast_expr
				(list 'sql_type_expr (sql_info_formula info) (sql_info_datatype info) nil) target '()))
			(equal? target "DECIMAL") (sql_info_formula info)
			(sql_info_formula (tsql_convert_value_info info target nil))))))
	(define formula (cons tsql_nullable_difference operands))
	(sql_info (if (has? '("MONEY" "SMALLMONEY") target)
		(list tsql_cast_money formula (equal? target "SMALLMONEY"))
		(equal? target "DECIMAL") (sql_type_lower_annotations '()
			(tsql_cast_expr (list 'sql_type_expr formula (list target dimensions) nil) target dimensions))
		(equal? target "REAL") (sql_type_lower_annotations '()
			(tsql_cast_expr (list 'sql_type_expr formula "FLOAT" nil) "REAL" '())) formula)
		(if (empty_list? dimensions) target (list target dimensions)) nil))))
(define tsql_compile_subtraction (lambda (infos)
	(if (find infos tsql_arithmetic_temporal? nil) (tsql_compile_temporal_day_arithmetic infos true)
		(and (equal? (count infos) 2)
			(or (and (has? tsql_addition_text_types (sql_info_type (car infos)))
				(equal? (tsql_addition_type (sql_info_type (cadr infos))) "INT"))
				(and (equal? (tsql_addition_type (sql_info_type (car infos))) "INT")
					(has? tsql_addition_text_types (sql_info_type (cadr infos))))))
		(tsql_compile_text_integer_subtraction infos)
		(find infos (lambda (info) (has? '("MONEY" "SMALLMONEY") (sql_info_type info))) nil)
		(tsql_compile_monetary_subtraction infos)
		(sql_call_info '- infos))))
(define tsql_sub_expr (lambda (left right)
	(list 'sql_compile_expr 'tsql_compile_subtraction left right)))

(define tsql_integer_divide (lambda (left right)
	(if (or (nil? left) (nil? right)) nil
		(equal? right 0) (tsql_raise_error 8134 "division by zero" 1)
		(and (equal? left (tsql_parse_integer "-9223372036854775808")) (equal? right -1))
		(error "integer division overflow")
		(intdiv left right))))
(define tsql_fractional_divide (lambda (left right)
	(if (or (nil? left) (nil? right)) nil
		(equal? right 0) (tsql_raise_error 8134 "division by zero" 1)
		(/ left right))))
(define tsql_compile_divide (lambda (infos) (begin
	(define types (map infos (lambda (info) (tsql_addition_type (sql_info_type info)))))
	(define active (filter types (lambda (type) (not (equal? type "NULL")))))
	(define target (if (empty_list? active) "INT"
		(find tsql_addition_numeric_types (lambda (type) (has? active type)) nil)))
	(if (or (nil? target) (equal? target "BIT")
		(not (reduce active (lambda (valid type) (and valid
			(or (has? tsql_addition_numeric_types type) (has? tsql_addition_text_types type)))) true)))
		(error "unsupported SQL operands for /") true)
	(define integer (has? '("BIGINT" "INT" "SMALLINT" "TINYINT") target))
	(define operands (map infos (lambda (info)
		(if (has? tsql_addition_text_types (tsql_addition_type (sql_info_type info)))
			(sql_type_formula '() (tsql_cast_expr (sql_info_formula info)
				(if integer target "FLOAT") '())) (sql_info_formula info)))))
	(define formula (cons (if integer (quote tsql_integer_divide) (quote tsql_fractional_divide)) operands))
	(sql_info (if integer (sql_type_formula '() (tsql_cast_expr formula target '())) formula) target nil))))
(define tsql_divide_expr (lambda (left right)
	(list (quote sql_compile_expr) (quote tsql_compile_divide) left right)))
/* Currency multiplication keeps the higher declared monetary type rather
than asking the common arithmetic compiler to infer one from native values.
https://learn.microsoft.com/en-us/sql/t-sql/language-elements/multiply-transact-sql */
(define tsql_nullable_product (close_procedure (lambda (left right)
	(if (or (nil? left) (nil? right)) nil (* left right)))))
(define tsql_compile_multiplication (lambda (infos) (begin
	(define types (map infos (lambda (info) (tsql_addition_type (sql_info_type info)))))
	(define target (find tsql_addition_numeric_types (lambda (type) (has? types type)) nil))
	(if (has? '("MONEY" "SMALLMONEY") target) (begin
		(if (reduce types (lambda (valid type) (and valid
			(or (equal? type "NULL")
				(and (not (equal? type "BIT")) (has? tsql_addition_numeric_types type))
				(has? tsql_addition_text_types type)))) true) true
			(error "unsupported SQL operands for *"))
		(define operands (map infos (lambda (info)
			(sql_type_formula '() (tsql_cast_expr
				(list 'sql_type_expr (sql_info_formula info) (sql_info_datatype info) (sql_info_collation info))
				target '())))))
		(sql_info (list tsql_cast_money (cons tsql_nullable_product operands)
			(equal? target "SMALLMONEY")) target nil))
		(sql_call_info '* infos)))))
(define tsql_fold_multiplicative (lambda (acc term) (match term
	'("divide" value) (tsql_divide_expr acc value)
	'("multiply" value) (list 'sql_compile_expr 'tsql_compile_multiplication acc value)
	_ (sql_fold_multiplicative_term acc term))))
(define tsql_fold_additive (lambda (acc term) (match term
	'("add" value) (tsql_add_expr acc value)
	'("sub" value) (tsql_sub_expr acc value)
	'(operation value) (if (has? '("and" "or" "xor" "left" "right") operation)
		(tsql_bit_expr operation (list acc value)) (sql_fold_additive_term acc term)))))
/* Implicit conversions consume declared contracts, including BIT's Boolean
carrier. Width/scale are not reconstructed from runtime values; unresolved
character dimensions retain the existing unbounded text conversion contract. */
(define tsql_convert_value_info (lambda (info type collation) (begin
	(define source (tsql_addition_type (sql_info_type info)))
	(define target (tsql_addition_type (sql_datatype_name type)))
	(define dimensions (match type '(_name dimensions) dimensions _ '()))
	(define formula (sql_info_formula info))
	(define formula (if (and (equal? source "BIT") (not (equal? target "BIT")))
		(list (quote match) formula (quote nil) nil (quote true) 1 (quote false) 0) formula))
	/* Monetary source declarations round when converted to integers; other
	numerics truncate. MATCH preserves NULL and evaluates the source once. */
	(define formula (if (and (has? '("MONEY" "SMALLMONEY") source)
		(has? '("BIGINT" "INT" "SMALLINT" "TINYINT") target))
		(list (quote match) formula (quote nil) nil (quote value) (list (quote round) (quote value))) formula))
	(define typed (list (quote sql_type_expr) formula
		(if (equal? source "BIT") "INT" (sql_info_datatype info)) (sql_info_collation info)))
	(define converted (if (or (equal? source "NULL")
		(and (equal? source target) (or (not (tsql_temporal_type? target))
			(expression_equal? dimensions (sql_info_dimensions info))))) formula
		(and (has? tsql_common_text_types target) (has? tsql_common_text_types source)) formula
		(and (has? tsql_addition_text_types target) (has? tsql_addition_numeric_types source))
		(list tsql_cast_text formula -1 (has? '("NVARCHAR" "NCHAR") target) "windows-1252")
		(and (has? tsql_addition_numeric_types target)
			(or (has? tsql_addition_numeric_types source) (has? tsql_addition_text_types source)
				/* The existing integer cast selects byte decoding from the SQL
				declaration; the same payload declared as text remains decimal text. */
				(and (has? '("BINARY" "VARBINARY") source) (tsql_integer_width target))))
		(if (has? '("DECIMAL" "MONEY" "SMALLMONEY") target)
			(list tsql_cast_float formula)
			(sql_type_lower_annotations '() (tsql_cast_expr typed target '())))
		(and (tsql_temporal_type? target)
			(or (tsql_temporal_numeric_target? target)
				(not (or (equal? source "BIT") (sql_numeric_type? (sql_info_type info))))))
		(sql_type_lower_annotations '() (tsql_cast_expr typed target dimensions))
		(error "unsupported implicit conversion from " source " to " target)))
	(sql_info converted type collation))))
/* Common temporal precision belongs to declared inputs. Legacy DATETIME
contributes three fractional digits; DATE/SMALLDATETIME have no fraction.
Character widths and native value carriers never infer calendar precision. */
(define tsql_common_temporal_datatype (lambda (target infos)
	(if (has? '("DATETIME2" "TIME") target)
		(list target (list (reduce infos (lambda (scale info) (begin
			(define type (sql_info_type info))
			(max scale (if (equal? type "DATETIME") 3
				(has? '("DATETIME2" "TIME") type)
				(if (empty_list? (sql_info_dimensions info)) 7 (car (sql_info_dimensions info))) 0)))) 0)))
		target)))
/* A common SQL contract for bounded syntax values. This is compile-time
metadata, shared by COALESCE and inline row constructors. It neither samples
runtime values nor stores a second relational representation. */
(define tsql_common_value_contract (lambda (infos null_type) (begin
	(define types (map infos (lambda (info) (tsql_addition_type (sql_info_type info)))))
	(define active (filter types (lambda (type) (not (equal? type "NULL")))))
	(define target (if (empty_list? active) (coalesceNil null_type (error "a declared non-NULL type is required"))
		(coalesceNil
			(find (merge (list tsql_temporal_types tsql_addition_numeric_types tsql_common_text_types))
				(lambda (type) (has? active type)) nil)
			(if (reduce active (lambda (same type) (and same (equal? type (car active)))) true)
				(car active) (error "unsupported common SQL type combination")))))
	(define collation (if (has? tsql_common_text_types target)
		(reduce infos (lambda (collation info)
			(sql_merge_collation collation (sql_info_collation info))) nil) nil))
	(sql_info nil (tsql_common_temporal_datatype target infos) collation))))
(define tsql_common_value_infos (lambda (infos null_type) (begin
	(define contract (tsql_common_value_contract infos null_type))
	(map infos (lambda (info) (tsql_convert_value_info info
		(sql_info_datatype contract) (sql_info_collation contract)))))))
/* UNION chooses column contracts from declarations, just like CASE and
inline values. Coercions become ordinary expressions before the shared planner;
no runtime value tag determines a SQL type. A nested set keeps its boundary. */
(define tsql_union_value_contract (lambda (infos) (begin
	(define contract (tsql_common_value_contract infos "INT"))
	(define type (sql_info_type contract))
	(define active (filter infos (lambda (info) (not (equal? (sql_info_type info) "NULL")))))
	/* Equal character types choose the longest declared input, including
	MAX. Unknown dimensions stay unknown; values never determine a width. */
	(define known (and (has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR") type)
		(not (empty_list? active))
		(reduce active (lambda (same info)
			(and same (equal? (sql_info_type info) type) (equal? (count (sql_info_dimensions info)) 1))) true)))
	(define datatype (if known
		(list type (list (reduce active (lambda (width info) (begin
			(define next (car (sql_info_dimensions info)))
			(if (or (equal? width -1) (equal? next -1)) -1 (max width next)))) 0))) (sql_info_datatype contract)))
	(sql_info nil datatype (sql_info_collation contract)))))

/* Set metadata selects a target declaration, not an expression compiler scope.
The original branch expression compiles once in its producing lexical scope;
only then can the existing implicit-conversion owner consume its triple. */
(define tsql_union_value_compiler (lambda (datatype collation) (lambda (infos)
	(tsql_convert_value_info (car infos) datatype collation))))

(define tsql_union_contracts (lambda (database branches policy planning_session tx) (begin
	(define templates (map branches (lambda (branch)
		(if (query_block? branch) branch
			(make_query_block database
				(list (list "__tsql_set_input" database branch false nil nil))
				(merge (map (queryplan_result_titles branch) (lambda (title)
					(list title (list (quote get_column) "__tsql_set_input" false title false)))))
				true '() nil '() nil nil '() '() '())))))
	/* Inspect expanded dependencies to infer contracts, but publish the
	original sources. The root expansion owns view/catalog replacement;
	returning its generated sources here would expand them a second time. */
	(define recipes (map templates (lambda (branch)
		(sql_bind_catalog_declarations (sql_expand_views branch policy planning_session tx) tx planning_session))))
	(define fields (map recipes (lambda (branch)
		(logical_relation_fields branch))))
	(define original_fields (mapIndex templates (lambda (index branch)
		/* Wildcards need the inspected output schema, but scalar expressions
		keep their original subqueries for the later root expansion. */
		(expand_query_block_fields (query_block_export_sources (nth recipes index)) (qb_fields branch)))))
	(define width (intdiv (count (car fields)) 2))
	(if (reduce fields (lambda (valid field) (and valid (equal? (count field) (* width 2)))) true)
		true (error "UNION branch column count mismatch"))
	(define contracts (map (produceN width) (lambda (column)
		(tsql_union_value_contract (mapIndex recipes (lambda (index branch)
			(sql_expr_info (qb_sources branch) (nth (nth fields index) (+ (* column 2) 1)))))))))
	(mapIndex recipes (lambda (index branch)
		(begin
			(define original (nth templates index))
			(make_query_block (qb_schema original) (qb_sources original)
				(merge (map (produceN width) (lambda (column) (begin
					(define contract (nth contracts column))
					(list (nth (nth fields index) (* column 2))
						(list 'sql_compile_expr (list 'tsql_union_value_compiler
							(list 'quote (sql_info_datatype contract)) (list 'quote (sql_info_collation contract)))
							(nth (nth original_fields index) (+ (* column 2) 1))))))))
				(qb_where original) (qb_group original) (qb_having original) (qb_order original)
				(qb_limit original) (qb_offset original) (qb_hidden original) (qb_stages original) (qb_facts original))))))))

/* NULL fallback has the same declared precedence as other T-SQL expressions.
The shared lazy coalesce operator remains responsible for evaluation; this
callback consumes SQL triples and leaves only converted formulas in the plan.
Temporal scale follows the common declaration; exact decimal precision is
separate frontend work. */
(define tsql_compile_coalesce (lambda (infos) (begin
	(if (< (count infos) 2) (error "COALESCE requires at least two arguments") true)
	(define converted (tsql_common_value_infos infos nil))
	(sql_info (cons (quote coalesceNil) (map converted sql_info_formula))
		(sql_info_datatype (car converted)) (sql_info_collation (car converted))))))
(define tsql_coalesce_expr (lambda (arguments)
	(cons (quote sql_compile_expr) (cons (quote tsql_compile_coalesce) arguments))))

/* ISNULL keeps its first declared contract, unlike COALESCE precedence.
A literal NULL takes the replacement contract; two literal NULLs use INT.
Only replacement evaluation is lazy, while compatibility is checked for every
row shape at compilation. Preserve declared temporal and character dimensions.
Only a known character width selects an explicit replacement conversion;
an unknown width never becomes CAST's default length of thirty. */
(define tsql_compile_isnull (lambda (infos) (begin
	(if (equal? (count infos) 2) true (error "ISNULL requires two arguments"))
	(define first (car infos))
	(define replacement (cadr infos))
	(define contract (if (equal? (sql_info_type first) "NULL") replacement first))
	(define type (if (equal? (sql_info_type contract) "NULL") "INT"
		(or (tsql_temporal_type? (sql_info_type contract)) (has? tsql_common_text_types (sql_info_type contract)))
		(sql_info_datatype contract) (sql_info_type contract)))
	(define converted (tsql_convert_value_info replacement type (sql_info_collation contract)))
	(define replacement_formula (if (and (has? tsql_addition_text_types (sql_info_type contract))
		(equal? (count (sql_info_dimensions contract)) 1))
		(sql_type_formula '() (tsql_cast_expr
			(list 'sql_type_expr (sql_info_formula converted) (sql_info_datatype converted) (sql_info_collation converted))
			(sql_info_type contract) (sql_info_dimensions contract)))
		(sql_info_formula converted)))
	(sql_info (list (quote coalesceNil) (sql_info_formula first) replacement_formula)
		type (sql_info_collation contract)))))
(define tsql_isnull_expr (lambda (arguments)
	(cons (quote sql_compile_expr) (cons (quote tsql_compile_isnull) arguments))))
/* NULLIF compares converted declared operands but returns its original first
contract. Scalar text equality must consume the selected collation explicitly;
there is no scan-bound operator to supply it for this conditional expression. */
(define tsql_compile_nullif (lambda (infos) (begin
	(if (equal? (count infos) 2) true (error "NULLIF requires two arguments"))
	(define first (car infos))
	(if (has? '("NULL" "any") (sql_info_type first))
		(error "NULLIF requires a declared first type") true)
	(define values (tsql_common_value_infos infos nil))
	(define first_value (car values))
	(define collation (if (and (has? tsql_addition_text_types (sql_info_type first_value))
		(nil? (sql_info_collation first_value))) "utf8mb4_general_ci" (sql_collation_name first_value)))
	(define comparison (list (quote sql_compare)
		(sql_info_formula first_value) (sql_info_formula (cadr values))
		(collate collation false) "equal??"))
	(sql_info (list (quote if) comparison nil (sql_info_formula first))
		(sql_info_datatype first) (sql_info_collation first)))))
(define tsql_nullif_expr (lambda (arguments)
	(cons (quote sql_compile_expr) (cons (quote tsql_compile_nullif) arguments))))

/* Conditional value contracts use the same declared precedence as COALESCE.
Conditions and result values remain visible to the shared binder/decorrelator;
the compile callback emits one ordinary variadic IF, without a runtime type
wrapper or a second evaluator. Exact decimal dimensions remain separate work. */
(define tsql_case_compiler (lambda (size) (lambda (infos) (begin
	(define predicates (slice infos 0 size))
	(if (reduce predicates (lambda (valid info)
		(and valid (equal? (sql_info_type info) "BOOLEAN"))) true)
		true (error "CASE/IIF requires a Boolean condition"))
	(define values (tsql_common_value_infos (slice infos size (count infos)) nil))
	(sql_info (cons (quote if) (append
		(merge (map (produceN size) (lambda (i)
			(list (sql_info_formula (nth predicates i)) (sql_info_formula (nth values i))))))
		(sql_info_formula (nth values size))))
		(sql_info_datatype (car values)) (sql_info_collation (car values)))))))
(define tsql_case_expr (lambda (conditions otherwise) (begin
	(define size (count conditions))
	(cons (quote sql_compile_expr) (cons (list (quote tsql_case_compiler) size)
		(append (merge (list (map conditions car) (map conditions cadr))) otherwise))))))
/* CHOOSE is a scalar selection, not an eager list of SQL values. Native MATCH
binds the converted ordinal once and evaluates only its chosen value. SQL
contracts are resolved for every value before this ordinary expression exists. */
(define tsql_compile_choose (lambda (infos) (begin
	(define index_info (car infos))
	(define index_type (tsql_addition_type (sql_info_type index_info)))
	(if (or (equal? index_type "NULL") (has? tsql_addition_numeric_types index_type)
		(has? tsql_addition_text_types index_type)) true
		(error "CHOOSE requires an implicitly convertible integer index"))
	(define branches (tsql_common_value_infos (cdr infos) nil))
	(define ordinal (sql_type_lower_annotations '()
		(tsql_cast_expr (if (has? '("MONEY" "SMALLMONEY") index_type)
			(list (quote round) (sql_info_formula index_info)) (sql_info_formula index_info)) "INT" '())))
	(sql_info (cons (quote match) (cons ordinal (merge (list
		(merge (map (produceN (count branches)) (lambda (i)
			(list (+ i 1) (sql_info_formula (nth branches i))))))
		(list (quote _) nil)))))
		(sql_info_datatype (car branches)) (sql_info_collation (car branches))))))
(define tsql_choose_expr (lambda (index values)
	(cons (quote sql_compile_expr) (cons (quote tsql_compile_choose)
		(cons index values)))))

/* Bind the shared logical tree for metadata only. No physical plan, compiled
row evaluator or native mutation is constructed by this analysis path. The
caller's policy remains explicit through every nested view expansion. */
(define tsql_describe_sources (lambda (node tx) (begin
	(if (query_block? node) (map (qb_sources node) (lambda (source)
		(if (and (source_is_base_table? source)
			(empty_list? (get_schema (source_schema source) (source_relation source) tx)))
			(error "description table does not exist: " (source_relation source)) true))) true)
	(tsql_map_executable_children node (lambda (child) (tsql_describe_sources child tx)))
	node)))
(define tsql_describe_query (lambda (query policy planning_session tx)
	(bind_query_names (tsql_describe_sources (tsql_bind_column_compilers
		(sql_expand_views query policy planning_session tx) (tsql_column_compiler planning_session)) tx) '())))
/* Explicit wire descriptors come from the existing SQL expression metadata,
never from the first row. Unknown expression types stay text in this first slice. */
/* Width/scale originate in declaration syntax, never in the first output row.
The shared expression compiler still owns the SQL type; these additional wire
dimensions are metadata only and are absent from the executable plan. */
(define tsql_result_dimensions (lambda (sources expression parameters) (match expression
	((symbol sql_compile_column) _compiler column) (tsql_result_dimensions sources column parameters)
	(cons (symbol sql_compile_expr) (cons ((symbol tsql_builtin_compiler) "CHAR") _arguments)) '(1)
	((symbol tsql_char) _value _money) '(1)
	(cons (symbol sql_compile_expr) (cons ((symbol tsql_datename_compiler) _unit) _arguments)) '(4000)
	((symbol sql_compile_expr) ((symbol tsql_text_compiler) length _national _fixed) _value) (list length)
	((symbol sql_type_expr) formula type _collation)
	(if (list? type) (nth type 1) (tsql_result_dimensions sources formula parameters))
	((symbol sql_compile_expr) ((symbol tsql_temporal_compiler) _target _style ((symbol quote) dimensions)) _value) dimensions
	((symbol sql_compile_expr) ((symbol tsql_dateadd_compiler) _unit _months _seconds) _amount value)
	(tsql_result_dimensions sources value parameters)
	((symbol sql_compile_expr) ((symbol tsql_text_compiler) length _national _fixed) _value) (list length)
	(cons (symbol sql_compile_expr) (cons ((symbol tsql_builtin_compiler) name) (cons value _arguments)))
	(if (has? '("UPPER" "LOWER" "LTRIM" "RTRIM" "TRIM" "LEFT" "RIGHT" "SUBSTRING" "SIGN") name)
		(tsql_result_dimensions sources value parameters)
		(equal? name "POWER") (begin
			(define dimensions (tsql_result_dimensions sources value parameters))
			(if (equal? (count (coalesceNil dimensions '())) 2) (list 38 (cadr dimensions)) nil)) nil)
	((symbol tsql_left) value _amount) (tsql_result_dimensions sources value parameters)
	((symbol tsql_right) value _amount) (tsql_result_dimensions sources value parameters)
	((symbol tsql_substring) value _start _amount _binary) (tsql_result_dimensions sources value parameters)
	((symbol tsql_upper) value) (tsql_result_dimensions sources value parameters)
	((symbol tsql_lower) value) (tsql_result_dimensions sources value parameters)
	((symbol tsql_ltrim) value) (tsql_result_dimensions sources value parameters)
	((symbol tsql_rtrim) value) (tsql_result_dimensions sources value parameters)
	((symbol tsql_trim) value) (tsql_result_dimensions sources value parameters)
	((symbol tsql_parsename) _value _piece) '(128)
	(cons (symbol sql_compile_expr) (cons (symbol tsql_compile_quotename) _arguments)) '(258)
	(cons (symbol sql_compile_expr) (cons (symbol tsql_compile_parsename) _arguments)) '(128)
	((symbol sql_compile_expr) ((symbol tsql_binary_compiler) length _fixed _style) _value) (list length)
	((symbol sql_compile_expr) ((symbol tsql_styled_text_compiler) _style _type ((symbol quote) dimensions)) _value)
	(if (empty_list? dimensions) '(30) dimensions)
	((symbol sql_compile_expr) ((symbol tsql_try_conversion_compiler) type ((symbol quote) dimensions) _style _bounded) _value)
	(tsql_decl_dimensions type (if (and (empty_list? dimensions)
		(has? '("VARCHAR" "NVARCHAR" "CHAR" "NCHAR" "BINARY" "VARBINARY") type)) '(30) dimensions))
	(cons (symbol sql_compile_expr) (cons compiler _arguments))
	(if (or (expression_equal? compiler tsql_compile_stuff) (expression_equal? compiler tsql_compile_replicate))
		(sql_info_dimensions (sql_expr_info sources expression))
		(tsql_result_dimensions sources (sql_type_formula sources expression) parameters))
	((symbol tsql_cast_decimal) _value precision scale) (list precision scale)
	((symbol tsql_quote_name) _value _delimiter) '(258)
	((symbol tsql_space) _amount) '(8000)
	((symbol tsql_str) _value width _places) (tsql_str_dimensions width)
	((symbol tsql_read_identity) _state) '(38 0)
	((symbol tsql_cast_binary) _value length _fixed) (list length)
	((symbol tsql_integer_to_binary) _value _width length _fixed) (list length)
	((symbol tsql_cast_text) _value length _national _encoding) (list length)
	((symbol tsql_cast_fixed_text) _value length _national _encoding) (list length)
	((symbol tsql_cast_money_text) _value _style length _national _fixed) (list length)
	((symbol tsql_cast_numeric_text) _value length _national _fixed _asterisk _boolean) (list length)
	((symbol hex2bin) hex) (if (string? hex) (list (intdiv (strlen hex) 2)) nil)
	((symbol session) key) (begin
		(define declaration (find parameters (lambda (item) (equal? key (concat "__tsql_parameter:" (car item)))) nil))
		(if declaration (cdr (tsql_effective_parameter_contract (cadr declaration)))
			(equal? key "schema") '(128)
			(has? '("__tsql_scope_identity" "__tsql_identity") key) '(38 0) nil))
	((symbol get_column) alias alias_ic column column_ic) (begin
		(define info (sql_get_column_info sources alias alias_ic column column_ic))
		(match (sql_info_formula info)
			((symbol get_column) bound_alias _bound_ic bound_column _column_ic) (begin
				(define src (source_for_alias sources (if (empty_list? sources) nil (source_alias (car sources))) bound_alias false))
				(if (nil? src) nil (source_is_base_table? src)
					(begin
						(define metadata (source_column_declaration src bound_column false))
						(if metadata (metadata "Dimensions") nil))
					(match (source_relation src)
						((symbol query-block) _schema inner_sources fields _where _group _having _order _limit _offset _hidden _stages _facts)
						(begin
							(define inner_expression ((expand_query_block_fields inner_sources fields) bound_column))
							(define inner_scope (merge (list inner_sources
								(if (source_lateral? src) (source_prefix_for_alias sources (source_alias src)) '()))))
							/* Derived outputs consume the same declared conversion formula as
							direct outputs, retaining the producer's permitted lateral scope.
							Row values never supply result dimensions. */
							(coalesceNil (tsql_result_dimensions inner_scope inner_expression parameters)
								(tsql_result_dimensions inner_scope
									(sql_info_formula (sql_expr_info inner_scope inner_expression)) parameters)))
						_ nil)))
			_ nil))
	/* Closed frontend helpers carry the same syntax-declared dimensions as
	their symbolic forms above. This is compiler inspection of known formula
	arguments, never inference from a runtime value or execution of the helper. */
	(cons helper arguments)
	(if (and (equal? (count arguments) 1)
		(has? (list tsql_upper tsql_lower tsql_ltrim tsql_rtrim tsql_trim) helper))
		(tsql_result_dimensions sources (car arguments) parameters)
		(or (expression_equal? helper tsql_cast_text) (expression_equal? helper tsql_cast_fixed_text)
			(expression_equal? helper tsql_cast_numeric_text)) (list (nth arguments 1))
		(expression_equal? helper tsql_cast_money_text) (list (nth arguments 2))
		(expression_equal? helper tsql_cast_decimal) (list (nth arguments 1) (nth arguments 2))
		nil)
	_ nil)))
/* Provenance is a declaration fact. Computed values never inherit a base
column's editability, and an outer source remains nullable after binding. */
/* Before shared name binding, only an operand's head carries source_outer.
Its logical boundary records every member that receives the same NULL row.
Read that declaration fact without modifying join flags or inspecting values. */
(define tsql_result_source_nullable? (lambda (source facts)
	(or (source_outer? source)
		(not (nil? (find (merge (list
			(qassoc_get facts 'nullable_join_scopes '())
			(qassoc_get facts 'join_operand_scopes '())))
			(lambda (unit) (has? (cons (car unit) (cadr unit)) (source_alias source))) nil))))))
(define tsql_result_origin (lambda (sources facts info follow_derived) (match (sql_info_formula info)
	((symbol get_column) alias alias_ic name name_ic) (begin
		(define source (source_for_alias sources (if (empty_list? sources) nil (source_alias (car sources))) alias alias_ic))
		(if (nil? source) nil
			(source_is_base_table? source) (begin
				(define column (source_column_declaration source name name_ic))
				(if column (list source (if (tsql_result_source_nullable? source facts)
					(set_assoc column "Null" true) column)) nil))
			follow_derived (match (source_relation source)
				((symbol query-block) _schema inner_sources fields _where _group _having _order _limit _offset _hidden _stages inner_facts)
				(begin
					(define expression ((expand_query_block_fields inner_sources fields) name))
					(define origin (tsql_result_origin inner_sources inner_facts (sql_expr_info inner_sources expression) true))
					(if (and origin (tsql_result_source_nullable? source facts))
						(list (car origin) (set_assoc (cadr origin) "Null" true)) origin))
				_ nil)
			nil))
	_ nil)))
/* Both client adapters consume the same expression triples. Following a view's
native declaration is permitted only for its column catalog, never to infer
editability or keys for a derived result. */
(define tsql_result_entries (lambda (query follow_derived) (match query
	((symbol query-block) _schema sources fields _where _group _having _order _limit _offset _hidden _stages facts)
	(extract_assoc (logical_relation_fields query) (lambda (name expression) (begin
		(define info (sql_expr_info sources expression))
		(list name expression info (tsql_result_origin sources facts info follow_derived)))))
	_ (error "unsupported result projection contract"))))
(define tsql_result_flags (lambda (origin editable) (if (nil? origin) '(1 nil) (begin
	(define source (car origin))
	(define column (cadr origin))
	(define identity (equal? (column "Extra") "auto_increment"))
	(define version (equal?? (column "RawType") "ROWVERSION"))
	(list (+ (if (or (source_outer? source) (column "Null")) 1 0)
		(if (and editable (not (or identity version))) 4 0) (if identity 16 0))
		(if version 80 0))))))
/* Native text families have no storage width cap. Describe existing cross-
frontend declarations through the T-SQL large-text contract without changing
RawType, admitting new DDL spellings or widening explicit VARCHAR(n). */
(define tsql_metadata_base_type (lambda (type) (begin
	(define type (toUpper type))
	(if (has? '("TINYTEXT" "MEDIUMTEXT" "LONGTEXT") type) "TEXT" type))))
/* Wire shapes come only from a declared SQL contract, including when no
row or supplied parameter value exists. */
(define tsql_wire_contract (lambda (type dimensions) (begin
	(define type (tsql_metadata_base_type type))
	(define numeric (and (has? '("DECIMAL" "NUMERIC") type) (>= (count dimensions) 2)))
	(define money (has? '("MONEY" "SMALLMONEY") type))
	(define kind (if (equal? type "SQL_VARIANT") 98
		(equal? type "UNIQUEIDENTIFIER") 36
		(has? '("INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT") type) 38
		/* SqlClient recognizes the generated timestamp user type only on the
		fixed binary token. VARBINARY would expose a writable bulk field. */
		money 110 numeric 108 (equal? type "ROWVERSION") 173 (tsql_binary_type? type) 165
		(has? '("VARCHAR" "TEXT") type) 167 (equal? type "CHAR") 175
		(has? '("FLOAT" "DOUBLE" "REAL" "DECIMAL" "NUMERIC") type) 109
		(has? '("BOOLEAN" "BOOL" "BIT") type) 104
		(equal? type "DATE") 40 (equal? type "TIME") 41 (equal? type "DATETIME2") 42
		(has? '("DATETIME" "SMALLDATETIME") type) 111 231))
	(define precision (if money (if (equal? type "SMALLMONEY") 10 19) numeric (car dimensions) 0))
	(define scale (if money 4 numeric (cadr dimensions)
		(has? '("TIME" "DATETIME2") type) (if (empty_list? dimensions) 7 (car dimensions)) 0))
	(define size (match kind
		36 16
		98 8016
		40 3
		41 (if (<= scale 2) 3 (<= scale 4) 4 5)
		42 (+ 3 (if (<= scale 2) 3 (<= scale 4) 4 5))
		111 (if (equal? type "SMALLDATETIME") 4 8)
		110 (if (equal? type "SMALLMONEY") 4 8)
		109 (if (has? '("REAL" "FLOAT" "DOUBLE") type)
			(tsql_declared_byte_length (list "RawType" type "Dimensions" dimensions)) 8)
		/* Framework SqlBulkCopy writes four coefficient words, using this
		advertised buffer for its length prefix. SQL storage length is separate. */
		108 17
		173 8
		165 (if (and (not (empty_list? dimensions)) (> (car dimensions) 0)) (car dimensions) 65535)
		/* Legacy LOB declarations keep their native SQL type. Transport them
		through the existing MAX/PLP family, never a guessed bounded width.
		TEXT is ANSI, NTEXT is national text; IMAGE already uses binary MAX. */
		167 (if (equal? type "TEXT") 65535 (empty_list? dimensions) 8000
			(equal? (car dimensions) -1) 65535 (max 1 (car dimensions)))
		175 (if (empty_list? dimensions) 1 (max 1 (car dimensions)))
		231 (if (equal? type "NTEXT") 65535
			(and (has? '("NVARCHAR" "NCHAR") type) (not (empty_list? dimensions)))
			(if (equal? (car dimensions) -1) 65535 (* 2 (car dimensions))) 8000)
		104 1
		_ (match type "TINYINT" 1 "SMALLINT" 2 "INT" 4 "INTEGER" 4 _ 8)))
	(merge (list "kind" kind "size" size "precision" precision "scale" scale)
		(if (has? '(167 175 231) kind) (list "collation" (hex2bin "0904d00034")) '())))))
/* Result columns and collation metadata share the same transport encoding.
Only compile-time SQL declarations choose the five-byte collation. */
(define tsql_wire_info_contract (lambda (info dimensions) (begin
	(define wire (tsql_wire_contract (sql_info_type info) dimensions))
	(if (and (has? '("CHAR" "VARCHAR" "TEXT") (tsql_metadata_base_type (sql_info_type info)))
		(equal? (tsql_character_encoding info) "utf-8"))
		(map_assoc wire (lambda (key value)
			(if (equal? key "collation") (hex2bin "0904000400") value))) wire))))
(define tsql_result_columns (lambda (query parameters) (match query
	((symbol query-block) _schema sources fields _where _group _having _order _limit _offset _hidden _stages _facts)
	(begin
		(define entries (tsql_result_entries query false))
		(define origins (map entries (lambda (entry) (nth entry 3))))
		(define keys (if (and (equal? (count sources) 1) (source_is_base_table? (car sources)))
			(source_primary_key_columns (car sources)) '()))
		(define direct_names (map (filter origins (lambda (origin) (not (nil? origin))))
			(lambda (origin) ((cadr origin) "Field"))))
		(define editable (and (equal? (count sources) 1) (not (empty_list? keys))
			(empty_list? (coalesceNil _group '())) (empty_list? (coalesceNil _stages '()))
			(not (source_outer? (car sources)))
			(reduce keys (lambda (valid key) (and valid (has? direct_names key))) true)))
		(map entries (lambda (entry) (begin
			(define name (car entry))
			(define expression (nth entry 1))
			(define info (nth entry 2))
			(define origin (nth entry 3))
			(define flags (tsql_result_flags origin editable))
			(define type (sql_info_type info))
			(define dimensions (if (empty_list? (sql_info_dimensions info))
				(coalesceNil (tsql_result_dimensions sources expression parameters)
					(tsql_result_dimensions sources (sql_info_formula info) parameters) '())
				(sql_info_dimensions info)))
			(define wire (tsql_wire_info_contract info dimensions))
			(merge wire (list "name" name "flags" (car flags) "user_type" (coalesceNil (cadr flags) 0) "sql_type" type
				"sql_datatype" (if (empty_list? dimensions) type (list type dimensions))
				"sql_collation" (sql_info_collation info))
				(if origin (list "source" (list (source_schema (car origin)) "dbo"
					(source_relation (car origin)) ((cadr origin) "Field"))) '()))))))
	((symbol union-block) _mode branches _order _limit _offset _facts)
	(map (tsql_result_columns (car branches) parameters) (lambda (column)
		(set_assoc (set_assoc (set_assoc column "flags" 1) "source" nil) "user_type" 0)))
	_ (error "unsupported T-SQL result metadata shape"))))
/* Format only the external result edge. Inner projections, ordering and native
storage keep their Unix values. The declaration selects each formatter while
compiling; the emitted row callback contains no SQL type tags or dispatch. */
(define tsql_result_output_plan (lambda (plan descriptors)
	(if (reduce descriptors (lambda (found column)
		(or found (equal? (column "sql_type") "SQL_VARIANT") (tsql_temporal_type? (column "sql_type")))) false)
		(list (quote begin)
			(list (quote define) (quote __tsql_raw_resultrow) (quote resultrow))
			(list (quote define) (quote resultrow)
				(list (quote if) (list (quote session) "wire_metadata") (quote __tsql_raw_resultrow)
					(list (quote lambda) (list (quote row))
						(list (quote __tsql_raw_resultrow)
							(cons (quote list) (merge (map (produceN (count descriptors)) (lambda (index) (begin
								(define column (nth descriptors index))
								(define value (list (quote nth) (quote row) (+ 1 (* index 2))))
								(define pattern (match (column "sql_type")
									"DATE" "%Y-%m-%d" "TIME" "%H:%i:%s"
									"DATETIME" "%Y-%m-%d %H:%i:%s" "DATETIME2" "%Y-%m-%d %H:%i:%s"
									"SMALLDATETIME" "%Y-%m-%d %H:%i:%s" _ nil))
								/* Stored TIME may be plain clock text. Choose its existing
								anchor conversion from the declared type at compile time. */
								(define calendar_value (if (equal? (column "sql_type") "TIME")
									(tsql_calendar_text_formula value true) value))
								(list (column "name") (if (equal? (column "sql_type") "SQL_VARIANT")
									(list tsql_variant_scalar value) pattern
									(tsql_temporal_style_expr
										(sql_info calendar_value (column "sql_datatype") (column "sql_collation")) 121 true) value)))))))))))
			plan)
		plan)))

(define tsql_collation_compiler (lambda (name) (lambda (infos) (begin
	(define info (car infos))
	(sql_info (sql_info_formula info) (sql_info_datatype info) (list name 0))))))

/* ROUND owns SQL length, truncation and return-width rules at this frontend
compile boundary. Integer inputs never pass through floating-point scaling.
The existing DECIMAL(p,s) cases still need the full declaration contract. */
(define tsql_round_int64_minimum (tsql_parse_integer "-9223372036854775808"))
(define tsql_round_int64_maximum (tsql_parse_integer "9223372036854775807"))
(define tsql_round_integer_powers '(1 10 100 1000 10000 100000 1000000 10000000 100000000
	1000000000 10000000000 100000000000 1000000000000 10000000000000 100000000000000
	1000000000000000 10000000000000000 100000000000000000 1000000000000000000))
(define tsql_round_integer_factor (lambda (places)
	(if (or (nil? places) (>= places 0) (<= places -19)) nil
		(nth tsql_round_integer_powers (- 0 places)))))
(define tsql_round_integer (lambda (value places mode factor minimum maximum)
	(if (or (nil? value) (nil? places) (nil? mode)) nil
		(>= places 0) value
		(< places -19) 0
		(equal? places -19) (if (or (not (equal? mode 0))
			(and (> value -5000000000000000000) (< value 5000000000000000000))) 0
			(tsql_raise_error 8115 "integer rounding overflow" 1))
		(begin
			(define quotient (intdiv value factor))
			(define remainder (mod value factor))
			(define half (intdiv factor 2))
			(define rounded (+ quotient (if (not (equal? mode 0)) 0
				(>= remainder half) 1 (<= remainder (- 0 half)) -1 0)))
			/* Check the rounded quotient before multiplication, including MIN. */
			(if (or (< rounded (intdiv minimum factor)) (> rounded (intdiv maximum factor)))
				(tsql_raise_error 8115 "integer rounding overflow" 1) (* rounded factor))))))
/* Module-owned immutable powers cover every finite decimal unit used here.
Dynamic lengths only index this table; constant lengths become plan literals. */
(define tsql_round_real_factors (map (produceN 632) (lambda (index)
	(simplify (concat "1e" (- 308 index))))))
(define tsql_round_real_factor (close_procedure ((lambda (tsql_round_real_factors) (lambda (places)
	(if (or (nil? places) (< places -308) (> places 323)) nil
		(nth tsql_round_real_factors (+ places 308))))) tsql_round_real_factors)))
(define tsql_round_real (close_procedure ((lambda (tsql_float_max tsql_raise_error) (lambda (value places mode factor)
	(if (or (nil? value) (nil? places) (nil? mode)) nil
		(> places 323) value
		(< places -308) 0
		(equal? value 0) 0
		/* Finer-than-representation rounding must not overflow while scaling. */
		(and (< factor 1) (or (> value (* tsql_float_max factor))
			(< value (- 0 (* tsql_float_max factor))))) value
		(begin
			(define scaled (/ value factor))
			(define rounded (if (equal? mode 0) (round scaled)
				(< scaled 0) (ceil scaled) (floor scaled)))
			(define result (* rounded factor))
			(if (and (>= result (- 0 tsql_float_max)) (<= result tsql_float_max)) result
				(tsql_raise_error 8115 "floating rounding overflow" 1)))))) tsql_float_max tsql_raise_error)))
(define tsql_round_money (lambda (value places mode factor)
	(if (or (nil? value) (nil? places) (nil? mode)) nil
		(begin
			(define coefficient (tsql_money_coefficient value false))
			(define rounded (tsql_round_integer coefficient (- places 4) mode factor
				tsql_round_int64_minimum tsql_round_int64_maximum))
			(tsql_cast_money (/ rounded 10000) false)))))
(define tsql_compile_round (lambda (infos) (begin
	(if (or (equal? (count infos) 2) (equal? (count infos) 3)) true
		(error "ROUND requires value, length and optional function"))
	(define input (car infos))
	(define type (tsql_addition_type (sql_info_type input)))
	/* Preserve the existing limited decimal path until precision and scale
	are carried by the common compile-time declaration contract. */
	(if (equal? type "DECIMAL") (sql_call_info (sql_builtins "ROUND") infos)
		(begin
			(define integer (has? '("NULL" "TINYINT" "SMALLINT" "INT" "BIGINT") type))
			(define monetary (has? '("MONEY" "SMALLMONEY") type))
			(if (or integer monetary (has? '("FLOAT" "REAL") type)) true
				(error "ROUND requires a supported declared numeric input"))
			(define places (tsql_convert_value_info (cadr infos) "INT" nil))
			(define mode (if (equal? (count infos) 3)
				(tsql_convert_value_info (nth infos 2) "INT" nil) (sql_info 0 "INT" nil)))
			(define folded (try (lambda () (list true (tryfold (sql_info_formula places))))
				(lambda (_cause) (list false nil))))
			(define known (car folded))
			(define length (if known (cadr folded) (quote round_length)))
			(define adjusted_length (if monetary (if known (if (nil? length) nil (- length 4))
				(list (quote -) length 4)) length))
			(define factor_helper (if (or integer monetary) (quote tsql_round_integer_factor) (quote tsql_round_real_factor)))
			(define factor (if known
				(if (or integer monetary) (tsql_round_integer_factor adjusted_length) (tsql_round_real_factor adjusted_length))
				(list factor_helper adjusted_length)))
			(define body (if integer
				(list (quote tsql_round_integer) (quote round_value) (quote round_length) (quote round_function) factor
					(if (equal? type "BIGINT") tsql_round_int64_minimum -2147483648)
					(if (equal? type "BIGINT") tsql_round_int64_maximum 2147483647))
				(list (if monetary (quote tsql_round_money) (quote tsql_round_real))
					(quote round_value) (quote round_length) (quote round_function) factor)))
			(sql_info (list (list (quote lambda) (list (quote round_value) (quote round_length) (quote round_function)) body)
				(sql_info_formula input) (sql_info_formula places) (sql_info_formula mode))
				(if integer (if (equal? type "BIGINT") "BIGINT" "INT") monetary "MONEY" "FLOAT") nil))))))

/* Integer ABS keeps native precision and checks the declared result range.
Other scalar return families are selected by the frontend compiler below. */
(define tsql_integer_absolute (lambda (value minimum)
	(if (nil? value) nil (equal? value minimum)
		(tsql_raise_error 8115 "integer absolute overflow" 1)
		(< value 0) (- 0 value) value)))
(define tsql_compile_numeric_scalar (lambda (name infos) (begin
	(if (equal? (count infos) 1) true (error name " requires one argument"))
	(define input (car infos))
	(define type (tsql_addition_type (sql_info_type input)))
	/* DECIMAL result precision and scale require the pending declaration
	contract. Preserve that existing path without claiming its completion. */
	(if (equal? type "DECIMAL") (sql_call_info (sql_builtins name) infos)
		(begin
			(define integer (has? '("NULL" "TINYINT" "SMALLINT" "INT" "BIGINT") type))
			(define money (has? '("MONEY" "SMALLMONEY") type))
			(define floating (or (has? '("FLOAT" "REAL" "BIT") type)
				(has? tsql_addition_text_types type)))
			(if (or integer money floating) true
				(error name " requires a declared numeric input"))
			(define value (sql_info_formula (if floating
				(tsql_convert_value_info input "FLOAT" nil) input)))
			/* MATCH evaluates the operand once and protects generic floor/ceil
			from NULL. SQL types never depend on the resulting value carrier. */
			(define formula (if integer
				(if (equal? name "ABS") (list (quote tsql_integer_absolute) value
					(if (equal? type "BIGINT") (tsql_parse_integer "-9223372036854775808") -2147483648)) value)
				(list (quote match) value (quote nil) nil (quote scalar_value)
					(if (equal? name "ABS")
						(list (quote if) (list (quote <) (quote scalar_value) 0)
							(list (quote -) 0 (quote scalar_value)) (quote scalar_value))
						(list (if (equal? name "FLOOR") (quote floor) (quote ceil)) (quote scalar_value))))))
			(sql_info (if money (list tsql_cast_money formula false) formula)
				(if integer (if (equal? type "BIGINT") "BIGINT" "INT") money "MONEY" "FLOAT") nil))))))

/* These functions consume FLOAT contracts. Convert once before arithmetic:
an integer carrier must not overflow before the declared floating operation. */
(define tsql_square (lambda (value) (match value
	nil nil
	operand (begin
		(define number (float operand))
		(float (tsql_numeric_input (* number number)))))))
(define tsql_square_root (lambda (value)
	(if (nil? value) nil
		(< value 0) (error "SQRT requires a nonnegative floating operand")
		(sqrt (float value)))))
(define tsql_compile_floating_scalar (lambda (name infos) (begin
	(if (equal? (count infos) 1) true (error name " requires one argument"))
	(define input (tsql_convert_value_info (car infos) "FLOAT" nil))
	(sql_info (list (if (equal? name "SQUARE") (quote tsql_square) (quote tsql_square_root))
		(sql_info_formula input)) "FLOAT" nil))))

/* Character normalization keeps declaration metadata at compilation. Each
ordinary scalar helper consumes its operand once; SQL NULL is not text.
Default trimming removes only ASCII space, preserving other whitespace. */
(define tsql_upper (lambda (value) (if (nil? value) nil (toUpper value))))
(define tsql_lower (lambda (value) (if (nil? value) nil (toLower value))))
(define tsql_ltrim (lambda (value) (if (nil? value) nil (regexp_replace value "^ +" ""))))
(define tsql_rtrim (lambda (value) (if (nil? value) nil (regexp_replace value " +$" ""))))
(define tsql_trim (close_procedure ((lambda (left right)
	(lambda (value) (right (left value)))) tsql_ltrim tsql_rtrim)))
/* Equality pads character operands only with U+0020, equivalently removing
that suffix from both. Arguments are ordinary scalar inputs evaluated once.
Collation is selected from SQL declarations at compilation, never runtime tags.
This is not a range, LIKE, native UNIQUE or grouping policy. */
(define tsql_padded_text_equal (close_procedure ((lambda (trim_right)
	(lambda (left right collation)
		(equal_collate (trim_right left) (trim_right right) collation))) tsql_rtrim)))
(define tsql_padded_text_in (close_procedure ((lambda (compare)
	(lambda (values probe collation)
		(if (nil? probe) nil
			(reduce values (lambda (result value)
				(if (identical? result true) true (begin
					(define equal (compare probe value collation))
					(if (identical? equal true) true
						(or (nil? result) (nil? equal)) nil false)))) false)))) tsql_padded_text_equal)))
/* SQL positions and conversions are frontend contracts. Native prefix operators
supply complete UTF16 boundaries; byte offsets are used only after validation.
A boundary inside a surrogate pair is explicitly unsupported by this carrier.
No character list, decoder, relation or runtime SQL type is constructed here. */
(define tsql_slice_prefix (lambda (text units binary)
	(if binary (substr text 0 (min units (strlen text)))
		(begin
			(define prefix (utf16_prefix text units))
			(if (and (< (strlen prefix) (strlen text)) (< (utf16_length prefix) units))
				(error "string slice boundary inside a surrogate pair is unsupported") prefix)))))
(define tsql_left (lambda (value amount)
	(if (or (nil? value) (nil? amount)) nil
		(< amount 0) (error "LEFT length must be nonnegative")
		(tsql_slice_prefix (string value) amount false))))
(define tsql_right (lambda (value amount)
	(if (or (nil? value) (nil? amount)) nil
		(< amount 0) (error "RIGHT length must be nonnegative")
		(equal? amount 0) ""
		(begin
			(define text (string value))
			(define size (utf16_length text))
			(if (>= amount size) text
				(substr text (strlen (tsql_slice_prefix text (- size amount) false))))))))
(define tsql_substring (lambda (value start amount binary)
	(if (or (nil? value) (nil? start) (nil? amount)) nil
		(< amount 0) (error "SUBSTRING length must be nonnegative")
		(equal? amount 0) ""
		(begin
			/* Negative plus nonnegative cannot overflow the signed carrier.
			Do not subtract a possibly minimal start or add two large endpoints. */
			(define extent (if (< start 1) (max 0 (+ start (- amount 1))) amount))
			(if (equal? extent 0) ""
				(begin
					(define text (string value))
					(define prefix (tsql_slice_prefix text (if (> start 1) (- start 1) 0) binary))
					(tsql_slice_prefix (substr text (strlen prefix)) extent binary)))))))
(define tsql_compile_slicing (lambda (name infos) (begin
	(if (equal? (count infos) (if (equal? name "SUBSTRING") 3 2)) true
		(error name " has an invalid argument count"))
	(define input (car infos))
	(define type (sql_info_type input))
	(define binary (and (equal? name "SUBSTRING") (has? '("BINARY" "VARBINARY" "IMAGE") type)))
	(if (and (equal? name "SUBSTRING")
		(not (has? '("NULL" "CHAR" "VARCHAR" "NCHAR" "NVARCHAR" "TEXT" "NTEXT" "SYSNAME" "BINARY" "VARBINARY" "IMAGE") type)))
		(error "SUBSTRING requires declared character or binary data") true)
	(if (and (not (equal? name "SUBSTRING")) (has? '("TEXT" "NTEXT" "IMAGE") type))
		(error name " requires a convertible character declaration") true)
	(define output (if binary "VARBINARY"
		(has? '("NCHAR" "NVARCHAR" "NTEXT" "SYSNAME") type) "NVARCHAR" "VARCHAR"))
	(define converted (if (or binary (has? '("NULL" "CHAR" "VARCHAR" "NCHAR" "NVARCHAR" "TEXT" "NTEXT" "SYSNAME") type))
		input ((tsql_text_compiler -1 false false) (list input))))
	(define positions (map (cdr infos) (lambda (info)
		(sql_info_formula (tsql_convert_value_info info "BIGINT" nil)))))
	(define arguments (cons (sql_info_formula converted) positions))
	(sql_info (if (equal? name "SUBSTRING")
		(cons (quote tsql_substring) (append arguments binary))
		(cons (if (equal? name "LEFT") (quote tsql_left) (quote tsql_right)) arguments))
		(if (empty_list? (sql_info_dimensions converted)) output (list output (sql_info_dimensions converted)))
		(if binary nil (sql_info_collation converted))))))

(define tsql_compile_string_normalization (lambda (name infos) (begin
	(if (equal? (count infos) 1) true (error name " requires one argument"))
	(define input (car infos))
	(define type (sql_info_type input))
	(define output (if (has? '("NVARCHAR" "NCHAR" "NTEXT" "SYSNAME") type) "NVARCHAR" "VARCHAR"))
	/* Already-character formulas retain their original declaration nodes for
	result dimensions. Do not insert an unbounded cast around a known width. */
	(define converted (if (or (has? tsql_addition_text_types type)
		(has? '("NULL" "TEXT" "NTEXT" "SYSNAME" "BINARY" "VARBINARY") type)) input
		(tsql_convert_value_info input output (sql_info_collation input))))
	(define helper (match name
		"UPPER" tsql_upper "LOWER" tsql_lower
		"LTRIM" tsql_ltrim "RTRIM" tsql_rtrim "TRIM" tsql_trim))
	(sql_info (list helper (sql_info_formula converted))
		(if (empty_list? (sql_info_dimensions input)) output (list output (sql_info_dimensions input)))
		(sql_info_collation input)))))

/* Byte/code-point interpretation is selected from SQL declarations here. The
runtime primitives know only explicit encodings and raw byte coordinates. */
(define tsql_char (lambda (value money) (if (nil? value) nil (begin
	(define code (tsql_cast_integer (if money (round value) value) -2147483648 2147483647))
	(single_byte_decode (byte_char code) "windows-1252")))))
(define tsql_compile_char (lambda (infos) (begin
	(if (equal? (count infos) 1) true (error "CHAR requires one argument"))
	(define input (car infos))
	(define type (tsql_addition_type (sql_info_type input)))
	(if (has? '("NULL" "BIT" "TINYINT" "SMALLINT" "INT" "BIGINT" "REAL" "FLOAT" "DECIMAL" "MONEY" "SMALLMONEY" "CHAR" "VARCHAR" "NCHAR" "NVARCHAR") type)
		true (error "CHAR requires an implicitly convertible integer expression"))
	(sql_info (list tsql_char (sql_info_formula input) (has? '("MONEY" "SMALLMONEY") type))
		(list "CHAR" '(1)) '("utf8mb4_general_ci" 4)))))
(define tsql_ascii (lambda (value encoding binary)
	(if (nil? value) nil
		(byte_value (if (or binary (equal? encoding "utf-8")) value
			(tsql_encode_character value encoding 1)) 0))))
(define tsql_compile_ascii (lambda (infos) (begin
	(if (equal? (count infos) 1) true (error "ASCII requires one argument"))
	(define input (car infos))
	(define binary (tsql_binary_type? (sql_info_type input)))
	(if (has? '("TEXT" "NTEXT" "IMAGE") (sql_info_type input))
		(error "ASCII requires a convertible character or binary declaration") true)
	(define converted (if (or binary (has? '("NULL" "CHAR" "VARCHAR" "NCHAR" "NVARCHAR") (sql_info_type input))) input
		((tsql_text_compiler -1 false false) infos)))
	(sql_info (list tsql_ascii (sql_info_formula converted)
		(if binary "windows-1252" (tsql_character_encoding converted)) binary) "INT" nil))))
(define tsql_fixed_datalength (lambda (value width) (if (nil? value) nil width)))
(define tsql_encoded_datalength (lambda (value encoding)
	(if (nil? value) nil (strlen (tsql_encode_character value encoding -1)))))
(define tsql_national_datalength (lambda (value) (if (nil? value) nil (* 2 (utf16_length value)))))
(define tsql_binary_datalength (lambda (value) (if (nil? value) nil (strlen value))))
(define tsql_compile_datalength (lambda (infos) (begin
	(if (equal? (count infos) 1) true (error "DATALENGTH requires one argument"))
	(define input (car infos))
	(define type (sql_info_type input))
	(define dimensions (sql_info_dimensions input))
	(define formula (sql_info_formula input))
	(if (and (has? '("DECIMAL" "NUMERIC") type) (empty_list? dimensions))
		(error "DATALENGTH requires a declared precision for this decimal expression") true)
	(define result_type (if (and (has? '("VARCHAR" "NVARCHAR" "VARBINARY") type) (equal? dimensions '(-1))) "BIGINT" "INT"))
	(define expression (if (has? '("VARCHAR" "TEXT") type)
		(list tsql_encoded_datalength formula (tsql_character_encoding input))
		(has? '("NVARCHAR" "NTEXT") type) (list tsql_national_datalength formula)
		(has? '("VARBINARY" "IMAGE") type) (list tsql_binary_datalength formula)
		(equal? type "NULL") (list tsql_fixed_datalength formula 0)
		(begin
			(define width (tsql_declared_byte_length (list "RawType" type "Dimensions" dimensions)))
			(if width (list tsql_fixed_datalength formula width)
				(error "DATALENGTH requires a supported SQL declaration; got " type)))))
	(sql_info expression result_type nil))))
(define tsql_len (lambda (value unit encoding) (if (nil? value) nil (begin
	(define text (tsql_rtrim value))
	(match unit
		"utf16" (utf16_length text)
		"rune" (text_length text "rune")
		_ (strlen (if encoding (tsql_encode_character text encoding -1) text)))))))
(define tsql_compile_len (lambda (infos) (begin
	(if (equal? (count infos) 1) true (error "LEN requires one argument"))
	(define input (car infos))
	(define type (sql_info_type input))
	(if (has? '("TEXT" "NTEXT" "IMAGE") type) (error "LEN requires a convertible character or binary declaration") true)
	(define binary (tsql_binary_type? type))
	(define national (has? '("NCHAR" "NVARCHAR") type))
	(define converted (if (or binary national (has? '("CHAR" "VARCHAR" "NULL") type)) input
		((tsql_text_compiler -1 false false) infos)))
	(define encoding (if (or national binary) nil (tsql_character_encoding converted)))
	(define collation (sql_info_collation input))
	(define supplementary (if national
		(if (nil? collation) false (regexp_test (car collation) "(?i)_sc(_|$)")) false))
	(define unit (if (or (equal? encoding "utf-8") supplementary) "rune" national "utf16" "byte"))
	(sql_info (list tsql_len (sql_info_formula converted) unit encoding)
		(if (and (has? '("VARCHAR" "NVARCHAR" "VARBINARY") type) (equal? (sql_info_dimensions input) '(-1))) "BIGINT" "INT") nil))))

/* Runtime operator factories capture every frontend helper as an actual
procedure value. Workers execute the emitted SCM without resolving dialect
helper names in their global environment.
Editing coordinates and output capacity are separate SQL contracts.
Collations can count supplementary scalars once while NVARCHAR capacity still
counts UTF16 units. Positions are validated before any allocation or slicing. */
(define tsql_edit_prefix (lambda (text amount unit) (begin
	(define prefix (text_prefix text amount unit))
	(if (and (< (strlen prefix) (strlen text)) (< (text_length prefix unit) amount))
		(error "text edit boundary inside a surrogate pair is unsupported") prefix))))
(define tsql_stuff_operator (lambda (take_prefix raise_error) (lambda (value start amount replacement unit size_unit limit)
	(if (or (nil? value) (nil? start) (nil? amount)) nil
		(or (<= start 0) (< amount 0) (> start (text_length value unit))) nil
		(begin
			(define prefix (take_prefix value (- start 1) unit))
			(define remaining (substr value (strlen prefix)))
			(define removed (take_prefix remaining (min amount (text_length remaining unit)) unit))
			(define suffix (substr remaining (strlen removed)))
			(define replacement (coalesceNil replacement ""))
			(define size (+ (text_length prefix size_unit) (text_length replacement size_unit) (text_length suffix size_unit)))
			(if (> size limit) (raise_error 8115 "STUFF result exceeds its declared return capacity" 1) true)
			(concat prefix replacement suffix))))))
(define tsql_replicate_operator (lambda (take_prefix raise_error) (lambda (value amount size_unit limit large utf8)
	(if (or (nil? value) (nil? amount)) nil
		(< amount 0) nil
		(or (equal? amount 0) (equal? value "")) ""
		(begin
			(define width (text_length value size_unit))
			(define repetitions (intdiv limit width))
			(if large (begin
				(if (or (> amount repetitions) (> amount (intdiv 2147483647 (strlen value))))
					(raise_error 8115 "REPLICATE result exceeds its declared return capacity" 1) true)
				(string_repeat value amount))
				(begin
					(define complete (min amount repetitions))
					(define prefix (string_repeat value complete))
					(if (<= amount repetitions) prefix
						(concat prefix (if utf8 (utf8_prefix value (mod limit width))
							(take_prefix value (mod limit width) size_unit)))))))))))
(define tsql_edit_decoder (lambda (decode) (lambda (bytes encoding) (if (nil? bytes) nil (begin
	(define decoded (decode bytes encoding))
	(if (nil? decoded) (error "undefined byte in the supported nonnational character encoding") decoded))))))
(define tsql_edit_character_info (lambda (info national)
	(if (or (sql_text_type? (sql_info_type info)) (equal? (sql_info_type info) "NULL")) info
		((tsql_text_compiler -1 national false) (list info)))))
(define tsql_edit_national? (lambda (info)
	(has? '("NCHAR" "NVARCHAR" "SYSNAME") (sql_info_type info))))
(define tsql_edit_large? (lambda (info)
	(equal? (sql_info_dimensions info) '(-1))))
(define tsql_edit_collation (lambda (infos)
	(reduce infos (lambda (collation info) (sql_merge_collation collation (sql_info_collation info))) nil)))
(define tsql_edit_return_type (lambda (national binary large)
	(list (if binary "VARBINARY" national "NVARCHAR" "VARCHAR")
		(list (if large -1 national 4000 8000)))))
(define tsql_compile_stuff (lambda (infos) (begin
	(if (equal? (count infos) 4) true (error "STUFF requires four arguments"))
	(define input (car infos))
	(define replacement (nth infos 3))
	(define binary (tsql_binary_type? (sql_info_type input)))
	(if (has? '("TEXT" "NTEXT" "IMAGE") (sql_info_type input))
		(error "STUFF requires a convertible character or binary declaration") true)
	(if (if binary (or (tsql_binary_type? (sql_info_type replacement))
		(has? '("NULL" "CHAR" "VARCHAR") (sql_info_type replacement)))
		(not (tsql_binary_type? (sql_info_type replacement)))) true
		(error "STUFF replacement conversion between binary and character data is not implemented"))
	(define national (and (not binary) (or (tsql_edit_national? input) (tsql_edit_national? replacement))))
	(define large (tsql_edit_large? input))
	(define text (if binary input (tsql_edit_character_info input national)))
	(define replacement (if binary replacement (tsql_edit_character_info replacement national)))
	(define collation (if binary nil (tsql_edit_collation (list text replacement))))
	(define encoding (if national "utf-8" binary (tsql_character_encoding replacement)
		(tsql_character_encoding (sql_info nil "VARCHAR" collation))))
	(define positions (if binary "byte" (not national) (if (equal? encoding "utf-8") "rune" "byte")
		(and collation (regexp_test (car collation) "(?i)_sc(?:_|$)")) "rune" "utf16"))
	(define size_unit (if national "utf16" "byte"))
	(define text_formula (if (or binary national) (sql_info_formula text)
		(list tsql_encode_character (sql_info_formula text) encoding -1)))
	(define replacement_formula (if (or national (tsql_binary_type? (sql_info_type replacement)))
		(sql_info_formula replacement) (list tsql_encode_character (sql_info_formula replacement) encoding -1)))
	(define limit (if large (if national 1073741823 2147483647) national 4000 8000))
	(define formula (list (tsql_stuff_operator tsql_edit_prefix tsql_raise_error) text_formula
		(sql_info_formula (tsql_convert_value_info (nth infos 1) "BIGINT" nil))
		(sql_info_formula (tsql_convert_value_info (nth infos 2) "BIGINT" nil))
		replacement_formula positions size_unit limit))
	(sql_info (if (or binary national) formula (list (tsql_edit_decoder tsql_decode_character) formula encoding))
		(tsql_edit_return_type national binary large) collation))))
(define tsql_compile_replicate (lambda (infos) (begin
	(if (equal? (count infos) 2) true (error "REPLICATE requires two arguments"))
	(define input (car infos))
	(if (has? '("TEXT" "NTEXT" "IMAGE") (sql_info_type input))
		(error "REPLICATE requires a convertible character declaration") true)
	(define national (tsql_edit_national? input))
	(define large (tsql_edit_large? input))
	(define binary (tsql_binary_type? (sql_info_type input)))
	(define converted (if binary input (tsql_edit_character_info input national)))
	(define encoding (if national "utf-8" (tsql_character_encoding converted)))
	(define value (if (or national binary) (sql_info_formula converted)
		(list tsql_encode_character (sql_info_formula converted) encoding -1)))
	(define formula (list (tsql_replicate_operator tsql_edit_prefix tsql_raise_error) value
		(sql_info_formula (tsql_convert_value_info (cadr infos) "BIGINT" nil))
		(if national "utf16" "byte")
		(if large (if national 1073741823 2147483647) national 4000 8000) large
		(and (not national) (equal? encoding "utf-8"))))
	(sql_info (if national formula (list (tsql_edit_decoder tsql_decode_character) formula encoding))
		(tsql_edit_return_type national false large) (if binary nil (sql_info_collation converted))))))
/* STR owns field width and half-away-from-zero rounding in the frontend.
The runtime formatter only renders the already rounded floating value. */
(define tsql_str (close_procedure ((lambda (tsql_round_real tsql_round_real_factor) (lambda (value width places)
	(if (or (nil? value) (nil? width) (nil? places)) nil
		(or (<= width 0) (< places 0)) nil
		(> width 8000) (error "STR fields wider than 8000 are unsupported")
		(begin
			(define precision (min places 16))
			(define rounded (tsql_round_real value precision 0 (tsql_round_real_factor precision)))
			(define text (format_fixed (if (equal? rounded 0) 0 rounded) precision))
			(if (> (strlen text) width) (string_repeat "*" width)
				(concat (string_repeat " " (- width (strlen text))) text)))))) tsql_round_real tsql_round_real_factor)))
(define tsql_str_dimensions (lambda (width) (begin
	(define known (try (lambda () (tryfold width)) (lambda (_cause) nil)))
	(list (if (and (number? known) (> known 0) (<= known 8000)) known 8000)))))
(define tsql_compile_str (lambda (infos) (begin
	(if (and (>= (count infos) 1) (<= (count infos) 3)) true
		(error "STR requires value and optional length and decimal arguments"))
	(define input (tsql_convert_value_info (car infos) "FLOAT" nil))
	(define width (if (>= (count infos) 2)
		(tsql_convert_value_info (cadr infos) "INT" nil) (sql_info 10 "INT" nil)))
	(define places (if (equal? (count infos) 3)
		(tsql_convert_value_info (nth infos 2) "INT" nil) (sql_info 0 "INT" nil)))
	(sql_info (list tsql_str (sql_info_formula input) (sql_info_formula width) (sql_info_formula places))
		(list "VARCHAR" (tsql_str_dimensions (sql_info_formula width))) nil))))

/* Numeric validation is deliberately distinct from TRY_CONVERT(INT,...):
SQL numeric text also includes decimal, floating and monetary spellings.
The frontend chooses accepted declarations and every SQL return contract. */
(define tsql_numeric_currency "[$¢£¤¥৲৳฿៛₠₡₢₣₤₥₦₧₨₩₪₫€₭₮₯₰₱﷼﹩＄￠￡￥￦]")
(define tsql_isnumeric_text (lambda (value) (if (nil? value) 0 (begin
	(define text (sql_trim value))
	(if (equal? text "") 0
		(regexp_test text (concat "^(?:[+.,-]|" tsql_numeric_currency ")$")) 1
		(begin
			(define money (regexp_test text (concat "^(?:[+-]?" tsql_numeric_currency "|" tsql_numeric_currency "[+-]?)[0-9.,]+$")))
			(define plain (if money (regexp_replace text tsql_numeric_currency "") text))
			(define monetary (or money (regexp_test plain ",")))
			(define valid (if monetary
				(regexp_test plain "^[+-]?(?:[0-9][0-9,]*(?:\\.[0-9]*)?|\\.[0-9]+)$")
				(regexp_test plain "^[+-]?(?:[0-9]+\\.?[0-9]*|\\.[0-9]+)(?:[eEdD][+-]?[0-9]+)?$")))
			(if (not valid) 0
				(try (lambda () (begin
					(define number (tsql_numeric_input (replace (replace (replace plain "," "") "D" "e") "d" "e")))
					(if monetary (begin (tsql_money_coefficient number false) 1) 1)))
					(lambda (_cause) 0)))))))))
(define tsql_compile_isnumeric (lambda (infos) (begin
	(if (equal? (count infos) 1) true (error "ISNUMERIC requires one argument"))
	(define input (car infos))
	(define type (tsql_addition_type (sql_info_type input)))
	(define character (has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR" "TEXT" "NTEXT" "SYSNAME") type))
	(if (or character (sql_numeric_type? type) (has? '("NULL" "BIT" "MONEY" "SMALLMONEY") type)) true
		(error "ISNUMERIC does not support this declaration: " type))
	(sql_info (if character (list tsql_isnumeric_text (sql_info_formula input))
		(list (quote if) (list (quote nil?) (sql_info_formula input)) 0 1)) "INT" nil))))
(define tsql_sign (lambda (value) (if (nil? value) nil (< value 0) -1 (> value 0) 1 0)))
(define tsql_compile_sign (lambda (infos) (begin
	(if (equal? (count infos) 1) true (error "SIGN requires one argument"))
	(define input (car infos))
	(define type (tsql_addition_type (sql_info_type input)))
	(if (equal? type "BIT") (error "SIGN does not accept BIT") true)
	(define integer (has? '("NULL" "TINYINT" "SMALLINT" "INT" "BIGINT") type))
	(define monetary (has? '("MONEY" "SMALLMONEY") type))
	(define decimal (equal? type "DECIMAL"))
	(define floating (or (has? '("FLOAT" "REAL") type) (has? tsql_addition_text_types type)))
	(if (or integer monetary decimal floating) true (error "SIGN requires a numeric declaration"))
	(define converted (if floating (tsql_convert_value_info input "FLOAT" nil) input))
	(sql_info (list tsql_sign (sql_info_formula converted))
		(if integer (if (equal? type "BIGINT") "BIGINT" "INT")
			monetary "MONEY" decimal (sql_info_datatype input) "FLOAT") nil))))
(define tsql_power (lambda (value exponent) (if (or (nil? value) (nil? exponent)) nil (begin
	(if (and (< value 0) (not (equal? exponent (floor exponent))))
		(tsql_raise_error 3623 "invalid floating point operation" 1) true)
	(tsql_numeric_input (number_power value exponent))))))
(define tsql_power_integer (lambda (value exponent minimum maximum) (begin
	(define result (tsql_power value exponent))
	(if (nil? result) nil
		/* A floating value at +2^63 cannot distinguish INT64_MAX. Test the
		exclusive bound before the generic integer conversion can round it. */
		(and (not (int? result)) (>= result (simplify "9223372036854775808.0")))
		(tsql_raise_error 8115 "POWER integer overflow" 1)
		(tsql_cast_integer result minimum maximum)))))
(define tsql_compile_power (lambda (infos) (begin
	(if (equal? (count infos) 2) true (error "POWER requires two arguments"))
	(define input (car infos))
	(define type (tsql_addition_type (sql_info_type input)))
	(define exponent (cadr infos))
	(if (equal? (sql_info_type exponent) "BIT") (error "POWER exponent does not accept BIT") true)
	(define exponent (tsql_convert_value_info exponent "FLOAT" nil))
	(define integer (has? '("NULL" "TINYINT" "SMALLINT" "INT" "BIGINT") type))
	(define monetary (has? '("MONEY" "SMALLMONEY") type))
	(define decimal (equal? type "DECIMAL"))
	(define floating (or (has? '("FLOAT" "REAL" "BIT") type) (has? tsql_addition_text_types type)))
	(if (or integer monetary decimal floating) true (error "POWER requires a numeric declaration"))
	(define converted (if floating (tsql_convert_value_info input "FLOAT" nil) input))
	(define value (sql_info_formula converted))
	(define power (sql_info_formula exponent))
	(define formula (if integer
		(list tsql_power_integer value power
			(if (equal? type "BIGINT") tsql_round_int64_minimum -2147483648)
			(if (equal? type "BIGINT") tsql_round_int64_maximum 2147483647))
		(list tsql_power value power)))
	(define dimensions (sql_info_dimensions input))
	(define scale (if (and decimal (not (empty_list? dimensions))) (cadr dimensions) 0))
	(sql_info (if monetary (list tsql_cast_money formula false)
		decimal (list tsql_cast_decimal formula 38 scale) formula)
		(if integer (if (equal? type "BIGINT") "BIGINT" "INT")
			monetary "MONEY" decimal (list "DECIMAL" (list 38 scale)) "FLOAT") nil))))

/* Search syntax, SQL NULL contracts and declared result types stay in this
frontend. Runtime operators receive plain strings and explicit coordinates. */
(define tsql_search_limit (lambda (value)
	(if (or (> (utf16_length value) 8000) (regexp_test value "\x00"))
		(error "search patterns require at most 8000 UTF16 units and no NUL") value)))
(define tsql_charindex (lambda (pattern text start fold unit)
	(if (or (nil? pattern) (nil? text) (nil? start)) nil (begin
		(tsql_search_limit pattern)
		(+ 1 (string_find text pattern (- (max start 1) 1) fold unit))))))
(define tsql_regex_literal (lambda (token)
	(if (has? '("." "*" "+" "?" "(" ")" "[" "]" "{" "}" "^" "$" "|" "\\") token)
		(concat "\\" token) token)))
(define tsql_pattern_token_regex (lambda (token)
	(if (equal? token "%") ".*" (equal? token "_") "."
		(and (> (strlen token) 1) (equal? (substr token 0 1) "["))
		(begin
			(define inner (substr token 1 (- (strlen token) 2)))
			(define negated (equal? (substr inner 0 1) "^"))
			(define members (if negated (substr inner 1) inner))
			/* Backslash and brackets are literal class members in T-SQL. */
			(define escaped (map (regexp_matches members "(?s:.)") (lambda (char)
				(if (has? '("\\" "[" "]") char) (concat "\\" char) char))))
			(concat "[" (if negated "^" "") (apply concat escaped) "]"))
		(tsql_regex_literal token))))
(define tsql_pattern_regex (lambda (pattern fold) (begin
	(define pattern (if fold (toLower pattern) pattern))
	(define tokens (regexp_matches pattern "\\[(?:\\^?\\]?[^\\]]*)\\]|(?s:.)"))
	(define leading (and (not (empty_list? tokens)) (equal? (car tokens) "%")))
	(define body (reduce tokens (lambda (body token)
		(if (and (equal? body "") (equal? token "%")) body
			(concat body (tsql_pattern_token_regex token)))) ""))
	(if (equal? body "") "(?s)^"
		(concat "(?s)" (if leading "" "^") body "$")))))
(define tsql_patindex (lambda (pattern text fold unit)
	(if (nil? pattern) nil (nil? text) (error "PATINDEX expression must not be NULL")
		(equal? pattern "") 0
		(begin
			(tsql_search_limit pattern)
			(if (regexp_test text "\x00") (error "PATINDEX expression must not contain NUL") true)
			(+ 1 (regexp_find text (tsql_pattern_regex pattern fold) fold unit))))))
(define tsql_search_text_info (lambda (info)
	(if (or (sql_text_type? (sql_info_type info)) (equal? (sql_info_type info) "NULL")) info
		((tsql_text_compiler -1 false false) (list info)))))
(define tsql_compile_search (lambda (name) (lambda (infos) (begin
	(if (if (equal? name "CHARINDEX") (has? '(2 3) (count infos)) (equal? (count infos) 2)) true
		(error name " has an invalid argument count"))
	(define original (cadr infos))
	(if (or (tsql_binary_type? (sql_info_type original)) (tsql_binary_type? (sql_info_type (car infos)))
		(and (equal? name "CHARINDEX")
			(or (has? '("TEXT" "NTEXT") (sql_info_type original)) (has? '("TEXT" "NTEXT") (sql_info_type (car infos))))))
		(error name " requires supported character declarations") true)
	(define pattern (tsql_search_text_info (car infos)))
	(define input (tsql_search_text_info original))
	(define collation (sql_merge_collation (sql_info_collation pattern) (sql_info_collation input)))
	(define collation_name (toLower (if (nil? collation) "utf8mb4_general_ci" (car collation))))
	(if (regexp_test collation_name "_ai(?:_|$)") (error "accent-insensitive search is unsupported") true)
	(define fold (regexp_test collation_name "_ci(?:_|$)"))
	(define unit (if (regexp_test collation_name "_sc(?:_|$)") "rune" "utf16"))
	(define arguments (list (sql_info_formula pattern) (sql_info_formula input)))
	(define call (if (equal? name "CHARINDEX")
		(cons (quote tsql_charindex) (merge (list arguments (list
			(if (equal? (count infos) 3) (sql_info_formula (tsql_convert_value_info (nth infos 2) "BIGINT" nil)) 1) fold unit))))
		(cons (quote tsql_patindex) (merge (list arguments (list fold unit))))))
	(sql_info call (if (equal? (sql_info_dimensions original) '(-1)) "BIGINT" "INT") nil)))))
(define tsql_compile_like (lambda (infos) (begin
	(define collation (sql_merge_collation (sql_info_collation (car infos)) (sql_info_collation (cadr infos))))
	(sql_info (list (quote strlike) (sql_info_formula (car infos)) (sql_info_formula (cadr infos))
		(if (nil? collation) "utf8mb4_general_ci" (car collation))) "BOOLEAN" nil))))

(define tsql_builtin_compiler (lambda (name) (lambda (infos)
	(if (has? '("NEWID" "UUIDV7") name) (begin
		(if (empty_list? infos) true (error name " takes no arguments"))
		(sql_info (list (if (equal? name "NEWID") (quote uuid_v4) (quote uuid_v7))) "UNIQUEIDENTIFIER" nil))
		(equal? name "CHAR") (tsql_compile_char infos)
		(equal? name "ASCII") (tsql_compile_ascii infos)
		(equal? name "DATALENGTH") (tsql_compile_datalength infos)
		(equal? name "LEN") (tsql_compile_len infos)
		(equal? name "ROUND") (tsql_compile_round infos)
		(equal? name "ISNUMERIC") (tsql_compile_isnumeric infos)
		(equal? name "SIGN") (tsql_compile_sign infos)
		(equal? name "POWER") (tsql_compile_power infos)
		(has? '("ABS" "FLOOR" "CEILING") name) (tsql_compile_numeric_scalar name infos)
		(has? '("SQUARE" "SQRT") name) (tsql_compile_floating_scalar name infos)
		(has? '("LEFT" "RIGHT" "SUBSTRING") name) (tsql_compile_slicing name infos)
		(has? '("UPPER" "LOWER" "LTRIM" "RTRIM" "TRIM") name) (tsql_compile_string_normalization name infos)
		(sql_call_info (coalesce (tsql_builtins name) (error "unknown function " name)) infos)))))

(define tsql_builtin_expr (lambda (name arguments) (begin
	(if (has? '("STUFF" "REPLICATE") name)
		(cons (quote sql_compile_expr) (cons (if (equal? name "STUFF") tsql_compile_stuff tsql_compile_replicate) arguments))
		(has? '("CHARINDEX" "PATINDEX") name)
		(cons (quote sql_compile_expr) (cons (list (quote tsql_compile_search) name) arguments))
		(equal? name "ISDATE")
		(cons (quote sql_compile_expr) (cons (quote tsql_compile_isdate) arguments))
		(equal? name "PARSENAME")
		(cons (quote sql_compile_expr) (cons (quote tsql_compile_parsename) arguments))
		(equal? name "STR") (cons (quote sql_compile_expr) (cons (quote tsql_compile_str) arguments))
		(begin
			(if (tsql_builtins name) true (error "unknown function " name))
			(cons (quote sql_compile_expr) (cons (list (quote tsql_builtin_compiler) name) arguments)))))))

/* ODBC scalar calls have their own declared names. Reuse the ordinary SQL
expression compilers only where their contracts agree; other ODBC functions
remain explicit unsupported cases rather than guessed native aliases. */
(define tsql_odbc_scalar_expr (lambda (name arguments)
	(if (has? '("YEAR" "MONTH") name) (begin
		(if (equal? (count arguments) 1) true (error name " requires one argument"))
		(tsql_calendar_scalar_expr (toLower name) (car arguments)))
		(tsql_builtin_expr (match name
			"REPLACE" "REPLACE" "UCASE" "UPPER" "LCASE" "LOWER" "ABS" "ABS"
			_ (error "unsupported ODBC scalar function " name)) arguments))))

(define tsql_builtins (lambda (name) (match name
	/* First-class callers retain lazy fallback; SQL calls use the declared
	ISNULL compiler contract rather than this scalar operator alias. */
	"NEWID" uuid_v4
	"UUIDV7" uuid_v7
	"ISNULL" coalesceNil
	"LEFT" tsql_left
	"RIGHT" tsql_right
	"SUBSTRING" (lambda (value start amount) (tsql_substring value start amount false))
	"CHAR" tsql_char
	"ASCII" tsql_ascii
	"DATALENGTH" tsql_binary_datalength
	"LEN" (lambda (value) (tsql_len value "byte" "windows-1252"))

	"ISNUMERIC" tsql_isnumeric_text
	"SIGN" tsql_sign
	"POWER" tsql_power
	"SQUARE" tsql_square
	"SQRT" tsql_square_root
	"LTRIM" tsql_ltrim
	"RTRIM" tsql_rtrim
	"TRIM" tsql_trim
	_ (if (has? '("ABS" "CEILING" "FLOOR" "ROUND" "UPPER" "LOWER" "REPLACE" "CONCAT" "YEAR" "MONTH" "DAY") name)
		(sql_builtins name) nil))))

/* GO belongs to the script envelope, never the SQL statement grammar. Quoted
values/identifiers and nested comments are consumed as complete lexical tokens;
a separator is recognized only on a physical line containing no SQL text. */
(define tsql_script_comment (parser '("/*"
	(* (or tsql_script_comment
		(regex "[^*/]+" false false)
		/* Consume punctuation singly without swallowing either delimiter. */
		(not (regex "[*/]" false false)
			(atom "/*" false false) (atom "*/" false false))) empty)
	"*/") nil "^$"))
(define tsql_script_tokens (parser (define tokens (* (or
	(parser (define text (capture tsql_script_comment)) (list "comment" (car text)))
	(parser (define text (regex "(?:--[^\\r\\n]*)" false false)) (list "comment" text))
	(parser (define text (regex "(?:'(?:''|[^'])*'|\\\"(?:\\\"\\\"|[^\\\"])*\\\"|\\[(?:\\]\\]|[^\\]])*\\])" false false)) (list "quoted" text))
	(parser (define text (regex "(?:\\r\\n|\\r|\\n)" false false)) (list "newline" text))
	/* Punctuation is one token: consuming its next character could swallow
	the opening quote of an adjacent literal or a physical newline. */
	(parser (define text (or
		(regex "[^'\\\"\\[\\r\\n/-]+" false false)
		(not (regex "[/-]" false false) (atom "/*" false false) (atom "--" false false))))
		(list "text" text))
) empty)) tokens "^$"))
(define tsql_script_flush_line (lambda (state newline) (begin
	(define line (apply concat (reverse (nth state 2))))
	(define code (apply concat (reverse (nth state 3))))
	(define separator (and (not (nth state 4))
		(regexp_test code "(?i)^[\\t ]*GO(?:[\\t ]+[0-9]+)?[\\t ]*$")))
	(if separator (begin
		(define repetitions (regexp_replace (sql_trim code) "(?i)^GO[\\t ]*" ""))
		(define repetitions (if (equal? repetitions "") 1 (tsql_parse_integer repetitions)))
		(if (<= repetitions 0) (error "GO count must be positive") true)
		(define text (apply concat (reverse (nth state 1))))
		(list (if (equal? (strtrim text) "") (car state)
			(cons (list "text" text "repeat" repetitions) (car state))) '() '() '() false))
		(list (car state) (cons (concat line newline) (nth state 1)) '() '() false)))))
(define tsql_script_batches (lambda (text) (begin
	/* Strip only the leading export marker; literal contents stay unchanged. */
	(define text (if (and (>= (strlen text) 3) (equal? (substr text 0 3) (unicode_char 65279)))
		(substr text 3) text))
	(define state (reduce (tsql_script_tokens text) (lambda (state token)
		(match token
			'("newline" newline) (tsql_script_flush_line state newline)
			'(kind text) (if (and (equal? kind "comment") (regexp_test text "[\r\n]"))
				/* Keep a spanning comment balanced in the preceding batch while
				resetting recognition to the physical line where it closes. */
				(list (car state) (cons (concat (apply concat (reverse (nth state 2))) text)
					(nth state 1)) '() '() false)
				(list (car state) (nth state 1) (cons text (nth state 2))
					(if (equal? kind "text") (cons text (nth state 3)) (nth state 3))
					(or (nth state 4) (equal? kind "quoted"))))))
		(list '() '() '() '() false)))
	(define final (tsql_script_flush_line state ""))
	(define tail (apply concat (reverse (nth final 1))))
	(reverse (if (equal? (strtrim tail) "") (car final)
		(cons (list "text" tail "repeat" 1) (car final)))))))

(define tsql_connection_session (lambda (session)
	(coalesceNil (session "__tsql_connection_session") session)))

(import "tsql-parse-context.scm")

/* Metadata scalars require the current consuming policy. Direct queries pass
that context explicitly; stored views refuse this scalar before publication. */
(define tsql_column_length (lambda (database policy relation column tx)
	(if (or (nil? relation) (nil? column)) nil (begin
		(define object ((parser '( (? (define owner tsql_identifier) ".")
			(define name tsql_identifier)) (list (if (or (nil? owner) (equal?? owner "dbo")) database owner) name)) relation))
		(define object_schema (car object))
		(define name (coalesceNil (show (table object_schema false (cadr object) true tx) "name") (cadr object)))
		(if policy (policy object_schema name false) true)
		(define handle (table object_schema false name false tx))
		(define declaration (if (nil? handle) nil (tsql_effective_column (show handle "column" column true tx) nil)))
		(if (nil? declaration) nil (tsql_declared_byte_length declaration))))))

(define tsql_view_has_metadata_scalar? (lambda (node) (match node
	((symbol quote) _datum) false
	(cons (symbol tsql_column_length) _arguments) true
	(cons _head _tail) (not (nil? (find node tsql_view_has_metadata_scalar? nil)))
	_ false)))

(define parse_tsql (lambda (database s policy planning_session tx) (begin
	(define schema database)
	(define frontend_options (if (nil? planning_session) (newsession) planning_session))
	(define ansi_nulls (tsql_ansi_nulls frontend_options))
	(define provided_context (frontend_options "__tsql_lexical_context"))
	(define lexical_context (if (nil? provided_context)
		(tsql_parse_context s (tsql_quoted_identifier frontend_options)
			(coalesceNil (frontend_options "__tsql_static_module") false)) provided_context))
	(define s (lexical_context "text"))
	(if planning_session (begin
		(planning_session "__tsql_temporary_planning" true)
		(define temporary_bindings (coalesceNil (planning_session "__tsql_temporary_bindings") '()))
		(planner_record_guard_condition
			(list (quote equal?) (list (quote session) "__tsql_temporary_bindings")
				(list (quote quote) temporary_bindings)) planning_session)) true)
	(define parse_started_ns (nanotime))
	/* Import bindings are compile-time name contracts. Keep the authenticated
	policy and confine every published dependency, including expanded views. */
	(define account_policy policy)
	(define import_target (frontend_options "tsql_import_target"))
	(define policy (if (nil? import_target) account_policy (lambda (database name write) (begin
		(if (and (equal? database import_target)
			(not (and (equal? database "system") (expression_equal? name true) write))) true
			(error "import reference outside destination and source alias: " database))
		(if account_policy (account_policy database name write) true)))))
	/* Requirements are immutable action tuples in the resulting formula. They
	are rechecked for every execution, including a warm cached plan. Neither an
	account snapshot nor a connection/session capability becomes a cache key. */
	(define access_policy policy)
	(define write_targets (newsession))
	(define require_action (lambda (database name action) (if access_policy (begin
		(access_policy database name true)
		(tsql_record_permission frontend_options database
			(if (or (sql_private_relation? name) (and (string? name) (tsql_temporary_name? name))) "CONNECT" action))
		(write_targets (json_encode (list database name)) true)
		true) true)))
	(define policy (if (nil? access_policy) nil (lambda (database name write) (begin
		(if (equal?? database "information_schema") true
			(and (equal? database "system") (expression_equal? name true) write)
			(tsql_record_permission frontend_options database "NATIVE ADMIN")
			(tsql_record_permission frontend_options database
				(if (or write (expression_equal? name true) (sql_private_relation? name)
					(write_targets (json_encode (list database name)))) "CONNECT" "SELECT")))
		(access_policy database name write)))))
	(define describe (frontend_options "__tsql_describe"))
	(define script_syntax (frontend_options "__tsql_script_syntax"))
	(define expression_only (frontend_options "__tsql_expression_only"))
	(define function_body (frontend_options "__tsql_scalar_function_body"))
	(define analyze_parameters (frontend_options "__tsql_parameter_analysis"))
	(define parameters (coalesceNil (frontend_options "__tsql_parameter_declarations") (list)))
	(define parameter_expression (lambda (name spelling) (begin
		(define name (toLower name))
		(define declaration (filter parameters (lambda (item) (equal? (car item) name))))
		(if script_syntax (list (quote tsql_batch_variable) name)
			(and analyze_parameters (empty_list? declaration))
			(list (quote sql_type_expr) (list (quote tsql_unbound_parameter) name spelling) "any" nil)
			(empty_list? declaration) (error "undeclared T-SQL parameter: @" name)
			(equal?? (car (cadr (car declaration))) "TABLE") (error "table variable cannot be used as a scalar: @" name)
			(list (quote sql_type_expr) (if function_body
				(tsql_scalar_function_variable name) (list (quote session) (concat "__tsql_parameter:" name)))
				(begin
					(define contract (tsql_effective_parameter_contract (cadr (car declaration))))
					(if (empty_list? (cdr contract)) (car contract) (list (car contract) (cdr contract)))) nil)))))

	(define placeholder_counter (newsession))
	(placeholder_counter "n" 0)

	(define extract_title_or_sql (lambda (captured_result)
		(match (car (cdr captured_result))
			((symbol sql_compile_column) _compiler ((symbol get_column) _alias _alias_ic column _column_ic)) column
			'((symbol get_column) nil _ col _) col
			'((symbol get_column) tblvar _ col _) col
			_ (begin
				(define raw_sql (car captured_result))

				(if (> (strlen raw_sql) 256)
					(concat (substr raw_sql 0 200) "..." (fnv_hash raw_sql))
					raw_sql))
		)
	))

	(define tsql_expression (parser (or
		(parser '((define a tsql_expression1) (atom "OR" true) (define b (+ tsql_expression1 (atom "OR" true)))) (cons (quote or) (cons a b)))
		tsql_expression1
	)))

	(define tsql_expression1 (parser (or
		(parser '((define a tsql_expression2) (atom "AND" true) (define b (+ tsql_expression2 (atom "AND" true)))) (cons (quote and) (cons a b)))
		tsql_expression2
	)))

	(define tsql_in_values (parser (or
		(parser '("(" "?*" ")") (sql_in_array_parameter placeholder_counter))
		(parser '("(" (define values (+ tsql_expression ",")) ")") (cons list values)))))

	(define tsql_expression2 (parser (or

		(parser '((atom "NOT" true) (atom "EXISTS" true) "(" (define sub tsql_select) ")") (list (quote not) (tsql_predicate_expr (list (quote inner_select_exists) sub))))
		(parser '((atom "NOT" true) (define expr tsql_expression2)) '('sql_not expr))

		(parser '((define a tsql_expression3) (atom "IN" true) "(" (define sub tsql_select) ")") (tsql_predicate_expr '('inner_select_in a sub)))
		(parser '((define a tsql_expression3) (atom "NOT" true) (atom "IN" true) "(" (define sub tsql_select) ")") (list (quote not) (tsql_predicate_expr (list (quote inner_select_in) a sub))))

		(parser '((define a tsql_expression3) (define op (or
			(parser "=" "eq")
			(parser "<>" "ne")
			(parser "!=" "ne")
			(parser "<=" "le")
			(parser ">=" "ge")
			(parser "<" "lt")
			(parser ">" "gt")
		)) (define b tsql_expression2)) (tsql_comparison_expr_using ansi_nulls op a b))

		(parser '((define a tsql_expression3) (atom "LIKE" true) (define b tsql_expression2)) (list (quote sql_compile_expr) (quote tsql_compile_like) a b))
		(parser '((define a tsql_expression3) (atom "NOT" true) (atom "LIKE" true) (define b tsql_expression2)) (list (quote sql_not) (list (quote sql_compile_expr) (quote tsql_compile_like) a b)))

		(parser '((define a tsql_expression3) (atom "IN" true) (define b tsql_in_values)) (tsql_in_expr b a))
		(parser '((define a tsql_expression3) (atom "NOT" true) (atom "IN" true) (define b tsql_in_values)) (list (quote sql_not) (tsql_in_expr b a)))

		(parser '((define a tsql_expression3) (atom "BETWEEN" true) (define low tsql_expression3) (atom "AND" true) (define high tsql_expression3)) (list (quote and) (list (quote >=) a low) (list (quote <=) a high)))
		(parser '((define a tsql_expression3) (atom "NOT" true) (atom "BETWEEN" true) (define low tsql_expression3) (atom "AND" true) (define high tsql_expression3)) (list (quote sql_not) (list (quote and) (list (quote >=) a low) (list (quote <=) a high))))
		tsql_expression3
	)))

	(define tsql_expression4 (parser '(
		(define a tsql_expression5)
		(define terms (* (parser '(
			(define op (or
				(parser "*" "multiply")
				(parser "/" "divide")

				(parser "%" "modulo")
			))
			(define value tsql_expression5)
		) (list op value)) empty true))
	) (reduce terms tsql_fold_multiplicative a)))

	(define tsql_unary_expression (parser (or
		(parser '("~" (define expr tsql_unary_expression)) (tsql_bit_expr "not" (list expr)))
		/* Signed literals must retain the exact lexer carrier before generic unary syntax. */
		tsql_expression6
		(parser '("+" (define expr tsql_unary_expression)) (tsql_positive_expr expr))
		(parser '("-" (define expr tsql_unary_expression)) (tsql_negate_expr expr)))))
	(define tsql_collated_expression (parser '((define value tsql_unary_expression)
		(? (atom "COLLATE" true) (define collation tsql_identifier)))
		(if (nil? collation) value
			(list (quote sql_compile_expr) (list (quote tsql_collation_compiler) collation) value))))
	(define tsql_expression5 (parser (or
		(parser '((define expr tsql_collated_expression) (atom "IS" true) (atom "NULL" true)) '('nil? expr))
		(parser '((define expr tsql_collated_expression) (atom "IS" true) (atom "NOT" true) (atom "NULL" true)) '('not '('nil? expr)))
		tsql_collated_expression)))

	/* Both count spellings emit the same native COUNT descriptor. Only the
	frontend result contract differs; ALL uses its ordinary non-NULL input. */
	(define tsql_count_type (parser (or (atom "COUNT" true true "INT")
		(atom "COUNT_BIG" true true "BIGINT"))))

	(define tsql_window_orderby_item (parser '(
		(define col tsql_expression)
		(define dir (or
			(parser (atom "DESC" true) >)
			(parser (atom "ASC" true) <)
			(parser empty <)
		))
	) (list col dir)))

	(define tsql_window_spec (parser '(
		(? (atom "PARTITION" true) (atom "BY" true) (define partition_by (+ tsql_expression ",")))
		(? (atom "ORDER" true) (atom "BY" true) (define order_by (+ tsql_window_orderby_item ",")))
	) (list (coalesce partition_by '()) (coalesce order_by '()))))

	(define column_length_expr (lambda (relation column)
		(list (quote sql_type_expr)
			(list (quote tsql_column_length) database
				(tsql_metadata_policy (list (coalesceNil (frontend_options "username") "root")
					(coalesceNil (frontend_options "__tsql_principal_id") (tsql_account_id (coalesceNil (frontend_options "username") "root")))))
				relation column (quote tx)) "INT" nil)))

	(define tsql_expression6 (parser (or
		(parser '("{" (atom "FN" true) (define name tsql_identifier_unquoted)
			"(" (define arguments (* tsql_expression ",")) ")" "}")
			(tsql_odbc_scalar_expr (toUpper name) arguments))

		/* UPDATE(column) consumes an identifier, never that column's row value.
		The completion module supplies immutable operation/name facts once; the
		ordinary planner sees their explicit scalar session dependencies. */
		(parser '((atom "UPDATE" true) "(" (define column tsql_identifier) ")")
			(list 'sql_type_expr (list 'tsql_trigger_column_updated
				(list 'session "__tsql_trigger_event")
				(list 'session "__tsql_trigger_columns")
				(list 'session "__tsql_trigger_target_columns") column) "BOOLEAN" nil))
		(parser '("(" (define sub tsql_select) ")") '('inner_select sub))
		(parser '("(" (define a tsql_expression) ")") a)
		(parser '((define operation (or (parser (atom "LEFT_SHIFT" true) "left")
			(parser (atom "RIGHT_SHIFT" true) "right"))) "(" (define value tsql_expression)
			"," (define amount tsql_expression) ")") (tsql_bit_expr operation (list value amount)))
		(parser '((atom "OBJECT_ID" true) "(" (define name tsql_expression)
			(? "," (define kind tsql_expression)) ")")
			(list 'sql_compile_expr (list 'tsql_compile_object_id schema) name
				(coalesceNil kind (list 'sql_type_expr nil "NVARCHAR" nil))))
		(parser '((atom "QUOTENAME" true) "(" (define value tsql_expression)
			(? "," (define delimiter tsql_expression)) ")")
			(list (quote sql_compile_expr) (quote tsql_compile_quotename) value
				(coalesceNil delimiter (list (quote sql_type_expr) "[" "NVARCHAR" nil))))
		(parser '((atom "SPACE" true) "(" (define amount tsql_expression) ")")
			(list (quote sql_compile_expr) (quote tsql_compile_space) amount))
		(parser '((atom "TRY_CAST" true) "(" (define value tsql_expression) (atom "AS" true)
			(define type tsql_declared_type) (define dimensions tsql_type_dimensions) ")")
			(tsql_try_convert_expr value type dimensions nil true))
		(parser '((atom "TRY_CONVERT" true) "(" (define type tsql_declared_type) (define dimensions tsql_type_dimensions)
			"," (define value tsql_expression) (? "," (define style (or tsql_int
				(parser (atom "NULL" true) (quote sql_null_style))))) ")")
			(tsql_try_convert_expr value type dimensions style false))
		(parser '((atom "CAST" true) "(" (define value tsql_expression) (atom "AS" true)
			(define type tsql_declared_type) (define dimensions tsql_type_dimensions) ")")
			(tsql_cast_expr value type dimensions))
		(parser '((atom "CONVERT" true) "(" (define type tsql_declared_type) (define dimensions tsql_type_dimensions)
			"," (define value tsql_expression) (? "," (define style (or tsql_int
				(parser (atom "NULL" true) (quote sql_null_style))))) ")")
			(tsql_convert_expr value type dimensions style))

		(parser '((atom "COL_LENGTH" true) "(" (define relation tsql_expression) ","
			(define column tsql_expression) ")") (column_length_expr relation column))

		/* Atom result values encode constant contracts directly, so keyword
		alternatives do not need their own parser generators. */
		(parser '((define type (or (atom "GETDATE" true true "DATETIME")
			(atom "GETUTCDATE" true true "DATETIME") (atom "SYSDATETIME" true true "DATETIME2")
			(atom "SYSUTCDATETIME" true true "DATETIME2"))) "(" ")")
			(list (quote sql_type_expr) (list (quote now)) type nil))
		(parser (atom "@@DATEFIRST" true) (tsql_datefirst_expr))
		(parser (atom "@@OPTIONS" true) (list 'sql_type_expr
			(list '+ 16 (if ansi_nulls 32 0) (if (tsql_quoted_identifier frontend_options) 256 0)
				(list 'if (list 'session "__tsql_nocount") 512 0)) "INT" nil))
		(parser (atom "@@MICROSOFTVERSION" true)
			(list 'sql_type_expr tsql_server_microsoft_version "INT" nil))
		(parser (atom "@@LANGUAGE" true) (list (quote sql_type_expr) (tsql_language_formula) (list "NVARCHAR" '(128)) '("utf8mb4_general_ci" 4)))
		(parser (atom "@@FETCH_STATUS" true)
			(list 'sql_type_expr (list 'coalesceNil (list 'session "__tsql_fetch_status") -9) "INT" nil))
		(parser (atom "@@TRANCOUNT" true)
			(list 'sql_type_expr (list 'if (list 'session "transaction") 1 0) "INT" nil))
		(parser (atom "@@DBTS" true)
			(list (quote sql_type_expr) (list (quote tsql_current_rowversion) database (quote tx)) (list "VARBINARY" (list 8)) nil))
		(parser '((atom "DATEADD" true) "(" (define unit tsql_identifier) ","
			(define amount tsql_expression) "," (define value tsql_expression) ")")
			(tsql_dateadd_expr unit amount value))
		(parser '((define type (or (parser (atom "DATEDIFF" true) "INT") (parser (atom "DATEDIFF_BIG" true) "BIGINT")))
			"(" (define unit tsql_identifier) ","
			(define first tsql_expression) "," (define last tsql_expression) ")")
			(tsql_datediff_expr unit first last type))
		(parser '((atom "DATEPART" true) "(" (define unit tsql_identifier) "," (define value tsql_expression) ")")
			(tsql_datepart_expr unit value))
		(parser '((atom "DATENAME" true) "(" (define unit tsql_identifier) "," (define value tsql_expression) ")")
			(tsql_datename_expr unit value))
		(parser '((atom "DATEFROMPARTS" true) "(" (define year tsql_expression) ","
			(define month tsql_expression) "," (define day tsql_expression) ")") (tsql_datefromparts_expr year month day))
		(parser '((atom "EOMONTH" true) "(" (define value tsql_expression)
			(? "," (define amount tsql_expression)) ")") (tsql_eomonth_expr value (coalesceNil amount (list (quote sql_type_expr) 0 "INT" nil))))
		(parser '((define unit (or (atom "YEAR" true true "year")
			(atom "MONTH" true true "month") (atom "DAY" true true "day")))
			"(" (define value tsql_expression) ")") (tsql_calendar_scalar_expr unit value))

		(parser '((atom "EXISTS" true) "(" (define sub tsql_select) ")") (tsql_predicate_expr '('inner_select_exists sub)))

		(parser '((atom "CASE" true) (define conditions (+ (parser '((atom "WHEN" true) (define a tsql_expression) (atom "THEN" true) (define b tsql_expression)) '(a b)))) (? (atom "ELSE" true) (define elsebranch tsql_expression)) (atom "END" true)) (tsql_case_expr conditions elsebranch))

		(parser '((atom "CASE" true) (define expr tsql_expression) (define conditions (+ (parser '((atom "WHEN" true) (define a tsql_expression) (atom "THEN" true) (define b tsql_expression)) '(a b)))) (? (atom "ELSE" true) (define elsebranch tsql_expression)) (atom "END" true)) (tsql_case_expr (map conditions (lambda (condition)
			(list (tsql_comparison_expr_using ansi_nulls "eq" expr (car condition)) (cadr condition)))) elsebranch))

		(parser '((atom "IIF" true) "(" (define condition tsql_expression) ","
			(define yes tsql_expression) "," (define no tsql_expression) ")")
			(tsql_case_expr (list (list condition yes)) no))
		(parser '((atom "CHOOSE" true) "(" (define index tsql_expression) ","
			(define values (+ tsql_expression ",")) ")") (tsql_choose_expr index values))

		(parser '((define count_type tsql_count_type) "(" "*" ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") (list (quote sql_type_expr) (tsql_window_expr "COUNT" (list) _over) count_type nil))
		(parser '((define count_type tsql_count_type) "(" (? (atom "ALL" true)) (define e tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") (list (quote sql_type_expr) (tsql_window_expr "COUNT" (list e) _over) count_type nil))
		(parser '((atom "SUM" true) "(" (define s tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") (tsql_window_expr "SUM" (list s) _over))
		(parser '((atom "AVG" true) "(" (define s tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") (tsql_window_expr "AVG" (list s) _over))
		/* DISTINCT is redundant for ordinary extrema, but illegal with OVER.
		Retain its syntax until that distinction is known, then emit the
		same typed aggregate/window operators as their unmodified forms. */
		(parser '((define name (or (atom "MIN" true true "MIN") (atom "MAX" true true "MAX")))
			"(" (define modifier (? (or (atom "ALL" true true "ALL") (atom "DISTINCT" true true "DISTINCT"))))
			(define s tsql_expression) ")"
			(define over (? (parser '((atom "OVER" true) "(" (define spec tsql_window_spec) ")") spec))))
			(if (not (nil? over))
				(if (equal? modifier "DISTINCT") (error "DISTINCT is not allowed with OVER")
					(tsql_window_expr name (list s) over))
				(begin (define d (sql_aggregates name))
					(tsql_aggregate_expr name '('aggregate s (car d) (cadr d)) s))))

		(parser '((define count_type tsql_count_type) "(" (atom "DISTINCT" true) (define e tsql_expression) ")")
			'('sql_type_expr '('count_distinct e) count_type nil))
		(parser '((define count_type tsql_count_type) "(" "*" ")")
			'('sql_type_expr '((quote aggregate) 1 (quote +) 0) count_type nil))
		(parser '((define count_type tsql_count_type) "(" (? (atom "ALL" true)) (define e tsql_expression) ")")
			'('sql_type_expr '('aggregate '((quote if) '((quote nil?) e) 0 1) (quote +) 0) count_type nil))
		(parser '((atom "SUM" true) "(" (define s tsql_expression) ")") (begin (define d (sql_aggregates "SUM")) (tsql_aggregate_expr "SUM" '('aggregate s (car d) (cadr d)) s)))
		(parser '((atom "AVG" true) "(" (define s tsql_expression) ")")
			(tsql_aggregate_expr "AVG" (sql_avg_expr s (sql_aggregates "SUM") (sql_aggregates "COUNT")) s))

		(parser '((define fn tsql_identifier_unquoted) "(" (define arg tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")")
			(tsql_window_expr (toUpper fn) (list arg) _over))

		(parser '((define name (or (atom "LEFT" true true "LEFT")
			(atom "RIGHT" true true "RIGHT") (atom "SUBSTRING" true true "SUBSTRING")))
			"(" (define args (* tsql_expression ",")) ")") (tsql_builtin_expr name args))

		(parser '((atom "CONCAT" true) "(" (define p (+ tsql_expression ",")) ")")
			(cons (quote concat) (map p (lambda (value) (list (quote coalesceNil) value "")))))

		(parser '((define name (or (atom "TRIM" true true "TRIM")
			(atom "LTRIM" true true "LTRIM") (atom "RTRIM" true true "RTRIM")))
			"(" (define args (* tsql_expression ",")) ")") (tsql_builtin_expr name args))

		(parser '((atom "COALESCE" true) "(" (define args (* tsql_expression ",")) ")") (tsql_coalesce_expr args))

		(parser '((atom "ISNULL" true) "(" (define args (* tsql_expression ",")) ")") (tsql_isnull_expr args))
		(parser '((atom "NULLIF" true) "(" (define args (* tsql_expression ",")) ")") (tsql_nullif_expr args))
		(parser '((atom "NCHAR" true) "(" (define value tsql_expression) ")") (tsql_nchar_expr value))

		(parser (atom "NULL" true) (sql_null_literal))
		(parser (atom "CURRENT_TIMESTAMP" true) (list (quote sql_type_expr) (list (quote now)) "DATETIME" nil))

		/* Current database is invocation context, bound by the shared planner.
		Its SQL declaration is independent of the session value carrier. */
		(parser '((atom "USER_NAME" true) "(" (define principal tsql_expression) ")")
			(begin
				(if policy (policy schema "database_principals" false) true)
				(tsql_catalog_name_call schema "principal" principal)))
		(parser '((atom "OBJECT_NAME" true) "(" (define object tsql_expression) ")")
			(begin
				(if policy (policy schema "objects" false) true)
				(tsql_catalog_name_call schema "object" object)))
		(parser '((atom "USER_NAME" true) "(" ")")
			(list (quote sql_type_expr)
				(list tsql_database_principal_name schema
					(list (quote session) "username") (list (quote session) "__tsql_principal_id"))
				(list "NVARCHAR" '(128)) '("utf8mb4_general_ci" 4)))
		(parser '((define name (or (atom "HOST_NAME" true true "hostname")
			(atom "APP_NAME" true true "appname"))) "(" ")")
			(list (quote sql_type_expr) (list tsql_client_context_name
				(list (quote session) (concat "__tsql_client_" name)))
				(list "NVARCHAR" '(128)) '("utf8mb4_general_ci" 4)))
		(parser '((atom "SUSER_SNAME" true) "(" ")")
			(list (quote sql_type_expr) (list tsql_security_context_name (list (quote session) "username"))
				(list "NVARCHAR" '(128)) '("utf8mb4_general_ci" 4)))
		(parser '((atom "SCHEMA_NAME" true) "(" (define args (* tsql_expression ",")) ")")
			(begin
				(if (<= (count args) 1) true (error "SCHEMA_NAME requires zero or one argument"))
				(list 'sql_type_expr (list tsql_schema_name_value
					(if (empty_list? args) 1 (tsql_cast_expr (car args) "INT" '())))
					(list "NVARCHAR" '(128)) '("utf8mb4_general_ci" 4))))
		(parser '((atom "DB_NAME" true) "(" ")")
			(list (quote sql_type_expr) (list (quote session) "schema") "NVARCHAR" '("utf8mb4_general_ci" 4)))
		(parser '((atom "IS_MEMBER" true) "(" (define args (* tsql_expression ",")) ")")
			(begin
				(if (equal? (count args) 1) true (error "IS_MEMBER requires one argument"))
				(list (quote sql_type_expr)
					(list tsql_is_member (tsql_cast_expr (car args) "NVARCHAR" '(128))
						(list (quote session) "schema") (tsql_actor_formula)) "INT" nil)))
		(parser '((atom "IS_SRVROLEMEMBER" true) "(" (define args (* tsql_expression ",")) ")")
			(begin
				(if (or (< (count args) 1) (> (count args) 2))
					(error "IS_SRVROLEMEMBER requires one or two arguments") true)
				(list (quote sql_type_expr)
					(list tsql_is_srvrolemember (tsql_cast_expr (car args) "NVARCHAR" '(128))
						(if (equal? (count args) 1) (list (quote session) "username")
							(tsql_cast_expr (nth args 1) "NVARCHAR" '(128)))
						(list (quote session) "username")) "INT" nil)))

		(parser '((atom "SERVERPROPERTY" true) "(" (define property tsql_expression) ")")
			(list (quote sql_type_expr) (list tsql_serverproperty
				(tsql_cast_expr property "NVARCHAR" '(128))) "SQL_VARIANT" nil))
		(parser '((atom "HAS_DBACCESS" true) "(" (define name tsql_expression) ")")
			(list (quote sql_type_expr)
				(list (quote tsql_has_dbaccess) (tsql_cast_expr name "NVARCHAR" '(128))
					(tsql_actor_formula)) "INT" nil))
		(parser '((atom "SCOPE_IDENTITY" true) "(" ")") (tsql_identity_expr true))
		(parser '((define name (or (atom "ERROR_MESSAGE" true true "message")
			(atom "ERROR_NUMBER" true true "number")
			(atom "ERROR_STATE" true true "state")
			(atom "ERROR_SEVERITY" true true "severity"))) "(" ")")
			(list (quote sql_type_expr) (list (quote session)
				(concat "__tsql_error_" name)) (if (equal? name "message") "NVARCHAR" "INT") nil))
		(parser '((atom "@@IDENTITY" true)) (tsql_identity_expr false))

		(parser (define token (capture tsql_variable_name))
			(parameter_expression (nth token 1) (substr (car token) 1)))

		(parser '((define function_owner tsql_identifier) "." (define fn tsql_identifier)
			"(" (define args (* (or
				(parser (atom "DEFAULT" true) (list "scalar-parameter-default"))
				tsql_expression) ",")) ")")
			(if (not (equal?? function_owner "dbo")) (error "unsupported scalar function schema: " function_owner)
				(equal? (lexical_context "module_kind") "VIEW")
				(list 'tsql_view_scalar_call schema fn args)
				script_syntax (list "scalar-function-call" fn args)
				(and function_body (or (frontend_options "__tsql_logical_query")
					(frontend_options "__tsql_logical_update")) (nil? access_policy))
				(list 'tsql_deferred_scalar_call schema fn args)
				(and function_body (not (or (frontend_options "__tsql_logical_query")
					(frontend_options "__tsql_logical_update"))))
				(error "scalar function call requires a relational plan")
				(nil? access_policy) (error "scalar function call requires an authorized SQL plan")
				(tsql_scalar_function_call schema fn args policy frontend_options tx false)))

		(parser '((define fn tsql_identifier_unquoted) "(" (define args (* tsql_expression ",")) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") (tsql_window_expr (toUpper fn) args _over))

		(parser '((define fn tsql_identifier_unquoted) "(" (define args (* tsql_expression ",")) ")")
			(begin (define d (sql_aggregates (toUpper fn)))
				(if (not (nil? d))
					'('aggregate (car args) (car d) (cadr d))
					(tsql_builtin_expr (toUpper fn) args))))

		(parser "?" (begin
			(define n (placeholder_counter "n"))
			(placeholder_counter "n" (+ n 1))
			(list (quote session) (concat "v" (string (+ n 1))))))
		tsql_hex_literal
		tsql_number
		tsql_text_literal
		tsql_column
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

	(define tsql_join_operand_source? (lambda (src) (match (source_relation src)
		((symbol tsql_pending_source_order) _names relation)
		(tsql_join_operand_source? (source_with_relation src relation))
		((symbol tsql_pending_nullable_join) _left _right) true
		((symbol tsql_pending_join_operand) _left _right) true
		_ false)))
	(define tsql_join_operand_aliases (lambda (sources)
		(merge (map sources (lambda (src) (match (source_relation src)
			((symbol tsql_pending_source_order) _names relation)
			(tsql_join_operand_aliases (list (source_with_relation src relation)))
			((symbol tsql_pending_nullable_join) _left right) (tsql_join_operand_aliases right)
			((symbol tsql_pending_join_operand) _left right) (tsql_join_operand_aliases right)
			_ (list (source_alias src))))))))
	(define tsql_join_export_order (lambda (sources)
		(reduce sources (lambda (ordered src) (begin
			(define names (match (source_relation src)
				((symbol tsql_pending_source_order) names _relation) names
				((symbol tsql_pending_nullable_join) _left right) (tsql_join_export_order right)
				((symbol tsql_pending_join_operand) _left right) (tsql_join_export_order right)
				_ (list (source_alias src))))
			(reduce names (lambda (current name)
				(if (has? current name) current (append current name))) ordered))) '())))
	(define tsql_with_join_export_order (lambda (src names)
		(source_with_relation src (list 'tsql_pending_source_order names
			(match (source_relation src)
				((symbol tsql_pending_source_order) _names relation) relation
				_ (source_relation src))))))
	(define tsql_attach_join_suffix (lambda (left suffix)
		(match suffix
			'((symbol tsql_pending_right_suffix) preserved predicate)
			(begin
				/* Preserve the complete right cloud before appending the nullable
				left unit. Its internal ONs cannot see that left unit's aliases. */
				(define order (merge (list (tsql_join_export_order left) (tsql_join_export_order preserved))))
				(define preserved (cons (tsql_with_join_export_order (car preserved) order) (cdr preserved)))
				(tsql_attach_join_suffix preserved (list (if (equal? (count left) 1)
					(match (car left) '(id database relation _ _join declarations)
						(list id database relation true predicate declarations))
					(list (source_alias (car left)) (source_schema (car left))
						(list 'tsql_pending_join_operand (tsql_join_operand_aliases preserved) left)
						true predicate nil)))))
			(cons src '()) (match (source_relation src)
				((symbol tsql_pending_nullable_join) _left right)
				(append left (source_with_relation src (list 'tsql_pending_nullable_join
					(tsql_join_operand_aliases left) right)))
				((symbol tsql_pending_join_operand) _left right)
				(append left (source_with_relation src (list 'tsql_pending_join_operand
					(tsql_join_operand_aliases left) right)))
				_ (append left src))
			_ (merge left suffix))))

	/* A joined table is an operand on either side of JOIN. Parentheses delimit
	that operand; they do not turn its aliases into a derived SELECT scope. */
	(define tsql_join_input (parser (define input tabledefs) input))
	(define tsql_join_suffix (parser (or
		(parser '((atom "RIGHT" true) (? (atom "OUTER" true)) (atom "JOIN" true)
			(define preserved tsql_join_input) (atom "ON" true) (define predicate tsql_expression))
			(list 'tsql_pending_right_suffix preserved predicate))
		(parser '((atom "CROSS" true) (atom "APPLY" true) (define r tabledef)) (list (list (source_alias r) (source_schema r)
			(list (quote tsql_pending_apply) (source_relation r)) false nil nil)))
		(parser '((atom "OUTER" true) (atom "APPLY" true) (define r tabledef)) (list (list (source_alias r) (source_schema r)
			(list (quote tsql_pending_apply) (source_relation r)) true nil nil)))
		(parser '((atom "LEFT" true) (? (atom "OUTER" true)) (atom "JOIN" true) (define r tabledef) (atom "ON" true) (define e tsql_expression)) (match r '(id schema tbl _ nil declarations) '('(id schema tbl true e declarations))))
		(parser '((atom "LEFT" true) (? (atom "OUTER" true)) (atom "JOIN" true)
			(define right tsql_join_input)
			(atom "ON" true) (define predicate tsql_expression))
			(match right
				(cons head children) (begin
					(if (and (not (source_outer? head)) (or (not (empty_list? children))
						(tsql_join_operand_source? head))) true
						(error "nested LEFT JOIN requires a preserved head and a joined right operand"))
					/* Only a flat LEFT chain is a candidate for the optional guard
					rewrite. Recursive operands retain their complete QB boundary. */
					(define recipe (if (and (nil? (find right tsql_join_operand_source? nil))
						(reduce children (lambda (valid child) (and valid (source_outer? child))) true))
						'tsql_pending_nullable_join 'tsql_pending_join_operand))
					(list (list (source_alias head) (source_schema head)
						(list recipe nil right) true predicate nil)))
				_ (error "nested LEFT JOIN requires a preserved head and a joined right operand")))
		(parser '((? (atom "INNER" true)) (atom "JOIN" true) (define r tsql_join_input) (atom "ON" true) (define e tsql_expression))
			(begin
				/* Associative inner joins share one predicate cloud. A right
				operand containing a nullable input instead keeps its complete
				query-block boundary, with a non-nullable enclosing INNER ON. */
				(if (and (equal? (count r) 1) (tsql_join_operand_source? (car r)))
					(list (source_with_join_expr (car r) e))
					(find r (lambda (src) (or (source_outer? src) (tsql_join_operand_source? src))) nil)
					(list (list (source_alias (car r)) (source_schema (car r))
						(list 'tsql_pending_join_operand nil r) false e nil))
					/* The enclosing ON sees the complete right input. Attach it after
					that input so the common binder sees every referenced alias. */
					(mapIndex r (lambda (index src)
						(if (equal? index (- (count r) 1))
							(match src '(id database relation _ join declarations)
								(list id database relation false (combine_where join e) declarations)) src))))))
		(parser '((? (atom "CROSS" true)) (atom "JOIN" true) (define r tsql_join_input)) r)
	)))
	/* Parentheses delimit joined-table operands at any recursive depth.
	Their native source aliases remain visible through the existing operand
	certificates; they never acquire a derived SELECT alias scope. */
	(define tabledefs (parser (or
		(parser '((define l tabledefs) (define r tsql_join_suffix)) (tsql_attach_join_suffix l r))
		(parser (define t tabledef) '(t))
		(parser '("(" (define grouped tabledefs) ")") (begin
			(if (and (equal? (count grouped) 1) (tsql_join_operand_source? (car grouped))) grouped
				(> (count grouped) 1) (list (list (source_alias (car grouped)) (source_schema (car grouped))
					(list 'tsql_pending_join_operand nil grouped) false true nil))
				(error "parenthesized source requires a joined table"))))
	)))
	(define tsql_from_sources (parser '(
		(define first tabledefs)
		(define rest (* (parser '("," (define next tabledefs)) next) empty)))
		(cons first rest)))

	/* Lateral visibility is a logical source fact. The common binder and
	decorrelator own dependency elimination; APPLY emits no execution code. */
	(define tsql_apply_source (lambda (src outer) (match src
		'(id database relation _outer _join declarations) (list id database
			/* APPLY exports consume wildcard declarations before the enclosing
			binder runs, just like CTE exports. Use the common view/catalog
			boundary before building that export directory. */
			(match (sql_expand_views relation policy planning_session tx)
				((symbol query-block) db sources fields condition group having order limit offset hidden stages facts)
				/* Freeze exported contracts in the producer's lexical SQL scope.
				The existing compiler callback disappears before physical lowering. */
				(make_query_block db sources
					(map_assoc (expand_query_block_fields sources fields) (lambda (_title expr)
						(if (star_expr? expr) expr
							(list (quote sql_compile_expr) (quote car) expr))))
					condition group having order limit offset hidden stages
					(qassoc_set facts (quote lateral) true))
				_ (error "APPLY currently requires a SELECT query"))
			outer nil nil))))

	/* SQL Server TABLES/COLUMNS describe one selected catalog. The native
	metadata relation remains shared with other frontends; only this logical
	projection maps its database namespace to TABLE_CATALOG / dbo. */
	(define tsql_information_schema_source (lambda (alias qualifier name) (begin
		(define kind (toLower name))
		(if (not (has? '("tables" "columns") kind))
			(error "unsupported T-SQL table metadata relation: " name)
			(begin
				(define owner (database_name (tsql_qualifier_database qualifier schema frontend_options) true))
				(if (nil? owner) (error "catalog database does not exist") true)
				(if policy (policy owner kind false) true)
				(define metadata_alias "__tsql_information_schema")
				(define column (lambda (name) (list 'get_column metadata_alias false name false)))
				(define fields (merge (map (information_schema_column_catalog "information_schema" kind)
					(lambda (declaration) (begin
						(define name (declaration "Field"))
						(list name (if (equal? name "table_catalog")
							(list 'sql_type_expr owner (list "NVARCHAR" '(128)) nil)
							(equal? name "table_schema")
							(list 'sql_type_expr "dbo" (list "NVARCHAR" '(128)) nil)
							(list 'sql_compile_column 'tsql_compile_column (column name)))))))))
				(define condition (list 'and
					(list 'equal? (column "table_schema") owner)
					(list tsql_metadata_visible? (tsql_actor_formula) owner true)
					(list 'not (list internal_column_name? (column "table_name")))))
				(list alias owner (make_query_block owner
					(list (list metadata_alias "information_schema" kind false nil nil)) fields
					condition nil nil nil nil nil '() '() '()) false nil nil))))))

	(define tsql_read_source (lambda (alias database name) (begin
		(define invocation_source (tsql_transition_binding database name frontend_options))
		(define resolved (tsql_resolve_relation database name frontend_options tx))
		(define database (car resolved))
		(define name (cadr resolved))
		(define canonical (tsql_relation_name database name tx))
		(planner_record_guard_condition
			(list (quote equal?) (list (quote tsql_relation_name) database name (quote tx)) canonical)
			planning_session)
		(if policy (policy database canonical false) true)
		(if (and (equal?? database "information_schema") (has? '("tables" "columns") (toLower canonical)))
			(tsql_information_schema_source alias nil canonical)
			(equal?? database "information_schema") (begin
				/* Other historical metadata views retain their existing contract;
				TABLES/COLUMNS above own the SQL Server catalog/schema projection.
				Keep the native relation logical; never copy metadata rows here. */
				(define kind (toLower canonical))
				(define entry (find (list
					(list "schemata" "schema_name" nil)
					(list "tables" "table_schema" "table_name")
					(list "columns" "table_schema" "table_name")
					(list "statistics" "table_schema" "table_name")
					(list "table_constraints" "table_schema" "table_name")
					(list "key_column_usage" "table_schema" "table_name")
					(list "referential_constraints" "constraint_schema" "table_name")
					(list "views" "table_schema" "table_name")
					(list "routines" "routine_schema" nil)
					(list "parameters" "specific_schema" nil)
					(list "triggers" "event_object_schema" "event_object_table")
					(list "check_constraints" "constraint_schema" nil))
					(lambda (entry) (equal? (car entry) kind)) nil))
				(if entry (begin
					(define metadata_alias "__tsql_information_schema")
					(define condition (list tsql_metadata_visible?
						(tsql_actor_formula)
						(list (quote get_column) metadata_alias false (cadr entry) false)
						(not (equal? kind "schemata"))))
					(define condition (if (nil? (nth entry 2)) condition
						(list (quote and) condition (list (quote not) (list (quote regexp_test)
							(list (quote get_column) metadata_alias false (nth entry 2) false) "^\\.")))))
					(list alias schema (make_query_block schema
						(list (list metadata_alias database canonical false nil nil))
						(list "*" (list (quote sql_compile_column) (quote tsql_compile_column)
							(list (quote get_column) nil false "*" false)))
						condition nil nil nil nil nil '() '() '()) false nil nil))
					(has? '("character_sets" "collations" "collation_character_set_applicability" "engines") kind)
					(list alias database canonical false nil nil)
					(error "unsupported T-SQL information schema relation: " canonical)))
			(list alias database canonical false nil
				(if invocation_source
					(set_assoc (sql_relation_declarations database canonical tx) "lifetime" "invocation") nil))))))
	/* Define legacy routing after both producers. The catalog builder above
	must not capture a still-uninitialized local tsql_read_source binding. */
	(define tsql_information_schema_read_source (lambda (alias qualifier name)
		(if (has? '("tables" "columns") (toLower name))
			(tsql_information_schema_source alias qualifier name)
			(nil? qualifier) (tsql_read_source alias
				(tsql_qualifier_database "information_schema" schema frontend_options) name)
			(error "unsupported qualified T-SQL information schema relation: " name))))

	/* This is the declared SQL relation name, not an address of the native
	inputs inside a view or virtual catalog. Explicit aliases hide it even
	when their spelling happens to equal the original name. */
	(define tsql_named_source (lambda (src namespace relation visible)
		(source_with_binding_name src (source_schema src) namespace relation visible)))

	/* System relations are ordinary derived logical queries over the shared
	metadata source. No SQL text matching or independent catalog executor. */
	(define tsql_catalog_source (lambda (alias qualifier name) (begin
		(define owner (if (nil? qualifier) schema (tsql_qualifier_database qualifier schema frontend_options)))
		(define owner (database_name owner true))
		(if (nil? owner) (error "catalog database does not exist") true)
		(if (has? tsql_virtual_catalog_names (toLower name)) true (error "unsupported system relation: sys." name))
		/* The physical metadata namespace is not the requested database's
		authority. The global databases view retains its per-row access domain. */
		(if (and policy (not (equal?? name "databases"))) (policy owner name false) true)
		(define query (if (has? '("columns" "all_columns") (toLower name))
			(tsql_catalog_columns_query owner (equal?? name "all_columns"))
			(has? '("database_permissions" "database_principals" "server_principals" "database_role_members") (toLower name))
			(tsql_security_catalog_query owner (toLower name))
			(equal?? name "databases") (begin
				(define source_alias "__tsql_databases")
				(define database_name_expr (list (quote get_column) source_alias false "schema_name" false))
				(define query (make_query_block owner
					(list (list source_alias "information_schema" "schemata" false nil nil))
					(list "name" (list (quote sql_type_expr) database_name_expr (list "NVARCHAR" (list 128)) (list "utf8mb4_general_ci" 2))
						"database_id" (list (quote sql_type_expr)
							(list (quote tsql_database_id) database_name_expr) "INT" nil)
						"is_auto_close_on" (list (quote sql_type_expr) false "BIT" nil)
						"is_auto_shrink_on" (list (quote sql_type_expr) false "BIT" nil)
						"recovery_model_desc" (list (quote sql_type_expr) "SIMPLE" (list "NVARCHAR" '(60)) nil)
						"is_trustworthy_on" (list (quote sql_type_expr)
							(list (quote tsql_database_trustworthy) database_name_expr) "BIT" nil))
					(list (quote equal?) (list (quote tsql_has_dbaccess) database_name_expr
						(tsql_actor_formula)) 1)
					nil nil nil nil nil '() '() '()))
				query) (begin
				(tsql_catalog_objects_query owner planning_session tx))))
		(list alias owner query false nil nil))))
	(define tsql_catalog_owner (parser (or
		(atom "sys" true) (atom "[sys]" true)
		(parser '("\"" (atom "sys" true) "\"") "sys"))))
	(define tsql_catalog_prefix (parser (or
		(parser '((define catalog tsql_identifier) "." tsql_catalog_owner ".") catalog)
		(parser '(tsql_catalog_owner ".") nil))))
	(define tsql_information_schema_owner (parser (or
		(atom "information_schema" true) (atom "[information_schema]" true)
		(parser '("\"" (atom "information_schema" true) "\"") "information_schema"))))
	(define tsql_information_schema_prefix (parser (or
		(parser '((define catalog tsql_identifier) "." tsql_information_schema_owner ".") catalog)
		(parser '(tsql_information_schema_owner ".") nil))))
	(define tabledef (parser (or
		(parser '((define owner tsql_identifier) "." (define fn tsql_identifier)
			"(" (define arguments (* (or
				(parser (atom "DEFAULT" true) (list "scalar-parameter-default")) tsql_expression) ",")) ")"
			(? (atom "AS" true)) (? (define alias tsql_identifier)))
			(begin
				(if (equal?? owner "dbo") true
					(error "unsupported table function schema: " owner))
				(list (coalesceNil alias fn) schema
					(list 'tsql_pending_table_call fn arguments (coalesceNil alias fn)) false nil nil)))
		(parser '((define variable tsql_variable_name)
			(? (atom "AS" true)) (? (define alias tsql_identifier)))
			(if script_syntax
				(list (coalesceNil alias variable) schema (list 'tsql_local_relation_name variable) false nil nil)
				(tsql_relational_table_source schema variable alias frontend_options)))

		(parser '((define catalog tsql_information_schema_prefix) (define name tsql_identifier)
			(atom "AS" true) (define alias tsql_identifier)) (tsql_named_source (tsql_information_schema_read_source alias catalog name) "information_schema" name false))
		(parser '((define catalog tsql_information_schema_prefix) (define name tsql_identifier)
			(define alias tsql_identifier)) (tsql_named_source (tsql_information_schema_read_source alias catalog name) "information_schema" name false))
		(parser '((define catalog tsql_information_schema_prefix) (define name tsql_identifier))
			(tsql_named_source (tsql_information_schema_read_source name catalog name) "information_schema" name true))

		(parser '((define catalog tsql_catalog_prefix) (define name tsql_identifier)
			(atom "AS" true) (define alias tsql_identifier)) (tsql_named_source (tsql_catalog_source alias catalog name) "sys" name false))
		(parser '((define catalog tsql_catalog_prefix) (define name tsql_identifier)
			(define alias tsql_identifier)) (tsql_named_source (tsql_catalog_source alias catalog name) "sys" name false))
		(parser '((define catalog tsql_catalog_prefix) (define name tsql_identifier))
			(tsql_named_source (tsql_catalog_source name catalog name) "sys" name true))

		/* VALUES is syntax data, lowered to the same logical SELECT/UNION
		blocks as other inline relations. The common planner owns joins,
		grouping, ordering, limits and physical execution. */
		(parser '("(" (atom "VALUES" true)
			(define rows (+ (parser '("(" (define row (+ tsql_expression ",")) ")") row) ","))
			")" (? (atom "AS" true)) (define id tsql_identifier)
			"(" (define aliases (+ tsql_identifier ",")) ")")
			(list id schema (list (quote tsql_pending_values) aliases rows) false nil nil))

		(parser '((atom "(" true) (define query tsql_select) (atom ")" true) (atom "AS" true) (define id tsql_identifier) "(" (define aliases (+ tsql_identifier ",")) ")")
			(list id schema (list (quote tsql_pending_aliases) query aliases) false nil nil))
		(parser '((atom "(" true) (define query tsql_select) (atom ")" true) (define id tsql_identifier) "(" (define aliases (+ tsql_identifier ",")) ")")
			(list id schema (list (quote tsql_pending_aliases) query aliases) false nil nil))
		(parser '((atom "(" true) (define query tsql_select) (atom ")" true) (atom "AS" true) (define id tsql_identifier)) '(id schema query false nil nil))
		(parser '((atom "(" true) (define query tsql_select) (atom ")" true) (define id tsql_identifier)) '(id schema query false nil nil))

		(parser '((define owner tsql_schema_identifier) (define tbl tsql_identifier) (atom "AS" true) (define id tsql_identifier) (? tsql_table_hints)) (tsql_named_source (tsql_read_source id (tsql_qualifier_database owner database frontend_options) tbl) "dbo" tbl false))
		(parser '((define owner tsql_schema_identifier) (define tbl tsql_identifier) (define id tsql_identifier) (? tsql_table_hints)) (tsql_named_source (tsql_read_source id (tsql_qualifier_database owner database frontend_options) tbl) "dbo" tbl false))
		(parser '((define owner tsql_schema_identifier) (define tbl tsql_identifier) (? tsql_table_hints)) (tsql_named_source (tsql_read_source tbl (tsql_qualifier_database owner database frontend_options) tbl) "dbo" tbl true))
		(parser '((define tbl tsql_identifier) (atom "AS" true) (define id tsql_identifier) (? tsql_table_hints)) (list id schema (list (quote tsql_source_name) tbl false) false nil nil))
		(parser '((define tbl tsql_identifier) (define id tsql_identifier) (? tsql_table_hints)) (list id schema (list (quote tsql_source_name) tbl false) false nil nil))
		(parser '((define tbl tsql_identifier) (? tsql_table_hints)) (list tbl schema (list (quote tsql_source_name) tbl true) false nil nil))
	)))

	(define from nil)

	(define condition nil)

	(define group nil)

	(define having nil)

	(define order nil)

	(define limit nil)

	(define offset nil)

	/* A nested set expression is a normal derived relation. This preserves its
	duplicate/window boundary using the shared binder and planner. */
	(define tsql_query_from_source (lambda (query) (make_query_block schema
		(list (list "__tsql_query_source" schema query false nil nil))
		(list "*" (list (quote sql_compile_column) (quote tsql_compile_column)
			(list (quote get_column) "__tsql_query_source" false "*" false)))
		true nil nil nil nil nil '() '() '())))

	/* UNION is left associative. Flatten only an adjacent union with the same
	mode and no intervening result window; every branch keeps its own TOP. */
	(define tsql_union_query (lambda (mode left right) (begin
		(define branches (match left
			((symbol union-block) previous branches order limit offset facts)
			(if (and (equal? previous mode) (nil? order) (nil? limit) (nil? offset)
				(empty_list? facts)) (append branches right)
				(list left right))
			_ (list left right)))
		(list (quote union-block) mode branches nil nil nil '()))))

	(define tsql_inner_select_kind (lambda (sym) (begin
		(if (equal?? sym "inner_select")
			(quote inner_select)
			(if (equal?? sym "inner_select_in")
				(quote inner_select_in)
				(if (equal?? sym "inner_select_exists")
					(quote inner_select_exists)
					(match sym
						(symbol inner_select) (quote inner_select)
						'inner_select (quote inner_select)
						'(quote inner_select) (quote inner_select)
						(symbol inner_select_in) (quote inner_select_in)
						'inner_select_in (quote inner_select_in)
						'(quote inner_select_in) (quote inner_select_in)
						(symbol inner_select_exists) (quote inner_select_exists)
						'inner_select_exists (quote inner_select_exists)
						'(quote inner_select_exists) (quote inner_select_exists)
						_ nil)
				)
			)
		)
	)))

	(define tsql_expr_contains_inner_select (lambda (expr) (match expr
		(cons sym args) (or
			(not (nil? (tsql_inner_select_kind sym)))
			(reduce args (lambda (a b) (or a (tsql_expr_contains_inner_select b))) false))
		_ false
	)))

	(define tsql_dataset_contains_inner_select (lambda (dataset)
		(reduce dataset (lambda (a b) (or a (tsql_expr_contains_inner_select b))) false)
	))

	(define tsql_values_row_query (lambda (database columns row)
		(make_query_block database '()
			(merge (map (produceN (count columns)) (lambda (i) (list (nth columns i) (nth row i)))))
			true nil nil nil nil nil '() '() '())))

	(define tsql_values_to_select_query (lambda (schema2 coldesc datasets)
		(match datasets
			(cons only '()) (tsql_values_row_query schema2 coldesc only)
			(cons first rest) (reduce rest (lambda (acc row)
				(tsql_union_query (quote all) acc (tsql_values_row_query schema2 coldesc row)))
				(tsql_values_row_query schema2 coldesc first))
			_ (error "INSERT VALUES requires at least one row")
		)
	))

	(define tsql_inline_values_query (lambda (aliases rows) (begin
		(define width (count aliases))
		(if (reduce rows (lambda (valid row) (and valid (equal? (count row) width))) true)
			true (error "inline VALUES requires equal row and alias widths"))
		(if (equal? width (count (merge_unique (list (map aliases toLower))))) true
			(error "duplicate inline VALUES column alias"))
		/* The bounded syntax matrix supplies every column's declaration.
		Only formulas and ordinary logical queries enter planning. */
		(define columns (map (produceN width) (lambda (index)
			(tsql_common_value_infos (map rows (lambda (row)
				(sql_expr_info '() (nth row index)))) "INT"))))
		(define branches (map (produceN (count rows)) (lambda (row_index)
			(tsql_values_row_query schema aliases (map columns (lambda (column) (begin
				(define info (nth column row_index))
				(list (quote sql_type_expr) (sql_info_formula info)
					(sql_info_datatype info) (sql_info_collation info)))))))))
		(match branches (cons only '()) only
			_ (list (quote union-block) (quote all) branches nil nil nil '())))))

	/* Export names belong to SELECT syntax, not the rendered expression title.
	UNION uses its first branch's names; an explicit CTE column list supplies
	the contract for otherwise unnamed expressions. */
	(define tsql_cte_unnamed_output? (lambda (query) (match query
		((symbol query-block) _db _sources _fields _where _group _having _order _limit _offset _hidden _stages facts)
		(qassoc_get facts (quote tsql_parser_unnamed_output) false)
		((symbol union-block) _mode branches _order _limit _offset _facts)
		(tsql_cte_unnamed_output? (car branches))
		_ false)))
	(define tsql_select_logical_facts (lambda (facts)
		(filter facts (lambda (fact) (not (has?
			(list 'tsql_parser_query_hint 'tsql_parser_unnamed_output 'tsql_column_catalog
				'tsql_virtual_catalog 'tsql_parser_top_all 'tsql_parser_set_tail_top)
			(car fact)))))))
	(define tsql_check_nested_order (lambda (order has_window)
		(if (and (not (empty_list? (coalesceNil order '()))) (not has_window))
			(error "ORDER BY in a view, derived table or CTE requires TOP or OFFSET") true)))

	/* Right-associated nullable inputs may themselves contain a nullable
	operand. Emit their lexical certificates inside-out, before binding any
	names; the shared binder packages the same boundaries in that order. */
	(define tsql_expand_join_operands (lambda (sources)
		(reduce sources (lambda (state src) (begin
			(define src (match (source_relation src)
				((symbol tsql_pending_source_order) _names relation) (source_with_relation src relation)
				_ src))
			(if (tsql_join_operand_source? src)
				(match (source_relation src) '(recipe left right) (begin
					(define expanded (tsql_expand_join_operands right))
					(define head (car (car expanded)))
					(define children (cdr (car expanded)))
					(define scope (list (source_alias head) (source_aliases children) left))
					(list (merge (list (car state) (cons (list (source_alias head) (source_schema head)
						(source_relation head) (source_outer? src) (source_join_expr src) (source_declarations head)) children)))
						(merge (list (cadr state) (cadr expanded)
							(if (equal? recipe 'tsql_pending_nullable_join) (list scope) '())))
						(merge (list (nth state 2) (nth expanded 2)
							(if (equal? recipe 'tsql_pending_join_operand) (list scope) '()))))))
				(list (append (car state) src) (cadr state) (nth state 2)))))
			(list '() '() '()))))

	/* Parser-only source recipes are bound once at the statement's compile
	boundary. Local definitions become ordinary derived logical queries; native
	names retain the existing catalog, policy and schema-guard path. Nothing here
	is retained by a row callback or published as a separate relation catalog. */
	(define tsql_finalize_sources (lambda (node kind bindings blocked) (begin
		(define definition_scope (or (equal? kind "definition") (equal? kind "definition_source")))
		(define relation_scope (or definition_scope (equal? kind "relation")))
		(define child_kind (if definition_scope "definition" "query"))
		(define relation_kind (if definition_scope "definition" "relation"))
		(define source_kind (if definition_scope "definition_source" "source"))
		(match (if (equal? kind "definition_source") "source" kind)
			"definitions" (begin
				(define names (map node car))
				(if (equal? (count names) (count (merge_unique (list (map names toLower))))) true
					(error "duplicate CTE declaration name"))
				(reduce node (lambda (available definition) (match definition
					'(name aliases body) (begin
						(if (and (nil? aliases) (tsql_cte_unnamed_output? body))
							(error "CTE output expressions require column names") true)
						(define bound (tsql_finalize_sources body "definition" available names))
						/* CTE exports consume wildcard/name contracts before the outer
						query reaches its binder. Acquire them through the same common
						view/catalog boundary before expanding that export directory. */
						(define exported (tsql_finalize_sources (sql_expand_views bound policy planning_session tx) "exports" available names))
						(define titles (queryplan_result_titles exported))
						(define output_names (coalesceNil aliases titles))
						(if (equal? (count output_names) (count titles)) true
							(error "CTE output alias count does not match query column count"))
						(if (equal? (count output_names) (count (merge_unique (list (map output_names toLower))))) true
							(error "duplicate CTE output column name"))
						(cons (list name (if (nil? aliases) exported
							(tsql_apply_derived_column_aliases exported aliases))) available)))) bindings))
			"source" (match node '(id database relation outer join declarations) (begin
				(define resolved (match relation
					((symbol tsql_pending_table_call) name arguments alias)
					(list id database
						(if (or definition_scope script_syntax (nil? access_policy))
							(list 'tsql_view_table_call database name arguments alias)
							(tsql_table_function_call database name
								(tsql_finalize_sources arguments child_kind bindings blocked)
								alias policy planning_session tx)) outer nil nil)
					/* SELECT consumes this parser-only wildcard-order certificate
					while expanding a whole join cloud. DML binds sources here one
					at a time; the underlying relation has the same native identity,
					declarations and nullable position, without a derived-query scope. */
					((symbol tsql_pending_source_order) _names inner)
					(tsql_finalize_sources (list id database inner outer nil declarations)
						source_kind bindings blocked)
					((symbol tsql_source_name) name qualified_visible) (begin
						(define local (find bindings (lambda (entry) (equal?? (car entry) name)) nil))
						(if local (list id database (cadr local) outer nil nil)
							(find blocked (lambda (entry) (equal?? entry name)) nil)
							(error "recursive or forward CTE reference is unsupported: " name)
							(begin
								(define native (tsql_named_source (tsql_read_source id database name) "dbo" name qualified_visible))
								(list id (source_schema native) (source_relation native) outer nil (source_declarations native)))))
					((symbol tsql_pending_apply) inner)
					(tsql_apply_source (tsql_finalize_sources (list id database inner outer nil nil)
						source_kind bindings blocked) outer)
					((symbol tsql_pending_nullable_join) _left _right)
					(error "nested LEFT JOIN requires a SELECT query block")
					((symbol tsql_pending_join_operand) _left _right)
					(error "nested JOIN operand requires a SELECT query block")
					_ (list id database (tsql_finalize_sources relation relation_kind bindings blocked) outer nil (source_declarations node))))
				(list (source_alias resolved) (source_schema resolved) (source_relation resolved)
					(source_outer? resolved) (tsql_finalize_sources join child_kind bindings blocked) (source_declarations resolved))))
			_ (match node
				((symbol get_column) ((symbol qualified-relation) database namespace relation) relation_ic column column_ic)
				(list 'get_column (list 'qualified-relation
					(if (nil? database) nil (tsql_qualifier_database database schema frontend_options))
					namespace relation) relation_ic column column_ic)
				((symbol quote) _literal) node
				((symbol tsql_pending_with) definitions body) (begin
					(define available (tsql_finalize_sources definitions "definitions" bindings blocked))
					(tsql_finalize_sources body kind available '()))
				((symbol tsql_pending_aliases) body aliases)
				(tsql_apply_derived_column_aliases
					(tsql_finalize_sources (tsql_finalize_sources body kind bindings blocked)
						"exports" bindings blocked) aliases)
				((symbol tsql_pending_values) aliases rows)
				(tsql_inline_values_query aliases (tsql_finalize_sources rows child_kind bindings blocked))
				((symbol query-block) db sources fields condition group having order limit offset hidden stages facts)
				(begin
					(if (qassoc_get facts (quote tsql_select_assignments) false)
						(error "SELECT assignment is only allowed as a standalone statement") true)
					(define top_all (qassoc_get facts 'tsql_parser_top_all false))
					(if relation_scope (tsql_check_nested_order order
						(or top_all (not (nil? limit)) (not (nil? offset)))) true)
					(if (and definition_scope (qassoc_get facts 'tsql_parser_query_hint false))
						(error "OPTION is not allowed in a CTE definition") true)
					(define logical_facts (tsql_select_logical_facts facts))
					(define logical_facts (if (and relation_scope top_all (not (empty_list? (coalesceNil order '()))))
						(qassoc_set logical_facts 'validate_order_only true) logical_facts))
					/* A literal object-ID equality implies an address restriction only
					for this direct, nonnullable catalog input. Other truth contexts
					and joins retain the general catalog query unchanged. */
					(define scoped_sources (if (equal? (count sources) 1)
						(map sources (lambda (source) (tsql_catalog_restrict_source source condition frontend_options))) sources))
					(define export_order (qassoc_get logical_facts 'source_export_order
						(tsql_join_export_order scoped_sources)))
					(define expanded (tsql_expand_join_operands scoped_sources))
					(define logical_facts (if (equal? export_order (source_aliases (car expanded))) logical_facts
						(qassoc_set logical_facts 'source_export_order export_order)))
					(define logical_facts (if (empty_list? (cadr expanded)) logical_facts
						(qassoc_set logical_facts 'nullable_join_scopes (cadr expanded))))
					(define logical_facts (if (empty_list? (nth expanded 2)) logical_facts
						(qassoc_set logical_facts 'join_operand_scopes (nth expanded 2))))
					(define scoped_sources (car expanded))
					(define bound_sources (if (equal? kind "exports") scoped_sources
						(map scoped_sources (lambda (source) (tsql_finalize_sources source source_kind bindings blocked)))))
					(define export_sources (if (not (nil? (qassoc_get logical_facts 'source_export_order nil)))
						(map (qassoc_get logical_facts 'source_export_order '()) (lambda (name)
							(find bound_sources (lambda (src) (equal? (source_alias src) name)) nil)))
						bound_sources))
					(make_query_block db bound_sources
						(if (equal? kind "exports") (expand_query_block_fields export_sources fields)
							(tsql_finalize_sources fields child_kind bindings blocked))
						(tsql_finalize_sources condition child_kind bindings blocked)
						(tsql_finalize_sources group child_kind bindings blocked)
						(tsql_finalize_sources having child_kind bindings blocked)
						(tsql_finalize_sources order child_kind bindings blocked)
						limit offset hidden stages logical_facts))
				((symbol union-block) mode branches order limit offset facts) (begin
					(define tail_owns_order (and relation_scope (qassoc_get facts 'tsql_parser_set_tail_top false)))
					(if relation_scope (tsql_check_nested_order order
						(or tail_owns_order (not (nil? limit)) (not (nil? offset)))) true)
					(if (and tail_owns_order (not (nil? offset))) (error "TOP cannot be combined with OFFSET") true)
					/* A nested set's trailing ORDER belongs to the final SELECT's
					TOP. Its names resolve against that branch, including inputs
					that are absent from the UNION output. */
					(define tail_index (- (count branches) 1))
					(define branches (if tail_owns_order (mapIndex branches (lambda (i branch)
						(if (equal? i tail_index)
							(make_query_block (qb_schema branch) (qb_sources branch) (qb_fields branch)
								(qb_where branch) (qb_group branch) (qb_having branch) order
								(qb_limit branch) (qb_offset branch) (qb_hidden branch) (qb_stages branch) (qb_facts branch))
							branch))) branches))
					(make_union_block mode
						(tsql_union_contracts schema (map branches (lambda (branch)
							(tsql_finalize_sources branch kind bindings blocked))) policy planning_session tx)
						(if tail_owns_order nil (tsql_finalize_sources order child_kind bindings blocked))
						limit offset (tsql_select_logical_facts facts)))
				(cons _head _tail) (map node (lambda (item) (tsql_finalize_sources item child_kind bindings blocked)))
				_ node)))))

	(define tsql_select_prefix (parser (? (parser '(
		(atom "TOP" true)
		(define amount (or
			(parser '("(" (define count tsql_int) ")") count)
			(parser '("(" (define expression tsql_expression) ")") expression)
			tsql_int))
		(? (define percent (atom "PERCENT" true)))
	) (if percent (begin
			(define literal (match amount ((symbol sql_type_expr) value _type _collation) value _ amount))
			(if (and (number? literal) (equal? literal 100)) (list nil true)
				(error "TOP PERCENT currently requires a literal 100")))
			(number? amount) (if (< amount 0) (error "TOP requires a nonnegative count") (list amount false))
			(tsql_scalar_expression_allowed? amount) (list (list 'sql_compile_expr 'tsql_compile_row_count amount) false)
			(error "TOP requires an invocation scalar"))))))

	(define tsql_select_head (parser '(
		(atom "SELECT" true)

		(define distinct (or (parser (atom "DISTINCT" true) true)
			(parser (? (atom "ALL" true)) false)))
		(define top tsql_select_prefix)
		(define cols (+ (or
			(parser '((define target tsql_variable_name) "=" (define expression tsql_expression))
				(list (concat "__tsql_assignment:" target) expression false target))
			/* Assignment aliases must precede expression comparisons. */
			(parser '((define title (or tsql_identifier tsql_string)) "=" (define e tsql_expression)) '(title e))
			(parser (define qualified tsql_qualified_star) (list "*" qualified))
			(parser "*" (list "*" (list (quote sql_compile_column) (quote tsql_compile_column) (list (quote get_column) nil false "*" false))))
			(parser '((define tbl tsql_identifier_quoted) "." "*")
				(list "*" (list (quote sql_compile_column) (quote tsql_compile_column) '('get_column tbl true "*" false))))
			(parser '((define tbl tsql_identifier_unquoted) "." "*")
				(list "*" (list (quote sql_compile_column) (quote tsql_compile_column) '('get_column tbl true "*" false))))
			(parser '((define e tsql_expression) (atom "AS" true) (define title tsql_identifier)) '(title e))
			(parser '((define e tsql_expression) (atom "AS" true) (define title tsql_string)) '(title e))

			(parser '((define e tsql_expression) (define title tsql_identifier)) '(title e))
			(parser '((define e tsql_expression) (define title tsql_string)) '(title e))

			(parser (define captured (capture tsql_expression)) (begin
				(define expr (nth captured 1))
				(define named (match expr
					((symbol sql_compile_column) _compiler ((symbol get_column) _alias _alias_ic _column _column_ic)) true
					((symbol get_column) _alias _alias_ic _column _column_ic) true
					_ false))
				/* Only this syntax branch needs an unnamed-output marker. */
				(if named (list (extract_title_or_sql captured) expr)
					(list (extract_title_or_sql captured) expr false))))
		) ","))
	) (list distinct top cols)))

	(define tsql_select_tail (parser '(
		(?
			(atom "FROM" true)
			(define from tsql_from_sources)
		)
		(define condition (or (parser '(
			(atom "WHERE" true)
			(define condition2 tsql_expression)
		) condition2) (empty true)))

		(?
			(atom "GROUP" true)
			(atom "BY" true)
			(define group (+
				tsql_expression
				(atom "," true)
			))
		)
		(?
			(atom "HAVING" true)
			(define having tsql_expression)
		)

	) (list from condition group having)))

	(define tsql_select_core (parser '(
		(define head tsql_select_head)
		(define tail tsql_select_tail)
	) (begin
			(define distinct (car head))
			(define top (cadr head))
			(define cols (nth head 2))
			(define from (car tail))
			(define condition (cadr tail))
			(define group (nth tail 2))
			(define having (nth tail 3))
			(define assignments (filter cols (lambda (column) (equal? (count column) 4))))
			(if (and (not (empty_list? assignments)) (not (equal? (count assignments) (count cols))))
				(error "SELECT cannot mix variable assignments and result columns") true)
			(define targets (map assignments (lambda (column) (nth column 3))))
			(if (equal? (count targets) (count (merge_unique (list targets)))) true
				(error "duplicate SELECT assignment target"))
			(define fields (merge (map cols (lambda (column) (slice column 0 2)))))
			(define unnamed_output (not (nil? (find cols (lambda (column) (equal? (count column) 3)) nil))))
			(define projected_exprs (extract_assoc fields (lambda (_title expr) expr)))
			(define sources (if (nil? from) '() (merge from)))

			(define distinct_preserved_by_group (and distinct
				(and (not (nil? group))
					(and (not (empty_list? group))
						(reduce group (lambda (preserved expr)
							(and preserved (contains? projected_exprs expr))) true)))))
			(list (quote query-block) schema sources fields condition
				(if (and distinct (not distinct_preserved_by_group)) projected_exprs group)
				having
				nil (if top (car top) nil) nil '() '()
				(merge (list
					/* The common name binder resolves all joined/derived sources.
					HAVING names belong to the input scope in this frontend. */
					(list (list (quote having_input_scope) true))
					(if (and (not (nil? top)) (cadr top)) (list (list 'tsql_parser_top_all true)) '())
					(if (empty_list? targets) '() (list (list (quote tsql_select_assignments) targets)))
					(if distinct (list (list (quote select_distinct) true)) '())
					(if unnamed_output (list (list (quote tsql_parser_unnamed_output) true)) '())))))))

	/* Parentheses delimit query expressions. Keep this parser-only bit until
	the trailing window has been attached; it never enters logical planning. */
	(define tsql_select_branch (parser (or
		(parser (define query tsql_select_core) (list query false))
		(parser '("(" (define query tsql_select) ")") (list query true)))))
	(define tsql_select_window (lambda (query order suffix_stage parenthesized set_tail_top) (begin
		(if (and suffix_stage (nil? order)) (error "OFFSET requires ORDER BY") true)
		(if (and (nil? order) (nil? suffix_stage)) query (begin
			/* An outer window over a bracketed query is an ordinary derived source.
			Its inner TOP/order remains a semantic boundary for the shared planner. */
			(define query (if parenthesized (tsql_query_from_source query) query))
			(match query
				((symbol query-block) database sources fields condition group having previous_order limit offset hidden stages facts)
				(begin
					(if (and suffix_stage (or (not (nil? limit)) (qassoc_get facts 'tsql_parser_top_all false)))
						(error "TOP cannot be combined with OFFSET") true)
					(make_query_block database sources fields condition group having order
						(if suffix_stage (car suffix_stage) limit)
						(if suffix_stage (cadr suffix_stage) offset) hidden stages facts))
				((symbol union-block) mode branches previous_order limit offset facts)
				(list (quote union-block) mode branches order
					(if suffix_stage (car suffix_stage) limit)
					(if suffix_stage (cadr suffix_stage) offset)
					(if set_tail_top (qassoc_set facts 'tsql_parser_set_tail_top true) facts))))))))
	(define tsql_select_set_terms (parser (* (parser '(
		(atom "UNION" true)
		(define mode (or (parser (atom "ALL" true) (quote all)) (parser empty (quote distinct))))
		(define right tsql_select_branch)
	) (list mode (car right) (cadr right))) empty true)))
	(define tsql_select_order (parser '(
		(?
			(atom "ORDER" true)
			(atom "BY" true)
			(define order (+
				(parser '(
					(define captured (capture tsql_expression))
					(? (atom "COLLATE" true) (define coll tsql_identifier))
					(define direction_desc (or
						(parser (atom "DESC" true) >)
						(parser (atom "ASC" true) <)
						(parser empty <)
					))
				) (begin
						/* A bare integer names a SELECT-list position. Keep other
						expressions typed, including casts and folded constants. */
						(define col (cadr captured))
						(define token (sql_trim (car captured)))
						(list (if coll
							(list (quote sql_compile_expr) (list (quote tsql_collation_compiler) coll) col)
							(regexp_test token "^[0-9]+$") (tsql_parse_integer token)
							col) direction_desc)))

				(atom "," true)
			))
		)
	) order))
	(define tsql_select (parser '(
		(define left tsql_select_branch)
		(define terms tsql_select_set_terms)
		(define order tsql_select_order)

		(define suffix_stage tsql_select_suffix)
		(? (define query_hint tsql_query_hints))

	) (begin
			(define tail (if (empty_list? terms) nil (nth terms (- (count terms) 1))))
			(define tail_top (and (not (nil? tail)) (not (nth tail 2)) (query_block? (cadr tail))
				(or (not (nil? (qb_limit (cadr tail)))) (qassoc_get (qb_facts (cadr tail)) 'tsql_parser_top_all false))))
			(define query (tsql_select_window
				(reduce terms (lambda (query term) (tsql_union_query (car term) query (cadr term))) (car left))
				order suffix_stage (and (empty_list? terms) (cadr left)) tail_top))
			(if query_hint (match query
				((symbol query-block) db sources fields condition group having order limit offset hidden stages facts)
				(make_query_block db sources fields condition group having order limit offset hidden stages
					(cons (list (quote tsql_parser_query_hint) true) facts))
				((symbol union-block) mode branches order limit offset facts)
				(make_union_block mode branches order limit offset
					(cons (list (quote tsql_parser_query_hint) true) facts))) query))))

	(define tsql_cte_definitions (parser '(
		(atom "WITH" true)
		(define definitions (+ (parser '(
			(define name tsql_identifier)
			(define aliases (? (parser '("(" (define names (+ tsql_identifier ",")) ")") names)))
			(atom "AS" true) "(" (define body tsql_select) ")"
		) (list name aliases body)) ","))
	) definitions))
	(define tsql_root_select (parser (or
		(parser '((define definitions tsql_cte_definitions) (define body tsql_select))
			(list (quote tsql_pending_with) definitions body))
		tsql_select)))

	/* A mutation target may name a FROM alias. Resolve that alias before checking
	write permission; the grammar never creates a fictitious table for it. */
	(define tsql_write_target (parser (or
		(parser (define variable tsql_variable_name)
			(list nil (list 'tsql_local_relation_name variable)))
		(parser '(
			(? (define target_schema tsql_schema_identifier))
			(define target_name tsql_identifier)
			(? tsql_table_hints)
		) (list (if (nil? target_schema) nil (tsql_qualifier_database target_schema schema frontend_options)) target_name)))))
	(define tsql_write_source (lambda (target sources) (match target
		'(target_schema ((symbol tsql_local_relation_name) name))
		(tsql_relational_table_source schema name nil frontend_options)
		'(target_schema target_name) (begin
			(define target_database (tsql_temporary_database (tsql_qualifier_database target_schema schema frontend_options) target_name frontend_options tx))
			(define target_relation (if (tsql_temporary_name? target_name)
				(cadr (tsql_resolve_relation target_database target_name frontend_options tx)) target_name))
			(define alias_match (if target_schema nil (find sources (lambda (src)
				(equal?? (source_alias src) target_name)) nil)))
			(define table_matches (filter sources (lambda (src) (and
				(source_is_base_table? src)
				(equal? (source_schema src) target_database)
				(equal?? (source_relation src) target_relation)))))
			(define table_match (if (empty_list? table_matches) nil
				(equal? (count table_matches) 1) (car table_matches)
				(find table_matches (lambda (src) (equal?? (source_alias src) target_name)) nil)))
			(if (and (nil? alias_match) (> (count table_matches) 1) (nil? table_match))
				(error "ambiguous write target: " target_name) true)
			(define source (coalesceNil alias_match table_match
				(tsql_named_source (list target_name target_database
					(tsql_table_name target_database target_name planning_session tx) false nil nil)
					"dbo" target_name true)))
			(if (source_is_base_table? source) source (error "write target must be a base table"))))))
	(define tsql_write_sources (lambda (target sources)
		(if (has? sources target) sources (cons target sources))))
	(define tsql_write_limit (parser (or
		(parser '((atom "TOP" true) "(" (define n tsql_int) ")")
			(if (< n 0) (error "DML TOP must be nonnegative") n))
		(parser empty nil))))
	(define tsql_update_assignment (parser '(
		(define column tsql_column)
		(define op (or (parser "+=" "add") (parser "-=" "subtract")
			(parser "*=" "multiply") (parser "/=" "divide") (parser "%=" "modulo")
			(parser "&=" "and") (parser "|=" "or") (parser "^=" "xor")
			(parser "=" "assign")))
		(define expression (or (parser (atom "DEFAULT" true) (quote sql_default)) tsql_expression))
	) (match column
			((symbol sql_compile_column) _compiler ((symbol get_column) qualifier _relation_ic title _column_ic))
			(list title qualifier op expression))))
	(define tsql_update_values (lambda (target cols) (begin
		(define names (map cols (lambda (col) (toLower (car col)))))
		(if (equal? (count names) (count (merge_unique (list names)))) true
			(error "duplicate UPDATE assignment"))
		(define assigned (merge (map cols (lambda (col) (match col
			'(title qualifier op expression) (begin
				/* Match the declared target using the shared SQL name binder.
				A multipart name is never shortened into a column or UDT member. */
				(if (or (nil? qualifier)
					(if (qualified_relation_ref? qualifier)
						(source_binding_name_matches? target qualifier true)
						(equal?? qualifier (source_alias target)))) true
					(error "assignment qualifier must name the write target"))
				(define declaration (if (local_relation_reference? (source_relation target))
					(find (local_relation_columns (source_relation target))
						(lambda (column) (equal?? (column "Field") title)) nil)
					(tsql_target_column_declaration (source_schema target) (source_relation target) title planning_session tx)))
				(if (nil? declaration) (error "unknown UPDATE column: " title)
					(equal? (declaration "Extra") "auto_increment") (error "IDENTITY column cannot be updated") true)
				(define value (if (equal?? expression (quote sql_default))
					(if (equal? op "assign")
						(tsql_default_assignment_expr declaration (source_schema target) policy planning_session tx)
						(error "DEFAULT requires simple assignment")) expression))
				(define old (list (quote sql_compile_column) (quote tsql_compile_column)
					(list (quote get_column) (source_alias target) false (declaration "Field") false)))
				(if (has? '("and" "or" "xor") op) (tsql_bit_assignment_type (declaration "Type")) true)
				(define assigned (match op "assign" value
					"add" (tsql_add_expr old value)
					"subtract" (tsql_fold_additive old (list "sub" value))
					_ (if (has? '("and" "or" "xor") op) (tsql_bit_expr op (list old value))
						(tsql_fold_multiplicative old (list op value)))))
				(list (declaration "Field") (tsql_assignment_expr assigned declaration))))))))
		(merge (list assigned (merge (map
			(filter (if (local_relation_reference? (source_relation target))
				(local_relation_columns (source_relation target))
				(tsql_write_declarations (source_schema target) (source_relation target) planning_session tx))
				(lambda (column) (and (not (column "Computed")) (equal?? (column "RawType") "ROWVERSION"))))
			(lambda (column) (list (column "Field") (tsql_rowversion_expr (source_schema target)))))))))))
	(define tsql_update (parser '(
		(atom "UPDATE" true)
		(define limit tsql_write_limit)
		(define target tsql_write_target)
		(atom "SET" true)
		(define cols (+ tsql_update_assignment ","))
		(? (atom "FROM" true) (define from (+ tabledefs ",")))
		(? (atom "WHERE" true) (define condition tsql_expression))
		(? tsql_query_hints)
	) (lambda cte_arguments (begin
			(define cte_bindings (if (empty_list? cte_arguments) '() (car cte_arguments)))
			(define sources (map (merge (coalesceNil from '())) (lambda (source)
				(tsql_finalize_sources source "source" cte_bindings '()))))
			(if (and (nil? (car target))
				(find cte_bindings (lambda (entry) (equal?? (car entry) (cadr target))) nil)
				(not (find sources (lambda (source) (and (source_is_base_table? source)
					(equal?? (source_alias source) (cadr target)))) nil)))
				(error "write target must be a base table, not a CTE") true)
			(define bound_condition (tsql_finalize_sources (coalesceNil condition true) "query" cte_bindings '()))
			(define target (tsql_write_source target sources))
			(define logical_update (frontend_options "__tsql_logical_update"))
			(if logical_update
				(if (local_relation_reference? (source_relation target)) true
					(error "scalar function UPDATE requires a declared local table"))
				(begin
					(if policy (policy (source_schema target) (source_relation target) true) true)
					(require_action (source_schema target) (source_relation target) "UPDATE")))
			(define values (tsql_update_values target (tsql_finalize_sources cols "query" cte_bindings '())))
			(if logical_update
				(make_query_block (source_schema target) (tsql_write_sources target sources)
					values bound_condition nil nil nil limit nil nil nil
					(list (list 'dml true) (list 'dml_count_matched true)
						(list 'local_update_target (source_relation target))
						(list 'dml_target_alias (source_alias target))))
				(if describe (tsql_describe_dml (source_schema target) (tsql_write_sources target sources)
					values bound_condition)
					(tsql_build_dml_plan (source_schema target) (source_relation target) (source_alias target)
						(tsql_write_sources target sources) values
						bound_condition nil limit nil planning_session tx)))))))
	(define tsql_delete (parser '(
		(atom "DELETE" true)
		(define limit tsql_write_limit)
		(? (atom "FROM" true))
		(define target tsql_write_target)
		(? (atom "FROM" true) (define from (+ tabledefs ",")))
		(? (atom "WHERE" true) (define condition tsql_expression))
		(? tsql_query_hints)
	) (lambda cte_arguments (begin
			(define cte_bindings (if (empty_list? cte_arguments) '() (car cte_arguments)))
			(define sources (map (merge (coalesceNil from '())) (lambda (source)
				(tsql_finalize_sources source "source" cte_bindings '()))))
			(if (and (nil? (car target))
				(find cte_bindings (lambda (entry) (equal?? (car entry) (cadr target))) nil)
				(not (find sources (lambda (source) (and (source_is_base_table? source)
					(equal?? (source_alias source) (cadr target)))) nil)))
				(error "write target must be a base table, not a CTE") true)
			(define bound_condition (tsql_finalize_sources (coalesceNil condition true) "query" cte_bindings '()))
			(define target (tsql_write_source target sources))
			(if policy (policy (source_schema target) (source_relation target) true) true)
			(require_action (source_schema target) (source_relation target) "DELETE")
			(if describe (tsql_describe_dml (source_schema target) (tsql_write_sources target sources)
				(list "__prepare" (list (quote sql_type_expr) 1 "INT" nil)) bound_condition)
				(tsql_build_dml_plan (source_schema target) (source_relation target) (source_alias target)
					(tsql_write_sources target sources) nil bound_condition
					nil limit nil planning_session tx))))))

	(define tsql_truncate (parser '(
		(atom "TRUNCATE" true) (? (atom "TABLE" true))

		(? (define schema2 tsql_schema_identifier)) (define tbl tsql_identifier)
	) (lambda () (begin
			(if policy (policy (tsql_qualifier_database schema2 schema frontend_options) tbl true) true)
			(require_action (tsql_qualifier_database schema2 schema frontend_options) tbl "ALTER")
			/* Row DELETE cannot implement TRUNCATE's identity, trigger and FK
			contracts. Reject before planning until the native clear owner exists. */
			(list (quote error) "TRUNCATE TABLE is not supported: native clear semantics are unavailable")
	))))

	(define tsql_insert_literal_cell (parser (or
		(parser (atom "DEFAULT" true) (quote sql_default))
		(parser "?" (quote sql_insert_placeholder))
		tsql_literal)))

	(define tsql_insert_values_row (parser (or
		(parser '("(" (define dataset (* tsql_insert_literal_cell ",")) ")")
			(map dataset (lambda (value)
				(if (and (symbol? value) (equal?? value (quote sql_insert_placeholder)))
					(begin
						(define n (placeholder_counter "n"))
						(placeholder_counter "n" (+ n 1))
						(list (quote session) (concat "v" (string (+ n 1)))))
					value))))
		(parser '("(" (define dataset (* tsql_expression ",")) ")") dataset))))

	(define tsql_create_view (parser '(
		(define mode (or
			(parser '((atom "CREATE" true) (atom "OR" true) (atom "ALTER" true)) "replace")
			(parser (atom "CREATE" true) "error")
			(parser (atom "ALTER" true) "alter")))
		(atom "VIEW" true)
		(define ifnotexists (? (atom "IF" true) (atom "NOT" true) (atom "EXISTS" true)))
		(define target (or
			(parser '((define schema2 tsql_schema_identifier) (define id tsql_identifier)) '(schema2 id))
			(parser (define id tsql_identifier) '(nil id))))
		(define aliases (? (parser '("(" (define columns (+ tsql_identifier ",")) ")") columns)))
		(atom "AS" true)
		(define captured (capture tsql_root_select))
	) (lambda () (match target '(schema2 id) (begin
			(define view_schema (tsql_qualifier_database schema2 schema frontend_options))
			(if policy (policy view_schema id true) true)
			(require_action view_schema id (if (equal? mode "error") "CREATE VIEW" "ALTER"))
			(if (equal? mode "replace") (require_action view_schema id "CREATE VIEW") true)
			(if (and (equal? mode "alter") ifnotexists)
				(error "ALTER VIEW does not accept IF NOT EXISTS") true)
			(define query (sql_apply_view_column_aliases
				(tsql_finalize_sources (nth captured 1) "relation" '() '()) aliases))
			(if (or (tsql_temporary_name? id) (tsql_temporary_query? query))
				(error "persistent views cannot reference temporary relations") true)
			(if (tsql_view_has_metadata_scalar? query)
				(error "COL_LENGTH in stored views requires a consuming compile policy") true)
			/* Validate a transient bound copy. The declaration below retains
			only lexical function names and argument expressions. */
			(if script_syntax true (bind_query_names (tsql_expand_view_functions
				(sql_expand_views query policy planning_session tx) policy planning_session tx) '()))
			(tsql_declaration_formula (list (quote tsql_create_view_definition)
				(list (quote session) "__memcp_tx")
				view_schema id "tsql" (coalesceNil (lexical_context "module_source") (nth captured 0)) (list (quote quote) query)
				(if ifnotexists "ignore" mode))))))))

	/* This one prefix grammar consumes its final separator in every caller.
	Resolve only the default owner in a database-qualified name: MemCP has one
	native relation namespace per database, not a parallel SQL schema registry.
	The complete long form must match before validating its owner so ordinary
	two-part names can still backtrack to their existing native mapping. */
	(define tsql_schema_identifier (parser (or
		(parser '((define catalog tsql_identifier) "." (define owner tsql_identifier) ".")
			(if (equal?? owner "dbo") catalog (error "unsupported qualified owner: " owner)))
		(parser '((define catalog tsql_identifier) "." ".") catalog)
		(parser '((define owner tsql_identifier) ".")
			(if (equal?? owner "dbo") (quote tsql_default_owner) owner)))))
	(define tsql_expression3 (parser '(
		(define a tsql_expression4)
		(define terms (* (parser '(
			(define op (or (parser "+" "add") (parser "-" "sub")
				(parser "<<" "left") (parser ">>" "right")
				(parser "&" "and") (parser "|" "or") (parser "^" "xor")))
			(define value tsql_expression4)
		) (list op value)) empty true))
	) (reduce terms tsql_fold_additive a)))
	(define tsql_describe_dml (lambda (database sources values condition) (begin
		(define bound (tsql_describe_query (make_query_block database sources values condition
			nil nil nil nil nil '() '() '()) policy planning_session tx))
		(define fields (tsql_result_columns bound parameters))
		(if (list? fields) (list (quote quote) '()) (error "invalid prepared mutation contract")))))
	/* Assignment targets are frontend syntax until this statement boundary.
	Apply declared casts to ordinary logical projections, remove the parser fact,
	then let the shared planner own every source, filter, aggregate and order. */
	(define tsql_select_assignment_targets (lambda (query) (match query
		((symbol tsql_pending_with) _definitions body) (tsql_select_assignment_targets body)
		((symbol query-block) _db _sources _fields _condition _group _having _order _limit _offset _hidden _stages facts)
		(qassoc_get facts (quote tsql_select_assignments) nil)
		_ nil)))
	(define tsql_select_assignment_query (lambda (query targets) (match query
		((symbol tsql_pending_with) definitions body)
		(list (quote tsql_pending_with) definitions (tsql_select_assignment_query body targets))
		_ (begin
			(define fields (qb_fields query))
			(define converted (merge (map targets (lambda (target) (begin
				(define declaration (find parameters (lambda (item) (equal? (car item) target)) nil))
				(if declaration true (error "undeclared SELECT assignment target: @" target))
				(define contract (tsql_effective_parameter_contract (cadr declaration)))
				(define title (concat "__tsql_assignment:" target))
				(list title (tsql_implicit_cast_expr (fields title) (car contract) (cdr contract))))))))
			(make_query_block (qb_schema query) (qb_sources query) converted
				(qb_where query) (qb_group query) (qb_having query) (qb_order query)
				(qb_limit query) (qb_offset query) (qb_hidden query) (qb_stages query)
				(filter (qb_facts query) (lambda (fact)
					(not (equal? (car fact) (quote tsql_select_assignments))))))))))
	/* Ordered recursive assignment is a scalar fold over the existing engine
	result stream. Project only its bounded column inputs; no row bag or SQL
	interpreter is introduced, and source ordering/limits stay in the planner. */
	(define tsql_assignment_plan (lambda (query targets) (begin
		(define inputs (map (query_expr_session_reads query) cadr))
		(define target_keys (map targets (lambda (name) (concat "__tsql_parameter:" name))))
		(define recursive (reduce (query_expr_session_reads (qb_fields query))
			(lambda (found read) (or found (has? target_keys (cadr read)))) false))
		(define ordered (not (empty_list? (coalesceNil (qb_order query) '()))))
		(if (not (and recursive ordered))
			(list 'tsql_run_select_assignment 'session (list 'quote targets)
				(list 'quote inputs) 'tx (list 'lambda (list 'resultrow 'session)
					(build_queryplan_term query planning_session tx)) nil)
			(begin
				(if (and (<= (count (qb_sources query)) 1)
					(empty_list? (coalesceNil (qb_group query) '())) (nil? (qb_having query))
					(empty_list? (qb_stages query))
					(not (qassoc_get (qb_facts query) 'select_distinct false))
					(not (query_block_has_aggregates? query))) true
					(error "ordered recursive assignment requires an ungrouped single-source row stream"))
				(define formulas (map targets (lambda (target)
					(sql_type_formula (qb_sources query) ((qb_fields query) (concat "__tsql_assignment:" target))))))
				(define columns (merge_unique (map formulas sql_column_refs)))
				(define rewrite (lambda (expr) (match expr
					((symbol quote) _datum) expr
					((symbol get_column) _alias _alias_ic _name _name_ic) (begin
						(define index (find (produceN (count columns))
							(lambda (i) (expression_equal? expr (nth columns i))) nil))
						(if (nil? index) (error "recursive assignment column binding was lost")
							(list 'nth '__tsql_row (+ 1 (* 2 index)))))
					(cons head tail) (begin
						(define read (query_session_read_expr expr))
						(define index (if read (find (produceN (count target_keys))
							(lambda (i) (equal? (cadr read) (nth target_keys i))) nil) nil))
						(if (nil? index) (cons (rewrite head) (map tail rewrite))
							(list 'nth '__tsql_previous (+ 1 (* 2 index)))))
					_ expr)))
				(define values (map formulas rewrite))
				(if (reduce values (lambda (valid value)
					(and valid (tsql_scalar_expression_allowed? value))) true) true
					(error "ordered recursive assignment requires scalar row expressions"))
				(define reducer (sql_queryplan_compile_formula tx
					(list 'lambda (list '__tsql_previous '__tsql_row)
						(cons 'list (merge (mapIndex targets (lambda (i name)
							(list (concat "__tsql_assignment:" name) (nth values i)))))))))
				(define fields (if (empty_list? columns) (list "__tsql_fold_unit" 1)
					(merge (mapIndex columns (lambda (i column)
						(list (concat "__tsql_fold_input:" i) column))))))
				/* Derived inputs can flatten into UNION. Give every ordering
				value an output slot so that its original name is not lost when
				the reducer projects only its own scalar dependencies. */
				(define ordering (coalesceNil (qb_order query) '()))
				(define order_fields (merge (mapIndex ordering (lambda (i item)
					(list (concat "__tsql_fold_order:" i) (car item))))))
				(define order (mapIndex ordering (lambda (i item)
					(list (list 'get_column nil false (concat "__tsql_fold_order:" i) false) (cadr item)))))
				(define source (make_query_block (qb_schema query) (qb_sources query) (merge (list fields order_fields))
					(qb_where query) nil nil order (qb_limit query) (qb_offset query)
					(qb_hidden query) (qb_stages query) (qb_facts query)))
				(list 'tsql_run_select_assignment 'session (list 'quote targets)
					(list 'quote inputs) 'tx (list 'lambda (list 'resultrow 'session)
						(build_queryplan_term source planning_session tx)) reducer))))))
	/* SELECT, INSERT sources and diagnostics consume the same expanded relation.
	A planning factory must never survive into EXPLAIN as executable row code. */
	(define tsql_expand_select_query (lambda (query) (begin
		(define bound_query (tsql_finalize_sources query "query" '() '()))
		(define expanded (tsql_bind_column_compilers (sql_expand_views bound_query policy planning_session tx) (tsql_column_compiler planning_session)))
		(tsql_bind_column_compilers
			(tsql_expand_relational_scalar_calls expanded '() policy planning_session tx)
			(tsql_column_compiler planning_session)))))
	(define tsql_build_select_plan (lambda (query) (begin
		(define targets (tsql_select_assignment_targets query))
		(define query (if targets (tsql_select_assignment_query query targets) query))
		(define expanded (tsql_expand_select_query query))
		/* Binding validates names and contracts but lowers conversion callbacks.
		Describe the original declared expressions so NULL folding cannot erase
		their explicit widths or temporal precision. Analysis still consumes the
		bound tree; neither path constructs or executes a physical plan. */
		(define checked (if describe (tsql_describe_query expanded policy planning_session tx) expanded))
		(define descriptors (tsql_result_columns expanded parameters))
		(if analyze_parameters (list (quote quote) checked)
			describe (list (quote quote) (if targets '() descriptors))
			targets (tsql_assignment_plan expanded targets)
			(list (quote !begin)
				(list (quote resultfields) (list (quote quote) (queryplan_result_titles expanded))
					(list (quote quote) descriptors))
				(tsql_result_output_plan (build_queryplan_term expanded planning_session tx) descriptors))))))
	/* INSERT SELECT consumes the existing planner's result stream and the native
	insert operator. Identity allocation and constraints remain native. */
	(define tsql_insert_select_plan (lambda (database tbl columns explicit_columns query) (begin
		(define declarations (tsql_insert_declarations database tbl columns planning_session tx))
		(define columns (map declarations (lambda (column) (column "Field"))))
		(define expanded (tsql_expand_select_query query))
		(define identity (tsql_insert_identity database tbl planning_session tx))
		(define deferred (tsql_insert_deferred_columns database tbl columns policy planning_session tx))
		(define default_reads (query_expr_session_reads (map deferred cadr)))
		(define insert_expr (tsql_native_insert_expr (quote __tsql_insert_target) identity columns
			(cons list '((cons list (map (produceN (count columns))
				(lambda (i) (begin
					/* SQL source contracts come from the logical projection compiler. */
					(define info (sql_projection_info_at expanded i '()))
					(sql_type_formula '() (tsql_assignment_expr
						(list (quote sql_type_expr) '('nth 'item (+ (* i 2) 1))
							(sql_info_datatype info) (sql_info_collation info))
						(nth declarations i))))))))) 1 false deferred))
		(if (equal? (count (queryplan_result_titles expanded)) (count columns)) true
			(error "INSERT column count does not match SELECT column count"))
		(if describe (begin
			(define checked (tsql_describe_query expanded policy planning_session tx))
			/* Bind the source contract before accepting a pure prepared mutation. */
			(define checked_fields (tsql_result_columns checked parameters))
			(if (equal? (count checked_fields) (count declarations)) (list (quote quote) '())
				(error "invalid prepared INSERT source"))) (begin
				(define emit_count (symbol "__tsql_insert_emit_count"))
				(define inserted (symbol "__tsql_insert_emitted"))
				/* SELECT's completion is not the INSERT count (an empty constant
				source can return nil, UNION can return its last branch). Reuse the
				existing stream reducer to count actual native sink effects once. */
				(define native_plan (list (quote stream_window_reduce) 0 -1 (quote +)
					(list (quote lambda) '() 0)
					(list (quote lambda) (list emit_count)
						(list (list (quote lambda) (list (quote resultrow)) (build_queryplan_term expanded planning_session tx))
							(list (quote lambda) (list (quote item))
								(list (quote match) insert_expr inserted
									(list (quote begin) (list emit_count inserted) inserted)))))))

				/* AND propagates SQL NULL and evaluates later terms. Missing declaration
				metadata needs an explicit presence branch before reading its fields. */
				(define generated (if (nil? identity) false (not (has? columns (identity "Field")))))
				/* Completion is once per statement, outside the parallel row sink. */
				(define plan (list (quote match) native_plan (quote __tsql_stream_result)
					(list (quote begin)
						(if generated
							(list (quote if) (list (quote >) (quote __tsql_stream_result) 0)
								(list (quote tsql_unresolved_identity) (quote session) "streaming INSERT")
								(list (quote session) "__tsql_identity" nil))
							(if identity
								(list (quote tsql_unresolved_identity) (quote session) "explicit streaming IDENTITY insertion")
								(list (quote session) "__tsql_identity" nil)))
						(quote __tsql_stream_result))))
				/* The same outer producer and declared default bindings cover the
				entire stream, including zero rows and all ordered/UNION branches. */
				(tsql_identity_insert_statement database tbl identity explicit_columns columns
					(bind_invocation_scalar_reads default_reads plan "__tsql_default_binding_")))))
	))
	/* INSERT TOP constrains the existing source query's final window. It does
	not count, materialize, or reserve IDs for rows which the planner will skip. */
	(define tsql_insert_limit_query (lambda (query limit)
		(if (nil? limit) query (match query
			((symbol query-block) db sources fields where group having order old_limit offset hidden stages facts)
			(make_query_block db sources fields where group having order
				(if (nil? old_limit) limit (min old_limit limit)) offset hidden stages facts)
			((symbol union-block) mode branches order old_limit offset facts)
			(list (quote union-block) mode branches order
				(if (nil? old_limit) limit (min old_limit limit)) offset facts)
			_ (error "INSERT TOP requires a SELECT source")))))
	/* INSERT BULK declares the next connection-local wire stream. SQL types,
	access checks and the ordinary INSERT sink remain frontend responsibilities;
	no transport callback or transaction is retained in this declaration. */
	(define tsql_bulk_option (parser (or
		(parser (define name (regex "(?:CHECK_CONSTRAINTS|FIRE_TRIGGERS|KEEP_NULLS|TABLOCK)\\b" true)) (toUpper name))
		(parser '((define name (regex "(?:ROWS_PER_BATCH|KILOBYTES_PER_BATCH)\\b" true)) "=" (define amount tsql_int))
			(if (> amount 0) (toUpper name) (error "bulk batch hint must be positive"))))))
	(define tsql_bulk_column (parser '(
		(define name tsql_identifier) (define type tsql_declared_type) (define dimensions tsql_type_dimensions)
		(? (atom "COLLATE" true) (define collation tsql_identifier)))
		(list name type (tsql_decl_dimensions type dimensions) collation)))
	(define tsql_insert_bulk (parser '(
		(atom "INSERT" true) (atom "BULK" true)
		(? (define owner tsql_schema_identifier)) (define name tsql_identifier)
		"(" (define columns (+ tsql_bulk_column ",")) ")"
		(? (atom "WITH" true) "(" (define options (+ tsql_bulk_option ",")) ")")
	) (lambda () (begin
			(define database (tsql_temporary_database (tsql_qualifier_database owner schema frontend_options) name frontend_options tx))
			(define name (tsql_table_name database name planning_session tx))
			(if policy (policy database name true) true)
			(require_action database name "INSERT")
			(tsql_insert_declarations database name (map columns car) planning_session tx)
			(if describe (error "INSERT BULK requires an executing TDS connection")
				(list (quote tsql_prepare_bulk) database name (list (quote quote) columns)
					(list (quote quote) (coalesceNil options '())) (quote session) (quote tx)))))))
	/* Target resolution is shared with ordinary DML. A local target contributes
	only its declared relation; the physical statement binds its native input. */
	(define tsql_insert_target (parser (define target tsql_write_target) target))
	(define tsql_insert_target_source (lambda (target cte_bindings) (begin
		(match target '(target_schema name) (if (and (nil? target_schema) (string? name)
			(find cte_bindings (lambda (entry) (equal?? (car entry) name)) nil))
			(error "INSERT target must be a base table, not a CTE") true) _ true)
		(tsql_write_source target '()))))
	(define tsql_insert_into (parser '(
		(atom "INSERT" true) (define limit tsql_write_limit) (? (atom "INTO" true))
		(define target tsql_insert_target)
		(? "(" (define columns (+ tsql_identifier ",")) ")")
		(atom "VALUES" true) (define rows (+ tsql_insert_values_row ","))
	) (lambda cte_arguments (begin
			(define cte_bindings (if (empty_list? cte_arguments) '() (car cte_arguments)))
			(define source (tsql_insert_target_source target cte_bindings))
			(define database (source_schema source))
			(define tbl (source_relation source))
			(if policy (policy database tbl true) true)
			(require_action database tbl "INSERT")
			(define explicit_columns (not (nil? columns)))
			(define identity (tsql_insert_identity database tbl planning_session tx))
			(define declarations (tsql_insert_declarations database tbl
				(coalesce columns (tsql_insert_target_columns database tbl tx)) planning_session tx))
			(define columns (map declarations (lambda (column) (column "Field"))))
			(if (reduce rows (lambda (valid row) (and valid (equal? (count row) (count columns)))) true)
				true (error "INSERT column count does not match value count"))
			/* VALUES is a syntax-bounded batch. Limit that batch before native ID
			reservation; SELECT limits stay in the shared logical query block. */
			(define rows (map (if (nil? limit) rows (slice rows 0 limit)) (lambda (row)
				(map (produceN (count columns)) (lambda (i)
					(tsql_assignment_value
						(tsql_finalize_sources (nth row i) "query" cte_bindings '()) (nth declarations i) database policy planning_session tx))))))
			(if (reduce rows (lambda (found row) (or found (tsql_dataset_contains_inner_select row))) false)
				(tsql_insert_select_plan database tbl columns explicit_columns (tsql_values_to_select_query database columns rows))
				(begin
					(define typed_rows (map rows (lambda (row) (map (produceN (count columns))
						(lambda (i) (tsql_assignment_expr (nth row i) (nth declarations i)))))))
					(if describe (tsql_describe_dml database '()
						(merge (map typed_rows (lambda (row)
							(merge (map (produceN (count columns)) (lambda (i)
								(list (nth columns i) (nth row i)))))))) true)
						(tsql_identity_insert_statement database tbl identity explicit_columns columns
							(tsql_native_insert_expr (quote __tsql_insert_target) identity columns (sql_insert_values_expr
								(map typed_rows (lambda (row) (map row (lambda (value) (sql_type_formula '() value)))))) (count rows) true (tsql_insert_deferred_columns database tbl columns policy planning_session tx))))))
	))))
	/* Deferred cells are expanded before the native all-default row. Other
	defaults and generated values use the same operator as ordinary insertion. */
	(define tsql_insert_default (parser '(
		(atom "INSERT" true) (define limit tsql_write_limit) (? (atom "INTO" true))
		(define target tsql_insert_target)
		(atom "DEFAULT" true) (atom "VALUES" true)
	) (lambda cte_arguments (begin
			(define cte_bindings (if (empty_list? cte_arguments) '() (car cte_arguments)))
			(define source (tsql_insert_target_source target cte_bindings))
			(define database (source_schema source))
			(define tbl (source_relation source))
			(if policy (policy database tbl true) true)
			(require_action database tbl "INSERT")
			(define identity (tsql_insert_identity database tbl planning_session tx))
			(if describe (begin
				(tsql_insert_declarations database tbl '() planning_session tx)
				(list (quote quote) '()))
				(tsql_identity_insert_statement database tbl identity false '()
					(tsql_native_insert_expr (quote __tsql_insert_target) identity '()
						(if (equal?? limit 0) '('list) '('list '(list)))
						(if (equal?? limit 0) 0 1) true (tsql_insert_deferred_columns database tbl '() policy planning_session tx))))))))
	(define tsql_insert_select (parser '(
		(atom "INSERT" true) (define limit tsql_write_limit) (? (atom "INTO" true))
		(define target tsql_insert_target)
		(? "(" (define columns (+ tsql_identifier ",")) ")")
		(define query tsql_select)
	) (lambda cte_arguments (begin
			(define cte_bindings (if (empty_list? cte_arguments) '() (car cte_arguments)))
			(define source (tsql_insert_target_source target cte_bindings))
			(define database (source_schema source))
			(define tbl (source_relation source))
			(if policy (policy database tbl true) true)
			(require_action database tbl "INSERT")
			(define identity (tsql_insert_identity database tbl planning_session tx))
			(tsql_insert_select_plan database tbl (coalesce columns
				(tsql_insert_target_columns database tbl tx)) (not (nil? columns))
				(tsql_insert_limit_query (tsql_finalize_sources query "query" cte_bindings '()) limit))
	))))
	/* A WITH prefix supplies lexical source definitions to the existing
	mutation action. It does not create an execution scope or another DML path. */
	(define tsql_cte_mutation (parser '(
		(define definitions tsql_cte_definitions)
		(define statement (or tsql_insert_into tsql_insert_default tsql_insert_select tsql_update tsql_delete))
	) (lambda () (statement (tsql_finalize_sources definitions "definitions" '() '())))))
	/* Only UNIQUE declarations publish native keys. Ordinary acceleration
	declarations carry no catalog state: the common planner owns autoindexes. */
	(define tsql_create_index (parser '(
		(atom "CREATE" true) (define unique (? (atom "UNIQUE" true))) tsql_key_placement
		(atom "INDEX" true) (define name tsql_identifier) (atom "ON" true)
		(? (define schema2 tsql_schema_identifier)) (define tbl tsql_identifier)
		"(" (define columns tsql_key_columns) ")"
		(? (atom "INCLUDE" true) "(" (define included (+ tsql_identifier ",")) ")")
		(? (atom "WHERE" true) (define predicate tsql_expression))
		(? (define options tsql_index_options))
		(? tsql_index_location)
	) (lambda () (begin
			(define database (tsql_temporary_database (tsql_qualifier_database schema2 schema frontend_options) tbl frontend_options tx))
			(define tbl (tsql_table_name database tbl planning_session tx))
			(if policy (policy database tbl true) true)
			(require_action database tbl "ALTER")
			(if (and unique predicate) (error "filtered UNIQUE requires native filtered-key support") true)
			(tsql_check_index_options options unique)
			/* PRIMARY is a reserved native key identity, not an index alias. */
			(if (and unique (equal? (toUpper name) "PRIMARY"))
				(error "index name PRIMARY is reserved") true)
			(define declarations (get_schema database tbl tx))
			(define columns (map columns (lambda (name) (begin
				(define declaration (find declarations (lambda (column) (equal?? (column "Field") name)) nil))
				(if declaration (declaration "Field") (error "unknown index column: " name))))))
			(map included (lambda (name) (if (find declarations (lambda (column) (equal?? (column "Field") name)))
				true (error "unknown included column: " name))))
			(reduce columns (lambda (seen name)
				(if (has? seen name) (error "duplicate index column: " name) (cons name seen))) '())
			(if (not unique) true (list (quote if)
				(list (quote createkey) (list (quote table) database false tbl false (quote tx)) name true
					(cons (quote list) columns) (quote tx) (list (quote list) "nulls_not_distinct" true))
				true (list (quote error) "index already exists: " name)))))))
	(define tsql_alter_index (parser '(
		(atom "ALTER" true) (atom "INDEX" true) (or (atom "ALL" true) tsql_identifier) (atom "ON" true)
		(? (define schema2 tsql_schema_identifier)) (define tbl tsql_identifier)
		(define action (regex "(?:REBUILD|REORGANIZE|DISABLE)\\b" true))
		(? (atom "PARTITION" true) "=" (or tsql_int (atom "ALL" true)))
		(? (define options tsql_index_options))
	) (lambda () (begin
			(if policy (policy (tsql_qualifier_database schema2 schema frontend_options) tbl true) true)
			(require_action (tsql_qualifier_database schema2 schema frontend_options) tbl "ALTER")
			(get_schema (tsql_qualifier_database schema2 schema frontend_options) tbl tx)
			(if (equal? (toUpper action) "DISABLE") (error "DISABLE cannot bypass native constraints") true)
			(tsql_check_index_options options true)
			true))))
	(define tsql_drop_index (parser '(
		(atom "DROP" true) (atom "INDEX" true)
		(? (atom "IF" true) (atom "EXISTS" true))
		(define name tsql_identifier) (atom "ON" true)
		(? (define schema2 tsql_schema_identifier)) (define tbl tsql_identifier)
		(? tsql_index_options)
	) (lambda () (begin
			(if policy (policy (tsql_qualifier_database schema2 schema frontend_options) tbl true) true)
			(require_action (tsql_qualifier_database schema2 schema frontend_options) tbl "ALTER")
			(if (equal? (toUpper name) "PRIMARY") (error "PRIMARY is a constraint, not an acceleration index") true)
			/* A physical name carries no catalog state. Until native key removal
			protects referenced constraints, do not guess whether this name owns
			a real key or run a metadata check followed by an unsafe drop. */
			(error "DROP INDEX requires native constraint-safe key removal")))))
	(define tsql_foreign_key_mode (parser (or
		(parser '((atom "NO" true) (atom "ACTION" true)) "restrict")
		(parser (atom "CASCADE" true) "cascade")
		(parser '((atom "SET" true) (atom "NULL" true)) "set null"))))
	/* Scalar defaults retain SQL source when constant folding cannot prove a
	value. Contextual expressions are never sampled at DDL; the native clock
	and UUID markers preserve their existing generator contracts. */
	(define tsql_default_value (parser (or
		(parser tsql_clock_default '("default_expression" "CURRENT_TIMESTAMP"))
		(parser (define generator sql_uuid_default) (list "default_expression" generator))
		(parser (define captured (capture tsql_expression)) (if script_syntax
			/* Batch capture owns spelling only. Callee declarations and operand
			types are resolved by the later authorized statement parser. */
			(list "default_expression" (car captured) "default_deferred" true)
			(begin
				(define info (sql_expr_info '() (cadr captured)))
				(if (tsql_default_expression_allowed? (sql_info_formula info)) true
					(error "DEFAULT requires row- and parameter-free scalar SQL"))
				/* Folding is an optimization, not the scalar recipe's eligibility.
				Never sample a contextual expression; never evaluate residual SCM. */
				(define folded (if (tsql_default_dynamic? (sql_info_formula info)) (list false nil)
					(try (lambda () (list true (sql_constant_value (sql_info_formula info))))
						(lambda (_failure) (list false nil)))))
				(if (car folded)
					/* Destination conversion stays outside the folding catch. */
					(list "default" (list (quote sql_type_expr) (cadr folded)
						(sql_info_datatype info) (sql_info_collation info)) "__tsql_default_source" (car captured))
					(list "default_expression" (car captured) "default_deferred" true))))))))
	(define column_attributes (tsql_column_attributes tsql_default_value))
	(define column_definition (parser '(
		(define name tsql_identifier) (define type tsql_schema_type)
		(define dimensions tsql_type_dimensions) (define attributes column_attributes)
	) (list name type (cons (quote list) dimensions) attributes)))
	(define foreign_key_definition (parser '(
		(? (atom "CONSTRAINT" true) (define name tsql_identifier))
		(atom "FOREIGN" true) (atom "KEY" true) "(" (define columns (+ tsql_identifier ",")) ")"
		(atom "REFERENCES" true) (? (define target_schema tsql_schema_identifier)) (define target tsql_identifier)
		"(" (define target_columns (+ tsql_identifier ",")) ")"
		(define actions (? (or
			(parser '((atom "ON" true) (atom "DELETE" true) (define d tsql_foreign_key_mode)
				(? (atom "ON" true) (atom "UPDATE" true) (define u tsql_foreign_key_mode))) (list u d))
			(parser '((atom "ON" true) (atom "UPDATE" true) (define u tsql_foreign_key_mode)
				(? (atom "ON" true) (atom "DELETE" true) (define d tsql_foreign_key_mode))) (list u d)))))
		/* This modifier affects SQL Server replication-agent writes only. The
		frontend offers ordinary client execution, where native FK enforcement
		and historical-row validation remain unchanged. It never changes MOSI
		or native replication admission into a SQL constraint bypass. */
		(? (atom "NOT" true) (atom "FOR" true) (atom "REPLICATION" true))
	) (list (quote list) "foreign" name (cons (quote list) columns) target
			(cons (quote list) target_columns) (if actions (car actions) nil)
			(if actions (cadr actions) nil) target_schema)))
	(define tsql_create_table (parser '(
		(atom "CREATE" true) (atom "TABLE" true)
		(define ifnotexists (? (atom "IF" true) (atom "NOT" true) (atom "EXISTS" true)))
		(? (define schema2 tsql_schema_identifier)) (define name tsql_identifier)
		"(" (define columns (+ (or
			/* Placement and direction are physical index preferences. Logical
			uniqueness uses native keys; MemCP selects its own columnar access. */
			(parser '((? (atom "CONSTRAINT" true) (define key tsql_identifier)) (atom "PRIMARY" true) (atom "KEY" true) tsql_key_placement "(" (define keys tsql_key_columns) ")" tsql_key_layout)
				(list (quote list) "unique" "PRIMARY" (cons (quote list) keys)
					(cons (quote list) (if key (list "constraint_name" key) '()))))
			(parser '((? (atom "CONSTRAINT" true) (define key tsql_identifier)) (atom "UNIQUE" true) tsql_key_placement
				"(" (define keys tsql_key_columns) ")" tsql_key_layout)
				(list (quote list) "unique" (coalesce key (reduce keys (lambda (a b) (concat a "_" b)) name)) (cons (quote list) keys)
					(cons (quote list) (merge '("nulls_not_distinct" true)
						(if key (list "constraint_name" key) '())))))
			foreign_key_definition
			(parser '((define name tsql_identifier) (atom "AS" true)
				(define expression tsql_expression) (define persisted (? (atom "PERSISTED" true))))
				(list 'list "computed" name expression (if persisted true false)))
			(parser (define column column_definition) (match column '(name type dimensions attributes)
				(list (quote list) "column" name type dimensions (cons (quote list) attributes))))
		) ",")) ")"
		(? tsql_index_location)
	) (lambda () (begin
			(if policy (policy (tsql_qualifier_database schema2 schema frontend_options) name true) true)
			(require_action (tsql_qualifier_database schema2 schema frontend_options) name "CREATE TABLE")
			/* Native FK endpoints share the CREATE target database. Preserve the
			parsed qualifier until here, then pass only the existing native FK
			declaration to sql_create_table; never silently discard another owner. */
			(define target_database (tsql_qualifier_database schema2 schema frontend_options))
			(define columns (map columns (lambda (definition) (match definition
				((symbol list) "foreign" key key_columns target target_columns update_mode delete_mode target_schema)
				(if (equal? (tsql_qualifier_database target_schema target_database frontend_options) target_database)
					(list (quote list) "foreign" key key_columns target target_columns update_mode delete_mode)
					(error "cross-database foreign keys are not supported"))
				_ definition))))
			(define columns (map (tsql_table_primary_columns columns) (lambda (column)
				(tsql_prepare_column target_database column planning_session))))
			(define columns (tsql_prepare_computed_columns target_database columns planning_session))
			(if (and (tsql_temporary_name? name) schema2)
				(error "qualified temporary creation is not supported") true)
			(if (sql_private_relation? name) (error "private SQL relation names cannot be created directly") true)
			(define create (list (if (tsql_temporary_name? name) (quote tsql_create_temporary) (quote tsql_create_relation))
				target_database name (cons (quote list) (map columns tsql_identity_native_column))
				(tsql_identity_table_options columns) (if ifnotexists true false) (quote tx)))
			(tsql_declaration_formula (if (tsql_temporary_name? name) (append create (quote session)) create))))))
	(define tsql_alter_table (parser '(
		(atom "ALTER" true) (atom "TABLE" true)
		(? (define schema2 tsql_schema_identifier)) (define name tsql_identifier)
		(define changes (+ (or
			(parser '((define mode (or
				(parser '((atom "WITH" true) (atom "CHECK" true) (atom "CHECK" true)) (list true true))
				(parser '((atom "WITH" true) (atom "NOCHECK" true) (atom "CHECK" true)) (list true false))
				(parser (atom "NOCHECK" true) (list false false))
				(parser (atom "CHECK" true) (list true false))))
				(atom "CONSTRAINT" true)
				(define targets (or (parser (atom "ALL" true) true) (+ tsql_identifier ","))))
				(lambda (database name) (begin
					(define handle (table database false name false tx))
					(if handle true (error "constraint table does not exist: " name))
					/* Resolve every named SQL target before any native state change.
					ALL is resolved by the native child-owned declaration directory;
					committed/unknown targets are never silently skipped. */
					(define targets (if (equal? targets true) true
						(map targets (lambda (target) (begin
							(tsql_constraint_name (show handle "constraints" tx) target))))))
					(list (quote constraint_state) (list (quote table) database false name false (quote tx))
						(list (quote quote) targets) (car mode) (nth mode 1) (quote tx) (quote session)))))
			(parser '((? (atom "WITH" true) (atom "CHECK" true))
				(atom "ADD" true) (atom "CONSTRAINT" true) (define key tsql_identifier)
				(atom "CHECK" true) "(" (define expression (capture tsql_expression)) ")")
				(lambda (database name)
					(list (quote tsql_define_check) database name key (car expression) (quote tx) (quote session))))
			(parser '((atom "ALTER" true) (atom "COLUMN" true) (define column tsql_identifier)
				(define type tsql_declared_type) (define dimensions tsql_type_dimensions)
				(? (atom "COLLATE" true) (define collation tsql_identifier))
				(define nullable (or (parser '((atom "NOT" true) (atom "NULL" true)) false)
					(parser (atom "NULL" true) true) (parser empty true)))
			) (lambda (database name) (begin
					(define handle (table database false name false tx))
					(define previous (if (nil? handle) nil (show handle "column" column true tx)))
					(if (nil? previous) (error "ALTER column does not exist: " column) true)
					(define dimensions (tsql_decl_dimensions type dimensions))
					(define builtin (tsql_builtin_type_declaration type))
					(define dimensions (if builtin (builtin "dimensions") dimensions))
					(define type (tsql_storage_type type))
					(define collation (coalesceNil collation (coalesceNil (previous "Collation") "")))
					(if (tsql_compatible_column_declaration? previous type dimensions nullable collation) true
						(error "ALTER COLUMN requires native validation or conversion"))
					(list (quote !begin) (list (quote altercolumn) (list (quote table) database false name false (quote tx))
						(previous "Field") "declaration"
						(list (quote quote) (list (list type dimensions nullable collation)
							(tsql_column_declaration_tuple previous))) (quote tx)) true))))

			(parser '((atom "ADD" true) (? (atom "CONSTRAINT" true) (define key tsql_identifier))
				(atom "PRIMARY" true) (atom "KEY" true) tsql_key_placement "(" (define columns tsql_key_columns) ")" tsql_key_layout)
				(lambda (database name) (begin
					/* Nullable existing columns need native ALTER nullability support.
					Reject this shape rather than adding a parallel validation scan. */
					(if (reduce columns (lambda (valid column) (and valid
						(not ((find (show (table database false name false tx)) (lambda (meta) (equal? (meta "Field") column))) "Null")))) true)
						true (error "ADD PRIMARY KEY currently requires NOT NULL columns"))
					(list (quote createkey) (list (quote table) database false name false (quote tx)) "PRIMARY" true
						(cons (quote list) columns) (quote tx)
						(cons (quote list) (if key (list "constraint_name" key) '()))))))
			(parser '((atom "ADD" true) (atom "CONSTRAINT" true) (define key tsql_identifier)
				(atom "UNIQUE" true) tsql_key_placement "(" (define columns tsql_key_columns) ")" tsql_key_layout)
				(lambda (database name)
					(list (quote createkey) (list (quote table) database false name false (quote tx)) key true
						(cons (quote list) columns) (quote tx)
						(list (quote list) "nulls_not_distinct" true "constraint_name" key))))
			(parser '((atom "ADD" true) (define definition foreign_key_definition))
				(lambda (database name) (match definition
					((symbol list) "foreign" key key_columns target target_columns update_mode delete_mode target_schema)
					(begin
						(define parent_database (tsql_qualifier_database target_schema database frontend_options))
						(if (equal? parent_database database) true (error "cross-database foreign keys are not supported"))
						/* Check reference authority before resolving names or exposing
						parent declarations. Child ALTER was admitted by the outer action. */
						(require_action parent_database target "REFERENCES")
						(define target (tsql_table_name parent_database target planning_session tx))
						(define child_handle (table database false name false tx))
						(define parent_handle (table parent_database false target false tx))
						(if (and child_handle parent_handle) true (error "foreign key endpoint does not exist"))
						(define child (tsql_foreign_columns child_handle (cdr key_columns) tx))
						(define parent (tsql_foreign_columns parent_handle (cdr target_columns) tx))
						(define update_mode (coalesceNil update_mode "restrict"))
						(define delete_mode (coalesceNil delete_mode "restrict"))
						(tsql_validate_foreign_columns child parent update_mode delete_mode planning_session)
						(define key (coalesceNil key (concat name "_" ((car child) "Field") "_fk")))
						(list (quote if) (list (quote createforeignkey)
							(list (quote table) database false name false (quote tx)) key
							(list (quote quote) (map child (lambda (column) (column "Field"))))
							(list (quote table) parent_database false target false (quote tx))
							(list (quote quote) (map parent (lambda (column) (column "Field"))))
							update_mode delete_mode (quote tx)
							(list (quote quote) (list "expected_child" (map child tsql_foreign_column_facts)
								"expected_parent" (map parent tsql_foreign_column_facts))))
							true (list (quote error) "foreign key already exists: " key))))))
			(parser '((atom "ADD" true) (? (atom "CONSTRAINT" true) (define key tsql_identifier))
				(atom "DEFAULT" true) (define default tsql_default_value)
				(atom "FOR" true) (define column tsql_identifier))
				(lambda (database name) (begin
					(define declaration (tsql_column_description database name column true nil tx))
					(if (nil? declaration) (error "DEFAULT column does not exist: " column) true)
					(define default (tsql_prepare_default default
						(coalesceNil (declaration "BaseType") (declaration "RawType"))
						(lambda (value) (eval (sql_type_formula '() (tsql_assignment_expr value declaration))))))
					(define value (default "default"))
					(list (quote !begin)
						(list (quote altercolumn) (list (quote table) database false name false (quote tx))
							(declaration "Field") "add_default"
							(list (quote quote) (list value (coalesceNil (default "default_expression") "")
								(coalesceNil key (concat "DF_" name "_" (declaration "Field")))
								(if (default "default_deferred") true false))) (quote tx)) true))))
			(parser '((atom "ADD" true) (define columns (+ column_definition ",")))
				(lambda (database name)
					(begin
						/* Native ADD has no identity backfill for historical rows.
						Reject this form instead of manufacturing a parallel allocator. */
						(if (find columns (lambda (column) ((nth column 3) "auto_increment")))
							(error "ALTER ADD IDENTITY requires native historical-row initialization") true)
						(list (quote tsql_add_columns) database name
							(cons (quote list) (map columns (lambda (column)
								(match column '(col type dimensions attributes)
									(tsql_prepare_column database (list (quote list) "column" col type dimensions
										(cons (quote list) attributes)) planning_session))))) (quote tx)))))
			(parser '((atom "DROP" true) (atom "COLUMN" true)
				(define ifexists (? (atom "IF" true) (atom "EXISTS" true)))
				(define columns (+ tsql_identifier ",")))
				(lambda (database name)
					(cons (quote !begin) (map columns (lambda (col)
						(list (quote altertable) (list (quote table) database false name false (quote tx))
							(if ifexists "drop_if_exists" "drop")
							(coalesceNil (resolve_column_name database name col true) col)))))))
			(parser '((atom "DROP" true) (atom "CONSTRAINT" true)
				(define ifexists (? (atom "IF" true) (atom "EXISTS" true)))
				(define keys (+ tsql_identifier ",")))
				(lambda (database name) (cons (quote !begin) (map keys (lambda (key)
					(list (quote if) (list (quote or)
						(list (quote altercolumn) (list (quote table) database false name false (quote tx)) key "drop_default" nil (quote tx))
						(list (quote dropforeignkey) (list (quote table) database false name false (quote tx)) key)
						(list (quote dropkey) (list (quote table) database false name false (quote tx)) key (quote tx)))
						true (if ifexists true (list (quote error) "constraint does not exist: " key))))))))
		) ","))
	) (lambda () (begin
			(define database (tsql_temporary_database (tsql_qualifier_database schema2 schema frontend_options) name frontend_options tx))
			(if policy (policy database name true) true)
			(require_action database name "ALTER")
			(define name (tsql_table_name database name planning_session tx))
			(cons (quote !begin) (map changes (lambda (change) (change database name))))))))
	(define tsql_database_file_field (parser (or
		(parser '((atom "NAME" true) "=" (define value (or tsql_string tsql_identifier))) (list "name" value))
		(parser '((atom "FILENAME" true) "=" (define value tsql_string)) (list "filename" value))
		(parser '((atom "SIZE" true) "=" (define value tsql_int) (? (atom "MB" true))) (list "size" value)))))
	(define tsql_database_file_definition (parser '("(" (define fields (+ tsql_database_file_field ",")) ")") fields))
	(define tsql_database_files (parser (or
		(parser '((atom "ON" true) (? (atom "PRIMARY" true)) (define data tsql_database_file_definition)
			(? (atom "LOG" true) (atom "ON" true) (define log tsql_database_file_definition)))
			(if log (list (list "data" data) (list "log" log)) (list (list "data" data))))
		(parser empty '()))))
	(define tsql_transaction_keyword (if script_syntax
		(parser (or (atom "TRAN" true) (atom "TRANSACTION" true)))
		(parser (? (or (atom "TRAN" true) (atom "TRANSACTION" true))))))
	(define tsql_rollback (parser '((atom "ROLLBACK" true)
		(? (or (atom "TRAN" true) (atom "TRANSACTION" true))))
		(lambda () (list (quote tsql_rollback_transaction) (quote session)))))
	(define tsql_commit (parser '((atom "COMMIT" true) (? (or (atom "TRAN" true) (atom "TRANSACTION" true))))
		(lambda () (list (quote tsql_commit_transaction) (quote session)))))
	(define p (parser (or
		tsql_cte_mutation
		(parser (define query tsql_root_select) (lambda () (tsql_build_select_plan query)))
		(parser '((atom "EXPLAIN" true) (atom "IR" true) (define query tsql_root_select)) (lambda () (explain_queryplan_ir (tsql_expand_select_query query))))
		(parser '((atom "EXPLAIN" true) (atom "REORDER" true) (define query tsql_root_select)) (lambda () (explain_queryplan_reorder (tsql_expand_select_query query) planning_session)))
		(parser '((atom "EXPLAIN" true) (atom "COMPILE" true) (define query tsql_root_select)) (lambda () (explain_queryplan_compile (tsql_expand_select_query query) parse_started_ns (strlen s) planning_session)))
		(parser '((atom "EXPLAIN" true) (atom "PHYSICAL" true) (define query tsql_root_select)) (lambda () (explain_queryplan_physical (tsql_expand_select_query query) planning_session)))
		(parser '((atom "EXPLAIN" true) (define query tsql_root_select))
			(lambda () (list 'resultrow (list 'list "code" (pretty_print (optimize
				(build_queryplan_term (tsql_expand_select_query query)
					planning_session tx)) (settings "ExplainWidth"))))))
		tsql_insert_bulk tsql_insert_into tsql_insert_default tsql_insert_select tsql_update tsql_delete tsql_truncate
		tsql_create_view tsql_create_table tsql_alter_table tsql_create_index tsql_alter_index tsql_drop_index
		(parser '((define mode (or (parser (atom "GRANT" true) "GRANT")
			(parser (atom "REVOKE" true) "REVOKE") (parser (atom "DENY" true) "DENY")))
			(? (atom "GRANT" true) (atom "OPTION" true) (atom "FOR" true))
			(define actions (+ (parser '((define words (+ (not (regex "[A-Za-z_][A-Za-z0-9_]*")
				(atom "ON" true) (atom "TO" true) (atom "FROM" true))))
				(? "(" (+ tsql_identifier ",") ")")) words) ","))
			(? (atom "ON" true) (? (+ (regex "[A-Za-z_][A-Za-z0-9_]*")) "::")
				(+ tsql_identifier ".") (? "(" (+ tsql_identifier ",") ")"))
			(define direction (or (parser (atom "TO" true) "TO") (parser (atom "FROM" true) "FROM")))
			(define principals (+ tsql_identifier ","))
			(? (atom "WITH" true) (atom "GRANT" true) (atom "OPTION" true))
			(? (atom "CASCADE" true)) (? (atom "AS" true) tsql_identifier))
			(lambda () (begin
				(if (equal? direction (if (equal? mode "REVOKE") "FROM" "TO")) true
					(error "invalid access request direction"))
				(list (quote tsql_change_permissions) mode
					(list (quote quote) actions) (list (quote quote) principals) (quote session) (quote tx)))))
		(parser '((atom "CREATE" true) (atom "TYPE" true)
			(? (define owner tsql_identifier) ".") (define name tsql_identifier)
			(atom "FROM" true) (define base tsql_declared_type) (define dimensions tsql_type_dimensions)
			(define nullable (or (parser '((atom "NOT" true) (atom "NULL" true)) false)
				(parser (atom "NULL" true) true) (parser empty true))))
			(lambda () (begin
				(if (or (nil? owner) (equal?? owner "dbo")) true (error "unsupported type owner: " owner))
				(if policy (policy schema true true) true)
				(require_action schema name "CREATE TYPE")
				(list (quote create_sql_type_alias) schema name base (list (quote quote) dimensions) nullable (quote tx)))))
		(parser '((atom "DROP" true) (atom "TYPE" true)
			(define ifexists (? (atom "IF" true) (atom "EXISTS" true)))
			(? (define owner tsql_identifier) ".") (define name tsql_identifier))
			(lambda () (begin
				(if (or (nil? owner) (equal?? owner "dbo")) true (error "unsupported type owner: " owner))
				(if policy (policy schema true true) true)
				(require_action schema name "ALTER")
				(list (quote drop_sql_type_alias) schema name (if ifexists true false) (quote tx)))))
		(parser '((atom "CREATE" true) (atom "DATABASE" true)
			(define ifnotexists (? (atom "IF" true) (atom "NOT" true) (atom "EXISTS" true)))
			(define name tsql_identifier) (define files tsql_database_files)) (lambda () (begin
				/* The common CREATE owner rechecks the actor under its DDL gate;
				an absent destination is not an ordinary database-access request. */
				(sql_database_create_authorization access_policy name (if ifnotexists true false))
				(list (quote tsql_create_database) (quote tx) name (if ifnotexists true false)
					(list (quote quote) files) (quote session)))))
		(parser '((atom "ALTER" true) (atom "DATABASE" true) (define name tsql_identifier) (atom "SET" true)
			(define option (or (parser (atom "RECOVERY" true) "recovery")
				(parser (atom "AUTO_CLOSE" true) "auto_close") (parser (atom "AUTO_SHRINK" true) "auto_shrink")
				(parser (atom "TRUSTWORTHY" true) "trustworthy")))
			(define value (or (parser (atom "SIMPLE" true) "simple") (parser (atom "FULL" true) "full")
				(parser (atom "BULK_LOGGED" true) "bulk_logged") (parser (atom "ON" true) "on") (parser (atom "OFF" true) "off"))))
			(lambda () (begin
				(require_action name true "ALTER DATABASE")
				(list (quote tsql_alter_database) (quote tx) name option value (quote session)))))
		(parser '((atom "DROP" true) (atom "DATABASE" true)
			(define ifexists (? (atom "IF" true) (atom "EXISTS" true))) (define name tsql_identifier))
			(lambda () (begin
				/* An absent namespace has no database owner or CONTROL grant.
				Require native administration, then let the native DROP validate
				the identity and apply its ordinary missing/IF EXISTS contract. */
				(if (nil? (database_name name true))
					(if policy (policy "system" true true) true)
					(require_action name true "CONTROL"))
				(list (quote tsql_drop_database) (list (quote session) "__memcp_tx")
					name (if ifexists true false) (quote session)))))
		(parser '((atom "DROP" true) (atom "TABLE" true)
			(define ifexists (? (atom "IF" true) (atom "EXISTS" true)))
			(? (define schema2 tsql_schema_identifier)) (define name tsql_identifier))
			(lambda () (begin
				(define database (tsql_qualifier_database schema2 schema frontend_options))
				(tsql_declaration_formula (if (tsql_temporary_name? name) (begin
					(tsql_temporary_check_name name)
					(if policy (policy database name true) true)
					(require_action database name "ALTER")
					(list (quote tsql_drop_temporary) name (if ifexists true false) (quote session) (quote tx)))
					(begin
						(if policy (policy database name true) true)
						(require_action database name "ALTER")
						(list (quote tsql_drop_table) database name (if ifexists true false)
							(quote session) (quote tx))))))))
		(parser '((atom "DROP" true) (atom "VIEW" true)
			(define ifexists (? (atom "IF" true) (atom "EXISTS" true)))
			(? (define schema2 tsql_schema_identifier)) (define name tsql_identifier))
			(lambda () (begin
				(define database (tsql_qualifier_database schema2 schema frontend_options))
				(define name (coalesceNil (tsql_view_name database name) name))
				(if policy (policy database name true) true)
				(require_action database name "ALTER")
				(tsql_declaration_formula (list (quote drop_sql_view) (list (quote session) "__memcp_tx")
					database name (if ifexists true false))))))
		(parser '((atom "SET" true) (atom "IDENTITY_INSERT" true)
			(? (define schema2 tsql_schema_identifier)) (define name tsql_identifier)
			(define enabled (or (parser (atom "ON" true) true) (parser (atom "OFF" true) false))))
			(lambda () (begin
				(define database (tsql_temporary_database (tsql_qualifier_database schema2 schema frontend_options) name frontend_options tx))
				(define name (tsql_table_name database name planning_session tx))
				(if policy (policy database name true) true)
				(require_action database name "ALTER")
				(list (quote tsql_set_identity_insert) (quote session) database name enabled (quote tx)))))
		(parser (define setting tsql_format_only_setting)
			(lambda () (list 'session "__tsql_fmtonly" (cadr setting))))
		/* Isolation remains executable module code. Native ACID staging is selected
		by the T-SQL request owner; the scoped setter never replaces that owner.
		https://learn.microsoft.com/sql/t-sql/statements/set-transaction-isolation-level-transact-sql */
		(parser '((atom "SET" true) (atom "TRANSACTION" true) (atom "ISOLATION" true) (atom "LEVEL" true)
			(define level (or
				(parser '((atom "READ" true) (atom "UNCOMMITTED" true)) "READ UNCOMMITTED")
				(parser '((atom "READ" true) (atom "COMMITTED" true)) "READ COMMITTED")
				(parser '((atom "REPEATABLE" true) (atom "READ" true)) "REPEATABLE READ")
				(parser (atom "SNAPSHOT" true) "SNAPSHOT")
				(parser (atom "SERIALIZABLE" true) "SERIALIZABLE"))))
			(lambda () (list (quote tsql_set_transaction_isolation) (quote session) level)))
		(parser '((atom "SET" true) (atom "NOCOUNT" true)
			(define enabled (or (parser (atom "ON" true) true) (parser (atom "OFF" true) false))))
			(lambda () (list (quote session) "__tsql_nocount" enabled)))
		(parser '((atom "SET" true) (atom "DATEFORMAT" true)
			(define order (or (parser (define name tsql_variable_name) (parameter_expression name name))
				tsql_string tsql_identifier)))
			(lambda () (list (quote session) "__tsql_dateformat" (list tsql_dateformat_value (sql_type_formula '() order)))))
		(parser '((atom "SET" true) (atom "LANGUAGE" true)
			(define language (or (parser (define name tsql_variable_name) (parameter_expression name name))
				tsql_string tsql_identifier)))
			(lambda () (list tsql_set_language (quote session) (sql_type_formula '() language))))
		(parser '((atom "SET" true) (atom "DATEFIRST" true)
			(define first (or tsql_int
				(parser (define name tsql_variable_name) (parameter_expression name name)))))
			(lambda () (list (quote session) "__tsql_datefirst" (list tsql_datefirst_value (sql_type_formula '() first)))))
		/* QUOTED_IDENTIFIER has already acted at the lexical batch boundary.
		SQL Server 2017+ keeps ANSI_PADDING ON even after SET ... OFF;
		there is no alternate storage mode or session state to capture.
		https://learn.microsoft.com/sql/t-sql/statements/set-ansi-padding-transact-sql */
		(parser '((atom "SET" true) (define options (+ (or (atom "ANSI_NULLS" true) (atom "QUOTED_IDENTIFIER" true) (atom "ANSI_PADDING" true)) ","))
			(define mode (or (atom "ON" true) (atom "OFF" true))))
			(begin
				(lambda () (if (find options (lambda (option) (equal?? option "ANSI_NULLS")) nil)
					(list tsql_set_ansi_nulls 'session (equal?? mode "ON")) (list 'quote nil)))))
		(parser '((atom "USE" true) (define name tsql_identifier)) (lambda () (begin
			(if import_target (begin
				(if policy (policy import_target true false) true)
				(list (quote tsql_import_database) (quote session) name))
				(list (quote tsql_use_database) (quote session) name)))))
		(parser '((atom "BEGIN" true) tsql_transaction_keyword)
			(lambda () (list (quote tsql_begin_transaction) (quote session) (quote tx))))
		tsql_commit
		tsql_rollback
	)))


	(define command_parser (if analyze_parameters
		(parser (define query tsql_root_select) (lambda () (tsql_build_select_plan query)))
		describe (parser (or (parser (define query tsql_root_select) (lambda () (tsql_build_select_plan query)))
			tsql_cte_mutation tsql_insert_into tsql_insert_default tsql_insert_select tsql_update tsql_delete))
		(parser (or p (parser empty (lambda () ""))))))
	(if (frontend_options "__tsql_logical_update")
		((parser '((* ";") (define update tsql_update) (* ";")) (update)
			"^(?:/\\*(?s:.*?)\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+") s)
		(frontend_options "__tsql_logical_query")
		((parser '((* ";") (define query tsql_root_select) (* ";"))
			(begin
				/* Consume only the root assignment through the ordinary target
				conversion owner. Nested assignments retain their existing guard. */
				(define targets (tsql_select_assignment_targets query))
				(define prepared (if targets (tsql_select_assignment_query query targets) query))
				(define resolved (tsql_finalize_sources prepared "query" '() '()))
				(if targets (make_query_block (qb_schema resolved) (qb_sources resolved) (qb_fields resolved)
					(qb_where resolved) (qb_group resolved) (qb_having resolved) (qb_order resolved)
					(qb_limit resolved) (qb_offset resolved) (qb_hidden resolved) (qb_stages resolved)
					(qassoc_set (qb_facts resolved) 'tsql_select_assignments targets)) resolved))
			"^(?:/\\*(?s:.*?)\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+") s)
		expression_only
		((parser (define expression tsql_expression) (begin
			(if (and function_body (not (tsql_scalar_expression_allowed? expression)))
				(error "scalar function expression requires a relational plan") true)
			(define source (frontend_options "__tsql_expression_source"))
			(define finalized (tsql_finalize_sources expression "query" '() '()))
			(if (frontend_options "__tsql_expression_syntax") finalized
				(nil? source) (sql_expr_info '() finalized)
				(begin
					(define bound (bind_query_names (make_query_block schema (list source)
						(list "value" finalized) true nil nil nil nil nil nil nil nil) '()))
					(sql_expr_info (qb_sources bound) (cadr (qb_fields bound)))))) "^(?:/\\*(?s:.*?)\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+") s)
		script_syntax (begin
			/* Control grammar captures the ordinary leaf grammar without compiling its
			statement yet. Earlier DDL in a batch must publish before later DML binds.
			Control nodes are syntax data, not another query or transaction engine. */
			(define tsql_script_variable tsql_variable_name)
			(define tsql_script_declaration (parser '(
				(define name tsql_script_variable) (? (atom "AS" true))
				(define type tsql_declared_type) (define dimensions tsql_type_dimensions)
				(? "=" (define initializer (capture tsql_expression)))
			) (list name (tsql_parameter_declaration schema type dimensions)
					(if initializer (car initializer) nil))))
			(define tsql_script_exec_value (parser (or
				(parser '((define variable tsql_script_variable) (or (atom "OUTPUT" true) (atom "OUT" true)))
					(list "output" variable))
				(parser (atom "DEFAULT" true) (list "default"))
				(parser (define value (capture tsql_expression)) (car value)))))
			(define tsql_script_exec_argument (parser (or
				(parser '((define name tsql_script_variable) "=" (define value tsql_script_exec_value))
					(list name value))
				(parser (define value tsql_script_exec_value) (list "" value)))))
			/* The optional EXEC spelling is restricted to the first batch item.
			Both spellings produce the same invocation data and use one executor. */
			(define tsql_script_invocation (parser (or
				(parser '((define status tsql_script_variable) "="
					(? (define owner tsql_schema_identifier)) (define name tsql_identifier)
					(define arguments (* tsql_script_exec_argument ",")))
					(list "execute-status" (tsql_qualifier_database owner schema frontend_options) name arguments status))
				(parser '((? (define owner tsql_schema_identifier)) (define name tsql_identifier)
					(define arguments (* tsql_script_exec_argument ",")))
					(list "execute" (tsql_qualifier_database owner schema frontend_options) name arguments)))))
			(define tsql_script_command (parser (or
				(parser '((define projection (capture tsql_select_head))
					(atom "INTO" true) (? (define owner tsql_schema_identifier))
					(define name tsql_identifier)
					(define remainder (capture (parser '(tsql_select_tail tsql_select_set_terms
						tsql_select_order tsql_select_suffix (? tsql_query_hints)) true))))
					(begin
						(if (and (nil? owner) (tsql_temporary_name? name)) true
							(error "SELECT INTO currently requires an unqualified local temporary target"))
						(tsql_temporary_check_name name)
						(if (find (nth (cadr projection) 2) (lambda (column) (not (equal? (count column) 2))) nil)
							(error "SELECT INTO requires named result columns without variable assignments") true)
						(list "select-into" name (concat (car projection) " " (car remainder)))))
				(parser (define statement (capture (or tsql_rollback tsql_commit))) (list "recovery" (strtrim (car statement))))
				(parser '((atom "BEGIN" true) (atom "TRY" true) (define body (* tsql_script_item empty))
					(atom "END" true) (atom "TRY" true) (atom "BEGIN" true) (atom "CATCH" true)
					(define recovery (* tsql_script_item empty)) (atom "END" true) (atom "CATCH" true))
					(list "try" body recovery))
				(parser '((atom "WHILE" true) (define condition (capture tsql_expression))
					(define body tsql_script_command)) (list "while" (car condition) body))
				(parser (atom "BREAK" true) (list "break"))
				(parser (atom "CONTINUE" true) (list "continue"))
				(parser '((atom "GOTO" true) (define name tsql_identifier)) (list "goto" (toLower name)))
				(parser '((define name tsql_identifier) ":") (list "label" (toLower name)))
				(parser '((atom "RETURN" true) (? (define value (capture tsql_expression))))
					(list "return" (if value (car value) nil)))
				(parser '((atom "PRINT" true) (define value (capture tsql_expression)))
					(list "print" (car value)))
				(parser '((atom "RAISERROR" true) "(" (define arguments (+ (capture tsql_expression) ",")) ")")
					(list "raiserror" (map arguments car)))
				(parser '((atom "THROW" true) (? (define arguments
					'((capture tsql_expression) "," (capture tsql_expression) "," (capture tsql_expression)))))
					(list "throw" (if arguments (list (car (nth arguments 0)) (car (nth arguments 2)) (car (nth arguments 4))) nil)))
				(parser '((atom "IF" true) (define condition (capture tsql_expression))
					(define yes tsql_script_command) (* ";")
					(? (atom "ELSE" true) (define no tsql_script_command)))
					(list "if" (car condition) yes no))
				(parser '((atom "BEGIN" true) (define commands (* tsql_script_item empty)) (atom "END" true))
					(list "block" commands))
				(parser '((atom "DECLARE" true) (define name tsql_identifier) (atom "CURSOR" true)
					(define options (* (or (atom "LOCAL" true) (atom "GLOBAL" true)
						(atom "FORWARD_ONLY" true) (atom "SCROLL" true) (atom "STATIC" true)
						(atom "KEYSET" true) (atom "DYNAMIC" true) (atom "FAST_FORWARD" true)
						(atom "READ_ONLY" true) (atom "SCROLL_LOCKS" true) (atom "OPTIMISTIC" true)
						(atom "TYPE_WARNING" true)) empty))
					(atom "FOR" true) (define query (capture tsql_root_select)))
					(list "cursor-declare" (toLower name) (tsql_cursor_options (map options toUpper)) (car query)))
				(parser '((atom "OPEN" true) (define global (? (atom "GLOBAL" true))) (define name tsql_identifier))
					(list "cursor-open" (toLower name) (if global true false)))
				(parser '((atom "CLOSE" true) (define global (? (atom "GLOBAL" true))) (define name tsql_identifier))
					(list "cursor-close" (toLower name) (if global true false)))
				(parser '((atom "DEALLOCATE" true) (define global (? (atom "GLOBAL" true))) (define name tsql_identifier))
					(list "cursor-deallocate" (toLower name) (if global true false)))
				(parser '((atom "FETCH" true)
					(? (define movement (or
						(parser (define direction (or (atom "NEXT" true) (atom "PRIOR" true)
							(atom "FIRST" true) (atom "LAST" true))) (list (toLower direction) nil))
						(parser '((define direction (or (atom "ABSOLUTE" true) (atom "RELATIVE" true)))
							(define amount (capture (or tsql_int tsql_script_variable)))) (list (toLower direction) (car amount)))))
						(atom "FROM" true))
					(define global (? (atom "GLOBAL" true))) (define name tsql_identifier)
					(? (atom "INTO" true) (define targets (+ tsql_script_variable ","))))
					(list "cursor-fetch" (toLower name) (if global true false)
						(coalesceNil movement (list "next" nil)) targets))
				(parser '((atom "DECLARE" true) (define name tsql_script_variable)
					(? (atom "AS" true)) (atom "TABLE" true) "("
					(define columns (+ column_definition ",")) ")")
					(list "declare-table" name columns))
				(parser (define source (capture '((atom "INSERT" true) (? (atom "INTO" true))
					tsql_script_variable (? "(" (+ tsql_identifier ",") ")")
					(or '((atom "VALUES" true) (+ tsql_insert_values_row ","))
						'((atom "DEFAULT" true) (atom "VALUES" true))))))
					(list "insert-local-source" (car source)))
				(parser '((atom "INSERT" true) (? (atom "INTO" true))
					(define name tsql_script_variable)
					(? "(" (define columns (+ tsql_identifier ",")) ")")
					(define query (capture tsql_root_select)))
					(list "insert-local" name columns (car query)))
				(parser '((atom "DECLARE" true) (define declarations (+ tsql_script_declaration ",")))
					(list "declare" declarations))
				(parser '((atom "SET" true) (define name tsql_script_variable)
					(define operation (or (parser "&=" "&") (parser "|=" "|") (parser "^=" "^") (parser "=" nil)))
					(define value (capture tsql_expression)))
					(list "assign" name operation (car value)))
				(parser '((atom "DROP" true) (atom "TRIGGER" true)
					(define ignore (? (atom "IF" true) (atom "EXISTS" true)))
					(? (define owner tsql_schema_identifier)) (define name tsql_identifier))
					(list "drop-trigger" (tsql_qualifier_database owner schema frontend_options) name (if ignore true false)))
				(parser '((atom "DROP" true) (or (atom "PROCEDURE" true) (atom "PROC" true))
					(define ignore (? (atom "IF" true) (atom "EXISTS" true)))
					(? (define owner tsql_schema_identifier)) (define name tsql_identifier))
					(list "drop-procedure" (tsql_qualifier_database owner schema frontend_options) name (if ignore true false)))
				(parser '((atom "DROP" true) (atom "FUNCTION" true)
					(define ignore (? (atom "IF" true) (atom "EXISTS" true)))
					(define names (+ (parser '((? (define owner tsql_identifier) ".") (define name tsql_identifier))
						(begin
							(if (or (nil? owner) (equal?? owner "dbo")) true
								(error "unsupported scalar function schema: " owner))
							name)) ",")))
					(list "drop-function" names (if ignore true false)))
				(parser '((atom "INSERT" true) (? (atom "INTO" true))
					(? (define owner tsql_schema_identifier)) (define target tsql_identifier)
					(? "(" (define columns (+ tsql_identifier ",")) ")")
					(or (atom "EXEC" true) (atom "EXECUTE" true))
					(define invocation tsql_script_invocation))
					(list "insert-execute" (tsql_qualifier_database owner schema frontend_options)
						target columns invocation))
				(parser '((or (atom "EXEC" true) (atom "EXECUTE" true))
					"(" (define text (capture tsql_expression)) ")")
					(list "execute-dynamic" (car text)))
				(parser '((or (atom "EXEC" true) (atom "EXECUTE" true))
					(define invocation tsql_script_invocation)) invocation)
				(parser (define declaration (capture tsql_create_view))
					(tsql_context_statement lexical_context (strtrim (car declaration))))
				(parser (define statement (capture p)) (list "statement" (strtrim (car statement))))
			)))
			(define tsql_script_item (parser '((* ";") (define command tsql_script_command) (* ";")) command))
			(define tsql_script_procedure_parameter (parser '(
				(define name tsql_script_variable) (? (atom "AS" true))
				(define type tsql_declared_type) (define dimensions tsql_type_dimensions)
				/* Procedure defaults are literal declarations, unlike DECLARE
				initializers; they cannot capture an invocation's caller locals. */
				(? "=" (define initializer (capture (or tsql_literal (parser '("+" (define value tsql_number)) value)))))
				(define output (? (or (atom "OUTPUT" true) (atom "OUT" true))))
			) (list name (tsql_parameter_declaration schema type dimensions)
					(if initializer (car initializer) nil) (if output true false))))
			(define tsql_script_scalar_parameter (parser '(
				(define name tsql_script_variable) (? (atom "AS" true))
				(define type tsql_declared_type) (define dimensions tsql_type_dimensions)
				(? "=" (define initializer (capture tsql_parameter_default_literal)))
			) (begin
					(define declaration (list name (tsql_parameter_declaration schema type dimensions)))
					(if (nil? initializer) declaration (append declaration (car initializer))))))
			(define tsql_script_scalar_function (parser '(
				(atom "CREATE" true) (atom "FUNCTION" true)
				(? (define owner tsql_identifier) ".") (define name tsql_identifier)
				"(" (define parameters (* tsql_script_scalar_parameter ",")) ")"
				(atom "RETURNS" true) (define type tsql_declared_type) (define dimensions tsql_type_dimensions)
				(atom "AS" true) (define body (capture tsql_script_item)) (* ";")
			) (begin
					(if (or (nil? owner) (equal?? owner "dbo")) true
						(error "unsupported scalar function schema: " owner))
					(list "batch" (list (list "create-scalar-function" name parameters
						(tsql_parameter_declaration schema type dimensions)
						(coalesceNil (lexical_context "module_source") (strtrim (car body)))))))))
			(define tsql_script_table_function (parser '(
				(atom "CREATE" true) (atom "FUNCTION" true)
				(? (define owner tsql_identifier) ".") (define name tsql_identifier)
				"(" (define parameters (* tsql_script_scalar_parameter ",")) ")"
				(atom "RETURNS" true) (define return_name tsql_script_variable)
				(atom "TABLE" true) "(" (define columns (+ column_definition ",")) ")"
				(atom "AS" true) (define body (capture tsql_script_item)) (* ";")
			) (begin
					(if (or (nil? owner) (equal?? owner "dbo")) true
						(error "unsupported table function schema: " owner))
					(list "batch" (list (list "create-table-function" name parameters return_name columns
						(coalesceNil (lexical_context "module_source") (strtrim (car body)))))))))

			(define tsql_script_procedure (parser '(
				(define operation (or
					(parser '((atom "CREATE" true) (define replace (? (atom "OR" true) (atom "ALTER" true))))
						(list "create-procedure" (if replace true false)))
					(parser (atom "ALTER" true) (list "alter-procedure"))))
				(or (atom "PROCEDURE" true) (atom "PROC" true))
				(? (define owner tsql_schema_identifier)) (define name tsql_identifier)
				/* SQL Server accepts a parenthesized parameter header too; both
				forms use the same declaration/default/output grammar. Reference:
				https://learn.microsoft.com/sql/relational-databases/native-client-odbc-how-to/use-table-valued-parameters-odbc */
				(define parameters (or
					(parser '("(" (define declarations (* tsql_script_procedure_parameter ",")) ")") declarations)
					(* tsql_script_procedure_parameter ",")))
				(atom "AS" true) (define body (capture (+ tsql_script_item empty)))
			) (begin
					/* Plain ALTER retains its native must-exist operation; it is not
					an upsert or a frontend catalog check followed by CREATE. */
					(define definition (list (car operation) (tsql_qualifier_database owner schema frontend_options) name
						(tsql_unique_parameters parameters) (coalesceNil (lexical_context "module_source") (strtrim (car body)))))
					(list "batch" (list (if (equal? (car operation) "create-procedure")
						(append definition (cadr operation)) definition))))))
			/* Statement triggers retain validated SQL commands, not a row-trigger
			translation. The native completion owner supplies transition relations. */
			(define tsql_script_trigger (parser '(
				(define operation (or
					(parser '((atom "CREATE" true) (define replace (? (atom "OR" true) (atom "ALTER" true))))
						(list "create-trigger" (if replace true false)))
					(parser (atom "ALTER" true) (list "alter-trigger"))))
				(atom "TRIGGER" true)
				(? (define owner tsql_schema_identifier)) (define name tsql_identifier)
				(atom "ON" true)
				(? (define target_owner tsql_schema_identifier)) (define target tsql_identifier)
				(define timing (or
					(parser (or (atom "FOR" true) (atom "AFTER" true)) "after_statement")
					(parser '((atom "INSTEAD" true) (atom "OF" true)) "replace_statement")))
				(define events (+ (or (parser (atom "INSERT" true) "insert")
					(parser (atom "UPDATE" true) "update") (parser (atom "DELETE" true) "delete")) ","))
				(atom "AS" true) (define body (capture (+ tsql_script_item empty)))
			) (begin
					(if (equal? (count events) (count (merge_unique (list events)))) true
						(error "duplicate trigger event"))
					(define definition (list (car operation)
						(tsql_qualifier_database owner schema frontend_options) name
						(tsql_qualifier_database target_owner schema frontend_options) target
						events (coalesceNil (lexical_context "module_source") (strtrim (car body))) timing))
					(list "batch" (list (if (equal? (car operation) "create-trigger")
						(append definition (cadr operation)) definition))))))
			(define script_parser (parser (define batch (or tsql_script_trigger tsql_script_procedure tsql_script_table_function tsql_script_scalar_function
				(parser '((* ";") (? (define first (or tsql_script_command tsql_script_invocation)))
					(define commands (* tsql_script_item empty)) (* ";"))
					(list "batch" (if first (cons first commands) commands))))) batch "^(?:/\\*(?s:.*?)\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+"))

			(script_parser s))
		((parser '((* ";") (define command command_parser) (* ";")) (if (string? command) command (command)) "^(?:/\\*(?s:.*?)\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+") s))
)))

(define tsql_validate_script_control (lambda (commands loop_depth)
	(map commands (lambda (command) (match command
		'("block" body) (tsql_validate_script_control body loop_depth)
		'("if" _condition yes no) (tsql_validate_script_control (if no (list yes no) (list yes)) loop_depth)
		'("while" _condition body) (tsql_validate_script_control (list body) (+ loop_depth 1))
		'("try" body recovery) (tsql_validate_script_control (merge (list body recovery)) loop_depth)
		'("break") (if (> loop_depth 0) true (error "BREAK requires an enclosing WHILE"))
		'("continue") (if (> loop_depth 0) true (error "CONTINUE requires an enclosing WHILE"))
		_ true)))))

/* BEGIN/IF/WHILE do not create label scopes. TRY and CATCH do: a jump may
leave a protected region but must never enter one from outside. These records
contain source positions only; validation finishes before any SQL leaf runs. */
(define tsql_script_label_records (lambda (commands path region)
	(merge (map (produceN (count commands)) (lambda (index) (begin
		(define command (nth commands index))
		(define location (concat path "/" index))
		(match command
			'("label" name) (list (list "label" name region))
			'("goto" name) (list (list "goto" name region))
			'("block" body) (tsql_script_label_records body location region)
			'("if" _condition yes no) (tsql_script_label_records (if no (list yes no) (list yes)) location region)
			'("while" _condition body) (tsql_script_label_records (list body) location region)
			'("try" body recovery) (merge (list
				(tsql_script_label_records body (concat location "/try") (concat location "/try/"))
				(tsql_script_label_records recovery (concat location "/catch") (concat location "/catch/"))))
			_ (list))))))))
(define tsql_validate_script_labels (lambda (commands) (begin
	(define records (tsql_script_label_records commands "" ""))
	(define labels (reduce records (lambda (labels record)
		(if (equal? (car record) "label")
			(if (has_assoc? labels (cadr record)) (error "duplicate label: " (cadr record))
				(set_assoc labels (cadr record) (nth record 2))) labels)) (list)))
	(map records (lambda (record) (if (equal? (car record) "goto") (begin
		(define name (cadr record))
		(if (has_assoc? labels name) true (error "GOTO target is not defined in this batch: " name))
		(define target (labels name))
		(if (equal? (substr (nth record 2) 0 (strlen target)) target) true
			(error "GOTO cannot enter a TRY or CATCH region: " name))) true))))))

(define tsql_parse_batch_context (lambda (schema text quoted static_module ansi) (begin
	(define context (tsql_parse_context text quoted static_module))
	(define syntax (newsession))
	(syntax "__tsql_script_syntax" true)
	(syntax "__tsql_ansi_nulls" ansi)
	(syntax "__tsql_lexical_context" context)
	(define parsed (parse_tsql schema text nil syntax nil))
	(tsql_validate_script_control (cadr parsed) 0)
	(tsql_validate_script_labels (cadr parsed))
	(list "syntax" parsed "quoted_identifier" (context "quoted_identifier")))))
/* Standalone syntax inspection has no connection or stored module context. */
(define parse_tsql_batch (lambda (schema text)
	((tsql_parse_batch_context schema text true false true) "syntax")))
