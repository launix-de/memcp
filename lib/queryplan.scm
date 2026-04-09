/*
Copyright (C) 2023, 2024, 2026  Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

/*

How MemCPs query plan builder works
-----------------------------------

MemCP will not implement any filtering or ordering on scheme lists directly since this will be very costly.
Instead, the storage engine is used to do these operations. The storage engine will automatically analyze a
lambda expression for filtering/ordering and will eventually create and use indexes.

Every filter and sort will be executed on a base table. Therefore, in GROUP BY clauses, a temporary table
has to be created. Also for cross joins (joins that either have no equality condition between the tables or
the equality is not on a unique column), there has to be a temporary cross-table.

when building a queryplan, there is a parameter `tables` which contains all tables that have to be joined.
Relevant for the iterator is now the "core". which is:
the list of tables in tables t1 that are not connected over a join t1,t2,t1.col1=t2.col2 where there is a unique key (t2.col2)
(helper function (unique? schema tbl col col col))

if the core consists of a single table, scan this table
if the core consists of two or more tables, create a temporary join table --> prejoins
if there is a group function, create a temporary preaggregate table
(helper function temptable(tbllist, collist) -> tbllist is the list of tables to be joined and collist is the list of (table, col) that will also be unique)

*/

/* helper functions:
- (build_queryplan schema tables fields condition groups schemas) builds a lisp expression that runs the query and calls resultrow for each result tuple
- (build_scan schema tables cols map reduce neutral neutral2 condition groups) builds a lisp expression that scans the tables
- (extract_columns_for_tblvar expr tblvar) extracts a list of used columns for each tblvar '(tblvar col)
- (replace_columns expr) replaces all (get_column ...) and (aggregate ...) with values

*/

/* Registers invalidation triggers on src_table to drop pj_table on any DML.
Uses code-generator pattern: values baked into quoted lambda body at register time,
so no closure capture — the trigger body serializes cleanly as a self-contained expression. */
(define register_prejoin_invalidation (lambda (src_schema src_table pj_schema pj_table) (begin
	(define prefix (concat ".prejoin:" pj_table "|" src_table "|"))
	(define drop_body (eval (list 'lambda (list 'OLD 'NEW 'session) (list 'droptable pj_schema pj_table true))))
	(createtrigger src_schema src_table (concat prefix "after_insert")     "after_insert"     "" drop_body false)
	(createtrigger src_schema src_table (concat prefix "after_update")     "after_update"     "" drop_body false)
	(createtrigger src_schema src_table (concat prefix "after_delete")     "after_delete"     "" drop_body false)
	(createtrigger src_schema src_table (concat prefix "after_drop_table") "after_drop_table" "" drop_body false)
	(createtrigger src_schema src_table (concat prefix "after_drop_column") "after_drop_column" "" drop_body false)
	true)))

/* Registers incremental maintenance triggers on src_table to keep pj_table in sync.
delete_fn/insert_fn/update_fn are code-generator-produced lambda expressions (no closures).
Lifecycle triggers use code-generator pattern for the drop body as well.
update_fn embeds delete_fn/insert_fn as proc literals in its body (no closure capture). */
(define register_prejoin_incremental (lambda (src_schema src_table pj_schema pj_table delete_fn insert_fn update_fn) (begin
	(define prefix (concat ".pj_incr:" pj_table "|" src_table "|"))
	(createtrigger src_schema src_table (concat prefix "after_delete") "after_delete" "" delete_fn false)
	(createtrigger src_schema src_table (concat prefix "after_insert") "after_insert" "" insert_fn false)
	(createtrigger src_schema src_table (concat prefix "after_update") "after_update" "" update_fn false)
	(define drop_body (eval (list 'lambda (list 'OLD 'NEW 'session) (list 'droptable pj_schema pj_table true))))
	(createtrigger src_schema src_table (concat prefix "after_drop_table") "after_drop_table" "" drop_body false)
	(createtrigger src_schema src_table (concat prefix "after_drop_column") "after_drop_column" "" drop_body false)
	true)))

/* prejoin_canonical_sources maps a materialized prejoin table name to an assoc
of physical prejoin column name -> source expression. make_keytable uses this
to canonicalize get_column markers on prejoin temps back to their original
source expressions, instead of baking the prejoin table name into the key name. */
(define prejoin_canonical_sources (newsession))
/* materialized_source_expr_lookup maps a temp table to an assoc of source-expression
string -> physical field name. Later GROUP stages can then rewrite both original
_unn_* get_column terms and their canonicalized forms onto the prejoin's actual
physical columns without guessing from aliases or suffixes. */
(define materialized_source_expr_lookup (newsession))
/* session-sensitive runtime predicate columns must not be reused across plan
builds, because their truth value depends on current session state. */
(define session_runtime_plan_counter (newsession))
(define alias_lookup_variants (lambda (alias_)
	(reduce (filter (list
		alias_
		(visible_occurrence_alias alias_)
		(if (string? alias_) (sanitize_temp_name alias_) nil)
		(if (string? (visible_occurrence_alias alias_)) (sanitize_temp_name (visible_occurrence_alias alias_)) nil)
		(if (string? alias_) (symbol alias_) nil)
		(if (string? (visible_occurrence_alias alias_)) (symbol (visible_occurrence_alias alias_)) nil)
		(if (string? alias_) nil (string alias_))
		(if (string? (visible_occurrence_alias alias_)) nil
			(if (nil? (visible_occurrence_alias alias_)) nil (string (visible_occurrence_alias alias_)))))
		(lambda (x) (not (nil? x))))
		(lambda (acc alias_v) (append_unique acc alias_v))
		'())))
(define assoc_lookup_variants (lambda (assoc variants)
	(reduce variants (lambda (found key_v)
		(if (not (nil? found))
			found
			(if (nil? assoc) nil (get_assoc assoc key_v))))
		nil)
))
(define alias_variants_match (lambda (left right insensitive)
	(reduce (alias_lookup_variants left) (lambda (matched left_v)
		(or matched
			(reduce (alias_lookup_variants right) (lambda (matched2 right_v)
				(or matched2 ((if insensitive equal?? equal?) left_v right_v)))
				false)))
		false)
))
(define materialized_expr_alias_variants alias_lookup_variants)
(define materialized_source_expr_keys (lambda (expr)
	(match expr
		'((symbol get_column) alias_ ti col ci) (reduce
			(reduce (materialized_expr_alias_variants alias_) (lambda (variants alias_v)
				(merge variants (list
					(list (symbol get_column) alias_v ti col ci)
					(normalize_visible_aliases (list (symbol get_column) alias_v ti col ci))
					(normalize_canonical_aliases (list (symbol get_column) alias_v ti col ci))
					(list (quote get_column) alias_v ti col ci)
					(normalize_visible_aliases (list (quote get_column) alias_v ti col ci))
					(normalize_canonical_aliases (list (quote get_column) alias_v ti col ci)))))
				'())
			(lambda (acc expr_variant)
				(merge acc (list
					(string expr_variant)
					(sanitize_temp_name (string expr_variant)))))
			'())
		'((quote get_column) alias_ ti col ci) (reduce
			(reduce (materialized_expr_alias_variants alias_) (lambda (variants alias_v)
				(merge variants (list
					(list (quote get_column) alias_v ti col ci)
					(normalize_visible_aliases (list (quote get_column) alias_v ti col ci))
					(normalize_canonical_aliases (list (quote get_column) alias_v ti col ci))
					(list (symbol get_column) alias_v ti col ci)
					(normalize_visible_aliases (list (symbol get_column) alias_v ti col ci))
					(normalize_canonical_aliases (list (symbol get_column) alias_v ti col ci)))))
				'())
			(lambda (acc expr_variant)
				(merge acc (list
					(string expr_variant)
					(sanitize_temp_name (string expr_variant)))))
			'())
		_ (list
			(string expr)
			(sanitize_temp_name (string expr))
			(string (normalize_canonical_aliases expr))
			(sanitize_temp_name (string (normalize_canonical_aliases expr))))
	)
))
(define planned_materialized_fields (newsession))
(define merge_schema_fields_unique (lambda (field_lists)
	(reduce (merge field_lists) (lambda (acc coldef)
		(if (reduce acc (lambda (found existing)
			(or found (equal?? (existing "Field") (coldef "Field"))))
			false)
			acc
			(merge acc (list coldef))))
		'())))
(define expand_star_fields_with_schemas (lambda (fields schemas) (begin
	(define _expand_alias_cols (lambda (alias def)
		/* Visible schema exports may merge alias/planned/shown column descriptors.
		Star expansion must dedupe by field name here so wrappers over materialized
		subqueries do not emit the same visible field twice. */
		(merge (map (merge_schema_fields_unique (list def)) (lambda (coldesc)
			'((coldesc "Field") '((quote get_column) alias false (coldesc "Field") false))
	)))))
	(define _schema_matches_alias (lambda (candidate target ignorecase)
		(or ((if ignorecase equal?? equal?) candidate target)
			((if ignorecase equal?? equal?) (visible_occurrence_alias candidate) target))))
	(define _latest_schema_for_alias (lambda (target ignorecase)
		(begin
			(define latest (newsession))
			(extract_assoc schemas (lambda (alias def)
				(if (_schema_matches_alias alias target ignorecase)
					(latest "v" def)
					nil)))
			(coalesceNil (latest "v") nil))))
	(merge (extract_assoc fields (lambda (col expr) (match expr
		'((symbol get_column) nil _ "*" _) (merge (extract_assoc schemas _expand_alias_cols))
		'((quote get_column) nil _ "*" _) (merge (extract_assoc schemas _expand_alias_cols))
		'((symbol get_column) tblvar ignorecase "*" _) (begin
			(define latest_def (_latest_schema_for_alias tblvar ignorecase))
			(if (nil? latest_def) '() (_expand_alias_cols tblvar latest_def)))
		'((quote get_column) tblvar ignorecase "*" _) (begin
			(define latest_def (_latest_schema_for_alias tblvar ignorecase))
			(if (nil? latest_def) '() (_expand_alias_cols tblvar latest_def)))
		(list col expr)
)))))))
(define materialized_source_schema (lambda (tschema ttbl alias schemas)
	(begin
		(define alias_cols (if (or (nil? alias) (not (has_assoc? schemas alias))) '() (coalesceNil (schemas alias) '())))
		(define planned_cols (coalesceNil (planned_materialized_fields ttbl) '()))
		(define shown_cols (if (and (string? ttbl)
			(>= (strlen ttbl) 1)
			(equal? (substr ttbl 0 1) ".")
			(has? (show tschema) ttbl))
			(show tschema ttbl)
			'()))
		(merge_schema_fields_unique (list alias_cols planned_cols shown_cols)))))
(define materialized_source_physical_schema (lambda (tschema ttbl alias schemas)
	(begin
		(define planned_cols (coalesceNil (planned_materialized_fields ttbl) '()))
		/* Design contract: only columns that are part of the explicit materialized
		stage schema count as physical planner inputs. Dynamic show()/compute-column
		metadata may expose virtual cache columns that are not safe scan inputs for
		a later stage and would reintroduce early aggregate/get_column substitution.
		Visible wrappers may still consult materialized_source_schema, but scan-time
		lowering must stay on stable planned columns only. */
		(merge_schema_fields_unique (list planned_cols)))))
/* Some rewrite paths carry alias provenance as (visible_alias canonical_source).
For physical scans the visible alias matters; for canonical naming / temp reuse the
canonical source side must be used so equivalent queries share the same temp cols. */
(define visible_occurrence_alias (lambda (alias_)
	(match alias_
		'(visible_alias _) visible_alias
		_ (if (string? alias_)
			(begin
				(define _parts (split alias_ "\0"))
				(if (> (count _parts) 1) (nth _parts (- (count _parts) 1)) alias_))
			alias_))))
(define normalize_visible_aliases (lambda (expr)
	(match expr
		'((symbol get_column) alias_ ti col ci)
		(list (quote get_column) (visible_occurrence_alias alias_) ti col ci)
		'((quote get_column) alias_ ti col ci)
		(list (quote get_column) (visible_occurrence_alias alias_) ti col ci)
		(cons sym args)
		(cons sym (map args normalize_visible_aliases))
		expr
	)
))
(define normalize_canonical_aliases (lambda (expr)
	(match expr
		'((symbol get_column) alias_ ti col ci)
		(match alias_
			'(_ canonical_alias) (list (quote get_column) canonical_alias ti col ci)
			_ expr)
		'((quote get_column) alias_ ti col ci)
		(match alias_
			'(_ canonical_alias) (list (quote get_column) canonical_alias ti col ci)
			_ expr)
		(cons sym args)
		(cons sym (map args normalize_canonical_aliases))
		expr
	)
)))
/* temp table / keytable names must not embed NUL alias separators from flattened
derived tables. Keep the canonical structure, but drop the separator byte in the
physical storage name so partition files remain valid on disk. */
(define sanitize_temp_name (lambda (name)
	(if (string? name) (replace name "\0" "") name)
)))
(define query_temp_table_options '("engine" "cache"))
(define query_temp_table_options_code '(list "engine" "cache"))
/* Design contract: get_column / aggregate / window sentinels stay logical for as
long as possible and are only lowered to physical scan symbols at the final
build_scan boundary. Materialized derived tables therefore must not be keyed by
their visible SQL alias alone (`t`, `x`, ...), because later keytable names are
derived from that materialized source identity. If two unrelated wrappers reuse
the same alias on a shared server, alias-only temp identities would let stale
createcolumn results bleed across queries. The rows themselves are session-bound
so stored compute lambdas can still resolve them after the surrounding lexical
scope is gone. */
(define materialized-subquery-key (lambda (id subquery)
	(concat "__mat:" id ":" (sha1 (string (normalize_canonical_aliases subquery))))))
(define materialized-subquery-source (lambda (id subquery)
	(list (list (quote context) "session") (materialized-subquery-key id subquery))))
(define materialized-subquery-init (lambda (id subquery rows_expr)
	(list (list (quote context) "session") (materialized-subquery-key id subquery) rows_expr)))
(define materialized-source? (lambda (table-source)
	(or
		(and (string? table-source) (>= (strlen table-source) 1) (equal? (substr table-source 0 1) "."))
		(match table-source
			(cons (cons (symbol context) '("session")) _) true
			(cons (cons '(quote context) '("session")) _) true
			false
		))
))
/* rewrite_source_aliases: replace get_column table aliases according to alias_map.
Used to store prejoin lineage in the same canonical source namespace that also
defines the physical prejoin column names. */
(define resolve_source_alias (lambda (alias_map alias_)
	(coalesce
		(assoc_lookup_variants alias_map (map (alias_lookup_variants alias_) string))
		alias_)))
(define rewrite_source_aliases (lambda (alias_map expr)
	(match (normalize_canonical_aliases expr)
		'((symbol get_column) alias_ ti col ci)
		(list (quote get_column) (resolve_source_alias alias_map alias_) ti col ci)
		'((quote get_column) alias_ ti col ci)
		(list (quote get_column) (resolve_source_alias alias_map alias_) ti col ci)
		(cons sym args)
		(cons sym (map args (lambda (arg) (rewrite_source_aliases alias_map arg))))
		expr
		(if (equal? expr (symbol (string expr)))
			(begin
				(define resolved (resolve_source_alias alias_map expr))
				(if (equal? resolved expr)
					expr
					(symbol (string resolved))))
			expr)
	)
))
/* Planner contract: schema-based case repair is an untangle_query concern.
Once a logical expression leaves untangle_query, every get_column in planner IR
must already be `(get_column exact_alias false exact_field false)`.
canonicalize_expr/serialize_canonical_expr and later physical plan stages therefore work strictly
case-sensitively: they may rewrite aliases in a canonical source namespace, but
they must not guess/fix alias or column casing anymore. */
(define logical_expr_has_case_flags (lambda (expr) (match expr
	'((symbol get_column) _ ti _ ci) (or ti ci)
	'((quote get_column) _ ti _ ci) (or ti ci)
	(cons sym args) (if (_is_opaque_scope_sym sym)
		false
		(or (logical_expr_has_case_flags sym)
			(reduce args (lambda (found arg) (or found (logical_expr_has_case_flags arg))) false)))
	false
)))
(define require_canonical_logical_expr (lambda (context expr)
	(if (logical_expr_has_case_flags expr)
		(error (concat "planner contract violated: " context " still contains case-insensitive get_column markers in " (serialize expr)))
		expr)
))
/* Naming contract:
- canonicalize_expr rewrites planner-local aliases into the stable canonical
source namespace, but only for already-canonical logical expressions
- serialize_canonical_expr turns that canonical logical IR into a stable key
Callers that need the canonical expression for more than one downstream use must
keep it in a local define instead of rebuilding it from scratch. */
(define canonicalize_expr (lambda (expr alias_map)
	(rewrite_source_aliases alias_map
		(require_canonical_logical_expr "canonicalize_expr" expr))
))
(define serialize_canonical_expr (lambda (expr)
	(serialize (require_canonical_logical_expr "serialize_canonical_expr" expr))
))
/* Explain helpers: keep planner debugging on a stable, compact serialization
surface so tests can assert planner structure without depending on pretty-print
layout. */
(define explain_emit_rows (lambda (rows)
	(cons (quote begin) (map rows (lambda (row)
		(list (quote resultrow) (cons (quote list) row))))))
)
(define explain_plan_root (lambda (plan)
	(match plan
		(cons sym _) (string sym)
		_ (string plan)
	)
))
(define explain_normalize_stage (lambda (stage)
	(list
		(cons (quote group-cols) (coalesceNil (stage_group_cols stage) '()))
		(list (quote having) (stage_having_expr stage))
		(list (quote order) (coalesceNil (stage_order_list stage) '()))
		(list (quote limit-partition-cols) (coalesceNil (stage_limit_partition_cols stage) 0))
		(list (quote limit) (stage_limit_val stage))
		(list (quote offset) (stage_offset_val stage))
		(list (quote group-alias) nil)
		(list (quote dedup) (stage_is_dedup stage))
		(list (quote partition-aliases) (stage_partition_aliases stage))
		(list (quote init) (stage_init_code stage))
	)
))
(define explain_normalize_stages (lambda (stages)
	(map stages explain_normalize_stage)
))
/* explain_queryplan_ir: expose planner IR around untangle_query/join_reorder.
Returns compact stage/kind/value rows for stable SQL-level inspection. */
(define explain_queryplan_ir (lambda (query) (begin
	(define _uq_result (apply untangle_query (merge query (list nil))))
	(define _uq_init (if (>= (count _uq_result) 8) (nth _uq_result 7) '()))
	(define _uq_7tuple (list (nth _uq_result 0) (nth _uq_result 1) (nth _uq_result 2) (nth _uq_result 3) (nth _uq_result 4) (nth _uq_result 5) (nth _uq_result 6)))
	(define _jr_result (apply join_reorder _uq_7tuple))
	(define _plan (apply build_queryplan (merge _jr_result (list nil))))
	(explain_emit_rows (list
		(list "stage" "untangle" "kind" "tables" "value" (serialize (nth _uq_result 1)))
		(list "stage" "untangle" "kind" "fields" "value" (serialize (nth _uq_result 2)))
		(list "stage" "untangle" "kind" "condition" "value" (serialize (nth _uq_result 3)))
		(list "stage" "untangle" "kind" "groups" "value" (serialize (explain_normalize_stages (nth _uq_result 4))))
		(list "stage" "untangle" "kind" "init" "value" (serialize _uq_init))
		(list "stage" "reorder" "kind" "tables" "value" (serialize (nth _jr_result 1)))
		(list "stage" "reorder" "kind" "changed" "value" (not (equal? (nth _uq_result 1) (nth _jr_result 1))))
		(list "stage" "plan" "kind" "root" "value" (explain_plan_root _plan))
	))
)))
/* explain_queryplan_reorder: focused view for join-reorder work. */
(define explain_queryplan_reorder (lambda (query) (begin
	(define _uq_result (apply untangle_query (merge query (list nil))))
	(define _uq_7tuple (list (nth _uq_result 0) (nth _uq_result 1) (nth _uq_result 2) (nth _uq_result 3) (nth _uq_result 4) (nth _uq_result 5) (nth _uq_result 6)))
	(define _jr_result (apply join_reorder _uq_7tuple))
	(define table_rows_for_stage (lambda (stage_name tables)
		(map (produceN (count tables)) (lambda (idx) (match (nth tables idx)
			'(alias schema tbl isOuter joinexpr)
			(list
				"stage" stage_name
				"position" idx
				"alias" (string alias)
				"schema" (string schema)
				"table" (string tbl)
				"outer" isOuter
				"joinexpr" (serialize (coalesceNil joinexpr true)))
			_ (list
				"stage" stage_name
				"position" idx
				"alias" ""
				"schema" ""
				"table" (serialize (nth tables idx))
				"outer" nil
				"joinexpr" "true"
			)
		)))
	))
	(explain_emit_rows (merge
		(table_rows_for_stage "untangle" (nth _uq_result 1))
		(table_rows_for_stage "reorder" (nth _jr_result 1))
	))
)))
/* Compatibility wrapper for older call sites. New planner code should keep the
canonical expression in a local define and only serialize it at the edge. */
(define canonical_expr_name (lambda (expr columns params alias_map)
	(serialize_canonical_expr (canonicalize_expr expr alias_map))
))
/* build_occurrence_alias_map: assign a stable canonical source namespace to query
aliases. Single occurrences keep the physical table name for maximal reuse.
If the same physical table appears multiple times in one query tree, append an
occurrence index so self-joins do not collapse distinct roles. */
(define build_occurrence_alias_map (lambda (tables) (begin
	(define total_counts (newsession))
	(map tables (lambda (td) (match td
		'(_ tschema ttbl _ _) (begin
			(define src (concat tschema "." ttbl))
			(total_counts src (+ 1 (coalesceNil (total_counts src) 0)))
			nil)
		nil)))
	(define seen_counts (newsession))
	(define alias_pairs (map tables (lambda (td) (match td
		'(tv tschema ttbl _ _) (begin
			(define src (concat tschema "." ttbl))
			(define idx (coalesceNil (seen_counts src) 0))
			(define canon (if (> (coalesceNil (total_counts src) 0) 1)
				(concat src "#" idx)
				src))
			(seen_counts src (+ idx 1))
			(list tv canon))
		(list "" "")))))
	(reduce alias_pairs
		(lambda (acc pair) (match pair
			'(tv canon) (begin
				(define visible (visible_occurrence_alias tv))
				(define tv_sanitized (if (string? tv) (sanitize_temp_name tv) tv))
				(define visible_sanitized (if (string? visible) (sanitize_temp_name visible) visible))
				(reduce (filter (list tv visible tv_sanitized visible_sanitized canon)
					(lambda (alias_v) (not (nil? alias_v))))
					(lambda (acc2 alias_v) (set_assoc acc2 (string alias_v) canon))
					acc))
			(list "" "") acc))
		'()))
)))

(define rewrite_materialized_source_columns (lambda (tbl tblvar expr)
	(begin
		(define source_alias_map (prejoin_canonical_sources tbl))
		(if (or (nil? source_alias_map) (not (list? source_alias_map)))
			expr
			(match expr
				'((symbol get_column) (eval tblvar) ti col ci)
				(begin
					(coalesce (source_alias_map col) expr))
				'((quote get_column) (eval tblvar) ti col ci)
				(begin
					(coalesce (source_alias_map col) expr))
				(cons sym args)
				(cons sym (map args (lambda (arg) (rewrite_materialized_source_columns tbl tblvar arg))))
				expr)))))

/* lower_materialized_source_expr: deterministically lower expressions on a
materialized temp source onto the original source-expression namespace recorded
for that temp source. This keeps keytable/cache names stable and avoids leaking
raw _unn_* occurrence aliases into physical temp column names. */
(define lower_materialized_source_expr (lambda (tbl tblvar expr)
	(begin
		(define expr_lookup (materialized_source_expr_lookup tbl))
		(define source_alias_map (prejoin_canonical_sources tbl))
		(define planned_cols (coalesceNil (planned_materialized_fields tbl) '()))
		(define lower_node (lambda (node) (begin
			(define normalized_node (normalize_canonical_aliases node))
			(define node_keys (materialized_source_expr_keys node))
			(define planned_source_expr
				(match node
					'((symbol get_column) _ _ col _)
					(reduce planned_cols (lambda (found coldef)
						(if (not (nil? found))
							found
							(if (equal? (coldef "Field") col)
								(coalesceNil (coldef "Expr") nil)
								nil)))
						nil)
					'((quote get_column) _ _ col _)
					(reduce planned_cols (lambda (found coldef)
						(if (not (nil? found))
							found
							(if (equal? (coldef "Field") col)
								(coalesceNil (coldef "Expr") nil)
								nil)))
						nil)
					nil))
			(define planned_field
				(reduce planned_cols (lambda (found coldef)
					(if (not (nil? found))
						found
						(begin
							(define source_expr (coalesceNil (coldef "Expr") nil))
							(if (and (not (nil? source_expr))
								(or (equal? (normalize_canonical_aliases source_expr) normalized_node)
									(reduce node_keys (lambda (matched key)
										(or matched (has? (materialized_source_expr_keys source_expr) key)))
										false)))
								(coldef "Field")
								nil))))
					nil))
			(define direct_source_expr
				(match node
					'((symbol get_column) _ _ col _)
					(coalesce planned_source_expr
						(if (nil? source_alias_map) nil (source_alias_map col)))
					'((quote get_column) _ _ col _)
					(coalesce planned_source_expr
						(if (nil? source_alias_map) nil (source_alias_map col)))
					nil))
			(define direct_field
				(coalesce planned_field
					(if (nil? expr_lookup)
						nil
						(reduce (materialized_source_expr_keys node) (lambda (found key)
							(if (not (nil? found))
								found
								(coalesce (expr_lookup key) nil)))
							nil))))
			(if (not (nil? direct_source_expr))
				direct_source_expr
				(if (not (nil? direct_field))
					(coalesce
						(if (nil? source_alias_map) nil (source_alias_map direct_field))
						node)
					(match node
						(cons sym args)
						(cons sym (map args lower_node))
						_
						(rewrite_materialized_source_columns tbl tblvar node)))))))
		(lower_node expr)
)))

/* returns a list of all tblvar aliases referenced via get_column in expr */
(define extract_tblvars (lambda (expr)
	(match expr
		'((symbol get_column) tblvar _ _ _) (if (nil? tblvar) '() '(tblvar))
		'((quote get_column) tblvar _ _ _) (if (nil? tblvar) '() '(tblvar))
		(cons sym args) (merge_unique (map args extract_tblvars))
		'()
	)
))

/* returns a list of '(string...) */
(define extract_columns_for_tblvar (lambda (tblvar expr)
	(match expr
		'((symbol get_column) (eval tblvar) _ col _) (if (equal? col "*") '() '(col)) /* TODO: case matching */
		'((quote get_column) (eval tblvar) _ col _) (if (equal? col "*") '() '(col))
		(cons sym args) /* function call */ (merge_unique (map args (lambda (arg) (extract_columns_for_tblvar tblvar arg))))
		'()
	)
))

/* changes (get_column tblvar ti col ci) into its symbol */
(define replace_columns_from_expr (lambda (expr)
	(match expr
		(cons (symbol aggregate) args) /* aggregates: don't dive in */ expr
		'((symbol get_column) tblvar _ col _) (if (nil? tblvar) (symbol (concat "__unresolved__." col)) (symbol (concat tblvar "." col)))
		'((quote get_column) tblvar _ col _) (if (nil? tblvar) (symbol (concat "__unresolved__." col)) (symbol (concat tblvar "." col)))
		(cons sym args) /* function call */ (cons sym (map args replace_columns_from_expr))
		expr /* literals */
	)
))

/* scalar subselect helper wrappers */
(define scalar_scan (lambda (schema tbl filtercols filterfn mapcols mapfn reduce neutral reduce2) (begin
	(define result (scan (session "__memcp_tx") schema (scan-runtime-source tbl) filtercols filterfn mapcols mapfn reduce neutral reduce2))
	(if (equal? result neutral) nil result)
)))
(define scalar_scan_order (lambda (schema tbl filtercols filterfn sortcols sortdirs offset limit mapcols mapfn reduce neutral) (begin
	(define result (scan_order (session "__memcp_tx") schema (scan-runtime-source tbl) filtercols filterfn sortcols sortdirs 0 offset limit mapcols mapfn reduce neutral))
	(if (equal? result neutral) nil result)
)))
(define scan-runtime-source (lambda (table-source) (match table-source
	'(materialized-subquery key) (list (list (quote context) "session") key)
	'((symbol materialized-subquery) key) (list (list (quote context) "session") key)
	'((quote materialized-subquery) key) (list (list (quote context) "session") key)
	table-source
)))

(define extend_codegen_lambda (lambda (fn extra_params)
	(match fn
		'((symbol lambda) params body) (list (quote lambda) (merge (list params extra_params)) body)
		'((symbol lambda) params body numvars) (list (quote lambda) (merge (list params extra_params)) body numvars)
		'((quote lambda) params body) (list (quote lambda) (merge (list params extra_params)) body)
		'((quote lambda) params body numvars) (list (quote lambda) (merge (list params extra_params)) body numvars)
		_ fn
	)
))

(define append_codegen_list (lambda (lst extra_items)
	(if (list? lst)
		(merge (list lst extra_items))
		lst)
))

/* nested scan batching is a peephole optimizer pass over the generated
queryplan AST: once build_scan has emitted a child scan tree, rewrite only the
first reachable scan/scan_batch node so it can consume parent batchdata via #N
pseudocolumns without changing the higher-level join planning rules. */
/* batchify_first_scan: peephole rewrite that converts the first reachable
scan/scan_batch in a plan tree into a scan_batch that consumes parent
batchdata via #N pseudocolumns.  Uses two-level match: outer match
destructures the AST node, inner match dispatches on the head symbol name
(via (string head)) so (symbol X) and (quote X) are handled uniformly.
Virtual tables (where tbl is a list, not a string) are excluded via
(string? tbl) and fall through without rewriting. */
(define batchify_first_scan (lambda (plan batch_params batch_pseudocols stride_expr batchdata_sym)
	(match plan
		/* scan / scan_batch with a real (string) table name */
		(cons scanhead (cons tx (cons schema (cons (string? tbl) rest))))
		(match (string scanhead)
			"scan" (match rest
				(merge '(filtercols filterfn mapcols mapfn reduce neutral reduce2 isOuter) _)
				(if isOuter nil
					(list (quote scan_batch) tx schema tbl
						(append_codegen_list filtercols batch_pseudocols)
						(extend_codegen_lambda filterfn batch_params)
						(append_codegen_list mapcols batch_pseudocols)
						(extend_codegen_lambda mapfn batch_params)
						stride_expr batchdata_sym
						reduce neutral reduce2 isOuter))
				nil)
			"scan_batch" (match rest
				(merge '(filtercols filterfn mapcols mapfn inner_stride inner_batchdata reduce neutral reduce2 isOuter) _)
				(if isOuter nil
					(list (quote scan_batch) tx schema tbl
						(append_codegen_list filtercols batch_pseudocols)
						(extend_codegen_lambda filterfn batch_params)
						(append_codegen_list mapcols batch_pseudocols)
						(extend_codegen_lambda mapfn batch_params)
						inner_stride inner_batchdata
						reduce neutral reduce2 isOuter))
				nil)
			nil)
		/* wrapper nodes: recurse into the contained value/scan */
		(cons wraphead (cons arg1 arg2))
		(match (string wraphead)
			"nth" (begin /* (nth inner_scan idx) */
				(define rewritten (batchify_first_scan arg1 batch_params batch_pseudocols stride_expr batchdata_sym))
				(if (nil? rewritten) nil
					(list (quote nth) rewritten (car arg2))))
			"define" (begin /* (define sym value) */
				(define rewritten (batchify_first_scan (car arg2) batch_params batch_pseudocols stride_expr batchdata_sym))
				(if (nil? rewritten) nil
					(list (quote define) arg1 rewritten)))
			"set" (begin /* (set sym value) */
				(define rewritten (batchify_first_scan (car arg2) batch_params batch_pseudocols stride_expr batchdata_sym))
				(if (nil? rewritten) nil
					(list (quote set) arg1 rewritten)))
			"begin" (batchify_begin_forms plan wraphead rest batch_params batch_pseudocols stride_expr batchdata_sym)
			"!begin" (batchify_begin_forms plan wraphead rest batch_params batch_pseudocols stride_expr batchdata_sym)
			"begin_mut" (batchify_begin_forms plan wraphead rest batch_params batch_pseudocols stride_expr batchdata_sym)
			nil)
		nil)))

/* batchify_begin_forms: helper for batchifying inside begin/!begin/begin_mut blocks.
Tries preferred forms (scan, scan_batch, nth, begin variants) first, then
falls back to any form. */
(define batchify_begin_forms (lambda (plan head rest batch_params batch_pseudocols stride_expr batchdata_sym) (begin
	(define rewrite_forms_by_predicate (lambda (forms should_try)
		(match forms
			'() nil
			(cons form tail) (begin
				(if (should_try form)
					(begin
						(define rewritten_form (batchify_first_scan form batch_params batch_pseudocols stride_expr batchdata_sym))
						(if (nil? rewritten_form)
							(match (rewrite_forms_by_predicate tail should_try)
								nil nil
								rewritten_tail (cons form rewritten_tail))
							(cons rewritten_form tail)))
					(match (rewrite_forms_by_predicate tail should_try)
						nil nil
						rewritten_tail (cons form rewritten_tail)))))))
	(define is_preferred_form (lambda (form) (match form
		(cons fh _) (match (string fh)
			"scan" true "scan_batch" true "nth" true
			"begin" true "!begin" true "begin_mut" true
			false)
		false)))
	(match (rewrite_forms_by_predicate rest is_preferred_form)
		nil (match (rewrite_forms_by_predicate rest (lambda (form) true))
			nil nil
			rewritten_rest (cons head rewritten_rest))
		rewritten_rest (cons head rewritten_rest)))))

/* builds the outer scan shell for the peephole-rewritten child plan. the join
order and scan tree come from build_scan already; this helper only swaps the
row-at-a-time inner scan calls for buffered scan_batch flushes. */
(define build_batched_regular_scan (lambda (schema tbl filtercols outer_filter_lambda scan_mapcols scan_mapfn_params batch_map_params direct_inner_scan batched_inner_scan batch_stride batch_capacity is_update_target isOuter) (begin
	(define _outer_batch_row_lambda
		(list (quote lambda) scan_mapfn_params
			(list (quote begin)
				(list (quote define) (symbol "__record") (list (quote list)))
				(cons (quote append_mut) (cons (symbol "__record") batch_map_params)))))
	(if (nil? batched_inner_scan)
		(scan_wrapper 'scan schema tbl
			(cons list filtercols)
			outer_filter_lambda
			scan_mapcols
			(list (symbol "lambda") scan_mapfn_params direct_inner_scan)
			(if is_update_target (symbol "+") nil)
			(if is_update_target 0 nil)
			nil
			isOuter)
		(begin
			(define _inner_flush_define
				(list (quote define) (symbol "__inner_flush")
					(list (quote lambda) (list (symbol "__batchbuf")) batched_inner_scan)))
			(if is_update_target
				(list (quote begin)
					_inner_flush_define
					(list (quote nth)
						(list (quote scan) '(session "__memcp_tx") schema (scan-runtime-source tbl)
							(cons list filtercols)
							outer_filter_lambda
							scan_mapcols
							_outer_batch_row_lambda
							(list (quote lambda) (list (symbol "acc") (symbol "rowvals"))
								(list (quote begin)
									(list (quote define) (symbol "__state")
										(list (quote if) (list (quote nil?) (list (quote nth) (symbol "acc") 1))
											(list (quote list) (list (quote nth) (symbol "acc") 0) (list (quote list)))
											(symbol "acc")))
									(list (quote define) (symbol "__batchdata0") (list (quote nth) (symbol "__state") 1))
									(list (quote define) (symbol "__batchdata") (list (quote apply) (quote append_mut) (list (quote cons) (symbol "__batchdata0") (symbol "rowvals"))))
									(list (quote nth_mut) (symbol "__state") 1 (symbol "__batchdata"))
									(list (quote if) (list (quote >=) (list (quote count) (symbol "__batchdata")) batch_capacity)
										(list (quote begin)
											(list (quote nth_mut) (symbol "__state") 0
												(list (quote +) (list (quote nth) (symbol "__state") 0) (list (symbol "__inner_flush") (symbol "__batchdata"))))
											(list (quote reset_mut) (symbol "__batchdata")))
										true)
									(symbol "__state")))
							(list (quote list) 0 nil)
							(list (quote lambda) (list (symbol "acc") (symbol "shardstate"))
								(list (quote begin)
									(list (quote define) (symbol "__shardbuf") (list (quote nth) (symbol "shardstate") 1))
									(list (quote define) (symbol "__shardresult")
										(list (quote if)
											(list (quote or)
												(list (quote nil?) (symbol "__shardbuf"))
												(list (quote equal?) (list (quote count) (symbol "__shardbuf")) 0))
											(list (quote nth) (symbol "shardstate") 0)
											(list (quote +) (list (quote nth) (symbol "shardstate") 0) (list (symbol "__inner_flush") (symbol "__shardbuf")))))
									(list (quote list) (list (quote +) (list (quote nth) (symbol "acc") 0) (symbol "__shardresult")) nil)))
							isOuter)
						0))
				(list (quote begin)
					_inner_flush_define
					(list (quote scan) '(session "__memcp_tx") schema (scan-runtime-source tbl)
						(cons list filtercols)
						outer_filter_lambda
						scan_mapcols
						_outer_batch_row_lambda
						(list (quote lambda) (list (symbol "batchdata") (symbol "rowvals"))
							(list (quote begin)
								(list (quote define) (symbol "__batchbuf0")
									(list (quote if) (list (quote nil?) (symbol "batchdata"))
										(list (quote list))
										(symbol "batchdata")))
								(list (quote define) (symbol "__batchbuf") (list (quote apply) (quote append_mut) (list (quote cons) (symbol "__batchbuf0") (symbol "rowvals"))))
								(list (quote if) (list (quote >=) (list (quote count) (symbol "__batchbuf")) batch_capacity)
									(list (quote begin)
										(list (symbol "__inner_flush") (symbol "__batchbuf"))
										(list (quote reset_mut) (symbol "__batchbuf")))
									true)
								(symbol "__batchbuf")))
						nil
						(list (quote lambda) (list (symbol "acc") (symbol "shardbuf"))
							(list (quote begin)
								(list (quote if)
									(list (quote or)
										(list (quote nil?) (symbol "shardbuf"))
										(list (quote equal?) (list (quote count) (symbol "shardbuf")) 0))
									true
									(list (symbol "__inner_flush") (symbol "shardbuf")))
								nil))
						isOuter))))))))
/* returns a list of all aggregates in this expr */
(define extract_aggregates (lambda (expr)
	(match expr
		(cons (symbol aggregate) args) '(args)
		(cons '(quote aggregate) args) '(args)
		(cons sym args) /* function call */ (merge (map args extract_aggregates))
		/* literal */ '()
	)
))

/* session-sensitive COUNT/EXISTS stages must be classified before the physical
planner starts splitting aggregate conditions apart. */
(define expr_uses_session_state (lambda (expr)
	(match expr
		(symbol session) true
		'(quote session) true
		(cons (symbol session) _) true
		(cons '(quote session) _) true
		(cons (symbol context) '("session")) true
		(cons (cons (symbol context) '("session")) _) true
		(cons (symbol ?) '("__memcp_tx")) true
		(cons '(quote ?) '("__memcp_tx")) true
		(cons sym args) (reduce args (lambda (acc arg) (or acc (expr_uses_session_state arg))) false)
		false
	)
))

/* returns a list of all window function nodes (fn args over) in this expr */
(define extract_window_funcs (lambda (expr)
	(match expr
		(cons (symbol window_func) rest) (list rest)
		(cons sym args) /* function call */ (merge (map args extract_window_funcs))
		/* literal */ '()
	)
))

/* extract_all_get_columns: return all (get_column tblvar _ col _) refs as ("tblvar.col" expr) pairs */
(define extract_all_get_columns (lambda (expr)
	(match expr
		'((symbol get_column) tblvar _ col _) (if (nil? tblvar) '() (list (list (concat tblvar "." col) expr)))
		'((quote get_column) tblvar _ col _) (if (nil? tblvar) '() (list (list (concat tblvar "." col) expr)))
		(cons sym args) (merge (map args extract_all_get_columns))
		'()
	)
))

/* extract_all_table_aliases: return a flat list of all table aliases referenced
via get_column nodes in an expression.  Used by LEFT JOIN pruning to detect
which tables are actually read. */
(define extract_all_table_aliases (lambda (expr)
	(match expr
		'((symbol get_column) alias_ _ _ _) (if (nil? alias_) '() (list (string alias_)))
		'((quote get_column) alias_ _ _ _) (if (nil? alias_) '() (list (string alias_)))
		(cons sym args) (merge (map args extract_all_table_aliases))
		'()
	)
))

/* extract_scanned_tables: walk an expression AST and return all (schema table) pairs from scan/scan_order calls.
Used to detect which tables a computor lambda reads from, so we can register invalidation triggers. */
(define extract_scanned_tables (lambda (expr)
	(match expr
		(cons (symbol scan) (cons current_tx (cons schema (cons tbl rest)))) (cons (list schema tbl) (merge (map rest extract_scanned_tables)))
		(cons (symbol scan_order) (cons current_tx (cons schema (cons tbl rest)))) (cons (list schema tbl) (merge (map rest extract_scanned_tables)))
		(cons (symbol scalar_scan) (cons current_tx (cons schema (cons tbl rest)))) (cons (list schema tbl) (merge (map rest extract_scanned_tables)))
		(cons (symbol scalar_scan_order) (cons current_tx (cons schema (cons tbl rest)))) (cons (list schema tbl) (merge (map rest extract_scanned_tables)))
		(cons sym args) (merge (map args extract_scanned_tables))
		'()
	)
))

/* expr_has_scan: returns true if an AST expression contains any scan node. */
(define expr_has_scan (lambda (expr)
	(match expr
		(cons head rest) (match (string head)
			"scan" true "scan_order" true "scan_batch" true
			"scalar_scan" true "scalar_scan_order" true
			_ (reduce rest (lambda (acc child) (or acc (expr_has_scan child))) false))
		false)))

/* expr_refs_outer_var: returns true if an expression references (var N) at the
top level (outside any nested lambda). */
(define expr_refs_outer_var (lambda (expr)
	(match expr
		(cons head rest) (match (string head)
			"var" true
			"lambda" false
			_ (reduce rest (lambda (acc child) (or acc (expr_refs_outer_var child))) false))
		false)))

/* expr_is_parallelizable: safe to wrap in a parallel thunk if it contains
a scan AND does not reference outer-scope vars. */
(define expr_is_parallelizable (lambda (expr)
	(and (expr_has_scan expr) (not (expr_refs_outer_var expr)))))

/* parallelize_resultrows: post-processing pass over the finished query plan AST.
Rewrites (resultrow (list k1 v1 k2 v2 ...)) nodes: if >=2 value expressions
are parallelizable, wrap them in parallel_map for concurrent evaluation.
Only recurses into transparent wrappers (begin/if/define/time). */
(define parallelize_resultrows (lambda (ast)
	(match ast
		(cons head rest)
		(if (equal? (string head) "resultrow")
			(match rest
				(cons (cons list_head kv_pairs) rr_rest)
				(begin
					(define vals (extract_assoc kv_pairs (lambda (k v) v)))
					(define complex_count (reduce vals (lambda (acc v) (+ acc (if (expr_is_parallelizable v) 1 0))) 0))
					(if (< complex_count 2)
						ast
						(begin
							(define keys (extract_assoc kv_pairs (lambda (k v) k)))
							(define thunks (map vals (lambda (v) (list (quote lambda) '() v))))
							(define pmap_call (list (symbol "parallel_map")
								(cons (symbol "list") thunks)
								(list (quote lambda) (list (symbol "__pf")) (list (symbol "__pf")))))
							(define reassembled (cons list_head
								(merge (mapIndex keys (lambda (i k) (list k (list (symbol "nth") (symbol "__pr") i)))))))
							(list (quote begin)
								(list (quote define) (symbol "__pr") pmap_call)
								(cons head (cons reassembled rr_rest))))))
				ast)
			(match (string head)
				"begin" (cons head (map rest parallelize_resultrows))
				"!begin" (cons head (map rest parallelize_resultrows))
				"if" (cons head (map rest parallelize_resultrows))
				"define" (cons head (map rest parallelize_resultrows))
				"time" (cons head (map rest parallelize_resultrows))
				ast))
		ast)))

/* split_condition: selection pushdown for nested-loop join planning.
Splits an AND-condition into (now, later): predicates evaluatable with currently
bound tables vs predicates that must wait for inner tables to be scanned.
Enables index-based filtering in scan/scan_order by pushing predicates down. */
(define split_condition (lambda (expr tables) (match expr
	'((symbol get_column) tblvar _ col _) /* a column */ (match tables
		'() '(expr true) /* last condition: compute now */
		(cons (cons (eval tblvar) _) _) '(true expr) /* col depends on tblvar */
		(cons _ tablesrest) (split_condition expr tablesrest) /* check next table in join plan */
		(error "invalid tables list")
	)
	'((quote get_column) tblvar _ col _) /* a column */ (match tables
		'() '(expr true) /* last condition: compute now */
		(cons (cons (eval tblvar) _) _) '(true expr) /* col depends on tblvar */
		(cons _ tablesrest) (split_condition expr tablesrest) /* check next table in join plan */
		(error "invalid tables list")
	)
	'((symbol aggregate) _ _ _) (if (equal? tables '()) '(expr true) '(true expr))
	'((quote aggregate) _ _ _) (if (equal? tables '()) '(expr true) '(true expr))
	(cons (symbol and) conditions) /* splittable and */ (split_condition_and conditions tables)
	/* Scope contract: runtime subplans and other opaque scope nodes carry their
	own alias domain. split_condition must not recurse into them, otherwise inner
	get_column refs from correlated scalar subqueries get misclassified as later
	join refs of the surrounding group stage and leak into keytable lowering. */
	(cons sym args) /* non-splittable function call */ (if (_is_opaque_scope_sym sym)
		'(expr true)
		(split_condition_combine sym args tables))
	/* literal */ '(expr true)
)))
(define split_condition_combine (lambda (sym args tables) (if
	(reduce args (lambda (other arg) (match (split_condition arg tables) '(_ true) other true)) false) /* if one of the args is later, everything is later */
	'(true (cons sym args))
	'((cons sym args) true)
)))
(define split_condition_and (lambda (l tables) (match l
	'() '(true true)
	(cons head tail) (match '((split_condition head tables) (split_condition_and tail tables))
		'('(true true) '(x y)) '(x y)
		'('(true y) '(x true)) '(x y)
		'('(x true) '(true y)) '(x y)
		'('(x y) '(true true)) '(x y)
		'('(x1 y) '(x2 true)) '('('and x1 x2) y)
		'('(x1 true) '(x2 y)) '('('and x1 x2) y)
		'('(true y1) '(x y2)) '(x '('and y1 y2))
		'('(x y1) '(true y2)) '(x '('and y1 y2))
		'('(x1 y1) '(x2 y2)) '('('and x1 x2) '('and y1 y2))
	)
)))

(define flatten_and_terms (lambda (expr) (match expr
	(cons sym parts) (if (or (equal? sym (quote and)) (equal? sym '(quote and)) (equal? sym 'and))
		(merge (map parts flatten_and_terms))
		(if (or (nil? expr) (equal? expr true)) '() (list expr)))
	_ (if (or (nil? expr) (equal? expr true)) '() (list expr))
)))

(define combine_and_terms (lambda (parts) (begin
	(define _parts (filter parts (lambda (x) (and (not (nil? x)) (not (equal? x true))))))
	(if (equal? _parts '()) true
		(if (equal? 1 (count _parts)) (car _parts)
			(cons (quote and) _parts)))
)))

/* split_scan_condition: keep joinexpr separate from global WHERE.
Returns (now later) for one scan level:
- INNER scans: joinexpr parts evaluatable now are pushed into now; later parts stay deferred.
- OUTER scans: only joinexpr stays on the scan (ON semantics); global WHERE terms are deferred. */
(define split_scan_condition (lambda (isOuter joinexpr scan_condition rest_tables) (begin
	(match (split_condition (coalesceNil scan_condition true) rest_tables) '(raw_now_condition raw_later_condition)
		(match (split_condition (coalesceNil joinexpr true) rest_tables) '(join_now_condition join_later_condition)
			(if (not isOuter)
				(list
					(combine_and_terms (merge (flatten_and_terms raw_now_condition) (flatten_and_terms join_now_condition)))
					(combine_and_terms (merge (flatten_and_terms raw_later_condition) (flatten_and_terms join_later_condition))))
				(list
					(combine_and_terms (flatten_and_terms join_now_condition))
					(combine_and_terms (merge
						(flatten_and_terms raw_now_condition)
						(flatten_and_terms raw_later_condition)
						(flatten_and_terms join_later_condition))))))))))

/* helper to check list membership */
(define list_contains (lambda (lst item) (reduce lst (lambda (acc x) (or acc (equal? x item))) false)))

/* has_only_tblvar_refs: returns true if expr contains get_column refs and ALL of them
reference only the given tblvar. Returns false if any get_column references another alias,
or if expr has no get_column refs at all (pure literal → not a tblvar-only condition). */
(define has_only_tblvar_refs (lambda (expr tblvar) (match expr
	'((symbol get_column) alias_ _ _ _) (equal? alias_ tblvar)
	'((quote get_column) alias_ _ _ _) (equal? alias_ tblvar)
	(cons sym args) (reduce args (lambda (acc arg) (begin
		(define child (has_only_tblvar_refs arg tblvar))
		(if (nil? acc) child (if (nil? child) acc (and acc child))))) nil)
	nil /* literal: no refs → nil (unknown) */
)))

/* extract_pure_tblvar_conditions: from an AND expression, extract parts that
reference ONLY tblvar columns (no outer refs). Returns the AND of those parts, or true. */
(define extract_pure_tblvar_conditions (lambda (expr tblvar) (match expr
	(cons (symbol and) parts) (reduce parts (lambda (acc part)
		(if (equal? (has_only_tblvar_refs part tblvar) true)
			(if (equal? acc true) part (list (quote and) acc part))
			acc)) true)
	_ (if (equal? (has_only_tblvar_refs expr tblvar) true) expr true)
)))

/* extract_non_pure_tblvar_conditions: from an AND expression, extract parts that
reference OTHER tables too (not only tblvar). Complement of extract_pure_tblvar_conditions. */
(define extract_non_pure_tblvar_conditions (lambda (expr tblvar) (match expr
	(cons (symbol and) parts) (reduce parts (lambda (acc part)
		(if (not (equal? (has_only_tblvar_refs part tblvar) true))
			(if (equal? acc true) part (list (quote and) acc part))
			acc)) true)
	_ (if (not (equal? (has_only_tblvar_refs expr tblvar) true)) expr true)
)))

/* helper to collect all column references in an expression */
(define collect_all_column_refs (lambda (expr) (match expr
	'((symbol get_column) tblvar _ col _) (list (list tblvar col))
	(cons sym args) (merge_unique (map args collect_all_column_refs))
	'()
)))

(define extract_outer_columns_for_tblvar (lambda (tblvar expr) (match expr
	(cons sym args) (if (or (equal? sym (quote outer)) (equal? sym '(quote outer)) (equal? sym '(symbol outer)))
		(match args
			'(symname) (begin
				(define parts (split (string symname) "."))
				(match parts
					(list tbl col) (if (equal?? tbl (string tblvar)) (list col) '())
					_ '()
				)
			)
			_ '()
		)
		(merge_unique (map args (lambda (arg) (extract_outer_columns_for_tblvar tblvar arg))))
	)
	'()
)))

/* columns of tblvar that are needed only because later table-local joinexprs reference them.
These columns must still be mapped by the current scan so nested join filters can see them. */
(define extract_later_joinexpr_columns_for_tblvar (lambda (tblvar tables)
	(merge_unique (map tables (lambda (td) (match td
		'(_ _ _ _ je) (if (nil? je) '() (extract_columns_for_tblvar tblvar je))
		'()))))
))

/* symbols that canonicalize_columns must NOT recurse into — they have their own scope */
(define _is_opaque_scope_sym (lambda (sym) (match sym
	/* inner_select markers are NOT opaque — they are logical markers that
	must be transparent for outer-ref detection during Neumann decorrelation.
	Only physical runtime code (scan, !begin, etc.) is opaque. */
	(symbol !begin) true '(quote !begin) true '!begin true
	(symbol scan) true '(quote scan) true 'scan true
	(symbol scan_order) true '(quote scan_order) true 'scan_order true
	(symbol scalar_scan) true '(quote scalar_scan) true 'scalar_scan true
	(symbol newpromise) true '(quote newpromise) true 'newpromise true
	(symbol newsession) true '(quote newsession) true 'newsession true
	_ false)))
(define is_quote_scope_sym (lambda (sym) (match sym
	(symbol quote) true '(quote quote) true 'quote true
	(symbol quasiquote) true '(quote quasiquote) true 'quasiquote true
	_ false)))
(define expr_has_opaque_scope (lambda (expr) (match expr
	(cons sym args) (or
		(_is_opaque_scope_sym sym)
		(reduce args (lambda (found arg) (or found (expr_has_opaque_scope arg))) false))
	false)))
/* extract runtime-sensitive subexpressions that do not depend on table rows.
Those terms must affect cache identity (temp column names) even if the
relational key/domain stays unchanged. */
(define extract_runtime_cache_terms (lambda (expr) (match expr
	(cons sym args) (if (_is_opaque_scope_sym sym)
		(merge_unique (map args extract_runtime_cache_terms))
		(if (and (expr_uses_session_state expr) (equal? (extract_tblvars expr) '()))
			(list expr)
			(merge_unique (map args extract_runtime_cache_terms))))
	'()
)))
/* runtime_cache_suffix_from_exprs: derive a stable, value-sensitive appendix for
temp column names from session-/context-dependent terms. The suffix is computed
at plan-build time, so repeated queries with the same runtime values reuse the
same cache column while different session values get separate temp columns. */
(define planner_eval_runtime_term (lambda (expr)
	(define _bind_context_session (lambda (node) (match node
		(symbol session) '(context "session")
		(cons sym args) (cons (_bind_context_session sym) (map args _bind_context_session))
		node)))
	(eval (_bind_context_session expr))
))
(define runtime_cache_suffix_from_exprs (lambda (exprs) (begin
	(define terms (merge_unique (map exprs extract_runtime_cache_terms)))
	(if (equal? terms '())
		""
		(concat "|rt:"
			(serialize_canonical_expr
				(canonicalize_expr
					(map terms (lambda (term) (list term (planner_eval_runtime_term term))))
					'(list)))))
)))
(define assoc_keys_as_dataset_rows (lambda (dict width)
	(map (extract_assoc dict (lambda (k v) k))
		(lambda (k)
			(if (list? k)
				k
				(if (<= width 1)
					(list k)
					(map (produceN width) (lambda (_) nil))))
))))
/* Column-resolution contract:
- parser-level get_column markers may still carry ti/ci flags inside untangle_query
- they must be resolved against schema metadata exactly once before the logical IR
crosses into build_queryplan
- later planner stages operate strictly case-sensitively on canonical aliases and
field names and must not re-run schema repair
resolve_schema_column_ref_scoped is the shared lookup primitive for that boundary:
it canonicalizes alias/field casing from schema metadata and keeps the preferred
search order for unqualified refs (main tables before helper/unnested aliases). */
(define main_scope_alias? (lambda (alias)
	(begin
		(define s (string alias))
		(and (equal? (replace s "\0" "") s)
			(not (and (>= (strlen s) 5) (equal? (substr s 0 5) "_unn_")))))))
/* Shared schema-resolution contract:
- all alias/column lookups flow through these helpers
- callers choose whether they want the first visible match or require uniqueness
- the resolver itself is the only place that may interpret alias variants or
schema-driven column casing inside queryplan.scm */
(define schema_alias_matches (lambda (query_alias schema_alias ti)
	((if ti equal?? equal?) query_alias schema_alias)
))
(define resolve_schema_alias_scoped (lambda (schemas alias_ ti)
	(if (nil? alias_) nil
		(reduce_assoc schemas (lambda (found alias cols)
			(if (and (nil? found) (schema_alias_matches alias_ alias ti))
				alias
				found))
			nil)
	)
))
(define schema_column_def (lambda (cols col ci)
	(reduce cols (lambda (found coldef)
		(if (not (nil? found))
			found
			(if ((if ci equal?? equal?) (coldef "Field") col) coldef nil)))
		nil)
))
(define schema_has_column? (lambda (cols col ci)
	(not (nil? (schema_column_def cols col ci)))
))
(define canonical_schema_column_name (lambda (cols col ci)
	(begin
		(define coldef (if (nil? cols) nil (schema_column_def cols col ci)))
		(coalesce (if (nil? coldef) nil (coldef "Field")) col))
))
(define collect_schema_column_matches_scoped (lambda (local_schemas visible_schemas alias_ ti col ci) (begin
	(define collect_matches (lambda (schemas alias_pred)
		(reduce_assoc schemas (lambda (acc alias cols)
			(if (and (alias_pred alias) (schema_has_column? cols col ci))
				(merge acc (list (list alias (canonical_schema_column_name cols col ci))))
				acc))
			'())))
	(if (nil? alias_)
		(begin
			(define _main (collect_matches local_schemas main_scope_alias?))
			(if (equal? _main '())
				(collect_matches local_schemas (lambda (alias) true))
				_main))
		(collect_matches visible_schemas (lambda (alias) (schema_alias_matches alias_ alias ti))))
)))
(define first_schema_column_match_scoped (lambda (local_schemas visible_schemas alias_ ti col ci)
	(match (collect_schema_column_matches_scoped local_schemas visible_schemas alias_ ti col ci)
		(cons head _) head
		nil
)))
(define unique_schema_column_match_scoped (lambda (local_schemas visible_schemas alias_ ti col ci)
	(match (collect_schema_column_matches_scoped local_schemas visible_schemas alias_ ti col ci)
		(cons head '()) head
		nil
)))
(define resolve_schema_column_ref_scoped (lambda (local_schemas visible_schemas alias_ ti col ci)
	(begin
		(define resolved (first_schema_column_match_scoped local_schemas visible_schemas alias_ ti col ci))
		(if (nil? resolved) nil
			(list (nth resolved 0) (nth resolved 1)))
	)
))
(define resolve_unique_schema_column_ref_scoped (lambda (local_schemas visible_schemas alias_ ti col ci)
	(begin
		(define resolved (unique_schema_column_match_scoped local_schemas visible_schemas alias_ ti col ci))
		(if (nil? resolved) nil
			(list (nth resolved 0) (nth resolved 1)))
	)
))
(define resolve_schema_column_expr_scoped (lambda (local_schemas visible_schemas alias_ ti col ci)
	(begin
		(define resolved (first_schema_column_match_scoped local_schemas visible_schemas alias_ ti col ci))
		(if (nil? resolved)
			nil
			(list (quote get_column) (nth resolved 0) false (nth resolved 1) false))
	)
))
/* canonicalize_columns_scoped resolves ti/ci flags to canonical casing.
local_schemas are the aliases visible in the current scope, while visible_schemas
also contains outer aliases needed for qualified outer refs like src.ID.
Unqualified refs must only match local aliases so recursive untangling keeps
free get_columns free instead of accidentally binding them to an outer table. */
(define canonicalize_columns_scoped (lambda (expr local_schemas visible_schemas) (match expr
	'((symbol get_column) alias_ ti col ci) (if (or ti ci)
		(begin
			(define resolved (resolve_schema_column_expr_scoped local_schemas visible_schemas alias_ ti col ci))
			(if (nil? resolved)
				expr /* leave unresolved — replace_find_column will handle or error */
				resolved))
		expr /* ti=false ci=false: already canonical */
	)
	'((quote get_column) alias_ ti col ci) (if (or ti ci)
		(begin
			(define resolved (resolve_schema_column_expr_scoped local_schemas visible_schemas alias_ ti col ci))
			(if (nil? resolved)
				expr /* leave unresolved — replace_find_column will handle or error */
				resolved))
		expr /* ti=false ci=false: already canonical */
	)
	/* do not recurse into opaque scope nodes — inner_select, runtime code */
	(cons sym args) (if (_is_opaque_scope_sym sym) expr
		(cons (canonicalize_columns_scoped sym local_schemas visible_schemas)
			(map args (lambda (a) (canonicalize_columns_scoped a local_schemas visible_schemas)))))
	expr
)))))
/* canonicalize_columns keeps the old single-schema API for callers that do not
cross a scope boundary. */
(define canonicalize_columns (lambda (expr all_schemas)
	(canonicalize_columns_scoped expr all_schemas all_schemas)
))
/* finalize_logical_expr is the only normalization gate from untangle_query into
the downstream planner.
Order matters:
1. canonicalize_columns: resolve parser-level ti/ci flags against schemas
2. rewrite_expr: lower visible derived-table aliases to their logical source expr
3. canonicalize_columns again: any get_column introduced by rewrite_expr must
also leave untangle_query in exact schema casing
After this helper, later planner stages must only see exact/case-sensitive
get_column markers and may no longer run schema-based repair heuristics. */
(define finalize_logical_expr_scoped (lambda (expr local_schemas visible_schemas rewrite_expr enforce_contract) (begin
	(define finalized
		(canonicalize_columns_scoped
			(rewrite_expr (canonicalize_columns_scoped expr local_schemas visible_schemas))
			local_schemas
			visible_schemas))
	(if enforce_contract
		(require_canonical_logical_expr "untangle_query output" finalized)
		finalized))
))
(define finalize_logical_expr (lambda (expr all_schemas rewrite_expr enforce_contract)
	(finalize_logical_expr_scoped expr all_schemas all_schemas rewrite_expr enforce_contract)
))
(define finalize_logical_stage_scoped (lambda (stage local_schemas visible_schemas rewrite_expr enforce_contract) (begin
	(define fin (lambda (expr) (finalize_logical_expr_scoped expr local_schemas visible_schemas rewrite_expr enforce_contract)))
	(define sg (coalesceNil (stage_group_cols stage) '()))
	(define sh (stage_post_group_condition_expr stage))
	(define so (coalesceNil (stage_order_list stage) '()))
	(define sl (stage_limit_val stage))
	(define soff (stage_offset_val stage))
	(define spa (stage_partition_aliases stage))
	(if (stage_is_dedup stage)
		(stage_preserve_cache_meta stage (make_dedup_stage (map sg fin) spa))
		(if (and (not (nil? spa)) (or (nil? sg) (equal? sg '())))
			(stage_preserve_cache_meta stage (make_partition_stage spa
				(map so (lambda (o) (match o '(c d) (list (fin c) d))))
				(coalesceNil (stage_limit_partition_cols stage) 0) sl soff (stage_init_code stage)))
			(stage_preserve_cache_meta stage (make_group_stage_with_condition
				(map sg fin)
				(fin sh)
				(map so (lambda (o) (match o '(c d) (list (fin c) d))))
				sl soff spa (stage_init_code stage)
				(begin (define sc (stage_condition stage)) (if (nil? sc) nil (fin sc)))))))
)))
(define finalize_logical_stage (lambda (stage all_schemas rewrite_expr enforce_contract)
	(finalize_logical_stage_scoped stage all_schemas all_schemas rewrite_expr enforce_contract)
))
/* canonicalize all get_column markers in a group stage */
(define canonicalize_stage (lambda (stage all_schemas) (begin
	(define canon (lambda (expr) (canonicalize_columns expr all_schemas)))
	(define sg (coalesceNil (stage_group_cols stage) '()))
	(define sh (stage_having_expr stage))
	(define so (coalesceNil (stage_order_list stage) '()))
	(define sl (stage_limit_val stage))
	(define soff (stage_offset_val stage))
	(define spa (stage_partition_aliases stage))
	(if (stage_is_dedup stage)
		(stage_preserve_cache_meta stage (make_dedup_stage (map sg canon) spa))
		(if (and (not (nil? spa)) (or (nil? sg) (equal? sg '())))
			/* partition stage (aliases but no group): preserve partition-aliases and limit-partition-cols */
			(stage_preserve_cache_meta stage (make_partition_stage spa
				(map so (lambda (o) (match o '(c d) (list (canon c) d))))
				(coalesceNil (stage_limit_partition_cols stage) 0) sl soff (stage_init_code stage)))
			/* group stage (possibly scoped with aliases) */
			(stage_preserve_cache_meta stage (make_group_stage_with_condition
				(map sg canon)
				(canon sh)
				(map so (lambda (o) (match o '(c d) (list (canon c) d))))
				sl soff spa (stage_init_code stage)
				(begin (define sc (stage_condition stage)) (if (nil? sc) nil (canon sc)))))))
)))

(import "sql-metadata.scm")

/* group stage constructors and accessors - shared between untangle_query and build_queryplan
All stages have partition-aliases (scope): nil = global (all tables), list = scoped to those tables.
All stages have init: nil = no init code, or code to run before the scan. */
(define normalize_stage_aliases (lambda (aliases)
	(if (nil? aliases)
		nil
		(if (list? aliases)
			aliases
			(list aliases)))))
/* stage-condition: optional WHERE condition scoped to this stage's tables.
For scoped stages from Neumann unnesting, carries the inner subquery's WHERE
so build_queryplan applies it as keytable scan filter without polluting the
global condition (which would cause cross-stage condition leakage). */
(define make_group_stage (lambda (group having order limit offset aliases init)
	(list
		(cons (quote group-cols) (coalesce group '()))
		(list (quote having) having)
		(list (quote order) (coalesce order '()))
		(list (quote limit-partition-cols) 0)
		(list (quote limit) limit)
		(list (quote offset) offset)
		(list (quote dedup) false)
		(list (quote partition-aliases) (normalize_stage_aliases aliases))
		(list (quote init) init)
		(list (quote stage-condition) nil)
	)
))
(define make_group_stage_with_condition (lambda (group having order limit offset aliases init cond)
	(list
		(cons (quote group-cols) (coalesce group '()))
		(list (quote having) having)
		(list (quote order) (coalesce order '()))
		(list (quote limit-partition-cols) 0)
		(list (quote limit) limit)
		(list (quote offset) offset)
		(list (quote dedup) false)
		(list (quote partition-aliases) (normalize_stage_aliases aliases))
		(list (quote init) init)
		(list (quote stage-condition) cond)
	)
))
(define make_partition_stage (lambda (aliases order partition_cols limit offset init)
	(list
		(cons (quote group-cols) '())
		(list (quote having) nil)
		(list (quote order) (coalesce order '()))
		(list (quote limit-partition-cols) partition_cols)
		(list (quote limit) limit)
		(list (quote offset) offset)
		(list (quote dedup) false)
		(list (quote partition-aliases) (normalize_stage_aliases aliases))
		(list (quote init) init)
	)
))
(define make_dedup_stage (lambda (group aliases)
	(list
		(cons (quote group-cols) (coalesce group '()))
		(list (quote having) nil)
		(list (quote order) '())
		(list (quote limit-partition-cols) 0)
		(list (quote limit) nil)
		(list (quote offset) nil)
		(list (quote dedup) true)
		(list (quote partition-aliases) (normalize_stage_aliases aliases))
		(list (quote init) nil)
	)
))
(define stage_group_cols (lambda (stage) (reduce stage (lambda (acc item)
	(if (nil? acc) (match item
		(cons (quote group-cols) cols) cols
		_ nil
	) acc)
) nil)))
(define stage_having_expr (lambda (stage) (reduce stage (lambda (acc item)
	(if (nil? acc) (match item
		(cons (quote having) rest) (if (nil? rest) nil (car rest))
		_ nil
	) acc)
) nil)))
/* Compatibility alias: older unnesting logic still refers to the logical
post-group predicate under this name. On current master it is the HAVING expr. */
(define stage_post_group_condition_expr stage_having_expr)
(define stage_order_list (lambda (stage) (reduce stage (lambda (acc item)
	(if (nil? acc) (match item
		(cons (quote order) rest) (if (nil? rest) '() (car rest))
		_ nil
	) acc)
) nil)))
(define stage_limit_val (lambda (stage) (reduce stage (lambda (acc item)
	(if (nil? acc) (match item
		(cons (quote limit) rest) (if (nil? rest) nil (car rest))
		_ nil
	) acc)
) nil)))
(define stage_offset_val (lambda (stage) (reduce stage (lambda (acc item)
	(if (nil? acc) (match item
		(cons (quote offset) rest) (if (nil? rest) nil (car rest))
		_ nil
	) acc)
) nil)))
(define stage_limit_partition_cols (lambda (stage) (reduce stage (lambda (acc item)
	(if (nil? acc) (match item
		(cons (quote limit-partition-cols) rest) (if (nil? rest) 0 (car rest))
		_ nil
	) acc)
) nil)))
(define stage_partition_aliases (lambda (stage) (reduce stage (lambda (acc item)
	(if (nil? acc) (match item
		(cons (quote partition-aliases) rest) (if (nil? rest) nil (normalize_stage_aliases (car rest)))
		_ nil
	) acc)
) nil)))
(define stage_init_code (lambda (stage) (reduce stage (lambda (acc item)
	(if (nil? acc) (match item
		(cons (quote init) rest) (if (nil? rest) nil (car rest))
		_ nil
	) acc)
) nil)))
(define stage_condition (lambda (stage) (reduce stage (lambda (acc item)
	(if (nil? acc) (match item
		(cons (quote stage-condition) rest) (if (nil? rest) nil (car rest))
		_ nil
	) acc)
) nil)))
(define stage_cache_policy (lambda (stage) (reduce stage (lambda (acc item)
	(if (nil? acc) (match item
		(cons (quote cache-policy) rest) (if (nil? rest) nil (car rest))
		_ nil
	) acc)
) nil)))
(define stage_cache_query (lambda (stage) (reduce stage (lambda (acc item)
	(if (nil? acc) (match item
		(cons (quote cache-query) rest) (if (nil? rest) nil (car rest))
		_ nil
	) acc)
) nil)))
(define stage_is_dedup (lambda (stage) (reduce stage (lambda (acc item)
	(if acc acc (match item
		'((quote dedup) true) true
		_ false
	))
) false)))
(define stage_with_cache_policy (lambda (stage policy)
	(if (nil? policy) stage
		(cons (list (quote cache-policy) policy)
			(filter stage (lambda (item) (match item
				(cons (quote cache-policy) _) false
				true))))))
)
(define stage_with_cache_query (lambda (stage query)
	(if (nil? query) stage
		(cons (list (quote cache-query) query)
			(filter stage (lambda (item) (match item
				(cons (quote cache-query) _) false
				true))))))
)
(define stage_preserve_cache_meta (lambda (old_stage new_stage)
	(stage_with_cache_query
		(stage_with_cache_policy new_stage (stage_cache_policy old_stage))
		(stage_cache_query old_stage)
	)
))

/* query term helpers */
(define query_union_all_parts (lambda (query) (match query
	'(union_all branches order limit offset) (list branches order limit offset)
	'((symbol union_all) branches order limit offset) (list branches order limit offset)
	'((quote union_all) branches order limit offset) (list branches order limit offset)
	_ nil
)))
(define query_is_select_core (lambda (query) (and (list? query) (>= (count query) 9))))
(define query_branch_field_names (lambda (query) (match query
	'(schema tables fields condition group having order limit offset) (extract_assoc fields (lambda (k v) k))
	_ '()
)))

/* make_keytable_schema: compute keytable name and schema without creating the table.
Used by untangle to predict the keytable name for HAVING subselect decorrelation.
Returns (keytable_name key_col_names schema_def) where schema_def is a list of
column descriptors suitable for the schemas assoc in untangle_query.
Does NOT handle FK→PK reuse (returns nil for that case — caller must check). */
(define make_keytable_schema (lambda (schema tbl keys tblvar) (begin
	(define alias_map (list (list tblvar (concat schema "." tbl))))
	(define key_names (map keys (lambda (k)
		(sanitize_temp_name
			(canonical_expr_name (normalize_canonical_aliases (lower_materialized_source_expr tbl tblvar k)) '(list) '(list) alias_map)))))
	(define keytable_name (concat "." tbl ":" key_names))
	(define schema_def (map key_names (lambda (colname) (list "Field" colname "Type" "any"))))
	(list keytable_name key_names schema_def)
)))

/* make_keytable: create a canonically named group/key table with sloppy engine
Returns (keytable_name init_code fk_pk_col) where init_code is plan-time code that ensures
the table exists at execution time (survives cache eviction of sloppy tables).
fk_pk_col is non-nil when FK→PK reuse is active (parent table used instead of temp keytable).
condition_suffix: if non-nil, appended to name (for dedup stages with WHERE) */
(define make_keytable (lambda (schema tbl keys tblvar condition_suffix) (begin
	(define physical_tbl (string? tbl))
	(define keytable_source_name (if physical_tbl
		tbl
		(string tblvar)))
	/* FK→PK reuse: if single-column GROUP BY on a FK column without condition,
	reuse the parent (referenced) table instead of creating a temp keytable.
	The rest of the grouped pipeline must still see the normal logical key name,
	so install a temp alias column on the parent table when the physical PK name
	differs from the canonical GROUP BY key name. */
	(define fk_result (if (and physical_tbl (nil? condition_suffix) (equal? 1 (count keys)))
		(match (car keys)
			'('get_column (eval tblvar) false scol false) (begin
				(define fk_info (get_fk_target schema tbl scol))
				(if (not (nil? fk_info))
					(begin
						(define alias_map (list (list tblvar (concat schema "." tbl))))
						(define key_name
							(sanitize_temp_name
								(canonical_expr_name
									(normalize_canonical_aliases
										(lower_materialized_source_expr tbl tblvar (car keys)))
									'(list) '(list) alias_map)))
						(define parent_tbl (car fk_info))
						(define parent_col (car (cdr fk_info)))
						(if (equal? key_name parent_col)
							(list parent_tbl nil key_name)
							(begin
								(createcolumn schema parent_tbl key_name "any" '() '("temp" true)
									(list parent_col)
									(eval (list 'lambda (list (symbol parent_col)) (symbol parent_col))))
								(list parent_tbl
									(list 'createcolumn schema parent_tbl key_name "any"
										(list 'quote '())
										(list 'quote '("temp" true))
										(list 'quote (list parent_col))
										(list 'lambda (list (symbol parent_col)) (symbol parent_col)))
									key_name))))
					nil))
			nil)
		nil))
	(if (not (nil? fk_result))
		fk_result
		(begin
			(define alias_map (list (list tblvar (concat schema "." keytable_source_name))))
			(define key_names (map keys (lambda (k)
				(sanitize_temp_name
					(canonical_expr_name (normalize_canonical_aliases (lower_materialized_source_expr tbl tblvar k)) '(list) '(list) alias_map)))))
			(define condition_name (if (nil? condition_suffix) nil
				(fnv_hash (concat
					(canonical_expr_name (normalize_canonical_aliases (lower_materialized_source_expr tbl tblvar condition_suffix)) '(list) '(list) alias_map)
					(runtime_cache_suffix_from_exprs (list condition_suffix))))))
			(define key_name_at (lambda (i) (nth key_names i)))
			(define key_at (lambda (i) (nth keys i)))
			(define keytable_name (if (nil? condition_suffix)
				(concat "." keytable_source_name ":" key_names)
				(concat "." keytable_source_name ":" key_names "|" condition_name)))
			/* compute column definitions and partition spec at compile time */
			(define kt_cols (cons
				'("unique" "group" key_names)
				(map key_names (lambda (colname) '("column" colname "any" '() '())))))
			(define kt_partition (if physical_tbl
				(merge (map (produceN (count keys)) (lambda (i)
					(match (key_at i)
						'('get_column (eval tblvar) false scol false) (list (list (key_name_at i) (shardcolumn schema tbl scol)))
						'()))))
				'()))
			/* Keytable creation happens ONLY at runtime via init_code below.
			The query plan cache means compile-time createtable would only run on
			first parse, leaving the keytable vulnerable to cache eviction before
			the next (cached) execution. Runtime init is idempotent and fast. */
			/* build runtime init code to re-create after potential cache eviction (mirrors prejoin pattern) */
			(define kt_cols_code (cons 'list
				(cons
					(cons 'list (cons "unique" (cons "group" (list (cons 'list key_names)))))
					(map key_names (lambda (colname) (list 'list "column" colname "any" '(list) '(list)))))))
			(define kt_partition_code (cons 'list (if physical_tbl
				(merge (map (produceN (count keys)) (lambda (i)
					(match (key_at i)
						'('get_column (eval tblvar) false scol false) (list (list 'list (key_name_at i) (cons 'list (shardcolumn schema tbl scol))))
						'()))))
				'())))
			(define init_code (list 'begin
				(list 'define '__kt_created (list 'createtable schema keytable_name kt_cols_code query_temp_table_options_code true))
				(list 'if '__kt_created
					(list 'partitiontable schema keytable_name kt_partition_code)
					nil)
				(list 'touch_keytable schema keytable_name)
				'__kt_created))
			/* return (name init_code nil) — third element nil means no FK reuse */
			(list keytable_name init_code nil)))
)))

/* build_agg_window_plan: generates the full plan for aggregate window functions (SUM/COUNT/MIN/MAX OVER).
Uses keytable infrastructure (same as GROUP BY): make_keytable + collect + createcolumn + scalar fetch.
Result query runs on the BASE table; window_func expressions are replaced with scalar keytable scans. */
(define build_agg_window_plan (lambda (schema tbl tblvar tables over_partition wf_resolved condition groups schemas replace_find_column fields isOuter replace_columns_from_expr extract_columns_for_tblvar scan_wrapper) (begin
	(define has_partition (not (equal? over_partition '())))
	(define partition_exprs (map over_partition replace_find_column))
	(define group_keys (if has_partition partition_exprs '(1)))
	(define canon_alias_map (list (list tblvar (concat schema "." tbl))))
	(define materialized_source (materialized-source? tbl))
	(define expr_name (lambda (expr)
		(canonical_expr_name (normalize_canonical_aliases (rewrite_materialized_source_columns tbl tblvar expr)) '(list) '(list) canon_alias_map)))
	(set condition (replace_find_column (coalesceNil condition true)))
	(define window_runtime_suffix (runtime_cache_suffix_from_exprs (merge
		(list condition)
		partition_exprs
		(merge (map wf_resolved (lambda (wf) (match wf '(fn args _) args '())))))))
	(define kt_result (make_keytable schema tbl group_keys tblvar nil))
	(match kt_result '(grouptbl keytable_init fk_pk_col) (begin
		(define is_fk_reuse (not (nil? fk_pk_col)))
		(define tblvar_cols (if has_partition (merge_unique (map group_keys (lambda (col) (extract_columns_for_tblvar tblvar col)))) '()))
		(define materialized_cols (if materialized_source
			(materialized_source_physical_schema schema tbl tblvar schemas)
			'()))
		/* Design contract:
		Keep aggregate/window sentinels logical while naming and wiring stages.
		Only the scan expression of the current materialized source may lower a
		nested aggregate marker to the already materialized column that computes it.
		This prevents raw (aggregate ...) nodes from leaking into build_scan while
		still avoiding early physical substitution in the logical stage graph. */
		(define lower_window_runtime_expr (lambda (expr) (match expr
			(cons (symbol aggregate) agg_args) (begin
				(define agg_name (canonical_expr_name (normalize_canonical_aliases agg_args) '(list) '(list) canon_alias_map))
				(define match_col (if materialized_source
					(reduce materialized_cols (lambda (found coldef)
						(if (not (nil? found)) found
							(begin
								(define field_name (coldef "Field"))
								(if (and (>= (strlen field_name) (+ (strlen agg_name) 1))
									(equal? (substr field_name 0 (strlen agg_name)) agg_name)
									(equal? (substr field_name (strlen agg_name) 1) "|"))
									field_name
									nil))))
						nil)
					nil))
				(if (nil? match_col)
					(match agg_args
						'(agg_expr agg_reduce agg_neutral)
						(list (quote aggregate) (lower_window_runtime_expr agg_expr) agg_reduce agg_neutral)
						_ expr)
					(list (quote get_column) tblvar false match_col false)))
			(cons '(quote aggregate) agg_args) (begin
				(define agg_name (canonical_expr_name (normalize_canonical_aliases agg_args) '(list) '(list) canon_alias_map))
				(define match_col (if materialized_source
					(reduce materialized_cols (lambda (found coldef)
						(if (not (nil? found)) found
							(begin
								(define field_name (coldef "Field"))
								(if (and (>= (strlen field_name) (+ (strlen agg_name) 1))
									(equal? (substr field_name 0 (strlen agg_name)) agg_name)
									(equal? (substr field_name (strlen agg_name) 1) "|"))
									field_name
									nil))))
						nil)
					nil))
				(if (nil? match_col)
					(match agg_args
						'(agg_expr agg_reduce agg_neutral)
						(list (quote aggregate) (lower_window_runtime_expr agg_expr) agg_reduce agg_neutral)
						_ expr)
					(list (quote get_column) tblvar false match_col false)))
			(cons sym args) (cons sym (map args lower_window_runtime_expr))
			expr)))
		(set filtercols (if has_partition
			(merge_unique (list
				(extract_columns_for_tblvar tblvar condition)
				(extract_outer_columns_for_tblvar tblvar condition)))
			'()))
		/* collect plan */
		(define collect_plan (if (equal? group_keys '(1))
			'('insert schema grouptbl '(list "1") '(list '(list 1)) '(list) '('lambda '() true) true)
			(begin
				(define keycols (merge_unique (map group_keys (lambda (expr) (extract_columns_for_tblvar tblvar expr)))))
				(scan_wrapper 'scan schema tbl
					(cons list filtercols)
					'('lambda (map filtercols (lambda (col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr condition)))
					(cons list keycols)
					'('lambda (map keycols (lambda (col) (symbol (concat tblvar "." col)))) (cons 'list (map group_keys (lambda (expr) (replace_columns_from_expr expr)))))
					'('lambda '('acc 'rowvals) '('set_assoc 'acc 'rowvals true))
					'(list)
					'('lambda '('acc 'sharddict) '('insert schema grouptbl (cons 'list (map group_keys expr_name)) '('extract_assoc 'sharddict '('lambda '('k 'v) 'k)) '(list) '('lambda '() true) true))
					isOuter))))
		/* aggregate descriptors */
		(define condition_hash (fnv_hash (concat (expr_name condition) window_runtime_suffix)))
		(define agg_col_name (lambda (ag) (fnv_hash (concat (expr_name ag) "|" (expr_name condition) window_runtime_suffix))))
		(define fk_child_col (if is_fk_reuse (if has_partition (match (car group_keys) '('get_column _ false scol false) scol) nil) nil))
		(define ags (map wf_resolved (lambda (wf) (match wf '(fn args _) (begin
			/* args already resolved via replace_find_column in wf_resolved */
			(define map_expr (if (equal? fn "COUNT") 1 (if (nil? args) 1 (car args))))
			(define sep (if (and (equal? fn "GROUP_CONCAT") (> (count args) 1)) (cadr args) ","))
			(match fn "SUM" (list map_expr '+ 0) "COUNT" (list 1 '+ 0) "MIN" (list map_expr 'min nil) "MAX" (list map_expr 'max nil)
				"GROUP_CONCAT" (list '('concat map_expr) '('lambda '('a 'b) '('if '('nil? 'a) 'b '('concat 'a sep 'b))) nil)
				(error (concat "unsupported aggregate window function: " fn))))))))
		/* createcolumn on KEYTABLE */
		(define agg_plans (map ags (lambda (ag) (match ag '(expr reduce neutral) (begin
			(define runtime_expr (lower_window_runtime_expr expr))
			(define cols (extract_columns_for_tblvar tblvar runtime_expr))
			'('createcolumn schema grouptbl (agg_col_name ag) "any" '(list) '(list "temp" true)
				(cons list (map group_keys (lambda (col) (if is_fk_reuse fk_pk_col (expr_name col)))))
				'('lambda (map group_keys (lambda (col) (symbol (if is_fk_reuse fk_pk_col (expr_name col)))))
					(scan_wrapper 'scan schema tbl
						(cons list (merge tblvar_cols filtercols))
						'('lambda (map (merge tblvar_cols filtercols) (lambda (col) (symbol (concat tblvar "." col)))) (optimize (cons 'and (cons (replace_columns_from_expr condition) (map group_keys (lambda (col) '('equal? (replace_columns_from_expr col) '('outer (symbol (if is_fk_reuse fk_pk_col (expr_name col)))))))))))
						(cons list cols)
						'('lambda (map cols (lambda (col) (symbol (concat tblvar "." col)))) (replace_columns_from_expr runtime_expr))
						reduce neutral nil isOuter))))))))
		(define compute_plan (cons 'parallel agg_plans))
		/* replace window_func with scalar fetch */
		(define replace_wf_with_fetch (lambda (expr) (match expr
			(cons (symbol window_func) wf_rest) (begin
				(define wf_fn (car wf_rest))
				(define wf_args (cadr wf_rest))
				(define map_expr (if (equal? wf_fn "COUNT") 1 (if (nil? wf_args) 1 (replace_find_column (car wf_args)))))
				(define sep (if (and (equal? wf_fn "GROUP_CONCAT") (> (count wf_args) 1)) (cadr wf_args) ","))
				(define ag_col (agg_col_name (match wf_fn "SUM" (list map_expr '+ 0) "COUNT" (list 1 '+ 0) "MIN" (list map_expr 'min nil) "MAX" (list map_expr 'max nil)
					"GROUP_CONCAT" (list '('concat map_expr) '('lambda '('a 'b) '('if '('nil? 'a) 'b '('concat 'a sep 'b))) nil)
					(list map_expr '+ 0))))
				(if has_partition (begin
					(define kt_key_names (map group_keys (lambda (col) (if is_fk_reuse fk_pk_col (expr_name col)))))
					/* outer refs need raw column names (tblvar.col), not canonical expr_name */
					(define raw_col_names (map group_keys (lambda (col) (match col '('get_column _ _ c _) c (expr_name col)))))
					(list 'scan '(session "__memcp_tx") schema grouptbl
						(cons 'list kt_key_names)
						/* filter: (equal? grouptbl.kt_key (outer tblvar.raw_col)) — zip kt_key_names with raw_col_names */
						(list 'lambda
							(map kt_key_names (lambda (kn) (symbol (concat grouptbl "." kn))))
							(cons 'and (map (produceN (count kt_key_names) (lambda (i) i)) (lambda (i)
								(list 'equal? (symbol (concat grouptbl "." (nth kt_key_names i))) (list 'outer (symbol (concat tblvar "." (nth raw_col_names i)))))))))
						(list 'list ag_col)
						'('lambda '('__v) '__v)
						'('lambda '('__a '__b) '__b) nil nil false))
					(list 'scan '(session "__memcp_tx") schema grouptbl '(list) '('lambda '() true)
						(list 'list ag_col)
						'('lambda '('__v) '__v)
						'('lambda '('__a '__b) '__b) nil nil false)))
			(cons sym args_) (cons sym (map args_ replace_wf_with_fetch))
			expr)))
		(define new_fields (map_assoc fields (lambda (k v) (replace_wf_with_fetch (replace_find_column v)))))
		(define scan_plan (build_queryplan schema tables new_fields condition groups schemas replace_find_column nil))
		(list 'begin keytable_init '('time collect_plan "collect") '('time compute_plan "compute") scan_plan)))
)))

/* make_col_replacer: create a function that rewrites column/aggregate references to point at a group table
is_dedup=true: leave aggregates intact (for dedup stages)
is_dedup=false: replace aggregates with column fetches (for normal group stages) */
(define make_col_replacer (lambda (grouptbl condition is_dedup expr_name src_tblvar agg_col_name) (begin
	(define colname (lambda (expr) (if (nil? expr_name) (concat expr) (expr_name expr))))
	/* strip_agg_scope: remove optional 4th scope-tag from aggregate tuple */
	(define strip_agg_scope (lambda (r) (match r '(a b c _s) (list a b c) r)))
	(define replacer (lambda (expr) (match expr
		(cons (symbol aggregate) rest) (if is_dedup
			expr
			'('get_column grouptbl false (agg_col_name (strip_agg_scope rest)) false))
		(cons '(quote aggregate) rest) (if is_dedup
			expr
			'('get_column grouptbl false (agg_col_name (strip_agg_scope rest)) false))
		'((symbol get_column) (eval src_tblvar) ti col ci) '('get_column grouptbl ti (colname '('get_column src_tblvar ti col ci)) ci)
		/* rewrite (outer tblvar.col) inside scalar subselects to reference keytable column */
		'('outer sym) (begin
			(define symStr (concat sym))
			(define prefix (concat src_tblvar "."))
			(define prefixLen (strlen prefix))
			(if (and (>= (strlen symStr) prefixLen) (equal? (substr symStr 0 prefixLen) prefix))
				(begin
					(define col (substr symStr prefixLen (- (strlen symStr) prefixLen)))
					(define gc_expr '('get_column src_tblvar false col false))
					(define kt_col (colname gc_expr))
					'('outer (symbol (concat grouptbl "." kt_col))))
				expr))
		(cons sym args) (cons sym (map args replacer))
		expr
	)))
	replacer
)))

/* rewrite_for_prejoin: rewrite only columns coming from the prejoin source scope.
Outer tables must stay untouched so scoped GROUP stages can still join the
materialized prejoin/keytable back to the surrounding row stream. */
(define rewrite_for_prejoin (lambda (pjvar alias_map expr)
	(match expr
		'((symbol get_column) tblvar _ col _) (if (or (nil? tblvar) (nil? (alias_map tblvar))) expr
			'('get_column pjvar false (canonical_expr_name (normalize_canonical_aliases expr) '(list) '(list) alias_map) false))
		'((quote get_column) tblvar _ col _) (if (or (nil? tblvar) (nil? (alias_map tblvar))) expr
			'('get_column pjvar false (canonical_expr_name (normalize_canonical_aliases expr) '(list) '(list) alias_map) false))
		(cons sym args) (cons sym (map args (lambda (a) (rewrite_for_prejoin pjvar alias_map a))))
		expr
	)
))

/* replace_tblvar_with_dict: replace (get_column tv _ col _) refs for a specific tv
with (list 'get_assoc dict_sym col) — for use in building trigger body S-expressions */
(define replace_tblvar_with_dict (lambda (tv dict_sym expr)
	(match expr
		'((symbol get_column) tblvar _ col _)
		(if (equal? tblvar tv)
			(list 'get_assoc dict_sym col)
			expr)
		'((quote get_column) tblvar _ col _)
		(if (equal? tblvar tv)
			(list 'get_assoc dict_sym col)
			expr)
		(cons sym args) (cons sym (map args (lambda (a) (replace_tblvar_with_dict tv dict_sym a))))
		expr
	)
))

/* build_pj_insert_scan: build the nested-scan S-expression for an INSERT trigger on trigger_tv.
Skips scanning trigger_tv (its cols come from (get_assoc NEW "col") at runtime),
scans all other tables, and inserts matching rows into pj_schema/pjtbl.
pj_schema, pjtbl, mat_cols, mat_col_names are passed explicitly to avoid free-variable capture issues.
Returns an S-expression that, when wrapped in (lambda (OLD NEW session) ...) and eval'd, performs the insert. */
(define build_pj_insert_scan (lambda (scan_tables scan_condition trigger_tv is_outermost pj_schema pjtbl mat_cols mat_col_names)
	(match scan_tables
		(cons '(tblvar schema tbl isOuter joinexpr) rest)
		(if (equal? tblvar trigger_tv)
			/* skip trigger table: replace its refs in both the carried scan_condition
			and any joinexprs that still reference this table. Also fold the skipped
			stage's own joinexpr into scan_condition, otherwise that join predicate is
			lost completely when the trigger table itself is not scanned. */
			(begin
				(define rewritten_condition
					(replace_tblvar_with_dict trigger_tv 'NEW scan_condition))
				(define rewritten_joinexpr
					(if (nil? joinexpr) true
						(replace_tblvar_with_dict trigger_tv 'NEW joinexpr)))
				(define combined_condition
					(if (equal? rewritten_joinexpr true)
						rewritten_condition
						(combine_and_terms (list rewritten_condition rewritten_joinexpr))))
				(define rewritten_rest
					(map rest (lambda (td) (match td
						'(rest_tblvar rest_schema rest_tbl rest_isOuter rest_joinexpr)
						(list rest_tblvar rest_schema rest_tbl rest_isOuter
							(if (nil? rest_joinexpr) nil
								(replace_tblvar_with_dict trigger_tv 'NEW rest_joinexpr)))
						td))))
				(build_pj_insert_scan rewritten_rest
					combined_condition
					trigger_tv is_outermost pj_schema pjtbl mat_cols mat_col_names)
			)
			/* scan this other table */
			(begin
				(set cols (merge_unique (list
					(extract_columns_for_tblvar tblvar scan_condition)
					(merge_unique (map mat_cols (lambda (mc) (extract_columns_for_tblvar tblvar (cadr mc)))))
					(extract_outer_columns_for_tblvar tblvar scan_condition)
					(merge_unique (map mat_cols (lambda (mc) (extract_outer_columns_for_tblvar tblvar (cadr mc)))))
					(extract_later_joinexpr_columns_for_tblvar tblvar rest))))
				(match (split_scan_condition isOuter joinexpr scan_condition rest) '(now_condition later_condition) (begin
					(set filtercols (merge_unique (list
						(extract_columns_for_tblvar tblvar now_condition)
						(extract_outer_columns_for_tblvar tblvar now_condition))))
					(list 'scan '(session "__memcp_tx") schema tbl
						(cons 'list filtercols)
						/* filter lambda: (lambda (tv.col ...) compiled_condition) */
						(list 'lambda (map filtercols (lambda (c) (symbol (concat tblvar "." c))))
							(optimize (replace_columns_from_expr now_condition)))
						(cons 'list cols)
						/* map lambda: (lambda (tv.col ...) recursive_inner_scan) */
						(list 'lambda (map cols (lambda (c) (symbol (concat tblvar "." c))))
							(build_pj_insert_scan rest later_condition trigger_tv false pj_schema pjtbl mat_cols mat_col_names))
						/* reduce: merge */
						(list 'lambda (list 'acc 'sub) (list 'merge 'acc 'sub))
						(list)
						/* reduce2: outermost inserts into pjtbl, inner levels merge */
						(if is_outermost
							(list 'lambda (list 'acc 'shard_rows)
								(list 'insert pj_schema pjtbl (cons 'list mat_col_names) 'shard_rows (list) (list 'lambda (list) true) true))
							(list 'lambda (list 'acc 'shard_rows) (list 'merge 'acc 'shard_rows)))
						isOuter)
				))
			)
		)
		/* base case: all tables processed. Produce one row with trigger_tv cols from NEW.
		replace_columns_from_expr converts remaining (get_column ...) refs to symbol variable refs. */
		(list 'if (optimize (replace_columns_from_expr (coalesceNil scan_condition true)))
			(list 'list (cons 'list
				(map mat_cols (lambda (mc)
					(match (cadr mc)
						'((symbol get_column) tv _ col _)
						(if (equal? tv trigger_tv)
							(list 'get_assoc 'NEW col)
							(symbol (concat tv "." col)))
						'((quote get_column) tv _ col _)
						(if (equal? tv trigger_tv)
							(list 'get_assoc 'NEW col)
							(symbol (concat tv "." col)))
						/* fallback: replace trigger_tv refs and convert to symbol */
						(replace_tblvar_with_dict trigger_tv 'NEW (replace_columns_from_expr (cadr mc))))))))
			(list))
	)
))

/* build_prejoin_delete_plan: route prejoin helper row removal through the
normal DELETE planner. The old bespoke scan+$update builder duplicated DML
codegen and drifted out of sync with the shared mutation path. */
(define build_prejoin_delete_plan (lambda (pj_schema pjtbl ti_col_pairs) (begin
	(define delete_alias "_pj")
	(define delete_condition
		(if (equal? 1 (count ti_col_pairs))
			(list 'equal?
				(list 'get_column delete_alias false (car (car ti_col_pairs)) false)
				(list 'get_assoc 'OLD (cadr (car ti_col_pairs))))
			(cons 'and
				(map ti_col_pairs (lambda (p)
					(list 'equal?
						(list 'get_column delete_alias false (car p) false)
						(list 'get_assoc 'OLD (cadr p))))))))
		(build_dml_plan pj_schema pjtbl delete_alias
			(list (list delete_alias pj_schema pjtbl false nil))
			nil
			delete_condition
			nil nil nil))))

/* Derived-table flattening must not recurse blindly into opaque runtime scopes,
because those already contain lowered var/resultrow shapes. Rewrite only outer
refs inside scan filter/map lambdas so wrapped correlated scalar subselects keep
seeing the correctly prefixed outer alias. */
(define prefix_flattened_outer_ref (lambda (flatten_id inner_schemas outer_arg) (begin
	(define s (string outer_arg))
	(define parts (split s "."))
	(match parts
		(list tbl col) (if (not (nil? (inner_schemas tbl)))
			(list (quote outer) (symbol (concat flatten_id "\0" tbl "." col)))
			(list (quote outer) outer_arg))
		_ (list (quote outer) outer_arg))
)))
(define rewrite_opaque_outer_expr_for_flatten (lambda (flatten_id inner_schemas expr) (match expr
	'((symbol outer) outer_arg) (prefix_flattened_outer_ref flatten_id inner_schemas outer_arg)
	'((quote outer) outer_arg) (prefix_flattened_outer_ref flatten_id inner_schemas outer_arg)
	(cons sym args) (if (is_quote_scope_sym sym)
		expr
		(if (reduce args (lambda (found arg) (or found (opaque_expr_has_outer_ref arg))) false)
			(cons sym (map args (lambda (arg)
				(if (opaque_expr_has_outer_ref arg)
					(rewrite_opaque_outer_expr_for_flatten flatten_id inner_schemas arg)
					arg))))
			expr))
	expr
)))
(define opaque_expr_has_outer_ref (lambda (expr) (match expr
	'((symbol outer) _) true
	'((quote outer) _) true
	(cons sym args) (if (is_quote_scope_sym sym)
		false
		(reduce args (lambda (found arg) (or found (opaque_expr_has_outer_ref arg))) false))
	false
)))
(define rewrite_opaque_outer_lambda_for_flatten (lambda (flatten_id inner_schemas fn) (match fn
	'((symbol lambda) params body)
	(list (quote lambda) params (if (opaque_expr_has_outer_ref body)
		(rewrite_opaque_outer_expr_for_flatten flatten_id inner_schemas body)
		body))
	'((symbol lambda) params body numvars)
	(list (quote lambda) params (if (opaque_expr_has_outer_ref body)
		(rewrite_opaque_outer_expr_for_flatten flatten_id inner_schemas body)
		body) numvars)
	'((quote lambda) params body)
	(list (quote lambda) params (if (opaque_expr_has_outer_ref body)
		(rewrite_opaque_outer_expr_for_flatten flatten_id inner_schemas body)
		body))
	'((quote lambda) params body numvars)
	(list (quote lambda) params (if (opaque_expr_has_outer_ref body)
		(rewrite_opaque_outer_expr_for_flatten flatten_id inner_schemas body)
		body) numvars)
	fn
)))
(define rewrite_opaque_outer_alias_for_flatten (lambda (flatten_id inner_schemas expr) (match expr
	(cons (symbol !begin) forms)
	(cons (quote !begin) (map forms (lambda (form) (rewrite_opaque_outer_alias_for_flatten flatten_id inner_schemas form))))
	(cons '(quote !begin) forms)
	(cons '(quote !begin) (map forms (lambda (form) (rewrite_opaque_outer_alias_for_flatten flatten_id inner_schemas form))))
	(cons (symbol begin) forms)
	(cons (quote begin) (map forms (lambda (form) (rewrite_opaque_outer_alias_for_flatten flatten_id inner_schemas form))))
	(cons '(quote begin) forms)
	(cons '(quote begin) (map forms (lambda (form) (rewrite_opaque_outer_alias_for_flatten flatten_id inner_schemas form))))
	(cons (symbol set) (cons lhs (cons rhs tail)))
	(cons (quote set) (cons lhs (cons (rewrite_opaque_outer_alias_for_flatten flatten_id inner_schemas rhs) tail)))
	(cons '(quote set) (cons lhs (cons rhs tail)))
	(cons '(quote set) (cons lhs (cons (rewrite_opaque_outer_alias_for_flatten flatten_id inner_schemas rhs) tail)))
	(cons scanhead (cons tx (cons schema3 (cons tbl3 rest))))
	(match (string scanhead)
		"scan" (match rest
			(cons filtercols (cons filterfn (cons mapcols (cons mapfn tail))))
			(cons scanhead
				(cons tx
					(cons schema3
						(cons tbl3
							(cons filtercols
								(cons (rewrite_opaque_outer_lambda_for_flatten flatten_id inner_schemas filterfn)
									(cons mapcols
										(cons (rewrite_opaque_outer_lambda_for_flatten flatten_id inner_schemas mapfn) tail))))))))
			expr)
		"scan_order" (match rest
			(cons filtercols (cons filterfn (cons sortcols (cons sortdirs (cons sortpartcols (cons offset (cons limit (cons mapcols (cons mapfn tail)))))))))
			(cons scanhead
				(cons tx
					(cons schema3
						(cons tbl3
							(cons filtercols
								(cons (rewrite_opaque_outer_lambda_for_flatten flatten_id inner_schemas filterfn)
									(cons sortcols
										(cons sortdirs
											(cons sortpartcols
												(cons offset
													(cons limit
														(cons mapcols
															(cons (rewrite_opaque_outer_lambda_for_flatten flatten_id inner_schemas mapfn) tail)))))))))))))
			expr)
		"scan_batch" (match rest
			(cons filtercols (cons filterfn (cons mapcols (cons mapfn tail))))
			(cons scanhead
				(cons tx
					(cons schema3
						(cons tbl3
							(cons filtercols
								(cons (rewrite_opaque_outer_lambda_for_flatten flatten_id inner_schemas filterfn)
									(cons mapcols
										(cons (rewrite_opaque_outer_lambda_for_flatten flatten_id inner_schemas mapfn) tail))))))))
			expr)
		_ expr)
	expr
))))

/*
=== untangle_query: logical rewrite / Neumann decorrelation ===

Implements the algebraic unnesting transformation from Neumann/Kemper (BTW 2015)
and the holistic top-down extension (Neumann BTW 2025). Transforms a parsed SQL
query with arbitrarily nested correlated subqueries into a flat relational IR:

INPUT:  parsed query (schema tables fields condition group having order limit offset)
OUTPUT: (schema tables fields condition groups schemas replace_find_column)

The output is a single flat table list where every correlated subquery has been
replaced by a LEFT JOIN table entry. Dependencies between nesting levels are
expressed as join conditions; aggregation boundaries are expressed as group-stages
with partition-aliases (scoping). There is no nested runtime code in the output.
The IR must stay purely logical: no inner_select, subscan, or derived-source
materialization model may remain after untangle_query.

Key transformations:
- Derived tables (FROM subqueries): flattened into parent table list with column renaming
- Scalar subselects: decorrelated via unnest_subselect into LEFT JOIN + partition-stage (Path B)
or LEFT JOIN + scoped GROUP-stage (Path A for aggregates)
- IN/EXISTS/NOT IN/NOT EXISTS: rewritten to COUNT(*) aggregates, then decorrelated via Path A
- Domain column extension: Neumann Γ_{A∪D;f} — outer correlation columns added to GROUP BY
- Condition merging: WHERE and JOIN ON conditions unified into a single condition list
- Unused LEFT JOIN pruning: tables not referenced in output are eliminated

Does NOT: choose join order (join_reorder), create keytables (build_queryplan),
or generate runtime scan code (build_queryplan).
FROM (SELECT ...) must be inlined here by renaming/term replacement; aggregate
window functions without a true physical ORDER requirement also belong here as
ordinary group/keytable rewrites, not as later physical planner semantics.
*/
(define untangle_query (lambda (schema tables fields condition group having order limit offset outer_schemas_param) (begin
	(set rename_prefix (coalesce rename_prefix ""))
	(define outer_schemas_chain (coalesceNil outer_schemas_param '()))
	(define sq_cache (newsession))
	(sq_cache "init" '())

	/* COUNT(DISTINCT) rewrite helpers - do not descend into inner_select nodes (subqueries are processed separately) */
	(define _cd_is_subquery (lambda (sym) (match sym
		'inner_select true '(quote inner_select) true (symbol inner_select) true
		'inner_select_in true '(quote inner_select_in) true (symbol inner_select_in) true
		'inner_select_exists true '(quote inner_select_exists) true (symbol inner_select_exists) true
		_ false)))
	(define _cd_find (lambda (expr) (match expr
		'((symbol count_distinct) _) true
		(cons sym args) (if (_cd_is_subquery sym) false (reduce args (lambda (a b) (or a (_cd_find b))) false))
		false)))
	(define _cd_extract (lambda (expr) (match expr
		'((symbol count_distinct) e) (list e)
		(cons sym args) (if (_cd_is_subquery sym) '() (merge (map args _cd_extract)))
		'())))
	(define _cd_replace (lambda (expr) (match expr
		'((symbol count_distinct) e) '((quote aggregate) 1 (quote +) 0)
		(cons sym args) (if (_cd_is_subquery sym) expr (cons sym (map args _cd_replace)))
		expr)))
	(define _cd_has (reduce_assoc fields (lambda (a k v) (or a (_cd_find v))) false))
	/* if count_distinct present: save original having/order/limit/offset, replace fields,
	clear having/order/limit/offset (they belong to the outer/final group stage) */
	(define _cd_distinct_exprs (if _cd_has (reduce_assoc fields (lambda (a k v) (merge a (_cd_extract v))) '()) nil))
	(define _cd_having (if _cd_has having nil))
	(define _cd_order (if _cd_has order nil))
	(define _cd_limit (if _cd_has limit nil))
	(define _cd_offset (if _cd_has offset nil))
	(define _cd_user_group group)
	(define fields (if _cd_has (map_assoc fields (lambda (k v) (_cd_replace v))) fields))
	(define having (if _cd_has nil having))
	(define order (if _cd_has nil order))
	(define limit (if _cd_has nil limit))
	(define offset (if _cd_has nil offset))

	(define make_replace_find_column_subselect (lambda (schemas2 outer_schemas) (begin
		/* force optimizer to retain both params by using them directly in the outer body */
		(define _s schemas2)
		(define _o outer_schemas)
		(define alias_exists_in_schema (lambda (schemas alias_name table_insensitive) (reduce_assoc schemas (lambda (acc alias cols)
			(or acc ((if table_insensitive equal?? equal?) alias_name alias))
		) false)))
		(define column_exists_in_schema (lambda (schemas alias_name table_insensitive column_name column_insensitive) (begin
			(define matches (reduce_assoc schemas (lambda (acc alias cols)
				(if (and (or (nil? alias_name) ((if table_insensitive equal?? equal?) alias_name alias))
					(reduce cols (lambda (found coldef) (or found ((if column_insensitive equal?? equal?) (coldef "Field") column_name))) false))
					(cons alias acc)
					acc)
			) '()))
			(match matches
				'() nil
				(cons only _) only
			)
		)))
		/* wrap_outer_leaves: replace get_column leaf nodes with (outer tblvar.col) symbol references
		so that derived-table computed columns are accessible via the optimizer's outer-scope mechanism */
		(define is_get_column_sym (lambda (sym)
			(or (equal? sym (quote get_column))
				(equal? sym '(quote get_column))
				(equal? sym '(symbol get_column))
			)
		))
		/* canonical_column_in_schema: returns the Field name as stored in the schema (canonical casing) */
		(define canonical_column_in_schema (lambda (schemas alias_name table_insensitive column_name column_insensitive)
			(reduce_assoc schemas (lambda (acc alias cols)
				(if (not (nil? acc)) acc
					(if (or (nil? alias_name) ((if table_insensitive equal?? equal?) alias_name alias))
						(reduce cols (lambda (found coldef)
							(if (not (nil? found)) found
								(if ((if column_insensitive equal?? equal?) (coldef "Field") column_name) (coldef "Field") nil))) nil)
						nil))
			) nil)
		))
		(define wrap_outer_leaves (lambda (expr) (match expr
			(cons sym args) (if (is_get_column_sym sym)
				(match args
					'(tblvar ti col ci) (if (nil? tblvar) expr (begin
						(define canonical (coalesce (canonical_column_in_schema _o tblvar ti col ci) col))
						(list (quote outer) (symbol (concat tblvar "." canonical)))))
					_ (cons (wrap_outer_leaves sym) (map args wrap_outer_leaves))
				)
				(cons (wrap_outer_leaves sym) (map args wrap_outer_leaves))
			)
			expr
		)))
		(define replace_get_column_subselect (lambda (alias_name table_insensitive column_name column_insensitive expr) (begin
			(define inner_alias (column_exists_in_schema _s alias_name table_insensitive column_name column_insensitive))
			(define inner_alias_exists (and (not (nil? alias_name)) (alias_exists_in_schema _s alias_name table_insensitive)))
			(if (and inner_alias_exists (nil? inner_alias))
				(error (concat "column " alias_name "." column_name " does not exist in subquery"))
				(if (not (nil? inner_alias))
					(if (or (nil? alias_name) table_insensitive column_insensitive)
						(begin
							(define inner_column (coalesce (canonical_column_in_schema _s alias_name table_insensitive column_name column_insensitive) column_name))
							'((quote get_column) inner_alias false inner_column false))
						expr)
					(begin
						(define outer_alias (column_exists_in_schema _o alias_name table_insensitive column_name column_insensitive))
						(if (nil? outer_alias)
							(if (nil? alias_name)
								(error (concat "column " column_name " does not exist in outer query"))
								expr)
							(begin
								/* check if the outer column is a computed expression (derived table) */
								(define outer_column (coalesce (canonical_column_in_schema _o alias_name table_insensitive column_name column_insensitive) column_name))
								(define outer_cols (_o outer_alias))
								(define outer_coldef (reduce outer_cols (lambda (a coldef) (if (and (nil? a) (equal? (coldef "Field") outer_column)) coldef a)) nil))
								(define outer_expr (if outer_coldef (outer_coldef "Expr") nil))
								(if (and (not (nil? outer_expr)) (not (expr_has_opaque_scope outer_expr)))
									/* derived table computed column without opaque scopes: inline expression
									with leaf get_column nodes replaced by (outer sym) references */
									(wrap_outer_leaves outer_expr)
									/* real table column or opaque expression: symbol lookup in outer scope */
									(list (quote outer) (symbol (concat outer_alias "." outer_column))))))
					)
				)
			)
		)))
		(define replace_find_column_subselect (lambda (expr) (match expr
			(cons sym args) (if (is_get_column_sym sym)
				(match args
					'(alias_name table_insensitive column_name column_insensitive) (replace_get_column_subselect alias_name table_insensitive column_name column_insensitive expr)
					_ (cons sym (map args replace_find_column_subselect))
				)
				/* canonicalize (outer tbl.col) symbols: normalize col to schema casing */
				(if (or (equal? sym (quote outer)) (equal? sym '(quote outer)))
					(match args
						(cons outer_sym '()) (begin
							(define _ps (split (string outer_sym) "."))
							(match _ps
								(list _tbl _col) (begin
									(define _canonical (coalesce (canonical_column_in_schema _o _tbl true _col true) _col))
									(if (equal? _col _canonical) expr
										(list (if (equal? sym (quote outer)) (quote outer) sym) (symbol (concat _tbl "." _canonical)))))
								_ (cons sym (map args replace_find_column_subselect))))
						_ (cons sym (map args replace_find_column_subselect)))
					(cons sym (map args replace_find_column_subselect)))
			)
			expr
		)))
		replace_find_column_subselect
	)))

	(define build_scalar_subselect (lambda (subquery outer_schemas) (begin
		(define union_parts (query_union_all_parts subquery))
		(if (not (nil? union_parts))
			(error "scalar subselect UNION ALL is not supported yet")
			(begin
				(define raw_vals (if (and (list? subquery) (>= (count subquery) 9))
					(list (nth subquery 4) (nth subquery 5) (nth subquery 6) (nth subquery 7) (nth subquery 8))
					(list nil nil nil nil nil)
				))
				(define raw_group (nth raw_vals 0))
				(define raw_having (nth raw_vals 1))
				(define raw_order (nth raw_vals 2))
				(define raw_limit (nth raw_vals 3))
				(define raw_offset (nth raw_vals 4))
				/* pass full outer schema chain so nested subqueries inside this scalar
				subselect can still resolve grandparent references (skip-level correlation) */
				(match (apply untangle_query (merge subquery (list outer_schemas)))
					'(schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column2 _init2)
					(begin
						(define groups2 (coalesceNil groups2 '()))
						(define groups2 (if (or (nil? groups2) (equal? groups2 '()))
							(if (or raw_group raw_having raw_order raw_limit raw_offset)
								(list (make_group_stage raw_group raw_having raw_order raw_limit raw_offset nil nil))
								groups2)
							groups2))
						(define replace_find_column_subselect (make_replace_find_column_subselect schemas2 outer_schemas))
						(define field_exprs (extract_assoc fields2 (lambda (k v) v)))
						(define value_expr (match field_exprs
							(cons only '()) only
							_ (error "scalar subselect must return single column")
						))
						(set fields2 (map_assoc fields2 (lambda (k v) (replace_find_column_subselect v))))
						(set condition2 (replace_find_column_subselect (coalesceNil condition2 true)))
						/* wrap remaining unresolved qualified get_column refs as (outer tbl.col).
						These are outer-outer refs that weren't in _s or _o — wrapping them
						preserves them through replace_columns_from_expr and allows
						replace_column_alias to prefix them during derived-table flattening. */
						(define wrap_unresolved_outer (lambda (e) (match e
							'((symbol get_column) alias_ ti col ci) (if (and (not (nil? alias_)) (or ti ci)
								/* only wrap as (outer) if the alias is actually in outer_schemas;
								if not in outer_schemas either, leave as-is for scan-context resolution
								(e.g. joinexpr refs to sibling tables like v.ID) */
								(not (nil? (reduce_assoc outer_schemas (lambda (a k v) (or a (equal?? k alias_))) false))))
								(list (quote outer) (symbol (concat alias_ "." col)))
								e)
							(cons sym args) (cons (wrap_unresolved_outer sym) (map args wrap_unresolved_outer))
							e
						)))
						(set fields2 (map_assoc fields2 (lambda (k v) (wrap_unresolved_outer v))))
						(set condition2 (wrap_unresolved_outer condition2))
						(define raw_contains_skip_level_nested_outer_ref (begin
							(define raw_query_local_aliases (lambda (query) (match query
								'(_ raw_tables _ _ _ _ _ _ _) (reduce raw_tables (lambda (acc td)
									(match td
										'(alias _ _ _ _) (append_unique acc alias)
										acc))
									'())
								'())))
							(define alias_in_list (lambda (aliases alias_name)
								(reduce aliases (lambda (acc alias_) (or acc (equal?? alias_ alias_name))) false)))
							(define raw_query_uses_alias_outside_current (lambda (query current_aliases) (match query
								'(_ raw_tables raw_fields raw_condition raw_group raw_having raw_order _ _) (begin
									(define nested_local_aliases (raw_query_local_aliases query))
									(define raw_expr_uses_alias_outside_current (lambda (expr) (match expr
										'((symbol get_column) alias_ _ _ _) (and (not (nil? alias_))
											(not (alias_in_list nested_local_aliases alias_))
											(not (alias_in_list current_aliases alias_)))
										'((quote get_column) alias_ _ _ _) (and (not (nil? alias_))
											(not (alias_in_list nested_local_aliases alias_))
											(not (alias_in_list current_aliases alias_)))
										(cons sym args) (reduce args (lambda (acc arg) (or acc (raw_expr_uses_alias_outside_current arg))) false)
										false)))
									(or
										(reduce_assoc raw_fields (lambda (acc _k v) (or acc (raw_expr_uses_alias_outside_current v))) false)
										(raw_expr_uses_alias_outside_current (coalesceNil raw_condition true))
										(reduce (coalesceNil raw_group '()) (lambda (acc gexpr) (or acc (raw_expr_uses_alias_outside_current gexpr))) false)
										(raw_expr_uses_alias_outside_current (coalesceNil raw_having true))
										(reduce (coalesceNil raw_order '()) (lambda (acc order_item)
											(or acc (match order_item
												'(col _dir) (raw_expr_uses_alias_outside_current col)
												false)))
											false)))
								false)))
							(define raw_query_contains_skip_level_nested_outer_ref (lambda (query current_aliases) (match query
								'(_ _ raw_fields raw_condition raw_group raw_having raw_order _ _) (begin
									(define nested_current_aliases (append_unique current_aliases (raw_query_local_aliases query)))
									(define raw_expr_contains_skip_level_nested_outer_ref (lambda (expr) (match expr
										(cons sym args) (begin
											(define kind (inner_select_kind sym))
											(define nested_subquery (if (nil? kind) nil
												(match kind
													(quote inner_select) (match args
														(cons inner_subquery '()) inner_subquery
														nil)
													(quote inner_select_in) (match args
														(cons _target_expr (cons inner_subquery '())) inner_subquery
														nil)
													(quote inner_select_exists) (match args
														(cons inner_subquery '()) inner_subquery
														nil)
													nil)))
											(or
												(and (not (nil? nested_subquery))
													(or
														(raw_query_uses_alias_outside_current nested_subquery nested_current_aliases)
														(raw_query_contains_skip_level_nested_outer_ref nested_subquery nested_current_aliases)))
												(reduce args (lambda (acc arg) (or acc (raw_expr_contains_skip_level_nested_outer_ref arg))) false)))
										false)))
									(or
										(reduce_assoc raw_fields (lambda (acc _k v) (or acc (raw_expr_contains_skip_level_nested_outer_ref v))) false)
										(raw_expr_contains_skip_level_nested_outer_ref (coalesceNil raw_condition true))
										(reduce (coalesceNil raw_group '()) (lambda (acc gexpr) (or acc (raw_expr_contains_skip_level_nested_outer_ref gexpr))) false)
										(raw_expr_contains_skip_level_nested_outer_ref (coalesceNil raw_having true))
										(reduce (coalesceNil raw_order '()) (lambda (acc order_item)
											(or acc (match order_item
												'(col _dir) (raw_expr_contains_skip_level_nested_outer_ref col)
												false)))
											false)))
								false)))
							(raw_query_contains_skip_level_nested_outer_ref subquery (raw_query_local_aliases subquery))))
						/* Software contract: scalar aggregates are split by canonical
						correlation, not by raw parser shape.
						- uncorrelated aggregates go through the helper-table/keytable path
						and may be globally memoized
						- correlated aggregates stay on the per-row direct scan path until
						the helper-table path can safely carry row-local promises
						The correlation test therefore has to run on resolved planner
						expressions so derived-table aliases and wrapped outer refs are
						classified correctly. */
						(define value_expr_rep (car (extract_assoc fields2 (lambda (k v) v))))
						(define _is_aggregate_sym (lambda (sym)
							(or (equal? sym (quote aggregate))
								(equal? sym '(quote aggregate))
								(equal? sym '(symbol aggregate))
						)))
						(define _agg_head (match value_expr_rep (cons sym _) sym _ nil))
						(define _agg_args (if (and _agg_head (_is_aggregate_sym _agg_head))
							(match value_expr_rep (cons _ args) args _ nil)
							nil))
						(define has_stage2 (and (not (nil? groups2)) (not (equal? groups2 '()))))
						(define stage2 (if has_stage2 (car groups2) nil))
						(define stage2_group (if stage2 (coalesceNil (stage_group_cols stage2) '()) '()))
						(define stage2_having (if stage2 (stage_having_expr stage2) nil))
						(define contains_noncolumn_outer_ref (lambda (expr) (match expr
							'((quote outer) outer_sym) (equal? 1 (count (split (string outer_sym) ".")))
							'((symbol outer) outer_sym) (equal? 1 (count (split (string outer_sym) ".")))
							(cons sym args) (or (contains_noncolumn_outer_ref sym) (reduce args (lambda (a arg) (or a (contains_noncolumn_outer_ref arg))) false))
							false
						)))
						(define has_noncolumn_outer_ref (or
							(contains_noncolumn_outer_ref value_expr)
							(contains_noncolumn_outer_ref condition2)
						))
						(define contains_inner_select_marker (lambda (expr) (match expr
							(cons sym args) (or
								(not (nil? (inner_select_kind sym)))
								(contains_inner_select_marker sym)
								(reduce args (lambda (found arg) (or found (contains_inner_select_marker arg))) false))
							false)))
						(define contains_outer_ref (lambda (expr) (match expr
							'((quote outer) _) true
							'((symbol outer) _) true
							(cons sym args) (or
								(contains_outer_ref sym)
								(reduce args (lambda (found arg) (or found (contains_outer_ref arg))) false))
							false)))
						(define collapse_runtime_outer_refs (lambda (expr) (match expr
							'((quote outer) inner_expr) (match inner_expr
								(symbol inner_sym) (if (equal? 1 (count (split (string inner_sym) ".")))
									inner_expr
									expr)
								'((symbol var) _) inner_expr
								'((quote var) _) inner_expr
								'((quote outer) _) (collapse_runtime_outer_refs inner_expr)
								'((symbol outer) _) (collapse_runtime_outer_refs inner_expr)
								_ expr)
							'((symbol outer) inner_expr) (match inner_expr
								(symbol inner_sym) (if (equal? 1 (count (split (string inner_sym) ".")))
									inner_expr
									expr)
								'((symbol var) _) inner_expr
								'((quote var) _) inner_expr
								'((quote outer) _) (collapse_runtime_outer_refs inner_expr)
								'((symbol outer) _) (collapse_runtime_outer_refs inner_expr)
								_ expr)
							(cons sym args) (cons sym (map args collapse_runtime_outer_refs))
							expr)))
						(define stage_contains_outer_ref (lambda (stage)
							(or
								(reduce (coalesceNil (stage_group_cols stage) '()) (lambda (found expr) (or found (contains_outer_ref expr))) false)
								(contains_outer_ref (coalesceNil (stage_post_group_condition_expr stage) true))
								(reduce (coalesceNil (stage_order_list stage) '()) (lambda (found order_item)
									(or found (match order_item
										'(col _dir) (contains_outer_ref col)
										(contains_outer_ref order_item)))) false))))
						(define scalar_has_outer_ref (or
							(reduce_assoc fields2 (lambda (found _k v) (or found (contains_outer_ref v))) false)
							(contains_outer_ref condition2)
							(reduce (coalesceNil groups2 '()) (lambda (found stage) (or found (stage_contains_outer_ref stage))) false)))
						(define use_ordered_scalar (or
							(and has_stage2 (not (equal? (coalesceNil (stage_order_list stage2) '()) '())))
							(and has_stage2 (not (nil? (stage_limit_val stage2))))
							(and has_stage2 (not (nil? (stage_offset_val stage2))))
						))
						(define use_direct_agg_scan (and
							(not (nil? _agg_args))
							(equal? (count _agg_args) 3)
							(nil? stage2_post_group_condition)
							(or (nil? stage2_group) (equal? stage2_group '()) (equal? stage2_group '(1)))
							(not (nil? tables2))
							(not (equal? tables2 '()))
							scalar_has_outer_ref
						))
						(define use_direct_scalar_scan (and
							(not use_direct_agg_scan)
							(not raw_contains_skip_level_nested_outer_ref)
							(equal? (extract_aggregates value_expr) '())
							(not (contains_inner_select_marker condition2))
							(not (contains_inner_select_marker value_expr))
							(not has_noncolumn_outer_ref)
							(nil? stage2_post_group_condition)
							(or (nil? stage2_group) (equal? stage2_group '()) (equal? stage2_group '(1)))
							(and (list? tables2) (equal? (count tables2) 1))
						))
						(define build_scalar_subselect_fallback (lambda () (begin
							(define _sq_hash (fnv_hash (concat tables2 "|" fields2 "|" condition2)))
							(define _sq_promise_name (concat "__scalar_promise_" _sq_hash))
							(define _sq_rr_name (concat "__scalar_resultrow_" _sq_hash))
							(begin
								(define replace_resultrow (lambda (expr) (match expr
									(cons sym args) (if (equal? sym (quote resultrow))
										(cons (symbol _sq_rr_name) (map args replace_resultrow))
										(if (and (equal? sym (quote symbol)) (equal? args '("resultrow")))
											(list (quote symbol) _sq_rr_name)
											(cons (replace_resultrow sym) (map args replace_resultrow))
										)
									)
									expr
								)))
								(define subplan (replace_resultrow (build_queryplan schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column_subselect nil)))
								(define _init_stmts (if (or (nil? _init2) (equal? _init2 '())) '() _init2))
								(cons (quote !begin) (merge _init_stmts (list
									(list (quote set) (symbol _sq_promise_name) (list (quote newpromise)))
									(list (quote set) (symbol _sq_rr_name)
										(list (quote lambda) (list (symbol "row"))
											(list (symbol _sq_promise_name) "once"
												(list (quote nth) (symbol "row") 1)
												"scalar subselect returned more than one row")
										)
									)
									subplan
									(list (symbol _sq_promise_name) "value")
								)))
							)
						)))
						(if use_direct_agg_scan
							(begin
								(define agg_item (nth _agg_args 0))
								(define agg_reduce (nth _agg_args 1))
								(define agg_neutral (nth _agg_args 2))
										(define build_scalar_agg_scan (lambda (scan_tables scan_condition)
											(match scan_tables
												(cons '(tblvar schema3 tbl3 isOuter3 joinexpr3) rest_tables) (begin
													(define cur_cols (merge_unique (list
														(extract_columns_for_tblvar tblvar scan_condition)
														(extract_columns_for_tblvar tblvar agg_item)
														(extract_outer_columns_for_tblvar tblvar scan_condition)
														(extract_outer_columns_for_tblvar tblvar agg_item)
												(extract_later_joinexpr_columns_for_tblvar tblvar rest_tables)
											)))
													(match (split_scan_condition isOuter3 joinexpr3 scan_condition rest_tables) '(now_condition later_condition) (begin
														(define filtercols (merge_unique (list
															(extract_columns_for_tblvar tblvar now_condition)
															(extract_outer_columns_for_tblvar tblvar now_condition)
														)))
														(define inner_body (build_scalar_agg_scan rest_tables later_condition))
														(define filterbody (collapse_runtime_outer_refs (replace_columns_from_expr now_condition)))
														(scan_wrapper 'scan schema3 tbl3
															(cons list filtercols)
															(list (quote lambda)
																(map filtercols (lambda (col) (symbol (concat tblvar "." col))))
																filterbody
															)
															(cons list cur_cols)
															(list (quote lambda)
																(map cur_cols (lambda (col) (symbol (concat tblvar "." col))))
																inner_body
													)
													(eval agg_reduce) agg_neutral (eval agg_reduce) isOuter3
												)
											))
										)
												'() (collapse_runtime_outer_refs (replace_columns_from_expr agg_item))
											)
										))
								(define _init_stmts_agg (if (or (nil? _init2) (equal? _init2 '())) '() _init2))
								(if (equal? _init_stmts_agg '())
									(build_scalar_agg_scan tables2 condition2)
									(cons (quote !begin) (merge _init_stmts_agg (list (build_scalar_agg_scan tables2 condition2)))))
							)
							(if use_direct_scalar_scan
								(begin
									(match (car tables2) '(tblvar schema3 tbl3 isOuter3 joinexpr3) (begin
										(if (not (nil? joinexpr3))
											(error "scalar subselect joins not supported in direct scalar scan"))
										(define stage2_order (if has_stage2 (coalesceNil (stage_order_list stage2) '()) '()))
										(define stage2_limit (if has_stage2 (stage_limit_val stage2) nil))
										(define stage2_offset (if has_stage2 (stage_offset_val stage2) nil))
										(define filtercols (merge_unique (list
											(extract_columns_for_tblvar tblvar condition2)
											(extract_outer_columns_for_tblvar tblvar condition2)
										)))
										(define mapcols (merge_unique (list
											(extract_columns_for_tblvar tblvar value_expr)
											(extract_outer_columns_for_tblvar tblvar value_expr)
										)))
										(define ordercols (merge (map stage2_order (lambda (order_item) (match order_item '(col dir) (match col
											'((symbol get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list col) '())
											'((quote get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list col) '())
											_ '()
										))))))
										(define dirs (merge (map stage2_order (lambda (order_item) (match order_item '(col dir) (match col
											'((symbol get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar) (list dir) '())
											'((quote get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar) (list dir) '())
											_ '()
										))))))
										(if (and use_ordered_scalar (not (equal? stage2_order '())) (not (equal? (count ordercols) (count stage2_order))))
											(error "scalar subselect ORDER BY must use direct columns"))
										(define wrap_generated_outer_refs_scalar (lambda (expr local_params) (match expr
											'((quote outer) inner_expr) (collapse_runtime_outer_refs expr)
											'((symbol outer) inner_expr) (collapse_runtime_outer_refs expr)
											(cons sym args) (cons sym (map args (lambda (arg) (wrap_generated_outer_refs_scalar arg local_params))))
											sym (begin
												/* Correlated direct scalar scans run inside the enclosing scan lambda.
												Leave dotted row symbols as lexical captures instead of wrapping them
												as (outer ...), because group/createcolumn scan bodies introduce their
												own outer scope for group keys. */
												sym)
											expr
										)))
										(define filterparams (map filtercols (lambda (col) (symbol (concat tblvar "." col)))))
										(define mapparams (map mapcols (lambda (col) (symbol (concat tblvar "." col)))))
										(define filterbody (wrap_generated_outer_refs_scalar (optimize (replace_columns_from_expr (coalesceNil condition2 true))) filterparams))
										(define valuebody (wrap_generated_outer_refs_scalar (optimize (replace_columns_from_expr value_expr)) mapparams))
										(define direct_has_noncolumn_outer_ref (or
											(contains_noncolumn_outer_ref filterbody)
											(contains_noncolumn_outer_ref valuebody)
										))
										(define direct_has_runtime_outer_ref (or
											(contains_outer_ref filterbody)
											(contains_outer_ref valuebody)
										))
										(if (or direct_has_noncolumn_outer_ref direct_has_runtime_outer_ref)
											(build_scalar_subselect_fallback)
											(begin
												(define _sq_hash (fnv_hash (concat tables2 "|" fields2 "|" condition2)))
												(define _sq_promise_name (concat "__scalar_promise_" _sq_hash))
												(define _init_stmts (if (or (nil? _init2) (equal? _init2 '())) '() _init2))
												(cons (quote !begin) (merge _init_stmts (list
													(list (quote set) (symbol _sq_promise_name) (list (quote newpromise)))
													(if use_ordered_scalar
														(list (quote scan_order)
															(list (quote session) "__memcp_tx")
															schema3
															(scan-runtime-source tbl3)
															(cons list filtercols)
															(list (quote lambda) filterparams filterbody)
															(cons list ordercols)
															(cons list dirs)
															0
															(coalesceNil stage2_offset 0)
															(coalesceNil stage2_limit 1)
															(cons list mapcols)
															(list (quote lambda) mapparams
																(list (symbol _sq_promise_name) "once" valuebody "scalar subselect returned more than one row"))
															nil
															nil
															false)
														(list (quote scan)
															(list (quote session) "__memcp_tx")
															schema3
															(scan-runtime-source tbl3)
															(cons list filtercols)
															(list (quote lambda) filterparams filterbody)
															(cons list mapcols)
															(list (quote lambda) mapparams
																(list (symbol _sq_promise_name) "once" valuebody "scalar subselect returned more than one row"))
															nil
															nil
															false))
													(list (symbol _sq_promise_name) "value")
												)))
										))
								)))
								(build_scalar_subselect_fallback))
						)
					)
				)
			)
		)
	)
	))
	(define build_exists_subselect (lambda (subquery outer_schemas) (match subquery
		'(schema2 tables2 fields2 condition2 group2 having2 order2 limit2 offset2)
		(list (quote coalesceNil)
			(build_scalar_subselect
				(list schema2 tables2
					(list "__exists" true)
					condition2 group2 having2 order2 (coalesceNil limit2 1) offset2)
				outer_schemas)
			false)
		false
	)))

	/* unnest_subselect: core Neumann decorrelation for a single subquery.
	Transforms a correlated scalar subquery into a LEFT JOIN table entry,
	eliminating the dependent join. Returns (substitution tables) or nil.

	Three paths based on subquery shape:
	Path A (aggregate): adds domain columns to GROUP BY (Neumann Γ_{A∪D}),
	flattens inner tables with scoped GROUP-stage. Handles COUNT/SUM/AVG/etc.
	Path B (non-agg + LIMIT): creates partition-stage for LIMIT per outer row,
	direct LEFT JOIN table entry. Handles ORDER BY + LIMIT 1 pattern.
	Path C (non-agg, no LIMIT): returns nil (fallback to inline evaluation).

	Recursive nesting: inner subqueries are decorrelated first by untangle_query.
	Their tables become "inner-scoped" (identified via partition-aliases) and are
	passed through to the outer level with joinexpr rewriting. Dependencies on
	tables outside the current scope stay as bare get_column references. */
	(define unnest_subselect (lambda (subquery outer_schemas) (begin
		(define union_parts_us (query_union_all_parts subquery))
		(if (not (nil? union_parts_us))
			nil /* UNION ALL not handled yet */
			(begin
				(define raw_vals_us (if (and (list? subquery) (>= (count subquery) 9))
					(list (nth subquery 4) (nth subquery 5) (nth subquery 6) (nth subquery 7) (nth subquery 8))
					(list nil nil nil nil nil)))
				(define raw_group_us (nth raw_vals_us 0))
				(define raw_having_us (nth raw_vals_us 1))
				(define raw_order_us (nth raw_vals_us 2))
				(define raw_limit_us (nth raw_vals_us 3))
				(define raw_offset_us (nth raw_vals_us 4))
				/* pass outer_schemas chain to recursive untangle so grandparent refs resolve */
				(match (apply untangle_query (merge subquery (list outer_schemas)))
					'(schema2_us tables2_us fields2_us condition2_us groups2_us schemas2_us rfcol2_us _init2_us) (begin
						(if (and (not (nil? _init2_us)) (not (equal? _init2_us '())))
							(sq_cache "init" (merge (coalesceNil (sq_cache "init") '()) _init2_us)))
						/* no-table subselect without aggregates: return field expression directly */
						(if (and (or (nil? tables2_us) (equal? tables2_us '()))
							(not (reduce_assoc fields2_us (lambda (a k v) (or a
								(begin (define _nta (lambda (e) (match e (cons (symbol aggregate) _) true (cons s args) (reduce args (lambda (a2 b) (or a2 (_nta b))) false) false))) (_nta v)))) false)))
							(list (car (extract_assoc fields2_us (lambda (k v) v))) '())
							(begin
								/* no-table with aggregates: inject virtual "(1)" one-row table.
								Only mutate tables2_us and schemas2_us — groups2_us is set below. */
								(define _nt_virtual_init (list (quote begin)
									(list (quote createtable) schema2_us "(1)"
										(list (list "unique" "group" (list "1")) (list "column" "1" "any" '() '()))
										(list "engine" "sloppy") true)
									(list (quote insert) schema2_us "(1)" (list "1") (list (list 1)) '() (list (quote lambda) '() true) true)))
								(if (or (nil? tables2_us) (equal? tables2_us '()))
									(begin
										(set tables2_us (list (list "(1)" schema2_us "(1)" false nil)))
										(set schemas2_us (list "(1)" (list (list "Field" "1" "Type" "any"))))))
								(define groups2_us (coalesceNil groups2_us '()))
								(define groups2_us (if (or (nil? groups2_us) (equal? groups2_us '()))
									(if (or raw_group_us raw_having_us raw_order_us raw_limit_us raw_offset_us)
										(list (make_group_stage raw_group_us raw_having_us raw_order_us raw_limit_us raw_offset_us nil _nt_virtual_init))
										groups2_us)
									groups2_us))
								/* resolve columns against inner and outer schemas */
								(define rfcs_us (make_replace_find_column_subselect schemas2_us outer_schemas))
								(set fields2_us (map_assoc fields2_us (lambda (k v) (rfcs_us v))))
								(set condition2_us (rfcs_us (coalesceNil condition2_us true)))
								/* wrap remaining unresolved qualified refs as (outer tbl.col) */
								(define _us_wrap (lambda (e) (match e
									'((symbol get_column) alias_ ti col ci) (if (and (not (nil? alias_)) (or ti ci))
										(list (quote outer) (symbol (concat alias_ "." col)))
										e)
									(cons sym args) (cons (_us_wrap sym) (map args _us_wrap))
									e)))
								(set fields2_us (map_assoc fields2_us (lambda (k v) (_us_wrap v))))
								(set condition2_us (_us_wrap condition2_us))
								/* extract all outer references from fields and condition.
								Detects both explicit (outer tbl.col) AND bare (get_column tbl false col false)
								where tbl is NOT in the inner table set (from nested unnesting).
								Skip opaque scopes (!begin, scan, etc.). */
								(define us_inner_aliases (map tables2_us (lambda (td) (match td '(a _ _ _ _) a ""))))
								(define _us_eor (lambda (expr) (match expr
									(cons sym args) (if (or (equal? sym (quote outer)) (equal? sym '(quote outer)))
										(match args (cons sym_arg '()) (list (string sym_arg)) '())
										(if (or (equal? sym (quote get_column)) (equal? sym '(quote get_column)) (equal? sym '(symbol get_column)))
											(match args '(alias_ _ col _) (if (and (not (nil? alias_)) (not (reduce us_inner_aliases (lambda (a ia) (or a (equal?? ia alias_))) false)))
												(list (concat alias_ "." col)) '())
												'())
											(if (_is_opaque_scope_sym sym) '()
												(merge_unique (map args _us_eor)))))
									'())))
								(define us_outer_refs (merge_unique
									(merge (extract_assoc fields2_us (lambda (k v) (_us_eor v))))
									(_us_eor condition2_us)))
								/* feasibility checks */
								(define us_has_outer (not (equal? us_outer_refs '())))
								/* separate own stages from inner scoped stages (from nested decorrelation) —
								must be defined BEFORE _us_inner_aliases which depends on _us_inner_stages */
								(define _us_own_stages (filter (coalesceNil groups2_us '()) (lambda (s) (nil? (stage_partition_aliases s)))))
								(define _us_inner_stages (filter (coalesceNil groups2_us '()) (lambda (s) (not (nil? (stage_partition_aliases s))))))
								/* count only OWN tables (not inner scoped ones from nested decorrelation) */
								(define _us_inner_aliases (merge (map _us_inner_stages (lambda (s) (coalesceNil (stage_partition_aliases s) '())))))
								(define _us_own_tables (filter tables2_us (lambda (t) (match t '(a _ _ _ _) (not (has? _us_inner_aliases a)) true))))
								(define us_single_tbl (and (list? _us_own_tables) (equal? (count _us_own_tables) 1)))
								/* check for aggregates in fields */
								(define _us_agg (lambda (expr) (match expr
									'((symbol aggregate) _ _ _) true
									(cons sym args) (reduce args (lambda (a b) (or a (_us_agg b))) false)
									false)))
								(define us_has_agg (reduce_assoc fields2_us (lambda (a k v) (or a (_us_agg v))) false))
								/* check for GROUP/HAVING in OWN stages only */
								(define us_has_stages (not (equal? _us_own_stages '())))
								(define us_has_grp (if us_has_stages
									(reduce _us_own_stages (lambda (acc stage) (or acc
										(begin
											(define g (stage_group_cols stage))
											(or (and (not (nil? g)) (not (equal? g '())) (not (equal? g '(1))))
												(not (nil? (stage_having_expr stage))))))) false)
									false))
								/* check for LIMIT/ORDER/OFFSET stages — deferred until 1-row constraint handling */
								(define us_has_limit (if us_has_stages
									(reduce _us_own_stages (lambda (acc stage) (or acc
										(not (nil? (stage_limit_val stage)))
										(not (nil? (stage_offset_val stage)))
										(begin
											(define o (coalesceNil (stage_order_list stage) '()))
											(and (not (nil? o)) (not (equal? o '())))))) false)
									false))
								/* check for outer refs in fields (not just condition) — these need
								more complex handling, fall back for now */
								(define us_outer_in_fields (not (equal?
									(merge (extract_assoc fields2_us (lambda (k v) (_us_eor v)))) '())))
								(if us_outer_in_fields nil /* outer refs in fields: not handled yet */
									(begin
										/* === Neumann unnesting: nD domain, single or multi-table === */
										/* generate unique alias using fnv_hash to avoid collisions across nesting levels */
										(define us_sq_idx (coalesceNil (sq_cache "idx") 0))
										(sq_cache "idx" (+ us_sq_idx 1))
										(define _us_own_tblname (match (car _us_own_tables) '(_ _ t _ _) (if (string? t) t "x") "x"))
										(define us_sq_prefix (concat "_unn_" _us_own_tblname "_" us_sq_idx))
										/* build alias rename map: only OWN tables get prefixed.
										Inner-scoped tables (from nested decorrelation) keep their alias. */
										(define us_alias_map (map _us_own_tables (lambda (td) (match td
											'(alias _ _ _ _) (list alias (if us_single_tbl us_sq_prefix (concat us_sq_prefix "\0" alias)))
											(list "" "")))))
										(define _us_lookup (lambda (a) (reduce us_alias_map (lambda (acc p) (if (nil? acc) (if (equal?? a (nth p 0)) (nth p 1) nil) acc)) nil)))
										/* value column and its source alias */
										(define us_value_key (car (extract_assoc fields2_us (lambda (k v) k))))
										(define us_value_expr (car (extract_assoc fields2_us (lambda (k v) v))))
										(define us_value_src (match us_value_expr '((symbol get_column) a _ _ _) a '((quote get_column) a _ _ _) a nil))
										(define us_value_new (if (nil? us_value_src) us_sq_prefix (coalesceNil (_us_lookup us_value_src) us_sq_prefix)))
										/* helper: does expr contain outer refs? Detects both (outer ...) and
										bare get_column refs to non-inner tables. Skip opaque scopes. */
										(define _us_hor (lambda (expr) (match expr
											(cons sym args) (if (or (equal? sym (quote outer)) (equal? sym '(quote outer)))
												true
												(if (or (equal? sym (quote get_column)) (equal? sym '(quote get_column)) (equal? sym '(symbol get_column)))
													(match args '(alias_ _ _ _) (and (not (nil? alias_)) (not (reduce us_inner_aliases (lambda (a ia) (or a (equal?? ia alias_))) false))) false)
													(if (_is_opaque_scope_sym sym) false
														(reduce args (lambda (a b) (or a (_us_hor b))) false))))
											false)))
										/* split condition into AND-parts */
										(define _us_fap (lambda (expr) (match expr
											(cons sym parts) (if (or (equal? sym (quote and)) (equal? sym '(quote and)) (equal? sym 'and))
												(merge (map parts _us_fap))
												(list expr))
											(list expr))))
										(define us_cond_parts (_us_fap condition2_us))
										(define us_inner_parts (filter us_cond_parts (lambda (p) (not (_us_hor p)))))
										(define us_outer_parts (filter us_cond_parts (lambda (p) (_us_hor p))))
										/* resolve (outer tbl.col) -> (get_column tbl false col false).
										Runtime resolves bare symbols via scope chain (multi-level lookup). */
										(define _us_ror (lambda (expr) (match expr
											(cons sym args) (if (or (equal? sym (quote outer)) (equal? sym '(quote outer)))
												(match args
													(cons sym_arg '()) (begin
														(define ps (split (string sym_arg) "."))
														(match ps
															(list tbl col) (list (quote get_column) tbl false col false)
															_ expr))
													_ expr)
												(cons (_us_ror sym) (map args _us_ror)))
											expr)))
										/* replace ANY inner alias with its renamed version */
										(define _us_ria (lambda (expr) (match expr
											'((symbol get_column) alias_ ti col ci) (begin
												(define na (_us_lookup alias_))
												(if (nil? na) expr (list (quote get_column) na false col false)))
											'((quote get_column) alias_ ti col ci) (begin
												(define na (_us_lookup alias_))
												(if (nil? na) expr (list (quote get_column) na false col false)))
											(cons sym args) (cons (_us_ria sym) (map args _us_ria))
											expr)))
										/* inner condition (non-correlated) - kept with original aliases for
										aggregate path; renamed for non-aggregate path */
										(define us_inner_cond_raw (if (equal? (count us_inner_parts) 0) nil
											(if (equal? (count us_inner_parts) 1) (car us_inner_parts)
												(cons (quote and) us_inner_parts))))
										/* extract domain columns from correlated equalities:
										(equal?? inner_expr outer_expr) → (inner_expr resolved_outer_expr) */
										(define us_domain_cols (filter (map us_outer_parts (lambda (part) (match part
											'((symbol equal??) a b) (if (_us_hor a) (if (not (_us_hor b)) (list b (_us_ror a)) nil) (if (_us_hor b) (list a (_us_ror b)) nil))
											'((quote equal??) a b) (if (_us_hor a) (if (not (_us_hor b)) (list b (_us_ror a)) nil) (if (_us_hor b) (list a (_us_ror b)) nil))
											nil))) (lambda (x) (not (nil? x)))))
										/* === Three-way branch: aggregate / non-agg+LIMIT / non-agg-no-LIMIT === */
										(if (or us_has_agg us_has_grp)
											/* === A: Aggregate → flatten inner tables + scoped GROUP stage ===
											Neumann Γ_{A∪D;f}: add domain cols to GROUP BY, flatten inner tables
											with prefix into outer table list. No materialization. */
											(begin
												/* rename inner aliases: alias → _sq0\0alias (recursive in all exprs) */
												(define _us_prefix_ria (lambda (expr) (match expr
													'((symbol get_column) alias_ ti col ci) (begin
														(define na (_us_lookup alias_))
														(if (nil? na) expr (list (quote get_column) na false col false)))
													'((quote get_column) alias_ ti col ci) (begin
														(define na (_us_lookup alias_))
														(if (nil? na) expr (list (quote get_column) na false col false)))
													(cons sym args) (cons (_us_prefix_ria sym) (map args _us_prefix_ria))
													expr)))
												/* prefix inner tables: alias → _sq0\0alias, tbl stays string */
												(define us_prefixed_tables (map tables2_us (lambda (td) (match td
													'(a s t io je) (list (coalesceNil (_us_lookup a) a) s t io
														(if (nil? je) nil (_us_prefix_ria je)))
													td))))
												/* inner condition (non-correlated), prefixed */
												(define us_inner_cond_prefixed (if (nil? us_inner_cond_raw) nil (_us_prefix_ria us_inner_cond_raw)))
												/* domain columns + original GROUP BY → scoped GROUP stage */
												(define us_orig_group (if us_has_stages (coalesceNil (stage_group_cols (car _us_own_stages)) '()) '()))
												(define us_orig_having (if us_has_stages (stage_having_expr (car _us_own_stages)) nil))
												(define _us_dom_group_cols (map us_domain_cols (lambda (dc) (_us_prefix_ria (nth dc 0)))))
												(define us_new_group (merge _us_dom_group_cols
													(if (or (equal? us_orig_group '()) (equal? us_orig_group '(1)))
														/* keep (1) for static aggregation if no domain cols */
														(if (equal? _us_dom_group_cols '()) us_orig_group '())
														(map us_orig_group _us_prefix_ria))))
												(define us_new_having (if (nil? us_orig_having) nil (_us_prefix_ria us_orig_having)))
												/* Domain D is exactly the free/unbound outer columns. Only
												correlated aggregates need a scoped GROUP stage over the
												prefixed inner aliases; for D = ∅ this must stay a global
												aggregate helper relation. */
												(define us_inner_aliases (map us_prefixed_tables (lambda (td) (match td '(a _ _ _ _) a ""))))
												(define us_stage_aliases (if (equal? _us_dom_group_cols '()) nil us_inner_aliases))
												/* preserve ORDER+LIMIT only for explicit GROUP BY subselects.
												For pure aggregates (Neumann domain extension), the LIMIT refers
												to the inner result per outer row, not the keytable total. */
												(define us_orig_order_a (if (and us_has_grp us_has_stages) (coalesceNil (stage_order_list (car _us_own_stages)) '()) '()))
												(define us_orig_limit_a (if (and us_has_grp us_has_stages) (stage_limit_val (car _us_own_stages)) nil))
												(define us_orig_offset_a (if (and us_has_grp us_has_stages) (stage_offset_val (car _us_own_stages)) nil))
												(define us_new_order (map us_orig_order_a (lambda (oi) (match oi '(col dir) (list (_us_prefix_ria col) dir) oi))))
												(define us_group_stage
													(stage_with_cache_query
														(stage_with_cache_policy
															(make_group_stage us_new_group us_new_having us_new_order us_orig_limit_a us_orig_offset_a us_stage_aliases nil)
															(count_subquery_cache_policy subquery))
														(if (nil? (count_subquery_cache_policy subquery)) nil subquery)))
												/* propagate inner scoped stages with prefix */
												(define _us_prefixed_inner_stages (map _us_inner_stages (lambda (s) (begin
													(define _psg (map (coalesceNil (stage_group_cols s) '()) _us_prefix_ria))
													(define _psh (if (nil? (stage_having_expr s)) nil (_us_prefix_ria (stage_having_expr s))))
													(define _pso (map (coalesceNil (stage_order_list s) '()) (lambda (o) (match o '(c d) (list (_us_prefix_ria c) d) o))))
													(define _psa (map (coalesceNil (stage_partition_aliases s) '()) (lambda (a) (coalesceNil (_us_lookup a) a))))
													(stage_preserve_cache_meta s
														(make_group_stage _psg _psh _pso (stage_limit_val s) (stage_offset_val s) _psa (stage_init_code s)))))))
												/* register prefixed tables */
												(sq_cache "tables" (merge us_prefixed_tables (coalesceNil (sq_cache "tables") '())))
												/* register scoped GROUP stage + propagated inner stages */
												(sq_cache "groups" (merge (list us_group_stage) _us_prefixed_inner_stages (coalesceNil (sq_cache "groups") '())))
												/* register schemas for prefixed aliases */
												(define us_prefixed_schemas (merge (map us_prefixed_tables (lambda (td) (match td
													'(a _ _ _ _) (begin
														(define _orig_a (reduce us_alias_map (lambda (acc p) (if (nil? acc) (if (equal? (nth p 1) a) (nth p 0) nil) acc)) nil))
														(define _s_cols (if (nil? _orig_a) nil (schemas2_us _orig_a)))
														(if (nil? _s_cols) '() (list a _s_cols)))
													'())))))
												(sq_cache "schemas" (merge us_prefixed_schemas (coalesceNil (sq_cache "schemas") '())))
												/* join condition: domain equalities (outer_expr = prefixed_inner_expr) */
												(define us_dom_je_parts (map us_domain_cols (lambda (dc)
													(list (quote equal??) (_us_prefix_ria (nth dc 0)) (nth dc 1)))))
												(define us_dom_je (if (equal? (count us_dom_je_parts) 0) true
													(if (equal? (count us_dom_je_parts) 1) (car us_dom_je_parts)
														(cons (quote and) us_dom_je_parts))))
												/* distribute inner condition parts per table joinexpr.
												Domain equalities go on the first table. Each inner condition
												part goes on the last-referenced table in join order. */
												(define _us_inner_parts_list (if (nil? us_inner_cond_prefixed) '()
													(match us_inner_cond_prefixed
														(cons (symbol and) parts) parts
														(cons (quote and) parts) parts
														(list us_inner_cond_prefixed))))
												/* helper: extract tblvar refs from expression */
												(define _us_expr_refs (lambda (expr) (match expr
													'((symbol get_column) tv _ _ _) (if (nil? tv) '() (list tv))
													'((quote get_column) tv _ _ _) (if (nil? tv) '() (list tv))
													(cons _ args) (reduce args (lambda (acc a) (merge acc (_us_expr_refs a))) '())
													'())))
												(define _us_alias_list (map us_prefixed_tables (lambda (td) (match td '(a _ _ _ _) a ""))))
												/* find the last alias (in join order) referenced by a condition part */
												(define _us_last_alias (lambda (part) (begin
													(define _refs (_us_expr_refs part))
													/* walk alias list; remember last alias that appears in _refs */
													(reduce _us_alias_list (lambda (best al)
														(if (reduce _refs (lambda (found r) (or found (equal?? r al))) false)
															al best)) nil))))
												/* collect parts assigned to a given alias */
												(define _us_parts_for (lambda (alias) (begin
													(define _my (filter _us_inner_parts_list (lambda (p) (equal?? (_us_last_alias p) alias))))
													(if (equal? (count _my) 0) nil
														(if (equal? (count _my) 1) (car _my)
															(cons (quote and) _my))))))
												/* set joinexpr per inner table */
												(if (not (nil? us_prefixed_tables))
													(sq_cache "tables" (begin
														(define _all_tbls (sq_cache "tables"))
														(define _first_alias (match (car us_prefixed_tables) '(a _ _ _ _) a ""))
														(map _all_tbls (lambda (td) (match td
															'(a s t io je) (if (not (reduce _us_alias_list (lambda (f al) (or f (equal?? al a))) false)) td
																(begin
																	(define _my_cond (_us_parts_for a))
																	(if (equal? a _first_alias)
																		(list a s t true (if (nil? _my_cond) us_dom_je
																			(if (equal? us_dom_je true) _my_cond
																				(list (quote and) us_dom_je _my_cond))))
																		(list a s t io (if (nil? _my_cond) je
																			(if (nil? je) _my_cond
																				(list (quote and) je _my_cond)))))))
															td))))))
												/* substitution: reference the prefixed value column */
												(define us_subst_raw (_us_prefix_ria us_value_expr))
												(define us_is_count (match us_value_expr
													'((symbol aggregate) _ (symbol +) 0) true
													'((quote aggregate) _ (symbol +) 0) true
													'((quote aggregate) _ '(symbol +) 0) true
													false))
												(define us_subst (if us_is_count (list (quote coalesceNil) us_subst_raw 0) us_subst_raw))
												/* return substitution + empty table entries (tables already in sq_cache) */
												(list us_subst '()))
											/* === B/C: Non-aggregate === */
											(begin
												/* value must be a simple column (not computed expression) for direct table entry */
												(if us_single_tbl
													/* === B/C: Non-agg → direct LEFT JOIN table entry ===
													Path B (has LIMIT): adds partition-stage for ORDER BY + LIMIT per outer row.
													Path C (no LIMIT): plain LEFT JOIN, no partition-stage. */
													(begin
														(define us_tdesc (car tables2_us))
														(define us_tblvar (nth us_tdesc 0))
														(define us_tbl_schema (nth us_tdesc 1))
														(define us_tbl_name (nth us_tdesc 2))
														(define us_orig_order (if us_has_stages (coalesceNil (stage_order_list (car _us_own_stages)) '()) '()))
														(define us_orig_limit (if us_has_stages (stage_limit_val (car _us_own_stages)) nil))
														(define us_orig_offset (if us_has_stages (stage_offset_val (car _us_own_stages)) nil))
														/* pass through inner-scoped tables (from nested decorrelation) with joinexpr rewriting */
														(define _us_inner_tbls (filter tables2_us (lambda (t) (match t '(a _ _ _ _) (has? _us_inner_aliases a) false))))
														(define _us_inner_tbls_rewritten (map _us_inner_tbls (lambda (td) (match td
															'(a s t io je) (list a s t io (if (nil? je) nil (_us_ria je)))
															td))))
														(if (not (equal? _us_inner_tbls_rewritten '()))
															(sq_cache "tables" (merge _us_inner_tbls_rewritten (coalesceNil (sq_cache "tables") '()))))
														/* Always register partition stage: Path B uses explicit LIMIT,
														Path C uses implicit LIMIT 1 (scalar subselect = at most one row) */
														(begin
															/* domain order: only include cols that reference the own table after rename.
															Indirect correlations (through inner-scoped tables) are handled by their
															own partition-stages and must not appear here (wrong table → crash). */
															(define us_dom_order (filter (map us_domain_cols (lambda (dc) (list (_us_ria (nth dc 0)) '<)))
																(lambda (oi) (match oi '(col _) (match col
																	'((symbol get_column) a _ _ _) (equal? a us_sq_prefix)
																	'((quote get_column) a _ _ _) (equal? a us_sq_prefix)
																	false) false))))
															(define us_renamed_order (map (coalesceNil us_orig_order '()) (lambda (oi) (match oi '(col dir) (list (_us_ria col) dir) oi))))
															(define us_part_order (merge us_dom_order us_renamed_order))
															(define us_dom_count (count us_dom_order))
															/* register partition stage: (a) own-table sort cols for correlated LIMIT,
															(b) explicit LIMIT from SQL, (c) uncorrelated needs global limit=1.
															Indirect-only correlations skip: join chain guarantees 1 row. */
															(if (or (not (equal? us_part_order '())) us_has_limit (not us_has_outer)) (begin
																(define us_part_stage (make_partition_stage (list us_sq_prefix) us_part_order us_dom_count (coalesceNil us_orig_limit 1) (coalesceNil us_orig_offset 0) nil))
																(sq_cache "partition_stages" (cons us_part_stage (coalesceNil (sq_cache "partition_stages") '()))))))
														/* propagate inner scoped stages with renaming */
														(if (not (equal? _us_inner_stages '()))
															(sq_cache "groups" (merge
																(map _us_inner_stages (lambda (s) (begin
																	(define _psg (map (coalesceNil (stage_group_cols s) '()) _us_ria))
																	(define _psh (if (nil? (stage_having_expr s)) nil (_us_ria (stage_having_expr s))))
																	(define _pso (map (coalesceNil (stage_order_list s) '()) (lambda (o) (match o '(c d) (list (_us_ria c) d) o))))
																	(define _psa (map (coalesceNil (stage_partition_aliases s) '()) (lambda (a) (coalesceNil (_us_lookup a) a))))
																	(stage_preserve_cache_meta s
																		(make_group_stage _psg _psh _pso (stage_limit_val s) (stage_offset_val s) _psa (stage_init_code s))))))
																(coalesceNil (sq_cache "groups") '()))))
														/* direct table entry with join condition (like non-agg non-LIMIT path) */
														(define us_join_lim (map us_outer_parts (lambda (p) (_us_ria (_us_ror p)))))
														(define us_inner_lim (_us_ria us_inner_cond_raw))
														(define us_full_lim (if (nil? us_inner_lim)
															(if (equal? (count us_join_lim) 0) true (if (equal? (count us_join_lim) 1) (car us_join_lim) (cons (quote and) us_join_lim)))
															(cons (quote and) (merge us_join_lim (list us_inner_lim)))))
														(define us_tbl_entries (list (list us_sq_prefix us_tbl_schema us_tbl_name true us_full_lim)))
														/* register schema for own table + pass through inner-scoped schemas */
														(define _us_inner_schema (schemas2_us us_tblvar))
														(define _us_passthrough_schemas (merge
															(if (not (nil? _us_inner_schema)) (list us_sq_prefix _us_inner_schema) '())
															(merge (map _us_inner_tbls (lambda (td) (match td
																'(a _ _ _ _) (begin
																	(define _isch (schemas2_us a))
																	(if (nil? _isch) '() (list a _isch)))
																'()))))))
														(if (not (equal? _us_passthrough_schemas '()))
															(sq_cache "schemas" (merge _us_passthrough_schemas (coalesceNil (sq_cache "schemas") '()))))
														/* substitution: apply _us_ria to the value expression.
														If value comes from own table, _us_ria renames it to us_sq_prefix.
														If value comes from inner-scoped table, it stays unchanged. */
														(define us_subst (_us_ria us_value_expr))
														(list us_subst us_tbl_entries))
													nil /* multi-table or computed value: not yet handled */
											))
										)
									)
								)
					)))
					nil /* untangle failed */
				)
			)
		)
	)))

	(define inner_select_kind (lambda (sym) (begin
		(if (string? sym)
			(if (equal?? sym "inner_select")
				(quote inner_select)
				(if (equal?? sym "inner_select_in")
					(quote inner_select_in)
					(if (equal?? sym "inner_select_exists")
						(quote inner_select_exists)
						nil)))
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
				_ nil
			)
		)
	)))
	(define not_symbol (lambda (sym) (match sym
		(symbol not) true
		'not true
		'(quote not) true
		_ false
	)))
	(define _contains_inner_select_marker (lambda (expr) (match expr
		(cons sym args) (or
			(not (nil? (inner_select_kind sym)))
			(_contains_inner_select_marker sym)
			(reduce args (lambda (found arg) (or found (_contains_inner_select_marker arg))) false))
		false)))
	(define count_subquery_cache_policy (lambda (query)
		(match query
			'(s t f c g h o l off) (begin
				(define only_count (match f
					'("__cnt" ((quote aggregate) 1 op 0)) (equal?? op (quote +))
					'("__cnt" ((symbol aggregate) 1 op 0)) (equal?? op (quote +))
					false))
				(if (and only_count (equal? g '(1)) (expr_uses_session_state query))
					(quote uncached-count)
					nil))
			nil)))
	/* _unnest_count_subselect: shared helper for IN/EXISTS/NOT IN/NOT EXISTS rewrite.
	Rewrites semi-joins (EXISTS/IN) and anti-joins (NOT EXISTS/NOT IN) as COUNT(*)
	aggregates instead of direct semi/anti-join operators. This is intentional:
	the COUNT-based approach produces a keytable computed column that benefits from
	MemCP's incremental aggregate cache — DML triggers invalidate only affected
	groups, so subsequent queries skip recomputation for unchanged partitions.
	Caching policy roadmap for session-sensitive predicates:
	1. First iteration: if the COUNT/EXISTS condition depends on volatile session
	state (for example @current_user, @fop_time, Betrachtungszeit, username),
	build_queryplan must prefer a cache-free execution path where GROUP ==
	current subselect domain and the predicate is evaluated on the current row
	stream instead of a reusable keytable cache.
	2. Second iteration: enable memoizing caches for predicates that depend on a
	stable session key (for example fixed user-id). The cache key must then
	include both the logical domain D and the memoized session value, i.e.
	semantically the cache lives on D x SessionKey rather than on D alone.
	If those entries are managed independently, cache eviction may also need
	row-wise cleanup hooks so the cache manager can register a domain plus
	memory budget together with a callback/DELETE plan for affected rows only.
	3. Third iteration: add cache-aware iterative rescans for monotone/session
	predicates by reasoning in SQL predicate algebra instead of application
	semantics. Example: x < @y is disjoint from x >= @y, and if @y < @z then
	the cached result for x < @y can be reused while only the delta predicate
	x >= @y AND x < @z has to be scanned; their union is equivalent to x < @z.
	build_queryplan can therefore decompose a broader query into reusable cache
	fragments plus a catch-up scan over the previously uncovered range.
	Until those stages are implemented, volatile session-dependent predicates must
	not be treated as freely reusable aggregate caches.
	Builds a COUNT(*) subquery from the original, optionally adding an equality condition
	(for IN/NOT IN: first_field = target_expr). Returns (substitution tables) or nil.
	comparison: (quote >) for positive match, (quote equal?) for negated match */
	(define _subquery_outer_refs (lambda (query outer_schemas) (begin
		(match (apply untangle_query (merge query (list outer_schemas)))
			'(_ tables2 fields2 condition2 _groups2 schemas2 _rfcol2 _init2) (begin
				(define _inner_aliases (map tables2 (lambda (td) (match td '(a _ _ _ _) a ""))))
				(define _eor (lambda (expr) (match expr
					(cons sym args) (if (or (equal? sym (quote outer)) (equal? sym '(quote outer)))
						(match args (cons sym_arg '()) (list (string sym_arg)) '())
						(if (or (equal? sym (quote get_column)) (equal? sym '(quote get_column)) (equal? sym '(symbol get_column)))
							(match args '(alias_ _ col _)
								(if (and (not (nil? alias_))
									(not (reduce _inner_aliases (lambda (a ia) (or a (equal?? ia alias_))) false)))
									(list (concat alias_ "." col))
									'())
								'())
							(if (_is_opaque_scope_sym sym) '()
								(merge_unique (map args _eor)))))
					'())))
				(merge_unique
					(merge (extract_assoc fields2 (lambda (_k v) (_eor v))))
					(_eor (coalesceNil condition2 true))))
			'()))))
	(define _subquery_has_outer_refs (lambda (query outer_schemas)
		(not (equal? (_subquery_outer_refs query outer_schemas) '()))))
	(define _outer_ref_is_direct_column (lambda (outer_schemas ref) (match (split ref ".")
		(list alias col) (begin
			(define cols (if (has_assoc? outer_schemas alias) (outer_schemas alias) nil))
			(define coldef (if (nil? cols)
				nil
				(reduce cols (lambda (found cd)
					(if (or (not (nil? found)) (not (equal?? (cd "Field") col)))
						found
						cd))
					nil)))
			(if (nil? coldef)
				false
				(begin
					(define expr (coalesceNil (coldef "Expr") nil))
					(match expr
						nil true
						'((quote get_column) _ _ _ _) true
						'((symbol get_column) _ _ _ _) true
						_ false))))
		_ false)))
	(define _subquery_outer_refs_are_direct_columns (lambda (query outer_schemas)
		(reduce (_subquery_outer_refs query outer_schemas) (lambda (all_ok ref)
			(and all_ok (_outer_ref_is_direct_column outer_schemas ref)))
			true)))
	(define unnest_scalar_subselect (lambda (subquery outer_schemas) (match subquery
		'(_ _ flds _ g h o l off) (begin
			(define value_expr (match flds
				(cons _ (cons v _)) v
				nil))
			(define has_outer_refs (_subquery_has_outer_refs subquery outer_schemas))
			(define has_aggregates (not (equal? (extract_aggregates (coalesceNil value_expr true)) '())))
			/* Aggregate subselects always produce 1 row — LIMIT 1 is redundant */
			(define effective_limit (if (and has_aggregates (or (nil? l) (equal? l 1))) nil l))
			/* Path A: correlated non-aggregate — full Neumann decorrelation to LEFT JOIN */
			(if (and has_outer_refs
				(_subquery_outer_refs_are_direct_columns subquery outer_schemas)
				(not (_contains_inner_select_marker subquery))
				(not (nil? value_expr))
				(not has_aggregates)
				(nil? h)
				(or (nil? g) (equal? g '()))
				(or (nil? o) (equal? o '()))
				(or (nil? effective_limit) (and (equal? effective_limit 1) (or (nil? o) (equal? o '()))))
				(or (nil? off) (equal? off 0)))
				(match (unnest_subselect subquery outer_schemas)
					'(subst tbls) (begin
						(sq_cache "scalar_tables" (merge tbls (coalesceNil (sq_cache "scalar_tables") '())))
						subst)
					nil)
				/* Path B: aggregate or uncorrelated — inject tables + scoped group stage
				into the outer query. build_queryplan turns this into keytable + createcolumn. */
				(begin
					(match (apply untangle_query (merge subquery (list outer_schemas)))
						'(schema2 tables2 fields2 condition2 groups2 schemas2 rfcol2 init2)
						(begin
							(if (and (not (nil? init2)) (not (equal? init2 '())))
								(sq_cache "init" (merge (coalesceNil (sq_cache "init") '()) init2)))
							(define groups2 (coalesceNil groups2 '()))
							/* strip redundant LIMIT from aggregate stages */
							(define groups2 (if has_aggregates
								(map groups2 (lambda (s)
									(make_group_stage (stage_group_cols s) (stage_having_expr s)
										(stage_order_list s) nil nil
										(stage_partition_aliases s) (stage_init_code s))))
								groups2))
							/* For aggregates without explicit GROUP BY, determine group keys:
							- Uncorrelated: GROUP BY (1) — single global group
							- Correlated: GROUP BY <inner_correlation_columns> — one group per
							  outer correlation value, enabling keytable JOIN on group keys.
							  This is Neumann's Γ_{D;f} where D = correlation domain. */
							(define inner_aliases (map tables2 (lambda (td) (match td '(a _ _ _ _) a ""))))
							(define is_inner_ref (lambda (expr)
								(reduce (extract_tblvars expr) (lambda (acc tv) (and acc (has? inner_aliases tv))) true)))
							(define is_outer_ref (lambda (expr)
								(and (not (equal? (extract_tblvars expr) '()))
									(reduce (extract_tblvars expr) (lambda (acc tv) (and acc (not (has? inner_aliases tv)))) true))))
							/* extract correlation equalities: (equal?? inner_expr outer_expr) */
							(define flatten_and (lambda (expr) (match expr
								(cons sym parts) (if (or (equal? sym (quote and)) (equal? sym '(quote and)))
									(merge (map parts flatten_and))
									(list expr))
								(list expr))))
							(define cond_parts (if (or (nil? condition2) (equal? condition2 true)) '() (flatten_and condition2)))
							(define correlation_keys (filter (map cond_parts (lambda (part) (match part
								'((symbol equal??) left right)
								(if (and (is_inner_ref left) (is_outer_ref right)) left
									(if (and (is_inner_ref right) (is_outer_ref left)) right nil))
								'((quote equal??) left right)
								(if (and (is_inner_ref left) (is_outer_ref right)) left
									(if (and (is_inner_ref right) (is_outer_ref left)) right nil))
								_ nil))) (lambda (x) (not (nil? x)))))
							(define group_keys (if (equal? correlation_keys '()) '(1) correlation_keys))
							/* generate unique alias */
							(define sq_idx (coalesceNil (sq_cache "idx") 0))
							(sq_cache "idx" (+ sq_idx 1))
							(define sq_alias (concat "sq" sq_idx))
							/* prefix inner table aliases */
							(define prefixed_tables (map tables2 (lambda (td) (match td
								'(alias s t io je) (list (concat sq_alias "\0" alias) s t io je)
								td))))
							(define prefixed_aliases (map prefixed_tables (lambda (td) (match td '(a _ _ _ _) a ""))))
							/* register prefixed tables */
							(sq_cache "tables" (merge prefixed_tables (coalesceNil (sq_cache "tables") '())))
							/* register prefixed schemas */
							(sq_cache "schemas" (merge
								(extract_assoc schemas2 (lambda (k v) (list (concat sq_alias "\0" k) v)))
								(coalesceNil (sq_cache "schemas") '())))
							/* prefix value expression column refs */
							(define value_expr2 (car (extract_assoc fields2 (lambda (k v) v))))
							(define prefix_expr (lambda (expr) (match expr
								'((symbol get_column) alias ti col ci) (if (or (nil? alias) (not (has? inner_aliases alias))) expr
									(list (quote get_column) (concat sq_alias "\0" alias) ti col ci))
								'((quote get_column) alias ti col ci) (if (or (nil? alias) (not (has? inner_aliases alias))) expr
									(list (quote get_column) (concat sq_alias "\0" alias) ti col ci))
								(cons sym args) (cons (prefix_expr sym) (map args prefix_expr))
								expr)))
							/* inject GROUP BY with prefixed keys */
							(define prefixed_group_keys (map group_keys prefix_expr))
							(define groups2 (if (and has_aggregates (or (nil? groups2) (equal? groups2 '())))
								(list (make_group_stage prefixed_group_keys nil nil nil nil nil nil))
								groups2))
							/* Split condition into correlation equalities (→ outer condition for
							JOIN) and local filters (→ stage-condition for keytable scan).
							Correlation equalities are handled by GROUP BY keys + JOIN; they
							must NOT stay in stage-condition or they'd reference outer tables
							inside the createcolumn scan where outer refs aren't available. */
							(define is_correlation_part (lambda (part) (match part
								'((symbol equal??) left right) (or (and (is_inner_ref left) (is_outer_ref right))
									(and (is_inner_ref right) (is_outer_ref left)))
								'((quote equal??) left right) (or (and (is_inner_ref left) (is_outer_ref right))
									(and (is_inner_ref right) (is_outer_ref left)))
								_ false)))
							(define local_cond_parts (filter cond_parts (lambda (p) (not (is_correlation_part p)))))
							(define corr_cond_parts (filter cond_parts is_correlation_part))
							(define prefixed_local_condition (if (equal? local_cond_parts '()) nil
								(prefix_expr (if (equal? 1 (count local_cond_parts)) (car local_cond_parts)
									(cons (quote and) local_cond_parts)))))
							/* correlation equalities go into outer condition for GROUP BY key JOIN */
							(if (not (equal? corr_cond_parts '()))
								(sq_cache "condition" (merge
									(map corr_cond_parts prefix_expr)
									(coalesceNil (sq_cache "condition") '()))))
							/* register scoped group stages with partition-aliases + stage-condition */
							(define scoped_stages (map groups2 (lambda (stage)
								(make_group_stage_with_condition (stage_group_cols stage) (stage_having_expr stage)
									(stage_order_list stage) (stage_limit_val stage) (stage_offset_val stage)
									prefixed_aliases (stage_init_code stage) prefixed_local_condition))))
							(sq_cache "groups" (merge scoped_stages (coalesceNil (sq_cache "groups") '())))
							/* Bind aggregates to their scoped table via a 4th data element.
							(aggregate 1 + 0) → (aggregate 1 + 0 "sq0\0pb_doc")
							The string scope-tag is pure data (not compiled by the optimizer).
							build_queryplan uses it to assign aggregates to the correct stage. */
							(define bind_agg_scope (lambda (expr) (match expr
								(cons (symbol aggregate) '(ae ar an))
								(list (quote aggregate) ae ar an (car prefixed_aliases))
								(cons '(quote aggregate) '(ae ar an))
								(list (quote aggregate) ae ar an (car prefixed_aliases))
								(cons sym args) (cons (bind_agg_scope sym) (map args bind_agg_scope))
								expr)))
							(bind_agg_scope (prefix_expr value_expr2)))
						nil))))
		nil)))
	(define _unnest_count_subselect (lambda (subquery outer_schemas target_expr comparison) (begin
		(define _resolve_outer (lambda (expr) (match expr
			'((symbol get_column) nil ti col ci) (begin
				(define _resolved (reduce_assoc outer_schemas (lambda (a alias cols)
					(if (reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false) alias a)) nil))
				(if (nil? _resolved) expr
					(list (quote get_column) _resolved false col false)))
			(cons sym args) (cons (_resolve_outer sym) (map args _resolve_outer))
			expr)))
		(define resolved_target_expr (if (nil? target_expr) nil (_resolve_outer target_expr)))
		/* UNION ALL: recurse into each branch, combine with OR (positive) or AND (negated) */
		(define _union_parts (query_union_all_parts subquery))
		(if (not (nil? _union_parts))
			(match _union_parts '(branches order limit offset)
				(if (or (not (nil? order)) (not (nil? limit)) (not (nil? offset)))
					nil /* UNION ALL with ORDER/LIMIT/OFFSET: not supported */
					(begin
						(define _first_cols (match branches
							(cons first_branch _) (query_branch_field_names first_branch)
							_ '()))
						(if (not (reduce branches (lambda (acc branch)
							(and acc (equal? (count (query_branch_field_names branch)) (count _first_cols)))) true))
							(error "UNION ALL branches must project the same number of columns")
							nil)
						(define _branch_exists_expr (lambda (branch) (match branch
							'(s t f c g h o l off) (begin
								(define _first_field (if (nil? resolved_target_expr) nil
									(match f (cons _ (cons v _)) v nil)))
								(if (and (not (nil? resolved_target_expr)) (nil? _first_field))
									nil
									(begin
										(define _branch_condition (if (nil? resolved_target_expr) c
											(if (or (nil? c) (equal? c true))
												(list (quote equal??) _first_field resolved_target_expr)
												(list (quote and) c (list (quote equal??) _first_field resolved_target_expr)))))
										(define _exists_expr (build_exists_subselect
											(list s t f _branch_condition g h o l off)
											outer_schemas))
										(if (equal?? comparison (quote >))
											_exists_expr
											(list (quote not) _exists_expr)))))
							nil)))
						(define _branch_results (filter (map branches _branch_exists_expr)
							(lambda (r) (not (nil? r)))))
						(if (or (equal? _branch_results '()) (not (equal? (count _branch_results) (count branches))))
							nil
							(if (equal? 1 (count _branch_results)) (car _branch_results)
								(cons (if (equal?? comparison (quote >)) (quote or) (quote and)) _branch_results))))))
			/* single subquery (non-UNION) path */
			(begin
				(define count-map-expr-for (lambda (cond-expr)
					(if (or (nil? cond-expr) (equal? cond-expr true))
						1
						(list (quote if) cond-expr 1 0))))
				(define _first_field (if (nil? target_expr) nil
					(match subquery '(_ _ flds _ _ _ _ _ _) (match flds (cons _ (cons v _)) v nil) nil)))
				(define target_expr resolved_target_expr)
				(if (and (nil? target_expr) (not (_subquery_has_outer_refs subquery outer_schemas)))
					(begin
						(define _count_sq (match subquery
							'(s t f c g h o l off) (list s t
								(list "__cnt" (list (quote aggregate) 1 (symbol "+") 0))
								c
								nil nil nil nil nil)
							nil))
						(if (nil? _count_sq)
							nil
							(begin
								(define _count_idx (coalesceNil (sq_cache "idx") 0))
								(sq_cache "idx" (+ _count_idx 1))
								(define _count_alias (concat "_uncorr_cnt_" _count_idx))
								(define mat_source (materialized-subquery-source _count_alias _count_sq))
								(define _count_rows_sym (symbol (concat "__uncorr_count_rows:" _count_idx)))
								(define _count_rr_sym (symbol (concat "__uncorr_count_rr:" _count_idx)))
								(define materialized_rows (list (quote begin)
									(list (quote set) _count_rows_sym (list (quote newsession)))
									(list _count_rows_sym "rows" '())
									(list (quote set) _count_rr_sym (symbol "resultrow"))
									(list (quote set) (symbol "resultrow")
										(list (quote lambda) (list (symbol "item"))
											(list _count_rows_sym "rows"
												(list (quote merge) (list _count_rows_sym "rows") (list (quote list) (symbol "item")))))
									)
									(build_queryplan_term _count_sq)
									(list (quote set) (symbol "resultrow") _count_rr_sym)
									(list _count_rows_sym "rows")))
								/* D = ∅: materialize the helper once and expose it as a normal
								one-row relation with visible column __cnt. The outer query still
								sees a regular table input, not a nested runtime subquery. */
								(sq_cache "init" (merge (coalesceNil (sq_cache "init") '())
									(list (materialized-subquery-init _count_alias _count_sq materialized_rows))))
								(sq_cache "tables" (merge
									(list (list _count_alias schema mat_source false nil))
									(coalesceNil (sq_cache "tables") '())))
								(sq_cache "schemas" (merge
									(list _count_alias (list (list "Field" "__cnt" "Type" "any")))
									(coalesceNil (sq_cache "schemas") '())))
								(list comparison
									(list (quote coalesceNil)
										(list (quote get_column) _count_alias false "__cnt" false)
										0)
									0))))
					(if (and (not (nil? target_expr)) (nil? _first_field))
						nil
						(begin
							(define _count_sq (match subquery
								'(s t f c g h o l off) (list s t
									(list "__cnt" (list (quote aggregate) (count-map-expr-for c) (symbol "+") 0))
									(if (nil? target_expr) c
										(if (or (nil? c) (equal? c true))
											(list (quote equal??) _first_field target_expr)
											(list (quote and) c (list (quote equal??) _first_field target_expr))))
									(list 1) nil nil nil nil)
								nil))
							(if (nil? _count_sq)
								nil
								(begin
									(define _result (unnest_subselect _count_sq outer_schemas))
									(if (nil? _result)
										nil
										(match _result '(_subst _tbls) (begin
											(sq_cache "tables" (merge _tbls (coalesceNil (sq_cache "tables") '())))
											(list comparison (list (quote coalesceNil) _subst 0) 0)))))))))))
	)))
	/* replace_inner_selects: walks an expression tree and replaces inner_select markers
	with their Neumann-decorrelated equivalents. Scalar subselects go through
	unnest_subselect directly; IN/EXISTS/NOT IN/NOT EXISTS are first rewritten to
	COUNT(*) aggregates via _unnest_count_subselect, then decorrelated via Path A.
	Returns the rewritten expression with subselects replaced by get_column refs
	or comparison expressions on the unnested aggregate columns. */
	(define replace_inner_selects (lambda (expr outer_schemas) (match expr
		(cons sym args) (begin
			(define kind (inner_select_kind sym))
			(define union_exists_expr (lambda (subquery negated) (begin
				(define union_parts (query_union_all_parts subquery))
				(if (nil? union_parts)
					nil
					(match union_parts '(branches union_order union_limit union_offset)
						(if (or (not (nil? union_order)) (not (nil? union_limit)) (not (nil? union_offset)))
							nil
							(begin
								(define branch_exprs (map branches (lambda (branch)
									(replace_inner_selects
										(if negated
											(list (quote not) (list (quote inner_select_exists) branch))
											(list (quote inner_select_exists) branch))
										outer_schemas))))
								(if (equal? branch_exprs '())
									nil
									(if (equal? (count branch_exprs) 1)
										(car branch_exprs)
										(cons (if negated (quote and) (quote or)) branch_exprs))))))))))
			(define union_in_expr (lambda (target_expr subquery negated) (begin
				(define union_parts (query_union_all_parts subquery))
				(if (nil? union_parts)
					nil
					(match union_parts '(branches union_order union_limit union_offset)
						(if (or (not (nil? union_order)) (not (nil? union_limit)) (not (nil? union_offset)))
							nil
							(begin
								(if (not (reduce branches (lambda (ok branch)
									(and ok (equal? 1 (count (query_branch_field_names branch)))))
									true))
									(error "UNION ALL subquery must project exactly one column for IN")
									nil)
								(define normalize_union_in_branch (lambda (branch)
									(match branch
										'(b_schema b_tables b_fields b_condition b_group b_having b_order b_limit b_offset)
										(begin
											(define first_field_expr (match b_fields
												(cons _ (cons v _)) v
												nil))
											(if (or (nil? first_field_expr) (not (or (nil? b_condition) (equal? b_condition true))))
												branch
												(list b_schema b_tables b_fields
													(list (quote equal??) first_field_expr first_field_expr)
													b_group b_having b_order b_limit b_offset)))
										branch)))
								(define rewritten_expr
									(if (equal? (count branches) 1)
										(if negated
											(list (quote not) (list (quote inner_select_in) target_expr (normalize_union_in_branch (car branches))))
											(list (quote inner_select_in) target_expr (normalize_union_in_branch (car branches))))
										(cons (if negated (quote and) (quote or))
											(map branches (lambda (branch)
												(if negated
													(list (quote not) (list (quote inner_select_in) target_expr (normalize_union_in_branch branch)))
													(list (quote inner_select_in) target_expr (normalize_union_in_branch branch))))))))
								(replace_inner_selects rewritten_expr outer_schemas))))))))
			(define not_expr (if (not_symbol sym)
				(match args
					(cons inner_expr '()) (match inner_expr
						(cons inner_sym inner_args) (begin
							(define inner_kind (inner_select_kind inner_sym))
							(if (equal?? inner_kind (quote inner_select_in))
								(match inner_args
									(cons target_expr (cons subquery '()))
									(coalesce
										(union_in_expr target_expr subquery true)
										(_unnest_count_subselect subquery outer_schemas target_expr (quote equal?))
										expr)
									_ nil)
								(if (equal?? inner_kind (quote inner_select_exists))
									(match inner_args
										(cons subquery '())
										(coalesce
											(union_exists_expr subquery true)
											(if (expr_uses_session_state subquery)
												(list (quote not) (build_exists_subselect subquery outer_schemas))
												(coalesce
													(_unnest_count_subselect subquery outer_schemas nil (quote equal?))
													(list (quote not) (build_exists_subselect subquery outer_schemas)))))
										_ nil)
									nil)))
						_ nil)
					_ nil)
				nil))
			(if (nil? not_expr)
				(match kind
					(quote inner_select) (match args
						(cons subquery '()) (coalesce
							(unnest_scalar_subselect subquery outer_schemas)
							(error (concat "unnest_scalar_subselect returned nil for subquery — build_scalar_subselect is removed")))
						_ (cons sym (map args (lambda (arg) (replace_inner_selects arg outer_schemas)))))
					(quote inner_select_in) (match args
						(cons target_expr (cons subquery '()))
						(coalesce
							(union_in_expr target_expr subquery false)
							(_unnest_count_subselect subquery outer_schemas target_expr (quote >))
							expr)
						_ (cons sym (map args (lambda (arg) (replace_inner_selects arg outer_schemas)))))
					(quote inner_select_exists) (match args
						(cons subquery '())
						(coalesce
							(union_exists_expr subquery false)
							(if (expr_uses_session_state subquery)
								(build_exists_subselect subquery outer_schemas)
								(coalesce
									(_unnest_count_subselect subquery outer_schemas nil (quote >))
									(build_exists_subselect subquery outer_schemas))))
						_ (cons sym (map args (lambda (arg) (replace_inner_selects arg outer_schemas)))))
					_ (cons sym (map args (lambda (arg) (replace_inner_selects arg outer_schemas)))))
				not_expr))
		expr
	)))

	/* no-FROM rewrite: inject virtual one-row table ".(1)" (like Oracle DUAL).
	Dot prefix hides from SHOW TABLES. Eliminates the no-table special case.
	set tables= must wrap the if (set is scope-local in this Scheme dialect). */
	(set tables (if (or (nil? tables) (equal? tables '()))
		(begin
			(createdatabase schema true)
			/* ".(1)" is the synthetic one-row DUAL table for no-FROM queries.
			It may already exist while being empty after cache eviction / restart /
			other recovery paths. Re-check runtime emptiness instead of inserting
			only on first CREATE, otherwise scalar/EXISTS no-FROM subqueries become
			silently empty and collapse to NULL/FALSE. */
			(begin
				(createtable schema ".(1)"
					(list (list "unique" "group" (list "1")) (list "column" "1" "any" (list) (list)))
					(list "engine" "sloppy") true)
				(if (table_empty? schema ".(1)")
					(insert schema ".(1)" (list "1") (list (list 1)) (list) (lambda () true) true)
					nil))
			(list (list ".(1)" schema ".(1)" false nil)))
		tables))
	(set zipped (zip (map tables (lambda (tbldesc) (match tbldesc
		'(alias schema (string? tbl) _ _) '('(tbldesc) '() true '(alias (get_schema schema tbl))) /* leave primary tables as is and load their schema definition */
		'(id schemax subquery isOuter joinexpr) (begin
			(define union_parts_from (query_union_all_parts subquery))
			(if (not (nil? union_parts_from))
				(match union_parts_from '(branches union_order union_limit union_offset) (begin
					(define output_cols (match branches
						(cons first_branch _) (query_branch_field_names first_branch)
						_ '()))
					(if (or (nil? output_cols) (equal? output_cols '()))
						(error "UNION ALL subquery must project at least one column"))
					(define rows_sym (symbol (concat "__from_union_rows:" id)))
					(define resultrow_sym (symbol (concat "__from_union_resultrow:" id)))
					(define materialized_rows (list (quote begin)
						(list (quote set) rows_sym (list (quote newsession)))
						(list rows_sym "rows" '())
						(list (quote set) resultrow_sym (symbol "resultrow"))
						(list (quote set) (symbol "resultrow")
							(list (quote lambda) (list (symbol "item"))
								(list rows_sym "rows"
									(list (quote merge) (list rows_sym "rows") (list (quote list) (symbol "item")))))
						)
						(build_queryplan_term subquery)
						(list (quote set) (symbol "resultrow") resultrow_sym)
						(list rows_sym "rows")
					))
					(sq_cache "init" (merge (coalesceNil (sq_cache "init") '())
						(list (materialized-subquery-init id subquery materialized_rows))))
					(list
						(list (list id schemax (materialized-subquery-source id subquery) isOuter joinexpr))
						'()
						true
						(list id (map output_cols (lambda (col) (list "Field" col "Type" "any"))))
					)
				))
				(match (apply untangle_query subquery) '(schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column2 _init2) (begin
					(if (and (not (nil? _init2)) (not (equal? _init2 '())))
						(sq_cache "init" (merge (coalesceNil (sq_cache "init") '()) _init2)))
					/* helper function add prefix to tblalias of every expression */
					(define replace_column_alias (lambda (expr) (match expr
						'((symbol get_column) nil ti col ci) (begin
							/* resolve unqualified column against inner schemas2; must match exactly one table.
							Skip aliases that contain \0 (null byte) — those are prefixed from flattened derived tables
							and should not participate in unqualified column resolution. */
							(define matches (reduce_assoc schemas2 (lambda (acc alias cols)
								(if (and (equal? (replace alias "\0" "") alias)
									(reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false))
									(cons alias acc)
									acc)) '()))
							(define matches (reduce matches (lambda (acc alias) (append_unique acc alias)) '()))
							(match matches
								(cons only '()) '('get_column (concat id "\0" only) ti col ci)
								'() (begin
									/* column not in schemas2 - check if it's a SELECT alias in fields2 */
									(if (nil? (fields2 col))
										expr /* leave unresolved — inner subselect scope will handle it */
										/* found in fields2 - resolve to the underlying expression */
										(replace_column_alias (fields2 col))
									)
								)
								(cons _ _) (error (concat "ambiguous column " col " in subquery"))
							)
						)
						'((symbol get_column) alias_ ti col ci) (if (not (nil? (schemas2 alias_)))
							'('get_column (concat id "\0" alias_) ti col ci)
							expr) /* alias not in schemas2 → inner subselect scope, leave as-is */
						'((symbol outer) outer_arg) (begin
							/* prefix outer variable reference if it refers to a table in schemas2 */
							(define s (string outer_arg))
							(define parts (split s "."))
							(match parts
								(list tbl col) (if (not (nil? (schemas2 tbl)))
									(list (quote outer) (symbol (concat id "\0" tbl "." col)))
									(list (quote outer) outer_arg))
								_ (list (quote outer) (replace_column_alias outer_arg))
							)
						)
						(cons sym args) /* function call */ (if (not (nil? (inner_select_kind sym)))
							expr /* inner subselects resolved later by replace_inner_selects */
							(if (is_quote_scope_sym sym)
								expr
								(if (_is_opaque_scope_sym sym)
									(rewrite_opaque_outer_alias_for_flatten id schemas2 expr)
									(cons sym (map args replace_column_alias)))))
						expr
					)))
					/* prefix all table aliases and transform their joinexprs */
					(set tablesPrefixed (map tables2 (lambda (x) (match x '(alias schema tbl a innerJoinexpr)
						(list (concat id "\0" alias) schema tbl a (if (nil? innerJoinexpr) nil (replace_column_alias innerJoinexpr)))))))
					/* helper function to transform joinexpr: only transform references to subquery alias id */
					(define transform_joinexpr (lambda (expr) (match expr
						'((symbol get_column) alias_ ti col ci) (if (equal?? alias_ id)
							/* reference to subquery alias -> resolve against inner schemas by passing nil alias */
							(replace_column_alias (list (quote get_column) nil ti col ci))
							/* reference to outer table -> keep as-is */
							expr)
						(cons sym args) /* function call */ (if (not (nil? (inner_select_kind sym))) expr /* inner subselects have their own scope */ (cons sym (map args transform_joinexpr)))
						expr
					)))
					/* transform and attach joinexpr to first table in tablesPrefixed */
					(set joinexpr2 (if (nil? joinexpr) nil (transform_joinexpr joinexpr)))
					/* for LEFT JOIN (isOuter=true), integrate condition2 into joinexpr to preserve LEFT JOIN semantics */
					(set condition2_transformed (replace_column_alias condition2))
					(set joinexpr2 (if isOuter
						/* merge condition2 into joinexpr for outer joins */
						(if (nil? joinexpr2)
							condition2_transformed
							(if (or (nil? condition2_transformed) (equal? condition2_transformed true))
								joinexpr2
								(list (quote and) joinexpr2 condition2_transformed)))
						joinexpr2))
					(if (and (not (nil? joinexpr2)) (not (nil? tablesPrefixed)))
						(set tablesPrefixed (cons
							/* inherit isOuter from the subquery's join type, not from inner table */
							(match (car tablesPrefixed) '(a s t io je) (list a s t isOuter joinexpr2))
							(cdr tablesPrefixed)))
					)
					(define flattened_table_aliases (map tablesPrefixed (lambda (td) (match td '(alias _ _ _ _) alias ""))))
					(define _has_dangling_flatten_ref (lambda (expr)
						(reduce (extract_all_get_columns expr) (lambda (acc mc)
							(or acc (match mc
								'(name '((symbol get_column) alias_ _ _ _))
								(begin
									(define alias_str (string alias_))
									(and (strlike alias_str (concat id "\0%"))
										(not (has? flattened_table_aliases alias_str))))
								'(name '((quote get_column) alias_ _ _ _))
								(begin
									(define alias_str (string alias_))
									(and (strlike alias_str (concat id "\0%"))
										(not (has? flattened_table_aliases alias_str))))
								false)))
							false)))
					(define flatten_has_dangling_output_ref
						(reduce_assoc fields2 (lambda (acc _k v)
							(or acc (_has_dangling_flatten_ref (replace_column_alias v))))
							false))
					/* window functions in subquery require materialization (cannot flatten because window needs its own ordering) */
					(define subquery_has_window (not (equal? (merge (extract_assoc fields2 (lambda (k v) (extract_window_funcs v)))) '())))
					/* TODO: group+order+limit+offset -> ordered scan list with aggregation layers (to avoid materialization) */
					/* Note: flat defines avoid nested begin scopes — (set) only updates the innermost Nodefine=false env */
					(define groups2_present (and (not (nil? groups2)) (not (equal? groups2 '()))))
					(define unsupported_groups (if groups2_present
						(reduce groups2 (lambda (acc stage)
							(or acc
								(begin
									(define g (stage_group_cols stage))
									(and (not (nil? g)) (not (equal? g '())))
								)
								(not (nil? (stage_having_expr stage)))
								(not (nil? (stage_limit_val stage)))
								(not (nil? (stage_offset_val stage)))
							)
						) false)
						false))
					(define use_materialize (or subquery_has_window unsupported_groups flatten_has_dangling_output_ref))
					/* Window-function LIMIT pushdown */
					(define mat_limit nil)
					(if subquery_has_window (begin
						(define _check_wf_limit (lambda (cond) (match cond
							'('<= '('get_column _ _ col _) n) (if (and (not (nil? (get_assoc fields2 col))) (not (equal? (extract_window_funcs (get_assoc fields2 col)) '())))
								(set mat_limit n) nil)
							'('< '('get_column _ _ col _) n) (if (and (not (nil? (get_assoc fields2 col))) (not (equal? (extract_window_funcs (get_assoc fields2 col)) '())))
								(set mat_limit (- n 1)) nil)
							'('and a b) (begin (_check_wf_limit a) (_check_wf_limit b))
							nil)))
						(_check_wf_limit condition)
					))
					/* if groups2 had only pass-through stages (no GROUP/HAVING/LIMIT/OFFSET), strip them for flattening */
					(if (and groups2_present (not unsupported_groups))
						(set groups2 nil))
					(if use_materialize
						(begin
							(define output_cols_sub (extract_assoc fields2 (lambda (k v) k)))
							(define rows_sym (symbol (concat "__from_subquery_rows:" id)))
							(define resultrow_sym (symbol (concat "__from_subquery_resultrow:" id)))
							/* Build the materialized inner plan from the already untangled IR of
							this subquery. Replanning from the raw AST here can drift from the
							current alias/scope environment and reintroduce wrapper-specific
							regressions. */
							(define mat_inner_plan (build_queryplan schema2 tables2 fields2 condition2 groups2 schemas2 replace_find_column2 nil))
							(define mat_init_stmts (if (or (nil? _init2) (equal? _init2 '())) '() _init2))
							(define mat_inner_plan (if (equal? mat_init_stmts '())
								mat_inner_plan
								(cons (quote !begin) (merge mat_init_stmts (list mat_inner_plan)))))
							(define materialized_rows (list (quote begin)
								(list (quote set) rows_sym (list (quote newsession)))
								(list rows_sym "rows" '())
								(list (quote set) resultrow_sym (symbol "resultrow"))
								(define cnt_sym (symbol (concat "__from_subquery_cnt:" id)))
								(if (nil? mat_limit)
									/* no limit */
									(list (quote set) (symbol "resultrow")
										(list (quote lambda) (list (symbol "item"))
											(list rows_sym "rows"
												(list (quote merge) (list rows_sym "rows") (list (quote list) (symbol "item")))))
									)
									/* with limit: stop collecting after mat_limit rows */
									(list (quote begin)
										(list (quote set) cnt_sym 0)
										(list (quote set) (symbol "resultrow")
											(list (quote lambda) (list (symbol "item"))
												(list (quote if) (list (quote <) cnt_sym mat_limit)
													(list (quote begin)
														(list (quote set) cnt_sym (list (quote +) cnt_sym 1))
														(list rows_sym "rows"
															(list (quote merge) (list rows_sym "rows") (list (quote list) (symbol "item")))))
													nil))))
								)
								mat_inner_plan
								(list (quote set) (symbol "resultrow") resultrow_sym)
								(list rows_sym "rows")
							))
							(sq_cache "init" (merge (coalesceNil (sq_cache "init") '())
								(list (materialized-subquery-init id subquery materialized_rows))))
							(list
								(list (list id schemax (materialized-subquery-source id subquery) isOuter joinexpr))
								'()
								true
								(list id (merge_schema_fields_unique (list
									(map output_cols_sub (lambda (col) (list "Field" col "Type" "any"))))))
							)
						)
						(begin
							/* for LEFT JOIN: condition2 was integrated into joinexpr, so return true as global filter */
							/* for INNER JOIN: condition2 becomes global filter (can be reordered) */
							(set globalFilter (if isOuter true (replace_column_alias condition2)))
							(define _check_inner_select (lambda (expr) (match expr (cons sym args) (if (not (nil? (inner_select_kind sym))) true (reduce args (lambda (a b) (or a (_check_inner_select b))) false)) false)))
							(define wrap_outer_join_projection (lambda (expr)
								(if (and isOuter (not (equal? joinexpr true)) (not (nil? joinexpr2)) (not (equal? joinexpr2 true)) (not (_check_inner_select joinexpr2)))
									(list (quote if) joinexpr2 expr nil)
									expr)))
							/* Column pruning: only carry forward subquery columns that the
							outer query actually references. */
							(define _referenced_cols (merge_unique (list
								(extract_columns_for_tblvar id fields)
								(extract_columns_for_tblvar id condition)
								(extract_columns_for_tblvar id (coalesceNil having true))
								(merge (map (coalesceNil order '()) (lambda (o) (extract_columns_for_tblvar id o)))))))
							(define schema_field_for_flatten (lambda (k v) (begin
								(define lowered (replace_column_alias v))
								(if (expr_has_opaque_scope lowered)
									(list "Field" k "Type" "any")
									(list "Field" k "Type" "any" "Expr" lowered)))))
							(define pruned_fields2 (if (equal? _referenced_cols '()) fields2
								(filter_assoc fields2 (lambda (k v) (has? _referenced_cols k)))))
							(list tablesPrefixed (list id (map_assoc fields2 (lambda (k v) (wrap_outer_join_projection (replace_column_alias v))))) globalFilter (merge (list id (extract_assoc fields2 schema_field_for_flatten)) (merge (extract_assoc schemas2 (lambda (k v) (list (concat id "\0" k) v))))))
						)
					)
				) (error "non matching return value for untangle_query"))
			)
		)
		(error (concat "unknown tabledesc: " tbldesc))
	)))))
	(set tablesList (car zipped))
	(set renameList (car (cdr zipped)))
	(set conditionList (car (cdr (cdr zipped))))
	(set schemasList (car (cdr (cdr (cdr zipped)))))
	/* schemas is an assoc array from alias -> columnlist */
	/* rewrite a flat table list according to inner selects */
	(set renamelist (merge renameList))
	(set tables (merge tablesList))
	(set schemas (merge schemasList))

	/* global WHERE stays separate from per-table joinexpr (ON). */
	(set condition (coalesceNil condition true))

	/* tells whether there is an aggregate inside */
	(define expr_find_aggregate (lambda (expr) (match expr
		'((symbol aggregate) item reduce neutral) true
		(cons sym args) /* function call */ (if (nil? (inner_select_kind sym))
			(reduce args (lambda (a b) (or a (expr_find_aggregate b))) false)
			false)
		false
	)))

	/* set group to 1 if fields contain aggregates even if not */
	(define group (coalesce group (if (reduce_assoc fields (lambda (a key v) (or a (expr_find_aggregate v))) false) '(1) nil)))

	/* find those columns that have no table */
	(define schema_alias_variants (lambda (alias)
		(reduce (filter (list
			alias
			(visible_occurrence_alias alias)
			(if (string? alias) (sanitize_temp_name alias) nil)
			(if (string? (visible_occurrence_alias alias)) (sanitize_temp_name (visible_occurrence_alias alias)) nil))
			(lambda (x) (not (nil? x))))
			(lambda (acc alias_v) (append_unique acc alias_v))
			'())))
	(define schema_alias_matches (lambda (query_alias schema_alias ti)
		(reduce (schema_alias_variants schema_alias) (lambda (matched alias_v)
			(or matched ((if ti equal?? equal?) query_alias alias_v)))
			false)))
	(define replace_find_column (lambda (expr) (match expr
		/* Ensure MySQL LIKE uses a collation at compile time:
		- If lhs is a text column, take collation from schema metadata.
		- Otherwise default to utf8mb4_general_ci (MySQL default in this project). */
		'((symbol strlike) a b c) (begin
			(define default_collation "utf8mb4_general_ci")
			(define find_column_collation (lambda (tblalias colname) (begin
				(define tblalias_str (if (string? tblalias) tblalias (string tblalias)))
				(define alias_lookup
					(coalesce
						(if (has_assoc? schemas tblalias_str) tblalias_str nil)
						nil))
				(define cols (if (nil? alias_lookup) nil (schemas alias_lookup)))
				(define coldef (if (list? cols)
					(reduce cols (lambda (a coldef)
						(if (or a (equal?? (coldef "Field") colname)) a coldef)
					) nil)
					nil))
				(coalesce (and coldef (coldef "Collation")) default_collation)
			)))
			(match a
				'((symbol get_column) nil _ col ci)
				(cons (quote strlike)
					(cons
						(replace_find_column a)
						(cons (replace_find_column b) (cons default_collation '()))))
				'((symbol get_column) alias_ ti col ci)
				(begin
					(define resolved
						(coalesce
							(reduce_assoc schemas (lambda (a alias cols)
								(if (and (schema_alias_matches alias_ alias ti)
									(reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false))
									alias
									a)
							) nil)
							alias_))
					(cons (quote strlike)
						(cons
							(replace_find_column a)
							(cons
								(replace_find_column b)
								(cons
									(if (equal?? c default_collation) (find_column_collation resolved col) c)
									'())))))
				_
				(cons (quote strlike)
					(cons (replace_find_column a) (cons (replace_find_column b) (cons c '()))))
			)
		)
		/* Unqualified column: prefer main tables over unnested/subquery tables.
		Main tables have no ':' prefix and no '_unn_' prefix in their alias. */
		'((symbol get_column) nil _ "*" _) expr
		'((quote get_column) nil _ "*" _) expr
		'((symbol get_column) _ _ "*" _) expr
		'((quote get_column) _ _ "*" _) expr
		'((symbol get_column) nil _ col ci) (begin
			/* First try main tables (aliases without ':' or '_unn_' prefix) */
			(define _is_main_alias (lambda (alias) (begin
				(define s (string alias))
				(and (not (strlike s "%:%"))
					(not (strlike s "%\0%"))
					(not (and (>= (strlen s) 5) (equal? (substr s 0 5) "_unn_")))))))
			(define main_match (reduce_assoc schemas (lambda (a alias cols)
				(if (and (_is_main_alias alias) (reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false))
					alias a)) nil))
			/* If not found in main tables, try subquery tables (aliases with ':') */
			(define any_match (if (nil? main_match)
				(reduce_assoc schemas (lambda (a alias cols)
					(if (reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false)
						alias a)) nil)
				main_match))
			(begin
				(define resolved_alias (coalesce any_match (error (concat "column " col " does not exist in tables"))))
				(define canonical_col (if ci (coalesce (reduce (schemas resolved_alias) (lambda (a coldef) (if (not (nil? a)) a (if (equal?? (coldef "Field") col) (coldef "Field") nil))) nil) col) col))
				'((quote get_column) resolved_alias false canonical_col false))
		)
		'((symbol get_column) alias_ ti col ci) (begin
			(define resolved_alias (reduce_assoc schemas (lambda (a alias cols)
				(if (and (schema_alias_matches alias_ alias ti)
					(reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false))
					alias
					a))
				nil))
			(if (nil? resolved_alias)
				expr
				(begin
					(define canonical_col (if ci
						(coalesce (reduce (schemas resolved_alias) (lambda (a coldef) (if (not (nil? a)) a (if (equal?? (coldef "Field") col) (coldef "Field") nil))) nil) col)
						col))
					'((quote get_column) resolved_alias false canonical_col false))))
		/* omit strict failure for false/false refs: freshly created temp columns are
		allowed to pass through unresolved until their stage materializes them */
		'((quote get_column) alias_ ti col ci) (begin
			(define resolved_alias (reduce_assoc schemas (lambda (a alias cols)
				(if (and (schema_alias_matches alias_ alias ti)
					(reduce cols (lambda (a coldef) (or a ((if ci equal?? equal?) (coldef "Field") col))) false))
					alias
					a))
				nil))
			(if (nil? resolved_alias)
				expr
				(begin
					(define canonical_col (if ci
						(coalesce (reduce (schemas resolved_alias) (lambda (a coldef) (if (not (nil? a)) a (if (equal?? (coldef "Field") col) (coldef "Field") nil))) nil) col)
						col))
					'((quote get_column) resolved_alias false canonical_col false))))
		(cons sym args) /* function call */ (if (_is_opaque_scope_sym sym)
			expr
			(cons sym (map args replace_find_column)))
		expr
	)))

	/* pass full schema chain (current + ancestors) so nested subselects can resolve grandparent refs */
	(define _ris_schemas (merge schemas outer_schemas_chain))
	(set tables (map tables (lambda (td) (match td
		'(tv tschema ttbl toisOuter tje)
		(list tv tschema ttbl toisOuter
			(if (nil? tje) nil (replace_inner_selects tje _ris_schemas)))
		td))))
	(set fields (map_assoc fields (lambda (k v) (replace_inner_selects v _ris_schemas))))
	(set condition (replace_inner_selects condition _ris_schemas))
	(set group (map group (lambda (g) (replace_inner_selects g _ris_schemas))))
	(set having (begin
		(define _hv_resolved (replace_inner_selects having _ris_schemas))
		/* check if any inner_select nodes remain — HAVING with subqueries
		requires post-group processing which is not yet implemented */
		(define _hv_check (lambda (expr) (match expr
			(cons sym args) (if (not (nil? (inner_select_kind sym))) true
				(reduce args (lambda (a b) (or a (_hv_check b))) false))
			false)))
		(if (and (not (nil? _hv_resolved)) (_hv_check _hv_resolved))
			(error "HAVING with subqueries not yet supported")
			_hv_resolved)))
	(set order (map order (lambda (o) (match o '(col dir) (list (replace_inner_selects col _ris_schemas) dir)))))
	/* Freeze visible top-level field refs against the currently visible tables
	before unnested helper tables are merged into schemas. This prevents later
	helper/keytable columns from stealing unrelated outer output bindings. */
	(define freeze_visible_field_refs (lambda (expr) (match expr
		'((symbol get_column) nil _ _ _) (replace_find_column expr)
		'((quote get_column) nil _ _ _) (replace_find_column expr)
		'((symbol get_column) alias_ _ _ _) (if (has_assoc? schemas alias_) (replace_find_column expr) expr)
		'((quote get_column) alias_ _ _ _) (if (has_assoc? schemas alias_) (replace_find_column expr) expr)
		(cons sym args) (if (_is_opaque_scope_sym sym)
			expr
			(cons sym (map args freeze_visible_field_refs)))
		expr)))
	(set fields (map_assoc fields (lambda (k v) (freeze_visible_field_refs v))))
	/* integrate unnested scalar subselects from Neumann unnesting.
	Tables from non-aggregate path (direct LEFT JOIN) do NOT need schema updates.
	Tables from aggregate path (materialized derived) DO need schemas for build_queryplan. */
	(define _sq_tbls (coalesceNil (sq_cache "tables") '()))
	(define _sq_scalar_tbls (coalesceNil (sq_cache "scalar_tables") '()))
	/* Contract: scalar helper tables used only for SELECT/expr projection keep
	their LEFT JOIN joinexpr local so NULL-preserving semantics survive.
	When the current WHERE references such a helper, it is no longer a pure
	projection helper: its joinexpr belongs to the row-domain and must be merged
	into the normal table/condition pipeline before grouping. */
	(define condition_ref_aliases (extract_tblvars condition))
	(define sq_scalar_condition_tbls (filter _sq_scalar_tbls (lambda (t) (match t
		'(tv _ ttbl _ _)
		(has? condition_ref_aliases (if (nil? tv) ttbl tv))
		false))))
	(define sq_scalar_projection_tbls (filter _sq_scalar_tbls (lambda (t)
		(not (has? sq_scalar_condition_tbls t)))))
	(set tables (merge tables _sq_tbls sq_scalar_condition_tbls sq_scalar_projection_tbls))
	(define _sq_schs (coalesceNil (sq_cache "schemas") '()))
	(if (not (equal? _sq_schs '())) (set schemas (merge schemas _sq_schs)))
	/* ensure materialized temp sources have a visible schema under their current alias.
	This keeps later planner passes from guessing temp columns via ad-hoc name heuristics. */
	(set schemas (reduce tables (lambda (acc td) (match td
		'(tv tschema ttbl _ _)
		(begin
			(define _existing (if (has_assoc? acc tv) (acc tv) nil))
			(define _resolved (coalesce _existing (materialized_source_schema tschema ttbl tv acc)))
			(if (nil? _resolved) acc
				(merge acc (list tv _resolved))))
		acc)) schemas))
	/* Design contract: logical get_column/aggregate/window sentinels should stay
	as long as possible and join semantics must stay attached to their stage.
	COUNT/IN/EXISTS helper tables still expose their correlation predicates here
	as global condition terms. Scalar projection helpers only stay local while
	they are projection-only; once WHERE references them, they join the normal
	row-domain and their joinexpr must participate in global filtering. */
	(define _sq_jes (filter (map (merge _sq_tbls sq_scalar_condition_tbls) (lambda (t) (match t '(_ _ _ _ je) je nil))) (lambda (x) (not (nil? x)))))
	(set condition (if (equal? _sq_jes '()) condition (cons (quote and) (cons condition _sq_jes))))
	/* Local filters from Path B are carried as stage-condition on scoped stages.
	Correlation equalities from correlated aggregate subselects go into the
	global condition for GROUP BY key JOIN resolution. */
	(define _sq_conds (coalesceNil (sq_cache "condition") '()))
	(if (not (equal? _sq_conds '()))
		(set condition (cons (quote and) (cons condition _sq_conds))))
	/* integrate partition stages from non-aggregate LIMIT unnesting */
	(define _sq_pstages (coalesceNil (sq_cache "partition_stages") '()))
	(define _sq_prop_groups (coalesceNil (sq_cache "groups") '()))
	(set groups (if (equal? _sq_pstages '()) groups (merge _sq_pstages (coalesceNil groups '()))))
	(set groups (if (equal? _sq_prop_groups '()) groups (merge _sq_prop_groups (coalesceNil groups '()))))
	/* canonicalize_for_rename: resolve case-insensitive column names to canonical form,
	but ONLY for columns referencing derived table aliases (keys in renamelist).
	Uses schemas to find canonical column name without calling replace_find_column. */
	(define canonicalize_for_rename (lambda (expr) (match expr
		'((symbol get_column) alias_ ti col ci) (if (and ci (not (nil? alias_)))
			(if (has_assoc? renamelist (string alias_))
				(begin
					(define alias_cols (schemas (string alias_)))
					(define canonical_col (if (nil? alias_cols) col
						(coalesce (reduce alias_cols (lambda (found coldef)
							(if (not (nil? found)) found
								(if (equal?? (coldef "Field") col) (coldef "Field") nil))) nil) col)))
					'((quote get_column) alias_ ti canonical_col ci))
				expr)
			expr)
		(cons sym args) (if (_is_opaque_scope_sym sym)
			expr
			(cons sym (map args canonicalize_for_rename)))
		expr
	)))

	/* apply renamelist (assoc of assoc of expr) */
	(define replace_rename (lambda (expr) (match expr
		'((symbol get_column) alias_ ti col ci) (if (nil? alias_)
			/* no tblalias -> search the field in all tables */
			(reduce_assoc renamelist (lambda (a k v) (coalesce (v col) a)) expr)
			/* tblalias -> look up the field */
			(begin
				(define alias_str (string alias_))
				(define alias_sym (symbol alias_str))
				(define rename_fn (if (has_assoc? renamelist alias_)
					(renamelist alias_)
					(if (has_assoc? renamelist alias_str)
						(renamelist alias_str)
						(if (has_assoc? renamelist alias_sym)
							(renamelist alias_sym)
							nil))))
				(if (nil? rename_fn) expr (rename_fn col))
			)
		)
		(cons sym args) /* function call */ (if (_is_opaque_scope_sym sym)
			expr
			(cons sym (map args replace_rename)))
		expr
	)))


	(define planner_visible_schemas (merge schemas outer_schemas_chain))
	(define finalize_visible_expr (lambda (expr)
		(finalize_logical_expr_scoped expr schemas planner_visible_schemas replace_rename enforce_planner_contract)))


	/* Contract boundary for user-visible expressions:
	fields, WHERE, GROUP/HAVING/ORDER and JOIN conditions all go through the same
	finalize_visible_expr gate exactly once here. After that the planner must only
	see exact get_column markers and may no longer re-run schema casing repair. */
	(set fields (map_assoc (expand_star_fields_with_schemas fields schemas) (lambda (col expr)
		(finalize_visible_expr (replace_find_column expr)))))

	/* return parameter list for build_queryplan */
	(set conditionAll (cons 'and (filter
		(cons (finalize_visible_expr condition) (map conditionList finalize_visible_expr))
		(lambda (x) (not (nil? x))))))
	(set tables (map tables (lambda (td) (match td
		'(tv tschema ttbl toisOuter tje)
		(list tv tschema ttbl toisOuter
			(if (nil? tje) nil (finalize_visible_expr tje)))
		td))))
	(set group (map group finalize_visible_expr))
	(set order (map order (lambda (o) (match o '(col dir) (list (finalize_visible_expr col) dir)))))

	(set having (finalize_visible_expr having))

	/* LEFT JOIN pruning: remove LEFT JOINed tables that are not referenced
	anywhere in the query (fields, condition, having, order, or sibling
	joinexprs). A LEFT JOIN that is never read contributes only NULL columns
	and cannot filter rows, so it is safe to drop entirely. */
	(define _all_referenced_aliases (merge_unique (list
		(extract_all_table_aliases fields)
		(extract_all_table_aliases conditionAll)
		(extract_all_table_aliases (coalesceNil having true))
		(merge (map (coalesceNil order '()) (lambda (o) (extract_all_table_aliases o))))
		(merge (map tables (lambda (td) (match td '(_ _ _ _ je) (if (nil? je) '() (extract_all_table_aliases je)) '())))))))
	(set tables (filter tables (lambda (td) (match td
		'(alias _ _ isOuter _) (or (not isOuter) (has? _all_referenced_aliases (string alias)))
		true))))

	(define groups (merge
		(coalesceNil _sq_pstages '())
		(coalesceNil _sq_prop_groups '())
		(if (coalesce _cd_distinct_exprs false)
			/* COUNT(DISTINCT): two group stages - first dedup, then aggregate */
			(list
				(make_dedup_stage
					(merge
						(map (coalesce _cd_user_group '()) finalize_visible_expr)
						(map _cd_distinct_exprs (lambda (e) (replace_find_column (finalize_visible_expr e)))))
					nil)
				(make_group_stage
					(if (nil? _cd_user_group) '(1) (map _cd_user_group (lambda (e) (replace_find_column (finalize_visible_expr e)))))
					(_cd_replace (finalize_visible_expr _cd_having))
					(map (coalesce _cd_order '()) (lambda (o) (match o '(col dir) (list (_cd_replace (finalize_visible_expr col)) dir))))
					_cd_limit _cd_offset nil nil))
			/* normal: single group stage */
			(if (or group having order limit offset) (list (make_group_stage group having order limit offset nil nil)) '()))))
	/* Contract boundary: untangle_query returns canonical logical IR.
	All case-insensitive parser markers are resolved here, before build_queryplan
	starts creating keytables/prejoins or serializing canonical expression names. */
	(define _canon_fields fields)
	(define _canon_condition conditionAll)
	(define _canon_groups (map (coalesceNil groups '()) (lambda (stage)
		(finalize_logical_stage_scoped stage schemas planner_visible_schemas replace_rename enforce_planner_contract))))
	/* eliminate unused LEFT JOINs: a LEFT JOIN is unused when none of its
	columns appear in fields or group stages. Join predicates reference the
	JOIN alias by construction and must not keep it alive. Only unnested
	aliases are protected explicitly because they may be referenced indirectly. */
	(define _unnested_aliases (map _sq_tbls (lambda (t) (match t '(alias _ _ _ _) alias _ nil))))
	(define _used_tvs (merge_unique
		_unnested_aliases
		(merge (extract_assoc _canon_fields (lambda (k v) (extract_tblvars v))))
		(extract_tblvars _canon_condition)
		(merge (map _canon_groups (lambda (stage)
			(merge_unique
				(merge (map (coalesceNil (stage_group_cols stage) '()) extract_tblvars))
				(extract_tblvars (coalesceNil (stage_having_expr stage) true))
				(merge (map (coalesceNil (stage_order_list stage) '()) (lambda (o) (match o '(col dir) (extract_tblvars col) (extract_tblvars o)))))
				(coalesceNil (stage_partition_aliases stage) '())))))))
	/* prune unused LEFT JOINs and unreferenced .(1) DUAL tables.
	.(1) is only pruned if other tables remain (it's the scan driver otherwise). */
	(define _has_non_dual (reduce tables (lambda (a t) (or a (match t '(_ _ tbl _ _) (not (equal? tbl ".(1)")) true))) false))
	(define _pruned_tables (filter tables (lambda (t) (match t
		'(alias _ tbl isOuter _) (if isOuter (has? _used_tvs alias)
			(if (and _has_non_dual (equal? tbl ".(1)")) (has? _used_tvs alias) true))
		true))))
	/* rebuild condition: drop AND-parts that reference ONLY eliminated aliases */
	(define _elim_aliases (filter (map tables (lambda (t) (match t
		'(alias _ _ true _) (if (has? _used_tvs alias) nil alias)
		nil))) (lambda (x) (not (nil? x)))))
	(define _canon_condition (if (equal? (count _pruned_tables) (count tables)) _canon_condition
		(begin
			/* flatten nested (and ...) to get individual condition parts */
			(define _flatten_and (lambda (expr)
				(match expr (cons (symbol and) parts) (merge (map parts _flatten_and))
					(list expr))))
			(define _cond_parts (_flatten_and _canon_condition))
			/* drop condition parts that reference ANY eliminated alias */
			(define _kept_parts (filter _cond_parts (lambda (part)
				(not (reduce (extract_tblvars part) (lambda (acc tv) (or acc (has? _elim_aliases tv))) false)))))
			(if (equal? 0 (count _kept_parts)) true
				(if (equal? 1 (count _kept_parts)) (car _kept_parts)
					(cons 'and _kept_parts))))))
	(list schema _pruned_tables _canon_fields _canon_condition _canon_groups schemas replace_find_column (coalesceNil (sq_cache "init") '()))
)
))

/*
=== CONTRACT: join_reorder ===

PURPOSE: Optimize table order for physical scan execution.
Determines which table to scan first in a nested-loop join based on
table sizes, available indexes, and predicate selectivity.
Pure physical optimization — does not change query semantics.

INPUT/OUTPUT: 7-tuple (schema tables fields condition groups schemas replace_find_column)

WHAT IT MAY DO:
- Reorder tables within a barrier-free scan segment

WHAT IT MUST NOT DO:
- Transform query structure (that is untangle_query's job)
- Decorrelate subqueries or create joins (that is untangle_query's job)
- Build physical scan plans (that is build_queryplan's job)
- Reorder tables across a join fence (LEFT/SEMI/ANTI JOIN boundary)
*/
/* conservative first pass: only reorder two-table INNER segments when the
second table carries strictly more local WHERE predicates than the first. */
(define jqr_td_alias (lambda (td) (nth td 0)))
(define jqr_td_schema (lambda (td) (nth td 1)))
(define jqr_td_table (lambda (td) (nth td 2)))
(define jqr_td_outer (lambda (td) (nth td 3)))
(define jqr_td_joinexpr (lambda (td) (nth td 4)))
(define jqr_td_with_joinexpr (lambda (td joinexpr)
	(list (jqr_td_alias td) (jqr_td_schema td) (jqr_td_table td) (jqr_td_outer td) joinexpr)))
(define jqr_flatten_join_terms (lambda (tables_)
	(merge (map tables_ (lambda (td)
		(flatten_and_terms (coalesceNil (jqr_td_joinexpr td) true)))))))
(define jqr_local_term_count (lambda (alias terms)
	(reduce terms
		(lambda (acc term) (begin
			(define refs (extract_tblvars term))
			(if (and (not (equal? refs '()))
				(reduce refs (lambda (ok tv) (and ok (equal?? tv alias))) true))
				(+ acc 1)
				acc)))
		0)))
(define jqr_has_order_sensitive_stage (lambda (groups)
	(reduce groups
		(lambda (acc stage)
			(or acc
				(not (equal? (coalesceNil (stage_order_list stage) '()) '()))
				(not (nil? (stage_limit_val stage)))
				(not (nil? (stage_offset_val stage)))))
		false)))
(define jqr_reorder_inner_segment (lambda (segment condition) (begin
	(if (not (equal? (count segment) 2))
		segment
		(begin
			(define td1 (nth segment 0))
			(define td2 (nth segment 1))
			(if (or (jqr_td_outer td1) (jqr_td_outer td2))
				segment
				(begin
					(define condition_terms (flatten_and_terms (coalesceNil condition true)))
					(define local1 (jqr_local_term_count (jqr_td_alias td1) condition_terms))
					(define local2 (jqr_local_term_count (jqr_td_alias td2) condition_terms))
					(if (> local2 local1)
						(list
							(jqr_td_with_joinexpr td2 true)
							(jqr_td_with_joinexpr td1 (combine_and_terms (jqr_flatten_join_terms segment))))
						segment))))))))
(define jqr_reorder_segments (lambda (tables_ condition) (begin
	(match (reduce tables_
		(lambda (state td) (match state
			'(out seg)
			(if (jqr_td_outer td)
				(list (merge out (jqr_reorder_inner_segment seg condition) (list td)) '())
				(list out (merge seg (list td))))
			state))
		(list '() '()))
		'(out seg) (merge out (jqr_reorder_inner_segment seg condition))
		tables_))))
(define join_reorder (lambda (schema tables fields condition groups schemas replace_find_column)
	(list schema
		(if (jqr_has_order_sensitive_stage groups) tables (jqr_reorder_segments tables condition))
		fields condition groups schemas replace_find_column)))

(define build_queryplan_term (lambda (query) (begin
	(define union_parts (query_union_all_parts query))
	(if (nil? union_parts)
		(if (query_is_select_core query)
			(begin
				(define _uq_result (apply untangle_query (merge query (list nil))))
				(define _uq_init (if (>= (count _uq_result) 8) (nth _uq_result 7) '()))
				(define _uq_7tuple (list (nth _uq_result 0) (nth _uq_result 1) (nth _uq_result 2) (nth _uq_result 3) (nth _uq_result 4) (nth _uq_result 5) (nth _uq_result 6)))
				(define _plan (apply build_queryplan (merge (apply join_reorder _uq_7tuple) (list nil))))
				(parallelize_resultrows (if (equal? _uq_init '()) _plan (cons (quote begin) (merge _uq_init (list _plan))))))
			(error "invalid SELECT query term"))
		(match union_parts '(branches order limit offset) (begin
			(if (or (nil? branches) (equal? branches '()))
				(error "UNION ALL requires at least one branch"))
			(define branch_meta (map branches (lambda (branch) (begin
				(if (not (query_is_select_core branch))
					(error "UNION ALL branch must be a SELECT query"))
				(match branch '(schema2 tables2 fields2 condition2 group2 having2 order2 limit2 offset2) (begin
					(if (or (not (nil? order2)) (not (nil? limit2)) (not (nil? offset2)))
						(error "UNION ALL branch ORDER/LIMIT/OFFSET is not supported yet"))
					(define branch_cols (query_branch_field_names branch))
					(list branch branch_cols (count branch_cols) schema2))
					_ (error "UNION ALL branch must be a SELECT query"))
			))))
			(define expected_cols (match branch_meta
				(cons first_meta _) (nth first_meta 2)
				_ 0))
			(define output_cols (match branch_meta
				(cons first_meta _) (nth first_meta 1)
				_ '()))
			(if (or (not (nil? order)) (not (nil? limit)) (not (nil? offset)))
				(error "UNION ALL with global ORDER BY/LIMIT/OFFSET is not supported yet"))
			(if (not (reduce branch_meta (lambda (ok meta) (and ok (equal? (nth meta 2) expected_cols))) true))
				(error "UNION ALL branches must project the same number of columns"))
			(define branch_plans (map branch_meta (lambda (meta) (begin
				(define branch (nth meta 0))
				(define branch_plan (build_queryplan_term branch))
				(define normalized_row (cons (quote list) (merge (map (produceN expected_cols) (lambda (idx)
					(list (nth output_cols idx) (list (quote nth) (symbol "row") (+ (* idx 2) 1)))
				)))))
				(list (quote begin)
					(list (quote set) (symbol "__union_prev_resultrow") (symbol "resultrow"))
					(list (quote set) (symbol "resultrow")
						(list (quote lambda) (list (symbol "row"))
							(list (symbol "__union_prev_resultrow") normalized_row)))
					branch_plan
					(list (quote set) (symbol "resultrow") (symbol "__union_prev_resultrow")))
			))))
			(cons (quote begin) branch_plans)
		))
	)
)))

/* build_dml_plan: route UPDATE/DELETE through the full query planner pipeline.
schema: target schema
target_tbl: target table name (the table being modified)
target_alias: alias of target table (or nil → uses target_tbl)
all_defs: list of table definitions ((alias schema tblname isOuter joinexpr) ...)
cols: flat assoc list (col1 expr1 col2 expr2 ...) for UPDATE, or nil/() for DELETE
condition: WHERE clause expression (raw, not pre-resolved)
order: ORDER BY list or nil
limit_val: LIMIT value or nil
offset_val: OFFSET value or nil
The pipeline resolves inner_selects in SET expressions, handles JOINs, subselects,
column resolution — then projects $update through the target table's scan.
The actual mutation is executed only via a temporary resultrow wrapper after the
full WHERE/join pipeline reached its final leaf. Keep this contract: inner scans
must stay pure row/filter pipelines, and DML side effects happen only at the
same boundary where SELECT would emit result rows. */
(define build_dml_plan (lambda (schema target_tbl target_alias all_defs cols condition order limit_val offset_val) (begin
	(define tgt (coalesce target_alias target_tbl))
	(define is_update (and (not (nil? cols)) (not (equal? cols '()))))
	/* For UPDATE: put SET expressions into synthetic fields so untangle_query processes them
	(including replace_inner_selects for scalar subselects).
	For DELETE: fields are empty — just the tables + condition. */
	(define set_fields (if is_update
		(begin
			(define col_names (extract_assoc cols (lambda (k v) k)))
			(define col_vals (extract_assoc cols (lambda (k v) v)))
			(merge (map (produceN (count col_names)) (lambda (i)
				(list (concat "$set:" (nth col_names i)) (nth col_vals i))))))
		'("$dml_dummy" 1))) /* need at least one field for the pipeline to work */
	/* Build synthetic SELECT 9-tuple: (schema tables fields condition group having order limit offset) */
	(define synthetic_select (list schema all_defs set_fields condition nil nil order limit_val offset_val))
	/* Run through untangle_query → join_reorder → build_queryplan */
	(define _uq_result (apply untangle_query (merge synthetic_select (list nil))))
	(define _uq_init (if (>= (count _uq_result) 8) (nth _uq_result 7) '()))
	(define _uq_7tuple (list (nth _uq_result 0) (nth _uq_result 1) (nth _uq_result 2) (nth _uq_result 3) (nth _uq_result 4) (nth _uq_result 5) (nth _uq_result 6)))
	(define pipeline_result (apply join_reorder _uq_7tuple))
	/* For UPDATE: reconstruct resolved cols from the pipeline's fields */
	(define resolved_target_cols (if is_update
		(begin
			(define resolved_fields (nth pipeline_result 2))
			(define cnames (extract_assoc cols (lambda (k v) k)))
			(merge (map cnames (lambda (cn) (begin
				(define set_key (concat "$set:" cn))
				/* Use a mutable flag (newsession) to track if match was found,
				avoiding equality check on the sentinel (0 == "__not_found__" is buggy) */
				(define _found (newsession))
				(_found "v" nil)
				(reduce_assoc resolved_fields (lambda (acc k v) (if (equal?? k set_key) (begin (_found "v" v) (_found "hit" true) v) acc)) nil)
				(list cn (if (_found "hit") (_found "v") (list (quote get_column) nil false cn false)))
		)))))
		'())) /* DELETE: empty cols signals deletion */
	/* Assemble final pipeline args.
	For UPDATE we must keep the real resolved SET expressions in fields so the
	planner still pulls all helper/materialized columns needed by those
	expressions through later scans. DML output is still suppressed by
	update_target; the fields only preserve planning dependencies. */
	(define final_fields (if is_update
		(nth pipeline_result 2)
		'("$dml" 1)))
	(define dml_tag (concat "__dml:" (fnv_hash (concat schema "|" target_tbl "|" tgt "|" cols "|" condition "|" order "|" limit_val "|" offset_val))))
	(define final_pipeline (list
		(nth pipeline_result 0) /* schema */
		(nth pipeline_result 1) /* tables */
		final_fields
		(nth pipeline_result 3) /* condition */
		(nth pipeline_result 4) /* groups */
		(nth pipeline_result 5) /* schemas */
		(nth pipeline_result 6) /* replace_find_column */
		(list tgt resolved_target_cols dml_tag) /* update_target: (alias cols dml_tag) — empty cols = DELETE */
	))
	(define dml_plan (apply build_queryplan final_pipeline))
	(define dml_prev_rr (symbol "__dml_prev_resultrow"))
	(define dml_rc (symbol "__dml_result_count"))
	(define wrapped_plan (list (quote begin)
		(list (quote set) dml_prev_rr (symbol "resultrow"))
		(list (quote set) (symbol "resultrow")
			(list (quote lambda) (list (symbol "item"))
				(list (quote if) (list (quote or)
					(list (quote nil?) (list (quote get_assoc) (symbol "item") "__update"))
					(list (quote not) (list (quote equal??) (list (quote get_assoc) (symbol "item") "__dml_tag") dml_tag)))
					0
					(list (quote if) (list (quote nil?) (list (quote get_assoc) (symbol "item") "__values"))
						(list (quote if) (list (quote apply) (list (quote get_assoc) (symbol "item") "__update") nil) 1 0)
						(list (quote if) (list (quote apply) (list (quote get_assoc) (symbol "item") "__update") (list (quote list) (list (quote get_assoc) (symbol "item") "__values"))) 1 0)))))
		(list (quote define) dml_rc dml_plan)
		(list (quote set) (symbol "resultrow") dml_prev_rr)
		dml_rc))
	(if (equal? _uq_init '()) wrapped_plan (cons (quote begin) (merge _uq_init (list wrapped_plan))))
)))

/* Convenience wrapper for multi-table UPDATE (called from sql_update) */
(define build_multi_table_update (lambda (schema tbl tblalias all_defs cols condition)
	(build_dml_plan schema tbl tblalias all_defs cols condition nil nil nil)))

/*
=== CONTRACT: build_queryplan ===

PURPOSE: Generate physical execution plans from the logical IR.
Takes a flat, already-reordered table list and translates it into executable SCM.

INPUT:  7-tuple (schema tables fields condition groups schemas replace_find_column)
After join_reorder, tables are in optimal scan order.

OUTPUT: executable Scheme expression (scan, keytable operations, resultrow, etc.)

WHAT IT DOES:
- Resolves get_column markers to variable references via replace_find_column
- Processes GROUP BY stages: creates keytables, collect/compute/grouped plans
- Processes ORDER BY / LIMIT: generates scan_order with offset/limit
- Generates nested scan loops via build_scan (follows table order from join_reorder)
- Handles window functions (ORC, aggregate, LAG/LEAD)

WHAT IT MUST NOT DO:
- Reorder tables (that is join_reorder's job)
- Flatten derived tables or unnest subqueries (that is untangle_query's job)
- Re-introduce logical subquery semantics. If build_queryplan still needs
inner_select/subscan/materialized-derived-source behavior, untangle_query has
not finished its job.

GROUP BY AGGREGATE PIPELINE:
1. collect_plan: extract unique group keys from base table into a keytable
2. compute_plan: for each aggregate, scan base table per group key,
store results as keytable columns named "expr|condition"
3. grouped_plan: scan populated keytable for final output (ORDER BY, HAVING, LIMIT)
*/
/*
=== build_queryplan: physical plan generation ===

Translates the flat logical IR from untangle_query into executable SCM scan code.
Consumes the table list, conditions, and group-stages and produces nested scan/scan_order
calls, keytable materialization (GROUP BY), and prejoin materialization (multi-table GROUP).

Processing order (recursive — each stage peels off one layer):
1. Group-stages with partition-aliases (scoped): separate into keytable fill + post-group scan
- Single-table group: make_keytable + collect keys + createcolumn per aggregate
- Multi-table group: prejoin materialization + keytable on the prejoin
- Aggregates are discovered in fields, order, having, AND condition (Neumann EXISTS/IN rewrite)
2. Partition-stages (LIMIT per partition): scan_order with partition columns
3. ORDER BY / LIMIT / OFFSET: scan_order on the remaining tables
4. Unordered scan: nested-loop scan over remaining tables

Key helpers:
- make_keytable: creates sloppy temp table for group keys + computed aggregate columns
- split_condition: selection pushdown — splits AND-parts by which tables they reference
- replace_columns_from_expr: rewrites get_column markers to runtime variable references
- scan_wrapper: generates scan/scan_order calls with filter/map/reduce structure
*/
/* update_target: nil for SELECT, or (tblalias (col1 expr1 col2 expr2 ...)) for multi-table UPDATE.
When set, the scan on tblalias includes $update in mapcols and the mapfn applies the SET expressions. */
(define build_queryplan (lambda (schema tables fields condition groups schemas replace_find_column update_target) (begin
	/*(print "build queryplan " '(schema tables fields condition groups schemas))*/

	/* TODO: order tables: outer joins behind */
	(set groups (coalesceNil groups '()))
	/* separate partition stages (have partition-aliases) from regular stages */
	/* separate partition stages (have aliases but NO group-cols) from regular/scoped group stages */
	(define partition_stages (filter groups (lambda (s) (begin
		(define _spa (stage_partition_aliases s))
		(define _sg (stage_group_cols s))
		(and (not (nil? _spa)) (or (nil? _sg) (equal? _sg '())))))))
	(set groups (filter groups (lambda (s) (begin
		(define _spa (stage_partition_aliases s))
		(define _sg (stage_group_cols s))
		(or (nil? _spa) (and (not (nil? _sg)) (not (equal? _sg '()))))))))
	(define groups_present (and (not (nil? groups)) (not (equal? groups '()))))
	(define stage (if groups_present (car groups) nil))
	(define rest_groups (if groups_present (cdr groups) nil))
	(set rest_groups (coalesceNil rest_groups '()))
	(define stage_group (if stage (stage_group_cols stage) nil))
	(define stage_having (if stage (stage_having_expr stage) nil))
	(define stage_order (if stage (stage_order_list stage) nil))
	(define stage_partcols (if stage (coalesceNil (stage_limit_partition_cols stage) 0) 0))
	(define stage_limit (if stage (stage_limit_val stage) nil))
	(define stage_offset (if stage (stage_offset_val stage) nil))

	/* window function detection */
	(define window_funcs_all (merge (extract_assoc fields (lambda (k v) (extract_window_funcs v)))))
	(define has_window (not (equal? window_funcs_all '())))
	/* Case 10: window functions in WHERE clause */
	(define window_in_condition (not (equal? (extract_window_funcs (coalesceNil condition true)) '())))
	/* Design contract:
	Materialized temp sources may expose aggregate results as physical temp
	columns, but logical aggregate sentinels must survive until the scan stage
	that actually reads that temp source. Lower them exactly once here. */
	(define lower_materialized_scan_expr (lambda (scan_schema scan_tbl scan_tblvar scan_expr agg_name_context) (begin
		(define materialized_source (materialized-source? scan_tbl))
		(if (not materialized_source)
			scan_expr
			(begin
				(define canon_alias_map (list (list scan_tblvar (concat scan_schema "." scan_tbl))))
				(define scan_expr_name (lambda (expr)
					(canonical_expr_name (normalize_canonical_aliases (lower_materialized_source_expr scan_tbl scan_tblvar expr)) '(list) '(list) canon_alias_map)))
				(define agg_col_name (lambda (ag)
					(concat (scan_expr_name ag) "|" (scan_expr_name agg_name_context) (runtime_cache_suffix_from_exprs (list ag agg_name_context)))))
				(define materialized_cols (materialized_source_physical_schema scan_schema scan_tbl scan_tblvar schemas))
				(define lookup_expr_field (lambda (expr) (begin
					(define expr_lookup (materialized_source_expr_lookup scan_tbl))
					(define expr_keys (materialized_source_expr_keys expr))
					(define direct_field (if (nil? expr_lookup) nil
						(reduce expr_keys (lambda (found key)
							(if (not (nil? found))
								found
								(coalesce (expr_lookup key) nil)))
							nil)))
					(if (not (nil? direct_field))
						direct_field
						(begin
							(define normalized_expr (normalize_canonical_aliases expr))
							(reduce materialized_cols (lambda (found coldef)
								(if (not (nil? found))
									found
									(begin
										(define source_expr (coalesceNil (coldef "Expr") nil))
										(if (and (not (nil? source_expr))
											(or (equal? (normalize_canonical_aliases source_expr) normalized_expr)
												(reduce expr_keys (lambda (matched key)
													(or matched (has? (materialized_source_expr_keys source_expr) key)))
													false)))
											(coldef "Field")
											nil))))
								nil))))))
				(define current_scan_agg_field (lambda (expr agg_args)
					(coalesce
						(lookup_expr_field expr)
						(begin
							(define target_col (agg_col_name agg_args))
							(reduce materialized_cols (lambda (found coldef)
								(if (not (nil? found))
									found
									(if (equal? (coldef "Field") target_col) target_col nil)))
								nil)))))
				(define lower_aggs (lambda (expr) (match expr
					(cons (symbol aggregate) agg_args) (begin
						(define match_col (current_scan_agg_field expr agg_args))
						(if (nil? match_col)
							expr
							(list (quote get_column) scan_tblvar false match_col false)))
					(cons '(quote aggregate) agg_args) (begin
						(define match_col (current_scan_agg_field expr agg_args))
						(if (nil? match_col)
							expr
							(list (quote get_column) scan_tblvar false match_col false)))
					(cons sym args) (cons sym (map args lower_aggs))
					expr)))
				(lower_aggs scan_expr))))))
	(define lower_materialized_scan_condition (lambda (scan_schema scan_tbl scan_tblvar scan_condition)
		(lower_materialized_scan_expr scan_schema scan_tbl scan_tblvar scan_condition scan_condition)))
	(define lower_materialized_emit_expr (lambda (scan_schema scan_tbl scan_tblvar scan_expr)
		(lower_materialized_scan_expr scan_schema scan_tbl scan_tblvar scan_expr true)))
	(define lower_materialized_emit_assoc (lambda (scan_schema scan_tbl scan_tblvar exprs)
		(map_assoc exprs (lambda (k v) (lower_materialized_emit_expr scan_schema scan_tbl scan_tblvar v)))))
	(if window_in_condition (error "window functions not allowed in WHERE clause"))

	/* window functions with GROUP BY: strip window expressions to inner
	aggregates so the normal GROUP BY path processes them. Save original
	fields so we can inject promise values after compute_plan. */
	(define _wg_store (newsession))
	(_wg_store "fields" nil)
	(if (and has_window stage_group) (begin
		(_wg_store "fields" fields) /* save original fields with window_func */
		(define strip_window_inner (lambda (expr)
			(if (and (list? expr) (> (count expr) 0) (equal?? (car expr) (quote window_func)))
				(begin (define args (nth expr 2))
					(if (and (list? args) (> (count args) 0)) (car args) 1))
				(if (list? expr) (map expr strip_window_inner) expr))))
		(set fields (map_assoc fields (lambda (k v) (strip_window_inner v))))
		(set has_window false)))

	(if stage_group (begin
		/* merge stage-condition from scoped stages into the local condition.
		This injects the inner subquery WHERE exactly when the owning stage
		is processed, preventing cross-stage condition leakage.
		stage-condition is already prefixed — merge AFTER replace_find_column
		to avoid crash on prefixed aliases that replace_find_column doesn't know. */
		(define scoped_cond (stage_condition stage))
		/* group: extract aggregate clauses and split the query into two parts: gathering the aggregates and outputting them */
		/* Design contract:
		Keep get_column / aggregate / window sentinels logical until the final scan
		code is emitted. A GROUP stage may resolve expressions for its own
		materialization, but when it wraps itself into a recursive prejoin/materialized
		stage it must forward the original logical AST into the next stage. Otherwise
		physical temp field names from an earlier materialization become part of the
		next stage's logical keys/fields and explode into nested "(get_column ...)"
		temp names. */
		(define raw_stage_group stage_group)
		(define raw_stage_having stage_having)
		/* Compatibility name: current planner stores the logical post-group
		predicate in HAVING. Keep an explicit raw alias here so recursive
		prejoin/group planning cannot silently drop it by referring to an
		undefined symbol. */
		(define raw_stage_post_group_condition raw_stage_having)
		(define raw_stage_order stage_order)
		(define raw_fields fields)
		(set stage_group (map stage_group replace_find_column))
		(set stage_having (replace_find_column stage_having))
		(set stage_order (map stage_order (lambda (o) (match o '(col dir) (list (replace_find_column col) dir)))))
		(define is_dedup (stage_is_dedup stage))
		(define _scoped_stage (not (nil? (stage_partition_aliases stage))))
		(define _field_agg_has_nested_agg (lambda (args)
			(reduce args (lambda (acc arg)
				(or acc (not (equal? (extract_aggregates arg) '()))))
				false)))
		(define _needs_outer_group_expr (lambda (expr) (match expr
			(cons (symbol aggregate) args)
			(and (equal? (extract_tblvars expr) '()) (_field_agg_has_nested_agg args))
			(cons '(quote aggregate) args)
			(and (equal? (extract_tblvars expr) '()) (_field_agg_has_nested_agg args))
			(cons _ args) (reduce args (lambda (acc arg) (or acc (_needs_outer_group_expr arg))) false)
			false)))
		/* Only NON-scoped (global) later group stages trigger aggregate deferral.
		Other scoped stages from Neumann unnesting are independent — their
		aggregates are processed in their own recursive build_queryplan call,
		not by deferring to a later global stage. */
		(define _has_existing_later_group_stage (reduce rest_groups (lambda (acc s)
			(or acc (begin
				(define _later_sg (stage_group_cols s))
				(define later_is_scoped (not (nil? (stage_partition_aliases s))))
				(and (not later_is_scoped) (not (nil? _later_sg)) (not (equal? _later_sg '()))))))
			false))
		(define _needs_synthetic_outer_group (and _scoped_stage
			(not _has_existing_later_group_stage)
			(or
				(reduce_assoc fields (lambda (acc _k expr) (or acc (_needs_outer_group_expr expr))) false)
				(_needs_outer_group_expr (coalesce stage_having true))
				(reduce (coalesce stage_order '()) (lambda (acc o)
					(or acc (match o '(col _dir) (_needs_outer_group_expr col) false))) false))))
		(define _has_later_group_stage (or _has_existing_later_group_stage _needs_synthetic_outer_group))
		(define _defer_field_agg (lambda (expr args)
			(and (equal? (extract_tblvars expr) '())
				_scoped_stage
				_has_later_group_stage
				(_field_agg_has_nested_agg args))))
		(define extract_stage_field_aggregates (lambda (expr deferred_outer) (match expr
			(cons (symbol aggregate) args)
			(if (and (not deferred_outer) (_defer_field_agg expr args))
				(merge (map args (lambda (arg) (extract_stage_field_aggregates arg true))))
				(list args))
			(cons '(quote aggregate) args)
			(if (and (not deferred_outer) (_defer_field_agg expr args))
				(merge (map args (lambda (arg) (extract_stage_field_aggregates arg true))))
				(list args))
			(cons sym args) (merge (map args (lambda (arg) (extract_stage_field_aggregates arg deferred_outer))))
			'())))
		/* collect all unique aggregate tuples (expr reduce neutral) from fields, ORDER BY, and HAVING.
		Each tuple becomes a computed column on the keytable, e.g. SUM(amount) -> ((get_column t amount) + 0).
		ORDER BY SUM(x) requires SUM(x) to be pre-computed here even if not in SELECT. */
		(define ags_raw (if is_dedup '() (extract_assoc fields (lambda (key expr) (extract_stage_field_aggregates expr false)))))
		(define ags (if is_dedup '() (merge_unique ags_raw))) /* aggregates in fields */
		(define ags (if is_dedup ags (merge_unique ags (merge_unique (map (coalesce stage_order '()) (lambda (x) (match x '(col dir) (extract_aggregates col)))))))) /* aggregates in order */
		(define ags (if is_dedup ags (merge_unique ags (extract_aggregates (coalesce stage_having true))))) /* aggregates in having */
		(define ags (if is_dedup ags (merge_unique ags (extract_aggregates (coalesce condition true))))) /* aggregates in condition (from Neumann EXISTS/IN rewrite) */
		/* TODO: replace (get_column nil ti col ci) in group, having and order with (coalesce (fields col) '('get_column nil false col false)) */

		/* determine which tables the GROUP BY applies to:
		- if stage has partition-aliases (scoped): only those tables
		- otherwise (global): all tables except partition-staged ones */
		(define _grp_ps_aliases (merge (map partition_stages (lambda (s) (coalesceNil (stage_partition_aliases s) '())))))
		(define _stage_scope (stage_partition_aliases stage))
		(define _grp_tables_raw (if (not (nil? _stage_scope))
			/* scoped GROUP: only the tables listed in the stage's aliases */
			(filter tables (lambda (t) (match t '(tv _ _ _ _) (has? _stage_scope tv) false)))
			/* global GROUP: all tables except partition-staged */
			(filter tables (lambda (t) (match t '(tv _ _ _ _) (not (has? _grp_ps_aliases tv)) true)))))
		(define _grp_ps_tables_raw (filter tables (lambda (t) (match t '(tv _ _ _ _)
			(and (not (has? (coalesceNil _stage_scope '()) tv))
				(or (has? _grp_ps_aliases tv) (not (nil? _stage_scope))))
			false))))
		(define _grp_ps_visible_aliases (merge_unique (map _grp_ps_tables_raw (lambda (td) (match td
			'(tv tschema ttbl _ _)
			(filter (list
				tv
				(visible_occurrence_alias tv)
				(if (equal? (visible_occurrence_alias tv) ttbl) (concat tschema "." ttbl) nil))
				(lambda (alias_) (not (nil? alias_))))
			'())))))
		(define _expr_refs_grp_ps_table (lambda (expr)
			(reduce (extract_all_get_columns expr) (lambda (acc mc)
				(or acc (match mc
					'(name '((symbol get_column) alias_ _ _ _))
					(or (has? _grp_ps_visible_aliases alias_)
						(has? _grp_ps_visible_aliases (visible_occurrence_alias alias_)))
					'(name '((quote get_column) alias_ _ _ _))
					(or (has? _grp_ps_visible_aliases alias_)
						(has? _grp_ps_visible_aliases (visible_occurrence_alias alias_)))
					false)))
				false)))
		(define _resolved_expr_refs_grp_ps_table (lambda (expr)
			(_expr_refs_grp_ps_table (replace_find_column expr))))
		(define _must_prejoin_outer_group_tables (and
			(not (equal? _grp_ps_tables_raw '()))
			(or
				(reduce stage_group (lambda (acc expr) (or acc (_resolved_expr_refs_grp_ps_table expr))) false)
				(reduce ags (lambda (acc ag) (or acc (_resolved_expr_refs_grp_ps_table ag))) false)
				(_resolved_expr_refs_grp_ps_table (coalesce stage_having true))
				(reduce (coalesce stage_order '()) (lambda (acc o) (or acc (match o '(col _dir) (_resolved_expr_refs_grp_ps_table col) false))) false))))
		(define _grp_tables (if _must_prejoin_outer_group_tables
			(merge _grp_tables_raw _grp_ps_tables_raw)
			_grp_tables_raw))
		(define _grp_ps_tables (if _must_prejoin_outer_group_tables
			'()
			_grp_ps_tables_raw))
		(match _grp_tables
			/* TODO: allow for more than just group by single table */
			/* TODO: outer tables that only join on group */
			'('(tblvar schema tbl isOuter _)) (begin
				(define ags (filter ags (lambda (ag) (match ag
					/* scope-tagged aggregate: accept only if scope matches current stage */
					'(agg_expr _ _ scope_alias)
					(if (nil? _stage_scope) true (has? _stage_scope scope_alias))
					'(agg_expr _ _)
					(begin
						(define refs (extract_tblvars agg_expr))
						(or (equal? refs '())
							(has? refs tblvar)))
					false))))
				/* strip scope-tag for downstream (expects 3-element tuples) */
				(set ags (map ags (lambda (ag) (match ag '(a b c _s) (list a b c) ag))))
				/* prepare preaggregate */
				(define canon_alias_map (list (list tblvar (concat schema "." tbl))))
				(define materialized_source (materialized-source? tbl))
				(define expr_name (lambda (expr)
					(sanitize_temp_name
						(canonical_expr_name (normalize_canonical_aliases (lower_materialized_source_expr tbl tblvar expr)) '(list) '(list) canon_alias_map))))
				(define count_ag '(1 + 0))
				(define canonical_count_col_name (lambda ()
					(concat "COUNT(*)|" (expr_name condition) (runtime_cache_suffix_from_exprs (list condition)))))
				(define agg_col_name (lambda (ag)
					(if (equal? ag count_ag)
						(canonical_count_col_name)
						(concat (expr_name ag) "|" (expr_name condition) (runtime_cache_suffix_from_exprs (list ag condition))))))
				(define rewrite_materialized_source_aggs_single (lambda (expr) (match expr
					(cons (symbol aggregate) agg_args) (begin
						(define target_col (agg_col_name agg_args))
						(define materialized_cols (materialized_source_physical_schema schema tbl tblvar schemas))
						(define match_col (reduce materialized_cols (lambda (found coldef)
							(if (not (nil? found)) found
								(begin
									(define field_name (coldef "Field"))
									(if (equal? field_name target_col)
										field_name
										nil))))
							nil))
						(if (nil? match_col)
							expr
							(list (quote get_column) tblvar false match_col false)))
					(cons '(quote aggregate) agg_args) (begin
						(define target_col (agg_col_name agg_args))
						(define materialized_cols (materialized_source_physical_schema schema tbl tblvar schemas))
						(define match_col (reduce materialized_cols (lambda (found coldef)
							(if (not (nil? found)) found
								(begin
									(define field_name (coldef "Field"))
									(if (equal? field_name target_col)
										field_name
										nil))))
							nil))
						(if (nil? match_col)
							expr
							(list (quote get_column) tblvar false match_col false)))
					(cons sym args) (cons sym (map args rewrite_materialized_source_aggs_single))
					expr)))
				(define rewrite_materialized_source_cols_single (lambda (expr) (match expr
					'((symbol get_column) _ _ _ _) (begin
						(define expr_lookup (materialized_source_expr_lookup tbl))
						(define visible_field_expr (match expr
							'((symbol get_column) alias_ _ col _)
							(if (or (nil? alias_) (not (has_assoc? schemas alias_))) nil
								(reduce (coalesceNil (schemas alias_) '()) (lambda (found coldef)
									(if (not (nil? found))
										found
										(if (and (equal? (coldef "Field") col) (has_assoc? coldef "Expr"))
											(coalesceNil (coldef "Expr") nil)
											nil)))
									nil))
							nil))
						(define direct_field (if (nil? expr_lookup) nil
							(reduce (materialized_source_expr_keys expr) (lambda (found key)
								(if (not (nil? found)) found
									(coalesce (expr_lookup key) nil)))
								nil)))
						(if (not (nil? direct_field))
							(list (quote get_column) tblvar false direct_field false)
							(begin
								(define materialized_cols (materialized_source_physical_schema schema tbl tblvar schemas))
								(define normalized_expr (normalize_canonical_aliases expr))
								(define match_col (reduce materialized_cols (lambda (found coldef)
									(if (not (nil? found))
										found
										(begin
											(define source_expr (coalesceNil (coldef "Expr") nil))
											(if (and (not (nil? source_expr))
												(equal? (normalize_canonical_aliases source_expr) normalized_expr))
												(coldef "Field")
												nil))))
									nil))
								(if (nil? match_col)
									(if (nil? visible_field_expr)
										expr
										(rewrite_materialized_source_cols_single visible_field_expr))
									(list (quote get_column) tblvar false match_col false)))))
					'((quote get_column) _ _ _ _) (begin
						(define expr_lookup (materialized_source_expr_lookup tbl))
						(define visible_field_expr (match expr
							'((quote get_column) alias_ _ col _)
							(if (or (nil? alias_) (not (has_assoc? schemas alias_))) nil
								(reduce (coalesceNil (schemas alias_) '()) (lambda (found coldef)
									(if (not (nil? found))
										found
										(if (and (equal? (coldef "Field") col) (has_assoc? coldef "Expr"))
											(coalesceNil (coldef "Expr") nil)
											nil)))
									nil))
							nil))
						(define direct_field (if (nil? expr_lookup) nil
							(reduce (materialized_source_expr_keys expr) (lambda (found key)
								(if (not (nil? found)) found
									(coalesce (expr_lookup key) nil)))
								nil)))
						(if (not (nil? direct_field))
							(list (quote get_column) tblvar false direct_field false)
							(begin
								(define materialized_cols (materialized_source_physical_schema schema tbl tblvar schemas))
								(define normalized_expr (normalize_canonical_aliases expr))
								(define match_col (reduce materialized_cols (lambda (found coldef)
									(if (not (nil? found))
										found
										(begin
											(define source_expr (coalesceNil (coldef "Expr") nil))
											(if (and (not (nil? source_expr))
												(equal? (normalize_canonical_aliases source_expr) normalized_expr))
												(coldef "Field")
												nil))))
									nil))
								(if (nil? match_col)
									(if (nil? visible_field_expr)
										expr
										(rewrite_materialized_source_cols_single visible_field_expr))
									(list (quote get_column) tblvar false match_col false)))))
					(cons sym args) (if (_is_opaque_scope_sym sym)
						expr
						(cons sym (map args rewrite_materialized_source_cols_single)))
					expr)))
				(define lower_visible_materialized_aggs_single (lambda (expr) (match expr
					(cons (symbol aggregate) agg_args) (begin
						(define agg_name (canonical_expr_name (normalize_canonical_aliases agg_args) '(list) '(list) canon_alias_map))
						(define match_col (reduce tables (lambda (acc td)
							(if (not (nil? acc))
								acc
								(match td '(tv tschema ttbl _ _)
									(begin
										(define source_cols (materialized_source_physical_schema tschema ttbl tv schemas))
										(reduce source_cols (lambda (found coldef)
											(if (not (nil? found)) found
												(begin
													(define field_name (coldef "Field"))
													(if (and (>= (strlen field_name) (+ (strlen agg_name) 1))
														(equal? (substr field_name 0 (strlen agg_name)) agg_name)
														(equal? (substr field_name (strlen agg_name) 1) "|"))
														(list tv field_name)
														nil))))
											nil)))))
							nil))
						(if (nil? match_col)
							(match agg_args
								'(agg_expr agg_reduce agg_neutral)
								(list (quote aggregate) (lower_visible_materialized_aggs_single agg_expr) agg_reduce agg_neutral)
								_ expr)
							(list (quote get_column) (car match_col) false (cadr match_col) false)))
					(cons '(quote aggregate) agg_args) (begin
						(define agg_name (canonical_expr_name (normalize_canonical_aliases agg_args) '(list) '(list) canon_alias_map))
						(define match_col (reduce tables (lambda (acc td)
							(if (not (nil? acc))
								acc
								(match td '(tv tschema ttbl _ _)
									(begin
										(define source_cols (materialized_source_physical_schema tschema ttbl tv schemas))
										(reduce source_cols (lambda (found coldef)
											(if (not (nil? found)) found
												(begin
													(define field_name (coldef "Field"))
													(if (and (>= (strlen field_name) (+ (strlen agg_name) 1))
														(equal? (substr field_name 0 (strlen agg_name)) agg_name)
														(equal? (substr field_name (strlen agg_name) 1) "|"))
														(list tv field_name)
														nil))))
											nil)))))
							nil))
						(if (nil? match_col)
							(match agg_args
								'(agg_expr agg_reduce agg_neutral)
								(list (quote aggregate) (lower_visible_materialized_aggs_single agg_expr) agg_reduce agg_neutral)
								_ expr)
							(list (quote get_column) (car match_col) false (cadr match_col) false)))
					(cons sym args) (if (_is_opaque_scope_sym sym)
						expr
						(cons sym (map args lower_visible_materialized_aggs_single)))
					expr)))
				(define resolved_stage_group (if materialized_source
					(map stage_group rewrite_materialized_source_cols_single)
					stage_group))
				(if materialized_source
					(set ags (map ags rewrite_materialized_source_cols_single)))
				/* MySQL-style grouped projections may still contain row-local expressions
				that are neither GROUP keys nor explicit aggregates. Those expressions must
				not be rewritten into fictitious keytable columns by recursively replacing
				their inner get_column markers. Instead, synthesize a stable "pick any
				non-nil" aggregate for the whole expression and fetch that aggregate like
				any other grouped value. This keeps grouped output generic and avoids
				phantom columns such as (get_column "...") on the keytable. */
				(define _group_any_reduce
					(list (quote lambda)
						(list (quote acc) (quote item))
						(list (quote if)
							(list (quote equal?) (quote item) nil)
							(quote acc)
							(list (quote if)
								(list (quote equal?) (quote acc) nil)
								(quote item)
								(quote acc)))))
				(define _group_value_ag (lambda (expr)
					(list expr _group_any_reduce nil)))
				(define _group_value_ag_expr (lambda (expr)
					(list (quote aggregate) expr _group_any_reduce nil)))
				(define _outer_stage_aliases (map _grp_ps_tables (lambda (td) (match td
					'(tv _ ttbl _ _) (if (nil? tv) ttbl tv)
					""))))
				(define _refs_only_outer_stage (lambda (expr)
					(begin
						(define _refs (extract_tblvars expr))
						(and (not (equal? _refs '()))
							(reduce _refs (lambda (acc tv) (and acc (has? _outer_stage_aliases tv))) true)))))
				(define _matches_group_expr (lambda (expr)
					(or
						(reduce stage_group (lambda (acc group_expr) (or acc (equal? group_expr expr))) false)
						(reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr expr))) false)
						(reduce resolved_stage_group (lambda (acc group_expr)
							(or acc (equal? group_expr (rewrite_materialized_source_cols_single expr))))
							false))))
				(define _field_has_agg_expr (lambda (expr) (match expr
					(cons (symbol aggregate) _) true
					(cons '(quote aggregate) _) true
					(cons sym args) (reduce args (lambda (a b) (or a (_field_has_agg_expr b))) false)
					false)))
				(define _expr_has_non_group_column_refs (lambda (expr)
					(reduce (extract_all_get_columns expr) (lambda (acc mc)
						(or acc
							(not (_matches_group_expr (cadr mc)))))
						false)))
				(define _field_needs_group_value_agg (lambda (expr)
					(and (not (_field_has_agg_expr expr))
						(not (_matches_group_expr expr))
						(not (and (not (nil? _stage_scope)) (_refs_only_outer_stage expr)))
						(_expr_has_non_group_column_refs expr))))
				/* Materialized sources may expose grouped pass-through fields as
				row-local LEFT-JOIN wrapper expressions. Aggregate them only after
				materializing that wrapper once as a temp source column; otherwise the
				later createcolumn/cache path can mistake the aggregate name for a
				physical source column on the underlying temp table. */
				(define group_value_local_key_expr (lambda (expr)
					(if materialized_source
						(rewrite_materialized_source_cols_single expr)
						expr)))
				(define group_value_local_col_name (lambda (expr)
					(begin
						(define logical_expr (if materialized_source
							(lower_materialized_source_expr tbl tblvar expr)
							expr))
						(define ref_key (sha1 (string
							(map (extract_all_get_columns logical_expr) (lambda (mc)
								(sanitize_temp_name (string (normalize_canonical_aliases (cadr mc)))))))))
						(define head_key (match logical_expr
							'((symbol get_column) _ _ col _) (concat "col:" (sanitize_temp_name col))
							'((quote get_column) _ _ col _) (concat "col:" (sanitize_temp_name col))
							(cons sym _) (sanitize_temp_name (string sym))
							_ (sanitize_temp_name (string logical_expr))))
						(concat ".group_value|" head_key "|" ref_key))))
				(define group_value_local_lookup (newsession))
				(define group_value_local_head_lookup (newsession))
				(define group_value_local_head_count (newsession))
				(define group_value_local_head_key (lambda (expr)
					(begin
						(define logical_expr (if materialized_source
							(lower_materialized_source_expr tbl tblvar expr)
							expr))
						(match logical_expr
							'((symbol get_column) _ _ col _) (concat "col:" (sanitize_temp_name col))
							'((quote get_column) _ _ col _) (concat "col:" (sanitize_temp_name col))
							(cons sym _) (sanitize_temp_name (string sym))
							_ (sanitize_temp_name (string logical_expr))))))
				(define group_value_local_expr (lambda (expr)
					(if (and materialized_source (_field_needs_group_value_agg expr))
						(begin
							(define key_expr (group_value_local_key_expr expr))
							(define logical_expr (if materialized_source
								(lower_materialized_source_expr tbl tblvar expr)
								expr))
							(define match_col (reduce
								(merge (materialized_source_expr_keys expr)
									(materialized_source_expr_keys key_expr)
									(materialized_source_expr_keys logical_expr))
								(lambda (found key)
									(if (not (nil? found))
										found
										(coalesce (group_value_local_lookup key) nil)))
								nil))
							(define head_key (group_value_local_head_key expr))
							(define head_col (if (equal? (coalesceNil (group_value_local_head_count head_key) 0) 1)
								(coalesce (group_value_local_head_lookup head_key) nil)
								nil))
							(list (quote get_column) tblvar false (coalesce match_col head_col (group_value_local_col_name expr)) false))
						expr)))
				(define group_value_local_setup_expr (lambda (expr) (begin
					(define lowered_expr (group_value_local_key_expr expr))
					(define col_name (group_value_local_col_name expr))
					(define cols (extract_columns_for_tblvar tblvar lowered_expr))
					(list (quote createcolumn) schema tbl col_name "any" '(list) '(list "temp" true)
						(cons (quote list) cols)
						(list (quote lambda) (map cols (lambda (col) (symbol (concat tblvar "." col))))
							(replace_columns_from_expr lowered_expr))))))
				(define group_value_local_fields (if materialized_source
					/* Keep each row-local grouped projection as one logical AST. `merge`
					would flatten list-valued expressions like `(if ...)` to their head
					symbol and arguments, which then materializes nonsense temp columns
					such as `(lambda () if)`. */
					(merge_unique (extract_assoc fields (lambda (_key expr)
						(if (_field_needs_group_value_agg expr)
							(list expr)
							'()))))
					'()))
				(if materialized_source
					(map group_value_local_fields (lambda (expr) (begin
						(define col_name (group_value_local_col_name expr))
						(define key_expr (group_value_local_key_expr expr))
						(define logical_expr (if materialized_source
							(lower_materialized_source_expr tbl tblvar expr)
							expr))
						(define head_key (group_value_local_head_key expr))
						(group_value_local_head_count head_key (+ 1 (coalesceNil (group_value_local_head_count head_key) 0)))
						(if (nil? (group_value_local_head_lookup head_key))
							(group_value_local_head_lookup head_key col_name)
							nil)
						(map (merge (materialized_source_expr_keys expr)
							(materialized_source_expr_keys key_expr)
							(materialized_source_expr_keys logical_expr))
							(lambda (key) (group_value_local_lookup key col_name)))
						nil))))
				(define group_value_local_compute_plan (if (equal? group_value_local_fields '()) nil
					(list (quote time)
						(cons (quote parallel) (map group_value_local_fields group_value_local_setup_expr))
						"group-value")))
				(define synthetic_field_ags (if is_dedup '()
					(merge (extract_assoc fields (lambda (_key expr)
						(if (_field_needs_group_value_agg expr)
							(list (_group_value_ag (group_value_local_expr expr)))
							'()))))))
				(define ags (if is_dedup ags (merge_unique ags synthetic_field_ags)))
				(define lower_materialized_agg_tuple (lambda (ag) (match ag
					'(ag_expr ag_reduce ag_neutral)
					(list
						(rewrite_materialized_source_cols_single
							(rewrite_materialized_source_aggs_single ag_expr))
						ag_reduce
						ag_neutral)
					_ ag)))
				(if materialized_source
					(set ags (map ags lower_materialized_agg_tuple)))

				/* preparation */
				(define tblvar_cols (merge_unique (map resolved_stage_group (lambda (col) (extract_columns_for_tblvar tblvar col)))))
				(set condition (replace_find_column (coalesceNil condition true)))
				/* merge stage-condition AFTER replace_find_column (already prefixed) */
				(if (and (not (nil? scoped_cond)) (not (equal? scoped_cond true)))
					(set condition (if (or (nil? condition) (equal? condition true))
						scoped_cond
						(list (quote and) condition scoped_cond))))
				(set condition (lower_visible_materialized_aggs_single condition))
				(if materialized_source
					(set condition (rewrite_materialized_source_aggs_single condition)))
				(define _flatten_and_parts (lambda (expr) (match expr
					(cons sym parts) (if (or (equal? sym (quote and)) (equal? sym '(quote and)) (equal? sym 'and))
						(merge (map parts _flatten_and_parts))
						(list expr))
					(list expr))))
				(define _condition_parts0 (_flatten_and_parts condition))
				/* Old runtime-local temp-column materialization pushed session-sensitive
				row predicates into createcolumn lambdas. That leaks the query-only
				session scope into storage compute code. Keep the predicate in the
				normal grouped scan and let the tx-aware compute/cache layer handle
				session variants instead of precomputing .runtime_pred temp columns. */
				(define runtime_local_compute_plan nil)
				/* 2-phase condition split:
				Phase 1: separate aggregate-containing AND-parts from non-aggregate parts.
				Aggregates cannot be evaluated as row filters — they need the keytable.
				Phase 2 (after keytable creation): replace aggregates with get_column refs,
				then split by table references for pushdown. */
				(define _has_agg_expr (lambda (expr) (match expr
					(cons (symbol aggregate) _) true
					(cons '(quote aggregate) _) true
					(cons sym args) (reduce args (lambda (a b) (or a (_has_agg_expr b))) false)
					false)))
				(define _cond_parts (_flatten_and_parts condition))
				(define _cond_agg_parts (filter _cond_parts _has_agg_expr))
				(define _cond_non_agg (filter _cond_parts (lambda (p) (not (_has_agg_expr p)))))
				(define _grp_refs_src_tbl (lambda (expr)
					(reduce (extract_tblvars expr) (lambda (acc tv) (or acc (equal?? tv tblvar))) false)))
				(define ags (if materialized_source
					(merge_unique ags (merge (extract_assoc fields (lambda (key expr) (extract_aggregates expr)))))
					ags))
				(define _grp_has_explicit_outer (lambda (expr) (match expr
					(cons sym args) (if (or (equal? sym (quote outer)) (equal? sym '(quote outer)))
						true
						(reduce args (lambda (acc arg) (or acc (_grp_has_explicit_outer arg))) false))
					false)))
				/* scoped GROUPs must not keep domain/key correlation equalities in the
				aggregate compute filter. Those equalities are represented by the keytable
				group key plus LEFT JOIN ON-clause; leaving them here leaks outer refs
				into the cache formula and breaks skip-level COUNT reuse. Immediate
				correlations to the current outer row still stay in the compute path. */
				(define _grp_key_corr_part (lambda (part) (match part
					'((symbol equal??) left right) (if (reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr left))) false)
						(and (not (_grp_refs_src_tbl right)) (_grp_has_explicit_outer right))
						(if (reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr right))) false)
							(and (not (_grp_refs_src_tbl left)) (_grp_has_explicit_outer left))
							false))
					'((quote equal??) left right) (if (reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr left))) false)
						(and (not (_grp_refs_src_tbl right)) (_grp_has_explicit_outer right))
						(if (reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr right))) false)
							(and (not (_grp_refs_src_tbl left)) (_grp_has_explicit_outer left))
							false))
					false)))
				(define _cond_key_corr (if (nil? _stage_scope) '()
					(filter _cond_non_agg _grp_key_corr_part)))
				(define _cond_non_agg_effective (if (nil? _stage_scope) _cond_non_agg
					(filter _cond_non_agg (lambda (p) (not (_grp_key_corr_part p))))))
				/* non-aggregate condition = keytable scan filter */
				(set condition (if (equal? 0 (count _cond_non_agg_effective)) true
					(if (equal? 1 (count _cond_non_agg_effective)) (car _cond_non_agg_effective)
						(cons (quote and) _cond_non_agg_effective))))
				/* split non-aggregate condition: parts referencing partition-staged tables go to grouped_plan */
				(define _grp_cond_split (split_condition condition _grp_ps_tables))
				(define _grp_ps_condition (match _grp_cond_split '(_ later) later))
				(set condition (match _grp_cond_split '(now _) now))
				/* Scope contract: split_condition may conservatively classify a term as
				"later" because it traversed into nested runtime/scalar-subquery code and
				saw inner aliases there. If the resulting later-part no longer contains a
				real reference to one of the current partition-stage aliases at this plan
				level, it still belongs to the row-domain filter of the current group. */
				(define grp_ps_aliases (if (nil? _grp_ps_tables) '()
					(map _grp_ps_tables (lambda (td) (match td
						'(tv _ ttbl _ _)
						(if (nil? tv) ttbl tv)
						"")))))
				(define grp_ps_condition_refs_stage (lambda (expr) (match expr
					'((symbol get_column) alias_ _ _ _) (and (not (nil? alias_)) (has? grp_ps_aliases alias_))
					'((quote get_column) alias_ _ _ _) (and (not (nil? alias_)) (has? grp_ps_aliases alias_))
					(cons sym args) (if (_is_opaque_scope_sym sym)
						false
						(reduce args (lambda (found arg) (or found (grp_ps_condition_refs_stage arg))) false))
					false)))
				(if (and (not (nil? _grp_ps_condition))
					(not (equal? _grp_ps_condition true))
					(not (grp_ps_condition_refs_stage _grp_ps_condition)))
					(begin
						(set condition (combine_and_terms (list condition _grp_ps_condition)))
						(set _grp_ps_condition true))
					nil)
				(define _grp_outer_aliases (if (nil? _grp_ps_tables) '()
					(map _grp_ps_tables (lambda (td) (match td
						'(tv _ ttbl _ _) (if (nil? tv) ttbl tv)
						"")))))
				(define _resolve_outer_group_field (lambda (expr) (match expr
					'((symbol get_column) alias_ ti col ci) (if (and (not (nil? alias_)) (has? _grp_outer_aliases alias_))
						expr
						(if (nil? alias_)
							(begin
								(define matches (filter _grp_ps_tables (lambda (td) (match td
									'(tv _ ttbl _ _)
									(begin
										(define lookup_alias (if (nil? tv) ttbl tv))
										(reduce (coalesceNil (schemas lookup_alias) '()) (lambda (found coldef)
											(or found ((if ci equal?? equal?) (coldef "Field") col))) false))
									false))))
								(if (equal? 1 (count matches))
									(match (car matches)
										'(tv _ ttbl _ _)
										(list (quote get_column) (if (nil? tv) ttbl tv) false col false)
										nil)
									nil))
							nil))
					'((quote get_column) alias_ ti col ci) (if (and (not (nil? alias_)) (has? _grp_outer_aliases alias_))
						expr
						(if (nil? alias_)
							(begin
								(define matches (filter _grp_ps_tables (lambda (td) (match td
									'(tv _ ttbl _ _)
									(begin
										(define lookup_alias (if (nil? tv) ttbl tv))
										(reduce (coalesceNil (schemas lookup_alias) '()) (lambda (found coldef)
											(or found ((if ci equal?? equal?) (coldef "Field") col))) false))
									false))))
								(if (equal? 1 (count matches))
									(match (car matches)
										'(tv _ ttbl _ _)
										(list (quote get_column) (if (nil? tv) ttbl tv) false col false)
										nil)
									nil))
							nil))
					nil)))
				(define collect_condition (if (and is_dedup materialized_source)
					(begin
						(define _dedup_collect_parts (filter (_flatten_and_parts condition) (lambda (part)
							(and (equal? (has_only_tblvar_refs part tblvar) true)
								(not (_has_agg_expr part))))))
						(if (equal? 0 (count _dedup_collect_parts)) true
							(if (equal? 1 (count _dedup_collect_parts)) (car _dedup_collect_parts)
								(cons (quote and) _dedup_collect_parts))))
					condition))
				(define _stage_join_outer_expr (lambda (expr) (match expr
					(cons sym args) (if (or (equal? sym (quote outer)) (equal? sym '(quote outer)))
						expr
						(cons sym (map args _stage_join_outer_expr)))
					_ (begin
						(define _parts (split (string expr) "."))
						(match _parts
							(list _tbl _col) (list (quote get_column) _tbl false _col false)
							_ expr)))))
				(define _dedup_join_term (lambda (part) (match part
					'((symbol equal??) left right) (if (reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr left))) false)
						(if (_grp_refs_src_tbl right) nil
							(list (quote equal??) (replace_col_for_dedup left) (_stage_join_outer_expr right)))
						(if (reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr right))) false)
							(if (_grp_refs_src_tbl left) nil
								(list (quote equal??) (_stage_join_outer_expr left) (replace_col_for_dedup right)))
							nil))
					'((quote equal??) left right) (if (reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr left))) false)
						(if (_grp_refs_src_tbl right) nil
							(list (quote equal??) (replace_col_for_dedup left) (_stage_join_outer_expr right)))
						(if (reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr right))) false)
							(if (_grp_refs_src_tbl left) nil
								(list (quote equal??) (_stage_join_outer_expr left) (replace_col_for_dedup right)))
							nil))
					nil)))
				(set filtercols (merge_unique (list
					(extract_columns_for_tblvar tblvar collect_condition)
					(extract_outer_columns_for_tblvar tblvar collect_condition))))
				(define session_sensitive_group_domain (expr_uses_session_state collect_condition))
				(define kt_result (make_keytable schema tbl resolved_stage_group tblvar
					(if (or is_dedup session_sensitive_group_domain) collect_condition nil)))
				(set grouptbl (car kt_result))
				(define keytable_init (car (cdr kt_result)))
				(define fk_pk_col (car (cdr (cdr kt_result))))
				(define is_fk_reuse (not (nil? fk_pk_col)))

				/* make_collect: builds collect plan with optional WHERE filter
				with_filter=true: apply WHERE condition (for DEDUP)
				with_filter=false: collect ALL group keys (for NORMAL) */
				(define make_collect (lambda (with_filter)
					'('time '('begin
						/* If grouping is global (group='(1)), avoid base scan and insert one key row */
						(if (equal? resolved_stage_group '(1))
							'('insert schema grouptbl '(list "1") '(list '(list 1)) '(list) '('lambda '() true) true)
							(begin
								/* key columns */
								(set keycols (merge_unique (map resolved_stage_group (lambda (expr) (extract_columns_for_tblvar tblvar expr)))))
								(scan_wrapper 'scan schema tbl
									(if with_filter (cons list filtercols) '(list))
									(if with_filter
										'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr collect_condition)))
										'((quote lambda) '() true))
									(cons list keycols)
									'((quote lambda)
										(map keycols (lambda (col) (symbol (concat tblvar "." col))))
										(cons (quote list) (map resolved_stage_group (lambda (expr) (replace_columns_from_expr expr))))) /* build records '(k1 k2 ...) */
									'((quote lambda) '('acc 'rowvals) '('set_assoc 'acc 'rowvals true)) /* add keys to assoc; each key is a dataset -> unique filtering */
									'(list) /* empty dict */
									'((quote lambda) '('acc 'sharddict)
										'('insert
											schema grouptbl
											(cons 'list (map resolved_stage_group expr_name))
											'('assoc_keys_as_dataset_rows 'sharddict (count resolved_stage_group)) /* turn keys from assoc into dataset rows */
											'(list) '('lambda '() true) true)
									)
									isOuter)
							)
						)
					) "collect")))

				(if is_dedup (begin
					/* DEDUP-ONLY stage: no aggregate computation, just collect unique keys and pass through to next stage */
					(define replace_col_for_dedup (make_col_replacer grouptbl collect_condition true expr_name tblvar agg_col_name))
					(define dedup_schema_def (map resolved_stage_group (lambda (expr)
						(list "Field" (if is_fk_reuse fk_pk_col (expr_name expr)) "Type" "any"))))
					(planned_materialized_fields grouptbl dedup_schema_def)
					/* transform rest_groups to reference grouptbl columns instead of source table columns;
					first resolve nil -> tblvar via replace_find_column, then map tblvar -> grouptbl */
					(define _dedup_refs_current (lambda (e)
						(reduce (extract_tblvars e) (lambda (acc tv) (or acc (equal?? tv tblvar))) false)))
					(define _dedup_resolve (lambda (e)
						(if (_dedup_refs_current e)
							(replace_col_for_dedup (replace_find_column e))
							(coalesce (_resolve_outer_group_field e) e))))
					(define _dedup_kt_is_outer (not (nil? _stage_scope)))
					(define _dedup_kt_je (if _dedup_kt_is_outer
						(begin
							(define _dedup_terms (filter (map _cond_key_corr _dedup_join_term) (lambda (x) (not (nil? x)))))
							(if (equal? _dedup_terms '()) nil
								(if (equal? 1 (count _dedup_terms)) (car _dedup_terms)
									(cons (quote and) _dedup_terms))))
						nil))
					(define transformed_rest_groups (map rest_groups (lambda (s)
						(stage_preserve_cache_meta s (make_group_stage_with_condition
							(map (stage_group_cols s) _dedup_resolve)
							(_dedup_resolve (stage_having_expr s))
							(map (coalesce (stage_order_list s) '()) (lambda (o) (match o '(col dir) (list (_dedup_resolve col) dir))))
							(stage_limit_val s)
							(stage_offset_val s)
							(stage_partition_aliases s)
							(stage_init_code s)
							(begin (define sc (stage_condition s)) (if (nil? sc) nil (_dedup_resolve sc)))))
					)))
					(define grouped_plan (build_queryplan schema
						(if _dedup_kt_is_outer
							(merge _grp_ps_tables (list (list grouptbl schema grouptbl true _dedup_kt_je)))
							(list (list grouptbl schema grouptbl false nil)))
						(map_assoc fields (lambda (k v) (_dedup_resolve v)))
						nil /* condition already applied in collect */
						transformed_rest_groups
						schemas
						replace_find_column
						update_target))
					(cons 'begin (merge
						(if (nil? keytable_init) '() (list keytable_init))
						(if (nil? runtime_local_compute_plan) '() (list runtime_local_compute_plan))
						(list (make_collect true))
						(list grouped_plan)))
				) (begin
						/* NORMAL group stage: extract aggregates, compute, and continue.
						replace_agg_with_fetch rewrites (aggregate expr + 0) -> (get_column grouptbl "expr|cond")
						so ORDER BY SUM(amount) becomes ORDER BY on a keytable column. */
						(define condition_hash (fnv_hash (concat (expr_name condition) (runtime_cache_suffix_from_exprs (list condition)))))
						(define canonical_count_col_name (lambda ()
							(concat "COUNT(*)|" condition_hash)))
						(define agg_col_name (lambda (ag)
							(if (equal? ag count_ag)
								(canonical_count_col_name)
								(fnv_hash (concat (expr_name ag) "|" (expr_name condition) (runtime_cache_suffix_from_exprs (list ag condition)))))))
						(define replace_agg_with_fetch (make_col_replacer grouptbl condition false expr_name tblvar agg_col_name))
						(define replace_group_key_or_fetch (lambda (expr) (if
							(reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr expr))) false)
							'('get_column grouptbl false (if is_fk_reuse fk_pk_col (expr_name expr)) false)
							(replace_agg_with_fetch expr)
						)))
						/* scoped GROUP stages from unnesting must not eagerly rewrite later
						outer aggregates like COUNT(*) in the SELECT list. Those belong to
						subsequent global group stages and carry no refs to the current
						scoped source table. */
						(define replace_group_field_expr (lambda (expr)
							/* Design contract: scoped GROUP stages may only lower current-stage
							keys/aggregates here. Outer-pass-through expressions stay logical
							and are resolved later by the recursive grouped_plan build, not by
							synthesizing ad-hoc keytable aggregates in this stage. */
							(if (and (not (nil? _stage_scope)) (_refs_only_outer_stage expr))
								expr
								(match expr
									'((symbol get_column) _ _ _ _) (if (_field_needs_group_value_agg expr)
										(replace_group_key_or_fetch (_group_value_ag_expr (group_value_local_expr expr)))
										(replace_group_key_or_fetch (rewrite_materialized_source_cols_single expr)))
									'((quote get_column) _ _ _ _) (if (_field_needs_group_value_agg expr)
										(replace_group_key_or_fetch (_group_value_ag_expr (group_value_local_expr expr)))
										(replace_group_key_or_fetch (rewrite_materialized_source_cols_single expr)))
									(cons (symbol aggregate) agg_rest)
									/* scope-tagged aggregate from another stage: pass through */
									(if (and (not (nil? _stage_scope)) (>= (count agg_rest) 4)
										(not (has? _stage_scope (nth agg_rest 3))))
										expr
									(begin (define agg_rest_stripped (match agg_rest '(a b c _s) (list a b c) agg_rest))
									(if (or (and (not (nil? _stage_scope)) _has_later_group_stage (equal? (extract_tblvars expr) '()) (not (equal? agg_rest_stripped count_ag)))
										(and (not materialized_source) (_field_agg_has_nested_agg agg_rest_stripped) (equal? (extract_tblvars expr) '())))
										(match agg_rest_stripped
											'(agg_expr agg_reduce agg_neutral)
											(list (quote aggregate) (replace_group_field_expr agg_expr) agg_reduce agg_neutral)
											_ expr)
										(replace_group_key_or_fetch expr))))
									(cons '(quote aggregate) agg_rest)
									/* scope-tagged aggregate from another stage: pass through */
									(if (and (not (nil? _stage_scope)) (>= (count agg_rest) 4)
										(not (has? _stage_scope (nth agg_rest 3))))
										expr
									(begin (define agg_rest_stripped (match agg_rest '(a b c _s) (list a b c) agg_rest))
									(if (or (and (not (nil? _stage_scope)) _has_later_group_stage (equal? (extract_tblvars expr) '()) (not (equal? agg_rest_stripped count_ag)))
										(and (not materialized_source) (_field_agg_has_nested_agg agg_rest_stripped) (equal? (extract_tblvars expr) '())))
										(match agg_rest_stripped
											'(agg_expr agg_reduce agg_neutral)
											(list (quote aggregate) (replace_group_field_expr agg_expr) agg_reduce agg_neutral)
											_ expr)
										(replace_group_key_or_fetch expr))))
									(cons sym args)
									(if (_field_needs_group_value_agg expr)
										(replace_group_key_or_fetch (_group_value_ag_expr (group_value_local_expr expr)))
										(if (_matches_group_expr expr)
											(replace_group_key_or_fetch expr)
											(if (_is_opaque_scope_sym sym)
												(replace_group_key_or_fetch expr)
												(cons sym (map args replace_group_field_expr)))))
									(replace_group_key_or_fetch expr)
						))))
						/* normalize outer-side join expressions into column AST so scan
						planning can request the needed outer columns even if they are not
						part of the current projection/order list. */
						(define _grp_join_outer_expr (lambda (expr) (match expr
							(cons sym args) (if (or (equal? sym (quote outer)) (equal? sym '(quote outer)))
								expr
								(cons sym (map args _grp_join_outer_expr)))
							_ (begin
								(define _parts (split (string expr) "."))
								(match _parts
									(list _tbl _col) (list (quote get_column) _tbl false _col false)
									_ expr)))))
						(define _grp_join_term (lambda (part) (match part
							'((symbol equal??) left right) (if (reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr left))) false)
								(if (_grp_refs_src_tbl right) nil
									(list (quote equal??) (replace_group_key_or_fetch left) (_grp_join_outer_expr right)))
								(if (reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr right))) false)
									(if (_grp_refs_src_tbl left) nil
										(list (quote equal??) (_grp_join_outer_expr left) (replace_group_key_or_fetch right)))
									nil))
							'((quote equal??) left right) (if (reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr left))) false)
								(if (_grp_refs_src_tbl right) nil
									(list (quote equal??) (replace_group_key_or_fetch left) (_grp_join_outer_expr right)))
								(if (reduce resolved_stage_group (lambda (acc group_expr) (or acc (equal? group_expr right))) false)
									(if (_grp_refs_src_tbl left) nil
										(list (quote equal??) (_grp_join_outer_expr left) (replace_group_key_or_fetch right)))
									nil))
							nil)))
						/* scoped GROUPs join the keytable back via LEFT JOIN. Any correlation
						equality already emitted into that ON-clause must not survive as an
						additional post-group filter, otherwise NOT IN / NOT EXISTS lose the
						empty-match row (NULL keytable side fails the redundant equality). */
						/* Design contract: every scoped GROUP stage re-attaches its keytable to
						the preserved outer row stream, even for global `(1)` groups, but only
						when such a real outer stream actually exists. The synthetic no-FROM
						DUAL `.(1)` row is just a planner helper; treating it as a preserved
						outer stream here would force unnecessary LEFT JOIN semantics and can
						drop top-level NOT IN / EXISTS filters. */
						(define _real_outer_ps_tables (filter _grp_ps_tables (lambda (td) (match td
							'(_ _ ttbl _ _) (not (equal? ttbl ".(1)"))
							true))))
						(define _kt_is_outer (and (not (nil? _stage_scope)) (not (equal? _real_outer_ps_tables '()))))
						(define _kt_terms (if _kt_is_outer
							(filter (map _cond_non_agg _grp_join_term) (lambda (x) (not (nil? x))))
							'()))
						(define _strip_kt_term (lambda (part)
							(not (reduce _kt_terms (lambda (acc kt_part) (or acc (equal? part kt_part))) false))))
						(define _flatten_gp_part (lambda (expr) (match expr
							(cons sym parts) (if (or (equal? sym (quote and)) (equal? sym '(quote and)) (equal? sym 'and))
								(merge (map parts _flatten_gp_part))
								(list expr))
							(list expr))))

						(define grouped_order (if (nil? stage_order) nil (map stage_order (lambda (o) (match o '(col dir) (list (replace_group_key_or_fetch col) dir))))))
						(define next_groups (merge
							(if (coalesce grouped_order stage_limit stage_offset) (list (make_group_stage nil nil grouped_order stage_limit stage_offset nil nil)) '())
							(if _needs_synthetic_outer_group (list (make_group_stage '(1) nil nil nil nil nil nil)) '())
							rest_groups
						))
						/* FK reuse: extract child FK column name */
						(define fk_child_col (if is_fk_reuse
							(match (car resolved_stage_group) '('get_column _ false scol false) scol)
							nil))
						/* COUNT payload is needed whenever later logic fetches an aggregate-backed
						existence/anti result from this stage. Global `(1)` groups still need it
						if the deferred post-group condition contains aggregate terms. Only the
						COUNT>0 empty-group filter itself stays restricted to true global groups. */
						(define _marker_in_fields (reduce_assoc fields (lambda (acc _k v) (or acc (_has_agg_expr v))) false))
						(define _marker_in_having (if (nil? stage_having) false (_has_agg_expr stage_having)))
						(define _marker_in_order (reduce (coalesce stage_order '()) (lambda (acc o) (or acc (match o '(col _dir) (_has_agg_expr col) false))) false))
						(define needs_count (or
							(not (equal? resolved_stage_group '(1)))
							(not (equal? _cond_agg_parts '()))
							_marker_in_fields
							_marker_in_having
							_marker_in_order))
						/* SQL GROUP BY semantics: unscoped non-global groups only exist for row
						keys that survive the pre-group row domain. Keep the logical aggregate
						sentinels until build_scan, but enforce this domain invariant here via
						COUNT(*) > 0 instead of materializing helper-side phantom groups.
						Global helper stages still use the narrower suppression rule so
						user-visible SELECT COUNT(*) FROM ... on empty input keeps its single
						neutral row. Scoped GROUP stages must not suppress empty matches here,
						because NOT EXISTS / NOT IN rely on the later LEFT JOIN + coalesceNil. */
						(define filter_empty_groups (and
							(nil? _stage_scope)
							(or
								(not (equal? resolved_stage_group '(1)))
								(and
									(not _marker_in_fields)
									(not _marker_in_having)
									(not _marker_in_order)))))
						(define ags (if needs_count (merge_unique ags (list count_ag)) ags))
						(define count_col_name (if needs_count (agg_col_name count_ag) nil))
						(define keytable_schema_def (merge
							(map resolved_stage_group (lambda (expr)
								(list "Field" (if is_fk_reuse fk_pk_col (expr_name expr)) "Type" "any")))
							(map ags (lambda (ag)
								(list "Field" (agg_col_name ag) "Type" "any")))))
						(planned_materialized_fields grouptbl keytable_schema_def)
						/* AND count>0 into HAVING so empty/non-matching groups are excluded */
						(define effective_having (if (and needs_count filter_empty_groups)
							(begin
								(define count_check '('> '('get_column grouptbl false count_col_name false) 0))
								(define replaced_having (replace_group_key_or_fetch stage_having))
								(if (or (nil? replaced_having) (equal? replaced_having true))
									count_check
									(list 'and replaced_having count_check)))
							(replace_group_key_or_fetch stage_having)))

						/* Phase 2: replace aggregates in the separated agg-condition parts,
						then combine everything: HAVING + replaced agg-parts + ps-table conditions */
						(define _replaced_agg_parts (map _cond_agg_parts replace_group_field_expr))
						/* partition-staged table predicates stay global filters.
						The keytable LEFT JOIN must only use correlations against group/domain
						keys, otherwise unrelated outer filters get attached to the wrong side. */
						(define _gp_parts (filter (merge (map (merge
							(if (or (nil? effective_having) (equal? effective_having true)) '() (list effective_having))
							_replaced_agg_parts
							(if (equal? _grp_ps_condition true) '() (list (replace_group_key_or_fetch _grp_ps_condition))))
							_flatten_gp_part))
							(lambda (x) (and (not (nil? x)) (not (equal? x true)) (_strip_kt_term x)))))
						(define _gp_condition (if (equal? 0 (count _gp_parts)) nil
							(if (equal? 1 (count _gp_parts)) (car _gp_parts)
								(cons (quote and) _gp_parts))))
						/* drop partition-stages covered by this scoped GROUP: the keytable
						guarantees 1 row per group key, making the partition LIMIT redundant */
						(define _remaining_pstages (filter partition_stages (lambda (ps)
							(not (reduce (coalesceNil (stage_partition_aliases ps) '()) (lambda (acc a)
								(or acc (has? (coalesceNil _stage_scope '()) a))) false)))))
						/* scoped GROUPs: outer tables come FIRST, keytable is LEFT JOINed
						AFTER them. This ensures outer rows without keytable matches still
						appear (with NULL aggregates → coalesceNil → 0).
						Essential for NOT EXISTS / NOT IN semantics. */
						/* keytable join condition: only keep equalities that bind a group/domain
						key to an outer expression. Filters on outer/prejoin tables stay in the
						global condition; they are not ON-conditions of the keytable join. */
						(define _kt_je (if _kt_is_outer
							(begin
								(if (equal? _kt_terms '()) nil
									(if (equal? 1 (count _kt_terms)) (car _kt_terms)
										(cons (quote and) _kt_terms))))
							nil))
						(define grouped_plan (build_queryplan schema
							(if _kt_is_outer
								(merge _grp_ps_tables (list (list grouptbl schema grouptbl true _kt_je)))
								(list (list grouptbl schema grouptbl false nil)))
							(map_assoc fields (lambda (k v) (replace_group_field_expr v)))
							_gp_condition
							(merge next_groups _remaining_pstages)
							schemas
							replace_find_column
							update_target))
						/* Software contract for filtered grouped aggregates:
						- keep a single canonical aggregate temp column per helper table
						- drive eager materialization via filtercols/filter on the canonical
						COUNT helper column, not via ad-hoc one-shot aggregate scans
						- this keeps the aggregate cache persistent and incrementally
						maintainable while still skipping empty groups */
						/* createcolumn options: filter by COUNT column so only groups with rows are computed */
						(define createcol_options (cons 'list (merge '("temp" true)
							(if (and needs_count filter_empty_groups)
								(list "filtercols" (list 'list count_col_name)
									"filter" '((quote lambda) (list (symbol count_col_name)) '('> (symbol count_col_name) 0)))
								'()))))

						(define match_runtime_materialized_agg_col (lambda (target_col agg_name)
							(if materialized_source
								(begin
									(define materialized_cols (materialized_source_physical_schema schema tbl tblvar schemas))
									(coalesce
										(reduce materialized_cols (lambda (found coldef)
											(if (not (nil? found))
												found
												(begin
													(define field_name (coldef "Field"))
													(if (equal? field_name target_col) field_name nil))))
											nil)
										(reduce materialized_cols (lambda (found coldef)
											(if (not (nil? found))
												found
												(begin
													(define field_name (coldef "Field"))
													(if (and (>= (strlen field_name) (+ (strlen agg_name) 1))
														(equal? (substr field_name 0 (strlen agg_name)) agg_name)
														(equal? (substr field_name (strlen agg_name) 1) "|"))
														field_name
														nil))))
											nil)
										(reduce materialized_cols (lambda (found coldef)
											(if (not (nil? found))
												found
												(begin
													(define source_expr (coalesceNil (coldef "Expr") nil))
													(match source_expr
														'((symbol get_column) _ _ source_col _)
														(if (or (equal? source_col target_col)
															(and (>= (strlen source_col) (+ (strlen agg_name) 1))
																(equal? (substr source_col 0 (strlen agg_name)) agg_name)
																(equal? (substr source_col (strlen agg_name) 1) "|")))
															(coldef "Field")
															nil)
														'((quote get_column) _ _ source_col _)
														(if (or (equal? source_col target_col)
															(and (>= (strlen source_col) (+ (strlen agg_name) 1))
																(equal? (substr source_col 0 (strlen agg_name)) agg_name)
																(equal? (substr source_col (strlen agg_name) 1) "|")))
															(coldef "Field")
															nil)
														nil))))
											nil)))
								nil)))
						(define lower_runtime_materialized_aggs_single (lambda (expr) (match expr
							(cons (symbol aggregate) agg_args) (begin
								(define target_col (agg_col_name agg_args))
								(define agg_name (canonical_expr_name (normalize_canonical_aliases agg_args) '(list) '(list) canon_alias_map))
								(define visible_expr (if materialized_source
									(lower_visible_materialized_aggs_single expr)
									expr))
								(define match_col (match_runtime_materialized_agg_col target_col agg_name))
								(if (not (equal? visible_expr expr))
									visible_expr
									(if (nil? match_col)
										(match agg_args
											'(agg_expr agg_reduce agg_neutral)
											(list (quote aggregate) (lower_runtime_materialized_aggs_single agg_expr) agg_reduce agg_neutral)
											_ expr)
										(list (quote get_column) tblvar false match_col false))))
							(cons '(quote aggregate) agg_args) (begin
								(define target_col (agg_col_name agg_args))
								(define agg_name (canonical_expr_name (normalize_canonical_aliases agg_args) '(list) '(list) canon_alias_map))
								(define visible_expr (if materialized_source
									(lower_visible_materialized_aggs_single expr)
									expr))
								(define match_col (match_runtime_materialized_agg_col target_col agg_name))
								(if (not (equal? visible_expr expr))
									visible_expr
									(if (nil? match_col)
										(match agg_args
											'(agg_expr agg_reduce agg_neutral)
											(list (quote aggregate) (lower_runtime_materialized_aggs_single agg_expr) agg_reduce agg_neutral)
											_ expr)
										(list (quote get_column) tblvar false match_col false))))
							(cons sym args) (cons sym (map args lower_runtime_materialized_aggs_single))
							expr)))
						(define agg_plans (map ags (lambda (ag) (match ag '(expr reduce neutral) (begin
							(define runtime_expr
								(rewrite_materialized_source_cols_single
									(rewrite_materialized_source_aggs_single
										(lower_runtime_materialized_aggs_single expr))))
							(set cols (merge_unique (list
								(extract_columns_for_tblvar tblvar runtime_expr)
								(extract_outer_columns_for_tblvar tblvar runtime_expr)
							)))
							/* COUNT column itself must not filter by itself (circular); all others filter by COUNT>0 */
							(define this_options (if (and needs_count (equal? (agg_col_name ag) count_col_name)) '(list "temp" true) createcol_options))
							'((quote createcolumn) schema grouptbl (agg_col_name ag) "any" '(list) this_options
								(cons list (map resolved_stage_group (lambda (col) (if is_fk_reuse fk_pk_col (expr_name col)))))
								'((quote lambda) (map resolved_stage_group (lambda (col) (symbol (if is_fk_reuse fk_pk_col (expr_name col)))))
									(scan_wrapper 'scan schema tbl
										(cons list (merge tblvar_cols filtercols))
										/* check group equality AND WHERE-condition */
										'((quote lambda) (map (merge tblvar_cols filtercols) (lambda (col) (symbol (concat tblvar "." col)))) (optimize (cons (quote and) (cons (replace_columns_from_expr condition) (map resolved_stage_group (lambda (col) '((quote equal?) (replace_columns_from_expr col) '((quote outer) (symbol (if is_fk_reuse fk_pk_col (expr_name col)))))))))))
										(cons list cols)
										'((quote lambda) (map cols (lambda(col) (symbol (concat tblvar "." col)))) (replace_columns_from_expr runtime_expr))
										reduce
										neutral
										nil
										false /* never isOuter in createcolumn: COUNT=0 for empty matches, not COUNT=1 */
									)
							))
						)))))
						/* COUNT is a dependency for filtered aggregates: non-count keytable
						columns may filter on COUNT>0 and therefore must not race the COUNT
						createcolumn itself. Keep COUNT synchronous, then parallelize the
						remaining independent aggregate columns. */
						(define agg_plan_indices (produceN (count ags)))
						(define count_plan (if needs_count
							(reduce agg_plan_indices (lambda (found i)
								(if (not (nil? found))
									found
									(if (equal? (agg_col_name (nth ags i)) count_col_name)
										(nth agg_plans i)
										nil)))
								nil)
							nil))
						(define non_count_agg_plans (reduce agg_plan_indices (lambda (acc i)
							(if (and needs_count (equal? (agg_col_name (nth ags i)) count_col_name))
								acc
								(merge acc (list (nth agg_plans i)))))
							'()))
						(define compute_plan
							(if (nil? count_plan)
								'('time (cons 'parallel agg_plans) "compute")
								(if (equal? non_count_agg_plans '())
									(list 'time count_plan "compute")
									(list 'begin
										(list 'time count_plan "compute-count")
										(list 'time (cons 'parallel non_count_agg_plans) "compute")))))

						/* invalidation is handled by registerComputeTriggers in ComputeColumn:
						DML triggers on the base table invalidate computed columns automatically.
						No forced invalidation needed here — the createcolumn/ComputeColumn path
						skips recompute when the proxy is still valid (no DML since last compute). */
						(define invalidation_plan nil)

						/* build key column pairs for keytable cleanup triggers: ((base_col kt_col) ...) */
						(define key_pairs (map resolved_stage_group (lambda (expr)
							(match expr
								'((symbol get_column) _ _ col _) (list col (expr_name expr))
								'((quote get_column) _ _ col _) (list col (expr_name expr))
								(list (expr_name expr) (expr_name expr))
						))))
						(define cleanup_plan (if (or is_fk_reuse (equal? resolved_stage_group '(1))) nil
							(list 'register_keytable_cleanup schema tbl schema grouptbl tblvar
								(cons 'list (map key_pairs (lambda (p) (list 'list (car p) (cadr p))))))))
						(define collect_plan (if is_fk_reuse '()
							(if session_sensitive_group_domain
								/* session-sensitive grouped domains must never reuse a stale
								keytable domain collected under a different session binding.
								Until the keytable identity fully carries those bindings end-to-end,
								rebuild the key domain explicitly for every execution. */
								(list (list 'begin
									(list 'droptable schema grouptbl true)
									keytable_init
									(make_collect false)))
								(if (not (nil? _stage_scope))
									/* scoped GROUPs: always collect (keytable may have stale data from prior queries) */
									(list (make_collect false))
									(list (list 'if (list 'or keytable_init (list 'table_empty? schema grouptbl))
										(make_collect false)
										nil))))))
						(cons 'begin (merge
							(if (nil? keytable_init) '() (list keytable_init))
							(if (nil? runtime_local_compute_plan) '() (list runtime_local_compute_plan))
							(if (nil? group_value_local_compute_plan) '() (list group_value_local_compute_plan))
							(if (nil? cleanup_plan) '() (list cleanup_plan))
							collect_plan
							(if (nil? invalidation_plan) '() (list invalidation_plan))
							(list compute_plan)
							(list
								/* window+GROUP BY injection: after keytable is computed,
								scan it to fill promises with global totals, then wrap
								grouped_plan's resultrow to inject promise values. */
								(if (nil? (_wg_store "fields")) grouped_plan
									(begin
										(define _wg_ctr (newsession)) (_wg_ctr "n" 0)
										(define _wg_nn (lambda () (begin (_wg_ctr "n" (+ (_wg_ctr "n") 1)) (concat "__wgp_" (_wg_ctr "n")))))
										/* Fields with nested window functions must be rebuilt from the
										grouped row plus global window promises. Pure non-window fields can
										still pass straight through as grouped row lookups. */
										(define _wg_pl (newsession)) (_wg_pl "l" '())
										(define _wg_row_fields (newsession))
										(map_assoc fields (lambda (fk fv)
											(begin
												(map (materialized_source_expr_keys fv) (lambda (key)
													(_wg_row_fields key fk)))
												nil)))
										(define _wg_find_row_field (lambda (expr)
											(reduce (materialized_source_expr_keys expr) (lambda (found key)
												(if (not (nil? found))
													found
													(coalesce (_wg_row_fields key) nil)))
												nil)))
										(define _wg_promises (newsession))
										(define _wg_window_value (lambda (expr) (begin
											(define existing (reduce (materialized_source_expr_keys expr) (lambda (found key)
												(if (not (nil? found))
													found
													(coalesce (_wg_promises key) nil)))
												nil))
											(if (not (nil? existing))
												(symbol existing)
												(begin
													(define pn (_wg_nn))
													(define wfn (nth expr 1))
													(define wargs (nth expr 2))
													(define inner_agg (if (and (list? wargs) (> (count wargs) 0)) (car wargs) 1))
													(define agg_tuple (match inner_agg (cons (symbol aggregate) rest) rest (list inner_agg (quote +) 0)))
													(define acn (agg_col_name agg_tuple))
													(map (materialized_source_expr_keys expr) (lambda (key)
														(_wg_promises key pn)))
													(_wg_pl "l" (cons (list pn acn wfn) (_wg_pl "l")))
													(symbol pn))))))
										(define _wg_emit_window_expr (lambda (expr) (begin
											(define has_nested_window (not (equal? (extract_window_funcs expr) '())))
											(if has_nested_window
												(match expr
													(cons (symbol window_func) _) (_wg_window_value expr)
													(cons (quote window_func) _) (_wg_window_value expr)
													(cons sym args) (cons sym (map args _wg_emit_window_expr))
													expr)
												(begin
													(define row_field (_wg_find_row_field expr))
													(if (not (nil? row_field))
														(list (quote get_assoc) (symbol "__wgr") row_field)
														(match expr
															(cons sym args) (cons sym (map args _wg_emit_window_expr))
															expr)))))))
										(define _wg_out_fields (map_assoc (_wg_store "fields") (lambda (k v)
											(if (equal? (extract_window_funcs v) '())
												(list (quote get_assoc) (symbol "__wgr") k)
												(_wg_emit_window_expr v)))))
										/* scan keytable for each promise: aggregate the column globally */
										(define _wg_scans (map (_wg_pl "l") (lambda (pi) (match pi '(pn acn wfn)
											(begin
												(define reduce_op (match wfn "SUM" (quote +) "COUNT" (quote +) "MIN" (quote min) "MAX" (quote max) (quote +)))
												(define neutral (match wfn "SUM" 0 "COUNT" 0 "MIN" nil "MAX" nil 0))
												(list (quote set) (symbol pn)
													(list (quote scan)
														'(session "__memcp_tx")
														schema
														(scan-runtime-source grouptbl)
														(list (quote list) acn)
														(list (quote lambda) (list (symbol acn)) true)
														(list (quote list) acn)
														(list (quote lambda) (list (symbol acn)) (symbol acn))
														reduce_op
														neutral
														nil
														false)))))))
										/* wrap grouped_plan: preserve field/value pairs so outer
										materialization and result serialization keep the visible column
										names attached to the rebuilt expressions. */
										(define _wg_rr_body (cons (quote list) (merge (extract_assoc _wg_out_fields (lambda (k v)
											(list k v))))))
										(cons 'begin (merge _wg_scans (list
											(list (quote set) (symbol "__wg_orig_rr") (symbol "resultrow"))
											(list (quote set) (symbol "resultrow")
												(list (quote lambda) (list (symbol "__wgr"))
													(list (symbol "__wg_orig_rr") _wg_rr_body)))
											grouped_plan))))))))
				))
			)
			(begin /* multi-table GROUP BY via prejoin materialization */
				/* Scoped groups only materialize the tables inside their domain. Outer
				tables stay outside so the recursive single-table GROUP path can keep the
				keytable LEFT-joined to the surrounding row stream. Global multi-table
				GROUPs still materialize all participating tables. */
				(define _grp_table_aliases (map _grp_tables (lambda (t) (match t '(tv _ _ _ _) tv ""))))
				(define _prejoin_local_joinexpr_part (lambda (part)
					(reduce (extract_tblvars part) (lambda (acc tv)
						(and acc (has? _grp_table_aliases tv)))
						true)))
				(define _split_prejoin_joinexpr (lambda (expr)
					(begin
						(define _parts (flatten_and_terms (coalesceNil expr true)))
						(list
							(combine_and_terms (filter _parts _prejoin_local_joinexpr_part))
							(combine_and_terms (filter _parts (lambda (part) (not (_prejoin_local_joinexpr_part part)))))))))
				(define _prejoin_joinexpr_split (reduce _grp_tables (lambda (acc td)
					(match acc '(tables_acc raw_deferred_acc deferred_acc)
						(match td '(tv tschema ttbl tisOuter tjoinexpr)
							(match (_split_prejoin_joinexpr tjoinexpr) '(local_joinexpr deferred_joinexpr)
								(list
									(merge tables_acc (list (list tv tschema ttbl tisOuter local_joinexpr)))
									(merge raw_deferred_acc (flatten_and_terms deferred_joinexpr))
									(merge deferred_acc (flatten_and_terms (replace_find_column deferred_joinexpr))))))))
					(list '() '() '())))
				(define prejoin_source_tables (car _prejoin_joinexpr_split))
				(define deferred_prejoin_joinexpr_parts_raw (cadr _prejoin_joinexpr_split))
				(define deferred_prejoin_joinexpr_parts (cadr (cdr _prejoin_joinexpr_split)))
				(define prejoin_alias_map (build_occurrence_alias_map prejoin_source_tables))
				(define _prejoin_alias_variants (lambda (tv tschema ttbl)
					(merge
						(list tv)
						(if (equal? (visible_occurrence_alias tv) tv) '() (list (visible_occurrence_alias tv)))
						(if (equal? (visible_occurrence_alias tv) ttbl) (list (concat tschema "." ttbl)) '()))))
				(define known_table_aliases (merge (map prejoin_source_tables (lambda (t) (match t
					'(tv tschema ttbl _ _) (_prejoin_alias_variants tv tschema ttbl)
					'())))))
				(define rewrite_materialized_source_aggs (lambda (expr nested_agg) (match expr
					(cons (symbol aggregate) agg_args)
					(if nested_agg
						(begin
							(define canonical_agg_args (canonicalize_expr agg_args prejoin_alias_map))
							(define agg_name (serialize_canonical_expr canonical_agg_args))
							(define match_col (reduce prejoin_source_tables (lambda (acc td)
								(if (not (nil? acc))
									acc
									(match td '(tv tschema ttbl _ _)
										(begin
											(define source_cols (materialized_source_physical_schema tschema ttbl tv schemas))
											(reduce source_cols (lambda (found coldef)
												(if (not (nil? found)) found
													(begin
														(define field_name (coldef "Field"))
														(if (and (>= (strlen field_name) (+ (strlen agg_name) 1))
															(equal? (substr field_name 0 (strlen agg_name)) agg_name)
															(equal? (substr field_name (strlen agg_name) 1) "|"))
															(list tv field_name)
															nil))))
												nil)))))
								nil))
							(if (nil? match_col)
								(match agg_args
									'(agg_expr agg_reduce agg_neutral)
									(list (quote aggregate) (rewrite_materialized_source_aggs agg_expr true) agg_reduce agg_neutral)
									_ expr)
								(list (quote get_column) (car match_col) false (cadr match_col) false)))
						(match agg_args
							'(agg_expr agg_reduce agg_neutral)
							(list (quote aggregate) (rewrite_materialized_source_aggs agg_expr true) agg_reduce agg_neutral)
							_ expr))
					(cons '(quote aggregate) agg_args)
					(if nested_agg
						(begin
							(define canonical_agg_args (canonicalize_expr agg_args prejoin_alias_map))
							(define agg_name (serialize_canonical_expr canonical_agg_args))
							(define match_col (reduce prejoin_source_tables (lambda (acc td)
								(if (not (nil? acc))
									acc
									(match td '(tv tschema ttbl _ _)
										(begin
											(define source_cols (materialized_source_physical_schema tschema ttbl tv schemas))
											(reduce source_cols (lambda (found coldef)
												(if (not (nil? found)) found
													(begin
														(define field_name (coldef "Field"))
														(if (and (>= (strlen field_name) (+ (strlen agg_name) 1))
															(equal? (substr field_name 0 (strlen agg_name)) agg_name)
															(equal? (substr field_name (strlen agg_name) 1) "|"))
															(list tv field_name)
															nil))))
												nil)))))
								nil))
							(if (nil? match_col)
								(match agg_args
									'(agg_expr agg_reduce agg_neutral)
									(list (quote aggregate) (rewrite_materialized_source_aggs agg_expr true) agg_reduce agg_neutral)
									_ expr)
								(list (quote get_column) (car match_col) false (cadr match_col) false)))
						(match agg_args
							'(agg_expr agg_reduce agg_neutral)
							(list (quote aggregate) (rewrite_materialized_source_aggs agg_expr true) agg_reduce agg_neutral)
							_ expr))
					(cons sym args) (cons sym (map args (lambda (arg) (rewrite_materialized_source_aggs arg nested_agg))))
					expr)))
				(define post_group_tables '())
				/* resolve condition and fields */
				(define raw_condition (coalesceNil condition true))
				(set condition (replace_find_column (coalesceNil condition true)))
				(define post_group_condition (combine_and_terms deferred_prejoin_joinexpr_parts))
				(define raw_post_group_condition (combine_and_terms deferred_prejoin_joinexpr_parts_raw))
				/* 2-phase: separate aggregate-containing parts from materialize condition.
				Aggregates belong in the grouped_plan (evaluated after GROUP BY keytable),
				not in the prejoin_materialize_plan (which just fills the prejoin table). */
				(define contains_aggregate (lambda (expr) (match expr
					(cons (symbol aggregate) _) true
					(cons '(quote aggregate) _) true
					(cons sym args) (reduce args (lambda (a b) (or a (contains_aggregate b))) false)
					false)))
				(define raw_condition_parts (match raw_condition
					(cons sym parts) (if (or (equal? sym (quote and)) (equal? sym '(quote and)))
						parts (list raw_condition))
					(list raw_condition)))
				(define condition_parts (match condition
					(cons sym parts) (if (or (equal? sym (quote and)) (equal? sym '(quote and)))
						parts (list condition))
					(list condition)))
				(define raw_aggregate_condition_parts (filter raw_condition_parts contains_aggregate))
				(define raw_materialize_condition_parts (filter raw_condition_parts (lambda (p) (not (contains_aggregate p)))))
				(define raw_materialize_condition (if (equal? 0 (count raw_materialize_condition_parts)) true
					(if (equal? 1 (count raw_materialize_condition_parts)) (car raw_materialize_condition_parts)
						(cons (quote and) raw_materialize_condition_parts))))
				(define aggregate_condition_parts (filter condition_parts contains_aggregate))
				(define materialize_condition_parts (filter condition_parts (lambda (p) (not (contains_aggregate p)))))
				(set condition (if (equal? 0 (count materialize_condition_parts)) true
					(if (equal? 1 (count materialize_condition_parts)) (car materialize_condition_parts)
						(cons (quote and) materialize_condition_parts))))
				/* Scoped prejoins must not consume outer-table predicates at materialize
				time. Keep local predicates on the prejoin source and defer the rest to the
				recursive grouped plan, where the keytable joins back to the outer row. */
				(define _grp_condition_split (split_condition condition _grp_ps_tables))
				(define condition (match _grp_condition_split '(prejoin_condition _) prejoin_condition))
				(define post_group_condition (match _grp_condition_split '(_ deferred_condition)
					(if (equal? deferred_condition true) post_group_condition
						(if (equal? post_group_condition true)
							deferred_condition
							(cons (quote and) (cons post_group_condition (list deferred_condition)))))))
				(define _raw_grp_condition_split (split_condition raw_materialize_condition _grp_ps_tables))
				(define raw_condition (match _raw_grp_condition_split '(raw_prejoin_condition _) raw_prejoin_condition))
				(define raw_post_group_condition (match _raw_grp_condition_split '(_ raw_deferred_condition)
					(if (equal? raw_deferred_condition true) raw_post_group_condition
						(if (equal? raw_post_group_condition true)
							raw_deferred_condition
							(cons (quote and) (cons raw_post_group_condition (list raw_deferred_condition)))))))
				/* Only true deferred condition aggregates belong in grouped_plan condition.
				Local HAVING on the recursive grouped prejoin stage must stay on the stage
				itself instead of being collapsed into the later count-cache condition path. */
				(define post_group_condition (if (or (nil? _grp_ps_tables) (equal? 0 (count aggregate_condition_parts))) post_group_condition
					(if (equal? post_group_condition true)
						(if (equal? 1 (count aggregate_condition_parts)) (car aggregate_condition_parts) (cons (quote and) aggregate_condition_parts))
						(cons (quote and) (cons post_group_condition aggregate_condition_parts)))))
				(define raw_post_group_condition (if (or (nil? _grp_ps_tables) (equal? 0 (count raw_aggregate_condition_parts))) raw_post_group_condition
					(if (equal? raw_post_group_condition true)
						(if (equal? 1 (count raw_aggregate_condition_parts)) (car raw_aggregate_condition_parts) (cons (quote and) raw_aggregate_condition_parts))
						(cons (quote and) (cons raw_post_group_condition raw_aggregate_condition_parts)))))
				(define _outer_visible_aliases (if (nil? _grp_ps_tables) '()
					(map _grp_ps_tables (lambda (td) (match td
						'(tv _ ttbl _ _) (if (nil? tv) ttbl (visible_occurrence_alias tv))
						"")))))
				(define _keep_outer_field_early (lambda (expr)
					(if (nil? _stage_scope)
						nil
						(match expr
							'((symbol get_column) alias_ _ _ _) (if (has? _outer_visible_aliases alias_) expr nil)
							'((quote get_column) alias_ _ _ _) (if (has? _outer_visible_aliases alias_) expr nil)
							_ nil))))
				(define resolved_fields (map_assoc fields (lambda (k v)
					(rewrite_materialized_source_aggs
						(coalesce (_keep_outer_field_early v) (replace_find_column v))
						false))))
				/* extract all get_column refs from group, fields, having, order, AND condition
				(condition may reference _unn_ tables whose columns must be in the prejoin) */
				(define all_referenced_columns (merge
					(merge (map stage_group extract_all_get_columns))
					(merge (extract_assoc resolved_fields (lambda (k v) (extract_all_get_columns v))))
					(if (nil? stage_post_group_condition) '() (extract_all_get_columns stage_post_group_condition))
					(merge (map (coalesce stage_order '()) (lambda (o) (match o '(col dir) (extract_all_get_columns col)))))
					(extract_all_get_columns (coalesceNil raw_condition true))
					(extract_all_get_columns (coalesceNil raw_post_group_condition true))
					(merge (map tables (lambda (td) (match td
						'(_ _ _ _ je) (if (nil? je) '() (extract_all_get_columns je))
						'()))))
				))
				/* filter out columns from partition-staged tables (they're not part of the prejoin) */
				(define all_referenced_columns (filter all_referenced_columns (lambda (mc) (match mc
					'(name '((symbol get_column) alias_ _ _ _)) (has? known_table_aliases alias_)
					'(name '((quote get_column) alias_ _ _ _)) (has? known_table_aliases alias_)
					true))))
				/* compute prejoin table name and alias */
				(define prejoin_alias ".pj")
				(define lower_prejoin_lineage_expr (lambda (expr) (begin
					(define _lower_once (lambda (cur)
						(reduce prejoin_source_tables (lambda (inner td) (match td
							'(tv _ ttbl _ _)
							(if (materialized-source? ttbl)
								(lower_materialized_source_expr ttbl tv inner)
								inner)
							inner))
							cur)))
					(define expr2 (_lower_once expr))
					(define expr3 (_lower_once expr2))
					(define expr4 (_lower_once expr3))
					(if (equal? expr4 expr3)
						expr4
						expr4))))
				(define canonicalize_prejoin_source_expr (lambda (expr)
					(rewrite_source_aliases prejoin_alias_map
						(normalize_visible_aliases
							(lower_prejoin_lineage_expr expr)))))
				/* canonical prejoin key: source tables only (no alias), for maximal reuse across equivalent queries */
				(define prejoin_columns (reduce all_referenced_columns (lambda (acc mc)
					(begin
						(define canonical_lineage_expr (canonicalize_expr
							(normalize_canonical_aliases (canonicalize_prejoin_source_expr (cadr mc)))
							prejoin_alias_map))
						(define canon_name (serialize_canonical_expr canonical_lineage_expr))
						(if (reduce acc (lambda (found mc2) (or found (equal? (car mc2) canon_name))) false)
							acc
							(merge acc (list (list canon_name (cadr mc))))))) '()))
				(define prejoin_column_names (map prejoin_columns car))
				(define prejoin_col_names prejoin_column_names)
				(define prejoin_schema_def (map prejoin_columns (lambda (mc)
					(list "Field" (car mc) "Type" "any" "Expr" (cadr mc)))))
				(define prejoin_row_domain_raw (combine_and_terms
					(merge
						(if (or (nil? raw_condition) (equal? raw_condition true)) '()
							(list raw_condition))
						(merge (map prejoin_source_tables (lambda (td) (match td
							'(_ _ _ _ tjoinexpr)
								(if (or (nil? tjoinexpr) (equal? tjoinexpr true)) '()
									(list tjoinexpr))
							'())))))))
				(define prejoin_condition_name (serialize_canonical_expr
					(canonicalize_expr
						(normalize_canonical_aliases (canonicalize_prejoin_source_expr prejoin_row_domain_raw))
						prejoin_alias_map)))
				(define prejointbl (concat ".prejoin:"
					(map prejoin_source_tables (lambda (t) (match t '(_ tschema ttbl _ _) (concat tschema "." ttbl)))
					) ":" prejoin_col_names "|" prejoin_condition_name))
				/* capture outer schema and table name for trigger code generation */
				(define prejoin_schema schema)
				(define pj_schema schema) /* needed in quoted runtime code below */
				(define prejoin_table_name prejointbl)
				(define temp_source_table? materialized-source?)
				(define _td_alias_variants (lambda (tv tschema ttbl) (begin
					(define _raw_aliases (filter (list
						tv
						(match tv '(visible _) visible nil)
						(visible_occurrence_alias tv)
						(coalesce (resolve_source_alias prejoin_alias_map tv) nil)
						(if (equal? (visible_occurrence_alias tv) ttbl) (concat tschema "." ttbl) nil))
						(lambda (x) (not (nil? x)))))
					(reduce (merge _raw_aliases
						(merge (map _raw_aliases (lambda (alias_v)
							(if (string? alias_v) (list (sanitize_temp_name alias_v)) '())))))
						(lambda (acc alias_v)
							(if (or (nil? alias_v) (has? acc alias_v))
								acc
								(merge acc (list alias_v))))
						'()))))
				(define prejoin_variant_exprs (lambda (expr) (match expr
					'((symbol get_column) alias_ ti col ci) (merge
						(list expr
							(canonicalize_prejoin_source_expr expr)
							(rewrite_source_aliases prejoin_alias_map expr))
						(merge (map prejoin_source_tables (lambda (td) (match td '(tv tschema ttbl _ _)
							(if (has? (_td_alias_variants tv tschema ttbl) alias_)
								(map (_td_alias_variants tv tschema ttbl) (lambda (alias_v)
									(list (quote get_column) alias_v ti col ci)))
								'())
							'())))))
					'((quote get_column) alias_ ti col ci) (merge
						(list expr
							(canonicalize_prejoin_source_expr expr)
							(rewrite_source_aliases prejoin_alias_map expr))
						(merge (map prejoin_source_tables (lambda (td) (match td '(tv tschema ttbl _ _)
							(if (has? (_td_alias_variants tv tschema ttbl) alias_)
								(map (_td_alias_variants tv tschema ttbl) (lambda (alias_v)
									(list (quote get_column) alias_v ti col ci)))
								'())
							'())))))
					_ (list expr
						(canonicalize_prejoin_source_expr expr)
						(rewrite_source_aliases prejoin_alias_map expr)))))
				(define prejoin_variant_names (lambda (expr)
					(reduce (map (prejoin_variant_exprs expr) (lambda (variant_expr)
						(sanitize_temp_name
							(serialize_canonical_expr
								(canonicalize_expr
									(normalize_canonical_aliases (canonicalize_prejoin_source_expr variant_expr))
									prejoin_alias_map)))))
						(lambda (acc variant_name) (append_unique acc variant_name))
						'())))
				(prejoin_canonical_sources prejointbl
					(merge (map prejoin_columns (lambda (mc) (begin
						(define source_expr (canonicalize_prejoin_source_expr (cadr mc)))
						(map (reduce (cons (car mc) (prejoin_variant_names (cadr mc)))
							(lambda (acc variant_name) (append_unique acc variant_name))
							'())
							(lambda (variant_name) (list variant_name source_expr))))))))
				(materialized_source_expr_lookup prejointbl
					(merge (map prejoin_columns (lambda (mc) (begin
						(define variant_exprs (reduce (prejoin_variant_exprs (cadr mc))
							(lambda (acc variant_expr) (append_unique acc variant_expr))
							'()))
						(merge (map variant_exprs (lambda (variant_expr)
							(map (materialized_source_expr_keys variant_expr) (lambda (k) (list k (car mc))))))))))))
				(planned_materialized_fields prejointbl prejoin_schema_def)
				/* create prejoin table at build time (needed for recursive build_queryplan -> make_keytable) */
				(createtable schema prejointbl
					(map prejoin_column_names (lambda (col) '("column" col "any" '() '())))
					query_temp_table_options true)
				/* legacy nested-loop materializer retained temporarily for trigger backfill paths.
				Query-time prejoin filling uses the canonical build_queryplan row stream below. */
				(define build_materialize_scan (lambda (scan_tables scan_condition is_outermost)
					(match scan_tables
						(cons '(tblvar schema tbl isOuter joinexpr) rest) (begin
							/* columns needed from this table for materialization + condition */
							(set cols (merge_unique (list
								(extract_columns_for_tblvar tblvar scan_condition)
								(merge_unique (map prejoin_columns (lambda (mc) (extract_columns_for_tblvar tblvar (cadr mc)))))
								(extract_outer_columns_for_tblvar tblvar scan_condition)
								(merge_unique (map prejoin_columns (lambda (mc) (extract_outer_columns_for_tblvar tblvar (cadr mc)))))
								(extract_later_joinexpr_columns_for_tblvar tblvar rest)
							)))
							(match (split_scan_condition isOuter joinexpr scan_condition rest) '(now_condition later_condition) (begin
								(set filtercols (merge_unique (list
									(extract_columns_for_tblvar tblvar now_condition)
									(extract_outer_columns_for_tblvar tblvar now_condition))))
								(scan_wrapper 'scan schema tbl
									(cons list filtercols)
									'((quote lambda) (map filtercols (lambda (col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr now_condition)))
									(cons list cols)
									'((quote lambda) (map cols (lambda (col) (symbol (concat tblvar "." col)))) (build_materialize_scan rest later_condition false))
									/* reduce: merge sub-results */
									'('lambda '('acc 'sub) '('merge 'acc 'sub))
									'(list)
									/* reduce2: outermost inserts into prejoin, inner levels merge */
									(if is_outermost
										'('lambda '('acc 'shard_rows) '('insert prejoin_schema prejointbl (cons 'list prejoin_column_names) 'shard_rows '(list) '('lambda '() true) true))
										'('lambda '('acc 'shard_rows) '('merge 'acc 'shard_rows)))
									isOuter
								)
							))
						)
						'() /* base case: produce one row wrapped in a list */
						'('if (optimize (replace_columns_from_expr (coalesceNil scan_condition true)))
							(list (quote list) (cons (quote list) (map prejoin_columns (lambda (mc) (replace_columns_from_expr (cadr mc))))))
							'(list))
					)
				))
				(define covered_partition_stages (filter partition_stages (lambda (ps)
					(reduce (coalesceNil (stage_partition_aliases ps) '()) (lambda (acc a)
						(or acc (has? known_table_aliases a))) false))))
				(define prejoin_materialize_fields (merge (map prejoin_columns (lambda (mc) (list (car mc) (cadr mc))))))
				(define prejoin_materialize_rowplan (build_queryplan schema
					prejoin_source_tables
					prejoin_materialize_fields
					raw_condition
					covered_partition_stages
					schemas
					replace_find_column
					nil))
				(define prejoin_materialize_plan (begin
					(define _pj_prev_rr (symbol "__pj_prev_resultrow"))
					(define _pj_row_sym (symbol "__pj_row"))
					(list 'begin
						(list 'set _pj_prev_rr (symbol "resultrow"))
						(list 'set (symbol "resultrow")
							(list 'lambda (list _pj_row_sym)
								(list 'insert prejoin_schema prejointbl
									(cons 'list prejoin_column_names)
									(list 'list
										(cons 'list (map prejoin_column_names (lambda (col)
											(list 'get_assoc _pj_row_sym col)))))
									(list)
									(list 'lambda (list) true)
									true)))
						prejoin_materialize_rowplan
						(list 'set (symbol "resultrow") _pj_prev_rr))))
				/* Design contract:
				Keep get_column / aggregate / window sentinels logical for as long as
				possible. Materialized stages may register lineage and visible schemas,
				but must not eagerly bake physical .prejoin/.cache field names into the
				next logical stage. The final get_column -> scan symbol substitution
				should happen only when building the actual scan/map/filter code path.
				The helper below is therefore only for scan-time lowering via the
				recursive replace_find_column of the next stage, not for pre-rewriting
				group keys/fields/having/order themselves. */
				/* rewrite all column references from the materialized source scope to
				the physical prejoin columns. Outer-scope references must stay intact. */
				(define rewrite_as_prejoin_column (lambda (expr) (match expr
					'((symbol get_column) src_alias ti col ci) (begin
						(define _visible_alias (visible_occurrence_alias src_alias))
						(define _scope_match (or (has? known_table_aliases src_alias)
							(has? known_table_aliases _visible_alias)))
						(define expr_lookup (materialized_source_expr_lookup prejointbl))
						(define _rewritten_source_expr (rewrite_source_aliases prejoin_alias_map (normalize_visible_aliases expr)))
						(define _logical_source_expr (lower_prejoin_lineage_expr _rewritten_source_expr))
						(define _lookup_exprs (list
							expr
							(normalize_visible_aliases expr)
							(normalize_canonical_aliases expr)
							_rewritten_source_expr
							(normalize_canonical_aliases _rewritten_source_expr)
							_logical_source_expr
							(normalize_canonical_aliases _logical_source_expr)))
						(define direct_field (if (nil? expr_lookup) nil
							(reduce _lookup_exprs (lambda (found lookup_expr)
								(if (not (nil? found))
									found
									(reduce (materialized_source_expr_keys lookup_expr) (lambda (found2 key)
										(if (not (nil? found2)) found2
											(coalesce (expr_lookup key) nil)))
										nil)))
								nil)))
						(if (not (nil? direct_field))
							(list (quote get_column) prejoin_alias false direct_field false)
							(if _scope_match
								(list (quote get_column) prejoin_alias false
									(sanitize_temp_name
										(serialize_canonical_expr (canonicalize_expr _logical_source_expr prejoin_alias_map)))
									false)
								expr)))
					'((quote get_column) src_alias ti col ci) (begin
						(define _visible_alias (visible_occurrence_alias src_alias))
						(define _scope_match (or (has? known_table_aliases src_alias)
							(has? known_table_aliases _visible_alias)))
						(define expr_lookup (materialized_source_expr_lookup prejointbl))
						(define _rewritten_source_expr (rewrite_source_aliases prejoin_alias_map (normalize_visible_aliases expr)))
						(define _logical_source_expr (lower_prejoin_lineage_expr _rewritten_source_expr))
						(define _lookup_exprs (list
							expr
							(normalize_visible_aliases expr)
							(normalize_canonical_aliases expr)
							_rewritten_source_expr
							(normalize_canonical_aliases _rewritten_source_expr)
							_logical_source_expr
							(normalize_canonical_aliases _logical_source_expr)))
						(define direct_field (if (nil? expr_lookup) nil
							(reduce _lookup_exprs (lambda (found lookup_expr)
								(if (not (nil? found))
									found
									(reduce (materialized_source_expr_keys lookup_expr) (lambda (found2 key)
										(if (not (nil? found2)) found2
											(coalesce (expr_lookup key) nil)))
										nil)))
								nil)))
						(if (not (nil? direct_field))
							(list (quote get_column) prejoin_alias false direct_field false)
							(if _scope_match
								(list (quote get_column) prejoin_alias false
									(sanitize_temp_name
										(serialize_canonical_expr (canonicalize_expr _logical_source_expr prejoin_alias_map)))
									false)
								expr)))
					(cons sym args) (cons sym (map args rewrite_as_prejoin_column))
					expr)))
				/* Preserve logical lineage into the recursive stage.
				Do not carry physical .prejoin/.cache column names forward; instead lower
				the raw stage expressions back onto their logical source lineage first.
				build_scan stays the only place that finally substitutes onto the current
				stage's physical scan symbols. */
				(define grouped_fields (map_assoc raw_fields (lambda (k v)
					(lower_prejoin_lineage_expr v))))
				(define grouped_keys (map (coalesce raw_stage_group '()) lower_prejoin_lineage_expr))
				(define grouped_stage_alias_result (if (nil? grouped_keys)
					nil
					(make_keytable_schema schema prejointbl grouped_keys prejoin_alias)))
				(define grouped_stage_alias (if (nil? grouped_stage_alias_result) nil
					(car grouped_stage_alias_result)))
				(define grouped_stage_key_names (if (nil? grouped_stage_alias_result) '()
					(car (cdr grouped_stage_alias_result))))
				(define rewrite_group_key_to_group_alias (lambda (expr)
					(coalesce
						(reduce (produceN (count grouped_keys)) (lambda (found i)
							(if (not (nil? found))
								found
								(if (equal? expr (nth grouped_keys i))
									(list (quote get_column) grouped_stage_alias false (nth grouped_stage_key_names i) false)
									nil)))
							nil)
						(match expr
							(cons sym args) (cons sym (map args rewrite_group_key_to_group_alias))
							expr))))
				(define grouped_outer_condition_aliases (map _grp_ps_tables (lambda (td) (match td
					'(tv _ _ _ _) (if (nil? tv) "" tv)
					""))))
				(define grouped_outer_condition_term? (lambda (expr)
					(reduce (extract_tblvars expr) (lambda (acc tv)
						(or acc (has? grouped_outer_condition_aliases tv)))
						false)))
				(define countlike_prejoin_aggregate_expr? (lambda (expr) (match expr
					1 true
					'(if _ 1 0) true
					'(if _ true false) true
					'(if _ false true) true
					'(if _ 0 1) true
					_ false)))
				(define rewrite_local_prejoin_count_term (lambda (expr) (match expr
					(cons (symbol aggregate) agg_args)
						(match agg_args
							'(agg_expr + 0)
								(if (countlike_prejoin_aggregate_expr? agg_expr)
									(list (quote aggregate) 1 + 0)
									expr)
							_ expr)
					(cons '(quote aggregate) agg_args)
						(match agg_args
							'(agg_expr + 0)
								(if (countlike_prejoin_aggregate_expr? agg_expr)
									(list (quote aggregate) 1 + 0)
									expr)
							_ expr)
					(cons sym args) (cons sym (map args rewrite_local_prejoin_count_term))
					expr)))
				(define keep_grouped_post_group_term (lambda (expr)
					(if (grouped_outer_condition_term? expr)
						expr
						(if (equal? (extract_aggregates expr) '())
							nil
							(rewrite_local_prejoin_count_term expr)))))
				(define grouped_having
					(rewrite_group_key_to_group_alias (lower_prejoin_lineage_expr raw_stage_post_group_condition)))
				(define grouped_order (if (nil? raw_stage_order) nil
					(map raw_stage_order (lambda (o) (match o '(col dir)
						(list (lower_prejoin_lineage_expr col) dir))))))
				(define grouped_outer_tables (map _grp_ps_tables (lambda (td) (match td
					'(tv tschema ttbl toisOuter je)
					(list (if (nil? tv) ttbl tv) tschema ttbl toisOuter je)
					td))))
				(define grouped_outer_aliases (map grouped_outer_tables (lambda (td) (match td '(tv _ _ _ _) tv ""))))
				(define grouped_outer_schema_bindings (merge (map grouped_outer_tables (lambda (td) (match td
					'(tv tschema ttbl _ _)
					(list tv (materialized_source_schema tschema ttbl tv schemas))
					'())))))
					/* recursive call with single prejoin table.
					Contract:
					1. The prejoin materializes the complete row-domain that can be decided
					   before grouping: local join predicates, row-local filters and
					   lineage columns needed later.
					2. The grouped cache built on top of that prejoin must only see
					   group-domain conditions. Terms containing aggregates are rewritten
					   against keytable/temp columns and evaluated on the grouped table.
					3. Purely local prejoin predicates must not be copied into the grouped
					   cache suffix again, otherwise unrelated grouped plans alias the same
					   prejoin differently and cache names/filters drift apart.
					Scoped groups keep their outer tables outside the prejoin so later
					field expressions can still read them after the keytable LEFT JOIN. */
				(define grouped_plan_condition_base (if (nil? _grp_ps_tables)
					nil
					(begin
						(define grouped_plan_condition_base_raw (combine_and_terms
							(filter (map (flatten_and_terms (coalesceNil raw_post_group_condition true))
								keep_grouped_post_group_term)
								(lambda (x) (and (not (nil? x)) (not (equal? x true)))))))
						(if (or (nil? grouped_plan_condition_base_raw) (equal? grouped_plan_condition_base_raw true))
							nil
							(lower_prejoin_lineage_expr grouped_plan_condition_base_raw)))))
				(define recursive_replace_find_column (lambda (expr)
					(match expr
						'((symbol get_column) alias_ _ _ _) (begin
							(define resolved (replace_find_column expr))
							(match resolved
								'((symbol get_column) resolved_alias _ _ _)
								(if (has? grouped_outer_aliases resolved_alias)
									resolved
									(rewrite_as_prejoin_column resolved))
								'((quote get_column) resolved_alias _ _ _)
								(if (has? grouped_outer_aliases resolved_alias)
									resolved
									(rewrite_as_prejoin_column resolved))
								_ resolved))
						'((quote get_column) alias_ _ _ _) (begin
							(define resolved (replace_find_column expr))
							(match resolved
								'((symbol get_column) resolved_alias _ _ _)
								(if (has? grouped_outer_aliases resolved_alias)
									resolved
									(rewrite_as_prejoin_column resolved))
								'((quote get_column) resolved_alias _ _ _)
								(if (has? grouped_outer_aliases resolved_alias)
									resolved
									(rewrite_as_prejoin_column resolved))
								_ resolved))
						(cons sym args) (cons sym (map args recursive_replace_find_column))
						expr)))
				(define recursive_replace_find_column_condition (lambda (expr)
					(match expr
						'((symbol get_column) alias_ ti col ci) (begin
							(define resolved (replace_find_column expr))
							(match resolved
								'((symbol get_column) resolved_alias _ resolved_col _)
								(if (has? grouped_outer_aliases resolved_alias)
									(list (quote outer) (symbol (concat resolved_alias "." resolved_col)))
									(recursive_replace_find_column resolved))
								'((quote get_column) resolved_alias _ resolved_col _)
								(if (has? grouped_outer_aliases resolved_alias)
									(list (quote outer) (symbol (concat resolved_alias "." resolved_col)))
									(recursive_replace_find_column resolved))
								_ (recursive_replace_find_column resolved)))
						'((quote get_column) alias_ ti col ci) (begin
							(define resolved (replace_find_column expr))
							(match resolved
								'((symbol get_column) resolved_alias _ resolved_col _)
								(if (has? grouped_outer_aliases resolved_alias)
									(list (quote outer) (symbol (concat resolved_alias "." resolved_col)))
									(recursive_replace_find_column resolved))
								'((quote get_column) resolved_alias _ resolved_col _)
								(if (has? grouped_outer_aliases resolved_alias)
									(list (quote outer) (symbol (concat resolved_alias "." resolved_col)))
									(recursive_replace_find_column resolved))
								_ (recursive_replace_find_column resolved)))
						(cons sym args) (cons sym (map args recursive_replace_find_column_condition))
						expr)))
				(define grouped_having_for_recursive (if (nil? grouped_having) nil
					(if (nil? _grp_ps_tables)
						grouped_having
						(recursive_replace_find_column_condition grouped_having))))
				(define grouped_plan_condition grouped_plan_condition_base)
				/* rebuild group stage for recursive call.
				HAVING stays attached to the recursive grouped stage. Only deferred
				post-group outer predicates continue as condition. */
				(define grouped_stage (if is_dedup
					(make_dedup_stage grouped_keys
						(if (nil? _stage_scope) nil (list prejoin_alias)))
					(make_group_stage grouped_keys grouped_having_for_recursive grouped_order stage_limit stage_offset
						(if (nil? _stage_scope) nil (list prejoin_alias))
						nil)))
				(define grouped_fields_for_recursive (if is_dedup
					(map_assoc raw_fields (lambda (k v)
						(recursive_replace_find_column v)))
					grouped_fields))
				(define transform_recursive_stage (lambda (s)
					(begin
						(define _sg (coalesceNil (stage_group_cols s) '()))
						(define _so (coalesceNil (stage_order_list s) '()))
						(define _spa (stage_partition_aliases s))
						(if (stage_is_dedup s)
							(stage_preserve_cache_meta s
								(make_dedup_stage
									(map _sg recursive_replace_find_column)
									_spa))
							(if (and (not (nil? _spa)) (or (nil? _sg) (equal? _sg '())))
								(stage_preserve_cache_meta s
									(make_partition_stage
										_spa
										(map _so (lambda (o) (match o '(col dir) (list (recursive_replace_find_column col) dir))))
										(coalesceNil (stage_limit_partition_cols s) 0)
										(stage_limit_val s)
										(stage_offset_val s)
										(stage_init_code s)))
								(stage_preserve_cache_meta s
									(make_group_stage
										(map _sg recursive_replace_find_column)
										(recursive_replace_find_column (stage_having_expr s))
										(map _so (lambda (o) (match o '(col dir) (list (recursive_replace_find_column col) dir))))
										(stage_limit_val s)
										(stage_offset_val s)
										_spa
										(stage_init_code s))))))))
				(define grouped_all_stages (cons grouped_stage
					(if is_dedup
						(map rest_groups transform_recursive_stage)
						rest_groups)))
				/* drop partition stages covered by the prejoin (all tables materialized) */
				(define remaining_partition_stages (filter partition_stages (lambda (ps)
					(not (reduce (coalesceNil (stage_partition_aliases ps) '()) (lambda (acc a)
						(or acc (has? known_table_aliases a))) false)))))
					(define grouped_result (if (nil? _grp_ps_tables)
						(begin
							(define no_outer_group_condition_raw (combine_and_terms
								(filter (flatten_and_terms (coalesceNil raw_post_group_condition true))
									contains_aggregate)))
							(define no_outer_group_condition (if (or (nil? no_outer_group_condition_raw) (equal? no_outer_group_condition_raw true))
								nil
								no_outer_group_condition_raw))
							/* no outer-scope aliases remain here, so the recursive call is a
							plain single-table GROUP BY over the materialized prejoin table.
							Only aggregate-dependent terms survive into the grouped filter. */
							(define no_outer_group_stage (if is_dedup
								(make_dedup_stage raw_stage_group nil)
								(make_group_stage raw_stage_group raw_stage_having raw_stage_order stage_limit stage_offset nil nil)))
						(build_queryplan schema
							(list (list prejoin_alias schema prejointbl false nil))
							raw_fields
							no_outer_group_condition
							(merge (list no_outer_group_stage) rest_groups remaining_partition_stages)
							(merge schemas (list prejoin_alias prejoin_schema_def))
							recursive_replace_find_column
							update_target))
					(build_queryplan schema
						(merge (list (list prejoin_alias schema prejointbl false nil)) grouped_outer_tables)
						grouped_fields_for_recursive
						grouped_plan_condition
						(merge grouped_all_stages remaining_partition_stages)
						(merge schemas (list prejoin_alias prejoin_schema_def) grouped_outer_schema_bindings)
						recursive_replace_find_column
						update_target)))
				/* build per-source-table incremental trigger functions.
				Deduplicate by physical table to avoid duplicate triggers. */
				(define seen_trigger_tables (newsession))
				(define pj_trigger_registrations
					(filter (map tables (lambda (trigger_tbl)
						(match trigger_tbl '(trigger_tv src_schema src_tbl _ _)
							(begin
								(define trigger_table_key (concat src_schema "." src_tbl))
								(if (or (temp_source_table? src_tbl) (seen_trigger_tables trigger_table_key)) nil
									(begin (seen_trigger_tables trigger_table_key true)
										/* collect (pj_col base_col) pairs for this source table */
										(define ti_col_pairs
											(reduce prejoin_columns (lambda (acc mc)
												(match (cadr mc)
													'((symbol get_column) tv _ col _)
													(if (equal? tv trigger_tv)
														(merge acc (list (list (car mc) col)))
														acc)
													'((quote get_column) tv _ col _)
													(if (equal? tv trigger_tv)
														(merge acc (list (list (car mc) col)))
														acc)
													acc)) (list)))
										/* DELETE trigger: run helper-row removal through the shared
										DML planner so internal maintenance stays aligned with
										ordinary DELETE/$update semantics. */
										(define delete_fn
											(eval (list 'lambda (list 'OLD 'NEW 'session)
												(build_prejoin_delete_plan prejoin_schema prejoin_table_name ti_col_pairs))))
										/* INSERT trigger: scan other tables with T_i cols fixed to NEW, insert rows.
										Design contract: planner-owned prejoin helpers are cache-engine tables.
										After restart or eviction they may exist as an empty cache shell before any
										full materialization has happened. Incremental triggers must NOT seed such
										an empty helper with partial rows, otherwise table_empty? stops being a
										reliable bootstrap signal and later GROUP queries will skip the required
										full rebuild. Therefore all incremental maintenance is gated on the helper
										already containing a materialized baseline. */
										(define raw_insert_fn
											(eval (list 'lambda (list 'OLD 'NEW 'session)
												(build_pj_insert_scan tables condition trigger_tv true prejoin_schema prejoin_table_name prejoin_columns prejoin_column_names))))
										(define insert_fn
											(eval (list 'lambda (list 'OLD 'NEW 'session)
												(list 'if (list 'table_empty? prejoin_schema prejoin_table_name)
													true
													(list raw_insert_fn 'OLD 'NEW 'session)))))
										/* UPDATE trigger: delete old prejoin rows + insert new for any row change.
										Code-generator pattern: embed delete_fn/insert_fn as proc literals in body
										so no closure capture — serializes cleanly for persistence. The same empty-
										helper contract applies here: if no baseline is materialized yet, skip the
										incremental step and let the next query rebuild the full cache. */
										(define raw_update_fn (eval (list 'lambda (list 'OLD 'NEW 'session) (list 'begin (list delete_fn 'OLD 'NEW 'session) (list raw_insert_fn 'OLD 'NEW 'session)))))
										(define update_fn
											(eval (list 'lambda (list 'OLD 'NEW 'session)
												(list 'if (list 'table_empty? prejoin_schema prejoin_table_name)
													true
													(list raw_update_fn 'OLD 'NEW 'session)))))
										/* emit the register call as an S-expression to be executed at query time */
										(list 'register_prejoin_incremental src_schema src_tbl prejoin_schema prejoin_table_name
											delete_fn insert_fn update_fn))))))) (lambda (x) (not (nil? x)))))
				/* assemble: create (if not exists) + materialize if empty + register triggers + grouped result */
				(cons 'begin (merge
					(list
						(list 'if (list 'createtable pj_schema prejointbl
							(cons 'list (map prejoin_column_names (lambda (col) (list 'list "column" col "any" '(list) '(list)))))
							query_temp_table_options_code true)
							(list 'time prejoin_materialize_plan "materialize")
							(list 'if (list 'table_empty? pj_schema prejointbl)
								(list 'time prejoin_materialize_plan "materialize")
								nil)))
					pj_trigger_registrations
					(list grouped_result)))
			)
		)
	) (optimize (begin
			/* grouping has been removed; now to the real data: */
			(if (and (not (nil? rest_groups)) (not (equal? rest_groups '()))) (error "non-group stage must be last"))
			(if has_window (begin
				/* ========= Window function scan path (LAG/LEAD) ========= */
				/* Case 8: different OVER clauses */
				(define first_over (nth (car window_funcs_all) 2))
				(if (not (reduce window_funcs_all (lambda (ok wf) (and ok (equal? (nth wf 2) first_over))) true))
					(error "multiple window functions with different OVER clauses not yet supported"))
				/* extract and resolve OVER info */
				(define over_partition (map (car first_over) replace_find_column))
				(define over_order (map (cadr first_over) (lambda (o) (match o '(col dir) (list (replace_find_column col) dir)))))
				(define effective_sort (merge (map over_partition (lambda (pe) (list pe <))) over_order))
				(define stage_order_resolved (map (coalesce stage_order '()) (lambda (x) (match x '(col dir) (list (replace_find_column col) dir)))))
				(define wf_resolved (map window_funcs_all (lambda (wf) (match wf '(fn args over)
					(list fn (map args replace_find_column) over)))))
				/* ========= ORC window function descriptors ========= */
				/* Build a mapfn that passes $set + N extra values through as (list $set composite).
				For 1 col: composite = scalar; for N>1: composite = (list v0 v1 ...). */
				(define build_key_mapfn (lambda (col_names) (begin
					(define key_params (map (produceN (count col_names) (lambda (i) i)) (lambda (i) (symbol (concat "__k" i)))))
					(define key_expr (if (equal? (count key_params) 1) (car key_params) (cons (quote list) key_params)))
					(define mapfn_params (cons (symbol "$set") key_params))
					(define mapfn_body (list (quote list) (symbol "$set") key_expr))
					(eval (list (quote lambda) mapfn_params mapfn_body)))))
				/* Build a mapfn for aggregate window functions: $set + sort_cols + agg_col */
				(define build_agg_mapfn (lambda (agg_col_name sort_col_names) (begin
					(define all_cols (merge sort_col_names (list agg_col_name)))
					(define params (map (produceN (count all_cols) (lambda (i) i)) (lambda (i) (symbol (concat "__v" i)))))
					(define mapfn_params (cons (symbol "$set") params))
					(define mapfn_body (cons (quote list) (cons (symbol "$set") params)))
					(eval (list (quote lambda) mapfn_params mapfn_body)))))
				/* Extract column name from a resolved expression */
				(define extract_col_name (lambda (expr) (match expr
					'((symbol get_column) _ _ c _) c
					'((quote get_column) _ _ c _) c
					_ nil)))
			/* orc_window_descriptor: fn × args × sort_col_names → (extra_mapcols mapfn reducefn reduceinit)
				Returns nil for non-ORC functions (LAG/LEAD stay on window_mut path). */
				(define orc_window_descriptor (lambda (fn wf_args sort_col_names)
					(match fn
						"ROW_NUMBER" (list '()
							(lambda ($set) (list $set))
							(lambda (acc mapped) (begin ((car mapped) (+ acc 1)) (+ acc 1)))
							0)
						"RANK" (list sort_col_names
							(build_key_mapfn sort_col_names)
							(lambda (acc mapped)
								(begin
									(define setter (car mapped))
									(define key (cadr mapped))
									(define prev_rank (nth acc 0))
									(define prev_rownum (nth acc 1))
									(define new_rownum (+ prev_rownum 1))
									(define new_rank (if (equal? key (nth acc 2)) prev_rank new_rownum))
									(setter new_rank)
									(list new_rank new_rownum key)))
							(list 0 0 nil))
						"DENSE_RANK" (list sort_col_names
							(build_key_mapfn sort_col_names)
							(lambda (acc mapped)
								(begin
									(define setter (car mapped))
									(define key (cadr mapped))
									(define prev_rank (car acc))
									(define new_rank (if (equal? key (cadr acc)) prev_rank (+ prev_rank 1)))
									(setter new_rank)
									(list new_rank key)))
							(list 0 nil))
						/* registry-based ordered aggregates as running ORC (only if ordered=true) */
						_ (begin
							(define agg_desc (sql_aggregates fn))
							(if (or (nil? agg_desc) (not (nth agg_desc 2))) nil
								(if (nil? wf_args) nil
									(begin
										(define agg_col (extract_col_name (car wf_args)))
										(if (nil? agg_col) nil
											(begin
												(define agg_reduce (car agg_desc))
												(define agg_neutral (cadr agg_desc))
												/* GROUP_CONCAT: build reducer with separator from args */
												(if (equal? fn "GROUP_CONCAT")
													(begin
														(define sep (if (> (count wf_args) 1) (cadr wf_args) ","))
														(list (list agg_col)
															(lambda ($set v) (list $set v))
															(lambda (acc mapped) (begin
																(define v (cadr mapped))
																(define new_acc (if (nil? acc) (concat v) (concat acc sep v)))
																((car mapped) new_acc)
																new_acc))
															nil))
													(list (list agg_col)
														(lambda ($set v) (list $set v))
														(lambda (acc mapped) (begin
															(define new_acc (agg_reduce acc (cadr mapped)))
															((car mapped) new_acc)
															new_acc))
														agg_neutral))))))))
				)))
				(define is_orc_window (lambda (wf) (match wf '(fn args _) (not (nil? (orc_window_descriptor fn args '()))))))
				/* aggregate window: look up fn in sql_aggregates registry → (reduce neutral ordered) */
				(define is_agg_window (lambda (wf) (match wf '(fn _ _) (not (nil? (sql_aggregates fn))))))
				/* is_ordered_agg: true if the aggregate is order-sensitive (e.g. GROUP_CONCAT) */
				(define is_ordered_agg (lambda (wf) (match wf '(fn _ _) (begin
					(define reg (sql_aggregates fn))
					(if (nil? reg) false (nth reg 2))))))
				/* classify: ORC (has ORDER BY + ORC-eligible or ordered aggregate),
				aggregate (no ORDER BY, or non-ordered aggregate ignoring ORDER BY),
				LAG/LEAD (everything else) */
				(define has_over_order (not (equal? over_order '())))
				(define all_orc_window (and has_over_order (reduce wf_resolved (lambda (acc wf) (and acc (or (is_orc_window wf) (is_ordered_agg wf)))) true)))
				/* agg window: non-ordered aggs always, OR ordered aggs WITHOUT ORDER BY (keytable, not ORC) */
				(define all_agg_window (and (not all_orc_window) (reduce wf_resolved (lambda (acc wf) (and acc (is_agg_window wf) (or (not (is_ordered_agg wf)) (not has_over_order)))) true)))
				(if all_orc_window
					(match tables
						/* ========= ORC materialization (ROW_NUMBER, RANK, DENSE_RANK, ...) ========= */
						'('(tblvar schema tbl isOuter _)) (begin
							/* extract ORC sort columns from OVER ORDER BY */
							(define orc_sort_col_names (map over_order (lambda (o) (match o '(col dir) (match col
								'((symbol get_column) _ _ c _) c
								'((quote get_column) _ _ c _) c
								_ (match (replace_find_column col)
									'((symbol get_column) _ _ c _) c
									'((quote get_column) _ _ c _) c
									_ (error (concat "unsupported ORC sort expression: " col))))))))
							(define orc_sort_dirs_vals (map over_order (lambda (o) (match o '(col dir)
								(if (equal? dir >) true false)))))
							/* get descriptor for the first window function (all share same OVER) */
							(define first_wf (car wf_resolved))
							(define wf_fn (car first_wf))
							(define wf_args (cadr first_wf))
							(define descriptor (orc_window_descriptor wf_fn wf_args orc_sort_col_names))
							(define inner_extra_mapcols (nth descriptor 0))
							(define inner_mapfn (nth descriptor 1))
							(define inner_reducefn (nth descriptor 2))
							(define inner_reduceinit (nth descriptor 3))
							/* partition wrapper: prepend partition cols, wrap reducer with boundary reset */
							(define has_partition (not (equal? over_partition '())))
							(define partition_col_names (if has_partition
								(map over_partition (lambda (pe) (match pe
									'((symbol get_column) _ _ c _) c
									'((quote get_column) _ _ c _) c
									_ (match (replace_find_column pe)
										'((symbol get_column) _ _ c _) c
										'((quote get_column) _ _ c _) c
										_ (error (concat "unsupported partition expression: " pe))))))
								'()))
							(define extra_mapcols (if has_partition (merge partition_col_names inner_extra_mapcols) inner_extra_mapcols))
							(define orc_mapfn (if has_partition (begin
							/* build mapfn: ($set part_cols... inner_cols...) → (cons partition_key inner_mapped)
								The inner reducer sees (cdr mapped); wrapper sees (car mapped) as partition key. */
								(define n_part (count partition_col_names))
								(define n_inner (count inner_extra_mapcols))
								(define all_params (cons (symbol "$set")
									(map (produceN (+ n_part n_inner) (lambda (i) i)) (lambda (i) (symbol (concat "__p" i))))))
								(define part_syms (slice all_params 1 (+ 1 n_part)))
								(define inner_syms (slice all_params (+ 1 n_part) (+ 1 n_part n_inner)))
								(define pk_expr (if (equal? n_part 1) (car part_syms) (cons (quote list) part_syms)))
								(define inner_call (cons inner_mapfn (cons (symbol "$set") inner_syms)))
								(eval (list (quote lambda) all_params (list (quote cons) pk_expr inner_call))))
								inner_mapfn))
							(define orc_reducefn (if has_partition (begin
								/* wrap: acc = (list inner_acc prev_pk); mapped = (cons pk inner_mapped) */
								(lambda (acc mapped)
									(begin
										(define pk (car mapped))
										(define inner_mapped (cdr mapped))
										(define prev_pk (cadr acc))
										(define inner_acc (car acc))
										(define eff_acc (if (or (nil? prev_pk) (equal? pk prev_pk)) inner_acc inner_reduceinit))
										(define new_inner (inner_reducefn eff_acc inner_mapped))
										(list new_inner pk))))
								inner_reducefn))
							(define orc_reduceinit (if has_partition (list inner_reduceinit nil) inner_reduceinit))
							/* unique temp column name */
							(define orc_col_name (concat ".orc_" wf_fn "_" tbl))
							/* compile time: add bare column so the scan plan can reference it */
							(createcolumn schema tbl orc_col_name "any" '() '("temp" true))
							/* replace window_func references with ORC column read */
							(define replace_wf (lambda (expr) (match expr
								(cons (symbol window_func) _) '((quote get_column) (eval tblvar) false (eval orc_col_name) false)
								(cons sym args_) (cons sym (map args_ replace_wf))
								expr)))
							(define new_fields (map_assoc fields (lambda (k v) (replace_wf v))))
							/* runtime plan: createcolumn with ORC params, then the actual scan */
							/* sortcols: partition cols (ASC) first, then ORDER BY cols */
							(define full_sort_cols (if has_partition (merge partition_col_names orc_sort_col_names) orc_sort_col_names))
							(define full_sort_dirs (if has_partition
								(merge (map partition_col_names (lambda (_) false)) orc_sort_dirs_vals)
								orc_sort_dirs_vals))
							/* partitioncount is auto-detected from reduceinit shape: (list init nil) → 1 partition key */
							(define orc_setup (lambda ()
								(createcolumn schema tbl orc_col_name "any" '()
									(list "sortcols" full_sort_cols "sortdirs" full_sort_dirs
										"mapcols" extra_mapcols
										"mapfn" orc_mapfn "reducefn" orc_reducefn
										"reduceinit" orc_reduceinit "temp" true))))
							(define scan_plan (build_queryplan schema tables new_fields condition groups schemas replace_find_column nil))
							(list (quote begin) (list orc_setup) scan_plan)
						)
						(error "window functions on joined tables not yet supported"))
					(if all_agg_window
						(match tables
							'('(tblvar schema tbl isOuter _))
							(build_agg_window_plan schema tbl tblvar tables over_partition wf_resolved condition groups schemas replace_find_column fields isOuter replace_columns_from_expr extract_columns_for_tblvar scan_wrapper)
							(error "window functions on joined tables not yet supported"))
						(begin
							/* ========= LAG/LEAD scan path (unchanged) ========= */
							/* Case 3: conflicting ORDER BY */
							(if (and (not (equal? stage_order_resolved '())) (not (equal? effective_sort stage_order_resolved)))
								(error "window ORDER BY with outer ORDER BY not yet supported"))
							(if (reduce wf_resolved (lambda (acc wf) (match wf '(fn _ _)
								(or acc (and (not (equal? fn "LAG")) (not (equal? fn "LEAD")))))) false)
								(error (concat "unsupported window function in LAG/LEAD context: " (car (car wf_resolved)))))
							/* single table only */
							(match tables
								'('(tblvar schema tbl isOuter _)) (begin
									(set condition (replace_find_column (coalesceNil condition true)))
									(define has_partition (not (equal? over_partition '())))
									/* compute stride_cols: all columns needed in output and window args */
									(define non_window_cols (merge_unique (extract_assoc fields (lambda (k v)
										(extract_columns_for_tblvar tblvar (replace_find_column v))))))
									(define wf_arg_cols (merge_unique (map wf_resolved (lambda (wf) (match wf '(fn args _)
										(merge_unique (map args (lambda (a) (extract_columns_for_tblvar tblvar a)))))))))
									(define partition_col_names (merge_unique (map over_partition (lambda (pe) (match pe
										'((symbol get_column) _ _ col _) '(col)
										'((quote get_column) _ _ col _) '(col)
										'())))))
									(define stride_cols (merge_unique (list non_window_cols wf_arg_cols partition_col_names)))
									(define stride (count stride_cols))
									/* window parameters */
									(define max_lag (reduce wf_resolved (lambda (acc wf) (match wf '(fn args _)
										(if (equal? fn "LAG") (max acc (if (> (count args) 1) (cadr args) 1)) acc))) 0))
									(define max_lead (reduce wf_resolved (lambda (acc wf) (match wf '(fn args _)
										(if (equal? fn "LEAD") (max acc (if (> (count args) 1) (cadr args) 1)) acc))) 0))
									(define window_size (+ max_lag 1 max_lead))
									(define skip max_lead)
									(define flush_count skip)
									(define current_row_pos (- window_size 1 skip))
									/* emit_fn parameter symbols */
									(define num_emit_params (* window_size stride))
									(define emit_params (map (produceN num_emit_params (lambda (i) i)) (lambda (i) (symbol (concat "__w" i)))))
									/* helper: find column index in stride_cols */
									(define col_index (lambda (col) (car (reduce stride_cols (lambda (acc c) (match acc '(idx found)
										(if found acc (if (equal?? c col) (list idx true) (list (+ idx 1) false))))) (list 0 false)))))
									/* rewrite field expression for emit_fn */
									(define rewrite_for_emit (lambda (expr row_pos) (match expr
										(cons (symbol window_func) wf_rest) (begin
											(define fn (car wf_rest))
											(define wf_args (cadr wf_rest))
											(define wf_offset (if (> (count wf_args) 1) (cadr wf_args) 1))
											(define wf_pos (if (equal? fn "LAG") (- current_row_pos wf_offset) (+ current_row_pos wf_offset)))
											(rewrite_for_emit (replace_find_column (car wf_args)) wf_pos))
										'((symbol get_column) (eval tblvar) _ col _) (nth emit_params (+ (* row_pos stride) (col_index col)))
										'((quote get_column) (eval tblvar) _ col _) (nth emit_params (+ (* row_pos stride) (col_index col)))
										'((symbol get_column) nil _ col _) (rewrite_for_emit (replace_find_column expr) row_pos)
										(cons sym args_) (cons sym (map args_ (lambda (a) (rewrite_for_emit a row_pos))))
										expr)))
									/* build emit_fn: (lambda (__w0 __w1 ...) (resultrow (list field_rewrites...))) */
									(define emit_body '((symbol "resultrow") (cons (symbol "list") (map_assoc fields (lambda (k v) (rewrite_for_emit v current_row_pos))))))
									(define emit_fn_ast (list (quote lambda) emit_params emit_body))
									/* build neutral */
									(define neutral_list (merge (list skip 0 stride) (produceN (* window_size stride) (lambda (_) nil))))
									(define neutral_ast (cons (quote list) neutral_list))
									/* sort cols/dirs from effective_sort */
									(define ordercols (merge (map effective_sort (lambda (order_item) (match order_item '(col dir) (match col
										'((symbol get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list col) '())
										'((quote get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list col) '())
										_ '()))))))
									(define sort_dirs (merge (map effective_sort (lambda (order_item) (match order_item '(col dir) (match col
										'((symbol get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar) (list dir) '())
										'((quote get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar) (list dir) '())
										_ '()))))))
									/* filter setup */
									(define filtercols (extract_columns_for_tblvar tblvar condition))
									/* symbols for emit_fn and fresh neutral */
									(define efn_sym (symbol "__emit_fn"))
									(define nfn_sym (symbol "__fresh_neutral"))
									(if has_partition (begin
										/* === Case 4: PARTITION BY + ORDER BY === */
										(define window_end (+ 3 (* window_size stride)))
										/* partition key expression in mapfn */
										(define partition_col_syms (map over_partition (lambda (pe) (match pe
											'((symbol get_column) _ _ col _) (symbol (concat tblvar "." col))
											'((quote get_column) _ _ col _) (symbol (concat tblvar "." col))))))
										(define pk_value_expr (if (equal? (count partition_col_syms) 1)
											(car partition_col_syms)
											(cons (quote list) partition_col_syms)))
										/* mapfn: returns (list stride_vals... partition_key) */
										(define mapfn_ast (list (quote lambda)
											(map stride_cols (lambda (col) (symbol (concat tblvar "." col))))
											(list (quote append)
												(cons (quote list) (map stride_cols (lambda (col) (symbol (concat tblvar "." col)))))
												pk_value_expr)))
										/* neutral with nil partition key */
										(define neutral_partition_ast (cons (quote list) (merge neutral_list (list nil))))
										/* partition-aware reducer */
										(define reducer_ast (list (quote lambda) '('acc 'mapped) (list (quote begin)
											'('define 'pk '('nth 'mapped stride))
											'('define 'vs '('slice 'mapped 0 stride))
											'('define 'prev_pk '('nth 'acc window_end))
											'('define 'win '('slice 'acc 0 window_end))
											(list (quote if) '('or '('nil? 'prev_pk) '('equal? 'pk 'prev_pk))
												'('append '('window_mut 'win efn_sym 'vs) 'pk)
												(list (quote begin)
													(if (> flush_count 0) '('window_flush 'win efn_sym flush_count) true)
													'('append '('window_mut nfn_sym efn_sym 'vs) 'pk))))))
										/* build scan with post-flush */
										(define scan_plan (list (quote begin)
											(list (quote define) efn_sym emit_fn_ast)
											(list (quote define) nfn_sym neutral_ast)
											(if (> flush_count 0) (begin
												(list (quote begin)
													(list (quote define) (symbol "__scan_result")
														(scan_wrapper 'scan_order schema tbl
															(cons list filtercols)
															'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr condition)))
															(cons list ordercols)
															(cons list sort_dirs)
															0 0 -1
															(cons list stride_cols)
															mapfn_ast
															reducer_ast
															neutral_partition_ast
															isOuter))
													(list (quote window_flush) (list (quote slice) (symbol "__scan_result") 0 window_end) efn_sym flush_count)))
												(scan_wrapper 'scan_order schema tbl
													(cons list filtercols)
													'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr condition)))
													(cons list ordercols)
													(cons list sort_dirs)
													0 0 -1
													(cons list stride_cols)
													mapfn_ast
													reducer_ast
													neutral_partition_ast
													isOuter))))
										scan_plan
									) (begin
											/* === Case 1: ORDER BY only, no partition === */
											/* mapfn: returns (list stride_vals...) */
											(define mapfn_ast '((quote lambda)
												(map stride_cols (lambda (col) (symbol (concat tblvar "." col))))
												(cons (quote list) (map stride_cols (lambda (col) (symbol (concat tblvar "." col)))))))
											/* simple reducer */
											(define reducer_ast '((quote lambda) '('acc 'mapped) '('window_mut 'acc efn_sym 'mapped)))
											/* build scan with post-flush */
											(define scan_plan (list (quote begin)
												(list (quote define) efn_sym emit_fn_ast)
												(if (> flush_count 0) (begin
													(list (quote begin)
														(list (quote define) (symbol "__scan_result")
															(scan_wrapper 'scan_order schema tbl
																(cons list filtercols)
																'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr condition)))
																(cons list ordercols)
																(cons list sort_dirs)
																0 0 -1
																(cons list stride_cols)
																mapfn_ast
																reducer_ast
																neutral_ast
																isOuter))
														(list (quote window_flush) (symbol "__scan_result") efn_sym flush_count)))
													(scan_wrapper 'scan_order schema tbl
														(cons list filtercols)
														'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr condition)))
														(cons list ordercols)
														(cons list sort_dirs)
														0 0 -1
														(cons list stride_cols)
														mapfn_ast
														reducer_ast
														neutral_ast
														isOuter))))
											scan_plan
									))
								)
								(error "window functions on joined tables not yet supported")
				))))
			) (if (coalesce stage_order stage_limit stage_offset) (begin
					/* ordered or limited scan */
					/* TODO: ORDER, LIMIT, OFFSET -> find or create all tables that have to be nestedly scanned. when necessary create prejoins. */
					(set stage_order (map (coalesce stage_order '()) (lambda (x) (match x '(col dir) (list (replace_find_column col) dir)))))
					/* If ORDER BY targets a single inner-joined table, drive the join from that
					table so LIMIT/OFFSET applies on the actual sort source instead of the
					original parser order. This keeps semantics correct for cases like
					ORDER BY derived_tbl.col DESC LIMIT n over inner joins. */
					(define _order_driver_aliases (reduce (coalesce stage_order '()) (lambda (acc order_item) (match order_item
						'(col _dir) (match col
							'((symbol get_column) alias_ _ _ _) (append_unique acc alias_)
							'((quote get_column) alias_ _ _ _) (append_unique acc alias_)
							_ acc)
						acc))
						'()))
					(define _all_inner_tables (reduce tables (lambda (acc td) (match td
						'(_ _ _ isOuter _) (and acc (not isOuter))
						acc))
						true))
					(define _order_driver_alias (if (and _all_inner_tables (equal? 1 (count _order_driver_aliases)))
						(car _order_driver_aliases)
						nil))
					(define ordered_tables (if (nil? _order_driver_alias)
						tables
						(merge
							(filter tables (lambda (td) (match td '(tblvar _ _ _ _) (equal?? tblvar _order_driver_alias) false)))
							(filter tables (lambda (td) (match td '(tblvar _ _ _ _) (not (equal?? tblvar _order_driver_alias)) true))))))
					/* build_scan now takes is_first parameter to apply offset/limit only to outermost scan */
					(define build_scan (lambda (tables condition is_first last_scan_ctx)
						(match tables
							(cons '(tblvar schema tbl isOuter joinexpr) tables) (begin /* outer scan */
								(define scan_condition (lower_materialized_scan_condition schema tbl tblvar condition))
								(define visible_fields (lower_materialized_emit_assoc schema tbl tblvar fields))
								(define is_update_target_ord (and (not (nil? update_target)) (equal?? tblvar (nth update_target 0))))
								(define visible_ut_cols_ord (if is_update_target_ord
									(lower_materialized_emit_assoc schema tbl tblvar (nth update_target 1))
									'()))
								(define ut_extra_cols_ord (if is_update_target_ord
									(merge_unique (extract_assoc visible_ut_cols_ord (lambda (k v) (extract_columns_for_tblvar tblvar v))))
									'()))
								(set cols (merge_unique
									(list
										(merge_unique
											(cons
												(extract_columns_for_tblvar tblvar scan_condition)
												(extract_assoc visible_fields (lambda (k v) (extract_columns_for_tblvar tblvar v)))
											)
										)
										(merge_unique
											(cons
												(extract_outer_columns_for_tblvar tblvar scan_condition)
												(extract_assoc visible_fields (lambda (k v) (extract_outer_columns_for_tblvar tblvar v)))
											)
										)
										(extract_later_joinexpr_columns_for_tblvar tblvar tables)
										ut_extra_cols_ord
									)
								))
								(match (split_scan_condition isOuter (replace_find_column (coalesceNil joinexpr true)) scan_condition tables) '(now_condition later_condition) (begin
									(define effective_later_condition (if (and isOuter (equal? now_condition later_condition)) true later_condition))
									(set cols (merge_unique (list
										cols
										(extract_columns_for_tblvar tblvar effective_later_condition)
										(extract_outer_columns_for_tblvar tblvar effective_later_condition))))
									(set filtercols (merge_unique (list (extract_columns_for_tblvar tblvar now_condition) (extract_outer_columns_for_tblvar tblvar now_condition))))
									/* check partition_stages for this table (non-first tables may have per-table partition limits) */
									(define _ps_ord (if is_first nil
										(reduce partition_stages (lambda (a s) (if (nil? a) (if (has? (coalesceNil (stage_partition_aliases s) '()) tblvar) s nil) a)) nil)))
									/* use partition stage's order if available, otherwise use stage_order */
									(define _eff_order (if (nil? _ps_ord) stage_order (coalesceNil (stage_order_list _ps_ord) '())))
									/* extract order cols for this tblvar */
									(set ordercols (merge (map _eff_order (lambda (order_item) (match order_item '(col dir) (match col
										'((symbol get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list col) '())
										'((quote get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list col) '())
										_ '()
									))))))
									(set dirs (merge (map _eff_order (lambda (order_item) (match order_item '(col dir) (match col
										'((symbol get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar) (list dir) '())
										'((quote get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar) (list dir) '())
										_ '()
									))))))

									/* offset/limit: for partition-staged tables use their limits, else only outermost */
									(define scan_offset (if (not (nil? _ps_ord)) (coalesceNil (stage_offset_val _ps_ord) 0)
										(if is_first stage_offset 0)))
									(define scan_limit (if (not (nil? _ps_ord)) (coalesceNil (stage_limit_val _ps_ord) -1)
										(if is_first (coalesceNil stage_limit -1) -1)))
									(define scan_partcols (if (not (nil? _ps_ord)) (coalesceNil (stage_limit_partition_cols _ps_ord) 0)
										(if is_first stage_partcols 0)))

									(define ord_scan_mapcols (if is_update_target_ord (cons list (cons "$update" cols)) (cons list cols)))
									(define ord_scan_mapfn_params (if is_update_target_ord
										(cons (symbol "$update") (map cols (lambda(col) (symbol (concat tblvar "." col)))))
										(map cols (lambda(col) (symbol (concat tblvar "." col))))))
									/* emit init code from partition stage if present */
									(define _ps_init (if (nil? _ps_ord) nil (stage_init_code _ps_ord)))
									(define _ord_scan (scan_wrapper 'scan_order schema tbl
										/* condition */
										(cons list filtercols)
										'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr now_condition)))
										/* sortcols, sortdirs */
										(cons list ordercols)
										(cons list dirs)
										scan_partcols
										scan_offset
										scan_limit
										/* extract columns and store them into variables */
										ord_scan_mapcols
										(list (symbol "lambda") ord_scan_mapfn_params (build_scan tables effective_later_condition false (list schema tbl tblvar)))
										/* reduce+neutral for DML */
										(if is_update_target_ord (symbol "+") nil)
										(if is_update_target_ord 0 nil)
										isOuter
									))
									(if (nil? _ps_init) _ord_scan (list (quote begin) _ps_init _ord_scan))
								))
							)
							'() /* final inner */ (if (nil? update_target)
								(begin
									(define emit_fields (if (nil? last_scan_ctx) fields
										(match last_scan_ctx
											'(scan_schema scan_tbl scan_tblvar) (lower_materialized_emit_assoc scan_schema scan_tbl scan_tblvar fields)
											fields)))
									'('if (optimize (replace_columns_from_expr condition)) '((symbol "resultrow") (cons (symbol "list") (map_assoc emit_fields (lambda (k v) (replace_columns_from_expr v)))))))
								/* DML mode: emit mutation payload; actual DELETE/UPDATE runs in build_dml_plan's resultrow wrapper */
								(begin (define _ut_cols (nth update_target 1))
									(define _ut_tag (nth update_target 2))
									(define _ut_cols (if (nil? last_scan_ctx) _ut_cols
										(match last_scan_ctx
											'(scan_schema scan_tbl scan_tblvar) (lower_materialized_emit_assoc scan_schema scan_tbl scan_tblvar _ut_cols)
											_ut_cols)))
									(if (equal? _ut_cols '())
										'('if (optimize (replace_columns_from_expr condition))
											'((symbol "resultrow") (list (symbol "list") "__dml_tag" _ut_tag "__update" '$update "__values" nil))
											0)
										'('if (optimize (replace_columns_from_expr condition))
											'((symbol "resultrow") (list (symbol "list") "__dml_tag" _ut_tag "__update" '$update "__values" (cons (symbol "list") (map_assoc _ut_cols (lambda (k v) (replace_columns_from_expr v))))))
											0))))
						)
					))
					(build_scan ordered_tables (replace_find_column condition) true nil)
				) (begin
						/* unordered unlimited scan */

						/* TODO: sort tables according to join plan */
						/* TODO: match tbl to inner query vs string */
						(define build_scan (lambda (tables condition last_scan_ctx bound_update_expr)
							(match tables
								(cons '(tblvar schema tbl isOuter joinexpr) tables) (begin /* outer scan */
									(define scan_condition (lower_materialized_scan_condition schema tbl tblvar condition))
									(define visible_fields (lower_materialized_emit_assoc schema tbl tblvar fields))
									/* check if this table is the UPDATE target */
									(define is_update_target (and (not (nil? update_target)) (equal?? tblvar (nth update_target 0))))
									(define visible_ut_cols (if is_update_target
										(lower_materialized_emit_assoc schema tbl tblvar (nth update_target 1))
										'()))
									/* also extract cols needed for SET expressions in update_target */
									(define ut_extra_cols (if is_update_target
										(merge_unique (extract_assoc visible_ut_cols (lambda (k v) (extract_columns_for_tblvar tblvar v))))
										'()))
									(set cols (merge_unique
										(list
											(merge_unique
												(cons
													(extract_columns_for_tblvar tblvar scan_condition)
													(extract_assoc visible_fields (lambda (k v) (extract_columns_for_tblvar tblvar v)))
												)
											)
											(merge_unique
												(cons
													(extract_outer_columns_for_tblvar tblvar scan_condition)
													(extract_assoc visible_fields (lambda (k v) (extract_outer_columns_for_tblvar tblvar v)))
												)
											)
											(extract_later_joinexpr_columns_for_tblvar tblvar tables)
											ut_extra_cols
										)
									))
									/* For UPDATE target: prepend $update to mapcols */
									(define scan_mapcols (if is_update_target (cons list (cons "$update" cols)) (cons list cols)))
									(define scan_mapfn_params (if is_update_target
										(cons (symbol "$update") (map cols (lambda(col) (symbol (concat tblvar "." col)))))
										(map cols (lambda(col) (symbol (concat tblvar "." col))))))
									/* split condition in those ANDs that still contain get_column from tables and those evaluatable now */
									(match (split_scan_condition isOuter (replace_find_column (coalesceNil joinexpr true)) scan_condition tables) '(now_condition later_condition) (begin
										(define effective_later_condition (if (and isOuter (equal? now_condition later_condition)) true later_condition))
										(set cols (merge_unique (list
											cols
											(extract_columns_for_tblvar tblvar effective_later_condition)
											(extract_outer_columns_for_tblvar tblvar effective_later_condition))))
										(set filtercols (merge_unique (list (extract_columns_for_tblvar tblvar now_condition) (extract_outer_columns_for_tblvar tblvar now_condition))))
										/* optimize: skip .(1) DUAL scan when no columns needed (1 row, no data) */
										(if (and (equal? tbl ".(1)") (equal? cols (list)) (equal? filtercols (list)))
											(begin
												/* The skipped DUAL row still carries any constant/materialized
												predicate that split_scan_condition classified as now_condition.
												Forward it explicitly, otherwise no-FROM predicates like
												NOT IN (subselect) vanish before the materialized temp scan
												can lower their aggregate sentinel. */
												(define deferred_condition (combine_and_terms (merge
													(flatten_and_terms now_condition)
													(flatten_and_terms effective_later_condition))))
												(build_scan tables deferred_condition last_scan_ctx bound_update_expr))
											(begin
												(define next_update_expr (if is_update_target (symbol "__dml_update_bound") bound_update_expr))
												(define child_scan (build_scan tables effective_later_condition (list schema tbl tblvar) next_update_expr))
												(define scan_body (if is_update_target
													(list (list (symbol "lambda") (list (symbol "__dml_update_bound")) child_scan) (symbol "$update"))
													(if (nil? bound_update_expr) child_scan
														(list (list (symbol "lambda") (list (symbol "__dml_update_bound")) child_scan) bound_update_expr))))
												/* check partition_stages: does this table have a per-table partition limit? */
												(define _ps (reduce partition_stages (lambda (a s) (if (nil? a) (if (has? (coalesceNil (stage_partition_aliases s) '()) tblvar) s nil) a)) nil))
												(if (not (nil? _ps))
													/* === partition-limited scan_order === */
													(begin
														(define _ps_filtercols (merge_unique (list (extract_columns_for_tblvar tblvar now_condition) (extract_outer_columns_for_tblvar tblvar now_condition))))
														(define _ps_order (coalesceNil (stage_order_list _ps) '()))
														(define _ps_partcols (coalesceNil (stage_limit_partition_cols _ps) 0))
														(define _ps_limit (coalesceNil (stage_limit_val _ps) -1))
														(define _ps_offset (coalesceNil (stage_offset_val _ps) 0))
														(define _ps_ordercols (merge (map _ps_order (lambda (oi) (match oi '(col dir) (match col
															'((symbol get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list col) '())
															'((quote get_column) alias_ ti col _) (if ((if ti equal?? equal?) alias_ tblvar) (list col) '())
															_ '()))))))
														(define _ps_dirs (merge (map _ps_order (lambda (oi) (match oi '(col dir) (match col
															'((symbol get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar) (list dir) '())
															'((quote get_column) alias_ ti _ _) (if ((if ti equal?? equal?) alias_ tblvar) (list dir) '())
															_ '()))))))
														/* emit init code from partition stage if present */
														(define _ps_init2 (stage_init_code _ps))
														(define _ps_scan (scan_wrapper 'scan_order schema tbl
															(cons list (merge_unique _ps_filtercols cols))
															'((quote lambda) (map (merge_unique _ps_filtercols cols) (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr now_condition)))
															(cons list _ps_ordercols)
															(cons list _ps_dirs)
															_ps_partcols _ps_offset _ps_limit
															scan_mapcols
															(list (symbol "lambda") scan_mapfn_params scan_body)
															nil nil isOuter))
														(if (nil? _ps_init2) _ps_scan (list (quote begin) _ps_init2 _ps_scan)))
													/* === regular scan === */
													(scan_wrapper 'scan schema tbl
														(cons list filtercols)
														'((quote lambda) (map filtercols (lambda(col) (symbol (concat tblvar "." col)))) (optimize (replace_columns_from_expr now_condition)))
														scan_mapcols
														(list (symbol "lambda") scan_mapfn_params scan_body)
														(if is_update_target (symbol "+") nil)
														(if is_update_target 0 nil)
														nil
														isOuter
										))))
									))
								)
								'() /* final inner (=scalar) */ (if (nil? update_target)
									(begin
										(define emit_fields (if (nil? last_scan_ctx) fields
											(match last_scan_ctx
												'(scan_schema scan_tbl scan_tblvar) (lower_materialized_emit_assoc scan_schema scan_tbl scan_tblvar fields)
												fields)))
										'('if (optimize (replace_columns_from_expr condition)) '((symbol "resultrow") (cons (symbol "list") (map_assoc emit_fields (lambda (k v) (replace_columns_from_expr v)))))))
									/* DML mode: emit mutation payload; actual DELETE/UPDATE runs in build_dml_plan's resultrow wrapper */
									(begin (define _ut_cols (nth update_target 1))
										(define _ut_tag (nth update_target 2))
										(define _ut_cols (if (nil? last_scan_ctx) _ut_cols
											(match last_scan_ctx
												'(scan_schema scan_tbl scan_tblvar) (lower_materialized_emit_assoc scan_schema scan_tbl scan_tblvar _ut_cols)
												_ut_cols)))
										(if (equal? _ut_cols '())
											/* DELETE */
											(list (quote if) (list (quote optimize) (replace_columns_from_expr condition))
												(list (symbol "resultrow") (list (symbol "list") "__dml_tag" _ut_tag "__update" bound_update_expr "__values" nil))
												0)
											/* UPDATE */
											(list (quote if) (list (quote optimize) (replace_columns_from_expr condition))
												(list (symbol "resultrow") (list (symbol "list") "__dml_tag" _ut_tag "__update" bound_update_expr "__values" (cons (symbol "list") (map_assoc _ut_cols (lambda (k v) (replace_columns_from_expr v))))))
												0))))
							)
						))
						(build_scan tables (replace_find_column condition) nil nil)
			)))
	)))
)))
)
