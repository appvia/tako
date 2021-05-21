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
		extensions         map[string]interface{}
		deploy             *composego.DeployConfig
		ports              []composego.ServicePortConfig
		expose             composego.StringOrNumberList
		volumes            []composego.ServiceVolumeConfig
		environment        composego.MappingWithEquals
		healthcheck        composego.HealthCheckConfig
		projectVolumes     composego.Volumes
		svcK8sConfig       config.SvcK8sConfig
	)

	BeforeEach(func() {
		projectServiceName = "db"
		extensions = make(map[string]interface{})
		deploy = &composego.DeployConfig{}
		ports = []composego.ServicePortConfig{}
		expose = composego.StringOrNumberList{}
		volumes = []composego.ServiceVolumeConfig{}
		environment = composego.MappingWithEquals{}
		healthcheck = composego.HealthCheckConfig{}
		projectVolumes = composego.Volumes{}

		svcK8sConfig = config.SvcK8sConfig{}
	})

	JustBeforeEach(func() {
		ext, err := svcK8sConfig.ToMap()
		Expect(err).NotTo(HaveOccurred())
		extensions = map[string]interface{}{
			config.K8SExtensionKey: ext,
		}

		projectService, err = NewProjectService(composego.ServiceConfig{
			Name:        projectServiceName,
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
				svcK8sConfig.Disabled = true
			})

			It("returns true", func() {
				Expect(projectService.enabled()).To(BeFalse())
			})
		})

		When("component toggle extension to set disable=false", func() {
			BeforeEach(func() {
				svcK8sConfig.Disabled = false
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
				svcK8sConfig.Workload.Replicas = replicas
			})

			It("will use the extension value", func() {
				Expect(projectService.replicas()).To(BeEquivalentTo(replicas))
			})
		})

		Context("when provided via both the extension and as part of the project service spec", func() {

			BeforeEach(func() {
				svcK8sConfig.Workload.Replicas = replicas

				deployBlockReplicas := uint64(2)
				deploy = &composego.DeployConfig{
					Replicas: &deployBlockReplicas,
				}
			})

			It("will use the extension value", func() {
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

		Context("when provided via extension", func() {
			BeforeEach(func() {
				svcK8sConfig.Workload.Autoscale.MaxReplicas = replicas
			})

			It("will use the extension value", func() {
				Expect(projectService.autoscaleMaxReplicas()).To(BeEquivalentTo(replicas))
			})
		})

		Context("when autoscale max replicas is not supplied in the extension", func() {
			It("will use default max number of replicas for autoscaling purposes ", func() {
				Expect(projectService.autoscaleMaxReplicas()).To(BeEquivalentTo(config.DefaultAutoscaleMaxReplicaNumber))
			})
		})
	})

	Describe("autoscaleTargetCPUUtilization", func() {
		cpuThreshold := 80 // 80% utilization should kick off the autoscaling

		Context("when provided via an extension", func() {
			BeforeEach(func() {
				svcK8sConfig.Workload.Autoscale.CPUThreshold = cpuThreshold
			})

			It("will use the extension value", func() {
				Expect(projectService.autoscaleTargetCPUUtilization()).To(BeEquivalentTo(cpuThreshold))
			})
		})

		Context("when autoscale target CPU utilization is not supplied in the extension", func() {
			It("will use default CPU threshold for autoscaling purposes ", func() {
				Expect(projectService.autoscaleTargetCPUUtilization()).To(BeEquivalentTo(config.DefaultAutoscaleCPUThreshold))
			})
		})
	})

	Describe("autoscaleTargetMemoryUtilization", func() {
		memThreshold := 80 // 80% utilization should kick off the autoscaling

		Context("when provided via an extension", func() {
			BeforeEach(func() {
				svcK8sConfig.Workload.Autoscale.MemoryThreshold = memThreshold
			})

			It("will use the extension value", func() {
				Expect(projectService.autoscaleTargetMemoryUtilization()).To(BeEquivalentTo(memThreshold))
			})
		})

		Context("when the autoscale target Memory utilization is not supplied in the extension", func() {
			It("will use default Memory threshold for autoscaling purposes ", func() {
				Expect(projectService.autoscaleTargetMemoryUtilization()).To(BeEquivalentTo(config.DefaultAutoscaleMemoryThreshold))
			})
		})
	})

	Describe("workloadType", func() {

		Context("when provided via extension", func() {
			workloadType := config.StatefulSetWorkload

			JustBeforeEach(func() {
				projectService.SvcK8sConfig.Workload.Type = workloadType
			})

			It("will use a extension value", func() {
				Expect(projectService.workloadType()).To(Equal(workloadType))
			})
		})

		Context("when not specified via extension", func() {
			It("will use a default workload type", func() {
				Expect(projectService.workloadType()).To(Equal(config.DefaultWorkload))
			})
		})

		Context("when deploy block `mode` defined as `global` and workload type is different than DaemonSet", func() {
			projectWorkloadType := config.StatefulSetWorkload

			JustBeforeEach(func() {
				projectService.SvcK8sConfig.Workload.Type = projectWorkloadType
				m, err := projectService.SvcK8sConfig.ToMap()
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
				Expect(projectService.SvcK8sConfig.Workload.Type).To(Equal(projectWorkloadType))
				Expect(projectService.Deploy.Mode).To(Equal("global"))

				projectService.workloadType()
				assertLog(logrus.WarnLevel,
					"Compose service defined as 'global' should map to K8s DaemonSet. Current configuration forces conversion to StatefulSet",
					map[string]string{
						"workload-type":   projectWorkloadType.String(),
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
				svcK8sConfig.Service.Type = validType
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

				var m map[string]interface{}
				invalidType := "some-invalid-type"

				JustBeforeEach(func() {
					var err error
					svcK8sConfig := config.SvcK8sConfig{}
					svcK8sConfig.Service.Type = invalidType

					m, err = svcK8sConfig.ToMap()
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns an error", func() {
					_, err := NewProjectService(composego.ServiceConfig{
						Name: "some service",
						Extensions: map[string]interface{}{
							config.K8SExtensionKey: m,
						},
					})
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("SvcK8sConfig.Service.Type"))
				})
			})

			Context("when node port is specified via extension but service type was different that NodePort", func() {
				nodePort := 1234

				BeforeEach(func() {
					svcK8sConfig.Service.Type = config.ClusterIPService
					svcK8sConfig.Service.NodePort = nodePort
				})

				It("returns an error", func() {
					_, err := projectService.serviceType()
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(fmt.Sprintf("`%s` workload service type must be set as `NodePort` when assigning node port value", projectServiceName)))
				})
			})

			Context("when node port is specified via extension and project service has multiple ports specified", func() {
				nodePort := 1234

				BeforeEach(func() {
					svcK8sConfig.Service.Type = config.NodePortService
					svcK8sConfig.Service.NodePort = nodePort
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

		Context("when specified via an extension", func() {
			nodePort := 1234

			BeforeEach(func() {
				svcK8sConfig.Service.NodePort = nodePort
			})

			It("will use the extension value", func() {
				Expect(projectService.nodePort()).To(Equal(int32(nodePort)))
			})
		})

		Context("when not specified via an extension", func() {
			It("will return 0", func() {
				Expect(projectService.nodePort()).To(Equal(int32(0)))
			})
		})
	})

	Describe("exposeService", func() {

		Context("when specified via an extension", func() {
			expose := "domain.com"

			BeforeEach(func() {
				svcK8sConfig.Service.Expose.Domain = expose
			})

			It("will use the extension value", func() {
				Expect(projectService.exposeService()).To(Equal(expose))
			})
		})

		Context("when not specified via an extension", func() {
			It("will return empty string", func() {
				Expect(projectService.exposeService()).To(Equal(""))
			})
		})

		Describe("validations", func() {

			Context("when service hasn't been exposed via an extension but TLS secret was provided", func() {
				BeforeEach(func() {
					svcK8sConfig.Service.Expose.Domain = ""
					svcK8sConfig.Service.Expose.TlsSecret = "my-tls-secret-name"
				})

				It("returns an error", func() {
					_, err := projectService.exposeService()
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("service can't have TLS secret name when it hasn't been exposed"))
				})
			})

		})

	})

	Describe("tlsSecretName", func() {

		Context("when specified via an extension", func() {
			tls := "my-secret"

			BeforeEach(func() {
				svcK8sConfig.Service.Expose.TlsSecret = tls
			})

			It("will use the extension value", func() {
				Expect(projectService.tlsSecretName()).To(Equal(tls))
			})
		})

		Context("when not specified via an extension", func() {
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

				expectedMaxSurge := intstr.FromString("25%")
				expectedMaxUnavailable := intstr.FromInt(cast.ToInt(parallelism))

				It("returns appropriate RollingUpdateDeployment object", func() {
					projectService.SvcK8sConfig.Workload.RollingUpdateMaxSurge = 0
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

				expectedMaxUnavailable := intstr.FromString("25%")
				expectedMaxSurge := intstr.FromInt(cast.ToInt(parallelism))

				It("returns appropriate RollingUpdateDeployment object", func() {
					projectService.SvcK8sConfig.Workload.RollingUpdateMaxSurge = 0
					Expect(projectService.getKubernetesUpdateStrategy()).To(Equal(&v1apps.RollingUpdateDeployment{
						MaxUnavailable: &expectedMaxUnavailable,
						MaxSurge:       &expectedMaxSurge,
					}))
				})

			})

		})

	})

	Describe("volumes", func() {

		volumeName := "vol_a"
		targetPath := "/some/path"

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
						VolumeName:   rfc1123(volumeName),
						Container:    targetPath,
						PVCName:      projectServiceName + "-claim0",
						PVCSize:      config.DefaultVolumeSize,
						StorageClass: config.DefaultVolumeStorageClass,
					},
				}))
			})

			Context("when volume contains a storage class in k8s extension", func() {
				storageClass := "ssd"

				BeforeEach(func() {
					projectVolumes = composego.Volumes{
						volumeName: composego.VolumeConfig{
							Name: volumeName,
							Extensions: map[string]interface{}{
								config.K8SExtensionKey: map[string]interface{}{
									"storageClass": storageClass,
								},
							},
						},
					}
				})

				It("will set the storage class as expected", func() {
					v, _ := projectService.volumes(&project)
					Expect(v[0].StorageClass).To(Equal(storageClass))
				})
			})

			Context("when volume contains a volume size in k8s extension", func() {
				storageSize := "1Gi"

				BeforeEach(func() {
					projectVolumes = composego.Volumes{
						volumeName: composego.VolumeConfig{
							Name: volumeName,
							Extensions: map[string]interface{}{
								config.K8SExtensionKey: map[string]interface{}{
									"size": storageSize,
								},
							},
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
		Context("not specified by deploy block", func() {
			When("not specified via extension", func() {
				It("returns resource request as zero values", func() {
					mem, cpu := projectService.resourceRequests()
					Expect(*mem).To(BeEquivalentTo(0))
					Expect(*cpu).To(BeEquivalentTo(0))
				})
			})
		})

		Context("specified by deploy block", func() {
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

			When("not specified via extension", func() {
				It("returns resource request as defined in deploy block", func() {
					mem, cpu := projectService.resourceRequests()
					Expect(*mem).To(BeEquivalentTo(1000))
					Expect(*cpu).To(BeEquivalentTo(100))
				})
			})

			When("CPU request is specified via extension", func() {
				BeforeEach(func() {
					svcK8sConfig.Workload.Resource.CPU = "0.2"
				})

				It("returns CPU request as defined by the extension", func() {
					_, cpu := projectService.resourceRequests()
					Expect(*cpu).To(BeEquivalentTo(200))
				})
			})

			When("Memory request is specified via extension", func() {
				BeforeEach(func() {
					svcK8sConfig.Workload.Resource.Memory = "1M"
				})

				It("returns Memory request as defined by the extension", func() {
					mem, _ := projectService.resourceRequests()
					Expect(*mem).To(BeEquivalentTo(1000000))
				})
			})
		})
	})

	Describe("resourceLimits", func() {
		Context("not specified by deploy block", func() {
			When("not specified via extension", func() {
				It("returns resource limits as zero values", func() {
					mem, cpu := projectService.resourceLimits()
					Expect(*mem).To(BeEquivalentTo(0))
					Expect(*cpu).To(BeEquivalentTo(0))
				})
			})
		})

		Context("specified by deploy block", func() {
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

			When("not specified via extension", func() {
				It("returns resource limit as defined in deploy block", func() {
					mem, cpu := projectService.resourceLimits()
					Expect(*mem).To(BeEquivalentTo(1000))
					Expect(*cpu).To(BeEquivalentTo(100))
				})
			})

			When("CPU limit is specified via extension", func() {
				BeforeEach(func() {
					svcK8sConfig.Workload.Resource.MaxCPU = "0.2"
				})

				It("returns CPU limit as defined by the extension", func() {
					_, cpu := projectService.resourceLimits()
					Expect(*cpu).To(BeEquivalentTo(200))
				})
			})

			When("Memory limit is specified via extension", func() {
				BeforeEach(func() {
					svcK8sConfig.Workload.Resource.MaxMemory = "200"
				})

				It("returns Memory limit as defined by the extension", func() {
					mem, _ := projectService.resourceLimits()
					Expect(*mem).To(BeEquivalentTo(200))
				})
			})
		})
	})

	Describe("runAsUser", func() {

		Context("when defined via an extension", func() {
			runAsUser := int64(1000)

			BeforeEach(func() {
				svcK8sConfig.Workload.PodSecurity.RunAsUser = &runAsUser
			})

			It("returns the extension value", func() {
				expected := runAsUser
				Expect(projectService.runAsUser()).To(Equal(&expected))
			})
		})

		Context("when not defined via an extension", func() {
			It("returns default value", func() {
				Expect(projectService.runAsUser()).To(Equal(config.DefaultSecurityContextRunAsUser))
			})
		})
	})

	Describe("runAsGroup", func() {

		Context("when defined via an extension", func() {
			runAsGroup := int64(1000)

			BeforeEach(func() {
				svcK8sConfig.Workload.PodSecurity.RunAsGroup = &runAsGroup
			})

			It("returns the extension value", func() {
				expected := runAsGroup
				Expect(projectService.runAsGroup()).To(Equal(&expected))
			})
		})

		Context("when not defined via an extension", func() {
			It("returns default value", func() {
				Expect(projectService.runAsGroup()).To(Equal(config.DefaultSecurityContextRunAsGroup))
			})
		})
	})

	Describe("fsGroup", func() {

		Context("when defined via an extension", func() {
			fsGroup := int64(1000)

			BeforeEach(func() {
				svcK8sConfig.Workload.PodSecurity.FsGroup = &fsGroup
			})

			It("returns the extension value", func() {
				expected := fsGroup
				Expect(projectService.fsGroup()).To(Equal(&expected))
			})
		})

		Context("when not defined via an extension", func() {
			It("returns default value", func() {
				Expect(projectService.fsGroup()).To(Equal(config.DefaultSecurityContextFsGroup))
			})
		})
	})

	Describe("imagePullPolicy", func() {

		Context("when defined via extension", func() {
			policy := "Always"

			BeforeEach(func() {
				svcK8sConfig.Workload.ImagePull.Policy = policy
			})

			It("returns the extension value", func() {
				Expect(projectService.imagePullPolicy()).To(Equal(v1.PullPolicy(policy)))
			})
		})

		Context("when not defined via extension", func() {
			It("returns default value", func() {
				Expect(projectService.imagePullPolicy()).To(Equal(v1.PullPolicy(config.DefaultImagePullPolicy)))
			})
		})

		Context("for invalid image pull policy", func() {
			policy := "invalid-policy-name"
			extensions := make(map[string]interface{})

			JustBeforeEach(func() {
				svcK8sConfig.Workload.ImagePull.Policy = policy
				m, err := svcK8sConfig.ToMap()
				Expect(err).NotTo(HaveOccurred())

				extensions[config.K8SExtensionKey] = m
			})

			It("warn the user and returns default value", func() {
				_, err := NewProjectService(composego.ServiceConfig{
					Extensions: extensions,
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("SvcK8sConfig.Workload.ImagePull.Policy"))
			})
		})
	})

	Describe("imagePullSecret", func() {

		Context("when defined via extension", func() {
			secret := "image-pull-secret"

			BeforeEach(func() {
				svcK8sConfig.Workload.ImagePull.Secret = secret
			})

			It("returns extension value", func() {
				Expect(projectService.imagePullSecret()).To(Equal(secret))
			})
		})

		Context("when not defined via extension", func() {
			It("returns default value", func() {
				Expect(projectService.imagePullSecret()).To(Equal(config.DefaultImagePullSecret))
			})
		})
	})

	Describe("serviceAccountName", func() {

		Context("when defined an extension", func() {
			sa := "sa"

			BeforeEach(func() {
				svcK8sConfig.Workload.ServiceAccountName = sa
			})

			It("returns the extension value", func() {
				Expect(projectService.serviceAccountName()).To(Equal(sa))
			})
		})

		Context("when not defined via an extension", func() {
			It("returns default value", func() {
				Expect(projectService.serviceAccountName()).To(Equal(config.DefaultServiceAccountName))
			})
		})
	})

	Describe("restartPolicy", func() {

		Context("when defined via extension", func() {
			policy := config.DefaultRestartPolicy

			JustBeforeEach(func() {
				projectService.SvcK8sConfig.Workload.RestartPolicy = policy
				m, err := projectService.SvcK8sConfig.ToMap()
				Expect(err).NotTo(HaveOccurred())

				svc := projectService.ServiceConfig
				svc.Extensions = map[string]interface{}{
					config.K8SExtensionKey: m,
				}

				projectService, err = NewProjectService(svc)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the extension value", func() {
				Expect(projectService.SvcK8sConfig.Workload.RestartPolicy).To(Equal(policy))
				Expect(projectService.restartPolicy()).To(Equal(v1.RestartPolicy(policy)))
			})
		})

		Context("when not defined via extension", func() {

			Context("and defined in the project service deploy block", func() {
				policy := config.DefaultRestartPolicy.String()

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
				policy := config.RestartPolicyNever.String()

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

			Context("when value defined in the OS environment", func() {
				osVal := "BAZ"

				BeforeEach(func() {
					_ = os.Setenv(key, osVal)
				})

				AfterEach(func() {
					_ = os.Unsetenv(key)
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
				svcK8sConfig.Workload.LivenessProbe.Type = config.ProbeTypeExec.String()
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
				})
			})

			When("any of time based parameters is set to 0", func() {
				BeforeEach(func() {
					svcK8sConfig.Workload.LivenessProbe.Type = config.ProbeTypeExec.String()
					svcK8sConfig.Workload.LivenessProbe.Timeout = 0
				})

				It("logs and returns error", func() {
					_, err := projectService.LivenessProbe()
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})
	})

	Describe("livenessHTTPProbe", func() {
		When("defined via extension", func() {
			Context("with all the parameters", func() {
				BeforeEach(func() {
					svcK8sConfig.Workload.LivenessProbe.Type = config.ProbeTypeHTTP.String()
					svcK8sConfig.Workload.LivenessProbe.HTTP.Path = "/status"
					svcK8sConfig.Workload.LivenessProbe.HTTP.Port = 8080
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
					svcK8sConfig.Workload.LivenessProbe.Type = config.ProbeTypeHTTP.String()
					svcK8sConfig.Workload.LivenessProbe.HTTP.Port = 8080
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
		When("defined via extension", func() {

			Context("and supplied as string port", func() {
				port := "8080"

				BeforeEach(func() {
					svcK8sConfig.Workload.LivenessProbe.Type = config.ProbeTypeTCP.String()
					svcK8sConfig.Workload.LivenessProbe.TCP.Port = 8080
				})

				It("returns the extension value", func() {
					p, err := projectService.LivenessProbe()
					Expect(err).To(Succeed())
					Expect(p.TCPSocket.Port.String()).To(Equal(port))
				})
			})

			Context("and empty port", func() {
				BeforeEach(func() {
					svcK8sConfig.Workload.LivenessProbe.Type = config.ProbeTypeTCP.String()
					svcK8sConfig.Workload.LivenessProbe.TCP.Port = 0
				})

				It("returns an error", func() {
					p, err := projectService.LivenessProbe()
					Expect(err).NotTo(HaveOccurred())
					Expect(p.TCPSocket.Port.String()).To(Equal("0"))
				})
			})

			Context("and no port in extension", func() {
				BeforeEach(func() {
					svcK8sConfig.Workload.LivenessProbe.Type = config.ProbeTypeTCP.String()
				})

				It("returns an error", func() {
					p, err := projectService.LivenessProbe()
					Expect(err).NotTo(HaveOccurred())
					p.TCPSocket.Port = intstr.FromString("")
				})
			})
		})

	})
})
