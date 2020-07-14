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
	"testing"

	"github.com/appvia/kube-devx/pkg/kev"
	"github.com/appvia/kube-devx/pkg/kev/app"
	"github.com/google/go-cmp/cmp"
)

func TestInitApp(t *testing.T) {
	tests := map[string]struct {
		composeFiles []string
		overrides    []string
		scenario     string
	}{
		"simple init": {
			[]string{
				"testdata/in-cluster-service/docker-compose.yml",
				"testdata/in-cluster-service/docker-compose.override.yml",
			},
			[]string{},
			"testdata/in-cluster-service/init",
		},
		"init with local override": {
			[]string{
				"testdata/in-cluster-service/docker-compose.yml",
				"testdata/in-cluster-service/docker-compose.override.yml",
			},
			[]string{"local"},
			"testdata/in-cluster-service/init",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			def, err := kev.Init(tc.composeFiles, tc.overrides)
			if err != nil {
				t.Fatalf("Unexpected error: [%s]\n", err)
			}
			assertEqualDef(t, def, tc.scenario, tc.overrides)
		})
	}
}

func assertEqualDef(tb testing.TB, actual *app.Definition, scenario string, overrides []string, v ...interface{}) {
	expected, err := loadDefinition(scenario, overrides)
	if err != nil {
		tb.Fatalf("Unexpected error: [%s]", err)
	}

	diff := cmp.Diff(actual, expected)
	if diff != "" {
		msg := fmt.Sprintf("actual definition does not match expected: %s", diff)
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+" \033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

func loadDefinition(scenario string, overrides []string) (*app.Definition, error) {
	composeContent, err := ioutil.ReadFile(path.Join(scenario, "compose.yml"))
	if err != nil {
		return nil, err
	}

	configContent, err := ioutil.ReadFile(path.Join(scenario, "config.yml"))
	if err != nil {
		return nil, err
	}

	loadedOverrides := map[string]app.FileConfig{}

	if len(overrides) > 0 {
		for _, o := range overrides {
			entries, err := ioutil.ReadDir(path.Join(scenario, "overrides", o))
			if err != nil {
				return nil, err
			}

			for _, f := range entries {
				content, err := ioutil.ReadFile(path.Join(scenario, "overrides", o, f.Name()))
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

	return &app.Definition{
		Base: app.ConfigTuple{
			Compose: app.FileConfig{
				File:    ".kev/.workspace/compose.yaml",
				Content: composeContent,
			},
			Config: app.FileConfig{
				File:    ".kev/config.yaml",
				Content: configContent,
			},
		},
		Overrides: loadedOverrides,
		Build:     app.BuildConfig{},
	}, nil
}
