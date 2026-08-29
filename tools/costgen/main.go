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
	// This mirrors planner_adaptive_observation_budget_ns: below this complete
	// query cost, picking either measured alternative satisfies the planner's
	// risk contract. Costgen still reports duration error, but does not reject a
	// model merely for interchanging two sub-budget variants.
	calibrationDecisionRiskBudgetNS = float64(100 * time.Millisecond)
	// BenchmarkRecSetBoundaryCrossover measures the storage operator's runtime
	// kernel switch. Costgen uses the same dimensionless inequality to describe
	// the work actually emitted; fitted planner coefficients still price that
	// work from end-to-end calibration observations.
	adaptiveOrderedRecsetSortUnitNS = 4.0
	adaptiveOrderedMembershipRowNS  = 6.0
)

var errInsufficientCalibrationObservations = errors.New("insufficient calibration observations")

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
	CalibrationComponent    string  `yaml:"calibration_component"`
	Race                    bool    `yaml:"race"`
	RaceGrace               float64 `yaml:"race_grace"`
	CacheState              string  `yaml:"cache_state"`
	CompileState            string  `yaml:"compile_state"`
	ExpectedDriverInputRows float64 `yaml:"expected_driver_input_rows"`
	ExpectedBroadTextBytes  float64 `yaml:"expected_broad_text_bytes"`
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
	DecisionSite                     string   `json:"decision_site"`
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
	CarrierRows                      *float64 `json:"carrier_rows"`
	CandidateDensity                 *float64 `json:"candidate_density"`
	ProjectedDriverRows              *float64 `json:"projected_driver_rows"`
	DriverInputRows                  *float64 `json:"driver_input_rows"`
	DriverOrderPartitioned           bool     `json:"driver_order_partitioned"`
	DriverRows                       *float64 `json:"driver_rows"`
	PrefilteredDriverRows            *float64 `json:"prefiltered_driver_rows"`
	ExpectedDriverRowsVisited        *float64 `json:"expected_driver_rows_visited"`
	Limit                            *float64 `json:"limit"`
	Offset                           *float64 `json:"offset"`
	ProbeBranches                    *float64 `json:"probe_branches"`
	DownstreamProbeBranches          *float64 `json:"downstream_probe_branches"`
	CandidateScanInvocations         *float64 `json:"candidate_scan_invocations"`
	CandidateFilterColumns           *float64 `json:"candidate_filter_columns"`
	CandidateMapColumns              *float64 `json:"candidate_map_columns"`
	CandidateCacheMapColumns         *float64 `json:"candidate_cache_map_columns"`
	CandidateCacheBacked             bool     `json:"candidate_cache_backed"`
	CandidateExpressionOperations    *float64 `json:"candidate_expression_operations"`
	CandidateExpressionDepth         *float64 `json:"candidate_expression_depth"`
	CandidateIndexFilterRows         *float64 `json:"candidate_index_filter_rows"`
	CandidateBroadTextMatchRows      *float64 `json:"candidate_broad_text_match_rows"`
	CandidateBroadTextMatchBytes     *float64 `json:"candidate_broad_text_match_bytes"`
	CandidateFilterValueRows         *float64 `json:"candidate_filter_value_rows"`
	CandidateExpressionOperationRows *float64 `json:"candidate_expression_operation_rows"`
	DriverScanInvocations            *float64 `json:"driver_scan_invocations"`
	DriverFilterColumns              *float64 `json:"driver_filter_columns"`
	DriverMapColumns                 *float64 `json:"driver_map_columns"`
	DriverExpressionOperations       *float64 `json:"driver_expression_operations"`
	DriverExpressionDepth            *float64 `json:"driver_expression_depth"`
	JoinInputRows                    *float64 `json:"join_input_rows"`
	JoinEstimatedRows                *float64 `json:"join_estimated_rows"`
	JoinOutputRows                   *float64 `json:"join_output_rows"`
	JoinTableCount                   *float64 `json:"join_table_count"`
	JoinLegacyProbeRows              *float64 `json:"join_legacy_probe_rows"`
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
	cacheState      string
	component       string
	decision        string
	plan            string
	y               float64
	currentEstimate float64
	holdout         bool
	noiseNS         float64
	censored        bool
	x               []float64
}

type constants struct {
	scanInvocationNS           int64
	scanRowNS                  int64
	filterColumnRowNS          int64
	mapColumnRowNS             int64
	expressionOperationNS      int64
	broadTextMatchRowNS        int64
	broadTextMatchByteNS       int64
	recsetStartupNS            int64
	recsetBuildRowNS           int64
	recsetProbeRowNS           int64
	recsetAggregateRowNS       int64
	groupCacheStartupNS        int64
	groupCacheBuildRowNS       int64
	groupCacheProbeRowNS       int64
	orderedDriverInputNS       int64
	orderedScanInvocationNS    int64
	orderedRecsetSortUnitNS    int64
	downstreamProbeRowNS       int64
	scalarPresenceProbeRowNS   int64
	membershipDirectProbeRowNS int64
}

