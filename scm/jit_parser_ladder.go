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
	"sort"
	"strings"
)

// A precedence ladder is a chain of rules X0 -> X1 -> ... -> Xk where each Xi's
// root is a jitParserChoice whose LAST alternative is a bare jitParserRuleRef to
// X{i+1}, X{i+1} is referenced only by Xi, and Xi has no explicit generator
// (identity). In tail-call terms the "no operator here" branch of Xi is a tail
// call to the next tighter precedence level; the other alternative(s) carry the
// operator productions (the non-tail, folding case).
//
// detectPrecedenceLadders finds these chains as read-only analysis. The intent
// is for the emitter to later lower a whole chain as one fallthrough block with
// a shared frame and a single memo slot, instead of k framed, memoized rule
// invocations for every bare primary. This file only detects + dumps them.

type ladderLevel struct {
	rule    int
	opAlts  []*jitParserNode // non-tail alternatives (operator productions)
	tailRef *jitParserNode   // the jitParserRuleRef to the next level
}

type precedenceLadder struct {
	levels []ladderLevel // X0 .. X{k-1}
	leaf   int           // Xk: the primary (first rule that is not a ladder link)
}

func (program *jitParserProgram) ruleRefCounts() []int {
	counts := make([]int, len(program.rules))
	var walk func(n *jitParserNode)
	walk = func(n *jitParserNode) {
		if n == nil {
			return
		}
		if n.kind == jitParserRuleRef && n.rule >= 0 && n.rule < len(counts) {
			counts[n.rule]++
		}
		for _, c := range n.children {
			walk(c)
		}
	}
	for i := range program.rules {
		walk(program.rules[i].root)
	}
	return counts
}

// ruleReferrers[r] is the set of rule ids whose root contains a ruleref to r.
func (program *jitParserProgram) ruleReferrers() []map[int]bool {
	out := make([]map[int]bool, len(program.rules))
	for i := range out {
		out[i] = map[int]bool{}
	}
	var walk func(from int, n *jitParserNode)
	walk = func(from int, n *jitParserNode) {
		if n == nil {
			return
		}
		if n.kind == jitParserRuleRef && n.rule >= 0 && n.rule < len(out) {
			out[n.rule][from] = true
		}
		for _, c := range n.children {
			walk(from, c)
		}
	}
	for i := range program.rules {
		walk(i, program.rules[i].root)
	}
	return out
}

// rulesReachableFromNode collects every rule id reachable from n by following
// rulerefs and their roots transitively.
func (program *jitParserProgram) rulesReachableFromNodes(nodes []*jitParserNode) map[int]bool {
	seen := map[int]bool{}
	var stack []*jitParserNode
	stack = append(stack, nodes...)
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if n == nil {
			continue
		}
		if n.kind == jitParserRuleRef && n.rule >= 0 && n.rule < len(program.rules) {
			if !seen[n.rule] {
				seen[n.rule] = true
				if root := program.rules[n.rule].root; root != nil {
					stack = append(stack, root)
				}
			}
		}
		stack = append(stack, n.children...)
	}
	return seen
}

// ladderLinkTarget reports whether rule r's root is a choice whose last child is
// a bare ruleref and whose generator is identity, i.e. the tail-call shape.
func (program *jitParserProgram) ladderLinkTarget(r int) (target int, opAlts []*jitParserNode, tailRef *jitParserNode, ok bool) {
	if !program.rules[r].generator.IsNil() {
		return 0, nil, nil, false // a real generator wraps the result -> not a tail call
	}
	root := program.rules[r].root
	if root == nil || root.kind != jitParserChoice || len(root.children) < 2 {
		return 0, nil, nil, false
	}
	last := root.children[len(root.children)-1]
	if last.kind != jitParserRuleRef {
		return 0, nil, nil, false
	}
	return last.rule, root.children[:len(root.children)-1], last, true
}

