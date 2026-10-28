/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "context"
import "github.com/launix-de/memcp/scm"

// Dynamic join/membership filters are immutable runtime values. Keep their
// native callable ABI locally, but serialize a constructor with key values and
// positions, never a Go closure or a per-row owner RPC. The peer constructs its
// private lookup once through ClusterObject.Serialize before scanning any rows.
type scanKeyPredicate struct {
	constraints []scanJoinOrderDriverConstraint
	filter      scm.Scmer
	skip        int
}

var scanKeyPredicateType = scm.RegisterCallableType(&scm.TypeDescriptor{Kind: "func", Label: "immutable key predicate", Params: []*scm.TypeDescriptor{{Kind: "any", Variadic: true}}, Return: &scm.TypeDescriptor{Kind: "bool"}})

func (*scanKeyPredicate) ClusterObjectType() string { return "scan-key-predicate" }
func (*scanKeyPredicate) HandleClusterRequest(context.Context, scm.ClusterRequest) (any, error) {
	return nil, fmt.Errorf("immutable key predicates use portable constructors")
}
func (predicate *scanKeyPredicate) Serialize(*scm.SerializationContext) (scm.Scmer, error) {
	constraints := make([]scm.Scmer, len(predicate.constraints))
	for i, constraint := range predicate.constraints {
		positions := make([]scm.Scmer, len(constraint.positions))
		for k, p := range constraint.positions {
			positions[k] = scm.NewInt(int64(p))
		}
		keys := make([]scm.Scmer, len(constraint.keys))
		for k, key := range constraint.keys {
			keys[k] = scm.NewSlice(key)
		}
		constraints[i] = scm.NewSlice([]scm.Scmer{scm.NewSlice(positions), scm.NewSlice(keys)})
	}
	return scm.NewSlice([]scm.Scmer{scm.NewSymbol("scan_key_predicate"), scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), scm.NewSlice(constraints)}), predicate.filter, scm.NewInt(int64(predicate.skip))}), nil
}
func newScanKeyPredicate(constraints []scanJoinOrderDriverConstraint, filter scm.Scmer, skip int) scm.Scmer {
	predicate := &scanKeyPredicate{constraints: constraints, filter: filter, skip: skip}
	keys := make([]recSetProjectKeys, len(constraints))
	for i, constraint := range constraints {
		keys[i].width = len(constraint.positions)
		for _, key := range constraint.keys {
			keys[i].values = append(keys[i].values, key...)
		}
		keys[i].buildNumericLookup()
	}
	return scm.NewClusterFunc(func(values ...scm.Scmer) scm.Scmer {
		for i, constraint := range constraints {
			var inline [8]scm.Scmer
			var key []scm.Scmer
			if len(constraint.positions) <= len(inline) {
				key = inline[:len(constraint.positions)]
			} else {
				key = make([]scm.Scmer, len(constraint.positions))
			}
			for k, position := range constraint.positions {
				key[k] = values[position]
			}
			if scanJoinOrderKeyHasNull(key) || !keys[i].contains(key) {
				return scm.NewBool(false)
			}
		}
		if filter.IsNil() {
			return scm.NewBool(true)
		}
		return scm.Apply(filter, values[skip:]...)
	}, predicate, scanKeyPredicateType)
}
func scanJoinKeyLess(values ...scm.Scmer) scm.Scmer {
	return scm.NewBool(scm.Less(values[0], values[1]))
}
func scanTrue(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }
func scanCollectRow(values ...scm.Scmer) scm.Scmer {
	row := scm.NewSlice(append([]scm.Scmer(nil), values[1:]...))
	return scm.NewSlice(append(values[0].Slice(), row))
}
func scanAppendRows(values ...scm.Scmer) scm.Scmer {
	return scm.NewSlice(append(values[0].Slice(), values[1].Slice()...))
}
func declareScanKeyPredicate(en *scm.Env) {
	scm.Declare(en, &scm.Declaration{Name: "scan_true", Fn: scanTrue, Type: &scm.TypeDescriptor{Kind: "func", Const: true, Params: []*scm.TypeDescriptor{{Kind: "any", Variadic: true}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
	scm.Declare(en, &scm.Declaration{Name: "scan_collect_row", Fn: scanCollectRow, Type: &scm.TypeDescriptor{Kind: "func", Params: []*scm.TypeDescriptor{{Kind: "list"}, {Kind: "any", Variadic: true}}, Return: &scm.TypeDescriptor{Kind: "list"}}})
	scm.Declare(en, &scm.Declaration{Name: "scan_append_rows", Fn: scanAppendRows, Type: &scm.TypeDescriptor{Kind: "func", Params: []*scm.TypeDescriptor{{Kind: "list"}, {Kind: "list"}}, Return: &scm.TypeDescriptor{Kind: "list"}}})
	scm.Declare(en, &scm.Declaration{Name: "scan_key_less", Fn: scanJoinKeyLess, Type: &scm.TypeDescriptor{Kind: "func", Const: true, Params: []*scm.TypeDescriptor{{Kind: "any"}, {Kind: "any"}}, Return: &scm.TypeDescriptor{Kind: "bool"}}})
	scm.Declare(en, &scm.Declaration{Name: "scan_key_predicate", Fn: func(args ...scm.Scmer) scm.Scmer {
		var constraints []scanJoinOrderDriverConstraint
		for _, value := range mustScmerSlice(args[0], "key constraints") {
			parts := mustScmerSlice(value, "key constraint")
			if len(parts) != 2 {
				panic("invalid key predicate constraint")
			}
			var constraint scanJoinOrderDriverConstraint
			for _, position := range mustScmerSlice(parts[0], "key positions") {
				position = position.WithoutSourceInfo()
				if !position.IsInt() || position.Int() < 0 {
					panic("invalid key predicate position")
				}
				constraint.positions = append(constraint.positions, int(position.Int()))
			}
			if len(constraint.positions) == 0 {
				panic("empty key predicate width")
			}
			for _, row := range mustScmerSlice(parts[1], "key rows") {
				key := mustScmerSlice(row, "key row")
				key = append([]scm.Scmer(nil), key...)
				for i, value := range key {
					key[i] = value.WithoutSourceInfo()
				}
				if len(key) != len(constraint.positions) {
					panic("invalid key predicate width")
				}
				constraint.keys = append(constraint.keys, key)
			}
			constraints = append(constraints, constraint)
		}
		skip := int(args[2].Int())
		if skip < 0 {
			panic("invalid residual offset")
		}
		return newScanKeyPredicate(constraints, args[1], skip)
	}, Type: &scm.TypeDescriptor{Kind: "func", Params: []*scm.TypeDescriptor{{Kind: "list"}, {Kind: "func|nil"}, {Kind: "int"}}, Return: &scm.TypeDescriptor{Kind: "func"}}})
}
