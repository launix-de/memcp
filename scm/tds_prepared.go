/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"fmt"
	"math"
	"strings"
)

const tdsPreparedLimit = 128
const tdsPreparedBytesLimit = 1 << 20

// Prepared statements belong exclusively to the connection's execution
// goroutine. They retain text and immutable declarations, never parameter
// values, transactions, invocation sessions, or storage references.
type tdsPreparedStatement struct {
	sql, declarations string
	parameters        []tdsDeclaration
	bytes             int
}

type tdsDeclaration struct {
	name string
	spec Scmer
}

// SQL declaration parsing and conversion are frontend recipes. The adapter
// validates only their immutable name/descriptor carrier and RPC framing.
func tdsDeclarations(database, text string, callback Scmer) (result []tdsDeclaration, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			result = nil
			err = fmt.Errorf("frontend declaration failure: %v", failure)
		}
	}()
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	if callback.IsNil() {
		return nil, fmt.Errorf("TDS declaration callback is unavailable")
	}
	recipe := Apply(callback, NewString(database), NewString(text))
	if !recipe.IsSlice() || len(recipe.Slice()) > 2100 {
		return nil, fmt.Errorf("invalid TDS declaration recipe")
	}
	declarations := make([]tdsDeclaration, 0, len(recipe.Slice()))
	seen := make(map[string]bool)
	for _, binding := range recipe.Slice() {
		if !binding.IsSlice() || len(binding.Slice()) != 2 || !binding.Slice()[0].IsString() {
			return nil, fmt.Errorf("invalid TDS declaration carrier")
		}
		name := binding.Slice()[0].String()
		if name == "" || seen[name] {
			return nil, fmt.Errorf("duplicate or empty TDS declaration binding")
		}
		seen[name] = true
		declarations = append(declarations, tdsDeclaration{name, binding.Slice()[1]})
	}
	return declarations, nil
}
func (c *tdsConnection) declarations(text string) ([]tdsDeclaration, error) {
	return tdsDeclarations(c.database, text, c.server.declarations)
}
func (c *tdsConnection) bindParameters(declarations []tdsDeclaration, arguments []tdsRPCParameter, wires []Scmer) map[string]Scmer {
	if len(declarations) != len(arguments) {
		panic("TDS RPC parameter count does not match declarations")
	}
	if !c.server.convertParameter.IsNil() && len(wires) != 0 && len(wires) != len(arguments) {
		panic("TDS parameter wire descriptor count mismatch")
	}
	if len(arguments) > 0 && c.server.convertParameter.IsNil() {
		panic("TDS parameter conversion callback is unavailable")
	}
	specs := make([]Scmer, 0, 2*len(declarations))
	for _, declaration := range declarations {
		specs = append(specs, NewString(declaration.name), declaration.spec)
	}
	params := make(map[string]Scmer, len(arguments))
	for i, argument := range arguments {
		if argument.flags != 0 {
			panic("TDS statement output parameters are unsupported")
		}
		wire := NewNil()
		if len(wires) > 0 {
			wire = wires[i]
		}
		// Names arrive as wire spelling. Frontend conversion returns the canonical
		// binding name together with its converted value, avoiding dialect casing.
		converted := Apply(c.server.convertParameter, argument.value, wire, declarations[i].spec, NewString(argument.name), NewString(declarations[i].name), NewSlice(specs))
		if !converted.IsSlice() || len(converted.Slice()) != 2 || !converted.Slice()[0].IsString() {
			panic("invalid TDS converted binding carrier")
		}
		name := converted.Slice()[0].String()
		found := false
		for _, declaration := range declarations {
			if declaration.name == name {
				found = true
				break
			}
		}
		if !found {
			panic("undeclared TDS RPC parameter")
		}
		if _, exists := params[name]; exists {
			panic("duplicate TDS RPC parameter")
		}
		params[name] = converted.Slice()[1]
	}
	return params
}
func tdsWireDescriptors(rpc tdsRPC, start int) []Scmer {
	if len(rpc.wireDescriptors) == 0 {
		return nil
	}
	if len(rpc.wireDescriptors) != len(rpc.arguments) || start > len(rpc.wireDescriptors) {
		panic("invalid TDS parameter descriptor carrier")
	}
	return rpc.wireDescriptors[start:]
}

func tdsRPCText(argument tdsRPCParameter, nullable bool) string {
	if argument.flags != 0 || (!argument.value.IsString() && !(nullable && argument.value.IsNil())) {
		panic("TDS RPC statement/declarations must be input text")
	}
	if argument.value.IsNil() {
		return ""
	}
	return argument.value.String()
}

func tdsRPCHandle(argument tdsRPCParameter) int32 {
	if argument.flags != 0 || !argument.value.IsInt() || argument.value.Int() <= 0 || argument.value.Int() > math.MaxInt32 {
		panic("invalid TDS prepared handle")
	}
	return int32(argument.value.Int())
}

