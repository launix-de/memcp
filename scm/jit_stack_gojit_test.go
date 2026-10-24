//go:build goexperiment.jit && (amd64 || arm64 || riscv64)

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
	"bytes"
	"context"
	"fmt"
	"math/bits"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"
)

func jitEntryCode(entry *JITEntryPoint) []byte {
	return unsafe.Slice((*byte)(entry.CodePtr), entry.CodeLen)
}

func TestJITFrameRootInitializationIsPreciseAndDeterministic(t *testing.T) {
	ctx := JITContext{}
	for _, root := range []jitStackRoot{
		{base: jitStackRootCallSP, offset: 16},
		{base: jitStackRootFrameBP, offset: -8},
		{base: jitStackRootFrameSP, offset: -16},
		{base: jitStackRootFrameSP, offset: 32},
		{base: jitStackRootFrameSP, offset: 32},
		{base: jitStackRootFrameBP, offset: -24},
	} {
		ctx.setStackPointer(root.base, root.offset, true)
	}
	want := []jitStackRoot{
		{base: jitStackRootFrameSP, offset: 32},
		{base: jitStackRootFrameBP, offset: -24},
		{base: jitStackRootFrameBP, offset: -8},
	}
	got := jitSortedFrameRoots(ctx.FrameRoots)
	if len(got) != len(want) {
		t.Fatalf("frame roots = %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("frame root %d = %v, want %v", index, got[index], want[index])
		}
	}
}

func TestJITScalarPointerSpillRemainsInStackMap(t *testing.T) {
	code := make([]byte, 512)
	start := unsafe.Pointer(&code[0])
	ctx := &JITContext{
		Start:      start,
		Ptr:        start,
		End:        unsafe.Add(start, len(code)-1),
		AllRegs:    1 << uint(RegRAX),
		AllFPRegs:  1 << uint(RegX2),
		FreeFPRegs: 1 << uint(RegX2),
		FrameReg:   RegRBP,
		StackReg:   RegRSP,
		ScratchReg: RegR11,
	}
	value := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: RegRAX, RelocatablePointer: true}
	ctx.BindReg(RegRAX, &value)

	if got := ctx.AllocReg(); got != RegRAX {
		t.Fatalf("spilled register %d, want %d", got, RegRAX)
	}
	root := jitStackRoot{base: jitStackRootFrameBP, offset: -8}
	if _, ok := ctx.StackRoots[root]; !ok {
		t.Fatal("relocatable scalar spill is missing from the stack map")
	}
}

func TestJITUnprovenScalarDoesNotUseFPOverflowHome(t *testing.T) {
	code := make([]byte, 512)
	start := unsafe.Pointer(&code[0])
	ctx := &JITContext{
		Start:      start,
		Ptr:        start,
		End:        unsafe.Add(start, len(code)-1),
		AllRegs:    1 << uint(RegRAX),
		AllFPRegs:  1 << uint(RegX2),
		FreeFPRegs: 1 << uint(RegX2),
		FrameReg:   RegRBP,
		StackReg:   RegRSP,
		ScratchReg: RegR11,
	}
	value := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: RegRAX}
	ctx.BindReg(RegRAX, &value)

	if got := ctx.AllocReg(); got != RegRAX {
		t.Fatalf("reclaimed register %d, want %d", got, RegRAX)
	}
	ctx.SyncDesc(&value)
	if value.Loc != LocStack {
		t.Fatalf("unproven scalar spilled to loc %d, want stack", value.Loc)
	}
	if ctx.FreeFPRegs&(1<<uint(RegX2)) == 0 {
		t.Fatal("unproven scalar consumed an FP overflow home")
	}
}

func TestJITPointerFreeScalarUsesFPOverflowHome(t *testing.T) {
	code := make([]byte, 512)
	start := unsafe.Pointer(&code[0])
	ctx := &JITContext{
		Start:      start,
		Ptr:        start,
		End:        unsafe.Add(start, len(code)-1),
		AllRegs:    1 << uint(RegRAX),
		AllFPRegs:  1 << uint(RegX2),
		FreeFPRegs: 1 << uint(RegX2),
		FrameReg:   RegRBP,
		StackReg:   RegRSP,
		ScratchReg: RegR11,
	}
	value := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: RegRAX, NoHeapPointer: true}
	ctx.BindReg(RegRAX, &value)

	if got := ctx.AllocReg(); got != RegRAX {
		t.Fatalf("reclaimed register %d, want %d", got, RegRAX)
	}
	ctx.SyncDesc(&value)
	if value.Loc != LocFPReg || value.Reg != RegX2 {
		t.Fatalf("scalar overflow home = loc %d reg %d, want XMM register %d", value.Loc, value.Reg, RegX2)
	}
	if ctx.MaxSpillOffset != 0 || len(ctx.StackRoots) != 0 {
		t.Fatalf("register overflow used stack: spill=%d roots=%v", ctx.MaxSpillOffset, ctx.StackRoots)
	}

	// The allocator returned RAX to its caller. Once that temporary dies, an
	// integer consumer must recover the original payload without boxing it.
	ctx.FreeReg(RegRAX)
	ctx.EnsureDesc(&value)
	if value.Loc != LocReg || value.Reg != RegRAX {
		t.Fatalf("restored scalar = loc %d reg %d, want GPR %d", value.Loc, value.Reg, RegRAX)
	}
}

func TestJITFreeDescReleasesScalarFPOverflowHome(t *testing.T) {
	code := make([]byte, 512)
	start := unsafe.Pointer(&code[0])
	ctx := &JITContext{
		Start:      start,
		Ptr:        start,
		End:        unsafe.Add(start, len(code)-1),
		AllRegs:    1 << uint(RegRAX),
		AllFPRegs:  1 << uint(RegX2),
		FreeFPRegs: 1 << uint(RegX2),
		FrameReg:   RegRBP,
		StackReg:   RegRSP,
		ScratchReg: RegR11,
	}
	value := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: RegRAX, NoHeapPointer: true}
	ctx.BindReg(RegRAX, &value)
	staleAlias := value
	_ = ctx.AllocReg()

	ctx.FreeDesc(&staleAlias)
	if ctx.FreeFPRegs&(1<<uint(RegX2)) == 0 || ctx.RegOwners[RegX2] != nil {
		t.Fatal("freeing a stale descriptor alias leaked its FP overflow home")
	}
}

func newJITStackMapTestContext(code []byte) *JITContext {
	start := unsafe.Pointer(&code[0])
	return &JITContext{
		Start:      start,
		Ptr:        start,
		End:        unsafe.Add(start, len(code)-1),
		AllRegs:    1<<uint(RegRAX) | 1<<uint(RegRBX),
		FreeRegs:   1<<uint(RegRAX) | 1<<uint(RegRBX),
		FrameReg:   RegRBP,
		StackReg:   RegRSP,
		ScratchReg: RegR11,
	}
}

func TestJITKnownSliceHeaderWithTwoRegistersKeepsPreciseRoot(t *testing.T) {
	ctx := newJITStackMapTestContext(make([]byte, 1024))
	source := JITValueDesc{Loc: LocStackPair, Type: tagSlice, StackOff: -16, Rooted: true}
	header := jitKnownSliceHeader(ctx, &source)
	if header.Loc != LocStackTriple || !header.Rooted {
		t.Fatalf("slice header did not get a stable stack home: %#v", header)
	}
	for word := int32(0); word < 3; word++ {
		root := jitStackRoot{base: jitStackRootFrameSP, offset: header.StackOff + word*8}
		_, pointer := ctx.StackRoots[root]
		if pointer != (word == 0) {
			t.Fatalf("slice header word %d pointer=%v; only the backing array is a GC root", word, pointer)
		}
	}
	if ctx.FreeRegs != ctx.AllRegs {
		t.Fatal("stack header extraction consumed the caller's remaining registers")
	}
}

