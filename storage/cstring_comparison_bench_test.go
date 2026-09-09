/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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

import "fmt"
import "sort"
import "strings"
import "testing"
import "encoding/base64"
import "github.com/launix-de/memcp/scm"

var cstringBenchSink bool

func BenchmarkCStringCompare(b *testing.B) {
	for _, at := range []int{0, 31, 63, 64} {
		left := strings.Repeat("a", 64)
		right := []byte(left)
		if at < 64 {
			right[at] = 'b'
		}
		column := buildStringColumn([]string{left, string(right), "0123456789abcdef"})
		a, c := column.GetValue(0), column.GetValue(1)
		s := scm.NewString(string(right))
		for _, op := range []struct {
			name string
			fn   func() bool
		}{
			{"equal-CS", func() bool { return scm.Equal(a, s) }},
			{"equalSQL-CS", func() bool { return scm.EqualSQL(a, s).Bool() }},
			{"less-CS", func() bool { return scm.Less(a, s) }},
			{"less-CC", func() bool { return scm.Less(a, c) }},
		} {
			b.Run(fmt.Sprintf("difference-%d/%s", at, op.name), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					cstringBenchSink = op.fn()
				}
			})
		}
	}
	uuid := buildStringColumn([]string{"550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440001"})
	a, c := uuid.GetValue(0), uuid.GetValue(1)
	b.Run("uuid-less-CC", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			cstringBenchSink = scm.Less(a, c)
		}
	})
}

func BenchmarkCStringIndexSort(b *testing.B) {
	values := make([]string, 4096)
	for i := range values {
		values[i] = fmt.Sprintf("abcdef0123456789%016x", (i*2654435761)%4096)
	}
	column := buildStringColumn(values)
	original := make([]scm.Scmer, len(values))
	for i := range original {
		original[i] = column.GetValue(uint32(i))
	}
	work := make([]scm.Scmer, len(original))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(work, original)
		sort.Slice(work, func(i, j int) bool { return scm.Less(work[i], work[j]) })
	}
}

var cstringValueSink scm.Scmer

func BenchmarkCStringStringOps(b *testing.B) {
	text := strings.Repeat("abcdef0123456789", 128)
	column := buildStringColumn([]string{text, text + "0", text + "1"})
	value := column.GetValue(0)
	for _, op := range []struct {
		name, builtin string
		tail          []scm.Scmer
	}{
		{"prefix", "strlike", []scm.Scmer{scm.NewString("abc%")}},
		{"suffix", "strlike", []scm.Scmer{scm.NewString("%6789")}},
		{"contains", "strlike", []scm.Scmer{scm.NewString("%def012%")}},
		{"absent", "strlike", []scm.Scmer{scm.NewString("%def112%")}},
		{"length", "strlen", nil},
		{"substring", "substr", []scm.Scmer{scm.NewInt(511), scm.NewInt(16)}},
	} {
		fn := scm.Globalenv.Vars[scm.Symbol(op.builtin)].Func()
		args := append([]scm.Scmer{value}, op.tail...)
		b.Run(op.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				cstringValueSink = fn(args...)
			}
		})
	}
}

func BenchmarkCStringLegacy(b *testing.B) {
	for _, at := range []int{0, 64} {
		left := strings.Repeat("a", 64)
		right := []byte(left)
		if at < 64 {
			right[at] = 'b'
		}
		p, _ := appendNibbles(nil, left, nibbleCharsetFor(FormatHexLower), 0)
		q, _ := appendNibbles(nil, string(right), nibbleCharsetFor(FormatHexLower), 0)
		a, c := scm.NewCString(&p[0], 2, 0, 64), scm.NewCString(&q[0], 2, 0, 64)
		plain := scm.NewString(string(right))
		b.Run(fmt.Sprintf("difference-%d/less-CC", at), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				cstringBenchSink = scm.Less(a, c)
			}
		})
		b.Run(fmt.Sprintf("difference-%d/equal-CS", at), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				cstringBenchSink = scm.Equal(a, plain)
			}
		})
	}
}

// Identical fixtures on the development baseline and candidate; constructors
// are intentionally reached through storage to measure the selected format.
func BenchmarkBase64Compare(b *testing.B) {
	for _, size := range []int{16, 2048} {
		text := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", size)))
		for _, at := range []int{0, len(text) - 4} {
			other := text[:at] + "A" + text[at+1:]
			col := buildStringColumn([]string{text, other})
			a, c := col.GetValue(0), col.GetValue(1)
			p := scm.NewString(other)
			for _, op := range []struct {
				name string
				fn   func() bool
			}{
				{"equal-BS", func() bool { return scm.Equal(a, p) }},
				{"less-BS", func() bool { return scm.Less(a, p) }},
				{"less-BB", func() bool { return scm.Less(a, c) }},
				{"equal-BB", func() bool { return scm.Equal(a, c) }},
				{"equal-same-BB", func() bool { return scm.Equal(a, a) }},
				{"equalSQL-BS", func() bool { return scm.EqualSQL(a, p).Bool() }},
				{"equalSQL-same-BB", func() bool { return scm.EqualSQL(a, a).Bool() }},
			} {
				b.Run(fmt.Sprintf("%d/difference-%d/%s", size, at, op.name), func(b *testing.B) {
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						cstringBenchSink = op.fn()
					}
				})
			}
		}
	}
}

func BenchmarkBase64IndexSort(b *testing.B) {
	values := make([]string, 4096)
	for i := range values {
		values[i] = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("prefix-%032d", (i*2654435761)%4096)))
	}
	col := buildStringColumn(values)
	original := make([]scm.Scmer, len(values))
	work := make([]scm.Scmer, len(values))
	for i := range original {
		original[i] = col.GetValue(uint32(i))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(work, original)
		sort.Slice(work, func(i, j int) bool { return scm.Less(work[i], work[j]) })
	}
}

func BenchmarkStringFormatBuild(b *testing.B) {
	for _, format := range []string{"hex", "base64", "raw-base64", "timestamp"} {
		for _, size := range []int{16, 2048} {
			if format == "timestamp" && size != 16 {
				continue
			}
			values := make([]string, 128)
			for i := range values {
				switch format {
				case "hex":
					values[i] = fmt.Sprintf("%0*x", size, i%8)
				case "base64", "raw-base64":
					raw := []byte(strings.Repeat("x", size))
					raw[len(raw)-1] = byte(i % 8)
					enc := base64.StdEncoding
					if format == "raw-base64" {
						enc = base64.RawStdEncoding
					}
					values[i] = enc.EncodeToString(raw)
				case "timestamp":
					values[i] = fmt.Sprintf("2026-09-09T12:34:56.%03d+02:00", i%8)
				}
			}
			b.Run(fmt.Sprintf("%s/%d", format, size), func(b *testing.B) {
				col := buildStringColumn(values)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					col = buildStringColumn(values)
				}
				b.StopTimer()
				b.ReportMetric(float64(len(col.ensureDict())), "dict-B")
				cstringValueSink = col.GetValue(0)
			})
		}
	}
}
