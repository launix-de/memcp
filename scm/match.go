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

import (
	"fmt"
	"regexp"
	"reflect"
	"strings"
)

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
	 - (regex "(.*)=(.*)" _ Symbol Symbol) will parse regex
	*/
	switch p := pattern.(type) {
		case SourceInfo:
			return match(val, p.value, en) // omit sourceinfo
		case float64, string:
			return reflect.DeepEqual(val, p)
		case Symbol:
			en.Vars[p] = val
			return true
		case []Scmer:
			switch p[0] {
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
				case Symbol("symbol"):
					// symbol literal
					switch v := val.(type) {
						case Symbol:
							return p[1].(Symbol) == v
						default:
							return false
					}
				case Symbol("concat"):
					switch v := val.(type) {
						case string: // only allowed for strings
							// examine the pattern
							if len(p) == 3 {
								switch p1 := p[1].(type) {
									case Symbol:
										if val, ok := en.FindRead(p1).Vars[p1]; ok {
											if val_str, ok := val.(string); ok {
												// concat sym sym but left sym is assigned
												switch p2 := p[2].(type) {
													case Symbol:
														// string Symbol
														if strings.HasPrefix(v, val_str) {
															// extract postfix and match
															en.Vars[p2] = v[len(val_str):]
															return true
														}
														// else
														return false
													default:
														// panic
												}
											}
										}
									case string:
										switch p2 := p[2].(type) {
											case Symbol:
												// string Symbol
												if strings.HasPrefix(v, p1) {
													// extract postfix and match
													en.Vars[p2] = v[len(p1):]
													return true
												}
												// else
												return false
											default:
												// panic
										}
									default:
										// panic
								}
								// TODO: Symbol string
							}
							panic("unknown concat pattern: " + fmt.Sprint(p))
						default:
							return false // non-strings are not matching
					}
				case Symbol("cons"):
					switch v := val.(type) {
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
					switch v := val.(type) {
						case string: // only allowed for strings
							switch p1 := p[1].(type) {
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
												en.Vars[p[i+2].(Symbol)] = match[i]
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


