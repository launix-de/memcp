//go:build goexperiment.jit && amd64

/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

package scm

import (
	"bytes"
	"math"
	"runtime"
	"testing"
	"unsafe"
)

func TestEmitCmpFloat64AvoidsDuplicateSameOperandMove(t *testing.T) {
	code := make([]byte, 16)
	ctx := &JITContext{
		Start: unsafe.Pointer(&code[0]),
		Ptr:   unsafe.Pointer(&code[0]),
		End:   unsafe.Pointer(&code[len(code)-1]),
	}
	ctx.EmitCmpFloat64(RegRAX, RegRAX)
	emitted := code[:uintptr(ctx.Ptr)-uintptr(ctx.Start)]
	want := []byte{
		0x66, 0x48, 0x0f, 0x6e, 0xc0, // MOVQ XMM0, RAX
		0x66, 0x0f, 0x2e, 0xc0, // UCOMISD XMM0, XMM0
	}
	if !bytes.Equal(emitted, want) {
		t.Fatalf("same-operand float comparison = %x, want %x", emitted, want)
	}
}

func emitParallelMoveTestCode(t *testing.T, batch *jitParallelRegMoveBatch) []byte {
	t.Helper()
	code := make([]byte, 128)
	ctx := &JITContext{
		Start:        unsafe.Pointer(&code[0]),
		Ptr:          unsafe.Pointer(&code[0]),
		End:          unsafe.Pointer(&code[len(code)-1]),
		SliceBase:    RegR12,
		ScratchReg:   RegR11,
		StackReg:     RegRSP,
		FrameReg:     RegRBP,
		RegisterBank: jitX86RegisterBank,
	}
	ctx.emitParallelRegMoveBatch(batch)
	if ctx.DynamicSP != 0 {
		t.Fatalf("parallel move left dynamic stack offset %d", ctx.DynamicSP)
	}
	return code[:uintptr(ctx.Ptr)-uintptr(ctx.Start)]
}

func TestParallelMoveBatchElidesIdentityAndOrdersDependencies(t *testing.T) {
	var batch jitParallelRegMoveBatch
	batch.add(RegRDX, RegRDX)
	batch.add(RegRAX, RegRBX)
	batch.add(RegRCX, RegRAX)

	// RCX must consume the old RAX before RAX is overwritten by RBX. Both x86
	// register moves are three bytes; the identity must emit nothing.
	code := emitParallelMoveTestCode(t, &batch)
	want := []byte{0x48, 0x89, 0xc1, 0x48, 0x89, 0xd8}
	if !bytes.Equal(code, want) {
		t.Fatalf("parallel dependency moves = %x, want %x", code, want)
	}
}

func TestParallelMoveBatchBreaksCycleWithOneSavedScratch(t *testing.T) {
	var batch jitParallelRegMoveBatch
	batch.add(RegRAX, RegRBX)
	batch.add(RegRBX, RegRAX)

	code := emitParallelMoveTestCode(t, &batch)
	// PUSH/POP R12 surround exactly three MOVs: save old RAX in scratch,
	// rotate RBX into RAX, then scratch into RBX.
	if len(code) != 13 || !bytes.Equal(code[:2], []byte{0x41, 0x54}) || !bytes.Equal(code[len(code)-2:], []byte{0x41, 0x5c}) {
		t.Fatalf("parallel cycle is not one saved-scratch rotation: %x", code)
	}
}

func TestParallelMoveBatchChoosesScratchOutsideCycle(t *testing.T) {
	var batch jitParallelRegMoveBatch
	batch.add(RegR12, RegR11)
	batch.add(RegR11, RegR12)

	// Both preferred role registers participate in the cycle. The solver must
	// select another register from the architecture-provided bank; reusing either
	// cycle member would fail to break the dependency (the old R12-specific
	// implementation could loop forever for this shape).
	code := emitParallelMoveTestCode(t, &batch)
	if len(code) != 13 {
		t.Fatalf("role-register cycle emitted %d bytes, want one saved-scratch rotation (13): %x", len(code), code)
	}
}

