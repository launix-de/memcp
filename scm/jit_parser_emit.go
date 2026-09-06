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

func jitStaticParserForExpr(ctx *JITContext, expr Scmer) (*ScmParser, bool) {
	for expr.IsSourceInfo() {
		expr = expr.SourceInfo().value
	}
	if expr.IsParser() && expr.Parser() != nil {
		return expr.Parser(), true
	}
	var desc JITValueDesc
	var found bool
	if expr.IsNthLocalVar() && ctx.Env != nil {
		index := int(expr.NthLocalVar())
		if index >= 0 && index < len(ctx.Env.Numbered) {
			desc, found = ctx.Env.Numbered[index], true
		}
	} else if expr.IsSymbol() && ctx.Env != nil {
		desc, found = ctx.Env.Lookup(expr.Symbol())
	}
	if found && desc.Loc == LocImm && desc.Imm.IsParser() && desc.Imm.Parser() != nil {
		return desc.Imm.Parser(), true
	}
	return nil, false
}

func jitParserTemplateForExpr(ctx *JITContext, expr Scmer) (*JITParserTemplate, bool) {
	for expr.IsSourceInfo() {
		expr = expr.SourceInfo().value
	}
	if expr.IsNthLocalVar() && ctx.Env != nil {
		index := int(expr.NthLocalVar())
		if index >= 0 && index < len(ctx.Env.Numbered) {
			desc := ctx.Env.Numbered[index]
			return desc.Parser, desc.Loc == LocParserTemplate && desc.Parser != nil
		}
	}
	items, ok := scmerSlice(expr)
	if !ok || len(items) == 0 || !scmerIsSymbol(items[0], "parser") {
		return nil, false
	}
	generator, whitespace := NewNil(), NewNil()
	if len(items) > 2 {
		generator = items[2]
	}
	if len(items) > 3 {
		whitespace = items[3]
	}
	var runtimeOuter *Env
	if ctx.RuntimeEnv.GetTag() == tagAny {
		runtimeOuter, _ = ctx.RuntimeEnv.Any().(*Env)
	}
	return &JITParserTemplate{
		Syntax: items[1], Generator: generator, Whitespace: whitespace,
		IgnoreResult: len(items) > 2, Outer: ctx.Env, RuntimeOuter: runtimeOuter,
	}, true
}

func jitEmitStaticParser(ctx *JITContext, parser *ScmParser, input JITValueDesc, result JITValueDesc) JITValueDesc {
	program, entryRule := parser.JITProgram, parser.JITRule
	if program == nil {
		program = jitBuildParserProgram(parser)
		entryRule = program.parserRule[parser]
	}
	programValue := NewAny(program)
	ctx.TrackImm(programValue)
	input = jitRootScmer(ctx, input)
	resultOff := ctx.AllocSpill(16)
	ctx.setStackPointer(jitStackRootFrameBP, resultOff, true)
	parserResult := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: resultOff, Rooted: true}
	programPair := jitCopyScmerToPair(ctx, JITValueDesc{Loc: LocImm, Type: tagAny, Imm: programValue})
	state := JITEmitGoCallResults(ctx, GoFuncAddr(jitParserAcquireStateNative), []JITValueDesc{programPair, input}, []uint8{2}, []uint8{1})[0]
	state.Type = tagAny
	state.Rooted = true
	ctx.FreeDesc(&programPair)
	outerRegs := ctx.PreserveOuterRegs()
	entry := jitCopyScmerToPair(ctx, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(entryRule)), NoHeapPointer: true})
	jitEmitParserProgramCore(ctx, program, input, state, entry, parserResult, true, entryRule)
	ctx.RestoreOuterRegs(outerRegs)
	programPair = jitCopyScmerToPair(ctx, JITValueDesc{Loc: LocImm, Type: tagAny, Imm: programValue})
	ctx.EmitGoCallVoid(GoFuncAddr(jitParserReleaseStateNative), []JITValueDesc{programPair, state})
	ctx.FreeDesc(&programPair)
	return jitPlaceScmerIntoTarget(ctx, parserResult, result)
}

func jitEmitParserTemplate(ctx *JITContext, template *JITParserTemplate, input JITValueDesc, result JITValueDesc) JITValueDesc {
	previousStackPhiTargets := ctx.StackPhiTargets
	ctx.StackPhiTargets = true
	defer func() { ctx.StackPhiTargets = previousStackPhiTargets }()
	program, entryRule := jitBuildParserTemplateProgram(template)
	programValue := NewAny(program)
	ctx.TrackImm(programValue)
	input = jitRootScmer(ctx, input)
	resultOff := ctx.AllocSpill(16)
	ctx.setStackPointer(jitStackRootFrameBP, resultOff, true)
	parserResult := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: resultOff, Rooted: true}
	programPair := jitCopyScmerToPair(ctx, JITValueDesc{Loc: LocImm, Type: tagAny, Imm: programValue})
	state := ctx.EmitGoCallScalar(GoFuncAddr(jitParserAcquireStateNative), []JITValueDesc{programPair, input}, 2)
	state.Type = tagAny
	ctx.FreeDesc(&programPair)
	state = jitRootScmer(ctx, state)
	outerRegs := ctx.PreserveOuterRegs()
	entry := jitCopyScmerToPair(ctx, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(entryRule)), NoHeapPointer: true})
	jitEmitParserProgramCore(ctx, program, input, state, entry, parserResult, true, entryRule)
	ctx.RestoreOuterRegs(outerRegs)
	programPair = jitCopyScmerToPair(ctx, JITValueDesc{Loc: LocImm, Type: tagAny, Imm: programValue})
	ctx.EmitGoCallVoid(GoFuncAddr(jitParserReleaseStateNative), []JITValueDesc{programPair, state})
	ctx.FreeDesc(&programPair)
	ctx.FreeDesc(&state)
	return jitPlaceScmerIntoTarget(ctx, parserResult, result)
}

type jitParserEmitter struct {
	ctx               *JITContext
	program           *jitParserProgram
	input             JITValueDesc
	state             JITValueDesc
	entry             JITValueDesc
	positionOff       int32
	continuationOff   int32
	generatorValueOff int32
	ruleLabels        []JITLabel
	continuations     []JITLabel
	dispatchLabel     JITLabel
	skipLabel         JITLabel
	inlineActions     bool
	skipperRule       int
	// memoCheck{Rule,Accepted,Rejected}Off and memoCheckLabel implement
	// emitMemoCheckedRuleRef as one shared, outlined block instead of
	// inlining it at every rule-reference call site: a call site is a
	// three-word store (rule id, accepted/rejected continuation ids) plus
	// a jmp, and the one shared block does the actual native memo-table
	// read, keeping the per-site cost small regardless of how many
	// thousands of rule references the grammar contains.
	memoCheckRuleOff     int32
	memoCheckAcceptedOff int32
	memoCheckRejectedOff int32
	memoCheckLabel       JITLabel
	// failLabel is jitEmitParserProgramCore's overall parse-failure label,
	// needed by emitMemoCheckBlock's defensive (never actually reachable -
	// node.rule always names a real rule) invalid-jump-table-index arms.
	failLabel JITLabel
	// noMemoDepth counts enclosing repeat nodes whose grammar author marked
	// noMemo (the (* sub sep #t) form). Left-to-right, unbranched repeat
	// bodies revisit no position twice, so rule references emitted while this
	// is >0 skip the memo entry-check via emitLexicalRuleRef: jitParserPushRuleFrame
	// still tags the frame by the target rule's own memoize bit and
	// jitParserCompleteRule still writes its memo entry unconditionally on
	// that bit, so callers elsewhere that do check the memo keep seeing
	// correct, up to date results - only the redundant pre-check here is
	// skipped, matching go-packrat's KleeneParser.NoMemo contract.
	noMemoDepth int
}

