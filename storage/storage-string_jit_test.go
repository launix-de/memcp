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
	"runtime"
	"testing"

	"github.com/launix-de/memcp/scm"
)

func buildStorageString(values []scm.Scmer) *StorageString {
	col := buildViaCompression(len(values), func(i int) scm.Scmer { return values[i] })
	if ss, ok := col.(*StorageString); ok {
		return ss
	}
	// If compression chose a different type, force StorageString
	s := new(StorageString)
	s.prepare()
	for i, v := range values {
		s.scan(uint32(i), v)
	}
	s.init(uint32(len(values)))
	for i, v := range values {
		s.build(uint32(i), v)
	}
	s.finish()
	return s
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
