/*
Copyright (C) 2023-2026 Launix GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

package storage

import (
	"testing"
	"time"

	"github.com/launix-de/NonLockingReadMap"
)

func TestDropTriggerDoesNotHoldSchemaLockWhileWaitingForTableDDL(t *testing.T) {
	db := &database{Name: "trigger-lock-order", srState: COLD}
	db.tables = NonLockingReadMap.New[table, string]()
	table := &table{Name: "items", schema: db}
	table.Triggers = []TriggerDescription{{Name: "items_after_drop"}}
	db.tables.Set(table)

	table.ddlMu.Lock()
	dropped := make(chan bool, 1)
	go func() { dropped <- db.dropTrigger("items_after_drop") }()

	// Give dropTrigger time to take its catalog snapshot and wait on ddlMu.
	time.Sleep(20 * time.Millisecond)
	schemaAvailable := make(chan struct{}, 1)
	go func() {
		db.schemalock.Lock()
		db.schemalock.Unlock()
		schemaAvailable <- struct{}{}
	}()

	select {
	case <-schemaAvailable:
	case <-time.After(2 * time.Second):
		table.ddlMu.Unlock()
		t.Fatal("dropTrigger held schemalock while waiting for table ddlMu")
	}

	table.ddlMu.Unlock()
	select {
	case ok := <-dropped:
		if !ok {
			t.Fatal("trigger was not removed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("dropTrigger did not finish after ddlMu was released")
	}
}
