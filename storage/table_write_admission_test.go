/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/launix-de/memcp/scm"
)

// Observing Done proves entry into the native admission select, without a
// scheduling delay, polling loop or production test hook.
type statementAdmissionContext struct {
	context.Context
	once    sync.Once
	waiting chan struct{}
}

func (ctx *statementAdmissionContext) Done() <-chan struct{} {
	ctx.once.Do(func() { close(ctx.waiting) })
	return ctx.Context.Done()
}

func statementAdmissionTransaction(implicit bool) *TxContext {
	tx := NewTxContext(TxACID)
	tx.Session = scm.NewSession()
	scm.Apply(tx.Session, scm.NewString("username"), scm.NewString("root"))
	tx.autoCommit = implicit
	return tx
}

func statementAdmissionFixture(t *testing.T) (*table, *atomic.Int64) {
	t.Helper()
	cleanup := setupGCTest(t)
	t.Cleanup(cleanup)
	CreateDatabase("gcdb", false)
	tbl, _ := CreateTable("gcdb", "statement_admission", Memory, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("value", "INT", nil, nil)
	tbl.Insert([]string{"id", "value"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewInt(10)}, {scm.NewInt(2), scm.NewInt(20)},
	}, nil, scm.NewNil(), false, nil)
	observed := new(atomic.Int64)
	language := "statement-admission-test"
	env := &scm.Env{Outer: &scm.Globalenv, Vars: scm.Vars{
		scm.Symbol("observed"): scm.NewFunc(func(...scm.Scmer) scm.Scmer {
			observed.Add(1)
			return scm.NewNil()
		}),
	}}
	registerTriggerLanguage(language, scm.NewFunc(func(...scm.Scmer) scm.Scmer {
		return scm.Eval(scm.Read(t.Name(), "(lambda (old new session tx) (observed))", false), env)
	}), false, false, true)
	t.Cleanup(func() {
		triggerLanguageCompilers.Lock()
		delete(triggerLanguageCompilers.byName, language)
		triggerLanguageCompilers.Unlock()
	})
	declaration := statementAdmissionTransaction(false)
	defer declaration.Rollback()
	createStatementTrigger(tbl, "statement_admission_update", AfterStatement, language,
		"observe completed update", []string{"update"}, false, false, declaration)
	if err := declaration.Commit(); err != nil {
		t.Fatal(err)
	}
	return tbl, observed
}

func statementAdmissionUpdate(tbl *table, tx *TxContext, id, value int64) scm.Scmer {
	return withTableMutation([]*table{tbl}, nativeMutationDescriptor{
		event: nativeMutationUpdate, columns: []string{"value"},
	}, tx, tx.Session, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		binding := scmerToTxContext(args[0])
		return tbl.scan(binding, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil,
			[]string{"id"}, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
				return scm.NewBool(args[0].Int() == id)
			}), []string{"$update"}, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
				scm.Apply(args[1], scm.NewSlice([]scm.Scmer{scm.NewString("value"), scm.NewInt(value)}))
				return scm.NewInt(args[0].Int() + 1)
			}), scm.NewInt(0), scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
				return scm.NewInt(args[0].Int() + args[1].Int())
			}), false)
	}))
}

func assertStatementAdmissionRows(t *testing.T, tbl *table, tx *TxContext, first, second int64) {
	t.Helper()
	var mu sync.Mutex
	rows := make(map[int64]int64)
	tbl.scan(tx, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil,
		scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }),
		[]string{"id", "value"}, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
			mu.Lock()
			defer mu.Unlock()
			if _, exists := rows[args[1].Int()]; exists {
				t.Error("duplicate visible row")
			}
			rows[args[1].Int()] = args[2].Int()
			return scm.NewNil()
		}), scm.NewNil(), scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewNil() }), false)
	if len(rows) != 2 || rows[1] != first || rows[2] != second {
		t.Fatalf("full visible rows = %v; want {1:%d, 2:%d}", rows, first, second)
	}
}

