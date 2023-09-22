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

func Compare(a, b Scmer) int {
	// TODO: nil
	switch a_ := a.(type) {
		case float64:
			b_ := ToFloat(b)
			if a_ == b_ {
				return 0
			}
			if a_ < b_ {
				return -1
			}
			return 1
		case string:
			b_ := String(b)
			if a_ == b_ {
				return 0
			}
			if a_ < b_ {
				return -1
			}
			return 1
		default:
			panic("Cannot compare " + fmt.Sprint(a) + " and " + fmt.Sprint(b))
	}
}
