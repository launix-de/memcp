//go:build php

// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later

package phpbridge

// #include "bridge.h"
import "C"

import "io"
import "os"
import "fmt"
import "mime"
import "sync"
import "time"
import "strings"
import "runtime/cgo"
import "sync/atomic"
import "path/filepath"
import "encoding/binary"
import "golang.org/x/text/encoding"
import "github.com/leonelquinteros/gotext"
import "github.com/launix-de/memcp/storage"
import "golang.org/x/text/encoding/htmlindex"

type textCatalog struct {
	estimated int64
	path      string
	stamp     time.Time
	size      int64
	mo        *gotext.Mo
	charset   encoding.Encoding
	last      atomic.Int64
}

var textCatalogCache = struct {
	sync.Mutex
	entries map[string]*textCatalog
}{entries: make(map[string]*textCatalog)}

type textRequest struct {
	catalogBytes int64
	catalogs     map[string][]*textCatalog
	imap         *imapWorker
}

//export memcp_text_request_start
func memcp_text_request_start() C.uintptr_t {
	return C.uintptr_t(cgo.NewHandle(&textRequest{catalogs: make(map[string][]*textCatalog)}))
}

//export memcp_text_request_end
func memcp_text_request_end(handle C.uintptr_t) {
	h := cgo.Handle(handle)
	r := h.Value().(*textRequest)
	if r.imap != nil {
		r.imap.close()
	}
	h.Delete()
}

