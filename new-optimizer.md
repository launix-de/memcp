<!--
Copyright (C) 2026 Carli2

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
-->

# Neuer Optimizer: allokationsarme Planner-Compilation

## Zuständigkeitsgrenze

Der Optimizer entscheidet, **welche Berechnung** ausgeführt wird und welche
Werte temporär, konstant, eindeutig besessen oder flüchtig sind. Alle
Transformationen, die Interpreter-Ausführung und Compilation gemeinsam
beschleunigen, gehören hierher. Dazu zählen konstante Teilbäume,
Escape/Lebensdauer, `!list`/`!!list`, Listenfusion, bekannte Aufrufziele und die
algorithmische Darstellung der Reorder-Mengen. Nachgelagerte Backends müssen
diese Entscheidungen nur noch respektieren.

## Bereits vorhandene Grundlage

Der Go-Optimizer besitzt schon die wesentlichen Bausteine:

- Typ-, Längen- und Ownership-Informationen in `TypeInfo` und
  `TypeDescriptor`;
- `NoEscape`, `FreshAlloc` und Ownership-Transfer für sichere `_mut`-Rewrites;
- `!list` als flüchtige Liste in den nummerierten Lambda-Slots;
- `!!list` als vorreservierter, zunächst leerer Slice mit Kapazität;
- Constant Folding;
- Fusionen für `map`, `filter`, `filter_map`, `flat_map` und mutierende
  Varianten;
- Rückübersetzung interner Formen durch `DeoptimizeExpr`.

Die nächste Arbeit sollte diese vorhandene Sprache und ihren Analysevertrag
erweitern, statt dieselben Entscheidungen in späteren Phasen erneut abzuleiten.

## Zielarchitektur: ein Bottom-up-Pass mit strukturierter `TypeInfo`

Der zentrale Vertrag bleibt `OptimizeEx(expr) -> (expr, TypeInfo)`. Jedes Kind
wird genau einmal im semantisch erforderlichen Kontext optimiert. Der
Elternknoten erhält die optimierten Kinder samt vollständiger `TypeInfo`, wendet
seine Deklaration beziehungsweise Peephole-Regel an und liefert wiederum
Ausdruck und `TypeInfo`. Es gibt keinen zweiten Analysebaum, keinen nachträglich
über den AST laufenden Typ-Pass und keinen separaten Rewrite-Optimizer.

`TypeInfo` muss dafür das bislang nur teilweise migrierte Typsystem vollständig
abbilden:

- Art und Form des Werts einschließlich exakter Länge oder Längengrenzen;
- Ownership, Transfer und Escape für den ganzen Wert sowie rekursiv für
  Listenelemente und Assoc-Keys;
- vollständig bekannte Konstanten sowie teilweise und keyweise bekannte Werte;
- bei Lambdas Parameter-, Capture-, Return- und Escape-Informationen sowie eine
  kompakte Transferfunktion für vom Parametertyp abhängige Resultate;
- Reinheit, beobachtbare Effekte und gegebenenfalls Panic-Verhalten, soweit sie
  für legale Rewrites benötigt werden.

Die kompakte Inline-Darstellung bleibt für den häufigen atomaren Fall erhalten.
Nur tatsächlich strukturierte Fakten benötigen eine unveränderliche,
strukturell geteilte Erweiterung. `TypeDescriptor` beschreibt weiterhin die
statische Deklaration eines Builtins; während der Optimierung wird nicht bei
jedem Hook-Aufruf von `TypeInfo` in neu allokierte `TypeDescriptor`-Bäume und
zurück konvertiert.

Callback-Fixpunkte widersprechen dem Ein-Pass-Modell nicht: Der Lambda-AST wird
einmal analysiert und optimiert. Dabei entsteht aus seiner `TypeInfo` eine
monotone Transferfunktion. Ein notwendiger Fixpunkt läuft danach nur über diese
kompakten Typfakten, niemals erneut über den Lambda-AST.

## 1. Escape- und Lebensdaueranalyse als harte Sicherheitsgrenze

Eine Arena-Liste ist nur zulässig, wenn weder ihr Slice noch ein daraus
erreichbarer Knoten den ausführenden Lambda-Frame überlebt. Der Optimizer muss
Werte mindestens in folgende Klassen einordnen:

1. **frame-lokal:** vollständig innerhalb des aktuellen Aufrufs erzeugt und
   konsumiert;
