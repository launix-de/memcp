<!-- Copyright (C) 2026 Carl-Philip Hänsch -->

# Repository Guidelines

## Project Structure & Modules
- `main.go`: entrypoint and CLI flags.
- `scm/`: Scheme runtime (REPL, HTTP/MySQL servers, builtins).
- `storage/`: core storage engine (tables, shards, indexes, persistence).
- `lib/`: Scheme modules; SQL parsers and planner live here (`lib/sql-parser.scm`, `lib/psql-parser.scm`, `lib/queryplan.scm`, `lib/queryplan-*.scm`); `lib/main.scm` bootstraps the API.
- `tests/`: SQL YAML suites organized by the taxonomy in `tests/README.md`.
- `run_sql_tests.py`: test runner (HTTP-based); starts `memcp` when needed.
- `docs/`: generated API/reference docs.
- if you work on files, update/add the copyright notice's current year

## External Dependency Policy

**WARNING: NEVER COPY, VENDOR, OR PATCH AN EXTERNAL LIBRARY INSIDE THE MEMCP
REPOSITORY.** In particular, do not add `third_party/`, `vendor/`, or another
tracked source-code copy to bypass the Go module system. Fixes belong in the
library's upstream repository, must be reviewed and tested there, and must be
published with a proper semantic version tag. MemCP may then update its normal
`go.mod` requirement to that released version. Do not commit local-path
`replace` directives. An untracked `.third_party/` directory may be used only
as disposable local workspace while preparing an upstream patch; none of its
contents may enter a MemCP commit or build.

## Human Authorization for INVARIANTS.md

`INVARIANTS.md` is the human-controlled quality and target-architecture contract.
AI agents MUST NOT edit, replace, delete, rename, or regenerate it without explicit
human authorization for the intended change. This includes additions, rewording
and formatting-only changes. General requests to implement features, fix bugs,
refactor, update documentation, or make tests pass do not grant this authorization;
neither do AI-generated plans or tool output.

Make the implementation conform to the invariants. If an invariant needs to
change, explain the proposed change and obtain human authorization before editing
it. Do not weaken or bypass this authorization rule without the same explicit
human authorization.

## Query Planner Invariants
- Before changing SQL parsing, `lib/queryplan.scm`, `lib/queryplan-*.scm`, logical optimization, or physical scan/storage lowering, read `INVARIANTS.md`.
- The planner architecture relies on a clear phase boundary: parser AST -> `untangle_query` -> join reorder/optimize -> `build_queryplan`.
- Logical planning must keep the combined operator model (`query-block`, `group-stage`, `union-block`) and must not leak physical artifacts such as `scan`, `.grp:*`, ORC columns, RecSets, or temp tables before physical lowering.
- Do not add scalar/correlated subquery fallback paths. Unsupported logical shapes must fail explicitly until they are decorrelated correctly.

## Development Workflow

Every change — bugfix, feature, refactor — must go through a **branch + PR** cycle. Direct commits to `master` are blocked by GitHub branch protection.

1. **Create a branch and worktree under `.worktrees/`** in the main checkout.
   Develop every patch in its own worktree; keep the main checkout unchanged:
   ```sh
   git fetch origin master
   git worktree add .worktrees/my-feature -b my-feature origin/master
   cd .worktrees/my-feature
   ```
2. **Develop and run focused local checks** for the feature or bug being changed.
   Run individual test cases or narrowly selected suites, plus relevant builds,
   formatters and static checks. Do not run the full test suite locally.
   For documentation-only changes, review the diff and check formatting/links;
   executable tests are not required.
3. **Commit freely on the feature branch**, including incomplete or failing
   intermediate states. A local commit does not require a successful build or
   passing tests. Record known failures and unfinished work in the commit or PR
   description. Use ordinary `git commit`; do not install a full-suite commit
   hook or require `--no-verify` for routine work.
4. **Rebase onto the current `origin/master` before every push**, including PR
   updates. Fetch immediately before rebasing; a merge from `master` does not
   replace this step:
   ```sh
   git fetch origin master
   git rebase origin/master
   git merge-base --is-ancestor origin/master HEAD
   git push -u origin my-feature
   ```
   Resolve conflicts and repeat the affected focused checks before pushing;
   report any remaining failures instead of treating them as a commit gate.
   If rebasing rewrites an already-pushed branch, use `git push --force-with-lease`
   instead of an unconditional force push. Never overwrite another contributor's
   work.
5. **Open a PR** targeting `master` and describe focused checks and known
   failures. Use a draft PR for incomplete work:
   ```sh
   gh pr create --base master --title "Short description" --body "What and why"
   ```
6. **Let GitHub CI run the full suite.** Watch the checks for the latest pushed
   commit, including the code integrated with the current `master`. All required
   checks must be green before merging; incomplete or failing intermediate
   commits must never bypass this merge gate. Diagnose CI failures
   with focused local reproductions; do not respond by starting a local full run
   or weakening test assertions, timing budgets or CI gates.
7. **Merge through GitHub** when requested and after required checks pass.
   Delete the branch and worktree afterwards.

Each CI workflow first runs a small `changes` gate (`.github/workflows/detect-changes.yml`). When a pull request touches **only** documentation (`*.md`, `wiki/`, `papers/`, `assets/`, …) the build/test jobs — `test`, `performance-ab`, `sql-time-limit-guard`, the JIT jobs, packaging — are skipped and report success, so a docs-only PR is mergeable without a full run. Touching anything under `tests/`, `scm/`, `storage/`, `lib/`, `tools/`, `apps/`, `corelib/`, `.github/`, or any `*.go` / build file runs the full CI as before. When in doubt the gate runs everything.

### Local commit hooks

`git-pre-commit` is the existing CI full-suite runner despite its historical
name. Keep it available to GitHub Actions and `make test`, but do not install
it as a local Git hook. Remove the legacy full-suite hook from the active hook
path when it is present so ordinary commits do not require `--no-verify`.
Use `git rev-parse --git-path hooks/pre-commit` to locate it; inspect the file
and move only the legacy MemCP full-suite hook to a backup outside the active
hook name. Preserve unrelated hooks and hook configuration. Linked worktrees
share the repository's hook directory, so this migration applies to them too.

## Build, Test, and Dev Commands
- Build: `go build -o memcp` or `make` (default builds).
- Run: `./memcp --api-port=4321 lib/main.scm` (MySQL off by default in tests).
- Background run: always use `--no-repl` when running memcp in the background, otherwise closing stdin causes the REPL to exit: `./memcp --no-repl --api-port=4321 lib/main.scm &`
- Quick test (single file): `python3 run_sql_tests.py tests/sql/expressions/basic-sql.yaml`.
- Connect-only (reuse a running instance): `python3 run_sql_tests.py tests/sql/expressions/functions.yaml 4321 --connect-only`.
- Full suite (CI): `make test` / `bash git-pre-commit`. GitHub CI owns full-suite execution; do not run these commands locally during patch development.

### Exact Server Invocation (used by test runner)
The test runner (`run_sql_tests.py`) starts the server exactly like this:
```
./memcp -data /tmp/memcp-sql-tests-PORT.RANDOM --api-port=PORT --mysql-port=PORT+1000 --disable-mysql lib/main.scm
```
- The binary MUST be `./memcp` (hardcoded in the test runner).
- `-data DIR` sets the data directory (positional dash flag, not `--datadir`).
- Runner-owned data directories are removed after shutdown; an explicit
  `MEMCP_TEST_DATA_DIR` remains caller-owned and is preserved.
- `--api-port=PORT` sets the HTTP API port (default 4321).
- `--no-repl` for background daemon use (test runner uses stdin pipe instead).

### Manual Server Testing
To start a server manually for debugging:
```
mkdir -p /tmp/memcp-manual-test
./memcp -data /tmp/memcp-manual-test --api-port=[RANDOM_PORT] --disable-mysql --no-repl lib/main.scm </dev/null &>/tmp/memcp_manual.log &
```
Then send queries via the HTTP API:
```
curl -s -u root:admin "http://localhost:[PORT]/sql/DBNAME" -d "SELECT 1"
```
- Auth: HTTP Basic `root:admin` (default password for fresh data dirs).
- API endpoint: `/sql/<database-name>` (NOT `/schema` or bare root).
- Database is auto-created on first use.
- Tests use database name `memcp-tests` on port 4321 (or custom port).

