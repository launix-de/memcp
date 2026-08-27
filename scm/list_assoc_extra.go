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

func init_list_assoc_extra() {
	Declare(&Globalenv, &Declaration{
		Name: "mapkey_assoc",

		Fn: func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			setAssoc := OptimizeProcToSerialFunction(Globalenv.Vars["set_assoc"])
			result := NewSlice(nil)
			if slice, fd := asAssoc(a[0], "mapkey_assoc"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					result = setAssoc(result, fn(slice[i], slice[i+1]), slice[i+1])
				}
			} else {
				fd.Iterate(func(k, v Scmer) bool {
					result = setAssoc(result, fn(k, v), v)
					return true
				})
			}
			return result
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a mapped dictionary according to a map function\nValues stay the same but keys are mapped.",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "dict", Description: "dictionary whose keys have to be mapped", NoEscape: true},
				{Kind: "func", Label: "map", Description: "map function func(key, value)->key where the first parameter is the old key, the second is the value. It must return the new key.", Params: []*TypeDescriptor{{Kind: "string", Label: "key"}, {Kind: "any", Label: "value"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: FirstParameterMutable("mapkey_assoc_mut"),
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "mapkey_assoc_mut",

		Fn: func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			setAssoc := OptimizeProcToSerialFunction(Globalenv.Vars["set_assoc_mut"])
			slice, fd := asAssoc(a[0], "mapkey_assoc_mut")
			if fd == nil {
				orig := append([]Scmer{}, slice...)
				result := NewSlice(slice[:0])
				for i := 0; i < len(orig); i += 2 {
					result = setAssoc(result, fn(orig[i], orig[i+1]), orig[i+1])
				}
				return result
			}
			result := NewSlice(nil)
			fd.Iterate(func(k, v Scmer) bool {
				result = setAssoc(result, fn(k, v), v)
				return true
			})
			return result
		},
		Type: &TypeDescriptor{Kind: "func", Description: "optimizer-only key remap for dictionaries",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "dict", Description: "owned dictionary whose keys have to be remapped"},
				{Kind: "func", Label: "map", Description: "map function func(key, value)->key", Params: []*TypeDescriptor{{Kind: "string", Label: "key"}, {Kind: "any", Label: "value"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,
		},
	})
}
