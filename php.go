//go:build php

// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import "os"
import "fmt"
import "sync"
import "time"
import "path"
import "strconv"
import "strings"
import "net/http"
import "path/filepath"
import "github.com/dunglas/frankenphp"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/phpbridge"

var phpLifecycle sync.Mutex
var phpReady = make(chan struct{})

type phpConfig struct {
	threads       int
	memoryLimit   int64
	maxWait       time.Duration
	outputBuffer  int
	opcacheMemory int
}

var phpSettings phpConfig

func parsePHPConfig(args []string) (phpConfig, error) {
	c := phpConfig{4, 256 << 20, 30 * time.Second, 4096, 128}
	for _, arg := range args {
		if !strings.HasPrefix(arg, "--php-") {
			continue
		}
		key, value, ok := strings.Cut(arg, "=")
		if !ok {
			return c, fmt.Errorf("%s requires =VALUE", key)
		}
		switch key {
		case "--php-threads", "--php-output-buffer", "--php-opcache-memory":
			n, err := strconv.Atoi(value)
			if err != nil || n < 0 || (key != "--php-output-buffer" && n == 0) {
				return c, fmt.Errorf("invalid %s: %q", key, value)
			}
			switch key {
			case "--php-threads":
				c.threads = n
			case "--php-output-buffer":
				c.outputBuffer = n
			case "--php-opcache-memory":
				if n < 32 {
					return c, fmt.Errorf("php-opcache-memory must be at least 32 MiB")
				}
				c.opcacheMemory = n
			}
		case "--php-memory-limit":
			shift := uint(0)
			if len(value) > 0 {
				switch value[len(value)-1] {
				case 'k', 'K':
					shift = 10
				case 'm', 'M':
					shift = 20
				case 'g', 'G':
					shift = 30
				}
				if shift > 0 {
					value = value[:len(value)-1]
				}
			}
			n, err := strconv.ParseUint(value, 10, int(63-shift))
			if err != nil || n<<shift < 8<<20 {
				return c, fmt.Errorf("php-memory-limit must be finite and at least 8 MiB (e.g. 256M)")
			}
			c.memoryLimit = int64(n << shift)
		case "--php-max-wait":
			d, err := time.ParseDuration(value)
			if err != nil || d < 0 {
				return c, fmt.Errorf("php-max-wait must be a nonnegative duration")
			}
			c.maxWait = d
		default:
			return c, fmt.Errorf("unsupported PHP option %s", key)
		}
	}
	return c, nil
}

func (c phpConfig) ini() map[string]string {
	limit := strconv.FormatInt(c.memoryLimit, 10)
	return map[string]string{
		"expose_php": "0", "display_errors": "0", "log_errors": "1",
		"memory_limit": limit, "max_memory_limit": limit,
		"output_buffering": strconv.Itoa(c.outputBuffer), "implicit_flush": "0",
		"opcache.enable": "1", "opcache.memory_consumption": strconv.Itoa(c.opcacheMemory),
		"opcache.interned_strings_buffer": "16", "opcache.max_accelerated_files": "20000",
		"opcache.validate_timestamps": "1", "opcache.revalidate_freq": "0", "opcache.jit": "disable",
	}
}

var phpStarted, phpStopping bool
var phpInitErr error
var phpAccessCachesMu sync.Mutex
var phpAccessCaches []*phpAccessCache

// HTTP listeners belong to Scheme's serve. This only publishes PHP settings
// after Scheme initialization; the first PHP request starts the shared runtime.
func startPHP(args []string) error {
	settings, err := parsePHPConfig(args)
	if err != nil {
		return err
	}
	if allocator, present := os.LookupEnv("USE_ZEND_ALLOC"); present && allocator != "1" {
		return fmt.Errorf("PHP memory quota requires USE_ZEND_ALLOC to be unset or 1")
	}
	phpSettings = settings
	close(phpReady)
	return nil
}

func ensurePHP() error {
	phpLifecycle.Lock()
	defer phpLifecycle.Unlock()
	if phpStopping {
		return fmt.Errorf("PHP is shutting down")
	}
	if phpStarted || phpInitErr != nil {
		return phpInitErr
	}
	if err := phpbridge.Register(&IOEnv); err != nil {
		phpInitErr = err
		return err
	}
	if err := frankenphp.Init(frankenphp.WithNumThreads(phpSettings.threads), frankenphp.WithMaxThreads(phpSettings.threads), frankenphp.WithMaxWaitTime(phpSettings.maxWait), frankenphp.WithPhpIni(phpSettings.ini())); err != nil {
		phpInitErr = err
		return err
	}
	phpStarted = true
	return nil
}

func stopPHP() {
	phpAccessCachesMu.Lock()
	for _, cache := range phpAccessCaches {
		cache.close()
	}
	phpAccessCaches = nil
	phpAccessCachesMu.Unlock()
	phpLifecycle.Lock()
	defer phpLifecycle.Unlock()
	phpStopping = true
	if phpStarted {
		frankenphp.Shutdown()
		phpStarted = false
	}
}

