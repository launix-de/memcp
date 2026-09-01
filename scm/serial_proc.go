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

import "reflect"

// SerialProcKind describes callback shapes whose semantics can be consumed by
// a physical operator without entering Eval. Operators must dispatch on Kind
// outside their row loops; Call is the compatibility path for code which does
// not have a shape-specific kernel.
type SerialProcKind uint8

const (
	SerialProcGeneral SerialProcKind = iota
	SerialProcConstant
	SerialProcArgument
	SerialProcNative
	SerialProcNativeArgConstant
)

// SerialProc exposes trivial executable shapes to physical operators. Callers
// retain the original procedure when analysis or serialization needs it; not
// duplicating it here keeps every scan mapper compact. General programs own a
// mutable environment and call-frame stack; prepare one per serial worker and
// do not copy or share it after execution starts.
type SerialProc struct {
	Kind     SerialProcKind
	Value    Scmer // constant result or native-function identity
	Argument int16
	// ConstantFirst records whether Value is the first operand of the
	// binary native call represented by SerialProcNativeArgConstant.
	ConstantFirst bool
	Function      func(...Scmer) Scmer
	borrowed      func([]Scmer) Scmer
}

func serialProcBody(v Scmer) Scmer {
	if stripped, ok := scmerStripSourceInfo(v); ok {
		return stripped
	}
	return v
}

func serialProcConstantBody(v Scmer) bool {
	switch v.GetTag() {
	case tagNil, tagBool, tagInt, tagFloat, tagString:
		return true
	default:
		return false
	}
}

func serialProcArgumentIndex(proc *Proc, body Scmer) (int, bool) {
	body = serialProcBody(body)
	if body.IsNthLocalVar() {
		return int(body.NthLocalVar()), true
	}
	params := serialProcBody(proc.Params)
	if !body.IsSymbol() || !params.IsSlice() {
		return 0, false
	}
	for i, param := range params.Slice() {
		param = serialProcBody(param)
		if param.IsSymbol() && mustSymbol(param) == mustSymbol(body) {
			return i, true
		}
	}
	return 0, false
}

func serialProcResolveNative(proc *Proc, value Scmer) (Scmer, bool) {
	value = serialProcBody(value)
	if value.GetTag() == tagFunc {
		return value, true
	}
	if !value.IsSymbol() {
		return NewNil(), false
	}
	environment := proc.En
	if environment == nil {
		environment = &Globalenv
	}
	binding := environment.FindRead(mustSymbol(value))
	if binding == nil {
		return NewNil(), false
	}
	resolved, ok := binding.Vars[mustSymbol(value)]
	return resolved, ok && resolved.GetTag() == tagFunc
}

func serialProcNativeForward(proc *Proc, body Scmer) (Scmer, bool) {
	body = serialProcBody(body)
	if !body.IsSlice() {
		return NewNil(), false
	}
	call := body.Slice()
	params := serialProcBody(proc.Params)
	if !params.IsSlice() || len(call) != len(params.Slice())+1 {
		return NewNil(), false
	}
	for i, operand := range call[1:] {
		argument, ok := serialProcArgumentIndex(proc, operand)
		if !ok || argument != i {
			return NewNil(), false
		}
	}
	native, ok := serialProcResolveNative(proc, call[0])
	if !ok {
		return NewNil(), false
	}
	// Operators reuse their argument buffers. A native function which retains
	// that variadic slice (not merely its values) must keep the adapter-created
	// fresh call frame or every retained result would alias the next row.
	declaration := DeclarationForValue(native)
	if declaration == nil || declaration.RetainsCallArgs {
		return NewNil(), false
	}
	return native, true
}

func serialProcLiteral(value Scmer) (Scmer, bool) {
	value = serialProcBody(value)
	if serialProcConstantBody(value) {
		return value, true
	}
	if value.SymbolEquals("true") || value.SymbolEquals("false") {
		return Globalenv.Vars[mustSymbol(value)], true
	}
	return NewNil(), false
}

func serialProcNativeArgConstant(proc *Proc, body Scmer) (native Scmer, argument int, constant Scmer, constantFirst bool, ok bool) {
	body = serialProcBody(body)
	if !body.IsSlice() || len(body.Slice()) != 3 {
		return NewNil(), 0, NewNil(), false, false
	}
	call := body.Slice()
	native, ok = serialProcResolveNative(proc, call[0])
	if !ok {
		return NewNil(), 0, NewNil(), false, false
	}
	declaration := DeclarationForValue(native)
	if declaration == nil || declaration.RetainsCallArgs {
		return NewNil(), 0, NewNil(), false, false
	}
	if argument, ok = serialProcArgumentIndex(proc, call[1]); ok {
		if constant, ok = serialProcLiteral(call[2]); ok {
			return native, argument, constant, false, true
		}
	}
	if argument, ok = serialProcArgumentIndex(proc, call[2]); ok {
		if constant, ok = serialProcLiteral(call[1]); ok {
			return native, argument, constant, true, true
		}
	}
	return NewNil(), 0, NewNil(), false, false
}

