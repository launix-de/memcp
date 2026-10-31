/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/
package scm

import "fmt"
import "maps"
import "bytes"
import "encoding/json"

const procedureWireVersion = 2
const maxProcedureWireBytes = 256 * 1024

// MarshalProcedure closes explicit captures through the existing serializer.
// SerializationContext exports supported runtime values as typed owner references;
// invocation views are prepared before validation and reconstruction happens
// before execution. Environments, native/JIT code and Go pointers never enter
// the wire. Both nodes must provide the same declared builtin contract. Unknown
// runtime values and unresolved captures fail explicitly instead of silently
// binding a different receiver-side global.
func MarshalProcedure(value Scmer, bindings ...*SerializationContext) (data []byte, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("procedure encoding: %v", failure)
			data = nil
		}
	}()
	if !value.IsProc() && value.IsNativeFunc() {
		if _, ok := ClusterObjectOf(value); ok || len(bindings) > 0 && bindings[0] != nil && bindings[0].ExportHandle != nil && DeclarationForValue(value) == nil {
			// Runtime callables keep their native local ABI. The normal serializer
			// reconstructs their portable value/peer handle once, before invocation.
			value = Eval(NewSlice([]Scmer{NewSymbol("lambda"), NewSymbol("args"), NewSlice([]Scmer{NewSymbol("apply"), value, NewSymbol("args")})}), &Globalenv)
		}
	}
	if !value.IsProc() {
		declaration := DeclarationForValue(value)
		if declaration == nil || value.IsSymbol() {
			return nil, fmt.Errorf("procedure encoding requires a Scheme procedure or declared builtin (tag %d)", value.GetTag())
		}
		value = Eval(NewSlice([]Scmer{NewSymbol("lambda"), NewSymbol("args"), NewSlice([]Scmer{NewSymbol("apply"), NewSymbol(declaration.Name), NewSymbol("args")})}), &Globalenv)
	}
	if len(bindings) == 0 || bindings[0] == nil {
		bindings = []*SerializationContext{new(SerializationContext)}
	}
	closed := CloseProcedure(value)
	bindings[0].prepare(closed, false)
	proc := closed.Proc()
	bound := procedureWireBindings(proc.Params, proc.Body, nil)
	if err := validateProcedureWire(proc.Body, bound, value.Proc().En, false, 0, bindings...); err != nil {
		return nil, err
	}
	var source bytes.Buffer
	Serialize(&source, closed, &Globalenv, bindings...)
	data, err = json.Marshal(struct {
		Version int    `json:"version"`
		Source  string `json:"source"`
	}{procedureWireVersion, source.String()})
	if len(data) > maxProcedureWireBytes {
		return nil, fmt.Errorf("procedure exceeds wire size limit")
	}
	return data, err
}