2. **callee-lokal:** an einen als `NoEscape` deklarierten Parameter übergeben
   und während des Aufrufs vollständig konsumiert;
3. **escaping:** zurückgegeben, in einer Closure eingefangen oder in einen
   längerlebigen Wert eingebaut;
4. **persistierend:** in Queryplan-Cache, Schema, Trigger-Code, Session,
   globaler Umgebung oder einer asynchronen Aufgabe gespeichert.

Nur die ersten beiden Klassen dürfen `!list`/`!!list` verwenden. Insbesondere
müssen folgende Senken das Escape-Bit setzen:

- Rückgabewerte eines Planner-Schritts;
- Queryplan-Cache und vorbereitete Pläne;
- Schema- und Tabellenmetadaten;
- Trigger-Code und andere gespeicherte ASTs;
- Closure-Captures;
- Session-/Globalwerte und Übergabe an unbekannte Callbacks;
- Werte, die parallele oder verzögert ausgeführte Arbeit erreichen.

`NoEscape` ist dabei ein Vertrag des aufgerufenen Builtins, kein aus dem Namen
abgeleiteter Optimismus. Bei unbekanntem Ziel oder unvollständigem Beweis bleibt
die normale Heap-Liste erhalten.

## 2. Frame-lokale Vorallokation für temporäre Listen

Für einen nachweislich nicht escapenden Bereich kann der Optimizer die maximal
gleichzeitig benötigte Zahl von Listenelementen bestimmen oder konservativ
begrenzen. Dann wird einmal ein `!!list`-Bereich reserviert und jede temporäre
Liste als `!list`-Slice eines disjunkten Teilbereichs dargestellt.

Das gewünschte Rewrite-Schema ist:

```scheme
; vorher: mehrere kurzlebige Listen mit separaten Allokationen
(consume (list a b) (list c d e))

; interne Form nachher: zwei disjunkte Bereiche in VarsNumbered
(consume (!list NthLocalVar(start-a) 2 a b)
	 (!list NthLocalVar(start-b) 3 c d e))
```

`!!list NthLocalVar(start) cap` stellt entsprechend einen zunächst leeren Slice
mit Kapazität über einem Bereich von `VarsNumbered` dar. Diese nummerierten
Slots des aktuellen Frames sind die „Arena“; es soll keine globale Arena und
keine geteilte, nicht-thread-sichere Datenstruktur entstehen.

Erforderliche Regeln:

- Slices dürfen denselben Bereich nur wiederverwenden, wenn ihre Lebenszeiten
  beweisbar nicht überlappen.
- Ein unbekannter Listenbedarf fällt auf die normale Listenallokation zurück.
- Sobald ein Wert eine Escape-Senke erreicht, wird entweder gar nicht erst auf
  Arena umgeschrieben oder genau an dieser Grenze einmal in eine eigenständige
  Liste kopiert.
- Ein finaler Queryplan ist grundsätzlich escaping; nur seine temporären
  Zwischenlisten dürfen im Frame liegen.
- Die Optimierung muss bereits im Interpreter vollständig funktionieren; ein
  nachgelagertes Backend ist dafür weder Voraussetzung noch semantischer
  Mitspieler.

## 3. Konstante Teilbäume

Reine Teilbäume mit konstanten Eingaben werden im Optimizer ausgewertet. Das
Ergebnis muss als unveränderliche Konstante oder als bei Bedarf frisch zu
kopierender Wert im optimierten AST stehen. Spätere Phasen führen keine eigene
Konstantenerkennung aus.

Besondere Vorsicht gilt bei Listen und anderen referenzartigen Werten:

- unveränderliche Verwendung darf dieselbe Konstante referenzieren;
- eine spätere `_mut`-Operation benötigt eine nachweislich frische Kopie;
- gespeicherte ASTs dürfen niemals auf einen Frame-lokalen `!list`-Bereich
  zeigen.

## 4. Listen-Pipelines und bekannte Größen

Die bestehenden Fusionen sollten gezielt auf die Planner-Hotspots erweitert
werden:

