//go:build goexperiment.jit && amd64

/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

package scm

import (
	"bytes"
	"math"
	"runtime"
	"testing"
	"unsafe"
)

func TestJITTypedInliningKeepsLargeNativeBoundaries(t *testing.T) {
	for _, cost := range []uint16{18, 38, 48, 49, 57, 256} {
		ctx := &JITContext{}
		decl := &Declaration{Type: &TypeDescriptor{JITInlineCost: cost}}
		args := []JITValueDesc{{Type: tagInt, Loc: LocStack}, {Type: tagInt, Loc: LocStack}}
		if got := jitGeneratedEmitterInline(ctx, decl, args); got != (cost <= 48) {
			t.Errorf("typed inline cost %d = %v", cost, got)
		}
	}
}

// Identical call shapes must not change admission when unrelated earlier
// code has claimed registers. The allocator still handles actual emission.
func TestJITInlineAdmissionDependsOnCallShape(t *testing.T) {
	for _, tc := range []struct {
		name string
		decl Declaration
		args []JITValueDesc
		want bool
	}{
		{"typed small body", Declaration{Type: &TypeDescriptor{JITInlineCost: 18}},
			[]JITValueDesc{{Type: tagInt, Loc: LocStack}}, true},
		{"typed large body", Declaration{Type: &TypeDescriptor{JITInlineCost: 49}},
			[]JITValueDesc{{Type: tagInt, Loc: LocStack}}, false},
		{"constant shape", Declaration{Type: &TypeDescriptor{JITInlineCost: 32}},
			[]JITValueDesc{{Type: JITTypeUnknown, Loc: LocImm, Imm: NewInt(1)}}, true},
		{"unknown dynamic argument", Declaration{Type: &TypeDescriptor{JITInlineCost: 18}},
			[]JITValueDesc{{Type: JITTypeUnknown, Loc: LocStackPair}}, false},
		{"no generated body", Declaration{RetainsCallArgs: true, Type: &TypeDescriptor{JITInlineCost: 65535}},
			[]JITValueDesc{{Type: tagInt, Loc: LocStack}}, false},
		{"unhandled lambda template", Declaration{RetainsCallArgs: true, Type: &TypeDescriptor{JITInlineCost: 18}},
			[]JITValueDesc{{Type: JITTypeUnknown, Loc: LocLambdaTemplate}}, false},
		{"variadic callback template", Declaration{RetainsCallArgs: true, Type: &TypeDescriptor{
			JITInlineCost: 18, JITInlineCallbacks: true,
			Params: []*TypeDescriptor{{Kind: "func", Params: []*TypeDescriptor{{Variadic: true}}}},
		}}, []JITValueDesc{{Type: JITTypeUnknown, Loc: LocLambdaTemplate}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, free := range []uint64{0, jitDefaultFreeGPRegs()} {
				for _, protected := range []uint64{0, jitDefaultFreeGPRegs()} {
					ctx := &JITContext{FreeRegs: free, ProtectedRegs: protected, AllRegs: jitDefaultFreeGPRegs()}
					if got := jitGeneratedEmitterInline(ctx, &tc.decl, tc.args); got != tc.want {
						t.Fatalf("free=%x protected=%x: inline=%v, want %v", free, protected, got, tc.want)
					}
				}
			}
		})
	}
}

// Compare filter truth against the interpreter, including UNKNOWN beneath NOT.
func TestJITFilterBooleanTruthTables(t *testing.T) {
	atoms := []Scmer{NewNil(), NewBool(false), NewBool(true), NewInt(0), NewInt(1)}
	for _, body := range []string{"(and a b)", "(or a b)", "(not (and a b))", "(not (or a b))", "(and (or a b) (not a))", "(or (and a b) (not b))", "(if a (or a b) (and a b))"} {
		t.Run(body, func(t *testing.T) {
			proc := calibrationProcedure("(lambda (a b) " + body + ")")
			kernel := CompileJITFilterBuffer(proc, []uint8{JITTypeUnknown, JITTypeUnknown})
			if kernel == nil {
				t.Fatal("filter kernel did not compile")
			}
			var values []Scmer
			var ids, want []uint32
			for _, a := range atoms {
				for _, b := range atoms {
					id := uint32(len(ids))
					ids = append(ids, id)
					values = append(values, a, b)
					env := &Env{Outer: &Globalenv, Vars: Vars{Symbol("a"): a, Symbol("b"): b}, VarsNumbered: []Scmer{a, b}}
					if Eval(proc.Body, env).Bool() {
						want = append(want, id)
					}
				}
			}
			n := kernel(ids, values)
			if n != len(want) {
				t.Fatalf("selected %v; interpreter selected %v", ids[:n], want)
			}
			for i, id := range want {
				if ids[i] != id {
					t.Fatalf("selected %v; interpreter selected %v", ids[:n], want)
				}
			}
		})
	}
}

