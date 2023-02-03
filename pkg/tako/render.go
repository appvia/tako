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

package tako

import (
	"fmt"
	"path/filepath"

	kmd "github.com/appvia/komando"
	"github.com/appvia/tako/pkg/tako/config"
	"github.com/appvia/tako/pkg/tako/converter"
	"github.com/pkg/errors"
)

// NewRenderRunner creates a render runner instance
func NewRenderRunner(workingDir string, opts ...Options) *RenderRunner {
	runner := &RenderRunner{
		Project: &Project{
			WorkingDir:   workingDir,
			eventHandler: func(s RunnerEvent, r Runner) error { return nil },
		},
	}
	runner.Init(opts...)
	return runner
}

// Run executes the runner returning results that can be written to disk
func (r *RenderRunner) Run() (map[string]string, error) {
	if r.LogVerbose() {
		cancelFunc, pr, pw := r.pipeLogsToUI()
		defer cancelFunc()
		defer pw.Close()
		defer pr.Close()
	}

	if err := r.LoadProject(); err != nil {
		return nil, err
	}

	if err := r.ValidateSources(r.manifest.Sources, config.SecretMatchers); err != nil {
		sg := r.UI.StepGroup()
		defer sg.Done()
		renderStepError(r.UI, sg.Add(""), renderStepValidatingSources, err)
		return nil, err
	}

	if err := r.VerifySkaffoldIfAvailable(); err != nil {
		return nil, err
	}

	if err := r.ValidateEnvSources(config.SecretMatchers); err != nil {
		return nil, err
	}

	if err := r.ReconcileEnvsAndWriteUpdates(); err != nil {
		return nil, err
	}

	results, err := r.RenderFromComposeToK8sManifests()

	return results, err
}

// LoadProject loads the project into memory including the tako manifest and related deployment environments.
func (r *RenderRunner) LoadProject() error {
	if err := r.eventHandler(PreLoadProject, r); err != nil {
		return newEventError(err, PreLoadProject)
	}
	r.UI.Header("Loading...")

	sg := r.UI.StepGroup()
	defer sg.Done()

	if !ManifestExistsForPath(filepath.Join(r.WorkingDir, ManifestFilename)) {
		err := errors.Errorf("Missing project manifest: %s", ManifestFilename)
		renderStepError(r.UI, sg.Add(""), renderStepLoad, err)
		return err
	}

	manifest, err := LoadManifest(r.WorkingDir)
	if err != nil {
		renderStepError(r.UI, sg.Add(""), renderStepLoad, err)
		return err
	}
	r.manifest = manifest
	r.manifest.UI = r.UI
	if err := r.eventHandler(PostLoadProject, r); err != nil {
		return newEventError(err, PostLoadProject)
	}

	return nil
}

