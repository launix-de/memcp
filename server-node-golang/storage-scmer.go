package main

import "fmt"

type StorageSCMER struct {
	values []scmer
}

func (s *StorageSCMER) getValue(i int) scmer {
	return s.values[i]
}

func (s *StorageSCMER) scan(i int, value scmer) {
	// storage is so simple, dont need scan
}
func (s *StorageSCMER) init(i int) {
	// allocate
	s.values = make([]scmer, i)
	fmt.Println("scan %d", i)
}
func (s *StorageSCMER) build(i int, value scmer) {
	// store
	s.values[i] = value
}
