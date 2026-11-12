<!-- Copyright (C) 2026 Carl-Philip Hänsch -->

# Reviewed valid-SQL implementation gaps

An expected SQL error means the statement is invalid or must be denied. It must
not stand in for a missing implementation of valid SQL. The cases below instead
assert their complete projected result with `rows` and `data`, and remain
`noncritical: true` while implementation is missing. Once the result is supported
and proved, remove `noncritical`; keep the positive oracle and historical name.
This change includes no engine, frontend or runner implementation.

The protected [migration manifest](../tools/sql_implementation_gap_migrations.json)
authorizes only these exact cases, SQL, dialects, previous errors, new result
oracles and complete case/fixture/execution prefixes (including session and
authentication-related fields). Hashes include typed JSON values, so
Boolean `true` cannot substitute for integer `1`. The manifest does not infer SQL
semantics or projection coverage: each listed oracle is reviewed against all
selected columns and fixture values. A record is not a license to remove other
assertions or reclassify another error.

This draft contains 26 derived statement migrations and one exact transport-suite
split. The wider scan found 85 explicitly
labelled candidates among 1409 top-level error assertions. That wording is only
an inventory lead: unknown catalogs, invalid arithmetic/type combinations,
malformed syntax and safety/admission failures remain negative. Binary encoding
and stateful DDL candidates need separate semantic and fixture review before any
entry can be added; they are not authorized by this list. No text-based automatic
conversion of all `unsupported` or `error: true` assertions exists.

Only deterministic presentation ordering or an explicit result alias is added
where needed. The original expressions, joins, groups, predicates, limits and
result multiplicities are retained. The quoted-identifier and WHILE controls
retain their original statement and add an observable result/restore. The nested
JOIN keeps the original twelve-row bag, now ordered only by its visible `id`.

| Fixture | Historical case identity | Expected result |
| --- | --- | --- |
| integration/query-shapes/tsql-document-workflows.yaml | A right nested outer join boundary is rejected until represented logically | 12 rows; Complete original join bag: six10, two11, and12/14/15/16; ORDER only makes visible id deterministic. |
| integration/query-shapes/tsql-document-workflows.yaml | TOP rejects a count subquery until relational bound lowering exists | 10 rows; MAX(line id)=1600 exceeds all ten fixture documents; tenant2 preserves duplicate id10. |
| planner/aggregates/composed-scalar-aggregate-shapes.yaml | conditional_sum_presence_offset_remains_explicitly_unsupported | 5 rows; Ungrouped aggregate returns one row; OFFSET1 removes it; scalar NULL coalesces toTRUE for every parent. |
| planner/set-operations/union-all.yaml | Derived UNION DISTINCT rejects unsupported computed output order | 2 rows; DISTINCT removes duplicate2 before projection n+1 and final order. |
| planner/subqueries/in-subquery.yaml | Grouped NOT IN remains explicitly unsupported | 3 rows; Grouped RHS is {1,2}; fixture source IDs are1..5. |
| planner/subqueries/in-subquery.yaml | Projected grouped IN remains explicitly unsupported | 5 rows; Grouped RHS is {1,2}, without NULL; all five source IDs checked. |
| sql/compatibility/psql-parser.yaml | Ordered aggregate window fails until default frames are supported | 4 rows; Fixture values10,30,50,20; unique id default frames yield running sums10,40,90,110. |
| sql/compatibility/psql-parser.yaml | Unsupported LEAD default fails explicitly | 4 rows; Two-row lead yields50,20,-1,-1; complete sorted multiset retained. |
| sql/compatibility/psql-parser.yaml | Unsupported window frame fails explicitly | 4 rows; Fixture values10,30,50,20; two-row windows10,40,80,70, sorted by visible result. |
| sql/compatibility/tsql-apply.yaml | Partitioned TOP in APPLY is explicitly unsupported | 5 rows; Lookup keys7,7,NULL,9,12; earliest matching child IDs10,10,NULL,12,NULL. |
| sql/compatibility/tsql-identifiers.yaml | Four part column references fail instead of selecting another table | 3 rows; Valid T-SQL behavior; complete projected values are specified. Historical negative identity retained. |
| sql/compatibility/tsql-identifiers.yaml | Three part column references fail instead of silently truncating the source | 3 rows; Valid T-SQL behavior; complete projected values are specified. Historical negative identity retained. |
| sql/compatibility/tsql-native-frontend.yaml | Unsupported double quoted string mode is rejected | 1 rows; Valid T-SQL behavior; complete projected values are specified. Historical negative identity retained. |
| sql/compatibility/tsql-parser.yaml | CTE rejects recursion explicitly until supported | 3 rows; Anchor1 and recursive n+1 while n<3 produce exactly1,2,3. |
| sql/compatibility/tsql-parser.yaml | IF rejects an internal semicolon before ELSE | 1 rows; Valid T-SQL behavior; complete projected values are specified. Historical negative identity retained. |
| sql/compatibility/tsql-parser.yaml | IF rejects relational EXISTS until logical support exists | 1 rows; Valid T-SQL behavior; complete projected values are specified. Historical negative identity retained. |
| sql/compatibility/tsql-parser.yaml | IF rejects scalar subqueries | 1 rows; Valid T-SQL behavior; complete projected values are specified. Historical negative identity retained. |
| sql/compatibility/tsql-parser.yaml | Ordered aggregate window is explicitly unsupported | 4 rows; Fixture values10,30,50,20; no earlier mutation of source rows. |
| sql/compatibility/tsql-scripts.yaml | REST rejects a relational scalar fallback in IF | 1 rows; Valid T-SQL behavior; complete projected values are specified. Historical negative identity retained. |
| sql/compatibility/tsql-scripts.yaml | WHILE rejects relational scalar fallback | 1 rows; Valid T-SQL behavior; complete projected values are specified. Historical negative identity retained. |
| sql/compatibility/tsql-temporal.yaml | Fractional temporal conversions fail rather than silently lose precision | 1 rows; Valid T-SQL behavior; complete projected values are specified. Historical negative identity retained. |
| sql/compatibility/tsql-temporal.yaml | Fractional time conversions fail rather than silently lose precision | 1 rows; Valid T-SQL behavior; complete projected values are specified. Historical negative identity retained. |
| sql/compatibility/tsql-temporal.yaml | TRY clock-only TIME refuses unsupported fractional carriers | 1 rows; Valid T-SQL behavior; complete projected values are specified. Historical negative identity retained. |
| sql/compatibility/tsql-temporal.yaml | TRY clock-only datetime refuses unsupported fractional carriers | 1 rows; Valid T-SQL behavior; complete projected values are specified. Historical negative identity retained. |
| sql/compatibility/tsql-temporal.yaml | TRY_CAST refuses unsupported fractional temporal carriers | 1 rows; Valid T-SQL behavior; complete projected values are specified. Historical negative identity retained. |
| sql/compatibility/tsql-temporal.yaml | TRY_CONVERT refuses unsupported fractional temporal carriers | 1 rows; Valid T-SQL behavior; complete projected values are specified. Historical negative identity retained. |
| sql/compatibility/tsql-scalar-character-loops.yaml | Unsupported loop control and relational bodies fail before publication | Exact five-way split: RETURN, finite CONTINUE and finite relational WHILE become independent positive noncritical cases; BREAK and an independent publication check remain critical. Existing invalid declarations, division errors and detailed BREAK result checks stay unchanged. |