func TestJITUnboxedScalarStabilizationDoesNotCreateGCStackRoot(t *testing.T) {
	t.Run("control flow", func(t *testing.T) {
		code := make([]byte, 512)
		ctx := newJITStackMapTestContext(code)
		value := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: RegRAX}
		ctx.BindReg(RegRAX, &value)

		ctx.StabilizeDescForControlFlow(&value)
		if _, exists := ctx.StackRoots[jitStackRoot{base: jitStackRootFrameSP, offset: 0}]; exists {
			t.Fatal("unboxed control-flow scalar is marked as a GC pointer")
		}
	})

	t.Run("nested call", func(t *testing.T) {
		code := make([]byte, 512)
		ctx := newJITStackMapTestContext(code)
		value := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: RegRAX}
		ctx.BindReg(RegRAX, &value)

		ctx.StabilizeDescAcrossNestedCall(&value)
		if _, exists := ctx.StackRoots[jitStackRoot{base: jitStackRootFrameBP, offset: -8}]; exists {
			t.Fatal("unboxed nested-call scalar is marked as a GC pointer")
		}
	})

	t.Run("parser environment", func(t *testing.T) {
		code := make([]byte, 512)
		ctx := newJITStackMapTestContext(code)
		value := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: RegRAX}
		ctx.BindReg(RegRAX, &value)

		stable := ctx.StabilizeJITEnv(&JITEnv{Vars: map[Symbol]JITValueDesc{"value": value}}, false)
		if _, exists := ctx.StackRoots[jitStackRoot{base: jitStackRootFrameBP, offset: -8}]; exists {
			t.Fatal("unboxed parser-environment scalar is marked as a GC pointer")
		}
		if got := stable.Vars["value"]; got.Loc != LocStack || got.RelocatablePointer {
			t.Fatalf("unexpected stabilized scalar: %+v", got)
		}
	})
}

func TestJITRelocatableScalarStabilizationCreatesGCStackRoot(t *testing.T) {
	code := make([]byte, 512)
	ctx := newJITStackMapTestContext(code)
	value := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: RegRAX, RelocatablePointer: true}
	ctx.BindReg(RegRAX, &value)

	ctx.StabilizeDescAcrossNestedCall(&value)
	if _, exists := ctx.StackRoots[jitStackRoot{base: jitStackRootFrameBP, offset: -8}]; !exists {
		t.Fatal("relocatable nested-call scalar is missing from the GC stack map")
	}
}

func TestJITSavedFramePointerIsNotAHeapRoot(t *testing.T) {
	ctx := &JITContext{Safepoints: []jitSafepoint{{}}}
	maps := ctx.finalizeStackMaps(16, 0)
	if len(maps) != 1 || maps[0].frameWords != 3 {
		t.Fatalf("unexpected stack map: %+v", maps)
	}
	if maps[0].pointerMap[0]&(1<<2) != 0 {
		t.Fatal("saved caller frame pointer is encoded as a GC heap root")
	}
}

func TestJITEntryGrowsStackInPrologue(t *testing.T) {
	value := NewSymbol("value")
	gcEcho := NewFunc(func(args ...Scmer) Scmer {
		runtime.GC()
		return args[0]
	})
	proc := NewProcStruct(Proc{
		Params:  NewSlice([]Scmer{value}),
		Body:    NewSlice([]Scmer{gcEcho, NewNthLocalVar(0)}),
		En:      &Globalenv,
		NumVars: 4096,
	})
	compiled := jitCompile(proc)
	if !compiled.IsProc() || compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		t.Fatal("large-frame procedure was not JIT compiled")
	}

	done := make(chan Scmer, 1)
	want := "pointer-bearing input survives stack relocation"
	go func() {
		done <- Apply(compiled, NewString(want))
	}()
	select {
	case result := <-done:
		if !result.IsString() || result.String() != want {
			t.Fatalf("unexpected result after JIT stack growth: %s", result.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("JIT stack growth did not resume the generated procedure")
	}
}

func TestJITDeferredCompilationReturnsPrivateProc(t *testing.T) {
	template := Eval(Read(t.Name(), `(lambda (value) value)`), &Globalenv)
	if !template.IsProc() || template.Proc() == nil {
		t.Fatal("lambda did not produce a Proc template")
	}
	compiled := jitCompileModeDeferred(true, template)
	if !compiled.IsProc() || compiled.Proc() == nil {
		t.Fatal("deferred compilation did not return a Proc")
	}
	if compiled.Proc() == template.Proc() {
		t.Fatal("deferred compilation returned the shared Proc template")
	}
	if template.Proc().JITCode == 0 || template.Proc().Compiled == nil {
		t.Fatal("shared Proc template was not installed after immediate stack-map publication")
	}
	if compiled.Proc().JITCode == 0 || compiled.Proc().Compiled == nil {
		t.Fatal("deferred compilation did not return its private compiled Proc")
	}
	if JITProcForFunction(compiled.Proc().Compiled.Native) != compiled.Proc() {
		t.Fatal("shared installation rewrote the private entry's funcval")
	}
}

func TestJITConcurrentSharedProcPublication(t *testing.T) {
	dispatch := CompileJIT(Eval(Read(t.Name(), `(lambda (callback value) (callback value))`), &Globalenv), true)
	for round := 0; round < 20; round++ {
		template := Eval(Read(t.Name(), `(lambda (value) (list value value))`), &Globalenv)
		start := make(chan struct{})
		var workers sync.WaitGroup
		for worker := 0; worker < 16; worker++ {
			workers.Add(1)
			go func() {
				defer workers.Done()
				<-start
				for attempt := 0; attempt < 20; attempt++ {
					compiled := CompileJIT(template, true)
					got := Apply(compiled, NewString("shared publication"))
					if attempt%2 != 0 {
						got = Apply(dispatch, compiled, NewString("shared publication"))
					}
					if !got.IsSlice() || len(got.Slice()) != 2 || got.Slice()[0].String() != "shared publication" || got.Slice()[1].String() != "shared publication" {
						t.Error("concurrent compilation returned corrupt values")
						return
					}
				}
			}()
		}
		close(start)
		workers.Wait()
	}
}

// Append metadata while other goroutines grow and scan stacks through older
// native entries. Both the new reservation and the old map prefix must remain
// usable when runtime/jit's backing table grows or allocation changes arenas.
func TestJITConcurrentStackMapAppendAndGC(t *testing.T) {
	var workers sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			var retained []Scmer
			for round := 0; round < 32; round++ {
				template := Eval(Read(t.Name(), `(lambda (value) value)`), &Globalenv)
				template.Proc().Body = NewSlice([]Scmer{NewFunc(func(args ...Scmer) Scmer {
					return jitPublicationTestGrowStack(32, args[0])
				}), NewNthLocalVar(0)})
				retained = append(retained, CompileJIT(template, true))
				for _, compiled := range []Scmer{retained[0], retained[len(retained)-1]} {
					want := strings.Repeat(fmt.Sprintf("map-%d/", round), 32)
					got := Apply(compiled, NewString(want))
					if !got.IsString() || got.String() != want {
						t.Error("stack-map append or stack relocation lost a live value")
						return
					}
				}
			}
			runtime.KeepAlive(retained)
		}()
	}
	workers.Wait()
}

