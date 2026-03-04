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
	"unsafe"
)

// TODO: create this file for other architectures, too

// jitCompileProc compiles a Proc body to amd64 machine code or returns nil.
func jitCompileProc(proc *Proc) []byte {
	code, _ := jitCompileProcWithRoots(proc)
	return code
}

// jitCompileProcWithRoots compiles a Proc body to amd64 machine code and
// returns GC roots for pointer constants embedded into immediates.
func jitCompileProcWithRoots(proc *Proc) ([]byte, []unsafe.Pointer) {
	body := proc.Body
	if body.GetTag() == tagSourceInfo {
		body = body.SourceInfo().value
	}
	return jitCompileExprBody(proc, body, proc.NumVars)
}

// jitCompileExprBody compiles a Scheme expression body to machine code
// using Declaration.JITEmit callbacks. Returns nil if any sub-expression
// is not JIT-compilable.
func jitCompileExprBody(proc *Proc, body Scmer, numVars int) (code []byte, roots []unsafe.Pointer) {
	defer func() {
		if r := recover(); r != nil {
			if JITLog {
				fmt.Println("JIT panic", r)
			}
			code = nil
			roots = nil
		}
	}()

	// Allocate temp buffer for code emission
	codeBuf := make([]byte, 16384)
	w := &JITWriter{
		Ptr:   unsafe.Pointer(&codeBuf[0]),
		Start: unsafe.Pointer(&codeBuf[0]),
		End:   unsafe.Add(unsafe.Pointer(&codeBuf[0]), len(codeBuf)-256),
	}

	// Free registers: all GPRs except RAX (result ptr), RBX (result aux),
	// RSP, RBP, R11 (scratch), R12 (slice base), R14 (Go goroutine ptr "g")
	freeRegs := uint64((1 << uint(RegRCX)) | (1 << uint(RegRDX)) |
		(1 << uint(RegRSI)) | (1 << uint(RegRDI)) |
		(1 << uint(RegR8)) | (1 << uint(RegR9)) | (1 << uint(RegR10)) |
		(1 << uint(RegR13)) | (1 << uint(RegR15)))
	ctx := &JITContext{
		W:         w,
		FreeRegs:  freeRegs,
		AllRegs:   freeRegs,
		SliceBase: RegR12,
	}

	// Emit: MOV R12, RAX (save slice base pointer)
	w.emitMovRegReg(RegR12, RegRAX)
	// Copy incoming variadic arguments into an emitter-local stack frame.
	// Go helper calls use PUSH/POP heavily and may overlap caller-provided
	// argument memory; reading NthLocalVar from a private copy is stable.
	frameBytes := 0
	if numVars > 0 {
		slots := numVars
		frameBytes = slots * 16
		if frameBytes < 128 {
			w.EmitSubRSP(uint8(frameBytes))
		} else {
			w.emitBytes(0x48, 0x81, 0xEC)
			w.emitU32(uint32(frameBytes))
		}
		for i := 0; i < slots; i++ {
			srcOff := int32(i * 16)
			dstOff := int32(i * 16)
			w.EmitMovRegMem(RegR11, RegR12, srcOff)
			w.EmitStoreRegMem(RegR11, RegRSP, dstOff)
			w.EmitMovRegMem(RegR11, RegR12, srcOff+8)
			w.EmitStoreRegMem(RegR11, RegRSP, dstOff+8)
		}
		w.emitMovRegReg(RegR12, RegRSP)
		ctx.SliceBaseTracksRSP = true
	}

	// Map lambda parameters to local stack slots so symbol lookup remains correct
	// even when the optimizer did not rewrite body symbols to NthLocalVar.
	if proc != nil {
		vars := make(map[Symbol]JITValueDesc, numVars)
		switch proc.Params.GetTag() {
		case tagSlice:
			params := proc.Params.Slice()
			for i := 0; i < len(params) && i < numVars; i++ {
				if params[i].GetTag() != tagSymbol {
					continue
				}
				vars[params[i].Symbol()] = JITValueDesc{
					Loc:      LocStackPair,
					Type:     JITTypeUnknown,
					StackOff: int32(i * 16),
				}
			}
		case tagSymbol:
			if numVars > 0 {
				vars[proc.Params.Symbol()] = JITValueDesc{
					Loc:      LocStackPair,
					Type:     JITTypeUnknown,
					StackOff: 0,
				}
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
			w.EmitMakeBool(result, desc)
		case tagInt:
			w.EmitMakeInt(result, desc)
		case tagFloat:
			w.EmitMakeFloat(result, desc)
		case tagNil:
			w.EmitMakeNil(result)
		default:
			return nil, nil
		}
		if frameBytes > 0 {
			w.EmitAddRSP32(int32(frameBytes))
		}
		w.emitByte(0xC3) // RET
	} else {
		// Ensure non-immediate results are in ABI return registers.
		ctx.EnsureDesc(&desc)
		switch desc.Loc {
		case LocRegPair:
			if desc.Reg != RegRAX {
				w.emitMovRegReg(RegRAX, desc.Reg)
			}
			if desc.Reg2 != RegRBX {
				w.emitMovRegReg(RegRBX, desc.Reg2)
			}
		case LocReg:
			ret := JITValueDesc{Loc: LocRegPair, Reg: RegRAX, Reg2: RegRBX}
			switch desc.Type {
			case tagBool:
				w.EmitMakeBool(ret, desc)
			case tagInt:
				w.EmitMakeInt(ret, desc)
			case tagFloat:
				w.EmitMakeFloat(ret, desc)
			default:
				return nil, nil
			}
		default:
			return nil, nil
		}
		if frameBytes > 0 {
			w.EmitAddRSP32(int32(frameBytes))
		}
		w.emitByte(0xC3) // RET
	}

	w.ResolveFixupsFinal()
	codeLen := int(uintptr(w.Ptr) - uintptr(w.Start))
	return codeBuf[:codeLen], ctx.ConstRoots
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
			ctx.W.EmitMakeBool(target, *src)
		case tagInt:
			ctx.W.EmitMakeInt(target, *src)
		case tagFloat:
			ctx.W.EmitMakeFloat(target, *src)
		case tagNil:
			ctx.W.EmitMakeNil(target)
		default:
			ptr, aux := src.Imm.RawWords()
			ctx.W.EmitMovRegImm64(target.Reg, uint64(ptr))
			ctx.W.EmitMovRegImm64(target.Reg2, aux)
		}
		return target
	case LocStack, LocStackPair:
		ctx.EnsureDesc(src)
		return jitPlaceIntoPair(ctx, src, target)
	case LocRegPair:
		if src.Reg != target.Reg {
			ctx.W.emitMovRegReg(target.Reg, src.Reg)
		}
		if src.Reg2 != target.Reg2 {
			ctx.W.emitMovRegReg(target.Reg2, src.Reg2)
		}
		if src.Reg != target.Reg && src.Reg2 != target.Reg2 {
			ctx.FreeDesc(src)
		}
		return target
	case LocReg:
		switch src.Type {
		case tagBool:
			ctx.W.EmitMakeBool(target, *src)
		case tagInt:
			ctx.W.EmitMakeInt(target, *src)
		case tagFloat:
			ctx.W.EmitMakeFloat(target, *src)
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
			ctx.W.emitMovRegReg(dst, valReg)
		}
		if cond.Type == tagFloat {
			mask := ctx.AllocReg()
			ctx.W.EmitMovRegImm64(mask, 0x7fffffffffffffff)
			ctx.W.EmitAndInt64(dst, mask)
			ctx.FreeReg(mask)
		} else if cond.Type == tagBool {
			// Bool payload is auxVal in bits [63:8]; low 8 bits hold the tag.
			ctx.W.EmitShrRegImm8(dst, 8)
		}
		ctx.W.EmitCmpRegImm32(dst, 0)
		ctx.W.EmitSetcc(dst, CcNE)
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
	ctx.W.EmitAndRegImm32(out.Reg, 1)
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
		ctx.W.EmitAndRegImm32(out.Reg, 1)
		out.Type = tagBool
		return out
	}
	tagReg := ctx.AllocReg()
	ctx.W.emitGetTagRegs(tagReg, tmp.Reg, tmp.Reg2)
	ctx.W.EmitCmpRegImm8(tagReg, uint8(tagNil))
	ctx.W.EmitSetcc(tagReg, CcE)
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

