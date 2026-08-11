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
	"math"
	"sort"
	"strconv"
)

const (
	joinOrderConnectedBudget = 10000
	joinOrderLinearizedLimit = 100
	joinOrderLinearizedChunk = 100
	joinOrderHypergraphChunk = 10
	joinOrderDPBudget        = 10000
)

type joinOrderNode struct {
	alias string
	rows  float64
}

type joinOrderPredicate struct {
	refs        []int
	set         joinOrderSet
	selectivity float64
}

type joinOrderGraph struct {
	nodes      []joinOrderNode
	predicates []joinOrderPredicate
	words      int
}

type joinOrderSet []uint64

type joinOrderPlan struct {
	set         joinOrderSet
	left        *joinOrderPlan
	right       *joinOrderPlan
	leaf        int
	cardinality float64
	cost        float64
	size        int
	atomic      bool
}

func newJoinOrderSet(words int) joinOrderSet {
	return make(joinOrderSet, words)
}

func singletonJoinOrderSet(words, node int) joinOrderSet {
	result := newJoinOrderSet(words)
	result[node/64] |= uint64(1) << uint(node%64)
	return result
}

func (s joinOrderSet) clone() joinOrderSet {
	result := make(joinOrderSet, len(s))
	copy(result, s)
	return result
}

func (s joinOrderSet) has(node int) bool {
	return s[node/64]&(uint64(1)<<uint(node%64)) != 0
}

func (s joinOrderSet) add(node int) {
	s[node/64] |= uint64(1) << uint(node%64)
}

func (s joinOrderSet) intersects(other joinOrderSet) bool {
	for i := range s {
		if s[i]&other[i] != 0 {
			return true
		}
	}
	return false
}

func (s joinOrderSet) disjoint(other joinOrderSet) bool {
	return !s.intersects(other)
}

func (s joinOrderSet) contains(other joinOrderSet) bool {
	for i := range s {
		if s[i]&other[i] != other[i] {
			return false
		}
	}
	return true
}

func (s joinOrderSet) union(other joinOrderSet) joinOrderSet {
	result := newJoinOrderSet(len(s))
	for i := range s {
		result[i] = s[i] | other[i]
	}
	return result
}

func (s joinOrderSet) difference(other joinOrderSet) joinOrderSet {
	result := newJoinOrderSet(len(s))
	for i := range s {
		result[i] = s[i] &^ other[i]
	}
	return result
}

func (s joinOrderSet) equal(other joinOrderSet) bool {
	for i := range s {
		if s[i] != other[i] {
			return false
		}
	}
	return true
}

func (s joinOrderSet) count() int {
	result := 0
	for _, word := range s {
		result += popcount64(word)
	}
	return result
}

func popcount64(value uint64) int {
	value -= (value >> 1) & 0x5555555555555555
	value = (value & 0x3333333333333333) + ((value >> 2) & 0x3333333333333333)
	return int((((value + (value >> 4)) & 0x0f0f0f0f0f0f0f0f) * 0x0101010101010101) >> 56)
}

func (s joinOrderSet) first() int {
	for wordIndex, word := range s {
		if word == 0 {
			continue
		}
		for bit := 0; bit < 64; bit++ {
			if word&(uint64(1)<<uint(bit)) != 0 {
				return wordIndex*64 + bit
			}
		}
	}
	return -1
}

func (s joinOrderSet) key() string {
	buffer := make([]byte, 0, len(s)*18)
	for _, word := range s {
		buffer = strconv.AppendUint(buffer, word, 16)
		buffer = append(buffer, ':')
	}
	return string(buffer)
}

func (s joinOrderSet) nodes() []int {
	result := make([]int, 0, s.count())
	for wordIndex, word := range s {
		for bit := 0; bit < 64; bit++ {
			if word&(uint64(1)<<uint(bit)) != 0 {
				result = append(result, wordIndex*64+bit)
			}
		}
	}
	return result
}

