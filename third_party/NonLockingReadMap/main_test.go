/*
Copyright (C) 2024  Carl-Philip Hänsch

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

package NonLockingReadMap

import "testing"

type testItem struct{ k string }

func (t *testItem) ComputeSize() uint { return 0 }
func (t *testItem) GetKey() string    { return t.k }

func TestSetDoesNotDuplicateKeys(t *testing.T) {
	m := New[testItem, string]()
	m.Set(&testItem{k: "x"})
	m.Set(&testItem{k: "x"})
	if got := len(m.GetAll()); got != 1 {
		t.Fatalf("expected 1 item, got %d", got)
	}
}
