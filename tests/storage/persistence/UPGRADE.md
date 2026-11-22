<!-- Copyright (C) 2026 Carl-Philip Hänsch -->
# Upgrade compatibility contract

## Candidate row-initialized column appends

The proposed `catalog-v21/` domain carries ordinary column appends whose
historical values were realized by the original transaction's native row WAL.
The recipe retains inert future default data and optional exclusive property
claims. It never retains or re-executes the initializer callback. Versions
1–20 reject the new fields, including explicit null spellings; their existing
constant initialization and declaration domains remain unchanged.

One existing native statement undo boundary spans the complete ADD batch.
The final private declaration follows completion of the existing parallel scan
workers. Rollback and savepoints restore the same transaction's row and catalog
suffix. Unaccepted wider WAL rows are handled by the existing hidden-row replay
and original transaction decision, without requiring publication of their
unfinished column declaration.

`column-row-initialization.yaml` supplies positive original-transaction,
exact-value, generator-failure, highwater, SAFE/LOGGED crash and restart controls.
Unordered property-removal composition remains a positive noncritical feature
case; it is rejected before effects in both orders. This proposal requires
independent source review and runtime qualification before publication.

The current source composition retains physical WAL pins and predecessor
retirement dependencies from the native transaction owner. Initializer row
replacements copy existing identity values explicitly and do not reserve new
identities; only later ordinary INSERT preparation advances the allocator.
The focused controls include unchanged identity observations, restart and a
pending original row carried through two rebuild generations before ADD.
These controls remain unexecuted until matching-native qualification.

The SQL slice binds closed UUID generators in SCM; statement-clock and
identity generators are separate declaration contracts. Actual-Tx nested
mutation/declaration calls retain ordinary exclusion. Arbitrary administrative
SCM initializers which deliberately invoke nil-Tx temporary-column DDL are
not qualified: that existing path can wait on the retained DDL lease and has
no original-call parentage. This proposal adds no ambient ownership inference.

The Upgrade Compatibility workflow proves that the candidate can open data
written by every supported predecessor. It always tests the pull request's
base commit (the previous `master`) and every Git tag matching
`upgrade-release-*`. A manual run may add temporary predecessor refs; these do
not replace the mandatory revisions.

## Pinning a release

Pin a released data format with an immutable annotated tag on the release
commit:

```sh
git tag -a upgrade-release-0.2 RELEASE_COMMIT -m "Upgrade fixture for 0.2"
git push origin upgrade-release-0.2
```

Every matching tag is discovered from Git and becomes a separate CI matrix
job. Do not move or reuse one of these tags: it is a permanent promise that
current MemCP can open data written by that revision. `workflow_dispatch`'s
`base_refs` input accepts a JSON list such as `["topic/old-format"]` for
additional one-off checks.

