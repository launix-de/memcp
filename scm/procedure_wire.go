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
import "strconv"
import "strings"

const procedureWireVersion = 3
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
	proc := closed.Proc()
	bound := procedureWireBindings(proc.Params, proc.Body, nil)
	if err := validateProcedureWire(proc.Body, bound, value.Proc().En, false, 0, bindings...); err != nil {
		return nil, err
	}
	var source bytes.Buffer
	source.WriteByte(procedureWireVersion)
	Serialize(&source, closed, &Globalenv, bindings...)
	data = source.Bytes()
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
	if len(data) == 0 || data[0] != procedureWireVersion {
		return NewNil(), fmt.Errorf("unsupported procedure envelope")
	}
	source := string(data[1:])
	tokens := tokenize("remote procedure", source, false)
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
	var resolveHandle func(uint64) (Handle, error)
	if len(options) > 0 {
		resolve = options[0].Resolve
		resolveHandle = options[0].ResolveHandle
	}
	expression, err = ResolveSerializedObjects(expression, resolve, resolveHandle)
	if err != nil {
		return NewNil(), err
	}
	// Capture constructors have run exactly once; now build the callable and
	// enclosed scope. The sender already optimized the plan; reconstruction
	// must not optimize it again. Ordinary query code stays inert until invocation.
	if len(options) > 0 && options[0].Cache != nil && jitEnabled {
		return compileProcedureFactory(expression, options[0].Cache), nil
	}
	return CompileJIT(Eval(expression, &Globalenv), true), nil
}

// Compile a pure factory, not a bound callback. Its runtime arguments become
// ordinary native closure captures. The existing lambda compiler therefore
// owns binding, GC roots and source reconstruction on a later delegation.
// Cache ownership/eviction belongs to the embedding's existing memory budget.
func compileProcedureFactory(expression Scmer, cache func(string, func() Scmer) Scmer) Scmer {
	used := make(map[Symbol]bool)
	var symbols func(Scmer)
	symbols = func(value Scmer) {
		if value.IsSymbol() {
			used[value.Symbol()] = true
		} else if value.IsSlice() {
			for _, item := range value.Slice() {
				symbols(item)
			}
		}
	}
	symbols(expression)
	var params, captures []Scmer
	var runtimeData func(Scmer) bool
	runtimeData = func(value Scmer) bool {
		if value.GetTag() >= 100 {
			return true
		}
		if _, object := ClusterObjectOf(value); object {
			return true
		}
		switch value.GetTag() {
		case tagAny, tagFastDict, tagProc, tagFuncEnv, tagClosure, tagJIT, tagParser, tagPromise, tagVector, tagBSON, tagCString, tagBString, tagLZ4:
			return true
		case tagFunc:
			return DeclarationForValue(value) == nil
		case tagSlice:
			for _, item := range value.Slice() {
				if runtimeData(item) {
					return true
				}
			}
		}
		return false
	}
	bind := func(value Scmer) Scmer {
		name := Symbol("\x00peer-capture:" + strconv.Itoa(len(captures)))
		for used[name] {
			name += ":"
		}
		params = append(params, NewSymbol(string(name)))
		captures = append(captures, value)
		return params[len(params)-1]
	}
	var factor func(Scmer) Scmer
	factor = func(value Scmer) Scmer {
		plain := value.WithoutSourceInfo()
		if _, object := ClusterObjectOf(plain); object {
			return bind(plain)
		}
		if plain.GetTag() >= 100 {
			return bind(plain)
		}
		switch plain.GetTag() {
		case tagAny, tagFastDict, tagProc, tagFuncEnv, tagClosure, tagJIT, tagParser, tagPromise, tagVector, tagBSON, tagCString, tagBString, tagLZ4:
			return bind(plain)
		case tagFunc:
			if DeclarationForValue(plain) == nil {
				return bind(plain)
			}
		}
		if !plain.IsSlice() {
			return value
		}
		items := plain.Slice()
		if len(items) == 2 && scmerIsSymbol(items[0], "quote") {
			// Quoted runtime data may enclose arbitrary native captures. Supplying
			// the whole value prevents a cached tree from retaining any of them.
			if runtimeData(items[1]) {
				return bind(items[1])
			}
			return value
		}
		var result []Scmer
		for i, item := range items {
			next := factor(item)
			if next != item && result == nil {
				result = append([]Scmer(nil), items...)
			}
			if result != nil {
				result[i] = next
			}
		}
		if result == nil {
			return value
		}
		return NewSlice(result)
	}
	body := factor(expression)
	factoryExpression := body
	if len(captures) > 0 {
		factoryExpression = NewSlice([]Scmer{NewSymbol("lambda"), NewSlice(params), body})
	}
	var source bytes.Buffer
	// Preserve numbered operands in cache identity too; diagnostic printing may
	// rename them to parameter symbols and is not an IR identity function.
	Serialize(&source, factoryExpression, &Globalenv, new(SerializationContext))
	factory := cache(source.String(), func() Scmer { return CompileJIT(Eval(ownProcedureCode(factoryExpression), &Globalenv), true) })
	if len(captures) == 0 {
		return factory
	}
	return Apply(factory, captures...)
}

