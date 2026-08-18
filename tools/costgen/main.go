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

// costgen measures real storage-engine costs for the query planner's
// physical-strategy cost model (the calibrated constants in
// lib/queryplan.scm: planner_direct_presence_probe_cost,
// planner_presence_carrier_cost, planner_recset_carrier_cost) and patches
// them in place, the same way `make jitgen` regenerates JIT emitters from a
// live analysis instead of hand-typed code.
//
// Why this exists: those three constants are calibrated, hand-typed magic
// numbers with no way to tell whether they still reflect reality. A
// storage-primitive perf change (e.g. the adaptive-index buildIndex fix in
// PR #525) silently invalidates them. Re-run `make costgen` whenever a
// storage-primitive perf change lands.
//
// Usage:
//
//	go run ./tools/costgen                # measure and print, don't modify
//	go run ./tools/costgen -patch          # measure and patch lib/queryplan.scm
package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// syncBuffer is a bytes.Buffer safe for one writer goroutine (the child
// process's stdout/stderr pump) and one reader goroutine (the readiness
// poller) at the same time.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func logStep(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "costgen: "+format+"\n", a...)
}

const beginMarker = "/* BEGIN GENERATED COST CONSTANTS."
const endMarker = "/* END GENERATED COST CONSTANTS */"

// probeIterations is how many repeated scan_exists / point-read calls each
// scenario performs. High enough to average out first-call warmup noise for
// the carrier scenarios, small enough that the direct-probe scenario (which
// intentionally has no reusable structure) still finishes quickly.
const probeIterations = 400

// bigRows is the driving-table size for the recset-carrier scenario. Chosen
// to land solidly in the "adaptive index actually gets built" regime
// (Settings.IndexThreshold default 5) while keeping the tool's own runtime
// reasonable.
const bigRows = 200000

func main() {
	patch := false
	for _, arg := range os.Args[1:] {
		if arg == "-patch" {
			patch = true
		}
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		fatal(err)
	}

	binPath, err := buildMemcp(repoRoot)
	if err != nil {
		fatal(fmt.Errorf("building memcp: %w", err))
	}
	defer os.Remove(binPath)

	dataDir, err := os.MkdirTemp("", "costgen-data-*")
	if err != nil {
		fatal(err)
	}
	defer os.RemoveAll(dataDir)

	apiPort, err := freePort()
	if err != nil {
		fatal(err)
	}
	mysqlPort, err := freePort()
	if err != nil {
		fatal(err)
	}

	results, err := runBenchmark(binPath, dataDir, apiPort, mysqlPort)
	if err != nil {
		fatal(fmt.Errorf("running benchmark: %w", err))
	}

	c := computeConstants(results)
	fmt.Println(c.describe())

	if patch {
		queryplanPath := filepath.Join(repoRoot, "lib", "queryplan.scm")
		if err := patchQueryplan(queryplanPath, c); err != nil {
			fatal(fmt.Errorf("patching %s: %w", queryplanPath, err))
		}
		fmt.Println("patched", queryplanPath)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "costgen:", err)
	os.Exit(1)
}

func findRepoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func buildMemcp(repoRoot string) (string, error) {
	binPath := filepath.Join(os.TempDir(), fmt.Sprintf("costgen-memcp-%d", os.Getpid()))
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = repoRoot
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}
	return binPath, nil
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// benchResults holds the raw nanosecond/count measurements parsed out of the
// benchmark script's final printed assoc list.
type benchResults struct {
	directNs      int64
	directCalls   int64
	ktBuildNs     int64
	ktReadNs      int64
	ktReadCalls   int64
	recsetBuildNs int64
	recsetProjNs  int64
	bigRows       int64
}

