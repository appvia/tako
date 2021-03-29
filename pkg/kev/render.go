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
func (r *RenderRunner) Run() (map[string]string, error) {
	if err := r.LoadProject(); err != nil {
		return nil, err
	}

	if err := r.ValidateSources(r.manifest.Sources, config.SecretMatchers); err != nil {
		return nil, err
	}

	if err := r.ValidateSkaffoldIfRequired(); err != nil {
		return nil, err
	}

	if err := r.ValidateEnvSources(config.SecretMatchers); err != nil {
		return nil, err
	}

	if err := r.ReconcileEnvsAndWriteUpdates(); err != nil {
		return nil, err
	}

	return r.RenderManifests()
}

func (r *RenderRunner) LoadProject() error {
	r.UI.Header("Loading...")

	sg := r.UI.StepGroup()
	defer sg.Done()

	if !ManifestExistsForPath(path.Join(r.workingDir, ManifestFilename)) {
		err := errors.Errorf("Missing project manifest: %s", ManifestFilename)
		renderStepError(r.UI, sg.Add(""), renderStepLoad, err)
		return err
	}

	manifest, err := LoadManifest(r.workingDir)
	if err != nil {
		renderStepError(r.UI, sg.Add(""), renderStepLoad, err)
		return err
	}
	r.manifest = manifest
	r.manifest.UI = r.UI
	return nil
}

func (r *RenderRunner) ValidateSkaffoldIfRequired() error {
	if len(r.manifest.Skaffold) == 0 {
		return nil
	}

	r.UI.Header("Verifying Skaffold...")
	sg := r.UI.StepGroup()
	defer sg.Done()

	step := sg.Add("Ensuring Skaffold manifest available")
	if !ManifestExistsForPath(r.manifest.Skaffold) {
		err := errors.Errorf("Missing Skaffold manifest %s", r.manifest.Skaffold)
		renderStepError(r.UI, step, renderStepLoadSkaffold, err)
		return err
	}
	step.Success("Skaffold manifest is available")
	return nil
}

func (r *RenderRunner) ValidateEnvSources(matchers []map[string]string) error {
	r.UI.Header("Validating deployment environments...")
	var detectHit bool

	filteredEnvs, err := r.manifest.GetEnvironments(r.config.envs)
	if err != nil {
		return err
	}

	for _, env := range filteredEnvs {
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

	if _, err := r.manifest.ReconcileConfig(r.config.envs...); err != nil {
		return err
	}

	return r.manifest.Environments.Write()
}

func (r *RenderRunner) RenderManifests() (map[string]string, error) {
	manifestFormat := r.config.manifestFormat
	r.UI.Header(fmt.Sprintf("Rendering manifests, format: %s...", manifestFormat))

	return r.manifest.RenderWithConvertor(
		converter.Factory(manifestFormat, r.UI),
		r.config.outputDir,
		r.config.manifestsAsSingleFile,
		r.config.envs,
		nil)
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

func printRenderProjectWithOptionsSuccess(ui kmd.UI, results map[string]string, envs Environments, manifestFormat string) {
	var namedValues []kmd.NamedValue
	for _, env := range envs {
		namedValues = append(namedValues, kmd.NamedValue{Name: env.Name, Value: results[env.Name]})
	}

	ui.Output("")
	ui.Output("Project manifests rendered!", kmd.WithStyle(kmd.SuccessBoldStyle))
	ui.Output(
		fmt.Sprintf("A set of '%s' manifests have been generated:", manifestFormat),
		kmd.WithStyle(kmd.SuccessStyle),
	)
	ui.NamedValues(namedValues, kmd.WithStyle(kmd.SuccessStyle))
	ui.Output("")
	ui.Output("The project can now be deployed to a Kubernetes cluster.", kmd.WithStyle(kmd.SuccessStyle))
	ui.Output("")
	ui.Output("To test locally:")
	ui.Output("Ensure you have a local cluster up and running with a configured context.", kmd.WithIndentChar("-"), kmd.WithIndent(1))
	ui.Output("Create a namespace: `kubectl create ns ns-example`.", kmd.WithIndentChar("-"), kmd.WithIndent(1))
	ui.Output("Apply the manifests to the cluster: `kubectl apply -f <manifests-dir>/<env> -n ns-example`.", kmd.WithIndentChar("-"), kmd.WithIndent(1))
	ui.Output("Discover the main service: `kubectl get svc -n ns-example`.", kmd.WithIndentChar("-"), kmd.WithIndent(1))
	ui.Output("Port forward to the main service: `kubectl port-forward service/<service_name> <service_port>:<destination_port> -n ns-example`.", kmd.WithIndentChar("-"), kmd.WithIndent(1))
}
