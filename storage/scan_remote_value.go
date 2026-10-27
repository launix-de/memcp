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
import "encoding/json"
import "github.com/launix-de/memcp/scm"

// scanWireValue represents native reducer data, not a second SQL aggregation
// model. Dictionary keys may themselves be tuples (including NULL domains).
// Runtime objects/procedures enter only with explicit export/decode contexts;
// aggregate-only callers remain data-only. Encoding happens at task boundaries,
// never per row. Physical query-local containers without a contract fail.
type scanWireValue struct {
	Reference *scm.ClusterReference `json:"reference,omitempty"`
	Procedure json.RawMessage       `json:"procedure,omitempty"`
	Kind      string                `json:"kind"`
	Scalar    json.RawMessage       `json:"scalar,omitempty"`
	Items     []scanWireValue       `json:"items,omitempty"`
}

func freezeScanValue(value scm.Scmer, depth int, bindings ...*scm.ClusterBindings) (scanWireValue, error) {
	// Parser locations are diagnostic wrappers, not runtime object identities.
	// Unwrap before choosing the recursive container codec as well as validation.
	value = value.WithoutSourceInfo()
	if depth > 64 {
		return scanWireValue{}, fmt.Errorf("remote aggregate nesting limit")
	}
	if len(bindings) > 0 && bindings[0] != nil {
		if object, ok := scm.ClusterObjectOf(value); ok {
			ref, err := bindings[0].Reference(object)
			return scanWireValue{Kind: "object", Reference: &ref}, err
		}
		if value.IsProc() {
			data, err := scm.MarshalProcedure(value, bindings[0])
			return scanWireValue{Kind: "procedure", Procedure: data}, err
		}
	}
	out := scanWireValue{Kind: "scalar"}
	var items []scm.Scmer
	if value.IsFastDict() {
		out.Kind = "dict"
		items = value.FastDict().ToList()
	} else if value.IsSlice() {
		out.Kind = "list"
		items = value.Slice()
	} else {
		if err := checkScanData(value, depth); err != nil {
			return out, err
		}
		data, err := json.Marshal(value)
		out.Scalar = data
		return out, err
	}
	out.Items = make([]scanWireValue, len(items))
	for i, item := range items {
		var err error
		out.Items[i], err = freezeScanValue(item, depth+1, bindings...)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}
func (value scanWireValue) thaw(depth int, decode ...scm.ProcedureDecode) (scm.Scmer, error) {
	if depth > 64 {
		return scm.NewNil(), fmt.Errorf("remote aggregate nesting limit")
	}
	if value.Reference != nil && value.Kind != "object" || len(value.Procedure) != 0 && value.Kind != "procedure" {
		return scm.NewNil(), fmt.Errorf("mixed runtime value encoding")
	}
	if value.Kind == "object" {
		if len(value.Items) != 0 || len(value.Scalar) != 0 || value.Reference == nil || len(decode) == 0 || decode[0].Resolve == nil {
			return scm.NewNil(), fmt.Errorf("runtime object requires a reconstruction context")
		}
		return decode[0].Resolve(*value.Reference)
	}
	if value.Kind == "procedure" {
		if len(value.Items) != 0 || len(value.Scalar) != 0 || len(decode) == 0 {
			return scm.NewNil(), fmt.Errorf("procedure requires a reconstruction context")
		}
		return scm.UnmarshalProcedure(value.Procedure, decode[0])
	}
	if value.Kind == "scalar" {
		if len(value.Items) != 0 {
			return scm.NewNil(), fmt.Errorf("invalid scalar aggregate")
		}
		return decodeScanData(value.Scalar, false)
	}
	if (value.Kind != "list" && value.Kind != "dict") || len(value.Scalar) != 0 || value.Kind == "dict" && len(value.Items)%2 != 0 {
		return scm.NewNil(), fmt.Errorf("invalid aggregate container")
	}
	items := make([]scm.Scmer, len(value.Items))
	for i, item := range value.Items {
		var err error
		items[i], err = item.thaw(depth+1, decode...)
		if err != nil {
			return scm.NewNil(), err
		}
	}
	if value.Kind == "dict" {
		dict := scm.NewFastDictValue(len(items) / 2)
		for i := 0; i < len(items); i += 2 {
			dict.Set(items[i], items[i+1], nil)
		}
		return scm.NewFastDict(dict), nil
	}
	return scm.NewSlice(items), nil
}

// remoteScanAccess reuses the native persisted boundary representation and
// binds actual scalar operands. Candidate filtering/index deployment stays in
// the existing scan engine; no residual predicate or mandatory constraint is
// discarded by delegation. Runtime physical source sets remain nonportable.
type remoteScanAccess struct {
	Boundaries []scm.Scmer `json:"boundaries,omitempty"`
	Values     []scm.Scmer `json:"values,omitempty"`
	Covered    bool        `json:"covered,omitempty"`
}

func freezeScanAccess(access scanAccess) (remoteScanAccess, error) {
	// These carry physical invocation state beyond scalar filter operands.
	// Reject rather than silently omit their execution/lifetime semantics.
	if access.runtime != nil && (access.runtime.mutation || access.runtime.prefixMerge != nil) {
		return remoteScanAccess{}, fmt.Errorf("remote scan requires a portable read access contract")
	}
	out := remoteScanAccess{Covered: access.plannerFilterCovered || access.runtime != nil && access.runtime.filterCovered}
	for i := 0; i < access.len(); i++ {
		original, _ := access.boundaryParts(i)
		box := *original
		if box.mapperSlot >= 0 {
			return out, fmt.Errorf("remote computed boundary requires a serialized mapper")
		}
		if box.analyzer == RecSetMatcher {
			return out, fmt.Errorf("physical RecSet cannot leave its query node")
		}
		box.lowerSlot = -1
		box.upperSlot = -1
		if original.lowerSlot != -1 {
			box.lowerSlot = len(out.Values)
			out.Values = append(out.Values, access.boundValue(i, false))
		}
		if original.upperSlot != -1 {
			box.upperSlot = len(out.Values)
			out.Values = append(out.Values, access.boundValue(i, true))
		}
		out.Boundaries = append(out.Boundaries, NewScanBoundaryScmer(&box))
	}
	for _, value := range out.Values {
		if err := checkScanData(value, 0); err != nil {
			return out, err
		}
	}
	return out, nil
}
func (access remoteScanAccess) bind() (scanAccess, error) {
	for _, item := range access.Boundaries {
		if !item.IsCustom(TagScanBoundary) {
			return scanAccess{}, fmt.Errorf("invalid remote boundary")
		}
		box := ScanBoundaryFromScmer(item)
		if box.MapperSlot() != -1 || box.Analyzer() == RecSetMatcher {
			return scanAccess{}, fmt.Errorf("unsupported remote physical input")
		}
		for _, slot := range []int{box.LowerSlot(), box.UpperSlot()} {
			if slot < -1 || slot >= len(access.Values) {
				return scanAccess{}, fmt.Errorf("invalid remote boundary slot")
			}
		}
	}
	for _, value := range access.Values {
		if err := checkScanData(value, 0); err != nil {
			return scanAccess{}, err
		}
	}
	return scanAccess{runtime: &scanAccessRuntime{suffix: scanAccessSegment{items: access.Boundaries, values: access.Values}, filterCovered: access.Covered}}, nil
}

// A cancelled or rejected completion must retire offers its receiver never
// acquired, including references nested in callback captures and object views.
func scanWireReferences(value scanWireValue) []scm.ClusterReference {
	var result []scm.ClusterReference
	seen := make(map[string]bool)
	var walk func(scanWireValue, int)
	var reference func(scm.ClusterReference, int)
	reference = func(ref scm.ClusterReference, depth int) {
		if depth > 64 || ref.Token == "" || seen[ref.URI+"/"+ref.Token] {
			return
		}
		seen[ref.URI+"/"+ref.Token] = true
		result = append(result, ref)
		var view scanWireValue
		if json.Unmarshal(ref.View, &view) == nil {
			walk(view, depth+1)
		}
	}
	walk = func(value scanWireValue, depth int) {
		if depth > 64 {
			return
		}
		if value.Reference != nil {
			reference(*value.Reference, depth+1)
		}
		for _, ref := range scm.ProcedureClusterReferences(value.Procedure) {
			reference(ref, depth+1)
		}
		for _, item := range value.Items {
			walk(item, depth+1)
		}
	}
	walk(value, 0)
	return result
}
