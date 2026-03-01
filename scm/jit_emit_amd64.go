//go:build amd64

/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package scm

import (
	"math"
	"unsafe"
)

// AMD64 register constants for the Go register ABI.
//
// Go register ABI (amd64): args in RAX, RBX, RCX, RDX, RSI, RDI, R8-R15
// Scmer return: RAX=ptr, RBX=aux
// Variadic args: RAX=slice_ptr, RBX=slice_len, RCX=slice_cap
const (
	RegRAX Reg = 0
	RegRCX Reg = 1
	RegRDX Reg = 2
	RegRBX Reg = 3
	RegRSP Reg = 4
	RegRBP Reg = 5
	RegRSI Reg = 6
	RegRDI Reg = 7
	RegR8  Reg = 8
	RegR9  Reg = 9
	RegR10 Reg = 10
	RegR11 Reg = 11
	RegR12 Reg = 12
	RegR13 Reg = 13
	RegR14 Reg = 14
	RegR15 Reg = 15
	// XMM registers start at 16
	RegX0  Reg = 16
	RegX1  Reg = 17
	RegX2  Reg = 18
	RegX3  Reg = 19
	RegX4  Reg = 20
	RegX5  Reg = 21
)

// emitByte appends a single byte to the writer.
func (w *JITWriter) emitByte(b byte) {
	*(*byte)(w.Ptr) = b
	w.Ptr = unsafe.Add(w.Ptr, 1)
}

// emitBytes appends raw bytes to the writer.
func (w *JITWriter) emitBytes(bs ...byte) {
	for _, b := range bs {
		*(*byte)(w.Ptr) = b
		w.Ptr = unsafe.Add(w.Ptr, 1)
	}
}

// emitU32 appends a little-endian uint32.
func (w *JITWriter) emitU32(v uint32) {
	*(*uint32)(w.Ptr) = v
	w.Ptr = unsafe.Add(w.Ptr, 4)
}

// emitU64 appends a little-endian uint64.
func (w *JITWriter) emitU64(v uint64) {
	*(*uint64)(w.Ptr) = v
	w.Ptr = unsafe.Add(w.Ptr, 8)
}

// --- Return emitters ---

// EmitReturnInt emits: MOV RAX, &scmerIntSentinel; MOV RBX, value; RET
// Constructs NewInt(value) in the return registers.
func (w *JITWriter) EmitReturnInt(src JITValueDesc) {
	// MOV RAX, imm64 (address of scmerIntSentinel)
	w.emitBytes(0x48, 0xB8)
	w.emitU64(uint64(uintptr(unsafe.Pointer(&scmerIntSentinel))))
	switch src.Loc {
	case LocReg:
		if src.Reg != RegRBX {
			// MOV RBX, src.Reg
			w.emitMovRegReg(RegRBX, src.Reg)
		}
	case LocImm:
		// MOV RBX, imm64
		w.emitBytes(0x48, 0xBB)
		w.emitU64(uint64(src.Imm.Int()))
	}
	w.emitByte(0xC3) // RET
}

// EmitReturnFloat emits: MOV RAX, &scmerFloatSentinel; MOVQ XMM→RBX; RET
// Constructs NewFloat(value) in the return registers.
func (w *JITWriter) EmitReturnFloat(src JITValueDesc) {
	// MOV RAX, imm64 (address of scmerFloatSentinel)
	w.emitBytes(0x48, 0xB8)
	w.emitU64(uint64(uintptr(unsafe.Pointer(&scmerFloatSentinel))))
	switch src.Loc {
	case LocReg:
		// MOVQ XMM -> RBX: 66 48 0F 7E C3 (for X0→RBX)
		w.emitMovqXmmToGpr(RegRBX, src.Reg)
	case LocImm:
		// MOV RBX, imm64 (raw float bits)
		w.emitBytes(0x48, 0xBB)
		w.emitU64(math.Float64bits(src.Imm.Float()))
	}
	w.emitByte(0xC3) // RET
}

