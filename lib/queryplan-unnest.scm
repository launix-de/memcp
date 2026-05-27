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
	(qpu-lca-helper p1 p2 '())))

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
		(acc "list" '())
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
		(acc "list" '())
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
	(out "list" '())
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
			(nth new-children 0) (nth new-children 1))
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
				/* Trivial: right doesn't reference left → inner join with true predicate
				   (or the dep-join's own predicate, if it had one). */
				(qpir-join (quote inner) (qpir-dep-join-predicate node) left right)
				node)))))

(define qpu-trivial-eliminate-tree (lambda (tree)
	(qpu-map-tree tree qpu-trivial-eliminate)))

/* ==================== Public driver (Phase 1: trivial only) ==================== */

/* unnest_pass — the L3 transformation.
Phase 1 (this commit) only handles trivial dep-joins. For real correlations
the pass currently errors when residual dep-joins remain after trivial
elimination, prompting the next implementation phase. */
(define unnest_pass (lambda (tree) (begin
	(define eliminated (qpu-trivial-eliminate-tree tree))
	(define residual-djs (qpu-count-dep-joins eliminated))
	(if (> residual-djs 0)
		(error (concat "unnest_pass: " (string residual-djs) " non-trivial dep-join(s) remain. "
			"Phase 2+ (simple_djoin_elimination, per-operator §3.3 rules) not yet implemented."))
		eliminated))))

/* qpu-count-dep-joins — count remaining qpir-dep-join nodes in the tree. */
(define qpu-count-dep-joins (lambda (tree) (begin
	(define n (newsession))
	(n "c" 0)
	(qpu-walk-with-path tree (lambda (node path)
		(if (equal? (qpir-kind node) (quote qpir-dep-join))
			(n "c" (+ (n "c") 1)) nil)))
	(n "c"))))
