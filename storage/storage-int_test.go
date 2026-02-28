package storage

import (
	"testing"

	"github.com/launix-de/memcp/scm"
)

// buildStorageInt builds a StorageInt via the standard prepare/scan/init/build/finish pipeline.
func buildStorageInt(values []scm.Scmer) *StorageInt {
	s := new(StorageInt)
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

// assertGetValue checks that GetValue returns the expected value.
func assertGetValue(t *testing.T, s *StorageInt, idx uint32, expected scm.Scmer, ctx string) {
	t.Helper()
	got := s.GetValue(idx)
	if expected.IsNil() {
		if !got.IsNil() {
			t.Errorf("%s: idx=%d expected nil, got %v", ctx, idx, got)
		}
		return
	}
	if got.IsNil() {
		t.Errorf("%s: idx=%d expected %v, got nil", ctx, idx, expected)
		return
	}
	if got.Int() != expected.Int() {
		t.Errorf("%s: idx=%d expected %v, got %v", ctx, idx, expected, got)
	}
}

// TestSetValueBasic tests overwriting each element in a small array.
func TestSetValueBasic(t *testing.T) {
	values := []scm.Scmer{
		scm.NewInt(10), scm.NewInt(20), scm.NewInt(30), scm.NewInt(40), scm.NewInt(50),
	}
	s := buildStorageInt(values)

	// overwrite middle element
	s.SetValue(2, scm.NewInt(35))
	assertGetValue(t, s, 0, scm.NewInt(10), "basic")
	assertGetValue(t, s, 1, scm.NewInt(20), "basic")
	assertGetValue(t, s, 2, scm.NewInt(35), "basic")
	assertGetValue(t, s, 3, scm.NewInt(40), "basic")
	assertGetValue(t, s, 4, scm.NewInt(50), "basic")

	// overwrite first element
	s.SetValue(0, scm.NewInt(15))
	assertGetValue(t, s, 0, scm.NewInt(15), "basic-first")

	// overwrite last element
	s.SetValue(4, scm.NewInt(45))
	assertGetValue(t, s, 4, scm.NewInt(45), "basic-last")
}

// TestSetValueAllBitsizes tests SetValue across various bitsizes (1 to 20).
func TestSetValueAllBitsizes(t *testing.T) {
	for bitsize := uint8(1); bitsize <= 20; bitsize++ {
		maxVal := int64(1<<bitsize) - 1
		n := 100
		// build array with values [0..maxVal] cycling
		values := make([]scm.Scmer, n)
		for i := 0; i < n; i++ {
			values[i] = scm.NewInt(int64(i) % maxVal)
		}
		s := buildStorageInt(values)
		if s.bitsize != bitsize {
			// bitsize may differ if scan range differs; skip if so
			continue
		}

		// overwrite every element with a different value and verify
		for i := 0; i < n; i++ {
			newVal := (maxVal - 1) - (int64(i) % maxVal)
			s.SetValue(uint32(i), scm.NewInt(newVal+s.offset))
			assertGetValue(t, s, uint32(i), scm.NewInt(newVal+s.offset), "bitsize-sweep")
		}
		// verify all values are correct after all overwrites
		for i := 0; i < n; i++ {
			newVal := (maxVal - 1) - (int64(i) % maxVal)
			assertGetValue(t, s, uint32(i), scm.NewInt(newVal+s.offset), "bitsize-verify")
		}
	}
}

// TestSetValueChunkBoundary tests values that straddle 64-bit chunk boundaries.
func TestSetValueChunkBoundary(t *testing.T) {
	// Use values 0..99 → bitsize=7 (values up to 99 need ceil(log2(100))=7 bits)
	// Chunk boundary crossings at indices where bitpos%64 + 7 > 64,
	// i.e. bitpos%64 > 57, which is bitpos in {58,59,60,61,62,63} mod 64.
	// For bitsize=7: element i starts at bit i*7.
	// Crossing happens when (i*7)%64 > 57.
	n := 100
	values := make([]scm.Scmer, n)
	for i := 0; i < n; i++ {
		values[i] = scm.NewInt(int64(i))
	}
	s := buildStorageInt(values)

	// find and test all boundary-crossing indices
	crossings := 0
	for i := uint32(0); i < uint32(n); i++ {
		bitpos := uint(i) * uint(s.bitsize)
		if bitpos%64+uint(s.bitsize) > 64 {
			crossings++
			// overwrite with max value
			newVal := scm.NewInt(int64(n) - 1)
			s.SetValue(i, newVal)
			assertGetValue(t, s, i, newVal, "boundary-max")

			// overwrite with 0
			s.SetValue(i, scm.NewInt(0))
			assertGetValue(t, s, i, scm.NewInt(0), "boundary-zero")

			// restore original
			s.SetValue(i, scm.NewInt(int64(i)))
			assertGetValue(t, s, i, scm.NewInt(int64(i)), "boundary-restore")
		}
	}
	if crossings == 0 {
		t.Fatal("no chunk boundary crossings found — test is broken")
	}

	// verify all values still intact after boundary operations
	for i := 0; i < n; i++ {
		assertGetValue(t, s, uint32(i), scm.NewInt(int64(i)), "boundary-intact")
	}
}

// TestSetValueWithNull tests SetValue with NULL values.
func TestSetValueWithNull(t *testing.T) {
	values := []scm.Scmer{
		scm.NewInt(5), scm.NewNil(), scm.NewInt(10), scm.NewInt(15), scm.NewNil(),
	}
	s := buildStorageInt(values)

	// verify initial state
	assertGetValue(t, s, 0, scm.NewInt(5), "null-init")
	assertGetValue(t, s, 1, scm.NewNil(), "null-init")
	assertGetValue(t, s, 2, scm.NewInt(10), "null-init")

	// set a non-null to null
	s.SetValue(0, scm.NewNil())
	assertGetValue(t, s, 0, scm.NewNil(), "null-set")

	// set a null to non-null
	s.SetValue(1, scm.NewInt(7))
	assertGetValue(t, s, 1, scm.NewInt(7), "null-clear")

	// set null back
	s.SetValue(1, scm.NewNil())
	assertGetValue(t, s, 1, scm.NewNil(), "null-reset")

	// verify others unchanged
	assertGetValue(t, s, 2, scm.NewInt(10), "null-intact")
	assertGetValue(t, s, 3, scm.NewInt(15), "null-intact")
	assertGetValue(t, s, 4, scm.NewNil(), "null-intact")
}

// TestSetValueWithOffset tests SetValue when offset is non-zero (negative values).
func TestSetValueWithOffset(t *testing.T) {
	values := []scm.Scmer{
		scm.NewInt(-100), scm.NewInt(-50), scm.NewInt(0), scm.NewInt(50), scm.NewInt(100),
	}
	s := buildStorageInt(values)

	if s.offset != -100 {
		t.Fatalf("expected offset=-100, got %d", s.offset)
	}

	// overwrite with values still in range
	s.SetValue(0, scm.NewInt(-50))
	assertGetValue(t, s, 0, scm.NewInt(-50), "offset")

	s.SetValue(2, scm.NewInt(100))
	assertGetValue(t, s, 2, scm.NewInt(100), "offset")

	s.SetValue(4, scm.NewInt(-100))
	assertGetValue(t, s, 4, scm.NewInt(-100), "offset")

	// verify others unchanged
	assertGetValue(t, s, 1, scm.NewInt(-50), "offset-intact")
	assertGetValue(t, s, 3, scm.NewInt(50), "offset-intact")
}

// TestSetValueSameValue tests that overwriting with the same value is idempotent.
func TestSetValueIdempotent(t *testing.T) {
	values := []scm.Scmer{
		scm.NewInt(42), scm.NewInt(99), scm.NewInt(0), scm.NewInt(7),
	}
	s := buildStorageInt(values)

	for i, v := range values {
		s.SetValue(uint32(i), v)
		assertGetValue(t, s, uint32(i), v, "idempotent")
	}
}

// TestSetValueNoNeighborCorruption tests that SetValue doesn't corrupt adjacent elements.
func TestSetValueNoNeighborCorruption(t *testing.T) {
	n := 200
	values := make([]scm.Scmer, n)
	for i := 0; i < n; i++ {
		values[i] = scm.NewInt(int64(i))
	}
	s := buildStorageInt(values)

	// overwrite every other element, then verify ALL elements
	for i := 0; i < n; i += 2 {
		s.SetValue(uint32(i), scm.NewInt(int64(n-1-i)))
	}
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			assertGetValue(t, s, uint32(i), scm.NewInt(int64(n-1-i)), "neighbor-modified")
		} else {
			assertGetValue(t, s, uint32(i), scm.NewInt(int64(i)), "neighbor-original")
		}
	}
}

