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

package kev

import (
	"strings"

	kmd "github.com/appvia/komando"
	"github.com/mitchellh/go-wordwrap"
)

type initStepType uint

const (
	initStepConfig initStepType = iota
	initStepComposeSource
	initStepParsingComposeConfig
	initStepGenerateManifest
	initStepValidatingSources
	initStepUpdateSkaffold
	initStepCreateSkaffold
)

var initStepStrings = map[initStepType]struct {
	Error        string
	ErrorDetails string
	Other        map[string]string
}{
	initStepConfig: {
		Error: "This project has already been initialised!",
	},

	initStepComposeSource: {
		Error: "Missing compose source file!",
		ErrorDetails: `
At least a single compose file and zero or more compose override files 
are required. These are used to initialise a project and to setup 
deployment environments. Without them a project cannot be initialised. 
		`,
	},

	initStepParsingComposeConfig: {
		Error: "Invalid compose source(s)!",
		ErrorDetails: `
The provided compose source(s) are invalid. 'init' requires valid 
compose source files - without them a project cannot be initialised or loaded. 
Use the command 'docker-compose -f <compose-source-file> config'
to double check your compose source(s) are valid.
`,
	},

	initStepGenerateManifest: {
		Error: "Cannot create manifest using compose source files!",
	},

	initStepValidatingSources: {
		Error: "Encountered an error while validating sources!",
	},

	initStepUpdateSkaffold: {
		Error: "Cannot update Skaffold manifest!",
	},

	initStepCreateSkaffold: {
		Error: "Cannot create Skaffold manifest!",
	},
}

func initStepError(ui kmd.UI, s kmd.Step, step initStepType, err error) {
	stepStrings := initStepStrings[step]
	s.Error(stepStrings.Error)
	ui.Output("")
	if v := stepStrings.ErrorDetails; v != "" {
		ui.Output(strings.TrimSpace(v), kmd.WithErrorStyle(), kmd.WithIndentChar(kmd.ErrorIndentChar))
		ui.Output("")
	}

	ui.Output(
		wordwrap.WrapString(err.Error(), kmd.RecommendedWordWrapLimit),
		kmd.WithErrorStyle(),
		kmd.WithIndentChar(kmd.ErrorIndentChar),
	)
}
