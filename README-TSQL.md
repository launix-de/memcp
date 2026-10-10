<!-- Copyright (C) 2026 Carl-Philip Hänsch; GPL-3.0-or-later -->

# Evaluating a desktop application with t-sql mode

Use the exact candidate commit and its accompanying `lib/` directory. The
binary and Scheme modules must come from the same revision. Keep the commit,
binary checksum, client version and connection settings with the test results.
This evaluation checks application compatibility; it does not certify a complete
application installation or authorize a production cutover.

## Start an isolated candidate

Use a fresh data directory or a disposable copy of the evaluation database.
Start from the candidate directory and capture standard output and errors:

```bash
./memcp --no-repl -data /path/to/evaluation-data \
  --api-port=4321 --disable-mysql \
  --tsql-port=1433 --tsql-database=evaluation \
  lib/main.scm > evaluation-server.log 2>&1
```

Configure the application for a direct TCP connection to the chosen host and
port, database `evaluation`, and a SQL username/password. The listener uses the
existing MemCP user catalog and database permissions. A fresh directory has
`root:admin`; create the evaluation user's permissions through the existing SQL
frontend. Integrated authentication and instance discovery are unsupported.

For an encrypted connection, add
`--tsql-tls-cert=cert.pem --tsql-tls-key=key.pem` and configure the client to trust
that certificate. This listener supports TLS 1.2. Without a certificate, choose
an unencrypted connection explicitly in the client.

Before starting installation, verify login and `SELECT 1 AS probe_value` through
the application's actual client stack. Then test Unicode, bound parameters,
table/column discovery and empty result metadata. Keep the server running through
installation so session settings and transaction state remain available.

## Import an existing evaluation database

Create a schema-and-data SQL script with the source's export tooling. The loader
accepts UTF-8, BOM-marked UTF-16 and gzip compression:

```scheme
(load_tsql "evaluation" "/path/to/export.sql" (sql_policy "root"))
```

Run this using the admin-only `/scm` endpoint or the Scheme console. `GO` is a
script separator, not a statement to send through the query endpoint. Database
headers select the source alias; supported references are confined to the chosen
destination. Native backup/package files and bulk-copy files have no reader.
An import error stops further statements; previously committed statements remain
applied. An unfinished transaction is rolled back. Use a fresh destination when
repeating a failed import.

Scalar and catalog predicates support `IF predicate statement` with an optional
`ELSE statement`, for example `IF OBJECT_ID(N'dbo.example',N'U') IS NOT NULL
DROP TABLE dbo.example`. Only the selected leaf statement is resolved and
executed, using the current session, permissions and transaction. Supported
leaves are SELECT, INSERT, UPDATE, DELETE, CREATE TABLE/TYPE/VIEW, ALTER TABLE,
DROP TABLE/TYPE/VIEW and SET. Blocks, loops, nested IF, relational predicates
such as IF EXISTS(SELECT ...), transaction control inside IF and conditional
result description are unsupported. Do not put a semicolon before ELSE.
For imports, a conditional guard and its leaf may span lines, including an
optional ELSE leaf. GO and semicolons separate completed statements; the
loader also recognizes consecutive line-start guards. Procedural blocks remain
outside this profile.

Create referenced parent tables before foreign-key declarations, and apply
supported foreign keys before loading child rows. ALTER cannot add a foreign key
to a populated or previously populated child table. Self/forward/cross-database
references and unchecked constraints are outside the current import profile.
Character foreign keys require byte-identical parent and child values.

## Exercise application workflows

Run installation/schema discovery first, then open an existing record, create a
record, update it, cancel an edit, and repeat with two connections editing the
same record. Check optimistic updates that bind the original eight-byte
ROWVERSION. Verify decimal/currency results against the source with the same
values and declared precision/scale. Commit and roll back explicit transactions,
then restart the server and confirm committed data and generated counters.
ROWVERSION activates only when declared during CREATE TABLE. Its declared binary
column uses persisted synchronous write hooks; ordinary tables have no such
hooks or extra column. Every matching update, including a no-op, changes the
token. Rollback restores the row while consuming the allocated token. Adding
ROWVERSION to an existing table remains unsupported because existing rows and
the new hooks must become visible atomically.
For generated identifiers, check both `@@IDENTITY` across requests and
`SCOPE_IDENTITY()` within the application's actual batch or prepared RPC.
Exercise foreign-key failures and permitted cascades, including nullable keys.

Use a fresh evaluation database when testing this architecture revision. Historical
coefficient and clock payloads remain readable, but tables with those older
declared carriers require an atomic data and index migration before accepting
writes or new constraints. Such a migration is not implemented yet.

Unsupported SQL or protocol shapes must produce an explicit error. Record the
first failure before continuing: later failures may merely reflect an incomplete
installation. Use the README's current compatibility limits to distinguish a
known missing feature from a regression.

## Capture a useful reproduction

Keep SQL text, parameter names, values and declared types, database name,
connection identifier, statement order, session settings, transaction boundaries,
result metadata, row counts and the exact error/SQLSTATE. A query list without
bindings and session order is not a complete replay. Include the client-side
trace and `evaluation-server.log` around the first failing statement.

To inspect engine execution during a short reproduction, an administrator can
set `(settings "TracePrint" true)` through `/scm`; disable it after capturing the
failure. Traces may contain query values, so keep originals with the private
evaluation data and anonymize a minimal reproduction before publishing it.

The server's `/tsql/evaluation` endpoint accepts one t-sql statement per request;
its HTTP response is newline-delimited JSON. Keep protocol-dependent tests on
the actual TDS connection rather than replacing them with HTTP requests.
