# PR Readiness: Topdown Cleanup

Branch: `codex/topdown-cleanup`

## Improved Query Classes

- Correlated scalar subqueries are lifted into a relational IR before lowering,
  including nested scalar chains where an inner marker references an outermost
  row.
- Correlated `EXISTS`, `IN`, and `NOT IN` membership predicates are exposed to
  the planner instead of staying as opaque `inner_select_*` expressions in the
  important nested cases covered by `tests/119_topdown_cleanup_regressions.yaml`.
- `UNION` inside membership predicates is duplicate-insensitive and can skip
  `union_distinct` materialization, while top-level `UNION` still keeps
  distinct semantics.
- Ordinary derived wrappers can be flattened when they only rename/project
  columns or chain filters/order stages that remain valid after flattening.
- Scalar subqueries with `LIMIT 1` keep their per-domain limit/cardinality
  contract when nested scalar markers are present.

## Removed Or Avoided Fallbacks

- Grouped planning for simple `GROUP BY` stays in `qpir-groupby`; regression
  gates now reject `legacy_materialized` and `.mat:` helpers for the covered
  group-cache cases.
- Nested correlated `IN` and `NOT IN` regression tests now reject leaked
  `inner_select_in` in `EXPLAIN IR`.
- Derived ACL-like wrapper tests reject broad `materialized-subquery-source` and
  `.mat:` fallback shapes for the covered flattened reference patterns.

## Still Legacy Or Compatibility Paths

- Physical materialization still exists for hard boundaries such as grouped
  subplans that must be lowered as grouped stages, window/order-limit
  boundaries that cannot be merged safely, and explicit compatibility paths.
- Large real-world dataview queries can still produce many helper/materialized
  stages; the Omnestum document query is not yet below the target runtime.
- The Scheme formatter/linter still reports depth warnings in several planner
  files; these should be resolved separately from semantic test hardening so
  the mechanical diff remains reviewable.
- Generated documentation deletion is unrelated to the topdown optimizer work
  and should either be removed from this PR or handled as a separate master-based
  cleanup PR.

## Verification

- `python3 run_sql_tests.py tests/119_topdown_cleanup_regressions.yaml --jobs 1 --log-times --fail-fast`
- `python3 run_sql_tests.py tests/14_order_limit.yaml tests/71_repartition_concurrent.yaml tests/83_group_cache_invalidation.yaml --jobs 1 --log-times --fail-fast`
