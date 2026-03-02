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
package main

import "golang.org/x/tools/go/ssa"

// emitInstr routes each SSA instruction to a dedicated emitter method.
// Phase 1 keeps lowering behavior stable by delegating each typed emitter
// to the existing legacy lowering body.
func (g *codeGen) emitInstr(instr ssa.Instruction) {
	em := perSSAInstrEmitter{g: g}
	em.Emit(instr)
}

type perSSAInstrEmitter struct {
	g *codeGen
}

func (e perSSAInstrEmitter) Emit(instr ssa.Instruction) {
	switch v := instr.(type) {
	case *ssa.IndexAddr:
		e.EmitIndexAddr(v)
	case *ssa.FieldAddr:
		e.EmitFieldAddr(v)
	case *ssa.UnOp:
		e.EmitUnOp(v)
	case *ssa.Call:
		e.EmitCall(v)
	case *ssa.BinOp:
		e.EmitBinOp(v)
	case *ssa.Return:
		e.EmitReturn(v)
	case *ssa.Phi:
		e.EmitPhi(v)
	case *ssa.If:
		e.EmitIf(v)
	case *ssa.Jump:
		e.EmitJump(v)
	case *ssa.Convert:
		e.EmitConvert(v)
	case *ssa.Alloc:
		e.EmitAlloc(v)
	case *ssa.Store:
		e.EmitStore(v)
	case *ssa.MakeClosure:
		e.EmitMakeClosure(v)
	case *ssa.MakeInterface:
		e.EmitMakeInterface(v)
	case *ssa.Panic:
		e.EmitPanic(v)
	case *ssa.Slice:
		e.EmitSlice(v)
	default:
		panic(instrDesc(instr))
	}
}

func (e perSSAInstrEmitter) EmitIndexAddr(v *ssa.IndexAddr)         { e.g.emitInstrLegacy(v) }
func (e perSSAInstrEmitter) EmitFieldAddr(v *ssa.FieldAddr)         { e.g.emitInstrLegacy(v) }
func (e perSSAInstrEmitter) EmitUnOp(v *ssa.UnOp)                   { e.g.emitInstrLegacy(v) }
func (e perSSAInstrEmitter) EmitCall(v *ssa.Call)                   { e.g.emitInstrLegacy(v) }
func (e perSSAInstrEmitter) EmitBinOp(v *ssa.BinOp)                 { e.g.emitInstrLegacy(v) }
func (e perSSAInstrEmitter) EmitReturn(v *ssa.Return)               { e.g.emitInstrLegacy(v) }
func (e perSSAInstrEmitter) EmitPhi(v *ssa.Phi)                     { e.g.emitInstrLegacy(v) }
func (e perSSAInstrEmitter) EmitIf(v *ssa.If)                       { e.g.emitInstrLegacy(v) }
func (e perSSAInstrEmitter) EmitJump(v *ssa.Jump)                   { e.g.emitInstrLegacy(v) }
func (e perSSAInstrEmitter) EmitConvert(v *ssa.Convert)             { e.g.emitInstrLegacy(v) }
func (e perSSAInstrEmitter) EmitAlloc(v *ssa.Alloc)                 { e.g.emitInstrLegacy(v) }
func (e perSSAInstrEmitter) EmitStore(v *ssa.Store)                 { e.g.emitInstrLegacy(v) }
func (e perSSAInstrEmitter) EmitMakeClosure(v *ssa.MakeClosure)     { e.g.emitInstrLegacy(v) }
func (e perSSAInstrEmitter) EmitMakeInterface(v *ssa.MakeInterface) { e.g.emitInstrLegacy(v) }
func (e perSSAInstrEmitter) EmitPanic(v *ssa.Panic)                 { e.g.emitInstrLegacy(v) }
func (e perSSAInstrEmitter) EmitSlice(v *ssa.Slice)                 { e.g.emitInstrLegacy(v) }
