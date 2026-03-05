//go:build amd64

/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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
	"fmt"
	"math"
	"syscall"
	"unsafe"
)

// TODO: create this file for other architectures, too

var jitCodeOverflowPanic = &struct{}{}

// jitCompileProc compiles a Proc body to amd64 machine code or returns nil.
func jitCompileProc(proc *Proc) []byte {
	code, _ := jitCompileProcWithRoots(proc)
	return code
}

// jitCompileProcWithRoots compiles a Proc body to amd64 machine code and
// returns GC roots for pointer constants embedded into immediates.
func jitCompileProcWithRoots(proc *Proc) ([]byte, []unsafe.Pointer) {
	const defaultCodeBufSize = 16 * 1024
	buf, err := allocExec(defaultCodeBufSize)
	if err != nil {
		return nil, nil
	}
	defer syscall.Munmap((*[1 << 30]byte)(buf.ptr)[:buf.n:buf.n])
	codeLen, roots, _ := jitCompileProcToExec(proc, buf)
	if codeLen == 0 {
		return nil, nil
	}
	code := make([]byte, codeLen)
	copy(code, (*[1 << 30]byte)(buf.ptr)[:codeLen:codeLen])
	return code, roots
}

// jitCompileProcToExec compiles a Proc body directly into writable executable memory.
// Returns code length, GC roots and an overflow flag.
func jitCompileProcToExec(proc *Proc, buf *execBuf) (int, []unsafe.Pointer, bool) {
	body := proc.Body
	if body.GetTag() == tagSourceInfo {
		body = body.SourceInfo().value
	}
	return jitCompileExprBodyToExec(proc, body, proc.NumVars, buf)
}

// jitCompileExprBodyToExec compiles a Scheme expression body into a writable
// executable buffer using Declaration.JITEmit callbacks.
func jitCompileExprBodyToExec(proc *Proc, body Scmer, numVars int, buf *execBuf) (codeLen int, roots []unsafe.Pointer, overflow bool) {
	defer func() {
		if r := recover(); r != nil {
			if r == jitCodeOverflowPanic {
				overflow = true
			}
			if JITLog {
				fmt.Println("JIT panic", r)
			}
			codeLen = 0
			roots = nil
		}
	}()

	// Free registers: all GPRs except RAX (result ptr), RBX (result aux),
	// RSP, RBP, R11 (scratch), R12 (slice base), R14 (Go goroutine ptr "g")
	freeRegs := uint64((1 << uint(RegRCX)) | (1 << uint(RegRDX)) |
		(1 << uint(RegRSI)) | (1 << uint(RegRDI)) |
		(1 << uint(RegR8)) | (1 << uint(RegR9)) | (1 << uint(RegR10)) |
		(1 << uint(RegR13)) | (1 << uint(RegR15)))
	ctx := &JITContext{
		Ptr:       buf.ptr,
		Start:     buf.ptr,
		End:       unsafe.Add(buf.ptr, buf.n),
		FreeRegs:  freeRegs,
		AllRegs:   freeRegs,
		SliceBase: RegR12,
	}

	// Emit: MOV R12, RAX (save slice base pointer)
	ctx.emitMovRegReg(RegR12, RegRAX)
	// Copy incoming variadic arguments into an emitter-local stack frame.
	// Go helper calls use PUSH/POP heavily and may overlap caller-provided
	// argument memory; reading NthLocalVar from a private copy is stable.
	frameBytes := 0
	if numVars > 0 {
		slots := numVars
		frameBytes = slots * 16
		if frameBytes < 128 {
			ctx.EmitSubRSP(uint8(frameBytes))
		} else {
			ctx.emitBytes(0x48, 0x81, 0xEC)
			ctx.emitU32(uint32(frameBytes))
		}
		for i := 0; i < slots; i++ {
			srcOff := int32(i * 16)
			dstOff := int32(i * 16)
			ctx.EmitMovRegMem(RegR11, RegR12, srcOff)
			ctx.EmitStoreRegMem(RegR11, RegRSP, dstOff)
			ctx.EmitMovRegMem(RegR11, RegR12, srcOff+8)
			ctx.EmitStoreRegMem(RegR11, RegRSP, dstOff+8)
		}
		ctx.emitMovRegReg(RegR12, RegRSP)
		ctx.SliceBaseTracksRSP = true
	}

	// Map lambda parameters to local stack slots so symbol lookup remains correct
	// even when the optimizer did not rewrite body symbols to NthLocalVar.
	if proc != nil {
		var vars map[Symbol]JITValueDesc
		putVar := func(sym Symbol, off int32) {
			if vars == nil {
				vars = make(map[Symbol]JITValueDesc, numVars)
			}
			vars[sym] = JITValueDesc{
				Loc:      LocStackPair,
				Type:     JITTypeUnknown,
				StackOff: off,
			}
		}
		switch proc.Params.GetTag() {
		case tagSlice:
			params := proc.Params.Slice()
			for i := 0; i < len(params) && i < numVars; i++ {
				if params[i].GetTag() != tagSymbol {
					continue
				}
				putVar(params[i].Symbol(), int32(i*16))
			}
		case tagSymbol:
			if numVars > 0 {
				putVar(proc.Params.Symbol(), 0)
			}
		}
		if len(vars) > 0 {
			ctx.Env = &JITEnv{Vars: vars}
		}
	}

	// Compile body, place result into RAX+RBX (Scmer return registers)
	result := JITValueDesc{Loc: LocRegPair, Reg: RegRAX, Reg2: RegRBX}
	desc := jitCompileExpr(ctx, body, RegR12, result)

	// If result came back as LocImm, materialize into RAX+RBX
	if desc.Loc == LocImm {
		switch desc.Imm.GetTag() {
		case tagBool:
			ctx.EmitMakeBool(result, desc)
		case tagInt:
			ctx.EmitMakeInt(result, desc)
		case tagFloat:
			ctx.EmitMakeFloat(result, desc)
		case tagNil:
			ctx.EmitMakeNil(result)
		default:
			return 0, nil, false
		}
		if frameBytes > 0 {
			ctx.EmitAddRSP32(int32(frameBytes))
		}
		ctx.emitByte(0xC3) // RET
	} else {
		// Ensure non-immediate results are in ABI return registers.
		ctx.EnsureDesc(&desc)
		switch desc.Loc {
		case LocRegPair:
			if desc.Reg != RegRAX {
				ctx.emitMovRegReg(RegRAX, desc.Reg)
			}
			if desc.Reg2 != RegRBX {
				ctx.emitMovRegReg(RegRBX, desc.Reg2)
			}
		case LocReg:
			ret := JITValueDesc{Loc: LocRegPair, Reg: RegRAX, Reg2: RegRBX}
			switch desc.Type {
			case tagBool:
				ctx.EmitMakeBool(ret, desc)
			case tagInt:
				ctx.EmitMakeInt(ret, desc)
			case tagFloat:
				ctx.EmitMakeFloat(ret, desc)
			default:
				return 0, nil, false
			}
		default:
			return 0, nil, false
		}
		if frameBytes > 0 {
			ctx.EmitAddRSP32(int32(frameBytes))
		}
		ctx.emitByte(0xC3) // RET
	}

	ctx.ResolveFixupsFinal()
	codeLen = int(uintptr(ctx.Ptr) - uintptr(ctx.Start))
	return codeLen, ctx.ConstRoots, false
}

func jitEnsureResultPair(ctx *JITContext, result JITValueDesc) JITValueDesc {
	if result.Loc == LocRegPair {
		return JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: result.Reg, Reg2: result.Reg2}
	}
	return JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
}

func jitPlaceIntoPair(ctx *JITContext, src *JITValueDesc, target JITValueDesc) JITValueDesc {
	if target.Loc != LocRegPair {
		panic("jit: jitPlaceIntoPair requires LocRegPair target")
	}
	// Keep descriptor location in sync with spill metadata before we read Reg/Reg2.
	if src.Loc != LocImm {
		ctx.EnsureDesc(src)
	}
	switch src.Loc {
	case LocImm:
		switch src.Imm.GetTag() {
		case tagBool:
			ctx.EmitMakeBool(target, *src)
		case tagInt:
			ctx.EmitMakeInt(target, *src)
		case tagFloat:
			ctx.EmitMakeFloat(target, *src)
		case tagNil:
			ctx.EmitMakeNil(target)
		default:
			ptr, aux := src.Imm.RawWords()
			ctx.EmitMovRegImm64(target.Reg, uint64(ptr))
			ctx.EmitMovRegImm64(target.Reg2, aux)
		}
		return target
	case LocStack, LocStackPair:
		ctx.EnsureDesc(src)
		return jitPlaceIntoPair(ctx, src, target)
	case LocRegPair:
		if src.Reg != target.Reg {
			ctx.emitMovRegReg(target.Reg, src.Reg)
		}
		if src.Reg2 != target.Reg2 {
			ctx.emitMovRegReg(target.Reg2, src.Reg2)
		}
		if src.Reg != target.Reg && src.Reg2 != target.Reg2 {
			ctx.FreeDesc(src)
		}
		return target
	case LocReg:
		switch src.Type {
		case tagBool:
			ctx.EmitMakeBool(target, *src)
		case tagInt:
			ctx.EmitMakeInt(target, *src)
		case tagFloat:
			ctx.EmitMakeFloat(target, *src)
		default:
			panic("jit: cannot materialize LocReg with unknown type into Scmer pair")
		}
		ctx.FreeDesc(src)
		return target
	default:
		panic("jit: unsupported source location for pair materialization")
	}
}

func jitCondToBool(ctx *JITContext, cond *JITValueDesc) JITValueDesc {
	return ctx.EmitBoolDesc(cond, JITValueDesc{Loc: LocAny})
}

// jitCondToBoolBorrowed evaluates truthiness without consuming cond.
func jitCondToBoolBorrowed(ctx *JITContext, cond *JITValueDesc) JITValueDesc {
	if cond.Loc == LocImm {
		return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(cond.Imm.Bool())}
	}
	if cond.Type == tagNil {
		return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
	}
	if cond.Type == tagDate {
		return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
	}

	// Known primitive truthiness without consuming the original value.
	if cond.Type == tagBool || cond.Type == tagInt || cond.Type == tagFloat {
		tmp := *cond
		ctx.EnsureDesc(&tmp)
		tmpLoc := tmp.Loc
		tmpReg := tmp.Reg
		tmpReg2 := tmp.Reg2
		var valReg Reg
		switch tmp.Loc {
		case LocReg:
			valReg = tmp.Reg
		case LocRegPair:
			valReg = tmp.Reg2
		default:
			panic("jit: borrowed bool test needs register value")
		}
		dst := ctx.AllocReg()
		if dst != valReg {
			ctx.emitMovRegReg(dst, valReg)
		}
		if cond.Type == tagFloat {
			mask := ctx.AllocReg()
			ctx.EmitMovRegImm64(mask, 0x7fffffffffffffff)
			ctx.EmitAndInt64(dst, mask)
			ctx.FreeReg(mask)
		} else if cond.Type == tagBool {
			// Bool payload is auxVal in bits [63:8]; low 8 bits hold the tag.
			ctx.EmitShrRegImm8(dst, 8)
		}
		ctx.EmitCmpRegImm32(dst, 0)
		ctx.EmitSetcc(dst, CcNE)
		switch tmpLoc {
		case LocReg:
			if dst != tmpReg {
				ctx.FreeReg(tmpReg)
			}
		case LocRegPair:
			if dst == tmpReg {
				ctx.FreeReg(tmpReg2)
			} else if dst == tmpReg2 {
				ctx.FreeReg(tmpReg)
			} else {
				ctx.FreeReg(tmpReg)
				ctx.FreeReg(tmpReg2)
			}
		default:
			ctx.FreeDesc(&tmp)
		}
		return JITValueDesc{Loc: LocReg, Type: tagBool, Reg: dst}
	}

	out := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Bool), []JITValueDesc{*cond}, 1)
	ctx.EmitAndRegImm32(out.Reg, 1)
	out.Type = tagBool
	return out
}

// jitIsNilBorrowed checks nil-ness without consuming v.
func jitIsNilBorrowed(ctx *JITContext, v *JITValueDesc) JITValueDesc {
	if v.Loc == LocImm {
		return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(v.Imm.IsNil())}
	}
	if v.Type != JITTypeUnknown {
		return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(v.Type == tagNil)}
	}
	tmp := *v
	ctx.EnsureDesc(&tmp)
	if tmp.Loc != LocRegPair {
		ctx.FreeDesc(&tmp)
		out := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.IsNil), []JITValueDesc{*v}, 1)
		ctx.EmitAndRegImm32(out.Reg, 1)
		out.Type = tagBool
		return out
	}
	tagReg := ctx.AllocReg()
	ctx.emitGetTagRegs(tagReg, tmp.Reg, tmp.Reg2)
	ctx.EmitCmpRegImm8(tagReg, tagNil)
	ctx.EmitSetcc(tagReg, CcE)
	ctx.FreeDesc(&tmp)
	return JITValueDesc{Loc: LocReg, Type: tagBool, Reg: tagReg}
}

