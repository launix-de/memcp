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
import "encoding/json"

const procedureWireVersion = 1
const maxProcedureWireBytes = 256 * 1024

// MarshalProcedure snapshots explicit captures before using the ordinary Scmer
// AST codec. It does not serialize Env, native/JIT code, sessions or Go pointers.
// Both nodes must provide the same global builtin contract. Unsupported values
// and unresolved lexical captures fail instead of silently becoming "?" or a
// different receiver-side global. This is a conservative first transport subset.
func MarshalProcedure(value Scmer) (data []byte, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("procedure encoding: %v", failure)
			data = nil
		}
	}()
	if !value.IsProc() {
		return nil, fmt.Errorf("procedure encoding requires a Scheme procedure")
	}
	closed := CloseProcedure(value)
	proc := closed.Proc()
	bound := procedureWireBindings(proc.Params, proc.Body, nil)
	if err := validateProcedureWire(proc.Body, bound, value.Proc().En, false, 0); err != nil {
		return nil, err
	}
	data, err = json.Marshal(struct {
		Version int   `json:"version"`
		Code    Scmer `json:"code"`
	}{procedureWireVersion, closed})
	if len(data) > maxProcedureWireBytes {
		return nil, fmt.Errorf("procedure exceeds wire size limit")
	}
	return data, err
}

// UnmarshalProcedure reconstructs an uncompiled procedure without evaluating
// its body. Only an authenticated task handler may later execute it. The codec
// is not a sandbox or an authorization boundary.
func UnmarshalProcedure(data []byte) (Scmer, error) {
	if len(data) > maxProcedureWireBytes {
		return NewNil(), fmt.Errorf("procedure exceeds wire size limit")
	}
	var envelope struct {
		Version int   `json:"version"`
		Code    Scmer `json:"code"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return NewNil(), err
	}
	if envelope.Version != procedureWireVersion || !envelope.Code.IsProc() {
		return NewNil(), fmt.Errorf("unsupported procedure envelope")
	}
	proc := envelope.Code.Proc()
	if err := validateProcedureWire(proc.Body, procedureWireBindings(proc.Params, proc.Body, nil), &Globalenv, false, 0); err != nil {
		return NewNil(), err
	}
	return envelope.Code, nil
}

func procedureWireBindings(params, body Scmer, parent map[Symbol]struct{}) map[Symbol]struct{} {
	bound := maps.Clone(parent)
	if bound == nil {
		bound = make(map[Symbol]struct{})
	}
	if params.IsSymbol() {
		bound[params.Symbol()] = struct{}{}
	} else if params.IsSlice() {
		for _, param := range params.Slice() {
			if param.IsSymbol() {
				bound[param.Symbol()] = struct{}{}
			}
		}
	}
	collectProcedureBindings(body, bound)
	return bound
}

func validateProcedureWire(value Scmer, bound map[Symbol]struct{}, origin *Env, literal bool, depth int) error {
	if depth > 128 {
		return fmt.Errorf("procedure exceeds wire nesting limit")
	}
	value = value.WithoutSourceInfo()
	switch value.GetTag() {
	case tagNil, tagBool, tagInt, tagFloat, tagString, tagCString, tagBString, tagLZ4, tagNthLocalVar:
		return nil
	case tagSymbol:
		if !literal {
			if _, local := bound[value.Symbol()]; !local && origin != nil {
				binding := origin.FindRead(value.Symbol())
				if binding != &Globalenv {
					if _, present := binding.Vars[value.Symbol()]; present {
						return fmt.Errorf("unclosed lexical capture %s", value.Symbol())
					}
				}
			}
		}
		return nil
	case tagFunc, tagSpecialForm:
		if literal || DeclarationForValue(value) == nil {
			return fmt.Errorf("procedure contains a process-local callable")
		}
		return nil
	case tagSlice:
		items := value.Slice()
		if !literal {
			items = lambdaSyntaxOperands(items)
		}
		if !literal && len(items) > 0 && items[0].IsSymbol() {
			switch items[0].String() {
			case "quote":
				if len(items) != 2 {
					return fmt.Errorf("invalid quoted procedure value")
				}
				return validateProcedureWire(items[1], bound, origin, true, depth+1)
			case "lambda":
				if len(items) != 3 && len(items) != 4 {
					return fmt.Errorf("invalid nested procedure")
				}
				return validateProcedureWire(items[2], procedureWireBindings(items[1], items[2], bound), origin, false, depth+1)
			}
		}
		for _, item := range items {
			if err := validateProcedureWire(item, bound, origin, literal, depth+1); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("procedure contains unsupported runtime value (tag %d, %T)", value.GetTag(), value.Any())
	}
}
