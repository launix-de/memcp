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
		Name: "now",
		Desc: "returns the current date/time",
		Fn: func(a ...Scmer) Scmer {
				return NewDate(time.Now().Unix())
			},
		Type: &TypeDescriptor{
			Return: &TypeDescriptor{Kind: "date"},
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "current_date",
		Desc: "returns the current date (midnight in session timezone)",
		Fn: func(a ...Scmer) Scmer {
				loc := GetCurrentSessionLocation()
				now := time.Now().In(loc)
				midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
				return NewDate(midnight.Unix())
			},
		Type: &TypeDescriptor{
			Return: &TypeDescriptor{Kind: "date"},
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "parse_date",
		Desc: "parses a date from a string",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "values to parse"}},
			Return: &TypeDescriptor{Kind: "date"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "format_date",
		Desc: "formats a unix timestamp, date, or datetime string into a date string",
		Fn: func(a ...Scmer) Scmer {
				if a[0].IsNil() {
					return NewNil()
				}
				t, ok := toTime(a[0])
				if !ok {
					return NewNil()
				}
				// apply session timezone for display
				t = t.In(GetCurrentSessionLocation())
				return NewString(formatDateMySQL(t, String(a[1])))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "timestamp", ParamDesc: "unix timestamp, date, or datetime string"}, &TypeDescriptor{Kind: "string", ParamName: "format", ParamDesc: "MySQL-style format string (e.g. %Y-%m-%d %H:%i:%s)"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})

	// EXTRACT(field FROM expr) - implemented as extract_date(expr, field)
		Declare(&Globalenv, &Declaration{
		Name: "extract_date",
		Desc: "extracts a date field (YEAR, MONTH, DAY, HOUR, MINUTE, SECOND, QUARTER, WEEK, DAYOFWEEK, WEEKDAY) from a date value",
		Fn: func(a ...Scmer) Scmer {
				if a[0].IsNil() {
					return NewNil()
				}
				t, ok := toTime(a[0])
				if !ok {
					return NewNil()
				}
				t = t.In(GetCurrentSessionLocation())
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "value", ParamDesc: "date value"}, &TypeDescriptor{Kind: "string", ParamName: "field", ParamDesc: "field name: YEAR, MONTH, DAY, HOUR, MINUTE, SECOND, QUARTER, WEEK, DAYOFWEEK, WEEKDAY"}},
			Return: &TypeDescriptor{Kind: "int"},
			Const: true,
		},
	})

	// DATE_ADD(expr, interval_seconds)
		Declare(&Globalenv, &Declaration{
		Name: "date_add",
		Desc: "adds an interval to a date value",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "value", ParamDesc: "date value"}, &TypeDescriptor{Kind: "int", ParamName: "amount", ParamDesc: "interval amount"}, &TypeDescriptor{Kind: "string", ParamName: "unit", ParamDesc: "interval unit: DAY, WEEK, MONTH, YEAR, HOUR, MINUTE, SECOND"}},
			Return: &TypeDescriptor{Kind: "date"},
			Const: true,
		},
	})

	// DATE_SUB(expr, amount, unit)
		Declare(&Globalenv, &Declaration{
		Name: "date_sub",
		Desc: "subtracts an interval from a date value",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "value", ParamDesc: "date value"}, &TypeDescriptor{Kind: "int", ParamName: "amount", ParamDesc: "interval amount"}, &TypeDescriptor{Kind: "string", ParamName: "unit", ParamDesc: "interval unit: DAY, WEEK, MONTH, YEAR, HOUR, MINUTE, SECOND"}},
			Return: &TypeDescriptor{Kind: "date"},
			Const: true,
		},
	})

	// DATE(expr) - truncate to date only (midnight)
		Declare(&Globalenv, &Declaration{
		Name: "date_trunc_day",
		Desc: "truncates a datetime to date (midnight UTC)",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "value", ParamDesc: "date/datetime value"}},
			Return: &TypeDescriptor{Kind: "date"},
			Const: true,
		},
	})

	// DATEDIFF(date1, date2) - returns number of days between two dates
		Declare(&Globalenv, &Declaration{
		Name: "datediff",
		Desc: "returns number of days between two dates (date1 - date2)",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "date1", ParamDesc: "first date"}, &TypeDescriptor{Kind: "any", ParamName: "date2", ParamDesc: "second date"}},
			Return: &TypeDescriptor{Kind: "int"},
			Const: true,
		},
	})

	// STR_TO_DATE(str, format) - parse string with MySQL format to date
		Declare(&Globalenv, &Declaration{
		Name: "str_to_date",
		Desc: "parses a string with MySQL format specifiers to a date",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "date string"}, &TypeDescriptor{Kind: "string", ParamName: "format", ParamDesc: "MySQL format string (e.g. %Y-%m-%d)"}},
			Return: &TypeDescriptor{Kind: "date"},
			Const: true,
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
