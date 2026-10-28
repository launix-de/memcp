/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/
package storage

import "fmt"
import "bytes"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

// schemeValue contains only existing Scheme reconstruction source; transport
// has no runtime type switch or aggregation semantics.
type schemeValue json.RawMessage

func (v schemeValue) MarshalJSON() ([]byte, error) {
	if len(v) == 0 {
		return []byte("null"), nil
	}
	return v, nil
}
func (v *schemeValue) UnmarshalJSON(data []byte) error { *v = append((*v)[:0], data...); return nil }

// The transport calls the ordinary serializer directly. Quoted lists remain
// data; Serialize emits constructors only where reconstruction requires them.
func freezeScanValue(value scm.Scmer, _ int, bindings ...*scm.SerializationContext) (data schemeValue, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			data = nil
			err = fmt.Errorf("value serialization: %v", failure)
		}
	}()
	if len(bindings) == 0 || bindings[0] == nil {
		bindings = []*scm.SerializationContext{new(scm.SerializationContext)}
	}
	value = value.WithoutSourceInfo()
	if value.IsSlice() || value.IsSymbol() {
		value = scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), value})
	}
	var source bytes.Buffer
	scm.Serialize(&source, value, &scm.Globalenv, bindings...)
	encoded, err := json.Marshal(source.String())
	return schemeValue(encoded), err
}
func (value schemeValue) thaw(_ int, decode ...scm.ProcedureDecode) (result scm.Scmer, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			result = scm.NewNil()
			err = fmt.Errorf("value reconstruction: %v", failure)
		}
	}()
	var source string
	if err = json.Unmarshal(value, &source); err != nil {
		return scm.NewNil(), err
	}
	expression := scm.Read("cluster value", source)
	var option scm.ProcedureDecode
	if len(decode) > 0 {
		option = decode[0]
	}
	if option.Prepare != nil {
		expression, err = option.Prepare(expression)
		if err != nil {
			return scm.NewNil(), err
		}
	}
	expression, err = scm.ResolveSerializedObjects(expression, option.Resolve)
	if err != nil {
		return scm.NewNil(), err
	}
	return scm.Eval(scm.Optimize(expression, &scm.Globalenv, nil), &scm.Globalenv).WithoutSourceInfo(), nil
}

// remoteScanAccess reuses the native persisted boundary representation and
// binds actual scalar operands. Candidate filtering/index deployment stays in
// the existing scan engine; no residual predicate or mandatory constraint is
// discarded by delegation. Runtime physical source sets remain nonportable.
type remoteScanAccess struct {
	Orders     []json.RawMessage `json:"orders,omitempty"`
	Boundaries []scm.Scmer       `json:"boundaries,omitempty"`
	Values     []schemeValue     `json:"values,omitempty"`
	Covered    bool              `json:"covered,omitempty"`
}

