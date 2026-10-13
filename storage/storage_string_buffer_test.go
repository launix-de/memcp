/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "os"
import "sync"
import "time"
import "unsafe"
import "sort"
import "fmt"
import "io"
import "bytes"
import "testing"
import "strings"
import "reflect"
import "encoding/json"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"

func TestStorageStringBuffer(t *testing.T) {
	codec := scm.StringCodecLZ4
	for _, distinct := range []bool{false, true} {
		var values []scm.Scmer
		for i := 0; i < 130; i++ {
			text := "Prüfbericht Geräteprüfung Äöß\n"
			if distinct {
				text += fmt.Sprint(i)
			}
			values = append(values, scm.NewString(text))
		}
		values = append(values, scm.NewNil(), scm.NewString(""), scm.NewString(strings.Repeat("long report \xff Geräte\n", 4000)))
		s := buildStorageString(values)
		if !s.CompressDictionary() {
			t.Fatal("did not compress raw storage")
		}
		if len(s.dictionary) != 0 || len(s.compressedDict) != 0 {
			t.Fatal("retained decoded buffer")
		}
		if s.laststr != "" || s.reverseMap != nil || s.sb.Len() != 0 {
			t.Fatal("retained construction text")
		}
		for i, want := range values {
			got := s.GetValue(uint32(i))
			if got.IsNil() != want.IsNil() || (!want.IsNil() && got.String() != want.String()) {
				t.Fatalf("codec %d distinct %t row %d", codec, distinct, i)
			}
		}
		target := make([]scm.Scmer, len(values))
		s.GetValueRange(0, uint32(len(values)), target, 1)
		if !target[0].IsBufferString() {
			t.Fatal("bulk reader materialized handle")
		}
		if got := testing.AllocsPerRun(30, func() { s.GetValueRange(0, uint32(len(values)), target, 1) }); got != 0 {
			t.Fatalf("bulk allocates %v", got)
		}
		var serialized bytes.Buffer
		s.Serialize(&serialized)
		var restored StorageString
		count := restored.Deserialize(bytes.NewReader(serialized.Bytes()[1:]))
		if count != uint(len(values)) {
			t.Fatal("restored count")
		}
		for i, want := range values {
			got := restored.GetValue(uint32(i))
			if got.IsNil() != want.IsNil() || (!want.IsNil() && got.String() != want.String()) {
				t.Fatalf("restored codec %d row %d", codec, i)
			}
		}
		// Every posting and candidate domain must be exactly the same after
		// direct compressed traversal, including Unicode and invalid UTF-8.
		plain := buildStorageString(values)
		got, want := buildBigramIndex(uint32(len(values)), s), buildBigramIndex(uint32(len(values)), plain)
		if !reflect.DeepEqual(got.grams, want.grams) {
			t.Fatalf("codec %d changed bigram postings", codec)
		}
	}
}

func TestOverlayBlobBufferPassthrough(t *testing.T) {
	values := []scm.Scmer{scm.NewString("ordinary Prüfreport"), scm.NewString("!!escaped"), scm.NewNil()}
	codec := scm.StringCodecLZ4
	s := buildStorageString(values)
	s.CompressDictionary()
	overlay := OverlayBlob{Base: s}
	target := make([]scm.Scmer, len(values))
	overlay.GetValueRange(0, uint32(len(values)), target, 1)
	if !target[0].IsBufferString() || !target[1].IsBufferString() || target[1].String() != "!escaped" || !target[2].IsNil() {
		t.Fatal("overlay broke compressed pass-through")
	}
	if got := testing.AllocsPerRun(30, func() { overlay.GetValueRange(0, uint32(len(values)), target, 1) }); got != 0 {
		t.Fatalf("codec %d overlay allocates %v", codec, got)
	}
	// Prepared batches may mix ordinary values with compressed handles. A
	// decoder is still needed when the first row itself is an ordinary string.
	mixed := []scm.Scmer{scm.NewString("plain"), s.GetValue(1), scm.NewNil()}
	overlay.resolveBlobBatch(mixed, len(mixed), 1)
	if mixed[0].String() != "plain" || !mixed[1].IsBufferString() || mixed[1].String() != "!escaped" || !mixed[2].IsNil() {
		t.Fatal("mixed batch lost compressed pass-through or literal escaping")
	}
	decoded := overlay.GetCachedReader(ReadDecoded)
	decoded.GetValueRange(0, uint32(len(values)), target, 1)
	if target[0].String() != "ordinary Prüfreport" || target[1].String() != "!escaped" || !target[2].IsNil() {
		t.Fatal("decoded reader stripped blob overlay")
	}
	if s.EvictDictionary() != 0 {
		t.Fatal("overlay did not retain base dictionary")
	}
	ReleaseColumnReader(decoded)
	clear(target)
	if s.EvictDictionary() <= 0 {
		t.Fatal("overlay leaked dictionary lease")
	}
}