- `map`/`filter`/`reduce`-Ketten nur einmal traversieren;
- bekannte Eingabelängen zur einmaligen Zielvorallokation nutzen;
- bei eindeutigem Ownership die vorhandenen `_mut`-Varianten wählen;
- unvermeidbare escaping Ergebnisse einmal passend dimensioniert erzeugen;
- Callback-Typen und Rückgabetypen bis zum Fixpunkt propagieren, damit die
  Ausführung keine redundanten Typprüfungen benötigt.

Das Ziel ist nicht, beliebige Listen pauschal mutierend zu machen, sondern
temporäre Konstruktionen durch einen belegbaren Lebensdauer- und
Ownership-Vertrag zu ersetzen.

## 5. Algorithmische Planner-Optimierung

Die Laufzeit von `queryplan.scm` wird nicht allein durch Dispatch und
Allokationen bestimmt. Reorder und Baumkonstruktion müssen zuerst auf
algorithmische Arbeit untersucht werden. Diese Entscheidungen gehören zum
Planner beziehungsweise Optimizer, nicht zum Emitter:

- Alias-Mengen als kompakte Bitsets statt wiederholt traversierter Listen;
- keine String-Erzeugung als Schlüssel für Subset-/Memo-Tabellen, wenn eine
  dichte numerische Repräsentation möglich ist;
- Memoisierung mit explizitem Zustandsbudget;
- Listen-Konkatenation und wiederholte Mengenoperationen aus inneren Schleifen
  ziehen;
- Zwischenbäume strukturell teilen, solange sie unveränderlich bleiben;
- erst an einer Escape-Grenze den dauerhaft gespeicherten AST materialisieren.

Kein späterer Ausführungsschritt darf einen quadratischen Algorithmus durch
eine implizite Repräsentationsänderung kaschieren.

## 6. Nachgewiesene Allokationsmuster im aktuellen Compiler

Die folgenden Punkte sind keine hypothetische Wunschliste. Sie folgen aus dem
aktuellen Go-Optimizer, aus `queryplan.scm` und aus den vorhandenen
`EXPLAIN`-/`EXPLAIN IR`-/`EXPLAIN COMPILE`-Abfragen.

Der aktuelle Queryplan-Compiler enthält unter anderem 523 `map`, 131 `filter`,
220 `reduce`, 230 `merge` und 2.264 `list`-Aufrufe. Die Anzahl allein ist kein
Optimierungsbeweis. Relevant sind die Kombinationen, die zusätzliche
Zwischenergebnisse erzeugen oder denselben Baum mehrfach durchlaufen.

### 6.1 Keine rekursive Reoptimierung für Metadaten

`optimizedExactListLength` und `optimizedExactAssocLength` rufen während eines
laufenden Optimizer-Passes erneut `OptimizeEx` auf. Falls eine Code-Konstante
materialisiert werden muss, kann derselbe Teilbaum sogar nochmals optimiert
werden. Ein Hook, der nur die bekannte Länge benötigt, darf nicht einen zweiten
Optimizer-Lauf über denselben Ausdruck auslösen.

Die einmal berechnete `TypeInfo` muss deshalb zusammen mit dem optimierten
Ausdruck zum Elternknoten fließen. Hooks erhalten die bereits ermittelten
Argumentdeskriptoren und fragen Länge, Ownership, Konstanz und Escape daraus ab.
Eine fehlende Information bleibt unbekannt; sie rechtfertigt keinen
Spekulationslauf.

Abnahmekriterium: Jeder Eingabeknoten wird im normalen Pass höchstens einmal
optimiert. Peepholes konsumieren die bereits optimierten Kinder und liefern ihre
Ersatzform mit `TypeInfo`; rekursive Rewrite-Neustarts sind nicht Teil des
Vertrags.

### 6.2 Copy-on-write statt vorsorglicher Vollkopien

`materializeCodeLiteral` reserviert bereits für jeden besuchten Listenknoten
einen vollständigen Ergebnis-Slice, bevor feststeht, ob sich überhaupt ein Kind
ändert. `CloneOptimizerExpression` kopiert für Callback-Analysen und spekulative
Rewrites komplette Teilbäume. Auch `rewriteNoEscapeListReturn` kopiert die Liste,
bevor feststeht, ob der aktuelle Knoten frame-lokal werden darf.

