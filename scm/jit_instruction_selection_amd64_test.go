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
		SliceBase: RegR12, ScratchReg: RegR11, StackReg: RegRSP, FrameReg: RegRBP, RegisterBank: jitX86RegisterBank,
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
