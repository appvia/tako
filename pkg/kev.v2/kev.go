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
	"fmt"
	"path"
	"path/filepath"
)

const (
	ManifestName = "kev.yaml"

	defaultEnv         = "dev"
	configFileTemplate = "docker-compose.kev.%s.yaml"
)

var tempConfig = `version: '3.7'
services:
  db:
    labels:
      io.appvia.kev.workload.image-pull-policy: "IfNotPresent"
      io.appvia.kev.workload.service-account-name: "default"
      io.appvia.kev.workload.type: "StatefulSet"
      io.appvia.kev.workload.replicas: "1"
      io.appvia.kev.workload.rolling-update-max-surge: "1"
      io.appvia.kev.workload.cpu: "0.1"
      io.appvia.kev.workload.memory: "10Mi"
      io.appvia.kev.workload.max-cpu: "0.5"
      io.appvia.kev.workload.max-memory: "500Mi"
      io.appvia.kev.workload.liveness-probe-disable: "false"
      io.appvia.kev.workload.liveness-probe-interval: "1m0s"
      io.appvia.kev.workload.liveness-probe-retries: "3"
      io.appvia.kev.workload.liveness-probe-initial-delay: "1m0s"
      io.appvia.kev.workload.liveness-probe-command: '["CMD", "echo", "Define healthcheck command for service db"]'
      io.appvia.kev.workload.liveness-probe-timeout: "10s"
      io.appvia.kev.service.type: "None"
  wordpress:
    labels:
      io.appvia.kev.workload.image-pull-policy: "IfNotPresent"
      io.appvia.kev.workload.service-account-name: "default"
      io.appvia.kev.workload.type: "Deployment"
      io.appvia.kev.workload.replicas: "1"
      io.appvia.kev.workload.rolling-update-max-surge: "1"
      io.appvia.kev.workload.cpu: "0.1"
      io.appvia.kev.workload.memory: "10Mi"
      io.appvia.kev.workload.max-cpu: "0.5"
      io.appvia.kev.workload.max-memory: "500Mi"
      io.appvia.kev.workload.liveness-probe-disable: "false"
      io.appvia.kev.workload.liveness-probe-interval: "1m0s"
      io.appvia.kev.workload.liveness-probe-retries: "3"
      io.appvia.kev.workload.liveness-probe-initial-delay: "1m0s"
      io.appvia.kev.workload.liveness-probe-command: '["CMD", "echo", "Define healthcheck command for service wordpress"]'
      io.appvia.kev.workload.liveness-probe-timeout: "10s"
      io.appvia.kev.service.type: "LoadBalancer"
volumes:
  db_data:
    labels:
      io.appvia.kev.volumes.class: "standard"
      io.appvia.kev.volumes.size: "100Mi"
`

// Init initialises a kev manifest including source compose files and environments.
// A default environment will be allocated if no environments were provided.
func Init(composeFiles, envs []string) (*Manifest, error) {
	configWorkingDir := filepath.Dir(composeFiles[0])

	environments := Environments{}
	if len(envs) == 0 {
		envs = append(envs, defaultEnv)
	}
	for _, env := range envs {
		environments = append(environments, Environment{
			Name:    env,
			Content: []byte(tempConfig),
			File:    path.Join(configWorkingDir, fmt.Sprintf(configFileTemplate, env)),
		})
	}

	return &Manifest{
		Sources:      composeFiles,
		Environments: environments,
	}, nil
}