func TestStorageStringBufferJITValueType(t *testing.T) {
	codec := scm.StringCodecLZ4
	s := buildStorageString([]scm.Scmer{scm.NewString("Alpha report"), scm.NewString("Beta report")})
	s.CompressDictionary()
	want := uint8(scm.TagLZ4)
	if got := s.JITValueType(); got != want {
		t.Fatalf("codec %d exact main tag %d want %d", codec, got, want)
	}
	if scm.JITEnabled() {
		reader := s.GetCachedReader(ReadCompressed)
		for i, text := range []string{"Alpha report", "Beta report"} {
			got := reader.GetValue(uint32(i))
			if got.String() != text || got.GetTag() != want {
				t.Fatalf("codec %d cached scalar JIT row %d got tag %d text %q", codec, i, got.GetTag(), got.String())
			}
		}
		getter, cleanup := jitBuildGetValueFunc(t, s, true)
		for i, text := range []string{"Alpha report", "Beta report"} {
			got := getter(int64(i))
			if got.String() != text || got.GetTag() != want {
				cleanup()
				t.Fatalf("codec %d scalar JIT row %d got tag %d text %q", codec, i, got.GetTag(), got.String())
			}
		}
		cleanup()
	}
	if got := (&OverlayBlob{Base: s}).JITValueType(); got != scm.JITTypeUnknown {
		t.Fatal("overlay promised uniform compressed tag")
	}
	empty := buildStorageString([]scm.Scmer{scm.NewString("report/path"), scm.NewString("")})
	empty.CompressDictionary()
	if empty.JITValueType() != scm.JITTypeUnknown {
		t.Fatal("mixed empty handles promised uniform tag")
	}
}

func TestStorageStringBufferAssignments(t *testing.T) {
	codec := scm.StringCodecLZ4
	texts := []string{"1e2", "1.5", "invalid number"}
	buffer, offsets := scm.BuildCompressedStringBuffer(texts)
	for _, typ := range []string{"INT", "FLOAT", "DECIMAL"} {
		col := column{Name: "n", Typ: typ, AllowNull: true}
		col.UpdateSanitizer()
		for i, text := range texts {
			outcome := func(value scm.Scmer) (result string) {
				defer func() {
					if p := recover(); p != nil {
						result = fmt.Sprint(p)
					}
				}()
				return col.sanitizer(value).String()
			}
			got, want := outcome(scm.NewBufferString(buffer, offsets[i], len(text))), outcome(scm.NewString(text))
			if got != want {
				t.Fatalf("codec %d %s assignment %q got %q want %q", codec, typ, text, got, want)
			}
		}
	}
}

var stringBufferBenchBool bool
var stringBufferBenchIndex *bigramIndex
var stringBufferBenchValue scm.Scmer

