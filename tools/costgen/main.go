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

// costgen discovers tagged YAML workloads, runs each forced physical
// alternative, validates results/operators, solves a non-negative cost
// equation system, and regenerates planner constants.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	beginMarker = "/* BEGIN GENERATED COST CONSTANTS."
	endMarker   = "/* END GENERATED COST CONSTANTS */"
	database    = "memcp-tests"
)

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

type metadata struct {
	PhysicalCalibration bool `yaml:"physical_calibration"`
}

type step struct {
	SQL string `yaml:"sql"`
	SCM string `yaml:"scm"`
}

type testCase struct {
	Name                string `yaml:"name"`
	SQL                 string `yaml:"sql"`
	SCM                 string `yaml:"scm"`
	PhysicalCalibration bool   `yaml:"physical_calibration"`
	Warmup              int    `yaml:"warmup"`
	Repetitions         int    `yaml:"repetitions"`
	Setup               []step `yaml:"setup"`
}

type suite struct {
	Path      string
	Metadata  metadata   `yaml:"metadata"`
	Setup     []step     `yaml:"setup"`
	Cleanup   []step     `yaml:"cleanup"`
	TestCases []testCase `yaml:"test_cases"`
}

type calibrationRow struct {
	DecisionID         string   `json:"decision_id"`
	Decision           string   `json:"decision"`
	Consumer           string   `json:"consumer"`
	Plan               string   `json:"plan"`
	OperatorFamily     string   `json:"operator_family"`
	OperatorConsistent bool     `json:"operator_consistent"`
	EstimatedNS        *float64 `json:"estimated_ns"`
	ActualNS           float64  `json:"actual_ns"`
	CandidateInputRows *float64 `json:"candidate_input_rows"`
	CandidateRows      *float64 `json:"candidate_rows"`
	DriverRows         *float64 `json:"driver_rows"`
	ResultEqual        bool     `json:"result_equal"`
}

type observation struct {
	caseName string
	plan     string
	y        float64
	x        []float64
}

type constants struct {
	startupNS             int64
	candidateScanRowNS    int64
	candidateRecsetRowNS  int64
	driverCacheBuildRowNS int64
	driverCacheProbeRowNS int64
}

func main() {
	patch := flag.Bool("patch", false, "rewrite lib/queryplan.scm")
	jsonl := flag.String("jsonl", "", "write raw measurements as JSONL")
	flag.Parse()

	root, err := repoRoot()
	if err != nil {
		fatal(err)
	}
	suites, err := discoverSuites(root)
	if err != nil {
		fatal(err)
	}
	if len(suites) == 0 {
		fatal(errors.New("no tests/**/*.yaml suite has metadata.physical_calibration: true"))
	}
	server, err := startServer(root)
	if err != nil {
		fatal(err)
	}
	defer server.stop()

	observations, raw, err := runSuites(server, suites)
	if err != nil {
		fatal(err)
	}
	if *jsonl != "" {
		if err := writeJSONL(*jsonl, raw); err != nil {
			fatal(err)
		}
	}
	c, err := solve(observations)
	if err != nil {
		fatal(err)
	}
	if err := validateDecisionOrdering(observations, c); err != nil {
		fatal(err)
	}
	fmt.Printf("membership startup:  %d ns\ncandidate scan:      %d ns/input-row\nrecset build:        %d ns/matching-row\ngroup-cache build:   %d ns/matching-row\ngroup-cache probe:   %d ns/driver-row\n",
		c.startupNS, c.candidateScanRowNS, c.candidateRecsetRowNS,
		c.driverCacheBuildRowNS, c.driverCacheProbeRowNS)
	printDecisionOrdering(observations, c)
	if *patch {
		path := filepath.Join(root, "lib", "queryplan.scm")
		if err := patchQueryplan(path, c); err != nil {
			fatal(err)
		}
		fmt.Println("patched", path)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "costgen:", err)
	os.Exit(1)
}

