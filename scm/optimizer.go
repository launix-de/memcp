/*
Copyright (C) 2024  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package scm

// NOTE: The original optimizer relied on the interface-based Scmer type. To
// keep the build working while the runtime is being ported to the new tagged
// representation we provide minimal stubs that keep previous entry points but
// no longer perform aggressive rewrites. When optimisation logic is migrated
// it can replace these helpers.

var SettingsHaveGoodBacktraces bool

// OptimizeProcToSerialFunction returns a callable suitable for repeated use.
// For already optimised functions we unwrap the underlying Go function. For
// other values we fall back to calling Apply so behaviour stays consistent.
func OptimizeProcToSerialFunction(val Scmer) func(...Scmer) Scmer {
	if val.IsNil() {
		return func(...Scmer) Scmer { return NewNil() }
	}

	switch auxTag(val.aux) {
	case tagFunc:
		return val.Func()
	case tagAny:
		if fn, ok := val.Any().(func(...Scmer) Scmer); ok {
			return fn
		}
		if proc, ok := val.Any().(Proc); ok {
			return func(args ...Scmer) Scmer {
				return Apply(NewAny(proc), args...)
			}
		}
	}

	// Fall back to dispatching through Apply; this keeps semantics identical
	// while the optimiser is being ported.
	return func(args ...Scmer) Scmer {
		return Apply(val, args...)
	}
}

// Optimize currently acts as a no-op. The call sites expect the original
// value back, so we simply return it unchanged.
func Optimize(val Scmer, env *Env) Scmer {
	return val
}

// OptimizeEx mirrors the old signature but simply reports that the value is
// unchanged and not a constant. This keeps the evaluator happy until the
// optimiser is reintroduced.
func OptimizeEx(val Scmer, env *Env, ome *optimizerMetainfo, useResult bool) (Scmer, bool, bool) {
	return val, true, false
}

// optimizerMetainfo existed in the old implementation. We keep a tiny shell so
// callers that manipulate it continue to compile, even though we do not use it
// right now.
type optimizerMetainfo struct{}

func newOptimizerMetainfo() optimizerMetainfo { return optimizerMetainfo{} }

func (ome *optimizerMetainfo) Copy() optimizerMetainfo { return optimizerMetainfo{} }