func BenchmarkStorageStringBufferCorpus(b *testing.B) {
	path := os.Getenv("MEMCP_STRING_BENCH_CORPUS")
	if path == "" {
		b.Skip("set MEMCP_STRING_BENCH_CORPUS to local text JSON")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		b.Fatal(err)
	}
	var texts []string
	if err = json.Unmarshal(data, &texts); err != nil {
		b.Fatal(err)
	}
	values := make([]scm.Scmer, len(texts))
	var total int64
	for i, text := range texts {
		values[i] = scm.NewString(text)
		total += int64(len(text))
	}
	baselineIndex := buildBigramIndex(uint32(len(texts)), buildStorageString(values))
	for _, mode := range []struct {
		name  string
		codec int
	}{{"raw", 0}, {"lz4", scm.StringCodecLZ4}} {
		storage := buildStorageString(values)
		if mode.codec != 0 && !storage.CompressDictionary() {
			b.Fatal("codec was not applied")
		}
		var serialized bytes.Buffer
		if mode.codec == 0 {
			storage.serializeRawFixtureV3(&serialized)
		} else {
			storage.Serialize(&serialized)
		}
		b.Logf("%s rows=%d source=%d resident=%d serialized=%d", mode.name, len(texts), total, storage.ComputeSize(), serialized.Len())
		handles := make([]scm.Scmer, len(texts))
		storage.GetValueRange(0, uint32(len(texts)), handles, 1)
		for i, v := range handles {
			if v.String() != texts[i] {
				b.Fatalf("%s corpus row %d changed", mode.name, i)
			}
		}
		if got := buildBigramIndex(uint32(len(texts)), storage); !reflect.DeepEqual(got.grams, baselineIndex.grams) {
			b.Fatalf("%s corpus bigram postings changed", mode.name)
		}
		b.Run(mode.name+"/get-range", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				storage.GetValueRange(0, uint32(len(texts)), handles, 1)
			}
		})
		b.Run(mode.name+"/equal-plain", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				j := (i * 2654435761) % len(texts)
				stringBufferBenchBool = scm.Equal(handles[j], values[j])
			}
		})
		b.Run(mode.name+"/less-adjacent", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				j := (i * 2654435761) % len(texts)
				stringBufferBenchBool = scm.Less(handles[j], handles[(j+1)%len(texts)])
			}
		})
		like := scm.Globalenv.Vars["strlike"].Func()
		pattern := scm.NewString("%Inventarnummer%")
		collation := scm.NewString("utf8mb4_general_ci")
		likeArgs := []scm.Scmer{scm.NewNil(), pattern, collation}
		b.Run(mode.name+"/like-random", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				j := (i * 2654435761) % len(texts)
				likeArgs[0] = handles[j]
				stringBufferBenchValue = like(likeArgs...)
			}
		})
		b.Run(mode.name+"/bigram-build", func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(total)
			for i := 0; i < b.N; i++ {
				stringBufferBenchIndex = buildBigramIndex(uint32(len(texts)), storage)
			}
		})
		b.Run(mode.name+"/unicode-scan", func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(total)
			var scratch scm.TextScratch
			var cursor scm.TextCursor
			for i := 0; i < b.N; i++ {
				sum := int64(0)
				for _, v := range handles {
					cursor.Reset(v)
					for {
						r, _, ok := cursor.NextRune(&scratch)
						if !ok {
							break
						}
						sum += int64(r)
					}
				}
				stringBufferBenchValue = scm.NewInt(sum)
			}
		})
		for _, access := range []struct {
			name string
			hint ColumnReadHints
			cold bool
		}{{"sort-compressed", ReadCompressed, false}, {"sort-decoded-cold", ReadDecoded, true}, {"sort-decoded-warm", ReadDecoded, false}} {
			b.Run(mode.name+"/"+access.name, func(b *testing.B) {
				b.ReportAllocs()
				if !access.cold && access.hint == ReadDecoded {
					reader := storage.GetCachedReader(access.hint)
					ReleaseColumnReader(reader)
				}
				for i := 0; i < b.N; i++ {
					if access.cold {
						storage.EvictDictionary()
					}
					reader := storage.GetCachedReader(access.hint)
					keys := make([]scm.Scmer, len(texts))
					rows := make([]uint32, len(texts))
					for row := range rows {
						rows[row] = uint32(row)
						keys[row] = reader.GetValue(uint32(row))
					}
					sort.Slice(rows, func(a, z int) bool { return scm.Less(keys[rows[a]], keys[rows[z]]) })
					for j := 1; j < len(rows); j++ {
						if texts[rows[j]] < texts[rows[j-1]] {
							b.Fatal("sort order changed")
						}
					}
					clear(keys)
					ReleaseColumnReader(reader)
				}
			})
		}
	}
}

func TestDictionaryReaderHintsAndEviction(t *testing.T) {
	codec := scm.StringCodecLZ4
	t.Run(fmt.Sprint(codec), func(t *testing.T) {
		values := make([]scm.Scmer, 300)
		for i := range values {
			values[i] = scm.NewString(fmt.Sprintf("Prüfbericht %d\n%s", i, strings.Repeat("gemeinsamer Text Äöß\n", 200)))
		}
		values[2], values[4] = scm.NewNil(), scm.NewString("")
		s := buildStorageString(values)
		if !s.CompressDictionary() {
			t.Fatal("no compression")
		}
		compressed := s.GetCachedReader(ReadCompressed)
		if !compressed.GetValue(0).IsBufferString() || s.dictCache.Load() != nil {
			t.Fatal("compressed hint decoded dictionary")
		}
		if s.GetCachedReader(ReadAuto).GetValue(0).IsBufferString() != true || s.dictCache.Load() != nil {
			t.Fatal("auto hint populated cache")
		}
		expected := buildBigramIndex(uint32(len(values)), buildStorageString(values))
		if got := buildBigramIndex(uint32(len(values)), s); !reflect.DeepEqual(got.grams, expected.grams) {
			t.Fatal("compressed bigrams changed")
		}
		if s.dictCache.Load() != nil {
			t.Fatal("bigram walk populated dictionary cache")
		}

		before := s.ComputeSize()
		decoded := s.GetCachedReader(ReadDecoded)
		cache := s.dictCache.Load()
		if cache == nil || cache.users != 1 || len(s.dictionary) != 0 || len(s.compressedDict) != 0 {
			t.Fatal("duplicate dictionary representation")
		}
		if got, want := s.ComputeSize()-before, cache.computeSize(); got != want {
			t.Fatalf("inclusive cache size %d != %d", got, want)
		}
		if got := ownedColumnMemory(s); got != before {
			t.Fatalf("parent charged decoded child: %d != %d", got, before)
		}
		for i, want := range values {
			got := decoded.GetValue(uint32(i))
			if got.IsNil() != want.IsNil() || (!got.IsNil() && got.String() != want.String()) {
				t.Fatalf("decoded value %d", i)
			}
			if got.IsBufferString() {
				t.Fatal("decoded hint returned compressed handle")
			}
		}
		target := make([]scm.Scmer, len(values))
		if got := testing.AllocsPerRun(20, func() { decoded.GetValueRange(0, uint32(len(values)), target, 1) }); got != 0 {
			t.Fatalf("decoded range allocates %v", got)
		}
		if !s.GetCachedReader(ReadCompressed).GetValue(0).IsBufferString() {
			t.Fatal("compressed hint ignored warm cache")
		}
		// Bigram walks also preserve their representation when a decoded cache exists.
		buildBigramIndex(uint32(len(values)), s)
		if cache.users != 1 {
			t.Fatal("bigram walk borrowed decoded cache")
		}
		auto := s.GetCachedReader(ReadAuto)
		if auto.(*decodedStringReader).cache != cache || cache.users != 2 {
			t.Fatal("auto did not reuse cache")
		}
		if stringDictCleanup(cache, nil) || s.EvictDictionary() != 0 {
			t.Fatal("evicted live dictionary")
		}
		ReleaseColumnReader(auto)
		ReleaseColumnReader(decoded)
		clear(target)
		if cache.users != 0 || s.EvictDictionary() <= 0 || s.dictCache.Load() != nil {
			t.Fatal("idle dictionary not reclaimed")
		}
		if got := s.GetCachedReader(ReadAuto).GetValue(0); !got.IsBufferString() || got.String() != values[0].String() {
			t.Fatal("auto after eviction")
		}
	})
}

