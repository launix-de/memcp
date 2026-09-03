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

// jitProcStackPointerOffsets mirrors the GC pointer words of Proc. The layout
// test derives the expected words from reflect.Type, so adding or moving a
// pointer-bearing field cannot silently stale JIT stack maps.
var jitProcStackPointerOffsets = [...]int32{
	int32(unsafe.Offsetof(Proc{}.Params)),
	int32(unsafe.Offsetof(Proc{}.Body)),
	int32(unsafe.Offsetof(Proc{}.En)),
	int32(unsafe.Offsetof(Proc{}.Compiled)),
	int32(unsafe.Offsetof(Proc{}.OptimizerMeta)),
}

func jitSpecialFormList(name string, args []Scmer) []Scmer {
	list := make([]Scmer, len(args)+1)
	list[0] = NewSymbol(name)
	copy(list[1:], args)
	return list
}

func jitEmitSpecialOuter(ctx *JITContext, args []Scmer, _ []JITValueDesc, result JITValueDesc) JITValueDesc {
	depth, validDepth := int64(0), false
	if len(args) == 2 {
		depth, validDepth = outerDepthLiteral(args[0])
	}
	if !validDepth {
		panic("jit: invalid outer reference")
	}
	current := ctx.Env
	for ; depth > 0; depth-- {
		if ctx.Env == nil {
			panic("jit: outer reference exceeds environment depth")
		}
		ctx.Env = ctx.Env.Outer
	}
	if ctx.Env == nil {
		panic("jit: outer reference exceeds environment depth")
	}
	defer func() { ctx.Env = current }()
	return jitCompileExpr(ctx, args[1], ctx.SliceBase, result)
}

func jitEmitSpecialQuote(ctx *JITContext, args []Scmer, _ []JITValueDesc, _ JITValueDesc) JITValueDesc {
	if len(args) == 0 {
		imm := NewNil()
		ctx.TrackImm(imm)
		return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: imm}
	}
	quoted := args[0]
	if quoted.GetTag() == tagSourceInfo {
		quoted = quoted.SourceInfo().value
	}
	ctx.TrackImm(quoted)
	return JITValueDesc{Loc: LocImm, Type: quoted.GetTag(), Imm: quoted}
}

func jitEmitSpecialEval(ctx *JITContext, args []Scmer, _ []JITValueDesc, result JITValueDesc) JITValueDesc {
	if len(args) == 0 {
		panic("jit: eval expects an expression")
	}
	callArgs := make([]Scmer, 0, 1+2*ctx.LocalSlotCount)
	callArgs = append(callArgs, args[0])
	callArgs = append(callArgs, jitRuntimeCaptureArgExprs(ctx)...)
	return jitEmitGoVariadicCallFromExprs(ctx, jitEvalSpecial, callArgs, ctx.SliceBase, result, false)
}

func jitEmitSpecialTime(ctx *JITContext, args []Scmer, _ []JITValueDesc, result JITValueDesc) JITValueDesc {
	if len(args) == 0 {
		panic("jit: time expects an expression")
	}
	body := jitCompileSpecialThunk(ctx, args[0], ctx.SliceBase, JITValueDesc{Loc: LocAny})
	callArgs := []JITValueDesc{body}
	if len(args) > 1 {
		callArgs = append(callArgs, jitCompileSpecialThunk(ctx, args[1], ctx.SliceBase, JITValueDesc{Loc: LocAny}))
	} else {
		callArgs = append(callArgs, JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()})
	}
	return jitEmitGoVariadicCallFromDescs(ctx, jitTimeSpecial, callArgs, result)
}

func jitEmitSpecialDefine(ctx *JITContext, args []Scmer, _ []JITValueDesc, result JITValueDesc) JITValueDesc {
	if len(args) != 2 {
		panic("jit: malformed define/set")
	}
	binding := args[0]
	for binding.IsSourceInfo() {
		binding = binding.SourceInfo().value
	}
	if !binding.IsSymbol() {
		panic("jit: define/set target is not a symbol")
	}
	valueTarget := jitAllocTrackedPair(ctx, JITTypeUnknown)
	previousDefiningSymbol := ctx.DefiningSymbol
	ctx.DefiningSymbol = binding.Symbol()
	value := jitCompileExpr(ctx, args[1], ctx.SliceBase, valueTarget)
	ctx.DefiningSymbol = previousDefiningSymbol
	if value.Loc != LocRegPair || value.Reg != valueTarget.Reg || value.Reg2 != valueTarget.Reg2 {
		ctx.FreeDesc(&valueTarget)
	}
	if value.Loc == LocParserTemplate && value.Parser != nil {
		value.Parser.Name = binding.Symbol()
		if ctx.Env == nil {
			ctx.Env = &JITEnv{Vars: make(map[Symbol]JITValueDesc)}
		}
		if ctx.Env.Vars == nil {
			ctx.Env.Vars = make(map[Symbol]JITValueDesc)
		}
		ctx.Env.Vars[binding.Symbol()] = value
		return value
	}
	ctx.EnsureDesc(&value)
	stored := value
	if stored.Loc != LocRegPair && stored.Loc != LocImm && stored.Loc != LocStackPair && stored.Loc != LocInputPair {
		pair := jitAllocTrackedPair(ctx, stored.Type)
		stored = jitPlaceIntoPair(ctx, &stored, pair)
	}
	off := ctx.AllocStack(16)
	ctx.EmitStoreScmerToStack(stored, off)
	if ctx.Env == nil {
		ctx.Env = &JITEnv{Vars: make(map[Symbol]JITValueDesc)}
	} else if ctx.Env.Vars == nil {
		ctx.Env.Vars = make(map[Symbol]JITValueDesc)
	}
	ctx.Env.Vars[binding.Symbol()] = JITValueDesc{Loc: LocStackPair, Type: stored.Type, StackOff: off, NoHeapPointer: stored.NoHeapPointer, Rooted: true}
	return stored
}