func TestParallelMoveBatchNeverUsesStackOrFrameRegisterAsScratch(t *testing.T) {
	var batch jitParallelRegMoveBatch
	batch.add(RegRAX, RegRBX)
	batch.add(RegRBX, RegRAX)

	ctx := JITContext{
		// Stack-backed argument lists deliberately use RSP as SliceBase. This
		// makes it a valid address base, not a writable temporary register.
		SliceBase:    RegRSP,
		ScratchReg:   RegR11,
		StackReg:     RegRSP,
		FrameReg:     RegRBP,
		RegisterBank: jitX86RegisterBank,
	}
	if scratch := ctx.parallelMoveScratch(&batch); scratch != RegR11 {
		t.Fatalf("parallel cycle scratch = %d, want non-frame scratch %d", scratch, RegR11)
	}
}

func TestDeferredRegisterMovesCollapseAcrossEmitterBoundaries(t *testing.T) {
	code := make([]byte, 128)
	ctx := &JITContext{
		Start:        unsafe.Pointer(&code[0]),
		Ptr:          unsafe.Pointer(&code[0]),
		End:          unsafe.Pointer(&code[len(code)-1]),
		SliceBase:    RegR12,
		ScratchReg:   RegR11,
		StackReg:     RegRSP,
		RegisterBank: jitX86RegisterBank,
	}

	// Model two independent inline emitters handing the same value through an
	// otherwise dead intermediate register. Ending the producer lifetime must
	// reserve its physical register for the alias rather than forcing an early
	// copy. The physical stream needs only the final RDI -> RDX move.
	ctx.AllRegs = uint64(jitRegisterMask(RegRDI, RegRSI, RegRDX))
	ctx.EmitMovRegReg(RegRSI, RegRDI)
	ctx.FreeReg(RegRDI)
	ctx.EmitMovRegReg(RegRDX, RegRSI)
	ctx.FreeReg(RegRSI)
	if ctx.Ptr != ctx.Start {
		t.Fatal("deferred moves emitted before a materialization barrier")
	}
	if ctx.FreeRegs&uint64(jitRegisterMask(RegRDI)) != 0 {
		t.Fatal("aliased physical source returned to allocator before materialization")
	}
	ctx.FlushRegisterMoves()
	emitted := code[:uintptr(ctx.Ptr)-uintptr(ctx.Start)]
	if want := []byte{0x48, 0x89, 0xfa}; !bytes.Equal(emitted, want) {
		t.Fatalf("collapsed deferred chain = %x, want %x", emitted, want)
	}
	if ctx.FreeRegs&uint64(jitRegisterMask(RegRDI)) == 0 {
		t.Fatal("physical source remained held after its final alias materialized")
	}
}

func TestDeferredRegisterMovesPreserveOldSourceBeforeOverwrite(t *testing.T) {
	code := make([]byte, 128)
	ctx := &JITContext{
		Start:        unsafe.Pointer(&code[0]),
		Ptr:          unsafe.Pointer(&code[0]),
		End:          unsafe.Pointer(&code[len(code)-1]),
		SliceBase:    RegR12,
		ScratchReg:   RegR11,
		StackReg:     RegRSP,
		RegisterBank: jitX86RegisterBank,
	}

	ctx.EmitMovRegReg(RegRSI, RegRDI)
	ctx.EmitMovRegReg(RegRDI, RegRAX)
	ctx.FlushRegisterMoves()
	// RSI must receive the old RDI before the later RAX -> RDI assignment.
	emitted := code[:uintptr(ctx.Ptr)-uintptr(ctx.Start)]
	want := []byte{0x48, 0x89, 0xfe, 0x48, 0x89, 0xc7}
	if !bytes.Equal(emitted, want) {
		t.Fatalf("overwrite-preserving deferred moves = %x, want %x", emitted, want)
	}
}

func jitFourScalarResults(seed uint32) (int64, bool, int64, int64) {
	return int64(seed) + 1, seed&1 != 0, int64(seed) + 3, int64(seed) + 5
}

