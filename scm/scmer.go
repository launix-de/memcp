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

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
	"unsafe"
)

// Scmer is a compact tagged value container (16 bytes). !! NEVER CHANGE IT TO MORE THAN THAT, THE STRUCT SIZE IS CRUCIAL FOR PERFORMANCE
type Scmer struct {
	ptr *byte  // must always be a valid pointer; integer and float encoding: data is stored in aux and ptr contains a dummy that identifies the type
	aux uint64 // type tag + extra data (len, etc.)
}

const (
	scmerStructOverhead = uint(16)
	goAllocOverhead     = uint(16)
)

// ComputeSize allows Scmer to satisfy storages that expect Sizable.
// It approximates the total memory consumption of the value including
// the inline Scmer representation and any heap allocations the value
// references.
func (s Scmer) ComputeSize() uint {
	return ComputeSize(s)
}

// Type tags (upper 16 bits of aux)
// Software Contract: data will ALWAYS be stored with the correct tag, so a tagAny will never store an integer value or a []Scmer, so e.g. a scm.Proc will never be packed into an interface{} by NewAny but always be stored in NewProc()
const (
	tagNil = iota
	tagString
	tagSymbol
	tagFloat
	tagInt
	tagBool
	tagSlice
	tagVector // []float64
	tagFunc
	tagFuncEnv
	tagProc
	tagParser
	tagNthLocalVar
	tagSourceInfo
	tagFastDict
	tagAny
	// custom tags >= 100
)

var scmerIntSentinel byte
var scmerFloatSentinel byte

// Helpers
func makeAux(tag uint16, val uint64) uint64 {
	return uint64(tag)<<48 | (val & ((1 << 48) - 1))
}
func auxTag(aux uint64) uint16 { return uint16(aux >> 48) }
func auxVal(aux uint64) uint64 { return aux & ((1 << 48) - 1) }
func (s Scmer) GetTag() uint16 {
	if s.ptr == &scmerIntSentinel {
		return tagInt
	}
	if s.ptr == &scmerFloatSentinel {
		return tagFloat
	}
	return auxTag(s.aux)
}

//
// Constructors
//

func NewNil() Scmer { return Scmer{nil, makeAux(tagNil, 0)} }

func NewBool(b bool) Scmer {
	if b {
		return Scmer{nil, makeAux(tagBool, 1)}
	}
	return Scmer{nil, makeAux(tagBool, 0)}
}

func NewInt(i int64) Scmer {
	return Scmer{&scmerIntSentinel, uint64(i)}
}

func NewFloat(f float64) Scmer {
	bits := math.Float64bits(f)
	return Scmer{&scmerFloatSentinel, bits}
}

func NewString(s string) Scmer {
	if len(s) == 0 {
		return Scmer{nil, makeAux(tagString, 0)}
	}
	return Scmer{unsafe.StringData(s), makeAux(tagString, uint64(len(s)))}
}

func NewSymbol(sym string) Scmer {
	if len(sym) == 0 {
		return Scmer{nil, makeAux(tagSymbol, 0)}
	}
	return Scmer{unsafe.StringData(sym), makeAux(tagSymbol, uint64(len(sym)))}
}

func NewSlice(slice []Scmer) Scmer {
	if len(slice) == 0 {
		return Scmer{nil, makeAux(tagSlice, 0)}
	}
	data := unsafe.SliceData(slice)
	return Scmer{(*byte)(unsafe.Pointer(data)), makeAux(tagSlice, uint64(len(slice)))}
}

func NewVector(vec []float64) Scmer {
	if len(vec) == 0 {
		return Scmer{nil, makeAux(tagVector, 0)}
	}
	data := unsafe.SliceData(vec)
	return Scmer{(*byte)(unsafe.Pointer(data)), makeAux(tagVector, uint64(len(vec)))}
}

func NewScmParser(fd *ScmParser) Scmer {
	var ptr *byte
	if fd != nil {
		ptr = (*byte)(unsafe.Pointer(fd))
	}
	return Scmer{ptr, makeAux(tagParser, 0)}
}

func NewFastDict(fd *FastDict) Scmer {
	var ptr *byte
	if fd != nil {
		ptr = (*byte)(unsafe.Pointer(fd))
	}
	return Scmer{ptr, makeAux(tagFastDict, 0)}
}