func jitChildContext(parent *JITContext) *JITContext {
	return &JITContext{
		Env:         parent.Env,
		FreeRegs:    parent.FreeRegs,
		AllRegs:     parent.AllRegs,
		W:           parent.W,
		StackOffset: parent.StackOffset,
		SliceBase:   parent.SliceBase,
		ConstRoots:  parent.ConstRoots,
		rootSet:     parent.rootSet,
	}
}

func jitMergeChildRoots(parent *JITContext, child *JITContext) {
	if child == nil {
		return
	}
	parent.ConstRoots = child.ConstRoots
	parent.rootSet = child.rootSet
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
					ctx.W.EmitJmp(trueLbl)
					return
				}
				for i := 1; i < len(list)-1; i++ {
					nextLbl := ctx.W.ReserveLabel()
					jitEmitCondJump(ctx, list[i], sliceBase, nextLbl, falseLbl)
					ctx.W.MarkLabel(nextLbl)
				}
				jitEmitCondJump(ctx, list[len(list)-1], sliceBase, trueLbl, falseLbl)
				return
			case "or":
				// Eval semantics: (or) => false
				if len(list) <= 1 {
					ctx.W.EmitJmp(falseLbl)
					return
				}
				for i := 1; i < len(list)-1; i++ {
					nextLbl := ctx.W.ReserveLabel()
					jitEmitCondJump(ctx, list[i], sliceBase, trueLbl, nextLbl)
					ctx.W.MarkLabel(nextLbl)
				}
				jitEmitCondJump(ctx, list[len(list)-1], sliceBase, trueLbl, falseLbl)
				return
			case "if":
				// Eval semantics: chain of condition/value pairs plus optional else.
				i := 1
				for i+1 < len(list) {
					thenCondLbl := ctx.W.ReserveLabel()
					nextCondLbl := ctx.W.ReserveLabel()
					jitEmitCondJump(ctx, list[i], sliceBase, thenCondLbl, nextCondLbl)
					ctx.W.MarkLabel(thenCondLbl)
					jitEmitCondJump(ctx, list[i+1], sliceBase, trueLbl, falseLbl)
					ctx.W.MarkLabel(nextCondLbl)
					i += 2
				}
				if i < len(list) {
					jitEmitCondJump(ctx, list[i], sliceBase, trueLbl, falseLbl)
				} else {
					// No else branch => nil => false
					ctx.W.EmitJmp(falseLbl)
				}
				return
			}
		}
	}

	cond := jitCompileExpr(ctx, expr, sliceBase, JITValueDesc{Loc: LocAny})
	b := jitCondToBool(ctx, &cond)
	if b.Loc == LocImm {
		if b.Imm.Bool() {
			ctx.W.EmitJmp(trueLbl)
		} else {
			ctx.W.EmitJmp(falseLbl)
		}
		return
	}
	ctx.W.EmitCmpRegImm32(b.Reg, 0)
	ctx.W.EmitJcc(CcNE, trueLbl)
	ctx.W.EmitJmp(falseLbl)
	ctx.FreeDesc(&b)
}

