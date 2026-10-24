/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify it under
the terms of the GNU General Public License as published by the Free Software
Foundation, either version 3 of the License, or (at your option) any later version.
This program is distributed WITHOUT ANY WARRANTY; without even the implied
warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See
<https://www.gnu.org/licenses/> for a copy of the GNU General Public License.
*/

package storage

import "fmt"
import "time"
import "bytes"
import "context"
import "crypto/sha1"
import "crypto/subtle"
import "encoding/json"
import "encoding/base64"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

type replicaRequest struct {
	SQL        string `json:"sql,omitempty"`
	PostgreSQL bool   `json:"postgresql,omitempty"`
	Epoch      uint64 `json:"epoch,omitempty"`
	Database   string `json:"database"`
	User       string `json:"user,omitempty"`
	Hash       string `json:"hash,omitempty"`
	Read       bool   `json:"read,omitempty"`
	State      string `json:"state,omitempty"`
	Salt       []byte `json:"salt,omitempty"`
	Scramble   []byte `json:"scramble,omitempty"`
}

type replicaAuth struct {
	Hash      string          `json:"hash"`
	Admin     bool            `json:"admin"`
	Databases map[string]bool `json:"databases"`
}

func (r *replicationManager) authenticate(user, hash string, salt, scramble []byte, requested string) (auth replicaAuth, err error) {
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("authentication failed")
		}
	}()
	password := scm.Apply(r.password, scm.NewString(user))
	if password.IsNil() || !replicationPasswordProof([]byte(password.String()), hash, salt, scramble) {
		return auth, fmt.Errorf("authentication failed")
	}
	auth = replicaAuth{Hash: base64.StdEncoding.EncodeToString([]byte(password.String())), Databases: make(map[string]bool)}
	policy := scm.Apply(r.policy, scm.NewString(user))
	auth.Admin = func() (allowed bool) {
		defer func() { _ = recover() }()
		return scm.Apply(policy, scm.NewString("system"), scm.NewBool(true), scm.NewBool(true)).Bool()
	}()
	if auth.Admin {
		return auth, nil
	}
	if requested != "" {
		if scm.Apply(policy, scm.NewString(requested), scm.NewString("*"), scm.NewBool(true)).Bool() {
			auth.Databases[requested] = true
		}
		return auth, nil
	}
	// Query just this user's grants, not every tenant in the catalog.
	access := GetDatabase("system").GetTable("access")
	schema := scm.NewSlice([]scm.Scmer{
		newScanAccessHeader(1, scanAccessConsumerScan, 0, -1),
		newScanBoundarySpec("username", EqualMatcher, 0, 0, true, true, "", false, -1, nil, nil, "", false),
	})
	collect := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		return scm.NewSlice(append(append([]scm.Scmer(nil), a[0].Slice()...), a[1]))
	})
	combine := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		return scm.NewSlice(append(append([]scm.Scmer(nil), a[0].Slice()...), a[1].Slice()...))
	})
	rows := access.scan(nil, schema, []scm.Scmer{scm.NewString(user)}, nil, scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewBool(true) }), []string{"database"}, collect, scm.NewSlice(nil), combine, false)
	for _, name := range rows.Slice() {
		auth.Databases[name.String()] = true
	}
	return auth, nil
}

func (r *replicationManager) handleMaster(ctx context.Context, id, method string, data json.RawMessage) (any, error) {
	select {
	case <-r.done:
		return nil, fmt.Errorf(mosi.NetworkError)
	default:
	}
	var req replicaRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	r.mu.Lock()
	retired := r.journal.Retired[id]
	r.mu.Unlock()
	if retired {
		return nil, fmt.Errorf("replica retired by administrator")
	}
	auth, err := r.authenticate(req.User, req.Hash, req.Salt, req.Scramble, req.Database)
	if err != nil {
		return nil, err
	}
	if method == "login" {
		return auth, nil
	}
	if method == "admin" {
		if !auth.Admin {
			return nil, fmt.Errorf("administrator access required")
		}
		return r.executeAdmin(req)
	}
	if !auth.Admin && !auth.Databases[req.Database] {
		return nil, fmt.Errorf("access denied for database %s", req.Database)
	}
	return nil, fmt.Errorf("unknown authentication method")
}

