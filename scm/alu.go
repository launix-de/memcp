/*
Copyright (C) 2023-2026  Carl-Philip Hänsch
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

import (
	crand "crypto/rand"
	"encoding/binary"
	"math"
	"strings"
)

func init_alu() {
	// string functions
	DeclareTitle("Arithmetic / Logic")

		Declare(&Globalenv, &Declaration{
		Name: "int?",
		Desc: "tells if the value is a integer",
		Fn: func(a ...Scmer) Scmer {
				return NewBool(a[0].GetTag() == tagInt)
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "value", ParamDesc: "value"}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
			JITEmit: func(ctx *JITContext, args []Scmer, descs []JITValueDesc, result JITValueDesc) JITValueDesc {
				d0 := descs[0]
				r0 := ctx.AllocReg()
				ctx.W.EmitGetTag(r0, d0.Reg, d0.Reg2)
				ctx.FreeDesc(&d0)
				ctx.W.EmitCmpRegImm32(r0, 4)
				ctx.W.EmitSetcc(r0, CcE)
				ctx.W.EmitMakeBool(result, JITValueDesc{Loc: LocReg, Reg: r0})
				return result
			},
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "number?",
		Desc: "tells if the value is a number",
		Fn: func(a ...Scmer) Scmer {
				tag := a[0].GetTag()
				return NewBool(tag == tagFloat || tag == tagInt || tag == tagDate)
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "value", ParamDesc: "value"}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "+",
		Desc: "adds two or more numbers",
		Fn: func(a ...Scmer) Scmer {
				// Fast path: accumulate ints until first non-int, then promote to float if needed
				var sumInt int64
				i := 0
				for i < len(a) {
					v := a[i]
					if v.IsInt() {
						sumInt += v.Int()
						i++
						continue
					}
					break
				}
				if i == len(a) {
					return NewInt(sumInt)
				}
				// Promote to float and continue
				sumFloat := float64(sumInt)
				for ; i < len(a); i++ {
					v := a[i]
					if v.IsNil() {
						return NewNil()
					}
					sumFloat += v.Float()
				}
				return NewFloat(sumFloat)
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "number", ParamName: "value...", ParamDesc: "values to add", Variadic: true}},
			Return: &TypeDescriptor{Kind: "number"},
			Const: true,
			Optimize: optimizeAssociative,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "-",
		Desc: "subtracts two or more numbers from the first one",
		Fn: func(a ...Scmer) Scmer {
				// Nil short-circuit
				for _, v := range a {
					if v.IsNil() {
						return NewNil()
					}
				}
				// Int-first, then promote to float if needed
				if a[0].IsInt() {
					diffInt := a[0].Int()
					i := 1
					for i < len(a) && a[i].IsInt() {
						diffInt -= a[i].Int()
						i++
					}
					if i == len(a) {
						return NewInt(diffInt)
					}
					diffFloat := float64(diffInt)
					for ; i < len(a); i++ {
						diffFloat -= a[i].Float()
					}
					return NewFloat(diffFloat)
				}
				// Float mode from the start
				diffFloat := a[0].Float()
				for i := 1; i < len(a); i++ {
					diffFloat -= a[i].Float()
				}
				return NewFloat(diffFloat)
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "number", ParamName: "value...", ParamDesc: "values", Variadic: true}},
			Return: &TypeDescriptor{Kind: "number"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "*",
		Desc: "multiplies two or more numbers",
		Fn: func(a ...Scmer) Scmer {
				// Nil short-circuit (SQL-style): if any arg is nil, result is nil
				for _, v := range a {
					if v.IsNil() {
						return NewNil()
					}
				}
				// Try integer mode: treat float operands with zero fractional part as integers
				prodInt := int64(1)
				i := 0
				for ; i < len(a); i++ {
					v := a[i]
					if v.IsInt() {
						prodInt *= v.Int()
						continue
					}
					if v.IsFloat() {
						f := v.Float()
						if f == math.Trunc(f) {
							prodInt *= int64(f)
							continue
						}
					}
					break // non-integer number encountered -> switch to float mode
				}
				if i == len(a) {
					return NewInt(prodInt)
				}
				// Float mode: include any prior integer product and continue in float
				prodFloat := float64(prodInt)
				for ; i < len(a); i++ {
					prodFloat *= a[i].Float()
				}
				return NewFloat(prodFloat)
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "number", ParamName: "value...", ParamDesc: "values", Variadic: true}},
			Return: &TypeDescriptor{Kind: "number"},
			Const: true,
			Optimize: optimizeAssociative,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "/",
		Desc: "divides two or more numbers from the first one",
		Fn: func(a ...Scmer) Scmer {
				// Nil short-circuit
				for _, v := range a {
					if v.IsNil() {
						return NewNil()
					}
				}
				v := a[0].Float()
				for _, i := range a[1:] {
					v /= i.Float()
				}
				return NewFloat(v)
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "number", ParamName: "value...", ParamDesc: "values", Variadic: true}},
			Return: &TypeDescriptor{Kind: "number"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "<=",
		Desc: "compares two numbers or strings",
		Fn: func(a ...Scmer) Scmer {
				return NewBool(!Less(a[1], a[0]))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "value...", ParamDesc: "values", Variadic: true}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "<",
		Desc: "compares two numbers or strings",
		Fn: func(a ...Scmer) Scmer {
				return NewBool(Less(a[0], a[1]))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "value...", ParamDesc: "values", Variadic: true}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: ">",
		Desc: "compares two numbers or strings",
		Fn: func(a ...Scmer) Scmer {
				return NewBool(Less(a[1], a[0]))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "value...", ParamDesc: "values", Variadic: true}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: ">=",
		Desc: "compares two numbers or strings",
		Fn: func(a ...Scmer) Scmer {
				return NewBool(!Less(a[0], a[1]))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "value...", ParamDesc: "values", Variadic: true}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "equal?",
		Desc: "compares two values of the same type, (equal? nil nil) is true",
		Fn: func(a ...Scmer) Scmer {
				return NewBool(Equal(a[0], a[1]))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "a", ParamDesc: "first value"}, &TypeDescriptor{Kind: "any", ParamName: "b", ParamDesc: "second value"}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "equal??",
		Desc: "performs a SQL compliant sloppy equality check on primitive values (number, int, string, bool. nil), strings are compared case insensitive, (equal? nil nil) is nil",
		Fn: func(a ...Scmer) Scmer {
				return EqualSQL(a[0], a[1])
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "a", ParamDesc: "first value"}, &TypeDescriptor{Kind: "any", ParamName: "b", ParamDesc: "second value"}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "equal_collate",
		Desc: "performs SQL equality with a specified collation (e.g. *_ci case-insensitive, *_bin case-sensitive); returns nil if either arg is nil",
		Fn: func(a ...Scmer) Scmer {
				if a[0].IsNil() || a[1].IsNil() {
					return NewNil()
				}
				coll := strings.ToLower(String(a[2]))
				ta := a[0].GetTag()
				tb := a[1].GetTag()
				if (ta == tagString || ta == tagSymbol) && (tb == tagString || tb == tagSymbol) {
					as := a[0].String()
					bs := a[1].String()
					if strings.Contains(coll, "_ci") {
						return NewBool(strings.EqualFold(as, bs))
					}
					return NewBool(as == bs)
				}
				return EqualSQL(a[0], a[1])
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "a", ParamDesc: "left side"}, &TypeDescriptor{Kind: "any", ParamName: "b", ParamDesc: "right side"}, &TypeDescriptor{Kind: "string", ParamName: "collation", ParamDesc: "collation name"}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "notequal_collate",
		Desc: "performs SQL inequality with a specified collation; returns nil if either arg is nil",
		Fn: func(a ...Scmer) Scmer {
				r := Globalenv.Vars["equal_collate"].Func()(a[0], a[1], a[2])
				if r.IsNil() {
					return r
				}
				return NewBool(!r.Bool())
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "a", ParamDesc: "left side"}, &TypeDescriptor{Kind: "any", ParamName: "b", ParamDesc: "right side"}, &TypeDescriptor{Kind: "string", ParamName: "collation", ParamDesc: "collation name"}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "!",
		Desc: "negates the boolean value",
		Fn: func(a ...Scmer) Scmer {
				return NewBool(!a[0].Bool())
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "bool", ParamName: "value", ParamDesc: "value"}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "not",
		Desc: "negates the boolean value",
		Fn: func(a ...Scmer) Scmer {
				return NewBool(!a[0].Bool())
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "bool", ParamName: "value", ParamDesc: "value"}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "nil?",
		Desc: "returns true if value is nil",
		Fn: func(a ...Scmer) Scmer {
				return NewBool(a[0].IsNil())
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "value", ParamDesc: "value"}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "min",
		Desc: "returns the smallest value",
		Fn: func(a ...Scmer) Scmer {
				var result Scmer
				for _, v := range a {
					if result.IsNil() {
						result = v
					} else if !v.IsNil() && Less(v, result) {
						result = v
					}
				}
				return result
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "number|string", ParamName: "value...", ParamDesc: "value", Variadic: true}},
			Return: &TypeDescriptor{Kind: "number|string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "max",
		Desc: "returns the highest value",
		Fn: func(a ...Scmer) Scmer {
				var result Scmer
				for _, v := range a {
					if result.IsNil() {
						result = v
					} else if !v.IsNil() && Less(result, v) {
						result = v
					}
				}
				return result
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "number|string", ParamName: "value...", ParamDesc: "value", Variadic: true}},
			Return: &TypeDescriptor{Kind: "number|string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "floor",
		Desc: "rounds the number down",
		Fn: func(a ...Scmer) Scmer {
				return NewFloat(math.Floor(a[0].Float()))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "number", ParamName: "value", ParamDesc: "value"}},
			Return: &TypeDescriptor{Kind: "number"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "ceil",
		Desc: "rounds the number up",
		Fn: func(a ...Scmer) Scmer {
				return NewFloat(math.Ceil(a[0].Float()))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "number", ParamName: "value", ParamDesc: "value"}},
			Return: &TypeDescriptor{Kind: "number"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "round",
		Desc: "rounds the number",
		Fn: func(a ...Scmer) Scmer {
				return NewFloat(math.Round(a[0].Float()))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "number", ParamName: "value", ParamDesc: "value"}},
			Return: &TypeDescriptor{Kind: "number"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "sql_abs",
		Desc: "SQL ABS(): returns absolute value, NULL-safe",
		Fn: func(a ...Scmer) Scmer {
				if a[0].IsNil() {
					return NewNil()
				}
				v := a[0].Float()
				if v < 0 {
					v = -v
				}
				// preserve int type
				if ToInt(a[0]) == int(v) && a[0].Float() == v {
					return NewInt(int64(v))
				}
				return NewFloat(v)
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "number", ParamName: "value", ParamDesc: "value"}},
			Return: &TypeDescriptor{Kind: "number"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "sqrt",
		Desc: "returns the square root of a number",
		Fn: func(a ...Scmer) Scmer {
				if a[0].IsNil() {
					return NewNil()
				}
				v := a[0].Float()
				if v < 0 {
					return NewNil()
				}
				return NewFloat(math.Sqrt(v))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "number", ParamName: "value", ParamDesc: "value"}},
			Return: &TypeDescriptor{Kind: "number"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "sql_rand",
		Desc: "SQL RAND(): returns a random float in [0,1)",
		Fn: func(a ...Scmer) Scmer {
				var buf [8]byte
				if _, err := crand.Read(buf[:]); err != nil {
					panic("sql_rand: " + err.Error())
				}
				// 53 random bits map exactly into float64 mantissa range.
				u := binary.LittleEndian.Uint64(buf[:]) >> 11
				return NewFloat(float64(u) / (1 << 53))
			},
		Type: &TypeDescriptor{
			Return: &TypeDescriptor{Kind: "number"},
			Const: true,
		},
	})
}
