/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
the terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/

package storage

import "io"
import "os"
import "fmt"
import "net"
import "time"
import "bytes"
import "os/exec"
import "strings"
import "testing"
import "net/http"
import "database/sql"
import "encoding/json"
import "path/filepath"
import _ "github.com/go-sql-driver/mysql"

type replicaProcess struct {
	t                                               *testing.T
	binary, root, data, port, replication, password string
	mysqlPort                                       string
	cmd                                             *exec.Cmd
	log                                             *os.File
}

func replicaTestPort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := fmt.Sprint(l.Addr().(*net.TCPAddr).Port)
	l.Close()
	return port
}

func (p *replicaProcess) start() {
	p.t.Helper()
	var err error
	p.log, err = os.OpenFile(filepath.Join(p.data, "process.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		p.t.Fatal(err)
	}
	p.cmd = exec.Command(p.binary, "-data", p.data, "--api-port="+p.port, "--disable-mysql", "--mysql-socket=", "--no-repl", "--root-password="+p.password, "--replication="+p.replication, "lib/main.scm")
	if p.mysqlPort != "" {
		args := make([]string, 0, len(p.cmd.Args))
		for _, a := range p.cmd.Args {
			if a != "--disable-mysql" {
				args = append(args, a)
			}
		}
		p.cmd.Args = append(args, "--mysql-port="+p.mysqlPort)
	}
	p.cmd.Dir, p.cmd.Stdout, p.cmd.Stderr = p.root, p.log, p.log
	if err := p.cmd.Start(); err != nil {
		p.t.Fatal(err)
	}
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+p.port, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	p.t.Fatalf("server did not start: %s", p.logs())
}

func (p *replicaProcess) kill() {
	p.t.Helper()
	if p.cmd != nil {
		p.cmd.Process.Kill()
		p.cmd.Wait()
		p.cmd = nil
		p.log.Close()
	}
}

func (p *replicaProcess) waitOffline() {
	p.t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+p.port, 100*time.Millisecond)
		if err != nil {
			return
		}
		conn.Close()
		time.Sleep(50 * time.Millisecond)
	}
	p.t.Fatalf("slave did not stop: %s", p.logs())
}

func (p *replicaProcess) logs() string {
	data, _ := os.ReadFile(filepath.Join(p.data, "process.log"))
	if len(data) > 12000 {
		data = data[len(data)-12000:]
	}
	return string(data)
}

func (p *replicaProcess) query(database, sql, password string) (int, string) {
	return p.queryAs("root", database, sql, password)
}

func (p *replicaProcess) queryAs(user, database, sql, password string) (int, string) {
	req, _ := http.NewRequest("POST", "http://127.0.0.1:"+p.port+"/sql/"+database, strings.NewReader(sql))
	req.SetBasicAuth(user, password)
	client := &http.Client{Timeout: 20 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, err.Error()
	}
	return res.StatusCode, string(data)
}

func (p *replicaProcess) admin(command string) string {
	p.t.Helper()
	req, _ := http.NewRequest("POST", "http://127.0.0.1:"+p.port+"/scm", strings.NewReader(command))
	req.SetBasicAuth("root", "replication-test-password")
	res, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		p.t.Fatal(err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		p.t.Fatalf("admin command: %d %s", res.StatusCode, data)
	}
	return string(data)
}

func (p *replicaProcess) mustQuery(sql string) string {
	p.t.Helper()
	status, body := p.query("replica_test", sql, "replication-test-password")
	if status != 200 {
		p.t.Fatalf("SQL %s: status %d: %s\n%s", sql, status, body, p.logs())
	}
	return body
}

func replicaNumber(body, column string) int {
	var row map[string]json.RawMessage
	if json.NewDecoder(strings.NewReader(body)).Decode(&row) != nil {
		return -1
	}
	var value int
	if json.Unmarshal(row[column], &value) != nil {
		return -1
	}
	return value
}

