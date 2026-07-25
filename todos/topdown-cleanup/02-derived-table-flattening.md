# AP2: Derived Table Flattening

## Problem

The Omnestum dataview query wraps the useful work in derived tables like
`FROM (SELECT ...) t`. Treating this as a physical materialization hides
filters, sort requirements, ACL predicates, and projection pruning from the
planner.

## Required Planner Behaviour

Ordinary derived tables must be inlined early in `untangle_query`.

Flattening must provide:

- projection renaming map
- condition merge after renaming
- outer WHERE pushdown where semantically valid
- visible projection columns
- hidden domain columns
- unused projection pruning
- order/limit metadata preserved as planner properties

Materialize only when required by:

- group cache
- conflicting window order
- shared CTE / DAG root
- explicit materialization semantics

## Omnestum Focus

The dataview columns such as document ID, year, location, mandate, visibility,
and file metadata must map back to source columns or canonical expressions.

Unused columns in the document list must not force evaluation of:

- scalar subselects
- count helpers
- file metadata helpers
- ACL projections that are not requested

## Tests

Add or extend anonymized YAML tests, for example:

- derived projection rename
- outer WHERE pushed through derived alias
- unused projection containing scalar subselect is not evaluated
- LEFT JOIN inside derived table stays correct
- ORDER BY outside derived table remains correct

Use `EXPLAIN IR` negative checks:

- no `legacy_materialized`
- no `.mat:*` for ordinary derived wrappers

## Done

- The Omnestum dataview wrapper is no longer materialized.
- The real query's IR shows a flattened projection/filter/order chain.
- Correctness YAML tests cover the flattening rules.
