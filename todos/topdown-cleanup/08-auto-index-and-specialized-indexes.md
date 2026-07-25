# AP8: Auto Index And Specialized Indexes

## Problem

MemCP intentionally ignores explicit MySQL index creation in many compatibility
paths. For Omnestum to be faster than MySQL, the planner and auto-index system
must learn/use index shapes that match the query, not just table DDL.

## Required Index Rule

The useful index order is:

1. WHERE condition columns and expressions
2. sort columns
3. stable tie-breakers where needed

## Candidate Index Forms

- equality/range prefix index
- ordered suffix reuse
- expression index for normalized search keys
- text gram / fulltext candidate index
- composite candidate + order index
- ACL/domain membership index

## Lifecycle Requirements

Any index design must define:

- shard rebuild behaviour
- delta/delete handling
- schema change invalidation
- column drop invalidation
- restart/persistence behaviour
- memory budget and eviction rules
- optional prewarm policy

Read queries must not silently write durable optimizer state unless the design
defines lifecycle, invalidation, and resource limits.

## Omnestum Focus

Likely useful shapes:

- `(mandate, year, location, document_id)`
- `(acl_user_or_role, document_id)`
- `(text_token_or_gram, document_id)`
- `(filter_columns..., uploaded_at DESC, document_id)`

## Tests

YAML correctness coverage:

- LIKE `%foo%`
- `_` wildcard
- escapes
- NULL
- Unicode
- case/collation
- deletes/deltas/rebuild
- restart if persisted

Performance/EXPLAIN coverage:

- candidate index selected
- ordered index reused
- no wrong result under deletes/deltas

## Done

- Planner can choose specialized candidate/order indexes for Omnestum-like
  queries.
- Index state has clear invalidation and memory behaviour.
