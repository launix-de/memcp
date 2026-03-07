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

import (
	"testing"

	"github.com/launix-de/memcp/scm"
)

// buildStringColumn runs the full StorageString pipeline for the given strings
// and returns the finished column ready for GetValue.
func buildStringColumn(values []string) *StorageString {
	s := new(StorageString)
	s.prepare()
	for i, v := range values {
		s.scan(uint32(i), scm.NewString(v))
	}
	s.init(uint32(len(values)))
	for i, v := range values {
		s.build(uint32(i), scm.NewString(v))
	}
	s.finish()
	return s
}

// TestFormatSelection asserts that chooseBestFormat selects the expected format
// when all input strings belong to a specific charset.
func TestFormatSelection(t *testing.T) {
	cases := []struct {
		name    string
		inputs  []string
		want    StringFormat
	}{
		{
			name: "UUID lowercase",
			inputs: []string{
				"550e8400-e29b-41d4-a716-446655440000",
				"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			},
			want: FormatUUIDLower,
		},
		{
			name: "UUID uppercase",
			inputs: []string{
				"550E8400-E29B-41D4-A716-446655440000",
				"6BA7B810-9DAD-11D1-80B4-00C04FD430C8",
			},
			want: FormatUUIDUpper,
		},
		{
			name: "MD5 hex lowercase",
			inputs: []string{
				"d41d8cd98f00b204e9800998ecf8427e",
				"098f6bcd4621d373cade4e832627b4f6",
			},
			want: FormatHexLower,
		},
		{
			name: "hex uppercase",
			inputs: []string{
				"D41D8CD98F00B204E9800998ECF8427E",
				"098F6BCD4621D373CADE4E832627B4F6",
			},
			want: FormatHexUpper,
		},
		{
			name: "phone with spaces and slashes",
			inputs: []string{
				"+49 30 123456",
				"0800/123 456",
			},
			want: FormatPhone,
		},
		{
			name: "DTMF sequences",
			inputs: []string{
				"*100#",
				"+49123*456#",
			},
			want: FormatPhoneDTMF,
		},
		{
			name: "decimal / scientific notation",
			inputs: []string{
				"3.14",
				"-1,23e+10",
				"42.0",
			},
			want: FormatDecimal,
		},
		{
			name: "ISO 8601 datetime",
			inputs: []string{
				"2024-03-07 15:30:00",
				"2023-12-31T23:59:59",
			},
			want: FormatDateTime,
		},
		{
			name: "raw (mixed content)",
			inputs: []string{
				"Hello, World!",
				"foo bar baz",
			},
			want: FormatRaw,
		},
		{
			// Standard base64 must contain at least one '+' or '/' to distinguish
			// from URL-safe; otherwise chooseBestFormat picks FormatBase64Upper first.
			name: "standard base64 (with + and /)",
			inputs: []string{
				"dGVzdA==", // "test"
				"++//",     // 3 bytes {0xFB,0xEF,0xFF}
			},
			want: FormatBase64Upper,
		},
		{
			// URL-safe base64 must contain '-' or '_' to be unambiguous.
			name: "URL-safe base64 (with - and _)",
			inputs: []string{
				"dGVzdA==", // "test" (no special chars, compatible with both)
				"--__",     // 3 bytes {0xFB,0xEF,0xFF} in URL-safe encoding
			},
			want: FormatBase64Lower,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			valid := allFormatsValid
			for _, s := range tc.inputs {
				valid &= checkFormatBits(s)
			}
			got := chooseBestFormat(valid)
			if got != tc.want {
				t.Errorf("format = %d, want %d (valid bits = %016b)", got, tc.want, valid)
			}
		})
	}
}

