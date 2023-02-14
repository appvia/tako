/**
 * Copyright 2023 Appvia Ltd <info@appvia.io>
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

package tako_test

import (
	"os"
	"path/filepath"

	"github.com/appvia/tako/pkg/tako"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PatchRunner", func() {
	const (
		outputDir string = "./testdata/patch-default/patched"
	)

	var (
		runner     *tako.PatchRunner
		workingDir string
		rErr       error
		dir        string
		images     []string
	)

	JustBeforeEach(func() {
		runner = tako.NewPatchRunner(workingDir,
			tako.WithPatchManifestsDir(dir),
			tako.WithPatchImages(images),
			tako.WithPatchOutputDir(outputDir))
		rErr = runner.Run()
	})

	JustAfterEach(func() {
		_ = os.RemoveAll(outputDir)
	})

	Context("Run", func() {

		Context("With k8s manifests directory specified without nested subdirectories", func() {
			BeforeEach(func() {
				dir = "./testdata/patch-default/dev"
				images = []string{
					"db=mysql:mycustomtag",
					"wordpress=wordpress:mycustomtag",
				}
			})

			It("only patches matching k8s manifests in the source directory", func() {
				Expect(rErr).ToNot(HaveOccurred())

				wp, _ := os.ReadFile(filepath.Join(outputDir, "wordpress-deployment.yaml"))
				db, _ := os.ReadFile(filepath.Join(outputDir, "db-statefulset.yaml"))

				Expect(string(wp)).To(ContainSubstring("image: wordpress:mycustomtag"))
				Expect(string(db)).To(ContainSubstring("image: mysql:mycustomtag"))
			})
		})

		Context("With k8s manifests directory specified that contains subdirectories containing manifests e.g. directory per each individual environment", func() {
			BeforeEach(func() {
				dir = "./testdata/patch-default"
				images = []string{
					"db=img1:tag1",
					"wordpress=img2:tag2",
				}
			})

			It("patches matching k8s manifests starting from the source directory root and travesing all subdirectories", func() {

				Expect(rErr).ToNot(HaveOccurred())

				devwp, _ := os.ReadFile(filepath.Join(outputDir, "dev", "wordpress-deployment.yaml"))
				devdb, _ := os.ReadFile(filepath.Join(outputDir, "dev", "db-statefulset.yaml"))

				Expect(string(devwp)).To(ContainSubstring("image: img2:tag2"))
				Expect(string(devdb)).To(ContainSubstring("image: img1:tag1"))

				prodwp, _ := os.ReadFile(filepath.Join(outputDir, "prod", "wordpress-deployment.yaml"))
				proddb, _ := os.ReadFile(filepath.Join(outputDir, "prod", "db-statefulset.yaml"))

				Expect(string(prodwp)).To(ContainSubstring("image: img2:tag2"))
				Expect(string(proddb)).To(ContainSubstring("image: img1:tag1"))
			})
		})

		When("None of the k8s manifests matched specified service name", func() {
			BeforeEach(func() {
				dir = "./testdata/patch-default/dev"
				images = []string{
					"nonexitingservice=img:tag",
				}
			})

			It("should not patch any file", func() {
				Expect(rErr).ToNot(HaveOccurred())

				// get all files in the output directory
				files, _ := os.ReadDir(outputDir)

				// assert that there are no files in the output directory
				Expect(len(files)).To(Equal(0))
			})
		})

	})

})
