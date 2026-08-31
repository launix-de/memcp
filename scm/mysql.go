/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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

import "context"
import "os"
import "fmt"
import "net"
import "sync"
import "strings"
import "github.com/launix-de/go-mysqlstack/sqldb"
import "runtime"
import "sync/atomic"
import "github.com/launix-de/go-mysqlstack/xlog"
import "github.com/launix-de/go-mysqlstack/driver"
import querypb "github.com/launix-de/go-mysqlstack/sqlparser/depends/query"
import "github.com/launix-de/go-mysqlstack/sqlparser/depends/sqltypes"

type mysqlCloser interface {
	Close()
}

var mysqlListenersMu sync.Mutex
var mysqlListeners []mysqlCloser

// build this function into your SCM environment to offer http server capabilities
func MySQLServe(a ...Scmer) Scmer {
	// params: port, authcallback, schemacallback, querycallback
	port := a[0].String()

	log := xlog.NewStdLog(xlog.Level(xlog.INFO))
	var handler MySQLWrapper
	handler.log = log
	handler.authcallback = a[1]
	handler.schemacallback = a[2]
	handler.querycallback = a[3]

	mysql, err := driver.NewListener(log, fmt.Sprintf(":%v", port), &handler)
	if err != nil {
		panic(err)
	}
	mysqlListenersMu.Lock()
	mysqlListeners = append(mysqlListeners, mysql)
	mysqlListenersMu.Unlock()
	go func() {
		defer mysql.Close()
		mysql.Accept()
	}()
	return NewBool(true)
}

// MySQLServeSocket listens on a Unix domain socket for MySQL protocol.
func MySQLServeSocket(a ...Scmer) Scmer {
	socketPath := a[0].String()

	// Remove stale socket file
	os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		panic(err)
	}

	// Make socket accessible to all local users
	os.Chmod(socketPath, 0777)

	log := xlog.NewStdLog(xlog.Level(xlog.INFO))
	var handler MySQLWrapper
	handler.log = log
	handler.authcallback = a[1]
	handler.schemacallback = a[2]
	handler.querycallback = a[3]

	mysql := driver.NewListenerFromNetListener(log, listener, &handler)
	mysqlListenersMu.Lock()
	mysqlListeners = append(mysqlListeners, mysql)
	mysqlListenersMu.Unlock()
	go func() {
		defer mysql.Close()
		mysql.Accept()
	}()
	return NewBool(true)
}

// driver.CreatePassword helper function
func MySQLPassword(a ...Scmer) Scmer {
	return NewString(string(driver.CreatePassword(a[0].String())))
}

type MySQLWrapper struct {
	log            *xlog.Log
	authcallback   Scmer
	schemacallback Scmer
	querycallback  Scmer
}

func mysqlScmSession(session *driver.Session) Scmer {
	if scmSessionAny, ok := mysqlsessions.Load(session.ID()); ok {
		return NewFunc(scmSessionAny.(func(...Scmer) Scmer))
	}
	newSession := NewSession().Func()
	mysqlsessions.Store(session.ID(), newSession)
	return NewFunc(newSession)
}

func withMySQLScmSession(session *driver.Session, fn func()) {
	SetValues(map[string]any{
		"session": mysqlScmSession(session),
		"context": context.Background(),
	}, fn)
}

/* session storage -> map from session id to SCM session object */
var mysqlsessions sync.Map

// mysqlStates maps driver session ID -> *SessionState for SHOW PROCESSLIST
var mysqlStates sync.Map

func refreshMySQLSessionProcesslistMeta(session *driver.Session) {
	if v, ok := mysqlStates.Load(session.ID()); ok {
		ss := v.(*SessionState)
		if user := session.User(); user != "" {
			ss.User = user
		}
		ss.SetDB(session.Schema())
	}
}

