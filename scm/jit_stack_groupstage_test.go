//go:build goexperiment.jit && amd64

/*
Copyright (C) 2026  MemCP Contributors

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

package scm

import (
	"fmt"
	"math/bits"
	"runtime"
	"strings"
	"testing"
	"unsafe"
)

// jitDecodeSafepointRoots expands one safepoint's per-base bitmaps back into
// concrete (base, offset) roots, mirroring the decoding finalizeStackMaps
// performs when converting them into the runtime pointer map.
func jitDecodeSafepointRoots(sp jitSafepoint) []jitStackRoot {
	var out []jitStackRoot
	for base, bitmap := range sp.roots {
		for index, value := range bitmap.bits {
			for value != 0 {
				bit := bits.TrailingZeros8(value)
				out = append(out, jitStackRoot{
					base:   jitStackRootBase(base),
					offset: bitmap.first + int32(index*64+bit*8),
				})
				value &= value - 1
			}
		}
	}
	return out
}

// TestJITGroupStageReducerFrameRootsCoverEverySafepoint guards against issue
// #949 (a suspected recurrence of #745's "found pointer to free object" class
// of bug) by mechanically checking an invariant instead of waiting for a GC
// to catch a stale pointer by chance. The shape below mirrors the real
// group-stage SUM reducer compiled for TPC-H Q10's ".grp:query:" keytable, as
// closely as a standalone expression can: a fused filter+reduce pipeline
// whose reduce step allocates (merge_assoc_mut with a freshly compiled
// closure argument) only when the filter predicate admits the element -- the
// same "trivial skip path beside an allocating path" shape as the real
// reducer's __join_scan_reduce_skip branch.
//
// jitCompileExprBodyToExec's prologue only zero-initializes the frame
// (jitSortedFrameRoots(ctx.FrameRoots)) for permanent frame words: BP-relative
// roots and non-negative-offset SP-relative roots. Every root any safepoint
// actually marks as live must be a member of that same set, or a call that
// takes the trivial branch will scan an uninitialized (possibly stale, from a
// prior invocation on the same reused goroutine stack) word as a pointer.
//
// As of this writing this fixture passes: the gap #949 is tracking has not
// been narrowed down to a standalone-expression repro yet (see the issue for
// what was ruled out). Keep this test as a regression guard for the shape
// that was checked, and extend it (or add siblings) once a failing shape is
// found.
func TestJITGroupStageReducerFrameRootsCoverEverySafepoint(t *testing.T) {
	var captured *JITContext
	jitTestPostEmitHook = func(ctx *JITContext) { captured = ctx }
	defer func() { jitTestPostEmitHook = nil }()

	compileJITExpressionTestProc(t, `(lambda (values)
		(reduce (filter values (lambda (value) (> value 1)))
			(lambda (acc value) (merge_assoc_mut acc (list value value) (lambda (old new) new)))
			'()))`)

	if captured == nil {
		t.Fatal("post-emit hook did not observe a compiled context")
	}
	if len(captured.Safepoints) == 0 {
		t.Fatal("fixture recorded no safepoints; it no longer exercises a Go call boundary")
	}

	for spIndex, sp := range captured.Safepoints {
		for _, root := range jitDecodeSafepointRoots(sp) {
			permanent := root.base == jitStackRootFrameBP ||
				(root.base == jitStackRootFrameSP && root.offset >= 0)
			if !permanent {
				// Dynamic call-area roots (jitStackRootCallSP, or negative-offset
				// FrameSP slots below the stable stack pointer) are initialized at
				// their call site, not by the frame-entry zeroing loop.
				continue
			}
			if _, ok := captured.FrameRoots[root]; !ok {
				t.Fatalf("safepoint %d marks %v as a live GC root, but it is absent from "+
					"FrameRoots and will not be zero-initialized at function entry -- a call "+
					"that skips the path setting it will scan stale stack data as a pointer",
					spIndex, root)
			}
		}
	}
}

// Escaping closures must publish captured heap pointers through write barriers,
// including when the closure allocation is black during concurrent marking.
func TestJITCapturedPairSurvivesGC(t *testing.T) {
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
				runtime.GC()
			}
		}
	}()
	defer func() { close(stop); <-done }()

	compiled := compileJITExpressionTestProc(t, `(lambda (a b) (lambda (value) (list a b value)))`)
	for round := 0; round < 100; round++ {
		closures := make([]Scmer, 10000)
		for i := range closures {
			closures[i] = Apply(compiled, NewString(strings.Repeat("a", 64)), NewString(strings.Repeat("b", 64)))
		}
		runtime.GC()
		for _, closure := range closures {
			got := Apply(closure, NewInt(7))
			if !got.IsSlice() || len(got.Slice()) != 3 || !Equal(got.Slice()[0], NewString(strings.Repeat("a", 64))) || !Equal(got.Slice()[1], NewString(strings.Repeat("b", 64))) || !Equal(got.Slice()[2], NewInt(7)) {
				t.Fatal("corrupted closure")
			}
		}
	}
}

func TestJITWideEscapingClosures(t *testing.T) {
	for n := 1; n <= 40; n++ {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			names := make([]string, n)
			args := make([]Scmer, n)
			for i := range names {
				names[i] = fmt.Sprintf("a%d", i)
				args[i] = NewString(fmt.Sprintf("value-%d", i))
			}
			compiled := compileJITExpressionTestProc(t, "(lambda ("+strings.Join(names, " ")+") (lambda () (list "+strings.Join(names, " ")+")))")
			result := Apply(Apply(compiled, args...))
			if !Equal(result, NewSlice(args)) {
				t.Fatalf("got %s", String(result))
			}
		})
	}
}

func TestJITRecursiveClosureRetainsInlineContext(t *testing.T) {
	template := compileJITExpressionTestProc(t, `(lambda (value) value)`).Proc()
	captures := []Scmer{NewString("captured"), NewNil()}
	bound := jitBindProcContext(jitProcContextAllocation(2), template, &captures[0], 2, true)
	context := unsafe.Slice((*Scmer)(unsafe.Add(unsafe.Pointer(bound), unsafe.Offsetof(ProcJIT{}.Context))), 2)
	if context[1].Proc() != bound {
		t.Fatal("recursive binding copied the header without its inline captures")
	}
	if !Equal(context[0], captures[0]) {
		t.Fatal("capture was lost")
	}
}
