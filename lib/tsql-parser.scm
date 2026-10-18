/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* Independent T-SQL grammar. Statements emit the canonical query AST and
existing native DDL/DML operations, as the other SQL frontends do. */
/* Integer tokens and integer text conversions use native int64 arithmetic.
The negative accumulator also represents the minimum int64 without overflow. */
(define tsql_parse_integer (lambda (text) (begin
	(define trimmed (sql_trim text))
	(if (equal? trimmed "") 0 (begin
		(if (regexp_test trimmed "^[+-]?[0-9]+$") true (error "invalid integer text"))
		(define negative (equal? (substr trimmed 0 1) "-"))
		(define digits (regexp_replace trimmed "^[+-]?0*" ""))
		(define bound (if negative "9223372036854775808" "9223372036854775807"))
		(if (or (> (strlen digits) 19) (and (equal? (strlen digits) 19) (> digits bound)))
			(error "integer conversion overflow") true)
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
	(if (find columns (lambda (column) (match column
		'("column" _name _type _dimensions attributes) (has_assoc? attributes "__tsql_alias_id") _ false)))
		(tsql_type_ddl database tx publish) (publish)))))
(define tsql_create_view_definition (lambda (tx database name dialect text query mode) (begin
	(if (show (table database false name true) "name") (error "a table already owns this relation name: " name) true)
	(define existing (tsql_view_name database name))
	(if (and (nil? existing) (equal? mode "alter")) (error "view does not exist: " name) true)
	(create_sql_view tx database (coalesceNil existing name) dialect text query
		(if (equal? mode "alter") "replace" mode)))))
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
		(list (quote sql_compile_column) tsql_compile_column '('get_column alias true name true)))
	(parser (define name tsql_identifier)
		(list (quote sql_compile_column) tsql_compile_column '('get_column nil true name true))))))
(define tsql_hex_literal (parser (define value sql_hex_literal)
	(list (quote sql_type_expr) value "VARBINARY" nil)))
(define tsql_literal (parser (or
	(parser (atom "NULL" true) (sql_null_literal)) tsql_hex_literal tsql_number tsql_text_literal)))
/* Binary semantics are chosen from declared SQL contracts at the shared
compile boundary. Byte equality reuses the native collation-aware operator;
no SQL type wrapper survives into execution or storage. */
(define tsql_binary_type? (lambda (type)
	(has? '("BINARY" "VARBINARY" "IMAGE" "ROWVERSION") type)))
(define tsql_comparison_expr (lambda (op left right)
	(list (quote sql_compile_expr) (lambda (infos) (begin
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
			(sql_info_formula comparison)) "BOOLEAN" (sql_info_collation comparison)))) left right)))
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
/* Catalog values describe SQL declarations. COL_LENGTH reads one immutable
native column row; it never scans data or treats compressed bytes as SQL width. */
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
	(if (number? value) (if (and (>= value (- 0 tsql_float_max)) (<= value tsql_float_max)) value (error "floating conversion overflow"))
		(if (string? value) (begin
			(define text (sql_trim value))
			(if (equal? text "") 0
				(if (regexp_test text "^[+-]?(?:[0-9]+\\.?[0-9]*|\\.[0-9]+)(?:[eE][+-]?[0-9]+)?$")
					(begin (define number (simplify text))
						(if (and (number? number) (>= number (- 0 tsql_float_max)) (<= number tsql_float_max)) number
							(error "floating conversion overflow")))
					(error "invalid numeric text"))))
			(if (equal? value true) 1 (if (equal? value false) 0 (error "unsupported numeric conversion")))))))
(define tsql_cast_integer (lambda (value minimum maximum)
	(if (nil? value) nil (begin
		(define integer (if (string? value) (tsql_parse_integer value)
			(if (int? value) value (intdiv (tsql_numeric_input value) 1))))
		(if (or (nil? integer) (< integer minimum) (> integer maximum))
			(error "integer conversion overflow") integer)))))
(define tsql_cast_float (lambda (value) (if (nil? value) nil (simplify (tsql_numeric_input value)))))
(define tsql_cast_decimal (lambda (value precision scale) (if (nil? value) nil (begin
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
		(error "decimal conversion overflow"))))))
(define tsql_cast_binary (lambda (value length fixed) (if (nil? value) nil (begin
	(if (string? value) true (error "numeric binary conversion is not implemented"))
	(define bytes (if (or (equal? length -1) (<= (strlen value) length)) value (substr value 0 length)))
	(if (and fixed (< (strlen bytes) length))
		(concat bytes (string_repeat (hex2bin "00") (- length (strlen bytes)))) bytes)))))
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
		(cadr (reduce (split text "") (lambda (prefix character) (begin
			(define units (if national (if (equal? (strlen character) 4) 2 1) (strlen character)))
			(if (< (car prefix) units) (list 0 (cadr prefix))
				(list (- (car prefix) units) (concat (cadr prefix) character)))))
			(list length ""))))))))
(define tsql_cast_fixed_text (lambda (value length national) (if (nil? value) nil (begin
	(define text (tsql_cast_text value length national))
	(define units (if national (reduce (split text "") (lambda (count character)
		(+ count (if (equal? (strlen character) 4) 2 1))) 0) (strlen text)))
	(concat text (string_repeat " " (- length units)))))))
/* A cast keeps its syntax-declared SQL type until the shared expression
compiler consumes the (expr, type, collation) contract. Runtime values do not
select that type, and no arithmetic operation is needed merely to mark it. */

(define tsql_cast_expr (lambda (value type dimensions) (begin
	(define name (toUpper type))
	(if (tsql_builtin_type_declaration name) (error "alias types cannot be explicit conversion targets") true)
	(tsql_decl_dimensions name dimensions)
	(define formula (match name
		"INT" (list tsql_cast_integer value -2147483648 2147483647)
		"INTEGER" (list tsql_cast_integer value -2147483648 2147483647)
		"SMALLINT" (list tsql_cast_integer value -32768 32767)
		"TINYINT" (list tsql_cast_integer value 0 255)
		"BIGINT" (list tsql_cast_integer value (tsql_parse_integer "-9223372036854775808") (tsql_parse_integer "9223372036854775807"))
		"FLOAT" (list tsql_cast_float value)
		/* Approximate types use native floating precision. Binary32 rounding and
		dialect precision boundaries remain noncritical compatibility work. */
		"REAL" (list tsql_cast_float value)
		"BIT" (list tsql_cast_bit value)
		"DECIMAL" (tsql_cast_decimal_expr value name dimensions)
		"NUMERIC" (tsql_cast_decimal_expr value name dimensions)
		"BINARY" (tsql_cast_binary_expr value dimensions true)
		"VARBINARY" (tsql_cast_binary_expr value dimensions false)
		"VARCHAR" (tsql_cast_text_expr value dimensions false false)
		"NVARCHAR" (tsql_cast_text_expr value dimensions true false)
		"CHAR" (tsql_cast_text_expr value dimensions false true)
		"NCHAR" (tsql_cast_text_expr value dimensions true true)
		"DATE" (tsql_temporal_cast_expr value name 0)
		"DATETIME" (tsql_temporal_cast_expr value name 0)
		"DATETIME2" (tsql_temporal_cast_expr value name 0)
		"SMALLDATETIME" (tsql_temporal_cast_expr value name 0)
		"TIME" (tsql_temporal_cast_expr value name 0)
		_ (error "unsupported conversion to " name)))
	(list (quote sql_type_expr) formula name nil))))
