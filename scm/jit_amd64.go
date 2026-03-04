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

import "fmt"
import "unsafe"

// TODO: create this file for other architectures, too

// all code snippets fill rax+rbx with the return value
func jitReturnLiteral(value Scmer) []byte {
	code := []byte{
		0x48, 0xB8, 7, 0, 0, 0, 0, 0, 0, 0, // mov rax, 7
		0x48, 0xBB, 7, 0, 0, 0, 0, 0, 0, 0, // mov rbx, 7
		0xC3,
	}
	// insert the literal into the immediate values
	*(*unsafe.Pointer)(unsafe.Pointer(&code[2])) = *(*unsafe.Pointer)(unsafe.Pointer(&value))
	*(*unsafe.Pointer)(unsafe.Pointer(&code[12])) = *((*unsafe.Pointer)(unsafe.Add(unsafe.Pointer(&value), 8)))
	return code
}

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
	// Trivial literal returns.
	switch body.GetTag() {
	case tagNil, tagBool, tagInt, tagFloat, tagString:
		return jitReturnLiteral(body), nil
	}
	// Expression compilation via JITEmit
	if body.GetTag() == tagSlice {
		return jitCompileExprBody(body)
	}
	if body.GetTag() == tagNthLocalVar {
		return jitCompileExprBody(body)
	}
	return nil, nil
}

// jitCompileExprBody compiles a Scheme expression body to machine code
// using Declaration.JITEmit callbacks. Returns nil if any sub-expression
// is not JIT-compilable.
func jitCompileExprBody(body Scmer) (code []byte, roots []unsafe.Pointer) {
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

	// Compile body, place result into RAX+RBX (Scmer return registers)
	result := JITValueDesc{Loc: LocRegPair, Reg: RegRAX, Reg2: RegRBX}
	desc := jitCompileExpr(ctx, body, RegR12, result)

	// If result came back as LocImm, materialize into RAX+RBX
	if desc.Loc == LocImm {
		switch desc.Imm.GetTag() {
		case tagBool:
			w.EmitReturnBool(desc)
		case tagInt:
			w.EmitReturnInt(desc)
		case tagFloat:
			w.EmitReturnFloat(desc)
		case tagNil:
			w.EmitReturnNil()
		default:
			return nil, nil
		}
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
			// Bool payload uses bit 0; strip marker bits before truth test.
			ctx.W.EmitAndRegImm32(dst, 1)
		}
		ctx.W.EmitCmpRegImm32(dst, 0)
		ctx.W.EmitSetcc(dst, CcNE)
		ctx.FreeDesc(&tmp)
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
	ctx.W.EmitCmpRegImm32(tagReg, int32(tagNil))
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
			endLbl := ctx.W.ReserveLabel()
			i := 1
			for i+1 < len(list) {
				thenLbl := ctx.W.ReserveLabel()
				nextCondLbl := ctx.W.ReserveLabel()
				jitEmitCondJump(ctx, list[i], sliceBase, thenLbl, nextCondLbl)
				ctx.W.MarkLabel(thenLbl)
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
			ctx.W.MarkLabel(endLbl)
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
			falseLbl := ctx.W.ReserveLabel()
			allTrueLbl := ctx.W.ReserveLabel()
			endLbl := ctx.W.ReserveLabel()
			for i := 1; i < len(list)-1; i++ {
				nextLbl := ctx.W.ReserveLabel()
				jitEmitCondJump(ctx, list[i], sliceBase, nextLbl, falseLbl)
				ctx.W.MarkLabel(nextLbl)
			}
			jitEmitCondJump(ctx, list[len(list)-1], sliceBase, allTrueLbl, falseLbl)
			ctx.W.MarkLabel(allTrueLbl)
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
			trueLbl := ctx.W.ReserveLabel()
			allFalseLbl := ctx.W.ReserveLabel()
			endLbl := ctx.W.ReserveLabel()
			for i := 1; i < len(list)-1; i++ {
				nextLbl := ctx.W.ReserveLabel()
				jitEmitCondJump(ctx, list[i], sliceBase, trueLbl, nextLbl)
				ctx.W.MarkLabel(nextLbl)
			}
			jitEmitCondJump(ctx, list[len(list)-1], sliceBase, trueLbl, allFalseLbl)
			ctx.W.MarkLabel(allFalseLbl)
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
		for i := 1; i < len(list); i++ {
			args[i-1] = jitCompileExpr(ctx, list[i], sliceBase, JITValueDesc{Loc: LocAny})
			// Keep argument descriptors tracked while compiling later args and
			// inside the callee JITEmit body. Without rebinding to args[] slots,
			// register spills/reuse can leave stale copies in args and break
			// non-commutative operators (e.g. subtraction).
			switch args[i-1].Loc {
			case LocReg:
				ctx.BindReg(args[i-1].Reg, &args[i-1])
			case LocRegPair:
				ctx.BindReg(args[i-1].Reg, &args[i-1])
				ctx.BindReg(args[i-1].Reg2, &args[i-1])
			}
		}
		// Arguments can be referenced multiple times inside jitgen-generated CFGs.
		// Keep them as non-owning source descriptors so FreeDesc on temporary
		// copies does not release the shared source placement prematurely.
		for i := range args {
			args[i].ID = 0
		}
		// Call the JITEmit callback
		out := decl.JITEmit(ctx, args, result)
		if out.Loc == LocImm {
			ctx.TrackImm(out.Imm)
		}
		return out
	default:
		panic(fmt.Sprintf("jit: unsupported expression tag=%d expr=%s", expr.GetTag(), SerializeToString(expr, &Globalenv)))
	}
}

func jitStackFrame(size uint8) []byte {
	return []byte{
		0x55,             //push   %rbp
		0x48, 0x89, 0xe5, //mov    %rsp,%rbp
		0x48, 0x83, 0xec, size, //sub    $0x10,%rsp
		// TODO: inner code
		// TODO: getter/setter mov    %rax,0x20(%rsp)
		0x48, 0x83, 0xc4, size, //add    $0x10,%rsp
		0x5d, //pop    %rbp
		0xc3, //ret
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