func validateMO(data []byte) (int64, error) {
	if len(data) < 28 {
		return 0, fmt.Errorf("truncated gettext catalog")
	}
	var order binary.ByteOrder
	switch binary.LittleEndian.Uint32(data) {
	case 0x950412de:
		order = binary.LittleEndian
	case 0xde120495:
		order = binary.BigEndian
	default:
		return 0, fmt.Errorf("invalid gettext catalog magic")
	}
	count := uint64(order.Uint32(data[8:]))
	estimated := count*256 + uint64(len(data)) + 1024
	for _, header := range []int{12, 16} {
		offset := uint64(order.Uint32(data[header:]))
		if offset+count*8 > uint64(len(data)) {
			return 0, fmt.Errorf("invalid gettext string table")
		}
		for i := uint64(0); i < count; i++ {
			entry := data[offset+i*8:]
			n, start := uint64(order.Uint32(entry)), uint64(order.Uint32(entry[4:]))
			estimated += n * 4
			if estimated > 64<<20 {
				return 0, fmt.Errorf("gettext catalog expands beyond 64 MiB")
			}
			if start+n > uint64(len(data)) {
				return 0, fmt.Errorf("invalid gettext string range")
			}
		}
	}
	return int64(estimated), nil
}
func loadTextCatalog(path string) (*textCatalog, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	textCatalogCache.Lock()
	old := textCatalogCache.entries[path]
	textCatalogCache.Unlock()
	if old != nil && old.stamp.Equal(info.ModTime()) && old.size == info.Size() {
		old.last.Store(time.Now().UnixNano())
		return old, nil
	}
	if info.Size() > 16<<20 {
		return nil, fmt.Errorf("gettext catalog exceeds 16 MiB")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (16<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 16<<20 {
		return nil, fmt.Errorf("gettext catalog exceeds 16 MiB")
	}
	estimate, err := validateMO(data)
	if err != nil {
		return nil, err
	}
	catalog := &textCatalog{estimated: estimate, path: path, stamp: info.ModTime(), size: info.Size(), mo: gotext.NewMo()}
	catalog.mo.Parse(data)
	catalog.last.Store(time.Now().UnixNano())
	if header := catalog.mo.Headers.Get("Content-Type"); header != "" {
		_, params, err := mime.ParseMediaType(header)
		if err != nil {
			return nil, err
		}
		if charset := params["charset"]; charset != "" && !strings.EqualFold(charset, "UTF-8") {
			catalog.charset, err = htmlindex.Get(charset)
			if err != nil {
				return nil, err
			}
		}
	}
	textCatalogCache.Lock()
	old = textCatalogCache.entries[path]
	textCatalogCache.entries[path] = catalog
	textCatalogCache.Unlock()
	if old != nil {
		storage.GlobalCache.Remove(old)
	}
	storage.GlobalCache.AddItem(catalog, estimate, storage.TypeCacheEntry, func(ptr any, _ *storage.CacheFreedBytes) bool {
		entry := ptr.(*textCatalog)
		if !textCatalogCache.TryLock() {
			return false
		}
		defer textCatalogCache.Unlock()
		if textCatalogCache.entries[entry.path] == entry {
			delete(textCatalogCache.entries, entry.path)
		}
		return true
	}, func(ptr any) time.Time { return time.Unix(0, ptr.(*textCatalog).last.Load()) }, nil)
	return catalog, nil
}
func textLanguages(locale, language string) []string {
	if locale == "C" || locale == "POSIX" {
		return nil
	}
	if language == "" {
		language = locale
	}
	var result []string
	for _, raw := range strings.Split(language, ":") {
		if raw == "C" || raw == "POSIX" {
			break
		}
		if raw == "" || strings.ContainsAny(raw, "/\\\x00") || raw == ".." {
			continue
		}
		result = append(result, raw)
		name := strings.SplitN(raw, ".", 2)[0]
		if name != raw {
			result = append(result, name)
		}
		short := strings.SplitN(name, "_", 2)[0]
		if short != name {
			result = append(result, short)
		}
	}
	return result
}
func (r *textRequest) translate(directory, domain, locale, language, codeset, message, plural string, n int, hasPlural bool, category string) (string, error) {
	if len(locale) > 256 || len(language) > 4096 {
		return "", fmt.Errorf("gettext locale preference is too long")
	}
	fallback := message
	if hasPlural && n != 1 {
		fallback = plural
	}
	if strings.ContainsAny(domain, "/\\\x00") || domain == ".." {
		return "", fmt.Errorf("invalid gettext domain")
	}
	key := directory + "\x00" + domain + "\x00" + locale + "\x00" + language + "\x00" + category
	catalogs, loaded := r.catalogs[key]
	if !loaded {
		var bytes int64
		for _, lang := range textLanguages(locale, language) {
			catalog, err := loadTextCatalog(filepath.Join(directory, lang, category, domain+".mo"))
			if err != nil {
				return "", err
			}
			if catalog != nil {
				bytes += catalog.estimated
				if bytes > 64<<20 {
					return "", fmt.Errorf("gettext lookup exceeds 64 MiB catalog budget")
				}
				catalogs = append(catalogs, catalog)
			}
		}
		if len(r.catalogs) < 256 && r.catalogBytes+bytes <= 64<<20 {
			r.catalogs[key] = catalogs
			r.catalogBytes += bytes
		}
	}
	for _, catalog := range catalogs {
		key, pluralKey := message, plural
		if catalog.charset != nil {
			var err error
			key, err = catalog.charset.NewEncoder().String(message)
			if err != nil {
				continue
			}
			pluralKey, err = catalog.charset.NewEncoder().String(plural)
			if err != nil {
				continue
			}
		}
		if !catalog.mo.IsTranslated(key) {
			continue
		}
		// PHP gettext returns raw translations, including percent placeholders.
		out := catalog.mo.Get(key, []any(nil)...)
		if hasPlural {
			out = catalog.mo.GetN(key, pluralKey, n)
		}
		if catalog.charset != nil {
			var err error
			out, err = catalog.charset.NewDecoder().String(out)
			if err != nil {
				return "", err
			}
		}
		if !strings.EqualFold(codeset, "UTF-8") && !strings.EqualFold(codeset, "UTF8") {
			encoding, err := htmlindex.Get(codeset)
			if err != nil {
				return "", err
			}
			return encoding.NewEncoder().String(out)
		}
		return out, nil
	}
	return fallback, nil
}

//export memcp_translate
func memcp_translate(handle C.uintptr_t, directory, domain, locale, language, codeset, message *C.char, messageLen C.size_t, plural *C.char, pluralLen C.size_t, n C.longlong, hasPlural C.int, category *C.char) (result C.memcp_text) {
	defer func() {
		if err := recover(); err != nil {
			result.error = C.CString(fmt.Sprint(err))
		}
	}()
	r := cgo.Handle(handle).Value().(*textRequest)
	out, err := r.translate(C.GoString(directory), C.GoString(domain), C.GoString(locale), C.GoString(language), C.GoString(codeset), C.GoStringN(message, C.int(messageLen)), C.GoStringN(plural, C.int(pluralLen)), int(n), hasPlural != 0, C.GoString(category))
	if err != nil {
		result.error = C.CString(err.Error())
		return
	}
	result.data = (*C.char)(C.CBytes([]byte(out)))
	result.length = C.size_t(len(out))
	return
}