// jitCompileExpr recursively compiles a Scheme expression to machine code.
// sliceBase is the GPR holding the variadic args slice pointer.
// result tells the emitter where to place the output.
// Panics on unsupported expressions (caught by jitCompileExprBody).
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
					ctx.W.EmitMovRegImm64(result.Reg, uint64(ptr))
					ctx.W.EmitMovRegImm64(result.Reg2, aux)
					d := JITValueDesc{Loc: LocRegPair, Type: src.Type, Reg: result.Reg, Reg2: result.Reg2}
					ctx.BindReg(result.Reg, &d)
					ctx.BindReg(result.Reg2, &d)
					return d
				case LocRegPair:
					ctx.EnsureDesc(&src)
					if src.Reg != result.Reg {
						ctx.W.emitMovRegReg(result.Reg, src.Reg)
					}
					if src.Reg2 != result.Reg2 {
						ctx.W.emitMovRegReg(result.Reg2, src.Reg2)
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
				ctx.W.emitMovRegReg(r, src.Reg)
				d := JITValueDesc{Loc: LocReg, Type: src.Type, Reg: r}
				ctx.BindReg(r, &d)
				return d
			case LocRegPair:
				r1 := ctx.AllocRegExcept(src.Reg, src.Reg2)
				r2 := ctx.AllocRegExcept(src.Reg, src.Reg2, r1)
				ctx.W.emitMovRegReg(r1, src.Reg)
				ctx.W.emitMovRegReg(r2, src.Reg2)
				d := JITValueDesc{Loc: LocRegPair, Type: src.Type, Reg: r1, Reg2: r2}
				ctx.BindReg(r1, &d)
				ctx.BindReg(r2, &d)
				return d
			}
		}
		if result.Loc == LocRegPair {
			ctx.W.EmitLoadArgPair(result.Reg, result.Reg2, sliceBase, idx)
			d := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: result.Reg, Reg2: result.Reg2}
			ctx.BindReg(result.Reg, &d)
			ctx.BindReg(result.Reg2, &d)
			return d
		}
		// Fallback: load from args slice: ptr at [base+i*16], aux at [base+i*16+8]
		ptrReg := ctx.AllocReg()
		auxReg := ctx.AllocReg()
		ctx.W.emitMovRegMem(ptrReg, sliceBase, int32(idx*16))
		ctx.W.emitMovRegMem(auxReg, sliceBase, int32(idx*16+8))
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
							ctx.W.MarkLabel(endLbl)
						}
						ctx.BindReg(target.Reg, &target)
						ctx.BindReg(target.Reg2, &target)
						return target
					}
					i += 2
					continue
				}
				if !hasDynamic {
					endLbl = ctx.W.ReserveLabel()
					hasDynamic = true
				}
				nextCondLbl := ctx.W.ReserveLabel()
				ctx.W.EmitCmpRegImm32(b.Reg, 0)
				ctx.W.EmitJcc(CcE, nextCondLbl)
				ctx.FreeDesc(&b)
				thenVal := jitCompileExpr(ctx, list[i+1], sliceBase, target)
				_ = jitPlaceIntoPair(ctx, &thenVal, target)
				ctx.W.EmitJmp(endLbl)
				ctx.W.MarkLabel(nextCondLbl)
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
				ctx.W.MarkLabel(endLbl)
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
					falseLbl = ctx.W.ReserveLabel()
					endLbl = ctx.W.ReserveLabel()
					hasDynamic = true
				}
				ctx.W.EmitCmpRegImm32(b.Reg, 0)
				ctx.W.EmitJcc(CcE, falseLbl)
				ctx.FreeDesc(&b)
			}
			if compileTimeFalse {
				if hasDynamic {
					ctx.W.MarkLabel(falseLbl)
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
			ctx.W.EmitJmp(endLbl)
			ctx.W.MarkLabel(falseLbl)
			falseDesc := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
			_ = jitPlaceIntoPair(ctx, &falseDesc, target)
			ctx.W.MarkLabel(endLbl)
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
					trueLbl = ctx.W.ReserveLabel()
					endLbl = ctx.W.ReserveLabel()
					hasDynamic = true
				}
				ctx.W.EmitCmpRegImm32(b.Reg, 0)
				ctx.W.EmitJcc(CcNE, trueLbl)
				ctx.FreeDesc(&b)
			}
			if compileTimeTrue {
				if hasDynamic {
					ctx.W.MarkLabel(trueLbl)
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
			ctx.W.EmitJmp(endLbl)
			ctx.W.MarkLabel(trueLbl)
			trueDesc := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
			_ = jitPlaceIntoPair(ctx, &trueDesc, target)
			ctx.W.MarkLabel(endLbl)
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
			endLbl := ctx.W.ReserveLabel()
			for i := 1; i < len(list); i++ {
				v := jitCompileExpr(ctx, list[i], sliceBase, JITValueDesc{Loc: LocAny})
				if i == len(list)-1 {
					_ = jitPlaceIntoPair(ctx, &v, target)
					break
				}
				if v.Loc == LocImm {
					if v.Imm.Bool() {
						_ = jitPlaceIntoPair(ctx, &v, target)
						ctx.W.EmitJmp(endLbl)
						break
					}
					continue
				}
				b := jitCondToBoolBorrowed(ctx, &v)
				if b.Loc == LocImm {
					if b.Imm.Bool() {
						_ = jitPlaceIntoPair(ctx, &v, target)
						ctx.W.EmitJmp(endLbl)
					}
					ctx.FreeDesc(&v)
					continue
				}
				takeLbl := ctx.W.ReserveLabel()
				nextLbl := ctx.W.ReserveLabel()
				ctx.W.EmitCmpRegImm32(b.Reg, 0)
				ctx.W.EmitJcc(CcNE, takeLbl)
				ctx.W.EmitJmp(nextLbl)
				ctx.W.MarkLabel(takeLbl)
				_ = jitPlaceIntoPair(ctx, &v, target)
				ctx.W.EmitJmp(endLbl)
				ctx.W.MarkLabel(nextLbl)
				ctx.FreeDesc(&b)
				ctx.FreeDesc(&v)
			}
			ctx.W.MarkLabel(endLbl)
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
			endLbl := ctx.W.ReserveLabel()
			for i := 1; i < len(list); i++ {
				v := jitCompileExpr(ctx, list[i], sliceBase, JITValueDesc{Loc: LocAny})
				if v.Loc == LocImm {
					if !v.Imm.IsNil() {
						_ = jitPlaceIntoPair(ctx, &v, target)
						ctx.W.EmitJmp(endLbl)
						break
					}
					continue
				}
				isNil := jitIsNilBorrowed(ctx, &v)
				if isNil.Loc == LocImm {
					if !isNil.Imm.Bool() {
						_ = jitPlaceIntoPair(ctx, &v, target)
						ctx.W.EmitJmp(endLbl)
					}
					ctx.FreeDesc(&v)
					continue
				}
				takeLbl := ctx.W.ReserveLabel()
				nextLbl := ctx.W.ReserveLabel()
				ctx.W.EmitCmpRegImm32(isNil.Reg, 0)
				ctx.W.EmitJcc(CcE, takeLbl) // isNil == 0 => take value
				ctx.W.EmitJmp(nextLbl)
				ctx.W.MarkLabel(takeLbl)
				_ = jitPlaceIntoPair(ctx, &v, target)
				ctx.W.EmitJmp(endLbl)
				ctx.W.MarkLabel(nextLbl)
				ctx.FreeDesc(&isNil)
				ctx.FreeDesc(&v)
			}
			ctx.W.MarkLabel(endLbl)
			ctx.BindReg(target.Reg, &target)
			ctx.BindReg(target.Reg2, &target)
			return target
		}
		decl, ok := declarations[name]
		if !ok || decl.JITEmit == nil {
			panic("jit: no JITEmit for " + name)
		}
		// Compile arguments (intermediate results use LocAny).
		// Use a stack-allocated buffer for the common case of ≤8 args;
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
			// Keep argument descriptors tracked while compiling later args and
			// inside the callee JITEmit body. Without rebinding to args[] slots,
			// register spills/reuse can leave stale copies in args and break
			// non-commutative operators (e.g. subtraction).
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
		// Keep call arguments out of compile-time spill slots (MemPtr-backed
		// descriptors), because such addresses would be baked into emitted code.
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
		0xC3, // RET
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
		// Build aux = (bool&1)<<8 | tagBool.
		// Keep it branchless so callers can feed arbitrary integer predicates.
		// First zero-extend the bool into RBX.
		w.emitMovRegReg(RegRBX, src.Reg)
		w.emitBytes(0x48, 0x81, 0xE3) // AND RBX, 0x01
		w.emitU32(1)
		w.EmitShlRegImm8(RegRBX, 8)
		// MOV RCX, tagBool
		w.emitBytes(0x48, 0xB9) // MOV RCX, imm64
		w.emitU64(uint64(tagBool))
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
		// dst.Reg2 = ((src.Reg & 1) << 8) | tagBool
		if dst.Reg2 != src.Reg {
			w.emitMovRegReg(dst.Reg2, src.Reg)
		}
		w.emitAndRegImm32(dst.Reg2, 1)
		w.EmitShlRegImm8(dst.Reg2, 8)
		w.EmitMovRegImm64(RegR11, uint64(tagBool))
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
		w.EmitMovRegImm64(dst.Reg2, math.Float64bits(src.Imm.Float())) // float bits stored in aux
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
	w.emitMovqGprToXmm(RegX0, dst)
	w.emitMovqGprToXmm(RegX1, src)
	w.emitSseOp(0x58, RegX0, RegX1) // ADDSD
	w.emitMovqXmmToGpr(dst, RegX0)
}

