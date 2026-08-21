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
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	beginMarker = "/* BEGIN GENERATED COST CONSTANTS."
	endMarker   = "/* END GENERATED COST CONSTANTS */"
	database    = "memcp-tests"
)

var activeServer *memcpServer

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
	PhysicalCalibration bool   `yaml:"physical_calibration"`
	CacheState          string `yaml:"cache_state"`
	CompileState        string `yaml:"compile_state"`
}

type step struct {
	SQL string `yaml:"sql"`
	SCM string `yaml:"scm"`
}

type testCase struct {
	Name                    string  `yaml:"name"`
	SQL                     string  `yaml:"sql"`
	SCM                     string  `yaml:"scm"`
	PhysicalCalibration     bool    `yaml:"physical_calibration"`
	CalibrationHoldout      bool    `yaml:"calibration_holdout"`
	CalibrationDecision     string  `yaml:"calibration_decision"`
	Race                    bool    `yaml:"race"`
	RaceGrace               float64 `yaml:"race_grace"`
	CacheState              string  `yaml:"cache_state"`
	CompileState            string  `yaml:"compile_state"`
	ExpectedDriverInputRows float64 `yaml:"expected_driver_input_rows"`
	Warmup                  int     `yaml:"warmup"`
	Repetitions             int     `yaml:"repetitions"`
	Setup                   []step  `yaml:"setup"`
}

type suite struct {
	Path      string
	Metadata  metadata   `yaml:"metadata"`
	Setup     []step     `yaml:"setup"`
	Cleanup   []step     `yaml:"cleanup"`
	TestCases []testCase `yaml:"test_cases"`
}

type calibrationRow struct {
	CaseName                         string   `json:"case_name"`
	Error                            string   `json:"error"`
	CacheState                       string   `json:"cache_state"`
	CompileState                     string   `json:"compile_state"`
	DecisionID                       string   `json:"decision_id"`
	Decision                         string   `json:"decision"`
	Consumer                         string   `json:"consumer"`
	Plan                             string   `json:"plan"`
	OperatorFamily                   string   `json:"operator_family"`
	OperatorConsistent               bool     `json:"operator_consistent"`
	EstimatedNS                      *float64 `json:"estimated_ns"`
	WholeQueryExecutionNS            float64  `json:"whole_query_execution_ns"`
	OperatorNS                       *float64 `json:"operator_ns"`
	TimedOut                         bool     `json:"timed_out"`
	LowerBoundNS                     float64  `json:"lower_bound_ns"`
	CandidateInputRows               *float64 `json:"candidate_input_rows"`
	CandidateRows                    *float64 `json:"candidate_rows"`
	CandidateDensity                 *float64 `json:"candidate_density"`
	ProjectedDriverRows              *float64 `json:"projected_driver_rows"`
	DriverInputRows                  *float64 `json:"driver_input_rows"`
	DriverRows                       *float64 `json:"driver_rows"`
	ExpectedDriverRowsVisited        *float64 `json:"expected_driver_rows_visited"`
	Limit                            *float64 `json:"limit"`
	Offset                           *float64 `json:"offset"`
	ProbeBranches                    *float64 `json:"probe_branches"`
	CandidateScanInvocations         *float64 `json:"candidate_scan_invocations"`
	CandidateFilterColumns           *float64 `json:"candidate_filter_columns"`
	CandidateMapColumns              *float64 `json:"candidate_map_columns"`
	CandidateCacheMapColumns         *float64 `json:"candidate_cache_map_columns"`
	CandidateExpressionOperations    *float64 `json:"candidate_expression_operations"`
	CandidateExpressionDepth         *float64 `json:"candidate_expression_depth"`
	CandidateFilterValueRows         *float64 `json:"candidate_filter_value_rows"`
	CandidateExpressionOperationRows *float64 `json:"candidate_expression_operation_rows"`
	DriverScanInvocations            *float64 `json:"driver_scan_invocations"`
	DriverFilterColumns              *float64 `json:"driver_filter_columns"`
	DriverMapColumns                 *float64 `json:"driver_map_columns"`
	DriverExpressionOperations       *float64 `json:"driver_expression_operations"`
	DriverExpressionDepth            *float64 `json:"driver_expression_depth"`
	ResultEqual                      bool     `json:"result_equal"`
	Rows                             int64    `json:"rows"`
	ResultHash                       string   `json:"result_hash"`
}

type calibrationDiscovery struct {
	Error        string     `json:"error"`
	DecisionID   string     `json:"decision_id"`
	Decision     string     `json:"decision"`
	Alternatives []string   `json:"alternatives"`
	EstimatedNS  []*float64 `json:"estimated_ns"`
}

type observation struct {
	caseName        string
	plan            string
	y               float64
	currentEstimate float64
	holdout         bool
	noiseNS         float64
	censored        bool
	x               []float64
}

