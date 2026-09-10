// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later

package storage

import "testing"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

func TestPHPSettings(t *testing.T) {
	keys := []string{"PHPThreads", "PHPMemoryLimit", "PHPMaxWaitMilliseconds", "PHPOutputBuffer", "PHPOpcacheMemory"}
	snapshot := PHPStartupSettings()
	defer ChangeSettings(scm.NewString("PHPIMAPBinary"), scm.NewString(snapshot.IMAPBinary))
	ChangeSettings(scm.NewString("PHPIMAPBinary"), scm.NewString("/usr/bin/php"))
	if PHPStartupSettings().IMAPBinary != "/usr/bin/php" {
		t.Fatal("IMAP startup snapshot lost")
	}
	original := []int64{snapshot.Threads, snapshot.MemoryLimit, snapshot.MaxWaitMilliseconds, snapshot.OutputBuffer, snapshot.OpcacheMemory}
	defer func() {
		for i, key := range keys {
			ChangeSettings(scm.NewString(key), scm.NewInt(original[i]))
		}
	}()
	values := []int64{2, 1024 << 20, 100, 0, 512 << 20}
	for i, key := range keys {
		ChangeSettings(scm.NewString(key), scm.NewInt(values[i]))
		if got := ChangeSettings(scm.NewString(key)).Int(); got != values[i] {
			t.Fatalf("%s: %d", key, got)
		}
	}
	data, err := json.Marshal(Settings)
	if err != nil {
		t.Fatal(err)
	}
	var restored SettingsT
	if err = json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.PHPIMAPBinary != "/usr/bin/php" {
		t.Fatal("IMAP setting lost during persistence")
	}
	if restored.PHPMemoryLimit != 1024<<20 || restored.PHPOpcacheMemory != 512<<20 || restored.PHPThreads != 2 || restored.PHPMaxWaitMilliseconds != 100 || restored.PHPOutputBuffer != 0 {
		t.Fatal("PHP settings lost during persistence")
	}
	for _, tc := range []struct {
		key   string
		value scm.Scmer
	}{
		{"PHPIMAPBinary", scm.NewInt(1)},
		{"PHPThreads", scm.NewInt(0)}, {"PHPThreads", scm.NewFloat(1.5)},
		{"PHPThreads", scm.NewString("4junk")}, {"PHPMemoryLimit", scm.NewInt(-1)},
		{"PHPMemoryLimit", scm.NewInt(1 << 20)}, {"PHPMaxWaitMilliseconds", scm.NewInt(-1)},
		{"PHPOutputBuffer", scm.NewInt(-1)}, {"PHPOpcacheMemory", scm.NewInt(1 << 20)},
	} {
		before := ChangeSettings(scm.NewString(tc.key))
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("accepted invalid %s", tc.key)
				}
			}()
			ChangeSettings(scm.NewString(tc.key), tc.value)
		}()
		if ChangeSettings(scm.NewString(tc.key)) != before {
			t.Errorf("invalid value changed %s", tc.key)
		}
	}
}
