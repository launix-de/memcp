//go:build ceph

/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

// The shared literal-frame and writer tests also cover Ceph when its optional
// native backend is built. Default builds do not link librados.
func init() {
	nativeDateWALCodecs = append(nativeDateWALCodecs, nativeDateWALCodec{"ceph", encodeLogEntry, decodeLogStream})
}
