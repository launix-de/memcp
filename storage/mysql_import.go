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

import "context"
import "database/sql"
import "errors"
import "fmt"
import "runtime"
import "strconv"
import "strings"
import "sync"
import "time"

import _ "github.com/go-sql-driver/mysql"
import "github.com/launix-de/memcp/scm"

const mysqlImportMetaSchema = "system"
const mysqlImportTriggersTable = "triggers"

func initMySQLImport(en scm.Env) {
	scm.Declare(&en, &scm.Declaration{
		"mysql_import", "imports schema+data from a MySQL server into MemCP",
		4, 8,
		[]scm.DeclarationParameter{
			{"host", "string", "MySQL host"},
			{"port", "int", "MySQL port"},
			{"username", "string", "MySQL username"},
			{"password", "string", "MySQL password"},
			{"sourcedb", "string|nil", "source database (omit/nil => all non-system dbs)"},
			{"targetdb", "string|nil", "target database (omit/nil => sourcedb)"},
			{"sourcetable", "string|nil", "source table (omit/nil => all tables in sourcedb)"},
			{"targettable", "string|nil", "target table (omit/nil => sourcetable)"},
		},
		"bool",
		func(a ...scm.Scmer) scm.Scmer {
			host := scm.String(a[0])
			port := scm.ToInt(a[1])
			user := scm.String(a[2])
			password := scm.String(a[3])
			var sourceDB, targetDB, sourceTable, targetTable string
			if len(a) > 4 && !a[4].IsNil() {
				sourceDB = scm.String(a[4])
			}
			if len(a) > 5 && !a[5].IsNil() {
				targetDB = scm.String(a[5])
			}
			if len(a) > 6 && !a[6].IsNil() {
				sourceTable = scm.String(a[6])
			}
			if len(a) > 7 && !a[7].IsNil() {
				targetTable = scm.String(a[7])
			}

			if targetDB == "" {
				targetDB = sourceDB
			}
			if targetTable == "" {
				targetTable = sourceTable
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			db, err := openMySQL(ctx, host, port, user, password, "")
			if err != nil {
				panic(err.Error())
			}
			defer db.Close()

			srcDBs, err := mysqlListDatabases(ctx, db, sourceDB)
			if err != nil {
				panic(err.Error())
			}

			type job struct {
				srcDB string
				dstDB string
				srcT  string
				dstT  string
			}
			jobs := make(chan job, 64)
			var wg sync.WaitGroup
			var firstErrMu sync.Mutex
			var firstErr error

			workerCount := runtime.GOMAXPROCS(0)
			if workerCount < 1 {
				workerCount = 1
			}
			if workerCount > 8 {
				workerCount = 8
			}
			for i := 0; i < workerCount; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := range jobs {
						if err := mysqlImportTable(ctx, db, j.srcDB, j.srcT, j.dstDB, j.dstT); err != nil {
							firstErrMu.Lock()
							if firstErr == nil {
								firstErr = err
							}
							firstErrMu.Unlock()
						}
					}
				}()
			}

			for _, src := range srcDBs {
				dst := targetDB
				if dst == "" {
					dst = src
				}
				srcTables, err := mysqlListTables(ctx, db, src, sourceTable)
				if err != nil {
					panic(err.Error())
				}
				for _, st := range srcTables {
					dt := targetTable
					if dt == "" {
						dt = st
					}
					jobs <- job{srcDB: src, dstDB: dst, srcT: st, dstT: dt}
				}
			}
			close(jobs)
			wg.Wait()

			firstErrMu.Lock()
			err = firstErr
			firstErrMu.Unlock()
			if err != nil {
				panic(err.Error())
			}
			return scm.NewBool(true)
		},
		false,
	})
}