// Like serveStatic, relative document roots resolve against the importing SCM
// file. Each returned handler has its own root/mount; PHP threads are shared.
func getServePHP(wd string) func(...scm.Scmer) scm.Scmer {
	return func(a ...scm.Scmer) scm.Scmer {
		root := a[0].String()
		if !filepath.IsAbs(root) {
			root = filepath.Join(wd, root)
		}
		root, err := filepath.Abs(root)
		if err != nil {
			panic(err)
		}
		root, err = filepath.EvalSymlinks(root)
		if err != nil {
			panic(err)
		}
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			panic("PHP root must be a directory")
		}
		mount, front := "", ""
		if len(a) > 1 {
			mount = strings.TrimSuffix(a[1].String(), "/")
		}
		if len(a) > 2 {
			front = a[2].String()
		}
		if mount != "" && (!strings.HasPrefix(mount, "/") || path.Clean(mount) != mount || strings.ContainsAny(mount, "?\\#")) {
			panic("PHP mount must be an absolute URL path prefix")
		}
		if front != "" && (filepath.Base(front) != front || !strings.HasSuffix(front, ".php")) {
			panic("PHP front controller must be a PHP filename inside the document root")
		}
		handler := phpHandler(root, front, mount)
		return scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
			req := scm.Apply(a[0], scm.NewString("req")).Any().(*http.Request)
			res := scm.Apply(a[1], scm.NewString("res")).Any().(http.ResponseWriter)
			select {
			case <-phpReady:
			case <-req.Context().Done():
				return scm.NewNil()
			}
			if err := ensurePHP(); err != nil {
				panic(err)
			}
			res.Header().Del("Content-Type")
			handler.ServeHTTP(res, req)
			return scm.NewNil()
		})
	}
}

// PHP gets only existing script paths. Static files stay inside the resolved
// document root; dotfiles and PHP configuration/source backups are not served.
func phpHandler(root, front, mount string) http.Handler {
	files := http.FileServer(http.Dir(root))
	access := newPHPAccessCache()
	phpAccessCachesMu.Lock()
	phpAccessCaches = append(phpAccessCaches, access)
	phpAccessCachesMu.Unlock()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		original := r
		if mount != "" {
			if r.URL.Path == mount {
				u := *r.URL
				u.Path += "/"
				http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
				return
			}
			if !strings.HasPrefix(r.URL.Path, mount+"/") {
				http.NotFound(w, r)
				return
			}
			r = r.Clone(r.Context())
			r.URL.Path = strings.TrimPrefix(r.URL.Path, mount)
			r.URL.RawPath = ""
		}

		for _, part := range strings.Split(r.URL.Path, "/") {
			if strings.HasPrefix(part, ".") || strings.Contains(part, "\\") {
				http.NotFound(w, r)
				return
			}
		}
		var status int
		r, status = access.rewrite(root, r)
		if status != 0 {
			http.Error(w, http.StatusText(status), status)
			return
		}
		var requestPath, filename string
		var info os.FileInfo
		var err error
		for resolve := 0; ; resolve++ {
			if resolve == 16 {
				http.Error(w, http.StatusText(http.StatusLoopDetected), http.StatusLoopDetected)
				return
			}
			requestPath = r.URL.Path
			filename = filepath.Join(root, filepath.FromSlash(requestPath))
			info, err = os.Stat(filename)
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
					u.Path = mount + u.Path + "/"
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

			// DirectoryIndex and the configured fallback select another resource;
			// that resource's access rules apply just like an internal rewrite.
			if err == nil && requestPath != r.URL.Path {
				candidate := r.Clone(r.Context())
				candidate.URL.Path = requestPath
				candidate.URL.RawPath = ""
				checked, status := access.rewrite(root, candidate)
				if status != 0 {
					http.Error(w, http.StatusText(status), status)
					return
				}
				r = checked
				if r.URL.Path != requestPath {
					continue
				}
			}
			break
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
		r = r.Clone(r.Context())
		r.URL.Path = requestPath
		r.URL.RawPath = ""
		// CGI maps '-' to '_'; drop ambiguous header spellings before the SAPI.
		for name := range r.Header {
			if strings.Contains(name, "_") {
				r.Header.Del(name)
			}
		}
		scriptName := requestPath
		if split := strings.Index(strings.ToLower(scriptName), ".php/"); split >= 0 {
			scriptName = scriptName[:split+4]
		}
		request, err := frankenphp.NewRequestWithContext(r, frankenphp.WithRequestDocumentRoot(root, false), frankenphp.WithOriginalRequest(original), frankenphp.WithRequestEnv(map[string]string{
			"SCRIPT_NAME": mount + scriptName, "PHP_SELF": mount + requestPath,
		}))
		if err != nil {
			http.Error(w, "PHP request initialization failed", 500)
			return
		}
		if err := frankenphp.ServeHTTP(w, request); err != nil {
			fmt.Fprintln(os.Stderr, "PHP request:", err)
		}
	})
}
