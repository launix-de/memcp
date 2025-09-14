package main

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>
#include <stdio.h>
#include <unistd.h>

// minimal forward decls to avoid requiring php headers
typedef struct _zval_struct zval;
typedef int (*php_embed_init_t)(int argc, char **argv);
typedef void (*php_embed_shutdown_t)(void);
typedef int (*php_request_startup_t)(void);
typedef void (*php_request_shutdown_t)(void*);
typedef int (*zend_eval_stringl_t)(const char* str, size_t str_len, zval* retval_ptr, const char* string_name);

static void* dlopen_any(const char* name) {
    return dlopen(name, RTLD_NOW|RTLD_GLOBAL);
}

static void* get_symbol(void* h, const char* name) {
    return dlsym(h, name);
}

static int call_php_embed_init(php_embed_init_t fn, int argc, char **argv) { return fn(argc, argv); }
static void call_php_embed_shutdown(php_embed_shutdown_t fn) { fn(); }
static int call_php_request_startup(php_request_startup_t fn) { return fn(); }
static void call_php_request_shutdown(php_request_shutdown_t fn) { fn(NULL); }
static int call_zend_eval_stringl(zend_eval_stringl_t fn, const char* s, size_t len, const char* name) { return fn(s, len, NULL, name); }

// capture both stdout and stderr into a single pipe
static int capture_begin2(int* saved_stdout, int* saved_stderr, int* rfd) {
    int fds[2];
    if (pipe(fds) != 0) return -1;
    fflush(stdout);
    fflush(stderr);
    *saved_stdout = dup(1);
    *saved_stderr = dup(2);
    dup2(fds[1], 1);
    dup2(fds[1], 2);
    close(fds[1]);
    *rfd = fds[0];
    return 0;
}
static void capture_end2(int saved_stdout, int saved_stderr) {
    fflush(stdout);
    fflush(stderr);
    dup2(saved_stdout, 1);
    dup2(saved_stderr, 2);
    close(saved_stdout);
    close(saved_stderr);
}
*/
import "C"

import (
    "encoding/base64"
    "errors"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "unsafe"

    "github.com/launix-de/memcp/scm"
)

// Embedded PHP engine loader (dlopen based)
type phpEngine struct {
    handle              unsafe.Pointer
    embedInit           C.php_embed_init_t
    embedShutdown       C.php_embed_shutdown_t
    requestStartup      C.php_request_startup_t
    requestShutdown     C.php_request_shutdown_t
    zendEvalStringl     C.zend_eval_stringl_t
    initialized         bool
}

var (
    phpEng phpEngine
    phpMu  sync.Mutex
)

func loadLibphp() error {
    if phpEng.handle != nil { return nil }
    candidates := []string{
        os.Getenv("MEMCP_PHP_LIB"),
        "libphp8.3.so",
        "libphp8.2.so",
        "libphp.so",
    }
    var h unsafe.Pointer
    for _, name := range candidates {
        if name == "" { continue }
        c := C.CString(name)
        h = C.dlopen_any(c)
        C.free(unsafe.Pointer(c))
        if h != nil { break }
    }
    if h == nil { return errors.New("cannot dlopen libphp (set MEMCP_PHP_LIB to full path)") }

    phpEng.handle = h
    sym := func(n string) unsafe.Pointer {
        cs := C.CString(n)
        p := C.get_symbol(h, cs)
        C.free(unsafe.Pointer(cs))
        return p
    }
    phpEng.embedInit = (C.php_embed_init_t)(sym("php_embed_init"))
    phpEng.embedShutdown = (C.php_embed_shutdown_t)(sym("php_embed_shutdown"))
    phpEng.requestStartup = (C.php_request_startup_t)(sym("php_request_startup"))
    phpEng.requestShutdown = (C.php_request_shutdown_t)(sym("php_request_shutdown"))
    phpEng.zendEvalStringl = (C.zend_eval_stringl_t)(sym("zend_eval_stringl"))

    if phpEng.embedInit == nil || phpEng.embedShutdown == nil || phpEng.requestStartup == nil || phpEng.requestShutdown == nil || phpEng.zendEvalStringl == nil {
        return errors.New("missing PHP symbols in libphp")
    }
    return nil
}

