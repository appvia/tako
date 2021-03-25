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
	"github.com/appvia/kev/pkg/kev/converter"
	kmd "github.com/appvia/komando"
	"github.com/pkg/errors"
)

func NewRenderRunner(workingDir string, opts ...Options) *RenderRunner {
	runner := &RenderRunner{Project: &Project{workingDir: workingDir}}
	runner.Init(opts...)
	return runner
}

// Run executes the runner returning results that can be written to disk
func (r *RenderRunner) Run() error {
	if err := r.LoadProject(); err != nil {
		return err
	}

	if err := r.ValidateSources(r.manifest.Sources, config.SecretMatchers); err != nil {
		return err
	}

	if err := r.ValidateEnvSources(r.manifest.Environments, config.SecretMatchers); err != nil {
		return err
	}

	if err := r.ReconcileEnvsAndWriteUpdates(); err != nil {
		return err
	}

	if err := r.RenderManifests(); err != nil {
		return err
	}

	return nil
}

func (r *RenderRunner) LoadProject() error {
	manifest, err := LoadManifest(r.workingDir)
	if err != nil {
		return err
	}
	r.manifest = manifest
	r.manifest.UI = r.UI
	return nil
}

func (r *RenderRunner) ValidateEnvSources(envs Environments, matchers []map[string]string) error {
	r.UI.Header("Validating deployment environments...")
	var detectHit bool

	for _, env := range envs {
		secretsDetected, err := r.detectSecretsInSources(env.ToSources(), matchers)
		if err != nil {
			return err
		}
		if secretsDetected {
			detectHit = true
		}
	}

	r.UI.Output("")
	r.UI.Output("Validation successful!")
	if detectHit {
		r.UI.Output(fmt.Sprintf(`However, to prevent secrets leaking, see help page:
%s`, SecretsReferenceUrl))
	}

	return nil
}

func (r *RenderRunner) ReconcileEnvsAndWriteUpdates() error {
	r.UI.Header("Detecting project updates...")
	_, err := r.manifest.ReconcileConfig()
	if err != nil {
		return errors.Wrap(err, "Could not reconcile project latest")
	}
	return r.manifest.Environments.Write()
}

func (r *RenderRunner) RenderManifests() error {
	manifestFormat := r.config.manifestFormat
	r.UI.Header(fmt.Sprintf("Rendering manifests, format: %s...", manifestFormat))

	_, err := r.manifest.RenderWithConvertor(
		converter.Factory(manifestFormat, r.UI),
		r.config.outputDir,
		r.config.manifestsAsSingleFile,
		r.config.envs,
		nil)

	return err
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