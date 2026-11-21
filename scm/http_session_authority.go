/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

// Credential fingerprints are process-private routing keys, never credentials
// or authenticated identities. Frontends additionally bind the validated
// principal incarnation before using a persistent session's capabilities.
var httpSessionCredentialSecret = func() []byte {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		panic(err)
	}
	return secret
}()

func httpCredentialSessionKey(clientID, username, password, authorization string) string {
	mac := hmac.New(sha256.New, httpSessionCredentialSecret)
	mac.Write([]byte(strconv.Itoa(len(username)) + ":" + username + strconv.Itoa(len(password)) + ":" + password + authorization))
	return clientID + "\x00" + hex.EncodeToString(mac.Sum(nil))
}

type sessionPrincipal struct {
	namespace string
	id        string
}

func initHTTPSessionAuthority() {
	Declare(&Globalenv, &Declaration{
		Name: "bind_session_principal",
		Fn: func(args ...Scmer) Scmer {
			state, ok := args[0].Any().(*SessionState)
			if !ok || state == nil {
				panic("expected native session state")
			}
			if !args[1].IsString() || !args[2].IsString() || args[1].String() == "" || args[2].String() == "" {
				panic("session principal requires a nonempty authority namespace and incarnation")
			}
			principal := &sessionPrincipal{namespace: args[1].String(), id: args[2].String()}
			if state.principal.CompareAndSwap(nil, principal) {
				return NewBool(true)
			}
			bound := state.principal.Load()
			if bound.namespace != principal.namespace || bound.id != principal.id {
				panic("session authority changed; use a new session identifier")
			}
			return NewBool(true)
		},
		Type: &TypeDescriptor{Kind: "func", HasSideEffects: true,
			Description: "immutably binds native session capabilities to a validated principal incarnation; authenticate before calling",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "state"},
				{Kind: "string", Label: "authority_namespace"},
				{Kind: "string", Label: "principal_incarnation"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
		},
	})
}
