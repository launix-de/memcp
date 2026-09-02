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

// SettingsTrackSourceCoverage preserves source wrappers until interpreter
// execution. It never marks the wrappers as covered itself.
var SettingsTrackSourceCoverage bool

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
	if len(items) > 0 && scmerIsSymbol(items[0], "quote") {
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

// OptimizeProcToSerialFunction is the compatibility adapter for callers which
// still require a Go variadic function. New physical hot paths must use
// PrepareSerialProc and caller-owned []Scmer frames: Eval's temporary variadic
// call arrays otherwise escape once per nested expression and row. The returned
// function MUST NEVER run on multiple threads simultaneously because its
// environment is reused to avoid allocations.
func OptimizeProcToSerialFunction(val Scmer) func(...Scmer) Scmer {
	if val.GetTag() == tagFunc {
		return val.Func()
	}
	if val.GetTag() == tagAny {
		if fn, ok := val.Any().(func(...Scmer) Scmer); ok {
			return fn
		}
	}
	borrowed := optimizeProcToSerialBorrowed(val)
	return func(args ...Scmer) Scmer { return borrowed(args) }
}

func optimizeProcToSerialBorrowed(val Scmer) func([]Scmer) Scmer {
	/* API contract:
	- the returned func must only be called with the correct number of declared parameters
	- thus we will perform no boundary checks
	- we enclose and share the environment over multiple runs, so the function must not be called simultaneously
	- for performance reason, we put as much checks and allocations out of the returned function and into our closure
	- TODO: we want to hook up the JIT here to produce some machine code for hotpaths
	*/
	if val.IsNil() {
		return func([]Scmer) Scmer { return NewNil() }
	}
	if val.GetTag() == tagFunc {
		fn := val.Func()
		return func(args []Scmer) Scmer { return fn(args...) }
	}
	if val.GetTag() == tagAny {
		if fn, ok := val.Any().(func(...Scmer) Scmer); ok {
			return func(args []Scmer) Scmer { return fn(args...) }
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
		return func([]Scmer) Scmer { return captured }
	}
	p := *proc
	// A Proc keeps its source body even after native compilation. Execute the
	// attached entry point now; future scan specialization may recompile the same
	// filter/map/reduce body against concrete storage and column types first.
	if p.Compiled != nil {
		return func(args []Scmer) Scmer {
			return p.Compiled.Call(args...)
		}
	}

	// constant body
	switch p.Body.GetTag() {
	case tagNil, tagBool, tagInt, tagFloat, tagString:
		constant := p.Body
		return func([]Scmer) Scmer { return constant }
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
			return func(args []Scmer) Scmer {
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
					return func(args []Scmer) Scmer {
						return args[idx]
					}
				}
			}
		}
	}

	numVars := p.NumVars
	// Numbered-only optimized lambdas carry their complete frame size as the
	// fourth lambda item. Walking a large generated callback again at every
	// adapter creation duplicates optimizer work and can dominate compilation.
	// Hand-built and named procedures retain the compatibility scan.
	if numVars == 0 || !p.NumberedOnly {
		if required := requiredNumberedSlots(p.Body); required > numVars {
			numVars = required
		}
	}
	var vars Vars
	en := &Env{Vars: vars, VarsNumbered: make([]Scmer, numVars), Outer: p.En, Nodefine: false}
	body := prepareSerialExpr(&p, p.Body)
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
			return func(args []Scmer) Scmer {
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
				return body(en)
			}
		}
		vars = make(Vars, len(paramSlice))
		en.Vars = vars
		return func(args []Scmer) Scmer {
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
			return body(en)
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
			return func(args []Scmer) Scmer {
				argsList := NewSlice(append([]Scmer(nil), args...))
				en.VarsNumbered[0] = argsList
				if bindNamed {
					en.Vars[sym] = argsList
				}
				return body(en)
			}
		}
		vars = make(Vars, 1)
		en.Vars = vars
		return func(args []Scmer) Scmer {
			en.Vars[sym] = NewSlice(append([]Scmer(nil), args...))
			return body(en)
		}
	}
	return func(args []Scmer) Scmer {
		return body(en)
	}
}

