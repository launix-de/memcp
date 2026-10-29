/*
	Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; see <https://www.gnu.org/licenses/>.
*/
package main

import "go/token"
import "go/types"
import "go/constant"
import "golang.org/x/tools/go/ssa"

// ssaReachability is an immutable specialization of a function, never a mutation
// of the shared SSA graph. Edges and value facts grow monotonically: unknown,
// constant, dynamic. Only executable incoming edges contribute to a Phi. In
// particular, the first iteration of a loop cannot prove its backedge dead.
// This proof precedes instruction support checks and machine-code lowering;
// an unsupported operation in a dead block cannot force a native fallback.
type ssaReachability struct {
	blocks map[*ssa.BasicBlock]bool
	edges  map[ssaControlEdge]bool
	facts  map[ssa.Value]ssaConstantFact
}
type ssaControlEdge struct {
	from      *ssa.BasicBlock
	successor int
}

type ssaConstantFact struct {
	value   *ssa.Const
	dynamic bool
}

func joinSSAConstant(a, b ssaConstantFact) ssaConstantFact {
	if a.dynamic || b.dynamic {
		return ssaConstantFact{dynamic: true}
	}
	if a.value == nil {
		return b
	}
	if b.value == nil {
		return a
	}
	if a.value.Value == nil && b.value.Value == nil {
		return a
	}
	if a.value.Value != nil && b.value.Value != nil && constant.Compare(a.value.Value, token.EQL, b.value.Value) {
		return a
	}
	return ssaConstantFact{dynamic: true}
}
func (r *ssaReachability) fact(v ssa.Value) ssaConstantFact {
	if c, ok := v.(*ssa.Const); ok {
		return ssaConstantFact{value: c}
	}
	if f, ok := r.facts[v]; ok {
		return f
	}
	if _, instruction := v.(ssa.Instruction); !instruction {
		return ssaConstantFact{dynamic: true}
	}
	return ssaConstantFact{}
}
func (r *ssaReachability) constant(v ssa.Value) *ssa.Const { return r.fact(v).value }
func (r *ssaReachability) successor(v *ssa.If) (int, bool) {
	c := r.constant(v.Cond)
	if c == nil || c.Value == nil || c.Value.Kind() != constant.Bool {
		return 0, false
	}
	if constant.BoolVal(c.Value) {
		return 0, true
	}
	return 1, true
}
func analyzeSSAReachability(fn *ssa.Function, bindings map[ssa.Value]*ssa.Const) *ssaReachability {
	r := &ssaReachability{blocks: make(map[*ssa.BasicBlock]bool), edges: make(map[ssaControlEdge]bool), facts: make(map[ssa.Value]ssaConstantFact)}
	if fn == nil || len(fn.Blocks) == 0 {
		return r
	}
	for _, p := range fn.Params {
		if c := bindings[p]; c != nil {
			r.facts[p] = ssaConstantFact{value: c}
		} else {
			r.facts[p] = ssaConstantFact{dynamic: true}
		}
	}
	for _, p := range fn.FreeVars {
		if c := bindings[p]; c != nil {
			r.facts[p] = ssaConstantFact{value: c}
		} else {
			r.facts[p] = ssaConstantFact{dynamic: true}
		}
	}
	for value, c := range bindings {
		if c != nil {
			r.facts[value] = ssaConstantFact{value: c}
		}
	}
	r.blocks[fn.Blocks[0]] = true
	changed := true
	for changed {
		changed = false
		edge := func(from *ssa.BasicBlock, successor int) {
			to := from.Succs[successor]
			key := ssaControlEdge{from, successor}
			if !r.edges[key] {
				r.edges[key] = true
				r.blocks[to] = true
				changed = true
			}
		}
		for _, b := range fn.Blocks {
			if !r.blocks[b] {
				continue
			}
			for _, instr := range b.Instrs {
				if value, ok := instr.(ssa.Value); ok {
					if bindings[value] != nil {
						continue
					}
					previous := r.facts[value]
					next := joinSSAConstant(previous, r.evaluate(value))
					if next.dynamic != previous.dynamic || next.value != previous.value {
						// Equal constants retain the old pointer, so the lattice converges.
						r.facts[value] = next
						changed = true
					}
				}
			}
			switch end := b.Instrs[len(b.Instrs)-1].(type) {
			case *ssa.If:
				if choice, known := r.successor(end); known {
					edge(b, choice)
				} else if r.fact(end.Cond).dynamic {
					for index := range b.Succs {
						edge(b, index)
					}
				}
			case *ssa.Jump:
				edge(b, 0)
			}
		}
	}
	return r
}

// Parallel edges can have the same predecessor and destination but distinct
// phi inputs. Identify an executable edge by its successor position, never by
// the two blocks alone.
func (r *ssaReachability) incoming(block *ssa.BasicBlock, predecessor int) bool {
	from := block.Preds[predecessor]
	occurrence := 0
	for _, prior := range block.Preds[:predecessor] {
		if prior == from {
			occurrence++
		}
	}
	for successor, to := range from.Succs {
		if to != block {
			continue
		}
		if occurrence == 0 {
			return r.edges[ssaControlEdge{from, successor}]
		}
		occurrence--
	}
	return false
}

