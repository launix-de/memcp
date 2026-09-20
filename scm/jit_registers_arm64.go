//go:build arm64

/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.
*/

package scm

// On arm64 the numeric value is the architectural register encoding. Legacy
// amd64-spelled aliases remain temporarily available to generated emitters;
// common lowering must use JITContext register roles instead.
const (
	RegRAX Reg = 0 // X0: first result / first ABI word
	RegRBX Reg = 1 // X1: second result / second ABI word
	RegRCX Reg = 2 // X2: third ABI word
	RegRDX Reg = 3
	RegRSI Reg = 4
	RegRDI Reg = 5
	RegR8  Reg = 6
	RegR9  Reg = 7
	RegR10 Reg = 8
	RegR11 Reg = 16 // IP0, backend scratch
	RegR12 Reg = 19 // callee-saved slice base
	RegR13 Reg = 20
	RegR14 Reg = 28 // Go g
	RegR15 Reg = 27 // Go toolchain temporary
	RegRBP Reg = 29
	RegRSP Reg = 31
)

const (
	RegX0 Reg = 32 + iota
	RegX1
	RegX2
	RegX3
	RegX4
	RegX5
	RegX6
	RegX7
	RegX8
	RegX9
	RegX10
	RegX11
	RegX12
	RegX13
	RegX14
	RegX15
)

const (
	jitFirstFPReg          = RegX0
	jitLastFPReg           = Reg(63)
	jitLastGPReg           = Reg(31)
	jitRegisterCount       = 64
	jitSupportsCalibration = false
)
