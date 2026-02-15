package storage

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/launix-de/memcp/scm"
)

// buildViaCompression runs the full compression pipeline:
// StorageSCMER.prepare → scan → proposeCompression → (new type).prepare → scan → init → build → finish
func buildViaCompression(n int, gen func(int) scm.Scmer) ColumnStorage {
	s := new(StorageSCMER)
	s.prepare()
	for i := 0; i < n; i++ {
		s.scan(uint(i), gen(i))
	}
	col := s.proposeCompression(uint(n))
	if col == nil {
		// no compression proposed, use StorageSCMER directly
		s.init(uint(n))
		for i := 0; i < n; i++ {
			s.build(uint(i), gen(i))
		}
		s.finish()
		return s
	}
	// run proposed compression through its pipeline
	for {
		col.prepare()
		for i := 0; i < n; i++ {
			col.scan(uint(i), gen(i))
		}
		col2 := col.proposeCompression(uint(n))
		if col2 == nil {
			break
		}
		col = col2
	}
	col.init(uint(n))
	for i := 0; i < n; i++ {
		col.build(uint(i), gen(i))
	}
	col.finish()
	return col
}

// serializeDeserialize round-trips a storage through Serialize/Deserialize.
func serializeDeserialize(col ColumnStorage) ColumnStorage {
	var buf bytes.Buffer
	col.Serialize(&buf)
	data := buf.Bytes()
	if len(data) == 0 {
		return nil
	}
	magic := data[0]
	reader := bytes.NewReader(data[1:]) // first byte already consumed by dispatch
	t, ok := storages[magic]
	if !ok {
		return nil
	}
	newCol := reflect.New(t).Interface().(ColumnStorage)
	newCol.Deserialize(reader)
	return newCol
}

// verifyStorage checks that every value in the storage matches the generator.
func verifyStorage(t *testing.T, col ColumnStorage, n int, gen func(int) scm.Scmer) {
	t.Helper()
	for i := 0; i < n; i++ {
		got := col.GetValue(uint(i))
		want := gen(i)
		if !scm.Equal(got, want) {
			t.Fatalf("GetValue(%d) = %v, want %v (storage type %T)", i, got, want, col)
		}
	}
}

// --- StorageInt tests ---

func TestStorageIntPipeline(t *testing.T) {
	// Build StorageInt directly (bypass proposeCompression heuristics)
	n := 200
	gen := func(i int) scm.Scmer { return scm.NewInt(int64(i * 3)) }
	col := new(StorageInt)
	col.prepare()
	for i := 0; i < n; i++ {
		col.scan(uint(i), gen(i))
	}
	col.init(uint(n))
	for i := 0; i < n; i++ {
		col.build(uint(i), gen(i))
	}
	col.finish()
	verifyStorage(t, col, n, gen)
}

func TestStorageIntWithNull(t *testing.T) {
	// Build StorageInt directly with NULL values
	n := 100
	gen := func(i int) scm.Scmer {
		if i%10 == 0 {
			return scm.NewNil()
		}
		return scm.NewInt(int64(i))
	}
	col := new(StorageInt)
	col.prepare()
	for i := 0; i < n; i++ {
		col.scan(uint(i), gen(i))
	}
	col.init(uint(n))
	for i := 0; i < n; i++ {
		col.build(uint(i), gen(i))
	}
	col.finish()
	verifyStorage(t, col, n, gen)
}

func TestStorageIntNegative(t *testing.T) {
	n := 100
	gen := func(i int) scm.Scmer { return scm.NewInt(int64(i) - 50) }
	col := new(StorageInt)
	col.prepare()
	for i := 0; i < n; i++ {
		col.scan(uint(i), gen(i))
	}
	col.init(uint(n))
	for i := 0; i < n; i++ {
		col.build(uint(i), gen(i))
	}
	col.finish()
	verifyStorage(t, col, n, gen)
}

func TestStorageIntLargeValues(t *testing.T) {
	n := 50
	gen := func(i int) scm.Scmer { return scm.NewInt(int64(i)*1000000000 + 999999999) }
	col := buildViaCompression(n, gen)
	verifyStorage(t, col, n, gen)
}

func TestStorageIntSerializeRoundtrip(t *testing.T) {
	n := 100
	gen := func(i int) scm.Scmer {
		if i == 50 {
			return scm.NewNil()
		}
		return scm.NewInt(int64((i*17 + 5) % 300))
	}
	col := buildViaCompression(n, gen)
	col2 := serializeDeserialize(col)
	if col2 == nil {
		t.Fatal("deserialization returned nil")
	}
	verifyStorage(t, col2, n, gen)
}

