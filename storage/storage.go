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
import "sync"
import "fmt"
import "time"
import "sort"
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
	String() string          // self-description
	scm.Sizable

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
	Serialize(io.Writer)        // write content to Writer
	Deserialize(io.Reader) uint // read from Reader (note that first byte is already read, so the reader starts at the second byte)
}

var storages = map[uint8]reflect.Type{
	1:  reflect.TypeOf(StorageSCMER{}),
	2:  reflect.TypeOf(StorageSparse{}),
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
		6, 10,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string|nil", "database where the table is located"},
			scm.DeclarationParameter{"table", "string|list", "name of the table to scan (or a list if you have temporary data)"},
			scm.DeclarationParameter{"filterColumns", "list", "list of columns that are fed into filter"},
			scm.DeclarationParameter{"filter", "func", "lambda function that decides whether a dataset is passed to the map phase. You can use any column of that table as lambda parameter. You should structure your lambda with an (and) at the root element. Every equal? < > <= >= will possibly translated to an indexed scan"},
			scm.DeclarationParameter{"mapColumns", "list", "list of columns that are fed into map"},
			scm.DeclarationParameter{"map", "func", "lambda function to extract data from the dataset. You can use any column of that table as lambda parameter. You can return a value you want to extract and pass to reduce, but you can also directly call insert, print or resultrow functions. If you declare a parameter named '$update', this variable will hold a function that you can use to delete or update a row. Call ($update) to delete the dataset, call ($update '(\"field1\" value1 \"field2\" value2)) to update certain columns."},
			scm.DeclarationParameter{"reduce", "func", "(optional) lambda function to aggregate the map results. It takes two parameters (a b) where a is the accumulator and b the new value. The accumulator for the first reduce call is the neutral element. The return value will be the accumulator input for the next reduce call. There are two reduce phases: shard-local and shard-collect. In the shard-local phase, a starts with neutral and b is fed with the return values of each map call. In the shard-collect phase, a starts with neutral and b is fed with the result of each shard-local pass."},
			scm.DeclarationParameter{"neutral", "any", "(optional) neutral element for the reduce phase, otherwise nil is assumed"},
			scm.DeclarationParameter{"reduce2", "func", "(optional) second stage reduce function that will apply a result of reduce to the neutral element/accumulator"},
			scm.DeclarationParameter{"isOuter", "bool", "(optional) if true, in case of no hits, call map once anyway with NULL values"},
		}, "any",
		func(a ...scm.Scmer) scm.Scmer {
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
			isOuter := false
			if len(a) > 9 && scm.ToBool(a[9]) {
				isOuter = true
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
				hadValue := false
				for _, val := range list {
					ds := dataset(val.([]scm.Scmer))
					// filter
					for i, col := range filtercols {
						filterparams[i], _ = ds.GetI(col)
					}
					if scm.ToBool(filterfn(filterparams...)) {
						hadValue = true
						// map
						for i, col := range mapcols {
							mapparams[i], _ = ds.GetI(col)
						}
						// reduce
						result = reducefn(result, mapfn(mapparams...))
					}
				}
				if !hadValue && isOuter {
					// outer join
					result = reducefn(result, mapfn(mapparams...)) // mapparams is filled with NULL
				}
				if len(a) > 8 && a[8] != nil {
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
			result := t.scan(filtercols, a[3], mapcols, a[5], aggregate, neutral, reduce2, isOuter)
			return result
		}, false,
	})
	scm.Declare(&en, &scm.Declaration{
		"scan_order", "does an ordered parallel filter and serial map-reduce pass on a single table and returns the reduced result",
		10, 13,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "database where the table is located"},
			scm.DeclarationParameter{"table", "string", "name of the table to scan"},
			scm.DeclarationParameter{"filterColumns", "list", "list of columns that are fed into filter"},
			scm.DeclarationParameter{"filter", "func", "lambda function that decides whether a dataset is passed to the map phase. You can use any column of that table as lambda parameter. You should structure your lambda with an (and) at the root element. Every equal? < > <= >= will possibly translated to an indexed scan"},
			scm.DeclarationParameter{"sortcols", "list", "list of columns to sort. Each column is either a string to point to an existing column or a func(cols...)->any to compute a sortable value"},
			scm.DeclarationParameter{"sortdirs", "list", "list of column directions to sort. Must be same length as sortcols. < means ascending, > means descending, (collate ...) will add collations"},
			scm.DeclarationParameter{"offset", "number", "number of items to skip before the first one is fed into map"},
			scm.DeclarationParameter{"limit", "number", "max number of items to read"},
			scm.DeclarationParameter{"mapColumns", "list", "list of columns that are fed into map"},
			scm.DeclarationParameter{"map", "func", "lambda function to extract data from the dataset. You can use any column of that table as lambda parameter. You can return a value you want to extract and pass to reduce, but you can also directly call insert, print or resultrow functions. If you declare a parameter named '$update', this variable will hold a function that you can use to delete or update a row. Call ($update) to delete the dataset, call ($update '(\"field1\" value1 \"field2\" value2)) to update certain columns."},
			scm.DeclarationParameter{"reduce", "func", "(optional) lambda function to aggregate the map results. It takes two parameters (a b) where a is the accumulator and b the new value. The accumulator for the first reduce call is the neutral element. The return value will be the accumulator input for the next reduce call. There are two reduce phases: shard-local and shard-collect. In the shard-local phase, a starts with neutral and b is fed with the return values of each map call. In the shard-collect phase, a starts with neutral and b is fed with the result of each shard-local pass."},
			scm.DeclarationParameter{"neutral", "any", "(optional) neutral element for the reduce phase, otherwise nil is assumed"},
			scm.DeclarationParameter{"isOuter", "bool", "(optional) if true, in case of no hits, call map once anyway with NULL values"},
		}, "any",
		func(a ...scm.Scmer) scm.Scmer {
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

			var aggregate scm.Scmer
			var neutral scm.Scmer
			if len(a) > 10 {
				aggregate = a[10]
			}
			if len(a) > 11 {
				neutral = a[11]
			}
			sortcols := a[4].([]scm.Scmer)
			sortdirs := make([]func(...scm.Scmer) scm.Scmer, len(sortcols))
			for i, dir := range a[5].([]scm.Scmer) {
				sortdirs[i] = dir.(func(...scm.Scmer) scm.Scmer)
			}

			isOuter := false
			if len(a) > 12 && scm.ToBool(a[12]) {
				isOuter = true
			}

			if list, ok := a[1].([]scm.Scmer); ok {
				// implementation on lists
				var result scm.Scmer = neutral
				filterfn := scm.OptimizeProcToSerialFunction(a[3])
				filterparams := make([]scm.Scmer, len(filtercols))
				mapfn := scm.OptimizeProcToSerialFunction(a[9])
				mapparams := make([]scm.Scmer, len(mapcols))
				reducefn := func(a ...scm.Scmer) scm.Scmer {
					return a[1]
				}
				if aggregate != nil {
					reducefn = scm.OptimizeProcToSerialFunction(aggregate)
				}
				var list2 []scm.Scmer // this is the list we will sort
				for _, val := range list {
					ds := dataset(val.([]scm.Scmer))
					// filter
					for i, col := range filtercols {
						filterparams[i], _ = ds.GetI(col)
					}
					if scm.ToBool(filterfn(filterparams...)) {
						list2 = append(list2, val)
					}
				}
				// prepare sorting stage
				scols := make([]func(uint) scm.Scmer, len(sortcols))
				for i, scol := range sortcols {
					if colname, ok := scol.(string); ok {
						// naive column sort
						scols[i] = func(i uint) scm.Scmer {
							v, _ := dataset(list2[i].([]scm.Scmer)).GetI(colname)
							return v
						}
					} else if proc, ok := scol.(scm.Proc); ok {
						// complex lambda columns (TODO: either remove lambda columns or add colname mapping)
						largs := make([]func(uint) scm.Scmer, len(proc.Params.([]scm.Scmer))) // allocate only once, reuse in loop
						for j, param := range proc.Params.([]scm.Scmer) {
							largs[j] = func(i uint) scm.Scmer {
								v, _ := dataset(list2[i].([]scm.Scmer)).GetI(string(param.(scm.Symbol)))
								return v
							}
						}
						procFn := scm.OptimizeProcToSerialFunction(proc)
						scols[i] = func(idx uint) scm.Scmer {
							largs_ := make([]scm.Scmer, len(largs))
							for i, getter := range largs {
								// fetch columns used for getter
								largs_[i] = getter(idx)
							}
							// execute getter
							return procFn(largs_...)
						}
					} else {
						panic("unknown sort criteria: " + fmt.Sprint(scol))
					}
				}
				sort.Slice(list2, func(i, j int) bool {
					// sort list2
					for c := 0; c < len(scols); c++ {
						a := scols[c](uint(i))
						b := scols[c](uint(j))
						if scm.ToBool(sortdirs[c](a, b)) {
							return false
						} else if scm.ToBool(sortdirs[c](b, a)) {
							return true
						} // else: go to next level
						// otherwise: move on to c++
					}
					return false // equal is not less
				})
				hadValue := false
				for _, val := range list2 {
					ds := dataset(val.([]scm.Scmer))
					hadValue = true
					// map
					for i, col := range mapcols {
						mapparams[i], _ = ds.GetI(col)
					}
					// reduce
					result = reducefn(result, mapfn(mapparams...))
				}
				if !hadValue && isOuter {
					// outer join
					result = reducefn(result, mapfn(mapparams...)) // mapparams is filled with NULL
				}
				return result
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
			result := t.scan_order(filtercols, a[3], sortcols, sortdirs, scm.ToInt(a[6]), scm.ToInt(a[7]), mapcols, a[9], aggregate, neutral, isOuter)
			return result
		}, false,
	})
	scm.Declare(&en, &scm.Declaration{
		"createdatabase", "creates a new database",
		1, 2,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the new database"},
			scm.DeclarationParameter{"ignoreexists", "bool", "if true, return false instead of throwing an error"},
		}, "bool",
		func(a ...scm.Scmer) scm.Scmer {
			ignoreexists := false
			if len(a) > 1 && scm.ToBool(a[1]) {
				ignoreexists = true
			}
			return CreateDatabase(scm.String(a[0]), ignoreexists)
		}, false,
	})
	scm.Declare(&en, &scm.Declaration{
		"dropdatabase", "drops a database",
		1, 2,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"ifexists", "bool", "if true, don't throw an error if it doesn't exist"},
		}, "bool",
		func(a ...scm.Scmer) scm.Scmer {
			ifexists := false
			if len(a) > 1 {
				ifexists = scm.ToBool(a[1])
			}
			return DropDatabase(scm.String(a[0]), ifexists)
		}, false,
	})
	scm.Declare(&en, &scm.Declaration{
		"createtable", "creates a new database",
		4, 5,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the new table"},
			scm.DeclarationParameter{"cols", "list", "list of columns and constraints, each '(\"column\" colname typename dimensions typeparams) where dimensions is a list of 0-2 numeric items or '(\"primary\" cols) or '(\"unique\" cols) or '(\"foreign\" cols tbl2 cols2 updatemode deletemode of 'restrict'|'cascade'|'set null')"},
			scm.DeclarationParameter{"options", "list", "further options like engine=safe|sloppy|memory"},
			scm.DeclarationParameter{"ifnotexists", "bool", "don't throw an error if table already exists"},
		}, "bool",
		func(a ...scm.Scmer) scm.Scmer {
			// parse options
			var pm PersistencyMode = Safe
			options := a[3].([]scm.Scmer)
			var auto_increment uint64 = 0
			engine := Settings.DefaultEngine
			collation := ""
			charset := ""
			comment := ""
			for i := 0; i < len(options); i += 2 {
				if options[i] == "engine" {
					engine = scm.String(options[i+1])
				} else if options[i] == "collation" {
					collation = scm.String(options[i+1])
				} else if options[i] == "charset" {
					charset = scm.String(options[i+1])
				} else if options[i] == "comment" {
					comment = scm.String(options[i+1])
				} else if options[i] == "auto_increment" {
					auto_increment, _ = strconv.ParseUint(scm.String(options[i+1]), 0, 64)
				} else {
					panic("unknown option: " + scm.String(options[i]))
				}
			}
			if engine == "memory" {
				pm = Memory
			} else if engine == "sloppy" {
				pm = Sloppy
			} else if engine == "logged" {
				pm = Logged
			} else if engine == "safe" {
				pm = Safe
			} else {
				panic("unknown engine: " + engine)
			}

			// create table
			ifnotexists := false
			if len(a) > 4 && scm.ToBool(a[4]) {
				ifnotexists = true
			}
			t, created := CreateTable(scm.String(a[0]), scm.String(a[1]), pm, ifnotexists)
			t.Collation = collation
			t.Charset = charset
			t.Comment = comment
			t.Auto_increment = auto_increment
			if created {
				// add columns and constraints
				for _, coldef := range a[2].([]scm.Scmer) {
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
					} else if def[0] == "foreign" {
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
						var updatemode foreignKeyMode
						if len(def) > 5 {
							updatemode = getForeignKeyMode(def[5])
						}
						var deletemode foreignKeyMode
						if len(def) > 6 {
							deletemode = getForeignKeyMode(def[6])
						}
						t.Foreign = append(t.Foreign, foreignKey{scm.String(def[1]), t.Name, cols1, t2name, cols2, updatemode, deletemode})
						if t2 != nil {
							// non-forward declaration
							t2.Foreign = append(t2.Foreign, foreignKey{scm.String(def[1]), t.Name, cols1, t2name, cols2, updatemode, deletemode})
						}
					} else if def[0] == "column" {
						// normal column
						colname := scm.String(def[1])
						typename := scm.String(def[2])
						dimensions_ := def[3].([]scm.Scmer)
						dimensions := make([]int, len(dimensions_))
						for i, d := range dimensions_ {
							dimensions[i] = scm.ToInt(d)
						}
						typeparams := def[4].([]scm.Scmer)
						t.CreateColumn(colname, typename, dimensions, typeparams)
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
		}, false,
	})
	scm.Declare(&en, &scm.Declaration{
		"createcolumn", "creates a new column in table",
		6, 8,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the new table"},
			scm.DeclarationParameter{"colname", "string", "name of the new column"},
			scm.DeclarationParameter{"type", "string", "name of the basetype"},
			scm.DeclarationParameter{"dimensions", "list", "dimensions of the type (e.g. for decimal)"},
			scm.DeclarationParameter{"options", "list", "assoc list with one of the following options: primary true, unique true, auto_increment true, null bool, comment string default string collate identifier"},
			scm.DeclarationParameter{"computorCols", "list", "list of columns that is passed into params of computor"},
			scm.DeclarationParameter{"computor", "func", "lambda expression that can take other column values and computes the value of that column"},
		}, "bool",
		func(a ...scm.Scmer) scm.Scmer {
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
			typeparams := a[5].([]scm.Scmer)
			ok := t.CreateColumn(colname, typename, dimensions, typeparams)

			if len(a) > 7 && a[7] != nil {
				// computed columns (interface might not be final)
				param_names_ := a[6].([]scm.Scmer)
				param_names := make([]string, len(param_names_))
				for i, pn := range param_names_ {
					param_names[i] = scm.String(pn)
				}
				t.ComputeColumn(colname, param_names, a[7])
			}

			return ok
		}, false,
	})
	scm.Declare(&en, &scm.Declaration{
		"createkey", "creates a new key on a table",
		5, 5,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the new table"},
			scm.DeclarationParameter{"keyname", "string", "name of the new key"},
			scm.DeclarationParameter{"unique", "bool", "whether the key is unique"},
			scm.DeclarationParameter{"columns", "list", "list of columns to include"},
		}, "bool",
		func(a ...scm.Scmer) scm.Scmer {
			// get tbl
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				panic("database " + scm.String(a[0]) + " does not exist")
			}
			t := db.Tables.Get(scm.String(a[1]))
			if t == nil {
				panic("table " + scm.String(a[0]) + "." + scm.String(a[1]) + " does not exist")
			}

			if !scm.ToBool(a[3]) {
				// we don't care about non-unique keys since we build our own indices
				return true
			}

			l := a[4].([]scm.Scmer)
			name := scm.String(a[2])
			cols := make([]string, len(l))
			for i, v := range l {
				cols[i] = scm.String(v)
			}

			db.schemalock.Lock()
			defer db.schemalock.Unlock()
			for _, u := range t.Unique {
				if u.Id == name {
					return false // key already exists
				}
			}

			// add unique key
			t.Unique = append(t.Unique, uniqueKey{name, cols})
			db.save()

			return true
		}, false,
	})
	scm.Declare(&en, &scm.Declaration{
		"createforeignkey", "creates a new foreign key on a table",
		8, 8,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"keyname", "string", "name of the new key"},
			scm.DeclarationParameter{"table1", "string", "name of the first table"},
			scm.DeclarationParameter{"columns1", "list", "list of columns to include"},
			scm.DeclarationParameter{"table2", "string", "name of the second table"},
			scm.DeclarationParameter{"columns2", "list", "list of columns to include"},
			scm.DeclarationParameter{"updatemode", "string", "restrict|cascade|set null"},
			scm.DeclarationParameter{"deletemode", "string", "restrict|cascade|set null"},
		}, "bool",
		func(a ...scm.Scmer) scm.Scmer {
			// get tbl
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				panic("database " + scm.String(a[0]) + " does not exist")
			}
			id := scm.String(a[1])
			t1 := db.Tables.Get(scm.String(a[2]))
			if t1 == nil {
				panic("table " + scm.String(a[0]) + "." + scm.String(a[2]) + " does not exist")
			}
			t2 := db.Tables.Get(scm.String(a[4]))
			if t2 == nil {
				panic("table " + scm.String(a[0]) + "." + scm.String(a[4]) + " does not exist")
			}

			l1 := a[3].([]scm.Scmer)
			cols1 := make([]string, len(l1))
			for i, v := range l1 {
				cols1[i] = scm.String(v)
			}

			l2 := a[5].([]scm.Scmer)
			cols2 := make([]string, len(l2))
			for i, v := range l2 {
				cols2[i] = scm.String(v)
			}

			db.schemalock.Lock()
			defer db.schemalock.Unlock()
			for _, u := range t1.Foreign {
				if u.Id == id {
					return false // key already exists
				}
			}

			// add unique key
			k := foreignKey{id, t1.Name, cols1, t2.Name, cols2, getForeignKeyMode(a[6]), getForeignKeyMode(a[7])}
			t1.Foreign = append(t1.Foreign, k)
			t2.Foreign = append(t2.Foreign, k)
			db.save()

			return true
		}, false,
	})
	scm.Declare(&en, &scm.Declaration{
		"shardcolumn", "tells us how it would partition a column according to their values. Returns a list of pivot elements.",
		3, 4,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the new table"},
			scm.DeclarationParameter{"colname", "string", "name of the column"},
			scm.DeclarationParameter{"numpartitions", "number", "number of partitions; optional. leave 0 if you want to detect the partiton number automatically or copy the partition schema of the table"},
		}, "list",
		func(a ...scm.Scmer) scm.Scmer {
			// get tbl
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				panic("database " + scm.String(a[0]) + " does not exist")
			}
			t := db.Tables.Get(scm.String(a[1]))
			if t == nil {
				panic("table " + scm.String(a[0]) + "." + scm.String(a[1]) + " does not exist")
			}
			numPartitions := 0
			if len(a) > 3 {
				numPartitions = scm.ToInt(a[3])
			}
			if numPartitions == 0 {
				// check if that paritition dimension already exists
				if t.Shards == nil && t.PShards != nil {
					for _, sd := range t.PDimensions {
						if sd.Column == scm.String(a[2]) {
							return sd.Pivots // found the column in partition schema: return exactly the same pivots as we found already
						}
					}
				}
				// otherwise: no partition schema yet: find out the best number of partitions
				// normally, we put ~60,000 items per shard, but to parallelize grouping, we should do less?
				numPartitions = int(1 + ((2 * t.Count()) / Settings.ShardSize))
			}
			// calculate them anew
			return t.NewShardDimension(scm.String(a[2]), numPartitions).Pivots

		}, false,
	})
	scm.Declare(&en, &scm.Declaration{
		"partitiontable", "suggests a partition scheme for a table. If the table has no partition scheme yet, it will immediately apply that scheme and return true. If the table already has a partition scheme, it will alter the partitioning score such that the partitioning scheme is considered in the next repartitioning and return false.",
		3, 3,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the new table"},
			scm.DeclarationParameter{"columns", "list", "associative list of string -> list representing column name -> pivots. You can compute pivots by (shardcolumn ...)"},
		}, "bool",
		func(a ...scm.Scmer) scm.Scmer {
			// get tbl
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				panic("database " + scm.String(a[0]) + " does not exist")
			}
			t := db.Tables.Get(scm.String(a[1]))
			if t == nil {
				panic("table " + scm.String(a[0]) + "." + scm.String(a[1]) + " does not exist")
			}
			cols := dataset(a[2].([]scm.Scmer))
			if t.PDimensions == nil {
				// apply partitioning schema
				ps := make([]shardDimension, len(cols)/2)
				for i := 0; i < len(ps); i++ {
					ps[i].Column = scm.String(cols[2*i])
					ps[i].Pivots = cols[2*i+1].([]scm.Scmer)
					ps[i].NumPartitions = len(ps[i].Pivots) + 1
				}
				if len(ps) > Settings.PartitionMaxDimensions {
					ps = ps[:Settings.PartitionMaxDimensions]
				}
				t.repartition(ps) // perform repartitioning immediately
				return true
			} else {
				// increase partitioning scores
				for i, c := range t.Columns {
					if pivots, ok := cols.Get(c.Name); ok {
						// that column is in the parititoning schema -> increase score
						t.Columns[i].PartitioningScore = c.PartitioningScore + len(pivots.([]scm.Scmer))
					}
				}
				return false
			}
		}, false,
	})
	scm.Declare(&en, &scm.Declaration{
		"altertable", "alters a table",
		4, 4,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the table"},
			scm.DeclarationParameter{"operation", "string", "one of owner|drop|engine|collation"},
			scm.DeclarationParameter{"parameter", "any", "name of the column to drop or value of the parameter"},
		}, "bool",
		func(a ...scm.Scmer) scm.Scmer {
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
			case "owner":
				return false // ignore
			default:
				panic("unimplemented alter table operation: " + scm.String(a[2]))
			}
		}, false,
	})
	scm.Declare(&en, &scm.Declaration{
		"altercolumn", "alters a column",
		5, 5,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the table"},
			scm.DeclarationParameter{"column", "string", "name of the column"},
			scm.DeclarationParameter{"operation", "string", "one of drop|type|collation|auto_increment|comment"},
			scm.DeclarationParameter{"parameter", "any", "name of the column to drop or value of the parameter"},
		}, "bool",
		func(a ...scm.Scmer) scm.Scmer {
			// get tbl
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				panic("database " + scm.String(a[0]) + " does not exist")
			}
			t := db.Tables.Get(scm.String(a[1]))
			if t == nil {
				panic("table " + scm.String(a[0]) + "." + scm.String(a[1]) + " does not exist")
			}
			for i, c := range t.Columns {
				if c.Name == scm.String(a[2]) {
					switch a[3] {
					case "drop":
						ok := t.DropColumn(scm.String(a[2]))
						db.save()
						return ok
					case "auto_increment":
						ai := scm.ToInt(a[4])
						if ai > 1 {
							// set ai value
							t.Auto_increment = uint64(ai)
							db.save()
							return true
						} else {
							// set ai flag for column
							t.Columns[i].AutoIncrement = scm.ToBool(a[4])
							db.save()
							return true
						}
					default:
						ok := t.Columns[i].Alter(scm.String(a[3]), a[4])
						db.save()
						return ok
					}
				}
			}
			panic("column " + scm.String(a[0]) + "." + scm.String(a[1]) + "." + scm.String(a[2]) + " does not exist")
		}, false,
	})
	scm.Declare(&en, &scm.Declaration{
		"droptable", "removes a table",
		2, 3,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the table"},
			scm.DeclarationParameter{"ifexists", "bool", "if true, don't throw an error if it already exists"},
		}, "bool",
		func(a ...scm.Scmer) scm.Scmer {
			if len(a) > 2 {
				DropTable(scm.String(a[0]), scm.String(a[1]), scm.ToBool(a[2]))
			} else {
				DropTable(scm.String(a[0]), scm.String(a[1]), false)
			}
			return true
		}, false,
	})
