<!-- Copyright (C) 2026 Carl-Philip Hänsch -->
<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Transparent cluster execution

Design for completing cluster transparency of all storage operations under
[INVARIANTS.md](INVARIANTS.md). This proposal describes the implementation
target; it does not claim that the current engine already implements it.

The unit of delegation is an ordinary native operation on an addressable
object, usually a shard. A peer acquires the necessary MOSI rights or delegates
the operation to a peer that can execute it. The existing operator, Scheme
lambda and transaction implementation supply semantics on either peer.
Placement minimizes expected communication subject to available memory and
the operation's rights, visibility, ordering and lifetime requirements.

## One execution path

Keep SQL planning independent of physical topology. Filters, MapReducers and
their captures use the existing Scheme serializer. Native objects expose the
existing `ClusterRequestHandler.HandleClusterRequest` interface; runtime
values additionally own their serialization. Object policy stays with the
object. Transport routes messages and carries results.

Every shard access passes through `GetRead` or `GetWrite`. An already
authorized resident access remains the normal fast path. Otherwise access
preparation acquires rights and admits memory, or raises the existing typed
acquisition failure before executing that operation's effects. The enclosing
native operation can then delegate its complete shard invocation. The receiver
calls the same native implementation, including its ordinary access checks.

For mutable state, choose between acquiring M and delegating the task to its
holder. Read-only work may use a valid O/S copy or an immutable prepared view.
Compare the expected bytes and exchanges for state transfer against the task,
its captures, results and subsequent dependent accesses. Prefer an already
authorized local execution when it is cheaper. Keep this decision in common
placement infrastructure rather than separate operator heuristics.

Delegation never replays an invocation whose effects may have started. Capacity
refusal before execution may select another peer; ambiguous completion is an
error with the native transaction outcome preserved, not permission to retry.
This restriction also applies when a nested operation fails after its caller
has already performed effects.

## Shard walkers and captured objects

Scans, RecSet/RecMap walkers and computed-column preparation or repair use the
same execution model. Their different operator contracts do not require
different distribution architectures.

A RecSet or RecMap retains shard-associated state near its producer or
consumer. A walker delegates work to those parts or transfers them when that
costs less. It does not first gather all bitmaps, row IDs or rows at the client
peer. Physical IDs remain tied to their original shard generation and query
lifetime; acquiring a successor never reinterprets them. Native references
retain those generations through the existing registration and lease protocol.

Computed columns retain their complete logical definition and dependency
closure. Preparation filters only select work to prewarm. A delegated repair
uses the ordinary implementation, including the ordered dependency range for
ORCs. Publication and reuse preserve source consistency and transaction
visibility. Shared cache entries retain logical keys and values, not captured
transactions or physical query-local IDs.

The worker executes the complete MapReducer, including subscans. Each subscan
starts its ordinary access decision on that worker: use local state, acquire a
shared copy, acquire modification rights, or delegate further. A logical table
handle does not force those accesses back through the client peer.

Sessions, cachemaps, return channels and other captured objects follow their
native access contracts through the same handle infrastructure. Immutable
prepared views permit local reads; mutations use the appropriate owner endpoint
or ownership transfer. No row-level network lookup, registration, reader counter
or extra synchronization is introduced merely because a capture is remote.

## Transaction identity and global allocation

Keep numeric transaction IDs. Introduce a small addressable `global` object
containing the peer roster and a monotonic next-free ID. Any peer can briefly
acquire M, reserve a half-open range of 10,000 IDs, advance and durably publish
the upper bound, and then allocate locally without further cluster exchanges.
The reservation is acknowledged only after it is safe against reuse following
a crash. Exhaustion or integer overflow fails explicitly rather than wrapping.

Unused IDs are never returned. A restarting peer abandons its old local range
and reserves a new one. A lost reservation acknowledgement may waste a range;
it must never cause a range to be allocated twice. Transaction IDs remain
unchanged on delegation, and receivers register the existing transaction.

Bootstrap establishes one authority for this object when creating a cluster;
joining peers discover it rather than initializing competing allocators.
There is no permanent initial-node role. Owner transfer, orderly shutdown and
recovery follow the existing MOSI rules. A failed owner does not prevent peers
from using already reserved ranges, but new reservations wait for its return
or explicit administrative recovery. Timeouts do not manufacture authority.
The roster is discovery state, not a substitute for each object's actual rights.

Session/connection IDs and transaction IDs remain separate identities with
different lifetimes. MySQL `CONNECTION_ID()` continues to identify the original
client session throughout delegation; a worker's local process registration
does not replace it. Numeric protocol fields retain their supported width.

## Coupled shard and transaction coherence

