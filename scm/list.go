/*
Copyright (C) 2023-2026  Carl-Philip Hänsch
Copyright (C) 2013  Pieter Kelchtermans (originally licensed unter WTFPL 2.0)

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
/*
 * A minimal Scheme interpreter, as seen in lis.py and SICP
 * http://norvig.com/lispy.html
 * http://mitpress.mit.edu/sicp/full-text/sicp/book/node77.html
 *
 * Pieter Kelchtermans 2013
 * LICENSE: WTFPL 2.0
 */
package scm

import "fmt"
import "runtime"
import "sync"

// optimizeMap is the optimizer hook for `map`. It applies default optimization
// (including FirstParameterMutable swap to map_mut), then fuses
// (map (produceN N) fn) → (produceN N fn) to eliminate the intermediate list.
func optimizeMap(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	// Run default optimization first (handles map → map_mut swap etc.)
	result, td := oc.applyDefaultOptimization(v, useResult, "map_mut")
	// Check if the optimized result is still a call to map/map_mut
	if result.IsSlice() {
		rv := result.Slice()
		if len(rv) == 3 {
			if sym, ok := scmerSymbol(rv[0]); ok && (sym == "map" || sym == "map_mut") {
				// Check if arg 1 is a (produceN N) call
				if rv[1].IsSlice() {
					inner := rv[1].Slice()
					if len(inner) == 2 {
						if isym, ok := scmerSymbol(inner[0]); ok && isym == "produceN" {
							// Fuse: (map (produceN N) fn) → (produceN N fn)
							return NewSlice([]Scmer{inner[0], inner[1], rv[2]}), td
						}
					}
				}
			}
		}
	}
	return result, td
}

// optimizeProduceN rewrites (produceN ...) to (produceN_mut ... nil) when the
// result is unused, so runtime can avoid result allocation.
func optimizeProduceN(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	result, td := oc.ApplyDefaultOptimization(v, useResult)
	if useResult || !result.IsSlice() {
		return result, td
	}
	rv := result.Slice()
	if len(rv) < 2 {
		return result, td
	}
	if sym, ok := scmerSymbol(rv[0]); !ok || sym != "produceN" {
		return result, td
	}
	out := make([]Scmer, 0, len(rv)+1)
	out = append(out, NewSymbol("produceN_mut"))
	out = append(out, rv[1:]...)
	if len(rv) == 2 {
		out = append(out, NewNil())
	}
	return NewSlice(out), &TypeDescriptor{}
}

// optimizeParallelN rewrites (parallelN ...) to (parallelN_mut ... nil) when
// the result is unused, so runtime can avoid result allocation.
func optimizeParallelN(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	result, td := oc.ApplyDefaultOptimization(v, useResult)
	if useResult || !result.IsSlice() {
		return result, td
	}
	rv := result.Slice()
	if len(rv) < 3 {
		return result, td
	}
	if sym, ok := scmerSymbol(rv[0]); !ok || sym != "parallelN" {
		return result, td
	}
	out := make([]Scmer, 0, len(rv)+1)
	out = append(out, NewSymbol("parallelN_mut"))
	out = append(out, rv[1:]...)
	if len(rv) == 3 {
		out = append(out, NewNil())
	}
	return NewSlice(out), &TypeDescriptor{}
}

func asSlice(v Scmer, ctx string) []Scmer {
	// Treat nil as empty list so higher-level code can be concise
	if v.IsNil() {
		return []Scmer{}
	}
	if v.IsSlice() {
		return v.Slice()
	}
	panic(fmt.Sprintf("%s expects a list, got %s", ctx, v.String()))
}

func asAssoc(v Scmer, ctx string) ([]Scmer, *FastDict) {
	// Treat nil as empty dictionary (assoc list)
	if v.IsNil() {
		return []Scmer{}, nil
	}
	if v.IsSlice() {
		return v.Slice(), nil
	}
	if v.IsFastDict() {
		return nil, v.FastDict()
	}
	panic(fmt.Sprintf("%s expects a dictionary", ctx))
}