func (emitter *jitParserEmitter) statePointer() JITValueDesc {
	state := emitter.state
	emitter.ctx.EnsureDesc(&state)
	if state.Loc != LocRegPair {
		panic("jit: parser state has no pointer representation")
	}
	stateReg := emitter.ctx.AllocRegExcept(state.Reg, state.Reg2)
	emitter.ctx.EmitMovRegMem(stateReg, state.Reg, 8)
	emitter.ctx.FreeDesc(&state)
	result := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: stateReg, RelocatablePointer: true}
	emitter.ctx.BindReg(result.Reg, &result)
	return result
}

func jitParserScalar(value int64) JITValueDesc {
	return JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(value), NoHeapPointer: true}
}

func jitParserBoolScalar(value bool) JITValueDesc {
	return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(value), NoHeapPointer: true}
}

func (emitter *jitParserEmitter) loadPosition() JITValueDesc {
	position := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: emitter.ctx.AllocReg(), NoHeapPointer: true}
	emitter.ctx.BindReg(position.Reg, &position)
	emitter.ctx.EmitMovRegMem(position.Reg, emitter.ctx.StackReg, emitter.positionOff)
	return position
}

func (emitter *jitParserEmitter) storePosition(position JITValueDesc) {
	if position.Loc != LocReg {
		panic("jit: parser position must be an unboxed register")
	}
	emitter.ctx.EmitStoreRegMem(position.Reg, emitter.ctx.StackReg, emitter.positionOff)
	emitter.ctx.FreeDesc(&position)
}

func (emitter *jitParserEmitter) immPair(value Scmer) JITValueDesc {
	desc := JITValueDesc{Loc: LocImm, Type: value.GetTag(), Imm: value}
	emitter.ctx.TrackImm(value)
	pair := jitAllocTrackedPair(emitter.ctx, value.GetTag())
	return jitPlaceIntoPair(emitter.ctx, &desc, pair)
}

func (emitter *jitParserEmitter) emitVoid(fn any, args ...JITValueDesc) {
	emitter.ctx.EmitGoCallVoid(GoFuncAddr(fn), args)
}

func (emitter *jitParserEmitter) pushValue(value JITValueDesc) {
	original := value
	if value.Loc != LocRegPair && value.Loc != LocStackPair && value.Loc != LocInputPair {
		value = emitter.immPair(value.Imm)
	}
	emitter.emitVoid(jitParserPushValueNative, emitter.state, value)
	if value.Loc == LocRegPair {
		emitter.ctx.FreeDesc(&value)
	}
	if original.Loc == LocRegPair && original.ID != value.ID {
		emitter.ctx.FreeDesc(&original)
	}
}

func (emitter *jitParserEmitter) discardValue() {
	emitter.emitVoid(jitParserDiscardValueNative, emitter.state)
}

func (emitter *jitParserEmitter) pushCheckpoint() {
	position := emitter.loadPosition()
	emitter.emitVoid(jitParserPushCheckpointNative, emitter.state, position)
	emitter.ctx.FreeDesc(&position)
}

func (emitter *jitParserEmitter) restoreCheckpoint() {
	position := emitter.ctx.EmitGoCallScalar(GoFuncAddr(jitParserRestoreCheckpointNative), []JITValueDesc{emitter.state}, 1)
	position.Type = tagInt
	emitter.storePosition(position)
}

func (emitter *jitParserEmitter) commitCheckpoint() {
	emitter.emitVoid(jitParserCommitCheckpointNative, emitter.state)
}

func (emitter *jitParserEmitter) substringAtPosition() JITValueDesc {
	text := emitter.input
	emitter.ctx.EnsureDesc(&text)
	if text.Loc != LocRegPair {
		panic("jit: parser input is not a Scmer pair")
	}
	position := emitter.loadPosition()
	emitter.ctx.EmitAddInt64(text.Reg, position.Reg)
	emitter.ctx.EmitShrRegImm8(text.Reg2, 8)
	emitter.ctx.EmitSubInt64(text.Reg2, position.Reg)
	emitter.ctx.EmitShlRegImm8(text.Reg2, 8)
	emitter.ctx.EmitMovRegImm64(emitter.ctx.ScratchReg, uint64(tagString))
	emitter.ctx.EmitOrInt64(text.Reg2, emitter.ctx.ScratchReg)
	emitter.ctx.FreeDesc(&position)
	text.Type = tagString
	return text
}

func (emitter *jitParserEmitter) advanceBy(match JITValueDesc) {
	if match.Loc != LocStackPair && match.Loc != LocRegPair {
		panic("jit: parser regex match is not a Scmer pair")
	}
	length := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: emitter.ctx.AllocReg(), NoHeapPointer: true}
	emitter.ctx.BindReg(length.Reg, &length)
	if match.Loc == LocStackPair {
		emitter.ctx.EmitMovRegMem(length.Reg, emitter.ctx.StackReg, match.StackOff+8)
	} else {
		emitter.ctx.EmitMovRegReg(length.Reg, match.Reg2)
	}
	emitter.ctx.EmitShrRegImm8(length.Reg, 8)
	position := emitter.loadPosition()
	emitter.ctx.EmitAddInt64(position.Reg, length.Reg)
	emitter.ctx.FreeDesc(&length)
	emitter.storePosition(position)
}

func (emitter *jitParserEmitter) atBreak(fail JITLabel) {
	position := emitter.loadPosition()
	accepted := emitter.ctx.EmitGoCallScalar(GoFuncAddr(jitParserAtBreakNative), []JITValueDesc{emitter.input, position}, 1)
	emitter.ctx.FreeDesc(&position)
	emitter.ctx.EmitCmpRegImm32(accepted.Reg, 0)
	emitter.ctx.FreeDesc(&accepted)
	emitter.ctx.EmitJump(CondEqual, fail)
}

func (emitter *jitParserEmitter) emitSkip(_ int, done JITLabel) {
	emitter.ctx.EmitMovRegImm64(emitter.ctx.ScratchReg, uint64(emitter.continuation(done)))
	emitter.ctx.EmitStoreRegMem(emitter.ctx.ScratchReg, emitter.ctx.StackReg, emitter.continuationOff)
	emitter.ctx.EmitJmp(emitter.skipLabel)
}

func (emitter *jitParserEmitter) emitSkipBody(done JITLabel) {
	program := emitter.program.rules[emitter.skipperRule].skipper
	if program == nil {
		emitter.ctx.EmitJmp(done)
		return
	}
	matched, failed := emitter.ctx.ReserveLabel(), emitter.ctx.ReserveLabel()
	captures := jitRegexCaptureTargets(emitter.ctx, 1)
	input := emitter.substringAtPosition()
	jitEmitNativeRegex(emitter.ctx, program, input, captures, matched, failed, failed, nil, false)
	emitter.ctx.MarkLabel(matched)
	emitter.advanceBy(captures[0])
	emitter.ctx.EmitJmp(done)
	emitter.ctx.MarkLabel(failed)
	emitter.ctx.EmitJmp(done)
	emitter.ctx.FreeStack(16)
}

func (emitter *jitParserEmitter) emitTerminal(node *jitParserNode, rule int, success, failure JITLabel) {
	matchStart := emitter.ctx.ReserveLabel()
	if node.skipWS {
		skipped := emitter.ctx.ReserveLabel()
		emitter.emitSkip(rule, skipped)
		emitter.ctx.MarkLabel(skipped)
		emitter.atBreak(failure)
	}
	emitter.ctx.MarkLabel(matchStart)
	matched := emitter.ctx.ReserveLabel()
	failed := emitter.ctx.ReserveLabel()
	captures := jitRegexCaptureTargets(emitter.ctx, node.regex.captures)
	input := emitter.substringAtPosition()
	jitEmitNativeRegex(emitter.ctx, node.regex, input, captures, matched, failed, failed, nil, false)
	emitter.ctx.MarkLabel(matched)
	emitter.advanceBy(captures[0])
	if node.skipWS {
		emitter.atBreak(failure)
	}
	if !node.ignoreResult {
		if node.kind == jitParserAtom {
			emitter.pushValue(JITValueDesc{Loc: LocImm, Type: node.value.GetTag(), Imm: node.value})
		} else {
			emitter.pushValue(captures[0])
		}
	}
	emitter.ctx.EmitJmp(success)
	emitter.ctx.MarkLabel(failed)
	expected := emitter.immPair(NewString(node.description))
	position := emitter.loadPosition()
	emitter.emitVoid(jitParserRecordFailureNative, emitter.state, position, expected)
	emitter.ctx.FreeDesc(&position)
	emitter.ctx.FreeDesc(&expected)
	emitter.ctx.EmitJmp(failure)
	emitter.ctx.FreeStack(int32(len(captures) * 16))
}

