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
// TODO(optimizer): match with '(a b c) pattern (= (list a b c)) does not work
// as destructuring in the /scm endpoint but works inside queryplan.scm when
// the optimizer compiles lambdas. In /scm: (match (list 1 2 3) '(a b c) a)
// returns nil. Workaround: use (cons a (cons b (cons c _))) patterns outside
// optimizer-compiled contexts.
package scm

import "regexp"
import "strconv"
import "strings"
import "time"

var SettingsHaveGoodBacktraces bool

func procBodyUsesNamedParam(body Scmer, named map[Symbol]struct{}) bool {
	if len(named) == 0 {
		return false
	}
	if stripped, ok := scmerStripSourceInfo(body); ok {
		body = stripped
	}
	if body.IsSymbol() {
		_, ok := named[mustSymbol(body)]
		return ok
	}
	if !body.IsSlice() {
		return false
	}
	items := body.Slice()
	if len(items) > 0 && items[0].IsSymbol() && items[0].String() == "quote" {
		return false
	}
	for _, item := range items {
		if procBodyUsesNamedParam(item, named) {
			return true
		}
	}
	return false
}

func procCanUseNumberedOnly(params, body Scmer, numVars int) bool {
	if numVars == 0 {
		return false
	}
	if stripped, ok := scmerStripSourceInfo(params); ok {
		params = stripped
	}
	named := make(map[Symbol]struct{})
	if params.IsSlice() {
		for _, param := range params.Slice() {
			if stripped, ok := scmerStripSourceInfo(param); ok {
				param = stripped
			}
			if param.IsSymbol() && !param.SymbolEquals("_") {
				named[mustSymbol(param)] = struct{}{}
			}
		}
	} else if params.IsSymbol() && !params.SymbolEquals("_") {
		named[mustSymbol(params)] = struct{}{}
	}
	return !procBodyUsesNamedParam(body, named)
}

// to optimize lambdas serially; the resulting function MUST NEVER run on multiple threads simultanously since state is reduced to save mallocs
func OptimizeProcToSerialFunction(val Scmer) func(...Scmer) Scmer {
	/* API contract:
	- the returned func must only be called with the correct number of declared parameters
	- thus we will perform no boundary checks
	- we enclose and share the environment over multiple runs, so the function must not be called simultaneously
	- for performance reason, we put as much checks and allocations out of the returned function and into our closure
	- TODO: we want to hook up the JIT here to produce some machine code for hotpaths
	*/
	if val.IsNil() {
		return func(...Scmer) Scmer { return NewNil() }
	}
	if val.GetTag() == tagFunc {
		return val.Func()
	}
	if val.GetTag() == tagAny {
		if fn, ok := val.Any().(func(...Scmer) Scmer); ok {
			return fn
		}
	}

	var proc *Proc
	switch val.GetTag() {
	case tagProc:
		proc = val.Proc()
	}
	if proc == nil {
		// Not a lambda/proc: treat as constant value and return it regardless of args.
		// This avoids attempting to Apply() non-callables like true/0/"x" which would panic.
		captured := val
		return func(args ...Scmer) Scmer { return captured }
	}
	p := *proc

	// constant body
	switch p.Body.GetTag() {
	case tagNil, tagBool, tagInt, tagFloat, tagString:
		constant := p.Body
		return func(...Scmer) Scmer { return constant }
	}

	// Fast-path: lambda body is exactly one of its parameters -> return that arg directly
	{
		body := p.Body
		if stripped, ok := scmerStripSourceInfo(body); ok {
			body = stripped
		}
		// numbered locals: (var i)
		if body.IsNthLocalVar() {
			idx := int(body.NthLocalVar())
			return func(args ...Scmer) Scmer {
				return args[idx]
			}
		}
		// named params: find exact parameter symbol match
		params := p.Params
		if stripped, ok := scmerStripSourceInfo(params); ok {
			params = stripped
		}
		if body.IsSymbol() && params.IsSlice() {
			parms := params.Slice()
			bSym := mustSymbol(body)
			for i, ps := range parms {
				if stripped, ok := scmerStripSourceInfo(ps); ok {
					ps = stripped
				}
				if ps.IsSymbol() && mustSymbol(ps) == bSym {
					idx := i
					return func(args ...Scmer) Scmer {
						return args[idx]
					}
				}
			}
		}
	}

	numVars := p.NumVars
	if required := requiredNumberedSlots(p.Body); required > numVars {
		numVars = required
	}
	var vars Vars
	en := &Env{Vars: vars, VarsNumbered: make([]Scmer, numVars), Outer: p.En, Nodefine: false}
	params := p.Params
	if stripped, ok := scmerStripSourceInfo(params); ok {
		params = stripped
	}
	if params.IsSlice() {
		paramSlice := params.Slice()
		if numVars > 0 {
			bindNamed := false
			if !p.NumberedOnly {
				named := make(map[Symbol]struct{}, len(paramSlice))
				for i, param := range paramSlice {
					if i >= numVars {
						break
					}
					if stripped, ok := scmerStripSourceInfo(param); ok {
						param = stripped
					}
					if param.IsSymbol() && !param.SymbolEquals("_") {
						named[mustSymbol(param)] = struct{}{}
					}
				}
				bindNamed = procBodyUsesNamedParam(p.Body, named)
				if bindNamed {
					vars = make(Vars, len(named))
					en.Vars = vars
				}
			}
			return func(args ...Scmer) Scmer {
				for i := 0; i < numVars; i++ {
					if i < len(args) {
						en.VarsNumbered[i] = args[i]
					} else {
						en.VarsNumbered[i] = NewNil()
					}
				}
				if bindNamed {
					for i, param := range paramSlice {
						if stripped, ok := scmerStripSourceInfo(param); ok {
							param = stripped
						}
						if !param.IsSymbol() || param.SymbolEquals("_") {
							continue
						}
						sym := mustSymbol(param)
						if i < len(args) {
							en.Vars[sym] = args[i]
						} else {
							en.Vars[sym] = NewNil()
						}
					}
				}
				return Eval(p.Body, en)
			}
		}
		vars = make(Vars, len(paramSlice))
		en.Vars = vars
		return func(args ...Scmer) Scmer {
			for i, param := range paramSlice {
				if stripped, ok := scmerStripSourceInfo(param); ok {
					param = stripped
				}
				if !param.IsSymbol() || param.SymbolEquals("_") {
					continue
				}
				sym := mustSymbol(param)
				if i < len(args) {
					en.Vars[sym] = args[i]
				} else {
					en.Vars[sym] = NewNil()
				}
			}
			return Eval(p.Body, en)
		}
	}
	if params.IsSymbol() {
		sym := mustSymbol(params)
		if p.NumVars > 0 {
			bindNamed := false
			if !p.NumberedOnly {
				bindNamed = procBodyUsesNamedParam(p.Body, map[Symbol]struct{}{sym: {}})
				if bindNamed {
					vars = make(Vars, 1)
					en.Vars = vars
				}
			}
			return func(args ...Scmer) Scmer {
				argsList := NewSlice(args)
				en.VarsNumbered[0] = argsList
				if bindNamed {
					en.Vars[sym] = argsList
				}
				return Eval(p.Body, en)
			}
		}
		vars = make(Vars, 1)
		en.Vars = vars
		return func(args ...Scmer) Scmer {
			en.Vars[sym] = NewSlice(args)
			return Eval(p.Body, en)
		}
	}
	return func(args ...Scmer) Scmer {
		return Eval(p.Body, en)
	}
}

// do preprocessing and optimization (Optimize is allowed to edit the value in-place)
func Optimize(val Scmer, env *Env) Scmer {
	result, _ := OptimizeWithStats(val, env)
	return result
}

// OptimizerStats exposes compile work without coupling callers to optimizer
// internals. All counters cover one top-level optimization.
type OptimizerStats struct {
	CompileNS        int64
	InputNodes       int
	OutputNodes      int
	Rewrites         int
	RejectedRewrites int
	BudgetRemaining  int
	CallbackAnalyses int
	CallbackClones   int
}

type optimizerRewriteState struct {
	remainingBudget  int
	inputNodes       int
	rewrites         int
	rejected         int
	callbackAnalyses int
	callbackClones   int
	active           map[string]bool
	seen             map[uint64]bool
}

const defaultOptimizerRewriteBudget = 64

// OptimizeWithStats returns both optimized code and bounded-work telemetry.
func OptimizeWithStats(val Scmer, env *Env) (Scmer, OptimizerStats) {
	started := time.Now()
	ome := newOptimizerMetainfo()
	ome.rewrite.inputNodes = optimizerNodeCount(val)
	v, _ := OptimizeEx(val, env, &ome, true)
	return v, OptimizerStats{
		CompileNS:        time.Since(started).Nanoseconds(),
		InputNodes:       ome.rewrite.inputNodes,
		OutputNodes:      optimizerNodeCount(v),
		Rewrites:         ome.rewrite.rewrites,
		RejectedRewrites: ome.rewrite.rejected,
		BudgetRemaining:  ome.rewrite.remainingBudget,
		CallbackAnalyses: ome.rewrite.callbackAnalyses,
		CallbackClones:   ome.rewrite.callbackClones,
	}
}

type optimizerMetainfo struct {
	variableReplacement   map[Symbol]Scmer
	variableTypes         map[Symbol]*TypeDescriptor
	numberedTypes         map[NthLocalVar]*TypeDescriptor
	setBlacklist          []Symbol
	nextSlot              *int // pointer to lambda's slot counter; nil outside lambda
	pendingCallbackParams []*TypeDescriptor
	pendingCallbackReturn *TypeDescriptor // structured escape information for the next lambda result
	loopDepth             int             // >0 inside scan/reduce callbacks; prevents hoisted defines from being inlined back into loops
	lambdaDepth           int             // >0 while optimizing a lambda body; keeps local definitions out of Env hints
	beginDepth            int             // >0 in lexical begin scopes; their definitions do not reach the caller Env
	inlineDepth           int
	inlineStack           map[Symbol]bool
	rewrite               *optimizerRewriteState
}

func newOptimizerMetainfo() (result optimizerMetainfo) {
	result.variableReplacement = make(map[Symbol]Scmer)
	result.variableTypes = make(map[Symbol]*TypeDescriptor)
	result.numberedTypes = make(map[NthLocalVar]*TypeDescriptor)
	result.rewrite = &optimizerRewriteState{
		remainingBudget: defaultOptimizerRewriteBudget,
		active:          make(map[string]bool),
		seen:            make(map[uint64]bool),
	}
	return
}

func optimizerNodeCount(expr Scmer) int {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	items, ok := scmerSlice(expr)
	if !ok {
		return 1
	}
	count := 1
	for _, item := range items {
		count += optimizerNodeCount(item)
	}
	return count
}

