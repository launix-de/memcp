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
package main

import "golang.org/x/tools/go/ssa"

// SSAValueRef is an opaque backend-owned handle for an SSA value.
// Frontends must never inspect physical register state from this handle.
type SSAValueRef uint32

// SSARegRef is an abstract architecture register handle.
type SSARegRef uint16

// SSARegClass captures logical register classes used by backends.
type SSARegClass uint8

const (
	SSARegClassGPR SSARegClass = iota
	SSARegClassFPR
	SSARegClassPair
)

// SSAEmitterHelpers contains only data-movement/legalization helpers.
// These helpers prepare operands for instruction-specific emitters.
type SSAEmitterHelpers interface {
	MoveToClass(v SSAValueRef, class SSARegClass) (SSAValueRef, error)
	MoveToReg(v SSAValueRef, reg SSARegRef) (SSAValueRef, error)
	MoveToPair(v SSAValueRef, reg1, reg2 SSARegRef) (SSAValueRef, error)
	Spill(v SSAValueRef) error
	Kill(v SSAValueRef) error
}

// SSAArchEmitter maps SSA instructions 1:1 to backend emitters.
// Exactly one method should be called per SSA instruction node.
type SSAArchEmitter interface {
	Helpers() SSAEmitterHelpers

	BeginFunction(fn *ssa.Function) error
	EndFunction() error
	BeginBlock(b *ssa.BasicBlock) error
	EndBlock(b *ssa.BasicBlock) error

	EmitConst(v *ssa.Const) (SSAValueRef, error)
	EmitBinOp(v *ssa.BinOp, x, y SSAValueRef) (SSAValueRef, error)
	EmitUnOp(v *ssa.UnOp, x SSAValueRef) (SSAValueRef, error)
	EmitConvert(v *ssa.Convert, x SSAValueRef) (SSAValueRef, error)
	EmitCall(v *ssa.Call, args []SSAValueRef) (SSAValueRef, error)
	EmitFieldAddr(v *ssa.FieldAddr, base SSAValueRef) (SSAValueRef, error)
	EmitIndexAddr(v *ssa.IndexAddr, base, index SSAValueRef) (SSAValueRef, error)
	EmitPhi(v *ssa.Phi, edges []SSAValueRef) (SSAValueRef, error)

	EmitIf(v *ssa.If, cond SSAValueRef) error
	EmitJump(v *ssa.Jump) error
	EmitReturn(v *ssa.Return, results []SSAValueRef) error
	EmitStore(v *ssa.Store, addr, value SSAValueRef) error
}
