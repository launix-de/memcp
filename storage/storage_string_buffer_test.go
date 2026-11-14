/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "time"
import "fmt"
import "io"
import "bytes"
import "testing"
import "strings"
import "reflect"
import "runtime"
import "encoding/binary"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"
import "github.com/pierrec/lz4/v4"

func TestStorageStringBuffer(t *testing.T) {
	codec := scm.StringCodecConcat
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
		if s.reverseMap != nil || s.sb.Len() != 0 {
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
	codec := scm.StringCodecConcat
	s := buildStorageString(values)
	s.CompressDictionary()
	overlay := OverlayBlob{Base: s}
	target := make([]scm.Scmer, len(values))
	overlay.GetValueRange(0, uint32(len(values)), target, 1)
	if !target[0].IsBufferString() || !target[1].IsString() || target[1].String() != "!escaped" || !target[2].IsNil() {
		t.Fatal("overlay broke compressed pass-through")
	}
	if got := testing.AllocsPerRun(30, func() { overlay.GetValueRange(0, uint32(len(values)), target, 1) }); got != 0 {
		t.Fatalf("codec %d overlay allocates %v", codec, got)
	}
	// Prepared batches may mix ordinary values with compressed handles. A
	// decoder is still needed when the first row itself is an ordinary string.
	mixed := []scm.Scmer{scm.NewString("plain"), s.GetValue(1), scm.NewNil()}
	overlay.resolveBlobBatch(mixed, len(mixed), 1)
	if mixed[0].String() != "plain" || !mixed[1].IsString() || mixed[1].String() != "!escaped" || !mixed[2].IsNil() {
		t.Fatal("mixed batch lost compressed pass-through or literal escaping")
	}
	decoded := overlay.GetCachedReader(ReadDecoded)
	decoded.GetValueRange(0, uint32(len(values)), target, 1)
	if target[0].String() != "ordinary Prüfreport" || target[1].String() != "!escaped" || !target[2].IsNil() {
		t.Fatal("decoded reader stripped blob overlay")
	}

	ReleaseColumnReader(decoded)
	clear(target)

}

func TestStorageStringBufferJITValueType(t *testing.T) {
	codec := scm.StringCodecConcat
	s := buildStorageString([]scm.Scmer{scm.NewString("Alpha report"), scm.NewString("Beta report")})
	s.CompressDictionary()
	want := uint8(scm.TagConcat)
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
	if empty.JITValueType() != scm.TagConcat {
		t.Fatal("mixed empty handles promised uniform tag")
	}
}

func TestStorageStringBufferAssignments(t *testing.T) {
	codec := scm.StringCodecConcat
	texts := []string{"1e2", "1.5", "invalid number"}
	buffer, offsets := scm.BuildStringConcatBuffer(texts)
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
			got, want := outcome(scm.NewBufferString(buffer, offsets[i])), outcome(scm.NewString(text))
			if got != want {
				t.Fatalf("codec %d %s assignment %q got %q want %q", codec, typ, text, got, want)
			}
		}
	}
}

var stringBufferBenchBool bool
var stringBufferBenchIndex *bigramIndex
var stringBufferBenchValue scm.Scmer