func (program *jitParserProgram) detectPrecedenceLadders() []precedenceLadder {
	referrers := program.ruleReferrers()
	inChain := make([]bool, len(program.rules))
	var out []precedenceLadder

	for start := range program.rules {
		if inChain[start] {
			continue
		}
		if _, _, _, ok := program.ladderLinkTarget(start); !ok {
			continue
		}
		// A rule is only a private ladder link if every ruleref to it comes from
		// a level already in the chain or from that level's operator-alternative
		// rule closure (the op-alts parse the tail as their lhs/rhs - that is the
		// speculative tail call, not an outside consumer).
		lad := precedenceLadder{}
		ownership := map[int]bool{start: true}
		cur, curOp, curTail := start, []*jitParserNode(nil), (*jitParserNode)(nil)
		_, curOp, curTail, _ = program.ladderLinkTarget(start)
		for {
			// the immediate operator-alternative rules of this level legitimately
			// reference the next level as their lhs/rhs operand
			for _, alt := range curOp {
				if alt.kind == jitParserRuleRef {
					ownership[alt.rule] = true
				}
			}
			lad.levels = append(lad.levels, ladderLevel{rule: cur, opAlts: curOp, tailRef: curTail})
			inChain[cur] = true
			next := curTail.rule
			outsideRef := false
			for from := range referrers[next] {
				if !ownership[from] {
					outsideRef = true
					break
				}
			}
			t2, o2, tr2, ok2 := program.ladderLinkTarget(next)
			_ = t2
			if !outsideRef && ok2 && !inChain[next] {
				ownership[next] = true
				cur, curOp, curTail = next, o2, tr2
				continue
			}
			lad.leaf = next
			break
		}
		if len(lad.levels) >= 2 {
			out = append(out, lad)
		} else {
			for _, lv := range lad.levels {
				inChain[lv.rule] = false
			}
		}
	}
	return out
}

// jitLadderFastPathEnabled gates the precedence-ladder speculative-descent
// emission. On by default; set MEMCP_LADDER_FASTPATH=0 for an A/B baseline.
func jitLadderFastPathEnabled() bool {
	return os.Getenv("MEMCP_LADDER_FASTPATH") != "0"
}

// foldSeqBridgeTarget recognises the left-assoc fold precedence level shape
//
//	'((define a LOWER) (define terms (* (op LOWER) ...))) (reduce terms FOLD a)
//
// i.e. a two-child sequence whose first child binds a ruleref and whose second
// binds a repeat, wrapped by a `reduce`/`reduce2` generator seeded with the
// first bind. Such a rule returns the first operand unchanged when the repeat
// matches nothing, so the ladder descent may pass straight through it. Returns
// the lower level's rule id.
func (program *jitParserProgram) foldSeqBridgeTarget(r int) (int, bool) {
	gen := program.rules[r].generator
	if gen.IsNil() || !gen.IsSlice() {
		return 0, false
	}
	head := gen.Slice()
	if len(head) == 0 || !(scmerIsSymbol(head[0], "reduce") || scmerIsSymbol(head[0], "reduce2")) {
		return 0, false
	}
	root := program.rules[r].root
	if root == nil || root.kind != jitParserSequence || len(root.children) != 2 {
		return 0, false
	}
	first := root.children[0]
	for first != nil && (first.kind == jitParserBind || first.kind == jitParserCapture) && len(first.children) == 1 {
		first = first.children[0]
	}
	if first == nil || first.kind != jitParserRuleRef {
		return 0, false
	}
	second := root.children[1]
	for second != nil && (second.kind == jitParserBind || second.kind == jitParserCapture) && len(second.children) == 1 {
		second = second.children[0]
	}
	if second == nil || (second.kind != jitParserZeroOrMore && second.kind != jitParserOneOrMore) {
		return 0, false
	}
	return first.rule, true
}