func ensurePHPInit() error {
    if phpEng.initialized { return nil }
    if err := loadLibphp(); err != nil { return err }
    argv := []*C.char{C.CString("memcp-php-embed")}
    defer C.free(unsafe.Pointer(argv[0]))
    rc := int(C.call_php_embed_init(phpEng.embedInit, 1, &argv[0]))
    if rc != 0 { return fmt.Errorf("php_embed_init failed rc=%d", rc) }
    phpEng.initialized = true
    return nil
}

func runPHP(docroot, script string, req *http.Request) (string, error) {
    if err := ensurePHPInit(); err != nil { return "", err }
    // change working directory to docroot for relative includes
    oldwd, _ := os.Getwd()
    _ = os.Chdir(docroot)
    defer func() { _ = os.Chdir(oldwd) }()
    // Prepare server globals in PHP via eval; capture stdout
    // Build include path
    if !filepath.IsAbs(script) {
        script = filepath.Join(docroot, script)
    }
    // choose index.php for directories
    fi, err := os.Stat(script)
    if err == nil && fi.IsDir() {
        script = filepath.Join(script, "index.php")
    }
    // body (base64 encode)
    var bodyB64 string
    if req.Body != nil {
        b, _ := io.ReadAll(req.Body)
        req.Body.Close()
        if len(b) > 0 {
            bodyB64 = base64.StdEncoding.EncodeToString(b)
        }
    }
    // Minimal server/setup; header() not propagated; body handled
    // NOTE: proper header handling requires hooking SAPI header funcs.
    q := req.URL.RawQuery
    method := req.Method
    uri := req.URL.RequestURI()
    sn := req.URL.Path
    host := req.Host
    var b strings.Builder
    b.WriteString("<?php\n")
    b.WriteString("error_reporting(E_ALL); ini_set('display_errors','1');\\n")
    b.WriteString("$_SERVER['DOCUMENT_ROOT'] = '")
    b.WriteString(phpEscape(docroot))
    b.WriteString("';\n")
    b.WriteString("$_SERVER['SCRIPT_FILENAME'] = '")
    b.WriteString(phpEscape(script))
    b.WriteString("';\n")
    b.WriteString("$_SERVER['SCRIPT_NAME'] = '")
    b.WriteString(phpEscape(sn))
    b.WriteString("';\n")
    b.WriteString("$_SERVER['REQUEST_METHOD'] = '")
    b.WriteString(phpEscape(method))
    b.WriteString("';\n")
    b.WriteString("$_SERVER['REQUEST_URI'] = '")
    b.WriteString(phpEscape(uri))
    b.WriteString("';\n")
    b.WriteString("$_SERVER['QUERY_STRING'] = '")
    b.WriteString(phpEscape(q))
    b.WriteString("';\n")
    b.WriteString("$_SERVER['HTTP_HOST'] = '")
    b.WriteString(phpEscape(host))
    b.WriteString("';\n")
    b.WriteString("parse_str($_SERVER['QUERY_STRING'], $_GET);\n")
    b.WriteString("$_POST = array();\n")
    ct := req.Header.Get("Content-Type")
    if ct != "" {
        b.WriteString("$_SERVER['CONTENT_TYPE'] = '")
        b.WriteString(phpEscape(ct))
        b.WriteString("';\n")
    }
    if bodyB64 != "" {
        b.WriteString("$_MEMCP_BODY = base64_decode('")
        b.WriteString(bodyB64)
        b.WriteString("');\n")
    }
    b.WriteString("if ($_SERVER['REQUEST_METHOD']==='POST' && isset($_MEMCP_BODY)) {\n")
    b.WriteString("  if (isset($_SERVER['CONTENT_TYPE']) && stripos($_SERVER['CONTENT_TYPE'], 'application/x-www-form-urlencoded')!==false) {\n")
    b.WriteString("    parse_str($_MEMCP_BODY, $_POST);\n")
    b.WriteString("  }\n")
    b.WriteString("}\n")
    // populate selected HTTP_* headers
    // (set common ones from Go; others can be added as needed)
    b.WriteString("if (!isset($_SERVER['SERVER_PROTOCOL'])) $_SERVER['SERVER_PROTOCOL']='HTTP/1.1';\\n")
    b.WriteString("if (!isset($_SERVER['SERVER_NAME'])) $_SERVER['SERVER_NAME']=$_SERVER['HTTP_HOST'];\\n")
    b.WriteString("include $_SERVER['SCRIPT_FILENAME'];\n")
    b.WriteString("?>")
    phpCode := b.String()

    // Surround eval with php_request_startup/shutdown and capture stdout
    var savedOut, savedErr, rfd C.int
    if C.capture_begin2(&savedOut, &savedErr, &rfd) != 0 { return "", errors.New("failed to capture output") }
    // startup
    if int(C.call_php_request_startup(phpEng.requestStartup)) != 0 {
        C.capture_end2(savedOut, savedErr)
        return "", errors.New("php_request_startup failed")
    }
    cstr := C.CString(phpCode)
    cname := C.CString("memcp-server")
    // eval
    _ = C.call_zend_eval_stringl(phpEng.zendEvalStringl, cstr, C.size_t(len(phpCode)), cname)
    C.free(unsafe.Pointer(cname))
    C.free(unsafe.Pointer(cstr))
    // shutdown request and flush any buffered stdout to the pipe
    C.call_php_request_shutdown(phpEng.requestShutdown)
    C.fflush(C.stdout)
    C.fflush(C.stderr)
    // restore stdio and close pipe write end so reader gets EOF
    C.capture_end2(savedOut, savedErr)
    // read from the pipe (now writer closed -> EOF)
    f := os.NewFile(uintptr(rfd), "php-stdio")
    out, _ := io.ReadAll(f)
    f.Close()
    return string(out), nil
}

