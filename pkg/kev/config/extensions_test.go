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
			parsedCfg  config.K8SConfiguration
			k8s        config.K8SConfiguration
			extensions = make(map[string]interface{})
			err        error
		)

		BeforeEach(func() {
			k8s = config.K8SConfiguration{}
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
				parsedCfg, err = config.K8SCfgFromMap(extensions)
				Expect(err).ToNot(HaveOccurred())
				Expect(parsedCfg).NotTo(BeNil())

				expectedLiveness := config.DefaultLivenessProbe()
				expectedLiveness.Type = config.ProbeTypeNone.String()

				Expect(parsedCfg.Workload.Replicas).To(Equal(10))
				Expect(parsedCfg.Workload.LivenessProbe).To(BeEquivalentTo(expectedLiveness))
				Expect(parsedCfg.Workload.ReadinessProbe).To(BeEquivalentTo(config.DefaultReadinessProbe()))
			})
		})

		When("there is no k8s extension present", func() {
			Context("Without RequirePresent configuration", func() {
				It("does not fail validations", func() {
					parsedCfg, err = config.K8SCfgFromMap(map[string]interface{}{})
					Expect(err).ToNot(HaveOccurred())
					Expect(parsedCfg).NotTo(BeNil())

					Expect(parsedCfg.Workload.Replicas).To(Equal(config.DefaultReplicaNumber))
					Expect(parsedCfg.Workload.LivenessProbe).To(BeEquivalentTo(config.DefaultLivenessProbe()))
					Expect(parsedCfg.Workload.ReadinessProbe).To(BeEquivalentTo(config.DefaultReadinessProbe()))
				})
			})
			Context("with RequireExtensions option", func() {
				When("RequireExtensions option is specified", func() {
					It("fails if map is empty", func() {
						parsedCfg, err = config.K8SCfgFromMap(map[string]interface{}{}, config.RequireExtensions())
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("Workload.Replicas is required"))
					})
				})
			})
		})

		Describe("validations", func() {
			Context("with missing workload", func() {
				BeforeEach(func() {
					extensions = map[string]interface{}{
						"x-k8s": map[string]interface{}{
							"bananas": 1,
						},
					}
				})

				It("returns an error", func() {
					parsedCfg, err = config.K8SCfgFromMap(extensions)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Workload.Replicas is required"))
				})
			})

			Context("invalid/empty workload", func() {
				BeforeEach(func() {
					extensions = map[string]interface{}{
						"x-k8s": map[string]interface{}{
							"workload": map[string]interface{}{
								"bananas": 1,
							},
						},
					}
				})

				When("workload is invalid", func() {
					It("returns an error", func() {
						parsedCfg, err = config.K8SCfgFromMap(extensions)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring("Workload.Replicas is required"))
					})
				})
			})

			Context("missing liveness probe type", func() {
				BeforeEach(func() {
					k8s.Workload.Replicas = 10
				})

				It("returns error", func() {
					parsedCfg, err = config.K8SCfgFromMap(extensions)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("Workload.LivenessProbe.Type is required"))
				})
			})

			Context("missing replicas", func() {
				BeforeEach(func() {
					k8s.Workload.LivenessProbe.Type = config.ProbeTypeNone.String()
				})

				It("returns error", func() {
					parsedCfg, err = config.K8SCfgFromMap(extensions)
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

	Describe("Merge", func() {
		It("merges target into base", func() {
			k8sBase := config.DefaultK8SConfig()
			var k8sTarget config.K8SConfiguration
			k8sTarget.Workload.Replicas = 10
			k8sTarget.Workload.LivenessProbe.Type = config.ProbeTypeNone.String()

			expected := k8sBase
			expected.Workload.Replicas = 10
			expected.Workload.LivenessProbe.Type = config.ProbeTypeNone.String()

			result, err := k8sBase.Merge(k8sTarget)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeEquivalentTo(expected))
		})

		Context("Fallback", func() {
			var extensions map[string]interface{}
			var parsedConf config.K8SConfiguration
			var err error

			JustBeforeEach(func() {
				parsedConf, err = config.K8SCfgFromMap(extensions)
				Expect(err).NotTo(HaveOccurred())
			})

			Context("configs are empty", func() {
				BeforeEach(func() {
					extensions = make(map[string]interface{})
				})

				It("returns default when map is empty", func() {
					result, err := config.DefaultK8SConfig().Merge(parsedConf)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(BeEquivalentTo(config.DefaultK8SConfig()))
				})
			})

			Context("configs are invalid", func() {
				BeforeEach(func() {
					extensions = nil
				})

				It("returns default when map is nil", func() {
					result, err := config.DefaultK8SConfig().Merge(parsedConf)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(BeEquivalentTo(config.DefaultK8SConfig()))
				})
			})
		})
	})

})
