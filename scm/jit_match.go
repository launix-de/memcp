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
	"regexp"
	"unsafe"
)

// Match lowering is architecture-independent. It describes values, branches,
// loads, and calls through the common JIT emitter API; only that API's machine
// instruction implementation belongs in an architecture-specific file.
var jitMatchEmptyListStorage [1]Scmer

func jitMatchAsGoSlice(value Scmer) []Scmer {
	list, ok := scmerAsSlice(value)
	if !ok {
		return nil
	}
	// A non-nil pointer distinguishes a successfully normalized empty list from
	// a non-list without adding a fourth ABI result word to the emitted call.
	if len(list) == 0 && unsafe.SliceData(list) == nil {
		return jitMatchEmptyListStorage[:0]
	}
	return list
}

func jitMatchSymbolLiteralWords(valuePtr *byte, valueAux uint64, expectedPtr *byte, expectedAux uint64) bool {
	value := Scmer{ptr: valuePtr, aux: valueAux}
	expected := Scmer{ptr: expectedPtr, aux: expectedAux}
	actualName, actualOK := scmerSymbolName(value)
	expectedName, expectedOK := scmerSymbolName(expected)
	return actualOK && expectedOK && actualName == expectedName
}

func jitMatchEqualWords(valuePtr *byte, valueAux uint64, expectedPtr *byte, expectedAux uint64) bool {
	return Equal(Scmer{ptr: valuePtr, aux: valueAux}, Scmer{ptr: expectedPtr, aux: expectedAux})
}

func jitMatchIsString(value Scmer) bool {
	_, ok := scmerAsString(value)
	return ok
}

type jitMatchOutcome struct {
	possible bool
	always   bool
}

func jitMatchBranchEnv(outer *JITEnv) *JITEnv {
	env := &JITEnv{Outer: outer}
	if outer != nil && len(outer.Numbered) > 0 {
		env.Numbered = append([]JITValueDesc(nil), outer.Numbered...)
	}
	return env
}

func jitMatchStableValue(ctx *JITContext, value JITValueDesc) JITValueDesc {
	if value.Loc == LocImm || value.Loc == LocStackPair || value.Loc == LocInputPair || value.Loc == LocVirtualSlice {
		return value
	}
	src := value
	ctx.EnsureDesc(&src)
	if src.Loc == LocReg {
		ptrReg := ctx.AllocRegExcept(src.Reg)
		target := JITValueDesc{Loc: LocRegPair, Type: src.Type, Reg: ptrReg, Reg2: ctx.AllocRegExcept(src.Reg, ptrReg)}
		src = jitPlaceIntoPair(ctx, &src, target)
	}
	if src.Loc != LocRegPair {
		panic("jit: match value is not a Scmer pair")
	}
	off := ctx.AllocStack(16)
	ctx.EmitStoreScmerToStack(src, off)
	ctx.FreeDesc(&src)
	return JITValueDesc{
		Loc: LocStackPair, Type: value.Type, StackOff: off,
		KnownSliceLen: value.KnownSliceLen, KnownSliceCap: value.KnownSliceCap,
		SliceSizeKnown: value.SliceSizeKnown, NoHeapPointer: value.NoHeapPointer,
	}
}

func jitMatchBindValue(ctx *JITContext, env *JITEnv, pattern Scmer, value JITValueDesc) {
	if pattern.IsSymbol() && pattern.SymbolEquals("_") {
		return
	}
	bound := value
	if value.Loc != LocImm {
		bound = jitMatchStableValue(ctx, value)
	}
	if pattern.IsNthLocalVar() {
		idx := int(pattern.NthLocalVar())
		if idx >= len(env.Numbered) {
			numbered := make([]JITValueDesc, idx+1)
			copy(numbered, env.Numbered)
			env.Numbered = numbered
		}
		env.Numbered[idx] = bound
		return
	}
	if !pattern.IsSymbol() {
		panic("jit: invalid match binding")
	}
	if env.Vars == nil {
		env.Vars = make(map[Symbol]JITValueDesc)
	}
	env.Vars[pattern.Symbol()] = bound
}