// --- StorageFloat tests ---

func TestStorageFloatPipeline(t *testing.T) {
	n := 100
	gen := func(i int) scm.Scmer { return scm.NewFloat(float64(i) * math.Pi) }
	col := new(StorageFloat)
	col.prepare()
	for i := 0; i < n; i++ {
		col.scan(uint(i), gen(i))
	}
	col.init(uint(n))
	for i := 0; i < n; i++ {
		col.build(uint(i), gen(i))
	}
	col.finish()
	verifyStorage(t, col, n, gen)
}

func TestStorageFloatWithNaN(t *testing.T) {
	n := 50
	gen := func(i int) scm.Scmer {
		if i%5 == 0 {
			return scm.NewNil()
		}
		return scm.NewFloat(float64(i) * 1.5)
	}
	col := buildViaCompression(n, gen)
	verifyStorage(t, col, n, gen)
}

func TestStorageFloatSerializeRoundtrip(t *testing.T) {
	n := 50
	gen := func(i int) scm.Scmer {
		if i%7 == 0 {
			return scm.NewNil()
		}
		return scm.NewFloat(float64(i) * 2.718)
	}
	col := buildViaCompression(n, gen)
	col2 := serializeDeserialize(col)
	if col2 == nil {
		t.Fatal("deserialization returned nil")
	}
	verifyStorage(t, col2, n, gen)
}

// --- StorageString tests ---

func TestStorageStringDictPipeline(t *testing.T) {
	vals := []string{"alpha", "beta", "gamma", "delta"}
	n := 200
	gen := func(i int) scm.Scmer { return scm.NewString(vals[i%4]) }
	col := buildViaCompression(n, gen)
	if _, ok := col.(*StorageString); !ok {
		t.Fatalf("expected *StorageString, got %T", col)
	}
	verifyStorage(t, col, n, gen)
}

func TestStorageStringNodictPipeline(t *testing.T) {
	// Many unique strings → nodict mode
	n := 200
	gen := func(i int) scm.Scmer {
		return scm.NewString(fmt.Sprintf("unique_value_%d", i))
	}
	col := buildViaCompression(n, gen)
	if _, ok := col.(*StorageString); !ok {
		t.Fatalf("expected *StorageString, got %T", col)
	}
	verifyStorage(t, col, n, gen)
}

func TestStorageStringWithNull(t *testing.T) {
	n := 100
	gen := func(i int) scm.Scmer {
		if i%5 == 0 {
			return scm.NewNil()
		}
		return scm.NewString(fmt.Sprintf("val_%d", i%10))
	}
	col := buildViaCompression(n, gen)
	verifyStorage(t, col, n, gen)
}

func TestStorageStringSerializeRoundtrip(t *testing.T) {
	vals := []string{"Berlin", "Munich", "Hamburg", "Cologne"}
	n := 100
	gen := func(i int) scm.Scmer {
		if i%10 == 0 {
			return scm.NewNil()
		}
		return scm.NewString(vals[i%4])
	}
	col := buildViaCompression(n, gen)
	col2 := serializeDeserialize(col)
	if col2 == nil {
		t.Fatal("deserialization returned nil")
	}
	verifyStorage(t, col2, n, gen)
}

// --- StorageDecimal tests ---

func TestDecimalHelpers(t *testing.T) {
	// trailingZeroPow10
	cases := []struct {
		v   int64
		exp int8
	}{
		{0, math.MaxInt8},
		{100, 2},
		{1550, 1},
		{7, 0},
		{-100, 2},
		{-7, 0},
		{1000000, 6},
	}
	for _, tc := range cases {
		got := trailingZeroPow10(tc.v)
		if got != tc.exp {
			t.Errorf("trailingZeroPow10(%d) = %d, want %d", tc.v, got, tc.exp)
		}
	}

	// isCloseToInt
	if !isCloseToInt(3.0) {
		t.Error("isCloseToInt(3.0) should be true")
	}
	if isCloseToInt(3.5) {
		t.Error("isCloseToInt(3.5) should be false")
	}
	if !isCloseToInt(1e15) {
		t.Error("isCloseToInt(1e15) should be true")
	}

	// detectFloatScale
	scaleTests := []struct {
		f   float64
		exp int8
	}{
		{0.0, math.MaxInt8},
		{100.0, 2},
		{7.0, 0},
		{3.5, -1},
		{12.57, -2},
	}
	for _, tc := range scaleTests {
		got := detectFloatScale(tc.f)
		if got != tc.exp {
			t.Errorf("detectFloatScale(%v) = %d, want %d", tc.f, got, tc.exp)
		}
	}
}

