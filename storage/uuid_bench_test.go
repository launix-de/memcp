/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "testing"
import "github.com/launix-de/memcp/scm"

var uuidHashSink uint64

func BenchmarkUUIDHash(b *testing.B) {
	v := comparisonCString("550e8400-e29b-41d4-a716-446655440000", FormatUUIDLower, 0)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		uuidHashSink = scm.HashKey(v)
	}
}

func BenchmarkUUIDBulk(b *testing.B) {
	values := make([]string, 1024)
	for i := range values {
		values[i] = fmt.Sprintf("550e8400-e29b-41d4-a716-%012x", i)
	}
	col := buildStringColumn(values)
	out := make([]scm.Scmer, len(values))
	last := scm.NewString(values[len(values)-1])
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		col.GetValueRange(0, uint32(len(out)), out, 1)
		if !scm.Equal(out[len(out)-1], last) {
			b.Fatal("bulk result mismatch")
		}
	}
}
