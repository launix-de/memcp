# Topdown Cleanup: Omnestum Query Planner Work Packages

Goal: make the real Omnestum document dataview query fast on MemCP by moving
from legacy/materialized query shapes to relational, indexable, shard-parallel
plans.

The work is guided by `todos/faq-unnesting.md` and by the real Omnestum query
captured locally under `/tmp/omnestum-dv-*.sql`.

Non-negotiable rules:

- No scalar subselect fallback in `untangle_query`, `build_queryplan`, or
  expression compilation.
- Ordinary derived tables are flattened, not materialized.
- Materialization needs a semantic or physical reason: group cache, conflicting
  window order, shared DAG/CTE root, or explicit materialization semantics.
- Domain `D` is duplicate-free and contains exactly the outer/session
  dependencies read by a dependent helper.
- SQL-visible behaviour is tested with anonymized `tests/*.yaml` files.
- Real Omnestum data and real user data stay local and out of git.
- Every performance PR needs correctness tests, EXPLAIN/IR evidence, and a
  master-vs-branch runtime comparison on the relevant query shape.

Recommended order:

1. `00-measurement-baseline.md`
2. `01-repair-topdown-branch.md`
3. `02-derived-table-flattening.md`
4. `03-domain-and-correlated-subqueries.md`
5. `04-exists-in-semijoin.md`
6. `05-group-stages-and-count-subqueries.md`
7. `06-order-limit-acl-planning.md`
8. `07-fulltext-candidate-stage.md`
9. `08-auto-index-and-specialized-indexes.md`
10. `09-ir-contract.md`
11. `10-omnestum-ci-regressions.md`
12. `11-performance-gates.md`

Current strategic assessment:

- `master` is useful as stability baseline, but its planner is not enough for
  the real Omnestum document query.
- `topdown-cleanup` is the right architectural direction because the problem is
  not only a missing index. The query must expose derived-table wrappers,
  ACL-subqueries, text/file candidates, sort order, and pagination as relational
  planner state.
- The branch must be made conflict-free and then split into small PRs. Do not
  merge a large mixed planner branch.
