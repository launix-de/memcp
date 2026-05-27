/*
Copyright (C) 2026  Carl-Philip Hänsch

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
== Layer 1 — Query Plan IR (algebra operators + 7-tuple leaves) ==

Per BTW2025 "Improving Unnesting of Complex Queries" §2.1, §3, and FAQ §31, §32.

This module defines the IR used by the holistic top-down unnesting compiler.
It is pure data — every transformation is a tree-to-tree rewrite, no shared
mutable state, no session caches.

Operator set (BTW2025 §2.1 + §3.3 rules):

qpir-scan   schema table                          — base table scan (leaf)
qpir-select predicate child                       — σ_p
qpir-map    projections child                     — χ (computed columns)
qpir-groupby keys aggs having child               — Γ_{keys; aggs}
qpir-window partition order computations child    — Ω
qpir-join   type predicate left right             — ⋈_p (inner/left/right/full/semi/anti)
qpir-dep-join predicate left right accessing      — ⋈ᵈ_p (BTW2025 notation)
qpir-union  order limit offset branches           — UNION ALL (FAQ §14)
qpir-iterate seed recursive iterationscans        — WITH RECURSIVE (FAQ §45)
qpir-leaf   7tuple                                — one SQL block (compiled by L4)

Invariants (per FAQ §1 / §7):
- After [lift_dep_joins_pass]: no qpir-leaf's 7-tuple contains inner_select markers
- After [unnest_pass]: no qpir-dep-join nodes remain in the tree
- Free variables F(N) must be ∅ at the root before lowering (Layer 4)
*/

/* ==================== Operator constructors ==================== */

/* qpir-scan: leaf scanning one base table.
schema, table — both strings naming a physical table */
(define qpir-scan (lambda (schema table)
	(list (quote qpir-scan) schema table)))

/* qpir-select: selection σ_p.
predicate — boolean expression
child — child operator */
(define qpir-select (lambda (predicate child)
	(list (quote qpir-select) predicate child)))

/* qpir-map: projection χ producing computed columns.
projections — assoc list of (name expr) pairs (same shape as 7-tuple fields)
child — child operator */
(define qpir-map (lambda (projections child)
	(list (quote qpir-map) projections child)))

/* qpir-groupby: Γ_{keys; aggs}.
keys — list of grouping expressions
aggs — assoc list of (name aggregate-expr) pairs
having — boolean expression filtering groups, or nil
child — child operator */
(define qpir-groupby (lambda (keys aggs having child)
	(list (quote qpir-groupby) keys aggs having child)))

/* qpir-window: window operator Ω.
partition — list of partition-by expressions
order — list of (expr direction) order items
computations — assoc list of (name window-func) pairs, plus limit/offset
child — child operator */
(define qpir-window (lambda (partition order computations child)
	(list (quote qpir-window) partition order computations child)))

/* qpir-join: regular join ⋈_p.
type — one of: inner, left, right, full, semi, anti
predicate — boolean expression
left, right — child operators */
(define qpir-join (lambda (type predicate left right)
	(list (quote qpir-join) type predicate left right)))

/* qpir-dep-join: dependent join ⋈ᵈ_p (BTW2025 §2.1).
The right side may reference columns from the left side (correlation).
predicate — boolean expression
left, right — child operators
accessing — list of operator references in right that read columns from left
(set by [annotate_dep_joins], used by [unnest_pass]).
This node MUST NOT appear in the tree after the unnesting pass. */
(define qpir-dep-join (lambda (predicate left right accessing)
	(list (quote qpir-dep-join) predicate left right accessing)))

/* qpir-union: UNION ALL across branches (FAQ §14).
order — list of (expr direction) order items, or nil
limit, offset — numeric or nil
branches — non-empty list of child operators with identical output schemas */
(define qpir-union (lambda (order limit offset branches)
	(list (quote qpir-union) order limit offset branches)))

/* qpir-iterate: WITH RECURSIVE iterate operator (FAQ §45, BTW2025 §4.2).
seed — child operator producing the initial rows
recursive — child operator producing the next iteration (uses iterationscans)
iterationscans — list of iterationscan placeholders within recursive that
read the previous iteration's rows */
(define qpir-iterate (lambda (seed recursive iterationscans)
	(list (quote qpir-iterate) seed recursive iterationscans)))