type constants struct {
	scanInvocationNS      int64
	scanRowNS             int64
	filterColumnRowNS     int64
	mapColumnRowNS        int64
	expressionOperationNS int64
	recsetStartupNS       int64
	recsetBuildRowNS      int64
	recsetProbeRowNS      int64
	recsetAggregateRowNS  int64
	groupCacheStartupNS   int64
	groupCacheBuildRowNS  int64
	groupCacheProbeRowNS  int64
	orderedDriverInputNS  int64
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
	queryplanPath := filepath.Join(root, "lib", "queryplan.scm")
	currentConstants, err := readCurrentConstants(queryplanPath)
	if err != nil {
		fatal(err)
	}
	server, err := startServer(root)
	if err != nil {
		fatal(err)
	}
	activeServer = server
	defer server.stop()
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupts)
	go func() {
		<-interrupts
		server.stop()
		os.Exit(130)
	}()

	observations, raw, err := runSuites(server, suites)
	if err != nil {
		fatal(err)
	}
	if *jsonl != "" {
		if err := writeJSONL(*jsonl, raw); err != nil {
			fatal(err)
		}
	}
	training := filterObservations(observations, false)
	fitTraining := filterCompleteExactPairs(training)
	if err := validateMeasurementSignal(fitTraining); err != nil {
		fatal(err)
	}
	c, err := solve(fitTraining, training, currentConstants)
	if err != nil {
		fatal(err)
	}
	if err := validateDecisionOrdering(training, c); err != nil {
		fatal(err)
	}
	fmt.Printf("scan invocation:      %d ns/invocation\nscan row:             %d ns/input-row\nfilter column:        %d ns/value\nmap column:           %d ns/value\nexpression operation: %d ns/row-operation\nrecset startup:       %d ns\nrecset build:         %d ns/matching-row\nrecset probe:         %d ns/driver-row\nrecset aggregate:     %d ns/driver-input-row\ngroup-cache startup:  %d ns\ngroup-cache build:    %d ns/matching-row\ngroup-cache probe:    %d ns/driver-row\nordered driver input: %d ns/(rows²/1M)\n",
		c.scanInvocationNS, c.scanRowNS, c.filterColumnRowNS, c.mapColumnRowNS,
		c.expressionOperationNS, c.recsetStartupNS, c.recsetBuildRowNS, c.recsetProbeRowNS,
		c.recsetAggregateRowNS, c.groupCacheStartupNS, c.groupCacheBuildRowNS,
		c.groupCacheProbeRowNS, c.orderedDriverInputNS)
	printModelComparison("training", training, c)
	holdout := filterObservations(observations, true)
	if len(holdout) > 0 {
		printModelComparison("holdout", holdout, c)
		if err := validateDecisionOrdering(holdout, c); err != nil {
			fatal(fmt.Errorf("holdout: %w", err))
		}
		if err := validateModelImprovement(holdout, c); err != nil {
			fatal(fmt.Errorf("holdout: %w", err))
		}
	}
	printDecisionOrdering(observations, c)
	if *patch {
		if err := patchQueryplan(queryplanPath, c); err != nil {
			fatal(err)
		}
		fmt.Println("patched", queryplanPath)
	}
}

func validateModelImprovement(rows []observation, c constants) error {
	current := measureModelError(rows, func(row observation) float64 { return row.currentEstimate })
	updated := measureModelError(rows, func(row observation) float64 { return estimatedNS(row, c) })
	if updated.medianAbsolutePercent > current.medianAbsolutePercent || updated.meanFactor > current.meanFactor {
		return fmt.Errorf("updated model does not improve exact holdouts: median %.1f%% -> %.1f%%, mean factor %.2fx -> %.2fx",
			current.medianAbsolutePercent, updated.medianAbsolutePercent,
			current.meanFactor, updated.meanFactor)
	}
	return nil
}

func fatal(err error) {
	if activeServer != nil {
		activeServer.stop()
		activeServer = nil
	}
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
	baseURL  string
	binPath  string
	dataDir  string
	cmd      *exec.Cmd
	out      *syncBuffer
	cancel   context.CancelFunc
	stopOnce sync.Once
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
	s.stopOnce.Do(func() {
		s.cancel()
		if s.cmd.Process != nil {
			_ = s.cmd.Process.Kill()
			_, _ = s.cmd.Process.Wait()
		}
		_ = os.Remove(s.binPath)
		_ = os.RemoveAll(s.dataDir)
	})
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
	return s.executeContext(ctx, endpoint, body)
}

func (s *memcpServer) executeContext(ctx context.Context, endpoint, body string) ([]byte, error) {
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
		_, err := s.execute("/sql/"+database, current.SQL, 30*time.Minute)
		return err
	}
	if current.SCM != "" {
		_, err := s.execute("/scm", current.SCM, 30*time.Minute)
		return err
	}
	return nil
}

