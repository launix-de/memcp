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
	"context"
	"fmt"
	"math"
	"math/bits"
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
 - Each emitter invocation owns a fixed BBDescriptor table; descriptor locals are shared by its renderers.
 - the BB chain is started by firstbb.render(). Each bb render function can tail-call other bb render functions in order to "enqueue" them -> jumps tail-call ONE sucessor BB, conditional jumps tail-call up to TWO successor BBs, so we have a DFS traversal of all reachable BBs
 - Generated BBs use canonical phi homes and are rendered at most once. A previously emitted block is reached through its label.
 - Concrete argument types and LocImm values still specialize operations and prune unreachable branches while rendering. Phi transfer layout and descriptor scope are planned by jitgen; generated block transitions need no incoming overlay arrays.
 - a BB render function calls the arch-specific instruction emitters that write into the JIT buffer aswell as other emitters to inline the functions
 - each

Input arguments (args):

  Each args[i] describes where the i-th operand lives at the point of the
  call. The emitter must handle all location modes:

  - LocImm:     compile-time constant. args[i].Imm holds a Scmer value;
                Imm.GetTag() carries the type. No register is allocated.
                The emitter SHOULD constant-fold when all inputs are LocImm.
  - LocReg:     unboxed primitive in args[i].Reg.
  - LocFPReg:   unboxed float in the backend's floating-point register file.
                The allocator may use the same physical bank as a temporary
                overflow home for another proven pointer-free scalar, but
                EnsureDesc restores that value before an emitter sees it.
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
  - LocFPReg:   result MUST be placed into result.Reg as a native float.
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

  - Allocate integer registers with ctx.AllocReg() and floating-point registers
    with ctx.AllocFPReg(); free either class with ctx.FreeReg(r).
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
	// Owner keeps the entry point which owns shared machine code alive. FuncValue
	// retains the GC-typed Go closure object whose first word is CodePtr.
	Owner      *JITEntryPoint
	FuncValue  any
	HiddenArgs []JITHiddenArg
	CodePtr    unsafe.Pointer   // start of code in arena
	CodeLen    int              // bytes used
	Arena      *jitArena        // owning arena (for free on GC)
	ConstRoots []unsafe.Pointer // GC roots for constants embedded into machine code
	// Dependencies retain native callees and embedded callable values. Their
	// stack maps must be ready before this entry point becomes executable.
	Dependencies []*JITEntryPoint
	reservation  *jitCodeReservation // readiness of this code and its embedded callees
	Proc         Proc                // original Proc for serialization
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
	// Closure metadata is shared by every Proc header bound to this machine
	// code. Runtime capture values follow the Proc header inline.
	CaptureBase    int
	CaptureCount   int
	CaptureKeys    []Scmer
	CaptureSymbols []Symbol
	JITArity       int
	JITDirect      uintptr
}

type JITCoverage struct {
	Expressions  int
	DynamicCalls int
	DirectProcs  int
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
	return jep.callFunction(jep.Native, args)
}

// callFunction applies entry-point metadata but invokes the supplied concrete
// funcval. Bound Proc instances share immutable compilation metadata while the
// funcval context identifies the instance whose inline capture tail must be
// visible in the closure-context register.
func (jep *JITEntryPoint) callFunction(function func(...Scmer) Scmer, args []Scmer) (result Scmer) {
	if jep == nil || function == nil {
		panic("JIT: nil entry point")
	}
	if JITLog && jep.DebugName != "" {
		fmt.Printf("JIT: call %s argc=%d\n", jep.DebugName, len(args))
	}
	// Appending closure or specialization inputs must not reuse caller-owned
	// capacity. A plain Proc call borrows its argument slice for the duration of
	// the call; builtins which retain it, notably list, own that decision.
	stableArgs := len(jep.HiddenArgs) != 0
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
			args = append(args, jitPrepareCallback(args[spec.SourceInput]))
		default:
			panic("JIT: invalid hidden argument kind")
		}
	}
	result = function(args...)
	runtime.KeepAlive(args)
	runtime.KeepAlive(jep)
	runtime.KeepAlive(jep.Owner)
	return result
}

func (proc *Proc) callJIT(args []Scmer) Scmer {
	if proc == nil || proc.JITCode == 0 || proc.Compiled == nil {
		panic("JIT: procedure has no native implementation")
	}
	entry := proc.Compiled
	if entry.JITDirect != 0 && (entry.JITArity < 0 || entry.JITArity == len(args)) {
		result := proc.jitFunction()(args...)
		runtime.KeepAlive(args)
		runtime.KeepAlive(proc)
		runtime.KeepAlive(entry)
		return result
	}
	// Context-free closures have one canonical static funcval. Specialized Proc
	// copies can share that entry without becoming closure contexts themselves.
	// A bound capture tail or a rebound lexical environment, however, belongs to
	// this concrete Proc and must remain in the closure-context register.
	function := entry.Native
	template := JITProcForFunction(function)
	if entry.CaptureCount != 0 || template == nil || template.En != proc.En {
		function = proc.jitFunction()
	}
	return entry.callFunction(function, args)
}

func attachProcJIT(proc *Proc, entry *JITEntryPoint) {
	jitNativeCodes.Store(uintptr(entry.CodePtr), struct{}{})
	proc.JITCode = uintptr(entry.CodePtr)
	proc.Compiled = entry
	entry.Native = proc.jitFunction()
	if entry.FuncValue == nil {
		entry.FuncValue = proc
	}
	entry.JITArity = 0
	entry.JITDirect = 0
	if len(entry.HiddenArgs) != 0 {
		return
	}
	params := proc.Params
	for params.GetTag() == tagSourceInfo {
		params = params.SourceInfo().value
	}
	switch params.GetTag() {
	case tagSlice:
		entry.JITArity = len(params.Slice())
	case tagNil:
		entry.JITArity = 0
	case tagSymbol:
		// A Scheme variadic parameter is one list binding, not the raw Go
		// variadic slice. Keep it on the metadata path until the native prolog
		// materializes that binding explicitly.
		return
	default:
		return
	}
	entry.JITDirect = 1
}

var (
	jitProcContextTypes sync.Map // map[int]unsafe.Pointer, opaque runtime/jit.TailType
	jitNativeCodes      sync.Map // map[uintptr]struct{}, exact JIT entry PCs
)

func jitProcContextAllocation(captureCount int) unsafe.Pointer {
	if captureCount < 0 {
		panic("jit: negative Proc capture count")
	}
	if cached, ok := jitProcContextTypes.Load(captureCount); ok {
		return cached.(unsafe.Pointer)
	}
	prepared := jitPrepareProcContextType(captureCount)
	actual, _ := jitProcContextTypes.LoadOrStore(captureCount, prepared)
	return actual.(unsafe.Pointer)
}

func (proc *Proc) jitFunction() func(...Scmer) Scmer {
	if proc == nil || proc.JITCode == 0 {
		return nil
	}
	valuePointer := unsafe.Pointer(proc)
	return *(*func(...Scmer) Scmer)(unsafe.Pointer(&valuePointer))
}

func jitAllocateProcContext(proc *Proc, captureCount int) *Proc {
	typ := jitProcContextAllocation(captureCount)
	bound := (*Proc)(jitRuntimeAllocTyped(typ))
	*bound = *proc
	return bound
}

// jitBindProcContext initializes escaping closures with Go write barriers.
// Allocation can return a black object while its captured values are still
// white. Raw JIT stores would hide those edges from concurrent marking.
func jitBindProcContext(prepared unsafe.Pointer, template *Proc, captures *Scmer, count int, bindSelf bool) *Proc {
	bound := (*Proc)(jitRuntimeAllocTyped(prepared))
	*bound = *template
	target := unsafe.Slice((*Scmer)(unsafe.Add(unsafe.Pointer(bound), unsafe.Offsetof(ProcJIT{}.Context))), count)
	copyCount := count
	if bindSelf && copyCount != 0 {
		copyCount--
	}
	copy(target, unsafe.Slice(captures, copyCount))
	if bindSelf && count != 0 {
		// NewProc copies only the header; self must retain this inline context.
		target[count-1] = Scmer{(*byte)(unsafe.Pointer(bound)), makeAux(tagProc, 0)}
	}
	return bound
}

// JITProcForFunction recovers the original Scheme procedure from a native JIT
// funcval in O(1). A code-range check precedes the sentinel read so ordinary Go
// function values are never interpreted as MemCP closure objects.
func JITProcForFunction(function func(...Scmer) Scmer) *Proc {
	if function == nil {
		return nil
	}
	valuePointer := *(*unsafe.Pointer)(unsafe.Pointer(&function))
	if valuePointer == nil {
		return nil
	}
	code := *(*uintptr)(valuePointer)
	if _, exists := jitNativeCodes.Load(code); !exists {
		return nil
	}
	return (*Proc)(valuePointer)
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

// JITValueDesc describes a runtime value while machine code is being emitted.
// The descriptor itself lives in Go and is mutable; the described value lives
// in the future JIT invocation. An emitted load/store changes the latter only
// when the generated code runs, while changing Loc/Reg/StackOff immediately
// changes the emitter's belief about that future location. Every helper must
// keep those two timelines separate.
//
// ID is allocator identity, not SSA identity. Descriptors with the same nonzero
// ID are aliases of one spill owner and SyncDesc makes their placement agree.
// ID 0 is a non-owning view: it may be passed to an emitter, but must not free or
// steal the source descriptor's registers. A helper which materializes such a
// view may assign it a fresh ID, thereby creating a new temporary owner.
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
	RegClass JITRegisterClass
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
	// ReturnTypeMerged means Type describes every emitted return arm, rather
	// than just the last rendered arm. It survives forwarding the same value;
	// each new declaration emission clears the caller-selected target proof.
	ReturnTypeMerged bool
	// RelocatablePointer distinguishes an unboxed address from ordinary scalar
	// integers. Such values need stack-map coverage while live across a Go call.
	RelocatablePointer bool
	// GoArray marks a one-word descriptor as the data address of a Go array.
	// Unlike a slice it has no adjacent length/capacity header.
	GoArray bool
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
	// StackFunc permits the outermost lambda produced for this exact result slot
	// to use an invocation-local Go funcval. Nested expressions do not inherit it.
	StackFunc bool
	// Condition is valid only for LocFlags. FlagsID == 0 denotes a comparison
	// consumed immediately by its block's terminating branch. Nonzero IDs may
	// cross emitter returns; the context materializes them before instructions,
	// register moves, spills, or control-flow boundaries can invalidate flags.
	Condition JITCondition
	// FlagsID identifies a lazily materialized boolean. Zero retains the
	// existing immediate-branch-only flags contract used inside generated CFGs.
	FlagsID uint32
}

// jitValueWordIsPointer reports whether word is a relocatable Go pointer in
// the descriptor's machine representation. Scalar descriptors are unboxed;
// unlike a two-word Scmer, their only word is a pointer exclusively when the
// producer says so explicitly through RelocatablePointer.
func jitValueWordIsPointer(value JITValueDesc, word int32) bool {
	if word != 0 {
		return false
	}
	switch value.Loc {
	case LocReg, LocStack:
		return value.RelocatablePointer
	case LocRegPair, LocStackPair, LocInputPair, LocRegTriple, LocStackTriple:
		return !value.NoHeapPointer
	default:
		return false
	}
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
	CondParity
	CondNotParity
)

// InvertJITCondition returns the complementary architecture-neutral branch.
func InvertJITCondition(condition JITCondition) JITCondition {
	switch condition {
	case CondEqual:
		return CondNotEqual
	case CondNotEqual:
		return CondEqual
	case CondSignedLess:
		return CondSignedGreaterOrEqual
	case CondSignedGreaterOrEqual:
		return CondSignedLess
	case CondSignedLessOrEqual:
		return CondSignedGreater
	case CondSignedGreater:
		return CondSignedLessOrEqual
	case CondUnsignedBelow:
		return CondUnsignedAboveOrEqual
	case CondUnsignedAboveOrEqual:
		return CondUnsignedBelow
	case CondUnsignedBelowOrEqual:
		return CondUnsignedAbove
	case CondUnsignedAbove:
		return CondUnsignedBelowOrEqual
	case CondParity:
		return CondNotParity
	case CondNotParity:
		return CondParity
	default:
		panic("jit: invalid branch condition")
	}
}

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
	CcP  = CondParity
	CcNP = CondNotParity
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
	LocClosurePair // One Scmer in the current Go funcval's typed closure environment
	LocFlags       // Ephemeral comparison result consumed immediately by a branch
	LocFPReg       // Native FP value, or a pointer-free scalar payload in an FP overflow home
)

// JITRegisterClass separates values which share liveness but cannot share a
// physical register. The common compiler uses these architecture-neutral
// classes; each backend maps them to its native register files.
type JITRegisterClass uint8

const (
	JITRegisterClassGPR JITRegisterClass = iota
	JITRegisterClassFP
)

type JITFloatOp uint8

const (
	JITFloatAdd JITFloatOp = iota
	JITFloatSub
	JITFloatMul
	JITFloatDiv
)

