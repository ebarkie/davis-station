// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package textcmd

import (
	"io"
	"net"
)

// Ctx is the command context that is passed to a function.
type Ctx struct {
	conn    net.Conn
	matches []string
}

// Arg returns the argument at index i.  0 is the entire
// command, 1 is the first argument, 2 is the second, etc.  If an
// argument does not exist an empty string is returned.
func (c Ctx) Arg(i int) (s string) {
	if len(c.matches) > i {
		s = c.matches[i]
	}

	return
}

// LocalAddr returns the local network address.
func (c Ctx) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// Reader return an interface that wraps the basic Read method.
func (c Ctx) Reader() io.Reader {
	return c.conn
}

// ReadWriter returns an interface that groups the basic Read and Write methods.
func (c Ctx) ReadWriter() io.ReadWriter {
	return c.conn
}

// RemoteAddr returns the remote network address.
func (c Ctx) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// Writer returns an interface that wraps the basic Write method.
func (c Ctx) Writer() io.Writer {
	return c.conn
}
