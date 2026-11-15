/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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
package storage

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/launix-de/memcp/scm"
)

func buildStorageString(values []scm.Scmer) *StorageString {
	s := new(StorageString)
	s.prepare()
	for i, v := range values {
		s.scan(uint32(i), v)
	}
	s.init(uint32(len(values)))
	for i, v := range values {
		s.build(uint32(i), v)
	}
	return rawStringFixture(s)
}

func TestStorageStringJITEmit(t *testing.T) {
	values := []scm.Scmer{
		scm.NewString("hello"), scm.NewString("world"), scm.NewString("foo"),
		scm.NewString("bar"), scm.NewString("baz"),
	}
	s := buildStorageString(values)
	jitGet, cleanup := jitBuildGetValueFunc(t, s, true)
	defer cleanup()

	for i := 0; i < len(values); i++ {
		got := jitGet(int64(i))
		expected := s.GetValue(uint32(i))
		if !scmerEqual(got, expected) {
			t.Errorf("idx=%d: JIT got %v, GetValue got %v", i, got, expected)
		}
	}
}

func TestStorageStringJITEmitWithNull(t *testing.T) {
	values := []scm.Scmer{
		scm.NewString("alpha"), scm.NewNil(), scm.NewString("beta"),
		scm.NewNil(), scm.NewString("gamma"),
	}
	s := buildStorageString(values)
	jitGet, cleanup := jitBuildGetValueFunc(t, s, true)
	defer cleanup()

	for i := 0; i < len(values); i++ {
		got := jitGet(int64(i))
		expected := s.GetValue(uint32(i))
		if !scmerEqual(got, expected) {
			t.Errorf("idx=%d: JIT got %v, GetValue got %v", i, got, expected)
		}
	}
}

func TestStorageStringJITEmitDictionary(t *testing.T) {
	// Many repeated values to trigger dictionary encoding
	dict := []string{"alpha", "beta", "gamma", "delta"}
	n := 200
	values := make([]scm.Scmer, n)
	for i := range values {
		values[i] = scm.NewString(dict[i%len(dict)])
	}
	s := buildStorageString(values)
	jitGet, cleanup := jitBuildGetValueFunc(t, s, true)
	defer cleanup()

	for i := 0; i < n; i++ {
		got := jitGet(int64(i))
		expected := s.GetValue(uint32(i))
		if !scmerEqual(got, expected) {
			t.Errorf("idx=%d: JIT got %q, GetValue got %q", i, got, expected)
		}
	}
}

func TestStorageStringJITEmitRegPtr(t *testing.T) {
	values := []scm.Scmer{
		scm.NewString("one"), scm.NewString("two"), scm.NewString("three"),
	}
	s := buildStorageString(values)
	jitGet, cleanup := jitBuildGetValueFunc(t, s, false)
	defer cleanup()

	for i := 0; i < len(values); i++ {
		got := jitGet(int64(i))
		expected := s.GetValue(uint32(i))
		if !scmerEqual(got, expected) {
			t.Errorf("idx=%d: JIT got %v, GetValue got %v", i, got, expected)
		}
	}
}

func TestStorageStringJITEmitUniqueStrings(t *testing.T) {
	// All unique strings — no dictionary, direct nodict encoding
	n := 50
	values := make([]scm.Scmer, n)
	for i := range values {
		values[i] = scm.NewString(fmt.Sprintf("unique-%d", i))
	}
	s := buildStorageString(values)
	jitGet, cleanup := jitBuildGetValueFunc(t, s, true)
	defer cleanup()

	for i := 0; i < n; i++ {
		got := jitGet(int64(i))
		expected := s.GetValue(uint32(i))
		if !scmerEqual(got, expected) {
			t.Errorf("idx=%d: JIT got %q, GetValue got %q", i, got, expected)
		}
	}
}

// Force stack scanning while a value returned by the storage emitter is live.
// This requires the same registered frames and pointer maps as production;
// executing an anonymous mmap function cannot safely cross this Go call.
type collectingStringEmitter struct{ *StorageString }

func (s collectingStringEmitter) JITEmit(ctx *scm.JITContext, idx, result scm.JITValueDesc) scm.JITValueDesc {
	out := s.StorageString.JITEmit(ctx, idx, result)
	ctx.StabilizeDescAcrossNestedCall(&out)
	ctx.EmitGoCallVoid(scm.GoFuncAddr(runtime.GC), nil)
	return out
}