func (c *tdsConnection) removePrepared(handle int32) {
	if statement, exists := c.prepared[handle]; exists {
		c.preparedBytes -= statement.bytes
		delete(c.prepared, handle)
	}
}

// Preparation defers SQL compilation until execution; it never runs a query to
// discover its schema. RETURN_METADATA requires an explicitly configured pure
// describe callback and otherwise fails rather than returning invented types.
func (c *tdsConnection) prepareRPC(rpc tdsRPC, response *tdsResponse) (query string, params map[string]Scmer, handle int32, name string, specs []tdsDeclaration) {
	arguments := rpc.arguments
	switch rpc.procedure {
	case 10:
		if len(arguments) < 1 {
			panic("missing sp_executesql statement")
		}
		query = tdsRPCText(arguments[0], false)
		if strings.TrimSpace(query) == "" {
			panic("empty sp_executesql statement")
		}
		if len(arguments) == 1 {
			return query, nil, 0, "", nil
		}
		declarations, err := c.declarations(tdsRPCText(arguments[1], true))
		if err != nil {
			panic(err)
		}
		return query, c.bindParameters(declarations, arguments[2:], tdsWireDescriptors(rpc, 2)), 0, "", declarations
	case 12, 15:
		if len(arguments) < 1 {
			panic("missing TDS prepared handle")
		}
		id := tdsRPCHandle(arguments[0])
		statement, exists := c.prepared[id]
		if !exists {
			panic("unknown TDS prepared handle")
		}
		if rpc.procedure == 15 {
			if len(arguments) != 1 {
				panic("sp_unprepare expects one handle")
			}
			c.removePrepared(id)
			return "", nil, 0, "", nil
		}
		return statement.sql, c.bindParameters(statement.parameters, arguments[1:], tdsWireDescriptors(rpc, 1)), 0, "", statement.parameters
	case 11, 13:
		if len(arguments) < 3 || arguments[0].flags != 1 || (!arguments[0].value.IsInt() && !arguments[0].value.IsNil()) {
			panic("prepared RPC requires an output handle, declarations and statement")
		}
		declarationText, statementText := tdsRPCText(arguments[1], true), tdsRPCText(arguments[2], false)
		if strings.TrimSpace(statementText) == "" {
			panic("empty TDS prepared statement")
		}
		if _, err := splitTDSBatch(statementText); err != nil {
			panic(err)
		}
		declarations, err := c.declarations(declarationText)
		if err != nil {
			panic(err)
		}
		size := len(statementText) + len(declarationText)
		if len(c.prepared) >= tdsPreparedLimit || size > tdsPreparedBytesLimit-c.preparedBytes {
			panic("TDS prepared statement limit reached")
		}
		if c.nextPrepared == math.MaxInt32 {
			panic("TDS prepared handle space exhausted")
		}
		if rpc.procedure == 11 {
			if len(arguments) > 4 {
				panic("sp_prepare expects at most four arguments")
			}
			if len(arguments) == 4 {
				option := arguments[3]
				if option.flags != 0 || !option.value.IsInt() || (option.value.Int() != 0 && option.value.Int() != 1) {
					panic("unsupported sp_prepare options")
				}
				if option.value.Int() == 1 {
					if c.server.describe.IsNil() {
						panic("prepared result metadata is unavailable")
					}
					descriptionSession := c.descriptionSession(declarations)
					c.sessionEvent("describe", descriptionSession, NewBool(true))
					descriptors := Apply(c.server.describe, NewString(c.database), NewString(statementText), NewString(declarationText), descriptionSession)
					sink := &tdsResult{w: response}
					var titles []Scmer
					for _, descriptor := range descriptors.Slice() {
						titles = append(titles, NewString(tdsDescriptorValue(descriptor, "name").String()))
					}
					sink.fields(NewSlice(titles), descriptors)
					sink.publish()
				}
			}
		} else {
			params = c.bindParameters(declarations, arguments[3:], tdsWireDescriptors(rpc, 3))
			query = statementText
		}
		c.nextPrepared++
		handle = c.nextPrepared
		if c.prepared == nil {
			c.prepared = make(map[int32]tdsPreparedStatement)
		}
		c.prepared[handle] = tdsPreparedStatement{statementText, declarationText, declarations, size}
		c.preparedBytes += size
		return query, params, handle, arguments[0].name, declarations
	}
	panic("unsupported TDS RPC")
}

func (c *tdsConnection) descriptionSession(declarations []tdsDeclaration) Scmer {
	session := c.session
	params := make(map[string]Scmer, len(declarations))
	for _, declaration := range declarations {
		params[declaration.name] = NewNil()
	}
	// Describe flags and parser scratch values must not enter the live connection.
	clone := NewSession()
	entries := session.Func()().Slice()
	for _, key := range entries {
		if key.IsString() {
			clone.Func()(key, session.Func()(key))
		}
	}
	return c.parameterSession(clone, params, declarations)
}