// Every historical raw version remains readable and saves as version 6.
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
			if loaded.textBuffer == nil || loaded.dictionary != "" {
				t.Fatal("historical raw dictionary not normalized before publication")
			}
			var current bytes.Buffer
			loaded.Serialize(&current)
			if current.Bytes()[3] != 6 || binary.LittleEndian.Uint64(current.Bytes()[8:16]) != uint64(len(values)) {
				t.Fatal("writer did not emit version 6")
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

// Historical V4 writer exists only to construct read-compatibility fixtures.
func (s *StorageString) serializeStringBuffer(w io.Writer) {
	copy := *s
	copy.expandConcatDictionary()
	s = &copy
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
	writeLegacyLZ4Buffer(w, s.dictionary)
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
			s.CompressDictionary()
			if kind == "extent" {
				s.starts.offset += 1000000
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
	condition := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer { return scm.NewBool(args[0].IsString()) })
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
		return scm.NewBool(args[0].IsString() && args[0].String() == "delta label")
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
		{"short full scan", short, scanAccess{}, 10000, ReadCompressed},
		{"large report full scan", long, scanAccess{}, 10000, ReadCompressed},
		{"short point", short, runtimeScanAccess(analyzedBoundaries{{col: "label", matcher: EqualMatcher, lower: scm.NewString("report label one"), upper: scm.NewString("report label one")}}), 10000, ReadCompressed},
		{"short range", short, runtimeScanAccess(analyzedBoundaries{{col: "label", matcher: RangeMatcher}}), 10000, ReadCompressed},
		{"bigram candidates", short, runtimeScanAccess(analyzedBoundaries{{col: "label", matcher: LikeMatcher}}), 10000, ReadCompressed},
		{"small population", short, scanAccess{}, 1, ReadCompressed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := filterColumnReadHints(tc.storage, tc.access, "label", tc.rows); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestMaterializedComputedReaderPreparesOutputOnly(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tmaterializedreaderdemand")
	tbl.CreateColumn("label", "TEXT", nil, nil)
	tbl.Insert([]string{"id", "label"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewString("input alpha")},
		{scm.NewInt(2), scm.NewString("input beta")},
	}, nil, scm.NewNil(), false, nil)
	RebuildTable(tbl, true, false)
	shard := tbl.ActiveShards()[0]
	input := buildStorageString([]scm.Scmer{scm.NewString(strings.Repeat("input alpha ", 10000)), scm.NewString(strings.Repeat("input beta ", 10000))})
	input.CompressDictionary()
	main := buildStorageString([]scm.Scmer{scm.NewString("output alpha"), scm.NewString("output beta")})
	main.CompressDictionary()
	shard.mu.Lock()
	shard.columns["label"] = input
	shard.mu.Unlock()
	proxy := &StorageComputeProxy{shard: shard, main: main, count: 2, compressed: true,
		inputCols: []string{"label"}, delta: make(map[uint32]scm.Scmer),
		computor: scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[0] })}
	reader := newCachedColumnReaderTx(proxy, nil, false, ReadDecoded|readOperatorOwned)
	if input.ReadCount() != 0 {
		t.Fatal("materialized output prepared an unused input instead of its actual main")
	}
	values := make([]scm.Scmer, 2)
	reader.GetValueRange(0, 2, values, 1)
	if values[0].String() != "output alpha" || values[1].String() != "output beta" {
		t.Fatal("materialized range bypassed its prepared output reader")
	}
	reader.GetValueMulti([]uint32{1, 0}, values, 1)
	if values[0].String() != "output beta" || values[1].String() != "output alpha" {
		t.Fatal("materialized row permutation lost its output reader")
	}
	proxy.mu.Lock()
	proxy.delta[1] = scm.NewString("own update")
	proxy.mu.Unlock()
	if reader.GetValue(1).String() != "own update" {
		t.Fatal("prepared materialized main hid a newer delta value")
	}
	replacement := buildStorageString([]scm.Scmer{scm.NewString("replacement alpha"), scm.NewString("replacement beta")})
	replacement.CompressDictionary()
	proxy.mu.Lock()
	proxy.main = replacement
	proxy.mu.Unlock()
	if reader.GetValue(0).String() != "replacement alpha" {
		t.Fatal("prepared reader reused a stale replaced main")
	}
	shard.mu.Lock()
	shard.deltaColumns["label"] = 0
	shard.publishDeltaRows([][]scm.Scmer{{scm.NewString("appended label")}})
	shard.mu.Unlock()
	if reader.GetValue(2).String() != "appended label" {
		t.Fatal("materialized reader lost existing appended-row repair or prepared unused input")
	}
	ReleaseColumnReader(reader)

}

type failingPreparedTextStorage struct{ *StorageString }

func (s *failingPreparedTextStorage) GetCachedReader(ColumnReadHints) ColumnReader {
	panic("second prepared input failed")
}

func TestComputedReaderSetupFailureReleasesEarlierChildren(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tcomputedsetupownership")
	tbl.CreateColumn("first", "TEXT", nil, nil)
	tbl.CreateColumn("second", "TEXT", nil, nil)
	tbl.Insert([]string{"id", "first", "second"}, [][]scm.Scmer{{scm.NewInt(1), scm.NewString("first"), scm.NewString("second")}}, nil, scm.NewNil(), false, nil)
	RebuildTable(tbl, true, false)
	shard := tbl.ActiveShards()[0]
	first := buildStorageString([]scm.Scmer{scm.NewString("first")})
	first.CompressDictionary()
	warm := first.GetCachedReader(ReadDecoded)
	ReleaseColumnReader(warm)
	shard.mu.Lock()
	shard.columns["first"] = first
	shard.columns["second"] = &failingPreparedTextStorage{buildStorageString([]scm.Scmer{scm.NewString("second")})}
	shard.mu.Unlock()
	proxy := &StorageComputeProxy{shard: shard, inputCols: []string{"first", "second"}}
	index := &StorageIndex{t: shard, Cols: []string{".mapped"}, ColMapCols: [][]string{{"first", "second"}},
		ColMapFn: []scm.Scmer{scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[0] })}}
	for _, prepare := range []func(){
		func() { newCachedColumnReaderTx(proxy, nil, false, ReadDecoded|readOperatorOwned) },
		func() {
			shard.mu.RLock()
			defer shard.mu.RUnlock()
			index.buildGetters(nil, nil)
		},
	} {
		warm := first.GetCachedReader(ReadDecoded)
		ReleaseColumnReader(warm)
		func() {
			defer func() {
				if recovered := recover(); recovered != "second prepared input failed" {
					t.Fatalf("wrong setup failure: %v", recovered)
				}
			}()
			prepare()
		}()

	}
}

