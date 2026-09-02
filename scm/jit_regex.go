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
	"fmt"
	"math/bits"
	"regexp"
	"regexp/syntax"
)

const (
	jitConstantRegexpTestName      = "jit-constant-regexp-test"
	jitConstantRegexpPredicateName = "jit-constant-regexp-predicate"
)

// jitConstantRegexpTest is the interpreter implementation of the hidden
// declaration. Its JIT emitter below never calls it or regexp at runtime.
func jitConstantRegexpTest(pattern, value Scmer) Scmer {
	if value.IsNil() {
		return NewNil()
	}
	return NewBool(pattern.Regex().MatchString(String(value)))
}

// jitRegexProgram is compile-time-only, architecture-independent regex IR.
// regexp/syntax is used while producing machine code; generated code contains
// only the specialized byte walker emitted below.
type jitRegexProgram struct {
	root      *syntax.Regexp
	captures  int
	beginText bool
	pattern   string
}

type jitRegexTermKind uint8

const (
	jitRegexNode jitRegexTermKind = iota
	jitRegexCaptureBegin
	jitRegexCaptureEnd
)

type jitRegexTerm struct {
	kind    jitRegexTermKind
	node    *syntax.Regexp
	capture int
}

func jitCompileRegexProgram(pattern *regexp.Regexp) *jitRegexProgram {
	if pattern == nil {
		panic("jit: nil constant regex")
	}
	source := pattern.String()
	root, err := syntax.Parse(source, syntax.Perl)
	if err != nil {
		panic("jit: parse precompiled regex: " + err.Error())
	}
	if root.MaxCap() != pattern.NumSubexp() {
		panic("jit: regex capture inventory changed while lowering")
	}
	return &jitRegexProgram{
		root:      root,
		captures:  pattern.NumSubexp() + 1,
		beginText: jitRegexBeginsAtText(root),
		pattern:   source,
	}
}

func jitRegexBeginsAtText(node *syntax.Regexp) bool {
	switch node.Op {
	case syntax.OpBeginText:
		return true
	case syntax.OpCapture:
		return jitRegexBeginsAtText(node.Sub[0])
	case syntax.OpConcat:
		for _, child := range node.Sub {
			if child.Op == syntax.OpEmptyMatch {
				continue
			}
			return jitRegexBeginsAtText(child)
		}
	}
	return false
}

func jitRegexFlatten(node *syntax.Regexp, keepCaptures bool) []jitRegexTerm {
	switch node.Op {
	case syntax.OpConcat:
		var terms []jitRegexTerm
		for _, child := range node.Sub {
			terms = append(terms, jitRegexFlatten(child, keepCaptures)...)
		}
		return terms
	case syntax.OpCapture:
		if !keepCaptures {
			return jitRegexFlatten(node.Sub[0], false)
		}
		terms := []jitRegexTerm{{kind: jitRegexCaptureBegin, capture: node.Cap}}
		terms = append(terms, jitRegexFlatten(node.Sub[0], true)...)
		return append(terms, jitRegexTerm{kind: jitRegexCaptureEnd, capture: node.Cap})
	default:
		return []jitRegexTerm{{kind: jitRegexNode, node: node}}
	}
}

type jitRegexEmitter struct {
	ctx           *JITContext
	program       *jitRegexProgram
	cursor        Reg
	end           Reg
	scan          Reg
	inputStartOff int32
	captureStarts []int32
	captures      []JITValueDesc
	stackStart    int32
}

