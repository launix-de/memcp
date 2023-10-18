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

package scm

import (
	"fmt"
	"bytes"
	"strings"
)

func String(v Scmer) string {
	switch v := v.(type) {
	case []Scmer:
		l := make([]string, len(v))
		for i, x := range v {
			l[i] = String(x)
		}
		return "(" + strings.Join(l, " ") + ")"
	case Proc:
		return "[func]"
	case func(...Scmer) Scmer:
		return "[native func]"
	default:
		return fmt.Sprint(v)
	}
}
func SerializeToString(v Scmer, en *Env, glob *Env) string {
	var b bytes.Buffer
	Serialize(&b, v, en, glob)
	return b.String()
}
func Serialize(b *bytes.Buffer, v Scmer, en *Env, glob *Env) {
	if en != glob {
		b.WriteString("(begin ")
		for k, v := range en.Vars {
			// if Symbol is defined in a lambda, print the real value
			// filter out redefinition of global functions
			if fmt.Sprint(glob.Vars[Symbol(k)]) != fmt.Sprint(v) {
				b.WriteString("(define ")
				b.WriteString(string(k)) // what if k contains spaces?? can it?
				b.WriteString(" ")
				Serialize(b, v, en.Outer, glob)
				b.WriteString(") ")
			}
		}
		Serialize(b, v, en.Outer, glob)
		b.WriteString(")")
		return
	}
	switch v := v.(type) {
	case []Scmer:
		if len(v) > 0 && v[0] == Symbol("list") {
			b.WriteByte('\'')
			v = v[1:]
		}
		b.WriteByte('(')
		for i, x := range v {
			if i != 0 {
				b.WriteByte(' ')
			}
			Serialize(b, x, en, glob)
		}
		b.WriteByte(')')
	case func(...Scmer) Scmer:
		// native func serialization is the hardest; reverse the env!
		// when later functional JIT is done, this must also handle deoptimization
		en2 := en
		for en2 != nil {
			for k, v2 := range en2.Vars {
				// compare function pointers (hacky but golang dosent give another opt)
				if fmt.Sprint(v) == fmt.Sprint(v2) {
					// found the right global function
					b.WriteString(fmt.Sprint(k)) // print out variable name
					return
				}
			}
			en2 = en2.Outer
		}
		b.WriteString("[unserializable native func]")
	case *ScmParser:
		b.WriteString("(parser ")
		Serialize(b, v.Syntax, glob, glob)
		b.WriteByte(' ')
		Serialize(b, v.Generator, en, glob)
		b.WriteByte(')')
	case Proc:
		b.WriteString("(lambda ")
		Serialize(b, v.Params, glob, glob)
		b.WriteByte(' ')
		Serialize(b, v.Body, v.En, glob)
		b.WriteByte(')')
	case Symbol:
		// print as Symbol (because we already used a begin-block for defining our env)
		b.WriteString(fmt.Sprint(v))
	case string:
		b.WriteByte('"')
		b.WriteString(strings.NewReplacer("\"", "\\\"", "\\", "\\\\", "\r", "\\r", "\n", "\\n").Replace(v))
		b.WriteByte('"')
	default:
		b.WriteString(fmt.Sprint(v))
	}
}
