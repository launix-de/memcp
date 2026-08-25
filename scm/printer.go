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

package scm

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

var schemeStringEscaper = strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\r", "\\r", "\n", "\\n")

const (
	fnv64Offset = uint64(14695981039346656037)
	fnv64Prime  = uint64(1099511628211)
)

// schemeTextWriter keeps the hot recursive serializer monomorphic. It either
// forwards to the caller's bytes.Buffer or consumes bytes directly into FNV-1a.
// Avoiding io.Writer/hash.Hash dispatch matters for the many tiny AST tokens.
type schemeTextWriter struct {
	buffer *bytes.Buffer
	hash   uint64
}

func bufferTextWriter(buffer *bytes.Buffer) *schemeTextWriter {
	return &schemeTextWriter{buffer: buffer}
}

func hashTextWriter() *schemeTextWriter {
	return &schemeTextWriter{hash: fnv64Offset}
}

func (w *schemeTextWriter) Write(value []byte) (int, error) {
	if w.buffer != nil {
		return w.buffer.Write(value)
	}
	for _, b := range value {
		w.hash = (w.hash ^ uint64(b)) * fnv64Prime
	}
	return len(value), nil
}

func (w *schemeTextWriter) WriteByte(value byte) error {
	if w.buffer != nil {
		return w.buffer.WriteByte(value)
	}
	w.hash = (w.hash ^ uint64(value)) * fnv64Prime
	return nil
}

func (w *schemeTextWriter) WriteString(value string) (int, error) {
	if w.buffer != nil {
		return w.buffer.WriteString(value)
	}
	for i := 0; i < len(value); i++ {
		w.hash = (w.hash ^ uint64(value[i])) * fnv64Prime
	}
	return len(value), nil
}