func (emitter *jitRegexEmitter) reserveMachineState(input JITValueDesc, invalidLabel JITLabel, nilLabel *JITLabel, stringify bool) {
	emitter.stackStart = emitter.ctx.BPOffset
	// A regex is a self-contained control-flow region. Path-local temporaries
	// from the producer are dead here unless they still own a descriptor; make
	// those registers available before reserving cursor/end/scan.
	emitter.ctx.ReclaimUntrackedRegs()
	original := input
	if stringify {
		original = jitMatchStableValue(emitter.ctx, input)
	}
	text := original
	emitter.ctx.EnsureDesc(&text)
	if text.Loc != LocRegPair {
		panic("jit: native regex expects a Scmer pair")
	}
	emitter.cursor = emitter.ctx.AllocRegExcept(text.Reg, text.Reg2)
	emitter.end = emitter.ctx.AllocRegExcept(text.Reg, text.Reg2, emitter.cursor)
	emitter.scan = emitter.ctx.AllocRegExcept(text.Reg, text.Reg2, emitter.cursor, emitter.end)
	emitter.ctx.ProtectReg(emitter.cursor)
	emitter.ctx.ProtectReg(emitter.end)
	emitter.ctx.ProtectReg(emitter.scan)

	if text.Type == JITTypeUnknown {
		tag := emitter.ctx.AllocRegExcept(text.Reg, text.Reg2)
		emitter.ctx.EmitMovRegReg(tag, text.Reg2)
		emitter.ctx.EmitAndRegImm32(tag, 0xff)
		emitter.ctx.EmitCmpRegImm32(tag, int32(tagString))
		valid := emitter.ctx.ReserveLabel()
		emitter.ctx.EmitJump(CondEqual, valid)
		if nilLabel != nil {
			emitter.ctx.EmitCmpRegImm32(tag, int32(tagNil))
			emitter.ctx.EmitJump(CondEqual, *nilLabel)
		}
		converted := invalidLabel
		if stringify {
			converted = emitter.ctx.ReserveLabel()
		}
		emitter.ctx.EmitJmp(converted)
		emitter.ctx.MarkLabel(valid)
		emitter.ctx.FreeReg(tag)
		emitter.emitTaggedStringState(text)
		if stringify {
			ready := emitter.ctx.ReserveLabel()
			emitter.ctx.EmitJmp(ready)
			emitter.ctx.MarkLabel(converted)
			emitter.ctx.FreeDesc(&text)
			goString := emitter.ctx.EmitGoCallScalar(GoFuncAddr(String), []JITValueDesc{original}, 2)
			emitter.emitGoStringState(goString)
			emitter.ctx.FreeDesc(&goString)
			emitter.ctx.MarkLabel(ready)
		} else {
			emitter.ctx.FreeDesc(&text)
		}
	} else if text.Type == tagNil && nilLabel != nil {
		emitter.ctx.EmitJmp(*nilLabel)
		emitter.emitTaggedStringState(text)
		emitter.ctx.FreeDesc(&text)
	} else if text.Type != tagString {
		if !stringify {
			emitter.ctx.EmitJmp(invalidLabel)
			emitter.emitTaggedStringState(text)
			emitter.ctx.FreeDesc(&text)
		} else {
			emitter.ctx.FreeDesc(&text)
			goString := emitter.ctx.EmitGoCallScalar(GoFuncAddr(String), []JITValueDesc{original}, 2)
			emitter.emitGoStringState(goString)
			emitter.ctx.FreeDesc(&goString)
		}
	} else {
		emitter.emitTaggedStringState(text)
		emitter.ctx.FreeDesc(&text)
	}

	emitter.inputStartOff = emitter.ctx.AllocStack(8)
	emitter.ctx.EmitStoreRegMem(emitter.cursor, emitter.ctx.StackReg, emitter.inputStartOff)
	if len(emitter.captures) != 0 {
		emitter.captureStarts = make([]int32, len(emitter.captures))
		for index := range emitter.captureStarts {
			emitter.captureStarts[index] = emitter.ctx.AllocStack(8)
		}
	}
}

func (emitter *jitRegexEmitter) emitTaggedStringState(text JITValueDesc) {
	emitter.ctx.EmitMovRegReg(emitter.cursor, text.Reg)
	emitter.ctx.EmitMovRegReg(emitter.end, text.Reg2)
	emitter.ctx.EmitShrRegImm8(emitter.end, 8)
	emitter.ctx.EmitAddInt64(emitter.end, emitter.cursor)
	emitter.ctx.EmitMovRegReg(emitter.scan, emitter.cursor)
}

func (emitter *jitRegexEmitter) emitGoStringState(text JITValueDesc) {
	emitter.ctx.EmitMovRegReg(emitter.cursor, text.Reg)
	emitter.ctx.EmitMovRegReg(emitter.end, text.Reg2)
	emitter.ctx.EmitAddInt64(emitter.end, emitter.cursor)
	emitter.ctx.EmitMovRegReg(emitter.scan, emitter.cursor)
}

func (emitter *jitRegexEmitter) releaseMachineState() {
	emitter.ctx.UnprotectReg(emitter.scan)
	emitter.ctx.UnprotectReg(emitter.end)
	emitter.ctx.UnprotectReg(emitter.cursor)
	emitter.ctx.FreeReg(emitter.scan)
	emitter.ctx.FreeReg(emitter.end)
	emitter.ctx.FreeReg(emitter.cursor)
	emitter.ctx.FreeStack(emitter.ctx.BPOffset - emitter.stackStart)
}

func (emitter *jitRegexEmitter) emitCaptureEmpty(index int) {
	if index < 0 || index >= len(emitter.captures) {
		return
	}
	target := emitter.captures[index]
	if target.Loc != LocStackPair {
		panic("jit: regex capture target must be a stable JITValueDesc")
	}
	emitter.ctx.EmitMovRegImm64(emitter.ctx.ScratchReg, 0)
	emitter.ctx.EmitStoreRegMem(emitter.ctx.ScratchReg, emitter.ctx.StackReg, target.StackOff)
	emitter.ctx.EmitMovRegImm64(emitter.ctx.ScratchReg, uint64(tagString))
	emitter.ctx.EmitStoreRegMem(emitter.ctx.ScratchReg, emitter.ctx.StackReg, target.StackOff+8)
	emitter.ctx.setStackPointer(jitStackRootFrameSP, target.StackOff-emitter.ctx.DynamicSP, true)
}

func (emitter *jitRegexEmitter) emitCaptureBegin(index int) {
	if index < 0 || index >= len(emitter.captureStarts) {
		return
	}
	emitter.ctx.EmitStoreRegMem(emitter.cursor, emitter.ctx.StackReg, emitter.captureStarts[index])
}

