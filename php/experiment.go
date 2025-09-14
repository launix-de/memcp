//go:build ignore

package main

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>
#include <stdio.h>

// Define missing macros
#ifndef RTLD_NOW
#define RTLD_NOW 2
#endif

typedef int (*php_embed_init_t)(int argc, char **argv);
typedef void (*php_embed_shutdown_t)(void);

// Trampolines to call function pointers from Go
int call_php_embed_init(php_embed_init_t fn, int argc, char **argv) {
    return fn(argc, argv);
}

void call_php_embed_shutdown(php_embed_shutdown_t fn) {
    fn();
}

// dlsym helper
static void* get_symbol(void* handle, const char* name) {
    return dlsym(handle, name);
}
*/
import "C"

// apt install libphp8.3-embed

import (
    "errors"
    "unsafe"
    "fmt"
    "os"
)

type PhpEngine struct {
    handle        unsafe.Pointer
    embedInit     C.php_embed_init_t
    embedShutdown C.php_embed_shutdown_t
}

func LoadPHP(path string) (*PhpEngine, error) {
    cpath := C.CString(path)
    defer C.free(unsafe.Pointer(cpath))

    handle := C.dlopen(cpath, C.RTLD_NOW)
    if handle == nil {
        return nil, errors.New("dlopen failed")
    }

    e := &PhpEngine{handle: handle}
    e.embedInit = (C.php_embed_init_t)(C.get_symbol(handle, C.CString("php_embed_init")))
    e.embedShutdown = (C.php_embed_shutdown_t)(C.get_symbol(handle, C.CString("php_embed_shutdown")))

    if e.embedInit == nil || e.embedShutdown == nil {
        return nil, errors.New("missing PHP symbols")
    }

    return e, nil
}

func (e *PhpEngine) Init() int {
    argv := []*C.char{C.CString("go-php-embed")}
    defer C.free(unsafe.Pointer(argv[0]))
    return int(C.call_php_embed_init(e.embedInit, 1, &argv[0]))
}

func (e *PhpEngine) Shutdown() {
    C.call_php_embed_shutdown(e.embedShutdown)
}

// Dummy memcp_query
func memcp_query(query string, params []interface{}) string {
    return fmt.Sprintf("[memcp executing] %s with params=%v", query, params)
}

func main() {
    os.Setenv("PHPRC", "/etc/php/8.3/apache2/php.ini") // embed
    phpPath := "libphp.so"
    eng, err := LoadPHP(phpPath)
    if err != nil {
        panic(err)
    }

    rc := eng.Init()
    if rc != 0 {
        panic("php_embed_init failed")
    }
    defer eng.Shutdown()

    fmt.Println("PHP engine initialized from", phpPath)
    //fmt.Println(memcp_query("SELECT 1", []interface{}{123, "abc"}))
}
