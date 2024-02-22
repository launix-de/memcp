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
	Returns string // any | string | number | bool | func | list | symbol | nil
	Fn func(...Scmer) Scmer
}

type DeclarationParameter struct {
	Name string
	Type string // any | string | number | bool | func | list | symbol | nil
	Desc string
}

var declaration_titles []string
var declarations map[string]*Declaration = make(map[string]*Declaration)
var declarations_hash map[string]*Declaration = make(map[string]*Declaration)

func DeclareTitle(title string) {
	declaration_titles = append(declaration_titles, "#" + title)
}

func Declare(env *Env, def *Declaration) {
	declaration_titles = append(declaration_titles, def.Name)
	declarations[def.Name] = def
	if def.Fn != nil {
		declarations_hash[fmt.Sprintf("%p", def.Fn)] = def
		env.Vars[Symbol(def.Name)] = def.Fn
	}
}

func types_match(given string, required string) bool {
	if given == "any" {
		return true // be graceful, we can't check it
	}
	if required == "any" {
		return true // this is always allowed
	}
	required_ := strings.Split(required, "|")
	given_ := strings.Split(given, "|")
	for _, r := range required_ {
		for _, g := range given_ {
			// TODO: in case of func: compare signatures??
			// TODO: list(subtype)
			if r == g {
				return true // if any given fits any required, the value is allowed
			}
		}
	}
	return false // not a single match
}

// panics if the code is bad (returns possible datatype, at least "any")
func Validate(val Scmer) string {
	var source_info SourceInfo
	switch v := val.(type) {
		case SourceInfo:
			source_info = v
			val = v.value
	}
	switch v := val.(type) {
		case nil:
			return "nil"
		case string:
			return "string"
		case float64:
			return "number"
		case bool:
			return "bool"
		case Proc:
			return "func"
		case func(...Scmer) Scmer:
			return "func"
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
						panic(source_info.String() + ": function " + def.Name + " expects at least " + fmt.Sprintf("%d", def.MinParameter) + " parameters")
					}
					if len(v)-1 > def.MaxParameter {
						panic(source_info.String() + ": function " + def.Name + " expects at most " + fmt.Sprintf("%d", def.MaxParameter) + " parameters")
					}
				}
				// validate params (TODO: exceptions like match??)
				for i := 1; i < len(v); i++ {
					if i != 1 || (v[0] != Symbol("lambda") && v[0] != Symbol("parser")) {
						typ := Validate(v[i])
						if def != nil {
							j := i-1 // parameter help
							if i-1 >= len(def.Params) {
								j = len(def.Params) - 1
							}
							// check parameter type
							// TODO: both types could also be lists separated by |
							// TODO: signature of lambda types??
							if !types_match(typ, def.Params[j].Type) {
								panic(fmt.Sprintf("%s: function %s expects parameter %d to be %s, but found value of type %s", source_info.String(), def.Name, i, def.Params[j].Type, typ))
							}
						}
					}
				}
				if def != nil {
					return def.Returns
				}
			}
	}
	return "any"
}

// do preprocessing and optimization
func Optimize(val Scmer, env *Env) Scmer {
	// TODO: strip source code information (source, line, col)
	// TODO: static code analysis like escape analysis + replace memory-safe functions with in-place memory manipulating versions (e.g. in set_assoc)
	return val
}

func Help(fn Scmer) {
	if fn == nil {
		fmt.Println("Available scm functions:")
		for _, title := range declaration_titles {
			if title[0] == '#' {
				fmt.Println("")
				fmt.Println("-- " + title[1:] + " --")
			} else {
				fmt.Println("  " + title + ": " + strings.Split(declarations[title].Desc, "\n")[0])
			}
		}
		fmt.Println("")
		fmt.Println("get further information by typing (help \"functionname\") to get more info")
	} else {
		var def *Declaration
		if s, ok := fn.(string); ok {
			if def2, ok := declarations[s]; ok {
				def = def2
			} else {
				panic("function not found: " + s)
			}
		} else if f, ok := fn.(func(...Scmer) Scmer); ok {
			if def2, ok := declarations_hash[fmt.Sprintf("%p", f)]; ok {
				def = def2
			}
		}

		if def != nil {
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
			panic("function not found: " + fmt.Sprint(fn))
		}
	}
}
