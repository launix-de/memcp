package scm

import "fmt"
import "runtime"
import "errors"
import "github.com/launix-de/go-mysqlstack/driver"
import "github.com/launix-de/go-mysqlstack/xlog"
import "github.com/launix-de/go-mysqlstack/sqlparser/depends/sqltypes"
import querypb "github.com/launix-de/go-mysqlstack/sqlparser/depends/query"

// build this function into your SCM environment to offer http server capabilities
func MySQLServe(a ...Scmer) Scmer {
	// params: port, authcallback, schemacallback, querycallback
	port := String(a[0])

	fmt.Println("Hallo World")
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
	return "ok"
}

// TODO: driver.CreatePassword helper function

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
	// TODO: load password stored from database
	// m.authcallback(user) -> password or nil
	stored := driver.CreatePassword("admin")
	if !session.TestPassword(stored) {
		return errors.New("Auth failed")
	}
	return nil
}
func (m *MySQLWrapper) ComInitDB(session *driver.Session, database string) error {
	m.log.Info("db "+database)
	session.SetSchema(database)
	// TODO: check access rights, reject if necessary
	// m.schemacallback
	return nil
}
func (m *MySQLWrapper) ComQuery(session *driver.Session, query string, bindVariables map[string]*querypb.BindVariable, callback func(*sqltypes.Result) error) error {
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
	// result from scheme
	result := Apply(m.querycallback, []Scmer{query,}).(string)
	callback(&sqltypes.Result {
		Fields: []*querypb.Field {
			{ Name: "ok", Type: querypb.Type_BIT },
		},
		Rows: [][]sqltypes.Value {
			{ sqltypes.MakeTrusted(querypb.Type_BIT, []byte(result)) },
		},
	})
	return nil
}

