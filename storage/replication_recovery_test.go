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

import "os"
import "sync"
import "time"
import "testing"
import "context"
import "encoding/json"
import "path/filepath"
import "github.com/google/uuid"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/memcp/storage/mosi"

func replicaRecoveryFixture() *replicationManager {
	return &replicationManager{
		master: true, id: uuid.NewString(), boot: uuid.NewString(), cells: make(map[string]*replicaCell),
		directory: replicaDirectory{Records: make(map[string]replicaRecord), Users: make(map[string]replicaAuth), Retired: make(map[string]bool), Initialized: make(map[string]bool)},
		backups:   make(map[string]replicaBackup), restoring: make(map[string]bool), confirmed: make(map[string]string), peersBoot: make(map[string]string),
	}
}

func TestReplicationRightsNeverWritePersistence(t *testing.T) {
	old := Basepath
	Basepath = filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(Basepath, []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { Basepath = old })
	r := replicaRecoveryFixture()
	resource := mosiResource{Database: "tenant"}
	cell := r.obtain(resource, mosi.M, "")
	if cell.rights.Phase() != mosi.M {
		t.Fatal("local permission was not published")
	}
	r.localRecord(resource, mosi.IO, "")
	cell.rights.Initialize(mosi.IO)
	pending, _ := cell.rights.Request(mosi.M)
	if err := r.acceptGrant(mosiPacket{ID: pending.Flight().ID(), Resource: resource, State: "M", Owner: r.id, Epoch: 2}); err != nil {
		t.Fatal(err)
	}
	<-pending.Flight().Done()
	if cell.rights.Phase() != mosi.M {
		t.Fatal("RAM grant did not complete")
	}
	data, err := os.ReadFile(Basepath)
	if err != nil || string(data) != "unchanged" {
		t.Fatal("rights acquisition touched persistent storage", err)
	}
}

func TestReplicationRecoveryNeedsLiveAuthority(t *testing.T) {
	r := replicaRecoveryFixture()
	resource := mosiResource{Database: "tenant"}
	r.restoring[resource.Database] = true
	r.backups[resource.key()] = replicaBackup{Resource: resource}
	if r.resourceKnown(resource) {
		t.Fatal("backup established local authority")
	}
	peer := uuid.NewString()
	request := replicaRequest{Boot: uuid.NewString(), Resources: []replicaRecord{{Resource: resource, State: "M", Owner: peer, Epoch: 8, InitialClaimed: true}}}
	if _, err := r.recoverPeer(context.Background(), peer, request, replicaAuth{Databases: map[string]bool{}}); err == nil {
		t.Fatal("unauthorized database report was accepted")
	}
	if r.resourceKnown(resource) {
		t.Fatal("denied report established authority")
	}
	reply, err := r.recoverPeer(context.Background(), peer, request, replicaAuth{Databases: map[string]bool{"tenant": true}})
	if err != nil || len(reply.Resources) != 1 || reply.Resources[0].State != "M" {
		t.Fatal("live authority did not reconstruct directory", err)
	}
	record := r.directory.Records[resource.key()]
	if record.Owner != peer || record.State != "IO" || r.resourceCell(resource).rights.Phase() != mosi.IO {
		t.Fatal("master used a remote owner's backup")
	}
	if !r.resourceKnown(resource) {
		t.Fatal("reported modifier cannot be routed")
	}
	// Losing the modifier never makes the master's backend authoritative.
	failure := func() (failure any) {
		defer func() { failure = recover() }()
		r.obtain(resource, mosi.S, "")
		return nil
	}()
	if failure != mosi.NetworkError {
		t.Fatalf("unreachable modifier did not fail: %v", failure)
	}
	if r.directory.Records[resource.key()].Owner != peer || r.resourceCell(resource).rights.Phase() != mosi.IO {
		t.Fatal("offline modifier was reclaimed")
	}
}