func predicateSet(graph *joinOrderGraph, predicate joinOrderPredicate) joinOrderSet {
	if predicate.set != nil {
		return predicate.set
	}
	result := newJoinOrderSet(graph.words)
	for _, ref := range predicate.refs {
		result.add(ref)
	}
	return result
}

func predicateCrosses(graph *joinOrderGraph, predicate joinOrderPredicate, left, right joinOrderSet) bool {
	refs := predicateSet(graph, predicate)
	leftIntersection := false
	rightIntersection := false
	for i := range refs {
		if (left[i]|right[i])&refs[i] != refs[i] {
			return false
		}
		leftIntersection = leftIntersection || left[i]&refs[i] != 0
		rightIntersection = rightIntersection || right[i]&refs[i] != 0
	}
	return leftIntersection && rightIntersection
}

func joinOrderConnected(graph *joinOrderGraph, left, right joinOrderSet) bool {
	for _, predicate := range graph.predicates {
		if predicateCrosses(graph, predicate, left, right) {
			return true
		}
	}
	return false
}

func joinOrderCardinality(graph *joinOrderGraph, left, right *joinOrderPlan) float64 {
	selectivity := 1.0
	for _, predicate := range graph.predicates {
		if predicateCrosses(graph, predicate, left.set, right.set) {
			selectivity *= predicate.selectivity
		}
	}
	result := left.cardinality * right.cardinality * selectivity
	if math.IsInf(result, 0) || result > math.MaxFloat64/4 {
		return math.MaxFloat64 / 4
	}
	if result < 1 {
		return 1
	}
	return result
}

func makeJoinOrderLeaf(graph *joinOrderGraph, node int) *joinOrderPlan {
	rows := graph.nodes[node].rows
	if rows < 1 {
		rows = 1
	}
	return &joinOrderPlan{
		set:         singletonJoinOrderSet(graph.words, node),
		leaf:        node,
		cardinality: rows,
		size:        1,
	}
}

func makeJoinOrderJoin(graph *joinOrderGraph, left, right *joinOrderPlan) *joinOrderPlan {
	cardinality := joinOrderCardinality(graph, left, right)
	return &joinOrderPlan{
		set:         left.set.union(right.set),
		left:        left,
		right:       right,
		leaf:        -1,
		cardinality: cardinality,
		cost:        left.cost + right.cost + cardinality,
		size:        left.size + right.size,
	}
}

func betterJoinOrderPlan(current, candidate *joinOrderPlan) *joinOrderPlan {
	if current == nil || candidate.cost < current.cost ||
		(candidate.cost == current.cost && joinOrderDriverCardinality(candidate) < joinOrderDriverCardinality(current)) {
		return candidate
	}
	return current
}

func joinOrderDriverCardinality(plan *joinOrderPlan) float64 {
	for plan.left != nil {
		plan = plan.left
	}
	return plan.cardinality
}

func bestJoinOrderOrientation(graph *joinOrderGraph, left, right *joinOrderPlan) *joinOrderPlan {
	return betterJoinOrderPlan(
		makeJoinOrderJoin(graph, left, right),
		makeJoinOrderJoin(graph, right, left),
	)
}

func enumerateConnectedJoinOrderSets(graph *joinOrderGraph, allowed joinOrderSet, budget int) ([]joinOrderSet, bool) {
	seen := make(map[string]joinOrderSet)
	frontier := make([]joinOrderSet, 0, allowed.count())
	for _, node := range allowed.nodes() {
		set := singletonJoinOrderSet(graph.words, node)
		seen[set.key()] = set
		frontier = append(frontier, set)
	}
	for len(frontier) > 0 {
		current := frontier[0]
		frontier = frontier[1:]
		for _, predicate := range graph.predicates {
			refs := predicateSet(graph, predicate)
			if !allowed.contains(refs) || !current.intersects(refs) || current.contains(refs) {
				continue
			}
			next := current.union(refs)
			key := next.key()
			if _, found := seen[key]; found {
				continue
			}
			seen[key] = next
			frontier = append(frontier, next)
			if budget > 0 && len(seen) > budget {
				return nil, false
			}
		}
	}
	result := make([]joinOrderSet, 0, len(seen))
	for _, set := range seen {
		result = append(result, set)
	}
	sort.Slice(result, func(i, j int) bool {
		leftCount := result[i].count()
		rightCount := result[j].count()
		if leftCount != rightCount {
			return leftCount < rightCount
		}
		return result[i].key() < result[j].key()
	})
	return result, true
}

