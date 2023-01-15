package main

//import "fmt"

type StorageString struct {
	dictionary []string
	// StorageInt for dictionary entries
}

func (s *StorageString) getValue(i uint) scmer {
	return s.dictionary[i]
}

func (s *StorageString) prepare() {
	// set up scan
}
func (s *StorageString) scan(i uint, value scmer) {
	// storage is so simple, dont need scan
}
func (s *StorageString) init(i uint) {
	// allocate
	s.dictionary = make([]string, i)
}
func (s *StorageString) build(i uint, value scmer) {
	// store
	switch v := value.(type) {
		case string:
			s.dictionary[i] = v
		default:
			s.dictionary[i] = "NULL" // TODO: proper null representation
	}
}
func (s *StorageString) proposeCompression() ColumnStorage {
	// dont't propose another pass
	return nil
}

