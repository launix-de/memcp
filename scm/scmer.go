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
	"fmt"
	"math"
	"strconv"
	"unsafe"
)

// Scmer is a compact tagged value container (16 bytes).
type Scmer struct {
	ptr unsafe.Pointer
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
	tagProc
	tagNthLocalVar
	tagAny
	// custom tags >= 100
)

// Helpers
func makeAux(tag uint16, val uint64) uint64 {
	return uint64(tag)<<48 | (val & ((1 << 48) - 1))
}
func auxTag(aux uint64) uint16 { return uint16(aux >> 48) }
func auxVal(aux uint64) uint64 { return aux & ((1 << 48) - 1) }

//
// Constructors
//

func NewNil() Scmer { return Scmer{nil, makeAux(tagNil, 0)} }

func NewBool(b bool) Scmer {
	if b {
		return Scmer{unsafe.Pointer(uintptr(1)), makeAux(tagBool, 0)}
	}
	return Scmer{unsafe.Pointer(uintptr(0)), makeAux(tagBool, 0)}
}

func NewInt(i int64) Scmer {
	return Scmer{unsafe.Pointer(uintptr(i)), makeAux(tagInt, 0)}
}

func NewFloat(f float64) Scmer {
	bits := math.Float64bits(f)
	return Scmer{unsafe.Pointer(uintptr(bits)), makeAux(tagFloat, 0)}
}

func NewString(s string) Scmer {
	if len(s) == 0 {
		return Scmer{nil, makeAux(tagString, 0)}
	}
	sh := (*[2]uintptr)(unsafe.Pointer(&s))
	return Scmer{unsafe.Pointer(sh[0]), makeAux(tagString, uint64(len(s)))}
}

func NewSymbol(sym string) Scmer {
	if len(sym) == 0 {
		return Scmer{nil, makeAux(tagSymbol, 0)}
	}
	sh := (*[2]uintptr)(unsafe.Pointer(&sym))
	return Scmer{unsafe.Pointer(sh[0]), makeAux(tagSymbol, uint64(len(sym)))}
}

func NewSlice(slice []Scmer) Scmer {
	if len(slice) == 0 {
		return Scmer{nil, makeAux(tagSlice, 0)}
	}
	sh := (*[3]uintptr)(unsafe.Pointer(&slice))
	return Scmer{unsafe.Pointer(sh[0]), makeAux(tagSlice, uint64(len(slice)))}
}

func NewVector(vec []float64) Scmer {
	sh := (*[3]uintptr)(unsafe.Pointer(&vec))
	return Scmer{unsafe.Pointer(sh[0]), makeAux(tagVector, uint64(len(vec)))}
}

func NewFunc(fn func(...Scmer) Scmer) Scmer {
	ptr := new(func(...Scmer) Scmer)
	*ptr = fn
	return Scmer{unsafe.Pointer(ptr), makeAux(tagFunc, 0)}
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
	return Scmer{unsafe.Pointer(ptr), makeAux(tagProc, 0)}
}

func NewNthLocalVar(idx NthLocalVar) Scmer {
	return Scmer{unsafe.Pointer(uintptr(idx)), makeAux(tagNthLocalVar, 0)}
}

func NewAny(v any) Scmer {
	p := new(any)
	*p = v
	return Scmer{unsafe.Pointer(p), makeAux(tagAny, 0)}
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
	case Proc:
		return NewProcStruct(vv)
	case *Proc:
		if vv == nil {
			return NewProc(nil)
		}
		return NewProc(vv)
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
	return Scmer{ptr, makeAux(tag, 0)}
}

func (s Scmer) IsCustom(tag uint16) bool {
	return auxTag(s.aux) == tag
}

func (s Scmer) Custom(tag uint16) unsafe.Pointer {
	if auxTag(s.aux) != tag {
		panic("wrong custom tag")
	}
	return s.ptr
}

//
// Accessors with conversion
//

func (s Scmer) IsNil() bool { return auxTag(s.aux) == tagNil }

func (s Scmer) IsBool() bool { return auxTag(s.aux) == tagBool }

func (s Scmer) IsInt() bool { return auxTag(s.aux) == tagInt }

func (s Scmer) IsFloat() bool { return auxTag(s.aux) == tagFloat }

func (s Scmer) IsString() bool { return auxTag(s.aux) == tagString }

func (s Scmer) IsSymbol() bool { return auxTag(s.aux) == tagSymbol }

func (s Scmer) SymbolEquals(name string) bool {
	return auxTag(s.aux) == tagSymbol && s.String() == name
}

func (s Scmer) IsSlice() bool { return auxTag(s.aux) == tagSlice }

func (s Scmer) IsVector() bool { return auxTag(s.aux) == tagVector }

func (s Scmer) Bool() bool {
	switch auxTag(s.aux) {
	case tagNil:
		return false
	case tagBool:
		return uintptr(s.ptr) != 0
	case tagInt:
		return int64(uintptr(s.ptr)) != 0
	case tagFloat:
		return math.Float64frombits(uint64(uintptr(s.ptr))) != 0.0
	case tagString, tagSymbol:
		return s.String() != ""
	case tagSlice:
		return len(s.Slice()) > 0
	case tagVector:
		return len(s.Vector()) > 0
	case tagAny:
		v := s.Any()
		switch vv := v.(type) {
		case nil:
			return false
		case bool:
			return vv
		case int:
			return vv != 0
		case int64:
			return vv != 0
		case float64:
			return vv != 0
		case string:
			return vv != ""
		case []Scmer:
			return len(vv) > 0
		default:
			return v != nil
		}
	case tagFunc:
		return true
	default:
		if auxTag(s.aux) == tagAny {
			return true
		}
		return s.ptr != nil
	}
}

