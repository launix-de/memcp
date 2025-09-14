/*
Copyright (C) 2024  Carl-Philip Hänsch

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
package scm

import "io"
import "os"
import "fmt"
import "sync"
import "time"
import "encoding/json"

type Tracefile struct {
	isFirst bool
	file    io.WriteCloser
	m       sync.Mutex
}

var Trace *Tracefile // default trace: set to not nil if you want to trace
var TracePrint bool  // whether to print traces to stdout

func SetTrace(on bool) { // sets Trace to nil or a value
	if Trace != nil {
		Trace.Close()
		Trace = nil
	}
	if on {
		// TODO: tracefolder
		f, err := os.Create(os.Getenv("MEMCP_TRACEDIR") + "trace_" + fmt.Sprint(time.Now().Unix()) + ".json")
		if err != nil {
			panic(err)
		}
		Trace = NewTrace(f)
	}
}

func NewTrace(file io.WriteCloser) *Tracefile {
	file.Write([]byte("["))
	result := new(Tracefile)
	result.file = file
	result.isFirst = true
	return result
}

func (t *Tracefile) Close() {
	t.file.Write([]byte("]"))
	t.file.Close()
}

func (t *Tracefile) Duration(name string, cat string, f func()) {
	t.EventHalf(name, cat, "B", 0, 0)
	defer t.EventHalf(name, cat, "E", 0, 0)
	f()
}

func (t *Tracefile) Event(name string, cat string, typ string) {
	t.EventHalf(name, cat, typ, 0, 0)
}

func (t *Tracefile) EventHalf(name string, cat string, typ string, tid int, pid int) {
	ts := time.Since(start).Microseconds()
	t.EventFull(name, cat, typ, ts, tid, pid)
}

/*
*

	@name string function
	@cat string comma separated categories (for filtering)
	@typ B/E for begin/end, X for events
	@ts timestamp in microseconds
	@pid process id
	@tid thread id
	@args ??
*/
func (t *Tracefile) EventFull(name string, cat string, typ string, ts int64, tid int, pid int) {
	t.m.Lock()
	if t.isFirst {
		t.isFirst = false
	} else {
		t.file.Write([]byte(",\n"))
	}
	t.file.Write([]byte("{\"name\": "))
	b, _ := json.Marshal(name) // name
	t.file.Write(b)
	t.file.Write([]byte(", \"cat\": "))
	b, _ = json.Marshal(cat) // cat
	t.file.Write(b)
	t.file.Write([]byte(", \"ph\": \""))
	t.file.Write([]byte(typ))
	t.file.Write([]byte("\", \"ts\": "))
	b, _ = json.Marshal(ts) // ts
	t.file.Write(b)
	t.file.Write([]byte(", \"pid\": "))
	b, _ = json.Marshal(pid) // pid
	t.file.Write(b)
	t.file.Write([]byte(", \"tid\": "))
	b, _ = json.Marshal(tid) // tid
	t.file.Write(b)
	t.file.Write([]byte(", \"s\": \"g\"}"))
	t.m.Unlock()
}

var start time.Time = time.Now()
