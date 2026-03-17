/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package storage

import "testing"

func TestScanErrorOmitsStackFromClientError(t *testing.T) {
	err := scanError{
		r:     "Column does not exist: `test`.`tbl`.`col`",
		stack: "goroutine 1 [running]:\nexample stack",
	}

	if got, want := err.Error(), "Column does not exist: `test`.`tbl`.`col`"; got != want {
		t.Fatalf("scanError.Error() = %q, want %q", got, want)
	}
}