func TestStorageDecimalPipeline(t *testing.T) {
	n := 100
	gen := func(i int) scm.Scmer { return scm.NewFloat(float64(i) * 0.01) }
	col := buildViaCompression(n, gen)
	if _, ok := col.(*StorageDecimal); !ok {
		t.Fatalf("expected *StorageDecimal, got %T", col)
	}
	// verify with float tolerance
	for i := 0; i < n; i++ {
		got := col.GetValue(uint(i))
		want := float64(i) * 0.01
		if got.IsNil() {
			t.Fatalf("GetValue(%d) = nil, want %v", i, want)
		}
		diff := math.Abs(got.Float() - want)
		if diff > 1e-9 {
			t.Fatalf("GetValue(%d) = %v, want %v (diff=%e)", i, got.Float(), want, diff)
		}
	}
}

func TestStorageDecimalWithNull(t *testing.T) {
	n := 100
	gen := func(i int) scm.Scmer {
		if i%10 == 0 {
			return scm.NewNil()
		}
		return scm.NewFloat(float64(i) * 0.1)
	}
	col := buildViaCompression(n, gen)
	for i := 0; i < n; i++ {
		got := col.GetValue(uint(i))
		if i%10 == 0 {
			if !got.IsNil() {
				t.Fatalf("GetValue(%d) = %v, want nil", i, got)
			}
		} else {
			want := float64(i) * 0.1
			diff := math.Abs(got.Float() - want)
			if diff > 1e-9 {
				t.Fatalf("GetValue(%d) = %v, want %v", i, got.Float(), want)
			}
		}
	}
}

func TestStorageDecimalIntegerMultiples(t *testing.T) {
	// multiples of 100 → scaleExp=2
	n := 50
	gen := func(i int) scm.Scmer { return scm.NewInt(int64(i) * 100) }
	col := buildViaCompression(n, gen)
	if dec, ok := col.(*StorageDecimal); ok {
		if dec.scaleExp != 2 {
			t.Errorf("expected scaleExp=2, got %d", dec.scaleExp)
		}
	}
	verifyStorage(t, col, n, gen)
}

func TestStorageDecimalSerializeRoundtrip(t *testing.T) {
	n := 100
	gen := func(i int) scm.Scmer { return scm.NewFloat(float64(i) * 0.01) }
	col := buildViaCompression(n, gen)
	col2 := serializeDeserialize(col)
	if col2 == nil {
		t.Fatal("deserialization returned nil")
	}
	for i := 0; i < n; i++ {
		got := col2.GetValue(uint(i))
		want := float64(i) * 0.01
		diff := math.Abs(got.Float() - want)
		if diff > 1e-9 {
			t.Fatalf("after roundtrip: GetValue(%d) = %v, want %v", i, got.Float(), want)
		}
	}
}

// --- StorageSeq tests ---

func TestStorageSeqPipeline(t *testing.T) {
	// 0, 2, 4, 6, ... → single sequence with stride 2
	n := 100
	gen := func(i int) scm.Scmer { return scm.NewInt(int64(i) * 2) }
	col := buildViaCompression(n, gen)
	if _, ok := col.(*StorageSeq); !ok {
		t.Fatalf("expected *StorageSeq, got %T", col)
	}
	verifyStorage(t, col, n, gen)
}

func TestStorageSeqMultipleSequences(t *testing.T) {
	// Two sequences: 0-49 with stride 1, then 1000-1049 with stride 1
	n := 100
	gen := func(i int) scm.Scmer {
		if i < 50 {
			return scm.NewInt(int64(i))
		}
		return scm.NewInt(int64(i) + 950)
	}
	col := buildViaCompression(n, gen)
	if _, ok := col.(*StorageSeq); !ok {
		t.Fatalf("expected *StorageSeq, got %T", col)
	}
	verifyStorage(t, col, n, gen)
}

