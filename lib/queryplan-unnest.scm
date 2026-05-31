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
== Layer 3 — unnest_pass: BTW2025 §3 holistic top-down decorrelation ==

Input: a Layer-1 IR tree (post-lift) that may contain qpir-dep-join nodes.
Output: an IR tree with NO qpir-dep-join — F(root) = ∅ — ready for join
reordering (Day 6) and physical lowering (Day 7).

Per the FAQ:
  - "why top-down and not bottom-up across nested dependent joins? bottom-up
     NK15 re-runs push-down for each inner dep-join against the full cross
     product of all outer domains above it, which blows up with nesting depth
     (the crash.sql case motivating BTW2025). top-down unnesting keeps a
     parent-chained UnnestingInfo so nested dep-joins share their outerRefs/
     cclasses/repr and are resolved in one pass."
  - "every untangle pass MUST try simple first — general unnesting adds a
     D-join and is strictly more expensive than a predicate pull-up."
  - "trivial dependent join: after simple unnesting, if no operator on the
     right side still references left-side columns (A(L) ∩ F(R) = ∅), convert
     ▷◁ directly into a regular ⋈ (or cross product when there is no
     predicate). no domain, no anti-pass, nothing — it is identity."

This module implements those algorithms with the qpir IR.

Module structure (this commit focuses on the foundation):

  qpu-walk-with-path : tree walker that tracks ancestor path; used for the
                       annotate sub-pass and for LCA computation
  qpu-collect-column-providers : for every qpir-leaf/scan in the tree, record
                       (path, provided-aliases)
  qpu-collect-column-accessors : for every column-ref expression in the tree,
                       record (accessor-path, (tblvar col))
  qpu-lca-path : longest common prefix of two paths
  annotate_dep_joins : populates each qpir-dep-join's `accessing` slot with
                       the list of paths of accessor ops that read columns
                       provided by the dep-join's left subtree

  qpu-rewrite-outer-refs : per-operator §3.3 rule applicator
  simple_djoin_elimination : §3.2 fig 3 — pure predicate pull-up
  djoin_elimination : the general algorithm
  unnest_pass : the driver

Phase 1 of this module (this commit):
  - Tree-walking infrastructure (qpu-walk-with-path, etc.)
  - annotate_dep_joins fully implemented + snapshot tests
  - Trivial elimination: dep-join with empty accessing → qpir-join

Subsequent phases:
  - simple_djoin_elimination (select/map pull-up)
  - per-operator §3.3 rules (groupby, scan, join)
  - cclasses + repr for outer-ref substitution
  - full driver

Per FAQ §1: unhandled shapes error loudly. No fallback paths.
*/

/* ==================== Path-tracking tree walker ==================== */

/* A "path" is a list of node references from root to current node (root
first, current last). Two paths share an ancestor iff one's prefix equals
the other's prefix up through that ancestor. We use object identity (via
the immutable list values themselves) to test equality. */

/* qpu-walk-with-path — recursively walk a qpir tree, calling
   (visitor node path) on each node. `path` is a list ending with `node`
   itself. visitor returns nil; this walker is for collection, not rewrite. */
(define qpu-walk-with-path (lambda (node visitor)
	(qpu-walk-helper node (list node) visitor)))

(define qpu-walk-helper (lambda (node path visitor) (begin
	(visitor node path)
	(define children (qpir-children node))
	(reduce children (lambda (acc c) (begin
		(qpu-walk-helper c (merge path (list c)) visitor)
		acc)) nil))))

/* qpu-path-equal? — true if two paths are identical (same length, same
node-list elements). Since qpir nodes are immutable lists, equal? compares
by value — two nodes with the same shape are considered equal. For our LCA
this matters: if two qpir-leaf-with-same-7tuple appear in different places,
they're "the same" by equal?. To avoid this we'd need explicit IDs; for now
the conservative behavior is sufficient because lift produces structurally
unique nodes. */
(define qpu-path-equal? (lambda (p1 p2)
	(and (equal? (count p1) (count p2)) (equal? p1 p2))))

/* qpu-lca-path — longest common prefix of two paths.
Returns the shared prefix list (may be empty). */
(define qpu-lca-path (lambda (p1 p2)
	(qpu-lca-helper p1 p2 (list))))

(define qpu-lca-helper (lambda (p1 p2 acc)
	(if (or (equal? (count p1) 0) (equal? (count p2) 0))
		acc
		(if (equal? (car p1) (car p2))
			(qpu-lca-helper (cdr p1) (cdr p2) (merge acc (list (car p1))))
			acc))))

/* qpu-path-last — last element of a path = the node itself. */
(define qpu-path-last (lambda (p)
	(if (equal? (count p) 0) nil (nth p (- (count p) 1)))))

/* ==================== Annotate dep-joins ==================== */

/* qpu-collect-providers — walk the tree, return a list of (path provided-aliases)
for every node whose qpir-provided-aliases is non-empty. Used to find which
operator provides each column reference. */
(define qpu-collect-providers (lambda (tree)
	(begin
		(define acc (newsession))
		(acc "list" (list))
		(qpu-walk-with-path tree (lambda (node path) (begin
			(define provided (qpir-provided-aliases node))
			(if (> (count provided) 0)
				(acc "list" (merge (acc "list") (list (list path provided))))
				nil))))
		(acc "list"))))

/* qpu-collect-accessors — walk the tree, for every node return a list of
(accessor-path ref) for each column ref in its OWN refs (not children's).
Used by annotate_dep_joins. */
(define qpu-collect-accessors (lambda (tree)
	(begin
		(define acc (newsession))
		(acc "list" (list))
		(qpu-walk-with-path tree (lambda (node path)
			(reduce (qpir-own-refs node) (lambda (a ref) (begin
				(acc "list" (merge (acc "list") (list (list path ref))))
				a)) nil)))
		(acc "list"))))

/* qpu-find-provider-path — given a column ref (tblvar col), find the
LONGEST provider path whose provided-aliases contains tblvar. (Longest =
deepest in the tree, closest to the accessor.) */
(define qpu-find-provider-path (lambda (ref providers) (begin
	(define tv (nth ref 0))
	(define matches (filter providers (lambda (p) (match p
		'(path provided) (has? provided tv)
		false))))
	(if (equal? (count matches) 0)
		nil  /* unresolved ref — free at the root, won't be in any subtree */
		(begin
			/* Pick longest path */
			(define best (car matches))
			(reduce (cdr matches) (lambda (b candidate)
				(if (> (count (nth candidate 0)) (count (nth b 0)))
					candidate b))
				best))))))

