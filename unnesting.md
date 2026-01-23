# Unnesting-Integration für euren SQL-Parser & Query-Plan-Builder

Dieses Dokument beschreibt, wie du **Unnesting / Dekorrelation** (nach dem Geist von *Neumann/Kemper: “Unnesting Arbitrary Queries”*) in deinen bestehenden Scheme-basierten SQL-Parser und Queryplan-Builder integrierst – **ohne** großflächige Temp-Tables, sondern bevorzugt **streaming/batched** und mit **persistenten Aggregations-Caches**.

Es basiert auf dem aktuellen Stand eurer Dateien und dem Ziel, **ohne klassische Temp-Tables** zu arbeiten:

- `sql-parser.scm`: erzeugt bereits AST-Knoten für Subqueries in Ausdrücken:
  - `(inner_select sub)` (skalare Subquery)
  - `(inner_select_in expr sub)` / negiert
  - `(inner_select_exists sub)` / negiert
- `queryplan.scm`: `untangle_query` flacht Subselects in `FROM` bereits ab (Alias-Prefixing, Column-Rewrite), **verwirft aber GROUP/LIMIT in Subselects**. `inner_select{,_in,_exists}` in Ausdrücken sind **noch nicht korrekt unterstützt**; eine frühere Rewrite-Idee ist fehlerhaft und sollte entfernt werden.

---

## 1) Zielbild: Was Unnesting bei euch leisten soll

**Problem:** Korrelierten Unterabfragen (Subqueries, die auf Spalten der äußeren Query referenzieren) werden in vielen Engines als “für jede Outer-Zeile neu ausführen” umgesetzt → **Nested Loop / Dependent Join**, oft teuer.

**Ziel:** Subqueries **dekorrelieren** (unnesten), so dass sie **einmal** (oder in Batches) für viele Outer-Bindungen ausgewertet werden und sich in **Joins + (Group/Distinct/Marker)** übersetzen lassen.

Wichtig bei euch:  
- Nested Loop Joins kann die Storage Engine schon.  
- Trotzdem soll es effizienter werden → *Scan-Multi / Batched Apply* (siehe unten).  
- **Fast keine Temp-Tables**:
  - erlaubt: **berechnete temporäre Spalten** auf einer Tabelle (z.B. Marker/Flags/Inline-Cache)
  - erlaubt: **persistente Aggregations-Temp-Tables** (Cache-Tabellen) für `T GROUP BY X,Y` inkl. Aggregaten (die wiederum berechnete temporäre Spalten sind)
  - Temporäre Tabellen und Spalten beginnen ihren Namen mit ., siehe auch Group-Code in queryplan.scm
  - vermeiden: allgemeines Materialisieren von D/Derived tables als klassische Temp-Tables
  - Nice-To-Have für später: Trigger, die bei Änderungen der Basistabellen die berechneten temporären Spalten und Aggregation-Temptables invalidieren oder aktualisieren; somit müssen Temptables und cols nicht jedes mal neu angelegt werden, sondern lediglich neu berechnet oder wiederverwendet werden; das genaue System wann invalidiert und wann die aggregation (z.B. SUM) in echtzeit aktualisiert wird, muss noch entschieden werden

---

## 2) Wo integrieren? (Empfohlene Pipeline)

Implementierung in: `untangle_query` (passiert vor `build_queryplan`)
Wichtig: zuerst **Untangle/Flatten** (FROM-Subselects), dann Unnesting von **inner_select**‑Ausdrücken.

0. **Untangle** `FROM`-Subselects:
   - Aliase prefixen (`id:alias`)
   - `(get_column ...)` sauber auflösen
   - Aktuell kein GROUP/LIMIT/OFFSET in Subselects
1. **Subquery-Knoten finden** in SELECT / WHERE / HAVING / ORDER BY.
2. **Korrelation bestimmen** (free vars / corr keys).
3. Je Subquery-Typ in **Join-fähige** Form transformieren:
   - `EXISTS` → Semi-Join oder Left-Join + Marker
   - `IN` → Semi-Join (später optional korrekt mit NULL-Logik)
   - scalar subquery → Left-Join auf “exactly one row per corr key” (über GROUP oder Top-1-per-group)
   - Tabellen-Liste erweitern, join-conditions hinzufügen