func logStep(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "costgen: "+format+"\n", args...)
}

func repoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func discoverSuites(root string) ([]suite, error) {
	var found []suite
	err := filepath.WalkDir(filepath.Join(root, "tests"), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var header struct {
			Metadata metadata `yaml:"metadata"`
		}
		if err := yaml.Unmarshal(data, &header); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if !header.Metadata.PhysicalCalibration {
			return nil
		}
		var parsed suite
		if err := yaml.Unmarshal(data, &parsed); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		parsed.Path = path
		found = append(found, parsed)
		return nil
	})
	sort.Slice(found, func(i, j int) bool { return found[i].Path < found[j].Path })
	return found, err
}

type memcpServer struct {
	baseURL string
	binPath string
	dataDir string
	cmd     *exec.Cmd
	out     *syncBuffer
	cancel  context.CancelFunc
}

func startServer(root string) (*memcpServer, error) {
	bin := filepath.Join(os.TempDir(), fmt.Sprintf("costgen-memcp-%d", os.Getpid()))
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("build memcp: %w: %s", err, output)
	}
	dataDir, err := os.MkdirTemp("", "costgen-data-*")
	if err != nil {
		return nil, err
	}
	port, err := freePort()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, bin, "-data", dataDir,
		fmt.Sprintf("--api-port=%d", port), "--disable-mysql", "--no-repl", "lib/main.scm")
	cmd.Dir = root
	out := &syncBuffer{}
	cmd.Stdout, cmd.Stderr = out, out
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}
	server := &memcpServer{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		binPath: bin,
		dataDir: dataDir,
		cmd:     cmd,
		out:     out,
		cancel:  cancel,
	}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(out.String(), fmt.Sprintf(":%d", port)) {
			if _, err := server.execute("/sql/system", "CREATE DATABASE IF NOT EXISTS `"+database+"`", 30*time.Second); err != nil {
				server.stop()
				return nil, fmt.Errorf("create calibration database: %w", err)
			}
			return server, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	server.stop()
	return nil, fmt.Errorf("server did not become ready: %s", out.String())
}

func (s *memcpServer) stop() {
	s.cancel()
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_, _ = s.cmd.Process.Wait()
	}
	_ = os.Remove(s.binPath)
	_ = os.RemoveAll(s.dataDir)
}

func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func (s *memcpServer) execute(endpoint, body string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+endpoint, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth("root", "admin")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	var output bytes.Buffer
	if _, err := output.ReadFrom(response.Body); err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", response.StatusCode, output.String())
	}
	return output.Bytes(), nil
}

func (s *memcpServer) runStep(current step) error {
	if current.SQL != "" {
		_, err := s.execute("/sql/"+database, current.SQL, 10*time.Minute)
		return err
	}
	if current.SCM != "" {
		_, err := s.execute("/scm", current.SCM, 10*time.Minute)
		return err
	}
	return nil
}