func rewriteNoEscapeListReturn(expr Scmer, flow *TypeDescriptor, nextSlot *int) Scmer {
	items, ok := scmerSlice(expr)
	if !ok || len(items) < 2 || !scmerIsSymbol(items[0], "list") || nextSlot == nil || flow == nil {
		return expr
	}
	rewrittenItems := append([]Scmer(nil), items...)
	for i := 1; i < len(rewrittenItems); i++ {
		if child := flow.Keys[strconv.Itoa(i-1)]; child != nil {
			rewrittenItems[i] = rewriteNoEscapeListReturn(rewrittenItems[i], child, nextSlot)
		}
	}
	if !flow.NoEscape {
		return NewSlice(rewrittenItems)
	}
	count := len(items) - 1
	start := *nextSlot
	*nextSlot += count
	rewritten := make([]Scmer, 0, count+3)
	rewritten = append(rewritten, NewSymbol("!list"), NewNthLocalVar(NthLocalVar(start)), NewInt(int64(count)))
	rewritten = append(rewritten, rewrittenItems[1:]...)
	return NewSlice(rewritten)
}

func optimizerLambdaParts(lambda Scmer) ([]Scmer, Scmer, bool) {
	items, ok := scmerSlice(lambda)
	if !ok || len(items) < 3 || !scmerIsSymbol(items[0], "lambda") {
		return nil, NewNil(), false
	}
	params, ok := scmerSlice(items[1])
	if !ok {
		return nil, NewNil(), false
	}
	return params, items[2], true
}

func listConstructorElements(items []Scmer) ([]Scmer, bool) {
	if len(items) == 0 {
		return nil, false
	}
	if scmerIsSymbol(items[0], "list") {
		return items[1:], true
	}
	if scmerIsSymbol(items[0], "!list") && len(items) >= 3 {
		return items[3:], true
	}
	if decl := DeclarationForValue(items[0]); decl != nil && decl.Name == "list" {
		return items[1:], true
	}
	return nil, false
}

func trackedValuePath(expr Scmer, symbol Symbol) (string, bool) {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	if sym, ok := scmerSymbol(expr); ok {
		return "", sym == symbol
	}
	items, ok := scmerSlice(expr)
	if !ok || len(items) < 2 {
		return "", false
	}
	var child string
	switch {
	case scmerIsSymbol(items[0], "car") && len(items) == 2:
		child = "0"
	case scmerIsSymbol(items[0], "cadr") && len(items) == 2:
		child = "1"
	case scmerIsSymbol(items[0], "nth") && len(items) == 3 && items[2].IsInt():
		child = strconv.FormatInt(items[2].Int(), 10)
	case scmerIsSymbol(items[0], "get_assoc") && len(items) == 3 && items[2].IsString():
		child = String(items[2])
	default:
		return "", false
	}
	base, ok := trackedValuePath(items[1], symbol)
	if !ok {
		return "", false
	}
	if base == "" {
		return child, true
	}
	return base + "/" + child, true
}

func callParameterDescriptor(call []Scmer, argIndex int) *TypeDescriptor {
	decl := DeclarationForValue(call[0])
	if decl == nil || decl.Type == nil || len(decl.Type.Params) == 0 {
		return nil
	}
	idx := argIndex
	if idx >= len(decl.Type.Params) {
		idx = len(decl.Type.Params) - 1
	}
	if idx < 0 {
		return nil
	}
	return decl.Type.Params[idx]
}

func analyzeTrackedEscapes(expr Scmer, symbol Symbol, resultEscapes bool, escaped map[string]bool) {
	if path, ok := trackedValuePath(expr, symbol); ok {
		if resultEscapes {
			escaped[path] = true
		}
		return
	}
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	items, ok := scmerSlice(expr)
	if !ok || len(items) == 0 || scmerIsSymbol(items[0], "quote") {
		return
	}
	if scmerIsSymbol(items[0], "begin") || scmerIsSymbol(items[0], "!begin") || scmerIsSymbol(items[0], "begin_mut") {
		for i := 1; i < len(items); i++ {
			analyzeTrackedEscapes(items[i], symbol, resultEscapes && i == len(items)-1, escaped)
		}
		return
	}
	if scmerIsSymbol(items[0], "if") {
		for i := 1; i < len(items); i++ {
			analyzeTrackedEscapes(items[i], symbol, resultEscapes && i >= 2, escaped)
		}
		return
	}
	if scmerIsSymbol(items[0], "lambda") {
		return
	}
	for i := 1; i < len(items); i++ {
		param := callParameterDescriptor(items, i-1)
		argEscapes := param == nil || !param.NoEscape
		analyzeTrackedEscapes(items[i], symbol, argEscapes, escaped)
	}
}

func callbackValueFlow(expr Scmer, escaped map[string]bool, path string, parentEscapes bool) *TypeDescriptor {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	escapes := parentEscapes || escaped[path]
	td := &TypeDescriptor{Kind: "any", NoEscape: !escapes, Length: UnknownLength}
	items, ok := scmerSlice(expr)
	if !ok {
		return td
	}
	elements, ok := listConstructorElements(items)
	if !ok {
		return td
	}
	td.Kind = "list"
	td.Transfer = true
	td.Length = len(elements)
	td.Keys = make(map[string]*TypeDescriptor, len(elements))
	for i, element := range elements {
		key := strconv.Itoa(i)
		childPath := key
		if path != "" {
			childPath = path + "/" + key
		}
		td.Keys[key] = callbackValueFlow(element, escaped, childPath, escapes)
	}
	return td
}

// CallbackReturnFlow describes which portions of producer's result can use its
// lambda frame after following consumer's selected parameter through projections
// and calls. Unknown consumers conservatively retain the complete value.
func CallbackReturnFlow(producer, consumer Scmer, consumerParam int) *TypeDescriptor {
	_, producerBody, producerOK := optimizerLambdaParts(producer)
	params, consumerBody, consumerOK := optimizerLambdaParts(consumer)
	if !producerOK {
		return nil
	}
	escaped := make(map[string]bool)
	if !consumerOK || consumerParam < 0 || consumerParam >= len(params) {
		escaped[""] = true
	} else if symbol, ok := scmerSymbol(params[consumerParam]); ok {
		analyzeTrackedEscapes(consumerBody, symbol, true, escaped)
	} else {
		escaped[""] = true
	}
	return callbackValueFlow(producerBody, escaped, "", false)
}

func copyTypeDescriptor(td *TypeDescriptor) *TypeDescriptor {
	if td == nil {
		return &TypeDescriptor{Kind: "any", Length: UnknownLength}
	}
	result := *td
	return &result
}

func callbackParameterType(td *TypeDescriptor) *TypeDescriptor {
	if td == nil {
		return nil
	}
	result := *td
	result.Const = false
	if len(td.Keys) > 0 {
		result.Keys = make(map[string]*TypeDescriptor, len(td.Keys))
		for key, child := range td.Keys {
			result.Keys[key] = callbackParameterType(child)
		}
	}
	result.Element = callbackParameterType(td.Element)
	return &result
}

// CloneOptimizerExpression copies mutable AST containers so optimizer hooks can
// analyze a speculative variant without rewriting the caller's original code.
func CloneOptimizerExpression(expr Scmer) Scmer {
	if expr.IsSourceInfo() {
		source := *expr.SourceInfo()
		source.value = CloneOptimizerExpression(source.value)
		return NewSourceInfo(source)
	}
	if expr.GetTag() == tagAny {
		if source, ok := expr.Any().(SourceInfo); ok {
			source.value = CloneOptimizerExpression(source.value)
			return NewSourceInfo(source)
		}
	}
	items, ok := scmerSlice(expr)
	if !ok {
		return expr
	}
	cloned := make([]Scmer, len(items))
	for i, item := range items {
		cloned[i] = CloneOptimizerExpression(item)
	}
	return NewSlice(cloned)
}

func descriptorKey(td *TypeDescriptor, key string) *TypeDescriptor {
	if td == nil || td.Keys == nil {
		return &TypeDescriptor{Kind: "any", Length: UnknownLength}
	}
	return copyTypeDescriptor(td.Keys[key])
}

func optimizerExpressionDescriptor(expr Scmer, env *Env, ome *optimizerMetainfo) *TypeDescriptor {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	if expr.IsNthLocalVar() {
		return ome.numberedTypes[expr.NthLocalVar()]
	}
	if sym, ok := scmerSymbol(expr); ok {
		if td := ome.variableTypes[sym]; td != nil {
			return td
		}
		if ti, exists := env.optimizerProcHint(sym); exists {
			return ti.ToTypeDescriptor()
		}
		return nil
	}
	items, ok := scmerSlice(expr)
	if !ok || len(items) == 0 {
		return nil
	}
	callName := ""
	var returnType *TypeDescriptor
	if decl := DeclarationForValue(items[0]); decl != nil {
		callName = decl.Name
		if decl.Type != nil && decl.Type.Return != nil {
			returnType = decl.Type.Return
		}
	} else if sym, ok := scmerSymbol(items[0]); ok {
		callName = string(sym)
		if ti, exists := env.optimizerProcHint(sym); exists {
			return ti.ToTypeDescriptor()
		}
	}
	if len(items) < 2 {
		return returnType
	}
	key := ""
	switch {
	case callName == "car" && len(items) == 2:
		key = "0"
	case callName == "cadr" && len(items) == 2:
		key = "1"
	case callName == "nth" && len(items) == 3 && items[2].IsInt():
		key = strconv.FormatInt(items[2].Int(), 10)
	case callName == "get_assoc" && len(items) == 3 && items[2].IsString():
		key = String(items[2])
	}
	if key == "" {
		return returnType
	}
	base := optimizerExpressionDescriptor(items[1], env, ome)
	if base == nil || base.Keys == nil {
		return returnType
	}
	if projected := base.Keys[key]; projected != nil {
		return projected
	}
	return returnType
}

// IncrLoopDepth increments the loop nesting depth. Called by scan optimizer hooks
// before optimizing callback lambdas (filter/map/reduce).
func (ome *optimizerMetainfo) IncrLoopDepth() { ome.loopDepth++ }

// DecrLoopDepth decrements the loop nesting depth.
func (ome *optimizerMetainfo) DecrLoopDepth() { ome.loopDepth-- }

func (ome *optimizerMetainfo) applyPendingCallbackParams(params Scmer, child *optimizerMetainfo) {
	list, ok := scmerSlice(params)
	if !ok {
		list = []Scmer{params}
	}
	for i, param := range list {
		sym, ok := scmerSymbol(param)
		if !ok {
			continue
		}
		var td *TypeDescriptor
		if i < len(ome.pendingCallbackParams) {
			td = callbackParameterType(ome.pendingCallbackParams[i])
		}
		if td == nil {
			continue
		}
		child.variableTypes[sym] = td
		if replacement, ok := child.variableReplacement[sym]; ok && replacement.IsNthLocalVar() {
			child.numberedTypes[replacement.NthLocalVar()] = td
		}
	}
	ome.pendingCallbackParams = nil
}

