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
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/appvia/kev/pkg/kev"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestInitProvidesEnvironmentConfig(t *testing.T) {
	files := []string{"testdata/in-cluster-wordpress/docker-compose.yaml"}
	manifest, err := kev.Init(files, []string{}, "")
	if err != nil {
		t.Fatalf("Unexpected error:\n%s", err)
	}

	var actual bytes.Buffer
	env, err := manifest.GetEnvironment("dev")
	if err != nil {
		t.Fatalf("Unexpected error:\n%s", err)
	}
	if _, err := env.WriteTo(&actual); err != nil {
		t.Fatalf("Unexpected error:\n%s", err)
	}

	expected, err := ioutil.ReadFile("testdata/in-cluster-wordpress/docker-compose.kev.dev.yaml")
	if err != nil {
		t.Fatalf("Unexpected error:\n%s", err)
	}

	diff := cmp.Diff(expected, actual.Bytes())
	if diff != "" {
		t.Fatalf("actual does not match expected:\n%s", diff)
	}
}

func TestCanLoadAManifest(t *testing.T) {
	expected := &kev.Manifest{
		Id: "random-uuid",
		Sources: &kev.Sources{
			Files: []string{
				"testdata/in-cluster-wordpress/docker-compose.yaml",
			},
		},
		Environments: kev.Environments{
			&kev.Environment{
				Name: "dev",
				File: "testdata/in-cluster-wordpress/docker-compose.kev.dev.yaml",
			},
		},
	}
	workingDir := "testdata/in-cluster-wordpress"
	actual, err := kev.LoadManifest(workingDir)
	if err != nil {
		t.Fatalf("Unexpected error:\n%s", err)
	}

	expected.Id = actual.Id
	diff := cmp.Diff(expected, actual, cmpopts.IgnoreUnexported(kev.Sources{}, kev.Environment{}))
	if diff != "" {
		t.Fatalf("actual does not match expected:\n%s", diff)
	}
}