func TestReplicationRestartRecoversRightsAndRequestDomain(t *testing.T) {
	r := replicaRecoveryFixture()
	peer := uuid.NewString()
	resource := mosiResource{Database: "tenant"}
	oldBoot := uuid.NewString()
	r.peersBoot[peer] = oldBoot
	r.initialClaims = map[string]map[string]bool{peer: {resource.key(): true}}
	for _, item := range []mosiResource{resource, {Database: "another_tenant"}} {
		r.directory.Records[item.key()] = replicaRecord{Resource: item, Owner: peer, Epoch: 7, State: "IO", InitialClaimed: true, Requests: map[string]replicaRequestMark{peer: {Sequence: 900, Write: true}}}
	}
	reply, err := r.recoverPeer(context.Background(), peer, replicaRequest{Boot: uuid.NewString(), Resources: []replicaRecord{{Resource: resource, Generation: "local-wal-generation"}}}, replicaAuth{Databases: map[string]bool{"tenant": true}})
	if err != nil || len(reply.Resources) != 1 {
		t.Fatal("slave restart handshake failed", err)
	}
	recovered := reply.Resources[0]
	if recovered.Owner != peer || recovered.State != "M" || recovered.Epoch != 7 || recovered.Generation != "local-wal-generation" || !recovered.InitialClaimed {
		t.Fatalf("bad restart rights: %+v", recovered)
	}
	for _, record := range r.directory.Records {
		if _, present := record.Requests[peer]; present {
			t.Fatal("new process retained old request sequence marks")
		}
	}
}

func TestReplicationRecoveryRetainsKnownReaders(t *testing.T) {
	r := replicaRecoveryFixture()
	resource := mosiResource{Database: "tenant"}
	owner, reader := uuid.NewString(), uuid.NewString()
	r.restoring[resource.Database] = true
	r.backups[resource.key()] = replicaBackup{Resource: resource}
	report := replicaRecord{Resource: resource, Owner: owner, State: "O", Epoch: 9, Readers: map[string]bool{reader: true}}
	reply, err := r.recoverPeer(context.Background(), owner, replicaRequest{Boot: uuid.NewString(), Resources: []replicaRecord{report}}, replicaAuth{Admin: true})
	if err != nil || reply.Resources[0].State != "O" || !r.directory.Records[resource.key()].Readers[reader] {
		t.Fatal("recovery silently removed a reader", err)
	}
}

