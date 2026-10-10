/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strings"
	"sync"
)

type tdsColumn struct {
	name       string
	kind       byte
	size       int
	scale      byte
	precision  byte
	declared   bool
	nullable   bool
	hidden     bool
	updatable  bool
	computed   bool
	key        bool
	rowversion bool
	identity   bool
	encode     Scmer
	source     [4]string // immutable database, schema, table and base-column declaration
}

// The result sink owns its metadata and a bounded discovery buffer. The mutex
// serializes parallel scan callbacks; no shard/catalog locks are acquired here.
// Declared SQL descriptors preserve types for empty/all-NULL results. Callers
// without descriptors retain bounded prefix inference for compatibility.
type tdsResult struct {
	mu             sync.Mutex
	w              io.Writer
	columns        []tdsColumn
	rows           [][]Scmer
	bytes          int
	count          uint64
	published      bool
	snapshotLimit  int // positive: collect immutable logical rows without publishing
	browseMetadata bool
}

func (r *tdsResult) fields(a ...Scmer) Scmer {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.columns != nil || r.published || (len(a) != 1 && len(a) != 2) {
		panic("result fields must be declared once before result rows")
	}
	r.columns = make([]tdsColumn, len(a[0].Slice()))
	for i, title := range a[0].Slice() {
		r.columns[i] = tdsColumn{name: title.String(), nullable: true}
		if len(a) == 2 {
			if len(a[1].Slice()) != len(r.columns) {
				panic("result descriptor count mismatch")
			}
			r.columns[i].describe(a[1].Slice()[i])
			if tdsDescriptorValue(a[1].Slice()[i], "browse").Bool() {
				r.browseMetadata = true
			}
		}
	}
	return NewBool(true)
}

func tdsValueKind(value Scmer) byte {
	switch value.GetTag() {
	case tagNil:
		return 0
	case tagInt:
		return 0x26
	case tagFloat:
		return 0x6d
	case tagBool:
		return 0x68
	case tagDate:
		panic("TDS date requires an explicit wire descriptor and codec")
	default:
		return 0xe7
	}
}

func (r *tdsResult) row(a ...Scmer) Scmer {
	r.mu.Lock()
	defer r.mu.Unlock()
	item := a[0].Slice()
	if len(item)%2 != 0 {
		panic("result row must contain name/value pairs")
	}
	if r.columns == nil {
		r.columns = make([]tdsColumn, len(item)/2)
		for i := range r.columns {
			r.columns[i] = tdsColumn{name: item[2*i].String(), nullable: true}
		}
	}
	values := make([]Scmer, len(r.columns))
	// Positional matches also preserve duplicate projection titles. Sparse
	// association rows (including NULL extension) are matched by name otherwise.
	positional := len(item) == 2*len(r.columns)
	if positional {
		for i, col := range r.columns {
			if item[2*i].String() != col.name {
				positional = false
				break
			}
		}
	}
	if positional {
		for i := range values {
			values[i] = item[2*i+1]
		}
	} else {
		for i := 0; i < len(item); i += 2 {
			found := false
			for j, col := range r.columns {
				if col.name == item[i].String() {
					values[j] = item[i+1]
					found = true
				}
			}
			if !found {
				panic("result row contains an undeclared column")
			}
		}
	}
	if r.snapshotLimit > 0 {
		for i, value := range values {
			switch value.GetTag() {
			case tagNil, tagInt, tagFloat, tagBool, tagDate:
			default:
				if !value.IsString() {
					panic("cursor snapshot requires frontend-bound immutable scalars")
				}
				values[i] = NewString(strings.Clone(value.String()))
			}
		}
		if len(r.rows) >= tdsCursorRowsLimit {
			panic("TDS cursor row limit exceeded")
		}
		r.bytes += int(ComputeSize(NewSlice(values))) + 64
	}
	r.count++
	if !r.published {
		for i, value := range values {
			if r.snapshotLimit == 0 {
				r.bytes += 16
				if value.IsString() {
					r.bytes += len(value.String())
				}
			}
			if value.IsNil() || r.columns[i].declared {
				continue
			}
			kind := tdsValueKind(value)
			if r.columns[i].kind == 0 {
				r.columns[i].kind = kind
			} else if r.columns[i].kind != kind {
				if (r.columns[i].kind == 0x26 && kind == 0x6d) || (r.columns[i].kind == 0x6d && kind == 0x26) {
					r.columns[i].kind = 0x6d
				} else {
					r.columns[i].kind = 0xe7
				}
			}
		}
		r.rows = append(r.rows, values)
		if r.snapshotLimit > 0 {
			if r.bytes > r.snapshotLimit {
				panic("TDS cursor memory limit exceeded")
			}
			return NewBool(true)
		}
		if len(r.rows) < 128 && r.bytes < 1<<20 {
			return NewBool(true)
		}
		r.publish()
		return NewBool(true)
	}
	r.emit(values)
	return NewBool(true)
}

