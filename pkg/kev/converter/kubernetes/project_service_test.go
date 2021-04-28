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
	"github.com/google/go-cmp/cmp"
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
		extensions         map[string]interface{}
		deploy             *composego.DeployConfig
		ports              []composego.ServicePortConfig
		expose             composego.StringOrNumberList
		volumes            []composego.ServiceVolumeConfig
		environment        composego.MappingWithEquals
		healthcheck        composego.HealthCheckConfig
		projectVolumes     composego.Volumes
		k8sconf            config.K8SConfiguration
	)

	BeforeEach(func() {
		projectServiceName = "db"
		labels = composego.Labels{}
		extensions = make(map[string]interface{})
		deploy = &composego.DeployConfig{}
		ports = []composego.ServicePortConfig{}
		expose = composego.StringOrNumberList{}
		volumes = []composego.ServiceVolumeConfig{}
		environment = composego.MappingWithEquals{}
		healthcheck = composego.HealthCheckConfig{}
		projectVolumes = composego.Volumes{}

		k8sconf = config.K8SConfiguration{}
	})

	JustBeforeEach(func() {
		ext, err := k8sconf.ToMap()
		Expect(err).NotTo(HaveOccurred())
		extensions = map[string]interface{}{
			config.K8SExtensionKey: ext,
		}

		projectService, err = NewProjectService(composego.ServiceConfig{
			Name:        projectServiceName,
			Labels:      labels,
			Deploy:      deploy,
			Ports:       ports,
			Expose:      expose,
			Environment: environment,
			HealthCheck: &healthcheck,
			Volumes:     volumes,
			Extensions:  extensions,
		})
		Expect(err).NotTo(HaveOccurred())

		services := composego.Services{}
		services = append(services, projectService.ServiceConfig)

		project = composego.Project{
			Volumes:    projectVolumes,
			Services:   services,
			Extensions: extensions,
		}
	})

	Describe("enabled", func() {
		When("component toggle extension is set to disable=true", func() {
			BeforeEach(func() {
				k8sconf.Disabled = true
			})

			It("returns true", func() {
				Expect(projectService.enabled()).To(BeFalse())
			})
		})

		When("component toggle extension to set disable=false", func() {
			BeforeEach(func() {
				k8sconf.Disabled = false
			})

			It("returns false", func() {
				Expect(projectService.enabled()).To(BeTrue())
			})
		})

		When("component toggle extension is not specified", func() {
			It("defaults to true", func() {
				Expect(projectService.enabled()).To(BeTrue())
			})
		})
	})

	Describe("replicas", func() {

		replicas := 10

		Context("when provided via extension", func() {

			BeforeEach(func() {
				k8sconf.Workload.Replicas = replicas
			})

			It("will use a label value", func() {
				Expect(projectService.replicas()).To(BeEquivalentTo(replicas))
			})
		})

		Context("when provided via both the extension and as part of the project service spec", func() {

			BeforeEach(func() {
				k8sconf.Workload.Replicas = replicas

				deployBlockReplicas := uint64(2)
				deploy = &composego.DeployConfig{
					Replicas: &deployBlockReplicas,
				}
			})

			It("will use a label value", func() {
				Expect(projectService.replicas()).To(BeEquivalentTo(replicas))
			})
		})

		Context("when replicas extension not present but specified as part of the project service spec", func() {
			replicas := uint64(2)

			BeforeEach(func() {
				deploy = &composego.DeployConfig{
					Replicas: &replicas,
				}
			})

			It("will use a replica number as specified in deploy block", func() {
				Expect(projectService.Deploy.Replicas).NotTo(BeNil())
				Expect(projectService.replicas()).To(BeEquivalentTo(replicas))
			})
		})

		Context("when there is no replicas extensions supplied nor deploy block contains number of replicas", func() {
			It("will use default number of replicas", func() {
				Expect(projectService.replicas()).To(BeEquivalentTo(config.DefaultReplicaNumber))
			})
		})
	})

	Describe("autoscaleMaxReplicas", func() {
		replicas := 10

		Context("when provided via label", func() {

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelWorkloadAutoscaleMaxReplicas: strconv.Itoa(replicas),
				}
			})

			It("will use a label value", func() {
				Expect(projectService.autoscaleMaxReplicas()).To(BeEquivalentTo(replicas))
			})
		})

		Context("when there is no autoscale max replicas label supplied", func() {
			It("will use default max number of replicas for autoscaling purposes ", func() {
				Expect(projectService.autoscaleMaxReplicas()).To(BeEquivalentTo(config.DefaultAutoscaleMaxReplicaNumber))
			})
		})
	})

	Describe("autoscaleTargetCPUUtilization", func() {
		cpuThreshold := 70 // 70% utilization should kick off the autoscaling

		Context("when provided via label", func() {

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelWorkloadAutoscaleCPUUtilizationThreshold: strconv.Itoa(cpuThreshold),
				}
			})

			It("will use a label value", func() {
				Expect(projectService.autoscaleTargetCPUUtilization()).To(BeEquivalentTo(cpuThreshold))
			})
		})

		Context("when there is no autoscale target CPU utilization label supplied", func() {
			It("will use default CPU threshold for autoscaling purposes ", func() {
				Expect(projectService.autoscaleTargetCPUUtilization()).To(BeEquivalentTo(config.DefaultAutoscaleCPUThreshold))
			})
		})
	})

	Describe("autoscaleTargetMemoryUtilization", func() {
		memThreshold := 70 // 70% utilization should kick off the autoscaling

		Context("when provided via label", func() {

			BeforeEach(func() {
				labels = composego.Labels{
					config.LabelWorkloadAutoscaleMemoryUtilizationThreshold: strconv.Itoa(memThreshold),
				}
			})

			It("will use a label value", func() {
				Expect(projectService.autoscaleTargetMemoryUtilization()).To(BeEquivalentTo(memThreshold))
			})
		})

		Context("when there is no autoscale target Memory utilization label supplied", func() {
			It("will use default Memory threshold for autoscaling purposes ", func() {
				Expect(projectService.autoscaleTargetMemoryUtilization()).To(BeEquivalentTo(config.DefaultAutoscaleMemoryThreshold))
			})
		})
	})

	Describe("workloadType", func() {

		Context("when provided via label", func() {

			workloadType := "StatefulSet"

			JustBeforeEach(func() {
				projectService.K8SConfig.Workload.Type = workloadType
			})

			It("will use a extension value", func() {
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

			JustBeforeEach(func() {
				projectService.K8SConfig.Workload.Type = projectWorkloadType
				m, err := projectService.K8SConfig.ToMap()
				Expect(err).NotTo(HaveOccurred())

				svc := projectService.ServiceConfig
				svc.Deploy = &composego.DeployConfig{
					Mode: "global",
				}
				svc.Extensions = map[string]interface{}{
					config.K8SExtensionKey: m,
				}

				projectService, err = NewProjectService(svc)
				Expect(err).NotTo(HaveOccurred())
			})

			It("warns the user about the mismatch", func() {
				Expect(projectService.K8SConfig.Workload.Type).To(Equal(projectWorkloadType))
				Expect(projectService.Deploy.Mode).To(Equal("global"))

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

		Context("when provided via extension", func() {
			validType := config.ClusterIPService

			BeforeEach(func() {
				k8sconf.Service.Type = validType
			})

			It("returns service type as expected", func() {
				Expect(projectService.serviceType()).To(Equal(validType))
			})
		})

		Context("when not specified via extension", func() {
			It("returns service type as expected", func() {
				Expect(projectService.serviceType()).To(Equal(config.DefaultService))
			})
		})

		Describe("validations", func() {

			Context("with an invalid service value", func() {

				invalidType := "some-invalid-type"

				It("returns an error", func() {
					k8sconf := config.K8SConfiguration{}
					k8sconf.Service.Type = invalidType

					m, err := k8sconf.ToMap()
					Expect(err).NotTo(HaveOccurred())

					_, err = NewProjectService(composego.ServiceConfig{
						Name: "some service",
						Extensions: map[string]interface{}{
							config.K8SExtensionKey: m,
						},
					})
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("K8SConfiguration.Service.Type"))
				})
			})

			Context("when node port is specified via extension but service type was different that NodePort", func() {
				nodePort := "1234"

				BeforeEach(func() {
					k8sconf.Service.Type = config.ClusterIPService
					labels = composego.Labels{
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
					k8sconf.Service.Type = config.NodePortService
					labels = composego.Labels{
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

			JustBeforeEach(func() {
				projectService.K8SConfig.Workload.RestartPolicy = policy
				m, err := projectService.K8SConfig.ToMap()
				Expect(err).NotTo(HaveOccurred())

				svc := projectService.ServiceConfig
				svc.Extensions = map[string]interface{}{
					config.K8SExtensionKey: m,
				}

				projectService, err = NewProjectService(svc)
				Expect(err).NotTo(HaveOccurred())
				Expect(projectService.K8SConfig.Workload.RestartPolicy).To(Equal(policy))
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

				JustBeforeEach(func() {
					projectService.Restart = policy
					ps, err := NewProjectService(projectService.ServiceConfig)
					Expect(err).NotTo(HaveOccurred())
					projectService = ps
				})

				It("returns restart policy defined on the project service level", func() {
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
							"restart-policy": policy,
						},
					)
				})

			})

			Context("when invalid policy has been provided", func() {
				policy := "invalid-policy"

				JustBeforeEach(func() {
					projectService.K8SConfig.Workload.RestartPolicy = policy
					m, err := projectService.K8SConfig.ToMap()
					Expect(err).NotTo(HaveOccurred())

					svc := projectService.ServiceConfig
					svc.Extensions = map[string]interface{}{
						config.K8SExtensionKey: m,
					}

					projectService, err = NewProjectService(svc)
					Expect(err).NotTo(HaveOccurred())
					Expect(projectService.K8SConfig.Workload.RestartPolicy).To(Equal(policy))
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

	Describe("liveness probe", func() {
		Context("when valid healthcheck and probe type are defined", func() {
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
				result, err := projectService.LivenessProbe()
				Expect(err).NotTo(HaveOccurred())
				Expect(cmp.Diff(result, &v1.Probe{
					Handler: v1.Handler{
						Exec: &v1.ExecAction{
							Command: []string{"my command"},
						},
					},
					TimeoutSeconds:      10,
					PeriodSeconds:       10,
					InitialDelaySeconds: 10,
					FailureThreshold:    3,
					SuccessThreshold:    3,
				})).To(BeEmpty())
			})
		})

		Describe("validations", func() {
			BeforeEach(func() {
				k8sconf.Workload.LivenessProbe.Type = config.ProbeTypeExec.String()
			})

			Context("when Test command is not defined", func() {
				BeforeEach(func() {
					healthcheck = composego.HealthCheckConfig{
						Test: composego.HealthCheckTest{
							"",
						},
					}
				})

				It("logs and returns error", func() {
					_, err := projectService.LivenessProbe()
					Expect(err).NotTo(HaveOccurred())
					// Expect(err).To(MatchError("Health check misconfigured"))

					// assertLog(logrus.ErrorLevel,
					// 	"Health check misconfigured",
					// 	map[string]string{},
					// )
				})
			})

			When("any of time based paramaters is set to 0", func() {
				BeforeEach(func() {
					k8sconf.Workload.LivenessProbe.Type = config.ProbeTypeExec.String()
					k8sconf.Workload.LivenessProbe.Timeout = 0
				})

				It("logs and returns error", func() {
					_, err := projectService.LivenessProbe()
					Expect(err).NotTo(HaveOccurred())
					// Expect(err).To(HaveOccurred())
					// Expect(err).To(MatchError("Health check misconfigured"))

					// assertLog(logrus.ErrorLevel,
					// 	"Health check misconfigured",
					// 	map[string]string{},
					// )
				})
			})
		})
	})

	Describe("livenessHTTPProbe", func() {
		When("defined via labels", func() {
			Context("with all the parameters", func() {
				BeforeEach(func() {
					k8sconf.Workload.LivenessProbe.Type = config.ProbeTypeHTTP.String()
					k8sconf.Workload.LivenessProbe.HTTP.Path = "/status"
					k8sconf.Workload.LivenessProbe.HTTP.Port = 8080
				})

				It("returns a handler", func() {
					result, err := projectService.LivenessProbe()
					Expect(err).To(BeNil())
					Expect(result.HTTPGet.Port.IntValue()).To(Equal(8080))
					Expect(result.HTTPGet.Path).To(Equal("/status"))
				})
			})

			Context("with missing path", func() {
				BeforeEach(func() {
					k8sconf.Workload.LivenessProbe.Type = config.ProbeTypeHTTP.String()
					k8sconf.Workload.LivenessProbe.HTTP.Port = 8080
				})

				It("returns an error", func() {
					Expect(projectService.Extensions).To(Equal(extensions))

					lp, err := projectService.LivenessProbe()
					Expect(err).NotTo(HaveOccurred())
					Expect(lp.HTTPGet).NotTo(BeNil())
					Expect(lp.HTTPGet.Path).To(Equal(""))
				})
			})
		})
	})

	Describe("livenessProbeTCP", func() {
		When("defined via labels", func() {

			Context("and supplied as string port", func() {
				port := "8080"

				BeforeEach(func() {
					k8sconf.Workload.LivenessProbe.Type = config.ProbeTypeTCP.String()
					k8sconf.Workload.LivenessProbe.TCP.Port = 8080
				})

				It("returns label value", func() {
					p, err := projectService.LivenessProbe()
					Expect(err).To(Succeed())
					Expect(p.TCPSocket.Port.String()).To(Equal(port))
				})
			})

			Context("and empty port", func() {
				BeforeEach(func() {
					k8sconf.Workload.LivenessProbe.Type = config.ProbeTypeTCP.String()
					k8sconf.Workload.LivenessProbe.TCP.Port = 0
				})

				It("returns an error", func() {
					p, err := projectService.LivenessProbe()
					Expect(err).NotTo(HaveOccurred())
					Expect(p.TCPSocket.Port.String()).To(Equal("0"))
				})
			})

			Context("and no port label", func() {
				BeforeEach(func() {
					k8sconf.Workload.LivenessProbe.Type = config.ProbeTypeTCP.String()
				})

				It("returns an error", func() {
					p, err := projectService.LivenessProbe()
					Expect(err).NotTo(HaveOccurred())
					p.TCPSocket.Port = intstr.FromString("")
				})
			})
		})

	})

	// 	Describe("livenessProbeCommand", func() {
	// 		When("defined via labels", func() {

	// 			Context("and supplied as string command", func() {
	// 				cmd := "my test command"

	// 				BeforeEach(func() {
	// 					labels = composego.Labels{
	// 						config.LabelWorkloadLivenessProbeCommand: cmd,
	// 					}
	// 				})

	// 				It("returns label value", func() {
	// 					result, err := projectService.LivenessProbe()
	// 					Expect(err).NotTo(HaveOccurred())
	// 					Expect().To(HaveLen(1))
	// 					Expect(projectService.livenessProbeCommand()).To(ContainElement(cmd))
	// 				})
	// 			})

	// 			Context("and specified as list", func() {
	// 				cmd := "[\"CMD\", \"echo\", \"Hello World\"]"

	// 				BeforeEach(func() {
	// 					labels = composego.Labels{
	// 						config.LabelWorkloadLivenessProbeCommand: cmd,
	// 					}
	// 				})

	// 				It("returns label value", func() {
	// 					Expect(projectService.livenessProbeCommand()).To(HaveLen(2))
	// 					Expect(projectService.livenessProbeCommand()).ToNot(ContainElements("CMD"))
	// 					Expect(projectService.livenessProbeCommand()).To(ContainElements("echo", "Hello World"))
	// 				})
	// 			})
	// 		})

	// 		When("defined via healthcheck block only", func() {
	// 			cmd := []string{
	// 				"CMD-SHELL",
	// 				"/my-test/command.sh",
	// 				"some-args",
	// 			}

	// 			JustBeforeEach(func() {
	// 				projectService.HealthCheck = &composego.HealthCheckConfig{
	// 					Test: composego.HealthCheckTest(cmd),
	// 				}
	// 			})

	// 			It("returns project service healthcheck test command", func() {
	// 				Expect(projectService.livenessProbeCommand()).To(HaveLen(2))
	// 				Expect(projectService.livenessProbeCommand()).To(ContainElements(cmd[1:]))
	// 			})
	// 		})

	// 		When("not defined by both label nor healthcheck block", func() {
	// 			It("returns default value", func() {
	// 				Expect(projectService.livenessProbeCommand()).To(ContainElements(config.DefaultLivenessProbeCommand))
	// 			})
	// 		})
	// 	})

	// 	Describe("livenessProbeInterval", func() {
	// 		When("defined via labels", func() {
	// 			interval := "30s"

	// 			BeforeEach(func() {
	// 				labels = composego.Labels{
	// 					config.LabelWorkloadLivenessProbeInterval: interval,
	// 				}
	// 			})

	// 			It("returns label value", func() {
	// 				Expect(projectService.livenessProbeInterval()).To(BeEquivalentTo(30))
	// 			})
	// 		})

	// 		When("defined via healthcheck block only", func() {
	// 			seconds := 10
	// 			interval := composego.Duration(time.Duration(seconds) * time.Second)

	// 			JustBeforeEach(func() {
	// 				projectService.HealthCheck = &composego.HealthCheckConfig{
	// 					Interval: &interval,
	// 				}
	// 			})

	// 			It("returns project service healthcheck interval", func() {
	// 				Expect(projectService.livenessProbeInterval()).To(BeEquivalentTo(seconds))
	// 			})
	// 		})

	// 		When("not defined by both label nor healthcheck block", func() {
	// 			It("returns default value", func() {
	// 				expected, _ := durationStrToSecondsInt(config.DefaultProbeInterval)
	// 				Expect(projectService.livenessProbeInterval()).To(Equal(*expected))
	// 			})
	// 		})
	// 	})

	// 	Describe("livenessProbeTimeout", func() {
	// 		When("defined via labels", func() {
	// 			timeout := "30s"

	// 			BeforeEach(func() {
	// 				labels = composego.Labels{
	// 					config.LabelWorkloadLivenessProbeTimeout: timeout,
	// 				}
	// 			})

	// 			It("returns label value", func() {
	// 				Expect(projectService.livenessProbeTimeout()).To(BeEquivalentTo(30))
	// 			})
	// 		})

	// 		When("defined via healthcheck block only", func() {
	// 			seconds := 3
	// 			timeout := composego.Duration(time.Duration(seconds) * time.Second)

	// 			JustBeforeEach(func() {
	// 				projectService.HealthCheck = &composego.HealthCheckConfig{
	// 					Timeout: &timeout,
	// 				}
	// 			})

	// 			It("returns project service healthcheck timeout", func() {
	// 				Expect(projectService.livenessProbeTimeout()).To(BeEquivalentTo(seconds))
	// 			})
	// 		})

	// 		When("not defined by both label nor healthcheck block", func() {
	// 			It("returns default value", func() {
	// 				expected, _ := durationStrToSecondsInt(config.DefaultProbeTimeout)
	// 				Expect(projectService.livenessProbeTimeout()).To(Equal(*expected))
	// 			})
	// 		})
	// 	})

	// 	Describe("livenessProbeInitialDelay", func() {
	// 		When("defined via labels", func() {
	// 			delay := "30s"

	// 			BeforeEach(func() {
	// 				labels = composego.Labels{
	// 					config.LabelWorkloadLivenessProbeInitialDelay: delay,
	// 				}
	// 			})

	// 			It("returns label value", func() {
	// 				Expect(projectService.livenessProbeInitialDelay()).To(BeEquivalentTo(30))
	// 			})
	// 		})

	// 		When("defined via healthcheck block only", func() {
	// 			seconds := 5
	// 			startPeriod := composego.Duration(time.Duration(seconds) * time.Second)

	// 			JustBeforeEach(func() {
	// 				projectService.HealthCheck = &composego.HealthCheckConfig{
	// 					StartPeriod: &startPeriod,
	// 				}
	// 			})

	// 			It("returns project service healthcheck start period", func() {
	// 				Expect(projectService.livenessProbeInitialDelay()).To(BeEquivalentTo(seconds))
	// 			})
	// 		})

	// 		When("not defined by both label nor healthcheck block", func() {
	// 			It("returns default value", func() {
	// 				expected, _ := durationStrToSecondsInt(config.DefaultProbeInitialDelay)
	// 				Expect(projectService.livenessProbeInitialDelay()).To(Equal(*expected))
	// 			})
	// 		})
	// 	})

	// 	Describe("livenessProbeRetries", func() {
	// 		When("defined via labels", func() {
	// 			retries := "3"

	// 			BeforeEach(func() {
	// 				labels = composego.Labels{
	// 					config.LabelWorkloadLivenessProbeRetries: retries,
	// 				}
	// 			})

	// 			It("returns label value", func() {
	// 				expected := 3
	// 				Expect(projectService.livenessProbeRetries()).To(BeEquivalentTo(expected))
	// 			})
	// 		})

	// 		When("defined via healthcheck block only", func() {
	// 			retries := uint64(5)

	// 			JustBeforeEach(func() {
	// 				projectService.HealthCheck = &composego.HealthCheckConfig{
	// 					Retries: &retries,
	// 				}
	// 			})

	// 			It("returns project service healthcheck retries", func() {
	// 				Expect(projectService.livenessProbeRetries()).To(BeEquivalentTo(retries))
	// 			})
	// 		})

	// 		When("not defined by both label nor healthcheck block", func() {
	// 			It("returns default value", func() {
	// 				Expect(projectService.livenessProbeRetries()).To(BeEquivalentTo(config.DefaultProbeRetries))
	// 			})
	// 		})
	// 	})

	// 	Describe("readinessProbe", func() {

	// 		Describe("validations", func() {

	// 			When("any of time based paramaters is set to 0", func() {
	// 				JustBeforeEach(func() {
	// 					projectService.Labels = composego.Labels{
	// 						config.LabelWorkloadReadinessProbeType:    config.ProbeTypeExec.String(),
	// 						config.LabelWorkloadReadinessProbeTimeout: "0",
	// 					}
	// 				})

	// 				It("logs and returns error", func() {
	// 					_, err := projectService.readinessProbe()
	// 					Expect(err).To(HaveOccurred())
	// 					Expect(err).To(MatchError("Readiness probe misconfigured"))

	// 					assertLog(logrus.ErrorLevel,
	// 						"Readiness probe misconfigured",
	// 						map[string]string{},
	// 					)
	// 				})
	// 			})
	// 		})
	// 	})

	// 	Describe("readinessProbeTCP", func() {
	// 		When("defined via labels", func() {

	// 			Context("and supplied as string port", func() {
	// 				port := "8080"

	// 				BeforeEach(func() {
	// 					labels = composego.Labels{
	// 						config.LabelWorkloadReadinessProbeType:    config.ProbeTypeTCP.String(),
	// 						config.LabelWorkloadReadinessProbeTCPPort: port,
	// 					}
	// 				})

	// 				It("returns label value", func() {
	// 					p, err := projectService.readinessProbe()
	// 					Expect(err).To(Succeed())
	// 					Expect(p.TCPSocket.Port.String()).To(Equal(port))
	// 				})
	// 			})

	// 			Context("and empty port", func() {
	// 				BeforeEach(func() {
	// 					labels = composego.Labels{
	// 						config.LabelWorkloadReadinessProbeType:    config.ProbeTypeTCP.String(),
	// 						config.LabelWorkloadReadinessProbeTCPPort: "",
	// 					}
	// 				})

	// 				It("returns an error", func() {
	// 					p, err := projectService.readinessProbe()
	// 					Expect(err).To(HaveOccurred())
	// 					Expect(err.Error()).To(Equal(fmt.Sprintf("%s needs to be a number", config.LabelWorkloadReadinessProbeTCPPort)))
	// 					Expect(p).To(BeNil())
	// 				})
	// 			})

	// 			Context("with NaN port", func() {
	// 				BeforeEach(func() {
	// 					labels = composego.Labels{
	// 						config.LabelWorkloadReadinessProbeType:    config.ProbeTypeTCP.String(),
	// 						config.LabelWorkloadReadinessProbeTCPPort: "asds",
	// 					}
	// 				})

	// 				It("returns an error", func() {
	// 					_, err := projectService.readinessProbe()
	// 					Expect(err).NotTo(Succeed())
	// 					Expect(err.Error()).To(ContainSubstring("%s needs to be a number", config.LabelWorkloadReadinessProbeTCPPort))
	// 				})
	// 			})

	// 			Context("and no port label", func() {
	// 				BeforeEach(func() {
	// 					labels = composego.Labels{
	// 						config.LabelWorkloadReadinessProbeType: config.ProbeTypeTCP.String(),
	// 					}
	// 				})

	// 				It("returns an error", func() {
	// 					p, err := projectService.readinessProbe()
	// 					Expect(err).To(HaveOccurred())
	// 					Expect(err.Error()).To(Equal(fmt.Sprintf("%s not correctly defined", config.LabelWorkloadReadinessProbeTCPPort)))
	// 					Expect(p).To(BeNil())
	// 				})
	// 			})
	// 		})

	// 		When("not defined by label", func() {
	// 			It("returns default value as empty string slice", func() {
	// 				pt, err := projectService.readinessProbeType()
	// 				Expect(err).To(Succeed())
	// 				Expect(*pt).To(Equal(config.ProbeTypeNone))
	// 				Expect(projectService.readinessProbeCommand()).To(Equal([]string{}))
	// 			})
	// 		})
	// 	})

	// 	Describe("readinessProbeHTTP", func() {
	// 		Context("and supplied as string port and path", func() {
	// 			port := "8080"
	// 			path := "/status"

	// 			BeforeEach(func() {
	// 				labels = composego.Labels{
	// 					config.LabelWorkloadReadinessProbeType:     config.ProbeTypeHTTP.String(),
	// 					config.LabelWorkloadReadinessProbeHTTPPort: port,
	// 					config.LabelWorkloadReadinessProbeHTTPPath: path,
	// 				}
	// 			})

	// 			It("returns readiness probe with HTTPGet", func() {
	// 				p, err := projectService.readinessProbe()
	// 				Expect(err).To(Succeed())
	// 				Expect(p.HTTPGet.Port.String()).To(Equal(port))
	// 				Expect(p.HTTPGet.Path).To(Equal(path))
	// 			})
	// 		})

	// 		Context("and empty port", func() {
	// 			BeforeEach(func() {
	// 				labels = composego.Labels{
	// 					config.LabelWorkloadReadinessProbeType:     config.ProbeTypeHTTP.String(),
	// 					config.LabelWorkloadReadinessProbeHTTPPort: "",
	// 					config.LabelWorkloadReadinessProbeHTTPPath: "/status",
	// 				}
	// 			})

	// 			It("returns an error", func() {
	// 				p, err := projectService.readinessProbe()
	// 				Expect(err).To(HaveOccurred())
	// 				Expect(err.Error()).To(Equal(fmt.Sprintf("%s needs to be a number", config.LabelWorkloadReadinessProbeHTTPPort)))
	// 				Expect(p).To(BeNil())
	// 			})
	// 		})

	// 		Context("with NaN port", func() {
	// 			BeforeEach(func() {
	// 				labels = composego.Labels{
	// 					config.LabelWorkloadReadinessProbeType:     config.ProbeTypeHTTP.String(),
	// 					config.LabelWorkloadReadinessProbeHTTPPort: "asd",
	// 					config.LabelWorkloadReadinessProbeHTTPPath: "/status",
	// 				}
	// 			})

	// 			It("returns an error", func() {
	// 				_, err := projectService.readinessProbe()
	// 				Expect(err).NotTo(BeNil())
	// 				Expect(err.Error()).To(ContainSubstring("%s needs to be a number", config.LabelWorkloadReadinessProbeHTTPPort))
	// 			})
	// 		})

	// 		Context("and no port label", func() {
	// 			BeforeEach(func() {
	// 				labels = composego.Labels{
	// 					config.LabelWorkloadReadinessProbeType:     config.ProbeTypeHTTP.String(),
	// 					config.LabelWorkloadReadinessProbeHTTPPath: "/status",
	// 				}
	// 			})

	// 			It("returns an error", func() {
	// 				p, err := projectService.readinessProbe()
	// 				Expect(err).To(HaveOccurred())
	// 				Expect(err.Error()).To(Equal(fmt.Sprintf("%s not correctly defined", config.LabelWorkloadReadinessProbeHTTPPort)))
	// 				Expect(p).To(BeNil())
	// 			})
	// 		})
	// 	})

	// 	Describe("readinessProbeCommand", func() {
	// 		When("defined via labels", func() {

	// 			Context("and supplied as string command", func() {
	// 				cmd := "my test command"

	// 				BeforeEach(func() {
	// 					labels = composego.Labels{
	// 						config.LabelWorkloadReadinessProbeType:    config.ProbeTypeExec.String(),
	// 						config.LabelWorkloadReadinessProbeCommand: cmd,
	// 					}
	// 				})

	// 				It("returns label value", func() {
	// 					Expect(projectService.readinessProbeCommand()).To(HaveLen(1))
	// 					Expect(projectService.readinessProbeCommand()).To(ContainElement(cmd))
	// 				})
	// 			})

	// 			Context("and specified as list", func() {
	// 				cmd := "[\"CMD\", \"echo\", \"Hello World\"]"

	// 				BeforeEach(func() {
	// 					labels = composego.Labels{
	// 						config.LabelWorkloadReadinessProbeType:    config.ProbeTypeExec.String(),
	// 						config.LabelWorkloadReadinessProbeCommand: cmd,
	// 					}
	// 				})

	// 				It("returns label value", func() {
	// 					Expect(projectService.readinessProbeCommand()).To(HaveLen(2))
	// 					Expect(projectService.readinessProbeCommand()).ToNot(ContainElements("CMD"))
	// 					Expect(projectService.readinessProbeCommand()).To(ContainElements("echo", "Hello World"))
	// 				})
	// 			})
	// 		})

	// 		When("not defined by label", func() {
	// 			It("returns default value as empty string slice", func() {
	// 				pt, err := projectService.readinessProbeType()
	// 				Expect(err).To(Succeed())
	// 				Expect(*pt).To(Equal(config.ProbeTypeNone))
	// 				Expect(projectService.readinessProbeCommand()).To(Equal([]string{}))
	// 			})
	// 		})
	// 	})

	// 	Describe("readinessProbeInterval", func() {
	// 		When("defined via labels", func() {
	// 			interval := "30s"

	// 			BeforeEach(func() {
	// 				labels = composego.Labels{
	// 					config.LabelWorkloadReadinessProbeType:     config.ProbeTypeExec.String(),
	// 					config.LabelWorkloadReadinessProbeInterval: interval,
	// 				}
	// 			})

	// 			It("returns label value", func() {
	// 				Expect(projectService.readinessProbeInterval()).To(BeEquivalentTo(30))
	// 			})
	// 		})

	// 		When("not defined by label", func() {
	// 			It("returns default value", func() {
	// 				expected, _ := durationStrToSecondsInt(config.DefaultProbeInterval)
	// 				Expect(projectService.readinessProbeInterval()).To(Equal(*expected))
	// 			})
	// 		})
	// 	})

	// 	Describe("readinessProbeTimeout", func() {
	// 		When("defined via labels", func() {
	// 			timeout := "30s"

	// 			BeforeEach(func() {
	// 				labels = composego.Labels{
	// 					config.LabelWorkloadReadinessProbeType:    config.ProbeTypeExec.String(),
	// 					config.LabelWorkloadReadinessProbeTimeout: timeout,
	// 				}
	// 			})

	// 			It("returns label value", func() {
	// 				Expect(projectService.readinessProbeTimeout()).To(BeEquivalentTo(30))
	// 			})
	// 		})

	// 		When("not defined by label", func() {
	// 			It("returns default value", func() {
	// 				expected, _ := durationStrToSecondsInt(config.DefaultProbeTimeout)
	// 				Expect(projectService.readinessProbeTimeout()).To(Equal(*expected))
	// 			})
	// 		})
	// 	})

	// 	Describe("readinessProbeInitialDelay", func() {
	// 		When("defined via labels", func() {
	// 			delay := "30s"

	// 			BeforeEach(func() {
	// 				labels = composego.Labels{
	// 					config.LabelWorkloadReadinessProbeType:         config.ProbeTypeExec.String(),
	// 					config.LabelWorkloadReadinessProbeInitialDelay: delay,
	// 				}
	// 			})

	// 			It("returns label value", func() {
	// 				Expect(projectService.readinessProbeInitialDelay()).To(BeEquivalentTo(30))
	// 			})
	// 		})

	// 		When("not defined by label", func() {
	// 			It("returns default value", func() {
	// 				expected, _ := durationStrToSecondsInt(config.DefaultProbeInitialDelay)
	// 				Expect(projectService.readinessProbeInitialDelay()).To(Equal(*expected))
	// 			})
	// 		})
	// 	})

	// 	Describe("readinessProbeRetries", func() {
	// 		When("defined via labels", func() {
	// 			retries := "3"

	// 			BeforeEach(func() {
	// 				labels = composego.Labels{
	// 					config.LabelWorkloadReadinessProbeType:    config.ProbeTypeExec.String(),
	// 					config.LabelWorkloadReadinessProbeRetries: retries,
	// 				}
	// 			})

	// 			It("returns label value", func() {
	// 				expected := 3
	// 				Expect(projectService.readinessProbeRetries()).To(BeEquivalentTo(expected))
	// 			})
	// 		})

	// 		When("not defined by label", func() {
	// 			It("returns default value", func() {
	// 				Expect(projectService.readinessProbeRetries()).To(BeEquivalentTo(config.DefaultProbeRetries))
	// 			})
	// 		})
	// 	})

	// 	Describe("readinessProbeDisabled", func() {
	// 		When("defined via labels with valid (truth) value", func() {
	// 			BeforeEach(func() {
	// 				labels = composego.Labels{
	// 					config.LabelWorkloadReadinessProbeType: config.ProbeTypeNone.String(),
	// 				}
	// 			})

	// 			It("returns label value", func() {
	// 				pt, err := projectService.readinessProbeType()
	// 				Expect(err).To(Succeed())
	// 				Expect(*pt).To(Equal(config.ProbeTypeNone))
	// 			})

	// 			It("returns a nil probe", func() {
	// 				probe, err := projectService.readinessProbe()
	// 				Expect(err).To(Succeed())
	// 				Expect(probe).To(BeNil())
	// 			})
	// 		})

	// 		When("defined via labels with invalid (non truthy) value", func() {
	// 			disabled := "FOO"

	// 			BeforeEach(func() {
	// 				labels = composego.Labels{
	// 					config.LabelWorkloadReadinessProbeType: disabled,
	// 				}
	// 			})

	// 			It("returns an error", func() {
	// 				p, err := projectService.readinessProbe()
	// 				Expect(err).To(HaveOccurred())
	// 				Expect(err.Error()).To(ContainSubstring("not a supported readiness probe type"))
	// 				Expect(p).To(BeNil())
	// 			})
	// 		})

	// 		When("not defined via labels at all", func() {
	// 			It("returns default value - disable by default", func() {
	// 				p, err := projectService.readinessProbe()
	// 				Expect(err).To(Succeed())
	// 				Expect(p).To(BeNil())
	// 			})
	// 		})
	// 	})

})
