/*
	Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; see <https://www.gnu.org/licenses/>.
*/
package main

import "go/token"
import "go/constant"
import "golang.org/x/tools/go/ssa"

// zeroRemainder proves a value's low bits from SSA edges, not from a recognized
// loop syntax. A phi may close an inductive proof only with the SAME residue;
// every executable entry and backedge must satisfy it. For example 0,+2 keeps
// bit zero clear even though the induction value itself is dynamic. Nothing is
// unrolled and the runtime one-pass emitter receives only the resulting fact.
//
// Powers of two are deliberate: integer overflow preserves their congruence.
// It does not preserve divisibility by an arbitrary divisor (e.g. uint8 +3).
// Only remainder ZERO is inferred for signed values, so negative inputs retain
// Go's remainder semantics. Zero divisors and unsupported proofs stay dynamic.
func (r *ssaReachability) zeroRemainder(v *ssa.BinOp) bool {
	if v.Op != token.REM {
		return false
	}
	divisor := r.constant(v.Y)
	if divisor == nil || divisor.Value == nil {
		return false
	}
	divisorWord, exact := constant.Uint64Val(divisor.Value)
	if !exact {
		signed, ok := constant.Int64Val(divisor.Value)
		if !ok {
			return false
		}
		divisorWord = uint64(signed)
		if signed < 0 {
			divisorWord = 0 - divisorWord
		}
	}
	if divisorWord == 0 || divisorWord&(divisorWord-1) != 0 {
		return false
	}
	mask := divisorWord - 1
	if mask == 0 {
		return true
	}
	active := make(map[ssa.Value]uint64)
	var prove func(ssa.Value, uint64) bool
	prove = func(value ssa.Value, residue uint64) bool {
		residue &= mask
		if known := r.constant(value); known != nil && known.Value != nil && known.Value.Kind() == constant.Int {
			return uint64(constWordValue(known.Value))&mask == residue
		}
		if previous, cyclic := active[value]; cyclic {
			return previous == residue
		}
		active[value] = residue
		defer delete(active, value)
		switch value := value.(type) {
		case *ssa.Phi:
			seen := false
			for i, edge := range value.Edges {
				if r.incoming(value.Block(), i) {
					seen = true
					if !prove(edge, residue) {
						return false
					}
				}
			}
			return seen
		case *ssa.BinOp:
			left, right := value.X, r.constant(value.Y)
			if right == nil && (value.Op == token.ADD || value.Op == token.MUL || value.Op == token.AND) {
				left, right = value.Y, r.constant(value.X)
			}
			if right == nil || right.Value == nil || right.Value.Kind() != constant.Int {
				return false
			}
			word := uint64(constWordValue(right.Value))
			switch value.Op {
			case token.ADD:
				return prove(left, residue-word)
			case token.SUB:
				return prove(left, residue+word)
			case token.AND:
				if word&mask == 0 {
					return residue == 0
				}
				if word&mask == mask {
					return prove(left, residue)
				}
			case token.MUL:
				if word&mask == 0 {
					return residue == 0
				}
			}
		case *ssa.ChangeType:
			return prove(value.X, residue)
		case *ssa.Convert:
			_, width, integer := intTypeInfo(value.X.Type())
			if integer && (width == 64 || mask < uint64(1)<<uint(width)) {
				return prove(value.X, residue)
			}
		}
		return false
	}
	return prove(v.X, 0)
}