// EmitSubFloat64 emits: SUBSD dst, src (XMM -= XMM)
func (w *JITWriter) EmitSubFloat64(dst, src Reg) {
	w.emitMovqGprToXmm(RegX0, dst)
	w.emitMovqGprToXmm(RegX1, src)
	w.emitSseOp(0x5C, RegX0, RegX1) // SUBSD
	w.emitMovqXmmToGpr(dst, RegX0)
}

// EmitMulFloat64 emits: MULSD dst, src (XMM *= XMM)
func (w *JITWriter) EmitMulFloat64(dst, src Reg) {
	w.emitMovqGprToXmm(RegX0, dst)
	w.emitMovqGprToXmm(RegX1, src)
	w.emitSseOp(0x59, RegX0, RegX1) // MULSD
	w.emitMovqXmmToGpr(dst, RegX0)
}

// EmitDivFloat64 emits: DIVSD dst, src (XMM /= XMM)
func (w *JITWriter) EmitDivFloat64(dst, src Reg) {
	w.emitMovqGprToXmm(RegX0, dst)
	w.emitMovqGprToXmm(RegX1, src)
	w.emitSseOp(0x5E, RegX0, RegX1) // DIVSD
	w.emitMovqXmmToGpr(dst, RegX0)
}

// EmitCmpFloat64Setcc compares two float64 bit-patterns from GPRs and writes
// 0/1 into dst using SETcc on the floating-point flags.
func (w *JITWriter) EmitCmpFloat64Setcc(dst, left, right Reg, cc byte) {
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
	w.emitMovqGprToXmm(RegX0, left)
	w.emitMovqGprToXmm(RegX1, right)
	// UCOMISD XMM0, XMM1
	w.emitBytes(0x66, 0x0F, 0x2E, 0xC1)
	w.EmitSetcc(dst, cc)
}

