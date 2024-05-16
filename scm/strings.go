/*
Copyright (C) 2023  Carl-Philip Hänsch

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

import "fmt"
import "html"
import "bytes"
import "strings"
import "net/url"

type LazyString struct {
	Hash string
	GetValue func() string
}

/* SQL LIKE operator implementation on strings */
func StrLike(str, pattern string) bool {
	for {
		// boundary check
		if len(pattern) == 0 {
			if len(str) == 0 {
				// we finished matching
				return true
			} else {
				// pattern is consumed but no string left: no match
				return false
			}
		}
		// now str[0] and pattern[0] are assured to exist
		if pattern[0] == '%' { // wildcard
			pattern = pattern[1:]
			if pattern == "" {
				return true // string ends with wildcard
			}
			// otherwise: match against all possible endings
			for i := len(str)-1; i >= 0; i-- { // run from right to left to be as greedy and performant as possible
				if str[i] == pattern[0] {
					// check if this caracter matches the rest
					if StrLike(str[i:], pattern) {
						return true // we found a match with this position as continuation
					}
				}
			}
			return false // no continuation found
		} else {
			if len(str) > 0 && (pattern[0] == '_' || pattern[0] == str[0]) {
				// match -> move one character forward
				pattern = pattern[1:]
				str = str[1:]
			} else {
				// mismatch -> we're out
				return false
			}
		}
	}
}

func init_strings() {
	// string functions
	DeclareTitle("Strings")

	Declare(&Globalenv, &Declaration{
		"string?", "tells if the value is a string",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value"},
		}, "bool",
		func(a ...Scmer) (result Scmer) {
			_, ok := a[0].(string)
			return ok
		},
	})
	Declare(&Globalenv, &Declaration{
		"concat", "concatenates stringable values and returns a string",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values to concat"},
		}, "string",
		func(a ...Scmer) Scmer {
			// concat strings
			var b bytes.Buffer
			for _, s := range a {
				b.WriteString(String(s))
			}
			return b.String()
		},
	})
	Declare(&Globalenv, &Declaration{
		"simplify", "turns a stringable input value in the easiest-most value (e.g. turn strings into numbers if they are numeric",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value to simplify"},
		}, "any",
		func(a ...Scmer) Scmer {
			// turn string to number or so
			return Simplify(String(a[0]))
		},
	})
	Declare(&Globalenv, &Declaration{
		"strlen", "returns the length of a string",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "input string"},
		}, "number",
		func(a ...Scmer) Scmer {
			// string
			return float64(len(String(a[0])))
		},
	})
	Declare(&Globalenv, &Declaration{
		"strlike", "matches the string against a wildcard pattern (SQL compliant)",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "input string"},
			DeclarationParameter{"pattern", "string", "pattern with % and _ in them"},
		}, "bool",
		func(a ...Scmer) Scmer {
			// string
			return StrLike(String(a[0]), String(a[1]))
		},
	})
	Declare(&Globalenv, &Declaration{
		"toLower", "turns a string into lower case",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "input string"},
		}, "string",
		func(a ...Scmer) Scmer {
			// string
			return strings.ToLower(String(a[0]))
		},
	})
	Declare(&Globalenv, &Declaration{
		"toUpper", "turns a string into upper case",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "input string"},
		}, "string",
		func(a ...Scmer) Scmer {
			// string
			return strings.ToUpper(String(a[0]))
		},
	})
	Declare(&Globalenv, &Declaration{
		"replace", "replaces all occurances in a string with another string",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"s", "string", "input string"},
			DeclarationParameter{"find", "string", "search string"},
			DeclarationParameter{"replace", "string", "replace string"},
		}, "string",
		func(a ...Scmer) Scmer {
			// string
			return strings.ReplaceAll(String(a[0]), String(a[1]), String(a[2]))
		},
	})
	Declare(&Globalenv, &Declaration{
		"split", "splits a string using a separator or space",
		1, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "input string"},
			DeclarationParameter{"separator", "string", "(optional) parameter, defaults to \" \""},
		}, "string",
		func(a ...Scmer) Scmer {
			// string, sep
			split := " "
			if len(a) > 1 {
				split = String(a[1])
			}
			ar := strings.Split(String(a[0]), split)
			result := make([]Scmer, len(ar))
			for i, v := range ar {
				result[i] = v
			}
			return result
		},
	})

	Declare(&Globalenv, &Declaration{
		"htmlentities", "escapes the string for use in HTML",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "input string"},
		}, "string",
		func(a ...Scmer) Scmer {
			// string
			return html.EscapeString(String(a[0]))
		},
	})
	Declare(&Globalenv, &Declaration{
		"urlencode", "encodes a string according to URI coding schema",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "string to encode"},
		}, "string",
		func (a ...Scmer) Scmer {
			return url.QueryEscape(String(a[0]))
		},
	})
	Declare(&Globalenv, &Declaration{
		"urldecode", "decodes a string according to URI coding schema",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "string", "string to decode"},
		}, "string",
		func (a ...Scmer) Scmer {
			if result, err := url.QueryUnescape(String(a[0])); err == nil {
				return result
			} else {
				panic("error while decoding URL: " + fmt.Sprint(err))
			}
		},
	})

}

