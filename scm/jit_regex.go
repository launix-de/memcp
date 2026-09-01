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

import "regexp"

const (
	jitConstantRegexpTestName      = "jit-constant-regexp-test"
	jitConstantRegexpPredicateName = "jit-constant-regexp-predicate"
)

func jitConstantRegexpTest(pattern, value Scmer) Scmer {
	if value.IsNil() {
		return NewNil()
	}
	return NewBool(pattern.Regex().MatchString(String(value)))
}

func jitConstantRegexpPredicate(pattern, value Scmer) bool {
	text, ok := scmerAsString(value)
	if !ok {
		panic("regex expects string")
	}
	return pattern.Regex().MatchString(text)
}

func jitConstantRegexpCaptureIndices(pattern, value Scmer) []int {
	text, ok := scmerAsString(value)
	if !ok {
		panic("regex expects string")
	}
	return pattern.Regex().FindStringSubmatchIndex(text)
}

func jitRegexpScmerArg(ctx *JITContext, value JITValueDesc) JITValueDesc {
	switch value.Loc {
	case LocRegPair, LocStackPair, LocInputPair:
		return value
	case LocImm:
		return jitCopyScmerToPair(ctx, value)
	default:
		ctx.EnsureDesc(&value)
		if value.Loc != LocRegPair && value.Loc != LocStackPair {
			panic("jit: regex expects a Scmer value")
		}
		return value
	}
}

func jitRegexpPatternArg(ctx *JITContext, pattern *regexp.Regexp) JITValueDesc {
	if pattern == nil {
		panic("jit: nil constant regex")
	}
	value := NewRegex(pattern)
	ctx.TrackImm(value)
	return jitCopyScmerToPair(ctx, JITValueDesc{Loc: LocImm, Type: tagRegex, Imm: value})
}

// jitEmitConstantRegexpTest emits a direct call for a precompiled regular
// expression. regexp_test's nil result remains distinct from a false match.
func jitEmitConstantRegexpTest(ctx *JITContext, pattern *regexp.Regexp, value JITValueDesc, result JITValueDesc) JITValueDesc {
	patternArg := jitRegexpPatternArg(ctx, pattern)
	valueArg := jitRegexpScmerArg(ctx, value)
	target := jitEnsureResultPair(ctx, result)
	out := ctx.EmitGoCallScalarInto(GoFuncAddr(jitConstantRegexpTest), []JITValueDesc{patternArg, valueArg}, target)
	ctx.FreeDesc(&patternArg)
	if value.Loc == LocImm {
		ctx.FreeDesc(&valueArg)
	}
	out.Type = JITTypeUnknown
	return out
}

// jitEmitConstantRegexpPredicate emits the branch-friendly boolean form used
// by match and, later, compiled parser terminals.
func jitEmitConstantRegexpPredicate(ctx *JITContext, pattern *regexp.Regexp, value JITValueDesc) JITValueDesc {
	patternArg := jitRegexpPatternArg(ctx, pattern)
	valueArg := jitRegexpScmerArg(ctx, value)
	out := ctx.EmitGoCallScalar(GoFuncAddr(jitConstantRegexpPredicate), []JITValueDesc{patternArg, valueArg}, 1)
	ctx.FreeDesc(&patternArg)
	if value.Loc == LocImm {
		ctx.FreeDesc(&valueArg)
	}
	out.Type = tagBool
	out.NoHeapPointer = true
	return out
}

