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

package kubernetes

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/appvia/kev/pkg/kev/config"
	composego "github.com/compose-spec/compose-go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	v1apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("ProjectService", func() {

	var (
		project            composego.Project
		projectService     ProjectService
		projectServiceName string
		labels             composego.Labels
		deploy             *composego.DeployConfig
		ports              []composego.ServicePortConfig
		expose             composego.StringOrNumberList
		volumes            []composego.ServiceVolumeConfig
		environment        composego.MappingWithEquals
		healthcheck        composego.HealthCheckConfig
		projectVolumes     composego.Volumes
	)

	BeforeEach(func() {
		projectServiceName = "db"
		labels = composego.Labels{}
		deploy = &composego.DeployConfig{}
		ports = []composego.ServicePortConfig{}
		expose = composego.StringOrNumberList{}
		volumes = []composego.ServiceVolumeConfig{}
		environment = composego.MappingWithEquals{}
		healthcheck = composego.HealthCheckConfig{}
		projectVolumes = composego.Volumes{}
	})

	JustBeforeEach(func() {
		projectService = ProjectService{
			Name:        projectServiceName,
			Labels:      labels,
			Deploy:      deploy,
			Ports:       ports,
			Expose:      expose,
			Environment: environment,
			HealthCheck: &healthcheck,
			Volumes:     volumes,
		}

		services := composego.Services{}
		services = append(services, composego.ServiceConfig(projectService))

		project = composego.Project{
			Volumes:  projectVolumes,
			Services: services,
		}
	})

	Describe("enabled", func() {

		When("component toggle label is set to Truthy value", func() {
			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelComponentEnabled: "true",
				}
			})

			It("returns true", func() {
				Expect(projectService.enabled()).To(BeTrue())
			})
		})

		When("component toggle label is set to Falsy value", func() {
			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelComponentEnabled: "false",
				}
			})

			It("returns false", func() {
				Expect(projectService.enabled()).To(BeFalse())
			})
		})

		When("component toggle label is set to any string value not representing a boolean value (one of: 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False.)", func() {
			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelComponentEnabled: "anyrandomstring",
				}
			})

			It("returns true", func() {
				Expect(projectService.enabled()).To(BeTrue())
			})
		})

		When("component toggle label is not specified", func() {
			It("defaults to true if no label", func() {
				Expect(projectService.enabled()).To(BeTrue())
			})
		})
	})

	Describe("replicas", func() {

		replicas := 10

		Context("when provided via label", func() {

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelWorkloadReplicas: strconv.Itoa(replicas),
				}
			})

			It("will use a label value", func() {
				Expect(projectService.replicas()).To(Equal(replicas))
			})
		})

		Context("when provided via both the label and as part of the project service spec", func() {

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelWorkloadReplicas: strconv.Itoa(replicas),
				}

				deployBlockReplicas := uint64(2)
				deploy = &composego.DeployConfig{
					Replicas: &deployBlockReplicas,
				}
			})

			It("will use a label value", func() {
				Expect(projectService.replicas()).To(Equal(int(replicas)))
			})
		})

		Context("when replicas label not present but specified as part of the project service spec", func() {

			replicas := uint64(2)

			BeforeEach(func() {
				deploy = &composego.DeployConfig{
					Replicas: &replicas,
				}
			})

			It("will use a replica number as specified in deploy block", func() {
				Expect(projectService.replicas()).To(Equal(int(replicas)))
			})
		})

		Context("when there is no replicas label supplied nor deploy block contains number of replicas", func() {
			It("will use default number of replicas", func() {
				Expect(projectService.replicas()).To(Equal(config.DefaultReplicaNumber))
			})
		})
	})

	Describe("workloadType", func() {

		Context("when provided via label", func() {

			workloadType := "StatefulSet"

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelWorkloadType: workloadType,
				}
			})

			It("will use a label value", func() {
				Expect(projectService.workloadType()).To(Equal(workloadType))
			})
		})

		Context("when not specified via label", func() {
			It("will use a default workload type", func() {
				Expect(projectService.workloadType()).To(Equal(config.DefaultWorkload))
			})
		})

		Context("when deploy block `mode` defined as `global` and workload type is different than DaemonSet", func() {

			projectWorkloadType := config.StatefulsetWorkload

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelWorkloadType: projectWorkloadType,
				}

				deploy = &composego.DeployConfig{
					Mode: "global",
				}
			})

			It("warns the user about the mismatch", func() {
				projectService.workloadType()
				assertLog(logrus.WarnLevel,
					"Compose service defined as 'global' should map to K8s DaemonSet. Current configuration forces conversion to StatefulSet",
					map[string]string{
						"workload-type":   projectWorkloadType,
						"project-service": projectServiceName,
					},
				)
			})
		})
	})

	Describe("serviceType", func() {

		Context("when provided via label", func() {
			validType := config.ClusterIPService

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelServiceType: validType,
				}
			})

			It("returns service type as expected", func() {
				Expect(projectService.serviceType()).To(Equal(validType))
			})
		})

		Context("when not specified via label", func() {
			It("returns service type as expected", func() {
				Expect(projectService.serviceType()).To(Equal(config.DefaultService))
			})
		})

		Describe("validations", func() {

			Context("with an invalid service value", func() {

				invalidType := "some-invalid-type"

				BeforeEach(func() {
					labels = composego.Labels{
						config.LabelServiceType: invalidType,
					}
				})

				It("returns an error", func() {
					_, err := projectService.serviceType()
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(fmt.Sprintf("`%s` workload service type `%s` not supported", projectServiceName, invalidType)))
				})
			})

			Context("when node port is specified via label but service type was different that NodePort", func() {
				nodePort := "1234"

				BeforeEach(func() {
					labels = composego.Labels{
						config.LabelServiceType:         config.ClusterIPService,
						config.LabelServiceNodePortPort: nodePort,
					}
				})

				It("returns an error", func() {
					_, err := projectService.serviceType()
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(fmt.Sprintf("`%s` workload service type must be set as `NodePort` when assiging node port value", projectServiceName)))
				})
			})

			Context("when node port is specified via label and project service has multiple ports specified", func() {
				nodePort := "1234"

				BeforeEach(func() {
					labels = composego.Labels{
						config.LabelServiceType:         config.NodePortService,
						config.LabelServiceNodePortPort: nodePort,
					}
					ports = []composego.ServicePortConfig{
						{
							Target:    8080,
							Published: 9090,
							Protocol:  string(v1.ProtocolTCP),
						},
						{
							Target:    8081,
							Published: 9091,
							Protocol:  string(v1.ProtocolTCP),
						},
					}
				})

				It("returns an error", func() {
					_, err := projectService.serviceType()
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(fmt.Sprintf("`%s` cannot set NodePort service port when project service has multiple ports defined", projectServiceName)))
				})
			})
		})
	})

	Describe("nodePort", func() {

		Context("when specified via labels", func() {
			nodePort := 1234

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelServiceNodePortPort: strconv.Itoa(nodePort),
				}
			})

			It("will use label value", func() {
				Expect(projectService.nodePort()).To(Equal(int32(nodePort)))
			})
		})

		Context("when not specified via labels", func() {
			It("will return 0", func() {
				Expect(projectService.nodePort()).To(Equal(int32(0)))
			})
		})
	})

	Describe("exposeService", func() {

		Context("when specified via labels", func() {
			expose := "true"

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelServiceExpose: expose,
				}
			})

			It("will use label value", func() {
				Expect(projectService.exposeService()).To(Equal(expose))
			})
		})

		Context("when not specified via labels", func() {
			It("will return empty string", func() {
				Expect(projectService.exposeService()).To(Equal(""))
			})
		})

		Describe("validations", func() {

			Context("when service hasn't been exposed via labels but TLS secret was provided", func() {
				BeforeEach(func() {
					labels = composego.Labels{
						config.LabelServiceExpose:          "",
						config.LabelServiceExposeTLSSecret: "my-tls-secret-name",
					}
				})

				It("returns an error", func() {
					_, err := projectService.exposeService()
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("Service can't have TLS secret name when it hasn't been exposed"))
				})
			})

		})

	})

	Describe("tlsSecretName", func() {

		Context("when specified via labels", func() {
			tls := "my-secret"

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelServiceExposeTLSSecret: tls,
				}
			})

			It("will use label value", func() {
				Expect(projectService.tlsSecretName()).To(Equal(tls))
			})
		})

		Context("when not specified via labels", func() {
			It("will return an empty string", func() {
				Expect(projectService.tlsSecretName()).To(Equal(""))
			})
		})
	})

	Describe("getKubernetesUpdateStrategy", func() {

		Context("when deploy block defined and contains UpdateConfig details", func() {

			parallelism := uint64(2)

			Context("with update config order set as `stop-first`", func() {

				BeforeEach(func() {
					deploy = &composego.DeployConfig{
						UpdateConfig: &composego.UpdateConfig{
							Order:       "stop-first",
							Parallelism: &parallelism,
						},
					}
				})

				expectedMaxSurge := intstr.FromInt(0)
				expectedMaxUnavailable := intstr.FromInt(cast.ToInt(parallelism))

				It("returns appropriate RollingUpdateDeployment object", func() {
					Expect(projectService.getKubernetesUpdateStrategy()).To(Equal(&v1apps.RollingUpdateDeployment{
						MaxUnavailable: &expectedMaxUnavailable,
						MaxSurge:       &expectedMaxSurge,
					}))
				})

			})

			Context("with update config order set as `start-first`", func() {

				BeforeEach(func() {
					deploy = &composego.DeployConfig{
						UpdateConfig: &composego.UpdateConfig{
							Order:       "start-first",
							Parallelism: &parallelism,
						},
					}
				})

				expectedMaxUnavailable := intstr.FromInt(0)
				expectedMaxSurge := intstr.FromInt(cast.ToInt(parallelism))

				It("returns appropriate RollingUpdateDeployment object", func() {
					Expect(projectService.getKubernetesUpdateStrategy()).To(Equal(&v1apps.RollingUpdateDeployment{
						MaxUnavailable: &expectedMaxUnavailable,
						MaxSurge:       &expectedMaxSurge,
					}))
				})

			})

		})

	})

	Describe("volumes", func() {

		volumeName := "vol-a"
		targetPath := "/some/path"
		volumeLabels := composego.Labels{}

		BeforeEach(func() {
			volumes = []composego.ServiceVolumeConfig{
				{
					Source: volumeName,
					Target: targetPath,
				},
			}

			projectVolumes = composego.Volumes{
				volumeName: composego.VolumeConfig{
					Name: volumeName,
				},
			}
		})

		Context("for project service with volumes", func() {

			It("returns a slice of Volumes objects", func() {
				Expect(projectService.volumes(&project)).To(Equal([]Volumes{
					{
						SvcName:      projectServiceName,
						MountPath:    ":" + targetPath,
						VolumeName:   volumeName,
						Container:    targetPath,
						PVCName:      projectServiceName + "-claim0",
						PVCSize:      config.DefaultVolumeSize,
						StorageClass: config.DefaultVolumeStorageClass,
					},
				}))
			})

			Context("when volume contains storage class label", func() {
				storageClass := "ssd"

				BeforeEach(func() {
					volumeLabels = composego.Labels{
						config.LabelVolumeStorageClass: storageClass,
					}

					projectVolumes = composego.Volumes{
						volumeName: composego.VolumeConfig{
							Name:   volumeName,
							Labels: volumeLabels,
						},
					}
				})

				It("will set the storage class as expected", func() {
					v, _ := projectService.volumes(&project)
					Expect(v[0].StorageClass).To(Equal(storageClass))
				})
			})

			Context("when volume contains volume size label", func() {
				storageSize := "1Gi"

				BeforeEach(func() {
					volumeLabels = composego.Labels{
						config.LabelVolumeSize: storageSize,
					}

					projectVolumes = composego.Volumes{
						volumeName: composego.VolumeConfig{
							Name:   volumeName,
							Labels: volumeLabels,
						},
					}
				})

				It("will set the volume size as expected", func() {
					v, _ := projectService.volumes(&project)
					Expect(v[0].PVCSize).To(Equal(storageSize))
				})
			})

		})

	})

	Describe("placement", func() {

		Context("when placement constraints information has been provided in deploy block", func() {

			BeforeEach(func() {
				deploy = &composego.DeployConfig{
					Placement: composego.Placement{
						Constraints: []string{
							"node.role==worker",
						},
					},
				}
			})

			It("returns expected placement detail", func() {
				Expect(projectService.placement()).To(HaveKeyWithValue("node-role.kubernetes.io/worker", "true"))
			})

		})

		Context("when placement constraints are not provided in deploy block", func() {
			It("returns nil", func() {
				Expect(projectService.placement()).To(BeNil())
			})
		})
	})

	Describe("resourceRequests", func() {

		Context("when resources aren't specified via labels and there is no resource reservation specified in deploy block", func() {
			It("returns resource request values as 0", func() {
				mem, cpu := projectService.resourceRequests()
				Expect(*mem).To(BeEquivalentTo(0))
				Expect(*cpu).To(BeEquivalentTo(0))
			})
		})

		Context("when resource reservation are defined by the deploy block", func() {

			BeforeEach(func() {
				deploy = &composego.DeployConfig{
					Resources: composego.Resources{
						Reservations: &composego.Resource{
							NanoCPUs:    "0.1",
							MemoryBytes: composego.UnitBytes(int64(1000)),
						},
					},
				}
			})

			Context("and resources aren't specified via labels", func() {
				It("returns resource request as defined in deploy block", func() {
					mem, cpu := projectService.resourceRequests()
					Expect(*mem).To(BeEquivalentTo(1000))
					Expect(*cpu).To(BeEquivalentTo(100))
				})
			})

			Context("and CPU request is specified via label", func() {
				BeforeEach(func() {
					labels = composego.Labels{
						config.LabelWorkloadCPU: "0.2",
					}
				})

				It("returns CPU request as defined by the label value ", func() {
					_, cpu := projectService.resourceRequests()
					Expect(*cpu).To(BeEquivalentTo(200))
				})
			})

			Context("and Memory request is specified via label", func() {
				BeforeEach(func() {
					labels = composego.Labels{
						config.LabelWorkloadMemory: "1M",
					}
				})

				It("returns Memory request as defined by the label value ", func() {
					mem, _ := projectService.resourceRequests()
					Expect(*mem).To(BeEquivalentTo(1000000))
				})
			})
		})
	})

	Describe("resourceLimits", func() {

		Context("when resource limits aren't specified via labels and there is no resource reservation limits specified in deploy block", func() {
			It("returns resource limits values as 0", func() {
				mem, cpu := projectService.resourceLimits()
				Expect(*mem).To(BeEquivalentTo(0))
				Expect(*cpu).To(BeEquivalentTo(0))
			})
		})

		Context("when resource limits are defined by the deploy block", func() {

			BeforeEach(func() {
				deploy = &composego.DeployConfig{
					Resources: composego.Resources{
						Limits: &composego.Resource{
							NanoCPUs:    "0.1",
							MemoryBytes: composego.UnitBytes(int64(1000)),
						},
					},
				}
			})

			Context("and resource limits aren't specified via labels", func() {
				It("returns resource limit as defined in deploy block", func() {
					mem, cpu := projectService.resourceLimits()
					Expect(*mem).To(BeEquivalentTo(1000))
					Expect(*cpu).To(BeEquivalentTo(100))
				})
			})

			Context("and CPU limit is specified via label", func() {
				BeforeEach(func() {
					labels = composego.Labels{
						config.LabelWorkloadMaxCPU: "0.2",
					}
				})

				It("returns CPU limit as defined by the label value ", func() {
					_, cpu := projectService.resourceLimits()
					Expect(*cpu).To(BeEquivalentTo(200))
				})
			})

			Context("and Memory limit is specified via label", func() {
				BeforeEach(func() {
					labels = composego.Labels{
						config.LabelWorkloadMaxMemory: "200",
					}
				})

				It("returns Memory limit as defined by the label value ", func() {
					mem, _ := projectService.resourceLimits()
					Expect(*mem).To(BeEquivalentTo(200))
				})
			})
		})
	})

	Describe("runAsUser", func() {

		Context("when defined via labels", func() {
			runAsUser := "1000"

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelWorkloadSecurityContextRunAsUser: runAsUser,
				}
			})

			It("returns label value", func() {
				Expect(projectService.runAsUser()).To(Equal(runAsUser))
			})
		})

		Context("when not defined via labels", func() {
			It("returns default value", func() {
				Expect(projectService.runAsUser()).To(Equal(config.DefaultSecurityContextRunAsUser))
			})
		})
	})

	Describe("runAsGroup", func() {

		Context("when defined via labels", func() {
			runAsGroup := "1000"

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelWorkloadSecurityContextRunAsGroup: runAsGroup,
				}
			})

			It("returns label value", func() {
				Expect(projectService.runAsGroup()).To(Equal(runAsGroup))
			})
		})

		Context("when not defined via labels", func() {
			It("returns default value", func() {
				Expect(projectService.runAsGroup()).To(Equal(config.DefaultSecurityContextRunAsGroup))
			})
		})
	})

	Describe("fsGroup", func() {

		Context("when defined via labels", func() {
			fsGroup := "1000"

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelWorkloadSecurityContextFsGroup: fsGroup,
				}
			})

			It("returns label value", func() {
				Expect(projectService.fsGroup()).To(Equal(fsGroup))
			})
		})

		Context("when not defined via labels", func() {
			It("returns default value", func() {
				Expect(projectService.fsGroup()).To(Equal(config.DefaultSecurityContextFsGroup))
			})
		})
	})

	Describe("imagePullPolicy", func() {

		Context("when defined via labels", func() {
			policy := "Always"

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelWorkloadImagePullPolicy: policy,
				}
			})

			It("returns label value", func() {
				Expect(projectService.imagePullPolicy()).To(Equal(v1.PullPolicy(policy)))
			})
		})

		Context("when not defined via labels", func() {
			It("returns default value", func() {
				Expect(projectService.imagePullPolicy()).To(Equal(v1.PullPolicy(config.DefaultImagePullPolicy)))
			})
		})

		Context("for invalid image pull policy", func() {
			policy := "invalid-policy-name"

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelWorkloadImagePullPolicy: policy,
				}
			})

			It("warn the user and returns default value", func() {
				Expect(projectService.imagePullPolicy()).To(Equal(v1.PullPolicy(config.DefaultImagePullPolicy)))

				assertLog(logrus.WarnLevel,
					fmt.Sprintf("Invalid image pull policy passed in via %s label. Defaulting to `IfNotPresent`.", config.LabelWorkloadImagePullPolicy),
					map[string]string{
						"project-service":   projectServiceName,
						"image-pull-policy": policy,
					},
				)
			})
		})
	})

	Describe("imagePullSecret", func() {

		Context("when defined via labels", func() {
			secret := "image-pull-secret"

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelWorkloadImagePullSecret: secret,
				}
			})

			It("returns label value", func() {
				Expect(projectService.imagePullSecret()).To(Equal(secret))
			})
		})

		Context("when not defined via labels", func() {
			It("returns default value", func() {
				Expect(projectService.imagePullSecret()).To(Equal(config.DefaultImagePullSecret))
			})
		})
	})

	Describe("serviceAccountName", func() {

		Context("when defined via labels", func() {
			sa := "sa"

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelWorkloadServiceAccountName: sa,
				}
			})

			It("returns label value", func() {
				Expect(projectService.serviceAccountName()).To(Equal(sa))
			})
		})

		Context("when not defined via labels", func() {
			It("returns default value", func() {
				Expect(projectService.serviceAccountName()).To(Equal(config.DefaultServiceAccountName))
			})
		})
	})

	Describe("restartPolicy", func() {

		Context("when defined via labels", func() {
			policy := config.DefaultRestartPolicy

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelWorkloadRestartPolicy: policy,
				}
			})

			It("returns label value", func() {
				Expect(projectService.restartPolicy()).To(Equal(v1.RestartPolicy(policy)))
			})
		})

		Context("when not defined via labels", func() {

			Context("and defined in the project service deploy block", func() {
				policy := config.DefaultRestartPolicy

				BeforeEach(func() {
					deploy = &composego.DeployConfig{
						RestartPolicy: &composego.RestartPolicy{
							Condition: policy,
						},
					}
				})

				It("returns restart policy condition value from deploy block", func() {
					Expect(projectService.restartPolicy()).To(Equal(v1.RestartPolicy(policy)))
				})
			})

			Context("and defined on the project service directly with no deploy block", func() {
				policy := config.RestartPolicyNever

				It("returns restart policy defined on the project service level", func() {
					projectService.Restart = policy
					Expect(projectService.restartPolicy()).To(Equal(v1.RestartPolicy(policy)))
				})
			})

		})

		Context("when not defined anywhere", func() {
			It("returns default value", func() {
				Expect(projectService.restartPolicy()).To(Equal(v1.RestartPolicy(config.DefaultRestartPolicy)))
			})
		})

		Describe("validations", func() {

			Context("and restart policy was specified as `unless-stopped`", func() {
				policy := "unless-stopped"

				BeforeEach(func() {
					deploy = &composego.DeployConfig{
						RestartPolicy: &composego.RestartPolicy{
							Condition: policy,
						},
					}
				})

				It("warns the user and defaults restart policy to `Always`", func() {
					Expect(projectService.restartPolicy()).To(Equal(v1.RestartPolicy(config.RestartPolicyAlways)))

					assertLog(logrus.WarnLevel,
						"Restart policy 'unless-stopped' is not supported, converting it to 'always'",
						map[string]string{
							"project-service": projectServiceName,
							"restart-policy":  policy,
						},
					)
				})

			})

			Context("when invalid policy has been provided", func() {
				policy := "invalid-policy"

				BeforeEach(func() {
					labels = composego.Labels{
						config.LabelWorkloadRestartPolicy: policy,
					}
				})

				It("warns the user and defaults restart policy to `Always`", func() {
					Expect(projectService.restartPolicy()).To(Equal(v1.RestartPolicy(config.RestartPolicyAlways)))

					assertLog(logrus.WarnLevel,
						"Restart policy is not supported, defaulting to 'Always'",
						map[string]string{
							"project-service": projectServiceName,
							"restart-policy":  policy,
						},
					)
				})
			})
		})
	})

	Describe("environment", func() {
		key := "FOO"
		val := "BAR"

		Context("with ENV Vars with values present", func() {
			BeforeEach(func() {
				environment = composego.MappingWithEquals{
					key: &val,
				}
			})

			It("includes them in returned object", func() {
				Expect(projectService.environment()).To(HaveKeyWithValue(key, &val))
			})
		})

		Context("with ENV Var with empty value", func() {

			BeforeEach(func() {
				environment = composego.MappingWithEquals{
					key: nil,
				}
			})

			Context("when value defined in the OS environemnt", func() {
				osVal := "BAZ"

				BeforeEach(func() {
					os.Setenv(key, osVal)
				})

				AfterEach(func() {
					os.Unsetenv(key)
				})

				It("takes the value from the OS environment", func() {
					Expect(projectService.environment()).To(HaveKeyWithValue(key, &osVal))
				})
			})

			Context("when value isn't defined in the OS environment", func() {
				It("warns the user and skips that environment variable altogether", func() {
					Expect(projectService.environment()).ToNot(HaveKeyWithValue(key, &val))

					assertLog(logrus.WarnLevel,
						"Env Var has no value and will be ignored",
						map[string]string{
							"project-service": projectServiceName,
							"env-var":         key,
						},
					)
				})
			})
		})
	})

	Describe("ports", func() {

		BeforeEach(func() {
			ports = []composego.ServicePortConfig{
				{
					Target:    8080,
					Published: 9090,
					Protocol:  string(v1.ProtocolTCP),
				},
			}
		})

		It("returns defined ports by default", func() {
			Expect(projectService.ports()).To(ContainElement(composego.ServicePortConfig{
				Target:    8080,
				Published: 9090,
				Protocol:  string(v1.ProtocolTCP),
			}))
			Expect(len(projectService.ports())).To(Equal(1))
		})

		Context("when Expose ports are also provided and different than those specified in Ports", func() {
			BeforeEach(func() {
				expose = composego.StringOrNumberList{
					"9999",
				}
			})

			It("adds expose ports to the list", func() {
				Expect(projectService.ports()).To(ContainElement(composego.ServicePortConfig{
					Target:    9999,
					Published: 9999,
					Protocol:  string(v1.ProtocolTCP),
				}))
				Expect(len(projectService.ports())).To(Equal(2))
			})
		})

		Context("when Expose ports are also provided but matching those already specified in Ports", func() {
			BeforeEach(func() {
				expose = composego.StringOrNumberList{
					"8080",
				}
			})

			It("doesn't add them to the list", func() {
				Expect(len(projectService.ports())).To(Equal(1))
			})
		})
	})

	Describe("healthcheck", func() {

		Context("when valid healthcheck is defined in deploy block", func() {
			timeout := composego.Duration(time.Duration(10) * time.Second)
			interval := composego.Duration(time.Duration(10) * time.Second)
			startPeriod := composego.Duration(time.Duration(10) * time.Second)
			retries := uint64(3)

			BeforeEach(func() {
				healthcheck = composego.HealthCheckConfig{
					Test: composego.HealthCheckTest{
						"CMD-SHELL",
						"my command",
					},
					Timeout:     &timeout,
					Interval:    &interval,
					StartPeriod: &startPeriod,
					Retries:     &retries,
				}
			})

			It("returns a Probe as expected", func() {
				Expect(projectService.healthcheck()).To(Equal(&v1.Probe{
					Handler: v1.Handler{
						Exec: &v1.ExecAction{
							Command: healthcheck.Test[1:],
						},
					},
					TimeoutSeconds:      10,
					PeriodSeconds:       10,
					InitialDelaySeconds: 10,
					FailureThreshold:    3,
				}))
			})
		})

		Describe("validations", func() {

			Context("when Test command is not defined", func() {
				BeforeEach(func() {
					healthcheck = composego.HealthCheckConfig{
						Test: composego.HealthCheckTest{
							"",
						},
					}
				})

				It("logs and returns error", func() {
					_, err := projectService.healthcheck()
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("Health check misconfigured"))

					assertLog(logrus.ErrorLevel,
						"Health check misconfigured",
						map[string]string{},
					)
				})
			})

			When("any of time based paramaters is set to 0", func() {
				JustBeforeEach(func() {
					projectService.Labels = composego.Labels{
						config.LabelWorkloadLivenessProbeTimeout: "0",
					}
				})

				It("logs and returns error", func() {
					_, err := projectService.healthcheck()
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("Health check misconfigured"))

					assertLog(logrus.ErrorLevel,
						"Health check misconfigured",
						map[string]string{},
					)
				})
			})
		})
	})

	Describe("livenessProbeCommand", func() {
		When("defined via labels", func() {

			Context("and supplied as string command", func() {
				cmd := "my test command"

				BeforeEach(func() {
					labels = composego.Labels{
						config.LabelWorkloadLivenessProbeCommand: cmd,
					}
				})

				It("returns label value", func() {
					Expect(projectService.livenessProbeCommand()).To(HaveLen(1))
					Expect(projectService.livenessProbeCommand()).To(ContainElement(cmd))
				})
			})

			Context("and specified as list", func() {
				cmd := "[\"CMD\", \"echo\", \"Hello World\"]"

				BeforeEach(func() {
					labels = composego.Labels{
						config.LabelWorkloadLivenessProbeCommand: cmd,
					}
				})

				It("returns label value", func() {
					Expect(projectService.livenessProbeCommand()).To(HaveLen(2))
					Expect(projectService.livenessProbeCommand()).ToNot(ContainElements("CMD"))
					Expect(projectService.livenessProbeCommand()).To(ContainElements("echo", "Hello World"))
				})
			})
		})

		When("defined via healthcheck block only", func() {
			cmd := []string{
				"CMD-SHELL",
				"/my-test/command.sh",
				"some-args",
			}

			JustBeforeEach(func() {
				projectService.HealthCheck = &composego.HealthCheckConfig{
					Test: composego.HealthCheckTest(cmd),
				}
			})

			It("returns project service healthcheck test command", func() {
				Expect(projectService.livenessProbeCommand()).To(HaveLen(2))
				Expect(projectService.livenessProbeCommand()).To(ContainElements(cmd[1:]))
			})
		})

		When("not defined by both label nor healthcheck block", func() {
			It("returns default value", func() {
				Expect(projectService.livenessProbeCommand()).To(ContainElements(config.DefaultLivenessProbeCommand))
			})
		})
	})

	Describe("livenessProbeInterval", func() {
		When("defined via labels", func() {
			interval := "30s"

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelWorkloadLivenessProbeInterval: interval,
				}
			})

			It("returns label value", func() {
				Expect(projectService.livenessProbeInterval()).To(BeEquivalentTo(30))
			})
		})

		When("defined via healthcheck block only", func() {
			seconds := 10
			interval := composego.Duration(time.Duration(seconds) * time.Second)

			JustBeforeEach(func() {
				projectService.HealthCheck = &composego.HealthCheckConfig{
					Interval: &interval,
				}
			})

			It("returns project service healthcheck interval", func() {
				Expect(projectService.livenessProbeInterval()).To(BeEquivalentTo(seconds))
			})
		})

		When("not defined by both label nor healthcheck block", func() {
			It("returns default value", func() {
				expected, _ := durationStrToSecondsInt(config.DefaultLivenessProbeInterval)
				Expect(projectService.livenessProbeInterval()).To(Equal(*expected))
			})
		})
	})

	Describe("livenessProbeTimeout", func() {
		When("defined via labels", func() {
			timeout := "30s"

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelWorkloadLivenessProbeTimeout: timeout,
				}
			})

			It("returns label value", func() {
				Expect(projectService.livenessProbeTimeout()).To(BeEquivalentTo(30))
			})
		})

		When("defined via healthcheck block only", func() {
			seconds := 3
			timeout := composego.Duration(time.Duration(seconds) * time.Second)

			JustBeforeEach(func() {
				projectService.HealthCheck = &composego.HealthCheckConfig{
					Timeout: &timeout,
				}
			})

			It("returns project service healthcheck timeout", func() {
				Expect(projectService.livenessProbeTimeout()).To(BeEquivalentTo(seconds))
			})
		})

		When("not defined by both label nor healthcheck block", func() {
			It("returns default value", func() {
				expected, _ := durationStrToSecondsInt(config.DefaultLivenessProbeTimeout)
				Expect(projectService.livenessProbeTimeout()).To(Equal(*expected))
			})
		})
	})

	Describe("livenessProbeInitialDelay", func() {
		When("defined via labels", func() {
			delay := "30s"

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelWorkloadLivenessProbeInitialDelay: delay,
				}
			})

			It("returns label value", func() {
				Expect(projectService.livenessProbeInitialDelay()).To(BeEquivalentTo(30))
			})
		})

		When("defined via healthcheck block only", func() {
			seconds := 5
			startPeriod := composego.Duration(time.Duration(seconds) * time.Second)

			JustBeforeEach(func() {
				projectService.HealthCheck = &composego.HealthCheckConfig{
					StartPeriod: &startPeriod,
				}
			})

			It("returns project service healthcheck start period", func() {
				Expect(projectService.livenessProbeInitialDelay()).To(BeEquivalentTo(seconds))
			})
		})

		When("not defined by both label nor healthcheck block", func() {
			It("returns default value", func() {
				expected, _ := durationStrToSecondsInt(config.DefaultLivenessProbeInitialDelay)
				Expect(projectService.livenessProbeInitialDelay()).To(Equal(*expected))
			})
		})
	})

	Describe("livenessProbeRetries", func() {
		When("defined via labels", func() {
			retries := "3"

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelWorkloadLivenessProbeRetries: retries,
				}
			})

			It("returns label value", func() {
				expected := 3
				Expect(projectService.livenessProbeRetries()).To(BeEquivalentTo(expected))
			})
		})

		When("defined via healthcheck block only", func() {
			retries := uint64(5)

			JustBeforeEach(func() {
				projectService.HealthCheck = &composego.HealthCheckConfig{
					Retries: &retries,
				}
			})

			It("returns project service healthcheck retries", func() {
				Expect(projectService.livenessProbeRetries()).To(BeEquivalentTo(retries))
			})
		})

		When("not defined by both label nor healthcheck block", func() {
			It("returns default value", func() {
				Expect(projectService.livenessProbeRetries()).To(BeEquivalentTo(config.DefaultLivenessProbeRetries))
			})
		})
	})

	Describe("livenessProbeDisabled", func() {
		When("defined via labels", func() {
			disabled := "true"

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelWorkloadLivenessProbeDisabled: disabled,
				}
			})

			It("returns label value", func() {
				Expect(projectService.livenessProbeDisabled()).To(BeTrue())
			})
		})

		When("defined via healthcheck block only", func() {
			disable := true

			JustBeforeEach(func() {
				projectService.HealthCheck = &composego.HealthCheckConfig{
					Disable: disable,
				}
			})

			It("returns project service healthcheck disable value", func() {
				Expect(projectService.livenessProbeDisabled()).To(BeTrue())
			})
		})

		When("not defined via labels", func() {
			It("returns default value", func() {
				Expect(projectService.livenessProbeDisabled()).To(BeFalse())
			})
		})
	})

})
