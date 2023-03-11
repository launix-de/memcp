/*
Copyright (C) 2023  Carl-Philip Hänsch

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

import "math"
import "github.com/launix-de/memcp/scm"

// main type for storage: can store any value, is inefficient but does type analysis how to optimize
type StorageFloat struct {
	values []float64
}

func (s *StorageFloat) String() string {
	return "float64"
}

func (s *StorageFloat) getValue(i uint) scm.Scmer {
	// NULL is encoded as NaN in SQL
	if math.IsNaN(s.values[i]) {
		return nil
	} else {
		return s.values[i]
	}
}

func (s *StorageFloat) scan(i uint, value scm.Scmer) {
}
func (s *StorageFloat) prepare() {
}
func (s *StorageFloat) init(i uint) {
	// allocate
	s.values = make([]float64, i)
}
func (s *StorageFloat) build(i uint, value scm.Scmer) {
	// store
	if value == nil {
		s.values[i] = math.NaN()
	} else {
		s.values[i] = value.(float64)
	}
}
func (s *StorageFloat) finish() {
}

func (s *StorageFloat) proposeCompression(i uint) ColumnStorage {
	// dont't propose another pass
	return nil
}

