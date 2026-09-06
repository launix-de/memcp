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
	// Owner keeps the entry point which owns shared machine code alive. FuncValue
	// retains the GC-typed Go closure object whose first word is CodePtr.
	Owner      *JITEntryPoint
	FuncValue  any
	HiddenArgs []JITHiddenArg
	CodePtr    unsafe.Pointer   // start of code in arena
	CodeLen    int              // bytes used
	Arena      *jitArena        // owning arena (for free on GC)
	ConstRoots []unsafe.Pointer // GC roots for constants embedded into machine code
	// Dependencies keep directly called JIT entry points and their executable
	// arenas alive for as long as this machine code can branch to them.
	Dependencies []*JITEntryPoint
	Proc         Proc // original Proc for serialization
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
			args = append(args, NewFunc(OptimizeProcToSerialFunction(args[spec.SourceInput])))
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
	// Condition is valid only for LocFlags. LocFlags is an ephemeral result of
	// a comparison that jitgen proved is consumed immediately by the terminating
	// branch in the same basic block. No intervening machine instruction may
	// clobber the architecture's condition state.
	Condition JITCondition
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
	LocClosurePair // One Scmer in the current Go funcval's typed closure environment
	LocFlags       // Ephemeral comparison result consumed immediately by a branch
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

// jitSafepoint is recorded while emitting a Go call. FrameSize is deliberately
// absent: the one-pass emitter only knows the final static frame size after the
// complete function has been written.
type jitSafepoint struct {
	pcOffset        int32
	dynamicSP       int32
	roots           []jitStackRoot
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
	// RegisterBank is supplied by the architecture backend. Generated register
	// plans contain abstract colors and never name a physical register.
	RegisterBank JITRegisterBank
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
	Safepoints []jitSafepoint
	Coverage   JITCoverage

	// Stack frame: emitter locals use [RSP + offset], while register spills use
	// [RBP - offset]. The two zones cannot overlap because the patched frame size
	// is MaxBPOffset + MaxSpillOffset. Epilog: leave; ret.
	BPOffset       int32 // current stack allocation point (grows on alloc, shrinks on free)
	MaxBPOffset    int32 // local-zone high-water mark
	SpillOffset    int32 // current spill-zone allocation point below RBP
	MaxSpillOffset int32 // spill-zone high-water mark

	ProtectedRegs       uint64  // bitmask of registers that must not be spilled
	ProtectedRegCounts  [16]int // per-register protection refcount (supports nested protection)
	RegisterHomeCost    [16]uint16
	RegisterHomeID      [16]uint16
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
	freeCount := 0
	for index := uint8(0); index < ctx.RegisterBank.Count; index++ {
		reg := ctx.RegisterBank.Registers[index]
		bit := uint64(1) << uint(reg)
		if ctx.AllRegs&bit != 0 && ctx.FreeRegs&bit != 0 && ctx.ProtectedRegs&bit == 0 {
			freeCount++
		}
	}
	budget := freeCount - int(ctx.RegisterBank.TemporaryReserve)
	for slotIndex := uint8(0); slotIndex < plan.Count; slotIndex++ {
		slot := plan.Slots[slotIndex]
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
		for laneCount > budget {
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
			budget += int(evicted.width)
		}
		if laneCount > budget {
			continue
		}
		var selected [3]Reg
		selectedCount := uint8(0)
		for index := uint8(0); index < ctx.RegisterBank.Count && int(selectedCount) < laneCount; index++ {
			reg := ctx.RegisterBank.Registers[index]
			bit := uint64(1) << uint(reg)
			if ctx.AllRegs&bit == 0 || ctx.FreeRegs&bit == 0 || ctx.ProtectedRegs&bit != 0 {
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
			ctx.FreeRegs &^= bit
			ctx.ProtectReg(reg)
			ctx.RegisterHomeCost[reg] = slot.Cost
			ctx.RegisterHomeID[reg] = ctx.nextRegisterHomeID
			homes.Registers[int(slot.Color)+int(lane)] = reg
			homes.Available |= 1 << (slot.Color + lane)
			homes.OwnedRegs |= bit
		}
		budget -= laneCount
	}
	return homes
}

func (ctx *JITContext) ReleaseRegisterHomes(homes JITRegisterHomes) {
	for index := uint8(0); index < ctx.RegisterBank.Count; index++ {
		reg := ctx.RegisterBank.Registers[index]
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
	protectedRegs       uint64
	protectedRegCounts  [16]int
	registerHomeCost    [16]uint16
	registerHomeID      [16]uint16
	pinnedRegisterHomes uint64
	regOwnerIDs         [16]uint32
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
	s := jitAllocStateSnapshot{
		freeRegs:            ctx.FreeRegs,
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
	for r := Reg(0); r <= RegR15; r++ {
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
	ctx.FreeRegs = s.freeRegs
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
	for r := Reg(0); r <= RegR15; r++ {
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
	regs  [16]Reg
	offs  [16]int32
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
	for r := Reg(0); r <= RegR15; r++ {
		if (ctx.AllRegs&(1<<uint(r))) == 0 || (ctx.FreeRegs&(1<<uint(r))) != 0 {
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
		ctx.EmitStoreRegMem(r, RegRBP, off)
		if ctx.regHoldsPointer(r) {
			ctx.setStackPointer(jitStackRootFrameBP, off, true)
		}
		p.regs[p.count] = r
		p.offs[p.count] = off
		p.count++
		ctx.RegOwners[r] = nil
		ctx.FreeRegs |= 1 << uint(r)
		if options.ReleaseHomes {
			// The snapshot retains the home metadata for Restore. Leaving it on a
			// now-free register would make a nested plan mistake its own register
			// for an occupied outer bundle.
			ctx.RegisterHomeCost[r] = 0
			ctx.RegisterHomeID[r] = 0
		}
	}
	homeMask := uint64(0)
	for r := Reg(0); r <= RegR15; r++ {
		if !options.ReleaseHomes && ctx.RegisterHomeID[r] != 0 {
			homeMask |= 1 << uint(r)
		}
	}
	// Released homes must not leave pin bits behind: a nested allocation may
	// legitimately use those physical registers until this boundary is restored.
	ctx.PinnedRegisterHomes &= homeMask | resultMask
	ctx.ProtectedRegs &= resultMask | homeMask
	for r := Reg(0); r <= RegR15; r++ {
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
		ctx.EmitMovRegMem(p.regs[i], RegRBP, p.offs[i])
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
	case LocFlags:
		if desc.Reg <= RegR15 {
			owner := ctx.RegOwners[desc.Reg]
			if owner == nil || owner == desc || (desc.ID != 0 && owner.ID == desc.ID) {
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

// GoABIIntRegs lists integer argument/result registers in Go's amd64
// ABIInternal order. R11 is the ninth argument register, so indirect static
// calls use the otherwise reserved R12 register as their call target.
var GoABIIntRegs = []Reg{RegRAX, RegRBX, RegRCX, RegRDI, RegRSI, RegR8, RegR9, RegR10, RegR11}

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
// moved. R12 breaks cycles, but it is also the long-lived input-slice base, so
// cycle resolution preserves it on the stack. R11 cannot be scratch here
// because it is Go ABIInternal's ninth integer argument register.
func (ctx *JITContext) emitParallelRegMoves(moves []jitRegMove) {
	scratchSaved := false
	defer func() {
		if scratchSaved {
			ctx.EmitPopReg(RegR12)
		}
	}()
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
			if !scratchSaved {
				ctx.EmitPushReg(RegR12)
				scratchSaved = true
			}
			cycleDst := moves[0].dst
			if cycleDst != RegR12 {
				ctx.emitMovRegReg(RegR12, cycleDst)
			}
			for i := range moves {
				if moves[i].src == cycleDst {
					moves[i].src = RegR12
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
	ctx.NeedsStableArgs = true
	entryDynamicSP := ctx.DynamicSP
	if numResultWords > len(GoABIIntRegs) {
		panic("jit: too many result words for Go ABI")
	}
	argLocations, stackArgWords := layoutGoCallArgs(argWords)
	if stackArgWords*8 > int(jitGoSpillBytes) {
		panic("jit: Go call arguments exceed reserved spill area")
	}
	// Owner-aware liveness with conservative fallback.
	var liveRegsArr [16]Reg
	liveRegs := ctx.collectLiveRegsForCall(&liveRegsArr)
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
			dstOff := argLocations[i].stackOff
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
		for i := range argWords {
			if !argLocations[i].inReg {
				continue
			}
			target := argLocations[i].reg
			if argWords[i].loc == LocReg && argWords[i].reg != target {
				moves = append(moves, jitRegMove{dst: target, src: argWords[i].reg})
			}
		}
		ctx.emitParallelRegMoves(moves)

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
		ctx.addDynamicStack(int32(resultBytes))
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
	onPublish func()
}

// complete publishes arena metadata in allocation order. Compilation itself
// may run concurrently. Before an entry point becomes reachable, complete also
// waits for every reservation allocated up to that compiler's completion. That
// set includes deferred lambdas embedded in the entry point even when another
// compiler's reservation was interleaved between parent and child.
func (a *jitArena) complete(reservation *jitCodeReservation, maps []jitStackMap) {
	a.completeMode(reservation, maps, true, nil)
}

// completeDeferred records metadata for code which cannot become reachable
// before its enclosing reservation is published. Nested special-form thunks
// use this to avoid waiting on the outer compiler which is currently emitting
// them; the outer completion publishes both reservations in allocation order.
func (a *jitArena) completeDeferred(reservation *jitCodeReservation, maps []jitStackMap, onPublish func()) {
	a.completeMode(reservation, maps, false, onPublish)
}

func (a *jitArena) completeMode(reservation *jitCodeReservation, maps []jitStackMap, wait bool, onPublish func()) {
	if a == nil || reservation == nil {
		return
	}
	a.metaMu.Lock()
	reservation.maps = maps
	reservation.onPublish = onPublish
	reservation.done = true
	waitThrough := 0
	if wait {
		waitThrough = len(a.reservations)
	}
	for a.metaNext < len(a.reservations) && a.reservations[a.metaNext].done {
		ready := a.reservations[a.metaNext]
		publishJITStackMaps(a, ready.maps)
		if ready.onPublish != nil {
			ready.onPublish()
			ready.onPublish = nil
		}
		ready.published = true
		a.metaNext++
	}
	a.metaCond.Broadcast()
	if wait {
		for a.metaNext < waitThrough {
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
			reservation = &jitCodeReservation{offset: a.offset}
			a.metaMu.Lock()
			a.reservations = append(a.reservations, reservation)
			a.metaMu.Unlock()
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
	a.metaCond = sync.NewCond(&a.metaMu)
	a.handle = registerJITArena(a)
	ptr = a.base
	reservation = &jitCodeReservation{}
	a.reservations = append(a.reservations, reservation)
	a.offset = size
	p.arenas = append(p.arenas, a)
	p.mu.Unlock()
	return ptr, a, reservation
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
				var d8 JITValueDesc
				_ = d8
				var d20 JITValueDesc
				_ = d20
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
				var d32 JITValueDesc
				_ = d32
				var d33 JITValueDesc
				_ = d33
				var d36 JITValueDesc
				_ = d36
				var d53 JITValueDesc
				_ = d53
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
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d79 JITValueDesc
				_ = d79
				var d80 JITValueDesc
				_ = d80
				var d81 JITValueDesc
				_ = d81
				var d82 JITValueDesc
				_ = d82
				var d83 JITValueDesc
				_ = d83
				var d86 JITValueDesc
				_ = d86
				var d118 JITValueDesc
				_ = d118
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
				var bbs [6]BBDescriptor
				bbs[2].PhiBase = int32(phiBase0) + int32(0)
				bbs[2].PhiCount = uint16(1)
				bbs[4].PhiBase = int32(phiBase0) + int32(16)
				bbs[4].PhiCount = uint16(1)
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
						d5 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d5)
					}
					ctx.FreeDesc(&d4)
					d6 = d5
					ctx.EnsureDesc(&d6)
					if d6.Loc != LocImm && d6.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
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
					ctx.EmitJump(d6.Condition, lbl7)
					ctx.EmitJmp(lbl8)
					snap10 := d1
					snap11 := d2
					snap12 := d3
					snap13 := d4
					snap14 := d5
					snap15 := d6
					snap16 := d8
					alloc17 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl7)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc17)
					d1 = snap10
					d2 = snap11
					d3 = snap12
					d4 = snap13
					d5 = snap14
					d6 = snap15
					d8 = snap16
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc17)
					d1 = snap10
					d2 = snap11
					d3 = snap12
					d4 = snap13
					d5 = snap14
					d6 = snap15
					d8 = snap16
					ps18 := PhiState{General: true}
					ps18.OverlayValues = make([]JITValueDesc, 9)
					ps18.OverlayValues[1] = d1
					ps18.OverlayValues[2] = d2
					ps18.OverlayValues[3] = d3
					ps18.OverlayValues[4] = d4
					ps18.OverlayValues[5] = d5
					ps18.OverlayValues[6] = d6
					ps18.OverlayValues[8] = d8
					ps18.PhiValues = make([]JITValueDesc, 1)
					d20 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
					ps18.PhiValues[0] = d20
					ps19 := PhiState{General: true}
					ps19.OverlayValues = make([]JITValueDesc, 21)
					ps19.OverlayValues[1] = d1
					ps19.OverlayValues[2] = d2
					ps19.OverlayValues[3] = d3
					ps19.OverlayValues[4] = d4
					ps19.OverlayValues[5] = d5
					ps19.OverlayValues[6] = d6
					ps19.OverlayValues[8] = d8
					ps19.OverlayValues[20] = d20
					snap21 := d1
					snap22 := d2
					snap23 := d3
					snap24 := d4
					snap25 := d5
					snap26 := d6
					snap27 := d8
					snap28 := d20
					alloc29 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps18)
					}
					ctx.RestoreAllocState(alloc29)
					d1 = snap21
					d2 = snap22
					d3 = snap23
					d4 = snap24
					d5 = snap25
					d6 = snap26
					d8 = snap27
					d20 = snap28
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps19)
					}
					return result
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					ctx.ReclaimUntrackedRegs()
					d30 = args[0]
					d30.ID = 0
					d31 = ctx.EmitGetTagDesc(&d30, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d30)
					ctx.EnsureDesc(&d31)
					var d32 JITValueDesc
					if d31.Loc == LocImm {
						d32 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d31.Imm.Int()) == uint64(0xa))}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d31.Reg, 10)
						d32 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondEqual}
						ctx.BindReg(r1, &d32)
					}
					ctx.FreeDesc(&d31)
					d33 = d32
					ctx.EnsureDesc(&d33)
					if d33.Loc != LocImm && d33.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d33.Loc == LocImm {
						if d33.Imm.Bool() {
							if ps.General {
							}
							ps34 := PhiState{General: ps.General}
							ps34.OverlayValues = make([]JITValueDesc, 34)
							ps34.OverlayValues[1] = d1
							ps34.OverlayValues[2] = d2
							ps34.OverlayValues[3] = d3
							ps34.OverlayValues[4] = d4
							ps34.OverlayValues[5] = d5
							ps34.OverlayValues[6] = d6
							ps34.OverlayValues[8] = d8
							ps34.OverlayValues[20] = d20
							ps34.OverlayValues[30] = d30
							ps34.OverlayValues[31] = d31
							ps34.OverlayValues[32] = d32
							ps34.OverlayValues[33] = d33
							return bbs[5].RenderPS(ps34)
						}
						if ps.General {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
						}
						ps35 := PhiState{General: ps.General}
						ps35.OverlayValues = make([]JITValueDesc, 34)
						ps35.OverlayValues[1] = d1
						ps35.OverlayValues[2] = d2
						ps35.OverlayValues[3] = d3
						ps35.OverlayValues[4] = d4
						ps35.OverlayValues[5] = d5
						ps35.OverlayValues[6] = d6
						ps35.OverlayValues[8] = d8
						ps35.OverlayValues[20] = d20
						ps35.OverlayValues[30] = d30
						ps35.OverlayValues[31] = d31
						ps35.OverlayValues[32] = d32
						ps35.OverlayValues[33] = d33
						ps35.PhiValues = make([]JITValueDesc, 1)
						d36 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
						ps35.PhiValues[0] = d36
						return bbs[4].RenderPS(ps35)
					}
					if !ps.General {
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					ctx.EmitJump(d33.Condition, lbl9)
					ctx.EmitJmp(lbl10)
					snap37 := d1
					snap38 := d2
					snap39 := d3
					snap40 := d4
					snap41 := d5
					snap42 := d6
					snap43 := d8
					snap44 := d20
					snap45 := d30
					snap46 := d31
					snap47 := d32
					snap48 := d33
					snap49 := d36
					alloc50 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl6)
					ctx.RestoreAllocState(alloc50)
					d1 = snap37
					d2 = snap38
					d3 = snap39
					d4 = snap40
					d5 = snap41
					d6 = snap42
					d8 = snap43
					d20 = snap44
					d30 = snap45
					d31 = snap46
					d32 = snap47
					d33 = snap48
					d36 = snap49
					ctx.MarkLabel(lbl10)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc50)
					d1 = snap37
					d2 = snap38
					d3 = snap39
					d4 = snap40
					d5 = snap41
					d6 = snap42
					d8 = snap43
					d20 = snap44
					d30 = snap45
					d31 = snap46
					d32 = snap47
					d33 = snap48
					d36 = snap49
					ps51 := PhiState{General: true}
					ps51.OverlayValues = make([]JITValueDesc, 37)
					ps51.OverlayValues[1] = d1
					ps51.OverlayValues[2] = d2
					ps51.OverlayValues[3] = d3
					ps51.OverlayValues[4] = d4
					ps51.OverlayValues[5] = d5
					ps51.OverlayValues[6] = d6
					ps51.OverlayValues[8] = d8
					ps51.OverlayValues[20] = d20
					ps51.OverlayValues[30] = d30
					ps51.OverlayValues[31] = d31
					ps51.OverlayValues[32] = d32
					ps51.OverlayValues[33] = d33
					ps51.OverlayValues[36] = d36
					ps52 := PhiState{General: true}
					ps52.OverlayValues = make([]JITValueDesc, 37)
					ps52.OverlayValues[1] = d1
					ps52.OverlayValues[2] = d2
					ps52.OverlayValues[3] = d3
					ps52.OverlayValues[4] = d4
					ps52.OverlayValues[5] = d5
					ps52.OverlayValues[6] = d6
					ps52.OverlayValues[8] = d8
					ps52.OverlayValues[20] = d20
					ps52.OverlayValues[30] = d30
					ps52.OverlayValues[31] = d31
					ps52.OverlayValues[32] = d32
					ps52.OverlayValues[33] = d33
					ps52.OverlayValues[36] = d36
					ps52.PhiValues = make([]JITValueDesc, 1)
					d53 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ps52.PhiValues[0] = d53
					snap54 := d1
					snap55 := d2
					snap56 := d3
					snap57 := d4
					snap58 := d5
					snap59 := d6
					snap60 := d8
					snap61 := d20
					snap62 := d30
					snap63 := d31
					snap64 := d32
					snap65 := d33
					snap66 := d36
					snap67 := d53
					alloc68 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps52)
					}
					ctx.RestoreAllocState(alloc68)
					d1 = snap54
					d2 = snap55
					d3 = snap56
					d4 = snap57
					d5 = snap58
					d6 = snap59
					d8 = snap60
					d20 = snap61
					d30 = snap62
					d31 = snap63
					d32 = snap64
					d33 = snap65
					d36 = snap66
					d53 = snap67
					if !bbs[5].Rendered {
						return bbs[5].RenderPS(ps51)
					}
					return result
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d69 := ps.PhiValues[0]
							ctx.EnsureDesc(&d69)
							ctx.EmitStoreToStack(d69, int32(bbs[2].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
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
						d70 := JITValueDesc{Loc: LocReg, Type: tagBool, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeBool(result, d70)
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					ctx.ReclaimUntrackedRegs()
					d71 = args[0]
					d71.ID = 0
					ctx.EnsureDesc(&d71)
					ctx.EnsureDesc(&d71)
					d71 = JITPrepareScmerGoArg(ctx, d71)
					ctx.SyncDesc(&d71)
					d72 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Proc), []JITValueDesc{d71}, 1)
					d72.NoHeapPointer = false
					ctx.BindReg(d72.Reg, &d72)
					ctx.FreeDesc(&d71)
					var d73 JITValueDesc
					ctx.EnsureDesc(&d72)
					if d72.Loc == LocImm {
						fieldAddr := uintptr(d72.Imm.Int()) + 0
						r2 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r2, fieldAddr)
						d73 = JITValueDesc{Loc: LocReg, Reg: r2}
						ctx.BindReg(r2, &d73)
					} else {
						off := int32(0)
						baseReg := d72.Reg
						r3 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r3, baseReg, off)
						d73 = JITValueDesc{Loc: LocReg, Reg: r3}
						ctx.BindReg(r3, &d73)
					}
					ctx.FreeDesc(&d72)
					ctx.EnsureDesc(&d73)
					var d74 JITValueDesc
					if d73.Loc == LocImm {
						d74 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d73.Imm.Int()) != uint64(0x0))}
					} else {
						ctx.EmitCmpRegImm32(d73.Reg, 0)
						r4 := ctx.AllocReg()
						ctx.EmitSetcc(r4, CondNotEqual)
						d74 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d74)
					}
					ctx.EnsureDesc(&d74)
					ctx.EmitStoreToStack(d74, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d74)
					ctx.FreeDesc(&d73)
					if ps.General {
					}
					ps75 := PhiState{General: ps.General}
					ps75.OverlayValues = make([]JITValueDesc, 75)
					ps75.OverlayValues[1] = d1
					ps75.OverlayValues[2] = d2
					ps75.OverlayValues[3] = d3
					ps75.OverlayValues[4] = d4
					ps75.OverlayValues[5] = d5
					ps75.OverlayValues[6] = d6
					ps75.OverlayValues[8] = d8
					ps75.OverlayValues[20] = d20
					ps75.OverlayValues[30] = d30
					ps75.OverlayValues[31] = d31
					ps75.OverlayValues[32] = d32
					ps75.OverlayValues[33] = d33
					ps75.OverlayValues[36] = d36
					ps75.OverlayValues[53] = d53
					ps75.OverlayValues[69] = d69
					ps75.OverlayValues[70] = d70
					ps75.OverlayValues[71] = d71
					ps75.OverlayValues[72] = d72
					ps75.OverlayValues[73] = d73
					ps75.OverlayValues[74] = d74
					ps75.PhiValues = make([]JITValueDesc, 1)
					if ps75.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps75)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d76 := ps.PhiValues[0]
							ctx.EnsureDesc(&d76)
							ctx.EmitStoreToStack(d76, int32(bbs[4].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
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
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
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
						d77 = d2
						if d77.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d77)
						ctx.EmitStoreToStack(d77, int32(bbs[2].PhiBase)+int32(0))
						if d2.Loc == LocReg {
							ctx.UnprotectReg(d2.Reg)
						} else if d2.Loc == LocRegPair {
							ctx.UnprotectReg(d2.Reg)
							ctx.UnprotectReg(d2.Reg2)
						}
					}
					ps78 := PhiState{General: ps.General}
					ps78.OverlayValues = make([]JITValueDesc, 78)
					ps78.OverlayValues[1] = d1
					ps78.OverlayValues[2] = d2
					ps78.OverlayValues[3] = d3
					ps78.OverlayValues[4] = d4
					ps78.OverlayValues[5] = d5
					ps78.OverlayValues[6] = d6
					ps78.OverlayValues[8] = d8
					ps78.OverlayValues[20] = d20
					ps78.OverlayValues[30] = d30
					ps78.OverlayValues[31] = d31
					ps78.OverlayValues[32] = d32
					ps78.OverlayValues[33] = d33
					ps78.OverlayValues[36] = d36
					ps78.OverlayValues[53] = d53
					ps78.OverlayValues[69] = d69
					ps78.OverlayValues[70] = d70
					ps78.OverlayValues[71] = d71
					ps78.OverlayValues[72] = d72
					ps78.OverlayValues[73] = d73
					ps78.OverlayValues[74] = d74
					ps78.OverlayValues[76] = d76
					ps78.OverlayValues[77] = d77
					ps78.PhiValues = make([]JITValueDesc, 1)
					d79 = d2
					ps78.PhiValues[0] = d79
					if ps78.General && bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					return bbs[2].RenderPS(ps78)
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
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
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
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					ctx.ReclaimUntrackedRegs()
					d80 = args[0]
					d80.ID = 0
					ctx.EnsureDesc(&d80)
					ctx.EnsureDesc(&d80)
					d80 = JITPrepareScmerGoArg(ctx, d80)
					ctx.SyncDesc(&d80)
					d81 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Proc), []JITValueDesc{d80}, 1)
					d81.NoHeapPointer = false
					ctx.BindReg(d81.Reg, &d81)
					ctx.FreeDesc(&d80)
					ctx.EnsureDesc(&d81)
					var d82 JITValueDesc
					if d81.Loc == LocImm {
						d82 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d81.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d81)
						if d81.Loc != LocReg && d81.Loc != LocRegPair && d81.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r5 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d81.Reg, 0)
						ctx.EmitSetcc(r5, CondNotEqual)
						d82 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
						ctx.BindReg(r5, &d82)
					}
					ctx.FreeDesc(&d81)
					d83 = d82
					ctx.EnsureDesc(&d83)
					if d83.Loc != LocImm && d83.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d83.Loc == LocImm {
						if d83.Imm.Bool() {
							if ps.General {
							}
							ps84 := PhiState{General: ps.General}
							ps84.OverlayValues = make([]JITValueDesc, 84)
							ps84.OverlayValues[1] = d1
							ps84.OverlayValues[2] = d2
							ps84.OverlayValues[3] = d3
							ps84.OverlayValues[4] = d4
							ps84.OverlayValues[5] = d5
							ps84.OverlayValues[6] = d6
							ps84.OverlayValues[8] = d8
							ps84.OverlayValues[20] = d20
							ps84.OverlayValues[30] = d30
							ps84.OverlayValues[31] = d31
							ps84.OverlayValues[32] = d32
							ps84.OverlayValues[33] = d33
							ps84.OverlayValues[36] = d36
							ps84.OverlayValues[53] = d53
							ps84.OverlayValues[69] = d69
							ps84.OverlayValues[70] = d70
							ps84.OverlayValues[71] = d71
							ps84.OverlayValues[72] = d72
							ps84.OverlayValues[73] = d73
							ps84.OverlayValues[74] = d74
							ps84.OverlayValues[76] = d76
							ps84.OverlayValues[77] = d77
							ps84.OverlayValues[79] = d79
							ps84.OverlayValues[80] = d80
							ps84.OverlayValues[81] = d81
							ps84.OverlayValues[82] = d82
							ps84.OverlayValues[83] = d83
							return bbs[3].RenderPS(ps84)
						}
						if ps.General {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
						}
						ps85 := PhiState{General: ps.General}
						ps85.OverlayValues = make([]JITValueDesc, 84)
						ps85.OverlayValues[1] = d1
						ps85.OverlayValues[2] = d2
						ps85.OverlayValues[3] = d3
						ps85.OverlayValues[4] = d4
						ps85.OverlayValues[5] = d5
						ps85.OverlayValues[6] = d6
						ps85.OverlayValues[8] = d8
						ps85.OverlayValues[20] = d20
						ps85.OverlayValues[30] = d30
						ps85.OverlayValues[31] = d31
						ps85.OverlayValues[32] = d32
						ps85.OverlayValues[33] = d33
						ps85.OverlayValues[36] = d36
						ps85.OverlayValues[53] = d53
						ps85.OverlayValues[69] = d69
						ps85.OverlayValues[70] = d70
						ps85.OverlayValues[71] = d71
						ps85.OverlayValues[72] = d72
						ps85.OverlayValues[73] = d73
						ps85.OverlayValues[74] = d74
						ps85.OverlayValues[76] = d76
						ps85.OverlayValues[77] = d77
						ps85.OverlayValues[79] = d79
						ps85.OverlayValues[80] = d80
						ps85.OverlayValues[81] = d81
						ps85.OverlayValues[82] = d82
						ps85.OverlayValues[83] = d83
						ps85.PhiValues = make([]JITValueDesc, 1)
						d86 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
						ps85.PhiValues[0] = d86
						return bbs[4].RenderPS(ps85)
					}
					if !ps.General {
						ps.General = true
						return bbs[5].RenderPS(ps)
					}
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d83.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					snap87 := d1
					snap88 := d2
					snap89 := d3
					snap90 := d4
					snap91 := d5
					snap92 := d6
					snap93 := d8
					snap94 := d20
					snap95 := d30
					snap96 := d31
					snap97 := d32
					snap98 := d33
					snap99 := d36
					snap100 := d53
					snap101 := d69
					snap102 := d70
					snap103 := d71
					snap104 := d72
					snap105 := d73
					snap106 := d74
					snap107 := d76
					snap108 := d77
					snap109 := d79
					snap110 := d80
					snap111 := d81
					snap112 := d82
					snap113 := d83
					snap114 := d86
					alloc115 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl4)
					ctx.RestoreAllocState(alloc115)
					d1 = snap87
					d2 = snap88
					d3 = snap89
					d4 = snap90
					d5 = snap91
					d6 = snap92
					d8 = snap93
					d20 = snap94
					d30 = snap95
					d31 = snap96
					d32 = snap97
					d33 = snap98
					d36 = snap99
					d53 = snap100
					d69 = snap101
					d70 = snap102
					d71 = snap103
					d72 = snap104
					d73 = snap105
					d74 = snap106
					d76 = snap107
					d77 = snap108
					d79 = snap109
					d80 = snap110
					d81 = snap111
					d82 = snap112
					d83 = snap113
					d86 = snap114
					ctx.MarkLabel(lbl12)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc115)
					d1 = snap87
					d2 = snap88
					d3 = snap89
					d4 = snap90
					d5 = snap91
					d6 = snap92
					d8 = snap93
					d20 = snap94
					d30 = snap95
					d31 = snap96
					d32 = snap97
					d33 = snap98
					d36 = snap99
					d53 = snap100
					d69 = snap101
					d70 = snap102
					d71 = snap103
					d72 = snap104
					d73 = snap105
					d74 = snap106
					d76 = snap107
					d77 = snap108
					d79 = snap109
					d80 = snap110
					d81 = snap111
					d82 = snap112
					d83 = snap113
					d86 = snap114
					ps116 := PhiState{General: true}
					ps116.OverlayValues = make([]JITValueDesc, 87)
					ps116.OverlayValues[1] = d1
					ps116.OverlayValues[2] = d2
					ps116.OverlayValues[3] = d3
					ps116.OverlayValues[4] = d4
					ps116.OverlayValues[5] = d5
					ps116.OverlayValues[6] = d6
					ps116.OverlayValues[8] = d8
					ps116.OverlayValues[20] = d20
					ps116.OverlayValues[30] = d30
					ps116.OverlayValues[31] = d31
					ps116.OverlayValues[32] = d32
					ps116.OverlayValues[33] = d33
					ps116.OverlayValues[36] = d36
					ps116.OverlayValues[53] = d53
					ps116.OverlayValues[69] = d69
					ps116.OverlayValues[70] = d70
					ps116.OverlayValues[71] = d71
					ps116.OverlayValues[72] = d72
					ps116.OverlayValues[73] = d73
					ps116.OverlayValues[74] = d74
					ps116.OverlayValues[76] = d76
					ps116.OverlayValues[77] = d77
					ps116.OverlayValues[79] = d79
					ps116.OverlayValues[80] = d80
					ps116.OverlayValues[81] = d81
					ps116.OverlayValues[82] = d82
					ps116.OverlayValues[83] = d83
					ps116.OverlayValues[86] = d86
					ps117 := PhiState{General: true}
					ps117.OverlayValues = make([]JITValueDesc, 87)
					ps117.OverlayValues[1] = d1
					ps117.OverlayValues[2] = d2
					ps117.OverlayValues[3] = d3
					ps117.OverlayValues[4] = d4
					ps117.OverlayValues[5] = d5
					ps117.OverlayValues[6] = d6
					ps117.OverlayValues[8] = d8
					ps117.OverlayValues[20] = d20
					ps117.OverlayValues[30] = d30
					ps117.OverlayValues[31] = d31
					ps117.OverlayValues[32] = d32
					ps117.OverlayValues[33] = d33
					ps117.OverlayValues[36] = d36
					ps117.OverlayValues[53] = d53
					ps117.OverlayValues[69] = d69
					ps117.OverlayValues[70] = d70
					ps117.OverlayValues[71] = d71
					ps117.OverlayValues[72] = d72
					ps117.OverlayValues[73] = d73
					ps117.OverlayValues[74] = d74
					ps117.OverlayValues[76] = d76
					ps117.OverlayValues[77] = d77
					ps117.OverlayValues[79] = d79
					ps117.OverlayValues[80] = d80
					ps117.OverlayValues[81] = d81
					ps117.OverlayValues[82] = d82
					ps117.OverlayValues[83] = d83
					ps117.OverlayValues[86] = d86
					ps117.PhiValues = make([]JITValueDesc, 1)
					d118 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ps117.PhiValues[0] = d118
					snap119 := d1
					snap120 := d2
					snap121 := d3
					snap122 := d4
					snap123 := d5
					snap124 := d6
					snap125 := d8
					snap126 := d20
					snap127 := d30
					snap128 := d31
					snap129 := d32
					snap130 := d33
					snap131 := d36
					snap132 := d53
					snap133 := d69
					snap134 := d70
					snap135 := d71
					snap136 := d72
					snap137 := d73
					snap138 := d74
					snap139 := d76
					snap140 := d77
					snap141 := d79
					snap142 := d80
					snap143 := d81
					snap144 := d82
					snap145 := d83
					snap146 := d86
					snap147 := d118
					alloc148 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps117)
					}
					ctx.RestoreAllocState(alloc148)
					d1 = snap119
					d2 = snap120
					d3 = snap121
					d4 = snap122
					d5 = snap123
					d6 = snap124
					d8 = snap125
					d20 = snap126
					d30 = snap127
					d31 = snap128
					d32 = snap129
					d33 = snap130
					d36 = snap131
					d53 = snap132
					d69 = snap133
					d70 = snap134
					d71 = snap135
					d72 = snap136
					d73 = snap137
					d74 = snap138
					d76 = snap139
					d77 = snap140
					d79 = snap141
					d80 = snap142
					d81 = snap143
					d82 = snap144
					d83 = snap145
					d86 = snap146
					d118 = snap147
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps116)
					}
					return result
					ctx.FreeDesc(&d82)
					return result
				}
				ps149 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps149)
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
			compiled := value.GetTag() == tagJIT || (value.GetTag() == tagProc && value.Proc() != nil && value.Proc().JITCode != 0)
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
				var d9 JITValueDesc
				_ = d9
				var d22 JITValueDesc
				_ = d22
				var d33 JITValueDesc
				_ = d33
				var d34 JITValueDesc
				_ = d34
				var d35 JITValueDesc
				_ = d35
				var d38 JITValueDesc
				_ = d38
				var d55 JITValueDesc
				_ = d55
				var d71 JITValueDesc
				_ = d71
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				var d75 JITValueDesc
				_ = d75
				var d77 JITValueDesc
				_ = d77
				var d78 JITValueDesc
				_ = d78
				var d80 JITValueDesc
				_ = d80
				var d81 JITValueDesc
				_ = d81
				var d82 JITValueDesc
				_ = d82
				var d83 JITValueDesc
				_ = d83
				var d86 JITValueDesc
				_ = d86
				var d115 JITValueDesc
				_ = d115
				var d143 JITValueDesc
				_ = d143
				var d144 JITValueDesc
				_ = d144
				var d145 JITValueDesc
				_ = d145
				var d146 JITValueDesc
				_ = d146
				var d147 JITValueDesc
				_ = d147
				var d149 JITValueDesc
				_ = d149
				var d151 JITValueDesc
				_ = d151
				var d186 JITValueDesc
				_ = d186
				var d189 JITValueDesc
				_ = d189
				var d226 JITValueDesc
				_ = d226
				var d305 JITValueDesc
				_ = d305
				var d306 JITValueDesc
				_ = d306
				var d307 JITValueDesc
				_ = d307
				var d308 JITValueDesc
				_ = d308
				var d310 JITValueDesc
				_ = d310
				var d311 JITValueDesc
				_ = d311
				var d312 JITValueDesc
				_ = d312
				var d313 JITValueDesc
				_ = d313
				var d314 JITValueDesc
				_ = d314
				var d316 JITValueDesc
				_ = d316
				var d317 JITValueDesc
				_ = d317
				var d319 JITValueDesc
				_ = d319
				var d320 JITValueDesc
				_ = d320
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(48))
				var bbs [11]BBDescriptor
				bbs[2].PhiBase = int32(phiBase0) + int32(0)
				bbs[2].PhiCount = uint16(1)
				bbs[4].PhiBase = int32(phiBase0) + int32(16)
				bbs[4].PhiCount = uint16(1)
				bbs[10].PhiBase = int32(phiBase0) + int32(32)
				bbs[10].PhiCount = uint16(1)
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
					ctx.EmitJump(d7.Condition, lbl12)
					ctx.EmitJmp(lbl13)
					snap11 := d1
					snap12 := d2
					snap13 := d3
					snap14 := d4
					snap15 := d5
					snap16 := d6
					snap17 := d7
					snap18 := d9
					alloc19 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl12)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc19)
					d1 = snap11
					d2 = snap12
					d3 = snap13
					d4 = snap14
					d5 = snap15
					d6 = snap16
					d7 = snap17
					d9 = snap18
					ctx.MarkLabel(lbl13)
					ctx.EmitJmp(lbl2)
					ctx.RestoreAllocState(alloc19)
					d1 = snap11
					d2 = snap12
					d3 = snap13
					d4 = snap14
					d5 = snap15
					d6 = snap16
					d7 = snap17
					d9 = snap18
					ps20 := PhiState{General: true}
					ps20.OverlayValues = make([]JITValueDesc, 10)
					ps20.OverlayValues[1] = d1
					ps20.OverlayValues[2] = d2
					ps20.OverlayValues[3] = d3
					ps20.OverlayValues[4] = d4
					ps20.OverlayValues[5] = d5
					ps20.OverlayValues[6] = d6
					ps20.OverlayValues[7] = d7
					ps20.OverlayValues[9] = d9
					ps20.PhiValues = make([]JITValueDesc, 1)
					d22 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
					ps20.PhiValues[0] = d22
					ps21 := PhiState{General: true}
					ps21.OverlayValues = make([]JITValueDesc, 23)
					ps21.OverlayValues[1] = d1
					ps21.OverlayValues[2] = d2
					ps21.OverlayValues[3] = d3
					ps21.OverlayValues[4] = d4
					ps21.OverlayValues[5] = d5
					ps21.OverlayValues[6] = d6
					ps21.OverlayValues[7] = d7
					ps21.OverlayValues[9] = d9
					ps21.OverlayValues[22] = d22
					snap23 := d1
					snap24 := d2
					snap25 := d3
					snap26 := d4
					snap27 := d5
					snap28 := d6
					snap29 := d7
					snap30 := d9
					snap31 := d22
					alloc32 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps20)
					}
					ctx.RestoreAllocState(alloc32)
					d1 = snap23
					d2 = snap24
					d3 = snap25
					d4 = snap26
					d5 = snap27
					d6 = snap28
					d7 = snap29
					d9 = snap30
					d22 = snap31
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps21)
					}
					return result
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					ctx.ReclaimUntrackedRegs()
					d33 = ctx.EmitGetTagDesc(&d4, JITValueDesc{Loc: LocAny})
					ctx.EnsureDesc(&d33)
					var d34 JITValueDesc
					if d33.Loc == LocImm {
						d34 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d33.Imm.Int()) == uint64(0xa))}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d33.Reg, 10)
						d34 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondEqual}
						ctx.BindReg(r1, &d34)
					}
					ctx.FreeDesc(&d33)
					d35 = d34
					ctx.EnsureDesc(&d35)
					if d35.Loc != LocImm && d35.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d35.Loc == LocImm {
						if d35.Imm.Bool() {
							if ps.General {
							}
							ps36 := PhiState{General: ps.General}
							ps36.OverlayValues = make([]JITValueDesc, 36)
							ps36.OverlayValues[1] = d1
							ps36.OverlayValues[2] = d2
							ps36.OverlayValues[3] = d3
							ps36.OverlayValues[4] = d4
							ps36.OverlayValues[5] = d5
							ps36.OverlayValues[6] = d6
							ps36.OverlayValues[7] = d7
							ps36.OverlayValues[9] = d9
							ps36.OverlayValues[22] = d22
							ps36.OverlayValues[33] = d33
							ps36.OverlayValues[34] = d34
							ps36.OverlayValues[35] = d35
							return bbs[5].RenderPS(ps36)
						}
						if ps.General {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
						}
						ps37 := PhiState{General: ps.General}
						ps37.OverlayValues = make([]JITValueDesc, 36)
						ps37.OverlayValues[1] = d1
						ps37.OverlayValues[2] = d2
						ps37.OverlayValues[3] = d3
						ps37.OverlayValues[4] = d4
						ps37.OverlayValues[5] = d5
						ps37.OverlayValues[6] = d6
						ps37.OverlayValues[7] = d7
						ps37.OverlayValues[9] = d9
						ps37.OverlayValues[22] = d22
						ps37.OverlayValues[33] = d33
						ps37.OverlayValues[34] = d34
						ps37.OverlayValues[35] = d35
						ps37.PhiValues = make([]JITValueDesc, 1)
						d38 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
						ps37.PhiValues[0] = d38
						return bbs[4].RenderPS(ps37)
					}
					if !ps.General {
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					ctx.EmitJump(d35.Condition, lbl14)
					ctx.EmitJmp(lbl15)
					snap39 := d1
					snap40 := d2
					snap41 := d3
					snap42 := d4
					snap43 := d5
					snap44 := d6
					snap45 := d7
					snap46 := d9
					snap47 := d22
					snap48 := d33
					snap49 := d34
					snap50 := d35
					snap51 := d38
					alloc52 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl14)
					ctx.EmitJmp(lbl6)
					ctx.RestoreAllocState(alloc52)
					d1 = snap39
					d2 = snap40
					d3 = snap41
					d4 = snap42
					d5 = snap43
					d6 = snap44
					d7 = snap45
					d9 = snap46
					d22 = snap47
					d33 = snap48
					d34 = snap49
					d35 = snap50
					d38 = snap51
					ctx.MarkLabel(lbl15)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc52)
					d1 = snap39
					d2 = snap40
					d3 = snap41
					d4 = snap42
					d5 = snap43
					d6 = snap44
					d7 = snap45
					d9 = snap46
					d22 = snap47
					d33 = snap48
					d34 = snap49
					d35 = snap50
					d38 = snap51
					ps53 := PhiState{General: true}
					ps53.OverlayValues = make([]JITValueDesc, 39)
					ps53.OverlayValues[1] = d1
					ps53.OverlayValues[2] = d2
					ps53.OverlayValues[3] = d3
					ps53.OverlayValues[4] = d4
					ps53.OverlayValues[5] = d5
					ps53.OverlayValues[6] = d6
					ps53.OverlayValues[7] = d7
					ps53.OverlayValues[9] = d9
					ps53.OverlayValues[22] = d22
					ps53.OverlayValues[33] = d33
					ps53.OverlayValues[34] = d34
					ps53.OverlayValues[35] = d35
					ps53.OverlayValues[38] = d38
					ps54 := PhiState{General: true}
					ps54.OverlayValues = make([]JITValueDesc, 39)
					ps54.OverlayValues[1] = d1
					ps54.OverlayValues[2] = d2
					ps54.OverlayValues[3] = d3
					ps54.OverlayValues[4] = d4
					ps54.OverlayValues[5] = d5
					ps54.OverlayValues[6] = d6
					ps54.OverlayValues[7] = d7
					ps54.OverlayValues[9] = d9
					ps54.OverlayValues[22] = d22
					ps54.OverlayValues[33] = d33
					ps54.OverlayValues[34] = d34
					ps54.OverlayValues[35] = d35
					ps54.OverlayValues[38] = d38
					ps54.PhiValues = make([]JITValueDesc, 1)
					d55 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ps54.PhiValues[0] = d55
					snap56 := d1
					snap57 := d2
					snap58 := d3
					snap59 := d4
					snap60 := d5
					snap61 := d6
					snap62 := d7
					snap63 := d9
					snap64 := d22
					snap65 := d33
					snap66 := d34
					snap67 := d35
					snap68 := d38
					snap69 := d55
					alloc70 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps54)
					}
					ctx.RestoreAllocState(alloc70)
					d1 = snap56
					d2 = snap57
					d3 = snap58
					d4 = snap59
					d5 = snap60
					d6 = snap61
					d7 = snap62
					d9 = snap63
					d22 = snap64
					d33 = snap65
					d34 = snap66
					d35 = snap67
					d38 = snap68
					d55 = snap69
					if !bbs[5].Rendered {
						return bbs[5].RenderPS(ps53)
					}
					return result
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d71 := ps.PhiValues[0]
							ctx.EnsureDesc(&d71)
							ctx.EmitStoreToStack(d71, int32(bbs[2].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					ps72 := PhiState{General: ps.General}
					ps72.OverlayValues = make([]JITValueDesc, 72)
					ps72.OverlayValues[1] = d1
					ps72.OverlayValues[2] = d2
					ps72.OverlayValues[3] = d3
					ps72.OverlayValues[4] = d4
					ps72.OverlayValues[5] = d5
					ps72.OverlayValues[6] = d6
					ps72.OverlayValues[7] = d7
					ps72.OverlayValues[9] = d9
					ps72.OverlayValues[22] = d22
					ps72.OverlayValues[33] = d33
					ps72.OverlayValues[34] = d34
					ps72.OverlayValues[35] = d35
					ps72.OverlayValues[38] = d38
					ps72.OverlayValues[55] = d55
					ps72.OverlayValues[71] = d71
					return bbs[7].RenderPS(ps72)
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					d4 = JITPrepareScmerGoArg(ctx, d4)
					ctx.SyncDesc(&d4)
					d73 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Proc), []JITValueDesc{d4}, 1)
					d73.NoHeapPointer = false
					ctx.BindReg(d73.Reg, &d73)
					var d74 JITValueDesc
					ctx.EnsureDesc(&d73)
					if d73.Loc == LocImm {
						fieldAddr := uintptr(d73.Imm.Int()) + 0
						r2 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r2, fieldAddr)
						d74 = JITValueDesc{Loc: LocReg, Reg: r2}
						ctx.BindReg(r2, &d74)
					} else {
						off := int32(0)
						baseReg := d73.Reg
						r3 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r3, baseReg, off)
						d74 = JITValueDesc{Loc: LocReg, Reg: r3}
						ctx.BindReg(r3, &d74)
					}
					ctx.FreeDesc(&d73)
					ctx.EnsureDesc(&d74)
					var d75 JITValueDesc
					if d74.Loc == LocImm {
						d75 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d74.Imm.Int()) != uint64(0x0))}
					} else {
						ctx.EmitCmpRegImm32(d74.Reg, 0)
						r4 := ctx.AllocReg()
						ctx.EmitSetcc(r4, CondNotEqual)
						d75 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d75)
					}
					ctx.EnsureDesc(&d75)
					ctx.EmitStoreToStack(d75, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d75)
					ctx.FreeDesc(&d74)
					if ps.General {
					}
					ps76 := PhiState{General: ps.General}
					ps76.OverlayValues = make([]JITValueDesc, 76)
					ps76.OverlayValues[1] = d1
					ps76.OverlayValues[2] = d2
					ps76.OverlayValues[3] = d3
					ps76.OverlayValues[4] = d4
					ps76.OverlayValues[5] = d5
					ps76.OverlayValues[6] = d6
					ps76.OverlayValues[7] = d7
					ps76.OverlayValues[9] = d9
					ps76.OverlayValues[22] = d22
					ps76.OverlayValues[33] = d33
					ps76.OverlayValues[34] = d34
					ps76.OverlayValues[35] = d35
					ps76.OverlayValues[38] = d38
					ps76.OverlayValues[55] = d55
					ps76.OverlayValues[71] = d71
					ps76.OverlayValues[73] = d73
					ps76.OverlayValues[74] = d74
					ps76.OverlayValues[75] = d75
					ps76.PhiValues = make([]JITValueDesc, 1)
					if ps76.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps76)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d77 := ps.PhiValues[0]
							ctx.EnsureDesc(&d77)
							ctx.EmitStoreToStack(d77, int32(bbs[4].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
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
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
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
						d78 = d2
						if d78.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d78)
						ctx.EmitStoreToStack(d78, int32(bbs[2].PhiBase)+int32(0))
						if d2.Loc == LocReg {
							ctx.UnprotectReg(d2.Reg)
						} else if d2.Loc == LocRegPair {
							ctx.UnprotectReg(d2.Reg)
							ctx.UnprotectReg(d2.Reg2)
						}
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
					ps79.OverlayValues[22] = d22
					ps79.OverlayValues[33] = d33
					ps79.OverlayValues[34] = d34
					ps79.OverlayValues[35] = d35
					ps79.OverlayValues[38] = d38
					ps79.OverlayValues[55] = d55
					ps79.OverlayValues[71] = d71
					ps79.OverlayValues[73] = d73
					ps79.OverlayValues[74] = d74
					ps79.OverlayValues[75] = d75
					ps79.OverlayValues[77] = d77
					ps79.OverlayValues[78] = d78
					ps79.PhiValues = make([]JITValueDesc, 1)
					d80 = d2
					ps79.PhiValues[0] = d80
					if ps79.General && bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					return bbs[2].RenderPS(ps79)
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
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
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					d4 = JITPrepareScmerGoArg(ctx, d4)
					ctx.SyncDesc(&d4)
					d81 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Proc), []JITValueDesc{d4}, 1)
					d81.NoHeapPointer = false
					ctx.BindReg(d81.Reg, &d81)
					ctx.EnsureDesc(&d81)
					var d82 JITValueDesc
					if d81.Loc == LocImm {
						d82 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d81.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d81)
						if d81.Loc != LocReg && d81.Loc != LocRegPair && d81.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r5 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d81.Reg, 0)
						ctx.EmitSetcc(r5, CondNotEqual)
						d82 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
						ctx.BindReg(r5, &d82)
					}
					ctx.FreeDesc(&d81)
					d83 = d82
					ctx.EnsureDesc(&d83)
					if d83.Loc != LocImm && d83.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d83.Loc == LocImm {
						if d83.Imm.Bool() {
							if ps.General {
							}
							ps84 := PhiState{General: ps.General}
							ps84.OverlayValues = make([]JITValueDesc, 84)
							ps84.OverlayValues[1] = d1
							ps84.OverlayValues[2] = d2
							ps84.OverlayValues[3] = d3
							ps84.OverlayValues[4] = d4
							ps84.OverlayValues[5] = d5
							ps84.OverlayValues[6] = d6
							ps84.OverlayValues[7] = d7
							ps84.OverlayValues[9] = d9
							ps84.OverlayValues[22] = d22
							ps84.OverlayValues[33] = d33
							ps84.OverlayValues[34] = d34
							ps84.OverlayValues[35] = d35
							ps84.OverlayValues[38] = d38
							ps84.OverlayValues[55] = d55
							ps84.OverlayValues[71] = d71
							ps84.OverlayValues[73] = d73
							ps84.OverlayValues[74] = d74
							ps84.OverlayValues[75] = d75
							ps84.OverlayValues[77] = d77
							ps84.OverlayValues[78] = d78
							ps84.OverlayValues[80] = d80
							ps84.OverlayValues[81] = d81
							ps84.OverlayValues[82] = d82
							ps84.OverlayValues[83] = d83
							return bbs[3].RenderPS(ps84)
						}
						if ps.General {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
						}
						ps85 := PhiState{General: ps.General}
						ps85.OverlayValues = make([]JITValueDesc, 84)
						ps85.OverlayValues[1] = d1
						ps85.OverlayValues[2] = d2
						ps85.OverlayValues[3] = d3
						ps85.OverlayValues[4] = d4
						ps85.OverlayValues[5] = d5
						ps85.OverlayValues[6] = d6
						ps85.OverlayValues[7] = d7
						ps85.OverlayValues[9] = d9
						ps85.OverlayValues[22] = d22
						ps85.OverlayValues[33] = d33
						ps85.OverlayValues[34] = d34
						ps85.OverlayValues[35] = d35
						ps85.OverlayValues[38] = d38
						ps85.OverlayValues[55] = d55
						ps85.OverlayValues[71] = d71
						ps85.OverlayValues[73] = d73
						ps85.OverlayValues[74] = d74
						ps85.OverlayValues[75] = d75
						ps85.OverlayValues[77] = d77
						ps85.OverlayValues[78] = d78
						ps85.OverlayValues[80] = d80
						ps85.OverlayValues[81] = d81
						ps85.OverlayValues[82] = d82
						ps85.OverlayValues[83] = d83
						ps85.PhiValues = make([]JITValueDesc, 1)
						d86 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
						ps85.PhiValues[0] = d86
						return bbs[4].RenderPS(ps85)
					}
					if !ps.General {
						ps.General = true
						return bbs[5].RenderPS(ps)
					}
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d83.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl16)
					ctx.EmitJmp(lbl17)
					snap87 := d1
					snap88 := d2
					snap89 := d3
					snap90 := d4
					snap91 := d5
					snap92 := d6
					snap93 := d7
					snap94 := d9
					snap95 := d22
					snap96 := d33
					snap97 := d34
					snap98 := d35
					snap99 := d38
					snap100 := d55
					snap101 := d71
					snap102 := d73
					snap103 := d74
					snap104 := d75
					snap105 := d77
					snap106 := d78
					snap107 := d80
					snap108 := d81
					snap109 := d82
					snap110 := d83
					snap111 := d86
					alloc112 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl4)
					ctx.RestoreAllocState(alloc112)
					d1 = snap87
					d2 = snap88
					d3 = snap89
					d4 = snap90
					d5 = snap91
					d6 = snap92
					d7 = snap93
					d9 = snap94
					d22 = snap95
					d33 = snap96
					d34 = snap97
					d35 = snap98
					d38 = snap99
					d55 = snap100
					d71 = snap101
					d73 = snap102
					d74 = snap103
					d75 = snap104
					d77 = snap105
					d78 = snap106
					d80 = snap107
					d81 = snap108
					d82 = snap109
					d83 = snap110
					d86 = snap111
					ctx.MarkLabel(lbl17)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc112)
					d1 = snap87
					d2 = snap88
					d3 = snap89
					d4 = snap90
					d5 = snap91
					d6 = snap92
					d7 = snap93
					d9 = snap94
					d22 = snap95
					d33 = snap96
					d34 = snap97
					d35 = snap98
					d38 = snap99
					d55 = snap100
					d71 = snap101
					d73 = snap102
					d74 = snap103
					d75 = snap104
					d77 = snap105
					d78 = snap106
					d80 = snap107
					d81 = snap108
					d82 = snap109
					d83 = snap110
					d86 = snap111
					ps113 := PhiState{General: true}
					ps113.OverlayValues = make([]JITValueDesc, 87)
					ps113.OverlayValues[1] = d1
					ps113.OverlayValues[2] = d2
					ps113.OverlayValues[3] = d3
					ps113.OverlayValues[4] = d4
					ps113.OverlayValues[5] = d5
					ps113.OverlayValues[6] = d6
					ps113.OverlayValues[7] = d7
					ps113.OverlayValues[9] = d9
					ps113.OverlayValues[22] = d22
					ps113.OverlayValues[33] = d33
					ps113.OverlayValues[34] = d34
					ps113.OverlayValues[35] = d35
					ps113.OverlayValues[38] = d38
					ps113.OverlayValues[55] = d55
					ps113.OverlayValues[71] = d71
					ps113.OverlayValues[73] = d73
					ps113.OverlayValues[74] = d74
					ps113.OverlayValues[75] = d75
					ps113.OverlayValues[77] = d77
					ps113.OverlayValues[78] = d78
					ps113.OverlayValues[80] = d80
					ps113.OverlayValues[81] = d81
					ps113.OverlayValues[82] = d82
					ps113.OverlayValues[83] = d83
					ps113.OverlayValues[86] = d86
					ps114 := PhiState{General: true}
					ps114.OverlayValues = make([]JITValueDesc, 87)
					ps114.OverlayValues[1] = d1
					ps114.OverlayValues[2] = d2
					ps114.OverlayValues[3] = d3
					ps114.OverlayValues[4] = d4
					ps114.OverlayValues[5] = d5
					ps114.OverlayValues[6] = d6
					ps114.OverlayValues[7] = d7
					ps114.OverlayValues[9] = d9
					ps114.OverlayValues[22] = d22
					ps114.OverlayValues[33] = d33
					ps114.OverlayValues[34] = d34
					ps114.OverlayValues[35] = d35
					ps114.OverlayValues[38] = d38
					ps114.OverlayValues[55] = d55
					ps114.OverlayValues[71] = d71
					ps114.OverlayValues[73] = d73
					ps114.OverlayValues[74] = d74
					ps114.OverlayValues[75] = d75
					ps114.OverlayValues[77] = d77
					ps114.OverlayValues[78] = d78
					ps114.OverlayValues[80] = d80
					ps114.OverlayValues[81] = d81
					ps114.OverlayValues[82] = d82
					ps114.OverlayValues[83] = d83
					ps114.OverlayValues[86] = d86
					ps114.PhiValues = make([]JITValueDesc, 1)
					d115 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ps114.PhiValues[0] = d115
					snap116 := d1
					snap117 := d2
					snap118 := d3
					snap119 := d4
					snap120 := d5
					snap121 := d6
					snap122 := d7
					snap123 := d9
					snap124 := d22
					snap125 := d33
					snap126 := d34
					snap127 := d35
					snap128 := d38
					snap129 := d55
					snap130 := d71
					snap131 := d73
					snap132 := d74
					snap133 := d75
					snap134 := d77
					snap135 := d78
					snap136 := d80
					snap137 := d81
					snap138 := d82
					snap139 := d83
					snap140 := d86
					snap141 := d115
					alloc142 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps114)
					}
					ctx.RestoreAllocState(alloc142)
					d1 = snap116
					d2 = snap117
					d3 = snap118
					d4 = snap119
					d5 = snap120
					d6 = snap121
					d7 = snap122
					d9 = snap123
					d22 = snap124
					d33 = snap125
					d34 = snap126
					d35 = snap127
					d38 = snap128
					d55 = snap129
					d71 = snap130
					d73 = snap131
					d74 = snap132
					d75 = snap133
					d77 = snap134
					d78 = snap135
					d80 = snap136
					d81 = snap137
					d82 = snap138
					d83 = snap139
					d86 = snap140
					d115 = snap141
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps113)
					}
					return result
					ctx.FreeDesc(&d82)
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
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
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != LocNone {
						d83 = ps.OverlayValues[83]
					}
					if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != LocNone {
						d86 = ps.OverlayValues[86]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					d4 = JITPrepareScmerGoArg(ctx, d4)
					d143 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d143.Loc == LocRegPair || d143.Loc == LocStackPair || d143.Loc == LocRegTriple || d143.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d4)
					ctx.SyncDesc(&d143)
					d144 = ctx.EmitGoCallScalar(GoFuncAddr(SerializeToString), []JITValueDesc{d4, d143}, 2)
					d144.NoHeapPointer = false
					ctx.BindReg(d144.Reg, &d144)
					ctx.BindReg(d144.Reg2, &d144)
					ctx.StabilizeDescForControlFlow(&d144)
					d145 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d145)
					var d146 JITValueDesc
					if d145.Loc == LocImm {
						d146 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d145.Imm.Int() > 1)}
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d145.Reg, 1)
						d146 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r6, Condition: CondSignedGreater}
						ctx.BindReg(r6, &d146)
					}
					ctx.FreeDesc(&d145)
					d147 = d146
					ctx.EnsureDesc(&d147)
					if d147.Loc != LocImm && d147.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d147.Loc == LocImm {
						if d147.Imm.Bool() {
							if ps.General {
							}
							ps148 := PhiState{General: ps.General}
							ps148.OverlayValues = make([]JITValueDesc, 148)
							ps148.OverlayValues[1] = d1
							ps148.OverlayValues[2] = d2
							ps148.OverlayValues[3] = d3
							ps148.OverlayValues[4] = d4
							ps148.OverlayValues[5] = d5
							ps148.OverlayValues[6] = d6
							ps148.OverlayValues[7] = d7
							ps148.OverlayValues[9] = d9
							ps148.OverlayValues[22] = d22
							ps148.OverlayValues[33] = d33
							ps148.OverlayValues[34] = d34
							ps148.OverlayValues[35] = d35
							ps148.OverlayValues[38] = d38
							ps148.OverlayValues[55] = d55
							ps148.OverlayValues[71] = d71
							ps148.OverlayValues[73] = d73
							ps148.OverlayValues[74] = d74
							ps148.OverlayValues[75] = d75
							ps148.OverlayValues[77] = d77
							ps148.OverlayValues[78] = d78
							ps148.OverlayValues[80] = d80
							ps148.OverlayValues[81] = d81
							ps148.OverlayValues[82] = d82
							ps148.OverlayValues[83] = d83
							ps148.OverlayValues[86] = d86
							ps148.OverlayValues[115] = d115
							ps148.OverlayValues[143] = d143
							ps148.OverlayValues[144] = d144
							ps148.OverlayValues[145] = d145
							ps148.OverlayValues[146] = d146
							ps148.OverlayValues[147] = d147
							return bbs[9].RenderPS(ps148)
						}
						if ps.General {
							ctx.SyncDesc(&d144)
							if d144.Loc == LocReg {
								ctx.ProtectReg(d144.Reg)
							} else if d144.Loc == LocRegPair {
								ctx.ProtectReg(d144.Reg)
								ctx.ProtectReg(d144.Reg2)
							}
							d149 = d144
							if d149.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.SyncDesc(&d149)
							if d149.Loc == LocStackPair {
								ctx.EmitCopyStackWords(d149, int32(bbs[10].PhiBase)+int32(0), 2)
							} else if d149.Loc == LocInputPair {
								ctx.EnsureDesc(&d149)
								ctx.EmitStoreScmerToStack(d149, int32(bbs[10].PhiBase)+int32(0))
							} else if d149.Loc == LocRegPair || d149.Loc == LocImm {
								ctx.EmitStoreScmerToStack(d149, int32(bbs[10].PhiBase)+int32(0))
							} else {
								ctx.EnsureDesc(&d149)
								ctx.EmitStoreToStack(d149, int32(bbs[10].PhiBase)+int32(0))
								ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[10].PhiBase)+int32(0))+8)
							}
							if d144.Loc == LocReg {
								ctx.UnprotectReg(d144.Reg)
							} else if d144.Loc == LocRegPair {
								ctx.UnprotectReg(d144.Reg)
								ctx.UnprotectReg(d144.Reg2)
							}
						}
						ps150 := PhiState{General: ps.General}
						ps150.OverlayValues = make([]JITValueDesc, 150)
						ps150.OverlayValues[1] = d1
						ps150.OverlayValues[2] = d2
						ps150.OverlayValues[3] = d3
						ps150.OverlayValues[4] = d4
						ps150.OverlayValues[5] = d5
						ps150.OverlayValues[6] = d6
						ps150.OverlayValues[7] = d7
						ps150.OverlayValues[9] = d9
						ps150.OverlayValues[22] = d22
						ps150.OverlayValues[33] = d33
						ps150.OverlayValues[34] = d34
						ps150.OverlayValues[35] = d35
						ps150.OverlayValues[38] = d38
						ps150.OverlayValues[55] = d55
						ps150.OverlayValues[71] = d71
						ps150.OverlayValues[73] = d73
						ps150.OverlayValues[74] = d74
						ps150.OverlayValues[75] = d75
						ps150.OverlayValues[77] = d77
						ps150.OverlayValues[78] = d78
						ps150.OverlayValues[80] = d80
						ps150.OverlayValues[81] = d81
						ps150.OverlayValues[82] = d82
						ps150.OverlayValues[83] = d83
						ps150.OverlayValues[86] = d86
						ps150.OverlayValues[115] = d115
						ps150.OverlayValues[143] = d143
						ps150.OverlayValues[144] = d144
						ps150.OverlayValues[145] = d145
						ps150.OverlayValues[146] = d146
						ps150.OverlayValues[147] = d147
						ps150.OverlayValues[149] = d149
						ps150.PhiValues = make([]JITValueDesc, 1)
						d151 = d144
						ps150.PhiValues[0] = d151
						return bbs[10].RenderPS(ps150)
					}
					if !ps.General {
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					ctx.EmitJump(d147.Condition, lbl18)
					ctx.EmitJmp(lbl19)
					snap152 := d1
					snap153 := d2
					snap154 := d3
					snap155 := d4
					snap156 := d5
					snap157 := d6
					snap158 := d7
					snap159 := d9
					snap160 := d22
					snap161 := d33
					snap162 := d34
					snap163 := d35
					snap164 := d38
					snap165 := d55
					snap166 := d71
					snap167 := d73
					snap168 := d74
					snap169 := d75
					snap170 := d77
					snap171 := d78
					snap172 := d80
					snap173 := d81
					snap174 := d82
					snap175 := d83
					snap176 := d86
					snap177 := d115
					snap178 := d143
					snap179 := d144
					snap180 := d145
					snap181 := d146
					snap182 := d147
					snap183 := d149
					snap184 := d151
					alloc185 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl18)
					ctx.EmitJmp(lbl10)
					ctx.RestoreAllocState(alloc185)
					d1 = snap152
					d2 = snap153
					d3 = snap154
					d4 = snap155
					d5 = snap156
					d6 = snap157
					d7 = snap158
					d9 = snap159
					d22 = snap160
					d33 = snap161
					d34 = snap162
					d35 = snap163
					d38 = snap164
					d55 = snap165
					d71 = snap166
					d73 = snap167
					d74 = snap168
					d75 = snap169
					d77 = snap170
					d78 = snap171
					d80 = snap172
					d81 = snap173
					d82 = snap174
					d83 = snap175
					d86 = snap176
					d115 = snap177
					d143 = snap178
					d144 = snap179
					d145 = snap180
					d146 = snap181
					d147 = snap182
					d149 = snap183
					d151 = snap184
					ctx.MarkLabel(lbl19)
					ctx.SyncDesc(&d144)
					if d144.Loc == LocReg {
						ctx.ProtectReg(d144.Reg)
					} else if d144.Loc == LocRegPair {
						ctx.ProtectReg(d144.Reg)
						ctx.ProtectReg(d144.Reg2)
					}
					d186 = d144
					if d186.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d186)
					if d186.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d186, int32(bbs[10].PhiBase)+int32(0), 2)
					} else if d186.Loc == LocInputPair {
						ctx.EnsureDesc(&d186)
						ctx.EmitStoreScmerToStack(d186, int32(bbs[10].PhiBase)+int32(0))
					} else if d186.Loc == LocRegPair || d186.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d186, int32(bbs[10].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d186)
						ctx.EmitStoreToStack(d186, int32(bbs[10].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[10].PhiBase)+int32(0))+8)
					}
					if d144.Loc == LocReg {
						ctx.UnprotectReg(d144.Reg)
					} else if d144.Loc == LocRegPair {
						ctx.UnprotectReg(d144.Reg)
						ctx.UnprotectReg(d144.Reg2)
					}
					ctx.EmitJmp(lbl11)
					ctx.RestoreAllocState(alloc185)
					d1 = snap152
					d2 = snap153
					d3 = snap154
					d4 = snap155
					d5 = snap156
					d6 = snap157
					d7 = snap158
					d9 = snap159
					d22 = snap160
					d33 = snap161
					d34 = snap162
					d35 = snap163
					d38 = snap164
					d55 = snap165
					d71 = snap166
					d73 = snap167
					d74 = snap168
					d75 = snap169
					d77 = snap170
					d78 = snap171
					d80 = snap172
					d81 = snap173
					d82 = snap174
					d83 = snap175
					d86 = snap176
					d115 = snap177
					d143 = snap178
					d144 = snap179
					d145 = snap180
					d146 = snap181
					d147 = snap182
					d149 = snap183
					d151 = snap184
					ps187 := PhiState{General: true}
					ps187.OverlayValues = make([]JITValueDesc, 187)
					ps187.OverlayValues[1] = d1
					ps187.OverlayValues[2] = d2
					ps187.OverlayValues[3] = d3
					ps187.OverlayValues[4] = d4
					ps187.OverlayValues[5] = d5
					ps187.OverlayValues[6] = d6
					ps187.OverlayValues[7] = d7
					ps187.OverlayValues[9] = d9
					ps187.OverlayValues[22] = d22
					ps187.OverlayValues[33] = d33
					ps187.OverlayValues[34] = d34
					ps187.OverlayValues[35] = d35
					ps187.OverlayValues[38] = d38
					ps187.OverlayValues[55] = d55
					ps187.OverlayValues[71] = d71
					ps187.OverlayValues[73] = d73
					ps187.OverlayValues[74] = d74
					ps187.OverlayValues[75] = d75
					ps187.OverlayValues[77] = d77
					ps187.OverlayValues[78] = d78
					ps187.OverlayValues[80] = d80
					ps187.OverlayValues[81] = d81
					ps187.OverlayValues[82] = d82
					ps187.OverlayValues[83] = d83
					ps187.OverlayValues[86] = d86
					ps187.OverlayValues[115] = d115
					ps187.OverlayValues[143] = d143
					ps187.OverlayValues[144] = d144
					ps187.OverlayValues[145] = d145
					ps187.OverlayValues[146] = d146
					ps187.OverlayValues[147] = d147
					ps187.OverlayValues[149] = d149
					ps187.OverlayValues[151] = d151
					ps187.OverlayValues[186] = d186
					ps188 := PhiState{General: true}
					ps188.OverlayValues = make([]JITValueDesc, 187)
					ps188.OverlayValues[1] = d1
					ps188.OverlayValues[2] = d2
					ps188.OverlayValues[3] = d3
					ps188.OverlayValues[4] = d4
					ps188.OverlayValues[5] = d5
					ps188.OverlayValues[6] = d6
					ps188.OverlayValues[7] = d7
					ps188.OverlayValues[9] = d9
					ps188.OverlayValues[22] = d22
					ps188.OverlayValues[33] = d33
					ps188.OverlayValues[34] = d34
					ps188.OverlayValues[35] = d35
					ps188.OverlayValues[38] = d38
					ps188.OverlayValues[55] = d55
					ps188.OverlayValues[71] = d71
					ps188.OverlayValues[73] = d73
					ps188.OverlayValues[74] = d74
					ps188.OverlayValues[75] = d75
					ps188.OverlayValues[77] = d77
					ps188.OverlayValues[78] = d78
					ps188.OverlayValues[80] = d80
					ps188.OverlayValues[81] = d81
					ps188.OverlayValues[82] = d82
					ps188.OverlayValues[83] = d83
					ps188.OverlayValues[86] = d86
					ps188.OverlayValues[115] = d115
					ps188.OverlayValues[143] = d143
					ps188.OverlayValues[144] = d144
					ps188.OverlayValues[145] = d145
					ps188.OverlayValues[146] = d146
					ps188.OverlayValues[147] = d147
					ps188.OverlayValues[149] = d149
					ps188.OverlayValues[151] = d151
					ps188.OverlayValues[186] = d186
					ps188.PhiValues = make([]JITValueDesc, 1)
					d189 = d144
					ps188.PhiValues[0] = d189
					snap190 := d1
					snap191 := d2
					snap192 := d3
					snap193 := d4
					snap194 := d5
					snap195 := d6
					snap196 := d7
					snap197 := d9
					snap198 := d22
					snap199 := d33
					snap200 := d34
					snap201 := d35
					snap202 := d38
					snap203 := d55
					snap204 := d71
					snap205 := d73
					snap206 := d74
					snap207 := d75
					snap208 := d77
					snap209 := d78
					snap210 := d80
					snap211 := d81
					snap212 := d82
					snap213 := d83
					snap214 := d86
					snap215 := d115
					snap216 := d143
					snap217 := d144
					snap218 := d145
					snap219 := d146
					snap220 := d147
					snap221 := d149
					snap222 := d151
					snap223 := d186
					snap224 := d189
					alloc225 := ctx.SnapshotAllocState()
					if !bbs[10].Rendered {
						bbs[10].RenderPS(ps188)
					}
					ctx.RestoreAllocState(alloc225)
					d1 = snap190
					d2 = snap191
					d3 = snap192
					d4 = snap193
					d5 = snap194
					d6 = snap195
					d7 = snap196
					d9 = snap197
					d22 = snap198
					d33 = snap199
					d34 = snap200
					d35 = snap201
					d38 = snap202
					d55 = snap203
					d71 = snap204
					d73 = snap205
					d74 = snap206
					d75 = snap207
					d77 = snap208
					d78 = snap209
					d80 = snap210
					d81 = snap211
					d82 = snap212
					d83 = snap213
					d86 = snap214
					d115 = snap215
					d143 = snap216
					d144 = snap217
					d145 = snap218
					d146 = snap219
					d147 = snap220
					d149 = snap221
					d151 = snap222
					d186 = snap223
					d189 = snap224
					if !bbs[9].Rendered {
						return bbs[9].RenderPS(ps187)
					}
					return result
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
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
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != LocNone {
						d83 = ps.OverlayValues[83]
					}
					if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != LocNone {
						d86 = ps.OverlayValues[86]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
						d186 = ps.OverlayValues[186]
					}
					if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
						d189 = ps.OverlayValues[189]
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
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
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != LocNone {
						d83 = ps.OverlayValues[83]
					}
					if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != LocNone {
						d86 = ps.OverlayValues[86]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
						d186 = ps.OverlayValues[186]
					}
					if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
						d189 = ps.OverlayValues[189]
					}
					ctx.ReclaimUntrackedRegs()
					d226 = d1
					ctx.EnsureDesc(&d226)
					if d226.Loc != LocImm && d226.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d226.Loc == LocImm {
						if d226.Imm.Bool() {
							if ps.General {
							}
							ps227 := PhiState{General: ps.General}
							ps227.OverlayValues = make([]JITValueDesc, 227)
							ps227.OverlayValues[1] = d1
							ps227.OverlayValues[2] = d2
							ps227.OverlayValues[3] = d3
							ps227.OverlayValues[4] = d4
							ps227.OverlayValues[5] = d5
							ps227.OverlayValues[6] = d6
							ps227.OverlayValues[7] = d7
							ps227.OverlayValues[9] = d9
							ps227.OverlayValues[22] = d22
							ps227.OverlayValues[33] = d33
							ps227.OverlayValues[34] = d34
							ps227.OverlayValues[35] = d35
							ps227.OverlayValues[38] = d38
							ps227.OverlayValues[55] = d55
							ps227.OverlayValues[71] = d71
							ps227.OverlayValues[73] = d73
							ps227.OverlayValues[74] = d74
							ps227.OverlayValues[75] = d75
							ps227.OverlayValues[77] = d77
							ps227.OverlayValues[78] = d78
							ps227.OverlayValues[80] = d80
							ps227.OverlayValues[81] = d81
							ps227.OverlayValues[82] = d82
							ps227.OverlayValues[83] = d83
							ps227.OverlayValues[86] = d86
							ps227.OverlayValues[115] = d115
							ps227.OverlayValues[143] = d143
							ps227.OverlayValues[144] = d144
							ps227.OverlayValues[145] = d145
							ps227.OverlayValues[146] = d146
							ps227.OverlayValues[147] = d147
							ps227.OverlayValues[149] = d149
							ps227.OverlayValues[151] = d151
							ps227.OverlayValues[186] = d186
							ps227.OverlayValues[189] = d189
							ps227.OverlayValues[226] = d226
							return bbs[7].RenderPS(ps227)
						}
						if ps.General {
						}
						ps228 := PhiState{General: ps.General}
						ps228.OverlayValues = make([]JITValueDesc, 227)
						ps228.OverlayValues[1] = d1
						ps228.OverlayValues[2] = d2
						ps228.OverlayValues[3] = d3
						ps228.OverlayValues[4] = d4
						ps228.OverlayValues[5] = d5
						ps228.OverlayValues[6] = d6
						ps228.OverlayValues[7] = d7
						ps228.OverlayValues[9] = d9
						ps228.OverlayValues[22] = d22
						ps228.OverlayValues[33] = d33
						ps228.OverlayValues[34] = d34
						ps228.OverlayValues[35] = d35
						ps228.OverlayValues[38] = d38
						ps228.OverlayValues[55] = d55
						ps228.OverlayValues[71] = d71
						ps228.OverlayValues[73] = d73
						ps228.OverlayValues[74] = d74
						ps228.OverlayValues[75] = d75
						ps228.OverlayValues[77] = d77
						ps228.OverlayValues[78] = d78
						ps228.OverlayValues[80] = d80
						ps228.OverlayValues[81] = d81
						ps228.OverlayValues[82] = d82
						ps228.OverlayValues[83] = d83
						ps228.OverlayValues[86] = d86
						ps228.OverlayValues[115] = d115
						ps228.OverlayValues[143] = d143
						ps228.OverlayValues[144] = d144
						ps228.OverlayValues[145] = d145
						ps228.OverlayValues[146] = d146
						ps228.OverlayValues[147] = d147
						ps228.OverlayValues[149] = d149
						ps228.OverlayValues[151] = d151
						ps228.OverlayValues[186] = d186
						ps228.OverlayValues[189] = d189
						ps228.OverlayValues[226] = d226
						return bbs[6].RenderPS(ps228)
					}
					if !ps.General {
						ps.General = true
						return bbs[8].RenderPS(ps)
					}
					lbl20 := ctx.ReserveLabel()
					lbl21 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d226.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl20)
					ctx.EmitJmp(lbl21)
					snap229 := d1
					snap230 := d2
					snap231 := d3
					snap232 := d4
					snap233 := d5
					snap234 := d6
					snap235 := d7
					snap236 := d9
					snap237 := d22
					snap238 := d33
					snap239 := d34
					snap240 := d35
					snap241 := d38
					snap242 := d55
					snap243 := d71
					snap244 := d73
					snap245 := d74
					snap246 := d75
					snap247 := d77
					snap248 := d78
					snap249 := d80
					snap250 := d81
					snap251 := d82
					snap252 := d83
					snap253 := d86
					snap254 := d115
					snap255 := d143
					snap256 := d144
					snap257 := d145
					snap258 := d146
					snap259 := d147
					snap260 := d149
					snap261 := d151
					snap262 := d186
					snap263 := d189
					snap264 := d226
					alloc265 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl8)
					ctx.RestoreAllocState(alloc265)
					d1 = snap229
					d2 = snap230
					d3 = snap231
					d4 = snap232
					d5 = snap233
					d6 = snap234
					d7 = snap235
					d9 = snap236
					d22 = snap237
					d33 = snap238
					d34 = snap239
					d35 = snap240
					d38 = snap241
					d55 = snap242
					d71 = snap243
					d73 = snap244
					d74 = snap245
					d75 = snap246
					d77 = snap247
					d78 = snap248
					d80 = snap249
					d81 = snap250
					d82 = snap251
					d83 = snap252
					d86 = snap253
					d115 = snap254
					d143 = snap255
					d144 = snap256
					d145 = snap257
					d146 = snap258
					d147 = snap259
					d149 = snap260
					d151 = snap261
					d186 = snap262
					d189 = snap263
					d226 = snap264
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl7)
					ctx.RestoreAllocState(alloc265)
					d1 = snap229
					d2 = snap230
					d3 = snap231
					d4 = snap232
					d5 = snap233
					d6 = snap234
					d7 = snap235
					d9 = snap236
					d22 = snap237
					d33 = snap238
					d34 = snap239
					d35 = snap240
					d38 = snap241
					d55 = snap242
					d71 = snap243
					d73 = snap244
					d74 = snap245
					d75 = snap246
					d77 = snap247
					d78 = snap248
					d80 = snap249
					d81 = snap250
					d82 = snap251
					d83 = snap252
					d86 = snap253
					d115 = snap254
					d143 = snap255
					d144 = snap256
					d145 = snap257
					d146 = snap258
					d147 = snap259
					d149 = snap260
					d151 = snap261
					d186 = snap262
					d189 = snap263
					d226 = snap264
					ps266 := PhiState{General: true}
					ps266.OverlayValues = make([]JITValueDesc, 227)
					ps266.OverlayValues[1] = d1
					ps266.OverlayValues[2] = d2
					ps266.OverlayValues[3] = d3
					ps266.OverlayValues[4] = d4
					ps266.OverlayValues[5] = d5
					ps266.OverlayValues[6] = d6
					ps266.OverlayValues[7] = d7
					ps266.OverlayValues[9] = d9
					ps266.OverlayValues[22] = d22
					ps266.OverlayValues[33] = d33
					ps266.OverlayValues[34] = d34
					ps266.OverlayValues[35] = d35
					ps266.OverlayValues[38] = d38
					ps266.OverlayValues[55] = d55
					ps266.OverlayValues[71] = d71
					ps266.OverlayValues[73] = d73
					ps266.OverlayValues[74] = d74
					ps266.OverlayValues[75] = d75
					ps266.OverlayValues[77] = d77
					ps266.OverlayValues[78] = d78
					ps266.OverlayValues[80] = d80
					ps266.OverlayValues[81] = d81
					ps266.OverlayValues[82] = d82
					ps266.OverlayValues[83] = d83
					ps266.OverlayValues[86] = d86
					ps266.OverlayValues[115] = d115
					ps266.OverlayValues[143] = d143
					ps266.OverlayValues[144] = d144
					ps266.OverlayValues[145] = d145
					ps266.OverlayValues[146] = d146
					ps266.OverlayValues[147] = d147
					ps266.OverlayValues[149] = d149
					ps266.OverlayValues[151] = d151
					ps266.OverlayValues[186] = d186
					ps266.OverlayValues[189] = d189
					ps266.OverlayValues[226] = d226
					ps267 := PhiState{General: true}
					ps267.OverlayValues = make([]JITValueDesc, 227)
					ps267.OverlayValues[1] = d1
					ps267.OverlayValues[2] = d2
					ps267.OverlayValues[3] = d3
					ps267.OverlayValues[4] = d4
					ps267.OverlayValues[5] = d5
					ps267.OverlayValues[6] = d6
					ps267.OverlayValues[7] = d7
					ps267.OverlayValues[9] = d9
					ps267.OverlayValues[22] = d22
					ps267.OverlayValues[33] = d33
					ps267.OverlayValues[34] = d34
					ps267.OverlayValues[35] = d35
					ps267.OverlayValues[38] = d38
					ps267.OverlayValues[55] = d55
					ps267.OverlayValues[71] = d71
					ps267.OverlayValues[73] = d73
					ps267.OverlayValues[74] = d74
					ps267.OverlayValues[75] = d75
					ps267.OverlayValues[77] = d77
					ps267.OverlayValues[78] = d78
					ps267.OverlayValues[80] = d80
					ps267.OverlayValues[81] = d81
					ps267.OverlayValues[82] = d82
					ps267.OverlayValues[83] = d83
					ps267.OverlayValues[86] = d86
					ps267.OverlayValues[115] = d115
					ps267.OverlayValues[143] = d143
					ps267.OverlayValues[144] = d144
					ps267.OverlayValues[145] = d145
					ps267.OverlayValues[146] = d146
					ps267.OverlayValues[147] = d147
					ps267.OverlayValues[149] = d149
					ps267.OverlayValues[151] = d151
					ps267.OverlayValues[186] = d186
					ps267.OverlayValues[189] = d189
					ps267.OverlayValues[226] = d226
					snap268 := d1
					snap269 := d2
					snap270 := d3
					snap271 := d4
					snap272 := d5
					snap273 := d6
					snap274 := d7
					snap275 := d9
					snap276 := d22
					snap277 := d33
					snap278 := d34
					snap279 := d35
					snap280 := d38
					snap281 := d55
					snap282 := d71
					snap283 := d73
					snap284 := d74
					snap285 := d75
					snap286 := d77
					snap287 := d78
					snap288 := d80
					snap289 := d81
					snap290 := d82
					snap291 := d83
					snap292 := d86
					snap293 := d115
					snap294 := d143
					snap295 := d144
					snap296 := d145
					snap297 := d146
					snap298 := d147
					snap299 := d149
					snap300 := d151
					snap301 := d186
					snap302 := d189
					snap303 := d226
					alloc304 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps267)
					}
					ctx.RestoreAllocState(alloc304)
					d1 = snap268
					d2 = snap269
					d3 = snap270
					d4 = snap271
					d5 = snap272
					d6 = snap273
					d7 = snap274
					d9 = snap275
					d22 = snap276
					d33 = snap277
					d34 = snap278
					d35 = snap279
					d38 = snap280
					d55 = snap281
					d71 = snap282
					d73 = snap283
					d74 = snap284
					d75 = snap285
					d77 = snap286
					d78 = snap287
					d80 = snap288
					d81 = snap289
					d82 = snap290
					d83 = snap291
					d86 = snap292
					d115 = snap293
					d143 = snap294
					d144 = snap295
					d145 = snap296
					d146 = snap297
					d147 = snap298
					d149 = snap299
					d151 = snap300
					d186 = snap301
					d189 = snap302
					d226 = snap303
					if !bbs[7].Rendered {
						return bbs[7].RenderPS(ps266)
					}
					return result
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
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
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != LocNone {
						d83 = ps.OverlayValues[83]
					}
					if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != LocNone {
						d86 = ps.OverlayValues[86]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
						d186 = ps.OverlayValues[186]
					}
					if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
						d189 = ps.OverlayValues[189]
					}
					if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != LocNone {
						d226 = ps.OverlayValues[226]
					}
					ctx.ReclaimUntrackedRegs()
					d305 = args[1]
					d305.ID = 0
					d307 = d305
					ctx.SyncDesc(&d307)
					if d307.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d307.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d307.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d307 = tmpScalar
					}
					d307 = JITPrepareScmerGoArg(ctx, d307)
					if d307.Loc != LocRegPair && d307.Loc != LocStackPair && d307.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d306 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d307}, 2)
					ctx.StabilizeDescForControlFlow(&d306)
					ctx.FreeDesc(&d305)
					if ps.General {
						ctx.SyncDesc(&d306)
						if d306.Loc == LocReg {
							ctx.ProtectReg(d306.Reg)
						} else if d306.Loc == LocRegPair {
							ctx.ProtectReg(d306.Reg)
							ctx.ProtectReg(d306.Reg2)
						}
						d308 = d306
						if d308.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d308)
						if d308.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d308, int32(bbs[10].PhiBase)+int32(0), 2)
						} else if d308.Loc == LocInputPair {
							ctx.EnsureDesc(&d308)
							ctx.EmitStoreScmerToStack(d308, int32(bbs[10].PhiBase)+int32(0))
						} else if d308.Loc == LocRegPair || d308.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d308, int32(bbs[10].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d308)
							ctx.EmitStoreToStack(d308, int32(bbs[10].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[10].PhiBase)+int32(0))+8)
						}
						if d306.Loc == LocReg {
							ctx.UnprotectReg(d306.Reg)
						} else if d306.Loc == LocRegPair {
							ctx.UnprotectReg(d306.Reg)
							ctx.UnprotectReg(d306.Reg2)
						}
					}
					ps309 := PhiState{General: ps.General}
					ps309.OverlayValues = make([]JITValueDesc, 309)
					ps309.OverlayValues[1] = d1
					ps309.OverlayValues[2] = d2
					ps309.OverlayValues[3] = d3
					ps309.OverlayValues[4] = d4
					ps309.OverlayValues[5] = d5
					ps309.OverlayValues[6] = d6
					ps309.OverlayValues[7] = d7
					ps309.OverlayValues[9] = d9
					ps309.OverlayValues[22] = d22
					ps309.OverlayValues[33] = d33
					ps309.OverlayValues[34] = d34
					ps309.OverlayValues[35] = d35
					ps309.OverlayValues[38] = d38
					ps309.OverlayValues[55] = d55
					ps309.OverlayValues[71] = d71
					ps309.OverlayValues[73] = d73
					ps309.OverlayValues[74] = d74
					ps309.OverlayValues[75] = d75
					ps309.OverlayValues[77] = d77
					ps309.OverlayValues[78] = d78
					ps309.OverlayValues[80] = d80
					ps309.OverlayValues[81] = d81
					ps309.OverlayValues[82] = d82
					ps309.OverlayValues[83] = d83
					ps309.OverlayValues[86] = d86
					ps309.OverlayValues[115] = d115
					ps309.OverlayValues[143] = d143
					ps309.OverlayValues[144] = d144
					ps309.OverlayValues[145] = d145
					ps309.OverlayValues[146] = d146
					ps309.OverlayValues[147] = d147
					ps309.OverlayValues[149] = d149
					ps309.OverlayValues[151] = d151
					ps309.OverlayValues[186] = d186
					ps309.OverlayValues[189] = d189
					ps309.OverlayValues[226] = d226
					ps309.OverlayValues[305] = d305
					ps309.OverlayValues[306] = d306
					ps309.OverlayValues[307] = d307
					ps309.OverlayValues[308] = d308
					ps309.PhiValues = make([]JITValueDesc, 1)
					d310 = d306
					ps309.PhiValues[0] = d310
					if ps309.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps309)
					return result
				}
				bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d311 := ps.PhiValues[0]
							ctx.EnsureDesc(&d311)
							ctx.EmitStoreScmerToStack(d311, int32(bbs[10].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
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
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != LocNone {
						d83 = ps.OverlayValues[83]
					}
					if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != LocNone {
						d86 = ps.OverlayValues[86]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
						d186 = ps.OverlayValues[186]
					}
					if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
						d189 = ps.OverlayValues[189]
					}
					if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != LocNone {
						d226 = ps.OverlayValues[226]
					}
					if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != LocNone {
						d305 = ps.OverlayValues[305]
					}
					if len(ps.OverlayValues) > 306 && ps.OverlayValues[306].Loc != LocNone {
						d306 = ps.OverlayValues[306]
					}
					if len(ps.OverlayValues) > 307 && ps.OverlayValues[307].Loc != LocNone {
						d307 = ps.OverlayValues[307]
					}
					if len(ps.OverlayValues) > 308 && ps.OverlayValues[308].Loc != LocNone {
						d308 = ps.OverlayValues[308]
					}
					if len(ps.OverlayValues) > 310 && ps.OverlayValues[310].Loc != LocNone {
						d310 = ps.OverlayValues[310]
					}
					if len(ps.OverlayValues) > 311 && ps.OverlayValues[311].Loc != LocNone {
						d311 = ps.OverlayValues[311]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d3 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					d312 = ctx.EmitGoCallScalar(GoFuncAddr(func() *[1]any { return new([1]any) }), nil, 1)
					d313 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.EnsureDesc(&d3)
					d314 = ctx.EmitGoCallScalar(GoFuncAddr(func(value string) any { return value }), []JITValueDesc{d3}, 2)
					ctx.FreeDesc(&d3)
					ctx.EnsureDesc(&d314)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[1]any, index int, value any) { dst[index] = value }), []JITValueDesc{d312, d313, d314})
					sliceResults315 := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *[1]any) []any { return value[0:1:1] }), []JITValueDesc{d312}, []uint8{3}, []uint8{1})
					d316 = sliceResults315[0]
					d317 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("warning: JIT fallback: %s\n")}
					ctx.EnsureDesc(&d317)
					if d317.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d317.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d317.Imm)
						ptrWord, _ := d317.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d317.Imm.String())))
						d317 = tmpPair
					} else if d317.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d317.Type, Reg: ctx.AllocRegExcept(d317.Reg), Reg2: ctx.AllocRegExcept(d317.Reg)}
						switch d317.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d317)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d317)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d317)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d317)
						d317 = tmpPair
					}
					if d317.Loc != LocRegPair && d317.Loc != LocStackPair && d317.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (fmt.Printf arg0)")
					}
					ctx.EnsureDesc(&d316)
					ctx.EnsureDesc(&d316)
					ctx.EnsureDesc(&d316)
					if d316.Loc != LocRegTriple && d316.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (fmt.Printf arg1)")
					}
					ctx.SyncDesc(&d317)
					ctx.SyncDesc(&d316)
					callResults318 := JITEmitGoCallResults(ctx, GoFuncAddr(fmt.Printf), []JITValueDesc{d317, d316}, []uint8{1, 2}, []uint8{0, 3})
					ctx.FreeDesc(&d317)
					d319 = callResults318[0]
					_ = d319
					d320 = callResults318[1]
					_ = d320
					if ps.General {
					}
					ps321 := PhiState{General: ps.General}
					ps321.OverlayValues = make([]JITValueDesc, 321)
					ps321.OverlayValues[1] = d1
					ps321.OverlayValues[2] = d2
					ps321.OverlayValues[3] = d3
					ps321.OverlayValues[4] = d4
					ps321.OverlayValues[5] = d5
					ps321.OverlayValues[6] = d6
					ps321.OverlayValues[7] = d7
					ps321.OverlayValues[9] = d9
					ps321.OverlayValues[22] = d22
					ps321.OverlayValues[33] = d33
					ps321.OverlayValues[34] = d34
					ps321.OverlayValues[35] = d35
					ps321.OverlayValues[38] = d38
					ps321.OverlayValues[55] = d55
					ps321.OverlayValues[71] = d71
					ps321.OverlayValues[73] = d73
					ps321.OverlayValues[74] = d74
					ps321.OverlayValues[75] = d75
					ps321.OverlayValues[77] = d77
					ps321.OverlayValues[78] = d78
					ps321.OverlayValues[80] = d80
					ps321.OverlayValues[81] = d81
					ps321.OverlayValues[82] = d82
					ps321.OverlayValues[83] = d83
					ps321.OverlayValues[86] = d86
					ps321.OverlayValues[115] = d115
					ps321.OverlayValues[143] = d143
					ps321.OverlayValues[144] = d144
					ps321.OverlayValues[145] = d145
					ps321.OverlayValues[146] = d146
					ps321.OverlayValues[147] = d147
					ps321.OverlayValues[149] = d149
					ps321.OverlayValues[151] = d151
					ps321.OverlayValues[186] = d186
					ps321.OverlayValues[189] = d189
					ps321.OverlayValues[226] = d226
					ps321.OverlayValues[305] = d305
					ps321.OverlayValues[306] = d306
					ps321.OverlayValues[307] = d307
					ps321.OverlayValues[308] = d308
					ps321.OverlayValues[310] = d310
					ps321.OverlayValues[311] = d311
					ps321.OverlayValues[312] = d312
					ps321.OverlayValues[313] = d313
					ps321.OverlayValues[314] = d314
					ps321.OverlayValues[316] = d316
					ps321.OverlayValues[317] = d317
					ps321.OverlayValues[319] = d319
					ps321.OverlayValues[320] = d320
					if ps321.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps321)
					return result
				}
				ps322 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps322)
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
				declaration := declarations["jit-enabled?"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
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
		if proc != nil && proc.Compiled != nil && proc.Compiled.CodePtr != nil {
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
		// Try increasing buffer sizes for overflow retry
		for _, codeCap := range [...]int{16 * 1024, 64 * 1024, 256 * 1024, 1024 * 1024, 4 * 1024 * 1024, 16 * 1024 * 1024} {
			ptr, arena, reservation := globalJITPool.Alloc(codeCap)
			buf := &execBuf{ptr: ptr, n: codeCap, arena: arena, reservation: reservation}
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
					arena.complete(reservation, buf.stackMaps)
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
				arena.completeDeferred(reservation, buf.stackMaps, onPublish)
				return Scmer{ptr: (*byte)(unsafe.Pointer(targetProc)), aux: makeAux(tagProc, 0)}
			}
			if waitForPublication {
				arena.complete(reservation, buf.stackMaps)
			} else {
				arena.completeDeferred(reservation, buf.stackMaps, nil)
			}
			globalJITPool.Free(arena)
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
	fmt.Printf("jitdump: import=%s selected=%t expressions=%d dynamic-calls=%d direct-procs=%d native-calls=%d inlined-calls=%d\n",
		name, selected, entry.Coverage.Expressions, entry.Coverage.DynamicCalls,
		entry.Coverage.DirectProcs, entry.Coverage.NativeCalls, entry.Coverage.InlinedCalls)
}
