# JIT Correctness Fix — Aufgabe für nächste Session

## Branch / Worktree
Branch: `jit-tailcall`
Worktree: `/home/carli/projekte/memcp/.claude/worktrees/jit-tailcall`

## Ziel
1. Infinite loop in TestStorageSeqJITEmitMultipleSequences fixen
2. Danach alle Tests grün + Benchmarks + Cleanup + Commit

---

## Aktueller Stand

### Was funktioniert (aus vorherigen Sessions)
- `TestStorageSeqJITEmitLinear` ✓
- `TestStorageSeqJITEmitStride` ✓
- `TestStorageSeqJITEmitConstants` ✓

### Angewandte Fixes (noch aktiv in storage/storage-seq.go)
1. **AllocReg ProtectedRegs-Fix** (`scm/jit_types.go` Zeile ~231): FreeRegs-Pfad schließt ProtectedRegs aus
2. **d117 ProtectReg** (~1897), **d152 ProtectReg** (~2452)
3. **idxInt ProtectReg** (~132), **thisptr ProtectReg** (~136)
4. **d45/d46/d47 ProtectReg** (~849-851 bei lbl13 entry):
   - Schützt diese Register, damit sie nicht während lbl13/lbl12 code-gen evicted werden
   - UnprotectReg bei lbl22 (~1946-1948) und lbl21 (~1969-1971)

### Aktuelles Problem: INFINITE LOOP

**Symptom**: Nach Fix 4 hängt `TestStorageSeqJITEmitMultipleSequences` beim ersten `jitGet(int64(0))`.

**Disassemblierter JIT-Code** liegt in `/tmp/jit_seq_multi.bin`:
```bash
objdump -D -b binary -m i386:x86-64 /tmp/jit_seq_multi.bin
```
Debug-Test dafür: `TestStorageSeqJITDebugMulti` in `storage/jit_debug_test.go`

**Bisherige Analyse des disassemblierten Codes** (Offsets für seqCount=2 Fall):
- `0x42` = lbl1 (outer loop top: load pivot/min/max)
- `0x163` = lbl2 (recordId[pivot] Ergebnis → Vergleich mit idx)
- `0x185` = lbl9 (idx >= recordId[pivot])
- `0x1a2` = lbl8 (idx < recordId[pivot])
- `0x1ce` = lbl11 (d46==d47 check)
- `0x22b` = lbl13 (second step)
- `0x387` = Vergleich idx vs recordId[d45]
- `0x502` = lbl22 (store d45,d47 to stack[56,64])
- `0x511` = lbl21 (store d46,d45-1 to stack[56,64])
- `0x3a9` = lbl12 (final: load stack[24], compute result)
- `0x81e` = lbl33 (check stack[56]==stack[64])
- `0x858` = lbl48 (new pivot, store to stack[0,8,16], JMP lbl1=0x42)
- `0x875` = lbl0 (epilogue + ret)

**Trace für idx=0, seqCount=2** (logisch korrekt, sollte terminieren):
- Initial: pivot=0, min=0, max=1
- lbl1→lbl9: recordId[0]=0, 0≥0 → lbl9 → store(1,0,1) → lbl11
- lbl11: d45=1, d46=0, d47=1, d46≠d47 → lbl13
- lbl13: recordId[d45=1]=5, idx=0 < 5 → lbl21
- lbl21: store d46=0 → stack[56], d45-1=0 → stack[64] → lbl33
- lbl33: stack[56]=0=stack[64] → converged → lbl12 → return 0 ✓

**Verdacht**: Der Trace scheint korrekt, aber der Test hängt. Mögliche Ursachen:
1. Ein Backward-Loop in den bit extraction Routinen (für start/stride) geht infinite
2. Verdächtiger Bug bei Offset 0x666: `MOV R15D, 64` / `SUB R15, R15` = 0 statt 64-bit_offset
   - Aber nur im cross-word Pfad (nicht getroffen für unsere Daten)
3. Vielleicht hängt es NICHT bei idx=0 sondern bei einem anderen idx-Wert

**Nächster Schritt**: Den Loop-Einstieg mit printf-Debugging oder einem Timeout isolieren.
Alternativ: `TestStorageSeqJITDebugMulti` durch `jitBuildGetValueFunc` ersetzen und mit kurzer Schleife + Timeout testen.

---

## Datenstruktur für seqCount=2 (wichtig fürs Debugging)

Test-Werte: `[0,1,2,3,4, 100,200,300,400,500]`
- seqCount=2, count=10
- recordId: bitsize=3, offset=0 → Werte [0, 5] (nicht [0,5,10]!)
- start:    bitsize=7, offset=0 → Werte [0, 100]
- stride:   bitsize=7, offset=1 → Werte [1-1=0, 100-1=99] (stored)

Stack-Layout (RSP-relativ, frame=0x70 Bytes):
- [0x00] (0): outer pivot
- [0x08] (8): outer min
- [0x10] (16): outer max
- [0x18] (24): converged sequence index (→ lbl12 input)
- [0x20] (32): step-1 new_pivot (d45)
- [0x28] (40): step-1 new_min (d46)
- [0x30] (48): step-1 new_max (d47)
- [0x38] (56): step-2 min candidate (→ lbl33 input)
- [0x40] (64): step-2 max candidate (→ lbl33 input)
- [0x48..] (72+): spill slots für bit-extraction loops

---

## Schnellster Fix-Ansatz

Option A: **Direkter Ansatz** - Den Grund des Hangs finden:
```bash
go test ./storage/ -run TestStorageSeqJITEmitMultipleSequences -v -count=1 -timeout 5s
```
Mit Print-Debugging: Vor dem ersten `jitGet` call print, nach erstem jitGet print, nach zweitem, etc.

Option B: Den **Debug-Test** für Multi-Case erweitern (Datei hat jetzt `TestStorageSeqJITDebugMulti`), um tatsächlich die JIT-Funktion aufzurufen mit SIGALRM-Timeout.

Option C: Statt der ProtectReg-Lösung: **Lazy reload** - d45/d46/d47 kurz vor lbl22/lbl21 aus Stack re-laden statt zu schützen.

---

## Aufräumen (nach Fix)
- `storage/jit_debug_test.go` löschen (TestStorageSeqJITDebug + TestStorageSeqJITDebugMulti)
- `jit-todo.md` löschen
- Commit
