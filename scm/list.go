/*
Copyright (C) 2023  Carl-Philip Hänsch
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
import "reflect"

func init_list() {
	// list functions
	DeclareTitle("Lists")

	Declare(&Globalenv, &Declaration{
		"count", "counts the number of elements in the list",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "base list"},
		}, "int",
		func(a ...Scmer) Scmer {
			// append a b ...: append item b to list a (construct list from item + tail)
			return float64(len(a[0].([]Scmer)))
		},
	})
	Declare(&Globalenv, &Declaration{
		"nth", "get the nth item of a list",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "base list"},
			DeclarationParameter{"index", "number", "index beginning from 0"},
		}, "any",
		func(a ...Scmer) Scmer {
			return a[0].([]Scmer)[ToInt(a[1])]
		},
	})
	Declare(&Globalenv, &Declaration{
		"append", "appends items to a list and return the extended list.\nThe original list stays unharmed.",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "base list"},
			DeclarationParameter{"item...", "any", "items to add"},
		}, "list",
		func(a ...Scmer) Scmer {
			// append a b ...: append item b to list a (construct list from item + tail)
			return append(a[0].([]Scmer), a[1:]...)
		},
	})
	Declare(&Globalenv, &Declaration{
		"append_unique", "appends items to a list but only if they are new.\nThe original list stays unharmed.",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "base list"},
			DeclarationParameter{"item...", "any", "items to add"},
		}, "list",
		func(a ...Scmer) Scmer {
			// append a b ...: append item b to list a (construct list from item + tail)
			list := a[0].([]Scmer)
			for _, el := range a {
				for _, el2 := range list {
					if reflect.DeepEqual(el, el2) {
						// ignore duplicates
						goto skipItem
					}
				}
				list = append(list, el)
				skipItem:
			}
			return list
		},
	})
	Declare(&Globalenv, &Declaration{
		"cons", "constructs a list from a head and a tail list",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"car", "any", "new head element"},
			DeclarationParameter{"cdr", "list", "tail that is appended after car"},
		}, "list",
		func(a ...Scmer) Scmer {
			// cons a b: prepend item a to list b (construct list from item + tail)
			switch car := a[0]; cdr := a[1].(type) {
			case []Scmer:
				return append([]Scmer{car}, cdr...)
			default:
				return []Scmer{car, cdr}
			}
		},
	})
	Declare(&Globalenv, &Declaration{
		"car", "extracts the head of a list",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list"},
		}, "any",
		func(a ...Scmer) Scmer {
			// head of tuple
			return a[0].([]Scmer)[0]
		},
	})
	Declare(&Globalenv, &Declaration{
		"cdr", "extracts the tail of a list\nThe tail of a list is a list with all items except the head.",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list"},
		}, "any",
		func(a ...Scmer) Scmer {
			// rest of tuple
			return a[0].([]Scmer)[1:]
		},
	})
	Declare(&Globalenv, &Declaration{
		"zip", "swaps the dimension of a list of lists. If one parameter is given, it is a list of lists that is flattened. If multiple parameters are given, they are treated as the components that will be zipped into the sub list",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list of lists of items"},
		}, "list",
		func (a ...Scmer) Scmer {
			list := a
			if len(a) == 1 {
				// one parameter: interpret as list of lists
				var ok bool
				list, ok = a[0].([]Scmer)
				if !ok {
					panic("invalid input for merge: " + fmt.Sprint(a))
				}
			}
			// merge arrays into one
			size := len(list[0].([]Scmer))
			result := make([]Scmer, size)
			for i := range result {
				subresult := make([]Scmer, len(list))
				for j, v := range list {
					subresult[j] = v.([]Scmer)[i]
				}
				result[i] = subresult
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"merge", "flattens a list of lists into a list containing all the subitems. If one parameter is given, it is a list of lists that is flattened. If multiple parameters are given, they are treated as lists that will be merged into one",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list of lists of items"},
		}, "list",
		func (a ...Scmer) Scmer {
			list := a
			if len(a) == 1 {
				// one parameter: interpret as list of lists
				var ok bool
				list, ok = a[0].([]Scmer)
				if !ok {
					panic("invalid input for merge: " + fmt.Sprint(a))
				}
			}
			// merge arrays into one
			size := 0
			for _, v := range list {
				size = size + len(v.([]Scmer))
			}
			result := make([]Scmer, size)
			pos := 0
			for _, v := range list {
				inner := v.([]Scmer)
				copy(result[pos:pos+len(inner)], inner)
				pos = pos + len(inner)
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"merge_unique", "flattens a list of lists into a list containing all the subitems. Duplicates are filtered out.",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list of lists of items"},
		}, "list",
		func (a ...Scmer) Scmer {
			list := a
			if len(a) == 1 {
				// one parameter: interpret as list of lists
				list = a[0].([]Scmer)
			}
			// merge arrays into one
			size := 0
			for _, v := range list {
				size = size + len(v.([]Scmer))
			}
			result := make([]Scmer, 0, size)
			for _, v := range list {
				inner := v.([]Scmer)
				for _, el := range inner {
					for _, el2 := range result {
						if reflect.DeepEqual(el, el2) {
							goto skipNext
						}
					}
					result = append(result, el)
					skipNext:
				}
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"has?", "checks if a list has a certain item (equal?)",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"haystack", "list", "list to search in"},
			DeclarationParameter{"needle", "any", "item to search for"},
		}, "bool",
		func(a ...Scmer) Scmer {
			// arr, element
			list := a[0].([]Scmer)
			for _, v := range list {
				if reflect.DeepEqual(a[1], v) {
					return true
				}
			}
			return false
		},
	})
	Declare(&Globalenv, &Declaration{
		"filter", "returns a list that only contains elements that pass the filter function",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list that has to be filtered"},
			DeclarationParameter{"condition", "func", "filter condition func(any)->bool"},
		}, "list",
		func(a ...Scmer) Scmer {
			result := make([]Scmer, 0)
			fn := OptimizeProcToSerialFunction(a[1])
			for _, v := range a[0].([]Scmer) {
				if ToBool(fn(v)) {
					result = append(result, v)
				}
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"map", "returns a list that contains the results of a map function that is applied to the list",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list that has to be mapped"},
			DeclarationParameter{"map", "func", "map function func(any)->any that is applied to each item"},
		}, "list",
		func(a ...Scmer) Scmer {
			list, _ := a[0].([]Scmer)
			result := make([]Scmer, len(list))
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				result[i] = fn(v)
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"mapIndex", "returns a list that contains the results of a map function that is applied to the list",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list that has to be mapped"},
			DeclarationParameter{"map", "func", "map function func(i, any)->any that is applied to each item"},
		}, "list",
		func(a ...Scmer) Scmer {
			list, _ := a[0].([]Scmer)
			result := make([]Scmer, len(list))
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				result[i] = fn(i, v)
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"reduce", "returns a list that contains the result of a map function",
		2, 3,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list that has to be reduced"},
			DeclarationParameter{"reduce", "func", "reduce function func(any any)->any where the first parameter is the accumulator, the second is a list item"},
			DeclarationParameter{"neutral", "any", "(optional) initial value of the accumulator, defaults to nil"},
		}, "any",
		func(a ...Scmer) Scmer {
			// arr, reducefn(a, b), [neutral]
			list, _ := a[0].([]Scmer)
			fn := OptimizeProcToSerialFunction(a[1])
			var result Scmer = nil
			i := 0
			if len(a) > 2 {
				result = a[2]
			} else {
				if len(list) > 0 {
					result = list[0]
					i = i + 1
				}
			}
			for i < len(list) {
				result = fn(result, list[i])
				i = i + 1
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"produce", "returns a list that contains produced items - it works like for(state = startstate, condition(state), state = iterator(state)) {yield state}",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"startstate", "any", "start state to begin with"},
			DeclarationParameter{"condition", "func", "func that returns true whether the state will be inserted into the result or the loop is stopped"},
			DeclarationParameter{"iterator", "func", "func that produces the next state"},
		}, "list",
		func(a ...Scmer) Scmer {
			// arr, reducefn(a, b), [neutral]
			result := make([]Scmer, 0)
			state := a[0]
			condition := OptimizeProcToSerialFunction(a[1])
			iterator := OptimizeProcToSerialFunction(a[2])
			for ToBool(condition(state)) {
				result = append(result, state)
				state = iterator(state)
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"produceN", "returns a list with numbers from 0..n-1",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"n", "number", "number of elements to produce"},
		}, "list",
		func(a ...Scmer) Scmer {
			// arr, reducefn(a, b), [neutral]
			n := ToInt(a[0])
			result := make([]Scmer, n)
			for i := 0; i < n; i++ {
				result[i] = float64(i) // TODO: leave to integer once a flexible type system is implemented
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"list?", "checks if a value is a list",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value to check"},
		}, "bool",
		func(a ...Scmer) Scmer {
			// arr
			_, ok := a[0].([]Scmer)
			return ok
		},
	})
	Declare(&Globalenv, &Declaration{
		"contains?", "checks if a value is in a list; uses the equal?? operator",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list to check"},
			DeclarationParameter{"value", "any", "value to check"},
		}, "bool",
		func(a ...Scmer) Scmer {
			// arr
			arr, _ := a[0].([]Scmer)
			for _, v := range arr {
				if Equal(v, a[1]) == true {
					return true
				}
			}
			return false
		},
	})

	// dictionary functions
	DeclareTitle("Associative Lists / Dictionaries")

	Declare(&Globalenv, &Declaration{
		"filter_assoc", "returns a filtered dictionary according to a filter function",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"dict", "list", "dictionary that has to be filtered"},
			DeclarationParameter{"condition", "func", "filter function func(string any)->bool where the first parameter is the key, the second is the value"},
		}, "list",
		func(a ...Scmer) Scmer {
			// list, fn(key, value)
			list := a[0].([]Scmer)
			result := make([]Scmer, 0)
			fn := OptimizeProcToSerialFunction(a[1])
			for i := 0; i < len(list); i += 2 {
				if ToBool(fn(list[i], list[i+1])) {
					result = append(result, list[i], list[i+1])
				}
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"map_assoc", "returns a mapped dictionary according to a map function\nKeys will stay the same but values are mapped.",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"dict", "list", "dictionary that has to be mapped"},
			DeclarationParameter{"map", "func", "map function func(string any)->any where the first parameter is the key, the second is the value. It must return the new value."},
		}, "list",
		func(a ...Scmer) Scmer {
			// apply fn(key value) to each assoc item and return mapped dict
			list := a[0].([]Scmer)
			result := make([]Scmer, len(list))
			fn := OptimizeProcToSerialFunction(a[1])
			var k Scmer
			for i, v := range list {
				if i % 2 == 0 {
					// key -> remain
					result[i] = v
					k = v
				} else {
					// value -> map fn(key, value)
					result[i] = fn(k, v)
				}
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"reduce_assoc", "reduces a dictionary according to a reduce function",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"dict", "list", "dictionary that has to be reduced"},
			DeclarationParameter{"reduce", "func", "reduce function func(any string any)->any where the first parameter is the accumulator, second is key, third is value. It must return the new accumulator."},
			DeclarationParameter{"neutral", "any", "initial value for the accumulator"},
		}, "any",
		func(a ...Scmer) Scmer {
			// dict, reducefn(a, key, value), neutral
			list := a[0].([]Scmer)
			result := a[2]
			reduce := OptimizeProcToSerialFunction(a[1])
			for i := 0; i < len(list); i += 2 {
				result = reduce(result, list[i], list[i+1])
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"has_assoc?", "checks if a dictionary has a key present",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"dict", "list", "dictionary that has to be checked"},
			DeclarationParameter{"key", "string", "key to test"},
		}, "bool",
		func(a ...Scmer) Scmer {
			// dict, element
			list := a[0].([]Scmer)
			for i := 0; i < len(list); i += 2 {
				if reflect.DeepEqual(list[i], a[1]) {
					return true
				}
			}
			return false
		},
	})
	Declare(&Globalenv, &Declaration{
		"extract_assoc", "applies a function (key value) on the dictionary and returns the results as a flat list",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"dict", "list", "dictionary that has to be checked"},
			DeclarationParameter{"map", "func", "func(string any)->any that flattens down each element"},
		}, "list",
		func(a ...Scmer) Scmer {
			// apply fn(key value) to each assoc item and return results as array
			list := a[0].([]Scmer)
			result := make([]Scmer, len(list) / 2)
			fn := OptimizeProcToSerialFunction(a[1])
			var k Scmer
			for i, v := range list {
				if i % 2 == 0 {
					// key -> remain
					k = v
				} else {
					// value -> map fn(key, value)
					result[i / 2] = fn(k, v)
				}
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"set_assoc", "returns a dictionary where a single value has been changed.\nThis function may destroy the input value for the sake of performance. You must not use the input value again.",
		3, 4,
		[]DeclarationParameter{
			DeclarationParameter{"dict", "list", "input dictionary that has to be changed. You must not use this value again."},
			DeclarationParameter{"key", "string", "key that has to be set"},
			DeclarationParameter{"value", "any", "new value to set"},
			DeclarationParameter{"merge", "func", "(optional) func(any any)->any that is called when a value is overwritten. The first parameter is the old value, the second is the new value. It must return the merged value that shall be pysically stored in the new dictionary."},
		}, "list",
		func(a ...Scmer) Scmer {
			list := a[0].([]Scmer)
			var fn func(a ...Scmer) Scmer
			if len(a) > 3 {
				fn = OptimizeProcToSerialFunction(a[3])
			}
			for i := 0; i < len(list); i += 2 {
				if reflect.DeepEqual(list[i], a[1]) {
					// overwrite
					if len(a) > 3 {
						// overwrite with merge function
						list[i + 1] = fn(list[i + 1], a[2])
					} else {
						// overwrite naive
						list[i + 1] = a[2]
					}
					return list // return changed list (this violates immutability for performance)
				}
			}
			// else: append
			return append(list, a[1], a[2])
		},
	})
	Declare(&Globalenv, &Declaration{
		"merge_assoc", "returns a dictionary where all keys from dict1 and all keys from dict2 are present.\nIf a key is present in both inputs, the second one will be dominant so the first value will be overwritten unless you provide a merge function",
		2, 3,
		[]DeclarationParameter{
			DeclarationParameter{"dict1", "list", "first input dictionary that has to be changed. You must not use this value again."},
			DeclarationParameter{"dict2", "list", "input dictionary that contains the new values that have to be added"},
			DeclarationParameter{"merge", "func", "(optional) func(any any)->any that is called when a value is overwritten. The first parameter is the old value, the second is the new value from dict2. It must return the merged value that shall be pysically stored in the new dictionary."},
		}, "list",
		func(a ...Scmer) Scmer {
			// naive implementation, bad performance
			set_assoc := Globalenv.Vars["set_assoc"].(func(...Scmer) Scmer)
			list := a[0]
			dict := a[1].([]Scmer)
			if len(a) > 2 {
				for i := 0; i < len(dict); i += 2 {
					list = set_assoc(list, dict[i], dict[i+1], a[2])
				}
			} else {
				for i := 0; i < len(dict); i += 2 {
					list = set_assoc(list, dict[i], dict[i+1])
				}
			}
			return list
		},
	})
}
