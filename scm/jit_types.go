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

import "reflect"

/*
JIT Emitter Contract
====================

Each Declaration may provide a JITEmit callback:

	func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc

This callback emits machine code for the operation. The contract between
caller and emitter is as follows.

Input arguments (args):

  Each args[i] describes where the i-th operand lives at the point of the
  call. The emitter must handle all location modes:

  - LocImm:     compile-time constant. args[i].Imm holds a Scmer value;
                Imm.GetTag() carries the type. No register is allocated.
                The emitter SHOULD constant-fold when all inputs are LocImm.
  - LocReg:     unboxed primitive in args[i].Reg.
  - LocRegPair: boxed Scmer in args[i].Reg (ptr) + args[i].Reg2 (aux).
  - LocStack:   value on the stack at args[i].StackOff.
  - LocMem:     value at fixed memory address args[i].MemPtr.

  The emitter takes ownership of input registers: it MUST call
  ctx.FreeDesc(&args[i]) for every register-located input it consumes.
  Inputs in LocImm/LocStack/LocMem need no freeing.

Result placement (result):

  The result parameter tells the emitter WHERE to put its output.

  - LocAny:     emitter chooses freely. May return LocImm (best: zero code
                emitted), LocReg, or anything else. Use this when the caller
                will immediately pass the result into another emitter.
  - LocReg:     result MUST be placed into result.Reg.
  - LocRegPair: result MUST be placed into result.Reg + result.Reg2.
  - LocStack:   result MUST be written to result.StackOff.
  - LocMem:     result MUST be written to result.MemPtr.

  The emitter returns a JITValueDesc describing where the result actually
  ended up. When result.Loc != LocAny, the returned desc must match.

Constant propagation:

  When all inputs are LocImm, emitters SHOULD compute the result at
  compile time and return JITValueDesc{Loc: LocImm, Imm: <result>}
  without emitting any machine code. This enables chains of operations
  on constants to collapse to a single LocImm value.

  When result.Loc == LocAny, returning LocImm is always valid and
  preferred. When result.Loc demands a specific register or memory
  location, the emitter must still materialize the constant there
  (e.g. via EmitMakeBool/EmitMakeInt with the LocImm source).

Register discipline:

  - Allocate registers with ctx.AllocReg(), free with ctx.FreeReg(r).
  - Free consumed input registers via ctx.FreeDesc(&args[i]).
  - Never hold more registers than necessary between operations.
  - Scratch registers (R11) are reserved for internal use by emit helpers.

Generated emitters (tools/jitgen):

  The jitgen tool reads Go SSA for Declaration function bodies and
  generates JITEmit closures that follow this contract automatically.
  Run: go run ./tools/jitgen/ -patch scm/alu.go
*/

// Reg represents a hardware register index. The actual register constants
// (RAX, R8, X0, etc.) are defined in architecture-specific files.
type Reg uint8

// JITTypeUnknown means the Scmer type is not known at compile time.
// All other type values are tag constants (tagInt, tagFloat, tagBool, etc.)
// so GetTag can be constant-folded when Type != JITTypeUnknown.
const JITTypeUnknown uint16 = 0xFFFF

// JITLoc describes where a value resides during JIT compilation.
type JITLoc uint8

const (
	LocNone    JITLoc = iota // Not yet assigned
	LocReg                   // In a register (Reg) — for primitive types
	LocRegPair               // In two registers (Reg=ptr, Reg2=aux) — for Scmer
	LocStack                 // On the stack (StackOff)
	LocMem                   // At a fixed memory address (MemPtr)
	LocImm                   // Compile-time constant (Imm)
	LocAny                   // "I don't care" — result may be constant, register, or memory
)