func init_list() {
	// list functions
	DeclareTitle("Lists")

	// list is already in Globalenv.Vars (scm.go init); register it
	// in declarations so serialization can resolve the function pointer.
	Declare(&Globalenv, &Declaration{
		Name: "list",
		Desc: "constructs a list from its arguments",
		Fn: List,
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "any", ParamName: "items", ParamDesc: "items to put into the list", Variadic: true},
			},
			Return: &TypeDescriptor{Kind: "list"},
			Const: true,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "count",
		Desc: "counts the number of elements in the list",
		Fn: func(a ...Scmer) Scmer {
			if a[0].GetTag() == tagSlice {
				return NewInt(int64(len(a[0].Slice())))
			}
			if a[0].GetTag() == tagFastDict {
				fd := a[0].FastDict()
				if fd == nil {
					return NewInt(0)
				}
				return NewInt(int64(len(fd.Pairs)))
			}
			panic("count expects a list")
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "base list", NoEscape: true},
			},
			Return: &TypeDescriptor{Kind: "int"},
			Const: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "nth",
		Desc: "get the nth item of a list",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "nth")
			idx := int(a[1].Int())
			if idx < 0 || idx >= len(list) {
				panic("nth index out of range")
			}
			return list[idx]
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "base list", NoEscape: true},
				{Kind: "number", ParamName: "index", ParamDesc: "index beginning from 0"},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "slice",
		Desc: "extract a sublist from start (inclusive) to end (exclusive).\n(slice list start end) returns elements list[start..end).",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "slice")
			start := int(a[1].Int())
			end := int(a[2].Int())
			if start < 0 {
				start = 0
			}
			if end > len(list) {
				end = len(list)
			}
			if start >= end {
				return NewSlice([]Scmer{})
			}
			result := make([]Scmer, end-start)
			copy(result, list[start:end])
			return NewSlice(result)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "base list", NoEscape: true},
				{Kind: "number", ParamName: "start", ParamDesc: "start index (inclusive)"},
				{Kind: "number", ParamName: "end", ParamDesc: "end index (exclusive)"},
			},
			Return: &TypeDescriptor{Kind: "list"},
			Const: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "reverse",
		Desc: "returns a new list with elements in reversed order.",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "reverse")
			n := len(list)
			result := make([]Scmer, n)
			for i := 0; i < n; i++ {
				result[i] = list[n-1-i]
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list to reverse", NoEscape: true},
			},
			Return: FreshAlloc,
			Const: true,
			Optimize: FirstParameterMutable("reverse_mut"),
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "append",
		Desc: "appends items to a list and return the extended list.\nThe original list stays unharmed.",
		Fn: func(a ...Scmer) Scmer {
			base := append([]Scmer{}, asSlice(a[0], "append")...)
			base = append(base, a[1:]...)
			return NewSlice(base)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "base list"},
				{Kind: "any", ParamName: "item...", ParamDesc: "items to add", Variadic: true},
			},
			Return: FreshAlloc,
			Const: true,
			Optimize: FirstParameterMutable("append_mut"),
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "append_unique",
		Desc: "appends items to a list but only if they are new.\nThe original list stays unharmed.",
		Fn: func(a ...Scmer) Scmer {
			list := append([]Scmer{}, asSlice(a[0], "append_unique")...)
			for _, el := range a[1:] {
				for _, el2 := range list {
					if Equal(el, el2) {
						// ignore duplicates
						goto skipItem
					}
				}
				list = append(list, el)
			skipItem:
			}
			return NewSlice(list)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "base list"},
				{Kind: "any", ParamName: "item...", ParamDesc: "items to add", Variadic: true},
			},
			Return: FreshAlloc,
			Const: true,
			Optimize: FirstParameterMutable("append_unique_mut"),
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "cons",
		Desc: "constructs a list from a head and a tail list",
		Fn: func(a ...Scmer) Scmer {
			car := a[0]
			if a[1].GetTag() == tagSlice {
				return NewSlice(append([]Scmer{car}, a[1].Slice()...))
			}
			return NewSlice([]Scmer{car, a[1]})
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "any", ParamName: "car", ParamDesc: "new head element"},
				{Kind: "list", ParamName: "cdr", ParamDesc: "tail that is appended after car", NoEscape: true},
			},
			Return: FreshAlloc,
			Const: true,
			Optimize: optimizeCons,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "car",
		Desc: "extracts the head of a list",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "car")
			if len(list) == 0 {
				panic("car on empty list")
			}
			return list[0]
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list", NoEscape: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "cdr",
		Desc: "extracts the tail of a list\nThe tail of a list is a list with all items except the head.",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "cdr")
			if len(list) == 0 {
				return NewSlice([]Scmer{})
			}
			return NewSlice(list[1:])
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list", NoEscape: true},
			},
			Return: FreshAlloc,
			Const: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "cadr",
		Desc: "extracts the second element of a list.\nEquivalent to (car (cdr x)).",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "cadr")
			if len(list) < 2 {
				panic("cadr on list with fewer than 2 elements")
			}
			return list[1]
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list", NoEscape: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "zip",
		Desc: "swaps the dimension of a list of lists. If one parameter is given, it is a list of lists that is flattened. If multiple parameters are given, they are treated as the components that will be zipped into the sub list",
		Fn: func(a ...Scmer) Scmer {
			lists := a
			if len(a) == 1 {
				lists = asSlice(a[0], "zip")
			}
			if len(lists) == 0 {
				return NewSlice([]Scmer{})
			}
			first := asSlice(lists[0], "zip element")
			size := len(first)
			result := make([]Scmer, size)
			for i := 0; i < size; i++ {
				subresult := make([]Scmer, len(lists))
				for j, v := range lists {
					current := asSlice(v, "zip item")
					if i >= len(current) {
						panic("zip expects lists of equal length")
					}
					subresult[j] = current[i]
				}
				result[i] = NewSlice(subresult)
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "any", ParamName: "list", ParamDesc: "list of lists of items", NoEscape: true, Variadic: true},
			},
			Return: FreshAlloc,
			Const: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "merge",
		Desc: "flattens a list of lists into a list containing all the subitems. If one parameter is given, it is a list of lists that is flattened. If multiple parameters are given, they are treated as lists that will be merged into one",
		Fn: func(a ...Scmer) Scmer {
			lists := a
			if len(a) == 1 {
				lists = asSlice(a[0], "merge")
			}
			size := 0
			for _, v := range lists {
				size += len(asSlice(v, "merge item"))
			}
			result := make([]Scmer, 0, size)
			for _, v := range lists {
				result = append(result, asSlice(v, "merge item")...)
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "any", ParamName: "list", ParamDesc: "list of lists of items", NoEscape: true, Variadic: true},
			},
			Return: FreshAlloc,
			Const: true,
			Optimize: optimizeMerge,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "merge_unique",
		Desc: "flattens a list of lists into a list containing all the subitems. Duplicates are filtered out.",
		Fn: func(a ...Scmer) Scmer {
			lists := a
			if len(a) == 1 {
				lists = asSlice(a[0], "merge_unique")
			}
			size := 0
			for _, v := range lists {
				size += len(asSlice(v, "merge_unique item"))
			}
			result := make([]Scmer, 0, size)
			for _, v := range lists {
				for _, el := range asSlice(v, "merge_unique item") {
					duplicate := false
					for _, existing := range result {
						if Equal(el, existing) {
							duplicate = true
							break
						}
					}
					if !duplicate {
						result = append(result, el)
					}
				}
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list of lists of items", NoEscape: true, Variadic: true},
			},
			Return: FreshAlloc,
			Const: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "has?",
		Desc: "checks if a list has a certain item (equal?)",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "has?")
			for _, v := range list {
				if Equal(a[1], v) {
					return NewBool(true)
				}
			}
			return NewBool(false)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "haystack", ParamDesc: "list to search in", NoEscape: true},
				{Kind: "any", ParamName: "needle", ParamDesc: "item to search for"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "filter",
		Desc: "returns a list that only contains elements that pass the filter function",
		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "filter")
			result := make([]Scmer, 0, len(input))
			fn := OptimizeProcToSerialFunction(a[1])
			for _, v := range input {
				if fn(v).Bool() {
					result = append(result, v)
				}
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list that has to be filtered", NoEscape: true},
				{Kind: "func", ParamName: "condition", ParamDesc: "filter condition func(item)->bool", Params: []*TypeDescriptor{{Kind: "any", ParamName: "item"}}, Return: &TypeDescriptor{Kind: "bool"}},
			},
			Return: FreshAlloc,
			Const: true,
			Optimize: FirstParameterMutable("filter_mut"),
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "find",
		Desc: "returns the first list element that passes the condition function, or nil/default if none matches",
		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "find")
			fn := OptimizeProcToSerialFunction(a[1])
			for _, v := range input {
				if fn(v).Bool() {
					return v
				}
			}
			if len(a) >= 3 {
				return a[2]
			}
			return NewNil()
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list to search", NoEscape: true},
				{Kind: "func", ParamName: "condition", ParamDesc: "predicate func(any)->bool that is applied until the first match", Params: []*TypeDescriptor{{Kind: "any", ParamName: "item"}}, Return: &TypeDescriptor{Kind: "bool"}},
				{Kind: "any", ParamName: "default", ParamDesc: "optional default value if nothing matches", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "map",
		Desc: "returns a list that contains the results of a map function that is applied to the list",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "map")
			result := make([]Scmer, len(list))
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				result[i] = fn(v)
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list that has to be mapped", NoEscape: true},
				{Kind: "func", ParamName: "map", ParamDesc: "map function func(any)->any that is applied to each item", Params: []*TypeDescriptor{{Kind: "any", ParamName: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: FreshAlloc,
			Const: true,
			Optimize: optimizeMap,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "mapIndex",
		Desc: "returns a list that contains the results of a map function that is applied to the list",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "mapIndex")
			result := make([]Scmer, len(list))
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				result[i] = fn(NewInt(int64(i)), v)
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list that has to be mapped", NoEscape: true},
				{Kind: "func", ParamName: "map", ParamDesc: "map function func(i, any)->any that is applied to each item", Params: []*TypeDescriptor{{Kind: "int", ParamName: "index"}, {Kind: "any", ParamName: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: FreshAlloc,
			Const: true,
			Optimize: FirstParameterMutable("mapIndex_mut"),
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "flatmap",
		Desc: "applies fn to each element and flattens the results into a single list (map+merge in one pass, no intermediate allocation)",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "flatmap")
			fn := OptimizeProcToSerialFunction(a[1])
			result := make([]Scmer, 0, len(list))
			for _, v := range list {
				mapped := fn(v)
				if mapped.IsNil() {
					continue
				}
				result = append(result, asSlice(mapped, "flatmap result")...)
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list to map and flatten", NoEscape: true},
				{Kind: "func", ParamName: "fn", ParamDesc: "func(item)->list that returns a list per element", Params: []*TypeDescriptor{{Kind: "any", ParamName: "item"}}, Return: &TypeDescriptor{Kind: "list"}},
			},
			Return: FreshAlloc,
			Const: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "reduce",
		Desc: "returns a list that contains the result of a map function",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "reduce")
			fn := OptimizeProcToSerialFunction(a[1])
			result := NewNil()
			i := 0
			if len(a) > 2 {
				result = a[2]
			} else if len(list) > 0 {
				result = list[0]
				i = 1
			}
			for i < len(list) {
				result = fn(result, list[i])
				i++
			}
			return result
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list that has to be reduced", NoEscape: true},
				{Kind: "func", Params: []*TypeDescriptor{{Transfer: true, ParamName: "acc"}, {ParamName: "item"}}, ParamName: "reduce", ParamDesc: "reduce function func(any any)->any where the first parameter is the accumulator, the second is a list item", Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "any", ParamName: "neutral", ParamDesc: "(optional) initial value of the accumulator, defaults to nil", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const: true,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "produce",
		Desc: "returns a list that contains produced items - it works like for(state = startstate, condition(state), state = iterator(state)) {yield state}",
		Fn: func(a ...Scmer) Scmer {
			result := make([]Scmer, 0)
			state := a[0]
			condition := OptimizeProcToSerialFunction(a[1])
			iterator := OptimizeProcToSerialFunction(a[2])
			for condition(state).Bool() {
				result = append(result, state)
				state = iterator(state)
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "any", ParamName: "startstate", ParamDesc: "start state to begin with"},
				{Kind: "func", ParamName: "condition", ParamDesc: "func that returns true whether the state will be inserted into the result or the loop is stopped", Params: []*TypeDescriptor{{Kind: "any", ParamName: "state"}}, Return: &TypeDescriptor{Kind: "bool"}},
				{Kind: "func", ParamName: "iterator", ParamDesc: "func that produces the next state", Params: []*TypeDescriptor{{Kind: "any", ParamName: "state"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: FreshAlloc,
			Const: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "produceN",
		Desc: "returns a list with numbers from 0..n-1, optionally mapped through a function",
		Fn: func(a ...Scmer) Scmer {
			n := int(a[0].Int())
			if n < 0 {
				n = 0
			}
			result := make([]Scmer, n)
			if len(a) > 1 && !a[1].IsNil() {
				// fused produceN+map: generate and transform in one pass
				fn := OptimizeProcToSerialFunction(a[1])
				for i := 0; i < n; i++ {
					result[i] = fn(NewInt(int64(i)))
				}
			} else {
				for i := 0; i < n; i++ {
					result[i] = NewInt(int64(i))
				}
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "number", ParamName: "n", ParamDesc: "number of elements to produce"},
				{Kind: "func", ParamName: "fn", ParamDesc: "(optional) map function applied to each index", Optional: true, Params: []*TypeDescriptor{{Kind: "int", ParamName: "index"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: FreshAlloc,
			Const: true,
			Optimize: optimizeProduceN,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "parallelN",
		Desc: "returns a list with numbers from 0..n-1 mapped in parallel through a function",
		Fn: func(a ...Scmer) Scmer {
			n := int(a[0].Int())
			if n < 0 {
				n = 0
			}
			result := make([]Scmer, n)
			fn := a[1]
			needsSerializedCall := fn.GetTag() == tagFunc || fn.GetTag() == tagFuncEnv
			var fnMu sync.Mutex
			callFn := func(i int) Scmer {
				if needsSerializedCall {
					fnMu.Lock()
					defer fnMu.Unlock()
				}
				return Apply(fn, NewInt(int64(i)))
			}
			workers := runtime.NumCPU()
			if workers < 1 {
				workers = 1
			}
			if workers > n {
				workers = n
			}
			jobs := make(chan int, workers)
			errs := make(chan any, workers)
			var wg sync.WaitGroup
			for w := 0; w < workers; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for i := range jobs {
						func() {
							defer func() {
								if r := recover(); r != nil {
									errs <- r
								}
							}()
							result[i] = callFn(i)
						}()
					}
				}()
			}
			for i := 0; i < n; i++ {
				jobs <- i
			}
			close(jobs)
			wg.Wait()
			close(errs)
			for err := range errs {
				if err != nil {
					panic(err)
				}
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "number", ParamName: "n", ParamDesc: "number of elements to produce"},
				{Kind: "func", ParamName: "fn", ParamDesc: "map function applied to each index in parallel", Params: []*TypeDescriptor{{Kind: "int", ParamName: "index"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: FreshAlloc,
			Const: true,
			Optimize: optimizeParallelN,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "produceN_mut",
		Desc: "in-place produceN variant (optimizer-only)",
		Fn: func(a ...Scmer) Scmer {
			n := int(a[0].Int())
			if n < 0 {
				n = 0
			}
			fn := OptimizeProcToSerialFunction(a[1])
			if len(a) < 3 || a[2].IsNil() {
				for i := 0; i < n; i++ {
					fn(NewInt(int64(i)))
				}
				return NewNil()
			}
			result := asSlice(a[2], "produceN_mut target")
			if len(result) < n {
				panic("produceN_mut target too small")
			}
			result = result[:n]
			for i := 0; i < n; i++ {
				result[i] = fn(NewInt(int64(i)))
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "number", ParamName: "n", ParamDesc: "number of elements to produce"},
				{Kind: "func", ParamName: "fn", ParamDesc: "map function applied to each index", Params: []*TypeDescriptor{{Kind: "int", ParamName: "index"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "list", ParamName: "target", ParamDesc: "(optional) preallocated target list", NoEscape: true, Optional: true},
			},
			Return: &TypeDescriptor{Kind: "list"},
			Const: true,
			Forbidden: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "parallelN_mut",
		Desc: "in-place parallelN variant (optimizer-only)",
		Fn: func(a ...Scmer) Scmer {
			n := int(a[0].Int())
			if n < 0 {
				n = 0
			}
			fn := a[1]
			needsSerializedCall := fn.GetTag() == tagFunc || fn.GetTag() == tagFuncEnv
			var fnMu sync.Mutex
			callFn := func(i int) Scmer {
				if needsSerializedCall {
					fnMu.Lock()
					defer fnMu.Unlock()
				}
				return Apply(fn, NewInt(int64(i)))
			}
			workers := runtime.NumCPU()
			if workers < 1 {
				workers = 1
			}
			if workers > n {
				workers = n
			}
			if len(a) < 3 || a[2].IsNil() {
				jobs := make(chan int, workers)
				errs := make(chan any, workers)
				var wg sync.WaitGroup
				for w := 0; w < workers; w++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						for i := range jobs {
							func() {
								defer func() {
									if r := recover(); r != nil {
										errs <- r
									}
								}()
								callFn(i)
							}()
						}
					}()
				}
				for i := 0; i < n; i++ {
					jobs <- i
				}
				close(jobs)
				wg.Wait()
				close(errs)
				for err := range errs {
					if err != nil {
						panic(err)
					}
				}
				return NewNil()
			}
			result := asSlice(a[2], "parallelN_mut target")
			if len(result) < n {
				panic("parallelN_mut target too small")
			}
			result = result[:n]
			jobs := make(chan int, workers)
			errs := make(chan any, workers)
			var wg sync.WaitGroup
			for w := 0; w < workers; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for i := range jobs {
						func() {
							defer func() {
								if r := recover(); r != nil {
									errs <- r
								}
							}()
							result[i] = callFn(i)
						}()
					}
				}()
			}
			for i := 0; i < n; i++ {
				jobs <- i
			}
			close(jobs)
			wg.Wait()
			close(errs)
			for err := range errs {
				if err != nil {
					panic(err)
				}
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "number", ParamName: "n", ParamDesc: "number of elements to produce"},
				{Kind: "func", ParamName: "fn", ParamDesc: "map function applied to each index in parallel", Params: []*TypeDescriptor{{Kind: "int", ParamName: "index"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "list", ParamName: "target", ParamDesc: "(optional) preallocated target list", NoEscape: true, Optional: true},
			},
			Return: &TypeDescriptor{Kind: "list"},
			Const: true,
			Forbidden: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "list?",
		Desc: "checks if a value is a list",
		Fn: func(a ...Scmer) Scmer {
			if a[0].IsSlice() {
				return NewBool(true)
			}
			return NewBool(false)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "any", ParamName: "value", ParamDesc: "value to check"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "contains?",
		Desc: "checks if a value is in a list; uses the equal?? operator",
		Fn: func(a ...Scmer) Scmer {
			arr := asSlice(a[0], "contains?")
			for _, v := range arr {
				if Equal(v, a[1]) {
					return NewBool(true)
				}
			}
			return NewBool(false)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list to check", NoEscape: true},
				{Kind: "any", ParamName: "value", ParamDesc: "value to check"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
		},
	})

	// dictionary functions
	DeclareTitle("Associative Lists / Dictionaries")

	Declare(&Globalenv, &Declaration{
		Name: "filter_assoc",
		Desc: "returns a filtered dictionary according to a filter function",
		Fn: func(a ...Scmer) Scmer {
			result := make([]Scmer, 0)
			fn := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "filter_assoc"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					if fn(slice[i], slice[i+1]).Bool() {
						result = append(result, slice[i], slice[i+1])
					}
				}
			} else {
				fd.Iterate(func(k, v Scmer) bool {
					if fn(k, v).Bool() {
						result = append(result, k, v)
					}
					return true
				})
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "dictionary that has to be filtered", NoEscape: true},
				{Kind: "func", ParamName: "condition", ParamDesc: "filter function func(string any)->bool where the first parameter is the key, the second is the value", Params: []*TypeDescriptor{{Kind: "string", ParamName: "key"}, {Kind: "any", ParamName: "value"}}, Return: &TypeDescriptor{Kind: "bool"}},
			},
			Return: FreshAlloc,
			Const: true,
			Optimize: FirstParameterMutable("filter_assoc_mut"),
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "find_assoc",
		Desc: "returns the first key/value pair that passes the condition function, or nil/default if none matches",
		Fn: func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "find_assoc"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					if fn(slice[i], slice[i+1]).Bool() {
						return NewSlice([]Scmer{slice[i], slice[i+1]})
					}
				}
			} else {
				var result Scmer
				found := false
				fd.Iterate(func(k, v Scmer) bool {
					if fn(k, v).Bool() {
						result = NewSlice([]Scmer{k, v})
						found = true
						return false
					}
					return true
				})
				if found {
					return result
				}
			}
			if len(a) >= 3 {
				return a[2]
			}
			return NewNil()
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "dictionary to search", NoEscape: true},
				{Kind: "func", ParamName: "condition", ParamDesc: "predicate func(string any)->bool that is applied until the first match", Params: []*TypeDescriptor{{Kind: "string", ParamName: "key"}, {Kind: "any", ParamName: "value"}}, Return: &TypeDescriptor{Kind: "bool"}},
				{Kind: "any", ParamName: "default", ParamDesc: "optional default value if nothing matches", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "map_assoc",
		Desc: "returns a mapped dictionary according to a map function\nKeys will stay the same but values are mapped.",
		Fn: func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "map_assoc"); fd == nil {
				result := make([]Scmer, len(slice))
				var key Scmer
				for i, v := range slice {
					if i%2 == 0 {
						result[i] = v
						key = v
					} else {
						result[i] = fn(key, v)
					}
				}
				return NewSlice(result)
			} else {
				result := make([]Scmer, 0, len(fd.Pairs))
				fd.Iterate(func(k, v Scmer) bool {
					result = append(result, k, fn(k, v))
					return true
				})
				return NewSlice(result)
			}
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "dictionary that has to be mapped", NoEscape: true},
				{Kind: "func", ParamName: "map", ParamDesc: "map function func(string any)->any where the first parameter is the key, the second is the value. It must return the new value.", Params: []*TypeDescriptor{{Kind: "string", ParamName: "key"}, {Kind: "any", ParamName: "value"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: FreshAlloc,
			Const: true,
			Optimize: FirstParameterMutable("map_assoc_mut"),
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "reduce_assoc",
		Desc: "reduces a dictionary according to a reduce function",
		Fn: func(a ...Scmer) Scmer {
			result := a[2]
			reduce := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "reduce_assoc"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					result = reduce(result, slice[i], slice[i+1])
				}
			} else {
				fd.Iterate(func(k, v Scmer) bool { result = reduce(result, k, v); return true })
			}
			return result
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "dictionary that has to be reduced", NoEscape: true},
				{Kind: "func", Params: []*TypeDescriptor{{Transfer: true, ParamName: "acc"}, {ParamName: "key"}, {ParamName: "value"}}, ParamName: "reduce", ParamDesc: "reduce function func(any string any)->any where the first parameter is the accumulator, second is key, third is value. It must return the new accumulator.", Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "any", ParamName: "neutral", ParamDesc: "initial value for the accumulator"},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "has_assoc?",
		Desc: "checks if a dictionary has a key present",
		Fn: func(a ...Scmer) Scmer {
			if slice, fd := asAssoc(a[0], "has_assoc?"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					if Equal(slice[i], a[1]) {
						return NewBool(true)
					}
				}
			} else {
				if _, ok := fd.Get(a[1]); ok {
					return NewBool(true)
				}
			}
			return NewBool(false)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "dictionary that has to be checked", NoEscape: true},
				{Kind: "string", ParamName: "key", ParamDesc: "key to test"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "get_assoc",
		Desc: "gets a value from a dictionary by key, returns nil if not found",
		Fn: func(a ...Scmer) Scmer {
			if slice, fd := asAssoc(a[0], "get_assoc"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					if Equal(slice[i], a[1]) {
						return slice[i+1]
					}
				}
			} else {
				if v, ok := fd.Get(a[1]); ok {
					return v
				}
			}
			// Return default value if provided, otherwise nil
			if len(a) >= 3 {
				return a[2]
			}
			return NewNil()
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "dictionary to look up", NoEscape: true},
				{Kind: "any", ParamName: "key", ParamDesc: "key to look up"},
				{Kind: "any", ParamName: "default", ParamDesc: "optional default value if key not found", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "extract_assoc",
		Desc: "applies a function (key value) on the dictionary and returns the results as a flat list",
		Fn: func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "extract_assoc"); fd == nil {
				result := make([]Scmer, len(slice)/2)
				var key Scmer
				for i, v := range slice {
					if i%2 == 0 {
						key = v
					} else {
						result[i/2] = fn(key, v)
					}
				}
				return NewSlice(result)
			} else {
				result := make([]Scmer, 0, len(fd.Pairs)/2)
				fd.Iterate(func(k, v Scmer) bool {
					result = append(result, fn(k, v))
					return true
				})
				return NewSlice(result)
			}
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "dictionary that has to be checked", NoEscape: true},
				{Kind: "func", ParamName: "map", ParamDesc: "func(key, value)->any that extracts one element per key-value pair", Params: []*TypeDescriptor{{Kind: "string", ParamName: "key"}, {Kind: "any", ParamName: "value"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: FreshAlloc,
			Const: true,
			Optimize: FirstParameterMutable("extract_assoc_mut"),
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "set_assoc",
		Desc: "returns a new dictionary where a single value has been changed.\nThe original dictionary is not modified.",
		Fn: func(a ...Scmer) Scmer {
			var mergeFn func(Scmer, Scmer) Scmer
			if len(a) > 3 {
				mfn := OptimizeProcToSerialFunction(a[3])
				mergeFn = func(oldV, newV Scmer) Scmer { return mfn(oldV, newV) }
			}
			slice, fd := asAssoc(a[0], "set_assoc")
			if fd == nil {
				// defensive copy — set_assoc must not mutate the original
				list := append([]Scmer{}, slice...)
				for i := 0; i < len(list); i += 2 {
					if Equal(list[i], a[1]) {
						if mergeFn != nil {
							list[i+1] = mergeFn(list[i+1], a[2])
						} else {
							list[i+1] = a[2]
						}
						return NewSlice(list)
					}
				}
				list = append(list, a[1], a[2])
				if len(list) >= 10 {
					fd := NewFastDictValue(len(list)/2 + 4)
					for i := 0; i < len(list); i += 2 {
						fd.Set(list[i], list[i+1], nil)
					}
					return NewFastDict(fd)
				}
				return NewSlice(list)
			} else {
				fd = fd.Copy()
				fd.Set(a[1], a[2], mergeFn)
				return NewFastDict(fd)
			}
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "input dictionary"},
				{Kind: "string", ParamName: "key", ParamDesc: "key that has to be set"},
				{Kind: "any", ParamName: "value", ParamDesc: "new value to set"},
				{Kind: "func", ParamName: "merge", ParamDesc: "(optional) func(any any)->any that is called when a value is overwritten. The first parameter is the old value, the second is the new value. It must return the merged value that shall be physically stored in the new dictionary.", Optional: true, Params: []*TypeDescriptor{{Kind: "any", ParamName: "old"}, {Kind: "any", ParamName: "new"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: FreshAlloc,
			Const: true,
			Optimize: FirstParameterMutable("set_assoc_mut"),
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "merge_assoc",
		Desc: "returns a dictionary where all keys from dict1 and all keys from dict2 are present.\nIf a key is present in both inputs, the second one will be dominant so the first value will be overwritten unless you provide a merge function",
		Fn: func(a ...Scmer) Scmer {
			setAssoc := OptimizeProcToSerialFunction(Globalenv.Vars["set_assoc"])
			dst := a[0]
			if slice, fd := asAssoc(a[1], "merge_assoc"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					if len(a) > 2 {
						dst = setAssoc(dst, slice[i], slice[i+1], a[2])
					} else {
						dst = setAssoc(dst, slice[i], slice[i+1])
					}
				}
			} else {
				if len(a) > 2 {
					fd.Iterate(func(k, v Scmer) bool { dst = setAssoc(dst, k, v, a[2]); return true })
				} else {
					fd.Iterate(func(k, v Scmer) bool { dst = setAssoc(dst, k, v); return true })
				}
			}
			return dst
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict1", ParamDesc: "first input dictionary that has to be changed. You must not use this value again."},
				{Kind: "list", ParamName: "dict2", ParamDesc: "input dictionary that contains the new values that have to be added"},
				{Kind: "func", ParamName: "merge", ParamDesc: "(optional) func(any any)->any that is called when a value is overwritten. The first parameter is the old value, the second is the new value from dict2. It must return the merged value that shall be pysically stored in the new dictionary.", Optional: true, Params: []*TypeDescriptor{{Kind: "any", ParamName: "old"}, {Kind: "any", ParamName: "new"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: FreshAlloc,
			Const: true,
			Optimize: FirstParameterMutable("merge_assoc_mut"),
		},
	})

	// _mut variants: optimizer-only, forbidden from .scm code
	// Tier 1: same-length, zero-copy

	Declare(&Globalenv, &Declaration{
		Name: "map_mut",
		Desc: "in-place map (optimizer-only)",
		Fn: func(a ...Scmer) Scmer {
			list := a[0].Slice()
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				list[i] = fn(v)
			}
			return NewSlice(list)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "owned list to map in-place"},
				{Kind: "func", ParamName: "map", ParamDesc: "map function", Params: []*TypeDescriptor{{Kind: "any", ParamName: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: FreshAlloc,
			Const: true,
			Forbidden: true,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "mapIndex_mut",
		Desc: "in-place mapIndex (optimizer-only)",
		Fn: func(a ...Scmer) Scmer {
			list := a[0].Slice()
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				list[i] = fn(NewInt(int64(i)), v)
			}
			return NewSlice(list)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "owned list to map in-place"},
				{Kind: "func", ParamName: "map", ParamDesc: "map function func(i, any)->any", Params: []*TypeDescriptor{{Kind: "int", ParamName: "index"}, {Kind: "any", ParamName: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: FreshAlloc,
			Const: true,
			Forbidden: true,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "map_assoc_mut",
		Desc: "in-place map_assoc (optimizer-only, slice path only)",
		Fn: func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "map_assoc_mut"); fd == nil {
				var key Scmer
				for i, v := range slice {
					if i%2 == 0 {
						key = v
					} else {
						slice[i] = fn(key, v)
					}
				}
				return NewSlice(slice)
			} else {
				// FastDict path: cannot mutate in-place, fall back to allocating
				result := make([]Scmer, 0, len(fd.Pairs))
				fd.Iterate(func(k, v Scmer) bool {
					result = append(result, k, fn(k, v))
					return true
				})
				return NewSlice(result)
			}
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "owned dictionary to map in-place"},
				{Kind: "func", ParamName: "map", ParamDesc: "map function func(key, value)->value", Params: []*TypeDescriptor{{Kind: "string", ParamName: "key"}, {Kind: "any", ParamName: "value"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: FreshAlloc,
			Const: true,
			Forbidden: true,
		},
	})

	// Tier 2: shrinking, write-cursor

	Declare(&Globalenv, &Declaration{
		Name: "filter_mut",
		Desc: "in-place filter (optimizer-only)",
		Fn: func(a ...Scmer) Scmer {
			input := a[0].Slice()
			fn := OptimizeProcToSerialFunction(a[1])
			w := 0
			for _, v := range input {
				if fn(v).Bool() {
					input[w] = v
					w++
				}
			}
			return NewSlice(input[:w])
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "owned list to filter in-place"},
				{Kind: "func", ParamName: "condition", ParamDesc: "filter condition func(any)->bool", Params: []*TypeDescriptor{{Kind: "any", ParamName: "item"}}, Return: &TypeDescriptor{Kind: "bool"}},
			},
			Return: FreshAlloc,
			Const: true,
			Forbidden: true,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "reverse_mut",
		Desc: "in-place reverse (optimizer-only)",
		Fn: func(a ...Scmer) Scmer {
			list := a[0].Slice()
			for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
				list[i], list[j] = list[j], list[i]
			}
			return NewSlice(list)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "owned list to reverse in-place"},
			},
			Return: FreshAlloc,
			Const: true,
			Forbidden: true,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "filter_assoc_mut",
		Desc: "in-place filter_assoc (optimizer-only)",
		Fn: func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "filter_assoc_mut"); fd == nil {
				w := 0
				for i := 0; i < len(slice); i += 2 {
					if fn(slice[i], slice[i+1]).Bool() {
						slice[w] = slice[i]
						slice[w+1] = slice[i+1]
						w += 2
					}
				}
				return NewSlice(slice[:w])
			} else {
				result := make([]Scmer, 0)
				fd.Iterate(func(k, v Scmer) bool {
					if fn(k, v).Bool() {
						result = append(result, k, v)
					}
					return true
				})
				return NewSlice(result)
			}
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "owned dictionary to filter in-place"},
				{Kind: "func", ParamName: "condition", ParamDesc: "filter function func(key, value)->bool", Params: []*TypeDescriptor{{Kind: "string", ParamName: "key"}, {Kind: "any", ParamName: "value"}}, Return: &TypeDescriptor{Kind: "bool"}},
			},
			Return: FreshAlloc,
			Const: true,
			Forbidden: true,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "extract_assoc_mut",
		Desc: "in-place extract_assoc (optimizer-only)",
		Fn: func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "extract_assoc_mut"); fd == nil {
				w := 0
				for i := 0; i < len(slice); i += 2 {
					slice[w] = fn(slice[i], slice[i+1])
					w++
				}
				return NewSlice(slice[:w])
			} else {
				result := make([]Scmer, 0, len(fd.Pairs)/2)
				fd.Iterate(func(k, v Scmer) bool {
					result = append(result, fn(k, v))
					return true
				})
				return NewSlice(result)
			}
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "owned dictionary to extract from in-place"},
				{Kind: "func", ParamName: "map", ParamDesc: "func(key, value)->any that extracts each element", Params: []*TypeDescriptor{{Kind: "string", ParamName: "key"}, {Kind: "any", ParamName: "value"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: FreshAlloc,
			Const: true,
			Forbidden: true,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "set_assoc_mut",
		Desc: "in-place set_assoc (optimizer-only, mutates input directly)",
		Fn: func(a ...Scmer) Scmer {
			var mergeFn func(Scmer, Scmer) Scmer
			if len(a) > 3 {
				mfn := OptimizeProcToSerialFunction(a[3])
				mergeFn = func(oldV, newV Scmer) Scmer { return mfn(oldV, newV) }
			}
			slice, fd := asAssoc(a[0], "set_assoc_mut")
			if fd == nil {
				list := slice
				for i := 0; i < len(list); i += 2 {
					if Equal(list[i], a[1]) {
						if mergeFn != nil {
							list[i+1] = mergeFn(list[i+1], a[2])
						} else {
							list[i+1] = a[2]
						}
						return NewSlice(list)
					}
				}
				list = append(list, a[1], a[2])
				if len(list) >= 10 {
					fd := NewFastDictValue(len(list)/2 + 4)
					for i := 0; i < len(list); i += 2 {
						fd.Set(list[i], list[i+1], nil)
					}
					return NewFastDict(fd)
				}
				return NewSlice(list)
			} else {
				fd.Set(a[1], a[2], mergeFn)
				return NewFastDict(fd)
			}
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "owned dictionary to mutate"},
				{Kind: "string", ParamName: "key", ParamDesc: "key to set"},
				{Kind: "any", ParamName: "value", ParamDesc: "new value"},
				{Kind: "func", ParamName: "merge", ParamDesc: "(optional) merge function", Optional: true, Params: []*TypeDescriptor{{Kind: "any", ParamName: "old"}, {Kind: "any", ParamName: "new"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: FreshAlloc,
			Const: true,
			Forbidden: true,
		},
	})

	// Tier 3: append/grow

	Declare(&Globalenv, &Declaration{
		Name: "append_mut",
		Desc: "in-place append (optimizer-only)",
		Fn: func(a ...Scmer) Scmer {
			base := asSlice(a[0], "append_mut")
			base = append(base, a[1:]...)
			return NewSlice(base)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "owned base list"},
				{Kind: "any", ParamName: "item...", ParamDesc: "items to add", Variadic: true},
			},
			Return: FreshAlloc,
			Const: true,
			Forbidden: true,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "append_unique_mut",
		Desc: "in-place append_unique (optimizer-only)",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "append_unique_mut")
			for _, el := range a[1:] {
				for _, el2 := range list {
					if Equal(el, el2) {
						goto skipItem
					}
				}
				list = append(list, el)
			skipItem:
			}
			return NewSlice(list)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "owned base list"},
				{Kind: "any", ParamName: "item...", ParamDesc: "items to add", Variadic: true},
			},
			Return: FreshAlloc,
			Const: true,
			Forbidden: true,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "merge_assoc_mut",
		Desc: "in-place merge_assoc (optimizer-only)",
		Fn: func(a ...Scmer) Scmer {
			setAssoc := OptimizeProcToSerialFunction(Globalenv.Vars["set_assoc_mut"])
			dst := a[0]
			if slice, fd := asAssoc(a[1], "merge_assoc_mut"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					if len(a) > 2 {
						dst = setAssoc(dst, slice[i], slice[i+1], a[2])
					} else {
						dst = setAssoc(dst, slice[i], slice[i+1])
					}
				}
			} else {
				if len(a) > 2 {
					fd.Iterate(func(k, v Scmer) bool { dst = setAssoc(dst, k, v, a[2]); return true })
				} else {
					fd.Iterate(func(k, v Scmer) bool { dst = setAssoc(dst, k, v); return true })
				}
			}
			return dst
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict1", ParamDesc: "owned first dictionary"},
				{Kind: "list", ParamName: "dict2", ParamDesc: "dictionary with new values"},
				{Kind: "func", ParamName: "merge", ParamDesc: "(optional) merge function", Optional: true, Params: []*TypeDescriptor{{Kind: "any", ParamName: "old"}, {Kind: "any", ParamName: "new"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: FreshAlloc,
			Const: true,
			Forbidden: true,
		},
	})
}

// optimizeMerge rewrites merge calls to avoid intermediate allocations:
//   (merge (map list fn)) → (flatmap list fn)           — single-arg merge over map
//   (merge (extract_assoc dict fn)) → (flatmap_assoc...) — not yet, but same idea
//   (merge a b (map list fn)) → flatten map result inline
func optimizeMerge(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	// First: apply default optimization to all args
	result, td := oc.ApplyDefaultOptimization(v, useResult)
	if rSlice, ok := scmerSlice(result); ok && len(rSlice) == 2 {
		// (merge X) where X is a single argument
		arg := rSlice[1]
		if inner, ok2 := scmerSlice(arg); ok2 && len(inner) == 3 {
			// Check if arg is (map list fn) or (map_mut list fn)
			if scmerIsSymbol(inner[0], "map") || scmerIsSymbol(inner[0], "map_mut") {
				// Rewrite to (flatmap list fn) — flatmap does map+flatten in one pass
				flatmapCall := []Scmer{NewSymbol("flatmap"), inner[1], inner[2]}
				return NewSlice(flatmapCall), FreshAlloc
			}
		}
	}
	return result, td
}

// optimizeCons rewrites cons when the tail is a freshly allocated list:
//   (cons head (map list fn)) → (cons head (map_mut list fn))  — already handled by _mut
//   (cons head (list a b c))  → (list head a b c)              — avoid double allocation
func optimizeCons(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	result, td := oc.ApplyDefaultOptimization(v, useResult)
	if rSlice, ok := scmerSlice(result); ok && len(rSlice) == 3 {
		tail := rSlice[2]
		if inner, ok2 := scmerSlice(tail); ok2 && len(inner) >= 1 {
			// (cons head (list a b c)) → (list head a b c)
			if scmerIsSymbol(inner[0], "list") {
				merged := make([]Scmer, 0, len(inner)+1)
				merged = append(merged, NewSymbol("list"))
				merged = append(merged, rSlice[1]) // head
				merged = append(merged, inner[1:]...) // tail items
				return NewSlice(merged), FreshAlloc
			}
		}
	}
	return result, td
}