// altIsAtomLed reports whether an alternative starts by matching a literal token
// or a leaf (regex / direct-return) rule rather than by recursing into another
// composite rule. Primary/"value" choices are overwhelmingly atom-led; operator
// levels start by parsing their lower operand.
func (program *jitParserProgram) altIsAtomLed(n *jitParserNode) bool {
	seen := map[int]bool{}
	for depth := 0; n != nil && depth < 12; depth++ {
		switch n.kind {
		case jitParserAtom, jitParserRegex:
			return true
		case jitParserRuleRef:
			if n.rule < 0 || n.rule >= len(program.rules) {
				return false
			}
			rr := program.rules[n.rule]
			if rr.directReturn != nil {
				return true
			}
			if rr.root != nil && rr.root.kind == jitParserRegex {
				return true
			}
			if seen[n.rule] {
				return false
			}
			seen[n.rule] = true
			n = rr.root
		case jitParserSequence, jitParserBind, jitParserCapture, jitParserExclude,
			jitParserOptional, jitParserChoice:
			if len(n.children) == 0 {
				return false
			}
			n = n.children[0]
		default:
			return false
		}
	}
	return false
}

// isPrimaryLevel reports whether rule r is the "primary" of a precedence ladder
// (a literal / paren / function-call choice) as opposed to an operator level.
func (program *jitParserProgram) isPrimaryLevel(r int) bool {
	root := program.rules[r].root
	if root == nil || root.kind != jitParserChoice {
		return false
	}
	alts := root.children
	if len(alts) > 1 && alts[len(alts)-1].kind == jitParserRuleRef {
		alts = alts[:len(alts)-1] // drop the tail ruleref
	}
	if len(alts) < 8 {
		return false
	}
	atomLed := 0
	for _, a := range alts {
		if program.altIsAtomLed(a) {
			atomLed++
		}
	}
	return atomLed*2 >= len(alts)
}

// computeLadderFastPaths maps every interior level of a precedence ladder to the
// primary rule the emitter may speculatively descend to: for a bare primary
// (no operator follows) every rule between the level and the primary is
// identity, so the primary's result IS the level's result. detectPrecedenceLadders
// stops at Choice/Sequence boundaries; this walk bridges the fold-sequence
// levels (sql_expression3/4) too and runs the chain down to the primary.
func (program *jitParserProgram) computeLadderFastPaths() map[int]int {
	out := map[int]int{}
	for _, lad := range program.detectPrecedenceLadders() {
		chain := make([]int, 0, 8)
		visited := map[int]bool{}
		cur := lad.levels[0].rule
		target := -1
		for {
			if cur < 0 || cur >= len(program.rules) || visited[cur] {
				break
			}
			visited[cur] = true
			if program.isPrimaryLevel(cur) {
				target = cur
				break
			}
			next := -1
			if _, _, tr, ok := program.ladderLinkTarget(cur); ok {
				next = tr.rule
			} else if b, ok := program.foldSeqBridgeTarget(cur); ok {
				next = b
			} else {
				target = cur // identity is preserved down to here; stop
				break
			}
			chain = append(chain, cur)
			cur = next
		}
		if target < 0 || len(chain) < 2 {
			continue
		}
		for _, r := range chain {
			if _, exists := out[r]; !exists {
				out[r] = target
			}
		}
	}
	return out
}

// primaryDirectReturnLeaves finds the direct-return literal-leaf rules that are
// immediate alternatives of a ladder primary: [0] the number leaf (gate class
// 1, first byte a digit), [1] the single-quote string leaf (gate class 2).
// Each is -1 when absent. These #776 leaves are regex + one action call, no
// frame, no memo - the fast path matches one directly for a digit/quote-led
// value, skipping the primary rule's own frame + memo entirely.
func (program *jitParserProgram) primaryDirectReturnLeaves(target int) [2]int {
	out := [2]int{-1, -1}
	root := program.rules[target].root
	if root == nil || root.kind != jitParserChoice || program.ruleFirstBytes == nil {
		return out
	}
	for _, c := range root.children {
		if c.kind != jitParserRuleRef || c.rule < 0 || c.rule >= len(program.rules) ||
			program.rules[c.rule].directReturn == nil {
			continue
		}
		fb := program.ruleFirstBytes[c.rule]
		if out[1] < 0 && fb.has('\'') {
			out[1] = c.rule
		}
		// a plain decimal number leaf accepts every digit but not "0x..." only
		if out[0] < 0 && fb.has('1') && fb.has('9') && !fb.has('\'') {
			out[0] = c.rule
		}
	}
	return out
}

