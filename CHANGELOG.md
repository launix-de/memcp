0.1 — 2026-03-09
=================

First public release of MemCP — a persistent main-memory database with MySQL and PostgreSQL wire compatibility.

### SQL Language

**DML**
- `SELECT` with `WHERE`, `GROUP BY`, `HAVING`, `ORDER BY`, `LIMIT`/`OFFSET`, `DISTINCT`
- `INSERT INTO` — single-row and multi-row, schema-qualified names
- `INSERT … ON DUPLICATE KEY UPDATE` (upsert)
- `UPDATE` with `WHERE`, including multi-table and JOIN-based updates
- `DELETE` with `WHERE` and schema-qualified names
- `TRUNCATE TABLE`

**Joins**
- `INNER JOIN`, `LEFT JOIN`, `RIGHT JOIN`, `CROSS JOIN`
- Subquery unnesting (Neumann-style): correlated `IN` subqueries with multiple tables rewritten to joins
- `UNION ALL`

**Subqueries**
- Scalar subqueries in expressions
- `IN (subquery)` and `NOT IN (subquery)`, multi-table variants
- `EXISTS` / `NOT EXISTS`
- Derived tables (subquery in `FROM`)
- Correlated subqueries

**Aggregation**
- `COUNT`, `SUM`, `AVG`, `MIN`, `MAX`
- Multi-column `GROUP BY` with `HAVING`
- `GROUP_CONCAT`
- **Computed columns (tempcol)** — expressions used repeatedly in queries (e.g. `YEAR(created_at)`, `price * quantity`) are materialised as a virtual column on first evaluation and cached in-shard; subsequent scans read the pre-computed value without re-evaluating the expression; lazily invalidated per-row on `UPDATE`/`DELETE` and incrementally updated where possible; evicted under memory pressure
- **Keytable / query cache** — `GROUP BY` results are cached in a temporary in-memory table (keytable) tied to the base table; triggers on the base table maintain the keytable incrementally on insert and delete, so repeated aggregation queries are answered from the cache without rescanning; keytables are memory-managed and evicted automatically when RAM is tight
- Incremental aggregate cache — repeated aggregation queries reuse pre-computed partial sums

**Window Functions**
- `ROW_NUMBER()`, `RANK()`, `DENSE_RANK()`
- `LEAD()`, `LAG()`
- `FIRST_VALUE()`, `LAST_VALUE()`
- `PARTITION BY` and `ORDER BY` inside `OVER()`

**Expressions and Operators**
- `CASE … WHEN … THEN … ELSE … END` (simple and searched)
- `COALESCE`, `NULLIF`, `IFNULL`
- `BETWEEN … AND …`, `NOT BETWEEN`
- `LIKE`, `NOT LIKE` (with `%` and `_` wildcards)
- `REGEXP` / `RLIKE`
- `IN (list)`, `NOT IN (list)`
- `IS NULL`, `IS NOT NULL`
- Arithmetic: `+`, `-`, `*`, `/`, `%`, `DIV`, `MOD`
- Bitwise: `&`, `|`, `^`, `~`, `<<`, `>>`
- Boolean: `AND`, `OR`, `NOT`

**String Functions**
`LENGTH`, `CHAR_LENGTH`, `UPPER`, `LOWER`, `SUBSTRING`, `SUBSTR`, `LEFT`, `RIGHT`,
`TRIM`, `LTRIM`, `RTRIM`, `LPAD`, `RPAD`, `REPLACE`, `CONCAT`, `CONCAT_WS`,
`LOCATE`, `INSTR`, `REPEAT`, `REVERSE`, `SPACE`, `STRCMP`, `SOUNDEX`, `MD5`

**Math Functions**
`ABS`, `ROUND`, `FLOOR`, `CEIL`, `CEILING`, `TRUNCATE`, `POWER`, `POW`, `SQRT`,
`LOG`, `LOG2`, `LOG10`, `EXP`, `SIGN`, `MOD`, `PI`, `RAND`, `GREATEST`, `LEAST`

**Date and Time Functions**
`NOW()`, `CURDATE()`, `CURTIME()`, `SYSDATE()`, `UTC_TIMESTAMP()`,
`DATE()`, `TIME()`, `YEAR()`, `MONTH()`, `DAY()`, `DAYOFWEEK()`, `DAYOFYEAR()`,
`HOUR()`, `MINUTE()`, `SECOND()`, `WEEK()`, `QUARTER()`,
`DATE_FORMAT()`, `DATE_ADD()`, `DATE_SUB()`, `DATEDIFF()`, `TIMESTAMPDIFF()`,
`STR_TO_DATE()`, `UNIX_TIMESTAMP()`, `FROM_UNIXTIME()`

**Type Conversion**
- `CAST(… AS type)`, `CONVERT(…, type)`, `CONVERT(… USING charset)`
- Implicit cross-type comparisons (int ↔ string ↔ float)

### DDL

