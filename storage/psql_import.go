/*
Copyright (C) 2023, 2024, 2025  Carl-Philip Hänsch

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

import _ "github.com/lib/pq"
import "github.com/launix-de/memcp/scm"

func initPSQLImport(en scm.Env) {
		scm.Declare(&en, &scm.Declaration{
		Name: "psql_import",
		Desc: "imports schema+data from a PostgreSQL server into MemCP",
		Fn: func(a ...scm.Scmer) scm.Scmer {
				host := "127.0.0.1"
				if !a[0].IsNil() {
					host = scm.String(a[0])
				}
				port := 5432
				if !a[1].IsNil() {
					port = scm.ToInt(a[1])
				}
				user := scm.String(a[2])
				password := scm.String(a[3])
				var sourceDB, sourceSchema, targetDB, sourceTable, targetTable string
				if len(a) > 4 && !a[4].IsNil() {
					sourceDB = scm.String(a[4])
				}
				if len(a) > 5 && !a[5].IsNil() {
					sourceSchema = scm.String(a[5])
				}
				if len(a) > 6 && !a[6].IsNil() {
					targetDB = scm.String(a[6])
				}
				if len(a) > 7 && !a[7].IsNil() {
					sourceTable = scm.String(a[7])
				}
				if len(a) > 8 && !a[8].IsNil() {
					targetTable = scm.String(a[8])
				}
	
				if targetDB == "" {
					targetDB = sourceDB
				}
				if targetTable == "" {
					targetTable = sourceTable
				}
	
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()
	
				// connect to default "postgres" db to list databases
				listDB, err := openPSQL(ctx, host, port, user, password, "postgres")
				if err != nil {
					panic(err.Error())
				}
				defer listDB.Close()
	
				srcDBs, err := psqlListDatabases(ctx, listDB, sourceDB)
				if err != nil {
					panic(err.Error())
				}
				listDB.Close()
	
				type job struct {
					srcDB     string
					srcSchema string
					srcTable  string
					dstDB     string
					dstTable  string
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
						var workerConn *sql.DB
						var workerDB string
						defer func() {
							if workerConn != nil {
								workerConn.Close()
							}
						}()
						for j := range jobs {
							if j.srcDB != workerDB {
								if workerConn != nil {
									workerConn.Close()
								}
								workerConn, err = openPSQL(ctx, host, port, user, password, j.srcDB)
								if err != nil {
									firstErrMu.Lock()
									if firstErr == nil {
										firstErr = err
									}
									firstErrMu.Unlock()
									workerConn = nil
									workerDB = ""
									continue
								}
								workerDB = j.srcDB
							}
							if workerConn == nil {
								continue
							}
							if e := psqlImportTable(ctx, workerConn, j.srcSchema, j.srcTable, j.dstDB, j.dstTable); e != nil {
								firstErrMu.Lock()
								if firstErr == nil {
									firstErr = e
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
					// connect temporarily to list schemas
					schemaConn, err := openPSQL(ctx, host, port, user, password, src)
					if err != nil {
						panic(err.Error())
					}
					schemas, err := psqlListSchemas(ctx, schemaConn, sourceSchema)
					schemaConn.Close()
					if err != nil {
						panic(err.Error())
					}
	
					for _, schema := range schemas {
						// for each schema we need a separate connection to list tables
						tableConn, err := openPSQL(ctx, host, port, user, password, src)
						if err != nil {
							panic(err.Error())
						}
						tables, err := psqlListTables(ctx, tableConn, schema, sourceTable)
						tableConn.Close()
						if err != nil {
							panic(err.Error())
						}
						for _, st := range tables {
							dt := targetTable
							if dt == "" {
								dt = st
							}
							jobs <- job{srcDB: src, srcSchema: schema, srcTable: st, dstDB: dst, dstTable: dt}
						}
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
		Type: &scm.TypeDescriptor{
			Params: []*scm.TypeDescriptor{&scm.TypeDescriptor{Kind: "string|nil", ParamName: "host", ParamDesc: "PostgreSQL host (nil => 127.0.0.1)"}, &scm.TypeDescriptor{Kind: "int|nil", ParamName: "port", ParamDesc: "PostgreSQL port (nil => 5432)"}, &scm.TypeDescriptor{Kind: "string", ParamName: "username", ParamDesc: "PostgreSQL username"}, &scm.TypeDescriptor{Kind: "string", ParamName: "password", ParamDesc: "PostgreSQL password"}, &scm.TypeDescriptor{Kind: "string|nil", ParamName: "sourcedb", ParamDesc: "source database (omit/nil => all non-system dbs)", Optional: true}, &scm.TypeDescriptor{Kind: "string|nil", ParamName: "sourceschema", ParamDesc: "source schema (omit/nil => all non-system schemas in sourcedb)", Optional: true}, &scm.TypeDescriptor{Kind: "string|nil", ParamName: "targetdb", ParamDesc: "target database (omit/nil => sourcedb)", Optional: true}, &scm.TypeDescriptor{Kind: "string|nil", ParamName: "sourcetable", ParamDesc: "source table (omit/nil => all tables in sourceschema)", Optional: true}, &scm.TypeDescriptor{Kind: "string|nil", ParamName: "targettable", ParamDesc: "target table (omit/nil => sourcetable)", Optional: true}},
			Return: &scm.TypeDescriptor{Kind: "bool"},
		},
	})
}

func openPSQL(ctx context.Context, host string, port int, user, password, database string) (*sql.DB, error) {
	dsn := "host=" + host +
		" port=" + strconv.Itoa(port) +
		" user=" + pqQuoteIdent(user) +
		" sslmode=disable"
	if password != "" {
		dsn += " password=" + pqQuoteIdent(password)
	}
	if database != "" {
		dsn += " dbname=" + pqQuoteIdent(database)
	}
	db, err := sql.Open("postgres", dsn)
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

// pqQuoteIdent wraps a string in single quotes for the DSN, escaping single quotes and backslashes.
func pqQuoteIdent(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	return "'" + s + "'"
}

func psqlListDatabases(ctx context.Context, db *sql.DB, wanted string) ([]string, error) {
	if wanted != "" {
		return []string{wanted}, nil
	}
	rows, err := db.QueryContext(ctx, "SELECT datname FROM pg_database WHERE datistemplate = false AND datname NOT IN ('postgres')")
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

func psqlListSchemas(ctx context.Context, db *sql.DB, wanted string) ([]string, error) {
	if wanted != "" {
		return []string{wanted}, nil
	}
	rows, err := db.QueryContext(ctx, `
SELECT schema_name
FROM information_schema.schemata
WHERE schema_name NOT IN ('pg_catalog', 'information_schema')
  AND schema_name NOT LIKE 'pg_%'
ORDER BY schema_name`)
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

func psqlListTables(ctx context.Context, db *sql.DB, schema string, wanted string) ([]string, error) {
	if wanted != "" {
		return []string{wanted}, nil
	}
	rows, err := db.QueryContext(ctx,
		`SELECT table_name FROM information_schema.tables WHERE table_schema=$1 AND table_type='BASE TABLE' ORDER BY table_name`,
		schema)
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

func psqlImportTable(ctx context.Context, db *sql.DB, srcSchema, srcTable, dstDB, dstTable string) error {
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

	// drop-semantics: replace target table content+schema
	DropTable(dstDB, dstTable, true)

	t, _ := CreateTable(dstDB, dstTable, Safe, true)
	if err := psqlImportColumns(ctx, db, srcSchema, srcTable, t); err != nil {
		return err
	}
	if err := psqlImportConstraints(ctx, db, srcSchema, srcTable, t); err != nil {
		return err
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
	return psqlCopyData(ctx, db, srcSchema, srcTable, t, cols)
}

func psqlImportColumns(ctx context.Context, db *sql.DB, srcSchema, srcTable string, t *table) error {
	rows, err := db.QueryContext(ctx, `
SELECT
  column_name,
  data_type,
  udt_name,
  is_nullable,
  column_default,
  character_maximum_length,
  numeric_precision,
  numeric_scale
FROM information_schema.columns
WHERE table_schema=$1 AND table_name=$2
ORDER BY ordinal_position`, srcSchema, srcTable)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name, dataType, udtName, isNullable sql.NullString
		var def sql.NullString
		var charMax, numPrec, numScale sql.NullInt64
		if err := rows.Scan(&name, &dataType, &udtName, &isNullable, &def, &charMax, &numPrec, &numScale); err != nil {
			return err
		}

		typ, dims := psqlMapType(dataType.String, udtName.String, charMax, numPrec, numScale)
		opts := make([]scm.Scmer, 0, 8)
		if isNullable.Valid && strings.EqualFold(isNullable.String, "NO") {
			opts = append(opts, scm.NewString("null"), scm.NewBool(false))
		}
		if def.Valid && def.String != "" {
			opts = append(opts, scm.NewString("default"), scm.NewString(def.String))
		}

		if ok := t.CreateColumn(name.String, typ, dims, opts); !ok {
			continue
		}
	}
	return rows.Err()
}

func psqlMapType(dataType string, udtName string, charMax, numPrec, numScale sql.NullInt64) (string, []int) {
	switch strings.ToLower(dataType) {
	case "smallint", "integer", "bigint",
		"smallserial", "serial", "bigserial":
		return "int", nil
	case "real", "double precision":
		return "float", nil
	case "numeric", "decimal":
		if numPrec.Valid && numScale.Valid {
			return "decimal", []int{int(numPrec.Int64), int(numScale.Int64)}
		}
		return "decimal", nil
	case "boolean":
		return "int", nil
	case "date":
		return "date", nil
	case "timestamp without time zone", "timestamp with time zone", "timestamp":
		return "datetime", nil
	case "time without time zone", "time with time zone", "time":
		return "time", nil
	case "character varying", "character":
		if charMax.Valid {
			return "varchar", []int{int(charMax.Int64)}
		}
		return "varchar", nil
	case "text":
		return "text", nil
	case "bytea":
		return "blob", nil
	case "json", "jsonb":
		return "json", nil
	case "uuid":
		return "varchar", []int{36}
	case "array", "user-defined":
		_ = udtName
		return "text", nil
	default:
		_ = udtName
		return "varchar", nil
	}
}

func psqlImportConstraints(ctx context.Context, db *sql.DB, srcSchema, srcTable string, t *table) error {
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
  tc.constraint_name,
  tc.constraint_type,
  kcu.column_name,
  kcu.ordinal_position
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu
  ON tc.constraint_schema = kcu.constraint_schema
 AND tc.table_name = kcu.table_name
 AND tc.constraint_name = kcu.constraint_name
WHERE tc.constraint_schema = $1
  AND tc.table_name = $2
  AND tc.constraint_type IN ('PRIMARY KEY', 'UNIQUE')
ORDER BY tc.constraint_name, kcu.ordinal_position`, srcSchema, srcTable)
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

func psqlCopyData(ctx context.Context, db *sql.DB, srcSchema, srcTable string, t *table, columns []string) error {
	quotedCols := make([]string, len(columns))
	for i, c := range columns {
		quotedCols[i] = `"` + strings.ReplaceAll(c, `"`, `""`) + `"`
	}
	query := `SELECT ` + strings.Join(quotedCols, ",") +
		` FROM "` + strings.ReplaceAll(srcSchema, `"`, `""`) + `"."` + strings.ReplaceAll(srcTable, `"`, `""`) + `"`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

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
			row[i] = psqlToScmer(raw[i])
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
	fmt.Println("psql_import:", srcSchema+"."+srcTable, "->", t.schema.Name+"."+t.Name)
	return nil
}

func psqlToScmer(v any) scm.Scmer {
	switch x := v.(type) {
	case nil:
		return scm.NewNil()
	case int64:
		return scm.NewInt(x)
	case float64:
		return scm.NewFloat(x)
	case bool:
		if x {
			return scm.NewInt(1)
		}
		return scm.NewInt(0)
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
