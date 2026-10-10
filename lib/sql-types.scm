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

A descriptor is (FORMULA TYPE COLLATION), optionally followed by the frontend declaration:
- TYPE: uppercase SQL type name, or "any".
- COLLATION: nil, or a (NAME COERCIBILITY) pair. Lower coercibility wins:
0 = explicit COLLATE, 2 = column, 4 = literal, 5 = numeric/ignorable. */

(define sql_info (lambda (formula type collation) (list formula type collation)))
(define sql_info_formula car)
(define sql_info_type cadr)
(define sql_info_collation (lambda (info) (nth info 2)))
/* A frontend declaration is immutable planning data, never a value tag. */
(define sql_info_declaration (lambda (info) (if (> (count info) 3) (nth info 3) nil)))
(define sql_declared_info (lambda (formula declaration collation)
	(list formula (car declaration) collation declaration)))
(define sql_info_spec (lambda (info)
	(coalesceNil (sql_info_declaration info) (list (sql_info_type info)))))

/* ---- type predicates ---- */

(define sql_text_type? (lambda (type)
	(has? '("CHAR" "VARCHAR" "TEXT" "TINYTEXT" "MEDIUMTEXT" "LONGTEXT"
		"NVARCHAR" "NCHAR" "NTEXT" "ENUM" "SET" "BINARY" "VARBINARY" "BLOB" "TINYBLOB" "MEDIUMBLOB" "LONGBLOB") type)))

(define sql_numeric_type? (lambda (type)
	(has? '("BOOL" "BOOLEAN" "BIT" "INT" "INTEGER" "BIGINT" "SMALLINT" "TINYINT"
		"MEDIUMINT" "DECIMAL" "NUMERIC" "MONEY" "SMALLMONEY" "FLOAT" "DOUBLE" "REAL") type)))

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
			(if (number? v) (if (equal? v (floor v)) "BIGINT" "DOUBLE")
				(if (or (equal? v true) (equal? v false)) "BOOLEAN" "any"))))))

/* Request values carry no SQL tags. Frontends pass declarations separately,
including for NULL, and include that full declaration in their cache guards. */
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

/* ---- expression type/collation resolution walk ----

sql_expr_info walks one expression bottom-up. For every (get_column ...) it
resolves the physical column name against the source catalog (superseding the
per-use resolution the physical lowerer does today) and reads its declared type
and collation. Enclosing operators fuse their operands. The returned formula is
always plain and canonical; type/collation ride alongside for the consumer. */

(define sql_arith_head? (lambda (head)
	(has? (list (quote +) (quote -) (quote *) (quote /) (quote intdiv)) head)))
(define sql_arith_op_name (lambda (head)
	(if (equal? head (quote /)) "divide" (if (equal? head (quote intdiv)) "intdiv" "add"))))
(define sql_comparison_head? (lambda (head)
	(has? (list (quote equal??) (quote equal?) (quote <) (quote >) (quote <=) (quote >=)) head)))