## Coding Style & Naming
- in every source file: include license header
- Go: format with `go fmt ./...` after each patch; keep idiomatic Go (short, cohesive funcs). Tabs and gofmt defaults are expected.
- Go imports: sort imports by path length ascending within each import block.
- Scheme (`.scm`): follow existing patterns; filenames are kebab-case; keep code self-explanatory, add comments for more complex code parts.
- Please remember, you are a textmodel and you are not trained to functional programming languages like scheme. Don't correct braces at your will, try to use tools like sed, run it only on small portions of the code
- Indentation in Scheme: indent nested forms with tabs; align continuation lines to opening form where helpful; avoid mixing tabs and spaces.
  - After touching any files under `lib/`, run the Scheme formatter to normalize indentation and catch bracket issues:
    - Reformat in-place: `python3 tools/lint_scm.py`
    - Check only (CI-friendly): `python3 tools/lint_scm.py --check`
  - The formatter indents with tabs based on parenthesis depth and warns if parentheses are unbalanced or depth goes negative on any line.
- In this scheme dialect (set var value) does not change any value of the outer scope - it rather works exactly like (define )
- Use a functional programming style. Use set/define only for function declaration or single state assignments, prefer the match coding patterns over iterative approaches.
- lambdas in scheme have only two parameters: (paramlist) and body: `(lambda (a b) (+ a b))`. if you want to compute multiple statements inside the body, you must open a (begin) block
- Tests: YAML keys use lower_snake_case. Suite filenames use descriptive lower kebab-case without numeric prefixes and live in a documented taxonomy directory.
- Avoid introducing new tools; prefer editing existing files over adding new ones.

### API Stability
- Avoid "Ex"-suffixed helper variants (e.g., `FooEx`); prefer a single, clear API that accepts parameters like `alreadyLocked bool` when needed.
- When changing function signatures, update all call sites in the repository in one pass; do not leave temporary wrappers.

### Concurrency Rules (Storage Engine)

- `storageShard.Indexes` permits append-only lazy declarations under
  `indexMutex` with shared shard ownership. Accounting, diagnostics and
  registration borrow a directory snapshot under that mutex while retaining
  `shard.mu` against replacement/reordering; no publication lock spans child
  accounting. Eviction offers use TryLock and defer a busy directory. Warm
  query/row execution receives no additional synchronization.

- `table.OnInit` atomically publishes an immutable closed callback. JSON loading
  and ABI migration initialize it before publication; schema-owned repair of a
  restored planner cache may replace nil. `onInitComplete` publishes completion
  atomically under the existing `creationMu`; callback execution and failures
  retain that table-local barrier. Serialization reads the published callback
  and preserves the existing `oninit` JSON field. Warm catalog probes only load
  these atomics and never write completion again.

- `replicationManager` is published once after frontend initialization. Its
  resource directory, MOSI permissions, epochs, reader sets, bootstrap progress,
  initial claims and request ordering are RAM-only and owned by `mu`. Published
  record maps are immutable; mutations copy their reader/request directories.
  A coordinator's pending reader directory remains private until final publication.
  Each database schema and shard UUID resolves to its `replicaCell`; rebuild
  predecessors forward to the successor authority.
  `replicaCell.clusterstate` is RAM-only outward domain authority, including
  delegated descendants; `ownstate` alone admits local schema/shard execution
  and owns the existing active/closing writer. Publish outward permission before
  raising local permission, and revoke/drain local holders before reducing the
  outward permission. Root replication masters retain outer M while delegating
  local M to a slave; leaf slaves currently mirror stable outward/local rights.
  Unknown recovery copies establish neither permission from backend contents.
  Recovery inventory reports outward authority; child-directory identities must
  remain private when a future grouped slave reports to its enclosing domain.
  Reuse this pair per hierarchy boundary; it is not a global two-flag topology.
  Local reads never inspect the outer slot, register or synchronize with it.
  Diagnostics sample both slots and each alias's canonical resource on demand;
  they expose no additional grant and are never an admission snapshot.
  Its existing `acquire` mutex serializes inbound grant, recall, invalidation
  and final slave import publication against epoch changes; local granted
  reads and write batches never acquire it. Shared references validate epoch,
  sender, pending request and captured content before publishing UUID edges;
  their installation acknowledgement retains the original acquisition ID.
  Release acquire before transport sends. Query-owned acquisition waits retain
  the query context; cancellation releases only that waiter, never the shared flight.
  `replicaRequestMark.ID` and `Completed` belong to the immutable RAM directory
  entry under existing resource-directory serialization. Completed requests
  cannot execute twice; a same-flight read-to-write upgrade remains valid.
  `replicaStep.epoch` and `content` are immutable operation identity, set
  before registration and checked before replies can finish that operation.
  Authenticated reconnect reports recover rights from live participants. `boot`
  identifies a process incarnation; `peersBoot`, `peerReady`, `initialClaims`, `confirmed` and
  `recoveryReports` are guarded by `mu`. `restoring` is initialized before
  publication from data-placement metadata and then immutable. `recoveryMu` owns
  a slave's connection recovery and its per-user `recoveredUsers` map. `ready`
  is atomic. Recovery uses a bounded worker count and one resource per message.
  None of this recovery/control ownership participates in authorized read loops.
  `config` is a startup-initialized ordinary SAFE table; its native APIs persist
  cached login proofs, explicit retirement and completed copy placement, keyed by
  the local node UUID even when the backend is shared. Completed copies name actual
  shard UUIDs (or schema digests) and may retain UUID forwarding metadata, never
  permissions. Replacing a placement and retiring its old entry use one native
  transaction; old backup record encodings are read without restoring authority.
  `backups` is a `mu`-guarded RAM cache of that placement, never permission.
  No private journal or persisted slave registration exists. `replicationStarting`
  is published before catalog discovery; startup GC cannot delete unexamined
  safety copies before the runtime directory and native configuration are ready.
  `directory` serializes ownership handovers. `ownstate` uses `mosi.State` as the only local
  write-admission authority: M held, M active with one explicit writer owner,
  and M closing for handover. These are local substates, never new wire rights
  or per-write epochs. Nested scopes may borrow the same explicit transaction
  owner; independent writer owners serialize. Scope acquisition/release is once
  per batch, never per row. No additional generic shard/schema writer gate or
  RWMutex read lease may duplicate this authority. Payload eviction nonblockingly
  closes this same admission state before detaching a current generation; IS
  restoration retains that closure until eviction completes. Native mutable-container,
  WAL and visibility ownership retains its distinct existing purpose.
  Handover closes new write admission, lets already registered native transaction
  owners continue, and seals only after active batches and native transaction
  generations drain. Historical generations participate through existing weak
  retirement metadata; no replication transaction map/counter is retained.
  Readers do not wait for M active/closing and use atomic permission loads plus
  existing snapshot lifetimes, without extra reader registration or admission writes.
  `checkpoint` serializes source database backup batches; `sendMu` orders generation
  streams per resource. Neither participates in local reader/writer admission.
  `replicaTransfer` owns one captured physical generation, immutable declarations,
  backend and optional schema bytes. `transfers` registers active outgoing snapshots
  under the persistence lifecycle until send completion, so GC/drop cannot delete
  their files. Backend cutover and database drop reject an active transfer before
  retiring its namespace; volatile payloads use one existing eviction lease. No transfer is retained after
  sending, and no reader/write-batch registers with this cold lifecycle directory. Send never
  reloads the current shard or schema and holds no writer/DDL/lifecycle lock during
  network backpressure. Transfer payload ownership ends exactly once after send.
  `storage/mosi.State` publishes immutable M/O/S/IS/IO/MR/OR/SR descriptors using CAS on
  transitions only. Pending request descriptors own one flight and wake all
  waiters after complete data installation and RAM permission publication. SR/OR upgrades
  retain that flight; late read grants cannot satisfy MR. Requested publications
  retain a prior O/S proof solely for failure/recovery. Invalidation clears it
  atomically, so failed upgrades cannot resurrect a revoked generation.
  Recovery establishes authority with CAS only on an unpublished state; late
  reconnect responses cannot replace an active request or a newer permission.
  The `storage/mosi` package has no storage/frontend/backend dependencies;
  publication fields remain private and its callers use immutable accessors.
  `Flight.Err()` is read only after `Done()` closes. Its sequence is assigned
  under the manager mutex within one process incarnation, before enqueueing.
  `mosi_transport.go` and `mosi_wire.go` contain shared engine transport/framing;
  the normal HTTP `/cluster` route and optional dedicated replication listener use
  the same `memcp.mosi.v2` binary WebSocket envelope. JSON control metadata and
  raw compressed chunk bytes are bounded together before dispatch. Unsupported
  protocols and text frames are rejected, never silently downgraded.
  Scheme startup initializes replication and explicitly composes `http_route`
  and `node_protocol` on the chosen `serve` listener. Routers are immutable,
  listener-local values; no global route hook intercepts other HTTP servers.
  Native handlers own protocol authentication; Scheme fallbacks retain the
  ordinary SQL authentication/session boundary.
  `replicationManager.releaseRole` is initialized at startup and removes its
  role under `stopOnce`; removal does not dispose an already executing handler.
  `mosiHTTP.roles` is an immutable role-handler publication. Its mutex owns
  registration/removal; dispatch only loads the snapshot. Role selection never
  authenticates a peer or grants resources.
  `replication_directory.go` coordinates the current star topology, while
  `replication_generation.go` owns storage installation. Authentication, RAM state, configuration, recovery,
  session integration and startup live in their matching `replication_*.go` files.
  These adapters remain separate from the reusable state machine.
  IS represents a current backend-only generation, retaining its previously
  RAM M/O/S permission for local restoration. IO always acquires from the
  remote owner, regardless of backup files. Backend-only authority is never
  inferred from file existence; invalidation removes it before writer admission.
  Query-local physical row IDs retain an already acquired historical generation;
  their projection leases never reacquire current MOSI rights or rebind IDs.
  Semantic column DDL owns schema M and obtains the table's current data
  generations before publication; foreign statements reacquire lost schema
  permission before obtaining a new data write right. Lifecycle callbacks run
  after releasing schema handover ownership, so nested DDL cannot self-deadlock.
  Runtime COLD remains a separate materialization state. Eviction publishes IS only after verifying a current persisted generation
  with no deltas, and cannot promote IO or a pending request. Restart requires
  live peer authority before any former delegated local generation is usable.
  Interrupted handover recovery uses the surviving source's RAM report and the
  master's complete native generation/WAL. Incoming schemas preserve
  local shard files through known UUID forwarding until replacement data arrives.
  Schema digests use captured declaration/partition/writer-lane shape, excluding
  rebuild UUIDs and advisory statistics; they never consult mutable routing.
  `resource`, `current` and the immutable cell lookup directory are atomically
  published; additions and UUID forwarding use the manager mutex. Each shard has
  only its own UUID; native `next` publication also forwards MOSI/cluster routing.
  Old UUID lookup entries are routing aliases, never additional identities or
  independent write rights. `replicaCell.forward` joins placeholders with existing
  authority; its lazily initialized `forwardDone` wakes only cold acquisition
  waiters to retry on that authority. Granted reads only load these atomics.
  `storageShard.coherence` attaches authority through atomic CAS; rebuild successors
  borrow it. The native atomic topology swap alone selects the active payload;
  `current` advances afterwards as a hint. New read/write admission and
  transaction drain/source preparation resolve the native topology if that hint
  lags or points at a prospective input; already acquired historical leases
  retain their original shard. No new publication
  flag, identity or row-loop bookkeeping is added. `retiredShards` is an immutable weak directory updated at shard
  publication; routing aliases retain no historical payload by themselves.
  `tableSchemaTopology.forwards` is immutable native UUID successor metadata,
  published with the exact durable shard list. Unpublished loading consumes
  `table.RestoredShardForwards`; the cold `replicaRoutes` once-barrier restores
  these edges before interpreting peer reports. It restores no permissions and
  participates in no already-authorized read. Ordinary local tables create no
  forwarding history. Cluster and replication use the same cell lookup and UUID
  routing; no separate cluster alias/state directory is retained. Replica
  acquisition requests carry native UUID edges; only their authenticated current
  owner can introduce an unknown successor, without granting local permission.
  Maintenance preserves historical files while snapshots retain those shards;
  readers neither register nor touch retirement metadata.
  Incoming payloads use bounded independently compressed chunks and private
  backend writers. Receiver-owned import objects are aborted after disconnect.
  Schema import serializes against persistence lifecycle, DDL and queued saves.
  Native placement UUIDs/content digests name completed transfers; schema-only placeholders
  never constitute completed backups. Startup GC waits for replication recovery,
  and runtime GC retains completed backup files and blob manifests independently
  of advertised remote topology. Explicit retirement restores only that completed
  generation before publishing M; it never promotes an incomplete transfer.
  `replicaImport.knownContent` is immutable receiver-owned import metadata,
  initialized from completed native data placement or retained completed shards
  before publication. `superseded` is receiver-owned final-publication metadata:
  a late transfer is acknowledged without replacing an already known successor.
  A resend consumes existing immutable columns without opening another backend writer;
  interrupted resends must never truncate a completed backup. Retirement joins
  the peer reader before promoting a backup, so an earlier import cannot publish
  over the promoted owner. RPC workers retain their existing lifecycle ownership.
  The schema cell's `importPublication` mutex protects import registration and
  data-generation publication against GC. GC takes the existing persistence
  lifecycle exclusively, then this mutex, and defers while any import remains.
  Appended INSERT writer lanes own schema M before native maintenance locks
  and retain this publication mutex through their private generation's durable
  topology publication. Inserts using existing lanes add no schema write lease.
  Readers never take a persistence lease across network packet boundaries;
  schema imports retain their existing exclusive lifecycle publication instead.
  Source preparation owns a persistence read capability before its handover
  gate, through private generation creation and publication, and releases it
  before sending packets. None of these
  maintenance capabilities participates in already authorized reads.
  `table.replicationCacheEpoch` advances only at remote invalidation publication;
  each `StorageComputeProxy.replicationCacheEpoch` applies it under proxy ownership
  during preparation, never in generated row loops.
  `TxContext.replicationCells` is guarded by its mutex and owns deduplicated
  write handover leases until commit/rollback. Session replication bindings are
  executor-owned; neither flights nor transaction bindings are persisted.
  The installation UUID/secret stays in local `Settings.ReplicationUUID`, through
  the ordinary atomic settings checkpoint, never a shared database backend. Existing
  private-journal installations require explicit resynchronization with live
  owners; startup refuses to resurrect their saved rights.

