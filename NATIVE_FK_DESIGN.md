<!-- Copyright (C) 2026 Carl-Philip Hänsch -->
<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

# Native foreign-key admission: proposal for review

**Design and executable specifications only. No new engine authority is
implemented by this change.** Baseline: `d72bfbfccb4b` (after the frontend identity and hint changes).

## Reuse the existing implementation

- `table.Foreign` / `foreignKey` in `storage/table.go` are the persisted
  constraint definitions. Both endpoint copies describe the same logical FK.
- `createtable` consumes `("foreign" name child_columns parent_table
  parent_columns update_mode delete_mode)`. `createforeignkey` and
  `dropforeignkey` in `storage/storage.go` already alter these definitions.
- `installFKTriggers` installs canonical system programs using
  `__fk_check_ref`, `__fk_on_parent_delete` and `__fk_on_parent_update`.
  Priority 100 validates transformed NEW after ordinary BEFORE programs.
  Restoration regenerates canonical programs from native FK definitions.
- `fkExistenceCheck` uses native exact-bound probes. Native scans and update
  callbacks implement restrict, cascade and set-null. Any NULL child-key
  component currently exempts that row from checking (simple matching).
- `TxContext` already owns visibility, undo, savepoints, commit/rollback,
  WAL outcomes and rebuild/repartition forwarding. It must keep that role.
- Logical-table `uniqueWriteLocks` already has 64 batch guard stripes with
  conservative coercing-equality hash domains, independent of shard identity.
  These are short-lived exclusive mutexes, not transaction reference claims.

There is also existing frontend policy leakage to remove in the implementation:
`foreignKeyChecksEnabled` in `storage/storage.go` reads the SQL-named session
and global variable `foreign_key_checks`. Resolve this in `lib/` at executing
statement/batch preparation and pass a generic native admission policy. Do not
add more SQL-variable lookups to generated row checks, and do not let one SQL
frontend's global switch silently disable constraints for the other frontends.
Any permitted unchecked mutation must invalidate an old-row validation claim
before it becomes visible; switching checks on cannot restore trust by itself.

Persist only semantic constraints. An FK requires a real parent candidate
key, never a declared physical access structure. Automatic indexes remain
planner/operator choices, with no names, kind or placement added to the schema
or compatibility catalogs. A UNIQUE declaration remains a native constraint
even when its syntax also mentions an index kind.

## Two distinct correctness gaps

1. A child can pass the existing parent probe, then a concurrent parent
   deletion can commit before the child is published. Both generated trigger
   sets can run and the result can still be an orphan. Cursor-stability undo
   also needs protection: a tentative parent birth must not support a foreign
   child that outlives parent rollback; child deletion rollback must not
   resurrect a reference after the parent is deleted.
2. `createforeignkey` currently publishes definitions and triggers without
   checking old rows. Adding a frontend scan before that call leaves a
   check/publication race. Trigger ordering alone does not close either gap.

The existing mandatory ALTER enforcement regression stays mandatory. Passing
an expected-error test because ALTER syntax is rejected does not prove FK
validation. New specifications therefore include successful installation and
post-install enforcement; rejection cases require an FK-specific error.

## Preferred direction: partitioned, CAS-published logical-key claims

The earlier 64-stripe lifetime-claim proposal is not approved. A stripe may
route a directory lookup, but must not grant exclusive transaction authority
over unrelated keys that happen to hash into it. Physical shards alone cannot
own this authority: a referenced key can be reached from several child shards,
and rebuild, repartition and rollback can change its physical location.

Partition an optional directory by logical key hash, with a separate atomic
claim state for each relevant key/comparison domain. This is a runtime adjunct
to the existing native candidate key, not another persisted constraint or row
index. Directory partitioning limits allocation/publication contention; normal
claim acquisition for an existing entry updates only that entry. No table,
shard or directory mutex is held across the transaction. No claim operation is
added to SELECT, planner metadata, or unrelated tables.

Each immutable published state describes shared reference owners and an
optional exclusive key-change owner, identified by transaction ID. Batch setup
deduplicates requested keys; CAS admits a compatible state, or returns a
retryable serialization conflict. A child batch sharing the same parent key
can coexist with other child batches. Only a conflicting parent-key mutation
needs incompatible authority. Authority is not a membership proof: the
existing transaction-visible parent probe still decides whether the row exists.

The ownership state and its transaction journal must agree even on admission
failure, trigger failure and cancellation. A journal entry records the acquired
entry identity, role and savepoint/reentrancy delta. Concurrent tasks belonging
to one transaction use its existing transaction-state ownership; a reusable
TxContext pointer is not an owner ID. Release follows the existing durable
outcome or completed visibility undo, never precedes it.

Directory retirement needs an explicit protocol too. An ownerless entry is
first CAS-marked retired; a concurrent acquirer either wins admission before
retirement or observes the retired state and retries lookup. Only then may
directory publication unlink that exact entry. Retained native pointers keep
the immutable retired object alive. Never reset/reuse a live entry in place,
nor retain every historically referenced key until the table is dropped.