func jitBuildIfTail(tail []Scmer) Scmer {
	if len(tail) == 0 {
		return NewNil()
	}
	if len(tail) == 1 {
		return tail[0]
	}
	parts := make([]Scmer, 0, len(tail)+1)
	parts = append(parts, NewSymbol("if"))
	parts = append(parts, tail...)
	return NewSlice(parts)
}

func jitEmitGoVariadicCallFromExprs(ctx *JITContext, fn func(...Scmer) Scmer, argExprs []Scmer, sliceBase Reg, result JITValueDesc) JITValueDesc {
	argc := len(argExprs)
	var argsSlice JITValueDesc
	stackBytes := int32(argc * 16)
	if argc > 0 {
		// Reserve one contiguous frame for all variadic Scmer arguments.
		ctx.EmitSubRSP32(stackBytes)
		if ctx.SliceBaseTracksRSP && ctx.SliceBase != RegRSP {
			ctx.EmitMovRegReg(ctx.SliceBase, RegRSP)
		}
		// Compile each argument directly towards its final stack slot.
		for i := 0; i < len(argExprs); i++ {
			slotOff := int32(i * 16)
			slot := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: slotOff}
			v := jitCompileExpr(ctx, argExprs[i], sliceBase, slot)
			if !(v.Loc == LocStackPair && v.MemPtr == 0 && v.StackOff == slotOff) {
				tmp := JITValueDesc{
					Loc:  LocRegPair,
					Type: JITTypeUnknown,
					Reg:  ctx.AllocReg(),
					Reg2: ctx.AllocReg(),
				}
				_ = jitPlaceIntoPair(ctx, &v, tmp)
				ctx.EmitStoreRegMem(tmp.Reg, RegRSP, slotOff)
				ctx.EmitStoreRegMem(tmp.Reg2, RegRSP, slotOff+8)
				ctx.FreeDesc(&tmp)
			}
			ctx.FreeDesc(&v)
		}
		// argslice: ptr + len (cap = len inside EmitGoCallVariadic).
		argsSlice = JITValueDesc{
			Loc:  LocRegPair,
			Type: JITTypeUnknown,
			Reg:  ctx.AllocReg(),
			Reg2: ctx.AllocReg(),
		}
		ctx.EmitMovRegReg(argsSlice.Reg, RegRSP)
		ctx.EmitMovRegImm64(argsSlice.Reg2, uint64(argc))
		ctx.BindReg(argsSlice.Reg, &argsSlice)
		ctx.BindReg(argsSlice.Reg2, &argsSlice)
	} else {
		argsSlice = JITValueDesc{
			Loc:  LocRegPair,
			Type: JITTypeUnknown,
			Reg:  ctx.AllocReg(),
			Reg2: ctx.AllocReg(),
		}
		ctx.EmitMovRegImm64(argsSlice.Reg, 0)
		ctx.EmitMovRegImm64(argsSlice.Reg2, 0)
		ctx.BindReg(argsSlice.Reg, &argsSlice)
		ctx.BindReg(argsSlice.Reg2, &argsSlice)
	}

	out := ctx.EmitGoCallVariadic(fn, argsSlice, result)
	ctx.FreeDesc(&argsSlice)
	if stackBytes != 0 {
		ctx.EmitAddRSP32(stackBytes)
		if ctx.SliceBaseTracksRSP && ctx.SliceBase != RegRSP {
			ctx.EmitMovRegReg(ctx.SliceBase, RegRSP)
		}
	}
	return out
}

// jitEmitCondJump emits branch code equivalent to Eval(...).Bool():
// jumps to trueLbl when expr is truthy, otherwise to falseLbl.
// It short-circuits nested (and ...)/(or ...)/(if ...) directly without
// forcing intermediate boolean materialization.
func jitEmitCondJump(ctx *JITContext, expr Scmer, sliceBase Reg, trueLbl, falseLbl uint8) {
	if expr.GetTag() == tagSourceInfo {
		expr = expr.SourceInfo().value
	}
	if expr.GetTag() == tagSlice {
		list := expr.Slice()
		if len(list) > 0 && list[0].IsSymbol() {
			switch string(list[0].Symbol()) {
			case "and":
				// Eval semantics: (and) => true
				if len(list) <= 1 {
					ctx.EmitJmp(trueLbl)
					return
				}
				for i := 1; i < len(list)-1; i++ {
					nextLbl := ctx.ReserveLabel()
					jitEmitCondJump(ctx, list[i], sliceBase, nextLbl, falseLbl)
					ctx.MarkLabel(nextLbl)
				}
				jitEmitCondJump(ctx, list[len(list)-1], sliceBase, trueLbl, falseLbl)
				return
			case "or":
				// Eval semantics: (or) => false
				if len(list) <= 1 {
					ctx.EmitJmp(falseLbl)
					return
				}
				for i := 1; i < len(list)-1; i++ {
					nextLbl := ctx.ReserveLabel()
					jitEmitCondJump(ctx, list[i], sliceBase, trueLbl, nextLbl)
					ctx.MarkLabel(nextLbl)
				}
				jitEmitCondJump(ctx, list[len(list)-1], sliceBase, trueLbl, falseLbl)
				return
			case "if":
				// Eval semantics: chain of condition/value pairs plus optional else.
				i := 1
				for i+1 < len(list) {
					thenCondLbl := ctx.ReserveLabel()
					nextCondLbl := ctx.ReserveLabel()
					jitEmitCondJump(ctx, list[i], sliceBase, thenCondLbl, nextCondLbl)
					ctx.MarkLabel(thenCondLbl)
					jitEmitCondJump(ctx, list[i+1], sliceBase, trueLbl, falseLbl)
					ctx.MarkLabel(nextCondLbl)
					i += 2
				}
				if i < len(list) {
					jitEmitCondJump(ctx, list[i], sliceBase, trueLbl, falseLbl)
				} else {
					// No else branch => nil => false
					ctx.EmitJmp(falseLbl)
				}
				return
			}
		}
	}

	cond := jitCompileExpr(ctx, expr, sliceBase, JITValueDesc{Loc: LocAny})
	b := jitCondToBool(ctx, &cond)
	if b.Loc == LocImm {
		if b.Imm.Bool() {
			ctx.EmitJmp(trueLbl)
		} else {
			ctx.EmitJmp(falseLbl)
		}
		return
	}
	ctx.EmitCmpRegImm32(b.Reg, 0)
	ctx.EmitJcc(CcNE, trueLbl)
	ctx.EmitJmp(falseLbl)
	ctx.FreeDesc(&b)
}

