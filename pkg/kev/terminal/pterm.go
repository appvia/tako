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

	"github.com/pterm/pterm"
)

func (lp linePrinter) printMsg() {
	lp.style.Println(lp.msg)
}

type pTermUI struct{}

// PTermUI returns a PTerm.sh implementation of UI
func PTermUI() UI {
	return &pTermUI{}
}

func (ui *pTermUI) Output(msg string, opts ...Option) {
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

func (ui *pTermUI) NamedValues(rows []NamedValue, opts ...Option) {
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

func (ui *pTermUI) Header(msg string, opts ...Option) {
	header(opts...).Println(msg)
}

func (ui *pTermUI) OutputWriters() (io.Writer, io.Writer, error) {
	return os.Stdout, os.Stderr, nil
}

func (ui *pTermUI) StepGroup() StepGroup {
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
