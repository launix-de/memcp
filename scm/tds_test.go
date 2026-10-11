/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import "testing"

func TestTDSParameterScalarCarrier(t *testing.T) {
	integer := tdsValueData(int64(9007199254740993))
	if !integer.IsInt() || integer.Int() != 9007199254740993 {
		t.Fatal("integer parameter lost its native representation or precision")
	}
	if value := tdsValueData(nil); !value.IsNil() {
		t.Fatal("NULL parameter became an opaque value")
	}
	if value := tdsValueData(true); !value.IsBool() || !value.Bool() {
		t.Fatal("Boolean parameter became an opaque value")
	}
	if value := tdsValueData(2.5); !value.IsFloat() || value.Float() != 2.5 {
		t.Fatal("floating parameter became an opaque value")
	}
	if value := tdsValueData("Grüße 🐘"); !value.IsString() || value.String() != "Grüße 🐘" {
		t.Fatal("text parameter lost its native representation")
	}
}
