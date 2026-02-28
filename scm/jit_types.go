/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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

// Reg represents a hardware register index. The actual register constants
// (RAX, R8, X0, etc.) are defined in architecture-specific files.
type Reg uint8

// JITType describes the known type of a value during JIT compilation.
type JITType uint8

const (
	JITTypeUnknown JITType = iota // Scmer, no specific type
	JITTypeInt64                  // Guaranteed int64
	JITTypeFloat64                // Guaranteed float64
	JITTypeString                 // Guaranteed string
	JITTypeBool                   // Guaranteed bool
)

// JITLoc describes where a value resides during JIT compilation.
type JITLoc uint8

const (
	LocNone    JITLoc = iota // Not yet assigned
	LocReg                   // In a register (Reg) — for primitive types
	LocRegPair               // In two registers (Reg=ptr, Reg2=aux) — for Scmer
	LocStack                 // On the stack (StackOff)
	LocMem                   // At a fixed memory address (MemPtr)
	LocImm                   // Compile-time constant (Imm)
)

// JITValueDesc describes a value during JIT compilation: its type and
// storage location. Flows through expression compilation for type
// propagation — analogous to optimizerMetainfo in the optimizer.
type JITValueDesc struct {
	Type     JITType
	Nullable bool
	Loc      JITLoc
	Reg      Reg
	Reg2     Reg     // second register (for Scmer: ptr+aux)
	StackOff int32   // stack offset (if Loc == LocStack)
	MemPtr   uintptr // memory address (if Loc == LocMem)
	Imm      int64   // immediate value (if Loc == LocImm)
}

// JITFixup records a forward reference that must be patched after all
// labels are placed.
type JITFixup struct {
	CodePos  int32 // position in code
	LabelID  uint8 // target label
	Size     uint8 // 1=rel8, 4=rel32
	Relative bool  // true for PC-relative jumps
}

// JITEnv manages variable descriptors during JIT compilation (like Env
// but for compile-time tracking of types and locations).
type JITEnv struct {
	Vars      map[Symbol]JITValueDesc
	Numbered  []JITValueDesc
	Outer     *JITEnv
	StackBase int32
}

// Lookup resolves a symbol through the scope chain.
func (env *JITEnv) Lookup(sym Symbol) (JITValueDesc, bool) {
	if desc, ok := env.Vars[sym]; ok {
		return desc, true
	}
	if env.Outer != nil {
		return env.Outer.Lookup(sym)
	}
	return JITValueDesc{}, false
}

// Set stores a variable descriptor in the current scope.
func (env *JITEnv) Set(sym Symbol, desc JITValueDesc) {
	env.Vars[sym] = desc
}

// JITContext is the central structure for descriptor-based JIT compilation.
type JITContext struct {
	Env         *JITEnv
	FreeRegs    uint64
	W           *JITWriter
	StackOffset int32
}

// AllocReg picks a free register from the bitmap and marks it used.
func (ctx *JITContext) AllocReg() Reg {
	if ctx.FreeRegs == 0 {
		panic("jit: no free registers")
	}
	// find lowest set bit
	bit := ctx.FreeRegs & (-ctx.FreeRegs)
	ctx.FreeRegs &^= bit
	r := Reg(0)
	for b := bit; b > 1; b >>= 1 {
		r++
	}
	return r
}

// FreeReg returns a register to the free pool.
func (ctx *JITContext) FreeReg(r Reg) {
	ctx.FreeRegs |= 1 << uint(r)
}

// FreeDesc releases any registers held by a value descriptor.
func (ctx *JITContext) FreeDesc(desc *JITValueDesc) {
	switch desc.Loc {
	case LocReg:
		ctx.FreeReg(desc.Reg)
	case LocRegPair:
		ctx.FreeReg(desc.Reg)
		ctx.FreeReg(desc.Reg2)
	}
	desc.Loc = LocNone
}