func TestStorageStringJITReaderSurvivesGC(t *testing.T) {
	values := []scm.Scmer{scm.NewString("reader first value"), scm.NewString("reader second value")}
	s := buildStorageString(values)
	reader, cleanup := jitBuildGetValueFunc(t, collectingStringEmitter{s}, true)
	defer cleanup()
	for range 3 {
		for i, want := range values {
			if got := reader(int64(i)); !scmerEqual(got, want) {
				t.Fatalf("reader[%d] after GC = %v, want %v", i, got, want)
			}
		}
	}
}

const benchStringN = 60000

func BenchmarkStorageStringGetValue(b *testing.B) {
	dict := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta"}
	values := make([]scm.Scmer, benchStringN)
	for i := range values {
		values[i] = scm.NewString(dict[i%len(dict)])
	}
	s := buildStorageString(values)

	b.Run("Go", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for j := uint32(0); j < benchStringN; j++ {
				s.GetValue(j)
			}
		}
	})

	b.Run("JIT_ConstFold", func(b *testing.B) {
		if runtime.GOARCH != "amd64" {
			b.Skip("JIT only on amd64")
		}
		jitGet, cleanup := jitBuildGetValueFunc(b, s, true)
		defer cleanup()
		// validate first
		for j := 0; j < 10; j++ {
			got := jitGet(int64(j))
			expected := s.GetValue(uint32(j))
			if !scmerEqual(got, expected) {
				b.Fatalf("idx=%d: mismatch JIT=%v Go=%v", j, got, expected)
			}
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for j := int64(0); j < benchStringN; j++ {
				jitGet(j)
			}
		}
	})

	b.Run("JIT_RegPtr", func(b *testing.B) {
		if runtime.GOARCH != "amd64" {
			b.Skip("JIT only on amd64")
		}
		jitGet, cleanup := jitBuildGetValueFunc(b, s, false)
		defer cleanup()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for j := int64(0); j < benchStringN; j++ {
				jitGet(j)
			}
		}
	})
}

// The storage reader and predicate share one generated loop. Exercise both
// nullable and fragmented entries so constructor spills cannot hide behind a
// materialized string or a separately compiled native storage reader.
func TestStorageStringFusedConcatFilter(t *testing.T) {
	if !scm.JITEnabled() {
		t.Skip("requires the JIT experiment")
	}
	values := make([]scm.Scmer, 137)
	for i := range values {
		if i%11 == 0 {
			values[i] = scm.NewNil()
		} else if i%17 == 0 {
			values[i] = scm.NewString("")
		} else {
			values[i] = scm.NewString(strings.Repeat("report geräteprüfung ", 25) + fmt.Sprint(i))
		}
	}
	column := buildStringCompressionFixture(values)
	fragmented := false
	for i, want := range values {
		got := column.GetJITGetValue()(uint32(i))
		if !scm.Equal(got, want) || (!want.IsNil() && got.GetTag() != scm.TagConcat) {
			t.Fatalf("scalar row %d: got %v (tag %d), want %v", i, got, got.GetTag(), want)
		}
		if !got.IsNil() {
			_, direct := column.textBuffer.Direct(scm.BufferStringOffset(got))
			fragmented = fragmented || !direct
		}
	}
	if !fragmented {
		t.Fatal("fixture did not create fragmented concat entries")
	}
	for _, source := range []string{
		`(lambda (label) (if (nil? label) false (> (strlen (concat label)) 100)))`,
		`(lambda (label) (if (nil? label) false (strlike (toLower label) "%report%")))`,
		`(lambda (label) (if (nil? label) false ((collate "utf8mb4_general_ci") label "zzzzz")))`,
	} {
		t.Run(source, func(t *testing.T) {
			procedure := scm.Eval(scm.Optimize(scm.Read(t.Name(), source, true), &scm.Globalenv, nil), &scm.Globalenv)
			if !procedure.IsProc() {
				t.Fatal("predicate is not a procedure")
			}
			filter := scm.CompileJITFilterStorage(procedure.Proc(), []uint8{scm.JITTypeUnknown}, []scm.JITStorageGetValueEmitter{column.JITEmit})
			if filter == nil {
				t.Fatal("fused storage filter did not compile")
			}
			var want []uint32
			for i, value := range values {
				if scm.Apply(procedure, value).Bool() {
					want = append(want, uint32(i))
				}
			}
			ids := make([]uint32, len(values))
			reset := func() {
				for i := range ids {
					ids[i] = uint32(i)
				}
			}
			reset()
			count := filter(ids, nil)
			if count != len(want) || !reflect.DeepEqual(ids[:count], want) {
				t.Fatalf("selected %v, want %v", ids[:count], want)
			}
			if allocations := testing.AllocsPerRun(20, func() { reset(); filter(ids, nil) }); allocations != 0 {
				t.Fatalf("fused concat predicate allocates %g times", allocations)
			}
		})
	}
}
