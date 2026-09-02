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
	for _, child := range node.children {
		emitter.pushCheckpoint()
		accepted, rejected := emitter.ctx.ReserveLabel(), emitter.ctx.ReserveLabel()
		emitter.emitNode(child, rule, accepted, rejected)
		emitter.ctx.MarkLabel(accepted)
		emitter.commitCheckpoint()
		emitter.ctx.EmitJmp(success)
		emitter.ctx.MarkLabel(rejected)
		emitter.restoreCheckpoint()
	}
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
	accepted, rejected := emitter.ctx.ReserveLabel(), emitter.ctx.ReserveLabel()
	emitter.emitNode(node.children[0], rule, accepted, rejected)
	emitter.ctx.MarkLabel(accepted)
	start := emitter.ctx.EmitGoCallScalar(GoFuncAddr(jitParserCheckpointPositionNative), []JITValueDesc{emitter.state}, 1)
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
	accepted, rejected := emitter.ctx.ReserveLabel(), emitter.ctx.ReserveLabel()
	position := emitter.loadPosition()
	enterArgs := []JITValueDesc{emitter.state, jitParserScalar(int64(node.rule)), jitParserScalar(emitter.continuation(accepted)),
		jitParserScalar(emitter.continuation(rejected)), position}
	var wordsBuf [16]goCallArgWord
	words := emitter.ctx.flattenArgs(enterArgs, &wordsBuf)
	var resultsBuf [16]Reg
	results := emitter.ctx.EmitGoCall(GoFuncAddr(jitParserEnterRuleNative), words, 3, &resultsBuf, nil)
	emitter.ctx.FreeDesc(&position)
	emitter.ctx.EmitCmpRegImm32(results[2], 0)
	for _, reg := range results {
		emitter.ctx.FreeReg(reg)
	}
	cacheMiss := emitter.ctx.ReserveLabel()
	emitter.ctx.EmitJump(CondEqual, cacheMiss)
	emitter.ctx.EmitStoreRegMem(results[0], emitter.ctx.StackReg, emitter.continuationOff)
	emitter.ctx.EmitStoreRegMem(results[1], emitter.ctx.StackReg, emitter.positionOff)
	emitter.ctx.EmitJmp(emitter.dispatchLabel)
	emitter.ctx.MarkLabel(cacheMiss)
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
	finished, failed := ctx.ReserveLabel(), ctx.ReserveLabel()
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
	enterArgs := []JITValueDesc{emitter.state, entryScalar, jitParserScalar(finishedID), jitParserScalar(failedID), position}
	var wordsBuf [16]goCallArgWord
	words := ctx.flattenArgs(enterArgs, &wordsBuf)
	var resultsBuf [16]Reg
	results := ctx.EmitGoCall(GoFuncAddr(jitParserEnterRuleNative), words, 3, &resultsBuf, nil)
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
