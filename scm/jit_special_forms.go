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
	return jitEmitGoVariadicCallFromExprs(ctx, jitEvalSpecial, callArgs, ctx.SliceBase, result)
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
	value := jitCompileExpr(ctx, args[1], ctx.SliceBase, result)
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
	if idx < 0 || idx >= ctx.LocalSlotCount || !ctx.SliceBaseTracksRSP {
		panic("jit: setN target outside invocation frame")
	}
	value := jitCompileExpr(ctx, args[1], ctx.SliceBase, result)
	ctx.EnsureDesc(&value)
	if value.Loc != LocRegPair && value.Loc != LocImm && value.Loc != LocStackPair && value.Loc != LocInputPair {
		pair := jitAllocTrackedPair(ctx, value.Type)
		value = jitPlaceIntoPair(ctx, &value, pair)
	}
	ctx.EmitStoreScmerToStack(value, int32(idx*16))
	return value
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
	callArgs := []Scmer{
		NewSlice([]Scmer{NewSymbol("quote"), args[0]}),
		NewSlice([]Scmer{NewSymbol("quote"), generator}),
		NewSlice([]Scmer{NewSymbol("quote"), whitespace}),
		NewBool(ignoreResult),
	}
	callArgs = append(callArgs, jitRuntimeCaptureArgExprs(ctx)...)
	return jitEmitGoVariadicCallFromExprs(ctx, jitParserSpecial, callArgs, ctx.SliceBase, result)
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
			ctx.Env = &JITEnv{Vars: make(map[Symbol]JITValueDesc), Outer: outerEnv}
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

func jitEmitSpecialReservedList(ctx *JITContext, args []Scmer, _ []JITValueDesc, result JITValueDesc) JITValueDesc {
	var capacityExpr Scmer
	switch {
	case len(args) == 2 && args[0].IsNthLocalVar():
		start := int(args[0].NthLocalVar())
		capacity := int(ToInt(args[1]))
		if capacity < 0 {
			capacity = 0
		}
		if start < 0 || start+capacity > ctx.LocalSlotCount {
			panic("jit: !!list slots outside invocation frame")
		}
		capacityExpr = NewInt(int64(capacity))
	case len(args) == 1:
		capacityExpr = args[0]
	default:
		panic("jit: malformed !!list")
	}
	capacity := jitCompileExpr(ctx, capacityExpr, ctx.SliceBase, JITValueDesc{Loc: LocAny})
	ctx.AutoImportSafe = false
	target := jitEnsureResultPair(ctx, result)
	out := ctx.EmitGoCallScalarInto(GoFuncAddr(jitMakeReservedList), []JITValueDesc{capacity}, target)
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
	target := jitEnsureResultPair(ctx, result)
	var endLabel uint8
	hasDynamic := false
	i := 0
	for i+1 < len(args) {
		condition := jitCompileExpr(ctx, args[i], ctx.SliceBase, JITValueDesc{Loc: LocAny})
		boolean := jitCondToBool(ctx, &condition)
		if boolean.Loc == LocImm {
			if boolean.Imm.Bool() {
				value := jitCompileExpr(ctx, args[i+1], ctx.SliceBase, target)
				_ = jitPlaceIntoPair(ctx, &value, target)
				if hasDynamic {
					ctx.MarkLabel(endLabel)
				}
				ctx.BindReg(target.Reg, &target)
				ctx.BindReg(target.Reg2, &target)
				return target
			}
			i += 2
			continue
		}
		if !hasDynamic {
			endLabel = ctx.ReserveLabel()
			hasDynamic = true
		}
		nextLabel := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(boolean.Reg, 0)
		ctx.EmitJcc(CcE, nextLabel)
		ctx.FreeDesc(&boolean)
		value := jitCompileExpr(ctx, args[i+1], ctx.SliceBase, target)
		_ = jitPlaceIntoPair(ctx, &value, target)
		ctx.EmitJmp(endLabel)
		ctx.MarkLabel(nextLabel)
		i += 2
	}
	if i < len(args) {
		value := jitCompileExpr(ctx, args[i], ctx.SliceBase, target)
		_ = jitPlaceIntoPair(ctx, &value, target)
	} else {
		nilValue := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
		_ = jitPlaceIntoPair(ctx, &nilValue, target)
	}
	if hasDynamic {
		ctx.MarkLabel(endLabel)
	}
	ctx.BindReg(target.Reg, &target)
	ctx.BindReg(target.Reg2, &target)
	return target
}

