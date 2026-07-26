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

(define qnode_with_attr (lambda (node key value)
	(qnode_with_attrs node (qassoc_set (qattrs node) key value))))

(define qnode_with_children (lambda (node children)
	(if (equal? children (qchildren node))
		node
		(qnode (qop node) (qid node) (qattrs node) children (qfacts node)))))

(define qnode_map_children (lambda (node fn)
	(begin
		(define children (qchildren node))
		(define rewritten (map children fn))
		(qnode_with_children node rewritten))))

(define qnode_with_facts (lambda (node facts)
	(if (equal? facts (qfacts node))
		node
		(qnode (qop node) (qid node) (qattrs node) (qchildren node) facts))))

(define qnode_with_fact (lambda (node key value)
	(qnode_with_facts node (qassoc_set (qfacts node) key value))))

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

(define table_aliases (lambda (tables)
	(map (coalesceNil tables '()) (lambda (td) (car td)))))

(define table_has_column? (lambda (td col ci) (match td
	'(_alias schema (string? tbl) _is_outer _join_expr)
	(reduce (show schema tbl) (lambda (found coldef)
		(or found ((if ci equal?? equal?) (coldef "Field") col)))
		false)
	'(_alias _schema subquery _is_outer _join_expr)
	(reduce_assoc (select_ast_fields subquery) (lambda (found key _expr)
		(or found ((if ci equal?? equal?) key col)))
		false)
	false)))

(define unqualified_column_in_tables? (lambda (tables col ci)
	(reduce (coalesceNil tables '()) (lambda (found td)
		(or found (table_has_column? td col ci)))
		false)))

(define expr_has_external_ref? (lambda (expr tables aliases) (match expr
	'((symbol get_column) tbl _ col ci) (if (nil? tbl)
		(not (unqualified_column_in_tables? tables col ci))
		(not (contains? aliases tbl)))
	'((quote get_column) tbl _ col ci) (if (nil? tbl)
		(not (unqualified_column_in_tables? tables col ci))
		(not (contains? aliases tbl)))
	(cons sym args)
	(or
		(expr_has_external_ref? sym tables aliases)
		(reduce args (lambda (found arg)
			(or found (expr_has_external_ref? arg tables aliases)))
			false))
	false)))

(define fields_have_external_ref? (lambda (fields tables aliases)
	(reduce_assoc (coalesceNil fields '()) (lambda (found _key expr)
		(or found (expr_has_external_ref? expr tables aliases)))
		false)))

(define order_has_external_ref? (lambda (order tables aliases)
	(reduce (coalesceNil order '()) (lambda (found item)
		(or found (match item
			'(expr _dir) (expr_has_external_ref? expr tables aliases)
			false)))
		false)))

(define select_ast_uncorrelated? (lambda (subquery)
	(match subquery
		'(_schema tables fields condition group having order _limit _offset) (begin
			(define aliases (table_aliases tables))
			(not (or
				(fields_have_external_ref? fields tables aliases)
				(expr_has_external_ref? condition tables aliases)
				(reduce (coalesceNil group '()) (lambda (found expr)
					(or found (expr_has_external_ref? expr tables aliases)))
					false)
				(expr_has_external_ref? having tables aliases)
				(order_has_external_ref? order tables aliases)))))
		false)))

