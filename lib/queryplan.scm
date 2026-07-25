/*
Copyright (C) 2023, 2024, 2026  Carl-Philip Haensch

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
Neumann compiler rebuild
------------------------

The only legal planner pipeline is:

	parser AST -> untangle_query -> join_reorder -> build_queryplan

The compiler IR is structured, not flat, but deliberately compact:

	(qnode op id attrs children facts)

The operator tree/DAG keeps the semantic boundaries needed by Neumann/BTW2025
without copying whole query states through every pass.  Children are shared.
Passes must return the same node when nothing changed and allocate only along
the modified path.  A later physical emitter may produce a flat low-allocation
execution plan; the logical compiler must not flatten away the algebra.

untangle_query owns arbitrary subqueries, including scalar subqueries,
IN/EXISTS, nested dependent joins and FROM (SELECT ...).  During untangling the
DAG may contain depjoin and derived nodes; before it leaves untangle_query all
dependent joins and derived-table boundaries must be eliminated or made
relational.  It must never emit runtime scans, promises, materialized subquery
sources, or scalar fallback code.

Every query must compile within the context budget of 1000ms.
*/

/* ------------------------------------------------------------------------- */
/* Assoc and compact node helpers                                             */

(define qassoc_get (lambda (xs key default)
	(coalesceNil
		(reduce (coalesceNil xs '()) (lambda (found entry)
			(if (not (nil? found))
				found
				(match entry
					(cons k rest) (if (equal? k key)
						(if (equal? (count rest) 1) (car rest) rest)
						nil)
					nil)))
			nil)
		default)))

(define qassoc_set (lambda (xs key value)
	(cons (list key value)
		(filter (coalesceNil xs '()) (lambda (entry) (match entry
			(cons k _) (not (equal? k key))
			true))))))