/* qpu-is-dep-join-path-elem? — true if a path element is a qpir-dep-join. */
(define qpu-is-dep-join-path-elem? (lambda (elem)
	(equal? (qpir-kind elem) (quote qpir-dep-join))))

/* qpu-find-enclosing-dep-join — given an accessor path and the provider
path for the column it reads, find the dep-join (if any) on the path
between provider and accessor. Returns the dep-join node, or nil if none.

The LCA(accessor, provider) is computed; if it's NOT the immediate provider,
that means there's a node between them — when that intermediate node is the
dep-join itself, the accessor sits in the dep-join's right subtree and reads
columns provided by the dep-join's left subtree, which IS the §3.1 condition
for "non-trivial dep-join". */
(define qpu-find-enclosing-dep-join (lambda (accessor-path provider-path) (begin
	(define lca (qpu-lca-path accessor-path provider-path))
	/* The dep-join we want is the last element of the LCA — if that's a
	   dep-join. If LCA's last is the provider itself (provider-path is prefix
	   of accessor-path), there is no dep-join between them. */
	(if (equal? (count lca) 0)
		nil
		(begin
			(define last-elem (qpu-path-last lca))
			(if (qpu-is-dep-join-path-elem? last-elem)
				last-elem
				nil))))))

/* qpu-build-accessing-map — produce an assoc from dep-join-node →
list-of-accessor-paths. (Multiple accessor paths may point to the same
dep-join.) */
(define qpu-build-accessing-map (lambda (tree) (begin
	(define providers (qpu-collect-providers tree))
	(define accessors (qpu-collect-accessors tree))
	(define out (newsession))
	(out "list" (list))
	(reduce accessors (lambda (acc pair) (match pair
		'(accessor-path ref) (begin
			(define provider-pair (qpu-find-provider-path ref providers))
			(if (nil? provider-pair) acc
				(begin
					(define provider-path (nth provider-pair 0))
					(define dj (qpu-find-enclosing-dep-join accessor-path provider-path))
					(if (nil? dj) acc
						(begin
							/* Append (dj accessor-path) to out "list". Keep duplicates
							   for now; consumers may dedupe. */
							(out "list" (merge (out "list") (list (list dj accessor-path))))
							acc)))))
		acc)) nil)
	(out "list"))))

/* ==================== Tree rewrite primitive ==================== */

/* qpu-rewrite-node — return a new node with updated children list. Used by
all rewrite operations on the immutable IR. The kind tag and own slots stay
the same; only the child slots are replaced. */
(define qpu-rewrite-children (lambda (node new-children)
	(match (qpir-kind node)
		(quote qpir-scan) node
		(quote qpir-leaf) node
		(quote qpir-select)
		(qpir-select (qpir-select-predicate node) (nth new-children 0))
		(quote qpir-map)
		(qpir-map (qpir-map-projections node) (nth new-children 0))
		(quote qpir-groupby)
		(qpir-groupby (qpir-groupby-keys node) (qpir-groupby-aggs node)
			(qpir-groupby-having node) (nth new-children 0))
		(quote qpir-window)
		(qpir-window (qpir-window-partition node) (qpir-window-order node)
			(qpir-window-computations node) (nth new-children 0))
		(quote qpir-join)
		(qpir-join (qpir-join-type node) (qpir-join-predicate node)
			(nth new-children 0) (nth new-children 1)
			(qpir-join-rhs-alias node))
		(quote qpir-dep-join)
		(qpir-dep-join (qpir-dep-join-predicate node)
			(nth new-children 0) (nth new-children 1)
			(qpir-dep-join-accessing node)
			(qpir-dep-join-rhs-alias node))
		(quote qpir-union)
		(qpir-union (qpir-union-order node) (qpir-union-limit node)
			(qpir-union-offset node) new-children)
		(quote qpir-iterate)
		(qpir-iterate (nth new-children 0) (nth new-children 1)
			(qpir-iterate-iterationscans node))
		node)))

/* qpu-map-tree — apply fn to every node bottom-up (children first).
fn receives the node (with already-rewritten children) and returns a new node. */
(define qpu-map-tree (lambda (node fn) (begin
	(define new-children (map (qpir-children node) (lambda (c)
		(qpu-map-tree c fn))))
	(fn (qpu-rewrite-children node new-children)))))

/* ==================== Trivial elimination ==================== */

/* qpu-trivial-eliminate — converts dep-joins whose right side has NO free
variables referencing the left side into regular inner joins.

Per FAQ trivial-dep-join: "after simple unnesting, if no operator on the
right side still references left-side columns (A(L) ∩ F(R) = ∅), convert
▷◁ directly into a regular ⋈". This sub-pass handles the case BEFORE any
simple-elimination work — for dep-joins that lift_dep_joins produced with
self-contained right sides (e.g. uncorrelated subqueries that snuck through). */
(define qpu-trivial-eliminate (lambda (node)
	(if (not (equal? (qpir-kind node) (quote qpir-dep-join)))
		node
		(begin
			(define left (qpir-dep-join-left node))
			(define right (qpir-dep-join-right node))
			(define left-aliases (qpir-provided-aliases left))
			(define right-free (qpir-free-vars right))
			(define correlated-free (filter right-free (lambda (ref) (match ref
				'(tv col) (has? left-aliases tv)
				false))))
			(if (equal? (count correlated-free) 0)
				/* Trivial: right doesn't reference left → INNER join.
				   For an UNCORRELATED subquery the right side always produces
				   exactly one row (COUNT returns 0, SUM returns NULL, etc. —
				   never empty). INNER and LEFT are semantically equivalent
				   here. INNER is what build_queryplan_inner reliably handles
				   for `LEFT JOIN ON TRUE` (the joinExpr=true case the
				   physical layer would degenerate to). */
				(qpir-join (quote inner) (qpir-dep-join-predicate node) left right
					(qpir-dep-join-rhs-alias node))
				node)))))

(define qpu-trivial-eliminate-tree (lambda (tree)
	(qpu-map-tree tree qpu-trivial-eliminate)))

/* ==================== Annotate (public API) ==================== */