/* Catalog type/collation for a base-table column, name canonicalized. Only a
resolved base table is rewritten: derived-table / stage-output columns and any
column whose name does not resolve are returned verbatim so downstream lowering
keeps its existing (ignorecase-flagged) resolution for them. */
(define sql_source_column_info (lambda (src original tblvar col col_ignorecase bindings)
	(if (not (source_is_base_table? src))
		(begin
			(define relation (source_relation src))
			(define staged (if (and bindings (stage_output_relation? relation)) (bindings (stage_output_relation_id relation) col) nil))
			(if staged (list original (sql_info_type staged) (sql_info_collation staged) (sql_info_declaration staged))
				(begin
					(define derived (if (and bindings (or (query_block? relation) (union_block? relation))) (sql_type_node relation bindings) nil))
					(define fields (if derived (qassoc_get (if (query_block? derived) (qb_facts derived) (union_facts derived)) 'result-types '()) '()))
					(define descriptor (reduce_assoc fields (lambda (found name value) (if (equal?? name col) value found)) nil))
					(if descriptor (list original (sql_info_type descriptor) (sql_info_collation descriptor) (sql_info_declaration descriptor))
						(begin
							(define description (if (or (string? relation) (stage_output_relation? relation)) nil
								(find (tsql_query_descriptions relation nil) (lambda (field) (equal?? (field "name") col)) nil)))
							(if description (sql_declared_info original (tsql_description_spec description) nil) (sql_info original "any" nil)))))))
		(begin
			(define canonical (source_column_name src col col_ignorecase))
			(if (nil? canonical)
				(sql_info original "any" nil)
				(begin
					(define meta (find (get_schema (source_schema src) (source_relation src))
						(lambda (c) (equal?? (c "Field") canonical)) nil))
					(define declaration (if (nil? meta) nil (tsql_column_spec meta)))
					(define type (if declaration (car declaration) (if (nil? meta) "any" (toUpper (coalesceNil (meta "RawType") "any")))))
					(define collname (if (nil? meta) nil (meta "Collation")))
					(define column_formula (list (quote get_column) (source_alias src) false canonical false))
					(define formula column_formula)
					(define collation (if (and (sql_text_type? type) (string? collname) (not (equal? collname ""))) (list collname 2) nil))
					(if declaration (sql_declared_info formula declaration collation) (sql_info formula type collation))))))))

(define sql_get_column_info (lambda (sources tblvar tbl_ic col col_ic bindings)
	(begin
		(define original (list (quote get_column) tblvar tbl_ic col col_ic))
		(define default_alias (if (empty_list? sources) nil (source_alias (car sources))))
		(define src (source_for_alias sources default_alias tblvar tbl_ic))
		(if (nil? src)
			(sql_info original "any" nil)
			(sql_source_column_info src original tblvar col col_ic bindings)))))

/* Comparison: BOOLEAN. The formula stays plain (canonical operands); the merged
operand collation rides in the info slot so a consumer can pick the Less
relation for scan bounds / ORDER without re-deriving it. */
(define sql_comparison_info (lambda (head left_info right_info)
	(begin
		(define lt (sql_info_type left_info))
		(define rt (sql_info_type right_info))
		(define plain (list head (sql_info_formula left_info) (sql_info_formula right_info)))
		(if (and (sql_text_type? lt) (sql_text_type? rt))
			(sql_info plain "BOOLEAN"
				(sql_merge_collation (sql_info_collation left_info) (sql_info_collation right_info)))
			(sql_info plain "BOOLEAN" nil)))))

/* IF / searched CASE lowers to (if cond1 val1 cond2 val2 ... else). Only the
value arms (and the trailing else) contribute to the result type/collation;
the conditions are booleans and would otherwise pollute the merge. */
(define sql_case_value_infos (lambda (infos)
	(begin
		(define n (count infos))
		(map (filter (produceN n)
			(lambda (i) (or (equal? i (- n 1)) (equal? (- i (* 2 (intdiv i 2))) 1))))
			(lambda (i) (nth infos i))))))

(define sql_generic_call_info (lambda (sources head args dialect bindings)
	(begin
		(define infos (map args (lambda (a) (sql_expr_info sources a dialect bindings))))
		(define forms (map infos sql_info_formula))
		(if (sql_arith_head? head)
			(sql_info (cons head forms)
				(reduce (cdr infos) (lambda (t i) (sql_type_fuse_arith (sql_arith_op_name head) t (sql_info_type i)))
					(sql_info_type (car infos)))
				nil)
			(if (and (sql_comparison_head? head) (equal? (count infos) 2))
				(sql_comparison_info head (car infos) (cadr infos))
				(begin
					(define rule (sql_function_rule head))
					(define rtype (cadr rule))
					(define mode (nth rule 2))
					(define value_infos (if (equal? mode "case") (sql_case_value_infos infos) infos))
					(define merged (sql_merge_infos value_infos))
					(define type (if (has? (list "merge" "case") mode) (sql_info_type merged) rtype))
					/* fixed: the function owns its result, no operand collation inherited.
					first: inherit from the first argument. text/merge/case: merge the
					value operands. */
					(define collation (if (not (sql_text_type? type)) nil
						(if (equal? mode "first")
							(if (empty_list? infos) nil (sql_info_collation (car infos)))
							(if (has? (list "text" "merge" "case") mode)
								(sql_info_collation merged)
								nil))))
					(sql_info (cons head forms) type collation)))))))

(define sql_call_info (lambda (sources head args dialect bindings)
	(if (or dialect (and (symbol? head) (strlike (string head) "tsql_%")))
		(begin (define bound (tsql_bind_expression sources head args bindings))
			(if (nil? bound) (sql_generic_call_info sources head args dialect bindings) bound))
		(sql_generic_call_info sources head args dialect bindings))))

(define sql_expr_info (lambda (sources expr dialect bindings)
	(match expr
		((symbol get_column) tblvar tbl_ic col col_ic) (sql_get_column_info sources tblvar tbl_ic col col_ic bindings)
		((quote get_column) tblvar tbl_ic col col_ic) (sql_get_column_info sources tblvar tbl_ic col col_ic bindings)
		((symbol quote) _datum) (sql_info expr "any" nil)
		((symbol session) _key) (sql_info expr "any" nil)
		((symbol session_globalvar) _key) (sql_info expr "any" nil)
		((symbol lambda) _params _body) (sql_info expr "any" nil)
		(cons head args) (sql_call_info sources head args dialect bindings)
		_ (sql_info expr (sql_type_of_literal expr)
			(if (string? expr) (list "utf8mb4_general_ci" 4) nil)))))

/* NULL-safe comparison primitive emitted by sql_comparison_info. The Less
relation and operator string are resolved at plan time. */
(define sql_compare (lambda (left right less operator collation)
	(if (or (nil? left) (nil? right)) nil
		(if (equal? operator "equal??") (and (not (less left right)) (not (less right left)))
			(if (equal? operator "equal?") (and (not (less left right)) (not (less right left)))
				(if (equal? operator "<") (less left right)
					(if (equal? operator ">") (less right left)
						(if (equal? operator "<=") (not (less right left))
							(if (equal? operator ">=") (not (less left right))
								(error "unsupported SQL comparison"))))))))))

/* ---- query-block level type resolution ----

sql_type_query_block walks every expression position of a query block through
sql_expr_info: it canonicalizes get_column against the sources and records the
projected column types/collations as the result-types fact. Formulas stay plain.
Derived-table sources and unions are passed through unchanged for now
(base-table columns are the immediate goal). */

(define sql_column_ref? (lambda (expr)
	(match expr
		((symbol get_column) alias alias_ci col col_ci) (or alias alias_ci col col_ci true)
		((quote get_column) alias alias_ci col col_ci) (or alias alias_ci col col_ci true)
		_ false)))

/* Plain canonical formula for one expression against the resolved sources. */
(define sql_type_formula (lambda (sources expr dialect bindings) (sql_info_formula (sql_expr_info sources expr dialect bindings))))

/* Result-column descriptor (nil TYPE COLLATION) for one expression. */
(define sql_type_result_descriptor (lambda (sources expr dialect bindings)
	(begin
		(define info (sql_expr_info sources expr dialect bindings))
		(if (sql_info_declaration info) (sql_declared_info nil (sql_info_declaration info) (sql_info_collation info))
			(sql_info nil (sql_info_type info) (sql_info_collation info))))))

/* Canonicalize a projected field, but leave a SELECT * / t.* entry as-is: the
planner has its own star-expansion (and GROUP-BY-primary-key rules) downstream. */
(define sql_type_field_formula (lambda (sources expr dialect bindings)
	(if (star_expr? expr) expr (sql_type_formula sources expr dialect bindings))))

/* Resolve the ORDER direction of a text-valued *expression* to its collation
callback. Bare column keys keep their raw < / > so the existing native
scan-order path (order_relations_for_source) is untouched; only computed keys
(UPPER(x), CONCAT(a, b), x COLLATE y) — which the old physical_expr_collation
always treated as "bin" — get the coercing relation. */
(define sql_type_order_item (lambda (sources item dialect bindings)
	(match item
		'(order_expr order_dir) (begin
			(define info (sql_expr_info sources order_expr dialect bindings))
			(define coll (sql_info_collation info))
			(if (and (or (equal? order_dir <) (equal? order_dir >))
				(not (sql_column_ref? (sql_info_formula info)))
				(sql_text_type? (sql_info_type info))
				(not (nil? coll)))
				(list (sql_info_formula info) (collate (car coll) (equal? order_dir >)))
				(list (sql_info_formula info) order_dir)))
		_ item)))

(define sql_type_query_block (lambda (block outer_sources bindings)
	(if (not (query_block? block))
		block
		(begin
			(define sources (qb_sources block))
			(define dialect (or (not (nil? bindings)) (equal? (qassoc_get (qb_facts block) 'frontend nil) "tsql")))
			(define outer (coalesceNil outer_sources '()))
			(define all_sources (if (empty_list? outer) sources (merge (list sources outer))))
			(make_query_block
				(qb_schema block)
				sources
				(map_assoc (qb_fields block)
					(lambda (field_title field_expr) (sql_type_field_formula all_sources field_expr dialect bindings)))
				(sql_type_formula all_sources (qb_where block) dialect bindings)
				(map (coalesceNil (qb_group block) '())
					(lambda (group_expr) (sql_type_formula all_sources group_expr dialect bindings)))
				(if (nil? (qb_having block)) nil (sql_type_formula all_sources (qb_having block) dialect bindings))
				(map (coalesceNil (qb_order block) '())
					(lambda (order_item) (sql_type_order_item all_sources order_item dialect bindings)))
				(qb_limit block) (qb_offset block) (qb_hidden block) (qb_stages block)
				(qassoc_set (qb_facts block) (quote result-types)
					(map_assoc (expand_query_block_fields sources (qb_fields block))
						(lambda (rt_title rt_expr) (sql_type_result_descriptor all_sources rt_expr dialect bindings)))))))))

/* Stage descriptors live in the existing immutable facts. Their lookup is
compiler-local; it retains only the logical program and declared metadata. */
(define sql_type_stage_column (lambda (stages id column path) (begin
	(if (has? path id) (error "cyclic stage type dependency") true)
	(define stage (find stages (lambda (candidate) (and (group_stage? candidate) (equal? (gs_id candidate) id))) nil))
	(if (nil? stage) nil (begin
		(define nested (lambda (other name) (sql_type_stage_column stages other name (append path id))))
		(define sources (canonical_helper_sources (gs_input stage)))
		(define key (find (mapIndex (gs_keys stage) (lambda (i expression) (list (group_key_col_name i) expression)))
			(lambda (item) (equal?? (car item) column)) nil))
		(define aggregate (find (gs_aggregates stage) (lambda (descriptor) (equal? (aggregate_col_name_using (gs_input stage) descriptor) column)) nil))
		(if key (sql_type_result_descriptor sources (cadr key) true nested)
			(if aggregate (sql_type_result_descriptor sources (cons 'aggregate aggregate) true nested)
				(begin
					(define projection (reduce_assoc (gs_output stage) (lambda (found name expression)
						(if (equal?? name column) (list expression) found)) nil))
					(if projection (sql_type_result_descriptor sources (car projection) true nested) nil)))))))))
(define sql_type_node (lambda (node bindings) (if (query_block? node) (begin
	(define sources (map (qb_sources node) (lambda (source)
		(if (or (query_block? (source_relation source)) (union_block? (source_relation source)))
			(source_with_relation source (sql_type_node (source_relation source) bindings)) source))))
	(sql_type_query_block (make_query_block (qb_schema node) sources (qb_fields node) (qb_where node) (qb_group node) (qb_having node)
		(qb_order node) (qb_limit node) (qb_offset node) (qb_hidden node) (qb_stages node) (qb_facts node)) '() bindings))
	(if (union_block? node) (begin
		(define branches (map (union_branches node) (lambda (branch) (sql_type_node branch bindings))))
		(define first (if (empty_list? branches) nil (car branches)))
		(define fields (if first (qassoc_get (if (query_block? first) (qb_facts first) (union_facts first)) 'result-types '()) '()))
		(make_union_block (union_mode node) branches (union_order node) (union_limit node) (union_offset node)
			(qassoc_set (union_facts node) 'result-types fields))) node))))
(define sql_type_tsql_node? (lambda (node)
	(if (query_block? node) (equal? (qassoc_get (qb_facts node) 'frontend nil) "tsql")
		(if (union_block? node) (reduce (union_branches node) (lambda (found branch) (or found (sql_type_tsql_node? branch))) false) false))))
(define sql_type_stage_facts (lambda (stage stages bindings)
	(if (group_stage? stage) (begin
		(define sources (canonical_helper_sources (gs_input stage)))
		(define keys (merge (mapIndex (gs_keys stage) (lambda (i expression)
			(list (group_key_col_name i) (sql_type_result_descriptor sources expression true bindings))))))
		(define aggregates (merge (map (gs_aggregates stage) (lambda (descriptor)
			(list (aggregate_col_name_using (gs_input stage) descriptor)
				(sql_type_result_descriptor sources (cons 'aggregate descriptor) true bindings))))))
		(make_group_stage (gs_id stage) (sql_type_node (gs_input stage) bindings) (gs_domain stage) (gs_keys stage) (gs_aggregates stage)
			(gs_having stage) (gs_output stage) (gs_order stage) (gs_limit stage) (gs_offset stage)
			(qassoc_set (gs_facts stage) 'result-types (merge keys aggregates)))) stage)))
(define sql_type_annotate_ir (lambda (ir)
	(if (sql_type_tsql_node? (ir_root ir)) (begin
		(define stages (ir_stages ir))
		(define bindings (lambda (id column) (sql_type_stage_column stages id column '())))
		(make_ir (ir_kind ir) (sql_type_node (ir_root ir) bindings)
			(map stages (lambda (stage) (sql_type_stage_facts stage stages bindings))) (ir_context_of ir) (ir_return ir)))
		(if (query_block? (ir_root ir))
			(make_ir (ir_kind ir) (sql_type_query_block (ir_root ir) '() nil) (ir_stages ir) (ir_context_of ir) (ir_return ir)) ir))))