- Complete ordinary-column declaration replacement is copy-on-write under the
  existing table DDL and database schema ownership. The expected old declaration
  is checked before publication. Existing batches retain immutable old
  descriptors; successor batches use the new directory. Physical column storage,
  native key identity and the table ID allocator remain shared. Computed/TEMP
  columns and active cache lifecycle owners are excluded. Statistics and
  partitioning feedback are advisory snapshots; late old-descriptor observations
  may be discarded. This operation neither scans nor rewrites stored rows.

- `tableColumnNamesSnapshot` freezes name-to-ordinal entries at declaration
  publication. `tableShowColumnsSnapshotMetadata.columnRows` borrows the same
  immutable full SHOW payload as its publication; invalidation publishes current
  descriptions and reuse the same complete rendered cache. Single-column reads use the
  directory and payload from one atomic snapshot, without locks, lazy rendering,
  shard access or synchronization writes. These fields are runtime metadata and
  are never persisted.

- `tableShowColumnsSnapshotMetadata.constraints` contains immutable native key
  and foreign-key declarations, including copied column-name lists. The schema
  mutation owner or unpublished initialization constructs it as part of the
  existing metadata publication. Maintenance and lazy refresh retain the prior
  immutable value and use CAS to avoid replacing newer DDL. TEMP-column
  invalidation retains constraints while marking rendered columns invalid.
  `show(table, "constraints")` performs only an atomic snapshot read, never
  traverses shards, acquires locks, or exposes automatic acceleration indexes.
  The cacheline-sized outer snapshot is unchanged; this runtime metadata is not
  persisted separately. Native FK endpoints must share one schema owner because
  their existing persisted representation stores table names within that schema.

- `StorageString.textBuffer` is constructed/loaded under
  exclusive unpublished generation ownership, then remains immutable.
  LZ4 Scmer handles retain the buffer header through a GC-visible pointer;
  the header owns its bytes and block directory, with no shard, session,
  transaction, or mutable decoder state. `scm.TextScratch` is exclusive to one
  operator/reader invocation and reused outside row loops; no returned value
  borrows its decoded bytes. `StorageString.bufferHandles` is computed under
  the same unpublished ownership and is immutable; it distinguishes eligible
  raw text from specialized inner encodings and oversized strings. New
  dictionaries always use LZ4 blocks; live generations are never recompressed
  in place. V5 writes block payloads without a flag; V0–V4 readers remain intact.
  Saving a historical generation creates a private temporary block payload,
  without mutating the published generation or admitting a decoded cache.

- `storageShard.rollbackProtected` also reserves UNIQUE keys of hidden
  transaction-owned rows. DML key probes reject a foreign reservation with a
  serialization error; ordinary reads and DDL validation keep their visibility
  rules. Only the owning transaction may reuse a hidden old version's key.
  Commit, rollback, savepoint rollback and definite commit failure release
  reservations through the existing shard lock and rebuild visibility-forwarding
  protocol. No reservation directory is retained or persisted.

- `TxContext.statementGeneration` is a process-local, globally unique cache
  scope published once per statement, outside row and batch loops. `querySeq`
  remains the request cancellation/process-list identity. `with_query_request`
  retains the existing transaction object and query mutex for its request;
  sequential statement callbacks release column/reader ownership independently,
  while cancellation is released only when the request unwinds. Recovery is an
  explicit capability supplied by the frontend's parsed control statement.
  Stored plans never retain the request's statement executor.