// Cache misses detach code tokens from the incoming buffer. Small substrings
// must not keep a complete packet (including invocation data) alive in a code
// cache. Hits neither copy nor retain the newly received syntax tree.
func ownProcedureCode(value Scmer) Scmer {
	switch value.GetTag() {
	case tagString:
		return NewString(strings.Clone(value.String()))
	case tagSymbol:
		return NewSymbol(strings.Clone(value.String()))
	case tagSlice:
		items := value.Slice()
		owned := make([]Scmer, len(items))
		for i, item := range items {
			owned[i] = ownProcedureCode(item)
		}
		return NewSlice(owned)
	default:
		return value
	}
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

// procedurePatternBindings records only captures. Pattern operators and symbol
// literals are not lexical reads; eval is the sole executable pattern operand.
func procedurePatternBindings(pattern Scmer, bound map[Symbol]struct{}) {
	pattern = pattern.WithoutSourceInfo()
	if pattern.IsSymbol() {
		switch pattern.Symbol() {
		case "_", "nil", "true", "false":
			return
		}
		bound[pattern.Symbol()] = struct{}{}
		return
	}
	if !pattern.IsSlice() {
		return
	}
	items := pattern.Slice()
	if len(items) == 0 {
		return
	}
	start := 0
	if head, ok := symbolName(items[0]); ok {
		switch head {
		case "quote", "symbol", "eval", "var":
			return
		case "regex":
			if len(items) < 2 {
				return
			}
			start = 2
		default:
			start = 1
		}
	}
	for _, item := range items[start:] {
		procedurePatternBindings(item, bound)
	}
}

func closeProcedurePattern(pattern Scmer, callFrame *Env, scopes []map[Symbol]struct{}) Scmer {
	value := pattern.WithoutSourceInfo()
	if !value.IsSlice() {
		return pattern
	}
	items := value.Slice()
	var closed []Scmer
	replace := func(index int, value Scmer) {
		if value == items[index] {
			return
		}
		if closed == nil {
			closed = append([]Scmer(nil), items...)
		}
		closed[index] = value
	}
	if len(items) == 2 && scmerIsSymbol(items[0], "eval") {
		replace(1, closeProcedureScopeCaptures(items[1], callFrame, scopes))
	} else {
		for i, item := range items {
			replace(i, closeProcedurePattern(item, callFrame, scopes))
		}
	}
	if closed == nil {
		return pattern
	}
	return NewSlice(closed)
}

func validateProcedurePattern(pattern Scmer, bound map[Symbol]struct{}, origin *Env, depth int, bindings ...*SerializationContext) error {
	if pattern.WithoutSourceInfo().IsRegex() {
		return nil
	}
	if depth > 128 {
		return fmt.Errorf("procedure pattern exceeds wire nesting limit")
	}
	value := pattern.WithoutSourceInfo()
	if value.IsSlice() {
		items := value.Slice()
		if len(items) == 2 && scmerIsSymbol(items[0], "eval") {
			return validateProcedureWire(items[1], bound, origin, false, depth+1, bindings...)
		}
		for _, item := range items {
			if err := validateProcedurePattern(item, bound, origin, depth+1, bindings...); err != nil {
				return err
			}
		}
		return nil
	}
	return validateProcedureWire(value, bound, origin, true, depth+1, bindings...)
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
		if !literal && len(items) > 0 {
			if declaration := DeclarationForValue(items[0]); declaration != nil && declaration.SerializeCall != nil {
				source, err := declaration.SerializeCall(items[1:])
				if err != nil {
					return err
				}
				return validateProcedureWire(source, bound, origin, false, depth+1, bindings...)
			}
		}
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
			case "match", "match_mut":
				if len(items) < 2 {
					return fmt.Errorf("invalid match procedure value")
				}
				if err := validateProcedureWire(items[1], bound, origin, false, depth+1, bindings...); err != nil {
					return err
				}
				i := 2
				for ; i+1 < len(items); i += 2 {
					if err := validateProcedurePattern(items[i], bound, origin, depth+1, bindings...); err != nil {
						return err
					}
					branch := maps.Clone(bound)
					procedurePatternBindings(items[i], branch)
					collectProcedureBindings(items[i+1], branch)
					if err := validateProcedureWire(items[i+1], branch, origin, false, depth+1, bindings...); err != nil {
						return err
					}
				}
				if i < len(items) {
					return validateProcedureWire(items[i], bound, origin, false, depth+1, bindings...)
				}
				return nil
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
