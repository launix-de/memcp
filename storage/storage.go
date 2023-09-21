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

import "os"
import "fmt"
import "time"
import "reflect"
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
	proposeCompression(i uint) ColumnStorage

	// store
	init(uint)
	build(uint, scm.Scmer)
	finish()

	// persistency (the callee takes ownership of the file handle, so he can close it immediately or set a finalizer)
	Serialize(*os.File) // write content to file (and maybe swap the old content out of ram) (must set finalizer if file is kept open)
	Deserialize(*os.File) uint // read from file (or swap in) (note that first byte is already read)
}

var storages = map[uint8]reflect.Type {
	 1: reflect.TypeOf(StorageSCMER{}),
	 2: reflect.TypeOf(StorageSparse{}),
	10: reflect.TypeOf(StorageInt{}),
	11: reflect.TypeOf(StorageSeq{}),
	12: reflect.TypeOf(StorageFloat{}),
	20: reflect.TypeOf(StorageString{}),
	21: reflect.TypeOf(StoragePrefix{}),
}

func Init(en scm.Env) {
	scm.DeclareTitle("Storage")
	scm.Declare(&en, &scm.Declaration{
		"scan", "does an unordered parallel filer-map-reduce pass on a single table and returns the reduced result",
		4, 6,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "database where the table is located"},
			scm.DeclarationParameter{"table", "string", "name of the table to scan"},
			scm.DeclarationParameter{"filter", "func", "lambda function that decides whether a dataset is passed to the map phase. You can use any column of that table as lambda parameter. You should structure your lambda with an (and) at the root element. Every equal? < > <= >= will possibly translated to an indexed scan"},
			scm.DeclarationParameter{"map", "func", "lambda function to extract data from the dataset. You can use any column of that table as lambda parameter. You can return a value you want to extract and pass to reduce, but you can also directly call insert, print or resultrow functions. If you declare a parameter named '$update', this variable will hold a function that you can use to delete or update a row. Call ($update) to delete the dataset, call ($update '(\"field1\" value1 \"field2\" value2)) to update certain columns."},
			scm.DeclarationParameter{"reduce", "func", "(optional) lambda function to aggregate the map results. It takes two parameters (a b) where a is the accumulator and b the new value. The accumulator for the first reduce call is the neutral element. The return value will be the accumulator input for the next reduce call. There are two reduce phases: shard-local and shard-collect. In the shard-local phase, a starts with neutral and b is fed with the return values of each map call. In the shard-collect phase, a starts with neutral and b is fed with the result of each shard-local pass."},
			scm.DeclarationParameter{"neutral", "any", "(optional) neutral element for the reduce phase, otherwise nil is assumed"},
		}, "any",
		func (a ...scm.Scmer) scm.Scmer {
			start := time.Now() // time measurement

			// params: table, condition, map, reduce, reduceSeed
			t := databases[scm.String(a[0])].Tables[scm.String(a[1])]
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
		},
	})
	// TODO: scan_order -> schema table filter sortcols(list of lambda|string) offset limit map reduce neutral; has only one reduce phase
	scm.Declare(&en, &scm.Declaration{
		"createdatabase", "creates a new database",
		1, 1,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the new database"},
		}, "bool",
		func (a ...scm.Scmer) scm.Scmer {
			CreateDatabase(scm.String(a[0]))
			return true
		},
	})
	scm.Declare(&en, &scm.Declaration{
		"dropdatabase", "creates a new database",
		1, 1,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the new database"},
		}, "bool",
		func (a ...scm.Scmer) scm.Scmer {
			DropDatabase(scm.String(a[0]))
			return true
		},
	})
	scm.Declare(&en, &scm.Declaration{
		"createtable", "creates a new database",
		3, 3,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the new table"},
			scm.DeclarationParameter{"cols", "list", "list of columns, each '(colname typename dimensions) where dimensions is a list of 0-2 numeric items"},
		}, "bool",
		func (a ...scm.Scmer) scm.Scmer {
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
			return true
		},
	})
	scm.Declare(&en, &scm.Declaration{
		"createtable", "creates a new database",
		2, 2,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the table"},
		}, "bool",
		func (a ...scm.Scmer) scm.Scmer {
			DropTable(scm.String(a[0]), scm.String(a[1]))
			return true
		},
	})
	scm.Declare(&en, &scm.Declaration{
		"insert", "inserts a new dataset into table",
		3, 3,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the table"},
			scm.DeclarationParameter{"row", "list", "list of the pattern '(\"col1\" value1 \"col2\" value2)"},
		}, "bool",
		func (a ...scm.Scmer) scm.Scmer {
			databases[scm.String(a[0])].Tables[scm.String(a[1])].Insert(dataset(a[2].([]scm.Scmer)))
			return true
		},
	})
	scm.Declare(&en, &scm.Declaration{
		"stat", "return memory statistics",
		0, 0,
		[]scm.DeclarationParameter{
		}, "string",
		func (a ...scm.Scmer) scm.Scmer {
			return PrintMemUsage()
		},
	})
	scm.Declare(&en, &scm.Declaration{
		"show", "show databases/tables/columns\n\n(show) will list all databases as a list of strings\n(show schema) will list all tables as a list of strings\n(show schema tbl) will list all columns as a list of dictionaries with the keys (name type dimensions)",
		0, 2,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "(optional) name of the database if you want to list tables or columns"},
			scm.DeclarationParameter{"table", "string", "(optional) name of the table if you want to list columns"},
		}, "any",
		func (a ...scm.Scmer) scm.Scmer {
			if len(a) == 0 {
				// show databases
				result := make([]scm.Scmer, len(databases))
				i := 0
				for k, _ := range databases {
					result[i] = k
					i = i + 1
				}
				return result
			} else if len(a) == 1 {
				// show tables
				return databases[scm.String(a[0])].ShowTables()
			} else if len(a) == 2 {
				// show columns
				return databases[scm.String(a[0])].Tables[scm.String(a[1])].ShowColumns()
			} else {
				panic("invalid call of show")
			}
		},
	})
	scm.Declare(&en, &scm.Declaration{
		"rebuild", "rebuilds all main storages and returns the amount of time it took",
		0, 0,
		[]scm.DeclarationParameter{
		}, "string",
		func (a ...scm.Scmer) scm.Scmer {
			start := time.Now()

			for _, db := range databases {
				db.rebuild()
				db.save()
			}

			return fmt.Sprint(time.Since(start))
		},
	})
	scm.Declare(&en, &scm.Declaration{
		"loadCSV", "loads a CSV file into a table and returns the amount of time it took.\nThe first line of the file must be the headlines. The headlines must match the table's columns exactly.",
		3, 4,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the table"},
			scm.DeclarationParameter{"filename", "string", "filename of the CSV file (global path or relative to working directory of memcp)"},
			scm.DeclarationParameter{"delimiter", "string", "(optional) delimiter defaults to \";\""},
		}, "string",
		func (a ...scm.Scmer) scm.Scmer {
			// schema, table, filename, delimiter
			start := time.Now()

			delimiter := ";"
			if len(a) > 3 {
				delimiter = scm.String(a[3])
			}
			LoadCSV(scm.String(a[0]), scm.String(a[1]), scm.String(a[2]), delimiter)

			return fmt.Sprint(time.Since(start))
		},
	})
	scm.Declare(&en, &scm.Declaration{
		"loadJSON", "loads a .jsonl file from disk into a database and returns the amount of time it took.\nJSONL is a linebreak separated file of JSON objects. Each JSON object is one dataset in the database. Before you add rows, you must declare the table in a line '#table <tablename>'. All other lines starting with # are comments. Columns are created dynamically as soon as they occur in a json object.",
		2, 2,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database where you want to put the tables in"},
			scm.DeclarationParameter{"filename", "string", "filename of the .jsonl file (global path or relative to working directory of memcp)"},
		}, "string",
		func (a ...scm.Scmer) scm.Scmer {
			// schema, filename
			start := time.Now()

			LoadJSON(scm.String(a[0]), scm.String(a[1]))

			return fmt.Sprint(time.Since(start))
		},
	})
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