(define qnode (lambda (op id attrs children facts)
	(list (quote qnode)
		op
		id
		(coalesceNil attrs '())
		(coalesceNil children '())
		(coalesceNil facts '()))))

(define qop (lambda (node) (nth node 1)))
(define qid (lambda (node) (nth node 2)))
(define qattrs (lambda (node) (nth node 3)))
(define qchildren (lambda (node) (nth node 4)))
(define qfacts (lambda (node) (nth node 5)))
(define qattr (lambda (node key default) (qassoc_get (qattrs node) key default)))
(define qfact (lambda (node key default) (qassoc_get (qfacts node) key default)))

(define qnode_with_attrs (lambda (node attrs)
	(if (equal? attrs (qattrs node))
		node
		(qnode (qop node) (qid node) attrs (qchildren node) (qfacts node)))))

(define qnode_with_children (lambda (node children)
	(if (equal? children (qchildren node))
		node
		(qnode (qop node) (qid node) (qattrs node) children (qfacts node)))))

(define qnode_with_facts (lambda (node facts)
	(if (equal? facts (qfacts node))
		node
		(qnode (qop node) (qid node) (qattrs node) (qchildren node) facts))))

/* ------------------------------------------------------------------------- */
/* Query and unnesting context                                                */

/*
Query:
	(qir kind schema root return context facts)

Context is a shared chain.  Scope changes cons a new context node instead of
copying all maps:
	(uctx parent attrs)

attrs contains deltas for outer-refs, cclasses, repr, domain, shared-roots and
the compile-budget-ms requirement.
*/

(define uctx (lambda (parent attrs)
	(list (quote uctx) parent (coalesceNil attrs '()))))
(define uctx_parent (lambda (ctx) (nth ctx 1)))
(define uctx_attrs (lambda (ctx) (nth ctx 2)))
(define uctx_attr (lambda (ctx key default)
	(coalesceNil
		(qassoc_get (uctx_attrs ctx) key nil)
		(if (nil? (uctx_parent ctx))
			default
			(uctx_attr (uctx_parent ctx) key default)))))
(define initial_uctx (lambda (outer_schemas)
	(uctx nil
		(list
			(list (quote outer-refs) (coalesceNil outer_schemas '()))
			(list (quote cclasses) '())
			(list (quote repr) '())
			(list (quote domain) '())
			(list (quote shared-roots) '())
			(list (quote compile-budget-ms) 1000)))))

(define qir (lambda (kind schema root return context facts)
	(list (quote qir)
		kind
		schema
		root
		return
		context
		(coalesceNil facts '()))))
(define ir_kind (lambda (ir) (nth ir 1)))
(define ir_schema (lambda (ir) (nth ir 2)))
(define ir_root (lambda (ir) (nth ir 3)))
(define ir_return (lambda (ir) (nth ir 4)))
(define ir_context_of (lambda (ir) (nth ir 5)))
(define ir_facts (lambda (ir) (nth ir 6)))
(define ir_with_return (lambda (ir return)
	(if (equal? return (ir_return ir))
		ir
		(qir (ir_kind ir) (ir_schema ir) (ir_root ir) return (ir_context_of ir) (ir_facts ir)))))

/* Compatibility accessors used only by rebuild tests. */
(define ir_sources (lambda (ir) (collect_qnodes_by_op (ir_root ir) (quote scan))))
(define ir_output_fields (lambda (ir) (qattr (ir_root ir) (quote output-fields) '())))
(define ir_hidden_fields (lambda (ir) (qattr (ir_root ir) (quote hidden-fields) '())))
(define ir_context_get (lambda (ctx key default) (uctx_attr ctx key default)))

/* ------------------------------------------------------------------------- */
/* Traversal and invariants                                                   */

(define collect_qnodes_by_op (lambda (node op)
	(if (nil? node)
		'()
		(merge
			(if (equal? (qop node) op) (list node) '())
			(merge (map (qchildren node) (lambda (child)
				(collect_qnodes_by_op child op))))))))

(define qnode_has_op? (lambda (node ops)
	(or
		(contains? ops (qop node))
		(reduce (qchildren node) (lambda (found child)
			(or found (qnode_has_op? child ops)))
			false))))

(define inner_select_head? (lambda (sym)
	(or
		(equal? sym (quote inner_select))
		(equal? sym '(quote inner_select))
		(equal? sym '(symbol inner_select))
		(equal? sym (quote inner_select_in))
		(equal? sym '(quote inner_select_in))
		(equal? sym '(symbol inner_select_in))
		(equal? sym (quote inner_select_exists))
		(equal? sym '(quote inner_select_exists))
		(equal? sym '(symbol inner_select_exists)))))

(define expr_contains_subquery? (lambda (expr) (match expr
	(cons sym args)
	(or
		(inner_select_head? sym)
		(expr_contains_subquery? sym)
		(reduce args (lambda (found arg)
			(or found (expr_contains_subquery? arg)))
			false))
	false)))

(define qnode_contains_subquery_expr? (lambda (node)
	(or
		(expr_contains_subquery? (qattrs node))
		(expr_contains_subquery? (qfacts node))
		(reduce (qchildren node) (lambda (found child)
			(or found (qnode_contains_subquery_expr? child)))
			false))))

(define require_unnested_ir (lambda (where ir)
	(if (or
		(qnode_has_op? (ir_root ir) '(depjoin derived))
		(qnode_contains_subquery_expr? (ir_root ir)))
		(error (concat "NEUMANN_INVARIANT_BROKEN: " where " produced non-unnested IR"))
		ir)))

(define neumann_fail (lambda (where detail)
	(error (concat "NEUMANN_REBUILD_UNIMPLEMENTED: " where ": " detail))))

/* ------------------------------------------------------------------------- */
/* Parser AST -> initial operator DAG                                         */

(define ast_scan_node (lambda (alias schema tbl is_outer join_expr)
	(qnode (quote scan) alias
		(list
			(list (quote schema) schema)
			(list (quote table) tbl)
			(list (quote join-predicate) (coalesceNil join_expr true)))
		'()
		(list
			(list (quote aliases) (list alias))
			(list (quote null-preserving) is_outer)
			(list (quote projected-columns) '())
			(list (quote hidden-domain-columns) '())
			(list (quote unique-keys) '())
			(list (quote cardinality) (quote unknown))
			(list (quote lineage) (list schema tbl))))))

(define ast_derived_node (lambda (alias schema subquery is_outer join_expr)
	(qnode (quote derived) alias
		(list
			(list (quote schema) schema)
			(list (quote subquery-ast) subquery)
			(list (quote join-predicate) (coalesceNil join_expr true)))
		'()
		(list
			(list (quote aliases) (list alias))
			(list (quote null-preserving) is_outer)
			(list (quote projected-columns) '())
			(list (quote hidden-domain-columns) '())
			(list (quote unique-keys) '())
			(list (quote cardinality) (quote unknown))
			(list (quote lineage) (list (quote derived) alias))))))

(define ast_table_node (lambda (td) (match td
	'(alias schema (string? tbl) is_outer join_expr)
	(ast_scan_node alias schema tbl is_outer join_expr)
	'(alias schema subquery is_outer join_expr)
	(ast_derived_node alias schema subquery is_outer join_expr)
	_ (neumann_fail "untangle_query" "unknown parser table descriptor"))))

(define join_two_nodes (lambda (left right) (begin
	(define right_attrs (qattrs right))
	(define right_facts (qfacts right))
	(define is_outer (qassoc_get right_facts (quote null-preserving) false))
	(qnode (quote join)
		(concat "join:" (qid left) ":" (qid right))
		(list
			(list (quote join-kind) (if is_outer (quote left) (quote inner)))
			(list (quote predicate) (qassoc_get right_attrs (quote join-predicate) true))
			(list (quote preserves) (if is_outer (list (qid left)) '())))
		(list left right)
		(list
			(list (quote aliases) (merge (qfact left (quote aliases) '()) (qfact right (quote aliases) '())))
			(list (quote hidden-domain-columns) '())
			(list (quote unique-keys) '())
			(list (quote cardinality) (quote unknown)))))))

(define join_table_nodes (lambda (nodes)
	(match nodes
		'() (qnode (quote empty-row) "empty-row" '() '()
			(list
				(list (quote aliases) '())
				(list (quote cardinality) 1)
				(list (quote unique-keys) '())))
		(cons first rest)
		(reduce rest (lambda (left right) (join_two_nodes left right)) first))))

(define attach_select_node (lambda (root predicate)
	(if (or (nil? predicate) (equal? predicate true))
		root
		(qnode (quote select) (concat "select:" (qid root))
			(list (list (quote predicate) predicate))
			(list root)
			(qfacts root)))))

(define attach_group_node (lambda (root group having)
	(if (and (equal? (coalesceNil group '()) '()) (nil? having))
		root
		(qnode (quote group) (concat "group:" (qid root))
			(list
				(list (quote keys) (coalesceNil group '()))
				(list (quote having) having))
			(list root)
			(qassoc_set (qfacts root) (quote cardinality) (quote unknown))))))

(define attach_order_limit_node (lambda (root order limit offset)
	(if (and (equal? (coalesceNil order '()) '()) (nil? limit) (nil? offset))
		root
		(qnode (quote order_limit) (concat "order_limit:" (qid root))
			(list
				(list (quote order) (coalesceNil order '()))
				(list (quote limit) limit)
				(list (quote offset) offset))
			(list root)
			(qfacts root)))))

(define attach_project_node (lambda (root fields hidden_fields)
	(qnode (quote project) (concat "project:" (qid root))
		(list
			(list (quote output-fields) (coalesceNil fields '()))
			(list (quote hidden-fields) (coalesceNil hidden_fields '())))
		(list root)
		(qfacts root))))

(define parser_select_to_initial_dag (lambda (schema tables fields condition group having order limit offset)
	(attach_project_node
		(attach_order_limit_node
			(attach_group_node
				(attach_select_node
					(join_table_nodes (map (coalesceNil tables '()) ast_table_node))
					(coalesceNil condition true))
				group having)
			order limit offset)
		fields '())))

/* ------------------------------------------------------------------------- */
/* untangle_query                                                             */

(define combine_and (lambda (left right)
	(if (or (nil? left) (equal? left true))
		(coalesceNil right true)
		(if (or (nil? right) (equal? right true))
			left
			(list (quote and) left right)))))

(define select_ast_zero_domain? (lambda (subquery) (match subquery
	'(_schema tables _fields _condition group having order _limit _offset)
	(and
		(equal? (coalesceNil tables '()) '())
		(equal? (coalesceNil group '()) '())
		(nil? having)
		(equal? (coalesceNil order '()) '()))
	false)))

(define select_ast_single_expr (lambda (subquery) (match subquery
	'(_schema _tables fields _condition _group _having _order _limit _offset)
	(match fields
		(cons _key (cons expr '())) expr
		_ (neumann_fail "untangle_query" "scalar subquery must project exactly one field"))
	_ (neumann_fail "untangle_query" "malformed scalar subquery"))))

(define select_ast_condition (lambda (subquery) (match subquery
	'(_schema _tables _fields condition _group _having _order _limit _offset)
	(coalesceNil condition true)
	_ (neumann_fail "untangle_query" "malformed scalar subquery"))))

(define select_ast_fields (lambda (subquery) (match subquery
	'(_schema _tables fields _condition _group _having _order _limit _offset)
	fields
	_ (neumann_fail "untangle_query" "malformed derived subquery"))))

(define select_ast_tables (lambda (subquery) (match subquery
	'(_schema tables _fields _condition _group _having _order _limit _offset)
	(coalesceNil tables '())
	_ (neumann_fail "untangle_query" "malformed derived subquery"))))

(define select_ast_simple_from_flattenable? (lambda (subquery) (match subquery
	'(_schema tables _fields condition group having _order limit offset)
	(and
		(or (equal? (coalesceNil tables '()) '()) (equal? (count tables) 1))
		(or (nil? condition) (equal? condition true))
		(equal? (coalesceNil group '()) '())
		(nil? having)
		(nil? limit)
		(nil? offset))
	false)))

(define rewrite_derived_expr (lambda (expr derived_aliases) (match expr
	'((symbol get_column) tbl ti col ci) (begin
		(define fields (if (nil? tbl)
			(if (equal? (count derived_aliases) 1) (cadr (car derived_aliases)) nil)
			(qassoc_get derived_aliases tbl nil)))
		(coalesceNil (get_assoc fields col) expr))
	'((quote get_column) tbl ti col ci) (begin
		(define fields (if (nil? tbl)
			(if (equal? (count derived_aliases) 1) (cadr (car derived_aliases)) nil)
			(qassoc_get derived_aliases tbl nil)))
		(coalesceNil (get_assoc fields col) expr))
	(cons sym args)
	(cons (rewrite_derived_expr sym derived_aliases)
		(map args (lambda (arg) (rewrite_derived_expr arg derived_aliases))))
	expr)))

(define derived_star_fields (lambda (tbl derived_aliases)
	(if (nil? tbl)
		(merge (map derived_aliases (lambda (entry) (match entry
			'(alias fields) (merge (extract_assoc fields (lambda (key expr)
				(list key (rewrite_derived_expr expr derived_aliases)))))
			'()))))
		(begin
			(define fields (qassoc_get derived_aliases tbl nil))
			(if (nil? fields)
				nil
				(merge (extract_assoc fields (lambda (key expr)
					(list key (rewrite_derived_expr expr derived_aliases))))))))))

(define expand_and_rewrite_fields (lambda (fields derived_aliases)
	(merge (extract_assoc (coalesceNil fields '()) (lambda (key expr) (match expr
		'((symbol get_column) tbl _ "*" _) (derived_star_fields tbl derived_aliases)
		'((quote get_column) tbl _ "*" _) (derived_star_fields tbl derived_aliases)
		(list key (rewrite_derived_expr expr derived_aliases))))))))

(define retarget_expr_alias (lambda (expr old_alias new_alias) (match expr
	'((symbol get_column) tbl ti col ci) (if (or (nil? tbl) (equal?? tbl old_alias))
		(list (quote get_column) new_alias false col ci)
		expr)
	'((quote get_column) tbl ti col ci) (if (or (nil? tbl) (equal?? tbl old_alias))
		(list (quote get_column) new_alias false col ci)
		expr)
	(cons sym args)
	(cons (retarget_expr_alias sym old_alias new_alias)
		(map args (lambda (arg) (retarget_expr_alias arg old_alias new_alias))))
	expr)))

(define retarget_fields_alias (lambda (fields old_alias new_alias)
	(map_assoc (coalesceNil fields '()) (lambda (_key expr)
		(retarget_expr_alias expr old_alias new_alias)))))

(define retarget_table_alias (lambda (td new_alias) (match td
	'(old_alias schema tbl is_outer join_expr)
	(list new_alias schema tbl is_outer (retarget_expr_alias join_expr old_alias new_alias))
	_ td)))

(define table_with_join (lambda (td is_outer join_expr) (match td
	'(alias schema tbl _old_outer _old_join)
	(list alias schema tbl is_outer join_expr)
	_ td)))

(define flatten_table_descriptor (lambda (td) (match td
	'(alias schema (string? tbl) is_outer join_expr)
	(list (list td) '())
	'(alias schema subquery is_outer join_expr)
	(if (and
		(select_ast_simple_from_flattenable? subquery))
		(begin
			(define inner_tables (select_ast_tables subquery))
			(define inner_fields_raw (untangle_fields_subqueries (select_ast_fields subquery)))
			(define retargeted (if (equal? (count inner_tables) 1)
				(begin
					(define inner_alias (car (car inner_tables)))
					(list
						(list (retarget_table_alias (car inner_tables) alias))
						(retarget_fields_alias inner_fields_raw inner_alias alias)))
				(list inner_tables inner_fields_raw)))
			(define inner_fields (cadr retargeted))
			(define derived_aliases (list (list alias inner_fields)))
			(define inner_with_join (if (equal? (count (car retargeted)) 1)
				(list (table_with_join (car (car retargeted)) is_outer (rewrite_derived_expr (coalesceNil join_expr true) derived_aliases)))
				(car retargeted)))
			(define flattened_inner (flatten_query_tables inner_with_join))
			(list (car flattened_inner)
				(merge (cadr flattened_inner) (list (list alias inner_fields)))))
		(list (list td) '()))
	_ (neumann_fail "untangle_query" "unknown parser table descriptor"))))

(define flatten_query_tables (lambda (tables)
	(reduce (coalesceNil tables '()) (lambda (acc td)
		(begin
			(define flat (flatten_table_descriptor td))
			(list
				(merge (car acc) (car flat))
				(merge (cadr acc) (cadr flat)))))
		(list '() '()))))

(define untangle_zero_domain_scalar (lambda (subquery)
	(if (select_ast_zero_domain? subquery)
		(begin
			(define condition (untangle_expr_subqueries (select_ast_condition subquery)))
			(define expr (untangle_expr_subqueries (select_ast_single_expr subquery)))
			(if (equal? condition true)
				expr
				(list (quote if) condition expr nil)))
		(neumann_fail "untangle_query" "correlated scalar subquery unnesting not ported yet"))))

(define untangle_zero_domain_exists (lambda (subquery)
	(if (select_ast_zero_domain? subquery)
		(untangle_expr_subqueries (select_ast_condition subquery))
		(neumann_fail "untangle_query" "correlated EXISTS unnesting not ported yet"))))

(define untangle_zero_domain_in (lambda (value subquery)
	(if (select_ast_zero_domain? subquery)
		(begin
			(define condition (untangle_expr_subqueries (select_ast_condition subquery)))
			(define expr (untangle_expr_subqueries (select_ast_single_expr subquery)))
			(list (quote and)
				condition
				(list (quote equal??)
					(untangle_expr_subqueries value)
					expr)))
		(neumann_fail "untangle_query" "correlated IN subquery unnesting not ported yet"))))

(define untangle_expr_subqueries (lambda (expr) (match expr
	(cons sym args)
	(if (inner_select_head? sym)
		(match sym
			(quote inner_select) (match args
				(cons subquery '()) (untangle_zero_domain_scalar subquery)
				_ (neumann_fail "untangle_query" "malformed scalar subquery"))
			(quote inner_select_exists) (match args
				(cons subquery '()) (untangle_zero_domain_exists subquery)
				_ (neumann_fail "untangle_query" "malformed EXISTS subquery"))
			(quote inner_select_in) (match args
				'(value subquery) (untangle_zero_domain_in value subquery)
				_ (neumann_fail "untangle_query" "malformed IN subquery"))
			_ (neumann_fail "untangle_query" "unknown expression subquery form"))
		(cons (untangle_expr_subqueries sym)
			(map args untangle_expr_subqueries)))
	expr)))

(define untangle_fields_subqueries (lambda (fields)
	(map_assoc (coalesceNil fields '()) (lambda (_key expr)
		(untangle_expr_subqueries expr)))))

(define untangle_order_subqueries (lambda (order)
	(map (coalesceNil order '()) (lambda (item) (match item
		'(expr dir) (list (untangle_expr_subqueries expr) dir)
		item)))))

(define untangle_query (lambda (schema tables fields condition group having order limit offset outer_schemas) (begin
	(define ctx (initial_uctx outer_schemas))
	(define flattened (flatten_query_tables tables))
	(define flat_tables (car flattened))
	(define derived_aliases (cadr flattened))
	(define root (parser_select_to_initial_dag schema flat_tables
		(expand_and_rewrite_fields (untangle_fields_subqueries fields) derived_aliases)
		(rewrite_derived_expr (untangle_expr_subqueries (coalesceNil condition true)) derived_aliases)
		(map (coalesceNil group '()) untangle_expr_subqueries)
		(if (nil? having) nil (rewrite_derived_expr (untangle_expr_subqueries having) derived_aliases))
		(untangle_order_subqueries (map (coalesceNil order '()) (lambda (item) (match item
			'(expr dir) (list (rewrite_derived_expr expr derived_aliases) dir)
			item))))
		limit offset))
	(define ir (qir (quote select) schema root (quote rows) ctx '()))
	(require_unnested_ir "untangle_query" ir))))

(define untangle_dml (lambda (kind schema target_table target_alias tables fields condition order limit offset)
	(ir_with_return
		(untangle_query schema tables fields condition nil nil order limit offset nil)
		(list kind target_table target_alias fields))))

(define untangle_query_term (lambda (query outer_schemas) (match query
	'(schema tables fields condition group having order limit offset)
	(untangle_query schema tables fields condition group having order limit offset outer_schemas)
	_ (neumann_fail "untangle_query_term" "query term kind not ported yet"))))

/* ------------------------------------------------------------------------- */
/* reorder                                                                    */

(define join_reorder (lambda (ir)
	(require_unnested_ir "join_reorder" ir)))

/* ------------------------------------------------------------------------- */
/* build_queryplan                                                            */

(define build_resultrow_expr (lambda (fields)
	(list (quote resultrow)
		(cons (quote list)
			(reduce_assoc (coalesceNil fields '()) (lambda (acc key expr)
				(merge acc (list key expr)))
				'())))))

(define dedupe_list (lambda (xs)
	(reduce (coalesceNil xs '()) (lambda (acc item)
		(append_unique acc item))
		'())))

(define scan_expr_columns (lambda (expr alias) (match expr
	'((symbol get_column) tbl _ col _) (if (or (nil? tbl) (equal?? tbl alias)) (list col) '())
	'((quote get_column) tbl _ col _) (if (or (nil? tbl) (equal?? tbl alias)) (list col) '())
	(cons sym args) (dedupe_list (merge (map args (lambda (arg) (scan_expr_columns arg alias)))))
	'())))

(define scan_fields_columns (lambda (fields alias)
	(dedupe_list (merge (extract_assoc (coalesceNil fields '()) (lambda (_key expr)
		(scan_expr_columns expr alias)))))))

(define lower_scan_expr (lambda (expr alias) (match expr
	'((symbol get_column) tbl _ col _) (if (or (nil? tbl) (equal?? tbl alias))
		(symbol (concat alias "." col))
		expr)
	'((quote get_column) tbl _ col _) (if (or (nil? tbl) (equal?? tbl alias))
		(symbol (concat alias "." col))
		expr)
	(cons sym args) (cons sym (map args (lambda (arg) (lower_scan_expr arg alias))))
	expr)))

(define lower_scan_fields (lambda (fields alias)
	(map_assoc (coalesceNil fields '()) (lambda (key expr)
		(lower_scan_expr expr alias)))))

(define lower_expr_for_aliases (lambda (expr aliases) (match expr
	'((symbol get_column) tbl _ col _) (if (nil? tbl)
		(if (equal? (count aliases) 1)
			(symbol (concat (car aliases) "." col))
			expr)
		(if (contains? aliases tbl)
			(symbol (concat tbl "." col))
			expr))
	'((quote get_column) tbl _ col _) (if (nil? tbl)
		(if (equal? (count aliases) 1)
			(symbol (concat (car aliases) "." col))
			expr)
		(if (contains? aliases tbl)
			(symbol (concat tbl "." col))
			expr))
	(cons sym args) (cons sym (map args (lambda (arg) (lower_expr_for_aliases arg aliases))))
	expr)))

(define fields_columns_for_alias (lambda (fields alias)
	(dedupe_list (merge (extract_assoc (coalesceNil fields '()) (lambda (_key expr)
		(scan_expr_columns expr alias)))))))

(define scan_order_columns (lambda (order alias)
	(dedupe_list (merge (map (coalesceNil order '()) (lambda (item) (match item
		'(expr _dir) (scan_expr_columns expr alias)
		_ '())))))))

(define scan_order_dirs (lambda (order)
	(map (coalesceNil order '()) (lambda (item) (match item
		'(_expr dir) dir
		<)))))

(define lower_project_empty_row (lambda (project_node child)
	(match (qop child)
		(quote empty-row) (build_resultrow_expr (qattr project_node (quote output-fields) '()))
		(quote select) (begin
			(define grandchild (car (qchildren child)))
			(if (equal? (qop grandchild) (quote empty-row))
				(list (quote if)
					(qattr child (quote predicate) true)
					(build_resultrow_expr (qattr project_node (quote output-fields) '()))
					nil)
				(neumann_fail "build_queryplan" "project/select lowerer only supports empty-row input yet")))
		_ (neumann_fail "build_queryplan" "project lowerer only supports empty-row input yet"))))

(define lower_project_scan (lambda (project_node scan_node predicate)
	(begin
		(define alias (qid scan_node))
		(define schema (qattr scan_node (quote schema) nil))
		(define tbl (qattr scan_node (quote table) nil))
		(define fields (qattr project_node (quote output-fields) '()))
		(define lowered_predicate (lower_scan_expr (coalesceNil predicate true) alias))
		(define lowered_fields (lower_scan_fields fields alias))
		(define filtercols (scan_expr_columns predicate alias))
		(define mapcols (dedupe_list (merge (list filtercols (scan_fields_columns fields alias)))))
		(define filter_params (map filtercols (lambda (col) (symbol (concat alias "." col)))))
		(define map_params (map mapcols (lambda (col) (symbol (concat alias "." col)))))
		(list (quote scan)
			'(session "__memcp_tx")
			(list (quote table) schema tbl)
			(cons (quote list) filtercols)
			(list (quote lambda) filter_params lowered_predicate)
			(cons (quote list) mapcols)
			(list (quote lambda) map_params (build_resultrow_expr lowered_fields))
			nil nil nil false))))

(define lower_project_scan_order (lambda (project_node order_node scan_node predicate)
	(begin
		(define alias (qid scan_node))
		(define schema (qattr scan_node (quote schema) nil))
		(define tbl (qattr scan_node (quote table) nil))
		(define fields (qattr project_node (quote output-fields) '()))
		(define order (qattr order_node (quote order) '()))
		(define lowered_predicate (lower_scan_expr (coalesceNil predicate true) alias))
		(define lowered_fields (lower_scan_fields fields alias))
		(define filtercols (scan_expr_columns predicate alias))
		(define ordercols (scan_order_columns order alias))
		(define mapcols (dedupe_list (merge (list filtercols ordercols (scan_fields_columns fields alias)))))
		(define filter_params (map filtercols (lambda (col) (symbol (concat alias "." col)))))
		(define map_params (map mapcols (lambda (col) (symbol (concat alias "." col)))))
		(list (quote scan_order)
			'(session "__memcp_tx")
			(list (quote table) schema tbl)
			(cons (quote list) filtercols)
			(list (quote lambda) filter_params lowered_predicate)
			(cons (quote list) ordercols)
			(cons (quote list) (scan_order_dirs order))
			0
			(coalesceNil (qattr order_node (quote offset) nil) 0)
			(coalesceNil (qattr order_node (quote limit) nil) -1)
			(cons (quote list) mapcols)
			(list (quote lambda) map_params (build_resultrow_expr lowered_fields))
			nil nil false))))

(define scan_call (lambda (op scan_node filtercols filter_expr mapcols map_expr is_outer order_node)
	(begin
		(define alias (qid scan_node))
		(define schema (qattr scan_node (quote schema) nil))
		(define tbl (qattr scan_node (quote table) nil))
		(define filter_params (map filtercols (lambda (col) (symbol (concat alias "." col)))))
		(define map_params (map mapcols (lambda (col) (symbol (concat alias "." col)))))
		(if (equal? op (quote scan_order))
			(list (quote scan_order)
				'(session "__memcp_tx")
				(list (quote table) schema tbl)
				(cons (quote list) filtercols)
				(list (quote lambda) filter_params filter_expr)
				(cons (quote list) (scan_order_columns (qattr order_node (quote order) '()) alias))
				(cons (quote list) (scan_order_dirs (qattr order_node (quote order) '())))
				0
				(coalesceNil (qattr order_node (quote offset) nil) 0)
				(coalesceNil (qattr order_node (quote limit) nil) -1)
				(cons (quote list) mapcols)
				(list (quote lambda) map_params map_expr)
				nil nil is_outer)
			(list (quote scan)
				'(session "__memcp_tx")
				(list (quote table) schema tbl)
				(cons (quote list) filtercols)
				(list (quote lambda) filter_params filter_expr)
				(cons (quote list) mapcols)
				(list (quote lambda) map_params map_expr)
				nil nil nil is_outer)))))

(define lower_project_join (lambda (project_node join_node order_node final_predicate)
	(match (qchildren join_node)
		'(left right) (if (and (equal? (qop left) (quote scan)) (equal? (qop right) (quote scan)))
			(begin
				(define left_alias (qid left))
				(define right_alias (qid right))
				(define aliases (list left_alias right_alias))
				(define fields (qattr project_node (quote output-fields) '()))
				(define join_predicate (combine_and
					(qattr join_node (quote predicate) true)
					(coalesceNil final_predicate true)))
				(define order (if (nil? order_node) '() (qattr order_node (quote order) '())))
				(define right_order_cols (scan_order_columns order right_alias))
				(if (not (equal? right_order_cols '()))
					(neumann_fail "build_queryplan" "join order by right side not ported yet")
					(begin
						(define left_cols (dedupe_list (merge (list
							(fields_columns_for_alias fields left_alias)
							(scan_expr_columns join_predicate left_alias)
							(scan_order_columns order left_alias)))))
						(define right_cols (dedupe_list (merge (list
							(fields_columns_for_alias fields right_alias)
							(scan_expr_columns join_predicate right_alias)))))
						(define lowered_predicate (lower_expr_for_aliases join_predicate aliases))
						(define lowered_fields (map_assoc fields (lambda (_key expr)
							(lower_expr_for_aliases expr aliases))))
						(define right_plan (scan_call (quote scan) right right_cols lowered_predicate right_cols
							(build_resultrow_expr lowered_fields)
							(equal? (qattr join_node (quote join-kind) (quote inner)) (quote left))
							nil))
						(scan_call (if (nil? order_node) (quote scan) (quote scan_order))
							left '() true left_cols right_plan false order_node))))
			(neumann_fail "build_queryplan" "join lowerer only supports scan/scan input yet"))
		_ (neumann_fail "build_queryplan" "join expects two children"))))

(define lower_qnode (lambda (node) (match (qop node)
	(quote project) (match (qchildren node)
		(cons child '()) (match (qop child)
			(quote empty-row) (lower_project_empty_row node child)
			(quote select) (match (qchildren child)
				(cons grandchild '()) (match (qop grandchild)
					(quote empty-row) (lower_project_empty_row node child)
					(quote scan) (lower_project_scan node grandchild (qattr child (quote predicate) true))
					_ (neumann_fail "build_queryplan" "select lowerer only supports empty-row or scan input yet"))
				_ (neumann_fail "build_queryplan" "select expects one child"))
			(quote scan) (lower_project_scan node child true)
			(quote order_limit) (match (qchildren child)
				(cons grandchild '()) (match (qop grandchild)
					(quote scan) (lower_project_scan_order node child grandchild true)
					(quote select) (match (qchildren grandchild)
						(cons scan_child '()) (if (equal? (qop scan_child) (quote scan))
							(lower_project_scan_order node child scan_child (qattr grandchild (quote predicate) true))
							(neumann_fail "build_queryplan" "order_limit/select lowerer only supports scan input yet"))
						_ (neumann_fail "build_queryplan" "select expects one child"))
					(quote join) (lower_project_join node grandchild child true)
					_ (neumann_fail "build_queryplan" "order_limit lowerer only supports scan input yet"))
				_ (neumann_fail "build_queryplan" "order_limit expects one child"))
			(quote join) (lower_project_join node child nil true)
			_ (neumann_fail "build_queryplan" "project lowerer only supports empty-row or scan input yet"))
		_ (neumann_fail "build_queryplan" "project expects one child"))
	_ (neumann_fail "build_queryplan" (concat "operator not ported yet: " (qop node))))))

(define build_queryplan (lambda (ir) (begin
	(require_unnested_ir "build_queryplan input" ir)
	(lower_qnode (ir_root ir)))))

(define neumann_compile_pipeline (lambda (ast)
	(build_queryplan
		(join_reorder
			(untangle_query_term ast nil)))))

(define neumann_compile_ir_pipeline (lambda (ir)
	(build_queryplan
		(join_reorder ir))))

/* ------------------------------------------------------------------------- */
/* Parser-facing adapters                                                     */

(define build_queryplan_term (lambda (query)
	(neumann_compile_pipeline query)))

(define build_queryplan_term_with_sink (lambda (query sink_mode)
	(neumann_compile_ir_pipeline
		(ir_with_return (untangle_query_term query nil) sink_mode))))

(define build_queryplan_term_from_logical (lambda (logical_ir)
	(neumann_compile_ir_pipeline logical_ir)))

(define build_queryplan_term_from_logical_with_sink (lambda (logical_ir sink_mode)
	(neumann_compile_ir_pipeline
		(ir_with_return logical_ir sink_mode))))

(define build_dml_plan (lambda (schema tbl tblalias all_defs cols condition order limit offset)
	(neumann_compile_ir_pipeline
		(untangle_dml
			(if (nil? cols) (quote delete) (quote update))
			schema tbl tblalias all_defs
			(if (nil? cols) (list "__dml_row" 1) cols)
			(coalesceNil condition true)
			order limit offset))))

(define sql_truncate (lambda (schema tbl)
	(build_dml_plan schema tbl nil
		(list (list tbl schema tbl false nil))
		nil true nil nil nil)))

(define explain_queryplan_ir (lambda (query)
	(list (quote resultrow)
		(list (quote list)
			"ir"
			(pretty_print (untangle_query_term query nil) (settings "ExplainWidth"))))))

(define explain_queryplan_reorder (lambda (query)
	(list (quote resultrow)
		(list (quote list)
			"reorder"
			(pretty_print
				(join_reorder (untangle_query_term query nil))
				(settings "ExplainWidth"))))))
