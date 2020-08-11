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
	composego "github.com/compose-spec/compose-go/types"
)

type Manifest struct {
	Sources      *Sources     `yaml:"compose,omitempty" json:"compose,omitempty"`
	Environments Environments `yaml:"environments,omitempty" json:"environments,omitempty"`
}

type Sources struct {
	Files  []string `yaml:"-" json:"-"`
	labels *labels
}

type Environments []*Environment

type Environment struct {
	Name   string `yaml:"-" json:"-"`
	File   string `yaml:"-" json:"-"`
	labels *labels
}

type labels struct {
	Version  string   `yaml:"version,omitempty" json:"version,omitempty" diff:"version"`
	Services Services `json:"services" diff:"services"`
	Volumes  Volumes  `yaml:",omitempty" json:"volumes,omitempty" diff:"volumes"`
}

type ComposeProject struct {
	version string
	*composego.Project
}

type ServiceConfig struct {
	Name        string             `yaml:"-" json:"-" diff:"name"`
	Labels      composego.Labels   `yaml:",omitempty" json:"labels,omitempty" diff:"labels"`
	Environment map[string]*string `yaml:",omitempty" json:"environment,omitempty" diff:"environment"`
}

// Services is a list of ServiceConfig
type Services []ServiceConfig

type Volumes map[string]VolumeConfig

type VolumeConfig struct {
	Name   string           `yaml:",omitempty" json:"name,omitempty" diff:"name"`
	Labels composego.Labels `yaml:",omitempty" json:"labels,omitempty" diff:"labels"`
}

type changeset struct {
	version  []change
	services changeGroup
	volumes  changeGroup
}

type changeGroup map[interface{}][]change

type change struct {
	parent, target, value  string
	index                  interface{}
	update, create, delete bool
}
