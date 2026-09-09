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
import "os/exec"
import "strings"
import "strconv"
import "testing"
import "net/http"
import "encoding/json"
import "path/filepath"

func TestPHPIntegration(t *testing.T) {
	for _, front := range []string{"", "index.php"} {
		name := "files"
		if front != "" {
			name = "front-controller"
		}
		t.Run(name, func(t *testing.T) { testPHPIntegration(t, front) })
	}
}

func testPHPIntegration(t *testing.T, front string) {
	binary, err := filepath.Abs("memcp-php")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	root := filepath.Join(dir, "public")
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
	config, _ := json.Marshal(map[string]string{"socket": socketPath})
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
	mountFile := filepath.Join(dir, "mount.scm")
	if err := os.WriteFile(mountFile, []byte(mount), 0600); err != nil {
		t.Fatal(err)
	}
	_, port, _ := net.SplitHostPort(address)
	apiFlag := "--api-port=" + port
	if front != "" {
		apiFlag = "--disable-api"
		if err := os.WriteFile(mountFile, []byte(mount+"\n(serve "+port+" (lambda (req res) (http_handler req res)) \"127.0.0.1\")"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(binary, "--no-repl", apiFlag, "--disable-mysql", "--mysql-socket="+socketPath, "-data", filepath.Join(dir, "data"), "-c", `(createdatabase "memcp-tests" true)`, "--php-threads=4", "lib/main.scm", mountFile)
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
	for _, action := range []string{"setup", "pdo", "wire", "buffers", "latency", "abandon", "verify"} {
		status, body, err := get("/app/probe.php?action=" + action)
		want := 200
		if action == "abandon" {
			want = 500
		}
		if err != nil || status != want {
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
	if err != nil || status != 404 {
		t.Fatalf("mount boundary: %d %v", status, err)
	}
	var wg sync.WaitGroup
	var intervals [4]struct{ Start, End float64 }
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			status, body, err := get(fmt.Sprintf("/app/probe.php?action=parallel&value=%d", i))
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
		for j, b := range intervals {
			if i != j && a.Start < b.End && b.Start < a.End {
				overlap = true
			}
		}
	}
	if !overlap {
		t.Error("PHP requests did not overlap")
	}
}
