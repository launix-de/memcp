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

import "regexp"

var SettingsHaveGoodBacktraces bool

// Global guard: once eval/import is seen during optimization, disable risky inlining
var optimizerSeenEvalImport bool

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

	var vars Vars
	if p.NumVars == 0 {
		vars = make(Vars)
	}
	en := &Env{Vars: vars, VarsNumbered: make([]Scmer, p.NumVars), Outer: p.En, Nodefine: false}
	params := p.Params
	if stripped, ok := scmerStripSourceInfo(params); ok {
		params = stripped
	}
	if params.IsSlice() {
		paramSlice := params.Slice()
		if p.NumVars > 0 {
			return func(args ...Scmer) Scmer {
				for i := 0; i < p.NumVars; i++ {
					if i < len(args) {
						en.VarsNumbered[i] = args[i]
					} else {
						en.VarsNumbered[i] = NewNil()
					}
				}
				return Eval(p.Body, en)
			}
		}
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
			return func(args ...Scmer) Scmer {
				en.VarsNumbered[0] = NewSlice(args)
				return Eval(p.Body, en)
			}
		}
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
	ome := newOptimizerMetainfo()
	v, _, _ := OptimizeEx(val, env, &ome, true)
	return v
}

type optimizerMetainfo struct {
	variableReplacement map[Symbol]Scmer
	setBlacklist        []Symbol
}

