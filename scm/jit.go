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
	"syscall"
	"unsafe"
)

/*
JIT Emitter Contract
====================

Each Declaration may provide a JITEmit callback:

	func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc

This callback emits machine code for the operation. The contract between
caller and emitter is as follows.

Emiter rules
-------------
 - The emiter must be a recursive 1-pass compiler that continuously writes into the JIT buffer
 - each emitter takes input args ([]JITValueDesc), result JITValueDesc with placement info for the result (e.g. store to stack, store into rax or "any" if we don't care) and returns a JITValueDesc with the actual result placement - some emitters, especially basic emitters can also deviate from this signature but the idea must stay the same
 - There following types of emitters exist:
  * basic emitters (defined in scm/jit_[ARCH].go) that produce actual machine code like arithmetic, move or jump instructions
  * hardcoded emiters for inlining Go-functions like IsString, String, NewString for extra speed and full semantics control
  * generated emiters produced by tools/jitgen for inlining scm functions. Jitgen takes in a Go function via Go compiler, analyzes the SSA and produce go code that is patched into go files like scm/alu.go
  * Scm JIT compiler which mimics the structure scm.Eval() function and produces a function call frame in order to turn a scm.Proc into func(...Scmer)Scmer
 - Emiters use the JITContext to request free registers or write bytes into the JIT buffer
 - All registers acquired by an emiter must be freed after leaving the emiter function
 - Complex emiters can have BBs (basic blocks -> jump-free blocks with a [conditional] jump at the end)
 - Only reachable BBs must be rendered -> if an "if" instruction has a constant condition, only render one additional BB
 - Emiters are chainable (inline function calls): A complex emiter calls another emiter.
 - BBs are not allowed to print return (0xC3), only a jump to the last BB so emiters stay chainable
 - Each BB is declared as a BBDescriptor on the stack of the emiter function
 - the BB chain is started by firstbb.render(). Each bb render function can tail-call other bb render functions in order to "enqueue" them -> jumps tail-call ONE sucessor BB, conditional jumps tail-call up to TWO successor BBs, so we have a DFS traversal of all reachable BBs
 - a BB can either be rendered as the general block (phi inputs are on stack) or a specialized block (phi inputs can be either on stack or overwritten with other JITValueDesc like immediate-values or type-annotated)
 - general BBs must be rendered at-most once, if the BB already exists, jump to it instead
 - specialized (non-gerneral) BBs can be rendered more than once but must be limited (e.g. 2 instances at most). If the limit is exceeded, the general block must be used instead.
 - specialized BBs can be used if some of the phi inputs are known-typed (tag != unknown) or even constant (locImm) to enable loop unrolling with specialized values (e.g. index 0 -> reads from args[0] -> args[0] is constant)
 - a BB render function calls the arch-specific instruction emitters that write into the JIT buffer aswell as other emitters to inline the functions
 - each

Input arguments (args):

  Each args[i] describes where the i-th operand lives at the point of the
  call. The emitter must handle all location modes:

  - LocImm:     compile-time constant. args[i].Imm holds a Scmer value;
                Imm.GetTag() carries the type. No register is allocated.
                The emitter SHOULD constant-fold when all inputs are LocImm.
  - LocReg:     unboxed primitive in args[i].Reg.
  - LocRegPair: boxed Scmer in args[i].Reg (ptr) + args[i].Reg2 (aux).
  - LocStack:   value on the stack at args[i].StackOff.
  - LocStackPair:
                two-word value at args[i].StackOff / args[i].StackOff+8
                (or spill-backed via MemPtr when applicable).
  - LocMem:     value at fixed memory address args[i].MemPtr.

  The emitter takes ownership of input registers: it MUST call
  ctx.FreeDesc(&args[i]) for every register-located input it consumes.
  Inputs in LocImm/LocStack/LocStackPair/LocMem need no freeing.

Result placement (result):

  The result parameter tells the emitter WHERE to put its output.

  - LocAny:     emitter chooses freely. May return LocImm (best: zero code
                emitted), LocReg, or anything else. Use this when the caller
                will immediately pass the result into another emitter.
  - LocReg:     result MUST be placed into result.Reg.
  - LocRegPair: result MUST be placed into result.Reg + result.Reg2.
  - LocStack:   result MUST be written to result.StackOff.
  - LocStackPair:
                result MUST be written as two words starting at
                result.StackOff (or spill-backed MemPtr location).
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

// JITEntryPoint holds a JIT-compiled function alongside its original
// Scheme representation for serialization and fallback.
type JITEntryPoint struct {
	Native     func(...Scmer) Scmer // compiled native function pointer
	Pages      []*JITPage           // mmap'd pages holding machine code
	Pool       *ShardJITPool        // pool for returning pages
	ConstRoots []unsafe.Pointer     // GC roots for constants embedded into machine code
	Proc       Proc                 // original Proc for serialization
}

// JITValueDesc describes a value during JIT compilation: its type and
// storage location. Flows through expression compilation for type
// propagation — analogous to optimizerMetainfo in the optimizer.
//
// Type uses the tag constants (tagInt, tagFloat, tagBool, ...) directly,
// or JITTypeUnknown (0xFF) when the type is not known at compile time.
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
	Type     uint8 // tag constant (tagInt, tagFloat, ...) or JITTypeUnknown
	Loc      JITLoc
	Reg      Reg
	Reg2     Reg     // second register (for Scmer: ptr+aux)
	StackOff int32   // stack offset (if Loc == LocStack)
	MemPtr   uintptr // memory address (if Loc == LocMem)
	Imm      Scmer   // compile-time constant (if Loc == LocImm); Imm.GetTag() carries type info
}

// ---- merged from scm/jit_types.go ----

// Reg represents a hardware register index. The actual register constants
// (RAX, R8, X0, etc.) are defined in architecture-specific files.
type Reg uint8

// JITTypeUnknown means the Scmer type is not known at compile time.
// All other type values are tag constants (tagInt, tagFloat, tagBool, etc.)
// so GetTag can be constant-folded when Type != JITTypeUnknown.
const JITTypeUnknown uint8 = 0xFF

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

// JITFixup records a forward reference that must be patched after all
// labels are placed.
type JITFixup struct {
	CodePos  int32 // position in code
	LabelID  uint8 // target label
	Size     uint8 // 1=rel8, 4=rel32
	Relative bool  // true for PC-relative jumps
}

// PhiState carries incoming phi overlays for recursive BB renderers.
// General=true means canonical BB emission mode (stack-backed phis / relocatable label target).
// General=false allows specialized overlays for bounded unrolling.
type PhiState struct {
	General       bool
	OverlayValues []JITValueDesc
	PhiValues     []JITValueDesc
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

type descSpillMeta struct {
	loc      JITLoc
	memPtr   uintptr
	stackOff int32
}

// JITContext is the central structure for descriptor-based JIT compilation.
type JITContext struct {
	Ptr     unsafe.Pointer // current write pointer (into mmap memory)
	End     unsafe.Pointer // page end minus reserve
	Start   unsafe.Pointer // page start for position calculation
	Pages   []*JITPage
	Current *JITPage

	Labels    [256]int32
	LabelNext uint8

	Fixups    [512]JITFixup
	FixupNext uint8

	Env       *JITEnv
	FreeRegs  uint64
	AllRegs   uint64 // original set of all allocatable registers (for spilling)
	SliceBase Reg    // register holding the args slice pointer (for variable-index access)
	// SliceBaseTracksRSP indicates that SliceBase is a mirror of RSP and must be
	// refreshed after helper calls (Go may grow/move the goroutine stack).
	SliceBaseTracksRSP bool
	RegOwners          [16]*JITValueDesc // register → owner descriptor (nil = untracked)
	SpillBuf           [4096]int64       // pre-allocated spill buffer (heap-stable, not on stack)
	SpillTop           int               // high-water mark in SpillBuf (next fresh slot)
	ProtectedRegs      uint64            // bitmask of registers that must not be spilled
	ProtectedRegCounts [16]int           // per-register protection refcount (supports nested protection)
	nextDescID         uint32
	descOwners         map[uint32]*JITValueDesc
	descSpills         map[uint32]descSpillMeta
	// ConstRoots holds pointer payloads from LocImm Scmer values that were
	// materialized into machine code immediates. Keeping these pointers in a
	// Go heap object reachable from JITEntryPoint prevents GC from reclaiming
	// referenced heap data while JIT code may still dereference it.
	ConstRoots []unsafe.Pointer
	rootSet    map[unsafe.Pointer]struct{}
}

// jitAllocStateSnapshot captures allocator/spill bookkeeping so emitter
// generation can render sibling BBs from identical allocator state.
type jitAllocStateSnapshot struct {
	freeRegs           uint64
	protectedRegs      uint64
	protectedRegCounts [16]int
	regOwners          [16]*JITValueDesc
	spillTop           int
	descSpills         map[uint32]descSpillMeta
}

func (ctx *JITContext) SnapshotAllocState() jitAllocStateSnapshot {
	s := jitAllocStateSnapshot{
		freeRegs:           ctx.FreeRegs,
		protectedRegs:      ctx.ProtectedRegs,
		protectedRegCounts: ctx.ProtectedRegCounts,
		regOwners:          ctx.RegOwners,
		spillTop:           ctx.SpillTop,
	}
	if len(ctx.descSpills) != 0 {
		s.descSpills = make(map[uint32]descSpillMeta, len(ctx.descSpills))
		for k, v := range ctx.descSpills {
			s.descSpills[k] = v
		}
	}
	return s
}

func (ctx *JITContext) RestoreAllocState(s jitAllocStateSnapshot) {
	ctx.FreeRegs = s.freeRegs
	ctx.ProtectedRegs = s.protectedRegs
	ctx.ProtectedRegCounts = s.protectedRegCounts
	ctx.RegOwners = s.regOwners
	ctx.SpillTop = s.spillTop
	if s.descSpills == nil {
		ctx.descSpills = nil
	} else {
		ctx.descSpills = make(map[uint32]descSpillMeta, len(s.descSpills))
		for k, v := range s.descSpills {
			ctx.descSpills[k] = v
		}
	}
}

// BBDescriptor stores per-basic-block emitter state.
// Phase 1 starts with single-block closure usage; relocation fields are
// prepared for follow-up BB-descriptor-based control-flow lowering.
type BBDescriptor struct {
	// Render is kept for compatibility with older generated emitters.
	Render func() JITValueDesc
	// RenderPS is the PhiState-aware recursive BB renderer.
	RenderPS func(ps PhiState) JITValueDesc
	Rendered bool
	Address  int32
	Pending  []JITFixup
	// VisitCount tracks how often this BB descriptor has been entered.
	// Unroll/specialization limits are derived from this per-BB state.
	VisitCount uint16
	// RenderCount is kept for compatibility with older generated emitters.
	RenderCount uint16
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

// ReclaimUntrackedRegs marks allocatable registers as free when they have no
// tracked owner descriptor. This is used at BB boundaries in closure emitters
// to prevent stale temporary allocations from exhausting the allocator.
func (ctx *JITContext) ReclaimUntrackedRegs() {
	for rr := Reg(0); rr <= RegR15; rr++ {
		bit := uint64(1 << uint(rr))
		if (ctx.AllRegs & bit) == 0 {
			continue
		}
		if (ctx.ProtectedRegs & bit) != 0 {
			continue
		}
		owner := ctx.RegOwners[rr]
		if owner == nil {
			ctx.FreeRegs |= bit
			continue
		}
		valid := false
		switch owner.Loc {
		case LocReg:
			valid = owner.Reg == rr
		case LocRegPair:
			valid = owner.Reg == rr || owner.Reg2 == rr
		}
		if !valid {
			ctx.RegOwners[rr] = nil
			ctx.FreeRegs |= bit
		}
	}
}

// AllocReg picks a free register from the bitmap and marks it used.
// If no registers are free, spills the highest-numbered in-use register
// to a pre-allocated buffer and returns it.
func (ctx *JITContext) AllocReg() Reg {
	// Sanitize stale owner links: if an owner descriptor no longer claims this
	// hardware register, drop the stale owner edge and mark the register free.
	for rr := Reg(0); rr <= RegR15; rr++ {
		if (ctx.AllRegs & (1 << uint(rr))) == 0 {
			continue
		}
		owner := ctx.RegOwners[rr]
		if owner == nil {
			continue
		}
		valid := false
		switch owner.Loc {
		case LocReg:
			valid = owner.Reg == rr
		case LocRegPair:
			valid = owner.Reg == rr || owner.Reg2 == rr
		}
		if !valid {
			ctx.RegOwners[rr] = nil
			ctx.FreeRegs |= 1 << uint(rr)
		}
	}

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
		ownerMask := uint64(0)
		ownerDump := ""
		for rr := Reg(0); rr <= RegR15; rr++ {
			if ctx.RegOwners[rr] != nil {
				ownerMask |= 1 << uint(rr)
				o := ctx.RegOwners[rr]
				ownerDump += fmt.Sprintf(" r%d(loc=%d reg=%d reg2=%d)", rr, o.Loc, o.Reg, o.Reg2)
			}
		}
		panic(fmt.Sprintf("jit: register spill required (fallback) free=%#x all=%#x prot=%#x owners=%#x%s", ctx.FreeRegs, ctx.AllRegs, ctx.ProtectedRegs, ownerMask, ownerDump))
	}

	owner := ctx.RegOwners[r]
	if pairSpill {
		slot := ctx.SpillTop
		if slot+1 >= len(ctx.SpillBuf) {
			panic("jit: spill buffer overflow")
		}
		ctx.SpillTop += 2
		spillAddrPtr := uintptr(unsafe.Pointer(&ctx.SpillBuf[slot]))
		ctx.EmitMovRegImm64(RegR11, uint64(spillAddrPtr))
		ctx.EmitStoreRegMem(pairR1, RegR11, 0)
		ctx.EmitStoreRegMem(pairR2, RegR11, 8)

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
	ctx.EmitMovRegImm64(RegR11, uint64(spillAddrPtr))
	ctx.EmitStoreRegMem(r, RegR11, 0) // MOV [R11], reg

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
		if desc.MemPtr != 0 {
			// Spill-buffer backed pair.
			ctx.EmitMovRegImm64(RegR11, uint64(desc.MemPtr))
			ctx.EmitMovRegMem(r1, RegR11, 0)
			ctx.EmitMovRegMem(r2, RegR11, 8)
		} else {
			// Regular frame-backed pair.
			ctx.EmitMovRegMem(r1, RegRSP, desc.StackOff)
			ctx.EmitMovRegMem(r2, RegRSP, desc.StackOff+8)
		}
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
	if ctx.descOwners == nil {
		ctx.descOwners = make(map[uint32]*JITValueDesc)
	}
	owner := ctx.descOwners[desc.ID]
	if owner == nil {
		owner = &JITValueDesc{}
		ctx.descOwners[desc.ID] = owner
	}
	*owner = *desc
	// A bound register is live and must not be treated as free.
	ctx.FreeRegs &^= 1 << uint(r)
	ctx.RegOwners[r] = owner
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
		ctx.EmitMovRegImm64(RegR11, uint64(desc.MemPtr))
		ctx.EmitMovRegMem(r, RegR11, 0)
	} else {
		// Generic LocStack value (non-spill): load from [RSP+StackOff].
		ctx.EmitMovRegMem(r, RegRSP, desc.StackOff)
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

// FreeDesc releases any registers held by a value descriptor.
func (ctx *JITContext) FreeDesc(desc *JITValueDesc) {
	// Non-owning descriptors (ID==0), e.g. copied call arguments, must not
	// mutate placement/free registers from the original source descriptor.
	if desc.ID == 0 {
		return
	}
	switch desc.Loc {
	case LocReg:
		if desc.Reg <= RegR15 {
			owner := ctx.RegOwners[desc.Reg]
			if owner == nil || owner == desc || (desc.ID != 0 && owner.ID == desc.ID) {
				ctx.FreeReg(desc.Reg)
			}
		}
	case LocRegPair:
		if desc.Reg <= RegR15 {
			owner := ctx.RegOwners[desc.Reg]
			if owner == nil || owner == desc || (desc.ID != 0 && owner.ID == desc.ID) {
				ctx.FreeReg(desc.Reg)
			}
		}
		if desc.Reg2 <= RegR15 {
			owner := ctx.RegOwners[desc.Reg2]
			if owner == nil || owner == desc || (desc.ID != 0 && owner.ID == desc.ID) {
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

func JITFloorBits(v uint64) uint64 {
	return math.Float64bits(math.Floor(math.Float64frombits(v)))
}

func JITCeilBits(v uint64) uint64 {
	return math.Float64bits(math.Ceil(math.Float64frombits(v)))
}

func JITSqrtBits(v uint64) uint64 {
	return math.Float64bits(math.Sqrt(math.Float64frombits(v)))
}

func JITAbsBits(v uint64) uint64 {
	return math.Float64bits(math.Abs(math.Float64frombits(v)))
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
	loc      JITLoc
	reg      Reg
	imm      uint64
	memPtr   uintptr
	stackOff int32
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
	// Preserve the argument slice base register across helper calls as well.
	// It is not part of the allocator pool but can still be needed by
	// subsequent argument loads in the same emitted function.
	switch ctx.SliceBase {
	case RegRSP, RegRBP, RegR11, RegR14:
		// never preserved here
	default:
		found := false
		for _, r := range liveRegs {
			if r == ctx.SliceBase {
				found = true
				break
			}
		}
		if !found {
			liveRegs = append(liveRegs, ctx.SliceBase)
		}
	}
	emitArgSetup := func(stackArgBaseDisp int32) {
		type regMove struct {
			dst Reg
			src Reg
		}
		moves := make([]regMove, 0, len(argWords))
		for i := range argWords {
			target := GoABIIntRegs[i]
			if argWords[i].loc == LocReg && argWords[i].reg != target {
				moves = append(moves, regMove{dst: target, src: argWords[i].reg})
			}
		}
		for len(moves) > 0 {
			emitIdx := -1
			for i := range moves {
				dstIsPendingSrc := false
				for j := range moves {
					if i == j {
						continue
					}
					if moves[j].src == moves[i].dst {
						dstIsPendingSrc = true
						break
					}
				}
				if !dstIsPendingSrc {
					emitIdx = i
					break
				}
			}
			if emitIdx == -1 {
				// Break a cycle via reserved scratch register R14.
				cycleDst := moves[0].dst
				if cycleDst != RegR14 {
					ctx.emitMovRegReg(RegR14, cycleDst)
				}
				for i := range moves {
					if moves[i].src == cycleDst {
						moves[i].src = RegR14
					}
				}
				continue
			}
			mv := moves[emitIdx]
			if mv.dst != mv.src {
				ctx.emitMovRegReg(mv.dst, mv.src)
			}
			moves = append(moves[:emitIdx], moves[emitIdx+1:]...)
		}

		for i := range argWords {
			target := GoABIIntRegs[i]
			switch argWords[i].loc {
			case LocReg:
				// Already handled by move planner (including no-op src==target).
			case LocImm:
				ctx.EmitMovRegImm64(target, argWords[i].imm)
			case LocStack:
				if argWords[i].memPtr != 0 {
					ctx.EmitMovRegImm64(RegR11, uint64(argWords[i].memPtr))
					ctx.EmitMovRegMem(target, RegR11, argWords[i].stackOff)
				} else {
					ctx.EmitMovRegMem(target, RegRSP, stackArgBaseDisp+argWords[i].stackOff)
				}
			default:
				panic("jit: unsupported Go-call arg location")
			}
		}
	}

	// Fast path: no live registers to preserve. Emit only argument setup + call.
	if len(liveRegs) == 0 {
		emitArgSetup(0)
		ctx.EmitCallIndirect(funcAddr)
		if ctx.SliceBaseTracksRSP && ctx.SliceBase != RegRSP {
			ctx.emitMovRegReg(ctx.SliceBase, RegRSP)
		}
		for i := 0; i < numResultWords; i++ {
			r := ctx.AllocReg()
			if r != GoABIIntRegs[i] {
				ctx.emitMovRegReg(r, GoABIIntRegs[i])
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
			ctx.emitBytes(0x48, 0x83, 0xEC, byte(resultBytes)) // SUB RSP, imm8
		} else {
			ctx.emitBytes(0x48, 0x81, 0xEC)
			ctx.emitU32(uint32(resultBytes)) // SUB RSP, imm32
		}
	}

	// Save live registers (PUSH)
	for _, r := range liveRegs {
		ctx.EmitPushReg(r)
	}
	// Align stack to 16 bytes if needed (odd total items)
	totalItems := numResultWords + len(liveRegs)
	padded := totalItems%2 == 1
	if padded {
		ctx.EmitPushReg(RegRAX) // dummy padding
	}

	// Move argument words into Go ABI registers (clobber-safe planner).
	stackArgBaseDisp := int32(resultBytes + len(liveRegs)*8)
	if padded {
		stackArgBaseDisp += 8
	}
	emitArgSetup(stackArgBaseDisp)

	// CALL
	ctx.EmitCallIndirect(funcAddr)

	// Store results to reserved stack slots (above saved regs + padding)
	paddingSize := 0
	if padded {
		paddingSize = 8
	}
	for i := 0; i < numResultWords; i++ {
		offset := int32(paddingSize + len(liveRegs)*8 + i*8)
		ctx.EmitStoreRegMem(GoABIIntRegs[i], RegRSP, offset)
	}

	// Restore (POP in reverse)
	if padded {
		ctx.EmitPopReg(RegRAX)
	}
	for i := len(liveRegs) - 1; i >= 0; i-- {
		ctx.EmitPopReg(liveRegs[i])
	}

	// Pop results from reserved slots into freshly allocated registers
	for i := 0; i < numResultWords; i++ {
		r := ctx.AllocReg()
		ctx.EmitPopReg(r)
		resultsBuf[i] = r
	}
	if ctx.SliceBaseTracksRSP && ctx.SliceBase != RegRSP {
		ctx.emitMovRegReg(ctx.SliceBase, RegRSP)
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
			off := a.StackOff
			if a.MemPtr != 0 {
				// MemPtr-backed values already point at the concrete spill slot.
				// StackOff is a logical slot id there, not an additional byte offset.
				off = 0
			}
			buf[n] = goCallArgWord{loc: LocStack, memPtr: a.MemPtr, stackOff: off}
			n++
		case LocStackPair:
			off0 := a.StackOff
			off1 := a.StackOff + 8
			if a.MemPtr != 0 {
				// MemPtr-backed pair points to element 0 directly.
				off0 = 0
				off1 = 8
			}
			buf[n] = goCallArgWord{loc: LocStack, memPtr: a.MemPtr, stackOff: off0}
			n++
			buf[n] = goCallArgWord{loc: LocStack, memPtr: a.MemPtr, stackOff: off1}
			n++
		case LocImm:
			var immWord uint64
			switch a.Type {
			case tagInt:
				immWord = uint64(a.Imm.Int())
			case tagBool:
				if a.Imm.Bool() {
					immWord = 1
				} else {
					immWord = 0
				}
			case tagFloat:
				immWord = math.Float64bits(a.Imm.Float())
			case tagNil:
				immWord = 0
			default:
				panic(fmt.Sprintf("jit: LocImm scalar Go-call arg requires explicit materialization (type=%d, tag=%d)", a.Type, a.Imm.GetTag()))
			}
			buf[n] = goCallArgWord{loc: LocImm, imm: immWord}
			n++
		case LocNone:
			buf[n] = goCallArgWord{loc: LocImm, imm: 0}
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
		d := JITValueDesc{Loc: LocReg, Reg: results[0]}
		ctx.BindReg(results[0], &d)
		return d
	}
	d := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: results[0], Reg2: results[1]}
	ctx.BindReg(results[0], &d)
	ctx.BindReg(results[1], &d)
	return d
}

// EmitMovPairToResult moves a LocRegPair value into the result descriptor registers.
func (ctx *JITContext) EmitMovPairToResult(src *JITValueDesc, dst *JITValueDesc) {
	if src.Reg != dst.Reg {
		ctx.emitMovRegReg(dst.Reg, src.Reg)
	}
	if src.Reg2 != dst.Reg2 {
		ctx.emitMovRegReg(dst.Reg2, src.Reg2)
	}
}

// EmitGoCallVoid calls a Go function with no return value.
func (ctx *JITContext) EmitGoCallVoid(funcAddr uint64, args []JITValueDesc) {
	var wordsBuf [16]goCallArgWord
	var resultsBuf [16]Reg
	words := ctx.flattenArgs(args, &wordsBuf)
	ctx.EmitGoCall(funcAddr, words, 0, &resultsBuf)
}

// ---- merged from scm/jit_writer.go ----

// JITPage represents one page of mmap'd executable memory.
type JITPage struct {
	RwBase unsafe.Pointer // writable mapping
	RxBase unsafe.Pointer // executable mapping
	Next   *JITPage
}

// ReserveLabel allocates a label ID for later placement via MarkLabel.
func (ctx *JITContext) ReserveLabel() uint8 {
	id := ctx.LabelNext
	ctx.LabelNext++
	ctx.Labels[id] = -1 // undefined until MarkLabel
	return id
}

// MarkLabel sets the position of a previously reserved label.
func (ctx *JITContext) MarkLabel(id uint8) {
	ctx.Labels[id] = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
}

// AddFixup records a forward reference to be patched by ResolveFixups.
func (ctx *JITContext) AddFixup(labelID uint8, size uint8, relative bool) {
	ctx.Fixups[ctx.FixupNext] = JITFixup{
		CodePos:  int32(uintptr(ctx.Ptr) - uintptr(ctx.Start)),
		LabelID:  labelID,
		Size:     size,
		Relative: relative,
	}
	ctx.FixupNext++
}

// ResolveFixups patches recorded forward references whose labels are defined.
// Fixups referencing still-undefined labels are kept for a later call.
func (ctx *JITContext) ResolveFixups() {
	j := uint8(0)
	for i := uint8(0); i < ctx.FixupNext; i++ {
		f := &ctx.Fixups[i]
		targetPos := ctx.Labels[f.LabelID]
		if targetPos < 0 {
			// label not yet defined — keep for later
			ctx.Fixups[j] = ctx.Fixups[i]
			j++
			continue
		}
		patchAddr := unsafe.Add(ctx.Start, int(f.CodePos))
		if f.Relative {
			offset := targetPos - (f.CodePos + int32(f.Size))
			*(*int32)(patchAddr) = offset
			ctx.tryRewriteTrailingJmpToNop(f, offset)
		} else {
			*(*int32)(patchAddr) = targetPos
		}
	}
	ctx.FixupNext = j
}

// ResolveFixupsFinal patches all remaining fixups, panicking on undefined labels.
func (ctx *JITContext) ResolveFixupsFinal() {
	for i := uint8(0); i < ctx.FixupNext; i++ {
		f := &ctx.Fixups[i]
		targetPos := ctx.Labels[f.LabelID]
		if targetPos < 0 {
			panic("jit: undefined label")
		}
		patchAddr := unsafe.Add(ctx.Start, int(f.CodePos))
		if f.Relative {
			offset := targetPos - (f.CodePos + int32(f.Size))
			*(*int32)(patchAddr) = offset
			ctx.tryRewriteTrailingJmpToNop(f, offset)
		} else {
			*(*int32)(patchAddr) = targetPos
		}
	}
	ctx.FixupNext = 0
}

// tryRewriteTrailingJmpToNop turns a resolved "jmp +0" (jump-to-next-ip) into
// five NOP bytes. This keeps one-pass forward emission simple while removing
// redundant trailing jumps after relocation.
func (ctx *JITContext) tryRewriteTrailingJmpToNop(f *JITFixup, offset int32) {
	if offset != 0 || f.Size != 4 || f.CodePos <= 0 {
		return
	}
	opAddr := unsafe.Add(ctx.Start, int(f.CodePos)-1)
	if *(*byte)(opAddr) != 0xE9 { // JMP rel32 opcode
		return
	}
	for i := 0; i < 5; i++ {
		*(*byte)(unsafe.Add(opAddr, i)) = 0x90 // NOP
	}
}

// ---- merged from scm/jit_entry.go ----

// ShardJITPool manages mmap'd page allocation per shard. Defined here as
// a placeholder; the full implementation will be added when the page
// allocator is built.
type ShardJITPool struct {
}

// ---- merged from scm/jit.go ----

var JITLog bool

// execBuf is a small wrapper for mmap'd memory
type execBuf struct {
	ptr unsafe.Pointer
	n   int // size
}

func allocExec(size int) (*execBuf, error) {
	page := syscall.Getpagesize()
	n := (size + page - 1) & ^(page - 1)
	b, err := syscall.Mmap(-1, 0, n, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		return nil, err
	}
	return &execBuf{ptr: unsafe.Pointer(&b[0]), n: n}, nil
}

func (e *execBuf) makeRX() error {
	// change to PROT_READ|PROT_EXEC
	data := (*[1 << 30]byte)(e.ptr)[:e.n:e.n]
	return syscall.Mprotect(data, syscall.PROT_READ|syscall.PROT_EXEC)
}

func init_jit() {
	DeclareTitle("JIT Compilation")

	Declare(&Globalenv, &Declaration{
		"jit", "compiles a lambda to optimized native code; passes through already compiled functions",
		1, 1,
		[]DeclarationParameter{
			{"fn", "any", "the function to compile", nil},
		}, "any",
		jitCompile,
		false, false, nil, nil, // not pure because it allocates executable memory
	})
	Declare(&Globalenv, &Declaration{
		"jit?", "tells whether a value is a JIT-compiled function descriptor",
		1, 1,
		[]DeclarationParameter{
			{"value", "any", "value to inspect", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(a[0].GetTag() == tagJIT)
		},
		true, false, nil, func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d0 JITValueDesc
			_ = d0
			var d1 JITValueDesc
			_ = d1
			var d2 JITValueDesc
			_ = d2
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
					ps.General = true
					return bbs[0].RenderPS(ps)
				}
			}
			bbs[0].VisitCount++
			if ps.General {
				if bbs[0].Rendered {
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d1 = ctx.EmitGetTagDesc(&d0, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(11))}
			} else {
				r0 := ctx.AllocReg()
				ctx.EmitCmpRegImm32(d1.Reg, 11)
				ctx.EmitSetcc(r0, CcE)
				d2 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
				ctx.BindReg(r0, &d2)
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d2.Loc == LocImm {
				ctx.EmitMakeBool(result, d2)
			} else {
				ctx.EmitMakeBool(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			result.Type = tagBool
			return result
			return result
			}
			ps3 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps3)
			return result
		},
	})
}

// jitCompile compiles a Proc to a native function (tagFunc)
// Already compiled functions (tagFunc, tagFuncEnv) are passed through unchanged
func jitCompile(a ...Scmer) Scmer {
	if len(a) != 1 {
		panic("jit: expects exactly 1 argument")
	}

	v := a[0]
	tag := v.GetTag()
	if JITLog {
		fmt.Printf("JIT: compile %s\n", SerializeToString(v, &Globalenv))
	}

	switch tag {
	case tagJIT:
		// Already compiled
		return v
	case tagFunc:
		// Already a native function - pass through
		return v

	case tagFuncEnv:
		// Already a native function with environment - pass through
		return v

	case tagProc:
		// Lambda/procedure - attempt native compilation first
		proc := v.Proc()
		for _, codeCap := range [...]int{16 * 1024, 64 * 1024, 256 * 1024, 1024 * 1024} {
			buf, err := allocExec(codeCap)
			if err != nil {
				break
			}
			codeLen, roots, overflow := jitCompileProcToExec(proc, buf)
			if codeLen > 0 {
				code := (*[1 << 30]byte)(buf.ptr)[:codeLen:codeLen]
				if JITLog {
					fmt.Printf("%X\n", code)
				}
				if err2 := buf.makeRX(); err2 == nil {
					fn2 := unsafe.Pointer(&struct{ *byte }{&code[0]})
					nativeFn := *(*func(...Scmer) Scmer)(unsafe.Pointer(&fn2))
					return NewJIT(&JITEntryPoint{
						Native:     nativeFn,
						ConstRoots: roots,
						Proc:       *proc,
					})
				}
			}
			syscall.Munmap((*[1 << 30]byte)(buf.ptr)[:buf.n:buf.n])
			if !overflow {
				break
			}
		}
		if JITLog {
			fmt.Println("<fallback>")
		}
		// Fallback returns the original lambda/procedure unchanged.
		return v

	default:
		panic(fmt.Sprintf("jit: cannot compile %v (tag %d)", v, tag))
	}
}