func TestStorageSeqSerializeRoundtrip(t *testing.T) {
	n := 100
	gen := func(i int) scm.Scmer { return scm.NewInt(int64(i) * 3) }
	col := buildViaCompression(n, gen)
	col2 := serializeDeserialize(col)
	if col2 == nil {
		t.Fatal("deserialization returned nil")
	}
	verifyStorage(t, col2, n, gen)
}

// --- StorageSparse tests ---

func TestStorageSparsePipeline(t *testing.T) {
	// >13% NULL → triggers sparse
	n := 100
	gen := func(i int) scm.Scmer {
		if i%5 != 0 {
			return scm.NewNil()
		}
		return scm.NewInt(int64(i))
	}
	col := buildViaCompression(n, gen)
	if _, ok := col.(*StorageSparse); !ok {
		t.Fatalf("expected *StorageSparse, got %T", col)
	}
	verifyStorage(t, col, n, gen)
}

func TestStorageSparseEdgeCases(t *testing.T) {
	// First and last elements non-null
	n := 100
	gen := func(i int) scm.Scmer {
		if i == 0 || i == n-1 {
			return scm.NewInt(int64(i))
		}
		return scm.NewNil()
	}
	col := buildViaCompression(n, gen)
	verifyStorage(t, col, n, gen)
}

func TestStorageSparseSerializeRoundtrip(t *testing.T) {
	n := 100
	gen := func(i int) scm.Scmer {
		if i%7 != 0 {
			return scm.NewNil()
		}
		return scm.NewInt(int64(i) * 10)
	}
	col := buildViaCompression(n, gen)
	col2 := serializeDeserialize(col)
	if col2 == nil {
		t.Fatal("deserialization returned nil")
	}
	verifyStorage(t, col2, n, gen)
}

// --- StorageEnum serialization round-trip ---

func TestStorageEnumSerializeRoundtrip(t *testing.T) {
	n := 1000
	gen := func(i int) scm.Scmer {
		if i%100 == 99 {
			return scm.NewInt(1)
		}
		return scm.NewInt(0)
	}
	col := buildViaCompression(n, gen)
	if _, ok := col.(*StorageEnum); !ok {
		t.Fatalf("expected *StorageEnum, got %T", col)
	}
	col2 := serializeDeserialize(col)
	if col2 == nil {
		t.Fatal("deserialization returned nil")
	}
	verifyStorage(t, col2, n, gen)
}

func TestStorageEnumStringSymbols(t *testing.T) {
	// 3 string symbols, skewed → should use StorageEnum
	n := 1000
	gen := func(i int) scm.Scmer {
		v := i % 100
		if v < 90 {
			return scm.NewString("common")
		}
		if v < 98 {
			return scm.NewString("rare")
		}
		return scm.NewString("ultra_rare")
	}
	col := buildViaCompression(n, gen)
	if _, ok := col.(*StorageEnum); !ok {
		t.Fatalf("expected *StorageEnum, got %T", col)
	}
	verifyStorage(t, col, n, gen)
	// also test serialization
	col2 := serializeDeserialize(col)
	verifyStorage(t, col2, n, gen)
}

// --- StorageSCMER proposeCompression decision tests ---

func TestProposeCompressionFloatOrDecimal(t *testing.T) {
	// Floats should produce StorageFloat or StorageDecimal (depending on precision detection)
	s := new(StorageSCMER)
	s.prepare()
	for i := 0; i < 100; i++ {
		s.scan(uint(i), scm.NewFloat(float64(i)*math.Pi))
	}
	result := s.proposeCompression(100)
	switch result.(type) {
	case *StorageFloat, *StorageDecimal:
		// both acceptable
	default:
		t.Errorf("floats: expected *StorageFloat or *StorageDecimal, got %T", result)
	}
}

func TestProposeCompressionIntOrSeq(t *testing.T) {
	// Linear integers may produce StorageInt or StorageSeq
	s := new(StorageSCMER)
	s.prepare()
	for i := 0; i < 100; i++ {
		s.scan(uint(i), scm.NewInt(int64(i*7+3)))
	}
	result := s.proposeCompression(100)
	switch result.(type) {
	case *StorageInt, *StorageSeq:
		// both acceptable
	default:
		t.Errorf("integers: expected *StorageInt or *StorageSeq, got %T", result)
	}
}

func TestProposeCompressionSeq(t *testing.T) {
	s := new(StorageSCMER)
	s.prepare()
	for i := 0; i < 100; i++ {
		s.scan(uint(i), scm.NewInt(int64(i*5)))
	}
	result := s.proposeCompression(100)
	if _, ok := result.(*StorageSeq); !ok {
		t.Errorf("sequential integers: expected *StorageSeq, got %T", result)
	}
}

