/*
Copyright (C) 2026  Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestFitNonnegativePhysicalCostEquation(t *testing.T) {
	coefficients := []float64{2, 3, 4, 5, 6}
	x := make([][]float64, len(coefficients))
	y := make([]float64, len(coefficients))
	for i, coefficient := range coefficients {
		features := make([]float64, len(coefficients))
		features[i] = 100
		x[i] = features
		y[i] = 100 * coefficient
	}
	got, err := fitNonnegative(x, y)
	if err != nil {
		t.Fatal(err)
	}
	for i := range got {
		if got[i] != coefficients[i] {
			t.Fatalf("fit[%d] = %f, want %f", i, got[i], coefficients[i])
		}
	}
}

func TestSolveKeepsBaselineWhenLegacyCarrierFitIsUnderdetermined(t *testing.T) {
	baseline := constants{scanRowNS: 123}
	got, err := solve(nil, nil, baseline)
	if err != nil {
		t.Fatal(err)
	}
	if got.scanRowNS != baseline.scanRowNS {
		t.Fatalf("scan row coefficient = %d, want preserved baseline %d", got.scanRowNS, baseline.scanRowNS)
	}
}

func TestPatchQueryplanMigratesGeneratedConstantSchema(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/queryplan.scm"
	original := "before\n" + beginMarker + " legacy\n" + endMarker + "\nafter\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := patchQueryplan(path, constants{orderedRecsetSortUnitNS: 7}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "(define planner_membership_ordered_recset_sort_unit_ns 7)") {
		t.Fatalf("migrated block does not contain new constant: %s", content)
	}
	if !strings.HasPrefix(content, "before\n") || !strings.HasSuffix(content, "\nafter\n") {
		t.Fatalf("patch changed content outside generated block: %q", content)
	}
}

func TestValidateRowsRejectsDecisionPlanMismatch(t *testing.T) {
	estimated, candidateInput, candidateRows, driverRows := 100.0, 1000.0, 100.0, 4000.0
	rows := []calibrationRow{
		{
			DecisionID: "membership_carrier:1", Decision: "membership_carrier",
			Plan: "candidate_keyset", OperatorFamily: "driver_order_membership_probe",
			OperatorConsistent: false, EstimatedNS: &estimated, WholeQueryExecutionNS: 200,
			CandidateInputRows: &candidateInput, CandidateRows: &candidateRows,
			DriverRows: &driverRows, ResultEqual: true,
		},
	}
	if err := validateRows(rows); err == nil {
		t.Fatal("validateRows accepted a chosen/emitted operator mismatch")
	}
}

func TestRowFeaturesKeepsBroadTextRowsAndBytesSeparate(t *testing.T) {
	value := func(v float64) *float64 { return &v }
	row := calibrationRow{
		Plan: "candidate_keyset", Consumer: "order_limit",
		CandidateInputRows: value(100), CandidateRows: value(10), ProjectedDriverRows: value(20),
		DriverInputRows: value(1000), DriverRows: value(5), ExpectedDriverRowsVisited: value(50),
		CandidateScanInvocations: value(1), CandidateFilterColumns: value(1),
		CandidateMapColumns: value(1), CandidateCacheMapColumns: value(2),
		CandidateExpressionOperations: value(1), CandidateBroadTextMatchRows: value(100),
		CandidateBroadTextMatchBytes: value(819200), DriverScanInvocations: value(1),
		DriverFilterColumns: value(0), DriverMapColumns: value(1), DriverExpressionOperations: value(0),
	}
	features, err := rowFeatures(row)
	if err != nil {
		t.Fatal(err)
	}
	if features[13] != 100 || features[14] != 819200 {
		t.Fatalf("broad text features = (%v, %v), want (100, 819200)", features[13], features[14])
	}
}

func TestRowFeaturesModelsOrderedJoinAlternatives(t *testing.T) {
	value := func(v float64) *float64 { return &v }
	base := calibrationRow{
		Decision: "scan_join_order", JoinInputRows: value(25_000),
		JoinEstimatedRows: value(4_000), JoinOutputRows: value(72),
		JoinTableCount: value(2), JoinLegacyProbeRows: value(600),
	}
	base.Plan = "scan_join_order"
	scanFeatures, err := rowFeatures(base)
	if err != nil {
		t.Fatal(err)
	}
	if scanFeatures[0] != 1 || scanFeatures[1] != 25_000 || scanFeatures[15] != 1 ||
		scanFeatures[3] != 25_072 || scanFeatures[4] != 4_000 || scanFeatures[18] != 0 {
		t.Fatalf("ordered join scan features = %v", scanFeatures)
	}
	base.Plan = "legacy_join_tree"
	legacyFeatures, err := rowFeatures(base)
	if err != nil {
		t.Fatal(err)
	}
	if legacyFeatures[18] != 600 {
		t.Fatalf("legacy ordered join probe work = %v, want 600", legacyFeatures[18])
	}
}

func TestRowFeaturesModelsAdaptiveOrderedBatchWork(t *testing.T) {
	value := func(v float64) *float64 { return &v }
	row := calibrationRow{
		Plan: "ordered_batch_accept", Consumer: "order_limit",
		CandidateInputRows: value(1000), CandidateRows: value(50), ProjectedDriverRows: value(50),
		DriverInputRows: value(1000), DriverRows: value(100), ExpectedDriverRowsVisited: value(100),
		Limit: value(10), Offset: value(0), ProbeBranches: value(2), CandidateScanInvocations: value(2),
		CandidateFilterColumns: value(2), CandidateMapColumns: value(1), CandidateCacheMapColumns: value(2),
		CandidateExpressionOperations: value(3), CandidateBroadTextMatchRows: value(1000),
		CandidateBroadTextMatchBytes: value(8192000), DriverScanInvocations: value(1),
		DriverFilterColumns: value(0), DriverMapColumns: value(1), DriverExpressionOperations: value(0),
	}
	features, err := rowFeatures(row)
	if err != nil {
		t.Fatal(err)
	}
	if features[0] != 9 || features[1] != 1740 || features[2] != 800 ||
		features[4] != 1200 || features[6] != 1240 {
		t.Fatalf("adaptive features = %v", features)
	}
	if features[13] != 400 || features[14] != 3276800 {
		t.Fatalf("repeated broad text features = (%v, %v), want (400, 3276800)",
			features[13], features[14])
	}
	if features[12] != 4 {
		t.Fatalf("ordered batch driver work = %v, want 4 rows²/1M", features[12])
	}
	if features[15] != 4 {
		t.Fatalf("ordered scan invocations = %v, want 4", features[15])
	}
	row.DriverOrderPartitioned = true
	partitionedFeatures, err := rowFeatures(row)
	if err != nil {
		t.Fatal(err)
	}
	if partitionedFeatures[12] != 200 {
		t.Fatalf("partition-pruned ordered batch work = %v, want twice the 100-row prefix",
			partitionedFeatures[12])
	}
}

func TestRowFeaturesModelsDirectPresenceProbes(t *testing.T) {
	value := func(v float64) *float64 { return &v }
	row := calibrationRow{
		Plan: "driver_filter_join_probe", Consumer: "filter",
		CandidateInputRows: value(100), CandidateRows: value(10), ProjectedDriverRows: value(20),
		DriverInputRows: value(1000), DriverRows: value(25), ExpectedDriverRowsVisited: value(25),
		ProbeBranches: value(3), CandidateScanInvocations: value(1), CandidateFilterColumns: value(1),
		CandidateMapColumns: value(1), CandidateCacheMapColumns: value(2),
		CandidateExpressionOperations: value(1), CandidateBroadTextMatchRows: value(0),
		CandidateBroadTextMatchBytes: value(0), DriverScanInvocations: value(1),
		DriverFilterColumns: value(0), DriverMapColumns: value(1), DriverExpressionOperations: value(0),
		DownstreamProbeBranches: value(2),
	}
	features, err := rowFeatures(row)
	if err != nil {
		t.Fatal(err)
	}
	if len(features) != 19 || features[16] != 75 {
		t.Fatalf("direct presence features = %v, want 75 probes", features)
	}
}

func TestRowFeaturesChargesOrderedProjectedRecsetSortWork(t *testing.T) {
	value := func(v float64) *float64 { return &v }
	row := calibrationRow{
		Plan: "candidate_keyset", Consumer: "order_limit",
		CandidateInputRows: value(100), CandidateRows: value(10), ProjectedDriverRows: value(25),
		DriverInputRows: value(1000), DriverRows: value(3), ExpectedDriverRowsVisited: value(120),
		CandidateScanInvocations: value(1), CandidateFilterColumns: value(1),
		CandidateMapColumns: value(1), CandidateCacheMapColumns: value(2),
		CandidateExpressionOperations: value(1), CandidateBroadTextMatchRows: value(0),
		CandidateBroadTextMatchBytes: value(0), DriverScanInvocations: value(1),
		DriverFilterColumns: value(0), DriverMapColumns: value(1), DriverExpressionOperations: value(0),
		DownstreamProbeBranches: value(2),
	}
	features, err := rowFeatures(row)
	if err != nil {
		t.Fatal(err)
	}
	if features[17] != 125 {
		t.Fatalf("ordered RecSet sort work = %v, want 125", features[17])
	}
	if features[18] != 50 {
		t.Fatalf("downstream probe rows = %v, want 50", features[18])
	}
}

func TestRowFeaturesModelsAdaptiveOrderedRecsetConsumer(t *testing.T) {
	value := func(v float64) *float64 { return &v }
	baseRow := calibrationRow{
		Plan: "candidate_keyset", Consumer: "order_limit",
		CandidateInputRows: value(100), CandidateRows: value(10), ProjectedDriverRows: value(100),
		DriverInputRows: value(1000), DriverRows: value(10), ExpectedDriverRowsVisited: value(100),
		Limit: value(10), Offset: value(0), ProbeBranches: value(1),
		CandidateScanInvocations: value(1), CandidateFilterColumns: value(0),
		CandidateMapColumns: value(0), CandidateCacheMapColumns: value(0),
		CandidateExpressionOperations: value(0), CandidateBroadTextMatchRows: value(0),
		CandidateBroadTextMatchBytes: value(0), DriverScanInvocations: value(1),
		DriverFilterColumns: value(0), DriverMapColumns: value(0), DriverExpressionOperations: value(0),
	}
	inverseRow := baseRow
	inverseRow.ProjectedDriverRows = value(4)
	inverse, err := rowFeatures(inverseRow)
	if err != nil {
		t.Fatal(err)
	}
	base, err := rowFeatures(baseRow)
	if err != nil {
		t.Fatal(err)
	}
	if inverse[17] != 8 || inverse[7] != 0 {
		t.Fatalf("inverse adaptive features = %v, want sort work 8 and no membership probes", inverse)
	}
	if base[17] != 0 || base[7] != 100 {
		t.Fatalf("base adaptive features = %v, want 100 membership probes and no sort work", base)
	}
}

func TestOrderedRecsetSortWorkMatchesRuntimeModel(t *testing.T) {
	if got := orderedRecsetSortWork(1); got != 1 {
		t.Fatalf("sort work(1) = %v, want 1", got)
	}
	if got := orderedRecsetSortWork(5); got != 15 {
		t.Fatalf("sort work(5) = %v, want 15", got)
	}
}

func TestFitOrderedRecsetSortUnitCancelsSharedCarrierWork(t *testing.T) {
	directA, baseA := make([]float64, 19), make([]float64, 19)
	directA[17], baseA[1] = 100, 20
	directB, baseB := make([]float64, 19), make([]float64, 19)
	directB[17], baseB[1] = 10, 200
	value, ok := fitOrderedRecsetSortUnit([]observation{
		{caseName: "broad", plan: "candidate_keyset", y: 11000, x: directA},
		{caseName: "broad", plan: "driver_order_membership_probe", y: 1000, x: baseA},
		{caseName: "sparse", plan: "candidate_keyset", y: 2000, x: directB},
		{caseName: "sparse", plan: "driver_order_membership_probe", y: 1000, x: baseB},
	}, constants{})
	if !ok || value != 100 {
		t.Fatalf("ordered RecSet sort unit = (%d, %v), want (100, true)", value, ok)
	}
}

func TestFitDownstreamProbeRowUsesDecisionOrdering(t *testing.T) {
	candidate, driver, batch := make([]float64, 19), make([]float64, 19), make([]float64, 19)
	candidate[18], driver[18], batch[18] = 100, 1000, 10
	rows := []observation{
		{caseName: "compound", decision: "membership_carrier", plan: "candidate_keyset", y: 2000, x: candidate},
		{caseName: "compound", decision: "membership_carrier", plan: "driver_order_membership_probe", y: 10000, x: driver},
		{caseName: "compound", decision: "membership_carrier", plan: "ordered_batch_accept", y: 1000, x: batch},
	}
	value, ok := fitDownstreamProbeRow(rows, constants{})
	if !ok || value <= 1 {
		t.Fatalf("downstream probe row = (%d, %v), want calibrated value above floor", value, ok)
	}
	model := constants{downstreamProbeRowNS: value}
	if err := validateDecisionOrdering(rows, model); err != nil {
		t.Fatal(err)
	}
}

func TestFitOrderedScanInvocationUsesExactBatchObservations(t *testing.T) {
	features := make([]float64, 19)
	features[15] = 4
	value, ok := fitOrderedScanInvocation([]observation{
		{caseName: "batch", plan: "candidate_keyset", y: 1000, x: make([]float64, 19)},
		{caseName: "batch", plan: "ordered_batch_accept", y: 2680, x: features},
	}, constants{})
	if !ok || value != 420 {
		t.Fatalf("ordered scan invocation = (%d, %v), want (420, true)", value, ok)
	}
}

func TestRowFeaturesModelsPrefilteredCandidateWork(t *testing.T) {
	value := func(v float64) *float64 { return &v }
	row := calibrationRow{
		Plan: "prefiltered_candidate_keyset", Consumer: "order_limit",
		CandidateInputRows: value(1000), CandidateRows: value(100), CandidateDensity: value(0.1),
		ProjectedDriverRows: value(20), DriverInputRows: value(10000), DriverRows: value(72),
		PrefilteredDriverRows: value(100), ExpectedDriverRowsVisited: value(100), ProbeBranches: value(2),
		CandidateScanInvocations: value(2), CandidateFilterColumns: value(1), CandidateMapColumns: value(1),
		CandidateCacheMapColumns: value(2), CandidateExpressionOperations: value(1),
		CandidateBroadTextMatchRows: value(1000), CandidateBroadTextMatchBytes: value(8000),
		DriverScanInvocations: value(1), DriverFilterColumns: value(1), DriverMapColumns: value(1),
		DriverExpressionOperations: value(2),
	}
	features, err := rowFeatures(row)
	if err != nil {
		t.Fatal(err)
	}
	// candidate work=200, matches=20, projection=420.
	if features[0] != 3 || features[1] != 10620 || features[2] != 10200 || features[5] != 1 ||
		features[3] != 420 || features[4] != 20200 || features[6] != 420 ||
		features[13] != 200 || features[14] != 1600 {
		t.Fatalf("prefiltered features = %v", features)
	}
}

func TestRowFeaturesChargesPrefilteredDownstreamWorkToLocalDriverSubset(t *testing.T) {
	value := func(v float64) *float64 { return &v }
	row := calibrationRow{
		Plan: "prefiltered_candidate_keyset", Consumer: "order_limit",
		CandidateInputRows: value(2000), CandidateRows: value(250), CandidateDensity: value(0.25),
		ProjectedDriverRows: value(1000), DriverInputRows: value(4000), DriverRows: value(72),
		PrefilteredDriverRows: value(4000), ExpectedDriverRowsVisited: value(300),
		ProbeBranches: value(2), DownstreamProbeBranches: value(1),
		CandidateScanInvocations: value(2), CandidateFilterColumns: value(1),
		CandidateMapColumns: value(1), CandidateCacheMapColumns: value(2),
		CandidateExpressionOperations: value(1), CandidateBroadTextMatchRows: value(2000),
		CandidateBroadTextMatchBytes: value(0), DriverScanInvocations: value(1),
		DriverFilterColumns: value(0), DriverMapColumns: value(4), DriverExpressionOperations: value(0),
		Limit: value(72), Offset: value(0),
	}
	features, err := rowFeatures(row)
	if err != nil {
		t.Fatal(err)
	}
	if features[18] != 4000 {
		t.Fatalf("prefiltered downstream probes = %v, want 4000", features[18])
	}
}

func TestFitDirectCarrierPairCancelsPairedCommonWork(t *testing.T) {
	candidateA, driverA := make([]float64, 19), make([]float64, 19)
	candidateA[5], candidateA[8], driverA[16] = 1, 1, 100
	candidateB, driverB := make([]float64, 19), make([]float64, 19)
	candidateB[5], candidateB[8], driverB[16] = 1, 1, 10
	startup, probe, ok := fitDirectCarrierPair([]observation{
		{caseName: "large", plan: "candidate_keyset", y: 10000, x: candidateA},
		{caseName: "large", plan: "driver_filter_join_probe", y: 42000, x: driverA},
		{caseName: "small", plan: "candidate_keyset", y: 10000, x: candidateB},
		{caseName: "small", plan: "driver_filter_join_probe", y: 9000, x: driverB},
	}, constants{})
	if !ok || startup != 4667 || probe != 367 {
		t.Fatalf("direct carrier pair = (%d, %d, %v), want (4667, 367, true)", startup, probe, ok)
	}
}

func TestDecisionAlternativesAcceptsDirectFilterConsumer(t *testing.T) {
	rows := []observation{
		{caseName: "filter", plan: "candidate_keyset"},
		{caseName: "filter", plan: "driver_filter_join_probe"},
	}
	groups, err := decisionAlternatives(rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups["filter"]) != 2 {
		t.Fatalf("unexpected alternatives: %+v", groups)
	}
}

func TestDecisionAlternativesAcceptsOrderedJoinFamily(t *testing.T) {
	rows := []observation{
		{caseName: "join", decision: "scan_join_order", plan: "legacy_join_tree"},
		{caseName: "join", decision: "scan_join_order", plan: "scan_join_order"},
	}
	groups, err := decisionAlternatives(rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups["join"]) != 2 {
		t.Fatalf("unexpected ordered join alternatives: %+v", groups)
	}
}

func TestValidateDecisionOrderingUsesOrderedJoinCostBoundary(t *testing.T) {
	legacy := func(name string, measured, estimated float64) observation {
		return observation{
			caseName: name, decision: "scan_join_order", plan: "legacy_join_tree",
			y: measured, currentEstimate: estimated, x: make([]float64, 19),
		}
	}
	ordered := func(name string, measured float64) observation {
		features := make([]float64, 19)
		features[0], features[1], features[3], features[4], features[15] = 1, 25_000, 25_001, 1, 1
		return observation{
			caseName: name, decision: "scan_join_order", plan: "scan_join_order",
			y: measured, x: features,
		}
	}
	rows := []observation{
		legacy("point", 70e6, 3.2e6), ordered("point", 230e6),
		legacy("broad", 439e6, 5.7e6), ordered("broad", 1.2e6),
	}
	constants := constants{
		scanInvocationNS: 122_080, scanRowNS: 1, mapColumnRowNS: 32,
		expressionOperationNS: 220, orderedScanInvocationNS: 3_027_639,
	}
	if err := validateDecisionOrdering(rows, constants); err != nil {
		t.Fatal(err)
	}
}

func TestMedianRowsAcceptsOrderedBatchPlans(t *testing.T) {
	runs := [][]calibrationRow{
		{{Plan: "driver_order_membership_probe", WholeQueryExecutionNS: 30}, {Plan: "ordered_batch_accept", WholeQueryExecutionNS: 20}},
		{{Plan: "driver_order_membership_probe", WholeQueryExecutionNS: 10}, {Plan: "ordered_batch_accept", WholeQueryExecutionNS: 40}},
		{{Plan: "driver_order_membership_probe", WholeQueryExecutionNS: 20}, {Plan: "ordered_batch_accept", WholeQueryExecutionNS: 30}},
	}
	rows, err := medianRows(runs)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Plan != "driver_order_membership_probe" || rows[0].WholeQueryExecutionNS != 20 ||
		rows[1].Plan != "ordered_batch_accept" || rows[1].WholeQueryExecutionNS != 30 {
		t.Fatalf("unexpected medians: %+v", rows)
	}
}

func TestValidateDecisionOrderingIncludesOrderedBatch(t *testing.T) {
	features := func(first float64) []float64 {
		values := make([]float64, 19)
		values[0] = first
		return values
	}
	rows := []observation{
		{caseName: "three-way", plan: "candidate_keyset", y: 1e9, x: features(1)},
		{caseName: "three-way", plan: "driver_order_membership_probe", y: 2e9, x: features(2)},
		{caseName: "three-way", plan: "ordered_batch_accept", y: 500e6, x: features(100)},
	}
	if err := validateDecisionOrdering(rows, constants{scanInvocationNS: 1}); err == nil {
		t.Fatal("three-way validation accepted an incorrectly ranked ordered batch plan")
	}
}

func TestDecisionOrderingAcceptsOnlyMeasuredNoiseTies(t *testing.T) {
	features := func(first float64) []float64 {
		values := make([]float64, 19)
		values[0] = first
		return values
	}
	rows := []observation{
		{caseName: "tie", plan: "candidate_keyset", y: 200e6, noiseNS: 10e6, x: features(2)},
		{caseName: "tie", plan: "driver_order_membership_probe", y: 220e6, noiseNS: 10e6, x: features(1)},
	}
	if err := validateDecisionOrdering(rows, constants{scanInvocationNS: 1}); err != nil {
		t.Fatalf("noise-equivalent alternatives rejected: %v", err)
	}
	rows[1].y = 400e6
	if err := validateDecisionOrdering(rows, constants{scanInvocationNS: 1}); err == nil {
		t.Fatal("materially slower estimated winner accepted as a noise tie")
	}
}

func TestDecisionOrderingAllowsNarrowSingleRaceUncertainty(t *testing.T) {
	features := func(first float64) []float64 {
		values := make([]float64, 19)
		values[0] = first
		return values
	}
	rows := []observation{
		{caseName: "single-race", plan: "candidate_keyset", y: 200e6, x: features(2)},
		{caseName: "single-race", plan: "driver_order_membership_probe", y: 400e6, x: features(200)},
		{caseName: "single-race", plan: "ordered_batch_accept", y: 208e6, x: features(1)},
	}
	if err := validateDecisionOrdering(rows, constants{scanInvocationNS: 1}); err != nil {
		t.Fatalf("narrow single-race uncertainty rejected: %v", err)
	}
	rows[2].y = 212e6
	if err := validateDecisionOrdering(rows, constants{scanInvocationNS: 1}); err == nil {
		t.Fatal("material single-race difference accepted as uncertainty")
	}
}

func TestDecisionOrderingAllowsOnlyCompleteSubBudgetAlternatives(t *testing.T) {
	features := func(first float64) []float64 {
		values := make([]float64, 19)
		values[0] = first
		return values
	}
	rows := []observation{
		{caseName: "low-risk", plan: "candidate_keyset", y: 10e6, x: features(2)},
		{caseName: "low-risk", plan: "driver_order_membership_probe", y: 200e6, x: features(200)},
		{caseName: "low-risk", plan: "ordered_batch_accept", y: 90e6, x: features(1)},
	}
	if err := validateDecisionOrdering(rows, constants{scanInvocationNS: 1}); err != nil {
		t.Fatalf("sub-budget alternative ordering rejected: %v", err)
	}
	rows[2].y = 101e6
	if err := validateDecisionOrdering(rows, constants{scanInvocationNS: 1}); err == nil {
		t.Fatal("over-budget alternative ordering accepted")
	}
}

func TestResidualRefinementMustImproveDecisionPool(t *testing.T) {
	features := func(first, ordered float64) []float64 {
		values := make([]float64, 19)
		values[0], values[15] = first, ordered
		return values
	}
	rows := []observation{
		{caseName: "stable", plan: "candidate_keyset", y: 200e6, x: features(1, 0)},
		{caseName: "stable", plan: "driver_order_membership_probe", y: 400e6, x: features(0, 1)},
	}
	current := constants{scanInvocationNS: 1, orderedScanInvocationNS: 2}
	trial := current
	trial.orderedScanInvocationNS = 100
	if got := acceptDecisionImprovingRefinement(rows, current, trial); got != current {
		t.Fatalf("equal-accuracy residual fit replaced selected baseline: %+v", got)
	}

	wrong := current
	better := wrong
	better.orderedScanInvocationNS = 0
	rows[0].y, rows[1].y = 400e6, 200e6
	if got := acceptDecisionImprovingRefinement(rows, wrong, better); got != better {
		t.Fatalf("decision-improving residual fit was rejected: %+v", got)
	}
}

func TestRaceCalibrationVariantsCancelsSlowerPlan(t *testing.T) {
	const decisionID = "membership_carrier:test"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		plan := "candidate_keyset"
		if strings.Contains(string(body), "driver_order_membership_probe") {
			plan = "driver_order_membership_probe"
			<-r.Context().Done()
			return
		}
		time.Sleep(10 * time.Millisecond)
		fmt.Fprintf(w, `{"decision_id":%q,"decision":"membership_carrier","plan":%q,"operator_family":%q,"operator_consistent":true,"estimated_ns":100,"whole_query_execution_ns":1000000,"candidate_input_rows":100,"candidate_rows":10,"candidate_density":0.1,"projected_driver_rows":100,"driver_input_rows":1000,"driver_rows":10,"expected_driver_rows_visited":100,"rows":1,"result_hash":"same"}`,
			decisionID, plan, plan)
	}))
	defer server.Close()

	fastEstimate, slowEstimate := 100.0, 1000.0
	memcp := &memcpServer{baseURL: server.URL}
	started := time.Now()
	rows, err := raceCalibrationVariants(memcp, "SELECT 1", decisionID,
		[]string{"candidate_keyset", "driver_order_membership_probe"},
		map[string]*float64{"candidate_keyset": &fastEstimate, "driver_order_membership_probe": &slowEstimate}, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(started) > 250*time.Millisecond {
		t.Fatalf("race did not cancel the slower plan promptly: %s", time.Since(started))
	}
	if len(rows) != 2 || rows[0].Plan != "candidate_keyset" || rows[0].TimedOut ||
		rows[1].Plan != "driver_order_membership_probe" || !rows[1].TimedOut {
		t.Fatalf("unexpected race rows: %+v", rows)
	}
	if rows[1].LowerBoundNS <= rows[0].WholeQueryExecutionNS {
		t.Fatalf("timeout lower bound %f must exceed winner time %f", rows[1].LowerBoundNS, rows[0].WholeQueryExecutionNS)
	}
}