func TestDictionaryStatementLease(t *testing.T) {
	s := buildStorageString([]scm.Scmer{scm.NewString("Prüfbericht Eins"), scm.NewString("Prüfbericht Zwei")})
	s.CompressDictionary()
	seed := s.GetCachedReader(ReadDecoded)
	ReleaseColumnReader(seed)
	tx := NewTxContext(TxCursorStability)
	tx.beginQuery(nil, 91, "dictionary test")
	reader := newCachedColumnReaderTx(s, tx, false, ReadAuto)
	cache := s.dictCache.Load()
	if cache.users != 1 || reader.GetValue(1).String() != "Prüfbericht Zwei" {
		t.Fatal("statement did not retain prepared reader")
	}
	if s.EvictDictionary() != 0 {
		t.Fatal("evicted active statement")
	}
	tx.endQuery(91)
	tx.endQuery(91)
	if cache.users != 0 || tx.queryReaderReleases != nil || s.EvictDictionary() <= 0 {
		t.Fatal("statement lease not released")
	}
}

func TestDictionaryConcurrentLeaseAndEviction(t *testing.T) {
	s := buildStorageString([]scm.Scmer{scm.NewString(strings.Repeat("Prüfbericht Gerät Äöß\n", 5000))})
	s.CompressDictionary()
	owner := s.GetCachedReader(ReadDecoded)
	var workers sync.WaitGroup
	for range 8 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for range 30 {
				r := s.GetCachedReader(ReadAuto)
				if r.GetValue(0).String() != owner.GetValue(0).String() {
					t.Error("concurrent dictionary changed")
				}
				ReleaseColumnReader(r)
				if s.EvictDictionary() != 0 {
					t.Error("evicted active owner")
				}
			}
		}()
	}
	workers.Wait()
	ReleaseColumnReader(owner)
	if s.EvictDictionary() <= 0 {
		t.Fatal("dictionary not evictable after last reader")
	}
}

func TestDictionaryPreparedReaderJITTag(t *testing.T) {
	if !scm.JITEnabled() {
		t.Skip("JIT disabled")
	}
	values := []scm.Scmer{scm.NewString("Prüfbericht Eins"), scm.NewString("Prüfbericht Zwei")}
	s := buildStorageString(values)
	s.CompressDictionary()
	reader := s.GetCachedReader(ReadDecoded)
	defer ReleaseColumnReader(reader)
	condition := scm.NewProcStruct(scm.Proc{
		Params: scm.NewSlice([]scm.Scmer{scm.NewSymbol("v")}),
		Body:   scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal?"), scm.NewSymbol("v"), values[1]}),
		En:     &scm.Globalenv,
	})
	shard := &storageShard{main_count: 2}
	filter := shard.prepareTypedScanFilter(condition, []ColumnStorage{s}, []ColumnReader{reader}, 1000000)
	if filter == nil {
		t.Fatal("typed filter not prepared")
	}
	defer filter.close()
	rows := []uint32{0, 1}
	if n := filter.filterMain(rows, []ColumnReader{reader}); n != 1 || rows[0] != 1 {
		t.Fatalf("prepared tag mismatch: %v", rows[:n])
	}
}