func runSuites(server *memcpServer, suites []suite) ([]observation, []calibrationRow, error) {
	var observations []observation
	var allRows []calibrationRow
	for _, currentSuite := range suites {
		logStep("suite %s", currentSuite.Path)
		for _, setup := range currentSuite.Setup {
			if err := server.runStep(setup); err != nil {
				return nil, nil, fmt.Errorf("%s setup: %w", currentSuite.Path, err)
			}
		}
		for _, test := range currentSuite.TestCases {
			if !test.PhysicalCalibration {
				continue
			}
			for _, setup := range test.Setup {
				if err := server.runStep(setup); err != nil {
					return nil, nil, fmt.Errorf("%s/%s setup: %w", currentSuite.Path, test.Name, err)
				}
			}
			query := strings.TrimSpace(test.SQL)
			if query == "" || test.SCM != "" {
				return nil, nil, fmt.Errorf("%s/%s: calibration cases require SQL", currentSuite.Path, test.Name)
			}
			if !strings.HasPrefix(strings.ToUpper(query), "EXPLAIN PHYSICAL CALIBRATE") {
				query = "EXPLAIN PHYSICAL CALIBRATE\n" + query
			}
			warmup, repetitions := test.Warmup, test.Repetitions
			if repetitions <= 0 {
				repetitions = 5
			}
			var runs [][]calibrationRow
			for run := 0; run < warmup+repetitions; run++ {
				payload, err := server.execute("/sql/"+database, query, 10*time.Minute)
				if err != nil {
					return nil, nil, fmt.Errorf("%s/%s: %w", currentSuite.Path, test.Name, err)
				}
				rows, err := decodeRows(payload)
				if err != nil {
					return nil, nil, fmt.Errorf("%s/%s: %w; response=%s", currentSuite.Path, test.Name, err, payload)
				}
				if err := validateRows(rows); err != nil {
					return nil, nil, fmt.Errorf("%s/%s: %w", currentSuite.Path, test.Name, err)
				}
				if run >= warmup {
					runs = append(runs, rows)
					allRows = append(allRows, rows...)
				}
			}
			medians, err := medianRows(runs)
			if err != nil {
				return nil, nil, fmt.Errorf("%s/%s: %w", currentSuite.Path, test.Name, err)
			}
			for _, row := range medians {
				features, err := rowFeatures(row)
				if err != nil {
					return nil, nil, fmt.Errorf("%s/%s: %w", currentSuite.Path, test.Name, err)
				}
				observations = append(observations, observation{caseName: test.Name, plan: row.Plan, y: row.ActualNS, x: features})
			}
		}
		for _, cleanup := range currentSuite.Cleanup {
			if err := server.runStep(cleanup); err != nil {
				return nil, nil, fmt.Errorf("%s cleanup: %w", currentSuite.Path, err)
			}
		}
	}
	return observations, allRows, nil
}

func decodeRows(payload []byte) ([]calibrationRow, error) {
	var rows []calibrationRow
	if err := json.Unmarshal(payload, &rows); err == nil {
		return rows, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	for {
		var row calibrationRow
		if err := decoder.Decode(&row); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return nil, errors.New("empty calibration response")
	}
	return rows, nil
}

func validateRows(rows []calibrationRow) error {
	if len(rows) != 2 {
		return fmt.Errorf("expected exactly two membership alternatives, got %d", len(rows))
	}
	seen := map[string]bool{}
	for _, row := range rows {
		if row.Decision != "membership_carrier" || row.DecisionID == "" {
			return fmt.Errorf("unexpected or anonymous decision: %+v", row)
		}
		if !row.OperatorConsistent || row.OperatorFamily != row.Plan {
			return fmt.Errorf("chosen/emitted mismatch for %s: chosen=%s emitted=%s", row.DecisionID, row.Plan, row.OperatorFamily)
		}
		if !row.ResultEqual {
			return fmt.Errorf("forced alternative %s changed the result", row.Plan)
		}
		if row.CandidateInputRows == nil || row.CandidateRows == nil || row.DriverRows == nil || row.EstimatedNS == nil {
			return fmt.Errorf("known-statistics row contains nil inputs: %+v", row)
		}
		if row.ActualNS <= 0 {
			return fmt.Errorf("invalid actual_ns for %s", row.Plan)
		}
		seen[row.Plan] = true
	}
	if !seen["candidate_keyset"] || !seen["driver_order_membership_probe"] {
		return fmt.Errorf("alternatives incomplete: %v", seen)
	}
	return nil
}

func medianRows(runs [][]calibrationRow) ([]calibrationRow, error) {
	byPlan := map[string][]calibrationRow{}
	for _, run := range runs {
		for _, row := range run {
			byPlan[row.Plan] = append(byPlan[row.Plan], row)
		}
	}
	var result []calibrationRow
	for _, plan := range []string{"candidate_keyset", "driver_order_membership_probe"} {
		rows := byPlan[plan]
		if len(rows) == 0 {
			return nil, fmt.Errorf("no rows for %s", plan)
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].ActualNS < rows[j].ActualNS })
		result = append(result, rows[len(rows)/2])
	}
	return result, nil
}

