package scm

import "fmt"
import "sync"
import "errors"
import "runtime"
import "runtime/debug"
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

func (m *MySQLWrapper) ServerVersion() string {
	return "MemCP"
}
func (m *MySQLWrapper) SetServerVersion() {
}
func (m *MySQLWrapper) NewSession(session *driver.Session) {
	m.log.Info("New Session from " + session.Addr())
	// initialize something??
}
func (m *MySQLWrapper) SessionInc(session *driver.Session) {
	// I think we can skip session counting
}
func (m *MySQLWrapper) SessionDec(session *driver.Session) {
	// I think we can skip session counting
}
func (m *MySQLWrapper) SessionClosed(session *driver.Session) {
	m.log.Info("Closed Session " + session.User() + " from " + session.Addr())
}
func (m *MySQLWrapper) SessionCheck(session *driver.Session) error {
	// we could reject clients here when server load is too full
	return nil
}

func (m *MySQLWrapper) AuthCheck(session *driver.Session) error {
	m.log.Info("Auth Check with " + session.User())
	// callback should load password from database
	password := Apply(m.authcallback, []Scmer{session.User(),})
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
	allowed := Apply(m.schemacallback, []Scmer{session.User(), database})
	if (!ToBool(allowed)) {
		return errors.New("access denied for database " + database)
	}
	session.SetSchema(database)
	return nil
}
func (m *MySQLWrapper) ComQuery(session *driver.Session, query string, bindVariables map[string]*querypb.BindVariable, callback func(*sqltypes.Result) error) error {
	var myerr error = nil
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
	// result from scheme
	func () {
		defer func () {
			if r := recover(); r != nil {
				myerr = fmt.Errorf("%v", r) // transmit r for error
				debug.PrintStack()
			}
		}()
		Apply(m.querycallback, []Scmer{session.Schema(), query, func (a... Scmer) Scmer {
			// function resultrow(item)
			item := a[0].([]Scmer)
			newitem := make([]sqltypes.Value, len(result.Fields))
			for i := 0; i < len(item); i += 2 {
				colname := String(item[i])
				colid, ok := colmap[colname]
				if ok {
					newitem[colid] = sqltypes.MakeTrusted(querypb.Type_TEXT, []byte(String(item[i+1])))
				} else {
					// add row to result
					colmap[colname] = len(result.Fields)
					newcol := new(querypb.Field)
					newcol.Name = colname
					newcol.Type = querypb.Type_TEXT
					result.Fields = append(result.Fields, newcol)
					newitem = append(newitem, sqltypes.MakeTrusted(querypb.Type_TEXT, []byte(String(item[i+1]))))
				}
			}
			resultlock.Lock()
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
			resultlock.Unlock()
			return "ok"
		},})
	}()
	if myerr != nil {
		return myerr
	}
	// flush the rest
	if result.State == sqltypes.RStateFields {
		result.State = sqltypes.RStateNone // full send
		// TODO: result.InsertID, result.RowsAffected,
		callback(&result)
	} else {
		// rest + finish
		callback(&result)
		result.State = sqltypes.RStateFinished
		callback(&result)
	}
	return myerr
}