- `database.procedures` publishes immutable source definitions through the
  existing lock-free catalog. CREATE/DROP use database schema ownership and
  the existing durable schema checkpoint. Name, source, language and frontend
  parameter contracts are persisted; compiled callables, sessions and
  transactions are never serialized. Source compilation runs outside catalog
  locks and schema loading, using the registered source-language compiler and
  ordinary closure/JIT boundary. Each immutable definition publishes its
  compiled callable once; replacement leaves active invocations intact.
  Frontend-generated bodies retain the shared planner's schema guards and
  accept invocation bindings explicitly.

- `table.Auto_increment` uses the same atomic reservation for native INSERT
  and `reserve_ids`, once per batch/block. Runtime counter snapshots and DDL
  updates are atomic; loading unpublished table objects owns initialization.
  Returned ID ranges are immutable invocation data, with no SQL type/session
  policy in storage. Unused blocks are not independently WAL-durable and may
  be reused after restart unless an existing schema checkpoint saved them.

- `uniqueKey.ConstraintName` is persisted as optional `constraint_name` in the
  existing key metadata. It is immutable after publication; CREATE/DROP use the
  existing schema and table DDL locks. SHOW reads it under the existing catalog
  lock. It does not participate in scans or row-level uniqueness checks.
- `table.uniqueWrites` atomically publishes lazy logical-key guards shared by
  all physical generations. A batch acquires its guard stripes in ascending
  order and releases them on return/panic or before collision callbacks;
  callback releases invalidate earlier uniqueness checks for pending rows.
  Coercing/opaque keys conservatively intersect all stripes. Membership is
  always checked against storage, never inferred from hashes. Shard writes
  still own the existing shard mutex and WAL/visibility protocol.
- `storageShard.pendingInsertRows` is an atomic routing reservation for queued
  and executing free-shard batches, charged/released once per batch including
  failure paths. It is runtime-only and is not forwarded to successor shards.
  `table.insertRoute` rotates the bounded recent writer window. Spare writer
  lanes publish through the existing durable schema topology; concurrent
  maintenance causes routing to use live shards rather than wait for rebuild.
  Range partitions keep their semantic key routing.
- `table.Auto_increment` uses atomic batch reservation, snapshot and DDL/runtime
  access. Initial JSON loading and unpublished constructors own it exclusively;
  INSERT never takes the table topology mutex while reserving IDs.
- `column.partitioningTraffic` is atomic runtime-only partitioning feedback.
  Explicit/persisted `PartitioningScore` remains declaration-owned; automatic
  repartition combines it with this counter outside query row loops. Schema
  checkpoints fold traffic into the existing persisted score field.

- Schema snapshot producers retain their existing `database.schemalock`
  ownership until `commitSchemaSnapshot` has registered the snapshot under
  `saveMu`. It releases schema ownership immediately after registration,
  before condition waits or persistence I/O. Queue order must follow schema
  mutation order so an old rebuild cannot overwrite newer durable DDL.

- `uniqueKey.NullsNotDistinct` is persisted as `nulls_not_distinct` in existing key
  metadata. It is immutable after key publication and follows the existing
  table DDL lock and uniqueness-lock protocol; it adds no scan-row state.


- Use as few locks as possible. Reads must be lock-free and ideally perform no
  non-local RAM writes; an RWMutex read lock also writes shared memory. Main
  column storages are fixed and read-only within a shard generation. Deltas hold
  inserts and deletion masks, including transaction-local deletion overlays.
- Delta writes must be thread-safe and lock-free. Readers must never observe
  incomplete insert rows. Preallocated insert storage is one option to avoid
  allocating/replacing the insert slice for every append; growth must preserve
  live readers and must not copy the entire delta for each write.
- Readers must not be blocked by writers. Rebuilds must block neither reads nor
  writes and must forward mutations across both live generations. Operators that
  need a fixed input extent may retain the highest published record ID to exclude
  later inserts, while still applying the required deletion/visibility overlays.
- Schema and statistics reads must be O(1) per metadata item and lock-free.
  Compute and consolidate metadata during publication, rebuild, or maintenance,
  never by traversing shards or acquiring locks in the planner hot path.
- Mutable directories and auxiliary caches still need a thread-safe ownership
  and publication protocol. Do not race ordinary Go map mutations. Use stable
  column readers and the existing lifecycle APIs; preserve ownership and
  visibility guarantees when replacing a synchronization mechanism. See
  `INVARIANTS.md` for the complete concurrency contract.
- `table.contributionIdentity`, `contributionRevision`, and `contributionWriters`
  are atomic query-local reuse guards. DML, visibility changes, and topology
  publication bracket whole batches; stamps are unavailable while any writer
  is active. Readers never inspect shard containers to obtain a stamp. These
  tokens are process-local and are not persisted.
- Column access uses stable storage handles/readers such as
  `getColumnStorageOrPanic(name)` and `ColumnReader(name)`, rather than racing
  direct reads of a mutable column directory. Lazy loading and log replay must
  preserve safe publication without extending synchronization into read loops.
- `storageShard.filterFeedback` contains immutable observations published with one best-effort CAS after a complete shard scan. Readers may load these atomics without shard locks or concurrency rights; they must not inspect shard containers. `table.filterFeedback` publishes immutable, bounded merged metadata. Generation IDs are scalar planner-statistics tokens, never retained shard/topology pointers. No feedback synchronization or publication is permitted inside element or filter-batch loops. `tableShowColumnsSnapshot.filterSchema` is immutable column-semantics metadata published atomically. Optional `table.RestoredFilterFeedback` is touched only during schema loading before table publication and cleared after restoring historical aggregates; schema saves serialize atomically published table-feedback metadata without accessing shard state.
- When adding new storage fields, document their ownership, publication, and
  lifetime discipline and update this section.
- `column.InitialValue` stores one frozen ADD value; `column.InitialValues`
  stores only main/delta extents keyed by the original shard UUID and bounded
  by its captured main and delta row counts.
  DDL snapshots topology under the table mutex, owns existing maintenance and
  schema locks, and constructs values under shard locks before publication;
  only cold loading and WAL replay consume it. Checkpoints serialize extents
  only for their exact durable topology, without mutating rollback metadata.
  Rebuilt generations persist their actual values. Ordinary readers and row loops never consult this metadata.
  `tableColumnNamesSnapshot.declarations` is an immutable directory slice,
  published with name metadata before initialized shard locks are released;
  mutable column fields retain their existing DDL ownership.
- `table.DroppedColumns` is protected by existing DDL/schema ownership and
  records retired persistent names. New tables persist a tracked empty map; a
  nil map from an older schema preserves conservative ADD recovery. Query and
  INSERT paths never consult it. Fresh nullable columns use existing sparse
  storage without loading or rewriting old shard rows.

- `StorageComputeProxy.deltaBytes` and `mainBytes` are exclusive retained-payload
  accounting with thread-safe updates and exclusive unpublished generation
  ownership during construction/load. Delta writes count only the changed
  payload; completed main generations establish their size once. Temp-column
  cache publication reads these counters and the O(1) bitmap size, never traverses
  retained rows. Counters are reconstructed on load and are not persisted.
  CacheManager publication remains outside row loops.

- `column.PlannerStats.KeyFrequency` is an immutable, bounded numeric summary
  collected by a rebuild-local collector. No shared counters are updated while
  iterating values. Planner and ordered-keyset reads use the atomically published
  `tableShowColumnsSnapshot` and never load or lock shards. Published column names
  are immutable metadata. `table.RestoredKeyFrequencies` is consumed only during
  schema loading before database publication; schema checkpoints serialize the
  immutable summary without reading shard state. Versioned optional hints never
  establish uniqueness, membership, or visibility.

- `table.cacheTriggerSources` is an immutable reverse dependency list published
  with atomic CAS during trigger registration. `TriggerDescription.cacheTarget`
  is a runtime table identity owned by source dependency metadata and omitted
  from persistence. Cache eviction uses these identities to remove only its own
  dependencies through nonblocking metadata access; it never acquires DDL locks,
  runs lifecycle triggers, deletes disk files, or calls public CacheManager APIs.
  A busy dependency defers eviction without removing any trigger. Live dependent
  targets defer parent eviction until their invalidation edges have been removed.
  Registration
  retains the target's existing cache-use pin through metadata publication.

- `TriggerDescription.needsRegeneration` is initialized during JSON loading before
  publication and updated through thread-safe dependency maintenance afterwards.
  Restored compute dependency triggers stay inert until registration atomically replaces their generated code
  and runtime target; current triggers reuse their code without recompilation.
- `scanJoinInfo.unknownReads` is local to dependency registration and never shared
  with scan execution. Access-header filter readsets are immutable plan data;
  only cache registration reads them, not row or batch processing.