func (emitter *jitRegexEmitter) emitCaptureEnd(index int) {
	if index < 0 || index >= len(emitter.captures) {
		return
	}
	target := emitter.captures[index]
	if target.Loc != LocStackPair {
		panic("jit: regex capture target must be a stable JITValueDesc")
	}
	start := emitter.ctx.AllocRegExcept(emitter.cursor, emitter.end, emitter.scan)
	aux := emitter.ctx.AllocRegExcept(emitter.cursor, emitter.end, emitter.scan, start)
	emitter.ctx.EmitMovRegMem(start, emitter.ctx.StackReg, emitter.captureStarts[index])
	emitter.ctx.EmitMovRegReg(aux, emitter.cursor)
	emitter.ctx.EmitSubInt64(aux, start)
	emitter.ctx.EmitShlRegImm8(aux, 8)
	emitter.ctx.EmitOrRegImm32(aux, int32(tagString))
	emitter.ctx.EmitStoreRegMem(start, emitter.ctx.StackReg, target.StackOff)
	emitter.ctx.EmitStoreRegMem(aux, emitter.ctx.StackReg, target.StackOff+8)
	emitter.ctx.setStackPointer(jitStackRootFrameSP, target.StackOff-emitter.ctx.DynamicSP, true)
	emitter.ctx.FreeReg(aux)
	emitter.ctx.FreeReg(start)
}

func (emitter *jitRegexEmitter) emitResetCaptures(terms []jitRegexTerm) {
	if len(emitter.captures) == 0 {
		return
	}
	seen := make([]bool, len(emitter.captures))
	var visit func([]jitRegexTerm)
	visit = func(items []jitRegexTerm) {
		for _, term := range items {
			if term.kind != jitRegexNode {
				if term.capture >= 0 && term.capture < len(seen) {
					seen[term.capture] = true
				}
				continue
			}
			if term.node.Op == syntax.OpAlternate {
				for _, branch := range term.node.Sub {
					visit(jitRegexFlatten(branch, true))
				}
			}
		}
	}
	visit(terms)
	for index, reset := range seen {
		if reset {
			emitter.emitCaptureEmpty(index)
		}
	}
}

func (emitter *jitRegexEmitter) emitProgram(successLabel, failLabel JITLabel) {
	attemptLabel := emitter.ctx.ReserveLabel()
	attemptFailLabel := emitter.ctx.ReserveLabel()
	emitter.ctx.MarkLabel(attemptLabel)
	emitter.ctx.EmitMovRegReg(emitter.cursor, emitter.scan)
	for index := range emitter.captures {
		emitter.emitCaptureEmpty(index)
	}
	terms := jitRegexFlatten(emitter.program.root, len(emitter.captures) != 0)
	if len(emitter.captures) != 0 {
		terms = append([]jitRegexTerm{{kind: jitRegexCaptureBegin, capture: 0}}, terms...)
		terms = append(terms, jitRegexTerm{kind: jitRegexCaptureEnd, capture: 0})
	}
	emitter.emitSequence(terms, successLabel, attemptFailLabel)

	emitter.ctx.MarkLabel(attemptFailLabel)
	if emitter.program.beginText {
		emitter.ctx.EmitJmp(failLabel)
		return
	}
	emitter.ctx.EmitAddRegImm32(emitter.scan, 1)
	emitter.ctx.EmitCmpInt64(emitter.scan, emitter.end)
	emitter.ctx.EmitJump(CondUnsignedBelowOrEqual, attemptLabel)
	emitter.ctx.EmitJmp(failLabel)
}

func (emitter *jitRegexEmitter) emitSequence(terms []jitRegexTerm, successLabel, failLabel JITLabel) {
	for len(terms) != 0 {
		term := terms[0]
		terms = terms[1:]
		switch term.kind {
		case jitRegexCaptureBegin:
			emitter.emitCaptureBegin(term.capture)
			continue
		case jitRegexCaptureEnd:
			emitter.emitCaptureEnd(term.capture)
			continue
		}

		node := term.node
		switch node.Op {
		case syntax.OpAlternate:
			emitter.emitAlternatives(node.Sub, terms, successLabel, failLabel)
			return
		case syntax.OpQuest:
			emitter.emitOptional(node.Sub[0], terms, successLabel, failLabel, node.Flags&syntax.NonGreedy == 0)
			return
		case syntax.OpStar:
			emitter.emitRepeat(node.Sub[0], 0, -1, node.Flags&syntax.NonGreedy == 0, terms, successLabel, failLabel)
			return
		case syntax.OpPlus:
			emitter.emitRepeat(node.Sub[0], 1, -1, node.Flags&syntax.NonGreedy == 0, terms, successLabel, failLabel)
			return
		case syntax.OpRepeat:
			emitter.emitRepeat(node.Sub[0], node.Min, node.Max, node.Flags&syntax.NonGreedy == 0, terms, successLabel, failLabel)
			return
		default:
			emitter.emitAtom(node, failLabel)
		}
	}
	emitter.ctx.EmitJmp(successLabel)
}

