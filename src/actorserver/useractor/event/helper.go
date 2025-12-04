package event

import "strings"

// NewBasicEvent new a basic event instance
func NewBasicEvent(name string, eType []int32, data M) *BasicEvent {
	if data == nil {
		data = make(map[string]interface{})
	}

	return &BasicEvent{
		name:  name,
		data:  data,
		eType: eType,
	}
}

func checkName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		panic("event: the event name cannot be empty")
	}

	if !eventNameReg.MatchString(name) {
		panic(`event: the event name is invalid, must match regex '^[a-zA-Z][\w-.]*$'`)
	}

	return name
}
