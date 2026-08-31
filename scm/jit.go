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

// JITCondition describes the result of the most recently emitted comparison.
// It is deliberately independent of any architecture's condition-code
// encoding; machine emitters translate it when they write a branch or a
// boolean result.
type JITCondition uint8

const (
	CondEqual JITCondition = iota
	CondNotEqual
	CondSignedLess
	CondSignedGreaterOrEqual
	CondSignedLessOrEqual
	CondSignedGreater
	CondUnsignedBelow
	CondUnsignedAboveOrEqual
	CondUnsignedBelowOrEqual
	CondUnsignedAbove
)

// Short condition names keep generated emitters source-compatible. New
// lowering code should use the descriptive, architecture-neutral names above.
const (
	CcE  = CondEqual
	CcNE = CondNotEqual
	CcL  = CondSignedLess
	CcGE = CondSignedGreaterOrEqual
	CcLE = CondSignedLessOrEqual
	CcG  = CondSignedGreater
	CcB  = CondUnsignedBelow
	CcAE = CondUnsignedAboveOrEqual
	CcBE = CondUnsignedBelowOrEqual
	CcA  = CondUnsignedAbove
)

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
	// Architecture register roles let common lowering describe placement without
	// depending on one instruction set's register names.
	StackReg     Reg
	FrameReg     Reg
	ScratchReg   Reg
	ResultPtrReg Reg
	ResultAuxReg Reg
	LastIntReg   Reg
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
	TransferInputArgs     bool
	HiddenArgs            []JITHiddenArg
	RuntimeEnv            Scmer
	AutoImportSafe        bool
	RecursiveLambdas      bool
	ActiveBuiltinEmitters map[*Declaration]uint16
	NeedsStableArgs       bool
	SelfSymbols           map[Symbol]struct{}
	SelfLoopLabel         uint8
	HasSelfLoop           bool
	SelfParamCount        int
	RegOwners             [16]*JITValueDesc // register → owner descriptor (nil = untracked)
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
	for r := Reg(0); r <= RegR15; r++ {
		if (ctx.AllRegs&(1<<uint(r))) == 0 || (ctx.FreeRegs&(1<<uint(r))) != 0 {
			continue
		}
		owner := ctx.RegOwners[r]
		off := ctx.AllocSpill(8)
		ctx.EmitStoreRegMem(r, RegRBP, off)
		if owner != nil && !owner.NoHeapPointer &&
			((owner.Loc == LocRegPair && owner.Reg == r) ||
				(owner.Loc == LocRegTriple && owner.Reg == r)) {
			ctx.setStackPointer(jitStackRootFrameBP, off, true)
		}
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

// SyncDesc updates an aliased descriptor after allocator spills without
// materializing stack-backed values into registers.
func (ctx *JITContext) SyncDesc(desc *JITValueDesc) {
	ctx.syncDescSpill(desc)
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

func JITBuildScmerCallback(callback Scmer) func(Scmer) {
	return func(value Scmer) { Apply(callback, value) }
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
		if head, ok := scmerSymbol(list[0]); ok {
			switch string(head) {
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
	if head, ok := scmerSymbol(list[0]); ok {
		switch string(head) {
		case "quote":
			return
		case "outer":
			if len(list) == 3 {
				arg := list[2]
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
	proc.OptimizerMeta = &ProcOptimizerMeta{Return: metadata.Return, HasReturn: metadata.HasReturn, Sequence: metadata.Sequence}
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
		go func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					errs <- recovered
				} else {
					errs <- nil
				}
			}()
			jitCallSpecialThunk(thunk)
		}()
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

func JITStringEqual(a, b string) bool { return a == b }

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

func jitPanicString(message string) {
	panic(message)
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
	// Protected registers are explicitly live across the current nested emitter
	// even when a borrowed descriptor has no allocator owner (or its owner was
	// moved to a spill slot). Calls must preserve that contract as well as normal
	// allocator ownership.
	allocatedMask := (ctx.AllRegs &^ ctx.FreeRegs) | (ctx.ProtectedRegs & ctx.AllRegs)
	for r := Reg(0); r <= RegR15; r++ {
		if ctx.RegOwners[r] != nil && (ctx.FreeRegs&(1<<uint(r))) != 0 && (ctx.ProtectedRegs&(1<<uint(r))) == 0 {
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
	return ctx.emitGoCall(funcAddr, argWords, numResultWords, resultsBuf, resultTargets, nil)
}

func (ctx *JITContext) EmitGoCallToStack(funcAddr uint64, argWords []goCallArgWord, resultStackOffs []int32) {
	var resultsBuf [16]Reg
	ctx.emitGoCall(funcAddr, argWords, len(resultStackOffs), &resultsBuf, nil, resultStackOffs)
}

func (ctx *JITContext) emitGoCall(funcAddr uint64, argWords []goCallArgWord, numResultWords int, resultsBuf *[16]Reg, resultTargets []Reg, resultStackOffs []int32) []Reg {
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
		if resultStackOffs != nil {
			for i, off := range resultStackOffs {
				ctx.EmitStoreRegMem(GoABIIntRegs[i], ctx.StackReg, off)
			}
			return nil
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
	if resultStackOffs != nil {
		for i, off := range resultStackOffs {
			ctx.EmitMovRegMem(RegR11, RegRSP, int32(i*8))
			ctx.EmitStoreRegMem(RegR11, ctx.StackReg, off+int32(resultBytes))
		}
		ctx.EmitReleaseStackBytes(int32(resultBytes))
		if ctx.SliceBaseTracksRSP && ctx.SliceBase != RegRSP {
			ctx.emitMovRegReg(ctx.SliceBase, RegRSP)
		}
		return nil
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
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["jit"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
				d1 := d0
				_ = d1
				ctx.StabilizeDescForControlFlow(&d1)
				bbpos_1_0 := int32(-1)
				_ = bbpos_1_0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d1)
				d2 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
				d3 := d1
				_ = d3
				ctx.StabilizeDescForControlFlow(&d3)
				d4 := d2
				_ = d4
				ctx.StabilizeDescForControlFlow(&d4)
				phiBase5 := ctx.AllocStack(int32(32))
				d6 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				_ = d6
				d7 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase5) + int32(16)}
				_ = d7
				lbl0 := ctx.ReserveLabel()
				bbpos_2_0 := int32(-1)
				_ = bbpos_2_0
				bbpos_2_1 := int32(-1)
				_ = bbpos_2_1
				bbpos_2_2 := int32(-1)
				_ = bbpos_2_2
				bbpos_2_3 := int32(-1)
				_ = bbpos_2_3
				bbpos_2_4 := int32(-1)
				_ = bbpos_2_4
				bbpos_2_5 := int32(-1)
				_ = bbpos_2_5
				bbpos_2_6 := int32(-1)
				_ = bbpos_2_6
				bbpos_2_7 := int32(-1)
				_ = bbpos_2_7
				bbpos_2_8 := int32(-1)
				_ = bbpos_2_8
				bbpos_2_9 := int32(-1)
				_ = bbpos_2_9
				bbpos_2_10 := int32(-1)
				_ = bbpos_2_10
				bbpos_2_11 := int32(-1)
				_ = bbpos_2_11
				bbpos_2_12 := int32(-1)
				_ = bbpos_2_12
				bbpos_2_13 := int32(-1)
				_ = bbpos_2_13
				bbpos_2_14 := int32(-1)
				_ = bbpos_2_14
				bbpos_2_15 := int32(-1)
				_ = bbpos_2_15
				bbpos_2_16 := int32(-1)
				_ = bbpos_2_16
				bbpos_2_17 := int32(-1)
				_ = bbpos_2_17
				bbpos_2_18 := int32(-1)
				_ = bbpos_2_18
				bbpos_2_19 := int32(-1)
				_ = bbpos_2_19
				bbpos_2_20 := int32(-1)
				_ = bbpos_2_20
				bbpos_2_21 := int32(-1)
				_ = bbpos_2_21
				bbpos_2_22 := int32(-1)
				_ = bbpos_2_22
				bbpos_2_23 := int32(-1)
				_ = bbpos_2_23
				bbpos_2_24 := int32(-1)
				_ = bbpos_2_24
				bbpos_2_25 := int32(-1)
				_ = bbpos_2_25
				bbpos_2_26 := int32(-1)
				_ = bbpos_2_26
				bbpos_2_27 := int32(-1)
				_ = bbpos_2_27
				bbpos_2_28 := int32(-1)
				_ = bbpos_2_28
				bbpos_2_29 := int32(-1)
				_ = bbpos_2_29
				bbpos_2_30 := int32(-1)
				_ = bbpos_2_30
				bbpos_2_31 := int32(-1)
				_ = bbpos_2_31
				bbpos_2_32 := int32(-1)
				_ = bbpos_2_32
				bbpos_2_33 := int32(-1)
				_ = bbpos_2_33
				bbpos_2_34 := int32(-1)
				_ = bbpos_2_34
				bbpos_2_35 := int32(-1)
				_ = bbpos_2_35
				bbpos_2_36 := int32(-1)
				_ = bbpos_2_36
				bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d7 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d8 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d8)
				var d9 JITValueDesc
				if d8.Loc == LocImm {
					d9 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d8.Imm.Int() != 1)}
				} else {
					r0 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d8.Reg, 1)
					ctx.EmitSetcc(r0, CondNotEqual)
					d9 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
					ctx.BindReg(r0, &d9)
				}
				ctx.FreeDesc(&d8)
				ctx.ReclaimUntrackedRegs()
				d10 := d9
				ctx.EnsureDesc(&d10)
				if d10.Loc != LocImm && d10.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl1 := ctx.ReserveLabel()
				lbl2 := ctx.ReserveLabel()
				lbl3 := ctx.ReserveLabel()
				lbl4 := ctx.ReserveLabel()
				if d10.Loc == LocImm {
					if d10.Imm.Bool() {
						ctx.MarkLabel(lbl3)
						ctx.EmitJmp(lbl1)
					} else {
						ctx.MarkLabel(lbl4)
						ctx.EmitJmp(lbl2)
					}
				} else {
					ctx.EmitCmpRegImm32(d10.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl3)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl3)
					ctx.EmitJmp(lbl1)
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
				}
				ctx.FreeDesc(&d9)
				bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl2)
				ctx.ResolveFixups()
				d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d7 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d11 := args[0]
				d11.ID = 0
				ctx.StabilizeDescForControlFlow(&d11)
				ctx.ReclaimUntrackedRegs()
				d12 := ctx.EmitGetTagDesc(&d11, JITValueDesc{Loc: LocAny})
				ctx.StabilizeDescForControlFlow(&d12)
				ctx.ReclaimUntrackedRegs()
				bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d7 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d12)
				var d13 JITValueDesc
				if d12.Loc == LocImm {
					d13 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d12.Imm.Int()) == uint64(0xb))}
				} else {
					r1 := ctx.AllocRegExcept(d12.Reg)
					ctx.EmitCmpRegImm32(d12.Reg, 11)
					ctx.EmitSetcc(r1, CondEqual)
					d13 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
					ctx.BindReg(r1, &d13)
				}
				ctx.ReclaimUntrackedRegs()
				d14 := d13
				ctx.EnsureDesc(&d14)
				if d14.Loc != LocImm && d14.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl5 := ctx.ReserveLabel()
				lbl6 := ctx.ReserveLabel()
				lbl7 := ctx.ReserveLabel()
				lbl8 := ctx.ReserveLabel()
				if d14.Loc == LocImm {
					if d14.Imm.Bool() {
						ctx.MarkLabel(lbl7)
						ctx.EmitJmp(lbl5)
					} else {
						ctx.MarkLabel(lbl8)
						ctx.EmitJmp(lbl6)
					}
				} else {
					ctx.EmitCmpRegImm32(d14.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl6)
				}
				ctx.FreeDesc(&d13)
				bbpos_2_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl6)
				ctx.ResolveFixups()
				d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d7 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d12)
				var d15 JITValueDesc
				if d12.Loc == LocImm {
					d15 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d12.Imm.Int()) == uint64(0x8))}
				} else {
					r2 := ctx.AllocRegExcept(d12.Reg)
					ctx.EmitCmpRegImm32(d12.Reg, 8)
					ctx.EmitSetcc(r2, CondEqual)
					d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
					ctx.BindReg(r2, &d15)
				}
				ctx.ReclaimUntrackedRegs()
				d16 := d15
				ctx.EnsureDesc(&d16)
				if d16.Loc != LocImm && d16.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl9 := ctx.ReserveLabel()
				lbl10 := ctx.ReserveLabel()
				lbl11 := ctx.ReserveLabel()
				if d16.Loc == LocImm {
					if d16.Imm.Bool() {
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl5)
					} else {
						ctx.MarkLabel(lbl11)
						ctx.EmitJmp(lbl9)
					}
				} else {
					ctx.EmitCmpRegImm32(d16.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl10)
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl9)
				}
				ctx.FreeDesc(&d15)
				bbpos_2_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl9)
				ctx.ResolveFixups()
				d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d7 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d12)
				var d17 JITValueDesc
				if d12.Loc == LocImm {
					d17 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d12.Imm.Int()) == uint64(0x9))}
				} else {
					r3 := ctx.AllocRegExcept(d12.Reg)
					ctx.EmitCmpRegImm32(d12.Reg, 9)
					ctx.EmitSetcc(r3, CondEqual)
					d17 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
					ctx.BindReg(r3, &d17)
				}
				ctx.ReclaimUntrackedRegs()
				d18 := d17
				ctx.EnsureDesc(&d18)
				if d18.Loc != LocImm && d18.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl12 := ctx.ReserveLabel()
				lbl13 := ctx.ReserveLabel()
				lbl14 := ctx.ReserveLabel()
				if d18.Loc == LocImm {
					if d18.Imm.Bool() {
						ctx.MarkLabel(lbl13)
						ctx.EmitJmp(lbl5)
					} else {
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl12)
					}
				} else {
					ctx.EmitCmpRegImm32(d18.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl13)
					ctx.EmitJmp(lbl14)
					ctx.MarkLabel(lbl13)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl14)
					ctx.EmitJmp(lbl12)
				}
				ctx.FreeDesc(&d17)
				bbpos_2_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl12)
				ctx.ResolveFixups()
				d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d7 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d12)
				var d19 JITValueDesc
				if d12.Loc == LocImm {
					d19 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d12.Imm.Int()) == uint64(0xa))}
				} else {
					r4 := ctx.AllocRegExcept(d12.Reg)
					ctx.EmitCmpRegImm32(d12.Reg, 10)
					ctx.EmitSetcc(r4, CondEqual)
					d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
					ctx.BindReg(r4, &d19)
				}
				ctx.ReclaimUntrackedRegs()
				d20 := d19
				ctx.EnsureDesc(&d20)
				if d20.Loc != LocImm && d20.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl15 := ctx.ReserveLabel()
				lbl16 := ctx.ReserveLabel()
				lbl17 := ctx.ReserveLabel()
				if d20.Loc == LocImm {
					if d20.Imm.Bool() {
						ctx.MarkLabel(lbl16)
						ctx.EmitJmp(lbl5)
					} else {
						ctx.MarkLabel(lbl17)
						ctx.EmitJmp(lbl15)
					}
				} else {
					ctx.EmitCmpRegImm32(d20.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl16)
					ctx.EmitJmp(lbl17)
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl17)
					ctx.EmitJmp(lbl15)
				}
				ctx.FreeDesc(&d19)
				bbpos_2_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl15)
				ctx.ResolveFixups()
				d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d7 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
				bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl1)
				ctx.ResolveFixups()
				d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d7 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
				bbpos_2_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl5)
				ctx.ResolveFixups()
				d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase5) + int32(0)}
				d7 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase5) + int32(16)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				r5 := ctx.AllocReg()
				r6 := ctx.AllocRegExcept(r5)
				d21 := JITValueDesc{Loc: LocRegPair, Reg: r5, Reg2: r6}
				ctx.BindReg(r5, &d21)
				ctx.BindReg(r6, &d21)
				ctx.EmitMovPairToResult(&d11, &d21)
				ctx.EmitJmp(lbl0)
				ctx.MarkLabel(lbl0)
				d22 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r5, Reg2: r6}
				ctx.BindReg(r5, &d22)
				ctx.BindReg(r6, &d22)
				ctx.BindReg(r5, &d22)
				ctx.BindReg(r6, &d22)
				ctx.FreeDesc(&d1)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d22)
				if d22.Loc == LocImm {
					if result.Loc == LocAny {
						return d22
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.EnsureDesc(&d22)
				if d22.Loc == LocRegPair {
					ctx.EmitMovPairToResult(&d22, &result)
					result.Type = d22.Type
				} else {
					switch d22.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d22)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d22)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d22)
						result.Type = tagFloat
					case tagNil:
						ctx.EmitMakeNil(result)
						result.Type = tagNil
					default:
						panic("jit: single-block scalar return with unknown type")
					}
				}
				return result
				return result
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
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["jit?"].Fn, args, result)
				}
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d8 JITValueDesc
				_ = d8
				var d12 JITValueDesc
				_ = d12
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
				var d28 JITValueDesc
				_ = d28
				var d31 JITValueDesc
				_ = d31
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
				var d50 JITValueDesc
				_ = d50
				var d51 JITValueDesc
				_ = d51
				var d52 JITValueDesc
				_ = d52
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
				var d56 JITValueDesc
				_ = d56
				var d57 JITValueDesc
				_ = d57
				var d58 JITValueDesc
				_ = d58
				var d59 JITValueDesc
				_ = d59
				var d60 JITValueDesc
				_ = d60
				var d61 JITValueDesc
				_ = d61
				var d63 JITValueDesc
				_ = d63
				var d64 JITValueDesc
				_ = d64
				var d66 JITValueDesc
				_ = d66
				var d67 JITValueDesc
				_ = d67
				var d68 JITValueDesc
				_ = d68
				var d69 JITValueDesc
				_ = d69
				var d70 JITValueDesc
				_ = d70
				var d71 JITValueDesc
				_ = d71
				var d72 JITValueDesc
				_ = d72
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d78 JITValueDesc
				_ = d78
				var d79 JITValueDesc
				_ = d79
				var d82 JITValueDesc
				_ = d82
				var d85 JITValueDesc
				_ = d85
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
				d1 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
				var bbs [6]BBDescriptor
				bbs[2].PhiBase = int32(phiBase0) + int32(0)
				bbs[2].PhiCount = uint16(1)
				bbs[4].PhiBase = int32(phiBase0) + int32(16)
				bbs[4].PhiCount = uint16(1)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[0].VisitCount >= 0 {
							ps.General = true
							return bbs[0].RenderPS(ps)
						}
					}
					bbs[0].VisitCount++
					if ps.General {
						if bbs[0].Rendered {
							ctx.EmitJmp(lbl1)
							return result
						}
						bbs[0].Rendered = true
						bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_0 = bbs[0].Address
						ctx.MarkLabel(lbl1)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					ctx.ReclaimUntrackedRegs()
					d3 = args[0]
					d3.ID = 0
					d4 = ctx.EmitGetTagDesc(&d3, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d3)
					ctx.EnsureDesc(&d4)
					var d5 JITValueDesc
					if d4.Loc == LocImm {
						d5 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d4.Imm.Int()) == uint64(0xb))}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d4.Reg, 11)
						ctx.EmitSetcc(r0, CondEqual)
						d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d5)
					}
					ctx.FreeDesc(&d4)
					d6 = d5
					ctx.EnsureDesc(&d6)
					if d6.Loc != LocImm && d6.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d6.Loc == LocImm {
						if d6.Imm.Bool() {
							if ps.General {
								ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[2].PhiBase)+int32(0))
							}
							ps7 := PhiState{General: ps.General}
							ps7.OverlayValues = make([]JITValueDesc, 7)
							ps7.OverlayValues[1] = d1
							ps7.OverlayValues[2] = d2
							ps7.OverlayValues[3] = d3
							ps7.OverlayValues[4] = d4
							ps7.OverlayValues[5] = d5
							ps7.OverlayValues[6] = d6
							ps7.PhiValues = make([]JITValueDesc, 1)
							d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
							ps7.PhiValues[0] = d8
							return bbs[2].RenderPS(ps7)
						}
						if ps.General {
						}
						ps9 := PhiState{General: ps.General}
						ps9.OverlayValues = make([]JITValueDesc, 9)
						ps9.OverlayValues[1] = d1
						ps9.OverlayValues[2] = d2
						ps9.OverlayValues[3] = d3
						ps9.OverlayValues[4] = d4
						ps9.OverlayValues[5] = d5
						ps9.OverlayValues[6] = d6
						ps9.OverlayValues[8] = d8
						return bbs[1].RenderPS(ps9)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d6.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl7)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl2)
					ps10 := PhiState{General: true}
					ps10.OverlayValues = make([]JITValueDesc, 9)
					ps10.OverlayValues[1] = d1
					ps10.OverlayValues[2] = d2
					ps10.OverlayValues[3] = d3
					ps10.OverlayValues[4] = d4
					ps10.OverlayValues[5] = d5
					ps10.OverlayValues[6] = d6
					ps10.OverlayValues[8] = d8
					ps10.PhiValues = make([]JITValueDesc, 1)
					d12 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
					ps10.PhiValues[0] = d12
					ps11 := PhiState{General: true}
					ps11.OverlayValues = make([]JITValueDesc, 13)
					ps11.OverlayValues[1] = d1
					ps11.OverlayValues[2] = d2
					ps11.OverlayValues[3] = d3
					ps11.OverlayValues[4] = d4
					ps11.OverlayValues[5] = d5
					ps11.OverlayValues[6] = d6
					ps11.OverlayValues[8] = d8
					ps11.OverlayValues[12] = d12
					snap13 := d1
					snap14 := d2
					snap15 := d3
					snap16 := d4
					snap17 := d5
					snap18 := d6
					snap19 := d8
					snap20 := d12
					alloc21 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps10)
					}
					ctx.RestoreAllocState(alloc21)
					d1 = snap13
					d2 = snap14
					d3 = snap15
					d4 = snap16
					d5 = snap17
					d6 = snap18
					d8 = snap19
					d12 = snap20
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps11)
					}
					return result
					ctx.FreeDesc(&d5)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[1].VisitCount >= 0 {
							ps.General = true
							return bbs[1].RenderPS(ps)
						}
					}
					bbs[1].VisitCount++
					if ps.General {
						if bbs[1].Rendered {
							ctx.EmitJmp(lbl2)
							return result
						}
						bbs[1].Rendered = true
						bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_1 = bbs[1].Address
						ctx.MarkLabel(lbl2)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					ctx.ReclaimUntrackedRegs()
					d22 = args[0]
					d22.ID = 0
					d23 = ctx.EmitGetTagDesc(&d22, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d22)
					ctx.EnsureDesc(&d23)
					var d24 JITValueDesc
					if d23.Loc == LocImm {
						d24 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d23.Imm.Int()) == uint64(0xa))}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d23.Reg, 10)
						ctx.EmitSetcc(r1, CondEqual)
						d24 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d24)
					}
					ctx.FreeDesc(&d23)
					d25 = d24
					ctx.EnsureDesc(&d25)
					if d25.Loc != LocImm && d25.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d25.Loc == LocImm {
						if d25.Imm.Bool() {
							if ps.General {
							}
							ps26 := PhiState{General: ps.General}
							ps26.OverlayValues = make([]JITValueDesc, 26)
							ps26.OverlayValues[1] = d1
							ps26.OverlayValues[2] = d2
							ps26.OverlayValues[3] = d3
							ps26.OverlayValues[4] = d4
							ps26.OverlayValues[5] = d5
							ps26.OverlayValues[6] = d6
							ps26.OverlayValues[8] = d8
							ps26.OverlayValues[12] = d12
							ps26.OverlayValues[22] = d22
							ps26.OverlayValues[23] = d23
							ps26.OverlayValues[24] = d24
							ps26.OverlayValues[25] = d25
							return bbs[5].RenderPS(ps26)
						}
						if ps.General {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
						}
						ps27 := PhiState{General: ps.General}
						ps27.OverlayValues = make([]JITValueDesc, 26)
						ps27.OverlayValues[1] = d1
						ps27.OverlayValues[2] = d2
						ps27.OverlayValues[3] = d3
						ps27.OverlayValues[4] = d4
						ps27.OverlayValues[5] = d5
						ps27.OverlayValues[6] = d6
						ps27.OverlayValues[8] = d8
						ps27.OverlayValues[12] = d12
						ps27.OverlayValues[22] = d22
						ps27.OverlayValues[23] = d23
						ps27.OverlayValues[24] = d24
						ps27.OverlayValues[25] = d25
						ps27.PhiValues = make([]JITValueDesc, 1)
						d28 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
						ps27.PhiValues[0] = d28
						return bbs[4].RenderPS(ps27)
					}
					if !ps.General {
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d25.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl10)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					ctx.EmitJmp(lbl5)
					ps29 := PhiState{General: true}
					ps29.OverlayValues = make([]JITValueDesc, 29)
					ps29.OverlayValues[1] = d1
					ps29.OverlayValues[2] = d2
					ps29.OverlayValues[3] = d3
					ps29.OverlayValues[4] = d4
					ps29.OverlayValues[5] = d5
					ps29.OverlayValues[6] = d6
					ps29.OverlayValues[8] = d8
					ps29.OverlayValues[12] = d12
					ps29.OverlayValues[22] = d22
					ps29.OverlayValues[23] = d23
					ps29.OverlayValues[24] = d24
					ps29.OverlayValues[25] = d25
					ps29.OverlayValues[28] = d28
					ps30 := PhiState{General: true}
					ps30.OverlayValues = make([]JITValueDesc, 29)
					ps30.OverlayValues[1] = d1
					ps30.OverlayValues[2] = d2
					ps30.OverlayValues[3] = d3
					ps30.OverlayValues[4] = d4
					ps30.OverlayValues[5] = d5
					ps30.OverlayValues[6] = d6
					ps30.OverlayValues[8] = d8
					ps30.OverlayValues[12] = d12
					ps30.OverlayValues[22] = d22
					ps30.OverlayValues[23] = d23
					ps30.OverlayValues[24] = d24
					ps30.OverlayValues[25] = d25
					ps30.OverlayValues[28] = d28
					ps30.PhiValues = make([]JITValueDesc, 1)
					d31 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ps30.PhiValues[0] = d31
					snap32 := d1
					snap33 := d2
					snap34 := d3
					snap35 := d4
					snap36 := d5
					snap37 := d6
					snap38 := d8
					snap39 := d12
					snap40 := d22
					snap41 := d23
					snap42 := d24
					snap43 := d25
					snap44 := d28
					snap45 := d31
					alloc46 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps30)
					}
					ctx.RestoreAllocState(alloc46)
					d1 = snap32
					d2 = snap33
					d3 = snap34
					d4 = snap35
					d5 = snap36
					d6 = snap37
					d8 = snap38
					d12 = snap39
					d22 = snap40
					d23 = snap41
					d24 = snap42
					d25 = snap43
					d28 = snap44
					d31 = snap45
					if !bbs[5].Rendered {
						return bbs[5].RenderPS(ps29)
					}
					return result
					ctx.FreeDesc(&d24)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d47 := ps.PhiValues[0]
							ctx.EnsureDesc(&d47)
							ctx.EmitStoreToStack(d47, int32(bbs[2].PhiBase)+int32(0))
						}
						if bbs[2].VisitCount >= 0 {
							ps.General = true
							return bbs[2].RenderPS(ps)
						}
					}
					bbs[2].VisitCount++
					if ps.General {
						if bbs[2].Rendered {
							ctx.EmitJmp(lbl3)
							return result
						}
						bbs[2].Rendered = true
						bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_2 = bbs[2].Address
						ctx.MarkLabel(lbl3)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						ctx.EmitMakeBool(result, d1)
					} else {
						ctx.EmitMovToReg(result.Reg2, d1)
						d48 := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d48)
						if d1.Loc == LocReg && d1.Reg != result.Reg2 {
							ctx.FreeReg(d1.Reg)
						}
					}
					result.Type = tagBool
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[3].VisitCount >= 0 {
							ps.General = true
							return bbs[3].RenderPS(ps)
						}
					}
					bbs[3].VisitCount++
					if ps.General {
						if bbs[3].Rendered {
							ctx.EmitJmp(lbl4)
							return result
						}
						bbs[3].Rendered = true
						bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_3 = bbs[3].Address
						ctx.MarkLabel(lbl4)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					ctx.ReclaimUntrackedRegs()
					d49 = args[0]
					d49.ID = 0
					ctx.EnsureDesc(&d49)
					lbl11 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r2 := ctx.AllocReg()
					r3 := ctx.AllocRegExcept(r2)
					ctx.EmitMovRegImm64(r2, 0)
					ctx.EmitMovRegImm64(r3, 0)
					d50 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r2, Reg2: r3}
					ctx.BindReg(r2, &d50)
					ctx.BindReg(r3, &d50)
					ctx.StabilizeDescForControlFlow(&d50)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d49)
					ctx.ReclaimUntrackedRegs()
					d51 = d49
					_ = d51
					ctx.ReclaimUntrackedRegs()
					d52 = ctx.EmitGetTagDesc(&d51, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d51)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d52)
					var d53 JITValueDesc
					if d52.Loc == LocImm {
						d53 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d52.Imm.Int()) != uint64(0xa))}
					} else {
						r4 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d52.Reg, 10)
						ctx.EmitSetcc(r4, CondNotEqual)
						d53 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d53)
					}
					ctx.FreeDesc(&d52)
					ctx.ReclaimUntrackedRegs()
					d54 = d53
					ctx.EnsureDesc(&d54)
					if d54.Loc != LocImm && d54.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					if d54.Loc == LocImm {
						if d54.Imm.Bool() {
							ctx.MarkLabel(lbl14)
							ctx.EmitJmp(lbl12)
						} else {
							ctx.MarkLabel(lbl15)
							ctx.EmitJmp(lbl13)
						}
					} else {
						ctx.EmitCmpRegImm32(d54.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl14)
						ctx.EmitJmp(lbl15)
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl12)
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl13)
					}
					ctx.FreeDesc(&d53)
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d55 = args[0]
					d55.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d56 JITValueDesc
					ctx.EnsureDesc(&d55)
					if d55.Loc == LocImm {
						ptrWord, _ := d55.Imm.RawWords()
						d56 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
					} else {
						if d55.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r5 := ctx.AllocReg()
						ctx.EmitMovRegReg(r5, d55.Reg)
						d56 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d56)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d56)
					ctx.EnsureDesc(&d56)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d56)
					ctx.EnsureDesc(&d56)
					ctx.FreeDesc(&d56)
					ctx.ReclaimUntrackedRegs()
					r6 := ctx.AllocReg()
					ctx.EnsureDesc(&d56)
					ctx.EnsureDesc(&d56)
					if d56.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r6, d56)
					}
					ctx.EmitJmp(lbl11)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
					ctx.MarkLabel(lbl11)
					d59 = JITValueDesc{Loc: LocReg, Reg: r6}
					ctx.BindReg(r6, &d59)
					ctx.BindReg(r6, &d59)
					ctx.FreeDesc(&d49)
					var d60 JITValueDesc
					ctx.EnsureDesc(&d59)
					if d59.Loc == LocImm {
						fieldAddr := uintptr(d59.Imm.Int()) + 56
						r7 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r7, fieldAddr)
						d60 = JITValueDesc{Loc: LocReg, Reg: r7}
						ctx.BindReg(r7, &d60)
					} else {
						off := int32(56)
						baseReg := d59.Reg
						r8 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r8, baseReg, off)
						d60 = JITValueDesc{Loc: LocReg, Reg: r8}
						ctx.BindReg(r8, &d60)
					}
					ctx.FreeDesc(&d59)
					ctx.EnsureDesc(&d60)
					var d61 JITValueDesc
					if d60.Loc == LocImm {
						d61 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d60.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d60)
						if d60.Loc != LocReg && d60.Loc != LocRegPair && d60.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r9 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d60.Reg, 0)
						ctx.EmitSetcc(r9, CondNotEqual)
						d61 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
						ctx.BindReg(r9, &d61)
					}
					ctx.EnsureDesc(&d61)
					ctx.EmitStoreToStack(d61, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d61)
					ctx.FreeDesc(&d60)
					if ps.General {
					}
					ps62 := PhiState{General: ps.General}
					ps62.OverlayValues = make([]JITValueDesc, 62)
					ps62.OverlayValues[1] = d1
					ps62.OverlayValues[2] = d2
					ps62.OverlayValues[3] = d3
					ps62.OverlayValues[4] = d4
					ps62.OverlayValues[5] = d5
					ps62.OverlayValues[6] = d6
					ps62.OverlayValues[8] = d8
					ps62.OverlayValues[12] = d12
					ps62.OverlayValues[22] = d22
					ps62.OverlayValues[23] = d23
					ps62.OverlayValues[24] = d24
					ps62.OverlayValues[25] = d25
					ps62.OverlayValues[28] = d28
					ps62.OverlayValues[31] = d31
					ps62.OverlayValues[47] = d47
					ps62.OverlayValues[48] = d48
					ps62.OverlayValues[49] = d49
					ps62.OverlayValues[50] = d50
					ps62.OverlayValues[51] = d51
					ps62.OverlayValues[52] = d52
					ps62.OverlayValues[53] = d53
					ps62.OverlayValues[54] = d54
					ps62.OverlayValues[55] = d55
					ps62.OverlayValues[56] = d56
					ps62.OverlayValues[57] = d57
					ps62.OverlayValues[58] = d58
					ps62.OverlayValues[59] = d59
					ps62.OverlayValues[60] = d60
					ps62.OverlayValues[61] = d61
					ps62.PhiValues = make([]JITValueDesc, 1)
					if ps62.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps62)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d63 := ps.PhiValues[0]
							ctx.EnsureDesc(&d63)
							ctx.EmitStoreToStack(d63, int32(bbs[4].PhiBase)+int32(0))
						}
						if bbs[4].VisitCount >= 0 {
							ps.General = true
							return bbs[4].RenderPS(ps)
						}
					}
					bbs[4].VisitCount++
					if ps.General {
						if bbs[4].Rendered {
							ctx.EmitJmp(lbl5)
							return result
						}
						bbs[4].Rendered = true
						bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_4 = bbs[4].Address
						ctx.MarkLabel(lbl5)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d2 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d2)
					if ps.General {
						ctx.SyncDesc(&d2)
						if d2.Loc == LocReg {
							ctx.ProtectReg(d2.Reg)
						} else if d2.Loc == LocRegPair {
							ctx.ProtectReg(d2.Reg)
							ctx.ProtectReg(d2.Reg2)
						}
						d64 = d2
						if d64.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d64)
						ctx.EmitStoreToStack(d64, int32(bbs[2].PhiBase)+int32(0))
						if d2.Loc == LocReg {
							ctx.UnprotectReg(d2.Reg)
						} else if d2.Loc == LocRegPair {
							ctx.UnprotectReg(d2.Reg)
							ctx.UnprotectReg(d2.Reg2)
						}
					}
					ps65 := PhiState{General: ps.General}
					ps65.OverlayValues = make([]JITValueDesc, 65)
					ps65.OverlayValues[1] = d1
					ps65.OverlayValues[2] = d2
					ps65.OverlayValues[3] = d3
					ps65.OverlayValues[4] = d4
					ps65.OverlayValues[5] = d5
					ps65.OverlayValues[6] = d6
					ps65.OverlayValues[8] = d8
					ps65.OverlayValues[12] = d12
					ps65.OverlayValues[22] = d22
					ps65.OverlayValues[23] = d23
					ps65.OverlayValues[24] = d24
					ps65.OverlayValues[25] = d25
					ps65.OverlayValues[28] = d28
					ps65.OverlayValues[31] = d31
					ps65.OverlayValues[47] = d47
					ps65.OverlayValues[48] = d48
					ps65.OverlayValues[49] = d49
					ps65.OverlayValues[50] = d50
					ps65.OverlayValues[51] = d51
					ps65.OverlayValues[52] = d52
					ps65.OverlayValues[53] = d53
					ps65.OverlayValues[54] = d54
					ps65.OverlayValues[55] = d55
					ps65.OverlayValues[56] = d56
					ps65.OverlayValues[57] = d57
					ps65.OverlayValues[58] = d58
					ps65.OverlayValues[59] = d59
					ps65.OverlayValues[60] = d60
					ps65.OverlayValues[61] = d61
					ps65.OverlayValues[63] = d63
					ps65.OverlayValues[64] = d64
					ps65.PhiValues = make([]JITValueDesc, 1)
					d66 = d2
					ps65.PhiValues[0] = d66
					if ps65.General && bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					return bbs[2].RenderPS(ps65)
					return result
				}
				bbs[5].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[5].VisitCount >= 0 {
							ps.General = true
							return bbs[5].RenderPS(ps)
						}
					}
					bbs[5].VisitCount++
					if ps.General {
						if bbs[5].Rendered {
							ctx.EmitJmp(lbl6)
							return result
						}
						bbs[5].Rendered = true
						bbs[5].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_5 = bbs[5].Address
						ctx.MarkLabel(lbl6)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					ctx.ReclaimUntrackedRegs()
					d67 = args[0]
					d67.ID = 0
					ctx.EnsureDesc(&d67)
					lbl16 := ctx.ReserveLabel()
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					bbpos_2_1 := int32(-1)
					_ = bbpos_2_1
					bbpos_2_2 := int32(-1)
					_ = bbpos_2_2
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r10 := ctx.AllocReg()
					r11 := ctx.AllocRegExcept(r10)
					ctx.EmitMovRegImm64(r10, 0)
					ctx.EmitMovRegImm64(r11, 0)
					d68 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r10, Reg2: r11}
					ctx.BindReg(r10, &d68)
					ctx.BindReg(r11, &d68)
					ctx.StabilizeDescForControlFlow(&d68)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d67)
					ctx.ReclaimUntrackedRegs()
					d69 = d67
					_ = d69
					ctx.ReclaimUntrackedRegs()
					d70 = ctx.EmitGetTagDesc(&d69, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d69)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					var d71 JITValueDesc
					if d70.Loc == LocImm {
						d71 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d70.Imm.Int()) != uint64(0xa))}
					} else {
						r12 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d70.Reg, 10)
						ctx.EmitSetcc(r12, CondNotEqual)
						d71 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r12}
						ctx.BindReg(r12, &d71)
					}
					ctx.FreeDesc(&d70)
					ctx.ReclaimUntrackedRegs()
					d72 = d71
					ctx.EnsureDesc(&d72)
					if d72.Loc != LocImm && d72.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl17 := ctx.ReserveLabel()
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					lbl20 := ctx.ReserveLabel()
					if d72.Loc == LocImm {
						if d72.Imm.Bool() {
							ctx.MarkLabel(lbl19)
							ctx.EmitJmp(lbl17)
						} else {
							ctx.MarkLabel(lbl20)
							ctx.EmitJmp(lbl18)
						}
					} else {
						ctx.EmitCmpRegImm32(d72.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl19)
						ctx.EmitJmp(lbl20)
						ctx.MarkLabel(lbl19)
						ctx.EmitJmp(lbl17)
						ctx.MarkLabel(lbl20)
						ctx.EmitJmp(lbl18)
					}
					ctx.FreeDesc(&d71)
					bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl18)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d73 = args[0]
					d73.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d74 JITValueDesc
					ctx.EnsureDesc(&d73)
					if d73.Loc == LocImm {
						ptrWord, _ := d73.Imm.RawWords()
						d74 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
					} else {
						if d73.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r13 := ctx.AllocReg()
						ctx.EmitMovRegReg(r13, d73.Reg)
						d74 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r13}
						ctx.BindReg(r13, &d74)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d74)
					ctx.EnsureDesc(&d74)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d74)
					ctx.EnsureDesc(&d74)
					ctx.FreeDesc(&d74)
					ctx.ReclaimUntrackedRegs()
					r14 := ctx.AllocReg()
					ctx.EnsureDesc(&d74)
					ctx.EnsureDesc(&d74)
					if d74.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r14, d74)
					}
					ctx.EmitJmp(lbl16)
					bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
					ctx.MarkLabel(lbl16)
					d77 = JITValueDesc{Loc: LocReg, Reg: r14}
					ctx.BindReg(r14, &d77)
					ctx.BindReg(r14, &d77)
					ctx.FreeDesc(&d67)
					ctx.EnsureDesc(&d77)
					var d78 JITValueDesc
					if d77.Loc == LocImm {
						d78 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d77.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d77)
						if d77.Loc != LocReg && d77.Loc != LocRegPair && d77.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r15 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d77.Reg, 0)
						ctx.EmitSetcc(r15, CondNotEqual)
						d78 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
						ctx.BindReg(r15, &d78)
					}
					ctx.FreeDesc(&d77)
					d79 = d78
					ctx.EnsureDesc(&d79)
					if d79.Loc != LocImm && d79.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d79.Loc == LocImm {
						if d79.Imm.Bool() {
							if ps.General {
							}
							ps80 := PhiState{General: ps.General}
							ps80.OverlayValues = make([]JITValueDesc, 80)
							ps80.OverlayValues[1] = d1
							ps80.OverlayValues[2] = d2
							ps80.OverlayValues[3] = d3
							ps80.OverlayValues[4] = d4
							ps80.OverlayValues[5] = d5
							ps80.OverlayValues[6] = d6
							ps80.OverlayValues[8] = d8
							ps80.OverlayValues[12] = d12
							ps80.OverlayValues[22] = d22
							ps80.OverlayValues[23] = d23
							ps80.OverlayValues[24] = d24
							ps80.OverlayValues[25] = d25
							ps80.OverlayValues[28] = d28
							ps80.OverlayValues[31] = d31
							ps80.OverlayValues[47] = d47
							ps80.OverlayValues[48] = d48
							ps80.OverlayValues[49] = d49
							ps80.OverlayValues[50] = d50
							ps80.OverlayValues[51] = d51
							ps80.OverlayValues[52] = d52
							ps80.OverlayValues[53] = d53
							ps80.OverlayValues[54] = d54
							ps80.OverlayValues[55] = d55
							ps80.OverlayValues[56] = d56
							ps80.OverlayValues[57] = d57
							ps80.OverlayValues[58] = d58
							ps80.OverlayValues[59] = d59
							ps80.OverlayValues[60] = d60
							ps80.OverlayValues[61] = d61
							ps80.OverlayValues[63] = d63
							ps80.OverlayValues[64] = d64
							ps80.OverlayValues[66] = d66
							ps80.OverlayValues[67] = d67
							ps80.OverlayValues[68] = d68
							ps80.OverlayValues[69] = d69
							ps80.OverlayValues[70] = d70
							ps80.OverlayValues[71] = d71
							ps80.OverlayValues[72] = d72
							ps80.OverlayValues[73] = d73
							ps80.OverlayValues[74] = d74
							ps80.OverlayValues[75] = d75
							ps80.OverlayValues[76] = d76
							ps80.OverlayValues[77] = d77
							ps80.OverlayValues[78] = d78
							ps80.OverlayValues[79] = d79
							return bbs[3].RenderPS(ps80)
						}
						if ps.General {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
						}
						ps81 := PhiState{General: ps.General}
						ps81.OverlayValues = make([]JITValueDesc, 80)
						ps81.OverlayValues[1] = d1
						ps81.OverlayValues[2] = d2
						ps81.OverlayValues[3] = d3
						ps81.OverlayValues[4] = d4
						ps81.OverlayValues[5] = d5
						ps81.OverlayValues[6] = d6
						ps81.OverlayValues[8] = d8
						ps81.OverlayValues[12] = d12
						ps81.OverlayValues[22] = d22
						ps81.OverlayValues[23] = d23
						ps81.OverlayValues[24] = d24
						ps81.OverlayValues[25] = d25
						ps81.OverlayValues[28] = d28
						ps81.OverlayValues[31] = d31
						ps81.OverlayValues[47] = d47
						ps81.OverlayValues[48] = d48
						ps81.OverlayValues[49] = d49
						ps81.OverlayValues[50] = d50
						ps81.OverlayValues[51] = d51
						ps81.OverlayValues[52] = d52
						ps81.OverlayValues[53] = d53
						ps81.OverlayValues[54] = d54
						ps81.OverlayValues[55] = d55
						ps81.OverlayValues[56] = d56
						ps81.OverlayValues[57] = d57
						ps81.OverlayValues[58] = d58
						ps81.OverlayValues[59] = d59
						ps81.OverlayValues[60] = d60
						ps81.OverlayValues[61] = d61
						ps81.OverlayValues[63] = d63
						ps81.OverlayValues[64] = d64
						ps81.OverlayValues[66] = d66
						ps81.OverlayValues[67] = d67
						ps81.OverlayValues[68] = d68
						ps81.OverlayValues[69] = d69
						ps81.OverlayValues[70] = d70
						ps81.OverlayValues[71] = d71
						ps81.OverlayValues[72] = d72
						ps81.OverlayValues[73] = d73
						ps81.OverlayValues[74] = d74
						ps81.OverlayValues[75] = d75
						ps81.OverlayValues[76] = d76
						ps81.OverlayValues[77] = d77
						ps81.OverlayValues[78] = d78
						ps81.OverlayValues[79] = d79
						ps81.PhiValues = make([]JITValueDesc, 1)
						d82 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
						ps81.PhiValues[0] = d82
						return bbs[4].RenderPS(ps81)
					}
					if !ps.General {
						ps.General = true
						return bbs[5].RenderPS(ps)
					}
					lbl21 := ctx.ReserveLabel()
					lbl22 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d79.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl21)
					ctx.EmitJmp(lbl22)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl22)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					ctx.EmitJmp(lbl5)
					ps83 := PhiState{General: true}
					ps83.OverlayValues = make([]JITValueDesc, 83)
					ps83.OverlayValues[1] = d1
					ps83.OverlayValues[2] = d2
					ps83.OverlayValues[3] = d3
					ps83.OverlayValues[4] = d4
					ps83.OverlayValues[5] = d5
					ps83.OverlayValues[6] = d6
					ps83.OverlayValues[8] = d8
					ps83.OverlayValues[12] = d12
					ps83.OverlayValues[22] = d22
					ps83.OverlayValues[23] = d23
					ps83.OverlayValues[24] = d24
					ps83.OverlayValues[25] = d25
					ps83.OverlayValues[28] = d28
					ps83.OverlayValues[31] = d31
					ps83.OverlayValues[47] = d47
					ps83.OverlayValues[48] = d48
					ps83.OverlayValues[49] = d49
					ps83.OverlayValues[50] = d50
					ps83.OverlayValues[51] = d51
					ps83.OverlayValues[52] = d52
					ps83.OverlayValues[53] = d53
					ps83.OverlayValues[54] = d54
					ps83.OverlayValues[55] = d55
					ps83.OverlayValues[56] = d56
					ps83.OverlayValues[57] = d57
					ps83.OverlayValues[58] = d58
					ps83.OverlayValues[59] = d59
					ps83.OverlayValues[60] = d60
					ps83.OverlayValues[61] = d61
					ps83.OverlayValues[63] = d63
					ps83.OverlayValues[64] = d64
					ps83.OverlayValues[66] = d66
					ps83.OverlayValues[67] = d67
					ps83.OverlayValues[68] = d68
					ps83.OverlayValues[69] = d69
					ps83.OverlayValues[70] = d70
					ps83.OverlayValues[71] = d71
					ps83.OverlayValues[72] = d72
					ps83.OverlayValues[73] = d73
					ps83.OverlayValues[74] = d74
					ps83.OverlayValues[75] = d75
					ps83.OverlayValues[76] = d76
					ps83.OverlayValues[77] = d77
					ps83.OverlayValues[78] = d78
					ps83.OverlayValues[79] = d79
					ps83.OverlayValues[82] = d82
					ps84 := PhiState{General: true}
					ps84.OverlayValues = make([]JITValueDesc, 83)
					ps84.OverlayValues[1] = d1
					ps84.OverlayValues[2] = d2
					ps84.OverlayValues[3] = d3
					ps84.OverlayValues[4] = d4
					ps84.OverlayValues[5] = d5
					ps84.OverlayValues[6] = d6
					ps84.OverlayValues[8] = d8
					ps84.OverlayValues[12] = d12
					ps84.OverlayValues[22] = d22
					ps84.OverlayValues[23] = d23
					ps84.OverlayValues[24] = d24
					ps84.OverlayValues[25] = d25
					ps84.OverlayValues[28] = d28
					ps84.OverlayValues[31] = d31
					ps84.OverlayValues[47] = d47
					ps84.OverlayValues[48] = d48
					ps84.OverlayValues[49] = d49
					ps84.OverlayValues[50] = d50
					ps84.OverlayValues[51] = d51
					ps84.OverlayValues[52] = d52
					ps84.OverlayValues[53] = d53
					ps84.OverlayValues[54] = d54
					ps84.OverlayValues[55] = d55
					ps84.OverlayValues[56] = d56
					ps84.OverlayValues[57] = d57
					ps84.OverlayValues[58] = d58
					ps84.OverlayValues[59] = d59
					ps84.OverlayValues[60] = d60
					ps84.OverlayValues[61] = d61
					ps84.OverlayValues[63] = d63
					ps84.OverlayValues[64] = d64
					ps84.OverlayValues[66] = d66
					ps84.OverlayValues[67] = d67
					ps84.OverlayValues[68] = d68
					ps84.OverlayValues[69] = d69
					ps84.OverlayValues[70] = d70
					ps84.OverlayValues[71] = d71
					ps84.OverlayValues[72] = d72
					ps84.OverlayValues[73] = d73
					ps84.OverlayValues[74] = d74
					ps84.OverlayValues[75] = d75
					ps84.OverlayValues[76] = d76
					ps84.OverlayValues[77] = d77
					ps84.OverlayValues[78] = d78
					ps84.OverlayValues[79] = d79
					ps84.OverlayValues[82] = d82
					ps84.PhiValues = make([]JITValueDesc, 1)
					d85 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ps84.PhiValues[0] = d85
					snap86 := d1
					snap87 := d2
					snap88 := d3
					snap89 := d4
					snap90 := d5
					snap91 := d6
					snap92 := d8
					snap93 := d12
					snap94 := d22
					snap95 := d23
					snap96 := d24
					snap97 := d25
					snap98 := d28
					snap99 := d31
					snap100 := d47
					snap101 := d48
					snap102 := d49
					snap103 := d50
					snap104 := d51
					snap105 := d52
					snap106 := d53
					snap107 := d54
					snap108 := d55
					snap109 := d56
					snap110 := d57
					snap111 := d58
					snap112 := d59
					snap113 := d60
					snap114 := d61
					snap115 := d63
					snap116 := d64
					snap117 := d66
					snap118 := d67
					snap119 := d68
					snap120 := d69
					snap121 := d70
					snap122 := d71
					snap123 := d72
					snap124 := d73
					snap125 := d74
					snap126 := d75
					snap127 := d76
					snap128 := d77
					snap129 := d78
					snap130 := d79
					snap131 := d82
					snap132 := d85
					alloc133 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps84)
					}
					ctx.RestoreAllocState(alloc133)
					d1 = snap86
					d2 = snap87
					d3 = snap88
					d4 = snap89
					d5 = snap90
					d6 = snap91
					d8 = snap92
					d12 = snap93
					d22 = snap94
					d23 = snap95
					d24 = snap96
					d25 = snap97
					d28 = snap98
					d31 = snap99
					d47 = snap100
					d48 = snap101
					d49 = snap102
					d50 = snap103
					d51 = snap104
					d52 = snap105
					d53 = snap106
					d54 = snap107
					d55 = snap108
					d56 = snap109
					d57 = snap110
					d58 = snap111
					d59 = snap112
					d60 = snap113
					d61 = snap114
					d63 = snap115
					d64 = snap116
					d66 = snap117
					d67 = snap118
					d68 = snap119
					d69 = snap120
					d70 = snap121
					d71 = snap122
					d72 = snap123
					d73 = snap124
					d74 = snap125
					d75 = snap126
					d76 = snap127
					d77 = snap128
					d78 = snap129
					d79 = snap130
					d82 = snap131
					d85 = snap132
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps83)
					}
					return result
					ctx.FreeDesc(&d78)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps134 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps134)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(32))
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
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["jit-warn-if-fallback"].Fn, args, result)
				}
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d9 JITValueDesc
				_ = d9
				var d13 JITValueDesc
				_ = d13
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
				var d29 JITValueDesc
				_ = d29
				var d32 JITValueDesc
				_ = d32
				var d48 JITValueDesc
				_ = d48
				var d50 JITValueDesc
				_ = d50
				var d51 JITValueDesc
				_ = d51
				var d52 JITValueDesc
				_ = d52
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
				var d56 JITValueDesc
				_ = d56
				var d57 JITValueDesc
				_ = d57
				var d58 JITValueDesc
				_ = d58
				var d59 JITValueDesc
				_ = d59
				var d60 JITValueDesc
				_ = d60
				var d61 JITValueDesc
				_ = d61
				var d63 JITValueDesc
				_ = d63
				var d64 JITValueDesc
				_ = d64
				var d66 JITValueDesc
				_ = d66
				var d67 JITValueDesc
				_ = d67
				var d68 JITValueDesc
				_ = d68
				var d69 JITValueDesc
				_ = d69
				var d70 JITValueDesc
				_ = d70
				var d71 JITValueDesc
				_ = d71
				var d72 JITValueDesc
				_ = d72
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d78 JITValueDesc
				_ = d78
				var d81 JITValueDesc
				_ = d81
				var d84 JITValueDesc
				_ = d84
				var d130 JITValueDesc
				_ = d130
				var d131 JITValueDesc
				_ = d131
				var d132 JITValueDesc
				_ = d132
				var d133 JITValueDesc
				_ = d133
				var d134 JITValueDesc
				_ = d134
				var d136 JITValueDesc
				_ = d136
				var d138 JITValueDesc
				_ = d138
				var d139 JITValueDesc
				_ = d139
				var d142 JITValueDesc
				_ = d142
				var d197 JITValueDesc
				_ = d197
				var d257 JITValueDesc
				_ = d257
				var d258 JITValueDesc
				_ = d258
				var d259 JITValueDesc
				_ = d259
				var d260 JITValueDesc
				_ = d260
				var d262 JITValueDesc
				_ = d262
				var d263 JITValueDesc
				_ = d263
				var d264 JITValueDesc
				_ = d264
				var d265 JITValueDesc
				_ = d265
				var d266 JITValueDesc
				_ = d266
				var d268 JITValueDesc
				_ = d268
				var d269 JITValueDesc
				_ = d269
				var d271 JITValueDesc
				_ = d271
				var d272 JITValueDesc
				_ = d272
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(48))
				d1 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
				d3 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
				_ = d3
				var bbs [11]BBDescriptor
				bbs[2].PhiBase = int32(phiBase0) + int32(0)
				bbs[2].PhiCount = uint16(1)
				bbs[4].PhiBase = int32(phiBase0) + int32(16)
				bbs[4].PhiCount = uint16(1)
				bbs[10].PhiBase = int32(phiBase0) + int32(32)
				bbs[10].PhiCount = uint16(1)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				bbpos_0_8 := int32(-1)
				_ = bbpos_0_8
				lbl9 := ctx.ReserveLabel()
				bbpos_0_9 := int32(-1)
				_ = bbpos_0_9
				lbl10 := ctx.ReserveLabel()
				bbpos_0_10 := int32(-1)
				_ = bbpos_0_10
				lbl11 := ctx.ReserveLabel()
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[0].VisitCount >= 0 {
							ps.General = true
							return bbs[0].RenderPS(ps)
						}
					}
					bbs[0].VisitCount++
					if ps.General {
						if bbs[0].Rendered {
							ctx.EmitJmp(lbl1)
							return result
						}
						bbs[0].Rendered = true
						bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_0 = bbs[0].Address
						ctx.MarkLabel(lbl1)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					ctx.ReclaimUntrackedRegs()
					d4 = args[0]
					d4.ID = 0
					ctx.StabilizeDescForControlFlow(&d4)
					d5 = ctx.EmitGetTagDesc(&d4, JITValueDesc{Loc: LocAny})
					ctx.EnsureDesc(&d5)
					var d6 JITValueDesc
					if d5.Loc == LocImm {
						d6 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0xb))}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d5.Reg, 11)
						ctx.EmitSetcc(r0, CondEqual)
						d6 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d6)
					}
					ctx.FreeDesc(&d5)
					d7 = d6
					ctx.EnsureDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							if ps.General {
								ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[2].PhiBase)+int32(0))
							}
							ps8 := PhiState{General: ps.General}
							ps8.OverlayValues = make([]JITValueDesc, 8)
							ps8.OverlayValues[1] = d1
							ps8.OverlayValues[2] = d2
							ps8.OverlayValues[3] = d3
							ps8.OverlayValues[4] = d4
							ps8.OverlayValues[5] = d5
							ps8.OverlayValues[6] = d6
							ps8.OverlayValues[7] = d7
							ps8.PhiValues = make([]JITValueDesc, 1)
							d9 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
							ps8.PhiValues[0] = d9
							return bbs[2].RenderPS(ps8)
						}
						if ps.General {
						}
						ps10 := PhiState{General: ps.General}
						ps10.OverlayValues = make([]JITValueDesc, 10)
						ps10.OverlayValues[1] = d1
						ps10.OverlayValues[2] = d2
						ps10.OverlayValues[3] = d3
						ps10.OverlayValues[4] = d4
						ps10.OverlayValues[5] = d5
						ps10.OverlayValues[6] = d6
						ps10.OverlayValues[7] = d7
						ps10.OverlayValues[9] = d9
						return bbs[1].RenderPS(ps10)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d7.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl12)
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(lbl12)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl13)
					ctx.EmitJmp(lbl2)
					ps11 := PhiState{General: true}
					ps11.OverlayValues = make([]JITValueDesc, 10)
					ps11.OverlayValues[1] = d1
					ps11.OverlayValues[2] = d2
					ps11.OverlayValues[3] = d3
					ps11.OverlayValues[4] = d4
					ps11.OverlayValues[5] = d5
					ps11.OverlayValues[6] = d6
					ps11.OverlayValues[7] = d7
					ps11.OverlayValues[9] = d9
					ps11.PhiValues = make([]JITValueDesc, 1)
					d13 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
					ps11.PhiValues[0] = d13
					ps12 := PhiState{General: true}
					ps12.OverlayValues = make([]JITValueDesc, 14)
					ps12.OverlayValues[1] = d1
					ps12.OverlayValues[2] = d2
					ps12.OverlayValues[3] = d3
					ps12.OverlayValues[4] = d4
					ps12.OverlayValues[5] = d5
					ps12.OverlayValues[6] = d6
					ps12.OverlayValues[7] = d7
					ps12.OverlayValues[9] = d9
					ps12.OverlayValues[13] = d13
					snap14 := d1
					snap15 := d2
					snap16 := d3
					snap17 := d4
					snap18 := d5
					snap19 := d6
					snap20 := d7
					snap21 := d9
					snap22 := d13
					alloc23 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps11)
					}
					ctx.RestoreAllocState(alloc23)
					d1 = snap14
					d2 = snap15
					d3 = snap16
					d4 = snap17
					d5 = snap18
					d6 = snap19
					d7 = snap20
					d9 = snap21
					d13 = snap22
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps12)
					}
					return result
					ctx.FreeDesc(&d6)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[1].VisitCount >= 0 {
							ps.General = true
							return bbs[1].RenderPS(ps)
						}
					}
					bbs[1].VisitCount++
					if ps.General {
						if bbs[1].Rendered {
							ctx.EmitJmp(lbl2)
							return result
						}
						bbs[1].Rendered = true
						bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_1 = bbs[1].Address
						ctx.MarkLabel(lbl2)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					ctx.ReclaimUntrackedRegs()
					d24 = ctx.EmitGetTagDesc(&d4, JITValueDesc{Loc: LocAny})
					ctx.EnsureDesc(&d24)
					var d25 JITValueDesc
					if d24.Loc == LocImm {
						d25 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d24.Imm.Int()) == uint64(0xa))}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d24.Reg, 10)
						ctx.EmitSetcc(r1, CondEqual)
						d25 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d25)
					}
					ctx.FreeDesc(&d24)
					d26 = d25
					ctx.EnsureDesc(&d26)
					if d26.Loc != LocImm && d26.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d26.Loc == LocImm {
						if d26.Imm.Bool() {
							if ps.General {
							}
							ps27 := PhiState{General: ps.General}
							ps27.OverlayValues = make([]JITValueDesc, 27)
							ps27.OverlayValues[1] = d1
							ps27.OverlayValues[2] = d2
							ps27.OverlayValues[3] = d3
							ps27.OverlayValues[4] = d4
							ps27.OverlayValues[5] = d5
							ps27.OverlayValues[6] = d6
							ps27.OverlayValues[7] = d7
							ps27.OverlayValues[9] = d9
							ps27.OverlayValues[13] = d13
							ps27.OverlayValues[24] = d24
							ps27.OverlayValues[25] = d25
							ps27.OverlayValues[26] = d26
							return bbs[5].RenderPS(ps27)
						}
						if ps.General {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
						}
						ps28 := PhiState{General: ps.General}
						ps28.OverlayValues = make([]JITValueDesc, 27)
						ps28.OverlayValues[1] = d1
						ps28.OverlayValues[2] = d2
						ps28.OverlayValues[3] = d3
						ps28.OverlayValues[4] = d4
						ps28.OverlayValues[5] = d5
						ps28.OverlayValues[6] = d6
						ps28.OverlayValues[7] = d7
						ps28.OverlayValues[9] = d9
						ps28.OverlayValues[13] = d13
						ps28.OverlayValues[24] = d24
						ps28.OverlayValues[25] = d25
						ps28.OverlayValues[26] = d26
						ps28.PhiValues = make([]JITValueDesc, 1)
						d29 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
						ps28.PhiValues[0] = d29
						return bbs[4].RenderPS(ps28)
					}
					if !ps.General {
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d26.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl14)
					ctx.EmitJmp(lbl15)
					ctx.MarkLabel(lbl14)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl15)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					ctx.EmitJmp(lbl5)
					ps30 := PhiState{General: true}
					ps30.OverlayValues = make([]JITValueDesc, 30)
					ps30.OverlayValues[1] = d1
					ps30.OverlayValues[2] = d2
					ps30.OverlayValues[3] = d3
					ps30.OverlayValues[4] = d4
					ps30.OverlayValues[5] = d5
					ps30.OverlayValues[6] = d6
					ps30.OverlayValues[7] = d7
					ps30.OverlayValues[9] = d9
					ps30.OverlayValues[13] = d13
					ps30.OverlayValues[24] = d24
					ps30.OverlayValues[25] = d25
					ps30.OverlayValues[26] = d26
					ps30.OverlayValues[29] = d29
					ps31 := PhiState{General: true}
					ps31.OverlayValues = make([]JITValueDesc, 30)
					ps31.OverlayValues[1] = d1
					ps31.OverlayValues[2] = d2
					ps31.OverlayValues[3] = d3
					ps31.OverlayValues[4] = d4
					ps31.OverlayValues[5] = d5
					ps31.OverlayValues[6] = d6
					ps31.OverlayValues[7] = d7
					ps31.OverlayValues[9] = d9
					ps31.OverlayValues[13] = d13
					ps31.OverlayValues[24] = d24
					ps31.OverlayValues[25] = d25
					ps31.OverlayValues[26] = d26
					ps31.OverlayValues[29] = d29
					ps31.PhiValues = make([]JITValueDesc, 1)
					d32 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ps31.PhiValues[0] = d32
					snap33 := d1
					snap34 := d2
					snap35 := d3
					snap36 := d4
					snap37 := d5
					snap38 := d6
					snap39 := d7
					snap40 := d9
					snap41 := d13
					snap42 := d24
					snap43 := d25
					snap44 := d26
					snap45 := d29
					snap46 := d32
					alloc47 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps31)
					}
					ctx.RestoreAllocState(alloc47)
					d1 = snap33
					d2 = snap34
					d3 = snap35
					d4 = snap36
					d5 = snap37
					d6 = snap38
					d7 = snap39
					d9 = snap40
					d13 = snap41
					d24 = snap42
					d25 = snap43
					d26 = snap44
					d29 = snap45
					d32 = snap46
					if !bbs[5].Rendered {
						return bbs[5].RenderPS(ps30)
					}
					return result
					ctx.FreeDesc(&d25)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d48 := ps.PhiValues[0]
							ctx.EnsureDesc(&d48)
							ctx.EmitStoreToStack(d48, int32(bbs[2].PhiBase)+int32(0))
						}
						if bbs[2].VisitCount >= 0 {
							ps.General = true
							return bbs[2].RenderPS(ps)
						}
					}
					bbs[2].VisitCount++
					if ps.General {
						if bbs[2].Rendered {
							ctx.EmitJmp(lbl3)
							return result
						}
						bbs[2].Rendered = true
						bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_2 = bbs[2].Address
						ctx.MarkLabel(lbl3)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					ps49 := PhiState{General: ps.General}
					ps49.OverlayValues = make([]JITValueDesc, 49)
					ps49.OverlayValues[1] = d1
					ps49.OverlayValues[2] = d2
					ps49.OverlayValues[3] = d3
					ps49.OverlayValues[4] = d4
					ps49.OverlayValues[5] = d5
					ps49.OverlayValues[6] = d6
					ps49.OverlayValues[7] = d7
					ps49.OverlayValues[9] = d9
					ps49.OverlayValues[13] = d13
					ps49.OverlayValues[24] = d24
					ps49.OverlayValues[25] = d25
					ps49.OverlayValues[26] = d26
					ps49.OverlayValues[29] = d29
					ps49.OverlayValues[32] = d32
					ps49.OverlayValues[48] = d48
					return bbs[7].RenderPS(ps49)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[3].VisitCount >= 0 {
							ps.General = true
							return bbs[3].RenderPS(ps)
						}
					}
					bbs[3].VisitCount++
					if ps.General {
						if bbs[3].Rendered {
							ctx.EmitJmp(lbl4)
							return result
						}
						bbs[3].Rendered = true
						bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_3 = bbs[3].Address
						ctx.MarkLabel(lbl4)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					r2 := d4.Loc == LocReg || d4.Loc == LocRegPair || d4.Loc == LocRegTriple
					r3 := d4.Reg
					if r2 {
						ctx.ProtectReg(r3)
					}
					r4 := d4.Loc == LocRegPair || d4.Loc == LocRegTriple
					r5 := d4.Reg2
					if r4 {
						ctx.ProtectReg(r5)
					}
					r6 := d4.Loc == LocRegTriple
					r7 := d4.Reg3
					if r6 {
						ctx.ProtectReg(r7)
					}
					lbl16 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r8 := ctx.AllocReg()
					r9 := ctx.AllocRegExcept(r8)
					ctx.EmitMovRegImm64(r8, 0)
					ctx.EmitMovRegImm64(r9, 0)
					d50 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r8, Reg2: r9}
					ctx.BindReg(r8, &d50)
					ctx.BindReg(r9, &d50)
					ctx.StabilizeDescForControlFlow(&d50)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d4)
					ctx.ReclaimUntrackedRegs()
					d51 = d4
					_ = d51
					ctx.ReclaimUntrackedRegs()
					d52 = ctx.EmitGetTagDesc(&d51, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d51)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d52)
					var d53 JITValueDesc
					if d52.Loc == LocImm {
						d53 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d52.Imm.Int()) != uint64(0xa))}
					} else {
						r10 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d52.Reg, 10)
						ctx.EmitSetcc(r10, CondNotEqual)
						d53 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
						ctx.BindReg(r10, &d53)
					}
					ctx.FreeDesc(&d52)
					ctx.ReclaimUntrackedRegs()
					d54 = d53
					ctx.EnsureDesc(&d54)
					if d54.Loc != LocImm && d54.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl17 := ctx.ReserveLabel()
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					lbl20 := ctx.ReserveLabel()
					if d54.Loc == LocImm {
						if d54.Imm.Bool() {
							ctx.MarkLabel(lbl19)
							ctx.EmitJmp(lbl17)
						} else {
							ctx.MarkLabel(lbl20)
							ctx.EmitJmp(lbl18)
						}
					} else {
						ctx.EmitCmpRegImm32(d54.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl19)
						ctx.EmitJmp(lbl20)
						ctx.MarkLabel(lbl19)
						ctx.EmitJmp(lbl17)
						ctx.MarkLabel(lbl20)
						ctx.EmitJmp(lbl18)
					}
					ctx.FreeDesc(&d53)
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl18)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d55 = args[0]
					d55.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d56 JITValueDesc
					ctx.EnsureDesc(&d55)
					if d55.Loc == LocImm {
						ptrWord, _ := d55.Imm.RawWords()
						d56 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
					} else {
						if d55.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r11 := ctx.AllocReg()
						ctx.EmitMovRegReg(r11, d55.Reg)
						d56 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
						ctx.BindReg(r11, &d56)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d56)
					ctx.EnsureDesc(&d56)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d56)
					ctx.EnsureDesc(&d56)
					ctx.FreeDesc(&d56)
					ctx.ReclaimUntrackedRegs()
					r12 := ctx.AllocReg()
					ctx.EnsureDesc(&d56)
					ctx.EnsureDesc(&d56)
					if d56.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r12, d56)
					}
					ctx.EmitJmp(lbl16)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
					ctx.MarkLabel(lbl16)
					d59 = JITValueDesc{Loc: LocReg, Reg: r12}
					ctx.BindReg(r12, &d59)
					ctx.BindReg(r12, &d59)
					if r2 {
						ctx.UnprotectReg(r3)
					}
					if r4 {
						ctx.UnprotectReg(r5)
					}
					if r6 {
						ctx.UnprotectReg(r7)
					}
					var d60 JITValueDesc
					ctx.EnsureDesc(&d59)
					if d59.Loc == LocImm {
						fieldAddr := uintptr(d59.Imm.Int()) + 56
						r13 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r13, fieldAddr)
						d60 = JITValueDesc{Loc: LocReg, Reg: r13}
						ctx.BindReg(r13, &d60)
					} else {
						off := int32(56)
						baseReg := d59.Reg
						r14 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r14, baseReg, off)
						d60 = JITValueDesc{Loc: LocReg, Reg: r14}
						ctx.BindReg(r14, &d60)
					}
					ctx.FreeDesc(&d59)
					ctx.EnsureDesc(&d60)
					var d61 JITValueDesc
					if d60.Loc == LocImm {
						d61 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d60.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d60)
						if d60.Loc != LocReg && d60.Loc != LocRegPair && d60.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r15 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d60.Reg, 0)
						ctx.EmitSetcc(r15, CondNotEqual)
						d61 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
						ctx.BindReg(r15, &d61)
					}
					ctx.EnsureDesc(&d61)
					ctx.EmitStoreToStack(d61, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d61)
					ctx.FreeDesc(&d60)
					if ps.General {
					}
					ps62 := PhiState{General: ps.General}
					ps62.OverlayValues = make([]JITValueDesc, 62)
					ps62.OverlayValues[1] = d1
					ps62.OverlayValues[2] = d2
					ps62.OverlayValues[3] = d3
					ps62.OverlayValues[4] = d4
					ps62.OverlayValues[5] = d5
					ps62.OverlayValues[6] = d6
					ps62.OverlayValues[7] = d7
					ps62.OverlayValues[9] = d9
					ps62.OverlayValues[13] = d13
					ps62.OverlayValues[24] = d24
					ps62.OverlayValues[25] = d25
					ps62.OverlayValues[26] = d26
					ps62.OverlayValues[29] = d29
					ps62.OverlayValues[32] = d32
					ps62.OverlayValues[48] = d48
					ps62.OverlayValues[50] = d50
					ps62.OverlayValues[51] = d51
					ps62.OverlayValues[52] = d52
					ps62.OverlayValues[53] = d53
					ps62.OverlayValues[54] = d54
					ps62.OverlayValues[55] = d55
					ps62.OverlayValues[56] = d56
					ps62.OverlayValues[57] = d57
					ps62.OverlayValues[58] = d58
					ps62.OverlayValues[59] = d59
					ps62.OverlayValues[60] = d60
					ps62.OverlayValues[61] = d61
					ps62.PhiValues = make([]JITValueDesc, 1)
					if ps62.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps62)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d63 := ps.PhiValues[0]
							ctx.EnsureDesc(&d63)
							ctx.EmitStoreToStack(d63, int32(bbs[4].PhiBase)+int32(0))
						}
						if bbs[4].VisitCount >= 0 {
							ps.General = true
							return bbs[4].RenderPS(ps)
						}
					}
					bbs[4].VisitCount++
					if ps.General {
						if bbs[4].Rendered {
							ctx.EmitJmp(lbl5)
							return result
						}
						bbs[4].Rendered = true
						bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_4 = bbs[4].Address
						ctx.MarkLabel(lbl5)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d2 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d2)
					if ps.General {
						ctx.SyncDesc(&d2)
						if d2.Loc == LocReg {
							ctx.ProtectReg(d2.Reg)
						} else if d2.Loc == LocRegPair {
							ctx.ProtectReg(d2.Reg)
							ctx.ProtectReg(d2.Reg2)
						}
						d64 = d2
						if d64.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d64)
						ctx.EmitStoreToStack(d64, int32(bbs[2].PhiBase)+int32(0))
						if d2.Loc == LocReg {
							ctx.UnprotectReg(d2.Reg)
						} else if d2.Loc == LocRegPair {
							ctx.UnprotectReg(d2.Reg)
							ctx.UnprotectReg(d2.Reg2)
						}
					}
					ps65 := PhiState{General: ps.General}
					ps65.OverlayValues = make([]JITValueDesc, 65)
					ps65.OverlayValues[1] = d1
					ps65.OverlayValues[2] = d2
					ps65.OverlayValues[3] = d3
					ps65.OverlayValues[4] = d4
					ps65.OverlayValues[5] = d5
					ps65.OverlayValues[6] = d6
					ps65.OverlayValues[7] = d7
					ps65.OverlayValues[9] = d9
					ps65.OverlayValues[13] = d13
					ps65.OverlayValues[24] = d24
					ps65.OverlayValues[25] = d25
					ps65.OverlayValues[26] = d26
					ps65.OverlayValues[29] = d29
					ps65.OverlayValues[32] = d32
					ps65.OverlayValues[48] = d48
					ps65.OverlayValues[50] = d50
					ps65.OverlayValues[51] = d51
					ps65.OverlayValues[52] = d52
					ps65.OverlayValues[53] = d53
					ps65.OverlayValues[54] = d54
					ps65.OverlayValues[55] = d55
					ps65.OverlayValues[56] = d56
					ps65.OverlayValues[57] = d57
					ps65.OverlayValues[58] = d58
					ps65.OverlayValues[59] = d59
					ps65.OverlayValues[60] = d60
					ps65.OverlayValues[61] = d61
					ps65.OverlayValues[63] = d63
					ps65.OverlayValues[64] = d64
					ps65.PhiValues = make([]JITValueDesc, 1)
					d66 = d2
					ps65.PhiValues[0] = d66
					if ps65.General && bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					return bbs[2].RenderPS(ps65)
					return result
				}
				bbs[5].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[5].VisitCount >= 0 {
							ps.General = true
							return bbs[5].RenderPS(ps)
						}
					}
					bbs[5].VisitCount++
					if ps.General {
						if bbs[5].Rendered {
							ctx.EmitJmp(lbl6)
							return result
						}
						bbs[5].Rendered = true
						bbs[5].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_5 = bbs[5].Address
						ctx.MarkLabel(lbl6)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					r16 := d4.Loc == LocReg || d4.Loc == LocRegPair || d4.Loc == LocRegTriple
					r17 := d4.Reg
					if r16 {
						ctx.ProtectReg(r17)
					}
					r18 := d4.Loc == LocRegPair || d4.Loc == LocRegTriple
					r19 := d4.Reg2
					if r18 {
						ctx.ProtectReg(r19)
					}
					r20 := d4.Loc == LocRegTriple
					r21 := d4.Reg3
					if r20 {
						ctx.ProtectReg(r21)
					}
					lbl21 := ctx.ReserveLabel()
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					bbpos_2_1 := int32(-1)
					_ = bbpos_2_1
					bbpos_2_2 := int32(-1)
					_ = bbpos_2_2
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r22 := ctx.AllocReg()
					r23 := ctx.AllocRegExcept(r22)
					ctx.EmitMovRegImm64(r22, 0)
					ctx.EmitMovRegImm64(r23, 0)
					d67 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r22, Reg2: r23}
					ctx.BindReg(r22, &d67)
					ctx.BindReg(r23, &d67)
					ctx.StabilizeDescForControlFlow(&d67)
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d4)
					ctx.ReclaimUntrackedRegs()
					d68 = d4
					_ = d68
					ctx.ReclaimUntrackedRegs()
					d69 = ctx.EmitGetTagDesc(&d68, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d68)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d69)
					var d70 JITValueDesc
					if d69.Loc == LocImm {
						d70 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d69.Imm.Int()) != uint64(0xa))}
					} else {
						r24 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d69.Reg, 10)
						ctx.EmitSetcc(r24, CondNotEqual)
						d70 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r24}
						ctx.BindReg(r24, &d70)
					}
					ctx.FreeDesc(&d69)
					ctx.ReclaimUntrackedRegs()
					d71 = d70
					ctx.EnsureDesc(&d71)
					if d71.Loc != LocImm && d71.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl22 := ctx.ReserveLabel()
					lbl23 := ctx.ReserveLabel()
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					if d71.Loc == LocImm {
						if d71.Imm.Bool() {
							ctx.MarkLabel(lbl24)
							ctx.EmitJmp(lbl22)
						} else {
							ctx.MarkLabel(lbl25)
							ctx.EmitJmp(lbl23)
						}
					} else {
						ctx.EmitCmpRegImm32(d71.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl24)
						ctx.EmitJmp(lbl25)
						ctx.MarkLabel(lbl24)
						ctx.EmitJmp(lbl22)
						ctx.MarkLabel(lbl25)
						ctx.EmitJmp(lbl23)
					}
					ctx.FreeDesc(&d70)
					bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl23)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d72 = args[0]
					d72.ID = 0
					ctx.ReclaimUntrackedRegs()
					var d73 JITValueDesc
					ctx.EnsureDesc(&d72)
					if d72.Loc == LocImm {
						ptrWord, _ := d72.Imm.RawWords()
						d73 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(ptrWord))}
					} else {
						if d72.Loc != LocRegPair {
							panic("jitgen: desc field base is not LocRegPair")
						}
						r25 := ctx.AllocReg()
						ctx.EmitMovRegReg(r25, d72.Reg)
						d73 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r25}
						ctx.BindReg(r25, &d73)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					ctx.FreeDesc(&d73)
					ctx.ReclaimUntrackedRegs()
					r26 := ctx.AllocReg()
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					if d73.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r26, d73)
					}
					ctx.EmitJmp(lbl21)
					bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl22)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
					ctx.MarkLabel(lbl21)
					d76 = JITValueDesc{Loc: LocReg, Reg: r26}
					ctx.BindReg(r26, &d76)
					ctx.BindReg(r26, &d76)
					if r16 {
						ctx.UnprotectReg(r17)
					}
					if r18 {
						ctx.UnprotectReg(r19)
					}
					if r20 {
						ctx.UnprotectReg(r21)
					}
					ctx.EnsureDesc(&d76)
					var d77 JITValueDesc
					if d76.Loc == LocImm {
						d77 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d76.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d76)
						if d76.Loc != LocReg && d76.Loc != LocRegPair && d76.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r27 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d76.Reg, 0)
						ctx.EmitSetcc(r27, CondNotEqual)
						d77 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r27}
						ctx.BindReg(r27, &d77)
					}
					ctx.FreeDesc(&d76)
					d78 = d77
					ctx.EnsureDesc(&d78)
					if d78.Loc != LocImm && d78.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d78.Loc == LocImm {
						if d78.Imm.Bool() {
							if ps.General {
							}
							ps79 := PhiState{General: ps.General}
							ps79.OverlayValues = make([]JITValueDesc, 79)
							ps79.OverlayValues[1] = d1
							ps79.OverlayValues[2] = d2
							ps79.OverlayValues[3] = d3
							ps79.OverlayValues[4] = d4
							ps79.OverlayValues[5] = d5
							ps79.OverlayValues[6] = d6
							ps79.OverlayValues[7] = d7
							ps79.OverlayValues[9] = d9
							ps79.OverlayValues[13] = d13
							ps79.OverlayValues[24] = d24
							ps79.OverlayValues[25] = d25
							ps79.OverlayValues[26] = d26
							ps79.OverlayValues[29] = d29
							ps79.OverlayValues[32] = d32
							ps79.OverlayValues[48] = d48
							ps79.OverlayValues[50] = d50
							ps79.OverlayValues[51] = d51
							ps79.OverlayValues[52] = d52
							ps79.OverlayValues[53] = d53
							ps79.OverlayValues[54] = d54
							ps79.OverlayValues[55] = d55
							ps79.OverlayValues[56] = d56
							ps79.OverlayValues[57] = d57
							ps79.OverlayValues[58] = d58
							ps79.OverlayValues[59] = d59
							ps79.OverlayValues[60] = d60
							ps79.OverlayValues[61] = d61
							ps79.OverlayValues[63] = d63
							ps79.OverlayValues[64] = d64
							ps79.OverlayValues[66] = d66
							ps79.OverlayValues[67] = d67
							ps79.OverlayValues[68] = d68
							ps79.OverlayValues[69] = d69
							ps79.OverlayValues[70] = d70
							ps79.OverlayValues[71] = d71
							ps79.OverlayValues[72] = d72
							ps79.OverlayValues[73] = d73
							ps79.OverlayValues[74] = d74
							ps79.OverlayValues[75] = d75
							ps79.OverlayValues[76] = d76
							ps79.OverlayValues[77] = d77
							ps79.OverlayValues[78] = d78
							return bbs[3].RenderPS(ps79)
						}
						if ps.General {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
						}
						ps80 := PhiState{General: ps.General}
						ps80.OverlayValues = make([]JITValueDesc, 79)
						ps80.OverlayValues[1] = d1
						ps80.OverlayValues[2] = d2
						ps80.OverlayValues[3] = d3
						ps80.OverlayValues[4] = d4
						ps80.OverlayValues[5] = d5
						ps80.OverlayValues[6] = d6
						ps80.OverlayValues[7] = d7
						ps80.OverlayValues[9] = d9
						ps80.OverlayValues[13] = d13
						ps80.OverlayValues[24] = d24
						ps80.OverlayValues[25] = d25
						ps80.OverlayValues[26] = d26
						ps80.OverlayValues[29] = d29
						ps80.OverlayValues[32] = d32
						ps80.OverlayValues[48] = d48
						ps80.OverlayValues[50] = d50
						ps80.OverlayValues[51] = d51
						ps80.OverlayValues[52] = d52
						ps80.OverlayValues[53] = d53
						ps80.OverlayValues[54] = d54
						ps80.OverlayValues[55] = d55
						ps80.OverlayValues[56] = d56
						ps80.OverlayValues[57] = d57
						ps80.OverlayValues[58] = d58
						ps80.OverlayValues[59] = d59
						ps80.OverlayValues[60] = d60
						ps80.OverlayValues[61] = d61
						ps80.OverlayValues[63] = d63
						ps80.OverlayValues[64] = d64
						ps80.OverlayValues[66] = d66
						ps80.OverlayValues[67] = d67
						ps80.OverlayValues[68] = d68
						ps80.OverlayValues[69] = d69
						ps80.OverlayValues[70] = d70
						ps80.OverlayValues[71] = d71
						ps80.OverlayValues[72] = d72
						ps80.OverlayValues[73] = d73
						ps80.OverlayValues[74] = d74
						ps80.OverlayValues[75] = d75
						ps80.OverlayValues[76] = d76
						ps80.OverlayValues[77] = d77
						ps80.OverlayValues[78] = d78
						ps80.PhiValues = make([]JITValueDesc, 1)
						d81 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
						ps80.PhiValues[0] = d81
						return bbs[4].RenderPS(ps80)
					}
					if !ps.General {
						ps.General = true
						return bbs[5].RenderPS(ps)
					}
					lbl26 := ctx.ReserveLabel()
					lbl27 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d78.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl26)
					ctx.EmitJmp(lbl27)
					ctx.MarkLabel(lbl26)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl27)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					ctx.EmitJmp(lbl5)
					ps82 := PhiState{General: true}
					ps82.OverlayValues = make([]JITValueDesc, 82)
					ps82.OverlayValues[1] = d1
					ps82.OverlayValues[2] = d2
					ps82.OverlayValues[3] = d3
					ps82.OverlayValues[4] = d4
					ps82.OverlayValues[5] = d5
					ps82.OverlayValues[6] = d6
					ps82.OverlayValues[7] = d7
					ps82.OverlayValues[9] = d9
					ps82.OverlayValues[13] = d13
					ps82.OverlayValues[24] = d24
					ps82.OverlayValues[25] = d25
					ps82.OverlayValues[26] = d26
					ps82.OverlayValues[29] = d29
					ps82.OverlayValues[32] = d32
					ps82.OverlayValues[48] = d48
					ps82.OverlayValues[50] = d50
					ps82.OverlayValues[51] = d51
					ps82.OverlayValues[52] = d52
					ps82.OverlayValues[53] = d53
					ps82.OverlayValues[54] = d54
					ps82.OverlayValues[55] = d55
					ps82.OverlayValues[56] = d56
					ps82.OverlayValues[57] = d57
					ps82.OverlayValues[58] = d58
					ps82.OverlayValues[59] = d59
					ps82.OverlayValues[60] = d60
					ps82.OverlayValues[61] = d61
					ps82.OverlayValues[63] = d63
					ps82.OverlayValues[64] = d64
					ps82.OverlayValues[66] = d66
					ps82.OverlayValues[67] = d67
					ps82.OverlayValues[68] = d68
					ps82.OverlayValues[69] = d69
					ps82.OverlayValues[70] = d70
					ps82.OverlayValues[71] = d71
					ps82.OverlayValues[72] = d72
					ps82.OverlayValues[73] = d73
					ps82.OverlayValues[74] = d74
					ps82.OverlayValues[75] = d75
					ps82.OverlayValues[76] = d76
					ps82.OverlayValues[77] = d77
					ps82.OverlayValues[78] = d78
					ps82.OverlayValues[81] = d81
					ps83 := PhiState{General: true}
					ps83.OverlayValues = make([]JITValueDesc, 82)
					ps83.OverlayValues[1] = d1
					ps83.OverlayValues[2] = d2
					ps83.OverlayValues[3] = d3
					ps83.OverlayValues[4] = d4
					ps83.OverlayValues[5] = d5
					ps83.OverlayValues[6] = d6
					ps83.OverlayValues[7] = d7
					ps83.OverlayValues[9] = d9
					ps83.OverlayValues[13] = d13
					ps83.OverlayValues[24] = d24
					ps83.OverlayValues[25] = d25
					ps83.OverlayValues[26] = d26
					ps83.OverlayValues[29] = d29
					ps83.OverlayValues[32] = d32
					ps83.OverlayValues[48] = d48
					ps83.OverlayValues[50] = d50
					ps83.OverlayValues[51] = d51
					ps83.OverlayValues[52] = d52
					ps83.OverlayValues[53] = d53
					ps83.OverlayValues[54] = d54
					ps83.OverlayValues[55] = d55
					ps83.OverlayValues[56] = d56
					ps83.OverlayValues[57] = d57
					ps83.OverlayValues[58] = d58
					ps83.OverlayValues[59] = d59
					ps83.OverlayValues[60] = d60
					ps83.OverlayValues[61] = d61
					ps83.OverlayValues[63] = d63
					ps83.OverlayValues[64] = d64
					ps83.OverlayValues[66] = d66
					ps83.OverlayValues[67] = d67
					ps83.OverlayValues[68] = d68
					ps83.OverlayValues[69] = d69
					ps83.OverlayValues[70] = d70
					ps83.OverlayValues[71] = d71
					ps83.OverlayValues[72] = d72
					ps83.OverlayValues[73] = d73
					ps83.OverlayValues[74] = d74
					ps83.OverlayValues[75] = d75
					ps83.OverlayValues[76] = d76
					ps83.OverlayValues[77] = d77
					ps83.OverlayValues[78] = d78
					ps83.OverlayValues[81] = d81
					ps83.PhiValues = make([]JITValueDesc, 1)
					d84 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ps83.PhiValues[0] = d84
					snap85 := d1
					snap86 := d2
					snap87 := d3
					snap88 := d4
					snap89 := d5
					snap90 := d6
					snap91 := d7
					snap92 := d9
					snap93 := d13
					snap94 := d24
					snap95 := d25
					snap96 := d26
					snap97 := d29
					snap98 := d32
					snap99 := d48
					snap100 := d50
					snap101 := d51
					snap102 := d52
					snap103 := d53
					snap104 := d54
					snap105 := d55
					snap106 := d56
					snap107 := d57
					snap108 := d58
					snap109 := d59
					snap110 := d60
					snap111 := d61
					snap112 := d63
					snap113 := d64
					snap114 := d66
					snap115 := d67
					snap116 := d68
					snap117 := d69
					snap118 := d70
					snap119 := d71
					snap120 := d72
					snap121 := d73
					snap122 := d74
					snap123 := d75
					snap124 := d76
					snap125 := d77
					snap126 := d78
					snap127 := d81
					snap128 := d84
					alloc129 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps83)
					}
					ctx.RestoreAllocState(alloc129)
					d1 = snap85
					d2 = snap86
					d3 = snap87
					d4 = snap88
					d5 = snap89
					d6 = snap90
					d7 = snap91
					d9 = snap92
					d13 = snap93
					d24 = snap94
					d25 = snap95
					d26 = snap96
					d29 = snap97
					d32 = snap98
					d48 = snap99
					d50 = snap100
					d51 = snap101
					d52 = snap102
					d53 = snap103
					d54 = snap104
					d55 = snap105
					d56 = snap106
					d57 = snap107
					d58 = snap108
					d59 = snap109
					d60 = snap110
					d61 = snap111
					d63 = snap112
					d64 = snap113
					d66 = snap114
					d67 = snap115
					d68 = snap116
					d69 = snap117
					d70 = snap118
					d71 = snap119
					d72 = snap120
					d73 = snap121
					d74 = snap122
					d75 = snap123
					d76 = snap124
					d77 = snap125
					d78 = snap126
					d81 = snap127
					d84 = snap128
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps82)
					}
					return result
					ctx.FreeDesc(&d77)
					return result
				}
				bbs[6].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[6].VisitCount >= 0 {
							ps.General = true
							return bbs[6].RenderPS(ps)
						}
					}
					bbs[6].VisitCount++
					if ps.General {
						if bbs[6].Rendered {
							ctx.EmitJmp(lbl7)
							return result
						}
						bbs[6].Rendered = true
						bbs[6].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_6 = bbs[6].Address
						ctx.MarkLabel(lbl7)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d4.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d4.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d4)
						} else if d4.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d4)
						} else if d4.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d4)
						} else if d4.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d4.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d4 = tmpPair
					} else if d4.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d4.Type, Reg: ctx.AllocRegExcept(d4.Reg), Reg2: ctx.AllocRegExcept(d4.Reg)}
						switch d4.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d4)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d4)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d4)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d4)
						d4 = tmpPair
					}
					if d4.Loc != LocRegPair && d4.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (SerializeToString arg0)")
					}
					d130 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d130.Loc == LocRegPair || d130.Loc == LocStackPair || d130.Loc == LocRegTriple || d130.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d4)
					ctx.SyncDesc(&d130)
					d131 = ctx.EmitGoCallScalar(GoFuncAddr(SerializeToString), []JITValueDesc{d4, d130}, 2)
					ctx.BindReg(d131.Reg, &d131)
					ctx.BindReg(d131.Reg2, &d131)
					ctx.StabilizeDescForControlFlow(&d131)
					d132 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d132)
					var d133 JITValueDesc
					if d132.Loc == LocImm {
						d133 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d132.Imm.Int() > 1)}
					} else {
						r28 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d132.Reg, 1)
						ctx.EmitSetcc(r28, CondSignedGreater)
						d133 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r28}
						ctx.BindReg(r28, &d133)
					}
					ctx.FreeDesc(&d132)
					d134 = d133
					ctx.EnsureDesc(&d134)
					if d134.Loc != LocImm && d134.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d134.Loc == LocImm {
						if d134.Imm.Bool() {
							if ps.General {
							}
							ps135 := PhiState{General: ps.General}
							ps135.OverlayValues = make([]JITValueDesc, 135)
							ps135.OverlayValues[1] = d1
							ps135.OverlayValues[2] = d2
							ps135.OverlayValues[3] = d3
							ps135.OverlayValues[4] = d4
							ps135.OverlayValues[5] = d5
							ps135.OverlayValues[6] = d6
							ps135.OverlayValues[7] = d7
							ps135.OverlayValues[9] = d9
							ps135.OverlayValues[13] = d13
							ps135.OverlayValues[24] = d24
							ps135.OverlayValues[25] = d25
							ps135.OverlayValues[26] = d26
							ps135.OverlayValues[29] = d29
							ps135.OverlayValues[32] = d32
							ps135.OverlayValues[48] = d48
							ps135.OverlayValues[50] = d50
							ps135.OverlayValues[51] = d51
							ps135.OverlayValues[52] = d52
							ps135.OverlayValues[53] = d53
							ps135.OverlayValues[54] = d54
							ps135.OverlayValues[55] = d55
							ps135.OverlayValues[56] = d56
							ps135.OverlayValues[57] = d57
							ps135.OverlayValues[58] = d58
							ps135.OverlayValues[59] = d59
							ps135.OverlayValues[60] = d60
							ps135.OverlayValues[61] = d61
							ps135.OverlayValues[63] = d63
							ps135.OverlayValues[64] = d64
							ps135.OverlayValues[66] = d66
							ps135.OverlayValues[67] = d67
							ps135.OverlayValues[68] = d68
							ps135.OverlayValues[69] = d69
							ps135.OverlayValues[70] = d70
							ps135.OverlayValues[71] = d71
							ps135.OverlayValues[72] = d72
							ps135.OverlayValues[73] = d73
							ps135.OverlayValues[74] = d74
							ps135.OverlayValues[75] = d75
							ps135.OverlayValues[76] = d76
							ps135.OverlayValues[77] = d77
							ps135.OverlayValues[78] = d78
							ps135.OverlayValues[81] = d81
							ps135.OverlayValues[84] = d84
							ps135.OverlayValues[130] = d130
							ps135.OverlayValues[131] = d131
							ps135.OverlayValues[132] = d132
							ps135.OverlayValues[133] = d133
							ps135.OverlayValues[134] = d134
							return bbs[9].RenderPS(ps135)
						}
						if ps.General {
							ctx.SyncDesc(&d131)
							if d131.Loc == LocReg {
								ctx.ProtectReg(d131.Reg)
							} else if d131.Loc == LocRegPair {
								ctx.ProtectReg(d131.Reg)
								ctx.ProtectReg(d131.Reg2)
							}
							d136 = d131
							if d136.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.SyncDesc(&d136)
							if d136.Loc == LocStackPair {
								ctx.EmitCopyStackWords(d136, int32(bbs[10].PhiBase)+int32(0), 2)
							} else if d136.Loc == LocInputPair {
								ctx.EnsureDesc(&d136)
								ctx.EmitStoreScmerToStack(d136, int32(bbs[10].PhiBase)+int32(0))
							} else if d136.Loc == LocRegPair || d136.Loc == LocImm {
								ctx.EmitStoreScmerToStack(d136, int32(bbs[10].PhiBase)+int32(0))
							} else {
								ctx.EnsureDesc(&d136)
								ctx.EmitStoreToStack(d136, int32(bbs[10].PhiBase)+int32(0))
								ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[10].PhiBase)+int32(0))+8)
							}
							if d131.Loc == LocReg {
								ctx.UnprotectReg(d131.Reg)
							} else if d131.Loc == LocRegPair {
								ctx.UnprotectReg(d131.Reg)
								ctx.UnprotectReg(d131.Reg2)
							}
						}
						ps137 := PhiState{General: ps.General}
						ps137.OverlayValues = make([]JITValueDesc, 137)
						ps137.OverlayValues[1] = d1
						ps137.OverlayValues[2] = d2
						ps137.OverlayValues[3] = d3
						ps137.OverlayValues[4] = d4
						ps137.OverlayValues[5] = d5
						ps137.OverlayValues[6] = d6
						ps137.OverlayValues[7] = d7
						ps137.OverlayValues[9] = d9
						ps137.OverlayValues[13] = d13
						ps137.OverlayValues[24] = d24
						ps137.OverlayValues[25] = d25
						ps137.OverlayValues[26] = d26
						ps137.OverlayValues[29] = d29
						ps137.OverlayValues[32] = d32
						ps137.OverlayValues[48] = d48
						ps137.OverlayValues[50] = d50
						ps137.OverlayValues[51] = d51
						ps137.OverlayValues[52] = d52
						ps137.OverlayValues[53] = d53
						ps137.OverlayValues[54] = d54
						ps137.OverlayValues[55] = d55
						ps137.OverlayValues[56] = d56
						ps137.OverlayValues[57] = d57
						ps137.OverlayValues[58] = d58
						ps137.OverlayValues[59] = d59
						ps137.OverlayValues[60] = d60
						ps137.OverlayValues[61] = d61
						ps137.OverlayValues[63] = d63
						ps137.OverlayValues[64] = d64
						ps137.OverlayValues[66] = d66
						ps137.OverlayValues[67] = d67
						ps137.OverlayValues[68] = d68
						ps137.OverlayValues[69] = d69
						ps137.OverlayValues[70] = d70
						ps137.OverlayValues[71] = d71
						ps137.OverlayValues[72] = d72
						ps137.OverlayValues[73] = d73
						ps137.OverlayValues[74] = d74
						ps137.OverlayValues[75] = d75
						ps137.OverlayValues[76] = d76
						ps137.OverlayValues[77] = d77
						ps137.OverlayValues[78] = d78
						ps137.OverlayValues[81] = d81
						ps137.OverlayValues[84] = d84
						ps137.OverlayValues[130] = d130
						ps137.OverlayValues[131] = d131
						ps137.OverlayValues[132] = d132
						ps137.OverlayValues[133] = d133
						ps137.OverlayValues[134] = d134
						ps137.OverlayValues[136] = d136
						ps137.PhiValues = make([]JITValueDesc, 1)
						d138 = d131
						ps137.PhiValues[0] = d138
						return bbs[10].RenderPS(ps137)
					}
					if !ps.General {
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					lbl28 := ctx.ReserveLabel()
					lbl29 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d134.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl28)
					ctx.EmitJmp(lbl29)
					ctx.MarkLabel(lbl28)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl29)
					ctx.SyncDesc(&d131)
					if d131.Loc == LocReg {
						ctx.ProtectReg(d131.Reg)
					} else if d131.Loc == LocRegPair {
						ctx.ProtectReg(d131.Reg)
						ctx.ProtectReg(d131.Reg2)
					}
					d139 = d131
					if d139.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d139)
					if d139.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d139, int32(bbs[10].PhiBase)+int32(0), 2)
					} else if d139.Loc == LocInputPair {
						ctx.EnsureDesc(&d139)
						ctx.EmitStoreScmerToStack(d139, int32(bbs[10].PhiBase)+int32(0))
					} else if d139.Loc == LocRegPair || d139.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d139, int32(bbs[10].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d139)
						ctx.EmitStoreToStack(d139, int32(bbs[10].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[10].PhiBase)+int32(0))+8)
					}
					if d131.Loc == LocReg {
						ctx.UnprotectReg(d131.Reg)
					} else if d131.Loc == LocRegPair {
						ctx.UnprotectReg(d131.Reg)
						ctx.UnprotectReg(d131.Reg2)
					}
					ctx.EmitJmp(lbl11)
					ps140 := PhiState{General: true}
					ps140.OverlayValues = make([]JITValueDesc, 140)
					ps140.OverlayValues[1] = d1
					ps140.OverlayValues[2] = d2
					ps140.OverlayValues[3] = d3
					ps140.OverlayValues[4] = d4
					ps140.OverlayValues[5] = d5
					ps140.OverlayValues[6] = d6
					ps140.OverlayValues[7] = d7
					ps140.OverlayValues[9] = d9
					ps140.OverlayValues[13] = d13
					ps140.OverlayValues[24] = d24
					ps140.OverlayValues[25] = d25
					ps140.OverlayValues[26] = d26
					ps140.OverlayValues[29] = d29
					ps140.OverlayValues[32] = d32
					ps140.OverlayValues[48] = d48
					ps140.OverlayValues[50] = d50
					ps140.OverlayValues[51] = d51
					ps140.OverlayValues[52] = d52
					ps140.OverlayValues[53] = d53
					ps140.OverlayValues[54] = d54
					ps140.OverlayValues[55] = d55
					ps140.OverlayValues[56] = d56
					ps140.OverlayValues[57] = d57
					ps140.OverlayValues[58] = d58
					ps140.OverlayValues[59] = d59
					ps140.OverlayValues[60] = d60
					ps140.OverlayValues[61] = d61
					ps140.OverlayValues[63] = d63
					ps140.OverlayValues[64] = d64
					ps140.OverlayValues[66] = d66
					ps140.OverlayValues[67] = d67
					ps140.OverlayValues[68] = d68
					ps140.OverlayValues[69] = d69
					ps140.OverlayValues[70] = d70
					ps140.OverlayValues[71] = d71
					ps140.OverlayValues[72] = d72
					ps140.OverlayValues[73] = d73
					ps140.OverlayValues[74] = d74
					ps140.OverlayValues[75] = d75
					ps140.OverlayValues[76] = d76
					ps140.OverlayValues[77] = d77
					ps140.OverlayValues[78] = d78
					ps140.OverlayValues[81] = d81
					ps140.OverlayValues[84] = d84
					ps140.OverlayValues[130] = d130
					ps140.OverlayValues[131] = d131
					ps140.OverlayValues[132] = d132
					ps140.OverlayValues[133] = d133
					ps140.OverlayValues[134] = d134
					ps140.OverlayValues[136] = d136
					ps140.OverlayValues[138] = d138
					ps140.OverlayValues[139] = d139
					ps141 := PhiState{General: true}
					ps141.OverlayValues = make([]JITValueDesc, 140)
					ps141.OverlayValues[1] = d1
					ps141.OverlayValues[2] = d2
					ps141.OverlayValues[3] = d3
					ps141.OverlayValues[4] = d4
					ps141.OverlayValues[5] = d5
					ps141.OverlayValues[6] = d6
					ps141.OverlayValues[7] = d7
					ps141.OverlayValues[9] = d9
					ps141.OverlayValues[13] = d13
					ps141.OverlayValues[24] = d24
					ps141.OverlayValues[25] = d25
					ps141.OverlayValues[26] = d26
					ps141.OverlayValues[29] = d29
					ps141.OverlayValues[32] = d32
					ps141.OverlayValues[48] = d48
					ps141.OverlayValues[50] = d50
					ps141.OverlayValues[51] = d51
					ps141.OverlayValues[52] = d52
					ps141.OverlayValues[53] = d53
					ps141.OverlayValues[54] = d54
					ps141.OverlayValues[55] = d55
					ps141.OverlayValues[56] = d56
					ps141.OverlayValues[57] = d57
					ps141.OverlayValues[58] = d58
					ps141.OverlayValues[59] = d59
					ps141.OverlayValues[60] = d60
					ps141.OverlayValues[61] = d61
					ps141.OverlayValues[63] = d63
					ps141.OverlayValues[64] = d64
					ps141.OverlayValues[66] = d66
					ps141.OverlayValues[67] = d67
					ps141.OverlayValues[68] = d68
					ps141.OverlayValues[69] = d69
					ps141.OverlayValues[70] = d70
					ps141.OverlayValues[71] = d71
					ps141.OverlayValues[72] = d72
					ps141.OverlayValues[73] = d73
					ps141.OverlayValues[74] = d74
					ps141.OverlayValues[75] = d75
					ps141.OverlayValues[76] = d76
					ps141.OverlayValues[77] = d77
					ps141.OverlayValues[78] = d78
					ps141.OverlayValues[81] = d81
					ps141.OverlayValues[84] = d84
					ps141.OverlayValues[130] = d130
					ps141.OverlayValues[131] = d131
					ps141.OverlayValues[132] = d132
					ps141.OverlayValues[133] = d133
					ps141.OverlayValues[134] = d134
					ps141.OverlayValues[136] = d136
					ps141.OverlayValues[138] = d138
					ps141.OverlayValues[139] = d139
					ps141.PhiValues = make([]JITValueDesc, 1)
					d142 = d131
					ps141.PhiValues[0] = d142
					snap143 := d1
					snap144 := d2
					snap145 := d3
					snap146 := d4
					snap147 := d5
					snap148 := d6
					snap149 := d7
					snap150 := d9
					snap151 := d13
					snap152 := d24
					snap153 := d25
					snap154 := d26
					snap155 := d29
					snap156 := d32
					snap157 := d48
					snap158 := d50
					snap159 := d51
					snap160 := d52
					snap161 := d53
					snap162 := d54
					snap163 := d55
					snap164 := d56
					snap165 := d57
					snap166 := d58
					snap167 := d59
					snap168 := d60
					snap169 := d61
					snap170 := d63
					snap171 := d64
					snap172 := d66
					snap173 := d67
					snap174 := d68
					snap175 := d69
					snap176 := d70
					snap177 := d71
					snap178 := d72
					snap179 := d73
					snap180 := d74
					snap181 := d75
					snap182 := d76
					snap183 := d77
					snap184 := d78
					snap185 := d81
					snap186 := d84
					snap187 := d130
					snap188 := d131
					snap189 := d132
					snap190 := d133
					snap191 := d134
					snap192 := d136
					snap193 := d138
					snap194 := d139
					snap195 := d142
					alloc196 := ctx.SnapshotAllocState()
					if !bbs[10].Rendered {
						bbs[10].RenderPS(ps141)
					}
					ctx.RestoreAllocState(alloc196)
					d1 = snap143
					d2 = snap144
					d3 = snap145
					d4 = snap146
					d5 = snap147
					d6 = snap148
					d7 = snap149
					d9 = snap150
					d13 = snap151
					d24 = snap152
					d25 = snap153
					d26 = snap154
					d29 = snap155
					d32 = snap156
					d48 = snap157
					d50 = snap158
					d51 = snap159
					d52 = snap160
					d53 = snap161
					d54 = snap162
					d55 = snap163
					d56 = snap164
					d57 = snap165
					d58 = snap166
					d59 = snap167
					d60 = snap168
					d61 = snap169
					d63 = snap170
					d64 = snap171
					d66 = snap172
					d67 = snap173
					d68 = snap174
					d69 = snap175
					d70 = snap176
					d71 = snap177
					d72 = snap178
					d73 = snap179
					d74 = snap180
					d75 = snap181
					d76 = snap182
					d77 = snap183
					d78 = snap184
					d81 = snap185
					d84 = snap186
					d130 = snap187
					d131 = snap188
					d132 = snap189
					d133 = snap190
					d134 = snap191
					d136 = snap192
					d138 = snap193
					d139 = snap194
					d142 = snap195
					if !bbs[9].Rendered {
						return bbs[9].RenderPS(ps140)
					}
					return result
					ctx.FreeDesc(&d133)
					return result
				}
				bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[7].VisitCount >= 0 {
							ps.General = true
							return bbs[7].RenderPS(ps)
						}
					}
					bbs[7].VisitCount++
					if ps.General {
						if bbs[7].Rendered {
							ctx.EmitJmp(lbl8)
							return result
						}
						bbs[7].Rendered = true
						bbs[7].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_7 = bbs[7].Address
						ctx.MarkLabel(lbl8)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
						d130 = ps.OverlayValues[130]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
						d132 = ps.OverlayValues[132]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d4, &result)
						result.Type = d4.Type
					} else {
						switch d4.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d4)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d4)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d4)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d4, &result)
							result.Type = d4.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[8].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[8].VisitCount >= 0 {
							ps.General = true
							return bbs[8].RenderPS(ps)
						}
					}
					bbs[8].VisitCount++
					if ps.General {
						if bbs[8].Rendered {
							ctx.EmitJmp(lbl9)
							return result
						}
						bbs[8].Rendered = true
						bbs[8].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_8 = bbs[8].Address
						ctx.MarkLabel(lbl9)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
						d130 = ps.OverlayValues[130]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
						d132 = ps.OverlayValues[132]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					ctx.ReclaimUntrackedRegs()
					d197 = d1
					ctx.EnsureDesc(&d197)
					if d197.Loc != LocImm && d197.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d197.Loc == LocImm {
						if d197.Imm.Bool() {
							if ps.General {
							}
							ps198 := PhiState{General: ps.General}
							ps198.OverlayValues = make([]JITValueDesc, 198)
							ps198.OverlayValues[1] = d1
							ps198.OverlayValues[2] = d2
							ps198.OverlayValues[3] = d3
							ps198.OverlayValues[4] = d4
							ps198.OverlayValues[5] = d5
							ps198.OverlayValues[6] = d6
							ps198.OverlayValues[7] = d7
							ps198.OverlayValues[9] = d9
							ps198.OverlayValues[13] = d13
							ps198.OverlayValues[24] = d24
							ps198.OverlayValues[25] = d25
							ps198.OverlayValues[26] = d26
							ps198.OverlayValues[29] = d29
							ps198.OverlayValues[32] = d32
							ps198.OverlayValues[48] = d48
							ps198.OverlayValues[50] = d50
							ps198.OverlayValues[51] = d51
							ps198.OverlayValues[52] = d52
							ps198.OverlayValues[53] = d53
							ps198.OverlayValues[54] = d54
							ps198.OverlayValues[55] = d55
							ps198.OverlayValues[56] = d56
							ps198.OverlayValues[57] = d57
							ps198.OverlayValues[58] = d58
							ps198.OverlayValues[59] = d59
							ps198.OverlayValues[60] = d60
							ps198.OverlayValues[61] = d61
							ps198.OverlayValues[63] = d63
							ps198.OverlayValues[64] = d64
							ps198.OverlayValues[66] = d66
							ps198.OverlayValues[67] = d67
							ps198.OverlayValues[68] = d68
							ps198.OverlayValues[69] = d69
							ps198.OverlayValues[70] = d70
							ps198.OverlayValues[71] = d71
							ps198.OverlayValues[72] = d72
							ps198.OverlayValues[73] = d73
							ps198.OverlayValues[74] = d74
							ps198.OverlayValues[75] = d75
							ps198.OverlayValues[76] = d76
							ps198.OverlayValues[77] = d77
							ps198.OverlayValues[78] = d78
							ps198.OverlayValues[81] = d81
							ps198.OverlayValues[84] = d84
							ps198.OverlayValues[130] = d130
							ps198.OverlayValues[131] = d131
							ps198.OverlayValues[132] = d132
							ps198.OverlayValues[133] = d133
							ps198.OverlayValues[134] = d134
							ps198.OverlayValues[136] = d136
							ps198.OverlayValues[138] = d138
							ps198.OverlayValues[139] = d139
							ps198.OverlayValues[142] = d142
							ps198.OverlayValues[197] = d197
							return bbs[7].RenderPS(ps198)
						}
						if ps.General {
						}
						ps199 := PhiState{General: ps.General}
						ps199.OverlayValues = make([]JITValueDesc, 198)
						ps199.OverlayValues[1] = d1
						ps199.OverlayValues[2] = d2
						ps199.OverlayValues[3] = d3
						ps199.OverlayValues[4] = d4
						ps199.OverlayValues[5] = d5
						ps199.OverlayValues[6] = d6
						ps199.OverlayValues[7] = d7
						ps199.OverlayValues[9] = d9
						ps199.OverlayValues[13] = d13
						ps199.OverlayValues[24] = d24
						ps199.OverlayValues[25] = d25
						ps199.OverlayValues[26] = d26
						ps199.OverlayValues[29] = d29
						ps199.OverlayValues[32] = d32
						ps199.OverlayValues[48] = d48
						ps199.OverlayValues[50] = d50
						ps199.OverlayValues[51] = d51
						ps199.OverlayValues[52] = d52
						ps199.OverlayValues[53] = d53
						ps199.OverlayValues[54] = d54
						ps199.OverlayValues[55] = d55
						ps199.OverlayValues[56] = d56
						ps199.OverlayValues[57] = d57
						ps199.OverlayValues[58] = d58
						ps199.OverlayValues[59] = d59
						ps199.OverlayValues[60] = d60
						ps199.OverlayValues[61] = d61
						ps199.OverlayValues[63] = d63
						ps199.OverlayValues[64] = d64
						ps199.OverlayValues[66] = d66
						ps199.OverlayValues[67] = d67
						ps199.OverlayValues[68] = d68
						ps199.OverlayValues[69] = d69
						ps199.OverlayValues[70] = d70
						ps199.OverlayValues[71] = d71
						ps199.OverlayValues[72] = d72
						ps199.OverlayValues[73] = d73
						ps199.OverlayValues[74] = d74
						ps199.OverlayValues[75] = d75
						ps199.OverlayValues[76] = d76
						ps199.OverlayValues[77] = d77
						ps199.OverlayValues[78] = d78
						ps199.OverlayValues[81] = d81
						ps199.OverlayValues[84] = d84
						ps199.OverlayValues[130] = d130
						ps199.OverlayValues[131] = d131
						ps199.OverlayValues[132] = d132
						ps199.OverlayValues[133] = d133
						ps199.OverlayValues[134] = d134
						ps199.OverlayValues[136] = d136
						ps199.OverlayValues[138] = d138
						ps199.OverlayValues[139] = d139
						ps199.OverlayValues[142] = d142
						ps199.OverlayValues[197] = d197
						return bbs[6].RenderPS(ps199)
					}
					if !ps.General {
						ps.General = true
						return bbs[8].RenderPS(ps)
					}
					lbl30 := ctx.ReserveLabel()
					lbl31 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d197.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl30)
					ctx.EmitJmp(lbl31)
					ctx.MarkLabel(lbl30)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl31)
					ctx.EmitJmp(lbl7)
					ps200 := PhiState{General: true}
					ps200.OverlayValues = make([]JITValueDesc, 198)
					ps200.OverlayValues[1] = d1
					ps200.OverlayValues[2] = d2
					ps200.OverlayValues[3] = d3
					ps200.OverlayValues[4] = d4
					ps200.OverlayValues[5] = d5
					ps200.OverlayValues[6] = d6
					ps200.OverlayValues[7] = d7
					ps200.OverlayValues[9] = d9
					ps200.OverlayValues[13] = d13
					ps200.OverlayValues[24] = d24
					ps200.OverlayValues[25] = d25
					ps200.OverlayValues[26] = d26
					ps200.OverlayValues[29] = d29
					ps200.OverlayValues[32] = d32
					ps200.OverlayValues[48] = d48
					ps200.OverlayValues[50] = d50
					ps200.OverlayValues[51] = d51
					ps200.OverlayValues[52] = d52
					ps200.OverlayValues[53] = d53
					ps200.OverlayValues[54] = d54
					ps200.OverlayValues[55] = d55
					ps200.OverlayValues[56] = d56
					ps200.OverlayValues[57] = d57
					ps200.OverlayValues[58] = d58
					ps200.OverlayValues[59] = d59
					ps200.OverlayValues[60] = d60
					ps200.OverlayValues[61] = d61
					ps200.OverlayValues[63] = d63
					ps200.OverlayValues[64] = d64
					ps200.OverlayValues[66] = d66
					ps200.OverlayValues[67] = d67
					ps200.OverlayValues[68] = d68
					ps200.OverlayValues[69] = d69
					ps200.OverlayValues[70] = d70
					ps200.OverlayValues[71] = d71
					ps200.OverlayValues[72] = d72
					ps200.OverlayValues[73] = d73
					ps200.OverlayValues[74] = d74
					ps200.OverlayValues[75] = d75
					ps200.OverlayValues[76] = d76
					ps200.OverlayValues[77] = d77
					ps200.OverlayValues[78] = d78
					ps200.OverlayValues[81] = d81
					ps200.OverlayValues[84] = d84
					ps200.OverlayValues[130] = d130
					ps200.OverlayValues[131] = d131
					ps200.OverlayValues[132] = d132
					ps200.OverlayValues[133] = d133
					ps200.OverlayValues[134] = d134
					ps200.OverlayValues[136] = d136
					ps200.OverlayValues[138] = d138
					ps200.OverlayValues[139] = d139
					ps200.OverlayValues[142] = d142
					ps200.OverlayValues[197] = d197
					ps201 := PhiState{General: true}
					ps201.OverlayValues = make([]JITValueDesc, 198)
					ps201.OverlayValues[1] = d1
					ps201.OverlayValues[2] = d2
					ps201.OverlayValues[3] = d3
					ps201.OverlayValues[4] = d4
					ps201.OverlayValues[5] = d5
					ps201.OverlayValues[6] = d6
					ps201.OverlayValues[7] = d7
					ps201.OverlayValues[9] = d9
					ps201.OverlayValues[13] = d13
					ps201.OverlayValues[24] = d24
					ps201.OverlayValues[25] = d25
					ps201.OverlayValues[26] = d26
					ps201.OverlayValues[29] = d29
					ps201.OverlayValues[32] = d32
					ps201.OverlayValues[48] = d48
					ps201.OverlayValues[50] = d50
					ps201.OverlayValues[51] = d51
					ps201.OverlayValues[52] = d52
					ps201.OverlayValues[53] = d53
					ps201.OverlayValues[54] = d54
					ps201.OverlayValues[55] = d55
					ps201.OverlayValues[56] = d56
					ps201.OverlayValues[57] = d57
					ps201.OverlayValues[58] = d58
					ps201.OverlayValues[59] = d59
					ps201.OverlayValues[60] = d60
					ps201.OverlayValues[61] = d61
					ps201.OverlayValues[63] = d63
					ps201.OverlayValues[64] = d64
					ps201.OverlayValues[66] = d66
					ps201.OverlayValues[67] = d67
					ps201.OverlayValues[68] = d68
					ps201.OverlayValues[69] = d69
					ps201.OverlayValues[70] = d70
					ps201.OverlayValues[71] = d71
					ps201.OverlayValues[72] = d72
					ps201.OverlayValues[73] = d73
					ps201.OverlayValues[74] = d74
					ps201.OverlayValues[75] = d75
					ps201.OverlayValues[76] = d76
					ps201.OverlayValues[77] = d77
					ps201.OverlayValues[78] = d78
					ps201.OverlayValues[81] = d81
					ps201.OverlayValues[84] = d84
					ps201.OverlayValues[130] = d130
					ps201.OverlayValues[131] = d131
					ps201.OverlayValues[132] = d132
					ps201.OverlayValues[133] = d133
					ps201.OverlayValues[134] = d134
					ps201.OverlayValues[136] = d136
					ps201.OverlayValues[138] = d138
					ps201.OverlayValues[139] = d139
					ps201.OverlayValues[142] = d142
					ps201.OverlayValues[197] = d197
					snap202 := d1
					snap203 := d2
					snap204 := d3
					snap205 := d4
					snap206 := d5
					snap207 := d6
					snap208 := d7
					snap209 := d9
					snap210 := d13
					snap211 := d24
					snap212 := d25
					snap213 := d26
					snap214 := d29
					snap215 := d32
					snap216 := d48
					snap217 := d50
					snap218 := d51
					snap219 := d52
					snap220 := d53
					snap221 := d54
					snap222 := d55
					snap223 := d56
					snap224 := d57
					snap225 := d58
					snap226 := d59
					snap227 := d60
					snap228 := d61
					snap229 := d63
					snap230 := d64
					snap231 := d66
					snap232 := d67
					snap233 := d68
					snap234 := d69
					snap235 := d70
					snap236 := d71
					snap237 := d72
					snap238 := d73
					snap239 := d74
					snap240 := d75
					snap241 := d76
					snap242 := d77
					snap243 := d78
					snap244 := d81
					snap245 := d84
					snap246 := d130
					snap247 := d131
					snap248 := d132
					snap249 := d133
					snap250 := d134
					snap251 := d136
					snap252 := d138
					snap253 := d139
					snap254 := d142
					snap255 := d197
					alloc256 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps201)
					}
					ctx.RestoreAllocState(alloc256)
					d1 = snap202
					d2 = snap203
					d3 = snap204
					d4 = snap205
					d5 = snap206
					d6 = snap207
					d7 = snap208
					d9 = snap209
					d13 = snap210
					d24 = snap211
					d25 = snap212
					d26 = snap213
					d29 = snap214
					d32 = snap215
					d48 = snap216
					d50 = snap217
					d51 = snap218
					d52 = snap219
					d53 = snap220
					d54 = snap221
					d55 = snap222
					d56 = snap223
					d57 = snap224
					d58 = snap225
					d59 = snap226
					d60 = snap227
					d61 = snap228
					d63 = snap229
					d64 = snap230
					d66 = snap231
					d67 = snap232
					d68 = snap233
					d69 = snap234
					d70 = snap235
					d71 = snap236
					d72 = snap237
					d73 = snap238
					d74 = snap239
					d75 = snap240
					d76 = snap241
					d77 = snap242
					d78 = snap243
					d81 = snap244
					d84 = snap245
					d130 = snap246
					d131 = snap247
					d132 = snap248
					d133 = snap249
					d134 = snap250
					d136 = snap251
					d138 = snap252
					d139 = snap253
					d142 = snap254
					d197 = snap255
					if !bbs[7].Rendered {
						return bbs[7].RenderPS(ps200)
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[9].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[9].VisitCount >= 0 {
							ps.General = true
							return bbs[9].RenderPS(ps)
						}
					}
					bbs[9].VisitCount++
					if ps.General {
						if bbs[9].Rendered {
							ctx.EmitJmp(lbl10)
							return result
						}
						bbs[9].Rendered = true
						bbs[9].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_9 = bbs[9].Address
						ctx.MarkLabel(lbl10)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
						d130 = ps.OverlayValues[130]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
						d132 = ps.OverlayValues[132]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != LocNone {
						d197 = ps.OverlayValues[197]
					}
					ctx.ReclaimUntrackedRegs()
					d257 = args[1]
					d257.ID = 0
					d259 = d257
					ctx.EnsureDesc(&d259)
					if d259.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d259.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d259)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d259)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d259)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d259.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d259 = tmpPair
					} else if d259.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d259.Reg), Reg2: ctx.AllocRegExcept(d259.Reg)}
						switch d259.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d259)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d259)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d259)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d259)
						d259 = tmpPair
					} else if d259.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d259.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d259.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
						switch tmpScalar.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, tmpScalar)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, tmpScalar)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, tmpScalar)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&tmpScalar)
						d259 = tmpPair
					}
					if d259.Loc != LocRegPair && d259.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d258 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d259}, 2)
					ctx.StabilizeDescForControlFlow(&d258)
					ctx.FreeDesc(&d257)
					if ps.General {
						ctx.SyncDesc(&d258)
						if d258.Loc == LocReg {
							ctx.ProtectReg(d258.Reg)
						} else if d258.Loc == LocRegPair {
							ctx.ProtectReg(d258.Reg)
							ctx.ProtectReg(d258.Reg2)
						}
						d260 = d258
						if d260.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d260)
						if d260.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d260, int32(bbs[10].PhiBase)+int32(0), 2)
						} else if d260.Loc == LocInputPair {
							ctx.EnsureDesc(&d260)
							ctx.EmitStoreScmerToStack(d260, int32(bbs[10].PhiBase)+int32(0))
						} else if d260.Loc == LocRegPair || d260.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d260, int32(bbs[10].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d260)
							ctx.EmitStoreToStack(d260, int32(bbs[10].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[10].PhiBase)+int32(0))+8)
						}
						if d258.Loc == LocReg {
							ctx.UnprotectReg(d258.Reg)
						} else if d258.Loc == LocRegPair {
							ctx.UnprotectReg(d258.Reg)
							ctx.UnprotectReg(d258.Reg2)
						}
					}
					ps261 := PhiState{General: ps.General}
					ps261.OverlayValues = make([]JITValueDesc, 261)
					ps261.OverlayValues[1] = d1
					ps261.OverlayValues[2] = d2
					ps261.OverlayValues[3] = d3
					ps261.OverlayValues[4] = d4
					ps261.OverlayValues[5] = d5
					ps261.OverlayValues[6] = d6
					ps261.OverlayValues[7] = d7
					ps261.OverlayValues[9] = d9
					ps261.OverlayValues[13] = d13
					ps261.OverlayValues[24] = d24
					ps261.OverlayValues[25] = d25
					ps261.OverlayValues[26] = d26
					ps261.OverlayValues[29] = d29
					ps261.OverlayValues[32] = d32
					ps261.OverlayValues[48] = d48
					ps261.OverlayValues[50] = d50
					ps261.OverlayValues[51] = d51
					ps261.OverlayValues[52] = d52
					ps261.OverlayValues[53] = d53
					ps261.OverlayValues[54] = d54
					ps261.OverlayValues[55] = d55
					ps261.OverlayValues[56] = d56
					ps261.OverlayValues[57] = d57
					ps261.OverlayValues[58] = d58
					ps261.OverlayValues[59] = d59
					ps261.OverlayValues[60] = d60
					ps261.OverlayValues[61] = d61
					ps261.OverlayValues[63] = d63
					ps261.OverlayValues[64] = d64
					ps261.OverlayValues[66] = d66
					ps261.OverlayValues[67] = d67
					ps261.OverlayValues[68] = d68
					ps261.OverlayValues[69] = d69
					ps261.OverlayValues[70] = d70
					ps261.OverlayValues[71] = d71
					ps261.OverlayValues[72] = d72
					ps261.OverlayValues[73] = d73
					ps261.OverlayValues[74] = d74
					ps261.OverlayValues[75] = d75
					ps261.OverlayValues[76] = d76
					ps261.OverlayValues[77] = d77
					ps261.OverlayValues[78] = d78
					ps261.OverlayValues[81] = d81
					ps261.OverlayValues[84] = d84
					ps261.OverlayValues[130] = d130
					ps261.OverlayValues[131] = d131
					ps261.OverlayValues[132] = d132
					ps261.OverlayValues[133] = d133
					ps261.OverlayValues[134] = d134
					ps261.OverlayValues[136] = d136
					ps261.OverlayValues[138] = d138
					ps261.OverlayValues[139] = d139
					ps261.OverlayValues[142] = d142
					ps261.OverlayValues[197] = d197
					ps261.OverlayValues[257] = d257
					ps261.OverlayValues[258] = d258
					ps261.OverlayValues[259] = d259
					ps261.OverlayValues[260] = d260
					ps261.PhiValues = make([]JITValueDesc, 1)
					d262 = d258
					ps261.PhiValues[0] = d262
					if ps261.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps261)
					return result
				}
				bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d263 := ps.PhiValues[0]
							ctx.EnsureDesc(&d263)
							ctx.EmitStoreScmerToStack(d263, int32(bbs[10].PhiBase)+int32(0))
						}
						if bbs[10].VisitCount >= 0 {
							ps.General = true
							return bbs[10].RenderPS(ps)
						}
					}
					bbs[10].VisitCount++
					if ps.General {
						if bbs[10].Rendered {
							ctx.EmitJmp(lbl11)
							return result
						}
						bbs[10].Rendered = true
						bbs[10].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_10 = bbs[10].Address
						ctx.MarkLabel(lbl11)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
						d130 = ps.OverlayValues[130]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
						d132 = ps.OverlayValues[132]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != LocNone {
						d197 = ps.OverlayValues[197]
					}
					if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != LocNone {
						d257 = ps.OverlayValues[257]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					if len(ps.OverlayValues) > 259 && ps.OverlayValues[259].Loc != LocNone {
						d259 = ps.OverlayValues[259]
					}
					if len(ps.OverlayValues) > 260 && ps.OverlayValues[260].Loc != LocNone {
						d260 = ps.OverlayValues[260]
					}
					if len(ps.OverlayValues) > 262 && ps.OverlayValues[262].Loc != LocNone {
						d262 = ps.OverlayValues[262]
					}
					if len(ps.OverlayValues) > 263 && ps.OverlayValues[263].Loc != LocNone {
						d263 = ps.OverlayValues[263]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d3 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					d264 = ctx.EmitGoCallScalar(GoFuncAddr(func() *[1]any { return new([1]any) }), nil, 1)
					d265 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.EnsureDesc(&d3)
					d266 = ctx.EmitGoCallScalar(GoFuncAddr(func(value string) any { return value }), []JITValueDesc{d3}, 2)
					ctx.FreeDesc(&d3)
					ctx.EnsureDesc(&d266)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[1]any, index int, value any) { dst[index] = value }), []JITValueDesc{d264, d265, d266})
					sliceResults267 := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *[1]any) []any { return value[0:1:1] }), []JITValueDesc{d264}, []uint8{3}, []uint8{1})
					d268 = sliceResults267[0]
					d269 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("warning: JIT fallback: %s\n")}
					ctx.EnsureDesc(&d269)
					if d269.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d269.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d269.Imm)
						ptrWord, _ := d269.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d269.Imm.String())))
						d269 = tmpPair
					} else if d269.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d269.Type, Reg: ctx.AllocRegExcept(d269.Reg), Reg2: ctx.AllocRegExcept(d269.Reg)}
						switch d269.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d269)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d269)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d269)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d269)
						d269 = tmpPair
					}
					if d269.Loc != LocRegPair && d269.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (fmt.Printf arg0)")
					}
					ctx.EnsureDesc(&d268)
					ctx.EnsureDesc(&d268)
					ctx.EnsureDesc(&d268)
					if d268.Loc != LocRegTriple && d268.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (fmt.Printf arg1)")
					}
					ctx.SyncDesc(&d269)
					ctx.SyncDesc(&d268)
					callResults270 := JITEmitGoCallResults(ctx, GoFuncAddr(fmt.Printf), []JITValueDesc{d269, d268}, []uint8{1, 2}, []uint8{0, 3})
					ctx.FreeDesc(&d269)
					d271 = callResults270[0]
					_ = d271
					d272 = callResults270[1]
					_ = d272
					if ps.General {
					}
					ps273 := PhiState{General: ps.General}
					ps273.OverlayValues = make([]JITValueDesc, 273)
					ps273.OverlayValues[1] = d1
					ps273.OverlayValues[2] = d2
					ps273.OverlayValues[3] = d3
					ps273.OverlayValues[4] = d4
					ps273.OverlayValues[5] = d5
					ps273.OverlayValues[6] = d6
					ps273.OverlayValues[7] = d7
					ps273.OverlayValues[9] = d9
					ps273.OverlayValues[13] = d13
					ps273.OverlayValues[24] = d24
					ps273.OverlayValues[25] = d25
					ps273.OverlayValues[26] = d26
					ps273.OverlayValues[29] = d29
					ps273.OverlayValues[32] = d32
					ps273.OverlayValues[48] = d48
					ps273.OverlayValues[50] = d50
					ps273.OverlayValues[51] = d51
					ps273.OverlayValues[52] = d52
					ps273.OverlayValues[53] = d53
					ps273.OverlayValues[54] = d54
					ps273.OverlayValues[55] = d55
					ps273.OverlayValues[56] = d56
					ps273.OverlayValues[57] = d57
					ps273.OverlayValues[58] = d58
					ps273.OverlayValues[59] = d59
					ps273.OverlayValues[60] = d60
					ps273.OverlayValues[61] = d61
					ps273.OverlayValues[63] = d63
					ps273.OverlayValues[64] = d64
					ps273.OverlayValues[66] = d66
					ps273.OverlayValues[67] = d67
					ps273.OverlayValues[68] = d68
					ps273.OverlayValues[69] = d69
					ps273.OverlayValues[70] = d70
					ps273.OverlayValues[71] = d71
					ps273.OverlayValues[72] = d72
					ps273.OverlayValues[73] = d73
					ps273.OverlayValues[74] = d74
					ps273.OverlayValues[75] = d75
					ps273.OverlayValues[76] = d76
					ps273.OverlayValues[77] = d77
					ps273.OverlayValues[78] = d78
					ps273.OverlayValues[81] = d81
					ps273.OverlayValues[84] = d84
					ps273.OverlayValues[130] = d130
					ps273.OverlayValues[131] = d131
					ps273.OverlayValues[132] = d132
					ps273.OverlayValues[133] = d133
					ps273.OverlayValues[134] = d134
					ps273.OverlayValues[136] = d136
					ps273.OverlayValues[138] = d138
					ps273.OverlayValues[139] = d139
					ps273.OverlayValues[142] = d142
					ps273.OverlayValues[197] = d197
					ps273.OverlayValues[257] = d257
					ps273.OverlayValues[258] = d258
					ps273.OverlayValues[259] = d259
					ps273.OverlayValues[260] = d260
					ps273.OverlayValues[262] = d262
					ps273.OverlayValues[263] = d263
					ps273.OverlayValues[264] = d264
					ps273.OverlayValues[265] = d265
					ps273.OverlayValues[266] = d266
					ps273.OverlayValues[268] = d268
					ps273.OverlayValues[269] = d269
					ps273.OverlayValues[271] = d271
					ps273.OverlayValues[272] = d272
					if ps273.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps273)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps274 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps274)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(48))
				return result
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

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["jit-enabled?"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d0.Loc == LocImm {
					ctx.EmitMakeBool(result, d0)
				} else {
					ctx.EmitMakeBool(result, d0)
					ctx.FreeReg(d0.Reg)
				}
				result.Type = tagBool
				return result
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
	if head, ok := scmerSymbol(items[0]); ok {
		switch string(head) {
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
