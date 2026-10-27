/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package scm

import "sync"
import "testing"
import "bytes"
import "encoding/json"

func TestUUIDGeneratorSerializationHasStableVersion(t *testing.T) {
	for _, name := range []string{"uuid_v1", "uuid_v4", "uuid_v7"} {
		for range 100 {
			var out bytes.Buffer
			Serialize(&out, Globalenv.Vars[Symbol(name)], &Globalenv)
			if got := out.String(); got != name {
				t.Fatalf("serialized %s as %s", name, got)
			}
		}
	}
}

func TestUUIDPackedOperations(t *testing.T) {
	text := "550e8400-e29b-41d4-a716-446655440000"
	v := ParseUUID(NewString("{550E8400-E29B-41D4-A716-446655440000}"))
	if !v.IsUUID() || v.String() != text {
		t.Fatalf("parse: %v", v.String())
	}
	plain := NewString(text)
	if !Equal(v, plain) || HashKey(v) != HashKey(plain) || Less(v, plain) || Less(plain, v) {
		t.Fatal("packed/plain equality, hash or order differs")
	}
	for _, swap := range []int64{0, 1} {
		binary := Globalenv.Vars["uuid_to_bin"].Func()(v, NewInt(swap))
		if len(binary.String()) != 16 {
			t.Fatal("not 16 bytes")
		}
		roundtrip := Globalenv.Vars["uuid_from_bin"].Func()(binary, NewInt(swap))
		if !Equal(v, roundtrip) {
			t.Fatal("binary roundtrip")
		}
	}
	encoded, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var loaded Scmer
	if err = json.Unmarshal(encoded, &loaded); err != nil || !Equal(v, loaded) {
		t.Fatalf("wire roundtrip: %s %v", encoded, err)
	}
	ms := UUIDTimestampMillis(ParseUUID(NewString("017f22e2-79b0-7cc3-98c4-dc0c0c07398f")))
	if ms.Int() != 1645557742000 {
		t.Fatalf("RFC vector timestamp: %v", ms)
	}
	if !UUIDTimestampMillis(v).IsNil() || !ParseUUID(NewNil()).IsNil() {
		t.Fatal("NULL contract")
	}
}

func TestUUIDV7ParallelGeneration(t *testing.T) {
	const workers, rows = 8, 256
	values := make(chan Scmer, workers*rows)
	var pending sync.WaitGroup
	for w := 0; w < workers; w++ {
		pending.Add(1)
		go func() {
			defer pending.Done()
			var previous Scmer
			for i := 0; i < rows; i++ {
				v := GenerateUUID(7)
				if i > 0 && !Less(previous, v) {
					t.Error("local v7 sequence is not increasing")
				}
				previous = v
				values <- v
			}
		}()
	}
	pending.Wait()
	close(values)
	seen := map[string]bool{}
	for v := range values {
		id, err := UUIDValue(v)
		if err != nil || id.Version() != 7 {
			t.Fatalf("v7 generation: %v", err)
		}
		key := id.String()
		if seen[key] {
			t.Fatal("duplicate generated UUID")
		}
		seen[key] = true
	}
}

func TestUUIDRejectsInvalidInputs(t *testing.T) {
	for _, value := range []Scmer{NewString("not-a-uuid"), NewInt(12), NewString("550e8400-e29b-41d4-a716-44665544000x")} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("accepted invalid UUID")
				}
			}()
			ParseUUID(value)
		}()
	}
	for _, version := range []int{1, 4, 7} {
		id, err := UUIDValue(GenerateUUID(version))
		if err != nil || int(id.Version()) != version {
			t.Fatalf("generator %d", version)
		}
	}
}

func TestUUIDPackedOperationsDoNotAllocate(t *testing.T) {
	a := GenerateUUID(7)
	b := GenerateUUID(7)
	if allocations := testing.AllocsPerRun(1000, func() { Equal(a, b); Less(a, b); HashKey(a) }); allocations != 0 {
		t.Fatalf("packed comparison/hash allocated %v", allocations)
	}
	stamp := Globalenv.Vars["uuid_timestamp"].Func()(ParseUUID(NewString("017f22e2-79b0-7cc3-98c4-dc0c0c07398f")))
	if stamp.String() != "2022-02-22 19:22:22.000+00:00" {
		t.Fatal(stamp.String())
	}
}
