/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* SQL type and collation resolution.

MemCP is dynamically typed: the default type is "any" and the default collation
is nil (no text-ordering constraint). Most expressions keep both. Stronger
assertions are derived only where an operation guarantees a result type
(arithmetic on known numerics, CAST, comparisons, function contracts) or where a
resolved column carries catalog metadata.

Resolution runs as one bottom-up walk over the already-planned query (sources
resolved), producing a (FORMULA TYPE COLLATION) descriptor per expression. The
FORMULA slot is always plain executable code: no sql_typed_value / sql_collation
wrapper nodes are ever emitted. TYPE/COLLATION are consumed on the spot (collate
callback selection for comparisons, ORDER BY, GROUP BY, DISTINCT) or recorded in
the query block's result-types fact for the wire/PDO result contract. Nothing
type-related survives into the executable plan.

A descriptor is (FORMULA TYPE COLLATION):
 - TYPE: uppercase SQL type name, or "any".
 - COLLATION: nil, or a (NAME COERCIBILITY) pair. Lower coercibility wins:
   0 = explicit COLLATE, 2 = column, 4 = literal, 5 = numeric/ignorable. */

(define sql_info (lambda (formula type collation) (list formula type collation)))
(define sql_info_formula car)
(define sql_info_type cadr)
(define sql_info_collation (lambda (info) (nth info 2)))

/* ---- type predicates ---- */

(define sql_text_type? (lambda (type)
	(has? '("CHAR" "VARCHAR" "TEXT" "TINYTEXT" "MEDIUMTEXT" "LONGTEXT"
		"ENUM" "SET" "BINARY" "VARBINARY" "BLOB" "TINYBLOB" "MEDIUMBLOB" "LONGBLOB") type)))

(define sql_numeric_type? (lambda (type)
	(has? '("BOOL" "BOOLEAN" "BIT" "INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT"
		"MEDIUMINT" "DECIMAL" "NUMERIC" "FLOAT" "DOUBLE" "REAL") type)))

