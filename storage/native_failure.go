/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "github.com/launix-de/memcp/scm"

// Diagnostic projection follows only explicit native language-value envelopes.
// It conveys no restoration, recovery or transaction authority.
func nativeFailureOriginalValue(cause any) any {
	for {
		if value, ok := cause.(scm.Scmer); ok {
			cause = value.Any()
			continue
		}
		if origin, ok := cause.(interface{ OriginalFailure() any }); ok {
			cause = origin.OriginalFailure()
			continue
		}
		return cause
	}
}
