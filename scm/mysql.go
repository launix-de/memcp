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

package scm

import "fmt"
import "sync"
import "errors"
import "runtime"
import "github.com/launix-de/go-mysqlstack/driver"
import "github.com/launix-de/go-mysqlstack/xlog"
import "github.com/launix-de/go-mysqlstack/sqlparser/depends/sqltypes"
import querypb "github.com/launix-de/go-mysqlstack/sqlparser/depends/query"

// build this function into your SCM environment to offer http server capabilities
func MySQLServe(a ...Scmer) Scmer {
	// params: port, authcallback, schemacallback, querycallback
	port := String(a[0])

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
	go func () {
		defer mysql.Close()
		mysql.Accept()
	}()
	return true
}

// driver.CreatePassword helper function
func MySQLPassword(a ...Scmer) Scmer {
	return string(driver.CreatePassword(String(a[0])))
}

type MySQLWrapper struct {
	log *xlog.Log
	authcallback Scmer
	schemacallback Scmer
	querycallback Scmer
}

/* session storage -> map from session id to SCM session object */
var mysqlsessions sync.Map

func (m *MySQLWrapper) ServerVersion() string {
	return "MemCP"
}
func (m *MySQLWrapper) SetServerVersion() {
}
func (m *MySQLWrapper) NewSession(session *driver.Session) {
	m.log.Info("New Session from " + session.Addr())
	// initialize something??
	mysqlsessions.Store(session.ID(), NewSession())
}
func (m *MySQLWrapper) SessionInc(session *driver.Session) {
	// I think we can skip session counting
}
func (m *MySQLWrapper) SessionDec(session *driver.Session) {
	// I think we can skip session counting
}
func (m *MySQLWrapper) SessionClosed(session *driver.Session) {
	m.log.Info("Closed Session " + session.User() + " from " + session.Addr())
	mysqlsessions.Delete(session.ID())
}
func (m *MySQLWrapper) SessionCheck(session *driver.Session) error {
	// we could reject clients here when server load is too full
	return nil
}

func (m *MySQLWrapper) AuthCheck(session *driver.Session) error {
	m.log.Info("Auth Check with " + session.User())
	// callback should load password from database
	password := Apply(m.authcallback, session.User())
	if password == nil {
		// user does not exist
		return errors.New("Auth failed")
	}
	if !session.TestPassword([]byte(String(password))) {
		return errors.New("Auth failed")
	}
	return nil
}
func (m *MySQLWrapper) ComInitDB(session *driver.Session, database string) error {
	m.log.Info("db "+database)
	allowed := Apply(m.schemacallback, session.User(), database)
	if (!ToBool(allowed)) {
		return errors.New("access denied for database " + database)
	}
	session.SetSchema(database)
	return nil
}
func ScmerToMySQL(v Scmer) sqltypes.Value {
	switch v2 := v.(type) {
		case nil:
			return sqltypes.MakeTrusted(querypb.Type_NULL_TYPE, nil)
		case float64:
			return sqltypes.NewFloat64(v2)
		case int64:
			return sqltypes.NewInt64(v2)
		case bool:
			if v2 {
				return sqltypes.NewInt32(1)
			} else {
				return sqltypes.NewInt32(0)
			}
		case string:
			return sqltypes.NewVarChar(v2) // TODO: also consider NewVarBinary
		default:
			return sqltypes.NewVarChar(String(v2))
	}
}
type ErrorWrapper string
func (s ErrorWrapper) Error() string {
	return string(s)
}
func (m *MySQLWrapper) ComQuery(session *driver.Session, query string, bindVariables map[string]*querypb.BindVariable, callback func(*sqltypes.Result) error) (myerr error) {
	if query == "select @@version_comment limit 1" {
		callback(&sqltypes.Result {
			Fields: []*querypb.Field {
				{ Name: "@@version_comment", Type: querypb.Type_TEXT },
			},
			Rows: [][]sqltypes.Value {
				{ sqltypes.MakeTrusted(querypb.Type_TEXT, []byte(runtime.GOOS)) },
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
	scmSession, _ := mysqlsessions.Load(session.ID())
	// result from scheme
	rowcount := func () Scmer {
		defer func () {
			if r := recover(); r != nil {
				PrintError("error in mysql connection: " + fmt.Sprint(r))
				myerr = ErrorWrapper(fmt.Sprint(r))
			}
		}()
		return Apply(m.querycallback, session.Schema(), query, func (a... Scmer) Scmer {
			// function resultrow(item)
			item := a[0].([]Scmer)
			resultlock.Lock()
			defer resultlock.Unlock()
			updateFlags(session, scmSession) // set transaction status

			newitem := make([]sqltypes.Value, len(result.Fields))
			for i := 0; i < len(item)-1; i += 2 {
				val := ScmerToMySQL(item[i+1])

				colname := String(item[i])
				colid, ok := colmap[colname]
				if ok {
					newitem[colid] = val
					if result.Fields[colid].Type == querypb.Type_NULL_TYPE {
						result.Fields[colid].Type = val.Type()
					}
				} else {
					// add row to result
					colmap[colname] = len(result.Fields)
					newcol := new(querypb.Field)
					newcol.Name = colname
					newcol.Type = val.Type()
					result.Fields = append(result.Fields, newcol)
					newitem = append(newitem, val)
				}
			}
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
			return true
		},
		scmSession)
	}()
	if myerr != nil {
		return myerr
	}
	// TODO: also set result.InsertID (maybe as a callback as 4th parameter to m.querycallback?)
	switch rowcount_ := rowcount.(type) {
		case float64:
			result.RowsAffected = uint64(rowcount_)
		case int64:
			result.RowsAffected = uint64(rowcount_)
	}
	// update status greeting
	updateFlags(session, scmSession)
	// flush the rest
	if result.State == sqltypes.RStateFields {
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

func updateFlags(s *driver.Session, session_ any) {
	session := session_.(func(...Scmer) Scmer)
	tx := session("transaction")
	if tx == nil {
		s.SetTransaction(false)
	} else {
		s.SetTransaction(true)
	}
}

