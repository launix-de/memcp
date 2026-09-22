//go:build riscv64

/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.
*/

package scm

import "syscall"

func jitArchEmitReturn(ctx *JITContext) { emitRISCV64(ctx, 0x00008067) }

func jitArchEmitLeafProlog(ctx *JITContext) {
	emitRISCV64(ctx, 0xFF010113) // ADDI SP, SP, -16
	emitRISCV64(ctx, 0x00113423) // SD RA, 8(SP)
	emitRISCV64(ctx, 0x00813023) // SD S0, 0(SP)
	emitRISCV64(ctx, 0x01010413) // ADDI S0, SP, 16
}

func jitArchEmitLeafEpilog(ctx *JITContext) {
	emitRISCV64(ctx, 0x00013403) // LD S0, 0(SP)
	emitRISCV64(ctx, 0x00813083) // LD RA, 8(SP)
	emitRISCV64(ctx, 0x01010113) // ADDI SP, SP, 16
	jitArchEmitReturn(ctx)
}

func jitFlushInstructionCache(start, end uintptr) {
	const sysRISCVFlushICache = 259
	_, _, errno := syscall.Syscall(sysRISCVFlushICache, start, end, 0)
	if errno != 0 {
		panic("jit: riscv64 instruction-cache flush failed: " + errno.Error())
	}
}

func riscv64GPR(reg Reg) uint32 {
	if reg > 31 {
		panic("jit: riscv64 operation requires a general-purpose register")
	}
	return uint32(reg)
}

func emitRISCV64(ctx *JITContext, instruction uint32) { ctx.emitU32(instruction) }

func riscvR(opcode, funct3, funct7 uint32, dst, left, right Reg) uint32 {
	return funct7<<25 | riscv64GPR(right)<<20 | riscv64GPR(left)<<15 |
		funct3<<12 | riscv64GPR(dst)<<7 | opcode
}

func riscvI(opcode, funct3 uint32, dst, src Reg, imm int32) uint32 {
	if imm < -2048 || imm > 2047 {
		panic("jit: riscv64 immediate is out of range")
	}
	return (uint32(imm)&0xfff)<<20 | riscv64GPR(src)<<15 | funct3<<12 |
		riscv64GPR(dst)<<7 | opcode
}

func jitArchEmitMovRegReg(ctx *JITContext, dst, src Reg) {
	emitRISCV64(ctx, riscvI(0x13, 0, dst, src, 0)) // ADDI
}

func jitArchEmitMovRegImm64(ctx *JITContext, dst Reg, imm uint64) {
	// Byte-at-a-time construction is intentionally simple and fully general;
	// later instruction selection may replace it with LUI/ADDI compression.
	emitRISCV64(ctx, riscvI(0x13, 0, dst, 0, int32(imm>>56)))
	for shift := 48; shift >= 0; shift -= 8 {
		emitRISCV64(ctx, riscvI(0x13, 1, dst, dst, 8))                        // SLLI
		emitRISCV64(ctx, riscvI(0x13, 6, dst, dst, int32((imm>>shift)&0xff))) // ORI
	}
}

func jitArchEmitLoad64(ctx *JITContext, dst, base Reg, disp int32) {
	emitRISCV64(ctx, riscvI(0x03, 3, dst, base, disp)) // LD
}

func jitArchEmitStore64(ctx *JITContext, src, base Reg, disp int32) {
	jitRISCV64Store(ctx, src, base, disp, 3)
}

func jitRISCV64Store(ctx *JITContext, src, base Reg, disp int32, funct3 uint32) {
	if disp < -2048 || disp > 2047 {
		panic("jit: riscv64 store displacement is out of range")
	}
	uimm := uint32(disp) & 0xfff
	emitRISCV64(ctx, uimm>>5<<25|riscv64GPR(src)<<20|riscv64GPR(base)<<15|
		funct3<<12|(uimm&0x1f)<<7|0x23)
}

func jitArchEmitLoad8(ctx *JITContext, dst, base Reg, disp int32) {
	emitRISCV64(ctx, riscvI(0x03, 4, dst, base, disp)) // LBU
}

func jitArchEmitLoad16(ctx *JITContext, dst, base Reg, disp int32) {
	emitRISCV64(ctx, riscvI(0x03, 5, dst, base, disp)) // LHU
}

func jitArchEmitLoad32(ctx *JITContext, dst, base Reg, disp int32) {
	emitRISCV64(ctx, riscvI(0x03, 6, dst, base, disp)) // LWU
}

func jitArchEmitStore8(ctx *JITContext, src, base Reg, disp int32) {
	jitRISCV64Store(ctx, src, base, disp, 0) // SB
}

func jitArchEmitStore16(ctx *JITContext, src, base Reg, disp int32) {
	jitRISCV64Store(ctx, src, base, disp, 1) // SH
}

func jitArchEmitStore32(ctx *JITContext, src, base Reg, disp int32) {
	jitRISCV64Store(ctx, src, base, disp, 2) // SW
}

func jitArchEmitZeroReg(ctx *JITContext, reg Reg) {
	emitRISCV64(ctx, riscvI(0x13, 0, reg, 0, 0)) // ADDI rd, zero, 0
}

func jitArchEmitOrInt64(ctx *JITContext, dst, src Reg) {
	emitRISCV64(ctx, riscvR(0x33, 6, 0, dst, dst, src))
}

