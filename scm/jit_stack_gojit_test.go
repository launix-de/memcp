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
	"bytes"
	"runtime"
	"sync"
	"testing"
	"time"
	"unsafe"
)

var jitFrameClearInstruction = []byte{0xf3, 0x48, 0xab}

func jitEntryCode(entry *JITEntryPoint) []byte {
	return unsafe.Slice((*byte)(entry.CodePtr), entry.CodeLen)
}

func TestJITFrameClearingLimitedToParserControlFlow(t *testing.T) {
	t.Run("query projection", func(t *testing.T) {
		compiled := compileJITExpressionTestProc(t, `(lambda (a b) (list "a" a "b" b))`)
		if bytes.Contains(jitEntryCode(compiled.Proc().Compiled), jitFrameClearInstruction) {
			t.Fatal("ordinary expression clears its entire JIT frame")
		}
	})

	t.Run("parser", func(t *testing.T) {
		compiled := compileJITExpressionTestProc(t, `(lambda (input) (begin
			(define word (parser (regex "[a-z]+" false false)))
			((parser '((define value word) $) value "") input)))`)
		if !bytes.Contains(jitEntryCode(compiled.Proc().Compiled), jitFrameClearInstruction) {
			t.Fatal("parser expression does not clear shared JIT frame targets")
		}
	})
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
	code := make([]byte, 128)
	start := unsafe.Pointer(&code[0])
	ctx := &JITContext{
		Start:      start,
		Ptr:        start,
		End:        unsafe.Add(start, len(code)-1),
		AllRegs:    1 << uint(RegRAX),
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

func TestJITUnboxedScalarStabilizationDoesNotCreateGCStackRoot(t *testing.T) {
	t.Run("control flow", func(t *testing.T) {
		code := make([]byte, 128)
		ctx := newJITStackMapTestContext(code)
		value := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: RegRAX}
		ctx.BindReg(RegRAX, &value)

		ctx.StabilizeDescForControlFlow(&value)
		if _, exists := ctx.StackRoots[jitStackRoot{base: jitStackRootFrameSP, offset: 0}]; exists {
			t.Fatal("unboxed control-flow scalar is marked as a GC pointer")
		}
	})

	t.Run("nested call", func(t *testing.T) {
		code := make([]byte, 128)
		ctx := newJITStackMapTestContext(code)
		value := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: RegRAX}
		ctx.BindReg(RegRAX, &value)

		ctx.StabilizeDescAcrossNestedCall(&value)
		if _, exists := ctx.StackRoots[jitStackRoot{base: jitStackRootFrameBP, offset: -8}]; exists {
			t.Fatal("unboxed nested-call scalar is marked as a GC pointer")
		}
	})

	t.Run("parser environment", func(t *testing.T) {
		code := make([]byte, 128)
		ctx := newJITStackMapTestContext(code)
		value := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: RegRAX}
		ctx.BindReg(RegRAX, &value)

		stable := ctx.StabilizeJITEnv(&JITEnv{Vars: map[Symbol]JITValueDesc{"value": value}})
		if _, exists := ctx.StackRoots[jitStackRoot{base: jitStackRootFrameBP, offset: -8}]; exists {
			t.Fatal("unboxed parser-environment scalar is marked as a GC pointer")
		}
		if got := stable.Vars["value"]; got.Loc != LocStack || got.RelocatablePointer {
			t.Fatalf("unexpected stabilized scalar: %+v", got)
		}
	})
}

func TestJITRelocatableScalarStabilizationCreatesGCStackRoot(t *testing.T) {
	code := make([]byte, 128)
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
}

func TestJITArenaDefersCallbackUntilStackMapPublication(t *testing.T) {
	first := &jitCodeReservation{}
	second := &jitCodeReservation{}
	arena := &jitArena{reservations: []*jitCodeReservation{first, second}}
	arena.metaCond = sync.NewCond(&arena.metaMu)
	published := false
	arena.completeDeferred(second, nil, func() { published = true })
	if published {
		t.Fatal("deferred entry became visible before the preceding stack maps")
	}
	arena.complete(first, nil)
	if !published || !second.published {
		t.Fatal("deferred entry was not installed with its stack maps")
	}
}

func TestJITArenaParentWaitsForInterleavedDeferredStackMaps(t *testing.T) {
	parent := &jitCodeReservation{}
	otherCompiler := &jitCodeReservation{}
	child := &jitCodeReservation{}
	arena := &jitArena{reservations: []*jitCodeReservation{parent, otherCompiler, child}}
	arena.metaCond = sync.NewCond(&arena.metaMu)
	arena.completeDeferred(child, nil, nil)

	parentComplete := make(chan struct{})
	go func() {
		arena.complete(parent, nil)
		close(parentComplete)
	}()
	select {
	case <-parentComplete:
		t.Fatal("parent became reachable before its interleaved child stack maps")
	case <-time.After(20 * time.Millisecond):
	}

	arena.complete(otherCompiler, nil)
	select {
	case <-parentComplete:
	case <-time.After(time.Second):
		t.Fatal("parent did not become reachable after its child stack maps")
	}
	if !parent.published || !otherCompiler.published || !child.published {
		t.Fatal("interleaved stack maps were not published as one reachable prefix")
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

func TestJITRegisterHomesTradeOuterForMoreValuableInnerPlan(t *testing.T) {
	code := make([]byte, 256)
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
	code := make([]byte, 256)
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
