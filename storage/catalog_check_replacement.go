/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

import "fmt"
import "github.com/launix-de/memcp/scm"

// Only one exact committed DROP/ADD pair per native CHECK identity belongs to
// this profile. The old recipe remains the actual predecessor proof; the new
// recipe is independently validated against the final selected row/header view.
func (record *catalogDecisionRecord) hasCheckReplacements() bool {
	for _, before := range record.CheckDrops {
		for _, after := range record.Checks {
			if before.Check != nil && after.Check != nil && scm.CaseFoldKey(before.Check.Id) == scm.CaseFoldKey(after.Check.Id) {
				return true
			}
		}
	}
	return false
}

func (record *catalogDecisionRecord) validateCheckReplacementProfile() error {
	if !record.hasCheckReplacements() {
		return fmt.Errorf("CHECK replacement journal requires an exact predecessor pair")
	}
	before := make(map[string]catalogCheckDeclaration, len(record.CheckDrops))
	for _, d := range record.CheckDrops {
		if d.Check == nil {
			return fmt.Errorf("CHECK replacement has no retired recipe")
		}
		name := scm.CaseFoldKey(d.Check.Id)
		if _, duplicate := before[name]; duplicate {
			return fmt.Errorf("CHECK replacement repeats a committed retirement")
		}
		before[name] = d
	}
	seen := make(map[string]bool, len(record.Checks))
	for _, d := range record.Checks {
		if d.Check == nil {
			return fmt.Errorf("CHECK replacement has no selected recipe")
		}
		name := scm.CaseFoldKey(d.Check.Id)
		if seen[name] {
			return fmt.Errorf("CHECK replacement repeats a selected addition")
		}
		seen[name] = true
		if old, replaced := before[name]; replaced {
			if old.Table != d.Table || old.Identity != d.Identity || old.Check.Id != d.Check.Id {
				return fmt.Errorf("CHECK replacement changed its native endpoint or identity")
			}
			for _, t := range record.Tables {
				if t != nil && t.Name == d.Table {
					return fmt.Errorf("CHECK replacement requires a committed predecessor generation")
				}
			}
			for _, drop := range record.TableDrops {
				if drop.Name == d.Table {
					return fmt.Errorf("CHECK replacement cannot retire its endpoint")
				}
			}
		}
	}
	return nil
}

func (record *catalogDecisionRecord) replacesCheck(d catalogCheckDeclaration) bool {
	if record.Format != 30 && record.Format != 31 || d.Check == nil {
		return false
	}
	for _, selected := range record.Checks {
		if selected.Check != nil && selected.Table == d.Table && selected.Identity == d.Identity && selected.Check.Id == d.Check.Id {
			return true
		}
	}
	return false
}
