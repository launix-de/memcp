/*
Copyright (C) 2023-2024  Carl-Philip Hänsch

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
	"regexp"
	"strings"
)

func valueFromPattern(pattern Scmer, en *Env) Scmer {
	switch p := pattern.(type) {
		case SourceInfo:
			return valueFromPattern(p.value, en)
		case Symbol:
			en = en.FindRead(p)
			if varcontent, ok := en.Vars[p]; ok {
				return varcontent
			} else {
				return p
			}
		case NthLocalVar:
			return en.VarsNumbered[p]
		case []Scmer:
			if len(p) == 2 && p[0] == Symbol("eval") {
				return Eval(p[1], en)
			}
	}
	return pattern
}

// pattern matching
func match(val Scmer, pattern Scmer, en *Env) bool {
	/* our custom implementation of match consisting of:
	(match value pattern result pattern result pattern result [default])
	where pattern may be string, number, Symbol or list or applications
	 - string and float64 will match on equality
	 - Symbol will read the value into a variable
	 - list will unify the list contents ([]scmer{"list", ...})
	 - _ is dontcare
	 - (concat string Symbol) will split prefix
	 - (concat Symbol string Symbol) will split infix
	 - (concat Symbol string) will split postfix
	 - (cons x y) will split a list (x and y will be unified)
	 - (merge '(a b c) rest) will split a list into multiple head elements and their rest (as alternative to cons)
	 - (regex "(.*)=(.*)" _ Symbol Symbol) will parse regex
	 - (eval expr) will match the value result from expr
	 - (string? Symbol) will match if value is a string and put the value into Symbol
	 - (number? Symbol) will match if value is a number and put the value into Symbol
	 - (list? Symbol) will match if value is a list and put the value into Symbol
	*/
	switch p := pattern.(type) {
		case SourceInfo:
			return match(val, p.value, en) // omit sourceinfo
		case int64, float64, string:
			return Equal(val, p)
		case Symbol:
			if p == Symbol("nil") {
				return val == nil
			} else if p == Symbol("true") {
				return val == true
			} else if p == Symbol("false") {
				return val == false
			} else {
				// unify value into variable
				en.Vars[p] = val
			}
			return true
		case NthLocalVar:
			en.VarsNumbered[p] = val
			return true
		case []Scmer:
			switch p[0] {
				case Symbol("eval"):
					// evaluate value and match then
					return Equal(Eval(p[1], en), val)
				case Symbol("var"):
					// unoptimized pattern
					en.VarsNumbered[ToInt(p[1])] = val
					return true
				case Symbol("list"):
					// list matching
					switch v := val.(type) {
						case []Scmer:
							p = p[1:] // extract rest of list
							// only list and list will match
							if len(v) != len(p) {
								return false
							}
							for i, p_item := range p {
								if !match(v[i], p_item, en) {
									return false
								}
							}
							return true
						default:
							return false
					}
				case Symbol("quote"):
					// symbol literal
					switch v := val.(type) {
						case Symbol:
							return p[1].(Symbol) == v
						default:
							return false
					}
				case Symbol("symbol"):
					// symbol literal
					switch v := val.(type) {
						case Symbol:
							return p[1].(Symbol) == v
						default:
							return false
					}
				case Symbol("string?"):
					// symbol literal
					switch v := val.(type) {
						case string:
							return match(v, p[1], en)
						case LazyString:
							return match(v, p[1], en)
						default:
							return false
					}
				case Symbol("number?"):
					// symbol literal
					switch v := val.(type) {
						case float64:
							return match(v, p[1], en)
						case int64:
							return match(v, p[1], en)
						default:
							return false
					}
				case Symbol("list?"):
					// symbol literal
					switch v := val.(type) {
						case []Scmer:
							return match(v, p[1], en)
						default:
							return false
					}
				case Symbol("ignorecase"):
					switch val2 := valueFromPattern(p[1], en).(type) {
						case LazyString:
							switch val1 := val.(type) {
								case string:
									return strings.EqualFold(val1, val2.GetValue())
							}
						case string:
							switch val1 := val.(type) {
								case string:
									return strings.EqualFold(val1, val2)
							}
					}
					return false
				case Symbol("concat"):
					return matchConcat(val, p[1:], en)
				case Symbol("merge"):
					switch v := val.(type) {
						case []Scmer: // only allowed for lists
							// examine the pattern
							if len(p) == 3 { // merge '(list) rest
								switch p1 := valueFromPattern(p[1], en).(type) {
									case []Scmer:
										if len(p1) > 0 && p1[0] == Symbol("list") && len(p1)-1 <= len(v) {
											for i := 1; i < len(p1); i++ {
												if !match(v[i-1], p1[i], en) {
													return false // pattern at position i dosen't match
												}
											}
											return match(v[len(p1)-1:], p[2], en) // match the rest of the array with the rest match pattern
										} else {
											return false
										}
									default:
										// panic
								}
								// TODO: Symbol string
							}
							panic("unknown merge pattern: " + fmt.Sprint(p))
						default:
							return false // non-lists are not matching
					}
				case Symbol("cons"):
					switch v := valueFromPattern(val, en).(type) {
						case []Scmer: // only matches on arrays
							if len(v) == 0 {
								return false // empty list does not match cons
							}
							return match(v[0], p[1], en) && match(v[1:], p[2], en)
						default:
							return false
					}
				case Symbol("regex"):
					// syntax: (match "v=5" (regex "^v=(.*)" _ v) (print "v is " v))
					// for multiline parsing, use (?ms:<REGEXP>)
					// for additional info, see https://github.com/google/re2/wiki/Syntax
					switch v := valueFromPattern(val, en).(type) {
						case string: // only allowed for strings
							switch p1 := valueFromPattern(p[1], en).(type) {
								case string:
									re, err := regexp.Compile(p1)
									if err != nil {
										panic(err)
									}
									if re.NumSubexp() != len(p) - 3 {
										panic("regex " + p1 + " contains " + fmt.Sprint(re.NumSubexp()) + " subexpressions, found " + fmt.Sprint(len(p)))
									}
									match := re.FindStringSubmatch(v)
									if match != nil {
										for i := 0; i <= re.NumSubexp(); i++ {
											if p[i+2] != Symbol("_") {
												switch v := p[i+2].(type) {
													case NthLocalVar:
														en.VarsNumbered[v] = match[i]
													case Symbol:
														en.Vars[v] = match[i]
													default:
														panic("regex variable invalid: "+SerializeToString(v, en))
												}
											}
										}
										return true
									} else {
										return false
									}
								case *regexp.Regexp:
									panic("TODO: precompiled regexp from optimize()")
								default:
									panic("regex expects string")
							}
						default:
							return false // non-strings are not matching regex
					}
				default:
					panic("unknown match pattern: " + fmt.Sprint(p))
			}
		default:
			panic("unknown match pattern: " + fmt.Sprint(p))
	}
}