func TestDictionaryCacheLedgerRejectsActiveEviction(t *testing.T) {
	s := buildStorageString([]scm.Scmer{scm.NewString(strings.Repeat("Prüfbericht Ledger\n", 100))})
	s.CompressDictionary()
	reader := s.GetCachedReader(ReadDecoded)
	cache := s.dictCache.Load()
	size := int64(cache.computeSize())
	past := time.Now().Add(-time.Hour).UnixNano()
	cache.lastUsed.Store(past)
	manager := &CacheManager{itemMap: make(map[any]*softItem)}
	item := newSoftItem(cache, size, TypeStringDict, stringDictCleanup, stringDictLastUsed, nil, 0, time.Second)
	manager.addInternal(item)
	item.registeredAt = past
	item.estimatedExpiry = past + int64(time.Second)
	manager.evictExpired()
	if manager.currentMemory != size || manager.itemMap[cache] != item {
		t.Fatal("failed eviction changed ledger")
	}
	ReleaseColumnReader(reader)
	item.estimatedExpiry = past + int64(time.Second)
	manager.evictExpired()
	if manager.currentMemory != 0 || len(manager.itemMap) != 0 || s.dictCache.Load() != nil {
		t.Fatal("successful eviction did not release once")
	}
	replacement := s.GetCachedReader(ReadDecoded)
	defer ReleaseColumnReader(replacement)
	newer := s.dictCache.Load()
	if newer == cache || !stringDictCleanup(cache, nil) || s.dictCache.Load() != newer {
		t.Fatal("old cache identity retired its replacement")
	}
}

func TestDictionaryConcurrentParentAccounting(t *testing.T) {
	s := buildStorageString([]scm.Scmer{scm.NewString(strings.Repeat("Prüfbericht Bilanz\n", 100))})
	s.CompressDictionary()
	overlay := &OverlayBlob{Base: s}
	expected := ownedColumnMemory(overlay)
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		for range 100 {
			r := s.GetCachedReader(ReadDecoded)
			ReleaseColumnReader(r)
			s.EvictDictionary()
		}
	}()
	for {
		if got := ownedColumnMemory(overlay); got != expected {
			t.Errorf("cache publication changed parent bytes: %d != %d", got, expected)
		}
		select {
		case <-finished:
			return
		default:
		}
	}
}

// Every historical raw version remains readable and saves as flag-free V5.
func TestStorageStringPersistenceMigration(t *testing.T) {
	for _, version := range []byte{0, 1, 2, 3, 4, 49} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			values := []string{"historical report one", "historical report two", "historical report one"}
			source := buildStringColumn(values)
			var old bytes.Buffer
			nodict := byte(0)
			count := source.values.count
			if source.nodict {
				nodict = 1
				count = source.starts.count
			}
			if version == 49 {
				old.Write([]byte{20, nodict, '1', '2', '3', '4', '5', '6'})
			} else {
				old.Write([]byte{20, nodict, byte(source.format), version, 0, 0, 0, 0})
			}
			if version != 0 && version != 49 {
				old.WriteByte(0)
			}
			binary.Write(&old, binary.LittleEndian, count)
			source.values.Serialize(&old)
			source.starts.Serialize(&old)
			source.lens.Serialize(&old)
			binary.Write(&old, binary.LittleEndian, uint64(len(source.dictionary)))
			old.WriteString(source.dictionary)
			var loaded StorageString
			loaded.Deserialize(bytes.NewReader(old.Bytes()[1:]))
			if loaded.textBuffer == nil || loaded.dictionary != "" || loaded.dictCache.Load() != nil {
				t.Fatal("historical raw dictionary not normalized before publication")
			}
			var current bytes.Buffer
			loaded.Serialize(&current)
			if current.Bytes()[3] != 5 || binary.LittleEndian.Uint64(current.Bytes()[8:16]) != uint64(len(values)) {
				t.Fatal("writer did not emit flag-free V5")
			}
			var restored StorageString
			restored.Deserialize(bytes.NewReader(current.Bytes()[1:]))
			for i, want := range values {
				if got := restored.GetValue(uint32(i)).String(); got != want {
					t.Fatalf("row %d got %q want %q", i, got, want)
				}
			}
		})
	}
	// Preserve the V4 flag-2 reader and its historical duplicate offsets.
	source := buildStringColumn([]string{"legacy block report one", "legacy block report two"})
	source.CompressDictionary()
	var old bytes.Buffer
	source.serializeStringBuffer(&old)
	var loaded StorageString
	loaded.Deserialize(bytes.NewReader(old.Bytes()[1:]))
	if loaded.GetValue(1).String() != "legacy block report two" {
		t.Fatal("V4 block payload changed")
	}
}