Der vorhandene Rückgabevertrag `(expression, TypeInfo)` reicht dafür aus. Bleiben
alle Kinder identisch, darf `expression` exakt der Eingangsknoten bleiben. Erst
beim ersten veränderten Kind wird ein neuer Slice angelegt und der unveränderte
Präfix übernommen. Unveränderte Geschwister und Teilbäume werden strukturell
geteilt. Alle semantischen Analysefakten stehen in der zugehörigen `TypeInfo`;
sie rechtfertigen keine defensive AST-Kopie.

Quellinformationen benötigen denselben Vertrag: Unveränderte `SourceInfo`-Werte
werden geteilt; nur ein tatsächlich geändertes Kind erzeugt einen neuen Wrapper.

### 6.3 Ein Pass liefert mehrere Baumfakten

Der Queryplan-Compiler läuft wiederholt über dieselben Ausdrücke, um jeweils nur
eine Eigenschaft zu bestimmen: referenzierte Aliase, externe Spalten,
Aggregate, Subqueries, Session-Abhängigkeiten, Stage-Outputs und
Predicate-Conjuncts. Join-Bäume werden getrennt nach Aliasmenge, erstem Alias
und Prädikaten traversiert. `EXPLAIN COMPILE` zählt denselben Plan anschließend
noch mehrmals für jede Operatorart.

Für unveränderliche ASTs soll eine compile-lokale Analyse pro Knoten gebündelt
werden. Ein Ergebnis kann beispielsweise enthalten:

```text
aliases, columns_by_alias, aggregates, stages, session_reads,
contains_subquery, contains_window, node_count, structural_hash
```

Der Cache-Schlüssel ist Knotenidentität plus der tatsächlich semantisch
relevante Kontext, etwa Default-Alias oder Scope-ID. Ein Rewrite erzeugt eine
neue Knotenidentität und invalidiert damit automatisch nur den geänderten Pfad.
Es darf keine globale, zwischen Compilations oder Threads geteilte mutable
Memo-Tabelle geben.

### 6.4 Telemetrie darf kein zweiter Optimizer sein

Der öffentliche `Optimize`-Pfad ruft seit PR #460 immer `OptimizeWithStats` auf.
Damit werden Eingabe und Ausgabe über `optimizerNodeCount` vollständig
traversiert, auch wenn der Aufrufer die Statistik verwirft. Der gleichzeitig
eingeführte `OptimizeRewrite` zählt Original und Rewrite erneut, hasht beide
Formen und verwaltet globale Budget-, Active- und Seen-Kataloge.

Diese Schicht ist kein Bestandteil der Zielarchitektur. `Optimize` ruft den
einen `OptimizeEx`-Pass direkt auf. Optionale Diagnosezähler hängen als
compile-lokaler, im Normalfall nicht vorhandener Observer an demselben Pass und
werden beim ohnehin stattfindenden Besuch inkrementiert. Aufwendig zu
bestimmende Ausgabefakten werden ausschließlich im expliziten Diagnosemodus
ermittelt; sie gehören nicht in das semantische Typsystem und dürfen den
Produktionspfad nicht belasten.

Peephole-Hooks erhalten bereits optimierte Argumente samt `TypeInfo` und geben
die kanonische Ersatzform samt korrekter `TypeInfo` zurück. Sie rufen den
Optimizer nicht rekursiv auf. Terminierung folgt aus einer festen,
idempotenten Kanonisierungsreihenfolge, nicht aus Fingerprints oder einem
willkürlichen globalen Rewrite-Budget. Ein neu konstruierter Knoten wird über
die Typfunktion seiner Deklaration beschrieben; bereits optimierte Kinder
werden dabei nicht erneut besucht.

### 6.5 Allokationsfreie Optimizer-Verwaltung im Normalfall

`applyDefaultOptimization` reserviert für praktisch jeden Funktionsaufruf ein
`[]TypeInfo` in Aufruflänge. `newOptimizerMetainfo`, Begin-Analyse und
Callback-Analyse erzeugen weitere Maps und Slices; wiederholte
`TypeInfo`-/`TypeDescriptor`-Konvertierungen kopieren verschachtelte
Deskriptoren. Aktuell ist `FlagEscape` zwar definiert, wird aber nirgends
gesetzt; `TypeInfoFromTD` übernimmt `NoEscape` nicht. Die strukturierte
Escape-Analyse ist im kompakten Typfluss daher noch nicht vollständig
implementiert.

Der Normalpfad soll daher:

- kleine Argumentdeskriptoren in einem festen lokalen Buffer halten und nur
  bei großer Arität auf einen Scratch-Slice ausweichen;