// Reusing a private, deferred Proc must still honor the public compilation
// boundary: native code is executable only after its stack maps are visible.
func TestJITCompileExistingProcWaitsForPublication(t *testing.T) {
	_, arena, blocker := globalJITPool.Alloc(16 * 1024)
	defer globalJITPool.Free(arena)
	defer arena.complete(blocker, nil, nil)
	template := Eval(Read(t.Name(), `(lambda (value) value)`), &Globalenv)
	compiled := jitCompileModeDeferred(true, template)
	entry := compiled.Proc().Compiled
	if entry == nil || entry.Arena != arena || entry.reservation.ready {
		t.Fatal("fixture did not defer the compiled Proc behind the blocker")
	}
	done := make(chan Scmer, 1)
	go func() { done <- CompileJIT(compiled, true) }()
	select {
	case <-done:
		t.Fatal("public compilation returned native code before its stack maps were published")
	case <-time.After(20 * time.Millisecond):
	}
	arena.complete(blocker, nil, nil)
	select {
	case value := <-done:
		if !entry.reservation.ready || !Equal(Apply(value, NewInt(42)), NewInt(42)) {
			t.Fatal("published procedure did not execute correctly")
		}
	case <-time.After(time.Second):
		t.Fatal("compilation did not return after stack-map publication")
	}
}

func TestJITArenaDefersCallbackUntilStackMapPublication(t *testing.T) {
	first := &jitCodeReservation{}
	second := &jitCodeReservation{}
	arena := &jitArena{reservations: []*jitCodeReservation{first, second}}
	published := false
	arena.completeDeferred(second, nil, nil, func() { published = true })
	if published {
		t.Fatal("deferred entry became visible before the preceding stack maps")
	}
	arena.complete(first, nil, nil)
	if !published || !second.published {
		t.Fatal("deferred entry was not installed with its stack maps")
	}
}

func TestJITArenaParentWaitsForInterleavedDeferredStackMaps(t *testing.T) {
	parent := &jitCodeReservation{}
	otherCompiler := &jitCodeReservation{}
	child := &jitCodeReservation{}
	arena := &jitArena{reservations: []*jitCodeReservation{parent, otherCompiler, child}}
	arena.completeDeferred(child, nil, nil, nil)

	parentComplete := make(chan struct{})
	go func() {
		arena.complete(parent, nil, []*JITEntryPoint{{reservation: child}})
		close(parentComplete)
	}()
	select {
	case <-parentComplete:
		t.Fatal("parent became reachable before its interleaved child stack maps")
	case <-time.After(20 * time.Millisecond):
	}

	arena.complete(otherCompiler, nil, nil)
	select {
	case <-parentComplete:
	case <-time.After(time.Second):
		t.Fatal("parent did not become reachable after its child stack maps")
	}
	if !parent.published || !otherCompiler.published || !child.published {
		t.Fatal("interleaved stack maps were not published as one reachable prefix")
	}
}

func TestJITParentWaitsForDeferredChildAcrossArenas(t *testing.T) {
	parentPtr, parentArena, parentReservation := globalJITPool.Alloc(jitArenaSize)
	defer globalJITPool.Free(parentArena)
	_, childArena, blocker := globalJITPool.Alloc(16 * 1024)
	defer globalJITPool.Free(childArena)
	defer childArena.complete(blocker, nil, nil)
	if parentArena == childArena {
		t.Fatal("fixture did not cross an arena boundary")
	}

	child := Eval(Read(t.Name(), `(lambda (value) value)`), &Globalenv)
	child.Proc().Body = NewSlice([]Scmer{NewFunc(func(args ...Scmer) Scmer {
		return jitPublicationTestGrowStack(64, args[0])
	}), NewNthLocalVar(0)})
	parent := Eval(Read(t.Name(), `(lambda (value) value)`), &Globalenv).Proc()
	parent.Body = NewSlice([]Scmer{child, NewNthLocalVar(0)})
	buf := &execBuf{ptr: parentPtr, n: jitArenaSize, arena: parentArena, reservation: parentReservation}
	codeLen, roots, dependencies, overflow, hiddenArgs, needsStableArgs, coverage := jitCompileProcToExec(parent, buf, true)
	if codeLen == 0 || overflow {
		parentArena.completeDeferred(parentReservation, buf.stackMaps, dependencies, nil)
		t.Fatal("parent did not compile")
	}
	if len(dependencies) == 0 {
		parentArena.completeDeferred(parentReservation, buf.stackMaps, dependencies, nil)
		t.Fatal("parent did not retain its deferred child dependency")
	}
	childEntry := dependencies[0]
	if childEntry.Arena != childArena {
		parentArena.completeDeferred(parentReservation, buf.stackMaps, dependencies, nil)
		t.Fatal("child was not compiled behind the blocker in the second arena")
	}
	defer runtime.KeepAlive(roots)
	defer runtime.KeepAlive(dependencies)

	parentComplete := make(chan struct{})
	go func() {
		parentArena.complete(parentReservation, buf.stackMaps, dependencies)
		close(parentComplete)
	}()
	select {
	case <-parentComplete:
		t.Fatal("parent became reachable before its child stack maps in another arena")
	case <-time.After(20 * time.Millisecond):
	}
	childArena.complete(blocker, nil, nil)
	select {
	case <-parentComplete:
	case <-time.After(time.Second):
		t.Fatal("parent did not become reachable after its child stack maps")
	}
	entry := &JITEntryPoint{
		CodePtr: parentPtr, CodeLen: codeLen, Arena: parentArena,
		StackFrameSize: buf.stackFrameSize, ConstRoots: roots, Dependencies: dependencies,
		HiddenArgs: hiddenArgs, NeedsStableArgs: needsStableArgs, Coverage: coverage,
		Proc: *parent, reservation: parentReservation,
	}
	attachProcJIT(parent, entry)
	defer jitNativeCodes.Delete(uintptr(parentPtr))
	result := make(chan Scmer, 1)
	// A fresh goroutine starts with a small stack. The Go callback must grow it
	// across both native frames and then scan them during GC.
	go func() { result <- entry.Call(NewString("relocated child")) }()
	select {
	case got := <-result:
		if !got.IsString() || got.String() != "relocated child" {
			t.Fatalf("native parent/child call after stack growth returned %s", String(got))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("native parent/child call did not finish after stack growth")
	}
}

//go:noinline
func jitPublicationTestGrowStack(depth int, value Scmer) Scmer {
	var padding [512]uintptr
	padding[0] = uintptr(depth)
	defer runtime.KeepAlive(&padding)
	if depth == 0 {
		runtime.GC()
		return value
	}
	return jitPublicationTestGrowStack(depth-1, value)
}

func TestJITSharedPublicationWaitsForTransitiveDependencies(t *testing.T) {
	grandchild, child, parent := &jitCodeReservation{}, &jitCodeReservation{}, &jitCodeReservation{}
	grandchildArena := &jitArena{reservations: []*jitCodeReservation{grandchild}}
	childArena := &jitArena{reservations: []*jitCodeReservation{child}}
	parentArena := &jitArena{reservations: []*jitCodeReservation{parent}}
	var installed []string
	childArena.completeDeferred(child, nil, []*JITEntryPoint{{reservation: grandchild}}, func() {
		installed = append(installed, "child")
	})
	parentArena.completeDeferred(parent, nil, []*JITEntryPoint{{reservation: child}}, func() {
		installed = append(installed, "parent")
	})
	if !child.published || !parent.published || child.ready || parent.ready || len(installed) != 0 {
		t.Fatal("shared callbacks ran before their transitive dependency was ready")
	}
	grandchildArena.complete(grandchild, nil, nil)
	if len(installed) != 2 || installed[0] != "child" || installed[1] != "parent" || !parent.ready {
		t.Fatalf("shared installation order = %v, want child then parent", installed)
	}
	if grandchild.dependents != nil || child.dependents != nil || child.onPublish != nil || parent.onPublish != nil {
		t.Fatal("finished publication retained dependency callbacks or reverse edges")
	}
}

func TestJITMetadataPublicationDoesNotWaitForCalleeReadiness(t *testing.T) {
	parent, blocker, child := &jitCodeReservation{}, &jitCodeReservation{}, &jitCodeReservation{}
	parentArena := &jitArena{reservations: []*jitCodeReservation{parent}}
	childArena := &jitArena{reservations: []*jitCodeReservation{blocker, child}}
	childArena.completeDeferred(child, nil, nil, nil)
	parentArena.completeDeferred(parent, nil, []*JITEntryPoint{{reservation: child}}, nil)
	if !parent.published || parent.ready || child.published {
		t.Fatal("fixture did not leave parent waiting on blocked child metadata")
	}
	// The blocker's maps must unblock the child even though the blocker itself
	// calls the parent. Readiness edges are child -> parent -> blocker.
	childArena.completeDeferred(blocker, nil, []*JITEntryPoint{{reservation: parent}}, nil)
	if !child.ready || !parent.ready || !blocker.ready {
		t.Fatal("metadata ordering deadlocked an acyclic callee dependency chain")
	}
}

func TestJITPublicationAllowsEntriesSharingReservation(t *testing.T) {
	reservation := &jitCodeReservation{}
	arena := &jitArena{reservations: []*jitCodeReservation{reservation}}
	first := &JITEntryPoint{reservation: reservation}
	second := &JITEntryPoint{reservation: reservation}
	arena.completeDeferred(reservation, nil, []*JITEntryPoint{first, second}, nil)
	if !reservation.ready {
		t.Fatal("entries sharing one reservation waited on their own metadata")
	}
}

func TestJITParentIgnoresUnrelatedSuccessor(t *testing.T) {
	parent, unrelated := &jitCodeReservation{}, &jitCodeReservation{}
	arena := &jitArena{reservations: []*jitCodeReservation{parent, unrelated}}
	defer arena.complete(unrelated, nil, nil)
	complete := make(chan struct{})
	go func() {
		arena.complete(parent, nil, nil)
		close(complete)
	}()
	select {
	case <-complete:
	case <-time.After(time.Second):
		t.Fatal("unrelated later compilation blocked parent publication")
	}
	if unrelated.published {
		t.Fatal("unrelated unfinished metadata was published")
	}
}

func TestJITTrackImmCollectsReadinessDependenciesAfterRootDeduplication(t *testing.T) {
	ctx := &JITContext{}
	value := NewProcStruct(Proc{})
	ctx.TrackImm(value)
	if len(ctx.ConstRoots) != 1 || len(ctx.EntryRoots) != 0 {
		t.Fatal("uncompiled Proc was not retained without a native dependency")
	}
	procEntry := &JITEntryPoint{reservation: &jitCodeReservation{}}
	value.Proc().Compiled = procEntry
	ctx.TrackImm(value)
	ctx.TrackImm(value)
	if len(ctx.ConstRoots) != 1 || len(ctx.EntryRoots) != 1 || ctx.EntryRoots[0] != procEntry {
		t.Fatal("GC-root deduplication lost the newly compiled Proc dependency")
	}
	helperEntry := &JITEntryPoint{reservation: &jitCodeReservation{}}
	helper := NewAny(helperEntry)
	ctx.TrackImm(helper)
	ctx.TrackImm(helper)
	if len(ctx.EntryRoots) != 2 || ctx.EntryRoots[1] != helperEntry {
		t.Fatal("Any-wrapped parser or regex helper was not a readiness dependency")
	}
}

func TestJITRegisterHomesFollowArchitectureBank(t *testing.T) {
	all := uint64(1<<uint(RegR13) | 1<<uint(RegR15) | 1<<uint(RegRCX))
	ctx := &JITContext{
		AllRegs:  all,
		FreeRegs: all,
		RegisterBank: JITRegisterBank{
			Registers:        [16]Reg{RegR13, RegR15, RegRCX},
			Count:            3,
			TemporaryReserve: 1,
		},
	}
	homes := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 1, Cost: 3}, {Color: 1, Width: 1, Cost: 2}, {Color: 2, Width: 1, Cost: 1}}, Count: 3})
	if homes.Available != 3 || homes.Registers[0] != RegR13 || homes.Registers[1] != RegR15 {
		t.Fatalf("homes = %#v, want the first two backend registers", homes)
	}
	if ctx.FreeRegs&(1<<uint(RegRCX)) == 0 {
		t.Fatal("allocator consumed the backend's temporary reserve")
	}
	ctx.ReleaseRegisterHomes(homes)
	if ctx.FreeRegs != all || ctx.ProtectedRegs != 0 {
		t.Fatalf("released state free=%#x protected=%#x, want free=%#x protected=0", ctx.FreeRegs, ctx.ProtectedRegs, all)
	}
}