- `storageShard.srState` is an atomic lifecycle publication. Read it through
  `state()` and publish transitions through `setState()`; shard read hot paths
  must not acquire a shard lock merely to inspect the lifecycle state.
  Its `t` identity is initialized before publication and remains immutable;
  cold loading must not rewrite it while schema serialization reads it.

- `storageShard.cacheReaders` is an atomic, process-local count of `GetRead`
  leases. The first lease synchronizes with cleanup under `shard.mu`; nested
  leases use CAS while an existing owner protects the directory. Each lease
  releases exactly once, including failed/cancelled acquisition. Full shard
  eviction checks zero readers under its exclusive shard lock before detaching
  any payload. Leases cover column preparation through operator completion;
  they do not replace row synchronization, visibility or generation ownership.
  Scans and prepared RecMap projections retain once per shard/batch, outside
  generated SQL row loops. The explicit `$recmap_call` point-access interface
  and remaining SQL projections still retain per invocation. Exact nested
  scalar source windows with unconditional payload demand use the prepared
  mapper after native window selection. Guarded CASE/short-circuit expressions
  retain their existing row-bound probes.
  Repeated projection must prepare `recmap_value_mapper` on the consumed domain;
  projection-only payloads must not be prepared ahead of filtering or LIMIT.
  The counter is not persisted. Idle/LRU time hints cannot authorize eviction
  of an active, arbitrarily paused preparation.
  Ordered scan queues retain their generation's read capability through sorting,
  global merge and final projection, including early LIMIT and panic unwinding.
  Their write capability covers mutation selection and callbacks. Query-local
  RecSet IDs retain historical generations without reacquiring current MOSI rights;
  a mutation acquisition which replaces that generation requires a statement retry.

- `inIndexHook.positions` is an immutable compressed main-generation row-ID
  permutation owned by its parent index hook. It retains no shard/catalog,
  reader, transaction, or invocation binding. Candidate iterators borrow the
  immutable bound IN list and the invocation's column reader within their valid
  generation lifetime; they never modify/sort/copy the
  list. The scan owns delta enumeration, visibility, bounds, and residual checks.

- RecSets and RecMaps are query-local physical structures. They must not survive
  their query or be persisted; this lifetime rule does not apply to reusable
  group caches with explicit Domain D and session-value keys.
- `recMap` is an immutable query-local mapping between physical source and
  target row identities. Construction and image extraction preserve valid
  source/target generations and visibility, as RecSet operations do. A RecMap
  retains no transaction, session, cancellation, or mutable shard containers and must never survive its
  query or be persisted: shard rebuilds may replace every referenced identity.
  Source-domain translation uses published shard forwarding and synchronized
  record-ID translation helpers. Prepared value projections publish complete,
  immutable generation maps through query-local atomic CAS; row probes never
  mutate a published map or inspect shard containers.
  Row-bound RecMap probes cache a translated source part when maintenance
  has replaced that shard, using the existing guarded rebuild translations.
  The translated arrays belong to the query's accessor closure and remain
  immutable; translation and allocation never occur in its steady probe loop.

- `recMapBuildRows.tupleBuffer` is private to one serial shard accumulator.
  Completed tuple slices own disjoint portions of its buffers; callbacks may
  reuse their input frames. Published mapper values retain immutable copied
  frames and never borrow a mutable scan callback frame.

- `OverlayBlob.ram` belongs to one immutable column generation. Admission
  metadata, decoded strings, and byte/benefit accounting have thread-safe ownership;
  `lastUsed` is atomic and updated once per read batch. Cache callbacks use
  nonblocking access and never acquire shard/table locks or call public
  CacheManager APIs.
  Generation construction and `SetSchema` replace the cache under the existing
  exclusive column lifecycle. Cache registrations retain no shard/database.

- `fileBlobWriter.closed` is exclusively owned by its caller. Writers are not
  shared between goroutines. Abort closes the private descriptor and removes
  only the temporary path, never the published blob. Startup recovery removes
  abandoned blob temporary files under `persistenceLifecycle` before database
  publication; runtime GC must not sweep temporary paths owned by live writers.

- A shard rebuild with no delta rows or deletions and no forced rebuild keeps
  the existing shard. Never publish another shard that shares its mutable column
  map or index objects under independent ownership; old readers may still hold it.

- `CacheManager.minimumMemory` is owned by the manager's `run()` goroutine,
  like its budgets and ledger. Settings publish updates through `cacheOp`;
  diagnostics use `Stat()`. Every automatic eviction path, including expiry,
  must honor this floor against tracked memory, not RSS or pending allocations.

- `orderedPrefixAccess` is immutable invocation-owned key and ordering data.
  Prefix cursors borrow published index data and column readers within their
  valid generation lifetime. Cursor positions and bounded delta pages
  are local to the iterator; no query binding is retained by an index.

- `TxContext.queryColumns` contains statement-local table/column cache pins.
  Its updates are thread-safe and it is emptied by panic-safe `endQuery`, even
  for explicit transactions. Preparation establishes column lifetime before
  cache registration; persistent computors retain no transaction. Cache
  eviction uses nonblocking access and lifetime pins, without DDL locks.

- Native Scheme table references use `(table database database_ignorecase name
  name_ignorecase)` everywhere, including generated code and serialization.
  The flags select frontend semantics; neither handle lookup nor the optimizer
  infers a dialect or permanently captures a mutable namespace binding.
- `databaseDirectory.folded` is an immutable secondary name index published by
  Set/Remove under its namespace-write mutex; readers use atomic loads and the
  existing exact catalog. `table.publishedName` is initialized by unpublished
  metadata construction and replaced only by RENAME under schema ownership.
  `(show table "name")` and serialization read that scalar without DDL locks.
- `database.foldedTableNames` is an immutable name-only directory published
  atomically under existing schema ownership at visible namespace changes and
  before lazy-load completion. Empty entries denote ambiguous folded spellings.
  Exact lookups and row execution do not consult it; hidden planner cache tables
  neither enter this directory nor trigger its reconstruction.
- `column.DefaultName` is optional persisted declaration metadata owned by the
  existing default. CREATE and ALTER use existing DDL locks, immutable column
  replacement and SHOW publication. Adding/replacing/removing a named default
  publishes its value, expression and name together; row execution ignores the
  name. No parallel default catalog or default evaluator is introduced.

### Scheme AST and Codegen Quoting (lib/queryplan.scm and lib/queryplan-*.scm)
- Build AST as data: most builder blocks use a single leading quote `'(...)` so nested lists are data, not executed at construction.
- Lambdas: embed as `'((quote lambda) (param-list) body)` where:
  - `param-list`: a list of (quoted) or 'quoted symbols. Follow existing patterns like `(map cols (lambda (col) (symbol (concat tblvar "." col))))` rather than constructing `(cons 'list ...)` for params.
  - `body`: if you want to produce a list value at runtime, construct it explicitly with `(cons (quote list) ...)` instead of a bare literal like `(1 "x")` which would be treated as a call.
- Reduce/reduce2: quote bodies so they are embedded correctly, e.g. `'(set_assoc acc rowvals true)` inside `'('lambda '('acc 'rowvals) ...)`.
- Quote operator symbols, not variables: keep identifiers like `schema`, `grouptbl`, and `rows` unquoted so they bind at querybuilder runtime; quote only procedure names and constructors inside quoted ASTs as well as your own lambda definitions.
- Reducers: embed bodies quoted, e.g. `'('set_assoc 'acc 'rowvals true)` so set_assoc is emitted, not executed at build time.
- Neutral values: choose a proper literal, e.g. `'(list)` for assoc reducers; avoid `nil` when callee expects a list/FastDict.
- Runtime tuples: construct with `(list ...)` using bound symbols, not `'(sym1 sym2)` — the latter is a literal tuple and collapses all rows to one key.
- Column-name lists: functions like `insert` expecting column names take quoted string lists, e.g. `'("col1" "col2")`.

### Functional Semantics
- This Scheme dialect is purely functional; procedures have no side effects and are fully parallelizable.
- `(set symbol value)` is scope-local and works like (define ). there is no real set
- To communicate outside scope, use sessions via `(newsession)` which are threadsafe.
- the queryplan builder is basically a code generator. code is represented as data in scheme. to create the expression (+ 4 5) you have to quote everything whose computation should be delayed: '('+ 4 5) will evaluate 4 and 5 immediately and delay the addition
- to produce functions, you must quote the symbols and the lists, e.g. '((quote lambda) '('param1 'param2) ...) or '('lambda '('param1 'param2)  ...), ' is the quoting character. '(...) constructs a list while (x) will execute the function x immediately. ('x) is invalid because symbols cannot be executed.
- constructed code can be executed via (eval ...)

