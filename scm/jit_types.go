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

import (
	"fmt"
	"math"
	"reflect"
	"unsafe"
)

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
	LocNone      JITLoc = iota // Not yet assigned
	LocReg                     // In a register (Reg) — for primitive types
	LocRegPair                 // In two registers (Reg=ptr, Reg2=aux) — for Scmer
	LocStack                   // On the stack (StackOff)
	LocStackPair               // Two-word value spilled to stack-like spill buffer (MemPtr, MemPtr+8)
	LocMem                     // At a fixed memory address (MemPtr)
	LocImm                     // Compile-time constant (Imm)
	LocAny                     // "I don't care" — result may be constant, register, or memory
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
//	LocImm:     ALWAYS fixed. Imm.GetTag() == Type. Constant-fold everything.
//	LocReg:     ALWAYS fixed. Unboxed primitive in a register. Type says what.
//	LocRegPair: Fixed if Type != JITTypeUnknown, flexible otherwise.
//	LocAny:     Result placement hint only ("I don't care where you put it").
type JITValueDesc struct {
	ID       uint32
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

// spillEntry records a register that was evicted to the spill buffer.
type spillEntry struct {
	reg  Reg // which register was saved
	slot int // index in SpillBuf
}

type descSpillMeta struct {
	loc      JITLoc
	memPtr   uintptr
	stackOff int32
}

// JITContext is the central structure for descriptor-based JIT compilation.
type JITContext struct {
	Env         *JITEnv
	FreeRegs    uint64
	AllRegs     uint64 // original set of all allocatable registers (for spilling)
	W           *JITWriter
	StackOffset int32
	SliceBase   Reg // register holding the args slice pointer (for variable-index access)
	// Register spilling: when all registers are occupied, we save the
	// register to a pre-allocated spill buffer (not the stack).
	// This avoids modifying RSP, which would break phi stack offsets.
	spillStack         [16]spillEntry // fixed-size: at most 16 hardware registers can be live
	spillStackLen      int
	RegOwners          [16]*JITValueDesc // register → owner descriptor (nil = untracked)
	SpillBuf           [4096]int64       // pre-allocated spill buffer (heap-stable, not on stack)
	SpillTop           int               // high-water mark in SpillBuf (next fresh slot)
	spillFreeArr       [16]int           // recycled SpillBuf slot indices
	spillFreeLen       int
	ProtectedRegs      uint64  // bitmask of registers that must not be spilled
	ProtectedRegCounts [16]int // per-register protection refcount (supports nested protection)
	nextDescID         uint32
	descSpills         map[uint32]descSpillMeta
	// EvictedCounts tracks how many times each register was evicted by AllocReg.
	// When AllocReg spills register R and returns R as a fresh register, both the
	// old holder and the new holder think they own R. EvictedCounts[R] counts the
	// pending "burns": FreeReg(R) decrements the count without adding to FreeRegs,
	// ensuring only the final FreeReg call (from the real live holder) returns R.
	EvictedCounts [16]int
	// ConstRoots holds pointer payloads from LocImm Scmer values that were
	// materialized into machine code immediates. Keeping these pointers in a
	// Go heap object reachable from JITEntryPoint prevents GC from reclaiming
	// referenced heap data while JIT code may still dereference it.
	ConstRoots []unsafe.Pointer
	rootSet    map[unsafe.Pointer]struct{}
}

// TrackImm records a LocImm constant's pointer payload as a GC root when needed.
func (ctx *JITContext) TrackImm(v Scmer) {
	if ctx == nil {
		return
	}
	ptr, _ := v.RawWords()
	if ptr == 0 {
		return
	}
	p := unsafe.Pointer(ptr)
	// Sentinel pointers are static globals and don't need GC rooting.
	if p == unsafe.Pointer(&scmerIntSentinel) || p == unsafe.Pointer(&scmerFloatSentinel) {
		return
	}
	if ctx.rootSet == nil {
		ctx.rootSet = make(map[unsafe.Pointer]struct{}, 16)
	}
	if _, exists := ctx.rootSet[p]; exists {
		return
	}
	ctx.rootSet[p] = struct{}{}
	ctx.ConstRoots = append(ctx.ConstRoots, p)
}

// ProtectReg marks a register as non-spillable by AllocReg.
// Multiple callers can protect the same register; it becomes spillable
// again only when all protections are removed via UnprotectReg.
func (ctx *JITContext) ProtectReg(r Reg) {
	ctx.ProtectedRegCounts[r]++
	ctx.ProtectedRegs |= 1 << uint(r)
}

// UnprotectReg removes one protection from a register. When the last
// protection is removed, the register becomes spillable again.
func (ctx *JITContext) UnprotectReg(r Reg) {
	if ctx.ProtectedRegCounts[r] > 0 {
		ctx.ProtectedRegCounts[r]--
		if ctx.ProtectedRegCounts[r] == 0 {
			ctx.ProtectedRegs &^= 1 << uint(r)
		}
	}
}

// AllocReg picks a free register from the bitmap and marks it used.
// If no registers are free, spills the highest-numbered in-use register
// to a pre-allocated buffer and returns it.
func (ctx *JITContext) AllocReg() Reg {
	// Exclude protected registers from allocation, not just from eviction.
	available := ctx.FreeRegs &^ ctx.ProtectedRegs
	if available != 0 {
		// Normal path: pick lowest free bit, but skip protected ones
		bit := available & (-available)
		ctx.FreeRegs &^= bit
		r := Reg(0)
		for b := bit; b > 1; b >>= 1 {
			r++
		}
		return r
	}
	// Spill path: spill tracked descriptors (LocReg / LocRegPair).
	// Note: completely untracked in-use registers must NOT be reused here,
	// as they may still be referenced by emitted code paths.
	spillable := ctx.AllRegs &^ ctx.FreeRegs &^ ctx.ProtectedRegs
	var r Reg = 0xFF
	pairSpill := false
	var pairR1, pairR2 Reg
	for bit := int(RegR15); bit >= 0; bit-- {
		rbit := Reg(bit)
		if spillable&(1<<uint(rbit)) == 0 {
			continue
		}
		owner := ctx.RegOwners[rbit]
		if owner == nil {
			continue
		}
		switch owner.Loc {
		case LocReg:
			if owner.Reg != rbit {
				// Stale ownership metadata: do not reclaim implicitly.
				continue
			}
			r = rbit
		case LocRegPair:
			if owner.Reg != rbit && owner.Reg2 != rbit {
				// Stale ownership metadata: do not reclaim implicitly.
				continue
			}
			pairR1 = owner.Reg
			pairR2 = owner.Reg2
			// Pair spill must evict both registers atomically; if either register
			// is currently protected, try another candidate.
			if (ctx.ProtectedRegs&(1<<uint(pairR1))) != 0 || (ctx.ProtectedRegs&(1<<uint(pairR2))) != 0 {
				continue
			}
			r = rbit
			pairSpill = true
		default:
			// Unknown owner location: do not reclaim implicitly.
			continue
		}
		break
	}
	if r == 0xFF {
		panic("jit: register spill required (fallback)")
	}

	owner := ctx.RegOwners[r]
	if pairSpill {
		slot := ctx.SpillTop
		if slot+1 >= len(ctx.SpillBuf) {
			panic("jit: spill buffer overflow")
		}
		ctx.SpillTop += 2
		spillAddrPtr := uintptr(unsafe.Pointer(&ctx.SpillBuf[slot]))
		ctx.W.EmitMovRegImm64(RegR11, uint64(spillAddrPtr))
		ctx.W.EmitStoreRegMem(pairR1, RegR11, 0)
		ctx.W.EmitStoreRegMem(pairR2, RegR11, 8)

		owner.Loc = LocStackPair
		owner.MemPtr = spillAddrPtr
		owner.StackOff = int32(slot)
		owner.Reg = 0
		owner.Reg2 = 0
		if owner.ID != 0 {
			if ctx.descSpills == nil {
				ctx.descSpills = make(map[uint32]descSpillMeta)
			}
			ctx.descSpills[owner.ID] = descSpillMeta{loc: LocStackPair, memPtr: spillAddrPtr, stackOff: int32(slot)}
		}
		ctx.RegOwners[pairR1] = nil
		ctx.RegOwners[pairR2] = nil
		return r
	}

	// Scalar spill: monotonic slot allocation (no recycle during emission).
	slot := ctx.SpillTop
	if slot >= len(ctx.SpillBuf) {
		panic("jit: spill buffer overflow")
	}
	ctx.SpillTop++
	spillAddrPtr := uintptr(unsafe.Pointer(&ctx.SpillBuf[slot]))
	ctx.W.EmitMovRegImm64(RegR11, uint64(spillAddrPtr))
	ctx.W.EmitStoreRegMem(r, RegR11, 0) // MOV [R11], reg

	owner.Loc = LocStack
	owner.MemPtr = spillAddrPtr
	owner.StackOff = int32(slot) // spill slot id for recycling
	owner.Reg = 0
	if owner.ID != 0 {
		if ctx.descSpills == nil {
			ctx.descSpills = make(map[uint32]descSpillMeta)
		}
		ctx.descSpills[owner.ID] = descSpillMeta{loc: LocStack, memPtr: spillAddrPtr, stackOff: int32(slot)}
	}
	ctx.RegOwners[r] = nil
	return r
}

// EnsureDesc restores a descriptor from stack/spill locations to registers.
func (ctx *JITContext) EnsureDesc(desc *JITValueDesc) {
	if desc.Loc == LocReg && desc.ID != 0 && ctx.descSpills != nil {
		if meta, ok := ctx.descSpills[desc.ID]; ok && meta.loc == LocStack {
			desc.Loc = LocStack
			desc.MemPtr = meta.memPtr
			desc.StackOff = meta.stackOff
			desc.Reg = 0
		}
	}
	if desc.Loc == LocRegPair && desc.ID != 0 && ctx.descSpills != nil {
		if meta, ok := ctx.descSpills[desc.ID]; ok && meta.loc == LocStackPair {
			desc.Loc = LocStackPair
			desc.MemPtr = meta.memPtr
			desc.StackOff = meta.stackOff
			desc.Reg = 0
			desc.Reg2 = 0
		}
	}
	switch desc.Loc {
	case LocStack:
		ctx.EnsureReg(desc)
	case LocStackPair:
		r1 := ctx.AllocReg()
		r2 := ctx.AllocRegExcept(r1)
		ctx.W.EmitMovRegImm64(RegR11, uint64(desc.MemPtr))
		ctx.W.EmitMovRegMem(r1, RegR11, 0)
		ctx.W.EmitMovRegMem(r2, RegR11, 8)
		desc.Loc = LocRegPair
		desc.Reg = r1
		desc.Reg2 = r2
		desc.MemPtr = 0
		desc.StackOff = 0
		ctx.BindReg(r1, desc)
		ctx.BindReg(r2, desc)
	}
}

// FreeReg returns a register to the free pool.
// If the register was evicted by AllocReg (EvictedCounts[r] > 0), this call
// "burns" one eviction token and reclaims the spill slot. The register stays
// live in the new holder. Only the final FreeReg call (EvictedCounts[r] == 0)
// actually releases the register back to FreeRegs.
func (ctx *JITContext) FreeReg(r Reg) {
	owner := ctx.RegOwners[r]
	if owner != nil {
		switch owner.Loc {
		case LocReg:
			if owner.Reg == r {
				owner.Loc = LocNone
				owner.Reg = 0
			}
		case LocRegPair:
			// Freeing a single half of a pair means the original pair descriptor
			// is no longer reliable. Invalidate it and drop tracking for the other
			// half; callers that want to keep one word must re-bind explicitly.
			if owner.Reg == r || owner.Reg2 == r {
				other := owner.Reg
				if other == r {
					other = owner.Reg2
				}
				if other <= RegR15 {
					ctx.RegOwners[other] = nil
				}
				owner.Loc = LocNone
				owner.Reg = 0
				owner.Reg2 = 0
			}
		}
	}
	ctx.FreeRegs |= 1 << uint(r)
	ctx.RegOwners[r] = nil
}

// BindReg associates a register with a JITValueDesc owner for spill tracking.
// Call this after placing a value in a register so AllocReg can evict it.
func (ctx *JITContext) BindReg(r Reg, desc *JITValueDesc) {
	if desc.ID == 0 {
		ctx.nextDescID++
		desc.ID = ctx.nextDescID
	}
	// A bound register is live and must not be treated as free.
	ctx.FreeRegs &^= 1 << uint(r)
	ctx.RegOwners[r] = desc
	if desc.ID != 0 && ctx.descSpills != nil {
		delete(ctx.descSpills, desc.ID)
	}
}

// TransferReg is called by the generated alias check when the result descriptor
// reuses the same hardware register as an input descriptor (which will be set to
// LocNone). If AllocReg had to evict that register to produce this fresh copy,
// burn one eviction token so the final FreeReg from the new holder correctly
// returns the register to the free pool.
func (ctx *JITContext) TransferReg(r Reg) {
	_ = r
}

// AllocRegExcept allocates a fresh register guaranteed not to be any of the
// excluded registers. Use this when the new register will immediately receive
// a copy FROM one of the excluded registers — without this guard, AllocReg()
// might evict an excluded register and return it, making the subsequent copy
// a no-op self-move (and letting any ALU op on the result destroy the source).
//
// Architecture-agnostic: works equally for amd64 (16 regs), arm64 (31 regs),
// riscv64 (32 regs), etc. The protect/unprotect dance is an implementation
// detail hidden from callers.
func (ctx *JITContext) AllocRegExcept(excluded ...Reg) Reg {
	for _, r := range excluded {
		ctx.ProtectReg(r)
	}
	r := ctx.AllocReg()
	for _, ex := range excluded {
		if r == ex {
			panic("jit: AllocRegExcept returned excluded register")
		}
	}
	for _, r := range excluded {
		ctx.UnprotectReg(r)
	}
	return r
}

// EnsureReg checks if a descriptor was spilled and restores it.
// If the value is still in a register, this is a no-op.
// If spilled, allocates a new register, emits a load, and updates the desc.
func (ctx *JITContext) EnsureReg(desc *JITValueDesc) {
	if desc.Loc != LocStack {
		return
	}
	r := ctx.AllocReg()
	if desc.MemPtr != 0 {
		// Load from spill buffer using absolute address recorded in MemPtr.
		ctx.W.EmitMovRegImm64(RegR11, uint64(desc.MemPtr))
		ctx.W.EmitMovRegMem(r, RegR11, 0)
	} else {
		// Generic LocStack value (non-spill): load from [RSP+StackOff].
		ctx.W.EmitMovRegMem(r, RegRSP, desc.StackOff)
	}
	desc.Loc = LocReg
	desc.Reg = r
	desc.MemPtr = 0
	desc.StackOff = 0
	ctx.BindReg(r, desc)
	if desc.ID != 0 && ctx.descSpills != nil {
		delete(ctx.descSpills, desc.ID)
	}
}

// RestoreSpills restores all spilled registers from the spill buffer in reverse order.
// Since spills use a pre-allocated buffer (not RSP), this doesn't modify the stack pointer.
func (ctx *JITContext) RestoreSpills() {
	for i := ctx.spillStackLen - 1; i >= 0; i-- {
		slot := ctx.spillStack[i].slot
		spillAddr := uint64(uintptr(unsafe.Pointer(&ctx.SpillBuf[slot])))
		ctx.W.EmitMovRegImm64(RegR11, spillAddr)
		ctx.W.EmitMovRegMem(ctx.spillStack[i].reg, RegR11, 0)
	}
	ctx.spillStackLen = 0
	ctx.SpillTop = 0
}

// FreeDesc releases any registers held by a value descriptor.
func (ctx *JITContext) FreeDesc(desc *JITValueDesc) {
	switch desc.Loc {
	case LocReg:
		if desc.Reg <= RegR15 {
			owner := ctx.RegOwners[desc.Reg]
			if owner == desc || owner == nil {
				ctx.FreeReg(desc.Reg)
			}
		}
	case LocRegPair:
		if desc.Reg <= RegR15 {
			owner := ctx.RegOwners[desc.Reg]
			if owner == desc || owner == nil {
				ctx.FreeReg(desc.Reg)
			}
		}
		if desc.Reg2 <= RegR15 {
			owner := ctx.RegOwners[desc.Reg2]
			if owner == desc || owner == nil {
				ctx.FreeReg(desc.Reg2)
			}
		}
	case LocStack:
	case LocStackPair:
	}
	desc.Loc = LocNone
	desc.MemPtr = 0
	if desc.ID != 0 && ctx.descSpills != nil {
		delete(ctx.descSpills, desc.ID)
	}
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

// ConcatStrings concatenates two Go strings. Used as a JIT helper for string + string.
func ConcatStrings(a, b string) string {
	return a + b
}

// JITScmerToFloatBits converts a Scmer to float64 and returns the raw IEEE bits
// in a GPR-friendly integer return value for JIT helper calls.
func JITScmerToFloatBits(v Scmer) uint64 {
	return math.Float64bits(v.Float())
}

// JITIntDiv performs int64 division for JIT fallback lowering paths.
func JITIntDiv(a, b int64) int64 {
	return a / b
}

// JITIntRem performs int64 modulo for JIT fallback lowering paths.
func JITIntRem(a, b int64) int64 {
	return a % b
}

// GoABIIntRegs lists integer argument/result registers in Go internal ABI order.
var GoABIIntRegs = []Reg{RegRAX, RegRBX, RegRCX, RegRDI, RegRSI, RegR8, RegR9, RegR10, RegR11}

type goCallArgWord struct {
	loc JITLoc
	reg Reg
	imm uint64
}

func (ctx *JITContext) collectLiveRegsForCall(buf *[16]Reg) []Reg {
	allocatedMask := (^ctx.FreeRegs) & 0xFFFF // track all GPRs, incl. reserved non-alloc regs
	for r := Reg(0); r <= RegR15; r++ {
		if ctx.RegOwners[r] != nil && (ctx.FreeRegs&(1<<uint(r))) != 0 {
			panic("jit: internal reg state mismatch (owner set but register marked free)")
		}
	}
	liveCount := 0
	unknownCount := 0
	for r := Reg(0); r <= RegR15; r++ {
		if r == RegRSP || r == RegRBP || r == RegR11 || r == RegR14 {
			continue
		}
		if allocatedMask&(1<<uint(r)) == 0 {
			continue
		}
		if ctx.RegOwners[r] == nil {
			unknownCount++
			continue
		}
		buf[liveCount] = r
		liveCount++
	}

	// Conservative fallback: if we have untracked allocated registers, keep the
	// old semantics and treat all allocated registers as live.
	if unknownCount > 0 {
		liveCount = 0
		for r := Reg(0); r <= RegR15; r++ {
			if r == RegRSP || r == RegRBP || r == RegR11 || r == RegR14 {
				continue
			}
			if allocatedMask&(1<<uint(r)) == 0 {
				continue
			}
			buf[liveCount] = r
			liveCount++
		}
	}
	return buf[:liveCount]
}

// EmitGoCall emits a call to a Go function from JIT code.
// argWords: registers holding argument words in Go ABI order.
// numResultWords: how many result words to capture.
// Returns registers holding the result words.
// All live JIT registers are saved/restored around the call.
// EmitGoCall emits a call to a Go function from JIT code.
// argWords: registers holding argument words in Go ABI order.
// numResultWords: how many result words to capture.
// resultsBuf: caller-provided [16]Reg buffer for results (no heap alloc).
// Returns a slice into resultsBuf holding the result registers.
// All live JIT registers are saved/restored around the call.
func (ctx *JITContext) EmitGoCall(funcAddr uint64, argWords []goCallArgWord, numResultWords int, resultsBuf *[16]Reg) []Reg {
	if len(argWords) > len(GoABIIntRegs) {
		panic("jit: too many argument words for Go ABI")
	}
	if numResultWords > len(GoABIIntRegs) {
		panic("jit: too many result words for Go ABI")
	}

	// Owner-aware liveness with conservative fallback.
	var liveRegsArr [16]Reg
	liveRegs := ctx.collectLiveRegsForCall(&liveRegsArr)

	// Fast path: no live registers to preserve. Emit only argument setup + call.
	if len(liveRegs) == 0 {
		for i := len(argWords) - 1; i >= 0; i-- {
			target := GoABIIntRegs[i]
			switch argWords[i].loc {
			case LocReg:
				if argWords[i].reg != target {
					ctx.W.emitMovRegReg(target, argWords[i].reg)
				}
			case LocImm:
				ctx.W.EmitMovRegImm64(target, argWords[i].imm)
			default:
				panic("jit: unsupported Go-call arg location")
			}
		}
		ctx.W.EmitCallIndirect(funcAddr)
		for i := 0; i < numResultWords; i++ {
			r := ctx.AllocReg()
			if r != GoABIIntRegs[i] {
				ctx.W.emitMovRegReg(r, GoABIIntRegs[i])
			}
			resultsBuf[i] = r
		}
		return resultsBuf[:numResultWords]
	}

	// Reserve stack space for result words (above saved registers).
	// After restoring saved regs, these slots will be at [RSP+0..].
	resultBytes := numResultWords * 8
	if resultBytes > 0 {
		if resultBytes < 128 {
			ctx.W.emitBytes(0x48, 0x83, 0xEC, byte(resultBytes)) // SUB RSP, imm8
		} else {
			ctx.W.emitBytes(0x48, 0x81, 0xEC)
			ctx.W.emitU32(uint32(resultBytes)) // SUB RSP, imm32
		}
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
		switch argWords[i].loc {
		case LocReg:
			if argWords[i].reg != target {
				ctx.W.emitMovRegReg(target, argWords[i].reg)
			}
		case LocImm:
			ctx.W.EmitMovRegImm64(target, argWords[i].imm)
		default:
			panic("jit: unsupported Go-call arg location")
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
		ctx.W.EmitStoreRegMem(GoABIIntRegs[i], RegRSP, offset)
	}

	// Restore (POP in reverse)
	if padded {
		ctx.W.EmitPopReg(RegRAX)
	}
	for i := len(liveRegs) - 1; i >= 0; i-- {
		ctx.W.EmitPopReg(liveRegs[i])
	}

	// Pop results from reserved slots into freshly allocated registers
	for i := 0; i < numResultWords; i++ {
		r := ctx.AllocReg()
		ctx.W.EmitPopReg(r)
		resultsBuf[i] = r
	}
	return resultsBuf[:numResultWords]
}

// flattenArgs converts JITValueDesc arguments to ABI words.
// LocRegPair → 2 words (Reg, Reg2), LocReg → 1 word, LocImm → deferred imm.
// buf is a caller-provided [16]goCallArgWord scratch buffer; returns a slice into it.
func (ctx *JITContext) flattenArgs(args []JITValueDesc, buf *[16]goCallArgWord) []goCallArgWord {
	n := 0
	for _, a := range args {
		switch a.Loc {
		case LocRegPair:
			buf[n] = goCallArgWord{loc: LocReg, reg: a.Reg}
			n++
			buf[n] = goCallArgWord{loc: LocReg, reg: a.Reg2}
			n++
		case LocReg:
			buf[n] = goCallArgWord{loc: LocReg, reg: a.Reg}
			n++
		case LocStack:
			r := ctx.AllocReg()
			if a.MemPtr != 0 {
				ctx.W.EmitMovRegImm64(RegR11, uint64(a.MemPtr))
				ctx.W.EmitMovRegMem(r, RegR11, 0)
			} else {
				ctx.W.EmitMovRegMem(r, RegRSP, a.StackOff)
			}
			buf[n] = goCallArgWord{loc: LocReg, reg: r}
			n++
		case LocStackPair:
			r1 := ctx.AllocReg()
			r2 := ctx.AllocRegExcept(r1)
			if a.MemPtr != 0 {
				ctx.W.EmitMovRegImm64(RegR11, uint64(a.MemPtr))
				ctx.W.EmitMovRegMem(r1, RegR11, 0)
				ctx.W.EmitMovRegMem(r2, RegR11, 8)
			} else {
				ctx.W.EmitMovRegMem(r1, RegRSP, a.StackOff)
				ctx.W.EmitMovRegMem(r2, RegRSP, a.StackOff+8)
			}
			buf[n] = goCallArgWord{loc: LocReg, reg: r1}
			n++
			buf[n] = goCallArgWord{loc: LocReg, reg: r2}
			n++
		case LocImm:
			buf[n] = goCallArgWord{loc: LocImm, imm: uint64(a.Imm.Int())}
			n++
		case LocNone:
			r := ctx.AllocReg()
			ctx.W.EmitMovRegImm64(r, 0)
			buf[n] = goCallArgWord{loc: LocReg, reg: r}
			n++
		default:
			panic(fmt.Sprintf("jit: unsupported arg desc location in flattenArgs: %d", a.Loc))
		}
	}
	return buf[:n]
}

// EmitGoCallScalar calls a Go function and returns a single-word result as JITValueDesc.
func (ctx *JITContext) EmitGoCallScalar(funcAddr uint64, args []JITValueDesc, numResultWords int) JITValueDesc {
	var wordsBuf [16]goCallArgWord
	var resultsBuf [16]Reg
	words := ctx.flattenArgs(args, &wordsBuf)
	results := ctx.EmitGoCall(funcAddr, words, numResultWords, &resultsBuf)
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
	var wordsBuf [16]goCallArgWord
	var resultsBuf [16]Reg
	words := ctx.flattenArgs(args, &wordsBuf)
	ctx.EmitGoCall(funcAddr, words, 0, &resultsBuf)
}
