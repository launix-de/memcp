/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* Independent T-SQL grammar. Statements emit the canonical query AST and
existing native DDL/DML operations, as the other SQL frontends do. */
/* Integer tokens and integer text conversions use native int64 arithmetic.
The negative accumulator also represents the minimum int64 without overflow. */
(define tsql_parse_integer (lambda (text) (begin
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
		(if negative coefficient (- 0 coefficient)))))))
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
(define tsql_identifier_unquoted (parser (not
	(regex "[a-zA-Z_][a-zA-Z0-9_]*")
	(atom "NOT" true)
	(atom "IN" true)
	(atom "BETWEEN" true)
	(atom "AS" true)
	(atom "ON" true)
	(atom "WHERE" true)
	(atom "GROUP" true)
	(atom "BY" true)
	(atom "VALUES" true)
	(atom "FROM" true)
	(atom "LEFT" true)
	(atom "RIGHT" true)
	(atom "INNER" true)
	(atom "OUTER" true)
	(atom "CROSS" true)
	(atom "JOIN" true)
	(atom "SELECT" true)
	(atom "UNION" true)
	(atom "ALL" true)
	(atom "INSERT" true)
	(atom "SET" true)
	(atom "ORDER" true)
	(atom "LIMIT" true)
	(atom "TRIM" true)
	(atom "LTRIM" true)
	(atom "RTRIM" true)
	(atom "TOP" true)
	(atom "OFFSET" true)
	(atom "FETCH" true)
	(atom "ROWS" true)
	(atom "ROW" true)
	(atom "ONLY" true)
	(atom "CASE" true)
	(atom "WHEN" true)
	(atom "THEN" true)
	(atom "ELSE" true)
	(atom "END" true)
	(atom "HAVING" true)
	(atom "WITH" true)
	(atom "OPTION" true)
	(atom "IF" true)
	(atom "BEGIN" true)
	(atom "DECLARE" true)
	(atom "EXEC" true)
	(atom "EXECUTE" true)
	(atom "CREATE" true)
	(atom "ALTER" true)
	(atom "DROP" true)
	(atom "UPDATE" true)
	(atom "DELETE" true)
	(atom "COMMIT" true)
	(atom "ROLLBACK" true)
	(atom "WHILE" true)
	(atom "BREAK" true)
	(atom "CONTINUE" true)
	(atom "RETURN" true)
	(atom "THROW" true)

)))
(define tsql_identifier_quoted (parser (or
	(parser '("[" (define id (regex "(?:[^\\]]|\\]\\])+" false false)) "]") (replace id "]]" "]"))
	(parser '("\"" (define id (regex "(?:[^\"]|\"\")+" false false)) "\"") (replace id "\"\"" "\"")))))
(define tsql_identifier (parser (or tsql_identifier_quoted tsql_identifier_unquoted)))
/* Resolve spelling at the frontend boundary. Native metadata owns table and
column names; views remain definitions in the existing system.views table.
Only consumed name bindings enter the shared plan guards, never catalog epochs. */
(define tsql_view_name (lambda (database name)
	(scan nil (table "system" false "views" false)
		(list 369436175368192 (scan_boundary "equal" "database" 0 0 true true "" false))
		(list database) '("name") (lambda (candidate) (equal?? candidate name)) '("name")
		(lambda (acc canonical) (if (nil? acc) canonical (error "ambiguous view name: " name)))
		nil (lambda (a b) (if (nil? a) b (nil? b) a (error "ambiguous view name: " name))))))
(define tsql_relation_name (lambda (database name)
	(coalesceNil (show (table database false name true) "name") (tsql_view_name database name) name)))
(define tsql_table_name (lambda (database name planning_session) (begin
	(define canonical (show (table database false name true) "name"))
	(planner_record_guard_condition
		(list (quote equal?) (list (quote show) (list (quote table) database false name true) "name") canonical)
		planning_session)
	(coalesceNil canonical name))))
(define tsql_create_relation (lambda (database name columns options ifnotexists tx) (begin
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
	(if (show (table database false name true) "name") (error "a table already owns this relation name: " name) true)
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

/* The @ prefix makes this a variable token; SQL keywords need no identifier
escaping here. Consume it as one token so whitespace cannot split the name. */
(define tsql_variable_name (parser (define name (regex "@[a-zA-Z_][a-zA-Z0-9_]*"))
	(toLower (substr name 1))))
(define tsql_string_token (parser '((define national (or
	(parser (atom "N'" true) true) (parser (atom "'" false) false)))
	(define value (regex "(?:''|[^'])*" false false)) (atom "'" false false))
	(list (replace value "''" "'") national)))
(define tsql_string (parser (define token tsql_string_token) (car token)))
(define tsql_text_literal (parser (define token tsql_string_token)
	(list (quote sql_type_expr) (car token) (if (cadr token) "NVARCHAR" "VARCHAR") nil)))
(define tsql_column (parser (or
	(parser '((define alias tsql_identifier) "." (define name tsql_identifier))
		(list (quote sql_compile_column) (quote tsql_compile_column) '('get_column alias true name true)))
	(parser (define name tsql_identifier)
		(list (quote sql_compile_column) (quote tsql_compile_column) '('get_column nil true name true))))))
(define tsql_hex_literal (parser (define value sql_hex_literal)
	(list (quote sql_type_expr) value "VARBINARY" nil)))
(define tsql_literal (parser (or
	(parser (atom "NULL" true) (sql_null_literal)) tsql_hex_literal tsql_number tsql_text_literal)))
/* Binary semantics are chosen from declared SQL contracts at the shared
compile boundary. Byte equality reuses the native collation-aware operator;
no SQL type wrapper survives into execution or storage. */
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
	(define comparison (if (and
		(or (tsql_binary_type? left_type) (equal? left_type "NULL"))
		(or (tsql_binary_type? right_type) (equal? right_type "NULL"))
		(or (tsql_binary_type? left_type) (tsql_binary_type? right_type)))
		(sql_info (if (has? '("eq" "ne") op)
			(list (quote equal_collate) (sql_info_formula left_info) (sql_info_formula right_info) "bin")
			(list (quote sql_compare) (sql_info_formula left_info) (sql_info_formula right_info)
				(collate "bin" false) (string head))) "BOOLEAN" nil)
		(sql_comparison_info head left_info right_info)))
	(sql_info (if (equal? op "ne") (list (quote sql_not) (sql_info_formula comparison))
		(sql_info_formula comparison)) "BOOLEAN" (sql_info_collation comparison))))))
/* Predicate syntax supplies its Boolean contract before the shared logical
markers are decorrelated. No executable annotation survives SQL compilation. */
(define tsql_predicate_expr (lambda (formula)
	(list (quote sql_type_expr) formula "BOOLEAN" nil)))
(define tsql_comparison_expr (lambda (op left right)
	(list (quote sql_compile_expr) (list (quote tsql_comparison_compiler) op) left right)))
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
(define tsql_numeric_input (lambda (value)
	(if (number? value) (if (and (>= value (- 0 tsql_float_max)) (<= value tsql_float_max)) value (tsql_raise_error 8115 "floating conversion overflow" 1))
		(if (string? value) (begin
			(define text (sql_trim value))
			(if (equal? text "") 0
				(if (regexp_test text "^[+-]?(?:[0-9]+\\.?[0-9]*|\\.[0-9]+)(?:[eE][+-]?[0-9]+)?$")
					(begin (define number (simplify text))
						(if (and (number? number) (>= number (- 0 tsql_float_max)) (<= number tsql_float_max)) number
							(tsql_raise_error 8115 "floating conversion overflow" 1)))
					(tsql_raise_error 8114 "invalid numeric text" 1))))
			(if (equal? value true) 1 (if (equal? value false) 0 (error "unsupported numeric conversion")))))))
(define tsql_cast_integer (lambda (value minimum maximum)
	(if (nil? value) nil (begin
		(define integer (if (string? value) (tsql_parse_integer value)
			(if (int? value) value (intdiv (tsql_numeric_input value) 1))))
		(if (or (nil? integer) (< integer minimum) (> integer maximum))
			(tsql_raise_error 8115 "integer conversion overflow" 1) integer)))))
(define tsql_cast_float (lambda (value) (if (nil? value) nil (simplify (tsql_numeric_input value)))))
(define tsql_cast_decimal (lambda (value precision scale) (if (nil? value) nil (begin
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
	(define bound (simplify (concat "1e" (- precision scale))))
	(if (and (> converted (- 0 bound)) (< converted bound)) converted
		(tsql_raise_error 8115 "decimal conversion overflow" 1))))))
/* Currency declarations have fixed scale four. Values remain ordinary native
numbers; only formatting and transport use an unscaled integer coefficient.
The float carrier cannot preserve every coefficient at the 64-bit endpoints.
Reject an out-of-range conversion before intdiv can wrap its sign. */
(define tsql_money_coefficient (lambda (value small) (if (nil? value) nil (begin
	(define scaled (round (* (sql_decimal_output (tsql_numeric_input value) 4) 10000)))
	(define limit (simplify "9223372036854775808.0"))
	/* At either 64-bit endpoint the float carrier cannot distinguish the
	last valid coefficient from an adjacent overflow. Refuse that ambiguity. */
	(if (or (<= scaled (- 0 limit)) (>= scaled limit))
		(tsql_raise_error 8115 "currency conversion overflow" 1) true)
	(define coefficient (intdiv scaled 1))
	(if (and small (or (< coefficient -2147483648) (> coefficient 2147483647)))
		(tsql_raise_error 8115 "small currency conversion overflow" 1) coefficient)))))
(define tsql_cast_money (lambda (value small)
	(if (nil? value) nil (/ (tsql_money_coefficient value small) 10000))))
(define tsql_money_text_input (lambda (value) (if (nil? value) nil (begin
	(define text (sql_trim value))
	(if (or (equal? text "")
		(regexp_test text "^(?:[+-]?\\p{Sc}?|\\p{Sc}[+-]?)(?:[0-9][0-9,]*(?:\\.[0-9]*)?|\\.[0-9]+)$")) true
		(tsql_raise_error 235 "invalid currency text" 1))
	(tsql_numeric_input (regexp_replace (replace text "," "") "\\p{Sc}" ""))))))
(define tsql_money_compiler (lambda (small) (lambda (infos) (begin
	(define input (car infos))
	(define type (sql_info_type input))
	(define character (has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR" "TEXT" "NTEXT") type))
	(if (or (sql_numeric_type? type) character (has? '("MONEY" "SMALLMONEY" "NULL") type)) true
		(error "currency conversion from " type " is not implemented"))
	(define formula (sql_info_formula input))
	(sql_info (list (quote tsql_cast_money)
		(if character (list (quote tsql_money_text_input) formula) formula) small)
		(if small "SMALLMONEY" "MONEY") nil)))))
(define tsql_cast_money_expr (lambda (value small)
	(list (quote sql_compile_expr) (list (quote tsql_money_compiler) small) value)))
/* These bounded strings describe a scale-four currency value, not a second
numeric representation. Split before taking absolute values so the signed
64-bit minimum never needs to be negated. */
(define tsql_money_group_digits (lambda (text)
	(if (<= (strlen text) 3) text
		(concat (tsql_money_group_digits (substr text 0 (- (strlen text) 3))) "," (substr text (- (strlen text) 3))))))
(define tsql_money_text (lambda (value style) (if (nil? value) nil (begin
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
		"." (string_repeat "0" (- (if four 4 2) (strlen tail))) tail)))))
(define tsql_cast_money_text (lambda (value style length national fixed) (if (nil? value) nil (begin
	(define text (tsql_money_text value style))
	(if (and (not (equal? length -1)) (> (strlen text) length))
		(tsql_raise_error 8115 "currency character conversion overflow" 1) true)
	((if fixed tsql_cast_fixed_text tsql_cast_text) text length national)))))
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
(define tsql_binary_to_integer (lambda (value width signed) (if (nil? value) nil (begin
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
			(- digit 16) (+ (* acc 16) digit)))) 0)))))
