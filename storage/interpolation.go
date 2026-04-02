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
	"math"
	"sort"

	"github.com/launix-de/memcp/scm"
)

// interpolationSearch finds the leftmost position where key could be inserted
// in the sorted range [lo, lo+n).  Uses interpolation to estimate the position
// for int, float, and string types; falls back to binary search for unknown types
// or when min/max are unavailable.
//
// getVal reads the value at a given position in the sorted index.
// minVal/maxVal are the first/last values in the sorted range.
func interpolationSearch(lo, n int, key scm.Scmer, minVal, maxVal scm.Scmer, getVal func(int) scm.Scmer) int {
	if n <= 0 {
		return lo
	}
	// Try interpolation if we have usable min/max
	if !minVal.IsNil() && !maxVal.IsNil() && !key.IsNil() {
		frac := valueFraction(key, minVal, maxVal)
		if frac >= 0 && frac <= 1 {
			// Interpolation guess
			guess := lo + int(frac*float64(n-1))
			if guess < lo {
				guess = lo
			}
			if guess >= lo+n {
				guess = lo + n - 1
			}
			// Probe the guess position
			gv := getVal(guess)
			if scm.Less(key, gv) || scm.Equal(key, gv) {
				// key <= guess: search in [lo, guess+1)
				upperN := guess - lo + 1
				if upperN > n {
					upperN = n
				}
				return lo + sort.Search(upperN, func(i int) bool {
					return !scm.Less(getVal(lo+i), key) // getVal(pos) >= key
				})
			}
			// key > guess: search in [guess+1, lo+n)
			newLo := guess + 1
			newN := lo + n - newLo
			if newN <= 0 {
				return lo + n
			}
			return newLo + sort.Search(newN, func(i int) bool {
				return !scm.Less(getVal(newLo+i), key)
			})
		}
	}
	// Fallback: plain binary search
	return lo + sort.Search(n, func(i int) bool {
		return !scm.Less(getVal(lo+i), key)
	})
}

// valueFraction estimates where key falls between minVal and maxVal as a
// fraction in [0.0, 1.0].  Returns -1 if the types are not interpolatable.
//
// Supports: int, float, string (common-prefix-skip + next-byte interpolation).
func valueFraction(key, minVal, maxVal scm.Scmer) float64 {
	// Int
	if key.IsInt() && minVal.IsInt() && maxVal.IsInt() {
		lo := minVal.Int()
		hi := maxVal.Int()
		if hi == lo {
			return 0.5
		}
		k := key.Int()
		return float64(k-lo) / float64(hi-lo)
	}
	// Float
	if key.IsFloat() && minVal.IsFloat() && maxVal.IsFloat() {
		lo := minVal.Float()
		hi := maxVal.Float()
		if hi == lo {
			return 0.5
		}
		k := key.Float()
		return (k - lo) / (hi - lo)
	}
	// Int key vs float boundaries or vice versa: convert to float
	kf, kok := toFloat(key)
	lof, lok := toFloat(minVal)
	hif, hok := toFloat(maxVal)
	if kok && lok && hok && hif != lof {
		return (kf - lof) / (hif - lof)
	}
	// String: skip common prefix, interpolate on divergence byte
	if key.IsString() && minVal.IsString() && maxVal.IsString() {
		return stringFraction(key.String(), minVal.String(), maxVal.String())
	}
	return -1 // unknown type, no interpolation
}

// toFloat converts an int or float Scmer to float64.
func toFloat(v scm.Scmer) (float64, bool) {
	if v.IsInt() {
		return float64(v.Int()), true
	}
	if v.IsFloat() {
		return v.Float(), true
	}
	return 0, false
}

// stringFraction estimates where key falls between lo and hi by skipping
// the common prefix and interpolating on the first divergent byte.
func stringFraction(key, lo, hi string) float64 {
	// Find common prefix length
	minLen := len(lo)
	if len(hi) < minLen {
		minLen = len(hi)
	}
	if len(key) < minLen {
		minLen = len(key)
	}
	prefix := 0
	for prefix < minLen && lo[prefix] == hi[prefix] {
		prefix++
	}
	// Extract the byte(s) after the common prefix for interpolation
	kb := stringByteAt(key, prefix)
	lb := stringByteAt(lo, prefix)
	hb := stringByteAt(hi, prefix)
	if hb <= lb {
		return 0.5
	}
	frac := float64(kb-lb) / float64(hb-lb)
	// Clamp to [0, 1]
	return math.Max(0, math.Min(1, frac))
}

// stringByteAt returns the byte value at position i, or 0 if beyond string end.
func stringByteAt(s string, i int) int {
	if i < len(s) {
		return int(s[i])
	}
	return 0
}
