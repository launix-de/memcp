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
func OptimizeProcToSerialFunction(val Scmer, env *Env) func (...Scmer) Scmer {
	if result, ok := val.(func(...Scmer) Scmer); ok {
		return result // already optimized
	}
	// TODO: JIT

	// otherwise: precreate a lambda
	p := val.(Proc) // precast procedure
	en := &Env{make(Vars), make([]Scmer, p.NumVars), p.En, false} // reusable environment
	switch params := p.Params.(type) {
	case []Scmer: // default case: 
		return func (args ...Scmer) Scmer {
			if len(params) > len(args) {
				panic(fmt.Sprintf("Apply: function with %d parameters is supplied with %d arguments", len(params), len(args)))
			}
			for i, param := range params {
				en.Vars[param.(Symbol)] = args[i]
			}
			return Eval(p.Body, en)
		}
	default: // otherwise: param list is stored in a single variable
		return func (args ...Scmer) Scmer {
			en.Vars[params.(Symbol)] = args
			return Eval(p.Body, en)
		}
	}
	panic("value is not compilable: " + String(val))
}

// do preprocessing and optimization (Optimize is allowed to edit the value in-place)
func Optimize(val Scmer, env *Env) Scmer {
	ome := newOptimizerMetainfo()
	v := OptimizeEx(val, env, &ome)
	fmt.Println(SerializeToString(v, env, env))
	return v
}
type optimizerMetainfo struct {
	variableReplacement map[Symbol]Scmer
	lambda *Proc // when inside a Proc
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
	result.lambda = ome.lambda
	return
}
func OptimizeEx(val Scmer, env *Env, ome *optimizerMetainfo) Scmer {
	fmt.Println("optimize node ",String(val))
	// TODO: strip source code information (source, line, col)
	// TODO: static code analysis like escape analysis + replace memory-safe functions with in-place memory manipulating versions (e.g. in set_assoc)
	// TODO: inline use-once
	// TODO: inplace functions (map -> map_inplace, filter -> filter_inplace) will manipulate the first parameter instead of allocating something new
	// TODO: pure imperative functions (map, produce_map, produceN_map) that execute code and return nothing
	// TODO: value chaining -> produce+map+filter -> inplace append (based on pure imperative)
	// TODO: cons/merge->append
	switch v := val.(type) {
		case SourceInfo:
			// strip SourceInfo from lambda declarations
			return OptimizeEx(v.value, env, ome)
		case Symbol:
			// replace variables with their counterparts
			if replacement, ok := ome.variableReplacement[v]; ok {
				return replacement
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
					switch xv := v[1].(type) {
						case float64:
							return NthLocalVar(xv)
						case uint64:
							return NthLocalVar(xv)
						case int:
							return NthLocalVar(xv)
						// whatever comes
					}
				}
				// pack lambdas into objects
				if v[0] == Symbol("lambda") {
					switch si := v[1].(type) {
						case SourceInfo:
							// strip SourceInfo from lambda declarations
							v[1] = si.value
					}
					p := Proc{v[1], v[2], env, 0}
					p.Optimize(env, ome)
					return p
				}
				// all items:
				if v[0] != Symbol("quote") && v[0] != Symbol("match") {
					// optimize all other parameters
					for i := 1; i < len(v); i++ {
						v[i] = OptimizeEx(v[i], env, ome)
					}
				}
				if (v[0] == Symbol("set") || v[0] == Symbol("define")) && len(v) == 3 {
					if _, ok := v[1].(NthLocalVar); ok {
						// change symbol of set/define to setN
						v[0] = Symbol("setN")
					}
				}
				if v[0] == Symbol("match") {
					// TODO: optimize matches
					for i := 2; i < len(v); i+= 2 {
						v[i] = OptimizeEx(v[i], env, ome)
					}
					if len(v)%2 == 1 {
						v[len(v)-1] = OptimizeEx(v[len(v)-1], env, ome)
					}
				}
				if v[0] == Symbol("parser") {
					// TODO: precompile parsers
					/*
					if len(e) > 3 {
						value = NewParser(e[1], e[2], e[3], en)
					} else if len(e) > 2 {
						value = NewParser(e[1], e[2], nil, en)
					} else {
						value = NewParser(e[1], nil, nil, en)
					}*/
				}
			}
	}
	return val
}
func (p *Proc) Optimize(env *Env, ome *optimizerMetainfo) {
	// optimize lambdas
	// prepare to optimize body
	return
	ome2 := ome.Copy()
	switch params := p.Params.(type) {
		case []Scmer: // parameter list
			if p.NumVars != 0 {
				panic("lambda function with unnamed variables must not have a parameter list")
			}
			for _, s := range params {
				ome2.variableReplacement[s.(Symbol)] = NthLocalVar(p.NumVars)
				p.NumVars++
			}
		case Symbol: // parameter variable
			if p.NumVars != 0 {
				panic("lambda function with unnamed variables must not have a parameter list")
			}
			ome2.variableReplacement[params] = NthLocalVar(p.NumVars)
			p.NumVars++
		case nil: // optimized parameterless version
		default:
			panic("unknown lambda parameter: " + String(params))
	}
	ome2.lambda = p
	p.Params = nil // replace parameter list with
	p.Body = OptimizeEx(p.Body, env, &ome2) // optimize body
}