(define tsql_cast_decimal_expr (lambda (value type dimensions) (begin
	(define dims (tsql_decl_dimensions type dimensions))
	(list (quote sql_compile_expr) (lambda (infos) (begin
		(define input (car infos))
		(define input_type (sql_info_type input))
		(if (or (sql_numeric_type? input_type)
			(has? '("NULL" "any" "MONEY" "SMALLMONEY" "CHAR" "VARCHAR" "NCHAR" "NVARCHAR" "TEXT" "NTEXT") input_type)) true
			(error "decimal conversion from " input_type " is not implemented"))
		(sql_info (list (quote tsql_cast_decimal) (sql_info_formula input) (car dims) (cadr dims)) type nil))) value))))
(define tsql_cast_binary_expr (lambda (value dimensions fixed) (begin
	(define length (car (tsql_decl_dimensions (if fixed "BINARY" "VARBINARY")
		(if (empty_list? dimensions) '(30) dimensions))))
	(list (quote sql_compile_expr) (lambda (infos) (begin
		(define input (car infos))
		/* Text requires its declared encoding, and numeric values have another
		padding direction. Refuse those shapes rather than guess from strings. */
		(if (or (tsql_binary_type? (sql_info_type input)) (has? '("NULL" "any") (sql_info_type input))) true
			(error "binary conversion from " (sql_info_type input) " is not implemented"))
		(sql_info (list (quote tsql_cast_binary) (sql_info_formula input) length fixed)
			(if fixed "BINARY" "VARBINARY") nil))) value))))
/* Assignment contracts come from native column declarations at compile time.
Integer narrowing is an ordinary frontend cast. Existing native decimal scale
normalization remains in use; this guard rejects overflow, including carry from
rounding, before native insertion. Full precision-limit emulation is deferred. */
(define tsql_assignment_decimal (lambda (value threshold) (if (nil? value) nil (begin
	(define number (tsql_numeric_input value))
	(if (or (>= number threshold) (<= number (- 0 threshold))) (error "decimal conversion overflow") number)))))
(define tsql_assignment_expr (lambda (value declaration) (begin
	(define type (toUpper (declaration "RawType")))
	(if (equal? type "ROWVERSION")
		(error "generated version column cannot be assigned explicitly")
		(declaration "AliasId") (tsql_cast_expr value type (declaration "Dimensions"))
		(has? '("INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT") type)
		(tsql_cast_expr value type '())
		(has? '("NVARCHAR" "NCHAR") type)
		(tsql_cast_expr value type (declaration "Dimensions"))
		(has? '("DECIMAL" "NUMERIC") type) (begin
			(define dimensions (declaration "Dimensions"))
			(define precision (if (empty_list? dimensions) 18 (car dimensions)))
			(define scale (if (< (count dimensions) 2) 0 (cadr dimensions)))
			(define threshold (- (simplify (concat "1e" (- precision scale)))
				(* 0.5 (simplify (concat "1e-" scale)))))
			(list (quote sql_type_expr) (list tsql_assignment_decimal value threshold) type nil))
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
(define tsql_insert_declarations (lambda (database table_name columns planning_session) (begin	/* Resolve each target exactly once. Duplicate targets can overwrite an
	already validated identity/default value in the native column vector. */
	(if (equal? (count columns) (count (merge_unique (list (map columns toLower)))))
		true (error "duplicate INSERT column"))
	(map columns (lambda (name) (begin
		(define declaration (tsql_target_column_declaration database table_name name planning_session))
		(if (nil? declaration) (error "unknown INSERT column: " name) declaration)))))))
(define tsql_cast_text_expr (lambda (value dimensions national fixed) (begin
	/* Resolve the operand at the shared compile boundary, where its sources
	and SQL contract are known. A parser-local walk would lose column types. */
	(define length (if (empty_list? dimensions) 30 (car dimensions)))
	(list (quote sql_compile_expr) (lambda (infos) (begin
		(define info (car infos))
		(define source (sql_info_type info))
		(define formula (if (tsql_temporal_type? source)
			(list format_date (tsql_calendar_difference_formula info)
				(match source "DATE" "%Y-%m-%d" "TIME" "%H:%i:%s"
					"DATETIME2" "%Y-%m-%d %H:%i:%s.000" _ "%Y-%m-%d %H:%i:%s"))
			(sql_info_formula info)))
		(if (and (plain_literal_expr? formula) (sql_numeric_type? (sql_info_type info))
			(not (equal? length -1)) (> (strlen (string formula)) length))
			(error "numeric character conversion overflow") true)
		(sql_info (list (if fixed (quote tsql_cast_fixed_text) (quote tsql_cast_text)) formula length national)
			(if (and national fixed) "NCHAR" national "NVARCHAR" fixed "CHAR" "VARCHAR")
			(sql_info_collation info)))) value))))
/* The default character contract is NCHAR(1), without a supplementary-character
collation. SQL conversion and range decisions stay here; the runtime operator
only encodes Unicode scalars. Isolated surrogate code units are unsupported. */
(define tsql_nchar (lambda (value round_input) (if (nil? value) nil (begin
	(define code (tsql_cast_integer (if round_input (round value) value) -2147483648 2147483647))
	(if (or (< code 0) (> code 65535)) nil (unicode_char code))))))
(define tsql_nchar_expr (lambda (value)
	(list (quote sql_compile_expr) (lambda (infos) (begin
		(define info (car infos))
		(define type (tsql_addition_type (sql_info_type info)))
		/* Implicit INT conversion is a SQL decision. Money rounds; other numeric
		inputs truncate. Temporal carriers never become integer codes. */
		(if (has? '("NULL" "BIT" "TINYINT" "SMALLINT" "INT" "BIGINT" "REAL" "FLOAT"
			"DECIMAL" "MONEY" "SMALLMONEY" "CHAR" "VARCHAR" "NCHAR" "NVARCHAR") type) true
			(error "NCHAR requires an implicitly convertible integer expression"))
		(sql_info (list tsql_nchar (sql_info_formula info) (has? '("MONEY" "SMALLMONEY") type)) "NCHAR" nil))) value)))
/* Export-style day/month/year spelling is a frontend concern. Parse into Unix
seconds with the existing date primitives; no temporal runtime type is added.
Two-digit years use the default 1950..2049 window. Fractional precision and the
configurable cutoff remain separate compatibility work. */
(define tsql_export_datetime_seconds (lambda (value) (if (nil? value) nil (begin
	(define text (toUpper (sql_trim (string value))))
	(if (regexp_test text "^[0-9]{2}[.-][0-9]{2}[.-][0-9]{2}( [0-9]{2}:[0-9]{2}:[0-9]{2}( (AM|PM))?)?$")
		true (error "invalid day/month/year datetime text"))
	(define year (substr text 6 2))
	(define date (concat (if (< (tsql_parse_integer year) 50) "20" "19") year "-" (substr text 3 2) "-" (substr text 0 2)))
	(define clock (if (equal? (strlen text) 8) "00:00:00" (substr text 9 8)))
	(define clock (if (> (strlen text) 17) (begin
		(define hour (tsql_parse_integer (substr clock 0 2)))
		(if (> hour 12) (error "invalid twelve-hour datetime text") true)
		(define hour (+ (mod hour 12) (if (equal? (substr text 18 2) "PM") 12 0)))
		(concat (if (< hour 10) "0" "") hour (substr clock 2 6))) clock))
	(define parsed (str_to_date (concat date " " clock) "%Y-%m-%d %H:%i:%s"))
	(if (nil? parsed) (error "invalid datetime value") (unix_timestamp parsed))))))
(define tsql_convert_expr (lambda (value type dimensions style) (begin
	(if (tsql_builtin_type_declaration type) (error "alias types cannot be explicit conversion targets") true)
	(tsql_decl_dimensions type dimensions)
	(match style
		(symbol sql_null_style) (list (quote sql_type_expr) nil type nil)
		_ (if (or (nil? style) (equal? style 0)) (tsql_cast_expr value type dimensions)
			(tsql_temporal_type? type) (tsql_temporal_cast_expr value type style)
			(has? '("VARCHAR" "NVARCHAR" "CHAR" "NCHAR") type)
			(begin
				(define pattern (tsql_temporal_style_pattern style true))
				(list (quote sql_compile_expr) (lambda (infos) (begin
					(define input (car infos))
					(define formula (list format_date (tsql_calendar_difference_formula input) pattern))
					(sql_info (list (if (has? '("CHAR" "NCHAR") type) tsql_cast_fixed_text tsql_cast_text)
						formula (if (empty_list? dimensions) 30 (car dimensions)) (has? '("NVARCHAR" "NCHAR") type)) type nil))) value)))
		(error "unsupported CONVERT style for " type)))))
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
(define tsql_aggregate_expr (lambda (name formula value)
	(list (quote sql_compile_expr) (lambda (infos)
		(tsql_aggregate_result name (sql_info_formula (car infos))
			(tsql_aggregate_type name (sql_info_type (cadr infos)))
			(sql_info_collation (cadr infos)))) formula value)))
/* Preserve the native window AST for logical stage construction. Its result
contract is compiled from resolved input triples, never from an empty scope. */
(define tsql_window_expr (lambda (name args over)
	(cons (quote sql_compile_expr)
		(cons (lambda (infos) (begin
			(define input_type (if (empty_list? args) "any" (sql_info_type (cadr infos))))
			(define type (tsql_aggregate_type name input_type))
			(define formula (match (sql_info_formula (car infos))
				((symbol window_func) function _arguments specification)
				(list (quote window_func) function (map (cdr infos) sql_info_formula) specification)
				_ (sql_info_formula (car infos))))
			(tsql_aggregate_result name formula type
				(if (empty_list? args) nil (sql_info_collation (cadr infos))))))
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
	(define anchored (if (regexp_test text "^[0-9]{2}:[0-9]{2}:[0-9]{2}(\\.[0-9]+)?$")
		(concat "1900-01-01 " text) text))
	(define parsed (parse_date (regexp_replace anchored "T" " ")))
	(if (nil? parsed) (error "invalid calendar input") parsed)))))
(define tsql_calendar_days (lambda (value) (if (nil? value) nil (begin
	(if (and (>= value -53690) (< value 2958464)) true (error "numeric datetime conversion overflow"))
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
		value (error "calendar result overflow")))))
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
		rounded (error "small datetime conversion overflow"))))))
(define tsql_temporal_style_pattern (lambda (style output) (match style
	1 (if output "%m/%d/%y" "%m/%d/%Y") 101 "%m/%d/%Y"
	4 (if output "%d.%m.%y" "%d.%m.%Y") 104 "%d.%m.%Y"
	12 (if output "%y%m%d" "%Y%m%d") 112 "%Y%m%d"
	8 "%H:%i:%s" 108 "%H:%i:%s"
	126 "%Y-%m-%dT%H:%i:%s"
	23 "%Y-%m-%d" 20 "%Y-%m-%d %H:%i:%s" 120 "%Y-%m-%d %H:%i:%s"
	21 (if output "%Y-%m-%d %H:%i:%s.000" "%Y-%m-%d %H:%i:%s")
	121 (if output "%Y-%m-%d %H:%i:%s.000" "%Y-%m-%d %H:%i:%s")
	_ (error "unsupported CONVERT style: " style))))
(define tsql_temporal_style_text (lambda (value style) (if (nil? value) nil (begin
	(define text (sql_trim value))
	(define pattern (tsql_temporal_style_pattern style false))
	(define text (if (has? '(1 12) style) (begin
		(define offset (if (equal? style 12) 0 6))
		(define year (substr text offset 2))
		(if (regexp_test year "^[0-9]{2}$") true (error "invalid short calendar year"))
		(concat (substr text 0 offset) (if (< (tsql_parse_integer year) 50) "20" "19")
			year (substr text (+ offset 2)))) text))
	(define value (if (has? '(21 121) style) (parse_date text) (str_to_date text pattern)))
	(if (nil? value) (error "invalid styled calendar input") value)))))
/* The current native carrier has whole seconds. Explicit conversions that
would retain fractional clocks fail until a generic higher-precision carrier
is available; accepting them here would silently lose the user's value. */
(define tsql_calendar_cast_text (lambda (value) (if (nil? value) nil (begin
	(if (regexp_test value "[0-9]{2}:[0-9]{2}:[0-9]{2}\\.[0-9]*[1-9]")
		(error "fractional temporal conversion is not implemented") value)))))
(define tsql_temporal_cast_expr (lambda (value target style)
	(list (quote sql_compile_expr) (lambda (infos) (begin
		(define input (car infos))
		(define source (sql_info_type input))
		(define input (if (and (sql_text_type? source) (not (equal? target "DATE")))
			(sql_info (list tsql_calendar_cast_text (sql_info_formula input)) source (sql_info_collation input)) input))
		(if (and (sql_numeric_type? source) (not (has? '("DATETIME" "SMALLDATETIME") target)))
			(error "numeric input is not convertible to " target) true)
		(if (and (not (equal? style 0)) (not (equal? style 4)))
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
		(sql_info converted target nil))) value)))
(define tsql_dateadd_expr (lambda (unit amount value) (begin
	(define unit (tsql_calendar_unit unit))
	(if (equal? unit "iso_week") (error "DATEADD does not support ISO week") true)
	(define months (match unit "year" 12 "quarter" 3 "month" 1 _ nil))
	(define seconds (match unit "second" 1 "minute" 60 "hour" 3600 "week" 604800 _ 86400))
	(list (quote sql_compile_expr) (lambda (infos) (begin
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
			_ (list tsql_calendar_range formula minimum)) type nil))) amount value))))
(define tsql_calendar_difference_formula (lambda (info)
	(if (equal? (sql_info_type info) "TIME") (list tsql_calendar_text (sql_info_formula info))
		(tsql_calendar_formula info))))
(define tsql_calendar_scalar_expr (lambda (unit value)
	(list (quote sql_compile_expr) (lambda (infos)
		(sql_info (list (match unit "year" (sql_builtins "YEAR") "month" (sql_builtins "MONTH") _ (sql_builtins "DAY"))
			(tsql_calendar_difference_formula (car infos))) "INT" nil)) value)))
(define tsql_calendar_unavailable_part (lambda (value)
	(if (nil? value) nil (error "calendar date part is not available for declared TIME"))))
(define tsql_calendar_part_formula (lambda (unit info)
	(if (equal? (sql_info_type info) "TIME")
		(if (has? '("hour" "minute" "second") unit) (tsql_calendar_difference_formula info)
			(list tsql_calendar_unavailable_part (sql_info_formula info)))
		(tsql_calendar_formula info))))
(define tsql_datediff_expr (lambda (unit first last type) (begin
	(define unit (tsql_calendar_unit unit))
	(if (equal? unit "iso_week") (error "DATEDIFF does not support ISO week") true)
	(define boundary (tsql_boundary_calculator unit))
	(define minimum (if (equal? type "INT") -2147483648 (tsql_parse_integer "-9223372036854775808")))
	(define maximum (if (equal? type "INT") 2147483647 (tsql_parse_integer "9223372036854775807")))
	(list (quote sql_compile_expr) (lambda (infos) (begin
		(define difference (list (quote -)
			(list boundary (tsql_calendar_difference_formula (cadr infos)))
			(list boundary (tsql_calendar_difference_formula (car infos)))))
		(sql_info (list tsql_cast_integer difference minimum maximum) type nil))) first last))))
(define tsql_datepart_expr (lambda (unit value) (begin
	(define unit (tsql_calendar_unit unit))
	(define part (match unit
		"year" (sql_builtins "YEAR") "quarter" (sql_builtins "QUARTER") "month" (sql_builtins "MONTH")
		"day" (sql_builtins "DAY") "dayofyear" tsql_calendar_day_of_year "hour" (sql_builtins "HOUR")
		"minute" (sql_builtins "MINUTE") "second" (sql_builtins "SECOND")
		"weekday" tsql_calendar_weekday "week" tsql_calendar_week "iso_week" tsql_calendar_iso_week))
	(if (has? '("week" "weekday") unit)
		(list (quote sql_compile_expr) (lambda (infos)
			(sql_info (list part (tsql_calendar_part_formula unit (car infos))
				(list tsql_datefirst_value (sql_info_formula (cadr infos)))) "INT" nil)) value (tsql_datefirst_expr))
		(list (quote sql_compile_expr) (lambda (infos)
			(sql_info (list part (tsql_calendar_part_formula unit (car infos))) "INT" nil)) value)))))


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
(define tsql_datefromparts_expr (lambda (year month day)
	(list (quote sql_compile_expr) (lambda (infos)
		(sql_info (cons tsql_calendar_from_parts (map infos tsql_calendar_integer_formula)) "DATE" nil)) year month day)))
(define tsql_calendar_month_end (lambda (value amount)
	(if (or (nil? value) (nil? amount)) nil (begin
		(define shifted (tsql_calendar_shift_months value amount))
		(define first (date_add (date_trunc_day shifted) (intdiv (- 1 (extract_date shifted "DAY")) 1) "DAY"))
		(tsql_calendar_range (date_add (date_add first 1 "MONTH") -1 "DAY") -62135596800)))))
(define tsql_eomonth_expr (lambda (value amount)
	(list (quote sql_compile_expr) (lambda (infos)
		(sql_info (list tsql_calendar_month_end (tsql_calendar_formula (car infos))
			(tsql_calendar_integer_formula (cadr infos))) "DATE" nil)) value amount)))

(define tsql_numeric_operand_type? (lambda (type)
	(or (has? '("NULL" "any" "MONEY" "SMALLMONEY") type)
		(and (sql_numeric_type? type) (not (has? '("BOOLEAN" "BOOL" "BIT") type))))))
(define tsql_positive_expr (lambda (value)
	(list (quote sql_compile_expr) (lambda (infos) (begin
		(define info (car infos))
		(if (tsql_numeric_operand_type? (sql_info_type info)) info
			(error "unary plus requires a numeric operand")))) value)))
(define tsql_negate_expr (lambda (value)
	(list (quote sql_compile_expr) (lambda (infos) (begin
		(define info (car infos))
		(define formula (sql_info_formula info))
		(define type (sql_info_type info))
		/* Diagnose constant conversion overflow using the declared range.
		Column-bearing operands are resolved by the shared compiler first. */
		(match formula
			'(conversion operand minimum maximum)
			(if (and (equal? conversion tsql_cast_integer) (plain_literal_expr? operand)
				(< minimum 0) (equal? (tsql_cast_integer operand minimum maximum) minimum))
				(error "unary integer conversion overflow") true)
			_ true)
		(if (tsql_numeric_operand_type? type) true
			(error "unary minus requires a numeric operand"))
		(sql_info (list (quote -) 0 formula) (if (equal? type "TINYINT") "SMALLINT" type) nil))) value)))
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
	(parser (define value tsql_literal) (sql_type_lower_annotations value)))))
(define tsql_clock_default (parser (or
	(parser '("(" tsql_clock_default ")") true)
	'((atom "GETDATE" true) "(" ")") (atom "CURRENT_TIMESTAMP" true))))
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
/* Emit the existing native INSERT. VALUES and DEFAULT supply their bounded
syntax batch; the existing SELECT result sink supplies one emitted row. Never
scan/count/materialize SELECT input to reserve IDs, or rewind consumed blocks.
The frontend checks the complete reserved block before any row publication.
Explicit NULL must fail here: native auto_increment would otherwise allocate
an unchecked replacement. Completion uses only this invocation's reserved range. */
(define tsql_native_insert_expr (lambda (database name columns values row_count capture_identity) (begin
	(define relation (list (quote table) database false name false))
	(define identity (find (get_schema database name)
		(lambda (column) (equal? (column "Extra") "auto_increment")) nil))
	(define native_insert (lambda (names rows callback)
		(list (quote insert) relation (list (quote quote) names) rows
			'(list) nil false callback (quote tx))))
	(define without_generated_identity (lambda (formula)
		(if capture_identity
			(list (quote begin)
				(list (quote define) (quote __tsql_insert_result) formula)
				(list (quote session) "__tsql_identity" nil)
				(quote __tsql_insert_result)) formula)))
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
					'('define '__tsql_explicit_result (native_insert columns (quote __tsql_rows) callback))
					(if capture_identity '('tsql_unresolved_identity 'session "explicit IDENTITY insertion") true)
					'__tsql_explicit_result)
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
	(list (quote sql_compile_expr) tsql_compile_addition left right)))
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
(define tsql_divide_expr (lambda (left right)
	(list (quote sql_compile_expr) (lambda (infos) (begin
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
		(sql_info (if integer (sql_type_formula '() (tsql_cast_expr formula target '())) formula) target nil))) left right)))
(define tsql_fold_multiplicative (lambda (acc term) (match term
	'("divide" value) (tsql_divide_expr acc value)
	_ (sql_fold_multiplicative_term acc term))))
(define tsql_fold_additive (lambda (acc term) (match term
	'("add" value) (tsql_add_expr acc value)
	_ (sql_fold_additive_term acc term))))
/* NULL fallback has the same declared precedence as other T-SQL expressions.
The shared lazy coalesce operator remains responsible for evaluation; this
callback consumes SQL triples and leaves only converted formulas in the plan.
Dimension merging and exact decimal precision are separate frontend work. */
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
	(map infos (lambda (info) (begin
		(define type (tsql_addition_type (sql_info_type info)))
		(define formula (sql_info_formula info))
		(define converted (if (or (equal? type "NULL") (equal? type target)) formula
			(and (has? tsql_addition_numeric_types target)
				(or (has? tsql_addition_numeric_types type) (has? tsql_addition_text_types type)))
			(if (has? '("DECIMAL" "MONEY" "SMALLMONEY") target)
				(list tsql_cast_float formula)
				(sql_type_lower_annotations (tsql_cast_expr formula target '())))
			(and (has? tsql_addition_text_types target) (has? tsql_addition_text_types type)) formula
			(error "unsupported implicit conversion from " type " to " target)))
		(sql_info converted target collation)))))))
(define tsql_coalesce_expr (lambda (arguments)
	(cons (quote sql_compile_expr) (cons (lambda (infos) (begin
		(if (< (count infos) 2) (error "COALESCE requires at least two arguments") true)
		(define converted (tsql_common_value_infos infos nil))
		(sql_info (cons (quote coalesceNil) (map converted sql_info_formula))
			(sql_info_type (car converted)) (sql_info_collation (car converted))))) arguments))))

/* Explicit wire descriptors come from the existing SQL expression metadata,
never from the first row. Unknown expression types stay text in this first slice. */
/* Width/scale originate in declaration syntax, never in the first output row.
The shared expression compiler still owns the SQL type; these additional wire
dimensions are metadata only and are absent from the executable plan. */
(define tsql_result_dimensions (lambda (sources expression parameters) (match expression
	((symbol sql_compile_column) _compiler column) (tsql_result_dimensions sources column parameters)
	((symbol sql_type_expr) formula _type _collation) (tsql_result_dimensions sources formula parameters)
	((symbol tsql_cast_decimal) _value precision scale) (list precision scale)
	((symbol tsql_read_identity) _state) '(38 0)
	((symbol tsql_cast_binary) _value length _fixed) (list length)
	((symbol tsql_cast_text) _value length _national) (list length)
	((symbol tsql_cast_fixed_text) _value length _national) (list length)
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
						(define metadata (find (get_schema (source_schema src) (source_relation src))
							(lambda (item) (equal? (item "Field") bound_column)) nil))
						(if metadata (metadata "Dimensions") nil))
					(match (source_relation src)
						((symbol query-block) _schema inner_sources fields _where _group _having _order _limit _offset _hidden _stages _facts)
						(tsql_result_dimensions inner_sources ((expand_query_block_fields inner_sources fields) bound_column) parameters)
						_ nil)))
			_ nil))
	_ nil)))
(define tsql_result_columns (lambda (query parameters) (match query
	((symbol query-block) _schema sources fields _where _group _having _order _limit _offset _hidden _stages _facts)
	(extract_assoc (expand_query_block_fields sources fields) (lambda (name expression) (begin
		(define info (sql_expr_info sources expression))
		(define type (sql_info_type info))
		(define dimensions (coalesceNil (tsql_result_dimensions sources expression parameters)
			(tsql_result_dimensions sources (sql_info_formula info) parameters) '()))
		(define numeric (and (has? '("DECIMAL" "NUMERIC") type) (>= (count dimensions) 2)))
		(define kind (if (has? '("INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT") type) 38
			numeric 108 (tsql_binary_type? type) 165
			(has? '("FLOAT" "DOUBLE" "REAL" "DECIMAL" "NUMERIC") type) 109
			(has? '("BOOLEAN" "BOOL" "BIT") type) 104 231))
		(define precision (if numeric (car dimensions) 0))
		(define scale (if numeric (cadr dimensions) 0))
		(define size (match kind
			108 (if (<= precision 9) 5 (<= precision 19) 9 (<= precision 28) 13 17)
			165 (if (equal? type "ROWVERSION") 8
				(and (not (empty_list? dimensions)) (> (car dimensions) 0)) (car dimensions) 65535)
			231 (if (and (has? '("NVARCHAR" "NCHAR") type) (not (empty_list? dimensions)))
				(if (equal? (car dimensions) -1) 65535 (* 2 (car dimensions))) 8000)
			104 1
			_ (match type "TINYINT" 1 "SMALLINT" 2 "INT" 4 "INTEGER" 4 _ 8)))
		(list "name" name "kind" kind "size" size "precision" precision "scale" scale "flags" 1 "sql_type" type))))
	((symbol union-block) _mode branches _order _limit _offset _facts) (tsql_result_columns (car branches) parameters)
	_ (error "unsupported T-SQL result metadata shape"))))
/* Format only the external result edge. Inner projections, ordering and native
storage keep their Unix values. The declaration selects each formatter while
compiling; the emitted row callback contains no SQL type tags or dispatch. */
(define tsql_result_output_plan (lambda (plan descriptors)
	(if (reduce descriptors (lambda (found column)
		(or found (tsql_temporal_type? (column "sql_type")))) false)
		(list (quote begin)
			(list (quote define) (quote __tsql_raw_resultrow) (quote resultrow))
			(list (quote define) (quote resultrow) (list (quote lambda) (list (quote row))
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
						(list (column "name") (if pattern (list format_date calendar_value pattern) value))))))))))
			plan)
		plan)))

(define tsql_builtins (lambda (name) (match name
	"ISNULL" coalesceNil
	"LEN" (lambda (value) (if (nil? value) nil (strlen (sql_rtrim value))))
	_ (if (has? '("ABS" "CEILING" "FLOOR" "ROUND" "SQRT" "UPPER" "LOWER" "REPLACE" "SUBSTRING" "CONCAT" "YEAR" "MONTH" "DAY") name)
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

(define parse_tsql (lambda (database s policy planning_session tx) (begin
	(define schema database)
	(define parse_started_ns (nanotime))
	(define frontend_options (if (nil? planning_session) (newsession) planning_session))
	(define describe (frontend_options "__tsql_describe"))
	(define script_syntax (frontend_options "__tsql_script_syntax"))
	(define expression_only (frontend_options "__tsql_expression_only"))
	(define parameters (coalesceNil (frontend_options "__tsql_parameter_declarations") (list)))
	(define parameter_expression (lambda (name) (begin
		(define name (toLower name))
		(define declaration (filter parameters (lambda (item) (equal? (car item) name))))
		(if script_syntax (list (quote tsql_batch_variable) name)
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

		(parser '((atom "NOT" true) (atom "EXISTS" true) "(" (define sub tsql_select) ")") (list (quote not) (list (quote inner_select_exists) sub)))
		(parser '((atom "NOT" true) (define expr tsql_expression2)) '('sql_not expr))

		(parser '((define a tsql_expression3) (atom "IN" true) "(" (define sub tsql_select) ")") '('inner_select_in a sub))
		(parser '((define a tsql_expression3) (atom "NOT" true) (atom "IN" true) "(" (define sub tsql_select) ")") (list (quote not) (list (quote inner_select_in) a sub)))

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

		(parser '((define a tsql_expression3) (atom "IN" true) (define b tsql_in_values)) '('sql_in b a))
		(parser '((define a tsql_expression3) (atom "NOT" true) (atom "IN" true) (define b tsql_in_values)) (list (quote sql_not) (list (quote sql_in) b a)))

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

	(define tsql_expression5 (parser (or
		(parser '((define expr tsql_expression6) (atom "IS" true) (atom "NULL" true)) '('nil? expr))
		(parser '((define expr tsql_expression6) (atom "IS" true) (atom "NOT" true) (atom "NULL" true)) '('not '('nil? expr)))
		tsql_expression6
		(parser '("+" (define expr tsql_expression6)) (tsql_positive_expr expr))
		(parser '("-" (define expr tsql_expression6)) (tsql_negate_expr expr))
	)))

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
			(list (lambda (relation column) (if (or (nil? relation) (nil? column)) nil (begin
				(define object ((parser '( (? (define owner tsql_identifier) ".")
					(define name tsql_identifier)) (list (if (or (nil? owner) (equal?? owner "dbo")) database owner) name)) relation))
				(define object_schema (car object))
				(define name (coalesceNil (show (table object_schema false (cadr object) true) "name") (cadr object)))
				(if policy (policy object_schema name false) true)
				(define handle (table object_schema false name false))
				(define declaration (if (nil? handle) nil (tsql_effective_column (show handle "column" column true) nil)))
				(if (nil? declaration) nil (tsql_declared_byte_length declaration))))) relation column)
			"INT" nil)))

	(define tsql_expression6 (parser (or

		(parser '("(" (define sub tsql_select) ")") '('inner_select sub))
		(parser '("(" (define a tsql_expression) ")") a)
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

		(parser '((atom "EXISTS" true) "(" (define sub tsql_select) ")") '('inner_select_exists sub))

		(parser '((atom "CASE" true) (define conditions (+ (parser '((atom "WHEN" true) (define a tsql_expression) (atom "THEN" true) (define b tsql_expression)) '(a b)))) (? (atom "ELSE" true) (define elsebranch tsql_expression)) (atom "END" true)) (merge '((quote if)) (merge conditions) '(elsebranch)))

		(parser '((atom "CASE" true) (define expr tsql_expression) (define conditions (+ (parser '((atom "WHEN" true) (define a tsql_expression) (atom "THEN" true) (define b tsql_expression)) '(a b)))) (? (atom "ELSE" true) (define elsebranch tsql_expression)) (atom "END" true)) (merge '((quote if)) (merge (extract_assoc (merge conditions) (lambda (a b) (list (tsql_comparison_expr "eq" expr a) b)))) '(elsebranch)))

		(parser '((atom "COUNT" true) "(" "*" ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") (tsql_window_expr "COUNT" (list) _over))
		(parser '((atom "COUNT" true) "(" (define e tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") (tsql_window_expr "COUNT" (list e) _over))
		(parser '((atom "SUM" true) "(" (define s tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") (tsql_window_expr "SUM" (list s) _over))
		(parser '((atom "AVG" true) "(" (define s tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") (tsql_window_expr "AVG" (list s) _over))
		(parser '((atom "MIN" true) "(" (define s tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") (tsql_window_expr "MIN" (list s) _over))
		(parser '((atom "MAX" true) "(" (define s tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") (tsql_window_expr "MAX" (list s) _over))

		(parser '((atom "COUNT" true) "(" (atom "DISTINCT" true) (define e tsql_expression) ")")
			'('sql_type_expr '('count_distinct e) "INT" nil))
		(parser '((atom "COUNT" true) "(" "*" ")")
			'('sql_type_expr '((quote aggregate) 1 (quote +) 0) "INT" nil))
		(parser '((atom "COUNT" true) "(" (define e tsql_expression) ")")
			'('sql_type_expr '('aggregate '((quote if) '((quote nil?) e) 0 1) (quote +) 0) "INT" nil))
		(parser '((atom "SUM" true) "(" (define s tsql_expression) ")") (begin (define d (sql_aggregates "SUM")) (tsql_aggregate_expr "SUM" '('aggregate s (car d) (cadr d)) s)))
		(parser '((atom "AVG" true) "(" (define s tsql_expression) ")")
			(tsql_aggregate_expr "AVG" (sql_avg_expr s (sql_aggregates "SUM") (sql_aggregates "COUNT")) s))
		(parser '((atom "MIN" true) "(" (define s tsql_expression) ")") (begin (define d (sql_aggregates "MIN")) (tsql_aggregate_expr "MIN" '('aggregate s (car d) (cadr d)) s)))
		(parser '((atom "MAX" true) "(" (define s tsql_expression) ")") (begin (define d (sql_aggregates "MAX")) (tsql_aggregate_expr "MAX" '('aggregate s (car d) (cadr d)) s)))

		(parser '((define fn tsql_identifier_unquoted) "(" (define arg tsql_expression) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")")
			(tsql_window_expr (toUpper fn) (list arg) _over))

		(parser '((atom "SUBSTRING" true) "(" (define s tsql_expression) "," (define start tsql_expression) "," (define len tsql_expression) ")") '((quote sql_substr) s start len))

		(parser '((atom "CONCAT" true) "(" (define p (+ tsql_expression ",")) ")")
			(cons (quote concat) (map p (lambda (value) (list (quote coalesceNil) value "")))))

		(parser '((atom "TRIM" true) "(" (define e tsql_expression) ")") '((quote sql_trim) e))
		(parser '((atom "LTRIM" true) "(" (define e tsql_expression) ")") '((quote sql_ltrim) e))
		(parser '((atom "RTRIM" true) "(" (define e tsql_expression) ")") '((quote sql_rtrim) e))

		(parser '((atom "COALESCE" true) "(" (define args (* tsql_expression ",")) ")") (tsql_coalesce_expr args))

		(parser '((atom "NULLIF" true) "(" (define a tsql_expression) "," (define b tsql_expression) ")") '((quote if) '((quote equal??) a b) nil a))
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

		(parser (define var tsql_variable_name) (parameter_expression var))

		(parser '((atom "LEFT" true) "(" (define s tsql_expression) "," (define n tsql_expression) ")") '((quote sql_substr) s 1 n))

		(parser '((atom "RIGHT" true) "(" (define s tsql_expression) "," (define n tsql_expression) ")") '((quote if) '((quote nil?) s) nil '((quote sql_substr) s '((quote +) 1 '((quote -) '((quote strlen) s) n)) n)))

		(parser '((define fn tsql_identifier_unquoted) "(" (define args (* tsql_expression ",")) ")" (atom "OVER" true) "(" (define _over tsql_window_spec) ")") (tsql_window_expr (toUpper fn) args _over))

		(parser '((define fn tsql_identifier_unquoted) "(" (define args (* tsql_expression ",")) ")")
			(begin (define d (sql_aggregates (toUpper fn)))
				(if (not (nil? d))
					'('aggregate (car args) (car d) (cadr d))
					(cons (coalesce (tsql_builtins (toUpper fn)) (error "unknown function " fn)) args))))

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
			(parser '((atom "CROSS" true) (atom "APPLY" true) (define r tabledef)) (list (tsql_apply_source r false)))
			(parser '((atom "OUTER" true) (atom "APPLY" true) (define r tabledef)) (list (tsql_apply_source r true)))
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
							(list (quote sql_compile_expr) (lambda (infos) (car infos)) expr))))
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
			(list id schema (tsql_inline_values_query aliases rows) false nil))

		(parser '((atom "(" true) (define query tsql_select) (atom ")" true) (atom "AS" true) (define id tsql_identifier) "(" (define aliases (+ tsql_identifier ",")) ")")
			(list id schema (tsql_apply_derived_column_aliases query aliases) false nil))
		(parser '((atom "(" true) (define query tsql_select) (atom ")" true) (define id tsql_identifier) "(" (define aliases (+ tsql_identifier ",")) ")")
			(list id schema (tsql_apply_derived_column_aliases query aliases) false nil))
		(parser '((atom "(" true) (define query tsql_select) (atom ")" true) (atom "AS" true) (define id tsql_identifier)) '(id schema query false nil))
		(parser '((atom "(" true) (define query tsql_select) (atom ")" true) (define id tsql_identifier)) '(id schema query false nil))

		(parser '((define schema tsql_schema_identifier) (atom "." true) (define tbl tsql_identifier) (atom "AS" true) (define id tsql_identifier) (? tsql_table_hints)) (tsql_read_source id schema tbl))
		(parser '((define schema tsql_schema_identifier) (atom "." true) (define tbl tsql_identifier) (define id tsql_identifier) (? tsql_table_hints)) (tsql_read_source id schema tbl))
		(parser '((define schema tsql_schema_identifier) (atom "." true) (define tbl tsql_identifier) (? tsql_table_hints)) (tsql_read_source tbl schema tbl))
		(parser '((define tbl tsql_identifier) (atom "AS" true) (define id tsql_identifier) (? tsql_table_hints)) (tsql_read_source id schema tbl))
		(parser '((define tbl tsql_identifier) (define id tsql_identifier) (? tsql_table_hints)) (tsql_read_source id schema tbl))
		(parser '((define tbl tsql_identifier) (? tsql_table_hints)) (tsql_read_source tbl schema tbl))
	)))

	(define from nil)

	(define condition nil)

	(define group nil)

	(define having nil)

	(define order nil)

	(define limit nil)

	(define offset nil)

	(define tsql_select_order (lambda (query) (match query
		((symbol query-block) schema tables fields condition group having order limit offset hidden stages facts) order
		_ nil
	)))

	(define tsql_select_limit (lambda (query) (match query
		((symbol query-block) schema tables fields condition group having order limit offset hidden stages facts) limit
		_ nil
	)))

	(define tsql_select_offset (lambda (query) (match query
		((symbol query-block) schema tables fields condition group having order limit offset hidden stages facts) offset
		_ nil
	)))

	(define tsql_select_clear_stage (lambda (query) (match query
		((symbol query-block) schema tables fields condition group having order limit offset hidden stages facts)
		(list (quote query-block) schema tables fields condition group having nil nil nil '() '() '())
		_ query
	)))

	(define tsql_union_all_parts (lambda (query)
		(match query
			((symbol union-block) (symbol all) branches order limit offset facts) (list branches order limit offset)
			_ nil)))

	(define tsql_union_distinct_parts (lambda (query)
		(match query
			((symbol union-block) (symbol distinct) branches order limit offset facts) (list branches order limit offset)
			_ nil)))

	(define tsql_union_all_query (lambda (left right) (begin
		(define right_parts (tsql_union_all_parts right))
		(if (nil? right_parts)
			(list (quote union-block)
				(quote all)
				(list left (tsql_select_clear_stage right))
				(tsql_select_order right)
				(tsql_select_limit right)
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
				(tsql_select_limit right)
				(tsql_select_offset right)
				'())
			(match right_parts '(branches order limit offset)
				(list (quote union-block) (quote distinct) (cons left branches) order limit offset '())))
	)))

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
				(tsql_union_all_query acc (tsql_values_row_query schema2 coldesc row)))
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

	(define tsql_select_core (parser '(
		(atom "SELECT" true)

		(define distinct (or (parser (atom "DISTINCT" true) true)
			(parser (? (atom "ALL" true)) false)))
		(define prefix_limit tsql_select_prefix)
		(define cols (+ (or
			/* Assignment aliases must precede expression comparisons. */
			(parser '((define title (or tsql_identifier tsql_string)) "=" (define e tsql_expression)) '(title e))
			(parser "*" (list "*" (list (quote sql_compile_column) tsql_compile_column (list (quote get_column) nil false "*" false))))
			(parser '((define tbl tsql_identifier_quoted) "." "*")
				(list "*" (list (quote sql_compile_column) tsql_compile_column '('get_column tbl true "*" false))))
			(parser '((define tbl tsql_identifier_unquoted) "." "*")
				(list "*" (list (quote sql_compile_column) tsql_compile_column '('get_column tbl true "*" false))))
			(parser '((define e tsql_expression) (atom "AS" true) (define title tsql_identifier)) '(title e))
			(parser '((define e tsql_expression) (atom "AS" true) (define title tsql_string)) '(title e))

			(parser '((define e tsql_expression) (define title tsql_identifier)) '(title e))
			(parser '((define e tsql_expression) (define title tsql_string)) '(title e))

			(parser (define captured (capture tsql_expression)) '((extract_title_or_sql captured) (car (cdr captured))))
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
				) (list col (if coll (collate coll (equal? direction_desc >)) direction_desc)))

				(atom "," true)
			))
		)

		(define suffix_stage tsql_select_suffix)
		(? tsql_query_hints)

	) (begin
			(if (and suffix_stage (nil? order)) (error "OFFSET requires ORDER BY") true)
			(if (and suffix_stage (not (nil? prefix_limit))) (error "TOP cannot be combined with OFFSET") true)
			(define projected_exprs (extract_assoc (merge cols) (lambda (_title expr) expr)))
			(define sources (if (nil? from) '() (merge from)))

			(define distinct_preserved_by_group (and distinct
				(and (not (nil? group))
					(and (not (empty_list? group))
						(reduce group (lambda (preserved expr)
							(and preserved (contains? projected_exprs expr))) true)))))
			(list (quote query-block) schema sources (merge cols) condition
				(if (and distinct (not distinct_preserved_by_group)) projected_exprs group)
				having
				order (if (nil? prefix_limit) (if suffix_stage (car suffix_stage) limit) prefix_limit)
				(if suffix_stage (cadr suffix_stage) offset) '() '()
				(merge (list
					/* The common name binder resolves all joined/derived sources.
					HAVING names belong to the input scope in this frontend. */
					(list (list (quote having_input_scope) true))
					(if distinct (list (list (quote select_distinct) true)) '())))))))

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

	/* A mutation target may name a FROM alias. Resolve that alias before checking
	write permission; the grammar never creates a fictitious table for it. */
	(define tsql_write_target (parser '(
		(? (define target_schema tsql_schema_identifier) ".")
		(define target_name tsql_identifier)
		(? tsql_table_hints)
	) (list target_schema target_name)))
	(define tsql_write_source (lambda (target sources) (match target
		'(target_schema target_name) (begin
			(define alias_match (if target_schema nil (find sources (lambda (src)
				(equal?? (source_alias src) target_name)) nil)))
			(define table_matches (filter sources (lambda (src) (and
				(source_is_base_table? src)
				(equal? (source_schema src) (coalesceNil target_schema schema))
				(equal?? (source_relation src) target_name)))))
			(define table_match (if (empty_list? table_matches) nil
				(equal? (count table_matches) 1) (car table_matches)
				(find table_matches (lambda (src) (equal?? (source_alias src) target_name)) nil)))
			(if (and (nil? alias_match) (> (count table_matches) 1) (nil? table_match))
				(error "ambiguous write target: " target_name) true)
			(define source (coalesceNil alias_match table_match
				(list target_name (coalesceNil target_schema schema)
					(tsql_table_name (coalesceNil target_schema schema) target_name planning_session) false nil)))
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
				(define old (list (quote sql_compile_column) tsql_compile_column
					(list (quote get_column) (source_alias target) false (declaration "Field") false)))
				(define assigned (match op "assign" value
					"add" (tsql_add_expr old value)
					"subtract" (tsql_fold_additive old (list "sub" value))
					_ (tsql_fold_multiplicative old (list op value))))
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
	) (lambda () (begin
			(define sources (merge (coalesceNil from '())))
			(define target (tsql_write_source target sources))
			(if policy (policy (source_schema target) (source_relation target) true) true)
			(tsql_build_dml_plan (source_schema target) (source_relation target) (source_alias target)
				(tsql_write_sources target sources) (tsql_update_values target cols)
				(coalesceNil condition true) nil limit nil planning_session tx)))))
	(define tsql_delete (parser '(
		(atom "DELETE" true)
		(define limit tsql_write_limit)
		(? (atom "FROM" true))
		(define target tsql_write_target)
		(? (atom "FROM" true) (define from (+ tabledefs ",")))
		(? (atom "WHERE" true) (define condition tsql_expression))
		(? tsql_query_hints)
	) (lambda () (begin
			(define sources (merge (coalesceNil from '())))
			(define target (tsql_write_source target sources))
			(if policy (policy (source_schema target) (source_relation target) true) true)
			(tsql_build_dml_plan (source_schema target) (source_relation target) (source_alias target)
				(tsql_write_sources target sources) nil (coalesceNil condition true)
				nil limit nil planning_session tx)))))

	(define tsql_truncate (parser '(
		(atom "TRUNCATE" true) (? (atom "TABLE" true))

		(? (define schema2 tsql_schema_identifier) ".") (define tbl tsql_identifier)
	) (lambda () (begin
			(if policy (policy (coalesce schema2 schema) tbl true) true)
			(define trunc_schema (coalesce schema2 schema))
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
			(parser '((define schema2 tsql_schema_identifier) "." (define id tsql_identifier)) '(schema2 id))
			(parser (define id tsql_identifier) '(nil id))))
		(define aliases (? (parser '("(" (define columns (+ tsql_identifier ",")) ")") columns)))
		(atom "AS" true)
		(define captured (capture tsql_select))
	) (lambda () (match target '(schema2 id) (begin
			(define view_schema (coalesce schema2 schema))
			(if policy (policy view_schema id true) true)
			(define query (sql_apply_view_column_aliases (nth captured 1) aliases))
			(list (quote tsql_create_view_definition)
				(list (quote session) "__memcp_tx")
				view_schema id "tsql" (nth captured 0) (list (quote quote) query)
				(if ifnotexists "ignore" mode)))))))

	/* A database-qualified dbo owner is still one native database. No second
	catalog or SQL-specific namespace is introduced into storage. */
	(define tsql_schema_identifier (parser (or
		(parser '((define owner tsql_identifier) "." (atom "dbo" true)) owner)
		(parser (define owner tsql_identifier) (if (equal?? owner "dbo") database owner)))))
	(define tsql_expression3 (parser '(
		(define a tsql_expression4)
		(define terms (* (parser '(
			(define op (or (parser "+" "add") (parser "-" "sub")))
			(define value tsql_expression4)
		) (list op value)) empty true))
	) (reduce terms tsql_fold_additive a)))
	(define tsql_build_select_plan (lambda (query) (begin
		(define expanded (tsql_bind_column_compilers (sql_expand_views query policy) (tsql_column_compiler planning_session)))
		(define descriptors (tsql_result_columns expanded parameters))
		(if describe (list (quote quote) descriptors) (list (quote !begin)
			(list (quote resultfields) (list (quote quote) (queryplan_result_titles expanded))
				(list (quote quote) descriptors))
			(tsql_result_output_plan (build_queryplan_term expanded planning_session tx) descriptors))))))
	/* INSERT SELECT consumes the existing planner's result stream and the native
	insert operator. Identity allocation and constraints remain native. */
	(define tsql_insert_select_plan (lambda (database tbl columns query) (begin
		(define declarations (tsql_insert_declarations database tbl columns planning_session))
		(define columns (map declarations (lambda (column) (column "Field"))))
		(define insert_expr (tsql_native_insert_expr database tbl columns
			(cons list '((cons list (map (produceN (count columns))
				(lambda (i) (sql_type_formula '() (tsql_assignment_expr
					'('nth 'item (+ (* i 2) 1)) (nth declarations i)))))))) 1 false))
		(define expanded (tsql_bind_column_compilers (sql_expand_views query policy) (tsql_column_compiler planning_session)))
		(if (equal? (count (queryplan_result_titles expanded)) (count columns)) true
			(error "INSERT column count does not match SELECT column count"))
		(define native_plan (build_queryplan_term expanded planning_session tx))
		(define identity (find (get_schema database tbl)
			(lambda (column) (equal? (column "Extra") "auto_increment")) nil))
		/* AND propagates SQL NULL and evaluates later terms. Missing declaration
		metadata needs an explicit presence branch before reading its fields. */
		(define generated (if (nil? identity) false (not (has? columns (identity "Field")))))
		/* Completion is once per statement, outside the parallel row sink. */
		(define plan (list (quote begin)
			(list (quote define) (quote __tsql_stream_result) native_plan)
			(if generated
				(list (quote if) (list (quote >) (quote __tsql_stream_result) 0)
					(list (quote tsql_unresolved_identity) (quote session) "streaming INSERT")
					(list (quote session) "__tsql_identity" nil))
				(if identity
					(list (quote tsql_unresolved_identity) (quote session) "explicit streaming IDENTITY insertion")
					(list (quote session) "__tsql_identity" nil)))
			(quote __tsql_stream_result)))
		(define ordered (or
			(and (query_block? query) (not (empty_list? (coalesceNil (qb_order query) '()))))
			(and (union_block? query) (not (empty_list? (coalesceNil (union_order query) '()))))))
		(if ordered
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
				(list (quote set) (quote resultrow) (list (quote lambda) (list (quote item)) insert_expr)) plan)))))
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
		(? (define schema2 tsql_schema_identifier) ".") (define tbl tsql_identifier)
		(? "(" (define columns (+ tsql_identifier ",")) ")")
		(atom "VALUES" true) (define rows (+ tsql_insert_values_row ","))
	) (lambda () (begin
			(define database (coalesce schema2 schema))
			(define tbl (tsql_table_name database tbl planning_session))
			(if policy (policy database tbl true) true)
			(define declarations (tsql_insert_declarations database tbl
				(coalesce columns (table_insertable_columns database tbl)) planning_session))
			(define columns (map declarations (lambda (column) (column "Field"))))
			(if (reduce rows (lambda (valid row) (and valid (equal? (count row) (count columns)))) true)
				true (error "INSERT column count does not match value count"))
			/* VALUES is a syntax-bounded batch. Limit that batch before native ID
			reservation; SELECT limits stay in the shared logical query block. */
			(define rows (map (if (nil? limit) rows (slice rows 0 limit)) (lambda (row)
				(map (produceN (count columns)) (lambda (i)
					(tsql_assignment_value (nth row i) (nth declarations i)))))))
			(if (reduce rows (lambda (found row) (or found (tsql_dataset_contains_inner_select row))) false)
				(tsql_insert_select_plan database tbl columns (tsql_values_to_select_query database columns rows))
				(tsql_native_insert_expr database tbl columns (sql_insert_values_expr (map rows (lambda (row) (map (produceN (count columns))
					(lambda (i) (sql_type_formula (list) (tsql_assignment_expr
						(nth row i) (nth declarations i)))))))) (count rows) true))
	))))
	/* An all-default row is the native empty-column INSERT. Defaults and
	generated values are filled by the same operator as ordinary insertion. */
	(define tsql_insert_default (parser '(
		(atom "INSERT" true) (define limit tsql_write_limit) (? (atom "INTO" true))
		(? (define schema2 tsql_schema_identifier) ".") (define tbl tsql_identifier)
		(atom "DEFAULT" true) (atom "VALUES" true)
	) (lambda () (begin
			(define database (coalesce schema2 schema))
			(define tbl (tsql_table_name database tbl planning_session))
			(if policy (policy database tbl true) true)
			(tsql_native_insert_expr database tbl '()
				(if (equal?? limit 0) '('list) '('list '(list)))
				(if (equal?? limit 0) 0 1) true)))))
	(define tsql_insert_select (parser '(
		(atom "INSERT" true) (define limit tsql_write_limit) (? (atom "INTO" true))
		(? (define schema2 tsql_schema_identifier) ".") (define tbl tsql_identifier)
		(? "(" (define columns (+ tsql_identifier ",")) ")")
		(define query tsql_select)
	) (lambda () (begin
			(define database (coalesce schema2 schema))
			(define tbl (tsql_table_name database tbl planning_session))
			(if policy (policy database tbl true) true)
			(tsql_insert_select_plan database tbl (coalesce columns (table_insertable_columns database tbl))
				(tsql_insert_limit_query query limit))
	))))
	/* Only UNIQUE declarations publish native keys. Ordinary acceleration
	declarations carry no catalog state: the common planner owns autoindexes. */
	(define tsql_create_index (parser '(
		(atom "CREATE" true) (define unique (? (atom "UNIQUE" true))) tsql_key_placement
		(atom "INDEX" true) (define name tsql_identifier) (atom "ON" true)
		(? (define schema2 tsql_schema_identifier) ".") (define tbl tsql_identifier)
		"(" (define columns tsql_key_columns) ")"
		(? (atom "INCLUDE" true) "(" (define included (+ tsql_identifier ",")) ")")
		(? (atom "WHERE" true) (define predicate tsql_expression))
		(? (define options tsql_index_options))
		(? tsql_index_location)
	) (lambda () (begin
			(define database (coalesce schema2 schema))
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
		(? (define schema2 tsql_schema_identifier) ".") (define tbl tsql_identifier)
		(define action (regex "(?:REBUILD|REORGANIZE|DISABLE)\\b" true))
		(? (atom "PARTITION" true) "=" (or tsql_int (atom "ALL" true)))
		(? (define options tsql_index_options))
	) (lambda () (begin
			(if policy (policy (coalesce schema2 schema) tbl true) true)
			(get_schema (coalesce schema2 schema) tbl)
			(if (equal? (toUpper action) "DISABLE") (error "DISABLE cannot bypass native constraints") true)
			(tsql_check_index_options options true)
			true))))
	(define tsql_drop_index (parser '(
		(atom "DROP" true) (atom "INDEX" true)
		(? (atom "IF" true) (atom "EXISTS" true))
		(define name tsql_identifier) (atom "ON" true)
		(? (define schema2 tsql_schema_identifier) ".") (define tbl tsql_identifier)
		(? tsql_index_options)
	) (lambda () (begin
			(if policy (policy (coalesce schema2 schema) tbl true) true)
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
		(parser (define expression tsql_expression)
			(list "default" (tryfold (sql_type_formula '() expression)))))))
	(define column_attributes (tsql_column_attributes tsql_default_value))
	(define column_definition (parser '(
		(define name tsql_identifier) (define type tsql_schema_type)
		(define dimensions tsql_type_dimensions) (define attributes column_attributes)
	) (list name type (cons (quote list) dimensions) attributes)))
	(define foreign_key_definition (parser '(
		(? (atom "CONSTRAINT" true) (define name tsql_identifier))
		(atom "FOREIGN" true) (atom "KEY" true) "(" (define columns (+ tsql_identifier ",")) ")"
		(atom "REFERENCES" true) (? tsql_schema_identifier ".") (define target tsql_identifier)
		"(" (define target_columns (+ tsql_identifier ",")) ")"
		(? (atom "ON" true) (atom "DELETE" true) (define deletemode tsql_foreign_key_mode))
		(? (atom "ON" true) (atom "UPDATE" true) (define updatemode tsql_foreign_key_mode))
	) (list (quote list) "foreign" name (cons (quote list) columns) target
			(cons (quote list) target_columns) updatemode deletemode)))
	(define tsql_create_table (parser '(
		(atom "CREATE" true) (atom "TABLE" true)
		(define ifnotexists (? (atom "IF" true) (atom "NOT" true) (atom "EXISTS" true)))
		(? (define schema2 tsql_schema_identifier) ".") (define name tsql_identifier)
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
			(if policy (policy (coalesce schema2 schema) name true) true)
			/* Explicit PRIMARY KEY policy precedes inherited alias nullability. */
			(define columns (map (tsql_table_primary_columns columns) (lambda (column)
				(tsql_prepare_column (coalesce schema2 schema) column planning_session))))
			(list (quote tsql_create_relation) (coalesce schema2 schema) name
				(cons (quote list) (map columns tsql_identity_native_column))
				(tsql_identity_table_options columns) (if ifnotexists true false) (quote tx))))))
	(define tsql_alter_table (parser '(
		(atom "ALTER" true) (atom "TABLE" true)
		(? (define schema2 tsql_schema_identifier) ".") (define name tsql_identifier)
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
							(list (quote quote) (default "default")) declaration))) nil))
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
			(define database (coalesce schema2 schema))
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
		(parser (define query tsql_select) (lambda () (tsql_build_select_plan query)))
		(parser '((atom "EXPLAIN" true) (atom "IR" true) (define query tsql_select)) (lambda () (explain_queryplan_ir (sql_expand_views query policy))))
		(parser '((atom "EXPLAIN" true) (atom "REORDER" true) (define query tsql_select)) (lambda () (explain_queryplan_reorder (sql_expand_views query policy) planning_session)))
		(parser '((atom "EXPLAIN" true) (atom "COMPILE" true) (define query tsql_select)) (lambda () (explain_queryplan_compile (sql_expand_views query policy) parse_started_ns (strlen s) planning_session)))
		(parser '((atom "EXPLAIN" true) (atom "PHYSICAL" true) (define query tsql_select)) (lambda () (explain_queryplan_physical (sql_expand_views query policy) planning_session)))
		(parser '((atom "EXPLAIN" true) (define query tsql_select))
			'(lambda () ('resultrow '('list "code" (pretty_print (optimize (build_queryplan_term (sql_expand_views query policy) planning_session tx)) (settings "ExplainWidth"))))))
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
			(? (define schema2 tsql_schema_identifier) ".") (define name tsql_identifier))
			(lambda () (begin
				(define database (coalesce schema2 schema))
				(define name (tsql_table_name database name planning_session))
				(if policy (policy database name true) true)
				(list (quote droptable) database name (if ifexists true false)))))
		(parser '((atom "DROP" true) (atom "VIEW" true)
			(define ifexists (? (atom "IF" true) (atom "EXISTS" true)))
			(? (define schema2 tsql_schema_identifier) ".") (define name tsql_identifier))
			(lambda () (begin
				(define database (coalesce schema2 schema))
				(define name (coalesceNil (tsql_view_name database name) name))
				(if policy (policy database name true) true)
				(list (quote drop_sql_view) (list (quote session) "__memcp_tx")
					database name (if ifexists true false)))))
		(parser '((atom "SET" true) (atom "NOCOUNT" true)
			(define enabled (or (parser (atom "ON" true) true) (parser (atom "OFF" true) false))))
			(lambda () (list (quote session) "__tsql_nocount" enabled)))
		(parser '((atom "SET" true) (atom "DATEFIRST" true)
			(define first (or tsql_int
				(parser (define name tsql_variable_name) (parameter_expression name)))))
			(lambda () (list (quote session) "__tsql_datefirst" (list tsql_datefirst_value (sql_type_formula '() first)))))
		/* These ON modes are the frontend's fixed comparison and lexical rules.
		Accept the declarations without adding state or branches to row execution;
		OFF and other settings remain explicit unsupported syntax. */
		(parser '((atom "SET" true) (or (atom "ANSI_NULLS" true) (atom "QUOTED_IDENTIFIER" true))
			(atom "ON" true)) (lambda () (list (quote quote) nil)))
		(parser '((atom "USE" true) (define name tsql_identifier)) (lambda () (begin
			(if policy (policy name true false) true)
			(list (quote session) "schema" name))))
		(parser '((atom "BEGIN" true) tsql_transaction_keyword)
			(lambda () (list (quote tx_begin) (list (quote tsql_connection_session) (quote session)) (quote tx))))
		(parser '((atom "COMMIT" true) (? (or (atom "TRAN" true) (atom "TRANSACTION" true))))
			(lambda () (list (quote tx_commit) (list (quote tsql_connection_session) (quote session)))))
		tsql_rollback
	)))


	(define command_parser (if describe
		(parser (define query tsql_select) (lambda () (tsql_build_select_plan query))) (parser (or p (parser empty (lambda () ""))))))
	(if expression_only
		((parser (define expression tsql_expression) (sql_expr_info '() expression) "^(?:/\\*(?s:.*?)\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+") s)
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
				(parser '((atom "SET" true) (define name tsql_script_variable) "=" (define value (capture tsql_expression)))
					(list "assign" name (car value)))
				(parser '((atom "DROP" true) (or (atom "PROCEDURE" true) (atom "PROC" true))
					(define ignore (? (atom "IF" true) (atom "EXISTS" true)))
					(? (define owner tsql_schema_identifier) ".") (define name tsql_identifier))
					(list "drop-procedure" (coalesce owner schema) name (if ignore true false)))
				(parser '((or (atom "EXEC" true) (atom "EXECUTE" true))
					"(" (define text (capture tsql_expression)) ")")
					(list "execute-dynamic" (car text)))
				(parser '((or (atom "EXEC" true) (atom "EXECUTE" true))
					(define status tsql_script_variable) "="
					(? (define owner tsql_schema_identifier) ".") (define name tsql_identifier)
					(define arguments (* tsql_script_exec_argument ",")))
					(list "execute-status" (coalesce owner schema) name arguments status))
				(parser '((or (atom "EXEC" true) (atom "EXECUTE" true))
					(? (define owner tsql_schema_identifier) ".") (define name tsql_identifier)
					(define arguments (* tsql_script_exec_argument ",")))
					(list "execute" (coalesce owner schema) name arguments))
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
				(? (define owner tsql_schema_identifier) ".") (define name tsql_identifier)
				(define parameters (* tsql_script_procedure_parameter ","))
				(atom "AS" true) (define body (capture (+ tsql_script_item empty)))
			) (list "batch" (list (list "create-procedure" (coalesce owner schema) name
					(tsql_unique_parameters parameters) (strtrim (car body)) (if replace true false))))))
			(define script_parser (parser (define batch (or tsql_script_procedure
				(parser '((* ";") (define commands (* tsql_script_item empty)) (* ";"))
					(list "batch" commands)))) batch "^(?:/\\*(?s:.*?)\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+"))

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
