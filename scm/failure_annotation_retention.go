/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import "reflect"

// RetainFailureAnnotations carries language metadata back to an already latched
// native cause. Only exact opaque identity proves that a later unwind describes
// that cause. Equal text or equal-looking language values convey no such proof.
// Completion, cleanup and other native envelopes are traversed, never replayed.
func RetainFailureAnnotations(original, incoming any) any {
	identity := failureIdentityOrigin(original)
	stack := explicitFailureStack(FromAny(original))
	var annotations []*annotatedFailure
	for {
		if carrier, ok := incoming.(interface{ OriginalStack() string }); ok && stack == "" {
			stack = carrier.OriginalStack()
		}
		if sameFailureIdentity(original, incoming) || sameFailureIdentity(identity, incoming) {
			retained := original
			for i := len(annotations) - 1; i >= 0; i-- {
				annotation := annotations[i]
				copy := &annotatedFailure{cause: FromAny(retained), key: annotation.key, value: annotation.value}
				if stack != "" {
					retained = &annotatedStackFailure{annotatedFailure: copy, stack: stack}
				} else {
					retained = copy
				}
			}
			return retained
		}
		if value, ok := incoming.(Scmer); ok {
			if value.GetTag() != tagAny {
				return original
			}
			incoming = value.Any()
			continue
		}
		switch failure := incoming.(type) {
		case *annotatedFailure:
			annotations = append(annotations, failure)
		case *annotatedStackFailure:
			annotations = append(annotations, failure.annotatedFailure)
		}
		if carrier, ok := incoming.(interface{ OriginalFailure() any }); ok {
			incoming = carrier.OriginalFailure()
		} else if carrier, ok := incoming.(interface{ Unwrap() error }); ok {
			incoming = carrier.Unwrap()
		} else {
			return original
		}
	}
}

func sameFailureIdentity(original, incoming any) bool {
	if value, ok := original.(Scmer); ok {
		if value.GetTag() == tagAny {
			return sameFailureIdentity(value.Any(), incoming)
		}
		other, ok := incoming.(Scmer)
		// A nonempty list is an existing opaque language container. Scalars,
		// including strings and the canonical empty list, have no cause identity.
		return ok && value.IsSlice() && len(value.Slice()) != 0 && value == other
	}
	if value, ok := incoming.(Scmer); ok && value.GetTag() == tagAny {
		return sameFailureIdentity(original, value.Any())
	}
	value := reflect.ValueOf(original)
	return value.IsValid() && value.Kind() == reflect.Pointer && !value.IsNil() &&
		reflect.TypeOf(incoming) == value.Type() && original == incoming
}

// Native scan/completion carriers may be value envelopes rather than objects.
// Their explicit original-cause chain still reaches the same opaque datum;
// retain the entire original envelope while using only that datum as proof.
func failureIdentityOrigin(value any) any {
	for {
		if language, ok := value.(Scmer); ok && language.GetTag() == tagAny {
			value = language.Any()
		} else if carrier, ok := value.(interface{ OriginalFailure() any }); ok {
			value = carrier.OriginalFailure()
		} else if carrier, ok := value.(interface{ Unwrap() error }); ok {
			value = carrier.Unwrap()
		} else {
			return value
		}
	}
}
