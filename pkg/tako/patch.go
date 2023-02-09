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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	kmd "github.com/appvia/komando"
	"github.com/appvia/tako/pkg/tako/log"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"sigs.k8s.io/yaml"
)

// NewPatchRunner creates a patch runner instance
func NewPatchRunner(workingDir string, opts ...Options) *PatchRunner {
	runner := &PatchRunner{
		Project: &Project{
			WorkingDir:   workingDir,
			eventHandler: func(s RunnerEvent, r Runner) error { return nil },
		},
	}
	runner.Init(opts...)
	return runner
}

// Run executes the runner returning results that can be written to disk
func (r *PatchRunner) Run() error {
	if r.LogVerbose() {
		cancelFunc, pr, pw := r.pipeLogsToUI()
		defer cancelFunc()
		defer pw.Close()
		defer pr.Close()
	}

	return r.PatchManifests()
}

// PatchManifests patches the deployment manifests for the specified services
func (r *PatchRunner) PatchManifests() error {

	r.UI.Header("Patching deployment manifests for specified services...")

	if err := r.eventHandler(PrePatchManifest, r); err != nil {
		return newEventError(err, PrePatchManifest)
	}

	for _, serviceImage := range r.config.PatchImages {

		// each patch image is supplied as a service=image pair
		// e.g. app=appvia/tako:latest
		parts := strings.Split(serviceImage, "=")
		svc, img := parts[0], parts[1]

		sg := r.UI.StepGroup()
		defer sg.Done()

		step := sg.Add(fmt.Sprintf("Patching service %s with image %s", svc, img))

		if err := r.patch(svc, img); err != nil {
			err := errors.Errorf("Failed to patch `%s` service image", svc)
			patchStepError(r.UI, step, patchStepPatchImages, err)
			return err
		}

		step.Success(fmt.Sprintf("Patched service %s with image %s", svc, img))
	}

	if err := r.eventHandler(PostPatchManifest, r); err != nil {
		return newEventError(err, PostPatchManifest)
	}

	return nil
}

// patch patches the deployment manifest for the specified service with the specified image
func (r *PatchRunner) patch(svc, img string) error {

	// walk the tree
	err := filepath.Walk(r.config.PatchManifestsDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() {

				f, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				j, err := yaml.YAMLToJSON(f)
				if err != nil {
					return err
				}

				kind := gjson.Get(string(j), "kind").String()
				name := gjson.Get(string(j), "metadata.name").String()

				// We only patch images in previously generated deployments, statefulsets and daemonsets!
				if kind != "Deployment" && kind != "StatefulSet" && kind != "DaemonSet" {
					return nil
				}

				// Check whether we match the service name
				if name != svc {
					return nil
				}

				jstring := string(j)

				jstring, err = sjson.Set(jstring, "spec.template.spec.containers.0.image", img)
				if err != nil {
					return err
				}

				y, err := yaml.JSONToYAML([]byte(jstring))
				if err != nil {
					return err
				}

				if len(r.config.PatchOutputDir) > 0 {
					// Write the patched file to specified output directory preserving the tree structure of the source directory.
					// e.g. if the output directory is /tmp/patched and source directory is /tmp/manifests and the file is /tmp/manifests/foo/bar/deployment.yaml
					// then the patched file will be written to /tmp/patched/foo/bar/deployment.yaml

					relPath, err := filepath.Rel(r.config.PatchManifestsDir, path)
					if err != nil {
						return err
					}

					// Create the directory if it doesn't exist
					err = os.MkdirAll(filepath.Join(r.config.PatchOutputDir, filepath.Dir(relPath)), 0755)
					if err != nil {
						return err
					}

					err = os.WriteFile(filepath.Join(r.config.PatchOutputDir, relPath), y, 0644)
					if err != nil {
						return err
					}

				} else {
					// Override the original file
					err = os.WriteFile(path, y, 0644)
					if err != nil {
						return err
					}
				}

			}

			return nil
		})

	if err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func printPatchWithOptionsError(appName string, ui kmd.UI) {
	ui.Output("")
	ui.Output("Project had errors during patch.\n"+
		fmt.Sprintf("'%s' experienced some errors while running patch. The output\n", appName)+
		"above should contain the failure messages. Please correct these errors and\n"+
		fmt.Sprintf("run '%s patch' again.", appName),
		kmd.WithErrorBoldStyle(),
		kmd.WithIndentChar(kmd.ErrorIndentChar),
	)
}

func printPatchWithOptionsSuccess(r *PatchRunner) error {
	ui := r.GetUI()

	if err := r.eventHandler(PrePrintSummary, r); err != nil {
		return newEventError(err, PrePrintSummary)
	}

	ui.Output("")
	ui.Output("Project manifests patched successfully!", kmd.WithStyle(kmd.SuccessBoldStyle))

	if err := r.eventHandler(PostPrintSummary, r); err != nil {
		return newEventError(err, PostPrintSummary)
	}

	ui.Output("")
	ui.Output("The project can now be deployed to a Kubernetes cluster.", kmd.WithStyle(kmd.SuccessStyle))
	ui.Output("")
	return nil
}