func (r *replicationManager) handleSlave(ctx context.Context, method string, data json.RawMessage) (any, error) {
	if method == "retire" {
		r.mu.Lock()
		r.journal.Retired[r.id] = true
		err := r.saveLocked()
		r.mu.Unlock()
		if err != nil {
			return nil, err
		}
		r.retired.Store(true)
		if r.shutdown != nil {
			time.AfterFunc(50*time.Millisecond, r.shutdown)
		}
		return true, nil
	}
	return nil, fmt.Errorf("unknown administrative callback")
}

// replicationHTTPAuthentication forwards an online login to the master before
// the ordinary frontend sees it. Its cache is private to this installation;
// the master never trusts locally edited access rows.
func replicationHTTPAuthentication(user, password string) bool {
	if r := replication.Load(); r == nil || r.master {
		return true
	}
	hash := base64.StdEncoding.EncodeToString([]byte(scm.MySQLPassword(scm.NewString(password)).String()))
	return replicationAuthentication(replicaRequest{User: user, Hash: hash})
}

func replicationMySQLAuthentication(user string, salt, scramble []byte) bool {
	return replicationAuthentication(replicaRequest{User: user, Salt: salt, Scramble: scramble})
}

func replicationPasswordProof(raw []byte, hash string, salt, scramble []byte) bool {
	if hash != "" {
		return subtle.ConstantTimeCompare([]byte(base64.StdEncoding.EncodeToString(raw)), []byte(hash)) == 1
	}
	if len(raw) != sha1.Size || len(scramble) != sha1.Size || len(salt) != 20 {
		return false
	}
	second := sha1.Sum(raw)
	challenge := sha1.New()
	challenge.Write(salt)
	challenge.Write(second[:])
	expected := challenge.Sum(nil)
	for i := range expected {
		expected[i] ^= raw[i]
	}
	return subtle.ConstantTimeCompare(expected, scramble) == 1
}

func replicationAuthentication(proof replicaRequest) bool {
	r := replication.Load()
	if r == nil || r.master {
		return true
	}
	var auth replicaAuth
	if peer := r.peer(""); peer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := peer.call(ctx, "login", proof, &auth)
		cancel()
		if err != nil {
			if _, denied := err.(mosiRemoteError); denied {
				return false
			}
			r.mu.Lock()
			auth = r.journal.Users[proof.User]
			r.mu.Unlock()
		} else {
			r.mu.Lock()
			previous, exists := r.journal.Users[proof.User]
			r.journal.Users[proof.User] = auth
			old, _ := json.Marshal(previous)
			current, _ := json.Marshal(auth)
			if !exists || !bytes.Equal(old, current) {
				err = r.saveLocked()
			}
			r.mu.Unlock()
			if err != nil {
				return false
			}
		}
	} else {
		r.mu.Lock()
		auth = r.journal.Users[proof.User]
		r.mu.Unlock()
	}
	rawHash, err := base64.StdEncoding.DecodeString(auth.Hash)
	if err != nil || !replicationPasswordProof(rawHash, proof.Hash, proof.Salt, proof.Scramble) {
		return false
	}
	r.authMu.Lock()
	defer r.authMu.Unlock()
	encoded, _ := json.Marshal(auth)
	if r.authInstalled == nil {
		r.authInstalled = make(map[string]string)
	}
	if r.authInstalled[proof.User] == string(encoded) {
		return true
	}
	// Publish only this proven user's credentials through the existing native
	// table APIs, so SQL/HTTP/MySQL use the ordinary authentication callbacks.
	db := GetDatabase("system")
	users := db.GetTable("user")
	row := []scm.Scmer{scm.NewString(proof.User), scm.NewString(string(rawHash)), scm.NewBool(auth.Admin)}
	update := scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		return scm.Apply(a[0], scm.NewSlice([]scm.Scmer{scm.NewString("password"), row[1], scm.NewString("admin"), row[2]}))
	})
	users.Insert([]string{"username", "password", "admin"}, [][]scm.Scmer{row}, []string{"$update"}, update, false, nil)
	access := db.GetTable("access")
	for name := range auth.Databases {
		access.Insert([]string{"username", "database"}, [][]scm.Scmer{{scm.NewString(proof.User), scm.NewString(name)}}, nil, scm.NewFunc(func(a ...scm.Scmer) scm.Scmer { return scm.NewInt(0) }), false, nil)
	}
	r.authInstalled[proof.User] = string(encoded)
	return true
}