func (m *MySQLWrapper) ServerVersion() string {
	// MySQL clients parse the leading numeric version before enabling feature
	// probes such as SHOW CREATE TRIGGER. Keep the compatibility baseline
	// explicit and retain MemCP in the vendor suffix.
	return "5.7.44-MemCP"
}
func (m *MySQLWrapper) SetServerVersion() {
}
func (m *MySQLWrapper) NewSession(session *driver.Session) {
	m.log.Info("%s", "New Session from "+session.Addr())
	mysqlsessions.Store(session.ID(), NewSession().Func())
	ss := RegisterSession(session.User(), session.Addr(), session.Schema())
	mysqlStates.Store(session.ID(), ss)
	refreshMySQLSessionProcesslistMeta(session)
}
func (m *MySQLWrapper) SessionInc(session *driver.Session) {
	// I think we can skip session counting
}
func (m *MySQLWrapper) SessionDec(session *driver.Session) {
	// I think we can skip session counting
}
func (m *MySQLWrapper) SessionClosed(session *driver.Session) {
	m.log.Info("%s", "Closed Session "+session.User()+" from "+session.Addr())
	mysqlsessions.Delete(session.ID())
	if v, ok := mysqlStates.LoadAndDelete(session.ID()); ok {
		st := v.(*SessionState)
		st.ReleaseAllLocks()
		UnregisterSession(st.ID)
	}
}
func (m *MySQLWrapper) SessionCheck(session *driver.Session) error {
	// we could reject clients here when server load is too full
	return nil
}

func (m *MySQLWrapper) AuthCheck(session *driver.Session) error {
	m.log.Info("%s", "Auth Check with "+session.User())
	// callback should load password from database
	var password Scmer
	withMySQLScmSession(session, func() {
		password = Apply(m.authcallback, NewString(session.User()))
	})
	if password.IsNil() {
		// user does not exist
		return sqldb.NewSQLError(sqldb.ER_ACCESS_DENIED_ERROR, session.User(), session.Addr(), "YES")
	}
	if !session.TestPassword([]byte(password.String())) {
		return sqldb.NewSQLError(sqldb.ER_ACCESS_DENIED_ERROR, session.User(), session.Addr(), "YES")
	}
	refreshMySQLSessionProcesslistMeta(session)
	return nil
}
func (m *MySQLWrapper) ComInitDB(session *driver.Session, database string) error {
	m.log.Info("%s", "db "+database)
	var allowed Scmer
	withMySQLScmSession(session, func() {
		allowed = Apply(m.schemacallback, NewString(session.User()), NewString(database))
	})
	if !allowed.Bool() {
		return sqldb.NewSQLErrorf(sqldb.ER_ACCESS_DENIED_ERROR, "access denied for database %s", database)
	}
	session.SetSchema(database)
	refreshMySQLSessionProcesslistMeta(session)
	return nil
}

func MySQLToScmer(v sqltypes.Value) Scmer {
	if v.IsNull() {
		return NewNil()
	}
	switch {
	case v.IsIntegral():
		n, err := v.ParseInt64()
		if err == nil {
			return NewInt(n)
		}
	case v.IsFloat():
		f, err := v.ParseFloat64()
		if err == nil {
			return NewFloat(f)
		}
	}
	return NewString(v.ToString())
}

type ErrorWrapper string

func (s ErrorWrapper) Error() string {
	return string(s)
}

const mysqlClientErrorMessageLimit = 2048

// mysqlClientErrorMessage keeps protocol errors below mysqlnd's fixed command
// buffer. Internal panics include their full stack for the server log, but the
// first line contains the actionable cause and is the only part clients need.
func mysqlClientErrorMessage(message string) string {
	if lineEnd := strings.IndexByte(message, '\n'); lineEnd >= 0 {
		message = message[:lineEnd]
	}
	if len(message) <= mysqlClientErrorMessageLimit {
		return message
	}
	return message[:mysqlClientErrorMessageLimit]
}

func updateMySQLFieldMetadata(field *querypb.Field, val Scmer) {
	field.Charset = 0
	switch val.GetTag() {
	case tagFloat:
		field.Type = querypb.Type_FLOAT64
	case tagInt:
		field.Type = querypb.Type_INT64
	case tagBool:
		field.Type = querypb.Type_INT32
	case tagNil:
		field.Type = querypb.Type_NULL_TYPE
	default:
		field.Type = querypb.Type_VARCHAR
		field.Charset = 45 // utf8mb4_general_ci
	}
}

