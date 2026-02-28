/*
Copyright (C) 2025  Carl-Philip Hänsch

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

import (
	"math"
	"strings"
)

func init_vector() {
	// string functions
	DeclareTitle("Vectors")

	Declare(&Globalenv, &Declaration{
		"dot", "produced the dot product",
		2, 3,
		[]DeclarationParameter{
			DeclarationParameter{"v1", "list", "vector1", nil},
			DeclarationParameter{"v2", "list", "vector2", nil},
			DeclarationParameter{"mode", "string", "DOT, COSINE, EUCLIDEAN, default is DOT", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			var result float64
			v1 := asSlice(a[0], "dot v1")
			v2 := asSlice(a[1], "dot v2")
			mode := "DOT"
			if len(a) > 2 {
				mode = strings.ToUpper(String(a[2]))
			}
			if mode == "COSINE" {
				// COSINE
				var lena float64 = 0
				var lenb float64 = 0
				for i := 0; i < len(v1) && i < len(v2); i++ {
					w1 := ToFloat(v1[i])
					w2 := ToFloat(v2[i])
					lena += w1 * w1
					lenb += w2 * w2
					result += w1 * w2
				}
				result = result / math.Sqrt(lena*lenb)
			} else {
				// DOT AND EUCLIDEAN
				for i := 0; i < len(v1) && i < len(v2); i++ {
					result += ToFloat(v1[i]) * ToFloat(v2[i])
				}
				if mode == "EUCLIDEAN" {
					result = math.Sqrt(result)
				}
			}
			return NewFloat(result)
		}, true, false, nil,
		nil,
	})
}
