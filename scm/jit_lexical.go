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
	strictCalls bool
	ordinary    map[Symbol]bool
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

// Metadata cannot inspect a future callable binding. Reuse the complete
// lexical traversal, additionally retaining the frame for callees whose
// ordinary invocation contract is unknown. The JIT's publication check keeps
// its existing mode; native dynamic calls have their own execution boundary.
func jitLexicalBindingsNeedRuntime(params, body Scmer) bool {
	frame := jitLexicalFrame{published: make(map[Symbol]struct{}), strictCalls: true}
	jitAddLambdaBoundParams(params, frame.published)
	return frame.needsRuntime(body, true)
}

func (frame *jitLexicalFrame) callableBinding(name Symbol) (ordinary, bound bool) {
	for current := frame; current != nil; current = current.outer {
		if _, exists := current.published[name]; exists {
			return current.ordinary[name], true
		}
	}
	return false, false
}

func (frame *jitLexicalFrame) ordinaryCallee(head Scmer) bool {
	head = head.WithoutSourceInfo()
	if head.IsProc() || head.GetTag() == tagFunc || head.IsJIT() || jitIsLambdaForm(head) {
		return true
	}
	if head.IsSymbol() {
		if ordinary, bound := frame.callableBinding(head.Symbol()); bound {
			return ordinary
		}
	}
	declaration := DeclarationForValue(head)
	return declaration != nil && !declaration.IsSpecialForm
}

func (frame *jitLexicalFrame) needsRuntime(expr Scmer, sequential bool) bool {
	expr = expr.WithoutSourceInfo()
	if !expr.IsSlice() || len(expr.Slice()) == 0 {
		return false
	}
	items := expr.Slice()
	if frame.strictCalls && items[0].WithoutSourceInfo().IsSymbol() {
		// A lexical alias must win over a same-spelled native declaration.
		// Existing syntax/free-symbol owners assume unshadowed special forms;
		// retain the whole frame when that assumption does not hold.
		head := items[0].WithoutSourceInfo()
		if _, bound := frame.callableBinding(head.Symbol()); bound {
			if declaration := DeclarationForValue(head); declaration != nil && declaration.IsSpecialForm {
				return true
			}
		}
	}
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
			branch := jitLexicalFrame{outer: frame, published: make(map[Symbol]struct{}), strictCalls: frame.strictCalls}
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
			frame = &jitLexicalFrame{outer: frame, strictCalls: frame.strictCalls}
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
		if frame.strictCalls && rhs.IsSlice() && len(rhs.Slice()) != 0 {
			head := rhs.Slice()[0].WithoutSourceInfo()
			if head.IsSymbol() {
				if _, bound := frame.callableBinding(head.Symbol()); bound {
					if declaration := DeclarationForValue(head); declaration != nil && declaration.IsSpecialForm {
						return true
					}
				}
			}
		}
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
		if frame.strictCalls {
			if frame.ordinary == nil {
				frame.ordinary = make(map[Symbol]bool)
			}
			frame.ordinary[name] = rhs.IsProc() || rhs.GetTag() == tagFunc || rhs.IsJIT() || jitIsLambdaForm(rhs)
		}
		return false
	}
	if frame.strictCalls {
		declaration := DeclarationForValue(items[0].WithoutSourceInfo())
		if declaration != nil && declaration.IsSpecialForm {
			// Forms not handled by a syntax owner or Eval's intrinsic dispatch
			// can interpret caller syntax in their declared callback.
			if declaration.evalDispatch == specialFormCall {
				return true
			}
		} else if !frame.ordinaryCallee(items[0]) {
			return true
		}
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
	if frame.strictCalls {
		inner := jitLexicalFrame{outer: frame, published: make(map[Symbol]struct{}), strictCalls: true}
		jitAddLambdaBoundParams(items[1], inner.published)
		if defining != "" {
			if _, shadowed := inner.published[defining]; !shadowed {
				inner.published[defining] = struct{}{}
				inner.ordinary = map[Symbol]bool{defining: true}
			}
		}
		return inner.needsRuntime(items[2], true)
	}
	return jitLexicalPublicationNeedsRuntime(items[1], items[2])
}
