//go:build arm64

/*
Copyright (C) 2026 Carl-Philip Hänsch
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/
package scm

import "unsafe"

const jitNativeTemp Reg = 17 // IP1; outside the allocator and Go argument bank.
const jitGoStackArgOffset = int32(8)

const (
	arm64FloatEqual     Reg = 21
	arm64FloatBelow     Reg = 22
	arm64FloatUnordered Reg = 23
)

// X18 is the platform register on several operating systems. Keep temporary
// addresses and arithmetic in registers unused by the allocator and argument
// bank, selecting a distinct register before overwriting an operand.
func arm64TempExcept(regs ...Reg) Reg {
	for _, candidate := range [...]Reg{jitNativeTemp, 24, 25} {
		available := true
		for _, reg := range regs {
			if candidate == reg {
				available = false
				break
			}
		}
		if available {
			return candidate
		}
	}
	panic("jit: arm64 temporary registers exhausted")
}

func jitGoABIIntRegs() []Reg { return []Reg{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15} }

func jitArchRegisterBank() JITRegisterBank {
	return JITRegisterBank{
		Registers:        [16]Reg{20, 8, 7, 6, 5, 4, 2, 26, 0, 1},
		Count:            10,
		TemporaryReserve: 7,
	}
}

func jitArchFPRegisterBank() JITRegisterBank {
	return JITRegisterBank{
		Registers: [16]Reg{
			RegX2, RegX3, RegX4, RegX5, RegX6, RegX7, RegX8,
			RegX9, RegX10, RegX11, RegX12, RegX13, RegX14,
		},
		Count:            13,
		TemporaryReserve: 2,
	}
}

func jitArchFreeGPRegs() uint64 {
	// Go's closure register X26 and toolchain temporary X27 may hold short
	// lived values, but X27 is excluded from persistent register homes above.
	return (1 << uint(RegRCX)) | (1 << uint(RegRDX)) | (1 << uint(RegRSI)) |
		(1 << uint(RegRDI)) | (1 << uint(RegR8)) | (1 << uint(RegR9)) |
		(1 << uint(RegR10)) | (1 << uint(RegR13)) | (1 << uint(RegR15))
}

func jitArchEmitShift64(ctx *JITContext, dst, count Reg, left bool) {
	op := jitNativeShiftRightReg
	if left {
		op = jitNativeShiftLeftReg
	}
	jitArchEmitNative(ctx, op, dst, count, 0, 0)
}

func arm64Cond(cc JITCondition) uint32 {
	switch cc {
	case CcE:
		return 0
	case CcNE:
		return 1
	case CcB:
		return 3
	case CcAE:
		return 2
	case CcBE:
		return 9
	case CcA:
		return 8
	case CcL:
		return 11
	case CcGE:
		return 10
	case CcLE:
		return 13
	case CcG:
		return 12
	case CcP:
		return 6
	case CcNP:
		return 7
	}
	panic("jit: invalid arm64 condition")
}
func arm64Cset(ctx *JITContext, dst Reg, condition uint32) {
	emitARM64(ctx, 0x9A9F07E0|((condition^1)<<12)|uint32(dst))
}
func arm64SetCondition(ctx *JITContext, dst Reg, cc JITCondition) {
	if !ctx.NativeFloatCompare {
		arm64Cset(ctx, dst, arm64Cond(cc))
		return
	}
	// Dedicated registers retain ZF/CF/PF equivalents across integer
	// materialization instructions between a floating comparison and its uses.
	invert := func() {
		temp := arm64TempExcept(dst)
		jitArchEmitMovRegImm64(ctx, temp, 1)
		jitArchEmitXorInt64(ctx, dst, temp)
	}
	switch cc {
	case CcE:
		jitArchEmitMovRegReg(ctx, dst, arm64FloatEqual)
	case CcNE:
		jitArchEmitMovRegReg(ctx, dst, arm64FloatEqual)
		invert()
	case CcB:
		jitArchEmitMovRegReg(ctx, dst, arm64FloatBelow)
	case CcAE:
		jitArchEmitMovRegReg(ctx, dst, arm64FloatBelow)
		invert()
	case CcBE:
		jitArchEmitMovRegReg(ctx, dst, arm64FloatEqual)
		jitArchEmitOrInt64(ctx, dst, arm64FloatBelow)
	case CcA:
		jitArchEmitMovRegReg(ctx, dst, arm64FloatEqual)
		jitArchEmitOrInt64(ctx, dst, arm64FloatBelow)
		invert()
	case CcP:
		jitArchEmitMovRegReg(ctx, dst, arm64FloatUnordered)
	case CcNP:
		jitArchEmitMovRegReg(ctx, dst, arm64FloatUnordered)
		invert()
	default:
		panic("jit: invalid arm64 float condition")
	}
}
func jitArchEmitBranch(ctx *JITContext, cc JITCondition, label JITLabel, conditional bool) {
	ctx.FlushRegisterMoves()
	if conditional {
		if ctx.NativeFloatCompare {
			arm64SetCondition(ctx, 17, cc)
			emitARM64(ctx, 0xB4000051)
		} else {
			emitARM64(ctx, 0x54000000|2<<5|(arm64Cond(cc)^1))
		}
	}
	ctx.AddFixup(label, 4, true)
	jitArchEmitBranchPlaceholder(ctx)
}
func jitArchEmitBranchPlaceholder(ctx *JITContext) { emitARM64(ctx, 0x14000000) }
func jitArchPatchBranch(pos unsafe.Pointer, disp int32) {
	if disp%4 != 0 || disp < -(1<<27) || disp >= 1<<27 {
		panic("jit: arm64 branch out of range")
	}
	*(*uint32)(pos) = 0x14000000 | (uint32(disp>>2) & 0x3ffffff)
}
func jitArchEmitImmediateFixup(ctx *JITContext, dst Reg) unsafe.Pointer {
	ctx.FlushRegisterMoves()
	pos := ctx.Ptr
	for shift := uint32(0); shift < 4; shift++ {
		opcode := uint32(0xF2800000)
		if shift == 0 {
			opcode = 0xD2800000
		}
		emitARM64(ctx, opcode|shift<<21|uint32(dst))
	}
	return pos
}
func jitArchPatchImmediate(pos unsafe.Pointer, value int32) {
	imm := uint64(int64(value))
	dst := *(*uint32)(pos) & 31
	for shift := uint32(0); shift < 4; shift++ {
		opcode := uint32(0xF2800000)
		if shift == 0 {
			opcode = 0xD2800000
		}
		*(*uint32)(unsafe.Add(pos, shift*4)) = opcode | shift<<21 | uint32(imm>>(16*shift)&0xffff)<<5 | dst
	}
}
func jitArchEmitFrameFixup(ctx *JITContext) unsafe.Pointer {
	pos := jitArchEmitImmediateFixup(ctx, 17)
	emitARM64(ctx, 0xCB3163FF) // SUB SP, SP, X17 (extended register).
	return pos
}
func arm64Address(ctx *JITContext, dst, base Reg, disp int32) {
	if disp == 0 {
		jitArchEmitMovRegReg(ctx, dst, base)
		return
	}
	if disp >= -4095 && disp <= 4095 {
		opcode := uint32(0x91000000)
		magnitude := disp
		if disp < 0 {
			opcode = 0xD1000000
			magnitude = -disp
		}
		emitARM64(ctx, opcode|uint32(magnitude)<<10|uint32(base)<<5|uint32(dst))
		return
	}
	temp := dst
	if dst == base || dst == RegRSP {
		temp = arm64TempExcept(base)
	}

	jitArchEmitMovRegImm64(ctx, temp, uint64(int64(disp)))
	if base == RegRSP || dst == RegRSP {
		emitARM64(ctx, 0x8B206000|uint32(temp)<<16|uint32(base)<<5|uint32(dst))
	} else {
		emitARM64(ctx, 0x8B000000|uint32(temp)<<16|uint32(base)<<5|uint32(dst))
	}
}
func jitArchEmitNative(ctx *JITContext, op jitNativeOp, a, b, c Reg, imm int64) {
	rd, rn, rm := uint32(a), uint32(b), uint32(c)
	switch op {
	case jitNativeReturn:
		jitArchEmitReturn(ctx)
	case jitNativeProlog:
		jitArchEmitLeafProlog(ctx)
	case jitNativeLeave, jitNativeEpilog:
		jitArchEmitMovRegReg(ctx, RegRSP, RegRBP)
		emitARM64(ctx, 0xA8C17BFD)
		if op == jitNativeEpilog {
			jitArchEmitReturn(ctx)
		}
	case jitNativeStackAdjust:
		arm64Address(ctx, RegRSP, RegRSP, int32(imm))
	case jitNativeCall:
		emitARM64(ctx, 0xD63F0000|rd<<5)
	case jitNativeMoreStackCall:
		jitArchEmitLoad64(ctx, 3, RegRBP, 8)
		emitARM64(ctx, 0xD63F0000|rd<<5)
	case jitNativeAddress:
		arm64Address(ctx, a, b, int32(imm))
	case jitNativeAtomicLoad64, jitNativeAtomicStore64:
		// Match internal/runtime/atomic's acquire load and release store. These
		// instructions have no displacement field, so form the complete address
		// in a register distinct from the value and the original base.
		address := arm64TempExcept(a, b)
		arm64Address(ctx, address, b, int32(imm))
		opcode := uint32(0xC8DFFC00) // LDAR Xt, [Xn].
		if op == jitNativeAtomicStore64 {
			opcode = 0xC89FFC00 // STLR Xt, [Xn].
		}
		emitARM64(ctx, opcode|uint32(address)<<5|rd)
	case jitNativeIndexAddress, jitNativeIndexLoad:
		shift := uint32(0)
		for factor := imm & 255; factor > 1; factor >>= 1 {
			shift++
		}
		dest := a
		if op == jitNativeIndexLoad {
			dest = 17
		}
		if b == RegRSP || dest == RegRSP {
			emitARM64(ctx, 0x8B206000|rm<<16|shift<<10|rn<<5|uint32(dest))
		} else {
			emitARM64(ctx, 0x8B000000|rm<<16|shift<<10|rn<<5|uint32(dest))
		}
		if op == jitNativeIndexLoad {
			switch imm >> 8 {
			case 1:
				jitArchEmitLoad8(ctx, a, dest, 0)
			case 2:
				jitArchEmitLoad16(ctx, a, dest, 0)
			case 4:
				jitArchEmitLoad32(ctx, a, dest, 0)
			case 8:
				jitArchEmitLoad64(ctx, a, dest, 0)
			default:
				panic("jit: invalid indexed width")
			}
		}
	case jitNativeCmpImm, jitNativeCmpByte:
		left := a
		if op == jitNativeCmpByte {
			left = arm64TempExcept(a)
			emitARM64(ctx, 0xD3401C00|rd<<5|uint32(left))
		}
		temp := arm64TempExcept(left)
		jitArchEmitMovRegImm64(ctx, temp, uint64(imm))
		ctx.NativeFloatCompare = false
		emitARM64(ctx, 0xEB00001F|uint32(temp)<<16|uint32(left)<<5)
	case jitNativeALU:
		if imm == 0x39 {
			ctx.NativeFloatCompare = false
			emitARM64(ctx, 0xEB00001F|rn<<16|rd<<5)
			return
		}
		opcode := map[int64]uint32{0x01: 0x8B000000, 0x29: 0xCB000000, 0x09: 0xAA000000, 0x21: 0x8A000000, 0x31: 0xCA000000}[imm]
		if opcode == 0 {
			panic("jit: invalid arm64 ALU")
		}
		emitARM64(ctx, opcode|rn<<16|rd<<5|rd)
	case jitNativeSetCondition:
		arm64SetCondition(ctx, a, JITCondition(imm))
	case jitNativeZeroExtend32:
		emitARM64(ctx, 0x2A0003E0|rd<<16|rd)
	case jitNativeShiftLeft:
		n := uint32(imm) & 63
		emitARM64(ctx, 0xD3400000|((64-n)&63)<<16|(63-n)<<10|rd<<5|rd)
	case jitNativeShiftRight:
		emitARM64(ctx, 0xD340FC00|(uint32(imm)&63)<<16|rd<<5|rd)
	case jitNativeShiftSigned:
		emitARM64(ctx, 0x9340FC00|(uint32(imm)&63)<<16|rd<<5|rd)
	case jitNativeShiftLeftReg:
		emitARM64(ctx, 0x9AC02000|rn<<16|rd<<5|rd)
	case jitNativeShiftRightReg:
		emitARM64(ctx, 0x9AC02400|rn<<16|rd<<5|rd)
	case jitNativeAddImm, jitNativeSubImm, jitNativeAddImm32, jitNativeSubImm32, jitNativeAndImm, jitNativeOrImm, jitNativeMulImm, jitNativeDivImm, jitNativeRemImm:
		temp := arm64TempExcept(a)
		jitArchEmitMovRegImm64(ctx, temp, uint64(imm))
		opcode := uint32(0)
		switch op {
		case jitNativeAddImm:
			opcode = 0x8B000000
		case jitNativeSubImm:
			opcode = 0xCB000000
		case jitNativeAddImm32:
			opcode = 0x0B000000
		case jitNativeSubImm32:
			opcode = 0x4B000000
		case jitNativeAndImm:
			opcode = 0x8A000000
		case jitNativeOrImm:
			opcode = 0xAA000000
		case jitNativeMulImm:
			opcode = 0x9B007C00
		case jitNativeDivImm, jitNativeRemImm:
			opcode = 0x9AC00C00
		}
		if (op == jitNativeDivImm || op == jitNativeRemImm) && imm == 0 {
			panic("jit: integer division by zero")
		}
		if op == jitNativeRemImm {
			quotient := arm64TempExcept(a, temp)
			emitARM64(ctx, opcode|uint32(temp)<<16|rd<<5|uint32(quotient))
			emitARM64(ctx, 0x9B008000|uint32(temp)<<16|rd<<10|uint32(quotient)<<5|rd)
		} else {
			emitARM64(ctx, opcode|uint32(temp)<<16|rd<<5|rd)
		}
	case jitNativeGPRToF:
		emitARM64(ctx, 0x9E670000|rn<<5|(rd-32))
	case jitNativeFToGPR:
		emitARM64(ctx, 0x9E660000|(rn-32)<<5|rd)
	case jitNativeFZero:
		emitARM64(ctx, 0x9E670000|31<<5|(rd-32))
	case jitNativeFLoad:
		temp := arm64TempExcept(b)
		jitArchEmitLoad64(ctx, temp, b, int32(imm))
		emitARM64(ctx, 0x9E670000|uint32(temp)<<5|(rd-32))
	case jitNativeFStore:
		temp := arm64TempExcept(b)
		emitARM64(ctx, 0x9E660000|(rd-32)<<5|uint32(temp))
		jitArchEmitStore64(ctx, temp, b, int32(imm))
	case jitNativeFloatOp:
		opcode := map[int64]uint32{0x10: 0x1E604000, 0x58: 0x1E602800, 0x5C: 0x1E603800, 0x59: 0x1E600800, 0x5E: 0x1E601800}[imm]
		if opcode == 0 {
			panic("jit: invalid arm64 FP operation")
		}
		if imm == 0x10 {
			emitARM64(ctx, opcode|(rn-32)<<5|(rd-32))
		} else {
			emitARM64(ctx, opcode|(rn-32)<<16|(rd-32)<<5|(rd-32))
		}
	case jitNativeIntToFloat:
		emitARM64(ctx, 0x9E620000|rn<<5|(rd-32))
		emitARM64(ctx, 0x9E660000|(rd-32)<<5|rn)
	case jitNativeFloatToInt:
		emitARM64(ctx, 0x9E670000|rn<<5)
		emitARM64(ctx, 0x9E780000|rd)
	case jitNativeFCompare:
		emitARM64(ctx, 0x1E602000|(rn-32)<<16|(rd-32)<<5)
		arm64Cset(ctx, arm64FloatEqual, 0)
		arm64Cset(ctx, arm64FloatBelow, 4)
		arm64Cset(ctx, arm64FloatUnordered, 6)
		jitArchEmitOrInt64(ctx, arm64FloatEqual, arm64FloatUnordered)
		jitArchEmitOrInt64(ctx, arm64FloatBelow, arm64FloatUnordered)
		ctx.NativeFloatCompare = true
	default:
		panic("jit: unsupported arm64 operation")
	}
}