func dphypJoinOrder(graph *joinOrderGraph, allowed joinOrderSet) (*joinOrderPlan, int) {
	connected, _ := enumerateConnectedJoinOrderSets(graph, allowed, 0)
	plans := make(map[string]*joinOrderPlan, len(connected))
	connectedByFirst := make([][]joinOrderSet, len(graph.nodes))
	for _, set := range connected {
		connectedByFirst[set.first()] = append(connectedByFirst[set.first()], set)
	}
	for _, set := range connected {
		if set.count() == 1 {
			plans[set.key()] = makeJoinOrderLeaf(graph, set.first())
			continue
		}
		var best *joinOrderPlan
		first := set.first()
		for _, leftSet := range connectedByFirst[first] {
			if leftSet.count() >= set.count() || !set.contains(leftSet) {
				continue
			}
			rightSet := set.difference(leftSet)
			left := plans[leftSet.key()]
			right := plans[rightSet.key()]
			if left == nil || right == nil || !joinOrderConnected(graph, leftSet, rightSet) {
				continue
			}
			best = betterJoinOrderPlan(best, bestJoinOrderOrientation(graph, left, right))
		}
		if best != nil {
			plans[set.key()] = best
		}
	}
	return plans[allowed.key()], len(plans)
}

type joinOrderDSU struct {
	parent []int
	rank   []int
}

func newJoinOrderDSU(size int) *joinOrderDSU {
	parent := make([]int, size)
	for i := range parent {
		parent[i] = i
	}
	return &joinOrderDSU{parent: parent, rank: make([]int, size)}
}

func (d *joinOrderDSU) find(node int) int {
	if d.parent[node] != node {
		d.parent[node] = d.find(d.parent[node])
	}
	return d.parent[node]
}

func (d *joinOrderDSU) union(left, right int) bool {
	left = d.find(left)
	right = d.find(right)
	if left == right {
		return false
	}
	if d.rank[left] < d.rank[right] {
		left, right = right, left
	}
	d.parent[right] = left
	if d.rank[left] == d.rank[right] {
		d.rank[left]++
	}
	return true
}

type joinOrderTreeEdge struct {
	left        int
	right       int
	selectivity float64
}

func regularJoinOrderEdges(graph *joinOrderGraph, allowed joinOrderSet) []joinOrderTreeEdge {
	type edgeKey struct {
		left  int
		right int
	}
	selectivities := make(map[edgeKey]float64)
	for _, predicate := range graph.predicates {
		if len(predicate.refs) == 2 && allowed.has(predicate.refs[0]) && allowed.has(predicate.refs[1]) {
			left, right := predicate.refs[0], predicate.refs[1]
			if right < left {
				left, right = right, left
			}
			key := edgeKey{left: left, right: right}
			if selectivity, found := selectivities[key]; found {
				selectivities[key] = selectivity * predicate.selectivity
			} else {
				selectivities[key] = predicate.selectivity
			}
		}
	}
	result := make([]joinOrderTreeEdge, 0, len(selectivities))
	for key, selectivity := range selectivities {
		result = append(result, joinOrderTreeEdge{
			left: key.left, right: key.right, selectivity: selectivity,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].left != result[j].left {
			return result[i].left < result[j].left
		}
		return result[i].right < result[j].right
	})
	return result
}

