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
	case SourceInfo:
		return String(v.value)
	case NthLocalVar:
		return fmt.Sprintf("(var %d)", v)
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
	case string:
		return v // this is not valid scm code! (but we need it to convert strings)
	case nil:
		return "nil"
	default:
		return fmt.Sprint(v)
	}
}
func SerializeToString(v Scmer, glob *Env) string {
	var b bytes.Buffer
	SerializeEx(&b, v, glob, glob, nil)
	return b.String()
}
func Serialize(b *bytes.Buffer, v Scmer, glob *Env) {
	SerializeEx(b, v, glob, glob, nil)
}
func SerializeEx(b *bytes.Buffer, v Scmer, en *Env, glob *Env, p *Proc) {
	if en != glob {
		b.WriteString("(begin ")
		for k, v := range en.Vars {
			// if Symbol is defined in a lambda, print the real value
			// filter out redefinition of global functions
			if fmt.Sprint(glob.Vars[Symbol(k)]) != fmt.Sprint(v) {
				b.WriteString("(define ")
				b.WriteString(string(k)) // what if k contains spaces?? can it?
				b.WriteString(" ")
				SerializeEx(b, v, en.Outer, glob, p)
				b.WriteString(") ")
			}
		}
		SerializeEx(b, v, en.Outer, glob, p)
		b.WriteString(")")
		return
	}
	switch v := v.(type) {
	case SourceInfo:
		SerializeEx(b, v.value, en, glob, p)
	case []Scmer:
		if len(v) == 2 && v[0] == Symbol("outer") {
			b.WriteString("(outer ")
			SerializeEx(b, v[1], en, glob, nil)
			b.WriteByte(')')
		} else {
			if len(v) > 0 && (v[0] == Symbol("list") || fmt.Sprint(v[0]) == fmt.Sprint(List)) {
				b.WriteByte('\'')
				v = v[1:]
			}
			b.WriteByte('(')
			for i, x := range v {
				if i != 0 {
					b.WriteByte(' ')
				}
				SerializeEx(b, x, en, glob, p)
			}
			b.WriteByte(')')
		}
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
		SerializeEx(b, v.Syntax, glob, glob, p)
		b.WriteByte(' ')
		SerializeEx(b, v.Generator, en, glob, p)
		b.WriteByte(')')
	// TODO: further parsers
	case Proc:
		b.WriteString("(lambda ")
		if (v.NumVars > 0 && v.Params == nil) {
			// TODO: deeoptimize
		}
		SerializeEx(b, v.Params, glob, glob, nil)
		b.WriteByte(' ')
		SerializeEx(b, v.Body, v.En, glob, &v)
		if v.NumVars > 0 {
			b.WriteByte(' ')
			b.WriteString(fmt.Sprint(v.NumVars))
		}
		b.WriteByte(')')
	case NthLocalVar:
		if p != nil && p.NumVars >= int(v) && p.Params != nil {
			if l, ok := p.Params.([]Scmer); ok {
				s, _ := l[v].(Symbol)
				b.WriteString(string(s))
			} else {
				s, _ := p.Params.(Symbol)
				b.WriteString(string(s))
			}
		} else {
			b.WriteString("(var ")
			b.WriteString(fmt.Sprint(v))
			b.WriteByte(')')
		}
	case Symbol:
		// print as Symbol (because we already used a begin-block for defining our env)
		if strings.Contains(string(v), " ") || strings.Contains(string(v), "(") || strings.Contains(string(v), ")") || strings.Contains(string(v), "\"") {
			b.WriteString("(unquote \"")
			b.WriteString(strings.Replace(string(v), "\"", "\\\"", -1))
			b.WriteString("\")")
		} else {
			b.WriteString(string(v))
		}
	case string:
		b.WriteByte('"')
		b.WriteString(strings.NewReplacer("\"", "\\\"", "\\", "\\\\", "\r", "\\r", "\n", "\\n").Replace(v))
		b.WriteByte('"')
	case nil:
		b.WriteString("nil")
	default:
		b.WriteString(fmt.Sprint(v))
	}
}
