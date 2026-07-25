# AP0: Measurement Baseline

## Problem

Planner work is not useful unless it improves the real Omnestum document
dataview query. Synthetic tests are needed for CI, but the performance gate must
come from the real query shape and real data volume.

## Inputs

- Real query files: `/tmp/omnestum-dv-*.sql`
- Important variants:
  - unfiltered document list
  - fulltext / inventory number / city filter
  - no sorter
  - ID-only projection
  - ACL columns included / excluded
- Real imported Omnestum data stays local.

## Required Measurements

For each relevant query variant, capture:

- cold and warm wall time, at least five repetitions
- `EXPLAIN`
- `EXPLAIN IR`
- serialized plan size
- row counts for relevant tables
- materialization markers:
  - `.mat:*`
  - `legacy_materialized`
  - `inner_select`
  - union distinct helpers in membership context
- shard parallelism evidence for long-running scans:
  - CPU time divided by wall time
  - taskset one-CPU comparison where useful

## Omnestum Focus

Measure separately:

- base document dataview without text filter
- text-filtered document dataview
- file/content related filter
- ACL-heavy variant
- order/pagination variant

The unfiltered list matters: if it is already slow, the issue is ACL,
sort/pagination, and derived-table planning, not only fulltext.

## Deliverables

- A local measurement script or documented command sequence.
- A table comparing `origin/master`, `topdown-cleanup`, and any candidate PR.
- Saved EXPLAIN/IR outputs for before/after review.

## Done

- The team can reproduce the same baseline locally.
- The slowest operators and unwanted materializations are named.
- No real Omnestum data is committed.
