/*
Copyright (C) 2023  Carl-Philip Hänsch

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
import "strings"

type Declaration struct {
	Name string
	Desc string
	MinParameter int
	MaxParameter int
	Params []DeclarationParameter
	Returns string // any | string | number | bool | func | list | symbol
	Fn func(...Scmer) Scmer
}

type DeclarationParameter struct {
	Name string
	Type string // any | string | number | bool | func | list | symbol
	Desc string
}

var declarations map[string]*Declaration = make(map[string]*Declaration)
var declarations_hash map[string]*Declaration = make(map[string]*Declaration)

func Declare(env *Env, def *Declaration) {
	declarations[def.Name] = def
	if def.Fn != nil {
		declarations_hash[fmt.Sprintf("%p", def.Fn)] = def
		env.Vars[Symbol(def.Name)] = def.Fn
	}
}

// panics if the code is bad (returns possible datatype, at least "any")
func Validate(source string, val Scmer) string {
	switch v := val.(type) {
		case string:
			return "string"
		case float64:
			return "number"
		case bool:
			return "bool"
		case []Scmer:
			if len(v) > 0 {
				// function with head
				var def *Declaration
				switch head := v[0].(type) {
					case Symbol:
						if def2, ok := declarations[string(head)]; ok {
							def = def2
						}
					case func(...Scmer) Scmer:
						if def2, ok := declarations[fmt.Sprintf("%p", head)]; ok {
							def = def2
						}
				}
				if def != nil {
					if len(v)-1 < def.MinParameter {
						panic(source + ": function " + def.Name + " expects at least " + fmt.Sprintf("%d", def.MinParameter) + " parameters")
					}
					if len(v)-1 > def.MaxParameter {
						panic(source + ": function " + def.Name + " expects at most " + fmt.Sprintf("%d", def.MaxParameter) + " parameters")
					}
				}
				// validate params (TODO: exceptions like match??)
				for i := 1; i < len(v); i++ {
					typ := Validate(source, v[i])
					if def != nil {
						j := i-1 // parameter help
						if i-1 >= len(def.Params) {
							j = len(def.Params) - 1
						}
						// check parameter type
						if typ != "any" && def.Params[j].Type != "any" && typ != def.Params[j].Type {
							panic(fmt.Sprintf("%s: function %s expects parameter %d to be %s, but found value of type %s", source, def.Name, i, typ, def.Params[j].Type))
						}
					}
				}
			}
	}
	return "any"
}

// do preprocessing and optimization
func Optimize(val Scmer, env *Env) Scmer {
	return val
}

func Help(fn string) {
	if fn == "" {
		fmt.Println("Available scm functions:")
		fmt.Println("")
		for fname, def := range declarations {
			fmt.Println("  " + fname + ": " + strings.Split(def.Desc, "\n")[0])
		}
		fmt.Println("")
		fmt.Println("get further information by typing (help \"functionname\") to get more info")
	} else {
		if def, ok := declarations[fn]; ok {
			fmt.Println("Help for: " + def.Name)
			fmt.Println("===")
			fmt.Println("")
			fmt.Println(def.Desc)
			fmt.Println("")
			fmt.Println("Allowed nø of parameters: ", def.MinParameter, "-", def.MaxParameter)
			fmt.Println("")
			for _, p := range def.Params {
				fmt.Println(" - " + p.Name + " (" + p.Type + "): " + p.Desc)
			}
			fmt.Println("")
		} else {
			panic("function not found: " + fn)
		}
	}
}