func TestImplicitStatementWaitsBeforeItsWriterLease(t *testing.T) {
	for _, cancelWait := range []bool{false, true} {
		name := "release"
		if cancelWait {
			name = "cancel"
		}
		t.Run(name, func(t *testing.T) {
			tbl, observed := statementAdmissionFixture(t)
			writer := statementAdmissionTransaction(false)
			if count := statementAdmissionUpdate(tbl, writer, 2, 21).Int(); count != 1 {
				t.Fatalf("original update affected %d rows", count)
			}
			owner := &tbl.writeAdmission
			owner.mu.Lock()
			retained := owner.statement == nil && owner.writers[writer.transactionState].leases > 0
			owner.mu.Unlock()
			if !retained || writer.State != TxActive || observed.Load() != 1 {
				writer.Rollback()
				t.Fatal("original producer did not drain while retaining its original writer")
			}
			assertStatementAdmissionRows(t, tbl, writer, 10, 21)
			waiter := statementAdmissionTransaction(true)
			ctx, cancel := context.WithCancel(context.Background())
			probe := &statementAdmissionContext{Context: ctx, waiting: make(chan struct{})}
			waiter.invocationContext = probe
			result, done := make(chan any, 1), make(chan struct{})
			go func() {
				defer close(done)
				var failure any
				func() {
					defer func() { failure = recover() }()
					if count := statementAdmissionUpdate(tbl, waiter, 1, 11).Int(); count != 1 {
						panic("implicit update did not affect exactly one row")
					}
					if err := waiter.Commit(); err != nil {
						panic(err)
					}
				}()
				result <- failure
			}()
			defer func() {
				cancel()
				writer.Rollback()
				select {
				case <-done:
					waiter.Rollback()
				case <-time.After(5 * time.Second):
					t.Error("native waiter did not drain after cancellation")
				}
			}()
			select {
			case failure := <-result:
				t.Fatalf("implicit statement returned before original writer release: %v", failure)
			case <-probe.waiting:
			case <-time.After(5 * time.Second):
				t.Fatal("implicit statement never entered native admission wait")
			}
			owner.mu.Lock()
			cleanWait := owner.changed != nil && owner.statement == nil &&
				owner.writers[waiter.transactionState].leases == 0 && len(owner.writers) == 1
			owner.mu.Unlock()
			if !cleanWait || observed.Load() != 1 {
				t.Fatal("waiting statement retained a raw writer or produced trigger effects")
			}
			if cancelWait {
				cancel()
			} else {
				writer.Rollback()
			}
			select {
			case failure := <-result:
				if cancelWait {
					err, ok := failure.(error)
					if !ok || !errors.Is(err, context.Canceled) {
						t.Fatalf("cancelled admission returned %v", failure)
					}
					owner.mu.Lock()
					absent := owner.writers[waiter.transactionState].leases == 0 && owner.statement == nil
					owner.mu.Unlock()
					if !absent || observed.Load() != 1 {
						t.Fatal("cancelled admission left a writer or trigger effects")
					}
					assertStatementAdmissionRows(t, tbl, writer, 10, 21)
					writer.Rollback()
					assertStatementAdmissionRows(t, tbl, nil, 10, 20)
				} else {
					if failure != nil {
						t.Fatalf("released admission failed: %v", failure)
					}
					if observed.Load() != 2 {
						t.Fatalf("completed statement triggers = %d; want 2", observed.Load())
					}
					assertStatementAdmissionRows(t, tbl, nil, 11, 20)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("native admission did not finish after release/cancellation")
			}
		})
	}
}

func TestOriginalStatementWriterUpgradeStillFailsWithoutWaiting(t *testing.T) {
	tbl, observed := statementAdmissionFixture(t)
	writer := statementAdmissionTransaction(false)
	defer writer.Rollback()
	statementAdmissionUpdate(tbl, writer, 2, 21)
	original := statementAdmissionTransaction(false)
	defer original.Rollback()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	probe := &statementAdmissionContext{Context: ctx, waiting: make(chan struct{})}
	original.invocationContext = probe
	var failure any
	func() {
		defer func() { failure = recover() }()
		statementAdmissionUpdate(tbl, original, 1, 11)
	}()
	if !strings.Contains(fmt.Sprint(failure), "native statement conflicts with an active table writer") {
		t.Fatalf("foreign original writer returned %v; want the existing exclusive-upgrade rejection", failure)
	}
	select {
	case <-probe.waiting:
		t.Fatal("original transaction waited while retaining its own writer")
	default:
	}
	if observed.Load() != 1 {
		t.Fatal("failed original upgrade ran its statement trigger")
	}
	assertStatementAdmissionRows(t, tbl, original, 10, 20)
	assertStatementAdmissionRows(t, tbl, writer, 10, 21)
}

func TestOrdinaryImplicitCursorStabilityWritersRemainParallel(t *testing.T) {
	cleanup := setupGCTest(t)
	defer cleanup()
	CreateDatabase("gcdb", false)
	tbl, _ := CreateTable("gcdb", "ordinary_admission", Memory, false)
	first, second := NewTxContext(TxCursorStability), NewTxContext(TxCursorStability)
	for _, tx := range []*TxContext{first, second} {
		tx.Session = scm.NewSession()
		tx.autoCommit = true
		defer tx.Rollback()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		probe := &statementAdmissionContext{Context: ctx, waiting: make(chan struct{})}
		tx.invocationContext = probe
		if tx.canWaitForImplicitStatement(tbl) {
			t.Fatal("ordinary CS writer selected the exclusive implicit profile")
		}
		withTableMutation([]*table{tbl}, nativeMutationDescriptor{event: nativeMutationUpdate},
			tx, tx.Session, scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }))
		select {
		case <-probe.waiting:
			t.Fatal("ordinary CS writer waited on another ordinary writer")
		default:
		}
	}
	owner := &tbl.writeAdmission
	owner.mu.Lock()
	parallel := len(owner.writers) == 2 && owner.statement == nil &&
		!owner.writers[first.transactionState].implicitStatement && !owner.writers[second.transactionState].implicitStatement
	owner.mu.Unlock()
	if !parallel {
		t.Fatal("ordinary CS writers lost their shared admission")
	}
}

