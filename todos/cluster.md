# MemCP Cluster – Kochbuch

Dieses Dokument beschreibt **das Node-Cluster-Feature von MemCP** als umsetzbares Kochbuch.
Es fasst **alle getroffenen Annahmen, Designentscheidungen und Protokolle** zusammen und erweitert sie
um konkrete Implementierungsleitlinien.

Umgesetzt wird ein **MOESI-basiertes Cache-Kohärenz-Protokoll** (Modified-Exclusive-Shared-Invalid als Cache-Zustände, mit einem per CRUSH zugewiesenen **Directory-Knoten** als Koordinator). Ownership wird **deterministisch per CRUSH-Algorithmus** auf Basis der Shard-UUID vergeben. Der Directory-Knoten eines Shards ist die **Single Source of Truth** für dessen Load-State – er muss den Shard nicht selbst geladen haben. Damit bei 128 GB RAM pro Node bis zu 10 Billionen Datenbank-Einträge für einen Cluster von 207 Nodes möglich, im Single-Node Verfahren passen ca. 80 Milliarden Items auf die Node.

Ziel:
- **Kein ZooKeeper, kein Raft, keine zusätzliche Infrastruktur**
- **Leaderless**
- **Shared-Read / Exclusive-Write** für **Shards und Tables**
- **Millionen Shards skalierbar**
- **RADOS als einzige persistente Wahrheit**
- **CRUSH-basierte deterministische Directory-Zuordnung** (kein Gossip nötig)
- **Zero-Config-artiger Docker-Betrieb**
- **Ein einzelnes MemCP-Binary**

---

## Vorbedingungen

anstatt einer zentralen Datei schema.json muss die Persistenzschicht abgeändert werden:
 - pro Tabelle ein unterordner <tblname>/ (im Dateien-Backend)
 - beim Laden werden alle ordner geladen und die darin enthaltene schema.json deserialisiert, daraus ergibt sich die Tabellenliste
 - statt schema setzen -> 2 Operationen write table schema (create+update), remove-table
 - die Shards befinden sich dann auch im Unterordner
 - Migrationslogik: will man eine datenbank laden, die noch eine schema.json enthalten, wird einmalig umstrukturiert

---

## 1. Grundannahmen

### 1.1 Betriebsumgebung
- Alle Nodes laufen im **selben Rechenzentrum**
- Netzwerksplits gelten als ausgeschlossen: bei einem Ausfall **warten** die anderen Nodes, bis die fehlende Node wieder erreichbar ist
- Ist eine Node **zu lange offline** (konfigurierbar), wird sie gezielt **isoliert** und der CRUSH-Algorithmus **rebalanced** – ihre Directory-Zuständigkeiten und Lock-States werden auf die verbleibenden Nodes verteilt
- Beim **Reconnect** registriert sich die Node **neu** im Cluster und bekommt ihren CRUSH-Anteil wieder zugewiesen
- RADOS ist hochverfügbar und erreichbar

### 1.2 Start & Bootstrap
- MemCP wird gestartet mit:

```bash
memcp -join <ip> -secret <secret> -rados* (rados related login data)
```

- Wird `-join` **nicht** angegeben:
  - Node startet im **Standalone-Modus**
  - weiß, dass sie die einzige Node ist
- Wird `-join <ip>` angegeben:
  - Node verbindet sich zu einem bestehenden Peer
  - erhält die vollständige Cluster-Membership (Node-Liste)
  - berechnet per CRUSH lokal, welche Directory-Zuständigkeiten sie übernimmt

RADOS-Zugangsdaten werden **direkt beim Start** (Env/Flags) übergeben.
Kein späterer Konfig-Austausch nötig.

---

## 2. Cluster-Membership

Jede Node hält lokal:

- Liste aller bekannten Nodes (Membership-View)
- aktuelle Membership-View-Version

Die Membership-View wird bei Join/Leave/Isolation aktualisiert und an alle Nodes verteilt. Daraus berechnet jede Node per CRUSH lokal die Directory-Zuordnungen – **kein Gossip nötig**, da CRUSH deterministisch ist und nur die Membership-View als Input braucht.

---

