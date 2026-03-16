package onexit

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/dc0d/onexit/fnpq"
)

type deferred struct {
	pq      fnpq.PriorityQueue
	mx      sync.Mutex
	cleanup sync.Once
	done    chan struct{}
}

func newDeferred() *deferred {
	return &deferred{done: make(chan struct{})}
}

func (d *deferred) Cleanup() {
	d.cleanup.Do(func() {
		defer close(d.done)
		d.mx.Lock()
		defer d.mx.Unlock()
		for d.pq.Len() > 0 {
			item := fnpq.Pop(&d.pq)
			item.Action()
		}
	})
}

func (d *deferred) Done() <-chan struct{} {
	return d.done
}

func (d *deferred) Register(action func(), priority ...int) {
	d.mx.Lock()
	defer d.mx.Unlock()
	_priority := 0
	if len(priority) > 0 {
		_priority = priority[0]
	}
	t := fnpq.NewItem(action, _priority)
	fnpq.Push(&d.pq, t)
}

var (
	_deferred = newDeferred()
)

func init() {
	onSignal(final)
}

func final() {
	_deferred.Cleanup()
}

// Done all registered deferred funcs are executed on signal.
// Call <-Done() at the end of main function (for example).
func Done() <-chan struct{} { return _deferred.Done() }

// Register a function to be executed on app exit (by receiving an os signal),
// based on priority.
func Register(action func(), priority ...int) {
	_deferred.Register(action, priority...)
}

func onSignal(f func(), sig ...os.Signal) {
	if f == nil {
		return
	}
	sigc := make(chan os.Signal, 1)
	if len(sig) > 0 {
		signal.Notify(sigc, sig...)
	} else {
		signal.Notify(sigc,
			syscall.SIGINT,
			syscall.SIGTERM,
			syscall.SIGQUIT,
			syscall.SIGTSTP)
	}
	go func() {
		<-sigc
		f()
	}()
}

// ForceExit ensures all registered deferred funcs are executed,
// and waits for completion of all of them, then calls os.Exit(code)
// with the provided code.
func ForceExit(code int) {
	forceExit(code, true)
}

func forceExit(code int, callExit bool) {
	final()
	<-Done()
	if callExit {
		os.Exit(code)
	}
}