func TestJITRegisterHomesUseSeparateIntegerAndFloatBanks(t *testing.T) {
	gpr := uint64(1 << uint(RegR13))
	fp := uint64(1 << uint(RegX2))
	ctx := &JITContext{
		AllRegs: gpr, FreeRegs: gpr,
		AllFPRegs: fp, FreeFPRegs: fp,
		RegisterBank:   JITRegisterBank{Registers: [16]Reg{RegR13}, Count: 1},
		FPRegisterBank: JITRegisterBank{Registers: [16]Reg{RegX2}, Count: 1},
	}
	plan := JITRegisterPlan{Slots: [16]JITRegisterSlot{
		{Color: 0, Width: 1, Cost: 2},
		{Color: 1, Width: 1, Class: JITRegisterClassFP, Cost: 2},
	}, Count: 2}
	homes := ctx.AllocRegisterHomes(plan)
	if homes.Available != 3 || homes.Registers[0] != RegR13 || homes.Registers[1] != RegX2 {
		t.Fatalf("mixed register homes = %#v, want R13 and XMM2", homes)
	}
	ctx.ReleaseRegisterHomes(homes)
	if ctx.FreeRegs != gpr || ctx.FreeFPRegs != fp || ctx.ProtectedRegs != 0 {
		t.Fatalf("released mixed homes: gpr=%#x fp=%#x protected=%#x", ctx.FreeRegs, ctx.FreeFPRegs, ctx.ProtectedRegs)
	}
}

func TestJITRegisterHomesTradeOuterForMoreValuableInnerPlan(t *testing.T) {
	code := make([]byte, 1024)
	start := unsafe.Pointer(&code[0])
	all := uint64(1<<uint(RegR13) | 1<<uint(RegR15) | 1<<uint(RegRCX))
	ctx := &JITContext{
		Start: start, Ptr: start, End: unsafe.Add(start, len(code)-1),
		AllRegs: all, FreeRegs: all, FrameReg: RegRBP, StackReg: RegRSP,
		RegisterBank: JITRegisterBank{Registers: [16]Reg{RegR13, RegR15, RegRCX}, Count: 3, TemporaryReserve: 1},
	}
	outer := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 2, Cost: 2}}, Count: 1})
	value := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: outer.Registers[0], Reg2: outer.Registers[1]}
	ctx.BindReg(value.Reg, &value)
	ctx.BindReg(value.Reg2, &value)

	inner := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 2, Cost: 10}}, Count: 1})
	if inner.Available != 3 || inner.Evictions != 1 {
		t.Fatalf("inner homes = %#v, want pair replacing one outer bundle", inner)
	}
	ctx.SyncDesc(&value)
	if value.Loc != LocStackPair {
		t.Fatalf("evicted outer value location = %d, want stack pair", value.Loc)
	}
	ctx.ReleaseRegisterHomes(inner)
	ctx.SyncDesc(&value)
	if value.Loc != LocRegPair || value.Reg != outer.Registers[0] || value.Reg2 != outer.Registers[1] {
		t.Fatalf("restored outer value = %#v, want original register pair", value)
	}
	ctx.ReleaseRegisterHomes(outer)
}

