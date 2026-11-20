/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "strings"
import "context"
import "net/url"
import "github.com/launix-de/memcp/scm"

// Catalog names are resolved dynamically through native schema ownership. They
// need no strong temporary registration or peer GC token. Runtime shard UUIDs,
// unlike names, identify one immutable generation and MUST be registered.
func (*table) ClusterObjectType() string { return "table" }
func (t *table) ClusterObjectPath() string {
	return "catalog/" + url.PathEscape(t.schema.Name) + "/" + url.PathEscape(t.clusterPublishedName())
}
func (t *table) executeClusterRequest(ctx context.Context, r scm.ClusterRequest) (any, error) {
	if err := ctx.Err(); err != nil {
		return scm.NewNil(), err
	}
	if r.Method != "view" {
		return scm.NewNil(), fmt.Errorf("unsupported catalog request %q", r.Method)
	}
	return scm.NewSlice([]scm.Scmer{scm.NewString(t.schema.Name), scm.NewString(t.clusterPublishedName())}), nil
}
func resolveClusterCatalog(uri string) (*table, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	marker := "/catalog/"
	at := strings.LastIndex(parsed.EscapedPath(), marker)
	if at < 0 {
		return nil, fmt.Errorf("not a catalog URI")
	}
	parts := strings.Split(parsed.EscapedPath()[at+len(marker):], "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid catalog URI")
	}
	dbName, err := url.PathUnescape(parts[0])
	if err != nil {
		return nil, err
	}
	tableName, err := url.PathUnescape(parts[1])
	if err != nil {
		return nil, err
	}
	db := GetDatabase(dbName)
	if db == nil {
		return nil, fmt.Errorf("unknown catalog database")
	}
	lease := db.GetRead(nil)
	defer lease.Release()
	t := db.GetTable(tableName)
	if t == nil {
		return nil, fmt.Errorf("unknown catalog table")
	}
	return t, nil
}
func init() {
	scm.RegisterClusterValueExtractor(TagTable, func(value scm.Scmer) scm.ClusterObject { return TableFromScmer(value) })
	scm.RegisterClusterObjectType("table", func(handle scm.ClusterHandle) scm.Scmer {
		t, err := resolveClusterCatalog(handle.Reference.URI)
		if err != nil {
			panic(err)
		}
		return NewTableScmer(t)
	})
}

func (t *table) clusterPublishedName() string {
	if name := t.publishedName.Load(); name != nil {
		return *name
	}
	return t.Name
}

func (t *table) Serialize(ctx *scm.SerializationContext) (scm.Scmer, error) {
	// Catalog identity is a portable name lookup, not a retained remote object.
	// This also serves ordinary local plan/cache serialization without a node
	// transport. The generic capture marker resolves it once on wire import;
	// native schema GetRead still guards lookup, never each mapped row.
	value := []scm.Scmer{scm.NewSymbol("table"), scm.NewString(t.schema.Name), scm.NewBool(false), scm.NewString(t.clusterPublishedName()), scm.NewBool(false), scm.NewNil()}
	if !t.committedAt(t.schema.catalogView().version) {
		creator := t.creator.Load()
		if ctx != nil {
			if tx := scmerToTxContext(ctx.InvocationScope()); tx != nil && (tx.ownsPendingTable(t) || tx.ownsScopedTable(t)) {
				creator = tx.transactionState
			}
		}
		if creator == nil || creator.State != TxActive || !(creator.ownsPendingTable(t) || creator.ownsScopedTable(t)) {
			return scm.NewNil(), fmt.Errorf("pending catalog reference requires its native transaction capability")
		}
		value[5] = scm.NewAny(creator)
	}
	return scm.NewSlice(value), nil
}

func (t *table) HandleClusterRequest(ctx context.Context, payload []byte) {
	scm.ReceiveClusterValue(ctx, payload, func(request scm.ClusterRequest) (any, error) { return t.executeClusterRequest(ctx, request) })
}

func (t *table) ClusterView(ctx context.Context) (scm.Scmer, error) {
	value, err := t.executeClusterRequest(ctx, scm.ClusterRequest{Method: "view"})
	return scm.ClusterViewValue(value, err)
}