// LoopDepth returns the current loop nesting depth.
func (ome *optimizerMetainfo) LoopDepth() int { return ome.loopDepth }
func (ome *optimizerMetainfo) Copy() (result optimizerMetainfo) {
	result.variableReplacement = make(map[Symbol]Scmer)
	result.variableTypes = make(map[Symbol]*TypeDescriptor)
	result.numberedTypes = make(map[NthLocalVar]*TypeDescriptor)
	for k, v := range ome.variableReplacement {
		result.variableReplacement[k] = NewSlice([]Scmer{NewSymbol("outer"), v})
	}
	result.setBlacklist = ome.setBlacklist
	result.loopDepth = ome.loopDepth
	result.lambdaDepth = ome.lambdaDepth
	result.beginDepth = ome.beginDepth
	result.inlineDepth = ome.inlineDepth
	result.inlineStack = ome.inlineStack
	result.rewrite = ome.rewrite
	// nextSlot is NOT propagated across lambda boundaries (each lambda has its own)
	return
}

// CopySharedScope is like Copy but for scopes that share VarsNumbered with
// their parent (begin, match). NthLocalVar entries are kept as-is instead of
// being wrapped in (outer ...) since they access the same VarsNumbered array.
func (ome *optimizerMetainfo) CopySharedScope() (result optimizerMetainfo) {
	result.variableReplacement = make(map[Symbol]Scmer)
	result.variableTypes = ome.variableTypes
	result.numberedTypes = ome.numberedTypes
	for k, v := range ome.variableReplacement {
		if v.IsNthLocalVar() {
			result.variableReplacement[k] = v
		} else {
			result.variableReplacement[k] = NewSlice([]Scmer{NewSymbol("outer"), v})
		}
	}
	result.setBlacklist = ome.setBlacklist
	result.nextSlot = ome.nextSlot // shared scope shares VarsNumbered
	result.loopDepth = ome.loopDepth
	result.lambdaDepth = ome.lambdaDepth
	result.beginDepth = ome.beginDepth
	result.inlineDepth = ome.inlineDepth
	result.inlineStack = ome.inlineStack
	result.rewrite = ome.rewrite
	return
}

const maxLeafInlineNodes = 24
const maxLeafInlineDepth = 8

func analyzeLeafInlineBody(expr Scmer, callee Symbol, paramCount int, refs []int, nodes *int) bool {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	*nodes++
	if *nodes > maxLeafInlineNodes {
		return false
	}
	if expr.IsNthLocalVar() {
		idx := int(expr.NthLocalVar())
		if idx < 0 || idx >= paramCount {
			return false
		}
		refs[idx]++
		return refs[idx] <= 1
	}
	items, ok := scmerSlice(expr)
	if !ok || len(items) == 0 {
		return true
	}
	if scmerIsSymbol(items[0], "quote") {
		return true
	}
	if head, ok := scmerSymbol(items[0]); ok {
		switch head {
		case callee, "lambda", "define", "set", "setN", "eval", "parser", "outer", "begin", "begin_mut", "!begin", "match", "match_mut":
			return false
		}
	}
	for _, item := range items {
		if !analyzeLeafInlineBody(item, callee, paramCount, refs, nodes) {
			return false
		}
	}
	return true
}

func substituteLeafInlineParams(expr Scmer, args []Scmer) Scmer {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	if expr.IsNthLocalVar() {
		return args[int(expr.NthLocalVar())]
	}
	items, ok := scmerSlice(expr)
	if !ok || len(items) == 0 || scmerIsSymbol(items[0], "quote") {
		return expr
	}
	rewritten := make([]Scmer, len(items))
	for i, item := range items {
		rewritten[i] = substituteLeafInlineParams(item, args)
	}
	return NewSlice(rewritten)
}

func leafInlineBindingsStable(expr Scmer, source, target *Env) bool {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	if expr.IsNthLocalVar() || !expr.IsSlice() && !expr.IsSymbol() {
		return true
	}
	if expr.IsSymbol() {
		sym := mustSymbol(expr)
		sourceOwner := source.FindRead(sym)
		targetOwner := target.FindRead(sym)
		if sourceOwner == nil || targetOwner == nil {
			return false
		}
		sourceValue, sourceOK := sourceOwner.Vars[sym]
		targetValue, targetOK := targetOwner.Vars[sym]
		return sourceOK && targetOK && sourceValue == targetValue
	}
	items := expr.Slice()
	if len(items) == 0 || scmerIsSymbol(items[0], "quote") {
		return true
	}
	for _, item := range items {
		if !leafInlineBindingsStable(item, source, target) {
			return false
		}
	}
	return true
}

func tryInlineLeafProc(v []Scmer, env *Env, ome *optimizerMetainfo, useResult bool) (Scmer, TypeInfo, bool) {
	if len(v) < 1 || ome.inlineDepth >= maxLeafInlineDepth {
		return NewNil(), tiZero, false
	}
	callee, ok := scmerSymbol(v[0])
	if !ok || (ome.inlineStack != nil && ome.inlineStack[callee]) {
		return NewNil(), tiZero, false
	}
	owner := env.FindRead(callee)
	if owner == nil {
		return NewNil(), tiZero, false
	}
	value, ok := owner.Vars[callee]
	if !ok || !value.IsProc() {
		return NewNil(), tiZero, false
	}
	proc := value.Proc()
	if proc == nil || proc.En == nil {
		return NewNil(), tiZero, false
	}
	params := proc.Params
	if stripped, ok := scmerStripSourceInfo(params); ok {
		params = stripped
	}
	paramItems, ok := scmerSlice(params)
	if !ok || len(paramItems) != len(v)-1 || proc.NumVars != len(paramItems) {
		return NewNil(), tiZero, false
	}
	refs := make([]int, len(paramItems))
	nodes := 0
	if !analyzeLeafInlineBody(proc.Body, callee, len(paramItems), refs, &nodes) {
		return NewNil(), tiZero, false
	}
	if !leafInlineBindingsStable(proc.Body, proc.En, env) {
		return NewNil(), tiZero, false
	}
	for _, count := range refs {
		if count != 1 {
			return NewNil(), tiZero, false
		}
	}
	inlined := substituteLeafInlineParams(proc.Body, v[1:])
	if ome.inlineStack == nil {
		ome.inlineStack = make(map[Symbol]bool)
	}
	ome.inlineStack[callee] = true
	ome.inlineDepth++
	result, ti := OptimizeEx(inlined, env, ome, useResult)
	ome.inlineDepth--
	delete(ome.inlineStack, callee)
	return result, ti, true
}

func scmerIsSymbol(v Scmer, name string) bool {
	if s, ok := symbolName(v); ok {
		return s == name
	}
	return false
}

func scmerSymbol(v Scmer) (Symbol, bool) {
	if s, ok := symbolName(v); ok {
		return Symbol(s), true
	}
	return "", false
}

func assignedSymbolsInLambda(body Scmer) map[Symbol]bool {
	var assigned map[Symbol]bool
	var visit func(Scmer)
	visit = func(expr Scmer) {
		if stripped, ok := scmerStripSourceInfo(expr); ok {
			expr = stripped
		}
		items, ok := scmerSlice(expr)
		if !ok || len(items) == 0 {
			return
		}
		if scmerIsSymbol(items[0], "quote") || scmerIsSymbol(items[0], "lambda") {
			return
		}
		if scmerIsSymbol(items[0], "set") && len(items) == 3 {
			if sym, ok := scmerSymbol(items[1]); ok {
				if assigned == nil {
					assigned = make(map[Symbol]bool)
				}
				assigned[sym] = true
			}
			visit(items[2])
			return
		}
		for _, item := range items {
			visit(item)
		}
	}
	visit(body)
	return assigned
}

func requiredNumberedSlots(expr Scmer) int {
	maxSlot := -1
	var visit func(Scmer)
	visit = func(current Scmer) {
		if stripped, ok := scmerStripSourceInfo(current); ok {
			current = stripped
		}
		if current.IsNthLocalVar() {
			if slot := int(current.NthLocalVar()); slot > maxSlot {
				maxSlot = slot
			}
			return
		}
		items, ok := scmerSlice(current)
		if !ok || len(items) == 0 || scmerIsSymbol(items[0], "quote") {
			return
		}
		if scmerIsSymbol(items[0], "lambda") {
			return
		}
		if (scmerIsSymbol(items[0], "!list") || scmerIsSymbol(items[0], "!!list")) &&
			len(items) >= 3 && items[1].IsNthLocalVar() && items[2].IsInt() {
			lastSlot := int(items[1].NthLocalVar()) + int(items[2].Int()) - 1
			if lastSlot > maxSlot {
				maxSlot = lastSlot
			}
		}
		for _, item := range items {
			visit(item)
		}
	}
	visit(expr)
	return maxSlot + 1
}

func scmerStripSourceInfo(v Scmer) (Scmer, bool) {
	if v.IsSourceInfo() {
		si := v.SourceInfo()
		si.coverage = true
		return si.value, true
	}
	if v.GetTag() == tagAny {
		if si, ok := v.Any().(SourceInfo); ok {
			si.coverage = true
			return si.value, true
		}
	}
	return v, false
}

func scmerSlice(v Scmer) ([]Scmer, bool) {
	if v.IsSlice() {
		return v.Slice(), true
	}
	if stripped, ok := scmerStripSourceInfo(v); ok {
		return scmerSlice(stripped)
	}
	if v.IsFastDict() {
		fd := v.FastDict()
		if fd == nil {
			return []Scmer{}, true
		}
		return fd.Pairs, true
	}
	return nil, false
}

// expressionContainsDynamicSyntaxCall detects forms that interpret an AST in
// their lexical environment. Symbols reachable by these forms must stay named.
func expressionContainsDynamicSyntaxCall(v Scmer) bool {
	if stripped, ok := scmerStripSourceInfo(v); ok {
		v = stripped
	}
	list, ok := scmerSlice(v)
	if !ok || len(list) == 0 {
		return false
	}
	if head, ok := scmerSymbol(list[0]); ok {
		if head == Symbol("eval") || head == Symbol("import") || head == Symbol("parser") {
			return true
		}
		if head == Symbol("quote") {
			return false
		}
	}
	for _, item := range list {
		if expressionContainsDynamicSyntaxCall(item) {
			return true
		}
	}
	return false
}

