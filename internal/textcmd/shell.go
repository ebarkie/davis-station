// Copyright (c) 2016 Eric Barkie. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package textcmd implements a simple text command shell and
// executor.
package textcmd

import (
	"errors"
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
	f func(Env) error
}

// Shell is a text command shell for which commands can be
// registered and executed.
type Shell struct {
	cmds []cmd
}

// Exec attempts to execute the passed string as a command.
func (sh Shell) Exec(conn net.Conn, s string) error {
	for _, cmd := range sh.cmds {
		if matches := cmd.r.FindStringSubmatch(s); matches != nil {
			return cmd.f(Env{Conn: conn, matches: matches})
		}
	}

	return ErrCmdNotFound
}

// Register adds a command to the text command shell.  It takes a
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
// which will be accessible as Env.Arg(1).
func (sh *Shell) Register(r string, f func(Env) error) {
	sh.cmds = append(sh.cmds, cmd{
		r: regexp.MustCompile("(?i)^" + r + "$"),
		f: f})
}
