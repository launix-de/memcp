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

import "io"
import "fmt"
import "time"
import "reflect"
import "runtime"
import "strings"
import "strconv"
import "github.com/launix-de/memcp/scm"
import units "github.com/docker/go-units"

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
	Serialize(io.Writer) // write content to Writer
	Deserialize(io.Reader) uint // read from Reader (note that first byte is already read, so the reader starts at the second byte)
}

var storages = map[uint8]reflect.Type {
	 1: reflect.TypeOf(StorageSCMER{}),
	 2: reflect.TypeOf(StorageSparse{}),
	10: reflect.TypeOf(StorageInt{}),
	11: reflect.TypeOf(StorageSeq{}),
	12: reflect.TypeOf(StorageFloat{}),
	20: reflect.TypeOf(StorageString{}),
	21: reflect.TypeOf(StoragePrefix{}),
	//30: reflect.TypeOf(OverlaySCMER{}),
	31: reflect.TypeOf(OverlayBlob{}),
}

func Init(en scm.Env) {
	scm.DeclareTitle("Storage")

	scm.Declare(&en, &scm.Declaration{
		"scan", "does an unordered parallel filter-map-reduce pass on a single table and returns the reduced result",
		6, 9,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string|nil", "database where the table is located"},
			scm.DeclarationParameter{"table", "string|list", "name of the table to scan (or a list if you have temporary data)"},
			scm.DeclarationParameter{"filterColumns", "list", "list of columns that are fed into filter"},
			scm.DeclarationParameter{"filter", "func", "lambda function that decides whether a dataset is passed to the map phase. You can use any column of that table as lambda parameter. You should structure your lambda with an (and) at the root element. Every equal? < > <= >= will possibly translated to an indexed scan"},
			scm.DeclarationParameter{"mapColumns", "list", "list of columns that are fed into map"},
			scm.DeclarationParameter{"map", "func", "lambda function to extract data from the dataset. You can use any column of that table as lambda parameter. You can return a value you want to extract and pass to reduce, but you can also directly call insert, print or resultrow functions. If you declare a parameter named '$update', this variable will hold a function that you can use to delete or update a row. Call ($update) to delete the dataset, call ($update '(\"field1\" value1 \"field2\" value2)) to update certain columns."},
			scm.DeclarationParameter{"reduce", "func", "(optional) lambda function to aggregate the map results. It takes two parameters (a b) where a is the accumulator and b the new value. The accumulator for the first reduce call is the neutral element. The return value will be the accumulator input for the next reduce call. There are two reduce phases: shard-local and shard-collect. In the shard-local phase, a starts with neutral and b is fed with the return values of each map call. In the shard-collect phase, a starts with neutral and b is fed with the result of each shard-local pass."},
			scm.DeclarationParameter{"neutral", "any", "(optional) neutral element for the reduce phase, otherwise nil is assumed"},
			scm.DeclarationParameter{"neutral2", "func", "(optional) second stage reduce function that will apply a result of reduce to the neutral element/accumulator"},
		}, "any",
		func (a ...scm.Scmer) scm.Scmer {
			filtercols_ := a[2].([]scm.Scmer)
			filtercols := make([]string, len(filtercols_))
			for i, c := range filtercols_ {
				filtercols[i] = scm.String(c)
			}
			mapcols_ := a[4].([]scm.Scmer)
			mapcols := make([]string, len(mapcols_))
			for i, c := range mapcols_ {
				mapcols[i] = scm.String(c)
			}
			if list, ok := a[1].([]scm.Scmer); ok {
				// implementation on lists
				var result scm.Scmer = nil
				if len(a) > 7 { // custom neutral element
					result = a[7]
				}
				filterfn := scm.OptimizeProcToSerialFunction(a[3])
				filterparams := make([]scm.Scmer, len(filtercols))
				mapfn := scm.OptimizeProcToSerialFunction(a[5])
				mapparams := make([]scm.Scmer, len(mapcols))
				reducefn := func(a ...scm.Scmer) scm.Scmer {
					return a[1]
				}
				if len(a) > 6 {
					reducefn = scm.OptimizeProcToSerialFunction(a[6])
				}
				for _, val := range list {
					ds := dataset(val.([]scm.Scmer))
					// filter
					for i, col := range filtercols {
						filterparams[i] = ds.Get(col)
					}
					if scm.ToBool(filterfn(filterparams...)) {
						// map
						for i, col := range mapcols {
							mapparams[i] = ds.Get(col)
						}
						// reduce
						result = reducefn(result, mapfn(mapparams...))
					}
				}
				if len(a) > 8 {
					reduce2 := scm.OptimizeProcToSerialFunction(a[8])
					result = reduce2(a[7], result)
				}
				return result
			}
			// otherwise: implementation on storage
			// params: table, condition, map, reduce, reduceSeed
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				panic("database " + scm.String(a[0]) + " does not exist")
			}
			t := db.Tables.Get(scm.String(a[1]))
			if t == nil {
				panic("table " + scm.String(a[0]) + "." + scm.String(a[1]) + " does not exist")
			}
			var aggregate scm.Scmer
			var neutral scm.Scmer
			if len(a) > 6 {
				aggregate = a[6]
			}
			if len(a) > 7 {
				neutral = a[7]
			}
			reduce2 := scm.Scmer(nil)
			if len(a) > 8 {
				reduce2 = a[8]
			}
			result := t.scan(filtercols, a[3], mapcols, a[5], aggregate, neutral, reduce2)
			return result
		},
	})
	scm.Declare(&en, &scm.Declaration{
		"scan_order", "does an ordered parallel filter and serial map-reduce pass on a single table and returns the reduced result",
		10, 12,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "database where the table is located"},
			scm.DeclarationParameter{"table", "string", "name of the table to scan"},
			scm.DeclarationParameter{"filterColumns", "list", "list of columns that are fed into filter"},
			scm.DeclarationParameter{"filter", "func", "lambda function that decides whether a dataset is passed to the map phase. You can use any column of that table as lambda parameter. You should structure your lambda with an (and) at the root element. Every equal? < > <= >= will possibly translated to an indexed scan"},
			scm.DeclarationParameter{"sortcols", "list", "list of columns to sort. Each column is either a string to point to an existing column or a func(cols...)->any to compute a sortable value"},
			scm.DeclarationParameter{"sortdirs", "list", "list of column directions to sort. Must be same length as sortcols. false means ASC, true means DESC"},
			scm.DeclarationParameter{"offset", "number", "number of items to skip before the first one is fed into map"},
			scm.DeclarationParameter{"limit", "number", "max number of items to read"},
			scm.DeclarationParameter{"mapColumns", "list", "list of columns that are fed into map"},
			scm.DeclarationParameter{"map", "func", "lambda function to extract data from the dataset. You can use any column of that table as lambda parameter. You can return a value you want to extract and pass to reduce, but you can also directly call insert, print or resultrow functions. If you declare a parameter named '$update', this variable will hold a function that you can use to delete or update a row. Call ($update) to delete the dataset, call ($update '(\"field1\" value1 \"field2\" value2)) to update certain columns."},
			scm.DeclarationParameter{"reduce", "func", "(optional) lambda function to aggregate the map results. It takes two parameters (a b) where a is the accumulator and b the new value. The accumulator for the first reduce call is the neutral element. The return value will be the accumulator input for the next reduce call. There are two reduce phases: shard-local and shard-collect. In the shard-local phase, a starts with neutral and b is fed with the return values of each map call. In the shard-collect phase, a starts with neutral and b is fed with the result of each shard-local pass."},
			scm.DeclarationParameter{"neutral", "any", "(optional) neutral element for the reduce phase, otherwise nil is assumed"},
		}, "any",
		func (a ...scm.Scmer) scm.Scmer {
			filtercols_ := a[2].([]scm.Scmer)
			filtercols := make([]string, len(filtercols_))
			for i, c := range filtercols_ {
				filtercols[i] = scm.String(c)
			}
			mapcols_ := a[8].([]scm.Scmer)
			mapcols := make([]string, len(mapcols_))
			for i, c := range mapcols_ {
				mapcols[i] = scm.String(c)
			}

			if list, ok := a[1].([]scm.Scmer); ok {
				// implementation on lists
				// TODO: version on primitive lists like in scan
				panic("scan_order is not implemented on lists yet")
				for range list {}
				return nil
			}
			// params: table, condition, map, reduce, reduceSeed
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				panic("database " + scm.String(a[0]) + " does not exist")
			}
			t := db.Tables.Get(scm.String(a[1]))
			if t == nil {
				panic("table " + scm.String(a[0]) + "." + scm.String(a[1]) + " does not exist")
			}
			var aggregate scm.Scmer
			var neutral scm.Scmer
			if len(a) > 10 {
				aggregate = a[10]
			}
			if len(a) > 11 {
				neutral = a[11]
			}
			sortcols := a[4].([]scm.Scmer)
			sortdirs := make([]bool, len(sortcols))
			for i, dir := range a[5].([]scm.Scmer) {
				sortdirs[i] = scm.ToBool(dir)
			}
			result := t.scan_order(filtercols, a[3], sortcols, sortdirs, scm.ToInt(a[6]), scm.ToInt(a[7]), mapcols, a[9], aggregate, neutral)
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
		4, 5,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the new table"},
			scm.DeclarationParameter{"cols", "list", "list of columns and constraints, each '(\"column\" colname typename dimensions typeparams) where dimensions is a list of 0-2 numeric items or '(\"primary\" cols) or '(\"unique\" cols) or '(\"foreign\" cols tbl2 cols2)"},
			scm.DeclarationParameter{"options", "list", "further options like engine=safe|sloppy|memory"},
			scm.DeclarationParameter{"ifnotexists", "bool", "don't throw an error if table already exists"},
		}, "bool",
		func (a ...scm.Scmer) scm.Scmer {
			// parse options
			var pm PersistencyMode = Safe
			options := a[3].([]scm.Scmer)
			var auto_increment uint64 = 0
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
				} else if options[i] == "collation" {
					// TODO: store the collation??
				} else if options[i] == "auto_increment" {
					auto_increment, _ = strconv.ParseUint(scm.String(options[i+1]), 0, 64)
				} else {
					panic("unknown option: " + scm.String(options[i]))
				}
			}

			// create table
			ifnotexists := false
			if len(a) > 4 && scm.ToBool(a[4]) {
				ifnotexists = true
			}
			t, created := CreateTable(scm.String(a[0]), scm.String(a[1]), pm, ifnotexists)
			t.Auto_increment = auto_increment
			if created {
				// add columns and constraints
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
						t2name := scm.String(def[3])
						t2 := t.schema.Tables.Get(t2name)
						t.Foreign = append(t.Foreign, foreignKey{scm.String(def[1]), t.Name, cols1, t2name, cols2})
						if t2 != nil {
							// non-forward declaration
							t2.Foreign = append(t2.Foreign, foreignKey{scm.String(def[1]), t.Name, cols1, t2name, cols2})
						}
						fmt.Println("!----! created foreign key")
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
				// add constraints that are added onto us
				for _, t2 := range t.schema.Tables.GetAll() {
					if t2 != t {
						for _, foreign := range t2.Foreign {
							if foreign.Tbl2 == t.Name {
								// copy foward declaration to our definition list
								t.Foreign = append(t.Foreign, foreign)
							}
						}
					}
				}
			}
			return true
		},
	})
	scm.Declare(&en, &scm.Declaration{
		"createcolumn", "creates a new column in table",
		6, 7,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the new table"},
			scm.DeclarationParameter{"colname", "string", "name of the new column"},
			scm.DeclarationParameter{"type", "string", "name of the basetype"},
			scm.DeclarationParameter{"dimensions", "list", "dimensions of the type (e.g. for decimal)"},
			scm.DeclarationParameter{"options", "string", "further options like AUTO_INCREMENT or NOT NULL"},
			scm.DeclarationParameter{"computor", "func", "lambda expression that can take other column values and computes the value of that column"},
		}, "bool",
		func (a ...scm.Scmer) scm.Scmer {
			// get tbl
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				panic("database " + scm.String(a[0]) + " does not exist")
			}
			t := db.Tables.Get(scm.String(a[1]))
			if t == nil {
				panic("table " + scm.String(a[0]) + "." + scm.String(a[1]) + " does not exist")
			}

			// normal column
			colname := scm.String(a[2])
			typename := scm.String(a[3])
			dimensions_ := a[4].([]scm.Scmer)
			dimensions := make([]int, len(dimensions_))
			for i, d := range dimensions_ {
				dimensions[i] = scm.ToInt(d)
			}
			typeparams := scm.String(a[5])
			// TODO: check if column exists (and then skip to compute)
			ok := t.CreateColumn(colname, typename, dimensions, typeparams)
			// todo: not null flags, PRIMARY KEY flag usw.
			if ok && strings.Contains(strings.ToLower(scm.String(typeparams)), "primary") { // the condition is hacky
				// append unique key
				t.Unique = append(t.Unique, uniqueKey{"PRIMARY", []string{colname}})
			}

			if len(a) > 6 && a[6] != nil {
				// computed columns (interface might not be final)
				t.ComputeColumn(colname, a[6])
			}
			
			return ok
		},
	})
	scm.Declare(&en, &scm.Declaration{
		"shardcolumn", "tells us how it would partition a column according to their values. Returns a list of pivot elements.",
		4, 4,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the new table"},
			scm.DeclarationParameter{"colname", "string", "name of the column"},
			scm.DeclarationParameter{"numpartitions", "number", "number of partitions"},
		}, "list",
		func (a ...scm.Scmer) scm.Scmer {
			// get tbl
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				panic("database " + scm.String(a[0]) + " does not exist")
			}
			t := db.Tables.Get(scm.String(a[1]))
			if t == nil {
				panic("table " + scm.String(a[0]) + "." + scm.String(a[1]) + " does not exist")
			}
			return t.NewShardDimension(scm.String(a[2]), scm.ToInt(a[3])).Pivots
		},
	})
	scm.Declare(&en, &scm.Declaration{
		"altertable", "alters a table",
		4, 4,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the new table"},
			scm.DeclarationParameter{"operation", "string", "one of drop|engine|collation|auto_increment"},
			scm.DeclarationParameter{"parameter", "any", "name of the column to drop or value of the parameter"},
		}, "bool",
		func (a ...scm.Scmer) scm.Scmer {
			// get tbl
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				panic("database " + scm.String(a[0]) + " does not exist")
			}
			t := db.Tables.Get(scm.String(a[1]))
			if t == nil {
				panic("table " + scm.String(a[0]) + "." + scm.String(a[1]) + " does not exist")
			}

			switch a[2] {
			case "drop":
				return t.DropColumn(scm.String(a[3]))
			default:
				panic("unimplemented alter table operation: " + scm.String(a[2]))
			}
			return true
		},
	})
	scm.Declare(&en, &scm.Declaration{
		"droptable", "removes a table",
		2, 3,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the table"},
			scm.DeclarationParameter{"ifexists", "bool", "if true, don't throw an error if it already exists"},
		}, "bool",
		func (a ...scm.Scmer) scm.Scmer {
			if len(a) > 2 {
				DropTable(scm.String(a[0]), scm.String(a[1]), scm.ToBool(a[2]))
			} else {
				DropTable(scm.String(a[0]), scm.String(a[1]), false)
			}
			return true
		},
	})
	scm.Declare(&en, &scm.Declaration{
		"insert", "inserts a new dataset into table",
		4, 6,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the table"},
			scm.DeclarationParameter{"columns", "list", "list of column names, e.g. '(\"ID\", \"value\")"},
			scm.DeclarationParameter{"datasets", "list", "list of list of column values, e.g. '('(1 10) '(2 15))"},
			scm.DeclarationParameter{"ignoreexists", "bool", "if true, it will return false on duplicate keys instead of throwing an error"},
			scm.DeclarationParameter{"mergeNull", "bool", "if true, it will handle NULL values as equal according to SQL 2003's definition of DISTINCT (https://en.wikipedia.org/wiki/Null_(SQL)#When_two_nulls_are_equal:_grouping,_sorting,_and_some_set_operations)"},
		}, "bool",
		func (a ...scm.Scmer) scm.Scmer {
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				panic("database " + scm.String(a[0]) + " does not exist")
			}
			ignoreexists := false
			if (len(a) > 4 && scm.ToBool(a[4])) {
				ignoreexists = true
			}
			mergeNull := false
			if (len(a) > 5 && scm.ToBool(a[5])) {
				mergeNull = true
			}
			cols_ := a[2].([]scm.Scmer)
			cols := make([]string, len(cols_))
			for i, col := range cols_ {
				cols[i] = scm.String(col)
			}
			rows_ := a[3].([]scm.Scmer)
			rows := make([][]scm.Scmer, len(rows_))
			for i, row := range rows_ {
				rows[i] = row.([]scm.Scmer)
			}
			return float64(db.Tables.Get(scm.String(a[1])).Insert(cols, rows, ignoreexists, mergeNull))
		},
	})
	scm.Declare(&en, &scm.Declaration{
		"stat", "return memory statistics",
		0, 2,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database (optional: all databases)"},
			scm.DeclarationParameter{"table", "string", "name of the table (if table is set, print the detailled storage stats)"},
		}, "string",
		func (a ...scm.Scmer) scm.Scmer {
			if len(a) == 0 {
				return PrintMemUsage()
			} else if len(a) == 1 {
				return GetDatabase(scm.String(a[0])).PrintMemUsage()
			} else if len(a) == 2 {
				return GetDatabase(scm.String(a[0])).Tables.Get(scm.String(a[1])).PrintMemUsage()
			} else {
				return nil
			}
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
		0, 2,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"all", "bool", "if true, rebuild all shards, even if nothing has changed (default: false)"},
			scm.DeclarationParameter{"repartition", "bool", "if true, also repartition (default: true)"},
		}, "string",
		func (a ...scm.Scmer) scm.Scmer {
			all := false
			if len(a) > 0 && scm.ToBool(a[0]) {
				all = true
			}
			repartition := true
			if len(a) > 1 {
				repartition = scm.ToBool(a[1])
			}

			return Rebuild(all, repartition)
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
        b.WriteString(fmt.Sprintf("Alloc = %v MiB\tTotalAlloc = %v MiB\tSys = %v MiB\tNumGC = %v", units.BytesSize(float64(m.Alloc)), units.BytesSize(float64(m.TotalAlloc)), units.BytesSize(float64(m.Sys)), m.NumGC))

	for _, db := range databases.GetAll() {
		b.WriteString("\n\n" + db.Name + "\n======\n")
		b.WriteString(db.PrintMemUsage())
	}
	return b.String()
}

