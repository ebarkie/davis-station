// Copyright (c) 2016 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package events is a simple events broker supporting subscribe,
// unsubscribe, and publish operations.
package events

// Broker is an event broker instance.
type Broker struct {
	events chan Event            // New events that will be broacast to all subscribers
	subs   map[chan Event]string // Subscriber map as [channel] = address:port
	sub    chan sub              // Subscribe requests pending add to map
	unsub  chan sub              // Unsubscribe requests pending removal from map
}

// Event represents an event that occurred.
type Event struct {
	Name string      // "archive", "loop", ...
	Data interface{} // weatherlink.Archive, loop, ...
}

type sub struct {
	name   string     // Subscriber name in the form address:port
	events chan Event // Subscriber event channel
}

// New creates a new event broker instance and launches the server
// that handles all incoming events and subscription requests.
func New() *Broker {
	b := &Broker{
		events: make(chan Event, 8),
		subs:   make(map[chan Event]string),
		sub:    make(chan sub),
		unsub:  make(chan sub),
	}

	go func() {
		for {
			select {
			case c := <-b.sub: // Subscribe requests
				b.subs[c.events] = c.name
			case c := <-b.unsub: // Unsubscribe requests
				delete(b.subs, c.events)
			case e := <-b.events: // Published events
				for c := range b.subs {
					select {
					case c <- e:
					default:
						// If a subscribers io.Writer can't flush for
						// approximately 15 minutes will begin to block.
					}
				}
			}
		}
	}()

	return b
}

// Subscribe registers a client to receive events.
func (b Broker) Subscribe(name string) chan Event {
	c := make(chan Event)
	b.sub <- sub{name: name, events: c}
	return c
}

// Unsubscribe removes a client that was previously receiving events.
func (b Broker) Unsubscribe(c chan Event) {
	b.unsub <- sub{events: c}
}

// Publish sends a new event to subscribers.
func (b Broker) Publish(e Event) {
	b.events <- e
}
