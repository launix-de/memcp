//go:build !amd64 && !arm64 && !riscv64

/*
Copyright (C) 2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.
*/

package scm

func jitArchEmitReturn(*JITContext)             { jitUnsupportedArchitecture() }
func jitArchEmitLeafProlog(*JITContext)         { jitUnsupportedArchitecture() }
func jitArchEmitLeafEpilog(*JITContext)         { jitUnsupportedArchitecture() }
func jitFlushInstructionCache(uintptr, uintptr) {}

func jitUnsupportedArchitecture() { panic("jit: unsupported architecture") }

func jitArchEmitMovRegReg(*JITContext, Reg, Reg)      { jitUnsupportedArchitecture() }
func jitArchEmitMovRegImm64(*JITContext, Reg, uint64) { jitUnsupportedArchitecture() }
func jitArchEmitLoad64(*JITContext, Reg, Reg, int32)  { jitUnsupportedArchitecture() }
func jitArchEmitStore64(*JITContext, Reg, Reg, int32) { jitUnsupportedArchitecture() }
func jitArchEmitLoad8(*JITContext, Reg, Reg, int32)   { jitUnsupportedArchitecture() }
func jitArchEmitLoad16(*JITContext, Reg, Reg, int32)  { jitUnsupportedArchitecture() }
func jitArchEmitLoad32(*JITContext, Reg, Reg, int32)  { jitUnsupportedArchitecture() }
func jitArchEmitStore8(*JITContext, Reg, Reg, int32)  { jitUnsupportedArchitecture() }
func jitArchEmitStore16(*JITContext, Reg, Reg, int32) { jitUnsupportedArchitecture() }
func jitArchEmitStore32(*JITContext, Reg, Reg, int32) { jitUnsupportedArchitecture() }
func jitArchEmitZeroReg(*JITContext, Reg)             { jitUnsupportedArchitecture() }
func jitArchEmitOrInt64(*JITContext, Reg, Reg)        { jitUnsupportedArchitecture() }
func jitArchEmitAndInt64(*JITContext, Reg, Reg)       { jitUnsupportedArchitecture() }
func jitArchEmitXorInt64(*JITContext, Reg, Reg)       { jitUnsupportedArchitecture() }
func jitArchEmitAddInt64(*JITContext, Reg, Reg)       { jitUnsupportedArchitecture() }
func jitArchEmitSubInt64(*JITContext, Reg, Reg)       { jitUnsupportedArchitecture() }
func jitArchEmitAddInt32(*JITContext, Reg, Reg)       { jitUnsupportedArchitecture() }
func jitArchEmitSubInt32(*JITContext, Reg, Reg)       { jitUnsupportedArchitecture() }
func jitArchEmitMulInt64(*JITContext, Reg, Reg)       { jitUnsupportedArchitecture() }
