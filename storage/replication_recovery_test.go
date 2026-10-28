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
	cell := r.obtain(resource, mosi.M, "", nil)
	if cell.ownstate.Phase() != mosi.M {
		t.Fatal("local permission was not published")
	}
	r.localRecord(resource, mosi.IO, "")
	cell.ownstate.Initialize(mosi.IO)
	pending, _ := cell.ownstate.Request(mosi.M)
	if err := r.acceptGrant(mosiPacket{ID: pending.Flight().ID(), Resource: resource, State: "M", Owner: r.id, Epoch: 2}); err != nil {
		t.Fatal(err)
	}
	<-pending.Flight().Done()
	if cell.ownstate.Phase() != mosi.M {
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
	record := r.directory.Records[r.resourceKey(resource)]
	if r.resourceCell(resource).clusterstate.Phase() != mosi.M {
		t.Fatal("live delegated owner did not establish outward domain M")
	}
	if record.Owner != peer || record.State != "IO" || r.resourceCell(resource).ownstate.Phase() != mosi.IO {
		t.Fatal("master used a remote owner's backup")
	}
	if !r.resourceKnown(resource) {
		t.Fatal("reported modifier cannot be routed")
	}
	// Losing the modifier never makes the master's backend authoritative.
	failure := func() (failure any) {
		defer func() { failure = recover() }()
		r.obtain(resource, mosi.S, "", nil)
		return nil
	}()
	if failure != mosi.NetworkError {
		t.Fatalf("unreachable modifier did not fail: %v", failure)
	}
	if r.directory.Records[r.resourceKey(resource)].Owner != peer || r.resourceCell(resource).ownstate.Phase() != mosi.IO {
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
		r.directory.Records[r.resourceKey(item)] = replicaRecord{Resource: item, Owner: peer, Epoch: 7, State: "IO", InitialClaimed: true, Requests: map[string]replicaRequestMark{peer: {Sequence: 900, Write: true}}}
	}
	reply, err := r.recoverPeer(context.Background(), peer, replicaRequest{Boot: uuid.NewString(), Resources: []replicaRecord{{Resource: resource, Content: "local-wal-content"}}}, replicaAuth{Databases: map[string]bool{"tenant": true}})
	if err != nil || len(reply.Resources) != 1 {
		t.Fatal("slave restart handshake failed", err)
	}
	recovered := reply.Resources[0]
	if recovered.Owner != peer || recovered.State != "M" || recovered.Epoch != 7 || recovered.Content != "local-wal-content" || !recovered.InitialClaimed {
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
	if err != nil || reply.Resources[0].State != "O" || !r.directory.Records[r.resourceKey(resource)].Readers[reader] {
		t.Fatal("recovery silently removed a reader", err)
	}
}

func TestReplicationGrantedReadsIgnoreDirectoryLocks(t *testing.T) {
	r := replicaRecoveryFixture()
	resource := mosiResource{Database: "tenant"}
	cell := r.resourceCell(resource)
	cell.ownstate.Initialize(mosi.S)
	// The complete manager/control plane can be busy without a RAM write or lock
	// in the already authorized schema path.
	r.mu.Lock()
	var workers sync.WaitGroup
	done := make(chan struct{})
	workers.Add(1)
	go func() {
		defer workers.Done()
		for i := 0; i < 10000; i++ {
			r.obtain(resource, mosi.S, "", nil)
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
	content := uuid.NewString()
	if err := r.saveBackup(resource, content); err != nil {
		t.Fatal(err)
	}
	r.localRecord(resource, mosi.M, content)
	// Another node may share system's backend, but must not share our cache or
	// the last complete physical content of its independent safety copy.
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
	if restarted.backups[restarted.resourceKey(resource)].Content != content {
		t.Fatal("completed backup placement did not survive restart")
	}
	if len(restarted.directory.Records) != 0 || restarted.resourceKnown(resource) {
		t.Fatal("persistent data reconstructed MOSI rights")
	}
	if files, err := filepath.Glob(filepath.Join(Basepath, ".replication-state*")); err != nil || len(files) != 0 {
		t.Fatal("native persistence created a private replication journal", files, err)
	}
}

// Native config updates use INSERT ... ON DUPLICATE UPDATE without a scan
// mapper. Update-only tombstones must survive an actual native WAL reload.
func TestReplicationConfigPromotionSurvivesRestart(t *testing.T) {
	source, engine := createDurabilityTestTable(t, "treplication_placement_restart", 0)
	config, _ := CreateTable(source.schema.Name, "config", Safe, false)
	for _, col := range []string{"node", "key", "value"} {
		config.CreateColumn(col, "text", nil, nil)
	}
	createTableKey(config, uniqueKey{Id: "PRIMARY", Cols: []string{"node", "key"}}, nil)
	r := replicaRecoveryFixture()
	r.config = config
	first := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
	second := mosiResource{Database: "tenant", Table: "other", Shard: uuid.NewString()}
	for _, resource := range []mosiResource{first, second} {
		if err := r.saveBackup(resource, resource.Shard); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.writeConfig(replicaConfigEntry{Kind: "backup", Name: first.key()}, replicaConfigEntry{Kind: "backup", Name: second.key()}); err != nil {
		t.Fatal(err)
	}
	reopened := newDatabase()
	reopened.Name, reopened.persistence, reopened.srState = source.schema.Name, engine, COLD
	reopened.ensureLoaded()
	defer reopened.closeTransactionLog()
	restarted := replicaRecoveryFixture()
	restarted.id = r.id
	if err := restarted.loadConfig(reopened.tables.Get("config")); err != nil {
		t.Fatal(err)
	}
	if len(restarted.backups) != 0 {
		t.Fatalf("removed placements returned after native WAL replay: %+v", restarted.backups)
	}
}

func TestReplicationRecoveryInterruptedHandover(t *testing.T) {
	for _, complete := range []bool{false, true} {
		t.Run(map[bool]string{false: "incomplete-data", true: "complete-data"}[complete], func(t *testing.T) {
			r := replicaRecoveryFixture()
			source := uuid.NewString()
			resource := mosiResource{Database: "tenant"}
			r.restoring[resource.Database] = true
			content := "prepared-schema-content-hash"
			backupGeneration := "old-schema-content-hash"
			if complete {
				backupGeneration = content
			}
			r.backups[resource.key()] = replicaBackup{Resource: resource, Content: backupGeneration}
			proof := replicaRecord{Resource: resource, Owner: source, State: "IO", Handoff: r.id, Epoch: 7, Content: content}
			_, err := r.recoverPeer(context.Background(), source, replicaRequest{Boot: uuid.NewString(), Resources: []replicaRecord{proof}}, replicaAuth{Admin: true})
			if err != nil {
				t.Fatal(err)
			}
			record := r.directory.Records[r.resourceKey(resource)]
			wantOwner := source
			if complete {
				wantOwner = r.id
			}
			if record.Owner != wantOwner || !r.resourceKnown(resource) {
				t.Fatal("interrupted handover has no safe data custodian", record)
			}
			if !complete && r.resourceCell(resource).ownstate.Phase() != mosi.IO {
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
	proof := replicaRecord{Resource: resource, Owner: source, State: "IO", Handoff: target, Epoch: 7, Content: "prepared-content"}
	if _, err := r.recoverPeer(context.Background(), source, replicaRequest{Boot: uuid.NewString(), Resources: []replicaRecord{proof}}, replicaAuth{Admin: true}); err != nil {
		t.Fatal(err)
	}
	if r.resourceKnown(resource) {
		t.Fatal("unreported recipient's possible M was reclaimed")
	}
	if r.resourceCell(resource).clusterstate.Phase() != mosi.IO {
		t.Fatal("unconfirmed handover established outward authority")
	}

	// The still-running recipient confirms that no M grant was installed.
	recipient := replicaRecord{Resource: resource, Owner: source, State: "IO", Epoch: 7}
	if _, err := r.recoverPeer(context.Background(), target, replicaRequest{Boot: uuid.NewString(), Resources: []replicaRecord{recipient}}, replicaAuth{Admin: true}); err != nil {
		t.Fatal(err)
	}
	if !r.resourceKnown(resource) || r.directory.Records[r.resourceKey(resource)].Owner != source {
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
	resource := mosiResource{Database: table.schema.Name, Table: table.Name, Shard: completed.uuid.String()}
	r := replicaRecoveryFixture()
	r.restoring[resource.Database] = true
	r.backups[resource.key()] = replicaBackup{Resource: resource, Content: completed.uuid.String()}
	// A schema packet reached storage, but its new physical data UUID did not.
	placeholder := new(storageShard)
	encoded := []byte(`{"uuid":"` + uuid.NewString() + `","mosi":"` + resource.Shard + `"}`)
	if err := json.Unmarshal(encoded, placeholder); err != nil {
		t.Fatal(err)
	}
	placeholder.t = table
	r.forwardResource(resource, mosiResource{Database: resource.Database, Table: resource.Table, Shard: placeholder.uuid.String()})
	table.mu.Lock()
	table.Shards = []*storageShard{placeholder}
	table.publishTopologyLocked()
	table.mu.Unlock()
	table.schema.save()
	owner := uuid.NewString()
	report := replicaRecord{Resource: resource, Owner: owner, State: "O", Epoch: 7, Content: placeholder.uuid.String(), Readers: map[string]bool{r.id: true}}
	if _, err := r.recoverPeer(context.Background(), owner, replicaRequest{Boot: uuid.NewString(), Resources: []replicaRecord{report}}, replicaAuth{Admin: true}); err != nil {
		t.Fatal(err)
	}
	cell := r.resourceCell(resource)
	if cell.ownstate.Phase() != mosi.S || cell.current.Load() == nil || cell.current.Load().uuid != completed.uuid {
		t.Fatal("recovery published S for an incomplete advertised content")
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
	if !r.resourceKnown(resource) || r.directory.Records[r.resourceKey(resource)].Owner != creator || cell.ownstate.Phase() != mosi.IO {
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
	resource := mosiResource{Database: table.schema.Name, Table: table.Name, Shard: original.uuid.String()}
	r.restoring[resource.Database] = true
	report := replicaRecord{Resource: resource, State: "IO", Owner: source, Handoff: r.id, Epoch: 7, Content: original.uuid.String()}
	if _, err := r.recoverPeer(context.Background(), source, replicaRequest{Boot: uuid.NewString(), Resources: []replicaRecord{report}}, replicaAuth{Admin: true}); err != nil {
		t.Fatal(err)
	}
	if r.directory.Records[r.resourceKey(resource)].Owner != r.id {
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
	r.resourceCell(mosiResource{Database: table.schema.Name}).ownstate.Initialize(mosi.M)
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
		t.Fatal("remote schema redirected storage away from the local current content/WAL")
	}
	remoteGeneration, err := r.schemaDigest(remoteSchema)
	if err != nil {
		t.Fatal(err)
	}
	localGeneration, err := r.schemaDigest(engine.ReadSchema())
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
	c.ownstate.Initialize(mosi.M)
	r.directory.Records[r.resourceKey(resource)] = replicaRecord{Resource: resource, State: "M", Owner: r.id, Epoch: 9}
	if err := r.invalidateResource(context.Background(), "", mosiPacket{Kind: "invalidate", ID: "old-revocation", Resource: resource, Epoch: 8}); err != nil {
		t.Fatal(err)
	}
	if c.ownstate.Phase() != mosi.M || r.directory.Records[r.resourceKey(resource)].State != "M" {
		t.Fatal("old invalidation revoked newer M")
	}
}

func TestReplicationDuplicateWriteDoesNotReacquireOwnership(t *testing.T) {
	r := replicaRecoveryFixture()
	resource := mosiResource{Database: "tenant"}
	c := r.resourceCell(resource)
	c.ownstate.Initialize(mosi.IO)
	pending, _ := c.ownstate.Request(mosi.M)
	r.directory.Records[r.resourceKey(resource)] = replicaRecord{Resource: resource, State: "IO", Owner: r.id, Epoch: 5}
	packet := mosiPacket{Kind: "write", ID: pending.Flight().ID(), Resource: resource, Sequence: 1}
	if err := r.coordinate(context.Background(), r.id, packet); err != nil {
		t.Fatal(err)
	}
	epoch := r.directory.Records[r.resourceKey(resource)].Epoch
	if err := r.coordinate(context.Background(), r.id, packet); err != nil {
		t.Fatal(err)
	}
	if r.directory.Records[r.resourceKey(resource)].Epoch != epoch {
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
		for _, mismatch := range []string{"peer", "resource", "epoch", "content"} {
			t.Run(kind+"/"+mismatch, func(t *testing.T) {
				r := replicaRecoveryFixture()
				source := "slave"
				peer := &mosiPeer{outgoing: make(chan mosiMessage, 8), done: make(chan struct{})}
				r.peers = map[string]*mosiPeer{source: peer, "other": peer}
				step := &replicaStep{peer: source, resource: resource, epoch: 9, content: "current", done: make(chan struct{})}
				r.steps.Store("handover", step)
				packet := mosiPacket{Kind: kind, ID: "handover", Resource: resource, Epoch: 9, Content: "current"}
				switch mismatch {
				case "peer":
					source = "other"
				case "resource":
					packet.Resource.Database = "other"
				case "epoch":
					packet.Epoch--
				case "content":
					packet.Content = "old"
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
					packet = mosiPacket{Kind: kind, ID: "handover", Resource: resource, Epoch: 9, Content: "current"}
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
	c.ownstate.Initialize(mosi.IO)
	pending, _ := c.ownstate.Request(mosi.M)
	r.directory.Records[r.resourceKey(resource)] = replicaRecord{Resource: resource, Owner: "other", Epoch: 9, State: "IO"}
	err := r.acceptGrant(mosiPacket{ID: pending.Flight().ID(), Resource: resource, Epoch: 8, State: "M", Owner: r.id})
	if err == nil || c.ownstate.Phase() != mosi.IO || r.directory.Records[r.resourceKey(resource)].Epoch != 9 {
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
	c.ownstate.Initialize(mosi.IO)
	read, _ := c.ownstate.Request(mosi.S)
	c.ownstate.Request(mosi.M) // upgrade wins before the read grant arrives
	r.directory.Records[r.resourceKey(resource)] = replicaRecord{Resource: resource, Owner: r.id, Epoch: 5, State: "IO"}
	packet := mosiPacket{Kind: "read", ID: read.Flight().ID(), Resource: resource, Sequence: 1}
	if err := r.coordinate(context.Background(), r.id, packet); err != nil {
		t.Fatal(err)
	}
	if c.ownstate.Phase() != mosi.MR {
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
	if c.ownstate.Phase() != mosi.M || r.directory.Records[r.resourceKey(resource)].Epoch != 6 {
		t.Fatal("write upgrade lost or executed repeatedly")
	}
}

func TestReplicationStaleTransferDoesNotOpenBackend(t *testing.T) {
	r := replicaRecoveryFixture()
	r.master = false
	r.upstream = &mosiPeer{outgoing: make(chan mosiMessage, 8), done: make(chan struct{})}
	resource := mosiResource{Database: "tenant"}
	r.directory.Records[r.resourceKey(resource)] = replicaRecord{Resource: resource, Epoch: 9}
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
	c.ownstate.Initialize(mosi.IO)
	pending, _ := c.ownstate.Request(mosi.S)
	c.ownstate.Request(mosi.M)
	packet := mosiPacket{Kind: "failure", ID: pending.Flight().ID(), Resource: resource, State: "S", Error: "read failed"}
	if err := r.receivePacket("", &packet); err != nil {
		t.Fatal(err)
	}
	if c.ownstate.Phase() != mosi.MR {
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
	c.ownstate.Initialize(mosi.S)
	r.directory.Records[r.resourceKey(resource)] = replicaRecord{Resource: resource, State: "S", Epoch: 9}
	packet := mosiPacket{Kind: "invalidate", ID: "invalidate", Resource: resource, Epoch: 10}
	if err := r.invalidateResource(context.Background(), "", packet); err == nil || c.ownstate.Phase() != mosi.S {
		t.Fatal("future invalidation changed current authority")
	}
	packet.Epoch = 9
	for range 2 {
		if err := r.invalidateResource(context.Background(), "", packet); err != nil {
			t.Fatal(err)
		}
		if c.ownstate.Phase() != mosi.IO {
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
		c.ownstate.Initialize(mosi.S)
		pending, _ := c.ownstate.Request(mosi.M)
		r.directory.Records[r.resourceKey(resource)] = replicaRecord{Resource: resource, State: "S", Epoch: 9}
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
		if c.ownstate.Phase() != mosi.M || r.directory.Records[r.resourceKey(resource)].Epoch != 10 {
			t.Fatal("old invalidation raced newer grant publication")
		}
	}
}

func TestReplicationReconnectPreservesCompletedRequestDomain(t *testing.T) {
	r := replicaRecoveryFixture()
	peer, boot := uuid.NewString(), uuid.NewString()
	resource := mosiResource{Database: "tenant"}
	mark := replicaRequestMark{Sequence: 900, Write: true, ID: "completed-flight", Completed: true}
	r.peersBoot[peer] = boot
	r.directory.Records[r.resourceKey(resource)] = replicaRecord{Resource: resource, Owner: peer, Epoch: 7, State: "IO", Requests: map[string]replicaRequestMark{peer: mark}}
	request := replicaRequest{Boot: boot, Resources: []replicaRecord{{Resource: resource, Content: "current"}}}
	reply, err := r.recoverPeer(context.Background(), peer, request, replicaAuth{Databases: map[string]bool{"tenant": true}})
	if err != nil || len(reply.Resources) != 1 {
		t.Fatal("same-incarnation recovery failed", err)
	}
	if r.directory.Records[r.resourceKey(resource)].Requests[peer] != mark {
		t.Fatal("reconnect reset completed request suppression")
	}
	if len(reply.Resources[0].Requests) != 0 {
		t.Fatal("recovery leaked another node's request marks")
	}
	request.Boot = uuid.NewString()
	if _, err := r.recoverPeer(context.Background(), peer, request, replicaAuth{Databases: map[string]bool{"tenant": true}}); err != nil {
		t.Fatal(err)
	}
	if _, present := r.directory.Records[r.resourceKey(resource)].Requests[peer]; present {
		t.Fatal("new process retained old flight identity")
	}
}

func TestReplicationRequestCompletionKeepsPublishedMapImmutable(t *testing.T) {
	r := replicaRecoveryFixture()
	resource := mosiResource{Database: "tenant"}
	peer := &mosiPeer{outgoing: make(chan mosiMessage, 8), done: make(chan struct{})}
	r.peers = map[string]*mosiPeer{"reader": peer}
	c := r.resourceCell(resource)
	c.ownstate.Initialize(mosi.O)
	pending, _ := c.ownstate.Request(mosi.M)
	r.directory.Records[r.resourceKey(resource)] = replicaRecord{Resource: resource, Owner: r.id, Epoch: 5, State: "O", Readers: map[string]bool{"reader": true}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- r.coordinate(ctx, r.id, mosiPacket{Kind: "write", ID: pending.Flight().ID(), Resource: resource, Sequence: 1})
	}()
	var packet *mosiPacket
	select {
	case message := <-peer.outgoing:
		packet = message.Packet
	case <-ctx.Done():
		t.Fatal("no invalidation sent")
	}
	r.mu.Lock()
	snapshot := r.directory.Records[r.resourceKey(resource)]
	before := snapshot.Requests[r.id]
	r.mu.Unlock()
	if before.Completed || before.ID != pending.Flight().ID() {
		t.Fatal("missing pending request publication")
	}
	if err := r.receivePacket("reader", &mosiPacket{Kind: "ack", ID: packet.ID, Resource: resource, Epoch: packet.Epoch}); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("handover did not complete")
	}
	if snapshot.Requests[r.id] != before {
		t.Fatal("completion mutated an already published request map")
	}
	if !r.directory.Records[r.resourceKey(resource)].Requests[r.id].Completed {
		t.Fatal("successor did not publish request completion")
	}
}

func TestReplicationTransferAckBindsCapturedGeneration(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "t_replica_ack_generation", 3)
	r := replicaRecoveryFixture()
	resource := mosiResource{Database: table.schema.Name, Table: table.Name, Shard: table.ActiveShards()[0].uuid.String()}
	if _, err := r.prepareResource(resource); err != nil {
		t.Fatal(err)
	}
	peer := &mosiPeer{outgoing: make(chan mosiMessage, 8), done: make(chan struct{})}
	r.peers = map[string]*mosiPeer{"slave": peer}
	defer close(peer.done)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- r.transmitResource(ctx, "slave", mosiPacket{ID: "snapshot", Resource: resource, Epoch: 9})
	}()
	for {
		select {
		case message := <-peer.outgoing:
			packet := message.Packet
			if packet.Kind == "begin" {
				value, exists := r.steps.Load(packet.ID)
				if !exists || packet.Content == "" || value.(*replicaStep).content != packet.Content {
					t.Fatal("ACK identity did not bind captured content")
				}
			}
			if packet.Kind == "end" {
				ack := *packet
				ack.Kind = "ack"
				if err := r.receivePacket("slave", &ack); err != nil {
					t.Fatal(err)
				}
				select {
				case err := <-done:
					if err != nil {
						t.Fatal(err)
					}
				case <-ctx.Done():
					t.Fatal("matching content ACK did not complete transfer")
				}
				return
			}
		case <-ctx.Done():
			t.Fatal("transfer did not finish")
		}
	}
}

func TestReplicationOuterAuthorityCannotAdmitDelegatedQueries(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "t_two_authority_delegation", 3)
	r := replicaRecoveryFixture()
	peer := uuid.NewString()
	shard := table.ActiveShards()[0]
	cell := r.declareShard(shard, peer, false)
	resource := *cell.resource.Load()
	schema := mosiResource{Database: table.schema.Name}
	r.directory.Records[r.resourceKey(schema)] = replicaRecord{Resource: schema, Owner: peer, State: "IO", Epoch: 1}
	schemaCell := r.resourceCell(schema)
	r.establishAuthority(schemaCell, r.directory.Records[r.resourceKey(schema)])
	previous := replication.Swap(r)
	defer replication.Store(previous)
	for _, access := range []func(){
		func() { shard.GetRead(nil).Release() }, func() { shard.GetWrite(nil).Release() },
		func() { table.schema.GetRead(nil).Release() }, func() { table.schema.GetWrite(nil).Release() },
	} {
		failure := func() (failure any) { defer func() { failure = recover() }(); access(); return nil }()
		if failure != mosi.NetworkError {
			t.Fatal("outer M authorized a delegated local access", failure)
		}
	}
	for _, c := range []*replicaCell{cell, schemaCell} {
		if c.clusterstate.Phase() != mosi.M || c.ownstate.Phase() != mosi.IO {
			t.Fatal("failed access changed domain ownership")
		}
	}
	if r.directory.Records[r.resourceKey(resource)].Owner != peer {
		t.Fatal("unreachable slave was silently promoted")
	}
}

func TestReplicationOwnReadsIgnoreOutwardHandover(t *testing.T) {
	r := replicaRecoveryFixture()
	resource := mosiResource{Database: "tenant"}
	c := r.resourceCell(resource)
	c.clusterstate.Initialize(mosi.M)
	c.ownstate.Initialize(mosi.S)
	handover := c.clusterstate.TryHandover(func() bool { return true })
	if handover == nil {
		t.Fatal("missing outward handover")
	}
	defer handover.Release()
	outer, own := c.clusterstate.Snapshot(), c.ownstate.Snapshot()
	// Busy control-plane locks and a closing outer slot add no synchronization
	// to an already authorized local reader. Test immutable publication identity
	// as well as completion, rather than merely checking the resulting phase.
	r.mu.Lock()
	c.acquire.Lock()
	c.directory.Lock()
	done := make(chan struct{})
	go func() {
		for range 10000 {
			r.obtain(resource, mosi.S, "", nil)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("local readers waited for outward/control ownership")
	}
	c.directory.Unlock()
	c.acquire.Unlock()
	r.mu.Unlock()
	<-done
	if c.clusterstate.Snapshot() != outer || c.ownstate.Snapshot() != own {
		t.Fatal("granted reads wrote an authority slot")
	}
}

func TestReplicationLeafGrantAndRevocationUpdateBothStates(t *testing.T) {
	r := replicaRecoveryFixture()
	r.master = false
	r.upstream = &mosiPeer{outgoing: make(chan mosiMessage, 8), done: make(chan struct{})}
	resource := mosiResource{Database: "tenant"}
	c := r.resourceCell(resource)
	pending, _ := c.ownstate.Request(mosi.M)
	if err := r.acceptGrant(mosiPacket{ID: pending.Flight().ID(), Resource: resource, Epoch: 9, Owner: r.id, State: "M"}); err != nil {
		t.Fatal(err)
	}
	if c.clusterstate.Phase() != mosi.M || c.ownstate.Phase() != mosi.M {
		t.Fatal("leaf grant missed an authority slot")
	}
	c.ownstate.BackendOnly()
	if c.ownstate.Phase() != mosi.IS || c.clusterstate.Phase() != mosi.M {
		t.Fatal("RAM eviction changed outward authority")
	}
	if err := r.invalidateResource(context.Background(), "", mosiPacket{ID: "revoke", Resource: resource, Epoch: 9}); err != nil {
		t.Fatal(err)
	}
	if c.ownstate.Phase() != mosi.IO || c.clusterstate.Phase() != mosi.IO {
		t.Fatal("outward revocation retained local permission")
	}
}

func TestReplicationRecoveryInventoryReportsOutwardAuthority(t *testing.T) {
	table, _ := createDurabilityTestTable(t, "t_two_authority_inventory", 3)
	r := replicaRecoveryFixture()
	r.master = false
	r.schemaShards(table.schema, r.id, true)
	c := r.shardCell(table.ActiveShards()[0])
	// Simulate the boundary of a future child group. The existing leaf adapter's
	// upstream-domain record remains ours; child-owner metadata belongs elsewhere.
	c.ownstate.Publish(mosi.IO)
	records := r.recoveryInventory(replicaAuth{Databases: map[string]bool{table.schema.Name: true}})
	found := false
	for _, record := range records {
		if record.Resource == *c.resource.Load() {
			found = true
			if record.State != "M" || record.Owner != r.id {
				t.Fatal("local IO hid delegated outward authority")
			}
		}
	}
	if !found {
		t.Fatal("missing inventory resource")
	}
}

func TestReplicationUnconfirmedReaderCannotRaiseOwnAuthority(t *testing.T) {
	r := replicaRecoveryFixture()
	resource := mosiResource{Database: "tenant"}
	r.restoring[resource.Database] = true
	r.backups[resource.key()] = replicaBackup{Resource: resource}
	reader, owner := uuid.NewString(), uuid.NewString()
	report := replicaRecord{Resource: resource, Owner: owner, State: "S", Epoch: 9, Readers: map[string]bool{r.id: true}}
	if _, err := r.recoverPeer(context.Background(), reader, replicaRequest{Boot: uuid.NewString(), Resources: []replicaRecord{report}}, replicaAuth{Admin: true}); err != nil {
		t.Fatal(err)
	}
	c := r.resourceCell(resource)
	if c.clusterstate.Phase() != mosi.IO || c.ownstate.Phase() != mosi.IO {
		t.Fatal("unconfirmed owner proof established stronger local than outward rights")
	}
}

func TestReplicationRestartRestoresUUIDEdgesWithoutRights(t *testing.T) {
	tbl, _ := createDurabilityTestTable(t, "t_restart_uuid_routes", 3)
	r := replicaRecoveryFixture()
	oldManager := replication.Swap(r)
	defer replication.Store(oldManager)
	original := tbl.ActiveShards()[0]
	r.declareShard(original, r.id, true)
	from := r.shardResource(original)
	RebuildTable(tbl, true, false)
	middle := r.shardResource(tbl.ActiveShards()[0])
	RebuildTable(tbl, true, false)
	current := r.shardResource(tbl.ActiveShards()[0])
	data, err := json.Marshal(tbl)
	if err != nil {
		t.Fatal(err)
	}
	loaded := new(table)
	if err := json.Unmarshal(data, loaded); err != nil {
		t.Fatal(err)
	}
	loaded.schema = tbl.schema
	for _, shard := range loaded.Shards {
		shard.t = loaded
	}
	replication.Store(nil)
	loaded.publishTopologyLocked()
	restarted := replicaRecoveryFixture()
	restarted.restoreTableRoutes(loaded)
	// An old peer may report a predecessor after the owner crashed following
	// another durable rebuild. Its stale hint cannot reverse the native edge.
	restarted.forwardResource(from, middle)
	if restarted.canonicalResource(from) != current || restarted.canonicalResource(middle) != current {
		t.Fatal("restart lost UUID forwarding or a stale peer reversed it")
	}
	cell := restarted.resourceCell(current)
	if cell.ownstate.Phase() != mosi.IO || cell.ownstate.Initialized() || len(restarted.directory.Records) != 0 {
		t.Fatal("persistent topology restored MOSI permission")
	}
}

func TestReplicationPromotionClearsAllNativePlacementAliases(t *testing.T) {
	source, engine := createDurabilityTestTable(t, "treplicationclearplacementaliases", 0)
	config, _ := CreateTable(source.schema.Name, "config", Safe, false)
	for _, name := range []string{"node", "key", "value"} {
		config.CreateColumn(name, "text", nil, nil)
	}
	createTableKey(config, uniqueKey{Id: "PRIMARY", Cols: []string{"node", "key"}}, nil)
	r := replicaRecoveryFixture()
	r.config = config
	old := mosiResource{Database: "tenant", Table: "items", Shard: uuid.NewString()}
	current := old
	current.Shard = uuid.NewString()
	unrelated := old
	unrelated.Shard = uuid.NewString()
	for _, resource := range []mosiResource{old, current, unrelated} {
		if err := r.saveBackup(resource, resource.Shard); err != nil {
			t.Fatal(err)
		}
	}
	other := replicaRecoveryFixture()
	other.config = config
	if err := other.saveBackup(old, old.Shard); err != nil {
		t.Fatal(err)
	}
	// Independently completed placements predate learning their native UUID edge.
	// Coherence merges their runtime entry, while both native rows still exist.
	r.forwardResource(old, current)
	if err := r.clearBackup(current); err != nil {
		t.Fatal(err)
	}
	live := replicaRecoveryFixture()
	live.id = r.id
	if err := live.loadConfig(config); err != nil {
		t.Fatal(err)
	}
	if len(live.backups) != 1 {
		t.Fatalf("runtime promotion cleanup failed: %+v", live.backups)
	}
	db := newDatabase()
	db.Name, db.persistence, db.srState = source.schema.Name, engine, COLD
	db.ensureLoaded()
	defer db.closeTransactionLog()
	restarted := replicaRecoveryFixture()
	restarted.id = r.id
	if err := restarted.loadConfig(db.tables.Get("config")); err != nil {
		t.Fatal(err)
	}
	if len(restarted.backups) != 1 || restarted.backups[unrelated.key()].Content != unrelated.Shard {
		t.Fatalf("promotion left old recovery placement or removed another resource: %+v", restarted.backups)
	}
	foreign := replicaRecoveryFixture()
	foreign.id = other.id
	if err := foreign.loadConfig(db.tables.Get("config")); err != nil {
		t.Fatal(err)
	}
	if len(foreign.backups) != 1 || foreign.backups[old.key()].Content != old.Shard {
		t.Fatal("promotion removed another node's completed copy")
	}
}