func prepareMySQLResultRow(fields *[]*querypb.Field, colmap map[string]int, item []Scmer, row []Scmer, schemaInitialized bool, refineNullTypes bool) ([]Scmer, bool) {
	if schemaInitialized {
		if len(item) == len(*fields)*2 {
			ordered := true
			for i, field := range *fields {
				if item[i*2].String() != field.Name {
					ordered = false
					break
				}
			}
			if ordered {
				for i := range *fields {
					val := item[i*2+1]
					row[i] = val
					if refineNullTypes && (*fields)[i].Type == querypb.Type_NULL_TYPE && !val.IsNil() {
						updateMySQLFieldMetadata((*fields)[i], val)
					}
				}
				return row, false
			}
		}
		for i := range row {
			row[i] = NewNil()
		}
	} else {
		row = row[:0]
	}
	unknownColumn := false
	for i := 0; i < len(item)-1; i += 2 {
		val := item[i+1]
		colname := item[i].String()
		colid, ok := colmap[colname]
		if ok {
			row[colid] = val
			if !schemaInitialized || (refineNullTypes && (*fields)[colid].Type == querypb.Type_NULL_TYPE && !val.IsNil()) {
				updateMySQLFieldMetadata((*fields)[colid], val)
			}
		} else if schemaInitialized {
			unknownColumn = true
		} else {
			colmap[colname] = len(*fields)
			newcol := new(querypb.Field)
			newcol.Name = colname
			updateMySQLFieldMetadata(newcol, val)
			*fields = append(*fields, newcol)
			row = append(row, val)
		}
	}
	return row, unknownColumn
}

func appendScmerToMySQLRow(row *driver.RowWriter, val Scmer) {
	switch val.GetTag() {
	case tagNil:
		row.Null()
	case tagFloat:
		row.Float64(val.Float())
	case tagInt:
		row.Int64(val.Int())
	case tagBool:
		row.Bool(val.Bool())
	default:
		row.String(val.String())
	}
}

const mysqlFieldDiscoveryRows = 1024

func mysqlFieldsResolved(fields []*querypb.Field) bool {
	for _, field := range fields {
		if field.Type == querypb.Type_NULL_TYPE {
			return false
		}
	}
	return true
}

func isSelectQuery(query string) bool {
	trimmed := strings.TrimSpace(query)
	for {
		if strings.HasPrefix(trimmed, "/*") {
			end := strings.Index(trimmed, "*/")
			if end == -1 {
				return false
			}
			trimmed = strings.TrimSpace(trimmed[end+2:])
			continue
		}
		if strings.HasPrefix(trimmed, "--") {
			end := strings.Index(trimmed, "\n")
			if end == -1 {
				return false
			}
			trimmed = strings.TrimSpace(trimmed[end+1:])
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			end := strings.Index(trimmed, "\n")
			if end == -1 {
				return false
			}
			trimmed = strings.TrimSpace(trimmed[end+1:])
			continue
		}
		break
	}
	if len(trimmed) < len("select") || !strings.EqualFold(trimmed[:len("select")], "select") {
		return false
	}
	if len(trimmed) == len("select") {
		return true
	}
	switch trimmed[len("select")] {
	case ' ', '\t', '\r', '\n', '(', '/':
		return true
	default:
		return false
	}
}

func consumeMySQLKeyword(input string, keyword string) (string, bool) {
	input = strings.TrimSpace(input)
	if len(input) < len(keyword) || !strings.EqualFold(input[:len(keyword)], keyword) {
		return input, false
	}
	if len(input) > len(keyword) {
		switch input[len(keyword)] {
		case ' ', '\t', '\r', '\n':
		default:
			return input, false
		}
	}
	return input[len(keyword):], true
}

func parseMySQLIdentifier(input string) (identifier string, rest string, ok bool) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", input, false
	}
	if input[0] == '`' {
		var value strings.Builder
		for i := 1; i < len(input); i++ {
			if input[i] != '`' {
				value.WriteByte(input[i])
				continue
			}
			if i+1 < len(input) && input[i+1] == '`' {
				value.WriteByte('`')
				i++
				continue
			}
			return value.String(), input[i+1:], true
		}
		return "", input, false
	}
	end := 0
	for end < len(input) {
		ch := input[end]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '_' || ch == '$' {
			end++
			continue
		}
		break
	}
	if end == 0 {
		return "", input, false
	}
	return input[:end], input[end:], true
}

