// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later
package storage

import "fmt"
import "bytes"
import "sort"
import "testing"
import "strings"
import "github.com/launix-de/memcp/scm"

// Keep this corpus and fixed probe counts identical across LZ4, raw and concat.
func stringCompressionFixture() []scm.Scmer {
	values := make([]scm.Scmer, 4096)
	for i := range values {
		text := fmt.Sprintf("trace %08d ns SELECT payload FROM fop_session WHERE id = '%08x' AND payload = '%s' /* Prüfbericht ÄÖß */", i*17, i, strings.Repeat("abcdef0123456789", 23))
		values[i] = scm.NewString(text)
	}
	return values
}

var concatBenchValue scm.Scmer
var concatBenchBool bool
var concatBenchString string
var concatBenchStorage *StorageString

func BenchmarkStringCompression(b *testing.B) {
	values := stringCompressionFixture()
	s := buildStringCompressionFixture(values)
	for i, value := range values {
		if got := s.GetValue(uint32(i)); !scm.Equal(got, value) {
			b.Fatal("fixture roundtrip", i)
		}
	}
	b.Logf("rows=%d resident=%d source=%d", len(values), s.ComputeSize(), len(values)*len(values[0].String()))
	b.Run("build", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			concatBenchStorage = buildStringCompressionFixture(values)
		}
	})
	for _, hint := range []ColumnReadHints{ReadCompressed, ReadAuto, ReadDecoded} {
		b.Run(fmt.Sprintf("read-%d", hint), func(b *testing.B) {
			reader := s.GetCachedReader(hint)
			defer ReleaseColumnReader(reader)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				concatBenchValue = reader.GetValue(uint32(i * 2654435761 % len(values)))
			}
		})
	}
	reader := s.GetCachedReader(ReadCompressed)
	defer ReleaseColumnReader(reader)
	handles := make([]scm.Scmer, len(values))
	reader.GetValueRange(0, uint32(len(values)), handles, 1)
	b.Run("compare", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			j := i * 2654435761 % len(values)
			concatBenchBool = scm.Less(handles[j], handles[(j+257)%len(values)])
		}
	})
	b.Run("append", func(b *testing.B) {
		dst := make([]byte, 0, 1024)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			concatBenchString, dst = handles[i*2654435761%len(values)].AppendString(dst[:0])
		}
	})
	b.Run("sort", func(b *testing.B) {
		order := make([]int, len(values))
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for j := range order {
				order[j] = j * 2654435761 % len(order)
			}
			sort.Slice(order, func(a, c int) bool { return scm.Less(handles[order[a]], handles[order[c]]) })
			if !sort.SliceIsSorted(order, func(a, c int) bool { return values[order[a]].String() < values[order[c]].String() }) {
				b.Fatal("sort result")
			}
		}
	})
}

func buildStringCompressionFixture(values []scm.Scmer) *StorageString {
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

func TestStringCompressionFootprint(t *testing.T) {
	values := stringCompressionFixture()
	s := buildStringCompressionFixture(values)
	source := 0
	for _, v := range values {
		source += len(v.String())
	}
	var disk bytes.Buffer
	if s.textBuffer == nil {
		s.serializeRawFixtureV3(&disk)
	} else {
		s.Serialize(&disk)
	}
	t.Logf("rows=%d source_bytes=%d resident_bytes=%d serialized_bytes=%d source_to_disk=%.3f", len(values), source, s.ComputeSize(), disk.Len(), float64(source)/float64(disk.Len()))
}