func rowFeatures(row calibrationRow) ([]float64, error) {
	switch row.Plan {
	case "candidate_keyset":
		return []float64{*row.CandidateInputRows, *row.CandidateRows, 0, 0}, nil
	case "driver_order_membership_probe":
		return []float64{*row.CandidateInputRows, 0, *row.CandidateRows, *row.DriverRows}, nil
	default:
		return nil, fmt.Errorf("unsupported plan %q", row.Plan)
	}
}

// solve fits the physical work both alternatives actually perform:
//
//	candidate = candidate_input*scan + candidate_rows*recset_build
//	driver    = candidate_input*scan + candidate_rows*cache_build + driver_rows*cache_probe
//
// Keeping candidate materialization in the driver equation prevents its cost
// from being incorrectly absorbed into a workload-specific per-driver probe.
// A shared startup is deliberately not fitted: it cancels from the plan
// inequality and otherwise absorbs noise while making the row constants
// underdetermined.
// Non-negative coordinate descent
// prevents noisy samples from generating impossible negative costs.
func solve(rows []observation) (constants, error) {
	if len(rows) < 5 {
		return constants{}, fmt.Errorf("need at least five observations, got %d", len(rows))
	}
	beta := []float64{1, 1, 1, 1}
	for iteration := 0; iteration < 10000; iteration++ {
		largestChange := 0.0
		for column := range beta {
			numerator, denominator := 0.0, 0.0
			for _, row := range rows {
				residual := row.y
				for other, value := range beta {
					if other != column {
						residual -= row.x[other] * value
					}
				}
				numerator += row.x[column] * residual
				denominator += row.x[column] * row.x[column]
			}
			if denominator == 0 {
				return constants{}, fmt.Errorf("cost equation column %d is not covered", column)
			}
			next := numerator / denominator
			if next < 1 {
				next = 1
			}
			change := next - beta[column]
			if change < 0 {
				change = -change
			}
			if change > largestChange {
				largestChange = change
			}
			beta[column] = next
		}
		if largestChange < 0.001 {
			break
		}
	}
	return constants{
		startupNS:             0,
		candidateScanRowNS:    int64(math.Round(beta[0])),
		candidateRecsetRowNS:  int64(math.Round(beta[1])),
		driverCacheBuildRowNS: int64(math.Round(beta[2])),
		driverCacheProbeRowNS: int64(math.Round(beta[3])),
	}, nil
}

func estimatedNS(row observation, c constants) float64 {
	beta := []float64{
		float64(c.candidateScanRowNS),
		float64(c.candidateRecsetRowNS),
		float64(c.driverCacheBuildRowNS),
		float64(c.driverCacheProbeRowNS),
	}
	total := 0.0
	for i, value := range row.x {
		total += value * beta[i]
	}
	return total
}

type decisionPair struct {
	candidate observation
	driver    observation
}

func decisionPairs(rows []observation) (map[string]decisionPair, error) {
	pairs := make(map[string]decisionPair)
	seen := make(map[string]map[string]bool)
	for _, row := range rows {
		pair := pairs[row.caseName]
		if seen[row.caseName] == nil {
			seen[row.caseName] = make(map[string]bool)
		}
		if seen[row.caseName][row.plan] {
			return nil, fmt.Errorf("duplicate %s observation for %q", row.plan, row.caseName)
		}
		seen[row.caseName][row.plan] = true
		switch row.plan {
		case "candidate_keyset":
			pair.candidate = row
		case "driver_order_membership_probe":
			pair.driver = row
		default:
			return nil, fmt.Errorf("unsupported plan %q", row.plan)
		}
		pairs[row.caseName] = pair
	}
	for name, plans := range seen {
		if !plans["candidate_keyset"] || !plans["driver_order_membership_probe"] {
			return nil, fmt.Errorf("incomplete alternative pair for %q", name)
		}
	}
	return pairs, nil
}

