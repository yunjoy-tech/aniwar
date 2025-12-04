package event

// IEvent defined the event interface
type IEvent interface {
	Name() string
	Get(key string) interface{}
	Set(key string, val interface{})
	Data() map[string]interface{}
	SetData(M) IEvent
	Abort(bool)
	IsAborted() bool
	Type() []int32
}

// BasicEvent define a basic event struct
type BasicEvent struct {
	name    string
	eType   []int32 // special the event type
	data    map[string]interface{}
	aborted bool // mark is aborted
}

// SetName set event name
func (e *BasicEvent) SetName(name string) *BasicEvent {
	e.name = name
	return e
}

// Name get event name
func (e *BasicEvent) Name() string {
	return e.name
}

// Get data by key
func (e *BasicEvent) Get(key string) interface{} {
	if v, ok := e.data[key]; ok {
		return v
	}

	return nil
}

// Set value by key
func (e *BasicEvent) Set(key string, val interface{}) {
	if e.data == nil {
		e.data = make(map[string]interface{})
	}

	e.data[key] = val
}

// Data get all data
func (e *BasicEvent) Data() map[string]interface{} {
	return e.data
}

// SetData set data to the event
func (e *BasicEvent) SetData(data M) IEvent {
	if data != nil {
		e.data = data
	}
	return e
}

// Abort event loop exec
func (e *BasicEvent) Abort(abort bool) {
	e.aborted = abort
}

// IsAborted check.
func (e *BasicEvent) IsAborted() bool {
	return e.aborted
}

// Type get event type
func (e *BasicEvent) Type() []int32 {
	return e.eType
}
