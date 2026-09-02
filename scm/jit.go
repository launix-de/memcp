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
	Native         func(...Scmer) Scmer // compiled native function pointer
	DebugName      string
	StackFrameSize int32
	// BoundArgs are lexical closure values appended to the public arguments.
	// Owner keeps the entry point which owns the shared machine code alive.
	BoundArgs []Scmer
	Owner     *JITEntryPoint
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
	// RecursiveLambdas makes lambda values constructed by this native body
	// compile their own body before they are returned or passed onward.
	RecursiveLambdas bool
	// Coverage counts lowered Scheme expressions and distinguishes generic
	// Eval/Apply bridges, compact native builtin calls, and inlined emitters.
	// It is diagnostic metadata, not a runtime profile.
	Coverage JITCoverage
}

type JITCoverage struct {
	Expressions  int
	DynamicCalls int
	NativeCalls  int
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
	stableArgs := jep.TransferInputArgs || len(jep.BoundArgs) != 0 || len(jep.HiddenArgs) != 0
	if stableArgs {
		args = append([]Scmer(nil), args...)
	}
	if jep.Proc.Params.GetTag() == tagSlice {
		paramCount := len(jep.Proc.Params.Slice()) - len(jep.BoundArgs)
		if paramCount < 0 {
			panic("JIT: invalid bound argument count")
		}
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
	args = append(args, jep.BoundArgs...)
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
	ensureJITStack(jep.StackFrameSize)
	result = jep.Native(args...)
	runtime.KeepAlive(args)
	runtime.KeepAlive(jep)
	runtime.KeepAlive(jep.Owner)
	return result
}

const jitStackProbeSize = 4096

// ensureJITStack grows the goroutine stack through ordinary Go frames before
// entering generated code. JIT frames are registered with the runtime and can
// be relocated at safepoints, but their generated prologue has no morestack
// call of its own.
//
//go:noinline
func ensureJITStack(frameSize int32) {
	if frameSize <= 0 {
		return
	}
	var probe [jitStackProbeSize]byte
	probe[0] = byte(frameSize)
	if frameSize > jitStackProbeSize {
		ensureJITStack(frameSize - jitStackProbeSize)
	}
	runtime.KeepAlive(&probe)
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
	// Parser carries compiler-only grammar shape. It is materialized only when
	// no native parser consumer can fuse the grammar into its surrounding code.
	Parser *JITParserTemplate
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

// JITLabel identifies an architecture-independent control-flow target. Labels
// are intentionally not byte-sized: generated parsers and other large native
// functions routinely contain more than 256 basic blocks.
type JITLabel uint32

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
	LocParserTemplate
)

// JITFixup records a forward reference that must be patched after all
// labels are placed.
type JITFixup struct {
	CodePos  int32    // position in code
	LabelID  JITLabel // target label
	Size     uint8    // 1=rel8, 4=rel32
	Relative bool     // true for PC-relative jumps
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

	Labels []int32
	Fixups []JITFixup

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
	HasFrame     bool
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
	RecursiveLambdas      bool
	ActiveBuiltinEmitters map[*Declaration]uint16
	BuiltinInlineCost     int
	NeedsStableArgs       bool
	StackPhiTargets       bool
	SelfSymbols           map[Symbol]struct{}
	DefiningSymbol        Symbol
	SelfLoopLabel         JITLabel
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

// TrackPointer retains a typed object whose address is embedded in generated
// machine code as a scalar. Unlike TrackImm, it also covers pointers carried
// through integer or register descriptors, whose Scmer representation cannot
// expose the referenced heap object to the garbage collector.
func (ctx *JITContext) TrackPointer(p unsafe.Pointer) {
	if ctx == nil || p == nil {
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

// EnsureDescsTogether materializes a set of operands without allowing a later
// reload to evict an earlier one. Generated binary operations use this at the
// final consumption point; otherwise two spilled values can repeatedly evict
// each other and leave both aliased descriptors naming the same register.
func (ctx *JITContext) EnsureDescsTogether(descs ...*JITValueDesc) {
	var protected [16]Reg
	protectedCount := 0
	for _, desc := range descs {
		ctx.EnsureDesc(desc)
		switch desc.Loc {
		case LocReg:
			ctx.ProtectReg(desc.Reg)
			protected[protectedCount] = desc.Reg
			protectedCount++
		case LocRegPair:
			ctx.ProtectReg(desc.Reg)
			protected[protectedCount] = desc.Reg
			protectedCount++
			ctx.ProtectReg(desc.Reg2)
			protected[protectedCount] = desc.Reg2
			protectedCount++
		case LocRegTriple:
			for _, reg := range [...]Reg{desc.Reg, desc.Reg2, desc.Reg3} {
				ctx.ProtectReg(reg)
				protected[protectedCount] = reg
				protectedCount++
			}
		}
	}
	for i := protectedCount - 1; i >= 0; i-- {
		ctx.UnprotectReg(protected[i])
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

// jitExpressionConsumesRuntimeEnv reports whether delayed syntax can resolve
// bindings which are not statically visible in its enclosing lambda body.
// Such lambdas must retain the complete lexical frame rather than only the
// symbols found by ordinary free-variable analysis.
func jitExpressionConsumesRuntimeEnv(expr Scmer) bool {
	for expr.IsSourceInfo() {
		expr = expr.SourceInfo().value
	}
	if !expr.IsSlice() {
		return false
	}
	items := expr.Slice()
	if len(items) == 0 {
		return false
	}
	if head, ok := scmerSymbol(items[0]); ok {
		switch string(head) {
		case "quote":
			return false
		case "eval", "parser":
			return true
		}
	}
	for _, item := range items {
		if jitExpressionConsumesRuntimeEnv(item) {
			return true
		}
	}
	return false
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

// jitBindLambdaCaptures turns lexical symbol and outer-slot reads into hidden
// native parameters. The closure builder binds those parameters once when the
// lambda value is created, so its machine code can be compiled with its
// enclosing procedure instead of invoking the compiler from the hot path.
func jitLambdaCaptureReference(index NthLocalVar, depth int) Scmer {
	local := NewNthLocalVar(index)
	if depth == 0 {
		return local
	}
	return NewSlice([]Scmer{NewSymbol("outer"), NewInt(int64(depth)), local})
}

func jitBindLambdaCaptures(expr Scmer, symbols map[Symbol]NthLocalVar, outerVars map[NthLocalVar]NthLocalVar) Scmer {
	return jitBindLambdaCapturesAtDepth(expr, symbols, outerVars, 0)
}

// jitBindLambdaSelfValues routes a named recursive closure used as a value
// through a hidden bound parameter. Direct calls in the procedure itself keep
// their symbolic head so the emitter can lower them to a loop or native
// recursion. A reference passed as a callback must instead name the concrete
// closure, because the shared template's lexical environment has no per-
// instance BoundArgs.
func jitBindLambdaSelfValues(expr Scmer, self Symbol, param NthLocalVar) Scmer {
	return jitBindLambdaSelfValuesAtDepth(expr, self, param, 0)
}

func jitBindLambdaSelfValuesAtDepth(expr Scmer, self Symbol, param NthLocalVar, depth int) Scmer {
	if expr.IsSourceInfo() {
		source := *expr.SourceInfo()
		source.value = jitBindLambdaSelfValuesAtDepth(source.value, self, param, depth)
		return NewSourceInfo(source)
	}
	if expr.IsSymbol() {
		if expr.Symbol() == self {
			return jitLambdaCaptureReference(param, depth)
		}
		return expr
	}
	if !expr.IsSlice() {
		return expr
	}
	items := expr.Slice()
	if len(items) == 0 {
		return expr
	}
	head, hasHead := scmerSymbol(items[0])
	if hasHead && string(head) == "quote" {
		return expr
	}
	if hasHead && string(head) == "lambda" && len(items) >= 3 {
		boundParams := make(map[Symbol]struct{})
		jitAddLambdaBoundParams(items[1], boundParams)
		if _, shadowed := boundParams[self]; shadowed {
			return expr
		}
		bound := append([]Scmer(nil), items...)
		bound[2] = jitBindLambdaSelfValuesAtDepth(items[2], self, param, depth+1)
		return NewSlice(bound)
	}
	bound := make([]Scmer, len(items))
	for index, item := range items {
		// Only a direct call in the recursive procedure itself can use the
		// native self-call lowering. Nested lambdas require the bound closure.
		if index == 0 && depth == 0 && hasHead && head == self {
			bound[index] = item
			continue
		}
		bound[index] = jitBindLambdaSelfValuesAtDepth(item, self, param, depth)
	}
	return NewSlice(bound)
}

func jitBindLambdaCapturesAtDepth(expr Scmer, symbols map[Symbol]NthLocalVar, outerVars map[NthLocalVar]NthLocalVar, depth int) Scmer {
	if expr.IsSourceInfo() {
		source := *expr.SourceInfo()
		source.value = jitBindLambdaCapturesAtDepth(source.value, symbols, outerVars, depth)
		return NewSourceInfo(source)
	}
	if expr.IsSymbol() {
		if param, exists := symbols[expr.Symbol()]; exists {
			return jitLambdaCaptureReference(param, depth)
		}
		return expr
	}
	if !expr.IsSlice() {
		return expr
	}
	items := expr.Slice()
	if len(items) == 0 {
		return expr
	}
	if head, ok := scmerSymbol(items[0]); ok {
		if string(head) == "quote" {
			return expr
		}
		if string(head) == "outer" && len(items) == 3 {
			key := items[2].WithoutSourceInfo()
			if key.IsNthLocalVar() {
				if param, exists := outerVars[key.NthLocalVar()]; exists {
					return jitLambdaCaptureReference(param, depth)
				}
			}
		}
		if string(head) == "lambda" && len(items) >= 3 {
			bound := append([]Scmer(nil), items...)
			innerSymbols := symbols
			if len(symbols) != 0 {
				innerSymbols = make(map[Symbol]NthLocalVar, len(symbols))
				for symbol, index := range symbols {
					innerSymbols[symbol] = index
				}
				params := make(map[Symbol]struct{})
				jitAddLambdaBoundParams(items[1], params)
				for symbol := range params {
					delete(innerSymbols, symbol)
				}
			}
			bound[2] = jitBindLambdaCapturesAtDepth(items[2], innerSymbols, outerVars, depth+1)
			return NewSlice(bound)
		}
		if (string(head) == "define" || string(head) == "set" || string(head) == "setN") && len(items) == 3 {
			bound := append([]Scmer(nil), items...)
			bound[2] = jitBindLambdaCapturesAtDepth(items[2], symbols, outerVars, depth)
			return NewSlice(bound)
		}
	}
	changed := false
	bound := make([]Scmer, len(items))
	for index, item := range items {
		bound[index] = jitBindLambdaCapturesAtDepth(item, symbols, outerVars, depth)
		changed = changed || bound[index] != item
	}
	if !changed {
		return expr
	}
	return NewSlice(bound)
}

func jitBindCompiledLambdaEntry(template Scmer, closure Scmer, captureArgs []Scmer) Scmer {
	if !template.IsProc() || template.Proc() == nil || template.Proc().Compiled == nil {
		panic("jit: invalid compiled lambda template")
	}
	proc := closure.Proc()
	if proc == nil {
		panic("jit: invalid bound lambda closure")
	}
	bound := make([]Scmer, 0, len(captureArgs)/2)
	for index := 0; index < len(captureArgs); index += 2 {
		if index+1 >= len(captureArgs) {
			panic("jit: invalid lambda capture")
		}
		bound = append(bound, captureArgs[index+1])
	}
	if template.Proc().Params.IsSlice() && proc.Params.IsSlice() {
		padding := len(template.Proc().Params.Slice()) - len(proc.Params.Slice()) - len(bound)
		if padding < 0 {
			panic("jit: invalid compiled lambda capture count")
		}
		if padding != 0 {
			padded := make([]Scmer, padding, padding+len(bound))
			for index := range padded {
				padded[index] = NewNil()
			}
			bound = append(padded, bound...)
		}
	}
	entry := *template.Proc().Compiled
	entry.BoundArgs = bound
	entry.Owner = template.Proc().Compiled
	entry.Arena = nil
	proc.Compiled = &entry
	return closure
}

func jitBuildBoundCompiledLambdaClosure(args ...Scmer) Scmer {
	if len(args) < 4 {
		panic("jit: bound lambda builder expects template and closure")
	}
	return jitBindCompiledLambdaEntry(args[0], jitBuildLambdaClosure(args[1:]...), args[4:])
}

func jitBuildNamedBoundCompiledLambdaClosure(args ...Scmer) Scmer {
	if len(args) < 5 {
		panic("jit: named bound lambda builder expects template and closure")
	}
	closure := jitBuildNamedLambdaClosure(args[1:]...)
	captures := args[5:]
	if len(captures) >= 2 && captures[len(captures)-2].SymbolEquals("\x00jit-bound-self") {
		captures = append([]Scmer(nil), captures...)
		captures[len(captures)-1] = closure
	}
	return jitBindCompiledLambdaEntry(args[0], closure, captures)
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

// jitBuildCompiledLambdaClosureWithRuntimeEnv preserves the exact lexical
// frame chain for the uncommon case where recursive compilation cannot lower
// an inner lambda. Compiled closures read their inputs directly; an interpreter
// fallback still needs the original outer depths to remain meaningful.
func jitBuildCompiledLambdaClosureWithRuntimeEnv(args ...Scmer) Scmer {
	if len(args) < 4 {
		panic("jit: runtime-bound lambda builder expects params, body, numVars and environment")
	}
	value := NewProcStruct(Proc{
		Params:  args[0],
		Body:    args[1],
		En:      jitRuntimeEnvFromCaptures(args[3:]),
		NumVars: int(ToInt(args[2])),
	})
	return jitCompile(value)
}

// jitBuildNamedCompiledLambdaClosure makes a define-bound lambda visible in
// its own capture environment before compilation. This gives the compiler a
// stable self identity for direct recursive calls without a placeholder value
// or a later mutation of the Scheme environment.
func jitBuildNamedLambdaClosure(args ...Scmer) Scmer {
	if len(args) < 4 {
		panic("jit: named lambda builder expects name, params, body and numVars")
	}
	name := mustSymbol(args[0])
	value := jitBuildLambdaClosure(args[1:]...)
	proc := value.Proc()
	if proc == nil || proc.En == nil {
		panic("jit: named lambda builder produced no procedure environment")
	}
	if proc.En.Vars == nil {
		proc.En.Vars = make(Vars, 1)
	}
	proc.En.Vars[name] = value
	return value
}

func jitBuildNamedCompiledLambdaClosure(args ...Scmer) Scmer {
	return jitCompile(jitBuildNamedLambdaClosure(args...))
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
		EmitTracePrint(message)
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
	return ctx.emitGoCall(funcAddr, argWords, numResultWords, resultsBuf, resultTargets, 0, nil)
}

func (ctx *JITContext) EmitGoCallToStack(funcAddr uint64, argWords []goCallArgWord, resultStackOffs []int32) {
	var resultsBuf [16]Reg
	ctx.emitGoCall(funcAddr, argWords, len(resultStackOffs), &resultsBuf, nil, ctx.StackReg, resultStackOffs)
}

// EmitGoCallToFrame emits a Go call whose result words are written directly
// into invocation-local RBP-relative spill slots. This lets producers feed a
// stack-backed consumer without first consuming allocator registers.
func (ctx *JITContext) EmitGoCallToFrame(funcAddr uint64, argWords []goCallArgWord, resultFrameOffs []int32) {
	var resultsBuf [16]Reg
	ctx.emitGoCall(funcAddr, argWords, len(resultFrameOffs), &resultsBuf, nil, ctx.FrameReg, resultFrameOffs)
}

func (ctx *JITContext) emitGoCall(funcAddr uint64, argWords []goCallArgWord, numResultWords int, resultsBuf *[16]Reg, resultTargets []Reg, resultSlotBase Reg, resultSlotOffs []int32) []Reg {
	ctx.NeedsStableArgs = true
	entryDynamicSP := ctx.DynamicSP
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
					ctx.EmitMovRegMem(RegR11, RegRSP, stackArgBaseDisp+entryDynamicSP+argWords[i].stackOff)
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
					ctx.EmitMovRegMem(target, RegRSP, stackArgBaseDisp+entryDynamicSP+argWords[i].stackOff)
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
		if resultSlotOffs != nil {
			for i, off := range resultSlotOffs {
				if resultSlotBase == ctx.StackReg {
					off += entryDynamicSP
				}
				ctx.EmitStoreRegMem(GoABIIntRegs[i], resultSlotBase, off)
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
	if resultSlotOffs != nil {
		for i, off := range resultSlotOffs {
			ctx.EmitMovRegMem(RegR11, RegRSP, int32(i*8))
			if resultSlotBase == ctx.StackReg {
				off += int32(resultBytes) + entryDynamicSP
			}
			ctx.EmitStoreRegMem(RegR11, resultSlotBase, off)
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
	for index := range args {
		ctx.SyncDesc(&args[index])
		a := args[index]
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
			valueType := a.Type
			// A zero-value descriptor historically denotes an untyped immediate.
			// Recover its scalar kind from the immutable Scmer payload instead of
			// mistaking every such value (notably constant receiver pointers) for
			// nil. Explicit non-nil type information still takes precedence.
			if valueType == JITTypeUnknown || (valueType == tagNil && !a.Imm.IsNil()) {
				valueType = a.Imm.GetTag()
			}
			switch valueType {
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
				panic(fmt.Sprintf("jit: LocImm scalar Go-call arg requires explicit materialization (type=%d, tag=%d)", valueType, a.Imm.GetTag()))
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
	var result JITValueDesc
	if numResultWords == 1 {
		result = JITValueDesc{Loc: LocReg, Reg: results[0]}
	} else if numResultWords == 3 {
		result = JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: results[0], Reg2: results[1], Reg3: results[2]}
	} else {
		result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: results[0], Reg2: results[1]}
	}
	// A result becomes spillable immediately. Generated emitters may call
	// ReclaimUntrackedRegs before its next use, so leaving ABI result registers
	// ownerless would silently release a still-live slice or Scmer value.
	for _, reg := range jitDescRegs(result) {
		ctx.BindReg(reg, &result)
	}
	return result
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

// EmitMovPairToResult moves a Scmer pair into the result descriptor registers.
// Stack-backed producers load directly into their requested destination and do
// not consume an intermediate register pair.
func (ctx *JITContext) EmitMovPairToResult(src *JITValueDesc, dst *JITValueDesc) {
	ctx.SyncDesc(src)
	if src.Loc == LocStackPair {
		base := ctx.StackReg
		if src.StackOff < 0 {
			base = ctx.FrameReg
		}
		ctx.EmitMovRegMem(dst.Reg, base, src.StackOff)
		ctx.EmitMovRegMem(dst.Reg2, base, src.StackOff+8)
		return
	}
	if src.Loc == LocInputPair {
		base := ctx.SliceBase
		if ctx.SliceBaseTracksRSP && int(src.StackOff) >= ctx.InputArgCount {
			base = ctx.ScratchReg
			ctx.EmitMovRegMem(base, RegRSP, ctx.OriginalArgsOff)
		}
		ctx.EmitMovRegMem(dst.Reg, base, src.StackOff*16)
		ctx.EmitMovRegMem(dst.Reg2, base, src.StackOff*16+8)
		return
	}
	if src.Loc != LocRegPair {
		panic("jit: pair move requires a register, stack, or input pair")
	}
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
func (ctx *JITContext) ReserveLabel() JITLabel {
	id := JITLabel(len(ctx.Labels))
	ctx.Labels = append(ctx.Labels, -1)
	return id
}

// MarkLabel sets the position of a previously reserved label.
func (ctx *JITContext) MarkLabel(id JITLabel) {
	if int(id) >= len(ctx.Labels) {
		panic("jit: invalid label")
	}
	ctx.Labels[id] = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
}

// AddFixup records a forward reference to be patched by ResolveFixups.
func (ctx *JITContext) AddFixup(labelID JITLabel, size uint8, relative bool) {
	ctx.Fixups = append(ctx.Fixups, JITFixup{
		CodePos:  int32(uintptr(ctx.Ptr) - uintptr(ctx.Start)),
		LabelID:  labelID,
		Size:     size,
		Relative: relative,
	})
}

// ResolveFixups patches recorded forward references whose labels are defined.
// Fixups referencing still-undefined labels are kept for a later call.
func (ctx *JITContext) ResolveFixups() {
	pending := ctx.Fixups[:0]
	for i := range ctx.Fixups {
		f := &ctx.Fixups[i]
		targetPos := ctx.Labels[f.LabelID]
		if targetPos < 0 {
			// label not yet defined — keep for later
			pending = append(pending, *f)
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
	ctx.Fixups = pending
}

// ResolveFixupsFinal patches all remaining fixups, panicking on undefined labels.
func (ctx *JITContext) ResolveFixupsFinal() {
	for i := range ctx.Fixups {
		f := &ctx.Fixups[i]
		targetPos := ctx.Labels[f.LabelID]
		if targetPos < 0 {
			panic(fmt.Sprintf("jit: undefined label %d referenced at code offset %d", f.LabelID, f.CodePos))
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
	ctx.Fixups = ctx.Fixups[:0]
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
				lbl0 := ctx.ReserveLabel()
				_ = lbl0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d1)
				d2 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
				d3 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
				d4 := d1
				_ = d4
				ctx.StabilizeDescForControlFlow(&d4)
				d5 := d2
				_ = d5
				ctx.StabilizeDescForControlFlow(&d5)
				d6 := d3
				_ = d6
				ctx.StabilizeDescForControlFlow(&d6)
				phiBase7 := ctx.AllocStack(int32(16))
				d8 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				_ = d8
				lbl1 := ctx.ReserveLabel()
				bbpos_2_0 := int32(-1)
				_ = bbpos_2_0
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_2_1 := int32(-1)
				_ = bbpos_2_1
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_2_2 := int32(-1)
				_ = bbpos_2_2
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_2_3 := int32(-1)
				_ = bbpos_2_3
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_2_4 := int32(-1)
				_ = bbpos_2_4
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_2_5 := int32(-1)
				_ = bbpos_2_5
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_2_6 := int32(-1)
				_ = bbpos_2_6
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbpos_2_7 := int32(-1)
				_ = bbpos_2_7
				lbl9 := ctx.ReserveLabel()
				_ = lbl9
				bbpos_2_8 := int32(-1)
				_ = bbpos_2_8
				lbl10 := ctx.ReserveLabel()
				_ = lbl10
				bbpos_2_9 := int32(-1)
				_ = bbpos_2_9
				lbl11 := ctx.ReserveLabel()
				_ = lbl11
				bbpos_2_10 := int32(-1)
				_ = bbpos_2_10
				lbl12 := ctx.ReserveLabel()
				_ = lbl12
				bbpos_2_11 := int32(-1)
				_ = bbpos_2_11
				lbl13 := ctx.ReserveLabel()
				_ = lbl13
				bbpos_2_12 := int32(-1)
				_ = bbpos_2_12
				lbl14 := ctx.ReserveLabel()
				_ = lbl14
				bbpos_2_13 := int32(-1)
				_ = bbpos_2_13
				lbl15 := ctx.ReserveLabel()
				_ = lbl15
				bbpos_2_14 := int32(-1)
				_ = bbpos_2_14
				lbl16 := ctx.ReserveLabel()
				_ = lbl16
				bbpos_2_15 := int32(-1)
				_ = bbpos_2_15
				lbl17 := ctx.ReserveLabel()
				_ = lbl17
				bbpos_2_16 := int32(-1)
				_ = bbpos_2_16
				lbl18 := ctx.ReserveLabel()
				_ = lbl18
				bbpos_2_17 := int32(-1)
				_ = bbpos_2_17
				lbl19 := ctx.ReserveLabel()
				_ = lbl19
				bbpos_2_18 := int32(-1)
				_ = bbpos_2_18
				lbl20 := ctx.ReserveLabel()
				_ = lbl20
				bbpos_2_19 := int32(-1)
				_ = bbpos_2_19
				lbl21 := ctx.ReserveLabel()
				_ = lbl21
				bbpos_2_20 := int32(-1)
				_ = bbpos_2_20
				lbl22 := ctx.ReserveLabel()
				_ = lbl22
				bbpos_2_21 := int32(-1)
				_ = bbpos_2_21
				lbl23 := ctx.ReserveLabel()
				_ = lbl23
				bbpos_2_22 := int32(-1)
				_ = bbpos_2_22
				lbl24 := ctx.ReserveLabel()
				_ = lbl24
				bbpos_2_23 := int32(-1)
				_ = bbpos_2_23
				lbl25 := ctx.ReserveLabel()
				_ = lbl25
				bbpos_2_24 := int32(-1)
				_ = bbpos_2_24
				lbl26 := ctx.ReserveLabel()
				_ = lbl26
				bbpos_2_25 := int32(-1)
				_ = bbpos_2_25
				lbl27 := ctx.ReserveLabel()
				_ = lbl27
				bbpos_2_26 := int32(-1)
				_ = bbpos_2_26
				lbl28 := ctx.ReserveLabel()
				_ = lbl28
				bbpos_2_27 := int32(-1)
				_ = bbpos_2_27
				lbl29 := ctx.ReserveLabel()
				_ = lbl29
				bbpos_2_28 := int32(-1)
				_ = bbpos_2_28
				lbl30 := ctx.ReserveLabel()
				_ = lbl30
				bbpos_2_29 := int32(-1)
				_ = bbpos_2_29
				lbl31 := ctx.ReserveLabel()
				_ = lbl31
				bbpos_2_30 := int32(-1)
				_ = bbpos_2_30
				lbl32 := ctx.ReserveLabel()
				_ = lbl32
				bbpos_2_31 := int32(-1)
				_ = bbpos_2_31
				lbl33 := ctx.ReserveLabel()
				_ = lbl33
				bbpos_2_32 := int32(-1)
				_ = bbpos_2_32
				lbl34 := ctx.ReserveLabel()
				_ = lbl34
				bbpos_2_33 := int32(-1)
				_ = bbpos_2_33
				lbl35 := ctx.ReserveLabel()
				_ = lbl35
				bbpos_2_34 := int32(-1)
				_ = bbpos_2_34
				lbl36 := ctx.ReserveLabel()
				_ = lbl36
				bbpos_2_35 := int32(-1)
				_ = bbpos_2_35
				lbl37 := ctx.ReserveLabel()
				_ = lbl37
				bbpos_2_36 := int32(-1)
				_ = bbpos_2_36
				lbl38 := ctx.ReserveLabel()
				_ = lbl38
				bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl2)
				ctx.ResolveFixups()
				d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d9 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d9)
				var d10 JITValueDesc
				if d9.Loc == LocImm {
					d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d9.Imm.Int() != 1)}
				} else {
					r0 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d9.Reg, 1)
					ctx.EmitSetcc(r0, CondNotEqual)
					d10 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
					ctx.BindReg(r0, &d10)
				}
				ctx.FreeDesc(&d9)
				ctx.ReclaimUntrackedRegs()
				d11 := d10
				ctx.EnsureDesc(&d11)
				if d11.Loc != LocImm && d11.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl39 := ctx.ReserveLabel()
				lbl40 := ctx.ReserveLabel()
				if d11.Loc == LocImm {
					if d11.Imm.Bool() {
						ctx.MarkLabel(lbl39)
						ctx.EmitJmp(lbl3)
					} else {
						ctx.MarkLabel(lbl40)
						ctx.EmitJmp(lbl4)
					}
				} else {
					ctx.EmitCmpRegImm32(d11.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl39)
					ctx.EmitJmp(lbl40)
					ctx.MarkLabel(lbl39)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl40)
					ctx.EmitJmp(lbl4)
				}
				ctx.FreeDesc(&d10)
				bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl4)
				ctx.ResolveFixups()
				d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d12 := args[0]
				d12.ID = 0
				ctx.StabilizeDescForControlFlow(&d12)
				ctx.ReclaimUntrackedRegs()
				d13 := ctx.EmitGetTagDesc(&d12, JITValueDesc{Loc: LocAny})
				ctx.StabilizeDescForControlFlow(&d13)
				ctx.ReclaimUntrackedRegs()
				bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl5)
				ctx.ResolveFixups()
				d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d13)
				var d14 JITValueDesc
				if d13.Loc == LocImm {
					d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d13.Imm.Int()) == uint64(0xb))}
				} else {
					r1 := ctx.AllocRegExcept(d13.Reg)
					ctx.EmitCmpRegImm32(d13.Reg, 11)
					ctx.EmitSetcc(r1, CondEqual)
					d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
					ctx.BindReg(r1, &d14)
				}
				ctx.ReclaimUntrackedRegs()
				d15 := d14
				ctx.EnsureDesc(&d15)
				if d15.Loc != LocImm && d15.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl41 := ctx.ReserveLabel()
				lbl42 := ctx.ReserveLabel()
				if d15.Loc == LocImm {
					if d15.Imm.Bool() {
						ctx.MarkLabel(lbl41)
						ctx.EmitJmp(lbl7)
					} else {
						ctx.MarkLabel(lbl42)
						ctx.EmitJmp(lbl8)
					}
				} else {
					ctx.EmitCmpRegImm32(d15.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl41)
					ctx.EmitJmp(lbl42)
					ctx.MarkLabel(lbl41)
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl42)
					ctx.EmitJmp(lbl8)
				}
				ctx.FreeDesc(&d14)
				bbpos_2_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl8)
				ctx.ResolveFixups()
				d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d13)
				var d16 JITValueDesc
				if d13.Loc == LocImm {
					d16 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d13.Imm.Int()) == uint64(0x8))}
				} else {
					r2 := ctx.AllocRegExcept(d13.Reg)
					ctx.EmitCmpRegImm32(d13.Reg, 8)
					ctx.EmitSetcc(r2, CondEqual)
					d16 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
					ctx.BindReg(r2, &d16)
				}
				ctx.ReclaimUntrackedRegs()
				d17 := d16
				ctx.EnsureDesc(&d17)
				if d17.Loc != LocImm && d17.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl43 := ctx.ReserveLabel()
				lbl44 := ctx.ReserveLabel()
				if d17.Loc == LocImm {
					if d17.Imm.Bool() {
						ctx.MarkLabel(lbl43)
						ctx.EmitJmp(lbl7)
					} else {
						ctx.MarkLabel(lbl44)
						ctx.EmitJmp(lbl9)
					}
				} else {
					ctx.EmitCmpRegImm32(d17.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl43)
					ctx.EmitJmp(lbl44)
					ctx.MarkLabel(lbl43)
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl44)
					ctx.EmitJmp(lbl9)
				}
				ctx.FreeDesc(&d16)
				bbpos_2_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl9)
				ctx.ResolveFixups()
				d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d13)
				var d18 JITValueDesc
				if d13.Loc == LocImm {
					d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d13.Imm.Int()) == uint64(0x9))}
				} else {
					r3 := ctx.AllocRegExcept(d13.Reg)
					ctx.EmitCmpRegImm32(d13.Reg, 9)
					ctx.EmitSetcc(r3, CondEqual)
					d18 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
					ctx.BindReg(r3, &d18)
				}
				ctx.ReclaimUntrackedRegs()
				d19 := d18
				ctx.EnsureDesc(&d19)
				if d19.Loc != LocImm && d19.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl45 := ctx.ReserveLabel()
				lbl46 := ctx.ReserveLabel()
				if d19.Loc == LocImm {
					if d19.Imm.Bool() {
						ctx.MarkLabel(lbl45)
						ctx.EmitJmp(lbl7)
					} else {
						ctx.MarkLabel(lbl46)
						ctx.EmitJmp(lbl10)
					}
				} else {
					ctx.EmitCmpRegImm32(d19.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl45)
					ctx.EmitJmp(lbl46)
					ctx.MarkLabel(lbl45)
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl46)
					ctx.EmitJmp(lbl10)
				}
				ctx.FreeDesc(&d18)
				bbpos_2_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl10)
				ctx.ResolveFixups()
				d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d13)
				var d20 JITValueDesc
				if d13.Loc == LocImm {
					d20 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d13.Imm.Int()) == uint64(0xa))}
				} else {
					r4 := ctx.AllocRegExcept(d13.Reg)
					ctx.EmitCmpRegImm32(d13.Reg, 10)
					ctx.EmitSetcc(r4, CondEqual)
					d20 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
					ctx.BindReg(r4, &d20)
				}
				ctx.ReclaimUntrackedRegs()
				d21 := d20
				ctx.EnsureDesc(&d21)
				if d21.Loc != LocImm && d21.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl47 := ctx.ReserveLabel()
				lbl48 := ctx.ReserveLabel()
				if d21.Loc == LocImm {
					if d21.Imm.Bool() {
						ctx.MarkLabel(lbl47)
						ctx.EmitJmp(lbl7)
					} else {
						ctx.MarkLabel(lbl48)
						ctx.EmitJmp(lbl11)
					}
				} else {
					ctx.EmitCmpRegImm32(d21.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl47)
					ctx.EmitJmp(lbl48)
					ctx.MarkLabel(lbl47)
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl48)
					ctx.EmitJmp(lbl11)
				}
				ctx.FreeDesc(&d20)
				bbpos_2_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl11)
				ctx.ResolveFixups()
				d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
				bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
				d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
				bbpos_2_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl7)
				ctx.ResolveFixups()
				d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase7) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				r5 := ctx.AllocReg()
				r6 := ctx.AllocRegExcept(r5)
				d22 := JITValueDesc{Loc: LocRegPair, Reg: r5, Reg2: r6}
				ctx.BindReg(r5, &d22)
				ctx.BindReg(r6, &d22)
				ctx.EmitMovPairToResult(&d12, &d22)
				ctx.EmitJmp(lbl1)
				ctx.MarkLabel(lbl1)
				d23 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r5, Reg2: r6}
				ctx.BindReg(r5, &d23)
				ctx.BindReg(r6, &d23)
				ctx.BindReg(r5, &d23)
				ctx.BindReg(r6, &d23)
				ctx.FreeDesc(&d1)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d23)
				if d23.Loc == LocImm {
					if result.Loc == LocAny {
						return d23
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.SyncDesc(&d23)
				if d23.Loc == LocRegPair || d23.Loc == LocStackPair || d23.Loc == LocInputPair {
					ctx.EmitMovPairToResult(&d23, &result)
					result.Type = d23.Type
				} else {
					switch d23.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d23)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d23)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d23)
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
			JITInlineCost:  202,
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
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
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
				var d64 JITValueDesc
				_ = d64
				var d67 JITValueDesc
				_ = d67
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
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
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
					ctx.EnsureDesc(&d49)
					d49 = JITPrepareScmerGoArg(ctx, d49)
					ctx.SyncDesc(&d49)
					d50 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Proc), []JITValueDesc{d49}, 1)
					d50.NoHeapPointer = false
					ctx.BindReg(d50.Reg, &d50)
					ctx.FreeDesc(&d49)
					var d51 JITValueDesc
					ctx.EnsureDesc(&d50)
					if d50.Loc == LocImm {
						fieldAddr := uintptr(d50.Imm.Int()) + 56
						r2 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r2, fieldAddr)
						d51 = JITValueDesc{Loc: LocReg, Reg: r2}
						ctx.BindReg(r2, &d51)
					} else {
						off := int32(56)
						baseReg := d50.Reg
						r3 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r3, baseReg, off)
						d51 = JITValueDesc{Loc: LocReg, Reg: r3}
						ctx.BindReg(r3, &d51)
					}
					ctx.FreeDesc(&d50)
					ctx.EnsureDesc(&d51)
					var d52 JITValueDesc
					if d51.Loc == LocImm {
						d52 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d51.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d51)
						if d51.Loc != LocReg && d51.Loc != LocRegPair && d51.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r4 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d51.Reg, 0)
						ctx.EmitSetcc(r4, CondNotEqual)
						d52 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d52)
					}
					ctx.EnsureDesc(&d52)
					ctx.EmitStoreToStack(d52, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d52)
					ctx.FreeDesc(&d51)
					if ps.General {
					}
					ps53 := PhiState{General: ps.General}
					ps53.OverlayValues = make([]JITValueDesc, 53)
					ps53.OverlayValues[1] = d1
					ps53.OverlayValues[2] = d2
					ps53.OverlayValues[3] = d3
					ps53.OverlayValues[4] = d4
					ps53.OverlayValues[5] = d5
					ps53.OverlayValues[6] = d6
					ps53.OverlayValues[8] = d8
					ps53.OverlayValues[12] = d12
					ps53.OverlayValues[22] = d22
					ps53.OverlayValues[23] = d23
					ps53.OverlayValues[24] = d24
					ps53.OverlayValues[25] = d25
					ps53.OverlayValues[28] = d28
					ps53.OverlayValues[31] = d31
					ps53.OverlayValues[47] = d47
					ps53.OverlayValues[48] = d48
					ps53.OverlayValues[49] = d49
					ps53.OverlayValues[50] = d50
					ps53.OverlayValues[51] = d51
					ps53.OverlayValues[52] = d52
					ps53.PhiValues = make([]JITValueDesc, 1)
					if ps53.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps53)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d54 := ps.PhiValues[0]
							ctx.EnsureDesc(&d54)
							ctx.EmitStoreToStack(d54, int32(bbs[4].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
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
						d55 = d2
						if d55.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d55)
						ctx.EmitStoreToStack(d55, int32(bbs[2].PhiBase)+int32(0))
						if d2.Loc == LocReg {
							ctx.UnprotectReg(d2.Reg)
						} else if d2.Loc == LocRegPair {
							ctx.UnprotectReg(d2.Reg)
							ctx.UnprotectReg(d2.Reg2)
						}
					}
					ps56 := PhiState{General: ps.General}
					ps56.OverlayValues = make([]JITValueDesc, 56)
					ps56.OverlayValues[1] = d1
					ps56.OverlayValues[2] = d2
					ps56.OverlayValues[3] = d3
					ps56.OverlayValues[4] = d4
					ps56.OverlayValues[5] = d5
					ps56.OverlayValues[6] = d6
					ps56.OverlayValues[8] = d8
					ps56.OverlayValues[12] = d12
					ps56.OverlayValues[22] = d22
					ps56.OverlayValues[23] = d23
					ps56.OverlayValues[24] = d24
					ps56.OverlayValues[25] = d25
					ps56.OverlayValues[28] = d28
					ps56.OverlayValues[31] = d31
					ps56.OverlayValues[47] = d47
					ps56.OverlayValues[48] = d48
					ps56.OverlayValues[49] = d49
					ps56.OverlayValues[50] = d50
					ps56.OverlayValues[51] = d51
					ps56.OverlayValues[52] = d52
					ps56.OverlayValues[54] = d54
					ps56.OverlayValues[55] = d55
					ps56.PhiValues = make([]JITValueDesc, 1)
					d57 = d2
					ps56.PhiValues[0] = d57
					if ps56.General && bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					return bbs[2].RenderPS(ps56)
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
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					ctx.ReclaimUntrackedRegs()
					d58 = args[0]
					d58.ID = 0
					ctx.EnsureDesc(&d58)
					ctx.EnsureDesc(&d58)
					d58 = JITPrepareScmerGoArg(ctx, d58)
					ctx.SyncDesc(&d58)
					d59 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Proc), []JITValueDesc{d58}, 1)
					d59.NoHeapPointer = false
					ctx.BindReg(d59.Reg, &d59)
					ctx.FreeDesc(&d58)
					ctx.EnsureDesc(&d59)
					var d60 JITValueDesc
					if d59.Loc == LocImm {
						d60 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d59.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d59)
						if d59.Loc != LocReg && d59.Loc != LocRegPair && d59.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r5 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d59.Reg, 0)
						ctx.EmitSetcc(r5, CondNotEqual)
						d60 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
						ctx.BindReg(r5, &d60)
					}
					ctx.FreeDesc(&d59)
					d61 = d60
					ctx.EnsureDesc(&d61)
					if d61.Loc != LocImm && d61.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d61.Loc == LocImm {
						if d61.Imm.Bool() {
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
							ps62.OverlayValues[54] = d54
							ps62.OverlayValues[55] = d55
							ps62.OverlayValues[57] = d57
							ps62.OverlayValues[58] = d58
							ps62.OverlayValues[59] = d59
							ps62.OverlayValues[60] = d60
							ps62.OverlayValues[61] = d61
							return bbs[3].RenderPS(ps62)
						}
						if ps.General {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
						}
						ps63 := PhiState{General: ps.General}
						ps63.OverlayValues = make([]JITValueDesc, 62)
						ps63.OverlayValues[1] = d1
						ps63.OverlayValues[2] = d2
						ps63.OverlayValues[3] = d3
						ps63.OverlayValues[4] = d4
						ps63.OverlayValues[5] = d5
						ps63.OverlayValues[6] = d6
						ps63.OverlayValues[8] = d8
						ps63.OverlayValues[12] = d12
						ps63.OverlayValues[22] = d22
						ps63.OverlayValues[23] = d23
						ps63.OverlayValues[24] = d24
						ps63.OverlayValues[25] = d25
						ps63.OverlayValues[28] = d28
						ps63.OverlayValues[31] = d31
						ps63.OverlayValues[47] = d47
						ps63.OverlayValues[48] = d48
						ps63.OverlayValues[49] = d49
						ps63.OverlayValues[50] = d50
						ps63.OverlayValues[51] = d51
						ps63.OverlayValues[52] = d52
						ps63.OverlayValues[54] = d54
						ps63.OverlayValues[55] = d55
						ps63.OverlayValues[57] = d57
						ps63.OverlayValues[58] = d58
						ps63.OverlayValues[59] = d59
						ps63.OverlayValues[60] = d60
						ps63.OverlayValues[61] = d61
						ps63.PhiValues = make([]JITValueDesc, 1)
						d64 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
						ps63.PhiValues[0] = d64
						return bbs[4].RenderPS(ps63)
					}
					if !ps.General {
						ps.General = true
						return bbs[5].RenderPS(ps)
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d61.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl12)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					ctx.EmitJmp(lbl5)
					ps65 := PhiState{General: true}
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
					ps65.OverlayValues[54] = d54
					ps65.OverlayValues[55] = d55
					ps65.OverlayValues[57] = d57
					ps65.OverlayValues[58] = d58
					ps65.OverlayValues[59] = d59
					ps65.OverlayValues[60] = d60
					ps65.OverlayValues[61] = d61
					ps65.OverlayValues[64] = d64
					ps66 := PhiState{General: true}
					ps66.OverlayValues = make([]JITValueDesc, 65)
					ps66.OverlayValues[1] = d1
					ps66.OverlayValues[2] = d2
					ps66.OverlayValues[3] = d3
					ps66.OverlayValues[4] = d4
					ps66.OverlayValues[5] = d5
					ps66.OverlayValues[6] = d6
					ps66.OverlayValues[8] = d8
					ps66.OverlayValues[12] = d12
					ps66.OverlayValues[22] = d22
					ps66.OverlayValues[23] = d23
					ps66.OverlayValues[24] = d24
					ps66.OverlayValues[25] = d25
					ps66.OverlayValues[28] = d28
					ps66.OverlayValues[31] = d31
					ps66.OverlayValues[47] = d47
					ps66.OverlayValues[48] = d48
					ps66.OverlayValues[49] = d49
					ps66.OverlayValues[50] = d50
					ps66.OverlayValues[51] = d51
					ps66.OverlayValues[52] = d52
					ps66.OverlayValues[54] = d54
					ps66.OverlayValues[55] = d55
					ps66.OverlayValues[57] = d57
					ps66.OverlayValues[58] = d58
					ps66.OverlayValues[59] = d59
					ps66.OverlayValues[60] = d60
					ps66.OverlayValues[61] = d61
					ps66.OverlayValues[64] = d64
					ps66.PhiValues = make([]JITValueDesc, 1)
					d67 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ps66.PhiValues[0] = d67
					snap68 := d1
					snap69 := d2
					snap70 := d3
					snap71 := d4
					snap72 := d5
					snap73 := d6
					snap74 := d8
					snap75 := d12
					snap76 := d22
					snap77 := d23
					snap78 := d24
					snap79 := d25
					snap80 := d28
					snap81 := d31
					snap82 := d47
					snap83 := d48
					snap84 := d49
					snap85 := d50
					snap86 := d51
					snap87 := d52
					snap88 := d54
					snap89 := d55
					snap90 := d57
					snap91 := d58
					snap92 := d59
					snap93 := d60
					snap94 := d61
					snap95 := d64
					snap96 := d67
					alloc97 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps66)
					}
					ctx.RestoreAllocState(alloc97)
					d1 = snap68
					d2 = snap69
					d3 = snap70
					d4 = snap71
					d5 = snap72
					d6 = snap73
					d8 = snap74
					d12 = snap75
					d22 = snap76
					d23 = snap77
					d24 = snap78
					d25 = snap79
					d28 = snap80
					d31 = snap81
					d47 = snap82
					d48 = snap83
					d49 = snap84
					d50 = snap85
					d51 = snap86
					d52 = snap87
					d54 = snap88
					d55 = snap89
					d57 = snap90
					d58 = snap91
					d59 = snap92
					d60 = snap93
					d61 = snap94
					d64 = snap95
					d67 = snap96
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps65)
					}
					return result
					ctx.FreeDesc(&d60)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps98 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps98)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				return result
			},
			JITInlineCost: 27,
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
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
				var d57 JITValueDesc
				_ = d57
				var d58 JITValueDesc
				_ = d58
				var d59 JITValueDesc
				_ = d59
				var d60 JITValueDesc
				_ = d60
				var d63 JITValueDesc
				_ = d63
				var d66 JITValueDesc
				_ = d66
				var d94 JITValueDesc
				_ = d94
				var d95 JITValueDesc
				_ = d95
				var d96 JITValueDesc
				_ = d96
				var d97 JITValueDesc
				_ = d97
				var d98 JITValueDesc
				_ = d98
				var d100 JITValueDesc
				_ = d100
				var d102 JITValueDesc
				_ = d102
				var d103 JITValueDesc
				_ = d103
				var d106 JITValueDesc
				_ = d106
				var d143 JITValueDesc
				_ = d143
				var d185 JITValueDesc
				_ = d185
				var d186 JITValueDesc
				_ = d186
				var d187 JITValueDesc
				_ = d187
				var d188 JITValueDesc
				_ = d188
				var d190 JITValueDesc
				_ = d190
				var d191 JITValueDesc
				_ = d191
				var d192 JITValueDesc
				_ = d192
				var d193 JITValueDesc
				_ = d193
				var d194 JITValueDesc
				_ = d194
				var d196 JITValueDesc
				_ = d196
				var d197 JITValueDesc
				_ = d197
				var d199 JITValueDesc
				_ = d199
				var d200 JITValueDesc
				_ = d200
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
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbpos_0_8 := int32(-1)
				_ = bbpos_0_8
				lbl9 := ctx.ReserveLabel()
				_ = lbl9
				bbpos_0_9 := int32(-1)
				_ = bbpos_0_9
				lbl10 := ctx.ReserveLabel()
				_ = lbl10
				bbpos_0_10 := int32(-1)
				_ = bbpos_0_10
				lbl11 := ctx.ReserveLabel()
				_ = lbl11
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
					ctx.EnsureDesc(&d4)
					d4 = JITPrepareScmerGoArg(ctx, d4)
					ctx.SyncDesc(&d4)
					d50 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Proc), []JITValueDesc{d4}, 1)
					d50.NoHeapPointer = false
					ctx.BindReg(d50.Reg, &d50)
					var d51 JITValueDesc
					ctx.EnsureDesc(&d50)
					if d50.Loc == LocImm {
						fieldAddr := uintptr(d50.Imm.Int()) + 56
						r2 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r2, fieldAddr)
						d51 = JITValueDesc{Loc: LocReg, Reg: r2}
						ctx.BindReg(r2, &d51)
					} else {
						off := int32(56)
						baseReg := d50.Reg
						r3 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r3, baseReg, off)
						d51 = JITValueDesc{Loc: LocReg, Reg: r3}
						ctx.BindReg(r3, &d51)
					}
					ctx.FreeDesc(&d50)
					ctx.EnsureDesc(&d51)
					var d52 JITValueDesc
					if d51.Loc == LocImm {
						d52 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d51.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d51)
						if d51.Loc != LocReg && d51.Loc != LocRegPair && d51.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r4 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d51.Reg, 0)
						ctx.EmitSetcc(r4, CondNotEqual)
						d52 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d52)
					}
					ctx.EnsureDesc(&d52)
					ctx.EmitStoreToStack(d52, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d52)
					ctx.FreeDesc(&d51)
					if ps.General {
					}
					ps53 := PhiState{General: ps.General}
					ps53.OverlayValues = make([]JITValueDesc, 53)
					ps53.OverlayValues[1] = d1
					ps53.OverlayValues[2] = d2
					ps53.OverlayValues[3] = d3
					ps53.OverlayValues[4] = d4
					ps53.OverlayValues[5] = d5
					ps53.OverlayValues[6] = d6
					ps53.OverlayValues[7] = d7
					ps53.OverlayValues[9] = d9
					ps53.OverlayValues[13] = d13
					ps53.OverlayValues[24] = d24
					ps53.OverlayValues[25] = d25
					ps53.OverlayValues[26] = d26
					ps53.OverlayValues[29] = d29
					ps53.OverlayValues[32] = d32
					ps53.OverlayValues[48] = d48
					ps53.OverlayValues[50] = d50
					ps53.OverlayValues[51] = d51
					ps53.OverlayValues[52] = d52
					ps53.PhiValues = make([]JITValueDesc, 1)
					if ps53.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps53)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d54 := ps.PhiValues[0]
							ctx.EnsureDesc(&d54)
							ctx.EmitStoreToStack(d54, int32(bbs[4].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
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
						d55 = d2
						if d55.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d55)
						ctx.EmitStoreToStack(d55, int32(bbs[2].PhiBase)+int32(0))
						if d2.Loc == LocReg {
							ctx.UnprotectReg(d2.Reg)
						} else if d2.Loc == LocRegPair {
							ctx.UnprotectReg(d2.Reg)
							ctx.UnprotectReg(d2.Reg2)
						}
					}
					ps56 := PhiState{General: ps.General}
					ps56.OverlayValues = make([]JITValueDesc, 56)
					ps56.OverlayValues[1] = d1
					ps56.OverlayValues[2] = d2
					ps56.OverlayValues[3] = d3
					ps56.OverlayValues[4] = d4
					ps56.OverlayValues[5] = d5
					ps56.OverlayValues[6] = d6
					ps56.OverlayValues[7] = d7
					ps56.OverlayValues[9] = d9
					ps56.OverlayValues[13] = d13
					ps56.OverlayValues[24] = d24
					ps56.OverlayValues[25] = d25
					ps56.OverlayValues[26] = d26
					ps56.OverlayValues[29] = d29
					ps56.OverlayValues[32] = d32
					ps56.OverlayValues[48] = d48
					ps56.OverlayValues[50] = d50
					ps56.OverlayValues[51] = d51
					ps56.OverlayValues[52] = d52
					ps56.OverlayValues[54] = d54
					ps56.OverlayValues[55] = d55
					ps56.PhiValues = make([]JITValueDesc, 1)
					d57 = d2
					ps56.PhiValues[0] = d57
					if ps56.General && bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					return bbs[2].RenderPS(ps56)
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
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					d4 = JITPrepareScmerGoArg(ctx, d4)
					ctx.SyncDesc(&d4)
					d58 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Proc), []JITValueDesc{d4}, 1)
					d58.NoHeapPointer = false
					ctx.BindReg(d58.Reg, &d58)
					ctx.EnsureDesc(&d58)
					var d59 JITValueDesc
					if d58.Loc == LocImm {
						d59 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d58.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d58)
						if d58.Loc != LocReg && d58.Loc != LocRegPair && d58.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r5 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d58.Reg, 0)
						ctx.EmitSetcc(r5, CondNotEqual)
						d59 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
						ctx.BindReg(r5, &d59)
					}
					ctx.FreeDesc(&d58)
					d60 = d59
					ctx.EnsureDesc(&d60)
					if d60.Loc != LocImm && d60.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d60.Loc == LocImm {
						if d60.Imm.Bool() {
							if ps.General {
							}
							ps61 := PhiState{General: ps.General}
							ps61.OverlayValues = make([]JITValueDesc, 61)
							ps61.OverlayValues[1] = d1
							ps61.OverlayValues[2] = d2
							ps61.OverlayValues[3] = d3
							ps61.OverlayValues[4] = d4
							ps61.OverlayValues[5] = d5
							ps61.OverlayValues[6] = d6
							ps61.OverlayValues[7] = d7
							ps61.OverlayValues[9] = d9
							ps61.OverlayValues[13] = d13
							ps61.OverlayValues[24] = d24
							ps61.OverlayValues[25] = d25
							ps61.OverlayValues[26] = d26
							ps61.OverlayValues[29] = d29
							ps61.OverlayValues[32] = d32
							ps61.OverlayValues[48] = d48
							ps61.OverlayValues[50] = d50
							ps61.OverlayValues[51] = d51
							ps61.OverlayValues[52] = d52
							ps61.OverlayValues[54] = d54
							ps61.OverlayValues[55] = d55
							ps61.OverlayValues[57] = d57
							ps61.OverlayValues[58] = d58
							ps61.OverlayValues[59] = d59
							ps61.OverlayValues[60] = d60
							return bbs[3].RenderPS(ps61)
						}
						if ps.General {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
						}
						ps62 := PhiState{General: ps.General}
						ps62.OverlayValues = make([]JITValueDesc, 61)
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
						ps62.OverlayValues[54] = d54
						ps62.OverlayValues[55] = d55
						ps62.OverlayValues[57] = d57
						ps62.OverlayValues[58] = d58
						ps62.OverlayValues[59] = d59
						ps62.OverlayValues[60] = d60
						ps62.PhiValues = make([]JITValueDesc, 1)
						d63 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
						ps62.PhiValues[0] = d63
						return bbs[4].RenderPS(ps62)
					}
					if !ps.General {
						ps.General = true
						return bbs[5].RenderPS(ps)
					}
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d60.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl16)
					ctx.EmitJmp(lbl17)
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl17)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					ctx.EmitJmp(lbl5)
					ps64 := PhiState{General: true}
					ps64.OverlayValues = make([]JITValueDesc, 64)
					ps64.OverlayValues[1] = d1
					ps64.OverlayValues[2] = d2
					ps64.OverlayValues[3] = d3
					ps64.OverlayValues[4] = d4
					ps64.OverlayValues[5] = d5
					ps64.OverlayValues[6] = d6
					ps64.OverlayValues[7] = d7
					ps64.OverlayValues[9] = d9
					ps64.OverlayValues[13] = d13
					ps64.OverlayValues[24] = d24
					ps64.OverlayValues[25] = d25
					ps64.OverlayValues[26] = d26
					ps64.OverlayValues[29] = d29
					ps64.OverlayValues[32] = d32
					ps64.OverlayValues[48] = d48
					ps64.OverlayValues[50] = d50
					ps64.OverlayValues[51] = d51
					ps64.OverlayValues[52] = d52
					ps64.OverlayValues[54] = d54
					ps64.OverlayValues[55] = d55
					ps64.OverlayValues[57] = d57
					ps64.OverlayValues[58] = d58
					ps64.OverlayValues[59] = d59
					ps64.OverlayValues[60] = d60
					ps64.OverlayValues[63] = d63
					ps65 := PhiState{General: true}
					ps65.OverlayValues = make([]JITValueDesc, 64)
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
					ps65.OverlayValues[54] = d54
					ps65.OverlayValues[55] = d55
					ps65.OverlayValues[57] = d57
					ps65.OverlayValues[58] = d58
					ps65.OverlayValues[59] = d59
					ps65.OverlayValues[60] = d60
					ps65.OverlayValues[63] = d63
					ps65.PhiValues = make([]JITValueDesc, 1)
					d66 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ps65.PhiValues[0] = d66
					snap67 := d1
					snap68 := d2
					snap69 := d3
					snap70 := d4
					snap71 := d5
					snap72 := d6
					snap73 := d7
					snap74 := d9
					snap75 := d13
					snap76 := d24
					snap77 := d25
					snap78 := d26
					snap79 := d29
					snap80 := d32
					snap81 := d48
					snap82 := d50
					snap83 := d51
					snap84 := d52
					snap85 := d54
					snap86 := d55
					snap87 := d57
					snap88 := d58
					snap89 := d59
					snap90 := d60
					snap91 := d63
					snap92 := d66
					alloc93 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps65)
					}
					ctx.RestoreAllocState(alloc93)
					d1 = snap67
					d2 = snap68
					d3 = snap69
					d4 = snap70
					d5 = snap71
					d6 = snap72
					d7 = snap73
					d9 = snap74
					d13 = snap75
					d24 = snap76
					d25 = snap77
					d26 = snap78
					d29 = snap79
					d32 = snap80
					d48 = snap81
					d50 = snap82
					d51 = snap83
					d52 = snap84
					d54 = snap85
					d55 = snap86
					d57 = snap87
					d58 = snap88
					d59 = snap89
					d60 = snap90
					d63 = snap91
					d66 = snap92
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps64)
					}
					return result
					ctx.FreeDesc(&d59)
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
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
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
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					d4 = JITPrepareScmerGoArg(ctx, d4)
					d94 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d94.Loc == LocRegPair || d94.Loc == LocStackPair || d94.Loc == LocRegTriple || d94.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d4)
					ctx.SyncDesc(&d94)
					d95 = ctx.EmitGoCallScalar(GoFuncAddr(SerializeToString), []JITValueDesc{d4, d94}, 2)
					d95.NoHeapPointer = false
					ctx.BindReg(d95.Reg, &d95)
					ctx.BindReg(d95.Reg2, &d95)
					ctx.StabilizeDescForControlFlow(&d95)
					d96 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d96)
					var d97 JITValueDesc
					if d96.Loc == LocImm {
						d97 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d96.Imm.Int() > 1)}
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d96.Reg, 1)
						ctx.EmitSetcc(r6, CondSignedGreater)
						d97 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
						ctx.BindReg(r6, &d97)
					}
					ctx.FreeDesc(&d96)
					d98 = d97
					ctx.EnsureDesc(&d98)
					if d98.Loc != LocImm && d98.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d98.Loc == LocImm {
						if d98.Imm.Bool() {
							if ps.General {
							}
							ps99 := PhiState{General: ps.General}
							ps99.OverlayValues = make([]JITValueDesc, 99)
							ps99.OverlayValues[1] = d1
							ps99.OverlayValues[2] = d2
							ps99.OverlayValues[3] = d3
							ps99.OverlayValues[4] = d4
							ps99.OverlayValues[5] = d5
							ps99.OverlayValues[6] = d6
							ps99.OverlayValues[7] = d7
							ps99.OverlayValues[9] = d9
							ps99.OverlayValues[13] = d13
							ps99.OverlayValues[24] = d24
							ps99.OverlayValues[25] = d25
							ps99.OverlayValues[26] = d26
							ps99.OverlayValues[29] = d29
							ps99.OverlayValues[32] = d32
							ps99.OverlayValues[48] = d48
							ps99.OverlayValues[50] = d50
							ps99.OverlayValues[51] = d51
							ps99.OverlayValues[52] = d52
							ps99.OverlayValues[54] = d54
							ps99.OverlayValues[55] = d55
							ps99.OverlayValues[57] = d57
							ps99.OverlayValues[58] = d58
							ps99.OverlayValues[59] = d59
							ps99.OverlayValues[60] = d60
							ps99.OverlayValues[63] = d63
							ps99.OverlayValues[66] = d66
							ps99.OverlayValues[94] = d94
							ps99.OverlayValues[95] = d95
							ps99.OverlayValues[96] = d96
							ps99.OverlayValues[97] = d97
							ps99.OverlayValues[98] = d98
							return bbs[9].RenderPS(ps99)
						}
						if ps.General {
							ctx.SyncDesc(&d95)
							if d95.Loc == LocReg {
								ctx.ProtectReg(d95.Reg)
							} else if d95.Loc == LocRegPair {
								ctx.ProtectReg(d95.Reg)
								ctx.ProtectReg(d95.Reg2)
							}
							d100 = d95
							if d100.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.SyncDesc(&d100)
							if d100.Loc == LocStackPair {
								ctx.EmitCopyStackWords(d100, int32(bbs[10].PhiBase)+int32(0), 2)
							} else if d100.Loc == LocInputPair {
								ctx.EnsureDesc(&d100)
								ctx.EmitStoreScmerToStack(d100, int32(bbs[10].PhiBase)+int32(0))
							} else if d100.Loc == LocRegPair || d100.Loc == LocImm {
								ctx.EmitStoreScmerToStack(d100, int32(bbs[10].PhiBase)+int32(0))
							} else {
								ctx.EnsureDesc(&d100)
								ctx.EmitStoreToStack(d100, int32(bbs[10].PhiBase)+int32(0))
								ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[10].PhiBase)+int32(0))+8)
							}
							if d95.Loc == LocReg {
								ctx.UnprotectReg(d95.Reg)
							} else if d95.Loc == LocRegPair {
								ctx.UnprotectReg(d95.Reg)
								ctx.UnprotectReg(d95.Reg2)
							}
						}
						ps101 := PhiState{General: ps.General}
						ps101.OverlayValues = make([]JITValueDesc, 101)
						ps101.OverlayValues[1] = d1
						ps101.OverlayValues[2] = d2
						ps101.OverlayValues[3] = d3
						ps101.OverlayValues[4] = d4
						ps101.OverlayValues[5] = d5
						ps101.OverlayValues[6] = d6
						ps101.OverlayValues[7] = d7
						ps101.OverlayValues[9] = d9
						ps101.OverlayValues[13] = d13
						ps101.OverlayValues[24] = d24
						ps101.OverlayValues[25] = d25
						ps101.OverlayValues[26] = d26
						ps101.OverlayValues[29] = d29
						ps101.OverlayValues[32] = d32
						ps101.OverlayValues[48] = d48
						ps101.OverlayValues[50] = d50
						ps101.OverlayValues[51] = d51
						ps101.OverlayValues[52] = d52
						ps101.OverlayValues[54] = d54
						ps101.OverlayValues[55] = d55
						ps101.OverlayValues[57] = d57
						ps101.OverlayValues[58] = d58
						ps101.OverlayValues[59] = d59
						ps101.OverlayValues[60] = d60
						ps101.OverlayValues[63] = d63
						ps101.OverlayValues[66] = d66
						ps101.OverlayValues[94] = d94
						ps101.OverlayValues[95] = d95
						ps101.OverlayValues[96] = d96
						ps101.OverlayValues[97] = d97
						ps101.OverlayValues[98] = d98
						ps101.OverlayValues[100] = d100
						ps101.PhiValues = make([]JITValueDesc, 1)
						d102 = d95
						ps101.PhiValues[0] = d102
						return bbs[10].RenderPS(ps101)
					}
					if !ps.General {
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d98.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl18)
					ctx.EmitJmp(lbl19)
					ctx.MarkLabel(lbl18)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl19)
					ctx.SyncDesc(&d95)
					if d95.Loc == LocReg {
						ctx.ProtectReg(d95.Reg)
					} else if d95.Loc == LocRegPair {
						ctx.ProtectReg(d95.Reg)
						ctx.ProtectReg(d95.Reg2)
					}
					d103 = d95
					if d103.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d103)
					if d103.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d103, int32(bbs[10].PhiBase)+int32(0), 2)
					} else if d103.Loc == LocInputPair {
						ctx.EnsureDesc(&d103)
						ctx.EmitStoreScmerToStack(d103, int32(bbs[10].PhiBase)+int32(0))
					} else if d103.Loc == LocRegPair || d103.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d103, int32(bbs[10].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d103)
						ctx.EmitStoreToStack(d103, int32(bbs[10].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[10].PhiBase)+int32(0))+8)
					}
					if d95.Loc == LocReg {
						ctx.UnprotectReg(d95.Reg)
					} else if d95.Loc == LocRegPair {
						ctx.UnprotectReg(d95.Reg)
						ctx.UnprotectReg(d95.Reg2)
					}
					ctx.EmitJmp(lbl11)
					ps104 := PhiState{General: true}
					ps104.OverlayValues = make([]JITValueDesc, 104)
					ps104.OverlayValues[1] = d1
					ps104.OverlayValues[2] = d2
					ps104.OverlayValues[3] = d3
					ps104.OverlayValues[4] = d4
					ps104.OverlayValues[5] = d5
					ps104.OverlayValues[6] = d6
					ps104.OverlayValues[7] = d7
					ps104.OverlayValues[9] = d9
					ps104.OverlayValues[13] = d13
					ps104.OverlayValues[24] = d24
					ps104.OverlayValues[25] = d25
					ps104.OverlayValues[26] = d26
					ps104.OverlayValues[29] = d29
					ps104.OverlayValues[32] = d32
					ps104.OverlayValues[48] = d48
					ps104.OverlayValues[50] = d50
					ps104.OverlayValues[51] = d51
					ps104.OverlayValues[52] = d52
					ps104.OverlayValues[54] = d54
					ps104.OverlayValues[55] = d55
					ps104.OverlayValues[57] = d57
					ps104.OverlayValues[58] = d58
					ps104.OverlayValues[59] = d59
					ps104.OverlayValues[60] = d60
					ps104.OverlayValues[63] = d63
					ps104.OverlayValues[66] = d66
					ps104.OverlayValues[94] = d94
					ps104.OverlayValues[95] = d95
					ps104.OverlayValues[96] = d96
					ps104.OverlayValues[97] = d97
					ps104.OverlayValues[98] = d98
					ps104.OverlayValues[100] = d100
					ps104.OverlayValues[102] = d102
					ps104.OverlayValues[103] = d103
					ps105 := PhiState{General: true}
					ps105.OverlayValues = make([]JITValueDesc, 104)
					ps105.OverlayValues[1] = d1
					ps105.OverlayValues[2] = d2
					ps105.OverlayValues[3] = d3
					ps105.OverlayValues[4] = d4
					ps105.OverlayValues[5] = d5
					ps105.OverlayValues[6] = d6
					ps105.OverlayValues[7] = d7
					ps105.OverlayValues[9] = d9
					ps105.OverlayValues[13] = d13
					ps105.OverlayValues[24] = d24
					ps105.OverlayValues[25] = d25
					ps105.OverlayValues[26] = d26
					ps105.OverlayValues[29] = d29
					ps105.OverlayValues[32] = d32
					ps105.OverlayValues[48] = d48
					ps105.OverlayValues[50] = d50
					ps105.OverlayValues[51] = d51
					ps105.OverlayValues[52] = d52
					ps105.OverlayValues[54] = d54
					ps105.OverlayValues[55] = d55
					ps105.OverlayValues[57] = d57
					ps105.OverlayValues[58] = d58
					ps105.OverlayValues[59] = d59
					ps105.OverlayValues[60] = d60
					ps105.OverlayValues[63] = d63
					ps105.OverlayValues[66] = d66
					ps105.OverlayValues[94] = d94
					ps105.OverlayValues[95] = d95
					ps105.OverlayValues[96] = d96
					ps105.OverlayValues[97] = d97
					ps105.OverlayValues[98] = d98
					ps105.OverlayValues[100] = d100
					ps105.OverlayValues[102] = d102
					ps105.OverlayValues[103] = d103
					ps105.PhiValues = make([]JITValueDesc, 1)
					d106 = d95
					ps105.PhiValues[0] = d106
					snap107 := d1
					snap108 := d2
					snap109 := d3
					snap110 := d4
					snap111 := d5
					snap112 := d6
					snap113 := d7
					snap114 := d9
					snap115 := d13
					snap116 := d24
					snap117 := d25
					snap118 := d26
					snap119 := d29
					snap120 := d32
					snap121 := d48
					snap122 := d50
					snap123 := d51
					snap124 := d52
					snap125 := d54
					snap126 := d55
					snap127 := d57
					snap128 := d58
					snap129 := d59
					snap130 := d60
					snap131 := d63
					snap132 := d66
					snap133 := d94
					snap134 := d95
					snap135 := d96
					snap136 := d97
					snap137 := d98
					snap138 := d100
					snap139 := d102
					snap140 := d103
					snap141 := d106
					alloc142 := ctx.SnapshotAllocState()
					if !bbs[10].Rendered {
						bbs[10].RenderPS(ps105)
					}
					ctx.RestoreAllocState(alloc142)
					d1 = snap107
					d2 = snap108
					d3 = snap109
					d4 = snap110
					d5 = snap111
					d6 = snap112
					d7 = snap113
					d9 = snap114
					d13 = snap115
					d24 = snap116
					d25 = snap117
					d26 = snap118
					d29 = snap119
					d32 = snap120
					d48 = snap121
					d50 = snap122
					d51 = snap123
					d52 = snap124
					d54 = snap125
					d55 = snap126
					d57 = snap127
					d58 = snap128
					d59 = snap129
					d60 = snap130
					d63 = snap131
					d66 = snap132
					d94 = snap133
					d95 = snap134
					d96 = snap135
					d97 = snap136
					d98 = snap137
					d100 = snap138
					d102 = snap139
					d103 = snap140
					d106 = snap141
					if !bbs[9].Rendered {
						return bbs[9].RenderPS(ps104)
					}
					return result
					ctx.FreeDesc(&d97)
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
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
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
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d4)
					if d4.Loc == LocRegPair || d4.Loc == LocStackPair || d4.Loc == LocInputPair {
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
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
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
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					ctx.ReclaimUntrackedRegs()
					d143 = d1
					ctx.EnsureDesc(&d143)
					if d143.Loc != LocImm && d143.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d143.Loc == LocImm {
						if d143.Imm.Bool() {
							if ps.General {
							}
							ps144 := PhiState{General: ps.General}
							ps144.OverlayValues = make([]JITValueDesc, 144)
							ps144.OverlayValues[1] = d1
							ps144.OverlayValues[2] = d2
							ps144.OverlayValues[3] = d3
							ps144.OverlayValues[4] = d4
							ps144.OverlayValues[5] = d5
							ps144.OverlayValues[6] = d6
							ps144.OverlayValues[7] = d7
							ps144.OverlayValues[9] = d9
							ps144.OverlayValues[13] = d13
							ps144.OverlayValues[24] = d24
							ps144.OverlayValues[25] = d25
							ps144.OverlayValues[26] = d26
							ps144.OverlayValues[29] = d29
							ps144.OverlayValues[32] = d32
							ps144.OverlayValues[48] = d48
							ps144.OverlayValues[50] = d50
							ps144.OverlayValues[51] = d51
							ps144.OverlayValues[52] = d52
							ps144.OverlayValues[54] = d54
							ps144.OverlayValues[55] = d55
							ps144.OverlayValues[57] = d57
							ps144.OverlayValues[58] = d58
							ps144.OverlayValues[59] = d59
							ps144.OverlayValues[60] = d60
							ps144.OverlayValues[63] = d63
							ps144.OverlayValues[66] = d66
							ps144.OverlayValues[94] = d94
							ps144.OverlayValues[95] = d95
							ps144.OverlayValues[96] = d96
							ps144.OverlayValues[97] = d97
							ps144.OverlayValues[98] = d98
							ps144.OverlayValues[100] = d100
							ps144.OverlayValues[102] = d102
							ps144.OverlayValues[103] = d103
							ps144.OverlayValues[106] = d106
							ps144.OverlayValues[143] = d143
							return bbs[7].RenderPS(ps144)
						}
						if ps.General {
						}
						ps145 := PhiState{General: ps.General}
						ps145.OverlayValues = make([]JITValueDesc, 144)
						ps145.OverlayValues[1] = d1
						ps145.OverlayValues[2] = d2
						ps145.OverlayValues[3] = d3
						ps145.OverlayValues[4] = d4
						ps145.OverlayValues[5] = d5
						ps145.OverlayValues[6] = d6
						ps145.OverlayValues[7] = d7
						ps145.OverlayValues[9] = d9
						ps145.OverlayValues[13] = d13
						ps145.OverlayValues[24] = d24
						ps145.OverlayValues[25] = d25
						ps145.OverlayValues[26] = d26
						ps145.OverlayValues[29] = d29
						ps145.OverlayValues[32] = d32
						ps145.OverlayValues[48] = d48
						ps145.OverlayValues[50] = d50
						ps145.OverlayValues[51] = d51
						ps145.OverlayValues[52] = d52
						ps145.OverlayValues[54] = d54
						ps145.OverlayValues[55] = d55
						ps145.OverlayValues[57] = d57
						ps145.OverlayValues[58] = d58
						ps145.OverlayValues[59] = d59
						ps145.OverlayValues[60] = d60
						ps145.OverlayValues[63] = d63
						ps145.OverlayValues[66] = d66
						ps145.OverlayValues[94] = d94
						ps145.OverlayValues[95] = d95
						ps145.OverlayValues[96] = d96
						ps145.OverlayValues[97] = d97
						ps145.OverlayValues[98] = d98
						ps145.OverlayValues[100] = d100
						ps145.OverlayValues[102] = d102
						ps145.OverlayValues[103] = d103
						ps145.OverlayValues[106] = d106
						ps145.OverlayValues[143] = d143
						return bbs[6].RenderPS(ps145)
					}
					if !ps.General {
						ps.General = true
						return bbs[8].RenderPS(ps)
					}
					lbl20 := ctx.ReserveLabel()
					lbl21 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d143.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl20)
					ctx.EmitJmp(lbl21)
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl7)
					ps146 := PhiState{General: true}
					ps146.OverlayValues = make([]JITValueDesc, 144)
					ps146.OverlayValues[1] = d1
					ps146.OverlayValues[2] = d2
					ps146.OverlayValues[3] = d3
					ps146.OverlayValues[4] = d4
					ps146.OverlayValues[5] = d5
					ps146.OverlayValues[6] = d6
					ps146.OverlayValues[7] = d7
					ps146.OverlayValues[9] = d9
					ps146.OverlayValues[13] = d13
					ps146.OverlayValues[24] = d24
					ps146.OverlayValues[25] = d25
					ps146.OverlayValues[26] = d26
					ps146.OverlayValues[29] = d29
					ps146.OverlayValues[32] = d32
					ps146.OverlayValues[48] = d48
					ps146.OverlayValues[50] = d50
					ps146.OverlayValues[51] = d51
					ps146.OverlayValues[52] = d52
					ps146.OverlayValues[54] = d54
					ps146.OverlayValues[55] = d55
					ps146.OverlayValues[57] = d57
					ps146.OverlayValues[58] = d58
					ps146.OverlayValues[59] = d59
					ps146.OverlayValues[60] = d60
					ps146.OverlayValues[63] = d63
					ps146.OverlayValues[66] = d66
					ps146.OverlayValues[94] = d94
					ps146.OverlayValues[95] = d95
					ps146.OverlayValues[96] = d96
					ps146.OverlayValues[97] = d97
					ps146.OverlayValues[98] = d98
					ps146.OverlayValues[100] = d100
					ps146.OverlayValues[102] = d102
					ps146.OverlayValues[103] = d103
					ps146.OverlayValues[106] = d106
					ps146.OverlayValues[143] = d143
					ps147 := PhiState{General: true}
					ps147.OverlayValues = make([]JITValueDesc, 144)
					ps147.OverlayValues[1] = d1
					ps147.OverlayValues[2] = d2
					ps147.OverlayValues[3] = d3
					ps147.OverlayValues[4] = d4
					ps147.OverlayValues[5] = d5
					ps147.OverlayValues[6] = d6
					ps147.OverlayValues[7] = d7
					ps147.OverlayValues[9] = d9
					ps147.OverlayValues[13] = d13
					ps147.OverlayValues[24] = d24
					ps147.OverlayValues[25] = d25
					ps147.OverlayValues[26] = d26
					ps147.OverlayValues[29] = d29
					ps147.OverlayValues[32] = d32
					ps147.OverlayValues[48] = d48
					ps147.OverlayValues[50] = d50
					ps147.OverlayValues[51] = d51
					ps147.OverlayValues[52] = d52
					ps147.OverlayValues[54] = d54
					ps147.OverlayValues[55] = d55
					ps147.OverlayValues[57] = d57
					ps147.OverlayValues[58] = d58
					ps147.OverlayValues[59] = d59
					ps147.OverlayValues[60] = d60
					ps147.OverlayValues[63] = d63
					ps147.OverlayValues[66] = d66
					ps147.OverlayValues[94] = d94
					ps147.OverlayValues[95] = d95
					ps147.OverlayValues[96] = d96
					ps147.OverlayValues[97] = d97
					ps147.OverlayValues[98] = d98
					ps147.OverlayValues[100] = d100
					ps147.OverlayValues[102] = d102
					ps147.OverlayValues[103] = d103
					ps147.OverlayValues[106] = d106
					ps147.OverlayValues[143] = d143
					snap148 := d1
					snap149 := d2
					snap150 := d3
					snap151 := d4
					snap152 := d5
					snap153 := d6
					snap154 := d7
					snap155 := d9
					snap156 := d13
					snap157 := d24
					snap158 := d25
					snap159 := d26
					snap160 := d29
					snap161 := d32
					snap162 := d48
					snap163 := d50
					snap164 := d51
					snap165 := d52
					snap166 := d54
					snap167 := d55
					snap168 := d57
					snap169 := d58
					snap170 := d59
					snap171 := d60
					snap172 := d63
					snap173 := d66
					snap174 := d94
					snap175 := d95
					snap176 := d96
					snap177 := d97
					snap178 := d98
					snap179 := d100
					snap180 := d102
					snap181 := d103
					snap182 := d106
					snap183 := d143
					alloc184 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps147)
					}
					ctx.RestoreAllocState(alloc184)
					d1 = snap148
					d2 = snap149
					d3 = snap150
					d4 = snap151
					d5 = snap152
					d6 = snap153
					d7 = snap154
					d9 = snap155
					d13 = snap156
					d24 = snap157
					d25 = snap158
					d26 = snap159
					d29 = snap160
					d32 = snap161
					d48 = snap162
					d50 = snap163
					d51 = snap164
					d52 = snap165
					d54 = snap166
					d55 = snap167
					d57 = snap168
					d58 = snap169
					d59 = snap170
					d60 = snap171
					d63 = snap172
					d66 = snap173
					d94 = snap174
					d95 = snap175
					d96 = snap176
					d97 = snap177
					d98 = snap178
					d100 = snap179
					d102 = snap180
					d103 = snap181
					d106 = snap182
					d143 = snap183
					if !bbs[7].Rendered {
						return bbs[7].RenderPS(ps146)
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
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
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
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					ctx.ReclaimUntrackedRegs()
					d185 = args[1]
					d185.ID = 0
					d187 = d185
					ctx.SyncDesc(&d187)
					if d187.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d187.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d187.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d187 = tmpScalar
					}
					d187 = JITPrepareScmerGoArg(ctx, d187)
					if d187.Loc != LocRegPair && d187.Loc != LocStackPair && d187.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d186 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d187}, 2)
					ctx.StabilizeDescForControlFlow(&d186)
					ctx.FreeDesc(&d185)
					if ps.General {
						ctx.SyncDesc(&d186)
						if d186.Loc == LocReg {
							ctx.ProtectReg(d186.Reg)
						} else if d186.Loc == LocRegPair {
							ctx.ProtectReg(d186.Reg)
							ctx.ProtectReg(d186.Reg2)
						}
						d188 = d186
						if d188.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d188)
						if d188.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d188, int32(bbs[10].PhiBase)+int32(0), 2)
						} else if d188.Loc == LocInputPair {
							ctx.EnsureDesc(&d188)
							ctx.EmitStoreScmerToStack(d188, int32(bbs[10].PhiBase)+int32(0))
						} else if d188.Loc == LocRegPair || d188.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d188, int32(bbs[10].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d188)
							ctx.EmitStoreToStack(d188, int32(bbs[10].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[10].PhiBase)+int32(0))+8)
						}
						if d186.Loc == LocReg {
							ctx.UnprotectReg(d186.Reg)
						} else if d186.Loc == LocRegPair {
							ctx.UnprotectReg(d186.Reg)
							ctx.UnprotectReg(d186.Reg2)
						}
					}
					ps189 := PhiState{General: ps.General}
					ps189.OverlayValues = make([]JITValueDesc, 189)
					ps189.OverlayValues[1] = d1
					ps189.OverlayValues[2] = d2
					ps189.OverlayValues[3] = d3
					ps189.OverlayValues[4] = d4
					ps189.OverlayValues[5] = d5
					ps189.OverlayValues[6] = d6
					ps189.OverlayValues[7] = d7
					ps189.OverlayValues[9] = d9
					ps189.OverlayValues[13] = d13
					ps189.OverlayValues[24] = d24
					ps189.OverlayValues[25] = d25
					ps189.OverlayValues[26] = d26
					ps189.OverlayValues[29] = d29
					ps189.OverlayValues[32] = d32
					ps189.OverlayValues[48] = d48
					ps189.OverlayValues[50] = d50
					ps189.OverlayValues[51] = d51
					ps189.OverlayValues[52] = d52
					ps189.OverlayValues[54] = d54
					ps189.OverlayValues[55] = d55
					ps189.OverlayValues[57] = d57
					ps189.OverlayValues[58] = d58
					ps189.OverlayValues[59] = d59
					ps189.OverlayValues[60] = d60
					ps189.OverlayValues[63] = d63
					ps189.OverlayValues[66] = d66
					ps189.OverlayValues[94] = d94
					ps189.OverlayValues[95] = d95
					ps189.OverlayValues[96] = d96
					ps189.OverlayValues[97] = d97
					ps189.OverlayValues[98] = d98
					ps189.OverlayValues[100] = d100
					ps189.OverlayValues[102] = d102
					ps189.OverlayValues[103] = d103
					ps189.OverlayValues[106] = d106
					ps189.OverlayValues[143] = d143
					ps189.OverlayValues[185] = d185
					ps189.OverlayValues[186] = d186
					ps189.OverlayValues[187] = d187
					ps189.OverlayValues[188] = d188
					ps189.PhiValues = make([]JITValueDesc, 1)
					d190 = d186
					ps189.PhiValues[0] = d190
					if ps189.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps189)
					return result
				}
				bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d191 := ps.PhiValues[0]
							ctx.EnsureDesc(&d191)
							ctx.EmitStoreScmerToStack(d191, int32(bbs[10].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
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
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != LocNone {
						d185 = ps.OverlayValues[185]
					}
					if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
						d186 = ps.OverlayValues[186]
					}
					if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
						d187 = ps.OverlayValues[187]
					}
					if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
						d188 = ps.OverlayValues[188]
					}
					if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
						d190 = ps.OverlayValues[190]
					}
					if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != LocNone {
						d191 = ps.OverlayValues[191]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d3 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					d192 = ctx.EmitGoCallScalar(GoFuncAddr(func() *[1]any { return new([1]any) }), nil, 1)
					d193 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.EnsureDesc(&d3)
					d194 = ctx.EmitGoCallScalar(GoFuncAddr(func(value string) any { return value }), []JITValueDesc{d3}, 2)
					ctx.FreeDesc(&d3)
					ctx.EnsureDesc(&d194)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[1]any, index int, value any) { dst[index] = value }), []JITValueDesc{d192, d193, d194})
					sliceResults195 := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *[1]any) []any { return value[0:1:1] }), []JITValueDesc{d192}, []uint8{3}, []uint8{1})
					d196 = sliceResults195[0]
					d197 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("warning: JIT fallback: %s\n")}
					ctx.EnsureDesc(&d197)
					if d197.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d197.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d197.Imm)
						ptrWord, _ := d197.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d197.Imm.String())))
						d197 = tmpPair
					} else if d197.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d197.Type, Reg: ctx.AllocRegExcept(d197.Reg), Reg2: ctx.AllocRegExcept(d197.Reg)}
						switch d197.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d197)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d197)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d197)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d197)
						d197 = tmpPair
					}
					if d197.Loc != LocRegPair && d197.Loc != LocStackPair && d197.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (fmt.Printf arg0)")
					}
					ctx.EnsureDesc(&d196)
					ctx.EnsureDesc(&d196)
					ctx.EnsureDesc(&d196)
					if d196.Loc != LocRegTriple && d196.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (fmt.Printf arg1)")
					}
					ctx.SyncDesc(&d197)
					ctx.SyncDesc(&d196)
					callResults198 := JITEmitGoCallResults(ctx, GoFuncAddr(fmt.Printf), []JITValueDesc{d197, d196}, []uint8{1, 2}, []uint8{0, 3})
					ctx.FreeDesc(&d197)
					d199 = callResults198[0]
					_ = d199
					d200 = callResults198[1]
					_ = d200
					if ps.General {
					}
					ps201 := PhiState{General: ps.General}
					ps201.OverlayValues = make([]JITValueDesc, 201)
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
					ps201.OverlayValues[54] = d54
					ps201.OverlayValues[55] = d55
					ps201.OverlayValues[57] = d57
					ps201.OverlayValues[58] = d58
					ps201.OverlayValues[59] = d59
					ps201.OverlayValues[60] = d60
					ps201.OverlayValues[63] = d63
					ps201.OverlayValues[66] = d66
					ps201.OverlayValues[94] = d94
					ps201.OverlayValues[95] = d95
					ps201.OverlayValues[96] = d96
					ps201.OverlayValues[97] = d97
					ps201.OverlayValues[98] = d98
					ps201.OverlayValues[100] = d100
					ps201.OverlayValues[102] = d102
					ps201.OverlayValues[103] = d103
					ps201.OverlayValues[106] = d106
					ps201.OverlayValues[143] = d143
					ps201.OverlayValues[185] = d185
					ps201.OverlayValues[186] = d186
					ps201.OverlayValues[187] = d187
					ps201.OverlayValues[188] = d188
					ps201.OverlayValues[190] = d190
					ps201.OverlayValues[191] = d191
					ps201.OverlayValues[192] = d192
					ps201.OverlayValues[193] = d193
					ps201.OverlayValues[194] = d194
					ps201.OverlayValues[196] = d196
					ps201.OverlayValues[197] = d197
					ps201.OverlayValues[199] = d199
					ps201.OverlayValues[200] = d200
					if ps201.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps201)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps202 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps202)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  38,
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
			JITInlineCost: 2,
		},
	})
}