func minimumSpanningJoinOrderTree(graph *joinOrderGraph, allowed joinOrderSet) []joinOrderTreeEdge {
	edges := regularJoinOrderEdges(graph, allowed)
	sort.SliceStable(edges, func(i, j int) bool {
		return edges[i].selectivity < edges[j].selectivity
	})
	dsu := newJoinOrderDSU(len(graph.nodes))
	result := make([]joinOrderTreeEdge, 0, allowed.count()-1)
	for _, edge := range edges {
		if dsu.union(edge.left, edge.right) {
			result = append(result, edge)
		}
	}
	// The papers make cross products explicit. Connect remaining components
	// deterministically with selectivity one instead of considering every pair.
	nodes := allowed.nodes()
	for i := 1; i < len(nodes); i++ {
		if dsu.union(nodes[i-1], nodes[i]) {
			result = append(result, joinOrderTreeEdge{left: nodes[i-1], right: nodes[i], selectivity: 1})
		}
	}
	return result
}

type ikkbzItem struct {
	nodes  []int
	factor float64
	cost   float64
}

func (item ikkbzItem) rank() float64 {
	if item.cost == 0 {
		return math.Inf(1)
	}
	return (item.factor - 1) / item.cost
}

func mergeIKKBZItems(left, right ikkbzItem) ikkbzItem {
	return ikkbzItem{
		nodes:  append(append([]int{}, left.nodes...), right.nodes...),
		factor: left.factor * right.factor,
		cost:   left.cost + left.factor*right.cost,
	}
}

func normalizeIKKBZChain(chain []ikkbzItem) []ikkbzItem {
	for {
		merged := false
		for i := 0; i+1 < len(chain); i++ {
			if chain[i].rank() <= chain[i+1].rank() {
				continue
			}
			compound := mergeIKKBZItems(chain[i], chain[i+1])
			chain = append(append(append([]ikkbzItem{}, chain[:i]...), compound), chain[i+2:]...)
			merged = true
			break
		}
		if !merged {
			return chain
		}
	}
}

func mergeIKKBZChains(chains [][]ikkbzItem) []ikkbzItem {
	result := make([]ikkbzItem, 0)
	for {
		best := -1
		for i := range chains {
			if len(chains[i]) == 0 {
				continue
			}
			if best < 0 || chains[i][0].rank() < chains[best][0].rank() {
				best = i
			}
		}
		if best < 0 {
			return result
		}
		result = append(result, chains[best][0])
		chains[best] = chains[best][1:]
	}
}

func ikkbzChain(graph *joinOrderGraph, adjacency [][]joinOrderTreeEdge, node, parent int, parentSelectivity float64) []ikkbzItem {
	children := make([][]ikkbzItem, 0)
	for _, edge := range adjacency[node] {
		child := edge.left
		if child == node {
			child = edge.right
		}
		if child == parent {
			continue
		}
		children = append(children, normalizeIKKBZChain(ikkbzChain(graph, adjacency, child, node, edge.selectivity)))
	}
	factor := 1.0
	cost := 0.0
	if parent >= 0 {
		factor = graph.nodes[node].rows * parentSelectivity
		cost = graph.nodes[node].rows
	}
	return append([]ikkbzItem{{nodes: []int{node}, factor: factor, cost: cost}}, mergeIKKBZChains(children)...)
}

func ikkbzOrders(graph *joinOrderGraph, allowed joinOrderSet) [][]int {
	edges := minimumSpanningJoinOrderTree(graph, allowed)
	adjacency := make([][]joinOrderTreeEdge, len(graph.nodes))
	for _, edge := range edges {
		adjacency[edge.left] = append(adjacency[edge.left], edge)
		adjacency[edge.right] = append(adjacency[edge.right], edge)
	}
	var bestOrder []int
	bestCost := math.Inf(1)
	for _, root := range allowed.nodes() {
		chain := ikkbzChain(graph, adjacency, root, -1, 1)
		order := make([]int, 0, allowed.count())
		for _, item := range chain {
			order = append(order, item.nodes...)
		}
		sequence := chain[0]
		for _, item := range chain[1:] {
			sequence = mergeIKKBZItems(sequence, item)
		}
		if sequence.cost < bestCost {
			bestCost = sequence.cost
			bestOrder = order
		}
	}
	return [][]int{bestOrder}
}

