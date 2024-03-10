/*
Copyright (C) 2023, 2024  Carl-Philip Hänsch

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
	// info
	GetValue(uint) scm.Scmer // read function
	String() string // self-description
	Size() uint // stat

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
		"scan", "does an unordered parallel filter-map-reduce pass on a single table and returns the reduced result",
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
			// params: table, condition, map, reduce, reduceSeed
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				panic("database " + scm.String(a[0]) + " does not exist")
			}
			t := db.Tables.Get(scm.String(a[1]))
			var aggregate scm.Scmer
			var neutral scm.Scmer
			if len(a) > 4 {
				aggregate = a[4]
			}
			if len(a) > 5 {
				neutral = a[5]
			}
			result := t.scan(a[2], a[3], aggregate, neutral)
			return result
		},
	})
	// TODO: scan_order -> schema table filter sortcols(list of lambda|string) offset limit map reduce neutral; has only one reduce phase
	scm.Declare(&en, &scm.Declaration{
		"scan_order", "does an ordered parallel filter and serial map-reduce pass on a single table and returns the reduced result",
		8, 10,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "database where the table is located"},
			scm.DeclarationParameter{"table", "string", "name of the table to scan"},
			scm.DeclarationParameter{"filter", "func", "lambda function that decides whether a dataset is passed to the map phase. You can use any column of that table as lambda parameter. You should structure your lambda with an (and) at the root element. Every equal? < > <= >= will possibly translated to an indexed scan"},
			scm.DeclarationParameter{"sortcols", "list", "list of columns to sort. Each column is either a string to point to an existing column or a func(cols...)->any to compute a sortable value"},
			scm.DeclarationParameter{"sortdirs", "list", "list of column directions to sort. Must be same length as sortcols. false means ASC, true means DESC"},
			scm.DeclarationParameter{"offset", "number", "number of items to skip before the first one is fed into map"},
			scm.DeclarationParameter{"limit", "number", "max number of items to read"},
			scm.DeclarationParameter{"map", "func", "lambda function to extract data from the dataset. You can use any column of that table as lambda parameter. You can return a value you want to extract and pass to reduce, but you can also directly call insert, print or resultrow functions. If you declare a parameter named '$update', this variable will hold a function that you can use to delete or update a row. Call ($update) to delete the dataset, call ($update '(\"field1\" value1 \"field2\" value2)) to update certain columns."},
			scm.DeclarationParameter{"reduce", "func", "(optional) lambda function to aggregate the map results. It takes two parameters (a b) where a is the accumulator and b the new value. The accumulator for the first reduce call is the neutral element. The return value will be the accumulator input for the next reduce call. There are two reduce phases: shard-local and shard-collect. In the shard-local phase, a starts with neutral and b is fed with the return values of each map call. In the shard-collect phase, a starts with neutral and b is fed with the result of each shard-local pass."},
			scm.DeclarationParameter{"neutral", "any", "(optional) neutral element for the reduce phase, otherwise nil is assumed"},
		}, "any",
		func (a ...scm.Scmer) scm.Scmer {
			// params: table, condition, map, reduce, reduceSeed
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				panic("database " + scm.String(a[0]) + " does not exist")
			}
			t := db.Tables.Get(scm.String(a[1]))
			var aggregate scm.Scmer
			var neutral scm.Scmer
			if len(a) > 8 {
				aggregate = a[8]
			}
			if len(a) > 9 {
				neutral = a[9]
			}
			sortcols := a[3].([]scm.Scmer)
			sortdirs := make([]bool, len(sortcols))
			for i, dir := range a[4].([]scm.Scmer) {
				sortdirs[i] = scm.ToBool(dir)
			}
			result := t.scan_order(a[2], sortcols, sortdirs, scm.ToInt(a[5]), scm.ToInt(a[6]), a[7], aggregate, neutral)
			return result
		},
	})
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
		4, 4,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the new table"},
			scm.DeclarationParameter{"cols", "list", "list of columns and constraints, each '(\"column\" colname typename dimensions typeparams) where dimensions is a list of 0-2 numeric items or '(\"primary\" cols) or '(\"unique\" cols) or '(\"foreign\" cols tbl2 cols2)"},
			scm.DeclarationParameter{"options", "list", "further options like engine=safe|sloppy|memory"},
		}, "bool",
		func (a ...scm.Scmer) scm.Scmer {
			// parse options
			var pm PersistencyMode = Safe
			options := a[3].([]scm.Scmer)
			for i := 0; i < len(options); i += 2 {
				if options[i] == "engine" {
					if options[i+1] == "memory" {
						pm = Memory
					} else if options[i+1] == "sloppy" {
						pm = Sloppy
					} else if options[i+1] == "safe" {
						pm = Safe
					} else {
						panic("unknown engine: " + scm.String(options[i+1]))
					}
				} else {
					panic("unknown option: " + scm.String(options[i]))
				}
			}

			// create table
			t := CreateTable(scm.String(a[0]), scm.String(a[1]), pm)
			for _, coldef := range(a[2].([]scm.Scmer)) {
				def := coldef.([]scm.Scmer)
				if len(def) == 0 {
					continue
				}
				if def[0] == "unique" {
					// id cols
					cols := make([]string, len(def[2].([]scm.Scmer)))
					for i, v := range def[2].([]scm.Scmer) {
						cols[i] = scm.String(v)
					}
					t.Unique = append(t.Unique, uniqueKey{scm.String(def[1]), cols})
				} else
				if def[0] == "foreign" {
					// id cols tbl cols2
					cols1 := make([]string, len(def[2].([]scm.Scmer)))
					for i, v := range def[2].([]scm.Scmer) {
						cols1[i] = scm.String(v)
					}
					cols2 := make([]string, len(def[4].([]scm.Scmer)))
					for i, v := range def[4].([]scm.Scmer) {
						cols2[i] = scm.String(v)
					}
					t2 := t.schema.Tables.Get(scm.String(def[3]))
					if t2 == nil {
						panic("Table in foreign key does not exist: " + scm.String(def[3]))
					}
					t.Foreign = append(t.Foreign, foreignKey{scm.String(def[1]), t, cols1, t2, cols2})
					t2.Foreign = append(t2.Foreign, foreignKey{scm.String(def[1]), t, cols1, t2, cols2})
				} else
				if def[0] == "column" {
					// normal column
					colname := scm.String(def[1])
					typename := scm.String(def[2])
					dimensions_ := def[3].([]scm.Scmer)
					dimensions := make([]int, len(dimensions_))
					for i, d := range dimensions_ {
						dimensions[i] = scm.ToInt(d)
					}
					typeparams := scm.String(def[4])
					t.CreateColumn(colname, typename, dimensions, typeparams)
					// todo: not null flags, PRIMARY KEY flag usw.
					if strings.Contains(strings.ToLower(scm.String(typeparams)), "primary") { // the condition is hacky
						// append unique key
						t.Unique = append(t.Unique, uniqueKey{"PRIMARY", []string{colname}})
					}
				}
			}
			return true
		},
	})
	scm.Declare(&en, &scm.Declaration{
		"droptable", "removes a table",
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
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				panic("database " + scm.String(a[0]) + " does not exist")
			}
			db.Tables.Get(scm.String(a[1])).Insert(dataset(a[2].([]scm.Scmer)))
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
				dbs := databases.GetAll()
				result := make([]scm.Scmer, len(dbs))
				for i, db := range dbs {
					result[i] = db.Name
				}
				return result
			} else if len(a) == 1 {
				// show tables
				db := GetDatabase(scm.String(a[0]))
				if db == nil {
					return nil // use this to check if a database exists
				}
				return db.ShowTables()
			} else if len(a) == 2 {
				// show columns
				db := GetDatabase(scm.String(a[0]))
				if db == nil {
					panic("database " + scm.String(a[0]) + " does not exist")
				}
				return db.Tables.Get(scm.String(a[1])).ShowColumns()
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

			dbs := databases.GetAll()
			for _, db := range dbs {
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

	for _, db := range databases.GetAll() {
		var dsize uint
		b.WriteString("\n\n" + db.Name + "\n======\nTable                    \tColumns\tShards\tSize/Bytes")
		for _, t := range db.Tables.GetAll() {
			var size uint = 10*8 + 32 * uint(len(t.Columns))
			for _, s := range t.Shards {
				size += s.Size()
			}
			b.WriteString(fmt.Sprintf("\n%-25s\t%d\t%d\t%d", t.Name, len(t.Columns), len(t.Shards), size));
			dsize += size
		}
		b.WriteString(fmt.Sprintf("\n= %d bytes", dsize));
	}
	return b.String()
}

func bToMb(b uint64) uint64 {
    return b / 1024 / 1024
}
