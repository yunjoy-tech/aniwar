package event

import (
	"fmt"
	"sort"
)

// IListener interface
type IListener interface {
	Handle(e IEvent) error
}

type ListenerFunc func(e IEvent) error

// Handle event. implements the IListener interface
func (fn ListenerFunc) Handle(e IEvent) error {
	return fn(e)
}

// ISubscriber is the event subscriber interface.
// you can register multi event listeners in a struct func.
type ISubscriber interface {
	// SubscribedEvents register event listeners
	// key: is event name
	// value: can be IListener or ListenerItem interface
	SubscribedEvents() map[string]interface{}
}

// ListenerItem storage an event listener and it's priority value.
type ListenerItem struct {
	Priority int
	Listener IListener
}

// ListenerQueue storage sorted IListener instance.
type ListenerQueue struct {
	items []*ListenerItem
}

// Len get items length
func (lq *ListenerQueue) Len() int {
	return len(lq.items)
}

// IsEmpty get items length == 0
func (lq *ListenerQueue) IsEmpty() bool {
	return len(lq.items) == 0
}

// Push get items length
func (lq *ListenerQueue) Push(li *ListenerItem) *ListenerQueue {
	lq.items = append(lq.items, li)
	return lq
}

// Sort the queue items by ListenerItem's priority.
// Priority: High > Low
func (lq *ListenerQueue) Sort() *ListenerQueue {
	// if lq.IsEmpty() {
	// 	return lq
	// }
	ls := ByPriorityItems(lq.items)

	// check items is sorted
	if !sort.IsSorted(ls) {
		sort.Sort(ls)
	}

	return lq
}

// Items get all ListenerItem
func (lq *ListenerQueue) Items() []*ListenerItem {
	return lq.items
}

// Remove a listener from the queue
func (lq *ListenerQueue) Remove(listener IListener) {
	if listener == nil {
		return
	}

	ptrVal := fmt.Sprintf("%p", listener)

	var newItems []*ListenerItem
	for _, li := range lq.items {
		liPtrVal := fmt.Sprintf("%p", li.Listener)
		if liPtrVal == ptrVal {
			continue
		}

		newItems = append(newItems, li)
	}

	lq.items = newItems
}

// Clear all listeners of ListenerQueue
// use re-slice to clear up the slice
func (lq *ListenerQueue) Clear() {
	lq.items = lq.items[:0]
}

// ByPriorityItems type.
// implements the sort.Interface
type ByPriorityItems []*ListenerItem

func (ls ByPriorityItems) Len() int {
	return len(ls)
}

func (ls ByPriorityItems) Less(i, j int) bool {
	return ls[i].Priority > ls[j].Priority
}

func (ls ByPriorityItems) Swap(i, j int) {
	ls[i], ls[j] = ls[j], ls[i]
}