/* annotate_dep_joins — public name for the §3.1 annotation pass.
Returns a session whose "map" key holds a list of
(dep-join-node accessor-path) pairs. Multiple entries may share a dep-join.

Per BTW2025 §3.1: dep-joins with empty accessing are trivial and can be
converted directly via qpu-trivial-eliminate. Dep-joins with non-empty
accessing need the general §3.2 elimination algorithm. */
(define annotate_dep_joins (lambda (tree) (begin
	(define pairs (qpu-build-accessing-map tree))
	(define session (newsession))
	(session "map" pairs)
	session)))

/* qpu-accessing-of — given the annotation session from annotate_dep_joins
and a dep-join node, return the list of accessor paths recorded against it. */
(define qpu-accessing-of (lambda (annotation dep-join-node) (begin
	(define pairs (annotation "map"))
	(reduce pairs (lambda (acc pair) (match pair
		'(dj accessor-path) (if (equal? dj dep-join-node)
			(merge acc (list accessor-path))
			acc)
		acc)) (list)))))

/* ==================== BTW2025 §3.3 groupby rule ==================== */

/* qpu-push-outer-refs-into-groupby — the FAQ §33 / BTW2025 §3.3 rule:
   D ⋈ᵈ Γ_A;agg(T) → Γ_{A∪A(D);agg}(D ⋈ T)

   Adds outer-ref expressions to the groupby's keys list so each combination
   of outer values becomes its own group. Returns a new groupby node;
   does NOT mutate the input.

   outer-ref-exprs is the list of (get_column tv col) expressions that
   correspond to the outer refs the dep-join needs to bind. They become
   new GROUP BY keys.

   Used by the unnest_pass when traversing a dep-join whose right subtree
   contains a qpir-groupby. */
(define qpu-push-outer-refs-into-groupby (lambda (gb outer-ref-exprs)
	(if (not (equal? (qpir-kind gb) (quote qpir-groupby)))
		(error "qpu-push-outer-refs-into-groupby: input is not qpir-groupby")
		(qpir-groupby
			(merge (qpir-groupby-keys gb) outer-ref-exprs)
			(qpir-groupby-aggs gb)
			(qpir-groupby-having gb)
			(qpir-groupby-child gb)))))

/* qpu-bottom-left-aliases — walk down qpir-dep-join / qpir-join chains in
the LEFT subtree to find the ORIGINAL outer's provided aliases. For chained
sibling dep-joins (e.g. two scalar subselects in the same outer's SELECT
list lift to (dep-join sq_19 (dep-join sq_18 outer-leaf sub_18) sub_19)),
this returns just outer-leaf's aliases — NOT sub_18's k and sq_18 which
would otherwise pollute sq_19's outer-aliases and break cclasses-based
substitution.

Per FAQ §42: nested/chained dep-joins must NOT inherit sibling-introduced
aliases as outer. The original outer scope is the bottom of the left
chain. */
(define qpu-bottom-left-aliases (lambda (node)
	(match (qpir-kind node)
		(quote qpir-dep-join) (qpu-bottom-left-aliases (qpir-dep-join-left node))
		(quote qpir-join)     (qpu-bottom-left-aliases (qpir-join-left node))
		(quote qpir-select)   (qpu-bottom-left-aliases (qpir-select-child node))
		(quote qpir-map)      (qpu-bottom-left-aliases (qpir-map-child node))
		(quote qpir-groupby)  (qpu-bottom-left-aliases (qpir-groupby-child node))
		(quote qpir-window)   (qpu-bottom-left-aliases (qpir-window-child node))
		(qpir-provided-aliases node))))

/* qpu-collect-outer-refs — for a dep-join, return the list of get_column
expressions in the RIGHT subtree that reference columns from the LEFT
subtree's ORIGINAL OUTER scope. */
(define qpu-collect-outer-refs (lambda (dj) (begin
	(define left-aliases (qpu-bottom-left-aliases (qpir-dep-join-left dj)))
	(define right-free (qpir-free-vars (qpir-dep-join-right dj)))
	(define outer-refs (filter right-free (lambda (ref) (match ref
		'(tv col) (has? left-aliases tv)
		false))))
	(map outer-refs (lambda (ref) (match ref
		'(tv col) (list (quote get_column) tv false col false)
		ref))))))

/* qpu-collect-outer-refs-with-aliases — variant used by the top-down driver
that takes an explicit outer-aliases set (the union of this dep-join's
immediate-left aliases + ALL ancestor dep-join lefts). Needed for FAQ §42
doubly-nested correlation: an inner dep-join's right may reference an
OUTERMOST table that isn't in its own immediate-left subtree. */
(define qpu-collect-outer-refs-with-aliases (lambda (dj outer-aliases) (begin
	(define right-free (qpir-free-vars (qpir-dep-join-right dj)))
	(define outer-refs (filter right-free (lambda (ref) (match ref
		'(tv col) (has? outer-aliases tv)
		false))))
	(map outer-refs (lambda (ref) (match ref
		'(tv col) (list (quote get_column) tv false col false)
		ref))))))

/* ==================== Predicate splitting (FAQ §3.2 simple-elim) ==================== */

/* qpu-and-conjuncts — flatten an AND-tree into a list of conjuncts.
A non-AND expression yields a single-element list. Constant `true` yields
the empty list (no constraint). */
(define qpu-and-conjuncts (lambda (expr)
	(if (or (nil? expr) (equal? expr true) (equal? expr (quote true)))
		(list)
		(if (qpu-expr-is-and? expr)
			(qpu-flatten-and-args (cdr expr))
			(list expr)))))

/* qpu-expr-is-and? — true if expr is a list whose head is the `and` symbol. */
(define qpu-expr-is-and? (lambda (expr) (match expr
	(cons head args) (match head
		(symbol and)        true
		(quote and)         true
		false)
	false)))

(define qpu-flatten-and-args (lambda (args)
	(reduce (coalesceNil args (list)) (lambda (acc a)
		(merge acc (qpu-and-conjuncts a))) (list))))

/* qpu-and-from-conjuncts — rebuild an AND expression from a conjunct list.
Empty → true; one element → bare; >1 → (and a b c …).
Defensive against nil input (in this dialect (list) can evaluate to nil and
count panics on nil). */
(define qpu-and-from-conjuncts (lambda (cs) (begin
	(define csl (coalesceNil cs (list)))
	(if (or (nil? csl) (equal? (count csl) 0)) true
		(if (equal? (count csl) 1) (nth csl 0)
			(cons (quote and) csl))))))