func (emitter *jitParserEmitter) emitSequence(node *jitParserNode, rule int, success, failure JITLabel) {
	emitter.pushCheckpoint()
	if !node.ignoreResult {
		emitter.emitVoid(jitParserPushMarkNative, emitter.state)
	}
	failed := emitter.ctx.ReserveLabel()
	for _, child := range node.children {
		next := emitter.ctx.ReserveLabel()
		emitter.emitNode(child, rule, next, failed)
		emitter.ctx.MarkLabel(next)
	}
	if !node.ignoreResult {
		emitter.emitVoid(jitParserMergeMarkNative, emitter.state, jitParserBoolScalar(false))
	}
	emitter.commitCheckpoint()
	emitter.ctx.EmitJmp(success)
	emitter.ctx.MarkLabel(failed)
	emitter.restoreCheckpoint()
	emitter.ctx.EmitJmp(failure)
}

func (emitter *jitParserEmitter) emitChoice(node *jitParserNode, rule int, success, failure JITLabel) {
	if d := emitter.program.choiceDispatchPlan(node); d.useful {
		emitter.emitChoiceDispatch(node, rule, d, success, failure)
		return
	}
	all := make([]int, len(node.children))
	for i := range all {
		all[i] = i
	}
	emitter.emitAltCascade(node, rule, all, success, failure)
}

// emitAltCascade tries the listed alternatives of node in order, each guarded by
// its own checkpoint; on the last failure it jumps to failure. This is the
// classic choice lowering, and the tail of every dispatch bucket.
func (emitter *jitParserEmitter) emitAltCascade(node *jitParserNode, rule int, indices []int, success, failure JITLabel) {
	for _, i := range indices {
		emitter.pushCheckpoint()
		accepted, rejected := emitter.ctx.ReserveLabel(), emitter.ctx.ReserveLabel()
		emitter.emitNode(node.children[i], rule, accepted, rejected)
		emitter.ctx.MarkLabel(accepted)
		emitter.commitCheckpoint()
		emitter.ctx.EmitJmp(success)
		emitter.ctx.MarkLabel(rejected)
		emitter.restoreCheckpoint()
	}
	emitter.ctx.EmitJmp(failure)
}

func altListKey(list []int) string {
	b := make([]byte, 0, len(list)*2)
	for _, x := range list {
		b = append(b, byte(x), byte(x>>8))
	}
	return string(b)
}

// jitParserPeekByteNative returns the byte at position, or -1 at end of input.
func jitParserPeekByteNative(input Scmer, position int64) int64 {
	text := input.String()
	if position < 0 || position >= int64(len(text)) {
		return -1
	}
	return int64(text[position])
}

// emitPeekByte returns a register with the byte at the current parse position
// (0..255), or -1 at end of input. No position advance. A Go call keeps this
// off the register-discipline critical path - it runs once per dispatched
// choice, replacing dozens of per-alternative helper calls.
func (emitter *jitParserEmitter) emitPeekByte() JITValueDesc {
	position := emitter.loadPosition()
	peek := emitter.ctx.EmitGoCallScalar(GoFuncAddr(jitParserPeekByteNative), []JITValueDesc{emitter.input, position}, 1)
	peek.Type = tagInt
	emitter.ctx.FreeDesc(&position)
	return peek
}

// emitChoiceDispatch reads the leading byte once and jumps to the small cascade
// of alternatives that could match it, instead of trying all ~N in turn.
func (emitter *jitParserEmitter) emitChoiceDispatch(node *jitParserNode, rule int, d choiceDispatch, success, failure JITLabel) {
	skipped := emitter.ctx.ReserveLabel()
	emitter.emitSkip(rule, skipped)
	emitter.ctx.MarkLabel(skipped)
	peek := emitter.emitPeekByte()

	fail := emitter.ctx.ReserveLabel()
	wild := fail
	if len(d.wild) > 0 {
		wild = emitter.ctx.ReserveLabel()
	}

	labels := make([]JITLabel, 256)
	type bucket struct {
		label JITLabel
		list  []int
	}
	var order []bucket
	made := map[string]JITLabel{}
	for b := 0; b < 256; b++ {
		list := d.buckets[b]
		if len(list) == 0 {
			labels[b] = wild
			continue
		}
		key := altListKey(list)
		if l, ok := made[key]; ok {
			labels[b] = l
			continue
		}
		l := emitter.ctx.ReserveLabel()
		made[key] = l
		labels[b] = l
		order = append(order, bucket{l, list})
	}

	emitter.ctx.EmitJumpTable(peek.Reg, labels, wild)
	emitter.ctx.FreeDesc(&peek)

	for _, bk := range order {
		emitter.ctx.MarkLabel(bk.label)
		emitter.emitAltCascade(node, rule, bk.list, success, fail)
	}
	if len(d.wild) > 0 {
		emitter.ctx.MarkLabel(wild)
		emitter.emitAltCascade(node, rule, d.wild, success, fail)
	}
	emitter.ctx.MarkLabel(fail)
	emitter.ctx.EmitJmp(failure)
}

func (emitter *jitParserEmitter) emitOptional(node *jitParserNode, rule int, success JITLabel) {
	emitter.pushCheckpoint()
	accepted, rejected := emitter.ctx.ReserveLabel(), emitter.ctx.ReserveLabel()
	emitter.emitNode(node.children[0], rule, accepted, rejected)
	emitter.ctx.MarkLabel(accepted)
	emitter.commitCheckpoint()
	emitter.ctx.EmitJmp(success)
	emitter.ctx.MarkLabel(rejected)
	emitter.restoreCheckpoint()
	if !node.ignoreResult {
		emitter.pushValue(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil(), NoHeapPointer: true})
	}
	emitter.ctx.EmitJmp(success)
}

func (emitter *jitParserEmitter) emitBind(node *jitParserNode, rule int, success, failure JITLabel) {
	accepted := emitter.ctx.ReserveLabel()
	emitter.emitNode(node.children[0], rule, accepted, failure)
	emitter.ctx.MarkLabel(accepted)
	emitter.emitVoid(jitParserBindValueNative, emitter.state, jitParserScalar(int64(node.binding)))
	if node.ignoreResult {
		emitter.discardValue()
	}
	emitter.ctx.EmitJmp(success)
}

func (emitter *jitParserEmitter) emitCapture(node *jitParserNode, rule int, success, failure JITLabel) {
	emitter.pushCheckpoint()
	skipped := emitter.ctx.ReserveLabel()
	emitter.emitSkip(rule, skipped)
	emitter.ctx.MarkLabel(skipped)
	start := emitter.loadPosition()
	emitter.emitVoid(jitParserPushPositionNative, emitter.state, start)
	emitter.ctx.FreeDesc(&start)
	accepted, rejected := emitter.ctx.ReserveLabel(), emitter.ctx.ReserveLabel()
	emitter.emitNode(node.children[0], rule, accepted, rejected)
	emitter.ctx.MarkLabel(accepted)
	start = emitter.ctx.EmitGoCallScalar(GoFuncAddr(jitParserPopPositionNative), []JITValueDesc{emitter.state}, 1)
	end := emitter.loadPosition()
	emitter.emitVoid(jitParserCaptureValueNative, emitter.state, emitter.input, start, end)
	emitter.ctx.FreeDesc(&start)
	emitter.ctx.FreeDesc(&end)
	emitter.commitCheckpoint()
	if node.ignoreResult {
		emitter.discardValue()
	}
	emitter.ctx.EmitJmp(success)
	emitter.ctx.MarkLabel(rejected)
	emitter.restoreCheckpoint()
	emitter.ctx.EmitJmp(failure)
}

