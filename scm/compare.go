/*
Copyright (C) 2023  Carl-Philip Hänsch

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
package scm

import "fmt"
import "strings"
import "unicode"
import "unicode/utf8"

func Equal(a, b Scmer) Scmer {
	// == NULL is always NULL
	if a == nil || b == nil {
		return nil
	}
	switch a_ := a.(type) {
		case string:
			switch b_ := b.(type) {
				case string:
					return strings.EqualFold(a_, b_)
				case float64:
					return a_ == String(b_)
				case bool:
					return ToBool(a) == b_
			}
		case float64:
			switch b_ := b.(type) {
				case string:
					return String(a_) == b_
				case float64:
					return a_ == b_
				case bool:
					return ToBool(a) == b_
			}
		case bool:
			switch b_ := b.(type) {
				case string:
					return a_ == ToBool(b)
				case float64:
					return a_ == ToBool(b)
				case bool:
					return a_ == b_
			}

	}
	panic("unknown comparison: " + fmt.Sprint(a) + " and " + fmt.Sprint(b))
}

// sort function for scmer
func Less(a, b Scmer) bool {
	switch a_ := a.(type) {
	case nil:
		return b != nil // nil is always less than any other value except for nil (which is equal)
	case int, uint, int64, uint64:
		return ToFloat(a) < ToFloat(b) // todo: more fine grained
	case float64:
		return a_ < ToFloat(b)
	case string:
		switch b_ := b.(type) {
			case float64:
				return ToFloat(a) < b_
			case string:
				return StringLess(a_, b_)
			case nil:
				return false
			default:
				panic("unknown type combo in comparison")
		}
	// are there any other types??
	default:
		panic("unknown type combo in comparison: " + String(a) + " < " + String(b))
	}
	return false
}

func StringLess(a, b string) bool {
    for {
        if len(a) == 0 {
            return false
        }
        if len(b) == 0 {
            return true
        }
        ar, sa := utf8.DecodeRuneInString(a)
        br, sb := utf8.DecodeRuneInString(b)

	// if case insensitive
        al := unicode.ToLower(ar)
        bl := unicode.ToLower(br)

        if al < bl {
            return true
        } else if al > bl {
            return false
        } else {
		a = a[sa:]
		b = b[sb:]
	}
    }
}
