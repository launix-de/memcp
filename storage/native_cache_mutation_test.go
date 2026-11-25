/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/launix-de/memcp/scm"
)

func cacheMutationFixture(t *testing.T, foreign bool) (*table, *table) {
	t.Helper()
	t.Cleanup(setupGCTest(t))
	CreateDatabase("gcdb", false)
	db := "gcdb"
	if foreign {
		db = "native-cache-input"
		CreateDatabase(db, false)
		t.Cleanup(func() { databases.Remove(db) })
	}
	source, _ := CreateTable("gcdb", "cache_source", Memory, false)
	target, _ := CreateTable(db, ".derived_input", Cache, false)
	for _, table := range []*table{source, target} {
		table.CreateColumn("id", "INT", nil, nil)
		table.CreateColumn("value", "INT", nil, nil)
		table.Insert([]string{"id", "value"}, [][]scm.Scmer{
			{scm.NewInt(1), scm.NewInt(10)}, {scm.NewInt(2), scm.NewInt(20)},
		}, nil, scm.NewNil(), false, nil)
	}
	return source, target
}

func registerCacheMutationTrigger(source, target *table, name string, body scm.Scmer) {
	cache := scm.NewNil()
	if target != nil {
		cache = NewTableScmer(target)
	}
	scm.Apply(scm.Eval(scm.NewSymbol("createtrigger"), &scm.Globalenv),
		NewTableScmer(source), scm.NewString(name), scm.NewString("after_update"),
		scm.NewString(""), scm.NewString(""), body, scm.NewBool(false), cache)
}

// Native bodies cannot be persisted as SQL/SCM source. Exercise the same
// target publication owner directly; the common API is tested below with a
// genuine serializable Scheme body.
func registerNativeCacheMutationTrigger(source, target *table, name string, body scm.Scmer) {
	trigger := TriggerDescription{Name: name, Timing: AfterUpdate, Func: body, Hidden: true, IsSystem: true}
	if target != nil {
		defer trigger.bindCacheTarget(target)()
	}
	source.AddTrigger(trigger)
}

func cacheMutationMirror(t *testing.T, target *table, original **TxContext) scm.Scmer {
	return scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		binding := scmerToTxContext(args[3])
		if original != nil && binding.transactionState != (*original).transactionState {
			panic("derived maintenance replaced its original transaction")
		}
		id, _ := args[1].FastDict().Get(scm.NewString("id"))
		value, _ := args[1].FastDict().Get(scm.NewString("value"))
		return statementAdmissionUpdate(target, binding, id.Int(), value.Int())
	})
}

func TestCacheMutationUsesOriginalUndoAndDecision(t *testing.T) {
	for _, foreign := range []bool{false, true} {
		t.Run(fmt.Sprint(foreign), func(t *testing.T) {
			source, target := cacheMutationFixture(t, foreign)
			tx := statementAdmissionTransaction(false)
			defer rollbackStatementAdmissionTransaction(tx)
			registerNativeCacheMutationTrigger(source, target, "native_cache_mirror", cacheMutationMirror(t, target, &tx))
			statementAdmissionUpdate(source, tx, 1, 17)
			assertStatementAdmissionRows(t, source, tx, 17, 20)
			assertStatementAdmissionRows(t, target, tx, 17, 20)
			assertStatementAdmissionRows(t, source, nil, 10, 20)
			assertStatementAdmissionRows(t, target, nil, 10, 20)
			sp := tx.CreateSavepoint()
			statementAdmissionUpdate(source, tx, 2, 29)
			assertStatementAdmissionRows(t, target, tx, 17, 29)
			tx.RollbackToSavepoint(sp)
			assertStatementAdmissionRows(t, source, tx, 17, 20)
			assertStatementAdmissionRows(t, target, tx, 17, 20)
			GlobalCache.Stop()
			defer resumeOwnershipTestCache()
			if keytableCleanup(target, nil) {
				t.Fatal("cache generation evicted while original transaction retained its overlay")
			}
			tx.Rollback()
			assertStatementAdmissionRows(t, source, nil, 10, 20)
			assertStatementAdmissionRows(t, target, nil, 10, 20)
			if !keytableCleanup(target, nil) || len(source.Triggers) != 0 {
				t.Fatal("decision did not release cache pin and exact reverse dependency")
			}
		})
	}
}

func TestCacheMutationDoesNotAuthorizeUnrelatedSibling(t *testing.T) {
	for _, public := range []bool{false, true} {
		t.Run(fmt.Sprint(public), func(t *testing.T) {
			source, target := cacheMutationFixture(t, true)
			registerNativeCacheMutationTrigger(source, target, "first_legitimate_cache", cacheMutationMirror(t, target, nil))
			other := target
			if public {
				other, _ = CreateTable(target.schema.Name, "public_target", Memory, false)
				other.CreateColumn("id", "INT", nil, nil)
				other.CreateColumn("value", "INT", nil, nil)
				other.Insert([]string{"id", "value"}, [][]scm.Scmer{{scm.NewInt(1), scm.NewInt(10)}, {scm.NewInt(2), scm.NewInt(20)}}, nil, scm.NewNil(), false, nil)
			}
			// Same physical cache was already legitimately admitted above. A
			// sibling without the actual input still cannot borrow that membership.
			registerNativeCacheMutationTrigger(source, nil, "unrelated_sibling", cacheMutationMirror(t, other, nil))
			tx := statementAdmissionTransaction(false)
			defer rollbackStatementAdmissionTransaction(tx)
			var failure any
			func() { defer func() { failure = recover() }(); statementAdmissionUpdate(source, tx, 1, 99) }()
			if failure == nil || !strings.Contains(fmt.Sprint(failure), "outside its admitted database") {
				t.Fatalf("unrelated sibling result = %v", failure)
			}
			assertStatementAdmissionRows(t, source, tx, 10, 20)
			assertStatementAdmissionRows(t, target, tx, 10, 20)
			assertStatementAdmissionRows(t, other, tx, 10, 20)
		})
	}
}

