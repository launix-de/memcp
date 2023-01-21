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
	count uint
	null uint
}

func (s *StorageSCMER) String() string {
	return "SCMER"
}

func (s *StorageSCMER) getValue(i uint) scm.Scmer {
	return s.values[i]
}

func (s *StorageSCMER) scan(i uint, value scm.Scmer) {
	s.count = s.count + 1
	switch v := value.(type) {
		case float64:
			if _, f := math.Modf(v); f != 0.0 {
				s.onlyInt = false
			}
		case string:
			s.onlyInt = false
			s.hasString = true
		case nil:
			s.null = s.null + 1 // count NULL
			// storageInt can also handle null
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
	if s.null * 13 > s.count * 100 {
		// sparse payoff against bitcompressed is at ~13%
		return new(StorageSparse)
	}
	if s.hasString {
		return new(StorageString)
	}
	if s.onlyInt {
		return new(StorageInt)
	}
	if s.null * 50 > s.count * 100 {
		// sparse payoff against StorageSCMER is at 2.1
		return new(StorageSparse)
	}
	// dont't propose another pass
	return nil
}