func jitMatchMaterializeImm(ctx *JITContext, value Scmer) JITValueDesc {
	ptrReg := ctx.AllocReg()
	auxReg := ctx.AllocRegExcept(ptrReg)
	target := JITValueDesc{Loc: LocRegPair, Type: value.GetTag(), Reg: ptrReg, Reg2: auxReg}
	source := JITValueDesc{Loc: LocImm, Type: value.GetTag(), Imm: value}
	return jitPlaceIntoPair(ctx, &source, target)
}

func jitEmitMatchBool(ctx *JITContext, condition JITValueDesc, failLabel JITLabel) jitMatchOutcome {
	if condition.Loc == LocImm {
		return jitMatchOutcome{possible: condition.Imm.Bool(), always: condition.Imm.Bool()}
	}
	ctx.EnsureDesc(&condition)
	if condition.Loc != LocReg {
		panic("jit: match condition is not scalar")
	}
	// Go's internal ABI only defines the low byte of a bool result. Clear stale
	// upper bits before using it as a branch condition.
	ctx.EmitAndRegImm32(condition.Reg, 1)
	ctx.EmitCmpRegImm32(condition.Reg, 0)
	ctx.EmitJump(CondEqual, failLabel)
	ctx.FreeDesc(&condition)
	return jitMatchOutcome{possible: true, always: false}
}

func jitEmitMatchTag(ctx *JITContext, value *JITValueDesc, tag uint8, failLabel JITLabel) jitMatchOutcome {
	condition := ctx.EmitTagEqualsBorrowed(value, tag, JITValueDesc{Loc: LocAny})
	return jitEmitMatchBool(ctx, condition, failLabel)
}

func jitMatchLoadElement(ctx *JITContext, slice *JITValueDesc, index int) JITValueDesc {
	ctx.EnsureDesc(slice)
	if slice.Loc != LocRegTriple {
		panic("jit: match list requires a Go slice header")
	}
	ctx.ProtectReg(slice.Reg)
	ctx.ProtectReg(slice.Reg2)
	ctx.ProtectReg(slice.Reg3)
	ptrReg := ctx.AllocRegExcept(slice.Reg, slice.Reg2, slice.Reg3)
	auxReg := ctx.AllocRegExcept(slice.Reg, slice.Reg2, slice.Reg3, ptrReg)
	ctx.UnprotectReg(slice.Reg3)
	ctx.UnprotectReg(slice.Reg2)
	ctx.UnprotectReg(slice.Reg)
	ctx.EmitMovRegMem(ptrReg, slice.Reg, int32(index*16))
	ctx.EmitMovRegMem(auxReg, slice.Reg, int32(index*16+8))
	result := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ptrReg, Reg2: auxReg}
	ctx.BindReg(ptrReg, &result)
	ctx.BindReg(auxReg, &result)
	return result
}