// jitCompileExpr recursively compiles a Scheme expression to machine code.
// sliceBase is the GPR holding the variadic args slice pointer.
// result tells the emitter where to place the output.
// Panics on unsupported expressions (caught by jitCompileExprBodyToExec).
func jitCompileExpr(ctx *JITContext, expr Scmer, sliceBase Reg, result JITValueDesc) JITValueDesc {
	if expr.GetTag() == tagSourceInfo {
		expr = expr.SourceInfo().value
	}
	switch expr.GetTag() {
	case tagNil:
		ctx.TrackImm(expr)
		return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: expr}
	case tagBool:
		ctx.TrackImm(expr)
		return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: expr}
	case tagInt:
		ctx.TrackImm(expr)
		return JITValueDesc{Loc: LocImm, Type: tagInt, Imm: expr}
	case tagFloat:
		ctx.TrackImm(expr)
		return JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: expr}
	case tagString:
		ctx.TrackImm(expr)
		return JITValueDesc{Loc: LocImm, Type: tagString, Imm: expr}
	case tagSymbol:
		sym := expr.Symbol()
		if string(sym) == "nil" {
			imm := NewNil()
			ctx.TrackImm(imm)
			return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: imm}
		}
		if ctx.Env != nil {
			if desc, ok := ctx.Env.Lookup(sym); ok {
				if desc.Loc == LocImm {
					ctx.TrackImm(desc.Imm)
				}
				return desc
			}
		}
		if v, ok := Globalenv.Vars[sym]; ok {
			ctx.TrackImm(v)
			return JITValueDesc{Loc: LocImm, Type: v.GetTag(), Imm: v}
		}
		panic("jit: unresolved symbol " + string(sym))
	case tagNthLocalVar:
		// Load parameter: check inline env first (JITEmitProcInline places args here).
		idx := int(expr.NthLocalVar())
		if ctx.Env != nil && idx < len(ctx.Env.Numbered) {
			src := ctx.Env.Numbered[idx]
			if result.Loc == LocRegPair {
				switch src.Loc {
				case LocImm:
					ctx.TrackImm(src.Imm)
					ptr, aux := src.Imm.RawWords()
					ctx.EmitMovRegImm64(result.Reg, uint64(ptr))
					ctx.EmitMovRegImm64(result.Reg2, aux)
					d := JITValueDesc{Loc: LocRegPair, Type: src.Type, Reg: result.Reg, Reg2: result.Reg2}
					ctx.BindReg(result.Reg, &d)
					ctx.BindReg(result.Reg2, &d)
					return d
				case LocRegPair:
					ctx.EnsureDesc(&src)
					if src.Reg != result.Reg {
						ctx.emitMovRegReg(result.Reg, src.Reg)
					}
					if src.Reg2 != result.Reg2 {
						ctx.emitMovRegReg(result.Reg2, src.Reg2)
					}
					d := JITValueDesc{Loc: LocRegPair, Type: src.Type, Reg: result.Reg, Reg2: result.Reg2}
					ctx.BindReg(result.Reg, &d)
					ctx.BindReg(result.Reg2, &d)
					return d
				}
			}
			switch src.Loc {
			case LocImm:
				ctx.TrackImm(src.Imm)
				return src // constants are always safe to alias
			case LocReg:
				// Allocate a fresh register so each use is independently freeable.
				r := ctx.AllocRegExcept(src.Reg)
				ctx.emitMovRegReg(r, src.Reg)
				d := JITValueDesc{Loc: LocReg, Type: src.Type, Reg: r}
				ctx.BindReg(r, &d)
				return d
			case LocRegPair:
				r1 := ctx.AllocRegExcept(src.Reg, src.Reg2)
				r2 := ctx.AllocRegExcept(src.Reg, src.Reg2, r1)
				ctx.emitMovRegReg(r1, src.Reg)
				ctx.emitMovRegReg(r2, src.Reg2)
				d := JITValueDesc{Loc: LocRegPair, Type: src.Type, Reg: r1, Reg2: r2}
				ctx.BindReg(r1, &d)
				ctx.BindReg(r2, &d)
				return d
			}
		}
		if result.Loc == LocRegPair {
			ctx.EmitLoadArgPair(result.Reg, result.Reg2, sliceBase, idx)
			d := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: result.Reg, Reg2: result.Reg2}
			ctx.BindReg(result.Reg, &d)
			ctx.BindReg(result.Reg2, &d)
			return d
		}
		// Fallback: load from args slice: ptr at [base+i*16], aux at [base+i*16+8]
		ptrReg := ctx.AllocReg()
		auxReg := ctx.AllocReg()
		ctx.emitMovRegMem(ptrReg, sliceBase, int32(idx*16))
		ctx.emitMovRegMem(auxReg, sliceBase, int32(idx*16+8))
		d := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ptrReg, Reg2: auxReg}
		ctx.BindReg(ptrReg, &d)
		ctx.BindReg(auxReg, &d)
		return d
	case tagSlice:
		list := expr.Slice()
		if len(list) == 0 {
			imm := NewNil()
			ctx.TrackImm(imm)
			return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: imm}
		}
		// Resolve operator
		if !list[0].IsSymbol() {
			panic("jit: non-symbol in call position")
		}
		name := string(list[0].Symbol())
		switch name {
		case "quote":
			if len(list) < 2 {
				imm := NewNil()
				ctx.TrackImm(imm)
				return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: imm}
			}
			q := list[1]
			if q.GetTag() == tagSourceInfo {
				q = q.SourceInfo().value
			}
			ctx.TrackImm(q)
			return JITValueDesc{Loc: LocImm, Type: q.GetTag(), Imm: q}
		case "if":
			if len(list) < 3 {
				imm := NewNil()
				ctx.TrackImm(imm)
				return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: imm}
			}
			target := jitEnsureResultPair(ctx, result)
			var endLbl uint8
			hasDynamic := false
			i := 1
			for i+1 < len(list) {
				cond := jitCompileExpr(ctx, list[i], sliceBase, JITValueDesc{Loc: LocAny})
				b := jitCondToBool(ctx, &cond)
				if b.Loc == LocImm {
					if b.Imm.Bool() {
						thenVal := jitCompileExpr(ctx, list[i+1], sliceBase, target)
						_ = jitPlaceIntoPair(ctx, &thenVal, target)
						if hasDynamic {
							ctx.MarkLabel(endLbl)
						}
						ctx.BindReg(target.Reg, &target)
						ctx.BindReg(target.Reg2, &target)
						return target
					}
					i += 2
					continue
				}
				if !hasDynamic {
					endLbl = ctx.ReserveLabel()
					hasDynamic = true
				}
				nextCondLbl := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(b.Reg, 0)
				ctx.EmitJcc(CcE, nextCondLbl)
				ctx.FreeDesc(&b)
				thenVal := jitCompileExpr(ctx, list[i+1], sliceBase, target)
				_ = jitPlaceIntoPair(ctx, &thenVal, target)
				ctx.EmitJmp(endLbl)
				ctx.MarkLabel(nextCondLbl)
				i += 2
			}
			if i < len(list) {
				elseVal := jitCompileExpr(ctx, list[i], sliceBase, target)
				_ = jitPlaceIntoPair(ctx, &elseVal, target)
			} else {
				nilDesc := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
				_ = jitPlaceIntoPair(ctx, &nilDesc, target)
			}
			if hasDynamic {
				ctx.MarkLabel(endLbl)
			}
			ctx.BindReg(target.Reg, &target)
			ctx.BindReg(target.Reg2, &target)
			return target
		case "and":
			if len(list) <= 1 {
				imm := NewBool(true)
				ctx.TrackImm(imm)
				return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: imm}
			}
			target := jitEnsureResultPair(ctx, result)
			var falseLbl uint8
			var endLbl uint8
			hasDynamic := false
			compileTimeFalse := false
			for i := 1; i < len(list); i++ {
				c := jitCompileExpr(ctx, list[i], sliceBase, JITValueDesc{Loc: LocAny})
				b := jitCondToBool(ctx, &c)
				if b.Loc == LocImm {
					if !b.Imm.Bool() {
						compileTimeFalse = true
						break
					}
					continue
				}
				if !hasDynamic {
					falseLbl = ctx.ReserveLabel()
					endLbl = ctx.ReserveLabel()
					hasDynamic = true
				}
				ctx.EmitCmpRegImm32(b.Reg, 0)
				ctx.EmitJcc(CcE, falseLbl)
				ctx.FreeDesc(&b)
			}
			if compileTimeFalse {
				if hasDynamic {
					ctx.MarkLabel(falseLbl)
				}
				falseDesc := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				_ = jitPlaceIntoPair(ctx, &falseDesc, target)
				ctx.BindReg(target.Reg, &target)
				ctx.BindReg(target.Reg2, &target)
				return target
			}
			if !hasDynamic {
				trueDesc := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
				_ = jitPlaceIntoPair(ctx, &trueDesc, target)
				ctx.BindReg(target.Reg, &target)
				ctx.BindReg(target.Reg2, &target)
				return target
			}
			trueDesc := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
			_ = jitPlaceIntoPair(ctx, &trueDesc, target)
			ctx.EmitJmp(endLbl)
			ctx.MarkLabel(falseLbl)
			falseDesc := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
			_ = jitPlaceIntoPair(ctx, &falseDesc, target)
			ctx.MarkLabel(endLbl)
			ctx.BindReg(target.Reg, &target)
			ctx.BindReg(target.Reg2, &target)
			return target
		case "or":
			if len(list) <= 1 {
				imm := NewBool(false)
				ctx.TrackImm(imm)
				return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: imm}
			}
			target := jitEnsureResultPair(ctx, result)
			var trueLbl uint8
			var endLbl uint8
			hasDynamic := false
			compileTimeTrue := false
			for i := 1; i < len(list); i++ {
				c := jitCompileExpr(ctx, list[i], sliceBase, JITValueDesc{Loc: LocAny})
				b := jitCondToBool(ctx, &c)
				if b.Loc == LocImm {
					if b.Imm.Bool() {
						compileTimeTrue = true
						break
					}
					continue
				}
				if !hasDynamic {
					trueLbl = ctx.ReserveLabel()
					endLbl = ctx.ReserveLabel()
					hasDynamic = true
				}
				ctx.EmitCmpRegImm32(b.Reg, 0)
				ctx.EmitJcc(CcNE, trueLbl)
				ctx.FreeDesc(&b)
			}
			if compileTimeTrue {
				if hasDynamic {
					ctx.MarkLabel(trueLbl)
				}
				trueDesc := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
				_ = jitPlaceIntoPair(ctx, &trueDesc, target)
				ctx.BindReg(target.Reg, &target)
				ctx.BindReg(target.Reg2, &target)
				return target
			}
			if !hasDynamic {
				falseDesc := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				_ = jitPlaceIntoPair(ctx, &falseDesc, target)
				ctx.BindReg(target.Reg, &target)
				ctx.BindReg(target.Reg2, &target)
				return target
			}
			falseDesc := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
			_ = jitPlaceIntoPair(ctx, &falseDesc, target)
			ctx.EmitJmp(endLbl)
			ctx.MarkLabel(trueLbl)
			trueDesc := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
			_ = jitPlaceIntoPair(ctx, &trueDesc, target)
			ctx.MarkLabel(endLbl)
			ctx.BindReg(target.Reg, &target)
			ctx.BindReg(target.Reg2, &target)
			return target
		case "coalesce":
			// Eval semantics:
			// return first truthy value; if none truthy, return last value; empty => nil.
			if len(list) <= 1 {
				imm := NewNil()
				ctx.TrackImm(imm)
				return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: imm}
			}
			target := jitEnsureResultPair(ctx, result)
			endLbl := ctx.ReserveLabel()
			for i := 1; i < len(list); i++ {
				v := jitCompileExpr(ctx, list[i], sliceBase, JITValueDesc{Loc: LocAny})
				if i == len(list)-1 {
					_ = jitPlaceIntoPair(ctx, &v, target)
					break
				}
				if v.Loc == LocImm {
					if v.Imm.Bool() {
						_ = jitPlaceIntoPair(ctx, &v, target)
						ctx.EmitJmp(endLbl)
						break
					}
					continue
				}
				b := jitCondToBoolBorrowed(ctx, &v)
				if b.Loc == LocImm {
					if b.Imm.Bool() {
						_ = jitPlaceIntoPair(ctx, &v, target)
						ctx.EmitJmp(endLbl)
					}
					ctx.FreeDesc(&v)
					continue
				}
				takeLbl := ctx.ReserveLabel()
				nextLbl := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(b.Reg, 0)
				ctx.EmitJcc(CcNE, takeLbl)
				ctx.EmitJmp(nextLbl)
				ctx.MarkLabel(takeLbl)
				_ = jitPlaceIntoPair(ctx, &v, target)
				ctx.EmitJmp(endLbl)
				ctx.MarkLabel(nextLbl)
				ctx.FreeDesc(&b)
				ctx.FreeDesc(&v)
			}
			ctx.MarkLabel(endLbl)
			ctx.BindReg(target.Reg, &target)
			ctx.BindReg(target.Reg2, &target)
			return target
		case "coalesceNil":
			// Eval semantics:
			// return first non-nil value among args; empty => nil.
			if len(list) <= 1 {
				imm := NewNil()
				ctx.TrackImm(imm)
				return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: imm}
			}
			target := jitEnsureResultPair(ctx, result)
			endLbl := ctx.ReserveLabel()
			for i := 1; i < len(list); i++ {
				v := jitCompileExpr(ctx, list[i], sliceBase, JITValueDesc{Loc: LocAny})
				if v.Loc == LocImm {
					if !v.Imm.IsNil() {
						_ = jitPlaceIntoPair(ctx, &v, target)
						ctx.EmitJmp(endLbl)
						break
					}
					continue
				}
				isNil := jitIsNilBorrowed(ctx, &v)
				if isNil.Loc == LocImm {
					if !isNil.Imm.Bool() {
						_ = jitPlaceIntoPair(ctx, &v, target)
						ctx.EmitJmp(endLbl)
					}
					ctx.FreeDesc(&v)
					continue
				}
				takeLbl := ctx.ReserveLabel()
				nextLbl := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(isNil.Reg, 0)
				ctx.EmitJcc(CcE, takeLbl) // isNil == 0 => take value
				ctx.EmitJmp(nextLbl)
				ctx.MarkLabel(takeLbl)
				_ = jitPlaceIntoPair(ctx, &v, target)
				ctx.EmitJmp(endLbl)
				ctx.MarkLabel(nextLbl)
				ctx.FreeDesc(&isNil)
				ctx.FreeDesc(&v)
			}
			ctx.MarkLabel(endLbl)
			ctx.BindReg(target.Reg, &target)
			ctx.BindReg(target.Reg2, &target)
			return target
		case "lambda":
			if len(list) < 3 {
				panic("jit: lambda expects params and body")
			}
			params := list[1]
			if params.IsSourceInfo() {
				params = params.SourceInfo().value
			}
			body := list[2]
			numVars := 0
			if len(list) > 3 {
				numVars = int(ToInt(list[3]))
			}

			// Build variadic builder args:
			// [params, body, numVars, key1, val1, ...]
			// keys are quoted Symbol or quoted NthLocalVar.
			argExprs := make([]Scmer, 0, 16)
			argExprs = append(argExprs, NewSlice([]Scmer{NewSymbol("quote"), params}))
			argExprs = append(argExprs, NewSlice([]Scmer{NewSymbol("quote"), body}))
			argExprs = append(argExprs, NewInt(int64(numVars)))

			// Capture non-global free symbol variables from current lexical scope.
			for _, sym := range jitLambdaFreeSymbols(params, body) {
				if ctx.Env != nil {
					if d, ok := ctx.Env.Lookup(sym); ok {
						_ = d
						argExprs = append(argExprs, NewSlice([]Scmer{NewSymbol("quote"), NewSymbol(string(sym))}))
						argExprs = append(argExprs, NewSymbol(string(sym)))
						continue
					}
				}
				// Globals are resolved through closure Outer env.
				if _, ok := Globalenv.Vars[sym]; ok {
					continue
				}
				// Leave unresolved symbols late-bound via closure Outer env.
			}

			// Capture optimized outer(var i) references as numbered captures.
			for _, idx := range jitLambdaOuterVarIndices(body) {
				key := NewNthLocalVar(idx)
				argExprs = append(argExprs, NewSlice([]Scmer{NewSymbol("quote"), key}))
				argExprs = append(argExprs, key)
			}

			return jitEmitGoVariadicCallFromExprs(ctx, jitBuildLambdaClosure, argExprs, sliceBase, result)
		}
		decl, ok := declarations[name]
		if !ok {
			panic("jit: unknown callable " + name)
		}
		if decl.JITEmit != nil {
			// Multiplication has deep internal loops/phis and is sensitive to
			// mutable register-backed argument descriptors; materialize its args
			// to stable stack slots first.
			if name == "*" {
				n := len(list) - 1
				args := make([]JITValueDesc, n)
				rawArgs := make([]JITValueDesc, n)
				protectedRegs := make([]Reg, 0, len(list)*2)
				for i := 1; i < len(list); i++ {
					v := jitCompileExpr(ctx, list[i], sliceBase, JITValueDesc{Loc: LocAny})
					ctx.EnsureDesc(&v)
					rawArgs[i-1] = v
					switch v.Loc {
					case LocReg:
						ctx.BindReg(v.Reg, &rawArgs[i-1])
						ctx.ProtectReg(v.Reg)
						protectedRegs = append(protectedRegs, v.Reg)
					case LocRegPair:
						ctx.BindReg(v.Reg, &rawArgs[i-1])
						ctx.BindReg(v.Reg2, &rawArgs[i-1])
						ctx.ProtectReg(v.Reg)
						ctx.ProtectReg(v.Reg2)
						protectedRegs = append(protectedRegs, v.Reg, v.Reg2)
					}
				}
				stackBytes := int32(n * 16)
				if n > 0 {
					ctx.EmitSubRSP32(stackBytes)
					for i, v := range rawArgs {
						slotOff := int32(i * 16)
						tmp := JITValueDesc{
							Loc:  LocRegPair,
							Type: JITTypeUnknown,
							Reg:  ctx.AllocReg(),
							Reg2: ctx.AllocReg(),
						}
						_ = jitPlaceIntoPair(ctx, &v, tmp)
						ctx.EmitStoreRegMem(tmp.Reg, RegRSP, slotOff)
						ctx.EmitStoreRegMem(tmp.Reg2, RegRSP, slotOff+8)
						ctx.FreeDesc(&tmp)
						args[i] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: slotOff}
					}
				}
				for _, r := range protectedRegs {
					ctx.UnprotectReg(r)
				}
				for i := range rawArgs {
					ctx.FreeDesc(&rawArgs[i])
				}
				out := decl.JITEmit(ctx, args, result)
				if n > 0 {
					if out.MemPtr == 0 && (out.Loc == LocStack || out.Loc == LocStackPair) {
						ctx.EnsureDesc(&out)
					}
					ctx.EmitAddRSP32(stackBytes)
					if ctx.SliceBaseTracksRSP && ctx.SliceBase != RegRSP {
						ctx.EmitMovRegReg(ctx.SliceBase, RegRSP)
					}
				}
				if out.Loc == LocImm {
					ctx.TrackImm(out.Imm)
				}
				return out
			}

			// Compile arguments (intermediate results use LocAny).
			// Use a stack-allocated buffer for the common case of <=8 args;
			// fall back to heap allocation for larger expressions.
			var argsBuf [8]JITValueDesc
			n := len(list) - 1
			var args []JITValueDesc
			if n <= len(argsBuf) {
				args = argsBuf[:n]
			} else {
				args = make([]JITValueDesc, n)
			}
			protectedRegs := make([]Reg, 0, len(list)*2)
			for i := 1; i < len(list); i++ {
				args[i-1] = jitCompileExpr(ctx, list[i], sliceBase, JITValueDesc{Loc: LocAny})
				switch args[i-1].Loc {
				case LocReg:
					ctx.BindReg(args[i-1].Reg, &args[i-1])
					ctx.ProtectReg(args[i-1].Reg)
					protectedRegs = append(protectedRegs, args[i-1].Reg)
				case LocRegPair:
					ctx.BindReg(args[i-1].Reg, &args[i-1])
					ctx.BindReg(args[i-1].Reg2, &args[i-1])
					ctx.ProtectReg(args[i-1].Reg)
					ctx.ProtectReg(args[i-1].Reg2)
					protectedRegs = append(protectedRegs, args[i-1].Reg, args[i-1].Reg2)
				}
			}
			defer func() {
				for _, r := range protectedRegs {
					ctx.UnprotectReg(r)
				}
			}()
			out := decl.JITEmit(ctx, args, result)
			if out.Loc == LocImm {
				ctx.TrackImm(out.Imm)
			}
			return out
		}
		if decl.Fn == nil {
			panic("jit: no JITEmit/Fn for " + name)
		}

		argc := len(list) - 1
		var argsSlice JITValueDesc
		stackBytes := int32(argc * 16)
		if argc > 0 {
			// Reserve one contiguous frame for all variadic Scmer arguments.
			ctx.EmitSubRSP32(stackBytes)
			if ctx.SliceBaseTracksRSP && ctx.SliceBase != RegRSP {
				ctx.EmitMovRegReg(ctx.SliceBase, RegRSP)
			}
			// Compile each argument directly towards its final stack slot.
			for i := 1; i < len(list); i++ {
				slotOff := int32((i - 1) * 16)
				slot := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: slotOff}
				v := jitCompileExpr(ctx, list[i], sliceBase, slot)
				if !(v.Loc == LocStackPair && v.MemPtr == 0 && v.StackOff == slotOff) {
					tmp := JITValueDesc{
						Loc:  LocRegPair,
						Type: JITTypeUnknown,
						Reg:  ctx.AllocReg(),
						Reg2: ctx.AllocReg(),
					}
					_ = jitPlaceIntoPair(ctx, &v, tmp)
					ctx.EmitStoreRegMem(tmp.Reg, RegRSP, slotOff)
					ctx.EmitStoreRegMem(tmp.Reg2, RegRSP, slotOff+8)
					ctx.FreeDesc(&tmp)
				}
				ctx.FreeDesc(&v)
			}
			// argslice: ptr + len (cap = len inside EmitGoCallVariadic).
			argsSlice = JITValueDesc{
				Loc:  LocRegPair,
				Type: JITTypeUnknown,
				Reg:  ctx.AllocReg(),
				Reg2: ctx.AllocReg(),
			}
			ctx.EmitMovRegReg(argsSlice.Reg, RegRSP)
			ctx.EmitMovRegImm64(argsSlice.Reg2, uint64(argc))
			ctx.BindReg(argsSlice.Reg, &argsSlice)
			ctx.BindReg(argsSlice.Reg2, &argsSlice)
		} else {
			argsSlice = JITValueDesc{
				Loc:  LocRegPair,
				Type: JITTypeUnknown,
				Reg:  ctx.AllocReg(),
				Reg2: ctx.AllocReg(),
			}
			ctx.EmitMovRegImm64(argsSlice.Reg, 0)
			ctx.EmitMovRegImm64(argsSlice.Reg2, 0)
			ctx.BindReg(argsSlice.Reg, &argsSlice)
			ctx.BindReg(argsSlice.Reg2, &argsSlice)
		}

		out := ctx.EmitGoCallVariadic(decl.Fn, argsSlice, result)
		ctx.FreeDesc(&argsSlice)
		if stackBytes != 0 {
			ctx.EmitAddRSP32(stackBytes)
			if ctx.SliceBaseTracksRSP && ctx.SliceBase != RegRSP {
				ctx.EmitMovRegReg(ctx.SliceBase, RegRSP)
			}
		}
		return out
	default:
		panic(fmt.Sprintf("jit: unsupported expression tag=%d expr=%s", expr.GetTag(), SerializeToString(expr, &Globalenv)))
	}
}

