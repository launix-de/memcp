# clean -- Garbage Collection fuer verwaiste Ressourcen

## Blob Cleanup

Crash-Waisen: Wenn MemCP nach IncrBlobRefcount aber vor FlushBlobRefcounts abstuerzt,
bleiben Blob-Dateien auf Disk die von keinem Refcount-Eintrag referenziert werden.

Ansatz:
1. Alle Blob-Hashes auf Disk auflisten (braucht ListBlobs pro Backend)
2. Alle Hashes aus refcounts.json laden
3. Differenz = Waisen → loeschen

Neue PersistenceEngine-Methode:
```go
ListBlobs() []string // gibt alle hex-hashes zurueck die auf Disk liegen
```

Implementierung pro Backend:
- FileStorage: `filepath.Walk(path+"blob/")`, Dateinamen sammeln
- CephStorage: RADOS object listing mit Prefix `blob/` (oder Manifest-Objekt fuehren)
- S3Storage: `ListObjectsV2` mit Prefix `blob/`

## Shard Cleanup

Wenn rebuild() panict, wird der neue Shard zwar per recover() aufgeraeumt,
aber die alten Shard-Dateien werden erst per `runtime.SetFinalizer` geloescht.
Wenn der Finalizer nie laeuft (z.B. weil der alte Shard-Pointer noch irgendwo
referenziert wird, oder der Prozess vorher stirbt), bleiben verwaiste
Column-Dateien und Log-Dateien auf Disk.

Ansatz:
1. Alle Shard-UUIDs aus schema.json lesen (= aktive Shards)
2. Alle Dateien im DB-Verzeichnis auflisten
3. Dateien deren UUID-Prefix nicht in der aktiven Menge liegt → loeschen

Neue PersistenceEngine-Methode:
```go
ListColumns() []string // gibt alle "<uuid>-<col>" Dateinamen zurueck
ListLogs() []string    // gibt alle "<uuid>.log*" Dateinamen zurueck
```

Oder alternativ eine generische Methode:
```go
ListObjects() []string // alle Objekte im DB-Prefix
```

Dann in clean():
- aktive UUIDs aus den geladenen Shards extrahieren
- alle Dateien/Objekte filtern die nicht zu aktiven Shards gehoeren
- diese loeschen
