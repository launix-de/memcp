/*
Copyright (C) 2026  Carl-Philip Hänsch

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
package storage

import "os"
import "fmt"
import "time"

const remoteStorageAttempts = 3

// PersistenceFailure is the only panic value emitted for an I/O failure at a
// persistence boundary. Transaction code may recover this type to abort a
// write; all unrelated panics must continue to propagate.
type PersistenceFailure struct {
	Backend   string
	Database  string
	Operation string
	Err       error
	// OutcomeUnknown is set only when the operation which publishes commit
	// authority completed but its final durability barrier failed. Callers must
	// not roll back in-memory state in that case: recovery may observe the
	// commit marker even though the client receives an error.
	OutcomeUnknown bool
}

func (e *PersistenceFailure) Error() string {
	message := fmt.Sprintf("persistence %s failed for %s database %s: %v", e.Operation, e.Backend, e.Database, e.Err)
	if e.OutcomeUnknown {
		message += " (commit outcome unknown)"
	}
	return message
}

func (e *PersistenceFailure) Unwrap() error { return e.Err }

func raisePersistenceFailure(backend string, database string, operation string, err error) {
	failure := &PersistenceFailure{
		Backend: backend, Database: database, Operation: operation, Err: err,
	}
	// This must never use MemCP tables: the statistics database may be the
	// failing backend and recursive failure reporting would never terminate.
	_, _ = fmt.Fprintf(os.Stderr, "CRITICAL: %s\n", failure.Error())
	panic(failure)
}

func reportPersistenceCleanupFailure(backend string, database string, operation string, err error) {
	_, _ = fmt.Fprintf(os.Stderr, "CRITICAL: persistence %s failed for %s database %s: %v; committed data is intact but orphan cleanup is incomplete\n", operation, backend, database, err)
}

func reportPersistenceAmbiguousFailure(backend string, database string, operation string, err error) {
	_, _ = fmt.Fprintf(os.Stderr, "CRITICAL: persistence %s failed for %s database %s: %v; publication completed but crash durability is unknown\n", operation, backend, database, err)
}

func recoverPersistenceFailure(errp *error) {
	if recovered := recover(); recovered != nil {
		if failure, ok := recovered.(*PersistenceFailure); ok {
			*errp = failure
			return
		}
		panic(recovered)
	}
}

func persistenceCall(operation func()) (err error) {
	defer recoverPersistenceFailure(&err)
	operation()
	return nil
}

func markCommitOutcomeUnknown(err error) error {
	failure, ok := err.(*PersistenceFailure)
	if !ok {
		return err
	}
	copy := *failure
	copy.OutcomeUnknown = true
	return &copy
}

func commitOutcomeUnknown(err error) bool {
	failure, ok := err.(*PersistenceFailure)
	return ok && failure.OutcomeUnknown
}

func remoteRetry(operation func() error) error {
	var err error
	for attempt := 0; attempt < remoteStorageAttempts; attempt++ {
		if err = operation(); err == nil {
			return nil
		}
		if attempt+1 < remoteStorageAttempts {
			time.Sleep(time.Duration(10*(1<<attempt)) * time.Millisecond)
		}
	}
	return err
}

func remoteRetryValue[T any](operation func() (T, error)) (T, error) {
	var value T
	err := remoteRetry(func() error {
		var err error
		value, err = operation()
		return err
	})
	return value, err
}
