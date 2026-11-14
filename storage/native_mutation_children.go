/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package storage

// Child capabilities belong to their SAME operation, not a transaction-wide
// registry. QueryWork still performs the normal join; this intrusive ownership
// retains a failed child continuation after endStatement detaches that list.
// All sibling links are protected by the parent's mu. parent is immutable.
func (parent *nativeMutationOperation) linkChildLocked(child *nativeMutationOperation) {
	child.parent = parent
	child.nextSibling = parent.firstChild
	if parent.firstChild != nil {
		parent.firstChild.previousSibling = child
	}
	parent.firstChild = child
	child.linked = true
}

func (parent *nativeMutationOperation) unlinkChildLocked(child *nativeMutationOperation) {
	if !child.linked {
		return
	}
	if child.previousSibling != nil {
		child.previousSibling.nextSibling = child.nextSibling
	} else {
		parent.firstChild = child.nextSibling
	}
	if child.nextSibling != nil {
		child.nextSibling.previousSibling = child.previousSibling
	}
	child.nextSibling, child.previousSibling, child.linked = nil, nil, false
}

// No parent/native-state lock is held while closing a child. A successful
// child removes itself, so completed per-row nested SQL statements cannot
// accumulate in this owner merely because the outer statement is still live.
func (operation *nativeMutationOperation) joinChildren() {
	for {
		operation.mu.Lock()
		child := operation.firstChild
		operation.mu.Unlock()
		if child == nil {
			return
		}
		child.close()
	}
}

// Recovery never turns a callback/reader's terminal control into a wait for
// itself. Only already sealed/drained work may be retried. Normal parent close
// remains the sole owner allowed to join actual running producer work.
func (operation *nativeMutationOperation) childrenDrainedLocked() bool {
	for child := operation.firstChild; child != nil; child = child.nextSibling {
		child.mu.Lock()
		ready := child.closed && child.producerDone && child.activeCalls == 0 && child.openFrames == 0 && child.workDrained && child.childrenDrainedLocked()
		if ready && child.effectScope != nil {
			child.effectScope.mu.Lock()
			ready = child.effectScope.active == 0
			child.effectScope.mu.Unlock()
		}
		child.mu.Unlock()
		if !ready {
			return false
		}
	}
	return true
}
