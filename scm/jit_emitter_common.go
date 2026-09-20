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

import "unsafe"

// JITIntOp is a semantic integer operation. Backends receive the operation
// together with its still-unmaterialized right operand and choose the shortest
// target instruction sequence in the one-pass emitter.
type JITIntOp uint8

const (
	JITIntAdd JITIntOp = iota
	JITIntSub
	JITIntMul
	JITIntAnd
	JITIntOr
	JITIntXor
)

type jitIntOperandKind uint8

const (
	jitIntOperandReg jitIntOperandKind = iota
	jitIntOperandImm
	jitIntOperandMem
)

// jitIntOperand is short-lived emitter state, not another optimization IR.
type jitIntOperand struct {
	kind jitIntOperandKind
	reg  Reg
	base Reg
	disp int32
	imm  int64
}

func jitCapturedEnv(en *Env) *JITEnv {
	if en == nil || en == &Globalenv {
		return nil
	}
	out := &JITEnv{Outer: jitCapturedEnv(en.Outer)}
	if len(en.VarsNumbered) != 0 {
		out.Numbered = make([]JITValueDesc, len(en.VarsNumbered))
		for i, value := range en.VarsNumbered {
			out.Numbered[i] = JITValueDesc{Loc: LocImm, Type: value.GetTag(), Imm: value}
		}
	}
	if len(out.Numbered) == 0 && out.Outer == nil {
		return nil
	}
	return out
}

// Keep the unwind marker above the register-argument spill area used by Go
// callees. MemCP's JIT call bridge supports at most nine ABI words (72 bytes).
const jitGoSpillBytes = uintptr(128)

// jitCompileProc compiles a Proc body to native machine code or returns nil.
func jitCompileProc(proc *Proc) []byte {
	code, _ := jitCompileProcWithRoots(proc)
	return code
}

// jitCompileProcWithRoots compiles a Proc body to native machine code and
// returns GC roots for pointer constants embedded into immediates.
func jitCompileProcWithRoots(proc *Proc) ([]byte, []unsafe.Pointer) {
	const defaultCodeBufSize = 16 * 1024
	ptr, arena, reservation := globalJITPool.Alloc(defaultCodeBufSize)
	buf := &execBuf{ptr: ptr, n: defaultCodeBufSize, arena: arena, reservation: reservation}
	codeLen, roots, dependencies, _, _, _, _ := jitCompileProcToExec(proc, buf, true)
	arena.complete(reservation, buf.stackMaps, dependencies)
	defer globalJITPool.Free(arena)
	if codeLen == 0 {
		return nil, nil
	}
	code := make([]byte, codeLen)
	copy(code, (*[1 << 30]byte)(buf.ptr)[:codeLen:codeLen])
	return code, roots
}

func (ctx *JITContext) ensureSpace(n uintptr) {
	if uintptr(ctx.Ptr)+n > uintptr(ctx.End) {
		panic(jitCodeOverflowPanic)
	}
}

// emitByte appends a single byte to the writer.
func (ctx *JITContext) emitByte(b byte) {
	if ctx.registerInstructionDepth == 0 && ctx.DeferredRegMoves.active != 0 {
		ctx.FlushRegisterMoves()
	}
	ctx.ensureSpace(1)
	*(*byte)(ctx.Ptr) = b
	ctx.Ptr = unsafe.Add(ctx.Ptr, 1)
}

// emitBytes appends raw bytes to the writer.
func (ctx *JITContext) emitBytes(bs ...byte) {
	if ctx.registerInstructionDepth == 0 && ctx.DeferredRegMoves.active != 0 {
		ctx.FlushRegisterMoves()
	}
	ctx.ensureSpace(uintptr(len(bs)))
	for _, b := range bs {
		*(*byte)(ctx.Ptr) = b
		ctx.Ptr = unsafe.Add(ctx.Ptr, 1)
	}
}

// emitU32 appends a little-endian uint32.
func (ctx *JITContext) emitU32(v uint32) {
	if ctx.registerInstructionDepth == 0 && ctx.DeferredRegMoves.active != 0 {
		ctx.FlushRegisterMoves()
	}
	ctx.ensureSpace(4)
	*(*uint32)(ctx.Ptr) = v
	ctx.Ptr = unsafe.Add(ctx.Ptr, 4)
}

// emitU64 appends a little-endian uint64.
func (ctx *JITContext) emitU64(v uint64) {
	if ctx.registerInstructionDepth == 0 && ctx.DeferredRegMoves.active != 0 {
		ctx.FlushRegisterMoves()
	}
	ctx.ensureSpace(8)
	*(*uint64)(ctx.Ptr) = v
	ctx.Ptr = unsafe.Add(ctx.Ptr, 8)
}