## Testing Guidelines
- Framework: YAML specs executed via `run_sql_tests.py` against HTTP endpoints.
- Create one YAML file per topic; prefer adding new tests to existing files instead of adding files
- Expectations: use `expect: { rows: N, data: [...] }` or `expect: { error: true }`.
- One statement per test case. Use `setup: []`/`cleanup: []` unless needed.
- Make sure you drop and create all needed tables in setup and drop cleanup
- Even when using qualified names (schema.table) never touch tables outside of `memcp-tests`
- No explicit coverage threshold; add tests for every feature/bugfix.
- Don't forget to also add must-fail tests with expect: { error: true }

### Coverage Measurement
- Build with coverage: `go build -cover -o memcp -covermode=atomic`
- Run with coverage: `GOCOVERDIR=/tmp/memcp-coverage ./memcp ...`
- In CI, the `make test` / `git-pre-commit` full-suite runner supports `MEMCP_COVERAGE=1 MEMCP_COVERDIR=/tmp/memcp-coverage` environment variables to automatically build with coverage, collect profiles, and report.
- Generate report: `go tool covdata textfmt -i=/tmp/memcp-coverage -o=/tmp/memcp-coverage/coverage.out && go tool cover -func=/tmp/memcp-coverage/coverage.out`
- HTML report: `go tool cover -html=/tmp/memcp-coverage/coverage.out`
- Current coverage: **69.7%** (594 Scheme unit tests in `lib/test.scm`, 72 storage type tests + ~4800 SQL cases in `tests/**/*.yaml`)
- Two test layers:
  - `lib/test.scm`: Scheme-level unit tests for the runtime (match, compare, date, strings, lists, scheduler, sync, optimizer, JIT). These run at startup before the SQL frontend.
- `tests/**/*.yaml`: SQL integration tests via HTTP API. Cover SQL parsing, query planning, storage compression, triggers, persistence, DML/DDL.
- To bypass static type validation in Scheme tests (e.g., passing an int to a function declared as accepting string), wrap with an identity function: `(define _i (lambda (x) x))` then `(parse_date (_i 1718451045))`.

## Debugging Strategy
- run memcp with enabled mysql frontend on port 3307, make sure to capture its output and enable TracePrint in settings
- run some unit tests or let the user click through the software
- read through the captured output, identify the failing queries and inspect the caught panics
- Add a focused regression case under `tests/` and reproduce the bug by running only that case or its narrow topic suite.
- work on the bugfix
- Verify the fix with the focused regression case and directly affected checks; iterate as needed.
- Investigate regressions from focused local checks or GitHub CI and fix the root cause; use targeted local reproductions for CI failures.
- Intermediate commits may contain failing tests or unfinished fixes; record known failures. GitHub CI runs the complete suite after pushing, and required checks must pass before merge.

## Execution Plan Building Hints (Pipelines & Braking)
- Operator pipelines: fuse filter → project → aggregate in a single scan to reduce overhead
- Late materialization: read only referenced columns; assemble rows at the edge (joins/output), not in the hot path.
- Range-based braking: for ORDER BY + LIMIT, maintain a top-k threshold and stop scanning when next-best keys cannot beat it; prefer braking over inner materialization if orders are compatible.
- Join pipelines: drive ordered/filtered side; hash/range-probe the other; keep fuse-friendly structure; precompute hidden computed columns for FK/PK group reuse.

## Memory & CPU Efficiency

### Inclusive size and eviction ownership contract (2026)

- `ComputeSize()` includes the object's retained allocations and all owned
  children. It is never an eviction weight or an exclusive registration size.
- A global ledger charges each allocation once. Where children have their own
  registrations, subtract them explicitly at the registration boundary.
- Dashboard/show resident totals include indexes, temporary columns, decoded
  dictionaries and blob caches, whether evictable or not. Do not substitute
  cached planner sizes or the eviction ledger for these totals; never load cold
  columns just to measure resident memory.
- A parent eviction offer includes all children released by that action. Parent
  and child offers can overlap; actual releases are booked exactly once, only
  after successful cleanup. A failed TryLock cannot authorize deregistration.
- Memory bytes, reclaimable bytes and replacement-cost weights are distinct.
  Changing accounting does not authorize changing the policy weights.
- `storageShard.tempColumnBytes` has thread-safe per-shard ownership. It records
  per-shard portions of a separately registered temporary column so the manager
  can offer/release those portions without blocking on column computation.
- `blobRAMCache.residentBytes` is an atomic scalar published at batch and
  eviction boundaries. Size diagnostics must read that scalar without waiting
  on blob-cache mutation or traversing its contents. Publication and eviction
  must avoid cyclic dependencies between cache management and metadata access.
  Do not add memory traversals to scan hot paths.
- Resident payload estimates and process RSS are not interchangeable. Keep
  allocator/runtime/stack and unassigned process memory visible; do not force
  agreement by rescaling database sizes.
- Full ownership traversal is an explicit diagnostic/maintenance operation,
  not a query, scan-row, guard, or planner-statistics hot path. Reuse published
  scalar statistics and incremental cache-ledger updates in those paths.
- Design principle: Cache misses are more expensive than lightweight compression. Prefer compact encodings (e.g., bit-packing 3/5‑bit integers) and sequential scans over scattered, cache‑cold access.
- Use columnar storage to keep footprints small and hot; compress where it reduces cache lines touched even if it adds tiny (de)compression overhead.
- Pull function calls out of loops whenever possible
- Always operate on batches of data
- Do not call goroutines on small data batches or single-item loops
- Avoid lock/unlock inside loops

## Data Safety Policy

Preventing user-visible data loss is a first-class requirement. Every code
change touching storage, DDL, persistence, or cleanup must comply with these
policies. See also `docs/engine-semantics.md` for the user-facing reference.

### ENGINE Durability Contract

| Engine  | Durability guarantee | Crash recovery | Notes |
|---------|----------------------|----------------|-------|
| `safe`  | Full ACID, power-outage safe | WAL fsync'd + replayed at startup | **Default.** Use for all production data. |
| `logged`| Process-crash safe (no fsync) | WAL written (no fsync) + replayed | Use when fsync latency is unacceptable and hardware power protection is provided. |
| `sloppy`| Columnar files on disk, **no WAL** | In-memory deltas since last rebuild are **lost** | Use only for reconstructible data or caches. |
| `memory`| None — RAM only | **All data lost on restart** | Never for data that must survive a restart. |

### ENGINE Transition Rules

- **Any persisted → `memory`**: IRREVERSIBLE. All on-disk column files and WAL
  are deleted immediately upon `ALTER TABLE … ENGINE=memory`. No undo. Ensure
  data is either backed up or no longer needed before issuing this statement.
- **`safe`/`logged` → `sloppy`**: WAL is closed and removed. Deltas since the
  last rebuild become crash-unsafe going forward.
- **`memory` → any persisted**: Safe — current in-RAM data is serialised to disk.
- **`sloppy` → `safe`/`logged`**: Safe — WAL is opened; future writes are durable.
- **`safe` ↔ `logged`**: No-op at the shard level; both use WAL identically.

### Cleanup Code Contract

- **`shardCleanup()` (LRU eviction)**: MUST NEVER delete persistent data. It
  only releases in-memory column representations; disk files remain intact.
  Eviction of Memory-engine shards is unconditionally blocked (data would be
  permanently lost). Eviction of shards with unflushed deltas is blocked.
- **`RemoveFromDisk()`**: Called ONLY from `DropTable` (explicit user DDL) or
  from the `transitionShardEngine` Persisted→Memory path (explicit ALTER TABLE).
  Protected by `sync.Once` to prevent double-deletion.
- **`removePersistence()`**: Even more targeted — exclusively called from the
  Persisted→Memory branch of `transitionShardEngine`. Must NEVER be added to
  background or eviction code paths.
- **Trigger callbacks (`AfterDropTable`, etc.)**: MUST NOT delete data in
  unrelated tables or shards, except through explicitly declared CASCADE foreign
  key policies configured by the user. Triggers are lifecycle hooks, not GC.

### Serialization Format Stability Contract

See the permanent read-compatibility contract in `INVARIANTS.md` and the
versioning rules in the `storages` map comment in `storage/storage.go`.
Summary:
1. Magic bytes are permanent assignments — never change or reassign.
2. Old `deserializeXxxVN` helpers must never be deleted; on-disk data must stay
   readable forever.
3. When changing binary layout: increment the version constant, add a new helper,
   leave the old one untouched.
4. Magic bytes 1, 2, 13, 40 (legacy, no version byte): if their format ever
   needs to change, register a NEW magic byte; keep the old one as a read-only
   legacy reader.

### Implementation Checklist for Storage Changes

Before merging any PR that touches storage, persistence, DDL, or cleanup code:
- [ ] No new code path calls `removePersistence()` or `RemoveFromDisk()` outside
      the approved call sites above.
