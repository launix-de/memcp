/*
 * A minimal Scheme interpreter, as seen in lis.py and SICP
 * http://norvig.com/lispy.html
 * http://mitpress.mit.edu/sicp/full-text/sicp/book/node77.html
 *
 * Pieter Kelchtermans 2013
 * LICENSE: WTFPL 2.0
 */
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"bytes"
)

func toBool(v scmer) bool {
	switch v.(type) {
		case nil:
			return false
		case string:
			return v != ""
		case number:
			return v != 0
		case bool:
			return v != false
		default:
			// []scmer, native function, lambdas
			return true
	}
}

/*
 Eval / Apply
*/

func eval(expression scmer, en *env) (value scmer) {
	switch e := expression.(type) {
	case string:
		value = e
	case number:
		value = e
	case symbol:
		value = en.FindRead(e).vars[e]
	case []scmer:
		switch car, _ := e[0].(symbol); car {
		case "quote":
			value = e[1]
		case "if":
			if toBool(eval(e[1], en)) {
				value = eval(e[2], en)
			} else {
				value = eval(e[3], en)
			}
		/* set! is forbidden due to side effects
		case "set!":
			v := e[1].(symbol)
			en2 := en.FindWrite(v)
			if en2 == nil {
				// not yet defined: set in innermost env
				en2 = en
			}
			en.vars[v] = eval(e[2], en)
			value = "ok"*/
		case "define", "set": // set only works in innermost env
			en.vars[e[1].(symbol)] = eval(e[2], en)
			value = "ok"
		case "lambda":
			value = proc{e[1], e[2], en}
		case "begin":
			for _, i := range e[1:] {
				value = eval(i, en)
			}
		default:
			operands := e[1:]
			values := make([]scmer, len(operands))
			for i, x := range operands {
				values[i] = eval(x, en)
			}
			value = apply(eval(e[0], en), values)
		}
	default:
		log.Println("Unknown expression type - EVAL", e)
	}
	return
}

func apply(procedure scmer, args []scmer) (value scmer) {
	switch p := procedure.(type) {
	case func(...scmer) scmer:
		value = p(args...)
	case proc:
		en := &env{make(vars), p.en}
		switch params := p.params.(type) {
		case []scmer:
			for i, param := range params {
				en.vars[param.(symbol)] = args[i]
			}
		default:
			en.vars[params.(symbol)] = args
		}
		value = eval(p.body, en)
	default:
		log.Println("Unknown procedure type - APPLY", p)
	}
	return
}

// TODO: func optimize für parzielle lambda-Ausdrücke und JIT

type proc struct {
	params, body scmer
	en           *env
}

/*
 Environments
*/

type vars map[symbol]scmer
type env struct {
	vars
	outer *env
}

func (e *env) FindRead(s symbol) *env {
	if _, ok := e.vars[s]; ok {
		return e
	} else {
		if e.outer == nil {
			return e
		}
		return e.outer.FindRead(s)
	}
}

func (e *env) FindWrite(s symbol) *env {
	if _, ok := e.vars[s]; ok {
		return e
	} else {
		if e.outer == nil {
			return nil
		}
		return e.outer.FindWrite(s)
	}
}

/*
 Primitives
*/

var globalenv env