// JITEmitProcInline emits a Proc's body inline into the current JIT stream.
// args[i] provides the pre-placed descriptor for the i-th parameter (NthLocalVar(i)).
// Each NthLocalVar reference emits a fresh register copy so the descriptor is
// independently freeable per use site (safe for expressions that reference a
// parameter more than once).
// sliceBase is passed through to jitCompileExpr for any fallback slice-based
// NthLocalVar loads (in practice not reached when all params are in args).
// Panics on any un-emittable sub-expression — callers should recover and fall back.
func JITEmitProcInline(ctx *JITContext, proc *Proc, args []JITValueDesc, sliceBase Reg, result JITValueDesc) JITValueDesc {
	innerEnv := &JITEnv{
		Numbered: args,
		Outer:    ctx.Env,
	}
	oldEnv := ctx.Env
	ctx.Env = innerEnv
	defer func() { ctx.Env = oldEnv }()

	body := proc.Body
	if body.GetTag() == tagSourceInfo {
		body = body.SourceInfo().value
	}
	return jitCompileExpr(ctx, body, sliceBase, result)
}

/* TODO: peephole optimizer:
- remove argument checks (test rbx,rbx 48 85 db 76 xx)
- shorten immediate values
- constant-fold operations
- inline functions
- jump to other functions
*/
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
	RegX0 Reg = 16
	RegX1 Reg = 17
	RegX2 Reg = 18
	RegX3 Reg = 19
	RegX4 Reg = 20
	RegX5 Reg = 21
)

// emitByte appends a single byte to the writer.
func (ctx *JITContext) ensureSpace(n uintptr) {
	if uintptr(ctx.Ptr)+n > uintptr(ctx.End) {
		panic(jitCodeOverflowPanic)
	}
}

func (ctx *JITContext) emitByte(b byte) {
	ctx.ensureSpace(1)
	*(*byte)(ctx.Ptr) = b
	ctx.Ptr = unsafe.Add(ctx.Ptr, 1)
}

// emitBytes appends raw bytes to the writer.
func (ctx *JITContext) emitBytes(bs ...byte) {
	ctx.ensureSpace(uintptr(len(bs)))
	for _, b := range bs {
		*(*byte)(ctx.Ptr) = b
		ctx.Ptr = unsafe.Add(ctx.Ptr, 1)
	}
}

// emitU32 appends a little-endian uint32.
func (ctx *JITContext) emitU32(v uint32) {
	ctx.ensureSpace(4)
	*(*uint32)(ctx.Ptr) = v
	ctx.Ptr = unsafe.Add(ctx.Ptr, 4)
}

// emitU64 appends a little-endian uint64.
func (ctx *JITContext) emitU64(v uint64) {
	ctx.ensureSpace(8)
	*(*uint64)(ctx.Ptr) = v
	ctx.Ptr = unsafe.Add(ctx.Ptr, 8)
}

// --- Return emitters ---

// EmitReturnInt emits: MOV RAX, &scmerIntSentinel; MOV RBX, value; RET
// Constructs NewInt(value) in the return registers.
func (ctx *JITContext) EmitReturnInt(src JITValueDesc) {
	// MOV RAX, imm64 (address of scmerIntSentinel)
	ctx.emitBytes(0x48, 0xB8)
	ctx.emitU64(uint64(uintptr(unsafe.Pointer(&scmerIntSentinel))))
	switch src.Loc {
	case LocReg:
		if src.Reg != RegRBX {
			// MOV RBX, src.Reg
			ctx.emitMovRegReg(RegRBX, src.Reg)
		}
	case LocImm:
		// MOV RBX, imm64
		ctx.emitBytes(0x48, 0xBB)
		ctx.emitU64(uint64(src.Imm.Int()))
	}
	ctx.emitByte(0xC3) // RET
}

// EmitReturnFloat emits: MOV RAX, &scmerFloatSentinel; MOVQ XMM→RBX; RET
// Constructs NewFloat(value) in the return registers.
func (ctx *JITContext) EmitReturnFloat(src JITValueDesc) {
	// MOV RAX, imm64 (address of scmerFloatSentinel)
	ctx.emitBytes(0x48, 0xB8)
	ctx.emitU64(uint64(uintptr(unsafe.Pointer(&scmerFloatSentinel))))
	switch src.Loc {
	case LocReg:
		// MOVQ XMM -> RBX: 66 48 0F 7E C3 (for X0→RBX)
		ctx.emitMovqXmmToGpr(RegRBX, src.Reg)
	case LocImm:
		// MOV RBX, imm64 (raw float bits)
		ctx.emitBytes(0x48, 0xBB)
		ctx.emitU64(math.Float64bits(src.Imm.Float()))
	}
	ctx.emitByte(0xC3) // RET
}

// EmitReturnNil emits: XOR EAX,EAX; XOR EBX,EBX; RET
func (ctx *JITContext) EmitReturnNil() {
	ctx.emitBytes(
		0x31, 0xC0, // XOR EAX, EAX
		0x31, 0xDB, // XOR EBX, EBX
		0xC3, // RET
	)
}

// EmitReturnBool emits: XOR EAX,EAX; MOV RBX, makeAux(tagBool, 0/1); RET
func (ctx *JITContext) EmitReturnBool(src JITValueDesc) {
	ctx.emitBytes(0x31, 0xC0) // XOR EAX, EAX (ptr = nil for bool)
	switch src.Loc {
	case LocImm:
		var val uint64
		if src.Imm.Bool() {
			val = 1
		}
		aux := makeAux(tagBool, val)
		ctx.emitBytes(0x48, 0xBB) // MOV RBX, imm64
		ctx.emitU64(aux)
	case LocReg:
		// Build aux = (bool&1)<<8 | tagBool.
		// Keep it branchless so callers can feed arbitrary integer predicates.
		// First zero-extend the bool into RBX.
		ctx.emitMovRegReg(RegRBX, src.Reg)
		ctx.emitBytes(0x48, 0x81, 0xE3) // AND RBX, 0x01
		ctx.emitU32(1)
		ctx.EmitShlRegImm8(RegRBX, 8)
		// MOV RCX, tagBool
		ctx.emitBytes(0x48, 0xB9) // MOV RCX, imm64
		ctx.emitU64(uint64(tagBool))
		// OR RBX, RCX
		ctx.emitBytes(0x48, 0x09, 0xCB)
	}
	ctx.emitByte(0xC3) // RET
}

// --- Scmer construction emitters (no RET) ---

// EmitMakeBool constructs a Scmer bool into dst.Reg (ptr) and dst.Reg2 (aux).
// src.Reg holds the 0/1 boolean value.
func (ctx *JITContext) EmitMakeBool(dst JITValueDesc, src JITValueDesc) {
	// dst.Reg = nil (XOR reg, reg)
	ctx.emitXorReg(dst.Reg)
	switch src.Loc {
	case LocImm:
		var bval uint64
		if src.Imm.Bool() {
			bval = 1
		}
		aux := makeAux(tagBool, bval)
		ctx.EmitMovRegImm64(dst.Reg2, aux)
	case LocReg:
		// dst.Reg2 = ((src.Reg & 1) << 8) | tagBool
		if dst.Reg2 != src.Reg {
			ctx.emitMovRegReg(dst.Reg2, src.Reg)
		}
		ctx.emitAndRegImm32(dst.Reg2, 1)
		ctx.EmitShlRegImm8(dst.Reg2, 8)
		ctx.EmitMovRegImm64(RegR11, uint64(tagBool))
		ctx.emitOrRegReg(dst.Reg2, RegR11)
	}
}

// EmitMakeInt constructs a Scmer int into dst.Reg (ptr) and dst.Reg2 (aux).
// src.Reg holds the int64 value.
func (ctx *JITContext) EmitMakeInt(dst JITValueDesc, src JITValueDesc) {
	ctx.EmitMovRegImm64(dst.Reg, uint64(uintptr(unsafe.Pointer(&scmerIntSentinel))))
	switch src.Loc {
	case LocReg:
		if dst.Reg2 != src.Reg {
			ctx.emitMovRegReg(dst.Reg2, src.Reg)
		}
	case LocImm:
		ctx.EmitMovRegImm64(dst.Reg2, uint64(src.Imm.Int()))
	}
}

// EmitMakeFloat constructs a Scmer float into dst.Reg (ptr) and dst.Reg2 (aux).
// src.Reg holds the float64 bits as uint64.
func (ctx *JITContext) EmitMakeFloat(dst JITValueDesc, src JITValueDesc) {
	ctx.EmitMovRegImm64(dst.Reg, uint64(uintptr(unsafe.Pointer(&scmerFloatSentinel))))
	switch src.Loc {
	case LocReg:
		if dst.Reg2 != src.Reg {
			ctx.emitMovRegReg(dst.Reg2, src.Reg)
		}
	case LocImm:
		ctx.EmitMovRegImm64(dst.Reg2, math.Float64bits(src.Imm.Float())) // float bits stored in aux
	}
}

