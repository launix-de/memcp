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
	"encoding/binary"
	"encoding/json"
	"io"
	"testing"
	"unsafe"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

func callJSONTest(t *testing.T, name string, args ...Scmer) Scmer {
	t.Helper()
	fn, ok := Globalenv.Vars[Symbol(name)]
	if !ok {
		t.Fatalf("missing builtin %s", name)
	}
	return Apply(fn, args...)
}

func TestBSONScmerRemainsSixteenBytes(t *testing.T) {
	if size := unsafe.Sizeof(Scmer{}); size != 16 {
		t.Fatalf("Scmer size = %d, want 16", size)
	}
}

func TestBSONScmerJSONRoundTrip(t *testing.T) {
	value, err := NewBSONFromJSON(`{"name":"Ada","active":true,"scores":[1,2.5,null]}`)
	if err != nil {
		t.Fatal(err)
	}
	if !value.IsBSON() {
		t.Fatalf("tag = %d, want tagBSON", value.GetTag())
	}
	var decoded any
	if err := json.Unmarshal([]byte(value.String()), &decoded); err != nil {
		t.Fatalf("invalid ordinary JSON output %q: %v", value.String(), err)
	}
	if got := jsonTypeName(bsonDecoded(value)); got != "OBJECT" {
		t.Fatalf("JSON type = %s, want OBJECT", got)
	}
}

func TestBSONScmerScalarConversions(t *testing.T) {
	integer, err := NewBSONFromJSON(`42`)
	if err != nil {
		t.Fatal(err)
	}
	if integer.Int() != 42 || integer.Float() != 42 || !integer.Bool() {
		t.Fatalf("unexpected scalar conversions: int=%d float=%v bool=%v", integer.Int(), integer.Float(), integer.Bool())
	}
	null, err := NewBSONFromJSON(`null`)
	if err != nil {
		t.Fatal(err)
	}
	if null.IsNil() || null.Bool() {
		t.Fatalf("JSON null must remain distinct from SQL NULL")
	}
}

func TestBSONAppendStringUsesCallerBuffer(t *testing.T) {
	value, err := NewBSONFromJSON(`{"escaped":"line\n\u0001","nested":[true,2.5,null]}`)
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]byte, 7, 256)
	text, appended := value.AppendString(buffer)
	if got, want := string(appended[7:]), `{"escaped":"line\n\u0001","nested":[true,2.5,null]}`; got != want || text != want {
		t.Fatalf("AppendString text=%q appended=%q want=%q", text, got, want)
	}
}

func TestBSONWriteAndStreamMatchString(t *testing.T) {
	value, err := NewBSONFromJSON(`{"customer":{"name":"Ada\nLovelace","rank":7},"items":[{"sku":"A","qty":2},null],"active":true}`)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	value.Write(&output)
	if got, want := output.String(), value.String(); got != want {
		t.Fatalf("written BSON = %q, want %q", got, want)
	}
	streamed, err := io.ReadAll(value.Stream())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(streamed), value.String(); got != want {
		t.Fatalf("streamed BSON = %q, want %q", got, want)
	}
}

func TestJSONExtractedBSONScalarCoercions(t *testing.T) {
	document, err := NewBSONFromJSON(`{"integer":41,"floating":1.5,"yes":true,"no":false,"numeric":"2.25","zero":"0","false_string":"false","empty":""}`)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		path     string
		integer  int64
		floating float64
		boolean  bool
	}{
		{"$.integer", 41, 41, true},
		{"$.floating", 1, 1.5, true},
		{"$.yes", 1, 1, true},
		{"$.no", 0, 0, false},
		{"$.numeric", 2, 2.25, true},
		{"$.zero", 0, 0, false},
		{"$.false_string", 0, 0, false},
		{"$.empty", 0, 0, false},
	}
	for _, test := range tests {
		extracted := callJSONTest(t, "json_extract", document, NewString(test.path))
		if !extracted.IsBSON() {
			t.Fatalf("%s did not preserve tagBSON", test.path)
		}
		if extracted.Int() != test.integer || extracted.Float() != test.floating || extracted.Bool() != test.boolean {
			t.Fatalf("%s conversions int=%d float=%v bool=%v", test.path, extracted.Int(), extracted.Float(), extracted.Bool())
		}
		if got := NewFloat(extracted.Float() + 1).Float(); got != test.floating+1 {
			t.Fatalf("%s + 1 = %v, want %v", test.path, got, test.floating+1)
		}
	}
}

