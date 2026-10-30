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

	want := "Column does not exist: `test`.`tbl`.`col`"
	if got := err.Error(); got != want {
		t.Fatalf("scanError.Error() = %q, want %q", got, want)
	}
}

// A scan may overlap the schema owner's eviction/publication work. Access-time
// accounting must use the retained declaration generation, never Columns' mutable
// slice header or backing array.
func TestTouchTempColumnsDuringDeclarationPublication(t *testing.T) {
	retained := &column{Name: "retained", IsTemp: true}
	retired := &column{Name: "retired", IsTemp: true}
	tbl := &table{Columns: []*column{retained, retired}}
	tbl.columnNamesSnapshot.Store(&tableColumnNamesSnapshot{declarations: append([]*column(nil), tbl.Columns...)})
	start, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		<-start
		for i := 0; i < 10000; i++ {
			tbl.Columns = tbl.Columns[:1]
			if i%2 == 0 {
				tbl.Columns = append(tbl.Columns, retired)
			}
			tbl.columnNamesSnapshot.Store(&tableColumnNamesSnapshot{declarations: append([]*column(nil), tbl.Columns...)})
		}
	}()
	close(start)
	for i := 0; i < 10000; i++ {
		touchTempColumns(tbl)
	}
	<-done
	if retained.lastAccessed == 0 {
		t.Fatal("scan did not touch its retained temporary column")
	}
}
