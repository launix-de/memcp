//go:build goexperiment.jit && amd64

// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later

package scm

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
	"unsafe"
)

// The intermediate Go header is a GC root even when NewSlice/NewString later
// canonicalizes an empty result. Check the emitted address before that step.
func TestJITSliceDataDoesNotPointPastInput(t *testing.T) {
	for _, header := range []string{"slice", "string", "array", "stack-slice", "stack-string", "spill-slice", "spill-string", "wide-slice"} {
		for _, indexCase := range []struct {
			low     int
			dynamic bool
		}{{0, false}, {2, false}, {3, false}, {0, true}, {2, true}, {3, true}} {
			low := indexCase.low
			t.Run(fmt.Sprintf("%s/%d/dynamic=%t", header, low, indexCase.dynamic), func(t *testing.T) {
				input := make([]Scmer, 6)
				base := unsafe.Pointer(unsafe.SliceData(input))
				const name = "jit_test_slice_boundary"
				declaration := &Declaration{
					Name: name,
					Fn:   func(...Scmer) Scmer { return NewNil() },
					Type: &TypeDescriptor{Kind: "func", Return: &TypeDescriptor{Kind: "int"},
						JITEmit: func(ctx *JITContext, _ []Scmer, _ []JITValueDesc, result JITValueDesc) JITValueDesc {
							ctx.TrackPointer(base)
							slice := JITValueDesc{Loc: LocMem, GoArray: true, SliceSizeKnown: true, KnownSliceLen: 3, KnownSliceCap: 3, MemPtr: uintptr(base)}
							size := int32(16)
							if header == "wide-slice" {
								size = 24
							}
							if header != "array" {
								ptr := ctx.AllocReg()
								length := ctx.AllocRegExcept(ptr)
								ctx.EmitMovRegImm64(ptr, uint64(uintptr(base)))
								ctx.EmitMovRegImm64(length, 3)
								slice = JITValueDesc{Loc: LocRegPair, Reg: ptr, Reg2: length}
								ctx.BindReg(ptr, &slice)
								ctx.BindReg(length, &slice)
								if strings.HasSuffix(header, "slice") {
									capacity := ctx.AllocRegExcept(ptr, length)
									ctx.EmitMovRegImm64(capacity, 3)
									slice.Loc, slice.Reg3 = LocRegTriple, capacity
									ctx.BindReg(capacity, &slice)
								} else {
									size = 1
								}
								if strings.Contains(header, "-") {
									stack, offset := ctx.StackReg, ctx.AllocStack(24)
									if strings.HasPrefix(header, "spill-") {
										stack, offset = ctx.FrameReg, ctx.AllocSpill(24)
									}
									ctx.EmitStoreRegMem(ptr, stack, offset)
									ctx.EmitStoreRegMem(length, stack, offset+8)
									loc := LocStackPair
									if slice.Loc == LocRegTriple {
										ctx.EmitStoreRegMem(slice.Reg3, stack, offset+16)
										loc = LocStackTriple
									}
									ctx.FreeDesc(&slice)
									slice = JITValueDesc{Loc: loc, StackOff: offset}
								}
							}
							index := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(low))}
							if indexCase.dynamic {
								reg := ctx.AllocReg()
								ctx.EmitMovRegImm64(reg, uint64(low))
								index = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: reg}
								ctx.BindReg(reg, &index)
							}
							ptr := ctx.EmitSliceDataAfterLow(&slice, &index, size)
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(uintptr(base)))
							ctx.EmitSubInt64(ptr, ctx.ScratchReg)
							return jitPlaceScmerIntoTarget(ctx, JITValueDesc{Loc: LocReg, Type: tagInt, Reg: ptr}, result)
						},
					},
				}
				Declare(&Globalenv, declaration)
				defer func() {
					delete(Globalenv.Vars, Symbol(name))
					delete(declarations, name)
					delete(declarationsByFunction, FunctionIdentity(declaration.Fn))
				}()
				compiled := compileJITExpressionTestProc(t, `(lambda () (jit_test_slice_boundary))`)
				got := Apply(compiled).Int()
				want := int64(low * 16)
				if header == "wide-slice" {
					want = int64(low * 24)
				}
				if strings.HasSuffix(header, "string") {
					want = int64(low)
				}
				if low == 3 {
					want = 0
				}
				if got != want {
					t.Fatalf("data offset = %d, want %d (no one-past-end GC pointer)", got, want)
				}
				runtime.KeepAlive(input)
			})
		}
	}
}

