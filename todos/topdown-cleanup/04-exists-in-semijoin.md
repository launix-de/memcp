# AP4: EXISTS / IN / NOT IN As Semijoin Stages

## Problem

ACL checks and file/document candidate checks are logically membership tests.
Materializing counts or evaluating nested `IN` / `EXISTS` per row makes the
Omnestum document query scan too broadly.

## Required Planner Behaviour

- Lower `EXISTS` to presence / semijoin.
- Lower `NOT EXISTS` to anti-pass per domain key.
- Lower `IN` and `NOT IN` with correct SQL NULL semantics.
- Treat `UNION` inside `IN` / `EXISTS` as duplicate-insensitive only in that
  membership context.
- Keep top-level `UNION` as `UNION DISTINCT`.
- Use explicit miss/anti passes when join reordering would lose per-domain miss
  visibility.

## Omnestum Focus

Build visible candidate sets:

- documents allowed by ACL
- documents matching file/content conditions
- documents matching metadata conditions

Then intersect / join these candidate sets on document ID before expensive
projection and sorting.

## Tests

YAML coverage:

- `IN` with `UNION` without `ALL` in membership context
- `NOT IN` with NULL in probe and result
- `EXISTS` with per-key miss
- `NOT EXISTS` anti-pass with duplicate domain keys
- anonymized ACL-style semijoin chain

## Done

- Omnestum plan no longer scans broad document rows and then applies nested ACL
  membership late.
- `EXPLAIN IR` shows semijoin / anti / membership stages.
- No unnecessary `union_distinct` materialization appears in membership context.