Das Ergebnis ist eine flache Query-IR, die euer existierender Join/Group-Planbuilder gut verarbeiten kann.

Ist der Query einmal "flach" (also besteht nur noch aus einer tabellenliste mit selects und where-conditions usw), kann build_queryplan dann die optimale Strategie wählen (aktuell muss aber noch nicht optimiert werden, es soll erst mal prinzipiell funktionieren)


## 3) Voraussetzungen im Compiler (Planbuilder-seitig)

### 3.1 Free-Var/Korrelationserkennung
Du brauchst eine Funktion, die in einem Ausdruck alle Spaltenreferenzen sammelt und entscheidet, ob sie **außen** oder **innen** gebunden sind. Praktisch: nutze das vorhandene Schema‑Wissen (via `SHOW`/Schema‑Metadaten) und eine abgewandelte Form von `replace_find_column`, um unqualifizierte Spalten gegen **inneres** Schema zu binden und nur echte Outer‑Refs als free vars zu markieren.

Konzeptuell:
- `local` = Column gehört zu Tabellen/Scopes der Subquery (hat vorrang, prüfe gegen das schema)
- `free` = Column gehört zum Outer-Scope → **corr keys** (der rest)

Du brauchst daraus:
- `corr_keys`: Liste der äußeren Spalten, die die Subquery parametrisieren
- `inner_local_refs`: innere Spalten/Tabellen (inkl. Auflösung unqualifizierter Spalten)

### 3.2 Normierte Join-Formen (mindestens logisch)
Damit Unnesting sauber geht, muss eure IR/Plan-Schicht (notfalls emuliert) Folgendes darstellen können:

- **Left Join** (für scalar subqueries + EXISTS-Marker); kann über `scan`/`scan_order` mit `isOuter=true` emuliert werden
- **Semi Join** (EXISTS/IN effizient) *(kann notfalls via Left Join + Marker emuliert werden)*
- **Anti Join** (NOT EXISTS/NOT IN) *(dito)*
- **Group By / Aggregation** (für scalar subqueries, DISTINCT, Marker)
- **Distinct** (oder Group By ohne Aggregat)

---

## 4) Storage Engine: Anforderungen unter euren Constraints

### 4.1 Was ihr schon habt
- **Nested Loop Join**: vorhanden
- `scan`/`scan_order` unterstützen `isOuter=true` → NULL‑Row bei no‑hit (Left‑Join‑Semantik)

### 4.2 Was ihr *minimal* zusätzlich braucht (ohne “klassische Temp-Tables”)
Um “D statt dependent join” ohne Temp-Tables zu schaffen, brauchst du eine **batched / multi-key** Ausführung:

#### A) `scan_multi` / paralleler sortierter scan auf mehreren Tabellen gleichzeitig (wichtigster Baustein, später)
Eine Scan-/Index-API, die mehrere Tabellen entgegennimmt und den Join schon Storage-Engine-seitig umsetzt (aber brauchen wir erst später wenn es ums Optimieren geht)
Alternativ: scan- und scan_order aufrufe so optimieren, dass der zuletzt durchlaufene Key gecacht wird und somit die Binärsuche auf O(1) verkürzt werden kann, wenn im inneren scan der nächste key abgefragt wird

- `index_lookup_multi(index, keys[]) -> stream(rows)`
- oder `scan_with_keyset(table, keyset) -> stream(rows)`

Damit kann man das Neumann/Kemper-Prinzip “Domain D der corr keys” **streaming** umsetzen, ohne D zu materialisieren.

#### B) Streaming-GroupBy / Hash-Aggregation
Für scalar subqueries / Marker brauchst du häufig `GROUP BY corr_keys`. Das sollte **streamingfähig** sein:

- HashAggregate: nimmt rows rein, hält HashMap corr_key → AggState, gibt am Ende oder in windows Ergebnisse aus.

#### C) Persistente Aggregations-Cache-Tabellen (von dir explizit erlaubt)
Für Muster wie:

- `SELECT corr_keys, agg(expr) FROM inner ... GROUP BY corr_keys`

kann der queryplan builder das Ergebnis bereits **persistent** cachen
(set grouptbl (concat "." tbl ":" group))
(createtable schema grouptbl ...)

Wichtig: Der Cache braucht eine **Invalidation-Strategie** (Versioning / MVCC / “table changed” counter).