func jitEmitSpecialSetN(ctx *JITContext, args []Scmer, _ []JITValueDesc, result JITValueDesc) JITValueDesc {
	if len(args) != 2 {
		panic("jit: malformed setN")
	}
	targetVar := args[0]
	for targetVar.IsSourceInfo() {
		targetVar = targetVar.SourceInfo().value
	}
	if !targetVar.IsNthLocalVar() {
		panic("jit: setN target is not a numbered local")
	}
	idx := int(targetVar.NthLocalVar())
	if idx < 0 || ctx.Env == nil || idx >= len(ctx.Env.Numbered) {
		panic("jit: setN target outside lexical frame")
	}
	value := jitCompileExpr(ctx, args[1], ctx.SliceBase, result)
	if value.Loc == LocParserTemplate && value.Parser != nil {
		ctx.Env.Numbered[idx] = value
		return value
	}
	if value.Loc != LocRegPair && value.Loc != LocImm && value.Loc != LocStackPair && value.Loc != LocInputPair {
		ctx.EnsureDesc(&value)
		pair := jitAllocTrackedPair(ctx, value.Type)
		value = jitPlaceIntoPair(ctx, &value, pair)
	}
	target := ctx.Env.Numbered[idx]
	ctx.EmitCopyScmerToDesc(&target, &value)
	target.Type = value.Type
	target.NoHeapPointer = value.NoHeapPointer
	target.Rooted = true
	ctx.Env.Numbered[idx] = target
	return jitPlaceScmerIntoTarget(ctx, value, result)
}

func jitEmitSpecialParser(ctx *JITContext, args []Scmer, _ []JITValueDesc, result JITValueDesc) JITValueDesc {
	if len(args) == 0 {
		panic("jit: parser expects syntax")
	}
	generator, whitespace := NewNil(), NewNil()
	ignoreResult := false
	if len(args) > 1 {
		generator = args[1]
		ignoreResult = true
	}
	if len(args) > 2 {
		whitespace = args[2]
	}
	var runtimeOuter *Env
	if ctx.RuntimeEnv.GetTag() == tagAny {
		runtimeOuter, _ = ctx.RuntimeEnv.Any().(*Env)
	}
	return JITValueDesc{Loc: LocParserTemplate, Type: tagParser, Parser: &JITParserTemplate{
		Syntax: args[0], Generator: generator, Whitespace: whitespace,
		IgnoreResult: ignoreResult, Outer: ctx.Env, RuntimeOuter: runtimeOuter,
	}}
}

func jitEmitSpecialOptimizerProcReturn(ctx *JITContext, args []Scmer, _ []JITValueDesc, result JITValueDesc) JITValueDesc {
	if len(args) != 2 {
		panic("jit: optimizer_proc_return expects procedure and return metadata")
	}
	value := jitCompileExpr(ctx, args[0], ctx.SliceBase, JITValueDesc{Loc: LocAny})
	metadata := jitCompileSpecialThunk(ctx, args[1], ctx.SliceBase, JITValueDesc{Loc: LocAny})
	return jitEmitGoVariadicCallFromDescs(ctx, jitOptimizerProcReturnSpecial, []JITValueDesc{value, metadata}, result)
}