## 3. Zwei-Ebenen-Modell: Directory-State & Cache-State

### 3.1 CRUSH-basierte Directory-Zuordnung
**Jeder Shard (UUID) und jede Table (db+table) hat genau einen Directory-Knoten**, berechnet per CRUSH-Algorithmus. Jede Node kann den zuständigen Directory-Knoten lokal berechnen – keine Netzwerk-Abfrage nötig.

Der Directory-Knoten **muss den Shard nicht selbst geladen haben**. Er ist ausschließlich die **Koordinationsinstanz**:
- Er weiß, welche Nodes den Shard aktuell im Cache halten und in welchem Zustand
- **Invalidate-Anfragen** gehen an den Directory-Knoten, der dann **gezielte Invalidate-Nachrichten** nur an die betroffenen Nodes verschickt
- **Exclusive-Lock-Anfragen** werden beim Directory-Knoten gestellt

### 3.2 Directory-State (was der CRUSH-Knoten trackt)
Der Directory-Knoten pflegt **nur für aktiv geladene Shards** eine Tracking-Struktur:

| Feld | Bedeutung |
|------|-----------|
| **Sharers-Liste** | Welche Nodes aktuell eine SHARED-Kopie halten |
| **Exclusive-Holder** | Welche Node (falls vorhanden) MODIFIED oder EXCLUSIVE hält |
| **Lock-Queue** | Ausstehende EXCLUSIVE-Anfragen (FIFO) |

Ein Shard, der **nirgends geladen** ist, taucht in keiner Tracking-Struktur auf und belegt **keinen Speicher** – weder auf irgendeiner Node noch beim Directory-Knoten. Erst beim ersten `ReqShared` oder `ReqExclusive` wird ein Eintrag angelegt.

### 3.3 Cache-State (pro Node pro Shard)
Jede Node kennt für ihre lokal gehaltenen Shards den eigenen Cache-Zustand:

| Zustand | Bedeutung |
|---------|-----------|
| **Modified (M)** | Node hat exklusiven Schreibzugriff, Daten sind dirty (noch nicht in RADOS persistiert) |
| **Exclusive (E)** | Node hat exklusiven Zugriff, Daten sind clean (konsistent mit RADOS) |
| **Shared (S)** | Node hält read-only Kopie, andere Nodes können ebenfalls SHARED halten |
| **Invalid (I)** | Shard ist nicht im lokalen Cache – **belegt keinen Speicher**, kein Eintrag nötig |

### 3.4 Zusammenspiel
- Directory-State ist die **globale Sicht** des CRUSH-Knotens: wer hat was
- Cache-State ist der **lokale Zustand** jeder einzelnen Node: was halte ich
- Übergänge (z.B. S→E oder E→M) werden immer über den Directory-Knoten koordiniert
- Der Directory-Knoten selbst kann gleichzeitig auch Cacher sein (dann hat er sowohl Directory-State als auch eigenen Cache-State)

---

## 4. Tables & Shards: Lock-Modell

### 4.1 Lock-Typen
Für **Tables und Shards** gilt identisch:

- `SHARED` (Read)
- `EXCLUSIVE` (Write)
kein NONE-Zustand, dann ist das objekt einfach nicht in der liste

### 4.2 Lokaler Lock-Manager
Jede Node implementiert lokal:
- Re-entrant shared reads
- Re-entrant exclusive writes
- lokale RefCounts (rein lokal, nicht distributed)

Diese Logik existiert bereits.

---

## 6. Shared-Read Ablauf

### 6.1 Fast Path (lokal)
- Node hält bereits `SHARED` oder `EXCLUSIVE` → lokalen readlock betreten → fertig

### 6.2 Remote Path
- Node berechnet Directory-Knoten per CRUSH
- Anfrage an Directory-Knoten:
  - `ReqShared(O)`
- Directory-Knoten prüft Directory-State:
  - Falls EXCLUSIVE/MODIFIED aktiv: wartet bis Holder seinen Writelock beendet und downgradet auf SHARED
  - Directory-Knoten trägt anfragende Node in Sharers-Liste ein
  - Antwort mit aktuellem Load-State