func TestStorageStringAlwaysCompressed(t *testing.T) {
	for name, inputs := range map[string][]string{
		"raw":       {"report item one", "report item two"},
		"hex":       {strings.Repeat("deadbeef", 100), strings.Repeat("cafebabe", 100)},
		"base64":    {"YWJjZGVmZ2hpamtsbW5vcA==", "cXJzdHV2d3h5ejAxMjM0NQ=="},
		"uuid":      {"550e8400-e29b-41d4-a716-446655440000", "6ba7b810-9dad-11d1-80b4-00c04fd430c8"},
		"oversized": {strings.Repeat("report ", scm.StringBufferMaxStringBytes/7+1), "small report"},
	} {
		t.Run(name, func(t *testing.T) {
			var s StorageString
			s.prepare()
			for i, v := range inputs {
				s.scan(uint32(i), scm.NewString(v))
			}
			s.init(uint32(len(inputs)))
			for i, v := range inputs {
				s.build(uint32(i), scm.NewString(v))
			}
			s.finish()
			if s.textBuffer == nil || s.dictionary != "" || s.dictCache.Load() != nil {
				t.Fatal("finish retained an uncompressed dictionary")
			}
			if name == "hex" && s.format == FormatRaw || name == "base64" && s.format == FormatRaw {
				t.Fatal("lost inner encoding")
			}
			var before, after bytes.Buffer
			s.Serialize(&before)
			var restored StorageString
			restored.Deserialize(bytes.NewReader(before.Bytes()[1:]))
			if restored.dictionary != "" || restored.dictCache.Load() != nil {
				t.Fatal("load retained decoded data")
			}
			// Prepare outside the row loop, then hold cache-management ownership.
			// Specialized compressed formats must not acquire that mutex again
			// when the operator consumes its already prepared representation.
			compressed := restored.GetCachedReader(ReadCompressed)
			done := make(chan []string, 1)
			restored.dictMu.Lock()
			go func() {
				values := make([]scm.Scmer, len(inputs))
				compressed.GetValueRange(0, uint32(len(inputs)), values, 1)
				texts := make([]string, len(values))
				for i, value := range values {
					texts[i] = value.String()
				}
				done <- texts
			}()
			var texts []string
			blocked := false
			select {
			case texts = <-done:
			case <-time.After(time.Second):
				blocked = true
			}
			restored.dictMu.Unlock()
			if blocked {
				texts = <-done
			}
			ReleaseColumnReader(compressed)
			if blocked || !reflect.DeepEqual(texts, inputs) {
				t.Fatal("prepared compressed reader blocked on management or changed values")
			}
			reader := restored.GetCachedReader(ReadDecoded)
			for i, want := range inputs {
				if got := reader.GetValue(uint32(i)).String(); got != want {
					t.Fatalf("row %d changed", i)
				}
			}
			restored.Serialize(&after)
			if !bytes.Equal(before.Bytes(), after.Bytes()) {
				t.Fatal("decoded cache changed persisted data")
			}
			if restored.EvictDictionary() != 0 {
				t.Fatal("evicted active cache")
			}
			ReleaseColumnReader(reader)
			if restored.EvictDictionary() <= 0 {
				t.Fatal("idle decoded cache not reclaimed")
			}
			if restored.dictionary != "" || restored.textBuffer == nil {
				t.Fatal("eviction changed primary payload")
			}
			for i, want := range inputs {
				if got := restored.GetValue(uint32(i)).String(); got != want {
					t.Fatalf("row %d after eviction changed", i)
				}
			}
		})
	}
}

func TestStringReaderOperatorOwnership(t *testing.T) {
	inputs := []string{"report item one", "report item two"}
	var source StorageString
	source.prepare()
	for i, value := range inputs {
		source.scan(uint32(i), scm.NewString(value))
	}
	source.init(uint32(len(inputs)))
	for i, value := range inputs {
		source.build(uint32(i), scm.NewString(value))
	}
	source.finish()
	tx := new(TxContext)
	tx.beginQuery(nil, 1, "reader ownership")
	reader := newCachedColumnReaderTx(&source, tx, false, ReadDecoded|readOperatorOwned)
	tx.endQuery(1)
	if got := reader.GetValue(0).String(); got != inputs[0] || source.EvictDictionary() != 0 {
		t.Fatal("statement completion released an operator-owned reader")
	}
	ReleaseColumnReader(reader)
	if source.EvictDictionary() <= 0 {
		t.Fatal("operator completion did not release its dictionary")
	}
	// A parked transaction is also valid for explicit decoded preparation.
	reader = newCachedColumnReaderTx(&source, tx, false, ReadDecoded|readOperatorOwned)
	if got := reader.GetValue(1).String(); got != inputs[1] || source.EvictDictionary() != 0 {
		t.Fatal("inactive transaction lost an explicitly prepared reader")
	}
	ReleaseColumnReader(reader)
	if source.EvictDictionary() <= 0 {
		t.Fatal("inactive transaction's operator did not release its dictionary")
	}
}

