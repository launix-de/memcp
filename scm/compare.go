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
				return a_ < b_
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

