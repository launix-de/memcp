/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "strings"
import "github.com/launix-de/memcp/scm"

// A mutation producer declares its operation before visiting any candidates.
// This is native write attribution, not a SQL syntax/row-count inference: even
// an empty producer has exactly one declared operation per target.
type nativeMutationEvent uint8

// Bounded producer metadata, independent of row cardinality. Names describe
// explicit assignments; they are never inferred from OLD/NEW equality. Other
// event producers may provide no assignment names. Frontend policy decides how
// to interpret this metadata for its language.
type nativeMutationDescriptor struct {
	event   nativeMutationEvent
	columns []string
}

func parseNativeMutationDescriptor(value scm.Scmer) nativeMutationDescriptor {
	if !value.IsSlice() || len(value.Slice()) != 2 || !value.Slice()[0].IsString() || !value.Slice()[1].IsSlice() {
		panic("native mutation descriptor requires an event and assignment-name list")
	}
	fields := value.Slice()
	descriptor := nativeMutationDescriptor{event: parseNativeMutationEvent(fields[0].String())}
	seen := make(map[string]struct{}, len(fields[1].Slice()))
	for _, field := range fields[1].Slice() {
		if !field.IsString() || field.String() == "" {
			panic("native mutation assignment names must be nonempty strings")
		}
		name := field.String()
		if _, repeated := seen[name]; !repeated {
			name = strings.Clone(name)
			descriptor.columns = append(descriptor.columns, name)
			seen[name] = struct{}{}
		}
	}
	return descriptor
}

const (
	nativeMutationInsert nativeMutationEvent = iota + 1
	nativeMutationUpdate
	nativeMutationDelete
)

func (event nativeMutationEvent) String() string {
	switch event {
	case nativeMutationInsert:
		return "insert"
	case nativeMutationUpdate:
		return "update"
	case nativeMutationDelete:
		return "delete"
	default:
		panic("unknown native mutation event")
	}
}

func parseNativeMutationEvent(name string) nativeMutationEvent {
	switch name {
	case "insert":
		return nativeMutationInsert
	case "update":
		return nativeMutationUpdate
	case "delete":
		return nativeMutationDelete
	default:
		panic(fmt.Sprintf("unknown native mutation event: %q", name))
	}
}