func runBenchmark(binPath, dataDir string, apiPort, mysqlPort int) (*benchResults, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, binPath, "-data", dataDir,
		fmt.Sprintf("--api-port=%d", apiPort),
		fmt.Sprintf("--mysql-port=%d", mysqlPort),
		"lib/main.scm")
	cmd.Dir = mustRepoRootFromBin()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout := &syncBuffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stdout

	logStep("starting memcp (api=%d mysql=%d data=%s)", apiPort, mysqlPort, dataDir)
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	if err := waitForListening(stdout, apiPort, 20*time.Second); err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("server never became ready: %w\noutput so far:\n%s", err, stdout.String())
	}
	logStep("server ready, running SQL setup (%d big rows)", bigRows)

	if err := setupData(mysqlPort); err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("SQL setup: %w", err)
	}
	logStep("SQL setup done, feeding benchmark script")

	if _, err := io.WriteString(stdin, benchmarkScript()); err != nil {
		cmd.Process.Kill()
		return nil, err
	}
	stdin.Close()

	logStep("waiting for the result line (not for a graceful shutdown, which can be slow)")
	waitErrCh := make(chan error, 1)
	go func() { waitErrCh <- cmd.Wait() }()

	resultDeadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(resultDeadline) {
		if resultLineFrom(stdout.String()) != "" {
			break
		}
		select {
		case err := <-waitErrCh:
			// process exited (or was context-killed) before printing a result
			if err != nil {
				return nil, fmt.Errorf("%w\noutput:\n%s", err, stdout.String())
			}
			return parseResults(stdout.String())
		case <-time.After(100 * time.Millisecond):
		}
	}
	// Result line is in the buffer; stop waiting for shutdown/table-compression
	// and kill the process outright.
	cmd.Process.Kill()
	<-waitErrCh
	logStep("result line captured, process terminated")

	return parseResults(stdout.String())
}

// mustRepoRootFromBin re-derives the repo root for the child process's
// working directory (it needs to find lib/main.scm relative to cwd).
func mustRepoRootFromBin() string {
	root, err := findRepoRoot()
	if err != nil {
		fatal(err)
	}
	return root
}

func waitForListening(out *syncBuffer, apiPort int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	needle := fmt.Sprintf(":%d", apiPort)
	for time.Now().Before(deadline) {
		if strings.Contains(out.String(), needle) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %q in server output", needle)
}

func setupData(mysqlPort int) error {
	var sb strings.Builder
	sb.WriteString("DROP DATABASE IF EXISTS costgen;\n")
	sb.WriteString("CREATE DATABASE costgen;\n")
	sb.WriteString("USE costgen;\n")
	sb.WriteString("CREATE TABLE side (id INT, flag INT);\n")
	sb.WriteString("INSERT INTO side (id, flag) VALUES (1,1),(2,0),(3,1),(4,0),(5,1),(6,0),(7,1),(8,0),(9,1),(10,0);\n")
	sb.WriteString("CREATE TABLE big (id INT, k INT);\n")
	const batchSize = 2000
	for start := 0; start < bigRows; start += batchSize {
		end := start + batchSize
		if end > bigRows {
			end = bigRows
		}
		sb.WriteString("INSERT INTO big (id, k) VALUES ")
		for i := start; i < end; i++ {
			if i > start {
				sb.WriteString(",")
			}
			fmt.Fprintf(&sb, "(%d,%d)", i, (i%10)+1)
		}
		sb.WriteString(";\n")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "mysql", "-h127.0.0.1", fmt.Sprintf("-P%d", mysqlPort), "-uroot", "-padmin")
	cmd.Stdin = strings.NewReader(sb.String())
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}
	return nil
}