func TestReplicationGrantedReadsIgnoreDirectoryLocks(t *testing.T) {
	r := replicaRecoveryFixture()
	resource := mosiResource{Database: "tenant"}
	cell := r.resourceCell(resource)
	cell.rights.Initialize(mosi.S)
	// The complete manager/control plane can be busy without a RAM write or lock
	// in the already authorized schema path.
	r.mu.Lock()
	var workers sync.WaitGroup
	done := make(chan struct{})
	workers.Add(1)
	go func() {
		defer workers.Done()
		for i := 0; i < 10000; i++ {
			r.obtain(resource, mosi.S, "")
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		r.mu.Unlock()
		workers.Wait()
		t.Fatal("authorized reads waited for control-plane ownership")
	}
	r.mu.Unlock()
	workers.Wait()
}

func TestReplicationConfigUsesNativePersistenceWithoutRights(t *testing.T) {
	source, engine := createDurabilityTestTable(t, "treplication_native_config", 0)
	config, _ := CreateTable(source.schema.Name, "config", Safe, false)
	config.CreateColumn("node", "text", nil, nil)
	config.CreateColumn("key", "text", nil, nil)
	config.CreateColumn("value", "text", nil, nil)
	createTableKey(config, uniqueKey{Id: "PRIMARY", Cols: []string{"node", "key"}}, nil)
	r := replicaRecoveryFixture()
	r.config = config
	auth := replicaAuth{Hash: "cached-proof", Databases: map[string]bool{"tenant": true}}
	if err := r.storeConfig(replicaConfigEntry{Kind: "users", Name: "alice", Auth: &auth}); err != nil {
		t.Fatal(err)
	}
	if err := r.storeConfig(replicaConfigEntry{Kind: "retired", Name: "retired-peer", Value: true}); err != nil {
		t.Fatal(err)
	}
	resource := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
	generation := uuid.NewString()
	if err := r.saveBackup(resource, generation); err != nil {
		t.Fatal(err)
	}
	r.localRecord(resource, mosi.M, generation)
	// Another node may share system's backend, but must not share our cache or
	// the last complete physical generation of its independent safety copy.
	other := replicaRecoveryFixture()
	other.config = config
	otherAuth := replicaAuth{Hash: "other-node-proof", Admin: true}
	if err := other.storeConfig(replicaConfigEntry{Kind: "users", Name: "alice", Auth: &otherAuth}); err != nil {
		t.Fatal(err)
	}
	if err := other.saveBackup(resource, uuid.NewString()); err != nil {
		t.Fatal(err)
	}
	// Reopen through the ordinary schema/WAL loader, without a replication file.
	db := newDatabase()
	db.Name, db.persistence, db.srState = source.schema.Name, engine, COLD
	db.ensureLoaded()
	defer db.closeTransactionLog()
	restarted := replicaRecoveryFixture()
	restarted.id = r.id
	if err := restarted.loadConfig(db.tables.Get("config")); err != nil {
		t.Fatal(err)
	}
	if restarted.directory.Users["alice"].Hash != "cached-proof" || !restarted.directory.Retired["retired-peer"] {
		t.Fatal("native config did not survive restart")
	}
	if restarted.backups[resource.key()].Generation != generation {
		t.Fatal("completed backup placement did not survive restart")
	}
	if len(restarted.directory.Records) != 0 || restarted.resourceKnown(resource) {
		t.Fatal("persistent data reconstructed MOSI rights")
	}
	if files, err := filepath.Glob(filepath.Join(Basepath, ".replication-state*")); err != nil || len(files) != 0 {
		t.Fatal("native persistence created a private replication journal", files, err)
	}
}

func TestReplicationRecoveryInterruptedHandover(t *testing.T) {
	for _, complete := range []bool{false, true} {
		t.Run(map[bool]string{false: "incomplete-data", true: "complete-data"}[complete], func(t *testing.T) {
			r := replicaRecoveryFixture()
			source := uuid.NewString()
			resource := mosiResource{Database: "tenant"}
			r.restoring[resource.Database] = true
			generation := "prepared-schema-content-hash"
			backupGeneration := "old-schema-content-hash"
			if complete {
				backupGeneration = generation
			}
			r.backups[resource.key()] = replicaBackup{Resource: resource, Generation: backupGeneration}
			proof := replicaRecord{Resource: resource, Owner: source, State: "IO", Handoff: r.id, Epoch: 7, Generation: generation}
			_, err := r.recoverPeer(context.Background(), source, replicaRequest{Boot: uuid.NewString(), Resources: []replicaRecord{proof}}, replicaAuth{Admin: true})
			if err != nil {
				t.Fatal(err)
			}
			record := r.directory.Records[resource.key()]
			wantOwner := source
			if complete {
				wantOwner = r.id
			}
			if record.Owner != wantOwner || !r.resourceKnown(resource) {
				t.Fatal("interrupted handover has no safe data custodian", record)
			}
			if !complete && r.resourceCell(resource).rights.Phase() != mosi.IO {
				t.Fatal("partial master data gained authority")
			}
		})
	}
}

func TestReplicationRecoveryWaitsForHandoverRecipient(t *testing.T) {
	r := replicaRecoveryFixture()
	source, target := uuid.NewString(), uuid.NewString()
	resource := mosiResource{Database: "tenant"}
	r.restoring[resource.Database] = true
	r.backups[resource.key()] = replicaBackup{Resource: resource}
	proof := replicaRecord{Resource: resource, Owner: source, State: "IO", Handoff: target, Epoch: 7, Generation: "prepared-content"}
	if _, err := r.recoverPeer(context.Background(), source, replicaRequest{Boot: uuid.NewString(), Resources: []replicaRecord{proof}}, replicaAuth{Admin: true}); err != nil {
		t.Fatal(err)
	}
	if r.resourceKnown(resource) {
		t.Fatal("unreported recipient's possible M was reclaimed")
	}
	// The still-running recipient confirms that no M grant was installed.
	recipient := replicaRecord{Resource: resource, Owner: source, State: "IO", Epoch: 7}
	if _, err := r.recoverPeer(context.Background(), target, replicaRequest{Boot: uuid.NewString(), Resources: []replicaRecord{recipient}}, replicaAuth{Admin: true}); err != nil {
		t.Fatal(err)
	}
	if !r.resourceKnown(resource) || r.directory.Records[resource.key()].Owner != source {
		t.Fatal("confirmed ungranted transfer lost its original data custodian")
	}
}

func TestReplicationStartupRejectsUserDataBeforeManagerPublication(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "treplication_startup_gate", 1)
	old, starting := replication.Swap(nil), replicationStarting.Swap(true)
	defer func() { replication.Store(old); replicationStarting.Store(starting) }()
	if replicationHTTPAuthentication("root", "admin") || replicationAuthentication(replicaRequest{User: "root"}) {
		t.Fatal("startup accepted local authentication before replication")
	}
	calls := []func(){
		func() { table.schema.GetRead(nil).Release() },
		func() { table.schema.GetWrite(nil).Release() },
		func() { table.ActiveShards()[0].GetRead(nil).Release() },
		func() { table.ActiveShards()[0].GetWrite(nil).Release() },
	}
	for _, call := range calls {
		failure := func() (failure any) { defer func() { failure = recover() }(); call(); return nil }()
		if failure != mosi.NetworkError {
			t.Fatalf("startup exposed data before coherence initialization: %v", failure)
		}
	}
}

