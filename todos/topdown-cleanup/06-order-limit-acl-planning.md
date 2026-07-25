# AP6: ORDER BY / LIMIT / OFFSET Against ACL Planning

## Problem

The unfiltered Omnestum document list is already slow. That means the planner
does not correctly balance ACL predicates against ordered pagination. Computing
"all visible documents" before sorting and limiting is too expensive.

## Required Planner Behaviour

- Represent order requirements explicitly in IR.
- Prefer ordered candidate drivers when compatible with filters.
- Apply ACL as a semijoin/filter on an ordered candidate stream when cheaper
  than full ACL materialization.
- Implement range-based braking for ORDER BY + LIMIT where safe:
  - maintain top-k threshold
  - stop shard/range scan when no later key can beat current threshold
- Handle OFFSET by collecting enough accepted rows, not by forcing a full scan
  when braking is possible.

## Correctness Requirements

Cover:

- ASC / DESC
- ties
- NULL order
- OFFSET
- LIMIT 0
- multi-shard merge
- delta inserts
- deletes
- non-covering ordered index

## Omnestum Focus

Important sort keys include:

- upload timestamp
- year
- location
- mandate
- file/document description
- document ID tie-breakers

ACL must be planned as part of WHERE, not as a late projection decoration.

## Tests

YAML coverage:

- ordered scan with expensive semijoin and LIMIT
- OFFSET with ACL misses
- multi-shard ordered merge with filtered candidates
- negative test where order cannot be reused

## Done

- The unfiltered Omnestum document list becomes significantly faster.
- EXPLAIN shows ordered driver and early limit/braking where applicable.
