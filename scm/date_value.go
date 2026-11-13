/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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

import "math"
import "time"
import "unsafe"
import "strings"

// NewDateNanoWithZone retains the existing second/zone payload and allocates
// immutable subsecond storage only when the value has a fractional clock.
func NewDateNanoWithZone(seconds int64, nanosecond, zone int) Scmer {
	if seconds < -(1<<44) || seconds >= 1<<44 || nanosecond < 0 || nanosecond >= 1000000000 || zone < 0 || zone > 2047 {
		panic("date parts out of range")
	}
	value := NewDateWithZone(seconds, zone)
	if nanosecond != 0 {
		fraction := uint32(nanosecond)
		value.ptr = (*byte)(unsafe.Pointer(&fraction))
	}
	return value
}

func (value Scmer) DateNanosecond() int {
	if value.GetTag() != tagDate || value.ptr == nil {
		return 0
	}
	return int(*(*uint32)(unsafe.Pointer(value.ptr)))
}

func (value Scmer) DateZone() int { return TagDateDecodeZone(auxVal(value.aux)) }

// ParseDateValue preserves all supplied nanoseconds. The language-independent
// calendar parser rejects extra digits instead of silently discarding them.
func ParseDateValue(text string) (Scmer, bool) {
	if isMySQLZeroDate(text) {
		return NewDate(mysqlZeroDateUnix), true
	}
	if clock := strings.LastIndexByte(text, ':'); clock >= 0 {
		if dot := strings.IndexByte(text[clock:], '.'); dot >= 0 {
			fraction := text[clock+dot+1:]
			if len(fraction) > 9 {
				return NewNil(), false
			}
		}
	}
	for _, layout := range allowedDateFormats {
		if value, err := time.Parse(layout, text); err == nil {
			return NewDateNanoWithZone(value.Unix(), value.Nanosecond(), 0), true
		}
	}
	return NewNil(), false
}

func compareDates(left, right Scmer) int {
	if left.Int() < right.Int() {
		return -1
	}
	if left.Int() > right.Int() {
		return 1
	}
	if left.DateNanosecond() < right.DateNanosecond() {
		return -1
	}
	if left.DateNanosecond() > right.DateNanosecond() {
		return 1
	}
	return 0
}

func initDateParts() {
	Declare(&Globalenv, &Declaration{Name: "date_from_parts", Fn: func(args ...Scmer) Scmer {
		for _, part := range args {
			if !part.IsInt() && !(part.IsFloat() && part.Float() == math.Trunc(part.Float())) {
				panic("date parts require whole numbers")
			}
		}
		return NewDateNanoWithZone(args[0].Int(), int(args[1].Int()), int(args[2].Int()))
	}, Type: &TypeDescriptor{Kind: "func", Const: true, Params: []*TypeDescriptor{{Kind: "number"}, {Kind: "number"}, {Kind: "number"}}, Return: &TypeDescriptor{Kind: "date"}}})
	Declare(&Globalenv, &Declaration{Name: "date_parts", Fn: func(args ...Scmer) Scmer {
		if args[0].IsNil() {
			return NewNil()
		}
		value, ok := toTime(args[0])
		if !ok {
			panic("invalid date value")
		}
		zone := 0
		if args[0].IsDate() {
			zone = args[0].DateZone()
		}
		return NewSlice([]Scmer{NewInt(value.Unix()), NewInt(int64(value.Nanosecond())), NewInt(int64(zone))})
	}, Type: &TypeDescriptor{Kind: "func", Const: true, Params: []*TypeDescriptor{{Kind: "any"}}, Return: &TypeDescriptor{Kind: "list"}}})
}
