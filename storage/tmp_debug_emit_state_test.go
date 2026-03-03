/*
Copyright (C) 2026  Carl-Philip Hänsch
*/
package storage

import (
	"testing"
	"time"

	"github.com/launix-de/memcp/scm"
)

func TestTmpStringJITRawWords(t *testing.T) {
	values := []scm.Scmer{
		scm.NewString("hello"), scm.NewString("world"), scm.NewString("foo"),
		scm.NewString("bar"), scm.NewString("baz"),
	}
	s := buildStorageString(values)
	jitGet, cleanup := jitBuildGetValueFunc(t, s, true)
	defer cleanup()

	for i := 0; i < len(values); i++ {
		got := jitGet(int64(i))
		exp := s.GetValue(uint32(i))
		g0, g1 := got.RawWords()
		e0, e1 := exp.RawWords()
		t.Logf("i=%d got=%#x/%#x exp=%#x/%#x gotStr=%q expStr=%q", i, g0, g1, e0, e1, got.String(), exp.String())
	}
}

func TestTmpSeqJITSingleCall(t *testing.T) {
	values := make([]scm.Scmer, 100)
	for i := range values {
		values[i] = scm.NewFloat(float64(i))
	}
	s := buildStorageSeq(values)
	jitGet, cleanup := jitBuildGetValueFunc(t, s, true)
	defer cleanup()

	done := make(chan scm.Scmer, 1)
	go func() {
		done <- jitGet(0)
	}()

	select {
	case v := <-done:
		t.Logf("jitGet(0)=%v", v)
	case <-time.After(2 * time.Second):
		t.Fatal("jitGet(0) timeout")
	}
}

func TestTmpSeqJITFindHangIndex(t *testing.T) {
	values := make([]scm.Scmer, 100)
	for i := range values {
		values[i] = scm.NewFloat(float64(i))
	}
	s := buildStorageSeq(values)
	jitGet, cleanup := jitBuildGetValueFunc(t, s, true)
	defer cleanup()

	for i := 0; i < len(values); i++ {
		done := make(chan scm.Scmer, 1)
		go func(idx int) {
			done <- jitGet(int64(idx))
		}(i)

		select {
		case v := <-done:
			if i < 5 || i > len(values)-5 {
				t.Logf("i=%d v=%v", i, v)
			}
		case <-time.After(200 * time.Millisecond):
			t.Fatalf("hang at index %d", i)
		}
	}
}

func TestTmpSeqJITStateCorruption(t *testing.T) {
	values := make([]scm.Scmer, 100)
	for i := range values {
		values[i] = scm.NewFloat(float64(i))
	}
	s := buildStorageSeq(values)
	jitGet, cleanup := jitBuildGetValueFunc(t, s, true)
	defer cleanup()

	for i := 0; i < len(values); i++ {
		got := jitGet(int64(i))
		pivot := s.lastValue.Load()
		t.Logf("idx=%d got=%v pivot=%d seqCount=%d", i, got, pivot, s.seqCount)
		if pivot < 0 || pivot > int64(s.seqCount)+8 {
			t.Fatalf("after jit idx=%d: pivot=%d seqCount=%d", i, pivot, s.seqCount)
		}
		done := make(chan scm.Scmer, 1)
		go func(idx int) { done <- s.GetValue(uint32(idx)) }(i)
		select {
		case <-done:
		case <-time.After(300 * time.Millisecond):
			t.Fatalf("GetValue hang after jit idx=%d pivot=%d seqCount=%d", i, pivot, s.seqCount)
		}
	}
}
