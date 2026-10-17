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
	(has? '("CHAR" "VARCHAR" "NCHAR" "NVARCHAR" "NTEXT" "TEXT" "TINYTEXT" "MEDIUMTEXT" "LONGTEXT"
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
			/* These are parser AST terminals, not runtime column values. Scalar
			equality also equates numeric zero/one with booleans; syntax identity
			must keep the frontend's numeric terminal contract intact. */
			(if (or (expression_equal? v true) (expression_equal? v false)) "BOOLEAN"
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
	/* Frontend text conversion owns its return type and inherits the input's
	declared collation. Neither comes from the converted runtime value. */
	(list (quote tsql_cast_text) "VARCHAR" "first")
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

/* Read a producer's declared projection contract, never its runtime values.
UNION can carry identical/NULL contracts without choosing dialect coercions;
heterogeneous branches require a frontend-assigned contract or casts. */
(define sql_projection_info_at (lambda (relation index outer_sources)
	(begin
		(define relation (normalize_query_ast relation))
		(if (query_block? relation) (begin
			(define fields (projection_exprs (expand_query_block_fields (qb_sources relation) (qb_fields relation))))
			(if (>= index (count fields)) (sql_info nil "any" nil)
				(sql_expr_info (merge (list (qb_sources relation) outer_sources)) (nth fields index))))
			(union_block? relation) (begin
				(define infos (map (union_branches relation) (lambda (branch) (sql_projection_info_at branch index outer_sources))))
				(define types (filter (map infos sql_info_type) (lambda (type) (not (equal? type "NULL")))))
				(define type (if (empty_list? types) "NULL" (car types)))
				(if (reduce types (lambda (same current) (and same (equal? current type))) true)
					(sql_info nil type (reduce infos (lambda (collation info)
						(sql_merge_collation collation (sql_info_collation info))) nil))
					(sql_info nil "any" nil)))
			(sql_info nil "any" nil)))))
/* A derived column keeps its producer's SQL contract and its own reference.
This resolves metadata only; the logical planner still owns projection rewrite. */
(define sql_projection_column_info (lambda (relation original col col_ignorecase outer_sources)
	(begin
		/* Type consumers can precede the binder. Normalize only the shared
		logical syntax node; no nested query is executed or planned here. */
		(define relation (normalize_query_ast relation))
		(define titles (queryplan_result_titles relation))
		(define index (find (produceN (count titles)) (lambda (index)
			(if col_ignorecase (equal?? (nth titles index) col) (equal? (nth titles index) col))) nil))
		(define info (if (nil? index) (sql_info nil "any" nil) (sql_projection_info_at relation index outer_sources)))
		(sql_info original (sql_info_type info) (sql_info_collation info)))))

/* Catalog type/collation for a base-table column, name canonicalized. Only a
resolved base table is rewritten: derived-table / stage-output columns and any
column whose name does not resolve are returned verbatim so downstream lowering
keeps its existing (ignorecase-flagged) resolution for them. */
(define sql_source_column_info (lambda (sources src original tblvar col col_ignorecase)
	(if (not (source_is_base_table? src))
		(sql_projection_column_info (source_relation src) original col col_ignorecase
			(if (source_lateral? src) (source_prefix_for_alias sources (source_alias src)) '()))
		(begin
			(define canonical (source_column_name src col col_ignorecase))
			(if (nil? canonical)
				(sql_info original "any" nil)
				(begin
					(define handle (table (source_schema src) false (source_relation src) false))
					(define meta (if (nil? handle) nil (show handle "column" canonical false)))
					(define type (if (nil? meta) "any" (toUpper (coalesceNil (meta "RawType") "any"))))
					(define collname (if (nil? meta) nil (meta "Collation")))
					(sql_info
						(list (quote get_column) (source_alias src) false canonical false)
						type
						(if (and (sql_text_type? type) (string? collname) (not (equal? collname "")))
							(list collname 2) nil))))))))

(define sql_get_column_info (lambda (sources tblvar tbl_ic col col_ic)
	(begin
		(define original (list (quote get_column) tblvar tbl_ic col col_ic))
		/* Metadata consumers can precede full query binding. Reuse the common
		column binder for unqualified joined inputs instead of assuming source 0. */
		(define reference (if (and (nil? tblvar) (not (single_source? sources)))
			(coalesceNil (try_bind_unqualified_column_in_scopes
				(list (binding_scope_entries sources '() "types" (list '() '() "types") 0)) col col_ic) original)
			original))
		(match reference
			((symbol get_column) alias alias_ic column column_ic) (begin
				(define default_alias (if (empty_list? sources) nil (source_alias (car sources))))
				(define src (source_for_alias sources default_alias alias alias_ic))
				(if (nil? src)
					(sql_info reference "any" nil)
					(sql_source_column_info sources src reference alias column column_ic)))))))

/* Frontend declaration resolution runs after the common name binder. Give
the compiler the existing immutable point description, not runtime values.
Derived columns already carry their producer's expression contract. */
(define sql_compile_column_info (lambda (sources compiler expression) (begin
	(define info (sql_expr_info sources expression))
	(define declaration (match (sql_info_formula info)
		((symbol get_column) alias alias_ic column column_ic) (begin
			(define src (source_for_alias sources
				(if (empty_list? sources) nil (source_alias (car sources))) alias alias_ic))
			(if (nil? src) nil (not (source_is_base_table? src)) nil (begin
				(define handle (table (source_schema src) false (source_relation src) false))
				(if (nil? handle) nil (show handle "column" column column_ic)))))
		_ nil))
	(define resolved (compiler info declaration))
	(if (expression_equal? (sql_info_formula info) (sql_info_formula resolved)) resolved
		(error "a column declaration compiler must preserve the column expression")))))

/* Comparison: BOOLEAN. Preserve plain operators for direct scan bounds;
computed text comparisons use the compile-time selected Less relation. The
merged collation remains in the info slot for other compiler consumers. */
(define sql_comparison_info (lambda (head left_info right_info)
	(begin
		(define lt (sql_info_type left_info))
		(define rt (sql_info_type right_info))
		(define plain (list head (sql_info_formula left_info) (sql_info_formula right_info)))
		(if (and (sql_text_type? lt) (sql_text_type? rt))
			(begin
				(define collation (sql_merge_collation (sql_info_collation left_info) (sql_info_collation right_info)))
				/* Direct column/literal comparisons retain their operator shape
				for storage bounds. Computed text needs the compiler-selected
				relation in its residual scalar comparison too. */
				(define computed? (lambda (info) (begin
					(define formula (sql_info_formula info))
					(and (not (sql_column_ref? formula)) (not (empty_list? (query_expr_alias_set "" formula '())))))))
				(sql_info (if (or (computed? left_info) (computed? right_info))
					(list (quote sql_compare) (sql_info_formula left_info) (sql_info_formula right_info)
						(collate (if (nil? collation) "bin" (car collation)) false) (string head))
					plain) "BOOLEAN" collation))
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

/* Stored compiler slots are global names or factory-call syntax with explicit
constants. Resolution stays at compilation; existing callable slots self-evaluate. */
(define sql_expression_compiler (lambda (compiler) (eval compiler)))

/* Call contracts consume resolved SQL triples from their producing scope. */
(define sql_call_info (lambda (head infos)
	(begin
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

(define sql_expr_info (lambda (sources expr)
	(match expr
		((symbol inner_select) query) (begin
			/* Only metadata is read here. The original scalar query stays in the
			logical AST for the common decorrelator and its cardinality checks. */
			(define outer_scope (binding_scope_entries sources '() "types" (list '() '() "types") 0))
			(define bound (bind_query_names query (list outer_scope)))
			(define info (if (equal? (count (queryplan_result_titles bound)) 1)
				(sql_projection_info_at bound 0 sources) (sql_info nil "any" nil)))
			(sql_info expr (sql_info_type info) (sql_info_collation info)))
		((quote inner_select) query) (sql_expr_info sources (list (quote inner_select) query))
		((symbol get_column) tblvar tbl_ic col col_ic) (sql_get_column_info sources tblvar tbl_ic col col_ic)
		((quote get_column) tblvar tbl_ic col col_ic) (sql_get_column_info sources tblvar tbl_ic col col_ic)
		((symbol quote) _datum) (sql_info expr "any" nil)
		/* Frontends attach the declared contract to their conversion formula.
		Resolve the operand normally; only its executable formula survives. */
		((symbol sql_type_expr) formula type collation)
		(begin
			(define info (sql_expr_info sources formula))
			(sql_info (sql_info_formula info) type
				(if (and (sql_text_type? type) (nil? collation)) (sql_info_collation info) collation)))
		/* Dialects compile type-directed operations from resolved argument triples.
		The callback is a compiler, never an executable SQL operator. */
		(cons (symbol sql_compile_expr) (cons compiler arguments))
		((sql_expression_compiler compiler) (map arguments (lambda (argument) (sql_expr_info sources argument))))
		((symbol sql_compile_column) compiler expression)
		(sql_compile_column_info sources compiler expression)
		((symbol session) _key) (sql_info expr "any" nil)
		((symbol session_globalvar) _key) (sql_info expr "any" nil)
		((symbol lambda) _params _body) (sql_info expr "any" nil)
		(cons head args) (sql_call_info head (map args (lambda (argument) (sql_expr_info sources argument))))
		_ (sql_info expr (sql_type_of_literal expr)
			(if (string? expr) (list "utf8mb4_general_ci" 4) nil)))))

/* NULL-propagating comparison primitive emitted by sql_comparison_info. The Less
relation and operator string are resolved at plan time. */
(define sql_compare (lambda (left right less operator)
	(if (or (nil? left) (nil? right)) nil
		(or (equal? operator "equal??") (equal? operator "equal?"))
		(and (not (less left right)) (not (less right left)))
		(equal? operator "<") (less left right)
		(equal? operator ">") (less right left)
		(equal? operator "<=") (not (less right left))
		(equal? operator ">=") (not (less left right))
		(error "unsupported SQL comparison"))))


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
(define sql_type_formula (lambda (sources expr) (sql_info_formula (sql_expr_info sources expr))))

/* Result-column descriptor (nil TYPE COLLATION) for one expression. */
(define sql_type_result_descriptor (lambda (sources expr)
	(begin
		(define info (sql_expr_info sources expr))
		(sql_info nil (sql_info_type info) (sql_info_collation info)))))

/* Canonicalize a projected field, but leave a SELECT * / t.* entry as-is: the
planner has its own star-expansion (and GROUP-BY-primary-key rules) downstream. */
(define sql_type_field_formula (lambda (sources expr)
	(if (star_expr? expr) expr (sql_type_formula sources expr))))

/* Resolve canonical persisted direction names while compiling. Bare columns
keep the native scan-order path; computed expressions and explicit COLLATE
contracts select their comparator once, outside executable row loops. */
(define sql_type_order_item (lambda (sources item)
	(match item
		'(order_expr order_dir) (begin
			(define order_dir (eval order_dir))
			(define info (sql_expr_info sources order_expr))
			(define coll (sql_info_collation info))
			(if (and (or (equal? order_dir <) (equal? order_dir >))
				(or (not (sql_column_ref? (sql_info_formula info)))
					(and (not (nil? coll)) (equal? (cadr coll) 0)))
				(sql_text_type? (sql_info_type info))
				(not (nil? coll)))
				(list (sql_info_formula info) (collate (car coll) (equal? order_dir >)))
				(list (sql_info_formula info) order_dir)))
		_ item)))

/* A mutation proof consumes these frontend contracts after executable
formulas have replaced their annotations. Never rediscover SQL operand types
from those resulting Scmer values (an integral FLOAT is still approximate).
Unannotated scalar terminals have no trustworthy integer syntax contract. */
(define sql_mutation_contract_trusted? (lambda (expr)
	(match expr
		((symbol sql_type_expr) _formula _type _collation) true
		((symbol get_column) _alias _alias_ic _column _column_ic) true
		((symbol sql_compile_column) _compiler _column) true
		/* A callback's declared result type alone does not establish exact key
		membership for computed probes (integer arithmetic may overflow). */
		(cons (symbol sql_compile_expr) _arguments) false
		(cons head args) (begin
			(define rule (sql_function_rule head))
			(and (not (nil? rule)) (not (equal? (cadr rule) "any")) (equal? (nth rule 2) "fixed")))
		_ false)))

(define sql_mutation_comparison_contracts (lambda (sources expr)
	(match expr
		((symbol quote) _datum) '()
		((symbol inner_select) _query) '()
		(cons head args) (begin
			(define operands (match expr
				(cons (symbol sql_compile_expr) (cons _compiler arguments))
				(if (equal? (count arguments) 2) arguments nil)
				'(op left right) (if (has? (list (quote equal?) (quote equal??)) op) (list left right) nil)
				_ nil))
			(define own (if operands (begin
				(define infos (map operands (lambda (operand) (sql_expr_info sources operand))))
				(define formula (sql_info_formula (sql_expr_info sources expr)))
				/* A generic compiler callback can choose any operation. Its two
				arguments authorize only the equality it actually emitted. */
				(if (and (not (nil? (equality_term_operands formula)))
					(expression_equal? (equality_term_operands formula) (map infos sql_info_formula)))
					(list (list formula (map infos (lambda (info) (sql_info nil (sql_info_type info) (sql_info_collation info))))
						(map operands sql_mutation_contract_trusted?))) '())) '()))
			(merge (cons own (map args (lambda (arg) (sql_mutation_comparison_contracts sources arg))))))
		_ '())))

/* Comparison contracts depend on declarations, never statistics in SHOW.
The native point description shares its immutable publication with type lookup;
only the consumed bounded fields participate in the warm guard. */
/* Guards retain the key semantics consumed by mutation admission as ordinary
serializable values. Constraint labels, FK declarations, snapshot identities
and statistics are not part of a uniqueness proof. Metadata enumeration costs
O(k) for the k declared keys and never reads shard data or automatic indexes. */
(define sql_mutation_key_contract (lambda (constraints primary_only)
	(if (nil? constraints) nil
		(map (filter (coalesceNil (constraints "Unique") '()) (lambda (key)
			(or (not primary_only) (equal? (key "Id") "PRIMARY"))))
			(lambda (key) (list (equal? (key "Id") "PRIMARY") (key "Cols")
				(if (key "nulls_not_distinct") true false)))))))
(define sql_mutation_column_contract (lambda (column)
	(if (nil? column) nil (list (column "RawType")
		(coalesceNil (column "Dimensions") '()) (column "Null")
		(coalesceNil (column "Collation") "")))))
(define sql_mutation_column_refs (lambda (expr)
	(match expr
		((symbol get_column) _alias _alias_ic _column _column_ic) (list expr)
		((symbol quote) _datum) '()
		((symbol inner_select) _query) '()
		(cons _head args) (merge (map args sql_mutation_column_refs))
		_ '())))
(define sql_mutation_contract_declarations (lambda (sources expressions)
	(map (merge_unique (map expressions sql_mutation_column_refs)) (lambda (reference) (begin
		(define resolved (match reference
			((symbol get_column) nil _alias_ic col col_ic)
			(if (single_source? sources) reference
				(coalesceNil (try_bind_unqualified_column_in_scopes
					(list (binding_scope_entries sources '() "types" (list '() '() "types") 0)) col col_ic) reference))
			_ reference))
		(match resolved
			((symbol get_column) alias alias_ic col col_ic) (begin
				(define src (source_for_alias sources (if (empty_list? sources) nil (source_alias (car sources))) alias alias_ic))
				(if (and src (source_is_base_table? src)
					(not (physical_helper_relation? (source_relation src)))
					(not (equal?? (source_schema src) "information_schema")))
					(begin
						(define handle (table (source_schema src) false (source_relation src) false))
						(define declaration (if handle (show handle "column" col col_ic) nil))
						(list (source_schema src) (source_relation src)
							(if declaration (declaration "Field") col)
							(sql_mutation_column_contract declaration))) nil))
			_ nil))))))

(define sql_type_query_block (lambda (block outer_sources)
	(if (not (query_block? block))
		block
		(begin
			(define original_sources (qb_sources block))
			(define original_scope (merge (list original_sources (coalesceNil outer_sources '()))))
			(define mutation (qassoc_get (qb_facts block) (quote dml) false))
			/* Capture before any frontend type getter runs. Later lowering uses
			these immutable declarations to invalidate the original contracts. */
			(define declarations (coalesceNil (qassoc_get (qb_facts block) (quote comparison-declarations) nil)
				(if mutation (filter (sql_mutation_contract_declarations original_scope (cons (qb_where block) (source_join_exprs original_sources))) (lambda (item) (not (nil? item)))) '())))
			(define mutation_contracts (coalesceNil (qassoc_get (qb_facts block) (quote comparison-contracts) nil)
				(if mutation (merge (map (cons (qb_where block) (source_join_exprs original_sources))
					(lambda (expr) (sql_mutation_comparison_contracts original_scope expr)))) '())))
			(define sources (sql_type_lower_annotations original_sources))
			(define outer (coalesceNil outer_sources '()))
			(define all_sources (if (empty_list? outer) sources (merge (list sources outer))))
			(make_query_block
				(qb_schema block)
				sources
				(map_assoc (qb_fields block)
					(lambda (field_title field_expr) (sql_type_field_formula all_sources field_expr)))
				(sql_type_formula all_sources (qb_where block))
				(map (coalesceNil (qb_group block) '())
					(lambda (group_expr) (sql_type_formula all_sources group_expr)))
				(if (nil? (qb_having block)) nil (sql_type_formula all_sources (qb_having block)))
				(map (coalesceNil (qb_order block) '())
					(lambda (order_item) (sql_type_order_item all_sources order_item)))
				(qb_limit block) (qb_offset block) (sql_type_lower_annotations (qb_hidden block)) (sql_type_lower_annotations (qb_stages block))
				(qassoc_set (qassoc_set (qassoc_set (qb_facts block) (quote comparison-declarations) declarations) (quote comparison-contracts) mutation_contracts) (quote result-types)
					(map_assoc (expand_query_block_fields sources (qb_fields block))
						(lambda (rt_title rt_expr) (sql_type_result_descriptor all_sources rt_expr)))))))))

/* Rewrite only paths containing frontend expression contracts. nil means
unchanged; a singleton holds the replacement, including a legitimate NULL.
Retain annotation-free sources/stage graphs instead of cloning every node
for frontends that have emitted no contracts. Quoted syntax stays opaque. */
(define sql_type_annotation_replacement (lambda (node)
	(match node
		((symbol sql_compile_column) _compiler _expression) (list (sql_type_formula '() node))
		((symbol sql_type_expr) _formula _type _collation) (list (sql_type_formula '() node))
		(cons (symbol sql_compile_expr) _arguments) (list (sql_type_formula '() node))
		((symbol quote) _datum) nil
		(cons head tail) (begin
			(define rewritten_head (sql_type_annotation_replacement head))
			(define rewritten_tail (sql_type_annotation_replacement tail))
			(if (and (nil? rewritten_head) (nil? rewritten_tail)) nil
				(list (cons
					(if (nil? rewritten_head) head (car rewritten_head))
					(if (nil? rewritten_tail) tail (car rewritten_tail))))))
		_ nil)))

(define sql_type_lower_annotations (lambda (node)
	(match (sql_type_annotation_replacement node)
		'(formula) formula
		_ node)))

/* Entry point: annotate root expressions, then remove remaining expression contracts
from nested logical inputs without inventing storage or planner operations. */
(define sql_type_annotate_ir (lambda (ir)
	(make_ir (ir_kind ir)
		(if (query_block? (ir_root ir))
			(sql_type_query_block (ir_root ir) '()) (sql_type_lower_annotations (ir_root ir)))
		(sql_type_lower_annotations (ir_stages ir)) (ir_context_of ir) (ir_return ir))))
