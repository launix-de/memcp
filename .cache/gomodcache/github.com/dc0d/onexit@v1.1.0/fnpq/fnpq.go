// Package fnpq contains type-specific version of same "generic" types,
// cpoied from standard library.
package fnpq

// An Item is something we manage in a priority queue.
type Item struct {
	Action   func() // The value of the item; arbitrary.
	priority int    // The priority of the item in the queue.
	// The index is needed by update and is maintained by the heap.Interface methods.
	index int // The index of the item in the heap.
}

// NewItem .
func NewItem(action func(), priority int) *Item {
	return &Item{
		Action:   action,
		priority: priority,
	}
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].priority > pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push .
func (pq *PriorityQueue) Push(x *Item) {
	n := len(*pq)
	item := x
	item.index = n
	*pq = append(*pq, item)
}

// Pop .
func (pq *PriorityQueue) Pop() *Item {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// update modifies the priority and value of an Item in the queue.
func (pq *PriorityQueue) update(item *Item, action func(), priority int) {
	item.Action = action
	item.priority = priority
	Fix(pq, item.index)
}