### Comparison and performance gates before implementation

- Reuse the real native key comparison contract. Hash collisions must resolve
  in the directory and must not establish equality. The existing coercing
  equality admits overlapping representation domains; merely applying
  `HashKey` to a runtime value is unsound. An exact canonical key is permissible
  only where equality/hash compatibility is proved. Otherwise protection must
  overlap conservatively, with the possible false-conflict cost documented.
  SQL datatype and collation decisions remain in frontend compilation.
- Do not replace the old 64 mutexes with 64 exclusive CAS owners: that keeps
  the same unrelated-key contention for the complete transaction lifetime.
- A naive immutable owner vector copies all concurrent owners on each CAS.
  Benchmark hot-parent workloads before selecting that representation; a CAS
  is not automatically cheap or allocation-free. Any alternative count/owner
  representation must still prove reentrancy, cancellation and savepoint undo.
- Admission is a native write-batch operation. Do not add per-row shared
  counter writes, per-reference goroutines, frontend scans, or an unbounded
  Scheme collection of relational keys.
- Test entry retirement versus admission, hash collisions, shared hot-parent
  batches, unrelated-key progress, same-key parent mutation and transaction-ID
  reuse in addition to the transaction/rebuild schedules below.

This is a design refinement for architecture review, not authorization to
implement a new directory, comparison contract or transaction journal.

## Transaction lifetime requirements shared by all candidates

Retain the existing native key definitions and generated enforcement programs;
the missing support is optional, transient shared-reference and exclusive
key-change authority. This is a proposal, not an existing directory or API.
Tables without relevant FKs allocate no claim directory;
ordinary SELECT, planner metadata and read loops perform no claim operations.

A child acquires shared parent-key authority **before** the native existence
probe and retains it through its transaction outcome. Parent deletion/key
replacement acquires incompatible exclusive authority **before** checking
children. Protect parent birth and old child references which undo can restore.
Own-transaction acquisition is reentrant; membership is still decided by
native scans with that transaction's visibility. It must never be inferred
from hashes or claim ownership. User-trigger changes require final native
admission for actual NEW values before publication.

Claims use logical table/key-column identity and tuple values, not physical
record or shard IDs. Native batch preparation deduplicates the required claim
set; broad/coercing domains conservatively overlap. A conflicting foreign owner
causes a retryable serialization failure; it does not wait with shard locks,
UNIQUE guards or a parked transaction holding a mutex. Shared references can
coexist, and unrelated keys can proceed independently.

Use the existing `TxContext` for a lazy acquisition journal, identified by
`TxContext.ID` rather than its reusable pointer. Savepoints record journal
boundaries and reentrant count deltas. Restore/hide rows and forward their
visibility first; release the corresponding authority second. Commit releases
only after existing durable-outcome handling; an unknown commit outcome must
follow the existing committed interpretation. Trigger failure, cancellation,
abort, savepoint rollback, full rollback and session reset clean up exactly
once. Nil-transaction native calls need equivalent invocation-owned cleanup.
CASCADE/SET NULL use the existing transaction and native mutation callbacks.

An atomic claim state needs a reviewed ownership/count representation and
bounded acquisition retry policy. Merely replacing a mutex with an atomic
counter does not provide shared owners, reentrancy, savepoints or safe undo.
Conservative equality can still cause false conflicts; measure the supported
comparison domains before claiming independent-key parallelism.

### Other candidates and limits

- Rechecking native FK predicates at ACID commit under the existing key guards
  may close final publication for staged writes. It still needs a dependency
  journal and does not alone protect visible tentative cursor-stability rows
  or their undo. A global/table commit gate is rejected.
- Physical parent-row reference counters would need owner identity, undo and
  transfer across updates/rebuilds. They have a larger generation-lifetime
  footprint and no demonstrated advantage over logical-key authority.
- Retaining a shard read lock or table mutex through the transaction is
  incompatible with `INVARIANTS.md`. A trigger-only frontend workaround cannot
  provide the required publication or transaction lifetime.

## Nonblocking validated DDL, after DML authority is safe

Extend the existing native FK definition/publication path; do not create a
second constraint catalog, metadata table or frontend population scanner.
Possible option names are `validate_existing` and `enforced`, with a published
read-only validation state. Names and layout require architecture review.

1. Validate arity, column existence, parent candidate-key coverage, actions and
   comparison contract using native declarations. SQL assignment/type rules
   remain compile-time frontend decisions. Cross-database references and
   unsupported matching/deferred actions must fail explicitly.
2. Publish one provisional native enforcement edge at a DML batch-admission
   boundary, before scanning old data. It belongs to the same canonical native
   FK definition; it is not yet a successful public/durable ADD. Every later
   relevant mutation uses the edge, claims and existing generated programs.
