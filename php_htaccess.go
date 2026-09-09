//go:build php

// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import "os"
import "fmt"
import "sync"
import "path"
import "regexp"
import "strings"
import "net/url"
import "net/http"
import "sync/atomic"
import "path/filepath"

// Access files are executable routing policy. Unsupported syntax and read
// errors deny requests, never silently expose a file. Published rules are immutable.
type phpAccessRule struct {
	pattern    *regexp.Regexp
	target     string
	conditions []string
	flags      map[string]bool
}
type phpAccessPolicy struct {
	rules []phpAccessRule
	err   error
}
type phpAccessEntry struct {
	dir    os.FileInfo
	policy atomic.Pointer[phpAccessPolicy]
	stop   func()
}
type phpAccessCache struct {
	mu      sync.Mutex
	entries map[string]*phpAccessEntry
}

func newPHPAccessCache() *phpAccessCache {
	return &phpAccessCache{entries: make(map[string]*phpAccessEntry)}
}
func (c *phpAccessCache) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range c.entries {
		if e.stop != nil {
			e.stop()
		}
	}
}
func (c *phpAccessCache) get(dir string, info os.FileInfo) *phpAccessPolicy {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.entries[dir]; e != nil {
		if os.SameFile(e.dir, info) {
			return e.policy.Load()
		}
		e.stop()
		delete(c.entries, dir)
	}
	e := &phpAccessEntry{dir: info}
	e.policy.Store(&phpAccessPolicy{err: fmt.Errorf("access policy not loaded")})
	stop, err := watchFile(filepath.Join(dir, ".htaccess"), func(data []byte, err error) {
		var p *phpAccessPolicy
		if os.IsNotExist(err) {
			p = &phpAccessPolicy{}
		} else if err != nil {
			p = &phpAccessPolicy{err: err}
		} else {
			p = parsePHPAccess(string(data))
		}
		e.policy.Store(p)
	})
	if err != nil {
		return &phpAccessPolicy{err: err}
	}
	e.stop = stop
	c.entries[dir] = e
	return e.policy.Load()
}

func parsePHPAccess(source string) *phpAccessPolicy {
	p := &phpAccessPolicy{}
	enabled := false
	var conditions []string
	fail := func(line int) *phpAccessPolicy {
		return &phpAccessPolicy{err: fmt.Errorf("unsupported or invalid .htaccess directive at line %d", line+1)}
	}
	for line, text := range strings.Split(source, "\n") {
		text = strings.TrimSpace(text)
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		f := strings.Fields(text)
		switch strings.ToLower(f[0]) {
		case "rewriteengine":
			if len(f) != 2 || (strings.ToLower(f[1]) != "on" && strings.ToLower(f[1]) != "off") {
				return fail(line)
			}
			enabled = strings.EqualFold(f[1], "on")
		case "rewritecond":
			if len(f) != 3 || f[1] != "%{REQUEST_FILENAME}" || (f[2] != "!-f" && f[2] != "!-d" && f[2] != "-f" && f[2] != "-d") {
				return fail(line)
			}
			conditions = append(conditions, f[2])
		case "rewriterule":
			if len(f) < 3 || len(f) > 4 {
				return fail(line)
			}
			flags := map[string]bool{}
			if len(f) == 4 {
				if !strings.HasPrefix(f[3], "[") || !strings.HasSuffix(f[3], "]") {
					return fail(line)
				}
				for _, flag := range strings.Split(strings.Trim(f[3], "[]"), ",") {
					flag = strings.ToUpper(flag)
					switch flag {
					case "NC", "L", "END", "QSA", "B", "UNSAFEALLOW3F", "F", "G":
						flags[flag] = true
					default:
						return fail(line)
					}
				}
			}
			pattern := f[1]
			if flags["NC"] {
				pattern = "(?i)" + pattern
			}
			re, err := regexp.Compile(pattern)
			if err != nil {
				return fail(line)
			}
			if strings.Contains(f[2], ":") || strings.Contains(f[2], "%{") || strings.Contains(f[2], "\\") {
				return fail(line)
			}
			p.rules = append(p.rules, phpAccessRule{re, f[2], conditions, flags})
			conditions = nil
		default:
			return fail(line)
		}
	}
	if len(conditions) != 0 {
		p.err = fmt.Errorf("RewriteCond without RewriteRule")
	}
	if !enabled {
		p.rules = nil
	}
	return p
}