- `TypeInfo` als Wert durch die Hooks reichen und `TypeDescriptor` nur an
  tatsächlichen Deklarationsgrenzen materialisieren;
- unveränderte Descriptor-Teilbäume teilen und nur geänderte Pfade kopieren;
- compile-lokale Scratch-Maps nach klaren Phasengrenzen wiederverwenden;
- für lexikalische Locals dichte IDs und Vektoren statt mehrerer
  `map[Symbol]...` verwenden, sobald der Scope katalogisiert ist.

Die Begin-Optimierung durchsucht den Body momentan zuerst nach Definitionen,
danach rekursiv nach Nutzungen und dynamischer Syntax und optimiert ihn erst im
dritten Lauf. Statt eines neuen Analysepasses soll der eigentliche
Bottom-up-Pass einen Scope von rechts nach links falten: Die `TypeInfo` des
bereits besuchten Suffixes enthält Nutzung, Capture, dynamische Syntax und
Escape-Senken der sichtbaren Bindings. So erhält eine Definition beim einzigen
Besuch bereits die Fakten ihrer späteren Verbraucher. Die Ausgabereihenfolge
bleibt unverändert; bei dynamischem `eval` werden die betroffenen Fakten
konservativ unbekannt.

### 6.6 Callback-Fixpunkte ohne wiederholtes Klonen

`OptimizeReducerCallback` kann bis zu 16 Fixpunktiterationen ausführen.
`AnalyzeCallback` klont und optimiert dabei den vollständigen Callback für jede
Iteration erneut. Gerade die vielen Planner-Reducer vervielfachen so Baum- und
Descriptor-Allokationen.

Die Optimierung des Callback-AST erzeugt einmal eine monotone Transferfunktion
über Parameter-, Capture- und Rückgabefakten. Der Fixpunkt läuft anschließend
nur über kompakte `TypeInfo`-Werte. Falls die Transferfunktion wegen dynamischer
Syntax nicht ableitbar ist, bleibt der konservative Typ unbekannt; wiederholtes
Vollklonen ist kein zulässiger Fallback.

## 7. Fusionsmuster aus `queryplan.scm`

Die vorhandene Fusion `filter(map(xs, f), p) -> filter_map(xs, f, p)` deckt nur
eine Richtung ab. Im Planner treten weitere wiederkehrende Formen auf.

### 7.1 Filter vor Projektion

```scheme
(map (filter xs predicate) mapper)
```

Das benötigt einen internen `filter_then_map`-Loop: Prädikat auf dem
Eingabeelement prüfen und nur Treffer direkt ins Ergebnis projizieren. Das ist
nicht dasselbe wie das bestehende `filter_map`, dessen Prädikat den bereits
projizierten Wert sieht. Reihenfolge, Seiteneffekte und Panic-Verhalten müssen
erhalten bleiben.

### 7.2 Flache Reduktion ohne `merge`

```scheme
(reduce (merge parts) reducer initial)
(reduce (merge (map xs mapper)) reducer initial)
```

Der Optimizer kann die äußeren und inneren Listen direkt nacheinander
traversieren. Weder die flache Liste noch bei der zweiten Form die Liste der
Teillisten muss entstehen. Das interne Ergebnis ist ein `reduce_flat`-Loop oder
eine äquivalente verschachtelte Reducer-Form.

### 7.3 Builder-Reduktionen

Im Planner kommen häufig Akkumulatoren der Form vor:

```scheme
(reduce xs (lambda (acc item) (merge acc (list item))) '())
(reduce xs (lambda (acc item) (cons transformed acc)) '())
```

Die erste Form wächst ohne Ownership-Rewrite wiederholt durch Kopieren. Bei
bewiesen frischem Akkumulator soll der Optimizer eine einmal dimensionierte
Builder-Liste wählen und direkt anhängen. Ist die Endgröße unbekannt, wird mit
der bekannten Obergrenze `count(xs)` voralloziert; für Filter-Reducer ist dies
eine sichere Kapazität, keine semantische Längenbehauptung. `cons` plus finales
`reverse` kann entsprechend in einen vorwärts schreibenden Builder überführt
werden, sofern kein Zwischenstand des Akkumulators beobachtbar ist.

### 7.4 Entfernen, Filtern und Anhängen in einem Durchlauf