- die Node liest danach schema.json der Tabelle oder den Shard-Inhalt aus dem RADOS oder bekommt sie gleich als Inhalt mitgesendet

### 6.3 Fallback
- Falls niemand das Objekt hält:
  - Load aus RADOS
  - Cache lokal als SHARED
  - Directory-Knoten wird über neuen Sharer informiert

---

## 7. Exclusive-Write Ablauf

### 7.1 Motivation
EXCLUSIVE wird benötigt für:
- Delta-Append
- Delete-Marker
- Table-Strukturänderungen (Shardlisten, Migration)

### 7.2 Ablauf
1. Node berechnet Directory-Knoten per CRUSH
2. Node sendet **ReqExclusive(O)** an den Directory-Knoten
3. Directory-Knoten prüft Directory-State:
   - Falls SHARED: Directory-Knoten sendet **gezielte Invalidate(O)** nur an die Nodes in der Sharers-Liste
   - Falls EXCLUSIVE/MODIFIED durch andere Node: Anfrage wird in Lock-Queue eingereiht
4. Nodes bestätigen Invalidate → droppen lokale Caches (Cache-State → Invalid)
5. Directory-Knoten erteilt EXCLUSIVE-Lock an anfragende Node
6. Node beginnt EXCLUSIVE-Arbeit (Cache-State → Exclusive, bei Writes → Modified)

Vorteil gegenüber Broadcast: bei Millionen Shards werden nur die tatsächlich betroffenen Nodes kontaktiert.

---

## 9. Tables: Shardlisten & Migration (ist aber schon implementiert)

### 9.1 Zwei Shardlisten
Eine Table enthält:
- `Shards[]`
- `PShards[]`

Nur während Migration sind beide aktiv.

### 9.2 PShards
- multidimensional partitioniert
- z. B. ID x Timestamp x Name
- logisch vollständig, physisch lazy materialisiert
- dünn besiedelte Partitionen sind erlaubt

### 9.3 Table-Updates
Table-Strukturen sind Go-Datenstrukturen und werden:
- bei SHARED auf mehreren Nodes vorhanden
- bei Updates wahlweise per **Patch/Broadcast** verteilt oder aus dem rados geladen

Keine Chunking-Listen.
Kein vollständiges Re-Upload bei Single-Shard-Änderung.

### 9.4 Konsistenz
Statemachine stellt sicher:
- niemals zwei gleichzeitige EXCLUSIVE-Writer
- niemals inkonsistente Leser (während des shared-read ist das objekt readonly)

---

## 10. Shards: Main / Delta / DeleteMarker

### 10.1 Storage-Modell
- Main: kompakt
- Delta: append-only
- DeleteMarker: append-only

### 10.2 Writes
- EXCLUSIVE nur für Delta/Delete-Append
- Rebuild darf parallel zu Reads laufen -> atomarer swap der shard-referenz in der Shard-Liste in Form eines Netzwerk-Broadcast

### 10.3 Rebuild
- Main + Delta + Deletes werden gemerged
- neue Shard entsteht mit neuer uuid
- **Shard-Swap ist semantisch neutral**
- Publish per Nachricht, warten auf ACK von allen
- nach Abschluss kann die alte Shard aus dem RADOS gelöscht werden

Leser dürfen vorher und nachher lesen.

---

## 11. Query Routing (leaderless)

- Jede Node kann Query-Router sein
- Ablauf:
  1. Query an beliebige Node
  2. Queryplan bauen wie gehabt, die Node führt ihn auch aus
  3. Table laden (SHARED) sobald man darauf zugreift, z.b. mit (show)
  4. bei Scan: nicht geownte Shards -> wenn nicht verfügbar: selbst laden falls RAM da ist, sonst eine andere Node beauftragen, wenn jemand anderes shared reader ist -> ihm den Scan-Job schicken
  5. Antworten über Reply-Token zurück

Kein zentraler Router.
Kein Leader.

---

## 12. Database-Ownership & Lastverteilung

### 12.1 Database-Ownership-Tabelle
In der **System-Datenbank** wird eine dynamisch balancierte Zuordnungstabelle gepflegt:

```
system.database_ownership (
  database_name  TEXT PRIMARY KEY,
  favorite_node  TEXT,        -- Node-ID der bevorzugten Node
  last_updated   TIMESTAMP
)
```

