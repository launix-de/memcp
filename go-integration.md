# Go-Native JIT Integration — Design Sketch

## Problem

Go compiles AOT (ahead-of-time). There is no standard mechanism to generate
and execute machine code at runtime based on type information that is only
available at execution time. Projects like MemCP need runtime code generation
because the "program" (a SQL query compiled to Scheme, then to a scan function)
is only known at query time, and the data types of the columns it touches are
only known when the storage engine is consulted.

The current solution (`tools/jitgen`) is a domain-specific meta-compiler:
it reads Go source via `golang.org/x/tools/go/ssa`, and generates Go code
that emits AMD64 machine code at runtime. This works but is fragile, hard
to maintain (~6400 lines), and tightly coupled to MemCP internals.

## Goal

A **general-purpose Go library** that lets any Go program JIT-compile
type-specialized functions at runtime, using Go's own SSA as the IR.

## Proposed Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Application Code                                           │
│                                                             │
│  // Mark a function for JIT specialization                  │
│  var Add = jit.Specialize(func(a, b int64) int64 {          │
│      return a + b                                           │
│  })                                                         │
│                                                             │
│  // Call with concrete types at runtime                     │
│  result := Add.Call(x, y)                                   │
└──────────────┬──────────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────────┐
│  jit.Specialize(fn)                                         │
│                                                             │
│  1. At init time: extract SSA from fn via reflection +      │
│     embedded metadata (see below)                           │
│  2. On first call with concrete types: lower SSA → MachIR   │
│  3. Assemble MachIR → executable page (mmap RWX)            │
│  4. Cache specialized variant keyed by type signature       │
│  5. Subsequent calls dispatch to cached native code         │
└──────────────┬──────────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────────┐
│  Three-Layer IR                                             │
│                                                             │
│  Layer 1: Go SSA (golang.org/x/tools/go/ssa)               │
│    - Platform-independent                                   │
│    - Type-aware                                             │
│    - Already handles phi nodes, control flow, closures      │
│                                                             │
│  Layer 2: MachIR (new, platform-independent low-level IR)   │
│    - Virtual registers (infinite supply)                    │
│    - Explicit loads/stores, comparisons, branches           │
│    - Type-specialized: int64 ADD, float64 MUL, etc.         │
│    - No Go runtime calls (GC, scheduler) in hot paths       │
│                                                             │
│  Layer 3: Native Code (platform-specific backend)           │
│    - Register allocation (linear scan or graph coloring)    │
│    - Instruction selection + scheduling                     │
│    - Code emission into mmap'd pages                        │
│    - Initially: AMD64 only. ARM64 as second target.         │
└─────────────────────────────────────────────────────────────┘
```

## Key Design Decisions

### 1. SSA Embedding at Build Time

The main obstacle: Go binaries do not carry SSA or source. We need a
`go generate` step (or Go toolchain plugin) that serializes the SSA of
marked functions into the binary:

```go
//go:generate jitprep ./...
```

`jitprep` would:
- Load the package with `golang.org/x/tools/go/packages`
- Build SSA for functions annotated with `//jit:specialize`
- Serialize the SSA into a compact binary format
- Generate a `_jit_metadata.go` file with `//go:embed` directives

At runtime, `jit.Specialize()` deserializes the SSA and has full type
information for lowering.

**Alternative**: Skip SSA embedding entirely. Instead, use Go's
`reflect` package + `runtime.FuncForPC` to identify the function, and
ship the SSA lowering rules as a build-time-generated lookup table.
This avoids embedding IR but limits specialization to predefined patterns.

### 2. Type Specialization Model

The core value proposition: a function like `func(a Scmer) Scmer` has
different optimal machine code depending on whether `a` is `int64`,
`float64`, or `string`. The JIT should:

```go
type SpecKey struct {
    ArgTypes  []TypeTag   // concrete types of arguments
    ConstArgs []int       // indices of arguments known at compile time
    ConstVals []any       // their values
}
```

On first call with a new `SpecKey`, the JIT:
1. Clones the SSA
2. Propagates known types (constant folding, dead branch elimination)
3. Lowers to MachIR with concrete register widths
4. Allocates registers and emits native code
5. Caches under the key

### 3. Register Allocation — Lessons from jitgen

The current jitgen uses a **runtime register allocator** (`AllocReg` /
`FreeReg` / `ProtectReg`), which is flexible but has subtle correctness
bugs (eviction-alias problem). A better approach:

**Build-time register allocation** via linear scan over the MachIR:
- Compute liveness intervals for all virtual registers
- Assign physical registers with spill slots for overflow
- No runtime allocator needed — the generated code uses fixed registers

This eliminates the entire class of eviction-alias bugs because register
assignment is static and verified before code emission.

```go
type LiveInterval struct {
    VReg     int      // virtual register
    Start    int      // first use (instruction index)
    End      int      // last use
    PReg     int      // assigned physical register (-1 = spilled)
    SpillOff int      // stack offset if spilled
}

func LinearScan(intervals []LiveInterval, numPhysRegs int) {
    // Standard linear-scan register allocation
    // Sort by start point, assign greedily, spill longest-lived on overflow
}
```

### 4. Safety Model

JIT-generated code runs in the same address space as Go. Safety concerns:

- **No GC interaction**: JIT code must not allocate Go heap objects.
  All temporaries live in registers or a pre-allocated scratch buffer.
- **No goroutine preemption**: JIT functions are non-preemptible (like
  `//go:nosplit`). Keep them short — microseconds, not milliseconds.
- **Stack bounds**: Use a fixed-size scratch area allocated before the
  JIT call. No dynamic stack growth.
- **Code pages**: `mmap` with `PROT_READ|PROT_EXEC` (W^X). Write
  during compilation, then `mprotect` to remove write permission.

### 5. Interface for MemCP

For MemCP specifically, the integration would replace the current
three-layer architecture with:

```go
// In storage/storage-int.go:
func (s *StorageInt) JITEmit(ctx *jit.Context, idx jit.Value) jit.Value {
    // The jit package handles register allocation, spilling, code emission.
    // No hand-written AMD64 — just declare the computation:
    base := ctx.LoadField(s, "data")        // []int32 data pointer
    elem := ctx.IndexLoad(base, idx, 4)     // 4-byte element load
    wide := ctx.SignExtend(elem, 32, 64)    // int32 → int64
    return ctx.TagInt(wide)                 // wrap as Scmer int
}
```

The `jit.Context` methods build MachIR nodes. After the full expression
tree is built, `ctx.Compile()` runs register allocation and emits native
code in one pass.

## What This Replaces

| Current (jitgen)                          | Proposed (jit library)                    |
|-------------------------------------------|-------------------------------------------|
| 6400-line Go code generator               | ~2000-line MachIR + backend               |
| Generates Go code that generates asm      | Directly generates asm from MachIR        |
| Runtime register allocator (bug-prone)    | Build-time linear scan (static, verified) |
| AMD64 only, deeply embedded               | Backend-swappable (AMD64, ARM64)          |
| Requires `make jitgen` + patching sources | Library call, no code generation step     |
| Domain-specific to MemCP                  | General-purpose, reusable                 |

## Realistic Effort Estimate

This is a significant project. Rough breakdown:

1. **MachIR definition + SSA lowering**: Core IR types, lowering from
   Go SSA for arithmetic, comparisons, loads, stores, branches, phi nodes.

2. **AMD64 backend**: Instruction selection, register allocation (linear
   scan), code emission, relocation/fixup.

3. **Runtime harness**: mmap management, specialization cache, calling
   convention bridge (Go → JIT → Go).

4. **MemCP integration**: Replace jitgen-generated `JITEmit` methods
   with direct `jit.Context` API calls.

## Prior Art

- **gojit** (github.com/nelhage/gojit): Proof-of-concept Go JIT, very
  minimal, no SSA lowering.
- **Cranelift**: Rust JIT backend, excellent design but not Go-native.
  Could be called via CGo but adds complexity.
- **LLVM MCJIT**: Full optimizing JIT, heavy dependency. Overkill for
  the simple arithmetic/comparison patterns MemCP needs.
- **DynASM**: LuaJIT's assembler DSL. Inspiring for the emit API but
  not Go-integrated.

## Conclusion

The general approach — a Go library that lowers Go SSA to a lightweight
MachIR, then emits native code with proper register allocation — would
solve the maintenance and correctness problems of the current jitgen
approach while being reusable for any Go project that needs runtime code
specialization. The key insight is separating register allocation
(build-time, static, verifiable) from code emission (runtime, fast),
eliminating the eviction-alias bug class entirely.
