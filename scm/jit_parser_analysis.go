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
	"os"
	"strings"
)

// Packrat memoization exists for exactly two reasons: it makes a rule that a
// backtracking grammar re-enters at the same input position return in O(1)
// instead of re-parsing, and it is the fixpoint store that Warth-style left
// recursion needs. A rule that a parse can only ever enter once per position,
// and that is not left recursive, gains nothing from a memo slot - every write
// to it is dead, and on a large input (thousands of INSERT rows) those dead
// writes and their backing arrays dominate the parser's allocations.
//
// analyzeMemoNeed is a sound, conservative static analysis over the finished
// jitParserProgram IR: it returns needMemo[rule], defaulting to true, and only
// clears it where the grammar structure proves the rule is entered at most
// once per position with success. Being wrong in the safe direction only costs
// a memo slot; being wrong the other way would reintroduce the shared-subtree
// double-parse class of bug, so every rule that any doubt touches keeps its
// memo.
//
// The result feeds both memo consumers: prepareMemoLayout only hands a dense
// memoRuleIndex slot to rules the analysis kept, which shrinks every
// per-position block, and the emitter routes a cleared rule through the
// check-free reference path so it never reads or writes the table.

type parserRuleSet []uint64

func newParserRuleSet(n int) parserRuleSet { return make(parserRuleSet, (n+63)>>6) }

func (s parserRuleSet) add(i int)      { s[i>>6] |= 1 << uint(i&63) }
func (s parserRuleSet) has(i int) bool { return s[i>>6]&(1<<uint(i&63)) != 0 }

// or merges other into s and reports whether that changed s.
func (s parserRuleSet) or(other parserRuleSet) bool {
	changed := false
	for i := range s {
		merged := s[i] | other[i]
		if merged != s[i] {
			s[i] = merged
			changed = true
		}
	}
	return changed
}

// tokenPrefix is the set of literal first tokens a node can begin with, or
// "any" when a regex, a rest match or a negative lookahead makes the first
// token unknowable. Only a fully known, pairwise disjoint set of prefixes
// across a choice's alternatives lets the analysis conclude that at most one
// alternative can consume past its first token.
type tokenPrefix struct {
	any  bool
	toks map[string]struct{}
}

func (p *tokenPrefix) markAny()          { p.any = true; p.toks = nil }
func (p *tokenPrefix) addToken(t string) {
	if p.any {
		return
	}
	if p.toks == nil {
		p.toks = make(map[string]struct{})
	}
	p.toks[t] = struct{}{}
}

// merge folds other into p and reports whether p changed.
func (p *tokenPrefix) merge(other tokenPrefix) bool {
	if p.any {
		return false
	}
	if other.any {
		p.markAny()
		return true
	}
	changed := false
	for t := range other.toks {
		if _, ok := p.toks[t]; !ok {
			p.addToken(t)
			changed = true
		}
	}
	return changed
}

type memoNeedAnalysis struct {
	program  *jitParserProgram
	n        int
	nullable []bool
	entersAt0 []parserRuleSet // rules enterable before consuming any input
	reaches   []parserRuleSet // rules reachable at any offset (transitive)
	first     []tokenPrefix
	needMemo  []bool
}

func (program *jitParserProgram) analyzeMemoNeed() []bool {
	a := &memoNeedAnalysis{program: program, n: len(program.rules)}
	a.nullable = make([]bool, a.n)
	a.entersAt0 = make([]parserRuleSet, a.n)
	a.reaches = make([]parserRuleSet, a.n)
	a.first = make([]tokenPrefix, a.n)
	a.needMemo = make([]bool, a.n)
	for r := 0; r < a.n; r++ {
		a.entersAt0[r] = newParserRuleSet(a.n)
		a.reaches[r] = newParserRuleSet(a.n)
	}

	a.computeNullable()
	a.computeReaches()
	a.computeEntersAt0()
	a.computeFirst()

	for r := 0; r < a.n; r++ {
		// Direct or indirect left recursion is the one case the memo table is
		// load bearing for correctness, never just speed.
		a.needMemo[r] = a.entersAt0[r].has(r)
	}
	a.flagChoicesAndRepeats()
	a.propagateToReferrers()

	if os.Getenv("MEMCP_DUMP_MEMO_ANALYSIS") != "" {
		a.dump()
	}
	return a.needMemo
}

func (a *memoNeedAnalysis) computeNullable() {
	for changed := true; changed; {
		changed = false
		for r := 0; r < a.n; r++ {
			if a.program.rules[r].root == nil {
				continue
			}
			if v := a.nodeNullable(a.program.rules[r].root); v && !a.nullable[r] {
				a.nullable[r] = true
				changed = true
			}
		}
	}
}