func jitMatchSliceTail(ctx *JITContext, slice *JITValueDesc) JITValueDesc {
	ctx.EnsureDesc(slice)
	if slice.Loc != LocRegTriple {
		panic("jit: match cons requires a Go slice header")
	}
	ctx.ProtectReg(slice.Reg)
	ctx.ProtectReg(slice.Reg2)
	ctx.ProtectReg(slice.Reg3)
	ptrReg := ctx.AllocRegExcept(slice.Reg, slice.Reg2, slice.Reg3)
	auxReg := ctx.AllocRegExcept(slice.Reg, slice.Reg2, slice.Reg3, ptrReg)
	ctx.UnprotectReg(slice.Reg3)
	ctx.UnprotectReg(slice.Reg2)
	ctx.UnprotectReg(slice.Reg)
	ctx.EmitMovRegReg(ptrReg, slice.Reg)
	ctx.EmitAddRegImm32(ptrReg, 16)
	// NewSlice canonicalizes a zero-capacity slice to a nil data pointer. Keep
	// the same invariant for the tail of a one-element match: a one-past-the-end
	// pointer is not a valid GC root even though the resulting length is zero.
	keepPtr := ctx.ReserveLabel()
	ctx.EmitCmpRegImm32(slice.Reg3, 1)
	ctx.EmitJump(CondNotEqual, keepPtr)
	ctx.EmitMovRegImm64(ptrReg, 0)
	ctx.MarkLabel(keepPtr)
	ctx.EmitMovRegReg(auxReg, slice.Reg2)
	ctx.EmitSubRegImm32(auxReg, 1)
	ctx.EmitShlRegImm8(auxReg, sliceCapBits)
	ctx.EmitMovRegReg(ctx.ScratchReg, slice.Reg3)
	ctx.EmitSubRegImm32(ctx.ScratchReg, 1)
	ctx.EmitOrInt64(auxReg, ctx.ScratchReg)
	ctx.EmitShlRegImm8(auxReg, 8)
	ctx.EmitOrRegImm32(auxReg, int32(tagSlice))
	result := JITValueDesc{Loc: LocRegPair, Type: tagSlice, Reg: ptrReg, Reg2: auxReg}
	if slice.SliceSizeKnown {
		result.KnownSliceLen = slice.KnownSliceLen - 1
		result.KnownSliceCap = slice.KnownSliceCap - 1
		result.SliceSizeKnown = true
	}
	ctx.BindReg(ptrReg, &result)
	ctx.BindReg(auxReg, &result)
	return result
}

func jitMatchListValue(ctx *JITContext, value JITValueDesc, failLabel JITLabel) (JITValueDesc, JITValueDesc, jitMatchOutcome) {
	if value.Loc == LocImm {
		list, ok := scmerAsSlice(value.Imm)
		if !ok {
			return JITValueDesc{}, JITValueDesc{}, jitMatchOutcome{}
		}
		normalized := JITValueDesc{
			Loc: LocImm, Type: tagSlice, Imm: NewSlice(list),
			KnownSliceLen: int32(len(list)), KnownSliceCap: int32(cap(list)), SliceSizeKnown: true,
		}
		return normalized, JITValueDesc{}, jitMatchOutcome{possible: true, always: true}
	}
	if value.Type != JITTypeUnknown && value.Type != tagSlice && value.Type != tagFastDict && value.Type != tagSourceInfo {
		return JITValueDesc{}, JITValueDesc{}, jitMatchOutcome{}
	}

	normalized := value
	typeKnown := value.Type == tagSlice
	if !typeKnown {
		header := ctx.EmitGoCallScalar(GoFuncAddr(jitMatchAsGoSlice), []JITValueDesc{value}, 3)
		ctx.BindReg(header.Reg, &header)
		ctx.BindReg(header.Reg2, &header)
		ctx.BindReg(header.Reg3, &header)
		ctx.EmitCmpRegImm32(header.Reg, 0)
		ctx.EmitJump(CondEqual, failLabel)
		normalized = ctx.EmitNewSliceFromGoSlice(&header)
	}
	normalized.KnownSliceLen = value.KnownSliceLen
	normalized.KnownSliceCap = value.KnownSliceCap
	normalized.SliceSizeKnown = value.SliceSizeKnown
	normalized = jitMatchStableValue(ctx, normalized)
	sliceValue := normalized
	header := jitKnownSliceHeader(ctx, &sliceValue)
	ctx.FreeDesc(&sliceValue)
	header.KnownSliceLen = value.KnownSliceLen
	header.KnownSliceCap = value.KnownSliceCap
	header.SliceSizeKnown = value.SliceSizeKnown
	return normalized, header, jitMatchOutcome{possible: true, always: typeKnown}
}

func jitMatchLiteral(ctx *JITContext, value JITValueDesc, pattern Scmer, failLabel JITLabel) jitMatchOutcome {
	if value.Loc == LocImm {
		matched := Equal(value.Imm, pattern)
		return jitMatchOutcome{possible: matched, always: matched}
	}
	expected := jitMatchMaterializeImm(ctx, pattern)
	condition := ctx.EmitGoCallScalar(GoFuncAddr(jitMatchEqualWords), []JITValueDesc{value, expected}, 1)
	ctx.FreeDesc(&expected)
	return jitEmitMatchBool(ctx, condition, failLabel)
}