func validateDecisionOrdering(rows []observation, c constants) error {
	pairs, err := decisionPairs(rows)
	if err != nil {
		return err
	}
	for name, pair := range pairs {
		actualCandidateWins := pair.candidate.y < pair.driver.y
		estimatedCandidateWins := estimatedNS(pair.candidate, c) < estimatedNS(pair.driver, c)
		if actualCandidateWins != estimatedCandidateWins {
			return fmt.Errorf("calibrated inequality disagrees for %q: actual candidate=%0.fns driver=%0.fns, estimated candidate=%0.fns driver=%0.fns",
				name, pair.candidate.y, pair.driver.y,
				estimatedNS(pair.candidate, c), estimatedNS(pair.driver, c))
		}
	}
	return nil
}

func printDecisionOrdering(rows []observation, c constants) {
	pairs, err := decisionPairs(rows)
	if err != nil {
		return
	}
	names := make([]string, 0, len(pairs))
	for name := range pairs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		pair := pairs[name]
		estimatedCandidate := estimatedNS(pair.candidate, c)
		estimatedDriver := estimatedNS(pair.driver, c)
		actualWinner := pair.driver.plan
		if pair.candidate.y < pair.driver.y {
			actualWinner = pair.candidate.plan
		}
		estimatedWinner := pair.driver.plan
		if estimatedCandidate < estimatedDriver {
			estimatedWinner = pair.candidate.plan
		}
		fmt.Printf("decision %-40s actual=%-30s driver-candidate=%+.3fms estimated=%-30s driver-candidate=%+.3fms\n",
			name, actualWinner, (pair.driver.y-pair.candidate.y)/1e6,
			estimatedWinner, (estimatedDriver-estimatedCandidate)/1e6)
	}
}

func writeJSONL(path string, rows []calibrationRow) error {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	for _, row := range rows {
		if err := encoder.Encode(row); err != nil {
			return err
		}
	}
	return os.WriteFile(path, output.Bytes(), 0o644)
}

func patchQueryplan(path string, c constants) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	begin := strings.Index(content, beginMarker)
	end := strings.Index(content, endMarker)
	if begin < 0 || end < begin {
		return errors.New("generated cost constants block not found")
	}
	end += len(endMarker)
	block := fmt.Sprintf(`/* BEGIN GENERATED COST CONSTANTS. DO NEVER MANUALLY EDIT THIS SECTION. RUN make costgen TO UPDATE.
Calibrated by tools/costgen from tests/**/*.yaml workloads tagged with
metadata.physical_calibration. Each observation is an executed, forced
EXPLAIN PHYSICAL CALIBRATE alternative with result and operator validation. */
(define planner_direct_presence_probe_cost (lambda (probe_rows)
	(planner_cost 0 0 (* probe_rows 48685) 0 0 0 0 0 probe_rows 0.75)))

(define planner_presence_carrier_cost (lambda (domain_rows probe_rows)
	(planner_cost 1421611 (* probe_rows 136938) 0 0 0 0
		(* domain_rows 8) 0 domain_rows 0.65)))

(define planner_recset_carrier_cost (lambda (domain_rows probe_rows)
	(planner_cost 365607 0 0 0 0 (* probe_rows 17681)
		(* probe_rows 1) 0 probe_rows 0.6)))

(define planner_membership_startup_ns %d)
(define planner_membership_candidate_scan_row_ns %d)
(define planner_membership_recset_build_row_ns %d)
(define planner_membership_group_cache_build_row_ns %d)
(define planner_membership_group_cache_probe_row_ns %d)
/* END GENERATED COST CONSTANTS */`, c.startupNS, c.candidateScanRowNS,
		c.candidateRecsetRowNS, c.driverCacheBuildRowNS, c.driverCacheProbeRowNS)
	return os.WriteFile(path, []byte(content[:begin]+block+content[end:]), 0o644)
}
