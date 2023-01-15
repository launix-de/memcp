package main

type StorageIndex struct {
	cols []string // sort equal-cols alphabetically, so similar conditions are canonical
	savings float64 // store the amount of time savings here -> add selectivity (outputted / size) on each
	sortedItems StorageInt // we can do binary searches here
	main_count uint
	inactive bool
}

func (t *table) findOrCreateIndexFor(condition scmer, sort bool) *StorageIndex {
	// TODO: return the best index in t.indexes or create new
	// even better: stream results to chan uint
	// TODO: analyze condition for AND clauses
	// analyze each AND clause for: column, lower, upper bound
	// sort columns -> at first, the lower==upper alphabetically; then one lower!=upper according to best selectivity; discard the rest
	// find an index that has at least the columns in that order we're searching for
	// if the index is inactive, use the other one
	// if sort, we must build the index anyways
}

func rebuildIndexes(t1 *table, t2 *table) {
	// TODO rebuild index in database rebuild
	// check if indexes share same prefix -> leave out the shorter one
	// savings = 0.9 * savings (decrease)
	// according to memory pressure -> threshold for discard savings
	// -> mark inactive if we can don't want to store this index
	// if two indexes are prefixed, give up the shorter one and add to savings
}

// iterate over index
func (s *StorageIndex) iterate(lower []scmer, upperLast scmer) chan uint {
	result := make(chan uint, 64)
	return result

	savings_threshold := 0.2 // TODO: global value
	go func() {
		if s.inactive {
			// index is not built yet
			if s.savings < savings_threshold {
				// iterate over all items because we don't want to store the index
				for i := uint(0); i < s.main_count; i++ {
					result <- i
				}
				result.close()
				return
			} else {
				// TODO: rebuild index
				// -> make([]uint, s.main_count)
				// -> quicksort it
				// -> serialize into sortedItems
				// s.inactive = false
			}
		}
		// TODO tree traversal result <- index
		// TODO: find lower
		uint i = main_count / 2
		// TODO: iterate until upperLast or lower[high-1] differs
		result.close()
	}
}