// ruleNames does a best-effort reverse lookup of ruleID -> grammar name via the
// parser objects registered during the build and the global environment.
func (program *jitParserProgram) ruleNames() map[int]string {
	names := map[int]string{}
	byParser := map[*ScmParser]int{}
	for p, id := range program.parserRule {
		byParser[p] = id
	}
	for sym, val := range Globalenv.Vars {
		if val.IsParser() {
			if id, ok := byParser[val.Parser()]; ok {
				names[id] = string(sym)
			}
		}
	}
	return names
}

// renderNode renders a node's operator shape. Unnamed (anonymous sub-choice)
// rulerefs are inlined so a reader sees the actual atoms/operators a ladder
// level carries; named rules are shown by name and not expanded.
func (program *jitParserProgram) renderNode(n *jitParserNode, names map[int]string, depth int) string {
	if n == nil || depth > 6 {
		return "…"
	}
	switch n.kind {
	case jitParserAtom:
		return fmt.Sprintf("%q", n.description)
	case jitParserRegex:
		return "/" + n.description + "/"
	case jitParserRuleRef:
		if nm, ok := names[n.rule]; ok {
			return nm
		}
		if n.rule >= 0 && n.rule < len(program.rules) {
			return program.renderNode(program.rules[n.rule].root, names, depth+1)
		}
		return fmt.Sprintf("rule#%d", n.rule)
	case jitParserBind:
		return "def(" + program.renderNode(n.children[0], names, depth+1) + ")"
	case jitParserCapture:
		return "cap(" + program.renderNode(n.children[0], names, depth+1) + ")"
	case jitParserSequence:
		parts := make([]string, 0, len(n.children))
		for _, c := range n.children {
			parts = append(parts, program.renderNode(c, names, depth+1))
		}
		return "[" + strings.Join(parts, " ") + "]"
	case jitParserChoice:
		parts := make([]string, 0, len(n.children))
		for _, c := range n.children {
			parts = append(parts, program.renderNode(c, names, depth+1))
		}
		return "(" + strings.Join(parts, " | ") + ")"
	case jitParserZeroOrMore:
		return "*(" + program.renderNode(n.children[0], names, depth+1) + ")"
	case jitParserOneOrMore:
		return "+(" + program.renderNode(n.children[0], names, depth+1) + ")"
	case jitParserOptional:
		return "?(" + program.renderNode(n.children[0], names, depth+1) + ")"
	case jitParserExclude:
		return "!(" + program.renderNode(n.children[0], names, depth+1) + ")"
	default:
		return fmt.Sprintf("<kind %d>", n.kind)
	}
}

var dumpedLadderSignatures = map[string]bool{}