// Historical V4 writer exists only to construct read-compatibility fixtures.
func (s *StorageString) serializeStringBuffer(w io.Writer) {
	write := func(v any) {
		if err := binary.Write(w, binary.LittleEndian, v); err != nil {
			panic(err)
		}
	}
	write(uint8(20))
	var nodict uint8
	if s.nodict {
		nodict = 1
	}
	write(nodict)
	write(uint8(s.format))
	write(uint8(4))
	write([4]byte{})
	write(uint8(2))
	count := s.values.count
	if s.nodict {
		count = s.starts.count
	}
	write(uint64(count))
	s.values.Serialize(w)
	s.starts.Serialize(w)
	s.lens.Serialize(w)
	write(uint32(s.starts.count))
	// Preserve V4 framing without retaining its redundant runtime offset array.
	for i := uint32(0); uint64(i) < s.starts.count; i++ {
		start := s.starts.GetValueUInt(i)
		if s.nodict && s.starts.hasNull && start == s.starts.null {
			write(uint32(0))
		} else {
			write(uint32(int64(start) + s.starts.offset))
		}
	}
	s.textBuffer.WriteTo(w)
}

// Historical disk baseline for the codec benchmark, never a production writer.
func (s *StorageString) serializeRawFixtureV3(w io.Writer) {
	var nodict byte
	count := s.values.count
	if s.nodict {
		nodict = 1
		count = s.starts.count
	}
	w.Write([]byte{20, nodict, byte(s.format), 3, 0, 0, 0, 0, 0})
	binary.Write(w, binary.LittleEndian, count)
	s.values.Serialize(w)
	s.starts.Serialize(w)
	s.lens.Serialize(w)
	binary.Write(w, binary.LittleEndian, uint64(len(s.dictionary)))
	io.WriteString(w, s.dictionary)
}

func TestStorageStringV5RejectsMalformed(t *testing.T) {
	for _, kind := range []string{"count", "format", "truncated", "extent"} {
		t.Run(kind, func(t *testing.T) {
			s := buildStringColumn([]string{"report one", "report two"})
			if kind == "extent" {
				s.lens.offset += 1000000
			}
			var wire bytes.Buffer
			s.Serialize(&wire)
			data := append([]byte(nil), wire.Bytes()...)
			switch kind {
			case "count":
				binary.LittleEndian.PutUint64(data[8:16], 999)
			case "format":
				data[2] = 255
			case "truncated":
				data = data[:len(data)-1]
			}
			defer func() {
				if recover() == nil {
					t.Fatal("accepted malformed V5 payload")
				}
			}()
			var restored StorageString
			restored.Deserialize(bytes.NewReader(data[1:]))
		})
	}
}

func TestStorageStringSpecialBoundaryDoesNotPinCache(t *testing.T) {
	for _, inputs := range [][]string{
		{strings.Repeat("deadbeef", 100), strings.Repeat("cafebabe", 100)},
		{"YWJjZGVmZ2hpamtsbW5vcA==", "cXJzdHV2d3h5ejAxMjM0NQ=="},
	} {
		s := buildStringColumn(inputs)
		s.CompressDictionary()
		key := (colGetter{source: s}).persistentValue(1, nil)
		if key.GetTag() != scm.TagString || key.String() != inputs[1] {
			t.Fatal("wrong owned boundary")
		}
		cache := s.dictCache.Load()
		if cache == nil {
			t.Fatal("missing decoded cache")
		}
		base := uintptr(unsafe.Pointer(unsafe.StringData(cache.text)))
		ptr := uintptr(unsafe.Pointer(unsafe.StringData(key.String())))
		if ptr >= base && ptr < base+uintptr(len(cache.text)) {
			t.Fatal("boundary still borrows cache")
		}
		if s.EvictDictionary() <= 0 || key.String() != inputs[1] {
			t.Fatal("boundary failed after eviction")
		}
	}
}

