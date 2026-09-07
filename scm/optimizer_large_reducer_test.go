/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.
*/
package scm

import (
	"strings"
	"testing"
)

// Large fused query plans must obey the same ownership rules as small reducers.
// In particular, compile-cost guards must not silently change an owned
// accumulator into a borrowed value: doing so selects the copying assoc helpers
// in precisely the plans where a copy is most expensive.
func TestLargeReducerPreservesOwnedAccumulator(t *testing.T) {
	body := make([]Scmer, 0, 304)
	body = append(body, NewSymbol("begin"))
	for i := 0; i < 300; i++ {
		body = append(body, NewInt(int64(i)))
	}
	body = append(body, NewSlice([]Scmer{
		NewSymbol("set_assoc"), NewSymbol("acc"), NewSymbol("key"), NewBool(true),
	}))
	callback := NewSlice([]Scmer{
		NewSymbol("lambda"),
		NewSlice([]Scmer{NewSymbol("acc"), NewSymbol("key")}),
		NewSlice(body),
	})

	meta := newOptimizerMetainfo()
	context := OptimizerContext{Env: &Globalenv, Ome: &meta}
	optimized, resultType := context.OptimizeReducerCallback(
		callback,
		&TypeDescriptor{Kind: "assoc", Transfer: true, Length: UnknownLength},
		&TypeDescriptor{Kind: "string", Length: UnknownLength},
	)
	serialized := SerializeToString(optimized, &Globalenv)
	if !strings.Contains(serialized, "set_assoc_mut") || strings.Contains(serialized, "(set_assoc ") {
		t.Fatalf("large reducer lost accumulator ownership: %s", serialized)
	}
	if resultType == nil || !resultType.Transfer {
		t.Fatalf("large reducer result lost transfer ownership: %#v", resultType)
	}
}
