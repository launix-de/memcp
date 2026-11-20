/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "context"
import "strings"
import "github.com/launix-de/memcp/scm"

// Default declarations amend a constructor-owned stored column. No row has
// consumed its default yet; later mutation phases need a separate native DDL
// contract and must not reuse this private-construction capability.
func (tx *TxContext) alterCatalogColumn(t *table, name, operation string, value scm.Scmer) scm.Scmer {
	if tx == nil || tx.State != TxActive || tx.decisionStarted || !tx.ownsPendingTable(t) {
		panic("column declaration requires its active native catalog owner")
	}
	switch operation {
	case "default", "default_expression", "add_default", "default_declaration", "drop_default":
	default:
		panic("transactional column alteration is not supported: " + operation)
	}
	if tx.coordinator != nil {
		result, err := tx.coordinator.Call(scanQueryContext(tx, context.Background()), scm.ClusterRequest{
			Method: "catalog_column", Arguments: []scm.Scmer{scm.NewString(t.schema.Name), scm.NewString(t.Name),
				scm.NewString(t.RuntimeIdentity), scm.NewString(name), scm.NewString(operation), value}})
		if err != nil {
			panic(err)
		}
		return result
	}
	result := scm.NewBool(false)
	tx.amendCatalogDeclaration(t, func(current *table) bool {
		for index, previous := range current.Columns {
			if (operation != "drop_default" && previous.Name != name) ||
				(operation == "drop_default" && (previous.DefaultName == "" || !strings.EqualFold(previous.DefaultName, name))) {
				continue
			}
			current.requireDeclaredColumnAlter(previous, operation, value)
			if previous.IsTemp || isRuntimeComputedColumn(previous) {
				panic("private column declaration requires an ordinary stored column")
			}
			replacement := cloneStoredColumnDeclaration(previous)
			result = replacement.Alter(operation, value)
			if replacement.DefaultName != "" && replacement.DefaultName != previous.DefaultName &&
				current.schema.hasDeclarationNameLocked(replacement.DefaultName, previous) {
				panic("declaration name already exists: " + replacement.DefaultName)
			}
			if operation == "add_default" || operation == "default_declaration" {
				result = scm.NewBool(true)
			}
			columns := append([]*column(nil), current.Columns...)
			columns[index] = replacement
			current.Columns = columns
			return true
		}
		if operation != "drop_default" {
			panic("column " + current.schema.Name + "." + current.Name + "." + name + " does not exist")
		}
		return false
	})
	return result
}

func (tx *transactionState) applyCatalogColumn(ctx context.Context, arguments []scm.Scmer) (scm.Scmer, error) {
	if len(arguments) != 6 {
		return scm.NewNil(), fmt.Errorf("invalid native catalog column declaration")
	}
	for _, argument := range arguments[:5] {
		if !argument.IsString() {
			return scm.NewNil(), fmt.Errorf("invalid native catalog column identity")
		}
	}
	invocation := &TxContext{transactionState: tx, invocationContext: ctx}
	db := GetDatabase(arguments[0].String())
	if db == nil {
		return scm.NewNil(), fmt.Errorf("catalog database was removed")
	}
	target := db.GetTableForTx(arguments[1].String(), false, invocation)
	if target == nil || target.RuntimeIdentity != arguments[2].String() {
		return scm.NewNil(), fmt.Errorf("catalog column generation was replaced")
	}
	return invocation.alterCatalogColumn(target, arguments[3].String(), arguments[4].String(), arguments[5]), nil
}