func newOptimizerMetainfo() (result optimizerMetainfo) {
	result.variableReplacement = make(map[Symbol]Scmer)
	return
}
func (ome *optimizerMetainfo) Copy() (result optimizerMetainfo) {
	result.variableReplacement = make(map[Symbol]Scmer)
	for k, v := range ome.variableReplacement {
		result.variableReplacement[k] = NewSlice([]Scmer{NewSymbol("outer"), v})
	}
	result.setBlacklist = ome.setBlacklist
	return
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

func scmerStripSourceInfo(v Scmer) (Scmer, bool) {
	if v.IsSourceInfo() {
		si := v.SourceInfo()
		return si.value, true
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
func OptimizeEx(val Scmer, env *Env, ome *optimizerMetainfo, useResult bool) (result Scmer, transferOwnership bool, isConstant bool) {
	if val.ptr == nil && val.aux == 0 {
		return NewNil(), true, true
	}

	switch val.GetTag() {
	case tagNil, tagBool, tagInt, tagFloat, tagString:
		return val, true, true
	case tagSymbol:
		sym := mustSymbol(val)
		if replacement, ok := ome.variableReplacement[sym]; ok {
			// Avoid trivial self-alias loops like x -> x or (outer x)
			if replacement.IsSymbol() && mustSymbol(replacement) == sym {
				return val, true, false
			}
			if slice, ok := scmerSlice(replacement); ok && len(slice) == 2 && scmerIsSymbol(slice[0], "outer") {
				if s2, ok := scmerSymbol(slice[1]); ok && s2 == sym {
					return val, true, false
				}
			}
			return OptimizeEx(replacement, env, ome, useResult)
		}
		return val, true, false
	case tagSlice:
		return optimizeList(val.Slice(), env, ome, useResult)
	case tagSourceInfo:
		si := *val.SourceInfo()
		if SettingsHaveGoodBacktraces {
			result, transferOwnership, isConstant = OptimizeEx(si.value, env, ome, useResult)
			if isConstant {
				return result, transferOwnership, true
			}
			si.value = result
			return NewSourceInfo(si), transferOwnership, false
		}
		return OptimizeEx(si.value, env, ome, useResult)
	case tagAny:
		payload := val.Any()
		if pv, ok := payload.(SourceInfo); ok {
			if SettingsHaveGoodBacktraces {
				result, transferOwnership, isConstant = OptimizeEx(pv.value, env, ome, useResult)
				if isConstant {
					return result, transferOwnership, true
				}
				pv.value = result
				return NewSourceInfo(pv), transferOwnership, false
			}
			return OptimizeEx(pv.value, env, ome, useResult)
		}
		if sym, ok := payload.(Symbol); ok {
			return OptimizeEx(NewSymbol(string(sym)), env, ome, useResult)
		}
		// no longer accept []Scmer in tagAny payloads
		if sm, ok := payload.(Scmer); ok {
			return OptimizeEx(sm, env, ome, useResult)
		}
		switch v := payload.(type) {
		case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, string:
			return FromAny(v), true, true
		}
		return val, transferOwnership, false
	default:
		return val, transferOwnership, false
	}
}

func optimizeList(v []Scmer, env *Env, ome *optimizerMetainfo, useResult bool) (result Scmer, transferOwnership bool, isConstant bool) {
	if len(v) == 0 {
		return NewSlice(v), transferOwnership, false
	}

	headSym, headOk := scmerSymbol(v[0])

	if headOk && headSym == Symbol("outer") && len(v) == 2 {
		inner, transferOwnership, isConstant := OptimizeEx(v[1], env, ome, useResult)
		if isConstant {
			return inner, true, true
		}
		v[1] = inner
		return NewSlice(v), transferOwnership, false
	}

	if headOk && headSym == Symbol("begin") {
		usedVariables := make(map[Symbol]int)
		variableContent := make(map[Symbol]Scmer)
		// Track top-level define positions and earliest top-level eval/import index
		defineTopIdx := make(map[Symbol]int)
		earliestEvalImport := -1
		var visitNode func(x Scmer, depth int, blacklist []Symbol)
		visitNode = func(x Scmer, depth int, blacklist []Symbol) {
			if stripped, ok := scmerStripSourceInfo(x); ok {
				x = stripped
			}
			if sub, ok := scmerSlice(x); ok && len(sub) > 0 {
				subHead, subHeadOk := scmerSymbol(sub[0])
				if subHeadOk && (subHead == Symbol("define") || subHead == Symbol("set")) {
					visitNode(sub[2], depth, blacklist)
					if sym, ok := scmerSymbol(sub[1]); ok {
						variableContent[sym] = sub[2]
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
				} else if !subHeadOk || subHead != Symbol("begin") {
					for i := 1; i < len(sub); i++ {
						visitNode(sub[i], depth+1, blacklist)
					}
				} else if subHead != Symbol("eval") {
					usedVariables[Symbol("eval")] = 1
					for i := 2; i < len(sub); i++ {
						visitNode(sub[i], depth, blacklist)
					}
				} else {
					for i := 1; i < len(sub); i++ {
						visitNode(sub[i], depth, blacklist)
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
					if depth > 0 {
						usedVariables[sym] = 100
					} else {
						usedVariables[sym] = usedVariables[sym] + 1
					}
				}
			}
		}
		for i := 1; i < len(v); i++ {
			visitNode(v[i], 0, nil)
			// Scan top-level statement order for define/set and eval/import safeguards
			expr := v[i]
			if stripped, ok := scmerStripSourceInfo(expr); ok {
				expr = stripped
			}
			if sub, ok := scmerSlice(expr); ok && len(sub) > 0 {
				if head, ok := scmerSymbol(sub[0]); ok {
					if (head == Symbol("define") || head == Symbol("set")) && len(sub) >= 3 {
						if sym, ok := scmerSymbol(sub[1]); ok {
							defineTopIdx[sym] = i
						}
					}
					if head == Symbol("eval") || head == Symbol("import") {
						if earliestEvalImport == -1 || i < earliestEvalImport {
							earliestEvalImport = i
						}
					}
				}
			}
		}
		ome2 := ome.Copy()
		// begin shares VarsNumbered with parent — strip (outer ...) for NthLocalVar
		for sym, content := range ome2.variableReplacement {
			if slice, ok := scmerSlice(content); ok && len(slice) == 2 && scmerIsSymbol(slice[0], "outer") {
				if slice[1].IsNthLocalVar() {
					ome2.variableReplacement[sym] = slice[1]
				}
			}
		}
		for sym, content := range variableContent {
			normalized := content
			if stripped, ok := scmerStripSourceInfo(content); ok {
				normalized = stripped
			}
			// Bring back old criterion: inline if used < 2 OR RHS is not a list
			shouldReplace := usedVariables[sym] < 2 || !normalized.IsSlice()
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
				if _, ok := defineTopIdx[mustSymbol(normalized)]; ok {
					shouldReplace = false
				}
			}
			// Safeguard: after a top-level eval/import appears, forbid inlining of later defines
			if earliestEvalImport >= 0 {
				if defIdx, ok := defineTopIdx[sym]; ok && defIdx >= earliestEvalImport {
					shouldReplace = false
				}
			}
			// Global safeguard: once eval/import was seen anywhere, stop begin inlining
			if optimizerSeenEvalImport {
				shouldReplace = false
			}
			if shouldReplace {
				delete(variableContent, sym)
				delete(usedVariables, sym)
				ome2.setBlacklist = append(ome2.setBlacklist, sym)
				ome2.variableReplacement[sym] = content
			}
		}
		if len(usedVariables) == 0 {
			v[0] = NewSymbol("!begin")
			for sym, content := range ome2.variableReplacement {
				if slice, ok := scmerSlice(content); ok && len(slice) == 2 && scmerIsSymbol(slice[0], "outer") {
					ome2.variableReplacement[sym] = slice[1]
				}
			}
		}
		for i := 1; i < len(v); i++ {
			var constant bool
			v[i], transferOwnership, constant = OptimizeEx(v[i], env, &ome2, i == len(v)-1 && useResult)
			if constant {
				if i == len(v)-1 {
					isConstant = true
				} else {
					v = append(v[:i], v[i+1:]...)
					i--
				}
			}
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
		if scmerIsSymbol(v[0], "!begin") && len(v) == 2 {
			return OptimizeEx(v[1], env, &ome2, useResult)
		}
		return NewSlice(v), transferOwnership, isConstant
	}

	if headOk && headSym == Symbol("var") && len(v) == 2 {
		return NewNthLocalVar(NthLocalVar(ToInt(v[1]))), false, false
	}

	if headOk && headSym == Symbol("unquote") && len(v) == 2 {
		unquoted := v[1]
		if stripped, ok := scmerStripSourceInfo(unquoted); ok {
			unquoted = stripped
		}
		switch unquoted.GetTag() {
		case tagString:
			return NewSymbol(unquoted.String()), true, false
		case tagAny:
			if s, ok := unquoted.Any().(string); ok {
				return NewSymbol(s), true, false
			}
		}
	}

	if headOk && headSym == Symbol("lambda") {
		params := v[1]
		if stripped, ok := scmerStripSourceInfo(params); ok {
			params = stripped
		}
		// Skip lambdas that already have explicit NumVars
		if len(v) > 3 {
			ome2 := ome.Copy()
			if list, ok := scmerSlice(params); ok {
				for _, param := range list {
					if sym, ok := scmerSymbol(param); ok {
						delete(ome2.variableReplacement, sym)
					}
				}
			} else if sym, ok := scmerSymbol(params); ok {
				delete(ome2.variableReplacement, sym)
			}
			v[2], transferOwnership, _ = OptimizeEx(v[2], env, &ome2, true)
			return NewSlice(v), transferOwnership, false
		}
		// Auto-number parameters
		ome2 := ome.Copy()
		slotIndex := 0
		if list, ok := scmerSlice(params); ok {
			for _, param := range list {
				if sym, ok := scmerSymbol(param); ok {
					if sym != Symbol("_") {
						ome2.variableReplacement[sym] = NewNthLocalVar(NthLocalVar(slotIndex))
					}
				}
				slotIndex++
			}
		} else if sym, ok := scmerSymbol(params); ok {
			ome2.variableReplacement[sym] = NewNthLocalVar(NthLocalVar(slotIndex))
			slotIndex++
		}
		// Set NumVars
		if slotIndex > 0 {
			v = append(v[:len(v):len(v)], NewInt(int64(slotIndex)))
		}
		v[2], transferOwnership, _ = OptimizeEx(v[2], env, &ome2, true)
		return NewSlice(v), transferOwnership, false
	}

	switch {
	case headOk && (headSym == Symbol("set") || headSym == Symbol("define")) && len(v) == 3:
		if sym, ok := scmerSymbol(v[1]); ok {
			for _, black := range ome.setBlacklist {
				if black == sym {
					if useResult {
						return ome.variableReplacement[sym], false, false
					}
					return NewNil(), true, true
				}
			}
			if repl, ok := ome.variableReplacement[sym]; ok && repl.IsNthLocalVar() {
				v[1] = repl
			}
		}
		if v[1].IsNthLocalVar() {
			v[0] = NewSymbol("setN")
		}
		v[2], transferOwnership, _ = OptimizeEx(v[2], env, ome, true)
	case headOk && headSym == Symbol("match"):
		v[1], transferOwnership, _ = OptimizeEx(v[1], env, ome, true)
		for i := 3; i < len(v); i += 2 {
			ome2 := ome.Copy()
			// match shares VarsNumbered with parent — strip (outer ...) for NthLocalVar
			for sym, content := range ome2.variableReplacement {
				if slice, ok := scmerSlice(content); ok && len(slice) == 2 && scmerIsSymbol(slice[0], "outer") {
					if slice[1].IsNthLocalVar() {
						ome2.variableReplacement[sym] = slice[1]
					}
				}
			}
			v[i-1] = OptimizeMatchPattern(v[1], v[i-1], env, ome, &ome2)
			v[i], transferOwnership, _ = OptimizeEx(v[i], env, &ome2, useResult)
		}
		if len(v)%2 == 1 {
			v[len(v)-1], transferOwnership, _ = OptimizeEx(v[len(v)-1], env, ome, useResult)
		}
		return NewSlice(v), transferOwnership, false
	case headOk && headSym == Symbol("parser"):
		return OptimizeParser(NewSlice(v), env, ome, false), true, false
	case !headOk || headSym != Symbol("quote"):
		allConstArgs := true
		for i := 0; i < len(v); i++ {
			var constant bool
			v[i], transferOwnership, constant = OptimizeEx(v[i], env, ome, true)
			if i > 0 && !constant {
				allConstArgs = false
			}
		}
		// Flatten nested + and * (associative operators)
		if headOk && (headSym == Symbol("+") || headSym == Symbol("*")) {
			for i := 1; i < len(v); i++ {
				if inner, ok := scmerSlice(v[i]); ok && len(inner) > 1 && scmerIsSymbol(inner[0], string(headSym)) {
					newV := make([]Scmer, 0, len(v)+len(inner)-2)
					newV = append(newV, v[:i]...)
					newV = append(newV, inner[1:]...)
					newV = append(newV, v[i+1:]...)
					v = newV
					i-- // re-examine this position
				}
			}
		}
		// If this expression is an eval/import call, globally disable further begin inlining
		if headOk && (headSym == Symbol("eval") || headSym == Symbol("import")) {
			optimizerSeenEvalImport = true
		}
		if scmerIsSymbol(v[0], "!begin") && allConstArgs {
			return v[len(v)-1], true, true
		}
		if d := DeclarationForValue(v[0]); d != nil && d.Foldable && allConstArgs && d.Fn != nil {
			for i := range v {
				if list, ok := scmerSlice(v[i]); ok && len(list) > 0 && (isList(list[0]) || scmerIsSymbol(list[0], "list")) {
					v[i] = NewSlice(list[1:])
				}
			}
			result, transferOwnership, isConstant = d.Fn(v[1:]...), true, true
			if list, ok := scmerSlice(result); ok {
				packed := make([]Scmer, 1, len(list)+1)
				packed[0] = NewFunc(List)
				packed = append(packed, list...)
				result = NewSlice(packed)
			}
			return result, transferOwnership, isConstant
		}
		if scmerIsSymbol(v[0], "and") {
			if len(v) == 2 {
				return OptimizeEx(v[1], env, ome, useResult)
			}
			allTrue := true
			for i := 1; i < len(v); i++ {
				arg := v[i]
				switch arg.GetTag() {
				case tagNil, tagBool, tagInt, tagFloat, tagString:
					if !ToBool(arg) {
						return NewBool(false), true, true
					}
					newArgs := append([]Scmer{}, v[:i]...)
					newArgs = append(newArgs, v[i+1:]...)
					return OptimizeEx(NewSlice(newArgs), env, ome, useResult)
				default:
					allTrue = false
				}
			}
			if allTrue {
				return NewBool(true), true, true
			}
			return NewSlice(v), transferOwnership, false
		}
	}

	return NewSlice(v), transferOwnership, false
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
			slice[1], _, _ = OptimizeEx(slice[1], env, ome2, true)
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
	if stripped, ok := scmerStripSourceInfo(val); ok {
		val = stripped
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
			slice[2], _, _ = OptimizeEx(slice[2], env, &ome2, !ignoreResult) // generator expr -> use variables
		}
		if len(slice) > 3 {
			slice[3], _, _ = OptimizeEx(slice[3], env, ome, true) // delimiter expr
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