func TestProposeCompressionSparse(t *testing.T) {
	s := new(StorageSCMER)
	s.prepare()
	for i := 0; i < 100; i++ {
		if i%4 == 0 {
			s.scan(uint(i), scm.NewInt(int64(i)))
		} else {
			s.scan(uint(i), scm.NewNil())
		}
	}
	result := s.proposeCompression(100)
	if _, ok := result.(*StorageSparse); !ok {
		t.Errorf("75%% NULL: expected *StorageSparse, got %T", result)
	}
}

func TestProposeCompressionDecimal(t *testing.T) {
	s := new(StorageSCMER)
	s.prepare()
	for i := 0; i < 100; i++ {
		s.scan(uint(i), scm.NewFloat(float64(i)*0.01))
	}
	result := s.proposeCompression(100)
	if _, ok := result.(*StorageDecimal); !ok {
		t.Errorf("two-decimal floats: expected *StorageDecimal, got %T", result)
	}
}

func TestProposeCompressionString(t *testing.T) {
	s := new(StorageSCMER)
	s.prepare()
	for i := 0; i < 100; i++ {
		s.scan(uint(i), scm.NewString(fmt.Sprintf("item_%d", i)))
	}
	result := s.proposeCompression(100)
	if _, ok := result.(*StorageString); !ok {
		t.Errorf("strings: expected *StorageString, got %T", result)
	}
}

// --- GetCachedReader tests for all types ---

func TestGetCachedReaderReturnsSelf(t *testing.T) {
	types := []struct {
		name string
		col  ColumnStorage
	}{
		{"StorageSCMER", &StorageSCMER{values: []scm.Scmer{scm.NewInt(1)}}},
		{"StorageInt", &StorageInt{}},
		{"StorageFloat", &StorageFloat{}},
		{"StorageString", &StorageString{}},
		{"StorageSeq", &StorageSeq{}},
		// StoragePrefix omitted: Serialize/Deserialize not implemented
		{"StorageSparse", &StorageSparse{}},
		{"StorageDecimal", &StorageDecimal{}},
		{"OverlayBlob", &OverlayBlob{}},
	}
	for _, tc := range types {
		reader := tc.col.GetCachedReader()
		if reader != ColumnReader(tc.col) {
			t.Errorf("%s.GetCachedReader() should return self", tc.name)
		}
	}
}

// --- ComputeSize tests ---

func TestComputeSizeNonZero(t *testing.T) {
	n := 100
	gen := func(i int) scm.Scmer { return scm.NewInt(int64(i)) }
	col := buildViaCompression(n, gen)
	sz := col.ComputeSize()
	if sz == 0 {
		t.Errorf("ComputeSize() returned 0 for %T", col)
	}
}

func TestComputeSizeAllTypes(t *testing.T) {
	tests := []struct {
		name string
		gen  func(int) scm.Scmer
	}{
		{"Int", func(i int) scm.Scmer { return scm.NewInt(int64(i * 7)) }},
		{"Float", func(i int) scm.Scmer { return scm.NewFloat(float64(i) * math.Pi) }},
		{"String", func(i int) scm.Scmer { return scm.NewString(fmt.Sprintf("s%d", i%10)) }},
		{"Decimal", func(i int) scm.Scmer { return scm.NewFloat(float64(i) * 0.01) }},
	}
	for _, tc := range tests {
		col := buildViaCompression(100, tc.gen)
		sz := col.ComputeSize()
		if sz == 0 {
			t.Errorf("%s: ComputeSize() returned 0 for %T", tc.name, col)
		}
	}
}

// --- String() description tests ---

func TestStringDescriptions(t *testing.T) {
	tests := []struct {
		name string
		gen  func(int) scm.Scmer
	}{
		{"Int", func(i int) scm.Scmer { return scm.NewInt(int64(i)) }},
		{"Float", func(i int) scm.Scmer { return scm.NewFloat(float64(i) * math.Pi) }},
		{"String", func(i int) scm.Scmer { return scm.NewString(fmt.Sprintf("s%d", i%10)) }},
	}
	for _, tc := range tests {
		col := buildViaCompression(100, tc.gen)
		s := col.String()
		if s == "" {
			t.Errorf("%s: String() returned empty for %T", tc.name, col)
		}
	}
}
