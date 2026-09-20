//go:build arm64

/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.
*/

package scm

func jitArchEmitReturn(ctx *JITContext) { emitARM64(ctx, 0xD65F03C0) }

func jitArchEmitLeafProlog(ctx *JITContext) {
	emitARM64(ctx, 0xA9BF7BFD) // STP X29, X30, [SP, #-16]!
	emitARM64(ctx, 0x910003FD) // MOV X29, SP
}

func jitArchEmitLeafEpilog(ctx *JITContext) {
	emitARM64(ctx, 0xA8C17BFD) // LDP X29, X30, [SP], #16
	jitArchEmitReturn(ctx)
}

func jitFlushInstructionCache(start, end uintptr)

func arm64GPR(reg Reg) uint32 {
	if reg > 31 {
		panic("jit: arm64 operation requires a general-purpose register")
	}
	return uint32(reg)
}

func emitARM64(ctx *JITContext, instruction uint32) { ctx.emitU32(instruction) }

func jitArchEmitMovRegReg(ctx *JITContext, dst, src Reg) {
	// ORR Xd, XZR, Xm (MOV alias).
	emitARM64(ctx, 0xAA0003E0|arm64GPR(src)<<16|arm64GPR(dst))
}

func jitArchEmitMovRegImm64(ctx *JITContext, dst Reg, imm uint64) {
	rd := arm64GPR(dst)
	emitARM64(ctx, 0xD2800000|uint32(imm&0xffff)<<5|rd) // MOVZ
	for shift := uint32(16); shift < 64; shift += 16 {
		part := uint32(imm>>shift) & 0xffff
		if part != 0 {
			emitARM64(ctx, 0xF2800000|(shift/16)<<21|part<<5|rd) // MOVK
		}
	}
}

func jitArchEmitLoad64(ctx *JITContext, dst, base Reg, disp int32) {
	rt, rn := arm64GPR(dst), arm64GPR(base)
	if disp >= 0 && disp%8 == 0 && disp/8 < 4096 {
		emitARM64(ctx, 0xF9400000|uint32(disp/8)<<10|rn<<5|rt) // LDR unsigned offset
		return
	}
	if disp >= -256 && disp <= 255 {
		emitARM64(ctx, 0xF8400000|(uint32(disp)&0x1ff)<<12|rn<<5|rt) // LDUR
		return
	}
	panic("jit: arm64 load displacement is out of range")
}

func jitArchEmitStore64(ctx *JITContext, src, base Reg, disp int32) {
	rt, rn := arm64GPR(src), arm64GPR(base)
	if disp >= 0 && disp%8 == 0 && disp/8 < 4096 {
		emitARM64(ctx, 0xF9000000|uint32(disp/8)<<10|rn<<5|rt) // STR unsigned offset
		return
	}
	if disp >= -256 && disp <= 255 {
		emitARM64(ctx, 0xF8000000|(uint32(disp)&0x1ff)<<12|rn<<5|rt) // STUR
		return
	}
	panic("jit: arm64 store displacement is out of range")
}

func arm64LoadStore(ctx *JITContext, reg, base Reg, disp int32, scale int32, unsignedBase, unscaledBase uint32) {
	rt, rn := arm64GPR(reg), arm64GPR(base)
	if disp >= 0 && disp%scale == 0 && disp/scale < 4096 {
		emitARM64(ctx, unsignedBase|uint32(disp/scale)<<10|rn<<5|rt)
		return
	}
	if disp >= -256 && disp <= 255 {
		emitARM64(ctx, unscaledBase|(uint32(disp)&0x1ff)<<12|rn<<5|rt)
		return
	}
	panic("jit: arm64 memory displacement is out of range")
}

func jitArchEmitLoad8(ctx *JITContext, dst, base Reg, disp int32) {
	arm64LoadStore(ctx, dst, base, disp, 1, 0x39400000, 0x38400000)
}

func jitArchEmitLoad16(ctx *JITContext, dst, base Reg, disp int32) {
	arm64LoadStore(ctx, dst, base, disp, 2, 0x79400000, 0x78400000)
}

func jitArchEmitLoad32(ctx *JITContext, dst, base Reg, disp int32) {
	arm64LoadStore(ctx, dst, base, disp, 4, 0xB9400000, 0xB8400000)
}

func jitArchEmitStore8(ctx *JITContext, src, base Reg, disp int32) {
	arm64LoadStore(ctx, src, base, disp, 1, 0x39000000, 0x38000000)
}

func jitArchEmitStore16(ctx *JITContext, src, base Reg, disp int32) {
	arm64LoadStore(ctx, src, base, disp, 2, 0x79000000, 0x78000000)
}

func jitArchEmitStore32(ctx *JITContext, src, base Reg, disp int32) {
	arm64LoadStore(ctx, src, base, disp, 4, 0xB9000000, 0xB8000000)
}

func jitArchEmitZeroReg(ctx *JITContext, reg Reg) {
	jitArchEmitMovRegImm64(ctx, reg, 0)
}

func jitArchEmitOrInt64(ctx *JITContext, dst, src Reg) {
	rd := arm64GPR(dst)
	emitARM64(ctx, 0xAA000000|arm64GPR(src)<<16|rd<<5|rd)
}

func jitArchEmitAndInt64(ctx *JITContext, dst, src Reg) {
	rd := arm64GPR(dst)
	emitARM64(ctx, 0x8A000000|arm64GPR(src)<<16|rd<<5|rd)
}

func jitArchEmitXorInt64(ctx *JITContext, dst, src Reg) {
	rd := arm64GPR(dst)
	emitARM64(ctx, 0xCA000000|arm64GPR(src)<<16|rd<<5|rd)
}

func jitArchEmitAddInt64(ctx *JITContext, dst, src Reg) {
	rd := arm64GPR(dst)
	emitARM64(ctx, 0x8B000000|arm64GPR(src)<<16|rd<<5|rd)
}

func jitArchEmitSubInt64(ctx *JITContext, dst, src Reg) {
	rd := arm64GPR(dst)
	emitARM64(ctx, 0xCB000000|arm64GPR(src)<<16|rd<<5|rd)
}

func jitArchEmitAddInt32(ctx *JITContext, dst, src Reg) {
	rd := arm64GPR(dst)
	emitARM64(ctx, 0x0B000000|arm64GPR(src)<<16|rd<<5|rd)
}

func jitArchEmitSubInt32(ctx *JITContext, dst, src Reg) {
	rd := arm64GPR(dst)
	emitARM64(ctx, 0x4B000000|arm64GPR(src)<<16|rd<<5|rd)
}

func jitArchEmitMulInt64(ctx *JITContext, dst, src Reg) {
	rd := arm64GPR(dst)
	emitARM64(ctx, 0x9B007C00|arm64GPR(src)<<16|rd<<5|rd) // MUL alias of MADD
}
