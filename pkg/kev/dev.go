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
	"os"

	"github.com/appvia/kev/pkg/kev/log"
	kmd "github.com/appvia/komando"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

// NewDevRunner creates a render runner instance
func NewDevRunner(workingDir string, handler ChangeHandler, opts ...Options) *DevRunner {
	runner := &DevRunner{chgHandler: handler, Project: &Project{workingDir: workingDir}}
	runner.Init(opts...)
	return runner
}

func (r *DevRunner) Run() error {
	var renderRunner *RenderRunner
	r.UI.Output("[development mode] ... watching for changes - press Ctrl+C to stop", kmd.WithStyle(kmd.LogStyle))

	runPreCommands := func() error {
		sg := r.UI.StepGroup()
		defer sg.Done()

		step := sg.Add(fmt.Sprintf("Running render for environment: %s", r.config.envs[0]))

		renderRunner = NewRenderRunner(r.workingDir, WithEnvs(r.config.envs), WithUI(kmd.NoOpUI()))
		if _, err := renderRunner.Run(); err != nil {
			renderStepError(r.UI, step, renderStepRenderGeneral, err)
			return err
		}

		step.Success()
		return nil
	}

	change := make(chan string, 50)
	defer close(change)

	// initial manifests generation for specified environments only
	if err := runPreCommands(); err != nil {
		return err
	}

	if r.config.skaffold {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		catchCtrlC(cancel)

		skaffoldConfigPath, skaffoldConfig, err := ActivateSkaffoldDevLoop(r.workingDir)
		if err != nil {
			return err
		}

		if err := WriteTo(skaffoldConfigPath, skaffoldConfig); err != nil {
			return errors.Wrap(err, "Couldn't write Skaffold config")
		}

		profileName := r.config.envs[0] + EnvProfileNameSuffix
		go RunSkaffoldDev(ctx, os.Stdout, skaffoldConfigPath, []string{profileName}, r.config)
	}

	go r.Watch(change)

	for {
		ch := <-change
		if len(ch) > 0 {
			r.UI.Output(
				fmt.Sprintf("Change detected in: %s", ch),
				kmd.WithIndent(1),
				kmd.WithIndentChar("♺"),
				kmd.WithStyle(kmd.LogStyle),
			)

			if r.chgHandler != nil {
				r.chgHandler(ch)
			}

			_ = runPreCommands()

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

	return nil
}

// Watch continuously watches source compose files & configured environment overrides
// notifying changes to a channel
func (r *DevRunner) Watch(change chan<- string) error {
	sg := r.UI.StepGroup()
	defer sg.Done()

	manifest, err := LoadManifest(r.workingDir)
	if err != nil {
		log.Errorf("Unable to load app manifest - %s", err)
		renderStepError(r.UI, sg.Add(""), renderStepLoad, err)
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
	filteredEnvs, err := manifest.GetEnvironments(r.config.envs)
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

func printDevProjectWithOptionsError(ui kmd.UI) {
	ui.Output("")
	ui.Output("Project had errors during dev.\n"+
		fmt.Sprintf("'%s' experienced some errors while running dev. The output\n", GetManifestName())+
		"above should contain the failure messages. Please correct these errors and\n"+
		fmt.Sprintf("run '%s dev' again.", GetManifestName()),
		kmd.WithErrorBoldStyle(),
		kmd.WithIndentChar(kmd.ErrorIndentChar),
	)
}
