# Subquery Cache: Lazy Computed Columns with Valid-Mask

*Caching scalar subquery results (aggregates, lookups) with on-demand computation and trigger-based invalidation*

> **Note**: Code examples are conceptual drafts. Adjust to implementation challenges.

---

## Core Insight

**Reading a single cached value is ALWAYS faster than executing a subquery scan.** There's no decision to make about *whether* to cache - the answer is always yes.

The real question is: **For which rows should we pre-compute?**

Answer: **Only for rows that are actually accessed.** Use lazy evaluation with a valid-mask.

---

## Use Cases

| Use Case | Example | Cache Location |
|----------|---------|----------------|
| **Lookup Join** | `JOIN ticketState ON ID = ticket.state` | Column on `ticket` |
| **Scalar Subquery** | `(SELECT name FROM users WHERE id = t.user_id)` | Column on outer table |
| **Aggregation** | `SELECT SUM(salary) FROM emp GROUP BY dept` | Column on group table |
| **Filtered Aggregate** | `SELECT COUNT(*) FROM t WHERE subquery = x` | Column on group table, depends on lookup cache |

All use cases share the same infrastructure: **StorageComputeProxy with valid-mask**.

---

## Preparation

### Status: DONE — `make_keytable` extracted

The inline group table creation has been refactored into `make_keytable` (queryplan.scm:242-252):

```scheme
/* Current implementation (queryplan.scm:242): */
(define make_keytable (lambda (schema tbl keys tblvar condition_suffix) (begin
	(define keytable_name (if (nil? condition_suffix)
		(concat "." tbl ":" keys)
		(concat "." tbl ":" keys "|" condition_suffix)))
	(createtable schema keytable_name (cons
		'("unique" "group" (map keys (lambda (col) (concat col))))
		(map keys (lambda (col) '("column" (concat col) "any" '() '())))
	) '("engine" "sloppy") true)
	(partitiontable schema keytable_name (merge (map keys (lambda (col)
		(match col '('get_column (eval tblvar) false scol false)
			'((concat col) (shardcolumn schema tbl scol)) '())))))
	keytable_name
)))
```

**Differences from the original proposal:**
- Takes 5 params (added `condition_suffix` for dedup stages) vs proposed 4
- Returns just the keytable_name string — not `(list setup_code keytable_name col_mapping)`
- Executes createtable/partitiontable at build time as side effect (simpler, works)
- No FK→PK reuse yet (future optimization)
- No col_mapping return (column names are derived via `(concat key_expr)` convention)

**Called at:** queryplan.scm:1251: `(set grouptbl (make_keytable schema tbl stage_group tblvar (if is_dedup condition nil)))`

### Still TODO: FK→PK reuse

When `make_keytable` has a single key that matches a foreign key, it could reuse the referenced table instead of creating a temp table. This would require:
- Querying FK metadata from the schema at plan time
- Returning the referenced table name + its PK column name
- Adjusting the `make_col_replacer` to use the actual PK column name

### Remarks for keytable usage

sometimes, we want to collect ALL keys (regardless of filter functions) and only apply the filter to the computed column (especially if we reuse a base table instead of creating a new temptable)
sometimes, we collect only new keys
if you scan over the keytable, always filter the having condition, cuz even if you only inserted the keys from your having condition, the reuse of a canonical named temptable can also contain keys from other runs
the storage engine's scan/scan_order function may materialize filters or sub-expressions if it benefits e.g. an index creation. Example: SELECT .... HAVING COUNT(*) > 1 will materialize the SELECT COUNT(*) subquery into one tempcol of the keytable, the tempcol also has a canonical name, so if we also SELECT COUNT(*) we can read and filter one and the same column



---

## Architecture: StorageComputeProxy

### Core Design

Ein **Proxy** wraps underlying storage und handled lazy computation transparent (scan.go braucht nur minimale Änderung für `$invalidate:`):

```
┌─────────────────────────────────────────────────────────────────┐
│ storageShard                                                     │
├─────────────────────────────────────────────────────────────────┤
│ columns: map[string]ColumnStorage                                │
│                                                                  │
│   "state":                      StorageInt{...}                  │
│   ".state→ticketState:(final)": StorageComputeProxy{             │
│                                   main:       nil / StorageInt   │
│                                   delta:      StorageSparse      │
│                                   validMask:  [0,1,0,1,1,0,...]  │
│                                   computor:   (lambda ...)       │
│                                   inputCols:  ["state"]          │
│                                 }                                │
└─────────────────────────────────────────────────────────────────┘
```

### StorageComputeProxy Definition

```go
// storage/compute_proxy.go

// StorageComputeProxy wraps a storage and adds lazy computation with valid-mask.
// It implements ColumnStorage interface - transparent to scan.go and other consumers.
type StorageComputeProxy struct {
    main        ColumnStorage      // Compressed storage after rebuild (nil initially)
    delta       StorageSparse      // All computed values go here (always writable)
    validMask   NonLockingBitmap   // 1=valid, 0=needs compute (covers all rows)
    computor    scm.Scmer          // Computation lambda
    inputCols   []string           // Input column names for computor
    shard       *storageShard      // Reference to shard for input reads
    colName     string             // Own column name (cycle protection)
    lock        RWMutex            // lock when reading/writing delta, write lock when Compress
}

// GetValue implements ColumnStorage - the key method for lazy evaluation
func (p *StorageComputeProxy) GetValue(idx uint) scm.Scmer {
    // Fast path: check valid-mask
    if p.validMask.Get(idx) {
        // Delta always has priority
	RLock
        if val, ok := p.delta.TryGet(idx); ok {
	    release RLock
            return val
        }
	release RLock
        if p.main != nil {
            return p.main.GetValue(idx)
        }
    }

    // Slow path: compute value
    args := make([]scm.Scmer, len(p.inputCols))
    for i, col := range p.inputCols {
        args[i] = p.shard.columns[col].GetValue(idx)
    }

    val := scm.Apply(p.computor, args...)

    // Store result in delta (always writable)
    Lock
    p.delta.SetValue(idx, val)
    release Lock
    p.validMask.Set(idx)

    // Check if delta is too full → trigger Compress
    if p.Density() > 0.03 {
        go p.Compress()  // async to not block current read; but the next read will have to wait for the rebuild
    }

    return val
}

// Invalidate clears the valid bit for a row (O(1), no recompute)
func (p *StorageComputeProxy) Invalidate(idx uint) {
    p.validMask.Clear(idx)
    // Note: old value stays in main/delta until recomputed
}

// InvalidateAll clears all valid bits
func (p *StorageComputeProxy) InvalidateAll() {
    p.validMask.ClearAll()
}

// Density returns delta fill ratio for compression decision
func (p *StorageComputeProxy) Density() float64 {
    return float64(p.delta.Len()) / float64(p.shard.main_count)
}

// Compress computes ALL values and rebuilds optimized storage.
// Called when precompute=true or when sparse computation exceeds 3% threshold.
func (p *StorageComputeProxy) Compress() {
    writelock
    if other thread already compressed -> unlock+early out
    count := p.shard.main_count
    fn := scm.OptimizeProcToSerialFunction(p.computor)
    colvalues := make([]scm.Scmer, len(p.inputCols))

    // getValue: compute invalid values, read valid from delta/main
    getValue := func(idx uint) scm.Scmer {
        if p.validMask.Get(idx) {
            if val, ok := p.delta.TryGet(idx); ok {
                return val
            }
            if p.main != nil {
                return p.main.GetValue(idx)
            }
        }
        for j, col := range p.inputCols {
            colvalues[j] = p.shard.columns[col].GetValue(idx)
        }
        return fn(colvalues...)
    }

    // proposeCompression loop
    var newcol ColumnStorage = new(StorageSCMER)
    for {
        newcol.prepare()
        for idx := uint(0); idx < count; idx++ {
            newcol.scan(idx, getValue(idx))
        }
        if newcol2 := newcol.proposeCompression(count); newcol2 == nil {
            break
        } else {
            newcol = newcol2
        }
    }
    newcol.init(count)
    for idx := uint(0); idx < count; idx++ {
        newcol.build(idx, getValue(idx))
    }
    newcol.finish()

    p.main = newcol
    p.delta.Clear()
    p.validMask.SetAll()
    unlock
}
```

