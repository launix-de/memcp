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
	"testing"
	"unsafe"
)

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
		RegisterBank: jitNativeRegisterBank,
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

func TestDeferredRegisterMovesCollapseAcrossEmitterBoundaries(t *testing.T) {
	code := make([]byte, 128)
	ctx := &JITContext{
		Start:        unsafe.Pointer(&code[0]),
		Ptr:          unsafe.Pointer(&code[0]),
		End:          unsafe.Pointer(&code[len(code)-1]),
		SliceBase:    RegR12,
		ScratchReg:   RegR11,
		StackReg:     RegRSP,
		RegisterBank: jitNativeRegisterBank,
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
		RegisterBank: jitNativeRegisterBank,
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

var jitFrameClearInstruction = []byte{0xf3, 0x48, 0xab}

//go:noinline
//go:nosplit
func jitTestZeroBuffer(dst *[32]byte) {
	*dst = [32]byte{}
}

func TestJITFrameClearingLimitedToParserControlFlow(t *testing.T) {
	t.Run("query projection", func(t *testing.T) {
		compiled := compileJITExpressionTestProc(t, `(lambda (a b) (list "a" a "b" b))`)
		if bytes.Contains(jitEntryCode(compiled.Proc().Compiled), jitFrameClearInstruction) {
			t.Fatal("ordinary expression clears its entire JIT frame")
		}
	})

	t.Run("parser", func(t *testing.T) {
		compiled := compileJITExpressionTestProc(t, `(lambda (input) (begin
			(define word (parser (regex "[a-z]+" false false)))
			((parser '((define value word) $) value "") input)))`)
		if !bytes.Contains(jitEntryCode(compiled.Proc().Compiled), jitFrameClearInstruction) {
			t.Fatal("parser expression does not clear shared JIT frame targets")
		}
	})
}

func TestJITPersistentRegisterBankExcludesGoScratchR15(t *testing.T) {
	for index := uint8(0); index < jitNativeRegisterBank.Count; index++ {
		if jitNativeRegisterBank.Registers[index] == RegR15 {
			t.Fatal("R15 may be used as a block-local temporary, not as a persistent control-flow home")
		}
	}
}

func TestJITFloatRegisterPressurePreservesGoZeroRegister(t *testing.T) {
	const name = "jit_test_fp_zero_register"
	for _, nativeCall := range []bool{false, true} {
		label := "return"
		if nativeCall {
			label = "native call"
		}
		t.Run(label, func(t *testing.T) {
			buffer := new([32]byte)
			declaration := &Declaration{
				Name: name,
				Fn:   func(...Scmer) Scmer { return NewInt(0) },
				Type: &TypeDescriptor{Kind: "func", Return: &TypeDescriptor{Kind: "int"},
					JITEmit: func(ctx *JITContext, _ []Scmer, _ []JITValueDesc, result JITValueDesc) JITValueDesc {
						// Exhaust the production FP bank, as scalar overflow and
						// generated builtin temporaries do under register pressure.
						ctx.EmitMovRegImm64(ctx.ScratchReg, 1)
						for ctx.FreeFPRegs != 0 {
							ctx.EmitMovGPRToFP(ctx.AllocFPReg(), ctx.ScratchReg)
						}
						if nativeCall {
							ctx.TrackPointer(unsafe.Pointer(buffer))
							ctx.EmitGoCallVoid(GoFuncAddr(jitTestZeroBuffer), []JITValueDesc{
								{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(buffer)))), NoHeapPointer: true},
							})
						}
						target := jitEnsureResultPair(ctx, result)
						ctx.EmitMakeInt(target, JITValueDesc{Loc: LocFPReg, Type: tagInt, Reg: RegX15})
						// Restore the ABI even on the broken implementation so the
						// test reports failure without corrupting the test runner.
						ctx.EmitMovRegImm64(ctx.ScratchReg, 0)
						ctx.EmitMovGPRToFP(RegX15, ctx.ScratchReg)
						return target
					},
				},
			}
			Declare(&Globalenv, declaration)
			defer func() {
				delete(Globalenv.Vars, Symbol(name))
				delete(declarations, name)
				delete(declarationsByFunction, FunctionIdentity(declaration.Fn))
			}()
			compiled := compileJITExpressionTestProc(t, `(lambda () (jit_test_fp_zero_register))`)
			got := compiled.Proc().jitFunction()()
			if got.Int() != 0 {
				t.Errorf("Go ABI zero register contains %d after FP register pressure", got.Int())
			}
			if *buffer != [32]byte{} {
				t.Errorf("native Go zero initialization wrote %x", *buffer)
			}
		})
	}
}