func TestJITFilterUnknownPreservesEvaluation(t *testing.T) {
	for _, op := range []string{"and", "or"} {
		for _, conditional := range []bool{false, true} {
			body := "(" + op + " a (if b true (error \"unknown-continued\")))"
			if conditional {
				body = "(if " + body + " true false)"
			}
			proc := calibrationProcedure("(lambda (a b) " + body + ")")
			kernel := CompileJITFilterBuffer(proc, []uint8{JITTypeUnknown, tagBool})
			if kernel == nil {
				t.Fatal("filter did not compile")
			}
			func() {
				defer func() {
					switch err := recover().(type) {
					case nil:
						t.Errorf("%s conditional=%v skipped evaluation after UNKNOWN", op, conditional)
					case Scmer:
						if !Equal(err, NewString("unknown-continued")) {
							t.Fatalf("unexpected panic: %v", err)
						}
					case string:
						if err != "unknown-continued" {
							t.Fatalf("unexpected panic: %v", err)
						}
					default:
						t.Fatalf("unexpected panic: %v", err)
					}
				}()
				kernel([]uint32{0}, []Scmer{NewNil(), NewBool(false)})
			}()
			decisive := NewBool(op == "or")
			want := 0
			if op == "or" {
				want = 1
			}
			if got := kernel([]uint32{0}, []Scmer{decisive, NewBool(false)}); got != want {
				t.Fatalf("%s did not short-circuit decisive value", op)
			}
		}
	}
}

func TestJITBufferLoopsPreserveClosureCaptures(t *testing.T) {
	values := []Scmer{NewInt(6), NewInt(9)}
	for _, source := range []string{
		"(lambda (limit) (lambda (a) (> a limit)))",
		"(lambda (limit) (lambda (unused) (lambda (a) (> a limit))))",
	} {
		producer := CompileJIT(NewProcStruct(*calibrationProcedure(source)), true)
		callback := Apply(producer, NewInt(7))
		if len(callback.Proc().Params.Slice()) == 1 && callback.Proc().Params.Slice()[0].SymbolEquals("unused") {
			callback = Apply(callback, NewNil())
		}
		_, captures := callback.Proc().JITCapturedLocals()
		if len(captures) == 0 {
			t.Fatal("test did not produce an inline closure capture")
		}
		kernel := CompileJITFilterBuffer(callback.Proc(), []uint8{tagInt})
		ids := []uint32{0, 1}
		if kernel == nil || kernel(ids, values) != 1 || ids[0] != 1 {
			t.Fatalf("filter lost captured limit: %s", source)
		}
	}
	producer := CompileJIT(NewProcStruct(*calibrationProcedure("(lambda (factor) (lambda (acc a) (+ acc (* a factor))))")), true)
	callback := Apply(producer, NewInt(2))
	reduce := CompileJITMapReduceBuffer(callback.Proc(), []uint8{tagInt})
	if reduce == nil || !Equal(reduce(NewInt(0), values, 2), NewInt(30)) {
		t.Fatal("reducer lost captured multiplier")
	}
}

func TestJITTypedFloatConversionLocations(t *testing.T) {
	for _, value := range []Scmer{NewInt(-7), NewInt(1<<53 + 1), NewInt(math.MinInt64), NewInt(math.MaxInt64), NewFloat(math.Copysign(0, -1)), NewFloat(math.Inf(1)), NewFloat(math.NaN()), NewFloat(1.25)} {
		for _, location := range []string{"scalar", "pair", "stack", "stack-pair", "fp", "fp-stack"} {
			if (location == "fp" || location == "fp-stack") && !value.IsFloat() {
				continue
			}
			for _, preserve := range []bool{false, true} {
				fn := CompileJITStorageGetValue(func(ctx *JITContext, input, target JITValueDesc) JITValueDesc {
					input.Type = value.GetTag()
					var bits uint64
					if value.IsFloat() {
						bits = math.Float64bits(value.Float())
					} else {
						bits = uint64(value.Int())
					}
					ctx.EmitMovRegImm64(input.Reg, bits)
					if location == "fp" || location == "fp-stack" {
						// Standalone storage emitters normally reserve only GPRs.
						// Enable one native FP home to exercise the shared Proc path.
						ctx.AllFPRegs = 1 << uint(RegX2)
						ctx.FreeFPRegs = ctx.AllFPRegs
						ctx.EnsureFPReg(&input)
					}
					if location == "pair" || location == "stack-pair" {
						input = jitCopyScmerToPair(ctx, input)
					}
					if location == "stack" || location == "stack-pair" || location == "fp-stack" {
						ctx.StabilizeDescForControlFlow(&input)
					}
					before := len(ctx.Safepoints)
					converted := ctx.EmitFloatDesc(input)
					if len(ctx.Safepoints) != before {
						t.Errorf("%v/%s: typed conversion emitted a Go call", value, location)
					}
					if preserve {
						ctx.FreeDesc(&converted)
						return jitPlaceIntoPair(ctx, &input, target)
					}
					ctx.EmitMakeFloat(target, converted)
					return target
				})
				if fn == nil {
					t.Fatalf("%v/%s/preserve=%v: compile failed", value, location, preserve)
				}
				got, want := fn(0), NewFloat(value.Float())
				if preserve {
					want = value
				}
				if got != want {
					t.Fatalf("%v/%s/preserve=%v: got %#v, want %#v", value, location, preserve, got, want)
				}
			}
		}
	}
}

