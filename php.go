//go:build php

// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import "os"
import "fmt"
import "net"
import "sync"
import "time"
import "context"
import "strconv"
import "strings"
import "net/http"
import "path/filepath"
import "github.com/dunglas/frankenphp"
import "github.com/launix-de/memcp/phpbridge"

var phpServer *http.Server
var phpLifecycle sync.Mutex

func startPHP(args []string) error {
	phpLifecycle.Lock()
	defer phpLifecycle.Unlock()
	root, listen, front := "", "127.0.0.1:8080", ""
	threads := 4
	for _, arg := range args {
		if !strings.HasPrefix(arg, "--php-") {
			continue
		}
		key, value, ok := strings.Cut(arg, "=")
		if !ok {
			return fmt.Errorf("use %s=value", key)
		}
		switch key {
		case "--php-root":
			root = value
		case "--php-listen":
			listen = value
		case "--php-front-controller":
			front = value
		case "--php-threads":
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 {
				return fmt.Errorf("php-threads must be positive")
			}
			threads = n
		default:
			return fmt.Errorf("unknown PHP option %s", key)
		}
	}
	if root == "" {
		for _, arg := range args {
			if strings.HasPrefix(arg, "--php-") {
				return fmt.Errorf("--php-root is required")
			}
		}
		return nil
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		return err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("PHP root is not a directory")
	}
	if front != "" && (filepath.Base(front) != front || !strings.HasSuffix(front, ".php")) {
		return fmt.Errorf("PHP front controller must be a PHP filename inside the document root")
	}
	listener, err := net.Listen("tcp", listen)
	if err != nil {
		return err
	}
	if err := phpbridge.Register(&IOEnv); err != nil {
		listener.Close()
		return err
	}
	if err = frankenphp.Init(frankenphp.WithNumThreads(threads), frankenphp.WithMaxThreads(threads), frankenphp.WithPhpIni(map[string]string{
		"expose_php": "0", "display_errors": "0", "log_errors": "1", "opcache.enable": "1",
	})); err != nil {
		listener.Close()
		return err
	}
	phpServer = &http.Server{Handler: phpHandler(abs, front), ReadHeaderTimeout: 10 * time.Second}
	server := phpServer
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, "PHP server:", err)
		}
	}()
	fmt.Printf("PHP serving %s at http://%s (%d threads)\n", abs, listener.Addr(), threads)
	return nil
}

func stopPHP() {
	phpLifecycle.Lock()
	defer phpLifecycle.Unlock()
	if phpServer == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := phpServer.Shutdown(ctx); err != nil {
		phpServer.Close()
	}
	frankenphp.Shutdown()
	phpServer = nil
}

// PHP gets only existing script paths. Static files stay inside the resolved
// document root; dotfiles and PHP configuration/source backups are not served.
func phpHandler(root, front string) http.Handler {
	files := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, part := range strings.Split(r.URL.Path, "/") {
			if strings.HasPrefix(part, ".") || strings.Contains(part, "\\") {
				http.NotFound(w, r)
				return
			}
		}
		requestPath := r.URL.Path
		filename := filepath.Join(root, filepath.FromSlash(requestPath))
		info, err := os.Stat(filename)
		// Keep PATH_INFO in the request passed to PHP, but check the actual
		// script on disk before letting the SAPI split the path.
		if err != nil {
			if split := strings.Index(strings.ToLower(requestPath), ".php/"); split >= 0 {
				filename = filepath.Join(root, filepath.FromSlash(requestPath[:split+4]))
				info, err = os.Stat(filename)
			}
		}
		if err == nil && info.IsDir() {
			if !strings.HasSuffix(requestPath, "/") {
				u := *r.URL
				u.Path += "/"
				http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
				return
			}
			requestPath += "index.php"
			filename = filepath.Join(filename, "index.php")
			info, err = os.Stat(filename)
			if os.IsNotExist(err) {
				requestPath = strings.TrimSuffix(requestPath, "index.php") + "index.html"
				filename = filepath.Join(filepath.Dir(filename), "index.html")
				info, err = os.Stat(filename)
			}
		}
		if os.IsNotExist(err) && front != "" {
			requestPath = "/" + front
			filename = filepath.Join(root, front)
			info, err = os.Stat(filename)
		}
		if err != nil || !info.Mode().IsRegular() {
			http.NotFound(w, r)
			return
		}
		real, err := filepath.EvalSymlinks(filename)
		relative, relErr := filepath.Rel(root, real)
		if err != nil || relErr != nil || !filepath.IsLocal(relative) {
			http.NotFound(w, r)
			return
		}
		if !strings.HasSuffix(strings.ToLower(filename), ".php") {
			base := strings.ToLower(filepath.Base(filename))
			if strings.Contains(base, ".php") || strings.HasSuffix(base, ".ini") || strings.HasSuffix(base, ".sql") || strings.HasSuffix(base, "~") {
				http.NotFound(w, r)
				return
			}
			files.ServeHTTP(w, r)
			return
		}
		original := r
		r = r.Clone(r.Context())
		r.URL.Path = requestPath
		// CGI maps '-' to '_'; drop ambiguous header spellings before the SAPI.
		for name := range r.Header {
			if strings.Contains(name, "_") {
				r.Header.Del(name)
			}
		}
		request, err := frankenphp.NewRequestWithContext(r, frankenphp.WithRequestDocumentRoot(root, false), frankenphp.WithOriginalRequest(original))
		if err != nil {
			http.Error(w, "PHP request initialization failed", 500)
			return
		}
		if err := frankenphp.ServeHTTP(w, request); err != nil {
			fmt.Fprintln(os.Stderr, "PHP request:", err)
		}
	})
}
