/*
Copyright (C) 2025  MemCP Contributors

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

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/launix-de/memcp/scm"
)

// syscall mmap wrappers using the standard library (deprecated but sufficient).
func syscallMmap(fd int, length int) ([]byte, error) {
	return syscall.Mmap(fd, 0, length, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
}

func syscallMunmap(b []byte) error { return syscall.Munmap(b) }

// encodeScmer prints a compact textual encoding of a Scheme AST to w.
// Unknowns print as "?".
// - Unknown symbols (not a global function and not one of the provided column names) => "?".
// - Lambdas (scm.Proc) => "?".
// - Go builtins (func(...scm.Scmer) scm.Scmer) => function name if found in Globalenv, else "?".
// For filters, pass the condition Proc.Body as v and the filter columns as context.
// For sort expressions, pass the string column name or the Proc.Body with its params as context.
// columnSymbols must be the Proc.Params list when encoding a lambda body. If present:
// - Any symbol equal to a param prints as the corresponding column name by index.
// - Any NthLocalVar(i) prints as columns[i] (when i < len(columns)); otherwise "?".
func encodeScmer(v scm.Scmer, w io.Writer, columns []string, columnSymbols []scm.Scmer) {
	cols := make(map[string]bool, len(columns))
	for _, c := range columns {
		cols[strings.ToLower(c)] = true
	}
	// Build symbol->index from Proc.Params to map lambda params to actual columns
	symIndex := make(map[string]int, len(columnSymbols))
	for i, s := range columnSymbols {
		if s.IsSymbol() {
			symIndex[strings.ToLower(s.String())] = i
			continue
		}
		if sym, ok := s.Any().(scm.Symbol); ok {
			symIndex[strings.ToLower(string(sym))] = i
		}
	}

	var enc func(scm.Scmer)
	writeSymbolOrColumn := func(s string) {
		sLower := strings.ToLower(s)
		// Prefer mapping lambda param -> column name
		if idx, ok := symIndex[sLower]; ok {
			if idx >= 0 && idx < len(columns) {
				io.WriteString(w, columns[idx])
				return
			}
			io.WriteString(w, "?")
			return
		}
		// Otherwise, if it looks like a global function/operator, print symbol
		if scm.Globalenv.FindRead(scm.Symbol(s)) != nil {
			io.WriteString(w, s)
			return
		}
		// Unknown
		io.WriteString(w, "?")
	}

	enc = func(node scm.Scmer) {
		switch {
		case node.IsNil():
			io.WriteString(w, "nil")
		case node.IsBool():
			if node.Bool() {
				io.WriteString(w, "true")
			} else {
				io.WriteString(w, "false")
			}
		case node.IsInt():
			io.WriteString(w, fmt.Sprint(node.Int()))
		case node.IsFloat():
			io.WriteString(w, fmt.Sprint(node.Float()))
		case node.IsString():
			io.WriteString(w, "\"")
			io.WriteString(w, node.String())
			io.WriteString(w, "\"")
		case node.IsSymbol():
			writeSymbolOrColumn(node.String())
		case node.IsSlice():
			slice := node.Slice()
			if len(slice) > 0 {
				if slice[0].IsSymbol() && slice[0].String() == "outer" {
					io.WriteString(w, "?")
					return
				}
			}
			io.WriteString(w, "(")
			for i, item := range slice {
				if i > 0 {
					io.WriteString(w, " ")
				}
				enc(item)
			}
			io.WriteString(w, ")")
		default:
			// Prefer tag-based decoding for special cases.
			if node.IsProc() {
				io.WriteString(w, "?")
				return
			}
			if node.IsNthLocalVar() {
				i := int(node.NthLocalVar())
				if i >= 0 && i < len(columns) {
					io.WriteString(w, columns[i])
				} else {
					io.WriteString(w, "?")
				}
				return
			}
			// Native function: try to resolve declaration if present.
			if def := scm.DeclarationForValue(node); def != nil {
				io.WriteString(w, def.Name)
				return
			}
			// Fallback unknown
			io.WriteString(w, "?")
		}
	}

	enc(v)
}

// helper that returns encoded string
func encodeScmerToString(v scm.Scmer, columns []string, columnSymbols []scm.Scmer) string {
	var b bytes.Buffer
	encodeScmer(v, &b, columns, columnSymbols)
	return b.String()
}

// Minimum table size required to collect scan statistics.
// Deprecated: use Settings.AnalyzeMinItems instead
const scanStatsMinInput int64 = 1000

// ensureSystemStatistic ensures the `system_statistic.scans` table exists with expected columns.
func ensureSystemStatistic() {
	const dbName = "system_statistic"
	const tblName = "scans"

	// create database if missing
	if GetDatabase(dbName) == nil {
		CreateDatabase(dbName, true)
	}
	db := GetDatabase(dbName)
	if db == nil {
		return // should not happen; avoid panicking during init
	}

	// create table if missing (use Sloppy persistency to avoid fsync costs)
	t, _ := CreateTable(dbName, tblName, Sloppy, true)
	if t == nil {
		t = db.GetTable(tblName)
		if t == nil {
			return
		}
	}
	// ensure persistency mode is Sloppy even if table pre-existed
	if t.PersistencyMode != Sloppy {
		t.PersistencyMode = Sloppy
		t.schema.save()
	}
	// ensure columns exist
	need := []struct {
		name string
		typ  string
	}{
		{"schema", "TEXT"},
		{"table", "TEXT"},
		{"ordered", "BOOL"},
		{"filter", "TEXT"},
		{"order", "TEXT"},
		{"inputCount", "INT"},
		{"outputCount", "INT"},
		// TODO: measurements are temporary; remove later (store in nanoseconds)
		{"analyze_ns", "INT"},
		{"exec_ns", "INT"},
	}
	have := make(map[string]bool)
	if t != nil {
		for _, c := range t.Columns {
			have[strings.ToLower(c.Name)] = true
		}
		for _, c := range need {
			if !have[strings.ToLower(c.name)] {
				t.CreateColumn(c.name, c.typ, nil, nil)
			}
		}
	}

	// --- New: ensure system_statistic.table_histogram exists ---
	// Schema: table_histogram(schema TEXT, table TEXT, model BLOB, UNIQUE(schema, table))
	ensureTable := func(db *database, name string, pm PersistencyMode) *table {
		tt, _ := CreateTable(dbName, name, pm, true)
		if tt == nil {
			tt = db.GetTable(name)
		}
		if tt == nil {
			return nil
		}
		if tt.PersistencyMode != pm {
			tt.PersistencyMode = pm
			tt.schema.save()
		}
		return tt
	}

	// helper: ensure columns exist by name/type
	ensureCols := func(tt *table, cols []struct{ name, typ string }) {
		if tt == nil {
			return
		}
		have := make(map[string]bool)
		for _, c := range tt.Columns {
			have[strings.ToLower(c.Name)] = true
		}
		for _, c := range cols {
			if !have[strings.ToLower(c.name)] {
				tt.CreateColumn(c.name, c.typ, nil, nil)
			}
		}
	}

	// helper: ensure a unique key with the exact set of columns exists
	ensureUnique := func(tt *table, id string, cols []string) {
		if tt == nil {
			return
		}
		// check if unique with same columns already exists (order-insensitive)
		has := false
		want := make(map[string]bool, len(cols))
		for _, c := range cols {
			want[strings.ToLower(c)] = true
		}
		for _, u := range tt.Unique {
			if len(u.Cols) != len(cols) {
				continue
			}
			ok := true
			for _, c := range u.Cols {
				if !want[strings.ToLower(c)] {
					ok = false
					break
				}
			}
			if ok {
				has = true
				break
			}
		}
		if !has {
			// append unique and persist
			tt.schema.schemalock.Lock()
			tt.Unique = append(tt.Unique, uniqueKey{Id: id, Cols: cols})
			tt.schema.save()
			tt.schema.schemalock.Unlock()
		}
	}

	// table_histogram
	th := ensureTable(db, "table_histogram", Sloppy)
	ensureCols(th, []struct{ name, typ string }{
		{"schema", "TEXT"},
		{"table", "TEXT"},
		{"model", "BLOB"}, // stored as string/blob; overlay handles long data
	})
	ensureUnique(th, "uniq_table_histogram_schema_table", []string{"schema", "table"})

	// base_models: base_models(id PRIMARY KEY, model)
	// Use Safe persistence to protect valuable models (logged + durable).
	bm := ensureTable(db, "base_models", Safe)
	ensureCols(bm, []struct{ name, typ string }{
		{"id", "TEXT"},
		{"model", "BLOB"},
	})
	// ensure PRIMARY KEY(id)
	// treat any unique on [id] as sufficient; otherwise add PRIMARY
	ensureUnique(bm, "PRIMARY", []string{"id"})
}

// safeLogScan writes a single row into system_statistic.scans. Failures are ignored.
// TODO: measurements are temporary; remove later (nanoseconds)
func safeLogScan(schema, table string, ordered bool, filter, order string, inputCount, outputCount, analyzeNs, execNs int64) {
	defer func() { _ = recover() }()
	db := GetDatabase("system_statistic")
	if db == nil {
		return
	}
	t := db.GetTable("scans")
	if t == nil {
		return
	}

	cols := []string{"schema", "table", "ordered", "filter", "order", "inputCount", "outputCount", "analyze_ns", "exec_ns"}
	row := []scm.Scmer{
		scm.NewString(schema),
		scm.NewString(table),
		scm.NewBool(ordered),
		scm.NewString(filter),
		scm.NewString(order),
		scm.NewInt(inputCount),
		scm.NewInt(outputCount),
		scm.NewInt(analyzeNs),
		scm.NewInt(execNs),
	}
	t.Insert(cols, [][]scm.Scmer{row}, nil, scm.NewNil(), false, nil)
}

// ---- AI IPC (shared-memory) estimator ----

// Estimator provides scan selectivity estimation via shared-memory IPC with ai_optimizer.py.
type Estimator struct {
	path       string
	cmd        *exec.Cmd
	mmap       []byte
	mu         sync.Mutex
	connected  bool
	lastStatus uint64
}

// Shared memory layout (little endian):
// Request header:
// [0..7]   reqSeq (uint64)
// [8..15]  respSeq (uint64)   // set by Python when ready
// [16..23] inputCount (uint64)
// [24..27] schemaLen (uint32)
// [28..31] tableLen  (uint32)
// [32..35] filterLen (uint32)
// [36..39] orderLen  (uint32)
// [40..47] respOutput (float64) // set by Python
// [48..55] statusSeq (uint64) // incremented by Python on new status message
// [56..59] statusLen (uint32)
// [60..63] statusCode (uint32) // 1=READY, 2=INFO, 3=ERROR
// Payloads:
// [64..64+REQ_MAX)   request payload: schema | table | filter | order (UTF-8)
// [64+REQ_MAX..end)  status payload: arbitrary UTF-8 status message
const (
	shmHeaderSize = 64
	reqMax        = 512 * 1024
	statMax       = 64 * 1024
)

// NewEstimator creates a shared memory file, mmaps it, and spawns the Python helper.
func NewEstimator(pythonPath string) (*Estimator, error) {
	// choose a tmpfs path if available
	base := "/dev/shm"
	if st, err := os.Stat(base); err != nil || !st.IsDir() {
		base = os.TempDir()
	}
	fn := filepath.Join(base, fmt.Sprintf("memcp-ai-%d-%d.shm", os.Getpid(), time.Now().UnixNano()))
	f, err := os.OpenFile(fn, os.O_CREATE|os.O_RDWR|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// set size
	if err := f.Truncate(int64(shmHeaderSize + reqMax + statMax)); err != nil {
		return nil, err
	}
	b, err := mmapFile(f)
	if err != nil {
		return nil, err
	}

	// spawn python helper
	pp := pythonPath
	if pp == "" {
		pp = "./ai_optimizer.py"
	}
	cmd := exec.Command(pp, "--ipc", fn)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		unmapFile(b)
		return nil, err
	}
	e := &Estimator{path: fn, cmd: cmd, mmap: b}
	// Wait up to 5s for READY; print any INFO/ERROR status lines
	_ = e.awaitReady(5 * time.Second)
	// Start background status pump for asynchronous INFO/ERROR logs
	go func() {
		for {
			e.mu.Lock()
			alive := e.mmap != nil
			e.mu.Unlock()
			if !alive {
				return
			}
			if !e.pollStatusOnce(true) {
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()
	return e, nil
}

// Close terminates the helper and unmaps the shared memory.
func (e *Estimator) Close() {
	if e == nil {
		return
	}
	if e.cmd != nil && e.cmd.Process != nil {
		_ = e.cmd.Process.Kill()
	}
	if e.mmap != nil {
		unmapFile(e.mmap)
	}
	_ = os.Remove(e.path)
}

// awaitReady polls status messages and marks the estimator connected when READY is received.
// It also prints INFO/ERROR messages received during the wait.
func (e *Estimator) awaitReady(d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if e.pollStatusOnce(true) {
			return e.connected
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !e.connected {
		fmt.Println("AIEstimator: not ready yet; continuing in background")
	}
	return e.connected
}

// pollStatusOnce reads a single status update (if any). When printMsgs is true, prints messages.
// Returns true if any status was consumed.
func (e *Estimator) pollStatusOnce(printMsgs bool) bool {
	if e.mmap == nil {
		return false
	}
	buf := e.mmap
	sseq := binary.LittleEndian.Uint64(buf[48:56])
	if sseq == 0 || sseq == e.lastStatus {
		return false
	}
	slen := int(binary.LittleEndian.Uint32(buf[56:60]))
	scode := binary.LittleEndian.Uint32(buf[60:64])
	if slen < 0 || slen > statMax {
		slen = 0
	}
	base := shmHeaderSize + reqMax
	var msg string
	if slen > 0 {
		msg = string(buf[base : base+slen])
	}
	e.lastStatus = sseq
	switch scode {
	case 1: // READY
		e.connected = true
		if printMsgs {
			if msg != "" {
				fmt.Println("AIEstimator:", msg)
			} else {
				fmt.Println("AIEstimator: ready")
			}
		}
	case 2: // INFO
		if printMsgs && msg != "" {
			fmt.Println("AIEstimator:", msg)
		}
	case 3: // ERROR
		if printMsgs && msg != "" {
			fmt.Println("AIEstimator ERROR:", msg)
		}
	default:
		if printMsgs && msg != "" {
			fmt.Println("AIEstimator status:", msg)
		}
	}
	return true
}

var globalEstimatorMu sync.Mutex
var globalEstimator *Estimator

// StartGlobalEstimator starts the estimator if not already running.
func StartGlobalEstimator() {
	globalEstimatorMu.Lock()
	defer globalEstimatorMu.Unlock()
	if globalEstimator != nil {
		return
	}
	fmt.Println("AIEstimator: starting...")
	est, err := NewEstimator("./ai_optimizer.py")
	if err != nil {
		fmt.Println("AIEstimator: failed to start:", err)
		return
	}
	if est != nil && est.cmd != nil && est.cmd.Process != nil {
		fmt.Println("AIEstimator: started (pid=", est.cmd.Process.Pid, ")")
	} else {
		fmt.Println("AIEstimator: started")
	}
	globalEstimator = est
}

// StopGlobalEstimator stops and frees the estimator.
func StopGlobalEstimator() {
	globalEstimatorMu.Lock()
	defer globalEstimatorMu.Unlock()
	if globalEstimator != nil {
		fmt.Println("AIEstimator: stopping...")
		globalEstimator.Close()
		globalEstimator = nil
		fmt.Println("AIEstimator: stopped")
	}
}

// ScanEstimate encodes condition and sortcols similar to scan/scan_order logging helpers
// and queries the Python estimator via shared memory. If sortcols is empty, treats as unordered.
func (e *Estimator) ScanEstimate(schema, table string, conditionCols []string, condition scm.Scmer, sortcols []scm.Scmer, inputCount int64, timeout time.Duration) (int64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.mmap == nil {
		return inputCount, fmt.Errorf("estimator not initialized")
	}
	// Drain pending statuses to keep logs flowing; if not connected, fallback
	for e.pollStatusOnce(true) {
	}
	if !e.connected {
		return inputCount, fmt.Errorf("estimator not ready")
	}

	// Encode filter string
	filter := ""
	if proc, ok := condition.Any().(scm.Proc); ok {
		var params []scm.Scmer
		if proc.Params.IsSlice() {
			params = proc.Params.Slice()
		} else if arr, ok := proc.Params.Any().([]scm.Scmer); ok {
			params = arr
		}
		filter = encodeScmerToString(proc.Body, conditionCols, params)
	}
	// Encode order string from sortcols
	var sb strings.Builder
	for i, sc := range sortcols {
		if i > 0 {
			sb.WriteByte('|')
		}
		if sc.IsString() {
			sb.WriteString(sc.String())
		} else {
			encodeScmer(sc, &sb, nil, nil)
		}
	}
	order := sb.String()
	// Build contiguous payload
	sSchema := []byte(schema)
	sTable := []byte(table)
	sFilter := []byte(filter)
	sOrder := []byte(order)
	total := len(sSchema) + len(sTable) + len(sFilter) + len(sOrder)
	if total > reqMax {
		return -1, fmt.Errorf("request too large: %d", total)
	}

	buf := e.mmap
	// Next sequence
	seq := binary.LittleEndian.Uint64(buf[0:8]) + 1
	// Header
	binary.LittleEndian.PutUint64(buf[16:24], uint64(inputCount))
	binary.LittleEndian.PutUint32(buf[24:28], uint32(len(sSchema)))
	binary.LittleEndian.PutUint32(buf[28:32], uint32(len(sTable)))
	binary.LittleEndian.PutUint32(buf[32:36], uint32(len(sFilter)))
	binary.LittleEndian.PutUint32(buf[36:40], uint32(len(sOrder)))
	// Payload
	off := shmHeaderSize
	copy(buf[off:off+len(sSchema)], sSchema)
	off += len(sSchema)
	copy(buf[off:off+len(sTable)], sTable)
	off += len(sTable)
	copy(buf[off:off+len(sFilter)], sFilter)
	off += len(sFilter)
	copy(buf[off:off+len(sOrder)], sOrder)
	// Publish seq last
	binary.LittleEndian.PutUint64(buf[0:8], seq)

	// Wait for respSeq == seq
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		respSeq := binary.LittleEndian.Uint64(buf[8:16])
		if respSeq == seq {
			bits := binary.LittleEndian.Uint64(buf[40:48])
			out := math.Float64frombits(bits)
			if out < 0 {
				out = 0
			}
			return int64(out + 0.5), nil
		}
		time.Sleep(200 * time.Microsecond)
	}
	return inputCount, fmt.Errorf("timeout waiting for AI response")
}

// sendGeneric sends an opcode and payload via the shared memory request area
// using a generic request (schemaLen/tableLen/filterLen/orderLen all zero).
// It returns the raw response payload.
func (e *Estimator) sendGeneric(op byte, payload []byte, timeout time.Duration) ([]byte, error) {
	if e == nil {
		return nil, errors.New("estimator not initialized")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.mmap == nil {
		return nil, errors.New("estimator not initialized")
	}
	buf := e.mmap
	// Next sequence
	seq := binary.LittleEndian.Uint64(buf[0:8]) + 1
	// zero normal header lengths and inputCount to signal generic op
	binary.LittleEndian.PutUint64(buf[16:24], 0)
	binary.LittleEndian.PutUint32(buf[24:28], 0)
	binary.LittleEndian.PutUint32(buf[28:32], 0)
	binary.LittleEndian.PutUint32(buf[32:36], 0)
	binary.LittleEndian.PutUint32(buf[36:40], 0)
	// write opcode + payload length + payload
	off := shmHeaderSize
	if off+1+4+len(payload) > shmHeaderSize+reqMax {
		return nil, errors.New("payload too large")
	}
	buf[off] = op
	off++
	binary.LittleEndian.PutUint32(buf[off:off+4], uint32(len(payload)))
	off += 4
	copy(buf[off:off+len(payload)], payload)
	// publish seq last
	binary.LittleEndian.PutUint64(buf[0:8], seq)
	// wait for response
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		respSeq := binary.LittleEndian.Uint64(buf[8:16])
		if respSeq == seq {
			roff := shmHeaderSize
			rlen := int(binary.LittleEndian.Uint32(buf[roff : roff+4]))
			roff += 4
			if roff+rlen > len(buf) || rlen < 0 {
				return nil, errors.New("invalid response length")
			}
			out := make([]byte, rlen)
			copy(out, buf[roff:roff+rlen])
			return out, nil
		}
		time.Sleep(200 * time.Microsecond)
	}
	return nil, errors.New("timeout waiting for response")
}

// SQL executes a SQL query via the estimator IPC (generic opcode 3) and
// returns the raw response string (as returned by the Python helper).
func (e *Estimator) SQL(sql string, timeout time.Duration) (string, error) {
	resp, err := e.sendGeneric(3, []byte(sql), timeout)
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

// FetchModel requests a binary model blob by ID via opcode 2. The Python
// helper responds with raw bytes of the model or empty on not found.
func (e *Estimator) FetchModel(id string, timeout time.Duration) ([]byte, error) {
	return e.sendGeneric(2, []byte(id), timeout)
}

// mmap helpers (portable enough)
func mmapFile(f *os.File) ([]byte, error) {
	sz, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	// Use syscall.Mmap (deprecated) to avoid extra deps
	//goland:noinspection ALL
	b, err := syscallMmap(int(f.Fd()), int(sz))
	return b, err
}

func unmapFile(b []byte) {
	_ = syscallMunmap(b)
}
