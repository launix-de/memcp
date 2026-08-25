/*
Copyright (C) 2026  Carl-Philip Hänsch

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
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
	"unsafe"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

// tagBSON keeps the BSON type in the low byte of auxVal and the payload length
// in the following 32 bits. The pointer refers directly to the BSON value
// payload, so nested documents can be exposed without constructing a Go DOM.
const (
	bsonTypeBits      = 8
	bsonLenBits       = 32
	bsonLenMask       = uint64(1<<bsonLenBits) - 1
	bsonFlagArrayAgg  = uint64(1) << 40
	bsonFlagObjectAgg = uint64(1) << 41
)

var bsonEmptyPayload byte

const bsonEscapedKeyPrefix = "\x01memcp-json-key:"

func encodeBSONKey(key string) string {
	if strings.ContainsRune(key, '\x00') || strings.HasPrefix(key, bsonEscapedKeyPrefix) {
		return bsonEscapedKeyPrefix + base64.RawStdEncoding.EncodeToString([]byte(key))
	}
	return key
}

func decodeBSONKey(key string) string {
	if !strings.HasPrefix(key, bsonEscapedKeyPrefix) {
		return key
	}
	decoded, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(key, bsonEscapedKeyPrefix))
	if err != nil {
		panic("corrupt escaped BSON JSON key")
	}
	return string(decoded)
}

// NewBSONValue constructs an immutable Scmer view over a raw BSON value.
// payload must use the value representation expected by bson.RawValue, not a
// complete BSON element containing a field name.
func NewBSONValue(typ bson.Type, payload []byte) Scmer {
	return newBSONValueFlags(typ, payload, 0)
}

// NewBSONRaw is the storage-facing constructor for a persisted BSON value.
func NewBSONRaw(typ byte, payload []byte) Scmer {
	return NewBSONValue(bson.Type(typ), payload)
}

// BSONRaw exposes the BSON type and immutable payload bytes for persistence.
func (s Scmer) BSONRaw() (byte, []byte) {
	typ, payload := bsonTypeAndBytes(s)
	return byte(typ), payload
}

func newBSONValueFlags(typ bson.Type, payload []byte, flags uint64) Scmer {
	if !typ.IsValid() {
		panic(fmt.Sprintf("invalid BSON type 0x%x", byte(typ)))
	}
	if uint64(len(payload)) > bsonLenMask {
		panic("BSON value exceeds the 32-bit BSON size limit")
	}
	ptr := &bsonEmptyPayload
	if len(payload) > 0 {
		ptr = unsafe.SliceData(payload)
	}
	packed := uint64(typ) | uint64(len(payload))<<bsonTypeBits | flags
	return Scmer{ptr: ptr, aux: makeAux(tagBSON, packed)}
}

func bsonHasFlag(value Scmer, flag uint64) bool {
	return value.IsBSON() && auxVal(value.aux)&flag != 0
}

// NewBSONFromJSON validates JSON text and converts it into BSON's typed value
// representation. Only JSON types are admitted; BSON extension types never
// enter SQL JSON values through this function.
func NewBSONFromJSON(text string) (Scmer, error) {
	value, err := parseBSONJSONText(text)
	if err != nil {
		return NewNil(), err
	}
	return NewBSONValue(value.Type, value.Value), nil
}

type bsonJSONPair struct {
	key   string
	value bson.RawValue
}

// parseBSONJSONText converts ordinary JSON directly into BSON. Container
// values are completed before being appended to their parent, so the parser
// never materializes a second map/slice representation of the document.
func parseBSONJSONText(text string) (bson.RawValue, error) {
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	value, err := parseBSONJSONToken(decoder)
	if err != nil {
		return bson.RawValue{}, err
	}
	if _, err := decoder.Token(); err == nil {
		return bson.RawValue{}, fmt.Errorf("multiple JSON values")
	} else if err != io.EOF {
		return bson.RawValue{}, err
	}
	return value, nil
}

func parseBSONJSONToken(decoder *json.Decoder) (bson.RawValue, error) {
	token, err := decoder.Token()
	if err != nil {
		return bson.RawValue{}, err
	}
	switch token := token.(type) {
	case json.Delim:
		switch token {
		case '{':
			pairs := make([]bsonJSONPair, 0, 8)
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return bson.RawValue{}, err
				}
				key, ok := keyToken.(string)
				if !ok {
					return bson.RawValue{}, fmt.Errorf("JSON object key is not a string")
				}
				value, err := parseBSONJSONToken(decoder)
				if err != nil {
					return bson.RawValue{}, err
				}
				pairs = append(pairs, bsonJSONPair{key: key, value: value})
			}
			if end, err := decoder.Token(); err != nil || end != json.Delim('}') {
				return bson.RawValue{}, fmt.Errorf("unterminated JSON object")
			}
			// MySQL JSON objects have one value per key and deterministic key
			// order. Stable sorting followed by replacement keeps the last
			// duplicate, matching encoding/json's previous map semantics.
			sort.SliceStable(pairs, func(i, j int) bool { return pairs[i].key < pairs[j].key })
			index, document := bsoncore.AppendDocumentStart(nil)
			for i := 0; i < len(pairs); {
				j := i + 1
				for j < len(pairs) && pairs[j].key == pairs[i].key {
					j++
				}
				pair := pairs[j-1]
				document = bsoncore.AppendValueElement(document, encodeBSONKey(pair.key), bsoncore.Value{
					Type: bsoncore.Type(pair.value.Type), Data: pair.value.Value,
				})
				i = j
			}
			document, err = bsoncore.AppendDocumentEnd(document, index)
			return bson.RawValue{Type: bson.TypeEmbeddedDocument, Value: document}, err
		case '[':
			index, array := bsoncore.AppendArrayStart(nil)
			for i := 0; decoder.More(); i++ {
				value, err := parseBSONJSONToken(decoder)
				if err != nil {
					return bson.RawValue{}, err
				}
				array = bsoncore.AppendValueElement(array, strconv.Itoa(i), bsoncore.Value{
					Type: bsoncore.Type(value.Type), Data: value.Value,
				})
			}
			if end, err := decoder.Token(); err != nil || end != json.Delim(']') {
				return bson.RawValue{}, fmt.Errorf("unterminated JSON array")
			}
			array, err = bsoncore.AppendArrayEnd(array, index)
			return bson.RawValue{Type: bson.TypeArray, Value: array}, err
		default:
			return bson.RawValue{}, fmt.Errorf("unexpected JSON delimiter %q", token)
		}
	case nil:
		return bson.RawValue{Type: bson.TypeNull}, nil
	case bool:
		return bson.RawValue{Type: bson.TypeBoolean, Value: bsoncore.AppendBoolean(nil, token)}, nil
	case string:
		return bson.RawValue{Type: bson.TypeString, Value: bsoncore.AppendString(nil, token)}, nil
	case json.Number:
		typ, payload, err := bson.MarshalValue(normalizeBSONInput(token))
		return bson.RawValue{Type: typ, Value: payload}, err
	default:
		return bson.RawValue{}, fmt.Errorf("unsupported JSON token %T", token)
	}
}

// NewBSONFromScmer converts an SQL/Scmer value into the JSON-compatible BSON
// representation used by JSON columns. Strings are parsed as JSON documents;
// already encoded BSON values are returned unchanged.
func NewBSONFromScmer(value Scmer) (Scmer, error) {
	if value.IsBSON() {
		return value, nil
	}
	if value.IsString() || value.IsSymbol() {
		return NewBSONFromJSON(value.String())
	}
	return bsonFromGo(jsonConstructorArgument(value))
}

func bsonTypeAndBytes(value Scmer) (bson.Type, []byte) {
	if value.GetTag() != tagBSON {
		panic(fmt.Sprintf("not BSON (tag=%d, value=%s)", value.GetTag(), value.String()))
	}
	packed := auxVal(value.aux)
	typ := bson.Type(packed & ((1 << bsonTypeBits) - 1))
	length := int((packed >> bsonTypeBits) & bsonLenMask)
	if length == 0 {
		return typ, nil
	}
	return typ, unsafe.Slice(value.ptr, length)
}

func bsonRawValue(value Scmer) bson.RawValue {
	typ, payload := bsonTypeAndBytes(value)
	return bson.RawValue{Type: typ, Value: payload}
}

func bsonFromSQLScalar(value Scmer) (Scmer, error) {
	if value.IsBSON() {
		return value, nil
	}
	switch {
	case value.IsNil():
		return NewBSONValue(bson.TypeNull, nil), nil
	case value.IsBool():
		return NewBSONValue(bson.TypeBoolean, bsoncore.AppendBoolean(nil, value.Bool())), nil
	case value.IsInt():
		integer := value.Int()
		if integer >= math.MinInt32 && integer <= math.MaxInt32 {
			return NewBSONValue(bson.TypeInt32, bsoncore.AppendInt32(nil, int32(integer))), nil
		}
		return NewBSONValue(bson.TypeInt64, bsoncore.AppendInt64(nil, integer)), nil
	case value.IsFloat():
		return NewBSONValue(bson.TypeDouble, bsoncore.AppendDouble(nil, value.Float())), nil
	case value.IsString() || value.IsSymbol():
		return NewBSONValue(bson.TypeString, bsoncore.AppendString(nil, value.String())), nil
	default:
		return bsonFromGo(jsonConstructorArgument(value))
	}
}

func bsonArrayFromValues(values []Scmer, flags uint64) Scmer {
	index, payload := bsoncore.AppendArrayStart(nil)
	for i := range values {
		value, err := bsonFromSQLScalar(values[i])
		if err != nil {
			panic(err)
		}
		raw := bsonRawValue(value)
		payload = bsoncore.AppendValueElement(payload, strconv.Itoa(i), bsoncore.Value{
			Type: bsoncore.Type(raw.Type), Data: raw.Value,
		})
	}
	var err error
	payload, err = bsoncore.AppendArrayEnd(payload, index)
	if err != nil {
		panic(err)
	}
	return newBSONValueFlags(bson.TypeArray, payload, flags)
}

func bsonArrayFromRawValues(values []bson.RawValue, flags uint64) Scmer {
	index, payload := bsoncore.AppendArrayStart(nil)
	for i := range values {
		payload = bsoncore.AppendValueElement(payload, strconv.Itoa(i), bsoncore.Value{
			Type: bsoncore.Type(values[i].Type), Data: values[i].Value,
		})
	}
	var err error
	payload, err = bsoncore.AppendArrayEnd(payload, index)
	if err != nil {
		panic(err)
	}
	return newBSONValueFlags(bson.TypeArray, payload, flags)
}

func bsonDocumentFromPairs(pairs []bsonJSONPair, flags uint64) Scmer {
	// JSON object keys are canonicalized to deterministic order. For duplicate
	// keys the last input pair wins.
	sort.SliceStable(pairs, func(i, j int) bool { return pairs[i].key < pairs[j].key })
	index, payload := bsoncore.AppendDocumentStart(nil)
	for i := 0; i < len(pairs); {
		j := i + 1
		for j < len(pairs) && pairs[j].key == pairs[i].key {
			j++
		}
		pair := pairs[j-1]
		payload = bsoncore.AppendValueElement(payload, encodeBSONKey(pair.key), bsoncore.Value{
			Type: bsoncore.Type(pair.value.Type), Data: pair.value.Value,
		})
		i = j
	}
	var err error
	payload, err = bsoncore.AppendDocumentEnd(payload, index)
	if err != nil {
		panic(err)
	}
	return newBSONValueFlags(bson.TypeEmbeddedDocument, payload, flags)
}

func bsonFromGo(value any) (Scmer, error) {
	return bsonFromGoFlags(value, 0)
}

func bsonFromGoFlags(value any, flags uint64) (Scmer, error) {
	if value == nil {
		return newBSONValueFlags(bson.TypeNull, nil, flags), nil
	}
	typ, payload, err := bson.MarshalValue(normalizeBSONInput(value))
	if err != nil {
		return NewNil(), err
	}
	return newBSONValueFlags(typ, payload, flags), nil
}

func normalizeBSONInput(value any) any {
	switch value := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		document := make(bson.D, 0, len(keys))
		for _, key := range keys {
			document = append(document, bson.E{Key: encodeBSONKey(key), Value: normalizeBSONInput(value[key])})
		}
		return document
	case bson.D:
		document := make(bson.D, len(value))
		for i := range value {
			document[i] = bson.E{Key: encodeBSONKey(value[i].Key), Value: normalizeBSONInput(value[i].Value)}
		}
		return document
	case []any:
		array := make(bson.A, len(value))
		for i := range value {
			array[i] = normalizeBSONInput(value[i])
		}
		return array
	case bson.A:
		array := make(bson.A, len(value))
		for i := range value {
			array[i] = normalizeBSONInput(value[i])
		}
		return array
	case json.Number:
		text := string(value)
		if !strings.ContainsAny(text, ".eE") {
			if integer, err := strconv.ParseInt(text, 10, 64); err == nil {
				if integer >= math.MinInt32 && integer <= math.MaxInt32 {
					return int32(integer)
				}
				return integer
			}
		}
		if decimal, err := bson.ParseDecimal128(text); err == nil {
			return decimal
		}
		if floating, err := strconv.ParseFloat(text, 64); err == nil {
			return floating
		}
		panic("JSON number cannot be represented as BSON: " + text)
	default:
		return value
	}
}

func parseJSONText(text string) (any, error) {
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return nil, fmt.Errorf("multiple JSON values")
	} else if err != io.EOF {
		return nil, err
	}
	return value, nil
}

func bsonDecoded(value Scmer) any {
	var decoded any
	if err := bsonRawValue(value).Unmarshal(&decoded); err != nil {
		panic(err)
	}
	return denormalizeBSONOutput(decoded)
}

func denormalizeBSONOutput(value any) any {
	switch value := value.(type) {
	case bson.D:
		document := make(bson.D, len(value))
		for i := range value {
			document[i] = bson.E{Key: decodeBSONKey(value[i].Key), Value: denormalizeBSONOutput(value[i].Value)}
		}
		return document
	case bson.A:
		array := make(bson.A, len(value))
		for i := range value {
			array[i] = denormalizeBSONOutput(value[i])
		}
		return array
	default:
		return value
	}
}

func bsonAny(value Scmer) any {
	return bsonToJSONGo(bsonDecoded(value))
}

func bsonToJSONGo(value any) any {
	switch value := value.(type) {
	case bson.D:
		result := make(map[string]any, len(value))
		for _, element := range value {
			result[element.Key] = bsonToJSONGo(element.Value)
		}
		return result
	case bson.A:
		result := make([]any, len(value))
		for i := range value {
			result[i] = bsonToJSONGo(value[i])
		}
		return result
	case bson.Decimal128:
		return json.Number(value.String())
	case int32:
		return int64(value)
	default:
		return value
	}
}

func appendOrdinaryJSON(dst []byte, value any, pretty bool, prefix string) ([]byte, error) {
	jsonValue := bsonToJSONGo(value)
	if pretty {
		encoded, err := json.MarshalIndent(jsonValue, prefix, "  ")
		return append(dst, encoded...), err
	}
	encoded, err := json.Marshal(jsonValue)
	return append(dst, encoded...), err
}

func appendJSONString(dst []byte, value string) []byte {
	dst = append(dst, '"')
	for len(value) > 0 {
		r, size := utf8.DecodeRuneInString(value)
		if r == utf8.RuneError && size == 1 {
			dst = append(dst, `\ufffd`...)
			value = value[1:]
			continue
		}
		value = value[size:]
		switch r {
		case '\\', '"':
			dst = append(dst, '\\', byte(r))
		case '\b':
			dst = append(dst, `\b`...)
		case '\f':
			dst = append(dst, `\f`...)
		case '\n':
			dst = append(dst, `\n`...)
		case '\r':
			dst = append(dst, `\r`...)
		case '\t':
			dst = append(dst, `\t`...)
		default:
			if r < 0x20 {
				const hex = "0123456789abcdef"
				dst = append(dst, `\u00`...)
				dst = append(dst, hex[byte(r)>>4], hex[byte(r)&0xf])
			} else {
				dst = utf8.AppendRune(dst, r)
			}
		}
	}
	return append(dst, '"')
}

func appendJSONStringBytes(dst []byte, value []byte) []byte {
	dst = append(dst, '"')
	for len(value) > 0 {
		r, size := utf8.DecodeRune(value)
		if r == utf8.RuneError && size == 1 {
			dst = append(dst, `\ufffd`...)
			value = value[1:]
			continue
		}
		value = value[size:]
		switch r {
		case '\\', '"':
			dst = append(dst, '\\', byte(r))
		case '\b':
			dst = append(dst, `\b`...)
		case '\f':
			dst = append(dst, `\f`...)
		case '\n':
			dst = append(dst, `\n`...)
		case '\r':
			dst = append(dst, `\r`...)
		case '\t':
			dst = append(dst, `\t`...)
		default:
			if r < 0x20 {
				const hex = "0123456789abcdef"
				dst = append(dst, `\u00`...)
				dst = append(dst, hex[byte(r)>>4], hex[byte(r)&0xf])
			} else {
				dst = utf8.AppendRune(dst, r)
			}
		}
	}
	return append(dst, '"')
}

func bsonForEachElement(value bson.RawValue, visit func([]byte, bson.RawValue) error) error {
	if value.Type != bson.TypeEmbeddedDocument && value.Type != bson.TypeArray {
		return fmt.Errorf("BSON type %s is not a container", value.Type)
	}
	if len(value.Value) < bsoncore.EmptyDocumentLength {
		return fmt.Errorf("invalid BSON container length %d", len(value.Value))
	}
	remaining := value.Value[4 : len(value.Value)-1]
	for len(remaining) > 0 {
		element, rest, ok := bsoncore.ReadElement(remaining)
		if !ok {
			return fmt.Errorf("invalid BSON element")
		}
		key, err := element.KeyBytesErr()
		if err != nil {
			return err
		}
		child := element.Value()
		if err := visit(key, bson.RawValue{Type: bson.Type(child.Type), Value: child.Data}); err != nil {
			return err
		}
		remaining = rest
	}
	return nil
}

// appendBSONJSON serializes a BSON-backed SQL JSON value directly into dst.
// It walks raw BSON values and never constructs bson.D, bson.A, maps, or
// interface slices. The returned bytes are ordinary JSON, not Extended JSON.
func appendBSONJSON(dst []byte, value bson.RawValue, pretty bool, prefix string, depth int) ([]byte, error) {
	indent := func(dst []byte, level int) []byte {
		dst = append(dst, '\n')
		dst = append(dst, prefix...)
		for range level {
			dst = append(dst, "  "...)
		}
		return dst
	}
	switch value.Type {
	case bson.TypeNull:
		return append(dst, "null"...), nil
	case bson.TypeBoolean:
		return strconv.AppendBool(dst, value.Boolean()), nil
	case bson.TypeInt32:
		return strconv.AppendInt(dst, int64(value.Int32()), 10), nil
	case bson.TypeInt64:
		return strconv.AppendInt(dst, value.Int64(), 10), nil
	case bson.TypeDouble:
		floating := value.Double()
		if math.IsNaN(floating) || math.IsInf(floating, 0) {
			return nil, fmt.Errorf("non-finite BSON double is not JSON")
		}
		return strconv.AppendFloat(dst, floating, 'g', -1, 64), nil
	case bson.TypeDecimal128:
		return append(dst, value.Decimal128().String()...), nil
	case bson.TypeString:
		return appendJSONString(dst, value.StringValue()), nil
	case bson.TypeEmbeddedDocument:
		dst = append(dst, '{')
		count := 0
		err := bsonForEachElement(value, func(key []byte, child bson.RawValue) error {
			if count > 0 {
				dst = append(dst, ',')
			}
			if pretty {
				dst = indent(dst, depth+1)
			}
			if bytes.HasPrefix(key, []byte(bsonEscapedKeyPrefix)) {
				dst = appendJSONString(dst, decodeBSONKey(string(key)))
			} else {
				dst = appendJSONStringBytes(dst, key)
			}
			dst = append(dst, ':')
			if pretty {
				dst = append(dst, ' ')
			}
			var err error
			dst, err = appendBSONJSON(dst, child, pretty, prefix, depth+1)
			count++
			return err
		})
		if err != nil {
			return nil, err
		}
		if pretty && count > 0 {
			dst = indent(dst, depth)
		}
		return append(dst, '}'), nil
	case bson.TypeArray:
		dst = append(dst, '[')
		count := 0
		err := bsonForEachElement(value, func(_ []byte, child bson.RawValue) error {
			if count > 0 {
				dst = append(dst, ',')
			}
			if pretty {
				dst = indent(dst, depth+1)
			}
			var err error
			dst, err = appendBSONJSON(dst, child, pretty, prefix, depth+1)
			count++
			return err
		})
		if err != nil {
			return nil, err
		}
		if pretty && count > 0 {
			dst = indent(dst, depth)
		}
		return append(dst, ']'), nil
	default:
		return nil, fmt.Errorf("BSON type %s is not valid SQL JSON", value.Type)
	}
}

func appendBSONText(dst []byte, value Scmer, pretty bool, prefix string) ([]byte, error) {
	return appendBSONJSON(dst, bsonRawValue(value), pretty, prefix, 0)
}

func bsonText(value Scmer) string {
	encoded, err := appendBSONText(nil, value, false, "")
	if err != nil {
		panic(err)
	}
	return string(encoded)
}

func bsonBool(value Scmer) bool {
	raw := bsonRawValue(value)
	switch raw.Type {
	case bson.TypeNull:
		return false
	case bson.TypeBoolean:
		return raw.Boolean()
	case bson.TypeInt32, bson.TypeInt64, bson.TypeDouble, bson.TypeDecimal128:
		return bsonFloat(value) != 0
	case bson.TypeString:
		// Keep JSON string coercion identical to ordinary SQL strings: empty,
		// "false", and numeric zero are false; other strings are true.
		return NewString(raw.StringValue()).Bool()
	case bson.TypeArray:
		values, err := raw.Array().Values()
		return err == nil && len(values) > 0
	case bson.TypeEmbeddedDocument:
		elements, err := raw.Document().Elements()
		return err == nil && len(elements) > 0
	default:
		return true
	}
}

func bsonInt(value Scmer) int64 {
	raw := bsonRawValue(value)
	switch raw.Type {
	case bson.TypeInt32:
		return int64(raw.Int32())
	case bson.TypeInt64:
		return raw.Int64()
	case bson.TypeDouble:
		return int64(raw.Double())
	case bson.TypeDecimal128:
		return numericPrefixInt(raw.Decimal128().String())
	case bson.TypeBoolean:
		if raw.Boolean() {
			return 1
		}
	case bson.TypeString:
		return numericPrefixInt(raw.StringValue())
	}
	return 0
}

func bsonFloat(value Scmer) float64 {
	raw := bsonRawValue(value)
	switch raw.Type {
	case bson.TypeInt32:
		return float64(raw.Int32())
	case bson.TypeInt64:
		return float64(raw.Int64())
	case bson.TypeDouble:
		return raw.Double()
	case bson.TypeDecimal128:
		return numericPrefixFloat(raw.Decimal128().String())
	case bson.TypeBoolean:
		if raw.Boolean() {
			return 1
		}
	case bson.TypeString:
		return numericPrefixFloat(raw.StringValue())
	}
	return 0
}

func bsonBytesEqual(a, b Scmer) bool {
	at, ab := bsonTypeAndBytes(a)
	bt, bb := bsonTypeAndBytes(b)
	return at == bt && bytes.Equal(ab, bb)
}

func bsonRawNumber(value bson.RawValue) (float64, bool) {
	switch value.Type {
	case bson.TypeInt32:
		return float64(value.Int32()), true
	case bson.TypeInt64:
		return float64(value.Int64()), true
	case bson.TypeDouble:
		return value.Double(), true
	case bson.TypeDecimal128:
		parsed, err := strconv.ParseFloat(value.Decimal128().String(), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func bsonRawEqual(a, b bson.RawValue) bool {
	if an, ok := bsonRawNumber(a); ok {
		bn, bok := bsonRawNumber(b)
		return bok && an == bn
	}
	if a.Type != b.Type {
		return false
	}
	switch a.Type {
	case bson.TypeNull:
		return true
	case bson.TypeBoolean:
		return a.Boolean() == b.Boolean()
	case bson.TypeString:
		return a.StringValue() == b.StringValue()
	case bson.TypeArray:
		av, aerr := a.Array().Values()
		bv, berr := b.Array().Values()
		if aerr != nil || berr != nil || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !bsonRawEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	case bson.TypeEmbeddedDocument:
		ae, aerr := a.Document().Elements()
		be, berr := b.Document().Elements()
		if aerr != nil || berr != nil || len(ae) != len(be) {
			return false
		}
		for i := range ae {
			if ae[i].Key() != be[i].Key() || !bsonRawEqual(ae[i].Value(), be[i].Value()) {
				return false
			}
		}
		return true
	default:
		return bytes.Equal(a.Value, b.Value)
	}
}

func bsonRawLess(a, b bson.RawValue) bool {
	if an, ok := bsonRawNumber(a); ok {
		if bn, bok := bsonRawNumber(b); bok {
			return an < bn
		}
	}
	rank := func(value bson.RawValue) int {
		switch value.Type {
		case bson.TypeNull:
			return 0
		case bson.TypeBoolean:
			return 1
		case bson.TypeInt32, bson.TypeInt64, bson.TypeDouble, bson.TypeDecimal128:
			return 2
		case bson.TypeString:
			return 3
		case bson.TypeArray:
			return 4
		case bson.TypeEmbeddedDocument:
			return 5
		default:
			return 6
		}
	}
	leftRank, rightRank := rank(a), rank(b)
	if leftRank != rightRank {
		return leftRank < rightRank
	}
	switch a.Type {
	case bson.TypeBoolean:
		return !a.Boolean() && b.Boolean()
	case bson.TypeString:
		return a.StringValue() < b.StringValue()
	default:
		left, _ := appendBSONJSON(nil, a, false, "", 0)
		right, _ := appendBSONJSON(nil, b, false, "", 0)
		return bytes.Compare(left, right) < 0
	}
}
