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
	switch v.GetTag() {
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
	case tagProc:
		// Pretty-print procedures as (lambda ...) expressions without
		// serializing the captured environment to avoid recursion.
		var b bytes.Buffer
		serializeProcShallow(&b, *v.Proc(), &Globalenv)
		return b.String()
	case tagJIT:
		var b bytes.Buffer
		serializeProcShallow(&b, v.JIT().Proc, &Globalenv)
		return b.String()
	case tagFastDict:
		fd := v.FastDict()
		if fd == nil {
			return "()"
		}
		l := make([]string, len(fd.Pairs))
		for i, x := range fd.Pairs {
			l[i] = String(x)
		}
		return "(" + strings.Join(l, " ") + ")"
	case tagSourceInfo:
		return String(v.SourceInfo().value)
	case tagAny:
		if si, ok := v.Any().(SourceInfo); ok {
			return String(si.value)
		}
		if idx, ok := v.Any().(NthLocalVar); ok {
			return fmt.Sprintf("(var %d)", idx)
		}
		if _, ok := v.Any().(func(...Scmer) Scmer); ok {
			return "[native func]"
		}
		if _, ok := v.Any().(func(*Env, ...Scmer) Scmer); ok {
			return "[native func]"
		}
		if r, ok := v.Any().(io.Reader); ok {
			var sb strings.Builder
			_, _ = io.Copy(&sb, r)
			return sb.String()
		}
		if s, ok := v.Any().(string); ok {
			return s
		}
		return fmt.Sprint(v.Any())
	default:
		return fmt.Sprintf("<scmer %d>", v.GetTag())
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
	switch v.GetTag() {
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
	case tagProc:
		// Serialize compiled procedures as (lambda ...) expressions, but
		// avoid walking the captured environment here to prevent cycles.
		serializeProcShallow(b, *v.Proc(), glob)
	case tagFastDict:
		fd := v.FastDict()
		b.WriteByte('(')
		if fd != nil {
			for i, x := range fd.Pairs {
				if i != 0 {
					b.WriteByte(' ')
				}
				SerializeEx(b, x, en, glob, p)
			}
		}
		b.WriteByte(')')
	case tagSourceInfo:
		SerializeEx(b, v.SourceInfo().value, en, glob, p)
	case tagRegex:
		b.WriteString(v.Regex().String())
	case tagNthLocalVar:
		idx := v.NthLocalVar()
		if p != nil && p.NumVars >= int(idx) && p.Params.GetTag() == tagSlice {
			params := p.Params.Slice()
			if int(idx) < len(params) && params[idx].IsSymbol() {
				b.WriteString(params[idx].String())
				return
			}
		}
		b.WriteString("(var ")
		b.WriteString(fmt.Sprint(idx))
		b.WriteByte(')')
	case tagAny:
		if si, ok := v.Any().(SourceInfo); ok {
			SerializeEx(b, si.value, en, glob, p)
			return
		}
		if idx, ok := v.Any().(NthLocalVar); ok {
			if p != nil && p.NumVars >= int(idx) && p.Params.GetTag() == tagSlice {
				params := p.Params.Slice()
				if int(idx) < len(params) && params[idx].IsSymbol() {
					b.WriteString(params[idx].String())
					return
				}
			}
			b.WriteString("(var ")
			b.WriteString(fmt.Sprint(idx))
			b.WriteByte(')')
			return
		}
		if sp, ok := v.Any().(*ScmParser); ok {
			b.WriteString("(parser ")
			SerializeEx(b, sp.Syntax, glob, glob, p)
			b.WriteByte(' ')
			SerializeEx(b, sp.Generator, en, glob, p)
			b.WriteByte(')')
			return
		}
		if f1, ok := v.Any().(func(...Scmer) Scmer); ok {
			serializeNativeFunc(b, f1, en)
			return
		}
		if f2, ok := v.Any().(func(*Env, ...Scmer) Scmer); ok {
			serializeNativeFunc(b, f2, en)
			return
		}
		if r, ok := v.Any().(io.Reader); ok {
			var sb strings.Builder
			_, _ = io.Copy(&sb, r)
			b.WriteString(sb.String())
			return
		}
		if s, ok := v.Any().(string); ok {
			b.WriteByte('"')
			b.WriteString(strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\r", "\\r", "\n", "\\n").Replace(s))
			b.WriteByte('"')
			return
		}
		b.WriteString(fmt.Sprint(v.Any()))
	case tagJIT:
		jep := v.JIT()
		serializeProcShallow(b, jep.Proc, glob)
	default:
		b.WriteString(v.String())
	}
}

func serializeProc(b *bytes.Buffer, v Proc, en *Env, glob *Env, parent *Proc) {
	b.WriteString("(lambda ")
	if v.NumVars > 0 && v.Params.GetTag() == tagNil {
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

// serializeProcShallow prints a procedure as a (lambda ...) form without
// embedding environment bindings. This avoids recursive printing when the
// closure captures itself or large environments.
func serializeProcShallow(b *bytes.Buffer, v Proc, glob *Env) {
	b.WriteString("(lambda ")
	SerializeEx(b, v.Params, glob, glob, nil)
	b.WriteByte(' ')
	// Print body using global env to avoid emitting (begin ... (define ...))
	SerializeEx(b, v.Body, glob, glob, &v)
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
			if v.GetTag() == tagFunc {
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
