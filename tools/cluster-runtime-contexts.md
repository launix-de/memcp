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
can have dictionary/tuple-valued physical accumulators. Callbacks, captures and
results use `scm.Serialize`. Dictionaries serialize as one bulk `set_assoc`
construction; capture constructors run before callback execution. Tuple keys,
NULLs and nested values retain their ordinary Scheme semantics.
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
representation or replace session calls with constants. The interface requires
`Serialize(*scm.SerializationContext) (scm.Scmer, error)`: the object chooses its
reconstruction expression. The printer and network value transport use that contract,
including standalone values without URI registration. Reference-backed objects
explicitly select the context's `ReferenceExpression`; routing-only endpoints
implement only `ClusterRequestHandler`. Runtime expressions reconstruct once at import.

Storage's scan-context binding lives in `storage`, supplied through the generic
`ProcedureDecode.Prepare` callback before compilation. The SCM procedure codec has no storage import, storage-operator binding rules
or dependency on a running storage engine.

`SerializationContext` captures one owner-defined view per statement. An optional
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

The owner registry pins native objects from serialization through a pending
transfer reservation. Import joins a shared peer lease before dropping that
reservation. The peer's weak directory deduplicates endpoints by owner,
incarnation and URI. Concurrent subscans retain that shared Go object; their
immutable session views and native TX participants stay query-local. There is no
per-use distributed reference count. Each new Go lifetime has a fresh lease ID,
so a delayed old GC release cannot revoke its replacement. Repeated imports of
one offer preserve value identity through a second weak directory.

`Serialize` constructors carry envelope-local capture IDs. `UnmarshalProcedure`
MUST construct those captures exactly once on the receiving peer, restore the
enclosed scope and then build the callable. This happens even for an empty shard
or a callback that is never invoked. Otherwise the owner's reservation could
outlive any real peer object and its missing GC callback would leak it. Ordinary
row code stays inert; row execution never reconstructs handles. Engine tests
force two overlapping empty subscans: after the first is collected the common
lease stays registered, and only the last GC release deregisters the owner.

`runtime.AddCleanup` only queues endpoint release, including callable cycles;
a worker confirms it through the existing channel. Last lease/offer removal
allows owner GC. An in-flight local handler retains its ordinary Go pointer;
weak retirement metadata protects its files and blob manifests until that
pointer is gone too. No GC callback performs network I/O.

A disconnected peer's established leases remain registered. A reconnect of the
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
that historical pointer. Native successor routing and object registration serve
distinct purposes: routing follows current authority, while the object registry
retains an exact historical generation until the last peer releases its handle.
Neither introduces a second shard identity or changes native snapshot lifetimes.

Distributed writes additionally need native participant prepare/commit/abort
and durable outcome recovery. Do not serialize the entire mutable TxContext as
one globally writable MOSI object: independent shards must stay independent.
Rebuild forwarding must translate physical row identities rather than merely
copy deletion masks. Unbounded grouped-result streaming and specialized physical
RecSet/merge-state transport are not implemented by this change and continue to
fail explicitly.

The message channel has one UUID collector protocol for calls and scan tasks.
There is no separate scan-task envelope or cluster execution switch: the
recipient resolves the database handler, which checks membership for shared
sources, reconstructs the request once and acquires its generation once. The
shard handler executes the prepared native invocation. A retained input supplies
an authenticated physical endpoint to this same path, without JSON embedded in
a Scheme string. Capacity refusals occur only before callback execution.

`ClusterRequestHandler` is the small routing contract. Serializable closure
values implement `ClusterObject`, which additionally requires object-owned
`Serialize`; routing-only databases, nodes and coherence cells do not.
Local text/hash serialization preserves native spellings. Network capture
markers belong only to explicit procedure serialization, never local plan keys.
Failed GC releases schedule their own bounded-rate retry; they do not rely on a
later GC or reconnect event to release a historical generation.


## Scan-family extension in progress

`--cluster-operators-only` compares fourteen SQL contracts with standalone results
and requires actual remote execution for every cluster case. It includes native
Top-N/OFFSET, computed/NULL ordering, an unlimited result spanning multiple pages,
LIMIT 1 without ORDER BY, EXISTS, scalar lookups, joins and ordered membership.
CI runs the same real-process scenario normally and with race-instrumented nodes.

The native request carries an explicit operator kind and invocation data through
the existing database/shard handlers. EXISTS uses its native short-circuit kernel;
lookup replies retain native match counts and cardinality checks. Batch strides,
row fields and negative boundary slots retain their binding semantics. Computed
boundary mappers use the same procedure serializer and import context as other
captures. Declared non-collation boundary relations use explicit wire constructors;
the persistent boundary codec still refuses process-local relations.

Ordered execution uses the native shard sort and the same coordinator merge for
local and remote queues. Workers freeze only comparison keys and requested column
values into an immutable result run, releasing native payload readers before
replying. These runs are query workspace, not another resident shard copy. The
run is a registered object with the physical generation's lifetime. Its initial
reply includes a bounded first batch. The coordinator assigns each shard job a
target of 16 to 4096 rows, dividing a 16384-row target window over the shard
fanout and capping it for LIMIT. The existing wire byte ceilings may shorten a
batch of wide rows. After consuming half the current batch, the coordinator
ACKs that progress and asynchronously requests exactly one successor batch.
Until then the peer retains its answers. The same serial merge consumes the
current window while the successor is in flight; callbacks flush within the
window instead of retaining projected rows across arbitrarily many batches.
LIMIT, errors and cancellation cancel and join the pending receive, then finish
the native shard job so it can release its answers before peer-handle GC.
Physical-generation handles retain their existing independent lifetime. No
per-row request or cancellation check is added. Serial callbacks run only for
globally selected rows, after partition/global offsets, limits and post-order
acceptance. This bounds the transport/merge windows; worker sort workspace and
the separate materializing join algorithm are not covered by that bound.

Without a combine function, ordinary scan callbacks retain their effects at the
coordinator. Remote workers perform the filter/projection and the same bounded
run mechanism supplies values to that callback. This decision follows the native
operator contract before serialization; it is not a retry after an arbitrary
callback/serializer failure. Reducible scans still compute native worker partials.

Join records retain immutable projected values and the imported run handle, so
late projection does not reload a shard locally or rebind old row IDs. Small outer
key sets become bounded worker-side semijoin filters before inner projection:
a four-key SUM must not transfer the full inner relation just to discard it at
the coordinator. These hints leave native shard fanout, runner combinations,
residuals, duplicate multiplicity and combine callbacks unchanged. Published O(1)
cardinality estimates avoid preparing hints for comparable-sized inputs; actual
encoded key bytes bound construction. An unavailable/oversized hint never
truncates keys or changes the authoritative join. Rebuilds
may publish a new UUID while the old physical input remains registered. The
existing weak peer handle/GC release protocol owns its retirement. Immutable
join/membership predicates serialize portable constructors: each worker builds its
lookup once, without process-local Go closures or per-row owner calls. Unportable
physical source RecSets/RecMaps still fail explicitly; these SQL cases do not
claim a general wire contract for those runtime containers.

None of these paths adds reader registration, row-loop cancellation probes,
shared-memory admission writes or another reader/writer gate. INVARIANTS.md is
unchanged.