func TestRetainedMutationProfilesRejectForeignStatementUpgrade(t *testing.T) {
	for _, profile := range []string{"dirty-implicit", "retained-callbacks", "multiple-targets", "child"} {
		t.Run(profile, func(t *testing.T) {
			tbl, observed := statementAdmissionFixture(t)
			other, _ := CreateTable("gcdb", "statement_admission_other", Memory, false)
			other.CreateColumn("id", "INT", nil, nil)
			other.CreateColumn("value", "INT", nil, nil)
			other.Insert([]string{"id", "value"}, [][]scm.Scmer{{scm.NewInt(1), scm.NewInt(10)},
				{scm.NewInt(2), scm.NewInt(20)}}, nil, scm.NewNil(), false, nil)
			writer := statementAdmissionTransaction(false)
			defer writer.Rollback()
			statementAdmissionUpdate(tbl, writer, 2, 21)
			original := statementAdmissionTransaction(true)
			ctx, cancel := context.WithCancel(context.Background())
			original.invocationContext = ctx
			defer cancel()
			switch profile {
			case "dirty-implicit":
				statementAdmissionUpdate(other, original, 1, 12)
			case "retained-callbacks":
				release := tbl.admitNativeMutationCallbacks(original)
				defer release()
			}
			result, done := make(chan any, 1), make(chan struct{})
			go func() {
				defer close(done)
				var failure any
				func() {
					defer func() { failure = recover() }()
					switch profile {
					case "multiple-targets":
						withTableMutation([]*table{tbl, other}, nativeMutationDescriptor{event: nativeMutationUpdate},
							original, original.Session, scm.NewFunc(func(...scm.Scmer) scm.Scmer {
								panic("conflicting multi-target producer unexpectedly ran")
							}))
					case "child":
						withTableMutation([]*table{other}, nativeMutationDescriptor{event: nativeMutationUpdate},
							original, original.Session, scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
								return statementAdmissionUpdate(tbl, scmerToTxContext(args[0]), 1, 11)
							}))
					default:
						statementAdmissionUpdate(tbl, original, 1, 11)
					}
				}()
				result <- failure
			}()
			defer func() {
				cancel()
				select {
				case <-done:
					original.Rollback()
				case <-time.After(5 * time.Second):
					t.Error("rejected producer did not drain after cancellation")
				}
			}()
			select {
			case failure := <-result:
				if !strings.Contains(fmt.Sprint(failure), "native statement conflicts with an active table writer") {
					t.Fatalf("%s returned %v; want the existing foreign-writer rejection", profile, failure)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("retained/multi-target/child profile waited for a foreign original writer")
			}
			if observed.Load() != 1 {
				t.Fatal("rejected producer ran a statement trigger")
			}
			assertStatementAdmissionRows(t, tbl, original, 10, 20)
			assertStatementAdmissionRows(t, tbl, writer, 10, 21)
			if profile == "dirty-implicit" {
				assertStatementAdmissionRows(t, other, original, 12, 20)
			}
			original.Rollback()
			assertStatementAdmissionRows(t, other, nil, 10, 20)
		})
	}
}

func TestImplicitStatementClaimExcludesNewForeignRawWriter(t *testing.T) {
	owner := new(tableWriteAdmission)
	first, second := statementAdmissionTransaction(true), statementAdmissionTransaction(true)
	defer first.Rollback()
	defer second.Rollback()
	claim, err := owner.admit(context.Background(), first.transactionState, tableImplicitStatementWriteAdmission, true)
	if err != nil {
		t.Fatal(err)
	}
	defer claim.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	probe := &statementAdmissionContext{Context: ctx, waiting: make(chan struct{})}
	type outcome struct {
		lease *tableWriteLease
		err   error
	}
	result := make(chan outcome, 1)
	go func() {
		lease, err := owner.admit(probe, second.transactionState, tableRowWriteAdmission, true)
		result <- outcome{lease, err}
	}()
	defer func() {
		cancel()
		_ = claim.Close()
		select {
		case value := <-result:
			if value.lease != nil {
				_ = value.lease.Close()
			}
		case <-time.After(5 * time.Second):
			t.Error("foreign raw admission did not drain")
		}
	}()
	select {
	case value := <-result:
		result <- value
		t.Fatalf("raw writer overtook the implicit statement claim: %v", value.err)
	case <-probe.waiting:
	case <-time.After(5 * time.Second):
		t.Fatal("raw writer did not enter the native wait")
	}
	owner.mu.Lock()
	exclusive := len(owner.writers) == 1 && owner.writers[first.transactionState].implicitStatement
	owner.mu.Unlock()
	if !exclusive {
		t.Fatal("implicit statement claim did not exclude the new raw writer")
	}
	_ = claim.Close()
	select {
	case value := <-result:
		if value.err != nil || value.lease == nil {
			result <- value
			t.Fatalf("released raw writer returned %v", value.err)
		}
		result <- value // The single cleanup owner closes the acquired lease.
	case <-time.After(5 * time.Second):
		t.Fatal("raw writer did not resume after the claim was released")
	}
}