func (s Scmer) Int() int64 {
	switch auxTag(s.aux) {
	case tagInt:
		return int64(uintptr(s.ptr))
	case tagFloat:
		return int64(math.Float64frombits(uint64(uintptr(s.ptr))))
	case tagString, tagSymbol:
		v, err := strconv.ParseInt(s.String(), 10, 64)
		if err != nil {
			return 0
		}
		return v
	case tagBool:
		if uintptr(s.ptr) != 0 {
			return 1
		}
		return 0
	case tagAny:
		switch v := s.Any().(type) {
		case int:
			return int64(v)
		case int8:
			return int64(v)
		case int16:
			return int64(v)
		case int32:
			return int64(v)
		case int64:
			return v
		case uint:
			return int64(v)
		case uint8:
			return int64(v)
		case uint16:
			return int64(v)
		case uint32:
			return int64(v)
		case uint64:
			return int64(v)
		case float32:
			return int64(v)
		case float64:
			return int64(v)
		case string:
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				return i
			}
		}
	}
	return 0
}

func (s Scmer) Float() float64 {
	switch auxTag(s.aux) {
	case tagFloat:
		return math.Float64frombits(uint64(uintptr(s.ptr)))
	case tagInt:
		return float64(int64(uintptr(s.ptr)))
	case tagString, tagSymbol:
		v, err := strconv.ParseFloat(s.String(), 64)
		if err != nil {
			return 0.0
		}
		return v
	case tagBool:
		if uintptr(s.ptr) != 0 {
			return 1.0
		}
		return 0.0
	case tagAny:
		switch v := s.Any().(type) {
		case int:
			return float64(v)
		case int8:
			return float64(v)
		case int16:
			return float64(v)
		case int32:
			return float64(v)
		case int64:
			return float64(v)
		case uint:
			return float64(v)
		case uint8:
			return float64(v)
		case uint16:
			return float64(v)
		case uint32:
			return float64(v)
		case uint64:
			return float64(v)
		case float32:
			return float64(v)
		case float64:
			return v
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}
	}
	return 0.0
}

func (s Scmer) String() string {
	switch auxTag(s.aux) {
	case tagString, tagSymbol:
		hdr := [2]uintptr{uintptr(s.ptr), uintptr(auxVal(s.aux))}
		return *(*string)(unsafe.Pointer(&hdr))
	case tagInt:
		return strconv.FormatInt(int64(uintptr(s.ptr)), 10)
	case tagFloat:
		return strconv.FormatFloat(math.Float64frombits(uint64(uintptr(s.ptr))), 'g', -1, 64)
	case tagBool:
		if uintptr(s.ptr) != 0 {
			return "true"
		}
		return "false"
	case tagNil:
		return "nil"
	case tagFunc:
		return "[func]"
	default:
		if auxTag(s.aux) == tagAny {
			return fmt.Sprintf("%v", *(*any)(s.ptr))
		}
		return fmt.Sprintf("<custom %d>", auxTag(s.aux))
	}
}

func (s Scmer) Slice() []Scmer {
	if auxTag(s.aux) != tagSlice {
		panic("not slice")
	}
	hdr := [3]uintptr{uintptr(s.ptr), uintptr(auxVal(s.aux)), uintptr(auxVal(s.aux))}
	return *(*[]Scmer)(unsafe.Pointer(&hdr))
}

func (s Scmer) Vector() []float64 {
	if auxTag(s.aux) != tagVector {
		panic("not vector")
	}
	hdr := [3]uintptr{uintptr(s.ptr), uintptr(auxVal(s.aux)), uintptr(auxVal(s.aux))}
	return *(*[]float64)(unsafe.Pointer(&hdr))
}

func (s Scmer) Func() func(...Scmer) Scmer {
	if auxTag(s.aux) != tagFunc {
		panic("not function")
	}
	return *(*func(...Scmer) Scmer)(s.ptr)
}

func (s Scmer) IsProc() bool { return auxTag(s.aux) == tagProc }

func (s Scmer) Proc() *Proc {
	if auxTag(s.aux) != tagProc {
		panic("not proc")
	}
	return (*Proc)(s.ptr)
}

func (s Scmer) IsNthLocalVar() bool { return auxTag(s.aux) == tagNthLocalVar }

func (s Scmer) NthLocalVar() NthLocalVar {
	if auxTag(s.aux) != tagNthLocalVar {
		panic("not nth local var")
	}
	return NthLocalVar(uintptr(s.ptr))
}

// Symbol returns the Scheme symbol value as Go string.
func (s Scmer) Symbol() string {
	if auxTag(s.aux) != tagSymbol {
		panic("not symbol")
	}
	return s.String()
}

// Any unwraps the Scmer into a Go value for legacy code.
func (s Scmer) Any() any {
	switch auxTag(s.aux) {
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
	case tagProc:
		return s.Proc()
	case tagNthLocalVar:
		return s.NthLocalVar()
	case tagAny:
		return *(*any)(s.ptr)
	default:
		panic(fmt.Sprintf("unknown tag %d in Any", auxTag(s.aux)))
	}
}

// Compatibility helpers for legacy code paths.
func ToBool(v Scmer) bool     { return v.Bool() }
func ToInt(v Scmer) int       { return int(v.Int()) }
func ToFloat(v Scmer) float64 { return v.Float() }
