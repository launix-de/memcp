# AP5: Group Stages And COUNT Subqueries

## Problem

COUNT and aggregate subqueries must not run as per-row scalar work. The FAQ
defines groups as hard stage boundaries; the planner must compile them into
keytable-backed group stages.

## Required Planner Behaviour

- Cut group stages after `untangle_query`.
- Compile each group stage as its own queryplan universe.
- Emit inner group stages before outer stages.
- Substitute aggregate markers in the outer query with keytable reads.
- Apply HAVING after group collect, never before aggregate computation.
- Preserve one aggregate row per domain binding for correlated static groups,
  even when the inner input is empty.
- Compile `COUNT(DISTINCT expr)` as two group stages:
  - distinct-value grouping
  - count over distinct groups

## Omnestum Focus

Dataview helper counts and ACL count checks must not be evaluated for every
document row. Unused count projections from derived dataview wrappers must be
pruned before helper stages are built.

## Tests

YAML coverage:

- correlated COUNT over empty input returns 0
- correlated COUNT preserves outer rows
- HAVING is applied after grouping
- unused scalar aggregate projection inside derived table is dropped
- COUNT DISTINCT lowered via two-stage grouping

## Done

- Count helpers appear only when required by the outer projection/filter.
- Omnestum IR has explicit group stages instead of scalar COUNT recursion.