scm.Declare(&en, &scm.Declaration{
		"insert", "inserts a new dataset into table and returns the number of successful items",
		4, 8,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the table"},
			scm.DeclarationParameter{"columns", "list", "list of column names, e.g. '(\"ID\", \"value\")"},
			scm.DeclarationParameter{"datasets", "list", "list of list of column values, e.g. '('(1 10) '(2 15))"},
			scm.DeclarationParameter{"onCollisionCols", "list", "list of columns of the old dataset that have to be passed to onCollision. Can also request $update."},
			scm.DeclarationParameter{"onCollision", "func", "the function that is called on each collision dataset. The first parameter is filled with the $update function, the second parameter is the dataset as associative list. If not set, an error is thrown in case of a collision."},
			scm.DeclarationParameter{"mergeNull", "bool", "if true, it will handle NULL values as equal according to SQL 2003's definition of DISTINCT (https://en.wikipedia.org/wiki/Null_(SQL)#When_two_nulls_are_equal:_grouping,_sorting,_and_some_set_operations)"},
			scm.DeclarationParameter{"onInsertid", "func", "(optional) callback (id)->any; called once with the first auto_increment id assigned for this INSERT"},
		}, "number",
		func(a ...scm.Scmer) scm.Scmer {
			db := GetDatabase(scm.String(a[0]))
			if db == nil {
				panic("database " + scm.String(a[0]) + " does not exist")
			}
			var onCollisionCols []string
			onCollision := scm.Scmer(nil)
			if len(a) > 5 {
				onCollisionCols_ := a[4].([]scm.Scmer)
				onCollisionCols = make([]string, len(onCollisionCols_))
				for i, c := range onCollisionCols_ {
					onCollisionCols[i] = scm.String(c)
				}
				onCollision = a[5]
			}
			mergeNull := false
			if len(a) > 6 && scm.ToBool(a[6]) {
				mergeNull = true
			}
			// optional onInsertid callback
			var onFirst func(int64)
			if len(a) > 7 && a[7] != nil {
				cb := a[7]
				var once sync.Once
				onFirst = func(id int64) { once.Do(func() { scm.Apply(cb, id) }) }
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
			// perform insert
			t := db.Tables.Get(scm.String(a[1]))
			inserted := t.Insert(cols, rows, onCollisionCols, onCollision, mergeNull, onFirst)
			return int64(inserted)
		}, false,
	})
	scm.Declare(&en, &scm.Declaration{
		"stat", "return memory statistics",
		0, 2,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database (optional: all databases)"},
			scm.DeclarationParameter{"table", "string", "name of the table (if table is set, print the detailled storage stats)"},
		}, "string",
		func(a ...scm.Scmer) scm.Scmer {
			if len(a) == 0 {
				return PrintMemUsage()
			} else if len(a) == 1 {
				return GetDatabase(scm.String(a[0])).PrintMemUsage()
			} else if len(a) == 2 {
				return GetDatabase(scm.String(a[0])).Tables.Get(scm.String(a[1])).PrintMemUsage()
			} else {
				return nil
			}
		}, false,
	})
	scm.Declare(&en, &scm.Declaration{
		"show", "show databases/tables/columns\n\n(show) will list all databases as a list of strings\n(show schema) will list all tables as a list of strings\n(show schema tbl) will list all columns as a list of dictionaries with the keys (name type dimensions)",
		0, 2,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "(optional) name of the database if you want to list tables or columns"},
			scm.DeclarationParameter{"table", "string", "(optional) name of the table if you want to list columns"},
		}, "any",
		func(a ...scm.Scmer) scm.Scmer {
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
		}, false,
	})
	scm.Declare(&en, &scm.Declaration{
		"rebuild", "rebuilds all main storages and returns the amount of time it took",
		0, 2,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"all", "bool", "if true, rebuild all shards, even if nothing has changed (default: false)"},
			scm.DeclarationParameter{"repartition", "bool", "if true, also repartition (default: true)"},
		}, "string",
		func(a ...scm.Scmer) scm.Scmer {
			all := false
			if len(a) > 0 && scm.ToBool(a[0]) {
				all = true
			}
			repartition := true
			if len(a) > 1 {
				repartition = scm.ToBool(a[1])
			}

			return Rebuild(all, repartition)
		}, false,
	})
	scm.Declare(&en, &scm.Declaration{
		"loadCSV", "loads a CSV stream into a table and returns the amount of time it took.\nThe first line of the file must be the headlines. The headlines must match the table's columns exactly.",
		3, 5,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database"},
			scm.DeclarationParameter{"table", "string", "name of the table"},
			scm.DeclarationParameter{"stream", "stream", "CSV file, load with: (stream filename)"},
			scm.DeclarationParameter{"delimiter", "string", "(optional) delimiter defaults to \";\""},
			scm.DeclarationParameter{"firstline", "bool", "(optional) if the first line contains the column names (otherwise, the tables column order is used)"},
		}, "string",
		func(a ...scm.Scmer) scm.Scmer {
			// schema, table, filename, delimiter
			start := time.Now()

			delimiter := ";"
			if len(a) > 3 {
				delimiter = scm.String(a[3])
			}
			firstline := true
			if len(a) > 4 {
				firstline = scm.ToBool(a[4])
			}
			LoadCSV(scm.String(a[0]), scm.String(a[1]), a[2].(io.Reader), delimiter, firstline)

			return fmt.Sprint(time.Since(start))
		}, false,
	})
	scm.Declare(&en, &scm.Declaration{
		"loadJSON", "loads a .jsonl file from stream into a database and returns the amount of time it took.\nJSONL is a linebreak separated file of JSON objects. Each JSON object is one dataset in the database. Before you add rows, you must declare the table in a line '#table <tablename>'. All other lines starting with # are comments. Columns are created dynamically as soon as they occur in a json object.",
		2, 2,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"schema", "string", "name of the database where you want to put the tables in"},
			scm.DeclarationParameter{"stream", "stream", "stream of the .jsonl file, read with: (stream filename)"},
		}, "string",
		func(a ...scm.Scmer) scm.Scmer {
			// schema, filename
			start := time.Now()

			LoadJSON(scm.String(a[0]), a[1].(io.Reader))

			return fmt.Sprint(time.Since(start))
		}, false,
	})
	scm.Declare(&en, &scm.Declaration{
		"settings", "reads or writes a global settings value. This modifies your data/settings.json.",
		0, 2,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"key", "string", "name of the key to set or get (for reference, rts)"},
			scm.DeclarationParameter{"value", "any", "new value of that setting"},
		}, "any",
		ChangeSettings, false,
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
		var size uint = 10*8 + 32*uint(len(t.Columns))
		for _, s := range t.Shards {
			size += s.ComputeSize()
		}
		for _, s := range t.PShards {
			size += s.ComputeSize()
		}
		b.WriteString(fmt.Sprintf("%-25s\t%d\t%d\t%d\t%s\n", t.Name, len(t.Columns), len(t.Shards)+len(t.PShards), len(t.PDimensions), units.BytesSize(float64(size))))
		dsize += size
	}
	b.WriteString(fmt.Sprintf("\ntotal size = %s\n", units.BytesSize(float64(dsize))))
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
		var ssz uint = 14 * 8 // overhead
		b.WriteString(fmt.Sprintf("Shard %d\n---\n", i))
		b.WriteString(fmt.Sprintf("main count: %d, delta count: %d, deletions: %d\n", s.main_count, len(s.inserts), s.deletions.Count()))
		for c, v := range s.columns {
			sz := v.ComputeSize()
			b.WriteString(fmt.Sprintf(" %s: %s, size = %s\n", c, v.String(), units.BytesSize(float64(sz))))
			ssz += sz
		}
		b.WriteString(" ---\n")
		for _, idx := range s.Indexes {
			indexSize := idx.ComputeSize()
			b.WriteString(fmt.Sprintf(" index %s: %s\n", idx.String(), units.BytesSize(float64(indexSize))))
			ssz += indexSize
		}
		b.WriteString(" ---\n")
		insertionSize := scm.ComputeSize(s.inserts)
		deletionSize := s.deletions.ComputeSize()
		ssz += insertionSize
		ssz += deletionSize
		b.WriteString(fmt.Sprintf(" + insertions %s\n", units.BytesSize(float64(insertionSize))))
		b.WriteString(fmt.Sprintf(" + deletions %s\n", units.BytesSize(float64(deletionSize))))
		b.WriteString(" ---\n")
		b.WriteString(fmt.Sprintf("= total %s\n\n", units.BytesSize(float64(ssz))))
		dsize += ssz
	}
	b.WriteString(fmt.Sprintf("= total %s\n\n", units.BytesSize(float64(dsize))))
	return b.String()
}
