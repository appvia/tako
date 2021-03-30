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
	"github.com/appvia/kev/pkg/kev/config"
	kmd "github.com/appvia/komando"
	"github.com/pkg/errors"
)

const (
	// SandboxEnv is a default environment name
	SandboxEnv = "dev"
)

var (
	// ManifestFilename is a name of main application manifest file
	ManifestFilename    = "kev.yaml"
	SecretsReferenceUrl = "https://github.com/appvia/kev/blob/master/docs/reference/config-params.md#reference-k8s-secret-key-value"
)

// InitProjectWithOptions initialises a kev project in the specified working directory
// using the provided options (if any).
func InitProjectWithOptions(workingDir string, opts ...Options) error {
	runner := NewInitRunner(workingDir, opts...)
	ui := runner.UI

	results, err := runner.Run()
	if err != nil {
		printInitProjectWithOptionsError(ui)
		return err
	}

	if err := results.Write(); err != nil {
		printInitProjectWithOptionsError(ui)
		return err
	}

	printInitProjectWithOptionsSuccess(ui, runner.manifest.Environments)
	return nil
}

func RenderProjectWithOptions(workingDir string, opts ...Options) error {
	runner := NewRenderRunner(workingDir, opts...)
	ui := runner.UI

	results, err := runner.Run()
	if err != nil {
		printRenderProjectWithOptionsError(ui)
		return err
	}

	envs, err := runner.Manifest().GetEnvironments(runner.config.envs)
	if err != nil {
		return err
	}

	printRenderProjectWithOptionsSuccess(ui, results, envs, runner.config.manifestFormat)

	return nil
}

func DevWithOptions(workingDir string, handler ChangeHandler, opts ...Options) error {
	runner := NewDevRunner(workingDir, handler, opts...)
	err := runner.Run()

	if err != nil {
		printDevProjectWithOptionsError(runner.UI)
		return err
	}

	return nil
}

// Reconcile reconciles changes with docker-compose sources against deployment environments.
func Reconcile(workingDir string) (*Manifest, error) {
	m, err := LoadManifest(workingDir)
	if err != nil {
		return nil, err
	}

	// TODO(es) Remove this after dev cmd is moved to use new render runner
	m.UI = kmd.NoOpUI()

	if _, err := m.ReconcileConfig(); err != nil {
		return nil, errors.Wrap(err, "Could not reconcile project latest")
	}
	return m, err
}

// DetectSecrets detects any potential secrets defined in environment variables
// found either in sources or override environments.
// Any detected secrets are logged using a warning log level.
func DetectSecrets(workingDir string) error {
	m, err := LoadManifest(workingDir)
	if err != nil {
		return err
	}

	runner := &InitRunner{Project: &Project{workingDir: workingDir}}
	runner.Init()
	if _, err := runner.detectSecretsInSources(m.Sources, config.SecretMatchers); err != nil {
		return err
	}

	if err := m.DetectSecretsInEnvs(config.SecretMatchers); err != nil {
		return err
	}
	return nil
}
