// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

// See Server-sent-event specification:
// https://en.wikipedia.org/wiki/Server-sent_events

type eventBroker struct {
	events chan event            // New events that will be broacast to all SSE clients
	subs   map[chan event]string // SSE subscriber map as [channel] = address:port
	sub    chan eventSub         // Subscribe requests pending add to map
	unsub  chan eventSub         // Unsubscribe requests pending removal from map
}

type eventSub struct {
	name   string     // Subscriber name in the form address:port
	events chan event // Subscriber event channel
}

type event struct {
	event string      // "archive" or "loop"
	data  interface{} // Either weatherlink.Archive or WrappedLoop
}

// subscribe registers a client to receive SSE events.
func (eb eventBroker) subscribe(name string) chan event {
	c := make(chan event)
	eb.sub <- eventSub{name: name, events: c}
	return c
}

// unsubscribe removes a client that was previously receiving SSE
// events.
func (eb eventBroker) unsubscribe(c chan event) {
	eb.unsub <- eventSub{events: c}
}

// publish sends a new SSE event to subscribers.
func (eb eventBroker) publish(e event) {
	eb.events <- e
}

// Server-sent events broker.  This waits for new loop events and
// broadcasts them to each channel in the subscribers map.
func eventsBroker(eb *eventBroker) {
	for {
		select {
		case c := <-eb.sub:
			eb.subs[c.events] = c.name
			Debug.Printf("HTTP-SSE connection from %s opened", c.name)
		case c := <-eb.unsub:
			Debug.Printf("HTTP-SSE connection from %s closed", eb.subs[c.events])
			delete(eb.subs, c.events)
		case e := <-eb.events:
			for c, name := range eb.subs {
				// If a subscribers io.Writer can't flush for approximately 15 minutes it
				// will begin to block.
				select {
				case c <- e:
				default:
					Warn.Printf("HTTP-SEE connection from %s is dropping events", name)
				}
			}
		}
	}
}
