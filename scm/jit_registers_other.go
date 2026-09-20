//go:build !amd64 && !arm64 && !riscv64

/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.
*/

package scm

// Unsupported targets still compile the generated emitter callbacks. These
// identifiers are never executed because jitEnabled is false.
const (
	RegRAX Reg = iota
	RegRCX
	RegRDX
	RegRBX
	RegRSP
	RegRBP
	RegRSI
	RegRDI
	RegR8
	RegR9
	RegR10
	RegR11
	RegR12
	RegR13
	RegR14
	RegR15
	RegX0
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
	jitLastFPReg           = RegX15
	jitLastGPReg           = RegR15
	jitRegisterCount       = 32
	jitSupportsCalibration = false
)