func TestJSONFunctionsPreserveNestedBSON(t *testing.T) {
	document, err := NewBSONFromJSON(`{"user":{"name":"Ada"},"tags":["sql","go"]}`)
	if err != nil {
		t.Fatal(err)
	}
	extracted := callJSONTest(t, "json_extract", document, NewString("$.user"))
	result := callJSONTest(t, "json_array", extracted, NewInt(7))
	if got, want := result.String(), `[{"name":"Ada"},7]`; got != want {
		t.Fatalf("JSON_ARRAY nested result = %s, want %s", got, want)
	}
}

func TestJSONModificationAndContainment(t *testing.T) {
	document, err := NewBSONFromJSON(`{"a":[1,2],"obsolete":true}`)
	if err != nil {
		t.Fatal(err)
	}
	modified := callJSONTest(t, "json_set", document, NewString("$.a[1]"), NewInt(9))
	modified = callJSONTest(t, "json_remove", modified, NewString("$.obsolete"))
	if got, want := modified.String(), `{"a":[1,9]}`; got != want {
		t.Fatalf("modified JSON = %s, want %s", got, want)
	}
	candidate, _ := NewBSONFromJSON(`{"a":[9]}`)
	if !callJSONTest(t, "json_contains", modified, candidate).Bool() {
		t.Fatal("JSON_CONTAINS returned false")
	}
}

func TestBSONSchemeSerializationRoundTrip(t *testing.T) {
	value, err := NewBSONFromJSON(`{"quoted":"a\\b\"c","array":[1,2]}`)
	if err != nil {
		t.Fatal(err)
	}
	serialized := SerializeToString(value, &Globalenv)
	restored := Eval(Read("BSON serialization test", serialized), &Globalenv)
	if !restored.IsBSON() || !Equal(value, restored) {
		t.Fatalf("BSON Scheme roundtrip failed: source=%s serialized=%s restored=%s", value.String(), serialized, restored.String())
	}
}

func TestBSONEscapesJSONKeysContainingNUL(t *testing.T) {
	value, err := NewBSONFromJSON("{\"a\\u0000b\":7}")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := value.String(), "{\"a\\u0000b\":7}"; got != want {
		t.Fatalf("NUL key roundtrip = %s, want %s", got, want)
	}
	extracted := callJSONTest(t, "json_extract", value, NewString("$.\"a\\u0000b\""))
	if extracted.Int() != 7 {
		t.Fatalf("NUL key extraction = %s, want 7", extracted.String())
	}
}

func TestJSONArrayAggFinalizeBuildsOneExactBSONArray(t *testing.T) {
	values := make([]Scmer, 105)
	for i := range values {
		switch i % 3 {
		case 0:
			values[i] = NewNil()
		case 1:
			values[i] = NewInt(int64(i))
		case 2:
			values[i] = bsonDocumentFromPairs([]bsonJSONPair{{
				key: "nested", value: bsonRawValue(NewBSONValue(bson.TypeString, bsoncore.AppendString(nil, "value"))),
			}}, 0)
		}
	}
	result := callJSONTest(t, "json_arrayagg_finalize", NewSlice(values))
	if !result.IsBSON() {
		t.Fatal("JSON_ARRAYAGG finalizer did not return BSON")
	}
	typ, payload := result.BSONRaw()
	if bson.Type(typ) != bson.TypeArray {
		t.Fatalf("BSON type = %v, want array", bson.Type(typ))
	}
	if declared := int(binary.LittleEndian.Uint32(payload[:4])); declared != len(payload) {
		t.Fatalf("BSON header length = %d, payload length = %d", declared, len(payload))
	}
	items, err := bson.Raw(payload).Values()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != len(values) {
		t.Fatalf("array length = %d, want %d", len(items), len(values))
	}
}

func TestJSONArrayAggFinalizePreservesEmptyAggregate(t *testing.T) {
	if result := callJSONTest(t, "json_arrayagg_finalize", NewNil()); !result.IsNil() {
		t.Fatalf("empty JSON_ARRAYAGG = %s, want SQL NULL", result.String())
	}
}

func BenchmarkBSONAppendString(b *testing.B) {
	value, err := NewBSONFromJSON(`{"customer":{"name":"Ada","rank":7},"items":[{"sku":"A","qty":2},{"sku":"B","qty":3}],"active":true}`)
	if err != nil {
		b.Fatal(err)
	}
	buffer := make([]byte, 0, 256)
	b.ReportAllocs()
	b.SetBytes(int64(len(value.String())))
	b.ResetTimer()
	for range b.N {
		text, appended := value.AppendString(buffer[:0])
		if len(text) == 0 {
			b.Fatal("empty BSON serialization")
		}
		buffer = appended[:0]
	}
}