// jitEmitConstantRegexpCaptures emits one regex execution, branches directly
// to failLabel when it does not match, and returns stack-backed capture values.
// Spilling each capture immediately keeps register pressure independent of the
// number of subexpressions.
func jitEmitConstantRegexpCaptures(ctx *JITContext, pattern *regexp.Regexp, value JITValueDesc, failLabel uint8) []JITValueDesc {
	stableValue := jitMatchStableValue(ctx, value)
	patternArg := jitRegexpPatternArg(ctx, pattern)
	valueArg := jitRegexpScmerArg(ctx, stableValue)
	header := ctx.EmitGoCallScalar(GoFuncAddr(jitConstantRegexpCaptureIndices), []JITValueDesc{patternArg, valueArg}, 3)
	ctx.FreeDesc(&patternArg)

	ctx.EmitCmpRegImm32(header.Reg, 0)
	ctx.EmitJump(CondEqual, failLabel)
	ctx.ProtectReg(header.Reg)
	ctx.ProtectReg(header.Reg2)
	ctx.ProtectReg(header.Reg3)
	text := stableValue
	ctx.EnsureDesc(&text)
	if text.Loc != LocRegPair {
		panic("jit: regex capture input is not a Scmer pair")
	}
	ctx.ProtectReg(text.Reg)
	ctx.ProtectReg(text.Reg2)
	captures := make([]JITValueDesc, pattern.NumSubexp()+1)
	for index := range captures {
		startReg := ctx.AllocRegExcept(header.Reg, header.Reg2, header.Reg3, text.Reg, text.Reg2)
		endReg := ctx.AllocRegExcept(header.Reg, header.Reg2, header.Reg3, text.Reg, text.Reg2, startReg)
		ctx.EmitMovRegMem(startReg, header.Reg, int32(index*16))
		ctx.EmitMovRegMem(endReg, header.Reg, int32(index*16+8))
		unmatchedLabel := ctx.ReserveLabel()
		captureLabel := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(startReg, -1)
		ctx.EmitJump(CondEqual, unmatchedLabel)
		ptrReg := ctx.AllocRegExcept(header.Reg, header.Reg2, header.Reg3, text.Reg, text.Reg2, startReg, endReg)
		ctx.EmitMovRegReg(ptrReg, text.Reg)
		ctx.EmitAddInt64(ptrReg, startReg)
		ctx.EmitSubInt64(endReg, startReg)
		ctx.EmitShlRegImm8(endReg, 8)
		ctx.EmitOrRegImm32(endReg, int32(tagString))
		ctx.EmitJmp(captureLabel)
		ctx.MarkLabel(unmatchedLabel)
		ctx.EmitMovRegImm64(ptrReg, 0)
		ctx.EmitMovRegImm64(endReg, uint64(tagString))
		ctx.MarkLabel(captureLabel)
		ctx.FreeReg(startReg)
		auxReg := endReg
		capture := JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ptrReg, Reg2: auxReg, Rooted: true}
		ctx.BindReg(ptrReg, &capture)
		ctx.BindReg(auxReg, &capture)
		off := ctx.AllocStack(16)
		ctx.EmitStoreScmerToStack(capture, off)
		ctx.FreeDesc(&capture)
		captures[index] = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: off, Rooted: true}
	}
	ctx.UnprotectReg(text.Reg2)
	ctx.UnprotectReg(text.Reg)
	ctx.FreeDesc(&text)
	ctx.UnprotectReg(header.Reg3)
	ctx.UnprotectReg(header.Reg2)
	ctx.UnprotectReg(header.Reg)
	ctx.FreeDesc(&header)
	return captures
}

func registerJITRegexBuiltins() {
	Declare(&Globalenv, &Declaration{
		Name: jitConstantRegexpTestName,
		Fn: func(arguments ...Scmer) Scmer {
			if len(arguments) != 2 || !arguments[0].IsRegex() {
				panic("jit constant regexp test expects a precompiled regex and a value")
			}
			return jitConstantRegexpTest(arguments[0], arguments[1])
		},
		Type: &TypeDescriptor{
			Kind:      "func",
			Forbidden: true,
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "pattern"},
				{Kind: "any", Label: "value"},
			},
			Return:         &TypeDescriptor{Kind: "any"},
			Const:          true,
			JITVirtualArgs: true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if len(sourceArgs) != 2 || len(args) != 2 {
					panic("jit: malformed constant regexp test")
				}
				pattern := sourceArgs[0].WithoutSourceInfo()
				if !pattern.IsRegex() {
					panic("jit: constant regexp test requires a precompiled regex")
				}
				return jitEmitConstantRegexpTest(ctx, pattern.Regex(), args[1], result)
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: jitConstantRegexpPredicateName,
		Fn: func(arguments ...Scmer) Scmer {
			if len(arguments) != 2 || !arguments[0].IsRegex() {
				panic("jit constant regexp predicate expects a precompiled regex and a value")
			}
			return NewBool(jitConstantRegexpPredicate(arguments[0], arguments[1]))
		},
		Type: &TypeDescriptor{
			Kind:      "func",
			Forbidden: true,
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "pattern"},
				{Kind: "any", Label: "value"},
			},
			Return:         &TypeDescriptor{Kind: "bool"},
			Const:          true,
			JITVirtualArgs: true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if len(sourceArgs) != 2 || len(args) != 2 {
					panic("jit: malformed constant regexp predicate")
				}
				pattern := sourceArgs[0].WithoutSourceInfo()
				if !pattern.IsRegex() {
					panic("jit: constant regexp predicate requires a precompiled regex")
				}
				predicate := jitEmitConstantRegexpPredicate(ctx, pattern.Regex(), args[1])
				return jitPlaceScmerIntoTarget(ctx, predicate, result)
			},
		},
	})
}
