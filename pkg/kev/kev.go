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
	composego "github.com/compose-spec/compose-go/types"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

const (
	// ManifestName main application manifest
	ManifestName = "kev.yaml"
	defaultEnv   = "dev"
)

// Init initialises a kev manifest including source compose files and environments.
// A default environment will be allocated if no environments were provided.
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
	m.DetectSecretsInSources(config.SecretMatchers)
	m.DetectSecretsInEnvs(config.SecretMatchers)
	return nil
}

// Render renders k8s manifests for a kev app. It returns an app definition with rendered manifest info
func Render(format string, singleFile bool, dir string, envs []string) error {
	// @todo filter specified envs, or all if none provided
	workDir, err := os.Getwd()
	if err != nil {
		log.Error("Couldn't get working directory")
		return err
	}

	manifest, err := LoadManifest(workDir)
	if err != nil {
		log.Error("Unable to load app manifest")
		return err
	}

	if _, err := manifest.CalculateSourcesBaseOverride(); err != nil {
		return errors.Wrap(err, "Unable to render")
	}

	filteredEnvs, err := manifest.GetEnvironments(envs)
	if err != nil {
		return errors.Wrap(err, "Unable to render")
	}

	rendered := map[string][]byte{}
	projects := map[string]*composego.Project{}
	files := map[string][]string{}
	sourcesFiles := manifest.GetSourcesFiles()

	for _, env := range filteredEnvs {
		p, err := manifest.MergeEnvIntoSources(env)
		if err != nil {
			return errors.Wrap(err, "Couldn't calculate compose project representation")
		}
		projects[env.Name] = p.Project
		files[env.Name] = append(sourcesFiles, env.File)
	}

	c := converter.Factory(format)
	outputPaths, err := c.Render(singleFile, dir, manifest.getWorkingDir(), projects, files, rendered)
	if err != nil {
		log.Errorf("Couldn't render manifests")
		return err
	}

	if len(manifest.Skaffold) > 0 {
		if err := UpdateSkaffoldProfiles(manifest.Skaffold, outputPaths); err != nil {
			log.Errorf("Couldn't update Skaffold profiles: %s", err)
			return err
		}
	}

	return nil
}

// Watch continuously watches source compose files & environment overrides and notifies changes to a channel
func Watch(workDir string, envs []string, change chan<- string) error {
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
	filteredEnvs, err := manifest.GetEnvironments(envs)
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

// ActivateSkaffoldDevLoop returns true when skaffold dev can be activated, false otherwise.
// It'll also attempt to reconcile Skaffold profiles before starting dev loop - this is done
// so that necessary profiles are added to the skaffold config. This is necessary as environment
// specific profile is supplied to skaffold so it knows what manifests to deploy and to which cluster.
func ActivateSkaffoldDevLoop(workDir string) (string, *SkaffoldManifest, bool) {
	manifest, err := LoadManifest(workDir)
	if err != nil {
		log.Errorf("Unable to load app manifest - %s", err)
		return "", nil, false
	}

	if len(manifest.Skaffold) == 0 {
		// kev wasn't initiated with --skaffold
		log.Warn(`Can't activate Skaffold dev loop. Kev wasn't initialized with --skaffold.
	If you don't currently have skaffold.yaml in your project you may bootstrap a new one with "skaffold init" command.
	Once you have skaffold.yaml in your project, make sure that Kev references it by adding "skaffold: skaffold.yaml" in kev.yaml!`)
		return "", nil, false
	}

	configPath := path.Join(workDir, manifest.Skaffold)

	if !fileExists(configPath) {
		log.Error(`Can't file Skaffold config file referenced by Kev manifest. Have you initialized Kev with --skaffold?
	If you don't currently have skaffold.yaml in your project you may bootstrap a new one with "skaffold init" command.
	Once you have skaffold.yaml in your project, make sure that Kev references it by adding "skaffold: skaffold.yaml" in kev.yaml!`)
		return "", nil, false
	}

	// Reconcile skaffold config and add potentially missing profiles before starting dev loop
	reconciledSkaffoldConfig, err := AddProfiles(configPath, manifest.GetEnvironmentsNames(), true)
	if err != nil {
		log.Warnf("Couldn't reconcile Skaffold config: %s. Required profiles haven't been added.", err)
		return "", nil, false
	}

	return configPath, reconciledSkaffoldConfig, true
}