func TestJITRegisterHomesRetainMoreValuableOuterPlan(t *testing.T) {
	code := make([]byte, 1024)
	start := unsafe.Pointer(&code[0])
	all := uint64(1<<uint(RegR13) | 1<<uint(RegR15) | 1<<uint(RegRCX))
	ctx := &JITContext{
		Start: start, Ptr: start, End: unsafe.Add(start, len(code)-1),
		AllRegs: all, FreeRegs: all, FrameReg: RegRBP, StackReg: RegRSP,
		RegisterBank: JITRegisterBank{Registers: [16]Reg{RegR13, RegR15, RegRCX}, Count: 3, TemporaryReserve: 1},
	}
	outer := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 2, Cost: 10}}, Count: 1})
	value := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: outer.Registers[0], Reg2: outer.Registers[1]}
	ctx.BindReg(value.Reg, &value)
	ctx.BindReg(value.Reg2, &value)

	inner := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 2, Cost: 2}}, Count: 1})
	if inner.Available != 0 || inner.Evictions != 0 {
		t.Fatalf("inner homes = %#v, want valuable outer bundle retained", inner)
	}
	if value.Loc != LocRegPair {
		t.Fatalf("outer value location = %d, want register pair", value.Loc)
	}
	ctx.ReleaseRegisterHomes(inner)
	ctx.ReleaseRegisterHomes(outer)
}

func TestJITRegisterBoundaryCanReleaseAndRestorePlannedHomes(t *testing.T) {
	code := make([]byte, 1024)
	start := unsafe.Pointer(&code[0])
	all := uint64(1<<uint(RegR13) | 1<<uint(RegR15) | 1<<uint(RegRCX))
	ctx := &JITContext{
		Start: start, Ptr: start, End: unsafe.Add(start, len(code)-1),
		AllRegs: all, FreeRegs: all, FrameReg: RegRBP, StackReg: RegRSP,
		RegisterBank: JITRegisterBank{Registers: [16]Reg{RegR13, RegR15, RegRCX}, Count: 3},
	}
	homes := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 1, Cost: 10}}, Count: 1})
	home := homes.Registers[0]
	value := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: home}
	ctx.BindReg(home, &value)

	boundary := ctx.PreserveRegisters(JITRegisterBoundaryOptions{ReleaseHomes: true})
	if ctx.FreeRegs&all != all {
		t.Fatalf("nested boundary free registers = %#x, want %#x", ctx.FreeRegs&all, all)
	}
	if ctx.ProtectedRegs&all != 0 || ctx.RegisterHomeID[home] != 0 {
		t.Fatalf("released home remains protected/registered: protected=%#x homeID=%d", ctx.ProtectedRegs, ctx.RegisterHomeID[home])
	}

	// Model arbitrary nested emission taking the old physical home. Restore must
	// discard this ownership and recover the outer value in the exact register.
	nested := ctx.AllocReg()
	nestedValue := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: nested}
	ctx.BindReg(nested, &nestedValue)
	boundary.Restore(ctx)
	ctx.SyncDesc(&value)
	if value.Loc != LocReg || value.Reg != home {
		t.Fatalf("restored outer value = %#v, want register %d", value, home)
	}
	if ctx.RegisterHomeID[home] == 0 || ctx.ProtectedRegs&(1<<uint(home)) == 0 {
		t.Fatalf("restored home metadata missing: protected=%#x homeID=%d", ctx.ProtectedRegs, ctx.RegisterHomeID[home])
	}
	ctx.ReleaseRegisterHomes(homes)
}

func TestJITRegisterHomesSkipFoldedTagLane(t *testing.T) {
	all := uint64(1<<uint(RegR13) | 1<<uint(RegR15))
	ctx := &JITContext{
		AllRegs: all, FreeRegs: all,
		RegisterBank: JITRegisterBank{Registers: [16]Reg{RegR13, RegR15}, Count: 2, TemporaryReserve: 1},
	}
	homes := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 2, Lanes: 2, Cost: 10}}, Count: 1})
	if homes.Available != 2 || homes.Registers[1] != RegR13 {
		t.Fatalf("payload-only homes = %#v, want logical lane 1 in first physical register", homes)
	}
	if ctx.FreeRegs&(1<<uint(RegR15)) == 0 {
		t.Fatal("folded tag lane consumed the temporary register reserve")
	}
	ctx.ReleaseRegisterHomes(homes)
}

// Benchmark the metadata retained while compiling a wide SQL values expression.
func BenchmarkJITSafepointSnapshots(b *testing.B) {
	code := make([]byte, 16)
	roots := make(map[jitStackRoot]struct{}, 2048)
	for i := int32(0); i < 2048; i++ {
		roots[jitStackRoot{base: jitStackRootFrameSP, offset: i * 16}] = struct{}{}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := JITContext{Start: unsafe.Pointer(&code[0]), Ptr: unsafe.Pointer(&code[0]), StackRoots: roots}
		for j := 0; j < 128; j++ {
			ctx.recordSafepoint(nil, 0)
		}
		runtime.KeepAlive(ctx.Safepoints)
	}
}

func TestJITSafepointSnapshotsPreserveExactRoots(t *testing.T) {
	code := make([]byte, 16)
	ctx := JITContext{Start: unsafe.Pointer(&code[0]), Ptr: unsafe.Pointer(&code[0]), DynamicSP: 32,
		StackRoots: map[jitStackRoot]struct{}{
			{base: jitStackRootFrameSP, offset: -16}: {},
			{base: jitStackRootFrameSP, offset: 8}:   {},
			{base: jitStackRootFrameBP, offset: -8}:  {},
		},
	}
	ctx.recordSafepoint([]int32{0}, 8)
	delete(ctx.StackRoots, jitStackRoot{base: jitStackRootFrameSP, offset: 8})
	ctx.StackRoots[jitStackRoot{base: jitStackRootFrameBP, offset: -24}] = struct{}{}
	ctx.recordSafepoint(nil, 0)
	ctx.DynamicSP = 0
	maps := ctx.finalizeStackMaps(64, 0)
	for i, want := range [][]byte{{0x26, 0x08}, {0x04, 0x0a}} {
		if !bytes.Equal(maps[i].pointerMap, want) {
			b := maps[i].pointerMap
			t.Fatalf("snapshot %d = %08b, want %08b", i, b, want)
		}
	}
}

func TestJITStackReleasePreservesOtherRoots(t *testing.T) {
	ctx := JITContext{DynamicSP: 32, BPOffset: 16, StackRoots: make(map[jitStackRoot]struct{})}
	for _, root := range []jitStackRoot{{base: jitStackRootFrameSP, offset: -32}, {base: jitStackRootFrameSP, offset: -25}, {base: jitStackRootFrameSP, offset: -16}, {base: jitStackRootFrameSP, offset: 0}, {base: jitStackRootFrameBP, offset: -32}} {
		ctx.StackRoots[root] = struct{}{}
	}
	// FreeStack removes [-32,-16), including unaligned offsets, without
	// dropping the adjacent dynamic frame or BP-based roots.
	ctx.FreeStack(16)
	if len(ctx.StackRoots) != 3 {
		t.Fatalf("remaining roots: %v", ctx.StackRoots)
	}
	for _, root := range []jitStackRoot{{base: jitStackRootFrameSP, offset: -16}, {base: jitStackRootFrameSP, offset: 0}, {base: jitStackRootFrameBP, offset: -32}} {
		if _, ok := ctx.StackRoots[root]; !ok {
			t.Fatalf("lost live root %v", root)
		}
	}
}

