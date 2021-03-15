/**
 * Copyright 2021 Appvia Ltd <info@appvia.io>
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
	"fmt"
	"io"
	"os"
	"time"
)

type noOpUI struct{}

// NoOpUI returns a no op implementation of UI
func NoOpUI() UI {
	return &noOpUI{}
}

func (ui *noOpUI) Output(msg string, opts ...Option) {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.Writer != nil {
		_, _ = cfg.Writer.Write([]byte(msg))
	}
}

func (ui *noOpUI) OutputWriters() (stdout, stderr io.Writer, err error) {
	return os.Stdout, os.Stderr, nil
}

func (ui *noOpUI) Header(msg string, opts ...Option) {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.Writer != nil {
		_, _ = cfg.Writer.Write([]byte(msg))
	}
}

func (ui *noOpUI) NamedValues(rows []NamedValue, opts ...Option) {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.Writer != nil {
		for _, row := range rows {
			_, _ = cfg.Writer.Write([]byte(fmt.Sprintf("%s - %v", row.Name, row.Value)))
		}
	}
}

type noOpStepGroup struct{}

func (ui *noOpUI) StepGroup() StepGroup { return &noOpStepGroup{} }

func (sg *noOpStepGroup) Add(_ string) Step { return &noOpStep{} }

func (sg *noOpStepGroup) Done() {}

type noOpStep struct{}

func (s *noOpStep) Success(delay time.Duration, a ...interface{}) {}

func (s *noOpStep) Warning(delay time.Duration, a ...interface{}) {}

func (s *noOpStep) Error(delay time.Duration, a ...interface{}) {}
