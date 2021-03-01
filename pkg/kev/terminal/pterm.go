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
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/pterm/pterm"
)

func (lp linePrinter) printMsg() {
	lp.style.Println(lp.msg)
}

type ptermStepGroup struct {
	steps      []*ptermStep
	stepActive bool
}

func newPtermStepGroup() *ptermStepGroup {
	return &ptermStepGroup{}
}

func (g *ptermStepGroup) Add(msg string) Step {
	step := newPtermStep(msg, g)
	if !g.stepActive {
		step.start()
	}
	g.steps = append(g.steps, step)
	return step
}

func (g *ptermStepGroup) Done() {
	g.steps = nil
	g.stepActive = false
}

func (g *ptermStepGroup) next(index int) {
	if len(g.steps) > index+1 {
		g.steps[index+1].start()
	}
}

type ptermStep struct {
	sg      *ptermStepGroup
	printer *pterm.SpinnerPrinter
	index   int
}

func (s *ptermStep) start() {
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

func newPtermStep(msg string, sg *ptermStepGroup) *ptermStep {
	index := len(sg.steps) + 1
	printer := spinner().WithText(msg)
	return &ptermStep{sg: sg, printer: printer, index: index}
}

func (s *ptermStep) Error(delay time.Duration, a ...interface{}) {
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

func (s *ptermStep) Success(delay time.Duration, a ...interface{}) {
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
func (s *ptermStep) Warning(delay time.Duration, a ...interface{}) {
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

type ptermUI struct{}

func PtermUI() UI {
	return &ptermUI{}
}

func (ui *ptermUI) Output(msg string, opts ...Option) {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}

	if len(cfg.IndentCharacter) > 0 {
		msg = cfg.IndentCharacter + " " + msg
	}

	for i := 0; i < cfg.Indent; i++ {
		msg = " " + msg
	}

	var prints []linePrinter
	style := pterm.Style{}
	switch cfg.Style {
	case ErrorStyle, ErrorBoldStyle:
		style = style.Add(pterm.ThemeDefault.FatalMessageStyle)
		if cfg.Style == ErrorBoldStyle {
			style = style.Add(pterm.Style{pterm.Bold})
		}

		lines := strings.Split(msg, "\n")
		if len(lines) > 0 {
			prints = append(prints, linePrinter{style: style, msg: lines[0]})

			for _, line := range lines[1:] {
				for i := 0; i < cfg.Indent; i++ {
					line = " " + line
				}
				prints = append(prints, linePrinter{style: pterm.Style{}, msg: "  " + line})
			}
		}
	case SuccessStyle, SuccessBoldStyle:
		style = style.Add(pterm.ThemeDefault.SuccessMessageStyle)
		if cfg.Style == SuccessBoldStyle {
			style = style.Add(pterm.Style{pterm.Bold})
		}
		prints = append(prints, linePrinter{style: style, msg: msg})
	case WarningStyle, WarningBoldStyle:
		style = style.Add(pterm.ThemeDefault.WarningMessageStyle)
		if cfg.Style == WarningBoldStyle {
			style = style.Add(pterm.Style{pterm.Bold})
		}
		prints = append(prints, linePrinter{style: style, msg: msg})
	case LogStyle:
		style = style.Add(pterm.Style{pterm.FgLightBlue})
		prints = append(prints, linePrinter{style: style, msg: msg})
	default:
		prints = append(prints, linePrinter{style: style, msg: msg})
	}

	for _, p := range prints {
		p.printMsg()
	}
}

func (ui *ptermUI) NamedValues(rows []NamedValue, opts ...Option) {
	var buf bytes.Buffer
	tr := tabwriter.NewWriter(&buf, 1, 8, 0, ' ', tabwriter.AlignRight)
	for _, row := range rows {
		switch v := row.Value.(type) {
		case int, uint, int8, uint8, int16, uint16, int32, uint32, int64, uint64:
			fmt.Fprintf(tr, "  %s: \t%d\n", row.Name, row.Value)
		case float32, float64:
			fmt.Fprintf(tr, "  %s: \t%f\n", row.Name, row.Value)
		case bool:
			fmt.Fprintf(tr, "  %s: \t%v\n", row.Name, row.Value)
		case string:
			if v == "" {
				continue
			}
			fmt.Fprintf(tr, "  %s: \t%s\n", row.Name, row.Value)
		default:
			fmt.Fprintf(tr, "  %s: \t%s\n", row.Name, row.Value)
		}
	}
	tr.Flush()

	// We want to trim the trailing newline
	text := buf.String()
	if len(text) > 0 && text[len(text)-1] == '\n' {
		text = text[:len(text)-1]
	}

	ui.Output(text, opts...)
}

func (ui *ptermUI) Header(msg string, opts ...Option) {
	header(opts...).Println(msg)
}

func (ui *ptermUI) OutputWriters() (io.Writer, io.Writer, error) {
	return os.Stdout, os.Stderr, nil
}

func (ui *ptermUI) StepGroup() StepGroup {
	return newPtermStepGroup()
}

func header(opts ...Option) *pterm.SectionPrinter {
	cfg := config{IndentCharacter: HeaderIndentChar}
	for _, opt := range opts {
		opt(&cfg)
	}

	style := pterm.Style{pterm.Bold}
	switch cfg.Style {
	case ErrorStyle, ErrorBoldStyle:
		style = style.Add(*pterm.NewStyle(pterm.FgRed))
	case WarningStyle:
		style = style.Add(*pterm.NewStyle(pterm.FgYellow))
	}

	return pterm.
		DefaultSection.
		WithStyle(&style).
		WithIndentCharacter(cfg.IndentCharacter).
		WithTopPadding(1).
		WithBottomPadding(0)
}

func spinner() pterm.SpinnerPrinter {
	var spinnerSequences = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	printer := pterm.SpinnerPrinter{
		Sequence:     spinnerSequences,
		Style:        pterm.NewStyle(pterm.FgBlack),
		Delay:        time.Millisecond * 100,
		MessageStyle: &pterm.Style{pterm.FgBlack},
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
