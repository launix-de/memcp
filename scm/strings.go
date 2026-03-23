/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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

import "io"
import "fmt"
import "html"
import "regexp"
import "strings"
import "net/url"
import "hash/fnv"
import "crypto/sha1"
import "crypto/sha256"
import "encoding/json"
import "encoding/base64"
import "encoding/hex"
import crand "crypto/rand"
import "golang.org/x/text/collate"
import "golang.org/x/text/language"
import "sync"
import "reflect"

// Collation metadata registry for stable serialization of comparator closures.
// Keyed by function pointer.
var collateRegistry sync.Map // map[uintptr]struct{Collation string; Reverse bool}

// (no additional globals needed)

// LookupCollate returns (collation, reverse, ok) for a previously built collate closure.
func LookupCollate(fn func(...Scmer) Scmer) (string, bool, bool) {
	if fn == nil {
		return "", false, false
	}
	if v, ok := collateRegistry.Load(reflect.ValueOf(fn).Pointer()); ok {
		m := v.(struct {
			Collation string
			Reverse   bool
		})
		return m.Collation, m.Reverse, true
	}
	return "", false, false
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
			for i := len(str) - 1; i >= 0; i-- { // run from right to left to be as greedy and performant as possible
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

func TransformFromJSON(a_ any) Scmer {
	switch a := a_.(type) {
	case map[string]any:
		// decode binary strings encoded by MarshalJSON
		if b64, ok := a["bytes"]; ok && len(a) == 1 {
			if s, ok := b64.(string); ok {
				if raw, err := base64.StdEncoding.DecodeString(s); err == nil {
					return NewString(string(raw))
				}
			}
		}
		result := make([]Scmer, 0, len(a)*2)
		for k, v := range a {
			result = append(result, NewString(k), TransformFromJSON(v))
		}
		return NewSlice(result)
	case []any:
		result := make([]Scmer, len(a))
		for i, v := range a {
			result[i] = TransformFromJSON(v)
		}
		return NewSlice(result)
	default:
		return FromAny(a_)
	}
}

func init_strings() {
	// string functions
	DeclareTitle("Strings")

		Declare(&Globalenv, &Declaration{
		Name: "string?",
		Desc: "tells if the value is a string",
		Fn: func(a ...Scmer) Scmer {
				_, ok := a[0].Any().(string)
				return NewBool(ok)
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "value", ParamDesc: "value"}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "concat",
		Desc: "concatenates stringable values and returns a string",
		Fn: func(a ...Scmer) Scmer {
				var sb strings.Builder
				for _, s := range a {
					if stream, ok := s.Any().(io.Reader); ok {
						_, _ = io.Copy(&sb, stream)
					} else {
						sb.WriteString(String(s))
					}
				}
				return NewString(sb.String())
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "value", ParamDesc: "first value to concat"}, &TypeDescriptor{Kind: "any", ParamName: "more...", ParamDesc: "additional values to concat", Variadic: true}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "substr",
		Desc: "returns a substring (0-based index)",
		Fn: func(a ...Scmer) Scmer {
				s := String(a[0])
				i := ToInt(a[1])
				if len(a) > 2 {
					return NewString(s[i : i+ToInt(a[2])])
				}
				return NewString(s[i:])
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "string to cut"}, &TypeDescriptor{Kind: "number", ParamName: "start", ParamDesc: "first character index (0-based)"}, &TypeDescriptor{Kind: "number", ParamName: "len", ParamDesc: "optional length", Optional: true}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "sql_substr",
		Desc: "SQL SUBSTR/SUBSTRING with 1-based index and bounds checking",
		Fn: func(a ...Scmer) Scmer {
				if a[0].IsNil() {
					return NewNil()
				}
				s := String(a[0])
				slen := len(s)
				start := ToInt(a[1]) - 1 // convert 1-based to 0-based
				if start < 0 {
					start = 0
				}
				if start >= slen {
					return NewString("")
				}
				if len(a) > 2 {
					n := ToInt(a[2])
					if start+n > slen {
						n = slen - start
					}
					if n < 0 {
						return NewString("")
					}
					return NewString(s[start : start+n])
				}
				return NewString(s[start:])
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "string to cut"}, &TypeDescriptor{Kind: "number", ParamName: "start", ParamDesc: "first character position (1-based)"}, &TypeDescriptor{Kind: "number", ParamName: "len", ParamDesc: "optional length", Optional: true}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "simplify",
		Desc: "turns a stringable input value in the easiest-most value (e.g. turn strings into numbers if they are numeric",
		Fn: func(a ...Scmer) Scmer {
				// turn string to number or so
				return Simplify(String(a[0]))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "value", ParamDesc: "value to simplify"}},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "strlen",
		Desc: "returns the length of a string",
		Fn: func(a ...Scmer) Scmer {
				return NewInt(int64(len(String(a[0]))))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "input string"}},
			Return: &TypeDescriptor{Kind: "int"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "strlike",
		Desc: "matches the string against a wildcard pattern (SQL compliant)",
		Fn: func(a ...Scmer) Scmer {
				value := String(a[0])
				pattern := String(a[1])
				collation := "utf8mb4_general_ci"
				if len(a) > 2 {
					collation = strings.ToLower(String(a[2]))
				}
				if strings.Contains(collation, "_ci") {
					value = strings.ToLower(value)
					pattern = strings.ToLower(pattern)
				}
				return NewBool(StrLike(value, pattern))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "input string"}, &TypeDescriptor{Kind: "string", ParamName: "pattern", ParamDesc: "pattern with % and _ in them"}, &TypeDescriptor{Kind: "string", ParamName: "collation", ParamDesc: "collation in which to compare them", Optional: true}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "strlike_cs",
		Desc: "matches the string against a wildcard pattern (case-sensitive)",
		Fn: func(a ...Scmer) Scmer {
				return NewBool(StrLike(String(a[0]), String(a[1])))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "input string"}, &TypeDescriptor{Kind: "string", ParamName: "pattern", ParamDesc: "pattern with % and _ in them"}, &TypeDescriptor{Kind: "string", ParamName: "collation", ParamDesc: "ignored (present for parser compatibility)", Optional: true}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "toLower",
		Desc: "turns a string into lower case",
		Fn: func(a ...Scmer) Scmer {
				return NewString(strings.ToLower(String(a[0])))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "input string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "toUpper",
		Desc: "turns a string into upper case",
		Fn: func(a ...Scmer) Scmer {
				return NewString(strings.ToUpper(String(a[0])))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "input string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "replace",
		Desc: "replaces all occurances in a string with another string",
		Fn: func(a ...Scmer) Scmer {
				return NewString(strings.ReplaceAll(String(a[0]), String(a[1]), String(a[2])))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "s", ParamDesc: "input string"}, &TypeDescriptor{Kind: "string", ParamName: "find", ParamDesc: "search string"}, &TypeDescriptor{Kind: "string", ParamName: "replace", ParamDesc: "replace string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "strtrim",
		Desc: "trims whitespace from both ends of a string",
		Fn: func(a ...Scmer) Scmer {
				return NewString(strings.TrimSpace(String(a[0])))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "input string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "strltrim",
		Desc: "trims whitespace from the left of a string",
		Fn: func(a ...Scmer) Scmer {
				return NewString(strings.TrimLeft(String(a[0]), " \t\n\r"))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "input string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "strrtrim",
		Desc: "trims whitespace from the right of a string",
		Fn: func(a ...Scmer) Scmer {
				return NewString(strings.TrimRight(String(a[0]), " \t\n\r"))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "input string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
	// SQL-level NULL-safe wrappers for TRIM/LTRIM/RTRIM
		Declare(&Globalenv, &Declaration{
		Name: "sql_trim",
		Desc: "SQL TRIM(): NULL-safe trim of whitespace from both ends",
		Fn: func(a ...Scmer) Scmer {
				if a[0].IsNil() {
					return NewNil()
				}
				return NewString(strings.TrimSpace(String(a[0])))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "input string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "sql_ltrim",
		Desc: "SQL LTRIM(): NULL-safe trim of whitespace from left",
		Fn: func(a ...Scmer) Scmer {
				if a[0].IsNil() {
					return NewNil()
				}
				return NewString(strings.TrimLeft(String(a[0]), " \t\n\r"))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "input string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "sql_rtrim",
		Desc: "SQL RTRIM(): NULL-safe trim of whitespace from right",
		Fn: func(a ...Scmer) Scmer {
				if a[0].IsNil() {
					return NewNil()
				}
				return NewString(strings.TrimRight(String(a[0]), " \t\n\r"))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "input string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "split",
		Desc: "splits a string using a separator or space",
		Fn: func(a ...Scmer) Scmer {
				split := " "
				if len(a) > 1 {
					split = String(a[1])
				}
				ar := strings.Split(String(a[0]), split)
				result := make([]Scmer, len(ar))
				for i, v := range ar {
					result[i] = NewString(v)
				}
				return NewSlice(result)
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "input string"}, &TypeDescriptor{Kind: "string", ParamName: "separator", ParamDesc: "(optional) parameter, defaults to \" \"", Optional: true}},
			Return: &TypeDescriptor{Kind: "list"},
			Const: true,
		},
	})

		Declare(&Globalenv, &Declaration{
		Name: "string_repeat",
		Desc: "repeats a string n times",
		Fn: func(a ...Scmer) Scmer {
				if a[0].IsNil() {
					return NewNil()
				}
				n := ToInt(a[1])
				if n <= 0 {
					return NewString("")
				}
				return NewString(strings.Repeat(String(a[0]), int(n)))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "string to repeat"}, &TypeDescriptor{Kind: "number", ParamName: "count", ParamDesc: "number of repetitions"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})

	/* comparison */
	collation_re := regexp.MustCompile("^([^_]+_)?(.+?)$") // caracterset_language_case
		Declare(&Globalenv, &Declaration{
		Name: "collate",
		Desc: "returns the `<` operator for a given collation. MemCP allows natural sorting of numeric literals.",
		Fn: func(a ...Scmer) Scmer {
				collation := String(a[0])
				ci := false
				if strings.HasSuffix(collation, "_ci") {
					ci = true
					collation = collation[:len(collation)-3]
				} else if strings.HasSuffix(collation, "_cs") {
					collation = collation[:len(collation)-3]
				}
				if m := collation_re.FindStringSubmatch(collation); m != nil {
					if m[2] == "bin" { // binary
						// Return closures that compare raw UTF-8 byte order; register for serialization
						if len(a) > 1 && ToBool(a[1]) {
							f := func(a ...Scmer) Scmer { return GreaterScm(a...) }
							collateRegistry.Store(reflect.ValueOf(f).Pointer(), struct {
								Collation string
								Reverse   bool
							}{Collation: String(a[0]), Reverse: true})
							return NewFunc(f)
						}
						f := func(a ...Scmer) Scmer { return LessScm(a...) }
						collateRegistry.Store(reflect.ValueOf(f).Pointer(), struct {
							Collation string
							Reverse   bool
						}{Collation: String(a[0]), Reverse: false})
						return NewFunc(f)
					}
					base := m[2]
					// Special-case MySQL-style "general" to simple case-insensitive first-letter ordering
					if strings.Contains(base, "general") {
						reverse := len(a) > 1 && ToBool(a[1])
						// general_ci heuristic:
						// - ASCII letters sort before non-ASCII always (both ASC and DESC).
						// - Treat leading "aa" as non-ASCII class to place after ASCII group in ASC and after ASCII even in DESC.
						// - Within ASCII, compare by lowercase first letter; tie-break by case-insensitive string compare.
						classify := func(s string) (isASCII bool, key byte, folded string) {
							if s == "" {
								return true, 0, s
							}
							sl := strings.ToLower(s)
							// map leading "aa" to non-ASCII class
							if len(sl) >= 2 && sl[0] == 'a' && sl[1] == 'a' {
								return false, 0, sl
							}
							b := sl[0]
							// check ASCII letter
							if b >= 'a' && b <= 'z' && (s[0] < 128) {
								return true, b, sl
							}
							return false, 0, sl
						}
						if reverse {
							f := func(a ...Scmer) Scmer {
								as := String(a[0])
								bs := String(a[1])
								aAsc, ak, af := classify(as)
								bAsc, bk, bf := classify(bs)
								var res bool
								if aAsc != bAsc {
									// ASCII ranks above non-ASCII for DESC too
									res = aAsc && !bAsc
								} else if aAsc { // both ASCII letters: reverse letter order
									if ak != bk {
										res = ak > bk
									} else {
										res = af > bf
									}
								} else {
									// both non-ASCII: keep stable fallback
									res = as > bs
								}
								return NewBool(res)
							}
							collateRegistry.Store(reflect.ValueOf(f).Pointer(), struct {
								Collation string
								Reverse   bool
							}{Collation: String(a[0]), Reverse: true})
							return NewFunc(f)
						}
						f := func(a ...Scmer) Scmer {
							as := String(a[0])
							bs := String(a[1])
							aAsc, ak, af := classify(as)
							bAsc, bk, bf := classify(bs)
							var res bool
							if aAsc != bAsc {
								// ASCII first for ASC
								res = aAsc && !bAsc
							} else if aAsc { // both ASCII letters
								if ak != bk {
									res = ak < bk
								} else {
									res = af < bf
								}
							} else {
								// both non-ASCII: leave at end
								res = as < bs
							}
							return NewBool(res)
						}
						collateRegistry.Store(reflect.ValueOf(f).Pointer(), struct {
							Collation string
							Reverse   bool
						}{Collation: String(a[0]), Reverse: false})
						return NewFunc(f)
					}
					tag, err := language.Parse(base) // treat as BCP 47
					if err != nil {
						// language not detected, try one of the aliases
						switch m[2] {
						case "danish":
							tag = language.Danish
						case "german1":
							tag = language.German
						case "german2":
							tag = language.German
						case "spanish":
							tag = language.Spanish
						case "swedish":
							tag = language.Swedish
						default:
							tag = language.Danish // default to danish for general-like collations (aa -> å semantics)
						}
					}
					var c *collate.Collator
					// the following options are available:
					// IgnoreCase -> when string ends with _ci
					// IgnoreDiacritics -> o == ö
					// IgnoreWidth: half width == width
					// Numeric -> sort numbers correctly
					if ci {
						c = collate.New(tag, collate.Numeric, collate.IgnoreCase)
					} else {
						c = collate.New(tag, collate.Numeric)
					}
	
					// return a LESS function specialized to that language and register for serialization
					reverse := len(a) > 1 && ToBool(a[1])
					if reverse {
						f := func(a ...Scmer) Scmer {
							var res bool
							// numeric fallback when both operands are numbers
							if (a[0].IsInt() || a[0].IsFloat()) && (a[1].IsInt() || a[1].IsFloat()) {
								res = ToFloat(a[0]) > ToFloat(a[1])
							}
							if !res {
								res = c.CompareString(String(a[0]), String(a[1])) == 1
							}
							return NewBool(res)
						}
						collateRegistry.Store(reflect.ValueOf(f).Pointer(), struct {
							Collation string
							Reverse   bool
						}{Collation: String(a[0]), Reverse: true})
						return NewFunc(f)
					}
					f := func(a ...Scmer) Scmer {
						// numeric fallback when both operands are numbers
						if (a[0].IsInt() || a[0].IsFloat()) && (a[1].IsInt() || a[1].IsFloat()) {
							return NewBool(ToFloat(a[0]) < ToFloat(a[1]))
						}
						return NewBool(c.CompareString(String(a[0]), String(a[1])) == -1)
					}
					collateRegistry.Store(reflect.ValueOf(f).Pointer(), struct {
						Collation string
						Reverse   bool
					}{Collation: String(a[0]), Reverse: false})
					return NewFunc(f)
				} else {
					if len(a) > 1 && ToBool(a[1]) {
						return NewFunc(GreaterScm)
					}
					return NewFunc(LessScm)
				}
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "collation", ParamDesc: "collation string of the form LANG or LANG_cs or LANG_ci where LANG is a BCP 47 code, for compatibility to MySQL, a CHARSET_ prefix is allowed and ignored as well as the aliases bin, danish, general, german1, german2, spanish and swedish are allowed for language codes"}, &TypeDescriptor{Kind: "bool", ParamName: "reverse", ParamDesc: "whether to reverse the order like in ORDER BY DESC", Optional: true}},
			Return: &TypeDescriptor{Kind: "func"},
			Const: true,
		},
	})

	/* escaping functions similar to PHP */
		Declare(&Globalenv, &Declaration{
		Name: "htmlentities",
		Desc: "escapes the string for use in HTML",
		Fn: func(a ...Scmer) Scmer {
				return NewString(html.EscapeString(String(a[0])))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "input string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "urlencode",
		Desc: "encodes a string according to URI coding schema",
		Fn: func(a ...Scmer) Scmer {
				return NewString(url.QueryEscape(String(a[0])))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "string to encode"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "urldecode",
		Desc: "decodes a string according to URI coding schema",
		Fn: func(a ...Scmer) Scmer {
				result, err := url.QueryUnescape(String(a[0]))
				if err != nil {
					panic("error while decoding URL: " + fmt.Sprint(err))
				}
				return NewString(result)
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "string to decode"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "json_encode",
		Desc: "encodes a value in JSON, treats lists as lists",
		Fn: func(a ...Scmer) Scmer {
				b, err := json.Marshal(a[0])
				if err != nil {
					panic(err)
				}
				return NewString(string(b))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "value", ParamDesc: "value to encode"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "json_encode_assoc",
		Desc: "encodes a value in JSON, treats lists as associative arrays",
		Fn: func(a ...Scmer) Scmer {
				// Build a Go structure where assoc lists (even-length lists or FastDict)
				// are represented as map[string]any, and leaf values remain Scmer so
				// Scmer.MarshalJSON applies for nested values.
				var transform func(Scmer) any
				transform = func(val Scmer) any {
					if val.IsSlice() {
						v := val.Slice()
						result := make(map[string]any)
						for i := 0; i < len(v)-1; i += 2 {
							result[String(v[i])] = transform(v[i+1])
						}
						return result
					}
					if val.IsFastDict() {
						fd := val.FastDict()
						result := make(map[string]any)
						if fd != nil {
							for i := 0; i < len(fd.Pairs)-1; i += 2 {
								result[String(fd.Pairs[i])] = transform(fd.Pairs[i+1])
							}
						}
						return result
					}
					// Keep as Scmer so its MarshalJSON semantics apply
					return val
				}
				b, err := json.Marshal(transform(a[0]))
				if err != nil {
					panic(err)
				}
				return NewString(string(b))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "any", ParamName: "value", ParamDesc: "value to encode"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "json_decode",
		Desc: "parses JSON into a map",
		Fn: func(a ...Scmer) Scmer {
				var result any
				err := json.Unmarshal([]byte(String(a[0])), &result)
				if err != nil {
					panic(err)
				}
				return TransformFromJSON(result)
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "string to decode"}},
			Const: true,
		},
	})

		Declare(&Globalenv, &Declaration{
		Name: "base64_encode",
		Desc: "encodes a string as Base64 (standard encoding)",
		Fn: func(a ...Scmer) Scmer {
				return NewString(base64.StdEncoding.EncodeToString([]byte(String(a[0]))))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "binary string to encode"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "base64_decode",
		Desc: "decodes a Base64 string (standard encoding)",
		Fn: func(a ...Scmer) Scmer {
				decoded, err := base64.StdEncoding.DecodeString(String(a[0]))
				if err != nil {
					panic("error while decoding base64: " + fmt.Sprint(err))
				}
				return NewString(string(decoded))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "base64-encoded string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
	sql_escapings := regexp.MustCompile("\\\\[\\\\'\"nr0]")
		Declare(&Globalenv, &Declaration{
		Name: "sql_unescape",
		Desc: "unescapes the inner part of a sql string",
		Fn: func(a ...Scmer) Scmer {
				input := String(a[0])
				out := sql_escapings.ReplaceAllStringFunc(input, func(m string) string {
					switch m {
					case "\\\\":
						return "\\"
					case "\\'":
						return "'"
					case "\\\"":
						return "\""
					case "\\n":
						return "\n"
					case "\\r":
						return "\r"
					case "\\0":
						return string([]byte{0})
					}
					return m
				})
				return NewString(out)
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "string to decode"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "bin2hex",
		Desc: "turns binary data into hex with lowercase letters",
		Fn: func(a ...Scmer) Scmer {
				input := String(a[0])
				result := make([]byte, 2*len(input))
				hexmap := "0123456789abcdef"
				for i := 0; i < len(input); i++ {
					result[2*i] = hexmap[input[i]/16]
					result[2*i+1] = hexmap[input[i]%16]
				}
				return NewString(string(result))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "string to decode"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "hex2bin",
		Desc: "decodes a hex string into binary data",
		Fn: func(a ...Scmer) Scmer {
				decoded, err := hex.DecodeString(String(a[0]))
				if err != nil {
					panic("error while decoding hex: " + fmt.Sprint(err))
				}
				return NewString(string(decoded))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "value", ParamDesc: "hex string (even length)"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})

		Declare(&Globalenv, &Declaration{
		Name: "randomBytes",
		Desc: "returns a string with numBytes cryptographically secure random bytes",
		Fn: func(a ...Scmer) Scmer {
				n := ToInt(a[0])
				if n < 0 {
					panic("randomBytes: numBytes must be non-negative")
				}
				buf := make([]byte, n)
				if n > 0 {
					if _, err := crand.Read(buf); err != nil {
						panic("error generating random bytes: " + fmt.Sprint(err))
					}
				}
				return NewString(string(buf))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "number", ParamName: "numBytes", ParamDesc: "number of random bytes"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})

		Declare(&Globalenv, &Declaration{
		Name: "regexp_replace",
		Desc: "replaces matches of a regex pattern in a string",
		Fn: func(a ...Scmer) Scmer {
				if a[0].IsNil() {
					return NewNil()
				}
				re, err := regexp.Compile(String(a[1]))
				if err != nil {
					panic("regexp_replace: invalid pattern: " + err.Error())
				}
				return NewString(re.ReplaceAllString(String(a[0]), String(a[2])))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "str", ParamDesc: "input string"}, &TypeDescriptor{Kind: "string", ParamName: "pattern", ParamDesc: "regex pattern"}, &TypeDescriptor{Kind: "string", ParamName: "replacement", ParamDesc: "replacement string"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
			Optimize: optimizeRegexpReplace,
		},
	})

		Declare(&Globalenv, &Declaration{
		Name: "fnv_hash",
		Desc: "computes a fast non-cryptographic 64-bit FNV-1a hash of a string, returns a 16-character hex string",
		Fn: func(a ...Scmer) Scmer {
				h := fnv.New64a()
				h.Write([]byte(String(a[0])))
				return NewString(fmt.Sprintf("%016x", h.Sum64()))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "str", ParamDesc: "input string to hash"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "sha1",
		Desc: "computes the SHA-1 digest of a string, returns a 40-character lowercase hex string",
		Fn: func(a ...Scmer) Scmer {
				sum := sha1.Sum([]byte(String(a[0])))
				return NewString(hex.EncodeToString(sum[:]))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "str", ParamDesc: "input string to hash"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})
		Declare(&Globalenv, &Declaration{
		Name: "sha256",
		Desc: "computes the SHA-256 digest of a string, returns a 64-character lowercase hex string",
		Fn: func(a ...Scmer) Scmer {
				sum := sha256.Sum256([]byte(String(a[0])))
				return NewString(hex.EncodeToString(sum[:]))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "str", ParamDesc: "input string to hash"}},
			Return: &TypeDescriptor{Kind: "string"},
			Const: true,
		},
	})

		Declare(&Globalenv, &Declaration{
		Name: "regexp_test",
		Desc: "tests if a string matches a regex pattern, returns true/false",
		Fn: func(a ...Scmer) Scmer {
				if a[0].IsNil() || a[1].IsNil() {
					return NewNil()
				}
				re, err := regexp.Compile(String(a[1]))
				if err != nil {
					panic("regexp_test: invalid pattern: " + err.Error())
				}
				return NewBool(re.MatchString(String(a[0])))
			},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", ParamName: "str", ParamDesc: "input string"}, &TypeDescriptor{Kind: "string", ParamName: "pattern", ParamDesc: "regex pattern"}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const: true,
			Optimize: optimizeRegexpTest,
		},
	})

}

// optimizeRegexpReplace precompiles the regex when the pattern argument is a constant string.
// This avoids calling regexp.Compile() on every invocation at runtime.
func optimizeRegexpReplace(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	// Optimize all arguments first
	result, td := oc.ApplyDefaultOptimization(v, useResult)
	if td != nil && td.Const {
		return result, td // already constant-folded
	}
	rv, ok := scmerSlice(result)
	if !ok || len(rv) < 4 {
		return result, td
	}
	// Check if the pattern (arg 2, index 2) is a constant string
	if !rv[2].IsString() {
		return result, td
	}
	pattern := rv[2].String()
	re, err := regexp.Compile(pattern)
	if err != nil {
		return result, td // let runtime handle the error
	}
	// Replace call with a precompiled closure
	compiled := NewFunc(func(a ...Scmer) Scmer {
		if a[0].IsNil() {
			return NewNil()
		}
		return NewString(re.ReplaceAllString(String(a[0]), String(a[1])))
	})
	// Rewrite: (regexp_replace str pattern repl) -> (compiled_fn str repl)
	return NewSlice([]Scmer{compiled, rv[1], rv[3]}), td
}

// optimizeRegexpTest precompiles the regex when the pattern argument is a constant string.
func optimizeRegexpTest(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	result, td := oc.ApplyDefaultOptimization(v, useResult)
	if td != nil && td.Const {
		return result, td
	}
	rv, ok := scmerSlice(result)
	if !ok || len(rv) < 3 {
		return result, td
	}
	// Check if the pattern (arg 2, index 2) is a constant string
	if !rv[2].IsString() {
		return result, td
	}
	pattern := rv[2].String()
	re, err := regexp.Compile(pattern)
	if err != nil {
		return result, td
	}
	compiled := NewFunc(func(a ...Scmer) Scmer {
		if a[0].IsNil() {
			return NewNil()
		}
		return NewBool(re.MatchString(String(a[0])))
	})
	// Rewrite: (regexp_test str pattern) -> (compiled_fn str)
	return NewSlice([]Scmer{compiled, rv[1]}), td
}
