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
	"regexp"
	"slices"
	"sync"
	"unicode"
	"unicode/utf8"
)

const (
	jitParserLargeInputBytes           = 64 << 10
	jitParserMemoEntriesPerByteHint    = 20
	jitParserMemoPreallocateLimit      = 4 << 20
	jitParserMemoRuleSlotsPerByteHint  = 120
	jitParserMemoRulePreallocateLimit  = 24 << 20
	jitParserRetainedMemoEntryCapacity = 1 << 18
)

type jitParserNodeKind uint8

const (
	jitParserAtom jitParserNodeKind = iota
	jitParserRegex
	jitParserSequence
	jitParserChoice
	jitParserExclude
	jitParserZeroOrMore
	jitParserOneOrMore
	jitParserOptional
	jitParserBind
	jitParserCapture
	jitParserRuleRef
	jitParserEnd
	jitParserEmpty
	jitParserRest
)

// jitParserNode is compile-time-only, architecture-independent parser IR.
// Runtime code contains direct control flow for these nodes; it never walks
// this representation.
type jitParserNode struct {
	kind         jitParserNodeKind
	children     []*jitParserNode
	rule         int
	binding      int
	value        Scmer
	regex        *jitRegexProgram
	skipWS       bool
	ignoreResult bool
	noMemo       bool
	description  string
}

type jitParserRule struct {
	root           *jitParserNode
	generator      Scmer
	action         Scmer
	bindings       []Symbol
	bindingLookup  map[Symbol]int
	skipper        *jitRegexProgram
	outer          *Env
	jitOuter       *JITEnv
	lexicalParent  int
	compiledAction *JITEntryPoint
	actionCaptures []Symbol
}

type jitParserProgram struct {
	rules         []jitParserRule
	parserRule    map[*ScmParser]int
	memoRuleIndex []int32
	memoRuleCount int
	inlineActions bool
	pool          sync.Pool
}

type jitParserBuilder struct {
	program       *jitParserProgram
	templates     map[jitParserTemplateKey]int
	active        map[*JITParserTemplate]int
	inlineActions bool
	compileEnv    *JITEnv
}

type jitParserTemplateKey struct {
	template      *JITParserTemplate
	lexicalParent int
}

// JITParserTemplate is a compile-time parser closure. Syntax and generator are
// static, while Outer describes the surrounding native frame. Keeping this
// value compiler-only lets a call consume the complete mutually recursive
// grammar without allocating ScmParser objects between local definitions.
type JITParserTemplate struct {
	Name         Symbol
	Syntax       Scmer
	Generator    Scmer
	Whitespace   Scmer
	IgnoreResult bool
	Outer        *JITEnv
	RuntimeOuter *Env
}

type jitParserDeferred struct {
	symbol Symbol
}

func jitUnwrapParserSyntax(value Scmer) Scmer {
	for value.IsSourceInfo() {
		value = value.SourceInfo().value
	}
	if value.GetTag() == tagAny {
		if optimized, ok := value.Any().(*optimizedParserSyntax); ok {
			return jitUnwrapParserSyntax(optimized.Syntax)
		}
	}
	return value
}

func jitBuildParserProgram(parser *ScmParser) *jitParserProgram {
	return jitBuildParserPrograms([]*ScmParser{parser})
}

func jitBuildParserPrograms(parsers []*ScmParser) *jitParserProgram {
	program := &jitParserProgram{parserRule: make(map[*ScmParser]int), inlineActions: true}
	builder := &jitParserBuilder{
		program: program, templates: make(map[jitParserTemplateKey]int),
		active: make(map[*JITParserTemplate]int), inlineActions: true,
	}
	for _, parser := range parsers {
		if parser == nil {
			panic("jit: nil parser")
		}
		builder.addScmParser(parser)
	}
	program.prepareMemoLayout()
	program.pool.New = func() any { return new(jitParserState) }
	return program
}

func jitBuildParserTemplateProgram(template *JITParserTemplate) (*jitParserProgram, int) {
	program := &jitParserProgram{parserRule: make(map[*ScmParser]int), inlineActions: true}
	builder := &jitParserBuilder{
		program: program, templates: make(map[jitParserTemplateKey]int),
		active: make(map[*JITParserTemplate]int), inlineActions: true, compileEnv: template.Outer,
	}
	rule := builder.addTemplate(template, -1)
	program.prepareMemoLayout()
	program.pool.New = func() any { return new(jitParserState) }
	return program, rule
}

func (program *jitParserProgram) prepareMemoLayout() {
	program.memoRuleIndex = make([]int32, len(program.rules))
	for index := range program.memoRuleIndex {
		program.memoRuleIndex[index] = -1
	}
	for rule := range program.rules {
		if program.rules[rule].lexicalParent < 0 {
			program.memoRuleIndex[rule] = int32(program.memoRuleCount)
			program.memoRuleCount++
		}
	}
}

func (builder *jitParserBuilder) addTemplate(template *JITParserTemplate, lexicalParent int) int {
	if template == nil {
		panic("jit: nil parser template")
	}
	lexicalParent = builder.templateLexicalParent(template, lexicalParent)
	key := jitParserTemplateKey{template: template, lexicalParent: lexicalParent}
	if rule, exists := builder.templates[key]; exists {
		return rule
	}
	// A recursive reference belongs to the rule currently being assembled. It
	// must not create a second context-specialized copy whose parent is itself.
	// This also closes mutually recursive template groups while still allowing
	// the same reusable template to be specialized for independent callers.
	if rule, recursive := builder.active[template]; recursive {
		return rule
	}
	ruleID := len(builder.program.rules)
	builder.templates[key] = ruleID
	builder.active[template] = ruleID
	defer delete(builder.active, template)
	builder.program.rules = append(builder.program.rules, jitParserRule{
		generator: template.Generator, bindingLookup: make(map[Symbol]int), jitOuter: template.Outer, outer: template.RuntimeOuter,
		lexicalParent: lexicalParent,
	})
	builder.predeclareBindings(ruleID, template.Syntax)
	var skipper *jitRegexProgram
	if template.Whitespace.IsNil() {
		skipper = jitCompileRegexProgram(packratDefaultSkipper())
	} else {
		skipper = jitCompileRegexProgram(regexp.MustCompile(template.Whitespace.String()))
	}
	root := builder.buildNode(jitUnwrapParserSyntax(template.Syntax), template.RuntimeOuter, template.Outer, ruleID, template.IgnoreResult)
	builder.program.rules[ruleID].root = root
	builder.program.rules[ruleID].skipper = skipper
	if lexicalParent < 0 {
		builder.finishRule(ruleID)
	}
	return ruleID
}

