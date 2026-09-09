/*
Copyright (C) 2025-2026  MemCP Contributors

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
import "bytes"
import "strconv"
import "strings"

// These operations expose language metadata, independent of SQL or storage.
func initExpressionMetadata() {
	Declare(&Globalenv, &Declaration{Name: "expression_syntax", Fn: func(a ...Scmer) Scmer {
		return ExpressionSyntax(a[0])
	}, Type: &TypeDescriptor{Kind: "func", Description: "exposes source-independent AST data with named special forms",
		Params: []*TypeDescriptor{{Kind: "any", Label: "expression"}}, Return: &TypeDescriptor{Kind: "any"}, Const: true,
		JITEmit: jitEmitExpressionMetadataCall("expression_syntax"),
	},
	})
	Declare(&Globalenv, &Declaration{Name: "expression_equal?", Fn: func(a ...Scmer) Scmer {
		return NewBool(astStructuralEqual(a[0], a[1]))
	}, Type: &TypeDescriptor{Kind: "func", Description: "compares expression data without scalar truth or numeric coercion",
		Params: []*TypeDescriptor{{Kind: "any", Label: "left"}, {Kind: "any", Label: "right"}},
		Return: &TypeDescriptor{Kind: "bool"}, Const: true,
		JITEmit: jitEmitExpressionMetadataCall("expression_equal?"),
	},
	})
	Declare(&Globalenv, &Declaration{Name: "expression_name", Fn: func(a ...Scmer) Scmer {
		names := make([]string, len(a[2].Slice()))
		for i, name := range a[2].Slice() {
			names[i] = name.String()
		}
		return NewString(ExpressionName(a[0], names, a[1].Slice()))
	}, Type: &TypeDescriptor{Kind: "func", Description: "encodes an expression with explicit names for its positional parameters",
		Params:  []*TypeDescriptor{{Kind: "any", Label: "expression"}, {Kind: "list", Label: "parameters"}, {Kind: "list", Label: "names"}},
		Return:  &TypeDescriptor{Kind: "string"},
		JITEmit: jitEmitExpressionMetadataCall("expression_name"),
	},
	})
	Declare(&Globalenv, &Declaration{Name: "expression_foldable?", Fn: func(a ...Scmer) Scmer {
		declaration := DeclarationForValue(a[0])
		return NewBool(declaration != nil && declaration.IsFoldable())
	}, Type: &TypeDescriptor{Kind: "func", Description: "reports whether a callable declares foldable evaluation",
		Params: []*TypeDescriptor{{Kind: "any", Label: "callable"}}, Return: &TypeDescriptor{Kind: "bool"},
		JITEmit: jitEmitExpressionMetadataCall("expression_foldable?"),
	},
	})
}

// Expression metadata traverses arbitrary trees and runtime declarations.
// Keep those operations behind a native call instead of inlining their
// recursive implementation into each compiled consumer.
func jitEmitExpressionMetadataCall(name string) func(*JITContext, []Scmer, []JITValueDesc, JITValueDesc) JITValueDesc {
	return func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		ctx.Coverage.NativeCalls++
		return jitEmitGeneratedCallBoundary(ctx, declarations[name], sourceArgs, args, result)
	}
}

// ExpressionSyntax normalizes reader-resolved special forms for data-oriented
// language tooling. Ordinary callable values retain their lexical identity.
func ExpressionSyntax(expr Scmer) Scmer {
	expr = expr.WithoutSourceInfo()
	if expr.GetTag() == tagSpecialForm || expr.GetTag() == tagFunc {
		if declaration := DeclarationForValue(expr); declaration != nil && declaration.IsSpecialForm {
			return NewSymbol(declaration.Name)
		}
	}
	if !expr.IsSlice() {
		return expr
	}
	items := expr.Slice()
	var result []Scmer
	for i, item := range items {
		normalized := ExpressionSyntax(item)
		if normalized != item && result == nil {
			result = append([]Scmer(nil), items...)
		}
		if result != nil {
			result[i] = normalized
		}
	}
	if result == nil {
		return expr
	}
	return NewSlice(result)
}

// WriteExpressionName prints a compact textual encoding of a Scheme AST to w.
// Unknowns print as "?".
// - Unknown symbols (not a global function and not one of the provided column names) => "?".
// - Lambdas (Proc) retain their process-local identity.
// - Go builtins (func(...Scmer) Scmer) => function name if found in Globalenv, else "?".
// For filters, pass the condition Proc.Body as v and the filter columns as context.
// For sort expressions, pass the string column name or the Proc.Body with its params as context.
// columnSymbols must be the Proc.Params list when encoding a lambda body. If present:
// - Any symbol equal to a param prints as the corresponding column name by index.
// - Any NthLocalVar(i) prints as columns[i] (when i < len(columns)); otherwise "?".
func WriteExpressionName(v Scmer, w io.Writer, columns []string, columnSymbols []Scmer) {
	cols := make(map[string]bool, len(columns))
	for _, c := range columns {
		cols[strings.ToLower(c)] = true
	}
	// Build symbol->index from Proc.Params to map lambda params to actual columns
	symIndex := make(map[string]int, len(columnSymbols))
	for i, s := range columnSymbols {
		if s.IsSymbol() {
			symIndex[strings.ToLower(s.String())] = i
			continue
		}
		if sym, ok := s.Any().(Symbol); ok {
			symIndex[strings.ToLower(string(sym))] = i
		}
	}

	var enc func(Scmer)
	writeSymbolOrColumn := func(s string) {
		sLower := strings.ToLower(s)
		// Prefer mapping lambda param -> column name
		if idx, ok := symIndex[sLower]; ok {
			if idx >= 0 && idx < len(columns) {
				io.WriteString(w, columns[idx])
				return
			}
			io.WriteString(w, "?")
			return
		}
		// Otherwise, if it looks like a global function/operator, print symbol
		if Globalenv.FindRead(Symbol(s)) != nil {
			io.WriteString(w, s)
			return
		}
		// Unknown
		io.WriteString(w, "?")
	}

	var numBuf [64]byte // stack-allocated buffer for number formatting
	enc = func(node Scmer) {
		switch {
		case node.IsNil():
			io.WriteString(w, "nil")
		case node.IsBool():
			if node.Bool() {
				io.WriteString(w, "true")
			} else {
				io.WriteString(w, "false")
			}
		case node.IsInt():
			b := strconv.AppendInt(numBuf[:0], node.Int(), 10)
			w.Write(b)
		case node.IsFloat():
			b := strconv.AppendFloat(numBuf[:0], node.Float(), 'g', -1, 64)
			w.Write(b)
		case node.IsString():
			s, _ := node.AppendString(nil) // zero-alloc for tagString
			io.WriteString(w, "\"")
			io.WriteString(w, s)
			io.WriteString(w, "\"")
		case node.IsSymbol():
			s, _ := node.AppendString(nil) // zero-alloc for tagSymbol
			writeSymbolOrColumn(s)
		case node.IsSlice():
			slice := node.Slice()
			if len(slice) > 0 {
				if slice[0].SymbolEquals("outer") {
					io.WriteString(w, "?")
					return
				}
				// Normalize !list optimizer form back to (list ...) for stable canonical names.
				// (!list NthLocalVar(start) count expr...) encodes (list expr...) but the
				// storage slot (items[1]) varies per call site, so two identical lists would
				// get different canonical names. Strip items[1] and items[2] and use "list".
				if slice[0].SymbolEquals("!list") && len(slice) >= 3 {
					count := int(ToInt(slice[2]))
					if count == len(slice)-3 {
						io.WriteString(w, "(list")
						for _, item := range slice[3:] {
							io.WriteString(w, " ")
							enc(item)
						}
						io.WriteString(w, ")")
						return
					}
				}
			}
			io.WriteString(w, "(")
			for i, item := range slice {
				if i > 0 {
					io.WriteString(w, " ")
				}
				enc(item)
			}
			io.WriteString(w, ")")
		default:
			// Prefer tag-based decoding for special cases.
			if node.IsProc() {
				// Use pointer address to produce a unique stable name per lambda within
				// the session. Prevents YEAR, MONTH, DAY, etc. from colliding on the
				// same canonical index name.
				io.WriteString(w, fmt.Sprintf("%p", node.Proc()))
				return
			}
			if node.IsNthLocalVar() {
				i := int(node.NthLocalVar())
				if i >= 0 && i < len(columns) {
					io.WriteString(w, columns[i])
				} else {
					io.WriteString(w, "?")
				}
				return
			}
			// Native function: try to resolve declaration if present.
			if def := DeclarationForValue(node); def != nil {
				io.WriteString(w, def.Name)
				return
			}
			// Fallback unknown
			io.WriteString(w, "?")
		}
	}

	enc(v)
}

// helper that returns encoded string
func ExpressionName(v Scmer, columns []string, columnSymbols []Scmer) string {
	var b bytes.Buffer
	WriteExpressionName(v, &b, columns, columnSymbols)
	return b.String()
}