/* qpu-expr-references-aliases? — true if expr contains any (get_column tv …)
whose tv is in `aliases`. Used to identify outer-correlation conjuncts. */
(define qpu-expr-references-aliases? (lambda (expr aliases)
	(begin
		(define refs (qpir-expr-column-refs expr))
		(> (count (filter refs (lambda (ref) (match ref
			'(tv col) (has? aliases tv)
			false)))) 0))))

/* qpu-split-predicate — split a predicate into (outer-conjuncts pure-conjuncts).
Outer-conjuncts reference any alias in outer-aliases; pure-conjuncts don't.
Returns the pair (outer-pred-or-true pure-pred-or-true) as already-ANDed
expressions ready for use as dep-join condition and remaining select. */
(define qpu-split-predicate (lambda (pred outer-aliases) (begin
	(define cs (qpu-and-conjuncts pred))
	(define outer-cs (filter cs (lambda (c)
		(qpu-expr-references-aliases? c outer-aliases))))
	(define pure-cs (filter cs (lambda (c)
		(not (qpu-expr-references-aliases? c outer-aliases)))))
	(list (qpu-and-from-conjuncts outer-cs) (qpu-and-from-conjuncts pure-cs)))))

/* ==================== cclasses + repr (BTW2025 §3.2, FAQ §40-§41) ==================== */

/* cclasses are equivalence classes over column references (tv col). Built up
as the unnest walker observes equality predicates `a = b` in qpir-select.
repr maps an OUTER reference to its EQUIVALENT INNER reference (when one
exists in the same class). The right-side walker uses repr to substitute
outer-refs in groupby keys and other operator slots so the unnested tree
no longer references outer columns from inside the right subtree.

Representation: a session whose "classes" key holds a list of class-lists,
each class being a list of `(tv col)` references. Singleton classes are
implied (any ref not appearing in any class is in its own singleton). */

(define qpu-make-cclasses (lambda () (begin
	(define cc (newsession))
	(cc "classes" (list))
	cc)))

/* qpu-cc-find-class — return the class (list of refs) containing the given
ref, or nil if the ref is not yet in any non-singleton class. */
(define qpu-cc-find-class (lambda (cc ref) (begin
	(define classes (cc "classes"))
	(reduce classes (lambda (found c)
		(if (and (nil? found) (has? c ref)) c found))
		nil))))

/* qpu-cc-union — add the equivalence ref-a ~ ref-b to cclasses. If either
ref is already in a class, merge into a single class. */
(define qpu-cc-union (lambda (cc ref-a ref-b) (begin
	(define classes (cc "classes"))
	(define class-a (qpu-cc-find-class cc ref-a))
	(define class-b (qpu-cc-find-class cc ref-b))
	(define merged-class
		(if (nil? class-a)
			(if (nil? class-b) (list ref-a ref-b) (merge class-b (list ref-a)))
			(if (nil? class-b) (merge class-a (list ref-b))
				(merge class-a class-b))))
	(define remaining (filter classes (lambda (c)
		(and (not (equal? c class-a)) (not (equal? c class-b))))))
	(cc "classes" (merge remaining (list (qpu-dedupe-list merged-class)))))))

/* qpu-dedupe-list — remove duplicate elements (using equal?). */
(define qpu-dedupe-list (lambda (xs)
	(reduce xs (lambda (acc x)
		(if (has? acc x) acc (merge acc (list x))))
		(list))))

/* qpu-cc-add-from-predicate — for every equality conjunct `(equal?? a b)` in
predicate with BOTH a and b being column references, add the cclass. Ignores
non-equality predicates and equalities with non-column operands. */
(define qpu-cc-add-from-predicate (lambda (cc predicate)
	(reduce (qpu-and-conjuncts predicate) (lambda (acc conj) (begin
		(define refs (qpu-cc-equality-refs conj))
		(if (nil? refs) acc
			(begin
				(qpu-cc-union cc (nth refs 0) (nth refs 1))
				acc)))) nil)))

(define qpu-cc-equality-refs (lambda (expr) (match expr
	(cons head args) (begin
		(define is-eq (match head
			(symbol equal??)   true
			(quote equal??)    true
			'(quote equal??)   true
			'equal??           true
			(symbol equal?)    true
			(quote equal?)     true
			'(quote equal?)    true
			'equal?            true
			false))
		(if (and is-eq (equal? (count args) 2))
			(begin
				(define a (nth args 0))
				(define b (nth args 1))
				(define refs-a (qpir-expr-column-refs a))
				(define refs-b (qpir-expr-column-refs b))
				(if (and (equal? (count refs-a) 1) (equal? (count refs-b) 1)
						(equal? (qpu-strip-expr-to-col-ref? a) true)
						(equal? (qpu-strip-expr-to-col-ref? b) true))
					(list (nth refs-a 0) (nth refs-b 0))
					nil))
			nil))
	nil)))

/* qpu-strip-expr-to-col-ref? — true when expr is a bare (get_column …) form
with no extra wrapping. Equalities between non-column expressions don't
contribute to cclasses (the equivalence wouldn't necessarily preserve under
substitution). */
(define qpu-strip-expr-to-col-ref? (lambda (expr) (match expr
	(cons head args) (match head
		(symbol get_column)     true
		(quote get_column)      true
		'(quote get_column)     true
		'get_column             true
		false)
	false)))

/* qpu-cc-build-repr — given cclasses and a list of outer-aliases, build a
repr map mapping outer-refs to their inner equivalents. An outer ref
`(tv col)` (where tv ∈ outer-aliases) maps to a ref in its class
whose tv is NOT in outer-aliases.

Preference order for picking the inner ref (FAQ §35 + scope-aware lowering):
  1. sq_X-aliased refs (rhs-aliases from dep-joins, which are stable at the
     scope where the cclass was assembled)
  2. Any other inner ref (base table column, etc.)

Picking sq_X over deep-nested base columns matters for doubly-nested cases:
the inner dep-join's projection sq_X.__kt_col is reachable AT the outer
scope where the cc-join-cond lives, while the base column `t.col` is buried
inside sq_X's derived (FAQ §38 scope graph).

Returns a session whose "map" key holds an assoc list of (outer-ref . inner-ref). */
(define qpu-cc-pick-inner-ref (lambda (inners) (begin
	/* Prefer sq_X-aliased refs over others. */
	(define sq-refs (filter inners (lambda (ref) (match ref
		'(tv col) (and (string? tv) (>= (strlen tv) 3)
			(equal? (substr tv 0 3) "sq_"))
		false))))
	(if (> (count sq-refs) 0) (nth sq-refs 0) (nth inners 0)))))

