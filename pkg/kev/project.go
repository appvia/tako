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
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/appvia/kev/pkg/kev/config"
	"github.com/appvia/kev/pkg/kev/log"
	kmd "github.com/appvia/komando"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Init initialises the base project to be used in a runner
func (p *Project) Init(opts ...Options) {
	p.ctx = context.Background()
	p.SetConfig(opts...)

	if len(p.AppName) == 0 {
		p.AppName = config.AppName
	}

	if p.UI == nil {
		p.UI = kmd.ConsoleUI()
	}

	if p.LogVerbose() {
		log.SetLogLevel(logrus.DebugLevel)
	}
}

// ValidateSources includes validation checks to ensure the compose sources are valid.
// This function can be extended to include different forms of
// validation (for now it detect any secrets found in the sources).
func (p *Project) ValidateSources(sources *Sources, matchers []map[string]string) error {
	if err := p.eventHandler(PreValidateSources, p); err != nil {
		return newEventError(err, PreValidateSources)
	}

	p.UI.Header("Validating compose sources...")

	secretsDetected, err := p.detectSecretsInSources(sources, matchers)
	if err != nil {
		return err
	}

	p.UI.Output("")

	if secretsDetected {
		if err := p.eventHandler(SecretsDetected, p); err != nil {
			return newEventError(err, SecretsDetected)
		}
		p.UI.Output(fmt.Sprintf(`To prevent secrets leaking, see help page:
%s`, SecretsReferenceUrl))
	}

	if err := p.eventHandler(PostValidateSources, p); err != nil {
		return newEventError(err, PostValidateSources)
	}
	return nil
}

func (p *Project) detectSecretsInSources(sources *Sources, matchers []map[string]string) (bool, error) {
	var detected bool

	sg := p.UI.StepGroup()
	defer sg.Done()
	for _, composeFile := range sources.Files {
		p.UI.Output(fmt.Sprintf("Detecting secrets in: %s", composeFile))
		composeProject, err := NewComposeProject([]string{composeFile})
		if err != nil {
			decoratedErr := errors.Errorf("%s\nsee compose file: %s", err.Error(), composeFile)
			initStepError(p.UI, sg.Add(""), initStepParsingComposeConfig, decoratedErr)
			return false, decoratedErr
		}

		for _, s := range composeProject.Services {
			step := sg.Add(fmt.Sprintf("Analysing service: %s", s.Name))
			serviceConfig := ServiceConfig{Name: s.Name, Environment: s.Environment}

			hits := serviceConfig.detectSecretsInEnvVars(matchers)
			if len(hits) == 0 {
				step.Success("None detected in service: ", s.Name)
				continue
			}

			detected = true
			step.Warning("Detected in service: ", s.Name)

			for _, hit := range hits {
				p.UI.Output(
					fmt.Sprintf("env var [%s] - %s", hit.envVar, hit.description),
					kmd.WithStyle(kmd.LogStyle),
					kmd.WithIndentChar(kmd.LogIndentChar),
					kmd.WithIndent(3),
				)
			}
		}
	}
	return detected, nil
}

// Manifest returns the project's manifest
func (p *Project) Manifest() *Manifest {
	return p.manifest
}

// GetUI returns the project's UI
func (p *Project) GetUI() kmd.UI {
	return p.UI
}

// GetConfig returns a projects config
func (p *Project) GetConfig() runConfig {
	return *p.config
}

// SetConfig sets or overwrites params in a project's config using opts.
func (p *Project) SetConfig(opts ...Options) {
	cfg := &runConfig{}
	if p.config != nil {
		cfg = p.config
	}
	for _, o := range opts {
		o(p, cfg)
	}
	p.config = cfg
}

// LogVerbose indicates whether the project is running in verbose mode
func (p *Project) LogVerbose() bool {
	return p.config.LogVerbose
}

// pipeLogsToUI pipes all logs to configured UI
func (p *Project) pipeLogsToUI() (context.CancelFunc, *io.PipeReader, *io.PipeWriter) {
	pr, pw := io.Pipe()
	log.SetOutput(pw)
	ctx, cancelFunc := context.WithCancel(context.Background())

	go p.displayLogs(pr, ctx)
	return cancelFunc, pr, pw
}