func TestJITFloatConversionDynamicFallback(t *testing.T) {
	for _, value := range []Scmer{NewInt(-7), NewFloat(1.25), NewString("12.5"), NewBool(true), NewNil()} {
		fn := CompileJITStorageGetValue(func(ctx *JITContext, input, target JITValueDesc) JITValueDesc {
			ctx.TrackImm(value)
			input = jitCopyScmerToPair(ctx, JITValueDesc{Loc: LocImm, Type: value.GetTag(), Imm: value})
			input.Type = JITTypeUnknown
			ctx.StabilizeDescForControlFlow(&input)
			converted := ctx.EmitFloatDesc(input)
			ctx.EmitMakeFloat(target, converted)
			return target
		})
		if fn == nil {
			t.Fatal("dynamic conversion did not compile")
		}
		if got, want := fn(0), NewFloat(value.Float()); got != want {
			t.Fatalf("%v: got %#v, want %#v", value, got, want)
		}
	}
}

func TestEmitCmpFloat64AvoidsDuplicateSameOperandMove(t *testing.T) {
	code := make([]byte, 16)
	ctx := &JITContext{
		Start: unsafe.Pointer(&code[0]),
		Ptr:   unsafe.Pointer(&code[0]),
		End:   unsafe.Pointer(&code[len(code)-1]),
	}
	ctx.EmitCmpFloat64(RegRAX, RegRAX)
	emitted := code[:uintptr(ctx.Ptr)-uintptr(ctx.Start)]
	want := []byte{
		0x66, 0x48, 0x0f, 0x6e, 0xc0, // MOVQ XMM0, RAX
		0x66, 0x0f, 0x2e, 0xc0, // UCOMISD XMM0, XMM0
	}
	if !bytes.Equal(emitted, want) {
		t.Fatalf("same-operand float comparison = %x, want %x", emitted, want)
	}
}

func emitParallelMoveTestCode(t *testing.T, batch *jitParallelRegMoveBatch) []byte {
	t.Helper()
	code := make([]byte, 128)
	ctx := &JITContext{
		Start:        unsafe.Pointer(&code[0]),
		Ptr:          unsafe.Pointer(&code[0]),
		End:          unsafe.Pointer(&code[len(code)-1]),
		SliceBase:    RegR12,
		ScratchReg:   RegR11,
		StackReg:     RegRSP,
		FrameReg:     RegRBP,
		RegisterBank: jitX86RegisterBank,
	}
	ctx.emitParallelRegMoveBatch(batch)
	if ctx.DynamicSP != 0 {
		t.Fatalf("parallel move left dynamic stack offset %d", ctx.DynamicSP)
	}
	return code[:uintptr(ctx.Ptr)-uintptr(ctx.Start)]
}

func TestParallelMoveBatchElidesIdentityAndOrdersDependencies(t *testing.T) {
	var batch jitParallelRegMoveBatch
	batch.add(RegRDX, RegRDX)
	batch.add(RegRAX, RegRBX)
	batch.add(RegRCX, RegRAX)

	// RCX must consume the old RAX before RAX is overwritten by RBX. Both x86
	// register moves are three bytes; the identity must emit nothing.
	code := emitParallelMoveTestCode(t, &batch)
	want := []byte{0x48, 0x89, 0xc1, 0x48, 0x89, 0xd8}
	if !bytes.Equal(code, want) {
		t.Fatalf("parallel dependency moves = %x, want %x", code, want)
	}
}

func TestParallelMoveBatchBreaksCycleWithOneSavedScratch(t *testing.T) {
	var batch jitParallelRegMoveBatch
	batch.add(RegRAX, RegRBX)
	batch.add(RegRBX, RegRAX)

	code := emitParallelMoveTestCode(t, &batch)
	// PUSH/POP R12 surround exactly three MOVs: save old RAX in scratch,
	// rotate RBX into RAX, then scratch into RBX.
	if len(code) != 13 || !bytes.Equal(code[:2], []byte{0x41, 0x54}) || !bytes.Equal(code[len(code)-2:], []byte{0x41, 0x5c}) {
		t.Fatalf("parallel cycle is not one saved-scratch rotation: %x", code)
	}
}

func TestParallelMoveBatchChoosesScratchOutsideCycle(t *testing.T) {
	var batch jitParallelRegMoveBatch
	batch.add(RegR12, RegR11)
	batch.add(RegR11, RegR12)

	// Both preferred role registers participate in the cycle. The solver must
	// select another register from the architecture-provided bank; reusing either
	// cycle member would fail to break the dependency (the old R12-specific
	// implementation could loop forever for this shape).
	code := emitParallelMoveTestCode(t, &batch)
	if len(code) != 13 {
		t.Fatalf("role-register cycle emitted %d bytes, want one saved-scratch rotation (13): %x", len(code), code)
	}
}

func TestParallelMoveBatchNeverUsesStackOrFrameRegisterAsScratch(t *testing.T) {
	var batch jitParallelRegMoveBatch
	batch.add(RegRAX, RegRBX)
	batch.add(RegRBX, RegRAX)

	ctx := JITContext{
		// Stack-backed argument lists deliberately use RSP as SliceBase. This
		// makes it a valid address base, not a writable temporary register.
		SliceBase:    RegRSP,
		ScratchReg:   RegR11,
		StackReg:     RegRSP,
		FrameReg:     RegRBP,
		RegisterBank: jitX86RegisterBank,
	}
	if scratch := ctx.parallelMoveScratch(&batch); scratch != RegR11 {
		t.Fatalf("parallel cycle scratch = %d, want non-frame scratch %d", scratch, RegR11)
	}
}

