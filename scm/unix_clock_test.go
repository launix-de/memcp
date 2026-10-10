/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"testing"
	"time"
)

func TestUnixTickCalendarsExactBoundaries(t *testing.T) {
	for _, fixture := range []struct{ year, month, day, hour, minute, second, remainder, rate int64 }{
		{1, 1, 1, 0, 0, 0, 0, 10000000}, {9999, 12, 31, 23, 59, 59, 9999999, 10000000},
		{1969, 12, 31, 23, 59, 59, 9999999, 10000000}, {1970, 1, 1, 0, 0, 0, 1, 10000000},
		{2024, 2, 29, 13, 45, 30, 1234567, 10000000}, {2024, 2, 29, 0, 0, 0, 1, 300},
	} {
		args := []Scmer{NewInt(fixture.year), NewInt(fixture.month), NewInt(fixture.day), NewInt(fixture.hour), NewInt(fixture.minute), NewInt(fixture.second), NewInt(fixture.remainder), NewInt(fixture.rate)}
		ticks := UnixFromParts(args...)
		parts := UnixParts(ticks, NewInt(fixture.rate)).Slice()
		for i := 0; i < 7; i++ {
			if parts[i].Int() != args[i].Int() {
				t.Fatal("Gregorian fields lost exact ticks", fixture, i)
			}
		}
		if !ticks.IsInt() {
			t.Fatal("Unix ticks became a custom runtime type")
		}
	}
	expectTDSPanic(t, func() {
		UnixFromParts(NewInt(2023), NewInt(2), NewInt(29), NewInt(0), NewInt(0), NewInt(0), NewInt(0), NewInt(10000000))
	})
	expectTDSPanic(t, func() { UnixParts(NewNil(), NewInt(10000000)) })
	expectTDSPanic(t, func() {
		UnixFromParts(NewInt(9999), NewInt(12), NewInt(31), NewInt(23), NewInt(59), NewInt(59), NewInt(0), NewInt(1000000000))
	})
}
func TestUnixLayoutAndOffsetAreExplicit(t *testing.T) {
	value := UnixParse(NewString("2024-02-29T13:45:30.1234567+02:30"), NewString("2006-01-02T15:04:05.9999999Z07:00"), NewInt(10000000), NewInt(0))
	got := UnixFormat(value, NewInt(10000000), NewString("2006-01-02T15:04:05.0000000Z07:00"), NewInt(150)).String()
	if got != "2024-02-29T13:45:30.1234567+02:30" {
		t.Fatal(got)
	}
	before := time.Now().Unix()
	clock := UnixClock(NewInt(10000000))
	after := time.Now().Unix()
	if clock.Int()/10000000 < before || clock.Int()/10000000 > after {
		t.Fatal("clock isn't Unix time")
	}
	if !NewDate(0).IsDate() {
		t.Fatal("existing whole-second date type changed")
	}
}

func TestUnixZoneOffsetsFollowInstant(t *testing.T) {
	for _, fixture := range []struct {
		text string
		want int64
	}{{"2024-01-15", 3600}, {"2024-07-15", 7200}} {
		instant := UnixParse(NewString(fixture.text), NewString("2006-01-02"), NewInt(10000000), NewInt(0))
		if got := UnixZoneOffset(instant, NewInt(10000000), NewString("Europe/Berlin")).Int(); got != fixture.want {
			t.Fatal("zone offset ignored instant", fixture, got)
		}
		if got := UnixZoneOffset(instant, NewInt(10000000), NewString("UTC")).Int(); got != 0 {
			t.Fatal("UTC offset changed", got)
		}
	}
	expectTDSPanic(t, func() { UnixZoneOffset(NewInt(0), NewInt(1), NewString("not/a/zone")) })
}