func (emitter *jitRegexEmitter) emitAlternatives(branches []*syntax.Regexp, rest []jitRegexTerm, successLabel, failLabel JITLabel) {
	if len(branches) == 0 {
		emitter.ctx.EmitJmp(failLabel)
		return
	}
	savedCursor := emitter.ctx.AllocStack(8)
	emitter.ctx.EmitStoreRegMem(emitter.cursor, emitter.ctx.StackReg, savedCursor)
	for index, branch := range branches {
		branchTerms := append(jitRegexFlatten(branch, len(emitter.captures) != 0), rest...)
		if index == len(branches)-1 {
			emitter.emitSequence(branchTerms, successLabel, failLabel)
			return
		}
		next := emitter.ctx.ReserveLabel()
		emitter.emitSequence(branchTerms, successLabel, next)
		emitter.ctx.MarkLabel(next)
		emitter.ctx.EmitMovRegMem(emitter.cursor, emitter.ctx.StackReg, savedCursor)
		emitter.emitResetCaptures(branchTerms)
	}
}

func (emitter *jitRegexEmitter) emitOptional(node *syntax.Regexp, rest []jitRegexTerm, successLabel, failLabel JITLabel, greedy bool) {
	empty := &syntax.Regexp{Op: syntax.OpEmptyMatch}
	branches := []*syntax.Regexp{node, empty}
	if !greedy {
		branches[0], branches[1] = branches[1], branches[0]
	}
	emitter.emitAlternatives(branches, rest, successLabel, failLabel)
}

func jitRegexSimpleWidth(node *syntax.Regexp) (int, bool) {
	switch node.Op {
	case syntax.OpLiteral:
		return len(string(node.Rune)), len(node.Rune) != 0
	case syntax.OpCharClass, syntax.OpAnyCharNotNL, syntax.OpAnyChar:
		return 1, true
	case syntax.OpConcat:
		width := 0
		for _, child := range node.Sub {
			childWidth, simple := jitRegexSimpleWidth(child)
			if !simple {
				return 0, false
			}
			width += childWidth
		}
		return width, width != 0
	}
	return 0, false
}

func (emitter *jitRegexEmitter) emitSimple(node *syntax.Regexp, failLabel JITLabel) {
	if node.Op == syntax.OpConcat {
		for _, child := range node.Sub {
			emitter.emitSimple(child, failLabel)
		}
		return
	}
	emitter.emitAtom(node, failLabel)
}

func (emitter *jitRegexEmitter) emitRepeat(node *syntax.Regexp, min, max int, greedy bool, rest []jitRegexTerm, successLabel, failLabel JITLabel) {
	if max >= 0 {
		if max < min {
			panic("jit: invalid regex repetition")
		}
		if min != 0 {
			required := make([]jitRegexTerm, 0, min+len(rest))
			for range min {
				required = append(required, jitRegexFlatten(node, len(emitter.captures) != 0)...)
			}
			if max != min {
				optional := &syntax.Regexp{Op: syntax.OpRepeat, Min: 0, Max: max - min, Flags: syntax.PerlX, Sub: []*syntax.Regexp{node}}
				if !greedy {
					optional.Flags |= syntax.NonGreedy
				}
				required = append(required, jitRegexTerm{kind: jitRegexNode, node: optional})
			}
			emitter.emitSequence(append(required, rest...), successLabel, failLabel)
			return
		}
		if max == 0 {
			emitter.emitSequence(rest, successLabel, failLabel)
			return
		}
		if max == 1 {
			emitter.emitOptional(node, rest, successLabel, failLabel, greedy)
			return
		}
		next := &syntax.Regexp{Op: syntax.OpRepeat, Min: 0, Max: max - 1, Flags: syntax.PerlX, Sub: []*syntax.Regexp{node}}
		if !greedy {
			next.Flags |= syntax.NonGreedy
		}
		combined := &syntax.Regexp{Op: syntax.OpConcat, Sub: []*syntax.Regexp{node, next}}
		emitter.emitOptional(combined, rest, successLabel, failLabel, greedy)
		return
	}

	width, simple := jitRegexSimpleWidth(node)
	if simple {
		emitter.emitSimpleRepeat(node, width, min, greedy, rest, successLabel, failLabel)
		return
	}
	if !jitRegexTailOnly(rest) {
		panic(fmt.Sprintf("jit: regex %q requires a variable-width repetition stack", emitter.program.pattern))
	}
	emitter.emitComplexTailRepeat(node, min, rest, successLabel, failLabel)
}

func jitRegexTailOnly(terms []jitRegexTerm) bool {
	for _, term := range terms {
		if term.kind == jitRegexCaptureEnd {
			continue
		}
		if term.kind == jitRegexNode && (term.node.Op == syntax.OpEndText || term.node.Op == syntax.OpEndLine || term.node.Op == syntax.OpEmptyMatch) {
			continue
		}
		return false
	}
	return true
}

