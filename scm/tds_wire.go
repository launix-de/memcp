/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"golang.org/x/text/encoding/htmlindex"
	"io"
	"math"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// This is an original, bounded implementation of the MS-TDS 7.4 wire format.
// Packet framing follows the TDS 7.4 specification.
var tdsCP1252, _ = htmlindex.Get("cp1252")

const tdsMaxMessage = 16 << 20
const tdsMaxValue = 64 << 20
const tdsPacketSize = 4096

type tdsMessage struct {
	kind, status byte
	data         []byte
}

func readTDSMessage(r io.Reader) (tdsMessage, error) {
	var message tdsMessage
	for packet := 0; ; packet++ {
		var header [8]byte
		if _, err := io.ReadFull(r, header[:]); err != nil {
			return message, err
		}
		n := int(binary.BigEndian.Uint16(header[2:4])) - 8
		if n < 0 || n+len(message.data) > tdsMaxMessage || (n == 0 && header[1]&1 == 0) || packet >= 65536 {
			return message, fmt.Errorf("invalid or oversized TDS packet")
		}
		if packet == 0 {
			message.kind, message.status = header[0], header[1]
		} else if message.kind != header[0] {
			return message, fmt.Errorf("TDS packet type changed within a message")
		}
		start := len(message.data)
		message.data = append(message.data, make([]byte, n)...)
		if _, err := io.ReadFull(r, message.data[start:]); err != nil {
			return message, err
		}
		if header[1]&1 != 0 {
			return message, nil
		}
	}
}