func benchmarkScript() string {
	return fmt.Sprintf(`
(define sess (newsession))
(with_autocommit sess (lambda ()
  (begin
    (define tx (sess "__memcp_tx"))
    (define keys (list 1 2 3 4 5 6 7 8 9 10))
    (define iterations (produceN %d))

    (define probe_once (lambda (k)
      (scan_exists tx (table "costgen" "side") (list "id" "flag")
        (lambda (id flag) (and (equal?? id k) (equal?? flag 1))))))

    (define direct_t0 (nanotime))
    (define direct_hits (reduce iterations (lambda (acc i)
      (+ acc (if (probe_once (+ 1 (mod i 10))) 1 0))) 0))
    (define direct_t1 (nanotime))

    (createtable "costgen" ".grp:costgen_bench" (list (list "column" "k0" "int" '() '())) (list "engine" "cache") true)
    (createcolumn (table "costgen" ".grp:costgen_bench") "flagval" "any" '() '() (list "k0")
      (lambda (k0) (probe_once k0)))
    (define kt_build_t0 (nanotime))
    (reduce keys (lambda (_ k) (insert (table "costgen" ".grp:costgen_bench") (list "k0") (list (list k)))) nil)
    /* Real compiler-generated keytables get bulk-populated with precomputed
    values during their build pass (group_insert_batches), not lazily on first
    read. Re-issuing createcolumn on the now-populated table forces the same
    eager Compress() a fresh, unfiltered createcolumn call does -- this keeps
    the timed read window measuring genuinely warm point-reads instead of
    accidentally re-triggering per-row materialization mid-scan (the scan
    condition below touches both k0 and flagval per row, so an unwarmed
    lazy column would recompute flagval while scanning toward a match). */
    (createcolumn (table "costgen" ".grp:costgen_bench") "flagval" "any" '() '() (list "k0")
      (lambda (k0) (probe_once k0)))
    (define kt_build_t1 (nanotime))

    (define kt_read_t0 (nanotime))
    (define kt_hits (reduce iterations (lambda (acc i)
      (+ acc (if (scan_exists tx (table "costgen" ".grp:costgen_bench") (list "k0" "flagval")
                   (lambda (k0 fv) (and (equal?? k0 (+ 1 (mod i 10))) (equal?? fv true)))) 1 0))) 0))
    (define kt_read_t1 (nanotime))

    (define recset_t0 (nanotime))
    (define rs (scan_recset tx (table "costgen" "side") (list "id") (lambda (id) (probe_once id))))
    (define recset_build_t1 (nanotime))
    (define proj (recset_project_join tx rs (list "id") (table "costgen" "big") (list "k")))
    (define recset_t1 (nanotime))
    (define big_rows (recset_count proj))

    (list
      "direct_ns" (- direct_t1 direct_t0) "direct_calls" (count iterations) "direct_hits" direct_hits
      "kt_build_ns" (- kt_build_t1 kt_build_t0)
      "kt_read_ns" (- kt_read_t1 kt_read_t0) "kt_read_calls" (count iterations) "kt_hits" kt_hits
      "recset_build_ns" (- recset_build_t1 recset_t0)
      "recset_proj_ns" (- recset_t1 recset_build_t1)
      "big_rows" big_rows)))))
`, probeIterations)
}

var resultLineRe = regexp.MustCompile(`^\s*=\s*\((.*)\)\s*$`)
var kvRe = regexp.MustCompile(`"([a-z_]+)"\s+(-?[0-9]+)`)
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// resultLineFrom returns the last line in output that matches the
// benchmark script's printed assoc list, or "" if none is present yet.
func resultLineFrom(output string) string {
	scanner := bufio.NewScanner(strings.NewReader(output))
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	var line string
	for scanner.Scan() {
		candidate := ansiRe.ReplaceAllString(scanner.Text(), "")
		if resultLineRe.MatchString(candidate) {
			line = candidate
		}
	}
	return line
}

func parseResults(output string) (*benchResults, error) {
	line := resultLineFrom(output)
	if line == "" {
		return nil, fmt.Errorf("no result line found in output:\n%s", output)
	}
	values := map[string]int64{}
	for _, m := range kvRe.FindAllStringSubmatch(line, -1) {
		v, err := strconv.ParseInt(m[2], 10, 64)
		if err != nil {
			continue
		}
		values[m[1]] = v
	}
	get := func(k string) int64 { return values[k] }
	r := &benchResults{
		directNs:      get("direct_ns"),
		directCalls:   get("direct_calls"),
		ktBuildNs:     get("kt_build_ns"),
		ktReadNs:      get("kt_read_ns"),
		ktReadCalls:   get("kt_read_calls"),
		recsetBuildNs: get("recset_build_ns"),
		recsetProjNs:  get("recset_proj_ns"),
		bigRows:       get("big_rows"),
	}
	if r.directCalls == 0 || r.ktReadCalls == 0 {
		return nil, fmt.Errorf("incomplete result line: %s", line)
	}
	return r, nil
}