A native TX-shard is identified by transaction ID and physical shard UUID in
its database. It owns the existing visibility overlays, undo information and
savepoint-relevant state. It remains an addressable object, but initially shares
the shard's MOSI coherence domain instead of negotiating independent rights.
This is a shared grant and handover protocol, not an additional state machine
per overlay.

Under M, a peer may create or modify TX-shards belonging to that physical
shard, subject to ordinary transaction rules. The grant covers the complete
associated state. Under O/S, imported TX-shard state is read-only. A subsequent
overlay change, including commit or rollback, requires M or delegation to the
authorized peer. This deliberately uses shard-level write exclusivity rather
than independent overlay writers on multiple peers.

Giving up M proactively transfers all relevant TX-shards with the shard's
state. M-to-M transfer installs exclusive authority at the destination;
O/S readers receive consistent read-only copies. Handover closes new mutation
admission and drains admitted mutations using the existing native ownership
boundary before acknowledging the new grant. It includes newly created
overlays, removals, undo state, savepoint bookkeeping and participant identity.
Later mutations invalidate or refresh shared state through the same protocol.
Copying only DeleteMask/UndeleteMask once is insufficient.

Rebuilds preserve distinct physical UUIDs and the existing mutation-forwarding
contract. TX-shard state and commit/rollback visibility changes participate in
that handover while predecessor and successor remain live. No transfer may
lose a mutation, apply it twice logically or resurrect a deleted row.

## One transaction across peers

Separate the transaction's stable identity and participant state from the
invocation's cancellation, readers and node-local execution budget. A remote
invocation attaches to the existing transaction and registers its participants;
it does not create an independent transaction or an untracked private overlay.
Normal scans already support `$update` and related mutation callbacks. Once
rights and transaction binding are correct, those same kernels execute remote
UPDATE/DELETE, including native trigger and constraint effects.

Extend the existing transaction implementation at its owning abstraction so
local and remote participants use the same validation, undo, WAL and
publication steps. MOSI coherence does not replace atomic commit: all
participants follow one transaction decision, with the durability and recovery
needed by the selected engine. Cursor-stability and ACID retain their distinct
publication modes. A decision owner is transaction-scoped, not a permanent
cluster master. An uncertain outcome remains uncertain until resolved by the
native recovery protocol; independent per-peer commits are not sufficient.

Savepoints, rollback, cache invalidation, INSERT, duplicate-key handling and
trigger-induced work use that same transaction identity and participant set.
DDL and schema operations acquire their native schema rights or delegate to
the authorized peer; they do not bypass this model with frontend-specific RPCs.

## Results and resource ownership

Independent shard work uses the existing fire-and-forget delegation and
registered return channel with collect-and-count completion. Combine results
only as permitted by the native operator. Without a combiner, preserve the
native accumulator dependencies while keeping the callback on the executing
worker. Global ordered operators retain their merge and ordering contract;
callback execution follows that order through bounded batches on workers.

Streaming queues, retained partials and ordered workspace count against
node-local budgets and use backpressure. Do not gather an unbounded relation
into lists or hide ordered-run memory outside the budget. Cancellation and
early termination stop and join owned work before releasing its state. The
original session's cancellation reaches all delegated descendants without
adding cancellation checks to row loops.

Strong registrations and imported leases retain physical generations and
captured resources until their work completes. GC queues releases; disconnect
neither revokes ownership nor proves an object can be deleted. Eviction and
lease cleanup never delete persistent user data.

## Implementation and acceptance

Implement the shared infrastructure before adding operator-specific wiring:

1. Add `global` authority, durable ID range reservation and native transaction
   identity propagation.
2. Extend shard handover to include complete TX-shard state and bind remote
   invocations to the existing transaction participants.
3. Complete shared acquire-or-delegate execution, transaction decisions and
   worker-local callback/subscan execution.
4. Connect all storage operations, including RecSet/RecMap walkers, computed
   columns, ordered operations, INSERT and schema mutations, to that path.

Acceptance compares identical workloads standalone and with forced remote
execution, using placement evidence rather than only low RAM settings. A
remote UPDATE followed by a nested scan and ROLLBACK is an initial integrated
case, not the complete acceptance suite. Cover every storage operation and
its critical combinations: both transaction modes, commit and savepoints,
constraints/triggers, source-consistent caches, ordered limits, ownership
transfer, rebuild, cancellation, and owner failure/restart. Verify ID range
uniqueness across concurrent reservations, transfer and crash recovery.

Extend the existing cluster harness and focused regression suites. Tests that
perform mutations locally and delegate only later reads do not demonstrate
remote mutation support. Required checks must pass without weakening semantic
assertions, timing budgets or CI gates. Keep local artifacts and test data in
the worktree; the full suite runs in CI.
