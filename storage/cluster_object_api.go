/*
Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later
*/
package storage

import "fmt"
import "strings"
import "context"
import "runtime"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

// Object references use persistent authenticated node connections. They are not
// connection handles themselves; imported values and their GC lease own lifetime.
func (service *nodeScanService) objectEndpoint(ctx context.Context, address string) (*mosiPeer, error) {
	endpoint, err := mosiEndpointURL(address)
	if err != nil {
		return nil, err
	}
	service.mu.Lock()
	for peer := range service.peers {
		if peer.objectEndpoint == endpoint {
			select {
			case <-peer.done:
				continue
			default:
				service.mu.Unlock()
				return peer, nil
			}
		}
	}
	service.mu.Unlock()
	if service.cluster != nil {
		for _, link := range service.cluster.view.Load().links {
			candidate, _ := mosiEndpointURL(link.member.Address)
			if candidate == endpoint {
				select {
				case <-link.peer.done:
					continue
				default:
					return link.peer, nil
				}
			}
		}
	}
	peer, _, err := service.connect(ctx, address)
	return peer, err
}
func (service *nodeScanService) objectStatus() scm.Scmer {
	r := service.objectRegistry()
	r.mu.Lock()
	defer r.mu.Unlock()
	rows := make([]scm.Scmer, 0, len(r.entries))
	for uri, entry := range r.entries {
		active := len(entry.leases)
		rows = append(rows, scm.NewSlice([]scm.Scmer{scm.NewString("uri"), scm.NewString(uri), scm.NewString("type"), scm.NewString(entry.object.ClusterObjectType()), scm.NewString("references"), scm.NewInt(int64(active)), scm.NewString("offers"), scm.NewInt(int64(len(entry.tokens)))}))
	}
	return scm.NewSlice(rows)
}
func initClusterObjectAPI(en *scm.Env) {
	scm.Declare(en, &scm.Declaration{Name: "cluster_objects", Fn: func(a ...scm.Scmer) scm.Scmer {
		service := nodeScans.Load()
		if service == nil {
			panic("cluster protocol is disabled")
		}
		ctx := scanQueryContext(scmerToTxContext(a[0]), context.Background())
		peer, err := service.objectEndpoint(ctx, a[1].String())
		if err != nil {
			panic(err)
		}
		var sources remoteScanDescription
		if err := service.control(ctx, peer, "scan.describe", remoteScanDescribe{scanWireVersion, a[2].String(), a[3].String()}, &sources); err != nil {
			panic(err)
		}
		defer service.discardObjectOffers(peer, sources.ClusterObjectOffers())
		values := make([]scm.Scmer, len(sources))
		for i, source := range sources {
			if source.Object == nil {
				panic("source omitted its generation reference")
			}
			value, err := service.importObject(ctx, peer, *source.Object)
			if err != nil {
				panic(err)
			}
			values[i] = value
		}
		return scm.NewSlice(values)
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true, Params: []*scm.TypeDescriptor{{Kind: "any", Label: "transaction"}, {Kind: "string", Label: "endpoint"}, {Kind: "string", Label: "database"}, {Kind: "string", Label: "table"}}, Return: &scm.TypeDescriptor{Kind: "list"}}})
	scm.Declare(en, &scm.Declaration{Name: "cluster_object_call", Fn: func(a ...scm.Scmer) scm.Scmer {
		object, ok := scm.ClusterObjectOf(a[1])
		if !ok {
			panic("value has no cluster object endpoint")
		}
		ctx := scanQueryContext(scmerToTxContext(a[0]), context.Background())
		value, err := scm.ClusterObjectValue(object, ctx, scm.ClusterRequest{Method: a[2].String(), Arguments: a[3:]})
		if err != nil {
			panic(err)
		}
		return value
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true, Params: []*scm.TypeDescriptor{{Kind: "any", Label: "transaction"}, {Kind: "any", Label: "object"}, {Kind: "string", Label: "method"}, {Kind: "any", Label: "arguments", Variadic: true}}, Return: &scm.TypeDescriptor{Kind: "any"}}})
	scm.Declare(en, &scm.Declaration{Name: "cluster_object_status", Fn: func(a ...scm.Scmer) scm.Scmer {
		// Explicit maintenance only; neither queries nor the registry force GC.
		if len(a) > 0 && a[0].Bool() {
			runtime.GC()
		}
		service := nodeScans.Load()
		if service == nil {
			return scm.NewString("[]")
		}
		data, err := json.Marshal(service.objectStatus())
		if err != nil {
			panic(err)
		}
		return scm.NewString(string(data))
	}, Type: &scm.TypeDescriptor{Kind: "func", HasSideEffects: true, Params: []*scm.TypeDescriptor{{Kind: "bool", Label: "collect", Optional: true}}, Return: &scm.TypeDescriptor{Kind: "string"}}})
}

// Resolving a serialized reference outside a procedure still uses the same
// authenticated endpoint and importer. Procedure decoding supplies its explicit
// invocation resolver so no network resolution remains in generated row code.
func resolveRuntimeClusterObject(ref scm.ClusterReference) (scm.Scmer, error) {
	service := nodeScans.Load()
	if service == nil {
		return scm.NewNil(), fmt.Errorf("cluster protocol is disabled")
	}
	peer := service.objectConnection(clusterObjectPeer{ref.Owner, ref.Incarnation})
	if peer == nil {
		slash := strings.LastIndex(ref.URI, "/")
		if slash < 0 {
			return scm.NewNil(), fmt.Errorf("invalid object URI")
		}
		var err error
		peer, err = service.objectEndpoint(context.Background(), ref.URI[:slash])
		if err != nil {
			return scm.NewNil(), err
		}
	}
	return service.importObject(context.Background(), peer, ref)
}