func openMySQL(ctx context.Context, host string, port int, user, password, database string) (*sql.DB, error) {
	addr := host + ":" + strconv.Itoa(port)
	dsn := user
	if password != "" {
		dsn += ":" + password
	}
	dsn += "@tcp(" + addr + ")/" + database + "?parseTime=true&multiStatements=true&interpolateParams=true"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(8)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func mysqlListDatabases(ctx context.Context, db *sql.DB, wanted string) ([]string, error) {
	if wanted != "" {
		return []string{wanted}, nil
	}
	rows, err := db.QueryContext(ctx, "SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		switch strings.ToLower(name) {
		case "mysql", "information_schema", "performance_schema", "sys":
			continue
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

func mysqlListTables(ctx context.Context, db *sql.DB, schema string, wanted string) ([]string, error) {
	if wanted != "" {
		return []string{wanted}, nil
	}
	rows, err := db.QueryContext(ctx, "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA=? AND TABLE_TYPE='BASE TABLE'", schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

func mysqlImportTable(ctx context.Context, db *sql.DB, srcDB, srcTable, dstDB, dstTable string) error {
	if dstDB == "" {
		return errors.New("target database must not be empty")
	}
	if dstTable == "" {
		return errors.New("target table must not be empty")
	}

	CreateDatabase(dstDB, true)
	dst := GetDatabase(dstDB)
	if dst == nil {
		return errors.New("could not create target database: " + dstDB)
	}

	t, created := CreateTable(dstDB, dstTable, Safe, true)
	if created {
		if err := mysqlImportColumns(ctx, db, srcDB, srcTable, t); err != nil {
			return err
		}
		if err := mysqlImportConstraints(ctx, db, srcDB, srcTable, t); err != nil {
			return err
		}
		if err := mysqlImportTriggers(ctx, db, srcDB, srcTable, dstDB, dstTable); err != nil {
			return err
		}
	} else {
		if err := mysqlImportConstraints(ctx, db, srcDB, srcTable, t); err != nil {
			return err
		}
		if err := mysqlImportTriggers(ctx, db, srcDB, srcTable, dstDB, dstTable); err != nil {
			return err
		}
	}

	cols := make([]string, 0, len(t.Columns))
	for _, c := range t.Columns {
		if !c.IsTemp {
			cols = append(cols, c.Name)
		}
	}
	if len(cols) == 0 {
		return nil
	}
	return mysqlCopyData(ctx, db, srcDB, srcTable, t, cols)
}

func mysqlImportColumns(ctx context.Context, db *sql.DB, srcDB, srcTable string, t *table) error {
	rows, err := db.QueryContext(ctx, `
SELECT
  COLUMN_NAME,
  DATA_TYPE,
  COLUMN_TYPE,
  IS_NULLABLE,
  COLUMN_DEFAULT,
  EXTRA,
  COLUMN_COMMENT,
  COLLATION_NAME,
  CHARACTER_MAXIMUM_LENGTH,
  NUMERIC_PRECISION,
  NUMERIC_SCALE
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA=? AND TABLE_NAME=?
ORDER BY ORDINAL_POSITION`, srcDB, srcTable)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name, dataType, columnType, isNullable, extra, comment sql.NullString
		var def sql.NullString
		var collation sql.NullString
		var charMax sql.NullInt64
		var numPrec sql.NullInt64
		var numScale sql.NullInt64
		if err := rows.Scan(&name, &dataType, &columnType, &isNullable, &def, &extra, &comment, &collation, &charMax, &numPrec, &numScale); err != nil {
			return err
		}

		typ, dims := mysqlMapType(dataType.String, columnType.String, charMax, numPrec, numScale)
		opts := make([]scm.Scmer, 0, 16)
		if isNullable.Valid && strings.EqualFold(isNullable.String, "NO") {
			opts = append(opts, scm.NewString("null"), scm.NewBool(false))
		}
		if def.Valid {
			opts = append(opts, scm.NewString("default"), scm.NewString(def.String))
		}
		if extra.Valid && strings.Contains(strings.ToLower(extra.String), "auto_increment") {
			opts = append(opts, scm.NewString("auto_increment"), scm.NewBool(true))
		}
		if comment.Valid && comment.String != "" {
			opts = append(opts, scm.NewString("comment"), scm.NewString(comment.String))
		}
		if collation.Valid && collation.String != "" {
			opts = append(opts, scm.NewString("collate"), scm.NewString(collation.String))
		}

		if ok := t.CreateColumn(name.String, typ, dims, opts); !ok {
			continue
		}
	}
	return rows.Err()
}

func mysqlMapType(dataType string, columnType string, charMax, numPrec, numScale sql.NullInt64) (string, []int) {
	switch strings.ToLower(dataType) {
	case "tinyint", "smallint", "mediumint", "int", "integer", "bigint":
		return "int", nil
	case "float", "double":
		return "float", nil
	case "decimal", "numeric":
		if numPrec.Valid && numScale.Valid {
			return "decimal", []int{int(numPrec.Int64), int(numScale.Int64)}
		}
		return "decimal", nil
	case "date":
		return "date", nil
	case "datetime", "timestamp":
		return "datetime", nil
	case "time":
		return "time", nil
	case "char", "varchar":
		if charMax.Valid {
			return "varchar", []int{int(charMax.Int64)}
		}
		return "varchar", nil
	case "text", "tinytext", "mediumtext", "longtext":
		return "text", nil
	case "blob", "tinyblob", "mediumblob", "longblob", "binary", "varbinary":
		return "blob", nil
	case "json":
		return "json", nil
	case "enum", "set":
		return "varchar", nil
	default:
		_ = columnType
		return "varchar", nil
	}
}

func mysqlImportConstraints(ctx context.Context, db *sql.DB, srcDB, srcTable string, t *table) error {
	type key struct {
		name   string
		unique bool
	}
	keys := map[key][]struct {
		seq int
		col string
	}{}

	rows, err := db.QueryContext(ctx, `
SELECT
  tc.CONSTRAINT_NAME,
  tc.CONSTRAINT_TYPE,
  kcu.COLUMN_NAME,
  kcu.ORDINAL_POSITION
FROM information_schema.TABLE_CONSTRAINTS tc
JOIN information_schema.KEY_COLUMN_USAGE kcu
  ON tc.CONSTRAINT_SCHEMA=kcu.CONSTRAINT_SCHEMA
 AND tc.TABLE_NAME=kcu.TABLE_NAME
 AND tc.CONSTRAINT_NAME=kcu.CONSTRAINT_NAME
WHERE tc.CONSTRAINT_SCHEMA=?
  AND tc.TABLE_NAME=?
  AND tc.CONSTRAINT_TYPE IN ('PRIMARY KEY','UNIQUE')
ORDER BY tc.CONSTRAINT_NAME, kcu.ORDINAL_POSITION`, srcDB, srcTable)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cname, ctype, col string
		var pos int
		if err := rows.Scan(&cname, &ctype, &col, &pos); err != nil {
			return err
		}
		k := key{name: cname, unique: true}
		keys[k] = append(keys[k], struct {
			seq int
			col string
		}{seq: pos, col: col})
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	t.schema.schemalock.Lock()
	defer t.schema.schemalock.Unlock()

existingLoop:
	for k, cols := range keys {
		colNames := make([]string, len(cols))
		for i := range cols {
			colNames[i] = cols[i].col
		}
		for _, u := range t.Unique {
			if u.Id == k.name && sameStringSlice(u.Cols, colNames) {
				continue existingLoop
			}
		}
		t.Unique = append(t.Unique, uniqueKey{k.name, colNames})
	}
	t.schema.save()
	return nil
}

func sameStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func mysqlEnsureTriggerTable() *table {
	CreateDatabase(mysqlImportMetaSchema, true)
	t, created := CreateTable(mysqlImportMetaSchema, mysqlImportTriggersTable, Safe, true)
	if created {
		_ = t.CreateColumn("schema", "varchar", nil, []scm.Scmer{scm.NewString("null"), scm.NewBool(false)})
		_ = t.CreateColumn("table", "varchar", nil, []scm.Scmer{scm.NewString("null"), scm.NewBool(false)})
		_ = t.CreateColumn("name", "varchar", nil, []scm.Scmer{scm.NewString("null"), scm.NewBool(false)})
		_ = t.CreateColumn("timing", "varchar", nil, []scm.Scmer{scm.NewString("null"), scm.NewBool(false)})
		_ = t.CreateColumn("event", "varchar", nil, []scm.Scmer{scm.NewString("null"), scm.NewBool(false)})
		_ = t.CreateColumn("statement", "text", nil, []scm.Scmer{scm.NewString("null"), scm.NewBool(false)})
	}
	return t
}

func mysqlImportTriggers(ctx context.Context, db *sql.DB, srcDB, srcTable, dstDB, dstTable string) error {
	meta := mysqlEnsureTriggerTable()
	if meta == nil {
		return errors.New("could not create system.triggers")
	}

	rows, err := db.QueryContext(ctx, `
SELECT
  TRIGGER_NAME,
  ACTION_TIMING,
  EVENT_MANIPULATION,
  ACTION_STATEMENT
FROM information_schema.TRIGGERS
WHERE TRIGGER_SCHEMA=? AND EVENT_OBJECT_TABLE=?`, srcDB, srcTable)
	if err != nil {
		return err
	}
	defer rows.Close()

	var batch [][]scm.Scmer
	for rows.Next() {
		var name, timing, event, stmt string
		if err := rows.Scan(&name, &timing, &event, &stmt); err != nil {
			return err
		}
		batch = append(batch, []scm.Scmer{
			scm.NewString(dstDB),
			scm.NewString(dstTable),
			scm.NewString(name),
			scm.NewString(timing),
			scm.NewString(event),
			scm.NewString(stmt),
		})
		if len(batch) >= 512 {
			meta.Insert([]string{"schema", "table", "name", "timing", "event", "statement"}, batch, nil, scm.NewNil(), false, nil)
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		meta.Insert([]string{"schema", "table", "name", "timing", "event", "statement"}, batch, nil, scm.NewNil(), false, nil)
	}
	return rows.Err()
}

func mysqlCopyData(ctx context.Context, db *sql.DB, srcDB, srcTable string, t *table, columns []string) error {
	quotedCols := make([]string, len(columns))
	for i, c := range columns {
		quotedCols[i] = "`" + strings.ReplaceAll(c, "`", "``") + "`"
	}
	query := "SELECT " + strings.Join(quotedCols, ",") + " FROM `" + strings.ReplaceAll(srcDB, "`", "``") + "`.`" + strings.ReplaceAll(srcTable, "`", "``") + "`"

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return err
	}
	_ = colTypes

	const batchSize = 2048
	values := make([][]scm.Scmer, 0, batchSize)
	raw := make([]any, len(columns))
	rawPtrs := make([]any, len(columns))
	for i := range raw {
		rawPtrs[i] = &raw[i]
	}

	for rows.Next() {
		if err := rows.Scan(rawPtrs...); err != nil {
			return err
		}
		row := make([]scm.Scmer, len(columns))
		for i := range raw {
			row[i] = mysqlToScmer(raw[i])
		}
		values = append(values, row)
		if len(values) >= batchSize {
			t.Insert(columns, values, nil, scm.NewNil(), false, nil)
			values = values[:0]
		}
	}
	if len(values) > 0 {
		t.Insert(columns, values, nil, scm.NewNil(), false, nil)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	fmt.Println("mysql_import:", srcDB+"."+srcTable, "->", t.schema.Name+"."+t.Name)
	return nil
}

func mysqlToScmer(v any) scm.Scmer {
	switch x := v.(type) {
	case nil:
		return scm.NewNil()
	case int64:
		return scm.NewInt(x)
	case float64:
		return scm.NewFloat(x)
	case bool:
		return scm.NewBool(x)
	case []byte:
		return scm.NewString(string(x))
	case string:
		return scm.NewString(x)
	case time.Time:
		return scm.NewString(x.Format("2006-01-02 15:04:05"))
	default:
		return scm.NewString(fmt.Sprint(v))
	}
}
