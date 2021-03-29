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

package kev

import (
	"fmt"
	"strings"

	kmd "github.com/appvia/komando"
	"github.com/mitchellh/go-wordwrap"
)

type renderStepType uint

const (
	renderStepLoad renderStepType = iota
	renderStepLoadSkaffold
	renderStepReconcile
	renderStepRenderGeneral
	renderStepRenderOverlay
)

var renderStepStrings = map[renderStepType]struct {
	Error        string
	ErrorDetails string
	Other        map[string]string
}{
	renderStepLoad: {
		Error: "Cannot load the project!",
	},

	renderStepLoadSkaffold: {
		Error: "Cannot find configured Skaffold manifest!",
		ErrorDetails: fmt.Sprintf(`
A valid %s is required as the project was initialised 
with Skaffold support. Please ensure one exists or you may need to 
run the 'init' command with the '--skaffold' flag.
		`, SkaffoldFileName),
	},

	renderStepReconcile: {
		Error: "Cannot detect project updates!",
	},

	renderStepRenderGeneral: {
		Error: "Cannot render project!",
	},

	renderStepRenderOverlay: {
		Error: "Cannot overlay environment settings during render!",
		ErrorDetails: fmt.Sprintf(`
'%s' cannot super impose environment settings over the compose source values.
This is important as it ensures that project rendered manifests will have 
environment specific settings.
		`, GetManifestName()),
	},
}

func renderStepError(ui kmd.UI, s kmd.Step, step renderStepType, err error) {
	stepStrings := renderStepStrings[step]
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
