/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestReportCollectorParallelReductionAndExceptionalCompletion(t *testing.T) {
	reducer := NewFunc(func(a ...Scmer) Scmer {
		if a[1].IsNil() {
			return a[0]
		}
		if a[0].IsNil() || a[1].Int() > a[0].Int() {
			return a[1]
		}
		return a[0]
	})
	for _, fail := range []bool{false, true} {
		var published atomic.Int32
		completion := NewFunc(func(a ...Scmer) Scmer {
			if a[0].Int() != 199 || a[1].Bool() == fail {
				t.Error("wrong completed reduction", a[0].Int(), a[1].Bool())
			}
			published.Add(1)
			return NewNil()
		})
		action := NewFunc(func(a ...Scmer) Scmer {
			var workers sync.WaitGroup
			for i := 0; i < 200; i++ {
				workers.Add(1)
				go func(value int) { defer workers.Done(); Apply(a[0], NewInt(int64(value))) }(i)
			}
			workers.Wait()
			if fail {
				panic("fixture action failed")
			}
			return NewInt(7)
		})
		if fail {
			expectTDSPanic(t, func() { CollectReports(action, NewNil(), reducer, completion) })
		} else if CollectReports(action, NewNil(), reducer, completion).Int() != 7 {
			t.Fatal("action return changed")
		}
		if published.Load() != 1 {
			t.Fatal("completion was not once")
		}
	}
}
