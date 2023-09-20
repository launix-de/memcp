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
	Fn func(...Scmer) Scmer
}

type DeclarationParameter struct {
	Name string
	Type string // any | string | number | func | list | symbol
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

// TODO: func Validate(val Scmer)
// TODO: func Optimize(val Scmer)
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
