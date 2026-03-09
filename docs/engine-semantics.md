<!--
Copyright (C) 2023-2026  Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
-->

# ENGINE Semantics and Durability Guarantees

Every table in MemCP has a **persistency mode** (also called the storage engine),
selected via `ENGINE=<name>` in `CREATE TABLE` or changed later via
`ALTER TABLE … ENGINE=<name>`.

## Quick Reference

| Engine   | Data survives restart? | Data survives unclean shutdown? | Use case |
|----------|------------------------|----------------------------------|----------|
| `safe`   | ✅ Yes                 | ✅ Yes — even on power outage (WAL fsync'd) | Production tables — default |
| `logged` | ✅ Yes                 | ✅ Yes on process crash; ⚠️ power outage may lose last WAL tail (no fsync) | When fsync latency matters and hardware power protection exists |
| `sloppy` | ✅ Yes (flushed data)  | ⚠️ Partial (unflushed deltas lost) | Caches, staging, reconstructible data |
| `memory` | ❌ No                  | ❌ No                            | Scratch tables, query intermediates |

## Detailed Descriptions

### `safe` (default)

Full ACID durability. Every committed write (INSERT, UPDATE, DELETE) is recorded
in a write-ahead log (WAL) and the log is fsync'd at transaction end. On the
next startup after an unclean shutdown, MemCP replays the WAL automatically and
restores all committed changes.

**Use `safe` for all production data that must survive crashes or restarts.**

The global default engine is `safe`. You can override the default with:
```sql
-- Scheme API
(settings "DefaultEngine" "sloppy")
```

### `logged`

Process-crash durability, without the overhead of fsync. The WAL is written
to disk, but MemCP does **not** call fsync at transaction end — the write
may be buffered in the OS page cache. This means:

- **Process crash or clean shutdown**: WAL is intact, replayed at startup, no
  data loss.
- **Sudden power loss or kernel panic**: the OS page cache may not have been
  flushed. The last few WAL entries since the most recent OS writeback could
  be lost.

Use `logged` when:
- Write throughput or latency is critical and the extra fsync of `safe` is a
  bottleneck.
- The hardware is protected against power loss (UPS, battery-backed RAID/NVMe,
  cloud persistent disk with power-loss protection).

### `sloppy`

Data is stored on disk as compressed columnar files, but **there is no
write-ahead log**. Writes accumulate in an in-memory delta store; the delta is
flushed to the main columnar storage during a `rebuild` operation (which runs
automatically when the delta grows large, or can be triggered manually via
`(rebuild)`).

- Data in the flushed (main) columnar storage: **durable across restarts**.
- Data in the in-memory delta (since the last rebuild): **lost on unclean shutdown**.

Use `sloppy` for:
- Cache or summary tables that can be reconstructed.
- High-write staging tables where some write loss is acceptable.
- Tables that are bulk-loaded and then only read.

**Do not use `sloppy` for data that must survive a crash without loss.**

### `memory`

All data is held in RAM only. Nothing is written to disk. All data is
**permanently lost** on any shutdown or restart — clean or unclean.

Memory-engine shards are never evicted from the cache (eviction would
cause permanent data loss for in-memory tables).

Use `memory` for:
- Temporary scratch tables within a session or computation.
- Query intermediates that are rebuilt from source on each startup.

## Changing the Engine (ALTER TABLE)

```sql
ALTER TABLE mytable ENGINE = safe;
ALTER TABLE mytable ENGINE = sloppy;
ALTER TABLE mytable ENGINE = memory;
```

### Transition Safety Matrix

| From \ To   | `safe`                             | `logged`                          | `sloppy`                                | `memory`                              |
|-------------|------------------------------------|------------------------------------|------------------------------------------|---------------------------------------|
| `safe`      | no-op                              | WAL kept, fsync disabled; power-outage risk going forward | WAL removed; future writes lose crash/power-outage safety | ⚠️ **IRREVERSIBLE** — disk deleted |
| `logged`    | fsync enabled; full power-outage safety | no-op                         | WAL removed; future writes lose crash safety | ⚠️ **IRREVERSIBLE** — disk deleted |
| `sloppy`    | WAL opened + fsync; fully durable  | WAL opened; crash-safe, not power-safe | no-op                             | ⚠️ **IRREVERSIBLE** — disk deleted |
| `memory`    | in-RAM data serialised to disk     | in-RAM data serialised to disk     | in-RAM data serialised to disk          | no-op                                 |

### ⚠️ WARNING: Persisted → `memory` is Irreversible

`ALTER TABLE … ENGINE=memory` on a table with engine `safe`, `logged`, or
`sloppy` **permanently and immediately deletes all on-disk column files and
WAL** for that table. There is no undo, no backup, no recycle bin.

Only issue this statement when you are certain the data is either:
- no longer needed, or
- already backed up externally.

## Cleanup Trigger Safety

`DROP TABLE` and `AfterDropTable` triggers delete only the data of the dropped
table itself. Cascade deletes in related tables happen **only** through
explicitly declared `FOREIGN KEY … ON DELETE CASCADE` constraints — not
through any automatic background cleanup.

LRU-eviction of shards (to free RAM under memory pressure) never deletes
persistent data; it only releases in-memory column representations. The
on-disk files remain intact and are lazily reloaded on the next access.

## Serialization Format Stability

MemCP uses a two-level versioning scheme for on-disk column files:

1. **Magic byte** (first byte of each column file): identifies the storage
   type. Magic byte assignments are permanent and never change.
2. **Version byte** (second byte for most types): identifies the layout
   version within that type. Old layout readers are never removed; existing
   data can always be read by any future MemCP version.

This means: a data directory written by an older MemCP version can always be
read by a newer version. There is no mandatory migration step.
