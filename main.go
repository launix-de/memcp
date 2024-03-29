/*
Copyright (C) 2023, 2024  Carl-Philip Hänsch

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
	memcp smart clusterable distributed database working best on nvme

	https://pkelchte.wordpress.com/2013/12/31/scm-go/

*/
package main

import "os"
import "io"
import "fmt"
import "flag"
import "bufio"
import "syscall"
import "runtime"
import "io/ioutil"
import "os/signal"
import "crypto/rand"
import "path/filepath"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage"

var IOEnv scm.Env

func getImport(path string) func (a ...scm.Scmer) scm.Scmer {
	return func (a ...scm.Scmer) scm.Scmer {
			filename := path + "/" + scm.String(a[0])
			// TODO: filepath.Walk for wildcards
			wd := filepath.Dir(filename)
			otherPath := scm.Env {
				scm.Vars {
					"__DIR__": path,
					"__FILE__": filename,
					"import": getImport(wd),
					"load": getLoad(wd),
				},
				&IOEnv,
				true,
			}
			bytes, err := ioutil.ReadFile(filename)
			if err != nil {
				panic(err)
			}
			return scm.EvalAll(filename, string(bytes), &otherPath)
		}
}

func getLoad(path string) func (a ...scm.Scmer) scm.Scmer {
	return func (a ...scm.Scmer) scm.Scmer {
			filename := path + "/" + scm.String(a[0])
			if len(a) > 2 {
				file, err := os.Open(filename)
				if err != nil {
					panic(err)
				}
				splitter := bufio.NewReader(file)
				delimiter := scm.String(a[2])
				if len(delimiter) != 1 {
					panic("load delimiter must be 1 byte long")
				}
				for {
					str, err := splitter.ReadString(delimiter[0])
					if err == io.EOF {
						break // file is finished
					}
					if err != nil {
						panic(err)
					}
					// go??
					scm.Apply(a[1], []scm.Scmer{str});
				}
			} else {
				// read in whole
				bytes, err := ioutil.ReadFile(filename)
				if err != nil {
					panic(err)
				}
				if len(a) > 1 {
					scm.Apply(a[1], []scm.Scmer{string(bytes)});
				} else {
					return string(bytes)
				}
			}
			return "ok"
		}
}

// workaround for flags package to allow multiple values
type arrayFlags []string

func (i *arrayFlags) String() string {
    return "dummy"
}

func (i *arrayFlags) Set(value string) error {
    *i = append(*i, value)
    return nil
}