// EmitReturnNil emits: XOR EAX,EAX; XOR EBX,EBX; RET
func (w *JITWriter) EmitReturnNil() {
	w.emitBytes(
		0x31, 0xC0, // XOR EAX, EAX
		0x31, 0xDB, // XOR EBX, EBX
		0xC3,       // RET
	)
}

// EmitReturnBool emits: XOR EAX,EAX; MOV RBX, makeAux(tagBool, 0/1); RET
func (w *JITWriter) EmitReturnBool(src JITValueDesc) {
	w.emitBytes(0x31, 0xC0) // XOR EAX, EAX (ptr = nil for bool)
	switch src.Loc {
	case LocImm:
		var val uint64
		if src.Imm.Bool() {
			val = 1
		}
		aux := makeAux(tagBool, val)
		w.emitBytes(0x48, 0xBB) // MOV RBX, imm64
		w.emitU64(aux)
	case LocReg:
		// MOVZX EBX, src.Reg (low byte); then OR with tagBool<<48
		// For simplicity: SHL + OR approach
		// First zero-extend the bool into RBX
		w.emitMovRegReg(RegRBX, src.Reg)
		w.emitBytes(0x48, 0x81, 0xE3) // AND RBX, 0x01
		w.emitU32(1)
		// MOV RCX, tagBool<<48
		w.emitBytes(0x48, 0xB9) // MOV RCX, imm64
		w.emitU64(uint64(tagBool) << 48)
		// OR RBX, RCX
		w.emitBytes(0x48, 0x09, 0xCB)
	}
	w.emitByte(0xC3) // RET
}

// --- Scmer construction emitters (no RET) ---

// EmitMakeBool constructs a Scmer bool into dst.Reg (ptr) and dst.Reg2 (aux).
// src.Reg holds the 0/1 boolean value.
func (w *JITWriter) EmitMakeBool(dst JITValueDesc, src JITValueDesc) {
	// dst.Reg = nil (XOR reg, reg)
	w.emitXorReg(dst.Reg)
	switch src.Loc {
	case LocImm:
		var bval uint64
		if src.Imm.Bool() {
			bval = 1
		}
		aux := makeAux(tagBool, bval)
		w.EmitMovRegImm64(dst.Reg2, aux)
	case LocReg:
		// dst.Reg2 = tagBool<<48 | (src.Reg & 1)
		if dst.Reg2 != src.Reg {
			w.emitMovRegReg(dst.Reg2, src.Reg)
		}
		w.emitAndRegImm32(dst.Reg2, 1)
		w.EmitMovRegImm64(RegR11, uint64(tagBool)<<48)
		w.emitOrRegReg(dst.Reg2, RegR11)
	}
}

// EmitMakeInt constructs a Scmer int into dst.Reg (ptr) and dst.Reg2 (aux).
// src.Reg holds the int64 value.
func (w *JITWriter) EmitMakeInt(dst JITValueDesc, src JITValueDesc) {
	w.EmitMovRegImm64(dst.Reg, uint64(uintptr(unsafe.Pointer(&scmerIntSentinel))))
	switch src.Loc {
	case LocReg:
		if dst.Reg2 != src.Reg {
			w.emitMovRegReg(dst.Reg2, src.Reg)
		}
	case LocImm:
		w.EmitMovRegImm64(dst.Reg2, uint64(src.Imm.Int()))
	}
}

// EmitMakeFloat constructs a Scmer float into dst.Reg (ptr) and dst.Reg2 (aux).
// src.Reg holds the float64 bits as uint64.
func (w *JITWriter) EmitMakeFloat(dst JITValueDesc, src JITValueDesc) {
	w.EmitMovRegImm64(dst.Reg, uint64(uintptr(unsafe.Pointer(&scmerFloatSentinel))))
	switch src.Loc {
	case LocReg:
		if dst.Reg2 != src.Reg {
			w.emitMovRegReg(dst.Reg2, src.Reg)
		}
	case LocImm:
		w.EmitMovRegImm64(dst.Reg2, uint64(src.Imm.Int())) // float bits stored as int64 in aux
	}
}