func TestReplicationRecoveredSharedBindsCompletedGeneration(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "treplication_recovered_generation", 3)
	table.schema.rebuild(true, false, false, table)
	completed := table.ActiveShards()[0]
	resource := mosiResource{Database: table.schema.Name, Table: table.Name, Shard: completed.logicalID().String()}
	r := replicaRecoveryFixture()
	r.restoring[resource.Database] = true
	r.backups[resource.key()] = replicaBackup{Resource: resource, Generation: completed.uuid.String()}
	// A schema packet reached storage, but its new physical data UUID did not.
	placeholder := new(storageShard)
	encoded := []byte(`{"uuid":"` + uuid.NewString() + `","mosi":"` + resource.Shard + `"}`)
	if err := json.Unmarshal(encoded, placeholder); err != nil {
		t.Fatal(err)
	}
	placeholder.t = table
	table.mu.Lock()
	table.Shards = []*storageShard{placeholder}
	table.publishTopologyLocked()
	table.mu.Unlock()
	table.schema.save()
	owner := uuid.NewString()
	report := replicaRecord{Resource: resource, Owner: owner, State: "O", Epoch: 7, Generation: placeholder.uuid.String(), Readers: map[string]bool{r.id: true}}
	if _, err := r.recoverPeer(context.Background(), owner, replicaRequest{Boot: uuid.NewString(), Resources: []replicaRecord{report}}, replicaAuth{Admin: true}); err != nil {
		t.Fatal(err)
	}
	cell := r.resourceCell(resource)
	if cell.rights.Phase() != mosi.S || cell.current.Load() == nil || cell.current.Load().uuid != completed.uuid {
		t.Fatal("recovery published S for an incomplete advertised generation")
	}
	old := replication.Swap(r)
	defer replication.Store(old)
	access := placeholder.GetRead(nil)
	defer access.Release()
	if access.shard.main_count != 3 {
		t.Fatalf("recovered backup lost rows: %d", access.shard.main_count)
	}
}

