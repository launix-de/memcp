<!-- Copyright (C) 2026 Carl-Philip Haensch -->

# MemCP reliability drill

`reliability_drill.py` is an opt-in destructive test for process-crash recovery,
transactional writes, concurrent rebuilds, and offline data-directory restores.
It is intentionally separate from the regular CI suite: it starts and kills its
own MemCP processes and retains enough state to reproduce a failure.

Build MemCP, then run the complete short drill:

```sh
go build -o memcp .
python3 tools/reliability_drill.py --mode all
```

For a longer concurrent write/rebuild workload:

```sh
python3 tools/reliability_drill.py --mode all --workers 12 --operations 1000
```

Use `--rebuild-crashes N` to change the default five randomized
rebuild/kill/recovery rounds. Record `--seed` from the manifest to replay their
delays exactly.

To race a transaction-wide update across `ShardSize=100` shards with hard
process kills during `COMMIT`, run:

```sh
python3 tools/reliability_drill.py --mode atomicity --commit-crashes 100
```

Every recovered generation must contain either the complete state before the
transaction or the complete state after it. Mixed `x` values are a failed
crash-atomicity invariant. Increase `--atomicity-rows` and `--commit-crashes`
for hour-long runs; both the random seed and every selected delay are retained.

Every run creates a new `/tmp/memcp-reliability-*` directory containing the
source data directory, one log per server generation, a replay seed,
`manifest.json`, and (for restore runs) the stopped data-directory snapshot.
Pass `--artifacts NEW_PATH` to choose a different **new** artifact directory.
The command refuses an existing path and never connects to an existing server,
uses `pkill`, or deletes its artifacts.

The current drill covers:

- committed multi-shard insert/update/delete plus trigger effects across a
  hard process kill;
- a hard kill racing `COMMIT` of one table-wide `x=x+1` ACID update across
  hundreds of shards;
- complete rollback of an uncommitted ACID transaction after a hard kill;
- statement rollback when a trigger fails partway through a multi-row insert;
- a hard kill racing a rebuild and publication of its replacement shards;
- concurrent disjoint writers and rebuilds followed by another hard kill;
- graceful offline snapshot, restore into a separate data directory, and
  checksum comparison.

This exercises process-crash recovery. It does not claim to emulate lost or
reordered device writes after a power failure. That requires a controlled
virtual block device or virtual machine and filesystem-specific fault testing.