func BenchmarkJITSmallStackRelease(b *testing.B) {
	ctx := JITContext{StackRoots: make(map[jitStackRoot]struct{})}
	for i := int32(0); i < 10000; i++ {
		ctx.StackRoots[jitStackRoot{base: jitStackRootFrameBP, offset: -8 * (i + 1)}] = struct{}{}
	}
	root := jitStackRoot{base: jitStackRootFrameSP, offset: -16}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.DynamicSP, ctx.BPOffset = 16, 16
		ctx.StackRoots[root] = struct{}{}
		ctx.FreeStack(16)
	}
}

// A deterministic cancellation boundary inside emission avoids a timing-based
// regression which might accidentally cancel only before compilation starts.
type jitCancelDuringCompile struct {
	context.Context
	cancel    context.CancelFunc
	remaining int
}

func (c *jitCancelDuringCompile) Err() error {
	c.remaining--
	if c.remaining == 0 {
		c.cancel()
	}
	return c.Context.Err()
}
func TestJITCompileCancellationReleasesProcedure(t *testing.T) {
	expression := Optimize(Read(t.Name(), `(lambda (x) (list (+ x 1) (+ x 2) (+ x 3) (+ x 4) (+ x 5) (+ x 6)))`), &Globalenv, nil)
	proc := Eval(expression, &Globalenv)
	base, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := &jitCancelDuringCompile{Context: base, cancel: cancel, remaining: 8}
	ss := &SessionState{}
	seq := ss.BeginQuery("Query", "compile cancellation")
	ss.SetQueryContext(seq, ctx)
	defer ss.EndQuery(seq, "Sleep", "")
	var caught any
	func() {
		defer func() { caught = recover() }()
		jitCompile(proc, NewAny(&mutexTestTransaction{ss: ss, seq: seq}))
	}()
	if caught != context.Canceled {
		t.Fatalf("compile error = %v, want cancellation", caught)
	}
	if proc.Proc().jitCompiling != 0 || proc.Proc().Compiled != nil {
		t.Fatal("cancelled compiler retained or published procedure")
	}
	compiled := jitCompile(proc)
	if compiled.Proc().Compiled == nil {
		t.Fatal("procedure could not be compiled after cancellation")
	}
	result := Apply(compiled, NewInt(10)).Slice()
	if len(result) != 6 || result[5].Int() != 16 {
		t.Fatalf("retry result %v", result)
	}
}

// Scoped locals must not replace the self capture passed into a nested
// callback. Direct recursion and callback recursion share the same closure.
func TestJITRecursiveCaptureAcrossLexicalScopes(t *testing.T) {
	source := `(lambda (enabled)
		(begin
			(define visit (lambda (n)
				(if (equal? n 0) (if enabled 7 11)
					(match n value
						(begin
							(define key (concat "local-" value))
							(car (map (list (- value 1)) (lambda (child) (visit child)))))))))
			(visit 3)))`
	procedure := jitCompile(Eval(Optimize(Read(t.Name(), source), &Globalenv, nil), &Globalenv))
	if procedure.Proc() == nil || procedure.Proc().JITCode == 0 {
		t.Fatal("recursive closure did not compile natively")
	}
	for _, enabled := range []bool{true, false, true} {
		want := int64(11)
		if enabled {
			want = 7
		}
		if got := Apply(procedure, NewBool(enabled)).Int(); got != want {
			t.Fatalf("recursive capture returned %d, want %d", got, want)
		}
	}
}

// Deep recursive source rewriting combines captures, match scopes and a wide
// tuple result. Native callback results must agree with interpreter execution.
func TestJITRecursiveCaptureDeepSourceMapping(t *testing.T) {
	source := `(lambda (query policy) (begin
	(define visit (lambda (node stack)
		(match node
			((symbol query-block) query_schema sources fields condition group having order limit offset hidden stages facts)
			(list (quote query-block)
				query_schema
				(map sources (lambda (source)
					(match source
						'(alias source_schema relation outer join_condition)
						(begin
							(define expanded_relation
								(if (string? relation)
									(begin
										(if policy (policy source_schema relation false) true)
										(if false
											relation
											(begin
												(define view (list "tag" "source" 7))
												(if (nil? view)
													relation
													(begin
														(define key (concat source_schema "." relation))
														(if (contains? stack key)
															(error (concat "circular view reference: " key))
															true)
														(define stored_ir (nth view 2))
														(if (nil? stored_ir)
															(error (concat "view " key " has no parsed IR"))
															(visit stored_ir (cons key stack))))))))
									(visit relation stack)))
							(list alias source_schema expanded_relation outer (visit join_condition stack)))
						_ (error "invalid query source while expanding views"))))
				(visit fields stack)
				(visit condition stack)
				(visit group stack)
				(visit having stack)
				(visit order stack)
				limit offset
				(visit hidden stack)
				(visit stages stack)
				(visit facts stack))
			((symbol union-block) mode branches order limit offset facts)
			(list (quote union-block) mode
				(map branches (lambda (branch) (visit branch stack)))
				(visit order stack) limit offset (visit facts stack))
			(cons head tail)
			(cons (visit head stack)
				(map tail (lambda (item) (visit item stack))))
			_ node)))
	(if true
		(visit query '())
		query)))`
	procedure := Eval(Optimize(Read(t.Name(), source), &Globalenv, nil), &Globalenv)
	empty := NewSlice(nil)
	input := NewSlice([]Scmer{NewSymbol("query-block"), NewString("s"), NewSlice([]Scmer{NewSlice([]Scmer{NewString("a"), NewString("s"), NewString("ref"), NewBool(false), NewBool(true)})}), empty, NewBool(true), empty, NewBool(true), empty, NewNil(), NewNil(), empty, empty, empty})
	want := Apply(procedure, input, NewBool(false))

	procedure = jitCompile(procedure)
	if procedure.Proc().JITCode == 0 {
		t.Fatal("source mapper did not compile natively")
	}
	got := Apply(procedure, input, NewBool(false))
	if !Equal(got, want) {
		t.Fatalf("native %s, interpreter %s", SerializeToString(got, &Globalenv), SerializeToString(want, &Globalenv))
	}
}

// The inner begin shares its invocation's numbered locals, not the enclosing
// procedure's hidden self capture. A local text value must not shadow that call.
func TestJITRecursiveSelfBindingAcrossSharedNumberedScopes(t *testing.T) {
	body := NewSlice([]Scmer{
		NewSymbol("lambda"), NewSlice([]Scmer{NewSymbol("child")}),
		NewSlice([]Scmer{NewSymbol("begin"),
			NewSlice([]Scmer{NewSymbol("setN"), NewNthLocalVar(1), NewString("local text")}),
			NewSlice([]Scmer{NewSymbol("visit"), NewNthLocalVar(0)}),
		}), NewInt(2),
	})
	env := &Env{Outer: &Globalenv, VarsNumbered: []Scmer{NewNil(), NewFunc(func(args ...Scmer) Scmer { return NewInt(args[0].Int() + 1) })}}
	callback := Eval(jitBindLambdaSelfValues(body, "visit", 1), env)
	if got := Apply(callback, NewInt(7)).Int(); got != 8 {
		t.Fatalf("self capture returned %d, want 8", got)
	}
}

