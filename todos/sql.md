# SQL Open Items (2026-02-21)

Some test cases are `noncritical: true`. Once implemented, remove the flag.

## High Priority

### Group-stage corner cases (derived tables / nested ORDER+LIMIT)
- **Status:** still failing in `tests/52_group_stage_corners.yaml` (currently noncritical).
- **Observed errors:** `Unknown function: nil`, incorrect row pruning for nested `ORDER BY ... LIMIT` in derived tables.
- **Needed:** stable staging/materialization for derived subqueries before outer ordering/limiting.

### Aggregation on joined tables (prejoins)
- **Status:** not implemented.
- **Observed error:** `Grouping and aggregates on joined tables is not implemented yet (prejoins)`.
- **Impact:** `COUNT(DISTINCT ...)` and grouped aggregates over joins in `tests/52_group_stage_corners.yaml`.

### UNION ALL with set-level ORDER/LIMIT/OFFSET
- **Status:** UNION ALL baseline is implemented, global ordering/limiting remains intentionally blocked.
- **Concept doc:** `todos/union-sort.md`
- **Tests:** noncritical in `tests/70_union_all.yaml`.

## Medium Priority

### Multi-table UPDATE semantics
- **Status:** parser/codegen now rejects multi-table targets with explicit error.
- **Current behavior:** `multi-table UPDATE is not implemented yet`.
- **Tests:** `tests/07_error_cases.yaml` (noncritical).
- **Next step:** implement proper multi-target update planning/execution (or keep explicit reject if out of scope).

### Remaining ERPL/dbcheck dialect gaps
- **Candidates:** `REPLACE INTO`, `SHOW TABLE STATUS ... LIKE ...` (see `todos/erpl.md`).
- **Status:** still tracked for ERPL parity.

## Already Covered

- LONGTEXT/LONGBLOB/DATETIME aliases
- RAND()/RANDOM()
- ENGINE=MEMORY semantics test coverage
- Prefix index length acceptance
- Session variables (`SET @x`, `SELECT @x`)