// --- Conversion emitters ---

// EmitCvtInt64ToFloat64 converts an int64 in gprSrc to float64 bits in gprSrc.
// Uses the XMM register corresponding to gprSrc as scratch:
//
//	CVTSI2SDQ xmm(gprSrc), gprSrc   — int64 → float64 in XMM
//	MOVQ      gprSrc, xmm(gprSrc)   — extract float64 bits back to GPR
func (w *JITWriter) EmitCvtInt64ToFloat64(xmmDst, gprSrc Reg) {
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
	w.emitBytes(0xF2, rex, 0x0F, 0x2A, modrm)
	// MOVQ xmm → gpr (66 REX.W 0F 7E /r) — extract float64 bits to GPR
	w.emitBytes(0x66, rex, 0x0F, 0x7E, modrm)
}

// EmitCvtFloatBitsToInt64 converts raw float64 bits in gprSrc to int64 in dst.
// Uses XMM0 as scratch:
//
//	MOVQ XMM0, gprSrc
//	CVTTSD2SI dst, XMM0
func (w *JITWriter) EmitCvtFloatBitsToInt64(dst, gprSrc Reg) {
	w.emitMovqGprToXmm(RegX0, gprSrc)
	xmm := RegX0 - 16
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x04 // REX.R
	}
	if xmm >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | (byte(dst&7) << 3) | byte(xmm&7)
	w.emitBytes(0xF2, rex, 0x0F, 0x2C, modrm)
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

// EmitLoadArgPair loads the idx-th Scmer (ptr+aux pair) from the args slice.
func (w *JITWriter) EmitLoadArgPair(dstPtr, dstAux, sliceBase Reg, idx int) {
	w.emitMovRegMem(dstPtr, sliceBase, int32(idx*16))   // ptr field
	w.emitMovRegMem(dstAux, sliceBase, int32(idx*16+8)) // aux field
}

