<!-- Copyright (C) 2026 Carl-Philip Hänsch -->

# T-SQL compatibility work plan

Review the language by statement family and expression contract. Application
examples set priorities; they do not establish complete compatibility. Each
implementation batch starts in a worktree, adds executable specifications, then
implements the forms and opens one thematic PR targeting master.

## Architecture

The frontend emits the common logical query representation and existing native
DDL/DML calls. The shared planner owns decorrelation, grouping, join ordering and
physical lowering. SQL expression compilation uses `(expr, type, collation)`;
only `expr` reaches execution. Type precedence comes from declarations and
syntax, independently of the internal value carrier.

Reuse native defaults, generated IDs, constraints, transactions and persistence.
Do not add dialect-specific copies. Physical index preferences are advisory;
retain actual uniqueness and reference constraints. Any required engine change
needs its own generic design and architecture review before implementation.

## Implementation batches

| Batch | Forms and semantic checks | Test inventory |
| --- | --- | --- |
| Projection and inline query sources | Assignment aliases, quoted aliases, explicit ALL, VALUES row constructors, column widths, duplicate names, declared type precedence, NULLs, ordering, TOP/OFFSET, joins and aggregates | `tests/integration/query-shapes/tsql-report-sources.yaml`; `tests/planner/set-operations/union-all.yaml` |
| Schema installation and upgrades | CREATE/ALTER TABLE, multi-column ADD with defaults, named keys, nullability/type changes, alias type declarations, views and dependency handling | `tests/sql/compatibility/tsql-native-frontend.yaml`; `tsql-parser.yaml`; `tsql-foreign-keys.yaml` |
| Batches and procedural control | GO, variables, IF/BEGIN/END, procedure declaration/execution, parameter contracts, OUTPUT/RETURN, loops, dynamic statements, error and transaction handling | Script/procedure specifications in `tests/sql/compatibility` |
| Writes and generated values | INSERT/UPDATE/DELETE forms, INSERT SELECT completion, joined writes, OUTPUT, MERGE, explicit identity writes, generated version values | `tests/sql/compatibility/tsql-parser.yaml`; `tsql-rowversion.yaml`; `tsql-rowversion-frontend.yaml` |
| Expressions and declared data contracts | Numeric/text/binary/date conversions, date styles/functions, implicit precedence and collations, aggregate result types | `tests/sql/compatibility/tsql-native-frontend.yaml`; `tsql-temporal.yaml`; `tsql-exact-numeric.yaml` |
| Driver and catalog integration | Session settings, declarations, prepare/execute, declared result metadata including empty results, database/schema/constraint inquiries | `tests/sql/compatibility/tsql-catalog.yaml`; `tsql-odbc.yaml` |

Several forms in each batch already have implementations. Check current master
and the specifications before adding code. Do not create one PR per keyword.
Exact extreme decimal precision remains separately deferred; its specifications
must not force a dialect-specific numeric system into storage or SCM.

## Current inline-source coverage and follow-ups

Inline VALUES syntax translates to ordinary query-block/UNION ALL terms. It
does not evaluate the SQL through another parser, store query results in a
frontend collection, or create frontend-managed temporary tables.

The new report-source suite covers simple projections, WHERE, ordering and
TOP/OFFSET; duplicate and NULL rows; numeric precedence; empty-result metadata;
alias escaping and scope; and existing native multi-column ALTER defaults.
Computed ORDER BY slots over derived UNIONs have shared-planner regressions.

Five positive specifications remain noncritical because their existing common
planner or completion paths are incomplete:

1. DISTINCT with TOP over an inline multi-row source.
2. An inline multi-row source joined to a native table.
3. GROUP BY/HAVING over an inline multi-row source.
4. The affected-row total of INSERT SELECT over a distributed inline source.
   The fixture also checks that all inserted values remain queryable.
5. Computed output ordering over a derived UNION DISTINCT. This shape still
   fails explicitly; sorting, deduplication and windows need one correct native
   contract before it becomes executable.