func (db *database) PrintMemUsage() string {
        // For info on each, see: https://golang.org/pkg/runtime/#MemStats
	var b strings.Builder
	b.WriteString("Table                    \tColumns\tShards\tDims\tSize/Bytes\n")
	var dsize uint
	for _, t := range db.Tables.GetAll() {
		var size uint = 10*8 + 32 * uint(len(t.Columns))
		for _, s := range t.Shards {
			size += s.Size()
		}
		b.WriteString(fmt.Sprintf("%-25s\t%d\t%d\t%d\t%s\n", t.Name, len(t.Columns), len(t.Shards) + len(t.PShards), len(t.PDimensions), units.BytesSize(float64(size))));
		dsize += size
	}
	b.WriteString(fmt.Sprintf("\ntotal size = %s\n", units.BytesSize(float64(dsize))));
	return b.String()
}

func (t *table) PrintMemUsage() string {
	var b strings.Builder
	var dsize uint = 0
	shards := t.Shards
	if shards == nil {
		shards = t.PShards
		b.WriteString(fmt.Sprint("Partitioning Schema:", t.PDimensions) + "\n\n")
	}
	for i, s := range shards {
		var ssz uint = 0
		b.WriteString(fmt.Sprintf("Shard %d\n---\n", i))
		b.WriteString(fmt.Sprintf("main count: %d, delta count: %d, deletions: %d\n", s.main_count, len(s.inserts), s.deletions.Count()))
		for c, v := range s.columns {
			sz := v.Size()
			b.WriteString(fmt.Sprintf("%s: %s, size = %s\n", c, v.String(), units.BytesSize(float64(sz))));
			ssz += sz
		}
		b.WriteString(fmt.Sprintf("= total %s\n\n", units.BytesSize(float64(ssz))));
		dsize += ssz
	}
	b.WriteString(fmt.Sprintf("= total %s\n\n", units.BytesSize(float64(dsize))));
	return b.String()
}