func TestReplicationIdentityUsesLocalSettingsCheckpoint(t *testing.T) {
	oldBase := Basepath
	settingsMu.Lock()
	oldID := Settings.ReplicationUUID
	Settings.ReplicationUUID = uuid.NewString()
	identity := Settings.ReplicationUUID
	settingsMu.Unlock()
	Basepath = t.TempDir()
	defer func() { Basepath = oldBase; settingsMu.Lock(); Settings.ReplicationUUID = oldID; settingsMu.Unlock() }()
	if err := saveSettings(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(Basepath, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var stored struct{ ReplicationUUID string }
	if err := json.Unmarshal(data, &stored); err != nil || stored.ReplicationUUID != identity {
		t.Fatal("node identity was not durably saved in local settings", err)
	}
	if files, err := filepath.Glob(filepath.Join(Basepath, ".settings-tmp-*")); err != nil || len(files) != 0 {
		t.Fatal("settings checkpoint left a live temporary file", files, err)
	}
}

func TestReplicationRuntimeSchemaCreatorConfirmsNewShard(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "treplication_runtime_creator", 1)
	r := replicaRecoveryFixture()
	creator := uuid.NewString()
	shard := table.ActiveShards()[0]
	cell := r.declareShard(shard, creator, false)
	resource := *cell.resource.Load()
	if err := r.saveBackup(resource, shard.uuid.String()); err != nil {
		t.Fatal(err)
	}
	if !r.resourceKnown(resource) || r.directory.Records[resource.key()].Owner != creator || cell.rights.Phase() != mosi.IO {
		t.Fatal("a running master's authenticated schema creator was mistaken for unknown restart authority")
	}
}

func TestReplicationRecoveryKeepsMasterWALAfterSurrender(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "treplication_master_wal", 3)
	table.schema.rebuild(true, false, false, table)
	original := table.ActiveShards()[0]
	table.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(99), scm.NewString("committed-by-master")}}, nil, scm.NewNil(), false, nil)
	r := replicaRecoveryFixture()
	source := uuid.NewString()
	resource := mosiResource{Database: table.schema.Name, Table: table.Name, Shard: original.logicalID().String()}
	r.restoring[resource.Database] = true
	report := replicaRecord{Resource: resource, State: "IO", Owner: source, Handoff: r.id, Epoch: 7, Generation: original.uuid.String()}
	if _, err := r.recoverPeer(context.Background(), source, replicaRequest{Boot: uuid.NewString(), Resources: []replicaRecord{report}}, replicaAuth{Admin: true}); err != nil {
		t.Fatal(err)
	}
	if r.directory.Records[resource.key()].Owner != r.id {
		t.Fatal("old slave I copy replaced the master's newer committed data")
	}
	old := replication.Swap(r)
	defer replication.Store(old)
	access := table.ActiveShards()[0].GetRead(nil)
	defer access.Release()
	if access.shard.Count() != 4 {
		t.Fatalf("master WAL was lost during recovery: rows=%d", access.shard.Count())
	}
}

func TestReplicationSchemaTransferPreservesLocalPhysicalGeneration(t *testing.T) {
	table, engine := createDurabilityTestTable(t, "treplication_local_physical", 3)
	r := replicaRecoveryFixture()
	old := replication.Swap(r)
	defer replication.Store(old)
	r.resourceCell(mosiResource{Database: table.schema.Name}).rights.Initialize(mosi.M)
	r.schemaShards(table.schema, r.id, true)
	table.schema.rebuild(true, false, false, table)
	table.schema.save()
	remoteSchema := append([]byte(nil), engine.ReadSchema()...)
	table.Insert([]string{"id", "payload"}, [][]scm.Scmer{{scm.NewInt(99), scm.NewString("newer-master-main")}}, nil, scm.NewNil(), false, nil)
	table.schema.rebuild(true, false, false, table)
	local := table.ActiveShards()[0]
	if err := r.installSchema(uuid.NewString(), table.schema.Name, engine, remoteSchema); err != nil {
		t.Fatal(err)
	}
	loaded := newDatabase()
	if err := json.Unmarshal(engine.ReadSchema(), loaded); err != nil {
		t.Fatal(err)
	}
	saved := loaded.tables.Get(table.Name).ActiveShards()[0]
	if saved.uuid != local.uuid {
		t.Fatal("remote schema redirected storage away from the local current generation/WAL")
	}
	remoteGeneration, err := replicaSchemaGeneration(remoteSchema)
	if err != nil {
		t.Fatal(err)
	}
	localGeneration, err := replicaSchemaGeneration(engine.ReadSchema())
	if err != nil || localGeneration != remoteGeneration {
		t.Fatal("local physical UUID mapping changed semantic schema identity", err)
	}
}