// JITFixup records a forward reference that must be patched after all
// labels are placed.
type JITFixup struct {
	CodePos  int32    // position in code
	LabelID  JITLabel // target label
	Size     uint8    // 1=rel8, 4=rel32
	Relative bool     // true for PC-relative jumps
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
	reg      Reg
	reg2     Reg
	reg3     Reg
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

// jitStackRootBitmap stores exact word locations relative to one frame base.
type jitStackRootBitmap struct {
	first int32 // byte offset represented by the first bit; may be negative
	bits  []byte
}

// jitTestPostEmitHook, when non-nil, receives the JITContext after body
// emission completes (FrameRoots and Safepoints fully populated) but before
// finalizeStackMaps discards it. Test-only introspection; production code
// never sets this.
var jitTestPostEmitHook func(ctx *JITContext)

// jitSafepoint is recorded while emitting a Go call. FrameSize is deliberately
// absent: the one-pass emitter only knows the final static frame size after the
// complete function has been written.
type jitSafepoint struct {
	pcOffset        int32
	dynamicSP       int32
	roots           [3]jitStackRootBitmap
	entry           bool
	entryFrameWords uintptr
	entryPointerMap []byte
}

// jitSortedFrameRoots returns the permanent frame words registered by the
// one-pass emitter in deterministic order. Dynamic call-area roots live below
// the stable stack pointer and are initialized at their call site.
func jitSortedFrameRoots(unique map[jitStackRoot]struct{}) []jitStackRoot {
	roots := make([]jitStackRoot, 0, len(unique))
	for root := range unique {
		roots = append(roots, root)
	}
	sort.Slice(roots, func(i, j int) bool {
		if roots[i].base != roots[j].base {
			return roots[i].base < roots[j].base
		}
		return roots[i].offset < roots[j].offset
	})
	return roots
}

// jitStackMap is the runtime-independent form passed through the common JIT
// code. The goexperiment.jit implementation converts it to runtime/jit maps;
// the vanilla implementation deliberately ignores it.
type jitStackMap struct {
	pcOffset   uintptr
	frameWords uintptr
	pointerMap []byte
	entry      bool
	// Entry maps describe the caller-owned register spill area used while a
	// JIT prologue calls morestack. The variadic Scheme ABI and typed storage
	// ABIs have different spill layouts, so these cannot be hard-coded by the
	// runtime adapter.
	entryFrameWords uintptr
	entryPointerMap []byte
}

// JITStorageGetValueFunc is the native scalar column-reader ABI.
type JITStorageGetValueFunc func(uint32) Scmer

// JITStorageGetValueRangeFunc is the native consecutive-range column-reader ABI.
type JITStorageGetValueRangeFunc func(uint32, uint32, []Scmer, int)

// JITStorageGetValueMultiFunc is the native arbitrary-record column-reader ABI.
type JITStorageGetValueMultiFunc func([]uint32, []Scmer, int)

// JITMapReduceBufferFunc folds rows from a row-major Scmer buffer. The row
// width and callback are compile-time properties of the function; rows remains
// explicit so zero-column reducers such as COUNT(*) need no synthetic values.
type JITMapReduceBufferFunc func(Scmer, []Scmer, int) Scmer

// JITFilterBufferFunc evaluates a typed row-major Scmer buffer and compacts
// the corresponding record identifiers in place. The row width and predicate
// are compile-time properties of the function.
type JITFilterBufferFunc func([]uint32, []Scmer) int

// JITBufferProbeSafe proves that replaying a numeric reducer cannot mutate its
// accumulator or invoke callbacks. Const alone is insufficient: aggregate
// containers can have ownership-based mutable implementations.
func JITBufferProbeSafe(proc *Proc, arity int) bool {
	if proc == nil || arity < 1 {
		return false
	}
	var safe func(Scmer) bool
	safe = func(expr Scmer) bool {
		for expr.GetTag() == tagSourceInfo {
			expr = expr.SourceInfo().value
		}
		switch expr.GetTag() {
		case tagNil, tagInt, tagFloat:
			return true
		case tagNthLocalVar:
			return int(expr.NthLocalVar()) < arity
		case tagSymbol:
			if proc.Params.GetTag() == tagSlice {
				for _, param := range proc.Params.Slice() {
					if param.GetTag() == tagSymbol && param.Symbol() == expr.Symbol() {
						return true
					}
				}
			}
			return false
		case tagSlice:
			list := expr.Slice()
			if len(list) == 0 {
				return false
			}
			decl := DeclarationForValue(list[0])
			if decl == nil || decl.RetainsCallArgs || decl.Type == nil || !decl.Type.Const || decl.Type.Return == nil {
				return false
			}
			switch decl.Type.Return.Kind {
			case "number", "number|nil", "int", "int|nil":
			default:
				return false
			}
			for _, param := range decl.Type.Params {
				switch param.Kind {
				case "number", "number|nil", "int", "int|nil":
				default:
					return false
				}
			}
			for _, arg := range list[1:] {
				if !safe(arg) {
					return false
				}
			}
			return true
		default:
			return false
		}
	}
	return safe(proc.Body)
}

// PrepareJITBufferProc returns source suitable for typed buffer-loop inlining.
// Native declarations with a JIT emitter are wrapped in a procedure so physical
// operators can specialize the same implementation without teaching the
// storage package about individual scalar or aggregate names.
func PrepareJITBufferProc(source Scmer, arity int) *Proc {
	if arity < 1 {
		return nil
	}
	if source.GetTag() == tagProc && source.Proc() != nil {
		return source.Proc()
	}
	if source.GetTag() == tagFunc {
		if proc := JITProcForFunction(source.Func()); proc != nil {
			return proc
		}
	}
	declaration := DeclarationForValue(source)
	if declaration == nil || declaration.Type == nil || declaration.Type.JITEmit == nil || len(declaration.Type.Params) != arity {
		return nil
	}
	params := make([]Scmer, arity)
	body := make([]Scmer, arity+1)
	body[0] = source
	for index := range params {
		parameter := NewSymbol(fmt.Sprintf("\x00buffer-arg-%d", index))
		params[index] = parameter
		body[index+1] = parameter
	}
	return &Proc{Params: NewSlice(params), Body: NewSlice(body), En: &Globalenv}
}

// JITStorageGetValueEmitter emits one scalar storage read. The bound method
// receiver is the concrete finished storage; index and result use the typed Go ABI.
type JITStorageGetValueEmitter func(*JITContext, JITValueDesc, JITValueDesc) JITValueDesc

// JITStorageGetValueRangeEmitter emits one consecutive bulk storage read.
type JITStorageGetValueRangeEmitter func(*JITContext, JITValueDesc, JITValueDesc, JITValueDesc, JITValueDesc, JITValueDesc) JITValueDesc

// JITStorageGetValueMultiEmitter emits one arbitrary-record bulk storage read.
type JITStorageGetValueMultiEmitter func(*JITContext, JITValueDesc, JITValueDesc, JITValueDesc, JITValueDesc) JITValueDesc

// JITContext is the central structure for descriptor-based JIT compilation.
// W is a self-reference for backward compatibility with hand-written emitters
// that use ctx.W.EmitXxx() (from the pre-consolidation JITWriter era).
type JITContext struct {
	CompileScope        Scmer // request cancellation; compile-time only, never retained by native code
	CompileContext      context.Context
	lambdaTemplates     *[]Scmer // private to one parent compilation, shared by buffer retries
	lambdaTemplateIndex int
	W                   *JITContext    // self-reference (backward compat for ctx.W.Emit calls)
	Ptr                 unsafe.Pointer // current write pointer (into mmap memory)
	End                 unsafe.Pointer // page end minus reserve
	Start               unsafe.Pointer // page start for position calculation

	Labels []int32
	Fixups []JITFixup

	Env        *JITEnv
	FreeRegs   uint64
	AllRegs    uint64 // original set of all allocatable registers (for spilling)
	FreeFPRegs uint64
	AllFPRegs  uint64
	SliceBase  Reg // register holding the args slice pointer (for variable-index access)
	// RegisterBank is supplied by the architecture backend. Generated register
	// plans contain abstract colors and never name a physical register.
	RegisterBank   JITRegisterBank
	FPRegisterBank JITRegisterBank
	// Architecture register roles let common lowering describe placement without
	// depending on one instruction set's register names.
	StackReg           Reg
	FrameReg           Reg
	ScratchReg         Reg
	ResultPtrReg       Reg
	ResultAuxReg       Reg
	NativeFloatCompare bool
	LastIntReg         Reg
	HasFrame           bool
	// OriginalArgsOff stores the incoming variadic slice data pointer in the
	// invocation-local frame.
	// Optimized local frames may repurpose SliceBase, while hidden GC roots still
	// live after the source-level arguments in the original Go-owned slice.
	OriginalArgsOff int32
	// ClosureFuncOff roots the incoming Go funcval in the native frame. Capture
	// descriptors address inline Scmer values directly relative to this pointer.
	ClosureFuncOff int32
	// RuntimeEnvOff roots the invocation's lexical *Env loaded directly from the
	// Proc funcval. Runtime symbol resolution therefore follows rebound closures
	// without allocating or boxing an environment value.
	RuntimeEnvOff  int32
	UsesRuntimeEnv bool
	// CurrentFuncOff roots the incoming Go funcval for recursive calls. Reusing
	// the funcval, rather than a bare code pointer, preserves closure captures.
	CurrentFuncOff int32
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
	LocalSlotCount        int
	HiddenArgs            []JITHiddenArg
	RuntimeEnv            Scmer
	RecursiveLambdas      bool
	ActiveBuiltinEmitters map[*Declaration]uint16
	NeedsStableArgs       bool
	// StorageInputsInRegisters is enabled for the optimistic first emission of
	// a finalized storage reader. Any Go-call boundary aborts that emission and
	// retries with rooted stack homes; a call-free getter keeps its incoming ABI
	// values resident throughout the generated loop.
	StorageInputsInRegisters bool
	StackPhiTargets          bool
	SelfSymbols              map[Symbol]struct{}
	DefiningSymbol           Symbol
	SelfLoopLabel            JITLabel
	HasSelfLoop              bool
	SelfParamCount           int
	RegOwners                [64]*JITValueDesc // register → owner descriptor (nil = untracked)
	// DeferredRegMoves is the physical half of descriptor-level lazy placement.
	// A public register move changes the logical location immediately, but its
	// bytes are held until a non-move instruction, control-flow boundary, or
	// safepoint needs the value. Keeping this state in JITContext lets nested
	// inline emitters collapse each other's hand-off moves.
	DeferredRegMoves jitDeferredRegMoves
	// At most one comparison can own the architecture's flags. Its reserved
	// register is written only when an instruction or value consumer needs it.
	lazyFlags    jitBooleanFlags
	flagsSerial  uint32
	branchSerial uint64 // emitted branches, used to prove a return has no runtime merge
	// registerInstructionDepth suppresses the conservative full barrier in the
	// raw byte writer while an architecture emitter with an explicit use/def
	// contract is encoding one instruction.
	registerInstructionDepth uint8
	// DynamicSP is the temporary distance below the static frame bottom. It
	// covers pushed live registers, variadic arrays, and the Go call area.
	DynamicSP    int32
	MaxDynamicSP int32
	// StackRoots contains pointer words that are live in the static frame at the
	// current emission point. Safepoints copy this set before sibling control
	// flow can restore a different allocator state.
	StackRoots map[jitStackRoot]struct{}
	// FrameRoots accumulates permanent pointer slots as they are first emitted.
	// Unlike StackRoots it is shared by branch contexts and never removes slots,
	// avoiding a second traversal over every safepoint after code generation.
	FrameRoots map[jitStackRoot]struct{}
	// SafepointRootScratch is reused while encoding snapshots; snapshots own
	// their bitmap storage and never reference this temporary root list.
	SafepointRootScratch []jitStackRoot
	Safepoints           []jitSafepoint
	Coverage             JITCoverage

	// Stack frame: emitter locals use [RSP + offset], while register spills use
	// [RBP - offset]. The two zones cannot overlap because the patched frame size
	// is MaxBPOffset + MaxSpillOffset. Epilog: leave; ret.
	BPOffset       int32 // current stack allocation point (grows on alloc, shrinks on free)
	MaxBPOffset    int32 // local-zone high-water mark
	SpillOffset    int32 // current spill-zone allocation point below RBP
	MaxSpillOffset int32 // spill-zone high-water mark

	ProtectedRegs       uint64  // bitmask of registers that must not be spilled
	ProtectedRegCounts  [64]int // per-register protection refcount (supports nested protection)
	RegisterHomeCost    [64]uint16
	RegisterHomeID      [64]uint16
	PinnedRegisterHomes uint64
	nextRegisterHomeID  uint16
	nextDescID          uint32
	descOwners          map[uint32]*JITValueDesc
	descSpills          map[uint32]descSpillMeta
	// ConstRoots holds pointer payloads from LocImm Scmer values that were
	// materialized into machine code immediates. Keeping these pointers in a
	// Go heap object reachable from JITEntryPoint prevents GC from reclaiming
	// referenced heap data while JIT code may still dereference it.
	ConstRoots []unsafe.Pointer
	rootSet    map[unsafe.Pointer]struct{}
	EntryRoots []*JITEntryPoint
	entrySet   map[*JITEntryPoint]struct{}
	Arena      *jitArena // owning arena for source map entries
}

var jitStorageNeedsStableInputs = &struct{}{}

// JITRegisterBank describes the long-lived general-purpose registers offered
// by an architecture backend. Registers are ordered by suitability for values
// crossing control-flow edges; constrained ABI registers therefore come last.
type JITRegisterBank struct {
	Registers        [16]Reg
	Count            uint8
	TemporaryReserve uint8
}

// JITRegisterSlot is an architecture-independent, statically colored value
// bundle. jitgen constructs these slots offline from Go SSA; runtime emission
// only maps them onto the register bank offered by the current architecture.
// Cost estimates the repeated load/store traffic avoided by retaining a slot.
// Width describes the logical bundle, whereas Lanes describes the words still
// required after runtime type information has folded tags or other components.
type JITRegisterSlot struct {
	Color uint8
	Width uint8
	Class JITRegisterClass
	// Lanes selects the words which still need physical storage after dynamic
	// type folding. Zero means all Width lanes for compact generated literals.
	Lanes uint8
	Cost  uint16
}

// JITRegisterPlan is emitted by jitgen. It is deliberately a fixed-size value:
// runtime emission only maps the preplanned slots and never rebuilds SSA or an
// interference graph.
type JITRegisterPlan struct {
	Slots [16]JITRegisterSlot
	Count uint8
}

// JITRegisterHomes maps architecture-independent colors to physical registers.
// Its fixed-size representation keeps JIT emission allocation-free.
type JITRegisterHomes struct {
	Registers [16]Reg
	Available uint16
	OwnedRegs uint64
	Evicted   [16]jitRegisterHomeEviction
	Evictions uint8
}

type jitRegisterHomeEviction struct {
	owner    *JITValueDesc
	original JITValueDesc
	regs     [3]Reg
	width    uint8
	offset   int32
	cost     uint16
	homeID   uint16
}

// AllocRegisterHomes retains the most valuable planned colors while preserving
// the backend's temporary-register budget. Excess colors keep their stack home.
//
// A selected register is protected for the lifetime of this nested emitter,
// but protection is not descriptor ownership. Protection prevents ordinary
// temporaries from taking the register; BindReg later attaches the currently
// live SSA value. This distinction is required because graph colors are reused
// by values with non-overlapping live ranges.
func (ctx *JITContext) AllocRegisterHomes(plan JITRegisterPlan) JITRegisterHomes {
	var homes JITRegisterHomes
	if plan.Count == 0 {
		return homes
	}
	var budgets [2]int
	for class, bank := range [...]JITRegisterBank{ctx.RegisterBank, ctx.FPRegisterBank} {
		all, free := ctx.AllRegs, ctx.FreeRegs
		if JITRegisterClass(class) == JITRegisterClassFP {
			all, free = ctx.AllFPRegs, ctx.FreeFPRegs
		}
		for index := uint8(0); index < bank.Count; index++ {
			reg := bank.Registers[index]
			bit := uint64(1) << uint(reg)
			if all&bit != 0 && free&bit != 0 && ctx.ProtectedRegs&bit == 0 {
				budgets[class]++
			}
		}
		budgets[class] -= int(bank.TemporaryReserve)
	}
	for slotIndex := uint8(0); slotIndex < plan.Count; slotIndex++ {
		slot := plan.Slots[slotIndex]
		class := slot.Class
		bank := ctx.RegisterBank
		all, free := ctx.AllRegs, ctx.FreeRegs
		if class == JITRegisterClassFP {
			bank, all, free = ctx.FPRegisterBank, ctx.AllFPRegs, ctx.FreeFPRegs
		}
		if slot.Width == 0 || slot.Width > 3 || int(slot.Color)+int(slot.Width) > len(homes.Registers) {
			continue
		}
		lanes := slot.Lanes
		if lanes == 0 {
			lanes = uint8(1<<slot.Width) - 1
		}
		lanes &= uint8(1<<slot.Width) - 1
		laneCount := 0
		for lane := uint8(0); lane < slot.Width; lane++ {
			if lanes&(1<<lane) != 0 {
				laneCount++
			}
		}
		for class == JITRegisterClassGPR && laneCount > budgets[class] {
			homeID := ctx.cheapestEvictableRegisterHome(slot.Cost)
			if homeID == 0 {
				break
			}
			evicted, ok := ctx.evictRegisterHome(homeID)
			if !ok {
				break
			}
			homes.Evicted[homes.Evictions] = evicted
			homes.Evictions++
			budgets[class] += int(evicted.width)
		}
		if laneCount > budgets[class] {
			continue
		}
		if class == JITRegisterClassFP {
			free = ctx.FreeFPRegs
		} else {
			free = ctx.FreeRegs
		}
		var selected [3]Reg
		selectedCount := uint8(0)
		for index := uint8(0); index < bank.Count && int(selectedCount) < laneCount; index++ {
			reg := bank.Registers[index]
			bit := uint64(1) << uint(reg)
			if all&bit == 0 || free&bit == 0 || ctx.ProtectedRegs&bit != 0 {
				continue
			}
			selected[selectedCount] = reg
			selectedCount++
		}
		if int(selectedCount) != laneCount {
			continue
		}
		ctx.nextRegisterHomeID++
		if ctx.nextRegisterHomeID == 0 {
			ctx.nextRegisterHomeID++
		}
		selectedLane := uint8(0)
		for lane := uint8(0); lane < slot.Width; lane++ {
			if lanes&(1<<lane) == 0 {
				continue
			}
			reg := selected[selectedLane]
			selectedLane++
			bit := uint64(1) << uint(reg)
			if class == JITRegisterClassFP {
				ctx.FreeFPRegs &^= bit
			} else {
				ctx.FreeRegs &^= bit
			}
			ctx.ProtectReg(reg)
			ctx.RegisterHomeCost[reg] = slot.Cost
			ctx.RegisterHomeID[reg] = ctx.nextRegisterHomeID
			homes.Registers[int(slot.Color)+int(lane)] = reg
			homes.Available |= 1 << (slot.Color + lane)
			homes.OwnedRegs |= bit
		}
		budgets[class] -= laneCount
	}
	return homes
}

func (ctx *JITContext) ReleaseRegisterHomes(homes JITRegisterHomes) {
	for reg := Reg(0); reg < jitRegisterCount; reg++ {
		bit := uint64(1) << uint(reg)
		if homes.OwnedRegs&bit == 0 {
			continue
		}
		ctx.RegisterHomeCost[reg] = 0
		ctx.RegisterHomeID[reg] = 0
		ctx.UnprotectReg(reg)
		ctx.FreeReg(reg)
	}
	for index := int(homes.Evictions) - 1; index >= 0; index-- {
		ctx.restoreRegisterHome(homes.Evicted[index])
	}
}

func (ctx *JITContext) cheapestEvictableRegisterHome(maxCost uint16) uint16 {
	bestCost := maxCost
	bestID := uint16(0)
	for index := uint8(0); index < ctx.RegisterBank.Count; index++ {
		reg := ctx.RegisterBank.Registers[index]
		bit := uint64(1) << uint(reg)
		id := ctx.RegisterHomeID[reg]
		cost := ctx.RegisterHomeCost[reg]
		if id == 0 || cost >= bestCost || ctx.PinnedRegisterHomes&bit != 0 || ctx.ProtectedRegCounts[reg] != 1 || ctx.RegOwners[reg] == nil {
			continue
		}
		valid := true
		for other := uint8(0); other < ctx.RegisterBank.Count; other++ {
			otherReg := ctx.RegisterBank.Registers[other]
			if ctx.RegisterHomeID[otherReg] != id {
				continue
			}
			otherBit := uint64(1) << uint(otherReg)
			if ctx.PinnedRegisterHomes&otherBit != 0 || ctx.ProtectedRegCounts[otherReg] != 1 || ctx.RegOwners[otherReg] == nil {
				valid = false
				break
			}
		}
		if valid {
			bestCost, bestID = cost, id
		}
	}
	return bestID
}

func (ctx *JITContext) evictRegisterHome(homeID uint16) (jitRegisterHomeEviction, bool) {
	var eviction jitRegisterHomeEviction
	for index := uint8(0); index < ctx.RegisterBank.Count; index++ {
		reg := ctx.RegisterBank.Registers[index]
		if ctx.RegisterHomeID[reg] != homeID {
			continue
		}
		if eviction.width == 0 {
			eviction.owner = ctx.RegOwners[reg]
			eviction.original = *eviction.owner
			eviction.cost = ctx.RegisterHomeCost[reg]
			eviction.homeID = homeID
		}
		if eviction.width >= uint8(len(eviction.regs)) || ctx.RegOwners[reg] == nil || ctx.RegOwners[reg].ID != eviction.owner.ID {
			return jitRegisterHomeEviction{}, false
		}
		eviction.regs[eviction.width] = reg
		eviction.width++
	}
	if eviction.width == 0 {
		return jitRegisterHomeEviction{}, false
	}
	words := jitDescWordCount(eviction.original)
	if words != int(eviction.width) {
		return jitRegisterHomeEviction{}, false
	}
	eviction.offset = ctx.AllocSpill(int32(words * 8))
	regs := jitDescRegs(eviction.original)
	for word, reg := range regs {
		ctx.EmitStoreRegMem(reg, ctx.FrameReg, eviction.offset+int32(word*8))
		ctx.setStackPointer(jitStackRootFrameBP, eviction.offset+int32(word*8), jitValueWordIsPointer(eviction.original, int32(word)))
		bit := uint64(1) << uint(reg)
		ctx.RegOwners[reg] = nil
		ctx.FreeRegs |= bit
		ctx.UnprotectReg(reg)
		ctx.RegisterHomeCost[reg] = 0
		ctx.RegisterHomeID[reg] = 0
	}
	eviction.owner.Reg, eviction.owner.Reg2, eviction.owner.Reg3 = 0, 0, 0
	eviction.owner.StackOff = eviction.offset
	switch words {
	case 1:
		eviction.owner.Loc = LocStack
	case 2:
		eviction.owner.Loc = LocStackPair
	case 3:
		eviction.owner.Loc = LocStackTriple
	}
	if eviction.owner.ID != 0 {
		if ctx.descSpills == nil {
			ctx.descSpills = make(map[uint32]descSpillMeta)
		}
		ctx.descSpills[eviction.owner.ID] = descSpillMeta{loc: eviction.owner.Loc, stackOff: eviction.offset}
	}
	return eviction, true
}

func (ctx *JITContext) restoreRegisterHome(eviction jitRegisterHomeEviction) {
	for word, reg := range jitDescRegs(eviction.original) {
		ctx.EmitMovRegMem(reg, ctx.FrameReg, eviction.offset+int32(word*8))
		bit := uint64(1) << uint(reg)
		ctx.FreeRegs &^= bit
		ctx.ProtectReg(reg)
		ctx.RegisterHomeCost[reg] = eviction.cost
		ctx.RegisterHomeID[reg] = eviction.homeID
	}
	*eviction.owner = eviction.original
	for _, reg := range jitDescRegs(eviction.original) {
		ctx.RegOwners[reg] = eviction.owner
	}
	if eviction.owner.ID != 0 {
		if ctx.descSpills == nil {
			ctx.descSpills = make(map[uint32]descSpillMeta)
		}
		ctx.descSpills[eviction.owner.ID] = descSpillMeta{
			loc: eviction.original.Loc, reg: eviction.original.Reg,
			reg2: eviction.original.Reg2, reg3: eviction.original.Reg3,
		}
	}
}

func jitDescWordCount(desc JITValueDesc) int {
	switch desc.Loc {
	case LocReg:
		return 1
	case LocRegPair:
		return 2
	case LocRegTriple:
		return 3
	default:
		return 0
	}
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
// generation can render sibling BBs from identical allocator state. It is not
// runtime state and emits no save/restore instructions. Machine values which
// must survive a runtime branch need explicit homes before this bookkeeping is
// rewound; restoring only these Go fields cannot resurrect a runtime register.
type jitAllocStateSnapshot struct {
	freeRegs            uint64
	freeFPRegs          uint64
	protectedRegs       uint64
	protectedRegCounts  [64]int
	registerHomeCost    [64]uint16
	registerHomeID      [64]uint16
	pinnedRegisterHomes uint64
	regOwnerIDs         [64]uint32
	ownerValues         []jitOwnerSnapshot
	firstNewDescID      uint32
	spillOffset         int32
	descSpills          []jitDescSpillSnapshot
	stackRoots          []jitStackRoot
	dynamicSP           int32
}

type jitOwnerSnapshot struct {
	id    uint32
	value JITValueDesc
}

type jitDescSpillSnapshot struct {
	id    uint32
	value descSpillMeta
}

func (ctx *JITContext) SnapshotAllocState() jitAllocStateSnapshot {
	// Sibling renderers rewind allocator metadata, not physical alias state.
	// Materialize the fallthrough state before taking the branch snapshot.
	ctx.FlushRegisterMoves()
	s := jitAllocStateSnapshot{
		freeRegs:            ctx.FreeRegs,
		freeFPRegs:          ctx.FreeFPRegs,
		protectedRegs:       ctx.ProtectedRegs,
		protectedRegCounts:  ctx.ProtectedRegCounts,
		registerHomeCost:    ctx.RegisterHomeCost,
		registerHomeID:      ctx.RegisterHomeID,
		pinnedRegisterHomes: ctx.PinnedRegisterHomes,
		firstNewDescID:      ctx.nextDescID + 1,
		spillOffset:         ctx.SpillOffset,
		dynamicSP:           ctx.DynamicSP,
	}
	if len(ctx.descOwners) != 0 {
		for id, owner := range ctx.descOwners {
			if owner == nil || owner.Loc == LocNone {
				delete(ctx.descOwners, id)
				continue
			}
			s.ownerValues = append(s.ownerValues, jitOwnerSnapshot{id: id, value: *owner})
		}
	}
	for r := Reg(0); r < jitRegisterCount; r++ {
		if owner := ctx.RegOwners[r]; owner != nil {
			s.regOwnerIDs[r] = owner.ID
		}
	}
	if len(ctx.descSpills) != 0 {
		for k, v := range ctx.descSpills {
			s.descSpills = append(s.descSpills, jitDescSpillSnapshot{id: k, value: v})
		}
	}
	if len(ctx.StackRoots) != 0 {
		for root := range ctx.StackRoots {
			s.stackRoots = append(s.stackRoots, root)
		}
	}
	return s
}

func (ctx *JITContext) RestoreAllocState(s jitAllocStateSnapshot) {
	ctx.FlushRegisterMoves()
	ctx.FreeRegs = s.freeRegs
	ctx.FreeFPRegs = s.freeFPRegs
	ctx.ProtectedRegs = s.protectedRegs
	ctx.ProtectedRegCounts = s.protectedRegCounts
	ctx.RegisterHomeCost = s.registerHomeCost
	ctx.RegisterHomeID = s.registerHomeID
	ctx.PinnedRegisterHomes = s.pinnedRegisterHomes
	ctx.SpillOffset = s.spillOffset
	// Descriptor identities are global to one emitted function. Restoring an
	// older basic-block snapshot must not make later descriptors reuse IDs whose
	// spill metadata was already emitted on a sibling path.
	ctx.DynamicSP = s.dynamicSP

	for id := range ctx.descOwners {
		if id >= s.firstNewDescID {
			delete(ctx.descOwners, id)
		}
	}
	for _, saved := range s.ownerValues {
		owner := ctx.descOwners[saved.id]
		if owner == nil {
			owner = &JITValueDesc{}
			ctx.descOwners[saved.id] = owner
		}
		*owner = saved.value
	}
	for r := Reg(0); r < jitRegisterCount; r++ {
		id := s.regOwnerIDs[r]
		if id == 0 {
			ctx.RegOwners[r] = nil
			continue
		}
		ctx.RegOwners[r] = ctx.descOwners[id]
	}

	clear(ctx.descSpills)
	if len(s.descSpills) != 0 {
		if ctx.descSpills == nil {
			ctx.descSpills = make(map[uint32]descSpillMeta, len(s.descSpills))
		}
		for _, saved := range s.descSpills {
			ctx.descSpills[saved.id] = saved.value
		}
	}
	clear(ctx.StackRoots)
	if len(s.stackRoots) != 0 {
		if ctx.StackRoots == nil {
			ctx.StackRoots = make(map[jitStackRoot]struct{}, len(s.stackRoots))
		}
		for _, root := range s.stackRoots {
			ctx.StackRoots[root] = struct{}{}
		}
	}
}

// BBDescriptor stores invocation-local state for a canonical block renderer.
// The generator plans phi layout and consumer/liveness facts offline. Concrete
// register placement and the decision to emit a branch remain runtime facts.
type BBDescriptor struct {
	Render   func() JITValueDesc
	Rendered bool
	// PhiBase is the frame-relative base of the block's canonical phi homes.
	PhiBase int32
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

func (ctx *JITContext) addDynamicStack(size int32) {
	ctx.DynamicSP += size
	if ctx.DynamicSP > ctx.MaxDynamicSP {
		ctx.MaxDynamicSP = ctx.DynamicSP
	}
}

// FreeStack releases size bytes from the stack frame.
func (ctx *JITContext) FreeStack(size int32) {
	newOffset := ctx.BPOffset - size
	physicalStart := newOffset - ctx.DynamicSP
	physicalEnd := ctx.BPOffset - ctx.DynamicSP
	ctx.clearStackRootRange(jitStackRootFrameSP, physicalStart, physicalEnd)
	ctx.BPOffset -= size
}

// clearStackRootRange removes only released stack addresses. Small call-frame
// releases must not scan every live root in a large enclosing expression.
func (ctx *JITContext) clearStackRootRange(base jitStackRootBase, start, end int32) {
	if int64(end)-int64(start) < int64(len(ctx.StackRoots)) {
		for offset := start; offset < end; offset++ {
			delete(ctx.StackRoots, jitStackRoot{base: base, offset: offset})
		}
		return
	}
	for root := range ctx.StackRoots {
		if root.base == base && root.offset >= start && root.offset < end {
			delete(ctx.StackRoots, root)
		}
	}
}

func (ctx *JITContext) setStackPointer(base jitStackRootBase, offset int32, pointer bool) {
	root := jitStackRoot{base: base, offset: offset}
	if pointer {
		if ctx.StackRoots == nil {
			ctx.StackRoots = make(map[jitStackRoot]struct{})
		}
		ctx.StackRoots[root] = struct{}{}
		if base == jitStackRootFrameBP || base == jitStackRootFrameSP && offset >= 0 {
			if ctx.FrameRoots == nil {
				ctx.FrameRoots = make(map[jitStackRoot]struct{})
			}
			ctx.FrameRoots[root] = struct{}{}
		}
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

// TrackImm retains a constant's GC root and any embedded native dependency.
func (ctx *JITContext) TrackImm(v Scmer) {
	if ctx == nil {
		return
	}
	if v.IsProc() && v.Proc() != nil {
		ctx.TrackEntry(v.Proc().Compiled)
	} else if v.GetTag() == tagAny {
		if entry, ok := v.Any().(*JITEntryPoint); ok {
			ctx.TrackEntry(entry)
		}
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

// TrackEntry retains a JIT entry point whose native address is embedded into
// this code. This typed edge is also the ownership edge for its arena cleanup.
func (ctx *JITContext) TrackEntry(entry *JITEntryPoint) {
	if ctx == nil || entry == nil {
		return
	}
	if ctx.entrySet == nil {
		ctx.entrySet = make(map[*JITEntryPoint]struct{}, 8)
	}
	if _, exists := ctx.entrySet[entry]; exists {
		return
	}
	ctx.entrySet[entry] = struct{}{}
	ctx.EntryRoots = append(ctx.EntryRoots, entry)
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
	for rr := Reg(0); rr < jitRegisterCount; rr++ {
		bit := uint64(1 << uint(rr))
		all := ctx.AllRegs
		if rr >= RegX0 {
			all = ctx.AllFPRegs
		}
		if (all & bit) == 0 {
			continue
		}
		if (ctx.ProtectedRegs & bit) != 0 {
			continue
		}
		// FreeReg deliberately keeps a dead register unavailable while a deferred
		// destination still aliases its physical contents. Reclamation must honor
		// that reservation; releaseUnusedDeferredSources returns it after the last
		// alias has been materialized.
		if ctx.DeferredRegMoves.held&bit != 0 {
			continue
		}
		owner := ctx.RegOwners[rr]
		if owner == nil {
			if rr >= RegX0 {
				ctx.FreeFPRegs |= bit
			} else {
				ctx.FreeRegs |= bit
			}
			continue
		}
		valid := false
		switch owner.Loc {
		case LocReg, LocFPReg, LocFlags:
			valid = owner.Reg == rr
		case LocRegPair:
			valid = owner.Reg == rr || owner.Reg2 == rr
		case LocRegTriple:
			valid = owner.Reg == rr || owner.Reg2 == rr || owner.Reg3 == rr
		}
		if !valid {
			ctx.RegOwners[rr] = nil
			if rr >= RegX0 {
				ctx.FreeFPRegs |= bit
			} else {
				ctx.FreeRegs |= bit
			}
		}
	}
}

// JITRegisterBoundary is a reversible allocator boundary around nested
// emission. It records where the outer machine program's live register values
// were spilled; Restore reloads those exact registers and reinstates the
// allocator metadata captured before the nested emitter ran.
//
// Fixed arrays are intentional. This operation sits on the JIT compilation
// path and must not allocate merely because a callback emitter needs temporary
// freedom on the register bank.
type JITRegisterBoundary struct {
	alloc jitAllocStateSnapshot
	regs  [64]Reg
	offs  [64]int32
	count uint8
}

// JITRegisterBoundaryOptions controls which outer placements remain visible to
// a nested emitter. ResultRegs are deliberately left in place so an inner
// producer can write directly to its caller-selected destination. ReleaseHomes
// trades residency for register capacity: false lets a nested weighted plan
// evict only cheaper homes; true spills every outer home and is appropriate for
// callbacks whose body needs the complete register bank.
type JITRegisterBoundaryOptions struct {
	ResultRegs   []Reg
	ReleaseHomes bool
}

// PreserveOuterRegs makes nested emission independent of ordinary temporary
// registers whose identities have already been embedded in outer control flow.
// Weighted homes remain resident so the nested plan can compare costs and evict
// them selectively. PreserveRegisters with ReleaseHomes is the stronger form
// for callbacks which need the complete register bank.
func (ctx *JITContext) PreserveOuterRegs() JITRegisterBoundary {
	return ctx.PreserveOuterRegsExcept()
}

// PreserveOuterRegsExcept preserves the outer allocator state while leaving
// result registers in place for a nested emitter to overwrite directly.
func (ctx *JITContext) PreserveOuterRegsExcept(resultRegs ...Reg) JITRegisterBoundary {
	return ctx.PreserveRegisters(JITRegisterBoundaryOptions{ResultRegs: resultRegs})
}

// PreserveRegisters opens a reusable register-allocation boundary. This is the
// common mechanism for generated builtin callbacks, parser actions and manual
// emitters such as regex walkers; those callers must not each invent their own
// ProtectReg/spill convention.
func (ctx *JITContext) PreserveRegisters(options JITRegisterBoundaryOptions) JITRegisterBoundary {
	p := JITRegisterBoundary{alloc: ctx.SnapshotAllocState()}
	var resultMask uint64
	for _, r := range options.ResultRegs {
		resultMask |= 1 << uint(r)
	}
	for r := Reg(0); r < jitRegisterCount; r++ {
		all, free := ctx.AllRegs, ctx.FreeRegs
		if r >= RegX0 {
			all, free = ctx.AllFPRegs, ctx.FreeFPRegs
		}
		if (all&(1<<uint(r))) == 0 || (free&(1<<uint(r))) != 0 {
			continue
		}
		if resultMask&(1<<uint(r)) != 0 {
			continue
		}
		// Weighted homes are the interface between the offline plan and nested
		// one-pass emission. Keep them resident here; an inner register plan may
		// selectively evict cheaper homes and restores their exact registers when
		// it returns. Ordinary temporaries still take the conservative save path.
		if !options.ReleaseHomes && ctx.RegisterHomeID[r] != 0 {
			continue
		}
		off := ctx.AllocSpill(8)
		if r >= RegX0 {
			ctx.EmitStoreFPRegMem(r, ctx.FrameReg, off)
		} else {
			ctx.EmitStoreRegMem(r, ctx.FrameReg, off)
		}
		if r < RegX0 && ctx.regHoldsPointer(r) {
			ctx.setStackPointer(jitStackRootFrameBP, off, true)
		}
		p.regs[p.count] = r
		p.offs[p.count] = off
		p.count++
		ctx.RegOwners[r] = nil
		if r >= RegX0 {
			ctx.FreeFPRegs |= 1 << uint(r)
		} else {
			ctx.FreeRegs |= 1 << uint(r)
		}
		if options.ReleaseHomes {
			// The snapshot retains the home metadata for Restore. Leaving it on a
			// now-free register would make a nested plan mistake its own register
			// for an occupied outer bundle.
			ctx.RegisterHomeCost[r] = 0
			ctx.RegisterHomeID[r] = 0
		}
	}
	homeMask := uint64(0)
	for r := Reg(0); r < jitRegisterCount; r++ {
		if !options.ReleaseHomes && ctx.RegisterHomeID[r] != 0 {
			homeMask |= 1 << uint(r)
		}
	}
	// Released homes must not leave pin bits behind: a nested allocation may
	// legitimately use those physical registers until this boundary is restored.
	ctx.PinnedRegisterHomes &= homeMask | resultMask
	ctx.ProtectedRegs &= resultMask | homeMask
	for r := Reg(0); r < jitRegisterCount; r++ {
		if (resultMask|homeMask)&(1<<uint(r)) == 0 {
			ctx.ProtectedRegCounts[r] = 0
		}
	}
	return p
}

func (ctx *JITContext) RestoreOuterRegs(p JITRegisterBoundary) {
	p.Restore(ctx)
}

// Restore closes a register boundary. Nested values must already have been
// committed to their result/stack destinations; restoring intentionally
// discards all allocator ownership created inside the boundary.
func (p JITRegisterBoundary) Restore(ctx *JITContext) {
	for i := uint8(0); i < p.count; i++ {
		if p.regs[i] >= RegX0 {
			ctx.EmitLoadFPRegMem(p.regs[i], ctx.FrameReg, p.offs[i])
		} else {
			ctx.EmitMovRegMem(p.regs[i], ctx.FrameReg, p.offs[i])
		}
	}
	ctx.RestoreAllocState(p.alloc)
}

// AllocReg picks a free register from the bitmap and marks it used.
// If no registers are free, spills the highest-numbered in-use register
// to a pre-allocated buffer and returns it.
func (ctx *JITContext) AllocReg() Reg {
	// Sanitize stale owner links: if an owner descriptor no longer claims this
	// hardware register, drop the stale owner edge and mark the register free.
	for rr := Reg(0); rr <= jitLastGPReg; rr++ {
		if (ctx.AllRegs & (1 << uint(rr))) == 0 {
			continue
		}
		owner := ctx.RegOwners[rr]
		if owner == nil {
			continue
		}
		valid := false
		switch owner.Loc {
		case LocReg, LocFlags:
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
	ctx.materializeBooleanFlags()
	// Note: completely untracked in-use registers must NOT be reused here,
	// as they may still be referenced by emitted code paths.
	spillable := ctx.AllRegs &^ ctx.FreeRegs &^ ctx.ProtectedRegs
	var r Reg = 0xFF
	pairSpill := false
	tripleSpill := false
	var spillR1, spillR2, spillR3 Reg
	for bit := int(jitLastGPReg); bit >= 0; bit-- {
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
		for rr := Reg(0); rr <= jitLastGPReg; rr++ {
			if ctx.RegOwners[rr] != nil {
				ownerMask |= 1 << uint(rr)
				o := ctx.RegOwners[rr]
				ownerDump += fmt.Sprintf(" r%d(loc=%d reg=%d reg2=%d reg3=%d)", rr, o.Loc, o.Reg, o.Reg2, o.Reg3)
			}
		}
		panic(fmt.Sprintf("jit: register spill required (fallback) free=%#x all=%#x prot=%#x owners=%#x%s", ctx.FreeRegs, ctx.AllRegs, ctx.ProtectedRegs, ownerMask, ownerDump))
	}

	owner := ctx.RegOwners[r]
	// A scalar register is only a payload while it remains inside generated
	// code.  When its type is known and it is not a relocatable Go pointer, an
	// unused FP register is a cheaper spill home than memory.  The original
	// register class remains attached to the descriptor so a later FP spill is
	// restored to the register file required by its consumer.  Full Scmer pairs
	// are deliberately excluded: their pointer word must remain visible to the
	// Go stack map unless NoHeapPointer proves otherwise, and splitting pairs
	// here would make their ownership non-atomic.
	if !pairSpill && !tripleSpill && jitScalarCanUseFPOverflow(owner) {
		availableFP := ctx.FreeFPRegs &^ ctx.ProtectedRegs
		if availableFP != 0 {
			fp := Reg(bits.TrailingZeros64(availableFP))
			ctx.FreeFPRegs &^= 1 << uint(fp)
			ctx.EmitMovGPRToFP(fp, r)
			owner.Loc = LocFPReg
			owner.Reg = fp
			ctx.RegOwners[r] = nil
			ctx.RegOwners[fp] = owner
			if owner.ID != 0 {
				if ctx.descSpills == nil {
					ctx.descSpills = make(map[uint32]descSpillMeta)
				}
				ctx.descSpills[owner.ID] = descSpillMeta{loc: LocFPReg, reg: fp}
			}
			return r
		}
	}
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
	ctx.setStackPointer(jitStackRootFrameBP, stackOff, owner.RelocatablePointer)

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

// jitScalarCanUseFPOverflow recognizes values whose sole machine word is data,
// never a Go pointer. Type knowledge is intentionally not used as a blanket
// permission: string/slice/procedure scalar views may carry addresses. Numeric,
// boolean and internal ordinal payloads are safe only when the producer also
// proves NoHeapPointer. RelocatablePointer is an explicit veto: internal
// runtime addresses may be represented as Scheme ints, but their register
// homes remain part of the Go pointer/lifetime contract.
func jitScalarCanUseFPOverflow(value *JITValueDesc) bool {
	if value == nil || value.Loc != LocReg || !value.NoHeapPointer || value.RelocatablePointer {
		return false
	}
	switch value.Type {
	case tagInt, tagFloat, tagBool, tagDate, tagNthLocalVar:
		return true
	default:
		return false
	}
}

// AllocFPReg allocates a scalar floating-point register independently from the
// GPR allocator. FP values cannot consume pointer-bearing GPR homes, and vice
// versa; keeping the files separate also maps directly to non-amd64 backends.
func (ctx *JITContext) AllocFPReg() Reg {
	available := ctx.FreeFPRegs &^ ctx.ProtectedRegs
	if available != 0 {
		bit := available & -available
		ctx.FreeFPRegs &^= bit
		return Reg(bits.TrailingZeros64(bit))
	}
	spillable := ctx.AllFPRegs &^ ctx.FreeFPRegs &^ ctx.ProtectedRegs
	for bitIndex := int(jitLastFPReg); bitIndex >= int(jitFirstFPReg); bitIndex-- {
		bit := uint64(1) << uint(bitIndex)
		if spillable&bit == 0 {
			continue
		}
		owner := ctx.RegOwners[bitIndex]
		if owner == nil || owner.Loc != LocFPReg || owner.Reg != Reg(bitIndex) {
			continue
		}
		off := ctx.AllocSpill(8)
		ctx.EmitStoreFPRegMem(Reg(bitIndex), ctx.FrameReg, off)
		owner.Loc = LocStack
		owner.StackOff = off
		owner.Reg = 0
		if owner.ID != 0 {
			if ctx.descSpills == nil {
				ctx.descSpills = make(map[uint32]descSpillMeta)
			}
			ctx.descSpills[owner.ID] = descSpillMeta{loc: LocStack, stackOff: off}
		}
		ctx.RegOwners[bitIndex] = nil
		return Reg(bitIndex)
	}
	panic("jit: floating-point register spill required without an owned value")
}

// EnsureDesc restores a descriptor from stack/spill locations to registers.
func (ctx *JITContext) syncDescSpill(desc *JITValueDesc) {
	if desc.ID == 0 || ctx.descSpills == nil {
		return
	}
	if meta, ok := ctx.descSpills[desc.ID]; ok {
		desc.Loc = meta.loc
		desc.MemPtr = 0
		desc.StackOff = meta.stackOff
		desc.Reg = meta.reg
		desc.Reg2 = meta.reg2
		desc.Reg3 = meta.reg3
	}
}

// SyncDesc updates an aliased descriptor after allocator spills without
// materializing stack-backed values into registers.
func (ctx *JITContext) SyncDesc(desc *JITValueDesc) {
	ctx.syncDescSpill(desc)
}

func (ctx *JITContext) EnsureDesc(desc *JITValueDesc) {
	if desc.Loc == LocFlags && desc.FlagsID != 0 {
		ctx.materializeBooleanFlags()
		desc.Loc = LocReg
		desc.FlagsID = 0
	}
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
	case LocClosurePair:
		r1 := ctx.AllocReg()
		r2 := ctx.AllocRegExcept(r1)
		ctx.EmitMovRegMem(ctx.ScratchReg, ctx.StackReg, ctx.ClosureFuncOff)
		captureOffset := int32(unsafe.Offsetof(ProcJIT{}.Context)) + desc.StackOff*16
		ctx.EmitMovRegMem(r1, ctx.ScratchReg, captureOffset)
		ctx.EmitMovRegMem(r2, ctx.ScratchReg, captureOffset+8)
		desc.Loc = LocRegPair
		desc.Reg = r1
		desc.Reg2 = r2
		ctx.BindReg(r1, desc)
		ctx.BindReg(r2, desc)
	case LocStack:
		if desc.RegClass == JITRegisterClassFP {
			ctx.EnsureFPReg(desc)
		} else {
			ctx.EnsureReg(desc)
		}
	case LocFPReg:
		// LocFPReg is also an overflow home for pointer-free scalar payloads.
		// Only native float values are consumed there directly; integer and
		// boolean consumers get their original GPR representation back.
		if desc.RegClass != JITRegisterClassFP {
			ctx.EnsureReg(desc)
		}
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
		case LocReg, LocFPReg:
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
	if ctx.lazyFlags.FlagsID != 0 && ctx.lazyFlags.Reg == r {
		ctx.materializeBooleanFlags()
	}
	heldAsDeferredSource := ctx.releaseDeferredReg(r)
	owner := ctx.RegOwners[r]
	if owner != nil {
		switch owner.Loc {
		case LocReg, LocFPReg:
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
				if other <= jitLastGPReg {
					ctx.RegOwners[other] = nil
				}
				owner.Loc = LocNone
				owner.Reg = 0
				owner.Reg2 = 0
			}
		case LocRegTriple:
			if owner.Reg == r || owner.Reg2 == r || owner.Reg3 == r {
				for _, other := range [...]Reg{owner.Reg, owner.Reg2, owner.Reg3} {
					if other != r && other <= jitLastGPReg {
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
	if !heldAsDeferredSource {
		if r >= RegX0 {
			ctx.FreeFPRegs |= 1 << uint(r)
		} else {
			ctx.FreeRegs |= 1 << uint(r)
		}
	}
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
	if r >= RegX0 {
		ctx.FreeFPRegs &^= 1 << uint(r)
	} else {
		ctx.FreeRegs &^= 1 << uint(r)
	}
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

// AllocRegExceptOperand preserves a lazy right operand in its current
// location. Register operands stay registered; stack operands remain eligible
// for direct memory folding without reserving an otherwise unrelated register.
func (ctx *JITContext) AllocRegExceptOperand(right *JITValueDesc, excluded ...Reg) Reg {
	ctx.SyncDesc(right)
	protected := false
	if right.Loc == LocReg {
		ctx.ProtectReg(right.Reg)
		protected = true
	}
	r := ctx.AllocRegExcept(excluded...)
	if protected {
		ctx.UnprotectReg(right.Reg)
	}
	return r
}

// EnsureReg checks if a descriptor was spilled and restores it.
// If the value is still in a register, this is a no-op.
// If spilled, allocates a new register, emits a load, and updates the desc.
func (ctx *JITContext) EnsureReg(desc *JITValueDesc) {
	if desc.Loc == LocFPReg {
		fp := desc.Reg
		r := ctx.AllocReg()
		ctx.EmitMovFPToGPR(r, fp)
		ctx.FreeReg(fp)
		desc.Loc = LocReg
		desc.Reg = r
		desc.MemPtr = 0
		desc.StackOff = 0
		ctx.BindReg(r, desc)
		return
	}
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

func (ctx *JITContext) EnsureFPReg(desc *JITValueDesc) {
	if desc.Loc == LocFPReg {
		return
	}
	r := ctx.AllocFPReg()
	switch desc.Loc {
	case LocStack:
		base := ctx.StackReg
		if desc.StackOff < 0 {
			base = ctx.FrameReg
		}
		ctx.EmitLoadFPRegMem(r, base, desc.StackOff)
	case LocReg:
		ctx.EmitMovGPRToFP(r, desc.Reg)
		ctx.FreeReg(desc.Reg)
	case LocRegPair:
		ctx.EmitMovGPRToFP(r, desc.Reg2)
	default:
		panic("jit: cannot materialize float in FP register")
	}
	desc.Loc = LocFPReg
	desc.RegClass = JITRegisterClassFP
	desc.Reg = r
	desc.Reg2 = 0
	desc.StackOff = 0
	ctx.BindReg(r, desc)
}

// EmitFloatBinary keeps typed operands and results in the FP register file.
// Constants use the backend scratch register, while SSA operands remain owned
// by their descriptors so chains and loop phis do not bounce through GPRs.
func (ctx *JITContext) EmitFloatBinary(left, right *JITValueDesc, op JITFloatOp) JITValueDesc {
	if left.Loc != LocImm {
		ctx.EnsureFPReg(left)
	}
	if right.Loc != LocImm {
		ctx.EnsureFPReg(right)
	}
	if left.Loc == LocFPReg {
		ctx.ProtectReg(left.Reg)
		defer ctx.UnprotectReg(left.Reg)
	}
	if right.Loc == LocFPReg {
		ctx.ProtectReg(right.Reg)
		defer ctx.UnprotectReg(right.Reg)
	}
	result := JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: ctx.AllocFPReg()}
	ctx.EmitMovToReg(result.Reg, *left)
	rightReg := right.Reg
	if right.Loc == LocImm {
		ctx.EmitMovToReg(RegX1, *right)
		rightReg = RegX1
	}
	switch op {
	case JITFloatAdd:
		ctx.EmitAddFP64(result.Reg, rightReg)
	case JITFloatSub:
		ctx.EmitSubFP64(result.Reg, rightReg)
	case JITFloatMul:
		ctx.EmitMulFP64(result.Reg, rightReg)
	case JITFloatDiv:
		ctx.EmitDivFP64(result.Reg, rightReg)
	default:
		panic("jit: unknown floating-point operation")
	}
	ctx.BindReg(result.Reg, &result)
	return result
}

func (ctx *JITContext) EmitFloatCompare(left, right *JITValueDesc, condition JITCondition) JITValueDesc {
	if left.Loc == LocImm && right.Loc == LocImm {
		x, y := left.Imm.Float(), right.Imm.Float()
		var value bool
		switch condition {
		case CcE:
			value = x == y
		case CcNE:
			value = x != y
		case CcL:
			value = x < y
		case CcLE:
			value = x <= y
		case CcG:
			value = x > y
		case CcGE:
			value = x >= y
		}
		return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(value)}
	}
	if left.Loc != LocImm {
		ctx.EnsureFPReg(left)
	}
	if right.Loc != LocImm {
		ctx.EnsureFPReg(right)
	}
	leftReg := left.Reg
	if left.Loc == LocImm {
		ctx.EmitMovToReg(RegX0, *left)
		leftReg = RegX0
	}
	rightReg := right.Reg
	if right.Loc == LocImm {
		ctx.EmitMovToReg(RegX1, *right)
		rightReg = RegX1
	}
	result := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: ctx.AllocReg()}
	ctx.EmitCmpFP64Setcc(result.Reg, leftReg, rightReg, condition)
	ctx.BindReg(result.Reg, &result)
	return result
}

// FreeDesc releases any registers held by a value descriptor.
func (ctx *JITContext) FreeDesc(desc *JITValueDesc) {
	// Non-owning descriptors (ID==0), e.g. copied call arguments, must not
	// mutate placement/free registers from the original source descriptor.
	if desc.ID == 0 {
		return
	}
	// Allocation can move the canonical owner to a stack or cross-class
	// overflow home while generated code still holds descriptor aliases. Free
	// the current placement, never the stale register named by such an alias.
	ctx.SyncDesc(desc)
	switch desc.Loc {
	case LocReg:
		if desc.Reg <= jitLastGPReg {
			owner := ctx.RegOwners[desc.Reg]
			if owner == nil || owner == desc || (desc.ID != 0 && owner.ID == desc.ID) {
				ctx.FreeReg(desc.Reg)
			}
		}
	case LocFPReg:
		owner := ctx.RegOwners[desc.Reg]
		if owner == nil || owner == desc || (desc.ID != 0 && owner.ID == desc.ID) {
			ctx.FreeReg(desc.Reg)
		}
	case LocRegPair:
		if desc.Reg <= jitLastGPReg {
			owner := ctx.RegOwners[desc.Reg]
			if owner == nil || owner == desc || (desc.ID != 0 && owner.ID == desc.ID) {
				ctx.FreeReg(desc.Reg)
			}
		}
		if desc.Reg2 <= jitLastGPReg {
			owner := ctx.RegOwners[desc.Reg2]
			if owner == nil || owner == desc || (desc.ID != 0 && owner.ID == desc.ID) {
				ctx.FreeReg(desc.Reg2)
			}
		}
	case LocRegTriple:
		for _, r := range [...]Reg{desc.Reg, desc.Reg2, desc.Reg3} {
			if r > jitLastGPReg {
				continue
			}
			owner := ctx.RegOwners[r]
			if owner == nil || owner == desc || (desc.ID != 0 && owner.ID == desc.ID) {
				ctx.FreeReg(r)
			}
		}
	case LocFlags:
		if desc.Reg <= jitLastGPReg {
			owner := ctx.RegOwners[desc.Reg]
			if owner == nil || owner == desc || (desc.ID != 0 && owner.ID == desc.ID) {
				if ctx.hasBooleanFlags(*desc) {
					ctx.lazyFlags = jitBooleanFlags{}
				}
				ctx.FreeReg(desc.Reg)
			}
		}
	case LocStack:
	case LocStackPair:
	case LocStackTriple:
	}
	desc.Loc = LocNone
	desc.MemPtr = 0
	if desc.ID != 0 {
		if owner := ctx.descOwners[desc.ID]; owner != nil {
			owner.Loc = LocNone
			owner.MemPtr = 0
		}
		if ctx.descSpills != nil {
			delete(ctx.descSpills, desc.ID)
		}
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

func jitSyntaxKind(value Scmer) SyntaxKind {
	declaration := DeclarationForValue(value)
	if declaration == nil || !declaration.IsSpecialForm {
		return SyntaxOrdinary
	}
	return declaration.SyntaxKind
}

func jitAddMatchPatternBoundSymbols(pattern Scmer, bound map[Symbol]struct{}) {
	for pattern.IsSourceInfo() {
		pattern = pattern.SourceInfo().value
	}
	if pattern.IsSymbol() {
		symbol := pattern.Symbol()
		switch symbol {
		case "_", "nil", "true", "false":
			return
		}
		bound[symbol] = struct{}{}
		return
	}
	if !pattern.IsSlice() {
		return
	}
	items := pattern.Slice()
	if len(items) == 0 {
		return
	}
	head, hasHead := scmerSymbol(items[0])
	if !hasHead {
		for _, item := range items {
			jitAddMatchPatternBoundSymbols(item, bound)
		}
		return
	}
	switch head {
	case "quote", "symbol", "eval", "ignorecase", "var":
		return
	case "regex":
		for _, item := range items[2:] {
			jitAddMatchPatternBoundSymbols(item, bound)
		}
		return
	case "merge":
		if len(items) > 2 {
			jitAddMatchPatternBoundSymbols(items[2], bound)
		}
		return
	}
	for _, item := range items[1:] {
		jitAddMatchPatternBoundSymbols(item, bound)
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
		switch jitSyntaxKind(list[0]) {
		case SyntaxQuote:
			return
		case SyntaxLambda:
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
		case SyntaxMatch:
			if len(list) > 1 {
				jitCollectLambdaFreeSymbols(list[1], bound, seen, out)
			}
			index := 2
			for index+1 < len(list) {
				branchBound := make(map[Symbol]struct{}, len(bound)+4)
				for symbol := range bound {
					branchBound[symbol] = struct{}{}
				}
				jitAddMatchPatternBoundSymbols(list[index], branchBound)
				jitCollectLambdaFreeSymbols(list[index+1], branchBound, seen, out)
				index += 2
			}
			if index < len(list) {
				jitCollectLambdaFreeSymbols(list[index], bound, seen, out)
			}
			return
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
	switch jitSyntaxKind(items[0]) {
	case SyntaxQuote:
		return false
	case SyntaxEval, SyntaxParser:
		return true
	}
	for _, item := range items {
		if jitExpressionConsumesRuntimeEnv(item) {
			return true
		}
	}
	return false
}

type jitLambdaOuterCapture struct {
	depth int
	index NthLocalVar
}

type jitLambdaNamedOuterCapture struct {
	depth  int
	symbol Symbol
}

func jitCollectLambdaOuterCaptures(expr Scmer, lambdaDepth int, countScopes bool,
	seen map[jitLambdaOuterCapture]struct{}, out *[]jitLambdaOuterCapture,
	namedSeen map[jitLambdaNamedOuterCapture]struct{}, namedOut *[]jitLambdaNamedOuterCapture,
) {
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
	switch jitSyntaxKind(list[0]) {
	case SyntaxQuote:
		return
	case SyntaxLambda:
		if len(list) >= 3 {
			jitCollectLambdaOuterCaptures(list[2], lambdaDepth+1, countScopes, seen, out, namedSeen, namedOut)
		}
		return
	case SyntaxBegin:
		if countScopes {
			for _, item := range list[1:] {
				jitCollectLambdaOuterCaptures(item, lambdaDepth+1, countScopes, seen, out, namedSeen, namedOut)
			}
			return
		}
	case SyntaxBeginMut:
		if countScopes {
			if len(list) > 1 {
				jitCollectLambdaOuterCaptures(list[1], lambdaDepth, countScopes, seen, out, namedSeen, namedOut)
			}
			for _, item := range list[2:] {
				jitCollectLambdaOuterCaptures(item, lambdaDepth+1, countScopes, seen, out, namedSeen, namedOut)
			}
			return
		}
	case SyntaxMatch:
		if countScopes {
			if len(list) > 1 {
				jitCollectLambdaOuterCaptures(list[1], lambdaDepth, countScopes, seen, out, namedSeen, namedOut)
			}
			for index := 3; index < len(list); index += 2 {
				jitCollectLambdaOuterCaptures(list[index], lambdaDepth+1, countScopes, seen, out, namedSeen, namedOut)
			}
			return
		}
	case SyntaxOuter:
		if len(list) == 3 {
			depth, validDepth := outerDepthLiteral(list[1])
			arg := list[2]
			if arg.IsSourceInfo() {
				arg = arg.SourceInfo().value
			}
			if validDepth && int(depth) > lambdaDepth {
				captureDepth := int(depth) - lambdaDepth - 1
				switch arg.GetTag() {
				case tagNthLocalVar:
					capture := jitLambdaOuterCapture{depth: captureDepth, index: arg.NthLocalVar()}
					if _, ok := seen[capture]; !ok {
						seen[capture] = struct{}{}
						*out = append(*out, capture)
					}
				case tagSymbol:
					capture := jitLambdaNamedOuterCapture{depth: captureDepth, symbol: arg.Symbol()}
					if _, ok := namedSeen[capture]; !ok {
						namedSeen[capture] = struct{}{}
						*namedOut = append(*namedOut, capture)
					}
				}
			}
		}
	}
	for _, item := range list {
		jitCollectLambdaOuterCaptures(item, lambdaDepth, countScopes, seen, out, namedSeen, namedOut)
	}
}

func jitLambdaOuterCaptures(body Scmer, countScopes bool) ([]jitLambdaOuterCapture, []jitLambdaNamedOuterCapture) {
	seen := make(map[jitLambdaOuterCapture]struct{}, 4)
	out := make([]jitLambdaOuterCapture, 0, 4)
	namedSeen := make(map[jitLambdaNamedOuterCapture]struct{}, 4)
	namedOut := make([]jitLambdaNamedOuterCapture, 0, 4)
	jitCollectLambdaOuterCaptures(body, 0, countScopes, seen, &out, namedSeen, &namedOut)
	return out, namedOut
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

func jitLambdaNamedCaptureReference(symbol Symbol, depth int) Scmer {
	value := NewSymbol(string(symbol))
	if depth == 0 {
		return value
	}
	return NewSlice([]Scmer{NewSymbol("outer"), NewInt(int64(depth)), value})
}

func jitBindLambdaCaptures(expr Scmer, symbols map[Symbol]NthLocalVar, outerVars map[jitLambdaOuterCapture]NthLocalVar, namedOuterVars map[jitLambdaNamedOuterCapture]NthLocalVar) Scmer {
	return jitBindLambdaCapturesAtDepth(expr, symbols, outerVars, namedOuterVars, 0)
}

// jitBindLambdaSelfValues routes a named recursive closure used as a value
// through a hidden bound parameter. Direct calls in the procedure itself keep
// their symbolic head so the emitter can lower them to a loop or native
// recursion. A reference passed as a callback must instead name the concrete
// closure, because the shared template's lexical environment has no per-
// instance BoundArgs.
func jitBindLambdaSelfValues(expr Scmer, self Symbol, param NthLocalVar) Scmer {
	return jitBindLambdaSelfValuesAtDepth(expr, self, param, 0, 0)
}

// Lexical scopes affect outer references even when they share one invocation.
// Only lambda boundaries decide whether a call can use native self recursion.
func jitBindLambdaSelfValuesAtDepth(expr Scmer, self Symbol, param NthLocalVar, depth, lambdaDepth int) Scmer {
	if expr.IsSourceInfo() {
		source := *expr.SourceInfo()
		source.value = jitBindLambdaSelfValuesAtDepth(source.value, self, param, depth, lambdaDepth)
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
		bound[2] = jitBindLambdaSelfValuesAtDepth(items[2], self, param, depth+1, lambdaDepth+1)
		return NewSlice(bound)
	}
	if hasHead && (head == "begin" || head == "begin_mut") {
		bound := append([]Scmer(nil), items...)
		start := 1
		if head == "begin_mut" {
			start = 2
		}
		for index := start; index < len(items); index++ {
			bound[index] = jitBindLambdaSelfValuesAtDepth(items[index], self, param, depth+1, lambdaDepth)
		}
		return NewSlice(bound)
	}
	if hasHead && (head == "match" || head == "match_mut") {
		bound := append([]Scmer(nil), items...)
		if len(items) > 1 {
			bound[1] = jitBindLambdaSelfValuesAtDepth(items[1], self, param, depth, lambdaDepth)
		}
		for index := 3; index < len(items); index += 2 {
			patternSymbols := make(map[Symbol]struct{})
			jitAddMatchPatternBoundSymbols(items[index-1], patternSymbols)
			if _, shadowed := patternSymbols[self]; !shadowed {
				bound[index] = jitBindLambdaSelfValuesAtDepth(items[index], self, param, depth+1, lambdaDepth)
			}
		}
		return NewSlice(bound)
	}
	if hasHead && (head == "define" || head == "set" || head == "setN") && len(items) == 3 {
		bound := append([]Scmer(nil), items...)
		bound[2] = jitBindLambdaSelfValuesAtDepth(items[2], self, param, depth, lambdaDepth)
		return NewSlice(bound)
	}
	bound := make([]Scmer, len(items))
	for index, item := range items {
		// Only a direct call in the recursive procedure itself can use the
		// native self-call lowering. Nested lambdas require the bound closure.
		if index == 0 && lambdaDepth == 0 && hasHead && head == self {
			bound[index] = item
			continue
		}
		bound[index] = jitBindLambdaSelfValuesAtDepth(item, self, param, depth, lambdaDepth)
	}
	return NewSlice(bound)
}

func jitBindLambdaCapturesAtDepth(expr Scmer, symbols map[Symbol]NthLocalVar, outerVars map[jitLambdaOuterCapture]NthLocalVar, namedOuterVars map[jitLambdaNamedOuterCapture]NthLocalVar, depth int) Scmer {
	if expr.IsSourceInfo() {
		source := *expr.SourceInfo()
		source.value = jitBindLambdaCapturesAtDepth(source.value, symbols, outerVars, namedOuterVars, depth)
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
			outerDepth, validDepth := outerDepthLiteral(items[1])
			if validDepth && int(outerDepth) > depth && key.IsNthLocalVar() {
				capture := jitLambdaOuterCapture{depth: int(outerDepth) - depth - 1, index: key.NthLocalVar()}
				if param, exists := outerVars[capture]; exists {
					return jitLambdaCaptureReference(param, depth)
				}
			}
			if validDepth && int(outerDepth) > depth && key.IsSymbol() {
				capture := jitLambdaNamedOuterCapture{depth: int(outerDepth) - depth - 1, symbol: key.Symbol()}
				if param, exists := namedOuterVars[capture]; exists {
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
			bound[2] = jitBindLambdaCapturesAtDepth(items[2], innerSymbols, outerVars, namedOuterVars, depth+1)
			return NewSlice(bound)
		}
		if string(head) == "begin" {
			bound := append([]Scmer(nil), items...)
			for index := 1; index < len(items); index++ {
				bound[index] = jitBindLambdaCapturesAtDepth(items[index], symbols, outerVars, namedOuterVars, depth+1)
			}
			return NewSlice(bound)
		}
		if string(head) == "begin_mut" {
			bound := append([]Scmer(nil), items...)
			if len(items) > 1 {
				bound[1] = jitBindLambdaCapturesAtDepth(items[1], symbols, outerVars, namedOuterVars, depth)
			}
			for index := 2; index < len(items); index++ {
				bound[index] = jitBindLambdaCapturesAtDepth(items[index], symbols, outerVars, namedOuterVars, depth+1)
			}
			return NewSlice(bound)
		}
		if string(head) == "match" || string(head) == "match_mut" {
			bound := append([]Scmer(nil), items...)
			if len(items) > 1 {
				bound[1] = jitBindLambdaCapturesAtDepth(items[1], symbols, outerVars, namedOuterVars, depth)
			}
			for index := 3; index < len(items); index += 2 {
				branchSymbols := symbols
				if len(symbols) != 0 {
					branchSymbols = make(map[Symbol]NthLocalVar, len(symbols))
					for symbol, slot := range symbols {
						branchSymbols[symbol] = slot
					}
					patternSymbols := make(map[Symbol]struct{})
					jitAddMatchPatternBoundSymbols(items[index-1], patternSymbols)
					for symbol := range patternSymbols {
						delete(branchSymbols, symbol)
					}
				}
				bound[index] = jitBindLambdaCapturesAtDepth(items[index], branchSymbols, outerVars, namedOuterVars, depth+1)
			}
			return NewSlice(bound)
		}
		if (string(head) == "define" || string(head) == "set" || string(head) == "setN") && len(items) == 3 {
			bound := append([]Scmer(nil), items...)
			bound[2] = jitBindLambdaCapturesAtDepth(items[2], symbols, outerVars, namedOuterVars, depth)
			return NewSlice(bound)
		}
	}
	changed := false
	bound := make([]Scmer, len(items))
	for index, item := range items {
		bound[index] = jitBindLambdaCapturesAtDepth(item, symbols, outerVars, namedOuterVars, depth)
		changed = changed || bound[index] != item
	}
	if !changed {
		return expr
	}
	return NewSlice(bound)
}

func jitRebindProcCapture(proc *Proc, env *Env, key, previous Scmer, hasPrevious bool, value Scmer) *Proc {
	if proc == nil || proc.Compiled == nil || proc.JITCode == 0 || len(proc.Compiled.CaptureKeys) != proc.Compiled.CaptureCount {
		return nil
	}
	index := -1
	if key.IsSymbol() {
		for candidate, symbol := range proc.Compiled.CaptureSymbols {
			if symbol == key.Symbol() {
				index = candidate
				break
			}
		}
	}
	for candidate, captureKey := range proc.Compiled.CaptureKeys {
		if index >= 0 {
			break
		}
		if Equal(captureKey, key) {
			index = candidate
			break
		}
	}
	if index < 0 && hasPrevious {
		for candidate, capture := range jitProcCaptures(proc) {
			if Equal(capture, previous) {
				index = candidate
				break
			}
		}
	}
	sourceCaptures := jitProcCaptures(proc)
	boundProc := jitAllocateProcContext(proc, len(sourceCaptures))
	boundProc.En = env
	boundCaptures := jitProcCaptures(boundProc)
	copy(boundCaptures, sourceCaptures)
	if index >= 0 {
		boundCaptures[index] = value
	}
	return boundProc
}

func jitProcCaptures(proc *Proc) []Scmer {
	if proc == nil || proc.Compiled == nil || proc.Compiled.CaptureCount == 0 {
		return nil
	}
	return unsafe.Slice((*Scmer)(unsafe.Add(unsafe.Pointer(proc), unsafe.Offsetof(ProcJIT{}.Context))), proc.Compiled.CaptureCount)
}

// JITCapturedLocals returns the numbered-local base and inline capture values
// of this concrete procedure. The returned slice aliases the ProcJIT tail and
// remains valid while proc is reachable. Analyzers use it to reconstruct the
// call frame without separating executable callbacks from their source Proc.
func (proc *Proc) JITCapturedLocals() (base int, captures []Scmer) {
	if proc == nil || proc.Compiled == nil {
		return 0, nil
	}
	return proc.Compiled.CaptureBase, jitProcCaptures(proc)
}

// closeJITProcedureCaptures replaces the hidden numbered parameters of a
// ProcJIT tail with ordinary Scheme literals. Closed procedures are persisted
// independently of executable mappings, so their serialized body must not
// depend on the process-local capture tail.
func closeJITProcedureCaptures(expr Scmer, captureBase int, captures []Scmer, depth int) Scmer {
	if expr.IsSourceInfo() {
		source := *expr.SourceInfo()
		source.value = closeJITProcedureCaptures(source.value, captureBase, captures, depth)
		return NewSourceInfo(source)
	}
	if depth == 0 && expr.IsNthLocalVar() {
		index := int(expr.NthLocalVar()) - captureBase
		if index >= 0 && index < len(captures) {
			return captures[index]
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
	if hasHead && head == Symbol("quote") {
		return expr
	}
	if hasHead && head == Symbol("outer") && len(items) == 3 {
		outerDepth, validDepth := outerDepthLiteral(items[1])
		key := items[2].WithoutSourceInfo()
		if validDepth && int(outerDepth) == depth && key.IsNthLocalVar() {
			index := int(key.NthLocalVar()) - captureBase
			if index >= 0 && index < len(captures) {
				return captures[index]
			}
		}
	}
	bound := append([]Scmer(nil), items...)
	if hasHead {
		switch head {
		case Symbol("lambda"):
			if len(items) >= 3 {
				bound[2] = closeJITProcedureCaptures(items[2], captureBase, captures, depth+1)
			}
			return NewSlice(bound)
		case Symbol("begin"):
			for index := 1; index < len(items); index++ {
				bound[index] = closeJITProcedureCaptures(items[index], captureBase, captures, depth+1)
			}
			return NewSlice(bound)
		case Symbol("begin_mut"):
			if len(items) > 1 {
				bound[1] = closeJITProcedureCaptures(items[1], captureBase, captures, depth)
			}
			for index := 2; index < len(items); index++ {
				bound[index] = closeJITProcedureCaptures(items[index], captureBase, captures, depth+1)
			}
			return NewSlice(bound)
		case Symbol("match"), Symbol("match_mut"):
			if len(items) > 1 {
				bound[1] = closeJITProcedureCaptures(items[1], captureBase, captures, depth)
			}
			for index := 3; index < len(items); index += 2 {
				bound[index] = closeJITProcedureCaptures(items[index], captureBase, captures, depth+1)
			}
			return NewSlice(bound)
		}
	}
	for index, item := range items {
		bound[index] = closeJITProcedureCaptures(item, captureBase, captures, depth)
	}
	return NewSlice(bound)
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
	env := jitRuntimeEnvFromCaptures(args[3:])
	value := NewProcStruct(Proc{
		Params:  args[0],
		Body:    args[1],
		En:      env,
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
				if proc := value.callable.Proc(); proc != nil && proc.JITCode != 0 {
					return proc.callJIT(value.args)
				}
			}
			return Apply(value.callable, value.args...)
		}
	}
	if thunk.GetTag() == tagProc {
		if proc := thunk.Proc(); proc != nil && proc.JITCode != 0 {
			return proc.callJIT(nil)
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

// GoABIIntRegs lists integer argument/result registers in the target's
// ABIInternal order. The architecture backend supplies this bank separately
// from the allocator's persistent register homes.
var GoABIIntRegs = jitGoABIIntRegs()

type goCallArgWord struct {
	loc      JITLoc
	reg      Reg
	imm      uint64
	stackOff int32
	// groupWords is set on the first word of each source argument. Go's
	// ABIInternal assigns an aggregate wholly to the stack when all of its words
	// do not fit in the remaining registers; it must never split a slice header.
	groupWords uint8
}

type goCallArgLocation struct {
	inReg    bool
	reg      Reg
	stackOff int32
}

func layoutGoCallArgs(words []goCallArgWord) ([]goCallArgLocation, int) {
	locations := make([]goCallArgLocation, len(words))
	regIndex, stackIndex := 0, 0
	for index := 0; index < len(words); {
		width := int(words[index].groupWords)
		if width == 0 {
			width = 1
		}
		if index+width > len(words) {
			panic("jit: invalid Go ABI argument group")
		}
		if regIndex+width <= len(GoABIIntRegs) {
			for part := 0; part < width; part++ {
				locations[index+part] = goCallArgLocation{inReg: true, reg: GoABIIntRegs[regIndex+part]}
			}
			regIndex += width
		} else {
			for part := 0; part < width; part++ {
				locations[index+part] = goCallArgLocation{stackOff: int32(stackIndex * 8)}
				stackIndex++
			}
		}
		index += width
	}
	return locations, stackIndex
}

func (ctx *JITContext) collectLiveRegsForCall(buf *[16]Reg) []Reg {
	// Only allocator-owned registers can contain live SSA values. Reserved ABI,
	// scratch and frame registers are handled explicitly by the call emitter.
	// Protected registers are explicitly live across the current nested emitter
	// even when a borrowed descriptor has no allocator owner (or its owner was
	// moved to a spill slot). Calls must preserve that contract as well as normal
	// allocator ownership.
	allocatedMask := (ctx.AllRegs &^ ctx.FreeRegs) | (ctx.ProtectedRegs & ctx.AllRegs)
	for r := Reg(0); r <= jitLastGPReg; r++ {
		if ctx.RegOwners[r] != nil && (ctx.FreeRegs&(1<<uint(r))) != 0 && (ctx.ProtectedRegs&(1<<uint(r))) == 0 {
			panic("jit: internal reg state mismatch (owner set but register marked free)")
		}
	}
	liveCount := 0
	unknownCount := 0
	for r := Reg(0); r <= jitLastGPReg; r++ {
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
		for r := Reg(0); r <= jitLastGPReg; r++ {
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

func (ctx *JITContext) collectLiveFPRegsForCall(buf *[16]Reg) []Reg {
	liveMask := (ctx.AllFPRegs &^ ctx.FreeFPRegs) | (ctx.ProtectedRegs & ctx.AllFPRegs)
	count := 0
	for reg := jitFirstFPReg; reg <= jitLastFPReg; reg++ {
		if liveMask&(uint64(1)<<uint(reg)) == 0 {
			continue
		}
		buf[count] = reg
		count++
	}
	return buf[:count]
}

// jitFPCallBoundary saves unboxed values in the fixed JIT frame. Go and JIT
// callees may clobber every XMM register even when they return boxed Scmer.
// Scalar FP values are not GC roots; stack growth relocates their frame homes.
type jitFPCallBoundary struct {
	ctx   *JITContext
	regs  [16]Reg
	offs  [16]int32
	count int
}

func (ctx *JITContext) preserveFPRegistersForCall() jitFPCallBoundary {
	boundary := jitFPCallBoundary{ctx: ctx}
	var live [16]Reg
	for index, reg := range ctx.collectLiveFPRegsForCall(&live) {
		boundary.regs[index] = reg
		boundary.offs[index] = ctx.AllocSpill(8)
		ctx.EmitStoreFPRegMem(reg, ctx.FrameReg, boundary.offs[index])
		boundary.count++
	}
	return boundary
}

func (boundary *jitFPCallBoundary) restore() {
	for index := 0; index < boundary.count; index++ {
		boundary.ctx.EmitLoadFPRegMem(boundary.regs[index], boundary.ctx.FrameReg, boundary.offs[index])
	}
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
const jitMaxParallelRegMoves = 16

type jitRegMove struct {
	dst Reg
	src Reg
}

// jitParallelRegMoveBatch describes one simultaneous register assignment.
// It is deliberately fixed-size: argument/result boundaries are emitted in a
// hot compiler path and must not allocate merely to shuffle machine words.
// Sources may repeat, destinations may not, and identity moves are discarded.
type jitParallelRegMoveBatch struct {
	moves [jitMaxParallelRegMoves]jitRegMove
	count uint8
}

// jitDeferredRegMoves represents sequential register copies as aliases to the
// physical register which still contains the requested value. Unlike a
// parallel move batch, assigning a register is an ordering boundary for older
// aliases which still depend on its previous contents.
//
// The fixed arrays make queuing and flushing allocation-free. Registers are
// architecture identifiers supplied by the backend; the scheduler itself does
// not know whether they spell RAX, X0, or a future RISC-V register.
type jitDeferredRegMoves struct {
	sources [64]Reg
	active  uint64
	// held marks physical source registers whose descriptor lifetime ended while
	// a deferred destination still aliases their contents. Such registers remain
	// unavailable to AllocReg until the final alias is emitted or discarded.
	held     uint64
	flushing bool
}

func (moves *jitDeferredRegMoves) source(reg Reg) Reg {
	for depth := 0; depth < len(moves.sources); depth++ {
		if moves.active&(uint64(1)<<uint(reg)) == 0 {
			return reg
		}
		reg = moves.sources[reg]
	}
	panic("jit: cyclic deferred register moves")
}

// deferRegMove records the value currently visible through src as dst's new
// logical value. If dst's old physical contents still feed another alias, that
// dependent value must be materialized before dst can be redefined.
func (ctx *JITContext) deferRegMove(dst, src Reg) {
	ctx.materializeBooleanFlags()
	if dst >= jitRegisterCount || src >= jitRegisterCount {
		panic("jit: deferred move register outside scheduler range")
	}
	if dst == src {
		return
	}
	var dependents uint64
	for pending := ctx.DeferredRegMoves.active; pending != 0; pending &= pending - 1 {
		candidate := Reg(bits.TrailingZeros64(pending))
		bit := uint64(1) << uint(candidate)
		if candidate != dst && ctx.DeferredRegMoves.source(candidate) == dst {
			dependents |= bit
		}
	}
	ctx.flushDeferredRegMoves(dependents)
	source := ctx.DeferredRegMoves.source(src)
	if source == dst {
		ctx.DeferredRegMoves.active &^= uint64(1) << uint(dst)
		return
	}
	ctx.DeferredRegMoves.sources[dst] = source
	ctx.DeferredRegMoves.active |= uint64(1) << uint(dst)
}

// releaseDeferredReg drops a dead logical destination. Any other pending value
// which still names the register's old physical contents is emitted first,
// because the allocator may hand the register to a destructive producer next.
func (ctx *JITContext) releaseDeferredReg(reg Reg) bool {
	if reg >= jitRegisterCount {
		return false
	}
	regBit := uint64(1) << uint(reg)
	if ctx.DeferredRegMoves.active&regBit != 0 {
		source := ctx.DeferredRegMoves.source(reg)
		representedElsewhere := false
		for pending := ctx.DeferredRegMoves.active; pending != 0; pending &= pending - 1 {
			candidate := Reg(bits.TrailingZeros64(pending))
			if candidate != reg && ctx.DeferredRegMoves.source(candidate) == source {
				representedElsewhere = true
				break
			}
		}
		if representedElsewhere {
			ctx.DeferredRegMoves.active &^= regBit
		} else {
			// The final logical alias may still be read through a non-owning
			// descriptor copy. Materialize it before returning its named register.
			ctx.flushDeferredRegMoves(regBit)
		}
	}
	hasDependents := false
	for pending := ctx.DeferredRegMoves.active; pending != 0; pending &= pending - 1 {
		candidate := Reg(bits.TrailingZeros64(pending))
		if candidate != reg && ctx.DeferredRegMoves.source(candidate) == reg {
			hasDependents = true
			break
		}
	}
	ctx.DeferredRegMoves.active &^= regBit
	if hasDependents {
		ctx.DeferredRegMoves.held |= uint64(1) << uint(reg)
		return true
	}
	ctx.releaseUnusedDeferredSources()
	return false
}

func (ctx *JITContext) releaseUnusedDeferredSources() {
	moves := &ctx.DeferredRegMoves
	for held := moves.held; held != 0; held &= held - 1 {
		source := Reg(bits.TrailingZeros64(held))
		bit := uint64(1) << uint(source)
		used := false
		for pending := moves.active; pending != 0; pending &= pending - 1 {
			candidate := Reg(bits.TrailingZeros64(pending))
			if moves.source(candidate) == source {
				used = true
				break
			}
		}
		if used {
			continue
		}
		moves.held &^= bit
		if source >= RegX0 {
			if ctx.AllFPRegs&bit != 0 {
				ctx.FreeFPRegs |= bit
			}
		} else if ctx.AllRegs&bit != 0 {
			ctx.FreeRegs |= bit
		}
	}
}

// flushDeferredRegMoves emits only the requested logical destinations. Their
// canonical sources are physical registers because deferRegMove flattens every
// alias chain. The parallel solver retains correct old-source semantics when
// several destinations are materialized together.
func (ctx *JITContext) flushDeferredRegMoves(mask uint64) {
	moves := &ctx.DeferredRegMoves
	mask &= moves.active
	if mask == 0 || moves.flushing {
		return
	}
	moves.flushing = true
	defer func() { moves.flushing = false }()
	var batch jitParallelRegMoveBatch
	for pending := mask; pending != 0; pending &= pending - 1 {
		dst := Reg(bits.TrailingZeros64(pending))
		batch.add(dst, moves.source(dst))
	}
	ctx.emitParallelRegMoveBatch(&batch)
	moves.active &^= mask
	ctx.releaseUnusedDeferredSources()
}

// FlushRegisterMoves is a full materialization barrier for control-flow,
// safepoint, stack-map, and raw-code-position boundaries.
func (ctx *JITContext) FlushRegisterMoves() {
	ctx.materializeBooleanFlags()
	ctx.flushDeferredRegMoves(ctx.DeferredRegMoves.active)
}

type jitBooleanFlags struct {
	FlagsID   uint32
	Reg       Reg
	Condition JITCondition
}

// DeferBooleanFlags records the result of the comparison just emitted. The
// caller reserves reg before that comparison; recording flags emits no code.
func (ctx *JITContext) DeferBooleanFlags(reg Reg, condition JITCondition) JITValueDesc {
	if ctx.lazyFlags.FlagsID != 0 {
		panic("jit: comparison overwrote an unmaterialized boolean")
	}
	ctx.flagsSerial++
	if ctx.flagsSerial == 0 {
		panic("jit: boolean flags identity overflow")
	}
	value := JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: reg, Condition: condition, FlagsID: ctx.flagsSerial}
	ctx.lazyFlags = jitBooleanFlags{FlagsID: value.FlagsID, Reg: value.Reg, Condition: value.Condition}
	return value
}

func (ctx *JITContext) hasBooleanFlags(value JITValueDesc) bool {
	return value.Loc == LocFlags && value.FlagsID != 0 && value.FlagsID == ctx.lazyFlags.FlagsID
}

func (ctx *JITContext) materializeBooleanFlags() {
	if ctx.lazyFlags.FlagsID == 0 {
		return
	}
	value := ctx.lazyFlags
	ctx.lazyFlags = jitBooleanFlags{}
	ctx.EmitSetcc(value.Reg, value.Condition)
	if owner := ctx.RegOwners[value.Reg]; owner != nil && owner.FlagsID == value.FlagsID {
		owner.Loc = LocReg
		owner.FlagsID = 0
	}
}

func jitRegisterMask(regs ...Reg) uint64 {
	var mask uint64
	for _, reg := range regs {
		if reg >= jitRegisterCount {
			panic("jit: register outside use/def mask")
		}
		mask |= uint64(1) << uint(reg)
	}
	return mask
}

// beginRegisterInstruction applies an architecture emitter's use/def contract.
// Reads materialize only their requested logical values. Before a write, old
// physical values still referenced by another alias are preserved; the written
// destinations themselves become concrete outputs of this instruction.
func (ctx *JITContext) beginRegisterInstruction(reads, writes uint64) {
	ctx.materializeBooleanFlags()
	if ctx.DeferredRegMoves.active == 0 {
		ctx.registerInstructionDepth++
		return
	}
	ctx.prepareDeferredRegisterInstruction(reads, writes)
	ctx.registerInstructionDepth++
}

func (ctx *JITContext) prepareDeferredRegisterInstruction(reads, writes uint64) {
	ctx.flushDeferredRegMoves(reads)
	var oldValueDependents uint64
	for pending := ctx.DeferredRegMoves.active &^ writes; pending != 0; pending &= pending - 1 {
		candidate := Reg(bits.TrailingZeros64(pending))
		bit := uint64(1) << uint(candidate)
		source := ctx.DeferredRegMoves.source(candidate)
		if writes&(uint64(1)<<uint(source)) != 0 {
			oldValueDependents |= bit
		}
	}
	ctx.flushDeferredRegMoves(oldValueDependents)
	ctx.DeferredRegMoves.active &^= writes
}

func (ctx *JITContext) endRegisterInstruction() {
	ctx.registerInstructionDepth--
}

func (batch *jitParallelRegMoveBatch) add(dst, src Reg) {
	if dst == src {
		return
	}
	for index := uint8(0); index < batch.count; index++ {
		if batch.moves[index].dst == dst {
			panic("jit: parallel move has duplicate destination")
		}
	}
	if batch.count == jitMaxParallelRegMoves {
		panic("jit: parallel move batch overflow")
	}
	batch.moves[batch.count] = jitRegMove{dst: dst, src: src}
	batch.count++
}

func (batch *jitParallelRegMoveBatch) uses(reg Reg) bool {
	for index := uint8(0); index < batch.count; index++ {
		if batch.moves[index].src == reg || batch.moves[index].dst == reg {
			return true
		}
	}
	return false
}

// parallelMoveScratch chooses by architecture role and register-bank order,
// never by a hard-coded x86 register number. The scratch value is saved only
// when a cycle actually needs it, so an allocated outer value remains intact.
func (ctx *JITContext) parallelMoveScratch(batch *jitParallelRegMoveBatch) Reg {
	eligible := func(reg Reg) bool {
		// The stack and frame registers define the generated frame. A backend may
		// also expose one as a value base (notably SliceBase=RSP), but a parallel
		// assignment must never rewrite either register while breaking a cycle.
		return reg != ctx.StackReg && reg != ctx.FrameReg && !batch.uses(reg)
	}
	if eligible(ctx.SliceBase) {
		return ctx.SliceBase
	}
	if eligible(ctx.ScratchReg) {
		return ctx.ScratchReg
	}
	for index := uint8(0); index < ctx.RegisterBank.Count; index++ {
		candidate := ctx.RegisterBank.Registers[index]
		if eligible(candidate) {
			return candidate
		}
	}
	panic("jit: parallel register cycle has no scratch register")
}

// emitParallelRegMoveBatch preserves every source until its last consumer has
// moved. Acyclic assignments are emitted in dependency order. Cycles use one
// backend-selected register whose previous value is preserved on the native
// stack. The batch is copied because solving rewrites and removes its edges.
func (ctx *JITContext) emitParallelRegMoveBatch(input *jitParallelRegMoveBatch) {
	batch := *input
	scratchSaved := false
	scratch := Reg(0)
	defer func() {
		if scratchSaved {
			ctx.EmitPopReg(scratch)
		}
	}()
	for batch.count > 0 {
		emitIdx := -1
		for i := uint8(0); i < batch.count; i++ {
			dstIsPendingSrc := false
			for j := uint8(0); j < batch.count; j++ {
				if i != j && batch.moves[j].src == batch.moves[i].dst {
					dstIsPendingSrc = true
					break
				}
			}
			if !dstIsPendingSrc {
				emitIdx = int(i)
				break
			}
		}
		if emitIdx == -1 {
			if !scratchSaved {
				scratch = ctx.parallelMoveScratch(&batch)
				ctx.EmitPushReg(scratch)
				scratchSaved = true
			}
			cycleDst := batch.moves[0].dst
			ctx.emitMovRegReg(scratch, cycleDst)
			for i := uint8(0); i < batch.count; i++ {
				if batch.moves[i].src == cycleDst {
					batch.moves[i].src = scratch
				}
			}
			continue
		}
		mv := batch.moves[emitIdx]
		ctx.emitMovRegReg(mv.dst, mv.src)
		for index := emitIdx; index+1 < int(batch.count); index++ {
			batch.moves[index] = batch.moves[index+1]
		}
		batch.count--
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

// JITEmitGoCallScmerToFrame emits a Go call whose Scmer result is produced
// directly into a rooted, frame-pointer-relative spill slot. CFG producers use
// this to avoid a transient result pair that would immediately be spilled at
// the outgoing edge.
func JITEmitGoCallScmerToFrame(ctx *JITContext, funcAddr uint64, args []JITValueDesc) JITValueDesc {
	off := ctx.AllocSpill(16)
	var wordsBuf [16]goCallArgWord
	words := ctx.flattenArgs(args, &wordsBuf)
	ctx.EmitGoCallToFrame(funcAddr, words, []int32{off, off + 8})
	ctx.setStackPointer(jitStackRootFrameBP, off, true)
	return JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: off, Rooted: true}
}

func (ctx *JITContext) emitGoCall(funcAddr uint64, argWords []goCallArgWord, numResultWords int, resultsBuf *[16]Reg, resultTargets []Reg, resultSlotBase Reg, resultSlotOffs []int32) []Reg {
	if ctx.StorageInputsInRegisters {
		panic(jitStorageNeedsStableInputs)
	}
	ctx.NeedsStableArgs = true
	entryDynamicSP := ctx.DynamicSP
	if numResultWords > len(GoABIIntRegs) {
		panic("jit: too many result words for Go ABI")
	}
	argLocations, stackArgWords := layoutGoCallArgs(argWords)
	if len(argWords)*8 > int(jitGoSpillBytes) || stackArgWords*8 > int(jitGoSpillBytes) {
		panic("jit: Go call arguments exceed reserved spill area")
	}
	// Owner-aware liveness with conservative fallback.
	var liveRegsArr [16]Reg
	liveRegs := ctx.collectLiveRegsForCall(&liveRegsArr)
	liveFP := ctx.preserveFPRegistersForCall()
	// A requested result register is dead immediately before the call: argument
	// setup has already consumed its old value and the call deliberately
	// overwrites it. Saving and restoring such a register only to overwrite it
	// again adds two instructions per loop iteration.
	if len(resultTargets) != 0 && len(liveRegs) != 0 {
		kept := liveRegs[:0]
		for _, live := range liveRegs {
			isResult := false
			for _, target := range resultTargets {
				if live == target {
					isResult = true
					break
				}
			}
			if !isResult {
				kept = append(kept, live)
			}
		}
		liveRegs = kept
	}
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
		for i := range argWords {
			if argLocations[i].inReg {
				continue
			}
			dstOff := argLocations[i].stackOff + jitGoStackArgOffset
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

		var moves jitParallelRegMoveBatch
		for i := range argWords {
			if !argLocations[i].inReg {
				continue
			}
			target := argLocations[i].reg
			if argWords[i].loc == LocReg && argWords[i].reg != target {
				moves.add(target, argWords[i].reg)
			}
		}
		ctx.emitParallelRegMoveBatch(&moves)

		for i := range argWords {
			if !argLocations[i].inReg {
				continue
			}
			target := argLocations[i].reg
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
		liveFP.restore()
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
		var moves jitParallelRegMoveBatch
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
				moves.add(r, GoABIIntRegs[i])
			}
			resultsBuf[i] = r
		}
		ctx.emitParallelRegMoveBatch(&moves)
		return resultsBuf[:numResultWords]
	}

	// Reserve stack space for result words (above saved registers).
	// After restoring saved regs, these slots will be at [RSP+0..].
	resultBytes := numResultWords * 8
	if resultBytes > 0 {
		ctx.EmitReserveStackBytes(int32(resultBytes))
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
	liveFP.restore()

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
		groupStart := n
		ctx.SyncDesc(&args[index])
		if args[index].Loc == LocClosurePair {
			ctx.EnsureDesc(&args[index])
		}
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
		buf[groupStart].groupWords = uint8(n - groupStart)
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
	// Flags remain lazy inside a predicate, but a value ABI needs a bool payload.
	if src.Loc == LocFlags {
		ctx.EnsureDesc(src)
	}
	ctx.SyncDesc(src)
	ctx.SyncDesc(dst)
	if dst.Loc == LocStackPair {
		ctx.EmitCopyScmerToDesc(dst, src)
		base := jitStackRootFrameSP
		offset := dst.StackOff - ctx.DynamicSP
		if dst.StackOff < 0 {
			base = jitStackRootFrameBP
			offset = dst.StackOff
		}
		ctx.setStackPointer(base, offset, true)
		return
	}
	if dst.Loc != LocRegPair {
		panic("jit: pair result destination requires a register or stack pair")
	}
	if src.Loc == LocImm {
		switch src.Imm.GetTag() {
		case tagBool:
			ctx.EmitMakeBool(*dst, *src)
		case tagInt:
			ctx.EmitMakeInt(*dst, *src)
		case tagFloat:
			ctx.EmitMakeFloat(*dst, *src)
		case tagNil:
			ctx.EmitMakeNil(*dst)
		default:
			ptr, aux := src.Imm.RawWords()
			ctx.EmitMovRegImm64(dst.Reg, uint64(ptr))
			ctx.EmitMovRegImm64(dst.Reg2, aux)
		}
		return
	}
	if src.Loc == LocReg {
		switch src.Type {
		case tagBool:
			ctx.EmitMakeBool(*dst, *src)
		case tagInt:
			ctx.EmitMakeInt(*dst, *src)
		case tagFloat:
			ctx.EmitMakeFloat(*dst, *src)
		default:
			panic("jit: scalar pair move requires a known primitive type")
		}
		return
	}
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
	if src.Loc == LocClosurePair || src.Loc == LocStack {
		ctx.EnsureDesc(src)
		ctx.EmitMovPairToResult(src, dst)
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
	base    unsafe.Pointer // start of mmap'd region
	mapping []byte         // original mmap slice, retained for munmap
	size    int            // total bytes
	offset  int            // bump pointer (next free byte), guarded by jitPool.mu
	handle  interface{}    // opaque registration handle (nil = unregistered)
	// live counts code reservations which can still become reachable or are
	// owned by a reachable JITEntryPoint. A sealed arena accepts no new code and
	// is unmapped as soon as live reaches zero. These fields use jitPool.mu.
	live     int
	sealed   bool
	unmapped bool

	sourceMu      sync.Mutex
	sourceEntries []jitSourceEntry
	sourceMap     atomic.Pointer[jitSourceMap]
	// Metadata and reservation readiness use jitPublication. Allocation takes
	// jitPool.mu before jitPublication; publication never takes the pool lock.
	reservations []*jitCodeReservation
	metaNext     int
}

// jitPublication serializes metadata publication across arenas. Code generation
// remains parallel. A reservation becomes callable only after its own metadata
// and every embedded callee are ready, even across arena boundaries.
var jitPublication = struct {
	sync.Mutex
	cond *sync.Cond
}{}

func init() {
	jitPublication.cond = sync.NewCond(&jitPublication.Mutex)
}

type jitCodeReservation struct {
	offset int // immutable after allocation
	size   int // guarded by jitPool.mu
	// The remaining fields are guarded by jitPublication.
	done       bool
	published  bool
	ready      bool
	pending    int
	dependents []*jitCodeReservation
	maps       []jitStackMap
	onPublish  func()
}

// complete publishes metadata in allocation order, then waits for this code's
// explicit dependencies. Unrelated later reservations do not delay its caller.
func (a *jitArena) complete(reservation *jitCodeReservation, maps []jitStackMap, dependencies []*JITEntryPoint) {
	a.completeMode(reservation, maps, dependencies, true, nil)
}

// completeDeferred returns private code to the enclosing compiler without
// waiting. Shared templates are installed only after the complete dependency
// graph is safe to execute; publishing the parent's maps alone is insufficient.
func (a *jitArena) completeDeferred(reservation *jitCodeReservation, maps []jitStackMap, dependencies []*JITEntryPoint, onPublish func()) {
	a.completeMode(reservation, maps, dependencies, false, onPublish)
}

func (a *jitArena) completeMode(reservation *jitCodeReservation, maps []jitStackMap, dependencies []*JITEntryPoint, wait bool, onPublish func()) {
	if a == nil || reservation == nil {
		return
	}
	jitPublication.Lock()
	defer jitPublication.Unlock()
	if !reservation.done {
		reservation.maps = maps
		reservation.onPublish = onPublish
		reservation.done = true
		for _, entry := range dependencies {
			dependency := entry.reservation
			if dependency != nil && dependency != reservation && !dependency.ready {
				reservation.pending++
				dependency.dependents = append(dependency.dependents, reservation)
			}
		}
	}
	var runnable []*jitCodeReservation
	for a.metaNext < len(a.reservations) && a.reservations[a.metaNext].done {
		next := a.reservations[a.metaNext]
		publishJITStackMaps(a, next.maps)
		next.maps = nil
		next.published = true
		if next.pending == 0 {
			runnable = append(runnable, next)
		}
		a.metaNext++
	}
	for len(runnable) != 0 {
		next := runnable[len(runnable)-1]
		runnable = runnable[:len(runnable)-1]
		if next.onPublish != nil {
			next.onPublish()
			next.onPublish = nil
		}
		next.ready = true
		for _, dependent := range next.dependents {
			dependent.pending--
			if dependent.published && dependent.pending == 0 {
				runnable = append(runnable, dependent)
			}
		}
		next.dependents = nil
	}
	jitPublication.cond.Broadcast()
	if wait {
		for !reservation.ready {
			jitPublication.cond.Wait()
		}
	}
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
	closed bool
}

const jitArenaSize = 1 << 20 // 1 MB per arena

// globalJITPool is the singleton arena pool.
var globalJITPool jitPool

// Alloc bump-allocates size bytes from the pool, 16-byte aligned.
func (p *jitPool) Alloc(size int) (ptr unsafe.Pointer, arena *jitArena, reservation *jitCodeReservation) {
	size = (size + 15) & ^15 // align to 16 bytes
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		panic("jit: allocation after pool shutdown")
	}

	// Try current arena
	if len(p.arenas) > 0 {
		a := p.arenas[len(p.arenas)-1]
		if a.offset+size <= a.size {
			ptr = unsafe.Add(a.base, a.offset)
			reservation = &jitCodeReservation{offset: a.offset, size: size}
			jitPublication.Lock()
			a.reservations = append(a.reservations, reservation)
			jitPublication.Unlock()
			a.offset += size
			a.live++
			p.mu.Unlock()
			return ptr, a, reservation
		}
		a.sealed = true
		if a.live == 0 {
			p.arenas = p.arenas[:len(p.arenas)-1]
			a.unmapped = true
			p.mu.Unlock()
			unmapJITArena(a)
			p.mu.Lock()
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
		base:    unsafe.Pointer(&b[0]),
		mapping: b,
		size:    arenaBytes,
		live:    1,
	}
	a.handle = registerJITArena(a)
	ptr = a.base
	reservation = &jitCodeReservation{size: size}
	a.reservations = append(a.reservations, reservation)
	a.offset = size
	p.arenas = append(p.arenas, a)
	p.mu.Unlock()
	return ptr, a, reservation
}

// Trim returns the unused tail of the newest reservation to the arena bump
// pointer. Emitters reserve for their worst case because code is written in one
// pass; once the exact size is known, sequential compilers should not retain
// that pessimistic capacity. A concurrently allocated successor makes the tail
// an unavoidable hole, but changing the recorded size remains safe and no
// published code address moves.
func (p *jitPool) Trim(a *jitArena, reservation *jitCodeReservation, used int) {
	if a == nil || reservation == nil {
		return
	}
	used = (used + 15) &^ 15
	p.mu.Lock()
	defer p.mu.Unlock()
	if used < 0 || used > reservation.size {
		panic("jit: invalid reservation trim")
	}
	oldEnd := reservation.offset + reservation.size
	reservation.size = used
	if !a.sealed && a.offset == oldEnd {
		a.offset = reservation.offset + used
	}
}

// Free releases one code reservation. Bump-allocated holes are not reused;
// sealing an arena and releasing its last owner reclaims the entire mapping.
func (p *jitPool) Free(a *jitArena) {
	if a == nil {
		return
	}
	p.mu.Lock()
	if a.live <= 0 {
		p.mu.Unlock()
		return
	}
	a.live--
	shouldUnmap := a.sealed && a.live == 0 && !a.unmapped
	if shouldUnmap {
		a.unmapped = true
		for i, candidate := range p.arenas {
			if candidate == a {
				p.arenas = append(p.arenas[:i], p.arenas[i+1:]...)
				break
			}
		}
	}
	p.mu.Unlock()
	if shouldUnmap {
		unmapJITArena(a)
	}
}

// ShutdownJIT retires all executable mappings after callers and background
// work have drained. It is deliberately terminal: compiling after shutdown is
// a programming error.
func ShutdownJIT() {
	globalJITPool.shutdown()
}

func (p *jitPool) shutdown() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	arenas := append([]*jitArena(nil), p.arenas...)
	p.arenas = nil
	for _, a := range arenas {
		a.sealed = true
		a.unmapped = true
	}
	p.mu.Unlock()
	for _, a := range arenas {
		unmapJITArena(a)
	}
}

func unmapJITArena(a *jitArena) {
	unregisterJITArena(a)
	if len(a.mapping) != 0 {
		if err := syscall.Munmap(a.mapping); err != nil {
			panic("jit: munmap arena failed: " + err.Error())
		}
		a.mapping = nil
		a.base = nil
	}
}

type jitCodeLease struct {
	pool  *jitPool
	arena *jitArena
	code  uintptr
}

func releaseJITEntryPoint(lease jitCodeLease) {
	jitNativeCodes.Delete(lease.code)
	lease.pool.Free(lease.arena)
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
	// Fallthrough moves belong to the predecessor. A branch targeting this
	// label must start after those bytes, never in front of them.
	ctx.FlushRegisterMoves()
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
		if f.Relative && jitPortableBackend {
			jitArchPatchBranch(patchAddr, targetPos-f.CodePos)
		} else if f.Relative {
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
	ctx.FlushRegisterMoves()
	for i := range ctx.Fixups {
		f := &ctx.Fixups[i]
		targetPos := ctx.Labels[f.LabelID]
		if targetPos < 0 {
			panic(fmt.Sprintf("jit: undefined label %d referenced at code offset %d", f.LabelID, f.CodePos))
		}
		patchAddr := unsafe.Add(ctx.Start, int(f.CodePos))
		if f.Relative && jitPortableBackend {
			jitArchPatchBranch(patchAddr, targetPos-f.CodePos)
		} else if f.Relative {
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
				{Kind: "any", Label: "scope", Optional: true, Description: "execution scope for compilation cancellation"},
			},
			Return:         &TypeDescriptor{Kind: "any"},
			HasSideEffects: true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				ctx.Coverage.NativeCalls++
				declaration := declarations["jit"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
			},
			JITVirtualArgs: true,
			JITInlineCost:  65535,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "jit?",

		Fn: func(a ...Scmer) Scmer {
			return NewBool(a[0].GetTag() == tagJIT || (a[0].GetTag() == tagProc && a[0].Proc() != nil && a[0].Proc().JITCode != 0))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "tells whether a value is a JIT-compiled function descriptor",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "value", Description: "value to inspect", NoEscape: true},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["jit?"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
				var d27 JITValueDesc
				_ = d27
				var d28 JITValueDesc
				_ = d28
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
				var d62 JITValueDesc
				_ = d62
				var d63 JITValueDesc
				_ = d63
				var d64 JITValueDesc
				_ = d64
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [6]BBDescriptor
				bbs[2].PhiBase = int32(phiBase0) + int32(0)
				bbs[4].PhiBase = int32(phiBase0) + int32(16)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d3 = args[0]
					d3.ID = 0
					d4 = d3
					d4.ID = 0
					d5 = ctx.EmitGetTagDesc(&d4, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d3)
					ctx.EnsureDesc(&d5)
					if d5.Loc == LocImm {
						d6 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(0xb))}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d5.Reg, 11)
						d6 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d6)
					}
					ctx.FreeDesc(&d5)
					d7 = d6
					ctx.EnsureDesc(&d7)
					if d7.Loc != LocImm && d7.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d7.Loc == LocImm {
						if d7.Imm.Bool() {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[2].PhiBase)+int32(0))
							return bbs[2].Render()
						}
						return bbs[1].Render()
					}
					lbl7 := ctx.ReserveLabel()
					ctx.EmitJump(d7.Condition, lbl7)
					ctx.EmitJmp(lbl2)
					ctx.FreeDesc(&d6)
					snap8 := d1
					snap9 := d2
					snap10 := d3
					snap11 := d4
					snap12 := d5
					snap13 := d6
					snap14 := d7
					alloc15 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl7)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc15)
					d1 = snap8
					d2 = snap9
					d3 = snap10
					d4 = snap11
					d5 = snap12
					d6 = snap13
					d7 = snap14
					if !bbs[2].Rendered {
						snap16 := d1
						snap17 := d2
						snap18 := d3
						snap19 := d4
						snap20 := d5
						snap21 := d6
						snap22 := d7
						alloc23 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc23)
						d1 = snap16
						d2 = snap17
						d3 = snap18
						d4 = snap19
						d5 = snap20
						d6 = snap21
						d7 = snap22
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d24 = args[0]
					d24.ID = 0
					d25 = d24
					d25.ID = 0
					d26 = ctx.EmitGetTagDesc(&d25, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d24)
					ctx.EnsureDesc(&d26)
					if d26.Loc == LocImm {
						d27 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d26.Imm.Int()) == uint64(0xa))}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d26.Reg, 10)
						d27 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondEqual}
						ctx.BindReg(r1, &d27)
					}
					ctx.FreeDesc(&d26)
					d28 = d27
					ctx.EnsureDesc(&d28)
					if d28.Loc != LocImm && d28.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d28.Loc == LocImm {
						if d28.Imm.Bool() {
							return bbs[5].Render()
						}
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
						return bbs[4].Render()
					}
					lbl8 := ctx.ReserveLabel()
					ctx.EmitJump(d28.Condition, lbl6)
					ctx.EmitJmp(lbl8)
					ctx.FreeDesc(&d27)
					snap29 := d1
					snap30 := d2
					snap31 := d3
					snap32 := d4
					snap33 := d5
					snap34 := d6
					snap35 := d7
					snap36 := d24
					snap37 := d25
					snap38 := d26
					snap39 := d27
					snap40 := d28
					alloc41 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl8)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc41)
					d1 = snap29
					d2 = snap30
					d3 = snap31
					d4 = snap32
					d5 = snap33
					d6 = snap34
					d7 = snap35
					d24 = snap36
					d25 = snap37
					d26 = snap38
					d27 = snap39
					d28 = snap40
					if !bbs[4].Rendered {
						snap42 := d1
						snap43 := d2
						snap44 := d3
						snap45 := d4
						snap46 := d5
						snap47 := d6
						snap48 := d7
						snap49 := d24
						snap50 := d25
						snap51 := d26
						snap52 := d27
						snap53 := d28
						alloc54 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc54)
						d1 = snap42
						d2 = snap43
						d3 = snap44
						d4 = snap45
						d5 = snap46
						d6 = snap47
						d7 = snap48
						d24 = snap49
						d25 = snap50
						d26 = snap51
						d27 = snap52
						d28 = snap53
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
					}
					return result
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d1)
					if ctx.branchSerial == branchSerial && ctx.hasBooleanFlags(d1) {
						return d1
					}
					if d1.Loc == LocImm {
						ctx.EmitMakeBool(result, d1)
					} else {
						ctx.EmitMovToReg(result.Reg2, d1)
						d55 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d55)
						if d1.Loc == LocReg && d1.Reg != result.Reg2 {
							ctx.FreeReg(d1.Reg)
						}
					}
					result.Type = tagBool
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d56 = args[0]
					d56.ID = 0
					d56 = JITPrepareScmerGoArg(ctx, d56)
					ctx.SyncDesc(&d56)
					d57 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Proc), []JITValueDesc{d56}, 1)
					d57.NoHeapPointer = false
					ctx.BindReg(d57.Reg, &d57)
					ctx.FreeDesc(&d56)
					ctx.EnsureDesc(&d57)
					if d57.Loc == LocImm {
						fieldAddr := uintptr(d57.Imm.Int()) + 0
						r2 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r2, fieldAddr)
						d58 = JITValueDesc{Loc: LocReg, Reg: r2}
						ctx.BindReg(r2, &d58)
					} else {
						off := int32(0)
						baseReg := d57.Reg
						r3 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r3, baseReg, off)
						d58 = JITValueDesc{Loc: LocReg, Reg: r3}
						ctx.BindReg(r3, &d58)
					}
					ctx.FreeDesc(&d57)
					ctx.EnsureDesc(&d58)
					if d58.Loc == LocImm {
						d59 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d58.Imm.Int()) != uint64(0x0))}
					} else {
						ctx.EmitCmpRegImm32(d58.Reg, 0)
						r4 := ctx.AllocRegExcept(d58.Reg)
						ctx.EmitSetcc(r4, CondNotEqual)
						d59 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d59)
					}
					ctx.EnsureDesc(&d59)
					ctx.EmitStoreToStack(d59, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d59)
					ctx.FreeDesc(&d58)
					return bbs[4].Render()
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d2)
					ctx.SyncDesc(&d2)
					if d2.Loc == LocReg || d2.Loc == LocFPReg {
						ctx.ProtectReg(d2.Reg)
					} else if d2.Loc == LocRegPair {
						ctx.ProtectReg(d2.Reg)
						ctx.ProtectReg(d2.Reg2)
					}
					d60 = d2
					if d60.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d60)
					ctx.EmitStoreToStack(d60, int32(bbs[2].PhiBase)+int32(0))
					if d2.Loc == LocReg || d2.Loc == LocFPReg {
						ctx.UnprotectReg(d2.Reg)
					} else if d2.Loc == LocRegPair {
						ctx.UnprotectReg(d2.Reg)
						ctx.UnprotectReg(d2.Reg2)
					}
					return bbs[2].Render()
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d61 = args[0]
					d61.ID = 0
					d61 = JITPrepareScmerGoArg(ctx, d61)
					ctx.SyncDesc(&d61)
					d62 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Proc), []JITValueDesc{d61}, 1)
					d62.NoHeapPointer = false
					ctx.BindReg(d62.Reg, &d62)
					ctx.FreeDesc(&d61)
					ctx.EnsureDesc(&d62)
					if d62.Loc == LocImm {
						d63 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d62.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d62)
						if d62.Loc != LocReg && d62.Loc != LocRegPair && d62.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r5 := ctx.AllocRegExcept(d62.Reg)
						ctx.EmitCmpRegImm32(d62.Reg, 0)
						ctx.EmitSetcc(r5, CondNotEqual)
						d63 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
						ctx.BindReg(r5, &d63)
					}
					ctx.FreeDesc(&d62)
					d64 = d63
					ctx.EnsureDesc(&d64)
					if d64.Loc != LocImm && d64.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d64.Loc == LocImm {
						if d64.Imm.Bool() {
							return bbs[3].Render()
						}
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
						return bbs[4].Render()
					}
					lbl9 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d64.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					ctx.EmitJmp(lbl9)
					snap65 := d1
					snap66 := d2
					snap67 := d3
					snap68 := d4
					snap69 := d5
					snap70 := d6
					snap71 := d7
					snap72 := d24
					snap73 := d25
					snap74 := d26
					snap75 := d27
					snap76 := d28
					snap77 := d55
					snap78 := d56
					snap79 := d57
					snap80 := d58
					snap81 := d59
					snap82 := d60
					snap83 := d61
					snap84 := d62
					snap85 := d63
					snap86 := d64
					alloc87 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl9)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc87)
					d1 = snap65
					d2 = snap66
					d3 = snap67
					d4 = snap68
					d5 = snap69
					d6 = snap70
					d7 = snap71
					d24 = snap72
					d25 = snap73
					d26 = snap74
					d27 = snap75
					d28 = snap76
					d55 = snap77
					d56 = snap78
					d57 = snap79
					d58 = snap80
					d59 = snap81
					d60 = snap82
					d61 = snap83
					d62 = snap84
					d63 = snap85
					d64 = snap86
					if !bbs[4].Rendered {
						snap88 := d1
						snap89 := d2
						snap90 := d3
						snap91 := d4
						snap92 := d5
						snap93 := d6
						snap94 := d7
						snap95 := d24
						snap96 := d25
						snap97 := d26
						snap98 := d27
						snap99 := d28
						snap100 := d55
						snap101 := d56
						snap102 := d57
						snap103 := d58
						snap104 := d59
						snap105 := d60
						snap106 := d61
						snap107 := d62
						snap108 := d63
						snap109 := d64
						alloc110 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc110)
						d1 = snap88
						d2 = snap89
						d3 = snap90
						d4 = snap91
						d5 = snap92
						d6 = snap93
						d7 = snap94
						d24 = snap95
						d25 = snap96
						d26 = snap97
						d27 = snap98
						d28 = snap99
						d55 = snap100
						d56 = snap101
						d57 = snap102
						d58 = snap103
						d59 = snap104
						d60 = snap105
						d61 = snap106
						d62 = snap107
						d63 = snap108
						d64 = snap109
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d63)
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITInlineCost: 27,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "jit-warn-if-fallback",

		Fn: func(a ...Scmer) Scmer {
			value := a[0]
			compiled := value.GetTag() == tagJIT || (value.GetTag() == tagProc && value.Proc() != nil && value.Proc().JITCode != 0)
			if JITEnabled() && !compiled {
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
				declaration := declarations["jit-warn-if-fallback"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d27 JITValueDesc
				_ = d27
				var d28 JITValueDesc
				_ = d28
				var d29 JITValueDesc
				_ = d29
				var d30 JITValueDesc
				_ = d30
				var d57 JITValueDesc
				_ = d57
				var d58 JITValueDesc
				_ = d58
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
				var d80 JITValueDesc
				_ = d80
				var d125 JITValueDesc
				_ = d125
				var d126 JITValueDesc
				_ = d126
				var d127 JITValueDesc
				_ = d127
				var d128 JITValueDesc
				_ = d128
				var d129 JITValueDesc
				_ = d129
				var d130 JITValueDesc
				_ = d130
				var d159 JITValueDesc
				_ = d159
				var d189 JITValueDesc
				_ = d189
				var d220 JITValueDesc
				_ = d220
				var d221 JITValueDesc
				_ = d221
				var d222 JITValueDesc
				_ = d222
				var d223 JITValueDesc
				_ = d223
				var d224 JITValueDesc
				_ = d224
				var d225 JITValueDesc
				_ = d225
				var d226 JITValueDesc
				_ = d226
				var d228 JITValueDesc
				_ = d228
				var d229 JITValueDesc
				_ = d229
				var d231 JITValueDesc
				_ = d231
				var d232 JITValueDesc
				_ = d232
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(48))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [11]BBDescriptor
				bbs[2].PhiBase = int32(phiBase0) + int32(0)
				bbs[4].PhiBase = int32(phiBase0) + int32(16)
				bbs[10].PhiBase = int32(phiBase0) + int32(32)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
				d3 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
				ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(32))
				_ = d3
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d4 = args[0]
					d4.ID = 0
					ctx.StabilizeDescForControlFlow(&d4)
					d5 = d4
					d5.ID = 0
					d6 = ctx.EmitGetTagDesc(&d5, JITValueDesc{Loc: LocAny})
					ctx.EnsureDesc(&d6)
					if d6.Loc == LocImm {
						d7 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d6.Imm.Int()) == uint64(0xb))}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d6.Reg, 11)
						d7 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d7)
					}
					ctx.FreeDesc(&d6)
					d8 = d7
					ctx.EnsureDesc(&d8)
					if d8.Loc != LocImm && d8.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d8.Loc == LocImm {
						if d8.Imm.Bool() {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[2].PhiBase)+int32(0))
							return bbs[2].Render()
						}
						return bbs[1].Render()
					}
					lbl12 := ctx.ReserveLabel()
					ctx.EmitJump(d8.Condition, lbl12)
					ctx.EmitJmp(lbl2)
					ctx.FreeDesc(&d7)
					snap9 := d1
					snap10 := d2
					snap11 := d3
					snap12 := d4
					snap13 := d5
					snap14 := d6
					snap15 := d7
					snap16 := d8
					alloc17 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl12)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc17)
					d1 = snap9
					d2 = snap10
					d3 = snap11
					d4 = snap12
					d5 = snap13
					d6 = snap14
					d7 = snap15
					d8 = snap16
					if !bbs[2].Rendered {
						snap18 := d1
						snap19 := d2
						snap20 := d3
						snap21 := d4
						snap22 := d5
						snap23 := d6
						snap24 := d7
						snap25 := d8
						alloc26 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc26)
						d1 = snap18
						d2 = snap19
						d3 = snap20
						d4 = snap21
						d5 = snap22
						d6 = snap23
						d7 = snap24
						d8 = snap25
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d27 = d4
					d27.ID = 0
					d28 = ctx.EmitGetTagDesc(&d27, JITValueDesc{Loc: LocAny})
					ctx.EnsureDesc(&d28)
					if d28.Loc == LocImm {
						d29 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d28.Imm.Int()) == uint64(0xa))}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d28.Reg, 10)
						d29 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondEqual}
						ctx.BindReg(r1, &d29)
					}
					ctx.FreeDesc(&d28)
					d30 = d29
					ctx.EnsureDesc(&d30)
					if d30.Loc != LocImm && d30.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d30.Loc == LocImm {
						if d30.Imm.Bool() {
							return bbs[5].Render()
						}
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
						return bbs[4].Render()
					}
					lbl13 := ctx.ReserveLabel()
					ctx.EmitJump(d30.Condition, lbl6)
					ctx.EmitJmp(lbl13)
					ctx.FreeDesc(&d29)
					snap31 := d1
					snap32 := d2
					snap33 := d3
					snap34 := d4
					snap35 := d5
					snap36 := d6
					snap37 := d7
					snap38 := d8
					snap39 := d27
					snap40 := d28
					snap41 := d29
					snap42 := d30
					alloc43 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl13)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc43)
					d1 = snap31
					d2 = snap32
					d3 = snap33
					d4 = snap34
					d5 = snap35
					d6 = snap36
					d7 = snap37
					d8 = snap38
					d27 = snap39
					d28 = snap40
					d29 = snap41
					d30 = snap42
					if !bbs[4].Rendered {
						snap44 := d1
						snap45 := d2
						snap46 := d3
						snap47 := d4
						snap48 := d5
						snap49 := d6
						snap50 := d7
						snap51 := d8
						snap52 := d27
						snap53 := d28
						snap54 := d29
						snap55 := d30
						alloc56 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc56)
						d1 = snap44
						d2 = snap45
						d3 = snap46
						d4 = snap47
						d5 = snap48
						d6 = snap49
						d7 = snap50
						d8 = snap51
						d27 = snap52
						d28 = snap53
						d29 = snap54
						d30 = snap55
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
					}
					return result
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					d57 = ctx.EmitGoCallScalar(GoFuncAddr(JITEnabled), []JITValueDesc{}, 1)
					d57.NoHeapPointer = true
					ctx.EmitAndRegImm32(d57.Reg, 1)
					d57.Type = tagBool
					ctx.BindReg(d57.Reg, &d57)
					d58 = d57
					ctx.EnsureDesc(&d58)
					if d58.Loc != LocImm && d58.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d58.Loc == LocImm {
						if d58.Imm.Bool() {
							return bbs[8].Render()
						}
						return bbs[7].Render()
					}
					ctx.EmitCmpRegImm32(d58.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl9)
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
					}
					ctx.FlushRegisterMoves()
					if !bbs[7].Rendered {
						snap59 := d1
						snap60 := d2
						snap61 := d3
						snap62 := d4
						snap63 := d5
						snap64 := d6
						snap65 := d7
						snap66 := d8
						snap67 := d27
						snap68 := d28
						snap69 := d29
						snap70 := d30
						snap71 := d57
						snap72 := d58
						alloc73 := ctx.SnapshotAllocState()
						bbs[7].Render()
						ctx.RestoreAllocState(alloc73)
						d1 = snap59
						d2 = snap60
						d3 = snap61
						d4 = snap62
						d5 = snap63
						d6 = snap64
						d7 = snap65
						d8 = snap66
						d27 = snap67
						d28 = snap68
						d29 = snap69
						d30 = snap70
						d57 = snap71
						d58 = snap72
					}
					if !bbs[8].Rendered {
						return bbs[8].Render()
					}
					return result
					ctx.FreeDesc(&d57)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d4 = JITPrepareScmerGoArg(ctx, d4)
					ctx.SyncDesc(&d4)
					d74 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Proc), []JITValueDesc{d4}, 1)
					d74.NoHeapPointer = false
					ctx.BindReg(d74.Reg, &d74)
					ctx.EnsureDesc(&d74)
					if d74.Loc == LocImm {
						fieldAddr := uintptr(d74.Imm.Int()) + 0
						r2 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r2, fieldAddr)
						d75 = JITValueDesc{Loc: LocReg, Reg: r2}
						ctx.BindReg(r2, &d75)
					} else {
						off := int32(0)
						baseReg := d74.Reg
						r3 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r3, baseReg, off)
						d75 = JITValueDesc{Loc: LocReg, Reg: r3}
						ctx.BindReg(r3, &d75)
					}
					ctx.FreeDesc(&d74)
					ctx.EnsureDesc(&d75)
					if d75.Loc == LocImm {
						d76 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d75.Imm.Int()) != uint64(0x0))}
					} else {
						ctx.EmitCmpRegImm32(d75.Reg, 0)
						r4 := ctx.AllocRegExcept(d75.Reg)
						ctx.EmitSetcc(r4, CondNotEqual)
						d76 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d76)
					}
					ctx.EnsureDesc(&d76)
					ctx.EmitStoreToStack(d76, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d76)
					ctx.FreeDesc(&d75)
					return bbs[4].Render()
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d2)
					ctx.SyncDesc(&d2)
					if d2.Loc == LocReg || d2.Loc == LocFPReg {
						ctx.ProtectReg(d2.Reg)
					} else if d2.Loc == LocRegPair {
						ctx.ProtectReg(d2.Reg)
						ctx.ProtectReg(d2.Reg2)
					}
					d77 = d2
					if d77.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d77)
					ctx.EmitStoreToStack(d77, int32(bbs[2].PhiBase)+int32(0))
					if d2.Loc == LocReg || d2.Loc == LocFPReg {
						ctx.UnprotectReg(d2.Reg)
					} else if d2.Loc == LocRegPair {
						ctx.UnprotectReg(d2.Reg)
						ctx.UnprotectReg(d2.Reg2)
					}
					return bbs[2].Render()
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d4 = JITPrepareScmerGoArg(ctx, d4)
					ctx.SyncDesc(&d4)
					d78 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Proc), []JITValueDesc{d4}, 1)
					d78.NoHeapPointer = false
					ctx.BindReg(d78.Reg, &d78)
					ctx.EnsureDesc(&d78)
					if d78.Loc == LocImm {
						d79 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d78.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d78)
						if d78.Loc != LocReg && d78.Loc != LocRegPair && d78.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r5 := ctx.AllocRegExcept(d78.Reg)
						ctx.EmitCmpRegImm32(d78.Reg, 0)
						ctx.EmitSetcc(r5, CondNotEqual)
						d79 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
						ctx.BindReg(r5, &d79)
					}
					ctx.FreeDesc(&d78)
					d80 = d79
					ctx.EnsureDesc(&d80)
					if d80.Loc != LocImm && d80.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d80.Loc == LocImm {
						if d80.Imm.Bool() {
							return bbs[3].Render()
						}
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
						return bbs[4].Render()
					}
					lbl14 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d80.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					ctx.EmitJmp(lbl14)
					snap81 := d1
					snap82 := d2
					snap83 := d3
					snap84 := d4
					snap85 := d5
					snap86 := d6
					snap87 := d7
					snap88 := d8
					snap89 := d27
					snap90 := d28
					snap91 := d29
					snap92 := d30
					snap93 := d57
					snap94 := d58
					snap95 := d74
					snap96 := d75
					snap97 := d76
					snap98 := d77
					snap99 := d78
					snap100 := d79
					snap101 := d80
					alloc102 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl14)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc102)
					d1 = snap81
					d2 = snap82
					d3 = snap83
					d4 = snap84
					d5 = snap85
					d6 = snap86
					d7 = snap87
					d8 = snap88
					d27 = snap89
					d28 = snap90
					d29 = snap91
					d30 = snap92
					d57 = snap93
					d58 = snap94
					d74 = snap95
					d75 = snap96
					d76 = snap97
					d77 = snap98
					d78 = snap99
					d79 = snap100
					d80 = snap101
					if !bbs[4].Rendered {
						snap103 := d1
						snap104 := d2
						snap105 := d3
						snap106 := d4
						snap107 := d5
						snap108 := d6
						snap109 := d7
						snap110 := d8
						snap111 := d27
						snap112 := d28
						snap113 := d29
						snap114 := d30
						snap115 := d57
						snap116 := d58
						snap117 := d74
						snap118 := d75
						snap119 := d76
						snap120 := d77
						snap121 := d78
						snap122 := d79
						snap123 := d80
						alloc124 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc124)
						d1 = snap103
						d2 = snap104
						d3 = snap105
						d4 = snap106
						d5 = snap107
						d6 = snap108
						d7 = snap109
						d8 = snap110
						d27 = snap111
						d28 = snap112
						d29 = snap113
						d30 = snap114
						d57 = snap115
						d58 = snap116
						d74 = snap117
						d75 = snap118
						d76 = snap119
						d77 = snap120
						d78 = snap121
						d79 = snap122
						d80 = snap123
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d79)
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d4 = JITPrepareScmerGoArg(ctx, d4)
					d125 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d125.Loc == LocRegPair || d125.Loc == LocStackPair || d125.Loc == LocRegTriple || d125.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d4)
					ctx.SyncDesc(&d125)
					d126 = ctx.EmitGoCallScalar(GoFuncAddr(SerializeToString), []JITValueDesc{d4, d125}, 2)
					d126.NoHeapPointer = false
					ctx.BindReg(d126.Reg, &d126)
					ctx.BindReg(d126.Reg2, &d126)
					ctx.StabilizeDescForControlFlow(&d126)
					d127 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d127)
					if d127.Loc == LocImm {
						d128 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d127.Imm.Int() > 1)}
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d127.Reg, 1)
						d128 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r6, Condition: CondSignedGreater}
						ctx.BindReg(r6, &d128)
					}
					ctx.FreeDesc(&d127)
					d129 = d128
					ctx.EnsureDesc(&d129)
					if d129.Loc != LocImm && d129.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d129.Loc == LocImm {
						if d129.Imm.Bool() {
							return bbs[9].Render()
						}
						ctx.SyncDesc(&d126)
						if d126.Loc == LocReg || d126.Loc == LocFPReg {
							ctx.ProtectReg(d126.Reg)
						} else if d126.Loc == LocRegPair {
							ctx.ProtectReg(d126.Reg)
							ctx.ProtectReg(d126.Reg2)
						}
						d130 = d126
						if d130.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d130)
						if d130.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d130, int32(bbs[10].PhiBase)+int32(0), 2)
						} else if d130.Loc == LocInputPair {
							ctx.EnsureDesc(&d130)
							ctx.EmitStoreScmerToStack(d130, int32(bbs[10].PhiBase)+int32(0))
						} else if d130.Loc == LocRegPair || d130.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d130, int32(bbs[10].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d130)
							ctx.EmitStoreToStack(d130, int32(bbs[10].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[10].PhiBase)+int32(0))+8)
						}
						if d126.Loc == LocReg || d126.Loc == LocFPReg {
							ctx.UnprotectReg(d126.Reg)
						} else if d126.Loc == LocRegPair {
							ctx.UnprotectReg(d126.Reg)
							ctx.UnprotectReg(d126.Reg2)
						}
						return bbs[10].Render()
					}
					lbl15 := ctx.ReserveLabel()
					ctx.EmitJump(d129.Condition, lbl10)
					ctx.EmitJmp(lbl15)
					ctx.FreeDesc(&d128)
					snap131 := d1
					snap132 := d2
					snap133 := d3
					snap134 := d4
					snap135 := d5
					snap136 := d6
					snap137 := d7
					snap138 := d8
					snap139 := d27
					snap140 := d28
					snap141 := d29
					snap142 := d30
					snap143 := d57
					snap144 := d58
					snap145 := d74
					snap146 := d75
					snap147 := d76
					snap148 := d77
					snap149 := d78
					snap150 := d79
					snap151 := d80
					snap152 := d125
					snap153 := d126
					snap154 := d127
					snap155 := d128
					snap156 := d129
					snap157 := d130
					alloc158 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl15)
					ctx.SyncDesc(&d126)
					if d126.Loc == LocReg || d126.Loc == LocFPReg {
						ctx.ProtectReg(d126.Reg)
					} else if d126.Loc == LocRegPair {
						ctx.ProtectReg(d126.Reg)
						ctx.ProtectReg(d126.Reg2)
					}
					d159 = d126
					if d159.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d159)
					if d159.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d159, int32(bbs[10].PhiBase)+int32(0), 2)
					} else if d159.Loc == LocInputPair {
						ctx.EnsureDesc(&d159)
						ctx.EmitStoreScmerToStack(d159, int32(bbs[10].PhiBase)+int32(0))
					} else if d159.Loc == LocRegPair || d159.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d159, int32(bbs[10].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d159)
						ctx.EmitStoreToStack(d159, int32(bbs[10].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[10].PhiBase)+int32(0))+8)
					}
					if d126.Loc == LocReg || d126.Loc == LocFPReg {
						ctx.UnprotectReg(d126.Reg)
					} else if d126.Loc == LocRegPair {
						ctx.UnprotectReg(d126.Reg)
						ctx.UnprotectReg(d126.Reg2)
					}
					ctx.EmitJmp(lbl11)
					ctx.RestoreAllocState(alloc158)
					d1 = snap131
					d2 = snap132
					d3 = snap133
					d4 = snap134
					d5 = snap135
					d6 = snap136
					d7 = snap137
					d8 = snap138
					d27 = snap139
					d28 = snap140
					d29 = snap141
					d30 = snap142
					d57 = snap143
					d58 = snap144
					d74 = snap145
					d75 = snap146
					d76 = snap147
					d77 = snap148
					d78 = snap149
					d79 = snap150
					d80 = snap151
					d125 = snap152
					d126 = snap153
					d127 = snap154
					d128 = snap155
					d129 = snap156
					d130 = snap157
					if !bbs[10].Rendered {
						snap160 := d1
						snap161 := d2
						snap162 := d3
						snap163 := d4
						snap164 := d5
						snap165 := d6
						snap166 := d7
						snap167 := d8
						snap168 := d27
						snap169 := d28
						snap170 := d29
						snap171 := d30
						snap172 := d57
						snap173 := d58
						snap174 := d74
						snap175 := d75
						snap176 := d76
						snap177 := d77
						snap178 := d78
						snap179 := d79
						snap180 := d80
						snap181 := d125
						snap182 := d126
						snap183 := d127
						snap184 := d128
						snap185 := d129
						snap186 := d130
						snap187 := d159
						alloc188 := ctx.SnapshotAllocState()
						bbs[10].Render()
						ctx.RestoreAllocState(alloc188)
						d1 = snap160
						d2 = snap161
						d3 = snap162
						d4 = snap163
						d5 = snap164
						d6 = snap165
						d7 = snap166
						d8 = snap167
						d27 = snap168
						d28 = snap169
						d29 = snap170
						d30 = snap171
						d57 = snap172
						d58 = snap173
						d74 = snap174
						d75 = snap175
						d76 = snap176
						d77 = snap177
						d78 = snap178
						d79 = snap179
						d80 = snap180
						d125 = snap181
						d126 = snap182
						d127 = snap183
						d128 = snap184
						d129 = snap185
						d130 = snap186
						d159 = snap187
					}
					if !bbs[9].Rendered {
						return bbs[9].Render()
					}
					return result
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
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
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[8].Render = func() JITValueDesc {
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
						return result
					}
					bbs[8].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d189 = d1
					ctx.EnsureDesc(&d189)
					if d189.Loc != LocImm && d189.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d189.Loc == LocImm {
						if d189.Imm.Bool() {
							return bbs[7].Render()
						}
						return bbs[6].Render()
					}
					ctx.EmitCmpRegImm32(d189.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl8)
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
					}
					ctx.FlushRegisterMoves()
					if !bbs[6].Rendered {
						snap190 := d1
						snap191 := d2
						snap192 := d3
						snap193 := d4
						snap194 := d5
						snap195 := d6
						snap196 := d7
						snap197 := d8
						snap198 := d27
						snap199 := d28
						snap200 := d29
						snap201 := d30
						snap202 := d57
						snap203 := d58
						snap204 := d74
						snap205 := d75
						snap206 := d76
						snap207 := d77
						snap208 := d78
						snap209 := d79
						snap210 := d80
						snap211 := d125
						snap212 := d126
						snap213 := d127
						snap214 := d128
						snap215 := d129
						snap216 := d130
						snap217 := d159
						snap218 := d189
						alloc219 := ctx.SnapshotAllocState()
						bbs[6].Render()
						ctx.RestoreAllocState(alloc219)
						d1 = snap190
						d2 = snap191
						d3 = snap192
						d4 = snap193
						d5 = snap194
						d6 = snap195
						d7 = snap196
						d8 = snap197
						d27 = snap198
						d28 = snap199
						d29 = snap200
						d30 = snap201
						d57 = snap202
						d58 = snap203
						d74 = snap204
						d75 = snap205
						d76 = snap206
						d77 = snap207
						d78 = snap208
						d79 = snap209
						d80 = snap210
						d125 = snap211
						d126 = snap212
						d127 = snap213
						d128 = snap214
						d129 = snap215
						d130 = snap216
						d159 = snap217
						d189 = snap218
					}
					if !bbs[7].Rendered {
						return bbs[7].Render()
					}
					return result
					return result
				}
				bbs[9].Render = func() JITValueDesc {
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
						return result
					}
					bbs[9].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d220 = args[1]
					d220.ID = 0
					d222 = d220
					ctx.SyncDesc(&d222)
					if d222.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d222.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d222.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d222 = tmpScalar
					}
					d222 = JITPrepareScmerGoArg(ctx, d222)
					if d222.Loc != LocRegPair && d222.Loc != LocStackPair && d222.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d221 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d222}, 2)
					ctx.StabilizeDescForControlFlow(&d221)
					ctx.FreeDesc(&d220)
					ctx.SyncDesc(&d221)
					if d221.Loc == LocReg || d221.Loc == LocFPReg {
						ctx.ProtectReg(d221.Reg)
					} else if d221.Loc == LocRegPair {
						ctx.ProtectReg(d221.Reg)
						ctx.ProtectReg(d221.Reg2)
					}
					d223 = d221
					if d223.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d223)
					if d223.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d223, int32(bbs[10].PhiBase)+int32(0), 2)
					} else if d223.Loc == LocInputPair {
						ctx.EnsureDesc(&d223)
						ctx.EmitStoreScmerToStack(d223, int32(bbs[10].PhiBase)+int32(0))
					} else if d223.Loc == LocRegPair || d223.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d223, int32(bbs[10].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d223)
						ctx.EmitStoreToStack(d223, int32(bbs[10].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[10].PhiBase)+int32(0))+8)
					}
					if d221.Loc == LocReg || d221.Loc == LocFPReg {
						ctx.UnprotectReg(d221.Reg)
					} else if d221.Loc == LocRegPair {
						ctx.UnprotectReg(d221.Reg)
						ctx.UnprotectReg(d221.Reg2)
					}
					return bbs[10].Render()
					return result
				}
				bbs[10].Render = func() JITValueDesc {
					if bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					bbs[10].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d1 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(32)}
					ctx.ReclaimUntrackedRegs()
					d224 = ctx.EmitGoCallScalar(GoFuncAddr(func() *[1]any { return new([1]any) }), nil, 1)
					d225 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.EnsureDesc(&d3)
					d226 = ctx.EmitGoCallScalar(GoFuncAddr(func(value string) any { return value }), []JITValueDesc{d3}, 2)
					ctx.FreeDesc(&d3)
					ctx.EnsureDesc(&d226)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[1]any, index int, value any) { dst[index] = value }), []JITValueDesc{d224, d225, d226})
					sliceResults227 := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *[1]any) []any { return value[0:1:1] }), []JITValueDesc{d224}, []uint8{3}, []uint8{1})
					d228 = sliceResults227[0]
					d229 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("warning: JIT fallback: %s\n")}
					ctx.EnsureDesc(&d229)
					if d229.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d229.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d229.Imm)
						ptrWord, _ := d229.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d229.Imm.String())))
						d229 = tmpPair
					} else if d229.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d229.Type, Reg: ctx.AllocRegExcept(d229.Reg), Reg2: ctx.AllocRegExcept(d229.Reg)}
						switch d229.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d229)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d229)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d229)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d229)
						d229 = tmpPair
					}
					if d229.Loc != LocRegPair && d229.Loc != LocStackPair && d229.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (fmt.Printf arg0)")
					}
					d228 = JITPrepareGoSliceArg(ctx, d228)
					if d228.Loc != LocRegTriple && d228.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (fmt.Printf arg1)")
					}
					ctx.SyncDesc(&d229)
					ctx.SyncDesc(&d228)
					callResults230 := JITEmitGoCallResults(ctx, GoFuncAddr(fmt.Printf), []JITValueDesc{d229, d228}, []uint8{1, 2}, []uint8{0, 3})
					ctx.FreeDesc(&d229)
					d231 = callResults230[0]
					_ = d231
					d232 = callResults230[1]
					_ = d232
					return bbs[7].Render()
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  39,
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

			// Read the executing binary configuration, rather than the generator build.
			JITEmit:       jitEmitEnabled,
			JITInlineCost: 2,
		},
	})
}

