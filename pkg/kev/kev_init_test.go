package kev_test

import (
	"io/ioutil"
	"path"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
	"github.com/appvia/kube-devx/pkg/kev"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Init", func() {
	var (
		workingDir       string
		manifest         *kev.Manifest
		skaffoldManifest *kev.SkaffoldManifest
		skaffold         bool
		mErr             error
	)

	JustBeforeEach(func() {
		manifest, skaffoldManifest, mErr = kev.Init([]string{}, []string{}, workingDir, skaffold)
	})

	Context("with no alternate compose files supplied", func() {
		Context("and without any docker-compose file in the directory", func() {
			BeforeEach(func() {
				workingDir = "./testdata"
			})

			It("should error", func() {
				Expect(mErr).To(HaveOccurred())
			})
		})

		Context("and with a compose.yml file in the directory", func() {
			BeforeEach(func() {
				workingDir = "./testdata/init-default/compose-yml"
			})

			It("should initialise the manifest using the file", func() {
				Expect(manifest.GetSourcesFiles()).To(Equal([]string{"testdata/init-default/compose-yml/compose.yml"}))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("and with a compose.yaml file in the directory", func() {
			BeforeEach(func() {
				workingDir = "./testdata/init-default/compose-yaml"
			})

			It("should initialise the manifest using the file", func() {
				Expect(manifest.GetSourcesFiles()).To(Equal([]string{"testdata/init-default/compose-yaml/compose.yaml"}))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("and with a docker-compose.yaml file in the directory", func() {
			BeforeEach(func() {
				workingDir = "./testdata/init-default/docker-compose-yaml"
			})

			It("should initialise the manifest using the file", func() {
				Expect(manifest.GetSourcesFiles()).To(Equal([]string{"testdata/init-default/docker-compose-yaml/docker-compose.yaml"}))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("and with a docker-compose.yaml file in the directory", func() {
			BeforeEach(func() {
				workingDir = "./testdata/init-default/docker-compose-yml"
			})

			It("should initialise the manifest using the file", func() {
				Expect(manifest.GetSourcesFiles()).To(Equal([]string{"testdata/init-default/docker-compose-yml/docker-compose.yml"}))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})

		Context("and with a docker-compose.yml file & optional override file in the directory", func() {
			BeforeEach(func() {
				workingDir = "./testdata/init-default/docker-compose-override"
			})

			It("should initialise the manifest using both files", func() {
				expected := []string{"" +
					"testdata/init-default/docker-compose-override/docker-compose.yaml",
					"testdata/init-default/docker-compose-override/docker-compose.override.yaml",
				}
				Expect(manifest.GetSourcesFiles()).To(Equal(expected))
			})

			It("should not error", func() {
				Expect(mErr).NotTo(HaveOccurred())
			})
		})
	})

	Context("with --skaffold flag passed", func() {
		BeforeEach(func() {
			skaffold = true
		})

		When("skaffold file already exists", func() {
			BeforeEach(func() {
				workingDir = "./testdata/init-default/skaffold"
			})

			It("doesn't force override the existing manifest", func() {
				skaffoldPath := path.Join(workingDir, kev.SkaffoldFileName)
				existingSkaffoldContent, _ := ioutil.ReadFile(skaffoldPath)
				skaffoldManifestContent, _ := yaml.Marshal(skaffoldManifest)

				Expect(skaffoldManifest).ToNot(BeNil())
				Expect(existingSkaffoldContent).ToNot(Equal(skaffoldManifestContent))

				Expect(mErr).ToNot(HaveOccurred())
			})

			It("adds path to existing skaffold in the kev manifest", func() {
				skaffoldPath := path.Join(workingDir, kev.SkaffoldFileName)
				Expect(manifest.Skaffold).To(Equal(skaffoldPath))
			})
		})

		When("skaffold file doesn't exist", func() {
			BeforeEach(func() {
				workingDir = "./testdata/init-default/compose-yaml"
			})

			It("generates skaffold manifest file", func() {
				Expect(skaffoldManifest).ToNot(BeNil())
				Expect(mErr).ToNot(HaveOccurred())
			})

			It("adds path to a newly created skaffold file in the kev manifest", func() {
				skaffoldPath := path.Join(workingDir, kev.SkaffoldFileName)
				Expect(manifest.Skaffold).To(Equal(skaffoldPath))
			})
		})
	})
})
