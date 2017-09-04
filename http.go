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

type httpCtx serverCtx

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

func (httpCtx) logHandler(h http.Handler) http.HandlerFunc {
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
			Debug.Println(msg)
		} else {
			Warn.Println(msg)
		}
	}
}

// archive is the endpoint for serving out archive records.
// GET /archive[?begin=2016-08-03T00:00:00Z][&end=2016-09-03T00:00:00Z]
func (c httpCtx) archive(w http.ResponseWriter, r *http.Request) {
	// Parse and validate begin and end parameters
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

	// Query archive from database and return
	archive := c.ad.Get(begin, end)
	if len(archive) < 1 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(archive)
}

// loop is the endpoint for serving out loop samples.
// GET /loop[?lastSequence=#]
func (c httpCtx) loop(w http.ResponseWriter, r *http.Request) {
	numLoops, lastLoop := c.lb.Last()

	// If there aren't enough samples (the server just started) or
	// there were no recent updates then send a HTTP service temporarily
	// unavailable response.
	if numLoops < loopsMin {
		w.Header().Set("Warning", ErrLoopsMin.Error())
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	if time.Since(lastLoop.Timestamp) > loopStaleAge {
		w.Header().Set("Warning", ErrLoopsAge.Error())
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	// Figure out if request is for loops since a sequence or just for
	// most recent loop.
	j := json.NewEncoder(w)
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
		endIndex := int(lastLoop.Seq - seq)
		if endIndex < 1 {
			w.WriteHeader(http.StatusNoContent)
		} else {
			if endIndex > numLoops {
				endIndex = numLoops
			}
			w.Header().Set("Content-Type", "application/json")
			j.Encode(c.lb.Loops()[0:endIndex])
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		j.Encode(lastLoop)
	}
}

// events is the endpoint for streaming loop samples using the Server-sent
// events.
// GET /events
func (c httpCtx) events(w http.ResponseWriter, r *http.Request) {
	// See Server-sent-event specification:
	// https://en.wikipedia.org/wiki/Server-sent_events

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	inEvents := c.eb.Subscribe(r.RemoteAddr)
	defer c.eb.Unsubscribe(inEvents)

	for {
		select {
		case <-w.(http.CloseNotifier).CloseNotify():
			// Client closed the connection
			return
		case e := <-inEvents:
			fmt.Fprintf(w, "event: %s\n", e.Event)
			r, _ := json.Marshal(e.Data)
			fmt.Fprintf(w, "data: %s\n\n", r)
			w.(http.Flusher).Flush()
		}
	}
}

// httpServer starts the HTTP server.  It's blocking and should be called as
// a goroutine.
func httpServer(sc serverCtx, bindAddr string) {
	// Inherit generic server context so we have access to things like
	// archive records and loop packets.
	c := httpCtx(sc)

	// Register routes
	http.HandleFunc("/archive", c.archive)
	http.HandleFunc("/loop", c.loop)
	http.HandleFunc("/events", c.events)

	// Listen and accept new connections
	s := http.Server{
		Addr:    bindAddr + ":8080",
		Handler: c.logHandler(http.DefaultServeMux),
	}
	Info.Printf("HTTP server started on %s", s.Addr)
	err := s.ListenAndServe()
	if err != nil {
		Error.Fatalf("HTTP server error: %s", err.Error())
	}
}
