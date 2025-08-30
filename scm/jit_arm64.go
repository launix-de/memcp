//go:build arm64
/*
Copyright (C) 2024  Carl-Philip Hänsch

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package scm

import "unsafe"

// TODO: create this file for other architectures, too

// all code snippets fill rax+rbx with the return value
func jitReturnLiteral(value Scmer) []byte {
    code := []byte{
	    // TODO
    }
    // insert the literal into the immediate values
    *(*unsafe.Pointer)(unsafe.Pointer(&code[ 2])) = *(*unsafe.Pointer)(unsafe.Pointer(&value))
    *(*unsafe.Pointer)(unsafe.Pointer(&code[12])) = *((*unsafe.Pointer)(unsafe.Add(unsafe.Pointer(&value), 8)))
    return code
}

func jitNthArgument(idx int) []byte { // up to 16 params
    // TODO: corner case 0, corner case >=16
    code := []byte{
	    // TODO
    }
    return code
}


func jitStackFrame(size uint8) []byte {
	return []byte{
	    // TODO
	}
}

/* TODO: peephole optimizer:
 - remove argument checks (test rbx,rbx 48 85 db 76 xx)
 - shorten immediate values
 - constant-fold operations 
 - inline functions
 - jump to other functions
*/
