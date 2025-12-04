package dequeue

// The Node structure simply encapsulates the data and the position within the
// Dequeue.
type Node struct {
	next *Node
	prev *Node
	data interface{}
}

// Next returns the next *Node.
func (n *Node) Next() *Node {
	return n.next
}

// Prev returns the previous *Node.
func (n *Node) Prev() *Node {
	return n.prev
}

// Data returns the contents in this node.
func (n *Node) Data() interface{} {
	return n.data
}

func (n *Node) append(m *Node) *Node {
	m.next = n.next
	m.prev = n
	n.next.prev = m
	n.next = m
	return m
}

func (n *Node) prepend(m *Node) *Node {
	m.prev = n.prev
	m.next = n
	n.prev.next = m
	n.prev = m
	return m
}

func (n *Node) remove() *Node {
	n.next.prev = n.prev
	n.prev.next = n.next
	return n
}

// The Dequeue structure implements a double ended Queue with O(1) performance
// properties. Internally it is a circularly linked list.
type Dequeue struct {
	Node
	len int
}

// New creates a properly initialized Dequeue structure and returns a pointer
// to that structure.
func New() *Dequeue {
	var d = new(Dequeue)
	d.Node.prev = &d.Node
	d.Node.next = &d.Node
	//d.data = nil
	return d
}

// Range executes the given function for each data in the Dequeue in order.
// If the given function returns false, it will stop the iteration.
func (d *Dequeue) Range(f func(dat interface{}) bool) {
	for n := d.First(); n != &d.Node; n = n.Next() {
		if !f(n.data) {
			break
		}
	}
}

// Len returns the number of entries in the Dequeue.
func (d *Dequeue) Len() int {
	return d.len
}

// Empty returns a boolean representing if the Dequeue is empty or not.
func (d *Dequeue) Empty() bool {
	//return d.len == 0
	return d.Node.next == &d.Node
}

// First returns the first *Node in the Dequeue. If the Dequeue is empty
// First() will return nil.
func (d *Dequeue) First() *Node {
	if d.Empty() {
		return nil
	}
	return d.Node.next
}

// Last returns the last *Node in the Dequeue. If the Dequeue is empty
// Last() will return nil.
func (d *Dequeue) Last() *Node {
	if d.Empty() {
		return nil
	}
	return d.Node.prev
}

// Push inserts a *Node containing the data on the end of the Dequeue.
func (d *Dequeue) Push(data interface{}) *Dequeue {
	var n = new(Node)
	n.data = data

	d.Node.prepend(n)
	d.len++

	return d
}

// Pop removes the *Node at the end of the Dequeue and returns the data it
// contains.
func (d *Dequeue) Pop() interface{} {
	if d.Empty() {
		return nil
	}

	var n = d.Last().remove()
	d.len--

	return n.data
}

// Shift removes the *Node at the beginning of the Dequeue and returns the data
// it contains.
func (d *Dequeue) Shift() interface{} {
	if d.Empty() {
		return nil
	}

	var n = d.First().remove()
	d.len--

	return n.data
}

// Unshift inserts a *Node containing the data on the beginning of the Dequeue.
func (d *Dequeue) Unshift(data interface{}) *Dequeue {
	var n = new(Node)
	n.data = data

	d.Node.append(n)
	d.len++

	return d
}