func NewFunc(fn func(...Scmer) Scmer) Scmer {
	ptr := new(func(...Scmer) Scmer)
	*ptr = fn
	return Scmer{(*byte)(unsafe.Pointer(ptr)), makeAux(tagFunc, 0)}
}

func NewFuncEnv(fn func(*Env, ...Scmer) Scmer) Scmer {
	ptr := new(func(*Env, ...Scmer) Scmer)
	*ptr = fn
	return Scmer{(*byte)(unsafe.Pointer(ptr)), makeAux(tagFuncEnv, 0)}
}

func NewProcStruct(p Proc) Scmer {
	cp := p
	return NewProc(&cp)
}

func NewProc(p *Proc) Scmer {
	if p == nil {
		return Scmer{nil, makeAux(tagProc, 0)}
	}
	ptr := new(Proc)
	*ptr = *p
	return Scmer{(*byte)(unsafe.Pointer(ptr)), makeAux(tagProc, 0)}
}

func NewNthLocalVar(idx NthLocalVar) Scmer {
	return Scmer{nil, makeAux(tagNthLocalVar, uint64(idx))}
}

func NewSourceInfo(si SourceInfo) Scmer {
	ptr := new(SourceInfo)
	*ptr = si
	return Scmer{(*byte)(unsafe.Pointer(ptr)), makeAux(tagSourceInfo, 0)}
}

func NewAny(v any) Scmer {
	p := new(any)
	*p = v
	return Scmer{(*byte)(unsafe.Pointer(p)), makeAux(tagAny, 0)}
}

func FromAny(v any) Scmer {
	switch vv := v.(type) {
	case Scmer:
		return vv
	case nil:
		return NewNil()
	case bool:
		return NewBool(vv)
	case int:
		return NewInt(int64(vv))
	case int8:
		return NewInt(int64(vv))
	case int16:
		return NewInt(int64(vv))
	case int32:
		return NewInt(int64(vv))
	case int64:
		return NewInt(vv)
	case uint:
		return NewInt(int64(vv))
	case uint8:
		return NewInt(int64(vv))
	case uint16:
		return NewInt(int64(vv))
	case uint32:
		return NewInt(int64(vv))
	case uint64:
		return NewInt(int64(vv))
	case float32:
		return NewFloat(float64(vv))
	case float64:
		return NewFloat(vv)
	case string:
		return NewString(vv)
	case Symbol:
		return NewSymbol(string(vv))
	case []Scmer:
		return NewSlice(vv)
	case []float64:
		return NewVector(vv)
	case func(...Scmer) Scmer:
		return NewFunc(vv)
	case func(*Env, ...Scmer) Scmer:
		return NewFuncEnv(vv)
	case Proc:
		return NewProcStruct(vv)
	case *Proc:
		if vv == nil {
			return NewProc(nil)
		}
		return NewProc(vv)
	case SourceInfo:
		return NewSourceInfo(vv)
	case *SourceInfo:
		if vv == nil {
			return NewNil()
		}
		return NewSourceInfo(*vv)
	case *FastDict:
		return NewFastDict(vv)
	case *ScmParser:
		return NewScmParser(vv)
	case FastDict:
		ptr := new(FastDict)
		*ptr = vv
		return NewFastDict(ptr)
	default:
		return NewAny(v)
	}
}

//
// Custom pointer-like values
//

func NewCustom(tag uint16, ptr unsafe.Pointer) Scmer {
	if tag < 100 {
		panic("custom tags should be >= 100 to avoid conflicts")
	}
	return Scmer{(*byte)(ptr), makeAux(tag, 0)}
}

func (s Scmer) IsCustom(tag uint16) bool {
	if s.ptr == &scmerIntSentinel {
		return tag == tagInt
	}
	if s.ptr == &scmerFloatSentinel {
		return tag == tagFloat
	}
	return auxTag(s.aux) == tag
}

func (s Scmer) Custom(tag uint16) unsafe.Pointer {
	if s.GetTag() != tag {
		panic("wrong custom tag")
	}
	return unsafe.Pointer(s.ptr)
}

//
// Accessors with conversion
//

func (s Scmer) IsNil() bool {
	if s.ptr == &scmerIntSentinel || s.ptr == &scmerFloatSentinel {
		return false
	}
	return auxTag(s.aux) == tagNil
}