func TestMaterializedReaderRepairPreservesExistingShardRead(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tmaterializedreadercapability")
	tbl.CreateColumn("label", "TEXT", nil, nil)
	tbl.Insert([]string{"id", "label"}, [][]scm.Scmer{
		{scm.NewInt(1), scm.NewString("first")}, {scm.NewInt(2), scm.NewString("second")},
	}, nil, scm.NewNil(), false, nil)
	RebuildTable(tbl, true, false)
	shard := tbl.ActiveShards()[0]
	main := buildStorageString([]scm.Scmer{scm.NewString("cached first"), scm.NewString("cached second")})
	main.CompressDictionary()
	proxy := &StorageComputeProxy{shard: shard, main: main, count: 2, compressed: true,
		inputCols: []string{"label"}, delta: make(map[uint32]scm.Scmer),
		computor: scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[0] })}
	shard.mu.Lock()
	shard.deltaColumns["label"] = 0
	shard.publishDeltaRows([][]scm.Scmer{{scm.NewString("appended")}})
	shard.mu.Unlock()
	tx := NewTxContext(TxCursorStability)
	tx.querySeq.Store(19)
	tx.queryActive.Store(true)
	shard.mu.RLock()
	reader := newCachedColumnReaderTx(proxy, tx, true, ReadDecoded|readOperatorOwned)
	writerDone := make(chan struct{})
	go func() {
		shard.mu.Lock()
		shard.mu.Unlock()
		close(writerDone)
	}()
	deadline := time.Now().Add(time.Second)
	for shard.mu.TryRLock() {
		shard.mu.RUnlock()
		if time.Now().After(deadline) {
			shard.mu.RUnlock()
			ReleaseColumnReader(reader)
			t.Fatal("writer did not queue behind the retained read capability")
		}
		runtime.Gosched()
	}
	done := make(chan scm.Scmer, 1)
	go func() { done <- reader.GetValue(2) }()
	var value scm.Scmer
	blocked := false
	select {
	case value = <-done:
	case <-time.After(100 * time.Millisecond):
		blocked = true
	}
	shard.mu.RUnlock()
	<-writerDone
	if blocked {
		value = <-done
	}
	ReleaseColumnReader(reader)
	if blocked || value.String() != "appended" || len(tx.queryReaderReleases) != 0 {
		t.Fatal("point repair re-entered an owned shard read or added row-bound statement leases")
	}
	tx.endQuery(19)
}

// The generated scalar entry must pass the native reader's full capability
// signature; an outdated Go-call ABI can leak arbitrary register contents into
// the lock/representation arguments even when a materialized row looks correct.
type jitCapabilityTextStorage struct {
	*StorageString
	locked bool
	hints  ColumnReadHints
	calls  int
}

func (s *jitCapabilityTextStorage) GetCachedReaderTx(_ *TxContext, locked bool, hints ColumnReadHints) ColumnReader {
	s.locked, s.hints = locked, hints
	s.calls++
	return s.StorageString.GetCachedReader(hints)
}

func TestComputedScalarJITPreservesDefaultReadCapabilities(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "tcomputedjitcapability")
	tbl.CreateColumn("label", "TEXT", nil, nil)
	tbl.Insert([]string{"id", "label"}, [][]scm.Scmer{{scm.NewInt(1), scm.NewString("native value")}}, nil, scm.NewNil(), false, nil)
	RebuildTable(tbl, true, false)
	shard := tbl.ActiveShards()[0]
	source := &jitCapabilityTextStorage{StorageString: buildStorageString([]scm.Scmer{scm.NewString("native value")})}
	shard.mu.Lock()
	shard.columns["label"] = source
	shard.mu.Unlock()
	proxy := &StorageComputeProxy{shard: shard, count: 1, inputCols: []string{"label"}, delta: make(map[uint32]scm.Scmer),
		computor: scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[0] })}
	jitGet, cleanup := jitBuildGetValueFunc(t, proxy, false)
	defer cleanup()
	value := jitGet(0)
	if value.String() != "native value" || source.calls != 1 || source.locked || source.hints != ReadAuto {
		t.Fatalf("native scalar ABI lost its default read capability: value=%v calls=%d locked=%v hints=%d", value, source.calls, source.locked, source.hints)
	}
}

