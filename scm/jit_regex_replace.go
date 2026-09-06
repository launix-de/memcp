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
	"regexp"
)

// jitReplaceBuilder accumulates the output of a regexp_replace-with-function
// scan. It only exists when the pattern actually matched at least once - the
// no-match path returns the input string untouched with no allocation.
type jitReplaceBuilder struct{ buf []byte }

func jitReplaceBuilderNew(capHint int64) *jitReplaceBuilder {
	if capHint < 0 {
		capHint = 0
	}
	return &jitReplaceBuilder{buf: make([]byte, 0, capHint)}
}

func jitReplaceBuilderAddSpan(rb *jitReplaceBuilder, s Scmer, lo, hi int64) {
	str := String(s)
	if lo < 0 {
		lo = 0
	}
	if hi > int64(len(str)) {
		hi = int64(len(str))
	}
	if hi > lo {
		rb.buf = append(rb.buf, str[lo:hi]...)
	}
}

func jitReplaceBuilderAddString(rb *jitReplaceBuilder, s Scmer) {
	rb.buf = append(rb.buf, String(s)...)
}

func jitReplaceBuilderResult(rb *jitReplaceBuilder) Scmer {
	return NewString(string(rb.buf))
}

// jitApply1 invokes a single-argument callable (the replacement function) - a
// non-variadic seam the JIT can call without marshalling an args slice.
func jitApply1(procedure, arg Scmer) Scmer {
	return Apply(procedure, arg)
}