func (s Scmer) IsBool() bool {
	if s.ptr == &scmerIntSentinel || s.ptr == &scmerFloatSentinel {
		return false
	}
	return auxTag(s.aux) == tagBool
}

func (s Scmer) IsInt() bool {
	if s.ptr == &scmerIntSentinel {
		return true
	}
	if s.ptr == &scmerFloatSentinel {
		return false
	}
	return auxTag(s.aux) == tagInt
}

func (s Scmer) IsFloat() bool {
	if s.ptr == &scmerFloatSentinel {
		return true
	}
	if s.ptr == &scmerIntSentinel {
		return false
	}
	return auxTag(s.aux) == tagFloat
}

func (s Scmer) IsString() bool {
	if s.ptr == &scmerIntSentinel || s.ptr == &scmerFloatSentinel {
		return false
	}
	return auxTag(s.aux) == tagString
}

func (s Scmer) IsSymbol() bool {
	if s.ptr == &scmerIntSentinel || s.ptr == &scmerFloatSentinel {
		return false
	}
	return auxTag(s.aux) == tagSymbol
}

func (s Scmer) SymbolEquals(name string) bool {
	return s.GetTag() == tagSymbol && s.String() == name
}

func (s Scmer) IsSlice() bool {
	if s.ptr == &scmerIntSentinel || s.ptr == &scmerFloatSentinel {
		return false
	}
	return auxTag(s.aux) == tagSlice
}

func (s Scmer) IsVector() bool {
	if s.ptr == &scmerIntSentinel || s.ptr == &scmerFloatSentinel {
		return false
	}
	return auxTag(s.aux) == tagVector
}

func (s Scmer) IsFastDict() bool {
	if s.ptr == &scmerIntSentinel || s.ptr == &scmerFloatSentinel {
		return false
	}
	return auxTag(s.aux) == tagFastDict
}

func (s Scmer) IsParser() bool {
	if s.ptr == &scmerIntSentinel || s.ptr == &scmerFloatSentinel {
		return false
	}
	return auxTag(s.aux) == tagParser
}

func (s Scmer) Bool() bool {
	switch s.GetTag() {
	case tagNil:
		return false
	case tagBool:
		return auxVal(s.aux) != 0
	case tagInt:
		return int64(s.aux) != 0
	case tagFloat:
		return math.Float64frombits(s.aux) != 0.0
	case tagString, tagSymbol:
		return s.String() != ""
	case tagSlice:
		return len(s.Slice()) > 0
	case tagVector:
		return len(s.Vector()) > 0
	case tagFastDict:
		fd := s.FastDict()
		return len(fd.Pairs) > 0
	default:
		return true
	}
}

func (s Scmer) Int() int64 {
	switch s.GetTag() {
	case tagNil:
		return 0
	case tagInt:
		return int64(s.aux)
	case tagFloat:
		return int64(math.Float64frombits(s.aux))
	case tagString, tagSymbol:
		v, err := strconv.ParseInt(s.String(), 10, 64)
		if err != nil {
			return 0
		}
		return v
	case tagBool:
		if auxVal(s.aux) != 0 {
			return 1
		}
		return 0
	case tagAny:
		return 1
	default:
		return 1
	}
	return 0
}

func (s Scmer) Float() float64 {
	switch s.GetTag() {
	case tagNil:
		return 0.0
	case tagFloat:
		return math.Float64frombits(s.aux)
	case tagInt:
		return float64(int64(s.aux))
	case tagString, tagSymbol:
		v, err := strconv.ParseFloat(s.String(), 64)
		if err != nil {
			return 0.0
		}
		return v
	case tagBool:
		if auxVal(s.aux) != 0 {
			return 1.0
		}
		return 0.0
	default:
		return 1.0
	}
}