func OptimizeEx(val Scmer, env *Env, ome *optimizerMetainfo, useResult bool) (Scmer, TypeInfo) {
	if val.ptr == nil && val.aux == 0 {
		return NewNil(), tiConstTransfer
	}
	if val.IsNthLocalVar() {
		if td := ome.numberedTypes[val.NthLocalVar()]; td != nil {
			return val, TypeInfoFromTD(td)
		}
	}

	switch val.GetTag() {
	case tagNil:
		return val, TypeInfo{kind: KindNil, flags: FlagTransfer | FlagConst, length: UnknownLength}
	case tagBool:
		return val, TypeInfo{kind: KindBool, flags: FlagTransfer | FlagConst, length: UnknownLength}
	case tagInt:
		return val, TypeInfo{kind: KindInt, flags: FlagTransfer | FlagConst, length: UnknownLength}
	case tagFloat:
		return val, TypeInfo{kind: KindFloat, flags: FlagTransfer | FlagConst, length: UnknownLength}
	case tagString:
		return val, TypeInfo{kind: KindString, flags: FlagTransfer | FlagConst, length: UnknownLength}
	case tagSymbol:
		sym := mustSymbol(val)
		varType := ome.variableTypes[sym]
		if replacement, ok := ome.variableReplacement[sym]; ok {
			if replacement.IsSymbol() && mustSymbol(replacement) == sym {
				if varType != nil {
					return val, TypeInfoFromTD(varType)
				}
				return val, tiTransfer
			}
			if slice, ok := scmerSlice(replacement); ok && len(slice) == 2 && scmerIsSymbol(slice[0], "outer") {
				if s2, ok := scmerSymbol(slice[1]); ok && s2 == sym {
					if varType != nil {
						return val, TypeInfoFromTD(varType)
					}
					return val, tiTransfer
				}
			}
			result, ti := OptimizeEx(replacement, env, ome, useResult)
			if varType != nil {
				ti = TypeInfoFromTD(varType)
			}
			return result, ti
		}
		if varType != nil {
			return val, TypeInfoFromTD(varType)
		}
		return val, tiTransfer
	case tagSlice:
		return optimizeList(val.Slice(), env, ome, useResult)
	case tagSourceInfo:
		siPtr := val.SourceInfo()
		siPtr.coverage = true
		if SettingsHaveGoodBacktraces {
			result, ti := OptimizeEx(siPtr.value, env, ome, useResult)
			if ti.Const() {
				return result, ti
			}
			si := *siPtr
			si.value = result
			return NewSourceInfo(si), ti.WithoutTransfer()
		}
		return OptimizeEx(siPtr.value, env, ome, useResult)
	case tagAny:
		payload := val.Any()
		if pv, ok := payload.(SourceInfo); ok {
			pv.coverage = true
			if SettingsHaveGoodBacktraces {
				result, ti := OptimizeEx(pv.value, env, ome, useResult)
				if ti.Const() {
					return result, ti
				}
				pv.value = result
				return NewSourceInfo(pv), ti.WithoutTransfer()
			}
			return OptimizeEx(pv.value, env, ome, useResult)
		}
		if sym, ok := payload.(Symbol); ok {
			return OptimizeEx(NewSymbol(string(sym)), env, ome, useResult)
		}
		if sm, ok := payload.(Scmer); ok {
			return OptimizeEx(sm, env, ome, useResult)
		}
		switch v := payload.(type) {
		case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, string:
			return FromAny(v), tiConstTransfer
		}
		return val, tiZero
	default:
		return val, tiZero
	}
}

// Common TypeInfo values (stack-allocated, no heap)
var (
	tiConstTransfer = TypeInfo{flags: FlagTransfer | FlagConst, length: UnknownLength}
	tiTransfer      = TypeInfo{flags: FlagTransfer, length: UnknownLength}
	tiZero          = TypeInfo{length: UnknownLength}
)

// optimizeExCompat is a temporary bridge: calls OptimizeEx and unpacks TypeInfo
// into the old (transfer, const) bools used internally by optimizeList.
// TODO: migrate optimizeList internals to use TypeInfo directly.
func optimizeExCompat(val Scmer, env *Env, ome *optimizerMetainfo, useResult bool) (Scmer, bool, bool) {
	result, ti := OptimizeEx(val, env, ome, useResult)
	return result, ti.Transfer(), ti.Const()
}

// canEliminateFromBegin checks whether a constant-folded expression in a
// begin block (non-last position, result unused) can safely be removed.
// Literals and pure function results can be dropped. Expressions from
// functions with HasSideEffects or without type info are kept.
func canEliminateFromBegin(val Scmer) bool {
	// Literals are always safe to eliminate
	switch val.GetTag() {
	case tagNil, tagBool, tagInt, tagFloat, tagString:
		return true
	}
	// For function call results that were constant-folded: check the original
	// declaration. If it has HasSideEffects: true, keep it.
	// Since the expression was already folded to a constant value by this point,
	// we can safely eliminate it — the fold already executed the side effect.
	// But if it was NOT foldable and still appears as a call, check the decl.
	if slice, ok := scmerSlice(val); ok && len(slice) > 0 {
		if d := DeclarationForValue(slice[0]); d != nil {
			if d.Type == nil {
				return false // no type info → conservative, keep
			}
			return !d.Type.HasSideEffects
		}
		return false // unknown function → conservative, keep
	}
	return true // constant value, safe to eliminate
}

type localBindingFacts struct {
	defineIdx int
	firstUse  int
	count     int
	used      bool
}

