/**
 * Copyright 2020 Appvia Ltd <info@appvia.io>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package terminal

import (
	"io"
	"time"

	"github.com/pterm/pterm"
)

type linePrinter struct {
	style pterm.Style
	msg   string
}

type config struct {
	// Writer is where the message will be written to.
	Writer io.Writer

	// The style the output should take on
	Style string

	// Any indent character that should be used
	IndentCharacter string
	Indent          int
}

const (
	HeaderStyle       = "header"
	ErrorStyle        = "error"
	ErrorBoldStyle    = "error-bold"
	WarningStyle      = "warning"
	LogStyle          = "log"
	WarningBoldStyle  = "warning-bold"
	SuccessStyle      = "success"
	SuccessBoldStyle  = "success-bold"
	ErrorIndentChar   = "✕"
	WarningIndentChar = "!"
	SuccessIndentChar = "✓"
	HeaderIndentChar  = "»"
	LogIndentChar     = "|"
)

// Option controls output styling.
type Option func(*config)

func WithIndentChar(char string) Option {
	return func(c *config) {
		c.IndentCharacter = char
	}
}

func WithErrorStyle() Option {
	return func(c *config) {
		c.Style = ErrorStyle
	}
}

func WithErrorBoldStyle() Option {
	return func(c *config) {
		c.Style = ErrorBoldStyle
	}
}

func WithStyle(style string) Option {
	return func(c *config) {
		c.Style = style
	}
}

// WithWriter specifies the writer for the output.
func WithWriter(w io.Writer) Option {
	return func(c *config) { c.Writer = w }
}

func WithIndent(i int) Option {
	return func(c *config) {
		c.Indent = i
	}
}

type NamedValue struct {
	Name  string
	Value interface{}
}

// UI is the primary interface for interacting with a user via the CLI.
//
// Some of the methods on this interface return values that have a lifetime
// such as Status and StepGroup. While these are still active (haven't called
// the close or equivalent method on these values), no other method on the
// UI should be called.
type UI interface {
	// Output outputs a message directly to the terminal. The remaining
	// arguments should be interpolations for the format string. After the
	// interpolations you may add Options.
	Output(msg string, opts ...Option)

	// OutputWriters returns stdout and stderr writers. These are usually
	// but not always TTYs. This is useful for subprocesses, network requests,
	// etc. Note that writing to these is not thread-safe by default so
	// you must take care that there is only ever one writer.
	OutputWriters() (stdout, stderr io.Writer, err error)

	Header(msg string, opts ...Option)

	NamedValues(rows []NamedValue, opts ...Option)

	// StepGroup returns a value that can be used to output individual (possibly
	// parallel) steps that have their own message, status indicator, spinner, and
	// body. No other output mechanism (Output, Input, Status, etc.) may be
	// called until the StepGroup is complete.
	StepGroup() StepGroup
}

type StepGroup interface {
	// Start a step in the output with the arguments making up the initial message
	Add(string) Step
	Done()
}

type Step interface {
	Success(delay time.Duration, a ...interface{})
	Warning(delay time.Duration, a ...interface{})
	Error(delay time.Duration, a ...interface{})
}