// templateLexicalParent keeps a caller specialization only when the template's
// generator can actually read a capture declared by that caller chain. Most
// grammar rules are closed over their Scheme environment and can therefore be
// shared globally; cloning those rules per call edge makes recursive SQL
// grammars grow combinatorially.
func (builder *jitParserBuilder) templateLexicalParent(template *JITParserTemplate, candidate int) int {
	if candidate < 0 || template.Generator.IsNil() {
		return -1
	}
	free := jitLambdaFreeSymbols(NewSlice(nil), template.Generator)
	if len(free) == 0 {
		return -1
	}
	used := make(map[Symbol]struct{}, len(free))
	for _, symbol := range free {
		used[symbol] = struct{}{}
	}
	for ruleID := candidate; ruleID >= 0; ruleID = builder.program.rules[ruleID].lexicalParent {
		for _, binding := range builder.program.rules[ruleID].bindings {
			if _, needed := used[binding]; needed {
				return candidate
			}
		}
	}
	return -1
}

func (builder *jitParserBuilder) predeclareBindings(ruleID int, syntax Scmer) {
	var visit func(Scmer)
	visit = func(value Scmer) {
		value = jitUnwrapParserSyntax(value)
		if !value.IsSlice() {
			return
		}
		items := value.Slice()
		if len(items) == 0 {
			return
		}
		if head, ok := scmerSymbol(jitUnwrapParserSyntax(items[0])); ok {
			switch head {
			case "parser":
				return
			case "define":
				if len(items) == 3 {
					if symbol, valid := scmerSymbol(jitUnwrapParserSyntax(items[1])); valid {
						builder.binding(ruleID, symbol)
					}
					visit(items[2])
				}
				return
			}
		}
		for _, item := range items {
			visit(item)
		}
	}
	visit(syntax)
}

func jitCompileEnvironmentParsers(environment *Env) {
	if !jitEnabled || environment == nil {
		return
	}
	environment = environment.definitionTarget()
	parsers := make([]*ScmParser, 0)
	seen := make(map[*ScmParser]struct{})
	for _, value := range environment.Vars {
		if !value.IsParser() || value.Parser() == nil || value.Parser().Compiled != nil {
			continue
		}
		parser := value.Parser()
		if _, exists := seen[parser]; exists {
			continue
		}
		seen[parser] = struct{}{}
		parsers = append(parsers, parser)
	}
	if len(parsers) == 0 {
		return
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			if _, incomplete := recovered.(jitParserDeferred); incomplete {
				return
			}
			panic(recovered)
		}
	}()
	program := jitBuildParserPrograms(parsers)
	body := NewSlice([]Scmer{
		NewSymbol("jit-parser-program"), NewAny(program),
		NewNthLocalVar(0), NewNthLocalVar(1), NewNthLocalVar(2),
	})
	compiled := jitCompile(NewProcStruct(Proc{
		Params: NewSlice([]Scmer{NewSymbol("input"), NewSymbol("state"), NewSymbol("entry")}),
		Body:   body, En: environment, NumVars: 3, NumberedOnly: true,
	}))
	if !compiled.IsProc() || compiled.Proc() == nil || compiled.Proc().Compiled == nil {
		panic("jit: parser grammar could not be compiled")
	}
	entry := compiled.Proc().Compiled
	entry.DebugName = "parser grammar"
	maybeLogJITCodeName(entry)
	for parser, rule := range program.parserRule {
		parser.Compiled = entry
		parser.JITProgram = program
		parser.JITRule = rule
	}
}

func (builder *jitParserBuilder) addScmParser(parser *ScmParser) int {
	if rule, ok := builder.program.parserRule[parser]; ok {
		return rule
	}
	ruleID := len(builder.program.rules)
	builder.program.parserRule[parser] = ruleID
	builder.program.rules = append(builder.program.rules, jitParserRule{
		generator:     parser.Generator,
		bindingLookup: make(map[Symbol]int),
		outer:         parser.Outer,
		jitOuter:      jitCapturedEnv(parser.Outer),
		lexicalParent: -1,
	})
	builder.predeclareBindings(ruleID, parser.Syntax)
	var skipper *jitRegexProgram
	if parser.Skipper != nil {
		skipper = jitCompileRegexProgram(parser.Skipper)
	} else {
		skipper = jitCompileRegexProgram(packratDefaultSkipper())
	}
	ignoreResult := !parser.Generator.IsNil()
	jitOuter := (*JITEnv)(nil)
	if builder.inlineActions {
		jitOuter = builder.program.rules[ruleID].jitOuter
	}
	root := builder.buildNode(jitUnwrapParserSyntax(parser.Syntax), parser.Outer, jitOuter, ruleID, ignoreResult)
	builder.program.rules[ruleID].root = root
	builder.program.rules[ruleID].skipper = skipper
	if !builder.inlineActions || builder.program.rules[ruleID].lexicalParent < 0 {
		builder.finishRule(ruleID)
	}
	return ruleID
}

func packratDefaultSkipper() *regexp.Regexp {
	return regexp.MustCompile("^(?:/\\*.*?\\*/|[\\r\\n\\t ]+)+")
}

func (builder *jitParserBuilder) addNestedRule(syntax, generator, whitespace Scmer, outer *Env, lexicalParent int) int {
	ruleID := len(builder.program.rules)
	builder.program.rules = append(builder.program.rules, jitParserRule{
		generator:     generator,
		bindingLookup: make(map[Symbol]int),
		outer:         outer,
		lexicalParent: lexicalParent,
	})
	builder.predeclareBindings(ruleID, syntax)
	var skipper *jitRegexProgram
	if whitespace.IsNil() {
		skipper = jitCompileRegexProgram(packratDefaultSkipper())
	} else {
		skipper = jitCompileRegexProgram(regexp.MustCompile(whitespace.String()))
	}
	root := builder.buildNode(jitUnwrapParserSyntax(syntax), outer, nil, ruleID, !generator.IsNil())
	builder.program.rules[ruleID].root = root
	builder.program.rules[ruleID].skipper = skipper
	if !builder.inlineActions || builder.program.rules[ruleID].lexicalParent < 0 {
		builder.finishRule(ruleID)
	}
	return ruleID
}