// displayLogs displays logs streamed in from the provided reader
// until the provided context signals that it is done.
func (p *Project) displayLogs(reader io.Reader, ctx context.Context) {
	buf := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := reader.Read(buf)
			if err != nil {
				return
			}
			line := string(buf[:n])
			p.UI.Output(
				strings.TrimSuffix(line, "\n"),
				kmd.WithIndent(1),
				kmd.WithIndentChar(kmd.LogIndentChar),
				kmd.WithStyle(kmd.LogStyle),
			)
		}
	}
}

// WithAppName configures a project app name
func WithAppName(name string) Options {
	return func(project *Project, cfg *runConfig) {
		project.AppName = name
	}
}

// WithUI configures a project with a terminal UI implementation
func WithUI(ui kmd.UI) Options {
	return func(project *Project, cfg *runConfig) {
		project.UI = ui
	}
}

// WithEventHandler configures a project with an event handler
func WithEventHandler(handler EventHandler) Options {
	return func(project *Project, cfg *runConfig) {
		project.eventHandler = handler
	}
}

// WithComposeSources configures a project's run config with a list of compose files as sources.
func WithComposeSources(c []string) Options {
	return func(project *Project, cfg *runConfig) {
		cfg.ComposeSources = c
	}
}

// WithEnvs configures a project's run config with a list of environment names.
func WithEnvs(c []string) Options {
	return func(project *Project, cfg *runConfig) {
		cfg.Envs = c
	}
}

// WithSkaffold configures a project's run config with Skaffold support.
func WithSkaffold(c bool) Options {
	return func(project *Project, cfg *runConfig) {
		cfg.Skaffold = c
	}
}

// WithManifestFormat configures a project's run config with a K8s manifest format for rendering.
func WithManifestFormat(c string) Options {
	return func(project *Project, cfg *runConfig) {
		cfg.ManifestFormat = c
	}
}

// WithManifestsAsSingleFile configures a project's run config with whether rendered K8s manifests
// should be bundled into a single file or not.
func WithManifestsAsSingleFile(c bool) Options {
	return func(project *Project, cfg *runConfig) {
		cfg.ManifestsAsSingleFile = c
	}
}

// WithOutputDir configures a project's run config with a location to render a project's K8s manifests.
func WithOutputDir(c string) Options {
	return func(project *Project, cfg *runConfig) {
		cfg.OutputDir = c
	}
}

// WithK8sNamespace configures a project's run config with a K8s namespace
// (used mostly during dev when Skaffold is enabled).
func WithK8sNamespace(c string) Options {
	return func(project *Project, cfg *runConfig) {
		cfg.K8sNamespace = c
	}
}

// WithKubecontext configures a project's run config with a K8s kubecontext
// (used mostly during dev when Skaffold is enabled).
func WithKubecontext(c string) Options {
	return func(project *Project, cfg *runConfig) {
		cfg.Kubecontext = c
	}
}

// WithSkaffoldTailEnabled configures a project's run config with log tailing for Skaffold
// (used mostly during dev when Skaffold is enabled).
func WithSkaffoldTailEnabled(c bool) Options {
	return func(project *Project, cfg *runConfig) {
		cfg.SkaffoldTail = c
	}
}

// WithSkaffoldManualTriggerEnabled configures a project's run config with manual trigger
// for Skaffold (used mostly during dev when Skaffold is enabled).
func WithSkaffoldManualTriggerEnabled(c bool) Options {
	return func(project *Project, cfg *runConfig) {
		cfg.SkaffoldManualTrigger = c
	}
}

// WithSkaffoldVerboseEnabled configures a project's run config with verbose mode
// for Skaffold (used mostly during dev when Skaffold is enabled).
func WithSkaffoldVerboseEnabled(c bool) Options {
	return func(project *Project, cfg *runConfig) {
		cfg.SkaffoldVerbose = c
	}
}

// WithExcludeServicesByEnv configures a project's run config with environments whose
// services should be excluded from any processing.
func WithExcludeServicesByEnv(c map[string][]string) Options {
	return func(project *Project, cfg *runConfig) {
		cfg.ExcludeServicesByEnv = c
	}
}

// WithLogVerbose configures a project's run config to enable or disable verbose
// logging at a debug log level.
func WithLogVerbose(c bool) Options {
	return func(project *Project, cfg *runConfig) {
		cfg.LogVerbose = c
	}
}