Der GOO-Join-Reorder erzeugt alle Paarkandidaten, filtert `nil`, reduziert auf
das Minimum und baut danach die Planliste mit `mapIndex`, `filter` und `append`
neu auf. Dafür sind zwei generische Muster sinnvoll:

- `argmin_map`: Kandidaten berechnen und das Minimum ohne Kandidatenliste
  bestimmen;
- `remove_indices_and_append`: zwei bekannte Positionen überspringen und den
  neuen Knoten in genau einer passend dimensionierten Liste anhängen.

Beide Regeln sind unabhängig vom SQL-Planner und helfen jedem funktional
geschriebenen Suchalgorithmus.

### 7.5 Mehrfaches Umschreiben durch Rewrite-Ketten

Derived-Flattening wendet eine Liste von Alias-/Projektions-Rewrites mit
`reduce` nacheinander auf WHERE, Felder, HAVING, ORDER und Hidden-Felder an.
Damit wird jeder Verbraucher einmal pro abgeleitetem Alias vollständig kopiert.

Wenn die Rewrites unabhängig und lexikalisch eindeutig sind, soll der
Optimizer ihre Lookup-Tabellen zuerst zu einer unveränderlichen
Substitutionsumgebung komponieren und den Verbraucher genau einmal traversieren.
Bei abhängigen Rewrites bleibt die Reihenfolge erhalten; auch dann kann ein
einziger Walker die geordnete Substitutionskette pro Referenz anwenden, statt
den ganzen Baum mehrfach aufzubauen.

## 8. Katalog- und Mengenmuster

### 8.1 Frische Assoc-Akkumulatoren in FastDict heben

`qassoc_set` implementiert ein Update als `cons` plus vollständiges `filter` des
alten Katalogs. Binding-, Stage-, Fakten- und Spaltenkataloge werden auf diese
Weise wiederholt aufgebaut. Auch `set_assoc` auf einer frischen Pair-Liste kann
bei wachsendem Katalog quadratische Kopierarbeit erzeugen.

Der Optimizer soll eine nur lokal aufgebaute Assoc-Reduktion erkennen und den
Akkumulator ab einer kleinen Schwelle in einen frischen FastDict heben. Das ist
analog zu `_mut`: Die Mutation ist nur erlaubt, wenn der Akkumulator eindeutig
besessen ist und kein Zwischenstand escaped. An der Rückgabegrenze bleibt der
existierende abstrakte Assoc-Vertrag erhalten; eine Rückkonvertierung erfolgt
nur, wenn ein konkreter Listenwert semantisch beobachtbar ist.

### 8.2 Alias- und Stage-Mengen als dichte IDs

Join-Reorder und Stage-Graph verwenden häufig Listen plus `contains?`,
`merge`, `filter` und `string aliases` für Teilmengen und Memo-Schlüssel. Sobald
ein compile-lokaler Katalog den sichtbaren Aliasen beziehungsweise Stage-IDs
dichte Nummern zuweist, kann der Optimizer diese Mengen als kleine Bitsets
darstellen.

Wichtige erkannte Operationen sind Union, Differenz, Teilmenge, Mitgliedschaft,
Popcount und kanonischer Memo-Key. Die Darstellung darf nicht anhand eines
Variablennamens gewählt werden, sondern nur aus einer geschlossenen
Producer-/Consumer-Kette, deren Werte nicht als Scheme-Liste beobachtet werden.

### 8.3 Einmalige Indizes statt wiederholter linearer Suche

Obwohl der Lowering-Catalog bereits Stage-Lookups kapselt, existieren weiterhin
zahlreiche `stage_by_id`, `source_aliases`, `source_by_alias` und
Spaltenkatalog-Suchen. Der Optimizer kann Schleifeninvarianten aus inneren
`map`/`filter`/`reduce`-Callbacks heben und einen unveränderlichen Index einmal
vor der Schleife aufbauen. Dies gilt insbesondere für:

- Source-Alias -> Source;
- Stage-ID -> Stage;
- SQL-Alias und gefalteter Alias -> Binding-Eintrag;
- Spaltenname und gefalteter Spaltenname -> exportierende Quellen;
- struktureller Aggregat-/Ausdrucksschlüssel -> vorhandener Eintrag.

Der Index muss in derselben Compile-Phase bleiben. Schemaänderungen und spätere
Planner-Rewrites dürfen keinen veralteten Index wiederverwenden.

