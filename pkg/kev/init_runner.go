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
	"time"

	"github.com/appvia/kev/pkg/kev/terminal"
)

type runCallbackFn func(interface{})
type runSubCmd func() error

type initRun struct {
	workingDir string
	ui         terminal.UI
	opts       InitOptions
	subCmds    []runSubCmd
	manifest   *Manifest
	results    WritableResults
}

func (r *initRun) withVerifyProject() runSubCmd {
	return func() error {
		r.ui.Header("Verifying project...")

		sg := r.ui.StepGroup()
		s := sg.Add("Ensuring this project has not already been initialised")

		if err := EnsureFirstInit(r.workingDir); err != nil {
			initStepError(r.ui, s, initStepConfig, err)
			return err
		}

		s.Success(time.Second * 5)
		return nil
	}
}

func (r *initRun) withVerifyProvidedComposeSources(fn runCallbackFn) runSubCmd {
	return func() error {
		if len(r.opts.ComposeSources) == 0 {
			return nil
		}

		r.ui.Header("Detecting compose sources...")
		sg := r.ui.StepGroup()

		for _, source := range r.opts.ComposeSources {
			s := sg.Add(fmt.Sprintf("Scanning for: %s", source))
			if fileExists(source) {
				s.Success(time.Second*5, "Using: %s", source)
			} else {
				err := fmt.Errorf("cannot find compose source %q", source)
				initStepError(r.ui, s, initStepComposeSource, err)
				return err
			}
		}
		fn(r.opts.ComposeSources)
		return nil
	}
}

func (r *initRun) withDetectDefaultComposeFiles(fn runCallbackFn) runSubCmd {
	return func() error {
		if len(r.opts.ComposeSources) > 0 {
			return nil
		}

		r.ui.Header("Detecting compose sources...")

		sg := r.ui.StepGroup()
		s := sg.Add(fmt.Sprintf("Scanning for compose configuration"))

		altComposeSources, err := findDefaultComposeFiles(r.workingDir)
		if err != nil {
			initStepError(r.ui, s, initStepComposeSource, err)
			return err
		}
		s.Success(time.Second * 5)
		for _, source := range altComposeSources {
			s := sg.Add(fmt.Sprintf("Using: %s", source))
			s.Success(time.Second * 5)
		}

		// composeSources = altComposeSources
		fn(altComposeSources)
		return nil
	}
}

func (r *initRun) withInitBase(composeSources []string) runSubCmd {
	return func() error {
		m, err := InitBase(r.workingDir, composeSources, r.opts.Envs)
		if err != nil {
			return err
		}

		r.manifest = m
		r.results = append(r.results, WritableResult{
			WriterTo: m,
			FilePath: path.Join(r.workingDir, ManifestFilename),
		})
		r.results = append(r.results, m.Environments.toWritableResults()...)
		return nil
	}
}

func (r *initRun) withSkaffold() runSubCmd {
	return func() error {
		if !r.opts.Skaffold {
			return nil
		}

		r.manifest.Skaffold = SkaffoldFileName
		project, err := r.manifest.SourcesToComposeProject()
		if err != nil {
			return err
		}

		skaffoldResults, err := CreateOrUpdateSkaffoldManifest(path.Join(r.workingDir, SkaffoldFileName), r.manifest.GetEnvironmentsNames(), project)
		if err != nil {
			return err
		}

		r.results = append(r.results, skaffoldResults...)
		return nil
	}
}

func (r *initRun) execute() (WritableResults, error) {
	for _, subCmd := range r.subCmds {
		if err := subCmd(); err != nil {
			r.ui.Output("Project had errors during initialisation.\n"+
				fmt.Sprintf("'%s' experienced some errors during project initialisation. The output\n", GetManifestName())+
				"above should contain the failure messages. Please correct these errors and\n"+
				fmt.Sprintf("run '%s init' again.", GetManifestName()),
				terminal.WithErrorBoldStyle(),
				terminal.WithIndentChar(terminal.ErrorIndentChar),
			)
			return nil, err
		}
	}

	return r.results, nil
}