func TestCacheMutationCommitsBothTargetsAtOriginalDecision(t *testing.T) {
	source, target := cacheMutationFixture(t, true)
	registerNativeCacheMutationTrigger(source, target, "native_cache_commit", cacheMutationMirror(t, target, nil))
	tx := statementAdmissionTransaction(false)
	defer rollbackStatementAdmissionTransaction(tx)
	statementAdmissionUpdate(source, tx, 1, 31)
	assertStatementAdmissionRows(t, source, nil, 10, 20)
	assertStatementAdmissionRows(t, target, nil, 10, 20)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	assertStatementAdmissionRows(t, source, nil, 31, 20)
	assertStatementAdmissionRows(t, target, nil, 31, 20)
}

func TestCacheMutationPublishesPinnedTargetAndRestoresInert(t *testing.T) {
	source, target := cacheMutationFixture(t, true)
	registerCacheMutationTrigger(source, target, "native_cache_restore", buildFKProc(scm.NewBool(true)))
	if len(source.Triggers) != 1 || source.Triggers[0].cacheTarget != target || target.cacheTriggerSources.Load().table != source {
		t.Fatal("registration did not publish exact target and reverse edge together")
	}
	encoded, err := json.Marshal(source.Triggers[0])
	if err != nil {
		t.Fatal(err)
	}
	var restored TriggerDescription
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.acquireTarget(nil) {
		t.Fatal("restored callback ran without a regenerated native target")
	}
	GlobalCache.Stop()
	defer resumeOwnershipTestCache()
	if !keytableCleanup(target, nil) || len(source.Triggers) != 0 {
		t.Fatal("eviction left active callbacks referring to the missing target")
	}
	replacement, _ := CreateTable(target.schema.Name, target.Name, Cache, false)
	registerCacheMutationTrigger(source, replacement, "native_cache_restore", buildFKProc(scm.NewBool(true)))
	if source.Triggers[0].cacheTarget != replacement || source.Triggers[0].cacheTarget == target {
		t.Fatal("same-name recreation reused the old physical target")
	}
}

func TestCacheMutationRejectsImportedInvocationBeforeEffects(t *testing.T) {
	source, target := cacheMutationFixture(t, true)
	tx := statementAdmissionTransaction(false)
	defer rollbackStatementAdmissionTransaction(tx)
	tx.remoteInvocation = true
	operation, err := newNativeMutationBinding(tx, tx.Session)
	if err != nil {
		t.Fatal(err)
	}
	defer operation.finishProducer()
	var failure any
	func() { defer func() { failure = recover() }(); target.retainCacheMutationTarget(operation.invocation) }()
	if failure == nil || !strings.Contains(fmt.Sprint(failure), "original local native producer") {
		t.Fatalf("imported cache authority = %v", failure)
	}
	assertStatementAdmissionRows(t, source, nil, 10, 20)
	assertStatementAdmissionRows(t, target, nil, 10, 20)
	if atomic.LoadInt64(&target.cacheUsers) != 0 {
		t.Fatal("rejected invocation retained a target pin")
	}
}

func TestCacheMutationDispatchPreservesSharedProducerHeaders(t *testing.T) {
	var calls, pins atomic.Int64
	headers := []TriggerDescription{{Name: "immutable_producer_callback", Func: scm.NewBool(true),
		Acquire: func(*TxContext) bool {
			if calls.Add(1)%2 == 1 {
				return false
			}
			pins.Add(1)
			return true
		},
		Release: func() { pins.Add(-1) },
	}}
	var wg sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for event := 0; event < 8; event++ {
				selected, release := nativeRowTriggerInvocations(nil, headers, true)
				if &selected[0] == &headers[0] {
					t.Error("event reused immutable producer header storage")
				}
				release()
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 128 || pins.Load() != 0 || headers[0].Func.IsNil() || headers[0].invocation != nil {
		t.Fatalf("shared producer corrupted: calls=%d pins=%d header=%+v", calls.Load(), pins.Load(), headers[0])
	}
}

func TestComputedPublicTargetPinKeepsOrdinaryInvocation(t *testing.T) {
	source, _ := cacheMutationFixture(t, false)
	tx := statementAdmissionTransaction(false)
	defer rollbackStatementAdmissionTransaction(tx)
	operation, err := newNativeMutationBinding(tx, tx.Session)
	if err != nil {
		t.Fatal(err)
	}
	operation.runProducer(func(binding *TxContext) scm.Scmer {
		operation.beginStatement(source.schema)
		operation.admitStatementTable(source)
		operation.prepareStatement()
		headers := []TriggerDescription{{Func: scm.NewBool(true), cacheTarget: source,
			Acquire: source.acquireCacheUseForTrigger, Release: source.releaseCacheUse}}
		selected, release := nativeRowTriggerInvocations(binding, headers, true)
		defer release()
		if selected[0].invocation != binding || selected[0].invocation.mutationOperation.cacheInput != nil {
			panic("public computed-column lifetime pin changed its native row authority")
		}
		return scm.NewNil()
	})
}