(define sql_temporal_type? (lambda (type)
	(has? '("DATE" "DATETIME" "TIMESTAMP" "TIME" "YEAR") type)))

/* ---- collation coercion ---- */

(define sql_collation_name (lambda (info)
	(if (nil? (sql_info_collation info)) "bin" (car (sql_info_collation info)))))

/* MySQL collation coercion. nil is maximally ignorable and never wins. Equal
coercibility with different names is an error unless one side is a bytewise or
unicode alias. Returns nil or a (NAME COERCIBILITY) pair. */
(define sql_merge_collation (lambda (a b)
	(if (nil? a) b (if (nil? b) a
		(if (equal? (car a) (car b)) (list (car a) (min (cadr a) (cadr b)))
			(if (< (cadr a) (cadr b)) a (if (> (cadr a) (cadr b)) b
				(if (equal? (cadr a) 0) (error "Illegal mix of explicit collations")
					(if (or (has? '("bin" "binary" "utf8" "utf8mb4") (car a)) (strlike (car a) "%_bin")) a
						(if (or (has? '("bin" "binary" "utf8" "utf8mb4") (car b)) (strlike (car b) "%_bin")) b
							(error "Illegal mix of collations")))))))))))

/* ---- type merge for CASE / COALESCE / UNION branches ---- */

(define sql_merge_type (lambda (ta tb)
	(if (equal? ta "NULL") tb
	(if (equal? tb "NULL") ta
	(if (equal? ta "any") "any"
	(if (equal? tb "any") "any"
	(if (equal? ta tb) ta
	(if (or (sql_text_type? ta) (sql_text_type? tb)) "VARCHAR"
	(if (or (equal? ta "DOUBLE") (equal? tb "DOUBLE")) "DOUBLE"
	(if (and (sql_numeric_type? ta) (sql_numeric_type? tb)) "DECIMAL" "VARCHAR"))))))))))

(define sql_merge_info (lambda (a b)
	(sql_info nil
		(sql_merge_type (sql_info_type a) (sql_info_type b))
		(sql_merge_collation (sql_info_collation a) (sql_info_collation b)))))
(define sql_merge_infos (lambda (infos)
	(reduce infos sql_merge_info (sql_info nil "NULL" nil))))

/* ---- arithmetic type fusion ---- */

/* Result type for a binary +,-,*,/,%,DIV. Unknown in, unknown out: any "any"
operand yields "any" so the runtime keeps its dynamic coercion. Division widens
to DECIMAL (MySQL). Temporal/mixed arithmetic stays "any" for now. */
(define sql_type_fuse_arith (lambda (op a b)
	(if (or (equal? a "any") (equal? b "any")) "any"
		(if (equal? a "NULL") "NULL"
			(if (equal? b "NULL") "NULL"
				(if (and (sql_numeric_type? a) (sql_numeric_type? b))
					(if (or (equal? op "divide") (equal? op "/")) "DECIMAL"
						(if (or (equal? a "DOUBLE") (equal? b "DOUBLE") (equal? a "REAL") (equal? b "REAL") (equal? a "FLOAT") (equal? b "FLOAT")) "DOUBLE"
							(if (or (equal? a "DECIMAL") (equal? b "DECIMAL") (equal? a "NUMERIC") (equal? b "NUMERIC")) "DECIMAL"
								"BIGINT")))
					"any"))))))

/* Static type of a scalar literal already produced by a leaf rule. */
(define sql_type_of_literal (lambda (v)
	(if (nil? v) "NULL"
		(if (string? v) "VARCHAR"
			(if (or (equal? v true) (equal? v false)) "BOOLEAN"
				(if (number? v) (if (equal? v (floor v)) "BIGINT" "DOUBLE")
					"any"))))))

/* Type of a request-local bind value (positional ? / @var). Recorded as a plan
cache guard by the caller; no captured value enters the result contract. */
(define sql_runtime_value_type (lambda (value)
	(if (nil? value) "NULL"
		(if (int? value) "BIGINT"
			(if (number? value) "DOUBLE"
				(if (string? value) "VARCHAR"
					(if (or (equal? value true) (equal? value false)) "BOOLEAN" "any")))))))

/* ---- function return contracts ----

Each entry is (HEAD TYPE MODE):
 - "fixed"  : result TYPE regardless of arguments
 - "first"  : result TYPE, collation inherited from the first argument
 - "text"   : result TYPE, collation merged from all arguments
 - "merge"  : result type and collation merged from all arguments (COALESCE, …)
 - "case"   : like "merge" but over the value arms only (IF/searched CASE) */

(define sql_core_function_rules (list
	(list (quote sql_compare) "BOOLEAN" "fixed")
	(list (quote count) "BIGINT" "fixed")
	(list (quote if) "NULL" "case")
	(list if "NULL" "case")
	(list (quote coalesceNil) "NULL" "merge")
	(list coalesceNil "NULL" "merge")
	(list (quote coalesce) "NULL" "merge")
	(list coalesce "NULL" "merge")
	(list (quote min) "NULL" "merge")
	(list min "NULL" "merge")
	(list (quote max) "NULL" "merge")
	(list max "NULL" "merge")
	(list (quote sql_concat) "VARCHAR" "text")
	(list sql_concat "VARCHAR" "text")
	(list (quote concat) "VARCHAR" "text")
	(list concat "VARCHAR" "text")
	(list (quote sql_substr) "VARCHAR" "first")
	(list sql_substr "VARCHAR" "first")
	(list (quote sql_trim) "VARCHAR" "first")
	(list sql_trim "VARCHAR" "first")
	(list (quote sql_ltrim) "VARCHAR" "first")
	(list sql_ltrim "VARCHAR" "first")
	(list (quote sql_rtrim) "VARCHAR" "first")
	(list sql_rtrim "VARCHAR" "first")
	(list (quote toUpper) "VARCHAR" "first")
	(list toUpper "VARCHAR" "first")
	(list (quote toLower) "VARCHAR" "first")
	(list toLower "VARCHAR" "first")
	(list (quote replace) "VARCHAR" "first")
	(list replace "VARCHAR" "first")
	(list (quote substr) "VARCHAR" "first")
	(list substr "VARCHAR" "first")
	(list (quote string_repeat) "VARCHAR" "first")
	(list string_repeat "VARCHAR" "first")
	(list (quote simplify) "DOUBLE" "fixed")
	(list simplify "DOUBLE" "fixed")
	(list (quote floor) "DOUBLE" "fixed")
	(list floor "DOUBLE" "fixed")
	(list (quote ceil) "DOUBLE" "fixed")
	(list ceil "DOUBLE" "fixed")
	(list (quote round) "DOUBLE" "fixed")
	(list round "DOUBLE" "fixed")
	(list (quote sqrt) "DOUBLE" "fixed")
	(list sqrt "DOUBLE" "fixed")
	(list (quote sql_abs) "DOUBLE" "fixed")
	(list sql_abs "DOUBLE" "fixed")
	(list (quote sql_rand) "DOUBLE" "fixed")
	(list sql_rand "DOUBLE" "fixed")
	(list (quote strlen) "BIGINT" "fixed")
	(list strlen "BIGINT" "fixed")
	(list (quote intdiv) "BIGINT" "fixed")
	(list intdiv "BIGINT" "fixed")
	(list (quote count_distinct) "BIGINT" "fixed")
	(list count_distinct "BIGINT" "fixed")
	(list (quote datediff) "BIGINT" "fixed")
	(list datediff "BIGINT" "fixed")
	(list (quote timestampdiff) "BIGINT" "fixed")
	(list timestampdiff "BIGINT" "fixed")
	(list (quote equal??) "BOOLEAN" "fixed")
	(list equal?? "BOOLEAN" "fixed")
	(list (quote equal?) "BOOLEAN" "fixed")
	(list equal? "BOOLEAN" "fixed")
	(list (quote <) "BOOLEAN" "fixed")
	(list < "BOOLEAN" "fixed")
	(list (quote >) "BOOLEAN" "fixed")
	(list > "BOOLEAN" "fixed")
	(list (quote <=) "BOOLEAN" "fixed")
	(list <= "BOOLEAN" "fixed")
	(list (quote >=) "BOOLEAN" "fixed")
	(list >= "BOOLEAN" "fixed")
	(list (quote sql_not) "BOOLEAN" "fixed")
	(list sql_not "BOOLEAN" "fixed")
	(list (quote not) "BOOLEAN" "fixed")
	(list not "BOOLEAN" "fixed")
	(list (quote nil?) "BOOLEAN" "fixed")
	(list nil? "BOOLEAN" "fixed")
	(list (quote and) "BOOLEAN" "fixed")
	(list and "BOOLEAN" "fixed")
	(list (quote or) "BOOLEAN" "fixed")
	(list or "BOOLEAN" "fixed")
	(list (quote strlike) "BOOLEAN" "fixed")
	(list strlike "BOOLEAN" "fixed")
	(list (quote equal_collate) "BOOLEAN" "fixed")
	(list equal_collate "BOOLEAN" "fixed")
	(list (quote notequal_collate) "BOOLEAN" "fixed")
	(list notequal_collate "BOOLEAN" "fixed")
	(list (quote now) "TIMESTAMP" "fixed")
	(list now "TIMESTAMP" "fixed")
	(list (quote current_date) "DATE" "fixed")
	(list current_date "DATE" "fixed")
	(list (quote date_trunc_day) "DATE" "fixed")
	(list date_trunc_day "DATE" "fixed")
	(list (quote from_unixtime) "DATETIME" "fixed")
	(list from_unixtime "DATETIME" "fixed")
	(list (quote sql_temporal_output) "VARCHAR" "fixed")
	(list sql_temporal_output "VARCHAR" "fixed")
	(list (quote sql_avg_divide) "DECIMAL" "fixed")
))

(define sql_extra_function_names (list
	(list "ABS" "DOUBLE" "fixed") (list "CEIL" "DOUBLE" "fixed") (list "CEILING" "DOUBLE" "fixed")
	(list "CHAR_LENGTH" "BIGINT" "fixed") (list "CHARACTER_LENGTH" "BIGINT" "fixed")
	(list "CONNECTION_ID" "BIGINT" "fixed") (list "CONVERT_TZ" "DATETIME" "fixed")
	(list "CURRENT_DATE" "DATE" "fixed") (list "CURRENT_TIMESTAMP" "TIMESTAMP" "fixed")
	(list "CURRENT_USER" "VARCHAR" "fixed") (list "DATABASE" "VARCHAR" "fixed")
	(list "DATE" "DATE" "fixed") (list "DATEDIFF" "BIGINT" "fixed") (list "DATE_FORMAT" "VARCHAR" "fixed")
	(list "DAY" "BIGINT" "fixed") (list "DAYOFMONTH" "BIGINT" "fixed") (list "DAYOFWEEK" "BIGINT" "fixed")
	(list "FLOOR" "DOUBLE" "fixed") (list "FROM_UNIXTIME" "DATETIME" "fixed")
	(list "GREATEST" "VARCHAR" "merge") (list "HOUR" "BIGINT" "fixed") (list "ISNULL" "BIGINT" "fixed")
	(list "JSON_EXTRACT" "VARCHAR" "fixed") (list "JSON_UNQUOTE" "VARCHAR" "fixed")
	(list "JSON_LENGTH" "BIGINT" "fixed") (list "JSON_VALUE" "VARCHAR" "fixed")
	(list "LEAST" "VARCHAR" "merge") (list "LENGTH" "BIGINT" "fixed") (list "LOWER" "VARCHAR" "first")
	(list "MINUTE" "BIGINT" "fixed") (list "MONTH" "BIGINT" "fixed") (list "NOW" "TIMESTAMP" "fixed")
	(list "PASSWORD" "VARCHAR" "fixed") (list "QUARTER" "BIGINT" "fixed") (list "RAND" "DOUBLE" "fixed")
	(list "RANDOM" "DOUBLE" "fixed") (list "REGEXP_REPLACE" "VARCHAR" "first")
	(list "REGEXP_SUBSTR" "VARCHAR" "first") (list "REPEAT" "VARCHAR" "first") (list "REPLACE" "VARCHAR" "first")
	(list "ROUND" "DOUBLE" "fixed") (list "SECOND" "BIGINT" "fixed") (list "SESSION_USER" "VARCHAR" "fixed")
	(list "SOUNDEX" "VARCHAR" "fixed") (list "SQRT" "DOUBLE" "fixed") (list "STR_TO_DATE" "DATETIME" "fixed")
	(list "SUBSTR" "VARCHAR" "first") (list "SUBSTRING" "VARCHAR" "first") (list "SYSDATE" "DATETIME" "fixed")
	(list "TIMESTAMPDIFF" "BIGINT" "fixed") (list "TO_TIMESTAMP" "DATETIME" "fixed")
	(list "UNIX_TIMESTAMP" "BIGINT" "fixed") (list "UPPER" "VARCHAR" "first") (list "USER" "VARCHAR" "fixed")
	(list "UTC_DATE" "DATE" "fixed") (list "UTC_TIME" "TIME" "fixed") (list "UTC_TIMESTAMP" "TIMESTAMP" "fixed")
	(list "WEEK" "BIGINT" "fixed") (list "WEEKDAY" "BIGINT" "fixed") (list "YEAR" "BIGINT" "fixed")
	(list "YEARWEEK" "BIGINT" "fixed") (list "SQL_NULL" "NULL" "fixed")
))

(define sql_cached_function_rules (once (lambda ()
	(merge (list sql_core_function_rules
		(map sql_extra_function_names (lambda (entry)
			(list (coalesceNil (sql_builtins (car entry)) (car entry)) (cadr entry) (nth entry 2)))))))))

(define sql_function_rules (lambda ()
	(if (or (nil? sql_builtins) (symbol? sql_builtins)) sql_core_function_rules (sql_cached_function_rules))))

/* Return contract for a call head. Unknown heads default to (HEAD "any" "fixed"). */
(define sql_function_rule (lambda (head)
	(coalesceNil (find (sql_function_rules) (lambda (entry)
		(or (equal? head (car entry))
			(and (symbol? head) (equal? (string head) (string (car entry)))))) nil)
		(list head "any" "fixed"))))
