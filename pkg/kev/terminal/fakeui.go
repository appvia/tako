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
	"strconv"
)

type fakeUI struct {
	log *fakeUILog
}

// FakeUIAndLog returns a fake UI implementation and a log to help tracking of UI ops in tests.
func FakeUIAndLog() (UI, UILog) {
	log := newFakeUILog()
	return &fakeUI{log: log}, log
}

func (ui *fakeUI) Output(msg string, opts ...Option) {
	msgWithOpts := map[string][]string{msg: getConfig(createConfig(opts))}
	ui.log.logOp(outputOp, msgWithOpts)
}

func (ui *fakeUI) OutputWriters() (stdout, stderr io.Writer, err error) {
	return os.Stdout, os.Stderr, nil
}

func (ui *fakeUI) Header(msg string, opts ...Option) {
	msgWithOpts := map[string][]string{msg: getConfig(createConfig(opts))}
	ui.log.logOp(headerOp, msgWithOpts)
}

func (ui *fakeUI) NamedValues(rows []NamedValue, opts ...Option) {
	for _, row := range rows {
		msg := fmt.Sprintf("%s: %v", row.Name, row.Value)
		msgWithOpts := map[string][]string{msg: getConfig(createConfig(opts))}
		ui.log.logOp(namedValuesOp, msgWithOpts)
	}
}

type fakeStepGroup struct {
	log *fakeUILog
}

type fakeStep struct {
	log *fakeUILog
}

func (ui *fakeUI) StepGroup() StepGroup {
	return &fakeStepGroup{log: ui.log}
}

func (sg *fakeStepGroup) Add(msg string) Step {
	msgWithOpts := map[string][]string{msg: {}}
	sg.log.logOp(stepOp, msgWithOpts)
	return &fakeStep{log: sg.log}
}

func (sg *fakeStepGroup) Done() {}

func (s *fakeStep) Success(a ...interface{}) {
	s.log.logStepStop(LogStepSuccess, a)
}

func (s *fakeStep) Warning(a ...interface{}) {
	s.log.logStepStop(LogStepWarning, a)
}

func (s *fakeStep) Error(a ...interface{}) {
	s.log.logStepStop(LogStepError, a)
}

func createConfig(opts []Option) config {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

func getConfig(cfg config) []string {
	var out []string
	if cfg.Indent > 0 {
		out = append(out, strconv.Itoa(cfg.Indent))
	}
	if len(cfg.IndentCharacter) > 0 {
		out = append(out, cfg.IndentCharacter)
	}
	if len(cfg.Style) > 0 {
		out = append(out, cfg.Style)
	}
	if len(out) == 0 {
		return []string{}
	}
	return out
}
