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

package tako

const (
	// SandboxEnv is a default environment name
	SandboxEnv = "dev"
)

var (
	// ManifestFilename is a name of main application manifest file
	ManifestFilename    = "tako.yaml"
	SecretsReferenceUrl = "https://github.com/appvia/tako/blob/master/docs/reference/config-params.md#reference-k8s-secret-key-value"
)

// InitProjectWithOptions initialises a Tako project in the specified working directory
// using the provided options (if any).
func InitProjectWithOptions(workingDir string, opts ...Options) error {
	runner := NewInitRunner(workingDir, opts...)
	ui := runner.UI

	results, err := runner.Run()
	if err != nil {
		printInitProjectWithOptionsError(runner.AppName, ui)
		return err
	}

	if err := results.Write(); err != nil {
		printInitProjectWithOptionsError(runner.AppName, ui)
		return err
	}

	return printInitProjectWithOptionsSuccess(runner, runner.manifest.Environments)
}

// RenderProjectWithOptions renders a Tako project's compose files into Kubernetes manifests
// using the provided options (if any).
func RenderProjectWithOptions(workingDir string, opts ...Options) error {
	runner := NewRenderRunner(workingDir, opts...)
	ui := runner.UI

	results, err := runner.Run()
	if err != nil {
		printRenderProjectWithOptionsError(runner.AppName, ui)
		return err
	}

	envs, err := runner.Manifest().GetEnvironments(runner.config.Envs)
	if err != nil {
		return err
	}

	return printRenderProjectWithOptionsSuccess(runner, results, envs)
}

// DevWithOptions runs a continuous development cycle detecting project updates and
// re-rendering compose files to Kubernetes manifests.
func DevWithOptions(workingDir string, opts ...Options) error {
	runner := NewDevRunner(workingDir, opts...)
	err := runner.Run()

	if err != nil {
		printDevProjectWithOptionsError(runner.AppName, runner.UI)
		return err
	}

	return nil
}

// PatchWithOptions patches kubernetes manifestes rendered with Tako and stored in the directory
// by substituting docker images referenced by specified compose service names. All mutations are performed in place.
func PatchWithOptions(workingDir string, opts ...Options) error {
	runner := NewPatchRunner(workingDir, opts...)
	ui := runner.UI

	if err := runner.Run(); err != nil {
		printPatchWithOptionsError(runner.AppName, ui)
		return err
	}

	return printPatchWithOptionsSuccess(runner)
}
