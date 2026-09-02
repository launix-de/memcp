/*
Copyright (C) 2023-2026  Carl-Philip Hänsch
Copyright (C) 2013  Pieter Kelchtermans (originally licensed unter WTFPL 2.0)

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
	"strconv"
	"strings"
	"sync/atomic"
)

var schemeStringUnescaper = strings.NewReplacer("\\\"", "\"", "\\\\", "\\", "\\n", "\n", "\\r", "\r", "\\t", "\t", "\\0", "\x00")

type SourceInfo struct {
	source string
	line   int
	col    int
	value  Scmer
	// Only evalWithSourceInfo may mark this flag. Static consumers such as the
	// optimizer may inspect or unwrap SourceInfo, but that is not execution.
	coverage uint32
}

func (source_info *SourceInfo) markInterpreted() {
	atomic.StoreUint32(&source_info.coverage, 1)
}

func (source_info *SourceInfo) wasInterpreted() bool {
	return atomic.LoadUint32(&source_info.coverage) != 0
}

func (source_info SourceInfo) String() string {
	return fmt.Sprintf("%s:%d:%d", source_info.source, source_info.line, source_info.col)
}

func Simplify(s string) Scmer {
	if len(s) > 0 && (s[0] == '[' || s[0] == '{') {
		if value, err := NewBSONFromJSON(s); err == nil {
			return value
		}
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return NewFloat(f)
	}
	return NewString(s)
}

func Read(source, s string) (expression Scmer) {
	tokens := tokenize(source, s)
	return readFrom(&tokens)
}

func EvalAll(source, s string, en *Env) (expression Scmer) {
	return evalAll(source, s, en, false)
}

// EvalAllJIT evaluates a Scheme module through the same parse, validate,
// optimize, and recursive JIT pipeline as an explicit (jit ...) call.
// Unsupported procedures stay ordinary Procs and retain the interpreter as an
// atomic fallback.
func EvalAllJIT(source, s string, en *Env) (expression Scmer) {
	return evalAll(source, s, en, true)
}

func evalAll(source, s string, en *Env, compileProcedures bool) (expression Scmer) {
	tokens := tokenize(source, s)
	deferredProcs := make(map[Symbol]struct{})
	for len(tokens) > 0 {
		code := readFrom(&tokens)
		definitionSymbol, hasDefinition := topLevelDefinitionSymbol(code)
		Validate(code, "any")
		code = Optimize(code, en, nil)
		expression = Eval(code, en)
		// On a vanilla Go toolchain there is no native entry point to produce.
		// Avoid walking every imported procedure for parser dependencies: besides
		// doing useless work, that delays every test-suite restart before the HTTP
		// listener becomes available.
		if compileProcedures && jitEnabled && expression.GetTag() == tagProc {
			if hasDefinition {
				compiled, entry, selected := jitCompileImportProc(definitionSymbol, expression)
				if entry == nil && jitExpressionContainsParser(expression.Proc().Body) {
					deferredProcs[definitionSymbol] = struct{}{}
				}
				if selected {
					expression = compiled
					target := en.definitionTarget()
					if target.Vars == nil {
						target.Vars = make(Vars)
					}
					target.Vars[definitionSymbol] = compiled
				}
			}
		}
	}
	if compileProcedures && jitEnabled {
		target := en.definitionTarget()
		// Parser-bearing procedures may refer to grammars declared later in the
		// module. Assemble those grammars first so the retry can fuse both the
		// parser machine and its generator expressions into the procedure body.
		jitCompileEnvironmentParsers(en)
		for sym := range deferredProcs {
			value, exists := target.Vars[sym]
			if !exists || value.GetTag() != tagProc || value.Proc() == nil || value.Proc().Compiled != nil {
				continue
			}
			compiled, _, selected := jitCompileImportProc(sym, value)
			if selected {
				target.Vars[sym] = compiled
			}
		}
	}
	return
}

func jitExpressionContainsParser(expression Scmer) bool {
	for expression.IsSourceInfo() {
		expression = expression.SourceInfo().value
	}
	items, ok := scmerSlice(expression)
	if !ok {
		return false
	}
	if len(items) != 0 {
		if scmerIsSymbol(items[0], "parser") {
			return true
		}
		if declaration := DeclarationForValue(items[0]); declaration != nil && declaration.IsSpecialForm && declaration.Name == "parser" {
			return true
		}
	}
	for _, item := range items {
		if jitExpressionContainsParser(item) {
			return true
		}
	}
	return false
}

func jitCompileImportProc(sym Symbol, value Scmer) (Scmer, *JITEntryPoint, bool) {
	compiled := jitCompileProbe(value)
	var entry *JITEntryPoint
	if compiled.GetTag() == tagProc && compiled.Proc() != nil {
		entry = compiled.Proc().Compiled
	}
	selected := entry != nil && jitAutoImportCoverageWorthwhile(entry.Coverage)
	if selected {
		value.Proc().Compiled = entry
		compiled = value
	}
	if entry != nil {
		entry.DebugName = string(sym)
		maybeLogJITCodeName(entry)
	}
	maybeLogJITImportCandidate(sym, entry, selected)
	if selected && JITLog {
		fmt.Printf("JIT: import %s code=%p bytes=%d hidden-args=%d expressions=%d dynamic-calls=%d inlined-calls=%d\n",
			sym, entry.CodePtr, entry.CodeLen, len(entry.HiddenArgs),
			entry.Coverage.Expressions, entry.Coverage.DynamicCalls, entry.Coverage.InlinedCalls)
	}
	return compiled, entry, selected
}

func jitAutoImportCoverageWorthwhile(coverage JITCoverage) bool {
	// A generic Apply bridge costs more than a handful of straight-line emitter
	// nodes. Keep probe compilation universal, but activate only native bodies
	// whose static coverage can amortize every remaining bridge. This decision is
	// deliberately independent of module paths and definition names.
	return coverage.DynamicCalls == 0 || coverage.Expressions >= coverage.DynamicCalls*8
}

func topLevelDefinitionSymbol(code Scmer) (Symbol, bool) {
	for code.IsSourceInfo() {
		code = code.SourceInfo().value
	}
	if !code.IsSlice() {
		return "", false
	}
	items := code.Slice()
	if len(items) < 3 {
		return "", false
	}
	head, headOK := scmerSymbol(items[0])
	if !headOK {
		if declaration := DeclarationForValue(items[0]); declaration != nil && declaration.IsSpecialForm {
			head, headOK = Symbol(declaration.Name), true
		}
	}
	symbol, symbolOK := scmerSymbol(items[1])
	if !headOK || (head != "define" && head != "set") || !symbolOK {
		return "", false
	}
	return symbol, true
}

// Syntactic Analysis
func readFrom(tokens *[]Scmer) (expression Scmer) {
	if len(*tokens) == 0 {
		return NewNil()
	}
	var source_info SourceInfo
	// pop first element from tokens
	token := (*tokens)[0]
	*tokens = (*tokens)[1:]
	if token.IsSourceInfo() {
		source_info = *token.SourceInfo()
		token = source_info.value
	}
	if token.IsSymbol() {
		sym := token.String()
		if sym == "(" {
			L := make([]Scmer, 0)
			for {
				if len(*tokens) == 0 {
					panic(source_info.String() + ": expecting matching )")
				}
				next := (*tokens)[0]
				if next.IsSymbol() && next.String() == ")" {
					*tokens = (*tokens)[1:]
					source_info.value = NewSlice(L)
					return NewSourceInfo(source_info)
				}
				L = append(L, readFrom(tokens))
			}
		}
		if sym == "'" && len(*tokens) > 0 {
			next := (*tokens)[0]
			if next.IsSourceInfo() {
				source_info = *next.SourceInfo()
				next = source_info.value
			}
			if next.IsSymbol() && next.String() == "(" {
				*tokens = (*tokens)[1:]
				L := make([]Scmer, 1)
				L[0] = NewSymbol("list")
				for {
					if len(*tokens) == 0 {
						panic(source_info.String() + ": expecting matching )")
					}
					next2 := (*tokens)[0]
					if next2.IsSymbol() && next2.String() == ")" {
						break
					}
					L = append(L, readFrom(tokens))
				}
				*tokens = (*tokens)[1:]
				listForm := NewSlice(L)
				if source_info.source != "" {
					source_info.value = listForm
					return NewSourceInfo(source_info)
				}
				return listForm
			}
			quoted := readFrom(tokens)
			quoteElems := make([]Scmer, 2)
			quoteElems[0] = NewSymbol("quote")
			quoteElems[1] = quoted
			quoteForm := NewSlice(quoteElems)
			if source_info.source != "" {
				source_info.value = quoteForm
				return NewSourceInfo(source_info)
			}
			return quoteForm
		}
		return token
	}
	return token
}

// Lexical Analysis
func tokenize(source, s string) []Scmer {
	/* tokenizer state machine:
		0 = expecting next item
		1 = inside Number
		2 = inside Symbol
		3 = inside string
		4 = inside escaping sequence of string
		5 = inside comment
		6 = comment ending * from * /

	tokens are either Number, Symbol, string or Symbol('(') or Symbol(')')
	*/

	/* TODO:
	- count lines, track line+col
	- for certain symbols (mostly only '(') store a position object in the token array (consisting of source, line, col)
	*/
	line := 1
	col := 0

	state := 0
	startToken := 0
	result := make([]Scmer, 0)
	for i, ch := range s {
		// line counting
		if ch == '\n' {
			line++
			col = 1
		} else {
			col++
		}

		if state == 1 && (ch == '.' || ch >= '0' && ch <= '9') {
			// another character added to Number
		} else if state == 2 && ch == '*' && s[startToken:i] == "/" {
			// begin of comment
			state = 5
		} else if state == 5 && ch == '*' {
			// comment seems to end
			state = 6
		} else if state == 5 {
			// consume another character in comment (TODO: nested comment counting??)
		} else if state == 6 && ch == '/' {
			// end comment
			state = 0
		} else if state == 6 {
			// continue comment
			state = 5
		} else if state == 2 && ch != ' ' && ch != '\r' && ch != '\n' && ch != '\t' && ch != ')' && ch != '(' {
			// another character added to Symbol
		} else if state == 3 && ch != '"' && ch != '\\' {
			// another character added to string
		} else if state == 3 && ch == '\\' {
			// escape sequence
			state = 4
		} else if state == 4 {
			state = 3 // continue with string
		} else if state == 3 && ch == '"' {
			// finish string
			result = append(result, NewString(schemeStringUnescaper.Replace(string(s[startToken+1:i]))))
			state = 0
		} else {
			// otherwise: state change!
			if state == 1 {
				// finish Number
				if f, err := strconv.ParseFloat(s[startToken:i], 64); err == nil {
					result = append(result, NewFloat(f))
				} else if s[startToken:i] == "-" {
					result = append(result, NewSymbol("-"))
				} else {
					result = append(result, NewSymbol("NaN"))
				}
			}
			if state == 2 {
				// finish Symbol
				result = append(result, NewSymbol(s[startToken:i]))
			}
			// now detect what to parse next
			startToken = i
			if ch == '(' {
				result = append(result, NewSourceInfo(SourceInfo{source: source, line: line, col: col, value: NewSymbol("(")}))
				state = 0
			} else if ch == ')' {
				result = append(result, NewSymbol(")"))
				state = 0
			} else if ch == '\'' {
				result = append(result, NewSymbol("'"))
				state = 0
			} else if ch == '"' {
				// start string
				state = 3
			} else if ch >= '0' && ch <= '9' || ch == '-' {
				// start Number
				state = 1
			} else if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
				// white space
				state = 0
			} else {
				// everything else is a Symbol! (Symbols only are stopped by ' ()')
				state = 2
			}

		}
	}
	// in the end: finish unfinished Symbols and Numbers
	if state == 1 {
		// finish Number
		if f, err := strconv.ParseFloat(s[startToken:], 64); err == nil {
			result = append(result, NewFloat(f))
		} else if s[startToken:] == "-" {
			result = append(result, NewSymbol("-"))
		} else {
			result = append(result, NewSymbol("NaN"))
		}
	}
	if state == 2 {
		// finish Symbol
		result = append(result, NewSymbol(s[startToken:]))
	}
	return result
}