/* qpir-leaf: wraps a 7-tuple SQL block as an IR leaf.
The 7-tuple is (schema tables fields condition group having order limit offset).
After [lift_dep_joins_pass] the 7-tuple's expressions contain NO inner_select markers.
Compiled to nested scans by Layer 4 (build_queryplan_inner). */
(define qpir-leaf (lambda (seven-tuple)
	(list (quote qpir-leaf) seven-tuple)))

/* ==================== Kind detection ==================== */

/* qpir-kind: returns the operator kind symbol of a node, or nil for non-IR. */
(define qpir-kind (lambda (node) (match node
	(cons sym _) (match sym
		(symbol qpir-scan) (quote qpir-scan)
		(symbol qpir-select) (quote qpir-select)
		(symbol qpir-map) (quote qpir-map)
		(symbol qpir-groupby) (quote qpir-groupby)
		(symbol qpir-window) (quote qpir-window)
		(symbol qpir-join) (quote qpir-join)
		(symbol qpir-dep-join) (quote qpir-dep-join)
		(symbol qpir-union) (quote qpir-union)
		(symbol qpir-iterate) (quote qpir-iterate)
		(symbol qpir-leaf) (quote qpir-leaf)
		nil)
	nil)))

(define qpir-node? (lambda (node)
	(not (nil? (qpir-kind node)))))

/* ==================== Field accessors ==================== */

(define qpir-scan-schema (lambda (n) (nth n 1)))
(define qpir-scan-table  (lambda (n) (nth n 2)))

(define qpir-select-predicate (lambda (n) (nth n 1)))
(define qpir-select-child     (lambda (n) (nth n 2)))

(define qpir-map-projections (lambda (n) (nth n 1)))
(define qpir-map-child       (lambda (n) (nth n 2)))

(define qpir-groupby-keys   (lambda (n) (nth n 1)))
(define qpir-groupby-aggs   (lambda (n) (nth n 2)))
(define qpir-groupby-having (lambda (n) (nth n 3)))
(define qpir-groupby-child  (lambda (n) (nth n 4)))

(define qpir-window-partition    (lambda (n) (nth n 1)))
(define qpir-window-order        (lambda (n) (nth n 2)))
(define qpir-window-computations (lambda (n) (nth n 3)))
(define qpir-window-child        (lambda (n) (nth n 4)))

(define qpir-join-type      (lambda (n) (nth n 1)))
(define qpir-join-predicate (lambda (n) (nth n 2)))
(define qpir-join-left      (lambda (n) (nth n 3)))
(define qpir-join-right     (lambda (n) (nth n 4)))

(define qpir-dep-join-predicate (lambda (n) (nth n 1)))
(define qpir-dep-join-left      (lambda (n) (nth n 2)))
(define qpir-dep-join-right     (lambda (n) (nth n 3)))
(define qpir-dep-join-accessing (lambda (n) (nth n 4)))

(define qpir-union-order    (lambda (n) (nth n 1)))
(define qpir-union-limit    (lambda (n) (nth n 2)))
(define qpir-union-offset   (lambda (n) (nth n 3)))
(define qpir-union-branches (lambda (n) (nth n 4)))

(define qpir-iterate-seed           (lambda (n) (nth n 1)))
(define qpir-iterate-recursive      (lambda (n) (nth n 2)))
(define qpir-iterate-iterationscans (lambda (n) (nth n 3)))

(define qpir-leaf-7tuple (lambda (n) (nth n 1)))

/* ==================== Child enumeration ==================== */

