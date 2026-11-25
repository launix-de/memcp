/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "github.com/launix-de/memcp/scm"

// A finite negative declaration-profile witness is semantic data, like an
// actual column-name conflict. It contains neither SQL policy/error numbers
// nor statement-restoration authority. Unknown evaluator/I/O failures keep
// their opaque original causes and native completion remains authoritative.
type catalogDeclarationProfileConflict struct {
	message    string
	diagnostic scm.Scmer
}

func (failure *catalogDeclarationProfileConflict) Error() string { return failure.message }
func (failure *catalogDeclarationProfileConflict) OriginalFailure() any {
	return failure.diagnostic
}

func newCatalogDeclarationProfileConflict(kind, message string) *catalogDeclarationProfileConflict {
	return &catalogDeclarationProfileConflict{message: message, diagnostic: scm.NewSlice([]scm.Scmer{
		scm.NewString("catalog-declaration-conflict"), scm.NewString(kind), scm.NewString(message),
	})}
}