func (r *ssaReachability) evaluate(value ssa.Value) ssaConstantFact {
	switch v := value.(type) {
	case *ssa.Phi:
		var result ssaConstantFact
		for i := range v.Block().Preds {
			if r.incoming(v.Block(), i) {
				result = joinSSAConstant(result, r.fact(v.Edges[i]))
			}
		}
		return result
	case *ssa.BinOp:
		if r.zeroRemainder(v) {
			return ssaConstantFact{value: ssa.NewConst(constant.MakeInt64(0), v.Type())}
		}
		a, b := r.fact(v.X), r.fact(v.Y)
		if a.dynamic || b.dynamic {
			return ssaConstantFact{dynamic: true}
		}
		if a.value == nil || b.value == nil {
			return ssaConstantFact{}
		}
		return foldSSAConstant(v.Type(), v.Op, a.value, b.value)
	case *ssa.UnOp:
		a := r.fact(v.X)
		if a.dynamic || a.value == nil {
			return a
		}
		if v.Op == token.NOT || v.Op == token.SUB || v.Op == token.XOR || v.Op == token.ADD {
			return foldSSAConstant(v.Type(), v.Op, a.value, nil)
		}
	case *ssa.ChangeType:
		a := r.fact(v.X)
		if a.value != nil {
			a.value = ssa.NewConst(a.value.Value, v.Type())
		}
		return a
	case *ssa.Convert:
		a := r.fact(v.X)
		if a.dynamic || a.value == nil {
			return a
		}
		if basic, ok := v.Type().Underlying().(*types.Basic); ok && basic.Info()&types.IsInteger != 0 && a.value.Value != nil && a.value.Value.Kind() == constant.Int {
			return ssaConstantFact{value: ssa.NewConst(canonicalSSAInteger(a.value.Value, v.Type()), v.Type())}
		}
	}
	return ssaConstantFact{dynamic: true}
}

// Fold only Go integer/boolean operations. Floating point arithmetic remains
// with the existing typed emitter, which owns IEEE rounding and NaN behavior.
// Operations which panic (division by zero, negative shifts) stay executable.
func foldSSAConstant(typ types.Type, op token.Token, a, b *ssa.Const) (result ssaConstantFact) {
	result.dynamic = true
	defer func() {
		if recover() != nil {
			result = ssaConstantFact{dynamic: true}
		}
	}()
	var value constant.Value
	if b != nil && (op == token.EQL || op == token.NEQ) && (a.Value == nil || b.Value == nil) {
		equal := a.Value == nil && b.Value == nil
		if op == token.NEQ {
			equal = !equal
		}
		value = constant.MakeBool(equal)
	} else {
		if a.Value == nil || (a.Value.Kind() != constant.Int && a.Value.Kind() != constant.Bool) {
			return
		}
		if b == nil {
			value = constant.UnaryOp(op, a.Value, 0)
		} else {
			if b.Value == nil || (b.Value.Kind() != constant.Int && b.Value.Kind() != constant.Bool) {
				return
			}
			switch op {
			case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
				value = constant.MakeBool(constant.Compare(a.Value, op, b.Value))
			case token.SHL, token.SHR:
				shift, exact := constant.Uint64Val(b.Value)
				if !exact {
					return
				}
				if shift > 64 {
					shift = 64
				}
				value = constant.Shift(a.Value, op, uint(shift))
			default:
				if op == token.QUO {
					op = token.QUO_ASSIGN
				}
				value = constant.BinaryOp(a.Value, op, b.Value)
			}
		}
	}
	if value.Kind() == constant.Int {
		value = canonicalSSAInteger(value, typ)
	}
	result = ssaConstantFact{value: ssa.NewConst(value, typ)}
	return
}
func canonicalSSAInteger(value constant.Value, typ types.Type) constant.Value {
	signed, bits, integer := intTypeInfo(typ)
	if !integer {
		return value
	}
	modulus := constant.Shift(constant.MakeInt64(1), token.SHL, uint(bits))
	value = constant.BinaryOp(value, token.REM, modulus)
	if constant.Sign(value) < 0 {
		value = constant.BinaryOp(value, token.ADD, modulus)
	}
	if signed && constant.Compare(value, token.GEQ, constant.Shift(constant.MakeInt64(1), token.SHL, uint(bits-1))) {
		value = constant.BinaryOp(value, token.SUB, modulus)
	}
	return value
}

// replacement removes identity phis after executable-edge pruning, including
// slices and other compiler-only values which cannot be represented as scalar
// constants. Loop-carried differing values retain their ordinary phi home.
// replacement preserves identities as well as constants. In particular, an
// untaken optional slice trim must not turn a virtual argument directory into
// a materialized phi. Loop-carried changes remain distinct incoming values.
func (r *ssaReachability) replacement(value ssa.Value) ssa.Value {
	if c := r.constant(value); c != nil {
		return c
	}
	if _, phi := value.(*ssa.Phi); !phi {
		return value
	}
	seen := make(map[ssa.Value]bool)
	for !seen[value] {
		seen[value] = true
		if c := r.constant(value); c != nil {
			return c
		}
		phi, ok := value.(*ssa.Phi)
		if !ok {
			return value
		}
		var only ssa.Value
		for i, edge := range phi.Edges {
			if !r.incoming(phi.Block(), i) || edge == phi {
				continue
			}
			if only != nil && only != edge {
				return value
			}
			only = edge
		}
		if only == nil {
			return value
		}
		value = only
	}
	return value
}