#### D) Temporäre *Spalten* statt Temp-Tables
Für EXISTS/IN ist es oft genug:
- Outer Tabelle/Stream bekommt eine **berechnete Marker-Spalte** (0/1/NULL) (das soll aber die storage engine entscheiden. wir übergeben nur die condition als lambda und die storage engine entscheidet selbst, ob sie noch Marker-Spalten anlegt, um komplexe Berechnungen z.B. auf Datums-Basis zu cachen und somit Indizes darauf erstellen zu können)
- Diese Marker-Spalte kann “on the fly” berechnet werden (oder als spaltenbasierter Cache)

Damit vermeidet man “Derived Temp Table materialize”.

### 4.3 Sehr empfehlenswert (für echte Performance)
- **Semi-Join Operator** (oder zumindest ein “Filter Join” Modus, der nur Outer-Zeilen durchlässt)
- **Anti-Join Operator**
- **Top-1-per-group** (für scalar subquery mit ORDER BY ... LIMIT 1)  -> scan hat bereits den limit parameter, man muss aber auch bestimmt gruppen-lokale limits und Sortierungen noch unterstützen, um alle Query-Verschachtelungen auflösen zu können
  Alternativ: Streaming-Window oder “min(rank)” trick, aber top-1-per-group ist am effizientesten.

---

## 5) Konkrete Implementationsanleitung (Schritt-für-Schritt)

### Schritt 0: Status quo sichern
Aktuell ist `untangle_query` für `FROM`‑Subselects da (Flattening, Alias‑Prefixing) und **inner_select**‑Ausdrücke sind noch nicht korrekt unterstützt.

### Schritt 1: Scopes sauber modellieren
Du brauchst eine Form von “Scope” / “Environment” während des Rewrites:

- outer scope: Tabellen/Aliase der äußeren Query
- inner scope: Tabellen/Aliase der Subquery

Implementiere Hilfsfunktionen:

- `(scope-has? scope table-alias)`
- `(expr-columns expr) -> list of (alias . col)` (mit Schema‑Lookup für unqualifizierte Spalten)
- `(free-vars expr outer-scope inner-scope)`

### Schritt 2: Subqueries einsammeln und ersetzen
In WHERE/HAVING/SELECT-Liste:
- finde `(inner_select_exists sub)` / `(inner_select_in expr sub)` / `(inner_select sub)`
- erstelle dafür ein “unnest item” mit:
  - typ (exists/in/scalar)
  - subquery AST
  - corr_keys = free vars
  - gewünschtes output: marker/value
- ersetze im Ausdruck den Knoten durch `(get_column <derivedAlias> <marker|valueCol>)`
- umwandeln in ganz normale Tabelle in der Join-Liste
- das kanonische Format für `build_queryplan` muss ggf. umgebaut werden (siehe Schritt 4), um **group‑lokale ORDER/LIMIT** zu unterstützen

**Wichtig:** keine äußeren Spalten in den inneren Subquery‑SELECT injizieren. Korrelation wird über **Key‑Encoding in Map/Reduce** gelöst (siehe Schritt 3).

### Schritt 3: Join-Erweiterung der Outer-Query
Für jedes “unnest item” erweiterst du die FROM/JOIN-Struktur der Outer-Query um einen Join auf einen “derived plan”.

Unter euren Temp-Table-Constraints heißt “derived” nicht zwingend materialisieren:
- im Plan: ein Subplan, der **nested scan** (korrekt) nutzt; später optional batched/scan_multi
- im Code: Query-Struktur, die euer Planbuilder in Map/Reduce codieren kann

**Korrelation im Planbuilder (Sketch):**
- Inner scan mappt auf `(corr_key, payload)`; corr_key wird aus inneren Spalten berechnet
- Reduce aggregiert per corr_key (EXISTS/IN: Marker; scalar: value + cardinality check)
- Outer scan join‑t über corr_key (Left/Semi/Anti per Marker + `isOuter`)

### Schritt 4: Planbuilder‑Interface auf Group‑Stufen umbauen
ORDER/LIMIT pro Gruppe braucht eine **stufige** Repräsentation:
- Ersetze `group/having/order/limit/offset` durch `groups`‑Liste
- Jede Stufe ist eine Liste mit Tags: `((group-cols ...) (having ...) (order ...) (limit ...) (offset ...))`
  - `group-cols` kann leer sein (dann ist es eine reine ORDER/LIMIT‑Stufe)
  - ORDER/LIMIT‑Stufen sollten **als letzte Stufe** kommen
