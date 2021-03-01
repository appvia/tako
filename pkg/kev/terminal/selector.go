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
	"os"

	"github.com/containerd/console"
	"github.com/mattn/go-isatty"
	sshterm "golang.org/x/crypto/ssh/terminal"
)

// TODO: create the basicUI
// type basicUI struct{}

// Returns a UI which will write to the current processes
// stdout/stderr.
func ConsoleUI() UI {
	// We do both of these checks because some sneaky environments fool
	// one or the other and we really only want the glint-based UI in
	// truly interactive environments.
	pterm := isatty.IsTerminal(os.Stdout.Fd()) && sshterm.IsTerminal(int(os.Stdout.Fd()))
	if pterm {
		pterm = false
		if c, err := console.ConsoleFromFile(os.Stdout); err == nil {
			if sz, err := c.Size(); err == nil {
				pterm = sz.Height > 0 && sz.Width > 0
			}
		}
	}

	if pterm {
		return PtermUI()
	} else {
		return PtermUI() // change to a basic buffer based UI for testing
	}
}
