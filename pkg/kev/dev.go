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
	"os"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/appvia/kev/pkg/kev/log"
	kmd "github.com/appvia/komando"
	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/go-wordwrap"
	"github.com/pkg/errors"
)

// NewDevRunner creates a render runner instance
func NewDevRunner(workingDir string, opts ...Options) *DevRunner {
	runner := &DevRunner{
		Project: &Project{
			WorkingDir: workingDir,
			eventHandler: func(e RunnerEvent, r Runner) error {
				return nil
			},
		},
	}
	runner.Init(opts...)
	if runner.config.Skaffold && len(runner.config.K8sNamespace) == 0 {
		runner.config.K8sNamespace = DefaultSkaffoldNamespace
	}
	return runner
}

// Run runs the dev command business logic
func (r *DevRunner) Run() error {
	if r.LogVerbose() {
		cancelFunc, pr, pw := r.pipeLogsToUI()
		defer cancelFunc()
		defer pw.Close()
		defer pr.Close()
	}

	if err := r.eventHandler(DevLoopStarting, r); err != nil {
		return newEventError(err, DevLoopStarting)
	}

	var renderRunner *RenderRunner
	r.UI.Output("[development mode] ... watching for changes - press Ctrl+C to stop", kmd.WithStyle(kmd.LogStyle))
	r.DisplaySkaffoldOptionsIfAvailable()

	runPreCommands := func(envs []string) error {
		sg := r.UI.StepGroup()
		defer sg.Done()

		var msg string
		if len(envs) == 0 {
			msg = "Running render for all environments"
		} else {
			msg = fmt.Sprintf("Running render for environment: %s", envs[0])
		}

		step := sg.Add(msg)

		renderRunner = NewRenderRunner(
			r.WorkingDir,
			WithEventHandler(r.eventHandler),
			WithEnvs(envs),
			WithUI(kmd.NoOpUI()),
		)
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
	if err := runPreCommands(r.config.Envs); err != nil {
		return err
	}

	if r.config.Skaffold {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		catchCtrlC(cancel, r.AppName, r.UI)

		skaffoldConfigPath, skaffoldConfig, err := ActivateSkaffoldDevLoop(r.WorkingDir)
		if err != nil {
			r.UI.Output("")
			r.UI.Output(
				wordwrap.WrapString(err.Error(), kmd.RecommendedWordWrapLimit),
				kmd.WithErrorStyle(),
				kmd.WithIndentChar(kmd.ErrorIndentChar),
			)
			return err
		}

		if err := WriteTo(skaffoldConfigPath, skaffoldConfig); err != nil {
			e := errors.Wrap(err, "Couldn't write Skaffold config")
			r.UI.Output("")
			r.UI.Output(
				wordwrap.WrapString(e.Error(), kmd.RecommendedWordWrapLimit),
				kmd.WithErrorStyle(),
				kmd.WithIndentChar(kmd.ErrorIndentChar),
			)
			return e
		}

		pr, pw := io.Pipe()
		defer pw.Close()
		defer pr.Close()

		profileName := r.config.Envs[0] + EnvProfileNameSuffix
		go RunSkaffoldDev(ctx, pw, skaffoldConfigPath, []string{profileName}, r.config)
		go r.displayLogs(pr, ctx)
	}

	go r.Watch(change)

	envRe := regexp.MustCompile(`^.*\.(.*)\.ya?ml$`)

	for {
		var env []string

		ch := <-change
		if len(ch) > 0 {
			r.UI.Output(
				fmt.Sprintf("Change detected in: %s", ch),
				kmd.WithIndent(1),
				kmd.WithIndentChar("♺ "),
				kmd.WithStyle(kmd.LogStyle),
			)

			match := envRe.FindStringSubmatch(ch)
			if len(match) > 0 {
				env = []string{match[1]}
			} else {
				env = []string{}
			}

			if err := r.eventHandler(DevLoopIterated, r); err != nil {
				return newEventError(err, DevLoopIterated)
			}

			_ = runPreCommands(env)

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

// Watch continuously watches source compose files & configured environment overrides
// notifying changes to a channel
func (r *DevRunner) Watch(change chan<- string) error {
	sg := r.UI.StepGroup()
	defer sg.Done()

	manifest, err := LoadManifest(r.WorkingDir)
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
	filteredEnvs, err := manifest.GetEnvironments(r.config.Envs)
	if err != nil {
		return err
	}

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

// DisplaySkaffoldOptionsIfAvailable displays Skaffold related flags and
// displays a summary of parameters used if Skaffold is enabled
func (r *DevRunner) DisplaySkaffoldOptionsIfAvailable() {
	config := r.config
	indent := 1
	if config.Skaffold {
		r.UI.Output(
			"Dev mode activated with Skaffold dev loop enabled",
			kmd.WithIndent(indent),
			kmd.WithIndentChar(kmd.LogIndentChar),
			kmd.WithStyle(kmd.LogStyle),
		)

		r.UI.Output(
			fmt.Sprintf("Will deploy to '%s' namespace. You may override it with '--namespace' flag.", config.K8sNamespace),
			kmd.WithIndent(indent),
			kmd.WithIndentChar(kmd.LogIndentChar),
			kmd.WithStyle(kmd.LogStyle),
		)

		if len(config.Kubecontext) == 0 {
			r.UI.Output(
				"Will use current kubectl context. You may override it with '--kubecontext' flag.",
				kmd.WithIndent(indent),
				kmd.WithIndentChar(kmd.LogIndentChar),
				kmd.WithStyle(kmd.LogStyle),
			)
		} else {
			r.UI.Output(
				fmt.Sprintf("Will use '%s' kube context. You may override it with '--kubecontext' flag.", config.Kubecontext),
				kmd.WithIndent(indent),
				kmd.WithIndentChar(kmd.LogIndentChar),
				kmd.WithStyle(kmd.LogStyle),
			)
		}

		if config.Envs[0] == SandboxEnv {
			r.UI.Output(
				fmt.Sprintf("Will use profile pointing at the sandbox '%s' environment. You may override it with '--kev-env' flag.", config.Envs[0]),
				kmd.WithIndent(indent),
				kmd.WithIndentChar(kmd.LogIndentChar),
				kmd.WithStyle(kmd.LogStyle),
			)
		} else {
			r.UI.Output(
				fmt.Sprintf("Will use profile pointing at Kev '%s' environment. You may override it with '--kev-env' flag.", config.Envs[0]),
				kmd.WithIndent(indent),
				kmd.WithIndentChar(kmd.LogIndentChar),
				kmd.WithStyle(kmd.LogStyle),
			)
		}

		if config.SkaffoldTail {
			r.UI.Output(
				"Will tail logs of deployed application.",
				kmd.WithIndent(indent),
				kmd.WithIndentChar(kmd.LogIndentChar),
				kmd.WithStyle(kmd.LogStyle),
			)
		} else {
			r.UI.Output(
				"Won't tail logs of deployed application. To enable log tailing use '--tail' flag.",
				kmd.WithIndent(indent),
				kmd.WithIndentChar(kmd.LogIndentChar),
				kmd.WithStyle(kmd.LogStyle),
			)
		}

		if config.SkaffoldManualTrigger {
			r.UI.Output(
				"Will stack up all the code changes and only perform build/push/deploy when triggered manually by hitting ENTER.",
				kmd.WithIndent(indent),
				kmd.WithIndentChar(kmd.LogIndentChar),
				kmd.WithStyle(kmd.LogStyle),
			)
		} else {
			r.UI.Output(
				"Will automatically trigger build/push/deploy on each application code change. To trigger changes manually use '--manual-trigger' flag.",
				kmd.WithIndent(indent),
				kmd.WithIndentChar(kmd.LogIndentChar),
				kmd.WithStyle(kmd.LogStyle),
			)
		}
	}
}

// catchCtrlC catches ctrl+c in dev loop when running Skaffold
func catchCtrlC(cancel context.CancelFunc, appName string, ui kmd.UI) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals,
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGPIPE,
	)

	go func() {
		<-signals
		signal.Stop(signals)
		cancel()
		ui.Output("")
		ui.Output(
			"Stopping Skaffold dev loop!",
			kmd.WithIndent(1),
			kmd.WithIndentChar(kmd.LogIndentChar),
			kmd.WithStyle(kmd.LogStyle),
		)
		ui.Output(
			fmt.Sprintf("'%s' will continue to reconcile and re-render K8s manifests for your project.", appName),
			kmd.WithIndent(1),
			kmd.WithIndentChar(kmd.LogIndentChar),
			kmd.WithStyle(kmd.LogStyle),
		)
		ui.Output(
			"Press Ctrl+C to stop.",
			kmd.WithIndent(1),
			kmd.WithIndentChar(kmd.LogIndentChar),
			kmd.WithStyle(kmd.LogStyle),
		)
	}()
}

func printDevProjectWithOptionsError(appName string, ui kmd.UI) {
	ui.Output("")
	ui.Output("Project had errors during dev.\n"+
		fmt.Sprintf("'%s' experienced some errors while running dev. The output\n", appName)+
		"above should contain the failure messages. Please correct these errors and\n"+
		fmt.Sprintf("run '%s dev' again.", appName),
		kmd.WithErrorBoldStyle(),
		kmd.WithIndentChar(kmd.ErrorIndentChar),
	)
}