func parseMySQLTableProjection(query string, defaultSchema string) (schema string, table string, columns []string, ok bool) {
	rest, ok := consumeMySQLKeyword(query, "SELECT")
	if !ok {
		return "", "", nil, false
	}
	rest = strings.TrimSpace(rest)
	for strings.HasPrefix(rest, "/*") {
		end := strings.Index(rest[2:], "*/")
		if end < 0 {
			return "", "", nil, false
		}
		rest = strings.TrimSpace(rest[end+4:])
	}
	if rest != "" && rest[0] == '*' {
		rest = rest[1:]
	} else {
		for {
			column, tail, parsed := parseMySQLIdentifier(rest)
			if !parsed {
				return "", "", nil, false
			}
			tail = strings.TrimSpace(tail)
			if strings.HasPrefix(tail, ".") {
				column, tail, parsed = parseMySQLIdentifier(tail[1:])
				if !parsed {
					return "", "", nil, false
				}
				tail = strings.TrimSpace(tail)
			}
			columns = append(columns, column)
			if !strings.HasPrefix(tail, ",") {
				rest = tail
				break
			}
			rest = tail[1:]
		}
	}
	rest, ok = consumeMySQLKeyword(rest, "FROM")
	if !ok {
		return "", "", nil, false
	}
	first, rest, ok := parseMySQLIdentifier(rest)
	if !ok {
		return "", "", nil, false
	}
	rest = strings.TrimSpace(rest)
	if strings.HasPrefix(rest, ".") {
		second, tail, parsed := parseMySQLIdentifier(rest[1:])
		if !parsed {
			return "", "", nil, false
		}
		schema, table, rest = first, second, tail
	} else {
		schema, table = defaultSchema, first
	}
	rest = strings.TrimSpace(rest)
	if strings.HasSuffix(rest, ";") {
		rest = strings.TrimSpace(rest[:len(rest)-1])
	}
	return schema, table, columns, rest == "" && schema != "" && table != ""
}

func mysqlShowColumnValue(row []Scmer, key string) (Scmer, bool) {
	for i := 0; i+1 < len(row); i += 2 {
		if row[i].String() == key {
			return row[i+1], true
		}
	}
	return NewNil(), false
}

func mysqlFieldFromShowColumn(row []Scmer) (*querypb.Field, bool) {
	name, ok := mysqlShowColumnValue(row, "Field")
	if !ok || name.String() == "" {
		return nil, false
	}
	rawType, ok := mysqlShowColumnValue(row, "RawType")
	if !ok {
		rawType, _ = mysqlShowColumnValue(row, "Type")
	}
	typeName := strings.ToUpper(rawType.String())
	if end := strings.IndexAny(typeName, "( "); end >= 0 {
		typeName = typeName[:end]
	}
	field := &querypb.Field{Name: name.String()}
	switch typeName {
	case "BOOL", "BOOLEAN":
		field.Type = querypb.Type_INT32
	case "TINYINT", "SMALLINT", "MEDIUMINT", "INT", "INTEGER", "BIGINT":
		field.Type = querypb.Type_INT64
	case "FLOAT":
		field.Type = querypb.Type_FLOAT32
	case "DOUBLE", "REAL":
		field.Type = querypb.Type_FLOAT64
	case "DECIMAL", "NUMERIC":
		field.Type = querypb.Type_DECIMAL
	case "BINARY", "VARBINARY":
		field.Type = querypb.Type_VARBINARY
		field.Charset = 63
	case "BLOB", "TINYBLOB", "MEDIUMBLOB", "LONGBLOB":
		field.Type = querypb.Type_BLOB
		field.Charset = 63
	case "DATE":
		field.Type = querypb.Type_DATE
	case "DATETIME", "TIMESTAMP":
		field.Type = querypb.Type_DATETIME
	case "TIME":
		field.Type = querypb.Type_TIME
	default:
		field.Type = querypb.Type_VARCHAR
		field.Charset = 45
	}
	return field, true
}

func mysqlFieldsForEmptyTableSelect(query string, defaultSchema string) (fields []*querypb.Field) {
	schema, table, projectedColumns, ok := parseMySQLTableProjection(query, defaultSchema)
	if !ok {
		return nil
	}
	declaration, ok := Globalenv.Vars[Symbol("show")]
	if !ok {
		return nil
	}
	defer func() {
		if recover() != nil {
			fields = nil
		}
	}()
	byName := make(map[string]*querypb.Field)
	for _, column := range Apply(declaration, NewString(schema), NewString(table)).Slice() {
		field, valid := mysqlFieldFromShowColumn(column.Slice())
		if !valid {
			return nil
		}
		byName[field.Name] = field
		if len(projectedColumns) == 0 {
			fields = append(fields, field)
		}
	}
	if len(projectedColumns) != 0 {
		for _, name := range projectedColumns {
			field, exists := byName[name]
			if !exists {
				return nil
			}
			fields = append(fields, field)
		}
	}
	return fields
}

