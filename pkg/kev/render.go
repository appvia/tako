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

	"github.com/appvia/kev/pkg/kev/config"
	kmd "github.com/appvia/komando"
)

func NewRenderRunner(workingDir string, opts ...Options) *RenderRunner {
	runner := &RenderRunner{Project: &Project{workingDir: workingDir}}
	runner.Init(opts...)
	return runner
}

// Run executes the runner returning results that can be written to disk
func (r *RenderRunner) Run() (WritableResults, error) {
	m, err := LoadManifest(r.workingDir)
	if err != nil {
		return nil, err
	}

	if err := r.ValidateSources(m.Sources, config.SecretMatchers); err != nil {
		return nil, err
	}

	if err := r.ValidateEnvSources(m.Environments, config.SecretMatchers); err != nil {
		return nil, err
	}
	// Validating compose sources
	// Validating deployment environments
	// Detecting project updates...(reconcile)
	// Rendering manifests, format: %s
	return nil, nil
}

func (p *RenderRunner) ValidateEnvSources(envs Environments, matchers []map[string]string) error {
	p.UI.Header("Validating deployment environments...")
	var detectHit bool

	for _, env := range envs {
		secretsDetected, err := p.detectSecretsInSources(env.ToSources(), matchers)
		if err != nil {
			return err
		}
		if secretsDetected {
			detectHit = true
		}
	}

	p.UI.Output("")
	p.UI.Output("Validation successful!")
	if detectHit {
		p.UI.Output(fmt.Sprintf(`However, to prevent secrets leaking, see help page:
%s`, SecretsReferenceUrl))
	}

	return nil
}

func printRenderProjectWithOptionsError(ui kmd.UI) {
	ui.Output("")
	ui.Output("Project had errors during render.\n"+
		fmt.Sprintf("'%s' experienced some errors during project render. The output\n", GetManifestName())+
		"above should contain the failure messages. Please correct these errors and\n"+
		fmt.Sprintf("run '%s render' again.", GetManifestName()),
		kmd.WithErrorBoldStyle(),
		kmd.WithIndentChar(kmd.ErrorIndentChar),
	)
}