// EmitMakeNil constructs a Scmer nil into dst.Reg (ptr) and dst.Reg2 (aux).
func (w *JITWriter) EmitMakeNil(dst JITValueDesc) {
	w.emitXorReg(dst.Reg)
	w.emitXorReg(dst.Reg2)
}

// emitXorReg emits XOR r32, r32 (zeros 64-bit register via 32-bit op)
func (w *JITWriter) emitXorReg(r Reg) {
	if r >= 8 {
		w.emitBytes(0x45, 0x31, byte(0xC0|(byte(r&7)<<3)|byte(r&7)))
	} else {
		w.emitBytes(0x31, byte(0xC0|(byte(r)<<3)|byte(r)))
	}
}

// emitAndRegImm32 emits AND r64, sign-extended imm32
func (w *JITWriter) emitAndRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01
	}
	modrm := byte(0xE0) | byte(dst&7) // /4 = AND
	w.emitBytes(rex, 0x81, modrm)
	w.emitU32(uint32(imm))
}

// emitOrRegReg emits OR dst, src (64-bit)
func (w *JITWriter) emitOrRegReg(dst, src Reg) {
	w.emitAluRegReg(0x09, dst, src) // OR r/m64, r64
}

// --- ALU emitters (type-specialized) ---

// EmitAddInt64 emits: ADD dst, src (GPR += GPR)
func (w *JITWriter) EmitAddInt64(dst, src Reg) {
	w.emitAluRegReg(0x01, dst, src) // ADD r/m64, r64
}

// EmitSubInt64 emits: SUB dst, src (GPR -= GPR)
func (w *JITWriter) EmitSubInt64(dst, src Reg) {
	w.emitAluRegReg(0x29, dst, src) // SUB r/m64, r64
}

// EmitImulInt64 emits: IMUL dst, src (GPR *= GPR, signed)
func (w *JITWriter) EmitImulInt64(dst, src Reg) {
	// IMUL dst, src: REX.W + 0F AF /r (dst = dst * src)
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x04 // REX.R
	}
	if src >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | (byte(dst&7) << 3) | byte(src&7)
	w.emitBytes(rex, 0x0F, 0xAF, modrm)
}

// EmitAddFloat64 emits: ADDSD dst, src (XMM += XMM)
func (w *JITWriter) EmitAddFloat64(dst, src Reg) {
	w.emitSseOp(0x58, dst, src) // ADDSD
}

// EmitSubFloat64 emits: SUBSD dst, src (XMM -= XMM)
func (w *JITWriter) EmitSubFloat64(dst, src Reg) {
	w.emitSseOp(0x5C, dst, src) // SUBSD
}

// EmitMulFloat64 emits: MULSD dst, src (XMM *= XMM)
func (w *JITWriter) EmitMulFloat64(dst, src Reg) {
	w.emitSseOp(0x59, dst, src) // MULSD
}

// EmitDivFloat64 emits: DIVSD dst, src (XMM /= XMM)
func (w *JITWriter) EmitDivFloat64(dst, src Reg) {
	w.emitSseOp(0x5E, dst, src) // DIVSD
}

// --- Conversion emitters ---

// EmitCvtInt64ToFloat64 emits: CVTSI2SDQ xmmDst, gprSrc
func (w *JITWriter) EmitCvtInt64ToFloat64(xmmDst, gprSrc Reg) {
	// F2 REX.W 0F 2A /r
	xmm := xmmDst - 16 // convert to XMM index
	rex := byte(0x48)
	if xmm >= 8 {
		rex |= 0x04 // REX.R
	}
	if gprSrc >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | (byte(xmm&7) << 3) | byte(gprSrc&7)
	w.emitBytes(0xF2, rex, 0x0F, 0x2A, modrm)
}

// EmitXorpdReg emits: XORPD xmm, xmm (zero a float register)
func (w *JITWriter) EmitXorpdReg(xmm Reg) {
	r := xmm - 16
	modrm := byte(0xC0) | (byte(r&7) << 3) | byte(r&7)
	if r >= 8 {
		w.emitBytes(0x66, 0x45, 0x0F, 0x57, modrm)
	} else {
		w.emitBytes(0x66, 0x0F, 0x57, modrm)
	}
}