func (s Scmer) String() string {
	switch s.GetTag() {
	case tagString, tagSymbol:
		hdr := [2]uintptr{uintptr(unsafe.Pointer(s.ptr)), uintptr(auxVal(s.aux))}
		return *(*string)(unsafe.Pointer(&hdr))
	case tagInt:
		return strconv.FormatInt(int64(s.aux), 10)
	case tagFloat:
		return strconv.FormatFloat(math.Float64frombits(s.aux), 'g', -1, 64)
	case tagBool:
		if auxVal(s.aux) != 0 {
			return "true"
		}
		return "false"
	case tagNil:
		return "nil"
	case tagFunc:
		decl := DeclarationForValue(s)
		if decl != nil {
			return decl.Name
		}
		return "[func]"
	case tagFuncEnv:
		return "[func]"
	case tagSlice:
		sl := s.Slice()
		if len(sl) == 0 {
			return "()"
		}
		var sb strings.Builder
		sb.WriteString("(")
		for i, el := range sl {
			if i > 0 {
				sb.WriteString(" ")
			}
			el.Write(&sb)
		}
		sb.WriteString(")")
		return sb.String()
	case tagFastDict:
		fd := s.FastDict()
		if fd == nil {
			return "()"
		}
		parts := make([]string, len(fd.Pairs))
		for i, el := range fd.Pairs {
			parts[i] = el.String()
		}
		return "(" + strings.Join(parts, " ") + ")"
	case tagParser:
		return fmt.Sprint(s.Parser())
		return "[parser]"
	case tagSourceInfo:
		return s.SourceInfo().String()
	default:
		if s.GetTag() == tagAny {
			return fmt.Sprintf("%v", *(*any)(unsafe.Pointer(s.ptr)))
		}
		return fmt.Sprintf("<custom %d>", s.GetTag())
	}
}

// Stream returns an io.Reader for the value.
// - If the underlying value is already an io.Reader (streams are encoded as Any), it is passed through.
// - Otherwise, the value is converted to its string form and a strings.Reader is returned.
func (s Scmer) Stream() io.Reader {
	if s.GetTag() == tagAny {
		if r, ok := s.Any().(io.Reader); ok {
			return r
		}
	}
	return strings.NewReader(s.String())
}

func (s Scmer) Slice() []Scmer {
	if s.GetTag() != tagSlice {
		panic("not slice")
	}
	ln := int(auxVal(s.aux))
	if ln == 0 || s.ptr == nil {
		return nil
	}
	head := (*Scmer)(unsafe.Pointer(s.ptr))
	headers := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(head)),
		Len:  ln,
		Cap:  ln,
	}
	return *(*[]Scmer)(unsafe.Pointer(&headers))
}

func (s Scmer) Vector() []float64 {
	if s.GetTag() != tagVector {
		panic("not vector")
	}
	hdr := [3]uintptr{uintptr(unsafe.Pointer(s.ptr)), uintptr(auxVal(s.aux)), uintptr(auxVal(s.aux))}
	return *(*[]float64)(unsafe.Pointer(&hdr))
}

func (s Scmer) FastDict() *FastDict {
	if s.GetTag() != tagFastDict {
		panic("not fastdict")
	}
	return (*FastDict)(unsafe.Pointer(s.ptr))
}

func (s Scmer) Parser() *ScmParser {
	if s.GetTag() != tagParser {
		panic("not parser")
	}
	return (*ScmParser)(unsafe.Pointer(s.ptr))
}

func (s Scmer) Func() func(...Scmer) Scmer {
	if s.GetTag() != tagFunc {
		panic("not function")
	}
	return *(*func(...Scmer) Scmer)(unsafe.Pointer(s.ptr))
}

func (s Scmer) FuncEnv() func(*Env, ...Scmer) Scmer {
	if s.GetTag() != tagFuncEnv {
		panic("not environment function")
	}
	return *(*func(*Env, ...Scmer) Scmer)(unsafe.Pointer(s.ptr))
}

func (s Scmer) IsProc() bool {
	if s.ptr == &scmerIntSentinel || s.ptr == &scmerFloatSentinel {
		return false
	}
	return auxTag(s.aux) == tagProc
}

func (s Scmer) Proc() *Proc {
	if s.GetTag() != tagProc {
		panic("not proc")
	}
	return (*Proc)(unsafe.Pointer(s.ptr))
}

func (s Scmer) IsNthLocalVar() bool {
	if s.ptr == &scmerIntSentinel || s.ptr == &scmerFloatSentinel {
		return false
	}
	return auxTag(s.aux) == tagNthLocalVar
}

func (s Scmer) NthLocalVar() NthLocalVar {
	if s.GetTag() != tagNthLocalVar {
		panic("not nth local var")
	}
	return NthLocalVar(auxVal(s.aux))
}

func (s Scmer) IsSourceInfo() bool {
	if s.ptr == &scmerIntSentinel || s.ptr == &scmerFloatSentinel {
		return false
	}
	return auxTag(s.aux) == tagSourceInfo
}

