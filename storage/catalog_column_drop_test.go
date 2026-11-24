/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "testing"
import "github.com/launix-de/memcp/scm"

func TestStoredColumnRetirementFencesLifecyclePublication(t *testing.T) {
	for _, savepoint := range []bool{false, true} {
		name := "rollback"
		if savepoint {
			name = "savepoint"
		}
		t.Run(name, func(t *testing.T) {
			tbl, _ := createDurabilityTestTable(t, "tcolumnretirelifecycle", 1)
			tx := NewTxContext(TxACID)
			defer func() {
				if tx.State == TxActive {
					tx.Rollback()
				}
			}()
			sp := tx.CreateSavepoint()
			if !tx.dropPublicColumn(tbl, "payload", false) {
				t.Fatal("stored declaration was not retired")
			}
			trigger := TriggerDescription{Name: ".retire-lifecycle-proof", Timing: AfterDropColumn, IsSystem: true,
				Func: scm.NewProcStruct(scm.Proc{Params: scm.NewSlice(nil), Body: scm.NewBool(true), En: &scm.Globalenv})}
			var failure any
			func() { defer func() { failure = recover() }(); tbl.AddTrigger(trigger) }()
			if failure == nil {
				t.Fatal("late lifecycle publication was accepted")
			}
			if len(tbl.Triggers) != 0 || len(tbl.columnDeclarationsForTx(tx)) != 1 || len(tbl.Columns) != 2 {
				t.Fatal("rejected lifecycle changed the trigger list or selected/public declarations")
			}
			if savepoint {
				tx.RollbackToSavepoint(sp)
			} else {
				tx.Rollback()
			}
			tbl.AddTrigger(trigger)
			if len(tbl.Triggers) != 1 || tbl.Triggers[0].Name != trigger.Name || len(tbl.columnDeclarationsForTx(tx)) != 2 {
				t.Fatal("retirement rewind did not reopen the native lifecycle publication owner")
			}
			if tx.State == TxActive {
				tx.Rollback()
			}
			assertInitializedColumn(t, tbl, "payload", scm.NewString("row-00000001"), 1)
		})
	}
}