### Automatic Compression (Density-Based)

The proxy automatically compresses when density exceeds 3%:

- **StorageSparse**: ≥192 bit/value (index + value + overhead)
- **StorageInt** (compressed): ~6 bit/value (typical bitpacking)
- **Ratio**: 192/6 = 32x → **Threshold: >3% density**

```go
// After sparse computation, check density and compress if beneficial:
if proxy.Density() > 0.03 {
    proxy.Compress()  // Compute all remaining values, rebuild optimized storage
}
```

### Key Benefits

1. **Minimal scan.go changes** - Proxy implements `ColumnStorage`; nur `$invalidate:` Support nötig
2. **JIT simplification** - `StorageComputeProxy.JIT()` mit valid-mask check
3. **Unified Main + Delta** - Valid-mask covers all rows
4. **Idempotent createcolumn** - Mehrfach aufrufbar, garantiert filter-Zeilen berechnet

### Important: Index Compatibility

**Problem**: Indizes auf computed columns sind komplex, weil:
1. Index enthält nur berechnete Werte - invalide Rows fehlen
2. Bei Insert in Delta: computed column ist erstmal invalid → Index weiß nichts davon
3. Bei Invalidierung: Index hat noch den alten Wert

**Mögliche Lösungen:**

wenn ein Index gebraucht wird, um nach der berechneten Spalte zu filtern, MÜSSEN alle Daten da sein
-> Compress aufrufen (hat ja ein early out wenn die Daten noch valide sind)
-> Index neu bauen
ggf. Änderungen an den delta-Werten an den Index propagieren, um Index-Rebuilds zu vermeiden

---

## Valid-Mask: Unified for Main + Delta

The valid-mask covers **all rows** in the shard (main_count + delta):

```
┌─────────────────────────────────────────────────────────────────┐
│ Valid-Mask Layout                                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│ Index:    0   1   2   3   4   5   ...  999  1000 1001 1002      │
│           ├───────────────────────────────┤├────────────────┤   │
│           │      Main Storage             ││ Delta (inserts)│   │
│                                                                  │
│ Bits:     1   0   1   1   0   0   ...   1    0    0    0        │
│           ↑   ↑                             ↑                   │
│         valid invalid                   new inserts = invalid   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

- **New inserts**: Valid-mask grows, new bits are 0 (invalid)
- **After rebuild**: Valid-mask persists with main storage
- **Invalidation**: Just clear the bit, recompute on next read
- Proxys werden nicht rebuilt - sie müssen erhalten bleiben (Ausnahme mit if in rebuild bauen)

---

## Cache Creation

### Scheme Syntax

`createcolumn` has a fixed 6-parameter signature. All optional features are encoded in `options` (assoc-list):

```scheme
; Signature (always 6 parameters):
(createcolumn schema table colname type dimensions options)

; options is an assoc-list that can contain:
;   "computor_cols" '("col1" "col2")  - input columns for computor
;   "computor" (lambda ...)           - computation function
;   "filter_cols" '("col1")           - input columns for filter (optional)
;   "filter" (lambda ...)             - filter function (optional)
;   ... plus existing options (primary, unique, null, default, etc.)
```
filter_cols und filter options müssen nicht mit serialisiert werden
mehrere aufrufe von createcolumn müssen idempotent sein (d.h. spalte wird nicht neu angelegt, sondern wiederverwendet; es wird sichergestellt, dass mindestens alle in filter beschriebenen zeilen valid sind)

**Examples:**

```scheme
; Query: SELECT (SELECT final FROM ticketState WHERE ID=ticket.state) FROM ticket
; → Precompute ALL rows (no filter → compute all, optimized storage):
(createcolumn "mydb" "ticket"
    ".state→ticketState.ID:(final)"
    "any" '()
    '("computor_cols" '("state")
      "computor" (lambda (state)
          (scan "mydb" "ticketState"
              '("ID") (lambda (ID) (equal? ID state))
              '("final") (lambda (final) final)
              (lambda (a v) v) nil nil))))
danach kann einfach ein scan nach ".state→ticketState.ID:(final)" erfolgen

; Query: SELECT (SELECT final FROM ticketState WHERE ID=ticket.state) FROM ticket WHERE ticket.ID = 4
; → Sparse: only compute for ticket.ID=4 (rest computed on-demand):
(createcolumn "mydb" "ticket"
    ".state→ticketState.ID:(final)"
    "any" '()
    '("computor_cols" '("state")
      "computor" (lambda (state)
          (scan "mydb" "ticketState"
              '("ID") (lambda (ID) (equal? ID state))
              '("final") (lambda (final) final)
              (lambda (a v) v) nil nil))
      "filter_cols" '("ID")
      "filter" (lambda (ID) (equal? ID 4))))  ; ← only compute for ticket.ID=4
