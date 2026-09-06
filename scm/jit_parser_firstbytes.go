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
	"regexp/syntax"
	"unicode"
)

// A parser choice is normally emitted as a linear cascade: every alternative
// pushes a checkpoint, runs its match, and on failure records a diagnostic and
// restores. Parsing a literal through sql_expression6's ~40 alternatives means
// ~38 failed keyword matches, each a handful of Go helper calls. When every
// alternative's first consumed byte is statically known and no alternative can
// match empty, the emitter can instead read that one byte and jump straight to
// the (usually one) alternative that could match it. firstByteSet is the
// per-node/per-rule set that drives that decision.
type firstByteSet struct {
	bits [4]uint64
	any  bool
}

func (s *firstByteSet) add(b byte) {
	if !s.any {
		s.bits[b>>6] |= 1 << (b & 63)
	}
}

func (s *firstByteSet) addRange(lo, hi int) {
	for b := lo; b <= hi && b < 256; b++ {
		s.add(byte(b))
	}
}

func (s *firstByteSet) markAny() {
	s.any = true
	s.bits = [4]uint64{}
}

func (s firstByteSet) has(b byte) bool {
	return s.any || s.bits[b>>6]&(1<<(b&63)) != 0
}

func (s firstByteSet) empty() bool {
	return !s.any && s.bits == [4]uint64{}
}

// union folds other into s and reports whether s changed.
func (s *firstByteSet) union(other firstByteSet) bool {
	if s.any {
		return false
	}
	if other.any {
		s.markAny()
		return true
	}
	changed := false
	for i := range s.bits {
		merged := s.bits[i] | other.bits[i]
		if merged != s.bits[i] {
			s.bits[i] = merged
			changed = true
		}
	}
	return changed
}

// computeFirstBytes fills program.ruleFirstBytes and program.ruleNullable by
// fixpoint. Both are read afterwards, per node, by the emitter.
func (program *jitParserProgram) computeFirstBytes() {
	n := len(program.rules)
	program.ruleFirstBytes = make([]firstByteSet, n)
	program.ruleNullable = make([]bool, n)

	for changed := true; changed; {
		changed = false
		for r := 0; r < n; r++ {
			root := program.rules[r].root
			if root == nil {
				continue
			}
			if program.nodeNullableFB(root) && !program.ruleNullable[r] {
				program.ruleNullable[r] = true
				changed = true
			}
			if program.ruleFirstBytes[r].union(program.nodeFirstBytes(root)) {
				changed = true
			}
		}
	}

	if os.Getenv("MEMCP_DUMP_DISPATCH") != "" {
		program.dumpDispatch()
	}
}

// choiceDispatch is the plan for emitting a choice as a byte-indexed jump
// instead of a linear cascade. buckets[b] lists, in original order, the
// alternatives that could match when the next byte is b (a subset that always
// includes every "wild" alternative). wild lists alternatives whose leading
// byte is unknowable or which can match empty - they must be tried for every
// byte. useful is set only when the dispatch meaningfully shrinks the worst
// cascade.
type choiceDispatch struct {
	buckets [256][]int
	wild    []int
	useful  bool
}

func (program *jitParserProgram) choiceDispatchPlan(node *jitParserNode) choiceDispatch {
	var d choiceDispatch
	if program.ruleFirstBytes == nil || node.kind != jitParserChoice || len(node.children) < 6 {
		return d
	}
	fbs := make([]firstByteSet, len(node.children))
	wildAt := make([]bool, len(node.children))
	concrete := 0
	for i, alt := range node.children {
		f := program.nodeFirstBytes(alt)
		if program.nodeNullableFB(alt) || f.any || f.empty() {
			wildAt[i] = true
			d.wild = append(d.wild, i)
			continue
		}
		fbs[i] = f
		concrete++
	}
	if concrete*3 < len(node.children)*2 { // need most alternatives concrete
		return d
	}
	worst := 0
	for b := 0; b < 256; b++ {
		var list []int
		for i := range node.children {
			if wildAt[i] || fbs[i].has(byte(b)) {
				list = append(list, i)
			}
		}
		d.buckets[b] = list
		if len(list) > worst {
			worst = len(list)
		}
	}
	d.useful = worst*2 < len(node.children)
	return d
}

func (program *jitParserProgram) dumpDispatch() {
	var walk func(*jitParserNode, int)
	total, dispatched, altsBefore, altsWorst := 0, 0, 0, 0
	walk = func(node *jitParserNode, rule int) {
		if node.kind == jitParserChoice && len(node.children) >= 2 {
			total++
			if d := program.choiceDispatchPlan(node); d.useful {
				dispatched++
				worst := 0
				for b := 0; b < 256; b++ {
					if len(d.buckets[b]) > worst {
						worst = len(d.buckets[b])
					}
				}
				altsBefore += len(node.children)
				altsWorst += worst
				fmt.Fprintf(os.Stderr, "[dispatch] rule#%d: %d alts (%d wild) -> worst bucket %d\n",
					rule, len(node.children), len(d.wild), worst)
			}
		}
		for _, c := range node.children {
			walk(c, rule)
		}
	}
	for r := range program.rules {
		if program.rules[r].root != nil {
			walk(program.rules[r].root, r)
		}
	}
	fmt.Fprintf(os.Stderr, "[dispatch] %d/%d choices dispatched; summed alternatives tried per hit: %d -> %d\n",
		dispatched, total, altsBefore, altsWorst)
}