func TestReplicationDelayedInvalidationPreservesNewGrant(t *testing.T) {
	r := replicaRecoveryFixture()
	r.master = false
	r.upstream = &mosiPeer{outgoing: make(chan mosiMessage, 8), done: make(chan struct{})}
	resource := mosiResource{Database: "tenant"}
	c := r.resourceCell(resource)
	c.rights.Initialize(mosi.M)
	r.directory.Records[resource.key()] = replicaRecord{Resource: resource, State: "M", Owner: r.id, Epoch: 9}
	if err := r.invalidateResource(context.Background(), "", mosiPacket{Kind: "invalidate", ID: "old-revocation", Resource: resource, Epoch: 8}); err != nil {
		t.Fatal(err)
	}
	if c.rights.Phase() != mosi.M || r.directory.Records[resource.key()].State != "M" {
		t.Fatal("old invalidation revoked newer M")
	}
}

func TestReplicationDuplicateWriteDoesNotReacquireOwnership(t *testing.T) {
	r := replicaRecoveryFixture()
	resource := mosiResource{Database: "tenant"}
	c := r.resourceCell(resource)
	c.rights.Initialize(mosi.IO)
	pending, _ := c.rights.Request(mosi.M)
	r.directory.Records[resource.key()] = replicaRecord{Resource: resource, State: "IO", Owner: r.id, Epoch: 5}
	packet := mosiPacket{Kind: "write", ID: pending.Flight().ID(), Resource: resource, Sequence: 1}
	if err := r.coordinate(context.Background(), r.id, packet); err != nil {
		t.Fatal(err)
	}
	epoch := r.directory.Records[resource.key()].Epoch
	if err := r.coordinate(context.Background(), r.id, packet); err != nil {
		t.Fatal(err)
	}
	if r.directory.Records[resource.key()].Epoch != epoch {
		t.Fatal("duplicate write executed a second ownership transition")
	}
	// Even after a later delegation, replaying the completed request must not
	// recall that owner's data or regain rights on behalf of the old requester.
	r.updateRecord(resource, func(record *replicaRecord) bool { record.Owner = "another-node"; record.Epoch++; return true })
	if err := r.coordinate(context.Background(), r.id, packet); err != nil {
		t.Fatal("completed replay attempted a new recall", err)
	}
}

func TestReplicationAcknowledgementMatchesHandover(t *testing.T) {
	resource := mosiResource{Database: "tenant"}
	for _, kind := range []string{"ack", "failure", "begin"} {
		for _, mismatch := range []string{"peer", "resource", "epoch", "generation"} {
			t.Run(kind+"/"+mismatch, func(t *testing.T) {
				r := replicaRecoveryFixture()
				source := "slave"
				peer := &mosiPeer{outgoing: make(chan mosiMessage, 8), done: make(chan struct{})}
				r.peers = map[string]*mosiPeer{source: peer, "other": peer}
				step := &replicaStep{peer: source, resource: resource, epoch: 9, generation: "current", done: make(chan struct{})}
				r.steps.Store("handover", step)
				packet := mosiPacket{Kind: kind, ID: "handover", Resource: resource, Epoch: 9, Generation: "current"}
				switch mismatch {
				case "peer":
					source = "other"
				case "resource":
					packet.Resource.Database = "other"
				case "epoch":
					packet.Epoch--
				case "generation":
					packet.Generation = "old"
				}
				if err := r.receivePacket(source, &packet); err == nil {
					t.Fatal("mismatched operation accepted")
				}
				select {
				case <-step.done:
					t.Fatal("mismatched packet completed handover")
				default:
				}
				if kind != "begin" {
					packet = mosiPacket{Kind: kind, ID: "handover", Resource: resource, Epoch: 9, Generation: "current"}
					if err := r.receivePacket("slave", &packet); err != nil {
						t.Fatal(err)
					}
					select {
					case <-step.done:
					default:
						t.Fatal("matching reply did not finish handover")
					}
				}
			})
		}
	}
}

