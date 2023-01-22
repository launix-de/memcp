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
/*
	cpdb smart clusterable distributed database working best on nvme

	https://pkelchte.wordpress.com/2013/12/31/scm-go/

*/
package main

import "fmt"
import "os"
import "io"
import "time"
import "io/ioutil"
import "path/filepath"
import "net/http"
import "github.com/launix-de/cpdb/scm"
import "github.com/launix-de/cpdb/storage"
import "github.com/lrita/numa"

var IOEnv scm.Env

func getImport(path string) func (a ...scm.Scmer) scm.Scmer {
	return func (a ...scm.Scmer) scm.Scmer {
			filename := path + "/" + a[0].(string)
			// TODO: filepath.Walk for wildcards
			otherPath := scm.Env {
				scm.Vars {
					"__DIR__": path,
					"__FILE__": filename,
					"import": getImport(filepath.Dir(filename)),
				},
				&IOEnv,
				true,
			}
			bytes, err := ioutil.ReadFile(filename)
			if err != nil {
				panic(err)
			}
			return scm.EvalAll(string(bytes), &otherPath)
		}
}

func httpHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/plain")
	io.WriteString(res, "Method = ")
	io.WriteString(res, req.Method)
	io.WriteString(res, "\n")
	io.WriteString(res, "Path = ")
	io.WriteString(res, req.URL.Path) // or: RawPath
	io.WriteString(res, "\n")
	io.WriteString(res, "Params = ")
	io.WriteString(res, fmt.Sprint(req.URL.Query())) // map[string][]string
	io.WriteString(res, "\n")
	io.WriteString(res, "Header = ")
	io.WriteString(res, fmt.Sprint(req.Header))
	io.WriteString(res, "\n")
	// TODO: req.Body io.ReadCloser
	// req.URL.User.Username()
	// req.URL.User.Password()
	// req.ContentLength == -1 or >= 0
	// req.Host -> multiple hostnames according to DNS system
	// req.RemoteAddr -> IP
}

func main() {
	fmt.Print(`cpdb Copyright (C) 2023   Carl-Philip Hänsch
    This program comes with ABSOLUTELY NO WARRANTY;
    This is free software, and you are welcome to redistribute it
    under certain conditions;
`)
	// print some info
	fmt.Println("NUMA support:", numa.Available())

	// define some IO functions (scm will not provide them since it is sandboxable)
	wd, _ := os.Getwd() // libraries are relative to working directory... is that right?
	IOEnv = scm.Env {
		scm.Vars {
			"print": func (a ...scm.Scmer) scm.Scmer {
					for _, s := range a {
						fmt.Print(scm.String(s))
					}
					fmt.Println()
					return "ok"
				},
			"import": getImport(wd),
		},
		&scm.Globalenv,
		true, // other defines go into Globalenv
	}
	// storage initialization
	storage.Init(scm.Globalenv)
	// scripts initialization
	scm.Eval(scm.Read("(import \"lib/main.scm\")"), &IOEnv)

	// HTTP endpoint (TODO: move into scheme helper library)
	server := &http.Server {
		Addr: ":4321",
		Handler: http.HandlerFunc(httpHandler),
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go server.ListenAndServe()
	fmt.Println("HTTP server active on http://localhost:4321")
	// TODO: ListenAndServeTLS

	// REPL shell
	scm.Repl(&IOEnv)
}