(define tsql_integer_compiler (lambda (type minimum maximum) (lambda (infos) (begin
	(define input (car infos))
	(sql_info (if (tsql_binary_type? (sql_info_type input))
		(list (quote tsql_binary_to_integer) (sql_info_formula input) (tsql_integer_width type) (< minimum 0))
		(list (quote tsql_cast_integer)
			(if (has? '("MONEY" "SMALLMONEY") (sql_info_type input))
				(list (quote tsql_money_integer) (sql_info_formula input)) (sql_info_formula input)) minimum maximum)) type nil)))))
(define tsql_money_integer (lambda (value) (if (nil? value) nil (round value))))
(define tsql_cast_integer_expr (lambda (value type minimum maximum)
	(list (quote sql_compile_expr) (list (quote tsql_integer_compiler) type minimum maximum) value)))
(define tsql_cast_bit (lambda (value)
	(if (nil? value) nil
		(if (and (string? value) (equal? (toUpper (sql_trim value)) "TRUE")) true
			(if (and (string? value) (equal? (toUpper (sql_trim value)) "FALSE")) false
				(not (equal? (tsql_numeric_input value) 0)))))))
/* Character casts own their width semantics in the frontend. National text
counts UTF-16 units and never returns half of a supplementary character. The
fast path avoids splitting the ordinary short numeric/text conversions. */
(define tsql_cast_text (lambda (value length national) (if (nil? value) nil (begin
	(define text (string value))
	(if (or (equal? length -1) (<= (strlen text) length)) text
		national (utf16_prefix text length)
		(cadr (reduce (split text "") (lambda (prefix character) (begin
			(define units (strlen character))
			(if (< (car prefix) units) (list 0 (cadr prefix))
				(list (- (car prefix) units) (concat (cadr prefix) character)))))
			(list length ""))))))))
(define tsql_cast_fixed_text (lambda (value length national) (if (nil? value) nil (begin
	(define text (tsql_cast_text value length national))
	(define units (if national (utf16_length text) (strlen text)))
	(concat text (string_repeat " " (- length units)))))))
/* A cast keeps its syntax-declared SQL type until the shared expression
compiler consumes the (expr, type, collation) contract. Runtime values do not
select that type, and no arithmetic operation is needed merely to mark it. */

(define tsql_cast_expr (lambda (value type dimensions) (begin
	(define name (toUpper type))
	(if (tsql_builtin_type_declaration name) (error "alias types cannot be explicit conversion targets") true)
	(tsql_decl_dimensions name dimensions)
	(define formula (match name
		"INT" (tsql_cast_integer_expr value name -2147483648 2147483647)
		"INTEGER" (tsql_cast_integer_expr value name -2147483648 2147483647)
		"SMALLINT" (tsql_cast_integer_expr value name -32768 32767)
		"TINYINT" (tsql_cast_integer_expr value name 0 255)
		"BIGINT" (tsql_cast_integer_expr value name (tsql_parse_integer "-9223372036854775808") (tsql_parse_integer "9223372036854775807"))
		"FLOAT" (list (quote tsql_cast_float) value)
		/* Approximate types use native floating precision. Binary32 rounding and
		dialect precision boundaries remain noncritical compatibility work. */
		"REAL" (list (quote tsql_cast_float) value)
		"MONEY" (tsql_cast_money_expr value false)
		"SMALLMONEY" (tsql_cast_money_expr value true)
		"BIT" (list (quote tsql_cast_bit) value)
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
	(list (quote sql_type_expr) formula name nil))))
(define tsql_decimal_compiler (lambda (type dims) (lambda (infos) (begin
	(define input (car infos))
	(define input_type (sql_info_type input))
	(if (or (sql_numeric_type? input_type)
		(has? '("NULL" "any" "MONEY" "SMALLMONEY" "CHAR" "VARCHAR" "NCHAR" "NVARCHAR" "TEXT" "NTEXT") input_type)) true
		(error "decimal conversion from " input_type " is not implemented"))
	(sql_info (list (quote tsql_cast_decimal) (sql_info_formula input) (car dims) (cadr dims)) type nil)))))
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
		(if fixed "BINARY" "VARBINARY") nil)))))
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
		(declaration "AliasId") (tsql_cast_expr value type (declaration "Dimensions"))
		(has? '("INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT") type)
		(tsql_cast_expr value type '())
		/* Minute rounding belongs to the declared assignment contract, before
		the existing sink receives the native calendar value. */
		(has? '("NVARCHAR" "NCHAR" "SMALLDATETIME") type)
		(tsql_cast_expr value type (declaration "Dimensions"))
		(has? '("MONEY" "SMALLMONEY") type) (tsql_cast_expr value type '())
		(has? '("DECIMAL" "NUMERIC") type) (begin
			(define dimensions (declaration "Dimensions"))
			(define precision (if (empty_list? dimensions) 18 (car dimensions)))
			(define scale (if (< (count dimensions) 2) 0 (cadr dimensions)))
			(define threshold (- (simplify (concat "1e" (- precision scale)))
				(* 0.5 (simplify (concat "1e-" scale)))))
			(list (quote sql_type_expr) (list (quote tsql_assignment_decimal) value threshold) type nil))
		value))))
/* DEFAULT is frontend syntax for the existing column declaration. Share this
translation between explicit INSERT cells and UPDATE assignments; omitted
INSERT columns continue through the native default mechanism unchanged. */
(define tsql_default_assignment_expr (lambda (declaration)
	(if (equal?? (declaration "DefaultExpression") "CURRENT_TIMESTAMP")
		(list (quote sql_type_expr) (list (quote now)) "DATETIME" nil)
		(and (nil? (declaration "Default")) (not (declaration "Null")))
		(error "nonnullable column has no usable DEFAULT: " (declaration "Field"))
		(list (quote sql_type_expr) (declaration "Default") (declaration "Type") (declaration "Collation")))))
(define tsql_assignment_value (lambda (expression declaration)
	(if (equal?? expression (quote sql_default)) (tsql_default_assignment_expr declaration) expression)))