func runSuites(server *memcpServer, suites []suite) ([]observation, []calibrationRow, error) {
	var observations []observation
	var allRows []calibrationRow
	for _, currentSuite := range suites {
		if currentSuite.Metadata.CacheState == "" || currentSuite.Metadata.CompileState == "" {
			return nil, nil, fmt.Errorf("%s: calibration metadata requires cache_state and compile_state", currentSuite.Path)
		}
		logStep("suite %s (%s, %s)", currentSuite.Path,
			currentSuite.Metadata.CacheState, currentSuite.Metadata.CompileState)
		for _, setup := range currentSuite.Setup {
			if err := server.runStep(setup); err != nil {
				return nil, nil, fmt.Errorf("%s setup: %w; server output: %s",
					currentSuite.Path, err, tail(server.out.String(), 8000))
			}
		}
		for _, test := range currentSuite.TestCases {
			if !test.PhysicalCalibration {
				continue
			}
			logStep("measure %s", test.Name)
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
			cacheState, compileState := currentSuite.Metadata.CacheState, currentSuite.Metadata.CompileState
			if test.CacheState != "" {
				cacheState = test.CacheState
			}
			if test.CompileState != "" {
				compileState = test.CompileState
			}
			if repetitions <= 0 {
				repetitions = 5
			}
			var runs [][]calibrationRow
			if test.Race {
				baseQuery := strings.TrimSpace(strings.TrimPrefix(query, "EXPLAIN PHYSICAL CALIBRATE"))
				var raceRows []calibrationRow
				raceRuns, raceRows, raceErr := runCalibrationRaces(server, baseQuery, test, repetitions)
				if raceErr != nil {
					return nil, nil, fmt.Errorf("%s/%s: %w", currentSuite.Path, test.Name, raceErr)
				}
				runs = raceRuns
				for rowIndex := range raceRows {
					raceRows[rowIndex].CaseName = test.Name
					raceRows[rowIndex].CacheState = cacheState
					raceRows[rowIndex].CompileState = compileState
				}
				allRows = append(allRows, raceRows...)
			} else {
				for run := 0; run < warmup+repetitions; run++ {
					payload, err := server.execute("/sql/"+database, query, 10*time.Minute)
					if err != nil {
						return nil, nil, fmt.Errorf("%s/%s: %w", currentSuite.Path, test.Name, err)
					}
					rows, err := decodeRows(payload)
					if err != nil {
						return nil, nil, fmt.Errorf("%s/%s: %w; response=%s", currentSuite.Path, test.Name, err, payload)
					}
					decision := test.CalibrationDecision
					if decision == "" {
						decision = "membership_carrier"
					}
					rows = filterCalibrationDecision(rows, decision)
					if err := validateRows(rows); err != nil {
						return nil, nil, fmt.Errorf("%s/%s: %w", currentSuite.Path, test.Name, err)
					}
					if run >= warmup {
						for rowIndex := range rows {
							rows[rowIndex].CaseName = test.Name
							rows[rowIndex].CacheState = cacheState
							rows[rowIndex].CompileState = compileState
						}
						runs = append(runs, rows)
						allRows = append(allRows, rows...)
					}
				}
			}
			medians, err := medianRows(runs)
			if err != nil {
				return nil, nil, fmt.Errorf("%s/%s: %w", currentSuite.Path, test.Name, err)
			}
			if test.ExpectedDriverInputRows > 0 {
				for _, row := range medians {
					if row.DriverInputRows == nil || *row.DriverInputRows != test.ExpectedDriverInputRows {
						actual := "nil"
						if row.DriverInputRows != nil {
							actual = fmt.Sprintf("%.0f", *row.DriverInputRows)
						}
						return nil, nil, fmt.Errorf("%s/%s: driver_input_rows=%s, want %.0f",
							currentSuite.Path, test.Name, actual, test.ExpectedDriverInputRows)
					}
				}
			}
			for _, row := range medians {
				features, err := rowFeatures(row)
				if err != nil {
					return nil, nil, fmt.Errorf("%s/%s: %w", currentSuite.Path, test.Name, err)
				}
				observations = append(observations, observation{
					caseName: test.Name, plan: row.Plan, y: row.WholeQueryExecutionNS,
					currentEstimate: *row.EstimatedNS, holdout: test.CalibrationHoldout,
					noiseNS: medianAbsoluteDeviation(runs, row.Plan), censored: row.TimedOut, x: features,
				})
			}
			logStep("measured %s", test.Name)
		}
		for _, cleanup := range currentSuite.Cleanup {
			if err := server.runStep(cleanup); err != nil {
				return nil, nil, fmt.Errorf("%s cleanup: %w", currentSuite.Path, err)
			}
		}
	}
	return observations, allRows, nil
}

func tail(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[len(value)-limit:]
}

type calibrationVariantResult struct {
	plan    string
	row     calibrationRow
	elapsed time.Duration
	err     error
}