Pro Datenbank kann eine **Favoriten-Node** hinterlegt werden. Diese Zuordnung dient der **Lastverteilung**: Queries für eine bestimmte Datenbank werden bevorzugt an die Favoriten-Node geroutet, die wahrscheinlich bereits die relevanten Shards im Cache hält.

### 12.2 REST-Frontend: 301/302 Redirects
Trifft ein HTTP-Request auf eine Node, die **nicht** die Favoriten-Node für die angefragte Datenbank ist:
- Antwort mit **301 (Moved Permanently)** oder **302 (Found)** auf die korrekte Node
- Client folgt dem Redirect und spricht künftig direkt mit der zuständigen Node

### 12.3 MySQL-Frontend: Request-Tunneling
Das MySQL-Protokoll unterstützt keine Redirects. Stattdessen:
- Node empfängt Query über MySQL-Verbindung
- Erkennt, dass eine andere Node Favorit ist
- **Tunnelt den Request** transparent an die Favoriten-Node weiter
- Gibt das Ergebnis an den Client zurück, als käme es von der lokalen Node

### 12.4 Dynamische Rebalancierung
Die Ownership-Tabelle wird periodisch angepasst basierend auf:
- Aktuelle Last pro Node (CPU, RAM, offene Verbindungen)
- Cache-Hitrate pro Datenbank pro Node
- Manuelle Overrides durch den Admin

---

## 13. Failure-Verhalten

### 13.1 Node-Ausfall (temporär)
- Node wird unerreichbar → andere Nodes **warten**
- Laufende Requests, die Shards dieser Node betreffen, blockieren (Timeout-basiert)
- Shards, für die die ausgefallene Node Directory-Knoten ist, sind temporär nicht koordinierbar
- Shards, die nur auf der ausgefallenen Node gecacht waren, sind temporär nicht lesbar (Fallback: RADOS-Load auf anderer Node)

### 13.2 Node-Ausfall (dauerhaft / Isolation)
- Überschreitet die Offline-Dauer einen konfigurierbaren Schwellwert:
  1. Node wird aus der **Membership-View entfernt** (Isolation)
  2. Neue Membership-View-Version wird an alle verbleibenden Nodes verteilt
  3. **CRUSH rebalanced**: Directory-Zuständigkeiten der isolierten Node werden auf verbleibende Nodes umverteilt
  4. Neue Directory-Knoten rekonstruieren Directory-State: fragen alle Nodes nach ihrem lokalen Cache-State für die betroffenen Shards, oder nutzen RADOS als Fallback
- Keine automatische Shard-Wanderung – nur die Directory-Koordination wird umverteilt

### 13.3 Reconnect nach Isolation
- Node startet effektiv **neu** im Cluster (wie ein frischer Join)
- Registriert sich über `-join` bei einem bestehenden Peer
- Erhält aktuelle Membership-View → CRUSH berechnet neue Zuständigkeiten
- Node bekommt ihren CRUSH-Anteil an Directory-Zuständigkeiten zurück
- Lokale Caches sind leer (Invalid) – werden bei Bedarf aus RADOS nachgeladen

### 13.4 Netzprobleme
- Keine RADOS-Verbindung → **keine Writes**
- Option: Prozess beendet sich selbst

---

## 14. Warum dieses Design skaliert

- keine globalen Locks
- keine Broadcasts für Invalidierung (gezielte Nachrichten über Directory-Knoten)
- keine Zusatzsysteme (kein Gossip, kein Raft, kein ZooKeeper)
- deterministische Directory-Zuordnung per CRUSH (kein Netzwerk-Lookup nötig)

**Komplexität ist lokal, nicht verteilt.**

---

## 15. Distributed Scan: Analyse & Implementierungsplan

### 15.1 Ist-Zustand der Scan-Pipeline

Die aktuelle Scan-Architektur (`storage/scan.go`, `storage/scan_order.go`) ist **bereits zweiphasig** und damit grundsätzlich verteilbar:

**scan() – ungeordnet** (`scan.go:45`):
- Phase 1 (shard-lokal, parallel): `reduce(neutral, map(row))` pro Shard → ein `scanResult` pro Shard
- Phase 2 (Shard-Collect, seriell): `reduce2(neutral, shard_result)` → finales Ergebnis
- Ergebnis pro Shard ist ein **einzelner Scmer-Wert** (Akkumulator) + Zähler → wenige Bytes bis KB

**scan_order() – geordnet** (`scan_order.go:104`):
- Jeder Shard filtert + sortiert lokal → `shardqueue` mit sortierten Item-Indizes
- Globaler Merge via `container/heap` (Priority Queue) → OFFSET/LIMIT seriell
- `shardqueue.mcols`/`scols` sind **Closures über Shard-Speicher** (Column-Reader)

**Aggregation** (`lib/queryplan.scm:1028`):
- GROUP BY in drei Phasen: Collect (unique keys) → Compute (pro Aggregat ein separater Scan) → Output
- `parallel`-Combinator führt alle Aggregate-Scans gleichzeitig aus
- Jedes Aggregat ist ein eigener `scan()`-Call mit eigenem reduce/reduce2

### 15.2 Was gut passt (geringer Aufwand)

**scan() mit reduce/reduce2 für Aggregationen:**
- Die Zwei-Phasen-Architektur bildet direkt auf Remote-Execution ab:
  - Coordinator schickt `(filter, map, reduce, neutral)` als Scheme-AST an Remote-Node
  - Remote-Node führt Phase 1 auf ihren Shards aus, liefert `scanResult` zurück
  - Coordinator führt Phase 2 (`reduce2`) über alle Remote-Ergebnisse
- `scanResult` ist trivial serialisierbar: ein Scmer-Wert + zwei int64-Zähler
- **Overhead pro Remote-Shard**: Framing + Scmer-Serialisierung (~Bytes bis wenige KB bei Aggregaten)
- Verglichen mit dem RADOS-Load eines ganzen Shards: **vernachlässigbar**

**Einfach verteilbare Aggregate** (SUM, COUNT, MIN, MAX):
- reduce und reduce2 sind identisch (assoziativ + kommutativ)
- Shard-Ergebnis = finales Teilergebnis → direkt zusammenführbar

### 15.3 Serialisierung von Filter/Map/Reduce-Lambdas (Code-Review)

Für Remote-Scans müssen die Lambdas (filter, map, reduce) über das Netzwerk übertragen werden. Der Code-Review von `scm/scmer.go` zeigt **mehrere Probleme**:

#### 15.3.1 Proc-Serialisierung (`scmer.go:807-816`)
`MarshalJSON` serialisiert `Proc` als `[{"symbol":"lambda"}, params, body, numVars?]`.
- **Params und Body** werden rekursiv serialisiert → grundsätzlich OK
- **Env (Closure-Umgebung) wird NICHT serialisiert** → bei `UnmarshalJSON` wird immer `Globalenv` eingesetzt (`scmer.go:954`)
- Für **reine Lambdas** (keine freien Variablen, nur Params + Builtins) ist das korrekt – Filter/Map/Reduce vom Queryplanner sind typischerweise rein
- Für **Closures** (z.B. GROUP BY compute_plan mit `(outer col)`) bricht die Serialisierung

#### 15.3.2 NthLocalVar-Bug (`scmer.go` – fehlt in MarshalJSON!)
Nach der Optimierung (`scm/optimizer.go`) werden Parameter-Referenzen im Body durch `NthLocalVar(idx)` ersetzt (Tag `tagNthLocalVar`). **Dieses Tag hat keinen Case in `MarshalJSON`** (Zeile 757-822):
- Fällt in den `default`-Case → `v.String()` → `"<custom 9>"` → **Roundtrip kaputt**
- **Alle optimierten Procs sind damit nicht serialisierbar**

**Fix benötigt:** `MarshalJSON`/`UnmarshalJSON` für `NthLocalVar`:
```go
// MarshalJSON:
case tagNthLocalVar:
    return map[string]any{"var": int(v.NthLocalVar())}

// UnmarshalJSON:
case map[string]any:
    if idx, ok := t["var"]; ok {  // NthLocalVar
        return NewNthLocalVar(NthLocalVar(idx.(int)))
    }
```

