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

func init_date() {
	// string functions
	DeclareTitle("Date")
	allowed_formats := []string{
		"2006-01-02 15:04:05.000000",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"06-01-02 15:04:05.000000",
		"06-01-02 15:04:05",
		"06-01-02 15:04",
		"06-01-02",
	}

	Declare(&Globalenv, &Declaration{
		"now", "returns the unix timestamp",
		0, 0,
		[]DeclarationParameter{}, "int",
		func(a ...Scmer) Scmer {
			return NewInt(time.Now().Unix())
		},
		false,
	})
	Declare(&Globalenv, &Declaration{
		"parse_date", "parses unix date from a string",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "values to parse"},
		}, "int",
		func(a ...Scmer) Scmer {
			for _, format := range allowed_formats { // try through all formats
				if t, err := time.Parse(format, String(a[0])); err == nil {
					return NewInt(t.Unix())
				}
			}
			return NewNil()
		},
		true,
	})
	Declare(&Globalenv, &Declaration{
		"format_date", "formats a unix timestamp or datetime string into a date string",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"timestamp", "any", "unix timestamp or datetime string"},
			DeclarationParameter{"format", "string", "MySQL-style format string (e.g. %Y-%m-%d %H:%i:%s)"},
		}, "string",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			var t time.Time
			if a[0].IsString() {
				// parse datetime string
				parsed := false
				for _, fmt := range allowed_formats {
					if pt, err := time.Parse(fmt, String(a[0])); err == nil {
						t = pt
						parsed = true
						break
					}
				}
				if !parsed {
					return NewNil()
				}
			} else {
				ts := ToInt(a[0])
				t = time.Unix(int64(ts), 0).UTC()
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
}
