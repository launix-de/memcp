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
	"unsafe"

	"go.mongodb.org/mongo-driver/v2/bson"
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
	value, err := parseJSONText(text)
	if err != nil {
		return NewNil(), err
	}
	return bsonFromGo(value)
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

func bsonText(value Scmer) string {
	encoded, err := appendOrdinaryJSON(nil, bsonDecoded(value), false, "")
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
		return raw.StringValue() != ""
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