// --- Load emitters ---

// EmitLoadArgInt64 emits code to load the int64 value of the idx-th variadic
// arg directly from the Scmer slice. Only valid when JIT type = int64.
// Loads a[idx].aux (which IS the raw int64) into dstReg.
// sliceBase is the GPR holding the slice pointer.
func (w *JITWriter) EmitLoadArgInt64(dst, sliceBase Reg, idx int) {
	// MOV dst, [sliceBase + idx*16 + 8]  (aux field)
	w.emitMovRegMem(dst, sliceBase, int32(idx*16+8))
}

// EmitLoadArgFloat64 emits code to load the float64 value of the idx-th arg.
// Only valid when JIT type = float64.
// Loads a[idx].aux bits into xmmDst via MOVQ.
func (w *JITWriter) EmitLoadArgFloat64(xmmDst, sliceBase Reg, idx int) {
	// MOVQ xmm, [sliceBase + idx*16 + 8]
	w.emitMovqMemToXmm(xmmDst, sliceBase, int32(idx*16+8))
}

// --- Compare emitters ---

// EmitCmpInt64 emits: CMP reg1, reg2
func (w *JITWriter) EmitCmpInt64(a, b Reg) {
	w.emitAluRegReg(0x39, a, b) // CMP r/m64, r64
}

// EmitJcc emits a conditional jump with a rel32 fixup.
func (w *JITWriter) EmitJcc(cc byte, labelID uint8) {
	w.emitBytes(0x0F, 0x80|cc) // Jcc rel32
	w.AddFixup(labelID, 4, true)
	w.emitU32(0) // placeholder
}

// EmitJmp emits an unconditional JMP rel32.
func (w *JITWriter) EmitJmp(labelID uint8) {
	w.emitByte(0xE9) // JMP rel32
	w.AddFixup(labelID, 4, true)
	w.emitU32(0) // placeholder
}

// Condition code constants for EmitJcc
const (
	CcE  byte = 0x04 // JE  / JZ  (ZF=1)
	CcNE byte = 0x05 // JNE / JNZ (ZF=0)
	CcL  byte = 0x0C // JL        (SF!=OF)
	CcGE byte = 0x0D // JGE       (SF=OF)
	CcLE byte = 0x0E // JLE       (ZF=1 || SF!=OF)
	CcG  byte = 0x0F // JG        (ZF=0 && SF=OF)
	CcB  byte = 0x02 // JB  (unsigned <)
	CcAE byte = 0x03 // JAE (unsigned >=)
)

// --- MOV helpers ---

// emitMovRegReg emits MOV dst, src (64-bit GPR to GPR)
func (w *JITWriter) emitMovRegReg(dst, src Reg) {
	rex := byte(0x48)
	if src >= 8 {
		rex |= 0x04 // REX.R
	}
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | (byte(src&7) << 3) | byte(dst&7)
	w.emitBytes(rex, 0x89, modrm) // MOV r/m64, r64
}

// EmitMovRegImm64 emits MOV reg, imm64
func (w *JITWriter) EmitMovRegImm64(dst Reg, imm uint64) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	w.emitBytes(rex, 0xB8|byte(dst&7))
	w.emitU64(imm)
}

// emitRegMemOp emits <opcode> dst, [base + disp] (REX.W r64, r/m64 with ModRM)
// opcode: 0x8B = MOV (load), 0x8D = LEA (address computation)
func (w *JITWriter) emitRegMemOp(opcode byte, dst, base Reg, disp int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x04 // REX.R
	}
	if base >= 8 {
		rex |= 0x01 // REX.B
	}
	baseEnc := byte(base & 7)
	dstEnc := byte(dst & 7)

	if disp == 0 && baseEnc != 5 { // RBP/R13 always needs disp
		modrm := (dstEnc << 3) | baseEnc
		if baseEnc == 4 { // RSP/R12 needs SIB
			w.emitBytes(rex, opcode, modrm, 0x24)
		} else {
			w.emitBytes(rex, opcode, modrm)
		}
	} else if disp >= -128 && disp <= 127 {
		modrm := 0x40 | (dstEnc << 3) | baseEnc
		if baseEnc == 4 {
			w.emitBytes(rex, opcode, modrm, 0x24, byte(int8(disp)))
		} else {
			w.emitBytes(rex, opcode, modrm, byte(int8(disp)))
		}
	} else {
		modrm := 0x80 | (dstEnc << 3) | baseEnc
		if baseEnc == 4 {
			w.emitBytes(rex, opcode, modrm, 0x24)
		} else {
			w.emitBytes(rex, opcode, modrm)
		}
		w.emitU32(uint32(disp))
	}
}