/* qpir-children: returns the list of child IR nodes for a node, or '() for leaves.
Used by generic tree walks. */
(define qpir-children (lambda (node)
	(match (qpir-kind node)
		(quote qpir-scan)    '()
		(quote qpir-leaf)    '()
		(quote qpir-select)  (list (qpir-select-child node))
		(quote qpir-map)     (list (qpir-map-child node))
		(quote qpir-groupby) (list (qpir-groupby-child node))
		(quote qpir-window)  (list (qpir-window-child node))
		(quote qpir-join)    (list (qpir-join-left node) (qpir-join-right node))
		(quote qpir-dep-join) (list (qpir-dep-join-left node) (qpir-dep-join-right node))
		(quote qpir-union)   (qpir-union-branches node)
		(quote qpir-iterate) (list (qpir-iterate-seed node) (qpir-iterate-recursive node))
		'())))

/* ==================== Expression column references ==================== */

/* qpir-expr-column-refs: extract all (get_column tv ti col ci) references from
an expression tree. Returns a list of (tblvar col) pairs. Used to compute
the free-variable set F(N) of operators. */
(define qpir-expr-column-refs (lambda (expr) (match expr
	'((symbol get_column) tv _ col _) (if (nil? tv) '() (list (list tv col)))
	'((quote get_column)  tv _ col _) (if (nil? tv) '() (list (list tv col)))
	(cons sym args)
	(merge (qpir-expr-column-refs sym)
		(reduce args (lambda (acc a) (merge acc (qpir-expr-column-refs a))) '()))
	'())))

/* qpir-expr-list-refs: extract column refs from a list of expressions. */
(define qpir-expr-list-refs (lambda (exprs)
	(reduce (coalesceNil exprs '()) (lambda (acc e)
		(merge acc (qpir-expr-column-refs e))) '())))

/* qpir-assoc-list-refs: extract column refs from an assoc list of (name expr) pairs. */
(define qpir-assoc-list-refs (lambda (assoc)
	(reduce (coalesceNil assoc '()) (lambda (acc pair) (match pair
		'(_name expr) (merge acc (qpir-expr-column-refs expr))
		acc)) '())))

/* qpir-order-list-refs: extract column refs from order items '((expr dir) ...). */
(define qpir-order-list-refs (lambda (order)
	(reduce (coalesceNil order '()) (lambda (acc item) (match item
		'(expr _dir) (merge acc (qpir-expr-column-refs expr))
		acc)) '())))

/* ==================== Provided columns (A) and free variables (F) ==================== */

/* qpir-provided-aliases: returns the list of table aliases that the subtree at
`node` makes visible to its parent. Used to determine which column refs are
bound vs free. */
(define qpir-provided-aliases (lambda (node)
	(match (qpir-kind node)
		(quote qpir-scan) (list (qpir-scan-table node))
		(quote qpir-leaf) (match (qpir-leaf-7tuple node)
			'(_schema tables _fields _cond _group _having _order _limit _offset)
			(map (coalesceNil tables '()) (lambda (td) (match td
				'(alias _s _t _io _je) (if (nil? alias) (nth td 2) alias)
				_ nil)))
			'())
		(quote qpir-select)  (qpir-provided-aliases (qpir-select-child node))
		(quote qpir-map)     (merge_unique (list
			/* map adds projections but doesn't strip table aliases — they pass through */
			(qpir-provided-aliases (qpir-map-child node))))
		(quote qpir-groupby) (qpir-provided-aliases (qpir-groupby-child node))
		(quote qpir-window)  (qpir-provided-aliases (qpir-window-child node))
		(quote qpir-join)    (merge_unique (list
			(qpir-provided-aliases (qpir-join-left node))
			(qpir-provided-aliases (qpir-join-right node))))
		(quote qpir-dep-join) (merge_unique (list
			(qpir-provided-aliases (qpir-dep-join-left node))
			(qpir-provided-aliases (qpir-dep-join-right node))))
		(quote qpir-union)   (if (or (nil? (qpir-union-branches node))
			(equal? (qpir-union-branches node) '()))
			'()
			(qpir-provided-aliases (car (qpir-union-branches node))))
		(quote qpir-iterate) (qpir-provided-aliases (qpir-iterate-seed node))
		'())))

/* qpir-own-refs: column refs used DIRECTLY by this node (not by its children).
Per operator: select uses predicate; map uses projections; etc. */
(define qpir-own-refs (lambda (node)
	(match (qpir-kind node)
		(quote qpir-scan)     '()
		(quote qpir-leaf)     (match (qpir-leaf-7tuple node)
			'(_schema _tables fields cond _group _having order _limit _offset)
			(merge
				(qpir-assoc-list-refs fields)
				(qpir-expr-column-refs (coalesceNil cond true))
				(qpir-order-list-refs order))
			'())
		(quote qpir-select)   (qpir-expr-column-refs (qpir-select-predicate node))
		(quote qpir-map)      (qpir-assoc-list-refs (qpir-map-projections node))
		(quote qpir-groupby)  (merge
			(qpir-expr-list-refs (qpir-groupby-keys node))
			(qpir-assoc-list-refs (qpir-groupby-aggs node))
			(qpir-expr-column-refs (coalesceNil (qpir-groupby-having node) true)))
		(quote qpir-window)   (merge
			(qpir-expr-list-refs (qpir-window-partition node))
			(qpir-order-list-refs (qpir-window-order node)))
		(quote qpir-join)     (qpir-expr-column-refs (qpir-join-predicate node))
		(quote qpir-dep-join) (qpir-expr-column-refs (qpir-dep-join-predicate node))
		(quote qpir-union)    (qpir-order-list-refs (qpir-union-order node))
		(quote qpir-iterate)  '()
		'())))

/* qpir-free-vars: F(N) — the set of (tblvar col) pairs referenced in the subtree
rooted at N that are NOT provided by N's children. Per BTW2025 §2.1.

An IR tree with F(root) ≠ ∅ cannot be lowered to physical (unbound refs).
After [unnest_pass], the root's F must be empty. */
(define qpir-free-vars (lambda (node) (begin
	(define provided (qpir-provided-aliases node))
	(define all-refs (merge (qpir-own-refs node)
		(reduce (qpir-children node) (lambda (acc c)
			(merge acc (qpir-free-vars c))) '())))
	(filter all-refs (lambda (ref) (match ref
		'(tv _col) (not (has? provided tv))
		false))))))

/* ==================== Pretty printer ==================== */

/* qpir-show: produce a compact human-readable string of an IR tree.
For debugging/snapshot tests. Limits expression detail to keep output readable. */
(define qpir-show (lambda (node) (qpir-show-indent node 0)))

(define qpir-show-indent (lambda (node depth) (begin
	(define pad (qpir-spaces depth))
	(match (qpir-kind node)
		(quote qpir-scan)
		(concat pad "(qpir-scan " (qpir-scan-schema node) "." (qpir-scan-table node) ")")
		(quote qpir-leaf)
		(concat pad "(qpir-leaf <7tuple>)")
		(quote qpir-select)
		(concat pad "(qpir-select " (qpir-short-expr (qpir-select-predicate node)) "\n"
			(qpir-show-indent (qpir-select-child node) (+ depth 2)) ")")
		(quote qpir-map)
		(concat pad "(qpir-map <" (string (count (qpir-map-projections node))) " projs>\n"
			(qpir-show-indent (qpir-map-child node) (+ depth 2)) ")")
		(quote qpir-groupby)
		(concat pad "(qpir-groupby keys=" (string (count (qpir-groupby-keys node)))
			" aggs=" (string (count (qpir-groupby-aggs node))) "\n"
			(qpir-show-indent (qpir-groupby-child node) (+ depth 2)) ")")
		(quote qpir-window)
		(concat pad "(qpir-window\n"
			(qpir-show-indent (qpir-window-child node) (+ depth 2)) ")")
		(quote qpir-join)
		(concat pad "(qpir-join " (string (qpir-join-type node)) " " (qpir-short-expr (qpir-join-predicate node)) "\n"
			(qpir-show-indent (qpir-join-left node) (+ depth 2)) "\n"
			(qpir-show-indent (qpir-join-right node) (+ depth 2)) ")")
		(quote qpir-dep-join)
		(concat pad "(qpir-dep-join " (qpir-short-expr (qpir-dep-join-predicate node)) "\n"
			(qpir-show-indent (qpir-dep-join-left node) (+ depth 2)) "\n"
			(qpir-show-indent (qpir-dep-join-right node) (+ depth 2)) ")")
		(quote qpir-union)
		(concat pad "(qpir-union " (string (count (qpir-union-branches node))) " branches)")
		(quote qpir-iterate)
		(concat pad "(qpir-iterate\n"
			(qpir-show-indent (qpir-iterate-seed node) (+ depth 2)) "\n"
			(qpir-show-indent (qpir-iterate-recursive node) (+ depth 2)) ")")
		(concat pad "<non-ir>")))))

(define qpir-spaces (lambda (n)
	(if (<= n 0) "" (concat " " (qpir-spaces (- n 1))))))

(define qpir-short-expr (lambda (expr)
	(if (nil? expr) "nil" (begin
		(define s (string expr))
		(if (> (strlen s) 40) (concat (substr s 0 37) "...") s)))))
