// Copyright (c) 2016-2017 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package main

// Simple text command executor.

import (
	"errors"
	"io"
	"net"
	"regexp"
)

// Errors.
var (
	ErrCmdNotFound = errors.New("Command not found")
	ErrCmdQuit     = errors.New("Quit command")
)

// cmd is a command definition.
type cmd struct {
	r *regexp.Regexp
	f func(CmdCtx) error
}

// TextCmds sets up a new text command executor.
type TextCmds struct {
	cmds []cmd
}

// Exec attempts to execute the passed string as a command.
func (tc TextCmds) Exec(conn net.Conn, s string) error {
	for _, c := range tc.cmds {
		if matches := c.r.FindStringSubmatch(s); matches != nil {
			return c.f(CmdCtx{conn, matches})
		}
	}

	return ErrCmdNotFound
}

// Register adds a command to the text command executor.  It takes a
// regular expression representing the command and a corresponding
// function to be called.
//
// Arguments should be grouped with ()'s in the regular expression
// so they are accessible through the Arg() method.
//
// Example:
// mycommand ([[:digit:]]*)
//
// The above represents a command with a required integer argument
// which will be accessible as CmdCtx.Arg(1).
func (tc *TextCmds) Register(s string, f func(CmdCtx) error) {
	tc.cmds = append(tc.cmds, cmd{
		r: regexp.MustCompile("(?i)^" + s + "$"),
		f: f})
}

// CmdCtx is the command context that is passed to a function.
type CmdCtx struct {
	conn    net.Conn
	matches []string
}

// Arg returns the argument at index i.  0 is the entire
// command, 1 is the first argument, 2 is the second, etc.  If an
// argument does not exist an empty string is returned.
func (c CmdCtx) Arg(i int) (s string) {
	if len(c.matches) > i {
		s = c.matches[i]
	}

	return
}

// LocalAddr returns the local network address.
func (c CmdCtx) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// Reader return an interface that wraps the basic Read method.
func (c CmdCtx) Reader() io.Reader {
	return c.conn
}

// ReadWriter returns an interface that groups the basic Read and Write methods.
func (c CmdCtx) ReadWriter() io.ReadWriter {
	return c.conn
}

// RemoteAddr returns the remote network address.
func (c CmdCtx) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// Writer returns an interface that wraps the basic Write method.
func (c CmdCtx) Writer() io.Writer {
	return c.conn
}
