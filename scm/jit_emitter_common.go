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
// callees. MemCP's JIT call bridge supports at most sixteen ABI words.
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
	if jitPortableBackend {
		panic("jit: byte instruction reached fixed-width backend")
	}
	if ctx.registerInstructionDepth == 0 && (ctx.DeferredRegMoves.active != 0 || ctx.lazyFlags.FlagsID != 0) {
		ctx.FlushRegisterMoves()
	}
	ctx.ensureSpace(1)
	*(*byte)(ctx.Ptr) = b
	ctx.Ptr = unsafe.Add(ctx.Ptr, 1)
}

// emitBytes appends raw bytes to the writer.
func (ctx *JITContext) emitBytes(bs ...byte) {
	if jitPortableBackend {
		panic("jit: byte instructions reached fixed-width backend")
	}
	if ctx.registerInstructionDepth == 0 && (ctx.DeferredRegMoves.active != 0 || ctx.lazyFlags.FlagsID != 0) {
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
	if ctx.registerInstructionDepth == 0 && (ctx.DeferredRegMoves.active != 0 || ctx.lazyFlags.FlagsID != 0) {
		ctx.FlushRegisterMoves()
	}
	ctx.ensureSpace(4)
	*(*uint32)(ctx.Ptr) = v
	ctx.Ptr = unsafe.Add(ctx.Ptr, 4)
}

// emitU64 appends a little-endian uint64.
func (ctx *JITContext) emitU64(v uint64) {
	if ctx.registerInstructionDepth == 0 && (ctx.DeferredRegMoves.active != 0 || ctx.lazyFlags.FlagsID != 0) {
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

// EmitAtomicLoad64 and EmitAtomicStore64 have sync/atomic's sequentially
// consistent semantics. Plain loads/stores are insufficient on weakly ordered
// targets; the backend supplies the required acquire/release instructions.
func (ctx *JITContext) EmitAtomicLoad64(dst, base Reg, disp int32) {
	ctx.beginRegisterInstruction(jitRegisterMask(base), jitRegisterMask(dst))
	defer ctx.endRegisterInstruction()
	if jitPortableBackend {
		jitArchEmitNative(ctx, jitNativeAtomicLoad64, dst, base, 0, int64(disp))
		return
	}
	jitArchEmitLoad64(ctx, dst, base, disp)
}

func (ctx *JITContext) EmitAtomicStore64(src, base Reg, disp int32) {
	ctx.beginRegisterInstruction(jitRegisterMask(src, base), 0)
	defer ctx.endRegisterInstruction()
	if jitPortableBackend {
		jitArchEmitNative(ctx, jitNativeAtomicStore64, src, base, 0, int64(disp))
		return
	}
	// On AMD64 a full fence after the aligned store closes the Store->Load
	// ordering hole while preserving the source register.
	jitArchEmitStore64(ctx, src, base, disp)
	ctx.emitBytes(0x0F, 0xAE, 0xF0) // MFENCE
}

// jitNativeOp describes the remaining instruction families shared by the
// fixed-width backends. Integer operands retain their semantic JITIntOp API.
type jitNativeOp uint8

const (
	jitNativeReturn jitNativeOp = iota
	jitNativeProlog
	jitNativeLeave
	jitNativeEpilog
	jitNativeStackAdjust
	jitNativeAndImm
	jitNativeOrImm
	jitNativeAddImm
	jitNativeSubImm
	jitNativeAddImm32
	jitNativeSubImm32
	jitNativeMulImm
	jitNativeDivImm
	jitNativeRemImm
	jitNativeCmpImm
	jitNativeCmpByte
	jitNativeSetCondition
	jitNativeALU
	jitNativeZeroExtend32
	jitNativeShiftLeft
	jitNativeShiftRight
	jitNativeShiftSigned
	jitNativeShiftLeftReg
	jitNativeShiftRightReg
	jitNativeAddress
	jitNativeIndexAddress
	jitNativeIndexLoad
	jitNativeFCompare
	jitNativeFStore
	jitNativeFLoad
	jitNativeFloatOp
	jitNativeFToGPR
	jitNativeGPRToF
	jitNativeIntToFloat
	jitNativeFloatToInt
	jitNativeFZero
	jitNativeMoreStackCall
	jitNativeCall
	jitNativeAtomicLoad64
	jitNativeAtomicStore64
)

// jitEmitNativeCall saves the frame base in a relocatable stack slot. Go's
// ABIInternal has no callee-saved GPRs on RISC-V. The minimum caller frame also
// supplies the link-register home required by Go stack arguments/spills.
func jitEmitNativeCall(ctx *JITContext, target Reg, addr uint64, setup func(int32), roots []int32) {
	const callBytes = int32(jitGoSpillBytes + 16)
	ctx.EmitReserveStackBytes(callBytes)
	ctx.EmitStoreRegMem(ctx.FrameReg, ctx.StackReg, callBytes-8)
	ctx.setStackPointer(jitStackRootCallSP, callBytes-8, true)
	if setup != nil {
		setup(callBytes)
	}
	if addr != 0 {
		ctx.EmitMovRegImm64(jitNativeTemp, addr)
		target = jitNativeTemp
	}
	jitArchEmitNative(ctx, jitNativeCall, target, 0, 0, 0)
	ctx.recordSafepoint(roots, callBytes)
	ctx.EmitMovRegMem(ctx.FrameReg, ctx.StackReg, callBytes-8)
	ctx.setStackPointer(jitStackRootCallSP, callBytes-8, false)
	ctx.EmitReleaseStackBytes(callBytes)
}

func jitEmitClearFrame(ctx *JITContext) unsafe.Pointer {
	ctx.emitMovRegReg(RegRDI, RegRSP)
	fixup := jitArchEmitImmediateFixup(ctx, RegRCX)
	ctx.emitXorReg(RegR11)
	loop, done := ctx.ReserveLabel(), ctx.ReserveLabel()
	ctx.MarkLabel(loop)
	ctx.EmitCmpRegImm32(RegRCX, 0)
	ctx.EmitJcc(CcE, done)
	ctx.EmitStoreRegMem(RegR11, RegRDI, 0)
	ctx.EmitAddRegImm32(RegRDI, 8)
	ctx.EmitSubRegImm32(RegRCX, 1)
	ctx.EmitJmp(loop)
	ctx.MarkLabel(done)
	return fixup
}

func jitEmitStackCheck(ctx *JITContext, guardOffset int32, grow JITLabel) unsafe.Pointer {
	fixup := jitArchEmitImmediateFixup(ctx, jitNativeTemp)
	ctx.emitMovRegReg(RegR11, RegRSP)
	ctx.EmitCmpInt64(RegR11, jitNativeTemp)
	ctx.EmitJcc(CcB, grow)
	ctx.EmitSubInt64(RegR11, jitNativeTemp)
	jitArchEmitLoad64(ctx, jitNativeTemp, RegR14, guardOffset)
	ctx.EmitCmpInt64(RegR11, jitNativeTemp)
	ctx.EmitJcc(CcBE, grow)
	return fixup
}

func jitEmitStackGrow(ctx *JITContext, moreStackPC uintptr) {
	// A small, fully described frame exists before morestack. Saving the link
	// register in this record allows the runtime to unwind an entry safepoint
	// with the same declarative recipe as an ordinary JIT frame.
	jitArchEmitNative(ctx, jitNativeProlog, 0, 0, 0, 0)
	jitArchEmitNative(ctx, jitNativeStackAdjust, 0, 0, 0, -32)
	ctx.EmitStoreRegMem(RegRAX, RegRSP, 0)
	ctx.EmitStoreRegMem(RegRBX, RegRSP, 8)
	ctx.EmitStoreRegMem(RegRCX, RegRSP, 16)
	ctx.EmitMovRegImm64(jitNativeTemp, uint64(moreStackPC))
	jitArchEmitNative(ctx, jitNativeMoreStackCall, jitNativeTemp, 0, 0, 0)
	ctx.Safepoints = append(ctx.Safepoints, jitSafepoint{
		pcOffset: int32(uintptr(ctx.Ptr) - uintptr(ctx.Start)), entry: true,
		entryFrameWords: 5, entryPointerMap: []byte{1},
	})
	jitArchEmitNative(ctx, jitNativeAddress, RegRBP, RegRSP, 0, 32)
	ctx.EmitMovRegMem(RegRAX, RegRSP, 0)
	ctx.EmitMovRegMem(RegRBX, RegRSP, 8)
	ctx.EmitMovRegMem(RegRCX, RegRSP, 16)
	jitArchEmitNative(ctx, jitNativeStackAdjust, 0, 0, 0, 32)
	jitArchEmitNative(ctx, jitNativeLeave, 0, 0, 0, 0)
}

// EmitShiftLeft and EmitShiftRight describe a uint64 shift independently of
// machine shift registers. bounded is a proven count <64, permitting the
// backend to omit the Go out-of-range correction. Only dst is overwritten.
func (ctx *JITContext) EmitShiftLeft(dst, count Reg, bounded bool) {
	ctx.emitShift64(dst, count, true, bounded)
}

func (ctx *JITContext) EmitShiftRight(dst, count Reg, bounded bool) {
	ctx.emitShift64(dst, count, false, bounded)
}

func (ctx *JITContext) emitShift64(dst, count Reg, left, bounded bool) {
	ctx.beginRegisterInstruction(jitRegisterMask(dst, count), jitRegisterMask(dst))
	defer ctx.endRegisterInstruction()
	var done JITLabel
	if !bounded {
		inRange := ctx.ReserveLabel()
		done = ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(count, 64)
		ctx.EmitJcc(CondUnsignedBelow, inRange)
		ctx.EmitXorInt64(dst, dst)
		ctx.EmitJmp(done)
		ctx.MarkLabel(inRange)
	}
	jitArchEmitShift64(ctx, dst, count, left)
	if !bounded {
		ctx.MarkLabel(done)
	}
}

// EmitInt64ToFloatBits converts an integer in-place to IEEE float64 bits;
// the backend chooses its internal floating-point scratch register.
func (ctx *JITContext) EmitInt64ToFloatBits(reg Reg) {
	ctx.EmitCvtInt64ToFloat64(RegX0, reg)
}
