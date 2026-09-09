//go:build php

// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later
package phpbridge

import "sync"
import "strings"
import "testing"
import "github.com/launix-de/memcp/scm"

func TestResultBufferConcurrentRows(t *testing.T) {
	var b resultBuffer
	b.captureFields(scm.NewSlice([]scm.Scmer{scm.NewSlice([]scm.Scmer{scm.NewString("id"), scm.NewString("BIGINT"), scm.NewString("bin")}), scm.NewString("optional")}))
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				b.captureRow(scm.NewSlice([]scm.Scmer{scm.NewString("id"), scm.NewInt(int64(worker*1000 + i))}))
			}
		}(worker)
	}
	wg.Wait()
	b.mu.Lock()
	if b.rowCount != 8000 {
		t.Errorf("rows: %d", b.rowCount)
	}
	seen := make(map[int64]bool)
	for i := 2; i < len(b.cells); i += 2 {
		id := int64(b.cells[i].integer)
		if seen[id] {
			t.Errorf("duplicate %d", id)
		}
		seen[id] = true
		if b.cells[i+1].kind != 0 {
			t.Error("missing column must be NULL")
		}
	}
	b.mu.Unlock()
	b.release()
	if len(b.cells) != 0 || b.cells != nil || b.metadata || len(b.columns) != 0 {
		t.Fatal("large result retained or metadata leaked")
	}
	b.captureFields(scm.NewSlice([]scm.Scmer{scm.NewString("next")}))
	b.captureRow(scm.NewSlice([]scm.Scmer{scm.NewString("next"), scm.NewString("clean")}))
	if string(b.data) != "nextclean" || b.rowCount != 1 || b.columnCount != 1 {
		t.Fatal("result state leaked across reuse")
	}
}

func TestResultBufferBoundedReuse(t *testing.T) {
	var b resultBuffer
	b.captureRow(scm.NewSlice([]scm.Scmer{scm.NewString("value"), scm.NewString(strings.Repeat("x", retainedResultBytes+1))}))
	b.release()
	if b.data != nil {
		t.Fatal("oversized byte buffer retained")
	}
	b.captureRow(scm.NewSlice([]scm.Scmer{scm.NewString("value"), scm.NewNil()}))
	if b.cells[1].kind != 0 {
		t.Fatal("stale data in NULL cell")
	}
	b.release()
	if cap(b.cells) == 0 {
		t.Fatal("small cell buffer not reused")
	}
}

func TestResultBufferDeclaredPackingAndDuplicateNames(t *testing.T) {
	var b resultBuffer
	b.captureFields(scm.NewSlice([]scm.Scmer{
		scm.NewSlice([]scm.Scmer{scm.NewString("x"), scm.NewString("BIGINT"), scm.NewString("bin")}),
		scm.NewSlice([]scm.Scmer{scm.NewString("x"), scm.NewString("VARCHAR"), scm.NewString("utf8mb4_unicode_520_ci")}),
	}))
	b.captureRow(scm.NewSlice([]scm.Scmer{scm.NewString("x"), scm.NewFloat(12), scm.NewString("x"), scm.NewInt(34)}))
	if b.cells[2].kind != 1 || b.cells[2].integer != 12 || b.cells[3].kind != 3 {
		t.Fatal("compiler types or duplicate positions lost during packing")
	}
	cell := b.cells[3]
	if string(b.data[int(cell.offset):int(cell.offset+cell.length)]) != "34" {
		t.Fatal("declared string was not packed as text")
	}
}