// calibratedConstants mirrors the three planner_*_cost lambdas' numeric
// literals in lib/queryplan.scm.
type calibratedConstants struct {
	directProbeNsPerRow int64
	carrierStartupNs    int64
	carrierBuildNsPerRow int64
	recsetStartupNs     int64
	recsetBuildNsPerRow int64
}

func computeConstants(r *benchResults) *calibratedConstants {
	directPerCall := r.directNs / r.directCalls
	ktReadPerCall := r.ktReadNs / r.ktReadCalls
	recsetPerRow := r.recsetProjNs / max64(r.bigRows, 1)
	return &calibratedConstants{
		directProbeNsPerRow: directPerCall,
		carrierStartupNs:    r.ktBuildNs,
		carrierBuildNsPerRow: ktReadPerCall,
		recsetStartupNs:     r.recsetBuildNs,
		recsetBuildNsPerRow: recsetPerRow,
	}
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func (c *calibratedConstants) describe() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "direct probe:    %6d ns/accepted-row (no reusable structure)\n", c.directProbeNsPerRow)
	fmt.Fprintf(&sb, "keytable carrier: %6d ns startup + %6d ns/driving-row (point-read)\n", c.carrierStartupNs, c.carrierBuildNsPerRow)
	fmt.Fprintf(&sb, "recset carrier:   %6d ns startup + %6d ns/driving-row (project-join)\n", c.recsetStartupNs, c.recsetBuildNsPerRow)
	return sb.String()
}

func patchQueryplan(path string, c *calibratedConstants) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	beginIdx := strings.Index(content, beginMarker)
	endIdx := strings.Index(content, endMarker)
	if beginIdx < 0 || endIdx < 0 || endIdx < beginIdx {
		return fmt.Errorf("could not find generated-cost-constants block markers")
	}
	endIdx += len(endMarker)

	block := fmt.Sprintf(`/* BEGIN GENERATED COST CONSTANTS. DO NEVER MANUALLY EDIT THIS SECTION. RUN make costgen TO UPDATE.
Calibrated by tools/costgen against a live storage engine (see that tool for the exact
benchmark scenarios). Re-run make costgen whenever a storage-primitive perf change lands
(e.g. an adaptive-index build fix) so these numbers keep tracking reality instead of
silently drifting stale.

domain_rows is the stage's own (usually small) input table -- what a carrier is built
FROM. probe_rows is the driving table's row/evaluation count -- how many times the
built carrier is actually READ. A keytable's dominant cost is per read (row_ns,
scaled by probe_rows); a RecSet's dominant cost is its one-pass build over the driving
side (build_ns, also scaled by probe_rows -- recset_project_join visits it once
regardless of domain size, which is why domain_rows barely matters there). */
(define planner_direct_presence_probe_cost (lambda (probe_rows)
	(planner_cost 0 0 (* probe_rows %d) 0 0 0 0 0 probe_rows 0.75)))

(define planner_presence_carrier_cost (lambda (domain_rows probe_rows)
	(planner_cost %d (* probe_rows %d) 0 0 0 0
		(* domain_rows 8) 0 domain_rows 0.65)))

(define planner_recset_carrier_cost (lambda (domain_rows probe_rows)
	(planner_cost %d 0 0 0 0 (* probe_rows %d)
		(* probe_rows 1) 0 probe_rows 0.6)))
/* END GENERATED COST CONSTANTS */`,
		c.directProbeNsPerRow,
		c.carrierStartupNs, c.carrierBuildNsPerRow,
		c.recsetStartupNs, c.recsetBuildNsPerRow)

	newContent := content[:beginIdx] + block + content[endIdx:]
	return os.WriteFile(path, []byte(newContent), 0644)
}
