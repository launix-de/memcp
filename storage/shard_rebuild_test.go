/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package storage

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/launix-de/memcp/scm"
)

func TestShardRebuildForwardsConcurrentInsertsViaNext(t *testing.T) {
	dir, err := os.MkdirTemp("", "memcp-shard-rebuild-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	oldBasepath := Basepath
	Basepath = dir
	defer func() { Basepath = oldBasepath }()

	Init(scm.Globalenv)
	LoadDatabases()
	defer databases.Remove("trebuildnext")

	CreateDatabase("trebuildnext", false)
	tbl, _ := CreateTable("trebuildnext", "items", Safe, false)
	tbl.CreateColumn("id", "INT", nil, nil)
	tbl.CreateColumn("payload", "TEXT", nil, nil)

	initialRows := make([][]scm.Scmer, 0, 20000)
	for i := 0; i < 20000; i++ {
		initialRows = append(initialRows, []scm.Scmer{
			scm.NewInt(int64(i + 1)),
			scm.NewString(fmt.Sprintf("%032x", i+1)),
		})
	}
	tbl.Insert([]string{"id", "payload"}, initialRows, nil, scm.NewNil(), false, nil)

	shard := tbl.Shards[0]
	rebuiltCh := make(chan *storageShard, 1)
	go func() {
		rebuiltCh <- shard.rebuild(true)
	}()

	deadline := time.Now().Add(3 * time.Second)
	for shard.loadNext() == nil {
		if time.Now().After(deadline) {
			t.Fatal("rebuild never published next shard")
		}
		runtime.Gosched()
	}

	extraRows := make([][]scm.Scmer, 0, 2000)
	for i := 0; i < 2000; i++ {
		extraRows = append(extraRows, []scm.Scmer{
			scm.NewInt(int64(20001 + i)),
			scm.NewString(fmt.Sprintf("%032x", 20001+i)),
		})
	}
	shard.Insert([]string{"id", "payload"}, extraRows, false, nil, false)

	rebuilt := <-rebuiltCh
	if rebuilt == nil {
		t.Fatal("rebuild returned nil shard")
	}
	if got, want := rebuilt.Count(), uint32(len(initialRows)+len(extraRows)); got != want {
		t.Fatalf("rebuilt shard count = %d, want %d", got, want)
	}
}