```
würde den delta erst mal nur mit der zeile, wo ID=4 ist mit einem validen wert füllen

### Mode Selection

| `filter` in options | Verhalten | Storage Type |
|---------------------|-----------|--------------|
| missing / `nil` | Alle Zeilen berechnen | Optimized (komprimiert) - direkt .Compress aufrufen |
| `(lambda () true)` | Alle Zeilen berechnen | Optimized (komprimiert) - direkt .Compress aufrufen |
| `(lambda (x) expr)` | Nur Zeilen wo filter=true berechnen | Sparse, oder komprimiert wenn >3% während der Schleife erreicht wird |

**Early-exit Optimierung**: Beim Durchlaufen mit Filter zählen wir die Treffer. Sobald >3% erreicht → abbrechen und `Compress()` aufrufen (berechnet alles und komprimiert).

### Storage-Side Implementation

```go
// storage/storage.go - createcolumn handler (unchanged signature, parse options)
func(a ...scm.Scmer) scm.Scmer {
    // ... existing parameter parsing ...
    typeparams := mustScmerSlice(a[5], "typeparams")
    ok := t.CreateColumn(colname, typename, dimensions, typeparams)

    // Extract computor options from typeparams assoc-list
    computorCols := getAssocList(typeparams, "computor_cols")
    computor := getAssoc(typeparams, "computor")
    filterCols := getAssocList(typeparams, "filter_cols")
    filter := getAssoc(typeparams, "filter")

    if !computor.IsNil() {
        t.ComputeColumn(colname,
            scmerSliceToStrings(computorCols), computor,
            scmerSliceToStrings(filterCols), filter)
    }

    return scm.NewBool(ok)
}
```

```go
// storage/compute.go - extended ComputeColumn signature

func (t *table) ComputeColumn(name string, inputCols []string, computor scm.Scmer,
                               filterCols []string, filter scm.Scmer) {
    // Validate: inputCols must not contain own name (cycle protection)
    for _, col := range inputCols {
        if col == name {
            panic("computed column cannot reference itself: " + name)
        }
    }

    // Determine mode:
    // - filter=nil → precompute all rows
    // - filter=(lambda () true) → precompute all rows
    // - filter=(lambda (x) expr) → sparse, only compute where filter=true
    precompute := filter.IsNil() || isProcWithBodyTrue(filter)

    // Find column definition
    var colDef *column
    for i := range t.Columns {
        if t.Columns[i].Name == name {
            colDef = &t.Columns[i]
            break
        }
    }
    if colDef == nil {
        panic("column " + t.Name + "." + name + " does not exist")
    }

    colDef.Computor = computor
    colDef.ComputorInputCols = inputCols

    // Initialize storage in all shards
    shardlist := t.Shards
    if shardlist == nil {
        shardlist = t.PShards
    }
    for _, s := range shardlist {
        s.ComputeColumn(name, inputCols, computor, filterCols, filter, precompute)
    }
}

// isProcWithBodyTrue checks if filter is (lambda () true)
func isProcWithBodyTrue(filter scm.Scmer) bool {
    if !filter.IsProc() {
        return false
    }
    // Check if body is literal true
    body := filter.ProcBody()
    return body.IsBool() && scm.ToBool(body)
}

func (s *storageShard) ComputeColumn(name string, inputCols []string, computor scm.Scmer,
                                             filterCols []string, filter scm.Scmer, precompute bool) {
    s.mu.Lock()
    defer s.mu.Unlock()

    totalRows := s.main_count + uint(len(s.inserts))

    // Check if proxy already exists
    proxy, exists := s.columns[name].(*StorageComputeProxy)

    if !exists {
        // Column doesn't exist → create new proxy
        proxy = &StorageComputeProxy{
            main:      nil,
            delta:     StorageSparse{},
            validMask: NewNonLockingBitmap(totalRows),  // all invalid initially
            computor:  computor,
            inputCols: inputCols,
            shard:     s,
            colName:   name,
        }
        s.columns[name] = proxy
    }

    // Now compute values based on filter
    if precompute {
        // No filter / filter=true → compute ALL and compress
        proxy.Compress()
    } else {
        // Filter given → compute only where filter=true AND currently invalid
        filterFn := scm.OptimizeProcToSerialFunction(filter)
        filtervals := make([]scm.Scmer, len(filterCols))
        threshold := totalRows * 3 / 100  // 3% threshold
        computed := uint(0)

        for idx := uint(0); idx < totalRows; idx++ {
            if proxy.validMask.Get(idx) {
                continue  // already valid, skip
            }

            // Evaluate filter
            for j, col := range filterCols {
                filtervals[j] = s.columns[col].GetValue(idx)
            }
            if scm.ToBool(filterFn(filtervals...)) {
                // Filter passed → compute via GetValue (stores in delta, sets valid)
                proxy.GetValue(idx)
                computed++

                // Early exit: >3% computed → switch to full compress
                if computed > threshold {
                    proxy.Compress()
                    return
                }
            }
        }
    }
}
```

---

## Invalidation Logic

We use the **existing generic trigger backend** (`storage/trigger.go`) for cache invalidation.

### Trigger System: Current State and Required Extensions

**Implemented** (in `storage/trigger.go`):
- TriggerTiming: BeforeInsert(0), AfterInsert(1), BeforeUpdate(2), AfterUpdate(3), BeforeDelete(4), AfterDelete(5)
- TriggerDescription: Name, Timing, Func (scm.Scmer/Proc), SourceSQL, IsSystem, Priority
- Trigger execution: ExecuteTriggers, ExecuteBeforeInsertTriggers, ExecuteBeforeUpdateTriggers, ExecuteBeforeDeleteTriggers
- SQL: CREATE TRIGGER / DROP TRIGGER
- Persistence: Triggers are serialized in schema.json via Scmer.MarshalJSON (Proc → JSON lambda form)
- IsSystem flag: system triggers are hidden from SHOW TRIGGERS

**Required extensions for subquery-cache**:

1. **`dropcolumn` trigger action**: The cache invalidation triggers call `(dropcolumn schema table col)`. This function must exist as a Scheme builtin that drops a computed column from a table, causing it to be recomputed lazily on next access.

2. **Trigger cleanup on DROP COLUMN**: When a computed column is dropped, all system triggers registered for its invalidation must also be removed. Naming convention `.cache:targetSchema.targetTable.colName|srcSchema.srcTable|TIMING` allows pattern-based cleanup.

3. **Trigger cleanup on DROP TABLE**: When a source table is dropped, all system triggers referencing it should be removed from other tables. Consider adding a `RemoveTriggersForTable(schema, table string)` helper.

4. **Error handling in triggers**: Trigger functions should use panic/recover internally. A failing AFTER trigger must NOT roll back the original operation (fire-and-forget for cache invalidation). BEFORE triggers that panic should abort the DML operation.

### What triggers invalidation?

1. **Changes to `computor_cols`**: NOT relevant - these columns don't change in-place. Updates use delete-mask + new insert, so the computed column for that row is gone anyway.

2. **Changes to tables scanned by `computor`**: The computor lambda may contain `(scan ...)` calls that read from other tables. Any change to those tables could affect computed values.

### Detecting scanned tables

When creating a computed column, analyze the `computor` AST to find all `(scan schema table ...)` calls. Register triggers on each of those tables.

```go
// Extract all tables scanned by computor
func ExtractScannedTables(computor scm.Scmer) []TableRef {
    var tables []TableRef
    // Walk AST, find (scan schema table ...) patterns
    // Return list of {schema, table} pairs
    return tables
}
```

### Phase 1: Full Invalidation (simple, correct)

For now, any change to a scanned table invalidates the **entire** computed column:

```go
func RegisterComputedColumnTriggers(targetSchema, targetTable, colName string, computor scm.Scmer) {
    scannedTables := ExtractScannedTables(computor)

    for _, src := range scannedTables {
        srcTbl := GetTable(src.Schema, src.Table)
        if srcTbl == nil {
            continue
        }

        triggerName := fmt.Sprintf(".cache:%s.%s.%s|%s.%s",
            targetSchema, targetTable, colName, src.Schema, src.Table)

        // Any change to source table → invalidate ALL rows of computed column
        for _, timing := range []TriggerTiming{AfterInsert, AfterUpdate, AfterDelete} {
            srcTbl.AddTrigger(TriggerDescription{
                Name:     triggerName + "|" + timing.String(),
                Timing:   timing,
                IsSystem: true,
                Func: scm.Compile(fmt.Sprintf(`
                    (lambda (OLD NEW)
                        (dropcolumn "%s" "%s" "%s"))`,
                    targetSchema, targetTable, colName)),
            })
        }
    }
}

