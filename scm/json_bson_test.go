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
	"encoding/json"
	"testing"
	"unsafe"
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
