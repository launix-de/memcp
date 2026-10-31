//go:build goexperiment.jit && (amd64 || arm64 || riscv64)

// Copyright (C) 2026 MemCP Contributors
// SPDX-License-Identifier: GPL-3.0-or-later
package scm

import (
	"fmt"

	"time"

	"runtime"
	"strings"
	"testing"
)

func TestJITCompilationTrimsReservation(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, `(lambda (value) (+ value 1))`)
	entry := compiled.Proc().Compiled
	want := (entry.CodeLen + 15) &^ 15
	if entry.reservation.size != want {
		t.Fatalf("reservation = %d bytes, want %d for %d bytes of code", entry.reservation.size, want, entry.CodeLen)
	}
	runtime.KeepAlive(compiled)
}

// Return a different Proc sharing the code owner. Dropping the source Proc
// must not release code while this copy is callable; dropping the copy must
// allow the existing JITEntryPoint cleanup to collect the Proc/entry cycle.
//
//go:noinline
func jitCleanupProcCopy(t *testing.T) (Scmer, uintptr, <-chan struct{}) {
	compiled := compileJITExpressionTestProc(t, `(lambda (value) (+ value 1))`)
	entry := compiled.Proc().Compiled
	done := make(chan struct{})
	runtime.AddCleanup(entry, func(ch chan struct{}) { close(ch) }, done)
	return NewProc(compiled.Proc()), uintptr(entry.CodePtr), done
}

func TestJITProcCopiesRetainAndReleaseCodeOwner(t *testing.T) {
	copy, code, done := jitCleanupProcCopy(t)
	for round := 0; round < 2; round++ {
		runtime.GC()
		if got := Apply(copy, NewInt(41)).Int(); got != 42 {
			t.Fatalf("shared code result = %d, want 42", got)
		}
		select {
		case <-done:
			t.Fatal("code owner collected while a Proc copy remained reachable")
		default:
		}
	}
	runtime.KeepAlive(copy)
	copy = NewNil()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		runtime.GC()
		select {
		case <-done:
			if _, retained := jitNativeCodes.Load(code); !retained {
				return
			}
		default:
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("code owner or native-code registration survived the last Proc copy")
}

// Wide nested callbacks model the captured aggregate callbacks emitted by a
// historical metric query. Every parent must outgrow the initial code buffer.
func jitRetryNestedSource(depth int) string {
	body := "(+ x0 x1 x2)"
	for level := depth - 1; level >= 0; level-- {
		var out strings.Builder
		fmt.Fprintf(&out, "(lambda (x%d) (list %s", level, body)
		for i := 0; i < 512; i++ {
			fmt.Fprintf(&out, " (+ x0 x%d %d)", level, i)
		}
		out.WriteString("))")
		body = out.String()
	}
	return body
}

func TestJITNestedBufferRetryCaptures(t *testing.T) {
	compiled := compileJITExpressionTestProc(t, jitRetryNestedSource(3))
	if compiled.Proc().Compiled.CodeLen <= 16*1024 {
		t.Fatal("fixture did not overflow the initial code buffer")
	}
	value := compiled
	for level := 0; level < 3; level++ {
		runtime.GC()
		row := Apply(value, NewInt(int64(level+1))).Slice()
		if len(row) != 513 || row[512].Int() != int64(1+level+1+511) {
			t.Fatalf("level %d: invalid captured row", level)
		}
		value = row[0]
		if level < 2 && (value.Proc() == nil || value.Proc().Compiled == nil) {
			t.Fatalf("level %d: callback not native", level)
		}
	}
	if value.Int() != 6 {
		t.Fatalf("nested result = %v", value)
	}
}

func BenchmarkJITNestedBufferRetry(b *testing.B) {
	expression := Optimize(Read("nested-retry", jitRetryNestedSource(3)), &Globalenv, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		compiled := jitCompile(Eval(expression, &Globalenv))
		if compiled.Proc().Compiled == nil {
			b.Fatal("native compilation failed")
		}
	}
}