// EmitMovRegReg changes the logical register value now, but defers machine-code
// emission until another instruction needs concrete register contents. This
// allows adjacent moves produced by nested inline emitters to collapse.
func (ctx *JITContext) EmitMovRegReg(dst, src Reg) {
	ctx.deferRegMove(dst, src)
}

// emitMovRegReg emits a register move immediately. It is reserved for the move
// scheduler and fixed prologue/epilogue sequences.
func (ctx *JITContext) emitMovRegReg(dst, src Reg) {
	jitArchEmitMovRegReg(ctx, dst, src)
}

// EmitMovRegImm64 loads an arbitrary 64-bit immediate into a register.
func (ctx *JITContext) EmitMovRegImm64(dst Reg, imm uint64) {
	ctx.beginRegisterInstruction(0, jitRegisterMask(dst))
	defer ctx.endRegisterInstruction()
	jitArchEmitMovRegImm64(ctx, dst, imm)
}

func (ctx *JITContext) emitXorReg(reg Reg) {
	jitArchEmitZeroReg(ctx, reg)
}

func (ctx *JITContext) emitOrRegReg(dst, src Reg) {
	jitArchEmitOrInt64(ctx, dst, src)
}

func (ctx *JITContext) EmitAddInt64(dst, src Reg) {
	ctx.beginRegisterInstruction(jitRegisterMask(dst, src), jitRegisterMask(dst))
	defer ctx.endRegisterInstruction()
	jitArchEmitAddInt64(ctx, dst, src)
}

func (ctx *JITContext) EmitSubInt64(dst, src Reg) {
	ctx.beginRegisterInstruction(jitRegisterMask(dst, src), jitRegisterMask(dst))
	defer ctx.endRegisterInstruction()
	jitArchEmitSubInt64(ctx, dst, src)
}

// The 32-bit variants return a canonical zero-extended uint32 value.
func (ctx *JITContext) EmitAddInt32(dst, src Reg) {
	ctx.beginRegisterInstruction(jitRegisterMask(dst, src), jitRegisterMask(dst))
	defer ctx.endRegisterInstruction()
	jitArchEmitAddInt32(ctx, dst, src)
}

func (ctx *JITContext) EmitSubInt32(dst, src Reg) {
	ctx.beginRegisterInstruction(jitRegisterMask(dst, src), jitRegisterMask(dst))
	defer ctx.endRegisterInstruction()
	jitArchEmitSubInt32(ctx, dst, src)
}

func (ctx *JITContext) EmitImulInt64(dst, src Reg) {
	ctx.beginRegisterInstruction(jitRegisterMask(dst, src), jitRegisterMask(dst))
	defer ctx.endRegisterInstruction()
	jitArchEmitMulInt64(ctx, dst, src)
}

// EmitIntBinary consumes right in its current location. Stack operands remain
// stack operands until this call: amd64 can fold them into the ALU instruction,
// while load/store architectures materialize them into the reserved scratch
// register immediately before use. No optimization pass follows.
func (ctx *JITContext) EmitIntBinary(op JITIntOp, width uint8, dst Reg, right *JITValueDesc) {
	operand := jitIntOperand{}
	switch right.Loc {
	case LocReg:
		operand.kind = jitIntOperandReg
		operand.reg = right.Reg
	case LocImm:
		operand.kind = jitIntOperandImm
		operand.imm = right.Imm.Int()
	case LocStack:
		operand.kind = jitIntOperandMem
		operand.base = ctx.StackReg
		if right.StackOff < 0 {
			operand.base = ctx.FrameReg
		}
		operand.disp = right.StackOff
	default:
		panic("jit: integer operand is not scalar")
	}
	ctx.emitIntBinaryOperand(op, width, dst, operand)
}

// EmitIntBinaryImm preserves constants until architecture-specific selection.
func (ctx *JITContext) EmitIntBinaryImm(op JITIntOp, width uint8, dst Reg, imm int64) {
	ctx.emitIntBinaryOperand(op, width, dst, jitIntOperand{kind: jitIntOperandImm, imm: imm})
}

