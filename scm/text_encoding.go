/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import "strings"
import "unicode/utf8"
import "unicode/utf16"
import "golang.org/x/text/encoding/charmap"
import "golang.org/x/text/encoding/ianaindex"

func singleByteEncoding(name string) *charmap.Charmap {
	switch strings.ToLower(name) {
	case "cp1252", "windows-1252":
		return charmap.Windows1252
	case "latin1", "iso-8859-1":
		return charmap.ISO8859_1
	}
	codec, err := ianaindex.IANA.Encoding(name)
	if err != nil {
		panic("unsupported single-byte encoding: " + name)
	}
	if mapping, ok := codec.(*charmap.Charmap); ok {
		return mapping
	}
	panic("unsupported single-byte encoding: " + name)
}

func encodeSingleByte(text, name string, limit int) string {
	if limit < -1 {
		panic("single-byte prefix limit must be nonnegative or -1")
	}
	if !utf8.ValidString(text) {
		panic("single_byte_encode requires valid UTF-8 text")
	}
	codec := singleByteEncoding(name)
	var output strings.Builder
	capacity := len(text)
	if limit >= 0 && limit < capacity {
		capacity = limit
	}
	output.Grow(capacity)
	for _, character := range text {
		if output.Len() == limit {
			break
		}
		value, ok := codec.EncodeRune(character)
		if !ok || codec.DecodeByte(value) == utf8.RuneError {
			panic("character is not representable in single-byte encoding: " + name)
		}
		output.WriteByte(value)
	}
	return output.String()
}

func decodeSingleByte(bytes, name string) Scmer {
	codec := singleByteEncoding(name)
	var output strings.Builder
	output.Grow(len(bytes))
	for index := 0; index < len(bytes); index++ {
		character := codec.DecodeByte(bytes[index])
		if character == utf8.RuneError {
			return NewNil()
		}
		output.WriteRune(character)
	}
	return NewString(output.String())
}

func initTextEncoding() {
	declare := func(name, description, result string, parameters []*TypeDescriptor, fn func(...Scmer) Scmer) {
		Declare(&Globalenv, &Declaration{
			Name: name, Fn: fn,
			Type: &TypeDescriptor{Kind: "func", Description: description, Params: parameters,
				Return: &TypeDescriptor{Kind: result}, Const: true,
				JITEmit: func(ctx *JITContext, source []Scmer, args []JITValueDesc, output JITValueDesc) JITValueDesc {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declarations[name], source, args, output)
				},
				JITVirtualArgs: true, JITInlineCost: 65535},
		})
	}
	declare("utf16le_encode", "encodes valid UTF-8 text as UTF-16 little-endian bytes without a byte-order mark", "string",
		[]*TypeDescriptor{{Kind: "string", Label: "text"}}, func(args ...Scmer) Scmer {
			if args[0].IsNil() {
				return NewNil()
			}
			text := args[0].String()
			if !utf8.ValidString(text) {
				panic("utf16le_encode requires valid UTF-8 text")
			}
			var output strings.Builder
			output.Grow(int(2 * utf16Length(text)))
			writeUnit := func(unit rune) {
				output.WriteByte(byte(unit))
				output.WriteByte(byte(unit >> 8))
			}
			for _, character := range text {
				if character > 0xffff {
					high, low := utf16.EncodeRune(character)
					writeUnit(high)
					writeUnit(low)
				} else {
					writeUnit(character)
				}
			}
			return NewString(output.String())
		})
	declare("utf8_prefix", "returns a whole-scalar UTF-8 prefix bounded by bytes", "string",
		[]*TypeDescriptor{{Kind: "string", Label: "text"}, {Kind: "number", Label: "limit"}}, func(args ...Scmer) Scmer {
			if args[0].IsNil() || args[1].IsNil() {
				return NewNil()
			}
			text, limit := args[0].String(), ToInt(args[1])
			if limit < 0 || !utf8.ValidString(text) {
				panic("UTF-8 prefix requires valid text and a nonnegative byte limit")
			}
			if limit >= len(text) {
				return NewString(text)
			}
			end := 0
			for offset, character := range text {
				next := offset + utf8.RuneLen(character)
				if next > limit {
					break
				}
				end = next
			}
			return NewString(text[:end])
		})
	textAndEncoding := []*TypeDescriptor{{Kind: "string", Label: "text"}, {Kind: "string", Label: "encoding"}}
	declare("single_byte_encode", "strictly encodes UTF-8 text into a named single-byte charset, optionally limiting encoded prefix bytes (-1 means unlimited)", "string",
		[]*TypeDescriptor{{Kind: "string", Label: "text"}, {Kind: "string", Label: "encoding"}, {Kind: "number", Label: "limit", Optional: true}},
		func(args ...Scmer) Scmer {
			if args[0].IsNil() {
				return NewNil()
			}
			limit := -1
			if len(args) > 2 {
				limit = ToInt(args[2])
			}
			return NewString(encodeSingleByte(args[0].String(), args[1].String(), limit))
		})
	declare("single_byte_decode", "decodes a named single-byte charset into UTF-8 text; undefined bytes return nil", "string", textAndEncoding,
		func(args ...Scmer) Scmer {
			if args[0].IsNil() {
				return NewNil()
			}
			return decodeSingleByte(args[0].String(), args[1].String())
		})
	declare("byte_char", "constructs one raw byte for an integer in [0,255]; out-of-range values return nil", "string",
		[]*TypeDescriptor{{Kind: "number", Label: "code"}}, func(args ...Scmer) Scmer {
			if args[0].IsNil() {
				return NewNil()
			}
			code := ToInt(args[0])
			if code < 0 || code > 255 {
				return NewNil()
			}
			return NewString(string([]byte{byte(code)}))
		})
	declare("byte_value", "returns the raw byte at a zero-based offset, or nil outside the byte string", "int",
		[]*TypeDescriptor{{Kind: "string", Label: "bytes"}, {Kind: "number", Label: "offset"}}, func(args ...Scmer) Scmer {
			if args[0].IsNil() || args[1].IsNil() {
				return NewNil()
			}
			value, offset := args[0].String(), ToInt(args[1])
			if offset < 0 || offset >= len(value) {
				return NewNil()
			}
			return NewInt(int64(value[offset]))
		})
}
