/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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
	"strings"
	"time"
)

var allowedDateFormats = []string{
	"2006-01-02 15:04:05.000000",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
	"06-01-02 15:04:05.000000",
	"06-01-02 15:04:05",
	"06-01-02 15:04",
	"06-01-02",
}

// mysqlZeroDateUnix is outside Go's practical SQL date range and fits exactly
// into Scmer's signed 45-bit date payload. It preserves MySQL's zero date
// without conflating it with the Unix epoch.
const mysqlZeroDateUnix int64 = -(1 << 44)

func isMySQLZeroDate(s string) bool {
	if s == "0000-00-00" || s == "0000-00-00 00:00:00" {
		return true
	}
	if !strings.HasPrefix(s, "0000-00-00 00:00:00.") {
		return false
	}
	fraction := strings.TrimPrefix(s, "0000-00-00 00:00:00.")
	return fraction != "" && strings.Trim(fraction, "0") == ""
}

// ParseDateString tries to parse a date/datetime string using the allowed formats.
// Returns the Unix timestamp and true on success, or 0 and false on failure.
func ParseDateString(s string) (int64, bool) {
	if isMySQLZeroDate(s) {
		return mysqlZeroDateUnix, true
	}
	for _, format := range allowedDateFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t.Unix(), true
		}
	}
	return 0, false
}

// toTime converts a Scmer value (tagDate, int, float, or string) to time.Time.
func toTime(v Scmer) (time.Time, bool) {
	if v.IsNil() {
		return time.Time{}, false
	}
	switch v.GetTag() {
	case tagDate:
		return time.Unix(v.Int(), 0).UTC(), true
	case tagInt:
		return time.Unix(v.Int(), 0).UTC(), true
	case tagFloat:
		return time.Unix(int64(v.Float()), 0).UTC(), true
	case tagString, tagSymbol:
		if ts, ok := ParseDateString(v.String()); ok {
			return time.Unix(ts, 0).UTC(), true
		}
		return time.Time{}, false
	default:
		return time.Unix(v.Int(), 0).UTC(), true
	}
}

func sqlTemporalOutput(value Scmer, sqlType string, timezone Scmer) Scmer {
	if value.IsNil() {
		return NewNil()
	}
	if value.GetTag() == tagDate && value.Int() == mysqlZeroDateUnix {
		switch strings.ToUpper(sqlType) {
		case "DATE":
			return NewString("0000-00-00")
		case "DATETIME", "TIMESTAMP":
			return NewString("0000-00-00 00:00:00")
		default:
			return value
		}
	}
	t, ok := toTime(value)
	if !ok {
		return value
	}
	loc, err := ResolveLocation(timezone.String())
	if err != nil {
		loc = time.UTC
	}
	switch strings.ToUpper(sqlType) {
	case "DATE":
		return NewString(t.UTC().Format("2006-01-02"))
	case "DATETIME", "TIMESTAMP":
		if value.GetTag() == tagDate {
			return NewString(DateToDisplay(value, loc))
		}
		return NewString(t.In(loc).Format("2006-01-02 15:04:05"))
	default:
		return value
	}
}