func (s Scmer) SourceInfo() *SourceInfo {
	if s.GetTag() != tagSourceInfo {
		panic("not source info")
	}
	return (*SourceInfo)(unsafe.Pointer(s.ptr))
}

// Symbol returns the Scheme symbol value as Go string.
func (s Scmer) Symbol() Symbol {
	if s.GetTag() != tagSymbol {
		panic("not symbol")
	}
	return Symbol(s.String())
}

// Any unwraps the Scmer into a Go value for legacy code.
func (s Scmer) Any() any {
	switch s.GetTag() {
	case tagNil:
		return nil
	case tagBool:
		return s.Bool()
	case tagInt:
		return s.Int()
	case tagFloat:
		return s.Float()
	case tagString:
		return s.String()
	case tagSymbol:
		return Symbol(s.String())
	case tagSlice:
		return s.Slice()
	case tagVector:
		return s.Vector()
	case tagFunc:
		return s.Func()
	case tagFuncEnv:
		return s.FuncEnv()
	case tagProc:
		return s.Proc()
	case tagNthLocalVar:
		return s.NthLocalVar()
	case tagSourceInfo:
		return *s.SourceInfo()
	case tagFastDict:
		return s.FastDict()
	case tagParser:
		return s.Parser()
	case tagAny:
		return *(*any)(unsafe.Pointer(s.ptr))
	default:
		panic(fmt.Sprintf("unknown tag %d in Any", s.GetTag()))
	}
}

// Compatibility helpers for legacy code paths.
func ToBool(v Scmer) bool     { return v.Bool() }
func ToInt(v Scmer) int       { return int(v.Int()) }
func ToFloat(v Scmer) float64 { return v.Float() }

// MarshalJSON encodes a Scmer into JSON by converting it to native Go
// types first (nil, bool, float64/int64, string, []any, map-like assoc
// list), ensuring stable and portable JSON output for persistence and APIs.
func (s Scmer) MarshalJSON() ([]byte, error) {
	// Custom, stable encoding for persistence of SCM values.
	var toJSONable func(Scmer) any
	// helper: find name of native func in Globalenv
	toJSONable = func(v Scmer) any {
		switch v.GetTag() {
		case tagNil:
			return nil
		case tagBool:
			return v.Bool()
		case tagInt:
			return v.Int()
		case tagFloat:
			return v.Float()
		case tagString:
			s := v.String()
			if !utf8.ValidString(s) {
				return map[string]any{"bytes": base64.StdEncoding.EncodeToString([]byte(s))}
			}
			return s
		case tagSymbol:
			return map[string]any{"symbol": v.String()}
		case tagSlice:
			list := v.Slice()
			out := make([]any, len(list))
			for i, it := range list {
				out[i] = toJSONable(it)
			}
			return out
		case tagFastDict:
			fd := v.FastDict()
			if fd == nil {
				return []any{}
			}
			pairs := fd.ToList()
			out := make([]any, len(pairs))
			for i, it := range pairs {
				out[i] = toJSONable(it)
			}
			return out
		case tagParser:
			return map[string]any{"parser": "TODO"}
		case tagVector:
			return v.Vector()
		case tagFunc:
			// Try to encode as the symbolic name if known; otherwise as "?"
			decl := DeclarationForValue(v)
			if decl != nil {
				return map[string]any{"symbol": decl.Name}
			}
			return map[string]any{"symbol": "?"}
		case tagFuncEnv:
			return map[string]any{"symbol": "?"}
		case tagProc:
			p := v.Proc()
			arr := make([]any, 0, 4)
			arr = append(arr, map[string]any{"symbol": "lambda"})
			arr = append(arr, toJSONable(p.Params))
			arr = append(arr, toJSONable(p.Body))
			if p.NumVars > 0 {
				arr = append(arr, p.NumVars)
			}
			return arr
		case tagSourceInfo:
			return toJSONable(v.SourceInfo().value)
		default:
			// Unknown custom tag -> fall back to string form
			return v.String()
		}
	}
	return json.Marshal(toJSONable(s))
}

