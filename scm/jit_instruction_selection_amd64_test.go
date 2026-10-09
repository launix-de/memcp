//go:build amd64

/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.
*/

package scm

import (
	"bytes"
	"testing"
	"unsafe"
)

func TestJITBooleanFlagsMaterializeAtBarriers(t *testing.T) {
	for _, barrier := range []string{"value", "instruction", "raw", "label", "snapshot"} {
		t.Run(barrier, func(t *testing.T) {
			buffer := make([]byte, 256)
			ctx := &JITContext{Start: unsafe.Pointer(&buffer[0]), Ptr: unsafe.Pointer(&buffer[0]), End: unsafe.Pointer(&buffer[len(buffer)-1]),
				ScratchReg: RegR11, StackReg: RegRSP, FrameReg: RegRBP}
			ctx.EmitCmpInt64(RegRAX, RegRBX)
			before := uintptr(ctx.Ptr)
			value := ctx.DeferBooleanFlags(RegRCX, CondSignedLess)
			if uintptr(ctx.Ptr) != before {
				t.Fatal("recording boolean flags emitted machine code")
			}
			switch barrier {
			case "value":
				ctx.EnsureDesc(&value)
			case "instruction":
				ctx.EmitAddRegImm32(RegRAX, 1)
			case "raw":
				ctx.EmitByte(0x90)
			case "label":
				ctx.MarkLabel(ctx.ReserveLabel())
			case "snapshot":
				ctx.SnapshotAllocState()
			}
			ctx.EnsureDesc(&value)
			if value.Loc != LocReg || ctx.lazyFlags.FlagsID != 0 {
				t.Fatal("barrier left stale flags")
			}
			// SETL CL; MOVZX ECX, CL must precede the clobbering instruction.
			got := buffer[before-uintptr(ctx.Start) : uintptr(ctx.Ptr)-uintptr(ctx.Start)]
			want := []byte{0x0f, 0x9c, 0xc1, 0x0f, 0xb6, 0xc9}
			if !bytes.HasPrefix(got, want) || bytes.Count(got, want) != 1 {
				t.Fatalf("boolean not materialized exactly once before barrier: %x", got)
			}
		})
	}
}

func TestJITAMD64SelectsDirectIntegerOperands(t *testing.T) {
	buffer := make([]byte, 64)
	ctx := &JITContext{
		Start: unsafe.Pointer(&buffer[0]), Ptr: unsafe.Pointer(&buffer[0]), End: unsafe.Pointer(&buffer[len(buffer)-1]),
		ScratchReg: RegR11, StackReg: RegRSP, FrameReg: RegRBP,
	}
	ctx.EmitIntBinaryImm(JITIntAdd, 64, RegRAX, 7)
	ctx.EmitIntBinaryImm(JITIntSub, 64, RegRAX, -7)
	stack := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: 8}
	ctx.EmitIntBinary(JITIntAdd, 64, RegRAX, &stack)
	ctx.EmitIntBinaryImm(JITIntMul, 64, RegRAX, 3)
	ctx.EmitIntBinaryImm(JITIntMul, 64, RegR9, 3)
	beforeIdentity := uintptr(ctx.Ptr)
	ctx.EmitIntBinaryImm(JITIntAdd, 64, RegRAX, 0)
	ctx.EmitIntBinaryImm(JITIntMul, 64, RegRAX, 1)
	if uintptr(ctx.Ptr) != beforeIdentity {
		t.Fatal("identity operands emitted machine code")
	}

	want := []byte{
		0x48, 0x83, 0xC0, 0x07,
		0x48, 0x83, 0xE8, 0xF9,
		0x48, 0x03, 0x44, 0x24, 0x08,
		0x48, 0x6B, 0xC0, 0x03,
		0x4D, 0x6B, 0xC9, 0x03,
	}
	got := buffer[:uintptr(ctx.Ptr)-uintptr(ctx.Start)]
	if !bytes.Equal(got, want) {
		t.Fatalf("unexpected amd64 selection:\n got %x\nwant %x", got, want)
	}
}