// Keep the intermediate empty header rooted across GC, before its consumer
// canonicalizes it. Enough 48-byte arrays exercise the end of allocator spans.
func TestJITEmptySliceHeaderSurvivesGC(t *testing.T) {
	const name = "jit_test_empty_slice_header"
	declaration := &Declaration{
		Name: name,
		Fn:   func(args ...Scmer) Scmer { return NewSlice(args[0].Slice()[3:]) },
		Type: &TypeDescriptor{Kind: "func", Params: []*TypeDescriptor{{Kind: "list"}}, Return: &TypeDescriptor{Kind: "list"},
			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				value := args[0]
				ctx.EnsureDesc(&value)
				ptr := ctx.AllocRegExcept(value.Reg, value.Reg2)
				ctx.EmitMovRegReg(ptr, value.Reg)
				header := JITValueDesc{Loc: LocRegPair, Reg: ptr}
				ctx.BindReg(ptr, &header)
				bound := ctx.AllocRegExcept(ptr)
				ctx.EmitMovRegImm64(bound, 3)
				header.Reg2 = bound
				ctx.BindReg(bound, &header)
				low := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(3)}
				data := ctx.EmitSliceDataAfterLow(&header, &low, 16)
				empty := JITValueDesc{Loc: LocRegPair, Type: tagSlice, Reg: data}
				ctx.BindReg(data, &empty)
				aux := ctx.AllocRegExcept(data)
				ctx.EmitMovRegImm64(aux, makeAux(tagSlice, 0))
				empty.Reg2 = aux
				ctx.BindReg(aux, &empty)
				return jitPlaceScmerIntoTarget(ctx, empty, result)
			},
		},
	}
	Declare(&Globalenv, declaration)
	defer func() {
		delete(Globalenv.Vars, Symbol(name))
		delete(declarations, name)
		delete(declarationsByFunction, FunctionIdentity(declaration.Fn))
	}()
	compiled := compileJITExpressionTestProc(t, `(lambda (value) (jit_test_empty_slice_header value))`)
	for round := 0; round < 3; round++ {
		roots := make([]Scmer, 10000)
		for i := range roots {
			roots[i] = Apply(compiled, NewSlice(make([]Scmer, 3)))
		}
		runtime.GC()
		runtime.KeepAlive(roots)
	}
}

func TestJITEmptyStringDoesNotPointPastInput(t *testing.T) {
	for _, test := range []struct{ name, source string }{
		{"regex capture", `(lambda (value) (match value (regex "a*()$" all empty) empty _ false))`},
		{"substring", `(lambda (value) (substr value (strlen value)))`},
		{"parser remainder", `(lambda (value) ((parser '((atom "aaaa" false) (define rest (regex "a*" false false)) $) rest "") value))`},
	} {
		t.Run(test.name, func(t *testing.T) {
			compiled := compileJITExpressionTestProc(t, test.source)
			input := strings.Repeat("a", 4)
			end := uintptr(unsafe.Pointer(unsafe.StringData(input))) + uintptr(len(input))
			got := Apply(compiled, NewString(input))
			if got.GetTag() != tagString || got.String() != "" {
				t.Fatalf("expected empty string, got %s", String(got))
			}
			if uintptr(unsafe.Pointer(got.ptr)) == end {
				got = NewNil()
				t.Fatal("JIT published one-past-end address as a GC pointer")
			}
			runtime.GC()
			runtime.KeepAlive(got)
			runtime.KeepAlive(input)
		})
	}
}

func TestJITEmptyStringSurvivesGCInCallerFrame(t *testing.T) {
	collect := NewFunc(func(args ...Scmer) Scmer {
		runtime.GC()
		return args[0]
	})
	for _, source := range []string{
		`(lambda (value collect) (match value (regex "a*()$" all empty) (begin (collect empty) empty) _ false))`,
		`(lambda (value collect) (begin (define rest ((parser '((regex "a+" false false) (define rest (regex "a*" false false)) $) rest "") value)) (collect rest) rest))`,
	} {
		compiled := compileJITExpressionTestProc(t, source)
		input := strings.Repeat("a", 64<<10)
		for i := 0; i < 4; i++ {
			got := Apply(compiled, NewString(input), collect)
			if got.GetTag() != tagString || got.String() != "" {
				t.Fatalf("expected empty string after GC, got %s", String(got))
			}
		}
		runtime.KeepAlive(input)
	}
}

func TestJITSQLParameterizationSurvivesGC(t *testing.T) {
	environment := &Env{Vars: make(Vars), Outer: &Globalenv}
	source, err := os.ReadFile("../lib/sql-parameters.scm")
	if err != nil {
		t.Fatal(err)
	}
	EvalAllJIT("sql-parameters.scm", string(source), environment)
	fn := environment.Vars[Symbol("parameterize_sql_select_literals")]
	if fn.Proc().Compiled == nil {
		t.Fatal("SQL parameterization did not compile")
	}
	query := "INSERT INTO test VALUES " + strings.Repeat("(12345),", 10000) + "(12345)"
	previousGCPercent := debug.SetGCPercent(1)
	defer debug.SetGCPercent(previousGCPercent)
	runtime.GC()
	got := Apply(fn, NewString(query))
	if got.GetTag() != tagSlice || len(got.Slice()) != 3 {
		t.Fatalf("unexpected parameterization result")
	}
	items := got.Slice()
	want := "INSERT INTO test VALUES " + strings.Repeat("(?),", 10000) + "(?)"
	if items[0].String() != want || len(items[1].Slice()) != 10001 {
		t.Fatal("parameterization changed SQL shape or literal bindings")
	}
	runtime.KeepAlive(got)
	runtime.KeepAlive(query)
}