func (s *Scmer) Write(w io.Writer) {
	// Stream a textual representation without intermediate large strings.
	// Mirrors String() formatting but writes directly and recurses for lists.
	if s == nil {
		io.WriteString(w, "nil")
		return
	}
	switch s.GetTag() {
	case tagNil:
		io.WriteString(w, "nil")
	case tagBool:
		if auxVal(s.aux) != 0 {
			io.WriteString(w, "true")
		} else {
			io.WriteString(w, "false")
		}
	case tagInt:
		// Fast path without allocations
		var buf [32]byte
		b := strconv.AppendInt(buf[:0], int64(s.aux), 10)
		w.Write(b)
	case tagFloat:
		var buf [64]byte
		f := math.Float64frombits(s.aux)
		b := strconv.AppendFloat(buf[:0], f, 'g', -1, 64)
		w.Write(b)
	case tagString, tagSymbol:
		io.WriteString(w, s.String())
	case tagFunc:
		io.WriteString(w, "[func]")
	case tagSlice:
		io.WriteString(w, "(")
		list := s.Slice()
		for i := range list {
			if i > 0 {
				io.WriteString(w, " ")
			}
			list[i].Write(w)
		}
		io.WriteString(w, ")")
	case tagParser:
		io.WriteString(w, "[parser]")
	case tagFastDict:
		io.WriteString(w, "(")
		fd := s.FastDict()
		if fd != nil {
			for i := 0; i < len(fd.Pairs); i++ {
				if i > 0 {
					io.WriteString(w, " ")
				}
				fd.Pairs[i].Write(w)
			}
		}
		io.WriteString(w, ")")
	case tagSourceInfo:
		io.WriteString(w, s.SourceInfo().String())
	default:
		if s.GetTag() == tagAny {
			// Fallback: format underlying Go value using fmt
			fmt.Fprintf(w, "%v", *(*any)(unsafe.Pointer(s.ptr)))
			return
		}
		// Unknown tag: fall back to the string representation
		io.WriteString(w, s.String())
	}
}

// UnmarshalJSON decodes JSON into a Scmer by first decoding into
// interface{} and then transforming to Scmer using TransformFromJSON.
func (s *Scmer) UnmarshalJSON(data []byte) error {
	// Use decoder with UseNumber to preserve ints
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return err
	}
	var from func(any) Scmer
	from = func(x any) Scmer {
		switch t := x.(type) {
		case nil:
			return NewNil()
		case bool:
			return NewBool(t)
		case json.Number:
			if i, err := t.Int64(); err == nil {
				return NewInt(i)
			}
			if f, err := t.Float64(); err == nil {
				return NewFloat(f)
			}
			return NewString(string(t))
		case float64:
			// Should not happen with UseNumber, but keep for robustness
			return NewFloat(t)
		case string:
			return NewString(t)
		case map[string]any:
			if sym, ok := t["symbol"]; ok {
				if name, ok2 := sym.(string); ok2 {
					return NewSymbol(name)
				}
			}
			// decode binary strings encoded by MarshalJSON
			if b64, ok := t["bytes"]; ok && len(t) == 1 {
				if str, ok2 := b64.(string); ok2 {
					if raw, err := base64.StdEncoding.DecodeString(str); err == nil {
						return NewString(string(raw))
					}
				}
			}
			// assoc map -> flatten to '("k" v ...)
			pairs := make([]Scmer, 0, len(t)*2)
			for k, v2 := range t {
				pairs = append(pairs, NewString(k), from(v2))
			}
			return NewSlice(pairs)
		case []any:
			// check for lambda form: [{"symbol":"lambda"}, params, body, (numVars?)]
			if len(t) >= 3 {
				if head, ok := t[0].(map[string]any); ok {
					if sym, ok2 := head["symbol"]; ok2 {
						if name, ok3 := sym.(string); ok3 && name == "lambda" {
							params := from(t[1])
							body := from(t[2])
							proc := Proc{Params: params, Body: body, En: &Globalenv}
							if len(t) > 3 {
								switch nv := t[3].(type) {
								case json.Number:
									if i, err := nv.Int64(); err == nil {
										proc.NumVars = int(i)
									}
								case float64:
									proc.NumVars = int(nv)
								case int64:
									proc.NumVars = int(nv)
								case int:
									proc.NumVars = nv
								}
							}
							return NewProcStruct(proc)
						}
					}
				}
			}
			// generic list
			out := make([]Scmer, len(t))
			for i := range t {
				out[i] = from(t[i])
			}
			return NewSlice(out)
		default:
			return FromAny(t)
		}
	}
	*s = from(v)
	return nil
}