func TestDeferredRegisterMovesCollapseAcrossEmitterBoundaries(t *testing.T) {
	code := make([]byte, 128)
	ctx := &JITContext{
		Start:        unsafe.Pointer(&code[0]),
		Ptr:          unsafe.Pointer(&code[0]),
		End:          unsafe.Pointer(&code[len(code)-1]),
		SliceBase:    RegR12,
		ScratchReg:   RegR11,
		StackReg:     RegRSP,
		RegisterBank: jitX86RegisterBank,
	}

	// Model two independent inline emitters handing the same value through an
	// otherwise dead intermediate register. Ending the producer lifetime must
	// reserve its physical register for the alias rather than forcing an early
	// copy. The physical stream needs only the final RDI -> RDX move.
	ctx.AllRegs = uint64(jitRegisterMask(RegRDI, RegRSI, RegRDX))
	ctx.EmitMovRegReg(RegRSI, RegRDI)
	ctx.FreeReg(RegRDI)
	ctx.EmitMovRegReg(RegRDX, RegRSI)
	ctx.FreeReg(RegRSI)
	if ctx.Ptr != ctx.Start {
		t.Fatal("deferred moves emitted before a materialization barrier")
	}
	if ctx.FreeRegs&uint64(jitRegisterMask(RegRDI)) != 0 {
		t.Fatal("aliased physical source returned to allocator before materialization")
	}
	ctx.ReclaimUntrackedRegs()
	if ctx.FreeRegs&uint64(jitRegisterMask(RegRDI)) != 0 {
		t.Fatal("reclamation returned an aliased physical source before materialization")
	}
	ctx.FlushRegisterMoves()
	emitted := code[:uintptr(ctx.Ptr)-uintptr(ctx.Start)]
	if want := []byte{0x48, 0x89, 0xfa}; !bytes.Equal(emitted, want) {
		t.Fatalf("collapsed deferred chain = %x, want %x", emitted, want)
	}
	if ctx.FreeRegs&uint64(jitRegisterMask(RegRDI)) == 0 {
		t.Fatal("physical source remained held after its final alias materialized")
	}
}

func TestDeferredRegisterMovesPreserveOldSourceBeforeOverwrite(t *testing.T) {
	code := make([]byte, 128)
	ctx := &JITContext{
		Start:        unsafe.Pointer(&code[0]),
		Ptr:          unsafe.Pointer(&code[0]),
		End:          unsafe.Pointer(&code[len(code)-1]),
		SliceBase:    RegR12,
		ScratchReg:   RegR11,
		StackReg:     RegRSP,
		RegisterBank: jitX86RegisterBank,
	}

	ctx.EmitMovRegReg(RegRSI, RegRDI)
	ctx.EmitMovRegReg(RegRDI, RegRAX)
	ctx.FlushRegisterMoves()
	// RSI must receive the old RDI before the later RAX -> RDI assignment.
	emitted := code[:uintptr(ctx.Ptr)-uintptr(ctx.Start)]
	want := []byte{0x48, 0x89, 0xfe, 0x48, 0x89, 0xc7}
	if !bytes.Equal(emitted, want) {
		t.Fatalf("overwrite-preserving deferred moves = %x, want %x", emitted, want)
	}
}

func jitFourScalarResults(seed uint32) (int64, bool, int64, int64) {
	return int64(seed) + 1, seed&1 != 0, int64(seed) + 3, int64(seed) + 5
}

func TestJITScmerConstructorsAllowAliasedPayloadRegister(t *testing.T) {
	tests := []struct {
		name string
		want Scmer
		emit func(*JITContext, JITValueDesc, JITValueDesc)
	}{
		{"int", NewInt(42), func(ctx *JITContext, dst, src JITValueDesc) {
			ctx.EmitMovRegImm64(src.Reg, 42)
			ctx.EmitMakeInt(dst, src)
		}},
		{"float", NewFloat(-157.84), func(ctx *JITContext, dst, src JITValueDesc) {
			ctx.EmitMovRegImm64(src.Reg, math.Float64bits(-157.84))
			ctx.EmitMakeFloat(dst, src)
		}},
		{"bool", NewBool(true), func(ctx *JITContext, dst, src JITValueDesc) {
			ctx.EmitMovRegImm64(src.Reg, 1)
			ctx.EmitMakeBool(dst, src)
		}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			fn := CompileJITStorageGetValue(func(ctx *JITContext, source, target JITValueDesc) JITValueDesc {
				if source.Reg != target.Reg {
					t.Fatalf("test requires an aliased source and pointer destination, got %v and %v", source.Reg, target.Reg)
				}
				test.emit(ctx, target, source)
				return target
			})
			if fn == nil {
				t.Fatal("aliased constructor did not compile")
			}
			if got := fn(0); !Equal(got, test.want) {
				t.Fatalf("aliased constructor = %v, want %v", got, test.want)
			}
		})
	}
}

