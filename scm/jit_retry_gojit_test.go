//go:build goexperiment.jit && amd64

// Copyright (C) 2026 MemCP Contributors
// SPDX-License-Identifier: GPL-3.0-or-later
package scm

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

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
