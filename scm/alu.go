/*
Copyright (C) 2023-2024  Carl-Philip Hänsch
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

import "math"
import "strconv"

//go:inline
func ToBool(v Scmer) bool {
	switch v2 := v.(type) {
		case nil:
			return false
		case LazyString:
			return v2.GetValue() != ""
		case string:
			return v2 != ""
		case float64:
			return v2 != 0.0
		case int64:
			return v2 != 0
		case bool:
			return v2 != false
		case Symbol:
			return v2 != Symbol("false") && v2 != Symbol("nil")
		case []Scmer:
			return len(v2) > 0
		default:
			// native function, lambdas
			return true
	}
}
//go:inline
func ToInt(v Scmer) int {
	switch vv := v.(type) {
		case nil:
			return 0
		case LazyString:
			x, _ := strconv.Atoi(vv.GetValue())
			return x
		case string:
			x, _ := strconv.Atoi(vv)
			return x
		case float64:
			return int(vv)
		case int64:
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
		case LazyString:
			x, _ := strconv.ParseFloat(vv.GetValue(), 64)
			return x
		case string:
			x, _ := strconv.ParseFloat(vv, 64)
			return x
		case float64:
			return vv
		case int64:
			return float64(vv)
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
		"int?", "tells if the value is a integer",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value"},
		}, "bool",
		func(a ...Scmer) (result Scmer) {
			_, ok2 := a[0].(int64)
			return ok2
		},
	})
	Declare(&Globalenv, &Declaration{
		"number?", "tells if the value is a number",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value"},
		}, "bool",
		func(a ...Scmer) (result Scmer) {
			_, ok := a[0].(float64)
			if ok {
				return true
			}
			_, ok2 := a[0].(int64)
			return ok2
		},
	})
	Declare(&Globalenv, &Declaration{
		"+", "adds two or more numbers",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values to add"},
		}, "number",
		func(a ...Scmer) Scmer {
			v := float64(0)
			for _, i := range a {
				if i == nil {
					return nil
				}
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
		"<=", "compares two numbers or strings",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values"},
		}, "bool",
		func(a ...Scmer) Scmer {
			return !Less(a[1], a[0])
		},
	})
	Declare(&Globalenv, &Declaration{
		"<", "compares two numbers or strings",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values"},
		}, "bool",
		func(a ...Scmer) Scmer {
			return Less(a[0], a[1])
		},
	})
	Declare(&Globalenv, &Declaration{
		">", "compares two numbers or strings",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values"},
		}, "bool",
		func(a ...Scmer) Scmer {
			return Less(a[1], a[0])
		},
	})
	Declare(&Globalenv, &Declaration{
		">=", "compares two numbers or strings",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values"},
		}, "bool",
		func(a ...Scmer) Scmer {
			return !Less(a[0], a[1])
		},
	})
	Declare(&Globalenv, &Declaration{
		"equal?", "compares two values of the same type, (equal? nil nil) is true",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values"},
		}, "bool",
		func(a ...Scmer) Scmer {
			return Equal(a[0], a[1])
		},
	})
	Declare(&Globalenv, &Declaration{
		"equal??", "performs a SQL compliant sloppy equality check on primitive values (number, int, string, bool. nil), strings are compared case insensitive, (equal? nil nil) is nil",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values"},
		}, "bool",
		func(a ...Scmer) Scmer {
			return EqualSQL(a[0], a[1])
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
	Declare(&Globalenv, &Declaration{
		"nil?", "returns true if value is nil",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value"},
		}, "bool",
		func(a ...Scmer) Scmer {
			return a[0] == nil;
		},
	})
	Declare(&Globalenv, &Declaration{
		"min", "returns the smallest value",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number|string", "value"},
		}, "number|string",
		func(a ...Scmer) (result Scmer) {
			for _, v := range a {
				if result == nil {
					result = v
				} else if v != nil && Less(v, result) {
					result = v
				}
			}
			return
		},
	})
	Declare(&Globalenv, &Declaration{
		"max", "returns the highest value",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number|string", "value"},
		}, "number|string",
		func(a ...Scmer) (result Scmer) {
			for _, v := range a {
				if result == nil {
					result = v
				} else if v != nil && Less(result, v) {
					result = v
				}
			}
			return
		},
	})
	Declare(&Globalenv, &Declaration{
		"floor", "rounds the number down",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "number", "value"},
		}, "number",
		func(a ...Scmer) (result Scmer) {
			return math.Floor(ToFloat(a[0]))
		},
	})
	Declare(&Globalenv, &Declaration{
		"ceil", "rounds the number up",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "number", "value"},
		}, "number",
		func(a ...Scmer) (result Scmer) {
			return math.Ceil(ToFloat(a[0]))
		},
	})
	Declare(&Globalenv, &Declaration{
		"round", "rounds the number",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "number", "value"},
		}, "number",
		func(a ...Scmer) (result Scmer) {
			return math.Round(ToFloat(a[0]))
		},
	})
}
