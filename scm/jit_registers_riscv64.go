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
	RegRDX Reg = 26 // Go closure context
	RegRSI Reg = 14
	RegRDI Reg = 15
	RegR8  Reg = 16
	RegR9  Reg = 17
	RegR10 Reg = 18
	RegR11 Reg = 30 // backend scratch; X31 is reserved by the Go toolchain
	RegR12 Reg = 28 // slice base, outside the Go ABI argument bank
	RegR13 Reg = 19
	RegR14 Reg = 27 // Go g
	RegR15 Reg = 25
	RegRBP Reg = 24 // dedicated frame base, outside Go argument registers
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
	jitSupportsCalibration = true
)

const jitPortableBackend = true

// Persistent homes prefer registers without fixed result/scratch roles.
func jitArchRegisterBank() JITRegisterBank {
	return JITRegisterBank{
		Registers:        [16]Reg{19, 18, 17, 16, 15, 14, 12, 26, 10, 11},
		Count:            10,
		TemporaryReserve: 7,
	}
}

func jitArchFPRegisterBank() JITRegisterBank {
	return JITRegisterBank{
		Registers: [16]Reg{RegX2, RegX3, RegX4, RegX5, RegX6, RegX7, RegX8,
			RegX9, RegX10, RegX11, RegX12, RegX13, RegX14},
		Count:            13,
		TemporaryReserve: 2,
	}
}

func jitArchFreeGPRegs() uint64 {
	// Exclude A0/A1 results, SP, frame base X24, scratch X30, slice base X28,
	// g X27, comparison snapshots X5-X7 and address temporaries X29/X31.
	return uint64(1)<<12 | uint64(1)<<26 | uint64(1)<<14 | uint64(1)<<15 |
		uint64(1)<<16 | uint64(1)<<17 | uint64(1)<<18 | uint64(1)<<19 | uint64(1)<<25
}