func phpEscape(s string) string {
    // simplest escape for inclusion in single-quoted PHP string
    s = strings.ReplaceAll(s, "\\", "\\\\")
    s = strings.ReplaceAll(s, "'", "\\'")
    return s
}

// Register the servePHP factory on plugin load
func init() {
    scm.Declare(&scm.Globalenv, &scm.Declaration{
        "servePHP",
        "serves PHP using embedded libphp; returns a handler lambda(req res)",
        1, 1,
        []scm.DeclarationParameter{
            scm.DeclarationParameter{"path", "string", "docroot or entry file (absolute or relative)"},
        }, "func",
        func(a ...scm.Scmer) scm.Scmer {
            target := scm.String(a[0])
            return func(a ...scm.Scmer) scm.Scmer { // req res
                req := a[0].([]scm.Scmer)[1].(*http.Request)
                res := a[1].([]scm.Scmer)[1].(http.ResponseWriter)
                // allow downstream to set Content-Type (network.go sets text/plain by default)
                res.Header().Del("Content-Type")
                // Resolve path at request time
                docroot := target
                if !filepath.IsAbs(docroot) {
                    cwd, _ := os.Getwd()
                    docroot = filepath.Join(cwd, docroot)
                }
                // Decide whether to serve static file or PHP script
                rel := strings.TrimPrefix(req.URL.Path, "/")
                if rel == "" { rel = "index.php" }
                full := filepath.Join(docroot, filepath.FromSlash(rel))
                if fi, err := os.Stat(full); err == nil {
                    if fi.IsDir() {
                        // try directory index
                        idx := filepath.Join(full, "index.php")
                        if _, err := os.Stat(idx); err == nil {
                            rel = strings.TrimPrefix(idx[len(docroot):], string(filepath.Separator))
                        } else {
                            http.NotFound(res, req)
                            return nil
                        }
                    } else {
                        // file exists
                        if strings.EqualFold(strings.ToLower(filepath.Ext(full)), ".php") {
                            // run php
                        } else {
                            // serve static
                            http.ServeFile(res, req, full)
                            return nil
                        }
                    }
                } else {
                    // no file: try front controller index.php
                    idx := filepath.Join(docroot, "index.php")
                    if _, err := os.Stat(idx); err == nil {
                        rel = "index.php"
                    } else {
                        http.NotFound(res, req)
                        return nil
                    }
                }

                phpMu.Lock()
                defer phpMu.Unlock()
                out, err := runPHP(docroot, rel, req)
                if err != nil {
                    res.Header().Set("Content-Type", "text/plain")
                    res.WriteHeader(http.StatusInternalServerError)
                    io.WriteString(res, "PHP embed error: ")
                    io.WriteString(res, err.Error())
                    return nil
                }
                // if PHP did not send headers, default to HTML
                if res.Header().Get("Content-Type") == "" {
                    res.Header().Set("Content-Type", "text/html; charset=utf-8")
                }
                res.WriteHeader(http.StatusOK)
                io.WriteString(res, out)
                return nil
            }
        }, false,
    })
}