func (r *tdsResult) write(data []byte) {
	if err := tdsWriteAll(r.w, data); err != nil {
		panic(err)
	}
}

func (r *tdsResult) publish() {
	if r.published || r.columns == nil {
		return
	}
	if len(r.columns) > 65534 {
		panic("too many TDS result columns")
	}
	var metadata bytes.Buffer
	metadata.WriteByte(0x81)
	tdsU16(&metadata, uint16(len(r.columns)))
	for i := range r.columns {
		col := &r.columns[i]
		if col.kind == 0x2a && !col.declared {
			col.scale = 7
		}
		if col.kind == 0 {
			col.kind = 0xe7
		}
		userType := uint32(0)
		if col.rowversion {
			userType = 0x50
		}
		tdsU32(&metadata, userType)
		flags := uint16(0)
		if col.nullable {
			flags = 1
		}
		if col.hidden {
			flags |= 0x2000
		}
		if col.updatable && col.source[2] != "" && !col.hidden && !col.computed && !col.rowversion {
			flags |= 0x04
		}
		if col.computed {
			flags |= 0x20
		}
		if col.identity {
			flags |= 0x10
		}
		tdsU16(&metadata, flags)
		metadata.WriteByte(col.kind)
		switch col.kind {
		case 0x26, 0x6d:
			size := col.size
			if size == 0 {
				size = 8
			}
			metadata.WriteByte(byte(size))
		case 0x68:
			metadata.WriteByte(1)
		case 0x6a, 0x6c:
			metadata.Write([]byte{byte(col.size), col.precision, col.scale})
		case 0x6e:
			metadata.WriteByte(byte(col.size))
		case 0x6f:
			metadata.WriteByte(byte(col.size))
		case 0x28:
		case 0x29, 0x2a, 0x2b:
			metadata.WriteByte(col.scale)
		case 0xa5, 0xad:
			size := col.size
			if size == 0 {
				size = 0xffff
			}
			tdsU16(&metadata, uint16(size))
		case 0xe7, 0xef, 0xa7, 0xaf:
			size := col.size
			if size == 0 {
				size = 0xffff
			}
			tdsU16(&metadata, uint16(size)) // NVARCHAR(MAX), PLP values
			metadata.Write([]byte{0x09, 0x04, 0xd0, 0x00, 0x34})
		}
		tdsBText(&metadata, col.name)
	}
	if r.browseMetadata {
		if len(r.columns) > 255 {
			panic("too many browse result columns")
		}
		var tableNames, info bytes.Buffer
		tables := make(map[[3]string]byte)
		for _, column := range r.columns {
			if column.source[2] == "" {
				continue
			}
			table := [3]string{column.source[0], column.source[1], column.source[2]}
			if tables[table] != 0 {
				continue
			}
			if len(tables) == 255 {
				panic("too many browse source tables")
			}
			tables[table] = byte(len(tables) + 1)
			tableNames.WriteByte(3)
			for _, part := range []string{table[0], table[1], table[2]} {
				encoded := tdsUTF16(part)
				if len(encoded)/2 > 65535 {
					panic("browse source name exceeds protocol bounds")
				}
				tdsU16(&tableNames, uint16(len(encoded)/2))
				tableNames.Write(encoded)
			}
		}
		for i, column := range r.columns {
			flags := byte(4) // expression: no proven base-column provenance
			table := tables[[3]string{column.source[0], column.source[1], column.source[2]}]
			if table != 0 {
				flags = 0
				if column.key {
					flags |= 0x08
				}
				if column.name != column.source[3] {
					flags |= 0x20
				}
			}
			if column.hidden {
				flags |= 0x10
			}
			info.Write([]byte{byte(i + 1), table, flags})
			if flags&0x20 != 0 {
				tdsBText(&info, column.source[3])
			}
		}
		if tableNames.Len() > 65535 || info.Len() > 65535 {
			panic("browse metadata exceeds protocol bounds")
		}
		if tableNames.Len() != 0 {
			tdsToken(&metadata, 0xa4, tableNames.Bytes())
		}
		tdsToken(&metadata, 0xa5, info.Bytes())
	}
	r.write(metadata.Bytes())
	r.published = true
	for _, row := range r.rows {
		r.emit(row)
	}
	r.rows = nil
}

