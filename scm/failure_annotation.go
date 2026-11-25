/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import "fmt"

// Language provenance is immutable metadata around the original failure. It
// cannot replace native completion, cancellation or transaction authority.
type annotatedFailure struct {
	cause Scmer
	key   string
	value Scmer
}

func (failure *annotatedFailure) Error() string        { return fmt.Sprint(failure.cause) }
func (failure *annotatedFailure) OriginalFailure() any { return failure.cause }
func (failure *annotatedFailure) Unwrap() error {
	if failure.cause.GetTag() == tagAny {
		cause, _ := failure.cause.Any().(error)
		return cause
	}
	return nil
}

// Only expose OriginalStack when an actual origin stack exists. An empty stack
// capability would suppress the normal diagnostic fallback in PrintError.
type annotatedStackFailure struct {
	*annotatedFailure
	stack string
}

func (failure *annotatedStackFailure) OriginalStack() string { return failure.stack }

func explicitFailureStack(value Scmer) string {
	for value.GetTag() == tagAny {
		if carrier, ok := value.Any().(interface{ OriginalStack() string }); ok {
			if stack := carrier.OriginalStack(); stack != "" {
				return stack
			}
		}
		carrier, ok := value.Any().(interface{ OriginalFailure() any })
		if !ok {
			break
		}
		value = FromAny(carrier.OriginalFailure())
	}
	return ""
}

func failureAnnotation(value Scmer, key string) Scmer {
	for value.GetTag() == tagAny {
		switch failure := value.Any().(type) {
		case *annotatedFailure:
			if failure.key == key {
				return failure.value
			}
		case *annotatedStackFailure:
			if failure.key == key {
				return failure.value
			}
		}
		carrier, ok := value.Any().(interface{ OriginalFailure() any })
		if !ok {
			break
		}
		value = FromAny(carrier.OriginalFailure())
	}
	return NewNil()
}

func initFailureAnnotations() {
	Declare(&Globalenv, &Declaration{
		Name: "failure_annotate",
		Fn: func(arguments ...Scmer) Scmer {
			failure := &annotatedFailure{cause: arguments[0], key: arguments[1].String(), value: arguments[2]}
			if stack := explicitFailureStack(arguments[0]); stack != "" {
				return NewAny(&annotatedStackFailure{annotatedFailure: failure, stack: stack})
			}
			return NewAny(failure)
		},
		Type: &TypeDescriptor{Kind: "func", Const: true,
			Description: "attaches immutable language provenance while retaining the original opaque failure and its native authority",
			Params:      []*TypeDescriptor{{Kind: "any", Label: "failure"}, {Kind: "string", Label: "key"}, {Kind: "any", Label: "value"}},
			Return:      &TypeDescriptor{Kind: "any"},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "failure_annotation",
		Fn:   func(arguments ...Scmer) Scmer { return failureAnnotation(arguments[0], arguments[1].String()) },
		Type: &TypeDescriptor{Kind: "func", Const: true,
			Description: "reads the nearest named annotation through explicit original-failure carriers; returns nil when absent",
			Params:      []*TypeDescriptor{{Kind: "any", Label: "failure"}, {Kind: "string", Label: "key"}},
			Return:      &TypeDescriptor{Kind: "any"},
		},
	})
}
