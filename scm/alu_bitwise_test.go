/*
Copyright (C) 2026 Carl-Philip Hänsch

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
	"math"
	"testing"
)

func TestExactIntegerBitOperators(t *testing.T) {
	for _, test := range []struct {
		name string
		args []Scmer
		want int64
	}{
		{"bit_and", []Scmer{NewInt(219), NewInt(46)}, 10},
		{"bit_or", []Scmer{NewInt(219), NewInt(46)}, 255},
		{"bit_xor", []Scmer{NewInt(219), NewInt(46)}, 245},
		{"bit_xor", []Scmer{NewInt(math.MinInt64), NewInt(-1)}, math.MaxInt64},
		{"bit_not", []Scmer{NewInt(math.MinInt64)}, math.MaxInt64},
		{"bit_shift_left", []Scmer{NewInt(1), NewInt(63)}, math.MinInt64},
		{"bit_shift_right", []Scmer{NewInt(-1), NewInt(1)}, math.MaxInt64},
		{"bit_shift_left", []Scmer{NewInt(-1), NewInt(64)}, 0},
		{"bit_shift_right", []Scmer{NewInt(-1), NewInt(math.MaxInt64)}, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := declarations[test.name].Fn(test.args...)
			if !got.IsInt() || got.Int() != test.want {
				t.Fatalf("got %v, want integer %d", got, test.want)
			}
		})
	}
}

func TestIntegerBitOperatorEmptyAndInvalidInputs(t *testing.T) {
	for _, name := range []string{"bit_and", "bit_or", "bit_xor", "bit_not", "bit_shift_left", "bit_shift_right"} {
		fn := declarations[name].Fn
		for _, args := range [][]Scmer{{NewNil(), NewInt(1)}, {NewInt(1), NewNil()}} {
			if name == "bit_not" {
				args = []Scmer{NewNil()}
			}
			if got := fn(args...); !got.IsNil() {
				t.Fatalf("%s did not propagate nil: %v", name, got)
			}
		}
		for _, value := range []Scmer{NewFloat(1), NewString("1"), NewBool(true)} {
			t.Run(name, func(t *testing.T) {
				defer func() {
					if recover() == nil {
						t.Fatal("noninteger operand was accepted")
					}
				}()
				fn(value, NewInt(1))
			})
		}
	}
	for _, name := range []string{"bit_shift_left", "bit_shift_right"} {
		t.Run(name+" negative count", func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("negative count was accepted")
				}
			}()
			declarations[name].Fn(NewInt(1), NewInt(-1))
		})
	}
}

// Exercise the generated declaration, not a hand-written copy of XOR lowering.
// Two live outer temporaries compete with operand reloads in a small GPR bank.
func TestJITGeneratedIntegerXorUnderRegisterPressure(t *testing.T) {
	if !jitEnabled {
		t.Skip("requires native JIT execution")
	}
	const name = "jit_test_pressured_integer_xor"
	declaration := &Declaration{
		Name: name,
		Fn:   func(args ...Scmer) Scmer { return NewInt(args[0].Int() ^ args[1].Int()) },
		Type: &TypeDescriptor{Kind: "func", Params: []*TypeDescriptor{{Kind: "int"}, {Kind: "int"}}, Return: &TypeDescriptor{Kind: "int"},
			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				for i := range args {
					ctx.StabilizeDescAcrossNestedCall(&args[i])
					// The test declaration accepts integer inputs only.
					args[i].Type = tagInt
					args[i].NoHeapPointer = true
				}
				home := JITValueDesc{Loc: LocStackPair, Type: tagInt, StackOff: ctx.AllocSpill(16)}
				boundary := ctx.PreserveRegisters(JITRegisterBoundaryOptions{ReleaseHomes: true})
				oldAll, oldFP, oldFreeFP := ctx.AllRegs, ctx.AllFPRegs, ctx.FreeFPRegs
				mask := uint64(0)
				available := oldAll
				for i := 0; i < 5; i++ {
					bit := available & -available
					mask |= bit
					available &^= bit
				}
				ctx.AllRegs, ctx.FreeRegs = mask, mask
				ctx.AllFPRegs, ctx.FreeFPRegs = 0, 0
				for i := 0; i < 2; i++ {
					r := ctx.AllocReg()
					ctx.EmitMovRegImm64(r, uint64(17+i))
					value := JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r, NoHeapPointer: true}
					ctx.BindReg(r, &value)
				}
				before := ctx.Coverage.InlinedCalls
				out := declarations["bit_xor"].Type.JITEmit(ctx, nil, args, JITValueDesc{Loc: LocAny})
				if ctx.Coverage.InlinedCalls == before {
					panic("pressure fixture did not inline the generated integer XOR")
				}
				ctx.EmitCopyScmerToDesc(&home, &out)
				ctx.FreeDesc(&out)
				ctx.AllRegs, ctx.AllFPRegs, ctx.FreeFPRegs = oldAll, oldFP, oldFreeFP
				boundary.Restore(ctx)
				return jitPlaceScmerIntoTarget(ctx, home, result)
			}},
	}
	Declare(&Globalenv, declaration)
	defer func() {
		delete(Globalenv.Vars, Symbol(name))
		delete(declarations, name)
		delete(declarationsByFunction, FunctionIdentity(declaration.Fn))
	}()
	compiled := compileJITExpressionTestProc(t, `(lambda (a b) (jit_test_pressured_integer_xor a b))`)
	for _, values := range [][2]int64{{219, 46}, {-9223372036854775808, -1}, {219, 219}, {0, 46}} {
		if got := Apply(compiled, NewInt(values[0]), NewInt(values[1])).Int(); got != values[0]^values[1] {
			t.Errorf("generated XOR under pressure(%d,%d)=%d, want %d", values[0], values[1], got, values[0]^values[1])
		}
	}
}
