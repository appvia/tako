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
	"os"
	"path"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestKev(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kev Suite")
}

func NewTempWorkingDir(composePath string) (tmpWd string, err error) {
	data, err := ioutil.ReadFile(path.Join("testdata", composePath))
	if err != nil {
		return "", err
	}

	base, err := ioutil.TempDir("", "cmd-test")
	if err != nil {
		return "", err
	}

	wdWithComposePath := path.Join(base, composePath)
	wd := filepath.Dir(wdWithComposePath)

	if err := os.MkdirAll(wd, os.ModePerm); err != nil {
		return "", err
	}

	copied, err := os.Create(wdWithComposePath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := copied.Close(); err != nil {
			fmt.Printf("%s, while closing copied compose source\n", err)
		}
	}()

	if _, err := copied.Write(data); err != nil {
		return "", nil
	}

	return wd, nil
}
