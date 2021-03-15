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
	"strings"
	"time"

	"github.com/pterm/pterm"
)

type pTermStepGroup struct {
	steps      []*pTermStep
	stepActive bool
}

func newPtermStepGroup() *pTermStepGroup {
	return &pTermStepGroup{}
}

func (g *pTermStepGroup) Add(msg string) Step {
	step := newPTermStep(msg, g)
	if !g.stepActive {
		step.start()
	}
	g.steps = append(g.steps, step)
	return step
}

func (g *pTermStepGroup) Done() {
	g.steps = nil
	g.stepActive = false
}

func (g *pTermStepGroup) next(index int) {
	if len(g.steps) > index+1 {
		g.steps[index+1].start()
	}
}

type pTermStep struct {
	sg      *pTermStepGroup
	printer *pterm.SpinnerPrinter
	index   int
}

func (s *pTermStep) start() {
	s.printer.IsActive = true

	go func() {
		for s.printer.IsActive {
			for _, seq := range s.printer.Sequence {
				if s.printer.IsActive {
					pterm.Printo(s.printer.Style.Sprint(seq) + " " + s.printer.MessageStyle.Sprint(s.printer.Text))
					time.Sleep(s.printer.Delay)
				}
			}
		}
	}()
}

func newPTermStep(msg string, sg *pTermStepGroup) *pTermStep {
	index := len(sg.steps) + 1
	printer := spinner().WithText(msg)
	return &pTermStep{sg: sg, printer: printer, index: index}
}

func (s *pTermStep) Error(delay time.Duration, a ...interface{}) {
	if delay.Seconds() > 0 {
		time.Sleep(delay)
	}

	if s.printer.FailPrinter == nil {
		s.printer.FailPrinter = &pterm.Error
	}

	if len(a) == 0 {
		a = []interface{}{s.printer.Text}
	}
	clearLine()
	pterm.Printo(s.printer.FailPrinter.Sprint(a...))
	_ = s.printer.Stop()
	s.sg.stepActive = false
}

func clearLine() {
	pterm.Printo(strings.Repeat(" ", pterm.GetTerminalWidth()))
}

func (s *pTermStep) Success(delay time.Duration, a ...interface{}) {
	if delay.Seconds() > 0 {
		time.Sleep(delay)
	}

	if s.printer.SuccessPrinter == nil {
		s.printer.SuccessPrinter = &pterm.Success
	}

	if len(a) == 0 {
		a = []interface{}{s.printer.Text}
	}
	clearLine()
	pterm.Printo(s.printer.SuccessPrinter.Sprint(a...))
	_ = s.printer.Stop()
	s.sg.stepActive = false

	s.sg.next(s.index)
}
func (s *pTermStep) Warning(delay time.Duration, a ...interface{}) {
	if delay.Seconds() > 0 {
		time.Sleep(delay)
	}

	if s.printer.WarningPrinter == nil {
		s.printer.WarningPrinter = &pterm.Warning
	}

	if len(a) == 0 {
		a = []interface{}{s.printer.Text}
	}
	clearLine()
	pterm.Printo(s.printer.WarningPrinter.Sprint(a...))
	_ = s.printer.Stop()
	s.sg.stepActive = false

	s.sg.next(s.index)
}

func spinner() pterm.SpinnerPrinter {
	var spinnerSequences = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	printer := pterm.SpinnerPrinter{
		Sequence:     spinnerSequences,
		Style:        pterm.NewStyle(pterm.FgDefault),
		Delay:        time.Millisecond * 100,
		MessageStyle: &pterm.Style{pterm.FgDefault},
		SuccessPrinter: &pterm.PrefixPrinter{
			MessageStyle: &pterm.ThemeDefault.SuccessMessageStyle,
			Prefix: pterm.Prefix{
				Style: &pterm.ThemeDefault.SuccessMessageStyle,
				Text:  SuccessIndentChar,
			},
		},
		FailPrinter: &pterm.PrefixPrinter{
			MessageStyle: &pterm.ThemeDefault.FatalMessageStyle,
			Prefix: pterm.Prefix{
				Style: &pterm.ThemeDefault.FatalMessageStyle,
				Text:  ErrorIndentChar,
			},
		},
		WarningPrinter: &pterm.PrefixPrinter{
			MessageStyle: &pterm.ThemeDefault.WarningMessageStyle,
			Prefix: pterm.Prefix{
				Style: &pterm.ThemeDefault.WarningMessageStyle,
				Text:  WarningIndentChar,
			},
		},
	}
	return printer
}
