//go:build amd64

/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.
*/

package scm

func jitArchEmitReturn(ctx *JITContext) { ctx.emitByte(0xC3) }

func jitArchEmitLeafProlog(*JITContext)     {}
func jitArchEmitLeafEpilog(ctx *JITContext) { jitArchEmitReturn(ctx) }

func jitFlushInstructionCache(_, _ uintptr) {}

func jitArchEmitMovRegReg(ctx *JITContext, dst, src Reg) {
	rex := byte(0x48)
	if src >= 8 {
		rex |= 0x04
	}
	if dst >= 8 {
		rex |= 0x01
	}
	ctx.emitBytes(rex, 0x89, 0xC0|(byte(src&7)<<3)|byte(dst&7))
}

func jitArchEmitMovRegImm64(ctx *JITContext, dst Reg, imm uint64) {
	dstEnc := byte(dst & 7)
	if imm == 0 {
		if dst >= 8 {
			ctx.emitByte(0x45)
		}
		ctx.emitBytes(0x31, 0xC0|(dstEnc<<3)|dstEnc)
		return
	}
	if imm <= 0xffffffff {
		if dst >= 8 {
			ctx.emitByte(0x41)
		}
		ctx.emitByte(0xB8 | dstEnc)
		ctx.emitU32(uint32(imm))
		return
	}
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01
	}
	ctx.emitBytes(rex, 0xB8|dstEnc)
	ctx.emitU64(imm)
}

func jitArchEmitLoad64(ctx *JITContext, dst, base Reg, disp int32) {
	ctx.emitRegMemOp(0x8B, dst, base, disp)
}

func jitArchEmitStore64(ctx *JITContext, src, base Reg, disp int32) {
	rex := byte(0x48)
	if src >= 8 {
		rex |= 0x04
	}
	if base >= 8 {
		rex |= 0x01
	}
	baseEnc := byte(base & 7)
	srcEnc := byte(src & 7)
	if disp == 0 && baseEnc != 5 {
		modrm := (srcEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(rex, 0x89, modrm, 0x24)
		} else {
			ctx.emitBytes(rex, 0x89, modrm)
		}
		return
	}
	if disp >= -128 && disp <= 127 {
		modrm := byte(0x40) | (srcEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(rex, 0x89, modrm, 0x24, byte(int8(disp)))
		} else {
			ctx.emitBytes(rex, 0x89, modrm, byte(int8(disp)))
		}
		return
	}
	modrm := byte(0x80) | (srcEnc << 3) | baseEnc
	if baseEnc == 4 {
		ctx.emitBytes(rex, 0x89, modrm, 0x24)
	} else {
		ctx.emitBytes(rex, 0x89, modrm)
	}
	ctx.emitU32(uint32(disp))
}

func jitArchEmitLoad8(ctx *JITContext, dst, base Reg, disp int32) {
	ctx.emitRegMemOp2(0x0F, 0xB6, dst, base, disp)
}

func jitArchEmitLoad16(ctx *JITContext, dst, base Reg, disp int32) {
	ctx.emitRegMemOp2(0x0F, 0xB7, dst, base, disp)
}

func jitArchEmitLoad32(ctx *JITContext, dst, base Reg, disp int32) {
	ctx.emitRegMemOp32(0x8B, dst, base, disp)
}

func jitArchEmitStore8(ctx *JITContext, src, base Reg, disp int32) {
	ctx.emitStoreRegMemWidth(src, base, disp, 0x88, false)
}

func jitArchEmitStore16(ctx *JITContext, src, base Reg, disp int32) {
	ctx.emitByte(0x66)
	ctx.emitStoreRegMemWidth(src, base, disp, 0x89, false)
}

func jitArchEmitStore32(ctx *JITContext, src, base Reg, disp int32) {
	ctx.emitStoreRegMemWidth(src, base, disp, 0x89, false)
}

func jitArchEmitZeroReg(ctx *JITContext, reg Reg) {
	if reg >= 8 {
		ctx.emitBytes(0x45, 0x31, byte(0xC0|(byte(reg&7)<<3)|byte(reg&7)))
	} else {
		ctx.emitBytes(0x31, byte(0xC0|(byte(reg)<<3)|byte(reg)))
	}
}

func jitArchEmitOrInt64(ctx *JITContext, dst, src Reg) {
	ctx.emitAluRegReg(0x09, dst, src)
}

func jitArchEmitAndInt64(ctx *JITContext, dst, src Reg) {
	ctx.emitAluRegReg(0x21, dst, src)
}

func jitArchEmitXorInt64(ctx *JITContext, dst, src Reg) {
	ctx.emitAluRegReg(0x31, dst, src)
}

func jitArchEmitAddInt64(ctx *JITContext, dst, src Reg) { ctx.emitAluRegReg(0x01, dst, src) }
func jitArchEmitSubInt64(ctx *JITContext, dst, src Reg) { ctx.emitAluRegReg(0x29, dst, src) }
func jitArchEmitAddInt32(ctx *JITContext, dst, src Reg) {
	ctx.emitAluRegRegWidth(0x01, dst, src, false)
}
func jitArchEmitSubInt32(ctx *JITContext, dst, src Reg) {
	ctx.emitAluRegRegWidth(0x29, dst, src, false)
}
func jitArchEmitMulInt64(ctx *JITContext, dst, src Reg) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x04
	}
	if src >= 8 {
		rex |= 0x01
	}
	ctx.emitBytes(rex, 0x0F, 0xAF, 0xC0|(byte(dst&7)<<3)|byte(src&7))
}

