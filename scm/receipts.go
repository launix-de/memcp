/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

package scm

import "sync"

// CollectReports owns a reduction for one action. The action must wait for
// all its producers before returning. Each producer reports a completed batch;
// neither this helper nor its callbacks retain engine or session identities.
// Completion runs once, including when the action panics, with its final
// accumulator and a flag telling whether the action returned successfully.
func CollectReports(a ...Scmer) (result Scmer) {
	var mu sync.Mutex
	accumulator := a[1]
	completed := false
	defer func() { Apply(a[3], accumulator, NewBool(completed)) }()
	report := NewFunc(func(values ...Scmer) Scmer {
		if len(values) != 1 {
			panic("report collector requires one value")
		}
		mu.Lock()
		defer mu.Unlock()
		accumulator = Apply(a[2], accumulator, values[0])
		return values[0]
	})
	result = Apply(a[0], report)
	completed = true
	return
}

func RegisterReceiptPrimitives() {
	report := &TypeDescriptor{Kind: "func", HasSideEffects: true, Params: []*TypeDescriptor{{Kind: "any"}}, Return: &TypeDescriptor{Kind: "any"}}
	Declare(&Globalenv, &Declaration{Name: "collect_reports", Fn: CollectReports, Type: &TypeDescriptor{Kind: "func", Description: "Reduce invocation-owned completed batch reports and publish once", HasSideEffects: true, Params: []*TypeDescriptor{
		{Kind: "func", HasSideEffects: true, Params: []*TypeDescriptor{report}, Return: &TypeDescriptor{Kind: "any"}},
		{Kind: "any"},
		{Kind: "func", Params: []*TypeDescriptor{{Kind: "any"}, {Kind: "any"}}, Return: &TypeDescriptor{Kind: "any"}},
		{Kind: "func", HasSideEffects: true, Params: []*TypeDescriptor{{Kind: "any"}, {Kind: "bool"}}, Return: &TypeDescriptor{Kind: "any"}},
	}, Return: &TypeDescriptor{Kind: "any"}}})
}
