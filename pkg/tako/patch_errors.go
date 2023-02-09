/**
 * Copyright 2023 Appvia Ltd <info@appvia.io>
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

package tako

import (
	"strings"

	kmd "github.com/appvia/komando"
	"github.com/mitchellh/go-wordwrap"
)

type patchStepType uint

const (
	patchStepPatchImages patchStepType = iota
)

var patchStepStrings = map[patchStepType]struct {
	Error        string
	ErrorDetails string
	Other        map[string]string
}{
	patchStepPatchImages: {
		Error: "Encountered an error while patching manifests with supplied service image!",
	},
}

func patchStepError(ui kmd.UI, s kmd.Step, step patchStepType, err error) {
	stepStrings := patchStepStrings[step]
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
