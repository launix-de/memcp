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
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/jtolds/gls"
)

/*
JIT Emitter Contract
====================

Each Declaration may provide a JITEmit callback:

	func(ctx *JITContext, args []Scmer, descs []JITValueDesc, result JITValueDesc) JITValueDesc

This callback emits machine code for the operation. The contract between
caller and emitter is as follows.

Emitter rules
-------------
 - The emitter must be a recursive 1-pass compiler that continuously writes into the JIT buffer
 - The emitter is intentionally not followed by a JIT optimizer or peephole pass. It must consume constants, known types, control-flow reachability, and requested result placement while walking the expression from front to back, and emit only instructions that remain necessary.
 - A statically known type test emits its result directly (often no instruction at all through LocImm). A statically impossible operation aborts the current compilation immediately so the caller can fall back safely. Only genuinely dynamic types receive a runtime tag check and branch.
 - Instruction selection is final at emission time: choose the shortest valid immediate/move form, fold constants, inline eligible operations, and omit unreachable branches instead of emitting general code for a later cleanup pass.
 - each emitter takes input args ([]JITValueDesc), result JITValueDesc with placement info for the result (e.g. store to stack, store into rax or "any" if we don't care) and returns a JITValueDesc with the actual result placement - some emitters, especially basic emitters can also deviate from this signature but the idea must stay the same
 - The following types of emitters exist:
  * basic emitters (defined in scm/jit_[ARCH].go) that produce actual machine code like arithmetic, move or jump instructions
  * hardcoded emiters for inlining Go-functions like IsString, String, NewString for extra speed and full semantics control
  * generated emitters produced by tools/jitgen for inlining scm functions. Jitgen takes in a Go function via Go compiler, analyzes the SSA and produces Go code that is patched into files like scm/alu.go
  * Scm JIT compiler which mimics the structure scm.Eval() function and produces a function call frame in order to turn a scm.Proc into func(...Scmer)Scmer
 - Emitters use the JITContext to request free registers or write bytes into the JIT buffer
 - All registers acquired by an emitter must be freed after leaving the emitter function
 - Complex emitters can have BBs (basic blocks -> jump-free blocks with a [conditional] jump at the end)
 - Only reachable BBs must be rendered -> if an "if" instruction has a constant condition, only render one additional BB
 - Emitters are chainable (inline function calls): A complex emitter calls another emitter.
 - BBs are not allowed to print return (0xC3), only a jump to the last BB so emitters stay chainable
 - Each BB is declared as a BBDescriptor on the stack of the emitter function
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
				result.StackOff.
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
	Native    func(...Scmer) Scmer // compiled native function pointer
	DebugName string
	// TransferInputArgs means the native body returns its complete variadic
	// argument array as an owned list. Call must make that array fresh because
	// apply may otherwise pass caller-owned list backing directly.
	TransferInputArgs bool
	HiddenArgs        []JITHiddenArg
	CodePtr           unsafe.Pointer   // start of code in arena
	CodeLen           int              // bytes used
	Arena             *jitArena        // owning arena (for free on GC)
	ConstRoots        []unsafe.Pointer // GC roots for constants embedded into machine code
	Proc              Proc             // original Proc for serialization
	// NeedsStableArgs records that emitted code crosses into Go. Precise JIT
	// stack maps now relocate the saved variadic data pointer during stack growth;
	// the flag remains diagnostic metadata for compiled entry points.
	NeedsStableArgs bool
	// AutoImportSafe is false for native bodies that still rely on an
	// experimental pointer-producing path. Explicit (jit ...) may exercise such
	// paths, but module import keeps the interpreter until they are proven safe.
	AutoImportSafe bool
	// RecursiveLambdas makes lambda values constructed by this native body
	// compile their own body before they are returned or passed onward.
	RecursiveLambdas bool
	// Coverage counts lowered Scheme expression nodes and calls which still
	// cross the generic Eval/Apply bridge. It is diagnostic metadata, not a
	// profile: one dynamic call may perform arbitrarily much runtime work.
	Coverage JITCoverage
}

type JITCoverage struct {
	Expressions  int
	DynamicCalls int
	InlinedCalls int
}

// Call keeps the entry point, embedded constant roots, and source arguments
// reachable for the complete native invocation, including panic unwinding.
// Pointer-bearing JIT locals are owned by the generated frame and described by
// precise runtime/jit safepoint maps rather than heap shadow roots.
func (jep *JITEntryPoint) Call(args ...Scmer) (result Scmer) {
	if jep == nil || jep.Native == nil {
		panic("JIT: nil entry point")
	}
	if JITLog && jep.DebugName != "" {
		fmt.Printf("JIT: call %s argc=%d\n", jep.DebugName, len(args))
	}
	// A transferred list must own fresh backing, and appending hidden inputs must
	// not reuse caller-owned capacity. Stack maps make an extra copy unnecessary
	// merely because native code calls back into Go.
	stableArgs := jep.TransferInputArgs || len(jep.HiddenArgs) != 0
	if stableArgs {
		args = append([]Scmer(nil), args...)
	}
	if jep.Proc.Params.GetTag() == tagSlice {
		paramCount := len(jep.Proc.Params.Slice())
		if len(args) > paramCount {
			panic(fmt.Sprintf("Apply: function with %d parameters is supplied with %d arguments", paramCount, len(args)))
		}
		if len(args) < paramCount {
			padded := make([]Scmer, paramCount)
			copy(padded, args)
			for i := len(args); i < paramCount; i++ {
				padded[i] = NewNil()
			}
			args = padded
		}
	}
	for _, spec := range jep.HiddenArgs {
		switch spec.Kind {
		case jitHiddenPreallocatedSlice:
			if spec.SourceInput < 0 || spec.SourceInput >= len(args) {
				panic("JIT: invalid hidden argument input")
			}
			length := len(asSlice(args[spec.SourceInput], "jit preallocation"))
			args = append(args, NewSlice(make([]Scmer, length)))
		case jitHiddenOptimizedCallback:
			if spec.SourceInput < 0 || spec.SourceInput >= len(args) {
				panic("JIT: invalid hidden argument input")
			}
			args = append(args, NewFunc(OptimizeProcToSerialFunction(args[spec.SourceInput])))
		default:
			panic("JIT: invalid hidden argument kind")
		}
	}
	defer func() {
		runtime.KeepAlive(args)
		runtime.KeepAlive(jep)
	}()
	return callJIT(jep.Native, args...)
}

type jitHiddenArgKind uint8

const (
	jitHiddenPreallocatedSlice jitHiddenArgKind = iota
	jitHiddenOptimizedCallback
)

type JITHiddenArg struct {
	Kind        jitHiddenArgKind
	SourceInput int
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
//	LocRegTriple: Three raw Go ABI words, currently used for slice ptr/len/cap.
//	LocAny:     Result placement hint only ("I don't care where you put it").
type JITValueDesc struct {
	ID       uint32
	Type     uint8 // tag constant (tagInt, tagFloat, ...) or JITTypeUnknown
	Loc      JITLoc
	Reg      Reg
	Reg2     Reg     // second register (for Scmer: ptr+aux)
	StackOff int32   // stack offset (if Loc == LocStack)
	Reg3     Reg     // third register (for Go slices: ptr+len+cap); occupies former padding to keep the descriptor ABI stable
	MemPtr   uintptr // memory address (only if Loc == LocMem)
	Imm      Scmer   // compile-time constant (if Loc == LocImm); Imm.GetTag() carries type info
	// KnownSliceLen/Cap carry optimizer-proven bounds for a slice descriptor.
	// The boolean keeps the zero value unambiguously "unknown" while still
	// representing an empty slice exactly.
	KnownSliceLen  int32
	KnownSliceCap  int32
	SliceSizeKnown bool
	// NoHeapPointer proves that the Scmer ptr word is nil or points at immutable
	// runtime storage rather than the Go heap. It is intentionally independent
	// of Type so unions such as int|float|nil retain the fact without claiming an
	// exact tag.
	NoHeapPointer bool
	// Rooted means a pointer-bearing runtime value has an independently live Go
	// owner or an invocation-local slot covered by every later safepoint map.
	Rooted bool
	// Virtual holds compiler-only aggregate elements. LocVirtualSlice never
	// reaches generated machine code; consumers either operate on its elements
	// directly or materialize it through a Go allocation trampoline.
	Virtual []JITValueDesc
	// Lambda carries compiler-only code shape for a known lambda. Runtime
	// captures remain descriptors in Outer and are never materialized merely to
	// cross a generated emitter boundary.
	Lambda *JITLambdaTemplate
}

type JITLambdaTemplate struct {
	Proc  Proc
	Outer *JITEnv
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
	LocStackPair               // Two-word value in the current invocation's frame (StackOff, StackOff+8)
	LocMem                     // At a fixed memory address (MemPtr)
	LocImm                     // Compile-time constant (Imm)
	LocAny                     // "I don't care" — result may be constant, register, or memory
	// Append new locations to preserve the numeric ABI used by generated
	// emitters and already compiled JIT integration code.
	LocRegTriple   // In three registers (Reg=ptr, Reg2=len, Reg3=cap) — for Go slices
	LocStackTriple // Three-word value in the current invocation's frame (StackOff..StackOff+16)
	LocVirtualSlice
	LocInputPair // Compiler-only reference to one Scmer in the native call's original variadic slice
	LocLambdaTemplate
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
	stackOff int32
}

type jitStackRootBase uint8

const (
	jitStackRootFrameSP jitStackRootBase = iota
	jitStackRootFrameBP
	jitStackRootCallSP
)

type jitStackRoot struct {
	base   jitStackRootBase
	offset int32
}

// jitSafepoint is recorded while emitting a Go call. FrameSize is deliberately
// absent: the one-pass emitter only knows the final static frame size after the
// complete function has been written.
type jitSafepoint struct {
	pcOffset  int32
	dynamicSP int32
	roots     []jitStackRoot
}

// jitStackMap is the runtime-independent form passed through the common JIT
// code. The goexperiment.jit implementation converts it to runtime/jit maps;
// the vanilla implementation deliberately ignores it.
type jitStackMap struct {
	pcOffset   uintptr
	frameWords uintptr
	pointerMap []byte
}

// JITContext is the central structure for descriptor-based JIT compilation.
// W is a self-reference for backward compatibility with hand-written emitters
// that use ctx.W.EmitXxx() (from the pre-consolidation JITWriter era).
type JITContext struct {
	W     *JITContext    // self-reference (backward compat for ctx.W.Emit calls)
	Ptr   unsafe.Pointer // current write pointer (into mmap memory)
	End   unsafe.Pointer // page end minus reserve
	Start unsafe.Pointer // page start for position calculation

	Labels    [256]int32
	LabelNext uint8

	Fixups    [512]JITFixup
	FixupNext uint8

	Env       *JITEnv
	FreeRegs  uint64
	AllRegs   uint64 // original set of all allocatable registers (for spilling)
	SliceBase Reg    // register holding the args slice pointer (for variable-index access)
	// OriginalArgsOff stores the incoming variadic slice data pointer in the
	// invocation-local frame.
	// Optimized local frames may repurpose SliceBase, while hidden GC roots still
	// live after the source-level arguments in the original Go-owned slice.
	OriginalArgsOff int32
	// SliceBaseTracksRSP indicates that SliceBase is a mirror of RSP and must be
	// refreshed after helper calls (Go may grow/move the goroutine stack).
	SliceBaseTracksRSP bool
	// InputArgCount is the fixed source-level parameter count. Virtual list
	// arguments may refer to these input pairs without loading them eagerly;
	// materializing a normal list still allocates fresh backing storage.
	InputArgCount int
	// LocalSlotCount is the number of 16-byte Scmer slots reserved in the
	// invocation frame. Optimizer-internal !list values may borrow a bounded
	// subrange while their NoEscape consumer is emitted inline.
	LocalSlotCount int
	// TransferInputArgs is set when emission proves that the native result is
	// exactly the complete variadic input array re-tagged as an owned list.
	TransferInputArgs bool
	HiddenArgs        []JITHiddenArg
	RuntimeEnv        Scmer
	AutoImportSafe    bool
	RecursiveLambdas  bool
	NeedsStableArgs   bool
	RegOwners         [16]*JITValueDesc // register → owner descriptor (nil = untracked)
	// DynamicSP is the temporary distance below the static frame bottom. It
	// covers pushed live registers, variadic arrays, and the Go call area.
	DynamicSP int32
	// StackRoots contains pointer words that are live in the static frame at the
	// current emission point. Safepoints copy this set before sibling control
	// flow can restore a different allocator state.
	StackRoots map[jitStackRoot]struct{}
	Safepoints []jitSafepoint
	Coverage   JITCoverage

	// Stack frame: emitter locals use [RSP + offset], while register spills use
	// [RBP - offset]. The two zones cannot overlap because the patched frame size
	// is MaxBPOffset + MaxSpillOffset. Epilog: leave; ret.
	BPOffset       int32 // current stack allocation point (grows on alloc, shrinks on free)
	MaxBPOffset    int32 // local-zone high-water mark
	SpillOffset    int32 // current spill-zone allocation point below RBP
	MaxSpillOffset int32 // spill-zone high-water mark

	ProtectedRegs      uint64  // bitmask of registers that must not be spilled
	ProtectedRegCounts [16]int // per-register protection refcount (supports nested protection)
	nextDescID         uint32
	descOwners         map[uint32]*JITValueDesc
	descSpills         map[uint32]descSpillMeta
	// ConstRoots holds pointer payloads from LocImm Scmer values that were
	// materialized into machine code immediates. Keeping these pointers in a
	// Go heap object reachable from JITEntryPoint prevents GC from reclaiming
	// referenced heap data while JIT code may still dereference it.
	ConstRoots []unsafe.Pointer
	rootSet    map[unsafe.Pointer]struct{}
	Arena      *jitArena // owning arena for source map entries
}

func (ctx *JITContext) RequestPreallocatedSlice(lengthInput int) JITValueDesc {
	hiddenIndex := ctx.InputArgCount + len(ctx.HiddenArgs)
	ctx.HiddenArgs = append(ctx.HiddenArgs, JITHiddenArg{Kind: jitHiddenPreallocatedSlice, SourceInput: lengthInput})
	return JITValueDesc{Loc: LocInputPair, Type: tagSlice, StackOff: int32(hiddenIndex)}
}

func (ctx *JITContext) RequestOptimizedCallback(sourceInput int) JITValueDesc {
	hiddenIndex := ctx.InputArgCount + len(ctx.HiddenArgs)
	ctx.HiddenArgs = append(ctx.HiddenArgs, JITHiddenArg{Kind: jitHiddenOptimizedCallback, SourceInput: sourceInput})
	return JITValueDesc{Loc: LocInputPair, Type: tagFunc, StackOff: int32(hiddenIndex)}
}

// jitAllocStateSnapshot captures allocator/spill bookkeeping so emitter
// generation can render sibling BBs from identical allocator state.
type jitAllocStateSnapshot struct {
	freeRegs           uint64
	protectedRegs      uint64
	protectedRegCounts [16]int
	regOwnerIDs        [16]uint32
	ownerValues        map[uint32]JITValueDesc
	spillOffset        int32
	descSpills         map[uint32]descSpillMeta
	stackRoots         map[jitStackRoot]struct{}
	dynamicSP          int32
}

func (ctx *JITContext) SnapshotAllocState() jitAllocStateSnapshot {
	s := jitAllocStateSnapshot{
		freeRegs:           ctx.FreeRegs,
		protectedRegs:      ctx.ProtectedRegs,
		protectedRegCounts: ctx.ProtectedRegCounts,
		spillOffset:        ctx.SpillOffset,
		dynamicSP:          ctx.DynamicSP,
	}
	if len(ctx.descOwners) != 0 {
		s.ownerValues = make(map[uint32]JITValueDesc, len(ctx.descOwners))
		for id, owner := range ctx.descOwners {
			if owner == nil {
				continue
			}
			s.ownerValues[id] = *owner
		}
	}
	for r := Reg(0); r <= RegR15; r++ {
		if owner := ctx.RegOwners[r]; owner != nil {
			s.regOwnerIDs[r] = owner.ID
		}
	}
	if len(ctx.descSpills) != 0 {
		s.descSpills = make(map[uint32]descSpillMeta, len(ctx.descSpills))
		for k, v := range ctx.descSpills {
			s.descSpills[k] = v
		}
	}
	if len(ctx.StackRoots) != 0 {
		s.stackRoots = make(map[jitStackRoot]struct{}, len(ctx.StackRoots))
		for root := range ctx.StackRoots {
			s.stackRoots[root] = struct{}{}
		}
	}
	return s
}

func (ctx *JITContext) RestoreAllocState(s jitAllocStateSnapshot) {
	ctx.FreeRegs = s.freeRegs
	ctx.ProtectedRegs = s.protectedRegs
	ctx.ProtectedRegCounts = s.protectedRegCounts
	ctx.SpillOffset = s.spillOffset
	// Descriptor identities are global to one emitted function. Restoring an
	// older basic-block snapshot must not make later descriptors reuse IDs whose
	// spill metadata was already emitted on a sibling path.
	ctx.DynamicSP = s.dynamicSP

	if s.ownerValues == nil {
		ctx.descOwners = nil
	} else {
		ctx.descOwners = make(map[uint32]*JITValueDesc, len(s.ownerValues))
		for id, owner := range s.ownerValues {
			copyOwner := owner
			ctx.descOwners[id] = &copyOwner
		}
	}
	for r := Reg(0); r <= RegR15; r++ {
		id := s.regOwnerIDs[r]
		if id == 0 || ctx.descOwners == nil {
			ctx.RegOwners[r] = nil
			continue
		}
		ctx.RegOwners[r] = ctx.descOwners[id]
	}

	if s.descSpills == nil {
		ctx.descSpills = nil
	} else {
		ctx.descSpills = make(map[uint32]descSpillMeta, len(s.descSpills))
		for k, v := range s.descSpills {
			ctx.descSpills[k] = v
		}
	}
	if s.stackRoots == nil {
		ctx.StackRoots = nil
	} else {
		ctx.StackRoots = make(map[jitStackRoot]struct{}, len(s.stackRoots))
		for root := range s.stackRoots {
			ctx.StackRoots[root] = struct{}{}
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
	// PhiBase is the base stack offset (relative to emitter RSP frame) for this
	// BB's phi slots. Each slot is 16 bytes (pair-aligned) and indexed by phi id.
	PhiBase int32
	// PhiCount is the number of phi slots associated with this BB.
	PhiCount uint16
	Pending  []JITFixup
	// VisitCount tracks how often this BB descriptor has been entered.
	// Unroll/specialization limits are derived from this per-BB state.
	VisitCount uint16
	// RenderCount is kept for compatibility with older generated emitters.
	RenderCount uint16
}

// AllocStack reserves size bytes on the stack frame. Returns the start offset
// from RBP (positive; first byte at [RBP - returnedOffset], last at [RBP - returnedOffset - size + 1]).
// Caller must FreeStack(size) when done.
func (ctx *JITContext) AllocStack(size int32) int32 {
	start := ctx.BPOffset
	ctx.BPOffset += size
	if ctx.BPOffset > ctx.MaxBPOffset {
		ctx.MaxBPOffset = ctx.BPOffset
	}
	return start
}

// FreeStack releases size bytes from the stack frame.
func (ctx *JITContext) FreeStack(size int32) {
	newOffset := ctx.BPOffset - size
	physicalStart := newOffset - ctx.DynamicSP
	physicalEnd := ctx.BPOffset - ctx.DynamicSP
	for root := range ctx.StackRoots {
		if root.base == jitStackRootFrameSP && root.offset >= physicalStart && root.offset < physicalEnd {
			delete(ctx.StackRoots, root)
		}
	}
	ctx.BPOffset -= size
}

func (ctx *JITContext) setStackPointer(base jitStackRootBase, offset int32, pointer bool) {
	root := jitStackRoot{base: base, offset: offset}
	if pointer {
		if ctx.StackRoots == nil {
			ctx.StackRoots = make(map[jitStackRoot]struct{})
		}
		ctx.StackRoots[root] = struct{}{}
		return
	}
	delete(ctx.StackRoots, root)
}

// AllocSpill reserves a slot in the invocation-local spill zone immediately
// below RBP. It returns a negative displacement suitable for [RBP+disp].
func (ctx *JITContext) AllocSpill(size int32) int32 {
	ctx.SpillOffset += size
	if ctx.SpillOffset > ctx.MaxSpillOffset {
		ctx.MaxSpillOffset = ctx.SpillOffset
	}
	return -ctx.SpillOffset
}

// TrackImm records a LocImm constant's pointer payload as a GC root when needed.
func (ctx *JITContext) TrackImm(v Scmer) {
	if ctx == nil {
		return
	}
	if v.ptr == nil {
		return
	}
	p := unsafe.Pointer(v.ptr)
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
		case LocRegTriple:
			valid = owner.Reg == rr || owner.Reg2 == rr || owner.Reg3 == rr
		}
		if !valid {
			ctx.RegOwners[rr] = nil
			ctx.FreeRegs |= bit
		}
	}
}

type jitNestedPreservation struct {
	alloc jitAllocStateSnapshot
	regs  []Reg
	offs  []int32
}

// PreserveOuterRegs makes nested emission independent of registers whose
// identities have already been embedded in outer control flow. The values are
// saved in invocation-local spill slots, all allocator registers become
// available to the nested emitter, and RestoreOuterRegs reloads the exact same
// registers before outer code resumes.
func (ctx *JITContext) PreserveOuterRegs() jitNestedPreservation {
	p := jitNestedPreservation{alloc: ctx.SnapshotAllocState()}
	ctx.ReclaimUntrackedRegs()
	for r := Reg(0); r <= RegR15; r++ {
		owner := ctx.RegOwners[r]
		if owner == nil || (ctx.AllRegs&(1<<uint(r))) == 0 || (ctx.FreeRegs&(1<<uint(r))) != 0 {
			continue
		}
		off := ctx.AllocSpill(8)
		ctx.EmitStoreRegMem(r, RegRBP, off)
		p.regs = append(p.regs, r)
		p.offs = append(p.offs, off)
		ctx.RegOwners[r] = nil
		ctx.FreeRegs |= 1 << uint(r)
	}
	ctx.ProtectedRegs = 0
	ctx.ProtectedRegCounts = [16]int{}
	return p
}

func (ctx *JITContext) RestoreOuterRegs(p jitNestedPreservation) {
	ctx.ReclaimUntrackedRegs()
	for i, r := range p.regs {
		ctx.EmitMovRegMem(r, RegRBP, p.offs[i])
	}
	ctx.RestoreAllocState(p.alloc)
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
		case LocRegTriple:
			valid = owner.Reg == rr || owner.Reg2 == rr || owner.Reg3 == rr
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
	tripleSpill := false
	var spillR1, spillR2, spillR3 Reg
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
			spillR1 = owner.Reg
			spillR2 = owner.Reg2
			// Pair spill must evict both registers atomically; if either register
			// is currently protected, try another candidate.
			if (ctx.ProtectedRegs&(1<<uint(spillR1))) != 0 || (ctx.ProtectedRegs&(1<<uint(spillR2))) != 0 {
				continue
			}
			r = rbit
			pairSpill = true
		case LocRegTriple:
			if owner.Reg != rbit && owner.Reg2 != rbit && owner.Reg3 != rbit {
				continue
			}
			spillR1 = owner.Reg
			spillR2 = owner.Reg2
			spillR3 = owner.Reg3
			if (ctx.ProtectedRegs&(1<<uint(spillR1))) != 0 ||
				(ctx.ProtectedRegs&(1<<uint(spillR2))) != 0 ||
				(ctx.ProtectedRegs&(1<<uint(spillR3))) != 0 {
				continue
			}
			r = rbit
			tripleSpill = true
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
				ownerDump += fmt.Sprintf(" r%d(loc=%d reg=%d reg2=%d reg3=%d)", rr, o.Loc, o.Reg, o.Reg2, o.Reg3)
			}
		}
		panic(fmt.Sprintf("jit: register spill required (fallback) free=%#x all=%#x prot=%#x owners=%#x%s", ctx.FreeRegs, ctx.AllRegs, ctx.ProtectedRegs, ownerMask, ownerDump))
	}

	owner := ctx.RegOwners[r]
	if pairSpill {
		stackOff := ctx.AllocSpill(16)
		ctx.EmitStoreRegMem(spillR1, RegRBP, stackOff)
		ctx.EmitStoreRegMem(spillR2, RegRBP, stackOff+8)
		ctx.setStackPointer(jitStackRootFrameBP, stackOff, !owner.NoHeapPointer)
		owner.Loc = LocStackPair
		owner.MemPtr = 0
		owner.StackOff = stackOff
		owner.Reg = 0
		owner.Reg2 = 0
		if owner.ID != 0 {
			if ctx.descSpills == nil {
				ctx.descSpills = make(map[uint32]descSpillMeta)
			}
			ctx.descSpills[owner.ID] = descSpillMeta{loc: LocStackPair, stackOff: stackOff}
		}
		ctx.RegOwners[spillR1] = nil
		ctx.RegOwners[spillR2] = nil
		return r
	}
	if tripleSpill {
		stackOff := ctx.AllocSpill(24)
		ctx.EmitStoreRegMem(spillR1, RegRBP, stackOff)
		ctx.EmitStoreRegMem(spillR2, RegRBP, stackOff+8)
		ctx.EmitStoreRegMem(spillR3, RegRBP, stackOff+16)
		ctx.setStackPointer(jitStackRootFrameBP, stackOff, true)
		owner.Loc = LocStackTriple
		owner.MemPtr = 0
		owner.StackOff = stackOff
		owner.Reg = 0
		owner.Reg2 = 0
		owner.Reg3 = 0
		if owner.ID != 0 {
			if ctx.descSpills == nil {
				ctx.descSpills = make(map[uint32]descSpillMeta)
			}
			ctx.descSpills[owner.ID] = descSpillMeta{loc: LocStackTriple, stackOff: stackOff}
		}
		ctx.RegOwners[spillR1] = nil
		ctx.RegOwners[spillR2] = nil
		ctx.RegOwners[spillR3] = nil
		return r
	}

	// Scalar spill: reserve a call-local slot in the generated function's frame.
	stackOff := ctx.AllocSpill(8)
	ctx.EmitStoreRegMem(r, RegRBP, stackOff)
	ctx.setStackPointer(jitStackRootFrameBP, stackOff, false)

	owner.Loc = LocStack
	owner.MemPtr = 0
	owner.StackOff = stackOff
	owner.Reg = 0
	if owner.ID != 0 {
		if ctx.descSpills == nil {
			ctx.descSpills = make(map[uint32]descSpillMeta)
		}
		ctx.descSpills[owner.ID] = descSpillMeta{loc: LocStack, stackOff: stackOff}
	}
	ctx.RegOwners[r] = nil
	return r
}

// EnsureDesc restores a descriptor from stack/spill locations to registers.
func (ctx *JITContext) syncDescSpill(desc *JITValueDesc) {
	if desc.Loc == LocReg && desc.ID != 0 && ctx.descSpills != nil {
		if meta, ok := ctx.descSpills[desc.ID]; ok && meta.loc == LocStack {
			desc.Loc = LocStack
			desc.MemPtr = 0
			desc.StackOff = meta.stackOff
			desc.Reg = 0
		}
	}
	if desc.Loc == LocRegPair && desc.ID != 0 && ctx.descSpills != nil {
		if meta, ok := ctx.descSpills[desc.ID]; ok && meta.loc == LocStackPair {
			desc.Loc = LocStackPair
			desc.MemPtr = 0
			desc.StackOff = meta.stackOff
			desc.Reg = 0
			desc.Reg2 = 0
		}
	}
	if desc.Loc == LocRegTriple && desc.ID != 0 && ctx.descSpills != nil {
		if meta, ok := ctx.descSpills[desc.ID]; ok && meta.loc == LocStackTriple {
			desc.Loc = LocStackTriple
			desc.MemPtr = 0
			desc.StackOff = meta.stackOff
			desc.Reg = 0
			desc.Reg2 = 0
			desc.Reg3 = 0
		}
	}
}

func (ctx *JITContext) EnsureDesc(desc *JITValueDesc) {
	ctx.syncDescSpill(desc)
	switch desc.Loc {
	case LocInputPair:
		r1 := ctx.AllocReg()
		r2 := ctx.AllocRegExcept(r1)
		base := ctx.SliceBase
		if ctx.SliceBaseTracksRSP && int(desc.StackOff) >= ctx.InputArgCount {
			base = ctx.AllocRegExcept(r1, r2)
			ctx.EmitMovRegMem(base, RegRSP, ctx.OriginalArgsOff)
		}
		ctx.EmitMovRegMem(r1, base, desc.StackOff*16)
		ctx.EmitMovRegMem(r2, base, desc.StackOff*16+8)
		if base != ctx.SliceBase {
			ctx.FreeReg(base)
		}
		desc.Loc = LocRegPair
		desc.Reg = r1
		desc.Reg2 = r2
		ctx.BindReg(r1, desc)
		ctx.BindReg(r2, desc)
	case LocStack:
		ctx.EnsureReg(desc)
	case LocStackPair:
		r1 := ctx.AllocReg()
		r2 := ctx.AllocRegExcept(r1)
		base := RegRSP
		if desc.StackOff < 0 {
			base = RegRBP
		}
		ctx.EmitMovRegMem(r1, base, desc.StackOff)
		ctx.EmitMovRegMem(r2, base, desc.StackOff+8)
		desc.Loc = LocRegPair
		desc.Reg = r1
		desc.Reg2 = r2
		desc.MemPtr = 0
		desc.StackOff = 0
		ctx.BindReg(r1, desc)
		ctx.BindReg(r2, desc)
	case LocStackTriple:
		r1 := ctx.AllocReg()
		r2 := ctx.AllocRegExcept(r1)
		r3 := ctx.AllocRegExcept(r1, r2)
		base := RegRSP
		if desc.StackOff < 0 {
			base = RegRBP
		}
		ctx.EmitMovRegMem(r1, base, desc.StackOff)
		ctx.EmitMovRegMem(r2, base, desc.StackOff+8)
		ctx.EmitMovRegMem(r3, base, desc.StackOff+16)
		desc.Loc = LocRegTriple
		desc.Reg = r1
		desc.Reg2 = r2
		desc.Reg3 = r3
		desc.MemPtr = 0
		desc.StackOff = 0
		ctx.BindReg(r1, desc)
		ctx.BindReg(r2, desc)
		ctx.BindReg(r3, desc)
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
		case LocRegTriple:
			if owner.Reg == r || owner.Reg2 == r || owner.Reg3 == r {
				for _, other := range [...]Reg{owner.Reg, owner.Reg2, owner.Reg3} {
					if other != r && other <= RegR15 {
						ctx.RegOwners[other] = nil
					}
				}
				owner.Loc = LocNone
				owner.Reg = 0
				owner.Reg2 = 0
				owner.Reg3 = 0
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
	base := RegRSP
	if desc.StackOff < 0 {
		base = RegRBP
	}
	ctx.EmitMovRegMem(r, base, desc.StackOff)
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
	case LocRegTriple:
		for _, r := range [...]Reg{desc.Reg, desc.Reg2, desc.Reg3} {
			if r > RegR15 {
				continue
			}
			owner := ctx.RegOwners[r]
			if owner == nil || owner == desc || (desc.ID != 0 && owner.ID == desc.ID) {
				ctx.FreeReg(r)
			}
		}
	case LocStack:
	case LocStackPair:
	case LocStackTriple:
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

func jitAddLambdaBoundParams(params Scmer, bound map[Symbol]struct{}) {
	if params.IsSourceInfo() {
		params = params.SourceInfo().value
	}
	switch params.GetTag() {
	case tagSlice:
		for _, p := range params.Slice() {
			if p.IsSourceInfo() {
				p = p.SourceInfo().value
			}
			if p.GetTag() == tagSymbol {
				bound[p.Symbol()] = struct{}{}
			}
		}
	case tagSymbol:
		bound[params.Symbol()] = struct{}{}
	}
}

func jitCollectLambdaFreeSymbols(expr Scmer, bound map[Symbol]struct{}, seen map[Symbol]struct{}, out *[]Symbol) {
	if expr.IsSourceInfo() {
		expr = expr.SourceInfo().value
	}
	switch expr.GetTag() {
	case tagSymbol:
		sym := expr.Symbol()
		if sym == Symbol("nil") {
			return
		}
		if _, isBound := bound[sym]; isBound {
			return
		}
		if _, exists := seen[sym]; exists {
			return
		}
		seen[sym] = struct{}{}
		*out = append(*out, sym)
	case tagSlice:
		list := expr.Slice()
		if len(list) == 0 {
			return
		}
		if list[0].IsSymbol() {
			switch string(list[0].Symbol()) {
			case "quote":
				return
			case "lambda":
				if len(list) < 3 {
					return
				}
				innerBound := make(map[Symbol]struct{}, len(bound)+4)
				for k := range bound {
					innerBound[k] = struct{}{}
				}
				jitAddLambdaBoundParams(list[1], innerBound)
				jitCollectLambdaFreeSymbols(list[2], innerBound, seen, out)
				return
			}
		}
		for _, item := range list {
			jitCollectLambdaFreeSymbols(item, bound, seen, out)
		}
	}
}

func jitLambdaFreeSymbols(params, body Scmer) []Symbol {
	bound := make(map[Symbol]struct{}, 8)
	jitAddLambdaBoundParams(params, bound)
	seen := make(map[Symbol]struct{}, 8)
	out := make([]Symbol, 0, 8)
	jitCollectLambdaFreeSymbols(body, bound, seen, &out)
	return out
}

func jitCollectLambdaOuterVarIndices(expr Scmer, seen map[NthLocalVar]struct{}, out *[]NthLocalVar) {
	if expr.IsSourceInfo() {
		expr = expr.SourceInfo().value
	}
	if expr.GetTag() != tagSlice {
		return
	}
	list := expr.Slice()
	if len(list) == 0 {
		return
	}
	if list[0].IsSymbol() {
		switch string(list[0].Symbol()) {
		case "quote":
			return
		case "outer":
			if len(list) == 2 {
				arg := list[1]
				if arg.IsSourceInfo() {
					arg = arg.SourceInfo().value
				}
				if arg.GetTag() == tagNthLocalVar {
					idx := arg.NthLocalVar()
					if _, ok := seen[idx]; !ok {
						seen[idx] = struct{}{}
						*out = append(*out, idx)
					}
				}
			}
		}
	}
	for _, item := range list {
		jitCollectLambdaOuterVarIndices(item, seen, out)
	}
}

func jitLambdaOuterVarIndices(body Scmer) []NthLocalVar {
	seen := make(map[NthLocalVar]struct{}, 4)
	out := make([]NthLocalVar, 0, 4)
	jitCollectLambdaOuterVarIndices(body, seen, &out)
	return out
}

// jitBuildLambdaClosure constructs a closure Proc from a lambda form plus
// captured symbol/value pairs:
//
//	[params, body, numVars, key1, val1, key2, val2, ...]
//
// keyN is either a Symbol (captured named variable) or an NthLocalVar
// (captured numbered outer variable).
func jitBuildLambdaClosure(args ...Scmer) Scmer {
	if len(args) < 3 {
		panic("jit: lambda builder expects params, body, numVars")
	}
	if (len(args)-3)%2 != 0 {
		panic("jit: lambda builder capture list must be symbol/value pairs")
	}
	params := args[0]
	body := args[1]
	numVars := int(ToInt(args[2]))

	maxIdx := -1
	symbolCap := 0
	for i := 3; i < len(args); i += 2 {
		key := args[i]
		if key.GetTag() == tagNthLocalVar {
			idx := int(key.NthLocalVar())
			if idx > maxIdx {
				maxIdx = idx
			}
		} else {
			symbolCap++
		}
	}

	var vars Vars
	if symbolCap > 0 {
		vars = make(Vars, symbolCap)
	}
	var varsNumbered []Scmer
	if maxIdx >= 0 {
		varsNumbered = make([]Scmer, maxIdx+1)
	}

	for i := 3; i < len(args); i += 2 {
		key := args[i]
		if key.GetTag() == tagNthLocalVar {
			varsNumbered[int(key.NthLocalVar())] = args[i+1]
			continue
		}
		sym := mustSymbol(key)
		vars[sym] = args[i+1]
	}
	captureEnv := &Env{
		Vars:         vars,
		VarsNumbered: varsNumbered,
		Outer:        &Globalenv,
		Nodefine:     false,
	}
	return NewProcStruct(Proc{
		Params:  params,
		Body:    body,
		En:      captureEnv,
		NumVars: numVars,
	})
}

func jitBuildCompiledLambdaClosure(args ...Scmer) Scmer {
	return jitCompile(jitBuildLambdaClosure(args...))
}

// jitRuntimeEnvFromCaptures reconstructs the visible lexical environment for
// special forms whose runtime contract consumes Scheme syntax as data. The
// first argument is the interpreter environment outside the native Proc; the
// remaining arguments are frame markers and symbol-or-numbered-key/value pairs.
var jitRuntimeEnvFrameMarker = &struct{}{}

func jitRuntimeEnvFromCaptures(args []Scmer) *Env {
	if len(args) == 0 || (len(args)-1)%2 != 0 {
		panic("jit: malformed runtime environment capture")
	}
	outer, ok := args[0].Any().(*Env)
	if !ok || outer == nil {
		panic("jit: invalid outer runtime environment")
	}
	type capturedFrame struct {
		vars     Vars
		numbered map[int]Scmer
	}
	frames := make([]capturedFrame, 0, 2)
	current := capturedFrame{}
	started := false
	for i := 1; i < len(args); i += 2 {
		key, value := args[i], args[i+1]
		if key.GetTag() == tagAny && key.Any() == jitRuntimeEnvFrameMarker {
			if started {
				frames = append(frames, current)
			}
			current = capturedFrame{}
			started = true
			continue
		}
		if key.IsNthLocalVar() {
			if current.numbered == nil {
				current.numbered = make(map[int]Scmer)
			}
			current.numbered[int(key.NthLocalVar())] = value
			continue
		}
		if current.vars == nil {
			current.vars = make(Vars)
		}
		current.vars[mustSymbol(key)] = value
	}
	if started || current.vars != nil || current.numbered != nil {
		frames = append(frames, current)
	}
	if len(frames) == 0 {
		return &Env{Outer: outer}
	}
	for i := len(frames) - 1; i >= 0; i-- {
		frame := frames[i]
		maxNumbered := -1
		for index := range frame.numbered {
			if index > maxNumbered {
				maxNumbered = index
			}
		}
		var numbered []Scmer
		if maxNumbered >= 0 {
			numbered = make([]Scmer, maxNumbered+1)
			for index, value := range frame.numbered {
				numbered[index] = value
			}
		}
		outer = &Env{Vars: frame.vars, VarsNumbered: numbered, Outer: outer}
	}
	return outer
}

func jitEvalSpecial(args ...Scmer) Scmer {
	if len(args) < 2 {
		panic("eval expects exactly one expression")
	}
	return Eval(args[0], jitRuntimeEnvFromCaptures(args[1:]))
}

func jitParserSpecial(args ...Scmer) Scmer {
	if len(args) < 5 {
		panic("parser expects syntax")
	}
	en := jitRuntimeEnvFromCaptures(args[4:])
	return NewScmParser(NewParser(args[0], args[1], args[2], en, args[3].Bool()))
}

type jitSpecialThunkValue struct {
	callable Scmer
	args     []Scmer
}

func jitMakeSpecialThunk(args ...Scmer) Scmer {
	if len(args) == 0 {
		panic("jit: special-form thunk expects a callable")
	}
	return NewAny(&jitSpecialThunkValue{callable: args[0], args: append([]Scmer(nil), args[1:]...)})
}

func jitCallSpecialThunk(thunk Scmer) Scmer {
	if thunk.GetTag() == tagAny {
		if value, ok := thunk.Any().(*jitSpecialThunkValue); ok && value != nil {
			if value.callable.GetTag() == tagProc {
				if proc := value.callable.Proc(); proc != nil && proc.Compiled != nil {
					return proc.Compiled.Call(value.args...)
				}
			}
			return Apply(value.callable, value.args...)
		}
	}
	if thunk.GetTag() == tagProc {
		if proc := thunk.Proc(); proc != nil && proc.Compiled != nil {
			return proc.Compiled.Call()
		}
	}
	return Apply(thunk)
}

func jitOptimizerProcReturnSpecial(args ...Scmer) Scmer {
	if len(args) != 2 {
		panic("optimizer_proc_return expects procedure and return metadata")
	}
	value, metadataThunk := args[0], args[1]
	if value.GetTag() != tagProc {
		return value
	}
	metadataValue := jitCallSpecialThunk(metadataThunk)
	if metadataValue.GetTag() != tagAny {
		panic("optimizer_proc_return expects internal return metadata")
	}
	metadata, ok := metadataValue.Any().(optimizerProcReturnTemplate)
	if !ok {
		panic("optimizer_proc_return received invalid return metadata")
	}
	proc := *value.Proc()
	proc.OptimizerMeta = &ProcOptimizerMeta{Return: metadata.Return, HasReturn: metadata.HasReturn}
	return NewProcStruct(proc)
}

func jitTimeSpecial(args ...Scmer) Scmer {
	if len(args) != 2 {
		panic("time expects an expression and optional label")
	}
	body, label := args[0], args[1]
	hasLabel := !label.IsNil()
	var start time.Time
	if TracePrint {
		start = time.Now()
	}
	var timedResult Scmer
	if Trace != nil {
		traceLabel := "(time)"
		if hasLabel {
			traceLabel = String(jitCallSpecialThunk(label))
		}
		Trace.Duration(traceLabel, "scm", func() {
			timedResult = jitCallSpecialThunk(body)
		})
	} else {
		timedResult = jitCallSpecialThunk(body)
	}
	if TracePrint {
		message := "trace " + time.Since(start).String()
		if hasLabel {
			message += " " + String(jitCallSpecialThunk(label))
		}
		TracePrintFunc(message)
	}
	return timedResult
}

func jitParallelSpecial(thunks ...Scmer) Scmer {
	if len(thunks) == 0 {
		return NewNil()
	}
	errs := make(chan any, len(thunks))
	for _, thunk := range thunks {
		thunk := thunk
		gls.Go(func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					errs <- recovered
				} else {
					errs <- nil
				}
			}()
			jitCallSpecialThunk(thunk)
		})
	}
	for range thunks {
		if err := <-errs; err != nil {
			panic(err)
		}
	}
	return NewNil()
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

// jitPanic forwards a JIT panic payload into Go panic handling.
func jitPanic(v Scmer) {
	panic(v)
}

// JITPanic forwards panic payloads from cross-package JIT emitters.
func JITPanic(v Scmer) {
	jitPanic(v)
}

// GoABIIntRegs lists integer argument/result registers in Go ABIInternal order.
// R11 is reserved as scratch/closure context and is not an argument register;
// words after R10 are passed in the caller's stack argument area.
var GoABIIntRegs = []Reg{RegRAX, RegRBX, RegRCX, RegRDI, RegRSI, RegR8, RegR9, RegR10}

type goCallArgWord struct {
	loc      JITLoc
	reg      Reg
	imm      uint64
	stackOff int32
}

func (ctx *JITContext) collectLiveRegsForCall(buf *[16]Reg) []Reg {
	// Only allocator-owned registers can contain live SSA values. Reserved ABI,
	// scratch and frame registers are handled explicitly by the call emitter.
	allocatedMask := ctx.AllRegs &^ ctx.FreeRegs
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
type jitRegMove struct {
	dst Reg
	src Reg
}

// emitParallelRegMoves preserves every source until its last consumer has
// moved. R11 is reserved as emitter scratch and breaks register cycles.
func (ctx *JITContext) emitParallelRegMoves(moves []jitRegMove) {
	for len(moves) > 0 {
		emitIdx := -1
		for i := range moves {
			dstIsPendingSrc := false
			for j := range moves {
				if i != j && moves[j].src == moves[i].dst {
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
			cycleDst := moves[0].dst
			if cycleDst != RegR11 {
				ctx.emitMovRegReg(RegR11, cycleDst)
			}
			for i := range moves {
				if moves[i].src == cycleDst {
					moves[i].src = RegR11
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
}

func (ctx *JITContext) EmitGoCall(funcAddr uint64, argWords []goCallArgWord, numResultWords int, resultsBuf *[16]Reg, resultTargets []Reg) []Reg {
	ctx.NeedsStableArgs = true
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
		if ctx.SliceBaseTracksRSP {
			break
		}
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
		// Stack arguments are written before register shuffling, while every
		// source descriptor still names its original value. Go ABIInternal puts
		// words after the eighth integer register at consecutive caller-SP slots.
		for i := len(GoABIIntRegs); i < len(argWords); i++ {
			dstOff := int32((i - len(GoABIIntRegs)) * 8)
			switch argWords[i].loc {
			case LocReg:
				ctx.EmitStoreRegMem(argWords[i].reg, RegRSP, dstOff)
			case LocImm:
				ctx.EmitMovRegImm64(RegR11, argWords[i].imm)
				ctx.EmitStoreRegMem(RegR11, RegRSP, dstOff)
			case LocStack:
				if argWords[i].stackOff < 0 {
					ctx.EmitMovRegMem(RegR11, RegRBP, argWords[i].stackOff)
				} else {
					ctx.EmitMovRegMem(RegR11, RegRSP, stackArgBaseDisp+argWords[i].stackOff)
				}
				ctx.EmitStoreRegMem(RegR11, RegRSP, dstOff)
			case LocInputPair:
				ctx.EmitMovRegMem(RegR11, ctx.SliceBase, argWords[i].stackOff)
				ctx.EmitStoreRegMem(RegR11, RegRSP, dstOff)
			default:
				panic("jit: unsupported Go-call stack arg location")
			}
		}

		moves := make([]jitRegMove, 0, len(argWords))
		regWordCount := len(argWords)
		if regWordCount > len(GoABIIntRegs) {
			regWordCount = len(GoABIIntRegs)
		}
		for i := 0; i < regWordCount; i++ {
			target := GoABIIntRegs[i]
			if argWords[i].loc == LocReg && argWords[i].reg != target {
				moves = append(moves, jitRegMove{dst: target, src: argWords[i].reg})
			}
		}
		ctx.emitParallelRegMoves(moves)

		for i := 0; i < regWordCount; i++ {
			target := GoABIIntRegs[i]
			switch argWords[i].loc {
			case LocReg:
				// Already handled by move planner (including no-op src==target).
			case LocImm:
				ctx.EmitMovRegImm64(target, argWords[i].imm)
			case LocStack:
				if argWords[i].stackOff < 0 {
					ctx.EmitMovRegMem(target, RegRBP, argWords[i].stackOff)
				} else {
					ctx.EmitMovRegMem(target, RegRSP, stackArgBaseDisp+argWords[i].stackOff)
				}
			case LocInputPair:
				ctx.EmitMovRegMem(target, ctx.SliceBase, argWords[i].stackOff)
			default:
				panic("jit: unsupported Go-call arg location")
			}
		}
	}

	// Fast path: no live registers to preserve. Emit only argument setup + call.
	if len(liveRegs) == 0 {
		ctx.emitCallIndirectWithSetup(funcAddr, func(callFrameBytes int32) {
			emitArgSetup(callFrameBytes)
		}, nil)
		if ctx.SliceBaseTracksRSP && ctx.SliceBase != RegRSP {
			ctx.emitMovRegReg(ctx.SliceBase, RegRSP)
		}
		moves := make([]jitRegMove, 0, numResultWords)
		for i := 0; i < numResultWords; i++ {
			var r Reg
			if i < len(resultTargets) {
				r = resultTargets[i]
			} else {
				// All Go ABI result registers remain live until every result word
				// has been copied. In particular, a three-word slice returns its
				// capacity in RCX; selecting RCX for the earlier data pointer would
				// overwrite that capacity before it is read.
				r = ctx.AllocRegExcept(GoABIIntRegs[:numResultWords]...)
			}
			if r != GoABIIntRegs[i] {
				moves = append(moves, jitRegMove{dst: r, src: GoABIIntRegs[i]})
			}
			resultsBuf[i] = r
		}
		ctx.emitParallelRegMoves(moves)
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
		ctx.DynamicSP += int32(resultBytes)
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
	transientRoots := make([]int32, 0, len(liveRegs))
	paddingBytes := int32(0)
	if padded {
		paddingBytes = 8
	}
	for i, r := range liveRegs {
		if ctx.regHoldsPointer(r) || r == ctx.SliceBase {
			transientRoots = append(transientRoots, paddingBytes+int32(len(liveRegs)-1-i)*8)
		}
	}
	// Move argument words into Go ABI registers (clobber-safe planner).
	stackArgBaseDisp := int32(resultBytes + len(liveRegs)*8)
	if padded {
		stackArgBaseDisp += 8
	}

	// CALL. Argument setup happens after the JIT unwind/spill area has been
	// allocated, because Go ABIInternal stack arguments start at the final
	// caller SP and are followed by the register spill slots.
	ctx.emitCallIndirectWithSetup(funcAddr, func(callFrameBytes int32) {
		emitArgSetup(stackArgBaseDisp + callFrameBytes)
	}, transientRoots)

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
		var r Reg
		if i < len(resultTargets) {
			r = resultTargets[i]
		} else {
			r = ctx.AllocReg()
		}
		ctx.EmitPopReg(r)
		resultsBuf[i] = r
	}
	if ctx.SliceBaseTracksRSP && ctx.SliceBase != RegRSP {
		ctx.emitMovRegReg(ctx.SliceBase, RegRSP)
	}
	return resultsBuf[:numResultWords]
}

// flattenArgs converts JITValueDesc arguments to ABI words.
// LocRegTriple → 3 words (Reg, Reg2, Reg3), LocRegPair → 2 words,
// LocReg → 1 word, LocImm → deferred imm.
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
		case LocRegTriple:
			buf[n] = goCallArgWord{loc: LocReg, reg: a.Reg}
			n++
			buf[n] = goCallArgWord{loc: LocReg, reg: a.Reg2}
			n++
			buf[n] = goCallArgWord{loc: LocReg, reg: a.Reg3}
			n++
		case LocReg:
			buf[n] = goCallArgWord{loc: LocReg, reg: a.Reg}
			n++
		case LocStack:
			buf[n] = goCallArgWord{loc: LocStack, stackOff: a.StackOff}
			n++
		case LocStackPair:
			buf[n] = goCallArgWord{loc: LocStack, stackOff: a.StackOff}
			n++
			buf[n] = goCallArgWord{loc: LocStack, stackOff: a.StackOff + 8}
			n++
		case LocInputPair:
			inputOff := a.StackOff * 16
			buf[n] = goCallArgWord{loc: LocInputPair, stackOff: inputOff}
			n++
			buf[n] = goCallArgWord{loc: LocInputPair, stackOff: inputOff + 8}
			n++
		case LocStackTriple:
			buf[n] = goCallArgWord{loc: LocStack, stackOff: a.StackOff}
			n++
			buf[n] = goCallArgWord{loc: LocStack, stackOff: a.StackOff + 8}
			n++
			buf[n] = goCallArgWord{loc: LocStack, stackOff: a.StackOff + 16}
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
	results := ctx.EmitGoCall(funcAddr, words, numResultWords, &resultsBuf, nil)
	// Result registers are already allocated by EmitGoCall (removed from FreeRegs).
	// Set RegOwners to nil — the caller MUST BindReg to a long-lived descriptor.
	// The nil ownership prevents AllocReg's spill path from evicting the result.
	for i := 0; i < numResultWords && i < len(results); i++ {
		ctx.RegOwners[results[i]] = nil
	}
	if numResultWords == 1 {
		return JITValueDesc{Loc: LocReg, Reg: results[0]}
	}
	if numResultWords == 3 {
		return JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: results[0], Reg2: results[1], Reg3: results[2]}
	}
	return JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: results[0], Reg2: results[1]}
}

// EmitGoCallScalarInto emits a scalar Go call directly into a fixed result
// placement. This avoids an allocator-register round trip when the enclosing
// JIT function returns the Go result in ABI registers unchanged.
func (ctx *JITContext) EmitGoCallScalarInto(funcAddr uint64, args []JITValueDesc, result JITValueDesc) JITValueDesc {
	if result.Loc != LocRegPair {
		panic("jit: fixed Go-call result currently requires LocRegPair")
	}
	var wordsBuf [16]goCallArgWord
	var resultsBuf [16]Reg
	words := ctx.flattenArgs(args, &wordsBuf)
	targets := [...]Reg{result.Reg, result.Reg2}
	results := ctx.EmitGoCall(funcAddr, words, 2, &resultsBuf, targets[:])
	result.Loc = LocRegPair
	result.Reg = results[0]
	result.Reg2 = results[1]
	ctx.BindReg(result.Reg, &result)
	ctx.BindReg(result.Reg2, &result)
	return result
}

// EmitMovPairToResult moves a LocRegPair value into the result descriptor registers.
func (ctx *JITContext) EmitMovPairToResult(src *JITValueDesc, dst *JITValueDesc) {
	if src.Reg != dst.Reg && src.Reg2 == dst.Reg {
		// Preserve the pointer word before writing the aux word into its source
		// register. This includes the full register-swap case.
		ctx.emitMovRegReg(RegR11, src.Reg)
		if src.Reg2 != dst.Reg2 {
			ctx.emitMovRegReg(dst.Reg2, src.Reg2)
		}
		ctx.emitMovRegReg(dst.Reg, RegR11)
		return
	}
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
	ctx.EmitGoCall(funcAddr, words, 0, &resultsBuf, nil)
}

// ---- merged from scm/jit_writer.go ----

// jitSourceEntry maps a code offset within an arena to a Scheme source location.
type jitSourceEntry struct {
	offset int32  // byte offset from arena base
	file   string // source file name
	line   int32  // 1-based line number
}

type jitSourceMap struct {
	entries []jitSourceEntry
}

// jitArena is a large mmap'd buffer, optionally registered with
// runtime/jit for unwinding (when built with GOEXPERIMENT=jit).
type jitArena struct {
	base   unsafe.Pointer // start of mmap'd region
	size   int            // total bytes
	offset int            // bump pointer (next free byte), guarded by jitPool.mu
	handle interface{}    // opaque registration handle (nil = unregistered)

	sourceMu      sync.Mutex
	sourceEntries []jitSourceEntry
	sourceMap     atomic.Pointer[jitSourceMap]
	metaMu        sync.Mutex
	metaCond      *sync.Cond
	reservations  []*jitCodeReservation
	metaNext      int
}

type jitCodeReservation struct {
	offset    int
	done      bool
	published bool
	maps      []jitStackMap
}

// complete publishes arena metadata in allocation order. Compilation itself
// may run concurrently, but an entry point does not become reachable before
// every earlier reservation has either published its maps or reported failure.
func (a *jitArena) complete(reservation *jitCodeReservation, maps []jitStackMap) {
	a.completeMode(reservation, maps, true)
}

// completeDeferred records metadata for code which cannot become reachable
// before its enclosing reservation is published. Nested special-form thunks
// use this to avoid waiting on the outer compiler which is currently emitting
// them; the outer completion publishes both reservations in allocation order.
func (a *jitArena) completeDeferred(reservation *jitCodeReservation, maps []jitStackMap) {
	a.completeMode(reservation, maps, false)
}

func (a *jitArena) completeMode(reservation *jitCodeReservation, maps []jitStackMap, wait bool) {
	if a == nil || reservation == nil {
		return
	}
	a.metaMu.Lock()
	reservation.maps = maps
	reservation.done = true
	for a.metaNext < len(a.reservations) && a.reservations[a.metaNext].done {
		ready := a.reservations[a.metaNext]
		publishJITStackMaps(a, ready.maps)
		ready.published = true
		a.metaNext++
	}
	a.metaCond.Broadcast()
	if wait {
		for !reservation.published {
			a.metaCond.Wait()
		}
	}
	a.metaMu.Unlock()
}

// addSourceEntry publishes an immutable, offset-sorted source-map snapshot.
// Runtime traceback callbacks read it without locks or allocations.
func (a *jitArena) addSourceEntry(entry jitSourceEntry) {
	if a == nil || entry.file == "" {
		return
	}
	a.sourceMu.Lock()
	a.sourceEntries = append(a.sourceEntries, entry)
	entries := append([]jitSourceEntry(nil), a.sourceEntries...)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].offset < entries[j].offset
	})
	a.sourceMap.Store(&jitSourceMap{entries: entries})
	a.sourceMu.Unlock()
}

func (a *jitArena) loadSourceEntries() []jitSourceEntry {
	if a == nil {
		return nil
	}
	snapshot := a.sourceMap.Load()
	if snapshot == nil {
		return nil
	}
	return snapshot.entries
}

// jitPool manages global JIT arena allocation.
type jitPool struct {
	mu     sync.Mutex
	arenas []*jitArena
}

const jitArenaSize = 1 << 20 // 1 MB per arena

// globalJITPool is the singleton arena pool.
var globalJITPool jitPool

// Alloc bump-allocates size bytes from the pool, 16-byte aligned.
func (p *jitPool) Alloc(size int) (ptr unsafe.Pointer, arena *jitArena, reservation *jitCodeReservation) {
	size = (size + 15) & ^15 // align to 16 bytes
	p.mu.Lock()
	defer p.mu.Unlock()

	// Try current arena
	if len(p.arenas) > 0 {
		a := p.arenas[len(p.arenas)-1]
		if a.offset+size <= a.size {
			ptr = unsafe.Add(a.base, a.offset)
			reservation = &jitCodeReservation{offset: a.offset}
			a.metaMu.Lock()
			a.reservations = append(a.reservations, reservation)
			a.metaMu.Unlock()
			a.offset += size
			return ptr, a, reservation
		}
	}

	// Allocate new arena
	arenaBytes := jitArenaSize
	if size > arenaBytes {
		arenaBytes = (size + 4095) & ^4095
	}
	b, err := syscall.Mmap(-1, 0, arenaBytes,
		syscall.PROT_READ|syscall.PROT_WRITE|syscall.PROT_EXEC,
		syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		panic("jit: mmap arena failed: " + err.Error())
	}
	a := &jitArena{
		base: unsafe.Pointer(&b[0]),
		size: arenaBytes,
	}
	a.metaCond = sync.NewCond(&a.metaMu)
	a.handle = registerJITArena(a)
	ptr = a.base
	reservation = &jitCodeReservation{}
	a.reservations = append(a.reservations, reservation)
	a.offset = size
	p.arenas = append(p.arenas, a)
	return ptr, a, reservation
}

// Free returns a code region to the arena. Currently a no-op placeholder;
// future: maintain a freelist and coalesce adjacent free blocks.
func (p *jitPool) Free(ptr unsafe.Pointer, size int) {
	// no-op for now — arenas are long-lived
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

// ---- merged from scm/jit.go ----

var JITLog bool

func init_jit() {
	DeclareTitle("JIT Compilation")

	Declare(&Globalenv, &Declaration{
		Name: "jit",

		Fn: jitCompile,
		Type: &TypeDescriptor{Kind: "func", Description: "compiles a lambda to optimized native code when this build enables JIT",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "fn", Description: "the function to compile"},
			},
			Return:         &TypeDescriptor{Kind: "any"},
			HasSideEffects: true,
			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["jit"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "jit?",

		Fn: func(a ...Scmer) Scmer {
			return NewBool(a[0].GetTag() == tagJIT || (a[0].GetTag() == tagProc && a[0].Proc() != nil && a[0].Proc().Compiled != nil))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "tells whether a value is a JIT-compiled function descriptor",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "value", Description: "value to inspect", NoEscape: true},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,
			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				argPinned0 := make([]Reg, 0, len(args)*2)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned0 = append(argPinned0, ai.Reg2)
						}
					}
				}
				d1 := args[0]
				d1.ID = 0
				d2 := ctx.EmitGetTagDesc(&d1, JITValueDesc{Loc: LocAny})
				ctx.FreeDesc(&d1)
				ctx.EnsureDesc(&d2)
				var d3 JITValueDesc
				if d2.Loc == LocImm {
					d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d2.Imm.Int()) == uint64(11))}
				} else {
					r0 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d2.Reg, 11)
					ctx.EmitSetcc(r0, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
					ctx.BindReg(r0, &d3)
				}
				ctx.FreeDesc(&d2)
				ctx.EnsureDesc(&d3)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d3.Loc == LocImm {
					ctx.EmitMakeBool(result, d3)
				} else {
					ctx.EmitMakeBool(result, d3)
					ctx.FreeReg(d3.Reg)
				}
				result.Type = tagBool
				return result
				for _, r := range argPinned0 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "jit-warn-if-fallback",

		Fn: func(a ...Scmer) Scmer {
			value := a[0]
			compiled := value.GetTag() == tagJIT || (value.GetTag() == tagProc && value.Proc() != nil && value.Proc().Compiled != nil)
			if jitEnabled && !compiled {
				label := SerializeToString(value, &Globalenv)
				if len(a) > 1 {
					label = String(a[1])
				}
				fmt.Printf("warning: JIT fallback: %s\n", label)
			}
			return value
		},
		Type: &TypeDescriptor{Kind: "func", Description: "prints a diagnostic warning when an enabled JIT build kept a procedure interpreted and returns the procedure unchanged",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "procedure", Description: "procedure expected to be a native compilation candidate"},
				{Kind: "string", Label: "label", Description: "optional diagnostic label", Optional: true},
			},
			Return:         &TypeDescriptor{Kind: "any"},
			HasSideEffects: true,
			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["jit-warn-if-fallback"].Fn, args, result)
			},
			JITVirtualArgs: true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "jit-enabled?",

		Fn: func(_ ...Scmer) Scmer {
			return NewBool(jitEnabled)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "tells whether this binary was built with the patched Go JIT runtime",
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				argPinned0 := make([]Reg, 0, len(args)*2)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned0 = append(argPinned0, ai.Reg2)
						}
					}
				}
				d1 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d1.Loc == LocImm {
					ctx.EmitMakeBool(result, d1)
				} else {
					ctx.EmitMakeBool(result, d1)
					ctx.FreeReg(d1.Reg)
				}
				result.Type = tagBool
				return result
				for _, r := range argPinned0 {
					ctx.UnprotectReg(r)
				}
				return result
			},
		},
	})
}

// jitCompile compiles a Proc to a native function (tagFunc)
// Already compiled functions (tagFunc, tagFuncEnv) are passed through unchanged
func jitCompile(a ...Scmer) Scmer {
	return jitCompileMode(true, a...)
}

func jitCompileMode(recursiveLambdas bool, a ...Scmer) Scmer {
	return jitCompileModePublish(recursiveLambdas, true, a...)
}

func jitCompileModeDeferred(recursiveLambdas bool, a ...Scmer) Scmer {
	return jitCompileModePublish(recursiveLambdas, false, a...)
}

func jitCompileModePublish(recursiveLambdas bool, waitForPublication bool, a ...Scmer) Scmer {
	if len(a) != 1 {
		panic("jit: expects exactly 1 argument")
	}

	v := a[0]
	tag := v.GetTag()
	if !jitEnabled {
		switch tag {
		case tagJIT, tagFunc, tagFuncEnv, tagProc:
			return v
		default:
			panic(fmt.Sprintf("jit: cannot compile %v (tag %d)", v, tag))
		}
	}
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
		// Lambda/procedure — compile into a pool arena
		proc := v.Proc()
		if proc != nil && proc.Compiled != nil {
			return v
		}
		// Try increasing buffer sizes for overflow retry
		for _, codeCap := range [...]int{16 * 1024, 64 * 1024, 256 * 1024, 1024 * 1024} {
			ptr, arena, reservation := globalJITPool.Alloc(codeCap)
			buf := &execBuf{ptr: ptr, n: codeCap, arena: arena, reservation: reservation}
			codeLen, roots, overflow, transferInputArgs, hiddenArgs, autoImportSafe, needsStableArgs, coverage := jitCompileProcToExec(proc, buf, recursiveLambdas)
			if waitForPublication {
				arena.complete(reservation, buf.stackMaps)
			} else {
				arena.completeDeferred(reservation, buf.stackMaps)
			}
			if codeLen > 0 {
				code := (*[1 << 30]byte)(ptr)[:codeLen:codeLen]
				if JITLog {
					fmt.Printf("%X\n", code)
				}
				maybeDumpJITCode(ptr, code)
				fn2 := unsafe.Pointer(&struct{ *byte }{&code[0]})
				nativeFn := *(*func(...Scmer) Scmer)(unsafe.Pointer(&fn2))
				sourceProc := *proc
				sourceProc.Compiled = nil
				jep := &JITEntryPoint{
					Native:            nativeFn,
					TransferInputArgs: transferInputArgs,
					HiddenArgs:        hiddenArgs,
					CodePtr:           ptr,
					CodeLen:           codeLen,
					Arena:             arena,
					ConstRoots:        roots,
					Proc:              sourceProc,
					AutoImportSafe:    autoImportSafe && jitAutoImportSyntaxSafe(sourceProc.Body),
					RecursiveLambdas:  recursiveLambdas,
					NeedsStableArgs:   needsStableArgs,
					Coverage:          coverage,
				}
				runtime.SetFinalizer(jep, func(jep *JITEntryPoint) {
					if jep.Arena != nil && jep.CodePtr != nil {
						globalJITPool.Free(jep.CodePtr, jep.CodeLen)
					}
				})
				compiledProc := sourceProc
				compiledProc.Compiled = jep
				return NewProcStruct(compiledProc)
			}
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

// jitAutoImportSyntaxSafe rejects procedures which manufacture callbacks. A
// compiled top-level procedure may otherwise use every expression the emitter
// accepted; unsupported expressions already abort compilation atomically.
// Callback closure compilation remains opt-in until its generated slice paths
// carry the same complete lifetime contract as the enclosing entry point.
func jitAutoImportSyntaxSafe(expr Scmer) bool {
	for expr.IsSourceInfo() {
		expr = expr.SourceInfo().value
	}
	if !expr.IsSlice() {
		return true
	}
	items := expr.Slice()
	if len(items) == 0 {
		return true
	}
	if items[0].IsSymbol() {
		switch items[0].String() {
		case "quote":
			return true
		case "lambda", "optimizer_proc_return":
			return false
		}
	}
	for _, item := range items {
		if !jitAutoImportSyntaxSafe(item) {
			return false
		}
	}
	return true
}

// execBuf is a small wrapper for writable memory (arena-backed or standalone)
type execBuf struct {
	ptr         unsafe.Pointer
	n           int       // size
	arena       *jitArena // owning arena (nil for standalone buffers)
	reservation *jitCodeReservation
	stackMaps   []jitStackMap
}

func maybeDumpJITCode(base unsafe.Pointer, code []byte) {
	dir := os.Getenv("MEMCP_JIT_DUMP_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Printf("jitdump: mkdir failed: %v\n", err)
		return
	}
	name := fmt.Sprintf("jit_%016x_len_%d.bin", uintptr(base), len(code))
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, code, 0o644); err != nil {
		fmt.Printf("jitdump: write failed: %v\n", err)
		return
	}
	fmt.Printf("jitdump: %s\n", path)
}