func TestStorageStringPreparedFilterReaderIsConsumed(t *testing.T) {
	Init(scm.Globalenv)
	const dbName = "test_prepared_string_filter"
	databases.Remove(dbName)
	t.Cleanup(func() { databases.Remove(dbName) })
	CreateDatabase(dbName, true)
	tbl, _ := CreateTable(dbName, "items", Memory, true)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("label", "TEXT", nil, nil)
	rows := make([][]scm.Scmer, 2048)
	for i := range rows {
		rows[i] = []scm.Scmer{scm.NewInt(int64(i)), scm.NewString(fmt.Sprintf("report label %04d", i))}
	}
	tbl.Insert([]string{"id", "label"}, rows, nil, scm.NewNil(), false, nil)
	if result := GetDatabase(dbName).rebuild(true, false, true); len(result.errors) > 0 {
		t.Fatal(result.errors)
	}
	tbl.Insert([]string{"id", "label"}, [][]scm.Scmer{{scm.NewInt(2048), scm.NewString("delta label")}}, nil, scm.NewNil(), false, nil)
	tx := NewTxContext(TxCursorStability)
	tx.beginQuery(nil, 71, "prepared string filter")
	defer tx.endQuery(71)
	condition := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer { return scm.NewBool(args[0].GetTag() == scm.TagString) })
	access := newScanAccessSchema(scanAccessConsumerScan, nil, -1)
	count := tbl.scan(tx, access, nil, []string{"label"}, condition, nil, scm.Globalenv.Vars[scm.Symbol("scan_count")], scm.NewInt(0), scm.Globalenv.Vars[scm.Symbol("+")], false)
	if count.Int() != int64(len(rows)+1) {
		t.Fatalf("prepared filter consumed compressed source: got %d", count.Int())
	}
	// The ordered scalar/bulk paths must use the same prepared representation.
	callback := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer { return args[1] })
	result := tbl.scan_order(tx, access, nil, []string{"label"}, condition, nil, nil, 0, 0, 1,
		[]string{"id"}, callback, scm.NewNil(), false, scm.NewNil(), nil, scm.NewNil())
	if result.IsNil() {
		t.Fatal("ordered filter bypassed prepared reader")
	}
	// Early-exit scans reuse an already admitted cache without decoding a cold one.
	shard := tbl.ActiveShards()[0]
	if _, found := shard.scanFirstRecord(scanAccess{}, []string{"label"}, condition, tx, nil, nil); !found {
		t.Fatal("first-record filter bypassed warm reader")
	}
	deltaCondition := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		return scm.NewBool(args[0].GetTag() == scm.TagString && args[0].String() == "delta label")
	})
	if id, found := shard.scanFirstRecord(scanAccess{}, []string{"label"}, deltaCondition, tx, nil, nil); !found || id != 2048 {
		t.Fatal("first-record prepared reader lost physical delta")
	}
	delta := tbl.scan_order(tx, access, nil, []string{"label"}, deltaCondition, nil, nil, 0, 0, 1,
		[]string{"id"}, callback, scm.NewNil(), false, scm.NewNil(), nil, scm.NewNil())
	if delta.IsNil() || delta.Int() != 2048 {
		t.Fatal("ordered prepared reader lost physical delta")
	}

}

func TestStorageStringFilterDecodingChoice(t *testing.T) {
	short := buildStringColumn([]string{"report label one", "report label two"})
	short.CompressDictionary()
	long := buildStringColumn([]string{strings.Repeat("long report one ", 100), strings.Repeat("long report two ", 100)})
	long.CompressDictionary()
	for _, tc := range []struct {
		name    string
		storage ColumnStorage
		access  scanAccess
		rows    uint32
		want    ColumnReadHints
	}{
		{"short full scan", short, scanAccess{}, 10000, ReadDecoded},
		{"large report full scan", long, scanAccess{}, 10000, ReadAuto},
		{"short point", short, runtimeScanAccess(analyzedBoundaries{{col: "label", matcher: EqualMatcher, lower: scm.NewString("report label one"), upper: scm.NewString("report label one")}}), 10000, ReadAuto},
		{"short range", short, runtimeScanAccess(analyzedBoundaries{{col: "label", matcher: RangeMatcher}}), 10000, ReadDecoded},
		{"bigram candidates", short, runtimeScanAccess(analyzedBoundaries{{col: "label", matcher: LikeMatcher}}), 10000, ReadAuto},
		{"small population", short, scanAccess{}, 1, ReadAuto},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := filterColumnReadHints(tc.storage, tc.access, "label", tc.rows); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestStorageStringProjectionPreparationFollowsDemand(t *testing.T) {
	s := buildStringColumn([]string{"short group key one", "short group key two"})
	s.CompressDictionary()
	tx := NewTxContext(TxCursorStability)
	tx.beginQuery(nil, 72, "selected projection")
	defer tx.endQuery(72)
	mapper := &ShardMapReducer{
		mainCols: []ColumnStorage{s}, mainBulkReaders: []ColumnReader{s.GetCachedReader(ReadCompressed)},
		mainValueFuncs: make([]scm.JITStorageGetValueFunc, 1), mainMultiFuncs: make([]scm.JITStorageGetValueMultiFunc, 1),
		currentTx: tx, mainCount: 2, directRead: true,
	}
	if s.dictCache.Load() != nil {
		t.Fatal("projection-only payload prepared before demand")
	}
	mapper.prefetchMainColumns([]uint32{1})
	if mapper.mainBulkValues[0].GetTag() != scm.TagString || mapper.mainBulkValues[0].String() != "short group key two" {
		t.Fatal("selected projection did not use prepared reader")
	}
	if s.dictCache.Load() == nil || s.EvictDictionary() != 0 {
		t.Fatal("selected projection cache is not leased")
	}
	before := s.dictCache.Load().users
	mapper.prefetchMainColumns([]uint32{0})
	if s.dictCache.Load().users != before {
		t.Fatal("repeated selected batch admitted another reader")
	}
}