- `CREATE DATABASE` / `DROP DATABASE`
- `CREATE TABLE` / `DROP TABLE` / `RENAME TABLE`
- `ALTER TABLE`: `ADD COLUMN`, `DROP COLUMN`, `MODIFY COLUMN`, `RENAME COLUMN`, `ADD INDEX`, `DROP INDEX`
- Column types: `INT`, `BIGINT`, `TINYINT`, `SMALLINT`, `MEDIUMINT`, `FLOAT`, `DOUBLE`, `DECIMAL`, `VARCHAR`, `TEXT`, `MEDIUMTEXT`, `LONGTEXT`, `BLOB`, `MEDIUMBLOB`, `LONGBLOB`, `BOOLEAN`, `DATE`, `DATETIME`, `TIMESTAMP`
- `DEFAULT` values including `CURRENT_TIMESTAMP`, `ON UPDATE CURRENT_TIMESTAMP`
- `AUTO_INCREMENT` / sequence columns
- `NOT NULL`, `UNIQUE`, `PRIMARY KEY` constraints
- `KEY` / `INDEX` definitions (used for index-accelerated scans)
- `FOREIGN KEY` definitions accepted (metadata-only, not enforced)
- `ENGINE=` accepted per-table (see Storage Engines below)
- `CHARACTER SET` / `COLLATE` accepted for MySQL compatibility

### Storage Engines

| Engine | Persistence | Memory-managed | Description |
|--------|-------------|----------------|-------------|
| `SAFE` | Disk (default) | Yes | Flushes page cache on every write — safe against power outages; data evicted from RAM when under memory pressure and reloaded on demand |
| `LOGGED` | Disk (WAL) | Yes | Log-structured writes without page-cache flushing — utilises full disk bandwidth; data survives crashes but not power loss |
| `SLOPPY` | Disk (async) | Yes | Asynchronous disk writes; faster ingestion, brief data loss on crash |
| `MEMORY` | RAM only | No | Never written to disk; never evicted; data lost on restart |
| `CACHE` | Schema only | Yes | RAM-only data, automatically cleared when MemCP approaches its memory limit; schema survives eviction |

### Storage and Compression

- **Columnar layout** — each column stored separately; queries read only the columns they use
- **Automatic column compression** — typical lossless compression ratio 3×–8×; compressed columns fit more data per cache line, enabling scan throughput that exceeds raw memory bandwidth; format chosen automatically per column from:
  - `StorageInt` — bit-packed integers; stores the value range as an offset and uses only as many bits per value as the range requires (e.g. 3 bits for values 0–7)
  - `StorageDecimal` — decimal values stored as scaled integers via `StorageInt`, preserving exact precision without floating-point error
  - `StorageFloat` — 64-bit IEEE 754 array for general floating-point columns
  - `StorageEnum` — k-ary rANS entropy coding for low-cardinality columns (up to 8 distinct values); encodes each symbol at its information-theoretic cost (e.g. a boolean that is 99% false uses ~0.08 bits/element instead of 1)
  - `StorageString` — dictionary-encoded strings; repeated values stored once; supports hex, base64, and case-folded variants to minimise dictionary size
  - `StoragePrefix` — shared-prefix compression for string columns with a common prefix (e.g. URLs, paths); prefix dictionary indexed by `StorageInt`
  - `StorageSparse` — stores only non-NULL/non-default values with a `StorageInt` record-ID index; ideal for mostly-NULL columns
  - `StorageSeq` — run-length / arithmetic-sequence encoding for auto-increment and timestamp columns; entire sequential runs stored as (start, stride, count) triples
- **Multi-shard architecture** — tables split across shards for parallel scan and concurrent write
- **Memory budget** — configurable RAM limit (default: 50% of system RAM); automatic LRU eviction with transparent reload
- **System-pressure awareness** — releases cache proactively when total free RAM drops below 10%
- **Automatic index building** — indexes are built and maintained automatically without administrator intervention; no `ANALYZE` or manual tuning required
- **Computed indexes** — indexes on expressions such as `YEAR(created_at)` or `LOWER(email)` are detected from query patterns and used transparently, e.g. `GROUP BY YEAR(x)` benefits from a precomputed index on `YEAR(x)`
- **Native indexes** — B-tree-style sorted indexes for range scans and equality lookups; index delta merging
- **Unique indexes** — enforced at write time across main and delta storage
- **Blob storage** — binary columns stored in separate slab files outside the columnar layout
- **CSV and JSON import** — `loadCSV` / `loadJSON` for bulk data loading
- **15-minute background compaction** — periodic shard rebuild to merge deltas and recompress

### Remote Storage Backends

- **S3 / MinIO** — store databases on Amazon S3 or any S3-compatible service; configure via JSON file in the data directory
- **Ceph/RADOS** — native RADOS object storage (build with `make ceph`; requires `librados-dev`)

### Protocol Compatibility

- **MySQL wire protocol** — TCP port (default 3307) and Unix domain socket (default `/run/memcp/memcp.sock`); compatible with any MySQL client or connector
- **PostgreSQL dialect** — `/psql/<database>` HTTP endpoint accepts PostgreSQL SQL syntax
- **HTTP REST API** — `/sql/<database>` accepts MySQL-dialect SQL; returns JSON; HTTP Basic authentication
- **SPARQL / RDF** — `/rdf/<database>` for SPARQL queries; `/rdf/<database>/load_ttl` for Turtle data import

