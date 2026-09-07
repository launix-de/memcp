/*
Copyright (C) 2026  Carl-Philip Hänsch

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
package storage

import (
	"runtime"
	"strings"
	"testing"

	"github.com/launix-de/memcp/scm"
)

//go:noinline
func growJITBoundaryTestStack(depth int) byte {
	var frame [512]byte
	frame[0] = byte(depth)
	if depth > 0 {
		frame[0] ^= growJITBoundaryTestStack(depth - 1)
	}
	runtime.KeepAlive(&frame)
	return frame[0]
}

func TestJITPreservesAdjacentScanBoundaryResults(t *testing.T) {
	if !scm.JITEnabled() {
		t.Skip("requires GOEXPERIMENT=jit")
	}
	env := scm.Env{Vars: scm.Vars{}, Outer: &scm.Globalenv}
	scm.Declare(&env, &scm.Declaration{
		Name: "jit_test_scan_boundary",
		Fn: func(args ...scm.Scmer) scm.Scmer {
			if len(args) != 8 {
				panic("jit_test_scan_boundary expects eight arguments")
			}
			return newScanBoundarySpec(scm.String(args[1]), EqualMatcher,
				scm.ToInt(args[2]), scm.ToInt(args[3]), args[4].Bool(), args[5].Bool(), scm.String(args[6]), args[7].Bool(),
				-1, nil, nil, "", false)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Params: []*scm.TypeDescriptor{
			{Kind: "string"}, {Kind: "string"}, {Kind: "number"}, {Kind: "number"},
			{Kind: "bool"}, {Kind: "bool"}, {Kind: "string"}, {Kind: "bool"},
		}, Return: &scm.TypeDescriptor{Kind: "any"}},
	})
	source := `(lambda ()
		(list 369436443803648
			(jit_test_scan_boundary "equal" "database" 0 0 true true "" false)
			(jit_test_scan_boundary "equal" "name" 1 1 true true "" false)))`
	proc := scm.Eval(scm.Optimize(scm.Read(t.Name(), source), &env, nil), &env)
	interpreted := scm.Apply(proc).Slice()
	for index := 1; index < len(interpreted); index++ {
		if !interpreted[index].IsCustom(TagScanBoundary) {
			t.Fatalf("interpreted schema item %d is %s, want scan boundary", index, scm.String(interpreted[index]))
		}
	}
	compiled := scm.CompileJIT(proc, true)
	items := scm.Apply(compiled).Slice()
	if len(items) != 3 {
		t.Fatalf("JIT returned %d schema items, want 3", len(items))
	}
	for index := 1; index < len(items); index++ {
		if !items[index].IsCustom(TagScanBoundary) {
			t.Fatalf("schema item %d is %s, want scan boundary", index, scm.String(items[index]))
		}
	}

	scm.Declare(&env, &scm.Declaration{
		Name: "jit_test_consume_boundary_schema",
		Fn: func(args ...scm.Scmer) scm.Scmer {
			items := args[0].Slice()
			runtime.GC()
			return scm.NewBool(len(items) == 3 && items[1].IsCustom(TagScanBoundary) && items[2].IsCustom(TagScanBoundary))
		},
		Type: &scm.TypeDescriptor{Kind: "func", Params: []*scm.TypeDescriptor{
			{Kind: "list", NoEscape: true},
		}, Return: &scm.TypeDescriptor{Kind: "bool"}},
	})
	consumerSource := `(lambda ()
		(jit_test_consume_boundary_schema
			(list 369436443803648
				(jit_test_scan_boundary "equal" "database" 0 0 true true "" false)
				(jit_test_scan_boundary "equal" "name" 1 1 true true "" false))))`
	consumerOptimized := scm.Optimize(scm.Read(t.Name()+" consumer", consumerSource), &env, nil)
	if serialized := scm.SerializeToString(consumerOptimized, &env); !strings.Contains(serialized, "(!list ") {
		t.Fatalf("same-goroutine noescape list was not stack-lowered: %s", serialized)
	}
	consumer := scm.Eval(consumerOptimized, &env)
	compiledConsumer := scm.CompileJIT(consumer, true)
	if !scm.Apply(compiledConsumer).Bool() {
		t.Fatal("GC lost a scan boundary in a noescape JIT-frame list")
	}

	scm.Declare(&env, &scm.Declaration{
		Name: "jit_test_consume_boundary_schema_parallel",
		Fn: func(args ...scm.Scmer) scm.Scmer {
			items := args[0].Slice()
			start := make(chan struct{})
			valid := make(chan bool, 1)
			go func() {
				<-start
				runtime.GC()
				valid <- len(items) == 3 && items[1].IsCustom(TagScanBoundary) && items[2].IsCustom(TagScanBoundary)
			}()
			_ = growJITBoundaryTestStack(128)
			runtime.GC()
			close(start)
			return scm.NewBool(<-valid)
		},
		Type: &scm.TypeDescriptor{Kind: "func", Params: []*scm.TypeDescriptor{
			{Kind: "list", NoEscape: true, CrossGoroutine: true},
		}, Return: &scm.TypeDescriptor{Kind: "bool"}},
	})
	parallelSource := `(lambda ()
		(jit_test_consume_boundary_schema_parallel
			(list 369436443803648
				(jit_test_scan_boundary "equal" "database" 0 0 true true "" false)
				(jit_test_scan_boundary "equal" "name" 1 1 true true "" false))))`
	parallelOptimized := scm.Optimize(scm.Read(t.Name()+" parallel", parallelSource), &env, nil)
	if serialized := scm.SerializeToString(parallelOptimized, &env); strings.Contains(serialized, "(!list ") {
		t.Fatalf("cross-goroutine noescape list was stack-lowered: %s", serialized)
	}
	parallel := scm.Eval(parallelOptimized, &env)
	compiledParallel := scm.CompileJIT(parallel, true)
	if !scm.Apply(compiledParallel).Bool() {
		t.Fatal("parallel GC lost a scan boundary in a noescape JIT-frame list")
	}
}