func jitMatchDynamicValue(ctx *JITContext, value, expected JITValueDesc, failLabel JITLabel) jitMatchOutcome {
	if value.Loc == LocImm && expected.Loc == LocImm {
		matched := Equal(value.Imm, expected.Imm)
		return jitMatchOutcome{possible: matched, always: matched}
	}
	materialize := func(desc JITValueDesc) JITValueDesc {
		if desc.Loc == LocImm {
			return jitMatchMaterializeImm(ctx, desc.Imm)
		}
		if desc.Loc == LocReg {
			ptrReg := ctx.AllocRegExcept(desc.Reg)
			target := JITValueDesc{Loc: LocRegPair, Type: desc.Type, Reg: ptrReg, Reg2: ctx.AllocRegExcept(desc.Reg, ptrReg)}
			return jitPlaceIntoPair(ctx, &desc, target)
		}
		return desc
	}
	actualArg := materialize(value)
	expectedArg := materialize(expected)
	condition := ctx.EmitGoCallScalar(GoFuncAddr(jitMatchEqualWords), []JITValueDesc{actualArg, expectedArg}, 1)
	if actualArg.Loc == LocRegPair && value.Loc != LocRegPair {
		ctx.FreeDesc(&actualArg)
	}
	if expectedArg.Loc == LocRegPair && expected.Loc != LocRegPair {
		ctx.FreeDesc(&expectedArg)
	}
	return jitEmitMatchBool(ctx, condition, failLabel)
}

func jitMatchBoolLiteral(ctx *JITContext, value JITValueDesc, expected bool, failLabel JITLabel) jitMatchOutcome {
	if value.Loc == LocImm {
		matched := value.Imm.IsBool() && value.Imm.Bool() == expected
		return jitMatchOutcome{possible: matched, always: matched}
	}
	if value.Type != JITTypeUnknown && value.Type != tagBool {
		return jitMatchOutcome{}
	}
	typeAlways := value.Type == tagBool
	if !typeAlways {
		jitEmitMatchTag(ctx, &value, tagBool, failLabel)
	}
	pair := value
	ctx.EnsureDesc(&pair)
	if pair.Loc != LocRegPair {
		panic("jit: bool match requires a Scmer pair")
	}
	expectedAux := makeAux(tagBool, 0)
	if expected {
		expectedAux = makeAux(tagBool, 1)
	}
	ctx.EmitCmpRegImm32(pair.Reg2, int32(expectedAux))
	ctx.EmitJump(CondNotEqual, failLabel)
	ctx.FreeDesc(&pair)
	return jitMatchOutcome{possible: true, always: false}
}

func jitMatchSymbol(ctx *JITContext, value JITValueDesc, expected Scmer, failLabel JITLabel) jitMatchOutcome {
	if value.Loc == LocImm {
		matched := func() bool {
			actualName, actualOK := scmerSymbolName(value.Imm)
			expectedName, expectedOK := scmerSymbolName(expected)
			return actualOK && expectedOK && actualName == expectedName
		}()
		return jitMatchOutcome{possible: matched, always: matched}
	}
	if value.Type != JITTypeUnknown && value.Type != tagSymbol && value.Type != tagSourceInfo {
		return jitMatchOutcome{}
	}
	expectedValue := jitMatchMaterializeImm(ctx, expected)
	condition := ctx.EmitGoCallScalar(GoFuncAddr(jitMatchSymbolLiteralWords), []JITValueDesc{value, expectedValue}, 1)
	ctx.FreeDesc(&expectedValue)
	return jitEmitMatchBool(ctx, condition, failLabel)
}