// JITValueDesc describes a value during JIT compilation: its type and
// storage location. Flows through expression compilation for type
// propagation — analogous to optimizerMetainfo in the optimizer.
//
// Type uses the tag constants (tagInt, tagFloat, tagBool, ...) directly,
// or JITTypeUnknown (0xFFFF) when the type is not known at compile time.
// This means GetTag can be constant-folded: if Type != JITTypeUnknown,
// the tag IS Type — no machine code needed.
//
// Type resolution (fixed vs flexible):
//
//   LocImm:     ALWAYS fixed. Imm.GetTag() == Type. Constant-fold everything.
//   LocReg:     ALWAYS fixed. Unboxed primitive in a register. Type says what.
//   LocRegPair: Fixed if Type != JITTypeUnknown, flexible otherwise.
//   LocAny:     Result placement hint only ("I don't care where you put it").
type JITValueDesc struct {
	Type     uint16 // tag constant (tagInt, tagFloat, ...) or JITTypeUnknown
	Nullable bool
	Loc      JITLoc
	Reg      Reg
	Reg2     Reg     // second register (for Scmer: ptr+aux)
	StackOff int32   // stack offset (if Loc == LocStack)
	MemPtr   uintptr // memory address (if Loc == LocMem)
	Imm      Scmer   // compile-time constant (if Loc == LocImm); Imm.GetTag() carries type info
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
	SliceBase   Reg // register holding the args slice pointer (for variable-index access)
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

// JITBuildMergeClosure wraps a func(...Scmer) Scmer into func(Scmer, Scmer) Scmer.
// Called from JIT code at runtime.
func JITBuildMergeClosure(mfn func(...Scmer) Scmer) func(Scmer, Scmer) Scmer {
	return func(oldV, newV Scmer) Scmer { return mfn(oldV, newV) }
}

// GoFuncAddr returns the entry point address of a Go function value.
func GoFuncAddr(fn interface{}) uint64 {
	return uint64(reflect.ValueOf(fn).Pointer())
}

// GoABIIntRegs lists integer argument/result registers in Go internal ABI order.
var GoABIIntRegs = []Reg{RegRAX, RegRBX, RegRCX, RegRDI, RegRSI, RegR8, RegR9, RegR10, RegR11}

// EmitGoCall emits a call to a Go function from JIT code.
// argWords: registers holding argument words in Go ABI order.
// numResultWords: how many result words to capture.
// Returns registers holding the result words.
// All live JIT registers are saved/restored around the call.
func (ctx *JITContext) EmitGoCall(funcAddr uint64, argWords []Reg, numResultWords int) []Reg {
	// Collect all currently-allocated registers (not free).
	// Skip RSP/RBP (frame), R11 (call scratch), R14 (Go goroutine ptr "g").
	var liveRegs []Reg
	for r := Reg(0); r <= RegR15; r++ {
		if r == RegRSP || r == RegRBP || r == RegR11 || r == RegR14 {
			continue
		}
		if ctx.FreeRegs&(1<<uint(r)) == 0 {
			liveRegs = append(liveRegs, r)
		}
	}

	// Reserve stack space for result words (above saved registers).
	// After restoring saved regs, these slots will be at [RSP+0..].
	resultBytes := numResultWords * 8
	if resultBytes > 0 {
		ctx.W.emitBytes(0x48, 0x83, 0xEC, byte(resultBytes)) // SUB RSP, imm8
	}

	// Save live registers (PUSH)
	for _, r := range liveRegs {
		ctx.W.EmitPushReg(r)
	}
	// Align stack to 16 bytes if needed (odd total items)
	totalItems := numResultWords + len(liveRegs)
	padded := totalItems%2 == 1
	if padded {
		ctx.W.EmitPushReg(RegRAX) // dummy padding
	}

	// Move argument words into Go ABI registers.
	// Process in reverse to avoid clobbering sources.
	for i := len(argWords) - 1; i >= 0; i-- {
		target := GoABIIntRegs[i]
		if argWords[i] != target {
			ctx.W.emitMovRegReg(target, argWords[i])
		}
	}

	// CALL
	ctx.W.EmitCallIndirect(funcAddr)

	// Store results to reserved stack slots (above saved regs + padding)
	paddingSize := 0
	if padded {
		paddingSize = 8
	}
	for i := 0; i < numResultWords; i++ {
		offset := int32(paddingSize + len(liveRegs)*8 + i*8)
		ctx.W.emitStoreRegMem(GoABIIntRegs[i], RegRSP, offset)
	}

	// Restore (POP in reverse)
	if padded {
		ctx.W.EmitPopReg(RegRAX)
	}
	for i := len(liveRegs) - 1; i >= 0; i-- {
		ctx.W.EmitPopReg(liveRegs[i])
	}

	// Pop results from reserved slots into freshly allocated registers
	results := make([]Reg, numResultWords)
	for i := 0; i < numResultWords; i++ {
		r := ctx.AllocReg()
		ctx.W.EmitPopReg(r)
		results[i] = r
	}
	return results
}

// flattenArgs converts JITValueDesc arguments to ABI word registers.
// LocRegPair → 2 words (Reg, Reg2), LocReg → 1 word, LocImm → materialize.
func (ctx *JITContext) flattenArgs(args []JITValueDesc) []Reg {
	var words []Reg
	for _, a := range args {
		switch a.Loc {
		case LocRegPair:
			words = append(words, a.Reg, a.Reg2)
		case LocReg:
			words = append(words, a.Reg)
		case LocImm:
			r := ctx.AllocReg()
			ctx.W.EmitMovRegImm64(r, uint64(a.Imm.Int()))
			words = append(words, r)
		}
	}
	return words
}

// EmitGoCallScalar calls a Go function and returns a single-word result as JITValueDesc.
func (ctx *JITContext) EmitGoCallScalar(funcAddr uint64, args []JITValueDesc, numResultWords int) JITValueDesc {
	words := ctx.flattenArgs(args)
	results := ctx.EmitGoCall(funcAddr, words, numResultWords)
	if numResultWords == 1 {
		return JITValueDesc{Loc: LocReg, Reg: results[0]}
	}
	return JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: results[0], Reg2: results[1]}
}

// EmitMovPairToResult moves a LocRegPair value into the result descriptor registers.
func (ctx *JITContext) EmitMovPairToResult(src *JITValueDesc, dst *JITValueDesc) {
	if src.Reg != dst.Reg {
		ctx.W.emitMovRegReg(dst.Reg, src.Reg)
	}
	if src.Reg2 != dst.Reg2 {
		ctx.W.emitMovRegReg(dst.Reg2, src.Reg2)
	}
}

// EmitGoCallVoid calls a Go function with no return value.
func (ctx *JITContext) EmitGoCallVoid(funcAddr uint64, args []JITValueDesc) {
	words := ctx.flattenArgs(args)
	ctx.EmitGoCall(funcAddr, words, 0)
}
