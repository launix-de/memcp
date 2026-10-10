/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"math/big"
	"time"
)

func unixRate(value Scmer) int64 {
	rate := integralArgument(value)
	if rate < 1 || rate > 1000000000 {
		panic("invalid Unix tick rate")
	}
	return rate
}

func unixInteger(value Scmer) int64 {
	return integralArgument(value)
}

func unixCheckedTicks(seconds, remainder, rate int64) Scmer {
	result := new(big.Int).Mul(big.NewInt(seconds), big.NewInt(rate))
	result.Add(result, big.NewInt(remainder))
	if !result.IsInt64() {
		panic("Unix tick integer overflow")
	}
	return NewInt(result.Int64())
}

func UnixParts(a ...Scmer) Scmer {
	ticks, rate := unixInteger(a[0]), unixRate(a[1])
	seconds, remainder := ticks/rate, ticks%rate
	if remainder < 0 {
		seconds--
		remainder += rate
	}
	t := time.Unix(seconds, 0).UTC()
	year, week := t.ISOWeek()
	return NewSlice([]Scmer{NewInt(int64(t.Year())), NewInt(int64(t.Month())), NewInt(int64(t.Day())), NewInt(int64(t.Hour())), NewInt(int64(t.Minute())), NewInt(int64(t.Second())), NewInt(remainder), NewInt(int64(t.Weekday())), NewInt(int64(t.YearDay())), NewInt(int64(year)), NewInt(int64(week))})
}

func UnixFromParts(a ...Scmer) Scmer {
	y, m, d, h, minute, second, remainder := unixInteger(a[0]), unixInteger(a[1]), unixInteger(a[2]), unixInteger(a[3]), unixInteger(a[4]), unixInteger(a[5]), unixInteger(a[6])
	rate := unixRate(a[7])
	if y < -292277022000 || y > 292277022000 || m < 1 || m > 12 || d < 1 || d > 31 || h < 0 || h > 23 || minute < 0 || minute > 59 || second < 0 || second > 59 || remainder < 0 || remainder >= rate {
		panic("invalid Gregorian calendar fields")
	}
	t := time.Date(int(y), time.Month(m), int(d), int(h), int(minute), int(second), 0, time.UTC)
	if int64(t.Year()) != y || int64(t.Month()) != m || int64(t.Day()) != d {
		panic("invalid Gregorian calendar day")
	}
	return unixCheckedTicks(t.Unix(), remainder, rate)
}

func UnixClock(a ...Scmer) Scmer {
	rate := unixRate(a[0])
	t := time.Now()
	return unixCheckedTicks(t.Unix(), int64(t.Nanosecond())*rate/1000000000, rate)
}

func UnixParse(a ...Scmer) Scmer {
	if !a[0].IsString() || !a[1].IsString() {
		panic("unix_parse requires text and an explicit layout")
	}
	rate, offset := unixRate(a[2]), unixInteger(a[3])
	if offset < -1439 || offset > 1439 {
		panic("invalid fixed timezone offset")
	}
	t, err := time.ParseInLocation(a[1].String(), a[0].String(), time.FixedZone("", int(offset*60)))
	if err != nil {
		panic(err)
	}
	numerator := new(big.Int).Mul(big.NewInt(t.Unix()), big.NewInt(1000000000))
	numerator.Add(numerator, big.NewInt(int64(t.Nanosecond())))
	numerator.Mul(numerator, big.NewInt(rate))
	truncate := true
	if len(a) > 4 {
		truncate = a[4].Bool()
	}
	ticks := roundQuotient(numerator, big.NewInt(1000000000), truncate)
	if !ticks.IsInt64() {
		panic("Unix tick integer overflow")
	}
	return NewInt(ticks.Int64())
}

func UnixFormat(a ...Scmer) Scmer {
	ticks, rate := unixInteger(a[0]), unixRate(a[1])
	if !a[2].IsString() {
		panic("unix_format requires an explicit layout")
	}
	offset := unixInteger(a[3])
	if offset < -1439 || offset > 1439 {
		panic("invalid fixed timezone offset")
	}
	seconds, remainder := ticks/rate, ticks%rate
	if remainder < 0 {
		seconds--
		remainder += rate
	}
	t := time.Unix(seconds, remainder*1000000000/rate).In(time.FixedZone("", int(offset*60)))
	return NewString(t.Format(a[2].String()))
}

// UnixZoneOffset resolves a named zone for an explicit instant. Callers choose
// whether to construct wall-clock values; this operation only returns seconds.
func UnixZoneOffset(a ...Scmer) Scmer {
	ticks, rate := unixInteger(a[0]), unixRate(a[1])
	if !a[2].IsString() {
		panic("unix_zone_offset requires a zone name")
	}
	location, err := ResolveLocation(a[2].String())
	if err != nil {
		panic(err)
	}
	seconds := ticks / rate
	if ticks%rate < 0 {
		seconds--
	}
	_, offset := time.Unix(seconds, 0).In(location).Zone()
	return NewInt(int64(offset))
}

func RegisterUnixPrimitives() {
	Declare(&Globalenv, &Declaration{Name: "unix_parts", Fn: UnixParts, Type: &TypeDescriptor{Kind: "func", Description: "Gregorian fields of an explicitly scaled Unix integer", Params: []*TypeDescriptor{{Kind: "number"}, {Kind: "number"}}, Return: &TypeDescriptor{Kind: "list"}, Const: true}})
	params := make([]*TypeDescriptor, 8)
	for i := range params {
		params[i] = &TypeDescriptor{Kind: "number"}
	}
	Declare(&Globalenv, &Declaration{Name: "unix_from_parts", Fn: UnixFromParts, Type: &TypeDescriptor{Kind: "func", Description: "Explicit Gregorian fields to a scaled Unix integer", Params: params, Return: &TypeDescriptor{Kind: "int"}, Const: true}})
	Declare(&Globalenv, &Declaration{Name: "unix_clock", Fn: UnixClock, Type: &TypeDescriptor{Kind: "func", Description: "Current Unix integer at the requested tick rate", HasSideEffects: true, Params: []*TypeDescriptor{{Kind: "number"}}, Return: &TypeDescriptor{Kind: "int"}}})
	Declare(&Globalenv, &Declaration{Name: "unix_parse", Fn: UnixParse, Type: &TypeDescriptor{Kind: "func", Description: "Parse an explicitly supplied calendar layout and tick rate", Params: []*TypeDescriptor{{Kind: "string"}, {Kind: "string"}, {Kind: "number"}, {Kind: "number"}, {Kind: "bool", Optional: true}}, Return: &TypeDescriptor{Kind: "int"}, Const: true}})
	Declare(&Globalenv, &Declaration{Name: "unix_format", Fn: UnixFormat, Type: &TypeDescriptor{Kind: "func", Description: "Format an explicitly scaled Unix integer with a supplied calendar layout", Params: []*TypeDescriptor{{Kind: "number"}, {Kind: "number"}, {Kind: "string"}, {Kind: "number"}}, Return: &TypeDescriptor{Kind: "string"}, Const: true}})
	Declare(&Globalenv, &Declaration{Name: "unix_zone_offset", Fn: UnixZoneOffset, Type: &TypeDescriptor{Kind: "func", Description: "Zone offset in seconds at an explicitly scaled Unix instant", Params: []*TypeDescriptor{{Kind: "number"}, {Kind: "number"}, {Kind: "string"}}, Return: &TypeDescriptor{Kind: "int"}, Const: true}})
}