// A byte-walker's branches may skip a spill instruction. Outer weighted homes
// must therefore keep one placement across the entire regex region rather than
// become path-local spills whose pointer map later revives a discarded value.
func TestJITRegexBranchesPreserveOuterHomes(t *testing.T) {
	for _, capture := range []bool{false, true} {
		for _, free := range []int{5, 6, 7} {
			t.Run(fmt.Sprintf("capture=%t/free=%d", capture, free), func(t *testing.T) {
				code := make([]byte, 1<<18)
				ctx := newJITStackMapTestContext(code)
				ctx.AllRegs = jitDefaultFreeGPRegs() | 1<<uint(RegRAX) | 1<<uint(RegRBX)
				ctx.FreeRegs = ctx.AllRegs
				ctx.RegisterBank = jitNativeRegisterBank
				ctx.SliceBase = RegR12
				ctx.W = ctx
				homes := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 2, Cost: 10}}, Count: 1})
				if homes.Available == 0 {
					t.Fatal("fixture did not allocate a weighted home")
				}
				value := JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: homes.Registers[0], Reg2: homes.Registers[1]}
				ctx.BindReg(value.Reg, &value)
				ctx.BindReg(value.Reg2, &value)
				held := make([]JITValueDesc, 0, 16)
				for bits.OnesCount64(ctx.FreeRegs&ctx.AllRegs&^ctx.ProtectedRegs) > free {
					r := ctx.AllocReg()
					held = append(held, JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r, RelocatablePointer: true})
					ctx.BindReg(r, &held[len(held)-1])
				}
				// Use stable descriptors: append must not invalidate registered owners.
				for i := range held {
					ctx.BindReg(held[i].Reg, &held[i])
				}
				input := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: ctx.AllocStack(16), Rooted: true}
				pattern := regexp.MustCompile(`\b[a-zA-Z_][a-zA-Z0-9_]*\b`)
				if capture {
					failed := ctx.ReserveLabel()
					captures := jitEmitConstantRegexpCaptures(ctx, pattern, input, failed)
					for i, capture := range captures {
						root := jitStackRoot{base: jitStackRootFrameSP, offset: capture.StackOff - ctx.DynamicSP}
						if _, ok := ctx.StackRoots[root]; !ok {
							t.Fatalf("capture %d lost its GC root at the region boundary", i)
						}
					}
					ctx.MarkLabel(failed)
				} else {
					jitEmitConstantRegexpPredicate(ctx, pattern, input)
				}
				for i := range held {
					wantReg := held[i].Reg
					ctx.SyncDesc(&held[i])
					if held[i].Loc != LocReg || held[i].Reg != wantReg {
						t.Fatalf("branch-local regex spill changed outer temporary %d: %#v", i, held[i])
					}
				}
				ctx.SyncDesc(&value)
				if value.Loc != LocRegPair || value.Reg != homes.Registers[0] || value.Reg2 != homes.Registers[1] {
					t.Fatalf("branch-local regex spill changed outer home: %#v", value)
				}
			})
		}
	}
}

// A skipped operand must not revive an expired spill from an earlier region.
// Check the map at the next call rather than relying on a lucky GC schedule.
func TestJITShortCircuitReleasesOperandRoots(t *testing.T) {
	for _, expression := range []string{
		`(and gate (jit_test_operand_root))`,
		`(or gate (jit_test_operand_root))`,
		`(if (and gate (jit_test_operand_root)) true false)`,
		`(if (or gate (jit_test_operand_root)) true false)`,
	} {
		t.Run(expression, func(t *testing.T) {
			var operandRoot jitStackRoot
			var produced, inspected bool
			declarationsForTest := []*Declaration{
				{Name: "jit_test_operand_root", Fn: func(...Scmer) Scmer { return NewBool(true) },
					Type: &TypeDescriptor{Kind: "func", Return: &TypeDescriptor{Kind: "bool"},
						JITEmit: func(ctx *JITContext, _ []Scmer, _ []JITValueDesc, _ JITValueDesc) JITValueDesc {
							off := ctx.AllocSpill(16)
							ctx.EmitStoreImm32Mem(ctx.FrameReg, off, 0)
							ctx.setStackPointer(jitStackRootFrameBP, off, true)
							operandRoot = jitStackRoot{base: jitStackRootFrameBP, offset: off}
							produced = true
							return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true), NoHeapPointer: true}
						}}},
				{Name: "jit_test_after_operand", Fn: func(...Scmer) Scmer { return NewBool(true) },
					Type: &TypeDescriptor{Kind: "func", Return: &TypeDescriptor{Kind: "bool"},
						JITEmit: func(ctx *JITContext, _ []Scmer, _ []JITValueDesc, _ JITValueDesc) JITValueDesc {
							if !produced {
								t.Fatal("operand was not compiled")
							}
							if _, live := ctx.StackRoots[operandRoot]; live {
								t.Errorf("expired operand root %v survives the short-circuit join", operandRoot)
							}
							inspected = true
							return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true), NoHeapPointer: true}
						}}},
			}
			for _, declaration := range declarationsForTest {
				Declare(&Globalenv, declaration)
				defer func(d *Declaration) {
					delete(Globalenv.Vars, Symbol(d.Name))
					delete(declarations, d.Name)
					delete(declarationsByFunction, FunctionIdentity(d.Fn))
				}(declaration)
			}
			compiled := compileJITExpressionTestProc(t, `(lambda (gate) (begin `+expression+` (jit_test_after_operand)))`)
			if !inspected {
				t.Fatal("join was not inspected")
			}
			for _, gate := range []bool{false, true} {
				if got := Apply(compiled, NewBool(gate)); !got.IsBool() || !got.Bool() {
					t.Fatalf("gate %t: %s", gate, String(got))
				}
			}
		})
	}
}

// A match result may cross a later allocating call even if its final arm is
// guaranteed to match. Its returned Rooted proof must survive branch allocator
// restoration and reach the actual safepoint bitmap, not only FrameRoots.
func TestJITMatchStackResultRemainsRootedAtFollowingSafepoint(t *testing.T) {
	for _, fixture := range []struct {
		name              string
		arms              []Scmer
		preRootedTarget   bool
		spillTarget       bool
		staleScalarTarget bool
		dynamicSP         int32
		dynamicSelector   bool
	}{
		{name: "binding-final-arm", arms: []Scmer{NewSymbol("_"), NewNthLocalVar(0)}},
		{name: "known-literal-final-arm", arms: []Scmer{NewInt(1), NewNthLocalVar(0)}},
		{name: "fallback-control", arms: []Scmer{NewInt(0), NewNil(), NewNthLocalVar(0)}},
		{name: "existing-root-control", arms: []Scmer{NewSymbol("_"), NewNthLocalVar(0)}, preRootedTarget: true},
		{name: "spill-final-arm", arms: []Scmer{NewSymbol("_"), NewNthLocalVar(0)}, spillTarget: true},
		{name: "old-scalar-target", arms: []Scmer{NewSymbol("_"), NewNthLocalVar(0)}, staleScalarTarget: true},
		{name: "dynamic-sp-target", arms: []Scmer{NewSymbol("_"), NewNthLocalVar(0)}, dynamicSP: 32},
		{name: "dynamic-sp-spill", arms: []Scmer{NewSymbol("_"), NewNthLocalVar(0)}, dynamicSP: 32, spillTarget: true},
		{name: "pointer-arm-before-scalar-arm", arms: []Scmer{NewInt(0), NewNthLocalVar(0), NewSymbol("_"), NewInt(2)}, dynamicSelector: true},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			ctx := newJITStackMapTestContext(make([]byte, 4096))
			ctx.StackPhiTargets = true
			ctx.SliceBase = RegR12
			if fixture.dynamicSP != 0 {
				ctx.AllocStack(64)
				ctx.DynamicSP = fixture.dynamicSP
			}
			ctx.Env = &JITEnv{Numbered: []JITValueDesc{{Loc: LocInputPair, Type: JITTypeUnknown, StackOff: 0}}}
			selector := NewInt(1)
			if fixture.dynamicSelector {
				ctx.AllRegs, ctx.FreeRegs = jitDefaultFreeGPRegs(), jitDefaultFreeGPRegs()
				ctx.Env.Numbered = append(ctx.Env.Numbered, JITValueDesc{Loc: LocInputPair, Type: JITTypeUnknown, StackOff: 1})
				selector = NewNthLocalVar(1)
			}
			target := JITValueDesc{Loc: LocAny}
			if fixture.spillTarget {
				target = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: ctx.AllocSpill(16), Rooted: true}
			}
			if fixture.preRootedTarget {
				target = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: ctx.AllocStack(16), Rooted: true}
				ctx.setStackPointer(jitStackRootFrameSP, target.StackOff, true)
			}
			if fixture.staleScalarTarget {
				target = JITValueDesc{Loc: LocStackPair, Type: tagInt, StackOff: ctx.AllocStack(16), NoHeapPointer: true, Rooted: true, SliceSizeKnown: true, KnownSliceLen: 7, KnownSliceCap: 7}
			}
			expression := append([]Scmer{NewSymbol("match"), selector}, fixture.arms...)
			out := jitCompileMatch(ctx, expression, ctx.SliceBase, target)
			if out.Loc != LocStackPair || !out.Rooted || out.NoHeapPointer {
				t.Fatalf("fixture requires an unknown pointer-bearing rooted stack result: %#v", out)
			}
			if out.SliceSizeKnown {
				t.Error("match output inherits the old target's slice shape")
			}
			if out.Type != JITTypeUnknown {
				t.Errorf("match output inherits old target type %d instead of its unknown arm type", out.Type)
			}
			root := jitStackRoot{base: jitStackRootFrameSP, offset: out.StackOff - ctx.DynamicSP}
			if out.StackOff < 0 {
				root.base, root.offset = jitStackRootFrameBP, out.StackOff
			}
			if _, live := ctx.StackRoots[root]; !live {
				t.Errorf("match returns Rooted=true but its live output slot %v is absent from StackRoots", root)
			}
			ctx.recordSafepoint(nil, 0)
			found := false
			for _, actual := range jitDecodeSafepointRoots(ctx.Safepoints[len(ctx.Safepoints)-1]) {
				found = found || actual == root
			}
			if !found {
				t.Errorf("following allocating-call safepoint omits the live match output slot %v", root)
			}
		})
	}
}

