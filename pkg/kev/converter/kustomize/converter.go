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

package kustomize

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/appvia/kev/pkg/kev/log"
	kmd "github.com/appvia/komando"
	composego "github.com/compose-spec/compose-go/types"
	"github.com/kubernetes-sigs/kustomize/pkg/types"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	// Name of the converter
	Name                  = "kustomize"
	singleFileDefaultName = "k8s.yaml"

	// MultiFileSubDir is default output directory name for kubernetes manifests
	MultiFileSubDir         = "kustomize"
	KustomizeBaseFolderName = "base"
)

// K8s is a native kubernetes manifests converter
type K8s struct {
	UI kmd.UI
}

// New return a native Kubernetes converter
func New() *K8s {
	return &K8s{}
}

func NewWithUI(ui kmd.UI) *K8s {
	return &K8s{UI: ui}
}

// Render generates outcome
func (c *K8s) Render(singleFile bool,
	dir, workDir string,
	projects map[string]*composego.Project,
	files map[string][]string,
	rendered map[string][]byte,
	excluded map[string][]string) (map[string]string, error) {

	renderOutputPaths := map[string]string{}
	envs := getSortedEnvs(projects)

	var outputFolder string
	if dir != "" {
		outputFolder = dir
	} else {
		outputFolder = filepath.Join(workDir, MultiFileSubDir)
	}

	err := ensureFolder(filepath.Join(outputFolder, KustomizeBaseFolderName))
	if err != nil {
		return nil, err
	}

	env := KustomizeBaseFolderName
	project := projects["dev"]

	log.Debugf("Rendering environment [%s]", env)

	envFile := files["dev"][len(files["dev"])-1]
	c.UI.Output(fmt.Sprintf("%s: %s", env, envFile))

	outEnvPath := filepath.Join(outputFolder, env)

	// @step generate multiple files - single file output not supported for Kustomize
	outFilePath := outEnvPath

	// @step kubernetes manifests output options
	convertOpts := ConvertOptions{
		InputFiles: files["dev"],
		OutFile:    outFilePath,
	}

	renderOutputPaths[env] = outFilePath

	// @step set excluded docker compose services for current project
	exc := []string{}
	if excluded != nil {
		if e, ok := excluded[env]; ok {
			exc = e
		}
	}

	// @step Get Kubernetes transformer that maps compose project to Kubernetes primitives
	k := &Kustomize{Opt: convertOpts, Project: project, Excluded: exc, UI: c.UI}

	// @step Do the transformation
	objects, err := k.Transform()
	if err != nil {
		return nil, err
	}

	// @step Produce objects
	err = PrintList(objects, convertOpts, rendered)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not render %s manifests to disk, details:\n", Name)
	}

	for _, env := range envs {
		project := projects[env]

		log.Debugf("Rendering environment [%s]", env)

		fmt.Println(files)

		envFile := files[env][len(files[env])-1]
		c.UI.Output(fmt.Sprintf("%s: %s", env, envFile))

		outEnvPath := filepath.Join(outputFolder, env)
		err := ensureFolder(outEnvPath)
		if err != nil {
			return nil, err
		}

		// @step generate multiple files - single file output not supported for Kustomize
		outFilePath := outEnvPath

		// @step kubernetes manifests output options
		convertOpts := ConvertOptions{
			InputFiles:  files[env],
			OutFile:     outFilePath,
			Environment: env,
		}

		renderOutputPaths[env] = outFilePath

		// @step set excluded docker compose services for current project
		exc := []string{}
		if excluded != nil {
			if e, ok := excluded[env]; ok {
				exc = e
			}
		}

		// @step Get Kubernetes transformer that maps compose project to Kubernetes primitives
		k := &Kustomize{Opt: convertOpts, Project: project, Excluded: exc, UI: c.UI}

		err = k.GenerateOverlay()
		if err != nil {
			return nil, err
		}
	}

	return renderOutputPaths, nil
}

func (k *Kustomize) GenerateOverlay() error {
	kustomization := &types.Kustomization{}
	kustomization.Bases = []string{"../base"}
	kustomization.NamePrefix = fmt.Sprintf("%s-", k.Opt.Environment)
	kustomization.GeneratorOptions = &types.GeneratorOptions{
		DisableNameSuffixHash: true,
	}

	bytes, err := yaml.Marshal(kustomization)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(k.Opt.OutFile, "kustomization.yaml"), bytes, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func ensureFolder(folderPath string) error {
	// @step create output directory
	// To generate outcome as a set of separate manifests first must create out directory
	// as Kompose logic checks for this and only will do that for existing directories,
	// otherwise will treat OutFile as regular file and output all manifests to that single file.
	if err := os.MkdirAll(folderPath, os.ModePerm); err != nil {
		return err
	}

	return nil
}

func getSortedEnvs(projects map[string]*composego.Project) []string {
	var out []string
	for env := range projects {
		out = append(out, env)
	}
	sort.Strings(out)
	return out
}