// EmitByte emits a single byte (exported for test harnesses).
func (w *JITWriter) EmitByte(b byte) {
	w.emitByte(b)
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

// EmitJmpToPos emits an unconditional JMP rel32 to an already-known code position.
func (w *JITWriter) EmitJmpToPos(targetPos int32) {
	curPos := int32(uintptr(w.Ptr)-uintptr(w.Start)) + 5
	off := targetPos - curPos
	w.emitByte(0xE9) // JMP rel32
	w.emitU32(uint32(off))
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
func (w *JITWriter) EmitMovRegReg(dst, src Reg) {
	if dst == src {
		return
	}
	w.emitMovRegReg(dst, src)
}

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

// EmitMovRegImm64 loads an immediate into a 64-bit register using the
// shortest encoding: XOR reg,reg (2-3 B) for 0, MOV r32,imm32 (5-6 B)
// for values ≤ 0xFFFFFFFF, or full MOV r64,imm64 (10 B) otherwise.
func (w *JITWriter) EmitMovRegImm64(dst Reg, imm uint64) {
	dstEnc := byte(dst & 7)
	if imm == 0 {
		// XOR r32, r32 — zero-extends to 64 bits (2 or 3 bytes)
		if dst >= 8 {
			w.EmitByte(0x45) // REX.R + REX.B
		}
		w.emitBytes(0x31, 0xC0|(dstEnc<<3)|dstEnc)
		return
	}
	if imm <= 0xFFFFFFFF {
		// MOV r32, imm32 — zero-extends to 64 bits (5 or 6 bytes)
		if dst >= 8 {
			w.EmitByte(0x41) // REX.B
		}
		w.EmitByte(0xB8 | dstEnc)
		w.emitU32(uint32(imm))
		return
	}
	// Full MOV r64, imm64 (10 bytes)
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	w.emitBytes(rex, 0xB8|dstEnc)
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

// EmitMovRegMem emits MOV dst, [base + disp32] (load 64-bit from memory) — exported wrapper.
func (w *JITWriter) EmitMovRegMem(dst, base Reg, disp int32) {
	w.emitMovRegMem(dst, base, disp)
}

// EmitMovRegMemB emits MOVZX dst, byte [base + disp32] (8-bit zero-extended load).
func (w *JITWriter) EmitMovRegMemB(dst, base Reg, disp int32) {
	w.emitRegMemOp2(0x0F, 0xB6, dst, base, disp)
}

// EmitMovRegMemW emits MOVZX dst, word [base + disp32] (16-bit zero-extended load).
func (w *JITWriter) EmitMovRegMemW(dst, base Reg, disp int32) {
	w.emitRegMemOp2(0x0F, 0xB7, dst, base, disp)
}

// EmitMovRegMemL emits MOV r32, [base + disp32] (32-bit zero-extended load).
func (w *JITWriter) EmitMovRegMemL(dst, base Reg, disp int32) {
	w.emitRegMemOp32(0x8B, dst, base, disp)
}

// EmitLeaRegMem emits LEA dst, [base + disp32] (compute address, no memory access)
// For IndexAddr: LEA dst, [sliceBase + idx*16] computes &a[idx]
func (w *JITWriter) EmitLeaRegMem(dst, base Reg, disp int32) {
	w.emitRegMemOp(0x8D, dst, base, disp)
}

// EmitMovRegMem64 loads a 64-bit value from an absolute memory address into dst.
// Uses dst itself as scratch for the address (avoids clobbering R11).
func (w *JITWriter) EmitMovRegMem64(dst Reg, addr uintptr) {
	w.EmitMovRegImm64(dst, uint64(addr))
	w.emitMovRegMem(dst, dst, 0)
}

// EmitMovRegMem32 loads a 32-bit value (zero-extended to 64 bits) from an absolute address.
// Uses dst itself as scratch for the address (avoids clobbering R11).
func (w *JITWriter) EmitMovRegMem32(dst Reg, addr uintptr) {
	w.EmitMovRegImm64(dst, uint64(addr))
	// MOV r32, [dst+0] — 32-bit load zero-extends to 64 bits (no REX.W)
	w.emitRegMemOp32(0x8B, dst, dst, 0)
}

// EmitMovRegMem8 loads a byte (zero-extended to 64 bits) from an absolute address.
// Uses dst itself as scratch for the address (avoids clobbering R11).
func (w *JITWriter) EmitMovRegMem8(dst Reg, addr uintptr) {
	w.EmitMovRegImm64(dst, uint64(addr))
	// MOVZX r64, byte [dst+0]
	w.emitRegMemOp2(0x0F, 0xB6, dst, dst, 0)
}

// EmitMovRegMem16 loads a 16-bit value (zero-extended to 64 bits) from an absolute address.
// Uses dst itself as scratch for the address (avoids clobbering R11).
func (w *JITWriter) EmitMovRegMem16(dst Reg, addr uintptr) {
	w.EmitMovRegImm64(dst, uint64(addr))
	// MOVZX r64, word [dst+0]
	w.emitRegMemOp2(0x0F, 0xB7, dst, dst, 0)
}

// emitRegMemOp32 emits a 32-bit register-memory operation (no REX.W, for zero-extending loads).
func (w *JITWriter) emitRegMemOp32(opcode byte, dst, base Reg, disp int32) {
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
				w.emitBytes(rex, opcode, modrm, 0x24)
			} else {
				w.emitBytes(rex, opcode, modrm)
			}
		} else {
			if baseEnc == 4 {
				w.emitBytes(opcode, modrm, 0x24)
			} else {
				w.emitBytes(opcode, modrm)
			}
		}
	} else if disp >= -128 && disp <= 127 {
		modrm := 0x40 | (dstEnc << 3) | baseEnc
		if needRex {
			if baseEnc == 4 {
				w.emitBytes(rex, opcode, modrm, 0x24, byte(int8(disp)))
			} else {
				w.emitBytes(rex, opcode, modrm, byte(int8(disp)))
			}
		} else {
			if baseEnc == 4 {
				w.emitBytes(opcode, modrm, 0x24, byte(int8(disp)))
			} else {
				w.emitBytes(opcode, modrm, byte(int8(disp)))
			}
		}
	} else {
		modrm := 0x80 | (dstEnc << 3) | baseEnc
		if needRex {
			if baseEnc == 4 {
				w.emitBytes(rex, opcode, modrm, 0x24)
			} else {
				w.emitBytes(rex, opcode, modrm)
			}
		} else {
			if baseEnc == 4 {
				w.emitBytes(opcode, modrm, 0x24)
			} else {
				w.emitBytes(opcode, modrm)
			}
		}
		w.emitU32(uint32(disp))
	}
}

// emitRegMemOp2 emits a 2-byte opcode register-memory operation with REX.W (for MOVZX etc.).
func (w *JITWriter) emitRegMemOp2(op1, op2 byte, dst, base Reg, disp int32) {
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
			w.emitBytes(rex, op1, op2, modrm, 0x24)
		} else {
			w.emitBytes(rex, op1, op2, modrm)
		}
	} else if disp >= -128 && disp <= 127 {
		modrm := 0x40 | (dstEnc << 3) | baseEnc
		if baseEnc == 4 {
			w.emitBytes(rex, op1, op2, modrm, 0x24, byte(int8(disp)))
		} else {
			w.emitBytes(rex, op1, op2, modrm, byte(int8(disp)))
		}
	} else {
		modrm := 0x80 | (dstEnc << 3) | baseEnc
		if baseEnc == 4 {
			w.emitBytes(rex, op1, op2, modrm, 0x24)
		} else {
			w.emitBytes(rex, op1, op2, modrm)
		}
		w.emitU32(uint32(disp))
	}
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

// emitMovqGprToXmm emits MOVQ xmmDst, gprSrc (66 REX.W 0F 6E /r)
func (w *JITWriter) emitMovqGprToXmm(xmm, gpr Reg) {
	x := xmm - 16
	rex := byte(0x48)
	if x >= 8 {
		rex |= 0x04 // REX.R
	}
	if gpr >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | (byte(x&7) << 3) | byte(gpr&7)
	w.emitBytes(0x66, rex, 0x0F, 0x6E, modrm)
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

// EmitCmpRegImm8 emits CMP r8, imm8 on the low byte of the register.
// This is used for compact Scmer tag checks where tags live in aux[7:0].
func (w *JITWriter) EmitCmpRegImm8(dst Reg, imm uint8) {
	rex := byte(0x40) // force low-byte register encoding (incl. SIL/DIL/BPL/SPL)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xF8) | byte(dst&7) // /7 = CMP, mod=11, r/m=dst
	w.emitBytes(rex, 0x80, modrm, imm)
}

// EmitAddRegImm32 emits ADD r64, sign-extended imm32.
func (w *JITWriter) EmitAddRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC0) | byte(dst&7) // /0 = ADD
	w.emitBytes(rex, 0x81, modrm)
	w.emitU32(uint32(imm))
}

