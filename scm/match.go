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

func scmerSymbolName(s Scmer) (string, bool) {
	if s.IsSymbol() {
		return s.String(), true
	}
	if auxTag(s.aux) == tagAny {
		if sym, ok := s.Any().(Symbol); ok {
			return string(sym), true
		}
	}
	return "", false
}

func scmerAsSlice(v Scmer) ([]Scmer, bool) {
	if v.IsSlice() {
		return v.Slice(), true
	}
	if auxTag(v.aux) == tagAny {
		if slice, ok := v.Any().([]Scmer); ok {
			return slice, true
		}
	}
	return nil, false
}

func scmerAsString(v Scmer) (string, bool) {
	if auxTag(v.aux) == tagString {
		return v.String(), true
	}
	if auxTag(v.aux) == tagAny {
		if s, ok := v.Any().(string); ok {
			return s, true
		}
	}
	return "", false
}

func valueFromPattern(pattern Scmer, en *Env) Scmer {
	if pattern.IsSourceInfo() {
		return valueFromPattern(pattern.SourceInfo().value, en)
	}
	switch v := pattern.Any().(type) {
	case SourceInfo:
		return valueFromPattern(v.value, en)
	case Symbol:
		en = en.FindRead(v)
		if value, ok := en.Vars[v]; ok {
			return value
		}
		return pattern
	case NthLocalVar:
		return en.VarsNumbered[v]
	case []Scmer:
		if len(v) == 2 && v[0].SymbolEquals("eval") {
			return Eval(v[1], en)
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
	if pattern.IsSourceInfo() {
		return match(val, pattern.SourceInfo().value, en)
	}
	switch p := pattern.Any().(type) {
	case SourceInfo:
		return match(val, p.value, en) // omit sourceinfo
	case int64, float64, string:
		return Equal(val, pattern)
	case Symbol:
		switch string(p) {
		case "nil":
			return val.IsNil()
		case "true":
			return val.Bool()
		case "false":
			return !val.Bool()
		default:
			en.Vars[p] = val
			return true
		}
	case NthLocalVar:
		en.VarsNumbered[p] = val
		return true
	case []Scmer:
		name, ok := scmerSymbolName(p[0])
		if !ok {
			panic("unknown match pattern head: " + SerializeToString(p[0], en))
		}
		switch name {
		case "eval":
			// evaluate value and match then
			return Equal(Eval(p[1], en), val)
		case "var":
			// unoptimized pattern
			en.VarsNumbered[int(p[1].Int())] = val
			return true
		case "list":
			// list matching
			if list, ok := scmerAsSlice(val); ok {
				p = p[1:]
				if len(list) != len(p) {
					return false
				}
				for i, pat := range p {
					if !match(list[i], pat, en) {
						return false
					}
				}
				return true
			}
			return false
		case "quote":
			// symbol literal
			if sym, ok := scmerSymbolName(val); ok {
				if expected, ok := scmerSymbolName(p[1]); ok {
					return sym == expected
				}
			}
			return false
		case "symbol":
			// symbol literal
			if sym, ok := scmerSymbolName(val); ok {
				if expected, ok := scmerSymbolName(p[1]); ok {
					return sym == expected
				}
			}
			return false
		case "string?":
			// symbol literal
			if _, ok := scmerAsString(val); ok {
				return match(val, p[1], en)
			}
			return false
		case "number?":
			// symbol literal
			if val.IsInt() || val.IsFloat() {
				return match(val, p[1], en)
			}
			return false
		case "list?":
			// symbol literal
			if list, ok := scmerAsSlice(val); ok {
				return match(NewSlice(list), p[1], en)
			}
			return false
		case "ignorecase":
			if val2, ok := scmerAsString(valueFromPattern(p[1], en)); ok {
				if val1, ok := scmerAsString(val); ok {
					return strings.EqualFold(val1, val2)
				}
			}
			return false
		case "concat":
			return matchConcat(val, p[1:], en)
		case "merge":
			if list, ok := scmerAsSlice(val); ok {
				if len(p) == 3 {
					if subPattern, ok := scmerAsSlice(valueFromPattern(p[1], en)); ok {
						if len(subPattern) > 0 {
							if head, ok := scmerSymbolName(subPattern[0]); ok && head == "list" && len(subPattern)-1 <= len(list) {
								for i := 1; i < len(subPattern); i++ {
									if !match(list[i-1], subPattern[i], en) {
										return false
									}
								}
								return match(NewSlice(list[len(subPattern)-1:]), p[2], en)
							}
						}
					}
				}
				panic("unknown merge pattern: " + fmt.Sprint(p))
			}
			return false
		case "cons":
			if list, ok := scmerAsSlice(valueFromPattern(val, en)); ok {
				if len(list) == 0 {
					return false
				}
				return match(list[0], p[1], en) && match(NewSlice(list[1:]), p[2], en)
			}
			return false
		case "regex":
			// syntax: (match "v=5" (regex "^v=(.*)" _ v) (print "v is " v))
			// for multiline parsing, use (?ms:<REGEXP>)
			// for additional info, see https://github.com/google/re2/wiki/Syntax
			if str, ok := scmerAsString(valueFromPattern(val, en)); ok {
				if patternStr, ok := scmerAsString(valueFromPattern(p[1], en)); ok {
					re, err := regexp.Compile(patternStr)
					if err != nil {
						panic(err)
					}
					if re.NumSubexp() != len(p)-3 {
						panic("regex " + patternStr + " contains " + fmt.Sprint(re.NumSubexp()) + " subexpressions, found " + fmt.Sprint(len(p)))
					}
					match := re.FindStringSubmatch(str)
					if match != nil {
						for i := 0; i <= re.NumSubexp(); i++ {
							if name, ok := scmerSymbolName(p[i+2]); ok && name == "_" {
								continue
							}
							if idx, ok := p[i+2].Any().(NthLocalVar); ok {
								en.VarsNumbered[idx] = NewString(match[i])
								continue
							}
							if sym, ok := scmerSymbolName(p[i+2]); ok {
								en.Vars[Symbol(sym)] = NewString(match[i])
								continue
							}
							panic("regex variable invalid: " + SerializeToString(p[i+2], en))
						}
						return true
					}
					return false
				}
			}
			panic("regex expects string")
		default:
			panic("unknown match pattern: " + fmt.Sprint(p))
		}
	default:
		panic("unknown match pattern: " + fmt.Sprint(p))
	}
}

func matchConcat(val Scmer, p []Scmer, en *Env) bool {
	str, ok := scmerAsString(val)
	if !ok {
		return false
	}
	if len(p) == 0 {
		return str == ""
	}
	if len(p) == 1 {
		target := valueFromPattern(p[0], en)
		if idx, ok := target.Any().(NthLocalVar); ok {
			en.VarsNumbered[idx] = NewString(str)
			return true
		}
		if name, ok := scmerSymbolName(target); ok {
			en.Vars[Symbol(name)] = NewString(str)
			return true
		}
		return false
	}
	first := valueFromPattern(p[0], en)
	if prefix, ok := scmerAsString(first); ok {
		if strings.HasPrefix(str, prefix) {
			return matchConcat(NewString(str[len(prefix):]), p[1:], en)
		}
		return false
	}
	if len(p) == 2 {
		suffixVal := valueFromPattern(p[1], en)
		if suffix, ok := scmerAsString(suffixVal); ok {
			if strings.HasSuffix(str, suffix) {
				base := str[:len(str)-len(suffix)]
				if idx, ok := first.Any().(NthLocalVar); ok {
					en.VarsNumbered[idx] = NewString(base)
					return true
				}
				if name, ok := scmerSymbolName(first); ok {
					en.Vars[Symbol(name)] = NewString(base)
					return true
				}
			}
			return false
		}
	}
	if len(p) >= 2 {
		delimVal := valueFromPattern(p[1], en)
		if delim, ok := scmerAsString(delimVal); ok {
			idx := strings.Index(str, delim)
			if idx == -1 {
				return false
			}
			prefix := str[:idx]
			rest := NewString(str[idx+len(delim):])
			if id, ok := first.Any().(NthLocalVar); ok {
				en.VarsNumbered[id] = NewString(prefix)
			} else if name, ok := scmerSymbolName(first); ok {
				en.Vars[Symbol(name)] = NewString(prefix)
			} else {
				return false
			}
			return matchConcat(rest, p[2:], en)
		}
	}
	panic("unknown concat pattern: " + fmt.Sprint(p))
}
