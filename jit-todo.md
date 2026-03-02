# JIT Correctness Fix — Aufgabe für nächste Session

## Branch / Worktree
Branch: `jit-tailcall`
Worktree: `/home/carli/projekte/memcp/.claude/worktrees/jit-tailcall`

## Ziel dieser Session
1. Remaining JIT-Korrektheitsfehler fixen
2. Benchmarks: JIT vs Go-GetValue vs Interpreter für StorageSeq

---

## Was bereits funktioniert
- `TestStorageSeqJITEmitLinear` ✓ (0,1,2,...,99 → korrekt)
- `TestStorageSeqJITEmitStride` ✓ (10,20,...,500 → korrekt nach ProtectReg d117-Fix)
- `TestStorageSeqJITDebug` ✓

## Was noch kaputt ist

### TestStorageSeqJITEmitMultipleSequences (FAIL)
Input: `[0,1,2,3,4, 100,200,300,400,500]` — zwei Sequenzen
```
idx=1: JIT got 0,   expected 1
idx=2: JIT got 0,   expected 2
idx=5: JIT got 0,   expected 100
idx=6: JIT got 6,   expected 200  ← völlig falsch
```
Vermutung: Bei mehreren Sequenzen (seqCount>1) hat die Bisektionslogik einen
Register-Clobbering-Bug. Wahrscheinlich überlebt ein weiterer Wert (z.B. d83=min
oder idxInt=idx) die Bisektion nicht korrekt.

### TestStorageSeqJITEmitWithNull (FAIL / panic)
Input: `[10, nil, 30, nil, 50]` — Sequenz mit Null-Werten
Panic in JIT-Code (ungültige Speicherzugriff). Wahrscheinlich hasNull-Pfad
generiert fehlerhaften Code oder ProtectReg fehlt für einen weiteren Wert.

### TestStorageSeqJITEmitRegPtr (vermutlich auch noch betroffen)
Nicht explizit getestet nach letztem Fix.

---

## Bisheriger Fix-Ansatz (bereits committed)

**Root cause war**: `AllocReg()` prüfte `ProtectedRegs` nur im Spill-Pfad,
nicht beim FreeRegs-Pfad. Fix in `scm/jit_types.go`:
```go
// Zeile ~230: FreeRegs-Pfad schließt jetzt ProtectedRegs aus
available := ctx.FreeRegs &^ ctx.ProtectedRegs
if available != 0 {
    bit := available & (-available)
    ctx.FreeRegs &^= bit
    ...
}
```

**Geschützte Variablen** (in `storage/storage-seq.go`):
- `d152` (stride): ProtectReg nach Zeile ~2452, UnprotectReg vor ~2985
- `d117` (start): ProtectReg nach Zeile ~1897, UnprotectReg vor ~3064

**Problem**: Weitere Variablen haben wahrscheinlich dasselbe Lifetime-Problem.

---

## Debugging-Strategie für neue Session

### Schritt 1: Welche Variablen leben lange?
In `storage/storage-seq.go` gibt es folgende "long-lived" Variablen, die
über den gesamten recordId-Extraktionsblock (Zeilen ~1900–2980) leben:
- `d117` (start) — bereits geschützt
- `d152` (stride) — bereits geschützt
- `idxInt` (idx-Eingabe) — lebt von Anfang bis ~2957, wird im IMUL verwendet
- `d83` (min aus Bisektion) — lebt von ~1370 bis ~2897 (FreeDesc bei lbl40)

Für jeden dieser Werte prüfen: Ist er LocReg? Braucht er ProtectReg?

### Schritt 2: MultipleSequences debuggen
```bash
go test ./storage/ -run TestStorageSeqJITEmitMultipleSequences -v -count=1
```
Ausgabe zeigt: idx=6 JIT=6, expected=200. Das ist `idx * stride_falsch = 6 * 1`.
Also stride=1 statt 100. Die zweite Sequenz (100,200,...,500) hat stride=100.
Wahrscheinlich wird stride der zweiten Sequenz nicht korrekt in d152 geladen.

### Schritt 3: Mit objdump analysieren (falls nötig)
```bash
go test ./storage/ -run TestStorageSeqJITDebug -v -count=1
objdump -D -b binary -m i386:x86-64 /tmp/jit_seq.bin | grep -A3 "imul"
```

---

## Struktureller Fix (mittelfristig)

Statt manueller ProtectReg-Calls für jede lange Variable: In `jitgen` eine
generelle Lösung implementieren. Optionen:
1. **Lazy computation**: stride/start erst kurz vor der Verwendung berechnen
2. **Stack-Spill + Reload**: Werte explizit auf Stack sichern und vor Verwendung laden
3. **jitgen Live-Range-Analyse**: jitgen erkennt automatisch lange Lifetimes
   und fügt ProtectReg ein

---

## Nach dem Korrektheits-Fix: Benchmarks

```bash
go test ./storage/ -bench 'BenchmarkStorageSeq' -run '^$' -benchtime=2s -count=3
```

Benchmark-Setup ist bereits in `storage/storage-seq_jit_test.go`:
- `BenchmarkStorageSeqGetValue/Go` — Go GetValue in Schleife
- `BenchmarkStorageSeqGetValue/JIT_ConstFold` — JIT mit const thisptr
- `BenchmarkStorageSeqGetValue/JIT_RegPtr` — JIT mit reg thisptr

Ziel: JIT sollte deutlich schneller sein als Go GetValue.
Eventuell auch Vergleich mit dem Scheme-Interpreter-Pfad.

---

## Aufräumen nach Fix
- `storage/jit_debug_test.go` löschen (temporärer Debug-Test)
- `jit-todo.md` löschen
- Commit mit allen Fixes

## Relevante Dateien
- `scm/jit_types.go`: AllocReg (Zeile ~229), ProtectReg, TransferReg
- `storage/storage-seq.go`: JITEmit, d117-Fix (~1897), d152-Fix (~2452)
- `storage/storage-seq_jit_test.go`: Korrektheitstests + Benchmarks
- `storage/jit_debug_test.go`: Debug (löschen nach Fix)
- `tools/jitgen/main.go`: Code-Generator