(define qpu-cc-build-repr (lambda (cc outer-aliases) (begin
	(define repr (newsession))
	(repr "map" (list))
	(define classes (cc "classes"))
	(reduce classes (lambda (acc c) (begin
		/* For each class, find outer refs and the preferred inner ref */
		(define outers (filter c (lambda (ref) (match ref
			'(tv col) (has? outer-aliases tv)
			false))))
		(define inners (filter c (lambda (ref) (match ref
			'(tv col) (not (has? outer-aliases tv))
			false))))
		(if (and (> (count outers) 0) (> (count inners) 0))
			(begin
				(define inner-ref (qpu-cc-pick-inner-ref inners))
				(reduce outers (lambda (a outer-ref) (begin
					(repr "map" (merge (repr "map") (list (list outer-ref inner-ref))))
					a)) nil)) nil)
		acc)) nil)
	repr)))

(define qpu-repr-lookup (lambda (repr ref) (begin
	(define entries (repr "map"))
	(reduce entries (lambda (found entry)
		(if (and (nil? found) (equal? (nth entry 0) ref))
			(nth entry 1)
			found))
		nil))))

/* qpu-substitute-expr — walk expr, replacing every (get_column tv ti col ci)
whose (tv col) appears as a key in repr with the substituted (tv' col') form. */
(define qpu-substitute-expr (lambda (expr repr) (match expr
	'((symbol get_column) tv ti col ci)
	(qpu-substitute-col-ref expr (list tv col) repr)
	'((quote get_column)  tv ti col ci)
	(qpu-substitute-col-ref expr (list tv col) repr)
	(cons head args) (cons head (map (coalesceNil args (list))
		(lambda (a) (qpu-substitute-expr a repr))))
	expr)))

(define qpu-substitute-col-ref (lambda (expr ref repr) (begin
	(define replacement (qpu-repr-lookup repr ref))
	(if (nil? replacement)
		expr
		(list (quote get_column)
			(nth replacement 0) false (nth replacement 1) false)))))

(define qpu-substitute-exprs (lambda (exprs repr)
	(map (coalesceNil exprs (list)) (lambda (e) (qpu-substitute-expr e repr)))))

/* ==================== Right-side walker (§3.3 rules) ==================== */

/* qpu-unnest-right — walk a dep-join's RIGHT subtree top-down applying the
BTW2025 §3.3 per-operator rules. Returns a pair (new-right join-predicate)
where:
  - new-right is the rewritten subtree (no more correlation to outer)
  - join-predicate is the AND of all outer-correlation conjuncts extracted
    from the right; becomes the dep-join's predicate after conversion.

This is the simple variant — uses NO cclasses-based substitution yet. The
correlation conjuncts that get extracted to the join condition are kept
as-is (e.g. `(equal?? pi.k po.k)`) rather than substituted via repr. The
groupby rule adds the outer-ref expressions to keys directly so each outer
combo gets its own group. */
/* qpu-collect-cclasses — first pass: walk the right subtree and accumulate
all column-equality predicates from qpir-select operators into cclasses. */
(define qpu-collect-cclasses (lambda (node cc)
	(match (qpir-kind node)
		(quote qpir-leaf)    nil
		(quote qpir-scan)    nil
		(quote qpir-select)  (begin
			(qpu-cc-add-from-predicate cc (qpir-select-predicate node))
			(qpu-collect-cclasses (qpir-select-child node) cc))
		(quote qpir-map)     (qpu-collect-cclasses (qpir-map-child node) cc)
		(quote qpir-groupby) (qpu-collect-cclasses (qpir-groupby-child node) cc)
		(quote qpir-window)  (qpu-collect-cclasses (qpir-window-child node) cc)
		(quote qpir-join)    (begin
			(qpu-cc-add-from-predicate cc (qpir-join-predicate node))
			(qpu-collect-cclasses (qpir-join-left node) cc)
			(qpu-collect-cclasses (qpir-join-right node) cc))
		nil)))

