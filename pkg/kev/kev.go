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
	"context"
	"fmt"
	"os"

	"github.com/appvia/kev/pkg/kev/config"
	"github.com/appvia/kev/pkg/kev/converter"
	"github.com/appvia/kev/pkg/kev/log"
	kmd "github.com/appvia/komando"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

const (
	// SandboxEnv is a default environment name
	SandboxEnv = "dev"
)

var (
	// ManifestFilename is a name of main application manifest file
	ManifestFilename    = "kev.yaml"
	SecretsReferenceUrl = "https://github.com/appvia/kev/blob/master/docs/reference/config-params.md#reference-k8s-secret-key-value"
)

// InitProjectWithOptions initialises a kev project in the specified working directory
// using the provided options (if any).
func InitProjectWithOptions(workingDir string, opts ...Options) error {
	runner := NewInitRunner(workingDir, opts...)
	ui := runner.UI

	results, err := runner.Run()
	if err != nil {
		printInitProjectWithOptionsError(ui)
		return err
	}

	if err := results.Write(); err != nil {
		printInitProjectWithOptionsError(ui)
		return err
	}

	printInitProjectWithOptionsSuccess(ui, runner.manifest.Environments)
	return nil
}

func RenderProjectWithOptions(workingDir string, opts ...Options) error {
	runner := NewRenderRunner(workingDir, opts...)
	ui := runner.UI

	err := runner.Run()
	if err != nil {
		printRenderProjectWithOptionsError(ui)
		return err
	}

	return nil
}

// Reconcile reconciles changes with docker-compose sources against deployment environments.
func Reconcile(workingDir string) (*Manifest, error) {
	m, err := LoadManifest(workingDir)
	if err != nil {
		return nil, err
	}

	// TODO(es) Remove this after render runner is completed
	m.UI = kmd.NoOpUI()

	if _, err := m.ReconcileConfig(); err != nil {
		return nil, errors.Wrap(err, "Could not reconcile project latest")
	}
	return m, err
}

// DetectSecrets detects any potential secrets defined in environment variables
// found either in sources or override environments.
// Any detected secrets are logged using a warning log level.
func DetectSecrets(workingDir string) error {
	m, err := LoadManifest(workingDir)
	if err != nil {
		return err
	}

	runner := &InitRunner{Project: &Project{workingDir: workingDir}}
	runner.Init()
	if _, err := runner.detectSecretsInSources(m.Sources, config.SecretMatchers); err != nil {
		return err
	}

	if err := m.DetectSecretsInEnvs(config.SecretMatchers); err != nil {
		return err
	}
	return nil
}

// Render renders k8s manifests for a kev app. It returns an app definition with rendered manifest info
// It takes optional exclusion list as map of environment name to a slice of excluded docker compose service names.
func Render(workingDir string, format string, singleFile bool, dir string, envs []string, excluded map[string][]string) error {
	manifest, err := LoadManifest(workingDir)
	if err != nil {
		return errors.Wrap(err, "Unable to load app manifest")
	}

	_, err = manifest.RenderWithConvertor(converter.Factory(format, nil), dir, singleFile, envs, excluded)
	return err
}

// Watch continuously watches source compose files & environment overrides and notifies changes to a channel
func Watch(workDir string, change chan<- string) error {
	manifest, err := LoadManifest(workDir)
	if err != nil {
		log.Errorf("Unable to load app manifest - %s", err)
		os.Exit(1)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer watcher.Close()

	done := make(chan bool)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op&fsnotify.Write == fsnotify.Write {
					change <- event.Name
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}

				log.Error(err)
			}
		}
	}()

	files := manifest.GetSourcesFiles()
	filteredEnvs, err := manifest.GetEnvironments([]string{})
	for _, e := range filteredEnvs {
		files = append(files, e.File)
	}

	for _, f := range files {
		err = watcher.Add(f)
		if err != nil {
			return err
		}
	}

	<-done

	return nil
}

// Dev contains dev command business logic
func Dev(opts *DevOptions, workDir string, preRunCommands []RunFunc, errHandler ErrorHandler, changeHandler ChangeHandler) error {

	runPreCommands := func() error {
		for _, preRunCmd := range preRunCommands {
			if err := preRunCmd(); err != nil {
				return err
			}
		}
		return nil
	}

	change := make(chan string, 50)
	defer close(change)

	// initial manifests generation for specified environments only
	if err := runPreCommands(); err != nil {
		return errHandler(err)
	}

	if opts.Skaffold {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		catchCtrlC(cancel)

		skaffoldConfigPath, skaffoldConfig, err := ActivateSkaffoldDevLoop(workDir)
		if err != nil {
			return errHandler(err)
		}

		if err := WriteTo(skaffoldConfigPath, skaffoldConfig); err != nil {
			return errHandler(errors.Wrap(err, "Couldn't write Skaffold config"))
		}

		profileName := opts.Kevenv + EnvProfileNameSuffix
		go RunSkaffoldDev(ctx, os.Stdout, skaffoldConfigPath, []string{profileName}, opts)
	}

	go Watch(workDir, change)

	for {
		ch := <-change
		if len(ch) > 0 {
			fmt.Printf("\n♻️  %s changed! Re-rendering manifests...\n\n", ch)

			if changeHandler != nil {
				changeHandler(ch)
			}

			if err := runPreCommands(); err != nil {
				errHandler(err)
			}

			// empty the buffer as we only ever do one re-render cycle per a batch of changes
			if len(change) > 0 {
				for range change {
					if len(change) == 0 {
						break
					}
				}
			}
		}
	}
}