func jitEmitSpecialBegin(scoped bool, reserve bool) func(*JITContext, []Scmer, []JITValueDesc, JITValueDesc) JITValueDesc {
	return func(ctx *JITContext, args []Scmer, _ []JITValueDesc, result JITValueDesc) JITValueDesc {
		if reserve && len(args) < 2 {
			panic("jit: begin_mut expects a reserve and body")
		}
		if len(args) == 0 {
			imm := NewNil()
			ctx.TrackImm(imm)
			return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: imm}
		}
		body := args
		if reserve {
			reserveExpr := args[0]
			for reserveExpr.IsSourceInfo() {
				reserveExpr = reserveExpr.SourceInfo().value
			}
			if reserveExpr.GetTag() != tagInt && reserveExpr.GetTag() != tagFloat {
				panic("jit: begin_mut reserve must be optimized to a number")
			}
			count := int(ToInt(reserveExpr))
			if count > ctx.LocalSlotCount {
				panic("jit: begin_mut reserve exceeds invocation frame")
			}
			body = args[1:]
		}
		outerEnv := ctx.Env
		if scoped {
			var numbered []JITValueDesc
			if outerEnv != nil {
				numbered = outerEnv.Numbered
			}
			ctx.Env = &JITEnv{Vars: make(map[Symbol]JITValueDesc), Numbered: numbered, Outer: outerEnv}
			defer func() { ctx.Env = outerEnv }()
		}
		for _, form := range body[:len(body)-1] {
			value := jitCompileExpr(ctx, form, ctx.SliceBase, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&value)
		}
		return jitCompileExpr(ctx, body[len(body)-1], ctx.SliceBase, result)
	}
}

func jitEmitSpecialStackList(ctx *JITContext, args []Scmer, _ []JITValueDesc, result JITValueDesc) JITValueDesc {
	return jitCompileStackList(ctx, jitSpecialFormList("!list", args), ctx.SliceBase, result)
}

func jitSameCaptureLocation(left, right JITValueDesc) bool {
	if left.Loc != right.Loc {
		return false
	}
	switch left.Loc {
	case LocInputPair, LocClosurePair, LocStack, LocStackPair, LocStackTriple:
		return left.StackOff == right.StackOff
	case LocReg, LocRegPair, LocRegTriple:
		return left.Reg == right.Reg && left.Reg2 == right.Reg2 && left.Reg3 == right.Reg3
	case LocImm:
		return Equal(left.Imm, right.Imm)
	case LocMem:
		return left.MemPtr == right.MemPtr
	default:
		return false
	}
}

func jitOuterCaptureSymbol(ctx *JITContext, capture jitLambdaOuterCapture) (Symbol, bool) {
	env := ctx.Env
	for depth := 0; depth < capture.depth && env != nil; depth++ {
		env = env.Outer
	}
	index := int(capture.index)
	if env == nil || index < 0 || index >= len(env.Numbered) {
		return "", false
	}
	target := env.Numbered[index]
	var best Symbol
	for symbol, candidate := range env.Vars {
		if !jitSameCaptureLocation(candidate, target) {
			continue
		}
		if symbol == Symbol("session") {
			return symbol, true
		}
		if best == "" || symbol < best {
			best = symbol
		}
	}
	return best, best != ""
}

func jitEmitSpecialReservedList(ctx *JITContext, args []Scmer, _ []JITValueDesc, result JITValueDesc) JITValueDesc {
	var capacityExpr Scmer
	switch {
	case len(args) == 2 && args[0].IsNthLocalVar():
		start := int(args[0].NthLocalVar())
		capacity := int(ToInt(args[1]))
		if capacity < 0 {
			capacity = 0
		}
		if start < 0 || ctx.Env == nil || start+capacity > len(ctx.Env.Numbered) {
			panic("jit: !!list slots outside invocation frame")
		}
		capacityExpr = NewInt(int64(capacity))
	case len(args) == 1:
		capacityExpr = args[0]
	default:
		panic("jit: malformed !!list")
	}
	capacity := jitCompileExpr(ctx, capacityExpr, ctx.SliceBase, JITValueDesc{Loc: LocAny})
	capacityPair := jitAllocTrackedPair(ctx, tagInt)
	capacity = jitPlaceScmerIntoTarget(ctx, capacity, capacityPair)
	target := jitEnsureResultPair(ctx, result)
	out := ctx.EmitGoCallScalarInto(GoFuncAddr(jitMakeReservedList), []JITValueDesc{capacity}, target)
	ctx.FreeDesc(&capacity)
	out.Type = tagSlice
	out.KnownSliceLen = 0
	if capacityExpr.IsInt() {
		out.KnownSliceCap = int32(ToInt(capacityExpr))
		if out.KnownSliceCap < 0 {
			out.KnownSliceCap = 0
		}
		out.SliceSizeKnown = true
	}
	return out
}

func jitEmitSpecialParallel(ctx *JITContext, args []Scmer, _ []JITValueDesc, result JITValueDesc) JITValueDesc {
	thunks := make([]JITValueDesc, 0, len(args))
	for _, child := range args {
		thunks = append(thunks, jitCompileSpecialThunk(ctx, child, ctx.SliceBase, JITValueDesc{Loc: LocAny}))
	}
	return jitEmitGoVariadicCallFromDescs(ctx, jitParallelSpecial, thunks, result)
}

