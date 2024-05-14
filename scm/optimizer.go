/*
Copyright (C) 2024  Carl-Philip Hänsch

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

import "fmt"

var SettingsHaveGoodBacktraces bool

// to optimize lambdas serially; the resulting function MUST NEVER run on multiple threads simultanously since state is reduced to save mallocs
func OptimizeProcToSerialFunction(val Scmer) func (...Scmer) Scmer {
	if val == nil {
		return func(...Scmer) Scmer {
			return nil
		}
	}
	if result, ok := val.(func(...Scmer) Scmer); ok {
		return result // already optimized
	}
	// TODO: JIT

	// otherwise: precreate a lambda
	p := val.(Proc) // precast procedure

	// some pre-optimizable corner cases
	switch p.Body.(type) {
		case float64, string, bool: // constants
			return func(...Scmer) Scmer {
				return p.Body
			}
	}

	en := &Env{make(Vars), make([]Scmer, p.NumVars), p.En, false} // reusable environment for one thread
	switch params := p.Params.(type) {
	case []Scmer: // default case: 
		if p.NumVars > 0 {
			return func (args ...Scmer) Scmer {
				for i, arg := range args {
					if i < p.NumVars {
						en.VarsNumbered[i] = arg
					} else {
						en.VarsNumbered[i] = nil // fill in nil values
					}
				}
				return Eval(p.Body, en)
			}
		} else {
			return func (args ...Scmer) Scmer {
				if len(params) > len(args) {
					panic(fmt.Sprintf("Apply: function with %d parameters is supplied with %d arguments", len(params), len(args)))
				}
				for i, param := range params {
					if param != Symbol("_") {
						en.Vars[param.(Symbol)] = args[i]
					}
				}
				return Eval(p.Body, en)
			}
		}
	case Symbol: // otherwise: param list is stored in a single variable
		if p.NumVars > 0 {
			return func (args ...Scmer) Scmer {
				en.VarsNumbered[0] = args
				return Eval(p.Body, en)
			}
		} else {
			return func (args ...Scmer) Scmer {
				en.Vars[params] = args
				return Eval(p.Body, en)
			}
		}
	case nil:
		return func (args ...Scmer) Scmer {
			return Eval(p.Body, en)
		}
	}
	panic("value is not compilable: " + String(val))
}

// do preprocessing and optimization (Optimize is allowed to edit the value in-place)
func Optimize(val Scmer, env *Env) Scmer {
	ome := newOptimizerMetainfo()
	v, _ := OptimizeEx(val, env, &ome, true)
	//fmt.Println(SerializeToString(v, env))
	return v
}
type optimizerMetainfo struct {
	variableReplacement map[Symbol]Scmer
	setBlacklist []Symbol
}
func newOptimizerMetainfo() (result optimizerMetainfo) {
	result.variableReplacement = make(map[Symbol]Scmer)
	return
}
func (ome *optimizerMetainfo) Copy() (result optimizerMetainfo) {
	result.variableReplacement = make(map[Symbol]Scmer)
	for k, v := range ome.variableReplacement {
		result.variableReplacement[k] = []Scmer{Symbol("outer"), v}
	}
	result.setBlacklist = ome.setBlacklist
	return
}
func OptimizeEx(val Scmer, env *Env, ome *optimizerMetainfo, useResult bool) (result Scmer, transferOwnership bool) {
	// TODO: static code analysis like escape analysis + replace memory-safe functions with in-place memory manipulating versions (e.g. in set_assoc)
	// TODO: inline use-once
	// TODO: inplace functions (map -> map_inplace, filter -> filter_inplace) will manipulate the first parameter instead of allocating something new
	// TODO: pure imperative functions (map, produce_map, produceN_map) that execute code and return nothing
	// TODO: value chaining -> produce+map+filter -> inplace append (based on pure imperative)
	// TODO: cons/merge->append
	// TODO: constant folding -> functions with constant parameters can be tagged that they are safe to execute AOT
	// TODO: currify -> functions can be partially executed (constmask -> specialized functions that return a func/lambda)
	switch v := val.(type) {
		case SourceInfo:
			if SettingsHaveGoodBacktraces {
				// in debug mode, we have better backtraces
				v.value, transferOwnership = OptimizeEx(v.value, env, ome, useResult)
				return v, transferOwnership
			} else {
				// strip SourceInfo from lambda declarations
				return OptimizeEx(v.value, env, ome, useResult)
			}
		case Symbol:
			// replace variables with their counterparts
			if replacement, ok := ome.variableReplacement[v]; ok {
				return replacement, false
			}
			return val, true // TODO: remove this return once there is a solution to mask out prefetch variables

			// prefetch system functions (not working yet -> sometimes lambdas redefine variables which are not allowed to replace)
			xen := env.FindRead(v)
			if xen != nil {
				if v, ok := xen.Vars[v]; ok {
					return v, false
				} else {
					return val, false
				}
			}
		case []Scmer:
			if len(v) > 0 {
				// TODO: if v[0] == list -> check if children are constant -> use quote instead
				if v[0] == Symbol("begin") {
					// analyze which variables are to be used
					usedVariables := make(map[Symbol]int)
					variableContent := make(map[Symbol]Scmer)
					var visitNode func (x Scmer, depth int, blacklist []Symbol)
					visitNode = func (x Scmer, depth int, blacklist []Symbol) {
						if sub, ok := x.([]Scmer); ok {
							if sub[0] == Symbol("define") || sub[0] == Symbol("set") {
								visitNode(sub[2], depth, blacklist)
								if sym, ok := sub[1].(Symbol); ok {
									// capture variable content in case we want to replace it
									variableContent[sym] = sub[2]
								}
							} else if sub[0] == Symbol("lambda") {
								if sym, ok := sub[1].(Symbol); ok {
									visitNode(sub[2], depth+1, append(blacklist, sym))
								} else if symlist, ok := sub[1].([]Scmer); ok {
									blacklist2 := blacklist
									for _, s := range symlist {
										blacklist2 = append(blacklist2, s.(Symbol))
									}
									visitNode(sub[2], depth+1, blacklist2)
								}
							} else if sub[0] != Symbol("begin") {
								for i := 1; i < len(sub); i++ {
									visitNode(sub[i], depth+1, blacklist)
								}
							} else {
								for i := 1; i < len(sub); i++ {
									visitNode(sub[i], depth, blacklist)
								}
							}
						}
						if sym, ok := x.(Symbol); ok {
							// increase usage count
							isBlacklisted := false
							for _, s := range blacklist {
								if s == sym {
									isBlacklisted = true
								}
							}
							if !isBlacklisted {
								cnt, _ := usedVariables[sym]
								usedVariables[sym] = cnt+1
							}
						}
					}
					for i := 1; i < len(v); i++ {
						visitNode(v[i], 0, []Symbol{})
					}
					ome2 := ome.Copy() // inherit scope
					for sym, content := range variableContent {
						usage, _ := usedVariables[sym]
						_, isArray := content.([]Scmer)
						if usage < 2 || !isArray {
							// remove this variable and inline instead
							delete(variableContent, sym)
							delete(usedVariables, sym)
							ome2.setBlacklist = append(ome2.setBlacklist, sym)
							ome2.variableReplacement[sym] = content
						}
					}
					if len(usedVariables) == 0 {
						v[0] = Symbol("!begin") // make them env-free
						for sym, content := range ome2.variableReplacement {
							if ar, ok := content.([]Scmer); ok {
								if ar[0] == Symbol("outer") {
									ome2.variableReplacement[sym] = ar[1] // peel out one (outer X) because of !begin
								}
							}
						}
					}
					for i := 1; i < len(v); i++ {
						v[i], transferOwnership = OptimizeEx(v[i], env, &ome2, i == len(v)-1)
					}
					return v, transferOwnership
				}
				// (var i) is a serialization artifact
				if v[0] == Symbol("var") && len(v) == 2 {
					return NthLocalVar(ToInt(v[1])), false
				}
				// (unquote s) is a serialization artifact
				if v[0] == Symbol("unquote") && len(v) == 2 {
					if s, ok := v[1].(string); ok {
						return Symbol(s), true // replace with the symbol directly
					}
				}
				// analyze lambdas (but don't pack them into *Proc since they need a fresh env)
				if v[0] == Symbol("lambda") {
					// normalize header and strip meta info
					switch si := v[1].(type) {
						case SourceInfo:
							// strip SourceInfo from lambda declarations
							v[1] = si.value
					}
					ome2 := ome.Copy()
					if l, ok := v[1].([]Scmer); ok {
						for _, param := range l {
							// TODO: param may be (unquote X)
							delete(ome2.variableReplacement, param.(Symbol)) // remove overrides
						}
					} else {
						delete(ome2.variableReplacement, v[1].(Symbol)) // remove overrides
					}
					// optimize body
					/* TODO: reactivate this code once the corner case of double nested scopes is solved
					numVars := 0
					if len(v) == 4 {
						numVars = ToInt(v[3]) // we already have a numvars
					} else {
						// get the params
						switch params := v[1].(type) {
							case []Scmer: // parameter list
								for _, s := range params {
									ome2.variableReplacement[s.(Symbol)] = NthLocalVar(numVars)
									numVars++
								}
							case Symbol: // parameter variable
								ome2.variableReplacement[params] = NthLocalVar(numVars)
								numVars++
							case nil: // parameterless version
							default:
								panic("unknown lambda parameter: " + String(params))
						}
						v = append(v, float64(numVars)) // add parameter
					}
					*/
					// p.Params = nil do not replace parameter list with nil, the execution engine must handle it different
					v[2], transferOwnership = OptimizeEx(v[2], env, &ome2, true) // optimize body
					return v, transferOwnership
				}

				if v[0] == Symbol("outer") {
					// TODO: (outer (lambda (vars) body)) can pull the outer into all symbols and lambdas
				}

				// now all the special cases

				// set/define
				if (v[0] == Symbol("set") || v[0] == Symbol("define")) && len(v) == 3 {
					if s, ok := v[1].(Symbol); ok {
						for _, sym := range ome.setBlacklist {
							if sym == s {
								if useResult {
									return ome.variableReplacement[s], false // omit SET; use the value
								} else {
									return nil, true // omit SET; don't use the value
								}
							}
						}
						if vp, ok := ome.variableReplacement[s]; ok {
							if lv, ok := vp.(NthLocalVar); ok {
								v[1] = lv // set local var -> replace with (var i)
							}
						}
					}
					if _, ok := v[1].(NthLocalVar); ok {
						// change symbol of set/define to setN
						v[0] = Symbol("setN")
					}
					v[2], _ = OptimizeEx(v[2], env, ome, true)
					// TODO: check if we could remove the set instruction and inline the value if it occurs only once
				} else if v[0] == Symbol("match") {
					v[1], _ = OptimizeEx(v[1], env, ome, true)
					/* code is deactivated since variables can be overwritten! */
					for i := 3; i < len(v); i+= 2 {
						// for each pattern
						ome2 := ome.Copy()
						v[i-1] = OptimizeMatchPattern(v[1], v[i-1], env, ome, &ome2) // optimize pattern and collect overwritten variables
						v[i], _ = OptimizeEx(v[i], env, &ome2, useResult) // optimize result and apply overwritten variables
					}
					if len(v)%2 == 1 {
						// last item
						v[len(v)-1], _ = OptimizeEx(v[len(v)-1], env, ome, useResult)
					}
				} else if v[0] == Symbol("parser") {
					return OptimizeParser(v, env, ome, false), true

				// last but not least: recurse over the arguments when we aren't a special case
				} else if v[0] != Symbol("quote") {
					// optimize all other parameters
					for i := 0; i < len(v); i++ {
						v[i], _ = OptimizeEx(v[i], env, ome, true)
					}
				}
			}
	}
	return val, transferOwnership
}
func OptimizeMatchPattern(value Scmer, pattern Scmer, env *Env, ome *optimizerMetainfo, ome2 *optimizerMetainfo) Scmer {
	// TODO: Prune patterns that are not matched by the value (happens mostly during inlining)
	switch p := pattern.(type) {
	case Symbol:
		// TODO: replace Symbol with NthLocalVar
		delete(ome2.variableReplacement, p)
		// TODO: insert replacement into ome2
		return p
	case []Scmer:
		if p[0] == Symbol("eval") {
			// optimize inner value
			p[1], _ = OptimizeEx(p[1], env, ome, true)
			return p
		} else if p[0] == Symbol("var") {
			// expand (it is faster)
			return NthLocalVar(ToInt(p[1]))
		} else {
			for i := 1; i < len(p); i++ {
				p[i] = OptimizeMatchPattern(nil, p[i], env, ome, ome2)
			}
		}
	}
	return pattern
}
func OptimizeParser(val Scmer, env *Env, ome *optimizerMetainfo, ignoreResult bool) Scmer {
	switch v := val.(type) {
		case []Scmer:
			if v[0] == Symbol("parser") {
				ign2 := ignoreResult
				if len(v) > 2 {
					ign2 = true // result of parser can be ignored when expr is executed
				}
				ome2 := ome.Copy()
				v[1] = OptimizeParser(v[1], env, &ome2, ign2) // syntax expr -> collect new variables
				if len(v) > 2 {
					v[2], _ = OptimizeEx(v[2], env, &ome2, !ignoreResult) // generator expr -> use variables
				}
				if len(v) > 3 {
					v[3], _ = OptimizeEx(v[3], env, ome, true) // delimiter expr
				}
			} else if v[0] == Symbol("define") {
				v[2] = OptimizeParser(v[2], env, ome, false)
				// TODO: numbered parameters v[1]
				if _, ok := ome.variableReplacement[v[1].(Symbol)]; ok {
					// remove entry from map so we really read out the real variable
					delete(ome.variableReplacement, v[1].(Symbol))
				}
			} else {
				// + * ? or atom regex
				for i := 1; i < len(v); i++ {
					v[i] = OptimizeParser(v[i], env, ome, ignoreResult)
				}
			}
	}
	// after optimization:
	// precompile parser if possible
	p := parseSyntax(val, env, ome, ignoreResult) // env = nil since we don't have the env yet
	if p != nil { // parseSyntax will return nil when the part is not translatable yet
		return p // part of that parser could be precompiled
	} else {
	}
	return val
}
