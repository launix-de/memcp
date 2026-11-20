/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import "hash"
import "crypto/md5"
import "crypto/sha1"
import "crypto/sha256"
import "crypto/sha512"
import "golang.org/x/crypto/md4"

func initHashDigest() {
	Declare(&Globalenv, &Declaration{
		Name: "hash_digest",
		Fn: func(args ...Scmer) Scmer {
			if args[0].IsNil() || args[1].IsNil() {
				return NewNil()
			}
			var digest hash.Hash
			switch args[1].String() {
			case "md4":
				digest = md4.New()
			case "md5":
				digest = md5.New()
			case "sha1":
				digest = sha1.New()
			case "sha256":
				digest = sha256.New()
			case "sha512":
				digest = sha512.New()
			default:
				panic("unsupported hash algorithm: " + args[1].String())
			}
			digest.Write([]byte(args[0].String()))
			return NewString(string(digest.Sum(nil)))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "computes a raw byte digest using md4, md5, sha1, sha256 or sha512",
			Params: []*TypeDescriptor{{Kind: "string", Label: "bytes"}, {Kind: "string", Label: "algorithm"}},
			Return: &TypeDescriptor{Kind: "string"}, Const: true,
			JITEmit: func(ctx *JITContext, source []Scmer, args []JITValueDesc, output JITValueDesc) JITValueDesc {
				ctx.Coverage.NativeCalls++
				return jitEmitGeneratedCallBoundary(ctx, declarations["hash_digest"], source, args, output)
			},
			JITVirtualArgs: true, JITInlineCost: 65535},
	})
}