/* Only declaration consumers register these O(1) guards. The common cache
recompiles a variant after DDL, without a schema epoch or runtime SQL type tag. */
(define tsql_declaration_contract (lambda (column)
	(list (column "RawType") (coalesceNil (column "Dimensions") '())
		(column "Null") (coalesceNil (column "Collation") "")
		(column "Default") (coalesceNil (column "DefaultExpression") "") (column "Extra"))))
(define tsql_declaration_matches? (lambda (column contract)
	(if (nil? column) false (expression_equal? (tsql_declaration_contract column) contract))))
(define tsql_column_description (lambda (database name column ignore_case planning_session) (begin
	(define handle (table database false name false))
	(if (nil? handle) nil (tsql_effective_column (show handle "column" column ignore_case) planning_session)))))
(define tsql_target_column_declaration (lambda (database name column planning_session) (begin
	(define declaration (tsql_column_description database name column true planning_session))
	(if (nil? declaration) nil (begin
		(if (declaration "AliasId")
			(tsql_guard_type_alias (declaration "DeclaredType") (declaration "AliasId") planning_session) true)
		(planner_record_guard_condition
			(list (quote tsql_declaration_matches?)
				(list (quote tsql_column_description) database name (declaration "Field") false nil)
				(list (quote quote) (tsql_declaration_contract declaration))) planning_session)
		declaration)))))
/* An omitted target list supplies only writable ordinary columns. Generated
identity/version declarations are frontend policy; their native storage and
allocation stay unchanged. Other dialects keep their own implicit-list rules. */
(define tsql_insert_target_columns (lambda (database name)
	(map (filter (get_schema database name) (lambda (column)
		(and (not (internal_column_name? (column "Field")))
			(not (equal? (column "Extra") "auto_increment"))
			(not (equal?? (column "RawType") "ROWVERSION")))))
		(lambda (column) (column "Field")))))
(define tsql_insert_declarations (lambda (database table_name columns planning_session) (begin	/* Resolve each target exactly once. Duplicate targets can overwrite an
	already validated identity/default value in the native column vector. */
	/* An empty vector has no column lookup that could reject an absent table. */
	(if (empty_list? columns)
		(if (table database false table_name false) true
			(error "INSERT target table does not exist: " table_name)) true)
	(if (equal? (count columns) (count (merge_unique (list (map columns toLower)))))
		true (error "duplicate INSERT column"))
	(map columns (lambda (name) (begin
		(define declaration (tsql_target_column_declaration database table_name name planning_session))
		(if (nil? declaration) (error "unknown INSERT column: " name) declaration)))))))
/* English legacy calendar text uses the existing Unix date primitives. The
bounded month directory is frontend syntax, not a second temporal carrier. */
(define tsql_calendar_month_names '("Jan" "Feb" "Mar" "Apr" "May" "Jun" "Jul" "Aug" "Sep" "Oct" "Nov" "Dec"))
(define tsql_calendar_month_output (lambda (value) (if (nil? value) nil (begin
	(define hour (extract_date value "HOUR"))
	(define clock (if (equal? (mod hour 12) 0) 12 (mod hour 12)))
	(define day (extract_date value "DAY"))
	(concat (nth tsql_calendar_month_names (- (extract_date value "MONTH") 1)) " "
		(if (< day 10) " " "") day " " (format_date value "%Y") " "
		(if (< clock 10) " " "") clock ":" (format_date value "%i") (if (< hour 12) "AM" "PM"))))))
(define tsql_calendar_month_input (lambda (value) (if (nil? value) nil (begin
	(define text (regexp_replace (toUpper (sql_trim value)) "\\s+" " "))
	(if (regexp_test text "^[A-Z]{3} [0-9]{1,2} [0-9]{4} [0-9]{1,2}:[0-9]{2}(AM|PM)$")
		true (error "invalid month-name calendar input"))
	(match (split text " ")
		'(month day year clock) (begin
			(define month_index (find (produceN 12) (lambda (index)
				(equal? month (toUpper (nth tsql_calendar_month_names index)))) nil))
			(if (nil? month_index) (error "invalid calendar month") true)
			(define hour (tsql_parse_integer (substr clock 0 (- (strlen clock) 5))))
			(if (and (>= hour 1) (<= hour 12)) true (error "invalid twelve-hour calendar clock"))
			(define hour (+ (mod hour 12) (if (equal? (substr clock (- (strlen clock) 2)) "PM") 12 0)))
			(define month (+ month_index 1))
			(define date (concat year "-" (if (< month 10) "0" "") month "-"
				(if (equal? (strlen day) 1) "0" "") day " "
				(if (< hour 10) "0" "") hour ":" (substr clock (- (strlen clock) 4) 2) ":00"))
			(define parsed (str_to_date date "%Y-%m-%d %H:%i:%s"))
			(if (nil? parsed) (error "invalid calendar date") parsed)))))))
/* Numeric text width and Boolean spelling follow the declared input type.
Only the small integer family uses the ASCII overflow marker; BIGINT and
national text retain their conversion error. Runtime carriers select no type. */
(define tsql_cast_numeric_text (lambda (value length national fixed asterisk boolean)
	(if (nil? value) nil (begin
		(define text (string (if boolean (if value 1 0) value)))
		(define text (if (and (not (equal? length -1)) (> (strlen text) length))
			(if asterisk "*" (tsql_raise_error 8115 "numeric character conversion overflow" 1)) text))
		((if fixed tsql_cast_fixed_text tsql_cast_text) text length national)))))
(define tsql_text_compiler (lambda (length national fixed) (lambda (infos) (begin
	(define info (car infos))
	(define source (sql_info_type info))
	(define formula (if (tsql_temporal_type? source)
		(if (has? '("DATETIME" "SMALLDATETIME") source)
			(list (quote tsql_calendar_month_output) (tsql_calendar_difference_formula info))
			(list format_date (tsql_calendar_difference_formula info)
				(match source "DATE" "%Y-%m-%d" "TIME" "%H:%i:%s"
					"DATETIME2" "%Y-%m-%d %H:%i:%s.000" _ "%Y-%m-%d %H:%i:%s")))
		(sql_info_formula info)))
	(define converted (if (has? '("MONEY" "SMALLMONEY") source)
		(list (quote tsql_cast_money_text) formula 0 length national fixed)
		(sql_numeric_type? source)
		(list (quote tsql_cast_numeric_text) formula length national fixed
			(and (not national) (has? '("TINYINT" "SMALLINT" "INT" "INTEGER") source))
			(has? '("BIT" "BOOL" "BOOLEAN") source))
		(list (if fixed (quote tsql_cast_fixed_text) (quote tsql_cast_text)) formula length national)))
	(sql_info converted
		(if (and national fixed) "NCHAR" national "NVARCHAR" fixed "CHAR" "VARCHAR")
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
	(sql_info (list (if (has? '("CHAR" "NCHAR") type) (quote tsql_cast_fixed_text) (quote tsql_cast_text))
		formula (if (empty_list? dimensions) 30 (car dimensions)) (has? '("NVARCHAR" "NCHAR") type)) type nil)))))
(define tsql_convert_text_compiler (lambda (style type dimensions) (lambda (infos) (begin
	(define input (car infos))
	(define source (sql_info_type input))
	(define length (if (empty_list? dimensions) 30 (car dimensions)))
	(define national (has? '("NVARCHAR" "NCHAR") type))
	(define fixed (has? '("CHAR" "NCHAR") type))
	(if (has? '("MONEY" "SMALLMONEY") source)
		(sql_info (list (quote tsql_cast_money_text) (sql_info_formula input) style
			length national fixed) type nil)
		(tsql_temporal_type? source) (if (equal? style 100)
			(if (has? '("DATETIME" "SMALLDATETIME") source)
				((tsql_text_compiler length national fixed) infos)
				(error "month-name output style requires a legacy calendar declaration"))
			(and (equal? source "TIME") (has? '(102 103 105 110 111) style))
			(error "date-only output style requires a calendar date")
			((tsql_styled_text_compiler (tsql_temporal_style_pattern style true) type dimensions) infos))
		(tsql_binary_type? source) (begin
			(if (has? '(1 2) style) true (error "unsupported binary conversion style"))
			(sql_info (list (if fixed (quote tsql_cast_fixed_text) (quote tsql_cast_text))
				(list (quote tsql_binary_to_hex) (sql_info_formula input) (equal? style 1) length)
				length national) type nil))
		((tsql_text_compiler length national fixed) infos))))))
(define tsql_convert_expr (lambda (value type dimensions style) (begin
	(if (tsql_builtin_type_declaration type) (error "alias types cannot be explicit conversion targets") true)
	(tsql_decl_dimensions type dimensions)
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
		(sql_info_type converted)
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
(define tsql_aggregate_compiler (lambda (name) (lambda (infos)
	(tsql_aggregate_result name (sql_info_formula (car infos))
		(tsql_aggregate_type name (sql_info_type (cadr infos)))
		(sql_info_collation (cadr infos))))))
(define tsql_aggregate_expr (lambda (name formula value)
	(list (quote sql_compile_expr) (list (quote tsql_aggregate_compiler) name) formula value)))
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
(define tsql_calendar_text (lambda (value) (if (nil? value) nil (begin
	(define text (sql_trim value))
	(if (regexp_test text "^[A-Za-z]{3}\\s") (tsql_calendar_month_input text) (begin
		(define anchored (if (regexp_test text "^[0-9]{2}:[0-9]{2}:[0-9]{2}(\\.[0-9]+)?$")
			(concat "1900-01-01 " text) text))
		(define parsed (parse_date (regexp_replace anchored "T" " ")))
		(if (nil? parsed) (tsql_raise_error 241 "invalid calendar input" 1) parsed)))))))
(define tsql_calendar_days (lambda (value) (if (nil? value) nil (begin
	(if (and (>= value -53690) (< value 2958464)) true (tsql_raise_error 242 "numeric datetime conversion overflow" 1))
	(define days (intdiv value 1))
	(define seconds (intdiv (* (- value days) 86400) 1))
	(date_add (date_add (parse_date "1900-01-01") days "DAY") seconds "SECOND")))))
(define tsql_calendar_formula (lambda (info) (begin
	(define formula (sql_info_formula info))
	(define type (sql_info_type info))
	(if (equal? type "NULL") nil
		(has? '("DATE" "DATETIME" "DATETIME2" "SMALLDATETIME") type) (list parse_date formula)
		(sql_numeric_type? type) (list tsql_calendar_days formula)
		(sql_text_type? type) (list tsql_calendar_text formula)
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
(define tsql_boundary_calculator (lambda (unit) (match unit
	"year" (sql_builtins "YEAR") "month" tsql_calendar_month_index "quarter" tsql_calendar_quarter_index
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
(define tsql_calendar_range (lambda (value minimum)
	(if (nil? value) nil (if (and (>= (unix_timestamp value) minimum) (<= (unix_timestamp value) 253402300799))
		value (tsql_raise_error 242 "calendar result overflow" 1)))))
/* Temporal contracts are compiled from SQL declarations. The native carrier
is the existing Unix date value; subsecond precision remains deferred rather
than introducing another runtime datatype or precision-emulation layer. */
(define tsql_temporal_type? (lambda (type)
	(has? '("DATE" "DATETIME" "DATETIME2" "SMALLDATETIME" "TIME") type)))
(define tsql_calendar_time_only (lambda (value) (if (nil? value) nil
	(date_add (parse_date "1900-01-01")
		(intdiv (mod (+ (mod (unix_timestamp value) 86400) 86400) 86400) 1) "SECOND"))))
(define tsql_calendar_small_datetime (lambda (value) (if (nil? value) nil (begin
	(define seconds (unix_timestamp value))
	(define rounded (date_add value (intdiv (- (* 60 (floor (/ (+ seconds 30) 60))) seconds) 1) "SECOND"))
	(if (and (>= (unix_timestamp rounded) -2208988800) (<= (unix_timestamp rounded) 3453321540))
		rounded (tsql_raise_error 242 "small datetime conversion overflow" 1))))))
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
(define tsql_temporal_style_text (lambda (value style) (if (nil? value) nil
	(equal? style 100) (tsql_calendar_month_input value) (begin
		(define text (sql_trim value))
		(define pattern (tsql_temporal_style_pattern style false))
		(define text (if (has? '(1 12) style) (begin
			(define offset (if (equal? style 12) 0 6))
			(if (< (strlen text) (+ offset 2)) (tsql_raise_error 241 "invalid short calendar year" 1) true)
			(define year (substr text offset 2))
			(if (regexp_test year "^[0-9]{2}$") true (tsql_raise_error 241 "invalid short calendar year" 1))
			(concat (substr text 0 offset) (if (< (tsql_parse_integer year) 50) "20" "19")
				year (substr text (+ offset 2)))) text))
		(define value (if (has? '(21 121) style) (parse_date text) (str_to_date text pattern)))
		(if (nil? value) (tsql_raise_error 241 "invalid styled calendar input" 1) value)))))
/* The current native carrier has whole seconds. Explicit conversions that
would retain fractional clocks fail until a generic higher-precision carrier
is available; accepting them here would silently lose the user's value. */
(define tsql_calendar_cast_text (lambda (value) (if (nil? value) nil (begin
	(if (regexp_test value "[0-9]{2}:[0-9]{2}:[0-9]{2}\\.[0-9]*[1-9]")
		(if (nil? (tsql_calendar_text value)) value
			(error "fractional temporal conversion is not implemented")) value)))))
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
	(if (and (sql_numeric_type? source) (not (has? '("DATETIME" "SMALLDATETIME") target)))
		(error "numeric input is not convertible to " target) true)
	(if (not (has? '(0 4 100) style))
		(tsql_temporal_style_pattern style false) true)
	(define formula (if (and (sql_text_type? source) (not (equal? style 0)))
		(if (equal? style 4) (list tsql_calendar_shift_seconds (parse_date "1970-01-01")
			(list tsql_export_datetime_seconds (sql_info_formula input)))
			(list tsql_temporal_style_text (sql_info_formula input) style))
		(tsql_calendar_difference_formula input)))
	(define converted (match target
		"DATE" (list tsql_calendar_range (list date_trunc_day formula) -62135596800)
		"TIME" (list tsql_calendar_time_only formula)
		"SMALLDATETIME" (list tsql_calendar_small_datetime formula)
		_ (list tsql_calendar_range formula (if (equal? target "DATETIME") -6847804800 -62135596800))))
	(sql_info converted target nil)))))
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
	(define integer (list (quote tsql_cast_integer) (sql_info_formula amount_info) -2147483648 2147483647))
	(define formula (list (if months tsql_calendar_shift_months tsql_calendar_shift_seconds)
		(tsql_calendar_difference_formula input)
		(list (quote *) integer (coalesceNil months seconds))))
	(sql_info (match type
		"TIME" (list tsql_calendar_time_only formula)
		"SMALLDATETIME" (list tsql_calendar_small_datetime formula)
		_ (list tsql_calendar_range formula minimum)) type nil)))))
(define tsql_dateadd_expr (lambda (unit amount value) (begin
	(define unit (tsql_calendar_unit unit))
	(if (equal? unit "iso_week") (error "DATEADD does not support ISO week") true)
	(define months (match unit "year" 12 "quarter" 3 "month" 1 _ nil))
	(define seconds (match unit "second" 1 "minute" 60 "hour" 3600 "week" 604800 _ 86400))
	(list (quote sql_compile_expr) (list (quote tsql_dateadd_compiler) unit months seconds) amount value))))
(define tsql_calendar_difference_formula (lambda (info)
	(if (equal? (sql_info_type info) "TIME") (list tsql_calendar_text (sql_info_formula info))
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
		(define difference (list (quote -)
			(list boundary (tsql_calendar_difference_formula (cadr infos)))
			(list boundary (tsql_calendar_difference_formula (car infos)))))
		(sql_info (list (quote tsql_cast_integer) difference minimum maximum) type nil))))))
(define tsql_datediff_expr (lambda (unit first last type) (begin
	(define unit (tsql_calendar_unit unit))
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
(define tsql_compile_positive (lambda (infos) (begin
	(define info (car infos))
	(if (tsql_numeric_operand_type? (sql_info_type info)) info
		(error "unary plus requires a numeric operand")))))
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
(define tsql_character_type? (lambda (type)
	(has? '("NULL" "CHAR" "VARCHAR" "NCHAR" "NVARCHAR") type)))
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
			(if (or (nil? parts) (> (count parts) 4) (nil? (car (reverse parts))) (> piece (count parts))) nil
				(begin
					(define part (nth parts (- (count parts) piece)))
					(if (and part (tsql_sysname? part)) part nil)))))))
(define tsql_compile_parsename (lambda (infos) (begin
	(if (equal? (count infos) 2) true (error "PARSENAME has an invalid argument count"))
	(define text_info (car infos))
	(define piece_info (cadr infos))
	(define piece (sql_type_formula '() (tsql_cast_expr
		(list (quote sql_type_expr) (sql_info_formula piece_info) (sql_info_type piece_info) (sql_info_collation piece_info))
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
		(list (quote tsql_cast_money) negated (equal? type "SMALLMONEY")) negated)
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
(define tsql_identity_mode_live? (lambda (mode)
	(if (nil? mode) false (begin
		(define handle (cadr mode))
		(define current (table (car mode) false (show handle "name") false))
		(if (identical? current handle)
			(begin (define declaration (show handle "column" (nth mode 2) false))
				(if (nil? declaration) false (equal? (declaration "Extra") "auto_increment"))) false)))))
(define tsql_set_identity_insert (lambda (session database name enabled) (begin
	((sql_policy (coalesceNil (session "username") "root")) database name true)
	(define handle (table database false name true))
	(if (nil? handle) (tsql_raise_error 208 "IDENTITY_INSERT target does not exist" 1) true)
	(define identity (find (show handle) (lambda (column) (equal? (column "Extra") "auto_increment")) nil))
	(if (nil? identity) (tsql_raise_error 8106 "IDENTITY_INSERT requires an IDENTITY column" 1) true)
	(define previous (session "__tsql_identity_insert"))
	(define same (if (nil? previous) false (identical? handle (cadr previous))))
	(if enabled
		(begin
			(if (if same false (tsql_identity_mode_live? previous))
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
(define tsql_insert_identity (lambda (database name planning_session) (begin
	(define identity (find (get_schema database name)
		(lambda (column) (equal? (column "Extra") "auto_increment")) nil))
	(if identity (tsql_target_column_declaration database name (identity "Field") planning_session)
		(begin
			/* Absence consumes a fact about every declaration, not only the
			provided targets. Enumerate their immutable metadata, never rows:
			DROP/CREATE or ADD may introduce identity beside unchanged targets. */
			(planner_record_guard_condition
				(list (quote nil?) (list (quote find) (list (quote get_schema) database name)
					(list (quote lambda) (list (quote column))
						(list (quote equal?) (list (quote column) "Extra") "auto_increment")) nil)) planning_session)
			nil)))))
(define tsql_identity_insert_statement (lambda (database name identity explicit_columns columns body)
	(list (quote begin)
		(list (quote define) (quote __tsql_insert_target) (list (quote table) database false name false))
		(if identity
			(list (quote tsql_check_identity_insert) (quote session) (quote __tsql_insert_target)
				explicit_columns (has? columns (identity "Field"))) true)
		body)))
/* Emit the existing native INSERT. VALUES and DEFAULT supply their bounded
syntax batch; the existing SELECT result sink supplies one emitted row. Never
scan/count/materialize SELECT input to reserve IDs, or rewind consumed blocks.
The frontend checks the complete reserved block before any row publication.
Explicit NULL must fail here: native auto_increment would otherwise allocate
an unchecked replacement. Completion uses only this invocation's reserved range. */
(define tsql_native_insert_expr (lambda (relation identity columns values row_count capture_identity) (begin
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
	(if (nil? identity) (without_generated_identity (native_insert columns values callback))
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
						'('define '__tsql_ids '('reserve_ids relation row_count))
						'('if '('> '('cadr '__tsql_ids) maximum) '('error "IDENTITY exceeds the declared range") true)
						'('define '__tsql_inserted (native_insert (cons (identity "Field") columns) own_rows nil))
						/* Preserve completion only after successful native admission. */
						'('if '('> '__tsql_inserted 0) '('begin
							'('session "last_insert_id" '('car '__tsql_ids))
							(if capture_identity '('tsql_complete_identity 'session '('cadr '__tsql_ids)) true)) true)
						'__tsql_inserted))))))))
(define tsql_select_prefix (parser (? (atom "TOP" true)
	(define top_count (or (parser '("(" (define n tsql_int) ")") n) sql_int)))
	(if (and top_count (< top_count 0)) (error "TOP requires a nonnegative count") top_count)))
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
(define tsql_addition_type (lambda (type) (match type
	"INTEGER" "INT" "DOUBLE" "FLOAT" "NUMERIC" "DECIMAL"
	"BOOLEAN" "BIT" "BOOL" "BIT" _ type)))
(define tsql_compile_addition (lambda (infos) (begin
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
		(error "unsupported SQL operands for +") true)
	(define formulas (map infos (lambda (info) (begin
		(define type (tsql_addition_type (sql_info_type info)))
		(define formula (sql_info_formula info))
		(if (and numeric (or (has? tsql_addition_text_types type) (equal? type "BIT")))
			(sql_type_formula '() (tsql_cast_expr formula
				(if (has? '("DECIMAL" "MONEY" "SMALLMONEY") target) "FLOAT" target) '()))
			formula)))))
	(sql_info (cons (if numeric (quote +) (quote concat)) formulas) target
		(if numeric nil (reduce infos (lambda (collation info)
			(sql_merge_collation collation (sql_info_collation info))) nil))))))
(define tsql_add_expr (lambda (left right)
	(list (quote sql_compile_expr) (quote tsql_compile_addition) left right)))
(define tsql_raise_error (lambda (number message state)
	(error (list "tsql-error" number message state))))
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
(define tsql_fold_multiplicative (lambda (acc term) (match term
	'("divide" value) (tsql_divide_expr acc value)
	_ (sql_fold_multiplicative_term acc term))))
(define tsql_fold_additive (lambda (acc term) (match term
	'("add" value) (tsql_add_expr acc value)
	'(operation value) (if (has? '("and" "or" "xor" "left" "right") operation)
		(tsql_bit_expr operation (list acc value)) (sql_fold_additive_term acc term)))))
/* Implicit conversions consume declared contracts, including BIT's Boolean
carrier. Width/scale are not reconstructed from runtime values; unresolved
character dimensions retain the existing unbounded text conversion contract. */
(define tsql_convert_value_info (lambda (info type collation) (begin
	(define source (tsql_addition_type (sql_info_type info)))
	(define target (tsql_addition_type type))
	(define formula (sql_info_formula info))
	(define formula (if (and (equal? source "BIT") (not (equal? target "BIT")))
		(list (quote match) formula (quote nil) nil (quote true) 1 (quote false) 0) formula))
	/* Monetary source declarations round when converted to integers; other
	numerics truncate. MATCH preserves NULL and evaluates the source once. */
	(define formula (if (and (has? '("MONEY" "SMALLMONEY") source)
		(has? '("BIGINT" "INT" "SMALLINT" "TINYINT") target))
		(list (quote match) formula (quote nil) nil (quote value) (list (quote round) (quote value))) formula))
	(define typed (list (quote sql_type_expr) formula
		(if (equal? source "BIT") "INT" (sql_info_type info)) (sql_info_collation info)))
	(define converted (if (or (equal? source "NULL") (equal? source target)) formula
		(and (has? tsql_addition_text_types target) (has? tsql_addition_text_types source)) formula
		(and (has? tsql_addition_text_types target) (has? tsql_addition_numeric_types source))
		(list (quote tsql_cast_text) formula -1 (has? '("NVARCHAR" "NCHAR") target))
		(and (has? tsql_addition_numeric_types target)
			(or (has? tsql_addition_numeric_types source) (has? tsql_addition_text_types source)
				/* The existing integer cast selects byte decoding from the SQL
				declaration; the same payload declared as text remains decimal text. */
				(and (has? '("BINARY" "VARBINARY") source) (tsql_integer_width target))))
		(if (has? '("DECIMAL" "MONEY" "SMALLMONEY") target)
			(list (quote tsql_cast_float) formula)
			(sql_type_lower_annotations (tsql_cast_expr typed target '())))
		(tsql_temporal_type? target)
		(sql_type_lower_annotations (tsql_cast_expr typed target '()))
		(error "unsupported implicit conversion from " source " to " target)))
	(sql_info converted type collation))))
/* A common SQL contract for bounded syntax values. This is compile-time
metadata, shared by COALESCE and inline row constructors. It neither samples
runtime values nor stores a second relational representation. */
(define tsql_common_value_infos (lambda (infos null_type) (begin
	(define types (map infos (lambda (info) (tsql_addition_type (sql_info_type info)))))
	(define active (filter types (lambda (type) (not (equal? type "NULL")))))
	(define target (if (empty_list? active) (coalesceNil null_type (error "a declared non-NULL type is required"))
		(coalesceNil
			(find (merge (list tsql_addition_numeric_types tsql_addition_text_types))
				(lambda (type) (has? active type)) nil)
			(if (reduce active (lambda (same type) (and same (equal? type (car active)))) true)
				(car active) (error "unsupported common SQL type combination")))))
	(define collation (if (has? tsql_addition_text_types target)
		(reduce infos (lambda (collation info)
			(sql_merge_collation collation (sql_info_collation info))) nil) nil))
	(map infos (lambda (info) (tsql_convert_value_info info target collation))))))
/* UNION chooses column contracts from declarations, just like CASE and
inline values. Coercions become ordinary expressions before the shared planner;
no runtime value tag determines a SQL type. A nested set keeps its boundary. */
(define tsql_union_contracts (lambda (database branches) (begin
	(define recipes (map branches (lambda (branch)
		(if (query_block? branch) branch
			(make_query_block database
				(list (list "__tsql_set_input" database branch false nil))
				(merge (map (queryplan_result_titles branch) (lambda (title)
					(list title (list (quote get_column) "__tsql_set_input" false title false)))))
				true '() nil '() nil nil '() '() '())))))
	(define fields (map recipes (lambda (branch)
		(expand_query_block_fields (qb_sources branch) (qb_fields branch)))))
	(define width (intdiv (count (car fields)) 2))
	(if (reduce fields (lambda (valid field) (and valid (equal? (count field) (* width 2)))) true)
		true (error "UNION branch column count mismatch"))
	(define infos (map (produceN width) (lambda (column)
		(tsql_common_value_infos (mapIndex recipes (lambda (index branch)
			(sql_expr_info (qb_sources branch) (nth (nth fields index) (+ (* column 2) 1))))) "INT"))))
	(mapIndex recipes (lambda (index branch)
		(make_query_block (qb_schema branch) (qb_sources branch)
			(merge (map (produceN width) (lambda (column) (begin
				(define info (nth (nth infos column) index))
				(list (nth (nth fields index) (* column 2))
					(list (quote sql_type_expr) (sql_info_formula info)
						(sql_info_type info) (sql_info_collation info)))))))
			(qb_where branch) (qb_group branch) (qb_having branch) (qb_order branch)
			(qb_limit branch) (qb_offset branch) (qb_hidden branch) (qb_stages branch) (qb_facts branch)))))))

/* NULL fallback has the same declared precedence as other T-SQL expressions.
The shared lazy coalesce operator remains responsible for evaluation; this
callback consumes SQL triples and leaves only converted formulas in the plan.
Dimension merging and exact decimal precision are separate frontend work. */
(define tsql_compile_coalesce (lambda (infos) (begin
	(if (< (count infos) 2) (error "COALESCE requires at least two arguments") true)
	(define converted (tsql_common_value_infos infos nil))
	(sql_info (cons (quote coalesceNil) (map converted sql_info_formula))
		(sql_info_type (car converted)) (sql_info_collation (car converted))))))
(define tsql_coalesce_expr (lambda (arguments)
	(cons (quote sql_compile_expr) (cons (quote tsql_compile_coalesce) arguments))))

/* ISNULL keeps its first declared contract, unlike COALESCE precedence.
A literal NULL takes the replacement contract; two literal NULLs use INT.
Only replacement evaluation is lazy, while compatibility is checked for every
row shape at compilation. Width/scale propagation remains separate work; an
unknown text width must not silently become CAST's default length of thirty. */
(define tsql_compile_isnull (lambda (infos) (begin
	(if (equal? (count infos) 2) true (error "ISNULL requires two arguments"))
	(define first (car infos))
	(define replacement (cadr infos))
	(define contract (if (equal? (sql_info_type first) "NULL") replacement first))
	(define type (if (equal? (sql_info_type contract) "NULL") "INT" (sql_info_type contract)))
	(define converted (tsql_convert_value_info replacement type (sql_info_collation contract)))
	(sql_info (list (quote coalesceNil) (sql_info_formula first) (sql_info_formula converted))
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
		(sql_info_type first) (sql_info_collation first)))))
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
		(sql_info_type (car values)) (sql_info_collation (car values)))))))
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
	(define ordinal (sql_type_lower_annotations
		(tsql_cast_expr (if (has? '("MONEY" "SMALLMONEY") index_type)
			(list (quote round) (sql_info_formula index_info)) (sql_info_formula index_info)) "INT" '())))
	(sql_info (cons (quote match) (cons ordinal (merge (list
		(merge (map (produceN (count branches)) (lambda (i)
			(list (+ i 1) (sql_info_formula (nth branches i))))))
		(list (quote _) nil)))))
		(sql_info_type (car branches)) (sql_info_collation (car branches))))))
(define tsql_choose_expr (lambda (index values)
	(cons (quote sql_compile_expr) (cons (quote tsql_compile_choose)
		(cons index values)))))

/* Bind the shared logical tree for metadata only. No physical plan, compiled
row evaluator or native mutation is constructed by this analysis path. The
caller's policy remains explicit through every nested view expansion. */
(define tsql_describe_sources (lambda (node) (match node
	((symbol quote) _datum) node
	(cons (symbol query-block) _tail) (begin
		(map (qb_sources node) (lambda (source)
			(if (and (source_is_base_table? source)
				(nil? (table (source_schema source) false (source_relation source) false)))
				(error "description table does not exist: " (source_relation source)) true)))
		(map (cdr node) tsql_describe_sources)
		node)
	(cons _head _tail) (begin (map node tsql_describe_sources) node)
	_ node)))
(define tsql_describe_query (lambda (query policy planning_session)
	(bind_query_names (tsql_describe_sources (tsql_bind_column_compilers
		(sql_expand_views query policy) (tsql_column_compiler planning_session))) '())))
/* Explicit wire descriptors come from the existing SQL expression metadata,
never from the first row. Unknown expression types stay text in this first slice. */
/* Width/scale originate in declaration syntax, never in the first output row.
The shared expression compiler still owns the SQL type; these additional wire
dimensions are metadata only and are absent from the executable plan. */
(define tsql_result_dimensions (lambda (sources expression parameters) (match expression
	((symbol sql_compile_column) _compiler column) (tsql_result_dimensions sources column parameters)
	((symbol sql_compile_expr) ((symbol tsql_text_compiler) length _national _fixed) _value) (list length)
	((symbol sql_type_expr) formula _type _collation) (tsql_result_dimensions sources formula parameters)
	((symbol sql_compile_expr) ((symbol tsql_temporal_compiler) _target _style ((symbol quote) dimensions)) _value) dimensions
	((symbol sql_compile_expr) ((symbol tsql_dateadd_compiler) _unit _months _seconds) _amount value)
	(tsql_result_dimensions sources value parameters)
	((symbol sql_compile_expr) ((symbol tsql_text_compiler) length _national _fixed) _value) (list length)
	(cons (symbol sql_compile_expr) (cons ((symbol tsql_builtin_compiler) name) (cons value _arguments)))
	(if (has? '("UPPER" "LOWER" "LTRIM" "RTRIM" "TRIM" "LEFT" "RIGHT" "SUBSTRING") name)
		(tsql_result_dimensions sources value parameters) nil)
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
	(cons (symbol sql_compile_expr) _arguments)
	(tsql_result_dimensions sources (sql_type_formula sources expression) parameters)
	((symbol tsql_cast_decimal) _value precision scale) (list precision scale)
	((symbol tsql_quote_name) _value _delimiter) '(258)
	((symbol tsql_space) _amount) '(8000)
	((symbol tsql_read_identity) _state) '(38 0)
	((symbol tsql_cast_binary) _value length _fixed) (list length)
	((symbol tsql_integer_to_binary) _value _width length _fixed) (list length)
	((symbol tsql_cast_text) _value length _national) (list length)
	((symbol tsql_cast_fixed_text) _value length _national) (list length)
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
						(define handle (table (source_schema src) false (source_relation src) false))
						(define metadata (if (nil? handle) nil (show handle "column" bound_column false)))
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
	_ nil)))
/* Provenance is a declaration fact. Computed values never inherit a base
column's editability, and an outer source remains nullable after binding. */
(define tsql_result_origin (lambda (sources info follow_derived) (match (sql_info_formula info)
	((symbol get_column) alias alias_ic name name_ic) (begin
		(define source (source_for_alias sources (if (empty_list? sources) nil (source_alias (car sources))) alias alias_ic))
		(if (nil? source) nil
			(source_is_base_table? source) (begin
				(define handle (table (source_schema source) false (source_relation source) false))
				(define column (if handle (show handle "column" name name_ic) nil))
				(if column (list source column) nil))
			follow_derived (match (source_relation source)
				((symbol query-block) _schema inner_sources fields _where _group _having _order _limit _offset _hidden _stages _facts)
				(begin
					(define expression ((expand_query_block_fields inner_sources fields) name))
					(define origin (tsql_result_origin inner_sources (sql_expr_info inner_sources expression) true))
					(if (and origin (source_outer? source))
						(list (car origin) (set_assoc (cadr origin) "Null" true)) origin))
				_ nil)
			nil))
	_ nil)))
/* Both client adapters consume the same expression triples. Following a view's
native declaration is permitted only for its column catalog, never to infer
editability or keys for a derived result. */
(define tsql_result_entries (lambda (query follow_derived) (match query
	((symbol query-block) _schema sources fields _where _group _having _order _limit _offset _hidden _stages _facts)
	(extract_assoc (expand_query_block_fields sources fields) (lambda (name expression) (begin
		(define info (sql_expr_info sources expression))
		(list name expression info (tsql_result_origin sources info follow_derived)))))
	_ (error "unsupported result projection contract"))))
(define tsql_result_flags (lambda (origin editable) (if (nil? origin) '(1 nil) (begin
	(define source (car origin))
	(define column (cadr origin))
	(define identity (equal? (column "Extra") "auto_increment"))
	(define version (equal?? (column "RawType") "ROWVERSION"))
	(list (+ (if (or (source_outer? source) (column "Null")) 1 0)
		(if (and editable (not (or identity version))) 4 0) (if identity 16 0))
		(if version 80 0))))))
/* Wire shapes come only from a declared SQL contract, including when no
row or supplied parameter value exists. */
(define tsql_wire_contract (lambda (type dimensions) (begin
	(define numeric (and (has? '("DECIMAL" "NUMERIC") type) (>= (count dimensions) 2)))
	(define money (has? '("MONEY" "SMALLMONEY") type))
	(define kind (if (has? '("INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT") type) 38
		money 110 numeric 108 (tsql_binary_type? type) 165
		(has? '("FLOAT" "DOUBLE" "REAL" "DECIMAL" "NUMERIC") type) 109
		(has? '("BOOLEAN" "BOOL" "BIT") type) 104
		(equal? type "DATE") 40 (equal? type "TIME") 41 (equal? type "DATETIME2") 42
		(has? '("DATETIME" "SMALLDATETIME") type) 111 231))
	(define precision (if money (if (equal? type "SMALLMONEY") 10 19) numeric (car dimensions) 0))
	(define scale (if money 4 numeric (cadr dimensions)
		(has? '("TIME" "DATETIME2") type) (if (empty_list? dimensions) 7 (car dimensions)) 0))
	(define size (match kind
		40 3
		41 (if (<= scale 2) 3 (<= scale 4) 4 5)
		42 (+ 3 (if (<= scale 2) 3 (<= scale 4) 4 5))
		111 (if (equal? type "SMALLDATETIME") 4 8)
		110 (if (equal? type "SMALLMONEY") 4 8)
		108 (if (<= precision 9) 5 (<= precision 19) 9 (<= precision 28) 13 17)
		165 (if (equal? type "ROWVERSION") 8
			(and (not (empty_list? dimensions)) (> (car dimensions) 0)) (car dimensions) 65535)
		231 (if (and (has? '("NVARCHAR" "NCHAR") type) (not (empty_list? dimensions)))
			(if (equal? (car dimensions) -1) 65535 (* 2 (car dimensions))) 8000)
		104 1
		_ (match type "TINYINT" 1 "SMALLINT" 2 "INT" 4 "INTEGER" 4 _ 8)))
	(list "kind" kind "size" size "precision" precision "scale" scale))))
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
			(define dimensions (coalesceNil (tsql_result_dimensions sources expression parameters)
				(tsql_result_dimensions sources (sql_info_formula info) parameters) '()))
			(define wire (tsql_wire_contract type dimensions))
			(merge wire (list "name" name "flags" (car flags) "user_type" (coalesceNil (cadr flags) 0) "sql_type" type)
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
		(or found (tsql_temporal_type? (column "sql_type")))) false)
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
									(list tsql_calendar_text value) value))
								(list (column "name") (if pattern (list format_date calendar_value pattern) value)))))))))))
			plan)
		plan)))

(define tsql_collation_compiler (lambda (name) (lambda (infos) (begin
	(define info (car infos))
	(sql_info (sql_info_formula info) (sql_info_type info) (list name 0))))))

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
(define tsql_round_real_factor (lambda (places)
	(if (or (nil? places) (< places -308) (> places 323)) nil
		(nth tsql_round_real_factors (+ places 308)))))
(define tsql_round_real (lambda (value places mode factor)
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
				(tsql_raise_error 8115 "floating rounding overflow" 1))))))
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
			(sql_info (if money (list (quote tsql_cast_money) formula false) formula)
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
(define tsql_trim (lambda (value) (tsql_rtrim (tsql_ltrim value))))
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
		output (if binary nil (sql_info_collation converted))))))

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
		"UPPER" (quote tsql_upper) "LOWER" (quote tsql_lower)
		"LTRIM" (quote tsql_ltrim) "RTRIM" (quote tsql_rtrim) "TRIM" (quote tsql_trim)))
	(sql_info (list helper (sql_info_formula converted)) output (sql_info_collation input)))))

(define tsql_builtin_compiler (lambda (name) (lambda (infos)
	(if (equal? name "ROUND") (tsql_compile_round infos)
		(has? '("ABS" "FLOOR" "CEILING") name) (tsql_compile_numeric_scalar name infos)
		(has? '("SQUARE" "SQRT") name) (tsql_compile_floating_scalar name infos)
		(has? '("LEFT" "RIGHT" "SUBSTRING") name) (tsql_compile_slicing name infos)
		(has? '("UPPER" "LOWER" "LTRIM" "RTRIM" "TRIM") name) (tsql_compile_string_normalization name infos)
		(sql_call_info (coalesce (tsql_builtins name) (error "unknown function " name)) infos)))))

(define tsql_builtin_expr (lambda (name arguments) (begin
	(if (equal? name "PARSENAME")
		(cons (quote sql_compile_expr) (cons (quote tsql_compile_parsename) arguments))
		(begin
			(if (tsql_builtins name) true (error "unknown function " name))
			(cons (quote sql_compile_expr) (cons (list (quote tsql_builtin_compiler) name) arguments)))))))

(define tsql_builtins (lambda (name) (match name
	/* First-class callers retain lazy fallback; SQL calls use the declared
	ISNULL compiler contract rather than this scalar operator alias. */
	"ISNULL" coalesceNil
	"LEFT" tsql_left
	"RIGHT" tsql_right
	"SUBSTRING" (lambda (value start amount) (tsql_substring value start amount false))
	"LEN" (lambda (value) (if (nil? value) nil (strlen (sql_rtrim value))))
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
		(regex "(?:[^*/]+|/[^*]|\\*[^/])" false false)) empty)
	"*/") nil "^$"))
(define tsql_script_tokens (parser (define tokens (* (or
	(parser (define text (capture tsql_script_comment)) (list "comment" (car text)))
	(parser (define text (regex "(?:--[^\\r\\n]*)" false false)) (list "comment" text))
	(parser (define text (regex "(?:'(?:''|[^'])*'|\\\"(?:\\\"\\\"|[^\\\"])*\\\"|\\[(?:\\]\\]|[^\\]])*\\])" false false)) (list "quoted" text))
	(parser (define text (regex "(?:\\r\\n|\\r|\\n)" false false)) (list "newline" text))
	(parser (define text (regex "(?:[^'\\\"\\[\\r\\n/-]+|/[^*]|-[^-]|[/-]$)" false false)) (list "text" text))
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

/* Metadata scalars require the current consuming policy. Direct queries pass
that context explicitly; stored views refuse this scalar before publication. */
(define tsql_column_length (lambda (database policy relation column)
	(if (or (nil? relation) (nil? column)) nil (begin
		(define object ((parser '( (? (define owner tsql_identifier) ".")
			(define name tsql_identifier)) (list (if (or (nil? owner) (equal?? owner "dbo")) database owner) name)) relation))
		(define object_schema (car object))
		(define name (coalesceNil (show (table object_schema false (cadr object) true) "name") (cadr object)))
		(if policy (policy object_schema name false) true)
		(define handle (table object_schema false name false))
		(define declaration (if (nil? handle) nil (tsql_effective_column (show handle "column" column true) nil)))
		(if (nil? declaration) nil (tsql_declared_byte_length declaration))))))

(define tsql_view_has_metadata_scalar? (lambda (node) (match node
	((symbol quote) _datum) false
	(cons (symbol tsql_column_length) _arguments) true
	(cons _head _tail) (not (nil? (find node tsql_view_has_metadata_scalar? nil)))
	_ false)))

(define parse_tsql (lambda (database s policy planning_session tx) (begin
	(define schema database)
	(define parse_started_ns (nanotime))
	(define frontend_options (if (nil? planning_session) (newsession) planning_session))
	/* Import bindings are compile-time name contracts. Keep the authenticated
	policy and confine every published dependency, including expanded views. */
	(define account_policy policy)
	(define import_target (frontend_options "tsql_import_target"))
	(define policy (if (nil? import_target) account_policy (lambda (database name write) (begin
		(if (and (equal? database import_target)
			(not (and (equal? database "system") (expression_equal? name true) write))) true
			(error "import reference outside destination and source alias: " database))
		(if account_policy (account_policy database name write) true)))))
	(define describe (frontend_options "__tsql_describe"))
	(define script_syntax (frontend_options "__tsql_script_syntax"))
	(define expression_only (frontend_options "__tsql_expression_only"))
	(define analyze_parameters (frontend_options "__tsql_parameter_analysis"))
	(define parameters (coalesceNil (frontend_options "__tsql_parameter_declarations") (list)))
	(define parameter_expression (lambda (name spelling) (begin
		(define name (toLower name))
		(define declaration (filter parameters (lambda (item) (equal? (car item) name))))
		(if script_syntax (list (quote tsql_batch_variable) name)
			(and analyze_parameters (empty_list? declaration))
			(list (quote sql_type_expr) (list (quote tsql_unbound_parameter) name spelling) "any" nil)
			(empty_list? declaration) (error "undeclared T-SQL parameter: @" name)
			(list (quote sql_type_expr) (list (quote session) (concat "__tsql_parameter:" name))
				(car (tsql_effective_parameter_contract (cadr (car declaration)))) nil)))))

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

		(parser '((define a tsql_expression3) (atom "COLLATE" true) (define collation tsql_identifier) "=" (define b tsql_expression2)) '('equal_collate a b collation))
		(parser '((define a tsql_expression3) (atom "COLLATE" true) (define collation tsql_identifier) "<>" (define b tsql_expression2)) '('notequal_collate a b collation))
		(parser '((define a tsql_expression3) (atom "COLLATE" true) (define collation tsql_identifier) "!=" (define b tsql_expression2)) '('notequal_collate a b collation))
		(parser '((define a tsql_expression3) (define op (or
			(parser "=" "eq")
			(parser "<>" "ne")
			(parser "!=" "ne")
			(parser "<=" "le")
			(parser ">=" "ge")
			(parser "<" "lt")
			(parser ">" "gt")
		)) (define b tsql_expression2)) (tsql_comparison_expr op a b))
		(parser '((define a tsql_expression3) (atom "COLLATE" true) (define collation tsql_identifier) (atom "LIKE" true) (define b tsql_expression2)) '('strlike a b collation))

		(parser '((define a tsql_expression3) (atom "LIKE" true) (define b tsql_expression2)) '('strlike a b "utf8mb4_general_ci"))
		(parser '((define a tsql_expression3) (atom "NOT" true) (atom "LIKE" true) (define b tsql_expression2)) '('sql_not '('strlike a b "utf8mb4_general_ci")))

		(parser '((define a tsql_expression3) (atom "IN" true) (define b tsql_in_values)) (tsql_predicate_expr '('sql_in b a)))
		(parser '((define a tsql_expression3) (atom "NOT" true) (atom "IN" true) (define b tsql_in_values)) (list (quote sql_not) (tsql_predicate_expr (list (quote sql_in) b a))))

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
	(define tsql_expression5 (parser (or
		(parser '((define expr tsql_unary_expression) (atom "IS" true) (atom "NULL" true)) '('nil? expr))
		(parser '((define expr tsql_unary_expression) (atom "IS" true) (atom "NOT" true) (atom "NULL" true)) '('not '('nil? expr)))
		tsql_unary_expression)))

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
			(list (quote tsql_column_length) database policy relation column) "INT" nil)))

	(define tsql_expression6 (parser (or

		(parser '("(" (define sub tsql_select) ")") '('inner_select sub))
		(parser '("(" (define a tsql_expression) ")") a)
		(parser '((define operation (or (parser (atom "LEFT_SHIFT" true) "left")
			(parser (atom "RIGHT_SHIFT" true) "right"))) "(" (define value tsql_expression)
			"," (define amount tsql_expression) ")") (tsql_bit_expr operation (list value amount)))
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
		(parser '((atom "DATEADD" true) "(" (define unit tsql_identifier) ","
			(define amount tsql_expression) "," (define value tsql_expression) ")")
			(tsql_dateadd_expr unit amount value))
		(parser '((define type (or (parser (atom "DATEDIFF" true) "INT") (parser (atom "DATEDIFF_BIG" true) "BIGINT")))
			"(" (define unit tsql_identifier) ","
			(define first tsql_expression) "," (define last tsql_expression) ")")
			(tsql_datediff_expr unit first last type))
		(parser '((atom "DATEPART" true) "(" (define unit tsql_identifier) "," (define value tsql_expression) ")")
			(tsql_datepart_expr unit value))
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
			(list (tsql_comparison_expr "eq" expr (car condition)) (cadr condition)))) elsebranch))

		(parser '((atom "IIF" true) "(" (define condition tsql_expression) ","
			(define yes tsql_expression) "," (define no tsql_expression) ")")
			(tsql_case_expr (list (list condition yes)) no))
		(parser '((atom "CHOOSE" true) "(" (define index tsql_expression) ","
			(define values (+ tsql_expression ",")) ")") (tsql_choose_expr index values))

		(parser '((define count_type tsql_count_type) "(" "*" ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") (list (quote sql_type_expr) (tsql_window_expr "COUNT" (list) _over) count_type nil))
		(parser '((define count_type tsql_count_type) "(" (? (atom "ALL" true)) (define e tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") (list (quote sql_type_expr) (tsql_window_expr "COUNT" (list e) _over) count_type nil))
		(parser '((atom "SUM" true) "(" (define s tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") (tsql_window_expr "SUM" (list s) _over))
		(parser '((atom "AVG" true) "(" (define s tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") (tsql_window_expr "AVG" (list s) _over))
		(parser '((atom "MIN" true) "(" (define s tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") (tsql_window_expr "MIN" (list s) _over))
		(parser '((atom "MAX" true) "(" (define s tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") (tsql_window_expr "MAX" (list s) _over))

		(parser '((define count_type tsql_count_type) "(" (atom "DISTINCT" true) (define e tsql_expression) ")")
			'('sql_type_expr '('count_distinct e) count_type nil))
		(parser '((define count_type tsql_count_type) "(" "*" ")")
			'('sql_type_expr '((quote aggregate) 1 (quote +) 0) count_type nil))
		(parser '((define count_type tsql_count_type) "(" (? (atom "ALL" true)) (define e tsql_expression) ")")
			'('sql_type_expr '('aggregate '((quote if) '((quote nil?) e) 0 1) (quote +) 0) count_type nil))
		(parser '((atom "SUM" true) "(" (define s tsql_expression) ")") (begin (define d (sql_aggregates "SUM")) (tsql_aggregate_expr "SUM" '('aggregate s (car d) (cadr d)) s)))
		(parser '((atom "AVG" true) "(" (define s tsql_expression) ")")
			(tsql_aggregate_expr "AVG" (sql_avg_expr s (sql_aggregates "SUM") (sql_aggregates "COUNT")) s))
		(parser '((atom "MIN" true) "(" (define s tsql_expression) ")") (begin (define d (sql_aggregates "MIN")) (tsql_aggregate_expr "MIN" '('aggregate s (car d) (cadr d)) s)))
		(parser '((atom "MAX" true) "(" (define s tsql_expression) ")") (begin (define d (sql_aggregates "MAX")) (tsql_aggregate_expr "MAX" '('aggregate s (car d) (cadr d)) s)))

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
		(parser '((atom "DB_NAME" true) "(" ")")
			(list (quote sql_type_expr) (list (quote session) "schema") "NVARCHAR" '("utf8mb4_general_ci" 4)))
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

	(define tabledefs (parser (or

		(parser '((define l tabledefs) (define x (or
			(parser '((atom "CROSS" true) (atom "APPLY" true) (define r tabledef)) (list (list (source_alias r) (source_schema r)
				(list (quote tsql_pending_apply) (source_relation r)) false nil)))
			(parser '((atom "OUTER" true) (atom "APPLY" true) (define r tabledef)) (list (list (source_alias r) (source_schema r)
				(list (quote tsql_pending_apply) (source_relation r)) true nil)))
			(parser '((atom "LEFT" true) (? (atom "OUTER" true)) (atom "JOIN" true) (define r tabledef) (atom "ON" true) (define e tsql_expression)) (match r '(id schema tbl _ nil) '('(id schema tbl true e))))
			(parser '((? (atom "INNER" true)) (atom "JOIN" true) (define r tabledef) (atom "ON" true) (define e tsql_expression)) (match r '(id schema tbl _ nil) '('(id schema tbl false e))))
			(parser '((? (atom "CROSS" true)) (atom "JOIN" true) (define r tabledefs)) r)
		))) (merge l x))

		(parser '((define l tabledef) (atom "RIGHT" true) (? (atom "OUTER" true)) (atom "JOIN" true) (define r tabledefs) (atom "ON" true) (define e tsql_expression))
			(match l '(id schema tbl _ nil)
				(merge r '('(id schema tbl true e)))))
		(parser (define t tabledef) '(t))
	)))

	/* Lateral visibility is a logical source fact. The common binder and
	decorrelator own dependency elimination; APPLY emits no execution code. */
	(define tsql_apply_source (lambda (src outer) (match src
		'(id database relation _outer _join) (list id database
			(match relation
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
			outer nil))))

	(define tsql_read_source (lambda (alias database name) (begin
		(define canonical (tsql_relation_name database name))
		(planner_record_guard_condition
			(list (quote equal?) (list (quote tsql_relation_name) database name) canonical)
			planning_session)
		(if policy (policy database canonical false) true)
		(list alias database canonical false nil))))
	(define tabledef (parser (or

		/* VALUES is syntax data, lowered to the same logical SELECT/UNION
		blocks as other inline relations. The common planner owns joins,
		grouping, ordering, limits and physical execution. */
		(parser '("(" (atom "VALUES" true)
			(define rows (+ (parser '("(" (define row (+ tsql_expression ",")) ")") row) ","))
			")" (? (atom "AS" true)) (define id tsql_identifier)
			"(" (define aliases (+ tsql_identifier ",")) ")")
			(list id schema (list (quote tsql_pending_values) aliases rows) false nil))

		(parser '((atom "(" true) (define query tsql_select) (atom ")" true) (atom "AS" true) (define id tsql_identifier) "(" (define aliases (+ tsql_identifier ",")) ")")
			(list id schema (list (quote tsql_pending_aliases) query aliases) false nil))
		(parser '((atom "(" true) (define query tsql_select) (atom ")" true) (define id tsql_identifier) "(" (define aliases (+ tsql_identifier ",")) ")")
			(list id schema (list (quote tsql_pending_aliases) query aliases) false nil))
		(parser '((atom "(" true) (define query tsql_select) (atom ")" true) (atom "AS" true) (define id tsql_identifier)) '(id schema query false nil))
		(parser '((atom "(" true) (define query tsql_select) (atom ")" true) (define id tsql_identifier)) '(id schema query false nil))

		(parser '((define owner tsql_schema_identifier) (define tbl tsql_identifier) (atom "AS" true) (define id tsql_identifier) (? tsql_table_hints)) (tsql_read_source id (tsql_qualifier_database owner database frontend_options) tbl))
		(parser '((define owner tsql_schema_identifier) (define tbl tsql_identifier) (define id tsql_identifier) (? tsql_table_hints)) (tsql_read_source id (tsql_qualifier_database owner database frontend_options) tbl))
		(parser '((define owner tsql_schema_identifier) (define tbl tsql_identifier) (? tsql_table_hints)) (tsql_read_source tbl (tsql_qualifier_database owner database frontend_options) tbl))
		(parser '((define tbl tsql_identifier) (atom "AS" true) (define id tsql_identifier) (? tsql_table_hints)) (list id schema (list (quote tsql_source_name) tbl) false nil))
		(parser '((define tbl tsql_identifier) (define id tsql_identifier) (? tsql_table_hints)) (list id schema (list (quote tsql_source_name) tbl) false nil))
		(parser '((define tbl tsql_identifier) (? tsql_table_hints)) (list tbl schema (list (quote tsql_source_name) tbl) false nil))
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
		(list (list "__tsql_query_source" schema query false nil))
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
					(sql_info_type info) (sql_info_collation info)))))))))
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

	/* Parser-only source recipes are bound once at the statement's compile
	boundary. Local definitions become ordinary derived logical queries; native
	names retain the existing catalog, policy and schema-guard path. Nothing here
	is retained by a row callback or published as a separate relation catalog. */
	(define tsql_finalize_sources (lambda (node kind bindings blocked) (begin
		(define definition_scope (or (equal? kind "definition") (equal? kind "definition_source")))
		(define child_kind (if definition_scope "definition" "query"))
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
						(define exported (tsql_finalize_sources bound "exports" available names))
						(define titles (queryplan_result_titles exported))
						(define output_names (coalesceNil aliases titles))
						(if (equal? (count output_names) (count titles)) true
							(error "CTE output alias count does not match query column count"))
						(if (equal? (count output_names) (count (merge_unique (list (map output_names toLower))))) true
							(error "duplicate CTE output column name"))
						(cons (list name (if (nil? aliases) exported
							(tsql_apply_derived_column_aliases exported aliases))) available)))) bindings))
			"source" (match node '(id database relation outer join) (begin
				(define resolved (match relation
					((symbol tsql_source_name) name) (begin
						(define local (find bindings (lambda (entry) (equal?? (car entry) name)) nil))
						(if local (list id database (cadr local) outer nil)
							(find blocked (lambda (entry) (equal?? entry name)) nil)
							(error "recursive or forward CTE reference is unsupported: " name)
							(begin
								(define native (tsql_read_source id database name))
								(list id database (source_relation native) outer nil))))
					((symbol tsql_pending_apply) inner)
					(tsql_apply_source (tsql_finalize_sources (list id database inner outer nil)
						source_kind bindings blocked) outer)
					_ (list id database (tsql_finalize_sources relation child_kind bindings blocked) outer nil)))
				(list (source_alias resolved) (source_schema resolved) (source_relation resolved)
					(source_outer? resolved) (tsql_finalize_sources join child_kind bindings blocked))))
			_ (match node
				((symbol quote) _literal) node
				((symbol tsql_pending_with) definitions body) (begin
					(define available (tsql_finalize_sources definitions "definitions" bindings blocked))
					(tsql_finalize_sources body "query" available '()))
				((symbol tsql_pending_aliases) body aliases)
				(tsql_apply_derived_column_aliases
					(tsql_finalize_sources (tsql_finalize_sources body child_kind bindings blocked)
						"exports" bindings blocked) aliases)
				((symbol tsql_pending_values) aliases rows)
				(tsql_inline_values_query aliases (tsql_finalize_sources rows child_kind bindings blocked))
				((symbol query-block) db sources fields condition group having order limit offset hidden stages facts)
				(begin
					(if definition_scope (begin
						(if (and (not (empty_list? (coalesceNil order '()))) (nil? limit) (nil? offset))
							(error "CTE ORDER BY requires TOP or OFFSET") true)
						(if (qassoc_get facts (quote tsql_parser_query_hint) false)
							(error "OPTION is not allowed in a CTE definition") true)) true)
					(define bound_sources (if (equal? kind "exports") sources
						(map sources (lambda (source) (tsql_finalize_sources source source_kind bindings blocked)))))
					(make_query_block db bound_sources
						(if (equal? kind "exports") (expand_query_block_fields bound_sources fields)
							(tsql_finalize_sources fields child_kind bindings blocked))
						(tsql_finalize_sources condition child_kind bindings blocked)
						(tsql_finalize_sources group child_kind bindings blocked)
						(tsql_finalize_sources having child_kind bindings blocked)
						(tsql_finalize_sources order child_kind bindings blocked)
						limit offset hidden stages
						(filter facts (lambda (fact) (and
							(not (equal? (car fact) (quote tsql_parser_query_hint)))
							(not (equal? (car fact) (quote tsql_parser_unnamed_output))))))))
				((symbol union-block) mode branches order limit offset facts) (begin
					(if (and definition_scope (not (empty_list? (coalesceNil order '())))
						(nil? limit) (nil? offset)) (error "CTE ORDER BY requires TOP or OFFSET") true)
					(list (quote union-block) mode
						(tsql_union_contracts schema (map branches (lambda (branch) (tsql_finalize_sources branch kind bindings blocked))))
						(tsql_finalize_sources order child_kind bindings blocked) limit offset
						(filter facts (lambda (fact) (and
							(not (equal? (car fact) (quote tsql_parser_query_hint)))
							(not (equal? (car fact) (quote tsql_parser_unnamed_output))))))))
				(cons _head _tail) (map node (lambda (item) (tsql_finalize_sources item child_kind bindings blocked)))
				_ node)))))

	(define tsql_select_core (parser '(
		(atom "SELECT" true)

		(define distinct (or (parser (atom "DISTINCT" true) true)
			(parser (? (atom "ALL" true)) false)))
		(define prefix_limit tsql_select_prefix)
		(define cols (+ (or
			/* Assignment aliases must precede expression comparisons. */
			(parser '((define title (or tsql_identifier tsql_string)) "=" (define e tsql_expression)) '(title e))
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
		(?
			(atom "FROM" true)
			(define from (+ tabledefs ","))
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

	) (begin
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
				nil prefix_limit nil '() '()
				(merge (list
					/* The common name binder resolves all joined/derived sources.
					HAVING names belong to the input scope in this frontend. */
					(list (list (quote having_input_scope) true))
					(if distinct (list (list (quote select_distinct) true)) '())
					(if unnamed_output (list (list (quote tsql_parser_unnamed_output) true)) '())))))))

	/* Parentheses delimit query expressions. Keep this parser-only bit until
	the trailing window has been attached; it never enters logical planning. */
	(define tsql_select_branch (parser (or
		(parser (define query tsql_select_core) (list query false))
		(parser '("(" (define query tsql_select) ")") (list query true)))))
	(define tsql_select_window (lambda (query order suffix_stage parenthesized) (begin
		(if (and suffix_stage (nil? order)) (error "OFFSET requires ORDER BY") true)
		(if (and (nil? order) (nil? suffix_stage)) query (begin
			/* An outer window over a bracketed query is an ordinary derived source.
			Its inner TOP/order remains a semantic boundary for the shared planner. */
			(define query (if parenthesized (tsql_query_from_source query) query))
			(match query
				((symbol query-block) database sources fields condition group having previous_order limit offset hidden stages facts)
				(begin
					(if (and suffix_stage (not (nil? limit))) (error "TOP cannot be combined with OFFSET") true)
					(make_query_block database sources fields condition group having order
						(if suffix_stage (car suffix_stage) limit)
						(if suffix_stage (cadr suffix_stage) offset) hidden stages facts))
				((symbol union-block) mode branches previous_order limit offset facts)
				(list (quote union-block) mode branches order
					(if suffix_stage (car suffix_stage) limit)
					(if suffix_stage (cadr suffix_stage) offset) facts)))))))
	(define tsql_select (parser '(
		(define left tsql_select_branch)
		(define terms (* (parser '(
			(atom "UNION" true)
			(define mode (or (parser (atom "ALL" true) (quote all)) (parser empty (quote distinct))))
			(define right tsql_select_branch)
		) (list mode (car right))) empty true))

		(?
			(atom "ORDER" true)
			(atom "BY" true)
			(define order (+
				(parser '(
					(define col tsql_expression)
					(? (atom "COLLATE" true) (define coll tsql_identifier))
					(define direction_desc (or
						(parser (atom "DESC" true) >)
						(parser (atom "ASC" true) <)
						(parser empty <)
					))
				) (list (if coll
						(list (quote sql_compile_expr) (list (quote tsql_collation_compiler) coll) col) col) direction_desc))

				(atom "," true)
			))
		)

		(define suffix_stage tsql_select_suffix)
		(? (define query_hint tsql_query_hints))

	) (begin
			(define query (tsql_select_window
				(reduce terms (lambda (query term) (tsql_union_query (car term) query (cadr term))) (car left))
				order suffix_stage (and (empty_list? terms) (cadr left))))
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
	(define tsql_write_target (parser '(
		(? (define target_schema tsql_schema_identifier))
		(define target_name tsql_identifier)
		(? tsql_table_hints)
	) (list (if (nil? target_schema) nil (tsql_qualifier_database target_schema schema frontend_options)) target_name)))
	(define tsql_write_source (lambda (target sources) (match target
		'(target_schema target_name) (begin
			(define alias_match (if target_schema nil (find sources (lambda (src)
				(equal?? (source_alias src) target_name)) nil)))
			(define table_matches (filter sources (lambda (src) (and
				(source_is_base_table? src)
				(equal? (source_schema src) (tsql_qualifier_database target_schema schema frontend_options))
				(equal?? (source_relation src) target_name)))))
			(define table_match (if (empty_list? table_matches) nil
				(equal? (count table_matches) 1) (car table_matches)
				(find table_matches (lambda (src) (equal?? (source_alias src) target_name)) nil)))
			(if (and (nil? alias_match) (> (count table_matches) 1) (nil? table_match))
				(error "ambiguous write target: " target_name) true)
			(define target_database (tsql_qualifier_database target_schema schema frontend_options))
			(define source (coalesceNil alias_match table_match
				(list target_name target_database
					(tsql_table_name target_database target_name planning_session) false nil)))
			(if (source_is_base_table? source) source (error "write target must be a base table"))))))
	(define tsql_write_sources (lambda (target sources)
		(if (has? sources target) sources (cons target sources))))
	(define tsql_write_limit (parser (or
		(parser '((atom "TOP" true) "(" (define n tsql_int) ")")
			(if (< n 0) (error "DML TOP must be nonnegative") n))
		(parser empty nil))))
	(define tsql_update_assignment (parser '(
		(? (define qualifier tsql_identifier) ".")
		(define title tsql_identifier)
		(define op (or (parser "+=" "add") (parser "-=" "subtract")
			(parser "*=" "multiply") (parser "/=" "divide") (parser "%=" "modulo")
			(parser "&=" "and") (parser "|=" "or") (parser "^=" "xor")
			(parser "=" "assign")))
		(define expression (or (parser (atom "DEFAULT" true) (quote sql_default)) tsql_expression))
	) (list title qualifier op expression)))
	(define tsql_update_values (lambda (target cols) (begin
		(define names (map cols (lambda (col) (toLower (car col)))))
		(if (equal? (count names) (count (merge_unique (list names)))) true
			(error "duplicate UPDATE assignment"))
		(merge (map cols (lambda (col) (match col
			'(title qualifier op expression) (begin
				(if (or (nil? qualifier) (equal?? qualifier (source_alias target))) true
					(error "assignment qualifier must name the write target"))
				(define declaration (tsql_target_column_declaration (source_schema target) (source_relation target) title planning_session))
				(if (nil? declaration) (error "unknown UPDATE column: " title)
					(equal? (declaration "Extra") "auto_increment") (error "IDENTITY column cannot be updated") true)
				(define value (if (equal?? expression (quote sql_default))
					(if (equal? op "assign")
						(tsql_default_assignment_expr declaration)
						(error "DEFAULT requires simple assignment")) expression))
				(define old (list (quote sql_compile_column) (quote tsql_compile_column)
					(list (quote get_column) (source_alias target) false (declaration "Field") false)))
				(if (has? '("and" "or" "xor") op) (tsql_bit_assignment_type (declaration "Type")) true)
				(define assigned (match op "assign" value
					"add" (tsql_add_expr old value)
					"subtract" (tsql_fold_additive old (list "sub" value))
					_ (if (has? '("and" "or" "xor") op) (tsql_bit_expr op (list old value))
						(tsql_fold_multiplicative old (list op value)))))
				(list (declaration "Field") (tsql_assignment_expr assigned declaration))))))))))
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
			(if policy (policy (source_schema target) (source_relation target) true) true)
			(define values (tsql_update_values target (tsql_finalize_sources cols "query" cte_bindings '())))
			(if describe (tsql_describe_dml (source_schema target) (tsql_write_sources target sources)
				values bound_condition)
				(tsql_build_dml_plan (source_schema target) (source_relation target) (source_alias target)
					(tsql_write_sources target sources) values
					bound_condition nil limit nil planning_session tx))))))
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
			(define trunc_schema (tsql_qualifier_database schema2 schema frontend_options))
			(cons '!begin (list
				(list (quote checktablemaintenance) trunc_schema tbl "truncate")
				(tsql_build_dml_plan trunc_schema tbl nil (list (list tbl trunc_schema tbl false nil)) nil true nil nil nil planning_session tx)))
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
			(if (and (equal? mode "alter") ifnotexists)
				(error "ALTER VIEW does not accept IF NOT EXISTS") true)
			(define query (sql_apply_view_column_aliases
				(tsql_finalize_sources (nth captured 1) "query" '() '()) aliases))
			(if (tsql_view_has_metadata_scalar? query)
				(error "COL_LENGTH in stored views requires a consuming compile policy") true)
			(bind_query_names (tsql_bind_column_compilers (sql_expand_views query policy)
				(tsql_column_compiler planning_session)) '())
			(list (quote tsql_create_view_definition)
				(list (quote session) "__memcp_tx")
				view_schema id "tsql" (nth captured 0) (list (quote quote) query)
				(if ifnotexists "ignore" mode)))))))

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
			nil nil nil nil nil '() '() '()) policy planning_session))
		(define fields (tsql_result_columns bound parameters))
		(if (list? fields) (list (quote quote) '()) (error "invalid prepared mutation contract")))))
	(define tsql_build_select_plan (lambda (query) (begin
		(define bound_query (tsql_finalize_sources query "query" '() '()))
		(define expanded (tsql_bind_column_compilers (sql_expand_views bound_query policy) (tsql_column_compiler planning_session)))
		/* Binding validates names and contracts but lowers conversion callbacks.
		Describe the original declared expressions so NULL folding cannot erase
		their explicit widths or temporal precision. Analysis still consumes the
		bound tree; neither path constructs or executes a physical plan. */
		(define checked (if describe (tsql_describe_query expanded policy planning_session) expanded))
		(define descriptors (tsql_result_columns expanded parameters))
		(if analyze_parameters (list (quote quote) checked)
			describe (list (quote quote) descriptors) (list (quote !begin)
				(list (quote resultfields) (list (quote quote) (queryplan_result_titles expanded))
					(list (quote quote) descriptors))
				(tsql_result_output_plan (build_queryplan_term expanded planning_session tx) descriptors))))))
	/* INSERT SELECT consumes the existing planner's result stream and the native
	insert operator. Identity allocation and constraints remain native. */
	(define tsql_insert_select_plan (lambda (database tbl columns explicit_columns query) (begin
		(define declarations (tsql_insert_declarations database tbl columns planning_session))
		(define columns (map declarations (lambda (column) (column "Field"))))
		(define expanded (tsql_bind_column_compilers (sql_expand_views query policy) (tsql_column_compiler planning_session)))
		(define identity (tsql_insert_identity database tbl planning_session))
		(define insert_expr (tsql_native_insert_expr (quote __tsql_insert_target) identity columns
			(cons list '((cons list (map (produceN (count columns))
				(lambda (i) (begin
					/* SQL source contracts come from the logical projection compiler. */
					(define info (sql_projection_info_at expanded i '()))
					(sql_type_formula '() (tsql_assignment_expr
						(list (quote sql_type_expr) '('nth 'item (+ (* i 2) 1))
							(sql_info_type info) (sql_info_collation info))
						(nth declarations i))))))))) 1 false))
		(if (equal? (count (queryplan_result_titles expanded)) (count columns)) true
			(error "INSERT column count does not match SELECT column count"))
		(if describe (begin
			(define checked (tsql_describe_query expanded policy planning_session))
			/* Bind the source contract before accepting a pure prepared mutation. */
			(define checked_fields (tsql_result_columns checked parameters))
			(if (equal? (count checked_fields) (count declarations)) (list (quote quote) '())
				(error "invalid prepared INSERT source"))) (begin
				(define native_plan (build_queryplan_term expanded planning_session tx))

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
				(define ordered (or
					(and (query_block? query) (not (empty_list? (coalesceNil (qb_order query) '()))))
					(and (union_block? query) (not (empty_list? (coalesceNil (union_order query) '()))))))
				(tsql_identity_insert_statement database tbl identity explicit_columns columns (if ordered
					(list (quote begin)
						(list (quote set) (quote __insert_count) (list (quote newsession)))
						(list (quote __insert_count) "count" 0)
						(list (quote set) (quote resultrow) (list (quote lambda) (list (quote item))
							(list (quote begin)
								(list (quote set) (quote __inserted) insert_expr)
								(list (quote __insert_count) "count" (list (quote +) (list (quote __insert_count) "count") (quote __inserted)))
								(quote __inserted))))
						plan (list (quote __insert_count) "count"))
					(list (quote begin)
						(list (quote set) (quote resultrow) (list (quote lambda) (list (quote item)) insert_expr)) plan))))))
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
	(define tsql_insert_into (parser '(
		(atom "INSERT" true) (define limit tsql_write_limit) (? (atom "INTO" true))
		(? (define schema2 tsql_schema_identifier)) (define tbl tsql_identifier)
		(? "(" (define columns (+ tsql_identifier ",")) ")")
		(atom "VALUES" true) (define rows (+ tsql_insert_values_row ","))
	) (lambda cte_arguments (begin
			(define cte_bindings (if (empty_list? cte_arguments) '() (car cte_arguments)))
			(if (and (nil? schema2) (find cte_bindings (lambda (entry) (equal?? (car entry) tbl)) nil))
				(error "INSERT target must be a base table, not a CTE") true)
			(define database (tsql_qualifier_database schema2 schema frontend_options))
			(define tbl (tsql_table_name database tbl planning_session))
			(if policy (policy database tbl true) true)
			(define explicit_columns (not (nil? columns)))
			(define identity (tsql_insert_identity database tbl planning_session))
			(define declarations (tsql_insert_declarations database tbl
				(coalesce columns (tsql_insert_target_columns database tbl)) planning_session))
			(define columns (map declarations (lambda (column) (column "Field"))))
			(if (reduce rows (lambda (valid row) (and valid (equal? (count row) (count columns)))) true)
				true (error "INSERT column count does not match value count"))
			/* VALUES is a syntax-bounded batch. Limit that batch before native ID
			reservation; SELECT limits stay in the shared logical query block. */
			(define rows (map (if (nil? limit) rows (slice rows 0 limit)) (lambda (row)
				(map (produceN (count columns)) (lambda (i)
					(tsql_assignment_value
						(tsql_finalize_sources (nth row i) "query" cte_bindings '()) (nth declarations i)))))))
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
								(map typed_rows (lambda (row) (map row (lambda (value) (sql_type_formula '() value)))))) (count rows) true)))))
	))))
	/* An all-default row is the native empty-column INSERT. Defaults and
	generated values are filled by the same operator as ordinary insertion. */
	(define tsql_insert_default (parser '(
		(atom "INSERT" true) (define limit tsql_write_limit) (? (atom "INTO" true))
		(? (define schema2 tsql_schema_identifier)) (define tbl tsql_identifier)
		(atom "DEFAULT" true) (atom "VALUES" true)
	) (lambda cte_arguments (begin
			(define cte_bindings (if (empty_list? cte_arguments) '() (car cte_arguments)))
			(if (and (nil? schema2) (find cte_bindings (lambda (entry) (equal?? (car entry) tbl)) nil))
				(error "INSERT target must be a base table, not a CTE") true)
			(define database (tsql_qualifier_database schema2 schema frontend_options))
			(define tbl (tsql_table_name database tbl planning_session))
			(if policy (policy database tbl true) true)
			(define identity (tsql_insert_identity database tbl planning_session))
			(if describe (begin
				(tsql_insert_declarations database tbl '() planning_session)
				(list (quote quote) '()))
				(tsql_identity_insert_statement database tbl identity false '()
					(tsql_native_insert_expr (quote __tsql_insert_target) identity '()
						(if (equal?? limit 0) '('list) '('list '(list)))
						(if (equal?? limit 0) 0 1) true)))))))
	(define tsql_insert_select (parser '(
		(atom "INSERT" true) (define limit tsql_write_limit) (? (atom "INTO" true))
		(? (define schema2 tsql_schema_identifier)) (define tbl tsql_identifier)
		(? "(" (define columns (+ tsql_identifier ",")) ")")
		(define query tsql_select)
	) (lambda cte_arguments (begin
			(define cte_bindings (if (empty_list? cte_arguments) '() (car cte_arguments)))
			(if (and (nil? schema2) (find cte_bindings (lambda (entry) (equal?? (car entry) tbl)) nil))
				(error "INSERT target must be a base table, not a CTE") true)
			(define database (tsql_qualifier_database schema2 schema frontend_options))
			(define tbl (tsql_table_name database tbl planning_session))
			(if policy (policy database tbl true) true)
			(define identity (tsql_insert_identity database tbl planning_session))
			(tsql_insert_select_plan database tbl (coalesce columns
				(tsql_insert_target_columns database tbl)) (not (nil? columns))
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
			(define database (tsql_qualifier_database schema2 schema frontend_options))
			(define tbl (tsql_table_name database tbl planning_session))
			(if policy (policy database tbl true) true)
			(if (and unique predicate) (error "filtered UNIQUE requires native filtered-key support") true)
			(tsql_check_index_options options unique)
			/* PRIMARY is a reserved native key identity, not an index alias. */
			(if (and unique (equal? (toUpper name) "PRIMARY"))
				(error "index name PRIMARY is reserved") true)
			(define declarations (get_schema database tbl))
			(define columns (map columns (lambda (name) (begin
				(define declaration (find declarations (lambda (column) (equal?? (column "Field") name)) nil))
				(if declaration (declaration "Field") (error "unknown index column: " name))))))
			(map included (lambda (name) (if (find declarations (lambda (column) (equal?? (column "Field") name)))
				true (error "unknown included column: " name))))
			(reduce columns (lambda (seen name)
				(if (has? seen name) (error "duplicate index column: " name) (cons name seen))) '())
			(if (not unique) true (list (quote if)
				(list (quote createkey) (list (quote table) database false tbl false) name true
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
			(get_schema (tsql_qualifier_database schema2 schema frontend_options) tbl)
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
			(if (equal? (toUpper name) "PRIMARY") (error "PRIMARY is a constraint, not an acceleration index") true)
			/* A physical name carries no catalog state. Until native key removal
			protects referenced constraints, do not guess whether this name owns
			a real key or run a metadata check followed by an unsafe drop. */
			(error "DROP INDEX requires native constraint-safe key removal")))))
	(define tsql_foreign_key_mode (parser (or
		(parser '((atom "NO" true) (atom "ACTION" true)) "restrict")
		(parser (atom "CASCADE" true) "cascade")
		(parser '((atom "SET" true) (atom "NULL" true)) "set null"))))
	/* Constant defaults compile through the same declared expression contracts
	as DML. Folding must succeed: volatile expressions are never sampled at DDL
	time. The existing native clock marker remains the explicit dynamic default. */
	(define tsql_default_value (parser (or
		(parser tsql_clock_default '("default_expression" "CURRENT_TIMESTAMP"))
		(parser (define expression tsql_expression) (begin
			(define info (sql_expr_info '() expression))
			/* Keep the parsed source contract until the column's destination is
			known. tsql_prepare_column freezes the converted native default. */
			(list "default" (list (quote sql_type_expr) (tryfold (sql_info_formula info))
				(sql_info_type info) (sql_info_collation info))))))))
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
		(? (atom "ON" true) (atom "DELETE" true) (define deletemode tsql_foreign_key_mode))
		(? (atom "ON" true) (atom "UPDATE" true) (define updatemode tsql_foreign_key_mode))
	) (list (quote list) "foreign" name (cons (quote list) columns) target
			(cons (quote list) target_columns) updatemode deletemode target_schema)))
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
			(parser (define column column_definition) (match column '(name type dimensions attributes)
				(list (quote list) "column" name type dimensions (cons (quote list) attributes))))
		) ",")) ")"
		(? tsql_index_location)
	) (lambda () (begin
			(if policy (policy (tsql_qualifier_database schema2 schema frontend_options) name true) true)
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
			(list (quote tsql_create_relation) target_database name
				(cons (quote list) (map columns tsql_identity_native_column))
				(tsql_identity_table_options columns) (if ifnotexists true false) (quote tx))))))
	(define tsql_alter_table (parser '(
		(atom "ALTER" true) (atom "TABLE" true)
		(? (define schema2 tsql_schema_identifier)) (define name tsql_identifier)
		(define changes (+ (or
			(parser '((atom "ALTER" true) (atom "COLUMN" true) (define column tsql_identifier)
				(define type tsql_declared_type) (define dimensions tsql_type_dimensions)
				(? (atom "COLLATE" true) (define collation tsql_identifier))
				(define nullable (or (parser '((atom "NOT" true) (atom "NULL" true)) false)
					(parser (atom "NULL" true) true) (parser empty true)))
			) (lambda (database name) (begin
					(define handle (table database false name false))
					(define previous (if (nil? handle) nil (show handle "column" column true)))
					(if (nil? previous) (error "ALTER column does not exist: " column) true)
					(define dimensions (tsql_decl_dimensions type dimensions))
					(define builtin (tsql_builtin_type_declaration type))
					(define dimensions (if builtin (builtin "dimensions") dimensions))
					(define type (tsql_storage_type type))
					(define collation (coalesceNil collation (coalesceNil (previous "Collation") "")))
					(if (tsql_compatible_column_declaration? previous type dimensions nullable collation) true
						(error "ALTER COLUMN requires native validation or conversion"))
					(list (quote !begin) (list (quote altercolumn) (list (quote table) database false name false)
						(previous "Field") "declaration"
						(list (quote quote) (list (list type dimensions nullable collation)
							(tsql_column_declaration_tuple previous)))) true))))

			(parser '((atom "ADD" true) (? (atom "CONSTRAINT" true) (define key tsql_identifier))
				(atom "PRIMARY" true) (atom "KEY" true) tsql_key_placement "(" (define columns tsql_key_columns) ")" tsql_key_layout)
				(lambda (database name) (begin
					/* Nullable existing columns need native ALTER nullability support.
					Reject this shape rather than adding a parallel validation scan. */
					(if (reduce columns (lambda (valid column) (and valid
						(not ((find (show database name) (lambda (meta) (equal? (meta "Field") column))) "Null")))) true)
						true (error "ADD PRIMARY KEY currently requires NOT NULL columns"))
					(list (quote createkey) (list (quote table) database false name false) "PRIMARY" true
						(cons (quote list) columns) (quote tx)
						(cons (quote list) (if key (list "constraint_name" key) '()))))))
			(parser '((atom "ADD" true) (atom "CONSTRAINT" true) (define key tsql_identifier)
				(atom "UNIQUE" true) tsql_key_placement "(" (define columns tsql_key_columns) ")" tsql_key_layout)
				(lambda (database name)
					(list (quote createkey) (list (quote table) database false name false) key true
						(cons (quote list) columns) (quote tx)
						(list (quote list) "nulls_not_distinct" true "constraint_name" key))))
			(parser '((atom "ADD" true) (? (atom "CONSTRAINT" true) (define key tsql_identifier))
				(atom "DEFAULT" true) (define default tsql_default_value)
				(atom "FOR" true) (define column tsql_identifier))
				(lambda (database name) (begin
					(define declaration (tsql_column_description database name column true nil))
					(if (nil? declaration) (error "DEFAULT column does not exist: " column) true)
					(define value (if (has_assoc? default "default")
						(eval (sql_type_formula '() (tsql_assignment_expr
							(default "default") declaration))) nil))
					(list (quote !begin)
						(list (quote altercolumn) (list (quote table) database false name false)
							(declaration "Field") "add_default"
							(list (quote quote) (list value (coalesceNil (default "default_expression") "")
								(coalesceNil key (concat "DF_" name "_" (declaration "Field")))))) true))))
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
						(list (quote altertable) (list (quote table) database false name false)
							(if ifexists "drop_if_exists" "drop")
							(coalesceNil (resolve_column_name database name col true) col)))))))
			(parser '((atom "DROP" true) (atom "CONSTRAINT" true)
				(define ifexists (? (atom "IF" true) (atom "EXISTS" true)))
				(define keys (+ tsql_identifier ",")))
				(lambda (database name) (cons (quote !begin) (map keys (lambda (key)
					(list (quote if) (list (quote or)
						(list (quote altercolumn) (list (quote table) database false name false) key "drop_default" nil)
						(list (quote dropforeignkey) (list (quote table) database false name false) key)
						(list (quote dropkey) (list (quote table) database false name false) key (quote tx)))
						true (if ifexists true (list (quote error) "constraint does not exist: " key))))))))
		) ","))
	) (lambda () (begin
			(define database (tsql_qualifier_database schema2 schema frontend_options))
			(define name (tsql_table_name database name planning_session))
			(if policy (policy database name true) true)
			(cons (quote !begin) (map changes (lambda (change) (change database name))))))))
	(define tsql_transaction_keyword (if script_syntax
		(parser (or (atom "TRAN" true) (atom "TRANSACTION" true)))
		(parser (? (or (atom "TRAN" true) (atom "TRANSACTION" true))))))
	(define tsql_rollback (parser '((atom "ROLLBACK" true)
		(? (or (atom "TRAN" true) (atom "TRANSACTION" true))))
		(lambda () (list (quote tx_rollback) (list (quote tsql_connection_session) (quote session))))))
	(define p (parser (or
		tsql_cte_mutation
		(parser (define query tsql_root_select) (lambda () (tsql_build_select_plan query)))
		(parser '((atom "EXPLAIN" true) (atom "IR" true) (define query tsql_root_select)) (lambda () (explain_queryplan_ir (sql_expand_views (tsql_finalize_sources query "query" '() '()) policy))))
		(parser '((atom "EXPLAIN" true) (atom "REORDER" true) (define query tsql_root_select)) (lambda () (explain_queryplan_reorder (sql_expand_views (tsql_finalize_sources query "query" '() '()) policy) planning_session)))
		(parser '((atom "EXPLAIN" true) (atom "COMPILE" true) (define query tsql_root_select)) (lambda () (explain_queryplan_compile (sql_expand_views (tsql_finalize_sources query "query" '() '()) policy) parse_started_ns (strlen s) planning_session)))
		(parser '((atom "EXPLAIN" true) (atom "PHYSICAL" true) (define query tsql_root_select)) (lambda () (explain_queryplan_physical (sql_expand_views (tsql_finalize_sources query "query" '() '()) policy) planning_session)))
		(parser '((atom "EXPLAIN" true) (define query tsql_root_select))
			'(lambda () ('resultrow '('list "code" (pretty_print (optimize (build_queryplan_term (sql_expand_views (tsql_finalize_sources query "query" '() '()) policy) planning_session tx)) (settings "ExplainWidth"))))))
		tsql_insert_into tsql_insert_default tsql_insert_select tsql_update tsql_delete tsql_truncate
		tsql_create_view tsql_create_table tsql_alter_table tsql_create_index tsql_alter_index tsql_drop_index
		(parser '((atom "CREATE" true) (atom "TYPE" true)
			(? (define owner tsql_identifier) ".") (define name tsql_identifier)
			(atom "FROM" true) (define base tsql_declared_type) (define dimensions tsql_type_dimensions)
			(define nullable (or (parser '((atom "NOT" true) (atom "NULL" true)) false)
				(parser (atom "NULL" true) true) (parser empty true))))
			(lambda () (begin
				(if (or (nil? owner) (equal?? owner "dbo")) true (error "unsupported type owner: " owner))
				(if policy (policy schema true true) true)
				(list (quote create_sql_type_alias) schema name base (list (quote quote) dimensions) nullable (quote tx)))))
		(parser '((atom "DROP" true) (atom "TYPE" true)
			(define ifexists (? (atom "IF" true) (atom "EXISTS" true)))
			(? (define owner tsql_identifier) ".") (define name tsql_identifier))
			(lambda () (begin
				(if (or (nil? owner) (equal?? owner "dbo")) true (error "unsupported type owner: " owner))
				(if policy (policy schema true true) true)
				(list (quote drop_sql_type_alias) schema name (if ifexists true false) (quote tx)))))
		(parser '((atom "CREATE" true) (atom "DATABASE" true)
			(define ifnotexists (? (atom "IF" true) (atom "NOT" true) (atom "EXISTS" true)))
			(define name tsql_identifier)) (lambda () (begin
				(if policy (policy "system" true true) true)
				(list (quote create_sql_database) (quote tx) name (if ifnotexists true false) nil nil))))
		(parser '((atom "DROP" true) (atom "DATABASE" true)
			(define ifexists (? (atom "IF" true) (atom "EXISTS" true))) (define name tsql_identifier))
			(lambda () (begin (if policy (policy "system" true true) true)
				(list (quote drop_sql_database) (list (quote session) "__memcp_tx") name (if ifexists true false)))))
		(parser '((atom "DROP" true) (atom "TABLE" true)
			(define ifexists (? (atom "IF" true) (atom "EXISTS" true)))
			(? (define schema2 tsql_schema_identifier)) (define name tsql_identifier))
			(lambda () (begin
				(define database (tsql_qualifier_database schema2 schema frontend_options))
				(define name (tsql_table_name database name planning_session))
				(if policy (policy database name true) true)
				(list (quote droptable) database name (if ifexists true false)))))
		(parser '((atom "DROP" true) (atom "VIEW" true)
			(define ifexists (? (atom "IF" true) (atom "EXISTS" true)))
			(? (define schema2 tsql_schema_identifier)) (define name tsql_identifier))
			(lambda () (begin
				(define database (tsql_qualifier_database schema2 schema frontend_options))
				(define name (coalesceNil (tsql_view_name database name) name))
				(if policy (policy database name true) true)
				(list (quote drop_sql_view) (list (quote session) "__memcp_tx")
					database name (if ifexists true false)))))
		(parser '((atom "SET" true) (atom "IDENTITY_INSERT" true)
			(? (define schema2 tsql_schema_identifier)) (define name tsql_identifier)
			(define enabled (or (parser (atom "ON" true) true) (parser (atom "OFF" true) false))))
			(lambda () (begin
				(define database (tsql_qualifier_database schema2 schema frontend_options))
				(if policy (policy database name true) true)
				(list (quote tsql_set_identity_insert) (quote session) database name enabled))))
		(parser '((atom "SET" true) (atom "NOCOUNT" true)
			(define enabled (or (parser (atom "ON" true) true) (parser (atom "OFF" true) false))))
			(lambda () (list (quote session) "__tsql_nocount" enabled)))
		(parser '((atom "SET" true) (atom "DATEFIRST" true)
			(define first (or tsql_int
				(parser (define name tsql_variable_name) (parameter_expression name name)))))
			(lambda () (list (quote session) "__tsql_datefirst" (list tsql_datefirst_value (sql_type_formula '() first)))))
		/* These ON modes are the frontend's fixed comparison and lexical rules.
		Accept the declarations without adding state or branches to row execution;
		OFF and other settings remain explicit unsupported syntax. */
		(parser '((atom "SET" true) (or (atom "ANSI_NULLS" true) (atom "QUOTED_IDENTIFIER" true))
			(atom "ON" true)) (lambda () (list (quote quote) nil)))
		(parser '((atom "USE" true) (define name tsql_identifier)) (lambda () (begin
			(if policy (policy (coalesceNil import_target name) true false) true)
			(if import_target (list (quote tsql_import_database) (quote session) name)
				(list (quote session) "schema" name)))))
		(parser '((atom "BEGIN" true) tsql_transaction_keyword)
			(lambda () (list (quote tx_begin) (list (quote tsql_connection_session) (quote session)) (quote tx))))
		(parser '((atom "COMMIT" true) (? (or (atom "TRAN" true) (atom "TRANSACTION" true))))
			(lambda () (list (quote tx_commit) (list (quote tsql_connection_session) (quote session)))))
		tsql_rollback
	)))


	(define command_parser (if analyze_parameters
		(parser (define query tsql_root_select) (lambda () (tsql_build_select_plan query)))
		describe (parser (or (parser (define query tsql_root_select) (lambda () (tsql_build_select_plan query)))
			tsql_cte_mutation tsql_insert_into tsql_insert_default tsql_insert_select tsql_update tsql_delete))
		(parser (or p (parser empty (lambda () ""))))))
	(if expression_only
		((parser (define expression tsql_expression) (sql_expr_info '() (tsql_finalize_sources expression "query" '() '())) "^(?:/\\*(?s:.*?)\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+") s)
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
				(parser (define statement (capture tsql_rollback)) (list "recovery" (strtrim (car statement))))
				(parser '((atom "BEGIN" true) (atom "TRY" true) (define body (* tsql_script_item empty))
					(atom "END" true) (atom "TRY" true) (atom "BEGIN" true) (atom "CATCH" true)
					(define recovery (* tsql_script_item empty)) (atom "END" true) (atom "CATCH" true))
					(list "try" body recovery))
				(parser '((atom "WHILE" true) (define condition (capture tsql_expression))
					(define body tsql_script_command)) (list "while" (car condition) body))
				(parser (atom "BREAK" true) (list "break"))
				(parser (atom "CONTINUE" true) (list "continue"))
				(parser '((atom "RETURN" true) (? (define value (capture tsql_expression))))
					(list "return" (if value (car value) nil)))
				(parser '((atom "RAISERROR" true) "(" (define arguments (+ (capture tsql_expression) ",")) ")")
					(list "raiserror" (map arguments car)))
				(parser '((atom "THROW" true) (? (define arguments
					'((capture tsql_expression) "," (capture tsql_expression) "," (capture tsql_expression)))))
					(list "throw" (if arguments (list (car (nth arguments 0)) (car (nth arguments 2)) (car (nth arguments 4))) nil)))
				(parser '((atom "IF" true) (define condition (capture tsql_expression))
					(define yes tsql_script_command)
					(? (atom "ELSE" true) (define no tsql_script_command)))
					(list "if" (car condition) yes no))
				(parser '((atom "BEGIN" true) (define commands (* tsql_script_item empty)) (atom "END" true))
					(list "block" commands))
				(parser '((atom "DECLARE" true) (define declarations (+ tsql_script_declaration ",")))
					(list "declare" declarations))
				(parser '((atom "SET" true) (define name tsql_script_variable)
					(define operation (or (parser "&=" "&") (parser "|=" "|") (parser "^=" "^") (parser "=" nil)))
					(define value (capture tsql_expression)))
					(list "assign" name operation (car value)))
				(parser '((atom "DROP" true) (or (atom "PROCEDURE" true) (atom "PROC" true))
					(define ignore (? (atom "IF" true) (atom "EXISTS" true)))
					(? (define owner tsql_schema_identifier)) (define name tsql_identifier))
					(list "drop-procedure" (tsql_qualifier_database owner schema frontend_options) name (if ignore true false)))
				(parser '((or (atom "EXEC" true) (atom "EXECUTE" true))
					"(" (define text (capture tsql_expression)) ")")
					(list "execute-dynamic" (car text)))
				(parser '((or (atom "EXEC" true) (atom "EXECUTE" true))
					(define invocation tsql_script_invocation)) invocation)
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
			(define tsql_script_procedure (parser '(
				(atom "CREATE" true) (define replace (? (atom "OR" true) (atom "ALTER" true)))
				(or (atom "PROCEDURE" true) (atom "PROC" true))
				(? (define owner tsql_schema_identifier)) (define name tsql_identifier)
				(define parameters (* tsql_script_procedure_parameter ","))
				(atom "AS" true) (define body (capture (+ tsql_script_item empty)))
			) (list "batch" (list (list "create-procedure" (tsql_qualifier_database owner schema frontend_options) name
					(tsql_unique_parameters parameters) (strtrim (car body)) (if replace true false))))))
			(define script_parser (parser (define batch (or tsql_script_procedure
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

(define parse_tsql_batch (lambda (schema text) (begin
	(define syntax (newsession))
	(syntax "__tsql_script_syntax" true)
	(define parsed (parse_tsql schema text nil syntax nil))
	(tsql_validate_script_control (cadr parsed) 0)
	parsed)))