func TestJITScmerConstructorsAllowAliasedPayloadRegister(t *testing.T) {
	tests := []struct {
		name string
		want Scmer
		emit func(*JITContext, JITValueDesc, JITValueDesc)
	}{
		{"int", NewInt(42), func(ctx *JITContext, dst, src JITValueDesc) {
			ctx.EmitMovRegImm64(src.Reg, 42)
			ctx.EmitMakeInt(dst, src)
		}},
		{"float", NewFloat(-157.84), func(ctx *JITContext, dst, src JITValueDesc) {
			ctx.EmitMovRegImm64(src.Reg, math.Float64bits(-157.84))
			ctx.EmitMakeFloat(dst, src)
		}},
		{"bool", NewBool(true), func(ctx *JITContext, dst, src JITValueDesc) {
			ctx.EmitMovRegImm64(src.Reg, 1)
			ctx.EmitMakeBool(dst, src)
		}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			fn := CompileJITStorageGetValue(func(ctx *JITContext, source, target JITValueDesc) JITValueDesc {
				if source.Reg != target.Reg {
					t.Fatalf("test requires an aliased source and pointer destination, got %v and %v", source.Reg, target.Reg)
				}
				test.emit(ctx, target, source)
				return target
			})
			if fn == nil {
				t.Fatal("aliased constructor did not compile")
			}
			if got := fn(0); !Equal(got, test.want) {
				t.Fatalf("aliased constructor = %v, want %v", got, test.want)
			}
		})
	}
}

func TestJITStorageScalarInputSurvivesScratchReclamation(t *testing.T) {
	fn := CompileJITStorageGetValue(func(ctx *JITContext, source, target JITValueDesc) JITValueDesc {
		// Generated CFG emitters reclaim registers whose descriptors no longer own
		// them. The incoming RAX is still live here and therefore must not become
		// scratch merely because it is also the eventual result register.
		ctx.ReclaimUntrackedRegs()
		scratch := ctx.AllocReg()
		ctx.EmitMovRegImm64(scratch, 99)
		ctx.FreeReg(scratch)
		value := JITValueDesc{Loc: LocRegPair, Type: tagInt, Reg: target.Reg, Reg2: target.Reg2}
		ctx.EmitMakeInt(value, source)
		return value
	})
	if fn == nil {
		t.Fatal("scalar storage JIT function did not compile")
	}
	if got, want := fn(17), NewInt(17); !Equal(got, want) {
		t.Fatalf("scalar input after scratch reclamation = %v, want %v", got, want)
	}
}

func TestJITLessHelperEmitsKnownIntegerComparison(t *testing.T) {
	fn := CompileJITStorageGetValue(func(ctx *JITContext, source, target JITValueDesc) JITValueDesc {
		source.Type = tagInt
		comparison := jitEmitLess(ctx, []JITValueDesc{
			source,
			{Loc: LocImm, Type: tagInt, Imm: NewInt(511)},
		}, JITValueDesc{Loc: LocReg, Type: tagBool, Reg: target.Reg2, ID: 0})
		ctx.EmitMakeBool(target, comparison)
		return target
	})
	if fn == nil {
		t.Fatal("known integer Less helper did not compile")
	}
	for _, test := range []struct {
		value uint32
		want  bool
	}{{510, true}, {511, false}, {512, false}} {
		if got := fn(test.value).Bool(); got != test.want {
			t.Fatalf("(< %d 511) = %v, want %v", test.value, got, test.want)
		}
	}
}