func (emitter *jitParserEmitter) continuation(label JITLabel) int64 {
	index := int64(len(emitter.continuations))
	emitter.continuations = append(emitter.continuations, label)
	return index
}

func (emitter *jitParserEmitter) emitRuleRef(node *jitParserNode, success, failure JITLabel) {
	// A lexical sub-rule (lexicalParent >= 0) is never memoized or left
	// recursive, and a rule referenced from inside a noMemo-marked unbranched
	// repeat body provably cannot have been visited at this position before
	// (see jitParserEmitter.noMemoDepth). Either way jitParserEnterRuleNative's
	// memo/left-recursion-head check can only ever report a miss here, so its
	// full call, 3-word return, and cache-hit branch are dead work; jump
	// straight to jitParserPushRuleFrame and the rule body instead.
	// memoRuleIndex < 0 also covers a non-lexical rule that analyzeMemoNeed
	// proved is entered at most once per position: it never has a stored entry
	// to hit and never writes one, so the check is pure overhead.
	if emitter.program.rules[node.rule].lexicalParent >= 0 || emitter.noMemoDepth > 0 ||
		emitter.program.memoRuleIndex[node.rule] < 0 {
		emitter.emitLexicalRuleRef(node, success, failure)
		return
	}
	emitter.emitMemoCheckedRuleRef(node, success, failure)
}

// sliceElemAddr computes &slice[index] for a Go slice field living
// fieldOffset bytes into the struct pointed to by base, where the slice's
// elements are 1<<elemShift bytes wide. Returns a fresh, caller-owned
// register holding the address. index is only read (via a copy), never
// mutated, so it stays valid for further use by the caller afterwards.
func (emitter *jitParserEmitter) sliceElemAddr(base Reg, fieldOffset int32, index Reg, elemShift uint8) Reg {
	addr := emitter.ctx.AllocReg()
	emitter.ctx.EmitMovRegMem(addr, base, fieldOffset) // slice.Data
	scaled := emitter.ctx.AllocRegExcept(addr, index)
	emitter.ctx.EmitMovRegReg(scaled, index)
	emitter.ctx.EmitShlRegImm8(scaled, elemShift)
	emitter.ctx.EmitAddInt64(addr, scaled)
	emitter.ctx.FreeReg(scaled)
	return addr
}

// loadScratchInt reads a persistent int64 scratch stack slot into a fresh,
// bound register - the same shape as loadPosition, just parameterized by
// offset, for the memo-check block's small set of call-site parameters.
func (emitter *jitParserEmitter) loadScratchInt(offset int32) JITValueDesc {
	value := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: emitter.ctx.AllocReg(), NoHeapPointer: true}
	emitter.ctx.BindReg(value.Reg, &value)
	emitter.ctx.EmitMovRegMem(value.Reg, emitter.ctx.StackReg, offset)
	return value
}

// storeScratchImm writes a compile-time-known int64 into a persistent
// scratch stack slot via the shared scratch register.
func (emitter *jitParserEmitter) storeScratchImm(value int64, offset int32) {
	emitter.ctx.EmitMovRegImm64(emitter.ctx.ScratchReg, uint64(value))
	emitter.ctx.EmitStoreRegMem(emitter.ctx.ScratchReg, emitter.ctx.StackReg, offset)
}

// emitMemoCheckedRuleRef is the general-case rule reference: node.rule is
// neither a lexical sub-rule nor reached from inside a noMemo repeat (see
// emitRuleRef), so a prior visit to this (rule, position) pair may really be
// memoized. The actual native memo-table read lives once, in the shared
// emitMemoCheckBlock (reachable from every such call site in the grammar -
// there can be thousands): a call site only has to hand it its three
// site-specific values (which rule, and where to resume on hit/miss) and
// jump there, instead of inlining the whole read-the-memo-table sequence
// (and, in the rare case, the jitParserEnterRuleNative fallback call) again
// at every single site. That keeps the per-site cost at a handful of
// instructions regardless of grammar size, matching how the interpreter's
// own bytecode would call a shared routine rather than duplicating it.
func (emitter *jitParserEmitter) emitMemoCheckedRuleRef(node *jitParserNode, success, failure JITLabel) {
	if emitter.program.memoRuleIndex[node.rule] < 0 {
		panic("jit: memo-checked rule ref on a non-memoizable rule")
	}
	accepted, rejected := emitter.ctx.ReserveLabel(), emitter.ctx.ReserveLabel()
	acceptedContinuation := emitter.continuation(accepted)
	rejectedContinuation := emitter.continuation(rejected)

	emitter.storeScratchImm(int64(node.rule), emitter.memoCheckRuleOff)
	emitter.storeScratchImm(acceptedContinuation, emitter.memoCheckAcceptedOff)
	emitter.storeScratchImm(rejectedContinuation, emitter.memoCheckRejectedOff)
	emitter.ctx.EmitJmp(emitter.memoCheckLabel)

	emitter.ctx.MarkLabel(accepted)
	if node.ignoreResult {
		emitter.discardValue()
	}
	emitter.ctx.EmitJmp(success)
	emitter.ctx.MarkLabel(rejected)
	emitter.ctx.EmitJmp(failure)
}

