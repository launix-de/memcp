//go:build php

// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import "io"
import "os"
import "fmt"
import "net"
import "sync"
import "time"
import "bufio"
import "context"
import "os/exec"
import "strconv"
import "strings"
import "testing"
import "net/http"
import "sync/atomic"
import "encoding/json"
import "path/filepath"
import binaryencoding "encoding/binary"

func TestPHPIntegration(t *testing.T) {
	for _, front := range []string{"", "index.php"} {
		name := "files"
		if front != "" {
			name = "front-controller"
		}
		t.Run(name, func(t *testing.T) { testPHPIntegration(t, front, "", false) })
	}
}

func TestPHPServeCLI(t *testing.T) {
	for _, mode := range []string{"split", "equals"} {
		t.Run(mode, func(t *testing.T) { testPHPIntegration(t, "index.php", mode, false) })
	}
}

func TestPHPServeCLIRequiresPath(t *testing.T) {
	for _, option := range []string{"--serve", "--serve="} {
		t.Run(option, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			output, err := exec.CommandContext(ctx, "./memcp-php", "--no-repl", "--disable-api", "--disable-mysql", "--mysql-socket=", "-data", t.TempDir(), option).CombinedOutput()
			if err == nil || !strings.Contains(string(output), "--serve requires a directory path") {
				t.Fatalf("expected missing PATH error: %v %s", err, output)
			}
		})
	}
}

func TestPHPWaitTimeout(t *testing.T) { testPHPIntegration(t, "", "", true) }

