/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

// CardinalityFailure is an immutable operator diagnostic. Frontends select
// their own error numbers; the runtime carries only the actual failure kind
// and message. Error preserves the historical human-readable diagnostic.
type CardinalityFailure struct{ message string }

func NewCardinalityFailure(message string) CardinalityFailure { return CardinalityFailure{message} }
func (failure CardinalityFailure) Error() string              { return failure.message }
func (failure CardinalityFailure) OriginalFailure() any {
	return NewSlice([]Scmer{NewString("cardinality-error"), NewString(failure.message)})
}
func (CardinalityFailure) ClusterObjectType() string { return "cardinality-failure" }
func (CardinalityFailure) ClusterImmutableView()     {}
func (failure CardinalityFailure) Serialize(*SerializationContext) (Scmer, error) {
	return NewSlice([]Scmer{NewSymbol("cardinality_failure"), NewString(failure.message)}), nil
}

func initCardinalityFailure() {
	Declare(&Globalenv, &Declaration{Name: "cardinality_failure", Fn: func(arguments ...Scmer) Scmer {
		if len(arguments) != 1 || !arguments[0].IsString() {
			panic("cardinality_failure requires a diagnostic message")
		}
		return NewAny(NewCardinalityFailure(arguments[0].String()))
	}, Type: &TypeDescriptor{Kind: "func", Const: true,
		Description: "Constructs an immutable cardinality diagnostic; error raises it without choosing a frontend-specific error code.",
		Params:      []*TypeDescriptor{{Kind: "string", Label: "message"}}, Return: &TypeDescriptor{Kind: "any"}}})
}
