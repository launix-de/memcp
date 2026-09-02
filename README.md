<!-- Copyright (C) 2023 - 2026 Carl-Philip Haensch; GPL-3.0-or-later -->

# MemCP — transactions, search, and analytics in one open-source database

MemCP is a persistent, column-oriented SQL database built for applications that
must update data continuously while also searching, filtering, and analyzing
large datasets. It brings an architecture best known from proprietary in-memory
ERP systems to a self-hosted GPL-licensed database.

MemCP keeps stable data in compact, read-optimized columns and accepts recent
changes in a write-friendly delta. Background rebuilds fold those changes back
into the compressed representation. Persistent data may be evicted from RAM and
loaded again on demand, so the complete database does not have to remain memory
resident.

Applications can connect through the MySQL wire protocol or submit MySQL- and
PostgreSQL-style SQL through separate HTTP endpoints. An RDF/SPARQL engine is
included as well.

> **Status: Beta.** MemCP is suitable for evaluation and controlled beta
> deployments. Common application SQL is covered by a large regression suite,
> while advanced dialect edge cases are still being completed. Validate the
> queries, durability configuration, backup procedure, and result parity of
> your application before a production cutover.

MemCP is licensed under GPL-3.0-or-later.

Read the [MemCP 0.9 beta release overview](https://launix.de/launix/memcp-beta-release-0-9-one-database-for-transactions-search-and-analytics/)
for the application story behind the current release.

## Why this project exists

During doctoral work at a university database chair, MemCP's initiator kept
encountering the same gap: open-source databases offered either conventional
transaction processing or columnar analytics, but no practical columnar
main/delta in-memory database for an ERP application that needed both. SAP HANA
demonstrated that the model could work at ERP scale, but the architecture
remained tied to a proprietary platform.

MemCP was started to make that database model available as ordinary free
software: run it on your own Linux server, connect an existing SQL application,
and use the same engine for transactions, full-text search, paged lists, and
analytics.

## Why MemCP?

- **Columnar execution:** scans and aggregates read only the columns they need.
- **Automatic indexing:** MemCP derives useful compound indexes from real
  access, filter, and ordering patterns.
- **Cost-based query planning:** the compiler rewrites correlated subqueries,
  reorders joins, and chooses a physical strategy for each part of a query.
- **Adaptive plans:** cached plans are retained while cardinalities remain in
  the same planning range and are reconsidered when relevant statistics cross
  a cost boundary.
- **Mixed workloads:** point access, writes, filtered lists, exact counts,
  full-text-like search, and complex analytics can use the same database.
- **Operational tooling:** packages, containers, imports, persistent storage,
  memory budgets, and an administration dashboard are included.

### MySQL client access

Existing MySQL clients and connectors can use MemCP's wire-protocol server.
SQL compatibility is broad but not yet a claim of complete MySQL equivalence;
test the exact query shapes used by your application.

```sql
CREATE TABLE users (id INT, name VARCHAR(100), email VARCHAR(255));
INSERT INTO users VALUES (1, 'Alice', 'alice@example.com');
SELECT * FROM users WHERE id = 1;
```

### SQL over HTTP

```bash
# Start MemCP with HTTP and MySQL interfaces
./memcp --api-port=4321 lib/main.scm

curl -u root:admin http://localhost:4321/sql/mydb \
  -d 'SELECT * FROM users'
```

Important HTTP endpoints:

- `/sql/<database>` — MySQL-dialect SQL
- `/psql/<database>` — PostgreSQL-dialect SQL
- `/rdf/<database>` — SPARQL queries
- `/rdf/<database>/load_ttl` — load RDF/Turtle data
- `/dashboard` — administration, system monitoring, query activity, storage,
  compression, and users

These are SQL-over-HTTP APIs rather than a resource-oriented REST data model.

## Architecture

The storage engine and protocol/runtime integration are implemented in Go. SQL
parsing, logical planning, optimization, and physical-plan generation are
implemented in Scheme.

The planner has an explicit phase boundary:

```text
SQL parser AST
    -> decorrelation and logical normalization
    -> join reorder and logical optimization
    -> cost-based physical lowering
    -> fused storage operators
```

Correlated subqueries are converted into reorderable joins before physical
planning rather than executed once per outer row. The physical planner can then
choose an execution strategy independently for each relevant part of the query.

The engine stores columns in compact representations, including bit-packed
integers, dictionaries, ranges, and sparse forms. Hot indexes and cached query
helpers are memory-managed. Persistent column data remains on disk when its
in-memory representation is evicted.

The optional JIT compiler is experimental. All benchmark ranges below were
measured without enabling the JIT.

## Performance

Performance depends on data distribution, query shape, cache state, durability
mode, and hardware. MemCP is not universally faster than another database.

Current application measurements include:

- **up to 30x faster than PostgreSQL** for selected real application searches
  and paged lists over the same dataset of roughly 800,000 documents. These
  queries combine full-text conditions, sorting, exact counts, pagination, and
  user-specific access checks;
- **from about 3% faster to 15x faster** on selected WordPress-oriented query
  workloads.

These ranges describe the measured workloads, not a blanket guarantee. Publish
or compare results together with the fixture, database configuration, indexes,
hardware, cold/warm state, repetitions, and individual query timings. MemCP's
repository includes explicit performance suites and CI compares every pull
request against its exact target revision.

## Docker

```bash
docker pull carli2/memcp
printf '%s\n' 'replace-with-a-long-random-password' > memcp-root-password.txt
chmod 600 memcp-root-password.txt
docker compose up -d
```

The container intentionally refuses to initialize a fresh data volume without
an explicit password. See [Container deployment](#container-deployment) for the
complete Compose configuration.

## Memory management

MemCP is designed to share a machine with other services while remaining within
an explicit RAM budget.

**Automatic compression** — MemCP selects compact representations per column.
Actual compression depends on the values and their distribution; benchmark it
with representative data rather than assuming a fixed compression ratio.

**Configurable memory budget** — by default MemCP's managed cache budget is 50%
of system RAM. Set an exact limit through the dashboard or Scheme API:

```bash
# Limit to 4 GB total
curl -u root:admin -X POST http://localhost:4321/scm \
  -d '(settings "MaxRamBytes" 4294967296)'

# Or as a percentage of total RAM (default: 50)
curl -u root:admin -X POST http://localhost:4321/scm \
  -d '(settings "MaxRamPercent" 40)'
```

**Automatic eviction** — when MemCP approaches its memory limit, it unloads
eligible, less-recently-used representations from RAM. Persistent data remains
on disk and is transparently loaded when a later query needs it.

**System-wide pressure awareness** — when the host runs low on available RAM,
MemCP can release cache entries even if its configured budget has not yet been
reached.

**Separate budget for persistent data** — a second managed budget (default: 30%
of system RAM) controls persisted shards and indexes independently of other
evictable query helpers. Both limits are tunable at runtime without restart.

### Storage engines

MemCP supports several storage engines, selectable per table via `CREATE TABLE ... ENGINE=<engine>` or `ALTER TABLE ... ENGINE=<engine>`:

| Engine | Durability contract | Evictable |
|--------|---------------------|-----------|
| `safe` | **Default.** WAL is fsynced at transaction/statement boundaries; committed writes survive process crashes and power loss. | Yes |
| `logged` | WAL is written without fsync. It survives a process crash or clean shutdown, but recent writes may be lost after power loss. | Yes |
| `sloppy` | Compressed column files are persisted, but there is no WAL. Deltas since the last rebuild are lost after an unclean shutdown. | Yes |
| `memory` | Rows exist only in RAM and are lost on restart. Schema and an optional reconstruction callback are persisted. | No |
| `cache` | Reconstructible RAM data; it may be cleared under memory pressure and rebuilt by its initializer. | Yes |

> **Data-safety warning:** `ALTER TABLE ... ENGINE=memory` from a persisted
> engine irreversibly deletes that table's on-disk column files and WAL. Back
> up or otherwise verify the data is disposable before making this transition.

For production data, use `safe` unless you have explicitly accepted another
engine's weaker durability contract.

### Test coverage

The repository currently contains more than 6,000 SQL/YAML cases across more
than 270 suites, in addition to Scheme runtime and Go storage tests. The SQL
taxonomy covers language semantics, planning, physical execution, concurrency,
storage, persistence, recovery, imports, RDF/SPARQL, anonymized integration
shapes, and performance regressions.

### Generated documentation

The generated API/reference documentation is not versioned. Build it locally when needed:

```bash
make docs
```

This writes the generated files to `docs/`.

## Quick start

```bash
# 1. Build MemCP
go mod download
make

# 2. Start the HTTP and MySQL interfaces
./memcp --api-port=4321 --mysql-port=3307 lib/main.scm

# Run as a background daemon (use --no-repl to avoid exiting when stdin closes)
./memcp --no-repl --api-port=4321 --mysql-port=3307 lib/main.scm &

# 3. Create your first database
curl -X POST http://localhost:4321/sql/system \
  -d "CREATE DATABASE myapp" \
  -u root:admin

# 4. Create a table
curl -X POST http://localhost:4321/sql/myapp \
  -d "CREATE TABLE products (id INT, name VARCHAR(100), price DECIMAL(10,2))" \
  -u root:admin

```

### Build with the experimental JIT

MemCP can compile Scheme expressions and query-planning code to native code. This
integration needs the `runtime/jit` support from the
[launix-de Go fork](https://github.com/launix-de/go/tree/jit-foreign-frames-go1.27.0):

```bash
# Build the patched Go toolchain next to the MemCP checkout.
git clone --branch jit-foreign-frames-go1.27.0 \
  https://github.com/launix-de/go.git ../go-jit
(cd ../go-jit/src && ./make.bash)

# Build MemCP with JIT support enabled.
PATH="$(cd ../go-jit/bin && pwd):$PATH" GOEXPERIMENT=jit go build -o memcp

# The experiment must also be enabled when running Go tests.
PATH="$(cd ../go-jit/bin && pwd):$PATH" GOEXPERIMENT=jit go test ./scm
```

The JIT is deliberately guarded by `GOEXPERIMENT=jit`. A normal build with an
official, unpatched Go compiler remains supported and uses the interpreter; it
does not compile or activate the experimental runtime integration.

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--api-port=PORT` | `4321` | HTTP API listen port |
| `--mysql-port=PORT` | `3307` | MySQL protocol listen port |
| `--mysql-socket=PATH` | `/tmp/memcp.sock` | MySQL Unix socket path |
| `--root-password=PASSWORD` | `admin` | Initial root password (first run only) |
| `--root-password-file=PATH` | — | Read the initial root password from a file |
| `--disable-api` | — | Disable HTTP API server |
| `--disable-mysql` | — | Disable MySQL protocol server |
| `--no-repl` | — | Disable interactive REPL (required for daemon/background use) |
| `--initialize` | — | Initialize a fresh data directory and exit cleanly |
| `-data DIR` | `./data` | Data directory |

### Authentication

> **Security note:** Never expose MemCP directly to the internet with default credentials. Always set a strong `--root-password` before any network-accessible deployment.

- Source builds default to `root` / `admin` unless `--root-password` is supplied
  for the first start of a fresh data directory.
- DEB and RPM installations generate a random initial password. Read it once
  with `sudo cat /etc/memcp/initial-root-password`, change it, and then delete
  that file.
- Containers require either the `memcp_root_password` secret shown below or an
  explicit `ROOT_PASSWORD` for a fresh volume. The secret is preferred because
  environment variables are visible through container metadata.
- Change the credentials with:
```bash
curl -X POST http://localhost:4321/sql/system \
  -d "ALTER USER root IDENTIFIED BY 'new-long-random-password'" \
  -u root:"$OLD_PASSWORD"
```

## Installation packages

Tagged releases publish a static Linux binary, an amd64 DEB, an x86_64 RPM and
source RPM, checksums, and a multi-architecture container image for amd64 and
arm64. The version in the tag must exactly match the first word in
`CHANGELOG.md` (for example `v0.2` for version `0.2`).

### Debian and Ubuntu

```bash
sudo apt install ./memcp_VERSION_amd64.deb
sudo cat /etc/memcp/initial-root-password
systemctl status memcp
```

The password file is readable only by root. After logging in and changing the
password, remove the one-time copy:

```bash
sudo rm /etc/memcp/initial-root-password
```

Configuration lives in `/etc/memcp/memcp.conf`, persistent data in
`/var/lib/memcp`, and the local MySQL socket in `/run/memcp/memcp.sock`.
Package upgrades stop MemCP gracefully before replacing the binary and restart
it afterwards. Removing or purging the package deliberately preserves database
data, the service account, and any still-existing initial password file.

The packaged systemd sandbox only permits writes below `/var/lib/memcp` and
`/run/memcp`. If `-data` is moved elsewhere, add that absolute directory with a
systemd drop-in (`ReadWritePaths=`); paths below `/home` additionally require an
appropriate `ProtectHome=` override.

### RPM distributions

```bash
sudo dnf install ./memcp_VERSION_x86_64.rpm
sudo cat /etc/memcp/initial-root-password
systemctl status memcp
```

RPM installations use the same paths and data-retention rules as DEB packages.

### Container deployment

Create `memcp-root-password.txt` outside the repository and keep it out of Git:

```yaml
services:
  memcp:
    image: carli2/memcp:VERSION
    ports:
      - "4321:4321"
      - "3307:3307"
    volumes:
      - memcp_data:/data
    secrets:
      - memcp_root_password

secrets:
  memcp_root_password:
    file: ./memcp-root-password.txt

volumes:
  memcp_data: {}
```

The image runs as the fixed unprivileged user/group `10001:10001`. Existing
host directories mounted at `/data` must therefore be writable by UID 10001.
The initial password is read only when `/data` is fresh; subsequent starts use
the credentials stored in the database.

For compatibility, `ROOT_PASSWORD` is also accepted on the first start:

```yaml
services:
  memcp:
    image: carli2/memcp:VERSION
    environment:
      ROOT_PASSWORD: replace-with-a-long-random-password
    ports:
      - "4321:4321"
      - "3307:3307"
    volumes:
      - memcp_data:/data
volumes:
  memcp_data: {}
```

### Building packages locally

```bash
make memcp.deb
make memcp.rpm
python3 tools/test_packaging.py --artifacts
```

Artifacts are written to `dist/`. Container builds are not part of these local
targets. `make docker-release` is an explicit multi-architecture push and
requires an authenticated Docker Buildx installation.

### Publishing a release

Repository maintainers configure `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` as
GitHub Actions secrets. After the release changelog PR is merged, create and
push an annotated tag whose name exactly matches `v` plus the changelog version.
The release workflow verifies that the commit belongs to `master`, rebuilds and
checks all artifacts, publishes the versioned and `latest` container tags,
creates SHA256 checksums and build-provenance attestations, and only then creates
the GitHub release. It must not be replaced by manually uploading locally built
files.

## Importing existing data

MemCP can bulk-import schema and data from MySQL or PostgreSQL with a single
Scheme call. The import **drops and recreates the selected target tables** on
every run. This makes an import reproducible after schema changes, but it is a
destructive replacement of existing target data. Use a separate target database
or take a verified backup when that data must be retained.

### Import from MySQL

```scheme
; import all databases (skip system dbs)
(mysql_import nil nil "root" "secret")

; import one specific database
(mysql_import nil nil "root" "secret" "myapp")

; import into a differently-named MemCP database
(mysql_import nil nil "root" "secret" "myapp" "myapp_memcp")
```

Parameters: `host` (nil → 127.0.0.1), `port` (nil → 3306), `username`, `password`,
`sourcedb` (nil → all), `targetdb` (nil → sourcedb), `sourcetable` (nil → all), `targettable` (nil → sourcetable).

### Import from PostgreSQL

PostgreSQL has an extra hierarchy level — **database → schema → table** — compared to MySQL.
The `sourceschema` parameter selects which schema(s) within the database to import.
All imported schemas land in the same MemCP database (`targetdb`).

```scheme
; import the public schema of one PostgreSQL database
(psql_import nil nil "postgres" "secret" "myapp" "public")

; import all non-system schemas of one database
(psql_import nil nil "postgres" "secret" "myapp" nil "myapp")

; import all databases (each becomes a separate MemCP database)
(psql_import nil nil "postgres" "secret")
```

Parameters: `host` (nil → 127.0.0.1), `port` (nil → 5432), `username`, `password`,
`sourcedb` (nil → all), `sourceschema` (nil → all non-system schemas), `targetdb` (nil → sourcedb),
`sourcetable` (nil → all), `targettable` (nil → sourcetable).

Both functions print a line for each imported table and return `true` on success.

## Use cases

MemCP is designed for applications that mix ordinary reads and writes with
large or structurally complex reads, for example:

- dashboards, reports, and grouped statistics;
- ordered and paged lists over large datasets;
- full-text-like searches combined with relational filters;
- multi-tenant or row-level ACL checks expressed through joins and correlated
  subqueries;
- notification and maintenance queries that repeatedly reuse the same grouped
  facts;
- RDF/SPARQL datasets alongside relational application data;
- development and compatibility testing through MySQL clients or SQL-over-HTTP.

## Contributing

Contributions should include focused regression coverage and preserve the
logical/physical planner boundary documented in `INVARIANTS.md`.

### CI behavior

- Pull requests to `master` run the full required `test` workflow.
- Direct pushes run `test` only on `master` (to avoid duplicate branch + PR runs).
- If a PR shows pending/old checks, push one fresh commit so the current workflow config is applied.

Useful contributions include SQL compatibility cases, correctness fixes,
measured performance work, documentation, storage formats, and new operators.

### Getting started
```bash
# 1. Fork the repository
# 2. Clone your fork
git clone https://github.com/launix-de/memcp.git

# 3. Set up development environment
cd memcp
go build -o memcp

# 4. Run the test suite (starts its own server automatically)
python3 run_sql_tests.py tests/sql/expressions/basic-sql.yaml

# 5. Make your changes and add tests
# 6. Submit a pull request!
```

## Testing

MemCP includes a comprehensive test framework:

```bash
# Run all tests
make test

# Or if you want to contribute, deploy this as a Pre-commit hook:
cp git-pre-commit .git/hooks/pre-commit

# Run specific test suites (starts its own server automatically)
python3 run_sql_tests.py tests/sql/expressions/basic-sql.yaml   # Basic operations
python3 run_sql_tests.py tests/sql/expressions/functions.yaml   # SQL functions
python3 run_sql_tests.py tests/sql/expressions/error-cases.yaml # Error handling

# Connect to an already-running instance (skip startup)
python3 run_sql_tests.py tests/sql/expressions/basic-sql.yaml 4321 --connect-only
```

## Performance testing

MemCP includes an auto-calibrating local performance framework and a required
CI A/B workflow. Pull requests measure the exact base revision and candidate
with the same fixtures, warm-up policy, repetitions, and runner. The default CI
regression allowance is 20% per protected case.

### Running performance tests

```bash
# Run perf tests (uses calibrated baselines)
PERF_TEST=1 make test

# Calibrate for your machine (run ~10 times to reach target time range)
PERF_TEST=1 PERF_CALIBRATE=1 make test

# Freeze row counts for bisecting performance regressions
PERF_TEST=1 PERF_NORECALIBRATE=1 make test

# Show query plans for each test
PERF_TEST=1 PERF_EXPLAIN=1 make test

# Run only the CI performance workload selection
python3 run_sql_tests.py --perf-ci
```

### How local calibration works

1. **Initial run** starts with 10,000 rows per test
2. Each calibration run **scales row counts by 30%** up/down
3. Target is **10-20 seconds** query time per test
4. Baselines are stored in `.perf_baseline.json`
5. After ~10 runs, row counts stabilize in the target range

The CI workflow additionally uses `PERF_AB_MODE=record` for the base revision
and `PERF_AB_MODE=compare` for the candidate. These modes are intended for the
automated base-versus-candidate protocol rather than ordinary development runs.

### Output format

```
✅ Perf: COUNT (7.9ms / 8700ms, 20,000 rows, 0.39µs/row, 11.4MB heap)
         │       │        │           │        │           └─ Heap memory after insert
         │       │        │           │        └─ Time per row
         │       │        │           └─ Calibrated row count
         │       │        └─ Threshold (from baseline × 1.1)
         │       └─ Actual query time
         └─ Test name
```

### Performance debugging cookbook

**Detecting a performance regression:**
```bash
# 1. Freeze baselines to use consistent row counts
PERF_TEST=1 PERF_NORECALIBRATE=1 make test

# 2. If a test fails threshold, you have a regression
```

**Bisecting a performance bug:**
```bash
# 1. Checkout the known-good commit, run calibration
git checkout good-commit
PERF_TEST=1 PERF_CALIBRATE=1 make test  # run 10x to calibrate

# 2. Save the baseline
cp .perf_baseline.json .perf_baseline_good.json

# 3. Bisect with frozen row counts
git bisect start
git bisect bad HEAD
git bisect good good-commit
git bisect run bash -c 'PERF_TEST=1 PERF_NORECALIBRATE=1 make test'
```

**Analyzing slow queries:**
```bash
# Show query plans to understand execution
PERF_TEST=1 PERF_EXPLAIN=1 make test
```

### Environment variables

| Variable | Values | Description |
|----------|--------|-------------|
| `PERF_TEST` | `0`/`1` | Enable performance tests |
| `PERF_CALIBRATE` | `0`/`1` | Update baselines with new times |
| `PERF_NORECALIBRATE` | `0`/`1` | Freeze row counts (for bisecting) |
| `PERF_EXPLAIN` | `0`/`1` | Show query plans |
| `PERF_REPEAT` | integer | Maximum repetitions for stable short-query samples |
| `PERF_MIN_MEASURE_MS` | milliseconds | Minimum accumulated sample duration |

## Remote storage backends

MemCP supports storing databases on remote storage backends instead of the local filesystem. To configure a remote backend, create a JSON configuration file in the data folder instead of a directory.

### S3 / MinIO Storage

Store your database on Amazon S3 or any S3-compatible storage (MinIO, Ceph RGW, etc.).

**Configuration file** (`data/mydb.json`):
```json
{
  "backend": "s3",
  "access_key_id": "your-access-key",
  "secret_access_key": "your-secret-key",
  "region": "us-east-1",
  "bucket": "memcp-data",
  "prefix": "databases"
}
```

**For MinIO or self-hosted S3-compatible storage:**
```json
{
  "backend": "s3",
  "access_key_id": "minioadmin",
  "secret_access_key": "minioadmin",
  "endpoint": "http://localhost:9000",
  "bucket": "memcp",
  "prefix": "data",
  "force_path_style": true
}
```

**Quick MinIO setup for testing:**
```bash
# Start MinIO with Docker
docker run -d --name minio \
  -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  minio/minio server /data --console-address ":9001"

# Create a bucket (via MinIO Console at http://localhost:9001)
# Or via mc CLI:
mc alias set local http://localhost:9000 minioadmin minioadmin
mc mb local/memcp
```

### Ceph/RADOS Storage

Store your database directly on Ceph RADOS for high-performance distributed storage.

**Why is Ceph optional?** The Ceph backend uses CGO to link against `librados` (the Ceph client library). This requires the C headers and library to be installed at compile time and the shared library at runtime. To keep the default build simple and portable, Ceph support is behind a build tag.

```bash
# Install Ceph development libraries (Ubuntu/Debian)
sudo apt-get install librados-dev

# Build MemCP with Ceph support
make ceph
# or: go build -tags=ceph
```

**Configuration file** (`data/mydb.json`):
```json
{
  "backend": "ceph",
  "username": "client.memcp",
  "cluster": "ceph",
  "pool": "memcp",
  "prefix": "databases"
}
```

**Optional fields:**
- `conf_file`: Path to ceph.conf (defaults to `/etc/ceph/ceph.conf`)

**Setting up a Ceph development cluster with vstart.sh:**
```bash
# Clone Ceph source
git clone https://github.com/ceph/ceph.git
cd ceph

# Install dependencies and build (only vstart target needed)
./install-deps.sh
pip install cython setuptools
./do_cmake.sh
cd build && ninja vstart

# Start a development cluster
cd ..
MON=1 OSD=3 MDS=0 MGR=1 ./build/bin/vstart.sh -d -n -x

# Create a pool for MemCP
./build/bin/ceph osd pool create memcp 32

# Create a user for MemCP (optional, can also use client.admin)
./build/bin/ceph auth get-or-create client.memcp \
  mon 'allow r' \
  osd 'allow rwx pool=memcp' \
  -o ceph.client.memcp.keyring
```

**Environment variables for vstart cluster:**
```bash
export CEPH_CONF=/path/to/ceph/build/ceph.conf
export CEPH_KEYRING=/path/to/ceph/build/keyring
```

### Backend Configuration Reference

| Field | Backend | Description |
|-------|---------|-------------|
| `backend` | all | Backend type: `"s3"` or `"ceph"` |
| `prefix` | all | Object key prefix for database objects |
| `access_key_id` | S3 | AWS or S3-compatible access key |
| `secret_access_key` | S3 | AWS or S3-compatible secret key |
| `region` | S3 | AWS region (e.g., `"us-east-1"`) |
| `endpoint` | S3 | Custom endpoint URL (for MinIO, etc.) |
| `bucket` | S3 | S3 bucket name |
| `force_path_style` | S3 | Use path-style URLs (required for MinIO) |
| `username` | Ceph | Ceph user (e.g., `"client.admin"`) |
| `cluster` | Ceph | Cluster name (usually `"ceph"`) |
| `conf_file` | Ceph | Path to ceph.conf (optional) |
| `pool` | Ceph | RADOS pool name |

## License

MemCP is free software licensed under GPL-3.0-or-later. See [LICENSE](LICENSE).

Questions and design discussions are welcome in [GitHub
Discussions](https://github.com/launix-de/memcp/discussions).
