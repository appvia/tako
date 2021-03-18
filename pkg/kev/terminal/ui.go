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

// linePrinter is used to print a msg using its related style on a line
type linePrinter struct {
	style pterm.Style
	msg   string
}

// config is used configure how messages are printed.
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
	HeaderStyle              = "header"
	ErrorStyle               = "error"
	ErrorBoldStyle           = "error-bold"
	WarningStyle             = "warning"
	LogStyle                 = "log"
	WarningBoldStyle         = "warning-bold"
	SuccessStyle             = "success"
	SuccessBoldStyle         = "success-bold"
	ErrorIndentChar          = "✕"
	WarningIndentChar        = "!"
	SuccessIndentChar        = "✓"
	HeaderIndentChar         = "»"
	LogIndentChar            = "|"
	RecommendedWordWrapLimit = 80
)

// Option controls output styling.
type Option func(*config)

// WithIndentChar configures output with an indent character
func WithIndentChar(char string) Option {
	return func(c *config) {
		c.IndentCharacter = char
	}
}

// WithErrorStyle configures output using the ErrorStyle
func WithErrorStyle() Option {
	return func(c *config) {
		c.Style = ErrorStyle
	}
}

// WithErrorBoldStyle configures output using the ErrorBoldStyle
func WithErrorBoldStyle() Option {
	return func(c *config) {
		c.Style = ErrorBoldStyle
	}
}

// WithErrorBoldStyle configures output using the a style
func WithStyle(style string) Option {
	return func(c *config) {
		c.Style = style
	}
}

// WithWriter specifies the writer for the output.
func WithWriter(w io.Writer) Option {
	return func(c *config) { c.Writer = w }
}

// WithIndent configures output to be indented
func WithIndent(i int) Option {
	return func(c *config) {
		c.Indent = i
	}
}

// NamedValue outputs content in the format: key: value
type NamedValue struct {
	Name  string
	Value interface{}
}

// UI is the primary interface for interacting with a user via the CLI.
//
// Some of the methods on this interface return values that have a lifetime
// such as StepGroup. While these are still active (haven't called
// the Done or equivalent method on these values), no other method on the
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

	// Output a header style value to the screen
	Header(msg string, opts ...Option)

	NamedValues(rows []NamedValue, opts ...Option)

	// StepGroup returns a value that can be used to output individual steps
	// that have their own message, status indicator, spinner, and
	// body. No other output mechanism (Output, Input, Status, etc.) may be
	// called until the StepGroup is complete.
	StepGroup() StepGroup
}

type StepGroup interface {
	// Start a step in the output with the arguments making up the initial message
	Add(string) Step
	// Marks the StepGroup as done
	Done()
}

type Step interface {
	// Completes a step marking it as successful, and starts the next step if there are any more steps.
	Success(delay time.Duration, a ...interface{})

	// Completes a step marking it as a warning, and starts the next step if there are any more steps.
	Warning(delay time.Duration, a ...interface{})

	// Completes a step marking it as an error, stops execution of an next steps.
	Error(delay time.Duration, a ...interface{})
}