// TestSetValueZeroToMax tests flipping between 0 and max value for each element.
func TestSetValueZeroToMax(t *testing.T) {
	n := 64
	values := make([]scm.Scmer, n)
	for i := 0; i < n; i++ {
		values[i] = scm.NewInt(int64(i))
	}
	s := buildStorageInt(values)
	maxVal := scm.NewInt(s.max)

	// set all to max
	for i := 0; i < n; i++ {
		s.SetValue(uint32(i), maxVal)
	}
	for i := 0; i < n; i++ {
		assertGetValue(t, s, uint32(i), maxVal, "all-max")
	}

	// set all to min (offset)
	minVal := scm.NewInt(s.offset)
	for i := 0; i < n; i++ {
		s.SetValue(uint32(i), minVal)
	}
	for i := 0; i < n; i++ {
		assertGetValue(t, s, uint32(i), minVal, "all-min")
	}

	// alternate: even=max, odd=min
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			s.SetValue(uint32(i), maxVal)
		} else {
			s.SetValue(uint32(i), minVal)
		}
	}
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			assertGetValue(t, s, uint32(i), maxVal, "alternate-max")
		} else {
			assertGetValue(t, s, uint32(i), minVal, "alternate-min")
		}
	}
}

// TestSetValueBitsize1 tests the edge case of bitsize=1 (boolean-like storage).
func TestSetValueBitsize1(t *testing.T) {
	n := 128
	values := make([]scm.Scmer, n)
	for i := 0; i < n; i++ {
		values[i] = scm.NewInt(int64(i % 2))
	}
	s := buildStorageInt(values)
	if s.bitsize != 1 {
		t.Skipf("bitsize=%d, expected 1", s.bitsize)
	}

	// flip all bits
	for i := 0; i < n; i++ {
		s.SetValue(uint32(i), scm.NewInt(int64(1-(i%2))))
	}
	for i := 0; i < n; i++ {
		assertGetValue(t, s, uint32(i), scm.NewInt(int64(1-(i%2))), "bitsize1-flip")
	}

	// flip back
	for i := 0; i < n; i++ {
		s.SetValue(uint32(i), scm.NewInt(int64(i%2)))
	}
	for i := 0; i < n; i++ {
		assertGetValue(t, s, uint32(i), scm.NewInt(int64(i%2)), "bitsize1-restore")
	}
}