func jitEmitSpecialMatch(name string) func(*JITContext, []Scmer, []JITValueDesc, JITValueDesc) JITValueDesc {
	return func(ctx *JITContext, args []Scmer, _ []JITValueDesc, result JITValueDesc) JITValueDesc {
		return jitCompileMatch(ctx, jitSpecialFormList(name, args), ctx.SliceBase, result)
	}
}

func jitEmitSpecialIf(ctx *JITContext, args []Scmer, _ []JITValueDesc, result JITValueDesc) JITValueDesc {
	if len(args) < 2 {
		imm := NewNil()
		ctx.TrackImm(imm)
		return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: imm}
	}
	target := result
	if target.Loc != LocStackPair {
		target = jitEnsureResultPair(ctx, result)
		ctx.ProtectReg(target.Reg)
		ctx.ProtectReg(target.Reg2)
		defer func() {
			ctx.UnprotectReg(target.Reg2)
			ctx.UnprotectReg(target.Reg)
		}()
	}
	endLabel := ctx.ReserveLabel()
	i := 0
	for i+1 < len(args) {
		thenLabel := ctx.ReserveLabel()
		nextLabel := ctx.ReserveLabel()
		jitEmitCondJump(ctx, args[i], ctx.SliceBase, thenLabel, nextLabel)
		ctx.MarkLabel(thenLabel)
		value := jitCompileExpr(ctx, args[i+1], ctx.SliceBase, target)
		_ = jitPlaceScmerIntoTarget(ctx, value, target)
		ctx.EmitJmp(endLabel)
		ctx.MarkLabel(nextLabel)
		i += 2
	}
	if i < len(args) {
		value := jitCompileExpr(ctx, args[i], ctx.SliceBase, target)
		_ = jitPlaceScmerIntoTarget(ctx, value, target)
	} else {
		nilValue := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
		_ = jitPlaceScmerIntoTarget(ctx, nilValue, target)
	}
	ctx.MarkLabel(endLabel)
	if target.Loc == LocRegPair {
		ctx.BindReg(target.Reg, &target)
		ctx.BindReg(target.Reg2, &target)
	}
	return target
}

func jitEmitSpecialIfCond(ctx *JITContext, args []Scmer, trueLabel, falseLabel JITLabel) {
	i := 0
	for i+1 < len(args) {
		thenLabel := ctx.ReserveLabel()
		nextLabel := ctx.ReserveLabel()
		jitEmitCondJump(ctx, args[i], ctx.SliceBase, thenLabel, nextLabel)
		ctx.MarkLabel(thenLabel)
		jitEmitCondJump(ctx, args[i+1], ctx.SliceBase, trueLabel, falseLabel)
		ctx.MarkLabel(nextLabel)
		i += 2
	}
	if i < len(args) {
		jitEmitCondJump(ctx, args[i], ctx.SliceBase, trueLabel, falseLabel)
	} else {
		ctx.EmitJmp(falseLabel)
	}
}

func jitEmitSpecialBoolFoldCond(takeWhen bool) JITCondEmitter {
	return func(ctx *JITContext, args []Scmer, trueLabel, falseLabel JITLabel) {
		if len(args) == 0 {
			if takeWhen {
				ctx.EmitJmp(falseLabel)
			} else {
				ctx.EmitJmp(trueLabel)
			}
			return
		}
		for i := 0; i < len(args)-1; i++ {
			nextLabel := ctx.ReserveLabel()
			if takeWhen {
				jitEmitCondJump(ctx, args[i], ctx.SliceBase, trueLabel, nextLabel)
			} else {
				jitEmitCondJump(ctx, args[i], ctx.SliceBase, nextLabel, falseLabel)
			}
			ctx.MarkLabel(nextLabel)
		}
		jitEmitCondJump(ctx, args[len(args)-1], ctx.SliceBase, trueLabel, falseLabel)
	}
}