func optimizeList(v []Scmer, env *Env, ome *optimizerMetainfo, useResult bool) (Scmer, TypeInfo) {
	var transferOwnership, isConstant bool
	if len(v) == 0 {
		return NewSlice(v), tiZero
	}

	headSym, headOk := scmerSymbol(v[0])

	if headOk && headSym == Symbol("outer") && len(v) == 2 {
		// When we see (outer expr), expr should be resolved in the parent scope.
		// Copy() wraps inherited variable replacements in (outer ...), but (outer expr)
		// already represents one scope transition. We need to "unwrap" one (outer ...)
		// level from the variable replacements to avoid double-wrapping.
		outerOme := optimizerMetainfo{
			variableReplacement: make(map[Symbol]Scmer),
			setBlacklist:        ome.setBlacklist,
		}
		for k, repl := range ome.variableReplacement {
			if slice, ok := scmerSlice(repl); ok && len(slice) == 2 && scmerIsSymbol(slice[0], "outer") {
				outerOme.variableReplacement[k] = slice[1]
			}
			// Local NthLocalVar replacements (current lambda params) are NOT
			// accessible in the outer scope, so we intentionally exclude them.
		}
		inner, transferOwnership, isConstant := optimizeExCompat(v[1], env, &outerOme, useResult)
		if isConstant {
			return inner, tiConstTransfer
		}
		v[1] = inner
		return NewSlice(v), MakeTypeInfo(transferOwnership, false)
	}

	if headOk && (headSym == Symbol("begin") || headSym == Symbol("begin_mut")) {
		bodyStart := 1
		reserve := 0
		if headSym == Symbol("begin_mut") {
			bodyStart = 2
			if len(v) > 1 {
				v[1], transferOwnership, _ = optimizeExCompat(v[1], env, ome, true)
				reserve = int(ToInt(v[1]))
				if reserve < 0 {
					reserve = 0
				}
			}
		}
		usedVariables := make(map[Symbol]int)
		variableContent := make(map[Symbol]Scmer)
		bindings := make(map[Symbol]localBindingFacts)
		bindingOrder := make([]Symbol, 0)
		earliestDynamicSyntax := -1
		currentTopIdx := 0
		for i := bodyStart; i < len(v); i++ {
			expr := v[i]
			if stripped, ok := scmerStripSourceInfo(expr); ok {
				expr = stripped
			}
			if sub, ok := scmerSlice(expr); ok && len(sub) > 0 {
				headExpr := sub[0]
				if stripped, ok := scmerStripSourceInfo(headExpr); ok {
					headExpr = stripped
				}
				if head, ok := scmerSymbol(headExpr); ok {
					if (head == Symbol("define") || head == Symbol("set")) && len(sub) >= 3 {
						if sym, ok := scmerSymbol(sub[1]); ok {
							facts := bindings[sym]
							if facts.count == 0 {
								bindingOrder = append(bindingOrder, sym)
							}
							facts.defineIdx = i
							facts.count++
							bindings[sym] = facts
						}
					}
					if head == Symbol("eval") || head == Symbol("import") {
						if earliestDynamicSyntax == -1 || i < earliestDynamicSyntax {
							earliestDynamicSyntax = i
						}
					}
				}
				if expressionContainsDynamicSyntaxCall(expr) {
					if earliestDynamicSyntax == -1 || i < earliestDynamicSyntax {
						earliestDynamicSyntax = i
					}
				}
			}
		}
		var visitNode func(x Scmer, depth int, blacklist []Symbol)
		visitNode = func(x Scmer, depth int, blacklist []Symbol) {
			if stripped, ok := scmerStripSourceInfo(x); ok {
				x = stripped
			}
			if sub, ok := scmerSlice(x); ok && len(sub) > 0 {
				subHeadExpr := sub[0]
				if stripped, ok := scmerStripSourceInfo(subHeadExpr); ok {
					subHeadExpr = stripped
				}
				subHead, subHeadOk := scmerSymbol(subHeadExpr)
				if subHeadOk && (subHead == Symbol("define") || subHead == Symbol("set")) {
					visitNode(sub[2], depth, blacklist)
					if depth == 0 {
						if sym, ok := scmerSymbol(sub[1]); ok {
							variableContent[sym] = sub[2]
						}
					}
				} else if subHeadOk && subHead == Symbol("lambda") {
					params := sub[1]
					if stripped, ok := scmerStripSourceInfo(params); ok {
						params = stripped
					}
					if sym, ok := scmerSymbol(params); ok {
						visitNode(sub[2], depth+1, append(append([]Symbol{}, blacklist...), sym))
					} else if list, ok := scmerSlice(params); ok {
						blacklist2 := append([]Symbol{}, blacklist...)
						for _, entry := range list {
							if s, ok := scmerSymbol(entry); ok {
								blacklist2 = append(blacklist2, s)
							}
						}
						visitNode(sub[2], depth+1, blacklist2)
					}
				} else if subHeadOk && (subHead == Symbol("begin") || subHead == Symbol("begin_mut")) {
					start := 1
					if subHead == Symbol("begin_mut") {
						start = 2
					}
					for i := start; i < len(sub); i++ {
						visitNode(sub[i], depth+1, blacklist)
					}
				} else if subHeadOk && subHead == Symbol("!begin") {
					for i := 1; i < len(sub); i++ {
						visitNode(sub[i], depth, blacklist)
					}
				} else if subHeadOk && subHead == Symbol("eval") {
					usedVariables[Symbol("eval")] = 1
					for i := 2; i < len(sub); i++ {
						visitNode(sub[i], depth+1, blacklist)
					}
				} else {
					// Also visit the head — it may be a variable used in call position (e.g., (accsess "key"))
					visitNode(sub[0], depth+1, blacklist)
					for i := 1; i < len(sub); i++ {
						visitNode(sub[i], depth+1, blacklist)
					}
				}
				return
			}
			if sym, ok := scmerSymbol(x); ok {
				isBlacklisted := false
				for _, b := range blacklist {
					if b == sym {
						isBlacklisted = true
						break
					}
				}
				if !isBlacklisted {
					if facts, tracked := bindings[sym]; tracked && !facts.used {
						facts.firstUse = currentTopIdx
						facts.used = true
						bindings[sym] = facts
					}
					if depth > 0 {
						usedVariables[sym] = 100
					} else {
						usedVariables[sym] = usedVariables[sym] + 1
					}
				}
			}
		}
		for i := bodyStart; i < len(v); i++ {
			currentTopIdx = i
			visitNode(v[i], 0, nil)
		}
		ome2 := ome.CopySharedScope()
		ome2.beginDepth++
		slotLimit := -1
		if headSym == Symbol("begin_mut") {
			slotIndex := 0
			if ome.nextSlot != nil {
				slotIndex = *ome.nextSlot
			}
			slotLimit = slotIndex + reserve
			ome2.nextSlot = &slotIndex
		}
		for sym, content := range variableContent {
			normalized := content
			if stripped, ok := scmerStripSourceInfo(content); ok {
				normalized = stripped
			}
			// Bring back old criterion: inline if used < 2 OR RHS is not a list
			shouldReplace := usedVariables[sym] < 2 || !normalized.IsSlice()
			// Convention: symbols starting with "tbl:" are pre-resolved table
			// pointers that must not be inlined back into inner loops.
			if strings.HasPrefix(string(sym), "tbl:") {
				shouldReplace = false
			}
			// Never inline aliases to symbols; this preserves outer/old-handler semantics
			if normalized.IsSymbol() {
				shouldReplace = false
			}
			// Safeguard: do not inline self-aliases
			if normalized.IsSymbol() {
				if mustSymbol(normalized) == sym {
					shouldReplace = false
				}
				// Safeguard: if RHS references a symbol that is defined at top-level in this begin, do not inline
				if _, ok := bindings[mustSymbol(normalized)]; ok {
					shouldReplace = false
				}
			}
			// Safeguard: if a dynamic syntax form appears anywhere in this begin,
			// keep all bindings explicit. eval can resolve dynamically-built symbols
			// against lexical scope, so partial inlining is unsafe.
			if earliestDynamicSyntax >= 0 {
				shouldReplace = false
			}
			if shouldReplace {
				delete(variableContent, sym)
				delete(usedVariables, sym)
				delete(bindings, sym)
				ome2.setBlacklist = append(ome2.setBlacklist, sym)
				ome2.variableReplacement[sym] = content
			}
		}
		var numberedLocals map[Symbol]Scmer
		if earliestDynamicSyntax < 0 && ome2.nextSlot != nil {
			for _, sym := range bindingOrder {
				if _, retained := variableContent[sym]; !retained {
					continue
				}
				facts := bindings[sym]
				if facts.count != 1 || (facts.used && facts.firstUse <= facts.defineIdx) || *ome2.nextSlot >= 256 {
					continue
				}
				if numberedLocals == nil {
					numberedLocals = make(map[Symbol]Scmer)
				}
				slot := NewNthLocalVar(NthLocalVar(*ome2.nextSlot))
				*ome2.nextSlot++
				numberedLocals[sym] = slot
				delete(usedVariables, sym)
				delete(bindings, sym)
			}
		}
		if len(usedVariables) == 0 && len(bindings) == 0 {
			v[0] = NewSymbol("!begin")
			for sym, content := range ome2.variableReplacement {
				if slice, ok := scmerSlice(content); ok && len(slice) == 2 && scmerIsSymbol(slice[0], "outer") {
					ome2.variableReplacement[sym] = slice[1]
				}
			}
		}
		for i := bodyStart; i < len(v); i++ {
			expr := v[i]
			if stripped, ok := scmerStripSourceInfo(expr); ok {
				expr = stripped
			}
			if items, ok := scmerSlice(expr); ok && len(items) == 3 && (scmerIsSymbol(items[0], "define") || scmerIsSymbol(items[0], "set")) {
				if sym, ok := scmerSymbol(items[1]); ok {
					if slot, numbered := numberedLocals[sym]; numbered {
						ome2.variableReplacement[sym] = slot
					}
				}
			}
			var constant bool
			v[i], transferOwnership, constant = optimizeExCompat(v[i], env, &ome2, i == len(v)-1 && useResult)
			if constant {
				if i == len(v)-1 {
					isConstant = true
				} else if canEliminateFromBegin(v[i]) {
					v = append(v[:i], v[i+1:]...)
					i--
				}
			}
		}
		if slotLimit >= 0 && ome2.nextSlot != nil && *ome2.nextSlot > slotLimit {
			panic("begin_mut reserved too few numbered vars")
		}
		// Flatten nested !begin blocks
		if scmerIsSymbol(v[0], "!begin") {
			for i := 1; i < len(v); i++ {
				if inner, ok := scmerSlice(v[i]); ok && len(inner) > 1 && scmerIsSymbol(inner[0], "!begin") {
					newV := make([]Scmer, 0, len(v)+len(inner)-2)
					newV = append(newV, v[:i]...)
					newV = append(newV, inner[1:]...)
					newV = append(newV, v[i+1:]...)
					v = newV
					i-- // re-examine this position
				}
			}
		}
		if scmerIsSymbol(v[0], "!begin") && len(v) == 1 {
			return NewNil(), tiConstTransfer
		}
		if scmerIsSymbol(v[0], "!begin") && len(v) == 2 {
			return OptimizeEx(v[1], env, &ome2, useResult)
		}
		if scmerIsSymbol(v[0], "begin") && len(v) == 2 {
			return OptimizeEx(v[1], env, &ome2, useResult)
		}
		if scmerIsSymbol(v[0], "begin") || scmerIsSymbol(v[0], "begin_mut") {
			isConstant = false
		}
		return NewSlice(v), MakeTypeInfo(transferOwnership, isConstant)
	}

	if headOk && headSym == Symbol("var") && len(v) == 2 {
		return NewNthLocalVar(NthLocalVar(ToInt(v[1]))), tiZero
	}

	if headOk && headSym == Symbol("unquote") && len(v) == 2 {
		unquoted := v[1]
		if stripped, ok := scmerStripSourceInfo(unquoted); ok {
			unquoted = stripped
		}
		switch unquoted.GetTag() {
		case tagString:
			return NewSymbol(unquoted.String()), tiTransfer
		case tagAny:
			if s, ok := unquoted.Any().(string); ok {
				return NewSymbol(s), tiTransfer
			}
		}
	}

	if headOk && headSym == Symbol("lambda") {
		params := v[1]
		assigned := assignedSymbolsInLambda(v[2])
		if stripped, ok := scmerStripSourceInfo(params); ok {
			params = stripped
		}
		// Dynamic syntax forms rely on symbol lookup in the current lexical scope
		// and cannot see NthLocalVar-only parameters.
		if expressionContainsDynamicSyntaxCall(v[2]) {
			ome2 := ome.Copy()
			ome2.lambdaDepth++
			for sym := range assigned {
				delete(ome2.variableReplacement, sym)
			}
			if list, ok := scmerSlice(params); ok {
				for _, param := range list {
					if sym, ok := scmerSymbol(param); ok {
						delete(ome2.variableReplacement, sym)
					}
				}
			} else if sym, ok := scmerSymbol(params); ok {
				delete(ome2.variableReplacement, sym)
			}
			v[2], _, _ = optimizeExCompat(v[2], env, &ome2, true)
			return NewSlice(v), tiZero
		}
		/* Lambdas with explicit NumVars still execute in numbered-call frames at
		runtime. Some generated plans keep symbolic parameter references in the
		body while adding NumVars later, so we must continue mapping declared
		parameters onto their numbered slots here instead of dropping them.
		Otherwise callbacks like $update remain unbound at runtime. */
		if len(v) > 3 {
			ome2 := ome.Copy()
			ome2.lambdaDepth++
			for sym := range assigned {
				delete(ome2.variableReplacement, sym)
			}
			declaredNumVars := int(ToInt(v[3]))
			numVars := declaredNumVars
			slotIndex := numVars
			if required := requiredNumberedSlots(v[2]); required > slotIndex {
				slotIndex = required
			}
			ome2.nextSlot = &slotIndex
			if ome.pendingCallbackReturn != nil {
				v[2] = rewriteNoEscapeListReturn(v[2], ome.pendingCallbackReturn, &slotIndex)
				ome.pendingCallbackReturn = nil
			}
			if list, ok := scmerSlice(params); ok {
				for i, param := range list {
					if i >= numVars {
						break
					}
					if sym, ok := scmerSymbol(param); ok && sym != Symbol("_") && !assigned[sym] {
						ome2.variableReplacement[sym] = NewNthLocalVar(NthLocalVar(i))
					}
				}
			} else if sym, ok := scmerSymbol(params); ok && !assigned[sym] {
				ome2.variableReplacement[sym] = NewNthLocalVar(0)
			}
			ome.applyPendingCallbackParams(params, &ome2)
			var bodyType TypeInfo
			v[2], bodyType = OptimizeEx(v[2], env, &ome2, true)
			if slotIndex != declaredNumVars {
				v[3] = NewInt(int64(slotIndex))
			}
			return NewSlice(v), bodyType.WithoutConst()
		}
		// Auto-number parameters
		ome2 := ome.Copy()
		ome2.lambdaDepth++
		for sym := range assigned {
			delete(ome2.variableReplacement, sym)
		}
		slotIndex := 0
		if list, ok := scmerSlice(params); ok {
			for _, param := range list {
				if sym, ok := scmerSymbol(param); ok {
					if sym != Symbol("_") && !assigned[sym] {
						ome2.variableReplacement[sym] = NewNthLocalVar(NthLocalVar(slotIndex))
					}
				}
				slotIndex++
			}
		} else if sym, ok := scmerSymbol(params); ok {
			if !assigned[sym] {
				ome2.variableReplacement[sym] = NewNthLocalVar(NthLocalVar(slotIndex))
			}
			slotIndex++
		}
		ome2.nextSlot = &slotIndex // allow !list/!!list to allocate extra slots
		if ome.pendingCallbackReturn != nil {
			v[2] = rewriteNoEscapeListReturn(v[2], ome.pendingCallbackReturn, &slotIndex)
			ome.pendingCallbackReturn = nil
		}
		ome.applyPendingCallbackParams(params, &ome2)
		var bodyType TypeInfo
		v[2], bodyType = OptimizeEx(v[2], env, &ome2, true)
		// Set NumVars (may have grown due to !list allocations)
		if slotIndex > 0 {
			v = append(v[:len(v):len(v)], NewInt(int64(slotIndex)))
		}
		return NewSlice(v), bodyType.WithoutConst()
	}

	switch {
	case headOk && (headSym == Symbol("set") || headSym == Symbol("define")) && len(v) == 3:
		var definedSym Symbol
		var hasDefinedSym bool
		if sym, ok := scmerSymbol(v[1]); ok {
			definedSym, hasDefinedSym = sym, true
			for _, black := range ome.setBlacklist {
				if black == sym {
					if useResult {
						return ome.variableReplacement[sym], tiZero
					}
					return NewNil(), tiConstTransfer
				}
			}
			if repl, ok := ome.variableReplacement[sym]; ok && repl.IsNthLocalVar() {
				v[1] = repl
			}
			if ome.lambdaDepth == 0 && ome.beginDepth == 0 {
				env.deleteOptimizerHint(sym)
			}
		}
		if v[1].IsNthLocalVar() {
			v[0] = NewSymbol("setN")
		}
		var returnType TypeInfo
		v[2], returnType = OptimizeEx(v[2], env, ome, true)
		transferOwnership = returnType.Transfer()
		if hasDefinedSym && ome.lambdaDepth == 0 && ome.beginDepth == 0 {
			rhs := v[2]
			if stripped, ok := scmerStripSourceInfo(rhs); ok {
				rhs = stripped
			}
			if items, ok := scmerSlice(rhs); ok && len(items) >= 3 && scmerIsSymbol(items[0], "lambda") && returnType.Kind() != KindAny {
				env.setOptimizerHint(definedSym, returnType.WithoutConst())
			}
		}
	case headOk && (headSym == Symbol("match") || headSym == Symbol("match_mut")):
		value, valueTransfer, _ := optimizeExCompat(v[1], env, ome, true)
		v[1] = value
		transferOwnership = valueTransfer
		if headSym == Symbol("match") && valueTransfer {
			v[0] = NewSymbol("match_mut")
		}
		for i := 3; i < len(v); i += 2 {
			ome2 := ome.CopySharedScope()
			v[i-1] = OptimizeMatchPattern(v[1], v[i-1], env, ome, &ome2)
			v[i], transferOwnership, _ = optimizeExCompat(v[i], env, &ome2, useResult)
		}
		if len(v)%2 == 1 {
			v[len(v)-1], transferOwnership, _ = optimizeExCompat(v[len(v)-1], env, ome, useResult)
		}
		return NewSlice(v), MakeTypeInfo(transferOwnership, false)
	case headOk && headSym == Symbol("parser"):
		return OptimizeParser(NewSlice(v), env, ome, false), tiTransfer
	case !headOk || headSym != Symbol("quote"):
		// Look up declaration for hook dispatch
		if callDecl := DeclarationForValue(v[0]); callDecl != nil && callDecl.Type != nil && callDecl.Type.Optimize != nil {
			oc := &OptimizerContext{Env: env, Ome: ome}
			result, td := callDecl.Type.Optimize(v, oc, useResult)
			return result, TypeInfoFromTD(td)
		}
		// Default optimization path
		oc := &OptimizerContext{Env: env, Ome: ome}
		result, td := oc.applyDefaultOptimization(v, useResult, "")
		return result, TypeInfoFromTD(td)
	}

	return NewSlice(v), MakeTypeInfo(transferOwnership, false)
}

