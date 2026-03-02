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

// JITTagMask is a bitset of possible runtime tags for a value.
// A single set bit means "known tag", multiple bits mean union of possibilities.
type JITTagMask uint32

const (
	JITTagMaskNil JITTagMask = 1 << iota
	JITTagMaskBool
	JITTagMaskInt
	JITTagMaskFloat
	JITTagMaskString
	JITTagMaskSlice
	JITTagMaskFastDict
)

const JITTagMaskAny = JITTagMaskNil | JITTagMaskBool | JITTagMaskInt |
	JITTagMaskFloat | JITTagMaskString | JITTagMaskSlice | JITTagMaskFastDict

// JITTypeFacts tracks compile-time knowledge for a value.
// More knowledge should allow emitters to generate less machine code.
type JITTypeFacts struct {
	// Possible is the current tag set. JITTagMaskAny means fully unknown.
	Possible JITTagMask
	// When HasConst is true, Const holds an exact compile-time value.
	HasConst bool
	Const    Scmer
}

// JITValueKind describes how a value currently exists in the backend pipeline.
type JITValueKind uint8

const (
	JITValueImm JITValueKind = iota
	JITValueVirtual
	JITValueMaterialized
)

// JITValueRef is an opaque value handle owned by the backend.
type JITValueRef struct {
	ID    uint32
	Kind  JITValueKind
	Facts JITTypeFacts
}

// JITRegRef is an abstract register handle used by architecture backends.
type JITRegRef uint16

// JITPlaceKind describes concrete placement requirements for materialization.
type JITPlaceKind uint8

const (
	JITPlaceAny JITPlaceKind = iota
	JITPlaceReg
	JITPlacePair
	JITPlaceStack
	JITPlaceMem
	JITPlaceImm
)

// JITPlace describes a concrete destination or source location.
type JITPlace struct {
	Kind JITPlaceKind
	Reg  JITRegRef
	Reg2 JITRegRef
	Slot int32
	Mem  uintptr
}

// JITRegClass defines logical register classes.
type JITRegClass uint8

const (
	JITRegClassGPR JITRegClass = iota
	JITRegClassFPR
	JITRegClassPair
)

// JITPinToken is a stable protection handle independent of mutable descriptors.
type JITPinToken struct {
	ID uint32
}

// JITAllocScope is a temporary-allocation scope.
type JITAllocScope interface {
	Release()
}

// JITAllocator owns virtual->physical mapping, spill decisions, and liveness.
// Frontends should never reason about physical registers directly.
type JITAllocator interface {
	Scope() JITAllocScope
	Temp(class JITRegClass, except ...JITRegRef) (JITRegRef, error)
	Ensure(v JITValueRef, class JITRegClass, except ...JITRegRef) (JITRegRef, error)
	Pin(v JITValueRef) (JITPinToken, error)
	Unpin(tok JITPinToken) error
	Kill(v JITValueRef) error
	Validate() error
}

// JITBlockLabel is an architecture-neutral label token.
type JITBlockLabel uint32

// JITIntOp are integer ALU operations.
type JITIntOp uint8

const (
	JITIntAdd JITIntOp = iota
	JITIntSub
	JITIntMul
	JITIntOr
	JITIntAnd
)

// JITCmpPred are integer compare predicates.
type JITCmpPred uint8

const (
	JITCmpEQ JITCmpPred = iota
	JITCmpNE
	JITCmpLT
	JITCmpLE
	JITCmpGT
	JITCmpGE
)

// JITShiftDir describes logical shift direction.
type JITShiftDir uint8

const (
	JITShiftLeft JITShiftDir = iota
	JITShiftRight
)

// JITDivRemKind selects quotient or remainder result.
type JITDivRemKind uint8

const (
	JITDivResult JITDivRemKind = iota
	JITRemResult
)

// JITArchEmitter lowers semantic operations to target machine code.
// All register movement details stay backend-internal.
type JITArchEmitter interface {
	Allocator() JITAllocator

	BeginFunction(name string) error
	EndFunction() error

	NewLabel() JITBlockLabel
	MarkLabel(label JITBlockLabel) error
	EmitJump(label JITBlockLabel) error
	EmitBranch(cond JITValueRef, thenLabel, elseLabel JITBlockLabel) error

	EmitConst(v Scmer) (JITValueRef, error)
	EmitMove(dst JITPlace, src JITValueRef) (JITValueRef, error)
	EmitLoad(base JITValueRef, offset int32, class JITRegClass) (JITValueRef, error)
	EmitStore(base JITValueRef, offset int32, src JITValueRef) error

	EmitIntBinOp(op JITIntOp, x, y JITValueRef) (JITValueRef, error)
	EmitIntCmp(pred JITCmpPred, x, y JITValueRef) (JITValueRef, error)
	EmitIntShift(dir JITShiftDir, x, amount JITValueRef) (JITValueRef, error)
	EmitIntDivRem(kind JITDivRemKind, x, y JITValueRef) (JITValueRef, error)

	EmitMakeScmer(tag uint16, payload JITValueRef, dst JITPlace) (JITValueRef, error)
}