func jitArchEmitAndInt64(ctx *JITContext, dst, src Reg) {
	emitRISCV64(ctx, riscvR(0x33, 7, 0, dst, dst, src))
}

func jitArchEmitXorInt64(ctx *JITContext, dst, src Reg) {
	emitRISCV64(ctx, riscvR(0x33, 4, 0, dst, dst, src))
}

func jitArchEmitAddInt64(ctx *JITContext, dst, src Reg) {
	emitRISCV64(ctx, riscvR(0x33, 0, 0, dst, dst, src))
}

func jitArchEmitSubInt64(ctx *JITContext, dst, src Reg) {
	emitRISCV64(ctx, riscvR(0x33, 0, 0x20, dst, dst, src))
}

func riscvZeroExtendWord(ctx *JITContext, reg Reg) {
	emitRISCV64(ctx, riscvI(0x13, 1, reg, reg, 32)) // SLLI
	emitRISCV64(ctx, riscvI(0x13, 5, reg, reg, 32)) // SRLI
}

func jitArchEmitAddInt32(ctx *JITContext, dst, src Reg) {
	jitArchEmitAddInt64(ctx, dst, src)
	riscvZeroExtendWord(ctx, dst)
}

func jitArchEmitSubInt32(ctx *JITContext, dst, src Reg) {
	jitArchEmitSubInt64(ctx, dst, src)
	riscvZeroExtendWord(ctx, dst)
}

func jitArchEmitMulInt64(ctx *JITContext, dst, src Reg) {
	emitRISCV64(ctx, riscvR(0x33, 0, 1, dst, dst, src))
}

func jitArchEmitIntBinary(ctx *JITContext, op JITIntOp, width uint8, dst Reg, right jitIntOperand, scratch Reg) {
	switch right.kind {
	case jitIntOperandReg:
		ctx.beginRegisterInstruction(jitRegisterMask(dst, right.reg), jitRegisterMask(dst))
		defer ctx.endRegisterInstruction()
		riscv64EmitIntBinaryReg(ctx, op, width, dst, right.reg)
	case jitIntOperandMem:
		ctx.beginRegisterInstruction(jitRegisterMask(dst, right.base), jitRegisterMask(dst, scratch))
		defer ctx.endRegisterInstruction()
		jitArchEmitLoad64(ctx, scratch, right.base, right.disp)
		riscv64EmitIntBinaryReg(ctx, op, width, dst, scratch)
	case jitIntOperandImm:
		if riscv64IntBinaryImmEncodable(op, right.imm) {
			ctx.beginRegisterInstruction(jitRegisterMask(dst), jitRegisterMask(dst))
			defer ctx.endRegisterInstruction()
			riscv64EmitIntBinaryImm(ctx, op, width, dst, right.imm)
			return
		}
		if scratch == dst {
			panic("jit: riscv64 integer immediate requires a distinct scratch register")
		}
		ctx.beginRegisterInstruction(jitRegisterMask(dst), jitRegisterMask(dst, scratch))
		defer ctx.endRegisterInstruction()
		jitArchEmitMovRegImm64(ctx, scratch, uint64(right.imm))
		riscv64EmitIntBinaryReg(ctx, op, width, dst, scratch)
	default:
		panic("jit: invalid riscv64 integer operand")
	}
}

func riscv64IntBinaryImmEncodable(op JITIntOp, imm int64) bool {
	if op == JITIntSub {
		return imm >= -2047 && imm <= 2048
	}
	if op != JITIntAdd && op != JITIntAnd && op != JITIntOr && op != JITIntXor {
		return false
	}
	return imm >= -2048 && imm <= 2047
}

func riscv64EmitIntBinaryReg(ctx *JITContext, op JITIntOp, width uint8, dst, src Reg) {
	if width == 32 {
		switch op {
		case JITIntAdd:
			jitArchEmitAddInt32(ctx, dst, src)
		case JITIntSub:
			jitArchEmitSubInt32(ctx, dst, src)
		default:
			panic("jit: riscv64 32-bit operation is not implemented")
		}
		return
	}
	switch op {
	case JITIntAdd:
		jitArchEmitAddInt64(ctx, dst, src)
	case JITIntSub:
		jitArchEmitSubInt64(ctx, dst, src)
	case JITIntMul:
		jitArchEmitMulInt64(ctx, dst, src)
	case JITIntAnd:
		jitArchEmitAndInt64(ctx, dst, src)
	case JITIntOr:
		jitArchEmitOrInt64(ctx, dst, src)
	case JITIntXor:
		jitArchEmitXorInt64(ctx, dst, src)
	default:
		panic("jit: invalid riscv64 integer operation")
	}
}

func riscv64EmitIntBinaryImm(ctx *JITContext, op JITIntOp, width uint8, dst Reg, imm int64) bool {
	encoded := imm
	funct3 := uint32(0)
	switch op {
	case JITIntAdd:
	case JITIntSub:
		if imm < -2047 || imm > 2048 {
			return false
		}
		encoded = -imm
	case JITIntAnd:
		funct3 = 7
	case JITIntOr:
		funct3 = 6
	case JITIntXor:
		funct3 = 4
	default:
		return false
	}
	if encoded < -2048 || encoded > 2047 {
		return false
	}
	emitRISCV64(ctx, riscvI(0x13, funct3, dst, dst, int32(encoded)))
	if width == 32 {
		riscvZeroExtendWord(ctx, dst)
	}
	return true
}