#### 15.3.3 Native Builtins (`scmer.go:798-804`)
`tagFunc` serialisiert als `{"symbol": "name"}` via `DeclarationForValue()`. Deserialisiert wird es als Symbol, das auf der Remote-Seite in `Globalenv` aufgelöst werden muss.
- **Funktioniert** für Standard-Builtins (`equal?`, `<`, `>`, `+`, `and`, `scan`, etc.) – diese existieren auf jeder Node
- **Bricht** für `tagFuncEnv` → serialisiert als `{"symbol": "?"}` → verloren. Sollte in Queryplan-Lambdas nicht vorkommen.

#### 15.3.4 Empfohlene Strategie
1. **NthLocalVar-Serialisierung fixen** (kleiner Patch in `scmer.go`) – Voraussetzung für alles
2. Filter/Map/Reduce als **serialisierte Proc** (JSON oder binär) übertragen
3. Remote-Node deserialisiert → Proc mit `Globalenv` → `OptimizeProcToSerialFunction` lokal aufrufen
4. Alternativ: den **Scheme-AST-Quelltext** (vor eval) übertragen und auf der Remote-Seite eval'n – vermeidet NthLocalVar-Problem komplett, erfordert aber Zugriff auf den AST im Queryplanner

### 15.4 Was fehlt: Scan-Delegation (Remote Scan RPC)

**Aktuell:** `iterateShards()` (`partition.go:53`) iteriert nur lokale Shards.

**Benötigt:** Entscheidungslogik im Scan:
```
für jeden Shard der Tabelle:
  - lokal geladen?     → lokaler Scan (wie bisher)
  - remote geladen?    → ScanRPC an die Node, die SHARED/EXCLUSIVE hält
  - nirgends geladen?  → selbst laden (falls RAM frei) ODER an andere Node delegieren
```

**Scan-RPC Protokoll:**
```
ScanRequest {
  schema, table      string
  shard_uuids        []UUID        // welche Shards zu scannen
  filter_ast         Scmer         // serialisierte Filter-Lambda (Proc oder AST)
  filter_cols        []string
  map_ast            Scmer         // serialisierte Map-Lambda
  map_cols           []string
  reduce_ast         Scmer         // Phase-1 Reduce
  neutral            Scmer
  // kein reduce2 – der Coordinator macht Phase 2
}

ScanResponse {
  results []struct {
    shard_uuid  UUID
    result      Scmer   // Phase-1 Akkumulator
    out_count   int64
    input_count int64
  }
}
```

### 15.5 Was fehlt: Binäre Scmer-Serialisierung

**Problem:** `MarshalJSON` ist textbasiert und suboptimal für High-Throughput:
- JSON-Overhead bei numerischen Werten (int64 als Dezimalstring)
- Kein Streaming-Support (kein Length-Prefix)

**Benötigt:** Kompaktes binäres Scmer-Wire-Format:
- Tag-Byte (Typ) + Payload (Little-Endian)
- Nil=0, Bool=1, Int=2 (8 Byte), Float=3 (8 Byte), String=4 (varint-len + bytes), Symbol=5, List=6 (varint-len + Elemente), Proc=7 (Params + Body als verschachteltes Scmer), NthLocalVar=8 (1 Byte Index), Func=9 (varint-len + Name)
- Für Aggregate-Ergebnisse typischerweise <100 Bytes pro Shard
- Length-Prefixed Framing für TCP-Streams

### 15.6 Overhead-Analyse: Distributed Aggregation

**Bester Fall – einfache Aggregate (SUM/COUNT/MIN/MAX):**
- 1 Scan-RPC pro Remote-Node (alle Shards einer Node gebündelt)
- Response: 1 Scmer-Wert pro Shard → ~16-64 Bytes
- Netzwerk-Overhead: **vernachlässigbar** gegenüber Scan-Rechenzeit

**Problematischer Fall – GROUP BY mit N Aggregaten:**
- Aktuell: Collect-Scan (1x) + N Compute-Scans (je 1 voller Tabellen-Scan)
- Bei M Remote-Nodes: **N × M Scan-RPCs** für die Compute-Phase
- Jeder Compute-Scan filtert auf `group_key AND WHERE` → nochmal voller Scan pro Aggregat