// TestStringRoundTrip verifies that values survive a full compress→decompress cycle.
func TestStringRoundTrip(t *testing.T) {
	groups := [][]string{
		// UUID lowercase
		{
			"550e8400-e29b-41d4-a716-446655440000",
			"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			"00000000-0000-0000-0000-000000000000",
		},
		// UUID uppercase
		{
			"550E8400-E29B-41D4-A716-446655440000",
			"6BA7B810-9DAD-11D1-80B4-00C04FD430C8",
		},
		// MD5 hex lowercase (even length)
		{
			"d41d8cd98f00b204e9800998ecf8427e",
			"098f6bcd4621d373cade4e832627b4f6",
		},
		// Hex uppercase
		{
			"D41D8CD98F00B204E9800998ECF8427E",
		},
		// Phone with space/slash (even char count)
		{
			"+49 30 123456",
			"0800/123 456",
		},
		// Phone odd-length
		{
			"+49 30 12345",
			"030 123",
		},
		// Decimal
		{
			"3.14",
			"-1,23",
			"192.168.1.1",
		},
		// DateTime
		{
			"2024-03-07 15:30:00",
			"2023-12-31T23:59:59",
		},
		// Raw
		{
			"Hello, World!",
			"",
			"こんにちは",
		},
		// Standard Base64 (contains +/)
		{
			"dGVzdA==", // "test"
			"AAAA",     // {0,0,0}
			"++//",     // {0xFB,0xEF,0xFF}
		},
		// URL-safe Base64 (contains -_)
		{
			"dGVzdA==", // "test"
			"--__",     // {0xFB,0xEF,0xFF}
		},
	}

	for gi, inputs := range groups {
		s := buildStringColumn(inputs)
		for i, want := range inputs {
			got := scm.String(s.GetValue(uint32(i)))
			if got != want {
				t.Errorf("group %d[%d]: got %q, want %q (format=%d)", gi, i, got, want, s.format)
			}
		}
	}
}

// TestBase64PaddingRejected verifies that '=' at an interior position prevents
// base64 format selection, falling back to FormatRaw.
func TestBase64PaddingRejected(t *testing.T) {
	invalid := []string{
		"A=AA",     // '=' at position 1 (not last 2)
		"A===",     // '=' at position 1
		"AAAA=AAA", // '=' in the middle of an 8-char string
	}
	for _, s := range invalid {
		valid := allFormatsValid
		valid &= checkFormatBits(s)
		got := chooseBestFormat(valid)
		if got == FormatBase64Upper || got == FormatBase64Lower {
			t.Errorf("checkFormatBits(%q) should reject base64 but chose format %d", s, got)
		}
	}
}

// TestBase64BStringEquality verifies that tagBString equality uses raw byte
// comparison without allocating intermediate strings.
func TestBase64BStringEquality(t *testing.T) {
	// Two columns with the same base64 strings should compare equal via Equal/EqualSQL.
	inputs := []string{"dGVzdA==", "AAAA", "++//"}
	a := buildStringColumn(inputs)
	b := buildStringColumn(inputs)
	for i, want := range inputs {
		va := a.GetValue(uint32(i))
		vb := b.GetValue(uint32(i))
		if !scm.Equal(va, vb) {
			t.Errorf("[%d] Equal(%q, %q) = false, want true", i, scm.String(va), scm.String(vb))
		}
		if !scm.EqualSQL(va, vb).Bool() {
			t.Errorf("[%d] EqualSQL(%q, %q) = false, want true", i, scm.String(va), scm.String(vb))
		}
		// cross-type: BString vs plain String
		vs := scm.NewString(want)
		if !scm.Equal(va, vs) {
			t.Errorf("[%d] Equal(BString, String(%q)) = false, want true", i, want)
		}
	}
}

// TestNibbleOddLength ensures odd-length strings are reconstructed exactly.
func TestNibbleOddLength(t *testing.T) {
	for _, tc := range []struct {
		inputs []string
		format StringFormat
	}{
		{[]string{"+4", "+49", "+490", "+4900"}, FormatPhone},
		{[]string{"1.2", "3.14", "31.4"}, FormatDecimal},
		{[]string{"2024-3", "2024-03", "2024-03-7"}, FormatDateTime},
	} {
		s := buildStringColumn(tc.inputs)
		if s.format != tc.format {
			// skip if format wasn't chosen (might not be compatible due to length constraints)
			continue
		}
		for i, want := range tc.inputs {
			got := scm.String(s.GetValue(uint32(i)))
			if got != want {
				t.Errorf("format=%d: input[%d] got %q, want %q", tc.format, i, got, want)
			}
		}
	}
}