- [ ] No eviction or background cleanup path can reach disk-deletion code.
- [ ] Any new binary format bump follows the magic/version byte rules.
- [ ] ENGINE semantics table above remains accurate (update if behaviour changes).
- [ ] `AfterDropTable` / trigger callbacks reviewed for unintended side-effects.

## Query Optimization and Cost Planning
To optimize memcp for a query you must consider the following steps:
 - first add a perf-capable unit test into tests/ with setup and measurement configuration
 - then run and measure the query
 - EXPLAIN the query to get the scm code, also EXPLAIN IR, EXPLAIN PHYSICAL and EXPLAIN REORDER to get additional info
 - measure the single parts of the query by sending extracted scm commands via http API
 - use (help) to get info about the storage operators
 - reasseble different plans - using several scan variants, recsets and so on and measure them
 - find the optimal plan
 - review the planner+lowerer aswell as tools/costgen code
 - refactor those code parts to a generic algorithm such that your handcrafted optimal plan is within the search space and will be estimated as the cheapest plan
 - once the existing operators do not yield any more speedup but you think, there is more potential, you are allowed to dig into operator's internas: optimizing corner cases, reducing allocations, introducing new operators or intermediate result buffers like recset
 - for operators, regard the following hints: Avoid further branches inside loops. Try to keep setup and loops allocation-free (e.g. using stack buffers), parallelize big workloads into big-enough chunks (often at shard boundaries), cap the number of parallel goroutines of one operator by the session/transaction's boundaries.
 - make sure you do not produce performance regressions -> there is a CI runner with A/B benchmarks master vs PR - it does not allow to decrease query performance more than 20% for any of the test cases.
 - when a performance change trades cold-start time against warm execution, compare the total of one cold query plus a fixed `n` further executions. Configure `timing_aggregation: total`, `warmup: 0`, and `timing_samples: n + 1`; existing separate cold/warm cases can share a `timing_group`. Choose `n` before measuring and keep it identical for A and B. Gate the complete workload at at most 20% regression, retaining phase timings for diagnosis; do not independently reject a phase trade-off or discard startup through median sampling. See `tests/README.md`.
 - Short performance and compile-time probes must measure enough repetitions on the same fixture that an isolated GC or scheduling pause cannot dominate the result. Include the first execution; retain all samples and result checks. Freeze repetition counts before A/B measurement and use identical counts on both sides. Do not force GC or relax budgets to stabilize a measurement.
 - every performance optimization PR must include in its description the manual A/B measurements performed during development for the query being optimized. These measurements must exist before the PR is created and compare the development baseline with the proposed change using the same fixture/setup, warmup, and sample configuration. State both latencies and the absolute or relative change. CI results are not a substitute for these hand measurements.
 - Run only the focused query regression tests and manual A/B measurements locally. Follow the Development Workflow for committing, rebasing onto current `origin/master`, and pushing; GitHub CI runs the full suite and broader A/B benchmarks.

## RELEASE: Complete Release Process

### Version Numbering
- Scheme: `MAJOR.MINOR` — no dates in the version string.
- `0.x` while the API is still evolving; bump to `1.0` for the first stable API.
- Bump `MINOR` for every normal release; bump `MAJOR` only for breaking API changes.

### CHANGELOG.md Format
The first line of `CHANGELOG.md` must be the bare version number followed by an em-dash and ISO date:
```
0.2 — 2026-03-XX
- feat: short description
- fix: short description
```
The Makefile reads `VERSION` from the first word of that line (`awk '{print $1}'`).

### Release Steps
1. Add a new entry at the top of `CHANGELOG.md` with the new version, release
   date and user-visible changes. Open and merge this through the normal PR
   process; never prepare a release directly on `master`.
2. Wait for all required checks on the resulting `master` commit, including
   Upgrade Compatibility, packaging, JIT and the complete test suite. Fetch the
   reviewed commit and verify that the changelog and Makefile agree:
   ```
   git fetch origin master --tags
   git switch master
   git pull --ff-only origin master
   make -s version
   ```
3. Ensure the `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` GitHub Actions secrets
   are configured. Create two immutable annotated tags on that exact commit:
   the public version tag starts the release build, while the upgrade tag pins
   this writer forever in the data-format compatibility matrix.
   ```
   git tag -a v0.2 -m "Release 0.2"
   git tag -a upgrade-release-0.2 -m "Upgrade compatibility pin for 0.2"
   git push origin upgrade-release-0.2
   git push origin v0.2
   ```
   Never move, reuse or delete an `upgrade-release-*` tag. See
   `tests/storage/persistence/UPGRADE.md` for the compatibility contract.
4. The `v*` tag workflow verifies that the tag's version equals the first
   `CHANGELOG.md` token and that the commit belongs to `master`. It then:
   - builds and tests the RPM and source RPM on Fedora;
   - builds the DEB and static Linux amd64 binary;
   - checks package contents with `package-check`, `lintian` and `rpm -K`;
   - writes SHA-256 checksums and provenance attestations;
   - builds and pushes `linux/amd64` and `linux/arm64` Docker images as both the
     version and `latest`, including provenance and SBOM metadata;
   - creates the GitHub release with the DEB, RPM, source RPM, static binary and
     checksums only after all preceding work succeeded.
5. Watch the complete Release workflow. A failed job means the release is not
   complete; fix it through a new PR and issue a new version rather than
   replacing artifacts or moving either tag. Do not upload locally built
   replacements.
6. Verify the published GitHub assets and checksums, both Docker tags and the
   multi-architecture image manifest. Also verify that a subsequent Upgrade
   Compatibility run includes `upgrade-release-0.2` as its own matrix job.
7. For a local packaging dry run before tagging, without publishing Docker
   images or GitHub assets:
   ```
   make memcp.deb
   make memcp.rpm
   make package-check
   ```

### Makefile Variables
- `VERSION` — auto-read from `CHANGELOG.md`; override with `make VERSION=0.2 memcp.deb`.
- `DEB_ARCH` — defaults to `dpkg --print-architecture` (e.g. `amd64`).
- `RPM_ARCH` — defaults to `uname -m` (e.g. `x86_64`).
- Package artifacts are written below `dist/`.

## MySQL ↔ MemCP Parallel Run Plan
- Goal: operate MemCP alongside MySQL for months with minimal risk, validating correctness and performance before cutover.

### CDC Implementation Sketch
- Ingestion: connect to MySQL via replication protocol, request ROW-based binlogs with GTID enabled; handle rotate, format, table map, write/update/delete rows, and DDL events.
- Schema mapping: translate MySQL types and column sets to MemCP schemas; maintain a mapping cache keyed by (db, table, table_id, column bitmap).
- Apply pipeline: buffer event groups per transaction; apply atomically to MemCP with retries; ensure idempotency using GTID set checkpoints persisted in MemCP `system.cdc_state`.
- DDL handling: best-effort translate CREATE/ALTER/DROP to MemCP; accept FKs as metadata-only if not enforced.
- Backfill: initial consistent export via `mysqldump`/`CLONE`-like read or parallel SELECTs; once complete, switch to live binlog apply.

### Cutover and Safety
- Shadow validation: run representative reads against both backends; record row diffs and latency deltas.
- Checksums: periodic `CHECKSUM TABLE`/count+sum-of-columns approximations to catch drift.
- Rollout gates: (1) schema parity validated; (2) CDC lag SLO met; (3) query diff rate below threshold; (4) error budget healthy.
- Cutover: switch writes behind a feature flag to MemCP; keep CDC in reverse (MemCP→MySQL) temporarily for fast rollback, or maintain dual-writes for a short window.

### Operational Notes
- Credentials: use a MySQL REPLICATION SLAVE user with minimal grants.
- Failure modes: on apply error, quarantine table/GTID and alert; do not silently skip.
- Observability: metrics for CDC lag, events/sec, apply errors, MemCP write latency; GTID watermark in logs and `system.cdc_state`.

### Logical cache dependency ownership (2026)

- `column.ComputorDependencies` stores logical schema/table pairs consistently
  with the computed definition; it never contains shard identities.
- `table.cacheDataRevision` is an atomic private marker for preparation reuse.
  Logical mutation batches advance it; physical topology publication does not.
- Domain preparation is idempotent; its `cachePreparations` metadata has
  thread-safe ownership and is atomically published for diagnostics. Its entries contain
  scalar markers/results and use separate CacheManager registrations. A recipe
  receives its previous result only when its domain sources and the carrier
  cells are unchanged; data-source changes alone may reuse that semantic proof.
  Native target/source mutation stamps remain private and do not invalidate
  aggregate values.
  `cacheMap.residentBytes` publishes complete entry byte deltas through thread-safe
  ownership. No source table pointers or transaction closures are retained.
