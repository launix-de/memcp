//go:build goexperiment.jit && amd64

// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later

package scm

import (
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
	"unsafe"
)

func TestJITEmptyStringDoesNotPointPastInput(t *testing.T) {
	for _, test := range []struct{ name, source string }{
		{"regex capture", `(lambda (value) (match value (regex "a*()$" all empty) empty _ false))`},
		{"substring", `(lambda (value) (substr value (strlen value)))`},
		{"parser remainder", `(lambda (value) ((parser '((atom "aaaa" false) (define rest (regex "a*" false false)) $) rest "") value))`},
	} {
		t.Run(test.name, func(t *testing.T) {
			compiled := compileJITExpressionTestProc(t, test.source)
			input := strings.Repeat("a", 4)
			end := uintptr(unsafe.Pointer(unsafe.StringData(input))) + uintptr(len(input))
			got := Apply(compiled, NewString(input))
			if got.GetTag() != tagString || got.String() != "" {
				t.Fatalf("expected empty string, got %s", String(got))
			}
			if uintptr(unsafe.Pointer(got.ptr)) == end {
				got = NewNil()
				t.Fatal("JIT published one-past-end address as a GC pointer")
			}
			runtime.GC()
			runtime.KeepAlive(got)
			runtime.KeepAlive(input)
		})
	}
}

func TestJITEmptyStringSurvivesGCInCallerFrame(t *testing.T) {
	collect := NewFunc(func(args ...Scmer) Scmer {
		runtime.GC()
		return args[0]
	})
	for _, source := range []string{
		`(lambda (value collect) (match value (regex "a*()$" all empty) (begin (collect empty) empty) _ false))`,
		`(lambda (value collect) (begin (define rest ((parser '((regex "a+" false false) (define rest (regex "a*" false false)) $) rest "") value)) (collect rest) rest))`,
	} {
		compiled := compileJITExpressionTestProc(t, source)
		input := strings.Repeat("a", 64<<10)
		for i := 0; i < 4; i++ {
			got := Apply(compiled, NewString(input), collect)
			if got.GetTag() != tagString || got.String() != "" {
				t.Fatalf("expected empty string after GC, got %s", String(got))
			}
		}
		runtime.KeepAlive(input)
	}
}

func TestJITSQLParameterizationSurvivesGC(t *testing.T) {
	environment := &Env{Vars: make(Vars), Outer: &Globalenv}
	source, err := os.ReadFile("../lib/sql-parameters.scm")
	if err != nil {
		t.Fatal(err)
	}
	EvalAllJIT("sql-parameters.scm", string(source), environment)
	fn := environment.Vars[Symbol("parameterize_sql_select_literals")]
	if fn.Proc().Compiled == nil {
		t.Fatal("SQL parameterization did not compile")
	}
	query := "INSERT INTO test VALUES " + strings.Repeat("(12345),", 10000) + "(12345)"
	previousGCPercent := debug.SetGCPercent(1)
	defer debug.SetGCPercent(previousGCPercent)
	runtime.GC()
	got := Apply(fn, NewString(query))
	if got.GetTag() != tagSlice || len(got.Slice()) != 3 {
		t.Fatalf("unexpected parameterization result")
	}
	items := got.Slice()
	want := "INSERT INTO test VALUES " + strings.Repeat("(?),", 10000) + "(?)"
	if items[0].String() != want || len(items[1].Slice()) != 10001 {
		t.Fatal("parameterization changed SQL shape or literal bindings")
	}
	runtime.KeepAlive(got)
	runtime.KeepAlive(query)
}