func jitMatchNumber(ctx *JITContext, value JITValueDesc, failLabel JITLabel) jitMatchOutcome {
	if value.Loc == LocImm {
		matched := value.Imm.IsInt() || value.Imm.IsFloat()
		return jitMatchOutcome{possible: matched, always: matched}
	}
	if value.Type != JITTypeUnknown {
		matched := value.Type == tagInt || value.Type == tagFloat
		return jitMatchOutcome{possible: matched, always: matched}
	}
	tmp := value
	ctx.EnsureDesc(&tmp)
	if tmp.Loc != LocRegPair {
		panic("jit: number match requires a Scmer pair")
	}
	tagReg := ctx.AllocRegExcept(tmp.Reg, tmp.Reg2)
	ctx.EmitGetTagRegs(tagReg, tmp.Reg, tmp.Reg2)
	accepted := ctx.ReserveLabel()
	ctx.EmitCmpRegImm8(tagReg, tagInt)
	ctx.EmitJump(CondEqual, accepted)
	ctx.EmitCmpRegImm8(tagReg, tagFloat)
	ctx.EmitJump(CondNotEqual, failLabel)
	ctx.MarkLabel(accepted)
	ctx.FreeReg(tagReg)
	ctx.FreeDesc(&tmp)
	return jitMatchOutcome{possible: true, always: false}
}

func jitMatchString(ctx *JITContext, value JITValueDesc, failLabel JITLabel) jitMatchOutcome {
	if value.Loc == LocImm {
		_, matched := scmerAsString(value.Imm)
		return jitMatchOutcome{possible: matched, always: matched}
	}
	if value.Type == tagString {
		return jitMatchOutcome{possible: true, always: true}
	}
	if value.Type != JITTypeUnknown && value.Type != tagAny && value.Type != tagSourceInfo {
		return jitMatchOutcome{}
	}
	condition := ctx.EmitGoCallScalar(GoFuncAddr(jitMatchIsString), []JITValueDesc{value}, 1)
	return jitEmitMatchBool(ctx, condition, failLabel)
}

func jitMatchFixedList(ctx *JITContext, value JITValueDesc, patterns []Scmer, env *JITEnv, failLabel JITLabel) jitMatchOutcome {
	if value.Loc == LocVirtualSlice {
		if len(value.Virtual) != len(patterns) {
			return jitMatchOutcome{}
		}
		allAlways := true
		for i, pattern := range patterns {
			outcome := jitCompileMatchPattern(ctx, value.Virtual[i], pattern, env, failLabel)
			if !outcome.possible {
				return jitMatchOutcome{}
			}
			allAlways = allAlways && outcome.always
		}
		return jitMatchOutcome{possible: true, always: allAlways}
	}
	normalized, header, listOutcome := jitMatchListValue(ctx, value, failLabel)
	if !listOutcome.possible {
		return jitMatchOutcome{}
	}
	if normalized.Loc == LocImm {
		list := normalized.Imm.Slice()
		if len(list) != len(patterns) {
			return jitMatchOutcome{}
		}
		for i, pattern := range patterns {
			outcome := jitCompileMatchPattern(ctx, JITValueDesc{Loc: LocImm, Type: list[i].GetTag(), Imm: list[i]}, pattern, env, failLabel)
			if !outcome.possible {
				return jitMatchOutcome{}
			}
		}
		return jitMatchOutcome{possible: true, always: true}
	}
	if normalized.Loc == LocRegPair {
		jitParkCallArgument(ctx, &normalized)
	}
	if header.SliceSizeKnown {
		if int(header.KnownSliceLen) != len(patterns) {
			ctx.FreeDesc(&header)
			ctx.FreeDesc(&normalized)
			return jitMatchOutcome{}
		}
	} else {
		ctx.EmitCmpRegImm32(header.Reg2, int32(len(patterns)))
		ctx.EmitJump(CondNotEqual, failLabel)
		listOutcome.always = false
	}
	ctx.FreeDesc(&header)
	allAlways := listOutcome.always
	for i, pattern := range patterns {
		sliceValue := normalized
		header = jitKnownSliceHeader(ctx, &sliceValue)
		ctx.FreeDesc(&sliceValue)
		element := jitMatchLoadElement(ctx, &header, i)
		ctx.FreeDesc(&header)
		outcome := jitCompileMatchPattern(ctx, element, pattern, env, failLabel)
		ctx.FreeDesc(&element)
		if !outcome.possible {
			ctx.FreeDesc(&normalized)
			return jitMatchOutcome{}
		}
		allAlways = allAlways && outcome.always
	}
	ctx.FreeDesc(&normalized)
	return jitMatchOutcome{possible: true, always: allAlways}
}

