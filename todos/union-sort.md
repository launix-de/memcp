# Shuffle/Merge-Operator (nur Konzept)

## Ziel
Ein Operator für das Mergen von bereits vorsortierten Teilströmen ohne globale Materialisierung.

## Start
Der Operator wird einmal gestartet mit:

- `n_slots`: Anzahl Eingabeströme
- `less_fn(a, b) -> bool`: Sortierkriterium
- `map_fn(item) -> mapped`
- `neutral`: Startwert des Akkumulators
- `reduce_fn(acc, mapped) -> acc`

Er liefert `n_slots` Queue-Handles zurück, genau einen pro Unterabfrage.

## Input-Vertrag

- Jede Unterabfrage schreibt nur in ihren eigenen Slot.
- Pro Push wird ein Scmer-Item geliefert. Für SQL-UNION sind das Assoc-Listen (wie `$update`), für Micro-Tests können es auch simple Werte (z.B. Integer) sein.
- Jeder Slot muss explizit als `done/close` markiert werden, wenn keine Items mehr kommen.

## Go-interne Abbildung

- Jeder Slot ist intern ein eigener `chan item`.
- Der Merge-Kern hält für jeden offenen Slot genau ein aktuelles Kopf-Element (wenn vorhanden).
- Danach wiederholt:
1. kleinstes Kopf-Element über alle Slots via `less_fn` wählen,
2. `map_fn` darauf anwenden,
3. Ergebnis in `reduce_fn` falten,
4. aus demselben Slot das nächste Element per Channel nachladen.
- Ende, wenn alle Slots geschlossen und leer sind.
- Initialer Akkumulator ist `neutral`.

## Queue-Semantik

- `push(item)` blockiert bei Backpressure (normale Channel-Semantik).
- Slot-Reihenfolge muss bereits vorsortiert sein; der Operator sortiert innerhalb eines Slots nicht nach.
- `push` nach `close` ist Fehler.

## Querybuilder-Integration

- Für jede UNION-Branch wird genau ein Slot/Queue reserviert.
- Im innersten Branch-Plan wird `resultrow` nicht direkt aufgerufen, sondern durch `queue.push(item)` ersetzt.
- `item` bleibt die Assoc-Liste im üblichen Row-Format.
- Nach Ende des Branch-Scans muss die Queue explizit mit `queue.close()` geschlossen werden.
- Wichtig: `close` muss auch bei leeren Branches erfolgen, damit der Merge-Operator terminieren kann.

## Ergebnis

Der Operator liefert den finalen Reduce-Akkumulator zurück (bei komplett leerem Input: `neutral`).

## Test Cases

1. **Zwei Slots, normaler Merge**
- Setup: `n_slots=2`, `less_fn` auf `item["k"]` aufsteigend.
- Slot 0 pusht: `k=1`, `k=3`, `k=5`; Slot 1 pusht: `k=2`, `k=4`, `k=6`; beide schließen.
- Erwartung: globale Ausgabe-Reihenfolge `1,2,3,4,5,6`; `reduce_fn` sieht genau diese Reihenfolge.

2. **Drei Slots, ungleich lange Streams**
- Slot 0: `1,10`; Slot 1: `2`; Slot 2: `3,4,5,6`; dann close.
- Erwartung: `1,2,3,4,5,6,10`; Ende erst nach `close` aller Slots.

3. **Leerer Slot**
- Slot 0 pusht nichts und schließt sofort; Slot 1 liefert `1,2,3`.
- Erwartung: identisches Ergebnis wie nur Slot 1; kein Deadlock.

4. **Dubletten über Slots**
- Slot 0: `k=1,k=2`; Slot 1: `k=1,k=3`.
- Erwartung: beide `k=1` werden ausgegeben (kein DISTINCT).

5. **Assoc-Listen-Format**
- Push mit flacher Assoc-Liste wie `("id" 7 "name" "x")`.
- Erwartung: `map_fn` kann Felder zuverlässig lesen; kein Sonderformat nötig.

6. **Backpressure**
- Kleiner Channel-Buffer, Konsument künstlich langsam.
- Erwartung: `push` blockiert kontrolliert, keine Busy-Loops, keine Datenverluste.

7. **Fehler: push nach close**
- Slot wird geschlossen, danach erneuter `push`.
- Erwartung: klarer Fehler pro Slot.

8. **Fehler: unsortierter Slot-Input**
- Slot liefert `k=1,k=5,k=3`.
- Erwartung: definierte Reaktion (mindestens dokumentierter Fehler/Assertion im Debug-Modus), da Slot-Input-Vertrag verletzt ist.

9. **Leerer Gesamtinput**
- Alle Slots schließen ohne Push.
- Erwartung: Rückgabewert ist exakt `neutral`.

## SQL/YAML-Testfälle (zusätzlich)

Diese Fälle sind als SQL-Integrationstests gedacht (z.B. in `tests/70_union_all.yaml` oder einer eigenen `tests/72_union_sort.yaml`):

```yaml
test_cases:
  - name: "UNION ALL global ORDER BY merged"
    sql: |
      SELECT id FROM t_a WHERE id IN (1,3,5)
      UNION ALL
      SELECT id FROM t_b WHERE id IN (2,4,6)
      ORDER BY id
    expect:
      rows: 6
      data:
        - id: 1
        - id: 2
        - id: 3
        - id: 4
        - id: 5
        - id: 6

  - name: "UNION ALL preserves duplicates in merge"
    sql: |
      SELECT id FROM t_a WHERE id = 1
      UNION ALL
      SELECT id FROM t_b WHERE id = 1
      ORDER BY id
    expect:
      rows: 2
      data:
        - id: 1
        - id: 1

  - name: "UNION ALL global ORDER BY with LIMIT"
    sql: |
      SELECT id FROM t_a
      UNION ALL
      SELECT id FROM t_b
      ORDER BY id
      LIMIT 3
    expect:
      rows: 3

  - name: "UNION ALL global ORDER BY with OFFSET LIMIT"
    sql: |
      SELECT id FROM t_a
      UNION ALL
      SELECT id FROM t_b
      ORDER BY id
      LIMIT 2 OFFSET 2
    expect:
      rows: 2

  - name: "UNION ALL with per-branch sorted subqueries"
    sql: |
      SELECT id FROM (
        SELECT id FROM t_a ORDER BY id DESC LIMIT 2
      ) a
      UNION ALL
      SELECT id FROM (
        SELECT id FROM t_b ORDER BY id ASC LIMIT 2
      ) b
      ORDER BY id
    expect:
      rows: 4

  - name: "UNION ALL ORDER BY unknown projection column"
    sql: |
      SELECT id FROM t_a
      UNION ALL
      SELECT id FROM t_b
      ORDER BY missing_col
    expect:
      error: true
```