func (program *jitParserProgram) nodeNullableFB(node *jitParserNode) bool {
	switch node.kind {
	case jitParserAtom:
		return false
	case jitParserRegex:
		return regexpNullable(node.regex.root)
	case jitParserEnd, jitParserEmpty, jitParserRest:
		return true
	case jitParserRuleRef:
		return program.ruleNullable[node.rule]
	case jitParserSequence:
		for _, c := range node.children {
			if !program.nodeNullableFB(c) {
				return false
			}
		}
		return true
	case jitParserChoice:
		for _, c := range node.children {
			if program.nodeNullableFB(c) {
				return true
			}
		}
		return false
	case jitParserZeroOrMore, jitParserOptional:
		return true
	case jitParserOneOrMore, jitParserBind, jitParserCapture:
		return program.nodeNullableFB(node.children[0])
	case jitParserExclude:
		if len(node.children) == 0 {
			return true
		}
		return program.nodeNullableFB(node.children[0])
	}
	return true
}

// nodeFirstBytes returns the set of bytes node can begin a match with. `any`
// means unknowable (a regex the analysis does not model, a rest match, a
// negative lookahead).
func (program *jitParserProgram) nodeFirstBytes(node *jitParserNode) firstByteSet {
	var out firstByteSet
	switch node.kind {
	case jitParserAtom, jitParserRegex:
		fbs, _ := regexpFirstBytes(node.regex.root)
		out.union(fbs)
	case jitParserRest, jitParserExclude:
		out.markAny()
	case jitParserEnd, jitParserEmpty:
		// zero width; contributes nothing, nullable cascade handled by caller
	case jitParserRuleRef:
		out.union(program.ruleFirstBytes[node.rule])
	case jitParserSequence:
		for _, c := range node.children {
			out.union(program.nodeFirstBytes(c))
			if !program.nodeNullableFB(c) {
				break
			}
		}
	case jitParserChoice:
		for _, c := range node.children {
			out.union(program.nodeFirstBytes(c))
		}
	case jitParserZeroOrMore, jitParserOneOrMore, jitParserOptional, jitParserBind, jitParserCapture:
		out.union(program.nodeFirstBytes(node.children[0]))
	}
	return out
}

// regexpNullable reports whether an anchored grammar regex can match the empty
// string.
func regexpNullable(re *syntax.Regexp) bool {
	switch re.Op {
	case syntax.OpEmptyMatch, syntax.OpBeginText, syntax.OpEndText,
		syntax.OpBeginLine, syntax.OpEndLine, syntax.OpWordBoundary,
		syntax.OpNoWordBoundary, syntax.OpStar, syntax.OpQuest:
		return true
	case syntax.OpLiteral:
		return len(re.Rune) == 0
	case syntax.OpCapture, syntax.OpPlus:
		return regexpNullable(re.Sub[0])
	case syntax.OpConcat:
		for _, sub := range re.Sub {
			if !regexpNullable(sub) {
				return false
			}
		}
		return true
	case syntax.OpAlternate:
		for _, sub := range re.Sub {
			if regexpNullable(sub) {
				return true
			}
		}
		return false
	}
	return false
}

// regexpFirstBytes returns the leading-byte set of an anchored grammar regex
// and whether it can also match empty. `any` in the set means the shape is not
// modelled (e.g. a Unicode class the byte walk would misrepresent).
func regexpFirstBytes(re *syntax.Regexp) (firstByteSet, bool) {
	var out firstByteSet
	switch re.Op {
	case syntax.OpEmptyMatch, syntax.OpBeginText, syntax.OpEndText,
		syntax.OpBeginLine, syntax.OpEndLine, syntax.OpWordBoundary,
		syntax.OpNoWordBoundary:
		return out, true

	case syntax.OpLiteral:
		if len(re.Rune) == 0 {
			return out, true
		}
		addRuneFirstBytes(&out, re.Rune[0], re.Flags&syntax.FoldCase != 0)
		return out, false

	case syntax.OpCharClass:
		for i := 0; i+1 < len(re.Rune); i += 2 {
			lo, hi := re.Rune[i], re.Rune[i+1]
			if lo > 0x7f {
				out.markAny() // multi-byte lead bytes; do not model
				return out, false
			}
			if hi > 0x7f {
				hi = 0x7f
				out.markAny()
			}
			out.addRange(int(lo), int(hi))
		}
		return out, false

	case syntax.OpAnyChar, syntax.OpAnyCharNotNL:
		out.markAny()
		return out, false

	case syntax.OpCapture:
		return regexpFirstBytes(re.Sub[0])

	case syntax.OpStar, syntax.OpQuest:
		fbs, _ := regexpFirstBytes(re.Sub[0])
		out.union(fbs)
		return out, true

	case syntax.OpPlus:
		return regexpFirstBytes(re.Sub[0])

	case syntax.OpConcat:
		nullable := true
		for _, sub := range re.Sub {
			fbs, subNull := regexpFirstBytes(sub)
			out.union(fbs)
			if !subNull {
				nullable = false
				break
			}
		}
		return out, nullable

	case syntax.OpAlternate:
		nullable := false
		for _, sub := range re.Sub {
			fbs, subNull := regexpFirstBytes(sub)
			out.union(fbs)
			nullable = nullable || subNull
		}
		return out, nullable
	}
	out.markAny()
	return out, false
}

func addRuneFirstBytes(out *firstByteSet, r rune, fold bool) {
	addOne := func(x rune) {
		if x < 0x80 {
			out.add(byte(x))
		} else {
			// first byte of a multi-byte UTF-8 sequence
			var buf [4]byte
			buf[0] = byte(0xC0 | (x >> 6))
			if x >= 0x800 {
				buf[0] = byte(0xE0 | (x >> 12))
			}
			if x >= 0x10000 {
				buf[0] = byte(0xF0 | (x >> 18))
			}
			out.add(buf[0])
		}
	}
	addOne(r)
	if fold {
		for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
			addOne(f)
		}
	}
}