// OptimizeSub optimizes a sub-expression and returns its result TypeDescriptor.
func (oc *OptimizerContext) OptimizeSub(val Scmer, useResult bool) (Scmer, *TypeDescriptor) {
	result, ti := OptimizeEx(val, oc.Env, oc.Ome, useResult)
	return result, ti.ToTypeDescriptor()
}

// OptimizerRewriteContract declares the safety proof and maximum AST growth
// for one recursive hook rewrite.
type OptimizerRewriteContract struct {
	Name             string
	PreconditionsMet bool
	MaxGrowthNodes   int
}

// OptimizeRewrite is the only supported path for a hook to recursively
// optimize rewritten code. It enforces reentrancy, fingerprint, work-budget,
// and AST-growth guards. The boolean is false when the caller must continue
// with its non-rewrite optimization path.
func (oc *OptimizerContext) OptimizeRewrite(original, rewritten Scmer, useResult bool, contract OptimizerRewriteContract) (Scmer, *TypeDescriptor, bool) {
	state := oc.Ome.rewrite
	if state == nil {
		state = newOptimizerMetainfo().rewrite
		oc.Ome.rewrite = state
	}
	activeKey := contract.Name + ":" + strconv.FormatUint(HashKey(original), 16)
	if contract.Name == "" || !contract.PreconditionsMet || state.remainingBudget <= 0 || state.active[activeKey] {
		state.rejected++
		return original, nil, false
	}
	originalNodes := optimizerNodeCount(original)
	rewrittenNodes := optimizerNodeCount(rewritten)
	if contract.MaxGrowthNodes >= 0 && rewrittenNodes-originalNodes > contract.MaxGrowthNodes {
		state.rejected++
		return original, nil, false
	}
	globalLimit := state.inputNodes*4 + 1024
	if state.inputNodes > 0 && rewrittenNodes > globalLimit {
		state.rejected++
		return original, nil, false
	}
	fingerprint := HashKey(rewritten)
	if state.seen[fingerprint] {
		state.rejected++
		return original, nil, false
	}
	state.seen[fingerprint] = true
	state.remainingBudget--
	state.rewrites++
	state.active[activeKey] = true
	defer func() {
		delete(state.active, activeKey)
		delete(state.seen, fingerprint)
	}()
	result, td := oc.OptimizeSub(rewritten, useResult)
	return result, td, true
}

// AnalyzeCallback computes a callback result type without consuming pending
// metadata or otherwise mutating the optimizer context used for emitted code.
func (oc *OptimizerContext) AnalyzeCallback(callback Scmer, params []*TypeDescriptor) *TypeDescriptor {
	if oc.Ome.rewrite != nil {
		oc.Ome.rewrite.callbackAnalyses++
		oc.Ome.rewrite.callbackClones++
	}
	analysisOme := newOptimizerMetainfo()
	analysisOme.rewrite = oc.Ome.rewrite
	analysisOme.loopDepth = oc.Ome.loopDepth
	analysis := OptimizerContext{Env: oc.Env, Ome: &analysisOme}
	analysis.SetCallbackParamTypes(params)
	_, result := analysis.OptimizeSub(CloneOptimizerExpression(callback), true)
	return result
}

// OptimizeReducerCallback derives accumulator ownership from the neutral value
// and the reducer's actual return flow. The descriptor join is monotonic: facts
// are retained only while they hold for the initial value and every iteration.
func (oc *OptimizerContext) OptimizeReducerCallback(callback Scmer, accumulator *TypeDescriptor, values ...*TypeDescriptor) (Scmer, *TypeDescriptor) {
	accumulator = normalizeOptimizerType(accumulator)
	params := make([]*TypeDescriptor, len(values)+1)
	copy(params[1:], values)
	if !accumulator.Transfer {
		params[0] = accumulator
		oc.SetCallbackParamTypes(params)
		optimized, result := oc.OptimizeSub(callback, true)
		return optimized, normalizeOptimizerType(result)
	}
	loopType := accumulator
	for iteration := 0; iteration < 16; iteration++ {
		params[0] = loopType
		result := normalizeOptimizerType(oc.AnalyzeCallback(callback, params))
		next, changed := mergeOptimizerTypes(loopType, result)
		if !changed {
			params[0] = next
			oc.SetCallbackParamTypes(params)
			optimized, finalResult := oc.OptimizeSub(callback, true)
			return optimized, normalizeOptimizerType(finalResult)
		}
		loopType = next
	}
	// The join is monotonic, but malformed custom descriptors must not make
	// optimizer compilation unbounded.
	params[0] = loopType
	oc.SetCallbackParamTypes(params)
	optimized, finalResult := oc.OptimizeSub(callback, true)
	return optimized, normalizeOptimizerType(finalResult)
}

func normalizeOptimizerType(td *TypeDescriptor) *TypeDescriptor {
	if td == nil {
		return &TypeDescriptor{Kind: "any", Length: UnknownLength}
	}
	return td
}

func mergeOptimizerTypes(current, result *TypeDescriptor) (*TypeDescriptor, bool) {
	current = normalizeOptimizerType(current)
	result = normalizeOptimizerType(result)
	merged := &TypeDescriptor{
		Kind:     current.Kind,
		NoEscape: current.NoEscape && result.NoEscape,
		Transfer: current.Transfer && result.Transfer,
		Const:    current.Const && result.Const,
		Length:   current.Length,
	}
	changed := merged.NoEscape != current.NoEscape || merged.Transfer != current.Transfer || merged.Const != current.Const
	if merged.Kind != result.Kind {
		merged.Kind = "any"
		changed = merged.Kind != current.Kind || changed
	}
	if merged.Length != result.Length {
		merged.Length = UnknownLength
		changed = merged.Length != current.Length || changed
	}
	if len(current.Keys) > 0 && len(result.Keys) > 0 {
		merged.Keys = make(map[string]*TypeDescriptor)
		for key, currentKey := range current.Keys {
			if resultKey := result.Keys[key]; resultKey != nil {
				var keyChanged bool
				merged.Keys[key], keyChanged = mergeOptimizerTypes(currentKey, resultKey)
				changed = changed || keyChanged
			}
		}
	}
	if len(merged.Keys) != len(current.Keys) {
		changed = true
	}
	if current.Element != nil && result.Element != nil {
		var elementChanged bool
		merged.Element, elementChanged = mergeOptimizerTypes(current.Element, result.Element)
		changed = changed || elementChanged
	} else if current.Element != nil {
		changed = true
	}
	return merged, changed
}

// SetCallbackParamTypes supplies structural callback parameter types. Nested
// Keys preserve ownership independently for projected list/association values.
func (oc *OptimizerContext) SetCallbackParamTypes(types []*TypeDescriptor) {
	oc.Ome.pendingCallbackParams = types
}

// SetCallbackReturnFlow provides structured escape information for the next
// callback result, including independently tracked list/association entries.
func (oc *OptimizerContext) SetCallbackReturnFlow(flow *TypeDescriptor) {
	oc.Ome.pendingCallbackReturn = flow
}

// ApplyDefaultOptimization runs the standard optimization pipeline on a call
// expression: callback ownership propagation, arg optimization, !list rewrite,
// and constant folding. Hooks can call this for default behavior.
func (oc *OptimizerContext) ApplyDefaultOptimization(v []Scmer, useResult bool) (Scmer, *TypeDescriptor) {
	return oc.applyDefaultOptimization(v, useResult, "")
}