func jitEmitSpecialBoolFold(takeWhen bool) func(*JITContext, []Scmer, []JITValueDesc, JITValueDesc) JITValueDesc {
	return func(ctx *JITContext, args []Scmer, _ []JITValueDesc, result JITValueDesc) JITValueDesc {
		identity := !takeWhen
		if len(args) == 0 {
			imm := NewBool(identity)
			ctx.TrackImm(imm)
			return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: imm}
		}
		unknownOff := ctx.AllocStack(8)
		ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0), NoHeapPointer: true}, unknownOff)
		decisiveLabel := ctx.ReserveLabel()
		unknownLabel := ctx.ReserveLabel()
		endLabel := ctx.ReserveLabel()
		for _, expression := range args {
			value := jitCompileExpr(ctx, expression, ctx.SliceBase, JITValueDesc{Loc: LocAny})
			nilValue := jitIsNilBorrowed(ctx, &value)
			if nilValue.Loc == LocImm && nilValue.Imm.Bool() {
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1), NoHeapPointer: true}, unknownOff)
				ctx.FreeDesc(&value)
				continue
			}

			var nilLabel, nextLabel JITLabel
			hasNilBranch := nilValue.Loc != LocImm
			if hasNilBranch {
				nilLabel = ctx.ReserveLabel()
				nextLabel = ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(nilValue.Reg, 0)
				ctx.EmitJcc(CcNE, nilLabel)
				ctx.FreeDesc(&nilValue)
			}

			boolean := jitCondToBoolBorrowed(ctx, &value)
			ctx.FreeDesc(&value)
			if boolean.Loc == LocImm {
				if boolean.Imm.Bool() == takeWhen {
					ctx.EmitJmp(decisiveLabel)
					break
				}
				if hasNilBranch {
					ctx.EmitJmp(nextLabel)
				}
			} else {
				ctx.EmitCmpRegImm32(boolean.Reg, 0)
				if takeWhen {
					ctx.EmitJcc(CcNE, decisiveLabel)
				} else {
					ctx.EmitJcc(CcE, decisiveLabel)
				}
				ctx.FreeDesc(&boolean)
				if hasNilBranch {
					ctx.EmitJmp(nextLabel)
				}
			}
			ctx.ReclaimUntrackedRegs()
			if hasNilBranch {
				ctx.MarkLabel(nilLabel)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1), NoHeapPointer: true}, unknownOff)
				ctx.MarkLabel(nextLabel)
			}
		}
		unknown := ctx.AllocReg()
		ctx.EmitLoadFromStack(unknown, unknownOff)
		ctx.EmitCmpRegImm32(unknown, 0)
		ctx.EmitJcc(CcNE, unknownLabel)
		ctx.FreeReg(unknown)
		target := jitEnsureResultPair(ctx, result)
		identityValue := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(identity)}
		_ = jitPlaceIntoPair(ctx, &identityValue, target)
		ctx.EmitJmp(endLabel)
		ctx.MarkLabel(decisiveLabel)
		decisiveValue := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(takeWhen)}
		_ = jitPlaceIntoPair(ctx, &decisiveValue, target)
		ctx.EmitJmp(endLabel)
		ctx.MarkLabel(unknownLabel)
		ctx.EmitMakeNil(target)
		ctx.MarkLabel(endLabel)
		ctx.BindReg(target.Reg, &target)
		ctx.BindReg(target.Reg2, &target)
		return target
	}
}

func jitEmitSpecialCoalesce(nilOnly bool) func(*JITContext, []Scmer, []JITValueDesc, JITValueDesc) JITValueDesc {
	return func(ctx *JITContext, args []Scmer, _ []JITValueDesc, result JITValueDesc) JITValueDesc {
		if len(args) == 0 {
			imm := NewNil()
			ctx.TrackImm(imm)
			return JITValueDesc{Loc: LocImm, Type: tagNil, Imm: imm}
		}
		target := JITValueDesc{
			Loc: LocStackPair, Type: JITTypeUnknown,
			StackOff: ctx.AllocStack(16), Rooted: true,
		}
		ctx.PrepareScmerStackTarget(target.StackOff)
		store := func(value *JITValueDesc) {
			ctx.EmitCopyScmerToDesc(&target, value)
			ctx.FreeDesc(value)
		}
		endLabel := ctx.ReserveLabel()
		if nilOnly {
			nilValue := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
			store(&nilValue)
		}
		for i, expression := range args {
			value := jitCompileExpr(ctx, expression, ctx.SliceBase, JITValueDesc{Loc: LocAny})
			if !nilOnly && i == len(args)-1 {
				store(&value)
				break
			}
			if value.Loc == LocImm {
				take := !value.Imm.IsNil()
				if !nilOnly {
					take = value.Imm.Bool()
				}
				if take {
					store(&value)
					ctx.EmitJmp(endLabel)
					break
				}
				continue
			}
			predicate := jitIsNilBorrowed(ctx, &value)
			takeOnZero := true
			if !nilOnly {
				predicate = jitCondToBoolBorrowed(ctx, &value)
				takeOnZero = false
			}
			if predicate.Loc == LocImm {
				take := !predicate.Imm.Bool()
				if !takeOnZero {
					take = predicate.Imm.Bool()
				}
				if take {
					store(&value)
					ctx.EmitJmp(endLabel)
				} else {
					ctx.FreeDesc(&value)
				}
				continue
			}
			takeLabel, nextLabel := ctx.ReserveLabel(), ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(predicate.Reg, 0)
			if takeOnZero {
				ctx.EmitJcc(CcE, takeLabel)
			} else {
				ctx.EmitJcc(CcNE, takeLabel)
			}
			ctx.EmitJmp(nextLabel)
			ctx.MarkLabel(takeLabel)
			store(&value)
			ctx.EmitJmp(endLabel)
			ctx.MarkLabel(nextLabel)
			ctx.FreeDesc(&predicate)
		}
		ctx.MarkLabel(endLabel)
		return jitPlaceScmerIntoTarget(ctx, target, result)
	}
}

