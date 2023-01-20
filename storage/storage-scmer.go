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
import "github.com/launix-de/cpdb/scm"

type StorageSCMER struct {
	values []scm.Scmer
	onlyInt bool
	hasString bool
}

func (s *StorageSCMER) getValue(i uint) scm.Scmer {
	return s.values[i]
}

func (s *StorageSCMER) scan(i uint, value scm.Scmer) {
	switch v := value.(type) {
		case scm.Number:
			if _, f := math.Modf(float64(v)); f != 0.0 {
				s.onlyInt = false
			}
		case float64:
			if _, f := math.Modf(v); f != 0.0 {
				s.onlyInt = false
			}
		case string:
			s.onlyInt = false
			s.hasString = true
		default:
			s.onlyInt = false
	}
}
func (s *StorageSCMER) prepare() {
	s.onlyInt = true
	s.hasString = false
}
func (s *StorageSCMER) init(i uint) {
	// allocate
	s.values = make([]scm.Scmer, i)
}
func (s *StorageSCMER) build(i uint, value scm.Scmer) {
	// store
	s.values[i] = value
}
func (s *StorageSCMER) finish() {
}

// soley to StorageSCMER
func (s *StorageSCMER) proposeCompression() ColumnStorage {
	if s.hasString {
		return new(StorageString)
	}
	if s.onlyInt {
		return new(StorageInt)
	}
	// dont't propose another pass
	return nil
}
