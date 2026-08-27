package client

type ActionFunc = func()

type ActionStack []ActionFunc

func (s *ActionStack) IsEmpty() bool {
	return len(*s) == 0
}

func (s *ActionStack) Push(f ActionFunc) {
	*s = append(*s, f)
}

func (s *ActionStack) Pop() (ActionFunc, bool) {
	if s.IsEmpty() {
		return nil, false
	} else {
		index := len(*s) - 1
		element := (*s)[index]
		*s = (*s)[:index]
		return element, true
	}
}

func (s *ActionStack) Clear() {
	*s = (*s)[:0]
}