func TestReplicationStaleGrantFailsPendingFlight(t *testing.T) {
	r := replicaRecoveryFixture()
	resource := mosiResource{Database: "tenant"}
	c := r.resourceCell(resource)
	c.rights.Initialize(mosi.IO)
	pending, _ := c.rights.Request(mosi.M)
	r.directory.Records[resource.key()] = replicaRecord{Resource: resource, Owner: "other", Epoch: 9, State: "IO"}
	err := r.acceptGrant(mosiPacket{ID: pending.Flight().ID(), Resource: resource, Epoch: 8, State: "M", Owner: r.id})
	if err == nil || c.rights.Phase() != mosi.IO || r.directory.Records[resource.key()].Epoch != 9 {
		t.Fatal("stale grant published authority", err)
	}
	select {
	case <-pending.Flight().Done():
		if pending.Flight().Err() == nil {
			t.Fatal("stale grant succeeded")
		}
	default:
		t.Fatal("stale grant stranded waiters")
	}
}

func TestReplicationCompletedReadStillAllowsWriteUpgrade(t *testing.T) {
	r := replicaRecoveryFixture()
	resource := mosiResource{Database: "tenant"}
	c := r.resourceCell(resource)
	c.rights.Initialize(mosi.IO)
	read, _ := c.rights.Request(mosi.S)
	c.rights.Request(mosi.M) // upgrade wins before the read grant arrives
	r.directory.Records[resource.key()] = replicaRecord{Resource: resource, Owner: r.id, Epoch: 5, State: "IO"}
	packet := mosiPacket{Kind: "read", ID: read.Flight().ID(), Resource: resource, Sequence: 1}
	if err := r.coordinate(context.Background(), r.id, packet); err != nil {
		t.Fatal(err)
	}
	if c.rights.Phase() != mosi.MR {
		t.Fatal("read reply bypassed pending write upgrade")
	}
	packet.Kind = "write"
	var workers sync.WaitGroup
	for range 32 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if err := r.coordinate(context.Background(), r.id, packet); err != nil {
				t.Error(err)
			}
		}()
	}
	workers.Wait()
	if c.rights.Phase() != mosi.M || r.directory.Records[resource.key()].Epoch != 6 {
		t.Fatal("write upgrade lost or executed repeatedly")
	}
}

func TestReplicationStaleTransferDoesNotOpenBackend(t *testing.T) {
	r := replicaRecoveryFixture()
	r.master = false
	r.upstream = &mosiPeer{outgoing: make(chan mosiMessage, 8), done: make(chan struct{})}
	resource := mosiResource{Database: "tenant"}
	r.directory.Records[resource.key()] = replicaRecord{Resource: resource, Epoch: 9}
	if err := r.receivePacket("", &mosiPacket{Kind: "begin", ID: "old", Resource: resource, Epoch: 8}); err == nil {
		t.Fatal("stale transfer opened backend")
	}
	if _, exists := r.imports.Load("/old"); exists {
		t.Fatal("stale transfer registered an import")
	}
	// A transfer which began before a newer grant must also be rejected at end,
	// before even attempting to publish this deliberately invalid schema payload.
	transfer := &replicaImport{header: mosiPacket{ID: "old", Resource: resource, Epoch: 8}}
	r.imports.Store("/old", transfer)
	err := r.receivePacket("", &mosiPacket{Kind: "end", ID: "old", Resource: resource, Epoch: 8})
	if err == nil || err.Error() != "stale transfer epoch" {
		t.Fatal("staged transfer published after newer epoch", err)
	}
	if _, exists := r.imports.Load("/old"); exists {
		t.Fatal("rejected transfer retained GC registration")
	}
}