// emitMemoCheckBlock emits the one shared body every emitMemoCheckedRuleRef
// call site jumps into (via emitter.memoCheckLabel), reading its rule id and
// accepted/rejected continuation ids from the fixed scratch slots the call
// site stored them in. Reading the packrat memo table
// (jitParserState.heads/memoOffsets/memoRules/memoEntries) is pure
// scalar/pointer arithmetic over Go's stable slice-header layout, so the two
// cases that make up virtually all rule references - a definite first visit
// (miss) and an already-resolved, non-left-recursive hit - are handled
// entirely natively, with no Go call: jitParserRecordFirstVisitNative for a
// miss (jitParserPushRuleFrame's void call plus the memo-active bookkeeping,
// nothing else), and an inline read of the cached value for a hit. Since
// this block is shared, neither case can jump directly to a call site's own
// accepted/rejected label the way the earlier per-site version did; both
// route through the continuation/dispatchLabel mechanism the miss/fallback
// paths already relied on before this change, and a genuine hit resolves
// via one indirect jump through that same jump table instead of a direct
// one - a small price for turning O(sites) duplicated code into O(1).
// Neither case changes what jitParserCompleteRule eventually writes to the
// memo - that still happens unconditionally, gated only by the target
// rule's own memoize bit set at frame-push time - so this only changes how
// an existing entry is discovered, never what ends up in it. The rare cases
// - a left-recursion head already registered at this position, or a memo
// entry still "active" (mid-evaluation, i.e. real left recursion) - fall
// back verbatim to jitParserEnterRuleNative, the only place implementing
// Warth-et-al iterative left-recursion growth; that fallback, too, exists
// only once here rather than once per call site.
func (emitter *jitParserEmitter) emitMemoCheckBlock() {
	ctx := emitter.ctx
	headsOff := int32(unsafe.Offsetof(jitParserState{}.heads))
	memoOffsetsOff := int32(unsafe.Offsetof(jitParserState{}.memoOffsets))
	memoRulesOff := int32(unsafe.Offsetof(jitParserState{}.memoRules))
	memoEntriesOff := int32(unsafe.Offsetof(jitParserState{}.memoEntries))
	programOff := int32(unsafe.Offsetof(jitParserState{}.program))
	memoRuleIndexOff := int32(unsafe.Offsetof(jitParserProgram{}.memoRuleIndex))
	entryPositionOff := int32(unsafe.Offsetof(jitParserMemoEntry{}.position))
	entrySuccessOff := int32(unsafe.Offsetof(jitParserMemoEntry{}.success))
	entryActiveOff := int32(unsafe.Offsetof(jitParserMemoEntry{}.active))
	entryValueOff := int32(unsafe.Offsetof(jitParserMemoEntry{}.value))
	entryValuePtrOff := entryValueOff + int32(unsafe.Offsetof(Scmer{}.ptr))
	entryValueAuxOff := entryValueOff + int32(unsafe.Offsetof(Scmer{}.aux))

	fallback, miss, hitFailure := ctx.ReserveLabel(), ctx.ReserveLabel(), ctx.ReserveLabel()
	ctx.MarkLabel(emitter.memoCheckLabel)

	// head := state.heads[position]; head != nil -> fallback (left
	// recursion may be in play).
	position := emitter.loadPosition()
	statePointer := emitter.statePointer()
	headAddr := emitter.sliceElemAddr(statePointer.Reg, headsOff, position.Reg, 3)
	ctx.FreeDesc(&statePointer)
	ctx.FreeDesc(&position)
	head := ctx.AllocReg()
	ctx.EmitMovRegMem(head, headAddr, 0)
	ctx.FreeReg(headAddr)
	ctx.EmitCmpRegImm32(head, 0)
	ctx.FreeReg(head)
	ctx.EmitJump(CondNotEqual, fallback)

	// offset := state.memoOffsets[position]; offset == 0 -> miss.
	position = emitter.loadPosition()
	statePointer = emitter.statePointer()
	offsetAddr := emitter.sliceElemAddr(statePointer.Reg, memoOffsetsOff, position.Reg, 2)
	ctx.FreeDesc(&statePointer)
	ctx.FreeDesc(&position)
	offset := ctx.AllocReg()
	ctx.EmitMovRegMemL(offset, offsetAddr, 0)
	ctx.FreeReg(offsetAddr)
	ctx.EmitCmpRegImm32(offset, 0)
	ctx.EmitJump(CondEqual, miss)

	// denseRule := state.program.memoRuleIndex[ruleID]. Unlike a per-site
	// design, ruleID is a runtime value in this shared block, so denseRule
	// costs one extra memory read instead of being a baked-in immediate.
	// Always >= 0 here: emitRuleRef only reaches this block for
	// non-lexical rules, and prepareMemoLayout gives every non-lexical
	// rule a non-negative dense index.
	ruleID := emitter.loadScratchInt(emitter.memoCheckRuleOff)
	statePointer = emitter.statePointer()
	programPtr := ctx.AllocRegExcept(statePointer.Reg, ruleID.Reg)
	ctx.EmitMovRegMem(programPtr, statePointer.Reg, programOff)
	ctx.FreeDesc(&statePointer)
	denseRuleAddr := emitter.sliceElemAddr(programPtr, memoRuleIndexOff, ruleID.Reg, 2)
	ctx.FreeReg(programPtr)
	ctx.FreeDesc(&ruleID)
	denseRule := ctx.AllocReg()
	ctx.EmitMovRegMemL(denseRule, denseRuleAddr, 0)
	ctx.FreeReg(denseRuleAddr)

	// index = offset - 1 + denseRule; state.memoRules[index] == 0 -> miss.
	ctx.EmitAddInt64(offset, denseRule)
	ctx.FreeReg(denseRule)
	ctx.EmitSubRegImm32(offset, 1)
	statePointer = emitter.statePointer()
	rulesAddr := emitter.sliceElemAddr(statePointer.Reg, memoRulesOff, offset, 2)
	ctx.FreeDesc(&statePointer)
	ctx.FreeReg(offset)
	entryIndex := ctx.AllocReg()
	ctx.EmitMovRegMemL(entryIndex, rulesAddr, 0)
	ctx.FreeReg(rulesAddr)
	ctx.EmitCmpRegImm32(entryIndex, 0)
	ctx.EmitJump(CondEqual, miss)

	// Fallthrough: a resolved memo entry exists at this position for this
	// rule and no left-recursion head is registered here - a definite hit.
	ctx.EmitSubRegImm32(entryIndex, 1)
	statePointer = emitter.statePointer()
	entryAddr := emitter.sliceElemAddr(statePointer.Reg, memoEntriesOff, entryIndex, 5)
	ctx.FreeDesc(&statePointer)
	ctx.FreeReg(entryIndex)

	active := ctx.AllocReg()
	ctx.EmitMovRegMemB(active, entryAddr, entryActiveOff)
	ctx.EmitCmpRegImm32(active, 0)
	ctx.FreeReg(active)
	ctx.EmitJump(CondNotEqual, fallback)

	entryPosition := ctx.AllocReg()
	ctx.EmitMovRegMemL(entryPosition, entryAddr, entryPositionOff)
	ctx.EmitStoreRegMem(entryPosition, ctx.StackReg, emitter.positionOff)
	ctx.FreeReg(entryPosition)

	entrySuccess := ctx.AllocReg()
	ctx.EmitMovRegMemB(entrySuccess, entryAddr, entrySuccessOff)
	ctx.EmitCmpRegImm32(entrySuccess, 0)
	ctx.FreeReg(entrySuccess)
	ctx.EmitJump(CondEqual, hitFailure)

	valuePtr := ctx.AllocReg()
	ctx.EmitMovRegMem(valuePtr, entryAddr, entryValuePtrOff)
	valueAux := ctx.AllocRegExcept(valuePtr)
	ctx.EmitMovRegMem(valueAux, entryAddr, entryValueAuxOff)
	ctx.FreeReg(entryAddr)
	cachedValue := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: valuePtr, Reg2: valueAux}
	ctx.BindReg(valuePtr, &cachedValue)
	ctx.BindReg(valueAux, &cachedValue)
	emitter.pushValue(cachedValue)
	acceptedCont := emitter.loadScratchInt(emitter.memoCheckAcceptedOff)
	ctx.EmitStoreRegMem(acceptedCont.Reg, ctx.StackReg, emitter.continuationOff)
	ctx.FreeDesc(&acceptedCont)
	ctx.EmitJmp(emitter.dispatchLabel)

	ctx.MarkLabel(hitFailure)
	rejectedCont := emitter.loadScratchInt(emitter.memoCheckRejectedOff)
	ctx.EmitStoreRegMem(rejectedCont.Reg, ctx.StackReg, emitter.continuationOff)
	ctx.FreeDesc(&rejectedCont)
	ctx.EmitJmp(emitter.dispatchLabel)

	// miss: definite first visit - record it and run the rule body. The
	// target rule is only known at runtime here (shared block), so
	// dispatch through the same jump table jitEmitParserProgramCore's own
	// entry dispatch uses, instead of a compile-time-constant jump.
	ctx.MarkLabel(miss)
	missRuleID := emitter.loadScratchInt(emitter.memoCheckRuleOff)
	missAccepted := emitter.loadScratchInt(emitter.memoCheckAcceptedOff)
	missRejected := emitter.loadScratchInt(emitter.memoCheckRejectedOff)
	missPosition := emitter.loadPosition()
	missState := emitter.statePointer()
	emitter.emitVoid(jitParserRecordFirstVisitNative, missState, missRuleID, missAccepted, missRejected, missPosition)
	ctx.FreeDesc(&missState)
	ctx.FreeDesc(&missPosition)
	ctx.FreeDesc(&missAccepted)
	ctx.FreeDesc(&missRejected)
	invalidMiss := ctx.ReserveLabel()
	ctx.EmitJumpTable(missRuleID.Reg, emitter.ruleLabels, invalidMiss)
	ctx.FreeDesc(&missRuleID)
	ctx.MarkLabel(invalidMiss)
	ctx.EmitJmp(emitter.failLabel)

	// fallback: left recursion may be in play - defer to the full
	// jitParserEnterRuleNative logic, unchanged from before this change.
	ctx.MarkLabel(fallback)
	fallbackRuleID := emitter.loadScratchInt(emitter.memoCheckRuleOff)
	fallbackAccepted := emitter.loadScratchInt(emitter.memoCheckAcceptedOff)
	fallbackRejected := emitter.loadScratchInt(emitter.memoCheckRejectedOff)
	position = emitter.loadPosition()
	statePointer = emitter.statePointer()
	enterArgs := []JITValueDesc{statePointer, fallbackRuleID, fallbackAccepted, fallbackRejected, position}
	var wordsBuf [16]goCallArgWord
	words := ctx.flattenArgs(enterArgs, &wordsBuf)
	var resultsBuf [16]Reg
	results := ctx.EmitGoCall(GoFuncAddr(jitParserEnterRuleNative), words, 3, &resultsBuf, nil)
	ctx.FreeDesc(&statePointer)
	ctx.FreeDesc(&position)
	ctx.FreeDesc(&fallbackAccepted)
	ctx.FreeDesc(&fallbackRejected)
	ctx.EmitCmpRegImm32(results[2], 0)
	for _, reg := range results {
		ctx.FreeReg(reg)
	}
	cacheMiss := ctx.ReserveLabel()
	ctx.EmitJump(CondEqual, cacheMiss)
	ctx.EmitStoreRegMem(results[0], ctx.StackReg, emitter.continuationOff)
	ctx.EmitStoreRegMem(results[1], ctx.StackReg, emitter.positionOff)
	ctx.FreeDesc(&fallbackRuleID)
	ctx.EmitJmp(emitter.dispatchLabel)
	ctx.MarkLabel(cacheMiss)
	invalidFallback := ctx.ReserveLabel()
	ctx.EmitJumpTable(fallbackRuleID.Reg, emitter.ruleLabels, invalidFallback)
	ctx.FreeDesc(&fallbackRuleID)
	ctx.MarkLabel(invalidFallback)
	ctx.EmitJmp(emitter.failLabel)
}

