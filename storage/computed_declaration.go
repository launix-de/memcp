/*
	Copyright (C) 2026 Carl-Philip Hänsch

SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "maps"

import "fmt"
import "encoding/json"
import "slices"
import "github.com/launix-de/memcp/scm"

// Declared recipes belong to schema, independently of whether any shard cache
// has been materialized. Existing runtime-only computed caches omit this field.
type computedColumnDefinition struct {
	Version  int    `json:"version"`
	Function string `json:"function"`
}

func (c *column) UnmarshalJSON(data []byte) (err error) {
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("invalid declared computation: %v", failure)
		}
	}()
	type declaration column
	if err := json.Unmarshal(data, (*declaration)(c)); err != nil {
		return err
	}
	if c.DefaultDeferred && (!c.Default.IsNil() || c.DefaultExpression == "") {
		return fmt.Errorf("deferred default requires an opaque source and no constant value")
	}
	if c.ComputorDefinition != nil {
		definition := c.ComputorDefinition
		if definition.Version != 1 || definition.Function == "" {
			return fmt.Errorf("invalid declared computation format")
		}
		// Serialized procedures include numbered local bindings. Decode them
		// through the language compiler before evaluating the restored closure.
		expression := scm.Read("declared computation", definition.Function, true)
		function := scm.Eval(scm.Optimize(expression, &scm.Globalenv, nil), &scm.Globalenv)
		if !function.IsProc() || hasImplicitComputeContext(function) {
			return fmt.Errorf("declared computation must be a procedure using explicit inputs")
		}
		c.Computor = function
	}
	return nil
}

func declareColumnComputor(c *column, input scm.Scmer, function scm.Scmer) {
	if function.IsJIT() {
		function = scm.NewProcStruct(function.JIT().Proc.Snapshot())
	}
	function = scm.CloseProcedure(function)
	if !function.IsProc() {
		panic("declared computation requires a closed Scheme procedure")
	}
	requireStatelessComputedCallback("declared computation", function)
	if _, err := scm.MarshalProcedure(function); err != nil {
		panic(fmt.Errorf("declared computation is not persistent: %w", err))
	}
	inputs := scmerSliceToStrings(mustScmerSlice(input, "computed input columns"))
	params := function.Proc().Params
	if !params.IsSlice() || len(params.Slice()) != len(inputs) {
		panic("declared computation parameter count does not match its input columns")
	}
	if c.AutoIncrement || c.hasDefault() || c.DefaultName != "" || !c.OnUpdate.IsNil() || c.IsTemp {
		panic("declared computation cannot also have defaults, identity, update or temporary storage")
	}
	c.Computor = function
	c.ComputorInputCols = append([]string(nil), inputs...)
	c.ComputorDefinition = &computedColumnDefinition{Version: 1, Function: scm.SerializeToString(function, &scm.Globalenv)}
}

// Resolve the entire immutable declaration before publication. Dependencies
// are same-table columns; cycles would otherwise recurse through lazy readers.
func (t *table) bindDeclaredComputations() {
	columns := make(map[string]*column, len(t.Columns))
	for _, c := range t.Columns {
		columns[c.Name] = c
	}
	state := make(map[string]uint8, len(columns))
	var visit func(*column)
	visit = func(c *column) {
		if c.ComputorDefinition == nil || state[c.Name] == 2 {
			return
		}
		if state[c.Name] == 1 {
			panic("cyclic declared computation: " + c.Name)
		}
		state[c.Name] = 1
		seen := make(map[string]bool, len(c.ComputorInputCols))
		for _, name := range c.ComputorInputCols {
			input := columns[name]
			if input == nil {
				panic("unknown computed input column: " + name)
			}
			if seen[name] {
				panic("duplicate computed input column: " + name)
			}
			seen[name] = true
			visit(input)
		}
		state[c.Name] = 2
	}
	for _, c := range t.Columns {
		visit(c)
	}
	for _, key := range t.Unique {
		t.requireStoredUniqueKey(key)
	}
	for _, c := range t.Columns {
		if c.ComputorDefinition != nil {
			for _, shard := range t.ActiveShards() {
				shard.columns[c.Name] = shard.makeComputedColumnProxy(c.Name, c)
			}
		}
	}
}

// Pending unique-key extraction currently binds stored values. A declared
// computation cannot be treated as an omitted/default NULL during admission.
func (t *table) requireStoredUniqueKey(key uniqueKey) {
	for _, name := range key.Cols {
		for _, column := range t.columnDeclarations() {
			if column.Name == name && column.ComputorDefinition != nil {
				panic("unique keys on declared computed columns are not supported")
			}
		}
	}
}

// Input declarations are part of the closed recipe's contract. Changing them
// needs a replacement recipe published atomically with the column declaration.
// Callers own table DDL; reject before removing storage or changing metadata.
func (t *table) requireNoDeclaredComputationInput(name string) {
	for _, c := range t.Columns {
		if c.ComputorDefinition != nil && slices.Contains(c.ComputorInputCols, name) {
			panic("column is an input of a declared computation: " + name)
		}
	}
}

func (t *table) requireDeclaredColumnAlter(current *column, operation string, value scm.Scmer) {
	if current.ComputorDefinition != nil {
		switch operation {
		case "default", "default_expression", "default_declaration", "add_default", "auto_increment", "temp", "update":
			panic("declared computation cannot also have defaults, identity, update or temporary storage")
		}
	}
	switch operation {
	case "type", "dimensions", "null", "collation":
		// A fresh descriptor reuses the native property's decoding without
		// touching the published descriptor or copying its runtime ownership.
		candidate := &column{Typ: current.Typ, Typdimensions: current.Typdimensions,
			AllowNull: current.AllowNull, Collation: current.Collation}
		candidate.Alter(operation, value)
		if candidate.Typ == current.Typ && candidate.AllowNull == current.AllowNull &&
			candidate.Collation == current.Collation && slices.Equal(candidate.Typdimensions, current.Typdimensions) {
			return
		}
		t.requireNoConstraintColumnChange(current.Name)
		t.requireNoDeclaredComputationInput(current.Name)
		if current.ComputorDefinition != nil {
			panic("declared computation requires its original output declaration")
		}
	case "properties":
		if t.checkColumnDependency(current.Name) != nil {
			candidate := columnPropertiesFromValue(value)
			if !maps.EqualFunc(current.DeclarationProperties, candidate, func(a, b scm.Scmer) bool { return a.GetTag() == b.GetTag() && scm.Equal(a, b) }) {
				panic("check input declaration properties require joint replacement")
			}
		}
	case "temp":
		if current.IsTemp != scm.ToBool(value) {
			t.requireNoConstraintColumnChange(current.Name)
		}
	}
}