/* qpu-unnest-right — walks the right subtree TOP-DOWN applying §3.3 rules
with cclasses substitution. Returns (new-node join-pred) where new-node has
NO references to outer-aliases (all substituted to inner equivalents) and
join-pred contains the IS NOT DISTINCT FROM conjuncts that go on the
converted dep-join. */
(define qpu-unnest-right (lambda (node outer-aliases outer-ref-exprs repr)
	(match (qpir-kind node)
		(quote qpir-leaf) (begin
			(define leaf-tuple (qpir-leaf-7tuple node))
			(define cond (qpp-tuple-condition leaf-tuple))
			(if (qpu-expr-references-aliases? cond outer-aliases)
				(error "qpu-unnest-right: qpir-leaf has WHERE referencing outer columns. lift phase 5 should have hoisted it into a qpir-select wrapper.")
				(list node true)))
		(quote qpir-scan) (list node true)

		(quote qpir-select) (begin
			(define raw-pred (qpir-select-predicate node))
			(define sub-pred (qpu-substitute-expr raw-pred repr))
			(define child-result (qpu-unnest-right (qpir-select-child node)
				outer-aliases outer-ref-exprs repr))
			(define child-new (nth child-result 0))
			(define child-join (nth child-result 1))
			(define split (qpu-split-predicate sub-pred outer-aliases))
			(define outer-pred (nth split 0))
			(define pure-pred (nth split 1))
			/* outer-pred should be empty/true after substitution — the
			   equalities that put refs in cclasses get replaced by tautologies.
			   Any residual outer-pred is a non-equality correlation; those
			   need the IS NOT DISTINCT FROM treatment per FAQ §41. */
			(define combined-join (qpu-and-from-conjuncts
				(merge (qpu-and-conjuncts child-join) (qpu-and-conjuncts outer-pred))))
			/* pure-pred may contain trivial tautologies like (equal?? x x) after
			   substitution — strip them. */
			(define simplified-pure (qpu-simplify-predicate pure-pred))
			(define new-node
				(if (or (nil? simplified-pure) (equal? simplified-pure true))
					child-new
					(qpir-select simplified-pure child-new)))
			(list new-node combined-join))

		(quote qpir-map) (begin
			(define new-projs (qpu-substitute-map-projections
				(qpir-map-projections node) repr))
			(define child-result (qpu-unnest-right (qpir-map-child node)
				outer-aliases outer-ref-exprs repr))
			(define child-new (nth child-result 0))
			(define child-join (nth child-result 1))
			(list (qpir-map new-projs child-new) child-join))

		(quote qpir-groupby) (begin
			(define child-result (qpu-unnest-right (qpir-groupby-child node)
				outer-aliases outer-ref-exprs repr))
			(define child-new (nth child-result 0))
			(define child-join (nth child-result 1))
			/* §3.3 / FAQ §33: push outer refs into keys. With cclasses, push
			   the SUBSTITUTED outer-ref-exprs so the keys reference inner
			   columns the child actually provides. Dedupe: multiple outer
			   refs may resolve to the same inner col via cclasses (e.g.
			   `WHERE d.did=e.did AND d.did=e.eid` → both project d.did);
			   without dedupe the keytable build fails with
			   "column ... already exists". */
			(define sub-outer-refs (qpu-substitute-exprs outer-ref-exprs repr))
			(define sub-aggs (qpu-substitute-map-projections
				(qpir-groupby-aggs node) repr))
			(define sub-having (qpu-substitute-expr
				(coalesceNil (qpir-groupby-having node) true) repr))
			(define existing-keys (coalesceNil (qpir-groupby-keys node) '()))
			(define new-keys (reduce sub-outer-refs (lambda (acc k)
				(if (has? acc k) acc (merge acc (list k))))
				existing-keys))
			(define final-having (if (equal? sub-having true) nil sub-having))
			(list (qpir-groupby new-keys sub-aggs final-having child-new) child-join))

		/* qpir-join inside the right subtree of a dep-join. This happens after
		   nested inner subqueries are lifted (recursive lift_dep_joins_pass) and
		   the bottom-up walker has converted the INNER dep-join to a regular
		   join BEFORE processing the OUTER dep-join. Walk into both children to
		   apply substitution; the join condition gets repr-substituted too.

		   We do NOT split the join's predicate against outer-aliases here — a
		   nested-correlation conjunct that referenced outer would have been
		   moved to the inner-join's predicate during the INNER dep-join's
		   processing. The OUTER walk just propagates substitution. */
		(quote qpir-join) (begin
			(define sub-pred (qpu-substitute-expr (qpir-join-predicate node) repr))
			(define left-result (qpu-unnest-right (qpir-join-left node)
				outer-aliases outer-ref-exprs repr))
			(define right-result (qpu-unnest-right (qpir-join-right node)
				outer-aliases outer-ref-exprs repr))
			(define left-new (nth left-result 0))
			(define right-new (nth right-result 0))
			(define combined-join (qpu-and-from-conjuncts
				(merge
					(qpu-and-conjuncts (nth left-result 1))
					(qpu-and-conjuncts (nth right-result 1)))))
			(list (qpir-join (qpir-join-type node) sub-pred left-new right-new
				(qpir-join-rhs-alias node)) combined-join))

		(error (concat "qpu-unnest-right: operator " (string (qpir-kind node))
			" not yet supported in right-side walker (phase 3)")))))

/* qpu-substitute-map-projections — apply repr substitution to every expression
in an assoc list of (name expr) projections. */
(define qpu-substitute-map-projections (lambda (projections repr)
	(map (coalesceNil projections (list)) (lambda (pair) (match pair
		'(name expr) (list name (qpu-substitute-expr expr repr))
		pair)))))

/* qpu-simplify-predicate — drop conjuncts that are trivially true, such as
(equal?? x x) after substitution. */
(define qpu-simplify-predicate (lambda (pred)
	(qpu-and-from-conjuncts
		(filter (qpu-and-conjuncts pred) (lambda (c)
			(not (qpu-is-tautology? c)))))))

/* qpu-col-ref-key — for tautology detection, reduce a (get_column tv ti col ci)
form to a canonical (tv col) key that ignores the ti/ci case-insensitivity
flags. Two refs that differ only in flags are SEMANTICALLY equal (same column)
but list-equal? would say no. Returns nil for non-get_column expressions. */
(define qpu-col-ref-key (lambda (expr) (match expr
	'((symbol get_column) tv ti col ci) (list tv col)
	'((quote get_column)  tv ti col ci) (list tv col)
	nil)))

(define qpu-is-tautology? (lambda (expr) (match expr
	(cons head args) (begin
		(define is-eq (match head
			(symbol equal??)   true
			(quote equal??)    true
			'(quote equal??)   true
			'equal??           true
			false))
		(if (and is-eq (equal? (count args) 2))
			(begin
				(define a (nth args 0))
				(define b (nth args 1))
				/* Direct equality */
				(if (equal? a b) true
					/* Or both are get_column refs to the SAME (tv col) ignoring
					   ti/ci flags (FAQ §22: column identity is by source, not by
					   the case-sensitivity of the access path). */
					(begin
						(define ka (qpu-col-ref-key a))
						(define kb (qpu-col-ref-key b))
						(and (not (nil? ka)) (not (nil? kb)) (equal? ka kb)))))
			false))
	false)))

/* qpu-build-join-condition-from-cclasses — for each outer-ref, generate an
IS-NOT-DISTINCT-FROM conjunct that links it to its inner equivalent (per repr).
Per FAQ §41 we always use IS NOT DISTINCT FROM (which is `equal??` here —
the NULL-safe equality) since we can't statically prove NOT-NULL on most
paths. Outer refs without a repr entry stay as raw equality predicates from
the original conjuncts (handled by combined-join). */
(define qpu-build-join-condition-from-cclasses (lambda (outer-refs repr)
	(qpu-and-from-conjuncts (reduce outer-refs (lambda (acc ref) (begin
		(define replacement (qpu-repr-lookup repr ref))
		(if (nil? replacement) acc
			(merge acc (list (list (quote equal??)
				(list (quote get_column) (nth ref 0) false (nth ref 1) false)
				(list (quote get_column) (nth replacement 0) false (nth replacement 1) false)))))))
		(list)))))

/* qpu-equate-conjuncts — for each (get_column tv col) in outer-ref-exprs,
produce a tautology-free predicate that asserts the outer ref equals its
"natural binding" in the unnested tree. With the simple algorithm we don't
substitute, so the join's predicate IS the outer-correlation predicate that
the right's select(s) carried — which is what qpu-unnest-right returns as
join-predicate. This helper exists so a future cclasses-aware variant can
build the IS-NOT-DISTINCT-FROM bindings directly. */
(define qpu-equate-conjuncts (lambda (outer-refs binding-map) (list)))

/* qpu-unnest-dep-join — the main per-dep-join transformer. Combines:
   - Right-side walker (qpu-unnest-right)
   - Conversion of qpir-dep-join → qpir-join preserving rhs-alias
   - Combination of dep-join's own predicate with the extracted join condition

Pre-condition: dj is a qpir-dep-join. */
(define qpu-unnest-dep-join (lambda (dj)
	(if (not (equal? (qpir-kind dj) (quote qpir-dep-join)))
		dj
		(begin
			(define left (qpir-dep-join-left dj))
			(define right (qpir-dep-join-right dj))
			/* Use bottom-left aliases: ignore aliases introduced by
			   sibling/previous dep-joins in the chain. */
			(define outer-aliases (qpu-bottom-left-aliases left))
			(define outer-ref-exprs (qpu-collect-outer-refs dj))
			/* Per FAQ §22 "per-key misses MUST survive": when unnesting a
			   dep-join, every outer row whose domain key is absent in the
			   helper must survive and get NULL-extended. That is LEFT JOIN
			   semantics, not INNER JOIN. The FAQ §11 EXISTS/IN rewrite
			   depends on this: COUNT(...) returning NULL for unmatched rows,
			   then COALESCE(NULL, 0) = 0, then the (> 0) check correctly
			   evaluates to false. With INNER JOIN the unmatched outer row
			   would silently vanish — wrong for NOT EXISTS, COALESCE-default
			   scalar subselects, and other LEFT-tolerant patterns.

			   Trivial case (no outer refs in right): INNER join is correct
			   here because an uncorrelated subquery always produces exactly
			   one row (COUNT returns 0, SUM returns NULL, etc. — never
			   empty). INNER ≡ LEFT for the single-row case, and INNER lowers
			   reliably through build_queryplan_inner without the
			   `LEFT JOIN ON TRUE` degeneracy.

			   EDGE CASES that DO produce 0 rows (HAVING-false in inner sub,
			   LIMIT 0, etc.) would lose the outer with INNER. Those are
			   tracked separately — needs LEFT-JOIN scalar contract that
			   doesn't trip the legacy LEFT-ON-TRUE issue. */
			(if (equal? (count outer-ref-exprs) 0)
				(qpir-join (quote inner) (qpir-dep-join-predicate dj) left right
					(qpir-dep-join-rhs-alias dj))
				(begin
					/* Pass 1: collect cclasses from select equalities in the right. */
					(define cc (qpu-make-cclasses))
					(qpu-collect-cclasses right cc)
					/* Build the repr substitution map: outer-ref → inner-equivalent. */
					(define repr (qpu-cc-build-repr cc outer-aliases))
					/* Pass 2: walk right with substitution applied. */
					(define outer-refs-as-pairs (map outer-ref-exprs (lambda (e) (match e
						'((symbol get_column) tv ti col ci) (list tv col)
						'((quote get_column)  tv ti col ci) (list tv col)
						nil))))
					(define right-result (qpu-unnest-right right outer-aliases
						outer-ref-exprs repr))
					(define new-right (nth right-result 0))
					(define extracted-join-pred (nth right-result 1))
					/* Join condition: combine the dep-join's own predicate, any residual
					   extracted outer-correlation (for non-equality correlations not in
					   cclasses), and IS-NOT-DISTINCT-FROM conjuncts for the cclass-bound
					   outer refs (FAQ §41 — NULL-safe equality). */
					(define cc-join-cond (qpu-build-join-condition-from-cclasses
						outer-refs-as-pairs repr))
					(define combined-pred (qpu-and-from-conjuncts
						(merge
							(merge (qpu-and-conjuncts (qpir-dep-join-predicate dj))
								(qpu-and-conjuncts extracted-join-pred))
							(qpu-and-conjuncts cc-join-cond))))
					/* LEFT JOIN preserves outer rows whose right side is empty (FAQ §22
					   "per-key misses"). For correlated scalars/EXISTS this is required. */
					(qpir-join (quote left) combined-pred left new-right
						(qpir-dep-join-rhs-alias dj))))))))

/* qpu-unnest-dep-join-with-aliases — variant of qpu-unnest-dep-join that
takes an explicit outer-aliases set (ancestor lefts ∪ this dep-join's own
left). Used by qpu-unnest-tree-topdown to honor FAQ §42's "inner dep-joins
must see the full ancestor chain" requirement. The logic mirrors
qpu-unnest-dep-join exactly — only the outer-aliases derivation differs. */
(define qpu-unnest-dep-join-with-aliases (lambda (dj outer-aliases)
	(if (not (equal? (qpir-kind dj) (quote qpir-dep-join)))
		dj
		(begin
			(define left (qpir-dep-join-left dj))
			(define right (qpir-dep-join-right dj))
			(define outer-ref-exprs
				(qpu-collect-outer-refs-with-aliases dj outer-aliases))
			(if (equal? (count outer-ref-exprs) 0)
				/* Trivial case: see qpu-unnest-dep-join comment for the
				   INNER-vs-LEFT trade-off documented in commit 5b6a9872f. */
				(qpir-join (quote inner) (qpir-dep-join-predicate dj) left right
					(qpir-dep-join-rhs-alias dj))
				(begin
					(define cc (qpu-make-cclasses))
					(qpu-collect-cclasses right cc)
					(define repr (qpu-cc-build-repr cc outer-aliases))
					(define outer-refs-as-pairs (map outer-ref-exprs (lambda (e) (match e
						'((symbol get_column) tv ti col ci) (list tv col)
						'((quote get_column)  tv ti col ci) (list tv col)
						nil))))
					(define right-result (qpu-unnest-right right outer-aliases
						outer-ref-exprs repr))
					(define new-right (nth right-result 0))
					(define extracted-join-pred (nth right-result 1))
					(define cc-join-cond (qpu-build-join-condition-from-cclasses
						outer-refs-as-pairs repr))
					(define combined-pred (qpu-and-from-conjuncts
						(merge
							(merge (qpu-and-conjuncts (qpir-dep-join-predicate dj))
								(qpu-and-conjuncts extracted-join-pred))
							(qpu-and-conjuncts cc-join-cond))))
					(qpir-join (quote left) combined-pred left new-right
						(qpir-dep-join-rhs-alias dj))))))))

/* qpu-unnest-tree-bottom-up — DEPRECATED in favor of qpu-unnest-tree-topdown
which handles the FAQ §42 doubly-nested case. Kept for backward compat with
existing qpipe-test assertions; the topdown driver is identical for single-
level / sibling-chained dep-joins (the common case). */
(define qpu-unnest-tree-bottom-up (lambda (tree)
	(qpu-map-tree tree qpu-unnest-dep-join)))

/* qpu-unnest-tree-topdown — walk the tree TOP-DOWN, threading the union of
ancestor-left aliases through the recursion. At each qpir-dep-join:
  - own-outer = qpu-bottom-left-aliases of its immediate left
  - extended = ancestor-aliases ∪ own-outer
  - Recurse into LEFT with ancestor-aliases (left doesn't see this dep-join's
    own outer in its right because left IS this dep-join's outer)
  - Recurse into RIGHT with extended (inner dep-joins see the full chain)
  - Convert THIS dep-join to qpir-join using extended as outer-aliases set

This fixes the leaked-free-var bug for doubly-nested correlated scalars
(FAQ §42): inner sq_N now correctly classifies outermost-table refs as
outer-correlations, extracts them to its join condition, and the cclasses
substitution makes the rewrite well-formed. */
(define qpu-unnest-tree-topdown-helper (lambda (node ancestor-aliases)
	(match (qpir-kind node)
		(quote qpir-dep-join) (begin
			(define left (qpir-dep-join-left node))
			(define right (qpir-dep-join-right node))
			(define own-outer (qpu-bottom-left-aliases left))
			(define extended (merge_unique (list ancestor-aliases own-outer)))
			(define left-rec (qpu-unnest-tree-topdown-helper left ancestor-aliases))
			(define right-rec (qpu-unnest-tree-topdown-helper right extended))
			(define new-dj (qpir-dep-join
				(qpir-dep-join-predicate node)
				left-rec right-rec
				(qpir-dep-join-accessing node)
				(qpir-dep-join-rhs-alias node)))
			(qpu-unnest-dep-join-with-aliases new-dj extended))
		(quote qpir-join) (qpir-join
			(qpir-join-type node)
			(qpir-join-predicate node)
			(qpu-unnest-tree-topdown-helper (qpir-join-left node) ancestor-aliases)
			(qpu-unnest-tree-topdown-helper (qpir-join-right node) ancestor-aliases)
			(qpir-join-rhs-alias node))
		(quote qpir-select) (qpir-select
			(qpir-select-predicate node)
			(qpu-unnest-tree-topdown-helper (qpir-select-child node) ancestor-aliases))
		(quote qpir-map) (qpir-map
			(qpir-map-projections node)
			(qpu-unnest-tree-topdown-helper (qpir-map-child node) ancestor-aliases))
		(quote qpir-groupby) (qpir-groupby
			(qpir-groupby-keys node)
			(qpir-groupby-aggs node)
			(qpir-groupby-having node)
			(qpu-unnest-tree-topdown-helper (qpir-groupby-child node) ancestor-aliases))
		(quote qpir-window) (qpir-window
			(qpir-window-partition node)
			(qpir-window-order node)
			(qpir-window-computations node)
			(qpu-unnest-tree-topdown-helper (qpir-window-child node) ancestor-aliases))
		(quote qpir-union) (qpir-union
			(qpir-union-order node)
			(qpir-union-limit node)
			(qpir-union-offset node)
			(map (qpir-union-branches node) (lambda (br)
				(qpu-unnest-tree-topdown-helper br ancestor-aliases))))
		node)))
(define qpu-unnest-tree-topdown (lambda (tree)
	(qpu-unnest-tree-topdown-helper tree '())))

/* ==================== Public driver (Phase 2: full algorithm for simple cases) ==================== */

/* unnest_pass — the L3 transformation. After this pass:
  - F(root) = ∅ (no unbound column references)
  - No qpir-dep-join remains anywhere in the tree

Phase 2 (this commit) handles BOTH trivial dep-joins AND non-trivial ones
whose right side is composed of qpir-select / qpir-map / qpir-groupby /
qpir-leaf (the canonical shape from lift_dep_joins_pass). The per-operator
§3.3 rules in qpu-unnest-right walk the right top-down extracting outer-
correlation predicates into a join condition and pushing outer-ref expressions
into qpir-groupby keys (FAQ §33 static-groupby rule).

Limitations (Phase 3+):
  - No cclasses/repr substitution yet — extracted predicates stay as
    `outer.x = inner.y` rather than being simplified to tautologies
  - No nested-dep-join parent-chained UnnestingInfo — independent dep-joins
    handled correctly via bottom-up walk; nested ones need top-down driver
  - qpir-join / qpir-union / qpir-iterate / qpir-window not supported as
    intermediate operators in the right-side walker yet */
(define unnest_pass (lambda (tree) (begin
	/* Switched to qpu-unnest-tree-topdown (was qpu-unnest-tree-bottom-up):
	   the topdown driver threads ancestor-aliases through the recursion so
	   inner dep-joins see the FULL outer scope (FAQ §42). For trees with
	   only single-level / sibling-chained dep-joins (the historical common
	   case) the two drivers produce identical results — the bottom-up one
	   is kept for compat with existing qpipe-test assertions but is no
	   longer the production path. */
	(define unnested (qpu-unnest-tree-topdown tree))
	(define residual-djs (qpu-count-dep-joins unnested))
	(if (> residual-djs 0)
		(error (concat "unnest_pass: " (string residual-djs) " dep-join(s) remain after unnest. "
			"qpu-unnest-dep-join failed to convert them — check shape against supported phase-2 patterns.")) nil)
	/* F(root) must be ∅ — every column ref bound by some provider. */
	(define fv (qpir-free-vars unnested))
	(if (> (count fv) 0)
		(error (concat "unnest_pass: " (string (count fv)) " free var(s) remain at root after unnest — "
			"output is not lowerable. First free ref: " (string (nth fv 0))))
		unnested))))

/* qpu-count-dep-joins — count remaining qpir-dep-join nodes in the tree. */
(define qpu-count-dep-joins (lambda (tree) (begin
	(define n (newsession))
	(n "c" 0)
	(qpu-walk-with-path tree (lambda (node path)
		(if (equal? (qpir-kind node) (quote qpir-dep-join))
			(n "c" (+ (n "c") 1)) nil)))
	(n "c"))))