func (a *memoNeedAnalysis) nodeNullable(node *jitParserNode) bool {
	switch node.kind {
	case jitParserAtom:
		return false
	case jitParserRegex:
		return true // a regex may match the empty string; over-approximate
	case jitParserEnd, jitParserEmpty, jitParserRest:
		return true
	case jitParserRuleRef:
		return a.nullable[node.rule]
	case jitParserSequence:
		for _, c := range node.children {
			if !a.nodeNullable(c) {
				return false
			}
		}
		return true
	case jitParserChoice:
		for _, c := range node.children {
			if a.nodeNullable(c) {
				return true
			}
		}
		return false
	case jitParserZeroOrMore, jitParserOptional:
		return true
	case jitParserOneOrMore, jitParserBind, jitParserCapture:
		return a.nodeNullable(node.children[0])
	case jitParserExclude:
		if len(node.children) == 0 {
			return true
		}
		return a.nodeNullable(node.children[0])
	}
	return true
}

func (a *memoNeedAnalysis) computeReaches() {
	for changed := true; changed; {
		changed = false
		for r := 0; r < a.n; r++ {
			if a.program.rules[r].root == nil {
				continue
			}
			if a.collectReaches(a.program.rules[r].root, a.reaches[r]) {
				changed = true
			}
		}
	}
}

// collectReaches ORs every rule reachable from node into out and reports change.
func (a *memoNeedAnalysis) collectReaches(node *jitParserNode, out parserRuleSet) bool {
	changed := false
	switch node.kind {
	case jitParserRuleRef:
		if !out.has(node.rule) {
			out.add(node.rule)
			changed = true
		}
		if out.or(a.reaches[node.rule]) {
			changed = true
		}
	}
	for _, c := range node.children {
		if a.collectReaches(c, out) {
			changed = true
		}
	}
	return changed
}

// subtreeReaches returns a fresh set of every rule reachable from node.
func (a *memoNeedAnalysis) subtreeReaches(node *jitParserNode) parserRuleSet {
	out := newParserRuleSet(a.n)
	a.collectReaches(node, out)
	return out
}

func (a *memoNeedAnalysis) computeEntersAt0() {
	for changed := true; changed; {
		changed = false
		for r := 0; r < a.n; r++ {
			if a.program.rules[r].root == nil {
				continue
			}
			if a.entersAt0[r].or(a.nodeEntersAt0(a.program.rules[r].root)) {
				changed = true
			}
		}
	}
}

// nodeEntersAt0 returns a fresh set of rules node can enter before it has
// consumed a single byte of input.
func (a *memoNeedAnalysis) nodeEntersAt0(node *jitParserNode) parserRuleSet {
	out := newParserRuleSet(a.n)
	switch node.kind {
	case jitParserRuleRef:
		out.add(node.rule)
		out.or(a.entersAt0[node.rule])
	case jitParserSequence:
		for _, c := range node.children {
			out.or(a.nodeEntersAt0(c))
			if !a.nodeNullable(c) {
				break
			}
		}
	case jitParserChoice, jitParserExclude:
		for _, c := range node.children {
			out.or(a.nodeEntersAt0(c))
		}
	case jitParserZeroOrMore, jitParserOneOrMore, jitParserOptional, jitParserBind, jitParserCapture:
		out.or(a.nodeEntersAt0(node.children[0]))
	}
	return out
}

func (a *memoNeedAnalysis) computeFirst() {
	for changed := true; changed; {
		changed = false
		for r := 0; r < a.n; r++ {
			if a.program.rules[r].root == nil {
				continue
			}
			if a.first[r].merge(a.nodeFirst(a.program.rules[r].root)) {
				changed = true
			}
		}
	}
}

func (a *memoNeedAnalysis) nodeFirst(node *jitParserNode) tokenPrefix {
	var out tokenPrefix
	switch node.kind {
	case jitParserAtom:
		if node.description == "" {
			out.markAny()
		} else {
			out.addToken(strings.ToLower(node.description))
		}
	case jitParserRegex, jitParserRest, jitParserExclude:
		out.markAny()
	case jitParserEnd, jitParserEmpty:
		// contribute nothing, nullable cascade handled by the caller
	case jitParserRuleRef:
		out.merge(a.first[node.rule])
	case jitParserSequence:
		for _, c := range node.children {
			out.merge(a.nodeFirst(c))
			if !a.nodeNullable(c) {
				break
			}
		}
	case jitParserChoice:
		for _, c := range node.children {
			out.merge(a.nodeFirst(c))
		}
	case jitParserZeroOrMore, jitParserOneOrMore, jitParserOptional, jitParserBind, jitParserCapture:
		out.merge(a.nodeFirst(node.children[0]))
	}
	return out
}

// flagChoicesAndRepeats walks every rule body and marks a rule as needing a
// memo whenever the structure admits a same-position re-entry the memo would
// otherwise absorb.
func (a *memoNeedAnalysis) flagChoicesAndRepeats() {
	for r := 0; r < a.n; r++ {
		if a.program.rules[r].root != nil {
			a.walkForFlags(a.program.rules[r].root)
		}
	}
}

