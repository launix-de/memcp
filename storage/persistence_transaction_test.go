/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/launix-de/memcp/scm"
)

type failingLogSwapPersistence struct {
	PersistenceEngine
}

func (f *failingLogSwapPersistence) SwapLog(string, []interface{}, bool) PersistenceLogfile {
	panic("forced log swap failure")
}

func TestFileTransactionalWALRoundTrip(t *testing.T) {
	engine := &FileStorage{path: filepath.Join(t.TempDir(), "db") + "/"}
	logfile := engine.OpenLog("roundtrip")
	txID := "boot-id/17"
	logfile.Write(LogEntryInsertHidden{
		cols:   []string{"id"},
		values: [][]scm.Scmer{{scm.NewInt(7)}},
		txID:   txID,
	})
	logfile.Write(LogEntryDelete{idx: 3, txID: txID})
	logfile.Write(LogEntryUndelete{idx: 4, txID: txID})
	logfile.Write(LogEntryCommit{txID: txID})
	logfile.Flush(true)
	logfile.Close()

	committed, entries, replayLog := engine.ReplayLog("roundtrip", false)
	defer replayLog.Close()
	if _, ok := committed[txID]; !ok {
		t.Fatal("transaction commit was not discovered before replay")
	}
	var restored []interface{}
	for entry := range entries {
		restored = append(restored, entry)
	}
	if len(restored) != 4 {
		t.Fatalf("restored %d WAL entries, want 4", len(restored))
	}
	insert, ok := restored[0].(LogEntryInsertHidden)
	if !ok || insert.txID != txID || len(insert.values) != 1 || scm.ToInt(insert.values[0][0]) != 7 {
		t.Fatalf("restored transactional insert = %#v", restored[0])
	}
	if deletion, ok := restored[1].(LogEntryDelete); !ok || deletion.idx != 3 || deletion.txID != txID {
		t.Fatalf("restored transactional delete = %#v", restored[1])
	}
	if undeletion, ok := restored[2].(LogEntryUndelete); !ok || undeletion.idx != 4 || undeletion.txID != txID {
		t.Fatalf("restored transactional undelete = %#v", restored[2])
	}
	if commit, ok := restored[3].(LogEntryCommit); !ok || commit.txID != txID {
		t.Fatalf("restored transaction commit = %#v", restored[3])
	}
}

func TestFileWALIgnoresTornFinalCommit(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "db")
	if err := os.MkdirAll(dir, 0750); err != nil {
		t.Fatal(err)
	}
	validID := "boot-id/31"
	tornID := "boot-id/32"
	body := "commit-tx " + validID + " " + fileCommitChecksum(validID) + "\n" +
		"commit-tx " + tornID + " " + fileCommitChecksum(tornID)
	if err := os.WriteFile(filepath.Join(dir, transactionLogName+".log"), []byte(body), 0640); err != nil {
		t.Fatal(err)
	}

	engine := &FileStorage{path: dir + "/"}
	_, entries, logfile := engine.ReplayLog(transactionLogName, false)
	defer logfile.Close()
	var restored []interface{}
	for entry := range entries {
		restored = append(restored, entry)
	}
	if len(restored) != 1 {
		t.Fatalf("restored %d commit records, want only the complete record", len(restored))
	}
	commit, ok := restored[0].(LogEntryCommit)
	if !ok || commit.txID != validID {
		t.Fatalf("restored commit = %#v, want %q", restored[0], validID)
	}
}

func TestDatabaseCommitAuthoritySurvivesReopen(t *testing.T) {
	engine := &FileStorage{path: filepath.Join(t.TempDir(), "db") + "/"}
	db := newDatabase()
	db.persistence = engine
	db.commitTransaction("boot-id/23", true)
	db.closeTransactionLog()

	reloaded := newDatabase()
	reloaded.persistence = engine
	if !reloaded.transactionCommitted("boot-id/23") {
		t.Fatal("durable database commit was not restored")
	}
	if reloaded.transactionCommitted("boot-id/24") {
		t.Fatal("transaction without a commit record was restored")
	}
	reloaded.closeTransactionLog()
}

func TestFileSwapLogPublishesCompleteReplacement(t *testing.T) {
	engine := &FileStorage{path: filepath.Join(t.TempDir(), "db") + "/"}
	old := engine.OpenLog(transactionLogName)
	old.Write(LogEntryCommit{txID: "old/1"})
	old.Write(LogEntryCommit{txID: "old/2"})
	old.Flush(true)

	replacement := engine.SwapLog(transactionLogName, []interface{}{
		LogEntryCommit{txID: "retained/1"},
	}, true)
	// The old descriptor still names the unlinked generation. A late write to
	// it must not modify the newly published log path.
	old.Write(LogEntryCommit{txID: "old/late"})
	old.Flush(true)
	old.Close()
	replacement.Write(LogEntryCommit{txID: "new/1"})
	replacement.Flush(true)
	replacement.Close()

	committed, entries, logfile := engine.ReplayLog(transactionLogName, false)
	defer logfile.Close()
	for range entries {
	}
	for _, txID := range []string{"retained/1", "new/1"} {
		if _, ok := committed[txID]; !ok {
			t.Errorf("replacement log does not contain %q", txID)
		}
	}
	for _, txID := range []string{"old/1", "old/2", "old/late"} {
		if _, ok := committed[txID]; ok {
			t.Errorf("replacement log still contains %q", txID)
		}
	}
}