```

### Phase 2: Selective Invalidation (future optimization)

**Sketch only - no implementation yet:**

For smarter invalidation, analyze the scan's filter condition to determine which rows are affected:

```
Scan structure:
(scan schema table filter_cols filter_lambda output_cols output_lambda reduce neutral braking)

Example computor:
(lambda (state)
    (scan "mydb" "ticketState"
        '("ID") (lambda (ID) (equal? ID state))    ; ← filter references 'state'
        '("final") (lambda (final) final)
        ...))
```

**Selective invalidation approach:**

1. Extract the scan's filter: `(equal? ID state)` where `state` is from computor input
2. On source table change, get changed key value (e.g., `OLD.ID` or `NEW.ID`)
3. Find rows in target table where `computor_col` matches changed key
4. Invalidate only those rows

```scheme
; Selective trigger (future):
(lambda (OLD NEW)
    (scan targetSchema targetTable
        '(targetKey "$invalidate:colName")
        (lambda (targetKey) (equal? targetKey (OLD "ID")))
        '("$invalidate:colName")
        (lambda ($inv) ($inv))
        ...))
```

**Challenges:**
- Filter may be complex (AND/OR/NOT) -> when we can't transform the code to build a trigger: full invalidation
- Multiple scans in one computor -> multiple triggers
- Join conditions vs. simple equality
- Not all filters allow selective invalidation
- drop column -> remove triggers (Triggers might have a list of foreign columns -> when column deletes, also drop trigger)

**For now**: Full invalidation is correct and simple. Optimize later based on real-world patterns.

### $invalidate:COLNAME Helper Column

For selective invalidation (Phase 2), we add `$invalidate:` as a virtual column prefix:

```go
// In scan's column reader setup:
if strings.HasPrefix(colName, "$invalidate:") {
    cacheColName := colName[12:]  // strip "$invalidate:"
    proxy, ok := shard.columns[cacheColName].(*StorageComputeProxy)
    if !ok {
        panic column does not exist
    }
    // Proxy is captured in closure - no map lookup per row!
    return func(idx uint) scm.Scmer {
        return scm.NewProc(func(args ...scm.Scmer) scm.Scmer {
            proxy.Invalidate(idx)
            return scm.NewBool(true)
        })
    }
}
```

---

## Cascading Invalidation

Wenn sich eine Source-Table ändert, können mehrere Computed Columns betroffen sein:

```
ticketState.UPDATE(ID=5)
    │
    ├──► ticket.".state→ticketState:(final)"  ── dropcolumn (scans ticketState) später .Invalidate(idx) -> neue Trigger-Art ON INVALIDATE
    │
    └──► other_table.".xyz→ticketState:(...)" ── dropcolumn (scans ticketState) später .Invalidate(idx)
```

Jede computed column registriert Trigger auf allen Tables, die ihr computor scannt. Bei Änderung → dropcolumn → nächster createcolumn baut neu.

---

## Aggregation Cache

Aggregation uses the same `StorageComputeProxy` on a **group table**:

```sql
SELECT dept, SUM(salary) FROM employees WHERE active GROUP BY dept
```

### Structure

```
Table: .employees:(dept)|true
┌─────────────────────────────────────────────────────────────────┐
│ columns:                                                         │
│   dept:                  StorageInt{1, 2, 3, 4, ...}            │
│   .(SUM salary)|true:  StorageComputeProxy{                   │
│                            main:       StorageFloat{...}        │
│                            delta:      StorageSparse{...}       │
│                            validMask:  [0, 0, 0, 0, ...]        │
│                            computor:   (scan + aggregate)       │
│                            inputCols:  ["dept"]                 │
│                          }                                      │
└─────────────────────────────────────────────────────────────────┘
```

### Incremental Updates (SUM/COUNT) - Future

**Phase 1**: Komplettes invalidate (dropcolumn) bei jeder Änderung an Source-Table.

**Phase 2**: Für additive Aggregates können Trigger inkrementell updaten statt invalidieren:

#### Which aggregates are incrementally updatable?

| Aggregate | Incremental? | Trigger Logic |
|-----------|-------------|---------------|
| **SUM(expr)** | Yes | `+= NEW.expr - OLD.expr` |
| **COUNT(*)** | Yes | `+= 1` (INSERT), `-= 1` (DELETE), no-op (UPDATE unless filter changes) |
| **COUNT(expr)** | Yes | adjust by ±1 based on NULL-ness of OLD.expr / NEW.expr |
| **MIN(expr)** | Partial | INSERT: `= min(cached, NEW.expr)`. DELETE: must rescan if deleted value == cached min |
| **MAX(expr)** | Partial | Same as MIN but mirrored |
| **AVG(expr)** | Yes | Store as (sum, count) pair internally, derive avg = sum/count |
| **GROUP_CONCAT** | No | Must rescan (order-dependent, separator-dependent) |

#### Trigger logic per DML type

**INSERT** (new row matches GROUP BY key `k`):
```
SUM:   proxy.delta[k] += NEW.expr
COUNT: proxy.delta[k] += 1
```

**DELETE** (old row had GROUP BY key `k`):
```
SUM:   proxy.delta[k] -= OLD.expr
COUNT: proxy.delta[k] -= 1
```

**UPDATE** (row changes from key `k_old` to `k_new`):
```
if k_old == k_new:
    SUM:   proxy.delta[k] += NEW.expr - OLD.expr
    COUNT: no-op (unless WHERE filter affected)
else:
    # treat as DELETE from k_old + INSERT into k_new
    proxy.delta[k_old] -= OLD.expr  (or -= 1 for COUNT)
    proxy.delta[k_new] += NEW.expr  (or += 1 for COUNT)
