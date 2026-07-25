# AP1: Repair Topdown Cleanup Branch

## Problem

`topdown-cleanup` is architecturally relevant but currently not suitable as a
base for review or performance work if it has merge conflicts or mixed unrelated
changes.

## Scope

Allowed:

- `lib/queryplan*.scm`
- parser files only when necessary to keep existing SQL syntax tests working
- targeted YAML tests

Avoid:

- storage/compute engine changes
- test-runner behaviour changes
- compatibility fixes unrelated to top-down unnesting
- FOP/Omnestum-specific names in MemCP

## Required Work

- Finish the backmerge against current `origin/master`.
- Resolve conflicts in:
  - `lib/queryplan.scm`
  - `lib/sql-parser.scm`
  - `lib/sql.scm`
- Re-run and fix:
  - `tests/41_in_subquery.yaml`
  - `tests/70_union_all.yaml`
  - topdown-specific `tests/118_*`, `119_*`, `120_*` if present
- Ensure top-level `UNION` keeps `UNION DISTINCT` semantics.
- Ensure `UNION` inside `IN` / `EXISTS` can skip duplicate materialization when
  SQL semantics are duplicate-insensitive.

## FAQ Constraints

- No scalar fallback.
- If a shape is unsupported, raise explicit unsupported unnesting error rather
  than executing it recursively.
- Do not add new `legacy_materialized` paths.

## Done

- Branch builds.
- Targeted YAML suites pass.
- `EXPLAIN IR` for membership `UNION` has no unnecessary
  `union_distinct` materialization.
- The remaining diff is narrow enough to split into reviewable PRs.