func (program *jitParserProgram) dumpPrecedenceLadders() {
	ladders := program.detectPrecedenceLadders()
	names := program.ruleNames()
	refs := program.ruleRefCounts()
	name := func(id int) string {
		if nm, ok := names[id]; ok {
			return nm
		}
		return fmt.Sprintf("rule#%d", id)
	}
	sort.Slice(ladders, func(i, j int) bool { return len(ladders[i].levels) > len(ladders[j].levels) })

	var b strings.Builder
	fmt.Fprintf(&b, "[ladder] %d precedence ladder(s) among %d rules\n", len(ladders), len(program.rules))
	for _, lad := range ladders {
		fmt.Fprintf(&b, "[ladder] start=%s  (%d detected levels)  leaf=%s leaf-alts=%d leaf-refs=%d leaf-memo=%v\n",
			name(lad.levels[0].rule), len(lad.levels), name(lad.leaf),
			program.altCount(lad.leaf), refs[lad.leaf], program.memoRuleIndex[lad.leaf] >= 0)
		for _, lv := range lad.levels {
			memoed := program.memoRuleIndex[lv.rule] >= 0
			fmt.Fprintf(&b, "[ladder]   %-10s memo=%-5v refs=%-3d tail->%-10s  ops:{%s}\n",
				name(lv.rule), memoed, refs[lv.rule], name(lv.tailRef.rule),
				strings.Join(program.scanOperatorKeywords(lv.opAlts), " "))
			for i, a := range lv.opAlts {
				fmt.Fprintf(&b, "[ladder]        alt %2d: %s\n", i, program.altShape(a, names))
			}
		}
		// walk the tail chain onward from the leaf so we see what the detector
		// stopped at and how the remaining precedence levels are structured.
		fmt.Fprintf(&b, "[ladder]   --- tail chain past leaf ---\n")
		cur := lad.leaf
		for step := 0; step < 10 && cur >= 0 && cur < len(program.rules); step++ {
			root := program.rules[cur].root
			if root == nil {
				break
			}
			kind := jitParserNodeKindName(root.kind)
			memoed := program.memoRuleIndex[cur] >= 0
			gen := "identity"
			if !program.rules[cur].generator.IsNil() {
				gen = "generator"
			}
			fmt.Fprintf(&b, "[ladder]   %-10s kind=%-10s children=%d memo=%-5v gen=%s refs=%d  ops:{%s}\n",
				name(cur), kind, len(root.children), memoed, gen, refs[cur],
				strings.Join(program.scanOperatorKeywords(root.children), " "))
			// follow the last child if it is a bare ruleref (choice tail) or the
			// first child if it is (define a <ruleref>) inside a sequence.
			nxt := -1
			if len(root.children) > 0 {
				if last := root.children[len(root.children)-1]; last.kind == jitParserRuleRef {
					nxt = last.rule
				} else if first := root.children[0]; first.kind == jitParserBind && len(first.children) == 1 && first.children[0].kind == jitParserRuleRef {
					nxt = first.children[0].rule
				}
			}
			if nxt == cur || nxt < 0 {
				break
			}
			cur = nxt
		}
	}
	sig := b.String()
	if dumpedLadderSignatures[sig] {
		return
	}
	dumpedLadderSignatures[sig] = true
	fmt.Fprint(os.Stderr, sig)
}

func jitParserNodeKindName(k jitParserNodeKind) string {
	switch k {
	case jitParserAtom:
		return "atom"
	case jitParserRegex:
		return "regex"
	case jitParserSequence:
		return "seq"
	case jitParserChoice:
		return "choice"
	case jitParserExclude:
		return "exclude"
	case jitParserZeroOrMore:
		return "zeroOrMore"
	case jitParserOneOrMore:
		return "oneOrMore"
	case jitParserOptional:
		return "optional"
	case jitParserBind:
		return "bind"
	case jitParserCapture:
		return "capture"
	case jitParserRuleRef:
		return "ruleref"
	case jitParserEnd:
		return "end"
	case jitParserEmpty:
		return "empty"
	case jitParserRest:
		return "rest"
	default:
		return fmt.Sprintf("kind%d", k)
	}
}

// walkAltBody visits every node reachable from the given roots, transparently
// following *unnamed* rulerefs into their target rule's root (the builder splits
// each production into its own anonymous rule) but stopping at *named* rulerefs
// and reporting them via onNamedRef. Atoms are reported via onAtom.
func (program *jitParserProgram) walkAltBody(roots []*jitParserNode, names map[int]string,
	onAtom func(string), onNamedRef func(int, string)) {
	visited := map[int]bool{}
	var walk func(n *jitParserNode, depth int)
	walk = func(n *jitParserNode, depth int) {
		if n == nil || depth > 40 {
			return
		}
		switch n.kind {
		case jitParserAtom:
			t := strings.TrimSpace(n.description)
			if t == "" {
				t = strings.TrimSpace(String(n.value))
			}
			if t != "" {
				onAtom(t)
			}
			return
		case jitParserRuleRef:
			if nm, ok := names[n.rule]; ok {
				onNamedRef(n.rule, nm)
				return
			}
			if n.rule >= 0 && n.rule < len(program.rules) && !visited[n.rule] {
				visited[n.rule] = true
				walk(program.rules[n.rule].root, depth+1)
			}
			return
		}
		for _, c := range n.children {
			walk(c, depth+1)
		}
	}
	for _, r := range roots {
		walk(r, 0)
	}
}