// RecSet residuals consume only visible main candidates through the existing
// bulk reader. Delta NULLs retain native visibility without forcing a decoded
// cache or passing lazy compressed handles to every scalar text predicate.
func TestRecSetResidualPreparesVisibleTextBatch(t *testing.T) {
	tbl := setupScanParallelTestTable(t, "recset_prepared_text")
	tbl.CreateColumn("label", "TEXT", nil, nil)
	rows := make([][]scm.Scmer, 64)
	for i := range rows {
		label := scm.NewString(fmt.Sprintf("rejected Gerätebericht %d", i))
		if i%4 == 0 {
			label = scm.NewString(fmt.Sprintf("accepted Gerätebericht %d", i))
		}
		if i == 7 {
			label = scm.NewNil()
		}
		rows[i] = []scm.Scmer{scm.NewInt(int64(i)), label}
	}
	tbl.Insert([]string{"id", "label"}, rows, nil, scm.NewNil(), false, nil)
	RebuildTable(tbl, true, false)
	shard := tbl.ActiveShards()[0]
	stored := shard.getColumnStorageOrPanic("label", false, nil)
	if overlay, ok := stored.(*OverlayBlob); ok {
		stored = overlay.Base
	}
	text, ok := stored.(*StorageString)
	if !ok || text.textBuffer == nil || !text.bufferHandles {
		t.Fatal("fixture must provide a cold compressed text generation")
	}
	tracked := &observedConcatReads{StorageString: text}
	shard.mu.Lock()
	shard.columns["label"] = tracked
	shard.deletions.Set(0, true)
	shard.mu.Unlock()
	tbl.Insert([]string{"id", "label"}, [][]scm.Scmer{
		{scm.NewInt(64), scm.NewString("accepted delta")},
		{scm.NewInt(65), scm.NewNil()},
	}, nil, scm.NewNil(), false, nil)
	priorReads := tracked.reads.Load()
	condition := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {

		if values[1].Int() == 0 {
			t.Fatal("deleted main row reached mixed text/numeric residual")
		}
		if id := values[1].Int(); id < 64 && !values[0].IsNil() &&
			!strings.HasSuffix(values[0].String(), " "+fmt.Sprint(id)) {
			t.Fatal("mixed residual columns came from different main rows")
		}
		return scm.NewBool(!values[0].IsNil() && strings.HasPrefix(values[0].String(), "accepted"))
	})
	got := tbl.scanRecSet(nil, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, []string{"label", "id"}, condition)
	if got.count != 16 || tracked.reads.Load()-priorReads != 63 {
		t.Fatalf("count=%d main reads=%d", got.count, tracked.reads.Load()-priorReads)
	}
	// A transaction-local delete must also be excluded before touching its
	// main value, without changing the shared deletion bitmap.
	tx := NewTxContext(TxACID)
	defer tx.Rollback()
	tx.AddToDeleteMask(shard, 4)
	priorReads = tracked.reads.Load()
	acid := tbl.scanRecSet(tx, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, []string{"label", "id"}, condition)
	if acid.count != 15 || tracked.reads.Load()-priorReads != 62 || shard.deletions.Get(4) {
		t.Fatalf("ACID count=%d main reads=%d shared delete=%v", acid.count, tracked.reads.Load()-priorReads, shard.deletions.Get(4))
	}
}

func writeLegacyLZ4Buffer(w io.Writer, text string) {
	write := func(v any) {
		if err := binary.Write(w, binary.LittleEndian, v); err != nil {
			panic(err)
		}
	}
	var payload []byte
	bounds := []uint32{0}
	var flags []byte
	var compressor lz4.Compressor
	tmp := make([]byte, lz4.CompressBlockBound(65536))
	for start := 0; start < len(text); start += 65536 {
		chunk := text[start:min(start+65536, len(text))]
		n, err := compressor.CompressBlock([]byte(chunk), tmp)
		if err != nil {
			panic(err)
		}
		if n == 0 || n >= len(chunk) {
			payload = append(payload, chunk...)
			flags = append(flags, 1)
		} else {
			payload = append(payload, tmp[:n]...)
			flags = append(flags, 0)
		}
		bounds = append(bounds, uint32(len(payload)))
	}
	write(uint8(2))
	write(uint32(len(text)))
	write(uint64(len(payload)))
	w.Write(payload)
	write(uint32(len(flags)))
	write(bounds)
	w.Write(flags)
}

