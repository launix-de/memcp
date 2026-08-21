/*
Copyright (C) 2026  Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

package main

import "testing"

func TestSolvePhysicalCostEquation(t *testing.T) {
	want := constants{
		startupNS: 0, candidateScanRowNS: 5, candidateRecsetRowNS: 20,
		driverCacheBuildRowNS: 30, driverCacheProbeRowNS: 80,
	}
	rows := []observation{
		{caseName: "a", plan: "candidate_keyset", x: []float64{100, 10, 0, 0}, y: 100*5 + 10*20},
		{caseName: "b", plan: "candidate_keyset", x: []float64{1000, 100, 0, 0}, y: 1000*5 + 100*20},
		{caseName: "c", plan: "candidate_keyset", x: []float64{1000, 800, 0, 0}, y: 1000*5 + 800*20},
		{caseName: "a", plan: "driver_order_membership_probe", x: []float64{100, 0, 10, 100}, y: 100*5 + 10*30 + 100*80},
		{caseName: "b", plan: "driver_order_membership_probe", x: []float64{1000, 0, 100, 1000}, y: 1000*5 + 100*30 + 1000*80},
		{caseName: "c", plan: "driver_order_membership_probe", x: []float64{1000, 0, 800, 4000}, y: 1000*5 + 800*30 + 4000*80},
	}
	got, err := solve(rows)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("solve() = %+v, want %+v", got, want)
	}
	if err := validateDecisionOrdering(rows, got); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRowsRejectsDecisionPlanMismatch(t *testing.T) {
	estimated, candidateInput, candidateRows, driverRows := 100.0, 1000.0, 100.0, 4000.0
	rows := []calibrationRow{
		{
			DecisionID: "membership_carrier:1", Decision: "membership_carrier",
			Plan: "candidate_keyset", OperatorFamily: "driver_order_membership_probe",
			OperatorConsistent: false, EstimatedNS: &estimated, ActualNS: 200,
			CandidateInputRows: &candidateInput, CandidateRows: &candidateRows,
			DriverRows: &driverRows, ResultEqual: true,
		},
	}
	if err := validateRows(rows); err == nil {
		t.Fatal("validateRows accepted a chosen/emitted operator mismatch")
	}
}