// emitLexicalRuleRef is the entry-check-free fast path carved out of
// emitRuleRef: jitParserPushRuleFrame directly (void, no memo/left-recursion
// bookkeeping applies) and an unconditional jump to the rule body, instead of
// a call whose memo/head lookup can only ever report a miss here.
func (emitter *jitParserEmitter) emitLexicalRuleRef(node *jitParserNode, success, failure JITLabel) {
	accepted, rejected := emitter.ctx.ReserveLabel(), emitter.ctx.ReserveLabel()
	position := emitter.loadPosition()
	statePointer := emitter.statePointer()
	emitter.emitVoid(jitParserPushRuleFrame, statePointer,
		jitParserScalar(int64(node.rule)),
		jitParserScalar(emitter.continuation(accepted)),
		jitParserScalar(emitter.continuation(rejected)),
		position,
		jitParserBoolScalar(false),
		jitParserBoolScalar(false))
	emitter.ctx.FreeDesc(&statePointer)
	emitter.ctx.FreeDesc(&position)
	emitter.ctx.EmitJmp(emitter.ruleLabels[node.rule])
	emitter.ctx.MarkLabel(accepted)
	if node.ignoreResult {
		emitter.discardValue()
	}
	emitter.ctx.EmitJmp(success)
	emitter.ctx.MarkLabel(rejected)
	emitter.ctx.EmitJmp(failure)
}

func (emitter *jitParserEmitter) emitRepeat(node *jitParserNode, rule int, success, failure JITLabel) {
	if node.noMemo {
		emitter.noMemoDepth++
		defer func() { emitter.noMemoDepth-- }()
	}
	emitter.pushCheckpoint()
	if !node.ignoreResult {
		emitter.emitVoid(jitParserPushMarkNative, emitter.state)
	}
	firstAccepted, firstRejected := emitter.ctx.ReserveLabel(), emitter.ctx.ReserveLabel()
	emitter.pushCheckpoint()
	emitter.emitNode(node.children[0], rule, firstAccepted, firstRejected)
	emitter.ctx.MarkLabel(firstAccepted)
	position := emitter.loadPosition()
	progress := emitter.ctx.EmitGoCallScalar(GoFuncAddr(jitParserCommitProgressNative), []JITValueDesc{emitter.state, position}, 1)
	emitter.ctx.FreeDesc(&position)
	emitter.ctx.EmitCmpRegImm32(progress.Reg, 0)
	emitter.ctx.FreeDesc(&progress)
	done := emitter.ctx.ReserveLabel()
	emitter.ctx.EmitJump(CondEqual, done)
	loop := emitter.ctx.ReserveLabel()
	emitter.ctx.EmitJmp(loop)
	emitter.ctx.MarkLabel(firstRejected)
	emitter.restoreCheckpoint()
	if node.kind == jitParserOneOrMore {
		emitter.restoreCheckpoint()
		emitter.ctx.EmitJmp(failure)
	} else {
		emitter.ctx.EmitJmp(done)
	}
	emitter.ctx.MarkLabel(loop)
	emitter.pushCheckpoint()
	separatorAccepted, iterationAccepted, iterationRejected := emitter.ctx.ReserveLabel(), emitter.ctx.ReserveLabel(), emitter.ctx.ReserveLabel()
	emitter.emitNode(node.children[1], rule, separatorAccepted, iterationRejected)
	emitter.ctx.MarkLabel(separatorAccepted)
	emitter.emitNode(node.children[0], rule, iterationAccepted, iterationRejected)
	emitter.ctx.MarkLabel(iterationAccepted)
	position = emitter.loadPosition()
	progress = emitter.ctx.EmitGoCallScalar(GoFuncAddr(jitParserCommitProgressNative), []JITValueDesc{emitter.state, position}, 1)
	emitter.ctx.FreeDesc(&position)
	emitter.ctx.EmitCmpRegImm32(progress.Reg, 0)
	emitter.ctx.FreeDesc(&progress)
	emitter.ctx.EmitJump(CondEqual, done)
	emitter.ctx.EmitJmp(loop)
	emitter.ctx.MarkLabel(iterationRejected)
	emitter.restoreCheckpoint()
	emitter.ctx.EmitJmp(done)
	emitter.ctx.MarkLabel(done)
	if !node.ignoreResult {
		emitter.emitVoid(jitParserMergeMarkNative, emitter.state, jitParserBoolScalar(false))
	}
	emitter.commitCheckpoint()
	emitter.ctx.EmitJmp(success)
}

func (emitter *jitParserEmitter) emitExclude(node *jitParserNode, rule int, success, failure JITLabel) {
	if len(node.children) == 0 {
		emitter.ctx.EmitJmp(failure)
		return
	}
	emitter.pushCheckpoint()
	mainAccepted, mainRejected := emitter.ctx.ReserveLabel(), emitter.ctx.ReserveLabel()
	emitter.emitNode(node.children[0], rule, mainAccepted, mainRejected)
	emitter.ctx.MarkLabel(mainRejected)
	emitter.restoreCheckpoint()
	emitter.ctx.EmitJmp(failure)
	emitter.ctx.MarkLabel(mainAccepted)
	continuation := emitter.loadPosition()
	emitter.emitVoid(jitParserPushPositionNative, emitter.state, continuation)
	emitter.ctx.FreeDesc(&continuation)
	for _, excluded := range node.children[1:] {
		start := emitter.ctx.EmitGoCallScalar(GoFuncAddr(jitParserCheckpointPositionNative), []JITValueDesc{emitter.state}, 1)
		emitter.storePosition(start)
		emitter.pushCheckpoint()
		excludedAccepted, excludedRejected := emitter.ctx.ReserveLabel(), emitter.ctx.ReserveLabel()
		emitter.emitNode(excluded, rule, excludedAccepted, excludedRejected)
		emitter.ctx.MarkLabel(excludedAccepted)
		emitter.restoreCheckpoint()
		discardedPosition := emitter.ctx.EmitGoCallScalar(GoFuncAddr(jitParserPopPositionNative), []JITValueDesc{emitter.state}, 1)
		emitter.ctx.FreeDesc(&discardedPosition)
		emitter.restoreCheckpoint()
		emitter.ctx.EmitJmp(failure)
		emitter.ctx.MarkLabel(excludedRejected)
		emitter.restoreCheckpoint()
	}
	continuation = emitter.ctx.EmitGoCallScalar(GoFuncAddr(jitParserPopPositionNative), []JITValueDesc{emitter.state}, 1)
	emitter.storePosition(continuation)
	emitter.commitCheckpoint()
	emitter.ctx.EmitJmp(success)
}