### MySQL Client Compatibility

- `SHOW DATABASES`, `SHOW TABLES`, `SHOW COLUMNS`, `SHOW INDEX`, `SHOW CREATE TABLE`
- `SHOW VARIABLES`, `SET NAMES`, `SET SESSION`/`GLOBAL` — accepted without error
- `INFORMATION_SCHEMA` — `TABLES`, `COLUMNS`, `SCHEMATA`
- `CREATE USER … IDENTIFIED BY`, `ALTER USER`, `DROP USER`
- `GRANT … ON *.* TO`, `REVOKE`
- `PASSWORD()` function for MySQL-native password hashes
- Trailing semicolons in statements

### Triggers

- `BEFORE` / `AFTER` triggers on `INSERT`, `UPDATE`, `DELETE`
- Access to `NEW` and `OLD` row values
- `INSERT INTO … SELECT` triggers fire per row
- Async trigger execution for `AFTER` operations

### Transactions

- `BEGIN` / `COMMIT` / `ROLLBACK` — single-statement atomicity; multi-statement transactions accepted by the protocol

### Web Dashboard

- Live gauges: CPU usage, memory (total, memcp, per-database), active connections, requests/sec
- Database browser: list databases and tables, browse rows, view shard statistics and compression ratios
- Query console: execute SQL or Scheme expressions with syntax-highlighted results
- User management: create/delete users, manage database access grants
- Schema viewer: show columns, types, and indexes
- Settings editor: adjust memory budgets, shutdown drain time, and other runtime parameters
- Responsive, single-page interface over WebSocket

### Deployment

- **Docker** — `docker pull carli2/memcp`; single image with REST API and MySQL port exposed
- **Debian package** — `make memcp.deb`; installs to `/usr/bin/memcp`, `/usr/lib/memcp/`; systemd unit with interactive `dpkg-reconfigure` configuration wizard; `dpkg --purge` cleans data
- **RPM package** — `make memcp.rpm`; same layout; systemd scriptlets for enable/start/stop/purge
- **Singularity/Apptainer** — `make memcp.sif`
- **GitHub releases** — tagged `v*` pushes automatically build and publish `.deb`, `.rpm`, and binary via Actions
- **`make install`** — POSIX-compliant with `PREFIX` and `SYSTEMD_DIR` variables (default `/usr/local`)
- **`--config=FILE`** — load CLI arguments from a config file (one flag per line); `/etc/memcp/memcp.conf` used by the systemd unit

### Extensible HTTP Router

- Custom REST API endpoints can be registered directly in Scheme, hooking into the same HTTP server that serves the SQL and dashboard endpoints
- Handlers have full access to the storage engine — queries execute in-process with no network hop, enabling sub-millisecond microservice response times
- Endpoints are registered at startup via `lib/main.scm` or any imported module, making it straightforward to ship a self-contained application server alongside the database

### Scheme Runtime

- Embedded Scheme interpreter with JIT compilation (x86-64)
- Functional core: `quote`, `eval`, `if`, `and`, `or`, `match`, `define`/`set`, `lambda`, `begin`, `error`, `apply`
- Arithmetic, string, list, and dictionary builtins
- Sessions (`newsession`) for thread-safe mutable state
- `import`, `load`, `stream`, `watch`, `readfile` — all path-relative to the source file's directory
- `serve` (HTTP), `mysql` / `mysql_socket` (MySQL protocol) — extensible server hooks
- Schedulers, sync primitives, and cron-style periodic jobs
- REPL with readline support; `--no-repl` for daemon use


0.1.4
=====

- PASSWORD(str) for password hashes

0.1.3
=====

- Parsec parsers
- implement SELECT, UPDATE, DELETE with WHERE

0.1.2
=====

- added Dockerfile
- added function help
- storage function: scan_order

0.1.1
=====

- IO functions: password
- user table for mysql auth
- mysql and REST API check for username/password

0.1.0
=====

- basic scheme functions: quote, eval, if, and, or, match, define/set, lambda, begin, error, symbol, list
- arithmetic scheme functions: +, -, *, /, <=, <, >=, >, equal?, !/not
- scheme string functions: simplify, strlen, concat, toLower, toUpper, split
- scheme list functions: append, cons, car, cdr, merge, has?, filter, map, reduce
- scheme dictionary functions: filter_assoc, map_assoc, reduce_assoc, set_assoc, has_assoc?, merge_assoc
- IO functions: print, import, load, serve, mysql
- storage functions: scan, createdatabase, dropdatabase, createtable, droptable, insert, stat, rebuild, loadCSV, loadJSON
- storage types: SCMER, int, sequence, string, dictionary, float
- SQL: support for SELECT * FROM, CREATE DATABASE, CREATE TABLE, SHOW DATABASES, SHOW TABLES, INSERT INTO
