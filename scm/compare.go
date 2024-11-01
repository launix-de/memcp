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

func EqualScm(a, b Scmer) Scmer { // case sensitive and can compare nil
	return Equal(a, b)
}
func Equal(a, b Scmer) bool { // case sensitive and can compare nil
	switch a_ := a.(type) {
		case LazyString:
			switch b_ := b.(type) {
				case LazyString:
					return a_.Hash == b_.Hash
				case string:
					return a_.GetValue() == b_
				case float64:
					return ToFloat(a_.GetValue()) == b_
				case int64:
					return ToInt(a_.GetValue()) == int(b_)
				case bool:
					return ToBool(a) == b_
				case []Scmer:
					return len(b_) == 0 && !ToBool(a)
				case nil:
					return !ToBool(a)
			}
		case string:
			switch b_ := b.(type) {
				case LazyString:
					return a_ == b_.GetValue()
				case string:
					return a_ == b_
				case float64:
					return ToFloat(a_) == b_
				case int64:
					return ToInt(a_) == int(b_)
				case bool:
					return ToBool(a) == b_
				case []Scmer:
					return len(b_) == 0 && !ToBool(a)
				case nil:
					return !ToBool(a)
			}
		case float64:
			switch b_ := b.(type) {
				case LazyString:
					return String(a_) == b_.GetValue()
				case string:
					return a_ == ToFloat(b_)
				case float64:
					return a_ == b_
				case int64:
					return a_ == float64(b_)
				case bool:
					return ToBool(a) == b_
				case []Scmer:
					return len(b_) == 0 && !ToBool(a)
				case nil:
					return !ToBool(a)
			}
		case int64:
			switch b_ := b.(type) {
				case LazyString:
					return String(a_) == b_.GetValue()
				case string:
					return int(a_) == ToInt(b_)
				case float64:
					return float64(a_) == b_
				case int64:
					return a_ == b_
				case bool:
					return ToBool(a) == b_
				case []Scmer:
					return len(b_) == 0 && !ToBool(a)
				case nil:
					return !ToBool(a)
			}
		case bool:
			switch b_ := b.(type) {
				case LazyString:
					return a_ == ToBool(b)
				case string:
					return a_ == ToBool(b)
				case float64:
					return a_ == ToBool(b)
				case int64:
					return a_ == ToBool(b)
				case bool:
					return a_ == b_
				case []Scmer:
					return len(b_) == 0 && !ToBool(a)
				case nil:
					return !a_
			}
		case nil:
			switch b_ := b.(type) {
				case LazyString:
					return !ToBool(b)
				case string:
					return !ToBool(b)
				case float64:
					return !ToBool(b)
				case int64:
					return !ToBool(b)
				case bool:
					return !b_
				case nil:
					return true
				case []Scmer:
					return len(b_) == 0 && !ToBool(a)
			}
		case []Scmer:
			switch b_ := b.(type) {
				case LazyString, string, float64, int64, bool, nil:
					return len(a_) == 0 && !ToBool(b)
				case []Scmer:
					if len(a_) != len(b_) {
						return false
					}
					for i, v := range a_ {
						if !Equal(v, b_[i]) {
							return false
						}
					}
					return true
			}

	}
	panic("unknown comparison: " + fmt.Sprint(a) + " and " + fmt.Sprint(b))
}

func EqualSQL(a, b Scmer) Scmer {
	// == NULL is always NULL
	if a == nil || b == nil {
		return nil
	}
	switch a_ := a.(type) {
		case LazyString:
			switch b_ := b.(type) {
				case LazyString:
					return a_.Hash == b_.Hash
				case string:
					return strings.EqualFold(a_.GetValue(), b_)
				case float64:
					return ToFloat(a_.GetValue()) == b_
				case int64:
					return ToInt(a_.GetValue()) == int(b_)
				case bool:
					return ToBool(a) == b_
				case []Scmer:
					return len(b_) == 0 && !ToBool(a)
			}
		case string:
			switch b_ := b.(type) {
				case LazyString:
					return strings.EqualFold(a_, b_.GetValue())
				case string:
					return strings.EqualFold(a_, b_)
				case float64:
					return ToFloat(a_) == b_
				case int64:
					return ToInt(a_) == int(b_)
				case bool:
					return ToBool(a) == b_
				case []Scmer:
					return len(b_) == 0 && !ToBool(a)
			}
		case float64:
			switch b_ := b.(type) {
				case LazyString:
					return String(a_) == b_.GetValue()
				case string:
					return a_ == ToFloat(b_)
				case float64:
					return a_ == b_
				case int64:
					return a_ == float64(b_)
				case bool:
					return ToBool(a) == b_
				case []Scmer:
					return len(b_) == 0 && !ToBool(a)
			}
		case int64:
			switch b_ := b.(type) {
				case LazyString:
					return String(a_) == b_.GetValue()
				case string:
					return int(a_) == ToInt(b_)
				case float64:
					return float64(a_) == b_
				case int64:
					return a_ == b_
				case bool:
					return ToBool(a) == b_
				case []Scmer:
					return len(b_) == 0 && !ToBool(a)
			}
		case bool:
			switch b_ := b.(type) {
				case LazyString:
					return a_ == ToBool(b)
				case string:
					return a_ == ToBool(b)
				case float64:
					return a_ == ToBool(b)
				case int64:
					return a_ == ToBool(b)
				case bool:
					return a_ == b_
				case []Scmer:
					return len(b_) == 0 && !ToBool(a)
			}
		case []Scmer:
			switch b_ := b.(type) {
				case LazyString, string, float64, int64, bool, nil:
					return len(a_) == 0 && !ToBool(b)
				case []Scmer:
					if len(a_) != len(b_) {
						return false
					}
					for i, v := range a_ {
						if !ToBool(EqualSQL(v, b_[i])) {
							return false
						}
					}
					return true
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
	case LazyString:
		switch b_ := b.(type) {
			case float64:
				return ToFloat(a) < b_
			case int64:
				return ToInt(a) < int(b_)
			case LazyString:
				return StringLess(a_.GetValue(), b_.GetValue())
			case string:
				return StringLess(a_.GetValue(), b_)
			case nil:
				return false
			default:
				panic("unknown type combo in comparison")
		}
	case string:
		switch b_ := b.(type) {
			case float64:
				return ToFloat(a) < b_
			case int64:
				return ToInt(a) < int(b_)
			case LazyString:
				return StringLess(a_, b_.GetValue())
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
