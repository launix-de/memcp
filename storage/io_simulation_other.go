//go:build !linux

/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "time"
import "context"

func waitSubMillisecondIO(context.Context, time.Time) (ready, supported bool) {
	return false, false
}