// jitCompile compiles a Proc to a native function (tagFunc)
// Already compiled functions (tagFunc, tagFuncEnv) are passed through unchanged
func jitCompile(a ...Scmer) Scmer {
	return jitCompileMode(true, a...)
}

func jitCompileMode(recursiveLambdas bool, a ...Scmer) Scmer {
	return jitCompileModePublish(recursiveLambdas, true, true, a...)
}

func jitCompileModeDeferred(recursiveLambdas bool, a ...Scmer) Scmer {
	return jitCompileModePublish(recursiveLambdas, false, true, a...)
}

func jitCompileProbe(a ...Scmer) Scmer {
	return jitCompileModePublish(true, true, false, a...)
}

func jitCompileModePublish(recursiveLambdas bool, waitForPublication, install bool, a ...Scmer) Scmer {
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
		for _, codeCap := range [...]int{16 * 1024, 64 * 1024, 256 * 1024, 1024 * 1024, 4 * 1024 * 1024, 16 * 1024 * 1024} {
			ptr, arena, reservation := globalJITPool.Alloc(codeCap)
			buf := &execBuf{ptr: ptr, n: codeCap, arena: arena, reservation: reservation}
			codeLen, roots, overflow, transferInputArgs, hiddenArgs, needsStableArgs, coverage := jitCompileProcToExec(proc, buf, recursiveLambdas)
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
					StackFrameSize:    buf.stackFrameSize,
					TransferInputArgs: transferInputArgs,
					HiddenArgs:        hiddenArgs,
					CodePtr:           ptr,
					CodeLen:           codeLen,
					Arena:             arena,
					ConstRoots:        roots,
					Proc:              sourceProc,
					RecursiveLambdas:  recursiveLambdas,
					NeedsStableArgs:   needsStableArgs,
					Coverage:          coverage,
				}
				runtime.SetFinalizer(jep, func(jep *JITEntryPoint) {
					if jep.Arena != nil && jep.CodePtr != nil {
						globalJITPool.Free(jep.CodePtr, jep.CodeLen)
					}
				})
				if install {
					proc.Compiled = jep
					return v
				}
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

// execBuf is a small wrapper for writable memory (arena-backed or standalone)
type execBuf struct {
	ptr            unsafe.Pointer
	n              int       // size
	arena          *jitArena // owning arena (nil for standalone buffers)
	reservation    *jitCodeReservation
	stackMaps      []jitStackMap
	stackFrameSize int32
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

func maybeLogJITCodeName(entry *JITEntryPoint) {
	if os.Getenv("MEMCP_JIT_DUMP_DIR") == "" || entry == nil || entry.DebugName == "" {
		return
	}
	fmt.Printf("jitdump: name=%s code=%p bytes=%d\n", entry.DebugName, entry.CodePtr, entry.CodeLen)
}

func maybeLogJITImportCandidate(name Symbol, entry *JITEntryPoint, selected bool) {
	if os.Getenv("MEMCP_JIT_DUMP_DIR") == "" {
		return
	}
	if entry == nil {
		fmt.Printf("jitdump: import=%s compiled=false\n", name)
		return
	}
	fmt.Printf("jitdump: import=%s selected=%t expressions=%d dynamic-calls=%d native-calls=%d inlined-calls=%d\n",
		name, selected, entry.Coverage.Expressions, entry.Coverage.DynamicCalls,
		entry.Coverage.NativeCalls, entry.Coverage.InlinedCalls)
}
