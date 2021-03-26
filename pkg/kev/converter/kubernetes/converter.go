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

package kubernetes

import (
	"fmt"
	"os"
	"path"
	"sort"

	"github.com/appvia/kev/pkg/kev/log"
	kmd "github.com/appvia/komando"
	composego "github.com/compose-spec/compose-go/types"
)

const (
	// Name of the converter
	Name                  = "kubernetes"
	singleFileDefaultName = "k8s.yaml"

	// MultiFileSubDir is default output directory name for kubernetes manifests
	MultiFileSubDir = "k8s"
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
	// for env, project := range projects {
	for _, env := range envs {
		project := projects[env]

		log.Debugf("Rendering environment [%s]", env)

		c.UI.Output(fmt.Sprintf("%s: %s", env, getEnvFile(env, files)))

		// @step override output directory if specified
		outDirPath := ""
		if dir != "" {
			// adding env name suffix to the custom directory to differentiate
			outDirPath = path.Join(dir, env)
		} else {
			outDirPath = path.Join(workDir, MultiFileSubDir, env)
		}

		// @step create output directory
		// To generate outcome as a set of separate manifests first must create out directory
		// as Kompose logic checks for this and only will do that for existing directories,
		// otherwise will treat OutFile as regular file and output all manifests to that single file.
		if err := os.MkdirAll(outDirPath, os.ModePerm); err != nil {
			return nil, err
		}

		// @step generate multiple / single file
		outFilePath := ""
		if singleFile {
			outFilePath = path.Join(outDirPath, singleFileDefaultName)
		} else {
			outFilePath = outDirPath
		}

		// @step kubernetes manifests output options
		convertOpts := ConvertOptions{
			InputFiles: files[env],
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
		k := &Kubernetes{Opt: convertOpts, Project: project, Excluded: exc, UI: c.UI}

		// @step Do the transformation
		objects, err := k.Transform()
		if err != nil {
			return nil, err
		}

		// @step Produce objects
		err = PrintList(objects, convertOpts, rendered)
		if err != nil {
			return nil, err
		}
	}

	return renderOutputPaths, nil
}

func getSortedEnvs(projects map[string]*composego.Project) []string {
	var out []string
	for env, _ := range projects {
		out = append(out, env)
	}
	sort.Strings(out)
	return out
}

func getEnvFile(env string, files map[string][]string) string {
	sources, ok := files[env]
	if !ok {
		return ""
	}
	return sources[len(sources)-1]
}
