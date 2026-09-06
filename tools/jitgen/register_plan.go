/*
Copyright (C) 2026  Carl-Philip Haensch

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
package main

import (
	"sort"

	"golang.org/x/tools/go/ssa"
)

// staticRegisterPlan contains architecture-independent colors. Physical
// registers and their constraints belong to the runtime architecture backend.
// Colors are ordered by estimated avoided loop traffic, allowing a backend to
// keep the most valuable subset when it has fewer persistent registers.
type staticRegisterPlan struct {
	colorByValue map[string]int
	colorCount   int
}

type registerPlanNode struct {
	value     *ssa.Phi
	weight    int
	neighbors map[int]struct{}
}

const exactRegisterColoringLimit = 20

func planLoopPhiRegisters(fn *ssa.Function) staticRegisterPlan {
	plan := staticRegisterPlan{colorByValue: map[string]int{}}
	if fn == nil {
		return plan
	}

	var nodes []registerPlanNode
	for _, block := range fn.Blocks {
		if !registerPlanLoopHeader(block) {
			continue
		}
		for _, instruction := range block.Instrs {
			phi, ok := instruction.(*ssa.Phi)
			if !ok {
				break
			}
			if isPhiPairType(phi.Type()) || isPhiTripleType(phi.Type()) {
				continue
			}
			// Phi-to-phi assignments need a parallel-copy schedule. Retain their
			// canonical stack transport until that is modeled by the planner.
			if registerPlanHasPhiInput(phi) {
				continue
			}
			weight := 1
			if refs := phi.Referrers(); refs != nil {
				// A use inside the loop is paid on every iteration. This simple
				// static weight is deterministic and sufficient for ordering homes.
				weight += len(*refs)
			}
			nodes = append(nodes, registerPlanNode{
				value:     phi,
				weight:    weight,
				neighbors: map[int]struct{}{},
			})
		}
	}
	if len(nodes) == 0 {
		return plan
	}

	index := make(map[ssa.Value]int, len(nodes))
	for i := range nodes {
		index[nodes[i].value] = i
	}
	liveIn, liveOut := registerPlanLiveness(fn, index)
	for blockIndex, block := range fn.Blocks {
		addRegisterPlanClique(nodes, liveIn[blockIndex])
		addRegisterPlanClique(nodes, liveOut[blockIndex])
		live := cloneRegisterSet(liveOut[blockIndex])
		for instructionIndex := len(block.Instrs) - 1; instructionIndex >= 0; instructionIndex-- {
			instruction := block.Instrs[instructionIndex]
			if phi, ok := instruction.(*ssa.Phi); ok {
				delete(live, index[phi])
				continue
			}
			for _, operand := range instruction.Operands(nil) {
				if operand != nil && *operand != nil {
					if node, exists := index[*operand]; exists {
						live[node] = struct{}{}
					}
				}
			}
			addRegisterPlanClique(nodes, live)
		}
		definitions := make(registerSet)
		for _, instruction := range block.Instrs {
			phi, ok := instruction.(*ssa.Phi)
			if !ok {
				break
			}
			if node, exists := index[phi]; exists {
				definitions[node] = struct{}{}
			}
		}
		addRegisterPlanClique(nodes, definitions)
	}

	colors, colorCount := colorRegisterPlan(nodes)
	if colorCount == 0 {
		return plan
	}
	weights := make([]int, colorCount)
	for node, color := range colors {
		weights[color] += nodes[node].weight
	}
	order := make([]int, colorCount)
	for color := range order {
		order[color] = color
	}
	sort.SliceStable(order, func(i, j int) bool {
		return weights[order[i]] > weights[order[j]]
	})
	remap := make([]int, colorCount)
	for priority, color := range order {
		remap[color] = priority
	}
	for node, color := range colors {
		plan.colorByValue[nodes[node].value.Name()] = remap[color]
	}
	plan.colorCount = colorCount
	return plan
}

func registerPlanLoopHeader(block *ssa.BasicBlock) bool {
	for _, predecessor := range block.Preds {
		if block.Dominates(predecessor) {
			return true
		}
	}
	return false
}

func registerPlanHasPhiInput(phi *ssa.Phi) bool {
	for _, edge := range phi.Edges {
		if _, ok := edge.(*ssa.Phi); ok {
			return true
		}
	}
	return false
}

type registerSet map[int]struct{}

func cloneRegisterSet(source registerSet) registerSet {
	result := make(registerSet, len(source))
	for value := range source {
		result[value] = struct{}{}
	}
	return result
}

func registerPlanLiveness(fn *ssa.Function, index map[ssa.Value]int) ([]registerSet, []registerSet) {
	uses := make([]registerSet, len(fn.Blocks))
	definitions := make([]registerSet, len(fn.Blocks))
	for blockIndex := range fn.Blocks {
		uses[blockIndex] = make(registerSet)
		definitions[blockIndex] = make(registerSet)
	}
	for blockIndex, block := range fn.Blocks {
		for _, instruction := range block.Instrs {
			if phi, ok := instruction.(*ssa.Phi); ok {
				if node, exists := index[phi]; exists {
					definitions[blockIndex][node] = struct{}{}
				}
				for edgeIndex, edge := range phi.Edges {
					if node, exists := index[edge]; exists && edgeIndex < len(block.Preds) {
						uses[block.Preds[edgeIndex].Index][node] = struct{}{}
					}
				}
				continue
			}
			for _, operand := range instruction.Operands(nil) {
				if operand == nil || *operand == nil {
					continue
				}
				if node, exists := index[*operand]; exists {
					if _, defined := definitions[blockIndex][node]; !defined {
						uses[blockIndex][node] = struct{}{}
					}
				}
			}
		}
	}

	liveIn := make([]registerSet, len(fn.Blocks))
	liveOut := make([]registerSet, len(fn.Blocks))
	for blockIndex := range fn.Blocks {
		liveIn[blockIndex] = make(registerSet)
		liveOut[blockIndex] = make(registerSet)
	}
	changed := true
	for changed {
		changed = false
		for blockIndex := len(fn.Blocks) - 1; blockIndex >= 0; blockIndex-- {
			block := fn.Blocks[blockIndex]
			out := cloneRegisterSet(uses[blockIndex])
			for _, successor := range block.Succs {
				for value := range liveIn[successor.Index] {
					out[value] = struct{}{}
				}
			}
			in := cloneRegisterSet(uses[blockIndex])
			for value := range out {
				if _, defined := definitions[blockIndex][value]; !defined {
					in[value] = struct{}{}
				}
			}
			if !sameRegisterSet(out, liveOut[blockIndex]) || !sameRegisterSet(in, liveIn[blockIndex]) {
				liveOut[blockIndex], liveIn[blockIndex] = out, in
				changed = true
			}
		}
	}
	return liveIn, liveOut
}

func sameRegisterSet(left, right registerSet) bool {
	if len(left) != len(right) {
		return false
	}
	for value := range left {
		if _, exists := right[value]; !exists {
			return false
		}
	}
	return true
}

func addRegisterPlanClique(nodes []registerPlanNode, values registerSet) {
	for left := range values {
		for right := range values {
			if left >= right {
				continue
			}
			nodes[left].neighbors[right] = struct{}{}
			nodes[right].neighbors[left] = struct{}{}
		}
	}
}

func colorRegisterPlan(nodes []registerPlanNode) ([]int, int) {
	if len(nodes) == 0 {
		return nil, 0
	}
	// Connected components are independent regions of the interference graph.
	// Coloring them separately bounds exact-search cost while safely reusing the
	// same abstract colors between unrelated loops and helpers.
	assignment := make([]int, len(nodes))
	for i := range assignment {
		assignment[i] = -1
	}
	colorCount := 0
	for _, component := range registerPlanComponents(nodes) {
		componentColors, componentCount := colorRegisterComponent(nodes, component)
		if componentCount > colorCount {
			colorCount = componentCount
		}
		for index, node := range component {
			assignment[node] = componentColors[index]
		}
	}
	return assignment, colorCount
}

func registerPlanComponents(nodes []registerPlanNode) [][]int {
	seen := make([]bool, len(nodes))
	var result [][]int
	for start := range nodes {
		if seen[start] {
			continue
		}
		seen[start] = true
		queue := []int{start}
		component := make([]int, 0, 1)
		for len(queue) != 0 {
			node := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			component = append(component, node)
			for neighbor := range nodes[node].neighbors {
				if !seen[neighbor] {
					seen[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}
		result = append(result, component)
	}
	return result
}

func colorRegisterComponent(nodes []registerPlanNode, component []int) ([]int, int) {
	greedy, upperBound := greedyRegisterColoring(nodes, component)
	if len(component) > exactRegisterColoringLimit {
		return greedy, upperBound
	}
	best := append([]int(nil), greedy...)
	assignment := make([]int, len(component))
	for index := range assignment {
		assignment[index] = -1
	}
	position := make(map[int]int, len(component))
	for index, node := range component {
		position[node] = index
	}

	var search func(colored, usedColors int)
	search = func(colored, usedColors int) {
		if usedColors >= upperBound {
			return
		}
		if colored == len(component) {
			upperBound = usedColors
			copy(best, assignment)
			return
		}
		positionIndex := selectRegisterColorNode(nodes, component, position, assignment)
		node := component[positionIndex]
		for color := 0; color <= usedColors && color < upperBound; color++ {
			if color == usedColors && usedColors+1 >= upperBound {
				continue
			}
			allowed := true
			for neighbor := range nodes[node].neighbors {
				if other, exists := position[neighbor]; exists && assignment[other] == color {
					allowed = false
					break
				}
			}
			if !allowed {
				continue
			}
			assignment[positionIndex] = color
			nextUsed := usedColors
			if color == usedColors {
				nextUsed++
			}
			search(colored+1, nextUsed)
			assignment[positionIndex] = -1
		}
	}
	search(0, 0)
	return best, upperBound
}

func greedyRegisterColoring(nodes []registerPlanNode, component []int) ([]int, int) {
	assignment := make([]int, len(component))
	for index := range assignment {
		assignment[index] = -1
	}
	position := make(map[int]int, len(component))
	for index, node := range component {
		position[node] = index
	}
	usedColors := 0
	for colored := 0; colored < len(component); colored++ {
		positionIndex := selectRegisterColorNode(nodes, component, position, assignment)
		node := component[positionIndex]
		for color := 0; ; color++ {
			allowed := true
			for neighbor := range nodes[node].neighbors {
				if other, exists := position[neighbor]; exists && assignment[other] == color {
					allowed = false
					break
				}
			}
			if allowed {
				assignment[positionIndex] = color
				if color >= usedColors {
					usedColors = color + 1
				}
				break
			}
		}
	}
	return assignment, usedColors
}

func selectRegisterColorNode(nodes []registerPlanNode, component []int, position map[int]int, assignment []int) int {
	best, bestSaturation, bestDegree, bestWeight := -1, -1, -1, -1
	for componentIndex, node := range component {
		if assignment[componentIndex] >= 0 {
			continue
		}
		neighborColors := make(map[int]struct{})
		for neighbor := range nodes[node].neighbors {
			if other, exists := position[neighbor]; exists && assignment[other] >= 0 {
				neighborColors[assignment[other]] = struct{}{}
			}
		}
		saturation := len(neighborColors)
		degree := len(nodes[node].neighbors)
		if saturation > bestSaturation ||
			(saturation == bestSaturation && degree > bestDegree) ||
			(saturation == bestSaturation && degree == bestDegree && nodes[node].weight > bestWeight) {
			best, bestSaturation, bestDegree, bestWeight = componentIndex, saturation, degree, nodes[node].weight
		}
	}
	return best
}