// emitMovRegMem emits MOV dst, [base + disp32] (load 64-bit from memory)
func (w *JITWriter) emitMovRegMem(dst, base Reg, disp int32) {
	w.emitRegMemOp(0x8B, dst, base, disp)
}

// EmitLeaRegMem emits LEA dst, [base + disp32] (compute address, no memory access)
// For IndexAddr: LEA dst, [sliceBase + idx*16] computes &a[idx]
func (w *JITWriter) EmitLeaRegMem(dst, base Reg, disp int32) {
	w.emitRegMemOp(0x8D, dst, base, disp)
}

// --- SSE helpers ---

// emitSseOp emits F2 0F <op> xmmDst, xmmSrc (scalar double operation)
func (w *JITWriter) emitSseOp(op byte, dst, src Reg) {
	d := dst - 16 // XMM index
	s := src - 16
	rex := byte(0)
	if d >= 8 || s >= 8 {
		rex = 0x40
		if d >= 8 {
			rex |= 0x04
		}
		if s >= 8 {
			rex |= 0x01
		}
	}
	modrm := byte(0xC0) | (byte(d&7) << 3) | byte(s&7)
	if rex != 0 {
		w.emitBytes(0xF2, rex, 0x0F, op, modrm)
	} else {
		w.emitBytes(0xF2, 0x0F, op, modrm)
	}
}

// emitMovqXmmToGpr emits MOVQ gprDst, xmmSrc (66 REX.W 0F 7E /r)
func (w *JITWriter) emitMovqXmmToGpr(gpr, xmm Reg) {
	x := xmm - 16
	rex := byte(0x48) // REX.W
	if x >= 8 {
		rex |= 0x04 // REX.R
	}
	if gpr >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | (byte(x&7) << 3) | byte(gpr&7)
	w.emitBytes(0x66, rex, 0x0F, 0x7E, modrm)
}

// emitMovqMemToXmm emits MOVQ xmmDst, [base + disp32] (F3 0F 7E /r m64)
func (w *JITWriter) emitMovqMemToXmm(xmm, base Reg, disp int32) {
	x := xmm - 16
	rex := byte(0)
	if x >= 8 || base >= 8 {
		rex = 0x40
		if x >= 8 {
			rex |= 0x04
		}
		if base >= 8 {
			rex |= 0x01
		}
	}
	baseEnc := byte(base & 7)
	xEnc := byte(x & 7)

	if rex != 0 {
		w.emitBytes(0xF3, rex, 0x0F, 0x7E)
	} else {
		w.emitBytes(0xF3, 0x0F, 0x7E)
	}

	if disp >= -128 && disp <= 127 {
		modrm := 0x40 | (xEnc << 3) | baseEnc
		if baseEnc == 4 {
			w.emitBytes(modrm, 0x24, byte(int8(disp)))
		} else {
			w.emitBytes(modrm, byte(int8(disp)))
		}
	} else {
		modrm := 0x80 | (xEnc << 3) | baseEnc
		if baseEnc == 4 {
			w.emitBytes(modrm, 0x24)
		} else {
			w.emitBytes(modrm)
		}
		w.emitU32(uint32(disp))
	}
}

// --- Compare helpers ---

