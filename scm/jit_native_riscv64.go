//go:build riscv64

/*
Copyright (C) 2026 Carl-Philip Hänsch
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/
package scm

import "unsafe"

const jitNativeTemp Reg = 31
const jitGoStackArgOffset = int32(8)

func jitGoABIIntRegs() []Reg {
	return []Reg{10, 11, 12, 13, 14, 15, 16, 17, 8, 9, 18, 19, 20, 21, 22, 23}
}
func riscvAddress(ctx *JITContext, dst, base Reg, disp int32) {
	if disp >= -2048 && disp <= 2047 {
		emitRISCV64(ctx, riscvI(0x13, 0, dst, base, disp))
		return
	}
	temp := dst
	if dst == base {
		temp = 31
		if base == temp {
			temp = 29
		}
	}

	jitArchEmitMovRegImm64(ctx, temp, uint64(int64(disp)))
	emitRISCV64(ctx, riscvR(0x33, 0, 0, dst, base, temp))
}
func riscvSetCondition(ctx *JITContext, dst Reg, cc JITCondition) {
	// Comparisons capture their operands in X5/X6, outside the allocator. These
	// snapshots survive result moves and the multiple SETcc consumers of NaNs.
	if ctx.NativeFloatCompare {
		switch cc {
		case CcE:
			jitArchEmitMovRegReg(ctx, dst, 5)
		case CcNE:
			emitRISCV64(ctx, riscvI(0x13, 4, dst, 5, 1))
		case CcB:
			jitArchEmitMovRegReg(ctx, dst, 6)
		case CcAE:
			emitRISCV64(ctx, riscvI(0x13, 4, dst, 6, 1))
		case CcBE:
			emitRISCV64(ctx, riscvR(0x33, 6, 0, dst, 5, 6))
		case CcA:
			emitRISCV64(ctx, riscvR(0x33, 6, 0, dst, 5, 6))
			emitRISCV64(ctx, riscvI(0x13, 4, dst, dst, 1))
		case CcP:
			jitArchEmitMovRegReg(ctx, dst, 7)
		case CcNP:
			emitRISCV64(ctx, riscvI(0x13, 4, dst, 7, 1))
		case CcL:
			jitArchEmitZeroReg(ctx, dst)
		case CcGE:
			emitRISCV64(ctx, riscvI(0x13, 0, dst, 0, 1))
		case CcLE:
			jitArchEmitMovRegReg(ctx, dst, 5)
		case CcG:
			emitRISCV64(ctx, riscvI(0x13, 4, dst, 5, 1))
		default:
			panic("jit: invalid riscv64 float condition")
		}
		return
	}
	switch cc {
	case CcE, CcNE:
		emitRISCV64(ctx, riscvR(0x33, 4, 0, dst, 5, 6))
		emitRISCV64(ctx, riscvI(0x13, 3, dst, dst, 1))
		if cc == CcNE {
			emitRISCV64(ctx, riscvI(0x13, 4, dst, dst, 1))
		}
	case CcL, CcGE, CcB, CcAE:
		f := uint32(2)
		if cc == CcB || cc == CcAE {
			f = 3
		}
		emitRISCV64(ctx, riscvR(0x33, f, 0, dst, 5, 6))
		if cc == CcGE || cc == CcAE {
			emitRISCV64(ctx, riscvI(0x13, 4, dst, dst, 1))
		}
	case CcLE, CcG, CcBE, CcA:
		f := uint32(2)
		if cc == CcBE || cc == CcA {
			f = 3
		}
		emitRISCV64(ctx, riscvR(0x33, f, 0, dst, 6, 5))
		if cc == CcLE || cc == CcBE {
			emitRISCV64(ctx, riscvI(0x13, 4, dst, dst, 1))
		}
	case CcP:
		jitArchEmitZeroReg(ctx, dst)
	case CcNP:
		emitRISCV64(ctx, riscvI(0x13, 0, dst, 0, 1))
	default:
		panic("jit: invalid riscv64 condition")
	}
}
func jitArchEmitBranch(ctx *JITContext, cc JITCondition, label JITLabel, conditional bool) {
	ctx.FlushRegisterMoves()
	if conditional {
		// The stack guard check keeps its frame-size operand in X31 on the
		// fallthrough path. Evaluate the condition in the other reserved temp.
		riscvSetCondition(ctx, 29, cc)
		emitRISCV64(ctx, 0x000E8663)
	} // BEQ X29, ZERO, +12
	ctx.AddFixup(label, 8, true)
	jitArchEmitBranchPlaceholder(ctx)
}
func jitArchEmitBranchPlaceholder(ctx *JITContext) {
	emitRISCV64(ctx, 0x00000F97)
	emitRISCV64(ctx, 0x000F8067)
}
func jitArchPatchBranch(pos unsafe.Pointer, disp int32) {
	if disp%4 != 0 {
		panic("jit: unaligned riscv64 branch")
	}
	high := (int64(disp) + 0x800) >> 12
	low := int64(disp) - (high << 12)
	*(*uint32)(pos) = uint32(high)<<12 | 0xF97
	*(*uint32)(unsafe.Add(pos, 4)) = riscvI(0x67, 0, 0, 31, int32(low))
}
func jitArchEmitImmediateFixup(ctx *JITContext, dst Reg) unsafe.Pointer {
	ctx.FlushRegisterMoves()
	pos := ctx.Ptr
	jitArchEmitMovRegImm64(ctx, dst, 0)
	return pos
}
func jitArchPatchImmediate(pos unsafe.Pointer, value int32) {
	dst := Reg((*(*uint32)(pos) >> 7) & 31)
	ctx := &JITContext{Ptr: pos, End: unsafe.Add(pos, 60)}
	jitArchEmitMovRegImm64(ctx, dst, uint64(int64(value)))
}
func jitArchEmitFrameFixup(ctx *JITContext) unsafe.Pointer {
	pos := jitArchEmitImmediateFixup(ctx, 31)
	emitRISCV64(ctx, riscvR(0x33, 0, 0x20, RegRSP, RegRSP, 31))
	return pos
}
func jitArchEmitNative(ctx *JITContext, op jitNativeOp, a, b, c Reg, imm int64) {
	switch op {
	case jitNativeReturn:
		jitArchEmitReturn(ctx)
	case jitNativeProlog:
		emitRISCV64(ctx, riscvI(0x13, 0, RegRSP, RegRSP, -16))
		jitArchEmitStore64(ctx, RegRBP, RegRSP, 0)
		jitArchEmitStore64(ctx, 1, RegRSP, 8)
		jitArchEmitMovRegReg(ctx, RegRBP, RegRSP)
	case jitNativeLeave, jitNativeEpilog:
		jitArchEmitMovRegReg(ctx, RegRSP, RegRBP)
		jitArchEmitLoad64(ctx, RegRBP, RegRSP, 0)
		jitArchEmitLoad64(ctx, 1, RegRSP, 8)
		emitRISCV64(ctx, riscvI(0x13, 0, RegRSP, RegRSP, 16))
		if op == jitNativeEpilog {
			jitArchEmitReturn(ctx)
		}
	case jitNativeStackAdjust:
		riscvAddress(ctx, RegRSP, RegRSP, int32(imm))
	case jitNativeCall:
		emitRISCV64(ctx, riscvI(0x67, 0, 1, a, 0))
	case jitNativeMoreStackCall:
		jitArchEmitLoad64(ctx, 1, RegRBP, 8)
		emitRISCV64(ctx, riscvI(0x67, 0, 5, a, 0)) // morestack returns through T0, preserving RA.
	case jitNativeAddress:
		riscvAddress(ctx, a, b, int32(imm))
	case jitNativeAtomicLoad64:
		address := b
		if imm != 0 {
			address = 29
			if b == address {
				address = 31
			}
			riscvAddress(ctx, address, b, int32(imm))
		}
		// Match internal/runtime/atomic.Load64: LR.D with acquire ordering.
		emitRISCV64(ctx, 0x1400302F|riscv64GPR(address)<<15|riscv64GPR(a)<<7)
	case jitNativeAtomicStore64:
		riscvEmitAtomicStore64(ctx, a, b, int32(imm))
	case jitNativeIndexAddress, jitNativeIndexLoad:
		var shift int32
		switch imm & 255 {
		case 1:
		case 2:
			shift = 1
		case 4:
			shift = 2
		case 8:
			shift = 3
		default:
			panic("jit: riscv64 indexed address scale must be 1, 2, 4, or 8")
		}
		// Preserve a base that already occupies the normal address temporary.
		address := Reg(31)
		if b == address {
			address = 29
		}
		emitRISCV64(ctx, riscvI(0x13, 1, address, c, shift))
		emitRISCV64(ctx, riscvR(0x33, 0, 0, address, b, address))
		if op == jitNativeIndexAddress {
			jitArchEmitMovRegReg(ctx, a, address)
		} else {
			switch imm >> 8 {
			case 1:
				jitArchEmitLoad8(ctx, a, address, 0)
			case 2:
				jitArchEmitLoad16(ctx, a, address, 0)
			case 4:
				jitArchEmitLoad32(ctx, a, address, 0)
			case 8:
				jitArchEmitLoad64(ctx, a, address, 0)
			default:
				panic("jit: invalid indexed width")
			}
		}
	case jitNativeCmpImm, jitNativeCmpByte:
		jitArchEmitMovRegReg(ctx, 5, a)
		if op == jitNativeCmpByte {
			emitRISCV64(ctx, riscvI(0x13, 7, 5, 5, 255))
		}
		jitArchEmitMovRegImm64(ctx, 6, uint64(imm))
		ctx.NativeFloatCompare = false
	case jitNativeALU:
		if imm == 0x39 {
			jitArchEmitMovRegReg(ctx, 5, a)
			jitArchEmitMovRegReg(ctx, 6, b)
			ctx.NativeFloatCompare = false
			return
		}
		f3, f7 := uint32(0), uint32(0)
		switch imm {
		case 0x01:
		case 0x29:
			f7 = 0x20
		case 0x09:
			f3 = 6
		case 0x21:
			f3 = 7
		case 0x31:
			f3 = 4
		default:
			panic("jit: invalid riscv64 ALU")
		}
		emitRISCV64(ctx, riscvR(0x33, f3, f7, a, a, b))
	case jitNativeSetCondition:
		riscvSetCondition(ctx, a, JITCondition(imm))
	case jitNativeZeroExtend32:
		riscvZeroExtendWord(ctx, a)
	case jitNativeShiftLeft:
		emitRISCV64(ctx, riscvI(0x13, 1, a, a, int32(imm&63)))
	case jitNativeShiftRight:
		emitRISCV64(ctx, riscvI(0x13, 5, a, a, int32(imm&63)))
	case jitNativeShiftSigned:
		emitRISCV64(ctx, riscvI(0x13, 5, a, a, int32(imm&63)|0x400))
	case jitNativeShiftLeftReg:
		emitRISCV64(ctx, riscvR(0x33, 1, 0, a, a, b))
	case jitNativeShiftRightReg:
		emitRISCV64(ctx, riscvR(0x33, 5, 0, a, a, b))
	case jitNativeAddImm, jitNativeSubImm, jitNativeAddImm32, jitNativeSubImm32, jitNativeAndImm, jitNativeOrImm, jitNativeMulImm, jitNativeDivImm, jitNativeRemImm:
		temp := Reg(31)
		if a == temp {
			temp = 29
		}
		jitArchEmitMovRegImm64(ctx, temp, uint64(imm))
		f3, f7 := uint32(0), uint32(0)
		switch op {
		case jitNativeSubImm, jitNativeSubImm32:
			f7 = 0x20
		case jitNativeAndImm:
			f3 = 7
		case jitNativeOrImm:
			f3 = 6
		case jitNativeMulImm:
			f7 = 1
		case jitNativeDivImm:
			f3 = 4
			f7 = 1
		case jitNativeRemImm:
			f3 = 6
			f7 = 1
		}
		if (op == jitNativeDivImm || op == jitNativeRemImm) && imm == 0 {
			panic("jit: integer division by zero")
		}
		emitRISCV64(ctx, riscvR(0x33, f3, f7, a, a, temp))
		if op == jitNativeAddImm32 || op == jitNativeSubImm32 {
			riscvZeroExtendWord(ctx, a)
		}
	case jitNativeGPRToF:
		emitRISCV64(ctx, 0xF2000053|uint32(b)<<15|uint32(a-32)<<7)
	case jitNativeFToGPR:
		emitRISCV64(ctx, 0xE2000053|uint32(b-32)<<15|uint32(a)<<7)
	case jitNativeFZero:
		emitRISCV64(ctx, 0xF2000053|uint32(a-32)<<7)
	case jitNativeFLoad:
		jitArchEmitLoad64(ctx, 31, b, int32(imm))
		emitRISCV64(ctx, 0xF2000053|31<<15|uint32(a-32)<<7)
	case jitNativeFStore:
		emitRISCV64(ctx, 0xE2000053|uint32(a-32)<<15|31<<7)
		jitArchEmitStore64(ctx, 31, b, int32(imm))
	case jitNativeFloatOp:
		fa, fb := uint32(a-32), uint32(b-32)
		if imm == 0x10 {
			emitRISCV64(ctx, 0x22000053|fb<<20|fb<<15|fa<<7)
			return
		}
		f7 := uint32(0)
		switch imm {
		case 0x58:
			f7 = 1
		case 0x5C:
			f7 = 5
		case 0x59:
			f7 = 9
		case 0x5E:
			f7 = 13
		default:
			panic("jit: invalid riscv64 FP operation")
		}
		// Go does not initialize FCSR. Encode round-to-nearest, ties-to-even
		// explicitly instead of depending on the caller's dynamic rounding mode.
		emitRISCV64(ctx, f7<<25|fb<<20|fa<<15|fa<<7|0x53)
	case jitNativeIntToFloat:
		emitRISCV64(ctx, 0xD2200053|uint32(b)<<15|uint32(a-32)<<7)
		emitRISCV64(ctx, 0xE2000053|uint32(a-32)<<15|uint32(b)<<7)
	case jitNativeFloatToInt:
		emitRISCV64(ctx, 0xF2000053|uint32(b)<<15)
		emitRISCV64(ctx, 0xC2201053|uint32(a)<<7)
	case jitNativeFCompare:
		fa, fb := uint32(a-32), uint32(b-32)
		emitRISCV64(ctx, 0xA2002053|fa<<20|fa<<15|5<<7) // FEQ.D left,left
		emitRISCV64(ctx, 0xA2002053|fb<<20|fb<<15|6<<7)
		emitRISCV64(ctx, riscvR(0x33, 7, 0, 7, 5, 6))
		emitRISCV64(ctx, riscvI(0x13, 4, 7, 7, 1)) // unordered
		emitRISCV64(ctx, 0xA2002053|fb<<20|fa<<15|5<<7)
		emitRISCV64(ctx, 0xA2001053|fb<<20|fa<<15|6<<7) // FLT.D
		jitArchEmitOrInt64(ctx, 5, 7)
		jitArchEmitOrInt64(ctx, 6, 7)
		ctx.NativeFloatCompare = true
	default:
		panic("jit: unsupported riscv64 operation")
	}
}

// Go stores use AMOSWAP.D.AQRL, with the discarded previous value in ZERO.
func riscvEmitAtomicStore64(ctx *JITContext, src, base Reg, disp int32) {
	if disp != 0 && (base == 29 && src == 31 || base == 31 && src == 29) {
		riscvAdjustAddress(ctx, base, int64(disp))
		riscvEmitAtomicStore64(ctx, src, base, 0)
		riscvAdjustAddress(ctx, base, -int64(disp))
		return
	}
	address := base
	if disp != 0 {
		address = 29
		if base == address || src == address {
			address = 31
		}
		riscvAddress(ctx, address, base, disp)
	}
	emitRISCV64(ctx, 0x0E00302F|riscv64GPR(src)<<20|riscv64GPR(address)<<15)
}

// The common emitter checks Go's out-of-range shift semantics. RISC-V accepts
// any GPR as its count operand, so no fixed-register shuffle is necessary.
func jitArchEmitShift64(ctx *JITContext, dst, count Reg, left bool) {
	op := jitNativeShiftRightReg
	if left {
		op = jitNativeShiftLeftReg
	}
	jitArchEmitNative(ctx, op, dst, count, 0, 0)
}