func main() {
	patch := flag.Bool("patch", false, "rewrite lib/queryplan-physical-expr.scm")
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
	queryplanPath := filepath.Join(root, "lib", "queryplan-physical-expr.scm")
	currentConstants, err := readCurrentConstants(queryplanPath)
	if err != nil {
		fatal(err)
	}
	// A generator schema change may rename or add constants which the planner
	// already references. Bootstrap the generated block with the previous
	// numeric values before starting the calibration server; the final patch
	// below replaces them with the newly fitted values. This keeps generated
	// constants tool-owned without making migrations impossible to execute.
	if *patch {
		if err := patchQueryplan(queryplanPath, currentConstants); err != nil {
			fatal(fmt.Errorf("bootstrap generated constants: %w", err))
		}
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
	// Coefficient fitting remains scoped to the membership-carrier family. Other
	// calibrated decision families are executed and result-checked above, but
	// must not be mistaken for additional equations of this coefficient model.
	membershipObservations := filterDecisionObservations(observations, "membership_carrier")
	training := filterObservations(membershipObservations, false)
	allTraining := filterObservations(membershipObservations, false)
	carrierTraining := filterCarrierObservations(training)
	fitTraining := filterCompleteExactPairs(carrierTraining)
	if err := validateMeasurementSignal(fitTraining); err != nil {
		fatal(err)
	}
	c, err := solve(carrierTraining, allTraining, currentConstants)
	if err != nil {
		fatal(err)
	}
	logStep("selected downstream probe coefficient=%d ns/probe", c.downstreamProbeRowNS)
	if err := validateDecisionOrdering(training, c); err != nil {
		fatal(err)
	}
	fmt.Printf("scan invocation:      %d ns/invocation\nscan row:             %d ns/input-row\nfilter column:        %d ns/value\nmap column:           %d ns/value\nexpression operation: %d ns/row-operation\nbroad text match:     %d ns/input-row + %d ns/input-byte\nrecset startup:       %d ns\nrecset build:         %d ns/matching-row\nrecset probe:         %d ns/driver-row\nrecset aggregate:     %d ns/driver-input-row\ngroup-cache startup:  %d ns\ngroup-cache build:    %d ns/matching-row\ngroup-cache probe:    %d ns/driver-row\nordered driver input: %d ns/(rows²/1M)\nordered scan startup: %d ns/invocation\nordered RecSet sort:  %d ns/nlog2n-unit\ndownstream probe:     %d ns/probe\nscalar presence probe:%d ns/probe\nmembership probe:     %d ns/probe\n",
		c.scanInvocationNS, c.scanRowNS, c.filterColumnRowNS, c.mapColumnRowNS,
		c.expressionOperationNS, c.broadTextMatchRowNS, c.broadTextMatchByteNS, c.recsetStartupNS, c.recsetBuildRowNS, c.recsetProbeRowNS,
		c.recsetAggregateRowNS, c.groupCacheStartupNS, c.groupCacheBuildRowNS,
		c.groupCacheProbeRowNS, c.orderedDriverInputNS, c.orderedScanInvocationNS, c.orderedRecsetSortUnitNS, c.downstreamProbeRowNS, c.scalarPresenceProbeRowNS,
		c.membershipDirectProbeRowNS)
	printModelComparison("training", training, c)
	holdout := filterObservations(membershipObservations, true)
	if len(holdout) > 0 {
		printModelComparison("holdout", holdout, c)
		if err := validateDecisionOrdering(holdout, c); err != nil {
			fatal(fmt.Errorf("holdout: %w", err))
		}
		if err := validateModelImprovement(holdout, c); err != nil {
			fatal(fmt.Errorf("holdout: %w", err))
		}
	}
	printDecisionOrdering(membershipObservations, c)
	orderedJoinObservations := filterDecisionObservations(observations, "scan_join_order")
	if len(orderedJoinObservations) > 0 {
		// scan_join_order deliberately reuses the calibrated scan/map/expression
		// work units instead of introducing an independently fitted coefficient
		// set. Its forced variants still form a mandatory ordering check: this
		// catches a lowerer formula which compiles but chooses the wrong operator.
		if err := validateDecisionOrdering(orderedJoinObservations, c); err != nil {
			fatal(fmt.Errorf("scan_join_order: %w", err))
		}
		printModelComparison("scan_join_order", orderedJoinObservations, c)
		printDecisionOrdering(orderedJoinObservations, c)
	}
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
	currentCorrect, total := decisionAccuracy(rows, func(row observation) float64 { return row.currentEstimate })
	updatedCorrect, _ := decisionAccuracy(rows, func(row observation) float64 { return estimatedNS(row, c) })
	if updatedCorrect < currentCorrect {
		return fmt.Errorf("updated model regresses holdout decisions: %d/%d -> %d/%d",
			currentCorrect, total, updatedCorrect, total)
	}
	/* Exact duration is a tie-breaker for an incomplete decision model. Once all
	holdout inequalities are correct, rejecting a newly identifiable coefficient
	because unrelated absolute residuals move by a few percent preserves stale
	constants indefinitely and contradicts solve's decision-first selection. */
	if updatedCorrect == total {
		return nil
	}
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
		/* The SQL runner accepts a broader YAML surface than this calibration
		tool needs. Avoid strictly decoding unrelated suites: only files which
		explicitly opt into physical calibration can contribute observations. */
		if !bytes.Contains(data, []byte("physical_calibration: true")) {
			return nil
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
		var deferredAdaptiveCases []testCase
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
			// This protected regression predates the adaptive RecSet scan kernel. Its
			// two former alternatives are now one storage operator selected from exact
			// runtime cardinalities, so there is no planner variant to force or fit.
			// Keep executing the query and require stable output; the storage benchmark
			// owns the equivalent-kernel crossover calibration.
			if test.CalibrationDecision == "ordered_recset_consumer" {
				// Execute after all fitted cases so this broad ordered query cannot warm
				// their tables, indexes or group caches and contaminate race ordering.
				deferredAdaptiveCases = append(deferredAdaptiveCases, test)
				logStep("deferred adaptive storage case %s", test.Name)
				continue
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
				baseQuery := strings.TrimSpace(strings.TrimPrefix(query, "EXPLAIN PHYSICAL CALIBRATE"))
				separateRuns, separateRows, separateErr := runCalibrationVariantsSeparately(
					server, baseQuery, test, warmup, repetitions)
				if separateErr != nil {
					return nil, nil, fmt.Errorf("%s/%s: %w", currentSuite.Path, test.Name, separateErr)
				}
				runs = separateRuns
				for rowIndex := range separateRows {
					separateRows[rowIndex].CaseName = test.Name
					separateRows[rowIndex].CacheState = cacheState
					separateRows[rowIndex].CompileState = compileState
				}
				allRows = append(allRows, separateRows...)
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
			if test.ExpectedBroadTextBytes > 0 {
				for _, row := range medians {
					if row.CandidateBroadTextMatchBytes == nil || *row.CandidateBroadTextMatchBytes < test.ExpectedBroadTextBytes {
						actual := "nil"
						if row.CandidateBroadTextMatchBytes != nil {
							actual = fmt.Sprintf("%.0f", *row.CandidateBroadTextMatchBytes)
						}
						return nil, nil, fmt.Errorf("%s/%s: candidate_broad_text_match_bytes=%s, want at least %.0f",
							currentSuite.Path, test.Name, actual, test.ExpectedBroadTextBytes)
					}
				}
			}
			for _, row := range medians {
				features, err := rowFeatures(row)
				if err != nil {
					return nil, nil, fmt.Errorf("%s/%s: %w", currentSuite.Path, test.Name, err)
				}
				observations = append(observations, observation{
					caseName: test.Name, decision: row.Decision, plan: row.Plan, y: row.WholeQueryExecutionNS,
					cacheState: cacheState, component: test.CalibrationComponent,
					currentEstimate: *row.EstimatedNS, holdout: test.CalibrationHoldout,
					noiseNS: medianAbsoluteDeviation(runs, row.Plan), censored: row.TimedOut, x: features,
				})
			}
			logStep("measured %s", test.Name)
		}
		for _, test := range deferredAdaptiveCases {
			if err := validateAdaptiveStorageCase(server, currentSuite.Path, test); err != nil {
				return nil, nil, err
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

func validateAdaptiveStorageCase(server *memcpServer, suitePath string, test testCase) error {
	query := strings.TrimSpace(test.SQL)
	warmup, repetitions := test.Warmup, test.Repetitions
	if repetitions <= 0 {
		repetitions = 5
	}
	var expected []byte
	for run := 0; run < warmup+repetitions; run++ {
		output, err := server.execute("/sql/"+database, query, 10*time.Minute)
		if err != nil {
			return fmt.Errorf("%s/%s adaptive execution: %w", suitePath, test.Name, err)
		}
		if run < warmup {
			continue
		}
		if expected == nil {
			expected = append([]byte(nil), output...)
		} else if !bytes.Equal(expected, output) {
			return fmt.Errorf("%s/%s adaptive executions returned different results",
				suitePath, test.Name)
		}
	}
	logStep("validated adaptive storage case %s", test.Name)
	return nil
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

func discoverCalibrationDecision(server *memcpServer, query, decisionName string) (*calibrationDiscovery, map[string]*float64, error) {
	discoveryPayload, err := server.execute("/sql/"+database,
		"EXPLAIN PHYSICAL CALIBRATE DISCOVER\n"+query, 10*time.Minute)
	if err != nil {
		return nil, nil, err
	}
	discovered, err := decodeDiscoveries(discoveryPayload)
	if err != nil {
		return nil, nil, fmt.Errorf("decode calibration discovery: %w; response=%s", err, discoveryPayload)
	}
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
				return nil, nil, fmt.Errorf("calibration requires one %s decision, found several", decisionName)
			}
			decision = &discovered[i]
		}
	}
	if decision == nil || len(decision.Alternatives) < 2 || len(decision.Alternatives) > 4 ||
		len(decision.EstimatedNS) != len(decision.Alternatives) {
		return nil, nil, fmt.Errorf("discovery did not expose two to four costed alternatives: %+v", decision)
	}
	estimates := map[string]*float64{}
	for i, plan := range decision.Alternatives {
		estimates[plan] = decision.EstimatedNS[i]
	}
	return decision, estimates, nil
}

func executeCalibrationVariant(server *memcpServer, query, decisionID, plan string, timeout time.Duration) (calibrationRow, error) {
	statement := "EXPLAIN PHYSICAL CALIBRATE VARIANT '" + sqlQuote(decisionID) + "' '" + sqlQuote(plan) + "'\n" + query
	payload, err := server.execute("/sql/"+database, statement, timeout)
	if err != nil {
		return calibrationRow{}, err
	}
	rows, err := decodeRows(payload)
	if err != nil || len(rows) != 1 {
		return calibrationRow{}, fmt.Errorf("decode %s variant: %v; response=%s", plan, err, payload)
	}
	if err := validateRaceWinner(rows[0], decisionID, plan); err != nil {
		return calibrationRow{}, err
	}
	return rows[0], nil
}

func runCalibrationVariantsSeparately(server *memcpServer, query string, test testCase, warmup, repetitions int) ([][]calibrationRow, []calibrationRow, error) {
	decision, _, err := discoverCalibrationDecision(server, query, test.CalibrationDecision)
	if err != nil {
		return nil, nil, err
	}
	var runs [][]calibrationRow
	var raw []calibrationRow
	for run := 0; run < warmup+repetitions; run++ {
		plans := append([]string(nil), decision.Alternatives...)
		if run%2 == 1 {
			plans[0], plans[1] = plans[1], plans[0]
		}
		rows := make([]calibrationRow, 0, len(plans))
		for _, plan := range plans {
			row, err := executeCalibrationVariant(server, query, decision.DecisionID, plan, 10*time.Minute)
			if err != nil {
				return nil, nil, err
			}
			rows = append(rows, row)
		}
		equal := true
		for i := 1; i < len(rows); i++ {
			equal = equal && rows[0].Rows == rows[i].Rows && rows[0].ResultHash == rows[i].ResultHash
		}
		for i := range rows {
			rows[i].ResultEqual = equal
		}
		if !equal {
			results := make([]string, 0, len(rows))
			for _, row := range rows {
				results = append(results, fmt.Sprintf("%s(rows=%d,hash=%s)",
					row.Plan, row.Rows, row.ResultHash))
			}
			return nil, nil, fmt.Errorf("separately executed variants returned different results: %s",
				strings.Join(results, ", "))
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].Plan < rows[j].Plan })
		if run >= warmup {
			runs = append(runs, rows)
			raw = append(raw, rows...)
		}
	}
	return runs, raw, nil
}

func runCalibrationRaces(server *memcpServer, query string, test testCase, repetitions int) ([][]calibrationRow, []calibrationRow, error) {
	decision, estimates, err := discoverCalibrationDecision(server, query, test.CalibrationDecision)
	if err != nil {
		return nil, nil, err
	}
	if len(decision.Alternatives) != 2 {
		return runCalibrationVariantsSeparately(server, query, test, 0, repetitions)
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
	cleanFirst, err := executeCalibrationVariant(server, query, decisionID, first.plan, 10*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("standalone winner %s: %w", first.plan, err)
	}
	first.row = cleanFirst
	if !second.row.TimedOut {
		cleanSecond, err := executeCalibrationVariant(server, query, decisionID, second.plan, 10*time.Minute)
		if err != nil {
			return nil, fmt.Errorf("standalone bounded alternative %s: %w", second.plan, err)
		}
		second.row = cleanSecond
		equal := first.row.Rows == second.row.Rows && first.row.ResultHash == second.row.ResultHash
		first.row.ResultEqual, second.row.ResultEqual = equal, equal
		if !equal {
			return nil, fmt.Errorf("standalone variants returned different results")
		}
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
	if row.EstimatedNS == nil || row.WholeQueryExecutionNS <= 0 {
		return fmt.Errorf("forced race variant has incomplete measurements: %+v", row)
	}
	if row.Decision == "scan_join_order" {
		if row.JoinInputRows == nil || row.JoinEstimatedRows == nil ||
			row.JoinOutputRows == nil || row.JoinTableCount == nil || row.JoinLegacyProbeRows == nil {
			return fmt.Errorf("ordered join variant has incomplete measurements: %+v", row)
		}
	} else if row.CandidateInputRows == nil || row.CandidateRows == nil ||
		row.DriverInputRows == nil || row.DriverRows == nil || row.ExpectedDriverRowsVisited == nil {
		return fmt.Errorf("membership variant has incomplete measurements: %+v", row)
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
	if len(rows) < 2 || len(rows) > 4 {
		return fmt.Errorf("expected two to four membership alternatives, got %d", len(rows))
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
			row.CandidateIndexFilterRows,
			row.CandidateBroadTextMatchRows, row.CandidateBroadTextMatchBytes,
			row.DriverScanInvocations, row.DriverFilterColumns,
			row.DriverMapColumns, row.DriverExpressionOperations,
			row.DriverExpressionDepth,
		}
		for _, input := range workInputs {
			if input == nil {
				return fmt.Errorf("physical work profile contains nil inputs: %+v", row)
			}
		}
		if *row.CandidateBroadTextMatchRows > 0 && *row.CandidateBroadTextMatchBytes <= 0 {
			return fmt.Errorf("broad-text decision has no value-byte statistics: %+v", row)
		}
		if *row.CandidateIndexFilterRows < 0 || *row.CandidateIndexFilterRows > *row.CandidateInputRows {
			return fmt.Errorf("candidate index filter rows exceed their input: %+v", row)
		}
		if row.WholeQueryExecutionNS <= 0 {
			return fmt.Errorf("invalid whole_query_execution_ns for %s", row.Plan)
		}
		seen[row.Plan] = true
		if row.Plan == "prefiltered_candidate_keyset" && row.PrefilteredDriverRows == nil {
			return fmt.Errorf("prefiltered alternative has no driver cardinality: %+v", row)
		}
	}
	if !seen["candidate_keyset"] ||
		(!seen["driver_order_membership_probe"] && !seen["driver_filter_join_probe"]) {
		return fmt.Errorf("alternatives incomplete: %v", seen)
	}
	if len(rows) == 3 && (!seen["ordered_batch_accept"] || !seen["driver_order_membership_probe"]) {
		return fmt.Errorf("three-way decision is missing ordered_batch_accept: %v", seen)
	}
	if len(rows) == 4 && (!seen["ordered_batch_accept"] ||
		!seen["driver_order_membership_probe"] || !seen["prefiltered_candidate_keyset"]) {
		return fmt.Errorf("four-way decision is incomplete: %v", seen)
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
	plans := make([]string, 0, len(byPlan))
	for plan := range byPlan {
		plans = append(plans, plan)
	}
	sort.Strings(plans)
	var result []calibrationRow
	for _, plan := range plans {
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
	if row.Decision == "scan_join_order" {
		if row.JoinInputRows == nil || row.JoinEstimatedRows == nil ||
			row.JoinOutputRows == nil || row.JoinTableCount == nil {
			return nil, fmt.Errorf("ordered join work profile contains nil inputs: %+v", row)
		}
		features := make([]float64, 19)
		features[0] = math.Max(0, *row.JoinTableCount-1)
		features[15] = 1
		features[1] = *row.JoinInputRows
		features[3] = *row.JoinOutputRows + *row.JoinInputRows*math.Max(0, *row.JoinTableCount-1)
		features[4] = *row.JoinEstimatedRows
		if row.Plan == "legacy_join_tree" {
			if row.JoinLegacyProbeRows == nil {
				return nil, fmt.Errorf("ordered legacy join profile has nil probe rows: %+v", row)
			}
			features[18] = *row.JoinLegacyProbeRows
		}
		return features, nil
	}
	scanInvocations := *row.CandidateScanInvocations + *row.DriverScanInvocations
	driverWorkRows := *row.DriverInputRows
	adaptiveProbeRows, adaptiveSortWork := 0.0, 0.0
	if row.Plan == "candidate_keyset" && row.Consumer == "order_limit" {
		sortWork := orderedRecsetSortWork(*row.ProjectedDriverRows)
		if sortWork*adaptiveOrderedRecsetSortUnitNS <
			*row.ExpectedDriverRowsVisited*adaptiveOrderedMembershipRowNS {
			driverWorkRows = *row.ProjectedDriverRows
			adaptiveSortWork = sortWork
		} else {
			driverWorkRows = *row.ExpectedDriverRowsVisited
			adaptiveProbeRows = driverWorkRows
		}
	}
	if row.Plan == "driver_order_membership_probe" || row.Plan == "scan_order" {
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
	if row.Plan == "driver_order_membership_probe" || row.Plan == "scan_order" {
		if row.CandidateCacheBacked {
			mapColumns = *row.CandidateCacheMapColumns
		}
	}
	driverMapRows := *row.ProjectedDriverRows
	if row.Plan == "driver_order_membership_probe" || row.Plan == "scan_order" {
		driverMapRows = *row.DriverRows
	}
	mapValues := *row.CandidateRows*mapColumns + driverMapRows**row.DriverMapColumns
	aggregateDriverRows := 0.0
	if row.Consumer == "aggregate" {
		aggregateDriverRows = *row.DriverInputRows
	}
	orderedDriverInputRows := 0.0
	orderedScanInvocations := 0.0
	downstreamProbeBranches := 0.0
	if row.DownstreamProbeBranches != nil {
		downstreamProbeBranches = *row.DownstreamProbeBranches
	}
	candidateDensity := 0.0
	candidateProbeBranches := 1.0
	if row.ProbeBranches != nil {
		candidateProbeBranches = math.Max(1, *row.ProbeBranches)
	}
	if row.CandidateDensity != nil {
		candidateDensity = *row.CandidateDensity
	} else if *row.CandidateInputRows > 0 {
		candidateDensity = math.Min(1,
			*row.CandidateRows/(*row.CandidateInputRows/candidateProbeBranches))
	}
	if row.Limit != nil {
		orderedDriverInputRows = *row.DriverInputRows * *row.DriverInputRows / 1_000_000
		orderedScanInvocations = *row.DriverScanInvocations
	}
	switch row.Plan {
	case "candidate_keyset":
		cacheStartup, cacheBuildRows := 0.0, 0.0
		if row.CandidateCacheBacked {
			cacheStartup, cacheBuildRows = 1, *row.CandidateRows
		}
		return []float64{
			scanInvocations, scanRows, filterValues, mapValues, expressionOperations,
			1, *row.CandidateRows + *row.ProjectedDriverRows, adaptiveProbeRows,
			cacheStartup, cacheBuildRows, 0,
			aggregateDriverRows, 0, *row.CandidateBroadTextMatchRows,
			*row.CandidateBroadTextMatchBytes, orderedScanInvocations, 0,
			adaptiveSortWork,
			*row.ProjectedDriverRows * downstreamProbeBranches,
		}, nil
	case "driver_order_membership_probe", "scan_order":
		recsetStartup, recsetBuildRows, recsetProbeRows := 1.0, *row.CandidateRows, *row.ExpectedDriverRowsVisited
		cacheStartup, cacheBuildRows, cacheProbeRows := 0.0, 0.0, 0.0
		if row.CandidateCacheBacked {
			recsetStartup, recsetBuildRows, recsetProbeRows = 0, 0, 0
			cacheStartup, cacheBuildRows, cacheProbeRows = 1, *row.CandidateRows, *row.ExpectedDriverRowsVisited
		}
		return []float64{
			scanInvocations, scanRows, filterValues, mapValues, expressionOperations,
			recsetStartup, recsetBuildRows, recsetProbeRows,
			cacheStartup, cacheBuildRows, cacheProbeRows,
			0, orderedDriverInputRows, 0,
			0, orderedScanInvocations, 0, 0,
			*row.ExpectedDriverRowsVisited * downstreamProbeBranches,
		}, nil
	case "ordered_batch_accept":
		fraction := 1.0
		if *row.DriverInputRows > 0 {
			fraction = math.Min(1, *row.ExpectedDriverRowsVisited / *row.DriverInputRows)
		}
		firstBatch := *row.Limit + *row.Offset
		batches, remaining, size := 1.0, *row.ExpectedDriverRowsVisited-firstBatch, firstBatch*2
		for remaining > 0 && size > 0 {
			batches++
			remaining -= size
			size *= 2
		}
		repeatFraction := math.Min(batches, fraction*batches)
		candidateWorkRows := *row.CandidateInputRows * repeatFraction
		candidateMatchRows := *row.CandidateRows * repeatFraction
		projectionRows := *row.ProbeBranches *
			(2**row.ExpectedDriverRowsVisited + candidateWorkRows + candidateMatchRows)
		orderedDriverWork := batches * *row.DriverInputRows * *row.DriverInputRows / 1_000_000
		if row.DriverOrderPartitioned {
			// Range-partitioned ORDER BY windows visit a geometric series of
			// disjoint shard prefixes, bounded by twice the final prefix.
			orderedDriverWork = 2 * *row.ExpectedDriverRowsVisited
		}
		return []float64{
			*row.DriverScanInvocations + batches**row.CandidateScanInvocations,
			*row.ExpectedDriverRowsVisited + candidateWorkRows + projectionRows,
			candidateWorkRows * *row.CandidateFilterColumns,
			0,
			candidateWorkRows * *row.CandidateExpressionOperations,
			0,
			projectionRows,
			0, 0, 0, 0, 0, orderedDriverWork,
			*row.CandidateBroadTextMatchRows * repeatFraction,
			*row.CandidateBroadTextMatchBytes * repeatFraction,
			batches * *row.DriverScanInvocations, 0, 0,
			*row.ExpectedDriverRowsVisited * candidateDensity * downstreamProbeBranches,
		}, nil
	case "driver_filter_join_probe":
		return []float64{
			scanInvocations, scanRows, filterValues, mapValues, expressionOperations,
			0, 0, 0, 0, 0, 0,
			aggregateDriverRows, 0, *row.CandidateBroadTextMatchRows,
			*row.CandidateBroadTextMatchBytes, 0,
			*row.DriverRows * *row.ProbeBranches, 0,
			*row.DriverRows * downstreamProbeBranches,
		}, nil
	case "prefiltered_candidate_keyset":
		branches := math.Max(1, *row.ProbeBranches)
		candidateDomainRows := *row.CandidateInputRows / branches
		candidateWorkRows := math.Min(*row.PrefilteredDriverRows, candidateDomainRows) * branches
		candidateMatchRows := candidateWorkRows * *row.CandidateDensity
		candidateFraction := 0.0
		if *row.CandidateInputRows > 0 {
			candidateFraction = math.Min(1, candidateWorkRows / *row.CandidateInputRows)
		}
		projectionRows := 2**row.PrefilteredDriverRows + candidateWorkRows + candidateMatchRows
		return []float64{
			*row.DriverScanInvocations + *row.CandidateScanInvocations,
			*row.DriverInputRows + candidateWorkRows + projectionRows,
			*row.DriverInputRows**row.DriverFilterColumns + candidateWorkRows**row.CandidateFilterColumns,
			projectionRows,
			*row.DriverInputRows**row.DriverExpressionOperations + candidateWorkRows**row.CandidateExpressionOperations,
			1, projectionRows, 0, 0, 0, 0, 0, 0,
			*row.CandidateBroadTextMatchRows * candidateFraction,
			*row.CandidateBroadTextMatchBytes * candidateFraction,
			orderedScanInvocations, 0, 0,
			*row.PrefilteredDriverRows * downstreamProbeBranches,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported plan %q", row.Plan)
	}
}

func orderedRecsetSortWork(rows float64) float64 {
	if rows <= 1 {
		return rows
	}
	return rows * math.Ceil(math.Log2(rows))
}

// solveEquationSystem fits the physical work both alternatives actually perform:
//
//	common = scan_invocations*startup + scan_rows*scan
//	       + filter_values*filter_column + map_values*map_column
//	       + expression_operation_rows*expression_operation
//	candidate = recset_startup + common
//	          + candidate_rows*recset_build + driver_rows*recset_probe
//
// Non-negative coordinate descent prevents noisy samples from generating
// impossible negative costs. Weighting every equation by 1/actual^2 minimizes
// relative rather than absolute error: otherwise multi-second driver probes
// drown out the millisecond candidate plans whose inequality we also need to
// predict correctly.
func solveEquationSystem(rows []observation) (constants, error) {
	candidateX, candidateY := make([][]float64, 0), make([]float64, 0)
	for _, row := range rows {
		if row.censored || row.plan != "candidate_keyset" || row.x[8] > 0 {
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
	if len(candidateX) < 6 {
		return constants{}, fmt.Errorf("%w: common scan fit needs six candidate observations, got %d",
			errInsufficientCalibrationObservations, len(candidateX))
	}
	common, err := fitNonnegative(candidateX, candidateY)
	if err != nil {
		return constants{}, fmt.Errorf("common candidate work: %w", err)
	}
	beta := []float64{
		common[0], common[1], common[2], common[3], common[4],
		1, common[5], 1,
	}
	return constants{
		scanInvocationNS:      int64(math.Round(beta[0])),
		scanRowNS:             int64(math.Round(beta[1])),
		filterColumnRowNS:     int64(math.Round(beta[2])),
		mapColumnRowNS:        int64(math.Round(beta[3])),
		expressionOperationNS: int64(math.Round(beta[4])),
		broadTextMatchRowNS:   1,
		broadTextMatchByteNS:  1,
		recsetStartupNS:       int64(math.Round(beta[5])),
		recsetBuildRowNS:      int64(math.Round(beta[6])),
		recsetProbeRowNS:      int64(math.Round(beta[7])),
		// Cache startup is identified below from cache-backed/direct pairs.
		// Build and probe stay at their floor until fixtures vary them independently.
		groupCacheStartupNS:        1,
		groupCacheBuildRowNS:       1,
		groupCacheProbeRowNS:       1,
		recsetAggregateRowNS:       1,
		orderedDriverInputNS:       1,
		orderedScanInvocationNS:    1,
		orderedRecsetSortUnitNS:    1,
		downstreamProbeRowNS:       1,
		membershipDirectProbeRowNS: 1,
	}, nil
}

// solve keeps an already calibrated base model unless a fresh unconstrained
// equation fit improves both its numerical error and its plan inequalities.
// The two large-shape additions are fitted separately because a cancelled race
// yields an inequality, not an exact duration: y_slow is only known to exceed
// lower_bound_ns.
func solve(exactRows, allRows []observation, baseline constants) (constants, error) {
	fitted, err := solveEquationSystem(exactRows)
	if err != nil && !errors.Is(err, errInsufficientCalibrationObservations) {
		return constants{}, err
	}
	if errors.Is(err, errInsufficientCalibrationObservations) {
		logStep("model selection preserves baseline: %v", err)
	}
	selected := baseline
	if err == nil {
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
		/* The model exists to rank physical alternatives. Prefer a fit which gets
		more measured decisions right even when a few absolute-duration residuals
		grow; use duration accuracy only to break equal-ranking fits. Requiring all
		metrics to improve lets stale constants survive indefinitely. */
		if fitCorrect > baseCorrect || (fitCorrect == baseCorrect &&
			fitError.medianAbsolutePercent <= baseError.medianAbsolutePercent &&
			fitError.meanFactor <= baseError.meanFactor) {
			selected = fitted
		}
		logStep("model selection baseline decisions=%d fitted decisions=%d selected_fitted=%t",
			baseCorrect, fitCorrect, selected == fitted)
	}

	/* Residual fits isolate individual coefficients after the joint model has
	been selected. They are still planner changes, not harmless duration
	calibration: accepting one merely because it reduces residual error can undo
	the selected model's inequalities. Apply each refinement atomically and only
	when it fixes an additional measured decision across the complete pool. */
	if value, ok := fitExactResidualPerRow(allRows, selected, 11); ok {
		trial := selected
		trial.recsetAggregateRowNS = value
		selected = acceptDecisionImprovingRefinement(allRows, selected, trial)
	}
	if value, ok := fitBroadTextResidualPerByte(allRows, selected); ok {
		trial := selected
		trial.broadTextMatchByteNS = value
		selected = acceptDecisionImprovingRefinement(allRows, selected, trial)
	}
	if value, ok := fitOrderedScanInvocation(allRows, selected); ok {
		trial := selected
		trial.orderedScanInvocationNS = value
		selected = acceptDecisionImprovingRefinement(allRows, selected, trial)
	}
	if value, ok := fitOrderedRecsetSortUnit(allRows, selected); ok {
		trial := selected
		trial.orderedRecsetSortUnitNS = value
		selected = acceptDecisionImprovingRefinement(allRows, selected, trial)
	}
	if value, ok := fitDownstreamProbeRow(allRows, selected); ok {
		trial := selected
		trial.downstreamProbeRowNS = value
		selected = acceptDecisionImprovingRefinement(allRows, selected, trial)
	}
	if startup, probe, ok := fitDirectCarrierPair(allRows, selected); ok {
		trial := selected
		trial.groupCacheStartupNS = startup
		trial.membershipDirectProbeRowNS = probe
		selected = acceptDecisionImprovingRefinement(allRows, selected, trial)
	}
	/* A race timeout mixes cold startup and incomplete operator work. It is a
	lower bound for that whole alternative, not evidence for a linear ordered
	driver coefficient. Only exact paired workloads may change that term. */
	// A cancelled broad-text candidate supplies the symmetric inequality:
	// its materialization cost is at least the observed lower bound. Preserve
	// that evidence instead of treating a timeout as an exact duration.
	for _, row := range allRows {
		if !row.censored || row.component != "broad_text_bytes" ||
			row.plan != "candidate_keyset" || len(row.x) <= 14 || row.x[14] <= 0 {
			continue
		}
		without := selected
		without.broadTextMatchByteNS = 0
		required := int64(math.Ceil((row.y - estimatedNS(row, without)) / row.x[14]))
		if required > selected.broadTextMatchByteNS {
			trial := selected
			trial.broadTextMatchByteNS = required
			selected = acceptDecisionImprovingRefinement(allRows, selected, trial)
		}
	}
	return selected, nil
}

func acceptDecisionImprovingRefinement(rows []observation, current, trial constants) constants {
	currentCorrect, total := decisionAccuracy(rows, func(row observation) float64 {
		return estimatedNS(row, current)
	})
	trialCorrect, _ := decisionAccuracy(rows, func(row observation) float64 {
		return estimatedNS(row, trial)
	})
	if total > 0 && trialCorrect > currentCorrect {
		return trial
	}
	return current
}

func fitBroadTextResidualPerByte(rows []observation, c constants) (int64, bool) {
	values := make([]float64, 0)
	for _, row := range rows {
		if row.censored || row.component != "broad_text_bytes" || row.plan != "candidate_keyset" ||
			len(row.x) <= 14 || row.x[14] <= 0 {
			continue
		}
		without := c
		without.broadTextMatchByteNS = 0
		values = append(values, math.Max(1, (row.y-estimatedNS(row, without))/row.x[14]))
	}
	if len(values) == 0 {
		return 0, false
	}
	sort.Float64s(values)
	return int64(math.Round(values[len(values)/2])), true
}

func fitOrderedScanInvocation(rows []observation, c constants) (int64, bool) {
	candidates := make(map[string]observation)
	for _, row := range rows {
		if !row.censored && row.component == "" && row.plan == "candidate_keyset" {
			candidates[row.caseName] = row
		}
	}
	values := make([]float64, 0)
	for _, row := range rows {
		candidate, paired := candidates[row.caseName]
		if row.censored || row.component != "" ||
			(row.plan != "driver_order_membership_probe" && row.plan != "ordered_batch_accept") ||
			!paired || len(row.x) <= 15 || row.x[15] <= 0 {
			continue
		}
		without := c
		without.orderedScanInvocationNS = 0
		/* Both alternatives execute the surrounding query. Their difference
		isolates ordered scan startup without charging common scan and result work
		to every adaptive batch. */
		values = append(values, math.Max(1, (row.y-candidate.y-
			(estimatedNS(row, without)-estimatedNS(candidate, without)))/row.x[15]))
	}
	if len(values) == 0 {
		return 0, false
	}
	sort.Float64s(values)
	return int64(math.Round(values[len(values)/2])), true
}

func fitOrderedRecsetSortUnit(rows []observation, c constants) (int64, bool) {
	direct := make(map[string]observation)
	base := make(map[string]observation)
	for _, row := range rows {
		if row.censored || row.component != "" || len(row.x) <= 17 {
			continue
		}
		switch row.plan {
		case "candidate_keyset":
			if row.x[17] > 0 {
				direct[row.caseName] = row
			}
		case "driver_order_membership_probe":
			base[row.caseName] = row
		}
	}
	values := make([]float64, 0)
	for name, directRow := range direct {
		baseRow, paired := base[name]
		if !paired {
			continue
		}
		without := c
		without.orderedRecsetSortUnitNS = 0
		/* Carrier production and result emission are shared by the forced pair.
		Subtracting the base alternative isolates the inverse-position extraction
		and sorting work used by the candidate carrier's adaptive scan. */
		residual := (directRow.y - baseRow.y) -
			(estimatedNS(directRow, without) - estimatedNS(baseRow, without))
		values = append(values, math.Max(1, residual/directRow.x[17]))
	}
	if len(values) == 0 {
		return 0, false
	}
	sort.Float64s(values)
	return int64(math.Round(values[len(values)/2])), true
}

func fitDownstreamProbeRow(rows []observation, c constants) (int64, bool) {
	byCase := make(map[string][]observation)
	for _, row := range rows {
		if !row.censored && row.component == "" && len(row.x) > 18 {
			byCase[row.caseName] = append(byCase[row.caseName], row)
		}
	}
	candidates := []int64{1, c.downstreamProbeRowNS}
	without := c
	without.downstreamProbeRowNS = 0
	for _, alternatives := range byCase {
		for leftIndex, left := range alternatives {
			for _, right := range alternatives[leftIndex+1:] {
				deltaWork := left.x[18] - right.x[18]
				if deltaWork == 0 {
					continue
				}
				value := ((left.y - right.y) -
					(estimatedNS(left, without) - estimatedNS(right, without))) / deltaWork
				if value >= 1 && !math.IsInf(value, 0) && !math.IsNaN(value) {
					candidates = append(candidates, int64(math.Round(value)))
				}
			}
		}
	}
	if len(candidates) == 2 && candidates[0] == candidates[1] {
		return 0, false
	}
	best, bestCorrect, bestError := candidates[0], -1, math.Inf(1)
	for _, candidate := range candidates {
		trial := c
		trial.downstreamProbeRowNS = candidate
		correct, _ := decisionAccuracy(rows, func(row observation) float64 {
			return estimatedNS(row, trial)
		})
		err := measureModelError(rows, func(row observation) float64 {
			return estimatedNS(row, trial)
		}).meanFactor
		if correct > bestCorrect || (correct == bestCorrect && err < bestError) {
			best, bestCorrect, bestError = candidate, correct, err
		}
	}
	return best, true
}

func fitDirectCarrierPair(rows []observation, c constants) (int64, int64, bool) {
	candidates := make(map[string]observation)
	for _, row := range rows {
		if !row.censored && row.component == "" && row.plan == "candidate_keyset" {
			candidates[row.caseName] = row
		}
	}
	x, y := make([][]float64, 0), make([]float64, 0)
	for _, row := range rows {
		candidate, paired := candidates[row.caseName]
		if row.censored || row.component != "" || row.plan != "driver_filter_join_probe" ||
			!paired || len(row.x) <= 16 || row.x[16] <= 0 {
			continue
		}
		without := c
		without.groupCacheStartupNS = 0
		without.membershipDirectProbeRowNS = 0
		/* Both variants execute the surrounding query. Fitting a direct probe from
		its absolute duration attributes all shared scan/join work to every probe
		and grossly overprices small selective drivers. Paired differences expose
		the two physical terms which do not cancel: the candidate's fixed group-cache
		startup and the driver's per-probe work. Signed features let non-negative
		least squares identify both without a hand-written crossover threshold. */
		if candidate.x[8] <= 0 {
			continue
		}
		x = append(x, []float64{candidate.x[8], -row.x[16]})
		y = append(y, candidate.y-row.y-
			(estimatedNS(candidate, without)-estimatedNS(row, without)))
	}
	if len(x) < 2 {
		return 0, 0, false
	}
	beta, err := fitNonnegative(x, y)
	if err != nil {
		return 0, 0, false
	}
	return int64(math.Round(beta[0])), int64(math.Round(beta[1])), true
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
	if row.decision == "scan_join_order" && row.plan == "legacy_join_tree" {
		// The legacy alternative is an already costed composite join tree. Its
		// planner-reported estimate is the authoritative baseline; rowFeatures
		// only expands the new scan_join_order operator into generated work units.
		return row.currentEstimate
	}
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
		float64(c.broadTextMatchRowNS),
		float64(c.broadTextMatchByteNS),
		float64(c.orderedScanInvocationNS),
		float64(c.membershipDirectProbeRowNS),
		float64(c.orderedRecsetSortUnitNS),
		float64(c.downstreamProbeRowNS),
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
		if row.censored || row.component != "" {
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

func filterRankObservations(rows []observation) []observation {
	filtered := make([]observation, 0, len(rows))
	for _, row := range rows {
		if row.component == "" {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func filterDecisionObservations(rows []observation, decision string) []observation {
	filtered := make([]observation, 0, len(rows))
	for _, row := range rows {
		if row.decision == decision {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func filterCarrierObservations(rows []observation) []observation {
	directCases := make(map[string]bool)
	for _, row := range rows {
		if row.plan == "driver_filter_join_probe" {
			directCases[row.caseName] = true
		}
	}
	filtered := make([]observation, 0, len(rows))
	for _, row := range rows {
		if row.plan != "ordered_batch_accept" && row.plan != "prefiltered_candidate_keyset" &&
			!directCases[row.caseName] {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func decisionAlternatives(rows []observation) (map[string]map[string]observation, error) {
	groups := make(map[string]map[string]observation)
	decisions := make(map[string]string)
	for _, row := range rows {
		if groups[row.caseName] == nil {
			groups[row.caseName] = make(map[string]observation)
		}
		if _, duplicate := groups[row.caseName][row.plan]; duplicate {
			return nil, fmt.Errorf("duplicate %s observation for %q", row.plan, row.caseName)
		}
		if existing := decisions[row.caseName]; existing != "" && existing != row.decision {
			return nil, fmt.Errorf("mixed decision families for %q", row.caseName)
		}
		decisions[row.caseName] = row.decision
		switch row.plan {
		case "candidate_keyset", "driver_order_membership_probe", "driver_filter_join_probe", "ordered_batch_accept", "prefiltered_candidate_keyset":
			groups[row.caseName][row.plan] = row
		case "legacy_join_tree", "scan_join_order":
			if row.decision != "scan_join_order" {
				return nil, fmt.Errorf("plan %q belongs to scan_join_order, got decision %q", row.plan, row.decision)
			}
			groups[row.caseName][row.plan] = row
		default:
			return nil, fmt.Errorf("unsupported plan %q", row.plan)
		}
	}
	for name, plans := range groups {
		if decisions[name] == "scan_join_order" {
			if _, legacy := plans["legacy_join_tree"]; !legacy {
				return nil, fmt.Errorf("incomplete ordered join alternatives for %q", name)
			}
			if _, ordered := plans["scan_join_order"]; !ordered {
				return nil, fmt.Errorf("incomplete ordered join alternatives for %q", name)
			}
			continue
		}
		if _, ok := plans["candidate_keyset"]; !ok {
			return nil, fmt.Errorf("incomplete alternatives for %q", name)
		}
		_, ordered := plans["driver_order_membership_probe"]
		_, direct := plans["driver_filter_join_probe"]
		if !ordered && !direct {
			return nil, fmt.Errorf("incomplete alternatives for %q", name)
		}
	}
	return groups, nil
}

func winningPlan(plans map[string]observation, estimate func(observation) float64) string {
	winner := ""
	best := math.Inf(1)
	for plan, row := range plans {
		if value := estimate(row); value < best {
			winner, best = plan, value
		}
	}
	return winner
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
	groups, err := decisionAlternatives(filterRankObservations(rows))
	if err != nil {
		return 0, 0
	}
	correct := 0
	for _, plans := range groups {
		actualWinner := winningPlan(plans, func(row observation) float64 { return row.y })
		estimatedWinner := winningPlan(plans, estimate)
		if actualWinner == estimatedWinner || plansStatisticallyEquivalent(plans, actualWinner, estimatedWinner) {
			correct++
		}
	}
	return correct, len(groups)
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
	signals := 0
	for _, pair := range pairs {
		difference := math.Abs(pair.driver.y - pair.candidate.y)
		noise := 3 * (pair.driver.noiseNS + pair.candidate.noiseNS)
		if difference > noise {
			signals++
		}
	}
	if signals == 0 {
		return errors.New("calibration workload has no carrier A/B signal outside its measured noise")
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
	groups, err := decisionAlternatives(filterRankObservations(rows))
	if err != nil {
		return err
	}
	for name, plans := range groups {
		actualWinner := winningPlan(plans, func(row observation) float64 { return row.y })
		estimatedWinner := winningPlan(plans, func(row observation) float64 { return estimatedNS(row, c) })
		if actualWinner != estimatedWinner && !plansStatisticallyEquivalent(plans, actualWinner, estimatedWinner) {
			actualRow := plans[actualWinner]
			estimatedRow := plans[estimatedWinner]
			return fmt.Errorf("calibrated ordering disagrees for %q: actual winner=%s (measured %.0f ns, estimated %.0f ns), estimated winner=%s (measured %.0f ns, estimated %.0f ns)",
				name, actualWinner, actualRow.y, estimatedNS(actualRow, c),
				estimatedWinner, estimatedRow.y, estimatedNS(estimatedRow, c))
		}
	}
	return nil
}

func plansStatisticallyEquivalent(plans map[string]observation, left, right string) bool {
	leftRow, leftOK := plans[left]
	rightRow, rightOK := plans[right]
	if !leftOK || !rightOK || leftRow.censored || rightRow.censored {
		return false
	}
	if math.Max(leftRow.y, rightRow.y) <= calibrationDecisionRiskBudgetNS {
		return true
	}
	difference := math.Abs(leftRow.y - rightRow.y)
	noise := 3 * (leftRow.noiseNS + rightRow.noiseNS)
	// A one-repetition race has no sample distribution and therefore reports a
	// zero MAD. Do not pretend that this makes two wall-clock measurements exact:
	// a narrow relative floor keeps an incidental winner flip from rejecting an
	// otherwise valid model, while materially different alternatives still fail.
	if noise == 0 {
		noise = 0.05 * math.Min(leftRow.y, rightRow.y)
	}
	return difference <= noise
}

func printDecisionOrdering(rows []observation, c constants) {
	groups, err := decisionAlternatives(filterRankObservations(rows))
	if err != nil {
		return
	}
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		plans := groups[name]
		actualWinner := winningPlan(plans, func(row observation) float64 { return row.y })
		estimatedWinner := winningPlan(plans, func(row observation) float64 { return estimatedNS(row, c) })
		fmt.Printf("decision %-40s actual=%-30s estimated=%-30s alternatives=%d\n",
			name, actualWinner, estimatedWinner, len(plans))
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
		"planner_membership_broad_text_match_row_ns",
		"planner_membership_broad_text_match_byte_ns",
		"planner_membership_ordered_scan_invocation_ns",
		"planner_membership_ordered_recset_sort_unit_ns",
		"planner_membership_downstream_probe_row_ns",
		"planner_scalar_presence_probe_row_ns",
		"planner_membership_direct_probe_row_ns",
	}
	values := make([]int64, len(names))
	content := string(data)
	for i, name := range names {
		prefix := "(define " + name + " "
		start := strings.Index(content, prefix)
		if start < 0 {
			// A one-nanosecond floor lets the first run fit new membership and
			// ordered-scan coefficients from their physical-consumer observations.
			if name == "planner_membership_direct_probe_row_ns" ||
				name == "planner_membership_ordered_scan_invocation_ns" ||
				name == "planner_membership_ordered_recset_sort_unit_ns" ||
				name == "planner_membership_downstream_probe_row_ns" {
				values[i] = 1
				continue
			}
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
		orderedDriverInputNS:       values[12],
		broadTextMatchRowNS:        values[13],
		broadTextMatchByteNS:       values[14],
		orderedScanInvocationNS:    values[15],
		orderedRecsetSortUnitNS:    values[16],
		downstreamProbeRowNS:       values[17],
		scalarPresenceProbeRowNS:   values[18],
		membershipDirectProbeRowNS: values[19],
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
(define planner_scalar_presence_probe_row_ns %d)
(define planner_direct_presence_probe_cost (lambda (probe_rows)
	(planner_cost 0 0 (* probe_rows planner_scalar_presence_probe_row_ns) 0 0 0 0 0 probe_rows 0.75)))

(define planner_membership_direct_probe_row_ns %d)
(define planner_membership_direct_probe_cost (lambda (probe_rows)
	(planner_cost 0 0 (* probe_rows planner_membership_direct_probe_row_ns) 0 0 0 0 0 probe_rows 0.75)))

(define planner_membership_downstream_probe_row_ns %d)
(define planner_membership_downstream_probe_cost (lambda (probe_rows)
	(planner_cost 0 0 (* probe_rows planner_membership_downstream_probe_row_ns) 0 0 0 0 0 probe_rows 0.75)))

(define planner_presence_carrier_cost (lambda (domain_rows probe_rows)
	(planner_cost 1421611 (* probe_rows 136938) 0 0 0 0
		(* domain_rows 8) 0 domain_rows 0.65)))

(define planner_recset_carrier_cost (lambda (domain_rows carrier_rows)
	(planner_cost 365607 0 0 0 0 (* carrier_rows 17681)
		(* carrier_rows 1) 0 carrier_rows 0.6)))

(define planner_membership_scan_invocation_ns %d)
(define planner_membership_scan_row_ns %d)
(define planner_membership_filter_column_row_ns %d)
(define planner_membership_map_column_row_ns %d)
(define planner_membership_expression_operation_row_ns %d)
(define planner_membership_broad_text_match_row_ns %d)
(define planner_membership_broad_text_match_byte_ns %d)
(define planner_membership_recset_startup_ns %d)
(define planner_membership_recset_build_row_ns %d)
(define planner_membership_recset_probe_row_ns %d)
(define planner_membership_recset_aggregate_row_ns %d)
(define planner_membership_group_cache_startup_ns %d)
(define planner_membership_group_cache_build_row_ns %d)
(define planner_membership_group_cache_probe_row_ns %d)
(define planner_membership_ordered_driver_input_row_ns %d)
(define planner_membership_ordered_scan_invocation_ns %d)
(define planner_membership_ordered_recset_sort_unit_ns %d)
/* END GENERATED COST CONSTANTS */`, c.scalarPresenceProbeRowNS, c.membershipDirectProbeRowNS,
		c.downstreamProbeRowNS,
		c.scanInvocationNS, c.scanRowNS,
		c.filterColumnRowNS, c.mapColumnRowNS, c.expressionOperationNS, c.broadTextMatchRowNS, c.broadTextMatchByteNS,
		c.recsetStartupNS, c.recsetBuildRowNS, c.recsetProbeRowNS,
		c.recsetAggregateRowNS, c.groupCacheStartupNS, c.groupCacheBuildRowNS,
		c.groupCacheProbeRowNS, c.orderedDriverInputNS, c.orderedScanInvocationNS,
		c.orderedRecsetSortUnitNS)
	return os.WriteFile(path, []byte(content[:begin]+block+content[end:]), 0o644)
}
