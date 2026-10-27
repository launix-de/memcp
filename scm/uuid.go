/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "fmt"
import "time"
import "unsafe"
import "encoding/binary"
import "github.com/google/uuid"

// UUIDs use the existing immutable 16-byte CString carrier, shared with stored
// UUID columns. The GC-visible pointer owns the allocation; no mutable generator
// state, dictionary lease or SQL type metadata is retained by a value.
func NewUUID(id uuid.UUID) Scmer { return NewCString(&id[0], 6, 0, 36) }

func (s Scmer) IsUUID() bool {
	return s.IsCString() && (auxVal(s.aux)>>CStringFormatShift == 6 || auxVal(s.aux)>>CStringFormatShift == 7)
}

func UUIDValue(value Scmer) (uuid.UUID, error) {
	if value.IsUUID() {
		return *(*uuid.UUID)(unsafe.Pointer(value.ptr)), nil
	}
	if !value.IsString() {
		return uuid.Nil, fmt.Errorf("UUID requires text or a UUID value")
	}
	return uuid.Parse(value.String())
}

func ParseUUID(value Scmer) Scmer {
	if value.IsNil() {
		return value
	}
	if value.IsUUID() && auxVal(value.aux)>>CStringFormatShift == 6 {
		return value
	}
	id, err := UUIDValue(value)
	if err != nil {
		panic("invalid UUID: " + err.Error())
	}
	return NewUUID(id)
}

func GenerateUUID(version int) Scmer {
	var id uuid.UUID
	var err error
	switch version {
	case 1:
		id, err = uuid.NewUUID()
	case 4:
		id, err = uuid.NewRandom()
	case 7:
		id, err = uuid.NewV7()
	default:
		panic("unsupported UUID generator version")
	}
	if err != nil {
		panic(err)
	}
	return NewUUID(id)
}

// writeUUIDText formats one packed value into caller-owned scratch. Storage,
// hashing and output all keep the immutable binary source alive through its Scmer.
func writeUUIDText(dst []byte, value Scmer) {
	alphabet := "0123456789abcdef"
	if auxVal(value.aux)>>CStringFormatShift == 7 {
		alphabet = "0123456789ABCDEF"
	}
	source := unsafe.Slice(value.ptr, 16)
	dst[8], dst[13], dst[18], dst[23] = '-', '-', '-', '-'
	positions := [16]uint8{0, 2, 4, 6, 9, 11, 14, 16, 19, 21, 24, 26, 28, 30, 32, 34}
	for i, pos := range positions {
		b := source[i]
		dst[pos], dst[pos+1] = alphabet[b>>4], alphabet[b&15]
	}
}

// UUIDTimestampMillis exposes the embedded RFC timestamp, not commit time.
func UUIDTimestampMillis(value Scmer) Scmer {
	if value.IsNil() {
		return value
	}
	id, err := UUIDValue(value)
	if err != nil {
		panic(err)
	}
	if id.Variant() != uuid.RFC4122 {
		return NewNil()
	}
	switch id.Version() {
	case 7:
		return NewInt(int64(binary.BigEndian.Uint64(id[:8]) >> 16))
	case 1, 6:
		seconds, nanos := id.Time().UnixTime()
		return NewInt(seconds*1000 + nanos/1e6)
	default:
		return NewNil()
	}
}

func uuidSwap(id uuid.UUID, inverse bool) uuid.UUID {
	if inverse {
		return uuid.UUID{id[4], id[5], id[6], id[7], id[2], id[3], id[0], id[1], id[8], id[9], id[10], id[11], id[12], id[13], id[14], id[15]}
	}
	return uuid.UUID{id[6], id[7], id[4], id[5], id[0], id[1], id[2], id[3], id[8], id[9], id[10], id[11], id[12], id[13], id[14], id[15]}
}

func uuidSwapFlag(a []Scmer) bool {
	if len(a) < 2 {
		return false
	}
	if !a[1].IsInt() || a[1].Int() < 0 || a[1].Int() > 1 {
		panic("UUID swap flag must be 0 or 1")
	}
	return a[1].Int() == 1
}

