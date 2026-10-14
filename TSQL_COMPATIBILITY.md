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
3. Outer-join NULL extension for a one-row inline source.
4. GROUP BY/HAVING over an inline multi-row source.
5. The affected-row total of INSERT SELECT over a distributed inline source.
   The fixture also checks that all inserted values remain queryable.

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

## Installation readiness

Passing these language suites does not establish that a complete installer
runs. The final check needs real connection/bootstrap queries, original schema
and procedure definitions, migration batches, and initial application reads and
writes through a real driver. Add a small independent regression for each
observed missing form and place the fix in the owning layer.