func TestJITStorageScalarInputSurvivesScratchReclamation(t *testing.T) {
	fn := CompileJITStorageGetValue(func(ctx *JITContext, source, target JITValueDesc) JITValueDesc {
		// Generated CFG emitters reclaim registers whose descriptors no longer own
		// them. The incoming RAX is still live here and therefore must not become
		// scratch merely because it is also the eventual result register.
		ctx.ReclaimUntrackedRegs()
		scratch := ctx.AllocReg()
		ctx.EmitMovRegImm64(scratch, 99)
		ctx.FreeReg(scratch)
		value := JITValueDesc{Loc: LocRegPair, Type: tagInt, Reg: target.Reg, Reg2: target.Reg2}
		ctx.EmitMakeInt(value, source)
		return value
	})
	if fn == nil {
		t.Fatal("scalar storage JIT function did not compile")
	}
	if got, want := fn(17), NewInt(17); !Equal(got, want) {
		t.Fatalf("scalar input after scratch reclamation = %v, want %v", got, want)
	}
}

func TestJITLessHelperEmitsKnownIntegerComparison(t *testing.T) {
	fn := CompileJITStorageGetValue(func(ctx *JITContext, source, target JITValueDesc) JITValueDesc {
		source.Type = tagInt
		comparison := jitEmitLess(ctx, []JITValueDesc{
			source,
			{Loc: LocImm, Type: tagInt, Imm: NewInt(511)},
		}, JITValueDesc{Loc: LocReg, Type: tagBool, Reg: target.Reg2, ID: 0})
		ctx.EmitMakeBool(target, comparison)
		return target
	})
	if fn == nil {
		t.Fatal("known integer Less helper did not compile")
	}
	for _, test := range []struct {
		value uint32
		want  bool
	}{{510, true}, {511, false}, {512, false}} {
		if got := fn(test.value).Bool(); got != test.want {
			t.Fatalf("(< %d 511) = %v, want %v", test.value, got, test.want)
		}
	}
}

func TestJITStorageReadersSharePackedCodeLifetime(t *testing.T) {
	scalar, ranged, multi := CompileJITStorageReaders(
		func(ctx *JITContext, source, target JITValueDesc) JITValueDesc {
			ctx.EmitMakeInt(target, source)
			return target
		},
		func(_ *JITContext, _, _, _, _, result JITValueDesc) JITValueDesc { return result },
		func(_ *JITContext, _, _, _, result JITValueDesc) JITValueDesc { return result },
	)
	if scalar == nil || ranged == nil || multi == nil {
		t.Fatal("storage reader batch did not compile every ABI")
	}

	scalarValue := *(*unsafe.Pointer)(unsafe.Pointer(&scalar))
	rangeValue := *(*unsafe.Pointer)(unsafe.Pointer(&ranged))
	multiValue := *(*unsafe.Pointer)(unsafe.Pointer(&multi))
	scalarHolder := (*jitStorageFuncValue)(scalarValue)
	rangeHolder := (*jitStorageFuncValue)(rangeValue)
	multiHolder := (*jitStorageFuncValue)(multiValue)
	if scalarHolder.owner == nil || scalarHolder.owner != rangeHolder.owner || scalarHolder.owner != multiHolder.owner {
		t.Fatal("storage reader funcvals do not share one code owner")
	}
	if len(scalarHolder.owner.entries) != 3 {
		t.Fatalf("shared code owner has %d entries, want 3", len(scalarHolder.owner.entries))
	}
	if gap := rangeHolder.code - scalarHolder.code; gap == 0 || gap >= 16*1024 {
		t.Fatalf("scalar-to-range code gap = %d, want densely packed functions", gap)
	}
	if gap := multiHolder.code - rangeHolder.code; gap == 0 || gap >= 16*1024 {
		t.Fatalf("range-to-multi code gap = %d, want densely packed functions", gap)
	}
	runtime.GC()
	if got := scalar(23); !Equal(got, NewInt(23)) {
		t.Fatalf("packed scalar result = %v, want 23", got)
	}
}

func TestJITGoCallFourScalarResults(t *testing.T) {
	fn := CompileJITStorageGetValue(func(ctx *JITContext, seed, target JITValueDesc) JITValueDesc {
		results := JITEmitGoCallResults(ctx, GoFuncAddr(jitFourScalarResults), []JITValueDesc{seed}, []uint8{1, 1, 1, 1}, []uint8{0, 0, 0, 0})
		for index := range results {
			ctx.EnsureDesc(&results[index])
		}
		ctx.EmitImulRegImm32(results[1].Reg, 10)
		ctx.EmitImulRegImm32(results[2].Reg, 100)
		ctx.EmitImulRegImm32(results[3].Reg, 1000)
		ctx.EmitAddInt64(results[0].Reg, results[1].Reg)
		ctx.EmitAddInt64(results[0].Reg, results[2].Reg)
		ctx.EmitAddInt64(results[0].Reg, results[3].Reg)
		ctx.EnsureDesc(&seed)
		ctx.EmitAddInt64(results[0].Reg, seed.Reg)
		value := JITValueDesc{Loc: LocRegPair, Type: tagInt, Reg: target.Reg, Reg2: target.Reg2}
		ctx.EmitMakeInt(value, results[0])
		return value
	})
	if fn == nil {
		t.Fatal("four-result JIT function did not compile")
	}
	if got, want := fn(7), NewInt(8+10+1000+12000+7); !Equal(got, want) {
		t.Fatalf("four-result Go ABI call = %v, want %v", got, want)
	}
}