// PrepareSerialProc classifies the dominant constant, argument-projection and
// exact native-forwarding callback shapes. More complex procedures retain the
// existing interpreter adapter until the fused scan JIT owns them.
func PrepareSerialProc(source Scmer) SerialProc {
	prepared := SerialProc{Argument: -1}
	if source.IsNil() {
		prepared.Kind = SerialProcConstant
		prepared.Value = NewNil()
		return prepared
	}
	if source.GetTag() == tagFunc {
		prepared.Kind = SerialProcNative
		prepared.Function = source.Func()
		prepared.Value = source
		return prepared
	}
	if source.GetTag() == tagAny {
		if fn, ok := source.Any().(func(...Scmer) Scmer); ok {
			prepared.Kind = SerialProcNative
			prepared.Function = fn
			prepared.Value = source
			return prepared
		}
	}
	if source.GetTag() != tagProc {
		prepared.Kind = SerialProcConstant
		prepared.Value = source
		return prepared
	}

	proc := source.Proc()
	// Compiled is the authoritative implementation of a Proc. Its retained
	// source body is diagnostic input, not necessarily an executable equivalent;
	// classifying that body could silently bypass code generation semantics.
	if proc.Compiled != nil {
		prepared.Kind = SerialProcGeneral
		prepared.borrowed = optimizeProcToSerialBorrowed(source)
		return prepared
	}
	body := serialProcBody(proc.Body)
	if serialProcConstantBody(body) {
		prepared.Kind = SerialProcConstant
		prepared.Value = body
		return prepared
	}
	if body.SymbolEquals("true") || body.SymbolEquals("false") {
		prepared.Kind = SerialProcConstant
		prepared.Value = Globalenv.Vars[mustSymbol(body)]
		return prepared
	}
	if argument, ok := serialProcArgumentIndex(proc, body); ok && argument <= 32767 {
		prepared.Kind = SerialProcArgument
		prepared.Argument = int16(argument)
		return prepared
	}
	if native, ok := serialProcNativeForward(proc, body); ok {
		prepared.Kind = SerialProcNative
		fn := native.Func()
		// Eval strips source annotations while resolving lambda arguments. Keep
		// exact forwarding equivalent for syntactic values such as a quoted
		// neutral list without allocating a fresh argument slice per call.
		prepared.Function = func(args ...Scmer) Scmer {
			for i, arg := range args {
				if stripped, ok := scmerStripSourceInfo(arg); ok {
					args[i] = stripped
				}
			}
			return fn(args...)
		}
		prepared.Value = native
		return prepared
	}
	if native, argument, constant, constantFirst, ok := serialProcNativeArgConstant(proc, body); ok && argument <= 32767 {
		prepared.Kind = SerialProcNativeArgConstant
		prepared.Function = native.Func()
		prepared.Argument = int16(argument)
		prepared.Value = constant
		prepared.ConstantFirst = constantFirst
		return prepared
	}

	prepared.Kind = SerialProcGeneral
	prepared.borrowed = optimizeProcToSerialBorrowed(source)
	return prepared
}

// Call evaluates a prepared callback with a caller-owned argument frame. Hot
// physical loops should dispatch dominant simple Kinds once; compound programs
// use Call so the prepared expression can reuse its nested native-call frames.
func (p *SerialProc) Call(args []Scmer) Scmer {
	switch p.Kind {
	case SerialProcConstant:
		return p.Value
	case SerialProcArgument:
		return args[int(p.Argument)]
	case SerialProcNative:
		return p.Function(args...)
	case SerialProcNativeArgConstant:
		var call [2]Scmer
		if p.ConstantFirst {
			call[0], call[1] = p.Value, args[int(p.Argument)]
		} else {
			call[0], call[1] = args[int(p.Argument)], p.Value
		}
		return p.Function(call[:]...)
	default:
		return p.borrowed(args)
	}
}

// IsNative reports whether the prepared callback is exactly the named global
// native implementation. It is used only for algebraically dominant kernels
// such as constant-one reduction by +.
func (p *SerialProc) IsNative(name Symbol) bool {
	if p.Kind != SerialProcNative {
		return false
	}
	binding := Globalenv.FindRead(name)
	if binding == nil {
		return false
	}
	value, ok := binding.Vars[name]
	return ok && value.GetTag() == tagFunc && value.ptr == p.Value.ptr && value.aux == p.Value.aux
}

// IsNativeArgConstant reports whether this callback is a binary native call
// with one procedure argument and one constant operand.
func (p *SerialProc) IsNativeArgConstant(name Symbol) bool {
	if p.Kind != SerialProcNativeArgConstant || p.Function == nil {
		return false
	}
	binding := Globalenv.FindRead(name)
	if binding == nil {
		return false
	}
	value, ok := binding.Vars[name]
	return ok && value.GetTag() == tagFunc && reflect.ValueOf(value.Func()).Pointer() == reflect.ValueOf(p.Function).Pointer()
}