func jitEmitSpecialIfCond(ctx *JITContext, args []Scmer, trueLabel, falseLabel uint8) {
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
	return func(ctx *JITContext, args []Scmer, trueLabel, falseLabel uint8) {
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
		target := jitEnsureResultPair(ctx, result)
		var takeLabel, endLabel uint8
		hasDynamic := false
		compileTimeTake := false
		for _, expression := range args {
			condition := jitCompileExpr(ctx, expression, ctx.SliceBase, JITValueDesc{Loc: LocAny})
			boolean := jitCondToBool(ctx, &condition)
			if boolean.Loc == LocImm {
				if boolean.Imm.Bool() == takeWhen {
					compileTimeTake = true
					break
				}
				continue
			}
			if !hasDynamic {
				takeLabel = ctx.ReserveLabel()
				endLabel = ctx.ReserveLabel()
				hasDynamic = true
			}
			ctx.EmitCmpRegImm32(boolean.Reg, 0)
			if takeWhen {
				ctx.EmitJcc(CcNE, takeLabel)
			} else {
				ctx.EmitJcc(CcE, takeLabel)
			}
			ctx.FreeDesc(&boolean)
		}
		if compileTimeTake {
			if hasDynamic {
				ctx.MarkLabel(takeLabel)
			}
			taken := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(takeWhen)}
			_ = jitPlaceIntoPair(ctx, &taken, target)
			ctx.BindReg(target.Reg, &target)
			ctx.BindReg(target.Reg2, &target)
			return target
		}
		identityValue := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(identity)}
		_ = jitPlaceIntoPair(ctx, &identityValue, target)
		if hasDynamic {
			ctx.EmitJmp(endLabel)
			ctx.MarkLabel(takeLabel)
			taken := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(takeWhen)}
			_ = jitPlaceIntoPair(ctx, &taken, target)
			ctx.MarkLabel(endLabel)
		}
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
		target := jitEnsureResultPair(ctx, result)
		endLabel := ctx.ReserveLabel()
		if nilOnly {
			nilValue := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
			_ = jitPlaceIntoPair(ctx, &nilValue, target)
		}
		for i, expression := range args {
			value := jitCompileExpr(ctx, expression, ctx.SliceBase, JITValueDesc{Loc: LocAny})
			if !nilOnly && i == len(args)-1 {
				_ = jitPlaceIntoPair(ctx, &value, target)
				break
			}
			if value.Loc == LocImm {
				take := !value.Imm.IsNil()
				if !nilOnly {
					take = value.Imm.Bool()
				}
				if take {
					_ = jitPlaceIntoPair(ctx, &value, target)
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
					_ = jitPlaceIntoPair(ctx, &value, target)
					ctx.EmitJmp(endLabel)
				}
				ctx.FreeDesc(&value)
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
			_ = jitPlaceIntoPair(ctx, &value, target)
			ctx.EmitJmp(endLabel)
			ctx.MarkLabel(nextLabel)
			ctx.FreeDesc(&predicate)
			ctx.FreeDesc(&value)
		}
		ctx.MarkLabel(endLabel)
		ctx.BindReg(target.Reg, &target)
		ctx.BindReg(target.Reg2, &target)
		return target
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
	argExprs := make([]Scmer, 0, 16)
	argExprs = append(argExprs, NewSlice([]Scmer{NewSymbol("quote"), params}))
	argExprs = append(argExprs, NewSlice([]Scmer{NewSymbol("quote"), body}))
	argExprs = append(argExprs, NewInt(int64(numVars)))
	for _, symbol := range jitLambdaFreeSymbols(params, body) {
		if ctx.Env != nil {
			if _, ok := ctx.Env.Lookup(symbol); ok {
				argExprs = append(argExprs, NewSlice([]Scmer{NewSymbol("quote"), NewSymbol(string(symbol))}))
				argExprs = append(argExprs, NewSymbol(string(symbol)))
				continue
			}
		}
		if _, ok := Globalenv.Vars[symbol]; ok {
			continue
		}
	}
	for _, index := range jitLambdaOuterVarIndices(body) {
		key := NewNthLocalVar(index)
		argExprs = append(argExprs, NewSlice([]Scmer{NewSymbol("quote"), key}))
		argExprs = append(argExprs, key)
	}
	builder := jitBuildLambdaClosure
	if ctx.RecursiveLambdas {
		builder = jitBuildCompiledLambdaClosure
	}
	return jitEmitGoVariadicCallFromExprs(ctx, builder, argExprs, ctx.SliceBase, result)
}
