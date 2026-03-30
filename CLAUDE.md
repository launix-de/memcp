all instructions see AGENTS.md

## Branch: unnesting4 — Neumann Query Unnesting

### Papers (MUST READ at start of every session)
- `/home/carli/projekte/memcp/papers/Unnesting-Arbitrary-Queries.pdf` (Neumann/Kemper NK15, BTW 2015)
- `/home/carli/projekte/memcp/papers/neumann-improving-unnesting-btw2025.pdf` (Neumann BTW 2025)

### Core Principles
1. **NO fallback plans for inline evaluation.** Every query is unnestable per Neumann. If `unnest_subselect` returns nil, it's a BUG — fix the unnesting, don't add inline workarounds.
2. **Every inner_select is unnestable.** The old `build_scalar_subselect`, `build_in_subselect`, `build_exists_subselect` were removed intentionally. They must NOT be brought back.
3. **Clean architecture only.** No workarounds, no magic strings, no special-case hacks. The flat table list + scoped group stages + partition stages IR is the correct abstraction.
4. **Tests green through bugfixing, not code destruction.** Identify WHY a case fails, trace the IR, find the root cause, fix it cleanly. Do NOT disable tests, mark them noncritical, or add fallback paths.

### Current Regression Status (60 regressions vs master)
The removed inline subselect functions handled cases that `unnest_subselect` currently returns nil for:
- `us_outer_in_fields` → nil (outer refs in SELECT list of subquery)
- `us_single_tbl = false` → nil (multi-table subselects)
- UNION ALL subqueries → nil
- Various computed expressions in subselect fields

These are NOT unsolvable — Neumann covers all of them. They need proper implementation of the push-down rules, not fallback plans.