func TestJITMatchFreshStackResultSurvivesRuntimeGC(t *testing.T) {
	const name = "jit_test_match_stack_result_gc"
	declaration := &Declaration{
		Name: name,
		Fn:   func(args ...Scmer) Scmer { return NewSlice([]Scmer{args[0], NewString("tail")}) },
		Type: &TypeDescriptor{Kind: "func", Params: []*TypeDescriptor{{Kind: "any"}}, Return: &TypeDescriptor{Kind: "list"},
			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				previousEnv, previousStackPhi := ctx.Env, ctx.StackPhiTargets
				ctx.Env = &JITEnv{Numbered: args, Outer: previousEnv}
				ctx.StackPhiTargets = true
				expression := []Scmer{NewSymbol("match"), NewInt(1), NewSymbol("_"),
					NewSlice([]Scmer{NewSymbol("list"), NewNthLocalVar(0), NewString("tail")})}
				value := jitCompileMatch(ctx, expression, ctx.SliceBase, JITValueDesc{Loc: LocAny})
				ctx.Env, ctx.StackPhiTargets = previousEnv, previousStackPhi
				// Collection happens during invocation, after the fresh branch result
				// is stored. There is no collection during JIT compilation.
				ctx.EmitGoCallToFrame(GoFuncAddr(runtime.GC), nil, nil)
				return jitPlaceScmerIntoTarget(ctx, value, result)
			}},
	}
	Declare(&Globalenv, declaration)
	defer func() {
		delete(Globalenv.Vars, Symbol(name))
		delete(declarations, name)
		delete(declarationsByFunction, FunctionIdentity(declaration.Fn))
	}()
	compiled := compileJITExpressionTestProc(t, `(lambda (value) (jit_test_match_stack_result_gc value))`)
	value := NewString(strings.Repeat("kept-", 20))
	got := Apply(compiled, value)
	want := NewSlice([]Scmer{value, NewString("tail")})
	if !Equal(got, want) {
		t.Fatalf("match result after runtime collection = %s, want %s", SerializeToString(got, &Globalenv), SerializeToString(want, &Globalenv))
	}
}

// A restored generator allocator can reuse a result whose previous pointer
// was already dropped. The binding getter must not revive that old root while
// executing the Go call that will replace it.
func TestJITParserBindingDoesNotRootReusedResultBeforeReturn(t *testing.T) {
	ctx := newJITStackMapTestContext(make([]byte, 16384))
	ctx.AllRegs, ctx.FreeRegs = jitDefaultFreeGPRegs(), jitDefaultFreeGPRegs()
	ctx.SliceBase = RegR12
	ctx.StackPhiTargets = true
	ctx.RuntimeEnv = NewAny(&Globalenv)
	ctx.Env = &JITEnv{}
	stateOff := ctx.AllocStack(16)
	ctx.setStackPointer(jitStackRootFrameSP, stateOff, true)
	emitter := jitParserEmitter{
		ctx: ctx,
		program: &jitParserProgram{rules: []jitParserRule{{
			generator: NewSlice([]Scmer{NewSymbol("concat"), NewNthLocalVar(0), NewString("suffix")}),
			bindings:  []Symbol{Symbol("value")}, lexicalParent: -1,
		}}},
		state:       JITValueDesc{Loc: LocStackPair, Type: tagAny, StackOff: stateOff, Rooted: true},
		positionOff: ctx.AllocStack(8), continuationOff: ctx.AllocStack(8),
		generatorValueOff: ctx.AllocSpill(16), inlineActions: true,
		dispatchLabel: ctx.ReserveLabel(), ruleLabels: []JITLabel{ctx.ReserveLabel()},
	}
	ctx.setStackPointer(jitStackRootFrameBP, emitter.generatorValueOff, true)
	beforeDiscarded := ctx.SnapshotAllocState()
	discarded := JITEmitGoCallResults(ctx, GoFuncAddr(jitTestDiscardedStringResult), nil, []uint8{2}, []uint8{1})[0]
	ctx.RestoreAllocState(beforeDiscarded)
	// An intervening collection call no longer roots the discarded value.
	ctx.EmitGoCallToFrame(GoFuncAddr(runtime.GC), nil, nil)
	root := jitStackRoot{base: jitStackRootFrameBP, offset: discarded.StackOff}
	for _, actual := range jitDecodeSafepointRoots(ctx.Safepoints[len(ctx.Safepoints)-1]) {
		if actual == root {
			t.Fatal("fixture still roots its discarded result before parser binding emission")
		}
	}
	bindingSafepoint := len(ctx.Safepoints)
	emitter.emitRuleReturn(0, true)
	if len(ctx.Safepoints) <= bindingSafepoint {
		t.Fatal("parser binding emitted no Go-call safepoint")
	}
	for _, actual := range jitDecodeSafepointRoots(ctx.Safepoints[bindingSafepoint]) {
		if actual == root {
			t.Errorf("binding getter safepoint roots discarded result slot %v before the getter returns its replacement", root)
		}
	}
	rootedConsumer := false
	// The final rule-return call follows allocator restoration. Calls inside
	// the generator must retain the new binding while consuming it.
	for _, safepoint := range ctx.Safepoints[bindingSafepoint+1 : len(ctx.Safepoints)-1] {
		for _, actual := range jitDecodeSafepointRoots(safepoint) {
			rootedConsumer = rootedConsumer || actual == root
		}
	}
	if !rootedConsumer {
		t.Error("generator consumer never roots the binding after the getter has returned")
	}
}
