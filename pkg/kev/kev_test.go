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

package kev_test

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/appvia/kube-devx/pkg/kev"
	"github.com/appvia/kube-devx/pkg/kev/app"
	"github.com/google/go-cmp/cmp"
)

func TestInit(t *testing.T) {
	tests := map[string]struct {
		composeFiles []string
		overrides    []string
		scenario     string
	}{
		"simple": {
			[]string{
				"testdata/in-cluster-service/docker-compose.yml",
				"testdata/in-cluster-service/docker-compose.override.yml",
			},
			[]string{},
			"testdata/in-cluster-service",
		},
		"with local override": {
			[]string{
				"testdata/in-cluster-service/docker-compose.yml",
				"testdata/in-cluster-service/docker-compose.override.yml",
			},
			[]string{"local"},
			"testdata/in-cluster-service",
		},
		"with external secrets and configs": {
			[]string{
				"testdata/externals/docker-compose.yml",
				"testdata/externals/docker-compose.override.yml",
			},
			[]string{},
			"testdata/externals",
		},
		"with env file": {
			[]string{
				"testdata/env-file/docker-compose.yml",
				"testdata/env-file/docker-compose.override.yml",
			},
			[]string{},
			"testdata/env-file",
		},
		"with deploy attribute": {
			[]string{
				"testdata/deploy/docker-compose.yml",
				"testdata/deploy/docker-compose.override.yml",
			},
			[]string{},
			"testdata/deploy",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			def, err := kev.Init(tc.composeFiles, tc.overrides)
			if err != nil {
				t.Fatalf("Unexpected error: [%s]\n", err)
			}
			assertEqualDef(t, def, tc.scenario, tc.overrides, []string{"init"})
		})
	}
}

func TestBuild(t *testing.T) {
	tests := map[string]struct {
		composeFiles []string
		overrides    []string
		scenario     string
	}{
		"simple": {
			[]string{
				"testdata/in-cluster-service/docker-compose.yml",
				"testdata/in-cluster-service/docker-compose.override.yml",
			},
			[]string{},
			"testdata/in-cluster-service",
		},
		"with local override": {
			[]string{
				"testdata/in-cluster-service/docker-compose.yml",
				"testdata/in-cluster-service/docker-compose.override.yml",
			},
			[]string{"local"},
			"testdata/in-cluster-service",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			def, err := kev.Init(tc.composeFiles, tc.overrides)
			if err != nil {
				t.Fatalf("Unexpected error: [%s]\n", err)
			}
			if err := def.DoBuild(); err != nil {
				t.Fatalf("Unexpected error: [%s]\n", err)
			}
			assertEqualDef(t, def, tc.scenario, tc.overrides, []string{"init", "build"})
		})
	}
}

func assertEqualDef(
	tb testing.TB,
	actual *app.Definition,
	scenario string,
	overrides, ops []string,
	v ...interface{},
) {
	expected, err := loadDefinition(scenario, overrides, ops)
	if err != nil {
		tb.Fatalf("Unexpected error: [%s]", err)
	}

	diff := cmp.Diff(actual, expected)
	if diff != "" {
		msg := fmt.Sprintf("actual definition does not match expected\n%s", diff)
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+" \033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

func loadDefinition(scenario string, overrides, ops []string) (*app.Definition, error) {
	joined := strings.ToLower(strings.Join(ops, " "))
	doInit := strings.Contains(joined, "init")
	doBuild := strings.Contains(joined, "build")

	def := &app.Definition{}
	loadedOverrides := map[string]app.FileConfig{}
	loadedBuild := app.BuildConfig{}

	var loadedBuildOverrides map[string]app.ConfigTuple

	if doInit {
		var tuple app.ConfigTuple
		compose := path.Join(scenario, "init", "compose.yml")
		composeFile := ".kev/.workspace/compose.yaml"
		config := path.Join(scenario, "init", "config.yml")
		configFile := ".kev/config.yaml"

		if err := loadConfigTuple(compose, composeFile, config, configFile, &tuple); err != nil {
			return nil, err
		}

		def.Base = tuple
	}

	if doInit && len(overrides) > 0 {
		for _, o := range overrides {
			entries, err := ioutil.ReadDir(path.Join(scenario, "init", "overrides", o))
			if err != nil {
				return nil, err
			}

			for _, f := range entries {
				content, err := ioutil.ReadFile(path.Join(scenario, "init", "overrides", o, f.Name()))
				if err != nil {
					return nil, err
				}

				loadedOverrides[o] = app.FileConfig{
					Content: content,
					File:    path.Join(".kev", o, "config.yaml"),
				}
			}
		}
	}

	def.Overrides = loadedOverrides

	if doBuild {
		var tuple app.ConfigTuple
		compose := path.Join(scenario, "build", "compose.yml")
		composeFile := ".kev/.workspace/build/compose.build.yaml"
		config := path.Join(scenario, "build", "config.yml")
		configFile := ".kev/.workspace/build/config.build.yaml"

		if err := loadConfigTuple(compose, composeFile, config, configFile, &tuple); err != nil {
			return nil, err
		}

		loadedBuild = app.BuildConfig{
			Base:      tuple,
			Overrides: map[string]app.ConfigTuple{},
		}
	}

	def.Build = loadedBuild

	if doBuild && len(overrides) > 0 {
		loadedBuildOverrides = def.Build.Overrides
		for _, o := range overrides {
			var tuple app.ConfigTuple
			compose := path.Join(scenario, "build", "overrides", o, "compose.build.yml")
			composeFile := path.Join(".kev", ".workspace", "build", o, "compose.build.yaml")
			config := path.Join(scenario, "build", "overrides", o, "config.build.yml")
			configFile := path.Join(".kev", ".workspace", "build", o, "config.build.yaml")

			if err := loadConfigTuple(compose, composeFile, config, configFile, &tuple); err != nil {
				return nil, err
			}
			loadedBuildOverrides[o] = tuple
		}
	}

	return def, nil
}

func loadConfigTuple(compose, composeFile, config, configFile string, target *app.ConfigTuple) error {
	composeContent, err := ioutil.ReadFile(compose)
	if err != nil {
		return err
	}

	configContent, err := ioutil.ReadFile(config)
	if err != nil {
		return err
	}

	target.Compose = app.FileConfig{
		File:    composeFile,
		Content: composeContent,
	}
	target.Config = app.FileConfig{
		File:    configFile,
		Content: configContent,
	}

	return nil
}
