# AP11: Performance Gates

## Problem

Green tests are not enough. The unnesting FAQ explicitly requires long-running
queries to become faster than master. Planner work must be gated by real query
runtime and plan quality.

## Required Gates

For every performance PR:

- run correctness YAMLs
- compare against `origin/master`
- measure relevant SQL query wall time
- inspect `EXPLAIN`
- inspect `EXPLAIN IR`
- track plan size
- track materialization/helper counts
- include CPU/wall ratio for shard-parallel queries where useful

## Omnestum Gates

Use real local Omnestum dump and query files:

- unfiltered document dataview
- fulltext-filtered dataview
- ACL-heavy dataview
- order/pagination-heavy dataview

## Target Milestones

Intermediate:

- real Omnestum document query below 5 seconds warm
- no ordinary derived-table materialization
- no recursive scalar subselect fallback

Final:

- fulltext-filtered document dataview below 1 second warm
- plan driven by candidate IDs / suitable auto-indexes
- stable under repeated PHP/UI interaction
- faster than MySQL for the target query class

## Done

- Performance results are reproducible locally.
- Each PR states whether it improves, preserves, or regresses the Omnestum
  performance gate.
- Regressions are either fixed before merge or explicitly scoped out with a
  follow-up blocker.
