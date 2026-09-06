/*
Copyright (C) 2026  Carl-Philip Hänsch

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

import "regexp"

// jitEmitConstantRegexpReplaceFunc lowers the hidden
// jit-constant-regexp-replace-func declaration - a `(regexp_replace s "<const>"
// f)` whose pattern is constant and whose replacement is a function.
//
// Target lowering: an inline byte walk over s - for every match of the constant
// pattern, copy the gap, run the (JIT-compiled) replacement on the matched
// slice, append its bytes to a grow-on-demand buffer (Go's append fast path
// inlined, runtime.growslice only on overflow); zero matches return s untouched
// with no allocation.
//
// v1 keeps a single native call boundary while the byte-buffer primitives and
// the resident regex scan loop are built; correctness and the optimizer /
// grammar composition land first.
func jitEmitConstantRegexpReplaceFunc(ctx *JITContext, pattern *regexp.Regexp, replacement Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
	reArg := jitCopyScmerToPair(ctx, JITValueDesc{Loc: LocImm, Type: tagRegex, Imm: NewRegex(pattern)})
	ctx.TrackImm(NewRegex(pattern))
	replArg := jitCopyScmerToPair(ctx, JITValueDesc{Loc: LocImm, Type: replacement.GetTag(), Imm: replacement})
	ctx.TrackImm(replacement)
	value := args[2]
	ctx.EnsureDesc(&value)
	out := ctx.EmitGoCallScalar(GoFuncAddr(jitConstantRegexpReplaceFunc), []JITValueDesc{reArg, replArg, value}, 2)
	out.Type = JITTypeUnknown
	out.Rooted = true
	ctx.FreeDesc(&reArg)
	ctx.FreeDesc(&replArg)
	return jitPlaceScmerIntoTarget(ctx, out, result)
}