// Optimize consumes val, preprocesses and optimizes it, and transfers ownership
// to the returned value. It may therefore reuse val's storage. When
// telemetryCallback is non-nil, it is called exactly once after the optimizer
// has finished. The callback itself is excluded from compile_ns.
func Optimize(val Scmer, env *Env, telemetryCallback func(Scmer)) Scmer {
	var started time.Time
	if telemetryCallback != nil {
		started = time.Now()
	}
	ome := newOptimizerMetainfo()
	// Recursive hook guards still need the original tree size. This walk goes
	// away with OptimizeRewrite; it is not performed for telemetry alone.
	ome.rewrite.inputNodes = optimizerNodeCount(val)
	v, _ := OptimizeEx(val, env, &ome, true)
	if telemetryCallback != nil {
		compileNS := time.Since(started).Nanoseconds()
		telemetryCallback(NewSlice([]Scmer{
			NewString("compile_ns"), NewInt(compileNS),
			NewString("input_nodes"), NewInt(int64(ome.rewrite.inputNodes)),
			NewString("output_nodes"), NewInt(int64(optimizerNodeCount(v))),
			NewString("rewrites"), NewInt(int64(ome.rewrite.rewrites)),
			NewString("rejected_rewrites"), NewInt(int64(ome.rewrite.rejected)),
			NewString("budget_remaining"), NewInt(int64(ome.rewrite.remainingBudget)),
			NewString("callback_analyses"), NewInt(int64(ome.rewrite.callbackAnalyses)),
			NewString("callback_clones"), NewInt(int64(ome.rewrite.callbackClones)),
		}))
	}
	return v
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

type optimizerMetainfo struct {
	variableReplacement       map[Symbol]optimizerReplacement
	variableTypes             map[Symbol]*TypeDescriptor
	numberedTypes             map[NthLocalVar]*TypeDescriptor
	ownedLocalBindings        map[Symbol]bool
	setBlacklist              []Symbol
	nextSlot                  *int // pointer to lambda's slot counter; nil outside lambda
	pendingCallbackParams     []*TypeDescriptor
	pendingCallbackReturn     *TypeDescriptor // structured escape information for the next lambda result
	loopDepth                 int             // >0 inside scan/reduce callbacks; prevents hoisted defines from being inlined back into loops
	lambdaDepth               int             // >0 while optimizing a lambda body; keeps local definitions out of Env hints
	beginDepth                int             // >0 in lexical begin scopes; their definitions do not reach the caller Env
	inlineDepth               int
	inlineStack               map[Symbol]bool
	specializationStack       map[procSpecializationStackKey]bool
	specializationParamMask   uint64
	specializationDepth       int
	specializationUsed        *bool
	specializationNestedUsed  *bool
	specializationRootMutUsed *bool
	specializationOwnedVars   map[Symbol]bool
	specializationOwnedSlots  map[NthLocalVar]bool
	rewrite                   *optimizerRewriteState
	captureArgumentTypes      bool
	argumentTypes             []TypeInfo
}

type optimizerReplacement struct {
	value      Scmer
	outerDepth int
}

func newOptimizerMetainfo() (result optimizerMetainfo) {
	result.variableReplacement = make(map[Symbol]optimizerReplacement)
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

func cloneTypeDescriptor(td *TypeDescriptor, cloned map[*TypeDescriptor]*TypeDescriptor) *TypeDescriptor {
	if td == nil {
		return nil
	}
	if result, exists := cloned[td]; exists {
		return result
	}
	result := *td
	cloned[td] = &result
	if len(td.Params) > 0 {
		result.Params = make([]*TypeDescriptor, len(td.Params))
		for i, param := range td.Params {
			result.Params[i] = cloneTypeDescriptor(param, cloned)
		}
	}
	result.Return = cloneTypeDescriptor(td.Return, cloned)
	if len(td.Keys) > 0 {
		result.Keys = make(map[string]*TypeDescriptor, len(td.Keys))
		for key, child := range td.Keys {
			result.Keys[key] = cloneTypeDescriptor(child, cloned)
		}
	}
	result.Element = cloneTypeDescriptor(td.Element, cloned)
	return &result
}

func immutableTypeInfo(info TypeInfo) TypeInfo {
	info.Extra = cloneTypeDescriptor(info.Extra, make(map[*TypeDescriptor]*TypeDescriptor))
	return info
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

func descriptorProjection(td *TypeDescriptor, key string) *TypeDescriptor {
	if td == nil {
		return nil
	}
	if child := td.Keys[key]; child != nil {
		return child
	}
	if td.Element != nil {
		return td.Element
	}
	return nil
}

func descriptorKey(td *TypeDescriptor, key string) *TypeDescriptor {
	return copyTypeDescriptor(descriptorProjection(td, key))
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
	if base == nil {
		return returnType
	}
	if projected := descriptorProjection(base, key); projected != nil {
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
			td = unknownOptimizerParameterType
		}
		child.variableTypes[sym] = td
		if replacement, ok := child.variableReplacement[sym]; ok && replacement.outerDepth == 0 && replacement.value.IsNthLocalVar() {
			child.numberedTypes[replacement.value.NthLocalVar()] = td
		}
	}
	ome.pendingCallbackParams = nil
}

// LoopDepth returns the current loop nesting depth.
func (ome *optimizerMetainfo) LoopDepth() int { return ome.loopDepth }
func (ome *optimizerMetainfo) Copy() (result optimizerMetainfo) {
	result.variableReplacement = make(map[Symbol]optimizerReplacement)
	result.variableTypes = make(map[Symbol]*TypeDescriptor)
	result.numberedTypes = make(map[NthLocalVar]*TypeDescriptor)
	for k, replacement := range ome.variableReplacement {
		replacement.outerDepth++
		result.variableReplacement[k] = replacement
	}
	result.setBlacklist = ome.setBlacklist
	result.loopDepth = ome.loopDepth
	result.lambdaDepth = ome.lambdaDepth
	result.beginDepth = ome.beginDepth
	result.inlineDepth = ome.inlineDepth
	result.inlineStack = ome.inlineStack
	result.specializationStack = ome.specializationStack
	result.specializationParamMask = ome.specializationParamMask
	result.specializationDepth = ome.specializationDepth
	result.specializationUsed = ome.specializationUsed
	result.specializationNestedUsed = ome.specializationNestedUsed
	result.specializationRootMutUsed = ome.specializationRootMutUsed
	result.rewrite = ome.rewrite
	// nextSlot is NOT propagated across lambda boundaries (each lambda has its own)
	return
}

// CopySharedScope is like Copy but for scopes that share VarsNumbered with
// their parent (begin, match). NthLocalVar entries are kept as-is instead of
// being wrapped in (outer ...) since they access the same VarsNumbered array.
func (ome *optimizerMetainfo) CopySharedScope() (result optimizerMetainfo) {
	result.variableReplacement = make(map[Symbol]optimizerReplacement)
	result.variableTypes = ome.variableTypes
	result.numberedTypes = ome.numberedTypes
	result.ownedLocalBindings = ome.ownedLocalBindings
	for k, replacement := range ome.variableReplacement {
		if replacement.outerDepth == 0 && replacement.value.IsNthLocalVar() {
			result.variableReplacement[k] = replacement
		} else {
			replacement.outerDepth++
			result.variableReplacement[k] = replacement
		}
	}
	result.setBlacklist = ome.setBlacklist
	result.nextSlot = ome.nextSlot // shared scope shares VarsNumbered
	result.loopDepth = ome.loopDepth
	result.lambdaDepth = ome.lambdaDepth
	result.beginDepth = ome.beginDepth
	result.inlineDepth = ome.inlineDepth
	result.inlineStack = ome.inlineStack
	result.specializationStack = ome.specializationStack
	result.specializationParamMask = ome.specializationParamMask
	result.specializationDepth = ome.specializationDepth
	result.specializationUsed = ome.specializationUsed
	result.specializationNestedUsed = ome.specializationNestedUsed
	result.specializationRootMutUsed = ome.specializationRootMutUsed
	result.specializationOwnedVars = ome.specializationOwnedVars
	result.specializationOwnedSlots = ome.specializationOwnedSlots
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
		case callee, "define", "set", "setN", "eval", "parser", "outer", "begin", "begin_mut", "!begin", "match", "match_mut":
			return false
		case "lambda":
			// A nested lambda is an opaque constant for this substitution. It is
			// safe only when it does not capture the surrounding Proc frame.
			return !expressionContainsOuterReference(expr)
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
	if !ok || len(items) == 0 || scmerIsSymbol(items[0], "quote") || scmerIsSymbol(items[0], "lambda") {
		return expr
	}
	rewritten := make([]Scmer, len(items))
	for i, item := range items {
		rewritten[i] = substituteLeafInlineParams(item, args)
	}
	return NewSlice(rewritten)
}

func expressionContainsOuterReference(expr Scmer) bool {
	var pending [maxLeafInlineNodes]Scmer
	var seen [maxLeafInlineNodes]Scmer
	pending[0] = expr
	pendingCount := 1
	seenCount := 0
	for pendingCount > 0 {
		pendingCount--
		current := pending[pendingCount]
		if stripped, ok := scmerStripSourceInfo(current); ok {
			current = stripped
		}
		visited := false
		for i := 0; i < seenCount; i++ {
			if seen[i] == current {
				visited = true
				break
			}
		}
		if visited {
			continue
		}
		if seenCount >= len(seen) {
			return true
		}
		seen[seenCount] = current
		seenCount++
		items, ok := scmerSlice(current)
		if !ok || len(items) == 0 || items[0].SymbolEquals("quote") {
			continue
		}
		if items[0].SymbolEquals("outer") {
			return true
		}
		if len(items) > len(pending)-pendingCount {
			return true
		}
		for _, item := range items {
			pending[pendingCount] = item
			pendingCount++
		}
	}
	return false
}

func leafInlineBindingsStable(expr Scmer, source, target *Env) bool {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	// Special forms replace unbound syntax symbols. Preserve the old inliner
	// boundary so ownership-aware Proc specialization still gets first use of
	// calls containing lazy selectors such as coalesceNil.
	if expr.GetTag() == tagSpecialForm {
		return false
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
		if (sourceOK && sourceValue.GetTag() == tagSpecialForm) || (targetOK && targetValue.GetTag() == tagSpecialForm) {
			return false
		}
		return sourceOK && targetOK && sourceValue == targetValue
	}
	items := expr.Slice()
	if len(items) == 0 || scmerIsSymbol(items[0], "quote") {
		return true
	}
	if scmerIsSymbol(items[0], "lambda") {
		return source == target && !expressionContainsOuterReference(expr)
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
	args := v[1:]
	params := proc.Params
	if stripped, ok := scmerStripSourceInfo(params); ok {
		params = stripped
	}
	paramItems, ok := scmerSlice(params)
	if !ok || len(paramItems) != len(args) || proc.NumVars != len(paramItems) {
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
	inlined := substituteLeafInlineParams(proc.Body, args)
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

type procSpecializationStackKey struct {
	meta *ProcOptimizerMeta
	key  procSpecializationKey
}

func markProcOwnershipSpecializationUse(ome *optimizerMetainfo, expr Scmer) {
	if ome.specializationUsed == nil || ome.lambdaDepth != ome.specializationDepth {
		return
	}
	mask := procOwnershipParameterMask(expr, 64)
	if mask&ome.specializationParamMask != 0 {
		*ome.specializationUsed = true
	}
}

func markNestedProcOwnershipSpecializationUse(ome *optimizerMetainfo, expr Scmer) {
	if ome.specializationNestedUsed == nil || ome.lambdaDepth != ome.specializationDepth {
		return
	}
	if nestedProcOwnershipExpression(ome, expr) {
		*ome.specializationNestedUsed = true
	}
}

func nestedProcOwnershipExpression(ome *optimizerMetainfo, expr Scmer) bool {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	if expr.IsNthLocalVar() && ome.specializationOwnedSlots[expr.NthLocalVar()] {
		return true
	}
	if sym, ok := scmerSymbol(expr); ok && ome.specializationOwnedVars[sym] {
		return true
	}
	items, ok := scmerSlice(expr)
	if !ok || len(items) < 2 {
		return false
	}
	if scmerIsSymbol(items[0], "car") || scmerIsSymbol(items[0], "cadr") || scmerIsSymbol(items[0], "nth") || scmerIsSymbol(items[0], "get_assoc") {
		if procOwnershipParameterMask(items[1], 64)&ome.specializationParamMask != 0 {
			return true
		}
		return nestedProcOwnershipExpression(ome, items[1])
	}
	if scmerIsSymbol(items[0], "coalesce") || scmerIsSymbol(items[0], "coalesceNil") {
		for _, item := range items[1:] {
			if nestedProcOwnershipExpression(ome, item) {
				return true
			}
		}
	}
	return false
}

func markRootProcOwnershipMutationUse(ome *optimizerMetainfo, expr Scmer) {
	if ome.specializationRootMutUsed == nil || ome.lambdaDepth != ome.specializationDepth {
		return
	}
	mask := procOwnershipParameterMask(expr, 64)
	if mask&ome.specializationParamMask != 0 {
		*ome.specializationRootMutUsed = true
	}
}

// procOwnershipParameterMask follows expressions which return one of their
// arguments unchanged. Ownership can cross such a selector only when the
// selector's merged return type proves every possible result transferable.
func procOwnershipParameterMask(expr Scmer, paramCount int) uint64 {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	if expr.IsNthLocalVar() {
		idx := uint(expr.NthLocalVar())
		if idx < uint(paramCount) && idx < 64 {
			return 1 << idx
		}
		return 0
	}
	items, ok := scmerSlice(expr)
	if !ok || len(items) < 2 || (!scmerIsSymbol(items[0], "coalesce") && !scmerIsSymbol(items[0], "coalesceNil")) {
		return 0
	}
	var mask uint64
	for _, item := range items[1:] {
		mask |= procOwnershipParameterMask(item, paramCount)
	}
	return mask
}

func procParameterOwnershipUses(expr Scmer, paramCount int) ([]int, []bool, []bool) {
	uses := make([]int, paramCount)
	captured := make([]bool, paramCount)
	consumed := make([]bool, paramCount)
	var visit func(Scmer, int, bool)
	visit = func(current Scmer, lambdaDepth int, throughOuter bool) {
		if stripped, ok := scmerStripSourceInfo(current); ok {
			current = stripped
		}
		if current.IsNthLocalVar() {
			idx := int(current.NthLocalVar())
			if idx >= 0 && idx < paramCount && (lambdaDepth == 0 || throughOuter) {
				uses[idx]++
				if lambdaDepth > 0 {
					captured[idx] = true
				}
			}
			return
		}
		items, ok := scmerSlice(current)
		if !ok || len(items) == 0 || scmerIsSymbol(items[0], "quote") {
			return
		}
		if lambdaDepth == 0 && len(items) > 1 {
			consumesFirst := scmerIsSymbol(items[0], "match") || scmerIsSymbol(items[0], "match_mut")
			if declaration := DeclarationForValue(items[0]); declaration != nil && declaration.Type != nil {
				consumesFirst = consumesFirst || declaration.OptimizeFirstArgTransfer
			}
			if consumesFirst {
				mask := procOwnershipParameterMask(items[1], paramCount)
				for idx := 0; idx < paramCount && idx < 64; idx++ {
					if mask&(1<<uint(idx)) != 0 {
						consumed[idx] = true
					}
				}
			}
		}
		if scmerIsSymbol(items[0], "lambda") {
			if len(items) > 2 {
				visit(items[2], lambdaDepth+1, false)
			}
			return
		}
		if scmerIsSymbol(items[0], "outer") {
			for _, item := range items[1:] {
				visit(item, lambdaDepth, true)
			}
			return
		}
		for _, item := range items {
			visit(item, lambdaDepth, throughOuter)
		}
	}
	visit(expr, 0, false)
	return uses, captured, consumed
}

func specializationOwnershipDescriptor(td *TypeDescriptor) *TypeDescriptor {
	if td == nil {
		return &TypeDescriptor{Kind: "any", Length: UnknownLength}
	}
	result := &TypeDescriptor{
		Kind:     "any",
		Transfer: td.Transfer && !td.Const,
		Length:   UnknownLength,
	}
	element := specializationOwnershipDescriptorOrNil(td.Element)
	elementOwned := descriptorContainsOwnership(element)
	if elementOwned {
		result.Element = element
	}
	if len(td.Keys) > 0 {
		result.Keys = make(map[string]*TypeDescriptor, len(td.Keys))
		for key, child := range td.Keys {
			childOwnership := specializationOwnershipDescriptor(child)
			// A borrowed leaf is indistinguishable from missing information unless
			// it must override an owned Element fallback.
			if elementOwned || descriptorContainsOwnership(childOwnership) {
				result.Keys[key] = childOwnership
			}
		}
		if len(result.Keys) == 0 {
			result.Keys = nil
		}
	}
	return result
}

func specializationOwnershipDescriptorOrNil(td *TypeDescriptor) *TypeDescriptor {
	if td == nil {
		return nil
	}
	return specializationOwnershipDescriptor(td)
}

func descriptorContainsOwnership(td *TypeDescriptor) bool {
	if td == nil {
		return false
	}
	if td.Transfer || descriptorContainsOwnership(td.Element) {
		return true
	}
	for _, child := range td.Keys {
		if descriptorContainsOwnership(child) {
			return true
		}
	}
	return false
}

func specializationHashText(seed uint64, value string) uint64 {
	result := seed
	for i := 0; i < len(value); i++ {
		result ^= uint64(value[i])
		result *= 1099511628211
	}
	return result
}

func specializationShapeHash(td *TypeDescriptor) (uint64, uint64) {
	if td == nil {
		return 0x243f6a8885a308d3, 0x13198a2e03707344
	}
	lo := specializationHashText(1469598103934665603, td.Kind)
	hi := specializationHashText(1099511628211, td.Kind)
	if td.Transfer {
		lo = combineStructuralHash(lo, 1)
		hi = combineStructuralHash(hi, 0x9e3779b97f4a7c15)
	}
	if td.CallsOnce {
		lo = combineStructuralHash(lo, 2)
		hi = combineStructuralHash(hi, 0xbf58476d1ce4e5b9)
	}
	if td.NoEscape {
		lo = combineStructuralHash(lo, 3)
		hi = combineStructuralHash(hi, 0x94d049bb133111eb)
	}
	if td.Const {
		lo = combineStructuralHash(lo, 4)
		hi = combineStructuralHash(hi, 0xd6e8feb86659fd93)
	}
	lo = combineStructuralHash(lo, uint64(td.Length))
	hi = combineStructuralHash(hi, uint64(td.Length)^0xa4093822299f31d0)
	for i, param := range td.Params {
		paramLo, paramHi := specializationShapeHash(param)
		lo = combineStructuralHash(lo, combineStructuralHash(uint64(i), paramLo))
		hi = combineStructuralHash(hi, combineStructuralHash(uint64(i), paramHi))
	}
	returnLo, returnHi := specializationShapeHash(td.Return)
	lo = combineStructuralHash(lo, returnLo)
	hi = combineStructuralHash(hi, returnHi)
	elementLo, elementHi := specializationShapeHash(td.Element)
	lo = combineStructuralHash(lo, elementLo)
	hi = combineStructuralHash(hi, elementHi)
	// TypeDescriptor.Keys is a map. Fold separately hashed entries using two
	// commutative accumulators so identical descriptors produce identical keys
	// without sorting or allocating a temporary key slice.
	var keysXor, keysSum, keysXorHi, keysSumHi uint64
	for key, child := range td.Keys {
		childLo, childHi := specializationShapeHash(child)
		keyLo := combineStructuralHash(specializationHashText(1469598103934665603, key), childLo)
		keyHi := combineStructuralHash(specializationHashText(1099511628211, key), childHi)
		keysXor ^= keyLo
		keysSum += keyLo * 0x9e3779b97f4a7c15
		keysXorHi ^= keyHi
		keysSumHi += keyHi * 0xbf58476d1ce4e5b9
	}
	lo = combineStructuralHash(combineStructuralHash(lo, keysXor), keysSum)
	hi = combineStructuralHash(combineStructuralHash(hi, keysXorHi), keysSumHi)
	return lo, hi
}

func procSpecializationKeyFromParams(mask uint64, paramTypes []*TypeDescriptor) procSpecializationKey {
	shapeLo := uint64(0x6a09e667f3bcc909)
	shapeHi := uint64(0xbb67ae8584caa73b)
	for i, paramType := range paramTypes {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		paramLo, paramHi := specializationShapeHash(paramType)
		shapeLo = combineStructuralHash(shapeLo, combineStructuralHash(uint64(i), paramLo))
		shapeHi = combineStructuralHash(shapeHi, combineStructuralHash(uint64(i), paramHi))
	}
	return procSpecializationKey{paramMask: mask, shapeLo: shapeLo, shapeHi: shapeHi}
}

func procSpecializationHasNestedOwnership(paramTypes []*TypeDescriptor) bool {
	for _, paramType := range paramTypes {
		if paramType != nil && (descriptorContainsOwnership(paramType.Element) || len(paramType.Keys) > 0) {
			return true
		}
	}
	return false
}

func procSpecializationKeyForArguments(proc *Proc, argTypes []TypeInfo) (procSpecializationKey, []*TypeDescriptor, bool) {
	params, ok := scmerSlice(proc.Params)
	if !ok || len(params) == 0 || len(params) > 64 || proc.NumVars < len(params) {
		return procSpecializationKey{}, nil, false
	}
	var ownershipCandidates uint64
	var callableCandidates uint64
	for i := range params {
		argIndex := i + 1
		if argIndex >= len(argTypes) {
			continue
		}
		argType := argTypes[argIndex]
		if argType.Kind() == KindFunc && argType.Extra != nil {
			callableCandidates |= 1 << uint(i)
		}
		// Transfer on scalar values only describes a fresh result; there is no
		// mutable ownership for the callee to consume. Specializing those values
		// would turn transient scalar type facts into an invalid calling contract.
		mutableKind := argType.Kind() == KindList || argType.Kind() == KindAssoc
		if !mutableKind || !argType.Transfer() || argType.Const() {
			continue
		}
		ownershipCandidates |= 1 << uint(i)
	}
	if ownershipCandidates|callableCandidates == 0 {
		return procSpecializationKey{}, nil, false
	}
	mask := callableCandidates
	if ownershipCandidates != 0 {
		uses, captured, consumed := procParameterOwnershipUses(proc.Body, len(params))
		for i := range params {
			if ownershipCandidates&(1<<uint(i)) == 0 {
				continue
			}
			// Transfer is linear. A specialized body may consume a parameter only
			// when that Proc frame has exactly one non-captured use of the value.
			if uses[i] == 1 && !captured[i] && consumed[i] {
				mask |= 1 << uint(i)
			}
		}
	}
	if mask == 0 {
		return procSpecializationKey{}, nil, false
	}
	paramTypes := make([]*TypeDescriptor, len(params))
	for i := range params {
		paramTypes[i] = &TypeDescriptor{Kind: "any", Length: UnknownLength}
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		if callableCandidates&(1<<uint(i)) != 0 {
			paramTypes[i] = cloneTypeDescriptor(argTypes[i+1].ToTypeDescriptor(), make(map[*TypeDescriptor]*TypeDescriptor))
			paramTypes[i].Transfer = false
			paramTypes[i].Const = false
			paramTypes[i].NoEscape = false
		} else {
			paramTypes[i] = specializationOwnershipDescriptor(argTypes[i+1].ToTypeDescriptor())
		}
	}
	return procSpecializationKeyFromParams(mask, paramTypes), paramTypes, true
}

func buildProcSpecialization(proc *Proc, key procSpecializationKey, paramTypes []*TypeDescriptor, stack map[procSpecializationStackKey]bool) (Scmer, bool, bool) {
	paramsValue := proc.Params
	if stripped, ok := scmerStripSourceInfo(paramsValue); ok {
		paramsValue = stripped
	}
	params := paramsValue.Slice()
	if len(paramTypes) != len(params) {
		return NewNil(), false, false
	}
	lambda := []Scmer{
		NewSymbol("lambda"),
		CloneOptimizerExpression(proc.Params),
		deoptimizeProcSpecializationExpr(CloneOptimizerExpression(proc.Body), len(params)),
	}
	if proc.NumVars > 0 {
		lambda = append(lambda, NewInt(int64(proc.NumVars)))
	}
	meta := newOptimizerMetainfo()
	used := false
	nestedUsed := false
	rootMutationUsed := false
	meta.pendingCallbackParams = paramTypes
	meta.specializationStack = stack
	meta.specializationParamMask = key.paramMask
	meta.specializationDepth = meta.lambdaDepth + 1
	meta.specializationUsed = &used
	meta.specializationNestedUsed = &nestedUsed
	meta.specializationRootMutUsed = &rootMutationUsed
	optimized, _ := OptimizeEx(NewSlice(lambda), proc.En, &meta, true)
	parts, ok := scmerSlice(optimized)
	if !ok || len(parts) < 3 || !scmerIsSymbol(parts[0], "lambda") {
		return NewNil(), nestedUsed, rootMutationUsed
	}
	specialized := *proc
	specialized.Params = parts[1]
	specialized.Body = parts[2]
	if len(parts) > 3 {
		specialized.NumVars = int(ToInt(parts[3]))
	}
	specialized.NumberedOnly = procCanUseNumberedOnly(specialized.Params, specialized.Body, specialized.NumVars)
	// A fresh argument alone is not sufficient reason to replace a named call
	// with an anonymous Proc literal. Publish a variant only when an ownership-
	// aware rewrite actually consumes one of the specialized parameters.
	if !used {
		return NewNil(), nestedUsed, rootMutationUsed
	}
	// Machine code belongs to the exact optimized body. Never inherit the
	// generic Proc's entry point when publishing a specialized body.
	specialized.Compiled = nil
	specialized.OptimizerMeta = &ProcOptimizerMeta{
		Return:    proc.OptimizerMeta.Return,
		HasReturn: proc.OptimizerMeta.HasReturn,
		Sequence:  proc.OptimizerMeta.Sequence,
	}
	variant := NewProcStruct(specialized)
	if proc.Compiled != nil {
		variant = jitCompileMode(proc.Compiled.RecursiveLambdas, variant)
	}
	return variant, nestedUsed, rootMutationUsed
}

func trySpecializeProcCall(v []Scmer, argTypes []TypeInfo, env *Env, ome *optimizerMetainfo) (Scmer, bool) {
	if len(v) < 2 {
		return NewNil(), false
	}
	hasTypeCandidate := false
	for _, argType := range argTypes[1:] {
		mutableKind := argType.Kind() == KindList || argType.Kind() == KindAssoc
		if (mutableKind && argType.Transfer() && !argType.Const()) || (argType.Kind() == KindFunc && argType.Extra != nil) {
			hasTypeCandidate = true
			break
		}
	}
	if !hasTypeCandidate {
		return NewNil(), false
	}
	callee, ok := scmerSymbol(v[0])
	if !ok {
		return NewNil(), false
	}
	owner := env.FindRead(callee)
	if owner == nil {
		return NewNil(), false
	}
	value, exists := owner.Vars[callee]
	if !exists || !value.IsProc() {
		return NewNil(), false
	}
	proc := value.Proc()
	if proc == nil || proc.En == nil || proc.OptimizerMeta == nil {
		return NewNil(), false
	}
	key, paramTypes, ok := procSpecializationKeyForArguments(proc, argTypes)
	if !ok {
		return NewNil(), false
	}
	return getProcSpecialization(proc, key, paramTypes, ome)
}

func getProcSpecialization(proc *Proc, key procSpecializationKey, paramTypes []*TypeDescriptor, ome *optimizerMetainfo) (Scmer, bool) {
	stackKey := procSpecializationStackKey{meta: proc.OptimizerMeta, key: key}
	if ome.specializationStack != nil && ome.specializationStack[stackKey] {
		return NewNil(), false
	}
	if variant, exists := proc.OptimizerMeta.specialization(key); exists {
		return variant, true
	}
	if proc.OptimizerMeta.specializationRejected(key) {
		return NewNil(), false
	}
	build, compile := proc.OptimizerMeta.beginSpecialization(key)
	if !compile {
		if build != nil {
			<-build.done
		}
		variant, exists := proc.OptimizerMeta.specialization(key)
		return variant, exists
	}
	if ome.specializationStack == nil {
		ome.specializationStack = make(map[procSpecializationStackKey]bool)
	}
	ome.specializationStack[stackKey] = true
	variant := NewNil()
	defer func() {
		delete(ome.specializationStack, stackKey)
		proc.OptimizerMeta.finishSpecialization(key, variant)
	}()
	var nestedUsed, rootMutationUsed bool
	variant, nestedUsed, rootMutationUsed = buildProcSpecialization(proc, key, paramTypes, ome.specializationStack)
	if procSpecializationHasNestedOwnership(paramTypes) && !nestedUsed && !rootMutationUsed {
		variant = NewNil()
	}
	return variant, variant.IsProc()
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

func materializeOptimizerReplacement(replacement optimizerReplacement) Scmer {
	if replacement.outerDepth == 0 {
		return replacement.value
	}
	return NewSlice([]Scmer{NewSymbol("outer"), NewInt(int64(replacement.outerDepth)), replacement.value})
}

// rebaseOuterReferencesAfterScopeRemoval adjusts explicit lexical addresses
// when an enclosing scope is removed. References which still resolve inside
// expression keep their depth; only references crossing expression's root lose
// one hop. Quoted syntax is data and match fallback arms do not enter the
// branch-local environment.
func rebaseOuterReferencesAfterScopeRemoval(expression Scmer, innerDepth int) Scmer {
	if expression.IsSourceInfo() {
		source := *expression.SourceInfo()
		source.value = rebaseOuterReferencesAfterScopeRemoval(source.value, innerDepth)
		return NewSourceInfo(source)
	}
	items, ok := scmerSlice(expression)
	if !ok || len(items) == 0 || scmerIsSymbol(items[0], "quote") {
		return expression
	}
	if scmerIsSymbol(items[0], "outer") && len(items) == 3 {
		depth, valid := outerDepthLiteral(items[1])
		if valid && depth > int64(innerDepth) {
			return NewSlice([]Scmer{items[0], NewInt(depth - 1), items[2]})
		}
		return expression
	}

	rewritten := make([]Scmer, len(items))
	copy(rewritten, items)
	switch {
	case scmerIsSymbol(items[0], "lambda"):
		if len(items) > 2 {
			rewritten[2] = rebaseOuterReferencesAfterScopeRemoval(items[2], innerDepth+1)
		}
	case scmerIsSymbol(items[0], "begin"):
		for i := 1; i < len(items); i++ {
			rewritten[i] = rebaseOuterReferencesAfterScopeRemoval(items[i], innerDepth+1)
		}
	case scmerIsSymbol(items[0], "begin_mut"):
		if len(items) > 1 {
			rewritten[1] = rebaseOuterReferencesAfterScopeRemoval(items[1], innerDepth)
		}
		for i := 2; i < len(items); i++ {
			rewritten[i] = rebaseOuterReferencesAfterScopeRemoval(items[i], innerDepth+1)
		}
	case scmerIsSymbol(items[0], "match") || scmerIsSymbol(items[0], "match_mut"):
		if len(items) > 1 {
			rewritten[1] = rebaseOuterReferencesAfterScopeRemoval(items[1], innerDepth)
		}
		for i := 3; i < len(items); i += 2 {
			rewritten[i] = rebaseOuterReferencesAfterScopeRemoval(items[i], innerDepth+1)
		}
		if len(items)%2 == 1 {
			rewritten[len(items)-1] = rebaseOuterReferencesAfterScopeRemoval(items[len(items)-1], innerDepth)
		}
	default:
		for i, item := range items {
			rewritten[i] = rebaseOuterReferencesAfterScopeRemoval(item, innerDepth)
		}
	}
	return NewSlice(rewritten)
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
		return v.SourceInfo().value, true
	}
	if v.GetTag() == tagAny {
		if si, ok := v.Any().(SourceInfo); ok {
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
	case tagBSON:
		return val, TypeInfo{kind: KindAny, flags: FlagTransfer | FlagConst, length: UnknownLength}
	case tagFunc:
		if td := val.CallableType(); td != nil {
			return val, TypeInfoFromTD(td)
		}
		return val, tiZero
	case tagSymbol:
		sym := mustSymbol(val)
		varType := ome.variableTypes[sym]
		if replacement, ok := ome.variableReplacement[sym]; ok {
			if replacement.outerDepth == 0 && replacement.value.IsSymbol() && mustSymbol(replacement.value) == sym {
				if varType != nil {
					return val, TypeInfoFromTD(varType)
				}
				return val, tiZero
			}
			if replacement.outerDepth > 0 {
				if s2, ok := scmerSymbol(replacement.value); ok && s2 == sym {
					if varType != nil {
						return val, TypeInfoFromTD(varType)
					}
					return val, tiZero
				}
			}
			result, ti := OptimizeEx(materializeOptimizerReplacement(replacement), env, ome, useResult)
			if varType != nil {
				ti = TypeInfoFromTD(varType)
			}
			return result, ti
		}
		if varType != nil {
			return val, TypeInfoFromTD(varType)
		}
		if binding := env.FindRead(sym); binding != nil {
			if bound, exists := binding.Vars[sym]; exists {
				if td := bound.CallableType(); td != nil {
					return val, TypeInfoFromTD(td)
				}
			}
		}
		return val, tiZero
	case tagSlice:
		return optimizeList(val.Slice(), env, ome, useResult)
	case tagSourceInfo:
		siPtr := val.SourceInfo()
		if SettingsTrackSourceCoverage {
			result, ti := OptimizeEx(siPtr.value, env, ome, useResult)
			if ti.Const() {
				return result, ti
			}
			siPtr.value = result
			return val, ti.WithoutTransfer()
		}
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
	tiConstTransfer               = TypeInfo{flags: FlagTransfer | FlagConst, length: UnknownLength}
	tiTransfer                    = TypeInfo{flags: FlagTransfer, length: UnknownLength}
	tiZero                        = TypeInfo{length: UnknownLength}
	unknownOptimizerParameterType = &TypeDescriptor{Kind: "any", Length: UnknownLength}
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
	case tagNil, tagBool, tagInt, tagFloat, tagString, tagBSON:
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
	defineIdx       int
	firstUse        int
	count           int
	useCount        int
	used            bool
	captured        bool
	repeatedCapture bool
	leafLambda      bool
	sinkRegion      *conditionalSinkRegion
	sinkRegionSeen  bool
	sinkConflict    bool
}

// conditionalSinkRegion represents the first exclusive branch below a begin
// body. Nested conditionals retain the same region, keeping common-region
// tracking O(1) per use instead of repeatedly comparing ancestor paths.
type conditionalSinkRegion struct {
	target     *Scmer
	hasBinding bool
}

func optimizerCallType(call []Scmer, env *Env, ome *optimizerMetainfo) *TypeDescriptor {
	if len(call) == 0 {
		return nil
	}
	if decl := DeclarationForValue(call[0]); decl != nil {
		return decl.Type
	}
	if sym, ok := scmerSymbol(call[0]); ok {
		if td := ome.variableTypes[sym]; td != nil {
			return td
		}
		if binding := env.FindRead(sym); binding != nil {
			if value, exists := binding.Vars[sym]; exists {
				if td := value.CallableType(); td != nil {
					return td
				}
			}
		}
		return nil
	}
	if call[0].IsNthLocalVar() {
		return ome.numberedTypes[call[0].NthLocalVar()]
	}
	// Resolve an immediately-created native operator from its declared return
	// type without recursively analysing the call-head subtree. The begin usage
	// walk must remain linear even for deeply nested generated expressions.
	callHead := call[0]
	if stripped, ok := scmerStripSourceInfo(callHead); ok {
		callHead = stripped
	}
	if constructor, ok := scmerSlice(callHead); ok && len(constructor) > 0 {
		if decl := DeclarationForValue(constructor[0]); decl != nil && decl.Type != nil && decl.Type.Return != nil && decl.Type.Return.Kind == "func" {
			return decl.Type.Return
		}
	}
	return nil
}

func optimizerCallParam(callType *TypeDescriptor, argIndex int) *TypeDescriptor {
	if callType == nil || len(callType.Params) == 0 || argIndex < 0 {
		return nil
	}
	if argIndex >= len(callType.Params) {
		argIndex = len(callType.Params) - 1
	}
	return callType.Params[argIndex]
}

func optimizerIsLambda(expr Scmer) bool {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	items, ok := scmerSlice(expr)
	return ok && len(items) >= 3 && scmerIsSymbol(items[0], "lambda")
}

func optimizerProcSequenceForDefinition(name Symbol, expression Scmer) procSequenceKind {
	if name != Symbol("split_and_terms") {
		return procSequenceNone
	}
	lambda, ok := scmerSlice(expression)
	if !ok || len(lambda) < 3 || !scmerIsSymbol(lambda[0], "lambda") {
		return procSequenceNone
	}
	body, ok := scmerSlice(lambda[2])
	if !ok || len(body) < 8 || len(body)%2 != 0 || (!scmerIsSymbol(body[0], "match") && !scmerIsSymbol(body[0], "match_mut")) {
		return procSequenceNone
	}
	input, ok := scmerSlice(body[1])
	if !ok || len(input) != 3 || !scmerIsSymbol(input[0], "coalesceNil") {
		return procSequenceNone
	}
	hasBinary, hasVariadic := false, false
	for i := 2; i+1 < len(body); i += 2 {
		pattern, patternOK := scmerSlice(body[i])
		result, resultOK := scmerSlice(body[i+1])
		if !patternOK || !resultOK {
			continue
		}
		if len(pattern) == 3 {
			head, headOK := scmerSlice(pattern[0])
			if headOK && len(head) == 2 &&
				(scmerIsSymbol(head[0], "symbol") || scmerIsSymbol(head[0], "quote")) &&
				scmerIsSymbol(head[1], "and") && len(result) >= 2 && scmerIsSymbol(result[0], "merge") {
				hasBinary = true
				continue
			}
		}
		if len(pattern) == 3 && scmerIsSymbol(pattern[0], "cons") &&
			len(result) >= 3 && scmerIsSymbol(result[0], "if") {
			hasVariadic = true
		}
	}
	if !hasBinary || !hasVariadic {
		return procSequenceNone
	}
	return procSequenceAndTerms
}

func optimizeList(v []Scmer, env *Env, ome *optimizerMetainfo, useResult bool) (Scmer, TypeInfo) {
	var transferOwnership, isConstant bool
	if len(v) == 0 {
		return NewSlice(v), tiZero
	}
	if callable, ok := resolveSpecialFormSymbol(v[0], env); ok {
		resolved := make([]Scmer, len(v))
		copy(resolved, v)
		resolved[0] = callable
		v = resolved
	}
	headSym, headOk := scmerSymbol(v[0])

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
		var cseBindings int
		v, cseBindings = optimizeBeginFoldableCSE(v, bodyStart, ome)
		if headSym == Symbol("begin_mut") && cseBindings > 0 {
			reserve += cseBindings
			v[1] = NewInt(int64(reserve))
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
		var leafScan *bool
		var leafScanRoot bool
		var visitNode func(x Scmer, depth int, captured bool, captureOnce bool, lambdaOnce bool, region *conditionalSinkRegion, blacklist []Symbol)
		visitNode = func(x Scmer, depth int, captured bool, captureOnce bool, lambdaOnce bool, region *conditionalSinkRegion, blacklist []Symbol) {
			if stripped, ok := scmerStripSourceInfo(x); ok {
				x = stripped
			}
			if sub, ok := scmerSlice(x); ok && len(sub) > 0 {
				subHeadExpr := sub[0]
				if stripped, ok := scmerStripSourceInfo(subHeadExpr); ok {
					subHeadExpr = stripped
				}
				subHead, subHeadOk := scmerSymbol(subHeadExpr)
				if leafScan != nil {
					if leafScanRoot {
						leafScanRoot = false
					} else if subHeadOk {
						switch subHead {
						case Symbol("begin"), Symbol("begin_mut"), Symbol("!begin"), Symbol("define"), Symbol("set"), Symbol("setN"), Symbol("lambda"), Symbol("eval"), Symbol("import"):
							*leafScan = false
						}
					}
				}
				if region != nil && subHeadOk && subHead == Symbol("setN") {
					region.hasBinding = true
				}
				if subHeadOk && (subHead == Symbol("define") || subHead == Symbol("set")) {
					if region != nil {
						region.hasBinding = true
					}
					var definedSym Symbol
					var scansDefinition bool
					if depth == 0 {
						definedSym, scansDefinition = scmerSymbol(sub[1])
					}
					previousLeafScan, previousLeafRoot := leafScan, leafScanRoot
					leaf := optimizerIsLambda(sub[2])
					if scansDefinition {
						leafScan, leafScanRoot = &leaf, true
					}
					visitNode(sub[2], depth, captured, captureOnce, false, region, blacklist)
					if scansDefinition {
						facts := bindings[definedSym]
						facts.leafLambda = leaf
						bindings[definedSym] = facts
						leafScan, leafScanRoot = previousLeafScan, previousLeafRoot
					}
					if depth == 0 {
						if sym, ok := scmerSymbol(sub[1]); ok {
							variableContent[sym] = sub[2]
						}
					}
				} else if subHeadOk && subHead == Symbol("lambda") {
					captureOnce = captureOnce && lambdaOnce
					params := sub[1]
					if stripped, ok := scmerStripSourceInfo(params); ok {
						params = stripped
					}
					if sym, ok := scmerSymbol(params); ok {
						visitNode(sub[2], depth+1, true, captureOnce, false, region, append(append([]Symbol{}, blacklist...), sym))
					} else if list, ok := scmerSlice(params); ok {
						blacklist2 := append([]Symbol{}, blacklist...)
						for _, entry := range list {
							if s, ok := scmerSymbol(entry); ok {
								blacklist2 = append(blacklist2, s)
							}
						}
						visitNode(sub[2], depth+1, true, captureOnce, false, region, blacklist2)
					}
				} else if subHeadOk && (subHead == Symbol("begin") || subHead == Symbol("begin_mut")) {
					start := 1
					if subHead == Symbol("begin_mut") {
						start = 2
					}
					for i := start; i < len(sub); i++ {
						visitNode(sub[i], depth+1, captured, captureOnce, false, region, blacklist)
					}
				} else if subHeadOk && subHead == Symbol("!begin") {
					for i := 1; i < len(sub); i++ {
						visitNode(sub[i], depth, captured, captureOnce, false, region, blacklist)
					}
				} else if subHeadOk && subHead == Symbol("eval") {
					if region != nil {
						region.hasBinding = true
					}
					usedVariables[Symbol("eval")] = 1
					for i := 2; i < len(sub); i++ {
						visitNode(sub[i], depth+1, captured, captureOnce, false, region, blacklist)
					}
				} else if subHeadOk && subHead == Symbol("if") {
					i := 1
					for i+1 < len(sub) {
						visitNode(sub[i], depth+1, captured, captureOnce, false, region, blacklist)
						branchRegion := region
						if branchRegion == nil {
							branchRegion = &conditionalSinkRegion{target: &sub[i+1]}
						}
						visitNode(sub[i+1], depth+1, captured, captureOnce, false, branchRegion, blacklist)
						i += 2
					}
					if i < len(sub) {
						branchRegion := region
						if branchRegion == nil {
							branchRegion = &conditionalSinkRegion{target: &sub[i]}
						}
						visitNode(sub[i], depth+1, captured, captureOnce, false, branchRegion, blacklist)
					}
				} else {
					// Also visit the head — it may be a variable used in call position (e.g., (accsess "key"))
					visitNode(sub[0], depth+1, captured, captureOnce, false, region, blacklist)
					callType := optimizerCallType(sub, env, ome)
					for i := 1; i < len(sub); i++ {
						param := optimizerCallParam(callType, i-1)
						argLambdaOnce := optimizerIsLambda(sub[i]) && param != nil && param.Kind == "func" && param.CallsOnce
						visitNode(sub[i], depth+1, captured, captureOnce, argLambdaOnce, region, blacklist)
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
					if facts, tracked := bindings[sym]; tracked {
						facts.useCount++
						if !facts.sinkRegionSeen {
							facts.sinkRegion = region
							facts.sinkRegionSeen = true
						} else if facts.sinkRegion != region {
							facts.sinkConflict = true
							facts.sinkRegion = nil
						}
						if captured {
							facts.captured = true
							if !captureOnce {
								facts.repeatedCapture = true
							}
						}
						if !facts.used {
							facts.firstUse = currentTopIdx
							facts.used = true
						}
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
			visitNode(v[i], 0, false, true, false, nil, nil)
		}
		// A multi-use closure may still have one exclusive execution region: for
		// example, both uses can live in the same arm of an outer if. Move its
		// binding to that arm before normal begin optimization. Reverse definition
		// order preserves the original ordering when several bindings share a target.
		if earliestDynamicSyntax < 0 {
			for i := len(bindingOrder) - 1; i >= 0; i-- {
				sym := bindingOrder[i]
				facts := bindings[sym]
				content, exists := variableContent[sym]
				if !exists || facts.count != 1 || facts.useCount < 2 || !facts.used ||
					facts.firstUse <= facts.defineIdx || facts.sinkConflict || facts.sinkRegion == nil ||
					facts.sinkRegion.target == nil || facts.sinkRegion.hasBinding || facts.repeatedCapture ||
					!optimizerIsLambda(content) {
					continue
				}
				target := facts.sinkRegion.target
				original := *target
				*target = NewSlice([]Scmer{
					NewSymbol("begin"),
					NewSlice([]Scmer{NewSymbol("define"), NewSymbol(string(sym)), content}),
					original,
				})
				v[facts.defineIdx] = NewNil()
				delete(variableContent, sym)
				delete(bindings, sym)
				delete(usedVariables, sym)
			}
		}
		ome2 := ome.CopySharedScope()
		ome2.beginDepth++
		ownedLocalBindings := make(map[Symbol]bool, len(ome.ownedLocalBindings)+len(bindings))
		for sym, owned := range ome.ownedLocalBindings {
			ownedLocalBindings[sym] = owned
		}
		for sym, facts := range bindings {
			delete(ownedLocalBindings, sym)
			if earliestDynamicSyntax < 0 {
				if facts.count == 1 && facts.used && facts.firstUse > facts.defineIdx &&
					facts.useCount == 1 && !facts.captured {
					ownedLocalBindings[sym] = true
				}
			}
		}
		ome2.ownedLocalBindings = ownedLocalBindings
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
			if facts, tracked := bindings[sym]; tracked && facts.count == 1 && facts.useCount == 1 &&
				facts.captured && !facts.repeatedCapture && facts.leafLambda {
				shouldReplace = true
				if ome.specializationUsed != nil {
					*ome.specializationUsed = true
				}
			}
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
				ome2.variableReplacement[sym] = optimizerReplacement{value: content}
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
			for sym, replacement := range ome2.variableReplacement {
				if replacement.outerDepth > 0 {
					replacement.outerDepth--
					ome2.variableReplacement[sym] = replacement
				}
			}
			for i := bodyStart; i < len(v); i++ {
				v[i] = rebaseOuterReferencesAfterScopeRemoval(v[i], 0)
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
						ome2.variableReplacement[sym] = optimizerReplacement{value: slot}
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
			return OptimizeEx(rebaseOuterReferencesAfterScopeRemoval(v[1], 0), env, &ome2, useResult)
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
						ome2.variableReplacement[sym] = optimizerReplacement{value: NewNthLocalVar(NthLocalVar(i))}
					}
				}
			} else if sym, ok := scmerSymbol(params); ok && !assigned[sym] {
				ome2.variableReplacement[sym] = optimizerReplacement{value: NewNthLocalVar(0)}
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
						ome2.variableReplacement[sym] = optimizerReplacement{value: NewNthLocalVar(NthLocalVar(slotIndex))}
					}
				}
				slotIndex++
			}
		} else if sym, ok := scmerSymbol(params); ok {
			if !assigned[sym] {
				ome2.variableReplacement[sym] = optimizerReplacement{value: NewNthLocalVar(NthLocalVar(slotIndex))}
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
		var hasDefinedSym bool
		var definedSym Symbol
		if sym, ok := scmerSymbol(v[1]); ok {
			hasDefinedSym = true
			definedSym = sym
			for _, black := range ome.setBlacklist {
				if black == sym {
					if useResult {
						return materializeOptimizerReplacement(ome.variableReplacement[sym]), tiZero
					}
					return NewNil(), tiConstTransfer
				}
			}
			if repl, ok := ome.variableReplacement[sym]; ok && repl.outerDepth == 0 && repl.value.IsNthLocalVar() {
				v[1] = repl.value
			}
		}
		if v[1].IsNthLocalVar() {
			v[0] = NewSymbol("setN")
		}
		var returnType TypeInfo
		v[2], returnType = OptimizeEx(v[2], env, ome, true)
		transferOwnership = returnType.Transfer()
		if v[1].IsNthLocalVar() {
			localType := returnType.ToTypeDescriptor()
			if localType == nil {
				localType = &TypeDescriptor{Kind: "any", Length: UnknownLength}
			}
			localType.Const = false
			if !ome.ownedLocalBindings[definedSym] {
				localType.Transfer = false
			}
			ome.numberedTypes[v[1].NthLocalVar()] = localType
		}
		if hasDefinedSym && ome.lambdaDepth == 0 && ome.beginDepth == 0 {
			rhs := v[2]
			if stripped, ok := scmerStripSourceInfo(rhs); ok {
				rhs = stripped
			}
			if items, ok := scmerSlice(rhs); ok && len(items) >= 3 && scmerIsSymbol(items[0], "lambda") {
				hasReturn := returnType.Kind() != KindAny
				procReturn := immutableTypeInfo(returnType.WithoutConst())
				v[2] = NewSlice([]Scmer{
					NewSymbol("optimizer_proc_return"),
					v[2],
					NewAny(optimizerProcReturnTemplate{
						Return:    procReturn,
						HasReturn: hasReturn,
						Sequence:  optimizerProcSequenceForDefinition(definedSym, v[2]),
					}),
				})
			}
		}
	case headOk && (headSym == Symbol("match") || headSym == Symbol("match_mut")):
		return optimizeMatch(v, headSym, env, ome, useResult)
	case headOk && headSym == Symbol("parser"):
		return OptimizeParser(NewSlice(v), env, ome, false), tiTransfer
	case !headOk || headSym != Symbol("quote"):
		// Look up declaration for hook dispatch
		if callDecl := DeclarationForValue(v[0]); callDecl != nil && callDecl.Optimize != nil {
			oc := &OptimizerContext{Env: env, Ome: ome}
			result, td := callDecl.Optimize(v, oc, useResult)
			return result, TypeInfoFromTD(td)
		}
		// Default optimization path
		oc := &OptimizerContext{Env: env, Ome: ome}
		result, td := oc.applyDefaultOptimization(v, useResult, "")
		return result, TypeInfoFromTD(td)
	}
	if len(v) == 2 {
		quoted, quotedList := scmerSlice(v[1])
		if v[1].IsNil() || (quotedList && len(quoted) == 0) {
			return NewSlice(v), TypeInfo{kind: KindList, flags: FlagTransfer | FlagConst, length: UnknownLength}
		}
	}

	return NewSlice(v), MakeTypeInfo(transferOwnership, false)
}

// optimizeOuter resolves the operand in the explicitly selected parent scope.
// The call head has already been resolved to its registered special form, so
// this hook must preserve the same scope accounting as symbolic input.
func optimizeOuter(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	if len(v) != 3 {
		return NewSlice(v), tiZero.ToTypeDescriptor()
	}
	depthValue, validDepth := outerDepthLiteral(v[1])
	if !validDepth {
		return NewSlice(v), tiZero.ToTypeDescriptor()
	}
	depth := int(depthValue)
	if depth == 0 {
		return oc.OptimizeSub(v[2], useResult)
	}
	outerOme := optimizerMetainfo{
		variableReplacement: make(map[Symbol]optimizerReplacement),
		setBlacklist:        oc.Ome.setBlacklist,
	}
	for symbol, replacement := range oc.Ome.variableReplacement {
		if replacement.outerDepth >= depth {
			replacement.outerDepth -= depth
			outerOme.variableReplacement[symbol] = replacement
		}
	}
	inner, transferOwnership, isConstant := optimizeExCompat(v[2], oc.Env, &outerOme, useResult)
	if isConstant {
		return inner, tiConstTransfer.ToTypeDescriptor()
	}
	if nested, ok := scmerSlice(inner); ok && len(nested) == 3 && scmerIsSymbol(nested[0], "outer") {
		if nestedDepth, validNestedDepth := outerDepthLiteral(nested[1]); validNestedDepth && nestedDepth <= int64(^uint64(0)>>1)-depthValue {
			return NewSlice([]Scmer{v[0], NewInt(depthValue + nestedDepth), nested[2]}), MakeTypeInfo(transferOwnership, false).ToTypeDescriptor()
		}
	}
	return NewSlice([]Scmer{v[0], v[1], inner}), MakeTypeInfo(transferOwnership, false).ToTypeDescriptor()
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
	// Re-analyzing a large fused callback with progressively weakened ownership
	// clones its complete AST on every fixed-point iteration. Stay conservative
	// for these uncommon shapes so compile cost remains bounded.
	if optimizerNodeCount(callback) > 256 {
		params[0] = &TypeDescriptor{Kind: "any", Length: UnknownLength}
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

// optimizeCoalesce preserves ownership only when every branch can transfer its
// result. This lets an optional owned list flow into an in-place consumer while
// keeping shared fallback values conservative.
func optimizeCoalesce(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	call := oc.applyDefaultOptimizationWithTypes(v, useResult, "")
	if len(call.argumentTypes) < 2 {
		return call.code, call.typeInfo
	}
	branchType := func(ti TypeInfo) *TypeDescriptor {
		if td := ti.ToTypeDescriptor(); td != nil {
			return td
		}
		return &TypeDescriptor{Kind: "any", Length: UnknownLength}
	}
	merged := branchType(call.argumentTypes[1])
	if len(call.argumentTypes) == 2 {
		if items, ok := scmerSlice(call.code); ok && len(items) == 2 {
			return items[1], merged
		}
	}
	for i := 2; i < len(call.argumentTypes); i++ {
		merged, _ = mergeOptimizerTypes(merged, branchType(call.argumentTypes[i]))
	}
	// The lazy selector remains executable code even when all branches happen
	// to be constant. Const means the returned AST is already self-evaluating.
	merged.Const = false
	return call.code, merged
}

// applyDefaultOptimization runs the standard optimization pipeline on a function
// call expression: callback ownership propagation, arg optimization, _mut swap
// (when mutName is non-empty), !list rewrite, and constant folding.
func (oc *OptimizerContext) applyDefaultOptimization(v []Scmer, useResult bool, mutName string) (Scmer, *TypeDescriptor) {
	env := oc.Env
	ome := oc.Ome
	head, _ := scmerSymbol(v[0])

	// An immediately invoked lambda is a real ownership boundary just like a
	// callback passed to a declared operator. Preserve the argument descriptors
	// while optimizing its body so freshly owned accumulators can still select
	// optimizer-only mutating helpers. Join reducers use this shape to wrap the
	// physical skip sentinel around their actual reducer.
	var immediateLambdaArity int
	if lambda, ok := scmerSlice(v[0]); ok && len(lambda) >= 3 && scmerIsSymbol(lambda[0], "lambda") {
		if params, paramsOK := scmerSlice(lambda[1]); paramsOK && len(params) == len(v)-1 {
			immediateLambdaArity = len(params)
		}
	}

	allConstArgs := true
	var firstArgType TypeInfo
	argTypes := make([]TypeInfo, len(v))
	var immediateLambdaParams []*TypeDescriptor
	if immediateLambdaArity > 0 {
		immediateLambdaParams = make([]*TypeDescriptor, immediateLambdaArity)
		for i := 1; i < len(v); i++ {
			v[i], argTypes[i] = OptimizeEx(v[i], env, ome, true)
			immediateLambdaParams[i-1] = argTypes[i].ToTypeDescriptor()
		}
	}

	// Resolve native and symbolic call heads through the same declaration. Code
	// generated by Scheme commonly contains either representation.
	callDecl := DeclarationForValue(v[0])
	var callType *TypeDescriptor
	if callDecl != nil {
		callType = callDecl.Type
	} else {
		callType = optimizerCallType(v, env, ome)
	}
	callName := string(head)
	if callDecl != nil {
		callName = callDecl.Name
	}

	// Optimize all args with callback ownership propagation
	for i := 0; i < len(v); i++ {
		if i == 0 && immediateLambdaParams != nil {
			ome.pendingCallbackParams = immediateLambdaParams
		} else {
			ome.pendingCallbackParams = nil
		}
		ome.pendingCallbackReturn = nil
		if i > 0 && callType != nil {
			paramIdx := i - 1
			if len(callType.Params) == 0 {
				// no type info
			} else if paramIdx >= len(callType.Params) {
				paramIdx = len(callType.Params) - 1
			}
			if paramIdx >= 0 && paramIdx < len(callType.Params) {
				if ti := callType.Params[paramIdx]; ti != nil && ti.Kind == "func" {
					ome.pendingCallbackParams = ti.Params
					ome.pendingCallbackReturn = ti.Return
				}
			}
		}
		var ti TypeInfo
		if i > 0 && immediateLambdaParams != nil {
			ti = argTypes[i]
		} else {
			v[i], ti = OptimizeEx(v[i], env, ome, true)
		}
		ome.pendingCallbackParams = nil
		ome.pendingCallbackReturn = nil
		argTypes[i] = ti
		if i == 1 {
			firstArgType = ti
		}
		if i > 0 && !ti.Const() {
			allConstArgs = false
		}
	}
	if ome.captureArgumentTypes {
		ome.argumentTypes = argTypes
	}
	if !allConstArgs {
		if inlined, ti, ok := tryInlineLeafProc(v, env, ome, useResult); ok {
			return inlined, ti.ToTypeDescriptor()
		}
		if specialized, ok := trySpecializeProcCall(v, argTypes, env, ome); ok {
			v[0] = specialized
		}
	}

	// _mut swap: when mutName is set and first arg is exclusively owned,
	// swap to the in-place variant
	firstArgTransferred := false
	if mutName != "" {
		firstArgFresh := false
		if len(v) >= 2 {
			arg1 := v[1]
			if stripped, ok := scmerStripSourceInfo(arg1); ok {
				arg1 = stripped
			}
			firstArgFresh = firstArgType.Transfer() && !firstArgType.Const()
			// merge_unique accepts either a segment catalog or variadic lists, so
			// its public return contract cannot expose first-argument ownership.
			// A direct catalog constructor is nevertheless fresh. Decide this
			// before NoEscape lowering can turn it into !list.
			if items, ok := scmerSlice(arg1); ok && mutName == "merge_unique_mut" {
				if _, directList := listConstructorElements(items); directList {
					firstArgFresh = true
				}
			}
		}
		if firstArgFresh && len(v) >= 2 {
			markProcOwnershipSpecializationUse(ome, v[1])
			markRootProcOwnershipMutationUse(ome, v[1])
			markNestedProcOwnershipSpecializationUse(ome, v[1])
			v[0] = NewSymbol(mutName)
			firstArgTransferred = true
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
				// A mutating variant returns the first argument's backing storage.
				// Rewriting that argument to a frame-local !list would let the
				// returned value escape after the frame has been reused.
				if i == 1 && firstArgTransferred {
					continue
				}
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
			td := &TypeDescriptor{Transfer: true, Const: true, Length: UnknownLength}
			if d.Type != nil && d.Type.Return != nil {
				td = &TypeDescriptor{Transfer: true, Const: true, Kind: d.Type.Return.Kind,
					Params: d.Type.Return.Params, Return: d.Type.Return.Return,
					HasSideEffects: d.Type.Return.HasSideEffects, Length: d.Type.Return.Length}
			}
			result = wrapConstListForCode(result, td, false)
			return result, td
		}
		if d.Type != nil && d.Type.Return != nil {
			retTD = d.Type.Return
		}
	}
	if retTD == nil {
		if v[0].IsProc() && v[0].Proc().OptimizerMeta != nil && v[0].Proc().OptimizerMeta.HasReturn {
			procReturn = v[0].Proc().OptimizerMeta.Return
			hasProcReturn = true
		} else if sym, ok := scmerSymbol(v[0]); ok {
			if ti, exists := env.optimizerProcHint(sym); exists {
				procReturn = ti
				hasProcReturn = true
			}
		}
	}

	// Unknown calls return borrowed values. A declaration or optimized Proc
	// return contract must explicitly transfer ownership to enable mutation.
	td := &TypeDescriptor{}
	if immediateLambdaParams != nil {
		// The call returns the lambda body's value, not its last argument. This
		// distinction keeps ownership monotonic across reducer fixed-point passes.
		td = copyTypeDescriptor(argTypes[0].ToTypeDescriptor())
		td.Const = false
	} else if hasProcReturn {
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
		td.Transfer = retTD.Transfer
		td.Kind = retTD.Kind
		td.Length = retTD.Length
		td.Params = retTD.Params
		td.Return = retTD.Return
		td.HasSideEffects = retTD.HasSideEffects
		td.Keys = retTD.Keys
		td.Element = retTD.Element
	} else {
		td.Transfer = false
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

type optimizedCall struct {
	code          Scmer
	typeInfo      *TypeDescriptor
	argumentTypes []TypeInfo
}

// applyDefaultOptimizationWithTypes returns the code and TypeInfo from the
// recursive call optimization together with the TypeInfo returned by each
// argument. It never revisits an argument to derive metadata.
func (oc *OptimizerContext) applyDefaultOptimizationWithTypes(v []Scmer, useResult bool, mutName string) optimizedCall {
	previousCapture := oc.Ome.captureArgumentTypes
	previousTypes := oc.Ome.argumentTypes
	oc.Ome.captureArgumentTypes = true
	code, typeInfo := oc.applyDefaultOptimization(v, useResult, mutName)
	argumentTypes := oc.Ome.argumentTypes
	oc.Ome.captureArgumentTypes = previousCapture
	oc.Ome.argumentTypes = previousTypes
	return optimizedCall{code: code, typeInfo: typeInfo, argumentTypes: argumentTypes}
}

const constListQuoteThreshold = 32

// wrapConstListForCode wraps a constant-folded Scmer value so it can safely
// be embedded in generated code. Raw list/slice values would be misinterpreted
// as function calls by Eval. The optimizer's return descriptor is authoritative:
// large constant lists are retained behind one non-transferable quote. Symbols are
// also quoted so they evaluate to the symbol value rather than being looked up
// as variable references.
// Only wraps plain slices — FastDicts are left as-is since they are self-evaluating.
func wrapConstListForCode(val Scmer, resultType *TypeDescriptor, embedded bool) Scmer {
	if val.IsSlice() {
		if !embedded && resultType != nil && resultType.Const && len(val.Slice()) >= constListQuoteThreshold {
			resultType.Transfer = false
			return NewSlice([]Scmer{NewSymbol("quote"), val})
		}
		list := val.Slice()
		packed := make([]Scmer, 1, len(list)+1)
		packed[0] = NewSymbol("list")
		for _, elem := range list {
			packed = append(packed, wrapConstListForCode(elem, resultType, true))
		}
		return NewSlice(packed)
	}
	if val.IsSymbol() {
		return NewSlice([]Scmer{NewSymbol("quote"), val})
	}
	return val
}

// unwrapConstListFromCode is the inverse of wrapConstListForCode: it extracts
// quoted values and still accepts the legacy recursive (list ...) encoding so
// that raw data values can be passed to a foldable function at constant-fold time.
func unwrapConstListFromCode(val Scmer) Scmer {
	if list, ok := scmerSlice(val); ok && len(list) == 2 && scmerIsSymbol(list[0], "quote") {
		return list[1]
	}
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
	var resultType *TypeDescriptor
	mergeResultType := func(td *TypeDescriptor) {
		td = copyTypeDescriptor(normalizeOptimizerType(td))
		td.Const = false
		if resultType == nil {
			resultType = td
			return
		}
		resultType, _ = mergeOptimizerTypes(resultType, td)
	}
	i := 1
	for i+1 < len(v) {
		cond, _ := oc.OptimizeSub(v[i], true)
		switch cond.GetTag() {
		case tagNil, tagBool, tagInt, tagFloat, tagString:
			if ToBool(cond) {
				// Constant true: this branch is always taken, return its body
				body, td := oc.OptimizeSub(v[i+1], useResult)
				if len(out) == 1 {
					return body, td
				}
				mergeResultType(td)
				return NewSlice(append(out, body)), resultType
			}
			// Constant false: skip this condition+then pair entirely
			i += 2
			continue
		}
		// Non-constant condition: keep it and optimize the then-branch
		then, td := oc.OptimizeSub(v[i+1], useResult)
		mergeResultType(td)
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
		mergeResultType(td)
		out = append(out, elseVal)
	} else if len(out) == 1 {
		// All conditions false and no else: nil
		return NewNil(), &TypeDescriptor{Transfer: true, Const: true}
	} else {
		mergeResultType(&TypeDescriptor{Transfer: true, Const: true, Kind: "nil", Length: UnknownLength})
	}
	if base, item, ok := optimizedUniqueAppendBranches(out); ok {
		return NewSlice([]Scmer{NewSymbol("append_unique_mut"), base, item}), FreshAlloc
	}
	// Single condition left: keep as (if cond then else)
	return NewSlice(out), resultType
}

func optimizedUniqueAppendBranches(expr []Scmer) (Scmer, Scmer, bool) {
	if len(expr) != 4 {
		return NewNil(), NewNil(), false
	}
	condition, conditionOK := scmerSlice(expr[1])
	otherwise, otherwiseOK := scmerSlice(expr[3])
	if !conditionOK || len(condition) != 3 || !scmerIsSymbol(condition[0], "contains?") ||
		!otherwiseOK || len(otherwise) != 3 || !scmerIsSymbol(otherwise[0], "append_mut") {
		return NewNil(), NewNil(), false
	}
	base := condition[1]
	item := condition[2]
	if !optimizerStableReference(base) || !optimizerStableReference(item) ||
		!structuralEqual(expr[2], base) || !structuralEqual(otherwise[1], base) || !structuralEqual(otherwise[2], item) {
		return NewNil(), NewNil(), false
	}
	return base, item, true
}

func optimizerStableReference(expr Scmer) bool {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	if expr.IsNthLocalVar() || expr.IsSymbol() {
		return true
	}
	items, ok := scmerSlice(expr)
	validDepth := false
	if ok && len(items) == 3 && scmerIsSymbol(items[0], "outer") {
		_, validDepth = outerDepthLiteral(items[1])
	}
	return validDepth && optimizerStableReference(items[2])
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

type matchPatternInfo struct {
	hash       uint64
	comparable bool
}

func matchPatternScalarInfo(pattern Scmer) matchPatternInfo {
	if stripped, ok := scmerStripSourceInfo(pattern); ok {
		pattern = stripped
	}
	switch pattern.GetTag() {
	case tagNil, tagBool, tagInt, tagFloat, tagDate, tagString, tagSymbol, tagNthLocalVar:
		return matchPatternInfo{hash: combineStructuralHash(uint64(pattern.GetTag()), HashKey(pattern)), comparable: true}
	default:
		return matchPatternInfo{}
	}
}

// optimizeMatchPattern returns the canonical pattern and its structural hash in
// the same recursive walk. Callers can therefore recognize unreachable duplicate
// branches without repeatedly analyzing their pattern trees.
func optimizeMatchPattern(pattern Scmer, env *Env, ome *optimizerMetainfo) (Scmer, matchPatternInfo) {
	if stripped, ok := scmerStripSourceInfo(pattern); ok {
		pattern = stripped
	}

	if sym, ok := scmerSymbol(pattern); ok {
		delete(ome.variableReplacement, sym)
		return pattern, matchPatternScalarInfo(pattern)
	}

	if slice, ok := scmerSlice(pattern); ok {
		if len(slice) == 0 {
			return NewSlice(slice), matchPatternInfo{hash: combineStructuralHash(0xbb67ae8584caa73b, 0), comparable: true}
		}
		if stripped, ok := scmerStripSourceInfo(slice[0]); ok {
			slice[0] = stripped
		}
		headSym, headOk := scmerSymbol(slice[0])
		if headOk && headSym == Symbol("eval") && len(slice) > 1 {
			slice[1], _ = OptimizeEx(slice[1], env, ome, true)
			return NewSlice(slice), matchPatternInfo{}
		}
		if headOk && headSym == Symbol("var") && len(slice) == 2 {
			pattern = NewNthLocalVar(NthLocalVar(ToInt(slice[1])))
			return pattern, matchPatternScalarInfo(pattern)
		}
		if headOk && (headSym == Symbol("symbol") || headSym == Symbol("quote")) && len(slice) == 2 {
			if _, literal := scmerSymbol(slice[1]); literal {
				// match implements both spellings as the same symbol-literal pattern.
				slice[0] = NewSymbol("symbol")
				if stripped, ok := scmerStripSourceInfo(slice[1]); ok {
					slice[1] = stripped
				}
				pattern = NewSlice(slice)
				literalInfo := matchPatternScalarInfo(slice[1])
				hash := combineStructuralHash(0xbb67ae8584caa73b, 2)
				hash = combineStructuralHash(hash, matchPatternScalarInfo(slice[0]).hash)
				hash = combineStructuralHash(hash, literalInfo.hash)
				return pattern, matchPatternInfo{hash: hash, comparable: literalInfo.comparable}
			}
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
				slice[i], _ = optimizeMatchPattern(slice[i], env, ome)
			}
			return NewSlice(slice), matchPatternInfo{}
		}
		hash := combineStructuralHash(0xbb67ae8584caa73b, uint64(len(slice)))
		headInfo := matchPatternScalarInfo(slice[0])
		if !headOk {
			slice[0], headInfo = optimizeMatchPattern(slice[0], env, ome)
		}
		comparable := headInfo.comparable
		hash = combineStructuralHash(hash, headInfo.hash)
		for i := 1; i < len(slice); i++ {
			var info matchPatternInfo
			slice[i], info = optimizeMatchPattern(slice[i], env, ome)
			hash = combineStructuralHash(hash, info.hash)
			comparable = comparable && info.comparable
		}
		return NewSlice(slice), matchPatternInfo{hash: hash, comparable: comparable}
	}

	return pattern, matchPatternScalarInfo(pattern)
}

func cloneDescriptorMap(source map[Symbol]*TypeDescriptor) map[Symbol]*TypeDescriptor {
	result := make(map[Symbol]*TypeDescriptor, len(source))
	for symbol, td := range source {
		result[symbol] = td
	}
	return result
}

func cloneNumberedDescriptorMap(source map[NthLocalVar]*TypeDescriptor) map[NthLocalVar]*TypeDescriptor {
	result := make(map[NthLocalVar]*TypeDescriptor, len(source))
	for slot, td := range source {
		result[slot] = td
	}
	return result
}

func cloneSymbolBoolMap(source map[Symbol]bool) map[Symbol]bool {
	result := make(map[Symbol]bool, len(source))
	for symbol, value := range source {
		result[symbol] = value
	}
	return result
}

func cloneNumberedBoolMap(source map[NthLocalVar]bool) map[NthLocalVar]bool {
	result := make(map[NthLocalVar]bool, len(source))
	for slot, value := range source {
		result[slot] = value
	}
	return result
}

func bindMatchPatternType(pattern Scmer, td *TypeDescriptor, ome *optimizerMetainfo, derived bool) {
	if stripped, ok := scmerStripSourceInfo(pattern); ok {
		pattern = stripped
	}
	if pattern.IsNthLocalVar() {
		ome.numberedTypes[pattern.NthLocalVar()] = td
		if derived && td != nil && td.Transfer {
			ome.specializationOwnedSlots[pattern.NthLocalVar()] = true
		}
		return
	}
	if sym, ok := scmerSymbol(pattern); ok {
		switch sym {
		case Symbol("_"), Symbol("nil"), Symbol("true"), Symbol("false"):
			return
		}
		ome.variableTypes[sym] = td
		if derived && td != nil && td.Transfer {
			ome.specializationOwnedVars[sym] = true
		}
		if replacement, exists := ome.variableReplacement[sym]; exists && replacement.outerDepth == 0 && replacement.value.IsNthLocalVar() {
			ome.numberedTypes[replacement.value.NthLocalVar()] = td
			if derived && td != nil && td.Transfer {
				ome.specializationOwnedSlots[replacement.value.NthLocalVar()] = true
			}
		}
		return
	}
	items, ok := scmerSlice(pattern)
	if !ok || len(items) == 0 {
		return
	}
	head, headOK := scmerSymbol(items[0])
	if !headOK {
		for i, child := range items {
			bindMatchPatternType(child, descriptorProjection(td, strconv.Itoa(i)), ome, true)
		}
		return
	}
	switch head {
	case Symbol("cons"):
		if len(items) == 3 {
			bindMatchPatternType(items[1], descriptorProjection(td, "0"), ome, true)
			bindMatchPatternType(items[2], matchTailDescriptor(td), ome, true)
		}
	case Symbol("list"):
		for i, child := range items[1:] {
			bindMatchPatternType(child, descriptorProjection(td, strconv.Itoa(i)), ome, true)
		}
	case Symbol("list?"):
		if len(items) == 2 {
			refined := copyTypeDescriptor(td)
			refined.Kind = "list"
			bindMatchPatternType(items[1], refined, ome, derived)
		}
	case Symbol("eval"), Symbol("quote"), Symbol("symbol"), Symbol("string?"), Symbol("number?"), Symbol("regex"), Symbol("ignorecase"), Symbol("merge"):
		return
	default:
		// Unknown named applications are match operators, not list values.
		return
	}
}

func matchTailDescriptor(td *TypeDescriptor) *TypeDescriptor {
	if td == nil {
		return &TypeDescriptor{Kind: "list", Length: UnknownLength}
	}
	tail := &TypeDescriptor{
		Kind:     "list",
		NoEscape: td.NoEscape,
		Transfer: td.Transfer,
		Length:   UnknownLength,
		Element:  td.Element,
	}
	if td.Length > 0 {
		tail.Length = td.Length - 1
		if tail.Length == 0 {
			tail.Length = UnknownLength
		}
	}
	for key, child := range td.Keys {
		index, err := strconv.Atoi(key)
		if err != nil || index == 0 {
			continue
		}
		if tail.Keys == nil {
			tail.Keys = make(map[string]*TypeDescriptor)
		}
		tail.Keys[strconv.Itoa(index-1)] = child
	}
	return tail
}

// optimizeBooleanRegexMatch lowers the common predicate spelling
// (match value (regex "pattern" _) true _ false) to one precompiled test.
// With no captures, constructing a match environment and FindStringSubmatch's
// result slice cannot contribute to the Scheme result.
func optimizeBooleanRegexMatch(v []Scmer, value Scmer) (Scmer, TypeInfo, bool) {
	if len(v) != 6 {
		return NewNil(), tiZero, false
	}
	trueResult := v[3].WithoutSourceInfo()
	falseResult := v[5].WithoutSourceInfo()
	trueLiteral := (trueResult.IsBool() && trueResult.Bool()) || scmerIsSymbol(trueResult, "true")
	falseLiteral := (falseResult.IsBool() && !falseResult.Bool()) || scmerIsSymbol(falseResult, "false")
	if !trueLiteral || !falseLiteral ||
		!scmerIsSymbol(v[4], "_") {
		return NewNil(), tiZero, false
	}
	pattern, ok := scmerSlice(v[2])
	if !ok || len(pattern) != 3 || !scmerIsSymbol(pattern[0], "regex") || !scmerIsSymbol(pattern[2], "_") {
		return NewNil(), tiZero, false
	}
	regexValue := pattern[1].WithoutSourceInfo()
	var compiled *regexp.Regexp
	switch {
	case regexValue.IsRegex():
		compiled = regexValue.Regex()
	case regexValue.IsString():
		var err error
		compiled, err = regexp.Compile(regexValue.String())
		if err != nil {
			return NewNil(), tiZero, false
		}
	default:
		return NewNil(), tiZero, false
	}
	if compiled.NumSubexp() != 0 {
		return NewNil(), tiZero, false
	}
	return NewSlice([]Scmer{
		NewSymbol(jitConstantRegexpPredicateName),
		NewRegex(compiled),
		value,
	}), TypeInfo{kind: KindBool, flags: FlagTransfer, length: UnknownLength}, true
}

func optimizeMatch(v []Scmer, headSym Symbol, env *Env, ome *optimizerMetainfo, useResult bool) (Scmer, TypeInfo) {
	value, valueType := OptimizeEx(v[1], env, ome, true)
	if result, resultType, optimized := optimizeBooleanRegexMatch(v, value); optimized {
		return result, resultType
	}
	transferOwnership := valueType.Transfer()
	var valueDescriptor *TypeDescriptor
	if td := valueType.ToTypeDescriptor(); td != nil && (len(td.Keys) > 0 || td.Element != nil) {
		// Pattern variables are runtime bindings. Preserve structural ownership,
		// but never expose Const as if a branch variable had a compile-time value.
		valueDescriptor = callbackParameterType(td)
	}
	head := v[0]
	if headSym == Symbol("match") && transferOwnership {
		markProcOwnershipSpecializationUse(ome, value)
		head = NewSymbol("match_mut")
	}
	out := make([]Scmer, 0, len(v))
	out = append(out, head, value)
	var seen map[uint64][]Scmer
	if (len(v)-2)/2 > 1 {
		seen = make(map[uint64][]Scmer, (len(v)-2)/2)
	}
	for i := 3; i < len(v); i += 2 {
		branchOme := ome.CopySharedScope()
		pattern, info := optimizeMatchPattern(v[i-1], env, &branchOme)
		if valueDescriptor != nil {
			branchOme.variableTypes = cloneDescriptorMap(ome.variableTypes)
			branchOme.numberedTypes = cloneNumberedDescriptorMap(ome.numberedTypes)
			branchOme.specializationOwnedVars = cloneSymbolBoolMap(ome.specializationOwnedVars)
			branchOme.specializationOwnedSlots = cloneNumberedBoolMap(ome.specializationOwnedSlots)
			bindMatchPatternType(pattern, valueDescriptor, &branchOme, false)
		}
		duplicate := false
		if info.comparable && seen != nil {
			for _, previous := range seen[info.hash] {
				if astStructuralEqual(previous, pattern) {
					duplicate = true
					break
				}
			}
		}
		if duplicate {
			continue
		}
		if info.comparable && seen != nil {
			seen[info.hash] = append(seen[info.hash], pattern)
		}
		result, resultTransfer, _ := optimizeExCompat(v[i], env, &branchOme, useResult)
		transferOwnership = resultTransfer
		out = append(out, pattern, result)
	}
	if len(v)%2 == 1 {
		fallback, fallbackTransfer, _ := optimizeExCompat(v[len(v)-1], env, ome, useResult)
		transferOwnership = fallbackTransfer
		out = append(out, fallback)
	}
	return NewSlice(out), MakeTypeInfo(transferOwnership, false)
}

func OptimizeParser(val Scmer, env *Env, ome *optimizerMetainfo, ignoreResult bool) Scmer {
	if val.IsSourceInfo() {
		val = val.SourceInfo().value
	} else if val.GetTag() == tagAny {
		if sourceInfo, ok := val.Any().(SourceInfo); ok {
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
			// The optimized tree is executable code. Preserve an empty list result
			// as quoted data; a bare empty AST denotes nil to Eval and the JIT.
			if slice[2].IsSlice() && len(slice[2].Slice()) == 0 {
				slice[2] = NewSlice([]Scmer{NewSymbol("quote"), slice[2]})
			}
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
		return NewAny(&optimizedParserSyntax{Syntax: val, Parser: p})
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
	if expr.GetTag() == tagSpecialForm {
		return NewSymbol(expr.SpecialFormName())
	}
	if !expr.IsSlice() {
		return expr
	}
	items := expr.Slice()
	if len(items) >= 3 && scmerIsSymbol(items[0], "!list") {
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
	if len(items) == 3 && scmerIsSymbol(items[0], "!!list") && items[1].IsNthLocalVar() {
		return NewSlice([]Scmer{NewSymbol("list")})
	}
	// recurse into sub-expressions
	newItems := make([]Scmer, len(items))
	changed := false
	for i, item := range items {
		newItems[i] = DeoptimizeExpr(item)
		if newItems[i] != item {
			changed = true
		}
	}
	if !changed {
		return expr
	}
	return NewSlice(newItems)
}

func specializationSlotSymbol(slot NthLocalVar) Scmer {
	return NewSymbol("__optimizer_slot_" + strconv.Itoa(int(slot)))
}

// deoptimizeProcSpecializationExpr restores optimizer-created local slots to
// stable internal names while rebuilding a Proc specialization. Parameters
// retain their numbered representation; begin-local setN bindings must become
// visible to the existing usage walk so type-driven rewrites can fire again.
// This replaces the ordinary DeoptimizeExpr traversal rather than adding a
// second analysis pass over the body.
func deoptimizeProcSpecializationExpr(expr Scmer, paramCount int) Scmer {
	if expr.IsNthLocalVar() {
		slot := expr.NthLocalVar()
		if int(slot) >= paramCount {
			return specializationSlotSymbol(slot)
		}
		return expr
	}
	if expr.GetTag() == tagSpecialForm {
		return NewSymbol(expr.SpecialFormName())
	}
	if !expr.IsSlice() {
		return expr
	}
	items := expr.Slice()
	if len(items) >= 3 && scmerIsSymbol(items[0], "!list") {
		count := int(ToInt(items[2]))
		if count == len(items)-3 {
			newItems := make([]Scmer, 1+count)
			newItems[0] = NewSymbol("list")
			for i := 0; i < count; i++ {
				newItems[1+i] = deoptimizeProcSpecializationExpr(items[3+i], paramCount)
			}
			return NewSlice(newItems)
		}
	}
	if len(items) == 3 && scmerIsSymbol(items[0], "!!list") && items[1].IsNthLocalVar() {
		return NewSlice([]Scmer{NewSymbol("list")})
	}
	newItems := make([]Scmer, len(items))
	for i, item := range items {
		newItems[i] = deoptimizeProcSpecializationExpr(item, paramCount)
	}
	if len(newItems) == 3 && scmerIsSymbol(newItems[0], "setN") && items[1].IsNthLocalVar() && int(items[1].NthLocalVar()) >= paramCount {
		newItems[0] = NewSymbol("define")
	}
	return NewSlice(newItems)
}