func (emitter *jitRegexEmitter) emitSimpleRepeat(node *syntax.Regexp, width, min int, greedy bool, rest []jitRegexTerm, successLabel, failLabel JITLabel) {
	for range min {
		emitter.emitSimple(node, failLabel)
	}
	minimumCursor := emitter.ctx.AllocStack(8)
	emitter.ctx.EmitStoreRegMem(emitter.cursor, emitter.ctx.StackReg, minimumCursor)
	if !greedy {
		retry := emitter.ctx.ReserveLabel()
		grow := emitter.ctx.ReserveLabel()
		emitter.ctx.MarkLabel(retry)
		candidate := emitter.ctx.AllocStack(8)
		emitter.ctx.EmitStoreRegMem(emitter.cursor, emitter.ctx.StackReg, candidate)
		emitter.emitSequence(rest, successLabel, grow)
		emitter.ctx.MarkLabel(grow)
		emitter.ctx.EmitMovRegMem(emitter.cursor, emitter.ctx.StackReg, candidate)
		emitter.emitSimple(node, failLabel)
		emitter.ctx.EmitJmp(retry)
		return
	}

	loop := emitter.ctx.ReserveLabel()
	done := emitter.ctx.ReserveLabel()
	retry := emitter.ctx.ReserveLabel()
	backtrack := emitter.ctx.ReserveLabel()
	trial := emitter.ctx.AllocStack(8)
	candidate := emitter.ctx.AllocStack(8)
	emitter.ctx.MarkLabel(loop)
	emitter.ctx.EmitStoreRegMem(emitter.cursor, emitter.ctx.StackReg, trial)
	emitter.emitSimple(node, done)
	emitter.ctx.EmitJmp(loop)
	emitter.ctx.MarkLabel(done)
	emitter.ctx.EmitMovRegMem(emitter.cursor, emitter.ctx.StackReg, trial)
	emitter.ctx.MarkLabel(retry)
	emitter.ctx.EmitStoreRegMem(emitter.cursor, emitter.ctx.StackReg, candidate)
	emitter.emitSequence(rest, successLabel, backtrack)
	emitter.ctx.MarkLabel(backtrack)
	emitter.ctx.EmitMovRegMem(emitter.cursor, emitter.ctx.StackReg, candidate)
	tmp := emitter.ctx.AllocRegExcept(emitter.cursor, emitter.end, emitter.scan)
	emitter.ctx.EmitMovRegMem(tmp, emitter.ctx.StackReg, minimumCursor)
	emitter.ctx.EmitCmpInt64(emitter.cursor, tmp)
	emitter.ctx.FreeReg(tmp)
	emitter.ctx.EmitJump(CondUnsignedBelowOrEqual, failLabel)
	emitter.ctx.EmitSubRegImm32(emitter.cursor, int32(width))
	emitter.ctx.EmitJmp(retry)
}

func (emitter *jitRegexEmitter) emitComplexTailRepeat(node *syntax.Regexp, min int, rest []jitRegexTerm, successLabel, failLabel JITLabel) {
	for range min {
		next := emitter.ctx.ReserveLabel()
		emitter.emitSequence(jitRegexFlatten(node, len(emitter.captures) != 0), next, failLabel)
		emitter.ctx.MarkLabel(next)
	}
	loop := emitter.ctx.ReserveLabel()
	matched := emitter.ctx.ReserveLabel()
	done := emitter.ctx.ReserveLabel()
	trial := emitter.ctx.AllocStack(8)
	emitter.ctx.MarkLabel(loop)
	emitter.ctx.EmitStoreRegMem(emitter.cursor, emitter.ctx.StackReg, trial)
	emitter.emitSequence(jitRegexFlatten(node, len(emitter.captures) != 0), matched, done)
	emitter.ctx.MarkLabel(matched)
	tmp := emitter.ctx.AllocRegExcept(emitter.cursor, emitter.end, emitter.scan)
	emitter.ctx.EmitMovRegMem(tmp, emitter.ctx.StackReg, trial)
	emitter.ctx.EmitCmpInt64(emitter.cursor, tmp)
	emitter.ctx.FreeReg(tmp)
	emitter.ctx.EmitJump(CondEqual, done)
	emitter.ctx.EmitJmp(loop)
	emitter.ctx.MarkLabel(done)
	emitter.ctx.EmitMovRegMem(emitter.cursor, emitter.ctx.StackReg, trial)
	emitter.emitSequence(rest, successLabel, failLabel)
}

func (emitter *jitRegexEmitter) emitAtom(node *syntax.Regexp, failLabel JITLabel) {
	switch node.Op {
	case syntax.OpNoMatch:
		emitter.ctx.EmitJmp(failLabel)
	case syntax.OpEmptyMatch:
		return
	case syntax.OpLiteral:
		emitter.emitLiteral(node, failLabel)
	case syntax.OpCharClass, syntax.OpAnyCharNotNL, syntax.OpAnyChar:
		emitter.emitByteClass(node, failLabel)
	case syntax.OpBeginText:
		tmp := emitter.ctx.AllocRegExcept(emitter.cursor, emitter.end, emitter.scan)
		emitter.ctx.EmitMovRegMem(tmp, emitter.ctx.StackReg, emitter.inputStartOff)
		emitter.ctx.EmitCmpInt64(emitter.cursor, tmp)
		emitter.ctx.FreeReg(tmp)
		emitter.ctx.EmitJump(CondNotEqual, failLabel)
	case syntax.OpEndText:
		emitter.ctx.EmitCmpInt64(emitter.cursor, emitter.end)
		emitter.ctx.EmitJump(CondNotEqual, failLabel)
	case syntax.OpBeginLine:
		emitter.emitBeginLine(failLabel)
	case syntax.OpEndLine:
		emitter.emitEndLine(failLabel)
	case syntax.OpWordBoundary:
		emitter.emitWordBoundary(failLabel, true)
	case syntax.OpNoWordBoundary:
		emitter.emitWordBoundary(failLabel, false)
	default:
		panic(fmt.Sprintf("jit: unsupported regex operation %s in %q", node.Op, emitter.program.pattern))
	}
}