func (ctx *JITContext) emitIntBinaryOperand(op JITIntOp, width uint8, dst Reg, right jitIntOperand) {
	if width != 32 && width != 64 {
		panic("jit: integer binary width must be 32 or 64")
	}
	if right.kind == jitIntOperandImm && width == 64 {
		switch op {
		case JITIntAdd, JITIntSub, JITIntOr, JITIntXor:
			if right.imm == 0 {
				ctx.emitIntBinaryIdentity(dst)
				return
			}
		case JITIntMul:
			if right.imm == 1 {
				ctx.emitIntBinaryIdentity(dst)
				return
			}
			if right.imm == 0 {
				ctx.beginRegisterInstruction(0, jitRegisterMask(dst))
				defer ctx.endRegisterInstruction()
				jitArchEmitZeroReg(ctx, dst)
				return
			}
		case JITIntAnd:
			if right.imm == -1 {
				ctx.emitIntBinaryIdentity(dst)
				return
			}
		}
	}
	jitArchEmitIntBinary(ctx, op, width, dst, right, ctx.ScratchReg)
}

func (ctx *JITContext) emitIntBinaryIdentity(dst Reg) {
	mask := jitRegisterMask(dst)
	ctx.beginRegisterInstruction(mask, mask)
	ctx.endRegisterInstruction()
}

func (ctx *JITContext) emitMovRegMem(dst, base Reg, disp int32) {
	jitArchEmitLoad64(ctx, dst, base, disp)
}

func (ctx *JITContext) EmitMovRegMem(dst, base Reg, disp int32) {
	ctx.beginRegisterInstruction(jitRegisterMask(base), jitRegisterMask(dst))
	defer ctx.endRegisterInstruction()
	ctx.emitMovRegMem(dst, base, disp)
}

func (ctx *JITContext) EmitMovRegMemB(dst, base Reg, disp int32) {
	ctx.beginRegisterInstruction(jitRegisterMask(base), jitRegisterMask(dst))
	defer ctx.endRegisterInstruction()
	jitArchEmitLoad8(ctx, dst, base, disp)
}

func (ctx *JITContext) EmitMovRegMemW(dst, base Reg, disp int32) {
	ctx.beginRegisterInstruction(jitRegisterMask(base), jitRegisterMask(dst))
	defer ctx.endRegisterInstruction()
	jitArchEmitLoad16(ctx, dst, base, disp)
}

func (ctx *JITContext) EmitMovRegMemL(dst, base Reg, disp int32) {
	ctx.beginRegisterInstruction(jitRegisterMask(base), jitRegisterMask(dst))
	defer ctx.endRegisterInstruction()
	jitArchEmitLoad32(ctx, dst, base, disp)
}

func (ctx *JITContext) EmitOrInt64(dst, src Reg) {
	ctx.beginRegisterInstruction(jitRegisterMask(dst, src), jitRegisterMask(dst))
	defer ctx.endRegisterInstruction()
	jitArchEmitOrInt64(ctx, dst, src)
}

func (ctx *JITContext) EmitAndInt64(dst, src Reg) {
	ctx.beginRegisterInstruction(jitRegisterMask(dst, src), jitRegisterMask(dst))
	defer ctx.endRegisterInstruction()
	jitArchEmitAndInt64(ctx, dst, src)
}

func (ctx *JITContext) EmitXorInt64(dst, src Reg) {
	ctx.beginRegisterInstruction(jitRegisterMask(dst, src), jitRegisterMask(dst))
	defer ctx.endRegisterInstruction()
	jitArchEmitXorInt64(ctx, dst, src)
}

func (ctx *JITContext) emitStoreRegMem(src, base Reg, disp int32) {
	jitArchEmitStore64(ctx, src, base, disp)
}

func (ctx *JITContext) EmitStoreRegMem(src, base Reg, disp int32) {
	ctx.beginRegisterInstruction(jitRegisterMask(src, base), 0)
	defer ctx.endRegisterInstruction()
	ctx.emitStoreRegMem(src, base, disp)
}

func (ctx *JITContext) EmitStoreRegMemB(src, base Reg, disp int32) {
	ctx.beginRegisterInstruction(jitRegisterMask(src, base), 0)
	defer ctx.endRegisterInstruction()
	jitArchEmitStore8(ctx, src, base, disp)
}

func (ctx *JITContext) EmitStoreRegMemW(src, base Reg, disp int32) {
	ctx.beginRegisterInstruction(jitRegisterMask(src, base), 0)
	defer ctx.endRegisterInstruction()
	jitArchEmitStore16(ctx, src, base, disp)
}

func (ctx *JITContext) EmitStoreRegMemL(src, base Reg, disp int32) {
	ctx.beginRegisterInstruction(jitRegisterMask(src, base), 0)
	defer ctx.endRegisterInstruction()
	jitArchEmitStore32(ctx, src, base, disp)
}
