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
	"testing"
)

func testJoinOrderGraph(nodeCount int, predicates []joinOrderPredicate) *joinOrderGraph {
	nodes := make([]joinOrderNode, nodeCount)
	for i := range nodes {
		nodes[i] = joinOrderNode{alias: string(rune('a' + i)), rows: float64(100 + i)}
	}
	return &joinOrderGraph{
		nodes:      nodes,
		predicates: predicates,
		words:      (nodeCount + 63) / 64,
	}
}

func chainJoinOrderPredicates(nodeCount int) []joinOrderPredicate {
	result := make([]joinOrderPredicate, 0, nodeCount-1)
	for i := 1; i < nodeCount; i++ {
		result = append(result, joinOrderPredicate{refs: []int{i - 1, i}, selectivity: 0.01})
	}
	return result
}

func cliqueJoinOrderPredicates(nodeCount int) []joinOrderPredicate {
	result := make([]joinOrderPredicate, 0, nodeCount*nodeCount/2)
	for left := 0; left < nodeCount; left++ {
		for right := left + 1; right < nodeCount; right++ {
			result = append(result, joinOrderPredicate{refs: []int{left, right}, selectivity: 0.1})
		}
	}
	return result
}

func TestAdaptiveJoinOrderUsesDPHypForSmallQueries(t *testing.T) {
	graph := testJoinOrderGraph(6, chainJoinOrderPredicates(6))
	strategy, plan, entries := adaptiveJoinOrder(graph)
	if strategy != "dphyp" {
		t.Fatalf("expected dphyp, got %s", strategy)
	}
	if plan == nil || plan.size != 6 || entries != 21 {
		t.Fatalf("unexpected exact plan: plan=%#v entries=%d", plan, entries)
	}
}

func TestAdaptiveJoinOrderUsesLinearizedDPForDenseMediumQueries(t *testing.T) {
	graph := testJoinOrderGraph(14, cliqueJoinOrderPredicates(14))
	strategy, plan, entries := adaptiveJoinOrder(graph)
	if strategy != "linearized-dp" {
		t.Fatalf("expected linearized-dp, got %s", strategy)
	}
	if plan == nil || plan.size != 14 || entries == 0 {
		t.Fatalf("unexpected linearized plan: plan=%#v entries=%d", plan, entries)
	}
}

func TestAdaptiveJoinOrderUsesGOODPForVeryLargeQueries(t *testing.T) {
	graph := testJoinOrderGraph(101, chainJoinOrderPredicates(101))
	strategy, plan, entries := adaptiveJoinOrder(graph)
	if strategy != "goo-linearized-dp" {
		t.Fatalf("expected goo-linearized-dp, got %s", strategy)
	}
	if plan == nil || plan.size != 101 || entries == 0 {
		t.Fatalf("unexpected GOO-DP plan: plan=%#v entries=%d", plan, entries)
	}
}

func TestAdaptiveJoinOrderKeepsLongChainsExact(t *testing.T) {
	graph := testJoinOrderGraph(100, chainJoinOrderPredicates(100))
	strategy, plan, entries := adaptiveJoinOrder(graph)
	if strategy != "dphyp" {
		t.Fatalf("expected dphyp for a graph with 5,050 connected subsets, got %s", strategy)
	}
	if plan == nil || plan.size != 100 || entries != 5050 {
		t.Fatalf("unexpected long-chain plan: plan=%#v entries=%d", plan, entries)
	}
}

func TestAdaptiveJoinOrderUsesDPHypInsideGOOForHypergraphs(t *testing.T) {
	predicates := cliqueJoinOrderPredicates(14)
	predicates = append(predicates, joinOrderPredicate{refs: []int{0, 1, 2}, selectivity: 0.01})
	graph := testJoinOrderGraph(14, predicates)
	strategy, plan, entries := adaptiveJoinOrder(graph)
	if strategy != "goo-dphyp" {
		t.Fatalf("expected goo-dphyp, got %s", strategy)
	}
	if plan == nil || plan.size != 14 || entries == 0 {
		t.Fatalf("unexpected hypergraph plan: plan=%#v entries=%d", plan, entries)
	}
}

func TestAdaptiveJoinOrderMakesCrossProductsExplicit(t *testing.T) {
	graph := testJoinOrderGraph(3, nil)
	strategy, plan, _ := adaptiveJoinOrder(graph)
	if strategy != "dphyp" || plan == nil || plan.size != 3 {
		t.Fatalf("expected a complete explicit cross-product plan, got %s %#v", strategy, plan)
	}
}

func TestAdaptiveJoinOrderBuildsInputsForAStandaloneHyperedge(t *testing.T) {
	graph := testJoinOrderGraph(3, []joinOrderPredicate{{
		refs: []int{0, 1, 2}, selectivity: 0.01,
	}})
	strategy, plan, _ := adaptiveJoinOrder(graph)
	if strategy != "dphyp" || plan == nil || plan.size != 3 {
		t.Fatalf("expected DPHyp to build a complete hyperedge input, got %s %#v", strategy, plan)
	}
}

func TestIKKBZCombinesPredicatesOnTheSameGraphEdge(t *testing.T) {
	graph := testJoinOrderGraph(3, []joinOrderPredicate{
		{refs: []int{0, 1}, selectivity: 0.5},
		{refs: []int{1, 0}, selectivity: 0.2},
		{refs: []int{1, 2}, selectivity: 0.2},
	})
	edges := regularJoinOrderEdges(graph, completeJoinOrderSet(graph))
	if len(edges) != 2 {
		t.Fatalf("expected two graph edges, got %#v", edges)
	}
	if edges[0].left != 0 || edges[0].right != 1 || math.Abs(edges[0].selectivity-0.1) > 1e-12 {
		t.Fatalf("expected combined selectivity for edge 0-1, got %#v", edges[0])
	}
}