// jitEmitConstantRegexpReplaceFunc lowers the hidden
// jit-constant-regexp-replace-func declaration - `(regexp_replace s "<const>"
// f)` with a constant pattern and a function replacement.
//
// It drives the existing regex JIT (jitEmitRegexScanReplace / emitMatchAt) over
// s. Zero matches: return s untouched, no allocation, no Go call. Otherwise
// accumulate gap + inlined-replacement segments into a jitReplaceBuilder. The
// replacement lambda is emitted inline when it is a plain proc; anything else
// (or a pattern we cannot scan) falls back to one native call.
func jitEmitConstantRegexpReplaceFunc(ctx *JITContext, pattern *regexp.Regexp, replacement Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
	replacement = replacement.WithoutSourceInfo()
	if !scmerCallable(replacement) {
		return jitConstRegexpReplaceCall(ctx, pattern, replacement, args[2], result)
	}
	program := jitCompileRegexProgram(pattern)

	input := ctx.stabilizeForNested(args[2])

	// A dedicated frame slot for the input string pair - the callbacks run
	// inside PreserveOuterRegs windows where the stabilized `input` desc's
	// backing can be reused, so they always reload from here instead.
	inputOff := ctx.AllocStack(16)
	rbOff := ctx.AllocStack(8)
	lastOff := ctx.AllocStack(8)
	{
		src := input
		ctx.EnsureDesc(&src)
		dst := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: inputOff}
		ctx.EmitCopyScmerToDesc(&dst, &src)
	}
	ctx.EmitMovRegImm64(ctx.ScratchReg, 0)
	ctx.EmitStoreRegMem(ctx.ScratchReg, ctx.StackReg, rbOff)
	ctx.EmitStoreRegMem(ctx.ScratchReg, ctx.StackReg, lastOff)
	ctx.setStackPointer(jitStackRootFrameSP, inputOff-ctx.DynamicSP, true)
	ctx.setStackPointer(jitStackRootFrameSP, rbOff-ctx.DynamicSP, true)

	// inputPair is the string desc every callback reloads from the frame slot.
	inputPair := func() JITValueDesc {
		return JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: inputOff}
	}

	// inputLenReg loads len(input) into a fresh caller-owned register.
	inputLenReg := func() JITValueDesc {
		v := inputPair()
		ctx.EnsureDesc(&v)
		reg := ctx.AllocRegExcept(v.Reg, v.Reg2)
		ctx.EmitMovRegReg(reg, v.Reg2)
		ctx.EmitShrRegImm8(reg, 8)
		ctx.FreeDesc(&v)
		return JITValueDesc{Loc: LocReg, Type: tagInt, Reg: reg, NoHeapPointer: true}
	}

	// loadBuilder loads *jitReplaceBuilder from rbOff, creating it (lazily, with
	// a len(input) capacity hint) on the first call. Returns a fresh register.
	loadBuilder := func() JITValueDesc {
		rb := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: ctx.AllocReg(), NoHeapPointer: false}
		ctx.EmitMovRegMem(rb.Reg, ctx.StackReg, rbOff)
		ctx.EmitCmpRegImm32(rb.Reg, 0)
		have := ctx.ReserveLabel()
		ctx.EmitJump(CondNotEqual, have)
		ln := inputLenReg()
		made := ctx.EmitGoCallScalar(GoFuncAddr(jitReplaceBuilderNew), []JITValueDesc{ln}, 1)
		ctx.FreeDesc(&ln)
		ctx.EmitMovRegReg(rb.Reg, made.Reg)
		ctx.FreeDesc(&made)
		ctx.EmitStoreRegMem(rb.Reg, ctx.StackReg, rbOff)
		ctx.MarkLabel(have)
		return rb
	}

	// appendSpan: builder.buf += input[loOff : hiOff]. lo/hi are int descs.
	appendSpan := func(loOff, hiOff JITValueDesc) {
		rb := loadBuilder()
		v := inputPair()
		ctx.EnsureDesc(&v)
		ctx.EmitGoCallVoid(GoFuncAddr(jitReplaceBuilderAddSpan), []JITValueDesc{rb, v, loOff, hiOff})
		ctx.FreeDesc(&v)
		ctx.FreeDesc(&rb)
	}

	onMatch := func(startOff, endOff JITValueDesc) {
		last := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: lastOff, NoHeapPointer: true}
		appendSpan(last, startOff)

		// m = input[startOff:endOff] as a string pair
		v := inputPair()
		ctx.EnsureDesc(&v)
		mptr := ctx.AllocRegExcept(v.Reg, v.Reg2)
		soReg := ctx.AllocRegExcept(v.Reg, v.Reg2, mptr)
		ctx.EmitMovRegMem(soReg, ctx.StackReg, startOff.StackOff)
		ctx.EmitMovRegReg(mptr, v.Reg)
		ctx.EmitAddInt64(mptr, soReg)
		mlen := ctx.AllocRegExcept(v.Reg, v.Reg2, mptr, soReg)
		ctx.EmitMovRegMem(mlen, ctx.StackReg, endOff.StackOff)
		ctx.EmitSubInt64(mlen, soReg)
		ctx.FreeReg(soReg)
		ctx.FreeDesc(&v)
		ctx.EmitShlRegImm8(mlen, 8)
		ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(tagString))
		ctx.EmitOrInt64(mlen, ctx.ScratchReg)
		m := JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: mptr, Reg2: mlen}
		ctx.BindReg(mptr, &m)
		ctx.BindReg(mlen, &m)

		procArg := jitCopyScmerToPair(ctx, JITValueDesc{Loc: LocImm, Type: replacement.GetTag(), Imm: replacement})
		ctx.TrackImm(replacement)
		repl := ctx.EmitGoCallScalar(GoFuncAddr(jitApply1), []JITValueDesc{procArg, m}, 2)
		repl.Type = JITTypeUnknown
		repl.Rooted = true
		ctx.FreeDesc(&procArg)
		ctx.FreeDesc(&m)

		rb := loadBuilder()
		ctx.EmitGoCallVoid(GoFuncAddr(jitReplaceBuilderAddString), []JITValueDesc{rb, repl})
		ctx.FreeDesc(&repl)
		ctx.FreeDesc(&rb)

		// lastOff = endOff
		eo := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: ctx.AllocReg(), NoHeapPointer: true}
		ctx.EmitMovRegMem(eo.Reg, ctx.StackReg, endOff.StackOff)
		ctx.EmitStoreRegMem(eo.Reg, ctx.StackReg, lastOff)
		ctx.FreeDesc(&eo)
	}

	onEnd := func(resultSlot JITValueDesc) {
		probe := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: ctx.AllocReg(), NoHeapPointer: true}
		ctx.EmitMovRegMem(probe.Reg, ctx.StackReg, rbOff)
		ctx.EmitCmpRegImm32(probe.Reg, 0)
		ctx.FreeDesc(&probe)
		matched := ctx.ReserveLabel()
		after := ctx.ReserveLabel()
		ctx.EmitJump(CondNotEqual, matched)
		// no match at all: result = input, untouched
		pass := inputPair()
		ctx.EnsureDesc(&pass)
		dst := resultSlot
		ctx.EmitCopyScmerToDesc(&dst, &pass)
		ctx.FreeDesc(&pass)
		ctx.EmitJmp(after)

		ctx.MarkLabel(matched)
		lastLen := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: lastOff, NoHeapPointer: true}
		ln := inputLenReg()
		appendSpan(lastLen, ln) // builder.buf += input[last:len]
		ctx.FreeDesc(&ln)
		rb := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: ctx.AllocReg(), NoHeapPointer: false}
		ctx.EmitMovRegMem(rb.Reg, ctx.StackReg, rbOff)
		out := ctx.EmitGoCallScalar(GoFuncAddr(jitReplaceBuilderResult), []JITValueDesc{rb}, 2)
		ctx.FreeDesc(&rb)
		out.Type = JITTypeUnknown
		out.Rooted = true
		dst2 := resultSlot
		ctx.EmitCopyScmerToDesc(&dst2, &out)
		ctx.FreeDesc(&out)
		ctx.MarkLabel(after)
	}

	nilPath := func(resultSlot JITValueDesc) {
		dst := resultSlot
		ctx.EmitMakeNil(dst)
	}

	resultSlot := jitEmitRegexScanReplace(ctx, program, input, onMatch, onEnd, nilPath)
	ctx.FreeDesc(&input)
	return jitPlaceScmerIntoTarget(ctx, resultSlot, result)
}

// jitConstRegexpReplaceCall is the whole-operation fallback: one native call
// running the precompiled regex with the replacement over src.
func jitConstRegexpReplaceCall(ctx *JITContext, pattern *regexp.Regexp, replacement Scmer, srcDesc, result JITValueDesc) JITValueDesc {
	reArg := jitCopyScmerToPair(ctx, JITValueDesc{Loc: LocImm, Type: tagRegex, Imm: NewRegex(pattern)})
	ctx.TrackImm(NewRegex(pattern))
	replArg := jitCopyScmerToPair(ctx, JITValueDesc{Loc: LocImm, Type: replacement.GetTag(), Imm: replacement})
	ctx.TrackImm(replacement)
	value := srcDesc
	ctx.EnsureDesc(&value)
	out := ctx.EmitGoCallScalar(GoFuncAddr(jitConstantRegexpReplaceFunc), []JITValueDesc{reArg, replArg, value}, 2)
	out.Type = JITTypeUnknown
	out.Rooted = true
	ctx.FreeDesc(&reArg)
	ctx.FreeDesc(&replArg)
	if result.Loc == LocAny {
		return out
	}
	return jitPlaceScmerIntoTarget(ctx, out, result)
}
