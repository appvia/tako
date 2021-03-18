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
	"path"
	"path/filepath"

	"github.com/appvia/kev/pkg/kev/config"
	"github.com/appvia/kev/pkg/kev/terminal"
)

// NewInitRunner returns a runner that can initialise a project using the provided options
func NewInitRunner(workingDir string, opts ...Options) *InitRunner {
	runner := &InitRunner{Project: &Project{workingDir: workingDir}}
	runner.Init(opts...)
	return runner
}

// Run executes the runner returning results that can be written to disk
func (r *InitRunner) Run() (WritableResults, error) {
	var skManifest *SkaffoldManifest

	if err := r.EnsureFirstInit(); err != nil {
		return nil, err
	}

	sources, err := r.DetectSources()
	if err != nil {
		return nil, err
	}

	if err := r.ValidateSources(sources, config.SecretMatchers); err != nil {
		return nil, err
	}

	if err := r.CreateManifestAndEnvironmentOverrides(sources); err != nil {
		return nil, err
	}

	r.UI.Header("Detecting Skaffold settings...")
	if r.config.skaffold {
		if skManifest, err = r.CreateOrUpdateSkaffoldManifest(); err != nil {
			return nil, err
		}
	} else {
		r.UI.Output("Skipping - no Skaffold options detected")
	}

	return createInitWritableResults(r.workingDir, r.manifest, skManifest), nil
}

// EnsureFirstInit ensures the project has not been already initialised
func (r *InitRunner) EnsureFirstInit() error {
	r.UI.Header("Verifying project...")
	sg := r.UI.StepGroup()
	defer sg.Done()
	s := sg.Add("Ensuring this project has not already been initialised")

	manifestPath := path.Join(r.workingDir, ManifestFilename)
	if ManifestExistsForPath(manifestPath) {
		absWd, _ := filepath.Abs(r.workingDir)
		err := fmt.Errorf("%s already exists at: %s", ManifestFilename, absWd)
		initStepError(r.UI, s, initStepConfig, err)
		return err
	}

	s.Success()
	return nil
}

// DetectSources detects the compose yaml sources required for initialisation
func (r *InitRunner) DetectSources() (*Sources, error) {
	r.UI.Header("Detecting compose sources...")

	sg := r.UI.StepGroup()
	defer sg.Done()
	if len(r.config.composeSources) > 0 {
		for _, source := range r.config.composeSources {
			s := sg.Add(fmt.Sprintf("Scanning for: %s", source))

			if !fileExists(source) {
				err := fmt.Errorf("cannot find compose source %q", source)
				initStepError(r.UI, s, initStepComposeSource, err)
				return nil, err
			}

			s.Success("Using: ", source)
		}

		return &Sources{Files: r.config.composeSources}, nil
	}

	s := sg.Add(fmt.Sprintf("Scanning for compose configuration"))
	defaults, err := findDefaultComposeFiles(r.workingDir)
	if err != nil {
		initStepError(r.UI, s, initStepComposeSource, err)
		return nil, err
	}
	s.Success()

	for _, source := range defaults {
		s := sg.Add(fmt.Sprintf("Using: %s", source))
		s.Success()
	}

	return &Sources{Files: defaults}, nil
}

// ValidateSources includes validation checks to ensure the compose sources are valid.
// This function can be extended to include different forms of
// validation (for now it detect any secrets found in the sources).
func (r *InitRunner) ValidateSources(sources *Sources, matchers []map[string]string) error {
	r.UI.Header("Validating compose sources...")

	secretsDetected, err := r.detectSecretsInSources(sources, matchers)
	if err != nil {
		return err
	}

	if secretsDetected {
		r.UI.Output("")
		r.UI.Output("Validation successful!")
		r.UI.Output(fmt.Sprintf(`However, to prevent secrets leaking, see help page:
%s`, SecretsReferenceUrl))
	}

	return nil
}

func (r *InitRunner) detectSecretsInSources(sources *Sources, matchers []map[string]string) (bool, error) {
	var detected bool

	sg := r.UI.StepGroup()
	defer sg.Done()
	for _, composeFile := range sources.Files {
		r.UI.Output(fmt.Sprintf("Detecting secrets in: %s", composeFile))
		composeProject, err := NewComposeProject([]string{composeFile})
		if err != nil {
			initStepError(r.UI, sg.Add(""), initStepParsingComposeConfig, err)
			return false, err
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
				r.UI.Output(
					fmt.Sprintf("env var [%s] - %s", hit.envVar, hit.description),
					terminal.WithStyle(terminal.LogStyle),
					terminal.WithIndentChar(terminal.LogIndentChar),
					terminal.WithIndent(3),
				)
			}
		}
	}
	return detected, nil
}