func (builder *jitParserBuilder) finishRule(ruleID int) {
	rule := &builder.program.rules[ruleID]
	if rule.generator.IsNil() {
		return
	}
	params := make([]Scmer, len(rule.bindings))
	for index, binding := range rule.bindings {
		params[index] = NewSymbol(string(binding))
	}
	if rule.jitOuter != nil {
		for _, symbol := range jitLambdaFreeSymbols(NewSlice(params), rule.generator) {
			if _, exists := rule.jitOuter.Lookup(symbol); exists {
				rule.actionCaptures = append(rule.actionCaptures, symbol)
				params = append(params, NewSymbol(string(symbol)))
			}
		}
	}
	outer := rule.outer
	if outer == nil {
		outer = &Globalenv
	}
	lambda := CloneOptimizerExpression(NewSlice([]Scmer{NewSymbol("lambda"), NewSlice(params), rule.generator}))
	action := Eval(Optimize(lambda, outer, nil), outer)
	if jitEnabled && jitRequiredLocalSlots(rule.generator, len(params)) <= len(params) {
		if compiled := jitCompileModeDeferred(true, action); compiled.IsProc() && compiled.Proc() != nil && compiled.Proc().Compiled != nil {
			action = compiled
			rule.compiledAction = compiled.Proc().Compiled
			rule.compiledAction.DebugName = fmt.Sprintf("parser action %d", ruleID)
		}
	}
	if !builder.inlineActions {
		rule.action = action
	}
}

func jitParserCallCompiledAction(entryValue Scmer, args []Scmer) Scmer {
	entry, ok := entryValue.Any().(*JITEntryPoint)
	if !ok || entry == nil {
		panic("jit: invalid compiled parser action")
	}
	return entry.Call(args...)
}

func (builder *jitParserBuilder) binding(ruleID int, symbol Symbol) int {
	rule := &builder.program.rules[ruleID]
	if index, ok := rule.bindingLookup[symbol]; ok {
		return index
	}
	index := len(rule.bindings)
	rule.bindingLookup[symbol] = index
	rule.bindings = append(rule.bindings, symbol)
	return index
}

func jitParserBool(items []Scmer, index int, fallback bool) bool {
	if index >= len(items) {
		return fallback
	}
	return items[index].Bool()
}

func (builder *jitParserBuilder) buildNodeJIT(value Scmer, outer *JITEnv, ruleID int, ignoreResult bool) *jitParserNode {
	return builder.buildNode(value, nil, outer, ruleID, ignoreResult)
}

