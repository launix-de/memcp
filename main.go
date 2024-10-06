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
import "time"
import "bufio"
import "sync"
import "syscall"
import "runtime"
import "io/ioutil"
import "os/signal"
import "crypto/rand"
import "path/filepath"
import "runtime/pprof"
import "github.com/google/uuid"
import "github.com/fsnotify/fsnotify"
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
					"watch": getWatch(wd),
					"serveStatic": scm.HTTPStaticGetter(wd),
				},
				nil,
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
					scm.Apply(a[1], str);
				}
			} else {
				// read in whole
				bytes, err := ioutil.ReadFile(filename)
				if err != nil {
					panic(err)
				}
				if len(a) > 1 {
					scm.Apply(a[1], string(bytes));
				} else {
					return string(bytes)
				}
			}
			return true
		}
}

func getWatch(path string) func (a ...scm.Scmer) scm.Scmer {
	return func (a ...scm.Scmer) scm.Scmer {
		filename := path + "/" + scm.String(a[0])
		reread := func () {
			// read in whole
			bytes, err := ioutil.ReadFile(filename)
			if err != nil {
				panic(err)
			}
			scm.Apply(a[1], string(bytes))
		}
		reread() // read once at the beginning in sync
		// watch for changes
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			panic(err)
		}
		go func() {
			for {
				select {
				case /*event :=*/ <- watcher.Events:
					// flush all other events
					for {
						time.Sleep(10 * time.Millisecond) // delay a bit, so we don't read empty files
						select {
						case <- watcher.Events:
							// ignore
						default:
							goto to_reread
						}
					}
					to_reread:
					// now reread the file
					func () {
						defer func() {
							if err := recover(); err != nil {
								// error happens during reload: log to console
								fmt.Println(err)
							}
						}()
						reread()
					}()
					watcher.Add(filename) // text editors rename, so we have to rewatch
				}
			}
		}()
		err = watcher.Add(filename)
		if err != nil {
			panic(err)
		}
		return true
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

func setupIO(wd string) {
	// define some IO functions (scm will not provide them since it is sandboxable)
	IOEnv = scm.Env {
		scm.Vars {},
		nil,
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
		"env", "returns the content of a environment variable",
		1, 2,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"var", "string", "envvar"},
			scm.DeclarationParameter{"default", "string", "default if the env is not found"},
		}, "string",
		func (a ...scm.Scmer) scm.Scmer {
			if len(a) > 1 {
				if val, ok := os.LookupEnv(scm.String(a[0])); ok {
					return val
				} else {
					return a[1]
				}
			} else {
				return os.Getenv(scm.String(a[0]))
			}
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
		}, "string|bool",
		(func(...scm.Scmer) scm.Scmer)(getLoad(wd)),
	})
	scm.Declare(&IOEnv, &scm.Declaration{
		"watch", "Loads a file and calls the callback. Whenever the file changes on disk, the file is load again.",
		2, 2,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"filename", "string", "filename relative to folder of source file"},
			scm.DeclarationParameter{"updatehandler", "func", "handler that receives the file content func(content)"},
		}, "bool",
		(func(...scm.Scmer) scm.Scmer)(getWatch(wd)),
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
		"serveStatic", "creates a static handler for use as a callback in (serve) - returns a handler lambda(req res)",
		1, 1,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"directory", "string", "folder with the files to serve"},
		}, "func",
		(func(...scm.Scmer) scm.Scmer)(scm.HTTPStaticGetter(wd)),
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
		"password", "Hashes a password with sha1 (for mysql user authentication)",
		1, 1,
		[]scm.DeclarationParameter{
			scm.DeclarationParameter{"password", "string", "plain text password to hash"},
		}, "string",
		scm.MySQLPassword,
	})
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

	profile := ""
	flag.StringVar(&profile, "profile", "", "Data folder for persistence")

	wd, _ := os.Getwd() // libraries are relative to working directory... or change with -wd PATH
	flag.StringVar(&wd, "wd", wd, "Working Directory for (import) and (load) (Default: .)")

	flag.Parse()
	imports := flag.Args()

	// storage initialization
	setupIO(wd)
	storage.Init(scm.Globalenv)
	storage.Basepath = basepath
	storage.LoadDatabases()
	// scripts initialization
	if len(imports) == 0 {
		// load default script
		IOEnv.Vars["import"].(func(...scm.Scmer)scm.Scmer)("lib/main.scm")
	} else {
		// load scripts from command line
		for _, scmfile := range imports {
			fmt.Println("Loading " + scmfile + " ...")
			IOEnv.Vars["import"].(func(...scm.Scmer)scm.Scmer)(scmfile)
		}
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
	// init profiling
	if profile != "" {
		f, err := os.Create(profile)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// start cron
	go cronroutine()

	// REPL shell
	scm.Repl(&IOEnv)

	// normal shutdown
	exitroutine()
}

var exitsignal chan bool = make(chan bool, 1) // set true to start shutdown routine and wait for all jobs
var exitable sync.WaitGroup
func cronroutine() {
	exitable.Add(1)
	for {
		// wait first
		select {
			case <- exitsignal:
				// memcp is about to exit; confirm the waitgroup and exit
				exitable.Done()
				return
			case <- time.After(time.Minute * 15): // rebuild shards for all 15 minutes
				// continue
		}

		fmt.Println("running 15min cron ...")
		fmt.Println("table compression done in ", storage.Rebuild(false, true))
	}
}

func exitroutine() {
	exitsignal <- true
	exitable.Wait()
	fmt.Println("Exit procedure...")
	if scm.ReplInstance != nil {
		// in case it dosen't exit properly
		scm.ReplInstance.Close()
	}
	fmt.Println("finalizing storage...")
	storage.UnloadDatabases()
	fmt.Println("finalizing memory...")
	runtime.GC() // this will call the finalizers on shards
	fmt.Println("Exit procedure finished")
}