func String(v Scmer) string {
	switch v.GetTag() {
	case tagNil:
		return "nil"
	case tagBool, tagInt, tagFloat:
		return v.String()
	case tagString, tagCString, tagBString, tagBSON:
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
	case tagPromise:
		return "[promise]"
	case tagFunc:
		return "[native func]"
	case tagProc:
		// Pretty-print procedures as (lambda ...) expressions without
		// serializing the captured environment to avoid recursion.
		var b bytes.Buffer
		serializeProcShallow(bufferTextWriter(&b), *v.Proc(), &Globalenv)
		return b.String()
	case tagJIT:
		var b bytes.Buffer
		serializeProcShallow(bufferTextWriter(&b), v.JIT().Proc, &Globalenv)
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

// WriteStringValue streams the exact representation returned by String. It is
// used by structural name hashing to avoid building complete AST strings.
func WriteStringValue(w *schemeTextWriter, v Scmer) {
	switch v.GetTag() {
	case tagNil:
		w.WriteString("nil")
	case tagBool, tagString, tagCString, tagBString, tagSymbol:
		w.WriteString(v.String())
	case tagBSON:
		if err := writeBSONJSON(w, bsonRawValue(v)); err != nil {
			panic(err)
		}
	case tagInt:
		var buffer [32]byte
		value := strconv.AppendInt(buffer[:0], v.Int(), 10)
		_, _ = w.Write(value)
	case tagFloat:
		var buffer [64]byte
		value := strconv.AppendFloat(buffer[:0], v.Float(), 'g', -1, 64)
		_, _ = w.Write(value)
	case tagSlice:
		w.WriteByte('(')
		for i, item := range v.Slice() {
			if i > 0 {
				w.WriteByte(' ')
			}
			WriteStringValue(w, item)
		}
		w.WriteByte(')')
	case tagVector:
		w.WriteString("#(")
		for i, item := range v.Vector() {
			if i > 0 {
				w.WriteByte(' ')
			}
			w.WriteString(fmt.Sprint(item))
		}
		w.WriteByte(')')
	case tagPromise:
		w.WriteString("[promise]")
	case tagFunc:
		w.WriteString("[native func]")
	case tagProc:
		serializeProcShallow(w, *v.Proc(), &Globalenv)
	case tagJIT:
		serializeProcShallow(w, v.JIT().Proc, &Globalenv)
	case tagFastDict:
		w.WriteByte('(')
		if dictionary := v.FastDict(); dictionary != nil {
			for i, item := range dictionary.Pairs {
				if i > 0 {
					w.WriteByte(' ')
				}
				WriteStringValue(w, item)
			}
		}
		w.WriteByte(')')
	case tagSourceInfo:
		WriteStringValue(w, v.SourceInfo().value)
	case tagAny:
		if source, ok := v.Any().(SourceInfo); ok {
			WriteStringValue(w, source.value)
			return
		}
		if index, ok := v.Any().(NthLocalVar); ok {
			w.WriteString("(var ")
			w.WriteString(strconv.FormatInt(int64(index), 10))
			w.WriteByte(')')
			return
		}
		if _, ok := v.Any().(func(...Scmer) Scmer); ok {
			w.WriteString("[native func]")
			return
		}
		if _, ok := v.Any().(func(*Env, ...Scmer) Scmer); ok {
			w.WriteString("[native func]")
			return
		}
		if reader, ok := v.Any().(io.Reader); ok {
			var buffer [4096]byte
			for {
				read, err := reader.Read(buffer[:])
				if read > 0 {
					_, _ = w.Write(buffer[:read])
				}
				if err != nil {
					break
				}
			}
			return
		}
		w.WriteString(fmt.Sprint(v.Any()))
	default:
		w.WriteString("<scmer ")
		w.WriteString(strconv.FormatInt(int64(v.GetTag()), 10))
		w.WriteByte('>')
	}
}
func SerializeToString(v Scmer, glob *Env) string {
	var b bytes.Buffer
	SerializeEx(&b, v, glob, glob, nil)
	return b.String()
}

func Serialize(b *bytes.Buffer, v Scmer, glob *Env) {
	serializeEx(bufferTextWriter(b), v, glob, glob, nil)
}

func SerializeEx(b *bytes.Buffer, v Scmer, en *Env, glob *Env, p *Proc) {
	serializeEx(bufferTextWriter(b), v, en, glob, p)
}

func serializeEx(b *schemeTextWriter, v Scmer, en *Env, glob *Env, p *Proc) {
	if en != glob {
		b.WriteString("(begin ")
		for k, v := range en.Vars {
			// if Symbol is defined in a lambda, print the real value
			// filter out redefinition of global functions
			if gv, ok := glob.Vars[k]; !ok || !Equal(gv, v) {
				b.WriteString("(define ")
				b.WriteString(string(k))
				b.WriteString(" ")
				serializeEx(b, v, en.Outer, glob, p)
				b.WriteString(") ")
			}
		}
		serializeEx(b, v, en.Outer, glob, p)
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
	case tagInt:
		var buffer [32]byte
		value := strconv.AppendInt(buffer[:0], v.Int(), 10)
		_, _ = b.Write(value)
	case tagFloat:
		var buffer [64]byte
		value := strconv.AppendFloat(buffer[:0], v.Float(), 'g', -1, 64)
		_, _ = b.Write(value)
	case tagString, tagCString, tagBString:
		b.WriteByte('"')
		b.WriteString(schemeStringEscaper.Replace(v.String()))
		b.WriteByte('"')
	case tagBSON:
		b.WriteString("(json_parse_bson \"")
		b.WriteString(schemeStringEscaper.Replace(v.String()))
		b.WriteString("\")")
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
			serializeEx(b, slice[1], en, glob, nil)
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
			serializeEx(b, x, en, glob, p)
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
	case tagPromise:
		b.WriteString("[promise]")
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
				serializeEx(b, x, en, glob, p)
			}
		}
		b.WriteByte(')')
	case tagSourceInfo:
		serializeEx(b, v.SourceInfo().value, en, glob, p)
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
			serializeEx(b, si.value, en, glob, p)
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
			serializeEx(b, sp.Syntax, glob, glob, p)
			b.WriteByte(' ')
			serializeEx(b, sp.Generator, en, glob, p)
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
			b.WriteString(schemeStringEscaper.Replace(s))
			b.WriteByte('"')
			return
		}
		b.WriteString(fmt.Sprint(v.Any()))
	case tagParser:
		sp := v.Parser()
		b.WriteString("(parser ")
		serializeEx(b, sp.Syntax, glob, glob, p)
		b.WriteByte(' ')
		serializeEx(b, sp.Generator, en, glob, p)
		b.WriteByte(')')
	case tagJIT:
		jep := v.JIT()
		serializeProcShallow(b, jep.Proc, glob)
	default:
		b.WriteString(v.String())
	}
}