func linearizedDPJoinOrder(graph *joinOrderGraph, allowed joinOrderSet) (*joinOrderPlan, int) {
	var best *joinOrderPlan
	dpEntries := 0
	for _, order := range ikkbzOrders(graph, allowed) {
		table := make([][]*joinOrderPlan, len(order))
		for i, node := range order {
			table[i] = make([]*joinOrderPlan, len(order))
			table[i][i] = makeJoinOrderLeaf(graph, node)
			dpEntries++
		}
		for length := 2; length <= len(order); length++ {
			for start := 0; start+length <= len(order); start++ {
				end := start + length - 1
				for split := start; split < end; split++ {
					left := table[start][split]
					right := table[split+1][end]
					if left == nil || right == nil || !joinOrderConnected(graph, left.set, right.set) {
						continue
					}
					table[start][end] = betterJoinOrderPlan(table[start][end], makeJoinOrderJoin(graph, left, right))
				}
				if table[start][end] != nil {
					dpEntries++
				}
			}
		}
		best = betterJoinOrderPlan(best, table[0][len(order)-1])
	}
	return best, dpEntries
}

func gooJoinOrder(graph *joinOrderGraph, allowed joinOrderSet) *joinOrderPlan {
	plans := make([]*joinOrderPlan, 0, allowed.count())
	for _, node := range allowed.nodes() {
		plans = append(plans, makeJoinOrderLeaf(graph, node))
	}
	for len(plans) > 1 {
		bestLeft, bestRight := -1, -1
		bestCardinality := math.Inf(1)
		for requireConnection := 1; requireConnection >= 0 && bestLeft < 0; requireConnection-- {
			for left := 0; left < len(plans); left++ {
				for right := left + 1; right < len(plans); right++ {
					if requireConnection == 1 && !joinOrderConnected(graph, plans[left].set, plans[right].set) {
						continue
					}
					cardinality := joinOrderCardinality(graph, plans[left], plans[right])
					if cardinality < bestCardinality {
						bestLeft, bestRight, bestCardinality = left, right, cardinality
					}
				}
			}
		}
		joined := bestJoinOrderOrientation(graph, plans[bestLeft], plans[bestRight])
		next := make([]*joinOrderPlan, 0, len(plans)-1)
		for i, plan := range plans {
			if i != bestLeft && i != bestRight {
				next = append(next, plan)
			}
		}
		plans = append(next, joined)
	}
	return plans[0]
}

func expensiveGOOSubtree(plan *joinOrderPlan, parentSize, limit int) *joinOrderPlan {
	if plan == nil || plan.atomic || plan.size == 1 {
		return nil
	}
	var best *joinOrderPlan
	if plan.size <= limit && parentSize > limit {
		best = plan
	}
	for _, child := range []*joinOrderPlan{plan.left, plan.right} {
		candidate := expensiveGOOSubtree(child, plan.size, limit)
		if candidate != nil && (best == nil || candidate.cost > best.cost) {
			best = candidate
		}
	}
	return best
}

func replaceGOOSubtree(plan, target, replacement *joinOrderPlan) *joinOrderPlan {
	if plan == target {
		return replacement
	}
	if plan.left != nil {
		plan.left = replaceGOOSubtree(plan.left, target, replacement)
	}
	if plan.right != nil {
		plan.right = replaceGOOSubtree(plan.right, target, replacement)
	}
	return plan
}

func refreshJoinOrderPlan(graph *joinOrderGraph, plan *joinOrderPlan) *joinOrderPlan {
	if plan.left == nil && plan.right == nil {
		return plan
	}
	plan.left = refreshJoinOrderPlan(graph, plan.left)
	plan.right = refreshJoinOrderPlan(graph, plan.right)
	plan.set = plan.left.set.union(plan.right.set)
	plan.cardinality = joinOrderCardinality(graph, plan.left, plan.right)
	plan.cost = plan.left.cost + plan.right.cost + plan.cardinality
	plan.size = plan.left.size + plan.right.size
	return plan
}