func TestJITStoreCustomScmerKeepsSpilledAddress(t *testing.T) {
	payload := new(byte)
	value := NewCustom(200, unsafe.Pointer(payload))
	fn := CompileJITStorageGetValueRange(func(ctx *JITContext, index, count, target, stride, result JITValueDesc) JITValueDesc {
		address := ctx.EmitSliceElementAddress(&target, &index, 16)
		addressOff := ctx.AllocStack(8)
		ctx.EmitStoreRegMem(address.Reg, ctx.StackReg, addressOff)
		ctx.FreeDesc(&address)
		address = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: addressOff, NoHeapPointer: true}
		stored := JITValueDesc{Loc: LocImm, Type: value.GetTag(), Imm: value, Rooted: true}
		ctx.EmitStoreScmerAt(&address, &stored)
		return result
	})
	if fn == nil {
		t.Fatal("spilled-address JIT function did not compile")
	}
	target := make([]Scmer, 1)
	fn(0, 1, target, 1)
	runtime.GC()
	if got := target[0]; got.GetTag() != value.GetTag() || got.ptr != value.ptr {
		t.Fatalf("stored custom Scmer = %v, want tag %d at %p", got, value.GetTag(), value.ptr)
	}
	runtime.KeepAlive(payload)
}

func TestJITLessHelperKnownLargeIntegers(t *testing.T) {
	for _, base := range []int64{1 << 53, -1 << 63, 1<<63 - 3} {
		fn := CompileJITStorageGetValue(func(ctx *JITContext, source, target JITValueDesc) JITValueDesc {
			source.Type = tagInt
			ctx.ProtectReg(source.Reg)
			offset := ctx.AllocReg()
			ctx.EmitMovRegImm64(offset, uint64(base))
			ctx.EmitAddInt64(source.Reg, offset)
			ctx.FreeReg(offset)
			ctx.UnprotectReg(source.Reg)
			comparison := jitEmitLess(ctx, []JITValueDesc{source,
				{Loc: LocImm, Type: tagInt, Imm: NewInt(base + 1)},
			}, JITValueDesc{Loc: LocReg, Type: tagBool, Reg: target.Reg2})
			ctx.EmitMakeBool(target, comparison)
			return target
		})
		if fn == nil {
			t.Fatal("known int64 comparison did not compile")
		}
		for i := uint32(0); i < 3; i++ {
			if got := fn(i).Bool(); got != (i < 1) {
				t.Fatalf("Less(%d, %d) = %v", base+int64(i), base+1, got)
			}
		}
	}
}

func TestJITComparisonReturnsFlagsThroughNewBool(t *testing.T) {
	for _, op := range []string{"<", ">", "<=", ">="} {
		t.Run(op, func(t *testing.T) {
			fn := CompileJITStorageGetValue(func(ctx *JITContext, source, target JITValueDesc) JITValueDesc {
				source.Type = tagInt
				value := declarations[op].Type.JITEmit(ctx, nil, []JITValueDesc{source,
					{Loc: LocImm, Type: tagInt, Imm: NewInt(511)}}, target)
				if !ctx.hasBooleanFlags(value) {
					t.Fatalf("%s materialized the returned comparison: %+v", op, value)
				}
				ctx.lazyFlags = jitBooleanFlags{}
				yes, done := ctx.ReserveLabel(), ctx.ReserveLabel()
				before := uintptr(ctx.Ptr)
				ctx.EmitJump(value.Condition, yes)
				code := unsafe.Slice((*byte)(unsafe.Pointer(before)), int(uintptr(ctx.Ptr)-before))
				if len(code) != 6 || code[0] != 0x0f || code[1] != 0x80|x86ConditionCode(value.Condition) {
					t.Fatalf("condition did not become a direct Jcc: %x", code)
				}
				ctx.FreeDesc(&value)
				ctx.EmitMakeInt(target, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(7)})
				ctx.EmitJmp(done)
				ctx.MarkLabel(yes)
				ctx.EmitMakeInt(target, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(9)})
				ctx.MarkLabel(done)
				return target
			})
			if fn == nil {
				t.Fatal("comparison branch did not compile")
			}
			for _, n := range []uint32{510, 511, 512} {
				want := int64(7)
				if (op == "<" && n < 511) || (op == ">" && n > 511) || (op == "<=" && n <= 511) || (op == ">=" && n >= 511) {
					want = 9
				}
				if got := fn(n).Int(); got != want {
					t.Fatalf("%s %d: got %d, want %d", op, n, got, want)
				}
			}
		})
	}
}

// Float comparisons materialize a Scmer pair rather than returning integer flags.
// Their non-null return type must survive the full declaration call boundary.
func TestJITComparisonMergedReturnType(t *testing.T) {
	for _, op := range []string{"<", ">", "<=", ">="} {
		t.Run(op, func(t *testing.T) {
			expr := NewSlice([]Scmer{NewSymbol(op), NewNthLocalVar(0), NewFloat(511)})
			fn := CompileJITStorageGetValue(func(ctx *JITContext, source, target JITValueDesc) JITValueDesc {
				source.Type = tagInt
				ctx.Env = &JITEnv{Numbered: []JITValueDesc{source}}
				out := jitCompileExpr(ctx, expr, ctx.SliceBase, target)
				if out.Type != tagBool {
					t.Fatalf("comparison lost proven bool type: %+v", out)
				}
				return out
			})
			if fn == nil {
				t.Fatal("comparison did not compile")
			}
			for _, n := range []uint32{510, 511, 512} {
				want := (op == "<" && n < 511) || (op == ">" && n > 511) || (op == "<=" && n <= 511) || (op == ">=" && n >= 511)
				if got := fn(n); got.GetTag() != tagBool || got.Bool() != want {
					t.Fatalf("%s %d: %v", op, n, got)
				}
			}
		})
	}
}

