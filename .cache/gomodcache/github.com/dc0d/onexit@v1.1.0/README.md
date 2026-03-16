[![GoDoc](https://godoc.org/github.com/dc0d/onexit?status.svg)](https://godoc.org/github.com/dc0d/onexit)

# onexit
Package helps with running functions on app exit (receiving an OS signal), based on their priority.

Functions will be registtered with a priority and be called based on that priority - a priority queue is implemented by employing a heap.

The last statement of `main()` can be `<-onexit.Done()` which waits for all registered functions to run, before exit.


```go
package main

import (
	"github.com/dc0d/onexit"
)

func main() {
	onexit.Register(Logout, 100)
	onexit.Register(func() { println("\n") })

	// ...

	<-onexit.Done()
}

func init() {
	onexit.Register(SyncLogger, -100)
}
```

Calling `os.Exit(code)` explicitly, will not trigger `onexit` to run registered functions. Because it causes the program to exit without waiting for anything. Instead, call `onexit.ForceExit(code)` which waits for all registered functions to execute and then calls `os.Exit(code)`.