func runCalibrationRaces(server *memcpServer, query string, test testCase, repetitions int) ([][]calibrationRow, []calibrationRow, error) {
	discoveryPayload, err := server.execute("/sql/"+database,
		"EXPLAIN PHYSICAL CALIBRATE DISCOVER\n"+query, 10*time.Minute)
	if err != nil {
		return nil, nil, err
	}
	discovered, err := decodeDiscoveries(discoveryPayload)
	if err != nil {
		return nil, nil, fmt.Errorf("decode calibration discovery: %w; response=%s", err, discoveryPayload)
	}
	decisionName := test.CalibrationDecision
	if decisionName == "" {
		decisionName = "membership_carrier"
	}
	var decision *calibrationDiscovery
	for i := range discovered {
		if discovered[i].Error != "" {
			return nil, nil, errors.New(discovered[i].Error)
		}
		if discovered[i].Decision == decisionName {
			if decision != nil {
				return nil, nil, fmt.Errorf("race requires one %s decision, found several", decisionName)
			}
			decision = &discovered[i]
		}
	}
	if decision == nil || len(decision.Alternatives) != 2 || len(decision.EstimatedNS) != 2 {
		return nil, nil, fmt.Errorf("race discovery did not expose two costed alternatives: %+v", decision)
	}
	estimates := map[string]*float64{}
	for i, plan := range decision.Alternatives {
		estimates[plan] = decision.EstimatedNS[i]
	}
	grace := test.RaceGrace
	if grace <= 0 {
		grace = 0.5
	}
	var runs [][]calibrationRow
	var raw []calibrationRow
	for run := 0; run < repetitions; run++ {
		rows, err := raceCalibrationVariants(server, query, decision.DecisionID,
			decision.Alternatives, estimates, grace)
		if err != nil {
			return nil, nil, err
		}
		runs = append(runs, rows)
		raw = append(raw, rows...)
	}
	return runs, raw, nil
}

func decodeDiscoveries(payload []byte) ([]calibrationDiscovery, error) {
	var rows []calibrationDiscovery
	if err := json.Unmarshal(payload, &rows); err == nil {
		return rows, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	for {
		var row calibrationDiscovery
		if err := decoder.Decode(&row); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return nil, errors.New("empty calibration discovery")
	}
	return rows, nil
}

func raceCalibrationVariants(server *memcpServer, query, decisionID string, plans []string, estimates map[string]*float64, grace float64) ([]calibrationRow, error) {
	result := make(chan calibrationVariantResult, 2)
	cancels := make(map[string]context.CancelFunc, len(plans))
	for _, plan := range plans {
		ctx, cancel := context.WithCancel(context.Background())
		cancels[plan] = cancel
		go func(plan string) {
			started := time.Now()
			statement := "EXPLAIN PHYSICAL CALIBRATE VARIANT '" + sqlQuote(decisionID) + "' '" + sqlQuote(plan) + "'\n" + query
			payload, err := server.executeContext(ctx, "/sql/"+database, statement)
			row := calibrationRow{}
			if err == nil {
				rows, decodeErr := decodeRows(payload)
				if decodeErr != nil || len(rows) != 1 {
					err = fmt.Errorf("decode %s variant: %v; response=%s", plan, decodeErr, payload)
				} else {
					row = rows[0]
				}
			}
			result <- calibrationVariantResult{plan: plan, row: row, elapsed: time.Since(started), err: err}
		}(plan)
	}
	first := <-result
	if first.err != nil {
		for _, cancel := range cancels {
			cancel()
		}
		return nil, first.err
	}
	if err := validateRaceWinner(first.row, decisionID, first.plan); err != nil {
		return nil, err
	}
	remainingPlan := plans[0]
	if remainingPlan == first.plan {
		remainingPlan = plans[1]
	}
	remaining := time.Duration(float64(first.elapsed) * grace)
	if remaining < time.Millisecond {
		remaining = time.Millisecond
	}
	timer := time.NewTimer(remaining)
	defer timer.Stop()
	var second calibrationVariantResult
	select {
	case second = <-result:
		if second.err != nil {
			return nil, second.err
		}
		if err := validateRaceWinner(second.row, decisionID, second.plan); err != nil {
			return nil, err
		}
		equal := first.row.Rows == second.row.Rows && first.row.ResultHash == second.row.ResultHash
		first.row.ResultEqual, second.row.ResultEqual = equal, equal
		if !equal {
			return nil, fmt.Errorf("raced variants returned different results")
		}
	case <-timer.C:
		cancels[remainingPlan]()
		second = <-result
		lowerBound := float64(first.elapsed+remaining) * float64(time.Nanosecond)
		second = calibrationVariantResult{plan: remainingPlan, row: first.row}
		second.row.Plan = remainingPlan
		second.row.OperatorFamily = remainingPlan
		second.row.EstimatedNS = estimates[remainingPlan]
		second.row.WholeQueryExecutionNS = lowerBound
		second.row.TimedOut = true
		second.row.LowerBoundNS = lowerBound
		second.row.ResultEqual = false
		second.row.ResultHash = ""
		second.row.Rows = 0
	}
	for _, cancel := range cancels {
		cancel()
	}
	rows := []calibrationRow{first.row, second.row}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Plan < rows[j].Plan })
	return rows, nil
}

func validateRaceWinner(row calibrationRow, decisionID, plan string) error {
	if row.Error != "" {
		return errors.New(row.Error)
	}
	if row.DecisionID != decisionID || row.Plan != plan || !row.OperatorConsistent || row.OperatorFamily != plan {
		return fmt.Errorf("forced race variant did not emit the requested operator: %+v", row)
	}
	if row.EstimatedNS == nil || row.CandidateInputRows == nil || row.CandidateRows == nil ||
		row.DriverInputRows == nil || row.DriverRows == nil || row.ExpectedDriverRowsVisited == nil ||
		row.WholeQueryExecutionNS <= 0 {
		return fmt.Errorf("forced race variant has incomplete measurements: %+v", row)
	}
	return nil
}