func (emitter *jitRegexEmitter) emitBounds(width int, failLabel JITLabel) {
	remaining := emitter.ctx.AllocRegExcept(emitter.cursor, emitter.end, emitter.scan)
	emitter.ctx.EmitMovRegReg(remaining, emitter.end)
	emitter.ctx.EmitSubInt64(remaining, emitter.cursor)
	emitter.ctx.EmitCmpRegImm32(remaining, int32(width))
	emitter.ctx.FreeReg(remaining)
	emitter.ctx.EmitJump(CondUnsignedBelow, failLabel)
}

func (emitter *jitRegexEmitter) emitLiteral(node *syntax.Regexp, failLabel JITLabel) {
	literal := []byte(string(node.Rune))
	if len(literal) == 0 {
		return
	}
	mask := make([]byte, len(literal))
	for index := range mask {
		mask[index] = 0xff
	}
	if node.Flags&syntax.FoldCase != 0 {
		for index, value := range literal {
			if value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z' {
				mask[index] = 0xdf
				literal[index] &= 0xdf
				continue
			}
			if value >= 0x80 {
				panic(fmt.Sprintf("jit: Unicode case folding is not yet native for %q", emitter.program.pattern))
			}
		}
	}
	emitter.emitBounds(len(literal), failLabel)
	for offset := 0; offset < len(literal); {
		width := 1
		remaining := len(literal) - offset
		switch {
		case remaining >= 8:
			width = 8
		case remaining >= 4:
			width = 4
		case remaining >= 2:
			width = 2
		}
		emitter.ctx.EmitMaskedLiteralCheck(emitter.cursor, int32(offset), literal[offset:offset+width], mask[offset:offset+width], failLabel)
		offset += width
	}
	emitter.ctx.EmitAddRegImm32(emitter.cursor, int32(len(literal)))
}

func jitRegexByteRanges(node *syntax.Regexp) [][2]byte {
	accepted := [256]bool{}
	switch node.Op {
	case syntax.OpAnyChar:
		for index := range accepted {
			accepted[index] = true
		}
	case syntax.OpAnyCharNotNL:
		for index := range accepted {
			accepted[index] = byte(index) != '\n'
		}
	case syntax.OpCharClass:
		for value := 0; value < 256; value++ {
			r := rune(value)
			for index := 0; index+1 < len(node.Rune); index += 2 {
				if r >= node.Rune[index] && r <= node.Rune[index+1] {
					accepted[value] = true
					break
				}
			}
		}
	default:
		panic("jit: byte ranges require a consuming character operation")
	}
	var ranges [][2]byte
	for start := 0; start < len(accepted); {
		if !accepted[start] {
			start++
			continue
		}
		end := start
		for end+1 < len(accepted) && accepted[end+1] {
			end++
		}
		ranges = append(ranges, [2]byte{byte(start), byte(end)})
		start = end + 1
	}
	return ranges
}

func (emitter *jitRegexEmitter) emitByteClass(node *syntax.Regexp, failLabel JITLabel) {
	emitter.emitBounds(1, failLabel)
	char := emitter.ctx.AllocRegExcept(emitter.cursor, emitter.end, emitter.scan)
	emitter.ctx.EmitMovRegMemB(char, emitter.cursor, 0)
	ranges := jitRegexByteRanges(node)
	if len(ranges) == 0 {
		emitter.ctx.FreeReg(char)
		emitter.ctx.EmitJmp(failLabel)
		return
	}
	matched := emitter.ctx.AllocRegExcept(emitter.cursor, emitter.end, emitter.scan, char)
	emitter.ctx.EmitMovRegImm64(matched, 0)
	for _, interval := range ranges {
		candidate := emitter.ctx.AllocRegExcept(emitter.cursor, emitter.end, emitter.scan, char, matched)
		emitter.ctx.EmitMovRegReg(candidate, char)
		if interval[0] != 0 {
			emitter.ctx.EmitSubRegImm32(candidate, int32(interval[0]))
		}
		emitter.ctx.EmitCmpRegImm32(candidate, int32(uint16(interval[1])-uint16(interval[0])))
		emitter.ctx.EmitSetcc(candidate, CondUnsignedBelowOrEqual)
		emitter.ctx.EmitOrInt64(matched, candidate)
		emitter.ctx.FreeReg(candidate)
	}
	emitter.ctx.FreeReg(char)
	emitter.ctx.EmitCmpRegImm32(matched, 0)
	emitter.ctx.FreeReg(matched)
	emitter.ctx.EmitJump(CondEqual, failLabel)
	emitter.ctx.EmitAddRegImm32(emitter.cursor, 1)
}

