package kev_test

import (
	"github.com/appvia/kube-devx/pkg/kev"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manifest", func() {
	var workingDir = "testdata/merge"

	Describe("MergeEnvIntoSources", func() {
		source, _ := kev.NewComposeProject([]string{workingDir + "/docker-compose.yaml"})

		Context("pre merge", func() {
			It("confirms there are no service labels", func() {
				sourceSvc, _ := source.GetService("db")
				Expect(sourceSvc.Labels).To(HaveLen(0))
			})

			It("confirms env var overrides", func() {
				sourceSvc, _ := source.GetService("db")
				overrideMeWithVal := "value"
				Expect(sourceSvc.Environment["OVERRIDE_ME_EMPTY"]).To(BeNil())
				Expect(sourceSvc.Environment["OVERRIDE_ME_WITH_VAL"]).To(Equal(&overrideMeWithVal))
			})

			It("confirms there are no volume labels", func() {
				sourceVol, _ := source.Volumes["db_data"]
				Expect(sourceVol.Labels).To(HaveLen(0))
			})
		})

		Context("post merge", func() {
			var (
				merged   *kev.ComposeProject
				mergeErr error
				env      *kev.Environment
			)

			manifest, err := kev.LoadManifest(workingDir)
			if err == nil {
				env, _ = manifest.GetEnvironment("dev")
				merged, mergeErr = manifest.MergeEnvIntoSources(env)
			}

			It("merged the environment labels into sources", func() {
				mergedSvc, _ := merged.GetService("db")
				envSvc, _ := env.GetService("db")
				Expect(mergedSvc.Labels).To(Equal(envSvc.Labels))
			})

			It("merged the environment env var overrides into sources", func() {
				mergedSvc, _ := merged.GetService("db")
				envSvc, _ := env.GetService("db")
				Expect(mergedSvc.Environment["OVERRIDE_ME_EMPTY"]).To(Equal(envSvc.Environment["OVERRIDE_ME_EMPTY"]))
				Expect(mergedSvc.Environment["OVERRIDE_ME_WITH_VAL"]).To(Equal(envSvc.Environment["OVERRIDE_ME_WITH_VAL"]))
			})

			It("merged the environment volume labels into sources", func() {
				mergedVol := merged.Volumes["db_data"]
				envVol, _ := env.GetVolume("db_data")
				Expect(mergedVol.Labels).To(Equal(envVol.Labels))
			})

			It("should not error", func() {
				Expect(mergeErr).NotTo(HaveOccurred())
			})
		})
	})
})