func serializeProc(b *schemeTextWriter, v Proc, en *Env, glob *Env, parent *Proc) {
	b.WriteString("(lambda ")
	if v.NumVars > 0 && v.Params.GetTag() == tagNil {
		// TODO: deoptimize numbered lambdas when needed
	}
	serializeEx(b, v.Params, glob, glob, nil)
	b.WriteByte(' ')
	serializeEx(b, v.Body, v.En, glob, &v)
	if v.NumVars > 0 {
		b.WriteByte(' ')
		b.WriteString(fmt.Sprint(v.NumVars))
	}
	b.WriteByte(')')
}

// serializeProcShallow prints a procedure as a (lambda ...) form without
// embedding environment bindings. This avoids recursive printing when the
// closure captures itself or large environments.
func serializeProcShallow(b *schemeTextWriter, v Proc, glob *Env) {
	b.WriteString("(lambda ")
	serializeEx(b, v.Params, glob, glob, nil)
	b.WriteByte(' ')
	// Print body using global env to avoid emitting (begin ... (define ...))
	serializeEx(b, v.Body, glob, glob, &v)
	if v.NumVars > 0 {
		b.WriteByte(' ')
		b.WriteString(fmt.Sprint(v.NumVars))
	}
	b.WriteByte(')')
}

func serializeNativeFunc(b *schemeTextWriter, fn any, en *Env) {
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

// PrettyPrint formats a Scmer value as a human-readable, indented string.
// Atoms and expressions whose compact serialization fits within width characters
// are kept on a single line. Longer list expressions are expanded: the head on
// the opening line, each argument on its own indented line, and the closing
// parenthesis on a line by itself. Consecutive lines that contain only closing
// parentheses are merged into a single line at the outermost indent level.
func PrettyPrint(v Scmer, glob *Env, width int) string {
	lines := prettyLines(v, glob, width, 0)
	// merge consecutive closing-paren-only lines
	result := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimLeft(line, "\t")
		if strings.Trim(trimmed, ")") == "" && trimmed != "" {
			// this line is only closing parens; collect following ones
			merged := line
			for i+1 < len(lines) {
				next := lines[i+1]
				nextTrimmed := strings.TrimLeft(next, "\t")
				if strings.Trim(nextTrimmed, ")") == "" && nextTrimmed != "" {
					merged = strings.TrimLeft(merged, "\t") // drop indent of earlier line
					// use indentation of the later (outer) line
					merged = next + strings.TrimLeft(merged, "\t")
					i++
				} else {
					break
				}
			}
			result = append(result, merged)
		} else {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

// prettyLines returns the lines (already indented with tabs) for a value.
func prettyLines(v Scmer, glob *Env, width int, indent int) []string {
	compact := SerializeToString(v, glob)
	indentStr := strings.Repeat("\t", indent)
	if len(compact) <= width || v.GetTag() != tagSlice {
		return []string{indentStr + compact}
	}
	slice := v.Slice()
	if len(slice) == 0 {
		return []string{indentStr + "()"}
	}
	headCompact := SerializeToString(slice[0], glob)
	lines := []string{indentStr + "(" + headCompact}
	for _, arg := range slice[1:] {
		lines = append(lines, prettyLines(arg, glob, width, indent+1)...)
	}
	// closing paren: append to last line if it is already a closing-paren-only line,
	// otherwise add a new line at the current indent
	last := lines[len(lines)-1]
	lastTrimmed := strings.TrimLeft(last, "\t")
	if strings.Trim(lastTrimmed, ")") == "" && lastTrimmed != "" {
		lines[len(lines)-1] = last + ")"
	} else {
		lines = append(lines, indentStr+")")
	}
	return lines
}