func (emitter *jitRegexEmitter) emitBeginLine(failLabel JITLabel) {
	base := emitter.ctx.AllocRegExcept(emitter.cursor, emitter.end, emitter.scan)
	emitter.ctx.EmitMovRegMem(base, emitter.ctx.StackReg, emitter.inputStartOff)
	done := emitter.ctx.ReserveLabel()
	emitter.ctx.EmitCmpInt64(emitter.cursor, base)
	emitter.ctx.EmitJump(CondEqual, done)
	emitter.ctx.EmitMovRegMemB(base, emitter.cursor, -1)
	emitter.ctx.EmitCmpRegImm32(base, '\n')
	emitter.ctx.EmitJump(CondEqual, done)
	emitter.ctx.EmitJmp(failLabel)
	emitter.ctx.MarkLabel(done)
	emitter.ctx.FreeReg(base)
}

func (emitter *jitRegexEmitter) emitEndLine(failLabel JITLabel) {
	done := emitter.ctx.ReserveLabel()
	emitter.ctx.EmitCmpInt64(emitter.cursor, emitter.end)
	emitter.ctx.EmitJump(CondEqual, done)
	tmp := emitter.ctx.AllocRegExcept(emitter.cursor, emitter.end, emitter.scan)
	emitter.ctx.EmitMovRegMemB(tmp, emitter.cursor, 0)
	emitter.ctx.EmitCmpRegImm32(tmp, '\n')
	emitter.ctx.FreeReg(tmp)
	emitter.ctx.EmitJump(CondEqual, done)
	emitter.ctx.EmitJmp(failLabel)
	emitter.ctx.MarkLabel(done)
}

func (emitter *jitRegexEmitter) emitWordClass(pointer Reg, offset int32) Reg {
	char := emitter.ctx.AllocRegExcept(emitter.cursor, emitter.end, emitter.scan, pointer)
	emitter.ctx.EmitMovRegMemB(char, pointer, offset)
	combined := emitter.ctx.AllocRegExcept(emitter.cursor, emitter.end, emitter.scan, pointer, char)
	emitter.ctx.EmitMovRegImm64(combined, 0)
	for _, interval := range [][2]byte{{'0', '9'}, {'A', 'Z'}, {'_', '_'}, {'a', 'z'}} {
		candidate := emitter.ctx.AllocRegExcept(emitter.cursor, emitter.end, emitter.scan, pointer, char, combined)
		emitter.ctx.EmitMovRegReg(candidate, char)
		emitter.ctx.EmitSubRegImm32(candidate, int32(interval[0]))
		emitter.ctx.EmitCmpRegImm32(candidate, int32(interval[1]-interval[0]))
		emitter.ctx.EmitSetcc(candidate, CondUnsignedBelowOrEqual)
		emitter.ctx.EmitOrInt64(combined, candidate)
		emitter.ctx.FreeReg(candidate)
	}
	emitter.ctx.FreeReg(char)
	return combined
}

func (emitter *jitRegexEmitter) emitWordBoundary(failLabel JITLabel, expected bool) {
	prev := emitter.ctx.AllocRegExcept(emitter.cursor, emitter.end, emitter.scan)
	emitter.ctx.EmitMovRegImm64(prev, 0)
	prevMissing := emitter.ctx.ReserveLabel()
	prevReady := emitter.ctx.ReserveLabel()
	base := emitter.ctx.AllocRegExcept(emitter.cursor, emitter.end, emitter.scan, prev)
	emitter.ctx.EmitMovRegMem(base, emitter.ctx.StackReg, emitter.inputStartOff)
	emitter.ctx.EmitCmpInt64(emitter.cursor, base)
	emitter.ctx.FreeReg(base)
	emitter.ctx.EmitJump(CondEqual, prevMissing)
	prevWord := emitter.emitWordClass(emitter.cursor, -1)
	emitter.ctx.EmitMovRegReg(prev, prevWord)
	emitter.ctx.FreeReg(prevWord)
	emitter.ctx.EmitJmp(prevReady)
	emitter.ctx.MarkLabel(prevMissing)
	emitter.ctx.MarkLabel(prevReady)

	next := emitter.ctx.AllocRegExcept(emitter.cursor, emitter.end, emitter.scan, prev)
	emitter.ctx.EmitMovRegImm64(next, 0)
	nextMissing := emitter.ctx.ReserveLabel()
	nextReady := emitter.ctx.ReserveLabel()
	emitter.ctx.EmitCmpInt64(emitter.cursor, emitter.end)
	emitter.ctx.EmitJump(CondEqual, nextMissing)
	nextWord := emitter.emitWordClass(emitter.cursor, 0)
	emitter.ctx.EmitMovRegReg(next, nextWord)
	emitter.ctx.FreeReg(nextWord)
	emitter.ctx.EmitJmp(nextReady)
	emitter.ctx.MarkLabel(nextMissing)
	emitter.ctx.MarkLabel(nextReady)
	emitter.ctx.EmitXorInt64(prev, next)
	emitter.ctx.FreeReg(next)
	emitter.ctx.EmitCmpRegImm32(prev, 0)
	emitter.ctx.FreeReg(prev)
	if expected {
		emitter.ctx.EmitJump(CondEqual, failLabel)
	} else {
		emitter.ctx.EmitJump(CondNotEqual, failLabel)
	}
}

func jitRegexCaptureTargets(ctx *JITContext, count int) []JITValueDesc {
	targets := make([]JITValueDesc, count)
	for index := range targets {
		off := ctx.AllocStack(16)
		targets[index] = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: off, Rooted: true}
	}
	return targets
}