func tdsWriteAll(w io.Writer, data []byte) error {
	for len(data) != 0 {
		n, err := w.Write(data)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}

// Only the query goroutine owns a response writer. Rows are streamed into
// packet-sized buffers, not accumulated into a complete result set.
type tdsResponse struct {
	w          io.Writer
	packet     []byte
	kind, id   byte
	packetSize int
}

func (w *tdsResponse) Write(data []byte) (int, error) {
	n := len(data)
	for len(data) > 0 {
		room := w.packetSize - 8 - len(w.packet)
		if room > len(data) {
			room = len(data)
		}
		w.packet = append(w.packet, data[:room]...)
		data = data[room:]
		if len(w.packet) == w.packetSize-8 {
			if err := w.flush(false); err != nil {
				return n - len(data), err
			}
		}
	}
	return n, nil
}

func (w *tdsResponse) flush(final bool) error {
	var header [8]byte
	header[0], header[6] = w.kind, w.id
	w.id++
	if final {
		header[1] = 1
	}
	binary.BigEndian.PutUint16(header[2:4], uint16(8+len(w.packet)))
	if err := tdsWriteAll(w.w, header[:]); err != nil {
		return err
	}
	if err := tdsWriteAll(w.w, w.packet); err != nil {
		return err
	}
	w.packet = w.packet[:0]
	return nil
}

func writeTDSMessage(w io.Writer, kind byte, data []byte) error {
	response := tdsResponse{w: w, kind: kind, id: 1, packetSize: tdsPacketSize}
	if _, err := response.Write(data); err != nil {
		return err
	}
	return response.flush(true)
}

func tdsUTF16(s string) []byte {
	units := utf16.Encode([]rune(s))
	data := make([]byte, 2*len(units))
	for i, unit := range units {
		binary.LittleEndian.PutUint16(data[2*i:], unit)
	}
	return data
}

func tdsText(data []byte) (string, error) {
	if len(data)%2 != 0 {
		return "", fmt.Errorf("odd UTF-16 length in TDS value")
	}
	units := make([]uint16, len(data)/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(data[2*i:])
	}
	for i := 0; i < len(units); i++ {
		if units[i] >= 0xd800 && units[i] <= 0xdbff {
			if i+1 == len(units) || units[i+1] < 0xdc00 || units[i+1] > 0xdfff {
				return "", fmt.Errorf("invalid UTF-16 surrogate in TDS value")
			}
			i++
		} else if units[i] >= 0xdc00 && units[i] <= 0xdfff {
			return "", fmt.Errorf("invalid UTF-16 surrogate in TDS value")
		}
	}
	return string(utf16.Decode(units)), nil
}

func tdsU16(b *bytes.Buffer, n uint16) { _ = binary.Write(b, binary.LittleEndian, n) }
func tdsU32(b *bytes.Buffer, n uint32) { _ = binary.Write(b, binary.LittleEndian, n) }
func tdsU64(b *bytes.Buffer, n uint64) { _ = binary.Write(b, binary.LittleEndian, n) }

func tdsBText(b *bytes.Buffer, s string) {
	data := tdsUTF16(s)
	if len(data)/2 > 255 {
		panic("TDS identifier exceeds 255 UTF-16 code units")
	}
	b.WriteByte(byte(len(data) / 2))
	b.Write(data)
}

func tdsToken(b *bytes.Buffer, token byte, data []byte) {
	b.WriteByte(token)
	tdsU16(b, uint16(len(data)))
	b.Write(data)
}

func tdsDone(b *bytes.Buffer, token byte, status uint16, rows uint64) {
	b.WriteByte(token)
	tdsU16(b, status)
	tdsU16(b, 0)
	tdsU64(b, rows)
}

func tdsError(b *bytes.Buffer, number uint32, message string) {
	// Bound the protocol message; detailed panics belong in the server log.
	if len(message) > 2048 {
		message = message[:2048]
	}
	var payload bytes.Buffer
	tdsU32(&payload, number)
	payload.Write([]byte{1, 16}) // state, severity
	text := tdsUTF16(message)
	tdsU16(&payload, uint16(len(text)/2))
	payload.Write(text)
	tdsBText(&payload, "MemCP")
	tdsBText(&payload, "")
	tdsU32(&payload, 1)
	tdsToken(b, 0xaa, payload.Bytes())
}

func tdsDatabase(b *bytes.Buffer, current, previous string) {
	var payload bytes.Buffer
	payload.WriteByte(1)
	tdsBText(&payload, current)
	tdsBText(&payload, previous)
	tdsToken(b, 0xe3, payload.Bytes())
}

func tdsTransaction(b *bytes.Buffer, kind byte, id uint64) {
	var payload bytes.Buffer
	payload.WriteByte(kind)
	if kind == 8 {
		payload.WriteByte(8)
		tdsU64(&payload, id)
		payload.WriteByte(0)
	} else {
		payload.Write([]byte{0, 8})
		tdsU64(&payload, id)
	}
	tdsToken(b, 0xe3, payload.Bytes())
}

type tdsDecoder struct {
	data       []byte
	err        error
	descriptor Scmer
}

func (d *tdsDecoder) take(n int) []byte {
	if d.err != nil {
		return nil
	}
	if n < 0 || n > len(d.data) {
		d.err = io.ErrUnexpectedEOF
		return nil
	}
	value := d.data[:n]
	d.data = d.data[n:]
	return value
}
func (d *tdsDecoder) u8() byte {
	v := d.take(1)
	if len(v) == 1 {
		return v[0]
	}
	return 0
}
func (d *tdsDecoder) u16() uint16 {
	v := d.take(2)
	if len(v) == 2 {
		return binary.LittleEndian.Uint16(v)
	}
	return 0
}
func (d *tdsDecoder) u32() uint32 {
	v := d.take(4)
	if len(v) == 4 {
		return binary.LittleEndian.Uint32(v)
	}
	return 0
}
func (d *tdsDecoder) u64() uint64 {
	v := d.take(8)
	if len(v) == 8 {
		return binary.LittleEndian.Uint64(v)
	}
	return 0
}
func (d *tdsDecoder) text(n int) string {
	value, err := tdsText(d.take(2 * n))
	if err != nil {
		d.err = err
	}
	return value
}

func tdsStripHeaders(data []byte) ([]byte, error) {
	if len(data) < 4 {
		return nil, io.ErrUnexpectedEOF
	}
	n := int(binary.LittleEndian.Uint32(data))
	if n < 4 || n > len(data) {
		return nil, fmt.Errorf("invalid TDS ALL_HEADERS length")
	}
	seen := map[uint16]bool{}
	for offset := 4; offset < n; {
		if n-offset < 6 {
			return nil, fmt.Errorf("truncated TDS stream header")
		}
		size := int(binary.LittleEndian.Uint32(data[offset:]))
		kind := binary.LittleEndian.Uint16(data[offset+4:])
		if size < 6 || size > n-offset || seen[kind] {
			return nil, fmt.Errorf("invalid or duplicate TDS stream header")
		}
		seen[kind] = true
		switch kind {
		case 2:
			if size != 18 || binary.LittleEndian.Uint32(data[offset+14:]) != 1 {
				return nil, fmt.Errorf("MARS transaction requests are unsupported")
			}
		case 3:
			if size != 26 {
				return nil, fmt.Errorf("invalid TDS trace activity header")
			}
		default:
			return nil, fmt.Errorf("unsupported TDS stream header")
		}
		offset += size
	}
	if !seen[2] {
		return nil, fmt.Errorf("TDS transaction descriptor is required")
	}
	return data[n:], nil
}

// RPC values are decoded as data, never interpolated into SQL text. Unsupported
// types/flags fail explicitly, so a caller cannot accidentally lose a parameter.
func (d *tdsDecoder) parameter() Scmer {
	kind := d.u8()
	d.descriptor = NewSlice([]Scmer{NewString("wire_kind"), NewInt(int64(kind))})
	switch kind {
	case 0x23, 0x63, 0x22: // legacy TEXT, NTEXT and IMAGE input parameters
		maxSize := d.u32()
		d.descriptor = NewSlice(append(d.descriptor.Slice(), NewString("wire_size"), NewInt(int64(maxSize))))
		var collation []byte
		if kind != 0x22 {
			collation = d.take(5)
		}
		n := d.u32()
		if n == math.MaxUint32 {
			return NewNil()
		}
		if n > tdsMaxValue || n > maxSize {
			d.err = fmt.Errorf("invalid or oversized legacy TDS value")
			return NewNil()
		}
		raw := d.take(int(n))
		if d.err != nil {
			return NewNil()
		}
		if kind == 0x22 {
			return NewString(string(raw))
		}
		if kind == 0x63 {
			text, err := tdsText(raw)
			if err != nil {
				d.err = err
			}
			return NewString(text)
		}
		text, err := tdsDecodeCharacter(raw, collation)
		if err != nil {
			d.err = err
		}
		return NewString(text)
	case 0x28, 0x29, 0x2a, 0x2b, 0x6f:
		return d.temporal(kind)
	case 0x6a, 0x6c, 0x6e:
		return d.numeric(kind)
	case 0x1f: // NULLTYPE (untyped NULL parameter)
		return NewNil()
	case 0x26, 0x68, 0x6d: // INTN, BITN, FLOATN
		maxSize := d.u8()
		d.descriptor = NewSlice(append(d.descriptor.Slice(), NewString("wire_size"), NewInt(int64(maxSize))))
		n := int(d.u8())
		validSize := (kind == 0x26 && (maxSize == 1 || maxSize == 2 || maxSize == 4 || maxSize == 8)) || (kind == 0x68 && maxSize == 1) || (kind == 0x6d && (maxSize == 4 || maxSize == 8))
		if !validSize || (n != 0 && n != int(maxSize)) {
			d.err = fmt.Errorf("invalid TDS scalar type length")
			return NewNil()
		}
		if n == 0 {
			return NewNil()
		}
		value := d.take(n)
		if d.err != nil {
			return NewNil()
		}
		if kind == 0x68 && n == 1 {
			return NewBool(value[0] != 0)
		}
		if kind == 0x6d {
			if n == 4 {
				return NewFloat(float64(math.Float32frombits(binary.LittleEndian.Uint32(value))))
			}
			if n == 8 {
				return NewFloat(math.Float64frombits(binary.LittleEndian.Uint64(value)))
			}
		} else if kind == 0x26 {
			switch n {
			case 1:
				return NewInt(int64(value[0]))
			case 2:
				return NewInt(int64(int16(binary.LittleEndian.Uint16(value))))
			case 4:
				return NewInt(int64(int32(binary.LittleEndian.Uint32(value))))
			case 8:
				return NewInt(int64(binary.LittleEndian.Uint64(value)))
			}
		}
		d.err = fmt.Errorf("unsupported TDS scalar length")
	case 0xe7, 0xef, 0xa7, 0xaf, 0xa5, 0xad: // text and binary
		maxSize := d.u16()
		d.descriptor = NewSlice(append(d.descriptor.Slice(), NewString("wire_size"), NewInt(int64(maxSize))))
		var collation []byte
		if kind != 0xa5 && kind != 0xad {
			collation = d.take(5)
		} // collation
		var value []byte
		if maxSize == 0xffff {
			total := d.u64()
			if total == math.MaxUint64 {
				return NewNil()
			}
			if total != math.MaxUint64-1 && total > tdsMaxValue {
				d.err = fmt.Errorf("TDS value too large")
				return NewNil()
			}
			for d.err == nil {
				n := int(d.u32())
				if n == 0 {
					break
				}
				if n > tdsMaxValue-len(value) {
					d.err = fmt.Errorf("TDS value too large")
					break
				}
				value = append(value, d.take(n)...)
			}
			if total != math.MaxUint64-1 && uint64(len(value)) != total {
				d.err = fmt.Errorf("TDS PLP length mismatch")
			}
		} else {
			n := d.u16()
			if n == 0xffff {
				return NewNil()
			}
			if n > maxSize {
				d.err = fmt.Errorf("invalid TDS string length")
				return NewNil()
			}
			value = d.take(int(n))
		}
		if kind == 0xa5 || kind == 0xad {
			return NewString(string(value))
		}
		if kind == 0xe7 || kind == 0xef {
			text, err := tdsText(value)
			if err != nil {
				d.err = err
			}
			return NewString(text)
		}
		text, err := tdsDecodeCharacter(value, collation)
		if err != nil {
			d.err = err
		}
		return NewString(text)
	default:
		d.err = fmt.Errorf("unsupported TDS parameter type 0x%02x", kind)
	}
	return NewNil()
}

type tdsRPCParameter struct {
	name  string
	flags byte
	value Scmer
}

type tdsRPC struct {
	procedure       uint16
	arguments       []tdsRPCParameter
	wireDescriptors []Scmer
	metadata        string
	options         uint16
}

func decodeTDSRPC(data []byte) (tdsRPC, error) {
	var request tdsRPC
	data, err := tdsStripHeaders(data)
	if err != nil {
		return request, err
	}
	d := tdsDecoder{data: data}
	n := d.u16()
	if n == 0xffff {
		request.procedure = d.u16()
	} else {
		name := d.text(int(n))
		switch strings.ToLower(name) {
		case "sp_cursoropen":
			request.procedure = 2
		case "sp_cursorfetch":
			request.procedure = 7
		case "sp_cursoroption":
			request.procedure = 8
		case "sp_cursorclose":
			request.procedure = 9
		case "sp_executesql":
			request.procedure = 10
		case "sp_prepare":
			request.procedure = 11
		case "sp_execute":
			request.procedure = 12
		case "sp_prepexec":
			request.procedure = 13
		case "sp_unprepare":
			request.procedure = 15
		default:
			request.metadata = name
		}
	}
	switch request.procedure {
	case 2, 7, 8, 9, 10, 11, 12, 13, 15:
	default:
		if request.metadata == "" {
			return request, fmt.Errorf("unsupported TDS RPC ID %d", request.procedure)
		}
	}
	request.options = d.u16()
	if request.options != 0 && !(request.procedure == 7 && request.options & ^uint16(6) == 0 || request.procedure == 9 && request.options == 2) {
		return request, fmt.Errorf("unsupported TDS RPC options")
	}
	for len(d.data) != 0 && d.err == nil {
		if len(request.arguments) >= 2103 {
			return request, fmt.Errorf("too many TDS RPC parameters")
		}
		name, flags := d.text(int(d.u8())), d.u8()
		if flags & ^byte(1) != 0 {
			return request, fmt.Errorf("TDS default/encrypted parameters are unsupported")
		}
		value := d.parameter()
		request.arguments = append(request.arguments, tdsRPCParameter{name, flags, value})
		request.wireDescriptors = append(request.wireDescriptors, d.descriptor)
	}
	return request, d.err
}

func tdsReturnHandle(b *bytes.Buffer, name string, handle int32) {
	tdsReturnInteger(b, name, 0, handle)
}

func tdsReturnInteger(b *bytes.Buffer, name string, ordinal uint16, value int32) {
	b.WriteByte(0xac)
	tdsU16(b, ordinal)
	tdsBText(b, name)
	b.WriteByte(1)              // output parameter
	tdsU32(b, 0)                // user type
	tdsU16(b, 0)                // flags
	b.Write([]byte{0x26, 4, 4}) // INTN metadata and value length
	tdsU32(b, uint32(value))
}

// Character parameters are decoded according to their declared wire collation.
// Unknown code pages reject high bytes rather than guessing and corrupting data.
func tdsDecodeCharacter(value, collation []byte) (string, error) {
	if len(collation) != 5 {
		return "", fmt.Errorf("invalid TDS collation")
	}
	info := binary.LittleEndian.Uint32(collation)
	if info&(1<<26) != 0 {
		if !utf8.Valid(value) {
			return "", fmt.Errorf("invalid UTF-8 TDS parameter")
		}
		return string(value), nil
	}
	locale := info & 0xfffff
	if collation[4] == 52 || (collation[4] == 0 && (locale&0x3ff == 7 || locale&0x3ff == 9)) {
		for _, b := range value {
			if b == 0x81 || b == 0x8d || b == 0x8f || b == 0x90 || b == 0x9d {
				return "", fmt.Errorf("undefined CP1252 TDS byte")
			}
		}
		decoded, err := tdsCP1252.NewDecoder().Bytes(value)
		return string(decoded), err
	}
	for _, b := range value {
		if b >= 128 {
			return "", fmt.Errorf("unsupported non-ASCII TDS code page; use NVARCHAR")
		}
	}
	return string(value), nil
}