func gooDPJoinOrder(graph *joinOrderGraph, allowed joinOrderSet, hypergraph bool) (*joinOrderPlan, int) {
	plan := gooJoinOrder(graph, allowed)
	limit := joinOrderLinearizedChunk
	if hypergraph {
		limit = joinOrderHypergraphChunk
	}
	budget := joinOrderDPBudget
	used := 0
	for budget > 0 {
		target := expensiveGOOSubtree(plan, plan.size+1, limit)
		if target == nil {
			break
		}
		var replacement *joinOrderPlan
		entries := 0
		if hypergraph {
			replacement, entries = dphypJoinOrder(graph, target.set)
		} else {
			replacement, entries = linearizedDPJoinOrder(graph, target.set)
		}
		if replacement == nil || entries <= 0 {
			target.atomic = true
			continue
		}
		replacement.atomic = true
		plan = replaceGOOSubtree(plan, target, replacement)
		plan = refreshJoinOrderPlan(graph, plan)
		budget -= entries
		used += entries
	}
	return plan, used
}

func completeJoinOrderSet(graph *joinOrderGraph) joinOrderSet {
	result := newJoinOrderSet(graph.words)
	for node := range graph.nodes {
		result.add(node)
	}
	return result
}

func joinOrderHasHyperedges(graph *joinOrderGraph) bool {
	for _, predicate := range graph.predicates {
		if len(predicate.refs) > 2 {
			return true
		}
	}
	return false
}

func addExplicitJoinOrderCrossProducts(graph *joinOrderGraph) {
	if len(graph.nodes) < 2 {
		return
	}
	dsu := newJoinOrderDSU(len(graph.nodes))
	for _, predicate := range graph.predicates {
		if len(predicate.refs) == 2 {
			dsu.union(predicate.refs[0], predicate.refs[1])
		}
	}
	representatives := make([]int, 0)
	seen := make(map[int]bool)
	for node := range graph.nodes {
		root := dsu.find(node)
		if !seen[root] {
			seen[root] = true
			representatives = append(representatives, node)
		}
	}
	for i := 1; i < len(representatives); i++ {
		set := newJoinOrderSet(graph.words)
		set.add(representatives[i-1])
		set.add(representatives[i])
		graph.predicates = append(graph.predicates, joinOrderPredicate{
			refs:        []int{representatives[i-1], representatives[i]},
			set:         set,
			selectivity: 1,
		})
	}
}

func prepareJoinOrderPredicates(graph *joinOrderGraph) {
	for index, predicate := range graph.predicates {
		if predicate.set == nil {
			graph.predicates[index].set = predicateSet(graph, predicate)
		}
	}
}

func adaptiveJoinOrder(graph *joinOrderGraph) (string, *joinOrderPlan, int) {
	prepareJoinOrderPredicates(graph)
	addExplicitJoinOrderCrossProducts(graph)
	allowed := completeJoinOrderSet(graph)
	if len(graph.nodes) < 14 {
		plan, entries := dphypJoinOrder(graph, allowed)
		return "dphyp", plan, entries
	}
	if len(graph.nodes) <= joinOrderLinearizedLimit {
		connected, withinBudget := enumerateConnectedJoinOrderSets(graph, allowed, joinOrderConnectedBudget)
		if withinBudget {
			plan, entries := dphypJoinOrder(graph, allowed)
			return "dphyp", plan, entries
		}
		_ = connected
	}
	if !joinOrderHasHyperedges(graph) {
		if len(graph.nodes) <= joinOrderLinearizedLimit {
			plan, entries := linearizedDPJoinOrder(graph, allowed)
			return "linearized-dp", plan, entries
		}
		plan, entries := gooDPJoinOrder(graph, allowed, false)
		return "goo-linearized-dp", plan, entries
	}
	plan, entries := gooDPJoinOrder(graph, allowed, true)
	return "goo-dphyp", plan, entries
}

