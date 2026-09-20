//go:build riscv64

/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.
*/

package scm

// On riscv64 the numeric value is the architectural X/F register encoding.
// Go ABIInternal starts integer arguments and results in A0 (X10).
const (
	RegRAX Reg = 10 // A0: first result / first ABI word
	RegRBX Reg = 11 // A1: second result / second ABI word
	RegRCX Reg = 12 // A2: third ABI word
	RegRDX Reg = 13
	RegRSI Reg = 14
	RegRDI Reg = 15
	RegR8  Reg = 16
	RegR9  Reg = 17
	RegR10 Reg = 28
	RegR11 Reg = 30 // backend scratch; X31 is reserved by the Go toolchain
	RegR12 Reg = 18 // S2, callee-saved slice base
	RegR13 Reg = 19
	RegR14 Reg = 27 // Go g
	RegR15 Reg = 25
	RegRBP Reg = 8
	RegRSP Reg = 2
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