// Apply per-directory rules from the root through existing descendants, and
// re-enter after internal redirects. This runs before either PHP or static serving.
func (c *phpAccessCache) rewrite(root string, r *http.Request) (*http.Request, int) {
	r = r.Clone(r.Context())
	for round := 0; round < 16; round++ {
		redirected := false
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		for depth := 0; depth <= len(parts); depth++ {
			if depth > 0 && parts[depth-1] == "" {
				break
			}
			prefix := "/" + strings.Join(parts[:depth], "/")
			dir := filepath.Join(root, filepath.FromSlash(prefix))
			st, err := os.Stat(dir)
			if err != nil || !st.IsDir() {
				break
			}
			real, err := filepath.EvalSymlinks(dir)
			rel, re := filepath.Rel(root, real)
			if err != nil || re != nil || (rel != "." && !filepath.IsLocal(rel)) {
				return r, 404
			}
			policy := c.get(dir, st)
			if policy.err != nil {
				fmt.Fprintln(os.Stderr, "PHP access policy:", dir, policy.err)
				return r, 500
			}
			for _, rule := range policy.rules {
				name := strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, strings.TrimSuffix(prefix, "/")), "/")
				matches := rule.pattern.FindStringSubmatch(name)
				if matches == nil {
					continue
				}
				filename := filepath.Join(root, filepath.FromSlash(r.URL.Path))
				info, statErr := os.Stat(filename)
				pass := true
				for _, cond := range rule.conditions {
					found := statErr == nil && ((strings.HasSuffix(cond, "f") && info.Mode().IsRegular()) || (strings.HasSuffix(cond, "d") && info.IsDir()))
					if strings.HasPrefix(cond, "!") {
						found = !found
					}
					pass = pass && found
				}
				if !pass {
					continue
				}
				if rule.flags["F"] {
					return r, 403
				}
				if rule.flags["G"] {
					return r, 410
				}
				if rule.target == "-" {
					if rule.flags["END"] {
						return r, 0
					}
					if rule.flags["L"] {
						break
					}
					continue
				}
				target := rule.target
				// Apache B escapes backreferences before query-string splitting. Never
				// allow a captured ampersand or question mark to inject new parameters.
				for n := 9; n >= 0; n-- {
					value := ""
					if n < len(matches) {
						value = matches[n]
					}
					if rule.flags["B"] {
						value = url.QueryEscape(value)
					}
					target = strings.ReplaceAll(target, fmt.Sprintf("$%d", n), value)
				}
				targetPath, query, hasQuery := strings.Cut(target, "?")
				decoded, err := url.PathUnescape(targetPath)
				if err != nil {
					return r, 500
				}
				if !strings.HasPrefix(decoded, "/") {
					decoded = path.Join(prefix, decoded)
				}
				if strings.Contains(decoded, "\\") {
					return r, 404
				}
				r.URL.Path = path.Clean(decoded)
				r.URL.RawPath = ""
				for _, part := range strings.Split(r.URL.Path, "/") {
					if strings.HasPrefix(part, ".") {
						return r, 404
					}
				}
				if hasQuery {
					if rule.flags["QSA"] && r.URL.RawQuery != "" {
						query += "&" + r.URL.RawQuery
					}
					r.URL.RawQuery = query
				}
				redirected = true
				if rule.flags["END"] {
					return r, 0
				}
				if rule.flags["L"] {
					break
				}
			}
			if redirected {
				break
			}
		}
		if !redirected {
			return r, 0
		}
	}
	return r, 508
}