// EmitMakeNil constructs a Scmer nil into dst.Reg (ptr) and dst.Reg2 (aux).
func (ctx *JITContext) EmitMakeNil(dst JITValueDesc) {
	ctx.emitXorReg(dst.Reg)
	ctx.emitXorReg(dst.Reg2)
}

// emitXorReg emits XOR r32, r32 (zeros 64-bit register via 32-bit op)
func (ctx *JITContext) emitXorReg(r Reg) {
	if r >= 8 {
		ctx.emitBytes(0x45, 0x31, byte(0xC0|(byte(r&7)<<3)|byte(r&7)))
	} else {
		ctx.emitBytes(0x31, byte(0xC0|(byte(r)<<3)|byte(r)))
	}
}

// emitAndRegImm32 emits AND r64, sign-extended imm32
func (ctx *JITContext) emitAndRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01
	}
	modrm := byte(0xE0) | byte(dst&7) // /4 = AND
	ctx.emitBytes(rex, 0x81, modrm)
	ctx.emitU32(uint32(imm))
}

// emitOrRegReg emits OR dst, src (64-bit)
func (ctx *JITContext) emitOrRegReg(dst, src Reg) {
	ctx.emitAluRegReg(0x09, dst, src) // OR r/m64, r64
}

// --- ALU emitters (type-specialized) ---

// EmitAddInt64 emits: ADD dst, src (GPR += GPR)
func (ctx *JITContext) EmitAddInt64(dst, src Reg) {
	ctx.emitAluRegReg(0x01, dst, src) // ADD r/m64, r64
}

// EmitSubInt64 emits: SUB dst, src (GPR -= GPR)
func (ctx *JITContext) EmitSubInt64(dst, src Reg) {
	ctx.emitAluRegReg(0x29, dst, src) // SUB r/m64, r64
}

// EmitImulInt64 emits: IMUL dst, src (GPR *= GPR, signed)
func (ctx *JITContext) EmitImulInt64(dst, src Reg) {
	// IMUL dst, src: REX.W + 0F AF /r (dst = dst * src)
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x04 // REX.R
	}
	if src >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | (byte(dst&7) << 3) | byte(src&7)
	ctx.emitBytes(rex, 0x0F, 0xAF, modrm)
}

// EmitAddFloat64 emits: ADDSD dst, src (XMM += XMM)
func (ctx *JITContext) EmitAddFloat64(dst, src Reg) {
	ctx.emitMovqGprToXmm(RegX0, dst)
	ctx.emitMovqGprToXmm(RegX1, src)
	ctx.emitSseOp(0x58, RegX0, RegX1) // ADDSD
	ctx.emitMovqXmmToGpr(dst, RegX0)
}

// EmitSubFloat64 emits: SUBSD dst, src (XMM -= XMM)
func (ctx *JITContext) EmitSubFloat64(dst, src Reg) {
	ctx.emitMovqGprToXmm(RegX0, dst)
	ctx.emitMovqGprToXmm(RegX1, src)
	ctx.emitSseOp(0x5C, RegX0, RegX1) // SUBSD
	ctx.emitMovqXmmToGpr(dst, RegX0)
}

// EmitMulFloat64 emits: MULSD dst, src (XMM *= XMM)
func (ctx *JITContext) EmitMulFloat64(dst, src Reg) {
	ctx.emitMovqGprToXmm(RegX0, dst)
	ctx.emitMovqGprToXmm(RegX1, src)
	ctx.emitSseOp(0x59, RegX0, RegX1) // MULSD
	ctx.emitMovqXmmToGpr(dst, RegX0)
}

// EmitDivFloat64 emits: DIVSD dst, src (XMM /= XMM)
func (ctx *JITContext) EmitDivFloat64(dst, src Reg) {
	ctx.emitMovqGprToXmm(RegX0, dst)
	ctx.emitMovqGprToXmm(RegX1, src)
	ctx.emitSseOp(0x5E, RegX0, RegX1) // DIVSD
	ctx.emitMovqXmmToGpr(dst, RegX0)
}

// EmitCmpFloat64Setcc compares two float64 bit-patterns from GPRs and writes
// 0/1 into dst using SETcc on the floating-point flags.
func (ctx *JITContext) EmitCmpFloat64Setcc(dst, left, right Reg, cc byte) {
	// UCOMISD sets CF/ZF/PF semantics; map signed integer CCs used by generic
	// lowering to their unordered/unsigned floating-point equivalents.
	switch cc {
	case CcL:
		cc = CcB
	case CcLE:
		cc = CcBE
	case CcG:
		cc = CcA
	case CcGE:
		cc = CcAE
	}
	ctx.emitMovqGprToXmm(RegX0, left)
	ctx.emitMovqGprToXmm(RegX1, right)
	// UCOMISD XMM0, XMM1
	ctx.emitBytes(0x66, 0x0F, 0x2E, 0xC1)
	ctx.EmitSetcc(dst, cc)
}

// --- Conversion emitters ---

// EmitCvtInt64ToFloat64 converts an int64 in gprSrc to float64 bits in gprSrc.
// Uses the XMM register corresponding to gprSrc as scratch:
//
//	CVTSI2SDQ xmm(gprSrc), gprSrc   — int64 → float64 in XMM
//	MOVQ      gprSrc, xmm(gprSrc)   — extract float64 bits back to GPR
func (ctx *JITContext) EmitCvtInt64ToFloat64(xmmDst, gprSrc Reg) {
	xmm := xmmDst - 16 // convert to XMM index (unsigned underflow is fine)
	rex := byte(0x48)
	if xmm >= 8 {
		rex |= 0x04 // REX.R
	}
	if gprSrc >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | (byte(xmm&7) << 3) | byte(gprSrc&7)
	// CVTSI2SDQ xmm, gpr (int64 → float64 in XMM)
	ctx.emitBytes(0xF2, rex, 0x0F, 0x2A, modrm)
	// MOVQ xmm → gpr (66 REX.W 0F 7E /r) — extract float64 bits to GPR
	ctx.emitBytes(0x66, rex, 0x0F, 0x7E, modrm)
}

// EmitCvtFloatBitsToInt64 converts raw float64 bits in gprSrc to int64 in dst.
// Uses XMM0 as scratch:
//
//	MOVQ XMM0, gprSrc
//	CVTTSD2SI dst, XMM0
func (ctx *JITContext) EmitCvtFloatBitsToInt64(dst, gprSrc Reg) {
	ctx.emitMovqGprToXmm(RegX0, gprSrc)
	xmm := RegX0 - 16
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x04 // REX.R
	}
	if xmm >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | (byte(dst&7) << 3) | byte(xmm&7)
	ctx.emitBytes(0xF2, rex, 0x0F, 0x2C, modrm)
}

// EmitXorpdReg emits: XORPD xmm, xmm (zero a float register)
func (ctx *JITContext) EmitXorpdReg(xmm Reg) {
	r := xmm - 16
	modrm := byte(0xC0) | (byte(r&7) << 3) | byte(r&7)
	if r >= 8 {
		ctx.emitBytes(0x66, 0x45, 0x0F, 0x57, modrm)
	} else {
		ctx.emitBytes(0x66, 0x0F, 0x57, modrm)
	}
}

// --- Load emitters ---

// EmitLoadArgInt64 emits code to load the int64 value of the idx-th variadic
// arg directly from the Scmer slice. Only valid when JIT type = int64.
// Loads a[idx].aux (which IS the raw int64) into dstReg.
// sliceBase is the GPR holding the slice pointer.
func (ctx *JITContext) EmitLoadArgInt64(dst, sliceBase Reg, idx int) {
	// MOV dst, [sliceBase + idx*16 + 8]  (aux field)
	ctx.emitMovRegMem(dst, sliceBase, int32(idx*16+8))
}

// EmitLoadArgFloat64 emits code to load the float64 value of the idx-th arg.
// Only valid when JIT type = float64.
// Loads a[idx].aux bits into xmmDst via MOVQ.
func (ctx *JITContext) EmitLoadArgFloat64(xmmDst, sliceBase Reg, idx int) {
	// MOVQ xmm, [sliceBase + idx*16 + 8]
	ctx.emitMovqMemToXmm(xmmDst, sliceBase, int32(idx*16+8))
}

// EmitLoadArgPair loads the idx-th Scmer (ptr+aux pair) from the args slice.
func (ctx *JITContext) EmitLoadArgPair(dstPtr, dstAux, sliceBase Reg, idx int) {
	ctx.emitMovRegMem(dstPtr, sliceBase, int32(idx*16))   // ptr field
	ctx.emitMovRegMem(dstAux, sliceBase, int32(idx*16+8)) // aux field
}

// EmitByte emits a single byte (exported for test harnesses).
func (ctx *JITContext) EmitByte(b byte) {
	ctx.emitByte(b)
}

// --- Compare emitters ---

// EmitCmpInt64 emits: CMP reg1, reg2
func (ctx *JITContext) EmitCmpInt64(a, b Reg) {
	ctx.emitAluRegReg(0x39, a, b) // CMP r/m64, r64
}

// EmitJcc emits a conditional jump with a rel32 fixup.
func (ctx *JITContext) EmitJcc(cc byte, labelID uint8) {
	ctx.emitBytes(0x0F, 0x80|cc) // Jcc rel32
	ctx.AddFixup(labelID, 4, true)
	ctx.emitU32(0) // placeholder
}

// EmitJmp emits an unconditional JMP rel32.
func (ctx *JITContext) EmitJmp(labelID uint8) {
	ctx.emitByte(0xE9) // JMP rel32
	ctx.AddFixup(labelID, 4, true)
	ctx.emitU32(0) // placeholder
}

// EmitJmpToPos emits an unconditional JMP rel32 to an already-known code position.
func (ctx *JITContext) EmitJmpToPos(targetPos int32) {
	curPos := int32(uintptr(ctx.Ptr)-uintptr(ctx.Start)) + 5
	off := targetPos - curPos
	ctx.emitByte(0xE9) // JMP rel32
	ctx.emitU32(uint32(off))
}

// Condition code constants for EmitJcc
const (
	CcE  byte = 0x04 // JE  / JZ  (ZF=1)
	CcNE byte = 0x05 // JNE / JNZ (ZF=0)
	CcBE byte = 0x06 // JBE (unsigned <=)
	CcA  byte = 0x07 // JA  (unsigned >)
	CcL  byte = 0x0C // JL        (SF!=OF)
	CcGE byte = 0x0D // JGE       (SF=OF)
	CcLE byte = 0x0E // JLE       (ZF=1 || SF!=OF)
	CcG  byte = 0x0F // JG        (ZF=0 && SF=OF)
	CcB  byte = 0x02 // JB  (unsigned <)
	CcAE byte = 0x03 // JAE (unsigned >=)
)

// --- MOV helpers ---

// emitMovRegReg emits MOV dst, src (64-bit GPR to GPR)
// EmitMovRegReg emits MOV r64, r64 (no-op if dst == src)
func (ctx *JITContext) EmitMovRegReg(dst, src Reg) {
	if dst == src {
		return
	}
	ctx.emitMovRegReg(dst, src)
}

func (ctx *JITContext) emitMovRegReg(dst, src Reg) {
	rex := byte(0x48)
	if src >= 8 {
		rex |= 0x04 // REX.R
	}
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | (byte(src&7) << 3) | byte(dst&7)
	ctx.emitBytes(rex, 0x89, modrm) // MOV r/m64, r64
}

// EmitMovRegImm64 loads an immediate into a 64-bit register using the
// shortest encoding: XOR reg,reg (2-3 B) for 0, MOV r32,imm32 (5-6 B)
// for values ≤ 0xFFFFFFFF, or full MOV r64,imm64 (10 B) otherwise.
func (ctx *JITContext) EmitMovRegImm64(dst Reg, imm uint64) {
	dstEnc := byte(dst & 7)
	if imm == 0 {
		// XOR r32, r32 — zero-extends to 64 bits (2 or 3 bytes)
		if dst >= 8 {
			ctx.EmitByte(0x45) // REX.R + REX.B
		}
		ctx.emitBytes(0x31, 0xC0|(dstEnc<<3)|dstEnc)
		return
	}
	if imm <= 0xFFFFFFFF {
		// MOV r32, imm32 — zero-extends to 64 bits (5 or 6 bytes)
		if dst >= 8 {
			ctx.EmitByte(0x41) // REX.B
		}
		ctx.EmitByte(0xB8 | dstEnc)
		ctx.emitU32(uint32(imm))
		return
	}
	// Full MOV r64, imm64 (10 bytes)
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	ctx.emitBytes(rex, 0xB8|dstEnc)
	ctx.emitU64(imm)
}

// emitRegMemOp emits <opcode> dst, [base + disp] (REX.W r64, r/m64 with ModRM)
// opcode: 0x8B = MOV (load), 0x8D = LEA (address computation)
func (ctx *JITContext) emitRegMemOp(opcode byte, dst, base Reg, disp int32) {
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
			ctx.emitBytes(rex, opcode, modrm, 0x24)
		} else {
			ctx.emitBytes(rex, opcode, modrm)
		}
	} else if disp >= -128 && disp <= 127 {
		modrm := 0x40 | (dstEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(rex, opcode, modrm, 0x24, byte(int8(disp)))
		} else {
			ctx.emitBytes(rex, opcode, modrm, byte(int8(disp)))
		}
	} else {
		modrm := 0x80 | (dstEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(rex, opcode, modrm, 0x24)
		} else {
			ctx.emitBytes(rex, opcode, modrm)
		}
		ctx.emitU32(uint32(disp))
	}
}