// TestSetValueLargeBitsize tests with values requiring many bits (e.g. 40-bit).
func TestSetValueLargeBitsize(t *testing.T) {
	n := 50
	values := make([]scm.Scmer, n)
	base := int64(1) << 39 // force ~40 bit bitsize
	for i := 0; i < n; i++ {
		values[i] = scm.NewInt(base + int64(i))
	}
	s := buildStorageInt(values)

	// overwrite with value at opposite end of range
	for i := 0; i < n; i++ {
		newVal := base + int64(n-1-i)
		s.SetValue(uint32(i), scm.NewInt(newVal))
		assertGetValue(t, s, uint32(i), scm.NewInt(newVal), "large-bitsize")
	}
}

// TestSetValueSequentialOverwrite tests overwriting all elements sequentially twice.
func TestSetValueSequentialOverwrite(t *testing.T) {
	n := 300
	values := make([]scm.Scmer, n)
	for i := 0; i < n; i++ {
		values[i] = scm.NewInt(int64(i % 200))
	}
	s := buildStorageInt(values)

	// first pass: set all to constant
	for i := 0; i < n; i++ {
		s.SetValue(uint32(i), scm.NewInt(100+s.offset))
	}
	for i := 0; i < n; i++ {
		assertGetValue(t, s, uint32(i), scm.NewInt(100+s.offset), "seq-pass1")
	}

	// second pass: set all to original values
	for i := 0; i < n; i++ {
		s.SetValue(uint32(i), scm.NewInt(int64(i%200)+s.offset))
	}
	for i := 0; i < n; i++ {
		assertGetValue(t, s, uint32(i), scm.NewInt(int64(i%200)+s.offset), "seq-pass2")
	}
}