func jitEmitSpecialLambda(ctx *JITContext, args []Scmer, _ []JITValueDesc, result JITValueDesc) JITValueDesc {
	if len(args) < 2 {
		panic("jit: lambda expects params and body")
	}
	params := args[0]
	if params.IsSourceInfo() {
		params = params.SourceInfo().value
	}
	body := args[1]
	numVars := 0
	if len(args) > 2 {
		numVars = int(ToInt(args[2]))
	}
	numVars = jitRequiredLocalSlots(body, numVars)
	argExprs := make([]Scmer, 0, 16)
	argExprs = append(argExprs, NewSlice([]Scmer{NewSymbol("quote"), params}))
	argExprs = append(argExprs, NewSlice([]Scmer{NewSymbol("quote"), body}))
	argExprs = append(argExprs, NewInt(int64(numVars)))
	freeSymbols := jitLambdaFreeSymbols(params, body)
	capturedSymbols := make([]Symbol, 0, len(freeSymbols))
	if jitExpressionConsumesRuntimeEnv(body) {
		seen := make(map[Symbol]struct{}, len(freeSymbols))
		for _, symbol := range freeSymbols {
			seen[symbol] = struct{}{}
		}
		for _, symbol := range jitVisibleSymbols(ctx) {
			if _, exists := seen[symbol]; !exists {
				freeSymbols = append(freeSymbols, symbol)
				seen[symbol] = struct{}{}
			}
		}
	}
	for _, symbol := range freeSymbols {
		if ctx.Env != nil {
			if _, ok := ctx.Env.Lookup(symbol); ok {
				capturedSymbols = append(capturedSymbols, symbol)
				argExprs = append(argExprs, NewSlice([]Scmer{NewSymbol("quote"), NewSymbol(string(symbol))}))
				argExprs = append(argExprs, NewSymbol(string(symbol)))
				continue
			}
		}
		if _, ok := Globalenv.Vars[symbol]; ok {
			continue
		}
		if runtimeEnv, ok := ctx.RuntimeEnv.Any().(*Env); ok && runtimeEnv != nil {
			if binding := runtimeEnv.FindRead(symbol); binding != nil && binding != &Globalenv {
				if _, exists := binding.Vars[symbol]; exists {
					capturedSymbols = append(capturedSymbols, symbol)
					argExprs = append(argExprs, NewSlice([]Scmer{NewSymbol("quote"), NewSymbol(string(symbol))}))
					argExprs = append(argExprs, NewSymbol(string(symbol)))
				}
			}
		}
	}
	consumesRuntimeEnv := jitExpressionConsumesRuntimeEnv(body)
	outerCaptures, namedOuterCaptures := jitLambdaOuterCaptures(body, !consumesRuntimeEnv)
	if consumesRuntimeEnv {
		seen := make(map[jitLambdaOuterCapture]struct{}, len(outerCaptures))
		for _, capture := range outerCaptures {
			seen[capture] = struct{}{}
		}
		for index := 0; index < ctx.LocalSlotCount; index++ {
			capture := jitLambdaOuterCapture{index: NthLocalVar(index)}
			if _, exists := seen[capture]; !exists {
				outerCaptures = append(outerCaptures, capture)
			}
		}
	}
	for _, capture := range outerCaptures {
		key := NewNthLocalVar(capture.index)
		argExprs = append(argExprs, NewSlice([]Scmer{NewSymbol("quote"), key}))
		argExprs = append(argExprs, jitLambdaCaptureReference(capture.index, capture.depth))
	}
	for _, capture := range namedOuterCaptures {
		argExprs = append(argExprs, NewSlice([]Scmer{NewSymbol("quote"), NewSymbol(string(capture.symbol))}))
		argExprs = append(argExprs, jitLambdaNamedCaptureReference(capture.symbol, capture.depth))
	}
	if ctx.RecursiveLambdas && len(capturedSymbols)+len(outerCaptures)+len(namedOuterCaptures) != 0 &&
		len(argExprs) == 3+2*(len(capturedSymbols)+len(outerCaptures)+len(namedOuterCaptures)) && !jitExpressionConsumesRuntimeEnv(body) {
		plainParams := params.WithoutSourceInfo()
		if plainParams.IsSlice() {
			publicParams := plainParams.Slice()
			captureBase := numVars
			symbolBindings := make(map[Symbol]NthLocalVar, len(capturedSymbols))
			for _, symbol := range capturedSymbols {
				symbolBindings[symbol] = NthLocalVar(captureBase + len(symbolBindings))
			}
			outerBindings := make(map[jitLambdaOuterCapture]NthLocalVar, len(outerCaptures))
			for _, capture := range outerCaptures {
				outerBindings[capture] = NthLocalVar(captureBase + len(symbolBindings) + len(outerBindings))
			}
			namedOuterBindings := make(map[jitLambdaNamedOuterCapture]NthLocalVar, len(namedOuterCaptures))
			for _, capture := range namedOuterCaptures {
				namedOuterBindings[capture] = NthLocalVar(captureBase + len(symbolBindings) + len(outerBindings) + len(namedOuterBindings))
			}
			captureCount := len(symbolBindings) + len(outerBindings) + len(namedOuterBindings)
			captureSymbols := append([]Symbol(nil), capturedSymbols...)
			for _, capture := range outerCaptures {
				symbol, _ := jitOuterCaptureSymbol(ctx, capture)
				captureSymbols = append(captureSymbols, symbol)
			}
			for _, capture := range namedOuterCaptures {
				captureSymbols = append(captureSymbols, capture.symbol)
			}
			boundBody := jitBindLambdaCaptures(body, symbolBindings, outerBindings, namedOuterBindings)
			argExprs[1] = NewSlice([]Scmer{NewSymbol("quote"), boundBody})
			if ctx.DefiningSymbol == "" {
				template := jitBuildLambdaClosure(NewSlice(publicParams), boundBody, NewInt(int64(captureBase+captureCount)))
				template.Proc().Compiled = &JITEntryPoint{
					CaptureBase:    captureBase,
					CaptureCount:   captureCount,
					CaptureKeys:    jitLambdaCaptureKeys(argExprs[3:]),
					CaptureSymbols: captureSymbols,
				}
				template = jitCompileModeDeferred(true, template)
				ctx.TrackImm(template)
				return jitEmitBoundLambdaProc(ctx, template, argExprs[3:], ctx.SliceBase, result, result.StackFunc, false)
			}
			selfParam := NthLocalVar(captureBase + captureCount)
			captureCount++
			boundBody = jitBindLambdaSelfValues(boundBody, ctx.DefiningSymbol, selfParam)
			template := jitBuildNamedLambdaClosure(
				NewSymbol(string(ctx.DefiningSymbol)), NewSlice(publicParams),
				boundBody, NewInt(int64(captureBase+captureCount)),
			)
			captures := append([]Scmer(nil), argExprs[3:]...)
			captures = append(captures,
				NewSlice([]Scmer{NewSymbol("quote"), NewSymbol("\x00jit-bound-self")}), NewNil())
			template.Proc().Compiled = &JITEntryPoint{
				CaptureBase:    captureBase,
				CaptureCount:   captureCount,
				CaptureKeys:    jitLambdaCaptureKeys(captures),
				CaptureSymbols: append(captureSymbols, ctx.DefiningSymbol),
			}
			template = jitCompileModeDeferred(true, template)
			ctx.TrackImm(template)
			return jitEmitBoundLambdaProc(ctx, template, captures, ctx.SliceBase, result, result.StackFunc, true)
		}
	}
	if ctx.RecursiveLambdas && ctx.DefiningSymbol != "" && len(argExprs) == 3 {
		closure := jitBuildNamedLambdaClosure(
			NewSymbol(string(ctx.DefiningSymbol)), params, body, NewInt(int64(numVars)),
		)
		compiled := jitCompileModeDeferred(true, closure)
		ctx.TrackImm(compiled)
		return jitPlaceScmerIntoTarget(ctx, JITValueDesc{Loc: LocImm, Type: tagProc, Imm: compiled}, result)
	}
	if ctx.RecursiveLambdas && ctx.DefiningSymbol == "" && len(argExprs) == 3 && !jitExpressionConsumesRuntimeEnv(body) {
		closure := jitBuildLambdaClosure(params, body, NewInt(int64(numVars)))
		compiled := jitCompileModeDeferred(true, closure)
		ctx.TrackImm(compiled)
		if result.StackFunc {
			return jitEmitBoundLambdaProc(ctx, compiled, nil, ctx.SliceBase, result, true, false)
		}
		return jitPlaceScmerIntoTarget(ctx, JITValueDesc{Loc: LocImm, Type: tagProc, Imm: compiled}, result)
	}
	builder := jitBuildLambdaClosure
	if ctx.RecursiveLambdas {
		if ctx.DefiningSymbol != "" {
			argExprs = append([]Scmer{NewSlice([]Scmer{NewSymbol("quote"), NewSymbol(string(ctx.DefiningSymbol))})}, argExprs...)
			if !jitExpressionConsumesRuntimeEnv(body) {
				builder = jitBuildNamedCompiledLambdaClosure
			} else {
				builder = jitBuildNamedLambdaClosure
			}
		} else {
			if !jitExpressionConsumesRuntimeEnv(body) {
				builder = jitBuildCompiledLambdaClosureWithRuntimeEnv
				argExprs = argExprs[:3]
				argExprs = append(argExprs, jitRuntimeCaptureArgExprs(ctx)...)
			}
		}
	}
	return jitEmitGoVariadicCallFromExprs(ctx, builder, argExprs, ctx.SliceBase, result, false)
}