func TestJITComparisonNullableReturnType(t *testing.T) {
	nullable := NewSlice([]Scmer{NewSymbol("if"), NewSlice([]Scmer{NewSymbol("<"), NewNthLocalVar(0), NewInt(1)}), NewNil(), NewNthLocalVar(0)})
	for _, op := range []string{"<", ">", "<=", ">="} {
		t.Run(op, func(t *testing.T) {
			expr := NewSlice([]Scmer{NewSymbol(op), nullable, NewFloat(511)})
			fn := CompileJITStorageGetValue(func(ctx *JITContext, source, target JITValueDesc) JITValueDesc {
				source.Type = tagInt
				ctx.Env = &JITEnv{Numbered: []JITValueDesc{source}}
				out := jitCompileExpr(ctx, expr, ctx.SliceBase, target)
				if out.Type != JITTypeUnknown {
					t.Fatalf("nullable comparison claimed exact type: %+v", out)
				}
				return out
			})
			if fn == nil {
				t.Fatal("nullable comparison did not compile")
			}
			if got := fn(0); !got.IsNil() {
				t.Fatalf("NULL comparison returned %v", got)
			}
			for _, n := range []uint32{510, 511, 512} {
				want := (op == "<" && n < 511) || (op == ">" && n > 511) || (op == "<=" && n <= 511) || (op == ">=" && n >= 511)
				if got := fn(n); got.GetTag() != tagBool || got.Bool() != want {
					t.Fatalf("%s %d: %v", op, n, got)
				}
			}
		})
	}
}

// A location reused by another emitter must not retain the previous value's
// proof. This deliberately unmerged emitter models the older generated CFGs.
func TestJITUnmergedReturnRejectsInheritedProof(t *testing.T) {
	const name = "jit_test_unmerged_return"
	native := func(a ...Scmer) Scmer {
		if a[0].Int() == 0 {
			return NewNil()
		}
		return NewBool(true)
	}
	Declare(&Globalenv, &Declaration{Name: name, Fn: native,
		Type: &TypeDescriptor{Kind: "func", Params: []*TypeDescriptor{{Kind: "int"}}, Return: &TypeDescriptor{Kind: "any"},
			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if result.ReturnTypeMerged {
					t.Fatal("new emitter inherited an unrelated return proof")
				}
				ctx.EnsureDesc(&args[0])
				yes, done := ctx.ReserveLabel(), ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(args[0].Reg, 0)
				ctx.EmitJcc(CcNE, yes)
				ctx.EmitMakeNil(result)
				ctx.EmitJmp(done)
				ctx.MarkLabel(yes)
				ctx.EmitMakeBool(result, JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)})
				ctx.MarkLabel(done)
				result.Type = tagBool // Only the last arm; no valid merge proof.
				return result
			}}})
	defer func() {
		delete(Globalenv.Vars, Symbol(name))
		delete(declarations, name)
		delete(declarationsByFunction, FunctionIdentity(native))
	}()
	expr := NewSlice([]Scmer{NewSymbol(name), NewNthLocalVar(0)})
	fn := CompileJITStorageGetValue(func(ctx *JITContext, source, target JITValueDesc) JITValueDesc {
		source.Type = tagInt
		ctx.Env = &JITEnv{Numbered: []JITValueDesc{source}}
		target.ReturnTypeMerged = true
		out := jitCompileExpr(ctx, expr, ctx.SliceBase, target)
		if out.Type != JITTypeUnknown {
			t.Fatalf("unmerged nullable result claimed exact type: %+v", out)
		}
		return out
	})
	if fn == nil || !fn(0).IsNil() || !fn(1).Bool() {
		t.Fatal("unmerged result lost NULL/boolean semantics")
	}
}

func BenchmarkJITFloatComparisonBranch(b *testing.B) {
	expr := NewSlice([]Scmer{NewSymbol("if"), NewSlice([]Scmer{NewSymbol("<"), NewNthLocalVar(0), NewFloat(511)}), NewInt(9), NewInt(7)})
	fn := CompileJITStorageGetValue(func(ctx *JITContext, source, target JITValueDesc) JITValueDesc {
		source.Type = tagInt
		ctx.Env = &JITEnv{Numbered: []JITValueDesc{source}}
		return jitCompileExpr(ctx, expr, ctx.SliceBase, target)
	})
	if fn == nil || fn(510).Int() != 9 || fn(511).Int() != 7 {
		b.Fatal("comparison branch failed")
	}
	b.ReportAllocs()
	b.ResetTimer()
	var result int64
	for i := 0; i < b.N; i++ {
		result += fn(uint32(i & 1023)).Int()
	}
	b.StopTimer()
	runtime.KeepAlive(fn)
	if result == 0 {
		b.Fatal("benchmark did not execute")
	}
}

