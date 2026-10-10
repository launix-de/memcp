/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

// Coefficients are ordinary strings, interpreted only by these explicit
// operators. The biased signed-128 encoding preserves integer order under a
// binary string comparator. Scale and application-level limits are not stored.
func coefficientBias() *big.Int { return new(big.Int).Lsh(big.NewInt(1), 127) }

// Integral arguments accept the runtime's ordinary numeric literals without
// changing generic conversions. Stored integers never pass through a float.
func integralArgument(value Scmer) int64 {
	if value.IsInt() {
		return value.Int()
	}
	if value.IsFloat() {
		x := value.Float()
		if !math.IsNaN(x) && !math.IsInf(x, 0) && x == math.Trunc(x) && x >= -0x1p63 && x < 0x1p63 {
			return int64(x)
		}
	}
	panic("explicit integer operation requires a finite integral signed-64-bit number")
}

func encodeCoefficient(integer *big.Int) Scmer {
	bias := coefficientBias()
	if integer.Cmp(new(big.Int).Neg(bias)) < 0 || integer.Cmp(bias) >= 0 {
		panic("signed 128-bit coefficient overflow")
	}
	biased := new(big.Int).Add(integer, bias)
	return NewString(fmt.Sprintf("%032x", biased))
}

func decodeCoefficient(value Scmer) *big.Int {
	if !value.IsString() {
		panic("coefficient requires a canonical string")
	}
	text := value.String()
	if len(text) != 32 || strings.Trim(text, "0123456789abcdef") != "" {
		panic("invalid canonical coefficient")
	}
	integer, ok := new(big.Int).SetString(text, 16)
	if !ok {
		panic("invalid canonical coefficient")
	}
	return integer.Sub(integer, coefficientBias())
}

func fixedPointPower(scale Scmer) *big.Int {
	value := integralArgument(scale)
	if value < 0 || value > 400 {
		panic("fixed-point scale outside supported calculation bounds")
	}
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(value), nil)
}

// roundQuotient rounds a rational to an integer. Truncation is toward zero;
// nearest rounding resolves exact halves away from zero.
func roundQuotient(numerator, denominator *big.Int, truncate bool) *big.Int {
	if denominator.Sign() == 0 {
		panic("integer division by zero")
	}
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(numerator, denominator, remainder)
	if !truncate && new(big.Int).Lsh(new(big.Int).Abs(remainder), 1).Cmp(new(big.Int).Abs(denominator)) >= 0 {
		if numerator.Sign()*denominator.Sign() < 0 {
			quotient.Sub(quotient, big.NewInt(1))
		} else {
			quotient.Add(quotient, big.NewInt(1))
		}
	}
	return quotient
}

func CoefficientEncode(a ...Scmer) Scmer {
	if !a[0].IsString() {
		panic("coefficient_encode requires integer text")
	}
	text := a[0].String()
	integer, ok := new(big.Int).SetString(text, 10)
	if !ok || integer.String() != text {
		panic("coefficient_encode requires canonical integer text")
	}
	return encodeCoefficient(integer)
}

func CoefficientDecode(a ...Scmer) Scmer { return NewString(decodeCoefficient(a[0]).String()) }
func CoefficientCompare(a ...Scmer) Scmer {
	return NewInt(int64(decodeCoefficient(a[0]).Cmp(decodeCoefficient(a[1]))))
}

// FixedPointCompare aligns coefficients only in temporary arbitrary-width
// integers. Neither operand is rescaled into the bounded persisted encoding.
func FixedPointCompare(a ...Scmer) Scmer {
	left := new(big.Int).Mul(decodeCoefficient(a[0]), fixedPointPower(a[3]))
	right := new(big.Int).Mul(decodeCoefficient(a[1]), fixedPointPower(a[2]))
	return NewInt(int64(left.Cmp(right)))
}

func IntegerToCoefficient(a ...Scmer) Scmer {
	return encodeCoefficient(big.NewInt(integralArgument(a[0])))
}

func IntegerOrderKey(a ...Scmer) Scmer {
	return NewString(fmt.Sprintf("%016x", uint64(integralArgument(a[0]))^(uint64(1)<<63)))
}

