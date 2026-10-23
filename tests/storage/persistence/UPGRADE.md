<!-- Copyright (C) 2026 Carl-Philip Hänsch -->
# Upgrade compatibility contract

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

## What each matrix job proves

The predecessor binary creates the complete fixture and supplies two oracles:

- an exact snapshot of every public table's stable schema metadata and every
  stored value;
- results from a public SQL workload, including repeated grouping, aggregate
  ordering, expression group keys, correlated equality/range lookups and
  windows with opposing order directions.

The same SQL workload runs on the predecessor, after a predecessor restart,
after opening the data with the candidate, after cold and warm source-table
updates, and after a candidate restart. Each execution must return the declared
result and the candidate's unchanged executions must equal the predecessor's
saved query oracle. The updates are restored before the full data snapshot is
compared.

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
