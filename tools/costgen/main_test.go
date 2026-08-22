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

func TestRowFeaturesModelsAdaptiveOrderedBatchWork(t *testing.T) {
	value := func(v float64) *float64 { return &v }
	row := calibrationRow{
		Plan: "scan_order_batch_accept", Consumer: "order_limit",
		CandidateInputRows: value(1000), CandidateRows: value(50), ProjectedDriverRows: value(50),
		DriverInputRows: value(1000), DriverRows: value(100), ExpectedDriverRowsVisited: value(100),
		Limit: value(10), Offset: value(0), CandidateScanInvocations: value(1),
		CandidateFilterColumns: value(2), CandidateMapColumns: value(1), CandidateCacheMapColumns: value(2),
		CandidateExpressionOperations: value(3), CandidateBroadTextMatchRows: value(1000),
		CandidateBroadTextMatchBytes: value(8192000), DriverScanInvocations: value(1),
		DriverFilterColumns: value(0), DriverMapColumns: value(1), DriverExpressionOperations: value(0),
	}
	features, err := rowFeatures(row)
	if err != nil {
		t.Fatal(err)
	}
	if features[0] != 4 || features[1] != 300 || features[2] != 200 ||
		features[4] != 300 || features[6] != 250 {
		t.Fatalf("adaptive features = %v", features)
	}
	if features[13] != 100 || features[14] != 819200 {
		t.Fatalf("fractional broad text features = (%v, %v), want (100, 819200)",
			features[13], features[14])
	}
}

func TestMedianRowsAcceptsOrderedBatchPlans(t *testing.T) {
	runs := [][]calibrationRow{
		{{Plan: "scan_order", WholeQueryExecutionNS: 30}, {Plan: "scan_order_batch_accept", WholeQueryExecutionNS: 20}},
		{{Plan: "scan_order", WholeQueryExecutionNS: 10}, {Plan: "scan_order_batch_accept", WholeQueryExecutionNS: 40}},
		{{Plan: "scan_order", WholeQueryExecutionNS: 20}, {Plan: "scan_order_batch_accept", WholeQueryExecutionNS: 30}},
	}
	rows, err := medianRows(runs)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Plan != "scan_order" || rows[0].WholeQueryExecutionNS != 20 ||
		rows[1].Plan != "scan_order_batch_accept" || rows[1].WholeQueryExecutionNS != 30 {
		t.Fatalf("unexpected medians: %+v", rows)
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