func (builder *jitParserBuilder) buildNode(value Scmer, outer *Env, jitOuter *JITEnv, ruleID int, ignoreResult bool) *jitParserNode {
	value = jitUnwrapParserSyntax(value)
	switch value.GetTag() {
	case tagFastDict:
		if value.FastDict() == nil {
			value = NewSlice(nil)
		} else {
			value = NewSlice(value.FastDict().Pairs)
		}
	case tagParser:
		return &jitParserNode{kind: jitParserRuleRef, rule: builder.addScmParser(value.Parser()), ignoreResult: ignoreResult}
	case tagString:
		literal := value.String()
		return &jitParserNode{
			kind:         jitParserAtom,
			value:        value,
			regex:        jitCompileRegexProgram(regexp.MustCompile("^(?:" + regexp.QuoteMeta(literal) + ")")),
			skipWS:       true,
			ignoreResult: ignoreResult,
			description:  literal,
		}
	case tagSymbol:
		switch value.Symbol() {
		case "$":
			return &jitParserNode{kind: jitParserEnd, ignoreResult: ignoreResult, description: "end of input"}
		case "empty":
			return &jitParserNode{kind: jitParserEmpty, ignoreResult: ignoreResult}
		case "rest":
			return &jitParserNode{kind: jitParserRest, ignoreResult: ignoreResult}
		}
		if jitOuter != nil {
			if desc, ok := jitOuter.Lookup(value.Symbol()); ok && desc.Loc == LocParserTemplate && desc.Parser != nil {
				return &jitParserNode{kind: jitParserRuleRef, rule: builder.addTemplate(desc.Parser, ruleID), ignoreResult: ignoreResult}
			}
			for env := jitOuter; env != nil; env = env.Outer {
				for _, desc := range env.Numbered {
					if desc.Loc == LocParserTemplate && desc.Parser != nil && desc.Parser.Name == value.Symbol() {
						return &jitParserNode{kind: jitParserRuleRef, rule: builder.addTemplate(desc.Parser, ruleID), ignoreResult: ignoreResult}
					}
				}
			}
		}
		if builder.compileEnv != nil {
			if desc, ok := builder.compileEnv.Lookup(value.Symbol()); ok && desc.Loc == LocParserTemplate && desc.Parser != nil {
				return &jitParserNode{kind: jitParserRuleRef, rule: builder.addTemplate(desc.Parser, ruleID), ignoreResult: ignoreResult}
			}
		}
		if outer == nil {
			panic("jit: unresolved parser template symbol: " + value.String())
		}
		env := outer.FindRead(value.Symbol())
		parserValue, ok := env.Vars[value.Symbol()]
		if !ok {
			panic(jitParserDeferred{symbol: value.Symbol()})
		}
		if !parserValue.IsParser() {
			panic("jit: parser rule does not resolve to a parser: " + value.String())
		}
		return &jitParserNode{kind: jitParserRuleRef, rule: builder.addScmParser(parserValue.Parser()), ignoreResult: ignoreResult}
	case tagNthLocalVar:
		index := int(value.NthLocalVar())
		if jitOuter != nil && index >= 0 && index < len(jitOuter.Numbered) {
			desc := jitOuter.Numbered[index]
			if desc.Loc == LocParserTemplate && desc.Parser != nil {
				return &jitParserNode{kind: jitParserRuleRef, rule: builder.addTemplate(desc.Parser, ruleID), ignoreResult: ignoreResult}
			}
		}
		if outer == nil || index < 0 || index >= len(outer.VarsNumbered) || !outer.VarsNumbered[index].IsParser() {
			panic("jit: unresolved numbered parser rule")
		}
		return &jitParserNode{kind: jitParserRuleRef, rule: builder.addScmParser(outer.VarsNumbered[index].Parser()), ignoreResult: ignoreResult}
	case tagAny:
		panic("jit: parser node has no retained grammar syntax")
	}

	if !value.IsSlice() {
		panic("jit: unsupported parser syntax " + value.String())
	}
	items := value.Slice()
	if len(items) == 0 {
		panic("jit: invalid empty parser syntax")
	}
	head, hasHead := scmerSymbol(jitUnwrapParserSyntax(items[0]))
	if hasHead {
		switch head {
		case "parser":
			generator, whitespace := NewNil(), NewNil()
			if len(items) > 2 {
				generator = items[2]
			}
			if len(items) > 3 {
				whitespace = items[3]
			}
			if jitOuter != nil {
				template := &JITParserTemplate{Syntax: items[1], Generator: generator, Whitespace: whitespace, IgnoreResult: !generator.IsNil(), Outer: jitOuter, RuntimeOuter: outer}
				return &jitParserNode{kind: jitParserRuleRef, rule: builder.addTemplate(template, ruleID), ignoreResult: ignoreResult}
			}
			return &jitParserNode{kind: jitParserRuleRef, rule: builder.addNestedRule(items[1], generator, whitespace, outer, ruleID), ignoreResult: ignoreResult}
		case "atom":
			literal := items[1].String()
			caseInsensitive := jitParserBool(items, 2, false)
			pattern := regexp.QuoteMeta(literal)
			if caseInsensitive {
				pattern = "(?i:" + pattern + ")"
			}
			result := items[1]
			if len(items) > 4 {
				result = items[4]
			}
			return &jitParserNode{kind: jitParserAtom, value: result,
				regex:  jitCompileRegexProgram(regexp.MustCompile("^(?:" + pattern + ")")),
				skipWS: jitParserBool(items, 3, true), ignoreResult: ignoreResult, description: literal}
		case "regex":
			pattern := items[1].String()
			if jitParserBool(items, 2, false) {
				pattern = "(?i:" + pattern + ")"
			}
			return &jitParserNode{kind: jitParserRegex,
				regex:  jitCompileRegexProgram(regexp.MustCompile("^(?:" + pattern + ")")),
				skipWS: jitParserBool(items, 3, true), ignoreResult: ignoreResult, description: items[1].String()}
		case "list":
			return builder.buildChildren(jitParserSequence, items[1:], outer, jitOuter, ruleID, ignoreResult)
		case "or":
			return builder.buildChildren(jitParserChoice, items[1:], outer, jitOuter, ruleID, ignoreResult)
		case "not":
			return builder.buildChildren(jitParserExclude, items[1:], outer, jitOuter, ruleID, ignoreResult)
		case "*", "+":
			children := []*jitParserNode{builder.buildNode(items[1], outer, jitOuter, ruleID, ignoreResult)}
			if len(items) > 2 {
				children = append(children, builder.buildNode(items[2], outer, jitOuter, ruleID, true))
			} else {
				children = append(children, &jitParserNode{kind: jitParserEmpty, ignoreResult: true})
			}
			kind := jitParserZeroOrMore
			if head == "+" {
				kind = jitParserOneOrMore
			}
			return &jitParserNode{kind: kind, children: children, ignoreResult: ignoreResult,
				noMemo: head == "*" && len(items) > 3 && items[3].Bool()}
		case "?":
			child := builder.buildChildren(jitParserSequence, items[1:], outer, jitOuter, ruleID, ignoreResult)
			if len(items) == 2 {
				child = builder.buildNode(items[1], outer, jitOuter, ruleID, ignoreResult)
			}
			return &jitParserNode{kind: jitParserOptional, children: []*jitParserNode{child}, ignoreResult: ignoreResult}
		case "define":
			if len(items) != 3 {
				panic("jit: malformed parser define")
			}
			symbol, ok := scmerSymbol(items[1])
			if !ok {
				panic("jit: parser define requires a symbol")
			}
			return &jitParserNode{kind: jitParserBind,
				children: []*jitParserNode{builder.buildNode(items[2], outer, jitOuter, ruleID, false)},
				binding:  builder.binding(ruleID, symbol), ignoreResult: ignoreResult}
		case "capture":
			return &jitParserNode{kind: jitParserCapture,
				children: []*jitParserNode{builder.buildNode(items[1], outer, jitOuter, ruleID, false)}, ignoreResult: ignoreResult}
		case "empty":
			return &jitParserNode{kind: jitParserEmpty, ignoreResult: ignoreResult}
		}
	}
	return builder.buildChildren(jitParserSequence, items, outer, jitOuter, ruleID, ignoreResult)
}

func (builder *jitParserBuilder) buildChildren(kind jitParserNodeKind, values []Scmer, outer *Env, jitOuter *JITEnv, ruleID int, ignoreResult bool) *jitParserNode {
	children := make([]*jitParserNode, len(values))
	for index, child := range values {
		children[index] = builder.buildNode(child, outer, jitOuter, ruleID, ignoreResult)
	}
	return &jitParserNode{kind: kind, children: children, ignoreResult: ignoreResult}
}

type jitParserCallFrame struct {
	rule         int
	success      int
	failure      int
	position     int
	valueBase    int
	bindingBase  int
	mutationBase int
	memoize      bool
	memoInitial  bool
	transient    bool
	growing      bool
}

type jitParserMemoKey struct {
	rule     int
	position int
}

type jitParserMemoEntry struct {
	value    Scmer
	position uint32
	success  bool
	active   bool
	head     *jitParserLeftRecursionHead
}

type jitParserLeftRecursionHead struct {
	rule     int
	position int
	involved []uint64
	evaluate []uint64
}

func jitParserRuleSetHas(set []uint64, rule int) bool {
	word := rule >> 6
	return word < len(set) && set[word]&(uint64(1)<<uint(rule&63)) != 0
}

func jitParserRuleSetAdd(set []uint64, rule int) {
	set[rule>>6] |= uint64(1) << uint(rule&63)
}

func jitParserRuleSetDelete(set []uint64, rule int) {
	set[rule>>6] &^= uint64(1) << uint(rule&63)
}

func (head *jitParserLeftRecursionHead) resetEvaluation() {
	copy(head.evaluate, head.involved)
}

type jitParserMutation struct {
	index int
	old   Scmer
}

type jitParserCheckpoint struct {
	position    int
	valueLen    int
	mutationLen int
	markLen     int
	positionLen int
}