func (emitter *jitParserEmitter) emitNode(node *jitParserNode, rule int, success, failure JITLabel) {
	switch node.kind {
	case jitParserAtom, jitParserRegex:
		emitter.emitTerminal(node, rule, success, failure)
	case jitParserSequence:
		emitter.emitSequence(node, rule, success, failure)
	case jitParserChoice:
		emitter.emitChoice(node, rule, success, failure)
	case jitParserExclude:
		emitter.emitExclude(node, rule, success, failure)
	case jitParserZeroOrMore, jitParserOneOrMore:
		emitter.emitRepeat(node, rule, success, failure)
	case jitParserOptional:
		emitter.emitOptional(node, rule, success)
	case jitParserBind:
		emitter.emitBind(node, rule, success, failure)
	case jitParserCapture:
		emitter.emitCapture(node, rule, success, failure)
	case jitParserRuleRef:
		emitter.emitRuleRef(node, success, failure)
	case jitParserEnd:
		skipped := emitter.ctx.ReserveLabel()
		emitter.emitSkip(rule, skipped)
		emitter.ctx.MarkLabel(skipped)
		position := emitter.loadPosition()
		text := emitter.input
		emitter.ctx.EnsureDesc(&text)
		emitter.ctx.EmitShrRegImm8(text.Reg2, 8)
		emitter.ctx.EmitCmpInt64(position.Reg, text.Reg2)
		emitter.ctx.FreeDesc(&position)
		emitter.ctx.FreeDesc(&text)
		emitter.ctx.EmitJump(CondNotEqual, failure)
		if !node.ignoreResult {
			emitter.pushValue(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil(), NoHeapPointer: true})
		}
		emitter.ctx.EmitJmp(success)
	case jitParserEmpty:
		if !node.ignoreResult {
			emitter.pushValue(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil(), NoHeapPointer: true})
		}
		emitter.ctx.EmitJmp(success)
	case jitParserRest:
		if !node.ignoreResult {
			emitter.pushValue(emitter.substringAtPosition())
		}
		text := emitter.input
		emitter.ctx.EnsureDesc(&text)
		emitter.ctx.EmitShrRegImm8(text.Reg2, 8)
		emitter.ctx.EmitStoreRegMem(text.Reg2, emitter.ctx.StackReg, emitter.positionOff)
		emitter.ctx.FreeDesc(&text)
		emitter.ctx.EmitJmp(success)
	default:
		panic("jit: unsupported parser IR node")
	}
}

func jitEmitParserProgram(ctx *JITContext, _ []Scmer, descs []JITValueDesc, result JITValueDesc) JITValueDesc {
	if len(descs) != 4 || descs[0].Loc != LocImm || descs[0].Imm.GetTag() != tagAny {
		panic("jit: parser program expects a constant program, input, state and entry")
	}
	program, ok := descs[0].Imm.Any().(*jitParserProgram)
	if !ok || program == nil || len(program.rules) == 0 {
		panic("jit: invalid parser program")
	}
	return jitEmitParserProgramCore(ctx, program, descs[1], descs[2], descs[3], result, program.inlineActions, 0)
}

func jitEmitParserProgramCore(ctx *JITContext, program *jitParserProgram, input, state, entry JITValueDesc, result JITValueDesc, inlineActions bool, skipperRule int) JITValueDesc {
	emitter := &jitParserEmitter{ctx: ctx, program: program, input: input, state: state, entry: entry, inlineActions: inlineActions, skipperRule: skipperRule}
	emitter.input.Type = tagString
	emitter.state.Type = tagAny
	emitter.entry.Type = tagInt
	emitter.positionOff = ctx.AllocStack(8)
	emitter.continuationOff = ctx.AllocStack(8)
	emitter.generatorValueOff = ctx.AllocSpill(16)
	ctx.setStackPointer(jitStackRootFrameBP, emitter.generatorValueOff, true)
	ctx.EmitMovRegImm64(ctx.ScratchReg, 0)
	ctx.EmitStoreRegMem(ctx.ScratchReg, ctx.StackReg, emitter.positionOff)
	emitter.ruleLabels = make([]JITLabel, len(program.rules))
	for index := range emitter.ruleLabels {
		emitter.ruleLabels[index] = ctx.ReserveLabel()
	}
	emitter.dispatchLabel = ctx.ReserveLabel()
	emitter.skipLabel = ctx.ReserveLabel()
	emitter.memoCheckRuleOff = ctx.AllocStack(8)
	emitter.memoCheckAcceptedOff = ctx.AllocStack(8)
	emitter.memoCheckRejectedOff = ctx.AllocStack(8)
	emitter.memoCheckLabel = ctx.ReserveLabel()
	finished, failed := ctx.ReserveLabel(), ctx.ReserveLabel()
	emitter.failLabel = failed
	finishedID, failedID := emitter.continuation(finished), emitter.continuation(failed)

	entryDesc := emitter.entry
	ctx.EnsureDesc(&entryDesc)
	var entryScalar JITValueDesc
	if entryDesc.Loc == LocRegPair {
		entryScalar = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: entryDesc.Reg2, NoHeapPointer: true}
	} else if entryDesc.Loc == LocReg {
		entryScalar = entryDesc
	} else {
		panic("jit: parser entry has no integer representation")
	}
	position := emitter.loadPosition()
	statePointer := emitter.statePointer()
	enterArgs := []JITValueDesc{statePointer, entryScalar, jitParserScalar(finishedID), jitParserScalar(failedID), position}
	var wordsBuf [16]goCallArgWord
	words := ctx.flattenArgs(enterArgs, &wordsBuf)
	var resultsBuf [16]Reg
	results := ctx.EmitGoCall(GoFuncAddr(jitParserEnterRuleNative), words, 3, &resultsBuf, nil)
	ctx.FreeDesc(&statePointer)
	ctx.FreeDesc(&position)
	ctx.EmitCmpRegImm32(results[2], 0)
	for _, reg := range results {
		ctx.FreeReg(reg)
	}
	entryMiss := ctx.ReserveLabel()
	ctx.EmitJump(CondEqual, entryMiss)
	ctx.EmitStoreRegMem(results[0], ctx.StackReg, emitter.continuationOff)
	ctx.EmitStoreRegMem(results[1], ctx.StackReg, emitter.positionOff)
	ctx.EmitJmp(emitter.dispatchLabel)
	ctx.MarkLabel(entryMiss)
	invalidEntry := ctx.ReserveLabel()
	ctx.EmitJumpTable(entryScalar.Reg, emitter.ruleLabels, invalidEntry)
	ctx.FreeDesc(&entryDesc)
	ctx.MarkLabel(invalidEntry)
	ctx.EmitJmp(failed)

	for ruleID := range program.rules {
		ctx.MarkLabel(emitter.ruleLabels[ruleID])
		accepted, rejected := ctx.ReserveLabel(), ctx.ReserveLabel()
		emitter.emitNode(program.rules[ruleID].root, ruleID, accepted, rejected)
		ctx.MarkLabel(accepted)
		emitter.emitRuleReturn(ruleID, true)
		ctx.MarkLabel(rejected)
		emitter.emitRuleReturn(ruleID, false)
	}

	emitter.emitMemoCheckBlock()

	ctx.MarkLabel(finished)
	skipped := ctx.ReserveLabel()
	emitter.emitSkip(skipperRule, skipped)
	ctx.MarkLabel(skipped)
	endPosition := emitter.loadPosition()
	text := emitter.input
	ctx.EnsureDesc(&text)
	ctx.EmitShrRegImm8(text.Reg2, 8)
	ctx.EmitCmpInt64(endPosition.Reg, text.Reg2)
	ctx.FreeDesc(&endPosition)
	ctx.FreeDesc(&text)
	ctx.EmitJump(CondNotEqual, failed)
	resultOff := ctx.AllocStack(16)
	done := ctx.ReserveLabel()
	out := ctx.EmitGoCallScalar(GoFuncAddr(jitParserFinish), []JITValueDesc{emitter.state}, 2)
	out.Type = JITTypeUnknown
	ctx.EmitStoreScmerToStack(out, resultOff)
	ctx.FreeDesc(&out)
	ctx.EmitJmp(done)

	ctx.MarkLabel(emitter.skipLabel)
	emitter.emitSkipBody(emitter.dispatchLabel)

	ctx.MarkLabel(emitter.dispatchLabel)
	continuation := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: ctx.AllocReg(), NoHeapPointer: true}
	ctx.BindReg(continuation.Reg, &continuation)
	ctx.EmitMovRegMem(continuation.Reg, ctx.StackReg, emitter.continuationOff)
	invalidContinuation := ctx.ReserveLabel()
	ctx.EmitJumpTable(continuation.Reg, emitter.continuations, invalidContinuation)
	ctx.FreeDesc(&continuation)
	ctx.MarkLabel(invalidContinuation)
	ctx.EmitJmp(failed)

	ctx.MarkLabel(failed)
	panicResult := ctx.EmitGoCallScalar(GoFuncAddr(jitParserPanic), []JITValueDesc{emitter.state, emitter.input}, 2)
	panicResult.Type = JITTypeUnknown
	ctx.EmitStoreScmerToStack(panicResult, resultOff)
	ctx.FreeDesc(&panicResult)
	ctx.MarkLabel(done)
	return jitPlaceScmerIntoTarget(ctx, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: resultOff, Rooted: true}, result)
}