// emitMovRegMem emits MOV dst, [base + disp32] (load 64-bit from memory)
func (ctx *JITContext) emitMovRegMem(dst, base Reg, disp int32) {
	ctx.emitRegMemOp(0x8B, dst, base, disp)
}

// EmitMovRegMem emits MOV dst, [base + disp32] (load 64-bit from memory) — exported wrapper.
func (ctx *JITContext) EmitMovRegMem(dst, base Reg, disp int32) {
	ctx.emitMovRegMem(dst, base, disp)
}

// EmitMovRegMemB emits MOVZX dst, byte [base + disp32] (8-bit zero-extended load).
func (ctx *JITContext) EmitMovRegMemB(dst, base Reg, disp int32) {
	ctx.emitRegMemOp2(0x0F, 0xB6, dst, base, disp)
}

// EmitMovRegMemW emits MOVZX dst, word [base + disp32] (16-bit zero-extended load).
func (ctx *JITContext) EmitMovRegMemW(dst, base Reg, disp int32) {
	ctx.emitRegMemOp2(0x0F, 0xB7, dst, base, disp)
}

// EmitMovRegMemL emits MOV r32, [base + disp32] (32-bit zero-extended load).
func (ctx *JITContext) EmitMovRegMemL(dst, base Reg, disp int32) {
	ctx.emitRegMemOp32(0x8B, dst, base, disp)
}

// EmitLeaRegMem emits LEA dst, [base + disp32] (compute address, no memory access)
// For IndexAddr: LEA dst, [sliceBase + idx*16] computes &a[idx]
func (ctx *JITContext) EmitLeaRegMem(dst, base Reg, disp int32) {
	ctx.emitRegMemOp(0x8D, dst, base, disp)
}

// EmitMovRegMem64 loads a 64-bit value from an absolute memory address into dst.
// Uses dst itself as scratch for the address (avoids clobbering R11).
func (ctx *JITContext) EmitMovRegMem64(dst Reg, addr uintptr) {
	ctx.EmitMovRegImm64(dst, uint64(addr))
	ctx.emitMovRegMem(dst, dst, 0)
}

// EmitMovRegMem32 loads a 32-bit value (zero-extended to 64 bits) from an absolute address.
// Uses dst itself as scratch for the address (avoids clobbering R11).
func (ctx *JITContext) EmitMovRegMem32(dst Reg, addr uintptr) {
	ctx.EmitMovRegImm64(dst, uint64(addr))
	// MOV r32, [dst+0] — 32-bit load zero-extends to 64 bits (no REX.W)
	ctx.emitRegMemOp32(0x8B, dst, dst, 0)
}

// EmitMovRegMem8 loads a byte (zero-extended to 64 bits) from an absolute address.
// Uses dst itself as scratch for the address (avoids clobbering R11).
func (ctx *JITContext) EmitMovRegMem8(dst Reg, addr uintptr) {
	ctx.EmitMovRegImm64(dst, uint64(addr))
	// MOVZX r64, byte [dst+0]
	ctx.emitRegMemOp2(0x0F, 0xB6, dst, dst, 0)
}

// EmitMovRegMem16 loads a 16-bit value (zero-extended to 64 bits) from an absolute address.
// Uses dst itself as scratch for the address (avoids clobbering R11).
func (ctx *JITContext) EmitMovRegMem16(dst Reg, addr uintptr) {
	ctx.EmitMovRegImm64(dst, uint64(addr))
	// MOVZX r64, word [dst+0]
	ctx.emitRegMemOp2(0x0F, 0xB7, dst, dst, 0)
}

// emitRegMemOp32 emits a 32-bit register-memory operation (no REX.W, for zero-extending loads).
func (ctx *JITContext) emitRegMemOp32(opcode byte, dst, base Reg, disp int32) {
	rex := byte(0x40)
	needRex := false
	if dst >= 8 {
		rex |= 0x04 // REX.R
		needRex = true
	}
	if base >= 8 {
		rex |= 0x01 // REX.B
		needRex = true
	}
	baseEnc := byte(base & 7)
	dstEnc := byte(dst & 7)

	if disp == 0 && baseEnc != 5 {
		modrm := (dstEnc << 3) | baseEnc
		if needRex {
			if baseEnc == 4 {
				ctx.emitBytes(rex, opcode, modrm, 0x24)
			} else {
				ctx.emitBytes(rex, opcode, modrm)
			}
		} else {
			if baseEnc == 4 {
				ctx.emitBytes(opcode, modrm, 0x24)
			} else {
				ctx.emitBytes(opcode, modrm)
			}
		}
	} else if disp >= -128 && disp <= 127 {
		modrm := 0x40 | (dstEnc << 3) | baseEnc
		if needRex {
			if baseEnc == 4 {
				ctx.emitBytes(rex, opcode, modrm, 0x24, byte(int8(disp)))
			} else {
				ctx.emitBytes(rex, opcode, modrm, byte(int8(disp)))
			}
		} else {
			if baseEnc == 4 {
				ctx.emitBytes(opcode, modrm, 0x24, byte(int8(disp)))
			} else {
				ctx.emitBytes(opcode, modrm, byte(int8(disp)))
			}
		}
	} else {
		modrm := 0x80 | (dstEnc << 3) | baseEnc
		if needRex {
			if baseEnc == 4 {
				ctx.emitBytes(rex, opcode, modrm, 0x24)
			} else {
				ctx.emitBytes(rex, opcode, modrm)
			}
		} else {
			if baseEnc == 4 {
				ctx.emitBytes(opcode, modrm, 0x24)
			} else {
				ctx.emitBytes(opcode, modrm)
			}
		}
		ctx.emitU32(uint32(disp))
	}
}

// emitRegMemOp2 emits a 2-byte opcode register-memory operation with REX.W (for MOVZX etc.).
func (ctx *JITContext) emitRegMemOp2(op1, op2 byte, dst, base Reg, disp int32) {
	rex := byte(0x48) // REX.W
	if dst >= 8 {
		rex |= 0x04 // REX.R
	}
	if base >= 8 {
		rex |= 0x01 // REX.B
	}
	baseEnc := byte(base & 7)
	dstEnc := byte(dst & 7)

	if disp == 0 && baseEnc != 5 {
		modrm := (dstEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(rex, op1, op2, modrm, 0x24)
		} else {
			ctx.emitBytes(rex, op1, op2, modrm)
		}
	} else if disp >= -128 && disp <= 127 {
		modrm := 0x40 | (dstEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(rex, op1, op2, modrm, 0x24, byte(int8(disp)))
		} else {
			ctx.emitBytes(rex, op1, op2, modrm, byte(int8(disp)))
		}
	} else {
		modrm := 0x80 | (dstEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(rex, op1, op2, modrm, 0x24)
		} else {
			ctx.emitBytes(rex, op1, op2, modrm)
		}
		ctx.emitU32(uint32(disp))
	}
}

// --- SSE helpers ---

// emitSseOp emits F2 0F <op> xmmDst, xmmSrc (scalar double operation)
func (ctx *JITContext) emitSseOp(op byte, dst, src Reg) {
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
		ctx.emitBytes(0xF2, rex, 0x0F, op, modrm)
	} else {
		ctx.emitBytes(0xF2, 0x0F, op, modrm)
	}
}

// emitMovqXmmToGpr emits MOVQ gprDst, xmmSrc (66 REX.W 0F 7E /r)
func (ctx *JITContext) emitMovqXmmToGpr(gpr, xmm Reg) {
	x := xmm - 16
	rex := byte(0x48) // REX.W
	if x >= 8 {
		rex |= 0x04 // REX.R
	}
	if gpr >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | (byte(x&7) << 3) | byte(gpr&7)
	ctx.emitBytes(0x66, rex, 0x0F, 0x7E, modrm)
}

// emitMovqGprToXmm emits MOVQ xmmDst, gprSrc (66 REX.W 0F 6E /r)
func (ctx *JITContext) emitMovqGprToXmm(xmm, gpr Reg) {
	x := xmm - 16
	rex := byte(0x48)
	if x >= 8 {
		rex |= 0x04 // REX.R
	}
	if gpr >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | (byte(x&7) << 3) | byte(gpr&7)
	ctx.emitBytes(0x66, rex, 0x0F, 0x6E, modrm)
}

// emitMovqMemToXmm emits MOVQ xmmDst, [base + disp32] (F3 0F 7E /r m64)
func (ctx *JITContext) emitMovqMemToXmm(xmm, base Reg, disp int32) {
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
		ctx.emitBytes(0xF3, rex, 0x0F, 0x7E)
	} else {
		ctx.emitBytes(0xF3, 0x0F, 0x7E)
	}

	if disp >= -128 && disp <= 127 {
		modrm := 0x40 | (xEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(modrm, 0x24, byte(int8(disp)))
		} else {
			ctx.emitBytes(modrm, byte(int8(disp)))
		}
	} else {
		modrm := 0x80 | (xEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(modrm, 0x24)
		} else {
			ctx.emitBytes(modrm)
		}
		ctx.emitU32(uint32(disp))
	}
}

// --- Compare helpers ---

// EmitCmpRegImm32 emits CMP r64, sign-extended imm32
func (ctx *JITContext) EmitCmpRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xF8) | byte(dst&7) // /7 = CMP
	ctx.emitBytes(rex, 0x81, modrm)
	ctx.emitU32(uint32(imm))
}

// EmitCmpRegImm8 emits CMP r8, imm8 on the low byte of the register.
// This is used for compact Scmer tag checks where tags live in aux[7:0].
func (ctx *JITContext) EmitCmpRegImm8(dst Reg, imm uint8) {
	rex := byte(0x40) // force low-byte register encoding (incl. SIL/DIL/BPL/SPL)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xF8) | byte(dst&7) // /7 = CMP, mod=11, r/m=dst
	ctx.emitBytes(rex, 0x80, modrm, imm)
}

// EmitAddRegImm32 emits ADD r64, sign-extended imm32.
func (ctx *JITContext) EmitAddRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | byte(dst&7) // /0 = ADD
	ctx.emitBytes(rex, 0x81, modrm)
	ctx.emitU32(uint32(imm))
}

// EmitSubRegImm32 emits SUB r64, sign-extended imm32.
func (ctx *JITContext) EmitSubRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xE8) | byte(dst&7) // /5 = SUB
	ctx.emitBytes(rex, 0x81, modrm)
	ctx.emitU32(uint32(imm))
}

// EmitOrRegImm32 emits OR r64, sign-extended imm32.
func (ctx *JITContext) EmitOrRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC8) | byte(dst&7) // /1 = OR
	ctx.emitBytes(rex, 0x81, modrm)
	ctx.emitU32(uint32(imm))
}

// EmitImulRegImm32 emits IMUL r64, r64, imm32.
func (ctx *JITContext) EmitImulRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x05 // REX.R | REX.B (reg and r/m are both dst)
	}
	modrm := byte(0xC0) | (byte(dst&7) << 3) | byte(dst&7)
	ctx.emitBytes(rex, 0x69, modrm)
	ctx.emitU32(uint32(imm))
}

// EmitIdivRegImm emits signed integer division of dst by imm and stores the quotient in dst.
func (ctx *JITContext) EmitIdivRegImm(dst Reg, imm int64) {
	if imm == 0 {
		panic("jit: divide by zero in EmitIdivRegImm")
	}
	restoreRAX := dst != RegRAX
	restoreRDX := dst != RegRDX
	if restoreRAX {
		ctx.EmitPushReg(RegRAX)
	}
	if restoreRDX {
		ctx.EmitPushReg(RegRDX)
	}
	if dst != RegRAX {
		ctx.emitMovRegReg(RegRAX, dst)
	}
	// CQO sign-extends RAX into RDX:RAX for IDIV.
	ctx.emitBytes(0x48, 0x99)
	ctx.EmitMovRegImm64(RegR11, uint64(imm))
	// IDIV r/m64
	ctx.emitBytes(0x49, 0xF7, 0xFB) // idiv r11
	if dst != RegRAX {
		ctx.emitMovRegReg(dst, RegRAX)
	}
	if restoreRDX {
		ctx.EmitPopReg(RegRDX)
	}
	if restoreRAX {
		ctx.EmitPopReg(RegRAX)
	}
}