func jitLambdaCaptureKeys(captureArgs []Scmer) []Scmer {
	if len(captureArgs)%2 != 0 {
		panic("jit: invalid lambda captures")
	}
	keys := make([]Scmer, len(captureArgs)/2)
	for index := range keys {
		keyExpr := captureArgs[index*2]
		if !keyExpr.IsSlice() || len(keyExpr.Slice()) != 2 || !keyExpr.Slice()[0].SymbolEquals("quote") {
			panic("jit: lambda capture key is not quoted")
		}
		keys[index] = keyExpr.Slice()[1]
	}
	return keys
}

// jitEmitBoundLambdaProc emits the complete Proc binder. Captures are evaluated
// into rooted frame slots before an escaping allocation, so the single
// mallocgc call is followed only by non-safepoint header/context stores.
func jitEmitBoundLambdaProc(ctx *JITContext, template Scmer, captureArgs []Scmer, sliceBase Reg, result JITValueDesc, stack, bindSelf bool) JITValueDesc {
	proc := template.Proc()
	if proc == nil || proc.Compiled == nil || proc.JITCode == 0 {
		panic("jit: lambda template is not compiled")
	}
	if len(captureArgs)%2 != 0 || len(captureArgs)/2 != proc.Compiled.CaptureCount {
		panic("jit: invalid lambda captures")
	}
	captureCount := len(captureArgs) / 2

	capturesOff := int32(0)
	if captureCount != 0 {
		capturesOff = ctx.AllocStack(int32(captureCount * 16))
		for index := 0; index < captureCount; index++ {
			if bindSelf && index == captureCount-1 {
				continue
			}
			jitCompileRootedCallValueAt(ctx, captureArgs[index*2+1], sliceBase, capturesOff+int32(index*16))
		}
	}

	contextOffset := int32(unsafe.Offsetof(ProcJIT{}.Context))
	objectBytes := contextOffset + int32(captureCount*16)
	ctx.TrackImm(template)
	var object JITValueDesc
	objectOff := int32(0)
	if stack {
		objectOff = ctx.AllocStack(objectBytes)
		object = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: ctx.AllocReg(), Rooted: true, RelocatablePointer: true}
		ctx.EmitLeaRegMem(object.Reg, ctx.StackReg, objectOff)
		ctx.BindReg(object.Reg, &object)
	} else {
		typ := jitProcContextAllocation(captureCount)
		ctx.TrackPointer(typ)
		object = ctx.EmitGoCallScalar(GoFuncAddr(jitRuntimeAllocTyped), []JITValueDesc{
			{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(typ))), NoHeapPointer: true},
		}, 1)
		object.Type = tagInt
		object.Rooted = true
		object.RelocatablePointer = true
	}
	ctx.ProtectReg(object.Reg)
	source := ctx.AllocReg()
	ctx.ProtectReg(source)
	ctx.EmitMovRegImm64(source, uint64(uintptr(unsafe.Pointer(proc))))
	for offset := int32(0); offset < int32(unsafe.Sizeof(Proc{})); offset += 8 {
		ctx.EmitMovRegMem(ctx.ScratchReg, source, offset)
		ctx.EmitStoreRegMem(ctx.ScratchReg, object.Reg, offset)
	}
	ctx.UnprotectReg(source)
	ctx.FreeReg(source)
	if stack {
		for _, offset := range jitProcStackPointerOffsets {
			ctx.setStackPointer(jitStackRootFrameSP, objectOff+offset, true)
		}
	}
	for index := 0; index < captureCount; index++ {
		contextAt := contextOffset + int32(index*16)
		if bindSelf && index == captureCount-1 {
			ctx.EmitStoreRegMem(object.Reg, object.Reg, contextAt)
			ctx.EmitMovRegImm64(ctx.ScratchReg, makeAux(tagProc, 0))
			ctx.EmitStoreRegMem(ctx.ScratchReg, object.Reg, contextAt+8)
		} else {
			ctx.EmitMovRegMem(ctx.ScratchReg, ctx.StackReg, capturesOff+int32(index*16))
			ctx.EmitStoreRegMem(ctx.ScratchReg, object.Reg, contextAt)
			ctx.EmitMovRegMem(ctx.ScratchReg, ctx.StackReg, capturesOff+int32(index*16)+8)
			ctx.EmitStoreRegMem(ctx.ScratchReg, object.Reg, contextAt+8)
		}
		if stack {
			ctx.setStackPointer(jitStackRootFrameSP, objectOff+contextAt, true)
		}
	}
	target := jitEnsureResultPair(ctx, result)
	ctx.EmitMovRegReg(target.Reg, object.Reg)
	ctx.EmitMovRegImm64(target.Reg2, makeAux(tagProc, 0))
	target.Type = tagProc
	target.Rooted = true
	ctx.UnprotectReg(object.Reg)
	ctx.FreeDesc(&object)
	ctx.BindReg(target.Reg, &target)
	ctx.BindReg(target.Reg2, &target)
	return target
}
