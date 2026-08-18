/*
Copyright (C) 2026  Carl-Philip Hänsch

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
	"math/rand"
	"testing"

	"github.com/launix-de/memcp/scm"
)

// stringBulkFixtures builds one StorageString per StringFormat by cycling a
// small set of format-valid sample values (with occasional NULLs), so the
// new arena-based GetValueRange/GetValueMulti decode path (writeNibblesInto,
// writeUUIDInto, base64.Encode) is exercised for every format, not just the
// FormatRaw case the generic TestBulkReadMatchesGetValue fixtures happen to
// produce.
func stringBulkFixtures(n int) []struct {
	name    string
	want    StringFormat
	samples []string
} {
	return []struct {
		name    string
		want    StringFormat
		samples []string
	}{
		{"HexLower", FormatHexLower, []string{
			"d41d8cd98f00b204e9800998ecf8427e", "098f6bcd4621d373cade4e832627b4f6", "0123456789abcdef",
		}},
		{"HexUpper", FormatHexUpper, []string{
			"D41D8CD98F00B204E9800998ECF8427E", "098F6BCD4621D373CADE4E832627B4F6", "0123456789ABCDEF",
		}},
		{"Phone", FormatPhone, []string{
			"+49 30 123456", "0800/123 456", "(030) 123-456",
		}},
		{"PhoneDTMF", FormatPhoneDTMF, []string{
			"*100#", "+49123*456#", "(1)2*3#",
		}},
		{"Decimal", FormatDecimal, []string{
			"3.14", "-1,23e+10", "42.0", "0.0001",
		}},
		{"DateTime", FormatDateTime, []string{
			"2024-03-07 15:30:00", "2023-12-31T23:59:59", "2020-01-01 00:00:00",
		}},
		{"UUIDLower", FormatUUIDLower, []string{
			"550e8400-e29b-41d4-a716-446655440000", "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		}},
		{"UUIDUpper", FormatUUIDUpper, []string{
			"550E8400-E29B-41D4-A716-446655440000", "6BA7B810-9DAD-11D1-80B4-00C04FD430C8",
		}},
		{"Base64Upper", FormatBase64Upper, []string{
			"dGVzdA==", "++//", "aGVsbG8gd29ybGQ=",
		}},
		{"Base64Lower", FormatBase64Lower, []string{
			"dGVzdA==", "--__", "aGVsbG8gd29ybGQ=",
		}},
		{"Raw", FormatRaw, []string{
			"Hello, World!", "foo bar baz", "not a hex string!",
		}},
	}
}

func verifyStringBulkAgainstGetValue(t *testing.T, name string, s *StorageString, n int) {
	t.Helper()

	want := make([]string, n)
	wantNil := make([]bool, n)
	for i := 0; i < n; i++ {
		v := s.GetValue(uint32(i))
		wantNil[i] = v.IsNil()
		if !wantNil[i] {
			want[i] = v.String()
		}
	}

	check := func(label string, got []scm.Scmer, recids []int) {
		for k, recid := range recids {
			if got[k].IsNil() != wantNil[recid] {
				t.Errorf("%s: %s[%d] (recid %d) nil=%v, want nil=%v", name, label, k, recid, got[k].IsNil(), wantNil[recid])
				continue
			}
			if !wantNil[recid] && got[k].String() != want[recid] {
				t.Errorf("%s: %s[%d] (recid %d) = %q, want %q", name, label, k, recid, got[k].String(), want[recid])
			}
			if !wantNil[recid] && !got[k].IsString() {
				t.Errorf("%s: %s[%d] (recid %d): bulk value is not eagerly a plain string (tag=%d)", name, label, k, recid, got[k].GetTag())
			}
		}
	}

	// full-range
	full := make([]scm.Scmer, n)
	s.GetValueRange(0, uint32(n), full, 1)
	fullRecids := make([]int, n)
	for i := range fullRecids {
		fullRecids[i] = i
	}
	check("GetValueRange(full)", full, fullRecids)

	// windowed range
	base := n / 4
	count := n / 2
	win := make([]scm.Scmer, count)
	s.GetValueRange(uint32(base), uint32(count), win, 1)
	winRecids := make([]int, count)
	for i := range winRecids {
		winRecids[i] = base + i
	}
	check("GetValueRange(window)", win, winRecids)

	// ascending gappy multi
	var ascending []uint32
	var ascendingInt []int
	for i := 0; i < n; i += 3 {
		ascending = append(ascending, uint32(i))
		ascendingInt = append(ascendingInt, i)
	}
	gotAsc := make([]scm.Scmer, len(ascending))
	s.GetValueMulti(ascending, gotAsc, 1)
	check("GetValueMulti(ascending)", gotAsc, ascendingInt)

	// shuffled multi with repeats
	rng := rand.New(rand.NewSource(3))
	shuffled := make([]uint32, n)
	shuffledInt := make([]int, n)
	for i := range shuffled {
		shuffled[i] = uint32(i)
		shuffledInt[i] = i
	}
	rng.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		shuffledInt[i], shuffledInt[j] = shuffledInt[j], shuffledInt[i]
	})
	shuffled = append(shuffled, shuffled[:n/5]...)
	shuffledInt = append(shuffledInt, shuffledInt[:n/5]...)
	gotShuf := make([]scm.Scmer, len(shuffled))
	s.GetValueMulti(shuffled, gotShuf, 1)
	check("GetValueMulti(shuffled)", gotShuf, shuffledInt)

	// stride>1
	const stride = 3
	target := make([]scm.Scmer, len(ascending)*stride)
	for i := range target {
		target[i] = scm.NewString("sentinel")
	}
	s.GetValueMulti(ascending, target, stride)
	strided := make([]scm.Scmer, len(ascending))
	for i := range ascending {
		strided[i] = target[i*stride]
	}
	check("GetValueMulti(stride=3)", strided, ascendingInt)
	for i := range ascending {
		for g := 1; g < stride; g++ {
			if target[i*stride+g].String() != "sentinel" {
				t.Errorf("%s: GetValueMulti stride=%d clobbered gap slot %d", name, stride, i*stride+g)
			}
		}
	}
}

// TestStringBulkMatchesGetValueNodict forces nodict mode (every value
// unique, so a dictionary wouldn't pay for itself) to exercise the
// resolvePositions branch that indexes starts/lens directly by recid instead
// of through an indirect dictionary-entry lookup.
func TestStringBulkMatchesGetValueNodict(t *testing.T) {
	const n = 300
	values := make([]scm.Scmer, n)
	for i := 0; i < n; i++ {
		// nodict only kicks in once the scanner sees >99 distinct values
		// among the first 100 rows, so keep those unique and sprinkle NULLs
		// only afterwards.
		if i >= 150 && i%11 == 0 {
			values[i] = scm.NewNil()
			continue
		}
		values[i] = scm.NewString("unique_raw_value_number_" + scm.String(scm.NewInt(int64(i))))
	}
	s := buildStorageString(values)
	if !s.nodict {
		t.Fatalf("fixture no longer forces nodict mode (all-unique values should defeat dictionary compression)")
	}
	verifyStringBulkAgainstGetValue(t, "nodict", s, n)
}

// buildStoragePrefix manually drives StoragePrefix's scan/build lifecycle.
// proposeCompression never selects StoragePrefix today (its Serialize/
// Deserialize is disabled, see the TODO in StorageString.proposeCompression),
// so it can currently only arise from reading legacy persisted data — this
// constructs one directly so the arena-based bulk path stays covered.
func buildStoragePrefix(prefixDict []string, values []scm.Scmer) *StoragePrefix {
	s := new(StoragePrefix)
	s.prefixdictionary = prefixDict
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

// TestPrefixBulkMatchesGetValue exercises the arena-based
// GetValueRange/GetValueMulti rewrite of StoragePrefix.applyPrefixInPlace
// against a plain per-element GetValue loop.
func TestPrefixBulkMatchesGetValue(t *testing.T) {
	const n = 300
	prefixDict := []string{"", "https://example.com/path/", "urn:isbn:"}
	values := make([]scm.Scmer, n)
	for i := 0; i < n; i++ {
		if i%13 == 0 {
			values[i] = scm.NewNil()
			continue
		}
		switch i % 3 {
		case 0:
			values[i] = scm.NewString("https://example.com/path/" + scm.String(scm.NewInt(int64(i))))
		case 1:
			values[i] = scm.NewString("urn:isbn:" + scm.String(scm.NewInt(int64(i))))
		default:
			values[i] = scm.NewString("no-prefix-match-" + scm.String(scm.NewInt(int64(i))))
		}
	}
	s := buildStoragePrefix(prefixDict, values)

	want := make([]scm.Scmer, n)
	for i := 0; i < n; i++ {
		want[i] = s.GetValue(uint32(i))
	}
	check := func(label string, got []scm.Scmer, recids []int) {
		for k, recid := range recids {
			if !scm.Equal(got[k], want[recid]) {
				t.Errorf("prefix: %s[%d] (recid %d) = %v, want %v", label, k, recid, got[k], want[recid])
			}
		}
	}

	full := make([]scm.Scmer, n)
	s.GetValueRange(0, uint32(n), full, 1)
	fullRecids := make([]int, n)
	for i := range fullRecids {
		fullRecids[i] = i
	}
	check("GetValueRange(full)", full, fullRecids)

	var ascending []uint32
	var ascendingInt []int
	for i := 0; i < n; i += 4 {
		ascending = append(ascending, uint32(i))
		ascendingInt = append(ascendingInt, i)
	}
	gotAsc := make([]scm.Scmer, len(ascending))
	s.GetValueMulti(ascending, gotAsc, 1)
	check("GetValueMulti(ascending)", gotAsc, ascendingInt)

	rng := rand.New(rand.NewSource(5))
	shuffled := make([]uint32, n)
	shuffledInt := make([]int, n)
	for i := range shuffled {
		shuffled[i] = uint32(i)
		shuffledInt[i] = i
	}
	rng.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		shuffledInt[i], shuffledInt[j] = shuffledInt[j], shuffledInt[i]
	})
	gotShuf := make([]scm.Scmer, len(shuffled))
	s.GetValueMulti(shuffled, gotShuf, 1)
	check("GetValueMulti(shuffled)", gotShuf, shuffledInt)

	const stride = 4
	target := make([]scm.Scmer, len(ascending)*stride)
	for i := range target {
		target[i] = scm.NewString("sentinel")
	}
	s.GetValueMulti(ascending, target, stride)
	strided := make([]scm.Scmer, len(ascending))
	for i := range ascending {
		strided[i] = target[i*stride]
	}
	check("GetValueMulti(stride=4)", strided, ascendingInt)
	for i := range ascending {
		for g := 1; g < stride; g++ {
			if target[i*stride+g].String() != "sentinel" {
				t.Errorf("prefix: GetValueMulti stride=%d clobbered gap slot %d", stride, i*stride+g)
			}
		}
	}
}

func TestStringBulkMatchesGetValue(t *testing.T) {
	const n = 300
	for _, fx := range stringBulkFixtures(n) {
		fx := fx
		t.Run(fx.name, func(t *testing.T) {
			values := make([]scm.Scmer, n)
			for i := 0; i < n; i++ {
				if i%11 == 0 {
					values[i] = scm.NewNil()
					continue
				}
				values[i] = scm.NewString(fx.samples[i%len(fx.samples)])
			}
			s := buildStorageString(values)
			if s.format != fx.want {
				t.Fatalf("%s: column compressed to format %d, want %d (test fixture no longer triggers the intended format)", fx.name, s.format, fx.want)
			}
			verifyStringBulkAgainstGetValue(t, fx.name, s, n)
		})
	}
}

// --- Allocation-count benchmarks: per-row GetValue+.String() (the old bulk
// path's behavior, and what a caller consuming every lazy CString/BString
// value from GetValue would pay) vs. the new arena-based GetValueMulti. ---

func BenchmarkStringPerRowVsBulk(b *testing.B) {
	const n = 2000
	for _, fx := range stringBulkFixtures(n) {
		values := make([]scm.Scmer, n)
		for i := 0; i < n; i++ {
			values[i] = scm.NewString(fx.samples[i%len(fx.samples)])
		}
		s := buildStorageString(values)
		recids := make([]uint32, n)
		for i := range recids {
			recids[i] = uint32(i)
		}
		target := make([]scm.Scmer, n)

		b.Run(fx.name+"/PerRowGetValue+String", func(b *testing.B) {
			b.ReportAllocs()
			for iter := 0; iter < b.N; iter++ {
				for i := 0; i < n; i++ {
					_ = s.GetValue(uint32(i)).String()
				}
			}
		})
		b.Run(fx.name+"/BulkGetValueMulti", func(b *testing.B) {
			b.ReportAllocs()
			for iter := 0; iter < b.N; iter++ {
				s.GetValueMulti(recids, target, 1)
			}
		})
	}
}

func BenchmarkPrefixPerRowVsBulk(b *testing.B) {
	const n = 2000
	prefixDict := []string{"", "https://example.com/path/"}
	values := make([]scm.Scmer, n)
	for i := 0; i < n; i++ {
		values[i] = scm.NewString("https://example.com/path/" + scm.String(scm.NewInt(int64(i))))
	}
	s := buildStoragePrefix(prefixDict, values)
	recids := make([]uint32, n)
	for i := range recids {
		recids[i] = uint32(i)
	}
	target := make([]scm.Scmer, n)

	b.Run("PerRowGetValue", func(b *testing.B) {
		b.ReportAllocs()
		for iter := 0; iter < b.N; iter++ {
			for i := 0; i < n; i++ {
				_ = s.GetValue(uint32(i))
			}
		}
	})
	b.Run("BulkGetValueMulti", func(b *testing.B) {
		b.ReportAllocs()
		for iter := 0; iter < b.N; iter++ {
			s.GetValueMulti(recids, target, 1)
		}
	})
}
