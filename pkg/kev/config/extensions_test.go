package config_test

import (
	"bytes"

	"github.com/appvia/kev/pkg/kev/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("Extentions", func() {

	Describe("parsing", func() {
		var (
			parsedCfg  config.K8SServiceConfig
			k8s        config.K8SServiceConfig
			extensions = make(map[string]interface{})
			err        error
		)

		BeforeEach(func() {
			k8s = config.K8SServiceConfig{}
		})

		JustBeforeEach(func() {
			var buf bytes.Buffer
			err := yaml.NewEncoder(&buf).Encode(map[string]interface{}{
				"x-k8s": k8s,
			})
			Expect(err).ToNot(HaveOccurred())

			err = yaml.NewDecoder(&buf).Decode(&extensions)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("works with defaults", func() {
			BeforeEach(func() {
				k8s.Workload.Replicas = 10
				k8s.Workload.LivenessProbe.Type = config.ProbeTypeNone.String()
			})

			It("creates the config using defaults when the mandatory properties are present", func() {
				parsedCfg, err = config.K8SServiceCfgFromMap(extensions)
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedCfg).NotTo(BeNil())

				Expect(parsedCfg.Workload.Replicas).To(Equal(10))
				Expect(parsedCfg.Service).To(BeZero())

				expectedLiveness := config.DefaultLivenessProbe()
				expectedLiveness.Type = config.ProbeTypeNone.String()

				Expect(parsedCfg.Workload.LivenessProbe).To(BeEquivalentTo(expectedLiveness))
				Expect(parsedCfg.Workload.ReadinessProbe).To(BeEquivalentTo(config.DefaultReadinessProbe()))
			})
		})

		Describe("fails on mandatory fields", func() {
			Context("missing liveness probe type", func() {
				BeforeEach(func() {
					k8s.Workload.Replicas = 10
				})

				It("returns error", func() {
					parsedCfg, err = config.K8SServiceCfgFromMap(extensions)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("Workload.LivenessProbe.Type is required"))
				})
			})

			Context("missing replicas", func() {
				BeforeEach(func() {
					k8s.Workload.LivenessProbe.Type = config.ProbeTypeNone.String()
				})

				It("returns error", func() {
					parsedCfg, err = config.K8SServiceCfgFromMap(extensions)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("Workload.Replicas is required"))
				})
			})
		})
	})

	Describe("Marshalling", func() {
		It("doesn't lose information in serialization", func() {
			expected := config.DefaultLivenessProbe()

			var buf bytes.Buffer
			err := yaml.NewEncoder(&buf).Encode(expected)
			Expect(err).ToNot(HaveOccurred())

			var actual config.LivenessProbe
			err = yaml.NewDecoder(&buf).Decode(&actual)
			Expect(err).ToNot(HaveOccurred())

			Expect(expected).To(BeEquivalentTo(actual))
		})

		It("marshals invalid probetype as empty string", func() {
			expected := config.DefaultLivenessProbe()
			expected.Type = config.ProbeType("asd").String()

			var buf bytes.Buffer
			err := yaml.NewEncoder(&buf).Encode(expected)
			Expect(err).ToNot(HaveOccurred())

			var actual config.LivenessProbe
			err = yaml.NewDecoder(&buf).Decode(&actual)
			Expect(err).ToNot(HaveOccurred())
			Expect(expected).To(BeEquivalentTo(actual))
			Expect(actual.Type).To(BeEmpty())
		})
	})

})
