package event

import (
	"gitee.com/bychannel/musae/framework/logger"
	"strings"
	"sync"
)

// IManager event manager interface
type IManager interface {
	AddEvent(IEvent)
	Listen(name string, listener IListener, priority ...int)
	Publish(name string, params M) (error, IEvent)
}

// Manager definition event manager. for manage events and listeners
type Manager struct {
	sync.Mutex
	// enable lock on publish event.
	EnableLock bool
	// name of the manager
	name string
	// it's a sample for new BasicEvent
	sample *BasicEvent
	// storage user custom IEvent instance. you can pre-define some IEvent instances.
	events map[string]IEvent
	// key is the event name, value is the queue of listener func
	listeners map[string]*ListenerQueue
	// storage all event names by listened
	listenedNames map[string]int
	// async publish event use channel
	eventCh chan IEvent
}

// NewManager create event manager
func NewManager(name string) *Manager {
	em := &Manager{
		name:          name,
		sample:        &BasicEvent{},
		events:        make(map[string]IEvent),
		listeners:     make(map[string]*ListenerQueue),
		listenedNames: make(map[string]int),
		eventCh:       make(chan IEvent, 10), // 涉及到非玩家手动的调用需求，加一点缓存
	}

	// 废弃，改成同步
	// threading.GoSafe(func() {
	//	func(em *Manager) {
	//		for {
	//			select {
	//			case e := <-em.eventCh:
	//				threading.RunSafeWithParam(func(param interface{}) {
	//					err := em.publish(e)
	//					if err != nil {
	//						logger.Error(err)
	//					}
	//				}, e)
	//			}
	//		}
	//	}(em)
	// })

	return em
}

// Listen register an event handler/listener with priority.
// if not, default level is NORMAL
func (em *Manager) Listen(name string, listener IListener, priority ...int) {
	pv := Normal
	if len(priority) > 0 {
		pv = priority[0]
	}

	em.addListenerItem(name, &ListenerItem{pv, listener})
}

// Subscribe add events by ISubscriber interface.
// you can register multi event listeners in a struct func.
func (em *Manager) Subscribe(s ISubscriber) {
	for name, listener := range s.SubscribedEvents() {
		switch lt := listener.(type) {
		case IListener:
			em.Listen(name, lt)
		case ListenerItem:
			em.addListenerItem(name, &lt)
		default:
			panic("event: the value must be an IListener or ListenerItem instance")
		}
	}
}

func (em *Manager) addListenerItem(name string, li *ListenerItem) {
	if name != Wildcard {
		name = checkName(name)
	}

	if li.Listener == nil {
		panic("event: the event '" + name + "' listener cannot be empty")
	}

	// if exists, append it.
	if lq, ok := em.listeners[name]; ok {
		lq.Push(li)
	} else {
		// first add.
		em.listenedNames[name] = 1
		em.listeners[name] = (&ListenerQueue{}).Push(li)
	}
}

// Publish event by name. if not found listener, will return (nil, nil)
func (em *Manager) Publish(name string, params M) (err error, e IEvent) {
	name = checkName(name)

	// must check the '*' global listeners
	if false == em.HasListeners(name) && false == em.HasListeners(Wildcard) {
		// has group listeners. "app.*" "aa.bb.*"
		// eg: "aa.bb.cc" will trigger listeners on the "aa.bb.*"
		pos := strings.LastIndexByte(name, '.')
		if pos < 0 || pos == len(name)-1 {
			return // not found listeners.
		}

		groupName := name[:pos+1] + Wildcard // "aa.bb.*"
		if false == em.HasListeners(groupName) {
			return // not found listeners.
		}
	}

	// call listeners use defined IEvent
	if e, ok := em.events[name]; ok {
		if params != nil {
			e.SetData(params)
		}

		err = em.publish(e)
		return err, e
	}

	// create a basic event instance
	e = em.copyBasicEvent(name, params)
	// call listeners handle event
	err = em.publish(e)
	return
}

func (em *Manager) SyncPublish(e IEvent) error {
	logger.Debugf("event manager sync publish success, event:%+v", e)
	return em.publish(e)
}

// MustPublish event by name. will panic on error
func (em *Manager) MustPublish(name string, params M) IEvent {
	err, e := em.Publish(name, params)
	if err != nil {
		panic(err)
	}
	return e
}

// AsyncPublish async publish event by 'go' keywords
func (em *Manager) AsyncPublish(e IEvent) {
	// go func(e IEvent) {
	//	_ = em.publish(e)
	// }(e)

	// 优化: 启动channel进行事件发布，启动单独的goroutine进行处理
	em.eventCh <- e
}

// AwaitPublish async publish event by 'go' keywords, but will wait return result
func (em *Manager) AwaitPublish(e IEvent) (err error) {
	ch := make(chan error)

	go func(e IEvent) {
		err := em.publish(e)
		ch <- err
	}(e)

	err = <-ch
	close(ch)
	return
}