// Benchmark the complete typed Scheme condition, including its generated
// comparison wrapper. The fixture is also usable unchanged on the baseline.
func BenchmarkJITIntegerBranch(b *testing.B) {
	expr := NewSlice([]Scmer{NewSymbol("if"),
		NewSlice([]Scmer{NewSymbol("<"), NewNthLocalVar(0), NewInt(511)}), NewInt(9), NewInt(7)})
	fn := CompileJITStorageGetValue(func(ctx *JITContext, source, target JITValueDesc) JITValueDesc {
		source.Type = tagInt
		ctx.Env = &JITEnv{Numbered: []JITValueDesc{source}}
		return jitCompileExpr(ctx, expr, ctx.SliceBase, target)
	})
	if fn == nil || fn(510).Int() != 9 || fn(511).Int() != 7 {
		b.Fatal("typed integer branch failed to compile or execute")
	}
	b.ReportAllocs()
	b.ResetTimer()
	var result int64
	for i := 0; i < b.N; i++ {
		result += fn(uint32(i & 1023)).Int()
	}
	b.StopTimer()
	runtime.KeepAlive(fn)
	if result == 0 {
		b.Fatal("benchmark did not execute")
	}
}

func TestJITDeferredComparisonSurvivesClobberAndSpill(t *testing.T) {
	for _, spill := range []bool{false, true} {
		fn := CompileJITStorageGetValue(func(ctx *JITContext, source, target JITValueDesc) JITValueDesc {
			source.Type = tagInt
			value := jitEmitLess(ctx, []JITValueDesc{source, {Loc: LocImm, Type: tagInt, Imm: NewInt(511)}}, JITValueDesc{Loc: LocAny})
			if spill {
				ctx.StabilizeDescForControlFlow(&value)
			}
			ctx.EmitMovRegImm64(RegR11, 0)
			ctx.EmitAddRegImm32(RegR11, 1)
			ctx.EnsureDesc(&value)
			ctx.EmitMakeBool(target, value)
			return target
		})
		if fn == nil {
			t.Fatal("deferred comparison did not compile")
		}
		for _, n := range []uint32{510, 511, 512} {
			if got := fn(n).Bool(); got != (n < 511) {
				t.Fatalf("spill=%v n=%d: got %v", spill, n, got)
			}
		}
	}
}

// Time an actual filter loop with eight typed comparisons. The batch survives
// every invocation because all rows qualify, so no reset/copy enters timing.
func BenchmarkJITBooleanFilterBatch(b *testing.B) {
	proc := calibrationProcedure("(lambda (v w) (and (> v -1) (< v 60001) (> w -1) (< w 60002) (>= v 0) (<= v 60000) (>= w 1) (<= w 60001)))")
	kernel := CompileJITFilterBuffer(proc, []uint8{tagInt, tagInt})
	if kernel == nil {
		b.Fatal("filter kernel did not compile")
	}
	ids := make([]uint32, 512)
	values := make([]Scmer, len(ids)*2)
	for i := range ids {
		ids[i] = uint32(i)
		values[i*2], values[i*2+1] = NewInt(int64(i)), NewInt(int64(i+1))
	}
	if kernel(ids, values) != len(ids) {
		b.Fatal("filter discarded qualifying rows")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if kernel(ids, values) != len(ids) {
			b.Fatal("filter changed its result")
		}
	}
}

func TestJITClosureCaptureForwardingUnderRegisterPressure(t *testing.T) {
	values := []Scmer{NewString("retained capture")}
	closure := jitBindProcContext(jitProcContextAllocation(1), &Proc{}, &values[0], 1, false)
	fn := CompileJITStorageGetValue(func(ctx *JITContext, _, target JITValueDesc) JITValueDesc {
		ctx.TrackPointer(unsafe.Pointer(closure))
		ctx.ClosureFuncOff = ctx.AllocStack(8)
		ctx.EmitMovRegImm64(RegR11, uint64(uintptr(unsafe.Pointer(closure))))
		ctx.EmitStoreRegMem(RegR11, ctx.StackReg, ctx.ClosureFuncOff)
		ctx.setStackPointer(jitStackRootFrameSP, ctx.ClosureFuncOff-ctx.DynamicSP, true)
		ctx.Env = &JITEnv{Numbered: []JITValueDesc{{Loc: LocClosurePair, Type: JITTypeUnknown, StackOff: 0, Rooted: true}}}
		off := ctx.AllocStack(16)
		var held [16]JITValueDesc
		count := 0
		for ctx.FreeRegs&^ctx.ProtectedRegs != 0 {
			reg := ctx.AllocReg()
			held[count] = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: reg, NoHeapPointer: true}
			ctx.EmitMovRegImm64(reg, uint64(count))
			ctx.BindReg(reg, &held[count])
			ctx.ProtectReg(reg)
			count++
		}
		forwarded := jitCompileRootedCallValueAt(ctx, NewNthLocalVar(0), ctx.SliceBase, off)
		ctx.EmitGoCallVoid(GoFuncAddr(runtime.GC), nil)
		for i := 0; i < count; i++ {
			ctx.UnprotectReg(held[i].Reg)
			ctx.FreeDesc(&held[i])
		}
		return jitPlaceIntoPair(ctx, &forwarded, target)
	})
	if fn == nil {
		t.Fatal("capture forwarding required an allocator register")
	}
	for i := 0; i < 4; i++ {
		if got := fn(uint32(i)); got.String() != "retained capture" {
			t.Fatalf("capture corrupted: %v", got)
		}
	}
	runtime.KeepAlive(closure)
}