func (a *memoNeedAnalysis) walkForFlags(node *jitParserNode) {
	switch node.kind {
	case jitParserChoice:
		a.flagChoice(node)
	case jitParserSequence:
		a.flagRepeatSiblings(node)
	}
	for _, c := range node.children {
		a.walkForFlags(c)
	}
}

// flagChoice marks q as needing a memo when two or more alternatives of the
// choice can reach q, unless the alternatives are guaranteed to be mutually
// exclusive by a fully known, pairwise disjoint set of first tokens and none
// of them can enter q before consuming that first token. In that case exactly
// one alternative ever proceeds past its first token, so q is entered once.
func (a *memoNeedAnalysis) flagChoice(node *jitParserNode) {
	alts := node.children
	if len(alts) < 2 {
		return
	}
	disjoint := true
	owner := make(map[string]int)
	for i, alt := range alts {
		fp := a.nodeFirst(alt)
		if fp.any {
			disjoint = false
			break
		}
		for t := range fp.toks {
			if prev, ok := owner[t]; ok && prev != i {
				disjoint = false
			}
			owner[t] = i
		}
	}
	altReach := make([]parserRuleSet, len(alts))
	altEnter := make([]parserRuleSet, len(alts))
	for i, alt := range alts {
		altReach[i] = a.subtreeReaches(alt)
		altEnter[i] = a.nodeEntersAt0(alt)
	}
	for q := 0; q < a.n; q++ {
		hits := 0
		for i := range alts {
			if altReach[i].has(q) {
				hits++
			}
		}
		if hits < 2 {
			continue
		}
		safe := disjoint
		if safe {
			for i := range alts {
				if altEnter[i].has(q) {
					safe = false
					break
				}
			}
		}
		if !safe {
			a.needMemo[q] = true
		}
	}
}

// flagRepeatSiblings handles the one non-choice re-entry: the failing final
// iteration of a *​/+ can leave an inner rule succeeded at position P, and a
// following sibling that can enter that same rule at offset 0 then parses it
// again at P.
func (a *memoNeedAnalysis) flagRepeatSiblings(seq *jitParserNode) {
	for i, child := range seq.children {
		if child.kind != jitParserZeroOrMore && child.kind != jitParserOneOrMore {
			continue
		}
		follow := newParserRuleSet(a.n)
		for j := i + 1; j < len(seq.children); j++ {
			follow.or(a.nodeEntersAt0(seq.children[j]))
			if !a.nodeNullable(seq.children[j]) {
				break
			}
		}
		body := a.subtreeReaches(child)
		for q := 0; q < a.n; q++ {
			if body.has(q) && follow.has(q) {
				a.needMemo[q] = true
			}
		}
	}
}

// propagateToReferrers closes the analysis under the obvious implication: if a
// rule that references q can itself be entered more than once at a position,
// so can q.
func (a *memoNeedAnalysis) propagateToReferrers() {
	referrers := make([]parserRuleSet, a.n)
	for q := 0; q < a.n; q++ {
		referrers[q] = newParserRuleSet(a.n)
	}
	for r := 0; r < a.n; r++ {
		if a.program.rules[r].root == nil {
			continue
		}
		direct := newParserRuleSet(a.n)
		a.collectDirectRefs(a.program.rules[r].root, direct)
		for q := 0; q < a.n; q++ {
			if direct.has(q) {
				referrers[q].add(r)
			}
		}
	}
	for changed := true; changed; {
		changed = false
		for q := 0; q < a.n; q++ {
			if a.needMemo[q] {
				continue
			}
			for r := 0; r < a.n; r++ {
				if referrers[q].has(r) && a.needMemo[r] {
					a.needMemo[q] = true
					changed = true
					break
				}
			}
		}
	}
}

func (a *memoNeedAnalysis) collectDirectRefs(node *jitParserNode, out parserRuleSet) {
	if node.kind == jitParserRuleRef {
		out.add(node.rule)
	}
	for _, c := range node.children {
		a.collectDirectRefs(c, out)
	}
}

func (a *memoNeedAnalysis) dump() {
	kept, freed := 0, 0
	var freedNames []string
	for r := 0; r < a.n; r++ {
		if a.program.rules[r].lexicalParent >= 0 {
			continue
		}
		if a.needMemo[r] {
			kept++
		} else {
			freed++
			if len(freedNames) < 60 {
				freedNames = append(freedNames, ruleDisplayName(&a.program.rules[r], r))
			}
		}
	}
	os.Stderr.WriteString("[memo-analysis] non-lexical rules: kept(memoized)=" +
		itoa(kept) + " freed(no memo)=" + itoa(freed) + "\n")
	os.Stderr.WriteString("[memo-analysis] freed sample: " + strings.Join(freedNames, ", ") + "\n")
}

func ruleDisplayName(rule *jitParserRule, id int) string {
	if rule.root != nil && rule.root.description != "" {
		return rule.root.description
	}
	return "rule#" + itoa(id)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [20]byte
	p := len(b)
	for i > 0 {
		p--
		b[p] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		p--
		b[p] = '-'
	}
	return string(b[p:])
}