func init() {
	globalenv = env{
		vars{ //aka an incomplete set of compiled-in functions
			"+": func(a ...scmer) scmer {
				v := a[0].(number)
				for _, i := range a[1:] {
					v += i.(number)
				}
				return v
			},
			"-": func(a ...scmer) scmer {
				v := a[0].(number)
				for _, i := range a[1:] {
					v -= i.(number)
				}
				return v
			},
			"*": func(a ...scmer) scmer {
				v := a[0].(number)
				for _, i := range a[1:] {
					v *= i.(number)
				}
				return v
			},
			"/": func(a ...scmer) scmer {
				v := a[0].(number)
				for _, i := range a[1:] {
					v /= i.(number)
				}
				return v
			},
			"<=": func(a ...scmer) scmer {
				return a[0].(number) <= a[1].(number)
			},
			"<": func(a ...scmer) scmer {
				return a[0].(number) < a[1].(number)
			},
			">": func(a ...scmer) scmer {
				return a[0].(number) > a[1].(number)
			},
			">=": func(a ...scmer) scmer {
				return a[0].(number) >= a[1].(number)
			},
			"equal?": func(a ...scmer) scmer {
				return reflect.DeepEqual(a[0], a[1])
			},
			"cons": func(a ...scmer) scmer {
				// cons a b: prepend item a to list b (construct list from item + tail)
				switch car := a[0]; cdr := a[1].(type) {
				case []scmer:
					return append([]scmer{car}, cdr...)
				default:
					return []scmer{car, cdr}
				}
			},
			"car": func(a ...scmer) scmer {
				// head of tuple
				return a[0].([]scmer)[0]
			},
			"cdr": func(a ...scmer) scmer {
				// rest of tuple
				return a[0].([]scmer)[1:]
			},
			"concat": func(a ...scmer) scmer {
				// concat strings
				var b bytes.Buffer
				for _, s := range a {
					b.WriteString(String(s))
				}
				return b.String()
			},
			"true": true,
			"false": false,
			"list": eval(read(
				"(lambda z z)"),
				&globalenv),
		},
		nil}
}

/* TODO: abs, quotient, remainder, modulo, gcd, lcm, expt, sqrt
zero?, negative?, positive?, off?, even?
max, min
sin, cos, tan, asin, acos, atan
exp, log
number->string, string->number
integer?, rational?, real?, complex?, number?
*/

/*
 Parsing
*/

//symbols, numbers, expressions, procedures, lists, ... all implement this interface, which enables passing them along in the interpreter
type scmer interface{}

type symbol string  //symbols are represented by strings
type number float64 //numbers by float64

func read(s string) (expression scmer) {
	tokens := tokenize(s)
	return readFrom(&tokens)
}

//Syntactic Analysis
func readFrom(tokens *[]scmer) (expression scmer) {
	//pop first element from tokens
	token := (*tokens)[0]
	*tokens = (*tokens)[1:]
	switch token.(type) {
		case symbol:
			if token == symbol("(") {
				L := make([]scmer, 0)
				for (*tokens)[0] != symbol(")") {
					L = append(L, readFrom(tokens))
				}
				*tokens = (*tokens)[1:]
				return L
			} else if token == symbol("'") && len(*tokens) > 0 {
				if (*tokens)[0] == symbol("(") {
					*tokens = (*tokens)[1:]
					// list literal
					L := make([]scmer, 1)
					L[0] = symbol("list")
					for (*tokens)[0] != symbol(")") {
						L = append(L, readFrom(tokens))
					}
					*tokens = (*tokens)[1:]
					return L
				} else {
					return token
				}
			} else {
				return token
			}
		default:
			// string, number
			return token
	}
}