// CreateManifestAndEnvironmentOverrides creates a base manifest and the related compose environment overrides
func (r *InitRunner) CreateManifestAndEnvironmentOverrides(sources *Sources) error {
	r.manifest = NewManifest(sources)
	r.manifest.UI = r.UI

	sg := r.UI.StepGroup()
	defer sg.Done()

	if _, err := r.manifest.CalculateSourcesBaseOverride(); err != nil {
		initStepError(r.UI, sg.Add(""), initStepGenerateManifest, err)
		return err
	}

	r.manifest.MintEnvironments(r.config.envs)
	return nil
}

// CreateOrUpdateSkaffoldManifest creates or updates a skaffold manifest
func (r *InitRunner) CreateOrUpdateSkaffoldManifest() (*SkaffoldManifest, error) {
	var err error
	var skManifest *SkaffoldManifest

	sg := r.UI.StepGroup()
	defer sg.Done()

	composeProject, err := r.manifest.SourcesToComposeProject()
	if err != nil {
		initStepError(r.UI, sg.Add(""), initStepParsingComposeConfig, err)
		return nil, err
	}

	skPath := path.Join(r.workingDir, SkaffoldFileName)
	envs := r.manifest.GetEnvironmentsNames()
	switch ManifestExistsForPath(skPath) {
	case true:
		updateStep := sg.Add(fmt.Sprintf("Adding deployment environments to existing Skaffold config: %s", skPath))
		// Skaffold manifest already present - add additional profiles to it!
		// Note: kev will skip profiles with names matching those of existing
		// profile names defined in Skaffold to avoid profile "hijack".
		if skManifest, err = AddProfiles(skPath, envs, true); err != nil {
			initStepError(r.UI, updateStep, initStepUpdateSkaffold, err)
			return nil, err
		}
		updateStep.Success()
	case false:
		createStep := sg.Add(fmt.Sprintf("Creating Skaffold config with deployment environment profiles at: %s", skPath))
		if skManifest, err = NewSkaffoldManifest(envs, composeProject); err != nil {
			initStepError(r.UI, createStep, initStepCreateSkaffold, err)
			return nil, err
		}
		createStep.Success()
	}

	r.manifest.Skaffold = SkaffoldFileName
	return skManifest, nil
}

func createInitWritableResults(workingDir string, manifest *Manifest, skManifest *SkaffoldManifest) WritableResults {
	var out []WritableResult
	out = append(out, WritableResult{
		WriterTo: manifest,
		FilePath: path.Join(workingDir, ManifestFilename),
	})
	out = append(out, manifest.Environments.toWritableResults()...)

	if skManifest != nil {
		out = append(out, WritableResult{
			WriterTo: skManifest,
			FilePath: path.Join(workingDir, SkaffoldFileName),
		})
	}
	return out
}

func printInitProjectWithOptionsError(ui terminal.UI) {
	ui.Output("")
	ui.Output("Project had errors during initialisation.\n"+
		fmt.Sprintf("'%s' experienced some errors during project initialisation. The output\n", GetManifestName())+
		"above should contain the failure messages. Please correct these errors and\n"+
		fmt.Sprintf("run '%s init' again.", GetManifestName()),
		terminal.WithErrorBoldStyle(),
		terminal.WithIndentChar(terminal.ErrorIndentChar),
	)
}

func printInitProjectWithOptionsSuccess(ui terminal.UI, envs Environments) {
	ui.Output("")
	ui.Output("Project initialised!", terminal.WithStyle(terminal.SuccessBoldStyle))
	ui.Output(fmt.Sprintf("A '%s' file was created. Do not edit this file.\n", ManifestFilename)+
		"It syncs your deployment environments to updates made \n"+
		"to your compose sources.",
		terminal.WithStyle(terminal.SuccessStyle),
	)
	var namedValues []terminal.NamedValue
	for _, env := range envs {
		namedValues = append(namedValues, terminal.NamedValue{Name: env.Name, Value: env.File})
	}
	ui.Output("")
	ui.Output("And, the following deployment env files have been created:", terminal.WithStyle(terminal.SuccessStyle))
	ui.NamedValues(namedValues, terminal.WithStyle(terminal.SuccessStyle))
	ui.Output("")
	ui.Output("Update these to configure your deployments per related environment.", terminal.WithStyle(terminal.SuccessStyle))
	ui.Output("")
	ui.Output(fmt.Sprintf("You may now call `%s render` to prepare your project for deployment.", GetManifestName()))
}
