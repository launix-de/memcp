//go:build !arm64 && !riscv64

/*
Copyright (C) 2026 Carl-Philip Hänsch
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/
package scm

import "unsafe"

const jitNativeTemp = RegR11
const jitGoStackArgOffset = int32(0)

func jitGoABIIntRegs() []Reg {
	return []Reg{RegRAX, RegRBX, RegRCX, RegRDI, RegRSI, RegR8, RegR9, RegR10, RegR11}
}
func jitArchEmitNative(*JITContext, jitNativeOp, Reg, Reg, Reg, int64) {
	panic("jit: fixed-width instruction on unsupported architecture")
}
func jitArchEmitBranch(*JITContext, JITCondition, JITLabel, bool) { panic("jit: unsupported branch") }
func jitArchEmitBranchPlaceholder(*JITContext)                    { panic("jit: unsupported branch") }
func jitArchPatchBranch(unsafe.Pointer, int32)                    { panic("jit: unsupported branch patch") }
func jitArchEmitFrameFixup(*JITContext) unsafe.Pointer            { panic("jit: unsupported frame") }
func jitArchEmitImmediateFixup(*JITContext, Reg) unsafe.Pointer   { panic("jit: unsupported immediate") }
func jitArchPatchImmediate(unsafe.Pointer, int32)                 { panic("jit: unsupported immediate patch") }

func jitArchRegisterBank() JITRegisterBank {
	return JITRegisterBank{Registers: [16]Reg{RegR13, RegR10, RegR9, RegR8, RegRDI, RegRSI, RegRCX, RegRDX, RegRAX, RegRBX}, Count: 10, TemporaryReserve: 7}
}
func jitArchFPRegisterBank() JITRegisterBank {
	return JITRegisterBank{Registers: [16]Reg{RegX2, RegX3, RegX4, RegX5, RegX6, RegX7, RegX8, RegX9, RegX10, RegX11, RegX12, RegX13, RegX14}, Count: 13, TemporaryReserve: 2}
}
func jitArchFreeGPRegs() uint64 {
	return uint64(1<<uint(RegRCX) | 1<<uint(RegRDX) | 1<<uint(RegRSI) | 1<<uint(RegRDI) | 1<<uint(RegR8) | 1<<uint(RegR9) | 1<<uint(RegR10) | 1<<uint(RegR13) | 1<<uint(RegR15))
}

func jitArchEmitShift64(ctx *JITContext, dst, count Reg, left bool) {
	work := dst
	if dst == RegRCX {
		work = RegR11
		if work == count {
			work = RegRAX
		}
		ctx.EmitPushReg(work)
		jitArchEmitMovRegReg(ctx, work, dst)
	}
	if count != RegRCX {
		if dst != RegRCX {
			ctx.EmitPushReg(RegRCX)
		}
		jitArchEmitMovRegReg(ctx, RegRCX, count)
	}
	rex := byte(0x48)
	if work >= 8 {
		rex |= 1
	}
	opcode := byte(0xE8)
	if left {
		opcode = 0xE0
	}
	ctx.emitBytes(rex, 0xD3, opcode|byte(work&7))
	if dst == RegRCX {
		jitArchEmitMovRegReg(ctx, dst, work)
		ctx.EmitPopReg(work)
	} else if count != RegRCX {
		ctx.EmitPopReg(RegRCX)
	}
}