func freezeScanAccess(access scanAccess, bindings ...*scm.SerializationContext) (remoteScanAccess, error) {
	// These carry physical invocation state beyond scalar filter operands.
	// Reject rather than silently omit their execution/lifetime semantics.
	if access.runtime != nil && (access.runtime.mutation || access.runtime.prefixMerge != nil) {
		return remoteScanAccess{}, fmt.Errorf("remote scan requires a portable read access contract")
	}
	out := remoteScanAccess{Covered: access.plannerFilterCovered || access.runtime != nil && access.runtime.filterCovered}
	for i := 0; i < access.len(); i++ {
		original, operands := access.boundaryParts(i)
		box := *original
		appendValue := func(value scm.Scmer) (int, error) {
			frozen, err := freezeScanValue(value, 0, bindings...)
			if err != nil {
				return 0, err
			}
			slot := len(out.Values)
			out.Values = append(out.Values, frozen)
			return slot, nil
		}
		if box.mapperSlot >= 0 {
			var err error
			box.mapperSlot, err = appendValue(operands[original.mapperSlot])
			if err != nil {
				return out, err
			}
		}
		if box.analyzer == RecSetMatcher {
			return out, fmt.Errorf("physical RecSet cannot leave its query node")
		}
		// Negative batch slots name fields in the serialized input batch. Do not
		// collapse them into a nil scalar before scanBatch binds each input row.
		for _, upper := range []bool{false, true} {
			slot := original.lowerSlot
			if upper {
				slot = original.upperSlot
			}
			if slot >= 0 {
				var err error
				slot, err = appendValue(access.boundValue(i, upper))
				if err != nil {
					return out, err
				}
			}
			if upper {
				box.upperSlot = slot
			} else {
				box.lowerSlot = slot
			}
		}
		var order json.RawMessage
		if box.order != nil {
			if _, _, collation := scm.LookupCollate(box.order); !collation {
				var err error
				var binding *scm.SerializationContext
				if len(bindings) > 0 {
					binding = bindings[0]
				}
				order, err = freezeScanDirection(box.order, binding)
				if err != nil {
					return out, err
				}
				box.order, box.orderMetadata = nil, ""
			}
		}
		out.Orders = append(out.Orders, order)
		out.Boundaries = append(out.Boundaries, NewScanBoundaryScmer(&box))
	}
	return out, nil
}
func (access remoteScanAccess) bind(decode ...scm.ProcedureDecode) (scanAccess, error) {
	if len(access.Orders) != 0 && len(access.Orders) != len(access.Boundaries) {
		return scanAccess{}, fmt.Errorf("invalid boundary order directory")
	}
	access.Boundaries = append([]scm.Scmer(nil), access.Boundaries...)
	for i, item := range access.Boundaries {
		if len(access.Orders) > 0 && len(access.Orders[i]) > 0 && string(access.Orders[i]) != "null" {
			options := scm.ProcedureDecode{}
			if len(decode) > 0 {
				options = decode[0]
			}
			constructor, err := scm.UnmarshalProcedure(access.Orders[i], options)
			if err != nil {
				return scanAccess{}, err
			}
			direction := scanSortDirections([]scm.Scmer{scm.Apply(constructor)})[0]
			if !item.IsCustom(TagScanBoundary) {
				return scanAccess{}, fmt.Errorf("invalid ordered boundary")
			}
			box := *ScanBoundaryFromScmer(item).(*scanBoundaryBox)
			box.order, box.orderMetadata = direction, orderRelationMeta(direction)
			item = NewScanBoundaryScmer(&box)
			access.Boundaries[i] = item
		}
		if !item.IsCustom(TagScanBoundary) {
			return scanAccess{}, fmt.Errorf("invalid remote boundary")
		}
		box := ScanBoundaryFromScmer(item)
		if box.MapperSlot() < -1 || box.MapperSlot() >= len(access.Values) || box.Analyzer() == RecSetMatcher {
			return scanAccess{}, fmt.Errorf("unsupported remote physical input")
		}
		for _, slot := range []int{box.LowerSlot(), box.UpperSlot()} {
			if slot >= len(access.Values) {
				return scanAccess{}, fmt.Errorf("invalid remote boundary slot")
			}
		}
	}
	values := make([]scm.Scmer, len(access.Values))
	for i, value := range access.Values {
		var err error
		values[i], err = value.thaw(0, decode...)
		if err != nil {
			return scanAccess{}, err
		}
	}
	return scanAccess{runtime: &scanAccessRuntime{suffix: scanAccessSegment{items: access.Boundaries, values: values}, filterCovered: access.Covered}}, nil
}

// A cancelled or rejected completion must retire offers its receiver never
// acquired, including references nested in callback captures and object views.
func scanWireReferences(value schemeValue) []scm.ClusterReference {
	return scm.ProcedureClusterReferences(value)
}

func referenceOffers(ref scm.ClusterReference) []scm.ClusterReference {
	result := []scm.ClusterReference{ref}
	for _, child := range scm.ProcedureClusterReferences(ref.View) {
		result = append(result, referenceOffers(child)...)
	}
	return result
}