// jitParserState is one pooled workspace for a complete parser invocation.
// Control and backtracking are explicit, so recursive grammars do not grow a
// Go or native call stack and parser nodes allocate no intermediate objects.
type jitParserState struct {
	program     *jitParserProgram
	frames      []jitParserCallFrame
	values      []Scmer
	bindings    []Scmer
	mutations   []jitParserMutation
	checkpoints []jitParserCheckpoint
	marks       []int
	positions   []int
	memoOffsets []uint32
	memoRules   []uint32
	memoEntries []jitParserMemoEntry
	heads       []*jitParserLeftRecursionHead
	farthest    int
	expected    []string
}

func jitParserMemoEntryCapacity(inputLength int) int {
	if inputLength <= jitParserLargeInputBytes {
		return 0
	}
	if inputLength >= jitParserMemoPreallocateLimit/jitParserMemoEntriesPerByteHint {
		return jitParserMemoPreallocateLimit
	}
	return inputLength * jitParserMemoEntriesPerByteHint
}

func (program *jitParserProgram) acquireState(inputLength int) *jitParserState {
	state := program.pool.Get().(*jitParserState)
	state.program = program
	state.frames = state.frames[:0]
	state.values = state.values[:0]
	state.bindings = state.bindings[:0]
	state.mutations = state.mutations[:0]
	state.checkpoints = state.checkpoints[:0]
	state.marks = state.marks[:0]
	state.positions = state.positions[:0]
	memoCapacity := jitParserMemoEntryCapacity(inputLength)
	if cap(state.memoEntries) < memoCapacity {
		state.memoEntries = make([]jitParserMemoEntry, 0, memoCapacity)
	} else {
		state.memoEntries = state.memoEntries[:0]
	}
	memoRuleCapacity := 0
	if inputLength > jitParserLargeInputBytes {
		if inputLength >= jitParserMemoRulePreallocateLimit/jitParserMemoRuleSlotsPerByteHint {
			memoRuleCapacity = jitParserMemoRulePreallocateLimit
		} else {
			memoRuleCapacity = inputLength * jitParserMemoRuleSlotsPerByteHint
		}
	}
	if cap(state.memoRules) < memoRuleCapacity {
		state.memoRules = make([]uint32, 0, memoRuleCapacity)
	} else {
		state.memoRules = state.memoRules[:0]
	}
	state.farthest = -1
	state.expected = state.expected[:0]
	if cap(state.heads) < inputLength+1 {
		state.heads = make([]*jitParserLeftRecursionHead, inputLength+1)
	} else {
		state.heads = state.heads[:inputLength+1]
		clear(state.heads)
	}
	if cap(state.memoOffsets) < inputLength+1 {
		state.memoOffsets = make([]uint32, inputLength+1)
	} else {
		state.memoOffsets = state.memoOffsets[:inputLength+1]
		clear(state.memoOffsets)
	}
	if cap(state.frames) < inputLength+8 {
		state.frames = make([]jitParserCallFrame, 0, inputLength+8)
	}
	return state
}

func (program *jitParserProgram) releaseState(state *jitParserState) {
	for index := range state.values {
		state.values[index] = NewNil()
	}
	for index := range state.bindings {
		state.bindings[index] = NewNil()
	}
	for index := range state.mutations {
		state.mutations[index].old = NewNil()
	}
	if cap(state.memoEntries) > jitParserRetainedMemoEntryCapacity {
		state.memoEntries = nil
		state.memoOffsets = nil
		state.memoRules = nil
	} else {
		clear(state.memoEntries)
		state.memoEntries = state.memoEntries[:0]
		clear(state.memoRules)
		state.memoRules = state.memoRules[:0]
		clear(state.memoOffsets)
	}
	state.program = nil
	program.pool.Put(state)
}

func (state *jitParserState) memoGet(key jitParserMemoKey) (jitParserMemoEntry, bool) {
	if key.position < 0 || key.position >= len(state.memoOffsets) || key.rule < 0 || key.rule >= len(state.program.rules) {
		return jitParserMemoEntry{}, false
	}
	denseRule := int(state.program.memoRuleIndex[key.rule])
	if denseRule < 0 {
		return jitParserMemoEntry{}, false
	}
	offset := state.memoOffsets[key.position]
	if offset == 0 {
		return jitParserMemoEntry{}, false
	}
	entryIndex := state.memoRules[int(offset)-1+denseRule]
	if entryIndex == 0 {
		return jitParserMemoEntry{}, false
	}
	return state.memoEntries[entryIndex-1], true
}

func (state *jitParserState) memoSet(key jitParserMemoKey, entry jitParserMemoEntry) {
	if key.position < 0 || key.position >= len(state.memoOffsets) || key.rule < 0 || key.rule >= len(state.program.rules) {
		panic("jit: parser memo key outside program")
	}
	denseRule := int(state.program.memoRuleIndex[key.rule])
	if denseRule < 0 {
		panic("jit: lexical parser rule cannot be memoized")
	}
	offset := state.memoOffsets[key.position]
	if offset == 0 {
		base := len(state.memoRules)
		state.memoRules = slices.Grow(state.memoRules, state.program.memoRuleCount)
		state.memoRules = state.memoRules[:base+state.program.memoRuleCount]
		clear(state.memoRules[base:])
		offset = uint32(base + 1)
		state.memoOffsets[key.position] = offset
	}
	index := int(offset) - 1 + denseRule
	if entryIndex := state.memoRules[index]; entryIndex != 0 {
		state.memoEntries[entryIndex-1] = entry
		return
	}
	state.memoEntries = append(state.memoEntries, entry)
	state.memoRules[index] = uint32(len(state.memoEntries))
}

func jitParserStateValue(value Scmer) *jitParserState {
	if value.GetTag() != tagAny {
		panic("jit: parser state is not opaque")
	}
	state, ok := value.Any().(*jitParserState)
	if !ok || state == nil {
		panic("jit: invalid parser state")
	}
	return state
}

func jitParserPushValue(stateValue, value Scmer) Scmer {
	state := jitParserStateValue(stateValue)
	state.values = append(state.values, value)
	return value
}

func jitParserDiscardValue(stateValue Scmer) Scmer {
	state := jitParserStateValue(stateValue)
	if len(state.values) == 0 {
		panic("jit: parser value stack underflow")
	}
	state.values[len(state.values)-1] = NewNil()
	state.values = state.values[:len(state.values)-1]
	return NewNil()
}