// EmitIremRegImm emits signed integer remainder of dst by imm and stores the remainder in dst.
func (ctx *JITContext) EmitIremRegImm(dst Reg, imm int64) {
	if imm == 0 {
		panic("jit: modulo by zero in EmitIremRegImm")
	}
	restoreRAX := dst != RegRAX
	restoreRDX := dst != RegRDX
	if restoreRAX {
		ctx.EmitPushReg(RegRAX)
	}
	if restoreRDX {
		ctx.EmitPushReg(RegRDX)
	}
	if dst != RegRAX {
		ctx.emitMovRegReg(RegRAX, dst)
	}
	// CQO sign-extends RAX into RDX:RAX for IDIV.
	ctx.emitBytes(0x48, 0x99)
	ctx.EmitMovRegImm64(RegR11, uint64(imm))
	// IDIV r/m64
	ctx.emitBytes(0x49, 0xF7, 0xFB) // idiv r11
	if dst != RegRDX {
		ctx.emitMovRegReg(dst, RegRDX)
	}
	if restoreRDX {
		ctx.EmitPopReg(RegRDX)
	}
	if restoreRAX {
		ctx.EmitPopReg(RegRAX)
	}
}

// EmitSetcc emits SETcc r/m8 + MOVZX r32, r8 → zero-extended 0 or 1 in full 64-bit register
func (ctx *JITContext) EmitSetcc(dst Reg, cc byte) {
	dstEnc := byte(dst & 7)
	// SETcc r/m8: 0F 9x /0
	if dst >= 8 {
		ctx.emitBytes(0x41, 0x0F, 0x90|cc, 0xC0|dstEnc)
	} else if dst >= 4 {
		ctx.emitBytes(0x40, 0x0F, 0x90|cc, 0xC0|dstEnc) // REX for SIL/DIL/BPL/SPL
	} else {
		ctx.emitBytes(0x0F, 0x90|cc, 0xC0|dstEnc)
	}
	// MOVZX r32, r8: 0F B6 /r (32-bit write zeros upper 32)
	modrm := byte(0xC0) | (dstEnc << 3) | dstEnc
	if dst >= 8 {
		ctx.emitBytes(0x45, 0x0F, 0xB6, modrm)
	} else if dst >= 4 {
		ctx.emitBytes(0x40, 0x0F, 0xB6, modrm)
	} else {
		ctx.emitBytes(0x0F, 0xB6, modrm)
	}
}

// --- Shift emitters ---

// EmitShlRegImm8 emits SHL r64, imm8 (logical shift left by immediate)
func (ctx *JITContext) EmitShlRegImm8(dst Reg, imm uint8) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xE0) | byte(dst&7) // /4 = SHL
	ctx.emitBytes(rex, 0xC1, modrm, imm)
}

// EmitShrRegImm8 emits SHR r64, imm8 (logical shift right by immediate)
func (ctx *JITContext) EmitShrRegImm8(dst Reg, imm uint8) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xE8) | byte(dst&7) // /5 = SHR
	ctx.emitBytes(rex, 0xC1, modrm, imm)
}

// EmitSarRegImm8 emits SAR r64, imm8 (arithmetic shift right by immediate)
func (ctx *JITContext) EmitSarRegImm8(dst Reg, imm uint8) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xF8) | byte(dst&7) // /7 = SAR
	ctx.emitBytes(rex, 0xC1, modrm, imm)
}

// EmitShlRegCl emits SHL r64, CL (shift left by variable amount in CL register)
func (ctx *JITContext) EmitShlRegCl(dst Reg) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xE0) | byte(dst&7) // /4 = SHL
	ctx.emitBytes(rex, 0xD3, modrm)
}

// EmitShrRegCl emits SHR r64, CL (shift right by variable amount in CL register)
func (ctx *JITContext) EmitShrRegCl(dst Reg) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xE8) | byte(dst&7) // /5 = SHR
	ctx.emitBytes(rex, 0xD3, modrm)
}

// EmitAndRegImm32 emits AND r64, imm32 (sign-extended)
func (ctx *JITContext) EmitAndRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xE0) | byte(dst&7) // /4 = AND
	ctx.emitBytes(rex, 0x81, modrm)
	ctx.emitU32(uint32(imm))
}

// EmitOrInt64 emits OR dst, src (64-bit OR)
func (ctx *JITContext) EmitOrInt64(dst, src Reg) {
	ctx.emitAluRegReg(0x09, dst, src) // OR r/m64, r64
}

// EmitAndInt64 emits AND dst, src (64-bit AND)
func (ctx *JITContext) EmitAndInt64(dst, src Reg) {
	ctx.emitAluRegReg(0x21, dst, src) // AND r/m64, r64
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
		ctx.EmitMakeInt(result, r)
		return result
	}
	if src.Type != JITTypeUnknown {
		// Type is known at compile time — constant-fold
		ctx.FreeDesc(src)
		r := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(src.Type))}
		if result.Loc == LocAny {
			return r
		}
		ctx.EmitMakeInt(result, r)
		return result
	}
	// Dynamic type: materialize spilled descriptors before reading Reg/Reg2.
	ctx.EnsureDesc(src)
	dst := ctx.AllocReg()
	ctx.emitGetTagRegs(dst, src.Reg, src.Reg2)
	ctx.FreeDesc(src)
	r := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: dst}
	if result.Loc == LocAny {
		return r
	}
	ctx.EmitMakeInt(result, r)
	ctx.FreeReg(dst)
	return result
}

// EmitTagEquals checks if a Scmer's type tag equals a constant.
// Equivalent to GetTag(src) == tag. Consumes src.
func (ctx *JITContext) EmitTagEquals(src *JITValueDesc, tag uint8, result JITValueDesc) JITValueDesc {
	if src.Loc == LocImm {
		r := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(src.Imm.GetTag() == tag)}
		if result.Loc == LocAny {
			return r
		}
		ctx.EmitMakeBool(result, r)
		return result
	}
	if src.Type != JITTypeUnknown {
		// Type is known at compile time — constant-fold
		ctx.FreeDesc(src)
		r := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(src.Type == tag)}
		if result.Loc == LocAny {
			return r
		}
		ctx.EmitMakeBool(result, r)
		return result
	}
	// Dynamic type: materialize spilled descriptors before reading Reg/Reg2.
	ctx.EnsureDesc(src)
	tagReg := ctx.AllocReg()
	ctx.emitGetTagRegs(tagReg, src.Reg, src.Reg2)
	ctx.FreeDesc(src)
	ctx.EmitCmpRegImm8(tagReg, tag)
	ctx.EmitSetcc(tagReg, CcE)
	r := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: tagReg}
	if result.Loc == LocAny {
		return r
	}
	ctx.EmitMakeBool(result, r)
	ctx.FreeReg(tagReg)
	return result
}

// EmitTagEqualsBorrowed checks if a Scmer's tag equals a constant without
// consuming/clobbering the source descriptor. This is required when the same
// SSA value is used both for a type predicate and later value extraction.
func (ctx *JITContext) EmitTagEqualsBorrowed(src *JITValueDesc, tag uint8, result JITValueDesc) JITValueDesc {
	emitOut := func(v JITValueDesc) JITValueDesc {
		if result.Loc == LocAny {
			return v
		}
		ctx.EmitMakeBool(result, v)
		ctx.FreeDesc(&v)
		return result
	}

	// Immediate and known-typed values can be folded without touching source regs.
	if src.Loc == LocImm {
		return emitOut(JITValueDesc{
			Loc:  LocImm,
			Type: tagBool,
			Imm:  NewBool(src.Imm.GetTag() == tag),
		})
	}
	if src.Type != JITTypeUnknown {
		return emitOut(JITValueDesc{
			Loc:  LocImm,
			Type: tagBool,
			Imm:  NewBool(src.Type == tag),
		})
	}

	// Borrowed fast path: read tag directly from pair registers without cloning.
	if src.Loc == LocRegPair {
		ctx.ProtectReg(src.Reg)
		ctx.ProtectReg(src.Reg2)
		tagReg := ctx.AllocRegExcept(src.Reg, src.Reg2)
		ctx.UnprotectReg(src.Reg2)
		ctx.UnprotectReg(src.Reg)
		ctx.emitGetTagRegs(tagReg, src.Reg, src.Reg2)
		ctx.EmitCmpRegImm8(tagReg, tag)
		ctx.EmitSetcc(tagReg, CcE)
		return emitOut(JITValueDesc{Loc: LocReg, Type: tagBool, Reg: tagReg})
	}

	// Other borrowed forms: detached copy so EmitTagEquals may consume it safely.
	tmp := *src
	tmp.ID = 0
	return ctx.EmitTagEquals(&tmp, tag, result)
}

// EmitBoolDesc evaluates Scmer truthiness equivalent to (Scmer).Bool().
// It consumes src and returns a bool descriptor (LocImm or LocReg).
// Fast paths are emitted for compile-time constants and known primitive types;
// dynamic/complex cases fall back to calling Scmer.Bool.
func (ctx *JITContext) EmitBoolDesc(src *JITValueDesc, result JITValueDesc) JITValueDesc {
	emitResult := func(v JITValueDesc) JITValueDesc {
		if result.Loc == LocAny {
			return v
		}
		ctx.EmitMakeBool(result, v)
		ctx.FreeDesc(&v)
		return result
	}

	if src.Loc == LocImm {
		return emitResult(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(src.Imm.Bool())})
	}
	if src.Type == tagNil {
		ctx.FreeDesc(src)
		return emitResult(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)})
	}
	if src.Type == tagDate {
		ctx.FreeDesc(src)
		return emitResult(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)})
	}

	// Known primitive types can be lowered directly without helper calls.
	if src.Type == tagBool || src.Type == tagInt || src.Type == tagFloat {
		ctx.EnsureDesc(src)
		srcLoc := src.Loc
		srcReg := src.Reg
		srcReg2 := src.Reg2
		var valReg Reg
		switch src.Loc {
		case LocReg:
			valReg = src.Reg
		case LocRegPair:
			valReg = src.Reg2 // aux payload contains bool/int/float bits
		default:
			// EnsureDesc should have materialized stack/mem forms.
			panic("jit: EmitBoolDesc primitive type not in register location")
		}

		dst := ctx.AllocReg()
		if valReg != dst {
			ctx.emitMovRegReg(dst, valReg)
		}

		if src.Type == tagFloat {
			// Float truthiness is float64(bits) != 0.0. Mask sign bit so -0.0
			// becomes zero, then compare against zero.
			mask := ctx.AllocReg()
			ctx.EmitMovRegImm64(mask, 0x7fffffffffffffff)
			ctx.EmitAndInt64(dst, mask)
			ctx.FreeReg(mask)
		} else if src.Type == tagBool {
			// Bool payload is auxVal in bits [63:8]; low 8 bits hold the tag.
			ctx.EmitShrRegImm8(dst, 8)
		}
		ctx.EmitCmpRegImm32(dst, 0)
		ctx.EmitSetcc(dst, CcNE)

		// Keep the register that now carries the boolean result alive.
		// FreeDesc on an aliased source would otherwise free dst.
		switch srcLoc {
		case LocReg:
			if dst != srcReg {
				ctx.FreeReg(srcReg)
			}
		case LocRegPair:
			if dst == srcReg {
				ctx.FreeReg(srcReg2)
			} else if dst == srcReg2 {
				ctx.FreeReg(srcReg)
			} else {
				ctx.FreeReg(srcReg)
				ctx.FreeReg(srcReg2)
			}
		default:
			ctx.FreeDesc(src)
		}
		return emitResult(JITValueDesc{Loc: LocReg, Type: tagBool, Reg: dst})
	}

	// Unknown or complex known types (string/symbol/slice/vector/fastdict/default):
	// materialize a Scmer pair and reuse the canonical runtime helper.
	pair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
	pair = jitPlaceIntoPair(ctx, src, pair)
	out := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Bool), []JITValueDesc{pair}, 1)
	// Go bool returns may leave upper bits undefined; normalize to 0|1.
	ctx.EmitAndRegImm32(out.Reg, 1)
	out.Type = tagBool
	ctx.FreeDesc(&pair)
	ctx.FreeDesc(src)
	return emitResult(out)
}

// EmitMovToReg moves a JITValueDesc value into a specific GPR register.
// Handles LocImm (materializes constant) and LocReg (register-to-register move).
func (ctx *JITContext) EmitMovToReg(dst Reg, src JITValueDesc) {
	switch src.Loc {
	case LocImm:
		ctx.EmitMovRegImm64(dst, uint64(src.Imm.Int()))
	case LocReg:
		if src.Reg != dst {
			ctx.emitMovRegReg(dst, src.Reg)
		}
	}
}

