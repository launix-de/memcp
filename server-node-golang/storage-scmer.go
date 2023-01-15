package main

import "math"

type StorageSCMER struct {
	values []scmer
	onlyInt bool
}

func (s *StorageSCMER) getValue(i uint) scmer {
	return s.values[i]
}

func (s *StorageSCMER) scan(i uint, value scmer) {
	switch value.(type) {
		case number:
			if _, f := math.Modf(float64(value.(number))); f != 0.0 {
				s.onlyInt = false
			}
		case float64:
			if _, f := math.Modf(value.(float64)); f != 0.0 {
				s.onlyInt = false
			}
		default:
			s.onlyInt = false
	}
}
func (s *StorageSCMER) prepare() {
	s.onlyInt = true
}
func (s *StorageSCMER) init(i uint) {
	// allocate
	s.values = make([]scmer, i)
}
func (s *StorageSCMER) build(i uint, value scmer) {
	// store
	s.values[i] = value
}

// soley to StorageSCMER
func (s *StorageSCMER) proposeCompression() ColumnStorage {
	if s.onlyInt {
		return new(StorageInt)
	}
	// dont't propose another pass
	return nil
}