// BatchPublish publish multi event at once.
// eg. BatchPublish("name1", "name2", &MyEvent{})
func (em *Manager) BatchPublish(es ...interface{}) (ers []error) {
	var err error
	for _, e := range es {
		if name, ok := e.(string); ok {
			err, _ = em.Publish(name, nil)
		} else if evt, ok := e.(IEvent); ok {
			err = em.publish(evt)
		}

		if err != nil {
			ers = append(ers, err)
		}
	}
	return
}

func (em *Manager) publish(e IEvent) (err error) {

	if em.EnableLock {
		em.Lock()
		defer em.Unlock()
	}

	e.Abort(false)
	name := e.Name()

	// find matched listeners
	lq, ok := em.listeners[name]
	if ok {
		// sort by priority before call.
		for _, li := range lq.Sort().Items() {
			err = li.Listener.Handle(e)
			if err != nil && e.IsAborted() {
				return
			}
		}
	}

	// has group listeners.
	pos := strings.LastIndexByte(name, '.')
	if pos > 0 && pos < len(name) {
		groupName := name[:pos+1] + Wildcard

		if lq, ok := em.listeners[groupName]; ok {
			for _, li := range lq.Sort().Items() {
				err = li.Listener.Handle(e)
				if err != nil && e.IsAborted() {
					return
				}
			}
		}
	}

	// has wildcard event listeners
	if lq, ok := em.listeners[Wildcard]; ok {
		for _, li := range lq.Sort().Items() {
			err = li.Listener.Handle(e)
			if err != nil && e.IsAborted() {
				break
			}
		}
	}
	return
}

// AddEvent add a defined event instance to manager.
func (em *Manager) AddEvent(e IEvent) {
	name := checkName(e.Name())
	em.events[name] = e
}

// GetEvent get a defined event instance by name
func (em *Manager) GetEvent(name string) (e IEvent, ok bool) {
	e, ok = em.events[name]
	return
}

// HasEvent has event check
func (em *Manager) HasEvent(name string) bool {
	_, ok := em.events[name]
	return ok
}

// RemoveEvent delete IEvent by name
func (em *Manager) RemoveEvent(name string) {
	if _, ok := em.events[name]; ok {
		delete(em.events, name)
	}
}

// RemoveEvents remove all registered events
func (em *Manager) RemoveEvents() {
	em.events = map[string]IEvent{}
}

// copyBasicEvent create new BasicEvent by clone em.sample
func (em *Manager) copyBasicEvent(name string, data M) *BasicEvent {
	var cp = *em.sample

	cp.SetName(name)
	cp.SetData(data)
	return &cp
}

// HasListeners has listeners for the event name.
func (em *Manager) HasListeners(name string) bool {
	_, ok := em.listenedNames[name]
	return ok
}

// Listeners get all listeners
func (em *Manager) Listeners() map[string]*ListenerQueue {
	return em.listeners
}

// ListenersByName get listeners by given event name
func (em *Manager) ListenersByName(name string) *ListenerQueue {
	return em.listeners[name]
}

// ListenersCount get listeners number for the event name.
func (em *Manager) ListenersCount(name string) int {
	if lq, ok := em.listeners[name]; ok {
		return lq.Len()
	}
	return 0
}

// ListenedNames get listened event names
func (em *Manager) ListenedNames() map[string]int {
	return em.listenedNames
}

// RemoveListener remove a given listener, you can limit event name.
//
// Usage:
//
//	RemoveListener("", listener)
//	RemoveListener("name", listener) // limit event name.
func (em *Manager) RemoveListener(name string, listener IListener) {
	if name != "" {
		if lq, ok := em.listeners[name]; ok {
			lq.Remove(listener)

			// delete from manager
			if lq.IsEmpty() {
				delete(em.listeners, name)
				delete(em.listenedNames, name)
			}
		}
		return
	}

	// name is empty. find all listener and remove matched.
	for name, lq := range em.listeners {
		lq.Remove(listener)

		// delete from manager
		if lq.IsEmpty() {
			delete(em.listeners, name)
			delete(em.listenedNames, name)
		}
	}
}

// RemoveListeners remove listeners by given name
func (em *Manager) RemoveListeners(name string) {
	_, ok := em.listenedNames[name]
	if ok {
		em.listeners[name].Clear()

		// delete from manager
		delete(em.listeners, name)
		delete(em.listenedNames, name)
	}
}

// Reset the manager, clear all data.
func (em *Manager) Reset() {
	// clear all listeners
	for _, lq := range em.listeners {
		lq.Clear()
	}

	// reset all
	em.name = ""
	em.events = make(map[string]IEvent)
	em.listeners = make(map[string]*ListenerQueue)
	em.listenedNames = make(map[string]int)
}
