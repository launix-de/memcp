// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later
package storage

import "fmt"
import "os"
import "bytes"
import "sort"
import "testing"
import "strings"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

func blobChoiceText(seed uint64, n int) string {
	text := make([]byte, n)
	for i := range text {
		seed ^= seed << 13
		seed ^= seed >> 7
		seed ^= seed << 17
		text[i] = byte(32 + seed%95)
	}
	return string(text)
}

type blobMeasureGroup struct {
	Name  string   `json:"name"`
	Texts []string `json:"texts"`
}

func blobMeasureGroups(b *testing.B) []blobMeasureGroup {
	shared := blobChoiceText(17, 6000) + " wartung"
	groups := []blobMeasureGroup{{Name: "shared"}, {Name: "unrelated"}, {Name: "duplicates"}, {Name: "within-document"}}
	for i := 0; i < 128; i++ {
		groups[0].Texts = append(groups[0].Texts, fmt.Sprintf("document %04d\n%s", i, shared))
		groups[1].Texts = append(groups[1].Texts, blobChoiceText(uint64(i+1), 6000))
		groups[2].Texts = append(groups[2].Texts, shared)
		groups[3].Texts = append(groups[3].Texts, fmt.Sprintf("document %04d\n%s", i, strings.Repeat("same paragraph inside this document ", 200)))
	}
	if path := os.Getenv("MEMCP_BLOB_CORPUS"); path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			b.Fatal(err)
		}
		var real []blobMeasureGroup
		if err := json.Unmarshal(raw, &real); err != nil {
			b.Fatal(err)
		}
		groups = append(groups, real...)
	}
	return groups
}

// retainedBlobRows models the decoded values retained by a warm blob cache.
// Cold probes use the actual OverlayBlob reader and include gzip materialization.
type retainedBlobRows []scm.Scmer

func (r retainedBlobRows) GetValue(i uint32) scm.Scmer { return r[i] }
func (r retainedBlobRows) GetValueRange(i, n uint32, dst []scm.Scmer, stride int) {
	for j := uint32(0); j < n; j++ {
		dst[int(j)*stride] = r[i+j]
	}
}
func (r retainedBlobRows) GetValueMulti(rows []uint32, dst []scm.Scmer, stride int) {
	for j, row := range rows {
		dst[j*stride] = r[row]
	}
}

var blobMeasureColumn ColumnStorage
var blobMeasureHash uint64