// emitGetTagRegs emits inline code for (Scmer).GetTag().
// Input: ptrReg holds s.ptr, auxReg holds s.aux.
// Output: result in dstReg as uint16.
// Logic: if ptr == &scmerIntSentinel → tagInt (4)
//
//	if ptr == &scmerFloatSentinel → tagFloat (3)
//	else → aux & 0xFF
func (ctx *JITContext) emitGetTagRegs(dst, ptrReg, auxReg Reg) {
	// CMP ptrReg, &scmerIntSentinel (via R11 as scratch)
	ctx.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(&scmerIntSentinel))))
	ctx.EmitCmpInt64(ptrReg, RegR11)
	// JE .is_int (patch later)
	ctx.emitBytes(0x0F, 0x84) // JE rel32
	isIntFixup := ctx.Ptr
	ctx.emitU32(0)

	// CMP ptrReg, &scmerFloatSentinel
	ctx.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(&scmerFloatSentinel))))
	ctx.EmitCmpInt64(ptrReg, RegR11)
	// JE .is_float (patch later)
	ctx.emitBytes(0x0F, 0x84) // JE rel32
	isFloatFixup := ctx.Ptr
	ctx.emitU32(0)

	// Default: dst = aux & 0xFF
	if dst != auxReg {
		ctx.emitMovRegReg(dst, auxReg)
	}
	ctx.EmitAndRegImm32(dst, 0xFF)
	// JMP .done
	ctx.emitByte(0xE9) // JMP rel32
	doneFixup := ctx.Ptr
	ctx.emitU32(0)

	// .is_int: dst = tagInt (4)
	isIntTarget := ctx.Ptr
	ctx.EmitMovRegImm64(dst, uint64(tagInt))
	// JMP .done
	ctx.emitByte(0xE9) // JMP rel32
	doneFixup2 := ctx.Ptr
	ctx.emitU32(0)

	// .is_float: dst = tagFloat (3)
	isFloatTarget := ctx.Ptr
	ctx.EmitMovRegImm64(dst, uint64(tagFloat))
	// fall through to .done

	// .done:
	doneTarget := ctx.Ptr

	// Patch fixups
	*(*int32)(isIntFixup) = int32(uintptr(isIntTarget) - uintptr(isIntFixup) - 4)
	*(*int32)(isFloatFixup) = int32(uintptr(isFloatTarget) - uintptr(isFloatFixup) - 4)
	*(*int32)(doneFixup) = int32(uintptr(doneTarget) - uintptr(doneFixup) - 4)
	*(*int32)(doneFixup2) = int32(uintptr(doneTarget) - uintptr(doneFixup2) - 4)
}

// --- PUSH/POP/CALL ---

// EmitPushReg emits PUSH r64
func (ctx *JITContext) EmitPushReg(r Reg) {
	if r >= 8 {
		ctx.emitBytes(0x41, 0x50|byte(r&7))
	} else {
		ctx.emitByte(0x50 | byte(r))
	}
}

// EmitPopReg emits POP r64
func (ctx *JITContext) EmitPopReg(r Reg) {
	if r >= 8 {
		ctx.emitBytes(0x41, 0x58|byte(r&7))
	} else {
		ctx.emitByte(0x58 | byte(r))
	}
}

// EmitCallIndirect emits MOV R11, imm64; CALL R11
func (ctx *JITContext) EmitCallIndirect(addr uint64) {
	ctx.EmitMovRegImm64(RegR11, addr)
	ctx.emitBytes(0x41, 0xFF, 0xD3) // CALL R11
}

// EmitGoCallVariadic emits a direct call to a func(...Scmer) Scmer function value.
//
// amd64 regabi function-value call:
//   - RDX = funcval pointer (payload/closure pointer)
//   - RAX = slice.data, RBX = slice.len, RCX = slice.cap
//   - CALL [RDX] (fnptr)
//
// argslice must describe a pair (ptr,len). The backing array is expected to be
// materialized and kept alive by caller-managed stack memory.
func (ctx *JITContext) EmitGoCallVariadic(f func(...Scmer) Scmer, argslice JITValueDesc, result JITValueDesc) JITValueDesc {
	fnData := *(*uintptr)(unsafe.Pointer(&f))
	if fnData == 0 {
		panic("jit: nil variadic function value")
	}

	arg := argslice
	if arg.Loc == LocStackPair {
		ctx.EnsureDesc(&arg)
	}
	if arg.Loc == LocImm {
		if !arg.Imm.IsNil() {
			panic("jit: variadic argslice LocImm must be nil")
		}
		arg = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
		ctx.EmitMovRegImm64(arg.Reg, 0)
		ctx.EmitMovRegImm64(arg.Reg2, 0)
		ctx.BindReg(arg.Reg, &arg)
		ctx.BindReg(arg.Reg2, &arg)
	}
	if arg.Loc != LocRegPair {
		panic(fmt.Sprintf("jit: variadic argslice must be LocRegPair/LocStackPair (got %d)", arg.Loc))
	}

	target := result
	targetHasRegs := target.Loc == LocRegPair
	if targetHasRegs {
		target.Type = JITTypeUnknown
	}

	ctx.ReclaimUntrackedRegs()
	var liveRegsArr [16]Reg
	liveCount := 0
	for r := Reg(0); r <= RegR15; r++ {
		if r == RegRSP || r == RegRBP || r == RegR11 || r == RegR14 {
			continue
		}
		bit := uint64(1 << uint(r))
		if (ctx.AllRegs&bit) == 0 || (ctx.FreeRegs&bit) != 0 {
			continue
		}
		owner := ctx.RegOwners[r]
		if owner == nil {
			continue
		}
		valid := false
		switch owner.Loc {
		case LocReg:
			valid = owner.Reg == r
		case LocRegPair:
			valid = owner.Reg == r || owner.Reg2 == r
		}
		if !valid {
			continue
		}
		liveRegsArr[liveCount] = r
		liveCount++
	}
	liveRegs := liveRegsArr[:0]
	for i := 0; i < liveCount; i++ {
		r := liveRegsArr[i]
		if r == arg.Reg || r == arg.Reg2 {
			continue
		}
		if targetHasRegs {
			if r == target.Reg || r == target.Reg2 {
				continue
			}
		}
		liveRegs = append(liveRegs, r)
	}
	switch ctx.SliceBase {
	case RegRSP, RegRBP, RegR11, RegR14:
	default:
		if !ctx.SliceBaseTracksRSP {
			found := false
			for _, r := range liveRegs {
				if r == ctx.SliceBase {
					found = true
					break
				}
			}
			if !found {
				liveRegs = append(liveRegs, ctx.SliceBase)
			}
		}
	}

	// Single per-call frame (thread-safe):
	// [rsp+0..] = saved live regs
	frameBytes := int32(len(liveRegs)) * 8
	if frameBytes%16 != 0 {
		frameBytes += 8 // keep alignment equivalent to pre-call RSP parity
	}
	if frameBytes != 0 {
		ctx.EmitSubRSP32(frameBytes)
	}

	for i, r := range liveRegs {
		ctx.EmitStoreRegMem(r, RegRSP, int32(i*8))
	}

	// Stage argslice into scratch regs, then set call registers.
	if arg.Reg != RegRAX {
		ctx.EmitMovRegReg(RegRAX, arg.Reg)
	}
	if arg.Reg2 != RegRBX {
		ctx.EmitMovRegReg(RegRBX, arg.Reg2)
	}
	ctx.EmitMovRegReg(RegRCX, RegRBX) // cap = len
	ctx.EmitMovRegImm64(RegRDX, uint64(fnData))
	ctx.EmitMovRegMem(RegR11, RegRDX, 0) // fnptr := [funcval]
	ctx.emitBytes(0x41, 0xFF, 0xD3)      // CALL R11

	if !targetHasRegs {
		target = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
		targetHasRegs = true
	}
	if target.Reg != RegRAX {
		ctx.EmitMovRegReg(target.Reg, RegRAX)
	}
	if target.Reg2 != RegRBX {
		ctx.EmitMovRegReg(target.Reg2, RegRBX)
	}
	for i, r := range liveRegs {
		ctx.EmitMovRegMem(r, RegRSP, int32(i*8))
	}
	if frameBytes != 0 {
		ctx.EmitAddRSP32(frameBytes)
	}
	ctx.BindReg(target.Reg, &target)
	ctx.BindReg(target.Reg2, &target)

	if ctx.SliceBaseTracksRSP && ctx.SliceBase != RegRSP {
		ctx.emitMovRegReg(ctx.SliceBase, RegRSP)
	}
	return target
}

// emitStoreRegMem emits MOV [base + disp], src (store 64-bit register to memory)
func (ctx *JITContext) emitStoreRegMem(src, base Reg, disp int32) {
	rex := byte(0x48)
	if src >= 8 {
		rex |= 0x04 // REX.R
	}
	if base >= 8 {
		rex |= 0x01 // REX.B
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
	} else if disp >= -128 && disp <= 127 {
		modrm := 0x40 | (srcEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(rex, 0x89, modrm, 0x24, byte(int8(disp)))
		} else {
			ctx.emitBytes(rex, 0x89, modrm, byte(int8(disp)))
		}
	} else {
		modrm := 0x80 | (srcEnc << 3) | baseEnc
		if baseEnc == 4 {
			ctx.emitBytes(rex, 0x89, modrm, 0x24)
		} else {
			ctx.emitBytes(rex, 0x89, modrm)
		}
		ctx.emitU32(uint32(disp))
	}
}

// --- GPR ALU encoding helper ---

// emitAluRegReg emits a REX.W ALU op: <opcode> r/m64, r64
// opcode: 0x01=ADD, 0x29=SUB, 0x39=CMP, 0x09=OR, 0x21=AND, 0x31=XOR
func (ctx *JITContext) emitAluRegReg(opcode byte, dst, src Reg) {
	rex := byte(0x48)
	if src >= 8 {
		rex |= 0x04
	}
	if dst >= 8 {
		rex |= 0x01
	}
	modrm := byte(0xC0) | (byte(src&7) << 3) | byte(dst&7)
	ctx.emitBytes(rex, opcode, modrm)
}

// EmitStoreRegMem is the exported version of emitStoreRegMem:
// MOV [base+disp], src (64-bit store).
func (ctx *JITContext) EmitStoreRegMem(src, base Reg, disp int32) {
	ctx.emitStoreRegMem(src, base, disp)
}

// EmitSubRSP emits SUB RSP, imm8 to reserve stack space.
func (ctx *JITContext) EmitSubRSP(n uint8) {
	ctx.emitBytes(0x48, 0x83, 0xEC, n)
}

// EmitAddRSP emits ADD RSP, imm8 to release stack space.
func (ctx *JITContext) EmitAddRSP(n uint8) {
	ctx.emitBytes(0x48, 0x83, 0xC4, n)
}

// EmitSubRSP32Fixup emits SUB RSP, imm32 with a zero placeholder and returns
// a pointer to the 4-byte immediate so it can be patched later via PatchInt32.
func (ctx *JITContext) EmitSubRSP32Fixup() unsafe.Pointer {
	ctx.emitBytes(0x48, 0x81, 0xEC)
	ctx.emitU32(0)
	return unsafe.Add(ctx.Ptr, -4)
}

// PatchInt32 writes a 32-bit little-endian value at the given position.
func (ctx *JITContext) PatchInt32(pos unsafe.Pointer, val int32) {
	*(*int32)(pos) = val
}

// EmitAddRSP32 emits ADD RSP, imm32.
func (ctx *JITContext) EmitAddRSP32(val int32) {
	ctx.emitBytes(0x48, 0x81, 0xC4)
	ctx.emitU32(uint32(val))
}

// EmitSubRSP32 emits SUB RSP, imm32.
func (ctx *JITContext) EmitSubRSP32(val int32) {
	ctx.emitBytes(0x48, 0x81, 0xEC)
	ctx.emitU32(uint32(val))
}

// EmitStoreToStack stores a JITValueDesc value to a stack slot at [RSP+disp].
// Uses R11 as scratch for LocImm values.
// No SpillDepth adjustment needed — spills use a separate buffer, not RSP.
func (ctx *JITContext) EmitStoreToStack(src JITValueDesc, disp int32) {
	switch src.Loc {
	case LocImm:
		var word uint64
		switch src.Imm.GetTag() {
		case tagFloat:
			word = math.Float64bits(src.Imm.Float())
		case tagBool:
			if src.Imm.Bool() {
				word = 1
			} else {
				word = 0
			}
		case tagNil:
			word = 0
		default:
			word = uint64(src.Imm.Int())
		}
		ctx.EmitMovRegImm64(RegR11, word)
		ctx.EmitStoreRegMem(RegR11, RegRSP, disp)
	case LocReg:
		ctx.EmitStoreRegMem(src.Reg, RegRSP, disp)
	}
}

// EmitLoadFromStack loads a value from stack slot [RSP+disp] into a register.
// No SpillDepth adjustment needed — spills use a separate buffer, not RSP.
func (ctx *JITContext) EmitLoadFromStack(dst Reg, disp int32) {
	ctx.EmitMovRegMem(dst, RegRSP, disp)
}

// EmitStoreScmerToStack stores a full Scmer (16 bytes: ptr at disp, aux at disp+8)
// from a LocRegPair or LocImm descriptor to consecutive stack slots [RSP+disp..RSP+disp+15].
// Uses R11 as scratch for LocImm values.
func (ctx *JITContext) EmitStoreScmerToStack(desc JITValueDesc, disp int32) {
	switch desc.Loc {
	case LocRegPair:
		ctx.EmitStoreRegMem(desc.Reg, RegRSP, disp)
		ctx.EmitStoreRegMem(desc.Reg2, RegRSP, disp+8)
	case LocImm:
		// Store ptr word
		ctx.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(desc.Imm.ptr))))
		ctx.EmitStoreRegMem(RegR11, RegRSP, disp)
		// Store aux word
		ctx.EmitMovRegImm64(RegR11, desc.Imm.aux)
		ctx.EmitStoreRegMem(RegR11, RegRSP, disp+8)
	default:
		panic("jit: EmitStoreScmerToStack: unsupported location")
	}
}
