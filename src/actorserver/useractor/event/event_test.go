package event_test

import (
	"bytes"
	"fmt"
	"gitlab.musadisca-games.com/wangxw/aniwar/src/actorserver/useractor/event"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var emptyListener = func(e event.IEvent) error {
	return nil
}

type testListener struct {
	userData string
}

func (l *testListener) Handle(e event.IEvent) error {
	if ret := e.Get("result"); ret != nil {
		str := ret.(string) + fmt.Sprintf(" -> %s(%s)", e.Name(), l.userData)
		e.Set("result", str)
	} else {
		e.Set("result", fmt.Sprintf("handled: %s(%s)", e.Name(), l.userData))
	}
	return nil
}

func TestEvent(t *testing.T) {
	e := &event.BasicEvent{}
	e.SetName("n1")
	e.SetData(event.M{
		"arg0": "val0",
	})

	e.Set("arg1", "val1")

	assert.False(t, e.IsAborted())
	e.Abort(true)
	assert.True(t, e.IsAborted())

	assert.Equal(t, "n1", e.Name())
	assert.Contains(t, e.Data(), "arg1")
	assert.Equal(t, "val0", e.Get("arg0"))
	assert.Equal(t, nil, e.Get("not-exist"))

	e.Set("arg1", "new val")
	assert.Equal(t, "new val", e.Get("arg1"))

	e1 := &event.BasicEvent{}
	e1.Set("k", "v")
	assert.Equal(t, "v", e1.Get("k"))
}

func TestAddEvent(t *testing.T) {
	defer event.GlobalManager.Reset()
	event.GlobalManager.RemoveEvents()

	// no name
	assert.Panics(t, func() {
		event.GlobalManager.AddEvent(&event.BasicEvent{})
	})

	_, ok := event.GlobalManager.GetEvent("evt1")
	assert.False(t, ok)

	// AddEvent
	e := event.NewBasicEvent("evt1", []int32{}, event.M{"k1": "inhere"})
	event.GlobalManager.AddEvent(e)
	// add by AttachTo
	event.GlobalManager.AddEvent(event.NewBasicEvent("evt2", []int32{}, nil))

	assert.False(t, e.IsAborted())
	assert.True(t, event.GlobalManager.HasEvent("evt1"))
	assert.True(t, event.GlobalManager.HasEvent("evt2"))
	assert.False(t, event.GlobalManager.HasEvent("not-exist"))

	// GetEvent
	r1, ok := event.GlobalManager.GetEvent("evt1")
	assert.True(t, ok)
	assert.Equal(t, e, r1)

	// RemoveEvent
	event.GlobalManager.RemoveEvent("evt2")
	assert.False(t, event.GlobalManager.HasEvent("evt2"))

	// RemoveEvents
	event.GlobalManager.RemoveEvents()
	assert.False(t, event.GlobalManager.HasEvent("evt1"))
}

func TestListen(t *testing.T) {
	defer event.GlobalManager.Reset()

	assert.Panics(t, func() {
		event.GlobalManager.Listen("", event.ListenerFunc(emptyListener), 0)
	})
	assert.Panics(t, func() {
		event.GlobalManager.Listen("name", nil, 0)
	})
	assert.Panics(t, func() {
		event.GlobalManager.Listen("++df", event.ListenerFunc(emptyListener), 0)
	})

	event.GlobalManager.Listen("n1", event.ListenerFunc(emptyListener), event.Min)
	assert.Equal(t, 1, event.GlobalManager.ListenersCount("n1"))
	assert.Equal(t, 0, event.GlobalManager.ListenersCount("not-exist"))
	assert.True(t, event.GlobalManager.HasListeners("n1"))
	assert.False(t, event.GlobalManager.HasListeners("name"))

	assert.NotEmpty(t, event.GlobalManager.Listeners())
	assert.NotEmpty(t, event.GlobalManager.ListenersByName("n1"))

	event.GlobalManager.RemoveListeners("n1")
	assert.False(t, event.GlobalManager.HasListeners("n1"))
}

func TestPublish(t *testing.T) {
	buf := new(bytes.Buffer)
	fn := func(e event.IEvent) error {
		_, _ = fmt.Fprintf(buf, "event: %s", e.Name())
		return nil
	}

	event.GlobalManager.Listen("evt1", event.ListenerFunc(fn), 0)
	event.GlobalManager.Listen("evt1", event.ListenerFunc(emptyListener), event.High)
	assert.True(t, event.GlobalManager.HasListeners("evt1"))

	err, e := event.GlobalManager.Publish("evt1", nil)
	assert.NoError(t, err)
	assert.Equal(t, "evt1", e.Name())
	assert.Equal(t, "event: evt1", buf.String())

	event.GlobalManager.AddEvent(event.NewBasicEvent("evt2", []int32{}, nil))
	event.GlobalManager.Listen("evt2", event.ListenerFunc(func(e event.IEvent) error {
		assert.Equal(t, "evt2", e.Name())
		assert.Equal(t, "v", e.Get("k"))
		return nil
	}), event.AboveNormal)

	assert.True(t, event.GlobalManager.HasListeners("evt2"))
	err, e = event.GlobalManager.Publish("evt2", event.M{"k": "v"})
	assert.NoError(t, err)
	assert.Equal(t, "evt2", e.Name())
	assert.Equal(t, map[string]interface{}{"k": "v"}, e.Data())

	// clear all
	event.GlobalManager.Reset()
	assert.False(t, event.GlobalManager.HasListeners("evt1"))
	assert.False(t, event.GlobalManager.HasListeners("evt2"))

	err, e = event.GlobalManager.Publish("not-exist", nil)
	assert.NoError(t, err)
	assert.Nil(t, e)
}

func TestMustPublish(t *testing.T) {
	defer event.GlobalManager.Reset()

	event.GlobalManager.Listen("n1", event.ListenerFunc(func(e event.IEvent) error {
		return fmt.Errorf("an error")
	}), event.Max)
	event.GlobalManager.Listen("n2", event.ListenerFunc(emptyListener), event.Min)

	assert.Panics(t, func() {
		_ = event.GlobalManager.MustPublish("n1", nil)
	})

	assert.NotPanics(t, func() {
		_ = event.GlobalManager.MustPublish("n2", nil)
	})
}

func TestManager_Publish_WithWildcard(t *testing.T) {
	buf := new(bytes.Buffer)
	mgr := event.NewManager("test")

	const Event2FurcasTicketCreate = "kapal.furcas.ticket.create"

	handler := event.ListenerFunc(func(e event.IEvent) error {
		_, _ = fmt.Fprintf(buf, "%s-%s|", e.Name(), e.Get("user"))
		return nil
	})

	mgr.Listen("kapal.furcas.ticket.*", handler)
	mgr.Listen(Event2FurcasTicketCreate, handler)

	err, _ := mgr.Publish(Event2FurcasTicketCreate, event.M{"user": "inhere"})
	assert.NoError(t, err)
	assert.Equal(
		t,
		"kapal.furcas.ticket.create-inhere|kapal.furcas.ticket.create-inhere|",
		buf.String(),
	)
	buf.Reset()

	// add Wildcard listen
	mgr.Listen("*", handler)

	err, _ = mgr.Publish(Event2FurcasTicketCreate, event.M{"user": "inhere"})
	assert.NoError(t, err)
	assert.Equal(
		t,
		"kapal.furcas.ticket.create-inhere|kapal.furcas.ticket.create-inhere|kapal.furcas.ticket.create-inhere|",
		buf.String(),
	)
}

func TestListenGroupEvent(t *testing.T) {
	em := event.NewManager("test")

	e1 := event.NewBasicEvent("app.evt1", []int32{}, event.M{"buf": new(bytes.Buffer)})
	em.AddEvent(e1)

	l2 := event.ListenerFunc(func(e event.IEvent) error {
		e.Get("buf").(*bytes.Buffer).WriteString(" > 2 " + e.Name())
		return nil
	})
	l3 := event.ListenerFunc(func(e event.IEvent) error {
		e.Get("buf").(*bytes.Buffer).WriteString(" > 3 " + e.Name())
		return nil
	})

	em.Listen("app.evt1", event.ListenerFunc(func(e event.IEvent) error {
		e.Get("buf").(*bytes.Buffer).WriteString("Hi > 1 " + e.Name())
		return nil
	}))
	em.Listen("app.*", l2)
	em.Listen("*", l3)

	buf := e1.Get("buf").(*bytes.Buffer)
	err, e := em.Publish("app.evt1", nil)
	assert.NoError(t, err)
	assert.Equal(t, e1, e)
	assert.Equal(t, "Hi > 1 app.evt1 > 2 app.evt1 > 3 app.evt1", buf.String())

	em.RemoveListener("app.*", l2)
	assert.Len(t, em.ListenedNames(), 2)
	em.Listen("app.*", event.ListenerFunc(func(e event.IEvent) error {
		return fmt.Errorf("an error")
	}))

	buf.Reset()
	err, e = em.Publish("app.evt1", nil)
	assert.Error(t, err)
	assert.Equal(t, "Hi > 1 app.evt1", buf.String())

	em.RemoveListeners("app.*")
	em.RemoveListener("", l3)
	em.Listen("app.*", l2) // re-add
	em.Listen("*", event.ListenerFunc(func(e event.IEvent) error {
		return fmt.Errorf("an error")
	}))
	assert.Len(t, em.ListenedNames(), 3)

	buf.Reset()
	err, e = em.Publish("app.evt1", nil)
	assert.Error(t, err)
	assert.Equal(t, e1, e)
	assert.Equal(t, "Hi > 1 app.evt1 > 2 app.evt1", buf.String())

	em.RemoveListener("", nil)

	// clear
	em.Reset()
	buf.Reset()
}

func TestManager_AsyncPublish(t *testing.T) {
	em := event.NewManager("test")
	em.Listen("e1", event.ListenerFunc(func(e event.IEvent) error {
		assert.Equal(t, map[string]interface{}{"k": "v"}, e.Data())
		e.Set("nk", "nv")
		return nil
	}))

	e1 := event.NewBasicEvent("e1", []int32{}, event.M{"k": "v"})
	em.AsyncPublish(e1)
	time.Sleep(time.Second / 10)
	assert.Equal(t, "nv", e1.Get("nk"))

	var wg sync.WaitGroup
	em.Listen("e2", event.ListenerFunc(func(e event.IEvent) error {
		defer wg.Done()
		assert.Equal(t, "v", e.Get("k"))
		return nil
	}))

	wg.Add(1)
	em.AsyncPublish(e1.SetName("e2"))
	wg.Wait()

	em.Reset()
}

func TestManager_AwaitPublish(t *testing.T) {
	em := event.NewManager("test")
	em.Listen("e1", event.ListenerFunc(func(e event.IEvent) error {
		assert.Equal(t, map[string]interface{}{"k": "v"}, e.Data())
		e.Set("nk", "nv")
		return nil
	}))

	e1 := event.NewBasicEvent("e1", []int32{}, event.M{"k": "v"})
	err := em.AwaitPublish(e1)

	assert.NoError(t, err)
	assert.Contains(t, e1.Data(), "nk")
	assert.Equal(t, "nv", e1.Get("nk"))
}

type testSubscriber struct {
	// ooo
}

func (s *testSubscriber) SubscribedEvents() map[string]interface{} {
	return map[string]interface{}{
		"e1": event.ListenerFunc(s.e1Handler),
		"e2": event.ListenerItem{
			Priority: event.AboveNormal,
			Listener: event.ListenerFunc(func(e event.IEvent) error {
				return fmt.Errorf("an error")
			}),
		},
		"e3": &testListener{},
	}
}

func (s *testSubscriber) e1Handler(e event.IEvent) error {
	e.Set("e1-key", "val1")
	return nil
}

type testSubscriber2 struct{}

func (s testSubscriber2) SubscribedEvents() map[string]interface{} {
	return map[string]interface{}{
		"e1": "invalid",
	}
}

func TestManager_Subscribe(t *testing.T) {
	em := event.NewManager("test")
	em.Subscribe(&testSubscriber{})

	assert.True(t, em.HasListeners("e1"))
	assert.True(t, em.HasListeners("e2"))
	assert.True(t, em.HasListeners("e3"))

	ers := em.BatchPublish("e1", event.NewBasicEvent("e2", []int32{}, nil))
	assert.Len(t, ers, 1)

	assert.Panics(t, func() {
		em.Subscribe(testSubscriber2{})
	})

	em.Reset()
}
