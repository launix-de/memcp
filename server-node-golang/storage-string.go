package main

//import "fmt"
import "strings"

type StorageString struct {
	dictionary string
	// StorageInt for dictionary entries
	starts StorageInt
	ends StorageInt
	// helpers
	sb strings.Builder
	reverseMap map[string]uint
}

func (s *StorageString) getValue(i uint) scmer {
	return s.dictionary[s.starts.getValueUInt(i):s.ends.getValueUInt(i)]
}

func (s *StorageString) prepare() {
	// set up scan
	s.starts.prepare()
	s.ends.prepare()
	s.reverseMap = make(map[string]uint)
}
func (s *StorageString) scan(i uint, value scmer) {
	// storage is so simple, dont need scan
	var v string
	switch v_ := value.(type) {
		case string:
			v = v_
		default:
			v = "NULL" // TODO: proper null representation
	}
	start, ok := s.reverseMap[v]
	if ok {
		// reuse of string
	} else {
		// learn
		start = uint(s.sb.Len())
		s.sb.WriteString(v)
		s.reverseMap[v] = start
	}
	s.starts.scan(i, number(start))
	s.ends.scan(i, number(start + uint(len(v))))
}
func (s *StorageString) init(i uint) {
	// allocate
	s.dictionary = s.sb.String() // extract string from stringbuilder
	s.sb.Reset() // free the memory
	// prefixed strings are not accounted with that, but maybe this could be checked later??
	s.starts.init(i)
	s.ends.init(i)
}
func (s *StorageString) build(i uint, value scmer) {
	// store
	var v string
	switch v_ := value.(type) {
		case string:
			v = v_
		default:
			v = "NULL" // TODO: proper null representation
	}
	start := s.reverseMap[v]
	// write start+end into sub storage maps
	s.starts.build(i, number(start))
	s.ends.build(i, number(start + uint(len(v))))
}
func (s *StorageString) finish() {
	s.reverseMap = make(map[string]uint) // free memory for reverse
	s.starts.finish()
	s.ends.finish()
}
func (s *StorageString) proposeCompression() ColumnStorage {
	// dont't propose another pass
	return nil
}