// EmitCmpRegImm32 emits CMP r64, sign-extended imm32
func (w *JITWriter) EmitCmpRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xF8) | byte(dst&7) // /7 = CMP
	w.emitBytes(rex, 0x81, modrm)
	w.emitU32(uint32(imm))
}

// EmitSetcc emits SETcc r/m8 + MOVZX r32, r8 → zero-extended 0 or 1 in full 64-bit register
func (w *JITWriter) EmitSetcc(dst Reg, cc byte) {
	dstEnc := byte(dst & 7)
	// SETcc r/m8: 0F 9x /0
	if dst >= 8 {
		w.emitBytes(0x41, 0x0F, 0x90|cc, 0xC0|dstEnc)
	} else if dst >= 4 {
		w.emitBytes(0x40, 0x0F, 0x90|cc, 0xC0|dstEnc) // REX for SIL/DIL/BPL/SPL
	} else {
		w.emitBytes(0x0F, 0x90|cc, 0xC0|dstEnc)
	}
	// MOVZX r32, r8: 0F B6 /r (32-bit write zeros upper 32)
	modrm := byte(0xC0) | (dstEnc << 3) | dstEnc
	if dst >= 8 {
		w.emitBytes(0x45, 0x0F, 0xB6, modrm)
	} else if dst >= 4 {
		w.emitBytes(0x40, 0x0F, 0xB6, modrm)
	} else {
		w.emitBytes(0x0F, 0xB6, modrm)
	}
}

// --- Shift emitters ---

// EmitShlRegImm8 emits SHL r64, imm8 (logical shift left by immediate)
func (w *JITWriter) EmitShlRegImm8(dst Reg, imm uint8) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xE0) | byte(dst&7) // /4 = SHL
	w.emitBytes(rex, 0xC1, modrm, imm)
}

// EmitShrRegImm8 emits SHR r64, imm8 (logical shift right by immediate)
func (w *JITWriter) EmitShrRegImm8(dst Reg, imm uint8) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xE8) | byte(dst&7) // /5 = SHR
	w.emitBytes(rex, 0xC1, modrm, imm)
}

// --- GetTag ---

// EmitGetTagDesc extracts the type tag from a Scmer value descriptor.
// Follows the standard emitter contract: consumes src (frees registers),
// places the tag int into result according to result.Loc.
func (ctx *JITContext) EmitGetTagDesc(src *JITValueDesc, result JITValueDesc) JITValueDesc {
	if src.Loc == LocImm {
		r := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(src.Imm.GetTag()))}
		if result.Loc == LocAny {
			return r
		}
		ctx.W.EmitMakeInt(result, r)
		return result
	}
	if src.Type != JITTypeUnknown {
		// Type is known at compile time — constant-fold
		ctx.FreeDesc(src)
		r := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(src.Type))}
		if result.Loc == LocAny {
			return r
		}
		ctx.W.EmitMakeInt(result, r)
		return result
	}
	dst := ctx.AllocReg()
	ctx.W.emitGetTagRegs(dst, src.Reg, src.Reg2)
	ctx.FreeDesc(src)
	r := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: dst}
	if result.Loc == LocAny {
		return r
	}
	ctx.W.EmitMakeInt(result, r)
	ctx.FreeReg(dst)
	return result
}

// EmitTagEquals checks if a Scmer's type tag equals a constant.
// Equivalent to GetTag(src) == tag. Consumes src, produces a bool.
func (ctx *JITContext) EmitTagEquals(src *JITValueDesc, tag uint16, result JITValueDesc) JITValueDesc {
	if src.Loc == LocImm {
		r := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(src.Imm.GetTag() == tag)}
		if result.Loc == LocAny {
			return r
		}
		ctx.W.EmitMakeBool(result, r)
		return result
	}
	if src.Type != JITTypeUnknown {
		// Type is known at compile time — constant-fold
		ctx.FreeDesc(src)
		r := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(src.Type == tag)}
		if result.Loc == LocAny {
			return r
		}
		ctx.W.EmitMakeBool(result, r)
		return result
	}
	tagReg := ctx.AllocReg()
	ctx.W.emitGetTagRegs(tagReg, src.Reg, src.Reg2)
	ctx.FreeDesc(src)
	ctx.W.EmitCmpRegImm32(tagReg, int32(tag))
	ctx.W.EmitSetcc(tagReg, CcE)
	r := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: tagReg}
	if result.Loc == LocAny {
		return r
	}
	ctx.W.EmitMakeBool(result, r)
	ctx.FreeReg(tagReg)
	return result
}

