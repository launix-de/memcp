package storage

import (
	"fmt"
	"testing"
	"github.com/launix-de/memcp/scm"
)

func TestTmpInspectSeq(t *testing.T) {
	n := 100
	values := make([]scm.Scmer, n)
	for i := range values { values[i] = scm.NewFloat(float64(i)) }
	s := buildStorageSeq(values)
	fmt.Println("seqCount", s.seqCount, "count", s.count)
	fmt.Println("recordId bits", s.recordId.bitsize, "offset", s.recordId.offset, "count", s.recordId.count, "cross", s.recordId.crossWord)
	fmt.Println("start bits", s.start.bitsize, "offset", s.start.offset, "count", s.start.count, "cross", s.start.crossWord)
	fmt.Println("stride bits", s.stride.bitsize, "offset", s.stride.offset, "count", s.stride.count, "cross", s.stride.crossWord)
	for i:=uint32(0); i<s.seqCount; i++ { fmt.Println("rid",i,s.recordId.GetValueUInt(i),"start",s.start.GetValueUInt(i),"stride",s.stride.GetValueUInt(i)) }
}