// EmitSubRegImm32 emits SUB r64, sign-extended imm32.
func (w *JITWriter) EmitSubRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xE8) | byte(dst&7) // /5 = SUB
	w.emitBytes(rex, 0x81, modrm)
	w.emitU32(uint32(imm))
}

// EmitOrRegImm32 emits OR r64, sign-extended imm32.
func (w *JITWriter) EmitOrRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xC8) | byte(dst&7) // /1 = OR
	w.emitBytes(rex, 0x81, modrm)
	w.emitU32(uint32(imm))
}

// EmitImulRegImm32 emits IMUL r64, r64, imm32.
func (w *JITWriter) EmitImulRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x05 // REX.R | REX.B (reg and r/m are both dst)
	}
	modrm := byte(0xC0) | (byte(dst&7) << 3) | byte(dst&7)
	w.emitBytes(rex, 0x69, modrm)
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

// EmitSarRegImm8 emits SAR r64, imm8 (arithmetic shift right by immediate)
func (w *JITWriter) EmitSarRegImm8(dst Reg, imm uint8) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xF8) | byte(dst&7) // /7 = SAR
	w.emitBytes(rex, 0xC1, modrm, imm)
}

// EmitShlRegCl emits SHL r64, CL (shift left by variable amount in CL register)
func (w *JITWriter) EmitShlRegCl(dst Reg) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xE0) | byte(dst&7) // /4 = SHL
	w.emitBytes(rex, 0xD3, modrm)
}

// EmitShrRegCl emits SHR r64, CL (shift right by variable amount in CL register)
func (w *JITWriter) EmitShrRegCl(dst Reg) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xE8) | byte(dst&7) // /5 = SHR
	w.emitBytes(rex, 0xD3, modrm)
}

// EmitAndRegImm32 emits AND r64, imm32 (sign-extended)
func (w *JITWriter) EmitAndRegImm32(dst Reg, imm int32) {
	rex := byte(0x48)
	if dst >= 8 {
		rex |= 0x01 // REX.B
	}
	modrm := byte(0xE0) | byte(dst&7) // /4 = AND
	w.emitBytes(rex, 0x81, modrm)
	w.emitU32(uint32(imm))
}

// EmitOrInt64 emits OR dst, src (64-bit OR)
func (w *JITWriter) EmitOrInt64(dst, src Reg) {
	w.emitAluRegReg(0x09, dst, src) // OR r/m64, r64
}

// EmitAndInt64 emits AND dst, src (64-bit AND)
func (w *JITWriter) EmitAndInt64(dst, src Reg) {
	w.emitAluRegReg(0x21, dst, src) // AND r/m64, r64
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
	// Dynamic type: materialize spilled descriptors before reading Reg/Reg2.
	ctx.EnsureDesc(src)
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
// Equivalent to GetTag(src) == tag. Consumes src.
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
	// Dynamic type: materialize spilled descriptors before reading Reg/Reg2.
	ctx.EnsureDesc(src)
	tagReg := ctx.AllocReg()
	ctx.W.emitGetTagRegs(tagReg, src.Reg, src.Reg2)
	ctx.FreeDesc(src)
	ctx.W.EmitCmpRegImm8(tagReg, uint8(tag))
	ctx.W.EmitSetcc(tagReg, CcE)
	r := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: tagReg}
	if result.Loc == LocAny {
		return r
	}
	ctx.W.EmitMakeBool(result, r)
	ctx.FreeReg(tagReg)
	return result
}

