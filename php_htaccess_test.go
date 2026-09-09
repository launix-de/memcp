//go:build php

// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later
package main

import "os"
import "time"
import "testing"
import "net/http/httptest"
import "path/filepath"

func TestPHPAccessRules(t *testing.T) {
	root := t.TempDir()
	app := filepath.Join(root, "app")
	os.Mkdir(app, 0700)
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(app, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("conf.json", "secret")
	write("config.json", "secret")
	write("hello.txt", "hello")
	write("fop-router.php", "<?php")
	write(".htaccess", `RewriteEngine On
RewriteRule ^conf\.json$ /missing [L]
RewriteRule ^config\.json$ - [F]
RewriteCond %{REQUEST_FILENAME} !-f
RewriteCond %{REQUEST_FILENAME} !-d
RewriteRule ^(.*)$ fop-router.php?_q=$1 [NC,L,QSA,B,UnsafeAllow3F]
`)
	cache := newPHPAccessCache()
	defer cache.close()
	for _, tc := range []struct {
		uri, path, q string
		status       int
	}{
		{"/app/conf.json", "/missing", "", 0}, {"/app/config.json", "/app/config.json", "", 403},
		{"/app/hello.txt", "/app/hello.txt", "", 0}, {"/app/", "/app/", "", 0},
		{"/app/view/a%26b%3Fc?existing=yes", "/app/fop-router.php", "view/a&b?c", 0},
	} {
		r, status := cache.rewrite(root, httptest.NewRequest("GET", tc.uri, nil))
		if status != tc.status || r.URL.Path != tc.path || r.URL.Query().Get("_q") != tc.q {
			t.Fatalf("%s: %d %s", tc.uri, status, r.URL.String())
		}
		if tc.q != "" && r.URL.Query().Get("existing") != "yes" {
			t.Fatal("QSA lost query")
		}
	}
	// Atomic replacement and later creation of formerly absent files invalidate
	// cached policy; malformed files fail closed rather than exposing config.
	expect := func(want int) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			_, s := cache.rewrite(root, httptest.NewRequest("GET", "/app/config.json", nil))
			if s == want {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("policy did not become %d", want)
	}
	write("replacement", "RewriteEngine On\nRewriteRule ^config.json$ - [G]\n")
	if err := os.Rename(filepath.Join(app, "replacement"), filepath.Join(app, ".htaccess")); err != nil {
		t.Fatal(err)
	}
	expect(410)
	write(".htaccess", "Require something-unsupported\n")
	expect(500)
	if err := os.Remove(filepath.Join(app, ".htaccess")); err != nil {
		t.Fatal(err)
	}
	expect(0)
	write(".htaccess", "RewriteEngine On\nRewriteRule ^config.json$ - [F]\n")
	expect(403)
	// Root policies are inherited, including by a nested app with its own file.
	if err := os.WriteFile(filepath.Join(root, ".htaccess"), []byte("RewriteEngine On\nRewriteRule ^app/config.json$ - [G]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	expect(410)
}

func TestPHPAccessRejectsUnsupportedRules(t *testing.T) {
	for _, source := range []string{"RewriteEngine On\nRewriteRule x y [UNKNOWN]", "RewriteEngine On\nRewriteCond %{REQUEST_FILENAME} !-f", "Options Indexes", "RewriteEngine On\nRewriteRule [ x"} {
		if parsePHPAccess(source).err == nil {
			t.Errorf("accepted %q", source)
		}
	}
}

// L stops the current directory's rules, not a descendant's access checks.
func TestPHPAccessLastRuleKeepsChildProtection(t *testing.T) {
	root := t.TempDir()
	app := filepath.Join(root, "app")
	if err := os.Mkdir(app, 0700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		filepath.Join(root, ".htaccess"):  "RewriteEngine On\nRewriteRule ^ - [L]\n",
		filepath.Join(app, ".htaccess"):   "RewriteEngine On\nRewriteRule ^config\\.json$ - [F]\n",
		filepath.Join(app, "config.json"): "secret",
	} {
		if err := os.WriteFile(name, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cache := newPHPAccessCache()
	defer cache.close()
	_, status := cache.rewrite(root, httptest.NewRequest("GET", "/app/config.json", nil))
	if status != 403 {
		t.Fatalf("child protection bypassed: %d", status)
	}
}

func TestPHPAccessEngineSettingAppliesToWholeFile(t *testing.T) {
	for _, source := range []string{
		"RewriteRule ^ - [F]\nRewriteEngine On\n",
		"RewriteEngine Off\nRewriteRule ^ - [F]\nRewriteEngine On\n",
	} {
		p := parsePHPAccess(source)
		if p.err != nil || len(p.rules) != 1 {
			t.Fatalf("rules missing: %+v", p)
		}
	}
	p := parsePHPAccess("RewriteEngine On\nRewriteRule ^ - [F]\nRewriteEngine Off\n")
	if p.err != nil || len(p.rules) != 0 {
		t.Fatalf("disabled rules active: %+v", p)
	}
}

func TestPHPAccessDirectoryRequestsApplyChildRules(t *testing.T) {
	root := t.TempDir()
	app := filepath.Join(root, "app")
	if err := os.Mkdir(app, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, ".htaccess"), []byte("RewriteEngine On\nRewriteRule ^$ - [F]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cache := newPHPAccessCache()
	defer cache.close()
	for _, uri := range []string{"/app", "/app/"} {
		_, status := cache.rewrite(root, httptest.NewRequest("GET", uri, nil))
		if status != 403 {
			t.Errorf("%s bypassed directory policy: %d", uri, status)
		}
	}
}

func TestPHPAccessProtectsImplicitIndexAndFallback(t *testing.T) {
	root := t.TempDir()
	for name, body := range map[string]string{
		".htaccess": "RewriteEngine On\nRewriteRule ^index\\.php$ - [F]\n",
		"index.php": "<?php echo 'protected';",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	handler := phpHandler(root, "index.php", "")
	for _, uri := range []string{"/", "/index.php", "/missing"} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest("GET", uri, nil))
		if recorder.Code != 403 {
			t.Errorf("%s bypassed script policy: %d", uri, recorder.Code)
		}
	}
}

func TestPHPAccessDirectoryReplacementRefreshesWatcher(t *testing.T) {
	root := t.TempDir()
	app := filepath.Join(root, "app")
	install := func(body string) {
		t.Helper()
		if err := os.Mkdir(app, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(app, ".htaccess"), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	install("RewriteEngine On\nRewriteRule ^secret$ - [G]\n")
	cache := newPHPAccessCache()
	defer cache.close()
	check := func(want int) {
		t.Helper()
		_, status := cache.rewrite(root, httptest.NewRequest("GET", "/app/secret", nil))
		if status != want {
			t.Fatalf("directory replacement policy: got %d, want %d", status, want)
		}
	}
	check(410)
	if err := os.Rename(app, filepath.Join(root, "old-app")); err != nil {
		t.Fatal(err)
	}
	install("RewriteEngine On\nRewriteRule ^secret$ - [F]\n")
	check(403)
}