func jitArchEmitIntBinary(ctx *JITContext, op JITIntOp, width uint8, dst Reg, right jitIntOperand, scratch Reg) {
	switch right.kind {
	case jitIntOperandReg:
		ctx.beginRegisterInstruction(jitRegisterMask(dst, right.reg), jitRegisterMask(dst))
		defer ctx.endRegisterInstruction()
		amd64EmitIntBinaryReg(ctx, op, width, dst, right.reg)
	case jitIntOperandMem:
		ctx.beginRegisterInstruction(jitRegisterMask(dst, right.base), jitRegisterMask(dst))
		defer ctx.endRegisterInstruction()
		amd64EmitIntBinaryMem(ctx, op, width, dst, right.base, right.disp)
	case jitIntOperandImm:
		if right.imm >= -1<<31 && right.imm <= 1<<31-1 {
			ctx.beginRegisterInstruction(jitRegisterMask(dst), jitRegisterMask(dst))
			defer ctx.endRegisterInstruction()
			amd64EmitIntBinaryImm(ctx, op, width, dst, int32(right.imm))
			return
		}
		if scratch == dst {
			panic("jit: amd64 large integer immediate requires a distinct scratch register")
		}
		ctx.beginRegisterInstruction(jitRegisterMask(dst), jitRegisterMask(dst, scratch))
		defer ctx.endRegisterInstruction()
		jitArchEmitMovRegImm64(ctx, scratch, uint64(right.imm))
		amd64EmitIntBinaryReg(ctx, op, width, dst, scratch)
	default:
		panic("jit: invalid amd64 integer operand")
	}
}

func amd64EmitIntBinaryReg(ctx *JITContext, op JITIntOp, width uint8, dst, src Reg) {
	if op == JITIntMul {
		if width != 64 {
			panic("jit: amd64 32-bit multiply selection is not implemented")
		}
		jitArchEmitMulInt64(ctx, dst, src)
		return
	}
	var opcode byte
	switch op {
	case JITIntAdd:
		opcode = 0x01
	case JITIntSub:
		opcode = 0x29
	case JITIntAnd:
		opcode = 0x21
	case JITIntOr:
		opcode = 0x09
	case JITIntXor:
		opcode = 0x31
	default:
		panic("jit: invalid amd64 integer operation")
	}
	ctx.emitAluRegRegWidth(opcode, dst, src, width == 64)
}

func amd64EmitIntBinaryMem(ctx *JITContext, op JITIntOp, width uint8, dst, base Reg, disp int32) {
	if op == JITIntMul {
		if width != 64 {
			panic("jit: amd64 32-bit multiply selection is not implemented")
		}
		ctx.emitRegMemOp2(0x0F, 0xAF, dst, base, disp)
		return
	}
	var opcode byte
	switch op {
	case JITIntAdd:
		opcode = 0x03
	case JITIntSub:
		opcode = 0x2B
	case JITIntAnd:
		opcode = 0x23
	case JITIntOr:
		opcode = 0x0B
	case JITIntXor:
		opcode = 0x33
	default:
		panic("jit: invalid amd64 integer operation")
	}
	if width == 64 {
		ctx.emitRegMemOp(opcode, dst, base, disp)
	} else {
		ctx.emitRegMemOp32(opcode, dst, base, disp)
	}
}

func amd64EmitIntBinaryImm(ctx *JITContext, op JITIntOp, width uint8, dst Reg, imm int32) {
	rex := byte(0x40)
	if width == 64 {
		rex |= 0x08
	}
	if dst >= 8 {
		rex |= 0x01
	}
	dstEnc := byte(dst & 7)
	if op == JITIntMul {
		if dst >= 8 {
			rex |= 0x04
		}
		opcode := byte(0x69)
		if imm >= -128 && imm <= 127 {
			opcode = 0x6B
		}
		ctx.emitBytes(rex, opcode, 0xC0|(dstEnc<<3)|dstEnc)
		if opcode == 0x6B {
			ctx.emitByte(byte(int8(imm)))
		} else {
			ctx.emitU32(uint32(imm))
		}
		return
	}
	var group byte
	switch op {
	case JITIntAdd:
		group = 0
	case JITIntOr:
		group = 1
	case JITIntAnd:
		group = 4
	case JITIntSub:
		group = 5
	case JITIntXor:
		group = 6
	default:
		panic("jit: invalid amd64 immediate operation")
	}
	opcode := byte(0x81)
	if imm >= -128 && imm <= 127 {
		opcode = 0x83
	}
	ctx.emitBytes(rex, opcode, 0xC0|group<<3|dstEnc)
	if opcode == 0x83 {
		ctx.emitByte(byte(int8(imm)))
	} else {
		ctx.emitU32(uint32(imm))
	}
}