// FirstParameterMutable returns an Optimize hook that runs default optimization
// and swaps to mutName when the first argument is exclusively owned.
func FirstParameterMutable(mutName string) func(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	return func(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
		return oc.applyDefaultOptimization(v, useResult, mutName)
	}
}

// applyDefaultOptimization runs the standard optimization pipeline on a function
// call expression: callback ownership propagation, arg optimization, _mut swap
// (when mutName is non-empty), !list rewrite, and constant folding.
func (oc *OptimizerContext) applyDefaultOptimization(v []Scmer, useResult bool, mutName string) (Scmer, *TypeDescriptor) {
	env := oc.Env
	ome := oc.Ome
	head, _ := scmerSymbol(v[0])

	allConstArgs := true
	var transferOwnership bool
	var firstArgType TypeInfo
	argTypes := make([]TypeInfo, len(v))

	// Resolve native and symbolic call heads through the same declaration. Code
	// generated by Scheme commonly contains either representation.
	callDecl := DeclarationForValue(v[0])
	callName := string(head)
	if callDecl != nil {
		callName = callDecl.Name
	}

	// Optimize all args with callback ownership propagation
	for i := 0; i < len(v); i++ {
		ome.pendingCallbackParams = nil
		ome.pendingCallbackReturn = nil
		if i > 0 && callDecl != nil {
			paramIdx := i - 1
			if callDecl.Type == nil || len(callDecl.Type.Params) == 0 {
				// no type info
			} else if paramIdx >= len(callDecl.Type.Params) {
				paramIdx = len(callDecl.Type.Params) - 1
			}
			if paramIdx >= 0 && callDecl.Type != nil && paramIdx < len(callDecl.Type.Params) {
				if ti := callDecl.Type.Params[paramIdx]; ti != nil && ti.Kind == "func" && len(ti.Params) > 0 {
					ome.pendingCallbackParams = ti.Params
					ome.pendingCallbackReturn = ti.Return
				}
			}
		}
		var ti TypeInfo
		v[i], ti = OptimizeEx(v[i], env, ome, true)
		ome.pendingCallbackParams = nil
		ome.pendingCallbackReturn = nil
		argTypes[i] = ti
		if i == 1 {
			firstArgType = ti
		}
		transferOwnership = ti.Transfer()
		if i > 0 && !ti.Const() {
			allConstArgs = false
		}
	}
	if !allConstArgs {
		if inlined, ti, ok := tryInlineLeafProc(v, env, ome, useResult); ok {
			return inlined, ti.ToTypeDescriptor()
		}
	}

	// _mut swap: when mutName is set and first arg is exclusively owned,
	// swap to the in-place variant
	if mutName != "" {
		firstArgFresh := false
		if len(v) >= 2 {
			arg1 := v[1]
			if si, ok := arg1.Any().(SourceInfo); ok {
				arg1 = si.value
			}
			if td := optimizerExpressionDescriptor(arg1, env, ome); td != nil {
				firstArgFresh = td.Transfer && !td.Const
			}
		}
		if firstArgFresh && len(v) >= 2 {
			v[0] = NewSymbol(mutName)
			transferOwnership = true
		}
	}

	// !!list rewrite: turn surface (!!list cap) into the internal
	// stack-backed form (!!list NthLocalVar(start) cap) when we are inside
	// an optimizer-numbered lambda frame.
	if scmerIsSymbol(v[0], "!!list") && len(v) == 2 && ome.nextSlot != nil {
		capacity := int(ToInt(v[1]))
		if capacity < 0 {
			capacity = 0
		}
		start := *ome.nextSlot
		*ome.nextSlot += capacity
		return NewSlice([]Scmer{NewSymbol("!!list"), NewNthLocalVar(NthLocalVar(start)), NewInt(int64(capacity))}), &TypeDescriptor{Transfer: true}
	}

	// !list rewrite: when an argument is (list expr...) passed to a function
	// whose parameter is annotated NoEscape:true, replace with (!list start count expr...)
	// so the list is stack-allocated into VarsNumbered instead of heap-allocated.
	if !allConstArgs && ome.nextSlot != nil {
		if decl := callDecl; decl != nil && decl.Type != nil && len(decl.Type.Params) > 0 {
			for i := 1; i < len(v); i++ {
				paramIdx := i - 1
				if paramIdx >= len(decl.Type.Params) {
					paramIdx = len(decl.Type.Params) - 1 // variadic: use last param
				}
				if paramIdx < 0 {
					continue
				}
				ti := decl.Type.Params[paramIdx]
				if ti == nil || !ti.NoEscape {
					continue // unknown or escaping parameter
				}
				// Check if this argument is a (list ...) call
				if inner, ok := scmerSlice(v[i]); ok && len(inner) >= 1 {
					innerDecl := DeclarationForValue(inner[0])
					if !scmerIsSymbol(inner[0], "list") && (innerDecl == nil || innerDecl.Name != "list") {
						continue
					}
					count := len(inner) - 1
					if count == 0 {
						continue // empty list, no benefit
					}
					start := *ome.nextSlot
					*ome.nextSlot += count
					rewritten := make([]Scmer, 0, count+3)
					rewritten = append(rewritten, NewSymbol("!list"))
					rewritten = append(rewritten, NewNthLocalVar(NthLocalVar(start)))
					rewritten = append(rewritten, NewInt(int64(count)))
					rewritten = append(rewritten, inner[1:]...)
					v[i] = NewSlice(rewritten)
				}
			}
		}
	}

	if scmerIsSymbol(v[0], "!begin") && allConstArgs {
		return v[len(v)-1], &TypeDescriptor{Transfer: true, Const: true}
	}

	// Look up declaration return type for propagation
	var retTD *TypeDescriptor
	var procReturn TypeInfo
	hasProcReturn := false
	if d := DeclarationForValue(v[0]); d != nil {
		argCount := len(v) - 1
		if d.IsFoldable() && allConstArgs && d.Fn != nil && argCount >= d.MinParams() && argCount <= d.MaxParams() {
			for i := range v {
				v[i] = unwrapConstListFromCode(v[i])
			}
			result := d.Fn(v[1:]...)
			result = wrapConstListForCode(result)
			td := &TypeDescriptor{Transfer: true, Const: true, Length: UnknownLength}
			if d.Type != nil && d.Type.Return != nil {
				td = &TypeDescriptor{Transfer: true, Const: true, Kind: d.Type.Return.Kind,
					Params: d.Type.Return.Params, Return: d.Type.Return.Return,
					HasSideEffects: d.Type.Return.HasSideEffects, Length: d.Type.Return.Length}
			}
			return result, td
		}
		if d.Type != nil && d.Type.Return != nil {
			retTD = d.Type.Return
		}
	}
	if retTD == nil {
		if sym, ok := scmerSymbol(v[0]); ok {
			if ti, exists := env.optimizerProcHint(sym); exists {
				procReturn = ti
				hasProcReturn = true
			}
		}
	}

	td := &TypeDescriptor{Transfer: transferOwnership}
	if hasProcReturn {
		td.Transfer = procReturn.Transfer()
		td.Kind = procReturn.kindName()
		td.Length = procReturn.Length()
		if procReturn.Extra != nil {
			td.Params = procReturn.Extra.Params
			td.Return = procReturn.Extra.Return
			td.HasSideEffects = procReturn.Extra.HasSideEffects
			td.Keys = procReturn.Extra.Keys
			td.Element = procReturn.Extra.Element
		}
	} else if retTD != nil {
		td.Kind = retTD.Kind
		td.Length = retTD.Length
		td.Params = retTD.Params
		td.Return = retTD.Return
		td.HasSideEffects = retTD.HasSideEffects
		td.Keys = retTD.Keys
		td.Element = retTD.Element
	} else {
		td.Length = UnknownLength
	}
	if callName == "list" {
		td.Kind = "list"
		td.Transfer = true
		td.Length = len(v) - 1
		td.Keys = make(map[string]*TypeDescriptor, len(v)-1)
		for i := 1; i < len(v); i++ {
			td.Keys[strconv.Itoa(i-1)] = argTypes[i].ToTypeDescriptor()
		}
	} else if callName == "!list" && len(v) >= 3 {
		td.Keys = make(map[string]*TypeDescriptor, len(v)-3)
		for i := 3; i < len(v); i++ {
			td.Keys[strconv.Itoa(i-3)] = argTypes[i].ToTypeDescriptor()
		}
	} else if len(v) >= 2 {
		key := ""
		switch {
		case callName == "car" && len(v) == 2:
			key = "0"
		case callName == "cadr" && len(v) == 2:
			key = "1"
		case callName == "nth" && len(v) == 3 && v[2].IsInt():
			key = strconv.FormatInt(v[2].Int(), 10)
		case callName == "get_assoc" && len(v) == 3 && v[2].IsString():
			key = String(v[2])
		}
		if key != "" {
			if projected := descriptorKey(firstArgType.ToTypeDescriptor(), key); projected != nil {
				td = projected
			}
		}
	}
	return NewSlice(v), td
}

// wrapConstListForCode wraps a constant-folded Scmer value so it can safely
// be embedded in generated code. Raw list/slice values would be misinterpreted
// as function calls by Eval, so they are wrapped as (list ...) calls recursively.
// Symbols are wrapped in (quote sym) so they evaluate to the symbol value rather
// than being looked up as variable references.
// Only wraps plain slices — FastDicts are left as-is since they are self-evaluating.
func wrapConstListForCode(val Scmer) Scmer {
	if val.IsSlice() {
		list := val.Slice()
		packed := make([]Scmer, 1, len(list)+1)
		packed[0] = NewSymbol("list")
		for _, elem := range list {
			packed = append(packed, wrapConstListForCode(elem))
		}
		return NewSlice(packed)
	}
	if val.IsSymbol() {
		return NewSlice([]Scmer{NewSymbol("quote"), val})
	}
	return val
}

// unwrapConstListFromCode is the inverse of wrapConstListForCode: it recursively
// strips (list ...) wrappers so that the raw data values can be passed to a
// foldable function at constant-fold time.
func unwrapConstListFromCode(val Scmer) Scmer {
	if list, ok := scmerSlice(val); ok && len(list) > 0 && (isList(list[0]) || scmerIsSymbol(list[0], "list")) {
		items := make([]Scmer, len(list)-1)
		for i, elem := range list[1:] {
			items[i] = unwrapConstListFromCode(elem)
		}
		return NewSlice(items)
	}
	return val
}