func sqlQuote(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func medianAbsoluteDeviation(runs [][]calibrationRow, plan string) float64 {
	values := make([]float64, 0, len(runs))
	for _, run := range runs {
		for _, row := range run {
			if row.Plan == plan && !row.TimedOut {
				values = append(values, row.WholeQueryExecutionNS)
			}
		}
	}
	if len(values) == 0 {
		return 0
	}
	sort.Float64s(values)
	median := values[len(values)/2]
	deviations := make([]float64, len(values))
	for i, value := range values {
		deviations[i] = math.Abs(value - median)
	}
	sort.Float64s(deviations)
	return deviations[len(deviations)/2]
}

func filterCalibrationDecision(rows []calibrationRow, decision string) []calibrationRow {
	filtered := make([]calibrationRow, 0, len(rows))
	for _, row := range rows {
		if row.Error != "" || row.Decision == decision {
			filtered = append(filtered, row)
		}
	}
	return filtered
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
	for _, row := range rows {
		if row.Error != "" {
			return errors.New(row.Error)
		}
	}
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
		if row.CandidateInputRows == nil || row.CandidateRows == nil || row.CandidateDensity == nil ||
			row.ProjectedDriverRows == nil || row.DriverInputRows == nil || row.DriverRows == nil ||
			row.ExpectedDriverRowsVisited == nil || row.ProbeBranches == nil || row.EstimatedNS == nil {
			return fmt.Errorf("known-statistics row contains nil inputs: %+v", row)
		}
		workInputs := []*float64{
			row.CandidateScanInvocations, row.CandidateFilterColumns,
			row.CandidateMapColumns, row.CandidateCacheMapColumns,
			row.CandidateExpressionOperations, row.CandidateExpressionDepth,
			row.DriverScanInvocations, row.DriverFilterColumns,
			row.DriverMapColumns, row.DriverExpressionOperations,
			row.DriverExpressionDepth,
		}
		for _, input := range workInputs {
			if input == nil {
				return fmt.Errorf("physical work profile contains nil inputs: %+v", row)
			}
		}
		if row.WholeQueryExecutionNS <= 0 {
			return fmt.Errorf("invalid whole_query_execution_ns for %s", row.Plan)
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
		sort.Slice(rows, func(i, j int) bool {
			return rows[i].WholeQueryExecutionNS < rows[j].WholeQueryExecutionNS
		})
		result = append(result, rows[len(rows)/2])
	}
	return result, nil
}

func rowFeatures(row calibrationRow) ([]float64, error) {
	scanInvocations := *row.CandidateScanInvocations + *row.DriverScanInvocations
	driverWorkRows := *row.DriverInputRows
	if row.Plan == "driver_order_membership_probe" {
		driverWorkRows = *row.ExpectedDriverRowsVisited
	}
	scanRows := *row.CandidateInputRows + driverWorkRows
	candidateFilterValues := *row.CandidateInputRows * *row.CandidateFilterColumns
	if row.CandidateFilterValueRows != nil {
		candidateFilterValues = *row.CandidateFilterValueRows
	}
	candidateExpressionOperations := *row.CandidateInputRows * *row.CandidateExpressionOperations
	if row.CandidateExpressionOperationRows != nil {
		candidateExpressionOperations = *row.CandidateExpressionOperationRows
	}
	filterValues := candidateFilterValues + driverWorkRows**row.DriverFilterColumns
	expressionOperations := candidateExpressionOperations + driverWorkRows**row.DriverExpressionOperations
	mapColumns := *row.CandidateMapColumns
	if row.Plan == "driver_order_membership_probe" {
		mapColumns = *row.CandidateCacheMapColumns
	}
	driverMapRows := *row.ProjectedDriverRows
	if row.Plan == "driver_order_membership_probe" {
		driverMapRows = *row.DriverRows
	}
	mapValues := *row.CandidateRows*mapColumns + driverMapRows**row.DriverMapColumns
	aggregateDriverRows := 0.0
	if row.Consumer == "aggregate" {
		aggregateDriverRows = *row.DriverInputRows
	}
	orderedDriverInputRows := 0.0
	if row.Limit != nil {
		orderedDriverInputRows = *row.DriverInputRows * *row.DriverInputRows / 1_000_000
	}
	switch row.Plan {
	case "candidate_keyset":
		return []float64{
			scanInvocations, scanRows, filterValues, mapValues, expressionOperations,
			1, *row.CandidateRows + *row.ProjectedDriverRows, *row.DriverRows, 0, 0, 0,
			aggregateDriverRows, 0,
		}, nil
	case "driver_order_membership_probe":
		return []float64{
			scanInvocations, scanRows, filterValues, mapValues, expressionOperations,
			0, 0, 0, 1, *row.CandidateRows, *row.ExpectedDriverRowsVisited,
			0, orderedDriverInputRows,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported plan %q", row.Plan)
	}
}

// solveEquationSystem fits the physical work both alternatives actually perform:
//
//	common = scan_invocations*startup + scan_rows*scan
//	       + filter_values*filter_column + map_values*map_column
//	       + expression_operation_rows*expression_operation
//	candidate = recset_startup + common
//	          + candidate_rows*recset_build + driver_rows*recset_probe
//	driver    = group_cache_startup + common
//	          + candidate_rows*cache_build + driver_rows*cache_probe
//
// Non-negative coordinate descent prevents noisy samples from generating
// impossible negative costs. Weighting every equation by 1/actual^2 minimizes
// relative rather than absolute error: otherwise multi-second driver probes
// drown out the millisecond candidate plans whose inequality we also need to
// predict correctly.
func solveEquationSystem(rows []observation) (constants, error) {
	if len(rows) < 11 {
		return constants{}, fmt.Errorf("need at least eleven training observations, got %d", len(rows))
	}
	candidateX, candidateY := make([][]float64, 0), make([]float64, 0)
	for _, row := range rows {
		if row.plan != "candidate_keyset" {
			continue
		}
		candidateX = append(candidateX, []float64{
			row.x[0], row.x[1], row.x[2], row.x[3], row.x[4], row.x[6],
		})
		// Startup and the final contains check stay at their conservative one-ns
		// floor. Projection/build rows, unlike whole-query startup, vary across
		// the workload and are therefore identifiable from exact winner runs.
		candidateY = append(candidateY, math.Max(1, row.y-row.x[5]-row.x[7]))
	}
	common, err := fitNonnegative(candidateX, candidateY)
	if err != nil {
		return constants{}, fmt.Errorf("common candidate work: %w", err)
	}
	pairs, err := decisionPairs(rows)
	if err != nil {
		return constants{}, err
	}
	groupX, groupY := make([][]float64, 0, len(pairs)), make([]float64, 0, len(pairs))
	for _, pair := range pairs {
		commonDifference := 0.0
		for i := 0; i < 5; i++ {
			commonDifference += (pair.driver.x[i] - pair.candidate.x[i]) * common[i]
		}
		candidateCarrier := pair.candidate.x[5] + pair.candidate.x[6]*common[5] + pair.candidate.x[7]
		groupX = append(groupX, pair.driver.x[8:11])
		groupY = append(groupY, math.Max(1,
			pair.driver.y-pair.candidate.y-commonDifference+candidateCarrier))
	}
	group, err := fitNonnegative(groupX, groupY)
	if err != nil {
		return constants{}, fmt.Errorf("paired carrier difference: %w", err)
	}
	beta := []float64{
		common[0], common[1], common[2], common[3], common[4],
		1, common[5], 1, group[0], group[1], group[2],
	}
	return constants{
		scanInvocationNS:      int64(math.Round(beta[0])),
		scanRowNS:             int64(math.Round(beta[1])),
		filterColumnRowNS:     int64(math.Round(beta[2])),
		mapColumnRowNS:        int64(math.Round(beta[3])),
		expressionOperationNS: int64(math.Round(beta[4])),
		recsetStartupNS:       int64(math.Round(beta[5])),
		recsetBuildRowNS:      int64(math.Round(beta[6])),
		recsetProbeRowNS:      int64(math.Round(beta[7])),
		groupCacheStartupNS:   int64(math.Round(beta[8])),
		groupCacheBuildRowNS:  int64(math.Round(beta[9])),
		groupCacheProbeRowNS:  int64(math.Round(beta[10])),
		recsetAggregateRowNS:  1,
		orderedDriverInputNS:  1,
	}, nil
}

// solve keeps an already calibrated base model unless a fresh unconstrained
// equation fit improves both its numerical error and its plan inequalities.
// The two large-shape additions are fitted separately because a cancelled race
// yields an inequality, not an exact duration: y_slow is only known to exceed
// lower_bound_ns.
func solve(exactRows, allRows []observation, baseline constants) (constants, error) {
	fitted, err := solveEquationSystem(exactRows)
	if err != nil {
		return constants{}, err
	}
	selected := baseline
	baseError := measureModelError(exactRows, func(row observation) float64 {
		return estimatedNS(row, baseline)
	})
	fitError := measureModelError(exactRows, func(row observation) float64 {
		return estimatedNS(row, fitted)
	})
	baseCorrect, _ := decisionAccuracy(exactRows, func(row observation) float64 {
		return estimatedNS(row, baseline)
	})
	fitCorrect, _ := decisionAccuracy(exactRows, func(row observation) float64 {
		return estimatedNS(row, fitted)
	})
	if fitError.medianAbsolutePercent <= baseError.medianAbsolutePercent &&
		fitError.meanFactor <= baseError.meanFactor && fitCorrect >= baseCorrect {
		selected = fitted
	}

	if value, ok := fitExactResidualPerRow(allRows, selected, 11); ok {
		selected.recsetAggregateRowNS = value
	}
	// A timed-out slower ordered alternative contributes lower_bound <= cost.
	// Choose the smallest coefficient satisfying every observed constraint.
	for _, row := range allRows {
		if !row.censored || len(row.x) <= 12 || row.x[12] <= 0 {
			continue
		}
		without := selected
		without.orderedDriverInputNS = 0
		required := int64(math.Ceil((row.y - estimatedNS(row, without)) / row.x[12]))
		if required > selected.orderedDriverInputNS {
			selected.orderedDriverInputNS = required
		}
	}
	return selected, nil
}

func fitExactResidualPerRow(rows []observation, c constants, feature int) (int64, bool) {
	values := make([]float64, 0)
	for _, row := range rows {
		if row.censored || len(row.x) <= feature || row.x[feature] <= 0 {
			continue
		}
		without := c
		without.recsetAggregateRowNS = 0
		values = append(values, math.Max(1, (row.y-estimatedNS(row, without))/row.x[feature]))
	}
	if len(values) == 0 {
		return 0, false
	}
	sort.Float64s(values)
	return int64(math.Round(values[len(values)/2])), true
}

func fitNonnegative(x [][]float64, y []float64) ([]float64, error) {
	if len(x) == 0 || len(x) != len(y) {
		return nil, errors.New("invalid equation matrix")
	}
	beta := make([]float64, len(x[0]))
	for i := range beta {
		beta[i] = 1
	}
	for iteration := 0; iteration < 10000; iteration++ {
		largestChange := 0.0
		for column := range beta {
			numerator, denominator := 0.0, 0.0
			for rowIndex, features := range x {
				weight := 1 / math.Max(1, y[rowIndex]*y[rowIndex])
				residual := y[rowIndex]
				for other, value := range beta {
					if other != column {
						residual -= features[other] * value
					}
				}
				numerator += weight * features[column] * residual
				denominator += weight * features[column] * features[column]
			}
			if denominator == 0 {
				return nil, fmt.Errorf("cost equation column %d is not covered", column)
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
	return beta, nil
}

func estimatedNS(row observation, c constants) float64 {
	beta := []float64{
		float64(c.scanInvocationNS),
		float64(c.scanRowNS),
		float64(c.filterColumnRowNS),
		float64(c.mapColumnRowNS),
		float64(c.expressionOperationNS),
		float64(c.recsetStartupNS),
		float64(c.recsetBuildRowNS),
		float64(c.recsetProbeRowNS),
		float64(c.groupCacheStartupNS),
		float64(c.groupCacheBuildRowNS),
		float64(c.groupCacheProbeRowNS),
		float64(c.recsetAggregateRowNS),
		float64(c.orderedDriverInputNS),
	}
	total := 0.0
	for i, value := range row.x {
		total += value * beta[i]
	}
	return total
}

func filterObservations(rows []observation, holdout bool) []observation {
	filtered := make([]observation, 0, len(rows))
	for _, row := range rows {
		if row.holdout == holdout {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func filterExactObservations(rows []observation) []observation {
	filtered := make([]observation, 0, len(rows))
	for _, row := range rows {
		if !row.censored {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func filterCompleteExactPairs(rows []observation) []observation {
	censoredCases := make(map[string]bool)
	for _, row := range rows {
		if row.censored {
			censoredCases[row.caseName] = true
		}
	}
	filtered := make([]observation, 0, len(rows))
	for _, row := range rows {
		if !censoredCases[row.caseName] {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

type modelError struct {
	medianAbsolutePercent float64
	p90AbsolutePercent    float64
	meanFactor            float64
}

func measureModelError(rows []observation, estimate func(observation) float64) modelError {
	percentErrors := make([]float64, 0, len(rows))
	factorTotal := 0.0
	exactCount := 0
	for _, row := range rows {
		if row.censored {
			continue
		}
		predicted := math.Max(1, estimate(row))
		actual := math.Max(1, row.y)
		percentErrors = append(percentErrors, math.Abs(predicted-actual)/actual*100)
		factorTotal += math.Max(predicted/actual, actual/predicted)
		exactCount++
	}
	sort.Float64s(percentErrors)
	if len(percentErrors) == 0 {
		return modelError{}
	}
	p90Index := int(math.Ceil(float64(len(percentErrors))*0.9)) - 1
	return modelError{
		medianAbsolutePercent: percentErrors[len(percentErrors)/2],
		p90AbsolutePercent:    percentErrors[p90Index],
		meanFactor:            factorTotal / float64(exactCount),
	}
}

func decisionAccuracy(rows []observation, estimate func(observation) float64) (int, int) {
	pairs, err := decisionPairs(rows)
	if err != nil {
		return 0, 0
	}
	correct := 0
	for _, pair := range pairs {
		if (pair.candidate.y < pair.driver.y) == (estimate(pair.candidate) < estimate(pair.driver)) {
			correct++
		}
	}
	return correct, len(pairs)
}

func printModelComparison(label string, rows []observation, c constants) {
	models := []struct {
		name     string
		estimate func(observation) float64
	}{
		{name: "current", estimate: func(row observation) float64 { return row.currentEstimate }},
		{name: "updated", estimate: func(row observation) float64 { return estimatedNS(row, c) }},
	}
	for _, model := range models {
		err := measureModelError(rows, model.estimate)
		correct, total := decisionAccuracy(rows, model.estimate)
		fmt.Printf("model %-8s %-7s median-abs-error=%6.1f%% p90-abs-error=%6.1f%% mean-factor=%5.2fx decisions=%d/%d\n",
			label, model.name, err.medianAbsolutePercent, err.p90AbsolutePercent,
			err.meanFactor, correct, total)
	}
	for _, row := range rows {
		updated := estimatedNS(row, c)
		if row.censored {
			fmt.Printf("estimate %-8s %-42s %-30s whole-query>%8.3fms timed-out current=%9.3fms updated=%9.3fms\n",
				label, row.caseName, row.plan, row.y/1e6, row.currentEstimate/1e6, updated/1e6)
			continue
		}
		fmt.Printf("estimate %-8s %-42s %-30s whole-query=%9.3fms mad=%7.3fms current=%9.3fms (%+7.1f%%) updated=%9.3fms (%+7.1f%%)\n",
			label, row.caseName, row.plan, row.y/1e6, row.noiseNS/1e6,
			row.currentEstimate/1e6, (row.currentEstimate-row.y)/row.y*100,
			updated/1e6, (updated-row.y)/row.y*100)
	}
}

func validateMeasurementSignal(rows []observation) error {
	pairs, err := decisionPairs(rows)
	if err != nil {
		return err
	}
	for name, pair := range pairs {
		difference := math.Abs(pair.driver.y - pair.candidate.y)
		noise := 3 * (pair.driver.noiseNS + pair.candidate.noiseNS)
		if difference <= noise {
			return fmt.Errorf("%q has no calibratable A/B signal: difference %.3fms <= 3*(MAD candidate + driver) %.3fms",
				name, difference/1e6, noise/1e6)
		}
	}
	return nil
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

func readCurrentConstants(path string) (constants, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return constants{}, err
	}
	names := []string{
		"planner_membership_scan_invocation_ns",
		"planner_membership_scan_row_ns",
		"planner_membership_filter_column_row_ns",
		"planner_membership_map_column_row_ns",
		"planner_membership_expression_operation_row_ns",
		"planner_membership_recset_startup_ns",
		"planner_membership_recset_build_row_ns",
		"planner_membership_recset_probe_row_ns",
		"planner_membership_recset_aggregate_row_ns",
		"planner_membership_group_cache_startup_ns",
		"planner_membership_group_cache_build_row_ns",
		"planner_membership_group_cache_probe_row_ns",
		"planner_membership_ordered_driver_input_row_ns",
	}
	values := make([]int64, len(names))
	content := string(data)
	for i, name := range names {
		prefix := "(define " + name + " "
		start := strings.Index(content, prefix)
		if start < 0 {
			return constants{}, fmt.Errorf("cost constant %s not found", name)
		}
		start += len(prefix)
		end := strings.IndexByte(content[start:], ')')
		if end < 0 {
			return constants{}, fmt.Errorf("cost constant %s is unterminated", name)
		}
		value, err := strconv.ParseInt(strings.TrimSpace(content[start:start+end]), 10, 64)
		if err != nil {
			return constants{}, fmt.Errorf("cost constant %s: %w", name, err)
		}
		values[i] = value
	}
	return constants{
		scanInvocationNS: values[0], scanRowNS: values[1],
		filterColumnRowNS: values[2], mapColumnRowNS: values[3],
		expressionOperationNS: values[4], recsetStartupNS: values[5],
		recsetBuildRowNS: values[6], recsetProbeRowNS: values[7],
		recsetAggregateRowNS: values[8], groupCacheStartupNS: values[9],
		groupCacheBuildRowNS: values[10], groupCacheProbeRowNS: values[11],
		orderedDriverInputNS: values[12],
	}, nil
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

(define planner_membership_scan_invocation_ns %d)
(define planner_membership_scan_row_ns %d)
(define planner_membership_filter_column_row_ns %d)
(define planner_membership_map_column_row_ns %d)
(define planner_membership_expression_operation_row_ns %d)
(define planner_membership_recset_startup_ns %d)
(define planner_membership_recset_build_row_ns %d)
(define planner_membership_recset_probe_row_ns %d)
(define planner_membership_recset_aggregate_row_ns %d)
(define planner_membership_group_cache_startup_ns %d)
(define planner_membership_group_cache_build_row_ns %d)
(define planner_membership_group_cache_probe_row_ns %d)
(define planner_membership_ordered_driver_input_row_ns %d)
/* END GENERATED COST CONSTANTS */`, c.scanInvocationNS, c.scanRowNS,
		c.filterColumnRowNS, c.mapColumnRowNS, c.expressionOperationNS,
		c.recsetStartupNS, c.recsetBuildRowNS, c.recsetProbeRowNS,
		c.recsetAggregateRowNS, c.groupCacheStartupNS, c.groupCacheBuildRowNS,
		c.groupCacheProbeRowNS, c.orderedDriverInputNS)
	return os.WriteFile(path, []byte(content[:begin]+block+content[end:]), 0o644)
}
