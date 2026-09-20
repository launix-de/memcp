// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later

#include "textflag.h"

// jitFlushInstructionCache makes newly emitted instructions visible to every
// core. Four-byte stepping is conservative for every architecturally valid
// cache-line size and keeps this helper independent of runtime internals.
TEXT ·jitFlushInstructionCache(SB), NOSPLIT, $0-16
	MOVD start+0(FP), R0
	MOVD end+8(FP), R1
	MOVD R0, R2
dcache:
	CMP R1, R2
	BHS dcache_done
	DC CVAU, R2
	ADD $4, R2
	B dcache
dcache_done:
	DSB $11
	MOVD R0, R2
icache:
	CMP R1, R2
	BHS icache_done
	// Go's arm64 assembler does not expose the IVAU operand spelling.
	// IC IVAU, R2 is encoded as SYS #3, C7, C5, #1, R2.
	WORD $0xd50b7522
	ADD $4, R2
	B icache
icache_done:
	DSB $11
	ISB $15
	RET