func init_date() {
	// string functions
	DeclareTitle("Date")

	Declare(&Globalenv, &Declaration{
		Name: "sql_temporal_output",

		Fn: func(a ...Scmer) Scmer {
			return sqlTemporalOutput(a[0], a[1].String(), a[2])
		},
		Type: &TypeDescriptor{Kind: "func", Description: "formats a temporal SQL result according to its compiler-tracked declared type",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "value", Description: "type-flexible temporal value"},
				{Kind: "string", Label: "sql_type", Description: "declared SQL temporal type"},
				{Kind: "string", Label: "timezone", Description: "explicit session timezone"},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["sql_temporal_output"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				argPinned0 := make([]Reg, 0, len(args)*3)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned0 = append(argPinned0, ai.Reg2)
						}
					} else if ai.Loc == LocRegTriple {
						for _, r := range [...]Reg{ai.Reg, ai.Reg2, ai.Reg3} {
							if !seenArgRegs[r] {
								ctx.ProtectReg(r)
								seenArgRegs[r] = true
								argPinned0 = append(argPinned0, r)
							}
						}
					}
				}
				defer func() {
					for _, r := range argPinned0 {
						ctx.UnprotectReg(r)
					}
				}()
				d1 := args[0]
				d1.ID = 0
				d2 := args[1]
				d2.ID = 0
				d4 := d2
				ctx.EnsureDesc(&d4)
				if d4.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					tag := d4.Imm.GetTag()
					switch tag {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d4)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d4)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d4)
					case tagNil:
						ctx.EmitMakeNil(tmpPair)
					default:
						ptrWord, auxWord := d4.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d4 = tmpPair
				} else if d4.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d4.Reg), Reg2: ctx.AllocRegExcept(d4.Reg)}
					switch d4.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d4)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d4)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d4)
					default:
						panic("jit: Scmer.String requires Scmer pair receiver")
					}
					ctx.FreeDesc(&d4)
					d4 = tmpPair
				} else if d4.Loc == LocMem {
					tmpScalar := JITValueDesc{Loc: LocReg, Type: d4.Type, Reg: ctx.AllocReg()}
					scratch := ctx.AllocRegExcept(tmpScalar.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d4.MemPtr))
					ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
					ctx.FreeReg(scratch)
					ctx.BindReg(tmpScalar.Reg, &tmpScalar)
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
					switch tmpScalar.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, tmpScalar)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, tmpScalar)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, tmpScalar)
					default:
						panic("jit: Scmer.String requires Scmer pair receiver")
					}
					ctx.FreeDesc(&tmpScalar)
					d4 = tmpPair
				}
				if d4.Loc != LocRegPair && d4.Loc != LocStackPair {
					panic("jit: Scmer.String receiver not materialized as pair")
				}
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d4}, 2)
				ctx.FreeDesc(&d2)
				d5 := args[2]
				d5.ID = 0
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				if d1.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					if d1.Imm.GetTag() == tagBool {
						ctx.EmitMakeBool(tmpPair, d1)
					} else if d1.Imm.GetTag() == tagInt {
						ctx.EmitMakeInt(tmpPair, d1)
					} else if d1.Imm.GetTag() == tagFloat {
						ctx.EmitMakeFloat(tmpPair, d1)
					} else if d1.Imm.GetTag() == tagNil {
						ctx.EmitMakeNil(tmpPair)
					} else {
						ptrWord, auxWord := d1.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d1 = tmpPair
				} else if d1.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
					switch d1.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d1)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d1)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d1)
					default:
						panic("jit: generic call arg scalar type unknown for 2-word value")
					}
					ctx.FreeDesc(&d1)
					d1 = tmpPair
				}
				if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (sqlTemporalOutput arg0)")
				}
				ctx.EnsureDesc(&d3)
				ctx.EnsureDesc(&d3)
				if d3.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d3.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.TrackImm(d3.Imm)
					ptrWord, _ := d3.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d3.Imm.String())))
					d3 = tmpPair
				} else if d3.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d3.Type, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
					switch d3.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d3)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d3)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d3)
					default:
						panic("jit: generic call arg scalar type unknown for 2-word value")
					}
					ctx.FreeDesc(&d3)
					d3 = tmpPair
				}
				if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (sqlTemporalOutput arg1)")
				}
				ctx.EnsureDesc(&d5)
				ctx.EnsureDesc(&d5)
				if d5.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d5.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					if d5.Imm.GetTag() == tagBool {
						ctx.EmitMakeBool(tmpPair, d5)
					} else if d5.Imm.GetTag() == tagInt {
						ctx.EmitMakeInt(tmpPair, d5)
					} else if d5.Imm.GetTag() == tagFloat {
						ctx.EmitMakeFloat(tmpPair, d5)
					} else if d5.Imm.GetTag() == tagNil {
						ctx.EmitMakeNil(tmpPair)
					} else {
						ptrWord, auxWord := d5.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d5 = tmpPair
				} else if d5.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d5.Type, Reg: ctx.AllocRegExcept(d5.Reg), Reg2: ctx.AllocRegExcept(d5.Reg)}
					switch d5.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d5)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d5)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d5)
					default:
						panic("jit: generic call arg scalar type unknown for 2-word value")
					}
					ctx.FreeDesc(&d5)
					d5 = tmpPair
				}
				if d5.Loc != LocRegPair && d5.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (sqlTemporalOutput arg2)")
				}
				ctx.SyncDesc(&d1)
				ctx.SyncDesc(&d3)
				ctx.SyncDesc(&d5)
				d6 := ctx.EmitGoCallScalar(GoFuncAddr(sqlTemporalOutput), []JITValueDesc{d1, d3, d5}, 2)
				ctx.BindReg(d6.Reg, &d6)
				ctx.BindReg(d6.Reg2, &d6)
				ctx.FreeDesc(&d1)
				ctx.FreeDesc(&d5)
				if d6.Loc == LocImm {
					if result.Loc == LocAny {
						return d6
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.EnsureDesc(&d6)
				if d6.Loc == LocRegPair {
					ctx.EmitMovPairToResult(&d6, &result)
					result.Type = d6.Type
				} else {
					switch d6.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d6)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d6)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d6)
						result.Type = tagFloat
					case tagNil:
						ctx.EmitMakeNil(result)
						result.Type = tagNil
					default:
						panic("jit: single-block scalar return with unknown type")
					}
				}
				return result
				return result
			},
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "now",

		Fn: func(a ...Scmer) Scmer {
			return NewDate(time.Now().Unix())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns the current date/time",
			Return: &TypeDescriptor{Kind: "date"},

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["now"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "nanotime",

		Fn: func(a ...Scmer) Scmer {
			return NewInt(time.Now().UnixNano())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a monotonic nanosecond timestamp for benchmarking (not wall-clock)",
			Return: &TypeDescriptor{Kind: "int"},

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["nanotime"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "current_date",

		Fn: func(a ...Scmer) Scmer {
			timezone := "UTC"
			if len(a) > 0 {
				timezone = a[0].String()
			}
			loc, err := ResolveLocation(timezone)
			if err != nil {
				loc = time.UTC
			}
			now := time.Now().In(loc)
			midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			return NewDate(midnight.Unix())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns the current date (midnight in session timezone)",
			Params: []*TypeDescriptor{{Kind: "string", Label: "timezone", Description: "explicit session timezone", Optional: true}},
			Return: &TypeDescriptor{Kind: "date"},

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["current_date"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "parse_date",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			if a[0].GetTag() == tagDate {
				return a[0]
			}
			if a[0].IsInt() || a[0].IsFloat() {
				return NewDate(a[0].Int())
			}
			if ts, ok := ParseDateString(a[0].String()); ok {
				return NewDate(ts)
			}
			return NewNil()
		},
		Type: &TypeDescriptor{Kind: "func", Description: "parses a date from a string",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "values to parse"}},
			Return: &TypeDescriptor{Kind: "date"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["parse_date"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "format_date",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			t, ok := toTime(a[0])
			if !ok {
				return NewNil()
			}
			timezone := "UTC"
			if len(a) > 2 {
				timezone = a[2].String()
			}
			loc, err := ResolveLocation(timezone)
			if err != nil {
				loc = time.UTC
			}
			t = t.In(loc)
			return NewString(formatDateMySQL(t, String(a[1])))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "formats a unix timestamp, date, or datetime string into a date string",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "timestamp", Description: "unix timestamp, date, or datetime string"}, &TypeDescriptor{Kind: "string", Label: "format", Description: "MySQL-style format string (e.g. %Y-%m-%d %H:%i:%s)"}, &TypeDescriptor{Kind: "string", Label: "timezone", Description: "explicit session timezone", Optional: true}},
			Return: &TypeDescriptor{Kind: "string"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["format_date"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})

	// EXTRACT(field FROM expr) - implemented as extract_date(expr, field)
	Declare(&Globalenv, &Declaration{
		Name: "extract_date",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			t, ok := toTime(a[0])
			if !ok {
				return NewNil()
			}
			timezone := "UTC"
			if len(a) > 2 {
				timezone = a[2].String()
			}
			loc, err := ResolveLocation(timezone)
			if err != nil {
				loc = time.UTC
			}
			t = t.In(loc)
			field := strings.ToUpper(a[1].String())
			switch field {
			case "YEAR":
				return NewInt(int64(t.Year()))
			case "MONTH":
				return NewInt(int64(t.Month()))
			case "DAY":
				return NewInt(int64(t.Day()))
			case "HOUR":
				return NewInt(int64(t.Hour()))
			case "MINUTE":
				return NewInt(int64(t.Minute()))
			case "SECOND":
				return NewInt(int64(t.Second()))
			case "QUARTER":
				return NewInt(int64((int(t.Month())-1)/3 + 1))
			case "WEEK":
				_, week := t.ISOWeek()
				return NewInt(int64(week))
			case "DAYOFWEEK":
				// MySQL: 1=Sunday, 2=Monday, ..., 7=Saturday
				return NewInt(int64(t.Weekday()) + 1)
			case "WEEKDAY":
				// MySQL WEEKDAY: 0=Monday, 1=Tuesday, ..., 6=Sunday
				return NewInt(int64((t.Weekday() + 6) % 7))
			default:
				panic("unknown EXTRACT field: " + field)
			}
		},
		Type: &TypeDescriptor{Kind: "func", Description: "extracts a date field (YEAR, MONTH, DAY, HOUR, MINUTE, SECOND, QUARTER, WEEK, DAYOFWEEK, WEEKDAY) from a date value",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "value", Description: "date value"}, &TypeDescriptor{Kind: "string", Label: "field", Description: "field name: YEAR, MONTH, DAY, HOUR, MINUTE, SECOND, QUARTER, WEEK, DAYOFWEEK, WEEKDAY"}, &TypeDescriptor{Kind: "string", Label: "timezone", Description: "explicit session timezone", Optional: true}},
			Return: &TypeDescriptor{Kind: "int"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["extract_date"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})

	// DATE_ADD(expr, interval_seconds)
	Declare(&Globalenv, &Declaration{
		Name: "date_add",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			t, ok := toTime(a[0])
			if !ok {
				return NewNil()
			}
			amount := int(a[1].Int())
			unit := strings.ToUpper(a[2].String())
			switch unit {
			case "SECOND":
				t = t.Add(time.Duration(amount) * time.Second)
			case "MINUTE":
				t = t.Add(time.Duration(amount) * time.Minute)
			case "HOUR":
				t = t.Add(time.Duration(amount) * time.Hour)
			case "DAY":
				t = t.AddDate(0, 0, amount)
			case "WEEK":
				t = t.AddDate(0, 0, amount*7)
			case "MONTH":
				t = t.AddDate(0, amount, 0)
			case "YEAR":
				t = t.AddDate(amount, 0, 0)
			default:
				panic("unknown DATE_ADD unit: " + unit)
			}
			return NewDate(t.Unix())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "adds an interval to a date value",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "value", Description: "date value"}, &TypeDescriptor{Kind: "int", Label: "amount", Description: "interval amount"}, &TypeDescriptor{Kind: "string", Label: "unit", Description: "interval unit: DAY, WEEK, MONTH, YEAR, HOUR, MINUTE, SECOND"}},
			Return: &TypeDescriptor{Kind: "date"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["date_add"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})

	// DATE_SUB(expr, amount, unit)
	Declare(&Globalenv, &Declaration{
		Name: "date_sub",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			t, ok := toTime(a[0])
			if !ok {
				return NewNil()
			}
			amount := int(a[1].Int())
			unit := strings.ToUpper(a[2].String())
			switch unit {
			case "SECOND":
				t = t.Add(-time.Duration(amount) * time.Second)
			case "MINUTE":
				t = t.Add(-time.Duration(amount) * time.Minute)
			case "HOUR":
				t = t.Add(-time.Duration(amount) * time.Hour)
			case "DAY":
				t = t.AddDate(0, 0, -amount)
			case "WEEK":
				t = t.AddDate(0, 0, -amount*7)
			case "MONTH":
				t = t.AddDate(0, -amount, 0)
			case "YEAR":
				t = t.AddDate(-amount, 0, 0)
			default:
				panic("unknown DATE_SUB unit: " + unit)
			}
			return NewDate(t.Unix())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "subtracts an interval from a date value",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "value", Description: "date value"}, &TypeDescriptor{Kind: "int", Label: "amount", Description: "interval amount"}, &TypeDescriptor{Kind: "string", Label: "unit", Description: "interval unit: DAY, WEEK, MONTH, YEAR, HOUR, MINUTE, SECOND"}},
			Return: &TypeDescriptor{Kind: "date"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["date_sub"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})

	// DATE(expr) - truncate to date only (midnight)
	Declare(&Globalenv, &Declaration{
		Name: "date_trunc_day",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			t, ok := toTime(a[0])
			if !ok {
				return NewNil()
			}
			midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			return NewDate(midnight.Unix())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "truncates a datetime to date (midnight UTC)",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "value", Description: "date/datetime value"}},
			Return: &TypeDescriptor{Kind: "date"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["date_trunc_day"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})

	// TIMESTAMPDIFF(unit, datetime1, datetime2) - returns datetime2 - datetime1 in the given unit
	Declare(&Globalenv, &Declaration{
		Name: "timestampdiff",

		Fn: func(a ...Scmer) Scmer {
			if a[1].IsNil() || a[2].IsNil() {
				return NewNil()
			}
			t1, ok1 := toTime(a[1])
			t2, ok2 := toTime(a[2])
			if !ok1 || !ok2 {
				return NewNil()
			}
			unit := strings.ToUpper(a[0].String())
			switch unit {
			case "MICROSECOND":
				return NewInt(t2.Sub(t1).Microseconds())
			case "SECOND":
				return NewInt(int64(t2.Sub(t1).Seconds()))
			case "MINUTE":
				return NewInt(int64(t2.Sub(t1).Minutes()))
			case "HOUR":
				return NewInt(int64(t2.Sub(t1).Hours()))
			case "DAY":
				d1 := time.Date(t1.Year(), t1.Month(), t1.Day(), 0, 0, 0, 0, time.UTC)
				d2 := time.Date(t2.Year(), t2.Month(), t2.Day(), 0, 0, 0, 0, time.UTC)
				return NewInt(int64(d2.Sub(d1).Hours() / 24))
			case "WEEK":
				d1 := time.Date(t1.Year(), t1.Month(), t1.Day(), 0, 0, 0, 0, time.UTC)
				d2 := time.Date(t2.Year(), t2.Month(), t2.Day(), 0, 0, 0, 0, time.UTC)
				return NewInt(int64(d2.Sub(d1).Hours() / 24 / 7))
			case "MONTH":
				years := int64(t2.Year() - t1.Year())
				months := int64(t2.Month() - t1.Month())
				total := years*12 + months
				// adjust if day of month hasn't been reached yet
				if t2.Day() < t1.Day() {
					total--
				}
				return NewInt(total)
			case "QUARTER":
				years := int64(t2.Year() - t1.Year())
				months := int64(t2.Month() - t1.Month())
				total := years*12 + months
				if t2.Day() < t1.Day() {
					total--
				}
				return NewInt(total / 3)
			case "YEAR":
				years := int64(t2.Year() - t1.Year())
				// adjust if month/day hasn't been reached yet
				if t2.Month() < t1.Month() || (t2.Month() == t1.Month() && t2.Day() < t1.Day()) {
					years--
				}
				return NewInt(years)
			default:
				panic("unknown TIMESTAMPDIFF unit: " + unit)
			}
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns the difference between two datetime values in the specified unit (datetime2 - datetime1)",
			Params: []*TypeDescriptor{
				{Kind: "string", Label: "unit", Description: "unit: MICROSECOND, SECOND, MINUTE, HOUR, DAY, WEEK, MONTH, QUARTER, YEAR"},
				{Kind: "any", Label: "datetime1", Description: "first datetime value"},
				{Kind: "any", Label: "datetime2", Description: "second datetime value"},
			},
			Return: &TypeDescriptor{Kind: "int"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["timestampdiff"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})

	// DATEDIFF(date1, date2) - returns number of days between two dates
	Declare(&Globalenv, &Declaration{
		Name: "datediff",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() {
				return NewNil()
			}
			t1, ok1 := toTime(a[0])
			t2, ok2 := toTime(a[1])
			if !ok1 || !ok2 {
				return NewNil()
			}
			d1 := time.Date(t1.Year(), t1.Month(), t1.Day(), 0, 0, 0, 0, time.UTC)
			d2 := time.Date(t2.Year(), t2.Month(), t2.Day(), 0, 0, 0, 0, time.UTC)
			days := int64(d1.Sub(d2).Hours() / 24)
			return NewInt(days)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns number of days between two dates (date1 - date2)",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", Label: "date1", Description: "first date"}, &TypeDescriptor{Kind: "any", Label: "date2", Description: "second date"}},
			Return: &TypeDescriptor{Kind: "int"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["datediff"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})

	// STR_TO_DATE(str, format) - parse string with MySQL format to date
	Declare(&Globalenv, &Declaration{
		Name: "str_to_date",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			// convert MySQL format to Go format
			mysqlFmt := a[1].String()
			goFmt := mysqlFormatToGo(mysqlFmt)
			if t, err := time.Parse(goFmt, a[0].String()); err == nil {
				return NewDate(t.Unix())
			}
			return NewNil()
		},
		Type: &TypeDescriptor{Kind: "func", Description: "parses a string with MySQL format specifiers to a date",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "value", Description: "date string"}, &TypeDescriptor{Kind: "string", Label: "format", Description: "MySQL format string (e.g. %Y-%m-%d)"}},
			Return: &TypeDescriptor{Kind: "date"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["str_to_date"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})
}

// mysqlFormatToGo converts a MySQL date format string to a Go time format string.
func mysqlFormatToGo(format string) string {
	var buf strings.Builder
	for i := 0; i < len(format); i++ {
		if format[i] == '%' && i+1 < len(format) {
			switch format[i+1] {
			case 'Y':
				buf.WriteString("2006")
			case 'y':
				buf.WriteString("06")
			case 'm':
				buf.WriteString("01")
			case 'd':
				buf.WriteString("02")
			case 'H':
				buf.WriteString("15")
			case 'i':
				buf.WriteString("04")
			case 's':
				buf.WriteString("05")
			case '%':
				buf.WriteByte('%')
			default:
				buf.WriteByte(format[i+1])
			}
			i++
		} else {
			buf.WriteByte(format[i])
		}
	}
	return buf.String()
}