func TestJITStorageReadersSharePackedCodeLifetime(t *testing.T) {
	scalar, ranged, multi := CompileJITStorageReaders(
		func(ctx *JITContext, source, target JITValueDesc) JITValueDesc {
			ctx.EmitMakeInt(target, source)
			return target
		},
		func(_ *JITContext, _, _, _, _, result JITValueDesc) JITValueDesc { return result },
		func(_ *JITContext, _, _, _, result JITValueDesc) JITValueDesc { return result },
	)
	if scalar == nil || ranged == nil || multi == nil {
		t.Fatal("storage reader batch did not compile every ABI")
	}

	scalarValue := *(*unsafe.Pointer)(unsafe.Pointer(&scalar))
	rangeValue := *(*unsafe.Pointer)(unsafe.Pointer(&ranged))
	multiValue := *(*unsafe.Pointer)(unsafe.Pointer(&multi))
	scalarHolder := (*jitStorageFuncValue)(scalarValue)
	rangeHolder := (*jitStorageFuncValue)(rangeValue)
	multiHolder := (*jitStorageFuncValue)(multiValue)
	if scalarHolder.owner == nil || scalarHolder.owner != rangeHolder.owner || scalarHolder.owner != multiHolder.owner {
		t.Fatal("storage reader funcvals do not share one code owner")
	}
	if len(scalarHolder.owner.entries) != 3 {
		t.Fatalf("shared code owner has %d entries, want 3", len(scalarHolder.owner.entries))
	}
	if gap := rangeHolder.code - scalarHolder.code; gap == 0 || gap >= 16*1024 {
		t.Fatalf("scalar-to-range code gap = %d, want densely packed functions", gap)
	}
	if gap := multiHolder.code - rangeHolder.code; gap == 0 || gap >= 16*1024 {
		t.Fatalf("range-to-multi code gap = %d, want densely packed functions", gap)
	}
	runtime.GC()
	if got := scalar(23); !Equal(got, NewInt(23)) {
		t.Fatalf("packed scalar result = %v, want 23", got)
	}
}

func TestJITGoCallFourScalarResults(t *testing.T) {
	fn := CompileJITStorageGetValue(func(ctx *JITContext, seed, target JITValueDesc) JITValueDesc {
		results := JITEmitGoCallResults(ctx, GoFuncAddr(jitFourScalarResults), []JITValueDesc{seed}, []uint8{1, 1, 1, 1}, []uint8{0, 0, 0, 0})
		for index := range results {
			ctx.EnsureDesc(&results[index])
		}
		ctx.EmitImulRegImm32(results[1].Reg, 10)
		ctx.EmitImulRegImm32(results[2].Reg, 100)
		ctx.EmitImulRegImm32(results[3].Reg, 1000)
		ctx.EmitAddInt64(results[0].Reg, results[1].Reg)
		ctx.EmitAddInt64(results[0].Reg, results[2].Reg)
		ctx.EmitAddInt64(results[0].Reg, results[3].Reg)
		ctx.EnsureDesc(&seed)
		ctx.EmitAddInt64(results[0].Reg, seed.Reg)
		value := JITValueDesc{Loc: LocRegPair, Type: tagInt, Reg: target.Reg, Reg2: target.Reg2}
		ctx.EmitMakeInt(value, results[0])
		return value
	})
	if fn == nil {
		t.Fatal("four-result JIT function did not compile")
	}
	if got, want := fn(7), NewInt(8+10+1000+12000+7); !Equal(got, want) {
		t.Fatalf("four-result Go ABI call = %v, want %v", got, want)
	}
}

func TestJITStoreCustomScmerKeepsSpilledAddress(t *testing.T) {
	payload := new(byte)
	value := NewCustom(200, unsafe.Pointer(payload))
	fn := CompileJITStorageGetValueRange(func(ctx *JITContext, index, count, target, stride, result JITValueDesc) JITValueDesc {
		address := ctx.EmitSliceElementAddress(&target, &index, 16)
		addressOff := ctx.AllocStack(8)
		ctx.EmitStoreRegMem(address.Reg, ctx.StackReg, addressOff)
		ctx.FreeDesc(&address)
		address = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: addressOff, NoHeapPointer: true}
		stored := JITValueDesc{Loc: LocImm, Type: value.GetTag(), Imm: value, Rooted: true}
		ctx.EmitStoreScmerAt(&address, &stored)
		return result
	})
	if fn == nil {
		t.Fatal("spilled-address JIT function did not compile")
	}
	target := make([]Scmer, 1)
	fn(0, 1, target, 1)
	runtime.GC()
	if got := target[0]; got.GetTag() != value.GetTag() || got.ptr != value.ptr {
		t.Fatalf("stored custom Scmer = %v, want tag %d at %p", got, value.GetTag(), value.ptr)
	}
	runtime.KeepAlive(payload)
}