func main() {
	fmt.Print(`memcp Copyright (C) 2023, 2024   Carl-Philip Hänsch
    This program comes with ABSOLUTELY NO WARRANTY;
    This is free software, and you are welcome to redistribute it
    under certain conditions;

`)

	// init random generator for UUIDs
	uuid.SetRand(rand.Reader)

	// parse command line options
	var commands arrayFlags
	flag.Var(&commands, "c", "Execute scm command")
	basepath := "data"
	flag.StringVar(&basepath, "data", "data", "Data folder for persistence")
	flag.Parse()
	imports := flag.Args()

	// define some IO functions (scm will not provide them since it is sandboxable)
	wd, _ := os.Getwd() // libraries are relative to working directory... is that right?
	IOEnv = scm.Env {
		scm.Vars {},
		&scm.Globalenv,
		true, // other defines go into Globalenv
	}
	scm.DeclareTitle("IO")
	scm.Declare(&IOEnv, &scm.Declaration{
		"print", "Prints values to stdout (only in IO environment)",
		1, 1000,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"value...", "any", "values to print"},
		}, "bool",
		func (a ...scm.Scmer) scm.Scmer {
			for _, s := range a {
				fmt.Print(scm.String(s))
			}
			fmt.Println()
			return true
		},
	})
	scm.Declare(&IOEnv, &scm.Declaration{
		"help", "Lists all functions or print help for a specific function",
		0, 1,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"topic", "string", "function to print help about"},
		}, "nil",
		func (a ...scm.Scmer) scm.Scmer {
			if len(a) == 0 {
				scm.Help(nil)
			} else {
				scm.Help(a[0])
			}
			return nil
		},
	})
	scm.Declare(&IOEnv, &scm.Declaration{
		"import", "Imports a file .scm file into current namespace",
		1, 1,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"filename", "string", "filename relative to folder of source file"},
		}, "any",
		(func(...scm.Scmer) scm.Scmer)(getImport(wd)),
	})
	scm.Declare(&IOEnv, &scm.Declaration{
		"load", "Loads a file and returns the string",
		1, 3,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"filename", "string", "filename relative to folder of source file"},
			scm.DeclarationParameter{"linehandler", "func", "handler that reads each line"},
			scm.DeclarationParameter{"delimiter", "string", "delimiter to extract"},
		}, "string",
		(func(...scm.Scmer) scm.Scmer)(getLoad(wd)),
	})
	scm.Declare(&IOEnv, &scm.Declaration{
		"serve", "Opens a HTTP server at a given port",
		2, 2,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"port", "number", "port number for HTTP server"},
			scm.DeclarationParameter{"handler", "func", "handler: lambda(req res) that handles the http request (TODO: detailed documentation)"},
		}, "bool",
		scm.HTTPServe,
	})
	scm.Declare(&IOEnv, &scm.Declaration{
		"mysql", "Imports a file .scm file into current namespace",
		4, 4,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"port", "number", "port number for MySQL server"},
			scm.DeclarationParameter{"getPassword", "func", "lambda(username string) string|nil has to return the password for a user or nil to deny login"},
			scm.DeclarationParameter{"schemacallback", "func", "lambda(username schema) bool handler check whether user is allowed to schem (string) - you should check access rights here"},
			scm.DeclarationParameter{"handler", "func", "lambda(schema sql resultrow session) handler to process sql query (string) in schema (string). resultrow is a lambda(list)"},
		}, "bool",
		scm.MySQLServe,
	})
	scm.Declare(&IOEnv, &scm.Declaration{
		"newsession", "Creates a new session which is itself a function either a getter (session key) or setter (session key value)",
		0, 0,
		[]scm.DeclarationParameter{
		}, "list",
		scm.NewSession,
	})
	scm.Declare(&IOEnv, &scm.Declaration{
		"password", "Hashes a password with sha1 (for mysql user authentication)",
		1, 1,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"password", "string", "plain text password to hash"},
		}, "string",
		scm.MySQLPassword,
	})

	// storage initialization
	storage.Init(scm.Globalenv)
	storage.Basepath = basepath
	storage.LoadDatabases()
	// scripts initialization
	scm.Eval(scm.Read("init", "(import \"lib/main.scm\")"), &IOEnv)
	// command line initialization
	for _, filename := range imports {
		fmt.Println("Loading " + filename + " ...")
		scm.Eval(scm.Read("command line", "(import \"" + filename + "\")"), &IOEnv)
	}
	for _, command := range commands {
		fmt.Println("Executing " + command + " ...")
		code := scm.Read("command line", command)
		scm.Validate(code, "any")
		code = scm.Optimize(code, &IOEnv)
		scm.Eval(code, &IOEnv)
	}

	// install exit handler
	cancelChan := make(chan os.Signal, 1)
	signal.Notify(cancelChan, syscall.SIGTERM, syscall.SIGINT)
	go (func () {
		<-cancelChan
		exitroutine()
		os.Exit(1)
	})()

	fmt.Print(`

    Type (help) to show help

`)
	// REPL shell
	scm.Repl(&IOEnv)

	// normal shutdown
	exitroutine()
}

func exitroutine() {
	fmt.Println("Exit procedure... syncing to disk")
	fmt.Println("table compression done in ", scm.Globalenv.Vars["rebuild"].(func(...scm.Scmer) scm.Scmer)())
	fmt.Println("finalizing memory...")
	runtime.GC() // this will call the finalizers on shards
	fmt.Println("Exit procedure finished")
}
