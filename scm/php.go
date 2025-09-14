/*
Copyright (C) 2023  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package scm

import (
    "fmt"
    "net/http"

    "github.com/dunglas/frankenphp"
    // import your ext_memcp package
    // _ "yourproject/ext_memcp"
)

// ext_memcp.go
//
// This file sets up a FrankenPHP extension that defines the
// PHP function memcp_query($sql, $params).
//
// You’ll need a corresponding stub (.stub.php) / arginfo to register it properly.

import "C"
import (
    "unsafe"
)

func init() {
    // Register the PHP extension at startup
    frankenphp.RegisterExtension(unsafe.Pointer(&C.ext_module_entry))
}

//export go_memcp_query
func go_memcp_query(sqlPtr *C.char, paramsPtr unsafe.Pointer) {
    // Here you need to convert C strings or PHP zval args into
    // Go types, run memcp logic, then return a PHP-compatible value.
    sql := C.GoString(sqlPtr)
    goParams := phpArrayToSlice(params)
    // Convert paramsPtr (which is PHP zval/array) into a Go slice or map...
    fmt.Println("memcp_query called with:", sql, goParams)
    // TODO: call memcp driver in Go, do query, prepare, return result
}

func phpArrayToSlice(arr *C.zval) []Scmer {
    var result []Scmer

    // Iterate PHP array using Zend API
    ht := (*C.HashTable)(unsafe.Pointer(arr.value.arr))
    var pos C.HashPosition
    var key *C.zend_string
    var val *C.zval

    C.zend_hash_internal_pointer_reset_ex(ht, &pos)
    for {
        if C.zend_hash_get_current_data_ex(ht, &val, &pos) == C.FAILURE {
            break
        }

        switch val.u1.vtype {
        case C.IS_TRUE, C.IS_FALSE:
            result = append(result, val.value.lval != 0)
        case C.IS_LONG:
            result = append(result, int64(val.value.lval))
        case C.IS_DOUBLE:
            result = append(result, float64(val.value.dval))
        case C.IS_STRING:
            result = append(result, C.GoString(val.value.str.val))
        default:
            // ignore or throw error for unsupported types
	    str := C.zval_get_string(val)
	    result = append(result, C.GoString(str.val))
	    C.zend_string_release(str)
        }

        C.zend_hash_move_forward_ex(ht, &pos)
    }
    return result
}


func main() {
    // Initialize FrankenPHP (you might set options here)
    if err := frankenphp.Init(); err != nil {
        panic(err)
    }

    // Create a new HTTP server
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        // Wrap the request into a FrankenPHP request
        fr := frankenphp.NewRequestWithContext(r)

        // Optionally set $_SERVER, $_GET, $_POST, etc. in fr
        // e.g. fr.SetServerVar("REQUEST_URI", r.RequestURI) ...

        // Serve PHP via FrankenPHP
        if err := frankenphp.ServeHTTP(w, fr); err != nil {
            http.Error(w, "PHP Error: "+err.Error(), http.StatusInternalServerError)
        }
    })

    fmt.Println("Listening on :8080")
    http.ListenAndServe(":8080", nil)
}

