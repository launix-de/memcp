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