// optimizeIf is the Optimize hook for the (if ...) special form.
// It eliminates dead branches when conditions are compile-time constants.
// (if true x ...) → x, (if false _ cond2 then2 ...) → (if cond2 then2 ...)
func optimizeIf(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	// Process condition-then pairs, folding away constant conditions
	out := []Scmer{v[0]} // keep head "if"
	i := 1
	for i+1 < len(v) {
		cond, _ := oc.OptimizeSub(v[i], true)
		switch cond.GetTag() {
		case tagNil, tagBool, tagInt, tagFloat, tagString:
			if ToBool(cond) {
				// Constant true: this branch is always taken, return its body
				body, td := oc.OptimizeSub(v[i+1], useResult)
				return body, td
			}
			// Constant false: skip this condition+then pair entirely
			i += 2
			continue
		}
		// Non-constant condition: keep it and optimize the then-branch
		then, _ := oc.OptimizeSub(v[i+1], useResult)
		out = append(out, cond, then)
		i += 2
	}
	// Remaining else branch
	if i < len(v) {
		elseVal, td := oc.OptimizeSub(v[i], useResult)
		if len(out) == 1 {
			// All conditions were constant-false: return else branch
			return elseVal, td
		}
		out = append(out, elseVal)
	} else if len(out) == 1 {
		// All conditions false and no else: nil
		return NewNil(), &TypeDescriptor{Transfer: true, Const: true}
	}
	// Single condition left: keep as (if cond then else)
	return NewSlice(out), nil
}

// optimizeAnd is the Optimize hook for the lazy (and ...) special form.
// Nil is SQL UNKNOWN, so it cannot short-circuit: a later false still wins.
func appendFlattenedLazyArgs(out []Scmer, arg Scmer, operator string) ([]Scmer, bool) {
	inner, ok := scmerSlice(arg)
	if !ok || len(inner) <= 1 || !scmerIsSymbol(inner[0], operator) {
		return append(out, arg), false
	}
	changed := true
	for i := 1; i < len(inner); i++ {
		var nested bool
		out, nested = appendFlattenedLazyArgs(out, inner[i], operator)
		changed = changed || nested
	}
	return out, changed
}

func optimizeAnd(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	// Optimize left-to-right. Do not touch operands after a known false: doing
	// so would violate the special form's lazy evaluation contract. Flatten
	// each optimized result so child rewrites cannot reintroduce nested ANDs.
	out := make([]Scmer, 1, len(v))
	out[0] = v[0]
	var onlyType *TypeDescriptor
	for i := 1; i < len(v); i++ {
		arg, td := oc.OptimizeSub(v[i], true)
		flattened, _ := appendFlattenedLazyArgs(nil, arg, "and")
		before := len(out)
		for _, item := range flattened {
			switch item.GetTag() {
			case tagBool, tagInt, tagFloat, tagString:
				if ToBool(item) {
					continue
				}
				if len(out) == 1 {
					return NewBool(false), &TypeDescriptor{Transfer: true, Const: true}
				}
				return NewSlice(append(out, NewBool(false))), nil
			default:
				out = append(out, item)
			}
		}
		if len(out) == before+1 && len(flattened) == 1 {
			onlyType = td
		} else if len(out) > before {
			onlyType = nil
		}
	}
	if len(out) == 1 {
		return NewBool(true), &TypeDescriptor{Transfer: true, Const: true}
	}
	if len(out) == 2 {
		return out[1], onlyType
	}
	return NewSlice(out), nil
}

// optimizeOr mirrors the lazy SQL three-valued OR evaluator. Nil must remain
// until runtime because a later true wins over UNKNOWN.
func optimizeOr(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	out := make([]Scmer, 1, len(v))
	out[0] = v[0]
	var onlyType *TypeDescriptor
	for i := 1; i < len(v); i++ {
		arg, td := oc.OptimizeSub(v[i], true)
		flattened, _ := appendFlattenedLazyArgs(nil, arg, "or")
		before := len(out)
		for _, item := range flattened {
			switch item.GetTag() {
			case tagBool, tagInt, tagFloat, tagString:
				if !ToBool(item) {
					continue
				}
				if len(out) == 1 {
					return NewBool(true), &TypeDescriptor{Transfer: true, Const: true}
				}
				return NewSlice(append(out, NewBool(true))), nil
			default:
				out = append(out, item)
			}
		}
		if len(out) == before+1 && len(flattened) == 1 {
			onlyType = td
		} else if len(out) > before {
			onlyType = nil
		}
	}
	if len(out) == 1 {
		return NewBool(false), &TypeDescriptor{Transfer: true, Const: true}
	}
	if len(out) == 2 {
		return out[1], onlyType
	}
	return NewSlice(out), nil
}

// optimizeAssociative is the Optimize hook for associative operators (+ and *).
// It runs default optimization and then flattens nested same-operator calls.
func optimizeAssociative(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	result, td := oc.ApplyDefaultOptimization(v, useResult)
	if td != nil && td.Const {
		return result, td
	}
	// Flatten nested same-operator calls
	rv, ok := scmerSlice(result)
	if !ok || len(rv) <= 1 {
		return result, td
	}
	headName := ""
	if s, ok := scmerSymbol(rv[0]); ok {
		headName = string(s)
	}
	if headName == "" {
		return result, td
	}
	changed := false
	for i := 1; i < len(rv); i++ {
		if inner, ok := scmerSlice(rv[i]); ok && len(inner) > 1 && scmerIsSymbol(inner[0], headName) {
			newV := make([]Scmer, 0, len(rv)+len(inner)-2)
			newV = append(newV, rv[:i]...)
			newV = append(newV, inner[1:]...)
			newV = append(newV, rv[i+1:]...)
			rv = newV
			changed = true
			i--
		}
	}
	if changed {
		return NewSlice(rv), td
	}
	return result, td
}

func OptimizeMatchPattern(value Scmer, pattern Scmer, env *Env, ome *optimizerMetainfo, ome2 *optimizerMetainfo) Scmer {
	if stripped, ok := scmerStripSourceInfo(pattern); ok {
		pattern = stripped
	}

	if sym, ok := scmerSymbol(pattern); ok {
		delete(ome2.variableReplacement, sym)
		return pattern
	}

	if slice, ok := scmerSlice(pattern); ok {
		if len(slice) == 0 {
			return NewSlice(slice)
		}
		headSym, headOk := scmerSymbol(slice[0])
		if headOk && headSym == Symbol("eval") && len(slice) > 1 {
			slice[1], _ = OptimizeEx(slice[1], env, ome2, true)
			return NewSlice(slice)
		}
		if headOk && headSym == Symbol("var") && len(slice) == 2 {
			return NewNthLocalVar(NthLocalVar(ToInt(slice[1])))
		}
		if headOk && headSym == Symbol("regex") && len(slice) > 1 {
			// Precompile constant regex patterns at optimization time
			if slice[1].IsString() {
				patternStr := slice[1].String()
				re, err := regexp.Compile(patternStr)
				if err != nil {
					panic("invalid regex pattern: " + patternStr + ": " + err.Error())
				}
				slice[1] = NewRegex(re)
			}
			for i := 2; i < len(slice); i++ {
				slice[i] = OptimizeMatchPattern(NewNil(), slice[i], env, ome, ome2)
			}
			return NewSlice(slice)
		}
		for i := 1; i < len(slice); i++ {
			slice[i] = OptimizeMatchPattern(NewNil(), slice[i], env, ome, ome2)
		}
		return NewSlice(slice)
	}

	return pattern
}

func OptimizeParser(val Scmer, env *Env, ome *optimizerMetainfo, ignoreResult bool) Scmer {
	if val.IsSourceInfo() {
		sourceInfo := val.SourceInfo()
		sourceInfo.coverage = true
		val = sourceInfo.value
	} else if val.GetTag() == tagAny {
		if sourceInfo, ok := val.Any().(SourceInfo); ok {
			sourceInfo.coverage = true
			val = sourceInfo.value
		}
	}

	slice, ok := scmerSlice(val)
	if !ok || len(slice) == 0 {
		return val
	}

	headSym, headOk := scmerSymbol(slice[0])
	if headOk && headSym == Symbol("parser") {
		ign2 := ignoreResult
		if len(slice) > 2 {
			ign2 = true // result of parser can be ignored when expr is executed
		}
		ome2 := ome.Copy()
		slice[1] = OptimizeParser(slice[1], env, &ome2, ign2) // syntax expr -> collect new variables
		if len(slice) > 2 {
			slice[2], _ = OptimizeEx(slice[2], env, &ome2, !ignoreResult) // generator expr -> use variables
		}
		if len(slice) > 3 {
			slice[3], _ = OptimizeEx(slice[3], env, ome, true) // delimiter expr
		}
		val = NewSlice(slice)
	} else if headOk && headSym == Symbol("define") {
		slice[2] = OptimizeParser(slice[2], env, ome, false)
		if sym, ok := scmerSymbol(slice[1]); ok {
			if _, present := ome.variableReplacement[sym]; present {
				delete(ome.variableReplacement, sym)
			}
		}
		val = NewSlice(slice)
	} else if headOk && headSym == Symbol("capture") {
		// capture wrapper - optimize sub-parser but keep capture structure
		slice[1] = OptimizeParser(slice[1], env, ome, false)
		val = NewSlice(slice)
	} else {
		for i := 1; i < len(slice); i++ {
			slice[i] = OptimizeParser(slice[i], env, ome, ignoreResult)
		}
		val = NewSlice(slice)
	}

	p := parseSyntax(val, env, ome, ignoreResult)
	if p != nil {
		return NewAny(p)
	}
	return val
}

// deoptimizeExpr rewrites optimizer-produced special forms back to plain equivalents
// so that buildComputedFn lambdas do not depend on VarsNumbered slots beyond params.
// Currently handles:
//
//	(!list NthLocalVar(start) count expr...) -> (list expr...)
//	(!!list NthLocalVar(start) cap) -> (list)
func DeoptimizeExpr(expr Scmer) Scmer {
	if !expr.IsSlice() {
		return expr
	}
	items := expr.Slice()
	if len(items) >= 3 && items[0].IsSymbol() && items[0].String() == "!list" {
		count := int(ToInt(items[2]))
		if count == len(items)-3 {
			newItems := make([]Scmer, 1+count)
			newItems[0] = NewSymbol("list")
			for i := 0; i < count; i++ {
				newItems[1+i] = DeoptimizeExpr(items[3+i])
			}
			return NewSlice(newItems)
		}
	}
	if len(items) == 3 && items[0].IsSymbol() && items[0].String() == "!!list" && items[1].IsNthLocalVar() {
		return NewSlice([]Scmer{NewSymbol("list")})
	}
	// recurse into sub-expressions
	newItems := make([]Scmer, len(items))
	changed := false
	for i, item := range items {
		newItems[i] = DeoptimizeExpr(item)
		if !Equal(newItems[i], item) {
			changed = true
		}
	}
	if !changed {
		return expr
	}
	return NewSlice(newItems)
}