The `upgrade-release-before-lz4-dictionaries-2026-10-14` pin identifies
`44d6af4d331fda2a9901375c0895a54f1a41260e`, immediately before the LZ4 dictionary
persistence change (#1107) and the subsequent APPLY planner change (#1110).
It preserves a predecessor from before both changes in the mandatory matrix.

The `upgrade-release-before-group-source-recipes-2026-09-29` pin identifies
`a9cc5da0f69b566d59db757461f90ff8ee6ee467`, before the point/range group-cache
carrier identity changes (#1010). Its binary warms and persists the older
helper structures before the candidate opens the same data directory. The test
harness must not remove or rewrite those structures to prepare the upgrade;
the engine owns compatibility with historical optimization state.

The `upgrade-release-before-concat-buffers-2026-11-14` pin identifies
`bdaa313a6`, before version 6 replaces runtime LZ4 dictionaries with immutable
concat pools. `up_concat_text` exercises repeated SQL-shaped fragments,
unique parameters, Unicode, empty values and NULL through legacy loading,
candidate writes and restarts.

## What each matrix job proves

The predecessor binary creates the complete fixture and supplies two oracles:

- an exact snapshot of every public table's stable schema metadata and every
  stored value;
- results from a public SQL workload, including repeated grouping, aggregate
  ordering, expression group keys, correlated equality/range lookups and
  windows with opposing order directions, and native calendar values with
  constant/varying whole seconds, NULLs, duplicate grouping, comparisons,
  indexed gathers and ordered pages.

The same SQL workload runs on the predecessor, after a predecessor restart,
after opening the data with the candidate, after cold and warm source-table
updates, and after a candidate restart. Each execution must return the declared
result and the candidate's unchanged executions must equal the predecessor's
saved query oracle. The updates are restored before the full data snapshot is
compared.

Calendar snapshots use public `DATE_FORMAT` expressions for DATE, DATETIME and
TIMESTAMP rather than a predecessor's integer or text wire carrier. The same
canonical expressions run on both writers and readers. These fixtures contain
whole seconds understood by every pin; they do not assert fractional precision
that an old writer could never produce.

Queries which can create adaptive caches run repeatedly. This is deliberate:
the workload must cross normal admission thresholds through public SQL rather
than constructing, naming or asserting an internal helper. Tests must not
inspect hidden schemas, column names, cache families or `EXPLAIN` operator
names. Coverage comes from realistic query breadth and lifecycle depth; an
implementation remains free to migrate, discard, rename or stop producing a
cache as long as public results remain correct.

The candidate additionally executes equivalent derived-table alias variants
against those same query results, including repeated aliases across grouping
barriers and grouped window totals. These extra checks run before candidate
DML, after cold/warm mutations, and after restart. A predecessor may itself
contain the alias-shadowing regression; its unchanged, distinctly aliased SQL
still supplies the oracle. The original old/new query comparisons remain
mandatory alongside the candidate's equivalent variants.

Engine format versions, cache names and cache layouts are a black box to this
test. It never edits the implementation to synthesize a layout change or checks
for a particular version string. The candidate first executes the public queries
on the predecessor's untouched persisted data, before any candidate DML. A
separate cold process then tests mutations before the first query.

A crash, query error or result mismatch fails the upgrade. Successful migration
or reconstruction is judged by the same public results and complete public-data
snapshot, not by the implementation mechanism. Coverage is limited to the
workloads exercised here; add result-sensitive queries for new cache shapes.

When a new persisted or partly persistent optimization is introduced, extend
the public fixture and query workload with a result-sensitive shape that can
exercise it naturally. The SQL must run on every pinned predecessor as well as
the candidate. If syntax itself cannot be understood by a supported pin, add a
versioned public workload format rather than weakening an existing pin.

For local validation, create a disposable workspace containing built `base/`
and `candidate/` trees, then run:

```sh
GITHUB_WORKSPACE="$PWD" bash candidate/tests/storage/persistence/upgrade-run.sh
```

Optional `UPGRADE_API_PORT` and `UPGRADE_MYSQL_PORT` values select isolated
ports. Full-suite validation remains a CI responsibility.


## Table allocator provenance

The native table allocator now persists a version 1 `allocator` schema member
and checksummed `allocator-v1` database-authority WAL records. The state binds
the table UUID, actual configured start, known/unknown issued state, highwater
as monotone UUID-bound facts. No second counter infers issue order.
SAFE reservations are flushed before acknowledgement;
LOGGED retains process-crash durability. Unused accepted intervals survive row
rollback and savepoint rewind. MEMORY and SLOPPY keep their existing durability
contracts. Checkpoint and storage MOVE preserve the same authority records.

Historical `Auto_increment` fields remain readable. They prove only a
highwater, not the original start or whether that start has been issued. The
reader does not assign a guessed seed. Native `table_id_state` exposes unknown
facts explicitly; TSQL `IDENT_CURRENT` reports unsupported provenance rather
than confusing it with SQL NULL for a missing or inaccessible object. A later
real reservation can establish its actual current value without claiming the
unknown historical start. New constructors persist their real start.

This is a new writer format; older binaries are not supported writers of data
containing the new allocator authority. The UUID-bound cluster allocator uses
the distinct `cluster.table_ids` operation with explicit request/reply version
2. The previous `cluster.reserve_ids` operation is no longer registered, so an
older unversioned request cannot be mistaken for a version marker. Within the
new protocol, an empty identity is accepted only for an actual UUID-less,
provenance-free predecessor table and only for reservation. Its response has
an explicitly empty provenance payload and preserves the old highwater-only
allocation contract; it cannot reserve against a UUID-backed replacement or
support the new current-value observation. The historical row/WAL readers
remain unchanged. Independent noncluster replication writers retain
their existing highwater merge; this does not establish a global current-value
owner, so the new observation API rejects that scope explicitly.
`table-id-allocator.yaml` supplies current-writer value, rollback, crash, checkpoint and name-reuse controls; predecessor-writer
provenance coverage remains a separate qualification obligation.

### Realized ADD with original creation and bound names

The current candidate writes realized historical initializers in the strict
`catalog-v10/v24/` envelope. Its explicit submagic makes the qualified V23
predecessor reject a new uncheckpointed decision in both authority-first load
and participant-first enrollment verification; it cannot treat that decision
as an unrelated transaction identifier. The earlier standalone V21 envelope
remains readable with its original tables-free profile. V1–V20 and V22/V23
retain their existing domains, including rejection of the new initializer
fields even when explicitly null.

V24 requires a realized-row append and reuses the sole original decision,
ordered declarations, actual created-generation proof and participant recovery
owner. Created tables and stable name bindings are optional; table retirement
and unordered FK/property streams are excluded. The new compound full-value
controls create a real second table and primary key before initializing a
nonempty existing table, retain exact captured UUIDs through original own-row
versions, commit/rollback/savepoint and real restarts, and verify fresh physical
binding when a retired visible column spelling is reused. A real third-row
factory exception must unwind the whole ADD batch while preserving an earlier
private created generation and committable row prefix. No qualifier here
substitutes a plan or journal-shape oracle for public values.

Qualification of old-reader refusal before checkpoint and interrupted accepted
compound decisions remains the matching-binary runner's responsibility; this
source candidate does not claim execution or crash evidence until recorded.
