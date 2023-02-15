/*
Copyright (C) 2023  Carl-Philip Hänsch

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

import "fmt"
import "time"
import "runtime"
import "strings"
import "github.com/launix-de/memcp/scm"

// THE basic storage pattern
type ColumnStorage interface {
	getValue(uint) scm.Scmer // read function
	String() string // self-description
	// buildup functions 1) prepare 2) scan, 3) proposeCompression(), if != nil repeat at 1, 4) init, 5) build; all values are passed through twice
	// analyze
	prepare()
	scan(uint, scm.Scmer)
	proposeCompression() ColumnStorage

	// store
	init(uint)
	build(uint, scm.Scmer)
	finish()
}

func Init(en scm.Env) {
	en.Vars["scan"] = func (a ...scm.Scmer) scm.Scmer {
		start := time.Now() // time measurement

		// params: table, condition, map, reduce, reduceSeed
		t := tables[scm.String(a[0])][scm.String(a[1])]
		var aggregate scm.Scmer
		var neutral scm.Scmer
		if len(a) > 4 {
			aggregate = a[4]
		}
		if len(a) > 5 {
			neutral = a[5]
		}
		result := t.scan(a[2], a[3], aggregate, neutral)
		fmt.Println("scan", time.Since(start))
		return result
	}
	en.Vars["createdatabase"] = func (a ...scm.Scmer) scm.Scmer {
		CreateDatabase(scm.String(a[0]))
		return "ok"
	}
	en.Vars["dropdatabase"] = func (a ...scm.Scmer) scm.Scmer {
		DropDatabase(scm.String(a[0]))
		return "ok"
	}
	en.Vars["createtable"] = func (a ...scm.Scmer) scm.Scmer {
		// params: tablename, (columndefs) mit (name, typ)
		t := CreateTable(scm.String(a[0]), scm.String(a[1]))
		for _, coldef := range(a[2].([]scm.Scmer)) {
			colname := scm.String(coldef.([]scm.Scmer)[0])
			typename := scm.String(coldef.([]scm.Scmer)[1])
			dimensions_ := coldef.([]scm.Scmer)[2].([]scm.Scmer)
			dimensions := make([]int, len(dimensions_))
			for i, d := range dimensions_ {
				dimensions[i] = scm.ToInt(d)
			}
			typeparams := scm.String(coldef.([]scm.Scmer)[3])
			t.CreateColumn(colname, typename, dimensions, typeparams)
			// todo: not null flags usw
		}
		return "ok"
	}
	en.Vars["droptable"] = func (a ...scm.Scmer) scm.Scmer {
		DropTable(scm.String(a[0]), scm.String(a[1]))
		return "ok"
	}
	en.Vars["insert"] = func (a ...scm.Scmer) scm.Scmer {
		tables[scm.String(a[0])][scm.String(a[1])].Insert(dataset(a[2].([]scm.Scmer)))
		return "ok"
	}
	en.Vars["stat"] = func (a ...scm.Scmer) scm.Scmer {
		return PrintMemUsage()
	}
	en.Vars["rebuild"] = func (a ...scm.Scmer) scm.Scmer {
		start := time.Now()

		for _, database := range tables {
			for _, t := range database {
				t.mu.Lock() // schema lock
				for i, s := range t.shards {
					// TODO: go + chan done
					t.shards[i] = s.rebuild()
				}
				t.mu.Unlock() // TODO: do this after chan done??
			}
		}

		return fmt.Sprint(time.Since(start))
	}
	en.Vars["loadCSV"] = func (a ...scm.Scmer) scm.Scmer {
		// table, filename, delimiter
		start := time.Now()

		delimiter := ";"
		if len(a) > 3 {
			delimiter = scm.String(a[3])
		}
		LoadCSV(scm.String(a[0]), scm.String(a[1]), scm.String(a[2]), delimiter)

		return fmt.Sprint(time.Since(start))
	}
	en.Vars["loadJSON"] = func (a ...scm.Scmer) scm.Scmer {
		start := time.Now()

		LoadJSON(scm.String(a[0]), scm.String(a[1]))

		return fmt.Sprint(time.Since(start))
	}
}

func PrintMemUsage() string {
	runtime.GC()
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        // For info on each, see: https://golang.org/pkg/runtime/#MemStats
	var b strings.Builder
        b.WriteString(fmt.Sprintf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v", bToMb(m.Alloc), bToMb(m.TotalAlloc), bToMb(m.Sys), m.NumGC))
	return b.String()
}

func bToMb(b uint64) uint64 {
    return b / 1024 / 1024
}