func jitMatchCons(ctx *JITContext, value JITValueDesc, patterns []Scmer, env *JITEnv, failLabel JITLabel) jitMatchOutcome {
	if len(patterns) != 2 {
		panic("jit: cons match expects head and tail patterns")
	}
	if value.Loc == LocVirtualSlice {
		if len(value.Virtual) == 0 {
			return jitMatchOutcome{}
		}
		tail := JITValueDesc{
			Loc: LocVirtualSlice, Type: tagSlice, Virtual: value.Virtual[1:],
			KnownSliceLen: int32(len(value.Virtual) - 1), KnownSliceCap: int32(len(value.Virtual) - 1), SliceSizeKnown: true,
		}
		first := jitCompileMatchPattern(ctx, value.Virtual[0], patterns[0], env, failLabel)
		second := jitCompileMatchPattern(ctx, tail, patterns[1], env, failLabel)
		return jitMatchOutcome{possible: first.possible && second.possible, always: first.always && second.always}
	}
	normalized, header, listOutcome := jitMatchListValue(ctx, value, failLabel)
	if !listOutcome.possible {
		return jitMatchOutcome{}
	}
	if normalized.Loc == LocImm {
		list := normalized.Imm.Slice()
		if len(list) == 0 {
			return jitMatchOutcome{}
		}
		head := JITValueDesc{Loc: LocImm, Type: list[0].GetTag(), Imm: list[0]}
		tailValue := NewSlice(list[1:])
		tail := JITValueDesc{Loc: LocImm, Type: tagSlice, Imm: tailValue, KnownSliceLen: int32(len(list) - 1), KnownSliceCap: int32(cap(list) - 1), SliceSizeKnown: true}
		first := jitCompileMatchPattern(ctx, head, patterns[0], env, failLabel)
		second := jitCompileMatchPattern(ctx, tail, patterns[1], env, failLabel)
		return jitMatchOutcome{possible: first.possible && second.possible, always: first.always && second.always}
	}
	if header.SliceSizeKnown {
		if header.KnownSliceLen == 0 {
			ctx.FreeDesc(&header)
			ctx.FreeDesc(&normalized)
			return jitMatchOutcome{}
		}
	} else {
		ctx.EmitCmpRegImm32(header.Reg2, 0)
		ctx.EmitJump(CondEqual, failLabel)
		listOutcome.always = false
	}
	head := jitMatchLoadElement(ctx, &header, 0)
	head = ctx.stabilizeForNested(head)
	tail := jitMatchSliceTail(ctx, &header)
	first := jitCompileMatchPattern(ctx, head, patterns[0], env, failLabel)
	second := jitCompileMatchPattern(ctx, tail, patterns[1], env, failLabel)
	ctx.FreeDesc(&head)
	ctx.FreeDesc(&tail)
	ctx.FreeDesc(&header)
	ctx.FreeDesc(&normalized)
	return jitMatchOutcome{
		possible: first.possible && second.possible,
		always:   listOutcome.always && first.always && second.always,
	}
}