func (m *MySQLWrapper) ComQuery(session *driver.Session, query string, bindVariables map[string]*querypb.BindVariable, output *driver.ResultWriter) (myerr error) {
	atomic.AddInt64(&TotalHTTPRequests, 1)
	var ss *SessionState
	var querySeq uint64
	queryCtx, queryCancel := context.WithCancel(context.Background())
	defer queryCancel()
	if v, ok := mysqlStates.Load(session.ID()); ok {
		ss = v.(*SessionState)
		refreshMySQLSessionProcesslistMeta(session)
		ss.Touch()
		querySeq = ss.BeginQuery("Query", query)
		ss.SetCancel(querySeq, queryCancel)
		ss.SetQueryContext(querySeq, queryCtx)
	}
	cancelID := session.SetQueryCancel(func() {
		queryCancel()
		if ss != nil {
			ss.KillQuery(querySeq)
		}
	})
	defer session.ClearQueryCancel(cancelID)
	defer func() {
		if ss != nil {
			ss.EndQuery(querySeq, "Sleep", "")
			ss.SetDB(session.Schema())
		}
	}()
	// max_allowed_packet: PHP PDO queries this to size buffers.
	// Return 40MB so large result sets work without tripping client-side
	// packet buffer limits on wide login/dashboard views.
	if query == "select @@max_allowed_packet" || query == "SELECT @@max_allowed_packet" {
		return output.WriteResult(&sqltypes.Result{
			Fields: []*querypb.Field{
				{Name: "@@max_allowed_packet", Type: querypb.Type_INT64},
			},
			Rows: [][]sqltypes.Value{
				{sqltypes.MakeTrusted(querypb.Type_INT64, []byte("41943040"))},
			},
		})
	}
	if query == "select @@version_comment limit 1" {
		return output.WriteResult(&sqltypes.Result{
			Fields: []*querypb.Field{
				{Name: "@@version_comment", Type: querypb.Type_TEXT, Charset: 45},
			},
			Rows: [][]sqltypes.Value{
				{sqltypes.MakeTrusted(querypb.Type_TEXT, []byte(runtime.GOOS))},
			},
		})
	}
	selectQuery := isSelectQuery(query)
	colmap := make(map[string]int)
	var fields []*querypb.Field
	var rowValues []Scmer
	schemaInitialized := false
	fieldsPublished := false
	var bufferedRows []Scmer
	bufferedRowCount := 0
	var rowStatus driver.RowStatus
	var resultlock sync.Mutex
	emitPreparedRow := func(values []Scmer) {
		row, err := output.BeginRow()
		if err != nil {
			panic(err)
		}
		defer row.Abort()
		for _, val := range values {
			appendScmerToMySQLRow(row, val)
		}
		status, err := row.End()
		rowStatus |= status
		if err != nil {
			panic(err)
		}
	}
	publishAndFlushRows := func() {
		if fieldsPublished {
			return
		}
		if err := output.SetFields(fields); err != nil {
			panic(err)
		}
		fieldsPublished = true
		width := len(fields)
		for offset := 0; offset < len(bufferedRows); offset += width {
			emitPreparedRow(bufferedRows[offset : offset+width])
		}
		bufferedRows = nil
		bufferedRowCount = 0
	}
	// load scm session object
	scmSessionAny, _ := mysqlsessions.Load(session.ID())
	// result from scheme
	sessionFunc := scmSessionAny.(func(...Scmer) Scmer)
	scmSessionScmer := NewFunc(sessionFunc)
	sessionFunc(NewString("username"), NewString(session.User()))
	sessionFunc(NewString("schema"), NewString(session.Schema()))
	// Populate bind variables (v1, v2, ...) from prepared-statement params into session
	for name, bv := range bindVariables {
		if bv == nil {
			continue
		}
		val, err := sqltypes.BindVariableToValue(bv)
		if err != nil {
			continue
		}
		sessionFunc(NewString(name), MySQLToScmer(val))
	}
	rowcount := func() Scmer {
		defer func() {
			if r := recover(); r != nil {
				if sqlErr, ok := r.(*sqldb.SQLError); ok {
					PrintError("error in mysql connection: " + sqlErr.Error())
					myerr = sqlErr
				} else {
					errMsg := fmt.Sprint(r)
					PrintError("error in mysql connection: " + errMsg)
					myerr = ErrorWrapper(mysqlClientErrorMessage(errMsg))
				}
			}
		}()
		callbackFn := NewFunc(func(a ...Scmer) Scmer {
			// function resultrow(item)
			item := a[0].Slice()
			resultlock.Lock()
			defer resultlock.Unlock()

			var unknownColumn bool
			rowValues, unknownColumn = prepareMySQLResultRow(&fields, colmap, item, rowValues, schemaInitialized, !fieldsPublished)
			if unknownColumn {
				rowStatus |= driver.RowTruncated
			}
			schemaInitialized = true
			if !fieldsPublished && !mysqlFieldsResolved(fields) && bufferedRowCount+1 < mysqlFieldDiscoveryRows {
				bufferedRows = append(bufferedRows, rowValues...)
				bufferedRowCount++
				return NewBool(true)
			}
			if !fieldsPublished {
				if bufferedRowCount == 0 {
					if err := output.SetFields(fields); err != nil {
						panic(err)
					}
					fieldsPublished = true
					emitPreparedRow(rowValues)
				} else {
					bufferedRows = append(bufferedRows, rowValues...)
					bufferedRowCount++
					publishAndFlushRows()
				}
			} else {
				emitPreparedRow(rowValues)
			}
			return NewBool(true)
		})
		// Execute query within GLS context so storage layer can access
		// the session (and its TxContext) via GetCurrentTx().
		var rc Scmer
		SetValues(map[string]any{
			"session":         scmSessionScmer,
			"sessionStatePtr": ss,
			"querySeq":        querySeq,
			"context":         queryCtx,
		}, func() {
			rc = Apply(m.querycallback, NewString(session.Schema()), NewString(query), callbackFn, scmSessionScmer,
				NewAny(&QueryExecutionContext{SessionState: ss, QuerySeq: querySeq}))
		})
		return rc
	}()
	if myerr != nil {
		return myerr
	}
	if schemaInitialized && !fieldsPublished {
		publishAndFlushRows()
	}
	if rowStatus != driver.RowComplete {
		m.log.Warning("mysql result rows required recovery flags=%d", rowStatus)
	}
	// Retrieve last_insert_id from the session (set by INSERT with AUTO_INCREMENT).
	// TODO: replace with a dedicated callback parameter to m.querycallback so the
	// Scheme side has full control over returned insert IDs without hardcoded fields.
	lastInsertId := sessionFunc(NewString("last_insert_id"))
	lastInsertID := uint64(0)
	if !lastInsertId.IsNil() {
		lastInsertID = uint64(lastInsertId.Int())
	}
	rowsAffected := uint64(rowcount.Int())
	// Transaction and schema status can change once per statement, not once per
	// emitted result row. Publish the final state before flushing the result.
	// A SELECT cannot change transaction or default-schema protocol flags.
	// Avoid three synchronized Scheme-session lookups on this hot path.
	if !selectQuery {
		updateFlags(session, sessionFunc)
	}
	if !fieldsPublished && selectQuery {
		fields = mysqlFieldsForEmptyTableSelect(query, session.Schema())
		if len(fields) == 0 {
			fields = []*querypb.Field{
				{Name: "_empty", Type: querypb.Type_NULL_TYPE},
			}
		}
		if err := output.SetFields(fields); err != nil {
			return err
		}
	}
	return output.Finish(rowsAffected, lastInsertID, 0)
}

func updateFlags(s *driver.Session, sessionFunc func(...Scmer) Scmer) {
	tx := sessionFunc(NewString("transaction"))
	if tx.IsNil() {
		s.SetTransaction(false)
	} else {
		s.SetTransaction(true)
	}
	// Update schema if changed via USE statement
	schema := sessionFunc(NewString("schema"))
	if !schema.IsNil() && schema.String() != "" {
		s.SetSchema(schema.String())
	}
}