func testPHPIntegration(t *testing.T, front, cli string, queueTimeout bool) {
	binary, err := filepath.Abs("memcp-php")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	root := filepath.Join(dir, "public")
	if cli != "" {
		root += ".scm"
	}
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile("tests/php/integration.php")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"index.php", "probe.php"} {
		if err := os.WriteFile(filepath.Join(root, name), source, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, app := range []string{"first", "second"} {
		language := "de"
		if app == "second" {
			language = "fr"
		}
		path := filepath.Join(root, "catalogs", app, language, "LC_MESSAGES")
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
		originals := []string{"", "%s Discounts", "Hello", "item\x00items"}
		translations := []string{"Content-Type: text/plain; charset=UTF-8\nPlural-Forms: nplurals=2; plural=(n != 1);\n", app + " %s", app, app + " singular\x00" + app + " plural"}
		data := make([]byte, 28+16*len(originals))
		binaryencoding.LittleEndian.PutUint32(data, 0x950412de)
		binaryencoding.LittleEndian.PutUint32(data[8:], uint32(len(originals)))
		binaryencoding.LittleEndian.PutUint32(data[12:], 28)
		binaryencoding.LittleEndian.PutUint32(data[16:], uint32(28+8*len(originals)))
		for i, value := range append(originals, translations...) {
			binaryencoding.LittleEndian.PutUint32(data[28+i*8:], uint32(len(value)))
			binaryencoding.LittleEndian.PutUint32(data[32+i*8:], uint32(len(data)))
			data = append(data, []byte(value)...)
			data = append(data, 0)
		}
		if err := os.WriteFile(filepath.Join(path, "messages.mo"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}

	for _, name := range []string{"hello.txt", ".env", "source.php.bak"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	secret := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "link.txt")); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	log, err := os.Create(filepath.Join(dir, "server.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	socketPath := filepath.Join(dir, "mysql.sock")
	// Apps are attached by Scheme on the ordinary HTTP listener. Resolve root
	// relative to this imported script, and keep a second mount independent.
	other := filepath.Join(dir, "other")
	if err := os.Mkdir(other, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "second.txt"), []byte("second app"), 0600); err != nil {
		t.Fatal(err)
	}
	freePort := func() string {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		_, port, _ := net.SplitHostPort(listener.Addr().String())
		listener.Close()
		return port
	}
	mysqlPort, otherPort := freePort(), freePort()
	imapPort := ""
	if os.Getenv("MEMCP_TEST_IMAP_BINARY") != "" {
		imapPort = testIMAPMailbox(t)
	}
	config, _ := json.Marshal(map[string]string{"socket": socketPath, "port": mysqlPort, "other_port": otherPort, "imap_port": imapPort})
	if err := os.WriteFile(filepath.Join(root, "wire.json"), config, 0600); err != nil {
		t.Fatal(err)
	}
	mount := fmt.Sprintf(`(define app (servePHP "public" "/app" %s))
(define second (servePHP "other" "/other"))
(define http_handler (begin (define previous http_handler) (lambda (req res)
 (match (req "path")
  (regex "^/app(/|$)" _ _) (app req res)
  (regex "^/other(/|$)" _ _) (second req res)
  "/outside" ((res "print") "Scheme handler")
  _ (previous req res)))))`, strconv.Quote(front))
	if cli != "" {
		mount = strings.ReplaceAll(mount, `"public"`, `"public.scm"`)
	}
	mount += "\n(mysql " + otherPort + " mysql_auth mysql_schema mysql_handler)"
	mountFile := filepath.Join(dir, "mount.scm")
	if err := os.WriteFile(mountFile, []byte(mount), 0600); err != nil {
		t.Fatal(err)
	}
	_, port, _ := net.SplitHostPort(address)
	apiFlag := "--api-port=" + port
	if front != "" && cli == "" {
		apiFlag = "--disable-api"
		if err := os.WriteFile(mountFile, []byte(mount+"\n(serve "+port+" (lambda (req res) (http_handler req res)) \"127.0.0.1\")"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(binary, "--no-repl", apiFlag, "--mysql-port="+mysqlPort, "--mysql-socket="+socketPath, "-data", filepath.Join(dir, "data"), "-c", `(createdatabase "memcp-tests" true)`, "-c", `(settings "PHPMemoryLimit" 33554432)`, "-c", `(settings "PHPMaxWaitMilliseconds" 5000)`, "lib/main.scm", mountFile)
	if helper := os.Getenv("MEMCP_TEST_IMAP_BINARY"); helper != "" {
		cmd.Args = append(cmd.Args, "-c", `(settings "PHPIMAPBinary" `+strconv.Quote(helper)+`)`)
	}
	if queueTimeout {
		cmd.Args = append(cmd.Args, "-c", `(settings "PHPMaxWaitMilliseconds" 100)`)
	}
	if cli != "" {
		workingDir, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		relative, err := filepath.Rel(workingDir, root)
		if err != nil {
			t.Fatal(err)
		}
		if cli == "split" {
			cmd.Args = append(cmd.Args, "--serve", relative)
		} else {
			cmd.Args = append(cmd.Args, "--serve="+relative)
		}
	}
	cmd.Stdout = log
	cmd.Stderr = log
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	t.Cleanup(func() {
		cmd.Process.Signal(os.Interrupt)
		select {
		case <-done:
		case <-time.After(40 * time.Second):
			cmd.Process.Kill()
			<-done
		}
		if t.Failed() {
			b, _ := os.ReadFile(log.Name())
			t.Log(string(b))
		}
	})
	client := &http.Client{Timeout: 10 * time.Second}
	get := func(path string) (int, string, error) {
		res, err := client.Get("http://" + address + path)
		if err != nil {
			return 0, "", err
		}
		defer res.Body.Close()
		b, err := io.ReadAll(res.Body)
		return res.StatusCode, string(b), err
	}
	ready := false
	for deadline := time.Now().Add(120 * time.Second); time.Now().Before(deadline); {
		status, body, err := get("/app/probe.php")
		if err == nil && status == 200 {
			var probe struct {
				Zts     bool
				Opcache bool
				Drivers []string
			}
			if err := json.Unmarshal([]byte(body), &probe); err != nil {
				t.Fatal(body, err)
			}
			if !probe.Zts || !probe.Opcache {
				t.Fatal("ZTS and OPcache required:", body)
			}
			if !strings.Contains(strings.Join(probe.Drivers, ","), "sqlite") {
				t.Fatal("pdo_sqlite required for coexistence test")
			}
			ready = true
			break
		}
		select {
		case err := <-done:
			done <- err
			t.Fatalf("PHP startup exited: %v", err)
		default:
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !ready {
		t.Fatal("PHP did not become ready")
	}
	if cli != "" {
		for _, path := range []string{"/", "/a/pretty/permalink"} {
			status, body, err := get(path + "?action=routing")
			var result map[string]string
			if err != nil || status != 200 || json.Unmarshal([]byte(body), &result) != nil || result["script"] != "/index.php" || result["uri"] != path+"?action=routing" {
				t.Fatalf("CLI root %s: %d %s %v", path, status, body, err)
			}
		}
		status, _, err := get("/dashboard")
		if err != nil || status != 401 {
			t.Fatalf("dashboard authentication: %d %v", status, err)
		}
		for _, path := range []string{"/dashboard", "/dashboard/api/whoami", "/sql/memcp-tests"} {
			req, err := http.NewRequest("GET", "http://"+address+path, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.SetBasicAuth("root", "admin")
			response, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(response.Body)
			response.Body.Close()
			if err != nil || response.StatusCode != 200 || strings.Contains(string(body), `"script":"/index.php"`) {
				t.Fatalf("MemCP route %s: %d %s %v", path, response.StatusCode, body, err)
			}
		}
	}
	if os.Getenv("MEMCP_TEST_IMAP_BINARY") != "" {
		for _, action := range []string{"imap", "imap-abandon", "imap-exit", "imap-oom", "imap"} {
			status, body, err := get("/app/probe.php?action=" + action)
			want := 200
			if action == "imap-abandon" || action == "imap-oom" {
				want = 500
			}
			if err != nil || status != want {
				t.Fatalf("%s: %d %s %v", action, status, body, err)
			}
		}
	}
	if os.Getenv("MEMCP_TEST_PHP_EXTENSIONS") != "" {
		status, body, err := get("/app/probe.php?action=extensions")
		if err != nil || status != 200 {
			t.Fatalf("extensions: %d %s %v", status, body, err)
		}
	}
	for round := 0; round < 3; round++ {
		var localeWG sync.WaitGroup
		for i := 0; i < 4; i++ {
			localeWG.Add(1)
			go func(i int) {
				defer localeWG.Done()
				app := []string{"first", "second"}[i%2]
				status, body, err := get("/app/probe.php?action=gettext&app=" + app)
				if err != nil || status != 200 || !strings.Contains(body, `"ok":true`) {
					t.Errorf("gettext %s: %d %s %v", app, status, body, err)
				}
			}(i)
		}
		localeWG.Wait()
	}
	for _, action := range []string{"setup", "quota", "oom-php", "oom-pdo", "pdo", "wire", "route-dsn", "buffers", "latency", "abandon", "verify"} {
		status, body, err := get("/app/probe.php?action=" + action)
		want := 200
		if action == "abandon" || strings.HasPrefix(action, "oom-") {
			want = 500
		}
		if err != nil || status != want || (strings.HasPrefix(action, "oom-") && strings.Contains(body, "Memory ceiling was not enforced")) {
			t.Fatalf("%s: status %d, %s, %v", action, status, body, err)
		}
	}
	missingStatus := 404
	if front != "" {
		missingStatus = 200
	}
	for path, want := range map[string]int{"/hello.txt": 200, "/.env": 404, "/source.php.bak": 404, "/link.txt": 404, "/absent.php": missingStatus, "/": 200} {
		status, body, err := get("/app" + path)
		if err != nil || status != want {
			t.Fatalf("%s: %d %s %v", path, status, body, err)
		}
	}
	status, body, err := get("/app/probe.php/extra?action=routing")
	var routing struct {
		Script   string
		PathInfo string `json:"path_info"`
		URI      string `json:"uri"`
	}
	if err != nil || status != 200 || json.Unmarshal([]byte(body), &routing) != nil || routing.Script != "/app/probe.php" || routing.PathInfo != "/extra" || routing.URI != "/app/probe.php/extra?action=routing" {
		t.Fatalf("PATH_INFO: %d %s %v", status, body, err)
	}
	req, err := http.NewRequest("POST", "http://"+address+"/app/probe.php?action=routing", strings.NewReader("value=hello%26world"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", "probe=cookie-value")
	req.Header.Set("X-Probe", "header-value")
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var posted map[string]string
	err = json.NewDecoder(response.Body).Decode(&posted)
	response.Body.Close()
	if err != nil || response.StatusCode != 200 || posted["method"] != "POST" || posted["post"] != "hello&world" || posted["cookie"] != "cookie-value" || posted["header"] != "header-value" {
		t.Fatalf("POST/cookie/header routing: %v %v", posted, err)
	}
	if front != "" {
		status, body, err := get("/app/example/route?action=routing")
		if err != nil || status != 200 || json.Unmarshal([]byte(body), &routing) != nil || routing.Script != "/app/index.php" || routing.URI != "/app/example/route?action=routing" {
			t.Fatalf("front controller: %d %s %v", status, body, err)
		}
	}
	for path, want := range map[string]string{"/outside": "Scheme handler", "/other/second.txt": "second app"} {
		status, body, err := get(path)
		if err != nil || status != 200 || body != want {
			t.Fatalf("mount %s: %d %s %v", path, status, body, err)
		}
	}
	status, _, err = get("/application/probe.php")
	if err != nil || (cli == "" && status != 404) || (cli != "" && status != 200) {
		t.Fatalf("mount boundary: %d %v", status, err)
	}
	var wg sync.WaitGroup
	var intervals [12]struct{ Start, End float64 }
	var rejected [12]bool
	for i := range intervals {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			status, body, err := get(fmt.Sprintf("/app/probe.php?action=parallel&value=%d", i))
			if queueTimeout && err == nil && status == 503 {
				rejected[i] = true
				return
			}
			if err != nil || status != 200 {
				t.Errorf("parallel: %d %s %v", status, body, err)
			}
			if err := json.Unmarshal([]byte(body), &intervals[i]); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	overlap := false
	for i, a := range intervals {
		active := 0
		for _, b := range intervals {
			if b.Start <= a.Start && a.Start < b.End {
				active++
			}
		}
		if active > 4 {
			t.Errorf("PHP thread cap exceeded: %d", active)
		}
		for j, b := range intervals {
			if i != j && a.Start < b.End && b.Start < a.End {
				overlap = true
			}
		}
	}
	if !overlap {
		t.Error("PHP requests did not overlap")
	}
	if queueTimeout {
		found := false
		for _, r := range rejected {
			found = found || r
		}
		if !found {
			t.Error("saturated PHP pool did not time out queued requests")
		}
		if status, body, err := get("/app/probe.php?action=verify"); err != nil || status != 200 {
			t.Fatalf("pool failed to recover: %d %s %v", status, body, err)
		}
	}
}

func TestPHPQuotaRequiresZendAllocator(t *testing.T) {
	for _, value := range []string{"0", "", "false"} {
		t.Run("value="+value, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "./memcp-php", "--no-repl", "--disable-api", "--disable-mysql", "--mysql-socket=", "-data", t.TempDir())
			cmd.Env = append(os.Environ(), "USE_ZEND_ALLOC="+value)
			output, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(output), "PHP memory quota requires USE_ZEND_ALLOC") {
				t.Fatalf("allocator bypass: %v %s", err, output)
			}
		})
	}
}

// A private mailbox exercises native c-client handles and stream results
// without depending on an external account or sending any mail.
func testIMAPMailbox(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var active atomic.Int32
	t.Cleanup(func() {
		listener.Close()
		deadline := time.Now().Add(time.Second)
		for active.Load() != 0 && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}
		if active.Load() != 0 {
			t.Errorf("IMAP request leaked %d mailbox connections", active.Load())
		}
	})
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			active.Add(1)
			go func() {
				defer active.Add(-1)
				defer conn.Close()
				conn.SetDeadline(time.Now().Add(20 * time.Second))
				fmt.Fprint(conn, "* OK isolated test mailbox\r\n")
				scanner := bufio.NewScanner(conn)
				for scanner.Scan() {
					parts := strings.SplitN(scanner.Text(), " ", 3)
					if len(parts) < 2 {
						return
					}
					tag, command := parts[0], strings.ToUpper(parts[1])
					switch command {
					case "CAPABILITY":
						fmt.Fprint(conn, "* CAPABILITY IMAP4rev1\r\n")
					case "LOGIN", "NOOP":
					case "SELECT", "EXAMINE":
						fmt.Fprint(conn, "* FLAGS (\\Seen)\r\n* 1 EXISTS\r\n* 0 RECENT\r\n* OK [UIDVALIDITY 1] valid\r\n* OK [UIDNEXT 2] next\r\n")
					case "FETCH":
						text := "hello mailbox\r\n"
						field := "BODY[TEXT]"
						if len(parts) > 2 && strings.Contains(parts[2], "BODY.PEEK[]") {
							field = "BODY[]"
							text = "From: sender@example.test\r\nSubject: Isolated IMAP\r\n\r\n" + text
						}
						fmt.Fprintf(conn, "* 1 FETCH (%s {%d}\r\n%s)\r\n", field, len(text), text)
					case "LOGOUT":
						fmt.Fprintf(conn, "* BYE closing\r\n%s OK logged out\r\n", tag)
						return
					default:
						fmt.Fprintf(conn, "%s BAD unsupported %s\r\n", tag, command)
						continue
					}
					fmt.Fprintf(conn, "%s OK complete\r\n", tag)
				}
			}()
		}
	}()
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	return port
}