func jitCompileMatchPattern(ctx *JITContext, value JITValueDesc, pattern Scmer, env *JITEnv, failLabel JITLabel) jitMatchOutcome {
	for pattern.IsSourceInfo() {
		pattern = pattern.SourceInfo().value
	}
	switch pattern.GetTag() {
	case tagInt, tagFloat, tagString:
		return jitMatchLiteral(ctx, value, pattern, failLabel)
	case tagNthLocalVar:
		if int(pattern.NthLocalVar()) >= len(env.Numbered) {
			return jitMatchLiteral(ctx, value, pattern, failLabel)
		}
		jitMatchBindValue(ctx, env, pattern, value)
		return jitMatchOutcome{possible: true, always: true}
	case tagSymbol:
		switch pattern.String() {
		case "nil":
			if value.Loc == LocImm {
				matched := value.Imm.IsNil()
				return jitMatchOutcome{possible: matched, always: matched}
			}
			if value.Type != JITTypeUnknown {
				matched := value.Type == tagNil
				return jitMatchOutcome{possible: matched, always: matched}
			}
			return jitEmitMatchTag(ctx, &value, tagNil, failLabel)
		case "true":
			return jitMatchBoolLiteral(ctx, value, true, failLabel)
		case "false":
			return jitMatchBoolLiteral(ctx, value, false, failLabel)
		default:
			jitMatchBindValue(ctx, env, pattern, value)
			return jitMatchOutcome{possible: true, always: true}
		}
	case tagSlice:
		p := pattern.Slice()
		if len(p) == 0 {
			panic("jit: empty match pattern")
		}
		if _, direct := symbolName(p[0]); !direct {
			if nested, ok := scmerAsSlice(p[0]); ok && len(nested) > 0 {
				if head, ok := scmerSymbolName(nested[0]); ok && (head == "symbol" || head == "quote") {
					return jitMatchFixedList(ctx, value, p, env, failLabel)
				}
			}
			panic("jit: unsupported nested match pattern")
		}
		name, _ := symbolName(p[0])
		switch name {
		case "eval":
			if len(p) != 2 {
				panic("jit: malformed eval match pattern")
			}
			outerEnv := ctx.Env
			ctx.Env = env
			expected := jitCompileExpr(ctx, p[1], ctx.SliceBase, JITValueDesc{Loc: LocAny})
			ctx.Env = outerEnv
			return jitMatchDynamicValue(ctx, value, expected, failLabel)
		case "var":
			if len(p) != 2 || !p[1].IsInt() {
				panic("jit: malformed var match pattern")
			}
			jitMatchBindValue(ctx, env, NewNthLocalVar(NthLocalVar(p[1].Int())), value)
			return jitMatchOutcome{possible: true, always: true}
		case "list":
			return jitMatchFixedList(ctx, value, p[1:], env, failLabel)
		case "quote", "symbol":
			if len(p) != 2 {
				panic("jit: malformed symbol match pattern")
			}
			return jitMatchSymbol(ctx, value, p[1], failLabel)
		case "string?":
			if len(p) != 2 {
				panic("jit: malformed string match pattern")
			}
			outcome := jitMatchString(ctx, value, failLabel)
			if !outcome.possible {
				return outcome
			}
			inner := jitCompileMatchPattern(ctx, value, p[1], env, failLabel)
			return jitMatchOutcome{possible: inner.possible, always: outcome.always && inner.always}
		case "number?":
			if len(p) != 2 {
				panic("jit: malformed number match pattern")
			}
			outcome := jitMatchNumber(ctx, value, failLabel)
			if !outcome.possible {
				return outcome
			}
			inner := jitCompileMatchPattern(ctx, value, p[1], env, failLabel)
			return jitMatchOutcome{possible: inner.possible, always: outcome.always && inner.always}
		case "list?":
			if len(p) != 2 {
				panic("jit: malformed list match pattern")
			}
			normalized, header, outcome := jitMatchListValue(ctx, value, failLabel)
			if !outcome.possible {
				return outcome
			}
			ctx.FreeDesc(&header)
			inner := jitCompileMatchPattern(ctx, normalized, p[1], env, failLabel)
			ctx.FreeDesc(&normalized)
			return jitMatchOutcome{possible: inner.possible, always: outcome.always && inner.always}
		case "cons":
			return jitMatchCons(ctx, value, p[1:], env, failLabel)
		case "regex":
			if len(p) < 3 {
				panic("jit: malformed regex match pattern")
			}
			pattern := p[1].WithoutSourceInfo()
			if pattern.IsString() {
				compiled, err := regexp.Compile(pattern.String())
				if err != nil {
					panic(err)
				}
				pattern = NewRegex(compiled)
			}
			if !pattern.IsRegex() {
				panic("regex expects string or precompiled regexp")
			}
			if pattern.Regex().NumSubexp() != len(p)-3 {
				panic(fmt.Sprintf("regex %s contains %d subexpressions, found %d", pattern.Regex(), pattern.Regex().NumSubexp(), len(p)-3))
			}
			captures := jitEmitConstantRegexpCaptures(ctx, pattern.Regex(), value, failLabel)
			for index, capture := range captures {
				jitMatchBindValue(ctx, env, p[index+2], capture)
			}
			return jitMatchOutcome{possible: true, always: false}
		default:
			panic("jit: unsupported match pattern " + name)
		}
	default:
		panic("jit: unsupported match pattern type")
	}
}