// EmitTagEqualsBorrowed checks if a Scmer's tag equals a constant without
// consuming/clobbering the source descriptor. This is required when the same
// SSA value is used both for a type predicate and later value extraction.
func (ctx *JITContext) EmitTagEqualsBorrowed(src *JITValueDesc, tag uint16, result JITValueDesc) JITValueDesc {
	emitOut := func(v JITValueDesc) JITValueDesc {
		if result.Loc == LocAny {
			return v
		}
		ctx.W.EmitMakeBool(result, v)
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
		ctx.W.emitGetTagRegs(tagReg, src.Reg, src.Reg2)
		ctx.W.EmitCmpRegImm8(tagReg, uint8(tag))
		ctx.W.EmitSetcc(tagReg, CcE)
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
		ctx.W.EmitMakeBool(result, v)
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
			ctx.W.emitMovRegReg(dst, valReg)
		}

		if src.Type == tagFloat {
			// Float truthiness is float64(bits) != 0.0. Mask sign bit so -0.0
			// becomes zero, then compare against zero.
			mask := ctx.AllocReg()
			ctx.W.EmitMovRegImm64(mask, 0x7fffffffffffffff)
			ctx.W.EmitAndInt64(dst, mask)
			ctx.FreeReg(mask)
		} else if src.Type == tagBool {
			// Bool payload is auxVal in bits [63:8]; low 8 bits hold the tag.
			ctx.W.EmitShrRegImm8(dst, 8)
		}
		ctx.W.EmitCmpRegImm32(dst, 0)
		ctx.W.EmitSetcc(dst, CcNE)

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
	ctx.W.EmitAndRegImm32(out.Reg, 1)
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
//
//	if ptr == &scmerFloatSentinel → tagFloat (3)
//	else → aux & 0xFF
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

	// Default: dst = aux & 0xFF
	if dst != auxReg {
		w.emitMovRegReg(dst, auxReg)
	}
	w.EmitAndRegImm32(dst, 0xFF)
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

// --- PUSH/POP/CALL ---

// EmitPushReg emits PUSH r64
func (w *JITWriter) EmitPushReg(r Reg) {
	if r >= 8 {
		w.emitBytes(0x41, 0x50|byte(r&7))
	} else {
		w.emitByte(0x50 | byte(r))
	}
}

// EmitPopReg emits POP r64
func (w *JITWriter) EmitPopReg(r Reg) {
	if r >= 8 {
		w.emitBytes(0x41, 0x58|byte(r&7))
	} else {
		w.emitByte(0x58 | byte(r))
	}
}

// EmitCallIndirect emits MOV R11, imm64; CALL R11
func (w *JITWriter) EmitCallIndirect(addr uint64) {
	w.EmitMovRegImm64(RegR11, addr)
	w.emitBytes(0x41, 0xFF, 0xD3) // CALL R11
}

// emitStoreRegMem emits MOV [base + disp], src (store 64-bit register to memory)
func (w *JITWriter) emitStoreRegMem(src, base Reg, disp int32) {
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
			w.emitBytes(rex, 0x89, modrm, 0x24)
		} else {
			w.emitBytes(rex, 0x89, modrm)
		}
	} else if disp >= -128 && disp <= 127 {
		modrm := 0x40 | (srcEnc << 3) | baseEnc
		if baseEnc == 4 {
			w.emitBytes(rex, 0x89, modrm, 0x24, byte(int8(disp)))
		} else {
			w.emitBytes(rex, 0x89, modrm, byte(int8(disp)))
		}
	} else {
		modrm := 0x80 | (srcEnc << 3) | baseEnc
		if baseEnc == 4 {
			w.emitBytes(rex, 0x89, modrm, 0x24)
		} else {
			w.emitBytes(rex, 0x89, modrm)
		}
		w.emitU32(uint32(disp))
	}
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

// EmitStoreRegMem is the exported version of emitStoreRegMem:
// MOV [base+disp], src (64-bit store).
func (w *JITWriter) EmitStoreRegMem(src, base Reg, disp int32) {
	w.emitStoreRegMem(src, base, disp)
}

// EmitSubRSP emits SUB RSP, imm8 to reserve stack space.
func (w *JITWriter) EmitSubRSP(n uint8) {
	w.emitBytes(0x48, 0x83, 0xEC, n)
}

// EmitAddRSP emits ADD RSP, imm8 to release stack space.
func (w *JITWriter) EmitAddRSP(n uint8) {
	w.emitBytes(0x48, 0x83, 0xC4, n)
}

// EmitSubRSP32Fixup emits SUB RSP, imm32 with a zero placeholder and returns
// a pointer to the 4-byte immediate so it can be patched later via PatchInt32.
func (w *JITWriter) EmitSubRSP32Fixup() unsafe.Pointer {
	w.emitBytes(0x48, 0x81, 0xEC, 0x00, 0x00, 0x00, 0x00)
	return unsafe.Add(w.Ptr, -4)
}

// PatchInt32 writes a 32-bit little-endian value at the given position.
func (w *JITWriter) PatchInt32(pos unsafe.Pointer, val int32) {
	*(*int32)(pos) = val
}

// EmitAddRSP32 emits ADD RSP, imm32.
func (w *JITWriter) EmitAddRSP32(val int32) {
	w.emitBytes(0x48, 0x81, 0xC4)
	p := w.Ptr
	*(*int32)(p) = val
	w.Ptr = unsafe.Add(p, 4)
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
		ctx.W.EmitMovRegImm64(RegR11, word)
		ctx.W.EmitStoreRegMem(RegR11, RegRSP, disp)
	case LocReg:
		ctx.W.EmitStoreRegMem(src.Reg, RegRSP, disp)
	}
}

// EmitLoadFromStack loads a value from stack slot [RSP+disp] into a register.
// No SpillDepth adjustment needed — spills use a separate buffer, not RSP.
func (ctx *JITContext) EmitLoadFromStack(dst Reg, disp int32) {
	ctx.W.EmitMovRegMem(dst, RegRSP, disp)
}

// EmitStoreScmerToStack stores a full Scmer (16 bytes: ptr at disp, aux at disp+8)
// from a LocRegPair or LocImm descriptor to consecutive stack slots [RSP+disp..RSP+disp+15].
// Uses R11 as scratch for LocImm values.
func (ctx *JITContext) EmitStoreScmerToStack(desc JITValueDesc, disp int32) {
	switch desc.Loc {
	case LocRegPair:
		ctx.W.EmitStoreRegMem(desc.Reg, RegRSP, disp)
		ctx.W.EmitStoreRegMem(desc.Reg2, RegRSP, disp+8)
	case LocImm:
		// Store ptr word
		ctx.W.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(desc.Imm.ptr))))
		ctx.W.EmitStoreRegMem(RegR11, RegRSP, disp)
		// Store aux word
		ctx.W.EmitMovRegImm64(RegR11, desc.Imm.aux)
		ctx.W.EmitStoreRegMem(RegR11, RegRSP, disp+8)
	default:
		panic("jit: EmitStoreScmerToStack: unsupported location")
	}
}