func IntegerFromOrderKey(a ...Scmer) Scmer {
	if !a[0].IsString() {
		panic("integer_from_order_key requires a string")
	}
	text := a[0].String()
	if len(text) != 16 || strings.Trim(text, "0123456789abcdef") != "" {
		panic("invalid canonical integer order key")
	}
	value, err := strconv.ParseUint(text, 16, 64)
	if err != nil {
		panic(err)
	}
	return NewInt(int64(value ^ (uint64(1) << 63)))
}

func CoefficientToInteger(a ...Scmer) Scmer {
	integer := roundQuotient(decodeCoefficient(a[0]), fixedPointPower(a[1]), a[2].Bool())
	if !integer.IsInt64() {
		panic("signed 64-bit integer overflow")
	}
	return NewInt(integer.Int64())
}

func FixedPointRescale(a ...Scmer) Scmer {
	numerator := new(big.Int).Mul(decodeCoefficient(a[0]), fixedPointPower(a[2]))
	return encodeCoefficient(roundQuotient(numerator, fixedPointPower(a[1]), a[3].Bool()))
}

func FixedPointBinary(a ...Scmer) Scmer {
	left := new(big.Rat).SetFrac(decodeCoefficient(a[1]), fixedPointPower(a[3]))
	right := new(big.Rat).SetFrac(decodeCoefficient(a[2]), fixedPointPower(a[4]))
	result := new(big.Rat)
	switch a[0].String() {
	case "+":
		result.Add(left, right)
	case "-":
		result.Sub(left, right)
	case "*":
		result.Mul(left, right)
	case "/", "%":
		if right.Sign() == 0 {
			panic("fixed-point division by zero")
		}
		result.Quo(left, right)
		if a[0].String() == "%" {
			quotient := new(big.Int).Quo(result.Num(), result.Denom())
			result.Sub(left, new(big.Rat).Mul(new(big.Rat).SetInt(quotient), right))
		}
	default:
		panic("unknown fixed-point operator")
	}
	numerator := new(big.Int).Mul(result.Num(), fixedPointPower(a[5]))
	return encodeCoefficient(roundQuotient(numerator, result.Denom(), a[6].Bool()))
}

func CoefficientFormat(a ...Scmer) Scmer {
	integer := decodeCoefficient(a[0])
	_ = fixedPointPower(a[1])
	scale := int(integralArgument(a[1]))
	negative := integer.Sign() < 0
	text := new(big.Int).Abs(integer).String()
	if scale != 0 {
		if len(text) <= scale {
			text = strings.Repeat("0", scale+1-len(text)) + text
		}
		text = text[:len(text)-scale] + "." + text[len(text)-scale:]
	}
	if negative {
		text = "-" + text
	}
	return NewString(text)
}

// RegisterExactPrimitives adds explicit coefficient operations; ordinary
// runtime arithmetic, comparison and value tags are untouched.
func RegisterExactPrimitives() {
	for _, declaration := range []struct {
		name   string
		fn     func(...Scmer) Scmer
		kinds  []string
		result string
	}{
		{"coefficient_encode", CoefficientEncode, []string{"string"}, "string"},
		{"coefficient_decode", CoefficientDecode, []string{"string"}, "string"},
		{"coefficient_compare", CoefficientCompare, []string{"string", "string"}, "int"},
		{"fixed_point_compare", FixedPointCompare, []string{"string", "string", "number", "number"}, "int"},
		{"integer_to_coefficient", IntegerToCoefficient, []string{"number"}, "string"},
		{"integer_order_key", IntegerOrderKey, []string{"number"}, "string"},
		{"integer_from_order_key", IntegerFromOrderKey, []string{"string"}, "int"},
		{"coefficient_to_integer", CoefficientToInteger, []string{"string", "number", "bool"}, "int"},
		{"fixed_point_rescale", FixedPointRescale, []string{"string", "number", "number", "bool"}, "string"},
		{"fixed_point_binary", FixedPointBinary, []string{"string", "string", "string", "number", "number", "number", "bool"}, "string"},
		{"coefficient_format", CoefficientFormat, []string{"string", "number"}, "string"},
	} {
		params := make([]*TypeDescriptor, len(declaration.kinds))
		for i, kind := range declaration.kinds {
			params[i] = &TypeDescriptor{Kind: kind}
		}
		Declare(&Globalenv, &Declaration{Name: declaration.name, Fn: declaration.fn, Type: &TypeDescriptor{Kind: "func", Description: "Explicit exact coefficient operation", Params: params, Return: &TypeDescriptor{Kind: declaration.result}, Const: true}})
	}
}