// UnmarshalProcedure MUST evaluate serialized capture constructors once on the
// receiving peer and restore their enclosed scope before returning the lambda.
// Ordinary row/query code stays inert until its authorized invocation. Imported
// objects must already exist in Go RAM even when that invocation has zero rows;
// otherwise no GC cleanup can release the sender's registration. The codec is
// for authenticated peers, not a sandbox or an authorization boundary.
func UnmarshalProcedure(data []byte, options ...ProcedureDecode) (value Scmer, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			value = NewNil()
			err = fmt.Errorf("procedure decoding: %v", failure)
		}
	}()
	if len(data) > maxProcedureWireBytes {
		return NewNil(), fmt.Errorf("procedure exceeds wire size limit")
	}
	var envelope struct {
		Version int    `json:"version"`
		Source  string `json:"source"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return NewNil(), err
	}
	if envelope.Version != procedureWireVersion {
		return NewNil(), fmt.Errorf("unsupported procedure envelope")
	}
	tokens := tokenize("remote procedure", envelope.Source)
	expression := readFrom(&tokens)
	items, ok := scmerAsSlice(expression)
	if !ok || (len(items) != 3 && len(items) != 4) || !scmerIsSymbol(items[0], "lambda") || len(tokens) != 0 {
		return NewNil(), fmt.Errorf("procedure envelope must contain exactly one lambda")
	}
	params := items[1].WithoutSourceInfo()
	if !params.IsSymbol() {
		if !params.IsSlice() {
			return NewNil(), fmt.Errorf("invalid procedure parameters")
		}
		for _, param := range params.Slice() {
			if !param.WithoutSourceInfo().IsSymbol() {
				return NewNil(), fmt.Errorf("invalid procedure parameter")
			}
		}
	}
	if len(items) == 4 {
		count := items[3].WithoutSourceInfo()
		if !count.IsInt() || count.Int() < 0 || count.Int() > 65536 {
			return NewNil(), fmt.Errorf("invalid procedure frame size")
		}
	}
	if err := validateProcedureWire(items[2], procedureWireBindings(params, items[2], nil), &Globalenv, false, 0); err != nil {
		return NewNil(), err
	}
	if len(options) > 0 && options[0].Prepare != nil {
		expression, err = options[0].Prepare(expression)
		if err != nil {
			return NewNil(), err
		}
	}
	// MUST import captures now, not on first row execution. Empty shards and
	// never-called callbacks still need real peer handles so GC can release the
	// owner's reservations. One invocation owns the reconstructed lambda/scope;
	// row loops neither resolve URIs nor register/deregister references.
	var resolve func(ClusterReference) (Scmer, error)
	var resolveHandle func(string, string, uint64) (Handle, error)
	if len(options) > 0 {
		resolve = options[0].Resolve
		resolveHandle = options[0].ResolveHandle
	}
	expression, err = ResolveSerializedObjects(expression, resolve, resolveHandle)
	if err != nil {
		return NewNil(), err
	}
	// Capture constructors have run exactly once; now build the callable and
	// enclosed scope. Ordinary query code remains inert until invocation.
	return Eval(Optimize(expression, &Globalenv, nil), &Globalenv), nil
}

func procedureWireBindings(params, body Scmer, parent map[Symbol]struct{}) map[Symbol]struct{} {
	params = params.WithoutSourceInfo()
	bound := maps.Clone(parent)
	if bound == nil {
		bound = make(map[Symbol]struct{})
	}
	if params.IsSymbol() {
		bound[params.Symbol()] = struct{}{}
	} else if params.IsSlice() {
		for _, param := range params.Slice() {
			param = param.WithoutSourceInfo()
			if param.IsSymbol() {
				bound[param.Symbol()] = struct{}{}
			}
		}
	}
	collectProcedureBindings(body, bound)
	return bound
}

func validateProcedureWire(value Scmer, bound map[Symbol]struct{}, origin *Env, literal bool, depth int, bindings ...*SerializationContext) error {
	if depth > 128 {
		return fmt.Errorf("procedure exceeds wire nesting limit")
	}
	value = value.WithoutSourceInfo()
	if _, ok := ClusterObjectOf(value); ok {
		return nil
	}
	switch value.GetTag() {
	case tagNil, tagBool, tagInt, tagFloat, tagDate, tagString, tagCString, tagBString, tagLZ4, tagNthLocalVar:
		return nil
	case tagSymbol:
		if !literal {
			if _, local := bound[value.Symbol()]; !local {
				if origin != nil {
					binding := origin.FindRead(value.Symbol())
					if binding != &Globalenv {
						if _, present := binding.Vars[value.Symbol()]; present {
							return fmt.Errorf("unclosed lexical capture %s", value.Symbol())
						}
					}
				}
				if DeclarationForValue(value) == nil {
					switch value.String() {
					case "outer", "var", "begin", "if", "match", "and", "or", "nil", "true", "false":
					default:
						return fmt.Errorf("procedure requires an explicit binding for %s", value.Symbol())
					}
				}
			}
		}
		return nil
	case tagProc:
		closed := CloseProcedure(value)
		proc := closed.Proc()
		return validateProcedureWire(proc.Body, procedureWireBindings(proc.Params, proc.Body, nil), value.Proc().En, false, depth+1, bindings...)
	case tagFunc:
		if DeclarationForValue(value) == nil && len(bindings) > 0 && bindings[0] != nil && bindings[0].ExportHandle != nil {
			return nil
		}
		fallthrough
	case tagSpecialForm:
		if literal || DeclarationForValue(value) == nil {
			return fmt.Errorf("procedure contains a process-local callable")
		}
		return nil
	case tagFastDict:
		for _, item := range value.FastDict().Pairs {
			if err := validateProcedureWire(item, bound, origin, true, depth+1, bindings...); err != nil {
				return err
			}
		}
		return nil
	case tagSlice:
		items := value.Slice()
		if !literal {
			items = lambdaSyntaxOperands(items)
		}
		if !literal && len(items) > 0 {
			head, _ := symbolName(items[0])
			switch head {
			case "quote":
				if len(items) != 2 {
					return fmt.Errorf("invalid quoted procedure value")
				}
				return validateProcedureWire(items[1], bound, origin, true, depth+1, bindings...)
			case "lambda":
				if len(items) != 3 && len(items) != 4 {
					return fmt.Errorf("invalid nested procedure")
				}
				return validateProcedureWire(items[2], procedureWireBindings(items[1], items[2], bound), origin, false, depth+1, bindings...)
			}
		}
		for _, item := range items {
			if err := validateProcedureWire(item, bound, origin, literal, depth+1, bindings...); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("procedure contains unsupported runtime value (tag %d, %T)", value.GetTag(), value.Any())
	}
}