func TestConcatReadersDoNotCreateDecodedOwners(t *testing.T) {
	inputs := []scm.Scmer{scm.NewString("shared query prefix alpha shared suffix"), scm.NewString("shared query prefix beta shared suffix"), scm.NewNil(), scm.NewString("")}
	source := buildStringCompressionFixture(inputs)
	before := GlobalCache.Stat()
	var retained scm.Scmer
	for _, hint := range []ColumnReadHints{ReadAuto, ReadCompressed, ReadDecoded} {
		reader := source.GetCachedReader(hint)
		got := make([]scm.Scmer, len(inputs)*2)
		if n := testing.AllocsPerRun(30, func() { reader.GetValueRange(0, uint32(len(inputs)), got, 2) }); n != 0 {
			t.Fatalf("hint %d allocated %v", hint, n)
		}
		for i, want := range inputs {
			if !scm.Equal(got[i*2], want) {
				t.Fatalf("hint %d row %d", hint, i)
			}
		}
		reader.GetValueMulti([]uint32{3, 1, 0, 2}, got, 1)
		if got[0].String() != "" || got[1].String() != inputs[1].String() || !got[3].IsNil() {
			t.Fatal("permutation changed")
		}
		retained = got[1]
		ReleaseColumnReader(reader)
	}
	after := GlobalCache.Stat()
	if before.CountByType[TypeStringDict] != after.CountByType[TypeStringDict] || before.SizeByType[TypeStringDict] != after.SizeByType[TypeStringDict] {
		t.Fatal("concat reader admitted a decoded cache")
	}
	source = nil
	runtime.GC()
	if retained.String() != inputs[1].String() {
		t.Fatal("handle lost its immutable owner")
	}
}

func TestLegacyV5BlockMigrationWithNulls(t *testing.T) {
	values := make([]scm.Scmer, 151)
	for i := 0; i < 150; i++ {
		values[i] = scm.NewString(fmt.Sprintf("legacy trace %04d Prüfbericht %s", i, strings.Repeat("abc0123", 1024)))
	}
	source := buildStorageString(values)
	var wire bytes.Buffer
	var nodict uint8
	if source.nodict {
		nodict = 1
	}
	wire.Write([]byte{20, nodict, byte(source.format), 5, 0, 0, 0, 0})
	binary.Write(&wire, binary.LittleEndian, uint64(len(values)))
	source.values.Serialize(&wire)
	source.starts.Serialize(&wire)
	source.lens.Serialize(&wire)
	writeLegacyLZ4Buffer(&wire, source.dictionary)
	var migrated StorageString
	migrated.Deserialize(bytes.NewReader(wire.Bytes()[1:]))
	if migrated.compressed || len(migrated.compressedDict) != 0 || migrated.dictionary != "" || migrated.textBuffer == nil {
		t.Fatal("legacy codec survived publication")
	}
	for i, want := range values {
		if !scm.Equal(migrated.GetValue(uint32(i)), want) {
			t.Fatalf("legacy row %d", i)
		}
	}
	var current bytes.Buffer
	migrated.Serialize(&current)
	if current.Bytes()[3] != 6 {
		t.Fatal("writer emitted old codec")
	}
	var restarted StorageString
	restarted.Deserialize(bytes.NewReader(current.Bytes()[1:]))
	for i, want := range values {
		if !scm.Equal(restarted.GetValue(uint32(i)), want) {
			t.Fatalf("restart row %d", i)
		}
	}
}

// This fixture-owned observer verifies skipped-row access without requiring
// production readers to update shared telemetry from scalar loops.
type observedConcatReads struct {
	*StorageString
	reads atomic.Uint64
}

func (s *observedConcatReads) GetCachedReader(ColumnReadHints) ColumnReader { return s }
func (s *observedConcatReads) GetValue(row uint32) scm.Scmer {
	s.reads.Add(1)
	return s.StorageString.GetValue(row)
}
func (s *observedConcatReads) GetValueRange(row, count uint32, target []scm.Scmer, stride int) {
	s.reads.Add(uint64(count))
	s.StorageString.GetValueRange(row, count, target, stride)
}
func (s *observedConcatReads) GetValueMulti(rows []uint32, target []scm.Scmer, stride int) {
	s.reads.Add(uint64(len(rows)))
	s.StorageString.GetValueMulti(rows, target, stride)
}