**Optimierung: Batched-Aggregate-Push-Down:**
- Statt N separate Scans: **einen einzigen Remote-Scan** mit allen N Aggregaten als kombiniertem reduce
- Remote-Node liefert Dict `{group_key → [agg1, agg2, ..., aggN]}` zurück
- Coordinator merged die Teil-Dicts
- Reduziert RPCs von N×M auf **1×M**
- **Erfordert Änderung in `queryplan.scm`:** der `compute_plan` muss alle Aggregate in einen Scan fusionieren können, statt `(cons 'parallel (map ags ...))` mit separaten Scans

### 15.7 Fehlende Funktionen: scan_order distributed

**Problem:** `scan_order` hält `shardqueue`-Closures über lokalen Shard-Speicher (`scan_order.go:28-35`).
Die `mcols`/`scols`-Felder sind `func(uint) Scmer`-Reader, die direkt auf Column-Speicher zugreifen.
Für Remote-Shards geht das nicht – die Daten liegen auf einer anderen Node.

**Ablauf beim Merge:** Die `globalqueue` (`scan_order.go:66`) zieht immer das kleinste Element aus dem Heap. Items, die zum selben Shard gehören, kommen **aufeinanderfolgend** aus derselben `shardqueue`. Erst wenn das nächste Shard-Element nicht mehr das global-kleinste ist, wechselt der Heap zu einem anderen Shard.

**Schlüsselidee:** Items pro Shard batchen – so lange Items pullen, wie sie zum selben Shard gehören. Dann den Batch an die Remote-Node schicken.

**Variante A – IDs + map() an den Shard schicken (favorisiert):**
- Coordinator sammelt Item-IDs (Shard-interne Indizes) aus der `shardqueue`
- Schickt `{shard_uuid, item_ids[], map_ast}` an die Remote-Node
- Remote-Node führt `map()` lokal aus – **Daten-Lokalität bleibt erhalten**, keine unnötige Spaltenübertragung
- Remote-Node liefert `map()`-Ergebnisse zurück (nur die projizierten Werte)
- **Vorteil:** Minimaler Netzwerk-Transfer, map() kann Indexes und Column-Kompression direkt nutzen
- **Nachteil:** Mehr Roundtrips (pro Batch ein RPC)

**Variante B – IDs + Spaltenliste an den Shard schicken:**
- Coordinator sammelt Item-IDs aus der `shardqueue`
- Schickt `{shard_uuid, item_ids[], column_names[]}` an die Remote-Node
- Remote-Node liefert die rohen Spaltenwerte zurück
- Coordinator führt `map()` lokal aus
- **Vorteil:** map()-Logik bleibt beim Coordinator, einfacheres Debugging
- **Nachteil:** Mehr Daten über das Netzwerk (alle angefragten Spalten, nicht nur map()-Output)

**Empfehlung:** Variante A – Daten-Lokalität ist bei großen Shards und komplexen map()-Funktionen der dominierende Faktor. Die map()-Funktion muss ohnehin serialisierbar sein (siehe 15.3), und auf der Remote-Seite kann `OptimizeProcToSerialFunction` die volle Column-Reader-Performance nutzen.

**Für die Filter+Sort-Phase** (die bereits in `scan_order` vor dem Merge passiert): hier bleibt der bestehende Ansatz – jeder Shard filtert und sortiert lokal. Für Remote-Shards wird dies über den Scan-RPC (15.4) abgedeckt: die Remote-Node liefert sortierte Item-IDs zurück, die dann in die `globalqueue` eingereiht werden.

### 15.8 Fehlende Funktionen: Zusammengesetzte Aggregate

Einige SQL-Aggregate sind **nicht trivial verteilbar:**