// scanOperatorKeywords collects the literal atom texts one hop into each
// (unnamed) op-alt rule, stopping at any further ruleref, so a ladder level's
// distinguishing operator tokens are visible at a glance.
func (program *jitParserProgram) scanOperatorKeywords(nodes []*jitParserNode) []string {
	names := program.ruleNames()
	seen := map[string]bool{}
	var out []string
	var collect func(n *jitParserNode, depth int)
	collect = func(n *jitParserNode, depth int) {
		if n == nil || depth > 10 {
			return
		}
		switch n.kind {
		case jitParserAtom:
			t := strings.TrimSpace(n.description)
			if t == "" {
				t = strings.TrimSpace(String(n.value))
			}
			if t != "" && !seen[t] {
				seen[t] = true
				out = append(out, fmt.Sprintf("%q", t))
			}
			return
		case jitParserRuleRef:
			return
		}
		for _, c := range n.children {
			collect(c, depth+1)
		}
	}
	for _, n := range nodes {
		root := n
		if n != nil && n.kind == jitParserRuleRef {
			if _, named := names[n.rule]; !named && n.rule >= 0 && n.rule < len(program.rules) {
				root = program.rules[n.rule].root
			}
		}
		collect(root, 0)
	}
	return out
}

// altShape renders one alternative's IMMEDIATE structure: one hop into an
// unnamed op-alt rule, then the sequence/choice shape with atoms and ruleref
// names, WITHOUT following any further ruleref. This shows exactly which
// operator token(s) distinguish the alternative and what its operands are.
func (program *jitParserProgram) altShape(n *jitParserNode, names map[int]string) string {
	if n == nil {
		return "<nil>"
	}
	root := n
	if n.kind == jitParserRuleRef {
		if _, named := names[n.rule]; !named && n.rule >= 0 && n.rule < len(program.rules) {
			root = program.rules[n.rule].root
		}
	}
	return program.shallowShape(root, names, 0)
}

func (program *jitParserProgram) shallowShape(n *jitParserNode, names map[int]string, depth int) string {
	if n == nil {
		return "?"
	}
	if depth > 4 {
		return "…"
	}
	kids := func(sep string) string {
		parts := make([]string, 0, len(n.children))
		for _, c := range n.children {
			parts = append(parts, program.shallowShape(c, names, depth+1))
		}
		return strings.Join(parts, sep)
	}
	switch n.kind {
	case jitParserAtom:
		t := strings.TrimSpace(n.description)
		if t == "" {
			t = strings.TrimSpace(String(n.value))
		}
		return fmt.Sprintf("%q", t)
	case jitParserRegex:
		return "/" + n.description + "/"
	case jitParserRuleRef:
		if nm, ok := names[n.rule]; ok {
			return nm
		}
		return fmt.Sprintf("<r%d>", n.rule)
	case jitParserBind:
		return "def(" + kids(" ") + ")"
	case jitParserCapture:
		return "cap(" + kids(" ") + ")"
	case jitParserSequence:
		return "[" + kids(" ") + "]"
	case jitParserChoice:
		return "(" + kids(" | ") + ")"
	case jitParserZeroOrMore:
		return "*(" + kids(" ") + ")"
	case jitParserOneOrMore:
		return "+(" + kids(" ") + ")"
	case jitParserOptional:
		return "?(" + kids(" ") + ")"
	case jitParserExclude:
		return "!(" + kids(" ") + ")"
	default:
		return jitParserNodeKindName(n.kind)
	}
}

// altCount reports how many top-level alternatives rule r's root choice has
// (0 if it is not a choice).
func (program *jitParserProgram) altCount(r int) int {
	if r < 0 || r >= len(program.rules) {
		return 0
	}
	root := program.rules[r].root
	if root == nil || root.kind != jitParserChoice {
		return 0
	}
	return len(root.children)
}