func init_uuid() {
	declare := func(name string, fn func(...Scmer) Scmer, params []*TypeDescriptor, kind string, constant bool) {
		Declare(&Globalenv, &Declaration{Name: name, Fn: fn, Type: &TypeDescriptor{Kind: "func", Description: "UUID " + name, Params: params, Return: &TypeDescriptor{Kind: kind}, Const: constant}})
	}
	value := []*TypeDescriptor{{Kind: "any", Label: "value"}}
	binaryParams := []*TypeDescriptor{{Kind: "any", Label: "value"}, {Kind: "int", Label: "swap", Optional: true}}
	// Native serialization identifies builtins by entry point. Capturing the
	// version in one loop closure would give all three generators the same
	// entry point and make persisted expressions/hashes select arbitrary names.
	declare("uuid_v1", func(a ...Scmer) Scmer { return GenerateUUID(1) }, nil, "string", false)
	declare("uuid_v4", func(a ...Scmer) Scmer { return GenerateUUID(4) }, nil, "string", false)
	declare("uuid_v7", func(a ...Scmer) Scmer { return GenerateUUID(7) }, nil, "string", false)
	declare("uuid_parse", func(a ...Scmer) Scmer { return ParseUUID(a[0]) }, value, "string", true)
	declare("uuid_valid", func(a ...Scmer) Scmer {
		if a[0].IsNil() {
			return NewNil()
		}
		_, err := UUIDValue(a[0])
		return NewBool(err == nil)
	}, value, "bool", true)
	declare("uuid_format", func(a ...Scmer) Scmer {
		if a[0].IsNil() {
			return a[0]
		}
		return NewString(ParseUUID(a[0]).String())
	}, value, "string", true)
	declare("uuid_version", func(a ...Scmer) Scmer {
		if a[0].IsNil() {
			return a[0]
		}
		id, err := UUIDValue(a[0])
		if err != nil {
			panic(err)
		}
		return NewInt(int64(id.Version()))
	}, value, "int", true)
	declare("uuid_extract_version", func(a ...Scmer) Scmer {
		if a[0].IsNil() {
			return a[0]
		}
		id, err := UUIDValue(a[0])
		if err != nil {
			panic(err)
		}
		if id.Variant() != uuid.RFC4122 {
			return NewNil()
		}
		return NewInt(int64(id.Version()))
	}, value, "int", true)
	declare("uuid_variant", func(a ...Scmer) Scmer {
		if a[0].IsNil() {
			return a[0]
		}
		id, err := UUIDValue(a[0])
		if err != nil {
			panic(err)
		}
		return NewString(id.Variant().String())
	}, value, "string", true)
	declare("uuid_timestamp_ms", func(a ...Scmer) Scmer { return UUIDTimestampMillis(a[0]) }, value, "int", true)
	declare("uuid_timestamp", func(a ...Scmer) Scmer {
		ms := UUIDTimestampMillis(a[0])
		if ms.IsNil() {
			return ms
		}
		return NewString(time.UnixMilli(ms.Int()).UTC().Format("2006-01-02 15:04:05.000+00:00"))
	}, value, "string", true)
	declare("uuid_to_bin", func(a ...Scmer) Scmer {
		if a[0].IsNil() || len(a) > 1 && a[1].IsNil() {
			return NewNil()
		}
		id, err := UUIDValue(a[0])
		if err != nil {
			panic(err)
		}
		if uuidSwapFlag(a) {
			id = uuidSwap(id, false)
		}
		return NewString(string(id[:]))
	}, binaryParams, "string", true)
	declare("uuid_from_bin", func(a ...Scmer) Scmer {
		if a[0].IsNil() || len(a) > 1 && a[1].IsNil() {
			return NewNil()
		}
		if !a[0].IsString() {
			panic("binary UUID requires exactly 16 bytes")
		}
		text := a[0].String()
		if len(text) != 16 {
			panic("binary UUID requires exactly 16 bytes")
		}
		var id uuid.UUID
		copy(id[:], text)
		if uuidSwapFlag(a) {
			id = uuidSwap(id, true)
		}
		return NewUUID(id)
	}, binaryParams, "string", true)
	// SQL Server converts the first 36 characters of character input.
	declare("tsql_uuid", func(a ...Scmer) Scmer {
		v := a[0]
		if v.IsNil() || v.IsUUID() {
			return ParseUUID(v)
		}
		text := v.String()
		if len(text) > 36 {
			text = text[:36]
		}
		return ParseUUID(NewString(text))
	}, value, "string", true)
}
