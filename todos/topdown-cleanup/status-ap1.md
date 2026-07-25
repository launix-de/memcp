# AP1 Status - topdown-cleanup branch repair

Date: 2026-07-25

Worktree:

`/home/carli/projekte/memcp/.claude/worktrees/memcp-topdown-cleanup`

Branch:

`codex/topdown-cleanup`

Confirmed cleanups:

- Removed the rejected unstaged GLS/session workaround from the worktree (`scm/sync.go`, `storage/compute.go`, `storage/partition.go` restored).
- Removed transient `.perf_baseline.json.lock`.
- Did not delete the untracked data backup directories.

Verification:

- `make`: passed.
- `python3 run_sql_tests.py tests/70_union_all.yaml`: 34/34 passed.
- `python3 run_sql_tests.py tests/41_in_subquery.yaml`: 27/27 passed. The previously reported UNION-in-membership IR regressions are already fixed in this worktree.
- `python3 run_sql_tests.py tests/118_nested_correlated_scalar_perf.yaml`: 4/4 passed.
- `python3 run_sql_tests.py tests/119_topdown_cleanup_regressions.yaml`: 6/9 passed.
- `python3 run_sql_tests.py tests/120_erpl_dashboard_hidden_regression.yaml`: 3/4 passed; dashboard full scalar shape took 1640 ms with a 1000 ms hard limit.

Open AP1 regressions:

- `scalar EXISTS result must be false, not NULL, when the nested EXISTS has no match`
- `skip-level correlated IN inside scalar chain must bind the outermost row`
- `skip-level NOT IN inside depth-4 scalar chain must compile under the default client timeout`
- `full dashboard scalar shape compiles under the strict query budget`

IR findings:

- The failing `EXISTS` case still lowers through a legacy `.mat:*` derived helper.
- The inner presence helper materializes only matching rows (`ref_id=1,2`) and loses the full outer domain row for `a.id=3`; the final LEFT miss therefore exposes `NULL` instead of SQL `FALSE`.
- The failing `IN` case exposes a join predicate containing `(get_column "d" false "ref_e" false)` in the outer plan even though alias `d` only exists inside the helper. The join key should be a projected/helper-domain column, not an inner alias leak.
- These are planner/lowering bugs in the active legacy materialization path, not data or parser problems.

Next AP1 step:

Patch the active legacy-derived/helper materialization path so presence predicates keep the required domain defaults and skip-level correlated keys are projected through helper boundaries. Do this against the existing anonymized `tests/119_topdown_cleanup_regressions.yaml` cases; do not add real Omnestum SQL to git.
