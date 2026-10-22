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
import "sync"
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
	// go test -exec wraps only the test binary, not processes it launches.
	// Cross tests must execute the target server explicitly through QEMU too.
	if executor := os.Getenv("MEMCP_REPLICATION_EXECUTOR"); executor != "" {
		p.cmd = exec.Command(executor, p.cmd.Args...)
	}
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
	startupTimeout := 45 * time.Second
	if configured := os.Getenv("MEMCP_REPLICATION_STARTUP_TIMEOUT"); configured != "" {
		startupTimeout, err = time.ParseDuration(configured)
		if err != nil || startupTimeout <= 0 {
			p.t.Fatalf("invalid replication startup timeout %q", configured)
		}
	}
	deadline := time.Now().Add(startupTimeout)
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
	for _, command := range []string{"CREATE DATABASE slave_admin_test", "CREATE USER slave_created_user IDENTIFIED BY 'slave-created-password'", "GRANT ALL ON replica_test.* TO slave_created_user"} {
		if status, body := slave.query("system", command, "replication-test-password"); status != 200 {
			t.Fatalf("forwarded administration: %d %s", status, body)
		}
	}
	if status, body := master.query("system", "SELECT COUNT(*) AS n FROM user WHERE username='slave_created_user'", master.password); status != 200 || replicaNumber(body, "n") != 1 {
		t.Fatalf("slave admin did not update master access: %d %s", status, body)
	}
	if _, err := os.Stat(filepath.Join(master.data, "slave_admin_test", "schema.json")); err != nil {
		t.Fatalf("slave CREATE DATABASE did not reach master: %v", err)
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
				t.Fatalf("master backup did not catch up\nMASTER: %s\nSLAVE: %s", masterSchema, slaveSchema)
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
	// DDL on the other node must first obtain current data generations, not
	// replace a modifier's deltas with the master's older safety backup.
	master.mustQuery("ALTER TABLE items ADD COLUMN extra INT DEFAULT 7")
	if body := slave.mustQuery("SELECT COUNT(*) AS n FROM items WHERE extra=7"); replicaNumber(body, "n") != 16 {
		t.Fatalf("remote DDL lost data or default values: %s", body)
	}
	slave.mustQuery("UPDATE items SET value=value WHERE id=1")
	slave.mustQuery("ALTER TABLE items DROP COLUMN extra")
	master.mustQuery("RENAME TABLE items TO renamed_items")
	if body := slave.mustQuery("SELECT COUNT(*) AS n FROM renamed_items"); replicaNumber(body, "n") != 16 {
		t.Fatalf("remote rename lost data: %s", body)
	}
	slave.mustQuery("RENAME TABLE renamed_items TO items")
	slave.mustQuery("UPDATE items SET value=value WHERE id=1")
	// A master read shares the slave's M generation: master S, slave O. Reads
	// on either process must preserve those flags instead of acquiring M.
	record := func(p *replicaProcess) replicaRecord {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(p.data, ".replication-state"))
		if err != nil {
			t.Fatal(err)
		}
		var journal replicaJournal
		if err := json.Unmarshal(data, &journal); err != nil {
			t.Fatal(err)
		}
		for _, record := range journal.Records {
			if record.Resource.Table == "items" {
				return record
			}
		}
		t.Fatal("items has no independent MOSI record")
		return replicaRecord{}
	}
	master.mustQuery("SELECT COUNT(*) AS n FROM items")
	if record(master).State != "S" || record(slave).State != "O" {
		t.Fatal("read failed to establish S/O")
	}
	sharedEpoch := record(master).Epoch
	master.mustQuery("EXPLAIN SELECT COUNT(*) AS n FROM items")
	if record(master).State != "S" || record(slave).State != "O" || record(master).Epoch != sharedEpoch {
		t.Fatal("EXPLAIN changed shared read ownership")
	}
	master.kill()
	if body := slave.mustQuery("SELECT COUNT(*) AS n FROM items"); replicaNumber(body, "n") != 16 {
		t.Fatal(body)
	}
	if status, body := slave.query("replica_test", "INSERT INTO items VALUES(9000,9000)", "replication-test-password"); status != 500 || !strings.Contains(body, replicationNetworkError) {
		t.Fatalf("O owner wrote without invalidating offline S reader: %d %s", status, body)
	}
	master.start()
	deadline = time.Now().Add(10 * time.Second)
	for {
		var encoded string
		if err := json.Unmarshal([]byte(master.admin("(replication_slaves)")), &encoded); err != nil {
			t.Fatal(err)
		}
		var statuses []replicaStatus
		_ = json.Unmarshal([]byte(encoded), &statuses)
		if len(statuses) == 1 && statuses[0].Connected {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("shared owner did not reconnect")
		}
		time.Sleep(50 * time.Millisecond)
	}
	var readers sync.WaitGroup
	errors := make(chan string, 32)
	for i := 0; i < 32; i++ {
		readers.Add(1)
		go func(i int) {
			defer readers.Done()
			p := master
			if i%2 == 0 {
				p = slave
			}
			for j := 0; j < 5; j++ {
				status, body := p.query("replica_test", "SELECT COUNT(*) AS n FROM items", "replication-test-password")
				if status != 200 || replicaNumber(body, "n") != 16 {
					errors <- fmt.Sprintf("parallel read: %d %s", status, body)
					return
				}
			}
		}(i)
	}
	readers.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
	if record(master).State != "S" || record(slave).State != "O" || record(master).Epoch != sharedEpoch {
		t.Fatal("parallel analytics modified S/O rights or ownership epoch")
	}
	// Execute the identical cached write twice, with sharing in between.
	// Both misses and hits must upgrade to M and invalidate other readers.
	for i := 0; i < 2; i++ {
		master.mustQuery("UPDATE items SET value=value+1 WHERE id=1")
		if record(master).State != "M" || record(slave).State != "IO" {
			t.Fatal("write failed to invalidate S/O")
		}
		slave.mustQuery("SELECT COUNT(*) AS n FROM items")
		if record(master).State != "O" || record(slave).State != "S" {
			t.Fatal("slave read failed to share master generation")
		}
	}
	slave.mustQuery("UPDATE items SET value=value-2 WHERE id=1")
	if record(slave).State != "M" || record(master).State != "IO" {
		t.Fatal("slave did not reclaim M")
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
	type readResult struct {
		status int
		body   string
	}
	waitingRead := make(chan readResult, 1)
	go func() {
		status, body := master.query("replica_test", "SELECT COUNT(*) AS n FROM items", "replication-test-password")
		waitingRead <- readResult{status, body}
	}()
	select {
	case result := <-waitingRead:
		t.Fatalf("transfer crossed uncommitted transaction: %d %s", result.status, result.body)
	case <-time.After(100 * time.Millisecond):
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case result := <-waitingRead:
		if result.status != 200 || replicaNumber(result.body, "n") != 17 {
			t.Fatalf("reader after commit: %d %s", result.status, result.body)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("reader was not released after committed transfer")
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
	deadline = time.Now().Add(10 * time.Second)
	for {
		masterData, _ := os.ReadFile(filepath.Join(master.data, ".replication-state"))
		slaveData, _ := os.ReadFile(filepath.Join(slave.data, ".replication-state"))
		var masterJournal, slaveJournal replicaJournal
		if json.Unmarshal(masterData, &masterJournal) != nil || json.Unmarshal(slaveData, &slaveJournal) != nil {
			t.Fatal("invalid backup journal")
		}
		complete := true
		for key, source := range slaveJournal.Records {
			if source.Resource.Database == "replica_test" && source.Resource.Shard != "" && source.State == "M" && (masterJournal.Records[key].Generation == "" || masterJournal.Records[key].Generation != source.Generation) {
				complete = false
			}
		}
		if complete {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("reconnected slave did not publish every rebuild backup")
		}
		time.Sleep(20 * time.Millisecond)
	}
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

func TestReplicationIndependentPartitionOwnership(t *testing.T) {
	if testing.Short() {
		t.Skip("partition ownership process test")
	}
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	binary := os.Getenv("MEMCP_REPLICATION_BINARY")
	if binary == "" {
		binary = filepath.Join(t.TempDir(), "memcp")
		command := exec.Command("go", "build", "-tags=nophp", "-o", binary, ".")
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("build: %v: %s", err, output)
		}
	}
	port := replicaTestPort(t)
	master := &replicaProcess{t: t, binary: binary, root: root, data: t.TempDir(), port: replicaTestPort(t), replication: port, password: "replication-test-password"}
	slave := &replicaProcess{t: t, binary: binary, root: root, data: t.TempDir(), port: replicaTestPort(t), replication: "127.0.0.1:" + port, password: "different-local-password"}
	t.Cleanup(func() {
		slave.kill()
		master.kill()
		if t.Failed() {
			t.Logf("MASTER\n%s\nSLAVE\n%s", master.logs(), slave.logs())
		}
	})
	master.start()
	if status, body := master.query("system", "CREATE DATABASE replica_test", master.password); status != 200 {
		t.Fatalf("database: %d %s", status, body)
	}
	master.mustQuery("CREATE TABLE items(id INT PRIMARY KEY, value INT) ENGINE=SAFE")
	master.mustQuery("INSERT INTO items VALUES(1,10),(9,90)")
	if body := master.admin(`(partitiontable (table "replica_test" false "items" false) (list "id" (list 5)))`); strings.TrimSpace(body) != "true" {
		t.Fatalf("partition: %s", body)
	}
	master.mustQuery("CREATE TABLE volatile_items(id INT, value TEXT) ENGINE=MEMORY")
	master.mustQuery("INSERT INTO volatile_items VALUES(1,'streamed-memory-generation')")
	slave.start()
	readyDeadline := time.Now().Add(10 * time.Second)
	for {
		status, _ := slave.query("system", "SELECT 1", master.password)
		if status == 200 {
			break
		}
		if time.Now().After(readyDeadline) {
			t.Fatal("initial upstream authentication did not become ready")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if body := slave.mustQuery("SELECT value FROM volatile_items WHERE id=1"); !strings.Contains(body, "streamed-memory-generation") {
		t.Fatalf("volatile generation transfer: %s", body)
	}
	if body := slave.mustQuery("SELECT COUNT(*) AS n FROM items"); replicaNumber(body, "n") != 2 {
		t.Fatal(body)
	}
	journal := func(process *replicaProcess) replicaJournal {
		data, err := os.ReadFile(filepath.Join(process.data, ".replication-state"))
		if err != nil {
			t.Fatal(err)
		}
		var journal replicaJournal
		if err := json.Unmarshal(data, &journal); err != nil {
			t.Fatal(err)
		}
		return journal
	}
	deadline := time.Now().Add(10 * time.Second)
	for !journal(slave).Initialized["replica_test"] {
		if time.Now().After(deadline) {
			t.Fatal("initial shard M claims did not finish")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if body := master.mustQuery("SELECT value FROM items WHERE id=1"); replicaNumber(body, "value") != 10 {
		t.Fatal(body)
	}
	counts := func(process *replicaProcess) map[string]int {
		result := make(map[string]int)
		for _, record := range journal(process).Records {
			if record.Resource.Table == "items" {
				result[record.State]++
			}
		}
		return result
	}
	if flags := counts(master); flags["S"] != 1 || flags["IO"] != 1 {
		t.Fatalf("master acquired unrelated partition: %v", flags)
	}
	if flags := counts(slave); flags["O"] != 1 || flags["M"] != 1 {
		t.Fatalf("slave lost independent partition ownership: %v", flags)
	}
	master.kill()
	slave.mustQuery("UPDATE items SET value=value+1 WHERE id=9")
	if status, body := slave.query("replica_test", "UPDATE items SET value=value+1 WHERE id=1", master.password); status != 500 || !strings.Contains(body, replicationNetworkError) {
		t.Fatalf("offline O write: %d %s", status, body)
	}
	master.start()
	slave.kill()
	// Restarted master may load its valid shared partition, but must not use
	// its older backend backup for the independently modified IO partition.
	if body := master.mustQuery("SELECT value FROM items WHERE id=1"); replicaNumber(body, "value") != 10 {
		t.Fatal(body)
	}
	if status, body := master.query("replica_test", "SELECT value FROM items WHERE id=9", master.password); status != 500 || !strings.Contains(body, replicationNetworkError) {
		t.Fatalf("IO fell back to stale backup: %d %s", status, body)
	}
	slave.start()
	if body := slave.mustQuery("SELECT value FROM items WHERE id=9"); replicaNumber(body, "value") != 91 {
		t.Fatalf("restart lost autonomous M write: %s", body)
	}
}
