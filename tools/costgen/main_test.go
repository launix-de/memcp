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
	want := constants{startupNS: 1000, candidateScanRowNS: 5, candidateMatchRowNS: 20, driverRowNS: 80}
	rows := []observation{
		{x: []float64{1, 100, 10, 0}, y: 1000 + 100*5 + 10*20},
		{x: []float64{1, 1000, 100, 0}, y: 1000 + 1000*5 + 100*20},
		{x: []float64{1, 1000, 800, 0}, y: 1000 + 1000*5 + 800*20},
		{x: []float64{1, 0, 0, 100}, y: 1000 + 100*80},
		{x: []float64{1, 0, 0, 1000}, y: 1000 + 1000*80},
		{x: []float64{1, 0, 0, 4000}, y: 1000 + 4000*80},
	}
	got, err := solve(rows)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("solve() = %+v, want %+v", got, want)
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