(define select_ast_simple_from_flattenable? (lambda (subquery) (match subquery
	'(_schema tables _fields condition group having _order limit offset)
	(and
		(or
			(<= (count (coalesceNil tables '())) 1)
			(nil? condition)
			(equal? condition true))
		(equal? (coalesceNil group '()) '())
		(nil? having)
		(nil? limit)
		(nil? offset))
	false)))

(define field_lookup_entry (lambda (fields col ignorecase)
	(begin
		(define exact (reduce_assoc (coalesceNil fields '()) (lambda (found key expr)
			(if (not (nil? found))
				found
				(if (equal? key col) (list (quote found) expr) nil)))
			nil))
		(if (or (not (nil? exact)) (not ignorecase))
			exact
			(reduce_assoc (coalesceNil fields '()) (lambda (found key expr)
				(if (not (nil? found))
					found
					(if (equal?? key col) (list (quote found) expr) nil)))
				nil)))))

(define derived_alias_entry (lambda (alias fields is_outer)
	(list alias fields is_outer (derived_presence_expr fields))))

(define derived_alias_name (lambda (entry) (nth entry 0)))
(define derived_alias_fields (lambda (entry) (nth entry 1)))
(define derived_alias_outer? (lambda (entry) (nth entry 2)))
(define derived_alias_presence (lambda (entry) (nth entry 3)))

(define find_derived_alias (lambda (derived_aliases alias)
	(reduce (coalesceNil derived_aliases '()) (lambda (found entry)
		(if (not (nil? found))
			found
			(if (equal?? (derived_alias_name entry) alias) entry nil)))
		nil)))

(define without_derived_alias (lambda (derived_aliases alias)
	(filter (coalesceNil derived_aliases '()) (lambda (entry)
		(not (equal?? (derived_alias_name entry) alias))))))

(define guard_derived_value (lambda (entry expr)
	(begin
		(define presence (derived_alias_presence entry))
		(if (or (not (derived_alias_outer? entry)) (nil? presence))
			expr
			(list (quote if)
				presence
				expr
				nil)))))

(define rewrite_derived_expr (lambda (expr derived_aliases) (match expr
	'((symbol get_column) tbl ti col ci) (begin
		(define entry (if (nil? tbl)
			(if (equal? (count derived_aliases) 1) (car derived_aliases) nil)
			(find_derived_alias derived_aliases tbl)))
		(if (nil? entry)
			expr
			(begin
				(define hit (field_lookup_entry (derived_alias_fields entry) col ci))
				(if (nil? hit)
					expr
					(guard_derived_value entry
						(rewrite_derived_expr (nth hit 1)
							(without_derived_alias derived_aliases (derived_alias_name entry))))))))
	'((quote get_column) tbl ti col ci) (begin
		(define entry (if (nil? tbl)
			(if (equal? (count derived_aliases) 1) (car derived_aliases) nil)
			(find_derived_alias derived_aliases tbl)))
		(if (nil? entry)
			expr
			(begin
				(define hit (field_lookup_entry (derived_alias_fields entry) col ci))
				(if (nil? hit)
					expr
					(guard_derived_value entry
						(rewrite_derived_expr (nth hit 1)
							(without_derived_alias derived_aliases (derived_alias_name entry))))))))
	(cons sym args)
	(cons (rewrite_derived_expr sym derived_aliases)
		(map args (lambda (arg) (rewrite_derived_expr arg derived_aliases))))
	expr)))

(define derived_star_fields (lambda (tbl derived_aliases)
	(if (nil? tbl)
		(merge (map derived_aliases (lambda (entry) (match entry
			'(alias fields _is_outer _presence) (merge (extract_assoc fields (lambda (key expr)
				(list key (guard_derived_value entry (rewrite_derived_expr expr derived_aliases))))))
			'()))))
		(begin
			(define entry (find_derived_alias derived_aliases tbl))
			(if (nil? entry)
				nil
				(merge (extract_assoc (derived_alias_fields entry) (lambda (key expr)
					(list key (guard_derived_value entry (rewrite_derived_expr expr derived_aliases)))))))))))

(define expand_and_rewrite_fields (lambda (fields derived_aliases)
	(merge (extract_assoc (coalesceNil fields '()) (lambda (key expr) (match expr
		'((symbol get_column) tbl _ "*" _) (derived_star_fields tbl derived_aliases)
		'((quote get_column) tbl _ "*" _) (derived_star_fields tbl derived_aliases)
		(list key (rewrite_derived_expr expr derived_aliases))))))))

(define qir_with_root (lambda (ir root)
	(if (equal? root (ir_root ir))
		ir
		(qir (ir_kind ir) (ir_schema ir) root (ir_return ir) (ir_context_of ir) (ir_facts ir)))))

(define column_in_list? (lambda (cols col ci)
	(reduce (coalesceNil cols '()) (lambda (found candidate)
		(or found ((if ci equal?? equal?) candidate col)))
		false)))

(define table_descriptor_columns (lambda (td)
	(match td
		'(_alias schema (string? tbl) _is_outer _join_expr)
		(map (show schema tbl) (lambda (coldef) (coldef "Field")))
		'(_alias _schema subquery _is_outer _join_expr)
		(extract_assoc (select_ast_fields subquery) (lambda (key _expr) (list key)))
		'())))

(define retarget_qnode_alias (lambda (node old_alias new_alias)
	(begin
		(define local_cols (dedupe_list (merge (map (collect_qnodes_by_op node (quote scan)) scan_schema_columns))))
		(retarget_qnode_alias_with_columns node old_alias new_alias local_cols))))

(define retarget_ir_alias (lambda (ir old_alias new_alias)
	(qir_with_root ir (retarget_qnode_alias (ir_root ir) old_alias new_alias))))

(define retarget_expr_alias_with_columns (lambda (expr old_alias new_alias local_cols) (match expr
	'((symbol get_column) tbl ti col ci) (if (and (not (nil? tbl)) (equal?? tbl old_alias))
		(list (quote get_column) new_alias false col ci)
		(if (and (nil? tbl) (not (column_in_list? local_cols col ci)))
			(list (quote get_column) new_alias false col ci)
			expr))
	'((quote get_column) tbl ti col ci) (if (and (not (nil? tbl)) (equal?? tbl old_alias))
		(list (quote get_column) new_alias false col ci)
		(if (and (nil? tbl) (not (column_in_list? local_cols col ci)))
			(list (quote get_column) new_alias false col ci)
			expr))
	'((symbol neumann_scalar) ir) (list (quote neumann_scalar) (retarget_ir_alias ir old_alias new_alias))
	'((quote neumann_scalar) ir) (list (quote neumann_scalar) (retarget_ir_alias ir old_alias new_alias))
	(cons sym args)
	(cons (retarget_expr_alias_with_columns sym old_alias new_alias local_cols)
		(map args (lambda (arg) (retarget_expr_alias_with_columns arg old_alias new_alias local_cols))))
	expr)))

(define retarget_qattrs_alias_with_columns (lambda (attrs old_alias new_alias local_cols)
	(map (coalesceNil attrs '()) (lambda (entry) (match entry
		'(key value) (list key (retarget_expr_alias_with_columns value old_alias new_alias local_cols))
		_ entry)))))

(define retarget_qnode_alias_with_columns (lambda (node old_alias new_alias local_cols)
	(qnode (qop node)
		(qid node)
		(retarget_qattrs_alias_with_columns (qattrs node) old_alias new_alias local_cols)
		(map (qchildren node) (lambda (child)
			(retarget_qnode_alias_with_columns child old_alias new_alias local_cols)))
		(qfacts node))))

(define retarget_expr_alias_for_tables (lambda (expr tables old_alias new_alias)
	(retarget_expr_alias_with_columns expr old_alias new_alias
		(dedupe_list (merge (map (coalesceNil tables '()) table_descriptor_columns))))))

(define expr_contains_neumann_subplan? (lambda (expr) (match expr
	'((symbol neumann_scalar) _ir) true
	'((quote neumann_scalar) _ir) true
	'((symbol neumann_in) _value _ir) true
	'((quote neumann_in) _value _ir) true
	(cons sym args)
	(or
		(expr_contains_neumann_subplan? sym)
		(reduce args (lambda (found arg)
			(or found (expr_contains_neumann_subplan? arg)))
			false))
	false)))

(define fields_contain_neumann_subplan? (lambda (fields)
	(reduce_assoc (coalesceNil fields '()) (lambda (found _key expr)
		(or found (expr_contains_neumann_subplan? expr)))
		false)))

(define retarget_derived_field_expr_with_columns (lambda (expr old_alias new_alias local_cols) (match expr
	'((symbol get_column) tbl ti col ci) (if (or
		(and (not (nil? tbl)) (equal?? tbl old_alias))
		(and (nil? tbl) (column_in_list? local_cols col ci)))
		(list (quote get_column) new_alias false col ci)
		expr)
	'((quote get_column) tbl ti col ci) (if (or
		(and (not (nil? tbl)) (equal?? tbl old_alias))
		(and (nil? tbl) (column_in_list? local_cols col ci)))
		(list (quote get_column) new_alias false col ci)
		expr)
	'((symbol neumann_scalar) ir) (list (quote neumann_scalar) (retarget_ir_alias ir old_alias new_alias))
	'((quote neumann_scalar) ir) (list (quote neumann_scalar) (retarget_ir_alias ir old_alias new_alias))
	'((symbol neumann_in) value ir) (list (quote neumann_in)
		(retarget_derived_field_expr_with_columns value old_alias new_alias local_cols)
		(retarget_ir_alias ir old_alias new_alias))
	'((quote neumann_in) value ir) (list (quote neumann_in)
		(retarget_derived_field_expr_with_columns value old_alias new_alias local_cols)
		(retarget_ir_alias ir old_alias new_alias))
	(cons sym args)
	(if (or (equal? sym (quote neumann_scalar)) (equal? sym '(quote neumann_scalar)) (equal? sym '(symbol neumann_scalar)))
		(list (quote neumann_scalar) (retarget_ir_alias (car args) old_alias new_alias))
		(if (or (equal? sym (quote neumann_in)) (equal? sym '(quote neumann_in)) (equal? sym '(symbol neumann_in)))
			(list (quote neumann_in)
				(retarget_derived_field_expr_with_columns (car args) old_alias new_alias local_cols)
				(retarget_ir_alias (nth args 1) old_alias new_alias))
			(cons (retarget_derived_field_expr_with_columns sym old_alias new_alias local_cols)
				(map args (lambda (arg) (retarget_derived_field_expr_with_columns arg old_alias new_alias local_cols))))))
	expr)))

(define retarget_fields_alias_for_tables (lambda (fields tables old_alias new_alias)
	(begin
		(define local_cols (dedupe_list (merge (map (coalesceNil tables '()) table_descriptor_columns))))
		(if (fields_contain_neumann_subplan? fields)
			(map_assoc (coalesceNil fields '()) (lambda (_key expr)
				(retarget_expr_alias_for_tables expr tables old_alias new_alias)))
			(map_assoc (coalesceNil fields '()) (lambda (_key expr)
				(retarget_derived_field_expr_with_columns expr old_alias new_alias local_cols)))))))

(define derived_presence_expr (lambda (fields)
	(if (>= (count (coalesceNil fields '())) 2)
		(nth fields 1)
		nil)))

(define null_guard_derived_fields (lambda (fields)
	(begin
		(define presence (derived_presence_expr fields))
		(if (nil? presence)
			fields
			(map_assoc (coalesceNil fields '()) (lambda (_key expr)
				(if (equal? expr presence)
					expr
					(list (quote if)
						(list (quote nil?) presence)
						nil
						expr))))))))

(define retarget_table_alias (lambda (td new_alias) (match td
	'(old_alias schema tbl is_outer join_expr)
	(list new_alias schema tbl is_outer join_expr)
	_ td)))

(define table_with_join (lambda (td is_outer join_expr) (match td
	'(alias schema tbl _old_outer _old_join)
	(list alias schema tbl is_outer join_expr)
	_ td)))

(define rewrite_table_join_expr (lambda (td derived_aliases) (match td
	'(alias schema tbl is_outer join_expr)
	(list alias schema tbl is_outer (rewrite_derived_expr join_expr derived_aliases))
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
						(retarget_fields_alias_for_tables inner_fields_raw inner_tables inner_alias alias)))
				(list inner_tables inner_fields_raw)))
			(define inner_fields (cadr retargeted))
			(define derived_aliases (list (derived_alias_entry alias inner_fields is_outer)))
			(define local_condition (if (equal? (count inner_tables) 1)
				(retarget_expr_alias_for_tables (untangle_expr_subqueries (select_ast_condition subquery)) inner_tables (car (car inner_tables)) alias)
				(untangle_expr_subqueries (select_ast_condition subquery))))
			(define flattened_join_expr (combine_and
				(rewrite_derived_expr (coalesceNil join_expr true) derived_aliases)
				local_condition))
			(define inner_with_join (if (equal? (count (car retargeted)) 1)
				(list (table_with_join (car (car retargeted)) is_outer flattened_join_expr))
				(car retargeted)))
			(define flattened_inner (flatten_query_tables inner_with_join))
			(list (car flattened_inner)
				(merge (cadr flattened_inner) (list (derived_alias_entry alias inner_fields is_outer)))))
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
		(list (quote neumann_scalar) (untangle_query_term subquery nil)))))

(define untangle_zero_domain_exists (lambda (subquery)
	(if (select_ast_zero_domain? subquery)
		(untangle_expr_subqueries (select_ast_condition subquery))
		(list (quote neumann_exists) (untangle_query_term subquery nil)))))

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
		(if (select_ast_uncorrelated? subquery)
			(list (quote neumann_in)
				(untangle_expr_subqueries value)
				(untangle_query_term subquery nil))
			(neumann_fail "untangle_query" "correlated IN subquery unnesting not ported yet")))))

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
	(define derived_union_root (match (coalesceNil tables '())
		(cons td '()) (match td
			'(alias _schema subquery false join_expr) (if (and
				(or (nil? join_expr) (equal? join_expr true))
				(or (nil? condition) (equal? condition true))
				(equal? (coalesceNil group '()) '())
				(nil? having)
				(equal? (coalesceNil order '()) '())
				(nil? limit)
				(nil? offset)
				(or
					(match subquery
						'((symbol union_all) _ _ _ _) true
						'((quote union_all) _ _ _ _) true
						'((symbol union_distinct) _ _ _ _) true
						'((quote union_distinct) _ _ _ _) true
						_ false)))
				(attach_project_node
					(ir_root (untangle_query_term subquery outer_schemas))
					(untangle_fields_subqueries fields)
					'())
				nil)
			_ nil)
		_ nil))
	(define root (if (nil? derived_union_root)
		(begin
			(define flattened (flatten_query_tables tables))
			(define derived_aliases (cadr flattened))
			(define flat_tables (map (car flattened) (lambda (td) (rewrite_table_join_expr td derived_aliases))))
			(parser_select_to_initial_dag schema flat_tables
				(expand_and_rewrite_fields (untangle_fields_subqueries fields) derived_aliases)
				(rewrite_derived_expr (untangle_expr_subqueries (coalesceNil condition true)) derived_aliases)
				(map (coalesceNil group '()) untangle_expr_subqueries)
				(if (nil? having) nil (rewrite_derived_expr (untangle_expr_subqueries having) derived_aliases))
				(untangle_order_subqueries (map (coalesceNil order '()) (lambda (item) (match item
					'(expr dir) (list (rewrite_derived_expr expr derived_aliases) dir)
					item))))
				limit offset))
		derived_union_root))
	(define ir (qir (quote select) schema root (quote rows) ctx '()))
	(require_unnested_ir "untangle_query" ir))))

(define untangle_dml (lambda (kind schema target_table target_alias tables fields condition order limit offset)
	(ir_with_return
		(untangle_query schema tables fields condition nil nil order limit offset nil)
		(list kind target_table target_alias fields))))

(define untangle_union_term (lambda (kind branches order limit offset outer_schemas)
	(begin
		(define child_irs (map (coalesceNil branches '()) (lambda (branch)
			(untangle_query_term branch outer_schemas))))
		(qir (quote select) nil
			(qnode (quote union)
				(concat "union:" (sha1 (string branches)))
				(list
					(list (quote union-kind) kind)
					(list (quote union-label) (if (equal? kind (quote distinct)) (quote union_distinct) (quote union_all)))
					(list (quote branches) (count (coalesceNil branches '())))
					(list (quote order) (coalesceNil order '()))
					(list (quote limit) limit)
					(list (quote offset) offset))
				(map child_irs ir_root)
				(list
					(list (quote aliases) '())
					(list (quote cardinality) (quote unknown))))
			(quote rows)
			(initial_uctx outer_schemas)
			(list (list (quote union-branches) child_irs))))))

(define untangle_query_term (lambda (query outer_schemas) (match query
	'(schema tables fields condition group having order limit offset)
	(untangle_query schema tables fields condition group having order limit offset outer_schemas)
	'((symbol union_all) branches order limit offset)
	(untangle_union_term (quote all) branches order limit offset outer_schemas)
	'((quote union_all) branches order limit offset)
	(untangle_union_term (quote all) branches order limit offset outer_schemas)
	'((symbol union_distinct) branches order limit offset)
	(untangle_union_term (quote distinct) branches order limit offset outer_schemas)
	'((quote union_distinct) branches order limit offset)
	(untangle_union_term (quote distinct) branches order limit offset outer_schemas)
	_ (neumann_fail "untangle_query_term" "query term kind not ported yet"))))

/* ------------------------------------------------------------------------- */
/* reorder                                                                    */

(define join_reorder (lambda (ir)
	(require_unnested_ir "join_reorder" ir)))

/* ------------------------------------------------------------------------- */
/* build_queryplan                                                            */

(define lower_embedded_scalars_with_specs (lambda (expr outer_specs) (match expr
	'((symbol neumann_scalar) ir) (lower_scalar_ir_with_specs ir outer_specs)
	'((quote neumann_scalar) ir) (lower_scalar_ir_with_specs ir outer_specs)
	'((symbol neumann_in) value ir) (lower_in_ir_with_specs value ir outer_specs)
	'((quote neumann_in) value ir) (lower_in_ir_with_specs value ir outer_specs)
	'((symbol neumann_exists) ir) (lower_exists_ir_with_specs ir outer_specs)
	'((quote neumann_exists) ir) (lower_exists_ir_with_specs ir outer_specs)
	(cons sym args) (cons (lower_embedded_scalars_with_specs sym outer_specs)
		(map args (lambda (arg) (lower_embedded_scalars_with_specs arg outer_specs))))
	expr)))

(define lower_embedded_scalars (lambda (expr)
	(lower_embedded_scalars_with_specs expr '())))

(define build_resultrow_expr (lambda (fields)
	(list (quote resultrow)
		(build_row_assoc_expr fields))))

(define build_row_assoc_expr (lambda (fields)
	(cons (quote list)
		(reduce_assoc (coalesceNil fields '()) (lambda (acc key expr)
			(merge acc (list key (lower_embedded_scalars expr))))
			'()))))

(define aggregate_expr? (lambda (expr) (match expr
	'((symbol aggregate) _ _ _) true
	'((quote aggregate) _ _ _) true
	'((symbol count_distinct) _) true
	'((quote count_distinct) _) true
	_ false)))

(define collect_aggregates (lambda (expr) (match expr
	(cons sym args) (if (aggregate_expr? expr)
		(list expr)
		(dedupe_list (merge (map args collect_aggregates))))
	'())))

(define collect_field_aggregates (lambda (fields)
	(dedupe_list (merge (extract_assoc (coalesceNil fields '()) (lambda (_key expr)
		(collect_aggregates expr)))))))

(define has_aggregates? (lambda (fields)
	(not (equal? (collect_field_aggregates fields) '()))))

(define aggregate_input_expr (lambda (agg) (match agg
	'((symbol aggregate) value _reducer _neutral) value
	'((quote aggregate) value _reducer _neutral) value
	'((symbol count_distinct) value) value
	'((quote count_distinct) value) value
	_ (neumann_fail "build_queryplan" "malformed aggregate expression"))))

(define aggregate_neutral_expr (lambda (agg) (match agg
	'((symbol aggregate) _value _reducer neutral) neutral
	'((quote aggregate) _value _reducer neutral) neutral
	'((symbol count_distinct) _value) '()
	'((quote count_distinct) _value) '()
	_ (neumann_fail "build_queryplan" "malformed aggregate expression"))))

(define aggregate_reducer_expr (lambda (agg acc_expr val_expr) (match agg
	'((symbol aggregate) _value reducer _neutral) (list reducer acc_expr val_expr)
	'((quote aggregate) _value reducer _neutral) (list reducer acc_expr val_expr)
	'((symbol count_distinct) _value) (list (quote append_unique) acc_expr val_expr)
	'((quote count_distinct) _value) (list (quote append_unique) acc_expr val_expr)
	_ (neumann_fail "build_queryplan" "malformed aggregate expression"))))

(define aggregate_index (lambda (aggs target idx)
	(match aggs
		(cons agg rest) (if (equal? agg target)
			idx
			(aggregate_index rest target (+ idx 1)))
		'() nil)))

(define replace_aggregate_refs (lambda (expr aggs agg_sym) (match expr
	(cons sym args) (if (aggregate_expr? expr)
		(begin
			(define idx (aggregate_index aggs expr 0))
			(if (nil? idx)
				expr
				(if (or (equal? sym (quote count_distinct)) (equal? sym '(quote count_distinct)) (equal? sym '(symbol count_distinct)))
					(list (quote count) (list (quote nth) agg_sym idx))
					(list (quote nth) agg_sym idx))))
		(cons (replace_aggregate_refs sym aggs agg_sym)
			(map args (lambda (arg) (replace_aggregate_refs arg aggs agg_sym)))))
	expr)))

(define replace_field_aggregates (lambda (fields aggs agg_sym)
	(map_assoc (coalesceNil fields '()) (lambda (_key expr)
		(replace_aggregate_refs expr aggs agg_sym)))))

(define same_get_column_ref? (lambda (left right) (match left
	'((symbol get_column) ltbl _ lcol lci) (match right
		'((symbol get_column) rtbl _ rcol rci) (and
			(or (equal? ltbl rtbl) (and (nil? ltbl) (nil? rtbl)))
			((if (or lci rci) equal?? equal?) lcol rcol))
		'((quote get_column) rtbl _ rcol rci) (and
			(or (equal? ltbl rtbl) (and (nil? ltbl) (nil? rtbl)))
			((if (or lci rci) equal?? equal?) lcol rcol))
		false)
	'((quote get_column) ltbl _ lcol lci) (match right
		'((symbol get_column) rtbl _ rcol rci) (and
			(or (equal? ltbl rtbl) (and (nil? ltbl) (nil? rtbl)))
			((if (or lci rci) equal?? equal?) lcol rcol))
		'((quote get_column) rtbl _ rcol rci) (and
			(or (equal? ltbl rtbl) (and (nil? ltbl) (nil? rtbl)))
			((if (or lci rci) equal?? equal?) lcol rcol))
		false)
	false)))

(define group_expr_equal? (lambda (left right)
	(or (equal? left right) (same_get_column_ref? left right))))

(define group_index (lambda (groups target idx)
	(match groups
		(cons group rest) (if (group_expr_equal? group target)
			idx
			(group_index rest target (+ idx 1)))
		'() nil)))

(define replace_group_refs (lambda (expr groups aggs key_sym agg_sym) (match expr
	(cons sym args) (if (aggregate_expr? expr)
		(begin
			(define agg_idx (aggregate_index aggs expr 0))
			(if (nil? agg_idx) expr (list (quote nth) agg_sym agg_idx)))
		(begin
			(define group_idx (group_index groups expr 0))
			(if (nil? group_idx)
				(cons (replace_group_refs sym groups aggs key_sym agg_sym)
					(map args (lambda (arg) (replace_group_refs arg groups aggs key_sym agg_sym))))
				(list (quote nth) key_sym group_idx))))
	(begin
		(define group_idx (group_index groups expr 0))
		(if (nil? group_idx) expr (list (quote nth) key_sym group_idx))))))

(define replace_group_fields (lambda (fields groups aggs key_sym agg_sym)
	(map_assoc (coalesceNil fields '()) (lambda (_key expr)
		(replace_group_refs expr groups aggs key_sym agg_sym)))))

(define field_key_for_expr (lambda (fields expr)
	(reduce_assoc (coalesceNil fields '()) (lambda (found key field_expr)
		(if (not (nil? found))
			found
			(if (equal? field_expr expr) key nil)))
		nil)))

(define group_order_key (lambda (fields order_item)
	(match order_item
		'(expr _dir) (field_key_for_expr fields expr)
		_ nil)))

(define group_sort_rows_expr (lambda (rows_sym fields order)
	(match (coalesceNil order '())
		(cons first _rest) (begin
			(define key (group_order_key fields first))
			(match first
				'(_expr dir) (if (nil? key)
					(neumann_fail "build_queryplan" "GROUP BY ORDER expression must be projected")
					(list (quote sort) rows_sym
						(list (quote lambda) (list (quote a) (quote b))
							(list dir
								(list (quote get_assoc) (quote a) key)
								(list (quote get_assoc) (quote b) key)))))
				_ rows_sym))
		'() rows_sym)))

(define group_limit_rows_expr (lambda (rows_sym limit offset)
	(begin
		(define start (coalesceNil offset 0))
		(if (nil? limit)
			(if (nil? offset)
				rows_sym
				(list (quote slice) rows_sym start (list (quote count) rows_sym)))
			(list (quote slice) rows_sym start (list (quote +) start limit))))))

(define dedupe_list (lambda (xs)
	(reduce (coalesceNil xs '()) (lambda (acc item)
		(append_unique acc item))
	'())))

(define scan_schema_columns (lambda (scan_node)
	(coalesceNil
		(qfact scan_node (quote schema-columns) nil)
		(map (show (qattr scan_node (quote schema) nil) (qattr scan_node (quote table) nil))
			(lambda (coldef) (coldef "Field"))))))

(define canonical_scan_col (lambda (scan_node col ignorecase)
	(if (or (nil? col) (not ignorecase))
		col
		(coalesceNil
			(reduce (scan_schema_columns scan_node) (lambda (found candidate)
				(if (not (nil? found))
					found
					(if (equal?? candidate col) candidate nil)))
				nil)
			col))))

(define expr_outer_columns_for_scan (lambda (expr scan_node) (match expr
	'((symbol get_column) tbl _ col ci) (if (and (not (nil? tbl)) (equal?? tbl (qid scan_node)))
		(list (canonical_scan_col scan_node col ci))
		'())
	'((quote get_column) tbl _ col ci) (if (and (not (nil? tbl)) (equal?? tbl (qid scan_node)))
		(list (canonical_scan_col scan_node col ci))
		'())
	'((symbol neumann_scalar) ir) (ir_outer_columns_for_scan ir scan_node)
	'((quote neumann_scalar) ir) (ir_outer_columns_for_scan ir scan_node)
	'((symbol neumann_in) value ir) (dedupe_list (merge (list
		(expr_outer_columns_for_scan value scan_node)
		(ir_outer_columns_for_scan ir scan_node))))
	'((quote neumann_in) value ir) (dedupe_list (merge (list
		(expr_outer_columns_for_scan value scan_node)
		(ir_outer_columns_for_scan ir scan_node))))
	'((symbol neumann_exists) ir) (ir_outer_columns_for_scan ir scan_node)
	'((quote neumann_exists) ir) (ir_outer_columns_for_scan ir scan_node)
	(cons sym args) (dedupe_list (merge
		(list
			(expr_outer_columns_for_scan sym scan_node)
			(dedupe_list (merge (map args (lambda (arg)
				(expr_outer_columns_for_scan arg scan_node))))))))
	'())))

(define assoc_outer_columns_for_scan (lambda (xs scan_node)
	(dedupe_list (merge (extract_assoc (coalesceNil xs '()) (lambda (_key expr)
		(expr_outer_columns_for_scan expr scan_node)))))))

(define qnode_outer_columns_for_scan (lambda (node scan_node)
	(dedupe_list (merge (list
		(expr_outer_columns_for_scan (qattrs node) scan_node)
		(dedupe_list (merge (map (qchildren node) (lambda (child)
			(qnode_outer_columns_for_scan child scan_node))))))))))

(define ir_outer_columns_for_scan (lambda (ir scan_node)
	(qnode_outer_columns_for_scan (ir_root ir) scan_node)))

(define scan_expr_columns (lambda (expr scan_node) (match expr
	'((symbol get_column) tbl _ col ci) (if (or
		(equal?? tbl (qid scan_node))
		(and (nil? tbl) (column_in_list? (scan_schema_columns scan_node) col ci)))
		(list (canonical_scan_col scan_node col ci))
		'())
	'((quote get_column) tbl _ col ci) (if (or
		(equal?? tbl (qid scan_node))
		(and (nil? tbl) (column_in_list? (scan_schema_columns scan_node) col ci)))
		(list (canonical_scan_col scan_node col ci))
		'())
	'((symbol neumann_scalar) ir) (ir_outer_columns_for_scan ir scan_node)
	'((quote neumann_scalar) ir) (ir_outer_columns_for_scan ir scan_node)
	'((symbol neumann_in) value ir) (dedupe_list (merge (list
		(scan_expr_columns value scan_node)
		(ir_outer_columns_for_scan ir scan_node))))
	'((quote neumann_in) value ir) (dedupe_list (merge (list
		(scan_expr_columns value scan_node)
		(ir_outer_columns_for_scan ir scan_node))))
	'((symbol neumann_exists) ir) (ir_outer_columns_for_scan ir scan_node)
	'((quote neumann_exists) ir) (ir_outer_columns_for_scan ir scan_node)
	(cons sym args) (dedupe_list (merge (map args (lambda (arg) (scan_expr_columns arg scan_node)))))
	'())))

(define scan_fields_columns (lambda (fields scan_node)
	(dedupe_list (merge (extract_assoc (coalesceNil fields '()) (lambda (_key expr)
		(scan_expr_columns expr scan_node)))))))

(define lower_scan_expr (lambda (expr scan_node) (match expr
	'((symbol get_column) tbl _ col ci) (if (or
		(equal?? tbl (qid scan_node))
		(and (nil? tbl) (column_in_list? (scan_schema_columns scan_node) col ci)))
		(symbol (concat (qid scan_node) "." (canonical_scan_col scan_node col ci)))
		expr)
	'((quote get_column) tbl _ col ci) (if (or
		(equal?? tbl (qid scan_node))
		(and (nil? tbl) (column_in_list? (scan_schema_columns scan_node) col ci)))
		(symbol (concat (qid scan_node) "." (canonical_scan_col scan_node col ci)))
		expr)
	'((symbol neumann_scalar) _ir) expr
	'((quote neumann_scalar) _ir) expr
	'((symbol neumann_in) value ir) (list (quote neumann_in) (lower_scan_expr value scan_node) ir)
	'((quote neumann_in) value ir) (list (quote neumann_in) (lower_scan_expr value scan_node) ir)
	'((symbol neumann_exists) _ir) expr
	'((quote neumann_exists) _ir) expr
	(cons sym args) (cons sym (map args (lambda (arg) (lower_scan_expr arg scan_node))))
	expr)))

(define lower_scan_fields (lambda (fields scan_node)
	(map_assoc (coalesceNil fields '()) (lambda (key expr)
		(lower_scan_expr expr scan_node)))))

(define specs_find_node (lambda (specs alias)
	(reduce specs (lambda (found spec)
		(if (not (nil? found))
			found
			(if (equal?? (qid (spec_node spec)) alias) (spec_node spec) nil)))
		nil)))

(define lower_expr_for_specs (lambda (expr specs) (match expr
	'((symbol get_column) tbl _ col ci) (begin
		(define aliases (specs_aliases specs))
		(define target_alias (if (nil? tbl) (car aliases) tbl))
		(define target_node (specs_find_node specs target_alias))
		(if (nil? target_node)
			expr
			(symbol (concat (qid target_node) "." (canonical_scan_col target_node col ci)))))
	'((quote get_column) tbl _ col ci) (begin
		(define aliases (specs_aliases specs))
		(define target_alias (if (nil? tbl) (car aliases) tbl))
		(define target_node (specs_find_node specs target_alias))
		(if (nil? target_node)
			expr
			(symbol (concat (qid target_node) "." (canonical_scan_col target_node col ci)))))
	'((symbol neumann_scalar) _ir) (lower_embedded_scalars_with_specs expr specs)
	'((quote neumann_scalar) _ir) (lower_embedded_scalars_with_specs expr specs)
	'((symbol neumann_in) _value _ir) (lower_embedded_scalars_with_specs expr specs)
	'((quote neumann_in) _value _ir) (lower_embedded_scalars_with_specs expr specs)
	'((symbol neumann_exists) _ir) (lower_embedded_scalars_with_specs expr specs)
	'((quote neumann_exists) _ir) (lower_embedded_scalars_with_specs expr specs)
	(cons sym args) (cons sym (map args (lambda (arg) (lower_expr_for_specs arg specs))))
	expr)))

(define get_column_columns_for_spec (lambda (tbl col ci spec specs)
	(begin
		(define alias (qid (spec_node spec)))
		(define aliases (specs_aliases specs))
		(if (nil? tbl)
			(if (equal? alias (car aliases)) (list (canonical_scan_col (spec_node spec) col ci)) '())
			(if (equal?? tbl alias) (list (canonical_scan_col (spec_node spec) col ci)) '())))))

(define expr_columns_for_spec (lambda (expr spec specs) (match expr
	'((symbol get_column) tbl _ col ci) (get_column_columns_for_spec tbl col ci spec specs)
	'((quote get_column) tbl _ col ci) (get_column_columns_for_spec tbl col ci spec specs)
	'((symbol neumann_scalar) ir) (ir_outer_columns_for_scan ir (spec_node spec))
	'((quote neumann_scalar) ir) (ir_outer_columns_for_scan ir (spec_node spec))
	'((symbol neumann_in) value ir) (dedupe_list (merge (list
		(expr_columns_for_spec value spec specs)
		(ir_outer_columns_for_scan ir (spec_node spec)))))
	'((quote neumann_in) value ir) (dedupe_list (merge (list
		(expr_columns_for_spec value spec specs)
		(ir_outer_columns_for_scan ir (spec_node spec)))))
	'((symbol neumann_exists) ir) (ir_outer_columns_for_scan ir (spec_node spec))
	'((quote neumann_exists) ir) (ir_outer_columns_for_scan ir (spec_node spec))
	(cons sym args) (dedupe_list (merge (map args (lambda (arg) (expr_columns_for_spec arg spec specs)))))
	'())))

(define fields_columns_for_spec_in_join (lambda (fields spec specs)
	(dedupe_list (merge (extract_assoc (coalesceNil fields '()) (lambda (_key expr)
		(expr_columns_for_spec expr spec specs)))))))

(define scan_order_columns (lambda (order scan_node)
	(dedupe_list (merge (map (coalesceNil order '()) (lambda (item) (match item
		'(expr _dir) (scan_expr_columns expr scan_node)
		_ '())))))))

(define scan_order_columns_for_spec (lambda (order spec specs)
	(dedupe_list (merge (map (coalesceNil order '()) (lambda (item) (match item
		'(expr _dir) (expr_columns_for_spec expr spec specs)
		_ '())))))))

(define scan_order_dirs (lambda (order)
	(map (coalesceNil order '()) (lambda (item) (match item
		'(_expr dir) dir
		<)))))

(define scan_effective_predicate (lambda (scan_node predicate)
	(combine_and
		(qattr scan_node (quote join-predicate) true)
		(coalesceNil predicate true))))

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
		(define outer_specs (list (list scan_node false true)))
		(define effective_predicate (scan_effective_predicate scan_node predicate))
		(define lowered_predicate (lower_embedded_scalars_with_specs (lower_scan_expr effective_predicate scan_node) outer_specs))
		(define lowered_fields (map_assoc (lower_scan_fields fields scan_node) (lambda (_key expr)
			(lower_embedded_scalars_with_specs expr outer_specs))))
		(define filtercols (scan_expr_columns effective_predicate scan_node))
		(define mapcols (dedupe_list (merge (list filtercols (scan_fields_columns fields scan_node)))))
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
		(define outer_specs (list (list scan_node false true)))
		(define effective_predicate (scan_effective_predicate scan_node predicate))
		(define lowered_predicate (lower_embedded_scalars_with_specs (lower_scan_expr effective_predicate scan_node) outer_specs))
		(define lowered_fields (map_assoc (lower_scan_fields fields scan_node) (lambda (_key expr)
			(lower_embedded_scalars_with_specs expr outer_specs))))
		(define filtercols (scan_expr_columns effective_predicate scan_node))
		(define ordercols (scan_order_columns order scan_node))
		(define mapcols (dedupe_list (merge (list filtercols ordercols (scan_fields_columns fields scan_node)))))
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

(define lower_project_global_aggregate_scan (lambda (project_node scan_node predicate)
	(begin
		(define alias (qid scan_node))
		(define schema (qattr scan_node (quote schema) nil))
		(define tbl (qattr scan_node (quote table) nil))
		(define fields (qattr project_node (quote output-fields) '()))
		(define aggs (collect_field_aggregates fields))
		(define inputs (map aggs aggregate_input_expr))
		(define outer_specs (list (list scan_node false true)))
		(define effective_predicate (scan_effective_predicate scan_node predicate))
		(define lowered_predicate (lower_embedded_scalars_with_specs (lower_scan_expr effective_predicate scan_node) outer_specs))
		(define lowered_inputs (map inputs (lambda (expr) (lower_embedded_scalars_with_specs (lower_scan_expr expr scan_node) outer_specs))))
		(define filtercols (scan_expr_columns effective_predicate scan_node))
		(define mapcols (dedupe_list (merge (list filtercols (dedupe_list (merge (map inputs (lambda (expr) (scan_expr_columns expr scan_node)))))))))
		(define filter_params (map filtercols (lambda (col) (symbol (concat alias "." col)))))
		(define map_params (map mapcols (lambda (col) (symbol (concat alias "." col)))))
		(define agg_sym (symbol "__neumann_agg"))
		(list (quote begin)
			(list (quote define) agg_sym
				(list (quote scan)
					'(session "__memcp_tx")
					(list (quote table) schema tbl)
					(cons (quote list) filtercols)
					(list (quote lambda) filter_params lowered_predicate)
					(cons (quote list) mapcols)
					(list (quote lambda) map_params (cons (quote list) lowered_inputs))
					(list (quote lambda) (list (quote acc) (quote rowvals))
						(cons (quote list) (map (produceN (count aggs)) (lambda (i)
							(aggregate_reducer_expr (nth aggs i)
								(list (quote nth) (quote acc) i)
								(list (quote nth) (quote rowvals) i))))))
					(cons (quote list) (map aggs aggregate_neutral_expr))
					nil false))
			(build_resultrow_expr (replace_field_aggregates fields aggs agg_sym))))))

(define scalar_single_expr (lambda (project_node)
	(match (qattr project_node (quote output-fields) '())
		(cons _key (cons expr '())) expr
		_ (neumann_fail "build_queryplan" "scalar subquery must project exactly one field"))))

(define scalar_from_rows_expr (lambda (rows_sym strict)
	(list (quote if)
		(list (quote equal?) (list (quote count) rows_sym) 0)
		nil
		(if strict
			(list (quote if)
				(list (quote equal?) (list (quote count) rows_sym) 1)
				(list (quote car) rows_sym)
				(list (quote error) "Subquery returns more than 1 row"))
			(list (quote car) rows_sym)))))

(define ir_single_output_key (lambda (ir)
	(match (qattr (ir_root ir) (quote output-fields) '())
		(cons key (cons _expr '())) key
		_ (neumann_fail "build_queryplan" "IN subquery must project exactly one field"))))

(define lower_in_scan_values (lambda (project_node scan_node predicate)
	(begin
		(define alias (qid scan_node))
		(define schema (qattr scan_node (quote schema) nil))
		(define tbl (qattr scan_node (quote table) nil))
		(define value_expr (scalar_single_expr project_node))
		(define lowered_predicate (lower_scan_expr (coalesceNil predicate true) scan_node))
		(define lowered_value (lower_scan_expr value_expr scan_node))
		(define filtercols (scan_expr_columns predicate scan_node))
		(define mapcols (dedupe_list (merge (list filtercols (scan_expr_columns value_expr scan_node)))))
		(define filter_params (map filtercols (lambda (col) (symbol (concat alias "." col)))))
		(define map_params (map mapcols (lambda (col) (symbol (concat alias "." col)))))
		(list (quote scan)
			'(session "__memcp_tx")
			(list (quote table) schema tbl)
			(cons (quote list) filtercols)
			(list (quote lambda) filter_params lowered_predicate)
			(cons (quote list) mapcols)
			(list (quote lambda) map_params lowered_value)
			(list (quote lambda) (list (quote acc) (quote value))
				(list (quote append_unique) (quote acc) (quote value)))
			'()
			(list (quote lambda) (list (quote acc) (quote shard_rows))
				(list (quote merge) (quote acc) (quote shard_rows)))
			false))))

(define lower_in_project_values (lambda (project_node)
	(match (qchildren project_node)
		(cons child '()) (match (qop child)
			(quote scan) (lower_in_scan_values project_node child true)
			(quote select) (match (qchildren child)
				(cons grandchild '()) (if (equal? (qop grandchild) (quote scan))
					(lower_in_scan_values project_node grandchild (qattr child (quote predicate) true))
					(neumann_fail "build_queryplan" "IN/select lowerer only supports scan input yet"))
				_ (neumann_fail "build_queryplan" "select expects one child"))
			_ (neumann_fail "build_queryplan" "IN lowerer only supports scan input yet"))
		_ (neumann_fail "build_queryplan" "IN project expects one child"))))

(define lower_in_project_or_join_with_specs (lambda (value project_node outer_specs)
	(match (qchildren project_node)
		(cons child '()) (match (qop child)
			(quote scan) (list (quote contains?)
				(lower_in_project_values project_node)
				(lower_expr_for_specs value outer_specs))
			(quote select) (match (qchildren child)
				(cons grandchild '()) (match (qop grandchild)
					(quote scan) (list (quote contains?)
						(lower_in_project_values project_node)
						(lower_expr_for_specs value outer_specs))
					(quote join) (lower_exists_join grandchild
						(combine_and
							(qattr child (quote predicate) true)
							(list (quote equal??) (scalar_single_expr project_node) value))
						outer_specs)
					_ (neumann_fail "build_queryplan" "IN/select lowerer only supports scan or join input yet"))
				_ (neumann_fail "build_queryplan" "select expects one child"))
			(quote join) (lower_exists_join child
				(list (quote equal??) (scalar_single_expr project_node) value)
				outer_specs)
			_ (neumann_fail "build_queryplan" "IN lowerer only supports scan or join input yet"))
		_ (neumann_fail "build_queryplan" "IN project expects one child"))))

(define lower_in_ir_with_specs (lambda (value ir outer_specs)
	(begin
		(require_unnested_ir "build_queryplan IN subquery" ir)
		(match (qop (ir_root ir))
			(quote project) (lower_in_project_or_join_with_specs value (ir_root ir) outer_specs)
			_ (neumann_fail "build_queryplan" "IN IR root must be project")))))

(define lower_exists_scan (lambda (scan_node predicate outer_specs)
	(begin
		(define alias (qid scan_node))
		(define schema (qattr scan_node (quote schema) nil))
		(define tbl (qattr scan_node (quote table) nil))
		(define lowered_predicate (lower_embedded_scalars_with_specs
			(lower_expr_for_specs (lower_scan_expr (coalesceNil predicate true) scan_node) outer_specs)
			outer_specs))
		(define filtercols (scan_expr_columns predicate scan_node))
		(define filter_params (map filtercols (lambda (col) (symbol (concat alias "." col)))))
		(list (quote scan)
			'(session "__memcp_tx")
			(list (quote table) schema tbl)
			(cons (quote list) filtercols)
			(list (quote lambda) filter_params lowered_predicate)
			(cons (quote list) filtercols)
			(list (quote lambda) filter_params true)
			(list (quote lambda) (list (quote acc) (quote value))
				(list (quote or) (quote acc) (quote value)))
			false
			(list (quote lambda) (list (quote acc) (quote shard_value))
				(list (quote or) (quote acc) (quote shard_value)))
			false))))

(define lower_exists_project (lambda (project_node outer_specs)
	(match (qchildren project_node)
		(cons child '()) (match (qop child)
			(quote scan) (lower_exists_scan child true outer_specs)
			(quote select) (match (qchildren child)
				(cons grandchild '()) (match (qop grandchild)
					(quote scan) (lower_exists_scan grandchild (qattr child (quote predicate) true) outer_specs)
					(quote join) (lower_exists_join grandchild (qattr child (quote predicate) true) outer_specs)
					_ (neumann_fail "build_queryplan" "EXISTS/select lowerer only supports scan input yet"))
				_ (neumann_fail "build_queryplan" "select expects one child"))
			(quote join) (lower_exists_join child true outer_specs)
			_ (neumann_fail "build_queryplan" "EXISTS lowerer only supports scan input yet"))
		_ (neumann_fail "build_queryplan" "EXISTS project expects one child"))))

(define lower_scan_specs_exists (lambda (specs all_specs final_predicate outer_specs prefix_aliases)
	(match specs
		(cons spec rest) (begin
			(define node (spec_node spec))
			(define alias (qid node))
			(define aliases_now (merge prefix_aliases (list alias)))
			(define predicates (merge (specs_predicates all_specs) (list (coalesceNil final_predicate true))))
			(define mapcols (dedupe_list (merge (map predicates (lambda (predicate)
				(expr_columns_for_spec predicate spec all_specs))))))
			(define filter_predicate (if (equal? rest '())
				(combine_and (spec_predicate spec) (coalesceNil final_predicate true))
				(spec_predicate spec)))
			(define visible_specs (merge
				(filter all_specs (lambda (s) (contains? aliases_now (qid (spec_node s)))))
				outer_specs))
			(define lowered_filter (lower_embedded_scalars_with_specs
				(lower_expr_for_specs filter_predicate visible_specs)
				outer_specs))
			(define map_params (map mapcols (lambda (col) (symbol (concat alias "." col)))))
			(define map_expr (if (equal? rest '())
				true
				(lower_scan_specs_exists rest all_specs final_predicate outer_specs aliases_now)))
			(list (quote scan)
				'(session "__memcp_tx")
				(list (quote table) (qattr node (quote schema) nil) (qattr node (quote table) nil))
				(cons (quote list) (expr_columns_for_spec filter_predicate spec all_specs))
				(list (quote lambda) (map (expr_columns_for_spec filter_predicate spec all_specs) (lambda (col) (symbol (concat alias "." col)))) lowered_filter)
				(cons (quote list) mapcols)
				(list (quote lambda) map_params map_expr)
				(list (quote lambda) (list (quote acc) (quote value))
					(list (quote or) (quote acc) (quote value)))
				false
				(list (quote lambda) (list (quote acc) (quote shard_value))
					(list (quote or) (quote acc) (quote shard_value)))
				(spec_outer? spec)))
		'() (neumann_fail "build_queryplan" "empty EXISTS join scan sequence"))))

(define lower_exists_join (lambda (join_node final_predicate outer_specs)
	(begin
		(define specs (join_scan_specs join_node))
		(lower_scan_specs_exists specs specs final_predicate outer_specs '()))))

(define lower_exists_union (lambda (union_node outer_specs)
	(reduce (qchildren union_node) (lambda (acc branch)
		(list (quote or)
			acc
			(match (qop branch)
				(quote project) (lower_exists_project branch outer_specs)
				_ (neumann_fail "build_queryplan" "EXISTS union branch must be project"))))
		false)))

(define lower_exists_ir_with_specs (lambda (ir outer_specs)
	(begin
		(require_unnested_ir "build_queryplan EXISTS subquery" ir)
		(match (qop (ir_root ir))
			(quote project) (lower_exists_project (ir_root ir) outer_specs)
			(quote union) (lower_exists_union (ir_root ir) outer_specs)
			_ (neumann_fail "build_queryplan" "EXISTS IR root must be project or union")))))

(define lower_scalar_scan (lambda (project_node scan_node predicate order_node outer_specs)
	(begin
		(define alias (qid scan_node))
		(define schema (qattr scan_node (quote schema) nil))
		(define tbl (qattr scan_node (quote table) nil))
		(define value_expr (scalar_single_expr project_node))
		(define effective_predicate (scan_effective_predicate scan_node predicate))
		(define lowered_predicate (lower_embedded_scalars_with_specs
			(lower_expr_for_specs (lower_scan_expr effective_predicate scan_node) outer_specs)
			outer_specs))
		(define lowered_value (lower_embedded_scalars_with_specs
			(lower_expr_for_specs (lower_scan_expr value_expr scan_node) outer_specs)
			outer_specs))
		(define filtercols (scan_expr_columns effective_predicate scan_node))
		(define ordercols (if (nil? order_node) '() (scan_order_columns (qattr order_node (quote order) '()) scan_node)))
		(define mapcols (dedupe_list (merge (list filtercols ordercols (scan_expr_columns value_expr scan_node)))))
		(define filter_params (map filtercols (lambda (col) (symbol (concat alias "." col)))))
		(define map_params (map mapcols (lambda (col) (symbol (concat alias "." col)))))
		(define rows_sym (symbol (concat "__neumann_scalar_rows_" (sha1 (string project_node)))))
		(define strict (or (nil? order_node) (nil? (qattr order_node (quote limit) nil)) (not (equal? (qattr order_node (quote limit) nil) 1))))
		(list (quote begin)
			(list (quote define) rows_sym
				(if (nil? order_node)
					(list (quote scan)
						'(session "__memcp_tx")
						(list (quote table) schema tbl)
						(cons (quote list) filtercols)
						(list (quote lambda) filter_params lowered_predicate)
						(cons (quote list) mapcols)
						(list (quote lambda) map_params lowered_value)
						(list (quote lambda) (list (quote acc) (quote value))
							(list (quote merge) (quote acc) (list (quote list) (quote value))))
						'()
						(list (quote lambda) (list (quote acc) (quote shard_rows))
							(list (quote merge) (quote acc) (quote shard_rows)))
						false)
					(list (quote scan_order)
						'(session "__memcp_tx")
						(list (quote table) schema tbl)
						(cons (quote list) filtercols)
						(list (quote lambda) filter_params lowered_predicate)
						(cons (quote list) ordercols)
						(cons (quote list) (scan_order_dirs (qattr order_node (quote order) '())))
						0
						(coalesceNil (qattr order_node (quote offset) nil) 0)
						(coalesceNil (qattr order_node (quote limit) nil) -1)
						(cons (quote list) mapcols)
						(list (quote lambda) map_params lowered_value)
						(list (quote lambda) (list (quote acc) (quote value))
							(list (quote merge) (quote acc) (list (quote list) (quote value))))
						'()
						false)))
			(scalar_from_rows_expr rows_sym strict)))))

(define lower_scalar_global_aggregate_scan (lambda (project_node scan_node predicate outer_specs)
	(begin
		(define alias (qid scan_node))
		(define schema (qattr scan_node (quote schema) nil))
		(define tbl (qattr scan_node (quote table) nil))
		(define fields (qattr project_node (quote output-fields) '()))
		(define value_expr (scalar_single_expr project_node))
		(define aggs (collect_field_aggregates fields))
		(define inputs (map aggs aggregate_input_expr))
		(define lowered_predicate (lower_embedded_scalars_with_specs
			(lower_expr_for_specs (lower_scan_expr (coalesceNil predicate true) scan_node) outer_specs)
			outer_specs))
		(define lowered_inputs (map inputs (lambda (expr) (lower_embedded_scalars_with_specs
			(lower_expr_for_specs (lower_scan_expr expr scan_node) outer_specs)
			outer_specs))))
		(define filtercols (scan_expr_columns predicate scan_node))
		(define mapcols (dedupe_list (merge (list filtercols (dedupe_list (merge (map inputs (lambda (expr) (scan_expr_columns expr scan_node)))))))))
		(define filter_params (map filtercols (lambda (col) (symbol (concat alias "." col)))))
		(define map_params (map mapcols (lambda (col) (symbol (concat alias "." col)))))
		(define agg_sym (symbol (concat "__neumann_scalar_agg_" (sha1 (string project_node)))))
		(list (quote begin)
			(list (quote define) agg_sym
				(list (quote scan)
					'(session "__memcp_tx")
					(list (quote table) schema tbl)
					(cons (quote list) filtercols)
					(list (quote lambda) filter_params lowered_predicate)
					(cons (quote list) mapcols)
					(list (quote lambda) map_params (cons (quote list) lowered_inputs))
					(list (quote lambda) (list (quote acc) (quote rowvals))
						(cons (quote list) (map (produceN (count aggs)) (lambda (i)
							(aggregate_reducer_expr (nth aggs i)
								(list (quote nth) (quote acc) i)
								(list (quote nth) (quote rowvals) i))))))
					(cons (quote list) (map aggs aggregate_neutral_expr))
					nil false))
			(lower_embedded_scalars (replace_aggregate_refs value_expr aggs agg_sym))))))

(define lower_project_group_scan (lambda (project_node group_node order_node scan_node predicate)
	(begin
		(define alias (qid scan_node))
		(define schema (qattr scan_node (quote schema) nil))
		(define tbl (qattr scan_node (quote table) nil))
		(define fields (qattr project_node (quote output-fields) '()))
		(define groups (qattr group_node (quote keys) '()))
		(define having (qattr group_node (quote having) nil))
		(define aggs (collect_field_aggregates fields))
		(define inputs (map aggs aggregate_input_expr))
		(define outer_specs (list (list scan_node false true)))
		(define lowered_predicate (lower_embedded_scalars_with_specs (lower_scan_expr (coalesceNil predicate true) scan_node) outer_specs))
		(define lowered_groups (map groups (lambda (expr) (lower_embedded_scalars_with_specs (lower_scan_expr expr scan_node) outer_specs))))
		(define lowered_inputs (map inputs (lambda (expr) (lower_embedded_scalars_with_specs (lower_scan_expr expr scan_node) outer_specs))))
		(define filtercols (scan_expr_columns predicate scan_node))
		(define mapcols (dedupe_list (merge (list
			filtercols
			(dedupe_list (merge (map groups (lambda (expr) (scan_expr_columns expr scan_node)))))
			(dedupe_list (merge (map inputs (lambda (expr) (scan_expr_columns expr scan_node)))))))))
		(define filter_params (map filtercols (lambda (col) (symbol (concat alias "." col)))))
		(define map_params (map mapcols (lambda (col) (symbol (concat alias "." col)))))
		(define groups_sym (symbol "__neumann_groups"))
		(define rows_sym (symbol "__neumann_group_rows"))
		(define sorted_sym (symbol "__neumann_group_sorted"))
		(define limited_sym (symbol "__neumann_group_limited"))
		(define key_sym (symbol "__key"))
		(define agg_sym (symbol "__agg"))
		(define old_sym (symbol "__old"))
		(define row_sym (symbol "__row"))
		(define new_sym (symbol "__new"))
		(define row_key_expr (list (quote nth) (quote rowvals) 0))
		(define row_vals_expr (list (quote nth) (quote rowvals) 1))
		(define agg_neutral (cons (quote list) (map aggs aggregate_neutral_expr)))
		(define merge_group_aggs
			(list (quote lambda) (list (quote old) (quote new))
				(cons (quote list) (map (produceN (count aggs)) (lambda (i)
					(aggregate_reducer_expr (nth aggs i)
						(list (quote nth) (quote old) i)
						(list (quote nth) (quote new) i)))))))
		(define final_fields (replace_group_fields fields groups aggs key_sym agg_sym))
		(define having_expr (if (nil? having) true (replace_group_refs having groups aggs key_sym agg_sym)))
		(list (quote begin)
			(list (quote define) groups_sym
				(list (quote scan)
					'(session "__memcp_tx")
					(list (quote table) schema tbl)
					(cons (quote list) filtercols)
					(list (quote lambda) filter_params lowered_predicate)
					(cons (quote list) mapcols)
					(list (quote lambda) map_params
						(list (quote list)
							(cons (quote list) lowered_groups)
							(cons (quote list) lowered_inputs)))
					(list (quote lambda) (list (quote acc) (quote rowvals))
						(list (quote begin)
							(list (quote define) old_sym (list (quote get_assoc) (quote acc) row_key_expr agg_neutral))
							(list (quote set_assoc) (quote acc) row_key_expr
								(cons (quote list) (map (produceN (count aggs)) (lambda (i)
									(aggregate_reducer_expr (nth aggs i)
										(list (quote nth) old_sym i)
										(list (quote nth) row_vals_expr i))))))))
					'()
					(list (quote lambda) (list (quote acc) (quote shard_groups))
						(list (quote merge_assoc) (quote acc) (quote shard_groups) merge_group_aggs))
					false))
			(list (quote define) rows_sym
				(list (quote filter)
					(list (quote extract_assoc) groups_sym
						(list (quote lambda) (list key_sym agg_sym)
							(list (quote begin)
								(list (quote print) "NEUMANN_DEBUG scalar_group_extract" key_sym agg_sym having_expr)
								(list (quote if) having_expr
									(build_row_assoc_expr final_fields)
									nil))))
					(list (quote lambda) (list (quote row))
						(list (quote not) (list (quote nil?) (quote row))))))
			(list (quote define) sorted_sym (group_sort_rows_expr rows_sym fields (if (nil? order_node) '() (qattr order_node (quote order) '()))))
			(list (quote define) limited_sym (group_limit_rows_expr sorted_sym
				(if (nil? order_node) nil (qattr order_node (quote limit) nil))
				(if (nil? order_node) nil (qattr order_node (quote offset) nil))))
			(list (quote map) limited_sym
				(list (quote lambda) (list row_sym)
					(list (quote resultrow) row_sym)))))))

(define order_aggregates (lambda (order)
	(dedupe_list (merge (map (coalesceNil order '()) (lambda (item) (match item
		'(expr _dir) (collect_aggregates expr)
		'())))))))

(define lower_scalar_group_scan (lambda (project_node group_node order_node scan_node predicate outer_specs)
	(begin
		(define alias (qid scan_node))
		(define schema (qattr scan_node (quote schema) nil))
		(define tbl (qattr scan_node (quote table) nil))
		(define fields (qattr project_node (quote output-fields) '()))
		(define value_expr (scalar_single_expr project_node))
		(define groups (qattr group_node (quote keys) '()))
		(define having (qattr group_node (quote having) nil))
		(define order (if (nil? order_node) '() (qattr order_node (quote order) '())))
		(define aggs (dedupe_list (merge (list
			(collect_field_aggregates fields)
			(collect_aggregates having)
			(order_aggregates order)))))
		(define inputs (map aggs aggregate_input_expr))
		(define lowered_predicate (lower_embedded_scalars_with_specs
			(lower_expr_for_specs (lower_scan_expr (coalesceNil predicate true) scan_node) outer_specs)
			outer_specs))
		(define lowered_groups (map groups (lambda (expr) (lower_embedded_scalars_with_specs
			(lower_expr_for_specs (lower_scan_expr expr scan_node) outer_specs)
			outer_specs))))
		(define lowered_inputs (map inputs (lambda (expr) (lower_embedded_scalars_with_specs
			(lower_expr_for_specs (lower_scan_expr expr scan_node) outer_specs)
			outer_specs))))
		(define filtercols (scan_expr_columns predicate scan_node))
		(define mapcols (dedupe_list (merge (list
			filtercols
			(dedupe_list (merge (map groups (lambda (expr) (scan_expr_columns expr scan_node)))))
			(dedupe_list (merge (map inputs (lambda (expr) (scan_expr_columns expr scan_node)))))))))
		(define filter_params (map filtercols (lambda (col) (symbol (concat alias "." col)))))
		(define map_params (map mapcols (lambda (col) (symbol (concat alias "." col)))))
		(define groups_sym (symbol (concat "__neumann_scalar_groups_" (sha1 (string project_node)))))
		(define rows_sym (symbol (concat "__neumann_scalar_group_rows_" (sha1 (string project_node)))))
		(define sorted_sym (symbol (concat "__neumann_scalar_group_sorted_" (sha1 (string project_node)))))
		(define limited_sym (symbol (concat "__neumann_scalar_group_limited_" (sha1 (string project_node)))))
		(define values_sym (symbol (concat "__neumann_scalar_group_values_" (sha1 (string project_node)))))
		(define key_sym (symbol "__key"))
		(define agg_sym (symbol "__agg"))
		(define row_sym (symbol "__row"))
		(define old_sym (symbol "__old"))
		(define row_key_expr (list (quote nth) (quote rowvals) 0))
		(define row_vals_expr (list (quote nth) (quote rowvals) 1))
		(define agg_neutral (cons (quote list) (map aggs aggregate_neutral_expr)))
		(define merge_group_aggs
			(list (quote lambda) (list (quote old) (quote new))
				(cons (quote list) (map (produceN (count aggs)) (lambda (i)
					(aggregate_reducer_expr (nth aggs i)
						(list (quote nth) (quote old) i)
						(list (quote nth) (quote new) i)))))))
		(define order_expr (match order
			(cons first _rest) (match first
				'(expr _dir) (replace_group_refs expr groups aggs key_sym agg_sym)
				_ nil)
			'() nil))
		(define scalar_fields (list "__value" value_expr "__order" (coalesceNil order_expr 0)))
		(define final_fields (replace_group_fields scalar_fields groups aggs key_sym agg_sym))
		(define having_expr (if (nil? having) true (replace_group_refs having groups aggs key_sym agg_sym)))
		(define order_dir (match order
			(cons first _rest) (match first
				'(_expr dir) dir
				_ <)
			'() <))
		(list (quote begin)
			(list (quote define) groups_sym
				(list (quote scan)
					'(session "__memcp_tx")
					(list (quote table) schema tbl)
					(cons (quote list) filtercols)
					(list (quote lambda) filter_params lowered_predicate)
					(cons (quote list) mapcols)
					(list (quote lambda) map_params
						(list (quote list)
							(cons (quote list) lowered_groups)
							(cons (quote list) lowered_inputs)))
					(list (quote lambda) (list (quote acc) (quote rowvals))
						(list (quote begin)
							(list (quote define) old_sym (list (quote get_assoc) (quote acc) row_key_expr agg_neutral))
							(list (quote set_assoc) (quote acc) row_key_expr
								(cons (quote list) (map (produceN (count aggs)) (lambda (i)
									(aggregate_reducer_expr (nth aggs i)
										(list (quote nth) old_sym i)
										(list (quote nth) row_vals_expr i))))))))
					'()
					(list (quote lambda) (list (quote acc) (quote shard_groups))
						(list (quote merge_assoc) (quote acc) (quote shard_groups) merge_group_aggs))
					false))
			(list (quote define) rows_sym
				(list (quote reduce_assoc) groups_sym
					(list (quote lambda) (list (quote acc) key_sym agg_sym)
						(list (quote if) having_expr
							(list (quote merge) (quote acc) (list (quote list) (build_row_assoc_expr final_fields)))
							(quote acc)))
					'()))
			(list (quote define) sorted_sym
				(if (equal? order '())
					rows_sym
					(list (quote sort) rows_sym
						(list (quote lambda) (list (quote a) (quote b))
							(list order_dir
								(list (quote get_assoc) (quote a) "__order")
								(list (quote get_assoc) (quote b) "__order"))))))
			(list (quote define) limited_sym (group_limit_rows_expr sorted_sym
				(if (nil? order_node) nil (qattr order_node (quote limit) nil))
				(if (nil? order_node) nil (qattr order_node (quote offset) nil))))
			(list (quote define) values_sym
				(list (quote map) limited_sym
					(list (quote lambda) (list row_sym)
						(list (quote get_assoc) row_sym "__value"))))
			(scalar_from_rows_expr values_sym (or (nil? order_node) (nil? (qattr order_node (quote limit) nil))))))))

(define lower_project_group (lambda (project_node group_node order_node)
	(match (qchildren group_node)
		(cons child '()) (match (qop child)
			(quote scan) (lower_project_group_scan project_node group_node order_node child true)
			(quote select) (match (qchildren child)
				(cons grandchild '()) (if (equal? (qop grandchild) (quote scan))
					(lower_project_group_scan project_node group_node order_node grandchild (qattr child (quote predicate) true))
					(neumann_fail "build_queryplan" "GROUP BY select lowerer only supports scan input yet"))
				_ (neumann_fail "build_queryplan" "select expects one child"))
			_ (neumann_fail "build_queryplan" "GROUP BY lowerer only supports scan input yet"))
		_ (neumann_fail "build_queryplan" "GROUP BY expects one child"))))

(define lower_scalar_group (lambda (project_node group_node order_node outer_specs)
	(match (qchildren group_node)
		(cons child '()) (match (qop child)
			(quote scan) (lower_scalar_group_scan project_node group_node order_node child true outer_specs)
			(quote select) (match (qchildren child)
				(cons grandchild '()) (if (equal? (qop grandchild) (quote scan))
					(lower_scalar_group_scan project_node group_node order_node grandchild (qattr child (quote predicate) true) outer_specs)
					(neumann_fail "build_queryplan" "scalar GROUP BY select lowerer only supports scan input yet"))
				_ (neumann_fail "build_queryplan" "select expects one child"))
			_ (neumann_fail "build_queryplan" "scalar GROUP BY lowerer only supports scan input yet"))
		_ (neumann_fail "build_queryplan" "scalar GROUP BY expects one child"))))

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
				(cons (quote list) (scan_order_columns (qattr order_node (quote order) '()) scan_node))
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

(define join_scan_specs (lambda (node) (match (qop node)
	(quote scan) (list (list node false true))
	(quote join) (match (qchildren node)
		'(left right) (if (equal? (qop right) (quote scan))
			(merge
				(join_scan_specs left)
				(list (list right
					(equal? (qattr node (quote join-kind) (quote inner)) (quote left))
					(qattr node (quote predicate) true))))
			(neumann_fail "build_queryplan" "join right child must be a scan"))
		_ (neumann_fail "build_queryplan" "join expects two children"))
	_ (neumann_fail "build_queryplan" "join lowerer only supports scan/join trees"))))

(define spec_node (lambda (spec) (nth spec 0)))
(define spec_outer? (lambda (spec) (nth spec 1)))
(define spec_predicate (lambda (spec) (nth spec 2)))

(define specs_aliases (lambda (specs)
	(map specs (lambda (spec) (qid (spec_node spec))))))

(define specs_predicates (lambda (specs)
	(map specs spec_predicate)))

(define scan_cols_for_spec (lambda (fields predicates order spec specs)
	(dedupe_list (merge (list
		(fields_columns_for_spec_in_join fields spec specs)
		(dedupe_list (merge (map predicates (lambda (predicate) (expr_columns_for_spec predicate spec specs)))))
		(scan_order_columns_for_spec order spec specs))))))

(define lower_scan_specs (lambda (specs all_specs fields order_node final_predicate prefix_aliases use_order)
	(match specs
		(cons spec rest) (begin
			(define node (spec_node spec))
			(define alias (qid node))
			(define aliases_now (merge prefix_aliases (list alias)))
			(define all_aliases (specs_aliases all_specs))
			(define order (if (nil? order_node) '() (qattr order_node (quote order) '())))
			(define predicates (merge (specs_predicates all_specs) (list (coalesceNil final_predicate true))))
			(define mapcols (scan_cols_for_spec fields predicates order spec all_specs))
			(define filter_predicate (if (equal? rest '())
				(combine_and (spec_predicate spec) (coalesceNil final_predicate true))
				(spec_predicate spec)))
			(define lowered_filter (lower_expr_for_specs filter_predicate (filter all_specs (lambda (s)
				(contains? aliases_now (qid (spec_node s)))))))
			(define map_expr (if (equal? rest '())
				(build_resultrow_expr (map_assoc fields (lambda (_key expr)
					(lower_expr_for_specs expr all_specs))))
				(lower_scan_specs rest all_specs fields order_node final_predicate aliases_now false)))
			(scan_call (if (and use_order (not (nil? order_node))) (quote scan_order) (quote scan))
				node
				(expr_columns_for_spec filter_predicate spec all_specs)
				lowered_filter
				mapcols
				map_expr
				(spec_outer? spec)
				order_node))
		'() (neumann_fail "build_queryplan" "empty join scan sequence"))))

(define lower_project_join (lambda (project_node join_node order_node final_predicate)
	(begin
		(define specs (join_scan_specs join_node))
		(lower_scan_specs specs specs
			(qattr project_node (quote output-fields) '())
			order_node final_predicate '() true))))

(define lower_project_scan_rows (lambda (project_node scan_node predicate order_node)
	(begin
		(define alias (qid scan_node))
		(define schema (qattr scan_node (quote schema) nil))
		(define tbl (qattr scan_node (quote table) nil))
		(define fields (qattr project_node (quote output-fields) '()))
		(define order (if (nil? order_node) '() (qattr order_node (quote order) '())))
		(define outer_specs (list (list scan_node false true)))
		(define effective_predicate (scan_effective_predicate scan_node predicate))
		(define lowered_predicate (lower_embedded_scalars_with_specs (lower_scan_expr effective_predicate scan_node) outer_specs))
		(define lowered_fields (map_assoc (lower_scan_fields fields scan_node) (lambda (_key expr)
			(lower_embedded_scalars_with_specs expr outer_specs))))
		(define filtercols (scan_expr_columns effective_predicate scan_node))
		(define ordercols (if (nil? order_node) '() (scan_order_columns order scan_node)))
		(define mapcols (dedupe_list (merge (list filtercols ordercols (scan_fields_columns fields scan_node)))))
		(define filter_params (map filtercols (lambda (col) (symbol (concat alias "." col)))))
		(define map_params (map mapcols (lambda (col) (symbol (concat alias "." col)))))
		(if (nil? order_node)
			(list (quote scan)
				'(session "__memcp_tx")
				(list (quote table) schema tbl)
				(cons (quote list) filtercols)
				(list (quote lambda) filter_params lowered_predicate)
				(cons (quote list) mapcols)
				(list (quote lambda) map_params (build_row_assoc_expr lowered_fields))
				(list (quote lambda) (list (quote acc) (quote row))
					(list (quote merge) (quote acc) (list (quote list) (quote row))))
				'()
				(list (quote lambda) (list (quote acc) (quote shard_rows))
					(list (quote merge) (quote acc) (quote shard_rows)))
				false)
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
				(list (quote lambda) map_params (build_row_assoc_expr lowered_fields))
				(list (quote lambda) (list (quote acc) (quote row))
					(list (quote merge) (quote acc) (list (quote list) (quote row))))
				'()
				false)))))

(define lower_scan_specs_rows (lambda (specs all_specs fields order_node final_predicate prefix_aliases use_order)
	(match specs
		(cons spec rest) (begin
			(define node (spec_node spec))
			(define alias (qid node))
			(define aliases_now (merge prefix_aliases (list alias)))
			(define order (if (nil? order_node) '() (qattr order_node (quote order) '())))
			(define predicates (merge (specs_predicates all_specs) (list (coalesceNil final_predicate true))))
			(define mapcols (scan_cols_for_spec fields predicates order spec all_specs))
			(define filter_predicate (if (equal? rest '())
				(combine_and (spec_predicate spec) (coalesceNil final_predicate true))
				(spec_predicate spec)))
			(define lowered_filter (lower_expr_for_specs filter_predicate (filter all_specs (lambda (s)
				(contains? aliases_now (qid (spec_node s)))))))
			(define map_expr (if (equal? rest '())
				(build_row_assoc_expr (map_assoc fields (lambda (_key expr)
					(lower_expr_for_specs expr all_specs))))
				(lower_scan_specs_rows rest all_specs fields order_node final_predicate aliases_now false)))
			(define rows_expr (scan_call (if (and use_order (not (nil? order_node))) (quote scan_order) (quote scan))
				node
				(expr_columns_for_spec filter_predicate spec all_specs)
				lowered_filter
				mapcols
				map_expr
				(spec_outer? spec)
				order_node))
			(if (equal? rest '())
				(list (quote reduce) rows_expr
					(list (quote lambda) (list (quote acc) (quote row))
						(list (quote merge) (quote acc) (list (quote list) (quote row))))
					'())
				rows_expr))
		'() (neumann_fail "build_queryplan" "empty join scan sequence"))))

(define lower_project_join_rows (lambda (project_node join_node order_node final_predicate)
	(begin
		(define specs (join_scan_specs join_node))
		(lower_scan_specs_rows specs specs
			(qattr project_node (quote output-fields) '())
			order_node final_predicate '() true))))

(define lower_qnode_rows (lambda (node) (match (qop node)
	(quote project) (match (qchildren node)
		(cons child '()) (match (qop child)
			(quote empty-row) (list (quote list) (build_row_assoc_expr (qattr node (quote output-fields) '())))
			(quote select) (match (qchildren child)
				(cons grandchild '()) (match (qop grandchild)
					(quote scan) (lower_project_scan_rows node grandchild (qattr child (quote predicate) true) nil)
					(quote join) (lower_project_join_rows node grandchild nil (qattr child (quote predicate) true))
					_ (neumann_fail "build_queryplan" "UNION branch select lowerer only supports scan or join input yet"))
				_ (neumann_fail "build_queryplan" "select expects one child"))
			(quote scan) (lower_project_scan_rows node child true nil)
			(quote order_limit) (match (qchildren child)
				(cons grandchild '()) (match (qop grandchild)
					(quote scan) (lower_project_scan_rows node grandchild true child)
					(quote select) (match (qchildren grandchild)
						(cons scan_child '()) (if (equal? (qop scan_child) (quote scan))
							(lower_project_scan_rows node scan_child (qattr grandchild (quote predicate) true) child)
							(neumann_fail "build_queryplan" "UNION branch order/select lowerer only supports scan input yet"))
						_ (neumann_fail "build_queryplan" "select expects one child"))
					_ (neumann_fail "build_queryplan" "UNION branch order lowerer only supports scan input yet"))
				_ (neumann_fail "build_queryplan" "order_limit expects one child"))
			(quote join) (lower_project_join_rows node child nil true)
			_ (neumann_fail "build_queryplan" "UNION branch project lowerer only supports scan/join input yet"))
		_ (neumann_fail "build_queryplan" "project expects one child"))
	(quote union) (lower_union_rows node)
	_ (neumann_fail "build_queryplan" "UNION branch root must be project"))))

(define union_branch_keys (lambda (project_node)
	(extract_assoc (qattr project_node (quote output-fields) '()) (lambda (key _expr) (string key)))))

(define union_check_branch_keys (lambda (keys branch)
	(if (not (equal? (count keys) (count (union_branch_keys branch))))
		(neumann_fail "build_queryplan" "UNION branch column count mismatch")
		true)))

(define union_normalize_branch (lambda (branch target_keys)
	(begin
		(define source_keys (union_branch_keys branch))
		(union_check_branch_keys target_keys branch)
		(list (quote map) (lower_qnode_rows branch)
			(list (quote lambda) (list (quote row))
				(cons (quote list)
					(merge (map (produceN (count target_keys)) (lambda (i)
								(list
									(list (quote quote) (nth target_keys i))
										(list (quote get_assoc) (quote row) (list (quote quote) (nth source_keys i)))))))))))))

(define union_rows_expr (lambda (union_node target_keys)
	(cons (quote merge) (map (qchildren union_node) (lambda (branch)
		(union_normalize_branch branch target_keys))))))

(define union_distinct_rows_expr (lambda (rows_expr)
	(list (quote reduce) rows_expr
		(list (quote lambda) (list (quote acc) (quote row))
			(list (quote append_unique) (quote acc) (quote row)))
		'())))

(define union_order_key (lambda (item)
	(match item
		'(((symbol get_column) _tbl _ti col _ci) _dir) col
		'(((quote get_column) _tbl _ti col _ci) _dir) col
		'(expr _dir) (string expr)
		_ nil)))

(define union_order_positions (lambda (target_keys order)
	(map (coalesceNil order '()) (lambda (item) (begin
		(define key (union_order_key item))
		(define pos (reduce (produceN (count target_keys)) (lambda (found i)
			(if (not (nil? found)) found
				(if (equal?? key (nth target_keys i)) i nil))) nil))
		(if (nil? pos)
			(neumann_fail "build_queryplan" (concat "UNION ORDER BY column not found: " key))
			pos))))))

(define union_order_dirs (lambda (order)
	(map (coalesceNil order '()) (lambda (item) (match item
		'(_expr dir) dir
		_ <)))))

(define union_branch_scan_parts (lambda (branch)
	(match (qop branch)
		(quote project) (match (qchildren branch)
			(cons child '()) (match (qop child)
				(quote scan) (list branch child true)
				(quote select) (match (qchildren child)
					(cons scan_child '()) (if (equal? (qop scan_child) (quote scan))
						(list branch scan_child (qattr child (quote predicate) true))
						nil)
					_ nil)
				_ nil)
			_ nil)
		_ nil)))

(define union_scan_sort_col (lambda (expr scan_node)
	(match expr
		'((symbol get_column) tbl _ col ci) (if (or (nil? tbl) (equal?? tbl (qid scan_node))) col
			(list (quote lambda)
				(map (scan_expr_columns expr scan_node) (lambda (c) (symbol (concat (qid scan_node) "." c))))
				(lower_scan_expr expr scan_node)))
		'((quote get_column) tbl _ col ci) (if (or (nil? tbl) (equal?? tbl (qid scan_node))) col
			(list (quote lambda)
				(map (scan_expr_columns expr scan_node) (lambda (c) (symbol (concat (qid scan_node) "." c))))
				(lower_scan_expr expr scan_node)))
		_ (list (quote lambda)
			(map (scan_expr_columns expr scan_node) (lambda (c) (symbol (concat (qid scan_node) "." c))))
			(lower_scan_expr expr scan_node)))))

(define union_scan_sortcols_supported? (lambda (branch_specs)
	(reduce branch_specs (lambda (ok spec)
		(and ok (reduce (nth spec 3) (lambda (ok2 col) (and ok2 (string? col))) true)))
		true)))

(define lower_union_order_scan_multi (lambda (union_node)
	(begin
		(define order (qattr union_node (quote order) '()))
		(if (equal? (coalesceNil order '()) '())
			nil
			(match (qchildren union_node)
				(cons first _rest) (begin
					(define target_keys (union_branch_keys first))
					(define positions (union_order_positions target_keys order))
					(define branch_parts (map (qchildren union_node) union_branch_scan_parts))
					(if (contains? branch_parts nil)
						nil
						(begin
							(define seen_sym (symbol (concat "__neumann_union_seen_" (sha1 (string union_node)))))
							(define row_sym (symbol (concat "__neumann_union_order_row_" (sha1 (string union_node)))))
							(define key_sym (symbol (concat "__neumann_union_order_key_" (sha1 (string union_node)))))
							(define distinct? (equal? (qattr union_node (quote union-kind) (quote all)) (quote distinct)))
							(define branch_specs (map branch_parts (lambda (part) (begin
								(define project_node (nth part 0))
								(define scan_node (nth part 1))
								(define predicate (nth part 2))
								(define effective_predicate (scan_effective_predicate scan_node predicate))
								(define fields (qattr project_node (quote output-fields) '()))
								(union_check_branch_keys target_keys project_node)
								(define field_exprs (extract_assoc fields (lambda (_key expr) expr)))
								(define normalized_fields (merge (map (produceN (count target_keys)) (lambda (i)
									(list (nth target_keys i) (nth field_exprs i))))))
								(define lowered_fields (map_assoc (lower_scan_fields normalized_fields scan_node) (lambda (_key expr)
									(lower_embedded_scalars expr))))
								(define sort_exprs (map positions (lambda (pos) (nth field_exprs pos))))
								(define sortcols (map sort_exprs (lambda (expr) (union_scan_sort_col expr scan_node))))
								(define filtercols (scan_expr_columns effective_predicate scan_node))
								(define mapcols (dedupe_list (merge (list
									(scan_fields_columns normalized_fields scan_node)
									(dedupe_list (merge (map sort_exprs (lambda (expr) (scan_expr_columns expr scan_node)))))))))
								(define filter_params (map filtercols (lambda (col) (symbol (concat (qid scan_node) "." col)))))
								(define map_params (map mapcols (lambda (col) (symbol (concat (qid scan_node) "." col)))))
								(define row_expr (build_row_assoc_expr lowered_fields))
								(list scan_node filtercols
									(list (quote lambda) filter_params (lower_scan_expr effective_predicate scan_node))
									sortcols
									mapcols
									(list (quote lambda) map_params
										(if distinct?
											(list (quote begin)
												(list (quote define) row_sym row_expr)
												(list (quote define) key_sym (list (quote serialize) row_sym))
												(list (quote if) (list seen_sym key_sym)
													nil
													(list (quote begin)
														(list seen_sym key_sym true)
														(list (quote resultrow) row_sym))))
											(list (quote resultrow) row_expr))))))))
							(if (not (union_scan_sortcols_supported? branch_specs))
								nil
								(begin
									(define plan (merge (list (quote scan_order_multi) '(session "__memcp_tx"))
										(list
											(cons (quote list) (map branch_specs (lambda (s)
												(list (quote table) (qattr (nth s 0) (quote schema) nil) (qattr (nth s 0) (quote table) nil)))))
											(cons (quote list) (map branch_specs (lambda (s) (cons (quote list) (nth s 1)))))
											(cons (quote list) (map branch_specs (lambda (s) (nth s 2))))
											(cons (quote list) (map branch_specs (lambda (s) (cons (quote list) (nth s 3)))))
											(cons (quote list) (union_order_dirs order))
											nil
											nil
											0
											(coalesceNil (qattr union_node (quote offset) nil) 0)
											(coalesceNil (qattr union_node (quote limit) nil) -1)
											(cons (quote list) (map branch_specs (lambda (s) (cons (quote list) (nth s 4)))))
											(cons (quote list) (map branch_specs (lambda (s) (nth s 5)))))))
									(if distinct?
										(list (quote begin)
											(list (quote define) seen_sym (list (quote newsession)))
											plan)
										plan))))))
				_ nil)))))

(define union_sort_rows_expr (lambda (rows_sym order)
	(if (equal? (coalesceNil order '()) '())
		rows_sym
		rows_sym)))

(define lower_union_rows (lambda (union_node)
	(match (qchildren union_node)
		(cons first _rest) (begin
			(define target_keys (union_branch_keys first))
			(define rows_sym (symbol (concat "__neumann_union_rows_" (sha1 (string union_node)))))
			(define distinct_sym (symbol (concat "__neumann_union_distinct_" (sha1 (string union_node)))))
			(define sorted_sym (symbol (concat "__neumann_union_sorted_" (sha1 (string union_node)))))
			(define rows_expr (union_rows_expr union_node target_keys))
			(list (quote begin)
				(list (quote define) rows_sym rows_expr)
				(list (quote define) distinct_sym
					(if (equal? (qattr union_node (quote union-kind) (quote all)) (quote distinct))
						(union_distinct_rows_expr rows_sym)
						rows_sym))
				(list (quote define) sorted_sym
					(union_sort_rows_expr distinct_sym (qattr union_node (quote order) '())))
				(group_limit_rows_expr sorted_sym
					(qattr union_node (quote limit) nil)
					(qattr union_node (quote offset) nil))))
		_ '())))

(define lower_union (lambda (union_node)
	(begin
		(define rows_sym (symbol (concat "__neumann_union_emit_" (sha1 (string union_node)))))
		(define row_sym (symbol (concat "__neumann_union_row_" (sha1 (string union_node)))))
		(define ordered_plan (lower_union_order_scan_multi union_node))
		(if (nil? ordered_plan)
			(list (quote begin)
				(list (quote define) rows_sym (lower_union_rows union_node))
				(list (quote map) rows_sym
					(list (quote lambda) (list row_sym)
						(list (quote resultrow) row_sym))))
			ordered_plan))))

(define lower_union_row_expr (lambda (expr row_sym) (match expr
	'((symbol get_column) _tbl _ti col _ci) (list (quote get_assoc) row_sym col)
	'((quote get_column) _tbl _ti col _ci) (list (quote get_assoc) row_sym col)
	(cons sym args)
	(cons (lower_union_row_expr sym row_sym)
		(map args (lambda (arg) (lower_union_row_expr arg row_sym))))
	expr)))

(define lower_project_global_aggregate_union (lambda (project_node union_node)
	(begin
		(define fields (qattr project_node (quote output-fields) '()))
		(define aggs (collect_field_aggregates fields))
		(define inputs (map aggs aggregate_input_expr))
		(define rows_sym (symbol (concat "__neumann_union_agg_rows_" (sha1 (string project_node)))))
		(define agg_sym (symbol (concat "__neumann_union_agg_" (sha1 (string project_node)))))
		(define row_sym (symbol (concat "__neumann_union_agg_row_" (sha1 (string project_node)))))
		(list (quote begin)
			(list (quote define) rows_sym (lower_union_rows union_node))
			(list (quote define) agg_sym
				(list (quote reduce) rows_sym
					(list (quote lambda) (list (quote acc) row_sym)
						(cons (quote list) (map (produceN (count aggs)) (lambda (i)
							(aggregate_reducer_expr (nth aggs i)
								(list (quote nth) (quote acc) i)
								(lower_union_row_expr (nth inputs i) row_sym))))))
					(cons (quote list) (map aggs aggregate_neutral_expr))))
			(build_resultrow_expr (replace_field_aggregates fields aggs agg_sym))))))

(define lower_qnode (lambda (node) (match (qop node)
	(quote project) (match (qchildren node)
		(cons child '()) (match (qop child)
			(quote empty-row) (lower_project_empty_row node child)
			(quote select) (match (qchildren child)
				(cons grandchild '()) (match (qop grandchild)
					(quote empty-row) (lower_project_empty_row node child)
					(quote scan) (if (has_aggregates? (qattr node (quote output-fields) '()))
						(lower_project_global_aggregate_scan node grandchild (qattr child (quote predicate) true))
						(lower_project_scan node grandchild (qattr child (quote predicate) true)))
					(quote join) (lower_project_join node grandchild nil (qattr child (quote predicate) true))
					_ (neumann_fail "build_queryplan" "select lowerer only supports empty-row or scan input yet"))
				_ (neumann_fail "build_queryplan" "select expects one child"))
			(quote scan) (if (has_aggregates? (qattr node (quote output-fields) '()))
				(lower_project_global_aggregate_scan node child true)
				(lower_project_scan node child true))
			(quote order_limit) (match (qchildren child)
				(cons grandchild '()) (match (qop grandchild)
					(quote scan) (lower_project_scan_order node child grandchild true)
					(quote select) (match (qchildren grandchild)
						(cons scan_child '()) (match (qop scan_child)
							(quote scan) (lower_project_scan_order node child scan_child (qattr grandchild (quote predicate) true))
							(quote join) (lower_project_join node scan_child child (qattr grandchild (quote predicate) true))
							_ (neumann_fail "build_queryplan" "order_limit/select lowerer only supports scan input yet"))
						_ (neumann_fail "build_queryplan" "select expects one child"))
					(quote join) (lower_project_join node grandchild child true)
					(quote group) (lower_project_group node grandchild child)
					_ (neumann_fail "build_queryplan" "order_limit lowerer only supports scan input yet"))
				_ (neumann_fail "build_queryplan" "order_limit expects one child"))
			(quote join) (lower_project_join node child nil true)
			(quote group) (lower_project_group node child nil)
			(quote union) (if (has_aggregates? (qattr node (quote output-fields) '()))
				(lower_project_global_aggregate_union node child)
				(neumann_fail "build_queryplan" "project/union lowerer only supports aggregate projection yet"))
			_ (neumann_fail "build_queryplan" "project lowerer only supports empty-row or scan input yet"))
		_ (neumann_fail "build_queryplan" "project expects one child"))
	(quote union) (lower_union node)
	_ (neumann_fail "build_queryplan" (concat "operator not ported yet: " (qop node))))))

(define lower_scalar_project (lambda (project_node outer_specs)
	(match (qchildren project_node)
		(cons child '()) (match (qop child)
			(quote scan) (if (has_aggregates? (qattr project_node (quote output-fields) '()))
				(lower_scalar_global_aggregate_scan project_node child true outer_specs)
				(lower_scalar_scan project_node child true nil outer_specs))
			(quote select) (match (qchildren child)
				(cons grandchild '()) (if (equal? (qop grandchild) (quote scan))
					(if (has_aggregates? (qattr project_node (quote output-fields) '()))
						(lower_scalar_global_aggregate_scan project_node grandchild (qattr child (quote predicate) true) outer_specs)
						(lower_scalar_scan project_node grandchild (qattr child (quote predicate) true) nil outer_specs))
					(neumann_fail "build_queryplan" "scalar/select lowerer only supports scan input yet"))
				_ (neumann_fail "build_queryplan" "select expects one child"))
			(quote order_limit) (match (qchildren child)
				(cons grandchild '()) (match (qop grandchild)
					(quote scan) (if (has_aggregates? (qattr project_node (quote output-fields) '()))
						(lower_scalar_global_aggregate_scan project_node grandchild true outer_specs)
						(lower_scalar_scan project_node grandchild true child outer_specs))
					(quote select) (match (qchildren grandchild)
						(cons scan_child '()) (if (equal? (qop scan_child) (quote scan))
							(if (has_aggregates? (qattr project_node (quote output-fields) '()))
								(lower_scalar_global_aggregate_scan project_node scan_child (qattr grandchild (quote predicate) true) outer_specs)
								(lower_scalar_scan project_node scan_child (qattr grandchild (quote predicate) true) child outer_specs))
							(neumann_fail "build_queryplan" "scalar/order_limit/select lowerer only supports scan input yet"))
						_ (neumann_fail "build_queryplan" "select expects one child"))
					(quote group) (lower_scalar_group project_node grandchild child outer_specs)
					_ (neumann_fail "build_queryplan" "scalar/order_limit lowerer only supports scan input yet"))
				_ (neumann_fail "build_queryplan" "order_limit expects one child"))
			(quote group) (lower_scalar_group project_node child nil outer_specs)
			_ (neumann_fail "build_queryplan" "scalar lowerer only supports scan input yet"))
		_ (neumann_fail "build_queryplan" "scalar project expects one child"))))

(define lower_scalar_ir_with_specs (lambda (ir outer_specs)
	(match (qop (ir_root ir))
		(quote project) (lower_scalar_project (ir_root ir) outer_specs)
		_ (neumann_fail "build_queryplan" "scalar IR root must be project"))))

(define lower_scalar_ir (lambda (ir)
	(lower_scalar_ir_with_specs ir '())))

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
