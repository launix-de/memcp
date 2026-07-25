# AP10: Omnestum Query Shape In CI

## Problem

The real Omnestum production query and data must not be committed, but the
planner must be protected against regressions in the same structural query
shape.

## Required Work

Extract anonymized minimal SQL suites from the real query:

- same nesting shape
- same ACL pattern
- same derived dataview wrapper
- same text/file candidate pattern
- same order/limit/pagination pattern
- small synthetic data only

No real names, user IDs, customer data, document titles, file names, or table
names from Omnestum.

## Suggested Suites

- `tests/66_derived_dataview_flatten.yaml`
- `tests/98_semijoin_acl_candidates.yaml`
- extend `tests/91_fulltext_like_index.yaml` or add a focused candidate-stage
  suite
- extend `tests/14_order_limit.yaml` and `tests/56_multishard.yaml` for ordered
  braking and multishard ACL

## Test Requirements

Every planner optimization PR needs:

- correctness test
- negative/edge case
- EXPLAIN or EXPLAIN IR shape test
- no Go-only test when behaviour is SQL-visible

## Done

- CI contains the Omnestum query structure without Omnestum data.
- Future planner changes cannot reintroduce derived materialization, scalar
  fallback, or late ACL filtering for this shape unnoticed.
