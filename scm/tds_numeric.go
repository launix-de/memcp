/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"encoding/binary"
	"fmt"
	"math/big"
)

func tdsDecimalSize(precision int) int {
	switch {
	case precision >= 1 && precision <= 9:
		return 5
	case precision >= 10 && precision <= 19:
		return 9
	case precision >= 20 && precision <= 28:
		return 13
	case precision >= 29 && precision <= 38:
		return 17
	default:
		panic("invalid TDS coefficient width")
	}
}

// Wire decoding produces an unscaled coefficient. Frontend callbacks receive
// the complete descriptor, including for NULL, and choose its interpretation.
func (d *tdsDecoder) numeric(kind byte) (value Scmer) {
	defer func() {
		if failure := recover(); failure != nil {
			d.err = fmt.Errorf("invalid TDS coefficient: %v", failure)
			value = NewNil()
		}
	}()
	maxSize := int(d.u8())
	d.descriptor = NewSlice([]Scmer{NewString("wire_kind"), NewInt(int64(kind)), NewString("wire_size"), NewInt(int64(maxSize))})
	if kind == 0x6a || kind == 0x6c {
		precision, scale := int(d.u8()), int(d.u8())
		if precision < 1 || precision > 38 || scale < 0 || scale > precision || maxSize != tdsDecimalSize(precision) {
			d.err = fmt.Errorf("invalid TDS coefficient metadata")
			return NewNil()
		}
		d.descriptor = NewSlice(append(d.descriptor.Slice(), NewString("precision"), NewInt(int64(precision)), NewString("scale"), NewInt(int64(scale))))
		n := int(d.u8())
		if n == 0 {
			return NewNil()
		}
		if (n != 5 && n != 9 && n != 13 && n != 17) || n > maxSize {
			d.err = fmt.Errorf("invalid TDS coefficient length")
			return NewNil()
		}
		sign := d.u8()
		if sign > 1 {
			d.err = fmt.Errorf("invalid TDS coefficient sign")
			return NewNil()
		}
		magnitude := d.take(n - 1)
		if d.err != nil {
			return NewNil()
		}
		reversed := make([]byte, len(magnitude))
		for i, b := range magnitude {
			reversed[len(magnitude)-1-i] = b
		}
		integer := new(big.Int).SetBytes(reversed)
		if sign == 0 {
			integer.Neg(integer)
		}
		return encodeCoefficient(integer)
	}
	n := int(d.u8())
	if (maxSize != 4 && maxSize != 8) || (n != 0 && n != maxSize) {
		d.err = fmt.Errorf("invalid TDS high-low integer length")
		return NewNil()
	}
	if n == 0 {
		return NewNil()
	}
	raw := d.take(n)
	if d.err != nil {
		return NewNil()
	}
	if n == 4 {
		return NewInt(int64(int32(binary.LittleEndian.Uint32(raw))))
	}
	return NewInt(int64(binary.LittleEndian.Uint32(raw))<<32 | int64(binary.LittleEndian.Uint32(raw[4:])))
}

// The caller supplies a coefficient already converted to the descriptor's
// scale. No cast, inferred type, or precision propagation occurs here.
func tdsEncodeNumeric(column tdsColumn, value Scmer) []byte {
	if value.IsNil() {
		return []byte{0}
	}
	if column.kind == 0x6e {
		if !value.IsInt() || (column.size != 4 && column.size != 8) {
			panic("TDS high-low integer requires an integer payload")
		}
		integer := value.Int()
		if column.size == 4 && (integer < -2147483648 || integer > 2147483647) {
			panic("TDS signed 32-bit payload overflow")
		}
		encoded := make([]byte, 1+column.size)
		encoded[0] = byte(column.size)
		if column.size == 4 {
			binary.LittleEndian.PutUint32(encoded[1:], uint32(integer))
		} else {
			binary.LittleEndian.PutUint32(encoded[1:], uint32(uint64(integer)>>32))
			binary.LittleEndian.PutUint32(encoded[5:], uint32(integer))
		}
		return encoded
	}
	integer := decodeCoefficient(value)
	negative := integer.Sign() < 0
	integer.Abs(integer)
	size := tdsDecimalSize(int(column.precision))
	if integer.BitLen() > (size-1)*8 {
		panic("TDS coefficient exceeds wire width")
	}
	encoded := make([]byte, size+1)
	encoded[0], encoded[1] = byte(size), 1
	if negative {
		encoded[1] = 0
	}
	magnitude := integer.Bytes()
	for i, b := range magnitude {
		encoded[2+len(magnitude)-1-i] = b
	}
	return encoded
}