func jitParserCaptureValue(stateValue, inputValue, startValue, endValue Scmer) Scmer {
	state := jitParserStateValue(stateValue)
	if len(state.values) == 0 {
		panic("jit: parser capture without a value")
	}
	input := inputValue.String()
	start, end := int(startValue.Int()), int(endValue.Int())
	if start < 0 || end < start || end > len(input) {
		panic("jit: invalid parser capture range")
	}
	result := NewSlice([]Scmer{NewString(input[start:end]), state.values[len(state.values)-1]})
	state.values[len(state.values)-1] = result
	return result
}

func jitParserEnterRule(stateValue, ruleValue, successValue, failureValue, positionValue Scmer) Scmer {
	state := jitParserStateValue(stateValue)
	ruleID := int(ruleValue.Int())
	if ruleID < 0 || ruleID >= len(state.program.rules) {
		panic("jit: parser rule index out of range")
	}
	rule := &state.program.rules[ruleID]
	frame := jitParserCallFrame{
		rule: ruleID, success: int(successValue.Int()), failure: int(failureValue.Int()),
		position: int(positionValue.Int()), valueBase: len(state.values), bindingBase: len(state.bindings),
		mutationBase: len(state.mutations), memoize: rule.lexicalParent < 0,
	}
	state.frames = append(state.frames, frame)
	for range rule.bindings {
		state.bindings = append(state.bindings, NewNil())
	}
	return NewNil()
}

func jitParserPushRuleFrame(state *jitParserState, ruleID, success, failure, position int, memoInitial, transient bool) {
	rule := &state.program.rules[ruleID]
	state.frames = append(state.frames, jitParserCallFrame{
		rule: ruleID, success: success, failure: failure, position: position,
		valueBase: len(state.values), bindingBase: len(state.bindings), mutationBase: len(state.mutations),
		memoize: rule.lexicalParent < 0, memoInitial: memoInitial, transient: transient,
	})
	for range rule.bindings {
		state.bindings = append(state.bindings, NewNil())
	}
}

func jitParserSetupLeftRecursion(state *jitParserState, key jitParserMemoKey) jitParserMemoEntry {
	memo, _ := state.memoGet(key)
	head := memo.head
	if head == nil {
		words := (len(state.program.rules) + 63) >> 6
		head = &jitParserLeftRecursionHead{
			rule: key.rule, position: key.position,
			involved: make([]uint64, words), evaluate: make([]uint64, words),
		}
		memo.head = head
		state.memoSet(key, memo)
	}
	for index := len(state.frames) - 1; index >= 0; index-- {
		frame := state.frames[index]
		if !frame.memoInitial {
			continue
		}
		frameKey := jitParserMemoKey{rule: frame.rule, position: frame.position}
		entry, _ := state.memoGet(frameKey)
		if entry.head == head {
			break
		}
		entry.head = head
		state.memoSet(frameKey, entry)
		if frameKey != key {
			jitParserRuleSetAdd(head.involved, frame.rule)
		}
	}
	memo, _ = state.memoGet(key)
	return memo
}

func jitParserResetRuleFrame(state *jitParserState, frame jitParserCallFrame, restoreMutations bool) {
	if restoreMutations {
		for index := len(state.mutations) - 1; index >= frame.mutationBase; index-- {
			mutation := state.mutations[index]
			state.bindings[mutation.index] = mutation.old
		}
	}
	for index := frame.valueBase; index < len(state.values); index++ {
		state.values[index] = NewNil()
	}
	for index := frame.bindingBase; index < len(state.bindings); index++ {
		state.bindings[index] = NewNil()
	}
	for index := frame.mutationBase; index < len(state.mutations); index++ {
		state.mutations[index].old = NewNil()
	}
	state.values = state.values[:frame.valueBase]
	state.bindings = state.bindings[:frame.bindingBase]
	state.mutations = state.mutations[:frame.mutationBase]
}

func jitParserDeliverRuleResult(state *jitParserState, frame jitParserCallFrame, position int, value Scmer, success bool, restoreMutations bool) (int64, int64, bool) {
	jitParserResetRuleFrame(state, frame, restoreMutations)
	state.frames = state.frames[:len(state.frames)-1]
	if success {
		state.values = append(state.values, value)
		return int64(frame.success), int64(position), false
	}
	return int64(frame.failure), int64(frame.position), false
}

// jitParserCompleteRule implements the seed-growing packrat algorithm for
// direct and indirect left recursion. The boolean result requests a direct
// jump back to the returned rule rather than continuation dispatch.
func jitParserCompleteRule(state *jitParserState, position int, value Scmer, success bool) (int64, int64, bool) {
	if len(state.frames) == 0 {
		panic("jit: parser rule stack underflow")
	}
	frame := state.frames[len(state.frames)-1]
	if !success {
		position = frame.position
		value = NewNil()
	}
	if frame.transient {
		key := jitParserMemoKey{rule: frame.rule, position: frame.position}
		memo, _ := state.memoGet(key)
		memo.value, memo.position, memo.success = value, uint32(position), success
		memo.active = false
		state.memoSet(key, memo)
		return jitParserDeliverRuleResult(state, frame, position, value, success, false)
	}
	if !frame.memoize {
		return jitParserDeliverRuleResult(state, frame, position, value, success, false)
	}
	key := jitParserMemoKey{rule: frame.rule, position: frame.position}
	memo, _ := state.memoGet(key)
	if frame.growing {
		if success && position > int(memo.position) {
			memo.value, memo.position, memo.success = value, uint32(position), true
			state.memoSet(key, memo)
			jitParserResetRuleFrame(state, frame, true)
			for range state.program.rules[frame.rule].bindings {
				state.bindings = append(state.bindings, NewNil())
			}
			memo.head.resetEvaluation()
			return int64(frame.rule), int64(frame.position), true
		}
		state.heads[frame.position] = nil
		return jitParserDeliverRuleResult(state, frame, int(memo.position), memo.value, memo.success, true)
	}
	memo.value, memo.position, memo.success = value, uint32(position), success
	if memo.head == nil {
		memo.active = false
		state.memoSet(key, memo)
		return jitParserDeliverRuleResult(state, frame, position, value, success, false)
	}
	state.memoSet(key, memo)
	if memo.head.rule != frame.rule {
		return jitParserDeliverRuleResult(state, frame, position, value, success, false)
	}
	memo.active = false
	state.memoSet(key, memo)
	if !success {
		return jitParserDeliverRuleResult(state, frame, position, value, false, true)
	}
	state.heads[frame.position] = memo.head
	state.frames[len(state.frames)-1].growing = true
	jitParserResetRuleFrame(state, frame, true)
	for range state.program.rules[frame.rule].bindings {
		state.bindings = append(state.bindings, NewNil())
	}
	memo.head.resetEvaluation()
	return int64(frame.rule), int64(frame.position), true
}