Single-row literal sources now share the ordinary logical derived-source
rewrite for INNER and LEFT JOIN. The match predicate, rather than any nullable
payload, determines NULL extension. Subsequent joins, WHERE, aggregates and
correlated subqueries consume the rewritten projection, including its declared
SQL type and invocation-bound positional parameters. No extra physical source,
frontend-managed row collection or storage change is needed.

This rewrite requires a literal/positional-parameter projection, no inner
filter, ordering, hidden expressions or logical stages, and no grouping, windows
or row-limit barrier. LEFT JOIN also requires a repeatable ON expression, proved
by the existing shared scalar purity check; clocks, random functions and mutable
session variables are rejected rather than independently deciding each output
column's NULL extension.
Executable projections and more general derived sources still need their own
shared relational proof; they must not be duplicated as scalar evaluations.

For these follow-ups, extend the common logical/lowering or native completion
contract. Preserve its domain, multiplicity, NULL-extension and cardinality
proofs. A different frontend execution path is not a fix. In particular, the
existing bounded parameter-row representation must prove filter, ordering,
grouping and correlated-domain correctness before exposing it more broadly.

Keep unsupported positive cases executable and noncritical. Remove the flag
after focused verification without weakening expected rows, values, errors or
time limits. Expected-error cases prove rejection only; they do not prove that
the corresponding successful statement is supported. Full regression, native
JIT and performance suites run in CI; local work uses focused reproductions.

## Procedural-control coverage and follow-ups

`tests/sql/compatibility/tsql-scripts.yaml` covers batch separators, lexical
variables, IF/BEGIN/END, persisted procedure definitions and parameter contracts,
WHILE/BREAK/CONTINUE, RETURN and captured procedure status, dynamic EXEC,
parameterized `sp_executesql`, TRY/CATCH, THROW and the current error context.
Successful parameterized execution returns status zero independently of native
DML row completion. System-qualified metadata procedures use the current database.
Control expressions compile through the ordinary expression triples; SQL leaves
use the shared planner and native statement/transaction lifecycle. Dynamic input
bindings keep their declared SQL contract and their own lexical variable scope.

The control follow-up also protects shared scalar aggregation across rollback:
multiple aggregates reduce into a state bounded by the SELECT descriptors,
using the existing native parallel scan. Transaction-local states must not fill
shared computed groups, and their local recipe names must match their readers.
The existing single-aggregate reduction remains unboxed.

Output parameter completion now covers stored procedures and `sp_executesql`:
`OUTPUT`/`OUT` copies a declared parameter back only when the call requests it,
using the callee's source contract and the caller's target contract. Input and
output values are isolated per lexical invocation; NULL, explicit RETURN,
DEFAULT arguments and source-only persistence retain those contracts. Shared
argument binding validates destinations before executing the procedure body.
No caller session or executable closure is serialized.

Catchable `RAISERROR` message text with severity 11 through 18 uses the current
TRY/CATCH request boundary. `ERROR_SEVERITY()` joins the existing error context;
parameterless THROW preserves it. Error-reporting procedures inherit their
caller's CATCH context, and nested handlers restore it locally. Simple `%s`, `%d`, `%i`, `%I64d` and `%%`
formats use declared expression contracts. Raising a frontend message after its
scalar-binding statement does not mark a successful native transaction statement
as failed. Numeric message catalogs, message options, exact long-message truncation and privileged/fatal
severities remain unsupported. Informational severity delivery and extended
format flags/width/precision have positive noncritical specifications. Direct
RPC output-value tokens and structured wire error details need separate connector
coverage; SQL batch OUTPUT bindings do not establish those wire contracts.

Remaining procedural work includes cursors, SELECT assignment from relational
queries, broader error metadata, informational messages and complete message
formatting, and connection/database context across nested invocations and batch
separators. The current number/state contract is supplied for explicit THROW,
RAISERROR and division by zero; other native failures retain their message
without an invented error code. Prepared and persisted plans must retain schema
guards across later DDL. These are compatibility gaps, not a reason to add
another execution engine.

## Installation readiness

Passing these language suites does not establish that a complete installer
runs. The final check needs real connection/bootstrap queries, original schema
and procedure definitions, migration batches, and initial application reads and
writes through a real driver. Add a small independent regression for each
observed missing form and place the fix in the owning layer.