// Use identical fixed -benchtime counts on the baseline and candidate. Build,
// index construction, and per-value operations are measured in separate runs.
func BenchmarkAdaptiveBlob(b *testing.B) {
	for _, group := range blobMeasureGroups(b) {
		b.Run(group.Name, func(b *testing.B) {
			values := make([]scm.Scmer, len(group.Texts))
			sourceBytes, maxLength := 0, 0
			for i, text := range group.Texts {
				values[i] = scm.NewString(text)
				sourceBytes += len(text)
				maxLength = max(maxLength, len(text))
			}
			for _, mode := range []string{"pipeline", "gzip", "concat"} {
				b.Run(mode, func(b *testing.B) {
					build := func() ColumnStorage {
						if mode == "pipeline" {
							return buildViaCompression(len(values), func(i int) scm.Scmer { return values[i] })
						}
						if mode == "concat" {
							return buildStringCompressionFixture(values)
						}
						col := &OverlayBlob{Base: new(StorageString)}
						col.prepare()
						for i, value := range values {
							col.scan(uint32(i), value)
						}
						col.init(uint32(len(values)))
						for i, value := range values {
							col.build(uint32(i), value)
						}
						col.finish()
						return col
					}
					column := build()
					var disk bytes.Buffer
					column.Serialize(&disk)
					handles := make(retainedBlobRows, len(values))
					decoded := map[string]scm.Scmer{}
					decodedBytes := 0
					for i, want := range values {
						value := column.GetValue(uint32(i))
						if got := value.String(); got != want.String() {
							b.Fatal("value mismatch", i)
						}
						if !value.IsBufferString() {
							text := value.String()
							if retained, ok := decoded[text]; ok {
								value = retained
							} else {
								decoded[text] = value
								decodedBytes += len(text)
							}
						}
						handles[i] = value
					}
					b.Logf("FOOTPRINT corpus=%s mode=%s rows=%d source=%d resident=%d decoded=%d serialized=%d", group.Name, mode, len(values), sourceBytes, column.ComputeSize(), decodedBytes, disk.Len())
					b.Run("build", func(b *testing.B) {
						b.ReportAllocs()
						b.ReportMetric(float64(column.ComputeSize()), "resident-B")
						b.ReportMetric(float64(decodedBytes), "decoded-B")
						b.ReportMetric(float64(disk.Len()), "wire-B")
						for i := 0; i < b.N; i++ {
							blobMeasureColumn = build()
						}
					})
					b.Run("bigram-cold", func(b *testing.B) {
						b.ReportAllocs()
						for i := 0; i < b.N; i++ {
							stringBufferBenchIndex = buildBigramIndex(uint32(len(values)), column)
						}
					})
					b.Run("bigram-warm", func(b *testing.B) {
						b.ReportAllocs()
						for i := 0; i < b.N; i++ {
							stringBufferBenchIndex = buildBigramIndex(uint32(len(values)), handles)
						}
					})
					like := scm.Globalenv.Vars["strlike"].Func()
					lessCI := scm.Collate(scm.NewString("utf8mb4_general_ci")).Func()
					pattern, collation := scm.NewString("%wartung%"), scm.NewString("utf8mb4_general_ci")
					for i, value := range handles {
						other := (i + 1) % len(values)
						if scm.Less(value, handles[other]) != scm.Less(values[i], values[other]) ||
							scm.Equal(value, handles[other]) != scm.Equal(values[i], values[other]) ||
							lessCI(value, handles[other]).Bool() != lessCI(values[i], values[other]).Bool() ||
							scm.HashKey(value) != scm.HashKey(values[i]) ||
							like(value, pattern, collation).Bool() != scm.StrLikeCollation(group.Texts[i], "%wartung%", "utf8mb4_general_ci", "\\") {
							b.Fatal("operator mismatch", i)
						}
					}
					for _, operation := range []string{"read-cold", "materialize-warm", "append-warm", "strlen-warm", "equal-warm", "less-warm", "less-ci-warm", "hash-warm", "like-warm"} {
						b.Run(operation, func(b *testing.B) {
							dst := make([]byte, 0, maxLength)
							b.ReportAllocs()
							b.ResetTimer()
							for i := 0; i < b.N; i++ {
								row := i * 2654435761 % len(values)
								value := handles[row]
								switch operation {
								case "read-cold":
									concatBenchValue = column.GetValue(uint32(row))
								case "materialize-warm":
									concatBenchString = value.String()
								case "append-warm":
									concatBenchString, dst = value.AppendString(dst[:0])
								case "strlen-warm":
									if value.IsBufferString() {
										concatBenchBool = scm.BufferStringLength(value) == len(group.Texts[row])
									} else {
										concatBenchBool = len(value.String()) == len(group.Texts[row])
									}
								case "equal-warm":
									concatBenchBool = scm.Equal(value, handles[(row+1)%len(values)])
								case "less-warm":
									concatBenchBool = scm.Less(value, handles[(row+1)%len(values)])
								case "less-ci-warm":
									concatBenchValue = lessCI(value, handles[(row+1)%len(values)])
								case "hash-warm":
									blobMeasureHash = scm.HashKey(value)
								case "like-warm":
									concatBenchValue = like(value, pattern, collation)
								}
							}
						})
					}
				})
			}
		})
	}
}

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

var concatBenchPool *scm.StringConcatCompression
var concatBenchOffsets []uint64

func BenchmarkStringConcatShortDictionary(b *testing.B) {
	texts := make([]string, 22000)
	for i := range texts {
		texts[i] = fmt.Sprintf("Article %d", i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		concatBenchPool, concatBenchOffsets = scm.BuildStringConcatBuffer(texts)
	}
	b.StopTimer()
	for i, offset := range concatBenchOffsets {
		if concatBenchPool.Extract(offset) != texts[i] {
			b.Fatal("short dictionary roundtrip", i)
		}
	}
}
