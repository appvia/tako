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

import (
	"context"
	"io"

	kmd "github.com/appvia/komando"
	composego "github.com/compose-spec/compose-go/types"
)

// runConfig stores configuration for a command
type runConfig struct {
	// ComposeSources is a list of compose files to use
	ComposeSources []string
	// Envs is a list of environments to use
	Envs []string
	// ManifestFormat is a format of the output manifests
	ManifestFormat string
	// ManifestsAsSingleFile indicates whether to render all manifests into a single file
	ManifestsAsSingleFile bool
	// AdditionalManifests is a list of additional manifests that should be added to the generated manifests set
	AdditionalManifests []string
	// OutputDir is a directory where to store the generated manifests
	OutputDir string
	// K8sNamespace is a target Kubernetes namespace
	K8sNamespace string
	// KubeContext is a target Kubernetes cluster context
	Kubecontext string
	// Skaffold is a flag indicating whether to generate skaffold.yaml
	Skaffold bool
	// SkaffoldTail is a flag indicating whether to tail skaffold logs
	SkaffoldTail bool
	// SkaffoldManualTrigger is a flag indicating whether trigger changes manually when skaffold dev loop is running
	SkaffoldManualTrigger bool
	// SkaffoldVerbose is a flag indicating whether to enable verbose logging for skaffold
	SkaffoldVerbose bool
	// ExcludeServicesByEnv is used to exclude an environment's set of services from processing.
	// Primary use is during render.
	ExcludeServicesByEnv map[string][]string
	// LogVerbose enables/disables verbose logging at a debug log level.
	LogVerbose bool
	// PatchManifestsDir is a directory where previously generated manifests that should be patched are stored.
	PatchManifestsDir string
	// PatchImages is a list of images that should be used when patching existing manifests.
	PatchImages []string
	// PatchOutputDir is a directory where patched manifests should be stored.
	// Output directory structure will reflect that of the source directory tree.
	// If patch output directory is not specified then manifests will be overriden in the source directory.
	PatchOutputDir string
}

// Options helps configure running project commands
type Options func(project *Project, cfg *runConfig)

// RunnerEvent a runner event.
// This could be a pre/post runner step hook or a general significant event.
// E.g.
// - Pre step event: PreLoadProject
// - post step event: PostLoadProject
// - significant event: SecretsDetected
type RunnerEvent uint

// EventHandler is a callback function that handles a runner event
type EventHandler func(RunnerEvent, Runner) error

// Runner an interface used by the EventHandler
type Runner interface {
	Manifest() *Manifest
	GetUI() kmd.UI
	GetConfig() runConfig
	SetConfig(opts ...Options)
}

// Project is the base struct for all runners.
// Runners must initialise a project using Init().
type Project struct {
	// AppName is the application name.
	AppName string
	// WorkingDir is the working directory.
	WorkingDir string
	// UI is the user interface.
	UI kmd.UI
	// manifest is the project manifest containing information on source compose files, environments etc.
	manifest *Manifest
	// config is the project configuration.
	config *runConfig
	// eventHandler is the event handler.
	eventHandler EventHandler
	ctx          context.Context
}

// InitRunner runs the required sequences to initialise a project.
type InitRunner struct {
	*Project
}

// RenderRunner runs the required sequences to render a project.
type RenderRunner struct {
	*Project
}

// DevRunner runs the required sequences to use dev with a project.
type DevRunner struct {
	*Project
}

// PatchRunner runs the required sequences to use patch with a project.
type PatchRunner struct {
	*Project
}

// Manifest contains the tracked project's docker-compose sources and deployment environments
type Manifest struct {
	Id           string       `yaml:"id,omitempty" json:"id,omitempty"`
	Sources      *Sources     `yaml:"compose,omitempty" json:"compose,omitempty"`
	Environments Environments `yaml:"environments,omitempty" json:"environments,omitempty"`
	Skaffold     string       `yaml:"skaffold,omitempty" json:"skaffold,omitempty"`
	UI           kmd.UI       `yaml:"-" json:"-"`
}

// Sources tracks a project's docker-compose sources
type Sources struct {
	Files    []string `yaml:"-" json:"-"`
	override *composeOverride
}

// Environments tracks a project's deployment environments
type Environments []*Environment

// Environment is a deployment environment
type Environment struct {
	Name     string `yaml:"-" json:"-"`
	File     string `yaml:"-" json:"-"`
	override *composeOverride
}

// composeOverride augments a compose project with an extension and env vars to produce
// k8s deployment config
type composeOverride struct {
	Version  string   `yaml:"version,omitempty" json:"version,omitempty" diff:"version"`
	Services Services `json:"services" diff:"services"`
	Volumes  Volumes  `yaml:",omitempty" json:"volumes,omitempty" diff:"volumes"`
	UI       kmd.UI   `yaml:"-" json:"-"`
}

// ComposeProject wrapper around a compose-go Project. It also provides the original
// compose file version.
type ComposeProject struct {
	version string
	*composego.Project
}

// ServiceConfig is a shallow version of a compose-go ServiceConfig
type ServiceConfig struct {
	Name        string                      `yaml:"-" json:"-" diff:"name"`
	Image       string                      `yaml:"image,omitempty" json:"-" diff:"image"`
	Environment composego.MappingWithEquals `yaml:",omitempty" json:"environment,omitempty" diff:"environment"`
	Extensions  map[string]interface{}      `yaml:",inline" json:"-"`
}

type secretHit struct {
	svcName     string
	envVar      string
	description string
}

// Services is a list of ServiceConfig
type Services []ServiceConfig

// Volumes is a mapping of volume name to VolumeConfig
type Volumes map[string]VolumeConfig

// VolumeConfig is a shallow version of a compose-go VolumeConfig
type VolumeConfig struct {
	Name       string                 `yaml:",omitempty" json:"name,omitempty" diff:"name"`
	Extensions map[string]interface{} `yaml:",inline" json:"-"`
}

// changeset tracks changes made to a version, services and volumes
type changeset struct {
	version  change
	services []change
	volumes  []change
}

// change describes a create, update or delete modification
// targeting an attribute in a version, service or volume.
type change struct {
	Type   string
	Value  interface{}
	Parent string
	Target string
	Index  interface{}
}

// WritableResults is a collection of WritableResult
type WritableResults []WritableResult

// WritableResult used to return results that can be written out to disk
type WritableResult struct {
	WriterTo io.WriterTo
	FilePath string
}
