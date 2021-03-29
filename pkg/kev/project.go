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

	kmd "github.com/appvia/komando"
	"github.com/pkg/errors"
)

// Init initialises the base project to be used in a runner
func (p *Project) Init(opts ...Options) {
	var cfg runConfig
	for _, o := range opts {
		o(p, &cfg)
	}
	p.config = cfg
	if p.UI == nil {
		p.UI = kmd.ConsoleUI()
	}
}

// ValidateSources includes validation checks to ensure the compose sources are valid.
// This function can be extended to include different forms of
// validation (for now it detect any secrets found in the sources).
func (p *Project) ValidateSources(sources *Sources, matchers []map[string]string) error {
	p.UI.Header("Validating compose sources...")

	secretsDetected, err := p.detectSecretsInSources(sources, matchers)
	if err != nil {
		return err
	}

	p.UI.Output("")
	p.UI.Output("Validation successful!")

	if secretsDetected {
		p.UI.Output(fmt.Sprintf(`However, to prevent secrets leaking, see help page:
%s`, SecretsReferenceUrl))
	}

	return nil
}

func (p *Project) detectSecretsInSources(sources *Sources, matchers []map[string]string) (bool, error) {
	var detected bool

	sg := p.UI.StepGroup()
	defer sg.Done()
	for _, composeFile := range sources.Files {
		p.UI.Output(fmt.Sprintf("Detecting secrets in: %s", composeFile))
		composeProject, err := NewComposeProject([]string{composeFile})
		if err != nil {
			decoratedErr := errors.Errorf("%s\nsee compose file: %s", err.Error(), composeFile)
			initStepError(p.UI, sg.Add(""), initStepParsingComposeConfig, decoratedErr)
			return false, decoratedErr
		}

		for _, s := range composeProject.Services {
			step := sg.Add(fmt.Sprintf("Analysing service: %s", s.Name))
			serviceConfig := ServiceConfig{Name: s.Name, Environment: s.Environment}

			hits := serviceConfig.detectSecretsInEnvVars(matchers)
			if len(hits) == 0 {
				step.Success("Non detected in service: ", s.Name)
				continue
			}

			detected = true
			step.Warning("Detected in service: ", s.Name)

			for _, hit := range hits {
				p.UI.Output(
					fmt.Sprintf("env var [%s] - %s", hit.envVar, hit.description),
					kmd.WithStyle(kmd.LogStyle),
					kmd.WithIndentChar(kmd.LogIndentChar),
					kmd.WithIndent(3),
				)
			}
		}
	}
	return detected, nil
}

// Manifest returns the project's manifest
func (p *Project) Manifest() *Manifest {
	return p.manifest
}

func WithComposeSources(c []string) Options {
	return func(project *Project, cfg *runConfig) {
		cfg.composeSources = c
	}
}

func WithEnvs(c []string) Options {
	return func(project *Project, cfg *runConfig) {
		cfg.envs = c
	}
}

func WithSkaffold(c bool) Options {
	return func(project *Project, cfg *runConfig) {
		cfg.skaffold = c
	}
}

func WithManifestFormat(c string) Options {
	return func(project *Project, cfg *runConfig) {
		cfg.manifestFormat = c
	}
}

func WithManifestsAsSingleFile(c bool) Options {
	return func(project *Project, cfg *runConfig) {
		cfg.manifestsAsSingleFile = c
	}
}

func WithOutputDir(c string) Options {
	return func(project *Project, cfg *runConfig) {
		cfg.outputDir = c
	}
}

func WithUI(ui kmd.UI) Options {
	return func(project *Project, cfg *runConfig) {
		project.UI = ui
	}
}
