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

import "testing"

func jitFourScalarResults(seed uint32) (int64, bool, int64, int64) {
	return int64(seed) + 1, seed&1 != 0, int64(seed) + 3, int64(seed) + 5
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
