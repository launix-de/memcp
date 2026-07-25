# AP9: IR Contract Between untangle_query And build_queryplan

## Problem

The current planner leaves too many SQL-like or legacy shapes for
`build_queryplan` to interpret later. That creates materialization fallbacks,
unclear helper stages, and hard-to-review generated Scheme.

## Required IR Fields

The IR exchanged after `untangle_query` must explicitly represent:

- relation source
- visible projections
- hidden domain columns
- dependency metadata
- canonical expression names
- join type: inner, left, semi, anti
- domain key
- group stage boundaries
- order requirements
- limit and offset
- candidate key
- materialization reason, if materialization is allowed

## Naming Rules

- Identifiers must be speaking.
- Identifiers must not start or end with underscore.
- Canonical helper names come from physical source columns and canonical
  expression trees, not visible aliases.

## Materialization Rules

Default should be equivalent to:

```scheme
(define materialize_reasons false)
```

This is an explanation control, not a license to keep legacy materialization.
The IR must not contain `legacy_materialized` for ordinary derived tables.

## Omnestum Focus

The document dataview query should show:

- flattened dataview wrapper
- ACL semijoin / anti / domain dependencies
- text/file candidate stages
- order and pagination requirements
- group stages only where actually required

## Tests

Use `EXPLAIN IR` shape tests plus result correctness:

- not contains `inner_select`
- not contains `legacy_materialized`
- not contains ordinary `.mat:*` for derived wrappers
- contains domain/semi/candidate/order markers where expected

## Done

- `build_queryplan` receives explicit relational planner state instead of
  guessing from nested SQL remnants.
- Omnestum IR is readable enough to explain why the chosen plan is fast.
