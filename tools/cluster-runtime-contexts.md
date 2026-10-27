<!-- Copyright (C) 2026 Carl-Philip Hänsch -->
<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Cluster SQL contexts: investigation and proposed architecture

This is an investigation of master `68ca280c9` (#1191), not an implemented
runtime-object protocol. This patch changes tests and documentation only.

## Run and evidence

```sh
python3 tools/replication_cluster_stress.py --binary ./memcp \
  --cluster-context-only --artifacts /tmp/cluster-contexts --seed 20261027
```

The process runner executes the same 14 SQL cases first standalone, then on
four authenticated cluster processes sharing a backend. Workloads are SQL;
existing Scheme infrastructure only configures fixtures, RAM and diagnostics.
The large table has 64 shards / 320,000 rows. An eight-row overlay fixture has
four shards. Cold literal/nonlinear controls require actual remote execution;
valid cache reuse is permitted for repeated session queries. For distributed
TX overlays the coordinator admits its two modified shards, then restores the
small RAM threshold before the full scan. Warm local inputs must remain usable.

Per-case JSON records SQL, sessions, returned/expected rows, errors and placement
evidence. Failure does not suppress later cases. Cleanup rolls back transactions
and checks post-rollback visibility even if the distributed query failed. Normal
and race CI context steps upload these reports and fail for unsupported behavior;
there are no expected failures or continue-on-error gates.

Normal/race runs on the tested master agree:

| Contract | Standalone | Cluster |
| --- | --- | --- |
| Literal expression in SUM | Pass | Pass; remote work observed |
| Plain SUM with named HTTP session | Pass | Pass |
| `SUM(@somevalue + v0)` | Pass | Rejected |
| Nonlinear `SUM((@somevalue + v0) % 97)` | Pass | Rejected |
| Rebinding identical cached SQL in one session | Pass | Rejected |
| Two concurrent sessions with different values | Pass | Rejected |
| NULL/value rebinding | Pass | Rejected |
| Partition filter using a session variable | Pass | Rejected |
| Explicit CS / ACID large-table reads (two cases) | Pass | Rejected |
| Four-shard CS / ACID rollback (two cases) | Pass | Pass on one worker |
| ACID delete/update overlays + distributed scan | Pass | Rejected |
| ACID COMMIT + cached SUM / ordered row read | Stale SUM | Stale SUM |

A locally successful overlay is not evidence of distributed transaction support.
The COMMIT case fills a shared SUM cache from another session while ACID writes
are hidden. After COMMIT, ordered rows correctly total 425, but SUM remains 360.
This is an independent cache-publication defect, not demonstrated loss of rows.
Investigate/fix it separately before trusting post-commit cached aggregates as a
cluster-coherence oracle. The patch preserves the failing assertion.

## Actual implementation boundaries

`clusterScanSupported` accepts CS autocommit only. `clusterScanExecutor` also
rejects slice/dictionary neutral values. EXPLAIN shows that session-dependent
SUM uses the normal group/cache pipeline, keyed by the session-value domain,
with grouped intermediate accumulators. A scalar SQL result does not imply
scalar physical intermediates. The observed rejection therefore happens before
the procedure codec can determine whether its session capture is transportable.

`remoteScanTask` carries a captured row horizon and a visibility-contract string,
not the caller's transaction/shard-overlay identity. Workers create fresh
CS-autocommit transactions. The procedure codec rejects process-local callables,
including an unexported `newsession`. Fixing only the serializer cannot remove
these independent restrictions.

`SnapshotEpoch` and `GlobalCommitEpoch` are process-local. Current `IsVisible`
applies native deletion and transaction overlay masks; copying the counter would
not implement a global historical snapshot. Define the intended isolation
contract explicitly rather than introducing a stronger accidental promise.

## Proposed architecture, for review

1. **Typed runtime reconstruction calls.** Adopt the proposed
   `(getsession "https://owner:port/cluster/<uuid>")` model, with corresponding
   typed transaction/overlay references. The URI identifies creator/home, object
   and lifetime/incarnation. Separate stable peer/object identity from the
   advertised network address. A restarted process must reject old objects,
   never silently instantiate an empty replacement. Exporting references retains
   the existing cluster authentication / replica user authorization boundaries.

   Integrate this at capture preparation, wire validation and reconstruction,
   not just the printer. Resolve each exported capture once into a receiver-owned
   invocation frame before scanning. A resolver call retained in every row's AST
   is still wrong even if its RPC response is cached. Persistent plans retain
   reconstruction instructions, never a previous invocation's session or TX.
   Multiplex object traffic on the existing bidirectional node channel with
   UUID result channels; no connection or RPC sender handle per object.

2. **SQL session inputs become immutable statement bindings.** Capture the
   required user-visible values once per statement and close physical callbacks
   over them. Preserve NULL, frontend semantics, cache domain keys and rebinding.
   Do not copy internal TX/password/query fields with the whole native session.
   For general sessions the resolver can provide an S read view and an owner
   mutation stub. Initially keep mutation authority at the creator; do not
   implicitly move M between analytic workers. Unknown dynamic keys or effects
   need bulk preparation / owner co-location, not per-row network requests.

3. **Keep the existing planner and reduction pipeline.** Extend the common
   partial-result codec for native grouped accumulators and merge semantics.
   Preserve the logical/physical phase boundary, group domains and caches.
   Do not rewrite SQL into a separate remote planner, disable caching or serialize
   shard execution to make these tests pass. Larger grouped results need bounded
   streaming; RecSets/RecMaps remain query-local physical objects.

4. **One TX coordinator, independent native shard participants.** The creator
   owns transaction control/terminal decisions. Send logical TX identity, mode,
   statement/cancellation identity and visibility bindings explicitly. Reuse
   native participant contexts and WAL instead of unrelated worker autocommit.
   Do not make the entire mutable `TxContext` one globally writable MOSI object:
   that would serialize independent shards. Native TX allocations are reused;
   the URI must distinguish logical lifetimes after COMMIT/ROLLBACK, not merely
   pointer identity. Process-local statement counters alone are not network-wide
   cache or cancellation identities.

5. **One overlay resource per TX and physical shard generation.** Key it by
   transaction URI, shard UUID and revision/generation. Its creator owns mutation
   and publication; prepare an immutable native read view per shard invocation.
   Row visibility remains local bitmap/handle reads. Mutations, savepoints and
   terminal decisions use participant messages, never per-row RPCs. Retain native
   active-TX pins, max record bounds, undo data and rebuild forwarding; changing
   shard UUID must also translate overlay row identities. A copied bitmap without
   its physical generation, or a serialized Go pointer, is insufficient.

6. **Specify cross-node commit/recovery before remote writes.** Use native
   durable prepare/commit/abort records, not a MOSI journal or backend bypass.
   Cancellation before prepare differs from failure after durable prepare;
   an unanswered request cannot be treated as abort. Query-owned read exports
   may expire on completion; prepared write participants need explicit recovery.
   No timeout promotes M or discards an uncertain decision.

## Decisions still needed

- Freeze SQL session reads at statement start? Recommended: yes. Which mutations
  and dynamic-key cases need owner stubs or co-location?
- Allow session M to migrate? Recommended initially: creator-owned mutations and
  S read snapshots only; generic object types can choose another policy later.
- First TX scope? Recommended: preserve native CS/ACID visibility, introduce
  read-only TX binding, then native shard participants and a defined commit
  protocol. Full historical distributed snapshot isolation is a separate choice.
- Which runtime references survive reconnect? Recommended: explicit scoped
  leases and incarnation fencing; ordinary query exports die with their owner,
  prepared TX state follows its durable recovery contract.

Fix the independent local COMMIT/cache defect in its own patch. Then implement
grouped partial results and statement session bindings before distributed write
participants. Keep these SQL contracts active throughout that work.