## Exact scalar-loop transport migration

The [review patch](implementation-gap-migrations/tsql-scalar-character-loops.patch)
targets a suite introduced in the feature branch, absent from this infrastructure
branch's base. It is intentionally a review artifact rather than an active copy
of tests whose scalar-loop implementation is absent here. Apply this exact patch
only after the separately approved infrastructure update reaches the trusted base.

The frozen source is feature commit `6f8c15794`: its 13 cases become 17 cases.
Every existing case except the one historical mixed sequence remains byte-for-byte
equivalent after YAML parsing. In particular all four proved BREAK/RETURN cases
retain their critical status and complete result expectations.

The mixed historical TDS case reused `invalid_loop` after each assumed failure.
Once BREAK succeeds, later declarations collide with that published function.
Independent function names and explicit cleanup remove that accidental coupling.
The historical case identity remains on the RETURN specification. The two existing
positive BREAK cases and both early-RETURN cases remain critical with their complete
result oracles intact. The migrated historical RETURN specification can initially
remain noncritical; proving that exact new specification permits promotion to critical.
Duplicate variable declarations and division by zero remain expected failures.

CONTINUE's old unconditional loop could not terminate with a positive counter.
Its positive specification decrements the counter before CONTINUE and observes
which subsequent assignments are skipped. The old relational WHILE referenced a
missing table and had no terminating row predicate. Its positive specification
uses a declared two-row fixture and `id=@count`, so each updated binding changes
the next condition. Calls cover active, absent, zero and NULL inputs. No missing
catalog or invalid-SQL rule becomes an implementation-gap exception.

This one transport migration is authorized by hashes of the **complete** old and
new suite and historical case, using typed JSON. That includes all statements,
child expectations, metadata, fixtures, session/authentication options and critical
sibling tests. There is no generic transport error-to-success exception. Changing
any element requires another exact administrative review. The added Python guard
cases cover altered child SQL/oracles, Boolean-versus-integer substitutions,
fixtures, sessions, authentication, removed children and weakened critical siblings.

## Administrative policy-update procedure

1. Prepare a separate infrastructure PR from current `master`, containing the
   exact inventory, positive oracles, policy data/code, focused Python guard tests
   and this documentation. Do not mix engine fixes into it.
2. Review every SQL and result tuple, regular-error exclusion, hash/context bound
   and protected-file diff. Run `python3 tools/test_sql_time_limit_guard.py` and
   the candidate checker without changing budgets or CI execution.
3. Obtain explicit maintainer authorization for the concrete PR and its exact
   commit before any administrative merge. Permission to prepare/review this PR
   is not permission to merge or bypass protection.
4. The current trusted-base workflow intentionally rejects changes to its own
   protected policy, including this PR. There is no self-approval label, environment
   variable or checker option. Only an authorized repository administrator may
   apply the reviewed policy update through the repository's controlled
   administrative merge procedure, if its branch rules permit. If no such
   procedure is available, stop and ask the maintainer; do not rewrite the workflow
   or disguise the protected change to make the feature PR green.
5. After that independently authorized update reaches the trusted base, rebase
   feature PRs and rerun the unchanged guards. They use the base checker and base
   manifest, never candidate policy code. Further manifest changes require the
   same separate review; implementing a listed case only requires promotion back
   to critical.

Time/plan/compile budgets, CI participation, disabled-case protection,
`not_contains` and existing positive assertions remain protected. Unlisted
error-to-success transitions and critical-to-noncritical changes are rejected.
No administrative action has been performed as part of preparing this proposal.
