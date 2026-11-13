/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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

func jitNamedDefinition(items []Scmer) (Symbol, bool) {
	if len(items) == 3 {
		if decl := DeclarationForValue(items[0].WithoutSourceInfo()); decl != nil && decl.IsSpecialForm {
			if decl.Name == "define" || decl.Name == "set" {
				return scmerSymbol(items[1])
			}
		}
	}
	return "", false
}

func jitIsLambdaForm(expr Scmer) bool {
	expr = expr.WithoutSourceInfo()
	return expr.IsSlice() && len(expr.Slice()) >= 3 && jitSyntaxKind(expr.Slice()[0]) == SyntaxLambda
}

// jitLexicalSequence returns the index of the first expression evaluated in
// the sequence's frame. begin_mut evaluates its reserve in the outer frame.
func jitLexicalSequence(head Scmer) (start int, scoped bool) {
	decl := DeclarationForValue(head.WithoutSourceInfo())
	if decl == nil || !decl.IsSpecialForm {
		return 0, false
	}
	switch decl.evalDispatch {
	case specialFormBegin:
		return 1, true
	case specialFormBeginMut:
		return 2, true
	case specialFormBangBegin:
		return 1, false
	}
	return 0, false
}

type jitLexicalFrame struct {
	outer       *jitLexicalFrame
	published   map[Symbol]struct{}
	captured    map[Symbol]struct{}
	capturedAll bool
}

func (frame *jitLexicalFrame) capture(name Symbol, ownDefinition bool) {
	for ; frame != nil; frame = frame.outer {
		if frame.captured == nil {
			frame.captured = make(map[Symbol]struct{})
		}
		frame.captured[name] = struct{}{}
		if _, bound := frame.published[name]; ownDefinition || bound {
			return
		}
	}
}

// Native named bindings use stack homes and closure captures copy values.
// Neither represents a frame location whose binding can appear or change
// after capture, or whose presence depends on a control-flow path. Let the
// interpreter own those invocations until native frames can represent them.
func jitLexicalPublicationNeedsRuntime(params, body Scmer) bool {
	frame := jitLexicalFrame{published: make(map[Symbol]struct{})}
	jitAddLambdaBoundParams(params, frame.published)
	return frame.needsRuntime(body, true)
}

func (frame *jitLexicalFrame) needsRuntime(expr Scmer, sequential bool) bool {
	expr = expr.WithoutSourceInfo()
	if !expr.IsSlice() || len(expr.Slice()) == 0 {
		return false
	}
	items := expr.Slice()
	switch jitSyntaxKind(items[0]) {
	case SyntaxQuote, SyntaxOuter, SyntaxParser:
		return false
	case SyntaxLambda:
		if len(items) < 3 {
			return false
		}
		return frame.inspectLambda(items, "")
	case SyntaxMatch:
		if len(items) > 1 && frame.needsRuntime(items[1], false) {
			return true
		}
		index := 2
		for index+1 < len(items) {
			// Pattern names shadow reads here. The interpreter publishes
			// DEFINE in the enclosing frame, so its branch matters.
			branch := jitLexicalFrame{outer: frame, published: make(map[Symbol]struct{})}
			jitAddMatchPatternBoundSymbols(items[index], branch.published)
			if branch.needsRuntime(items[index+1], false) {
				return true
			}
			index += 2
		}
		return index < len(items) && frame.needsRuntime(items[index], false)
	}
	if start, scoped := jitLexicalSequence(items[0]); start != 0 {
		if start == 2 && len(items) > 1 && frame.needsRuntime(items[1], false) {
			return true
		}
		if scoped {
			frame = &jitLexicalFrame{outer: frame}
			sequential = true
		}
		for index := start; index < len(items); index++ {
			if frame.needsRuntime(items[index], sequential) {
				return true
			}
		}
		return false
	}
	if name, defined := jitNamedDefinition(items); defined {
		_, captured := frame.captured[name]
		if !sequential || frame.capturedAll || captured {
			return true
		}
		rhs := items[2].WithoutSourceInfo()
		// A direct lambda is not observable before DEFINE returns and
		// supports a dedicated native self capture. Any other initializer
		// can publish a closure before publishing this name.
		if jitIsLambdaForm(rhs) {
			if frame.inspectLambda(rhs.Slice(), name) {
				return true
			}
		} else {
			if frame.needsRuntime(rhs, false) {
				return true
			}
			if _, captured := frame.captured[name]; frame.capturedAll || captured {
				return true
			}
		}
		if frame.published == nil {
			frame.published = make(map[Symbol]struct{})
		}
		frame.published[name] = struct{}{}
		return false
	}
	for _, item := range items {
		if frame.needsRuntime(item, false) {
			return true
		}
	}
	return false
}

func (frame *jitLexicalFrame) inspectLambda(items []Scmer, defining Symbol) bool {
	for _, name := range jitLambdaFreeSymbols(items[1], items[2]) {
		frame.capture(name, name == defining)
	}
	dynamic := jitExpressionConsumesRuntimeEnv(items[2])
	if dynamic {
		for current := frame; current != nil; current = current.outer {
			current.capturedAll = true
		}
	}
	_, namedOuter := jitLambdaOuterCaptures(items[2], !dynamic)
	for _, capture := range namedOuter {
		if capture.depth == 0 && capture.symbol == defining {
			// An explicit outer read of this definition needs the live
			// location. The native self parameter handles implicit reads.
			return true
		}
		target := frame
		for depth := 0; target != nil && depth < capture.depth; depth++ {
			target = target.outer
		}
		target.capture(capture.symbol, false)
	}
	return jitLexicalPublicationNeedsRuntime(items[1], items[2])
}