```

#### Corner cases

1. **WHERE filter on the aggregate query**: If the original query had `WHERE active = true`, the trigger must re-evaluate the filter for both OLD and NEW rows. Only adjust the aggregate if the row passes the filter.

2. **GROUP BY key changes**: UPDATE can move a row between groups. Must adjust both the old and new group's aggregate. If the new group doesn't exist in the keytable yet, insert it (or fall back to full rescan).

3. **New group keys from INSERT**: If INSERT creates a row with a GROUP BY key not yet in the keytable, the keytable needs a new row. This is cheap (sloppy engine INSERT) but must also initialize the aggregate column for that key.

4. **Group becomes empty after DELETE**: The keytable row stays (with aggregate=0 or NULL). HAVING filters handle this at query time. The keytable is never pruned by triggers.

5. **NULL handling**: `SUM(NULL)` is NULL, `COUNT(NULL)` doesn't count. Incremental triggers must check for NULL in both OLD and NEW values.

6. **Overflow**: Integer SUM can overflow. Same risk as full rescan — not a new issue.

#### Implementation sketch

```go
// For SUM/COUNT, the trigger computes a delta and applies it directly:
func (p *StorageComputeProxy) IncrementalUpdate(idx uint, delta scm.Scmer) {
    p.lock.Lock()
    if p.validMask.Get(idx) {
        oldVal := p.GetValueLocked(idx)
        newVal := scm.Add(oldVal, delta)
        p.delta.SetValue(idx, newVal)
    }
    // if not valid, no need to update — will be computed fresh on next read
    p.lock.Unlock()
}
```

#### When to fall back to full invalidation

- Aggregate is not incrementally updatable (MIN/MAX after delete of extreme, GROUP_CONCAT)
- WHERE filter is too complex to evaluate in the trigger
- GROUP BY key is an expression (not a simple column) — harder to map OLD/NEW to group
- Multiple tables in computor (e.g., filtered aggregate with subquery in WHERE)


---

## Prejoin Tables in GROUP BY → Subquery Cache

### Current Prejoin Flow (queryplan.scm:1353-1441)

Multi-table GROUP BY creates a `.prejoin:` temp table via nested-loop materialization, then recurses into `build_queryplan` with the prejoin as a single table:

```
SELECT dept.name, SUM(emp.salary) FROM emp JOIN dept ON dept.id = emp.dept_id GROUP BY dept.name

1. create .prejoin:emp,dept:(...)|condition   -- flat materialized join
2. insert all matching rows via nested scan
3. recursive build_queryplan with .prejoin as single table
   → make_keytable creates .(.prejoin:...):((get_column .pj false "dept.name" false))
   → aggregate columns computed on that group table
```

### Problem: Prejoin Tables Are Ephemeral

The current code explicitly drops and recreates prejoin tables each time (queryplan.scm:1434-1438):

```scheme
(list 'begin
    '('droptable pj_schema prejointbl true)      ;; ← kills any cached aggregates
    '('createtable pj_schema prejointbl ...)
    '('time materialize_plan "materialize")
    grouped_result)
```

This means the group table derived from the prejoin is ALSO ephemeral — any `StorageComputeProxy` columns on it are lost between queries. Subquery cache provides no benefit here.

### Solution: Make Prejoin Tables Cacheable

The prejoin table naming IS already canonical (`.prejoin:{tables}:{cols}|{condition}`), and it uses sloppy engine. To enable caching:

1. **Remove the explicit `droptable`** — rely on `createtable ... true` (IF NOT EXISTS) idempotency
2. **Before materialization, check if the prejoin table already has data** — if it does AND source tables haven't changed, skip materialization entirely
3. **Register invalidation triggers** on all source tables in the prejoin — on any INSERT/UPDATE/DELETE to source tables, drop the prejoin table (Phase 1: full invalidation) or selectively invalidate rows (Phase 2)

```scheme
/* Proposed change: */
(list 'begin
    ;; no droptable! createtable is IF NOT EXISTS
    '('createtable pj_schema prejointbl ...)
    ;; materialize only if table is empty or invalid
    '('if ('equal? 0 ('count_rows pj_schema prejointbl))
        '('time materialize_plan "materialize")
        nil)
    grouped_result)
