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
	"os"
	"path/filepath"
	"testing"

	"github.com/launix-de/memcp/scm"
)

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
	logfile.Sync()
	logfile.Close()

	committed, entries, replayLog := engine.ReplayLog("roundtrip")
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
	_, entries, logfile := engine.ReplayLog(transactionLogName)
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