func parseJoinOrderGraph(nodesValue, predicatesValue Scmer) *joinOrderGraph {
	nodeValues := asSlice(nodesValue, "neumann_join_order nodes")
	graph := &joinOrderGraph{
		nodes: make([]joinOrderNode, 0, len(nodeValues)),
		words: (len(nodeValues) + 63) / 64,
	}
	aliases := make(map[string]int, len(nodeValues))
	for index, nodeValue := range nodeValues {
		node := asSlice(nodeValue, "neumann_join_order node")
		if len(node) != 2 {
			panic("neumann_join_order node expects (alias cardinality)")
		}
		alias := node[0].String()
		rows := ToFloat(node[1])
		graph.nodes = append(graph.nodes, joinOrderNode{alias: alias, rows: rows})
		aliases[alias] = index
	}
	for _, predicateValue := range asSlice(predicatesValue, "neumann_join_order predicates") {
		predicate := asSlice(predicateValue, "neumann_join_order predicate")
		if len(predicate) != 2 {
			panic("neumann_join_order predicate expects ((aliases...) selectivity)")
		}
		refs := make([]int, 0)
		for _, aliasValue := range asSlice(predicate[0], "neumann_join_order predicate aliases") {
			alias := aliasValue.String()
			ref, found := aliases[alias]
			if !found {
				panic("neumann_join_order predicate references unknown alias " + alias)
			}
			refs = append(refs, ref)
		}
		if len(refs) < 2 {
			continue
		}
		selectivity := ToFloat(predicate[1])
		if selectivity <= 0 || selectivity > 1 || math.IsNaN(selectivity) {
			selectivity = 0.1
		}
		graph.predicates = append(graph.predicates, joinOrderPredicate{refs: refs, selectivity: selectivity})
	}
	return graph
}

func joinOrderPlanTreeScmer(graph *joinOrderGraph, plan *joinOrderPlan) Scmer {
	if plan.left == nil && plan.right == nil {
		return NewSlice([]Scmer{NewSymbol("join-leaf"), NewString(graph.nodes[plan.leaf].alias)})
	}
	return NewSlice([]Scmer{
		NewSymbol("join-node"),
		joinOrderPlanTreeScmer(graph, plan.left),
		joinOrderPlanTreeScmer(graph, plan.right),
	})
}

func joinOrderPlanAliases(graph *joinOrderGraph, plan *joinOrderPlan, result []Scmer) []Scmer {
	if plan.left == nil && plan.right == nil {
		return append(result, NewString(graph.nodes[plan.leaf].alias))
	}
	result = joinOrderPlanAliases(graph, plan.left, result)
	return joinOrderPlanAliases(graph, plan.right, result)
}

func init_joinorder() {
	Declare(&Globalenv, &Declaration{
		Name: "neumann_join_order",
		Desc: "adaptively construct a logical bushy join plan using DPHyp, IKKBZ/linearized DP, and GOO-DP",
		Fn: func(a ...Scmer) Scmer {
			graph := parseJoinOrderGraph(a[0], a[1])
			if len(graph.nodes) == 0 {
				return NewNil()
			}
			strategy, plan, entries := adaptiveJoinOrder(graph)
			if plan == nil {
				panic("neumann_join_order could not construct a connected plan")
			}
			return NewSlice([]Scmer{
				NewSlice([]Scmer{NewSymbol("strategy"), NewSymbol(strategy)}),
				NewSlice([]Scmer{NewSymbol("tree"), joinOrderPlanTreeScmer(graph, plan)}),
				NewSlice([]Scmer{NewSymbol("order"), NewSlice(joinOrderPlanAliases(graph, plan, nil))}),
				NewSlice([]Scmer{NewSymbol("cost"), NewFloat(plan.cost)}),
				NewSlice([]Scmer{NewSymbol("cardinality"), NewFloat(plan.cardinality)}),
				NewSlice([]Scmer{NewSymbol("dp_entries"), NewInt(int64(entries))}),
			})
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "nodes", ParamDesc: "(alias cardinality) pairs"},
				{Kind: "list", ParamName: "predicates", ParamDesc: "(aliases selectivity) pairs"},
			},
			Return: &TypeDescriptor{Kind: "list"},
			Const:  true,
		},
	})
}