```

### Invalidation for Prejoin Tables

Register triggers on each source table in the join:

```scheme
;; For each (tblvar schema tbl isOuter _) in tables:
;; Register AfterInsert/AfterUpdate/AfterDelete trigger that drops the prejoin table
(register_system_trigger schema tbl
    (concat ".cache:prejoin:" prejointbl "|" schema "." tbl)
    '(AfterInsert AfterUpdate AfterDelete)
    '(lambda (OLD NEW) (droptable pj_schema prejointbl true)))
```

When the prejoin table is dropped, the derived group table and its aggregate tempcols are either:
- Also dropped (if group table naming depends on prejoin name — which it does)
- Or orphaned and evicted by CacheManager

### Race Condition Improvement

Currently, concurrent GROUP BY queries on the same table race on the shared keytable: two queries may try to insert/compute on the same `.tbl:(keys)` temp table simultaneously, leading to duplicate work or stale aggregate columns. With proper trigger-based invalidation, the keytable becomes a stable cached artifact — queries only read from it, and only DML triggers mutate it. This naturally serializes writes (trigger fires once per DML) and allows concurrent reads without races.

### Benefit Chain

```
source table change
  → trigger drops .prejoin:... table
    → derived group table .(.prejoin:...):(...) also gone
      → next query: rematerializes prejoin + recomputes aggregates

No source change between queries:
  → prejoin table reused (skip materialization!)
    → group table reused
      → aggregate StorageComputeProxy columns still valid
        → 10-100x speedup for repeated multi-table aggregates
```

### Implementation Notes

- `count_rows` builtin needed (or check shard count) — lightweight check if table has data
- Prejoin tables registered with CacheManager as `TypeTempKeytable` (factor 10) — same as group tables
- The recursive `build_queryplan` call already treats the prejoin as a normal table, so group table creation, aggregate computation, and eventually StorageComputeProxy all work transitively
- ORDER BY + LIMIT on joined tables (queryplan.scm:1634 TODO) also benefits from cached prejoins

---

## Serialization

serialize all data (the whole storage) for safe reconstruction of the data.

to serialize the bitmask you can use `GetDataPtr()`

```go
// Serialize proxy state
func (p *StorageComputeProxy) Serialize(w io.Writer) error {
    write header
    // 1. Serialize main storage (may be nil)
    SerializeStorage(w, p.main)

    // 2. Serialize delta storage
    p.delta.Serialize(w)

    // 3. Serialize valid-mask
    maskPtr := p.validMask.GetDataPtr()
    maskSize := p.validMask.ByteSize()
    write maskSize
    w.Write(unsafe.Slice((*byte)(maskPtr), maskSize))

    // 4. Computor is reconstructed from column metadata at load time, filter is dropped
}

// Deserialize proxy state
func DeserializeComputeProxy(r io.Reader, colDef *column, shard *storageShard) *StorageComputeProxy {
	reverse serialize
}
```

---

## Canonical Naming

All cache objects use `.` prefix with deterministic structure. The naming must be **deterministic** so that:
- The same query (or semantically equivalent queries) reuse the same keytable/tempcol
- Different queries with different GROUP BY / conditions get distinct names
- Names are reconstructable from the query plan without lookup

### Keytable Names

**Current scheme:**
```
.{source_table}:({key_expr_1} {key_expr_2} ...)
```

Where each `key_expr` is the serialized Scheme expression, e.g. `(get_column cd_test false name false)`.

**Example:**
```
.employees:((get_column employees false dept false))
.cd_test:((get_column cd_test false category false) (get_column cd_test false name false))
```

For multi-stage GROUP BY (e.g. COUNT(DISTINCT)), subsequent stages nest with `.`:
```
..cd_test:((get_column cd_test false name false)):((get_column cd_test false name false))
```
This reads as: "keytable of the keytable of cd_test grouped by name, then grouped by name again."

**FK→PK optimization:** When `make_keytable` detects that the key is a single foreign key that references another table's primary key, it can reuse the referenced table directly instead of creating a temp table. In that case, the keytable name IS the referenced table name (no `.` prefix). The column mapping returns the PK column name instead of the serialized expression.

**Proposed canonical form for `make_keytable`:**
```
.{source_table}:({sorted_canonical_keys})
```
Where `sorted_canonical_keys` are the key expressions sorted lexicographically by their serialized form. This ensures `GROUP BY a, b` and `GROUP BY b, a` produce the same keytable (they have the same unique keys). The sort is only for naming — the actual key column order in the table follows the original GROUP BY order for partitioning alignment.

### Tempcol Names (Aggregate Columns on Keytables)

**Current scheme:**
```
{aggregate_triple}|{condition}
```

Where `aggregate_triple` is `(expr reduce neutral)` serialized, and `condition` is the WHERE condition serialized.

**Example:**
```
(1 + 0)|true                           -- COUNT(*) with no WHERE
(1 + 0)|(> (get_column t false val false) 30)   -- COUNT(*) WHERE val > 30
((get_column t false salary false) + 0)|true     -- SUM(salary) with no WHERE
```

This encodes the full semantics: what is being aggregated, how it is reduced, and under which filter condition. Two queries with the same aggregate expression and same WHERE clause will reference the same tempcol, enabling cache reuse.

**Proposed canonical form for tempcols:**
```
.({canonical_agg_expr} {reduce_op} {neutral})|{canonical_condition}
```

Canonicalization rules:
1. **Expressions**: `get_column` references use the resolved form `(get_column tblvar false col false)` — never the `nil` tblvar form
2. **Conditions**: Normalize to a canonical form (sorted AND-clauses, deduplicated). `nil` and `true` both canonicalize to `true`
3. **The `.` prefix** distinguishes computed tempcols from user-defined columns

### Lookup Cache Column Names

For scalar subquery / join lookup caches (StorageComputeProxy on base tables):

```
.{join_key_col}→{target_table}.{target_key}:({output_expr})
```

**Examples:**
```
.state→ticketState.ID:(final)          -- lookup ticketState.final via ticket.state = ticketState.ID
.dept_id→departments.id:(name)         -- lookup departments.name via emp.dept_id = departments.id
```

For compound keys:
```
.({key1},{key2})→{target_table}.({tkey1},{tkey2}):({output_expr})
```

### Summary Table

| Object | Naming Pattern | Example |
|--------|---------------|---------|
| **Keytable** (group) | `.{src}:({keys})` | `.employees:((get_column employees false dept false))` |
| **Keytable** (FK reuse) | `{referenced_table}` | `departments` |
| **Keytable** (nested) | `..{src}:({k1}):({k2})` | `..t:((gc t f name f)):((gc t f name f))` |
| **Tempcol** (aggregate) | `.({expr} {reduce} {neutral})\|{cond}` | `.(1 + 0)\|true` |
| **Lookup col** | `.{col}→{tbl}.{key}:({out})` | `.state→ticketState.ID:(final)` |

### Caveats for `make_keytable` Refactoring

1. **FK detection timing**: Checking whether a key matches a FK→PK relationship requires schema metadata (constraints) at plan time. Currently `build_queryplan` only receives `schema` (db name) and table tuples. FK metadata must be queryable from the schema or passed through.

2. **Keytable reuse and stale keys**: If a canonically-named keytable is reused across queries, it may contain keys from previous runs. This means:
   - HAVING must **always** be applied when scanning the keytable, even if the current query inserted only matching keys
   - The `collect_plan` must use `INSERT ... ON DUPLICATE KEY IGNORE` semantics (which `sloppy` engine already does)
   - Tempcols from previous runs may be stale — `createcolumn` must check validity via the valid-mask, not assume freshness

3. **Condition canonicalization**: Two semantically equivalent conditions like `a > 5 AND b < 10` vs `b < 10 AND a > 5` must produce the same canonical string. This requires sorting commutative operators' children. For now, rely on the optimizer to normalize conditions before they reach `build_queryplan`.

4. **Column mapping return value**: `make_keytable` must return a mapping from key expression → keytable column name, so that `replace_agg_with_fetch` / `replace_col_for_dedup` can correctly rewrite `(get_column src ...)` to `(get_column keytbl ...)`. When reusing an FK table, the column name is the actual PK column, not the serialized expression.

5. **Partitioning alignment**: The partitioning of the keytable should match the source table's sharding on the key column(s) for efficient collect scans. When reusing an FK table, the partitioning is already established and must not be changed.

---

## Open Implementation Details

### When is Compress() called?
- whenever sparse gets too full (3%) (even when we are midst in a GetValue or a initial build with filter)

### ValidMask growth on inserts
- happens automatically

### Thread safety
- delta must be locked with a RWMutex since it is implemented as a map
- rebuilds must acquire a full write mutex; in case multiple rebuilds (Compress) are triggered at the same time, early out if you see the validmask is full as soon as you get the write lock so we don't build it twice just because two goroutines tried to read a value over the 3% threshold 
- `GetValue` writes to `delta` and `validMask` - these must be thread-safe for concurrent reads
- `NonLockingBitmap` is threadsafe

---

## Implementation Readiness Audit

### What Already Exists

| Component | Status | Location |
|-----------|--------|----------|
| `make_keytable` function | DONE | queryplan.scm:242-252 |
| `make_col_replacer` (rewrite refs to group table) | DONE | queryplan.scm:257-267 |
| `rewrite_for_prejoin` | DONE | queryplan.scm:270-276 |
| Prejoin materialization (multi-table GROUP BY) | DONE | queryplan.scm:1353-1441 |
| `StorageSparse` (for delta storage) | DONE | storage/storage-sparse.go |
| `NonBlockingBitMap` (for valid-mask) | DONE | NonLockingReadMap package (used for deletions) |
| `ComputeColumn` (eager, full-compute) | DONE | storage/compute.go:37-153 |
| `createcolumn` with computor params 7-8 | DONE | storage/storage.go:472-514 (6-8 params) |
| Trigger system with IsSystem flag | DONE | storage/trigger.go |
| CacheManager with temp table/column lifecycle | DONE | storage/cache.go |
| Shard rebuild infrastructure | DONE | storage/shard.go |
| Canonical naming for keytables/tempcols | DONE | naming convention in make_keytable + compute_plan |

### What Must Be Built

| Component | Complexity | Blocking? | Notes |
|-----------|------------|-----------|-------|
| **`StorageComputeProxy` type** | HIGH | Phase 1 blocker | New type implementing `ColumnStorage`. Wraps main+delta+validMask+computor. Core of everything. |
| **Lazy `GetValue`** | HIGH | Part of proxy | Check validMask → read from delta/main → or compute + store in delta |
| **`Compress()` method** | MEDIUM | Part of proxy | Compute all values, run proposeCompression loop, replace main, clear delta |
| **`Invalidate/InvalidateAll`** | LOW | Part of proxy | Just clear bits in NonBlockingBitMap |
| **Serialization** | MEDIUM | Phase 1 | Serialize main + delta + validMask bytes. Reconstruct computor from column metadata. |
| **Extend `createcolumn` for filter** | LOW | Phase 1 | Parse `filter_cols`/`filter` from options assoc-list. Route to new `ComputeColumn` signature. |
| **Adapt `storageShard.ComputeColumn`** | MEDIUM | Phase 1 | Create proxy instead of StorageSCMER. Handle filter-based sparse init vs full Compress. |
| **`dropcolumn` Scheme builtin** | LOW | Phase 2 | Drop a computed column from all shards. Cleanup system triggers by naming convention. |
| **`ExtractScannedTables`** | LOW | Phase 2 | Walk computor AST for `(scan schema table ...)` patterns. |
| **Trigger registration** | MEDIUM | Phase 2 | Register AfterInsert/Update/Delete on source tables → dropcolumn target. |
| **Prejoin table caching** | MEDIUM | Phase 2/3 | Remove droptable, add data-presence check, register invalidation triggers on source tables. |
| **`count_rows` or equivalent** | LOW | Phase 2/3 | Needed for prejoin "already populated?" check. Could use shard count sum. |
| **FK→PK reuse in `make_keytable`** | LOW | Future | Schema metadata query for single-key FK detection. |
| **`$invalidate:COLNAME` virtual col** | MEDIUM | Phase 4 | Special column name prefix in scan column reader. |
| **JIT integration** | LOW | Phase 5 | `StorageComputeProxy.JIT()` with validMask bitmap check → fast/slow path. |

### Key Risk: `ComputeColumn` Refactor

The current `ComputeColumn` (compute.go:37-153) is **eagerly** computed — it builds a full `StorageSCMER` values array and stores it directly. The new version must:
1. Create a `StorageComputeProxy` instead of `StorageSCMER`
2. Handle the case where proxy already exists (idempotent createcolumn)
3. Support filter-based sparse computation
4. Deal with shards that have delta (inserts/deletions) — current code returns false and forces rebuild

The current code also requires delta-free shards (`s.deletions.Count() > 0 || len(s.inserts) > 0` → return false). The proxy approach can work with deltas because it reads values lazily, but the validMask must cover `main_count + len(inserts)`.

---

## Implementation Phases

### Phase 1: StorageComputeProxy (Week 1-2)
1. Implement `StorageComputeProxy` struct in `storage/compute_proxy.go`
   - Implement `ColumnStorage` interface (GetValue, proposeCompression, scan, build, init, finish, Serialize, Deserialize)
   - Lazy GetValue with validMask check → delta → main → compute path
   - Compress() with proposeCompression loop
   - Invalidate/InvalidateAll
   - Thread safety: RWMutex on delta, NonBlockingBitMap for validMask
2. Extend `createcolumn` to parse `filter_cols`/`filter` from options assoc-list (storage/storage.go:472)
3. Refactor `table.ComputeColumn` + `storageShard.ComputeColumn` to create proxy instead of eager StorageSCMER
4. Implement serialization (main + delta + validMask bytes, computor from column metadata)
5. Test: manual `createcolumn` with computor, verify lazy evaluation, verify Compress triggers at 3%
6. **No triggers yet** — manual dropcolumn for invalidation

### Phase 2: Lookup Cache + Triggers (Week 3-4)
1. Implement `dropcolumn` Scheme builtin — drop computed column from all shards, cleanup system triggers
2. `ExtractScannedTables` — walk computor AST for `(scan schema table ...)` patterns
3. `RegisterComputedColumnTriggers` — register AfterInsert/Update/Delete on source tables → dropcolumn
4. `analyze_join_for_cache` in queryplan.scm — detect cacheable JOINs, emit `createcolumn` with canonical names
5. Test invalidation: change source table → cache invalidated → next query recomputes

### Phase 3: Aggregation Cache + Prejoin Caching (Week 5-6)
1. Apply StorageComputeProxy to group table aggregate tempcols
2. **Make prejoin tables cacheable**: remove droptable, add data-presence check, register source-table triggers
3. Implement `count_rows` builtin (or inline shard count check) for prejoin freshness test
4. Register invalidation triggers on prejoin source tables
5. Test: repeated multi-table GROUP BY queries hit cache; DML on source invalidates

### Phase 4: Selective Invalidation (Week 7-8)
1. Analyze scan filters for selective invalidation patterns
2. Implement `$invalidate:COLNAME` virtual column in scan.go
3. Adapt trigger code: instead of dropcolumn, selectively invalidate affected rows

### Phase 5: Incremental Aggregates + JIT + Polish (Week 9-10)
1. `update_aggregate` builtin for SUM/COUNT — incremental trigger updates instead of full recompute
2. `StorageComputeProxy.JIT()` with validMask bitmap check → fast/slow path
3. Correctness tests (invalidation, concurrent access, cascading)
4. Performance benchmarks
5. Memory pressure handling (CacheManager eviction of proxy columns)

---

## Test Cases

```yaml
- name: "Lookup cache lazy evaluation"
  setup:
    - "CREATE TABLE status (id INT PRIMARY KEY, name VARCHAR(50))"
    - "CREATE TABLE ticket (id INT, status_id INT)"
    - "INSERT INTO status VALUES (1,'open'),(2,'closed')"
    - "INSERT INTO ticket VALUES (1,1),(2,2),(3,1)"
  queries:
    - query: "SELECT t.id FROM ticket t JOIN status s ON s.id = t.status_id WHERE s.name = 'open'"
      result: [[1],[3]]
    - query: "SELECT t.id FROM ticket t JOIN status s ON s.id = t.status_id WHERE s.name = 'open'"
      result: [[1],[3]]  # Cache hit

- name: "Lookup cache invalidation on source update"
  setup:
    - "CREATE TABLE status (id INT PRIMARY KEY, name VARCHAR(50))"
    - "CREATE TABLE ticket (id INT, status_id INT)"
    - "INSERT INTO status VALUES (1,'open'),(2,'closed')"
    - "INSERT INTO ticket VALUES (1,1),(2,2)"
  queries:
    - "SELECT t.id FROM ticket t JOIN status s ON s.id = t.status_id WHERE s.name = 'open'"
    - "UPDATE status SET name = 'archived' WHERE id = 1"
    - query: "SELECT t.id FROM ticket t JOIN status s ON s.id = t.status_id WHERE s.name = 'open'"
      result: []

- name: "Basic lookup cache"
  setup:
    - "CREATE TABLE a (id INT, val INT)"
    - "CREATE TABLE b (id INT, a_id INT)"
    - "INSERT INTO a VALUES (1,10),(2,20)"
    - "INSERT INTO b VALUES (1,1),(2,2)"
  queries:
    # JOIN wird zu computed column .a_id→a:(val) auf b
    - query: "SELECT b.id, a.val FROM b JOIN a ON a.id = b.a_id"
      result: [[1,10],[2,20]]

- name: "Cascading invalidation"
  setup:
    - "CREATE TABLE status (id INT PRIMARY KEY, is_open BOOL)"
    - "CREATE TABLE ticket (id INT, status_id INT)"
    - "INSERT INTO status VALUES (1,true),(2,false)"
    - "INSERT INTO ticket VALUES (1,1),(2,1),(3,2)"
  queries:
    - query: "SELECT COUNT(*) FROM ticket t JOIN status s ON s.id = t.status_id WHERE s.is_open"
      result: [[2]]
    - "UPDATE status SET is_open = false WHERE id = 1"
    - query: "SELECT COUNT(*) FROM ticket t JOIN status s ON s.id = t.status_id WHERE s.is_open"
      result: [[0]]  # Both lookup cache AND aggregate invalidated

- name: "Prejoin aggregate cache"
  setup:
    - "CREATE TABLE dept (id INT PRIMARY KEY, name VARCHAR(50))"
    - "CREATE TABLE emp (id INT, dept_id INT, salary INT)"
    - "INSERT INTO dept VALUES (1,'Engineering'),(2,'Sales')"
    - "INSERT INTO emp VALUES (1,1,100),(2,1,200),(3,2,150)"
  queries:
    # Multi-table GROUP BY → prejoin materialization → group table → aggregate
    - query: "SELECT d.name, SUM(e.salary) FROM emp e JOIN dept d ON d.id = e.dept_id GROUP BY d.name ORDER BY d.name"
      result: [["Engineering",300],["Sales",150]]
    # Second run should reuse cached prejoin + aggregate
    - query: "SELECT d.name, SUM(e.salary) FROM emp e JOIN dept d ON d.id = e.dept_id GROUP BY d.name ORDER BY d.name"
      result: [["Engineering",300],["Sales",150]]
    # DML on source table invalidates cache
    - "INSERT INTO emp VALUES (4,2,250)"
    - query: "SELECT d.name, SUM(e.salary) FROM emp e JOIN dept d ON d.id = e.dept_id GROUP BY d.name ORDER BY d.name"
      result: [["Engineering",300],["Sales",400]]  # Cache invalidated, recomputed
```

---

## Summary

**Key design decisions:**

1. **StorageComputeProxy** - Wraps storage + valid-mask + computor; implements `ColumnStorage` interface
2. **Unified valid-mask** - Covers Main + Delta rows uniformly
3. **Bit-based invalidation** - O(1) to mark entry invalid, recompute on next read
4. **Cascading invalidation** - Dependent caches automatisch re-invalidiert via Trigger
5. **Idempotent createcolumn** - Mehrfacher Aufruf garantiert dass filter-Zeilen berechnet sind
6. **Kanonische Namen** - Wiederverwendung durch deterministische Column-Namen
7. **Cacheable prejoins** - Multi-table GROUP BY prejoin tables persist and benefit from the same caching transitively via recursive build_queryplan

**Expected performance:**
- Lookup joins: 10-100x faster (eliminate nested scan overhead)
- Aggregations: 10-100x faster (avoid full table rescan)
- Multi-table aggregations: skip materialization entirely on cache hit
- Write overhead: minimal (bit clear or delta update)
- Memory overhead: ~8 MB per 1M cached rows

---

## Cross-Dependencies: JIT (todos/jit.md)

The JIT system compiles scan inner-loops to machine code. With `StorageComputeProxy`, JIT integration is straightforward.

### JIT für StorageComputeProxy

Alle computed columns nutzen StorageComputeProxy. Der Unterschied:

| Zustand | main | JIT Handling |
|---------|------|--------------|
| **Nach Compress()** | Optimierter Storage (StorageInt etc.) | Valid-mask check → fast path zu main.JIT() |
| **Sparse (vor Compress)** | nil | Valid-mask check → slow path (Go callback) |

### Proxy.JIT() Method

The proxy implements its own `JIT()` method:

```go
func (p *StorageComputeProxy) JIT(w *JITWriter, ctx *JITContext) JITValueDesc {
    // Wenn main noch nicht gebaut → immer slow path (Go callback)
    if p.main == nil {
        w.EmitGoCallback(p.getValueCallback, ctx.IdxReg)
        return JITValueDesc{Type: JITTypeUnknown, Loc: LocRegPair, Reg: RegReturn, Reg2: RegReturnAux}
    }

    // main existiert → valid-mask check + fast/slow path
    slowLabel := w.ReserveLabel()
    continueLabel := w.ReserveLabel()

    // test [validMask + idx/8], (1 << idx%8)
    EmitBitmapGet(w, p.validMask.GetDataPtr(), ctx.IdxReg, slowLabel)

    // Fast path: read from main (alle Werte sind nach Compress valid)
    desc := p.main.JIT(w, ctx)
    w.Jmp(continueLabel)

    // Slow path: Go callback (sollte nach Compress nicht mehr vorkommen)
    w.MarkLabel(slowLabel)
    w.EmitGoCallback(p.getValueCallback, ctx.IdxReg)

    w.MarkLabel(continueLabel)
    return desc
}
```

### Benefits of Proxy Approach for JIT

1. **Encapsulation**: All lazy-evaluation logic is in `Proxy.JIT()`, not scattered
2. **Reuse EmitBitmapGet**: Same pattern as deletion bitmap
3. **Single integration point**: JIT ruft immer `storage.JIT()` - Proxy ist transparent
