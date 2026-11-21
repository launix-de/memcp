/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "maps"
import "math"
import "unicode/utf8"
import "github.com/launix-de/memcp/scm"

// A claim interprets only inert native declaration data. The frontend owns the
// property's name, requested value and declaration eligibility rules.
type columnPropertyClaim struct {
	key      string
	value    scm.Scmer
	expected storedColumnType
}

func decodeColumnPropertyClaim(value scm.Scmer) columnPropertyClaim {
	if !value.IsSlice() || len(value.Slice()) != 3 {
		panic("column property claim requires key, scalar value and expected declaration")
	}
	fields := value.Slice()
	if !fields[0].IsString() || fields[0].String() == "" || !utf8.ValidString(fields[0].String()) {
		panic("column property claim requires a valid nonempty key")
	}
	if err := validateColumnProperty(fields[0].String(), fields[1]); err != nil {
		panic(err)
	}
	return columnPropertyClaim{fields[0].String(), fields[1], decodeStoredColumnType(fields[2])}
}

// The semantic primitive kind belongs to the V20 recipe. An integral floating
// point value must not become an integer precondition after JSON decoding.
func columnPropertyValueKind(value scm.Scmer) string {
	switch {
	case value.IsNil():
		return "nil"
	case value.IsBool():
		return "bool"
	case value.IsInt():
		return "int"
	case value.IsFloat():
		return "float"
	case value.IsString():
		return "string"
	default:
		panic("column property value must be a scalar")
	}
}

func columnPropertyValue(kind string, value scm.Scmer) (scm.Scmer, error) {
	switch kind {
	case "nil":
		if value.IsNil() {
			return value, nil
		}
	case "bool":
		if value.IsBool() {
			return value, nil
		}
	case "int":
		if value.IsInt() {
			return value, nil
		}
	case "float":
		if value.IsInt() || value.IsFloat() {
			v := value.Float()
			if !math.IsNaN(v) && !math.IsInf(v, 0) {
				return scm.NewFloat(v), nil
			}
		}
	case "string":
		if value.IsString() {
			return value, nil
		}
	}
	return scm.NewNil(), fmt.Errorf("invalid native column property scalar kind")
}

func sameColumnProperty(a, b scm.Scmer) bool {
	// Numbers retain their existing semantic equality through schema JSON,
	// which can represent an integral float as an integer. Booleans and text
	// remain separate identities; numeric-looking strings are exact text.
	if (a.IsInt() || a.IsFloat()) && (b.IsInt() || b.IsFloat()) {
		return scm.Equal(a, b)
	}
	if columnPropertyValueKind(a) != columnPropertyValueKind(b) {
		return false
	}
	if a.IsString() {
		return a.String() == b.String()
	}
	return scm.Equal(a, b)
}

// Only an observed owner of the requested scalar is a semantic conflict.
// The stable datum retains native completion identity across projections.
type columnPropertyConflict struct {
	message    string
	diagnostic scm.Scmer
}

func (failure *columnPropertyConflict) Error() string        { return failure.message }
func (failure *columnPropertyConflict) OriginalFailure() any { return failure.diagnostic }

func newColumnPropertyConflict(t *table, column, key, owner string, value scm.Scmer) *columnPropertyConflict {
	return &columnPropertyConflict{
		message: fmt.Sprintf("column property %s is already owned by %s.%s.%s", key, t.schema.Name, t.Name, owner),
		diagnostic: scm.NewSlice([]scm.Scmer{scm.NewString("column-property-conflict"),
			scm.NewString(t.Name), scm.NewString(column), scm.NewString(key), value, scm.NewString(owner)}),
	}
}

// Caller holds the existing table declaration publication owner and supplies
// its one immutable effective column directory. There are no callbacks or row
// operations in this metadata claim.
func (t *table) claimColumnProperty(columns []*column, index int, request columnPropertyClaim) *column {
	current := columns[index]
	if current.IsTemp || current.ComputorDefinition != nil || isRuntimeComputedColumn(current) {
		panic("column property claims require ordinary stored declarations")
	}
	if !storedColumnTypeOf(current).equal(request.expected) {
		panic("column property claim lost its expected declaration")
	}
	for _, candidate := range columns {
		if value, found := candidate.DeclarationProperties[request.key]; found && sameColumnProperty(value, request.value) {
			panic(newColumnPropertyConflict(t, current.Name, request.key, candidate.Name, request.value))
		}
	}
	properties := maps.Clone(current.DeclarationProperties)
	if properties == nil {
		properties = make(columnDeclarationProperties)
	}
	properties[request.key] = request.value
	t.requireDeclaredColumnAlter(current, "properties", properties.description())
	replacement := cloneStoredColumnDeclaration(current)
	replacement.DeclarationProperties = properties
	return replacement
}
