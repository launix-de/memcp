# AP3: Domain D And Correlated Subqueries

## Problem

Omnestum ACL checks and file/document helper queries are correlated. They must
not stay as recursive scalar or nested subselect evaluation. Their dependencies
must become explicit relational domain keys.

## Required Planner Behaviour

For each dependent helper:

- compute domain `D` as duplicate-free projection of all actually read outer
  references
- include session reads used as dependencies
- join helper results back by null-safe equality where needed
- preserve per-domain misses for LEFT / ANTI / SEMI semantics
- avoid duplicate domain bindings

Use `cclasses` / union-find for equality classes:

- collect equalities while walking operators
- substitute outer references with safe inner representatives when possible
- keep domain join or null-safe predicate when NULL semantics make substitution
  unsafe

## Omnestum Focus

ACL dependencies include shapes like:

- current user
- document ID
- mandate
- year
- location
- permission operation

For tests, user IDs may be hardcoded. Session variables and constants must still
be represented as dependencies, not read implicitly inside the helper.

## Tests

Add anonymized YAML tests for:

- correlated EXISTS with user constant
- correlated EXISTS with session variable
- duplicate outer rows do not duplicate helper domain
- nullable correlation key uses null-safe semantics
- nested correlated subquery does not create scalar fallback

## Done

- No correlated scalar subselect remains after `untangle_query`.
- `EXPLAIN IR` shows domain keys and helper joins.
- Omnestum ACL helpers become relational stages rather than recursive filters.
