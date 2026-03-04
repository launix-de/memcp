package storage

import (
	"fmt"
	"github.com/launix-de/memcp/scm"
	"testing"
)

func TestTmpInspectSeq(t *testing.T) {
	n := 100
	values := make([]scm.Scmer, n)
	for i := range values {
		values[i] = scm.NewFloat(float64(i))
	}
	s := buildStorageSeq(values)
	fmt.Println("seqCount", s.seqCount, "count", s.count)
	fmt.Println("recordId bits", s.recordId.bitsize, "offset", s.recordId.offset, "count", s.recordId.count, "cross", s.recordId.crossWord)
	fmt.Println("start bits", s.start.bitsize, "offset", s.start.offset, "count", s.start.count, "cross", s.start.crossWord)
	fmt.Println("stride bits", s.stride.bitsize, "offset", s.stride.offset, "count", s.stride.count, "cross", s.stride.crossWord)
	if len(s.recordId.chunk) > 0 {
		fmt.Println("recordId.chunk[0]", s.recordId.chunk[0])
	}
	if len(s.start.chunk) > 0 {
		fmt.Println("start.chunk[0]", s.start.chunk[0])
	}
	if len(s.stride.chunk) > 0 {
		fmt.Println("stride.chunk[0]", s.stride.chunk[0])
	}
	for i := uint32(0); i < s.seqCount; i++ {
		fmt.Println("rid", i, s.recordId.GetValueUInt(i), "start", s.start.GetValueUInt(i), "stride", s.stride.GetValueUInt(i))
	}
}
