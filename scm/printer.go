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
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
)

func String(v Scmer) string {
	switch auxTag(v.aux) {
	case tagNil:
		return "nil"
	case tagBool, tagInt, tagFloat:
		return v.String()
	case tagString:
		return v.String()
	case tagSymbol:
		return v.String()
	case tagSlice:
		slice := v.Slice()
		l := make([]string, len(slice))
		for i, x := range slice {
			l[i] = String(x)
		}
		return "(" + strings.Join(l, " ") + ")"
	case tagVector:
		vec := v.Vector()
		parts := make([]string, len(vec))
		for i, x := range vec {
			parts[i] = fmt.Sprint(x)
		}
		return "#(" + strings.Join(parts, " ") + ")"
	case tagFunc:
		return "[native func]"
	case tagAny:
		switch val := v.Any().(type) {
		case SourceInfo:
			return String(val.value)
		case NthLocalVar:
			return fmt.Sprintf("(var %d)", val)
		case *FastDict:
			l := make([]string, len(val.Pairs))
			for i, x := range val.Pairs {
				l[i] = String(x)
			}
			return "(" + strings.Join(l, " ") + ")"
		case Proc:
			return fmt.Sprintf("[func %s]", String(val.Body))
		case *Proc:
			return fmt.Sprintf("[func %s]", String(val.Body))
		case func(...Scmer) Scmer:
			return "[native func]"
		case func(*Env, ...Scmer) Scmer:
			return "[native func]"
		case io.Reader:
			var sb strings.Builder
			_, _ = io.Copy(&sb, val)
			return sb.String()
		case string:
			return val
		default:
			return fmt.Sprint(val)
		}
	default:
		return fmt.Sprintf("<scmer %d>", auxTag(v.aux))
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
			if gv, ok := glob.Vars[k]; !ok || !Equal(gv, v) {
				b.WriteString("(define ")
				b.WriteString(string(k))
				b.WriteString(" ")
				SerializeEx(b, v, en.Outer, glob, p)
				b.WriteString(") ")
			}
		}
		SerializeEx(b, v, en.Outer, glob, p)
		b.WriteString(")")
		return
	}
	switch auxTag(v.aux) {
	case tagNil:
		b.WriteString("nil")
	case tagBool:
		if v.Bool() {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case tagInt, tagFloat:
		b.WriteString(v.String())
	case tagString:
		b.WriteByte('"')
		b.WriteString(strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\r", "\\r", "\n", "\\n").Replace(v.String()))
		b.WriteByte('"')
	case tagSymbol:
		sym := v.String()
		if strings.ContainsAny(sym, " \"()") {
			b.WriteString("(unquote \"")
			b.WriteString(strings.ReplaceAll(sym, "\"", "\\\""))
			b.WriteString("\")")
		} else {
			b.WriteString(sym)
		}
	case tagSlice:
		slice := v.Slice()
		if len(slice) == 2 && slice[0].IsSymbol() && slice[0].String() == "outer" {
			b.WriteString("(outer ")
			SerializeEx(b, slice[1], en, glob, nil)
			b.WriteByte(')')
			return
		}
		if len(slice) > 0 && slice[0].IsSymbol() && slice[0].String() == "list" {
			b.WriteByte('\'')
			slice = slice[1:]
		}
		b.WriteByte('(')
		for i, x := range slice {
			if i != 0 {
				b.WriteByte(' ')
			}
			SerializeEx(b, x, en, glob, p)
		}
		b.WriteByte(')')
	case tagVector:
		vec := v.Vector()
		b.WriteString("#(")
		for i, x := range vec {
			if i != 0 {
				b.WriteByte(' ')
			}
			b.WriteString(fmt.Sprint(x))
		}
		b.WriteByte(')')
	case tagFunc:
		serializeNativeFunc(b, v.Func(), en)
	case tagAny:
		switch val := v.Any().(type) {
		case SourceInfo:
			SerializeEx(b, val.value, en, glob, p)
		case NthLocalVar:
			if p != nil && p.NumVars >= int(val) && auxTag(p.Params.aux) == tagSlice {
				params := p.Params.Slice()
				if int(val) < len(params) && params[val].IsSymbol() {
					b.WriteString(params[val].String())
					return
				}
			}
			b.WriteString("(var ")
			b.WriteString(fmt.Sprint(val))
			b.WriteByte(')')
		case *FastDict:
			b.WriteByte('(')
			for i, x := range val.Pairs {
				if i != 0 {
					b.WriteByte(' ')
				}
				SerializeEx(b, x, en, glob, p)
			}
			b.WriteByte(')')
		case Proc:
			serializeProc(b, val, en, glob, p)
		case *Proc:
			serializeProc(b, *val, en, glob, p)
		case *ScmParser:
			b.WriteString("(parser ")
			SerializeEx(b, val.Syntax, glob, glob, p)
			b.WriteByte(' ')
			SerializeEx(b, val.Generator, en, glob, p)
			b.WriteByte(')')
		case func(...Scmer) Scmer:
			serializeNativeFunc(b, val, en)
		case func(*Env, ...Scmer) Scmer:
			serializeNativeFunc(b, val, en)
		case io.Reader:
			var sb strings.Builder
			_, _ = io.Copy(&sb, val)
			b.WriteString(sb.String())
		case string:
			b.WriteByte('"')
			b.WriteString(strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\r", "\\r", "\n", "\\n").Replace(val))
			b.WriteByte('"')
		default:
			b.WriteString(fmt.Sprint(val))
		}
	default:
		b.WriteString(v.String())
	}
}

func serializeProc(b *bytes.Buffer, v Proc, en *Env, glob *Env, parent *Proc) {
	b.WriteString("(lambda ")
	if v.NumVars > 0 && auxTag(v.Params.aux) == tagNil {
		// TODO: deoptimize numbered lambdas when needed
	}
	SerializeEx(b, v.Params, glob, glob, nil)
	b.WriteByte(' ')
	SerializeEx(b, v.Body, v.En, glob, &v)
	if v.NumVars > 0 {
		b.WriteByte(' ')
		b.WriteString(fmt.Sprint(v.NumVars))
	}
	b.WriteByte(')')
}

func serializeNativeFunc(b *bytes.Buffer, fn any, en *Env) {
	switch f := fn.(type) {
	case func(...Scmer) Scmer:
		if col, rev, ok := LookupCollate(f); ok {
			b.WriteString("(collate \"")
			b.WriteString(strings.ReplaceAll(col, "\"", "\\\""))
			b.WriteString("\" ")
			if rev {
				b.WriteString("true")
			} else {
				b.WriteString("false")
			}
			b.WriteByte(')')
			return
		}
	}
	fnPtr := reflect.ValueOf(fn).Pointer()
	en2 := en
	for en2 != nil {
		for k, v := range en2.Vars {
			if auxTag(v.aux) == tagFunc {
				fv := v.Func()
				ov := reflect.ValueOf(fv)
				if ov.Kind() == reflect.Func && ov.Pointer() == fnPtr {
					b.WriteString(string(k))
					return
				}
			}
		}
		en2 = en2.Outer
	}
	b.WriteString("[unserializable native func]")
}