// EmitMovToReg moves a JITValueDesc value into a specific GPR register.
// Handles LocImm (materializes constant) and LocReg (register-to-register move).
func (ctx *JITContext) EmitMovToReg(dst Reg, src JITValueDesc) {
	switch src.Loc {
	case LocImm:
		ctx.W.EmitMovRegImm64(dst, uint64(src.Imm.Int()))
	case LocReg:
		if src.Reg != dst {
			ctx.W.emitMovRegReg(dst, src.Reg)
		}
	}
}


// emitGetTagRegs emits inline code for (Scmer).GetTag().
// Input: ptrReg holds s.ptr, auxReg holds s.aux.
// Output: result in dstReg as uint16.
// Logic: if ptr == &scmerIntSentinel → tagInt (4)
//        if ptr == &scmerFloatSentinel → tagFloat (3)
//        else → aux >> 48
func (w *JITWriter) emitGetTagRegs(dst, ptrReg, auxReg Reg) {
	// CMP ptrReg, &scmerIntSentinel (via R11 as scratch)
	w.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(&scmerIntSentinel))))
	w.EmitCmpInt64(ptrReg, RegR11)
	// JE .is_int (patch later)
	w.emitBytes(0x0F, 0x84) // JE rel32
	isIntFixup := w.Ptr
	w.emitU32(0)

	// CMP ptrReg, &scmerFloatSentinel
	w.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(&scmerFloatSentinel))))
	w.EmitCmpInt64(ptrReg, RegR11)
	// JE .is_float (patch later)
	w.emitBytes(0x0F, 0x84) // JE rel32
	isFloatFixup := w.Ptr
	w.emitU32(0)

	// Default: dst = aux >> 48
	if dst != auxReg {
		w.emitMovRegReg(dst, auxReg)
	}
	w.EmitShrRegImm8(dst, 48)
	// JMP .done
	w.emitByte(0xE9) // JMP rel32
	doneFixup := w.Ptr
	w.emitU32(0)

	// .is_int: dst = tagInt (4)
	isIntTarget := w.Ptr
	w.EmitMovRegImm64(dst, uint64(tagInt))
	// JMP .done
	w.emitByte(0xE9) // JMP rel32
	doneFixup2 := w.Ptr
	w.emitU32(0)

	// .is_float: dst = tagFloat (3)
	isFloatTarget := w.Ptr
	w.EmitMovRegImm64(dst, uint64(tagFloat))
	// fall through to .done

	// .done:
	doneTarget := w.Ptr

	// Patch fixups
	*(*int32)(isIntFixup) = int32(uintptr(isIntTarget) - uintptr(isIntFixup) - 4)
	*(*int32)(isFloatFixup) = int32(uintptr(isFloatTarget) - uintptr(isFloatFixup) - 4)
	*(*int32)(doneFixup) = int32(uintptr(doneTarget) - uintptr(doneFixup) - 4)
	*(*int32)(doneFixup2) = int32(uintptr(doneTarget) - uintptr(doneFixup2) - 4)
}

// --- GPR ALU encoding helper ---

// emitAluRegReg emits a REX.W ALU op: <opcode> r/m64, r64
// opcode: 0x01=ADD, 0x29=SUB, 0x39=CMP, 0x09=OR, 0x21=AND, 0x31=XOR
func (w *JITWriter) emitAluRegReg(opcode byte, dst, src Reg) {
	rex := byte(0x48)
	if src >= 8 {
		rex |= 0x04
	}
	if dst >= 8 {
		rex |= 0x01
	}
	modrm := byte(0xC0) | (byte(src&7) << 3) | byte(dst&7)
	w.emitBytes(rex, opcode, modrm)
}