func jitCompileMatch(ctx *JITContext, list []Scmer, sliceBase Reg, result JITValueDesc) JITValueDesc {
	if len(list) < 2 {
		panic("jit: match expects a value")
	}
	valueExpr := list[1]
	for valueExpr.IsSourceInfo() {
		valueExpr = valueExpr.SourceInfo().value
	}
	var value JITValueDesc
	if items, ok := scmerAsSlice(valueExpr); ok && len(items) > 0 && items[0].SymbolEquals("list") {
		values := make([]JITValueDesc, len(items)-1)
		for i := 1; i < len(items); i++ {
			item := jitCompileExpr(ctx, items[i], sliceBase, JITValueDesc{Loc: LocAny})
			values[i-1] = jitMatchStableValue(ctx, item)
		}
		value = JITValueDesc{
			Loc: LocVirtualSlice, Type: tagSlice, Virtual: values,
			KnownSliceLen: int32(len(values)), KnownSliceCap: int32(len(values)), SliceSizeKnown: true,
		}
	} else {
		value = jitCompileExpr(ctx, valueExpr, sliceBase, JITValueDesc{Loc: LocAny})
	}
	value = jitMatchStableValue(ctx, value)
	var target JITValueDesc
	if result.Loc == LocStackPair {
		target = result
	} else if ctx.StackPhiTargets && result.Loc == LocAny {
		target = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: ctx.AllocStack(16), Rooted: true}
	} else {
		target = jitEnsureResultPair(ctx, result)
	}
	endLabel := ctx.ReserveLabel()
	branchState := ctx.SnapshotAllocState()
	hasBranch := false
	terminated := false
	baseEnv := ctx.Env
	i := 2
	for i+1 < len(list) {
		ctx.RestoreAllocState(branchState)
		nextLabel := ctx.ReserveLabel()
		branchEnv := jitMatchBranchEnv(baseEnv)
		outcome := jitCompileMatchPattern(ctx, value, list[i], branchEnv, nextLabel)
		if outcome.possible {
			ctx.Env = branchEnv
			branchValue := jitCompileExpr(ctx, list[i+1], sliceBase, target)
			ctx.Env = baseEnv
			_ = jitPlaceScmerIntoTarget(ctx, branchValue, target)
			ctx.EmitJmp(endLabel)
			hasBranch = true
		}
		ctx.MarkLabel(nextLabel)
		ctx.RestoreAllocState(branchState)
		i += 2
		if outcome.always {
			terminated = true
			break
		}
	}
	ctx.Env = baseEnv
	if !terminated {
		ctx.RestoreAllocState(branchState)
		var fallback JITValueDesc
		if i < len(list) {
			fallback = jitCompileExpr(ctx, list[i], sliceBase, target)
		} else {
			fallback = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
		}
		_ = jitPlaceScmerIntoTarget(ctx, fallback, target)
	}
	if hasBranch {
		ctx.MarkLabel(endLabel)
	}
	ctx.FreeDesc(&value)
	if target.Loc == LocRegPair {
		ctx.BindReg(target.Reg, &target)
		ctx.BindReg(target.Reg2, &target)
	}
	return target
}

// EmitNewSliceFromGoSlice retags a Go []Scmer header as a Scheme list without
// allocating or copying its backing storage. Ownership of the data pointer is
// transferred from slice to the returned descriptor.