// Processes have different data directories and different initial passwords.
// Kill is SIGKILL, not graceful SHUTDOWN, so this exercises WAL recovery as
// well as authentication caching and reconnection. The directories survive.
func TestReplicationAlternatingProcessFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("process crash integration test")
	}
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	binary := os.Getenv("MEMCP_REPLICATION_BINARY")
	if binary == "" {
		binary = filepath.Join(t.TempDir(), "memcp")
		cmd := exec.Command("go", "build", "-tags=nophp", "-o", binary, ".")
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build: %v: %s", err, out)
		}
	}
	repPort := replicaTestPort(t)
	master := &replicaProcess{t: t, binary: binary, root: root, data: t.TempDir(), port: replicaTestPort(t), replication: repPort, password: "replication-test-password"}
	slave := &replicaProcess{t: t, binary: binary, root: root, data: t.TempDir(), port: replicaTestPort(t), replication: "127.0.0.1:" + repPort, password: "different-local-password"}
	slave.mysqlPort = replicaTestPort(t)
	t.Cleanup(func() {
		slave.kill()
		master.kill()
		if t.Failed() {
			t.Logf("MASTER\n%s\nSLAVE\n%s", master.logs(), slave.logs())
		}
	})
	master.start()
	if status, body := master.query("system", "CREATE DATABASE replica_test", "replication-test-password"); status != 200 {
		t.Fatalf("create database: %d %s", status, body)
	}
	master.mustQuery("CREATE TABLE items(id INT PRIMARY KEY, value INT) ENGINE=SAFE")
	master.mustQuery("INSERT INTO items VALUES(1, 10)")
	for _, sql := range []string{"CREATE DATABASE private_test", "CREATE USER replica_user IDENTIFIED BY 'user-password'", "GRANT ALL ON replica_test.* TO replica_user"} {
		if status, body := master.query("system", sql, "replication-test-password"); status != 200 {
			t.Fatalf("auth setup: %d %s", status, body)
		}
	}
	slave.start()
	mysql, err := sql.Open("mysql", "root:replication-test-password@tcp(127.0.0.1:"+slave.mysqlPort+")/replica_test?timeout=5s&readTimeout=5s&writeTimeout=5s")
	if err != nil {
		t.Fatal(err)
	}
	defer mysql.Close()
	deadline := time.Now().Add(10 * time.Second)
	for {
		var count int
		err = mysql.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
		if err == nil && count == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("first MySQL login/transfer: %v; %s", err, slave.logs())
		}
		time.Sleep(50 * time.Millisecond)
	}
	for {
		status, body := slave.query("replica_test", "SELECT COUNT(*) AS n FROM items", "replication-test-password")
		if status == 200 && replicaNumber(body, "n") == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("first transfer: %d %s", status, body)
		}
		time.Sleep(50 * time.Millisecond)
	}
	var settings struct{ ReplicationUUID string }
	data, _ := os.ReadFile(filepath.Join(slave.data, "settings.json"))
	if json.Unmarshal(data, &settings) != nil || settings.ReplicationUUID == "" {
		t.Fatal("slave identity not persisted")
	}
	identity := settings.ReplicationUUID
	if status, body := slave.queryAs("replica_user", "replica_test", "SELECT COUNT(*) AS n FROM items", "user-password"); status != 200 || replicaNumber(body, "n") != 1 {
		t.Fatalf("granted user rejected: %d %s", status, body)
	}
	if status, _ := slave.queryAs("replica_user", "private_test", "SELECT 1", "user-password"); status == 200 {
		t.Fatal("slave read an unauthorized database")
	}
	for round := 0; round < 3; round++ {
		master.kill()
		if round == 0 {
			slave.mustQuery("CREATE TABLE offline_created(id INT PRIMARY KEY, payload TEXT) ENGINE=SAFE")
			slave.mustQuery("INSERT INTO offline_created VALUES(1,'survived-offline-ddl')")
		}
		for i := 0; i < 5; i++ {
			id := round*5 + i + 2
			slave.mustQuery(fmt.Sprintf("INSERT INTO items VALUES(%d,%d)", id, id*10))
		}
		slave.mustQuery("UPDATE items SET value=value+1 WHERE id=1")
		master.start()
		deadline := time.Now().Add(10 * time.Second)
		for {
			// Wait for the reverse RPC connection to be re-established. A
			// matching backup schema is proof that an online checkpoint ran.
			slave.mustQuery("SELECT COUNT(*) AS n FROM items")
			masterSchema, _ := os.ReadFile(filepath.Join(master.data, "replica_test", "schema.json"))
			slaveSchema, _ := os.ReadFile(filepath.Join(slave.data, "replica_test", "schema.json"))
			if bytes.Equal(masterSchema, slaveSchema) {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("master backup did not catch up")
			}
			time.Sleep(100 * time.Millisecond)
		}
		slave.kill()
		status, body := master.query("replica_test", "SELECT * FROM items", "replication-test-password")
		if status != 500 || !strings.Contains(body, replicationNetworkError) {
			t.Fatalf("offline owner must fail without takeover: %d %s", status, body)
		}
		slave.start()
		expected := 1 + (round+1)*5
		if body := slave.mustQuery("SELECT COUNT(*) AS n FROM items"); replicaNumber(body, "n") != expected {
			t.Fatalf("lost committed rows: %s", body)
		}
		if body := slave.mustQuery("SELECT value FROM items WHERE id=1"); replicaNumber(body, "value") != 11+round {
			t.Fatalf("lost update: %s", body)
		}
		if body := slave.mustQuery("SELECT payload FROM offline_created WHERE id=1"); !strings.Contains(body, "survived-offline-ddl") {
			t.Fatalf("lost offline schema/data: %s", body)
		}
		data, _ = os.ReadFile(filepath.Join(slave.data, "settings.json"))
		json.Unmarshal(data, &settings)
		if settings.ReplicationUUID != identity {
			t.Fatal("reconnect changed identity")
		}
	}
	// An explicit transaction prevents transfer until it commits. The network
	// request times out first; a late offer must never let the modifier's next
	// request overwrite its committed data with the master's older checkpoint.
	late, err := sql.Open("mysql", "root:replication-test-password@tcp(127.0.0.1:"+slave.mysqlPort+")/replica_test?timeout=5s&readTimeout=15s&writeTimeout=5s")
	if err != nil {
		t.Fatal(err)
	}
	defer late.Close()
	tx, err := late.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec("INSERT INTO items VALUES(1000,999)"); err != nil {
		t.Fatal(err)
	}
	status, body := master.query("replica_test", "SELECT COUNT(*) FROM items", "replication-test-password")
	if status != 500 || !strings.Contains(body, replicationNetworkError) {
		t.Fatalf("held transaction transfer: %d %s", status, body)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	if body := slave.mustQuery("SELECT value FROM items WHERE id=1000"); replicaNumber(body, "value") != 999 {
		t.Fatalf("late transfer lost committed data: %s", body)
	}
	slave.mustQuery("DELETE FROM items WHERE id=1000")
	if status, _ := slave.query("replica_test", "SELECT 1", "wrong-password"); status != 401 {
		t.Fatalf("wrong password accepted: %d", status)
	}
	var statusesJSON string
	if err := json.Unmarshal([]byte(master.admin("(replication_slaves)")), &statusesJSON); err != nil {
		t.Fatal(err)
	}
	var statuses []replicaStatus
	if err := json.Unmarshal([]byte(statusesJSON), &statuses); err != nil || len(statuses) != 1 || statuses[0].ID != identity || !statuses[0].Connected {
		t.Fatalf("unexpected slave status: %s", statusesJSON)
	}
	slave.mustQuery("SHUTDOWN")
	slave.waitOffline()
	if status, body := master.query("system", "SELECT 1", "replication-test-password"); status != 200 {
		t.Fatalf("slave shutdown affected master: %d %s", status, body)
	}
	slave.kill()
	slave.start()
	slave.mustQuery("SELECT COUNT(*) AS n FROM items")
	master.admin(fmt.Sprintf("(replication_retire %q)", identity))
	slave.waitOffline()
	if body := master.mustQuery("SELECT COUNT(*) AS n FROM items"); replicaNumber(body, "n") != 16 {
		t.Fatalf("retired backup lost rows: %s", body)
	}
	if body := master.mustQuery("SELECT payload FROM offline_created WHERE id=1"); !strings.Contains(body, "survived-offline-ddl") {
		t.Fatalf("backup lost offline schema/data: %s", body)
	}
	master.kill()
	master.start()
	if body := master.mustQuery("SELECT COUNT(*) AS n FROM items"); replicaNumber(body, "n") != 16 {
		t.Fatalf("restart lost backup: %s", body)
	}
	slave.kill()
	slave.start()
	if status, _ := slave.query("replica_test", "SELECT * FROM items", "replication-test-password"); status == 200 {
		t.Fatal("retired slave reactivated stale ownership")
	}
}