func matchConcat(val Scmer, p []Scmer, en *Env) bool {
	switch v := val.(type) {
		case LazyString:
			panic("TODO: implement concat pattern on lazy strings")
		case string: // only allowed for strings
			// examine the pattern
			if len(p) == 0 && val == "" {
				// empty string
				return true
			}
			if len(p) == 1 {
				switch p0 := valueFromPattern(p[0], en).(type) {
					case NthLocalVar:
						// string = Symbol
						en.VarsNumbered[p0] = v
						return true
					case Symbol:
						// string = Symbol
						en.Vars[p0] = v
						return true
					default:
						// otherwise
				}
			}
			if len(p) >= 1 { // concat prefix rest
				switch p0 := valueFromPattern(p[0], en).(type) {
					case LazyString:
						panic("TODO: implement concat pattern on lazy strings")
					case string:
						// string Symbol
						if strings.HasPrefix(v, p0) {
							return matchConcat(v[len(p0):], p[1:], en) // match rest of pattern
						} else {
							return false
						}
					default:
				}
			}
			if len(p) == 2 { // var + suffix
				switch p0 := valueFromPattern(p[0], en).(type) {
					case NthLocalVar:
						// symbol + delimiter + rest
						switch p1 := valueFromPattern(p[1], en).(type) {
							case LazyString:
								panic("TODO: implement concat pattern on lazy strings")
							case string:
								// Symbol string
								if strings.HasSuffix(v, p1) {
									// extract postfix and match
									en.VarsNumbered[p0] = v[:len(v) - len(p1)]
									return true
								} else {
									return false
								}
						}
					case Symbol:
						switch p1 := valueFromPattern(p[1], en).(type) {
							case LazyString:
								panic("TODO: implement concat pattern on lazy strings")
							case string:
								// Symbol string
								if strings.HasSuffix(v, p1) {
									// extract postfix and match
									en.Vars[p0] = v[:len(v) - len(p1)]
									return true
								}
								// else
								return false
						}
					default:
						// panic
				}
			} else if len(p) >= 2 { // concat a b rest
				switch p0 := valueFromPattern(p[0], en).(type) {
					case NthLocalVar:
						// string Symbol
						switch p1 := valueFromPattern(p[1], en).(type) {
							case LazyString:
								panic("TODO: implement concat pattern on lazy strings")
							case string:
								idx := strings.Index(v, p1)
								if idx != -1 {
									// extract prefix and match rest
									en.VarsNumbered[p0] = v[:idx]
									return matchConcat(v[idx + len(p1):], p[2:], en)
								} else {
									return false
								}
							default:
								// panic
						}
					case Symbol:
						// string Symbol
						switch p1 := valueFromPattern(p[1], en).(type) {
							case LazyString:
								panic("TODO: implement concat pattern on lazy strings")
							case string:
								idx := strings.Index(v, p1)
								if idx != -1 {
									// extract prefix and match
									en.Vars[p0] = v[:idx]
									return matchConcat(v[idx + len(p1):], p[2:], en)
								} else {
									return false
								}
							default:
								// panic
						}
					default:
						// panic
				}
			}
			panic("unknown concat pattern: " + fmt.Sprint(p))
		default:
			return false // non-strings are not matching
	}
}


