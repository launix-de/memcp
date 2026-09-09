//go:build php

// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later

package phpbridge

import "testing"
import "encoding/binary"

func TestCatalogRejectsMalformedAndAmplifiedStrings(t *testing.T) {
	// Repeated ranges can make a small file allocate huge amounts in an MO
	// parser. Validate the expansion budget before handing it to the library.
	for _, kind := range []string{"magic", "table", "string", "expansion"} {
		t.Run(kind, func(t *testing.T) {
			data := make([]byte, 1<<20)
			put := func(at int, n uint32) { binary.LittleEndian.PutUint32(data[at:], n) }
			put(0, 0x950412de)
			put(8, 1)
			put(12, 28)
			put(16, 36)
			put(28, 1)
			put(32, 44)
			put(36, 1)
			put(40, 46)
			switch kind {
			case "magic":
				put(0, 0)
			case "table":
				put(12, 0xffffffff)
			case "string":
				put(32, 0xffffffff)
			case "expansion":
				put(8, 100)
				put(16, 28)
				for i := 0; i < 100; i++ {
					put(28+i*8, 1<<19)
					put(32+i*8, 1024)
				}
			}
			if _, err := validateMO(data); err == nil {
				t.Fatal("accepted invalid catalog")
			}
		})
	}
}
