//go:build riscv64

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

func TestJITRISCV64UniversalMoveAndArithmeticEmitters(t *testing.T) {
	buffer := make([]byte, 128)
	ctx := &JITContext{Start: unsafe.Pointer(&buffer[0]), Ptr: unsafe.Pointer(&buffer[0]), End: unsafe.Pointer(&buffer[len(buffer)-1])}
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
		0x00050593, 0x00853583, 0x00B53823,
		0x00154583, 0x00255583, 0x00456583,
		0x00B500A3, 0x00B51123, 0x00B52223,
		0x00B56533, 0x00B57533, 0x00B54533,
		0x00B50533, 0x40B50533,
		0x00B50533, 0x02051513, 0x02055513,
		0x40B50533, 0x02051513, 0x02055513,
		0x02B50533,
	}
	got := make([]uint32, len(want))
	for index := range got {
		got[index] = binary.LittleEndian.Uint32(buffer[index*4:])
	}
	if !slices.Equal(got, want) {
		t.Fatalf("unexpected riscv64 code:\n got %#x\nwant %#x", got, want)
	}
}
