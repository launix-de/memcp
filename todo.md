# Unnesting Arbitrary Queries

## Goal

Transform correlated scalar/IN/EXISTS subqueries into LEFT JOIN LIMIT 1
table entries inside `untangle_query`, so that the output is a flat table
list with predicates — no nested runtime code.

Based on Neumann/Kemper "Unnesting Arbitrary Queries" (BTW 2015).

## Pipeline

```
SQL → parse → untangle_query → join_reorder → build_queryplan
```

- `untangle_query`: Neumann unnesting. Correlated subqueries become
  LEFT JOIN LIMIT 1 table entries. Domain columns pushed through
  GROUP BY barriers. Output: flat table list + predicates + stages.
- `join_reorder`: Physical optimization. Reorder tables for optimal
  nested-loop execution (index matches, selectivity). Stub for now.
- `build_queryplan`: Physical plans. Takes the ordered table list and
  builds scan/scan_order/keytable/resultrow code.

## Current State

### Done
- 3-phase pipeline structure with contracts
- Case canonicalization at end of untangle_query (ti/ci → false/false)
- join_reorder stub
- Null-safe sql_mod_expr
- Various bugfixes (LEFT JOIN null guard, ci renamelist, outer_schemas threading)

### Not Done — the actual unnesting
- `build_scalar_subselect` still produces inline runtime code
  (!begin, scan, newsession) inside untangle_query
- Correlated subqueries are NOT yet converted to LEFT JOIN LIMIT 1
- The contract invariant (no physical ops in untangle output) is NOT enforced
- `join_reorder` is an empty stub

## What needs to happen

### Step 1: Correlated scalar → LEFT JOIN LIMIT 1

```sql
SELECT doc.id,
  (SELECT r.file FROM doc_revision r WHERE r.doc = doc.id
   ORDER BY r.created DESC LIMIT 1) AS file
FROM doc
```

Currently `build_scalar_subselect` compiles the scalar into inline
`!begin`/`scan`/`newsession` code that runs nested inside the outer scan.

Instead, `untangle_query` should produce:

```
tables: (
  (doc schema doc false nil)
  ($sq1 schema doc_revision true  ; LEFT JOIN
    (equal? (get_column $sq1 false doc false)
            (get_column doc false id false)))
)
fields: (
  "id"   (get_column doc false id false)
  "file" (get_column $sq1 false file false)
)
groups: (
  ; $sq1 gets ORDER BY created DESC LIMIT 1 as a stage
)
```

The scalar subquery becomes a regular table entry with:
- `isOuter = true` (LEFT JOIN semantics — NULL when no match)
- joinexpr = the correlation predicate
- An ORDER BY + LIMIT 1 stage attached to that table

### Step 2: Domain columns through GROUP BY

```sql
SELECT t1.id,
  (SELECT COUNT(*) FROM t2 WHERE t2.owner = t1.id) AS cnt
FROM t1 GROUP BY t1.id
```

When a correlated subquery crosses a GROUP BY barrier, the domain
columns (here: t1.id) must be added to the GROUP BY keys.
Neumann: D ⋈ Γ_A(T) == Γ_{A∪D}(D ⋈ T)

### Step 3: IN/EXISTS → SEMI/ANTI JOIN

```sql
SELECT * FROM t1 WHERE id IN (SELECT owner FROM t2)
```

becomes a SEMI JOIN table entry (or ANTI JOIN for NOT IN/NOT EXISTS).

### Step 4: join_reorder

Once tables are flat, implement actual reordering based on:
- Table sizes / scan estimates
- Available indexes
- Predicate selectivity
- JOIN fence constraints (LEFT/SEMI/ANTI: RHS must stay on RHS)