func (emitter *jitParserEmitter) emitRuleReturn(ruleID int, success bool) {
	position := emitter.loadPosition()
	if success && emitter.inlineActions {
		// Grammar branches leave only descriptor-owned values live. Generator
		// lowering starts a new expression region and may immediately reserve a
		// result pair, so reclaim dead path temporaries before entering it.
		emitter.ctx.ReclaimUntrackedRegs()
		rule := &emitter.program.rules[ruleID]
		var value JITValueDesc
		if rule.generator.IsNil() {
			value = emitter.ctx.EmitGoCallScalar(GoFuncAddr(jitParserRuleValueNative), []JITValueDesc{emitter.state}, 2)
			value.Type = JITTypeUnknown
		} else {
			generatorAlloc := emitter.ctx.SnapshotAllocState()
			lexicalRules := make([]int, 0, 4)
			for lexicalRule := ruleID; lexicalRule >= 0; lexicalRule = emitter.program.rules[lexicalRule].lexicalParent {
				lexicalRules = append(lexicalRules, lexicalRule)
			}
			bindingEnv := emitter.ctx.StabilizeJITEnv(rule.jitOuter)
			allArgs := make([][]JITValueDesc, len(lexicalRules))
			for depth := len(lexicalRules) - 1; depth >= 0; depth-- {
				lexicalRuleID := lexicalRules[depth]
				lexicalRule := &emitter.program.rules[lexicalRuleID]
				args := make([]JITValueDesc, len(lexicalRule.bindings))
				for index := range args {
					callArgs := []JITValueDesc{
						emitter.state, jitParserScalar(int64(lexicalRuleID)), jitParserScalar(int64(index)),
					}
					var wordsBuf [16]goCallArgWord
					words := emitter.ctx.flattenArgs(callArgs, &wordsBuf)
					off := emitter.ctx.AllocSpill(16)
					emitter.ctx.setStackPointer(jitStackRootFrameBP, off, true)
					emitter.ctx.EmitGoCallToFrame(GoFuncAddr(jitParserBindingValueForRuleNative), words, []int32{off, off + 8})
					args[index] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: off, Rooted: true}
				}
				frameEnv := &JITEnv{Vars: make(map[Symbol]JITValueDesc, len(lexicalRule.bindings)), Numbered: args, Outer: bindingEnv}
				for index, symbol := range lexicalRule.bindings {
					frameEnv.Vars[symbol] = args[index]
				}
				bindingEnv = frameEnv
				allArgs[depth] = args
			}
			valueTarget := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: emitter.generatorValueOff, Rooted: true}
			if action := rule.compiledAction; action != nil {
				actionArgs := append([]JITValueDesc(nil), allArgs[0]...)
				for _, symbol := range rule.actionCaptures {
					capture, exists := bindingEnv.Lookup(symbol)
					if !exists {
						panic("jit: parser action capture " + string(symbol) + " is unavailable")
					}
					actionArgs = append(actionArgs, capture)
				}
				entryValue := NewAny(action)
				emitter.ctx.TrackImm(entryValue)
				entry := emitter.immPair(entryValue)
				args := jitMaterializeVirtualGoSlice(emitter.ctx, actionArgs)
				value = emitter.ctx.EmitGoCallScalar(GoFuncAddr(jitParserCallCompiledAction), []JITValueDesc{entry, args}, 2)
				value.Type = JITTypeUnknown
				value.Rooted = true
				value = jitPlaceScmerIntoTarget(emitter.ctx, value, valueTarget)
				emitter.ctx.FreeDesc(&entry)
				emitter.ctx.FreeDesc(&args)
			} else {
				emitter.ctx.ReclaimUntrackedRegs()
				outerRegs := emitter.ctx.PreserveOuterRegs()
				proc := Proc{Body: rule.generator, NumVars: len(rule.bindings), NumberedOnly: true}
				value = JITEmitProcInlineWithEnv(emitter.ctx, &proc, bindingEnv, emitter.ctx.SliceBase, valueTarget)
				emitter.ctx.RestoreOuterRegs(outerRegs)
			}
			for _, args := range allArgs {
				for index := range args {
					emitter.ctx.FreeDesc(&args[index])
				}
			}
			emitter.ctx.RestoreAllocState(generatorAlloc)
			emitter.ctx.setStackPointer(jitStackRootFrameBP, emitter.generatorValueOff, !value.NoHeapPointer)
		}
		if value.Loc == LocImm {
			value = jitCopyScmerToPair(emitter.ctx, value)
		}
		value = jitRootScmer(emitter.ctx, value)
		var argsBuf [16]goCallArgWord
		args := emitter.ctx.flattenArgs([]JITValueDesc{emitter.state, position, value}, &argsBuf)
		var resultsBuf [16]Reg
		results := emitter.ctx.EmitGoCall(GoFuncAddr(jitParserReturnRuleValueNative), args, 3, &resultsBuf, nil)
		emitter.ctx.FreeDesc(&value)
		emitter.ctx.FreeDesc(&position)
		emitter.ctx.EmitStoreRegMem(results[0], emitter.ctx.StackReg, emitter.continuationOff)
		emitter.ctx.EmitStoreRegMem(results[1], emitter.ctx.StackReg, emitter.positionOff)
		emitter.ctx.EmitCmpRegImm32(results[2], 0)
		for _, reg := range results {
			emitter.ctx.FreeReg(reg)
		}
		emitter.ctx.EmitJump(CondEqual, emitter.dispatchLabel)
		emitter.ctx.EmitJmp(emitter.ruleLabels[ruleID])
		return
	}
	var argsBuf [16]goCallArgWord
	args := emitter.ctx.flattenArgs([]JITValueDesc{emitter.state, position, jitParserBoolScalar(success)}, &argsBuf)
	var resultsBuf [16]Reg
	results := emitter.ctx.EmitGoCall(GoFuncAddr(jitParserReturnRuleNative), args, 3, &resultsBuf, nil)
	emitter.ctx.FreeDesc(&position)
	emitter.ctx.EmitStoreRegMem(results[0], emitter.ctx.StackReg, emitter.continuationOff)
	emitter.ctx.EmitStoreRegMem(results[1], emitter.ctx.StackReg, emitter.positionOff)
	emitter.ctx.EmitCmpRegImm32(results[2], 0)
	for _, reg := range results {
		emitter.ctx.FreeReg(reg)
	}
	emitter.ctx.EmitJump(CondEqual, emitter.dispatchLabel)
	emitter.ctx.EmitJmp(emitter.ruleLabels[ruleID])
}
