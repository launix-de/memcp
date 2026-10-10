/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"encoding/binary"
	"fmt"
	"time"
)

// Epoch constants describe the wire layout; frontend codecs select the units.
var tdsDateEpoch = time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
var tdsLegacyEpoch = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)

func tdsTimeLength(scale byte) int {
	if scale <= 2 {
		return 3
	}
	if scale <= 4 {
		return 4
	}
	return 5
}
func tdsTimeFactor(scale byte) uint64 {
	factor := uint64(1)
	for i := byte(0); i < scale; i++ {
		factor *= 10
	}
	return factor
}
func tdsLittleValue(data []byte) uint64 {
	var value uint64
	for i, b := range data {
		value |= uint64(b) << (8 * i)
	}
	return value
}

// Temporal protocol payloads are tuples of day ordinal, wire-clock units and
// original offset minutes. They are transient adapter values, never Scmer tags.
func tdsTemporalPayload(days, clock, offset int64) Scmer {
	return NewSlice([]Scmer{NewInt(days), NewInt(clock), NewInt(offset)})
}
func tdsTemporalParts(value Scmer) (days, clock, offset int64) {
	if !value.IsSlice() || len(value.Slice()) != 3 {
		panic("TDS clock requires a bound three-integer payload")
	}
	parts := value.Slice()
	for _, part := range parts {
		if !part.IsInt() {
			panic("TDS clock payload must contain integers")
		}
	}
	return parts[0].Int(), parts[1].Int(), parts[2].Int()
}
func tdsEncodeTemporal(col tdsColumn, value Scmer) []byte {
	if value.IsNil() {
		return []byte{0}
	}
	days, clock, offset := tdsTemporalParts(value)
	if clock < 0 {
		panic("negative TDS clock field")
	}
	var width, extra int
	switch col.kind {
	case 0x28:
		width = 0
		extra = 3
		if clock != 0 || offset != 0 || days < 0 || days > 0xffffff {
			panic("invalid TDS date payload")
		}
	case 0x29, 0x2a, 0x2b:
		if col.scale > 7 {
			panic("invalid TDS clock scale")
		}

		width = tdsTimeLength(col.scale)
		if col.kind != 0x29 {
			extra = 3
			if days < 0 || days > 0xffffff {
				panic("TDS day field exceeds three bytes")
			}
		} else if days != 0 {
			panic("unexpected TDS time day field")
		}
		if uint64(clock) >= uint64(1)<<(8*width) {
			panic("TDS clock exceeds wire width")
		}
		if col.kind == 0x2b {
			extra += 2
			if offset < -32768 || offset > 32767 {
				panic("invalid TDS offset field")
			}
		} else if offset != 0 {
			panic("unexpected TDS offset field")
		}
	case 0x6f:
		if offset != 0 {
			panic("unexpected TDS legacy offset")
		}
		legacyDays := days
		result := make([]byte, 1+col.size)
		result[0] = byte(col.size)
		switch col.size {
		case 4:
			if legacyDays < 0 || legacyDays > 65535 || clock > 65535 {
				panic("invalid TDS short clock payload")
			}
			binary.LittleEndian.PutUint16(result[1:], uint16(legacyDays))
			binary.LittleEndian.PutUint16(result[3:], uint16(clock))
		case 8:
			if legacyDays < -2147483648 || legacyDays > 2147483647 || clock > 4294967295 {
				panic("invalid TDS long clock payload")
			}
			binary.LittleEndian.PutUint32(result[1:], uint32(int32(legacyDays)))
			binary.LittleEndian.PutUint32(result[5:], uint32(clock))
		default:
			panic("invalid TDS legacy clock width")
		}
		return result
	default:
		panic("unsupported TDS clock wire type")
	}
	result := make([]byte, 1+width+extra)
	result[0] = byte(width + extra)
	for i := 0; i < width; i++ {
		result[1+i] = byte(uint64(clock) >> (8 * i))
	}
	if col.kind != 0x29 {
		for i := 0; i < 3; i++ {
			result[1+width+i] = byte(uint64(days) >> (8 * i))
		}
	}
	if col.kind == 0x2b {
		binary.LittleEndian.PutUint16(result[1+width+3:], uint16(int16(offset)))
	}
	return result
}
func (d *tdsDecoder) temporal(kind byte) (value Scmer) {
	defer func() {
		if failure := recover(); failure != nil {
			d.err = fmt.Errorf("invalid TDS clock payload: %v", failure)
			value = NewNil()
		}
	}()
	size, scale := 0, byte(0)
	switch kind {
	case 0x6f:
		size = int(d.u8())
		if size != 4 && size != 8 {
			d.err = fmt.Errorf("invalid TDS clock metadata")
			return NewNil()
		}
	case 0x28:
		size = 3
	case 0x29, 0x2a, 0x2b:
		scale = d.u8()
		if scale > 7 {
			d.err = fmt.Errorf("invalid TDS clock scale")
			return NewNil()
		}
		size = tdsTimeLength(scale)
		if kind != 0x29 {
			size += 3
		}
		if kind == 0x2b {
			size += 2
		}
	default:
		d.err = fmt.Errorf("unsupported TDS clock wire type")
		return NewNil()
	}
	d.descriptor = NewSlice([]Scmer{NewString("wire_kind"), NewInt(int64(kind)), NewString("wire_size"), NewInt(int64(size)), NewString("scale"), NewInt(int64(scale))})
	n := int(d.u8())
	if n == 0 {
		return NewNil()
	}
	if n != size {
		d.err = fmt.Errorf("invalid TDS clock length")
		return NewNil()
	}
	data := d.take(n)
	if d.err != nil {
		return NewNil()
	}
	days, clock, offset := int64(0), int64(0), int64(0)
	if kind == 0x6f {
		if size == 4 {
			days = int64(binary.LittleEndian.Uint16(data))
			clock = int64(binary.LittleEndian.Uint16(data[2:]))
		} else {
			days = int64(int32(binary.LittleEndian.Uint32(data)))
			clock = int64(binary.LittleEndian.Uint32(data[4:]))
		}
	} else if kind == 0x28 {
		days = int64(tdsLittleValue(data))
	} else {
		width := tdsTimeLength(scale)
		clock = int64(tdsLittleValue(data[:width]))
		if kind != 0x29 {
			days = int64(tdsLittleValue(data[width : width+3]))
		}
		if kind == 0x2b {
			offset = int64(int16(binary.LittleEndian.Uint16(data[width+3:])))
		}
	}
	value = tdsTemporalPayload(days, clock, offset)
	// Validate framing and representable field bounds with the same pure packer.
	_ = tdsEncodeTemporal(tdsColumn{kind: kind, size: size, scale: scale}, value)
	return value
}
