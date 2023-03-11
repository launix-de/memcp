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

import "github.com/launix-de/memcp/scm"

type StorageSparse struct {
	values map[uint]scm.Scmer
}

func (s *StorageSparse) String() string {
	return "SCMER-sparse"
}

func (s *StorageSparse) getValue(i uint) scm.Scmer {
	return s.values[i]
}

func (s *StorageSparse) scan(i uint, value scm.Scmer) {
}
func (s *StorageSparse) prepare() {
}
func (s *StorageSparse) init(i uint) {
	s.values = make(map[uint]scm.Scmer)
}
func (s *StorageSparse) build(i uint, value scm.Scmer) {
	// store
	if value != nil {
		s.values[i] = value
	}
}
func (s *StorageSparse) finish() {
}

// soley to StorageSparse
func (s *StorageSparse) proposeCompression(i uint) ColumnStorage {
	return nil
}