func jitEmitNativeRegex(ctx *JITContext, program *jitRegexProgram, value JITValueDesc, captures []JITValueDesc, successLabel, failLabel, invalidLabel JITLabel, nilLabel *JITLabel, stringify bool) {
	emitter := &jitRegexEmitter{ctx: ctx, program: program, captures: captures}
	emitter.reserveMachineState(value, invalidLabel, nilLabel, stringify)
	emitter.emitProgram(successLabel, failLabel)
	emitter.releaseMachineState()
}

func jitEmitConstantRegexpTest(ctx *JITContext, pattern *regexp.Regexp, value JITValueDesc, result JITValueDesc) JITValueDesc {
	if value.Loc == LocImm {
		out := jitConstantRegexpTest(NewRegex(pattern), value.Imm)
		return jitPlaceScmerIntoTarget(ctx, JITValueDesc{Loc: LocImm, Type: out.GetTag(), Imm: out}, result)
	}
	program := jitCompileRegexProgram(pattern)
	target := jitEnsureResultPair(ctx, result)
	success := ctx.ReserveLabel()
	fail := ctx.ReserveLabel()
	nilResult := ctx.ReserveLabel()
	done := ctx.ReserveLabel()
	jitEmitNativeRegex(ctx, program, value, nil, success, fail, fail, &nilResult, true)
	ctx.MarkLabel(success)
	ctx.EmitMakeBool(target, JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)})
	ctx.EmitJmp(done)
	ctx.MarkLabel(fail)
	ctx.EmitMakeBool(target, JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)})
	ctx.EmitJmp(done)
	ctx.MarkLabel(nilResult)
	ctx.EmitMakeNil(target)
	ctx.MarkLabel(done)
	target.Type = JITTypeUnknown
	return target
}

func jitEmitConstantRegexpPredicate(ctx *JITContext, pattern *regexp.Regexp, value JITValueDesc) JITValueDesc {
	if value.Loc == LocImm {
		text, ok := scmerAsString(value.Imm)
		if !ok {
			panic("regex expects string")
		}
		return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(pattern.MatchString(text)), NoHeapPointer: true}
	}
	program := jitCompileRegexProgram(pattern)
	result := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: ctx.AllocReg(), NoHeapPointer: true}
	ctx.BindReg(result.Reg, &result)
	success := ctx.ReserveLabel()
	fail := ctx.ReserveLabel()
	invalid := ctx.ReserveLabel()
	done := ctx.ReserveLabel()
	jitEmitNativeRegex(ctx, program, value, nil, success, fail, invalid, nil, false)
	ctx.MarkLabel(success)
	ctx.EmitMovRegImm64(result.Reg, 1)
	ctx.EmitJmp(done)
	ctx.MarkLabel(fail)
	ctx.EmitMovRegImm64(result.Reg, 0)
	ctx.EmitJmp(done)
	ctx.MarkLabel(invalid)
	ctx.EmitGoPanic("regex expects string")
	ctx.EmitMovRegImm64(result.Reg, 0)
	ctx.MarkLabel(done)
	return result
}

// jitEmitConstantRegexpCaptures emits captures directly into their final
// JITValueDesc slots. No index slice or intermediate capture representation is
// created at runtime.
func jitEmitConstantRegexpCaptures(ctx *JITContext, pattern *regexp.Regexp, value JITValueDesc, failLabel JITLabel) []JITValueDesc {
	program := jitCompileRegexProgram(pattern)
	captures := jitRegexCaptureTargets(ctx, program.captures)
	// A capture match can sit below an if/match result and an inlined callback,
	// leaving too few registers for the byte walker's cursor, end and scan
	// state. The input and captures are stable stack values, so preserve the
	// outer register bank only under pressure and restore it on every outgoing
	// edge of this self-contained control-flow region.
	value = ctx.stabilizeForNested(value)
	if bits.OnesCount64(ctx.FreeRegs&ctx.AllRegs&^ctx.ProtectedRegs) >= 6 {
		success := ctx.ReserveLabel()
		invalid := ctx.ReserveLabel()
		jitEmitNativeRegex(ctx, program, value, captures, success, failLabel, invalid, nil, false)
		ctx.MarkLabel(invalid)
		ctx.EmitGoPanic("regex expects string")
		ctx.EmitJmp(failLabel)
		ctx.MarkLabel(success)
		return captures
	}
	outer := ctx.PreserveOuterRegs()
	success := ctx.ReserveLabel()
	failed := ctx.ReserveLabel()
	invalid := ctx.ReserveLabel()
	jitEmitNativeRegex(ctx, program, value, captures, success, failed, invalid, nil, false)
	ctx.MarkLabel(invalid)
	ctx.RestoreOuterRegs(outer)
	ctx.EmitGoPanic("regex expects string")
	ctx.EmitJmp(failLabel)
	ctx.MarkLabel(failed)
	ctx.RestoreOuterRegs(outer)
	ctx.EmitJmp(failLabel)
	ctx.MarkLabel(success)
	ctx.RestoreOuterRegs(outer)
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
			text, ok := scmerAsString(arguments[1])
			if !ok {
				panic("regex expects string")
			}
			return NewBool(arguments[0].Regex().MatchString(text))
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