// VerifySkaffoldIfAvailable ensures if a project was initialised with Skaffold support,
// that the configured Skaffold manifest does exist.
func (r *RenderRunner) VerifySkaffoldIfAvailable() error {
	if len(r.manifest.Skaffold) == 0 {
		return nil
	}

	if err := r.eventHandler(PreVerifySkaffold, r); err != nil {
		return newEventError(err, PreVerifySkaffold)
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

	if err := r.eventHandler(PostVerifySkaffold, r); err != nil {
		return newEventError(err, PostVerifySkaffold)
	}
	return nil
}

// ValidateEnvSources includes validation checks to ensure the deployment environments' compose sources are valid.
// This function can be extended to include different forms of
// validation (for now it detect any secrets found in the sources).
func (r *RenderRunner) ValidateEnvSources(matchers []map[string]string) error {
	if err := r.eventHandler(PreValidateEnvSources, r); err != nil {
		return newEventError(err, PreValidateEnvSources)
	}

	r.UI.Header("Validating compose environment overrides...")
	var detectHit bool

	filteredEnvs, err := r.manifest.GetEnvironments(r.config.Envs)
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
	if detectHit {
		if err := r.eventHandler(SecretsDetected, r); err != nil {
			return newEventError(err, SecretsDetected)
		}
		r.UI.Output(fmt.Sprintf(`To prevent secrets leaking, see help page:
%s`, SecretsReferenceUrl))
	}

	if err := r.eventHandler(PostValidateEnvSources, r); err != nil {
		return newEventError(err, PostValidateEnvSources)
	}
	return nil
}

// ReconcileEnvsAndWriteUpdates reconciles changes with docker-compose sources against deployment environments.
func (r *RenderRunner) ReconcileEnvsAndWriteUpdates() error {
	if err := r.eventHandler(PreReconcileEnvs, r); err != nil {
		return newEventError(err, PreReconcileEnvs)
	}

	r.UI.Header("Detecting project updates...")

	if _, err := r.manifest.ReconcileConfig(r.config.Envs...); err != nil {
		return err
	}

	if err := r.manifest.Environments.Write(); err != nil {
		sg := r.UI.StepGroup()
		defer sg.Done()
		renderStepError(r.UI, sg.Add(""), renderStepReconcileWrite, err)
		return err
	}

	if err := r.eventHandler(PostReconcileEnvs, r); err != nil {
		return newEventError(err, PostReconcileEnvs)
	}

	return nil
}

// RenderFromComposeToK8sManifests renders K8s manifests using the project's
// compose sources and deployment environments as the source. K8s manifests can rendered
// in different formats.
func (r *RenderRunner) RenderFromComposeToK8sManifests() (map[string]string, error) {
	if err := r.eventHandler(PreRenderFromComposeToK8sManifests, r); err != nil {
		return nil, newEventError(err, PreRenderFromComposeToK8sManifests)
	}

	manifestFormat := r.config.ManifestFormat
	r.UI.Header(fmt.Sprintf("Rendering manifests, format: %s...", manifestFormat))

	results, err := r.manifest.RenderWithConvertor(converter.Factory(manifestFormat, r.UI), r.config)
	if err != nil {
		return nil, err
	}

	if err := r.eventHandler(PostRenderFromComposeToK8sManifests, r); err != nil {
		return nil, newEventError(err, PostRenderFromComposeToK8sManifests)
	}
	return results, err
}

func printRenderProjectWithOptionsError(appName string, ui kmd.UI) {
	ui.Output("")
	ui.Output("Project had errors during render.\n"+
		fmt.Sprintf("'%s' experienced some errors during project render. The output\n", appName)+
		"above should contain the failure messages. Please correct these errors and\n"+
		fmt.Sprintf("run '%s render' again.", appName),
		kmd.WithErrorBoldStyle(),
		kmd.WithIndentChar(kmd.ErrorIndentChar),
	)
}

func printRenderProjectWithOptionsSuccess(r *RenderRunner, results map[string]string, envs Environments) error {
	var namedValues []kmd.NamedValue
	for _, env := range envs {
		namedValues = append(namedValues, kmd.NamedValue{Name: env.Name, Value: results[env.Name]})
	}

	ui := r.GetUI()
	ui.Output("")
	ui.Output("Project manifests rendered!", kmd.WithStyle(kmd.SuccessBoldStyle))

	if err := r.eventHandler(PrePrintSummary, r); err != nil {
		return newEventError(err, PrePrintSummary)
	}

	ui.Output(
		fmt.Sprintf("A set of '%s' manifests have been generated:", r.config.ManifestFormat),
		kmd.WithStyle(kmd.SuccessStyle),
	)
	ui.NamedValues(namedValues, kmd.WithStyle(kmd.SuccessStyle))

	if err := r.eventHandler(PostPrintSummary, r); err != nil {
		return newEventError(err, PostPrintSummary)
	}

	ui.Output("")
	ui.Output("The project can now be deployed to a Kubernetes cluster.", kmd.WithStyle(kmd.SuccessStyle))
	ui.Output("")
	ui.Output("To test locally:")
	ui.Output("Ensure you have a local cluster up and running with a configured context.", kmd.WithIndentChar("-"), kmd.WithIndent(1))
	ui.Output("Create a namespace: `kubectl create ns ns-example`.", kmd.WithIndentChar("-"), kmd.WithIndent(1))
	ui.Output("Apply the manifests to the cluster: `kubectl apply -f <manifests-dir>/<env> -n ns-example`.", kmd.WithIndentChar("-"), kmd.WithIndent(1))
	ui.Output("Discover the main service: `kubectl get svc -n ns-example`.", kmd.WithIndentChar("-"), kmd.WithIndent(1))
	ui.Output("Port forward to the main service: `kubectl port-forward service/<service_name> <service_port>:<destination_port> -n ns-example`.", kmd.WithIndentChar("-"), kmd.WithIndent(1))

	return nil
}
