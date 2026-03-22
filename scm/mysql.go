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

/* session storage -> map from session id to SCM session object */
var mysqlsessions sync.Map

// mysqlStates maps driver session ID -> *SessionState for SHOW PROCESSLIST
var mysqlStates sync.Map

func (m *MySQLWrapper) ServerVersion() string {
	return "MemCP"
}
func (m *MySQLWrapper) SetServerVersion() {
}
func (m *MySQLWrapper) NewSession(session *driver.Session) {
	m.log.Info("%s", "New Session from "+session.Addr())
	mysqlsessions.Store(session.ID(), NewSession().Func())
	ss := RegisterSession(session.User(), session.Addr(), session.Schema())
	mysqlStates.Store(session.ID(), ss)
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
	password := Apply(m.authcallback, NewString(session.User()))
	if password.IsNil() {
		// user does not exist
		return sqldb.NewSQLError(sqldb.ER_ACCESS_DENIED_ERROR, session.User(), session.Addr(), "YES")
	}
	if !session.TestPassword([]byte(password.String())) {
		return sqldb.NewSQLError(sqldb.ER_ACCESS_DENIED_ERROR, session.User(), session.Addr(), "YES")
	}
	return nil
}
func (m *MySQLWrapper) ComInitDB(session *driver.Session, database string) error {
	m.log.Info("%s", "db "+database)
	allowed := Apply(m.schemacallback, NewString(session.User()), NewString(database))
	if !allowed.Bool() {
		return sqldb.NewSQLErrorf(sqldb.ER_ACCESS_DENIED_ERROR, "access denied for database %s", database)
	}
	session.SetSchema(database)
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

func ScmerToMySQL(v Scmer) sqltypes.Value {
	switch v.GetTag() {
	case tagNil:
		return sqltypes.MakeTrusted(querypb.Type_NULL_TYPE, nil)
	case tagFloat:
		return sqltypes.NewFloat64(v.Float())
	case tagInt:
		return sqltypes.NewInt64(v.Int())
	case tagBool:
		if v.Bool() {
			return sqltypes.NewInt32(1)
		}
		return sqltypes.NewInt32(0)
	case tagDate:
		return sqltypes.NewVarChar(v.String())
	case tagString:
		return sqltypes.NewVarChar(v.String())
	default:
		return sqltypes.NewVarChar(v.String())
	}
}

type ErrorWrapper string

func (s ErrorWrapper) Error() string {
	return string(s)
}

func updateMySQLFieldMetadata(field *querypb.Field, val sqltypes.Value) {
	field.Type = val.Type()
	switch val.Type() {
	case querypb.Type_TEXT, querypb.Type_VARCHAR, querypb.Type_CHAR, querypb.Type_BLOB:
		field.Charset = 45 // utf8mb4_general_ci
	default:
		field.Charset = 0
	}
}

func appendMySQLResultRow(result *sqltypes.Result, colmap map[string]int, item []Scmer) []sqltypes.Value {
	newitem := make([]sqltypes.Value, len(result.Fields))
	for i := 0; i < len(item)-1; i += 2 {
		val := ScmerToMySQL(item[i+1])

		colname := item[i].String()
		colid, ok := colmap[colname]
		if ok {
			duplicateAliasInRow := colid < len(newitem) && !newitem[colid].IsNull()
			newitem[colid] = val
			if duplicateAliasInRow || result.Fields[colid].Type == querypb.Type_NULL_TYPE {
				updateMySQLFieldMetadata(result.Fields[colid], val)
			}
		} else {
			colmap[colname] = len(result.Fields)
			newcol := new(querypb.Field)
			newcol.Name = colname
			updateMySQLFieldMetadata(newcol, val)
			result.Fields = append(result.Fields, newcol)
			newitem = append(newitem, val)
		}
	}
	return newitem
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
	return strings.HasPrefix(strings.ToLower(trimmed), "select")
}
func (m *MySQLWrapper) ComQuery(session *driver.Session, query string, bindVariables map[string]*querypb.BindVariable, callback func(*sqltypes.Result) error) (myerr error) {
	atomic.AddInt64(&TotalHTTPRequests, 1)
	var ss *SessionState
	if v, ok := mysqlStates.Load(session.ID()); ok {
		ss = v.(*SessionState)
		ss.SetCommand("Query", query)
		ss.SetDB(session.Schema())
	}
	defer func() {
		if ss != nil {
			ss.SetCommand("Sleep", "")
			ss.SetDB(session.Schema())
		}
	}()
	// max_allowed_packet: PHP PDO queries this to size buffers.
	// Return 32MB (33554432) so large result sets work.
	if query == "select @@max_allowed_packet" || query == "SELECT @@max_allowed_packet" {
		callback(&sqltypes.Result{
			Fields: []*querypb.Field{
				{Name: "@@max_allowed_packet", Type: querypb.Type_INT64},
			},
			Rows: [][]sqltypes.Value{
				{sqltypes.MakeTrusted(querypb.Type_INT64, []byte("33554432"))},
			},
		})
		return nil
	}
	if query == "select @@version_comment limit 1" {
		callback(&sqltypes.Result{
			Fields: []*querypb.Field{
				{Name: "@@version_comment", Type: querypb.Type_TEXT, Charset: 45},
			},
			Rows: [][]sqltypes.Value{
				{sqltypes.MakeTrusted(querypb.Type_TEXT, []byte(runtime.GOOS))},
			},
		})
		return nil
	}
	colmap := make(map[string]int)
	// TODO: sqltypes.RStateNone for INSERTs
	var result sqltypes.Result
	var resultlock sync.Mutex
	result.State = sqltypes.RStateFields
	result.Rows = make([][]sqltypes.Value, 0, 1024)
	// load scm session object
	scmSessionAny, _ := mysqlsessions.Load(session.ID())
	// result from scheme
	sessionFunc := scmSessionAny.(func(...Scmer) Scmer)
	scmSessionScmer := NewFunc(sessionFunc)
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
					myerr = ErrorWrapper(errMsg)
				}
			}
		}()
		callbackFn := NewFunc(func(a ...Scmer) Scmer {
			// function resultrow(item)
			item := a[0].Slice()
			resultlock.Lock()
			defer resultlock.Unlock()
			updateFlags(session, sessionFunc) // set transaction status

			newitem := appendMySQLResultRow(&result, colmap, item)
			if len(result.Rows) == cap(result.Rows) {
				// flush
				callback(&result)
				if result.State == sqltypes.RStateFields {
					result.State = sqltypes.RStateRows
					callback(&result)
				}
				result.Rows = result.Rows[0:0] // slice off rest of buffer to restart
			}
			result.Rows = append(result.Rows, newitem)
			return NewBool(true)
		})
		// Execute query within GLS context so storage layer can access
		// the session (and its TxContext) via GetCurrentTx().
		var rc Scmer
		SetValues(map[string]any{
			"session":         scmSessionScmer,
			"sessionStatePtr": ss,
		}, func() {
			rc = Apply(m.querycallback, NewString(session.Schema()), NewString(query), callbackFn, scmSessionScmer)
		})
		return rc
	}()
	if myerr != nil {
		return myerr
	}
	// Retrieve last_insert_id from the session (set by INSERT with AUTO_INCREMENT).
	// TODO: replace with a dedicated callback parameter to m.querycallback so the
	// Scheme side has full control over returned insert IDs without hardcoded fields.
	lastInsertId := sessionFunc(NewString("last_insert_id"))
	if !lastInsertId.IsNil() {
		result.InsertID = uint64(lastInsertId.Int())
	}
	result.RowsAffected = uint64(rowcount.Int())
	// update status greeting
	updateFlags(session, sessionFunc)
	// flush the rest
	if result.State == sqltypes.RStateFields {
		if len(result.Fields) == 0 && isSelectQuery(query) {
			result.Fields = []*querypb.Field{
				{Name: "_empty", Type: querypb.Type_NULL_TYPE},
			}
		}
		result.State = sqltypes.RStateNone // full send
		callback(&result)
	} else {
		// rest + finish
		callback(&result)
		result.State = sqltypes.RStateFinished
		callback(&result)
	}
	return myerr
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
