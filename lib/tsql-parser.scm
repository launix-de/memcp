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
	(atom "APPLY" true)
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
)))
(define tsql_identifier_quoted (parser (or
	(parser '("[" (define id (regex "(?:[^\\]]|\\]\\])+" false false)) "]") (replace id "]]" "]"))
	(parser '("\"" (define id (regex "(?:[^\"]|\"\")+" false false)) "\"") (replace id "\"\"" "\"")))))
(define tsql_identifier (parser (or tsql_identifier_quoted tsql_identifier_unquoted)))
(define tsql_string (parser '((or (atom "N'" true) (atom "'" false))
	(define value (regex "(?:''|[^'])*" false false)) (atom "'" false false))
	(replace value "''" "'")))
(define tsql_column (parser (or
	(parser '((define alias tsql_identifier) "." (define name tsql_identifier)) '('get_column alias true name true))
	(parser (define name tsql_identifier) '('get_column nil true name true)))))
(define tsql_hex_literal (parser (define value sql_hex_literal)
	(list (quote sql_type_expr) value "VARBINARY" nil)))
(define tsql_literal (parser (or
	(parser (atom "NULL" true) (sql_null_literal)) tsql_hex_literal tsql_number tsql_string)))
/* Binary semantics are chosen from declared SQL contracts at the shared
compile boundary. Byte equality reuses the native collation-aware operator;
no SQL type wrapper survives into execution or storage. */
(define tsql_binary_type? (lambda (type)
	(has? '("BINARY" "VARBINARY" "IMAGE" "ROWVERSION" "TIMESTAMP") type)))
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
	(match name "NVARCHAR" "VARCHAR" "NCHAR" "CHAR" "BIT" "BOOLEAN" _ name)))
(define tsql_type_dimensions (parser (or
	(parser '("(" (atom "MAX" true) ")") (list -1))
	(parser '("(" (define precision tsql_int) "," (define scale tsql_int) ")") (list precision scale))
	(parser '("(" (define length tsql_int) ")") (list length))
	(parser empty (list)))))
/* The same type grammar is used for RPC declarations and expression casts. */
(define tsql_declared_type (parser '(
	(? (define namespace tsql_identifier) ".") (define type tsql_column_type)
) (if (or (nil? namespace) (equal?? namespace "sys")) type
		(error "unsupported type namespace: " namespace))))
(define tds_bind_declarations (lambda (_schema text)
	(if (equal? (strtrim text) "") (list)
		((parser (define declarations (+ (parser '(
			"@" (define name tsql_identifier_unquoted)
			(define type tsql_declared_type) (define dimensions tsql_type_dimensions)
		) (list (toLower name) (cons type (tsql_decl_dimensions type dimensions)))) ","))
			declarations "^(?:/\\*.*?\\*/|--[^\r\n]*(?:[\r\n]|$)|[\r\n\t ]+)+") text))))
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
(define tsql_decl_dimensions (lambda (type dimensions) (begin
	(define count_dimensions (count dimensions))
	(if (has? '("VARCHAR" "CHAR" "NVARCHAR" "NCHAR" "BINARY" "VARBINARY") type)
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
/* A cast keeps its syntax-declared SQL type until the shared expression
compiler consumes the (expr, type, collation) contract. Runtime values do not
select that type, and no arithmetic operation is needed merely to mark it. */

(define tsql_cast_expr (lambda (value type dimensions) (begin
	(define name (toUpper type))
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
		"VARCHAR" (tsql_cast_text_expr value dimensions false)
		"NVARCHAR" (tsql_cast_text_expr value dimensions true)
		_ (error "unsupported conversion to " name)))
	(list (quote sql_type_expr) formula name nil))))
/* Assignment contracts come from native column declarations at compile time.
Integer narrowing is an ordinary frontend cast. Existing native decimal scale
normalization remains in use; this guard rejects overflow, including carry from
rounding, before native insertion. Full precision-limit emulation is deferred. */
(define tsql_assignment_decimal (lambda (value threshold) (if (nil? value) nil (begin
	(define number (tsql_numeric_input value))
	(if (or (>= number threshold) (<= number (- 0 threshold))) (error "decimal conversion overflow") number)))))
(define tsql_assignment_expr (lambda (value declaration) (begin
	(define type (toUpper (declaration "RawType")))
	(if (has? '("ROWVERSION" "TIMESTAMP") type)
		(error "generated version column cannot be assigned explicitly")
		(has? '("INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT") type)
		(tsql_cast_expr value type '())
		(has? '("DECIMAL" "NUMERIC") type) (begin
			(define dimensions (declaration "Dimensions"))
			(define precision (if (empty_list? dimensions) 18 (car dimensions)))
			(define scale (if (< (count dimensions) 2) 0 (cadr dimensions)))
			(define threshold (- (simplify (concat "1e" (- precision scale)))
				(* 0.5 (simplify (concat "1e-" scale)))))
			(list (quote sql_type_expr) (list tsql_assignment_decimal value threshold) type nil))
		value))))
(define tsql_insert_declarations (lambda (database table_name columns) (begin
	/* Resolve each target exactly once. Duplicate targets can overwrite an
	already validated identity/default value in the native column vector. */
	(if (equal? (count columns) (count (merge_unique (list (map columns toLower)))))
		true (error "duplicate INSERT column"))
	(define declarations (get_schema database table_name))
	(map columns (lambda (name) (begin
		(define declaration (find declarations (lambda (column) (equal?? (column "Field") name)) nil))
		(if (nil? declaration) (error "unknown INSERT column: " name) declaration)))))))
(define tsql_cast_text_expr (lambda (value dimensions national) (begin
	/* Resolve the operand at the shared compile boundary, where its sources
	and SQL contract are known. A parser-local walk would lose column types. */
	(define length (if (empty_list? dimensions) 30 (car dimensions)))
	(list (quote sql_compile_expr) (lambda (infos) (begin
		(define info (car infos))
		(define formula (sql_info_formula info))
		(if (and (plain_literal_expr? formula) (sql_numeric_type? (sql_info_type info))
			(not (equal? length -1)) (> (strlen (string formula)) length))
			(error "numeric character conversion overflow") true)
		(sql_info (list (quote tsql_cast_text) formula length national)
			(if national "NVARCHAR" "VARCHAR") (sql_info_collation info)))) value))))
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
(define tsql_convert_expr (lambda (value type dimensions style)
	(if (or (nil? style) (equal? style 0)) (tsql_cast_expr value type dimensions)
		(begin
			(tsql_decl_dimensions type dimensions)
			(match (list type style)
				'("DATETIME" 4) (list (quote from_unixtime) (list tsql_export_datetime_seconds value))
				'(text_type 120) (if (has? '("VARCHAR" "NVARCHAR") text_type)
					(tsql_cast_expr (list format_date value "%Y-%m-%d %H:%i:%s") text_type dimensions)
					(error "unsupported CONVERT style for " text_type))
				_ (error "unsupported CONVERT style: " style))))))
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
	(define parsed (parse_date anchored))
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
(define tsql_dateadd_expr (lambda (unit amount value) (begin
	(define unit (tsql_calendar_unit unit))
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
		(define type (if (has? '("DATE" "DATETIME" "DATETIME2") input_type) input_type "DATETIME"))
		(define minimum (if (equal? type "DATETIME") -6847804800 -62135596800))
		(define integer (list tsql_cast_integer (sql_info_formula amount_info) -2147483648 2147483647))
		(define formula (list (if months tsql_calendar_shift_months tsql_calendar_shift_seconds)
			(tsql_calendar_formula input) (list (quote *) integer (coalesceNil months seconds))))
		(sql_info (list tsql_calendar_range formula minimum) type nil))) amount value))))
(define tsql_datediff_expr (lambda (unit first last type) (begin
	(define boundary (tsql_boundary_calculator (tsql_calendar_unit unit)))
	(define minimum (if (equal? type "INT") -2147483648 (tsql_parse_integer "-9223372036854775808")))
	(define maximum (if (equal? type "INT") 2147483647 (tsql_parse_integer "9223372036854775807")))
	(list (quote sql_compile_expr) (lambda (infos) (begin
		(define difference (list (quote -)
			(list boundary (tsql_calendar_formula (cadr infos)))
			(list boundary (tsql_calendar_formula (car infos)))))
		(sql_info (list tsql_cast_integer difference minimum maximum) type nil))) first last))))
(define tsql_datepart_expr (lambda (unit value) (begin
	(define part (match (tsql_calendar_unit unit)
		"year" (sql_builtins "YEAR") "quarter" (sql_builtins "QUARTER") "month" (sql_builtins "MONTH")
		"day" (sql_builtins "DAY") "dayofyear" tsql_calendar_day_of_year "hour" (sql_builtins "HOUR")
		"minute" (sql_builtins "MINUTE") "second" (sql_builtins "SECOND")
		_ (error "DATEPART week and weekday require session first-day-of-week support")))
	(list (quote sql_compile_expr) (lambda (infos)
		(sql_info (list part (tsql_calendar_formula (car infos))) "INT" nil)) value))))


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
		(if (or (has? '("NULL" "any" "MONEY" "SMALLMONEY") type)
			(and (sql_numeric_type? type) (not (has? '("BOOLEAN" "BOOL" "BIT") type)))) true
			(error "unary minus requires a numeric operand"))
		(sql_info (list (quote -) 0 formula) (if (equal? type "TINYINT") "SMALLINT" type) nil))) value)))
(define tsql_key_placement (parser (? (or (atom "CLUSTERED" true) (atom "NONCLUSTERED" true)))))
(define tsql_key_columns (parser (define columns (+ (parser '(
	(define name tsql_identifier) (? (or (atom "ASC" true) (atom "DESC" true)))) name) ",")) columns))
(define tsql_default_literal (parser (or
	(parser '("(" (define value tsql_default_literal) ")") value)
	(parser (define value tsql_literal) (sql_type_lower_annotations value)))))
(define tsql_clock_default (parser (or
	(parser '("(" tsql_clock_default ")") true)
	'((atom "GETDATE" true) "(" ")") (atom "CURRENT_TIMESTAMP" true))))
(define tsql_column_attributes (parser (define attrs (* (or
	(parser '((? (atom "CONSTRAINT" true) (define key tsql_identifier)) (atom "PRIMARY" true) (atom "KEY" true) tsql_key_placement)
		(if key (list "primary" true "null" false "constraint_name" key) '("primary" true "null" false)))
	(parser '((? (atom "CONSTRAINT" true) (define key tsql_identifier)) (atom "UNIQUE" true) tsql_key_placement)
		(if key (list "unique" true "nulls_not_distinct" true "constraint_name" key) '("unique" true "nulls_not_distinct" true)))
	(parser '((atom "NOT" true) (atom "NULL" true)) '("null" false))
	(parser (atom "NULL" true) '("null" true))
	(parser '((atom "IDENTITY" true) (? "(" (define seed tsql_int) "," (define increment tsql_int) ")"))
		(if (and (or (nil? seed) (>= seed 1)) (or (nil? increment) (equal? increment 1)))
			(list "auto_increment" true "identity_seed" (coalesceNil seed 1))
			(error "IDENTITY currently requires a positive seed and increment 1")))
	(parser '((atom "DEFAULT" true) tsql_clock_default) '("default_expression" "CURRENT_TIMESTAMP"))
	(parser '((atom "DEFAULT" true) (define value tsql_default_literal)) '("default" value))
))) (begin
		(if (and (find attrs (lambda (attr) (attr "auto_increment")))
			(find attrs (lambda (attr) (attr "null"))))
			(error "IDENTITY column was declared NULL") true)
		(if (and (find attrs (lambda (attr) (attr "primary")))
			(find attrs (lambda (attr) (attr "null"))))
			(error "PRIMARY KEY column was declared NULL") true)
		(if (and (> (count (filter attrs (lambda (attr) (or (attr "primary") (attr "unique"))))) 1)
			(find attrs (lambda (attr) (not (nil? (attr "constraint_name"))))))
			(error "Multiple inline key declarations with constraint names are not supported") true)
		(merge attrs))))
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
/* Emit the existing native INSERT. VALUES and DEFAULT supply their bounded
syntax batch; the existing SELECT result sink supplies one emitted row. Never
scan/count/materialize SELECT input to reserve IDs, or rewind consumed blocks.
The frontend checks the complete reserved block before any row publication.
Explicit NULL must fail here: native auto_increment would otherwise allocate
an unchecked replacement. Scope/session identity reporting is separate work. */
(define tsql_native_insert_expr (lambda (database name columns values row_count) (begin
	(define relation (list (quote table) database name))
	(define identity (find (get_schema database name)
		(lambda (column) (equal? (column "Extra") "auto_increment")) nil))
	(define native_insert (lambda (names rows callback)
		(list (quote insert) relation (list (quote quote) names) rows
			'(list) nil false callback (quote tx))))
	(define callback '('lambda '('id) '('session "last_insert_id" 'id)))
	(if (nil? identity) (native_insert columns values callback)
		(equal? row_count 0) (native_insert columns values nil)
		(begin
			(define explicit (find (produceN (count columns))
				(lambda (index) (equal?? (nth columns index) (identity "Field"))) nil))
			(if (not (nil? explicit))
				'('begin
					'('define '__tsql_rows values)
					'('reduce '__tsql_rows
						'('lambda '('valid 'row)
							'('if '('nil? '('nth 'row explicit)) '('error "IDENTITY cannot be assigned NULL") 'valid)) true)
					(native_insert columns (quote __tsql_rows) callback))
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
						'('if '('> '__tsql_inserted 0) '('session "last_insert_id" '('car '__tsql_ids)) true)
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
(define tsql_fold_additive (lambda (acc term) (match term
	'("add" value) (tsql_add_expr acc value)
	_ (sql_fold_additive_term acc term))))
/* NULL fallback has the same declared precedence as other T-SQL expressions.
The shared lazy coalesce operator remains responsible for evaluation; this
callback consumes SQL triples and leaves only converted formulas in the plan.
Dimension merging and exact decimal precision are separate frontend work. */
(define tsql_coalesce_expr (lambda (arguments)
	(cons (quote sql_compile_expr) (cons (lambda (infos) (begin
		(if (< (count infos) 2) (error "COALESCE requires at least two arguments") true)
		(define types (map infos (lambda (info) (tsql_addition_type (sql_info_type info)))))
		(define active (filter types (lambda (type) (not (equal? type "NULL")))))
		(if (empty_list? active) (error "COALESCE requires a declared non-NULL type") true)
		(define target (coalesceNil
			(find (merge (list tsql_addition_numeric_types tsql_addition_text_types))
				(lambda (type) (has? active type)) nil)
			(if (reduce active (lambda (same type) (and same (equal? type (car active)))) true)
				(car active) (error "unsupported COALESCE type combination"))))
		(define formulas (map infos (lambda (info) (begin
			(define type (tsql_addition_type (sql_info_type info)))
			(define formula (sql_info_formula info))
			(if (or (equal? type "NULL") (equal? type target)) formula
				(and (has? tsql_addition_numeric_types target)
					(or (has? tsql_addition_numeric_types type) (has? tsql_addition_text_types type)))
				(if (has? '("DECIMAL" "MONEY" "SMALLMONEY") target)
					(list tsql_cast_float formula)
					(sql_type_lower_annotations (tsql_cast_expr formula target '())))
				(and (has? tsql_addition_text_types target) (has? tsql_addition_text_types type)) formula
				(error "unsupported implicit COALESCE conversion from " type " to " target))))))
		(sql_info (cons (quote coalesceNil) formulas) target
			(if (has? tsql_addition_text_types target)
				(reduce infos (lambda (collation info) (sql_merge_collation collation (sql_info_collation info))) nil)
				nil)))) arguments))))

/* Explicit wire descriptors come from the existing SQL expression metadata,
never from the first row. Unknown expression types stay text in this first slice. */
(define tsql_result_columns (lambda (query) (match query
	((symbol query-block) _schema sources fields _where _group _having _order _limit _offset _hidden _stages _facts)
	(extract_assoc (expand_query_block_fields sources fields) (lambda (name expression) (begin
		(define type (sql_info_type (sql_expr_info sources expression)))
		(define kind (if (has? '("INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT") type) 38
			(if (has? '("FLOAT" "DOUBLE" "REAL" "DECIMAL" "NUMERIC") type) 109
				(if (has? '("BOOLEAN" "BOOL" "BIT") type) 104 231))))
		(define size (if (equal? kind 231) 8000 (if (equal? kind 104) 1
			(match type "TINYINT" 1 "SMALLINT" 2 "INT" 4 "INTEGER" 4 _ 8))))
		(list "name" name "kind" kind "size" size "flags" 1 "sql_type" type))))
	((symbol union-block) _mode branches _order _limit _offset _facts) (tsql_result_columns (car branches))
	_ (error "unsupported T-SQL result metadata shape"))))

(define tsql_builtins (lambda (name) (match name
	"ISNULL" coalesceNil
	"LEN" (lambda (value) (if (nil? value) nil (strlen (sql_rtrim value))))
	_ (if (has? '("ABS" "CEILING" "FLOOR" "ROUND" "SQRT" "UPPER" "LOWER" "REPLACE" "SUBSTRING" "CONCAT" "YEAR" "MONTH" "DAY") name)
		(sql_builtins name) nil))))

(define parse_tsql (lambda (database s policy planning_session tx) (begin
	(define schema database)
	(define parse_started_ns (nanotime))
	(define frontend_options (if (nil? planning_session) (newsession) planning_session))
	(define describe (frontend_options "__tsql_describe"))
	(define parameters (coalesceNil (frontend_options "__tsql_parameter_declarations") (list)))
	(define parameter_expression (lambda (name) (begin
		(define name (toLower name))
		(define declaration (filter parameters (lambda (item) (equal? (car item) name))))
		(if (empty_list? declaration) (error "undeclared T-SQL parameter: @" name)
			(list (quote sql_type_expr) (list (quote session) (concat "__tsql_parameter:" name))
				(car (cadr (car declaration))) nil)))))

	(define placeholder_counter (newsession))
	(placeholder_counter "n" 0)

	(define extract_title_or_sql (lambda (captured_result)
		(match (car (cdr captured_result))
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
	) (reduce terms sql_fold_multiplicative_term a)))

	(define tsql_expression5 (parser (or
		(parser '((define expr tsql_expression6) (atom "IS" true) (atom "NULL" true)) '('nil? expr))
		(parser '((define expr tsql_expression6) (atom "IS" true) (atom "NOT" true) (atom "NULL" true)) '('not '('nil? expr)))
		tsql_expression6
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

	(define tsql_expression6 (parser (or

		(parser '("(" (define sub tsql_select) ")") '('inner_select sub))
		(parser '("(" (define a tsql_expression) ")") a)
		(parser '((atom "CAST" true) "(" (define value tsql_expression) (atom "AS" true)
			(define type tsql_declared_type) (define dimensions tsql_type_dimensions) ")")
			(tsql_cast_expr value type dimensions))
		(parser '((atom "CONVERT" true) "(" (define type tsql_declared_type) (define dimensions tsql_type_dimensions)
			"," (define value tsql_expression) (? "," (define style tsql_int)) ")")
			(tsql_convert_expr value type dimensions style))

		(parser '((atom "GETDATE" true) "(" ")")
			(list (quote sql_type_expr) (list (quote now)) "DATETIME" nil))
		(parser '((atom "DATEADD" true) "(" (define unit tsql_identifier) ","
			(define amount tsql_expression) "," (define value tsql_expression) ")")
			(tsql_dateadd_expr unit amount value))
		(parser '((define type (or (parser (atom "DATEDIFF" true) "INT") (parser (atom "DATEDIFF_BIG" true) "BIGINT")))
			"(" (define unit tsql_identifier) ","
			(define first tsql_expression) "," (define last tsql_expression) ")")
			(tsql_datediff_expr unit first last type))
		(parser '((atom "DATEPART" true) "(" (define unit tsql_identifier) "," (define value tsql_expression) ")")
			(tsql_datepart_expr unit value))

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

		(parser '((atom "@" true) (define var tsql_identifier_unquoted)) (parameter_expression var))

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
		tsql_string
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

	(define tabledef (parser (or

		(parser '((atom "(" true) (define query tsql_select) (atom ")" true) (atom "AS" true) (define id tsql_identifier) "(" (define aliases (+ tsql_identifier ",")) ")")
			(list id schema (tsql_apply_derived_column_aliases query aliases) false nil))
		(parser '((atom "(" true) (define query tsql_select) (atom ")" true) (define id tsql_identifier) "(" (define aliases (+ tsql_identifier ",")) ")")
			(list id schema (tsql_apply_derived_column_aliases query aliases) false nil))
		(parser '((atom "(" true) (define query tsql_select) (atom ")" true) (atom "AS" true) (define id tsql_identifier)) '(id schema query false nil))
		(parser '((atom "(" true) (define query tsql_select) (atom ")" true) (define id tsql_identifier)) '(id schema query false nil))

		(parser '((define schema tsql_schema_identifier) (atom "." true) (define tbl tsql_identifier) (atom "AS" true) (define id tsql_identifier)) (begin (if policy (policy schema tbl false) true) '(id schema tbl false nil)))
		(parser '((define schema tsql_schema_identifier) (atom "." true) (define tbl tsql_identifier) (define id tsql_identifier)) (begin (if policy (policy schema tbl false) true) '(id schema tbl false nil)))
		(parser '((define schema tsql_schema_identifier) (atom "." true) (define tbl tsql_identifier)) (begin (if policy (policy schema tbl false) true) '(tbl schema tbl false nil)))
		(parser '((define tbl tsql_identifier) (atom "AS" true) (define id tsql_identifier)) (begin (if policy (policy schema tbl false) true) '(id schema tbl false nil)))
		(parser '((define tbl tsql_identifier) (define id tsql_identifier)) (begin (if policy (policy schema tbl false) true) '(id schema tbl false nil)))
		(parser '((define tbl tsql_identifier)) (begin (if policy (policy schema tbl false) true) '(tbl schema tbl false nil)))
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

	(define tsql_select_core (parser '(
		(atom "SELECT" true)

		(define distinct (? (atom "DISTINCT" true)))
		(define prefix_limit tsql_select_prefix)
		(define cols (+ (or
			(parser "*" '("*" '((quote get_column) nil false "*" false)))
			(parser '((define tbl tsql_identifier_quoted) "." "*") '("*" '((quote get_column) tbl false "*" false)))
			(parser '((define tbl tsql_identifier_unquoted) "." "*") '("*" '((quote get_column) tbl false "*" false)))
			(parser '((define e tsql_expression) (atom "AS" true) (define title tsql_identifier)) '(title e))
			(parser '((define e tsql_expression) (atom "AS" true) (define title tsql_string)) '(title e))

			(parser '((define e tsql_expression) (define title tsql_identifier)) '(title e))

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

	(define tsql_update (parser '(
		(atom "UPDATE" true)
		(define tbldefs (+ tabledefs ","))
		(atom "SET" true)
		(define cols (+ (or
			(parser '(tsql_identifier "." (define title tsql_identifier) "=" (define e tsql_expression)) '(title e))
			(parser '((define title tsql_identifier) "=" (define e tsql_expression)) '(title e))
		) ","))
		(? '(
			(atom "WHERE" true)
			(define condition tsql_expression)
		))

	) (begin
			(define all_defs (merge tbldefs))
			(define first_def (car all_defs))
			(define tbl (match first_def '(_ _ t _ _) t))
			(define tblalias (match first_def '(id _ _ _ _) id))

			(define target_schema (source_schema first_def))
			(if policy (policy target_schema tbl true) true)
			(define declarations (get_schema target_schema tbl))
			(define assignments (merge (extract_assoc (merge cols) (lambda (title expression) (begin
				(define declaration (find declarations (lambda (column) (equal?? (column "Field") title)) nil))
				(if (nil? declaration) (error "unknown UPDATE column: " title)
					(equal? (declaration "Extra") "auto_increment") (error "IDENTITY column cannot be updated")
					(list (declaration "Field") (tsql_assignment_expr expression declaration))))))))
			(build_dml_plan target_schema tbl tblalias all_defs assignments (coalesceNil condition true) order limit offset planning_session tx)
	)))

	(define tsql_delete (parser '(
		(atom "DELETE" true)
		(atom "FROM" true)

		(? (define schema2 tsql_schema_identifier) ".") (define tbl tsql_identifier)
		(? '(
			(atom "WHERE" true)
			(define condition tsql_expression)
		))

	) (begin

			(if policy (policy (coalesce schema2 schema) tbl true) true)

			(define del_schema (coalesce schema2 schema))
			(define del_defs (list (list tbl del_schema tbl false nil)))
			(build_dml_plan del_schema tbl nil del_defs nil (coalesceNil condition true) order limit offset planning_session tx)
	)))

	(define tsql_truncate (parser '(
		(atom "TRUNCATE" true) (? (atom "TABLE" true))

		(? (define schema2 tsql_schema_identifier) ".") (define tbl tsql_identifier)
	) (begin
			(if policy (policy (coalesce schema2 schema) tbl true) true)
			(define trunc_schema (coalesce schema2 schema))
			(cons '!begin (list
				(list (quote checktablemaintenance) trunc_schema tbl "truncate")
				(build_dml_plan trunc_schema tbl nil (list (list tbl trunc_schema tbl false nil)) nil true nil nil nil planning_session tx)))
	)))

	(define tsql_insert_literal_cell (parser (or
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
		(atom "CREATE" true)
		(define replace (? (atom "OR" true) (atom "ALTER" true)))
		(atom "VIEW" true)
		(define ifnotexists (? (atom "IF" true) (atom "NOT" true) (atom "EXISTS" true)))
		(define target (or
			(parser '((define schema2 tsql_schema_identifier) "." (define id tsql_identifier)) '(schema2 id))
			(parser (define id tsql_identifier) '(nil id))))
		(define aliases (? (parser '("(" (define columns (+ tsql_identifier ",")) ")") columns)))
		(atom "AS" true)
		(define captured (capture tsql_select))
	) (match target '(schema2 id) (begin
			(define view_schema (coalesce schema2 schema))
			(if policy (policy view_schema id true) true)
			(define query (sql_apply_view_column_aliases (nth captured 1) aliases))
			(list (quote create_sql_view)
				(list (quote session) "__memcp_tx")
				view_schema id "tsql" (nth captured 0) (list (quote quote) query)
				(if replace "replace" (if ifnotexists "ignore" "error")))))))

	(define tsql_schema_identifier (parser (define owner tsql_identifier)
		(if (equal?? owner "dbo") database owner)))
	(define tsql_expression3 (parser '(
		(define a tsql_expression4)
		(define terms (* (parser '(
			(define op (or (parser "+" "add") (parser "-" "sub")))
			(define value tsql_expression4)
		) (list op value)) empty true))
	) (reduce terms tsql_fold_additive a)))
	(define tsql_build_select_plan (lambda (query) (begin
		(define expanded (sql_expand_views query policy))
		(if describe (list (quote quote) (tsql_result_columns expanded)) (list (quote !begin)
			(list (quote resultfields) (list (quote quote) (queryplan_result_titles expanded))
				(list (quote quote) (tsql_result_columns expanded)))
			(build_queryplan_term expanded planning_session tx))))))
	/* INSERT SELECT consumes the existing planner's result stream and the native
	insert operator. Identity allocation and constraints remain native. */
	(define tsql_insert_select_plan (lambda (database tbl columns query) (begin
		(define declarations (tsql_insert_declarations database tbl columns))
		(define columns (map declarations (lambda (column) (column "Field"))))
		(define insert_expr (tsql_native_insert_expr database tbl columns
			(cons list '((cons list (map (produceN (count columns))
				(lambda (i) (sql_type_formula '() (tsql_assignment_expr
					'('nth 'item (+ (* i 2) 1)) (nth declarations i)))))))) 1))
		(define expanded (sql_expand_views query policy))
		(if (equal? (count (queryplan_result_titles expanded)) (count columns)) true
			(error "INSERT column count does not match SELECT column count"))
		(define plan (build_queryplan_term expanded planning_session tx))
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
	(define tsql_insert_into (parser '(
		(atom "INSERT" true) (atom "INTO" true)
		(? (define schema2 tsql_schema_identifier) ".") (define tbl tsql_identifier)
		(? "(" (define columns (+ tsql_identifier ",")) ")")
		(atom "VALUES" true) (define rows (+ tsql_insert_values_row ","))
	) (begin
			(define database (coalesce schema2 schema))
			(if policy (policy database tbl true) true)
			(define declarations (tsql_insert_declarations database tbl
				(coalesce columns (table_insertable_columns database tbl))))
			(define columns (map declarations (lambda (column) (column "Field"))))
			(if (reduce rows (lambda (valid row) (and valid (equal? (count row) (count columns)))) true)
				true (error "INSERT column count does not match value count"))
			(if (reduce rows (lambda (found row) (or found (tsql_dataset_contains_inner_select row))) false)
				(tsql_insert_select_plan database tbl columns (tsql_values_to_select_query database columns rows))
				(tsql_native_insert_expr database tbl columns (sql_insert_values_expr (map rows (lambda (row) (map (produceN (count columns))
					(lambda (i) (sql_type_formula (list) (tsql_assignment_expr (nth row i) (nth declarations i)))))))) (count rows)))
	)))
	/* An all-default row is the native empty-column INSERT. Defaults and
	generated values are filled by the same operator as ordinary insertion. */
	(define tsql_insert_default (parser '(
		(atom "INSERT" true) (? (atom "INTO" true))
		(? (define schema2 tsql_schema_identifier) ".") (define tbl tsql_identifier)
		(atom "DEFAULT" true) (atom "VALUES" true)
	) (begin
			(define database (coalesce schema2 schema))
			(if policy (policy database tbl true) true)
			(tsql_native_insert_expr database tbl '() '('list '(list)) 1))))
	(define tsql_insert_select (parser '(
		(atom "INSERT" true) (atom "INTO" true)
		(? (define schema2 tsql_schema_identifier) ".") (define tbl tsql_identifier)
		(? "(" (define columns (+ tsql_identifier ",")) ")")
		(define query tsql_select)
	) (begin
			(define database (coalesce schema2 schema))
			(if policy (policy database tbl true) true)
			(tsql_insert_select_plan database tbl (coalesce columns (table_insertable_columns database tbl)) query)
	)))
	/* Unique index declarations are native keys. Placement/direction syntax
	is a physical hint; admission and enforcement stay in the existing operator. */
	(define tsql_create_unique_index (parser '(
		(atom "CREATE" true) (atom "UNIQUE" true) tsql_key_placement
		(atom "INDEX" true) (define name tsql_identifier) (atom "ON" true)
		(? (define schema2 tsql_schema_identifier) ".") (define tbl tsql_identifier)
		"(" (define columns tsql_key_columns) ")"
	) (begin
			(define database (coalesce schema2 schema))
			(if policy (policy database tbl true) true)
			/* PRIMARY is a reserved native key identity, not an index alias. */
			(if (equal? (toUpper name) "PRIMARY")
				(error "index name PRIMARY is reserved") true)
			(define declarations (get_schema database tbl))
			(define columns (map columns (lambda (name) (begin
				(define declaration (find declarations (lambda (column) (equal?? (column "Field") name)) nil))
				(if declaration (declaration "Field") (error "unknown index column: " name))))))
			(reduce columns (lambda (seen name)
				(if (has? seen name) (error "duplicate index column: " name) (cons name seen))) '())
			(list (quote if)
				(list (quote createkey) (list (quote table) database tbl) name true
					(cons (quote list) columns) (quote tx) (list (quote list) "nulls_not_distinct" true))
				true (list (quote error) "index already exists: " name)))))
	(define tsql_foreign_key_mode (parser (or
		(parser '((atom "NO" true) (atom "ACTION" true)) "restrict")
		(parser (atom "CASCADE" true) "cascade")
		(parser '((atom "SET" true) (atom "NULL" true)) "set null"))))
	(define column_definition (parser '(
		(define name tsql_identifier) (define type tsql_column_type)
		(define dimensions tsql_type_dimensions) (define attributes tsql_column_attributes)
	) (begin
			(define dimensions (tsql_decl_dimensions type dimensions))
			(list name (tsql_storage_type type) (cons (quote list) dimensions)
				(tsql_identity_attributes type dimensions attributes)))))
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
			(parser '((? (atom "CONSTRAINT" true) (define key tsql_identifier)) (atom "PRIMARY" true) (atom "KEY" true) tsql_key_placement "(" (define keys tsql_key_columns) ")")
				(list (quote list) "unique" "PRIMARY" (cons (quote list) keys)
					(cons (quote list) (if key (list "constraint_name" key) '()))))
			(parser '((? (atom "CONSTRAINT" true) (define key tsql_identifier)) (atom "UNIQUE" true) tsql_key_placement
				"(" (define keys tsql_key_columns) ")")
				(list (quote list) "unique" (coalesce key (reduce keys (lambda (a b) (concat a "_" b)) name)) (cons (quote list) keys)
					(list (quote list) "nulls_not_distinct" true)))
			foreign_key_definition
			(parser (define column column_definition) (match column '(name type dimensions attributes)
				(list (quote list) "column" name type dimensions (cons (quote list) attributes))))
		) ",")) ")"
	) (begin
			(if policy (policy (coalesce schema2 schema) name true) true)
			(define columns (tsql_table_primary_columns columns))
			(list (quote sql_create_table) (coalesce schema2 schema) name
				(cons (quote list) (map columns tsql_identity_native_column))
				(tsql_identity_table_options columns) (if ifnotexists true false) (quote tx)))))
	(define tsql_alter_table (parser '(
		(atom "ALTER" true) (atom "TABLE" true)
		(? (define schema2 tsql_schema_identifier) ".") (define name tsql_identifier)
		(define changes (+ (or
			(parser '((atom "ADD" true) (? (atom "CONSTRAINT" true) (define key tsql_identifier))
				(atom "PRIMARY" true) (atom "KEY" true) tsql_key_placement "(" (define columns tsql_key_columns) ")")
				(lambda (database name) (begin
					/* Nullable existing columns need native ALTER nullability support.
					Reject this shape rather than adding a parallel validation scan. */
					(if (reduce columns (lambda (valid column) (and valid
						(not ((find (show database name) (lambda (meta) (equal? (meta "Field") column))) "Null")))) true)
						true (error "ADD PRIMARY KEY currently requires NOT NULL columns"))
					(list (quote createkey) (list (quote table) database name) "PRIMARY" true
						(cons (quote list) columns) (quote tx)
						(cons (quote list) (if key (list "constraint_name" key) '()))))))
			(parser '((atom "ADD" true) (atom "CONSTRAINT" true) (define key tsql_identifier)
				(atom "UNIQUE" true) tsql_key_placement "(" (define columns tsql_key_columns) ")")
				(lambda (database name)
					(list (quote createkey) (list (quote table) database name) key true
						(cons (quote list) columns) (quote tx)
						(list (quote list) "nulls_not_distinct" true))))
			(parser '((atom "ADD" true) (define columns (+ column_definition ",")))
				(lambda (database name)
					(begin
						/* Native ADD has no identity backfill for historical rows.
						Reject this form instead of manufacturing a parallel allocator. */
						(if (find columns (lambda (column) ((nth column 3) "auto_increment")))
							(error "ALTER ADD IDENTITY requires native historical-row initialization") true)
						(cons (quote !begin) (map columns (lambda (column)
							(match column '(col type dimensions attributes)
								(list (quote sql_create_column) (list (quote table) database name)
									col type dimensions (cons (quote list) (sql_add_column_attributes attributes))))))))))
			(parser '((atom "DROP" true) (atom "COLUMN" true) (define col tsql_identifier))
				(lambda (database name)
					(list (quote altertable) (list (quote table) database name) "drop" col)))
			(parser '((atom "DROP" true) (atom "CONSTRAINT" true) (define key tsql_identifier))
				(lambda (database name) (list (quote or)
					(list (quote dropforeignkey) (list (quote table) database name) key)
					(list (quote dropkey) (list (quote table) database name) key))))
		) ","))
	) (begin
			(if policy (policy (coalesce schema2 schema) name true) true)
			(cons (quote !begin) (map changes (lambda (change) (change (coalesce schema2 schema) name)))))))
	(define p (parser (or
		(parser (define query tsql_select) (tsql_build_select_plan query))
		(parser '((atom "EXPLAIN" true) (atom "IR" true) (define query tsql_select)) (explain_queryplan_ir (sql_expand_views query policy)))
		(parser '((atom "EXPLAIN" true) (atom "REORDER" true) (define query tsql_select)) (explain_queryplan_reorder (sql_expand_views query policy) planning_session))
		(parser '((atom "EXPLAIN" true) (atom "COMPILE" true) (define query tsql_select)) (explain_queryplan_compile (sql_expand_views query policy) parse_started_ns (strlen s) planning_session))
		(parser '((atom "EXPLAIN" true) (atom "PHYSICAL" true) (define query tsql_select)) (explain_queryplan_physical (sql_expand_views query policy) planning_session))
		(parser '((atom "EXPLAIN" true) (define query tsql_select))
			'('resultrow '('list "code" (pretty_print (optimize (build_queryplan_term (sql_expand_views query policy) planning_session tx)) (settings "ExplainWidth")))))
		tsql_insert_into tsql_insert_default tsql_insert_select tsql_update tsql_delete tsql_truncate
		tsql_create_view tsql_create_table tsql_alter_table tsql_create_unique_index
		(parser '((atom "CREATE" true) (atom "DATABASE" true)
			(define ifnotexists (? (atom "IF" true) (atom "NOT" true) (atom "EXISTS" true)))
			(define name tsql_identifier)) (begin
				(if policy (policy "system" true true) true)
				(list (quote createdatabase) name (if ifnotexists true false) nil nil)))
		(parser '((atom "DROP" true) (atom "DATABASE" true)
			(define ifexists (? (atom "IF" true) (atom "EXISTS" true))) (define name tsql_identifier))
			(begin (if policy (policy "system" true true) true)
				(list (quote drop_sql_database) (list (quote session) "__memcp_tx") name (if ifexists true false))))
		(parser '((atom "DROP" true) (atom "TABLE" true)
			(define ifexists (? (atom "IF" true) (atom "EXISTS" true)))
			(? (define schema2 tsql_schema_identifier) ".") (define name tsql_identifier))
			(begin (if policy (policy (coalesce schema2 schema) name true) true)
				(list (quote droptable) (coalesce schema2 schema) name (if ifexists true false))))
		(parser '((atom "DROP" true) (atom "VIEW" true)
			(define ifexists (? (atom "IF" true) (atom "EXISTS" true)))
			(? (define schema2 tsql_schema_identifier) ".") (define name tsql_identifier))
			(begin (if policy (policy (coalesce schema2 schema) name true) true)
				(list (quote drop_sql_view) (list (quote session) "__memcp_tx")
					(coalesce schema2 schema) name (if ifexists true false))))
		(parser '((atom "SET" true) (atom "NOCOUNT" true)
			(define enabled (or (parser (atom "ON" true) true) (parser (atom "OFF" true) false))))
			(list (quote session) "__tsql_nocount" enabled))
		/* These ON modes are the frontend's fixed comparison and lexical rules.
		Accept the declarations without adding state or branches to row execution;
		OFF and other settings remain explicit unsupported syntax. */
		(parser '((atom "SET" true) (or (atom "ANSI_NULLS" true) (atom "QUOTED_IDENTIFIER" true))
			(atom "ON" true)) (list (quote quote) nil))
		(parser '((atom "USE" true) (define name tsql_identifier)) (begin
			(if policy (policy name true false) true)
			(list (quote session) "schema" name)))
		(parser '((atom "BEGIN" true) (? (or (atom "TRAN" true) (atom "TRANSACTION" true))))
			(list (quote tx_begin) (quote session) (quote tx)))
		(parser '((atom "COMMIT" true) (? (or (atom "TRAN" true) (atom "TRANSACTION" true))))
			(list (quote tx_commit) (quote session)))
		(parser '((atom "ROLLBACK" true) (? (or (atom "TRAN" true) (atom "TRANSACTION" true))))
			(list (quote tx_rollback) (quote session)))
		""
	)))

	(define command_parser (if describe
		(parser (define query tsql_select) (tsql_build_select_plan query)) p))
	((parser '((* ";") (define command command_parser) (* ";")) command "^(?:/\\*(?s:.*?)\\*/|--[^\r\n]*[\r\n]|--[^\r\n]*$|[\r\n\t ]+)+") s)
)))