func TestJITAMD64IntegerSelectionPreservesDeferredMoves(t *testing.T) {
	buffer := make([]byte, 64)
	ctx := &JITContext{
		Start: unsafe.Pointer(&buffer[0]), Ptr: unsafe.Pointer(&buffer[0]), End: unsafe.Pointer(&buffer[len(buffer)-1]),
		SliceBase: RegR12, ScratchReg: RegR11, StackReg: RegRSP, FrameReg: RegRBP, RegisterBank: jitNativeRegisterBank,
	}

	ctx.EmitMovRegReg(RegR11, RegRDI)
	ctx.EmitIntBinaryImm(JITIntAdd, 64, RegRAX, 7)
	ctx.FlushRegisterMoves()
	want := []byte{
		0x48, 0x83, 0xC0, 0x07,
		0x49, 0x89, 0xFB,
	}
	got := buffer[:uintptr(ctx.Ptr)-uintptr(ctx.Start)]
	if !bytes.Equal(got, want) {
		t.Fatalf("short immediate discarded deferred scratch move:\n got %x\nwant %x", got, want)
	}

	ctx.Start = unsafe.Pointer(&buffer[0])
	ctx.Ptr = ctx.Start
	ctx.End = unsafe.Pointer(&buffer[len(buffer)-1])
	ctx.DeferredRegMoves = jitDeferredRegMoves{}
	ctx.EmitMovRegReg(RegRAX, RegRDI)
	ctx.EmitIntBinaryImm(JITIntAdd, 64, RegRAX, 0)
	got = buffer[:uintptr(ctx.Ptr)-uintptr(ctx.Start)]
	want = []byte{0x48, 0x89, 0xF8}
	if !bytes.Equal(got, want) {
		t.Fatalf("identity selection did not materialize its input:\n got %x\nwant %x", got, want)
	}
}

func TestJITDeferredBooleanBookkeepingDoesNotAllocate(t *testing.T) {
	buffer := make([]byte, 64)
	ctx := &JITContext{Start: unsafe.Pointer(&buffer[0]), Ptr: unsafe.Pointer(&buffer[0]), End: unsafe.Pointer(&buffer[len(buffer)-1]),
		ScratchReg: RegR11, StackReg: RegRSP, FrameReg: RegRBP}
	allocations := testing.AllocsPerRun(100, func() {
		ctx.Ptr = ctx.Start
		ctx.EmitCmpInt64(RegRAX, RegRBX)
		value := ctx.DeferBooleanFlags(RegRCX, CondSignedLess)
		ctx.EnsureDesc(&value)
	})
	if allocations != 0 {
		t.Fatalf("deferred boolean bookkeeping allocated %g objects", allocations)
	}
}

func TestJITUnusedBooleanDoesNotMaterialize(t *testing.T) {
	buffer := make([]byte, 64)
	ctx := &JITContext{Start: unsafe.Pointer(&buffer[0]), Ptr: unsafe.Pointer(&buffer[0]), End: unsafe.Pointer(&buffer[len(buffer)-1]),
		ScratchReg: RegR11, StackReg: RegRSP, FrameReg: RegRBP}
	ctx.EmitCmpInt64(RegRAX, RegRBX)
	before := ctx.Ptr
	value := ctx.DeferBooleanFlags(RegRCX, CondSignedLess)
	ctx.BindReg(value.Reg, &value)
	ctx.FreeDesc(&value)
	ctx.FlushRegisterMoves()
	if ctx.Ptr != before || ctx.lazyFlags.FlagsID != 0 {
		t.Fatal("unused boolean was materialized")
	}
}

func TestJITAllocatorRetainsDeferredBooleanRegister(t *testing.T) {
	buffer := make([]byte, 128)
	ctx := &JITContext{Start: unsafe.Pointer(&buffer[0]), Ptr: unsafe.Pointer(&buffer[0]), End: unsafe.Pointer(&buffer[len(buffer)-1]),
		ScratchReg: RegR11, StackReg: RegRSP, FrameReg: RegRBP,
		AllRegs: 1<<RegRAX | 1<<RegRBX, FreeRegs: 1 << RegRBX}
	ctx.EmitCmpInt64(RegRDI, RegRSI)
	value := ctx.DeferBooleanFlags(RegRAX, CondSignedLess)
	ctx.BindReg(RegRAX, &value)
	got := ctx.AllocReg()
	owner := ctx.RegOwners[RegRAX]
	if got == RegRAX || owner == nil || owner.ID != value.ID || owner.FlagsID != value.FlagsID {
		t.Fatal("allocator reclaimed the reserved register of a live deferred boolean")
	}
	if !ctx.hasBooleanFlags(value) {
		t.Fatal("allocating a free register unnecessarily materialized the comparison")
	}
}