| Aggregat | Verteilbar? | Strategie |
|----------|-------------|-----------|
| COUNT    | trivial     | SUM der Teil-COUNTs |
| SUM      | trivial     | SUM der Teil-SUMs |
| MIN/MAX  | trivial     | MIN/MAX der Teil-Ergebnisse |
| AVG      | **nein**    | → SUM + COUNT getrennt übertragen, Coordinator teilt |
| STDDEV   | **nein**    | → SUM, SUM_SQ, COUNT getrennt |
| GROUP_CONCAT | **bedingt** | Teillisten konkatenieren, Sortierung ggf. nochmal global |
| MEDIAN / Percentile | **nein** | Erfordert vollständige Wertliste oder approximative Algorithmen (t-digest) |

Da `scan()` die Zwei-Phasen-Architektur (reduce/reduce2) bereits transparent handhabt, muss der Queryplanner für einfache Aggregate **nichts ändern** – die Verteilung passiert unterhalb der scan-Ebene. Nur für nicht-triviale Aggregate (AVG, STDDEV) muss der Queryplanner ggf. die Zerlegung in Komponenten kennen (z.B. AVG → SUM + COUNT getrennt scannen, Coordinator teilt).

### 15.9 Zusammenfassung: Prioritäten

1. ~~**NthLocalVar-Serialisierung fixen**~~ – **erledigt** (`scmer.go`: MarshalJSON/UnmarshalJSON für `tagNthLocalVar` als `{"var": idx}`)
2. **Binäres Scmer-Wire-Format** – Voraussetzung für effizienten Transfer
3. **Scan-RPC für ungeordnete Scans** – direkter Cluster-Nutzen, passt in bestehende Zwei-Phasen-Architektur
4. **Batched-Aggregate-Push-Down** – größter Performance-Gewinn für den häufigsten Workload (GROUP BY)
5. **Scan-Delegation in iterateShards** – Integration in bestehende Scan-Orchestrierung
6. **Materialisierte scan_order für Remote-Shards** – für ORDER BY + LIMIT Queries
7. **Dekomponierbare Aggregate** – für korrekte verteilte AVG, STDDEV etc.
8. **Streaming scan_order** – Optimierung für große ORDER BY ohne LIMIT (spät)

---

## 16. TODO: Shared-Exclusive Zugriff (INSERT/DELETE-Forwards)

> **Spätere Erweiterung – noch nicht Teil der initialen Implementierung.**

Ziel: Ein **Shared-Exclusive (SE)**-Zugriffsmodus, bei dem eine Node exklusiv schreibt, während andere Nodes ihre read-only Kopien behalten und Updates gestreamt bekommen.

### 16.1 Konzept
- Eine Node hält **MODIFIED/EXCLUSIVE** für Writes (INSERT/DELETE)
- Andere Nodes behalten ihre **SHARED**-Kopien für schnelle Reads
- Der Writer **streamt Änderungen** (Delta-Appends, Delete-Marker) an alle SHARED-Holder
- SHARED-Holder wenden die gestreamten Änderungen lokal an, ohne den Shard aus dem RADOS neu laden zu müssen

### 16.2 Vorteile
- **Kein Cache-Miss nach Writes**: Leser müssen ihre Kopien nicht droppen und neu laden
- **Geringere RADOS-Last**: weniger Lese-Operationen nach Schreibvorgängen
- **Niedrigere Latenz**: Reads können während laufender Writes weiterlaufen

### 16.3 Offene Fragen
- Wie werden gestreamte Updates atomar angewendet? (Transaktionsgrenzen)
- Wie wird mit Streaming-Lag umgegangen? (Consistency-Garantien)
- Wann ist ein Fallback auf vollständiges Invalidate sinnvoll? (z.B. bei großen Batch-Writes)

---

## 17. Zusammenfassung

MemCP Cluster ist:

- **leaderless**
- **low-infra** (kein Gossip, kein Raft, kein ZooKeeper)
- **deterministisch (CRUSH-basierte Directory-Zuordnung)**
- **auf Millionen Shards skalierbar**
- **mit klaren Failure-Eigenschaften** (warten → isolieren → CRUSH-rebalance → reconnect)
- **effizient durch gezielte Invalidierung statt Broadcast**
- **saubere Trennung: Directory-State (CRUSH-Knoten) vs. Cache-State (jede Node)**

> Korrektheit kommt aus starker Koordination über deterministische Directory-Zuordnung,
> Skalierbarkeit aus gezielter Kommunikation statt Broadcasts.

---