func TestReplicationReadFailureDoesNotCancelWriteUpgrade(t *testing.T) {
	r := replicaRecoveryFixture()
	r.master = false
	r.upstream = &mosiPeer{outgoing: make(chan mosiMessage, 8), done: make(chan struct{})}
	resource := mosiResource{Database: "tenant"}
	c := r.resourceCell(resource)
	c.rights.Initialize(mosi.IO)
	pending, _ := c.rights.Request(mosi.S)
	c.rights.Request(mosi.M)
	packet := mosiPacket{Kind: "failure", ID: pending.Flight().ID(), Resource: resource, State: "S", Error: "read failed"}
	if err := r.receivePacket("", &packet); err != nil {
		t.Fatal(err)
	}
	if c.rights.Phase() != mosi.MR {
		t.Fatal("read failure cancelled stronger request")
	}
	select {
	case <-pending.Flight().Done():
		t.Fatal("read failure woke write waiters")
	default:
	}
	packet.State = "M"
	if err := r.receivePacket("", &packet); err != nil {
		t.Fatal(err)
	}
	select {
	case <-pending.Flight().Done():
		if pending.Flight().Err() == nil {
			t.Fatal("write failure succeeded")
		}
	default:
		t.Fatal("write failure did not wake waiters")
	}
}

func TestReplicationCurrentInvalidationAndFutureEpoch(t *testing.T) {
	r := replicaRecoveryFixture()
	r.master = false
	r.upstream = &mosiPeer{outgoing: make(chan mosiMessage, 8), done: make(chan struct{})}
	resource := mosiResource{Database: "tenant"}
	c := r.resourceCell(resource)
	c.rights.Initialize(mosi.S)
	r.directory.Records[resource.key()] = replicaRecord{Resource: resource, State: "S", Epoch: 9}
	packet := mosiPacket{Kind: "invalidate", ID: "invalidate", Resource: resource, Epoch: 10}
	if err := r.invalidateResource(context.Background(), "", packet); err == nil || c.rights.Phase() != mosi.S {
		t.Fatal("future invalidation changed current authority")
	}
	packet.Epoch = 9
	for range 2 {
		if err := r.invalidateResource(context.Background(), "", packet); err != nil {
			t.Fatal(err)
		}
		if c.rights.Phase() != mosi.IO {
			t.Fatal("current invalidation retained read rights")
		}
		reply := <-r.upstream.outgoing
		if reply.Packet.Kind != "ack" || reply.Packet.Epoch != 9 || reply.Packet.ID != packet.ID {
			t.Fatal("invalidation lost operation identity")
		}
	}
}

func TestReplicationInvalidationRacingNewGrant(t *testing.T) {
	for range 100 {
		r := replicaRecoveryFixture()
		r.master = false
		r.upstream = &mosiPeer{outgoing: make(chan mosiMessage, 8), done: make(chan struct{})}
		resource := mosiResource{Database: "tenant"}
		c := r.resourceCell(resource)
		c.rights.Initialize(mosi.S)
		pending, _ := c.rights.Request(mosi.M)
		r.directory.Records[resource.key()] = replicaRecord{Resource: resource, State: "S", Epoch: 9}
		start := make(chan struct{})
		var workers sync.WaitGroup
		workers.Add(2)
		go func() {
			defer workers.Done()
			<-start
			if err := r.invalidateResource(context.Background(), "", mosiPacket{Kind: "invalidate", ID: "old", Resource: resource, Epoch: 9}); err != nil {
				t.Error(err)
			}
		}()
		go func() {
			defer workers.Done()
			<-start
			if err := r.acceptGrant(mosiPacket{ID: pending.Flight().ID(), Resource: resource, State: "M", Epoch: 10, Owner: r.id}); err != nil {
				t.Error(err)
			}
		}()
		close(start)
		workers.Wait()
		if c.rights.Phase() != mosi.M || r.directory.Records[resource.key()].Epoch != 10 {
			t.Fatal("old invalidation raced newer grant publication")
		}
	}
}
