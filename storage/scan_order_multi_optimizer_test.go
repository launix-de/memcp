/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.
*/
package storage

import (
	"strings"
	"testing"

	"github.com/launix-de/memcp/scm"
)

func optimizeScanOrderMultiTestSource(t *testing.T, reducers string) string {
	t.Helper()
	Init(scm.Globalenv)
	source := `(scan_order_multi nil
		(list nil nil) (list nil nil) (list)
		(list (list) (list)) (list (lambda () true) (lambda () true))
		(list (list) (list)) (list)
		nil nil 0 0 10
		(list (list "key") (list "key")) ` + reducers + `
		(list) false)`
	optimized := scm.Optimize(scm.Read(t.Name(), source), &scm.Globalenv, nil)
	return scm.SerializeToString(optimized, &scm.Globalenv)
}

func TestScanOrderMultiOptimizesEveryOwnedReducer(t *testing.T) {
	serialized := optimizeScanOrderMultiTestSource(t, `(list
		(lambda (acc key) (set_assoc acc key 1))
		(lambda (acc key) (set_assoc acc key 2)))`)
	if count := strings.Count(serialized, "set_assoc_mut"); count != 2 {
		t.Fatalf("optimized %d of 2 owned reducers: %s", count, serialized)
	}
	if strings.Contains(serialized, "(set_assoc ") {
		t.Fatalf("copying assoc update remains in multi-source reducer: %s", serialized)
	}
}

func TestScanOrderMultiRejectsOwnershipUnlessEveryReducerPreservesIt(t *testing.T) {
	serialized := optimizeScanOrderMultiTestSource(t, `(list
		(lambda (acc key) (set_assoc acc key 1))
		(lambda (acc key) key))`)
	if strings.Contains(serialized, "set_assoc_mut") {
		t.Fatalf("owned update escaped across a borrowing sibling reducer: %s", serialized)
	}
}

func TestScanOrderMultiKeepsDynamicReducerList(t *testing.T) {
	serialized := optimizeScanOrderMultiTestSource(t, `runtime_reducers`)
	if !strings.Contains(serialized, "runtime_reducers") {
		t.Fatalf("dynamic reducer list was mistaken for a static callback list: %s", serialized)
	}
}

func TestScanOrderMultiTransfersDefaultResultToOuterReducer(t *testing.T) {
	Init(scm.Globalenv)
	scan := `(scan_order_multi nil
		(list nil nil) (list nil nil) (list)
		(list (list) (list)) (list (lambda () true) (lambda () true))
		(list (list) (list)) (list)
		nil nil 0 0 10
		(list (list "key") (list "key"))
		(list
			(lambda (acc key) (set_assoc acc key 1))
			(lambda (acc key) (set_assoc acc key 2)))
		(list) false)`
	source := `(set_assoc ` + scan + ` "outer" true)`
	optimized := scm.Optimize(scm.Read(t.Name(), source), &scm.Globalenv, nil)
	serialized := scm.SerializeToString(optimized, &scm.Globalenv)
	if count := strings.Count(serialized, "set_assoc_mut"); count != 3 {
		t.Fatalf("nested multi-source result lost ownership; optimized %d of 3 updates: %s", count, serialized)
	}
}

func optimizeNestedScanOrderTestSource(t *testing.T, notFound string) string {
	t.Helper()
	Init(scm.Globalenv)
	scan := `(scan_order nil nil (list) (list)
		(list) (lambda () true) (list) (list)
		0 0 10 (list "key")
		(lambda (acc key) (set_assoc acc key 1))
		(list) false` + notFound + `)`
	optimized := scm.Optimize(scm.Read(t.Name(), `(set_assoc `+scan+` "outer" true)`), &scm.Globalenv, nil)
	return scm.SerializeToString(optimized, &scm.Globalenv)
}

func TestScanOrderTransfersDefaultResultToOuterReducer(t *testing.T) {
	serialized := optimizeNestedScanOrderTestSource(t, "")
	if count := strings.Count(serialized, "set_assoc_mut"); count != 2 {
		t.Fatalf("nested ordered result lost ownership; optimized %d of 2 updates: %s", count, serialized)
	}
}

func TestScanOrderKeepsCustomNotFoundResultBorrowed(t *testing.T) {
	serialized := optimizeNestedScanOrderTestSource(t, ` '("existing" true)`)
	if count := strings.Count(serialized, "set_assoc_mut"); count != 1 {
		t.Fatalf("custom no-hit result transferred ownership; optimized %d of 1 safe updates: %s", count, serialized)
	}
}

func optimizeNestedScanOrderBatchAcceptTestSource(t *testing.T, notFound string) string {
	t.Helper()
	Init(scm.Globalenv)
	scan := `(scan_order_batch_accept nil nil (list) (list)
		(lambda (rows) rows) (list) (list)
		0 0 10 (list "key")
		(lambda (acc key) (set_assoc acc key 1))
		(list) false` + notFound + `)`
	optimized := scm.Optimize(scm.Read(t.Name(), `(set_assoc `+scan+` "outer" true)`), &scm.Globalenv, nil)
	return scm.SerializeToString(optimized, &scm.Globalenv)
}

func TestScanOrderBatchAcceptTransfersDefaultResultToOuterReducer(t *testing.T) {
	serialized := optimizeNestedScanOrderBatchAcceptTestSource(t, "")
	if count := strings.Count(serialized, "set_assoc_mut"); count != 2 {
		t.Fatalf("nested batch-accept result lost ownership; optimized %d of 2 updates: %s", count, serialized)
	}
}

func TestScanOrderBatchAcceptKeepsCustomNotFoundResultBorrowed(t *testing.T) {
	serialized := optimizeNestedScanOrderBatchAcceptTestSource(t, ` '("existing" true)`)
	if count := strings.Count(serialized, "set_assoc_mut"); count != 1 {
		t.Fatalf("custom batch-accept no-hit result transferred ownership; optimized %d of 1 safe updates: %s", count, serialized)
	}
}

func optimizeNestedScanJoinOrderTestSource(t *testing.T, suffix string) string {
	t.Helper()
	Init(scm.Globalenv)
	scan := `(scan_join_order nil
		(list nil) (list nil) (list)
		(list (list)) (list (lambda () true)) (list)
		(list) nil (list) (list)
		0 0 10 (list (list 0 "key"))
		(lambda (acc key) (set_assoc acc key 1))
		(list)` + suffix + `)`
	optimized := scm.Optimize(scm.Read(t.Name(), `(set_assoc `+scan+` "outer" true)`), &scm.Globalenv, nil)
	return scm.SerializeToString(optimized, &scm.Globalenv)
}

func TestScanJoinOrderTransfersDefaultResultToOuterReducer(t *testing.T) {
	serialized := optimizeNestedScanJoinOrderTestSource(t, "")
	if count := strings.Count(serialized, "set_assoc_mut"); count != 2 {
		t.Fatalf("nested join-order result lost ownership; optimized %d of 2 updates: %s", count, serialized)
	}
}

func TestScanJoinOrderKeepsCustomNotFoundResultBorrowed(t *testing.T) {
	serialized := optimizeNestedScanJoinOrderTestSource(t, ` nil false '("existing" true) false`)
	if count := strings.Count(serialized, "set_assoc_mut"); count != 1 {
		t.Fatalf("custom join-order no-hit result transferred ownership; optimized %d of 1 safe updates: %s", count, serialized)
	}
}
