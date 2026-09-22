//go:build arm64

/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.
*/

package scm

import (
	"encoding/binary"
	"slices"
	"testing"
	"unsafe"
)

func TestJITARM64UniversalMoveAndArithmeticEmitters(t *testing.T) {
	buffer := make([]byte, 128)
	ctx := &JITContext{Start: unsafe.Pointer(&buffer[0]), Ptr: unsafe.Pointer(&buffer[0]), End: unsafe.Pointer(&buffer[len(buffer)-1])}
	ctx.EmitMovRegImm64(RegRAX, 0x1122334455667788)
	ctx.EmitMovRegReg(RegRBX, RegRAX)
	ctx.FlushRegisterMoves()
	ctx.EmitMovRegMem(RegRBX, RegRAX, 8)
	ctx.EmitStoreRegMem(RegRBX, RegRAX, 16)
	ctx.EmitMovRegMemB(RegRBX, RegRAX, 1)
	ctx.EmitMovRegMemW(RegRBX, RegRAX, 2)
	ctx.EmitMovRegMemL(RegRBX, RegRAX, 4)
	ctx.EmitStoreRegMemB(RegRBX, RegRAX, 1)
	ctx.EmitStoreRegMemW(RegRBX, RegRAX, 2)
	ctx.EmitStoreRegMemL(RegRBX, RegRAX, 4)
	ctx.EmitOrInt64(RegRAX, RegRBX)
	ctx.EmitAndInt64(RegRAX, RegRBX)
	ctx.EmitXorInt64(RegRAX, RegRBX)
	ctx.EmitAddInt64(RegRAX, RegRBX)
	ctx.EmitSubInt64(RegRAX, RegRBX)
	ctx.EmitAddInt32(RegRAX, RegRBX)
	ctx.EmitSubInt32(RegRAX, RegRBX)
	ctx.EmitImulInt64(RegRAX, RegRBX)

	want := []uint32{
		0xD28EF100, 0xF2AAACC0, 0xF2C66880, 0xF2E22440,
		0xAA0003E1, 0xF9400401, 0xF9000801,
		0x39400401, 0x79400401, 0xB9400401,
		0x39000401, 0x79000401, 0xB9000401,
		0xAA010000, 0x8A010000, 0xCA010000,
		0x8B010000, 0xCB010000, 0x0B010000, 0x4B010000, 0x9B017C00,
	}
	got := make([]uint32, len(want))
	for index := range got {
		got[index] = binary.LittleEndian.Uint32(buffer[index*4:])
	}
	if !slices.Equal(got, want) {
		t.Fatalf("unexpected arm64 code:\n got %#x\nwant %#x", got, want)
	}
}

func TestJITARM64SelectsImmediateAndLateLoadOperands(t *testing.T) {
	buffer := make([]byte, 64)
	ctx := &JITContext{
		Start: unsafe.Pointer(&buffer[0]), Ptr: unsafe.Pointer(&buffer[0]), End: unsafe.Pointer(&buffer[len(buffer)-1]),
		ScratchReg: RegR11, StackReg: RegRSP, FrameReg: RegRBP,
	}
	ctx.EmitIntBinaryImm(JITIntAdd, 64, RegRAX, 7)
	ctx.EmitIntBinaryImm(JITIntSub, 64, RegRAX, -4096)
	stack := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: 8}
	ctx.EmitIntBinary(JITIntAdd, 64, RegRAX, &stack)

	want := []uint32{0x91001C00, 0x91400400, 0xF94007F0, 0x8B100000}
	got := make([]uint32, len(want))
	for index := range got {
		got[index] = binary.LittleEndian.Uint32(buffer[index*4:])
	}
	if !slices.Equal(got, want) {
		t.Fatalf("unexpected arm64 selection:\n got %#x\nwant %#x", got, want)
	}
}
