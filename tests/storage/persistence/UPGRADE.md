<!-- Copyright (C) 2026 Carl-Philip Hänsch -->
# Persisted helper upgrade coverage

`upgrade-run.sh` is the lifecycle used by the Upgrade Compatibility job and
local validation. The PR base revision writes the fixture using its own binary
and Scheme library. No candidate code generates the predecessor's schema.
`upgrade-helpers.py inventory` reads the resulting schema **after shutdown**
and fails if a required family was not materialized. The JSON inventory is
saved with the CI artifacts, alongside the unmodified application-data oracle.

| Persisted family | SQL producer / reader |
| --- | --- |
| Canonical group keytables | GROUP BY simple and expression keys |
| Aggregate temp columns | COUNT, SUM, ordered scalar subquery |
| Computed group sort columns | ORDER BY SUM(value) + 1 |
| Range boundary schemas | Correlated inequality + ordered LIMIT |
| Range aggregate state columns | Same range probe, asserted physical operator |
| Canonical lookup temp columns | Scalar ORDER BY over 4,000 driver rows |
| Ordered window columns | Partitioned ROW_NUMBER and LAG |

The old writer and its restart must produce the specified SQL results. The
candidate is checked after upgrade, after source DML and after another restart.
Cold DML occurs before helper queries can re-register dependency triggers;
warm DML verifies invalidation after materialization. Application rows are
restored and compared to the complete old-writer snapshot. Helper identities
may change: rebuilding, versioning or migrating them is allowed. Persisted
helper schemas are evidence of coverage, not required to remain byte-identical.

`upgrade-helper-negative.py` runs inside the **same existing CI job**. It changes
range boundary names and group key names in disposable Scheme copies without
changing their cache identity. For each mutation it requires:

1. The changed algorithm passes the complete lifecycle on fresh data.
2. The same algorithm fails with a missing-column error on the original
   writer's data, specifically during candidate validation.
3. A changed cache identity makes the complete upgrade lifecycle pass.

A setup error, arbitrary nonzero exit, or timeout is not accepted as the
negative proof. Mutation anchors are checked so future refactors must update
the control explicitly rather than silently turning it into a no-op.

This detects incompatible changes to exercised formats, not every conceivable
future format. When adding a persisted helper family, add a real SQL producer,
known result assertions, and a required inventory entry. If an existing helper
is retired, preserve an old-writer fixture for its supported upgrade path;
do not simply remove the coverage check to make a candidate green.

For local validation, create a disposable workspace with built `base/` and
`candidate/` trees (symlinks are supported), then run from that workspace:

```sh
GITHUB_WORKSPACE="$PWD" bash candidate/tests/storage/persistence/upgrade-run.sh
python3 candidate/tests/storage/persistence/upgrade-helper-negative.py candidate negative-results
```

Optional `UPGRADE_API_PORT` / `UPGRADE_MYSQL_PORT` override isolated test ports.
The full test suite remains a CI responsibility.