func (r *tdsResult) emit(row []Scmer) {
	// Validate and encode before publishing ROW: a conversion error must not
	// leave a half-written row which would swallow the following ERROR token.
	// A static cursor retains the original frontend values and codecs. Encoding
	// must never replace its snapshot values with transient wire tuples.
	row = append([]Scmer(nil), row...)
	encoded := make([][]byte, len(row))
	for i, value := range row {
		col := r.columns[i]
		kind := col.kind
		if !value.IsNil() && !col.encode.IsNil() {
			value = Apply(col.encode, value)
			row[i] = value
		}
		if !value.IsNil() && kind == 0x26 {
			value = tdsResultInteger(value)
			row[i] = value
		}
		if !value.IsNil() && kind == 0x68 {
			value = tdsResultBit(value)
			row[i] = value
		}
		if kind == 0x6a || kind == 0x6c || kind == 0x6e {
			encoded[i] = tdsEncodeNumeric(col, value)
			continue
		}
		if kind == 0x28 || kind == 0x29 || kind == 0x2a || kind == 0x2b || kind == 0x6f {
			encoded[i] = tdsEncodeTemporal(col, value)
			continue
		}
		if value.IsNil() {
			continue
		}
		switch kind {
		case 0xe7, 0xef, 0xa7, 0xaf, 0xa5, 0xad:
			encoded[i] = tdsStringBytes(col, value)
			n := len(encoded[i])
			if n > tdsMaxValue || (col.size > 0 && col.size != 0xffff && n > col.size) {
				panic("TDS declared result size exceeded")
			}
		case 0x26:
			if !value.IsInt() {
				panic("TDS result type changed after metadata publication")
			}
			n := value.Int()
			if (col.size == 1 && (n < 0 || n > 255)) || (col.size == 2 && (n < math.MinInt16 || n > math.MaxInt16)) || (col.size == 4 && (n < math.MinInt32 || n > math.MaxInt32)) {
				panic("TDS declared integer overflow")
			}
		case 0x6d:
			if !value.IsInt() && !value.IsFloat() {
				panic("TDS result type changed after metadata publication")
			}
			if value.IsInt() && (value.Int() > 1<<53 || value.Int() < -(1<<53)) {
				panic("integer cannot be represented exactly as TDS float")
			}
		case 0x68:
			if !value.IsBool() {
				panic("TDS result type changed after metadata publication")
			}
		default:
			panic("unsupported TDS result type")
		}
	}
	r.write([]byte{0xd1})
	var scalar [9]byte
	for i, value := range row {
		col := r.columns[i]
		kind := col.kind
		if kind == 0x6a || kind == 0x6c || kind == 0x6e || kind == 0x28 || kind == 0x29 || kind == 0x2a || kind == 0x2b || kind == 0x6f {
			r.write(encoded[i])
			continue
		}
		if kind == 0xe7 || kind == 0xef || kind == 0xa7 || kind == 0xaf || kind == 0xa5 || kind == 0xad {
			if col.size != 0 && col.size != 0xffff {
				if value.IsNil() {
					r.write([]byte{255, 255})
					continue
				}
				data := encoded[i]
				if len(data) > col.size {
					panic("TDS declared result size exceeded")
				}
				binary.LittleEndian.PutUint16(scalar[:2], uint16(len(data)))
				r.write(scalar[:2])
				r.write(data)
				continue
			}
			if value.IsNil() {
				binary.LittleEndian.PutUint64(scalar[:8], math.MaxUint64)
				r.write(scalar[:8])
				continue
			}
			text := encoded[i]
			binary.LittleEndian.PutUint64(scalar[:8], uint64(len(text)))
			r.write(scalar[:8])
			if len(text) != 0 {
				binary.LittleEndian.PutUint32(scalar[:4], uint32(len(text)))
				r.write(scalar[:4])
				r.write(text)
			}
			r.write([]byte{0, 0, 0, 0})
			continue
		}
		if value.IsNil() {
			r.write([]byte{0})
			continue
		}
		switch kind {
		case 0x26:
			if !value.IsInt() {
				panic("TDS result type changed after metadata publication")
			}
			size := col.size
			if size == 0 {
				size = 8
			}
			n := value.Int()
			if (size == 1 && (n < 0 || n > 255)) || (size == 2 && (n < math.MinInt16 || n > math.MaxInt16)) || (size == 4 && (n < math.MinInt32 || n > math.MaxInt32)) {
				panic("TDS declared integer overflow")
			}
			scalar[0] = byte(size)
			binary.LittleEndian.PutUint64(scalar[1:], uint64(n))
			r.write(scalar[:size+1])
		case 0x6d:
			if !value.IsInt() && !value.IsFloat() {
				panic("TDS result type changed after metadata publication")
			}
			if value.IsInt() && (value.Int() > 1<<53 || value.Int() < -(1<<53)) {
				panic("integer cannot be represented exactly as TDS float")
			}
			size := col.size
			if size == 4 {
				scalar[0] = 4
				binary.LittleEndian.PutUint32(scalar[1:], math.Float32bits(float32(value.Float())))
				r.write(scalar[:5])
			} else {
				scalar[0] = 8
				binary.LittleEndian.PutUint64(scalar[1:], math.Float64bits(value.Float()))
				r.write(scalar[:])
			}
		case 0x68:
			if !value.IsBool() {
				panic("TDS result type changed after metadata publication")
			}
			scalar[0], scalar[1] = 1, 0
			if value.Bool() {
				scalar[1] = 1
			}
			r.write(scalar[:2])
		default:
			panic(fmt.Sprintf("unsupported TDS result type %x", kind))
		}
	}
}