func benchmarkJSONArrayAggReduce(b *testing.B, count int) {
	entry := Globalenv.Vars[Symbol("json_arrayagg_entry")].Func()
	reduce := Globalenv.Vars[Symbol("json_arrayagg_reduce")].Func()
	values := make([]Scmer, count)
	for i := range values {
		values[i] = entry(bsonDocumentFromPairs([]bsonJSONPair{
			{key: "id", value: bsonRawValue(NewBSONValue(bson.TypeInt64, bsoncore.AppendInt64(nil, int64(i))))},
			{key: "serialNumber", value: bsonRawValue(NewBSONValue(bson.TypeString, bsoncore.AppendString(nil, "SN-1000000")))},
		}, 0))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result := NewNil()
		for i := range values {
			result = reduce(result, values[i])
		}
		if !result.IsBSON() {
			b.Fatal("JSON_ARRAYAGG did not return BSON")
		}
	}
}

func BenchmarkJSONArrayAggReduce3(b *testing.B) {
	benchmarkJSONArrayAggReduce(b, 3)
}

func BenchmarkJSONArrayAggReduce8(b *testing.B) {
	benchmarkJSONArrayAggReduce(b, 8)
}

func BenchmarkJSONArrayAggReduce1000(b *testing.B) {
	benchmarkJSONArrayAggReduce(b, 1000)
}

func benchmarkJSONArrayAggCollectFinalize(b *testing.B, count int) {
	values := make([]Scmer, count)
	for i := range values {
		values[i] = bsonDocumentFromPairs([]bsonJSONPair{
			{key: "id", value: bsonRawValue(NewBSONValue(bson.TypeInt64, bsoncore.AppendInt64(nil, int64(i))))},
			{key: "serialNumber", value: bsonRawValue(NewBSONValue(bson.TypeString, bsoncore.AppendString(nil, "SN-1000000")))},
		}, 0)
	}
	collected := NewSlice(values)
	finalize := Globalenv.Vars[Symbol("json_arrayagg_finalize")].Func()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result := finalize(collected)
		if !result.IsBSON() {
			b.Fatal("JSON_ARRAYAGG did not return BSON")
		}
	}
}

func BenchmarkJSONArrayAggCollectFinalize3(b *testing.B) {
	benchmarkJSONArrayAggCollectFinalize(b, 3)
}

func BenchmarkJSONArrayAggCollectFinalize8(b *testing.B) {
	benchmarkJSONArrayAggCollectFinalize(b, 8)
}

func BenchmarkJSONArrayAggCollectFinalize1000(b *testing.B) {
	benchmarkJSONArrayAggCollectFinalize(b, 1000)
}

func BenchmarkBSONSerializeNested1000(b *testing.B) {
	serials := bsonArrayFromRawValues([]bson.RawValue{
		bsonRawValue(NewBSONValue(bson.TypeString, bsoncore.AppendString(nil, "SN-1000000"))),
		bsonRawValue(NewBSONValue(bson.TypeString, bsoncore.AppendString(nil, "SN-1000001"))),
		bsonRawValue(NewBSONValue(bson.TypeString, bsoncore.AppendString(nil, "SN-1000002"))),
	}, 0)
	values := make([]bson.RawValue, 1000)
	for i := range values {
		values[i] = bsonRawValue(bsonDocumentFromPairs([]bsonJSONPair{
			{key: "id", value: bsonRawValue(NewBSONValue(bson.TypeInt64, bsoncore.AppendInt64(nil, int64(i))))},
			{key: "number", value: bsonRawValue(NewBSONValue(bson.TypeString, bsoncore.AppendString(nil, "DN-1000000")))},
			{key: "serialNumbers", value: bsonRawValue(serials)},
		}, 0))
	}
	value := bsonArrayFromRawValues(values, 0)

	b.Run("append-materialize", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			encoded, err := appendBSONText(nil, value, false, "")
			if err != nil {
				b.Fatal(err)
			}
			if len(encoded) == 0 {
				b.Fatal("empty BSON serialization")
			}
		}
	})
	b.Run("write-stream", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			value.Write(io.Discard)
		}
	})
}