func jitParserBindingValueNative(stateValue Scmer, binding int64) Scmer {
	state := jitParserStateValue(stateValue)
	if len(state.frames) == 0 {
		panic("jit: parser binding outside a rule")
	}
	frame := state.frames[len(state.frames)-1]
	index := frame.bindingBase + int(binding)
	if index < frame.bindingBase || index >= len(state.bindings) {
		panic("jit: parser binding index outside a rule")
	}
	return state.bindings[index]
}

func jitParserBindingValueForRuleNative(stateValue Scmer, ruleID, binding int64) Scmer {
	state := jitParserStateValue(stateValue)
	for index := len(state.frames) - 1; index >= 0; index-- {
		frame := state.frames[index]
		if int64(frame.rule) != ruleID {
			continue
		}
		bindingIndex := frame.bindingBase + int(binding)
		if binding < 0 || bindingIndex >= len(state.bindings) {
			panic("jit: parser binding index outside its lexical rule")
		}
		return state.bindings[bindingIndex]
	}
	panic("jit: parser lexical rule is not active")
}

func jitParserRuleValueNative(stateValue Scmer) Scmer {
	state := jitParserStateValue(stateValue)
	if len(state.frames) == 0 {
		panic("jit: parser value outside a rule")
	}
	frame := state.frames[len(state.frames)-1]
	if len(state.values) <= frame.valueBase {
		return NewNil()
	}
	return state.values[len(state.values)-1]
}

func jitParserReturnRuleValueNative(stateValue Scmer, position int64, value Scmer) (int64, int64, bool) {
	state := jitParserStateValue(stateValue)
	// Parser actions cross from the nested generated expression into the parser
	// machine state. Lists must own Go-managed backing storage at that boundary;
	// an inlined variadic producer may otherwise describe its transient call
	// array even though the Scmer header itself has already been materialized.
	if value.GetTag() == tagSlice {
		value = NewSlice(append([]Scmer(nil), value.Slice()...))
	}
	return jitParserCompleteRule(state, int(position), value, true)
}

func jitParserAcquireStateNative(programValue, input Scmer) Scmer {
	program, ok := programValue.Any().(*jitParserProgram)
	if !ok || program == nil {
		panic("jit: invalid parser program")
	}
	return NewAny(program.acquireState(len(input.String())))
}

func jitParserReleaseStateNative(programValue, stateValue Scmer) {
	program, ok := programValue.Any().(*jitParserProgram)
	if !ok || program == nil {
		panic("jit: invalid parser program")
	}
	program.releaseState(jitParserStateValue(stateValue))
}

func jitParserFinish(stateValue Scmer) Scmer {
	state := jitParserStateValue(stateValue)
	if len(state.values) != 1 || len(state.frames) != 0 {
		panic("jit: parser finished with an invalid machine stack")
	}
	return state.values[0]
}

func jitParserBindValue(stateValue Scmer, binding Scmer) Scmer {
	state := jitParserStateValue(stateValue)
	if len(state.frames) == 0 || len(state.values) == 0 {
		panic("jit: parser binding outside a rule")
	}
	frame := &state.frames[len(state.frames)-1]
	index := frame.bindingBase + int(binding.Int())
	if index < frame.bindingBase || index >= len(state.bindings) {
		panic("jit: parser binding index outside a rule")
	}
	state.mutations = append(state.mutations, jitParserMutation{index: index, old: state.bindings[index]})
	state.bindings[index] = state.values[len(state.values)-1]
	return state.values[len(state.values)-1]
}

func jitParserPushCheckpoint(stateValue, position Scmer) Scmer {
	state := jitParserStateValue(stateValue)
	state.checkpoints = append(state.checkpoints, jitParserCheckpoint{
		position: int(position.Int()), valueLen: len(state.values), mutationLen: len(state.mutations), markLen: len(state.marks),
		positionLen: len(state.positions),
	})
	return position
}

func jitParserRestoreCheckpoint(stateValue Scmer) Scmer {
	state := jitParserStateValue(stateValue)
	if len(state.checkpoints) == 0 {
		panic("jit: parser checkpoint underflow")
	}
	checkpoint := state.checkpoints[len(state.checkpoints)-1]
	state.checkpoints = state.checkpoints[:len(state.checkpoints)-1]
	for index := len(state.mutations) - 1; index >= checkpoint.mutationLen; index-- {
		mutation := state.mutations[index]
		state.bindings[mutation.index] = mutation.old
	}
	state.mutations = state.mutations[:checkpoint.mutationLen]
	state.values = state.values[:checkpoint.valueLen]
	state.marks = state.marks[:checkpoint.markLen]
	state.positions = state.positions[:checkpoint.positionLen]
	return NewInt(int64(checkpoint.position))
}

func jitParserCommitCheckpoint(stateValue Scmer) Scmer {
	state := jitParserStateValue(stateValue)
	if len(state.checkpoints) == 0 {
		panic("jit: parser checkpoint underflow")
	}
	state.checkpoints = state.checkpoints[:len(state.checkpoints)-1]
	return NewNil()
}

func jitParserCheckpointPosition(stateValue Scmer) Scmer {
	state := jitParserStateValue(stateValue)
	if len(state.checkpoints) == 0 {
		panic("jit: parser checkpoint underflow")
	}
	return NewInt(int64(state.checkpoints[len(state.checkpoints)-1].position))
}

func jitParserCommitProgress(stateValue, positionValue Scmer) Scmer {
	state := jitParserStateValue(stateValue)
	if len(state.checkpoints) == 0 {
		panic("jit: parser checkpoint underflow")
	}
	checkpoint := state.checkpoints[len(state.checkpoints)-1]
	if checkpoint.position == int(positionValue.Int()) {
		jitParserRestoreCheckpoint(stateValue)
		return NewBool(false)
	}
	jitParserCommitCheckpoint(stateValue)
	return NewBool(true)
}

func jitParserPushPosition(stateValue, positionValue Scmer) Scmer {
	state := jitParserStateValue(stateValue)
	state.positions = append(state.positions, int(positionValue.Int()))
	return NewNil()
}

func jitParserPopPosition(stateValue Scmer) Scmer {
	state := jitParserStateValue(stateValue)
	if len(state.positions) == 0 {
		panic("jit: parser position stack underflow")
	}
	position := state.positions[len(state.positions)-1]
	state.positions = state.positions[:len(state.positions)-1]
	return NewInt(int64(position))
}