- `build_queryplan` baut pro Stufe eine Scan/Aggregation‑Schicht und verkettet Stufen

Das Refactoring sollte zuerst passieren, bevor Unnesting “scharf” gemacht wird.

### Schritt 5: Batched Keyset (später)
`scan_multi`/Keyset‑Join kann später ergänzt werden, sobald die Korrektheit über nested scans steht.

### Schritt 6: Aggregations-Cache (optional, aber passend zu eurer Vorgabe)
Wenn ein innerer Subplan `GROUP BY corr_keys` erzeugt:
- prüfe Cache: existiert “signature” bereits?
- wenn ja: nutze cached results
- wenn nein: baue/calc und speichere

Signature sollte enthalten:
- Tabellen + Filter + Join-Graph (oder hash davon)
- corr_key cols
- Agg-Liste
- optional: versions der betroffenen Tabellen

---

## 6) Test und Evaluierung

make test

es gibt bereits inner select test cases, die mit unimportant markiert sind. Diese kann man wenn alles geht scharf schalten.


---

## 7) Wie ihr ohne Temp-Tables wirklich schnell werdet: `scan_multi` / Batch-Keyset

### 7.1 Operatoridee: `BatchedKeysetJoin`
Ein Join-Operator, der so arbeitet:

1. Outer liefert tuples.
2. Aus Outer werden corr_keys extrahiert und in einem Batch gesammelt (z.B. 1024 Keys).
3. Inner wird **einmal pro Batch** ausgeführt, aber mit Keyset-Filter:
   - `scan_multi(inner_table, keyset, join_predicates)`
4. Optional: inner Ergebnisse werden `GROUP BY corr_keys` aggregiert (HashAgg)
5. Dann wird das Ergebnis mit den Outer tuples gematched und weitergestreamt.

So entsteht die “Domain D” implizit als **Keyset im Operator**, nicht als Temp-Table.


### Warum das eure Constraints erfüllt
- Kein Materialize einer Derived table nötig
- Nur:
  - berechnete Marker/Value-Spalten im Stream
  - optionale persistente Aggregations-Cache-Tables

---

## Praktischer Einstieg (Empfehlung)

1. **Phase 1 (korrekt, schnell integrierbar):**
   - Rewrite EXISTS → Left Join + Marker
   - Rewrite scalar → Left Join + GroupBy(MIN)
   - Ausführung erstmal über eure bestehenden nested loops (korrekt, aber nicht maximal schnell)

2. **Phase 2 (Performance ohne Temp-Tables):**
   - BatchedKeysetJoin + scan_multi
   - HashAggregate on inner side
   - Semi-Join Operator (oder Filter-Join Mode)

3. **Phase 3 (Caching):**
   - persistente Agg-Caches für häufige GroupBy-Subqueries
   - Invalidation via table versions

---

## Anmerkung zu “fast ohne Temp-Tables”
Die klassische Literatur-Implementierung materialisiert oft D oder derived results. Unter euren Vorgaben ersetzt ihr das durch:

- **Operator-internes Keyset (D)**
- **Batched inner evaluation**
- **Marker/Value als temporäre Spalten im Stream**
- **optionaler persistenter Agg-Cache** statt “ad hoc temp table”

Damit bleibt ihr nah am theoretischen Unnesting-Prinzip, ohne Temp-Table-Orgie.

Dadurch, dass die Materialisierung sowieso im RAM passiert, und die Benennung über gruppierte temp-tables und temp-spalten kanonisch erfolgt, können die temp-caches ggf. query-übergreifend wiederverwendet werden

---

## Offene Punkte (Stand jetzt)
- NULL‑Semantik für `IN`/`NOT IN` und `EXISTS` definieren (Marker vs. tri‑valued)
- Fehlerbehandlung bei scalar subquery mit >1 Row (sollte hart failen)
- Invalidation/Versioning für Aggregations‑Caches klären
- Gruppen‑Stufen‑Refactor konkretisieren (Format + Übergangsstrategie)

*Ende.*