func TestDatabaseTransactionLogCompactionRetainsNewCommits(t *testing.T) {
	engine := &FileStorage{path: filepath.Join(t.TempDir(), "db") + "/"}
	db := newDatabase()
	db.persistence = engine
	db.commitTransaction("before/1", true)
	db.commitTransaction("before/2", true)
	obsolete, _ := db.transactionCompactionSnapshot()
	db.commitTransaction("during/1", true)
	db.compactTransactionLog(obsolete)

	if db.transactionCommitted("before/1") || db.transactionCommitted("before/2") {
		t.Fatal("compacted transaction IDs remain in the in-memory authority")
	}
	if !db.transactionCommitted("during/1") {
		t.Fatal("commit created during rebuild was removed")
	}
	db.closeTransactionLog()

	reloaded := newDatabase()
	reloaded.persistence = engine
	if reloaded.transactionCommitted("before/1") || reloaded.transactionCommitted("before/2") {
		t.Fatal("compacted transaction IDs were restored from disk")
	}
	if !reloaded.transactionCommitted("during/1") {
		t.Fatal("retained transaction ID was not restored from disk")
	}
	reloaded.closeTransactionLog()
}

func TestDatabaseTransactionLogSwapFailureKeepsOldAuthority(t *testing.T) {
	engine := &FileStorage{path: filepath.Join(t.TempDir(), "db") + "/"}
	db := newDatabase()
	db.persistence = engine
	db.commitTransaction("before/1", true)
	obsolete, _ := db.transactionCompactionSnapshot()
	db.persistence = &failingLogSwapPersistence{PersistenceEngine: engine}

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("compaction did not report the forced swap failure")
			}
		}()
		db.compactTransactionLog(obsolete)
	}()
	if !db.transactionCommitted("before/1") {
		t.Fatal("failed swap removed the old in-memory authority")
	}
	// The original append handle must remain usable after a pre-publication
	// swap failure.
	db.commitTransaction("after-failure/1", true)
	db.closeTransactionLog()

	reloaded := newDatabase()
	reloaded.persistence = engine
	if !reloaded.transactionCommitted("before/1") || !reloaded.transactionCommitted("after-failure/1") {
		t.Fatal("failed swap changed the durable transaction authority")
	}
	reloaded.closeTransactionLog()
}

// A native UPDATE callback can run without any scan mapper or INSERT success
// path. The commit itself owns durability for every changed generation.
func TestNativeACIDUpdatesSurviveRestart(t *testing.T) {
	for _, count := range []int{1, 2} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			first, engine := createDurabilityTestTable(t, "tnative_update_commit", 1)
			tables := []*table{first}
			if count == 2 {
				second, _ := CreateTable(first.schema.Name, "second", Safe, false)
				second.CreateColumn("id", "INT", nil, nil)
				second.CreateColumn("payload", "TEXT", nil, nil)
				second.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(1), scm.NewString("old")}}, nil, scm.NewNil(), false, nil)
				tables = append(tables, second)
			}
			tx := NewTxContext(TxACID)
			defer tx.Rollback()
			for _, table := range tables {
				table.ActiveShards()[0].UpdateFunction(0, false, false, tx)(scm.NewSlice([]scm.Scmer{scm.NewString("payload"), scm.NewString("committed")}))
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			reopened := newDatabase()
			reopened.Name, reopened.persistence, reopened.srState = first.schema.Name, engine, COLD
			reopened.ensureLoaded()
			defer reopened.closeTransactionLog()
			for _, original := range tables {
				sh := reopened.tables.Get(original.Name).ActiveShards()[0]
				sh.ensureLoaded()
				found := 0
				read := sh.ColumnReaderTx(nil, "payload", false, ReadAuto)
				for id := uint32(0); id < sh.main_count+uint32(len(sh.deltaRows())); id++ {
					if !sh.deletions.Get(uint(id)) {
						found++
						if got := read.GetValue(id); got.String() != "committed" {
							t.Errorf("%s after restart: %v", original.Name, got)
						}
					}
				}
				releaseColumnReaders([]ColumnReader{read})
				if found != 1 {
					t.Errorf("%s visible rows=%d", original.Name, found)
				}
			}
		})
	}
}

func TestFileReadOnlyReplayDoesNotCreateOrExposeAppendHandle(t *testing.T) {
	root := filepath.Join(t.TempDir(), "not-created")
	engine := &FileStorage{path: root + "/"}
	committed, entries, writer := engine.ReplayLog("missing-decision", true)
	for range entries {
		t.Fatal("missing log yielded an entry")
	}
	if len(committed) != 0 || writer != nil {
		t.Fatal("read-only absence created commit authority or an append handle")
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("read-only absence modified the backend: %v", err)
	}
	log := engine.SwapLog("recorded-decision", []interface{}{LogEntryCommit{txID: "committed"}}, true)
	log.Close()
	committed, entries, writer = engine.ReplayLog("recorded-decision", true)
	count := 0
	for range entries {
		count++
	}
	if _, found := committed["committed"]; !found || count != 1 || writer != nil {
		t.Fatal("read-only replay lost a decision or exposed a writer")
	}
}