func jitParserPushMark(stateValue Scmer) Scmer {
	state := jitParserStateValue(stateValue)
	state.marks = append(state.marks, len(state.values))
	return NewNil()
}

func jitParserMergeMark(stateValue, ignoreValue Scmer) Scmer {
	state := jitParserStateValue(stateValue)
	if len(state.marks) == 0 {
		panic("jit: parser value mark underflow")
	}
	base := state.marks[len(state.marks)-1]
	state.marks = state.marks[:len(state.marks)-1]
	var result Scmer
	if ignoreValue.Bool() {
		result = NewNil()
	} else {
		values := make([]Scmer, len(state.values)-base)
		copy(values, state.values[base:])
		result = NewSlice(values)
	}
	state.values = state.values[:base]
	state.values = append(state.values, result)
	return result
}

func jitParserRecordFailure(stateValue, position, expected Scmer) Scmer {
	state := jitParserStateValue(stateValue)
	pos := int(position.Int())
	if pos > state.farthest {
		state.farthest = pos
		state.expected = state.expected[:0]
	}
	if pos == state.farthest && len(state.expected) < 8 {
		state.expected = append(state.expected, expected.String())
	}
	return NewNil()
}

func jitParserPanic(stateValue, input Scmer) Scmer {
	state := jitParserStateValue(stateValue)
	position := state.farthest
	if position < 0 {
		position = 0
	}
	panic(fmt.Sprintf("parser failed at byte %d of %d; expected %v", position, len(input.String()), state.expected))
}

// Native wrappers use scalar Go ABI words for parser indices and positions.
// Scmer arguments remain pairs. Keeping this boundary explicit avoids boxing
// control data merely to cross an emitted helper call.
func jitParserEnterRuleNative(state *jitParserState, ruleValue, success, failure, position int64) (int64, int64, bool) {
	rule := int(ruleValue)
	if rule < 0 || rule >= len(state.program.rules) {
		panic("jit: parser rule index out of range")
	}
	if state.program.rules[rule].lexicalParent < 0 {
		key := jitParserMemoKey{rule: rule, position: int(position)}
		memo, exists := state.memoGet(key)
		var head *jitParserLeftRecursionHead
		if int(position) >= 0 && int(position) < len(state.heads) {
			head = state.heads[int(position)]
		}
		if head != nil {
			if jitParserRuleSetHas(head.evaluate, rule) {
				jitParserRuleSetDelete(head.evaluate, rule)
				jitParserPushRuleFrame(state, rule, int(success), int(failure), int(position), false, true)
				return 0, position, false
			}
			if !exists && rule != head.rule && !jitParserRuleSetHas(head.involved, rule) {
				return failure, position, true
			}
		}
		if exists {
			if memo.active {
				memo = jitParserSetupLeftRecursion(state, key)
			}
			if memo.success {
				state.values = append(state.values, memo.value)
				return success, int64(memo.position), true
			}
			return failure, int64(memo.position), true
		}
		if position > int64(^uint32(0)) {
			panic("jit: parser input exceeds supported position range")
		}
		state.memoSet(key, jitParserMemoEntry{position: uint32(position), active: true})
		jitParserPushRuleFrame(state, rule, int(success), int(failure), int(position), true, false)
		return 0, position, false
	}
	jitParserPushRuleFrame(state, rule, int(success), int(failure), int(position), false, false)
	return 0, position, false
}

func jitParserReturnRuleNative(stateValue Scmer, position int64, success bool) (int64, int64, bool) {
	state := jitParserStateValue(stateValue)
	value := NewNil()
	if success {
		if len(state.frames) == 0 {
			panic("jit: parser rule stack underflow")
		}
		frame := state.frames[len(state.frames)-1]
		if len(state.values) > frame.valueBase {
			value = state.values[len(state.values)-1]
		}
		rule := &state.program.rules[frame.rule]
		if !rule.generator.IsNil() {
			args := state.bindings[frame.bindingBase : frame.bindingBase+len(rule.bindings)]
			value = Apply(rule.action, args...)
		}
	}
	return jitParserCompleteRule(state, int(position), value, success)
}

func jitParserBindValueNative(state Scmer, binding int64) {
	jitParserBindValue(state, NewInt(binding))
}

func jitParserPushCheckpointNative(state Scmer, position int64) {
	jitParserPushCheckpoint(state, NewInt(position))
}

func jitParserRestoreCheckpointNative(state Scmer) int64 {
	return jitParserRestoreCheckpoint(state).Int()
}

func jitParserCommitCheckpointNative(state Scmer) {
	jitParserCommitCheckpoint(state)
}

func jitParserCheckpointPositionNative(state Scmer) int64 {
	return jitParserCheckpointPosition(state).Int()
}

func jitParserCommitProgressNative(state Scmer, position int64) bool {
	return jitParserCommitProgress(state, NewInt(position)).Bool()
}

func jitParserPushPositionNative(state Scmer, position int64) {
	jitParserPushPosition(state, NewInt(position))
}

func jitParserPopPositionNative(state Scmer) int64 {
	return jitParserPopPosition(state).Int()
}

func jitParserMergeMarkNative(state Scmer, ignore bool) {
	jitParserMergeMark(state, NewBool(ignore))
}

func jitParserCaptureValueNative(state, input Scmer, start, end int64) {
	jitParserCaptureValue(state, input, NewInt(start), NewInt(end))
}

func jitParserRecordFailureNative(state Scmer, position int64, expected Scmer) {
	jitParserRecordFailure(state, NewInt(position), expected)
}

func jitParserPushValueNative(state, value Scmer) {
	jitParserPushValue(state, value)
}

func jitParserDiscardValueNative(state Scmer) {
	jitParserDiscardValue(state)
}

func jitParserPushMarkNative(state Scmer) {
	jitParserPushMark(state)
}

func jitParserAtBreakNative(input Scmer, position int64) bool {
	text := input.String()
	pos := int(position)
	if pos <= 0 || pos >= len(text) {
		return true
	}
	previous, _ := utf8.DecodeLastRuneInString(text[:pos])
	current, _ := utf8.DecodeRuneInString(text[pos:])
	isWord := func(value rune) bool { return unicode.In(value, unicode.N, unicode.L, unicode.Pc) }
	return !isWord(previous) || !isWord(current)
}