3. Enrol pre-boundary in-flight mutations and rollback-restorable values.
   Existing transaction/lifetime bookkeeping must close the gap between
   entering a native write and observing the new edge. An epoch read without
   registration/recheck is insufficient. DDL may wait for conflicting older
   transaction outcomes or retry; readers and new unrelated writers must not
   wait for the validation scan. Cancellation must terminate the DDL owner.
4. Native validation scans captured visible extents using normal generation
   lifetime/visibility handling and FK probes; they use bounded batches and
   the existing parallel budget. Rebuild forwarding preserves these extents.
   Own provisional writes and old undo must be included or settled before a
   validation proof is accepted. No unbounded Scheme row/key materialization.
5. After all preceding mutation/undo obligations are settled and old rows pass,
   conditionally publish both endpoint descriptions and canonical triggers as
   one validated native declaration. Existing ordered schema snapshots handle
   durable publication. Schema ownership is released before persistence I/O;
   successful `safe` DDL cannot acknowledge before its required durable outcome.
6. Validation failure/cancellation removes only the provisional edge and its
   own runtime dependencies. No public validated FK remains. A concurrent
   DROP/rename/table retirement invalidates the candidate, rather than letting
   an old validator publish against replacement logical objects.

The registration boundary, metadata publication and durable-error handling are
still open implementation work, not proved by this sketch. Reusing
`contributionRevision` alone is insufficient: it neither retains reference
authority through parked transactions nor closes the final publication gap.
Already existing unchecked definitions must not become trusted merely on
upgrade. Any persisted validation flag needs a conservative legacy default.
Recovery loads native definitions before publishing databases and reconstructs
system programs; runtime claims/provisional validators are never persisted.

## Binding the four frontends

| Frontend | Existing integration | Required mapping |
| --- | --- | --- |
| `sql` | Named/unnamed CREATE FK tuples; ALTER calls `createforeignkey`; DROP calls native FK removal. | Normal ADD validates old rows. Keep simple NULL matching and supported restrict/cascade/set-null actions. Resolve the existing executing-session FK-check switch in the frontend and pass native policy explicitly; enabling it alone must not claim retrospective validation. |
| `psql` | Same CREATE tuple and ALTER builtin. | Normal ADD validates; eventual `NOT VALID` installs future-write enforcement without an old-row trust claim; `VALIDATE CONSTRAINT` requests native validation. `NO ACTION` can map to immediate restrict only while deferral is explicitly unsupported. Fix the unnamed CREATE grammar losing parsed update/delete actions: its current tuple omits both. `MATCH FULL`, deferred checks and SET DEFAULT require separate native semantic support or explicit rejection. |
| `tsql` | CREATE uses native tuples; named UNIQUE/PRIMARY declarations already use native keys. ALTER currently lacks ADD FK. | Normal ADD / `WITH CHECK ADD` validates. `WITH NOCHECK ADD` means enabled future-write enforcement with old rows unvalidated, whereas `NOCHECK CONSTRAINT` disables enforcement. Re-enabling without validation must not set trust; `WITH CHECK CHECK CONSTRAINT` validates and enables. Expose actual native validation/enforcement state through compatibility metadata. |
| `rdf` | SPARQL/import writes use ordinary native INSERT and mutation callbacks over RDF tables. | Native declared constraints must apply to these writes too. RDF graph membership and IRI references do not imply relational FKs; invent no SQL FK grammar or implicit constraints for RDF. An externally explicitly declared FK uses the same native engine path. No RDF datatype/term-identity redesign is required here. |

Metadata lists actual native UNIQUE/PRIMARY/FK declarations only. Physical
auto-indexes and ignored access hints do not become reported schema objects.
No frontend can select another parser/planner or bypass native visibility.

## Implementation and verification sequence

1. Review the new authority and transaction-lifetime design before core code.
2. Protect existing declared-FK DML first, independently of ALTER parsing.
3. Implement native validated admission and coherent/durable publication.
4. Bind three SQL grammars/options; verify RDF/native writes inherit enforcement.
5. Remove new noncritical flags only after exact focused expected behavior passes.

Deterministic native tests must schedule the real generated checks on both
sides: child-check/parent-delete/child-publication; opposite ordering; tentative
parent rollback; deleted-child rollback; own child-delete then parent-delete;
savepoint repeated references; CASCADE/SET NULL and failing user triggers;
final NEW mutation; concurrent CREATE/ADD/DROP; validation versus existing parked
writes; rebuild/repartition; commit failure/unknown durable outcome; restart;
NULL/composite/coercing equality and independent-key progress. Ordinary reads
must complete while an affected writer/validator is paused. Go race-detector
tests complement, not replace, these logical schedules and Native JIT results.

The small added SQL suite specifies successful checked admission, NULL
exemption, rejection of old orphans, future enforcement, and no residue after
failed ADD. Existing comprehensive T-SQL FK specifications and native
trigger/transaction suites remain in place. Before implementing a core change,
add deterministic lifecycle tests and compare fixed FK/non-FK DML workloads
against the same master fixture; no test/time-limit weakening is permitted.
