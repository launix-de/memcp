package main

import "math"

type StorageSCMER struct {
	values []scmer
	onlyInt bool
	hasString bool
}

func (s *StorageSCMER) getValue(i uint) scmer {
	return s.values[i]
}

func (s *StorageSCMER) scan(i uint, value scmer) {
	switch v := value.(type) {
		case number:
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
	s.values = make([]scmer, i)
}
func (s *StorageSCMER) build(i uint, value scmer) {
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
