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
can have dictionary/tuple-valued physical accumulators. The version-4 scan wire
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

**Reconstruct captured objects.** `Serialize` emits a `cluster_object` call with
an owner URI, process incarnation, type and independent reference token. The
receiver resolves these calls before Optimize/Eval, preserving repeated capture
identity. `scm.ClusterObject.HandleClusterRequest` is the shared endpoint contract
for sessions, cachemaps, transactions, transaction-shard visibility views,
physical generations and catalog objects. Serialization does not switch on their
representation or replace session calls with constants.

`ClusterBindings` captures one owner-defined view per statement. An optional
object-owned projection restricts it to demanded keys. Imported session/cachemap
prepared reads use immutable local views without a lock, CAS, network request
or shared RAM write per row. A cachemap miss uses its owner stub and publishes
the result privately; explicit mutations and factories go to the owner. This is an invocation
snapshot, not a promise of continuously refreshed shared session state. The
callable ABI and 16-byte Scmer representation remain unchanged. Plans retain no
invocation instance. Nested captures and quoted captured containers use the same
reference codec; ordinary quoted program text remains inert.

**Borrow native transaction visibility.** Tasks carry a registered transaction
reference; the object binds its read participant to the receiving invocation's
native executor and cancellation context. Modified generations retain their
native active-TX pins and overlays locally; untouched generations can execute
remotely. Overlay references expose a private deletion/undelete view tied to the
exact physical UUID. This is neither distributed mutation nor commit authority.
Process-local commit epochs do not become a global snapshot by copying a counter.

**Publish committed cache invalidation.** Another session can reconstruct a
pre-COMMIT cache while ACID changes are staged. Successful ACID COMMIT therefore
invalidates dependent computed caches after visibility publication and before
returning. It uses the existing cache generations, contribution guards and
preparation-time coherence checks also used by MOSI revocation. It adds no
row-loop synchronization or new shard read locks.

## Object dispatch and lifetime

The authenticated reader dispatches messages to independent goroutines. The
common handler resolves the node, database, resource cell or registered object
and delegates business policy to its `HandleClusterRequest` implementation.
MOSI packets retain FIFO order per resource, including payload installation
before permission publication. A data resource also waits for a preceding schema
publication introducing its namespace. These waits belong to workers, never the
reader. Independent shards and databases remain parallel. Pending packet bytes
are bounded. Requests and replies are fire-and-forget envelopes; the outer
operation collects replies using registered UUID channels. Panics, including SQL
errors and their originating stack, cross this same channel. Cancellation fences
shard tasks and propagates through nested invocations, without row-loop probes.

The owner registry holds the native object strongly while remote tokens exist.
Every export gets an independent token bound to the authenticated peer's process
incarnation. Offers become acquired handles before reconstruction; failed or
abandoned messages discard only unacquired offers. Repeated imports of one token
share a weakly cached Go lease, preventing one alias from releasing another.
The imported value retains that lease. `runtime.AddCleanup` enqueues a small
release when it becomes unreachable, including callable cycles; a worker delivers
and confirms the release through the existing channel. The GC callback performs
no network I/O. Last peer release deregisters the owner object. An in-flight
local handler still retains its ordinary Go pointer. Weak retirement metadata
keeps its files and blob manifests protected until that pointer is also gone.

A disconnected peer's acquired references remain registered. A reconnect of the
same authenticated process can continue or deliver releases. This protocol does
not infer death from a timeout, steal M rights, or implement global tracing of
cycles spanning owners. Explicit failed-peer retirement and uncertain transaction
decisions remain separate administrative/protocol work; reference release grants
no MOSI permission. Name-resolved catalog objects use dynamic URIs without strong
temporary registration. Shared-backend GC still requires a cluster-wide deletion
proof and remains disabled by its existing safety guard.

The `--remote-scans-only` Python scenario now also tests retained generations
using two real processes. The peer retains physical shard handles and pauses
outside row execution; the owner rebuilds to new UUIDs. Peer GC cannot release
live handles. After resuming, the peer reduces the old generations and verifies
the expected SQL SUM. Once its values become unreachable, peer GC releases the
old registrations. A subsequent SQL UPDATE and fresh SUM verify that current
topology and coherence were not affected. Artifacts record old/new UUIDs and the
release result. Engine tests additionally cover independent tokens, foreign
incarnations, duplicate imports, abandoned replies, nested runtime captures,
transaction visibility views and parallel dispatch/publication ordering.

Native UUID forwarding from PR #1193 is integrated. Physical endpoint identity
is the actual generation UUID; a logical resource cell is not substituted for
that historical pointer. This interface can be implemented by the forwarding
code without introducing a second registry or changing native snapshot lifetimes.

Distributed writes additionally need native participant prepare/commit/abort
and durable outcome recovery. Do not serialize the entire mutable TxContext as
one globally writable MOSI object: independent shards must stay independent.
Rebuild forwarding must translate physical row identities rather than merely
copy deletion masks. Unbounded grouped-result streaming and specialized physical
RecSet/merge-state transport are not implemented by this change and continue to
fail explicitly.
