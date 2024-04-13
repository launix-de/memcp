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
	en := &Env{make(Vars), make([]Scmer, p.NumVars), p.En, false} // reusable environment
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
					en.Vars[param.(Symbol)] = args[i]
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
	v := OptimizeEx(val, env, &ome)
	//fmt.Println(SerializeToString(v, env))
	return v
}
type optimizerMetainfo struct {
	variableReplacement map[Symbol]Scmer
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
	return
}
func (ome *optimizerMetainfo) RemoveSymbolsFromTree(tree Scmer) {
	switch v := tree.(type) {
	case Symbol:
		delete(ome.variableReplacement, v)
	case []Scmer:
		for i := 1; i < len(v); i++ {
			// recurse over parameters
			ome.RemoveSymbolsFromTree(v[i])
		}
	}
}
func OptimizeEx(val Scmer, env *Env, ome *optimizerMetainfo) Scmer {
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
			// strip SourceInfo from lambda declarations
			return OptimizeEx(v.value, env, ome)
		case Symbol:
			// replace variables with their counterparts
			if replacement, ok := ome.variableReplacement[v]; ok {
				return replacement
			}
			return val // TODO: remove this return once there is a solution to mask out prefetch variables

			// prefetch system functions (not working yet -> sometimes lambdas redefine variables which are not allowed to replace)
			xen := env.FindRead(v)
			if xen != nil {
				if v, ok := xen.Vars[v]; ok {
					return v
				} else {
					return val
				}
			}
		case []Scmer:
			if len(v) > 0 {
				if v[0] == Symbol("begin") {
					ome2 := ome.Copy() // inherit scope
					for i := 1; i < len(v) - 1; i++ {
						// TODO: v[i]'s return value is not used -> discard
					}
					for i := 1; i < len(v); i++ {
						v[i] = OptimizeEx(v[i], env, &ome2)
					}
					return v
				}
				// (var i) is a serialization artifact
				if v[0] == Symbol("var") && len(v) == 2 {
					return NthLocalVar(ToInt(v[1]))
				}
				// (unquote s) is a serialization artifact
				if v[0] == Symbol("unquote") && len(v) == 2 {
					if s, ok := v[1].(string); ok {
						return Symbol(s) // replace with the symbol directly
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
						emptyome := newOptimizerMetainfo()
						for i, item := range l {
							l[i] = OptimizeEx(item, nil, &emptyome)
							ome2.RemoveSymbolsFromTree(l[i]) // remove overrides
						}
					} else {
						ome2.RemoveSymbolsFromTree(v[1]) // remove overrides
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
					v[2] = OptimizeEx(v[2], env, &ome2) // optimize body
					return v
				}

				// now all the special cases

				// set/define
				if (v[0] == Symbol("set") || v[0] == Symbol("define")) && len(v) == 3 {
					if s, ok := v[1].(Symbol); ok {
						if vp, ok := ome.variableReplacement[s]; ok {
							if lv, ok := vp.(NthLocalVar); ok {
								v[1] = lv // set local var -> replace with (var i)
							}
						} else {
							// TODO: new variable -> increase variable count?? inline?
						}
					}
					if _, ok := v[1].(NthLocalVar); ok {
						// change symbol of set/define to setN
						v[0] = Symbol("setN")
					}
					v[2] = OptimizeEx(v[2], env, ome)
					// TODO: check if we could remove the set instruction and inline the value if it occurs only once
				} else if v[0] == Symbol("match") {
					// TODO: optimize matches with nvars and your own ome
					v[1] = OptimizeEx(v[1], env, ome)
					/* code is deactivated since variables can be overwritten! */
					for i := 3; i < len(v); i+= 2 {
						// for each pattern
						ome2 := ome.Copy()
						ome2.RemoveSymbolsFromTree(v[i-1])
						v[i] = OptimizeEx(v[i], env, &ome2)
					}
					if len(v)%2 == 1 {
						// last item
						v[len(v)-1] = OptimizeEx(v[len(v)-1], env, ome)
					}
				} else if v[0] == Symbol("parser") {
					return OptimizeParser(v, env, ome, false)

				// last but not least: recurse over the arguments when we aren't a special case
				} else if v[0] != Symbol("quote") {
					// optimize all other parameters
					for i := 1; i < len(v); i++ {
						v[i] = OptimizeEx(v[i], env, ome)
					}
				}
			}
	}
	return val
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
					v[2] = OptimizeEx(v[2], env, &ome2) // generator expr -> use variables
				}
				if len(v) > 3 {
					v[3] = OptimizeEx(v[3], env, ome) // delimiter expr
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