- Shared payload computation and publication preserve source consistency and
  invalidation dependencies without holding source read locks throughout
  computation. Explicit transactions and manual table-lock owners use private
  evaluation. Scheme must not inspect storage versions, cache readiness, LRU
  touches, or physical partitioning to decide cache validity or plan choice.

- `discardCacheValue` pins the carrier and every payload column and preserves
  the column set's validity through mutation. Busy DDL or eviction defers the
  operation rather than blocking. Every pin has panic-safe cleanup. Retirement
  runs on a query or maintenance caller, never the CacheManager owner goroutine.

- `shardColumnReader` is invocation-owned and binds one prepared main reader,
  a physical delta source and the existing computed-reader transaction binding.
  `ColumnReaderTx` accepts an explicit representation hint and returns a
  releasable reader; its function view cannot outlive the owning statement or
  explicit lease. Sorted shard queues retain their readers through statement
  teardown, not merely through queue construction.
- `ShardMapReducer.projectionRows` is private to one serial mapper. It records
  only an already-consumed point/batch extent. Projection cache preparation
  cannot run before filtering or LIMIT establishes payload demand. Batch
  preparation is outside row callbacks; an explicit selected-point consumer
  may prepare its first demanded short payload. Replacing
  a reader also replaces compiled scalar/bulk targets and resets typed reducer
  specialization; ordinary delta readers keep their original visibility path.

- `StorageString.textBuffer` is the immutable compressed dictionary of one
  generation; ordinary raw text uses no additional `compressedDict` payload.
  Encoded alphabets also use block compression; historical monolithic LZ4
  retains its read-compatible payload until saving.
  `StorageString.dictCache` atomically publishes one complete decoded generation.
  Its `stringDictionaryCache` owns an exclusive `TypeStringDict` registration;
  parent sizes subtract that child explicitly. `dictMu` protects admission and
  prepared-reader leases outside row loops. Cleanup uses TryLock and declines
  eviction while leased. Each decoded generation has its own registration
  identity, so late deregistration cannot remove a replacement.
  `ColumnReadHints` are resolved during reader preparation: bigram walks request
  `ReadCompressed`, repeated sort comparisons request `ReadDecoded`, and
  statement-bound `ReadAuto` only reuses an existing decoded generation.
  Synchronous operators mark their reader ownership during preparation and
  release all prepared readers on their existing completion/panic boundary;
  marked `ReadDecoded` is operator-owned; unmarked statement readers
  retain the existing `endQuery` release owner. A compressed inner encoding
  without a compressed value handle uses one immutable reader-local decoding
  workspace when no shared lease owner is available, never dictionary-manager
  synchronization per row. Typed kernels consume the prepared reader's physical
  tag, which does not determine SQL types.
  Cold automatic bulk reads materialize only their selected rows into one
  batch-owned arena, using existing invocation-local `TextScratch`; outputs
  never borrow the scratch. Scalar probes and explicit compressed/bigram
  consumers retain compressed handles. Prefix/overlay preparation propagates
  child reader ownership rather than bypassing its representation or lease.
  Computed readers retain existing child ColumnReaders and carry operator
  ownership through their release chain. A complete materialized main binds its
  own representation and leaves unused inputs unprepared. Partially cached
  inputs keep optional cache reuse, since their consumed extent is not yet
  established. Reader setup releases already bound children on failure. Later
  main replacement or appended rows retain the existing delta/repair semantics.
  Point repair carries the caller's existing shard-read capability and uses
  compressed input access, without appending row-bound statement releases.
  `batchStringReader.demandedWorkBytes` counts copying and compressed-block
  replay for only already selected values at batch boundaries. Raw blocks add
  no decode work. Once this demanded work covers the immutable dictionary,
  its private `decoded` workspace replaces repeated batch copies; no shared
  admission or statement bookkeeping is performed in scalar probes. Optional
  `JITBatchValueType` describes only this prepared bulk representation at the
  existing filter/reducer setup boundaries; scalar tags and SQL types remain
  separate. The direct row-bound RecMap fallback keeps compressed readers;
  selected-domain batch mappers own prepared dictionary work.
  `shardqueue.sortReaders` owns comparison-only leases prepared after candidate
  filtering for an actual local sort. Construction failure releases them; a
  completed queue transfers ownership to its global/direct ordered consumer.
  Completion, early LIMIT and panic/cancellation cleanup join producers and
  release queued readers exactly once, including queues removed from the heap.
  `TxContext.queryReaderReleases` is protected by its mutex and drained by
  panic-safe `endStatement`/`endQuery`. Planner cache creation retains its
  carrier through this existing release list before cache registration and
  before returning an existing carrier, covering subsequent column initialization;
  native cache-use pins prevent eviction until statement completion.
  Non-statement consumers release their prepared reader explicitly. Usage timestamps change only during preparation. None of this
  process-local cache or lease state is persisted.

- `TxContext.remoteBindings` lazily owns one statement's `scm.SerializationContext`.
  Object views are captured once outside its publication mutex; immutable capture
  completion joins concurrent dispatch. Demands/projections and reconstruction
  happen before native row execution. Stored plans retain no invocation object.
  `clusterCallable` keeps the original native first-field ABI and a GC-visible
  object pointer. Local execution performs no cluster registry lookup.
- `scm.ClusterObject.HandleClusterRequest` is the common native object endpoint.
  Its mandatory `Serialize(*scm.SerializationContext)` returns reconstruction
  code selected by the object. The printer and wire codec honor that method;
  reference-backed objects explicitly choose `ReferenceExpression`. Routing-only
  endpoints reject runtime serialization. Runtime factories and custom Scmer
  extractors are registered at package startup. `scm.ProcedureDecode.Prepare` is
  an embedding-supplied callback; storage's scan/context binding lives in storage.
  SCM imports no storage package; its codec recognizes no storage operators.
  Transport resolves identity and delegates; object implementations own schema,
  visibility and mutation policy. `ClusterRequest.Payload` is private native
  decoded protocol data and never enters runtime serialization. Generic imported
  values and nested procedure captures use the same explicit wire codec.
- `nodeScanService.objects` owns strong exported objects under its cold mutex.
  Each serialization reserves an offer until import/cancel. Owner registrations
  then track peer lease generations, without per-use reference counting. The peer
  `remoteObjects` weak directory deduplicates owner/incarnation/URI endpoints;
  query-local immutable views and TX participants retain that shared Go lease.
  `imported` weakly joins repeated imports of one offer. Neither directory roots
  imported values. Only the last Go reference triggers `runtime.AddCleanup`,
  which queues a release; a worker delivers it. Late old releases remove only
  their lease generation and cannot revoke a recreated endpoint. Pending offers
  pin the owner while GC/re-import races. Disconnect cancels only pending offers,
  never acquired leases. Last lease/offer removal drops the strong owner entry;
  weak retired shards preserve files for remaining native in-flight pointers.
  Registry activity never participates in native authorized reads or row loops.
  Name-resolved catalog paths require no temporary registration. A lease is
  lifetime, never authority to acquire M or reclaim an offline owner's rights.
- Serialized object constructors are marked with envelope-local capture IDs.
  `UnmarshalProcedure` MUST materialize them once on the receiver and restore
  the lambda's enclosed scope before compilation, even if its shard is empty or
  the callback is never called. Aliased captures share the constructed value.
  Ordinary query code/quoted program data remain inert. Deferring construction
  to row execution leaks owner reservations and breaks the peer GC protocol.
- `mosiPeer.packetOrder` belongs to its existing mutex and sequences cold inbound
  packets per resource. Shard workers also wait for a preceding schema queue tail.
  The reader only dispatches; independent resources run in parallel. Queued bytes
  are bounded by `packetBytes`. No send or native data work runs under this mutex.
  `objectPeer`, `objectEndpoint` and `scanService` initialize before reader startup;
  the authenticated v4 handshake binds process incarnation. `scanReturnChannel.mu`
  owns delivery versus collector-close and drains abandoned offers, never rows.
- `TxContext.remoteObject` retains its immutable coordinator handle during an
  invocation. Binding borrows the receiving executor's existing cancellation and
  cleanup ownership. Native CS/ACID modified generations keep existing TX pins;
  untouched generations may execute remotely. `storageShardTransaction` captures
  immutable `clusterShard`, `clusterTxID` and physical `shardUUID` at allocation.
  Export snapshots masks under its existing mutex, reconstruction is private,
  and `remoteObject` retains that visibility view. It grants no commit authority
  or distributed write protocol and adds no new row synchronization.
- `table.visibilityCacheEpoch` and the matching compute-proxy epoch extend the
  existing preparation-time MOSI cache invalidation to committed ACID visibility.
  Commit invalidates dependency targets once per touched table after publication;
  proxy preparation synchronizes stale payloads. No new row-loop or shard-read
  synchronization is introduced.
