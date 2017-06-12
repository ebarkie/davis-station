// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

// HTTP server for accessing weather station data.

import (
	"encoding/json"
	"fmt"
	"net/http"
	//_ "net/http/pprof"
	"strconv"
	"time"
)

type httpContext serverContext

type httpLogWrapper struct {
	http.CloseNotifier
	http.Flusher
	http.ResponseWriter
	status int
}

func (l *httpLogWrapper) Write(p []byte) (int, error) {
	return l.ResponseWriter.Write(p)
}

func (l *httpLogWrapper) WriteHeader(status int) {
	l.status = status
	l.ResponseWriter.WriteHeader(status)
}

func (httpContext) logHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record := &httpLogWrapper{
			CloseNotifier:  w.(http.CloseNotifier),
			Flusher:        w.(http.Flusher),
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		h.ServeHTTP(record, r)

		msg := fmt.Sprintf("HTTP connection from %s request %s %s response %d", r.RemoteAddr, r.Method, r.URL, record.status)
		if record.status < 299 {
			Debug.Print(msg)
		} else {
			Warn.Print(msg)
		}
	}
}

// archive is the endpoint for serving out archive records.
// GET /archive[?begin=2016-08-03T00:00:00Z][&end=2016-09-03T00:00:00Z]
func (c httpContext) archive(w http.ResponseWriter, r *http.Request) {
	// Parse and validate begin and end parameters.
	var begin, end time.Time
	var err error

	if r.URL.Query().Get("end") != "" {
		end, err = time.Parse(time.RFC3339, r.URL.Query().Get("end"))
		if err != nil {
			w.Header().Set("Warning", "Unable to parse end timestamp")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	} else {
		// Default end is now.
		end = time.Now()
	}

	if r.URL.Query().Get("begin") != "" {
		begin, err = time.Parse(time.RFC3339, r.URL.Query().Get("begin"))
		if err != nil {
			w.Header().Set("Warning", "Unable to parse begin timestamp")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	} else {
		// Default begin is 1 day before end.
		begin = end.AddDate(0, 0, -1)
	}

	if end.Before(begin) {
		w.Header().Set("Warning", "End timestamp precedes begin timestamp")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Large durations can be very resource intensive to marshal so
	// cap at 30 days.
	if end.Sub(begin) > (30 * (24 * time.Hour)) {
		w.Header().Set("Warning", "Duration exceeds maximum allowed")
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}

	// Query archive from database and return.
	archive := c.ad.Get(begin, end)
	if len(archive) < 1 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	j, _ := json.MarshalIndent(archive, "", "    ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(j)
}

// loop is the endpoint for serving out loop samples.
// GET /loop[?lastSequence=#]
func (c httpContext) loop(w http.ResponseWriter, r *http.Request) {
	c.ld.RLock()
	defer c.ld.RUnlock()

	// If there aren't enough samples (the server just started) or
	// there were no recent updates then send a HTTP service temporarily
	// unavailable response.
	if len(c.ld.loops) < loopsMin {
		w.Header().Set("Warning", "Not enough samples yet")
		w.WriteHeader(http.StatusServiceUnavailable)
	} else if time.Since(c.ld.loops[0].Update.Timestamp) > loopStaleAge {
		w.Header().Set("Warning", "Samples are too old")
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		// Figure out if request is for loops since a sequence or just for
		// most recent loop.
		var j []byte
		if r.URL.Query().Get("lastSequence") != "" {
			seq, _ := strconv.ParseInt(r.URL.Query().Get("lastSequence"), 10, 64)

			// There are no sequence gaps so it's simple subtraction to
			// determine the end index.  A few safeguards have to be added
			// though:
			//
			// If the requested sequence is ahead of the server then return
			// nothing.
			//
			// If the request sequence is so far back that it's been purged
			// then return everything.
			endIndex := int(c.ld.loops[0].Update.Sequence - seq)
			if endIndex < 1 {
				j, _ = json.Marshal(nil)
			} else {
				if endIndex > len(c.ld.loops) {
					endIndex = len(c.ld.loops)
				}
				j, _ = json.MarshalIndent(c.ld.loops[0:endIndex], "", "    ")
			}
		} else {
			j, _ = json.MarshalIndent(c.ld.loops[0], "", "    ")
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(j)
	}
}

// events is the endpoint for streaming loop samples using the Server-sent
// events.
// GET /events
func (c httpContext) events(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	incomingEvents := c.eb.subscribe(r.RemoteAddr)
	defer c.eb.unsubscribe(incomingEvents)

	for {
		select {
		case <-w.(http.CloseNotifier).CloseNotify():
			// Client closed the connection
			return
		case e := <-incomingEvents:
			fmt.Fprintf(w, "event: %s\n", e.event)
			r, _ := json.Marshal(e.data)
			fmt.Fprintf(w, "data: %s\n\n", r)
			w.(http.Flusher).Flush()
		}
	}
}

// httpServer starts the HTTP server.  It's blocking and should be called as
// a goroutine.
func httpServer(bindAddress string, sc serverContext) {
	c := httpContext(sc)
	http.HandleFunc("/archive", c.archive)
	http.HandleFunc("/loop", c.loop)
	http.HandleFunc("/events", c.events)

	s := http.Server{
		Addr:    bindAddress + ":8080",
		Handler: c.logHandler(http.DefaultServeMux),
	}
	Info.Printf("HTTP server started on %s", s.Addr)
	err := s.ListenAndServe()
	if err != nil {
		Error.Fatalf("HTTP server error: %s", err.Error())
	}
}