//Lexical Analysis
func tokenize(s string) []scmer {
	/* tokenizer state machine:
		0 = expecting next item
		1 = inside number
		2 = inside symbol
		3 = inside string
		4 = inside escaping sequence of string
	
	tokens are either number, symbol, string or symbol('(') or symbol(')')
	*/
	stringreplacer := strings.NewReplacer("\\\"", "\"", "\\\\", "\\", "\\n", "\n", "\\r", "\r", "\\t", "\t")
	state := 0
	startToken := 0
	result := make([]scmer, 0)
	for i, ch := range s {
		if state == 1 && (ch == '.' || ch >= '0' && ch <= '9') {
			// another character added to number
		} else if state == 2 && ch != ' ' && ch != '\r' && ch != '\n' && ch != '\t' && ch != ')' && ch != '(' {
			// another character added to symbol
		} else if state == 3 && ch != '"' && ch != '\\' {
			// another character added to string
		} else if state == 3 && ch == '\\' {
			// escape sequence
			state = 4
		} else if state == 4 {
			state = 3 // continue with string
		} else if state == 3 && ch == '"' {
			// finish string
			result = append(result, stringreplacer.Replace(string(s[startToken+1:i])))
			state = 0
		} else {
			// otherwise: state change!
			if state == 1 {
				// finish number
				if f, err := strconv.ParseFloat(s[startToken:i], 64); err == nil {
					result = append(result, number(f))
				} else if s[startToken:i] == "-" {
					result = append(result, symbol("-"))
				} else {
					result = append(result, symbol("NaN"))
				}
			}
			if state == 2 {
				// finish symbol
				result = append(result, symbol(s[startToken:i]))
			}
			// now detect what to parse next
			startToken = i
			if ch == '(' {
				result = append(result, symbol("("))
				state = 0
			} else if ch == ')' {
				result = append(result, symbol(")"))
				state = 0
			} else if ch == '"' {
				// start string
				state = 3
			} else if ch >= '0' && ch <= '9' || ch == '-' {
				// start number
				state = 1
			} else if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
				// white space
				state = 0
			} else {
				// everything else is a symbol! (symbols only are stopped by ' ()')
				state = 2
			}

		}
	}
	// in the end: finish unfinished symbols and numbers
	if state == 1 {
		// finish number
		if f, err := strconv.ParseFloat(s[startToken:], 64); err == nil {
			result = append(result, number(f))
		} else if s[startToken:] == "-" {
			result = append(result, symbol("-"))
		} else {
			result = append(result, symbol("NaN"))
		}
	}
	if state == 2 {
		// finish symbol
		result = append(result, symbol(s[startToken:]))
	}
	return result
}

/*
 Interactivity
*/

func String(v scmer) string {
	switch v := v.(type) {
	case []scmer:
		l := make([]string, len(v))
		for i, x := range v {
			l[i] = String(x)
		}
		return "(" + strings.Join(l, " ") + ")"
	case proc:
		return "[func]"
	case func(...scmer) scmer:
		return "[native func]"
	default:
		return fmt.Sprint(v)
	}
}
func Serialize(b *bytes.Buffer, v scmer, en *env) {
	switch v := v.(type) {
	case []scmer:
		b.WriteByte('(')
		for i, x := range v {
			if i != 0 {
				b.WriteByte(' ')
			}
			Serialize(b, x, en)
		}
		b.WriteByte(')')
	case func(...scmer) scmer:
		// native func serialization is the hardest; reverse the env!
		// when later functional JIT is done, this must also handle deoptimization
		en2 := en
		for en2 != nil {
			for k, v2 := range en2.vars {
				// compare function pointers (hacky but golang dosent give another opt)
				if fmt.Sprint(v) == fmt.Sprint(v2) {
					// found the right global function
					b.WriteString(fmt.Sprint(k)) // print out variable name
					return
				}
			}
			en2 = en2.outer
		}
		b.WriteString("[unserializable native func]")
	case proc:
		b.WriteString("(lambda ")
		Serialize(b, v.params, v.en)
		b.WriteByte(' ')
		Serialize(b, v.body, v.en)
		b.WriteByte(')')
	case symbol:
		for en != nil && en != &globalenv {
			if v, ok := en.vars[v]; ok {
				// if symbol is defined in a lambda, print the real value
				Serialize(b, v, en)
				return
			}
			en = en.outer
		}
		// otherwise print as symbol
		b.WriteString(fmt.Sprint(v))
	case string:
		b.WriteByte('"')
		b.WriteString(strings.NewReplacer("\"", "\\\"", "\\", "\\\\", "\r", "\\r", "\n", "\\n").Replace(v))
		b.WriteByte('"')
	default:
		b.WriteString(fmt.Sprint(v))
	}
}

func Repl() {
	scanner := bufio.NewScanner(os.Stdin)
	for fmt.Print("> "); scanner.Scan(); fmt.Print("> ") {
		var b bytes.Buffer
		Serialize(&b, eval(read(scanner.Text()), &globalenv), &globalenv)
		fmt.Println("==>", b.String())
	}
}