## 9. Pfade, Namen und strukturelle Identitäten

Der Binder erzeugt für jeden besuchten Ausdruckspfad mehrere Strings über
verschachtelte `concat`-Aufrufe. Bei einer Alias-Kollision sammelt
`binding_fresh_query_alias` zusätzlich erneut sämtliche Strings des gesamten
Query-Baums. Pfade werden im Erfolgsfall aber überwiegend nur benötigt, um eine
deterministische interne Identität zu bilden.

Der compile-lokale Pfad soll deshalb als kompakte Folge numerischer
Komponenten oder als Parent-Zeiger plus Kindindex geführt werden. Stringbildung
erfolgt erst für Fehlermeldung oder endgültigen Namen. Der Stringkatalog des
Binders wird einmal pro Query aufgebaut und danach weitergereicht, nicht pro
Kollision neu traversiert.

Für Hashnamen ist die vorhandene Fusion
`fnv_hash(serialize|string(x)) -> stable_structural_hash(x, mode)` der richtige
Präzedenzfall: Erkennen der öffentlichen String-Komposition im Optimizer,
direktes Streamen in den Konsumenten und kein materialisierter Zwischenstring.
Dasselbe Prinzip gilt für `pretty_print`-, Parser-, Kompressions- und
Serialisierungs-Pipelines, sofern Reihenfolge und Ausgabeformat exakt bewahrt
werden.

## 10. Strukturteilung bei Planner-Konstruktoren

Mehrere Planner-Pässe bauen einen vollständigen `query-block`, `group-stage`
oder Join-Baum neu auf, obwohl nur ein Feld oder ein tiefer Nachfolger geändert
wurde. Der Optimizer soll Konstruktoren mit unveränderlichen Eingängen als
persistent-data-Rewrites behandeln:

- identische Kinder liefern den ursprünglichen Knoten zurück;
- bei einer Änderung wird nur der Pfad zum geänderten Kind kopiert;
- `map` über Kinder gibt bei ausschließlich identischen Ergebnissen die
  ursprüngliche Kinderliste zurück;
- abgeleitete Fakten wie Alias-Bitset, Node-Count und Structural Hash können am
  unveränderten Knoten wiederverwendet werden.

Dies erfordert keine neue logische Operatorhierarchie. Die kombinierten
`query-block`, `group-stage` und `union-block` bleiben erhalten; optimiert wird
nur ihre physische Darstellung während der Compilation.

## 11. Messung mit den vorhandenen EXPLAIN-Abfragen

`EXPLAIN COMPILE` trennt bereits Parser, `untangle`, Reorder, physische
Vorbereitung, Emission und Serialisierung. Es misst derzeit jedoch nicht den
anschließenden Go-Optimizer, obwohl normales `EXPLAIN` genau diesen auf dem
emittierten Plan ausführt. Außerdem fehlen Allokations- und Strukturteilungswerte.

Die Telemetrie soll mindestens ergänzen:

- `optimizer_ns`, `optimizer_input_nodes`, `optimizer_output_nodes`;
- `optimizer_allocs` und `optimizer_alloc_bytes`;
- `optimizer_nodes_visited` und `optimizer_nodes_reoptimized`;
- `optimizer_nodes_cloned` und `optimizer_nodes_shared`;
- `callback_analyses`, `callback_cloned_nodes` und Fixpunktiterationen;
- Anzahl und Bytes temporärer Listen je Planner-Phase;
- Cache-Hit/Miss getrennt für Queryform, Spezialisierungsvariante und
  compile-lokale Analyseindizes.

Globale `runtime.MemStats`-Differenzen sind unter parallelen Compilations nicht
zurechenbar. Für CI-Budgets eignen sich deterministische interne Zähler und
Go-Benchmarks mit `testing.AllocsPerRun`; Prozessprofile dienen ergänzend zur
Hotspot-Suche.

Der vorhandene EXPLAIN-Korpus deckt die nötigen Skalierungsachsen bereits gut
ab und soll als Allocation-Corpus verwendet werden:

- 8 bis 160 wiederholte abhängige Scalar-/EXISTS-Teilbäume;
- zwei- bis sechsfacher Join inklusive bushy Trees und DPHyp;
- lange Derived-/Alias-Rewrite-Ketten;
- viele Stage-Outputs mit gemeinsamem Dependency-Graph;
- GROUP BY mit mehreren kanonischen Aggregaten;
- UNION-Zweigzahl, Fensteranzahl und Projektionsbreite;
- Ausdruckstiefe für CASE, AND/OR sowie korrelierte Membership.

Für jede Achse werden kalte Compilation, warmer Queryform-Cache und warmer
Spezialisierungsvariant-Cache getrennt gemessen. Zeit allein genügt nicht:
Allokationen und erzeugte Knoten müssen linear oder besser mit der jeweiligen
Achse skalieren.

## 12. Vorgeschlagene Implementierungsreihenfolge

1. Den normalen `Optimize`-Pfad wieder direkt auf den einen `OptimizeEx`-Pass
   führen; Statistik nur explizit und ohne implizite Vollbaumläufe erfassen.
2. `TypeInfo` als einzigen internen Analysevertrag vervollständigen:
   strukturiertes Escape, partielle Konstanten, Lambda-Transferfunktionen und
   verlustfreie Übernahme statischer `TypeDescriptor`-Fakten.
3. Hook-API auf bereits optimierte Argumente plus `TypeInfo` umstellen und die
   rekursive `OptimizeRewrite`-Schicht nach Migration ihrer Regeln entfernen.
4. Rekursive `OptimizeEx`-Aufrufe aus Längen-/Typanalyse entfernen und
   `TypeInfo` aus dem laufenden Pass direkt verwenden.
5. `materializeCodeLiteral`, Callback-Analyse und generische Baum-Rewrites auf
   lazy copy-on-write umstellen.
6. Per-Call-Deskriptoren, Begin-Scope-Analyse und Callback-Fixpunkte auf
   kompakte Werte und wiederverwendbaren compile-lokalen Scratch umstellen.
7. `EXPLAIN COMPILE` über einen optionalen Observer und Benchmarks um
   phasenbezogene Allocation-, Clone-, Sharing- und Besuchszähler ergänzen.
8. Die im Planner häufigen Formen `map(filter(...))`, `reduce(merge(...))`,
   Builder-Reducer und Argmin-Suchen fusionieren.
9. Pure Baumfakten pro Knoten gemeinsam berechnen und unveränderte
   Queryblock-/Stage-/Join-Unterbäume strukturell teilen.
10. Frische Assoc-Akkumulatoren, Alias-Mengen und Stage-Mengen in FastDicts,
   unveränderliche Indizes beziehungsweise Bitsets heben.
11. Erst danach `!list`/`!!list` für die verbleibenden nachweislich
   frame-lokalen temporären Listen einsetzen.
12. Nach jedem Schritt Differentialtests sowie kalte und warme Allocation-
   Skalierung gegen aktuellen Master messen.

## Abnahmekriterien

- Kein Arena-Slice erreicht Rückgabewert, Queryplan-Cache, Schema, Trigger-Code,
  Closure, Session oder asynchrone Arbeit.
- Explizite Negativtests decken jede Escape-Senke ab.
- Escaping Ergebnisse werden höchstens einmal an der Grenze materialisiert.
- Optimierter Interpreter und unoptimierte Ausführung liefern denselben Wert.
- Die Messung trennt Zeit und Allokationen für Baumkonstruktion, Reorder und
  Emission.
- Kein normaler Knoten wird zur Längen- oder Typbestimmung erneut optimiert.
- Jede Optimizer-Entscheidung stützt sich auf die strukturierte `TypeInfo` des
  laufenden Passes; es existiert kein paralleler Analyse- oder Rewrite-Pass.
- Der statistiklose `Optimize`-Pfad zählt oder hasht keinen vollständigen Baum
  zusätzlich zum eigentlichen Bottom-up-Pass.
- Callback-Fixpunkte klonen den Callback-AST nicht pro Iteration.
- Unveränderte Teilbäume werden nicht kopiert; Sharing- und Clone-Zähler machen
  dies im EXPLAIN-/Benchmark-Korpus sichtbar.
- Die häufigen Planner-Pipelines erzeugen keine materialisierten
  Zwischenlisten, wenn ein einzelner äquivalenter Loop ausreicht.
- Skalierungstests unterscheiden konstante, lineare, `N log N`- und
  quadratische Arbeit; ein kleinerer Absolutwert allein gilt nicht als Erfolg.
