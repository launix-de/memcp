/*
Copyright (C) 2024  Carl-Philip Hänsch

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
	"fmt"
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

// ParseDateString tries to parse a date/datetime string using the allowed formats.
// Returns the Unix timestamp and true on success, or 0 and false on failure.
func ParseDateString(s string) (int64, bool) {
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

func init_date() {
	// string functions
	DeclareTitle("Date")

	Declare(&Globalenv, &Declaration{
		"now", "returns the current date/time",
		0, 0,
		[]DeclarationParameter{}, "date",
		func(a ...Scmer) Scmer {
			return NewDate(time.Now().Unix())
		},
		false,
	})
	Declare(&Globalenv, &Declaration{
		"current_date", "returns the current date (midnight UTC)",
		0, 0,
		[]DeclarationParameter{}, "date",
		func(a ...Scmer) Scmer {
			now := time.Now().UTC()
			midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			return NewDate(midnight.Unix())
		},
		false,
	})
	Declare(&Globalenv, &Declaration{
		"parse_date", "parses a date from a string",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "values to parse"},
		}, "date",
		func(a ...Scmer) Scmer {
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
		true,
	})
	Declare(&Globalenv, &Declaration{
		"format_date", "formats a unix timestamp, date, or datetime string into a date string",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"timestamp", "any", "unix timestamp, date, or datetime string"},
			DeclarationParameter{"format", "string", "MySQL-style format string (e.g. %Y-%m-%d %H:%i:%s)"},
		}, "string",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			t, ok := toTime(a[0])
			if !ok {
				return NewNil()
			}
			format := String(a[1])
			// replace MySQL format specifiers manually to avoid Go magic number collisions
			var buf strings.Builder
			for i := 0; i < len(format); i++ {
				if format[i] == '%' && i+1 < len(format) {
					switch format[i+1] {
					case 'Y':
						buf.WriteString(fmt.Sprintf("%04d", t.Year()))
					case 'm':
						buf.WriteString(fmt.Sprintf("%02d", t.Month()))
					case 'd':
						buf.WriteString(fmt.Sprintf("%02d", t.Day()))
					case 'H':
						buf.WriteString(fmt.Sprintf("%02d", t.Hour()))
					case 'i':
						buf.WriteString(fmt.Sprintf("%02d", t.Minute()))
					case 's':
						buf.WriteString(fmt.Sprintf("%02d", t.Second()))
					case 'T':
						buf.WriteString(fmt.Sprintf("%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second()))
					case '%':
						buf.WriteByte('%')
					default:
						buf.WriteByte('%')
						buf.WriteByte(format[i+1])
					}
					i++ // skip format char
				} else {
					buf.WriteByte(format[i])
				}
			}
			return NewString(buf.String())
		},
		true,
	})

	// EXTRACT(field FROM expr) - implemented as extract_date(expr, field)
	Declare(&Globalenv, &Declaration{
		"extract_date", "extracts a date field (YEAR, MONTH, DAY, HOUR, MINUTE, SECOND) from a date value",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "date value"},
			DeclarationParameter{"field", "string", "field name: YEAR, MONTH, DAY, HOUR, MINUTE, SECOND"},
		}, "int",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			t, ok := toTime(a[0])
			if !ok {
				return NewNil()
			}
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
			default:
				panic("unknown EXTRACT field: " + field)
			}
		},
		true,
	})

	// DATE_ADD(expr, interval_seconds)
	Declare(&Globalenv, &Declaration{
		"date_add", "adds an interval to a date value",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "date value"},
			DeclarationParameter{"amount", "int", "interval amount"},
			DeclarationParameter{"unit", "string", "interval unit: DAY, WEEK, MONTH, YEAR, HOUR, MINUTE, SECOND"},
		}, "date",
		func(a ...Scmer) Scmer {
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
		true,
	})

	// DATE_SUB(expr, amount, unit)
	Declare(&Globalenv, &Declaration{
		"date_sub", "subtracts an interval from a date value",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "date value"},
			DeclarationParameter{"amount", "int", "interval amount"},
			DeclarationParameter{"unit", "string", "interval unit: DAY, WEEK, MONTH, YEAR, HOUR, MINUTE, SECOND"},
		}, "date",
		func(a ...Scmer) Scmer {
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
		true,
	})

	// DATE(expr) - truncate to date only (midnight)
	Declare(&Globalenv, &Declaration{
		"date_trunc_day", "truncates a datetime to date (midnight UTC)",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "date/datetime value"},
		}, "date",
		func(a ...Scmer) Scmer {
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
		true,
	})

	// STR_TO_DATE(str, format) - parse string with MySQL format to date
	Declare(&Globalenv, &Declaration{
		"str_to_date", "parses a string with MySQL format specifiers to a date",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "date string"},
			DeclarationParameter{"format", "string", "MySQL format string (e.g. %Y-%m-%d)"},
		}, "date",
		func(a ...Scmer) Scmer {
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
		true,
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
