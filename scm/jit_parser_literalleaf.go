/*
Copyright (C) 2026  Carl-Philip Hänsch

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
	"os"
)

// A value in `INSERT INTO t VALUES (1, 2, 3), ...` bottoms out at a leaf rule
// like `(define sql_number (parser (define x (regex "...")) (simplify x)))`.
// The current JIT emits, per parsed value: push a rule frame, jump to the shared
// rule body, match the regex, push the capture onto state.values, bind it into
// state.bindings, then in emitRuleReturn marshal the binding, allocate an args
// slice (jitMaterializeVirtualGoSlice), call the compiled action, and pop the
// frame through jitParserCompleteRule. The value is a pure function of the
// matched text - everything else is bookkeeping the caller never observes.
//
// analyzeLiteralLeaves marks such rules; emitRuleRef then emits, at each
// reference site, just: match the regex, run the compiled action on the matched
// text, pushValue. No frame, no bind, no memo, no alloc.
type directReturnPlan struct {
	regex  *jitRegexProgram
	action *JITEntryPoint
	desc   string
}

func (program *jitParserProgram) analyzeLiteralLeaves() {
	for r := range program.rules {
		rule := &program.rules[r]
		if rule.lexicalParent >= 0 || rule.compiledAction == nil ||
			len(rule.bindings) != 1 || len(rule.actionCaptures) != 0 {
			continue
		}
		node := rule.root
		if node != nil && node.kind == jitParserSequence && len(node.children) == 1 {
			node = node.children[0]
		}
		if node == nil || node.kind != jitParserBind || node.binding != 0 || len(node.children) != 1 {
			continue
		}
		inner := node.children[0]
		// v1: regex-bind leaves only. Atoms carry a word-boundary obligation
		// (atBreak before + after) and are a follow-up.
		if inner.kind != jitParserRegex || inner.regex == nil || inner.regex.captures != 1 {
			continue
		}
		rule.directReturn = &directReturnPlan{regex: inner.regex, action: rule.compiledAction, desc: inner.description}
		if os.Getenv("MEMCP_DUMP_LITERAL_LEAVES") != "" {
			fmt.Fprintf(os.Stderr, "[literal-leaf] rule#%d: %s\n", r, inner.description)
		}
	}
}