func tdsDescriptorValue(descriptor Scmer, key string) Scmer {
	slice, dict := asAssoc(descriptor, "TDS result descriptor")
	if dict != nil {
		v, _ := dict.Get(NewString(key))
		return v
	}
	for i := 0; i < len(slice); i += 2 {
		if slice[i].String() == key {
			return slice[i+1]
		}
	}
	return NewNil()
}
func (col *tdsColumn) describe(descriptor Scmer) {
	if source := tdsDescriptorValue(descriptor, "source"); !source.IsNil() {
		parts := source.Slice()
		if len(parts) != 4 {
			panic("invalid TDS base-column provenance")
		}
		for i, part := range parts {
			col.source[i] = part.String()
		}
	}
	col.updatable = tdsDescriptorValue(descriptor, "updatable").Bool() &&
		!tdsDescriptorValue(descriptor, "rowversion").Bool() &&
		!tdsDescriptorValue(descriptor, "identity").Bool()
	col.computed = tdsDescriptorValue(descriptor, "computed").Bool()
	col.key = col.source[2] != "" && tdsDescriptorValue(descriptor, "key").Bool()
	col.rowversion = tdsDescriptorValue(descriptor, "rowversion").Bool()
	wireKind := tdsDescriptorValue(descriptor, "wire_kind")
	if wireKind.IsNil() {
		panic("TDS result descriptor requires a frontend-bound wire kind")
	}
	if !wireKind.IsInt() || wireKind.Int() < 0 || wireKind.Int() > 255 {
		panic("invalid TDS wire kind")
	}
	col.kind = byte(wireKind.Int())
	col.declared = true
	col.identity = tdsDescriptorValue(descriptor, "identity").Bool()
	if nullable := tdsDescriptorValue(descriptor, "nullable"); !nullable.IsNil() {
		col.nullable = nullable.Bool()
	}
	if size := tdsDescriptorValue(descriptor, "wire_size"); !size.IsNil() {
		if !size.IsInt() || size.Int() < 0 || size.Int() > 65535 {
			panic("invalid TDS wire size")
		}
		col.size = int(size.Int())
	}
	if encode := tdsDescriptorValue(descriptor, "encode"); !encode.IsNil() {
		if !encode.IsProc() && !encode.IsNativeFunc() && !encode.IsJIT() && encode.GetTag() != tagClosure {
			panic("TDS wire codec requires a callable recipe")
		}
		col.encode = encode
	}
	if precision := tdsDescriptorValue(descriptor, "precision"); !precision.IsNil() {
		if !precision.IsInt() || precision.Int() < 0 || precision.Int() > 255 {
			panic("invalid TDS precision field")
		}
		col.precision = byte(precision.Int())
	}
	if scale := tdsDescriptorValue(descriptor, "scale"); !scale.IsNil() {
		if !scale.IsInt() || scale.Int() < 0 || scale.Int() > 255 {
			panic("invalid TDS scale field")
		}
		col.scale = byte(scale.Int())
	}
	switch col.kind {
	case 0x26:
		if col.size != 1 && col.size != 2 && col.size != 4 && col.size != 8 {
			panic("invalid TDS integer width")
		}
	case 0x68:
		if col.size != 1 {
			panic("invalid TDS bit width")
		}
	case 0x6d:
		if col.size != 4 && col.size != 8 {
			panic("invalid TDS float width")
		}
	case 0x6a, 0x6c:
		if col.precision < 1 || col.precision > 38 || col.scale > col.precision || col.size != tdsDecimalSize(int(col.precision)) {
			panic("invalid TDS coefficient metadata")
		}
	case 0x6e, 0x6f:
		if col.size != 4 && col.size != 8 {
			panic("invalid TDS high-low or legacy clock width")
		}
	case 0x28:
	case 0x29, 0x2a, 0x2b:
		if col.scale > 7 {
			panic("invalid TDS clock metadata")
		}
	case 0xe7, 0xef, 0xa7, 0xaf, 0xa5, 0xad:
	default:
		panic("unsupported TDS wire type")
	}
}
func tdsStringBytes(col tdsColumn, value Scmer) []byte {
	if !value.IsString() {
		if col.kind == 0xa5 || col.kind == 0xad {
			panic("TDS binary result must be a byte string")
		}
		if col.declared {
			panic("TDS text result requires a frontend-bound string payload")
		}
	}
	if col.kind == 0xe7 || col.kind == 0xef {
		return tdsUTF16(value.String())
	}
	data := []byte(value.String())
	if col.kind == 0xa7 || col.kind == 0xaf {
		encoded, err := tdsCP1252.NewEncoder().Bytes(data)
		if err != nil {
			panic("TDS character result is not representable in CP1252; use NVARCHAR")
		}
		return encoded
	}
	return data
}

func tdsResultInteger(value Scmer) Scmer {
	if value.IsInt() {
		return value
	}
	if value.IsFloat() {
		n := value.Float()
		if !math.IsNaN(n) && !math.IsInf(n, 0) && math.Trunc(n) == n && n >= -9223372036854775808.0 && n < 9223372036854775808.0 {
			return NewInt(int64(n))
		}
	}
	panic("TDS declared integer result is not an exact signed integer")
}
func tdsResultBit(value Scmer) Scmer {
	if value.IsBool() {
		return value
	}
	if value.IsInt() {
		if value.Int() == 0 || value.Int() == 1 {
			return NewBool(value.Int() == 1)
		}
	}
	if value.IsFloat() {
		if value.Float() == 0 || value.Float() == 1 {
			return NewBool(value.Float() == 1)
		}
	}
	panic("TDS declared bit result must be zero or one")
}
