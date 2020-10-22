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
	"os"
	"path"

	"github.com/appvia/kev/pkg/kev/config"
	"github.com/appvia/kev/pkg/kev/converter"
	"github.com/appvia/kev/pkg/kev/log"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

const (
	// ManifestName main application manifest
	ManifestName = "kev.yaml"
	sandboxEnv   = "dev"
)

// Init initialises a kev manifest including source compose files and environments.
// If no composeSources are provided, the working directory is introspected for valid compose files to act as sources.
// Also, an implicit sandbox environment will always be created.
func Init(composeSources, envs []string, workingDir string) (*Manifest, error) {
	m, err := NewManifest(composeSources, workingDir)
	if err != nil {
		return nil, err
	}

	if _, err := m.CalculateSourcesBaseOverride(); err != nil {
		return nil, err
	}

	return m.MintEnvironments(envs), nil
}

// Reconcile reconciles changes with docker-compose sources against deployment environments.
func Reconcile(workingDir string) (*Manifest, error) {
	m, err := LoadManifest(workingDir)
	if err != nil {
		return nil, err
	}
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

	if err := m.DetectSecretsInSources(config.SecretMatchers); err != nil {
		return err
	}
	if err := m.DetectSecretsInEnvs(config.SecretMatchers); err != nil {
		return err
	}
	return nil
}

// Render renders k8s manifests for a kev app. It returns an app definition with rendered manifest info
func Render(workingDir string, format string, singleFile bool, dir string, envs []string) error {
	manifest, err := LoadManifest(workingDir)
	if err != nil {
		return errors.Wrap(err, "Unable to load app manifest")
	}

	_, err = manifest.RenderWithConvertor(converter.Factory(format), dir, singleFile, envs)
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

// ActivateSkaffoldDevLoop checks whether skaffold dev loop can be activated, and returns an error if not.
// It'll also attempt to reconcile Skaffold profiles before starting dev loop - this is done
// so that necessary profiles are added to the Skaffold config. It's necessary as environment
// specific profile is supplied to Skaffold so it knows what manifests to deploy and to which k8s cluster.
func ActivateSkaffoldDevLoop(workDir string) (string, *SkaffoldManifest, error) {
	manifest, err := LoadManifest(workDir)
	if err != nil {
		return "", nil, errors.Wrap(err, "Unable to load app manifest")
	}

	msg := `
	If you don't currently have skaffold.yaml in your project you may bootstrap a new one with "skaffold init" command.
	Once you have skaffold.yaml in your project, make sure that Kev references it by adding "skaffold: skaffold.yaml" in kev.yaml!`

	if len(manifest.Skaffold) == 0 {
		return "", nil, errors.New("Can't activate Skaffold dev loop. Kev wasn't initialized with --skaffold." + msg)
	}

	configPath := path.Join(workDir, manifest.Skaffold)

	if !fileExists(configPath) {
		return "", nil, errors.New("Can't find Skaffold config file referenced by Kev manifest. Have you initialized Kev with --skaffold?" + msg)
	}

	// Reconcile skaffold config and add potentially missing profiles before starting dev loop
	reconciledSkaffoldConfig, err := AddProfiles(configPath, manifest.GetEnvironmentsNames(), true)
	if err != nil {
		return "", nil, errors.Wrap(err, "Couldn't reconcile Skaffold config - required profiles haven't been added.")
	}

	return configPath, reconciledSkaffoldConfig, nil
}