func jitEmitEnabled(_ *JITContext, _ []Scmer, _ []JITValueDesc, _ JITValueDesc) JITValueDesc {
	return JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(jitEnabled)}
}

// jitCompile compiles a Proc to a native function (tagFunc)
// Already compiled functions (tagFunc, tagFuncEnv) are passed through unchanged
func jitCompile(a ...Scmer) Scmer {
	return jitCompileMode(true, a...)
}

// CompileJIT compiles a procedure to native code when this build enables the
// JIT. recursiveLambdas controls whether nested procedures are compiled with
// their parent. Unsupported procedures and builds without JIT support retain
// the original procedure as their interpreter fallback.
func CompileJIT(procedure Scmer, recursiveLambdas bool) Scmer {
	return jitCompileMode(recursiveLambdas, procedure)
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
	if len(a) < 1 || len(a) > 2 {
		panic("jit: expects a procedure and optional execution scope")
	}
	scope := NewNil()
	if len(a) == 2 {
		scope = a[1]
	}
	compileContext := executionContextFrom(scope)
	if err := compileContext.Err(); err != nil {
		panic(err)
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
		if proc != nil && proc.Compiled != nil && proc.Compiled.CodePtr != nil {
			// Deferred compilation returns private native code before its maps
			// are published. Reuse must honor the same execution boundary as a
			// fresh compilation, including transitive callee dependencies.
			if waitForPublication && proc.Compiled.reservation != nil {
				jitPublication.Lock()
				for !proc.Compiled.reservation.ready {
					jitPublication.cond.Wait()
				}
				jitPublication.Unlock()
			}
			return v
		}
		if proc == nil || !atomic.CompareAndSwapUint32(&proc.jitCompiling, 0, 1) {
			return v
		}
		releaseCompiling := true
		defer func() {
			if releaseCompiling {
				atomic.StoreUint32(&proc.jitCompiling, 0)
			}
		}()
		plan := proc.Compiled
		var lambdaTemplates []Scmer
		// Try increasing buffer sizes for overflow retry
		for _, codeCap := range [...]int{16 * 1024, 64 * 1024, 256 * 1024, 1024 * 1024, 4 * 1024 * 1024, 16 * 1024 * 1024} {
			if err := compileContext.Err(); err != nil {
				panic(err)
			}
			ptr, arena, reservation := globalJITPool.Alloc(codeCap)
			buf := &execBuf{ptr: ptr, n: codeCap, arena: arena, reservation: reservation, compileScope: scope, lambdaTemplates: &lambdaTemplates}
			codeLen, roots, dependencies, overflow, hiddenArgs, needsStableArgs, coverage := jitCompileProcToExec(proc, buf, recursiveLambdas)
			if codeLen > 0 {
				code := (*[1 << 30]byte)(ptr)[:codeLen:codeLen]
				if JITLog {
					fmt.Printf("%X\n", code)
				}
				maybeDumpJITCode(ptr, code)
				sourceProc := *proc
				sourceProc.JITCode = 0
				sourceProc.Compiled = nil
				sourceProc.jitCompiling = 0
				jep := &JITEntryPoint{
					StackFrameSize:   buf.stackFrameSize,
					HiddenArgs:       hiddenArgs,
					CodePtr:          ptr,
					CodeLen:          codeLen,
					Arena:            arena,
					ConstRoots:       roots,
					Dependencies:     dependencies,
					reservation:      reservation,
					Proc:             sourceProc,
					RecursiveLambdas: recursiveLambdas,
					NeedsStableArgs:  needsStableArgs,
					Coverage:         coverage,
				}
				if plan != nil {
					jep.CaptureBase = plan.CaptureBase
					jep.CaptureCount = plan.CaptureCount
					jep.CaptureKeys = plan.CaptureKeys
					jep.CaptureSymbols = plan.CaptureSymbols
				}
				runtime.AddCleanup(jep, releaseJITEntryPoint, jitCodeLease{
					pool:  &globalJITPool,
					arena: arena,
					code:  uintptr(ptr),
				})
				if waitForPublication {
					arena.complete(reservation, buf.stackMaps, dependencies)
					targetProc := proc
					if !install {
						copy := sourceProc
						targetProc = &copy
					}
					attachProcJIT(targetProc, jep)
					return Scmer{ptr: (*byte)(unsafe.Pointer(targetProc)), aux: makeAux(tagProc, 0)}
				}

				// The enclosing reservation is not published yet, so return a
				// private Proc which only the enclosing compiler can reach. Install
				// the shared Proc after AddStackMaps publishes this reservation.
				copy := sourceProc
				targetProc := &copy
				attachProcJIT(targetProc, jep)
				var onPublish func()
				if install {
					releaseCompiling = false
					onPublish = func() {
						attachProcJIT(proc, jep)
						atomic.StoreUint32(&proc.jitCompiling, 0)
					}
				}
				arena.completeDeferred(reservation, buf.stackMaps, dependencies, onPublish)
				return Scmer{ptr: (*byte)(unsafe.Pointer(targetProc)), aux: makeAux(tagProc, 0)}
			}
			if waitForPublication {
				arena.complete(reservation, buf.stackMaps, dependencies)
			} else {
				arena.completeDeferred(reservation, buf.stackMaps, dependencies, nil)
			}
			globalJITPool.Free(arena)
			if err := compileContext.Err(); err != nil {
				panic(err)
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
	lambdaTemplates *[]Scmer
	compileScope    Scmer
	ptr             unsafe.Pointer
	n               int       // size
	arena           *jitArena // owning arena (nil for standalone buffers)
	reservation     *jitCodeReservation
	stackMaps       []jitStackMap
	dependencies    []*JITEntryPoint
	stackFrameSize  int32
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
	if entry == nil || entry.DebugName == "" {
		return
	}
	if os.Getenv("MEMCP_JIT_DUMP_DIR") != "" {
		fmt.Printf("jitdump: name=%s code=%p bytes=%d\n", entry.DebugName, entry.CodePtr, entry.CodeLen)
	}
	// Linux perf symbol map: lets `perf report` name JIT machine code instead of
	// showing raw addresses outside any Go function. Format: "<hexAddr> <hexSize> <name>".
	if os.Getenv("MEMCP_PERF_MAP") != "" && entry.CodePtr != nil && entry.CodeLen > 0 {
		path := fmt.Sprintf("/tmp/perf-%d.map", os.Getpid())
		if f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
			fmt.Fprintf(f, "%x %x %s\n", uintptr(entry.CodePtr), entry.CodeLen, entry.DebugName)
			f.Close()
		}
	}
}

func maybeLogJITImportCandidate(name Symbol, entry *JITEntryPoint, selected bool) {
	if os.Getenv("MEMCP_JIT_DUMP_DIR") == "" {
		return
	}
	if entry == nil {
		fmt.Printf("jitdump: import=%s compiled=false\n", name)
		return
	}
	fmt.Printf("jitdump: import=%s selected=%t expressions=%d dynamic-calls=%d direct-procs=%d native-calls=%d inlined-calls=%d\n",
		name, selected, entry.Coverage.Expressions, entry.Coverage.DynamicCalls,
		entry.Coverage.DirectProcs, entry.Coverage.NativeCalls, entry.Coverage.InlinedCalls)
}
