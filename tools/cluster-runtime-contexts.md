<!-- Copyright (C) 2026 Carl-Philip Hänsch -->
<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Cluster SQL contexts

## SQL process contracts

```sh
python3 tools/replication_cluster_stress.py --binary ./memcp \
  --cluster-context-only --artifacts /tmp/cluster-contexts --seed 20261027
```

The runner executes the same 14 SQL cases standalone and on four authenticated
cluster processes sharing a backend. Workloads are SQL; Scheme infrastructure
only configures fixtures, RAM and diagnostics. The large table has 64 shards /
320,000 rows. An eight-row overlay fixture has four shards. Cold literal and
nonlinear controls require actual remote execution; repeated queries may reuse
valid caches. For transaction overlays the coordinator admits its two modified
shards, then restores the small RAM threshold before the distributed scan.

The cases cover literal and session-dependent SUM, nonlinear expressions,
rebinding the same SQL, simultaneous independent sessions, NULL rebinding,
session-dependent partition filters, explicit CS/ACID reads, CS/ACID rollback,
local ACID update/delete overlays combined with remote reads, and COMMIT cache
coherence. The COMMIT case deliberately fills an aggregate cache from another
session while staged changes are hidden, then checks both rows and aggregates.

Per-case JSON records SQL, sessions, returned/expected rows, errors, EXPLAIN and
placement evidence. A failure does not suppress later cases. Cleanup rolls back
and checks visibility even if a distributed query failed. Normal and race CI
steps upload these reports and fail on unsupported behavior. No expected failures
or continue-on-error gates are used.

Before the implementation, master `68ca280c9` passed 13/14 standalone cases and
4/14 cluster cases. Its post-COMMIT SUM was stale even standalone; session-bound
reductions and explicit distributed transaction reads were rejected. The
implementation passes all 14 cases standalone and in the cluster in both normal
and race-instrumented process runs. The existing five-round replication crash /
multi-slave regression also passes. Detailed artifacts belong in the PR
validation record.

## Implemented execution contracts

**Keep native planning and reduction.** Session-dependent scalar SQL aggregates
can have dictionary/tuple-valued physical accumulators. The version-3 scan wire
codec preserves native dictionaries, tuple keys, NULLs and nested aggregate
payloads. Parser source locations are diagnostic wrappers, not runtime objects.
The receiver applies the existing reducer; there is no alternate SQL planner,
cache bypass or serial scan fallback. Frames retain the existing bounded size;
unbounded grouped-result streaming is not implemented by this patch.

**Preserve filter constraints.** Delegation uses the existing native boundary
codec and materializes its bound operands. NULL versus an unbounded endpoint,
mandatory constraints, collation and coverage survive transport. Computed
mappers, physical RecSets and specialized invocation-owned merge state fail
explicitly until they have a portable execution/lifetime contract.

**Freeze demanded session reads.** Physical procedure capture preparation shares
one lazy statement-owned session snapshot across dispatched shard callbacks.
Constant-key reads close over immutable data before wire validation. Only demanded
values enter the wire; internal session maps, TX objects, credentials and native
closures are not serialized. Session identity remains recognizable through HTTP
and MySQL frontends without changing the native callable ABI or Scmer size.
Snapshot work and synchronization happen at dispatch, never per row. Persistent
plans retain neither the session snapshot nor a previous invocation's bindings.
Dynamic keys, session mutations and arbitrary runtime objects still require an
explicit owner contract; they are not converted into per-row network calls.

**Borrow native transaction visibility.** Read tasks carry their logical origin
identity, CS/ACID mode and autocommit status. Remote execution creates an
invocation-owned native read participant. Modified input generations remain
resident with their existing active-TX pins and native overlays; exporting such
an input without its participant is rejected. Untouched generations can execute
remotely. This supports a transaction with local update/delete overlays plus
remote reads; it does not implement writes on multiple nodes or distributed
commit/recovery. Process-local commit epochs do not become a global historical
snapshot by copying a counter.

**Publish committed cache invalidation.** Another session can reconstruct a
pre-COMMIT cache while ACID changes are staged. Successful ACID COMMIT therefore
invalidates dependent computed caches after visibility publication and before
returning. It uses the existing cache generations, contribution guards and
preparation-time coherence checks also used by MOSI revocation. It adds no
row-loop synchronization or new shard read locks.

## Runtime object interfaces and lifetime: follow-up

General reconstruction calls can use typed object URIs such as
`https://owner:port/cluster/<uuid>`. A common `HandleClusterRequest` interface
should dispatch through the existing authenticated bidirectional node channel;
no sender/connection handle should be retained per object. Catalog names can be
resolved dynamically. Sessions, transactions and physical shard generations need
explicit identity and incarnation. Resolve invocation bindings before row loops.

The agreed lifetime direction is a strong owner registry with peer-reference
tokens. A peer keeps its handle while native readers still reference the object;
GC/finalization queues release of that handle to the owner. Last peer release
allows deregistration. Distinct tokens prevent a late old release from clearing
a newer handle from the same peer. Finalizers must enqueue, not block on network
I/O. Reconnect, failed peers and uncertain transaction decisions need an explicit
protocol; no timeout may take M from an offline replica.

The required old-generation test is: a peer retains the old shard for an active
scan, the owner publishes a rebuilt UUID, and the peer finishes using the old
generation before releasing it. Only that old registration is retired; neither
new-generation rights nor the old scan's visibility may change. UUID forwarding
is being developed separately and is not a prerequisite for the SQL contracts
above. This patch neither imports that unpublished implementation nor claims
that the object registry and its GC protocol are implemented.

Distributed writes additionally need native participant prepare/commit/abort
and durable outcome recovery. Do not serialize the entire mutable TxContext as
one globally writable MOSI object: independent shards must stay independent.
Transaction overlay identities must include the physical generation, and rebuild
forwarding must translate row identities rather than merely copy deletion masks.
