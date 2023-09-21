/*
Copyright (C) 2023  Carl-Philip Hänsch
Copyright (C) 2013  Pieter Kelchtermans (originally licensed unter WTFPL 2.0)

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
/*
 * A minimal Scheme interpreter, as seen in lis.py and SICP
 * http://norvig.com/lispy.html
 * http://mitpress.mit.edu/sicp/full-text/sicp/book/node77.html
 *
 * Pieter Kelchtermans 2013
 * LICENSE: WTFPL 2.0
 */
package scm

import "strconv"
import "reflect"

//go:inline
func ToBool(v Scmer) bool {
	switch v.(type) {
		case nil:
			return false
		case string:
			return v != ""
		case float64:
			return v != 0.0
		case bool:
			return v != false
		default:
			// []Scmer, native function, lambdas
			return true
	}
}
//go:inline
func ToInt(v Scmer) int {
	switch vv := v.(type) {
		case nil:
			return 0
		case string:
			x, _ := strconv.Atoi(vv)
			return x
		case float64:
			return int(vv)
		case bool:
			if vv {
				return 1
			} else {
				return 0
			}
		default:
			// []Scmer, native function, lambdas
			return 1
	}
}
//go:inline
func ToFloat(v Scmer) float64 {
	switch vv := v.(type) {
		case string:
			x, _ := strconv.ParseFloat(vv, 64)
			return x
		case float64:
			return vv
		case bool:
			if vv {
				return 1.0
			} else {
				return 0.0
			}
		default:
			// nil, []Scmer, native function, lambdas
			return 0.0
	}
}


func init_alu() {
	// string functions
	DeclareTitle("Arithmetic / Logic")

	Declare(&Globalenv, &Declaration{
		"+", "adds two or more numbers",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values to add"},
		}, "number",
		func(a ...Scmer) Scmer {
			v := ToFloat(a[0])
			for _, i := range a[1:] {
				v += ToFloat(i)
			}
			return v
		},
	})
	Declare(&Globalenv, &Declaration{
		"-", "subtracts two or more numbers from the first one",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values"},
		}, "number",
		func(a ...Scmer) Scmer {
			v := ToFloat(a[0])
			for _, i := range a[1:] {
				v -= ToFloat(i)
			}
			return v
		},
	})
	Declare(&Globalenv, &Declaration{
		"*", "multiplies two or more numbers",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values"},
		}, "number",
		func(a ...Scmer) Scmer {
			v := ToFloat(a[0])
			for _, i := range a[1:] {
				v *= ToFloat(i)
			}
			return v
		},
	})
	Declare(&Globalenv, &Declaration{
		"/", "divides two or more numbers from the first one",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values"},
		}, "number",
		func(a ...Scmer) Scmer {
			v := ToFloat(a[0])
			for _, i := range a[1:] {
				v /= ToFloat(i)
			}
			return v
		},
	})
	Declare(&Globalenv, &Declaration{
		"<=", "compares two numbers",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values"},
		}, "bool",
		func(a ...Scmer) Scmer {
			// TODO: string vs. float
			return a[0].(float64) <= a[1].(float64)
		},
	})
	Declare(&Globalenv, &Declaration{
		"<", "compares two numbers",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values"},
		}, "bool",
		func(a ...Scmer) Scmer {
			return a[0].(float64) < a[1].(float64)
		},
	})
	Declare(&Globalenv, &Declaration{
		">", "compares two numbers",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values"},
		}, "bool",
		func(a ...Scmer) Scmer {
			return a[0].(float64) > a[1].(float64)
		},
	})
	Declare(&Globalenv, &Declaration{
		">=", "compares two numbers",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values"},
		}, "bool",
		func(a ...Scmer) Scmer {
			return a[0].(float64) >= a[1].(float64)
		},
	})
	Declare(&Globalenv, &Declaration{
		"equal?", "compares two values of the same type",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values"},
		}, "bool",
		func(a ...Scmer) Scmer {
			return reflect.DeepEqual(a[0], a[1])
		},
	})
	Declare(&Globalenv, &Declaration{
		"!", "negates the boolean value",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "bool", "value"},
		}, "bool",
		func(a ...Scmer) Scmer {
			return !ToBool(a[0]);
		},
	})
	Declare(&Globalenv, &Declaration{
		"not", "negates the boolean value",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "bool", "value"},
		}, "bool",
		func(a ...Scmer) Scmer {
			return !ToBool(a[0]);
		},
	})
}
