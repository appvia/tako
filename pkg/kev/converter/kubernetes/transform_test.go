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
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/appvia/kev/pkg/kev/config"
	kmd "github.com/appvia/komando"
	composego "github.com/compose-spec/compose-go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1apps "k8s.io/api/apps/v1"
	v1batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	networking "k8s.io/api/networking/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("Transform", func() {

	var k Kubernetes
	var project composego.Project
	var projectService ProjectService
	var excluded []string
	var extensions map[string]interface{}

	BeforeEach(func() {
		project = composego.Project{
			Services: composego.Services{},
		}

		ps, err := NewProjectService(composego.ServiceConfig{
			Name:       "web",
			Image:      "some-image",
			Extensions: extensions,
		})
		Expect(err).NotTo(HaveOccurred())
		projectService = ps
	})

	JustBeforeEach(func() {
		project.Services = append(project.Services, projectService.ServiceConfig)

		k = Kubernetes{
			Opt:      ConvertOptions{},
			Project:  &project,
			Excluded: excluded,
			UI:       kmd.NoOpUI(),
		}
	})

	Describe("Transform", func() {
		When("service exclusion list is empty", func() {

			It("includes kubernetes objects for project services", func() {
				objs, err := k.Transform()
				Expect(err).NotTo(HaveOccurred())
				Expect(len(objs)).To(Equal(1))

				u, err := ToUnstructured(objs[0])
				name := u["metadata"].(map[string]interface{})["name"]

				Expect(err).NotTo(HaveOccurred())
				Expect(name).To(Equal(projectService.Name))
			})
		})

		When("excluded services are specified", func() {

			BeforeEach(func() {
				excluded = []string{projectService.Name}
			})

			It("doesn't include kubernetes objects for excluded project services", func() {
				objs, err := k.Transform()
				Expect(err).NotTo(HaveOccurred())
				Expect(len(objs)).To(Equal(0))
			})

		})
	})

	Describe("initPodSpec", func() {

		When("project service doesn't have image specified", func() {

			BeforeEach(func() {
				ps, err := NewProjectService(composego.ServiceConfig{
					Name: "web",
				})
				Expect(err).NotTo(HaveOccurred())
				projectService = ps
			})

			It("uses project service name as service image", func() {
				Expect(k.initPodSpec(projectService)).To(Equal(v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  projectService.Name,
							Image: projectService.Name,
						},
					},
					ServiceAccountName: "default",
				}))
			})
		})

		Context("with image pull secret specified via an extension", func() {
			BeforeEach(func() {
				svcK8sConfig := config.DefaultSvcK8sConfig()
				svcK8sConfig.Workload.ImagePull.Secret = "my-pp-secret"

				m, err := svcK8sConfig.Map()
				Expect(err).NotTo(HaveOccurred())

				projectService.Extensions = map[string]interface{}{
					config.K8SExtensionKey: m,
				}

				projectService, err = NewProjectService(projectService.ServiceConfig)
				Expect(err).NotTo(HaveOccurred())
			})

			It("uses passed image pull secret in the spec", func() {
				spec := k.initPodSpec(projectService)
				Expect(spec.ImagePullSecrets[0].Name).To(Equal("my-pp-secret"))
			})
		})

		Context("with service account name supplied via an extension", func() {
			BeforeEach(func() {
				svcK8sConfig := config.DefaultSvcK8sConfig()
				svcK8sConfig.Workload.ServiceAccountName = "my-service-account"

				m, err := svcK8sConfig.Map()
				Expect(err).NotTo(HaveOccurred())

				projectService.Extensions = map[string]interface{}{
					config.K8SExtensionKey: m,
				}

				projectService, err = NewProjectService(projectService.ServiceConfig)
				Expect(err).NotTo(HaveOccurred())
			})

			It("uses passed image pull policy in the spec", func() {
				spec := k.initPodSpec(projectService)
				Expect(spec.ServiceAccountName).To(Equal("my-service-account"))
			})
		})

		Context("with command specified via an extension or project service spec", func() {
			var (
				svcK8sConfig config.SvcK8sConfig
			)

			BeforeEach(func() {
				svcK8sConfig = config.DefaultSvcK8sConfig()
			})

			JustBeforeEach(func() {
				m, err := svcK8sConfig.Map()
				Expect(err).NotTo(HaveOccurred())

				projectService.Extensions = map[string]interface{}{
					config.K8SExtensionKey: m,
				}

				projectService, err = NewProjectService(projectService.ServiceConfig)
				Expect(err).NotTo(HaveOccurred())
			})

			When("command specified via a config extension", func() {
				BeforeEach(func() {
					svcK8sConfig.Workload.Command = []string{"/bin/bash", "-c", "sleep 1"}
				})

				It("uses command as specified in config extension", func() {
					spec := k.initPodSpec(projectService)
					Expect(spec.Containers[0].Command).To(Equal([]string{"/bin/bash", "-c", "sleep 1"}))
				})
			})

			When("command specified via project service spec", func() {
				BeforeEach(func() {
					projectService.Entrypoint = []string{"/default/command"}
				})

				It("uses command as specified in project service spec", func() {
					spec := k.initPodSpec(projectService)
					Expect(spec.Containers[0].Command).To(Equal([]string{"/default/command"}))
				})
			})

			When("command not specified in config extension nor in project service spec", func() {
				It("doesn't set up container command", func() {
					spec := k.initPodSpec(projectService)
					Expect(spec.Containers[0].Command).To(BeNil())
				})
			})
		})

		Context("with command arguments specified via an extension", func() {
			var (
				svcK8sConfig config.SvcK8sConfig
			)

			BeforeEach(func() {
				svcK8sConfig = config.DefaultSvcK8sConfig()
			})

			JustBeforeEach(func() {
				m, err := svcK8sConfig.Map()
				Expect(err).NotTo(HaveOccurred())

				projectService.Extensions = map[string]interface{}{
					config.K8SExtensionKey: m,
				}

				projectService, err = NewProjectService(projectService.ServiceConfig)
				Expect(err).NotTo(HaveOccurred())
			})

			When("command arguments specified via a config extension", func() {
				BeforeEach(func() {
					svcK8sConfig.Workload.CommandArgs = []string{"-c", "sleep 1"}
				})

				It("uses command args as specified via a config extension", func() {
					spec := k.initPodSpec(projectService)
					Expect(spec.Containers[0].Args).To(Equal([]string{"-c", "sleep 1"}))
				})
			})

			When("command arguments not specified via a config extension", func() {
				It("doesn't set up container command arguments", func() {
					spec := k.initPodSpec(projectService)
					Expect(spec.Containers[0].Args).To(BeNil())
				})
			})
		})

		It("generates pod spec as expected", func() {
			Expect(k.initPodSpec(projectService)).To(Equal(v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  projectService.Name,
						Image: projectService.Image,
					},
				},
				ServiceAccountName: "default",
			}))
		})

	})

	Describe("initPodSpecWithConfigMap", func() {

		When("project service references config(s)", func() {

			configName := "project-config-name"
			subPath := "path"
			mountPath := filepath.Join("/mount", subPath)

			BeforeEach(func() {
				project.Configs = composego.Configs{
					configName: composego.ConfigObjConfig{},
				}

				projectService.Configs = []composego.ServiceConfigObjConfig{
					{
						Source: configName,
						Target: mountPath,
					},
				}
			})

			It("initiates Pod spec with volumes mounting config maps", func() {
				spec := k.initPodSpecWithConfigMap(projectService)
				Expect(spec.Volumes).To(HaveLen(1))

				vol := spec.Volumes[0]
				Expect(vol.Name).To(Equal(configName))

				volumeMount := spec.Containers[0].VolumeMounts[0]
				Expect(volumeMount.Name).To(Equal(configName))
				Expect(volumeMount.MountPath).To(Equal(mountPath))
				Expect(volumeMount.SubPath).To(Equal(subPath))
			})

			Context("and config metadata is not specified in the project", func() {
				BeforeEach(func() {
					project.Configs = composego.Configs{}

					projectService.Configs = []composego.ServiceConfigObjConfig{
						{
							Source: configName,
							Target: mountPath,
						},
					}
				})

				It("ignores the project service config reference", func() {
					spec := k.initPodSpecWithConfigMap(projectService)
					Expect(spec.Volumes).To(HaveLen(0))
					Expect(spec.Containers[0].VolumeMounts).To(HaveLen(0))
				})
			})

			Context("and the config metadata points at external config", func() {
				BeforeEach(func() {
					project.Configs = composego.Configs{
						configName: composego.ConfigObjConfig{
							External: composego.External{
								External: true,
							},
						},
					}
				})

				It("ignores the project service external config reference", func() {
					spec := k.initPodSpecWithConfigMap(projectService)
					Expect(spec.Volumes).To(HaveLen(0))
					Expect(spec.Containers[0].VolumeMounts).To(HaveLen(0))
				})
			})
		})

		When("project service doesn't reference config", func() {
			It("returns Pod spec without volumes and volume mounts", func() {
				Expect(k.initPodSpecWithConfigMap(projectService)).To(Equal(v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  projectService.Name,
							Image: projectService.Image,
						},
					},
					ServiceAccountName: "default",
				}))
			})
		})
	})

	Describe("initSvc", func() {
		It("generates kubernetes service spec as expected", func() {
			Expect(k.initSvc(projectService)).To(Equal(&v1.Service{
				TypeMeta: meta.TypeMeta{
					Kind:       "Service",
					APIVersion: "v1",
				},
				ObjectMeta: meta.ObjectMeta{
					Name:   projectService.Name,
					Labels: configLabels(projectService.Name),
				},
				Spec: v1.ServiceSpec{
					Selector: configLabels(projectService.Name),
				},
			}))
		})

		When("project service name is longer than 63 characters", func() {
			BeforeEach(func() {
				projectService.Name = strings.Repeat("a", 100)
			})

			It("trims down service name to max 63 chars as per DNS extension standard RFC-1123", func() {
				Expect(k.initSvc(projectService).Name).To(HaveLen(63))
			})
		})
	})

	Describe("initConfigMapFromFileOrDir", func() {

		Context("with single file", func() {
			configMapName := "my_config_map"
			filePath := "../../testdata/converter/kubernetes/configmaps/config-a"

			Context("with file path matching one of project defined configs", func() {
				BeforeEach(func() {
					project.Configs = composego.Configs{
						"config-name": composego.ConfigObjConfig(
							composego.FileObjectConfig{
								Name: "project-config-name",
								File: filePath,
							},
						),
					}
				})

				It("returns config map with data taken from that single file", func() {
					cm, err := k.initConfigMapFromFileOrDir(projectService, configMapName, filePath)
					Expect(err).ToNot(HaveOccurred())
					Expect(cm.Name).To(Equal(rfc1123dns(configMapName)))
					Expect(cm.Data).To(HaveLen(1))
					Expect(cm.Data).To(HaveKey("config-a"))
					Expect(cm.Data).ToNot(HaveKey("config-b"))
					Expect(cm.Annotations).To(HaveKeyWithValue("use-subpath", "true"))
				})
			})

			Context("with file path not matching any of project defined configs", func() {
				It("returns an error", func() {
					_, err := k.initConfigMapFromFileOrDir(projectService, configMapName, filePath)
					Expect(err).To(HaveOccurred())
				})
			})

		})

		Context("with directory of files", func() {
			configMapName := "my_config_map"
			dir := "../../testdata/converter/kubernetes/configmaps/"

			It("returns config map with all files in that directory with data keyed by individual file name", func() {
				cm, err := k.initConfigMapFromFileOrDir(projectService, configMapName, dir)
				Expect(err).ToNot(HaveOccurred())
				Expect(cm.Name).To(Equal(rfc1123dns(configMapName)))
				Expect(cm.Data).To(HaveLen(2))
				Expect(cm.Data).To(HaveKey("config-a"))
				Expect(cm.Data).To(HaveKey("config-b"))
			})
		})
	})

	Describe("initConfigMap", func() {

		configMapName := "myConfig"
		data := map[string]string{
			"foo": "bar",
		}

		It("initialises a new ConfigMap", func() {
			Expect(k.initConfigMap(projectService, configMapName, data)).To(Equal(&v1.ConfigMap{
				TypeMeta: meta.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: meta.ObjectMeta{
					Name:   rfc1123dns(configMapName),
					Labels: configLabels(projectService.Name),
				},
				Data: data,
			}))
		})
	})

	Describe("initConfigMapFromFile", func() {

		Context("with invalid file path", func() {
			filePath := "/invalid/file/path"

			It("returns an error", func() {
				_, err := k.initConfigMapFromFile(projectService, filePath)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("for config file path not matching any of project defined configs", func() {
			filePath := "../../testdata/converter/kubernetes/configmaps/config-a"

			BeforeEach(func() {
				// explicitly reset for visibility
				project.Configs = composego.Configs{}
			})

			It("returns an error", func() {
				_, err := k.initConfigMapFromFile(projectService, filePath)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("No config found matching the file name"))
			})
		})

		// Other cases covered by initConfigMapFromFileOrDir
	})

	Describe("initConfigMapFromDir", func() {
		configMapName := "myConfig"

		Context("with invalid directory", func() {
			dir := "/invalid/directory"

			It("returns an error", func() {
				_, err := k.initConfigMapFromDir(projectService, configMapName, dir)
				Expect(err).To(HaveOccurred())
			})
		})

		// Other cases covered by initConfigMapFromFileOrDir
	})

	Describe("initDeployment", func() {
		var expectedPodSpec v1.PodSpec
		var expectedDeployment *v1apps.Deployment

		replicas := int32(1)

		JustBeforeEach(func() {
			expectedDeployment = &v1apps.Deployment{
				TypeMeta: meta.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				ObjectMeta: meta.ObjectMeta{
					Name:   projectService.Name,
					Labels: configAllLabels(projectService),
				},
				Spec: v1apps.DeploymentSpec{
					Replicas: &replicas,
					Selector: &meta.LabelSelector{
						MatchLabels: configLabels(projectService.Name),
					},
					Strategy: v1apps.DeploymentStrategy{
						Type: "RollingUpdate",
						RollingUpdate: &v1apps.RollingUpdateDeployment{
							MaxSurge:       &intstr.IntOrString{Type: 0, IntVal: 1, StrVal: ""},
							MaxUnavailable: &intstr.IntOrString{Type: 1, IntVal: 0, StrVal: "25%"},
						},
					},
					Template: v1.PodTemplateSpec{
						ObjectMeta: meta.ObjectMeta{
							Annotations: configAnnotations(projectService.Labels),
							Labels:      configLabels(projectService.Name),
						},
						Spec: expectedPodSpec,
					},
				},
			}
		})

		Context("for project service without configs", func() {
			BeforeEach(func() {
				expectedPodSpec = k.initPodSpec(projectService)
			})

			It("generates kubernetes deployment spec as expected", func() {
				d := k.initDeployment(projectService)
				Expect(d).To(Equal(expectedDeployment))

				podContainerVolumeMounts := d.Spec.Template.Spec.Containers[0].VolumeMounts
				Expect(podContainerVolumeMounts).To(HaveLen(0))
			})
		})

		Context("for project service with configs", func() {
			var (
				configName string
				mountPath  string
			)

			BeforeEach(func() {
				configName = "config"
				mountPath = "/mount/path"

				project.Configs = composego.Configs{
					configName: composego.ConfigObjConfig{
						File: "/path/to/config/file",
					},
				}

				projectService.Configs = []composego.ServiceConfigObjConfig{
					{
						Source: configName,
						Target: mountPath,
					},
				}

				expectedPodSpec = k.initPodSpecWithConfigMap(projectService)
			})

			It("generates kubernetes deployment spec as expected", func() {
				d := k.initDeployment(projectService)
				Expect(d).To(Equal(expectedDeployment))

				podContainerVolumeMounts := d.Spec.Template.Spec.Containers[0].VolumeMounts
				Expect(podContainerVolumeMounts).To(HaveLen(1))
				Expect(podContainerVolumeMounts[0].Name).To(Equal(configName))
				Expect(podContainerVolumeMounts[0].MountPath).To(Equal(mountPath))
			})
		})

		When("update strategy is defined in project service deploy block", func() {
			BeforeEach(func() {
				parallelism := uint64(2)
				projectService.Deploy = &composego.DeployConfig{
					UpdateConfig: &composego.UpdateConfig{
						Parallelism: &parallelism,
						Order:       "start-first",
					},
				}
				svcK8sConfig, err := config.SvcK8sConfigFromCompose(&projectService.ServiceConfig)
				Expect(err).ToNot(HaveOccurred())
				projectService.SvcK8sConfig = svcK8sConfig
			})

			It("it includes update strategy in the deployment spec", func() {
				d := k.initDeployment(projectService)
				Expect(d.Spec.Strategy.RollingUpdate.MaxSurge.IntValue()).To(Equal(2))
				Expect(d.Spec.Strategy.RollingUpdate.MaxUnavailable.IntValue()).To(Equal(0))
			})
		})

		Context("for project service configured with annotations", func() {
			BeforeEach(func() {
				svcK8sConfig := config.DefaultSvcK8sConfig()
				svcK8sConfig.Workload.Annotations = map[string]string{"key1": "value1"}
				ext, err := svcK8sConfig.Map()
				Expect(err).NotTo(HaveOccurred())

				projectService.Extensions = map[string]interface{}{config.K8SExtensionKey: ext}
				projectService, err = NewProjectService(projectService.ServiceConfig)
				Expect(err).NotTo(HaveOccurred())
			})

			It("generates annotations directly on the pod spec", func() {
				d := k.initDeployment(projectService)
				Expect(d.Spec.Template.Annotations).To(HaveLen(1))
				Expect(d.Spec.Template.Annotations).To(HaveKeyWithValue("key1", "value1"))
			})

			It("does not generate any annotations on the Deployment metadata object", func() {
				d := k.initDeployment(projectService)
				Expect(d.ObjectMeta.Annotations).To(HaveLen(0))
			})
		})
	})

	Describe("initDaemonSet", func() {

		It("initialises DaemonSet as expected", func() {
			Expect(k.initDaemonSet(projectService)).To(Equal(&v1apps.DaemonSet{
				TypeMeta: meta.TypeMeta{
					Kind:       "DaemonSet",
					APIVersion: "apps/v1",
				},
				ObjectMeta: meta.ObjectMeta{
					Name:   projectService.Name,
					Labels: configAllLabels(projectService),
				},
				Spec: v1apps.DaemonSetSpec{
					Template: v1.PodTemplateSpec{
						Spec: k.initPodSpec(projectService),
					},
				},
			}))
		})
	})

	Describe("initStatefulSet", func() {
		var expectedPodSpec v1.PodSpec
		var expectedSts *v1apps.StatefulSet

		replicas := int32(1)

		JustBeforeEach(func() {
			expectedSts = &v1apps.StatefulSet{
				TypeMeta: meta.TypeMeta{
					Kind:       "StatefulSet",
					APIVersion: "apps/v1",
				},
				ObjectMeta: meta.ObjectMeta{
					Name:   projectService.Name,
					Labels: configAllLabels(projectService),
				},
				Spec: v1apps.StatefulSetSpec{
					Replicas: &replicas,
					Selector: &meta.LabelSelector{
						MatchLabels: configLabels(projectService.Name),
					},
					Template: v1.PodTemplateSpec{
						ObjectMeta: meta.ObjectMeta{
							Annotations: configAnnotations(projectService.Labels, projectService.podAnnotations()),
							Labels:      configLabels(projectService.Name), // added
						},
						Spec: expectedPodSpec,
					},
					ServiceName: projectService.Name, // added
					UpdateStrategy: v1apps.StatefulSetUpdateStrategy{ // added
						Type:          v1apps.RollingUpdateStatefulSetStrategyType,
						RollingUpdate: &v1apps.RollingUpdateStatefulSetStrategy{},
					},
				},
			}
		})

		Context("for project service without configs", func() {
			BeforeEach(func() {
				expectedPodSpec = k.initPodSpec(projectService)
			})

			It("generates kubernetes deployment spec as expected", func() {
				d := k.initStatefulSet(projectService)
				Expect(d).To(Equal(expectedSts))

				podContainerVolumeMounts := d.Spec.Template.Spec.Containers[0].VolumeMounts
				Expect(podContainerVolumeMounts).To(HaveLen(0))
			})
		})

		Context("for project service with configs", func() {
			var (
				configName string
				mountPath  string
			)

			BeforeEach(func() {
				configName = "config"
				mountPath = "/mount/path"

				project.Configs = composego.Configs{
					configName: composego.ConfigObjConfig{
						File: "/path/to/config/file",
					},
				}

				projectService.Configs = []composego.ServiceConfigObjConfig{
					{
						Source: configName,
						Target: mountPath,
					},
				}

				expectedPodSpec = k.initPodSpecWithConfigMap(projectService)
			})

			It("generates kubernetes StatefulSet spec as expected", func() {
				d := k.initStatefulSet(projectService)
				Expect(d).To(Equal(expectedSts))

				podContainerVolumeMounts := d.Spec.Template.Spec.Containers[0].VolumeMounts
				Expect(podContainerVolumeMounts).To(HaveLen(1))
				Expect(podContainerVolumeMounts[0].Name).To(Equal(configName))
				Expect(podContainerVolumeMounts[0].MountPath).To(Equal(mountPath))
			})
		})

		Context("for project service configured with annotations", func() {
			BeforeEach(func() {
				svcK8sConfig := config.DefaultSvcK8sConfig()
				svcK8sConfig.Workload.Annotations = map[string]string{"key1": "value1"}
				ext, err := svcK8sConfig.Map()
				Expect(err).NotTo(HaveOccurred())

				projectService.Extensions = map[string]interface{}{config.K8SExtensionKey: ext}
				projectService, err = NewProjectService(projectService.ServiceConfig)
				Expect(err).NotTo(HaveOccurred())
			})

			It("generates annotations directly on the pod spec", func() {
				d := k.initStatefulSet(projectService)
				Expect(d.Spec.Template.Annotations).To(HaveLen(1))
				Expect(d.Spec.Template.Annotations).To(HaveKeyWithValue("key1", "value1"))
			})

			It("does not generate any annotations on the StatefulSet metadata object", func() {
				d := k.initStatefulSet(projectService)
				Expect(d.ObjectMeta.Annotations).To(HaveLen(0))
			})
		})
	})

	Describe("initJob", func() {
		var expectedPodSpec v1.PodSpec
		var expectedJob *v1batch.Job

		replicas := 1
		expectedParallelism := int32(replicas)
		expectedCompletions := int32(replicas)

		JustBeforeEach(func() {
			expectedJob = &v1batch.Job{
				TypeMeta: meta.TypeMeta{
					Kind:       "Job",
					APIVersion: "batch/v1",
				},
				ObjectMeta: meta.ObjectMeta{
					Name:   projectService.Name,
					Labels: configAllLabels(projectService),
				},
				Spec: v1batch.JobSpec{
					Parallelism: &expectedParallelism,
					Completions: &expectedCompletions,
					Selector: &meta.LabelSelector{
						MatchLabels: configLabels(projectService.Name),
					},
					Template: v1.PodTemplateSpec{
						ObjectMeta: meta.ObjectMeta{
							Annotations: configAnnotations(projectService.Labels),
							Labels:      configLabels(projectService.Name),
						},
						Spec: expectedPodSpec,
					},
				},
			}
		})

		Context("for project service without configs", func() {
			BeforeEach(func() {
				expectedPodSpec = k.initPodSpec(projectService)
			})

			It("generates kubernetes deployment spec as expected", func() {
				d := k.initJob(projectService, replicas)
				Expect(d).To(Equal(expectedJob))

				podContainerVolumeMounts := d.Spec.Template.Spec.Containers[0].VolumeMounts
				Expect(podContainerVolumeMounts).To(HaveLen(0))
			})
		})

		Context("for project service with configs", func() {
			var (
				configName string
				mountPath  string
			)

			BeforeEach(func() {
				configName = "config"
				mountPath = "/mount/path"

				project.Configs = composego.Configs{
					configName: composego.ConfigObjConfig{
						File: "/path/to/config/file",
					},
				}

				projectService.Configs = []composego.ServiceConfigObjConfig{
					{
						Source: configName,
						Target: mountPath,
					},
				}

				expectedPodSpec = k.initPodSpecWithConfigMap(projectService)
			})

			It("generates kubernetes StatefulSet spec as expected", func() {
				d := k.initJob(projectService, replicas)
				Expect(d).To(Equal(expectedJob))

				podContainerVolumeMounts := d.Spec.Template.Spec.Containers[0].VolumeMounts
				Expect(podContainerVolumeMounts).To(HaveLen(1))
				Expect(podContainerVolumeMounts[0].Name).To(Equal(configName))
				Expect(podContainerVolumeMounts[0].MountPath).To(Equal(mountPath))
			})
		})

		Context("for project service configured with annotations", func() {
			BeforeEach(func() {
				svcK8sConfig := config.DefaultSvcK8sConfig()
				svcK8sConfig.Workload.Annotations = map[string]string{"key1": "value1"}
				ext, err := svcK8sConfig.Map()
				Expect(err).NotTo(HaveOccurred())

				projectService.Extensions = map[string]interface{}{config.K8SExtensionKey: ext}
				projectService, err = NewProjectService(projectService.ServiceConfig)
				Expect(err).NotTo(HaveOccurred())
			})

			It("generates annotations directly on the pod spec", func() {
				d := k.initJob(projectService, replicas)
				Expect(d.Spec.Template.Annotations).To(HaveLen(1))
				Expect(d.Spec.Template.Annotations).To(HaveKeyWithValue("key1", "value1"))
			})

			It("does not generate any annotations on the Job metadata object", func() {
				d := k.initJob(projectService, replicas)
				Expect(d.ObjectMeta.Annotations).To(HaveLen(0))
			})
		})
	})

	Describe("initIngress", func() {
		port := int32(1234)

		When("project service extension exposing the k8s service using an empty string", func() {
			BeforeEach(func() {
				projectService.SvcK8sConfig.Service.Expose.Domain = ""
			})

			It("doesn't initiate an ingress", func() {
				Expect(k.initIngress(projectService, port)).To(BeNil())
			})
		})

		When("project service extension exposing the k8s service", func() {
			domain := "domain.name"
			ingressAnnotations := map[string]string{
				"kubernetes.io/ingress.class":    "external",
				"cert-manager.io/cluster-issuer": "prod-le-dns01",
			}

			BeforeEach(func() {
				projectService.SvcK8sConfig.Service.Expose.Domain = domain
				projectService.SvcK8sConfig.Service.Expose.IngressAnnotations = ingressAnnotations
			})

			It("initialises Ingress with a port routing to the project service name", func() {
				ing := k.initIngress(projectService, port)

				Expect(ing).To(Equal(&networkingv1.Ingress{
					TypeMeta: meta.TypeMeta{
						Kind:       "Ingress",
						APIVersion: "networking.k8s.io/v1",
					},
					ObjectMeta: meta.ObjectMeta{
						Name:        projectService.Name,
						Labels:      configLabels(projectService.Name),
						Annotations: ingressAnnotations,
					},
					Spec: networkingv1.IngressSpec{
						Rules: []networkingv1.IngressRule{
							{
								Host: domain,
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{
											{
												Path: "",
												Backend: networkingv1.IngressBackend{
													Service: &networkingv1.IngressServiceBackend{
														Name: projectService.Name,
														Port: networkingv1.ServiceBackendPort{
															Number: port,
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				}))
			})
		})

		When("project service extension exposing the k8s service using a domain name", func() {
			BeforeEach(func() {
				projectService.SvcK8sConfig.Service.Expose.Domain = "domain.name"
			})

			It("initialises Ingress with the correct service", func() {
				ingress := k.initIngress(projectService, port)
				configuredService := ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend.Service.Name
				Expect(configuredService).To(Equal(projectService.Name))
			})

			It("initialises Ingress with the correct port", func() {
				ingress := k.initIngress(projectService, port)
				configuredPort := ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend.Service.Port.Number
				Expect(configuredPort).To(Equal(port))
			})
		})

		When("project service extension exposing the k8s service using a domain with a path", func() {
			domain := "domain.name"
			path := "path"

			BeforeEach(func() {
				projectService.SvcK8sConfig.Service.Expose.Domain = filepath.Join(domain, path)
			})

			It("specifies host in the initialised Ingress", func() {
				ingress := k.initIngress(projectService, port)
				Expect(ingress.Spec.Rules[0].Host).To(Equal(domain))
			})

			It("specifies path in the initialised Ingress", func() {
				ingress := k.initIngress(projectService, port)
				ingressPath := ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Path
				Expect(ingressPath).To(Equal("/" + path))
			})
		})

		When("project service extension exposing the k8s service using a comma separated list of domain names", func() {
			domains := []string{
				"domain.name",
				"another.domain.name",
			}

			BeforeEach(func() {
				projectService.SvcK8sConfig.Service.Expose.Domain = strings.Join(domains, ",")
			})

			It("specifies all comma separated hosts in the initialised Ingress", func() {
				ingress := k.initIngress(projectService, port)
				Expect(ingress.Spec.Rules[0].Host).To(Equal(domains[0]))
				Expect(ingress.Spec.Rules[1].Host).To(Equal(domains[1]))
			})
		})

		When("project service extension exposing the k8s service using a default ingress backend", func() {
			BeforeEach(func() {
				projectService.SvcK8sConfig.Service.Expose.Domain = DefaultIngressBackendKeyword
			})

			It("creates a default backend in the initialised Ingress with no rules`", func() {
				ingress := k.initIngress(projectService, port)
				Expect(ingress.Spec.DefaultBackend.Service.Name).To(Equal(projectService.Name))
				Expect(ingress.Spec.DefaultBackend.Service.Port.Number).To(Equal(port))
				Expect(ingress.Spec.Rules).To(HaveLen(0))
			})
		})

		When("project service extension instructing to expose the k8s service with domain and ingress annotations", func() {
			ingressAnnotations := map[string]string{
				"kubernetes.io/ingress.class":    "external",
				"cert-manager.io/cluster-issuer": "prod-le-dns01",
			}

			BeforeEach(func() {
				projectService.SvcK8sConfig.Service.Expose.IngressAnnotations = ingressAnnotations
				projectService.SvcK8sConfig.Service.Expose.Domain = "domain.name"
			})

			It("initialises Ingress with configured ingress annotations", func() {
				ingress := k.initIngress(projectService, port)
				Expect(ingress.ObjectMeta.Annotations).To(Equal(ingressAnnotations))
			})
		})

		When("TLS secret name was specified via extension", func() {
			BeforeEach(func() {
				projectService.SvcK8sConfig.Service.Expose.Domain = "domain.name"
				projectService.SvcK8sConfig.Service.Expose.TlsSecret = "my-tls-secret"
			})

			It("will include it in the ingress spec", func() {
				ing := k.initIngress(projectService, port)

				Expect(ing.Spec.TLS).To(Equal([]networkingv1.IngressTLS{
					{
						Hosts:      []string{"domain.name"},
						SecretName: "my-tls-secret",
					},
				}))
			})
		})

		When("TLS secret name was specified via extension for service exposed with default ingress backend", func() {
			BeforeEach(func() {
				projectService.SvcK8sConfig.Service.Expose.Domain = DefaultIngressBackendKeyword
				projectService.SvcK8sConfig.Service.Expose.TlsSecret = "my-tls-secret"
			})

			It("does not create a TLS object in the ingress spec", func() {
				ing := k.initIngress(projectService, port)
				Expect(ing.Spec.TLS).To(HaveLen(0))
			})
		})
	})

	Describe("initHpa", func() {
		var obj runtime.Object

		Context("with supported object kind", func() {
			BeforeEach(func() {
				obj = &v1apps.Deployment{
					TypeMeta: meta.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
					},
				}
			})

			Context("with autoscaling options specified", func() {

				When("the maximum number of replicas is defined", func() {
					BeforeEach(func() {
						projectService.SvcK8sConfig.Workload.Autoscale.MaxReplicas = 10
					})

					It("initialises HPA with expected API version referencing passed object", func() {
						hpa := k.initHpa(projectService, obj)
						Expect(hpa.APIVersion).To(Equal("autoscaling/v2beta2"))
						Expect(hpa.Spec.ScaleTargetRef.Kind).To(Equal("Deployment"))
						Expect(hpa.Spec.ScaleTargetRef.APIVersion).To(Equal("apps/v1"))
						Expect(hpa.Spec.ScaleTargetRef.Name).To(Equal(projectService.Name))
					})

					When("workload CPU threshold parameters is also specified", func() {
						BeforeEach(func() {
							projectService.SvcK8sConfig.Workload.Autoscale.MaxReplicas = 10
							projectService.SvcK8sConfig.Workload.Autoscale.CPUThreshold = 65
						})

						It("initialises Horizontal Pod Autoscaler for a project service", func() {
							hpa := k.initHpa(projectService, obj)
							Expect(hpa.Spec.MaxReplicas).To(BeEquivalentTo(10))
							// first metrics is CPU
							Expect(hpa.Spec.Metrics[0].Resource.Name).To(BeEquivalentTo("cpu"))
							Expect(hpa.Spec.Metrics[0].Resource.Target.Type).To(BeEquivalentTo("Utilization"))
							Expect(*hpa.Spec.Metrics[0].Resource.Target.AverageUtilization).To(BeEquivalentTo(65))
						})
					})

					When("workload CPU threshold is not specified", func() {

						BeforeEach(func() {
							projectService.SvcK8sConfig.Workload.Autoscale.MaxReplicas = 10
						})

						It("initialises Horizontal Pod Autoscaler for a project service with default target CPU utilization of 70%", func() {
							hpa := k.initHpa(projectService, obj)
							Expect(hpa.Spec.MaxReplicas).To(BeEquivalentTo(10))
							// first metrics is CPU
							Expect(hpa.Spec.Metrics[0].Resource.Name).To(BeEquivalentTo("cpu"))
							Expect(hpa.Spec.Metrics[0].Resource.Target.Type).To(BeEquivalentTo("Utilization"))
							Expect(*hpa.Spec.Metrics[0].Resource.Target.AverageUtilization).To(BeEquivalentTo(70))
						})
					})

					When("autoscaling max replicas number is lower or equal to initial number of replicas", func() {
						BeforeEach(func() {
							projectService.SvcK8sConfig.Workload.Replicas = 10
							projectService.SvcK8sConfig.Workload.Autoscale.MaxReplicas = 5
						})

						It("doesn't initialise the Horizontal Pod Autoscaler", func() {
							hpa := k.initHpa(projectService, obj)
							Expect(hpa).To(BeNil())
						})
					})

					When("the maximum number of replicas is specified as 0", func() {
						BeforeEach(func() {
							projectService.SvcK8sConfig.Workload.Autoscale.MaxReplicas = 0
						})

						It("doesn't initialize Horizontal Pod Autoscaler for that project service", func() {
							hpa := k.initHpa(projectService, obj)
							Expect(hpa).To(BeNil())
						})
					})

					When("workload Memory threshold parameter is also specified", func() {
						BeforeEach(func() {
							projectService.SvcK8sConfig.Workload.Autoscale.MaxReplicas = 10
							projectService.SvcK8sConfig.Workload.Autoscale.MemoryThreshold = 40
						})

						It("initialises Horizontal Pod Autoscaler for a project service", func() {
							hpa := k.initHpa(projectService, obj)
							Expect(hpa.Spec.MaxReplicas).To(BeEquivalentTo(10))
							// second metric is Memory
							Expect(hpa.Spec.Metrics[1].Resource.Name).To(BeEquivalentTo("memory"))
							Expect(hpa.Spec.Metrics[1].Resource.Target.Type).To(BeEquivalentTo("Utilization"))
							Expect(*hpa.Spec.Metrics[1].Resource.Target.AverageUtilization).To(BeEquivalentTo(40))
						})
					})

					When("workload Memory threshold is not specified", func() {

						BeforeEach(func() {
							projectService.SvcK8sConfig.Workload.Autoscale.MaxReplicas = 10
						})

						It("initialises Horizontal Pod Autoscaler for a project service with default target Memory utilization of 70%", func() {
							hpa := k.initHpa(projectService, obj)
							Expect(hpa.Spec.MaxReplicas).To(BeEquivalentTo(10))
							// second metric is Memory
							Expect(hpa.Spec.Metrics[1].Resource.Name).To(BeEquivalentTo("memory"))
							Expect(hpa.Spec.Metrics[1].Resource.Target.Type).To(BeEquivalentTo("Utilization"))
							Expect(*hpa.Spec.Metrics[1].Resource.Target.AverageUtilization).To(BeEquivalentTo(70))
						})
					})
				})

				When("the maximum number of replicas is not defined", func() {
					It("doesn't initialize Horizontal Pod Autoscaler for that project service", func() {
						hpa := k.initHpa(projectService, obj)
						Expect(hpa).To(BeNil())
					})
				})

			})
		})

		Context("with not supported object kind", func() {
			BeforeEach(func() {
				obj = &v1apps.StatefulSet{
					TypeMeta: meta.TypeMeta{
						Kind:       "StatefulSet",
						APIVersion: "apps/v1",
					},
				}
			})

			It("doesn't initialize Horizontal Pod Autoscaler for that project service", func() {
				hpa := k.initHpa(projectService, obj)
				Expect(hpa).To(BeNil())
			})
		})

	})

	Describe("initSa", func() {
		When("service account name is specified as empty string in the workload configuration", func() {
			BeforeEach(func() {
				projectService.SvcK8sConfig.Workload.ServiceAccountName = ""
			})

			It("doesn't initialize ServiceAccount for that project service", func() {
				sa := k.initServiceAccount(projectService)
				Expect(sa).To(BeNil())
			})
		})

		When("service account name is defined as `default`", func() {
			BeforeEach(func() {
				projectService.SvcK8sConfig.Workload.ServiceAccountName = "default"
			})

			It("doesn't initialize ServiceAccount for that project service", func() {
				sa := k.initServiceAccount(projectService)
				Expect(sa).To(BeNil())
			})
		})

		When("service account name is specified with name different than `default`", func() {
			BeforeEach(func() {
				projectService.SvcK8sConfig.Workload.ServiceAccountName = "mysvcacc"
			})

			It("initializes ServiceAccount for the project service", func() {
				sa := k.initServiceAccount(projectService)
				Expect(sa).ToNot(BeNil())

				automountSAToken := false

				expected := &v1.ServiceAccount{
					TypeMeta: meta.TypeMeta{
						Kind:       "ServiceAccount",
						APIVersion: "v1",
					},
					ObjectMeta: meta.ObjectMeta{
						Name:        "mysvcacc",
						Labels:      configLabels(projectService.Name),
						Annotations: configAnnotations(projectService.Labels),
					},
					AutomountServiceAccountToken: &automountSAToken,
				}

				Expect(sa).To(Equal(expected))
			})
		})
	})

	Describe("createSecrets", func() {
		secretName := "my-secret"
		var secretConfig composego.SecretConfig

		JustBeforeEach(func() {
			project.Secrets = composego.Secrets{
				secretName: secretConfig,
			}
		})

		Context("for external secrets", func() {
			BeforeEach(func() {
				secretConfig = composego.SecretConfig(
					composego.FileObjectConfig{
						External: composego.External{
							External: true,
						},
					},
				)
			})

			It("logs a warning and doesn't create a secret", func() {
				s, err := k.createSecrets()
				Expect(err).ToNot(HaveOccurred())
				Expect(s).To(HaveLen(0))

				assertLog(logrus.WarnLevel,
					"https://kubernetes.io/docs/tasks/inject-data-application/distribute-credentials-secure/",
					map[string]string{})
			})
		})

		Context("for secrets referencing local file", func() {

			When("file exists", func() {
				BeforeEach(func() {
					secretConfig = composego.SecretConfig(
						composego.FileObjectConfig{
							File: "../../testdata/converter/kubernetes/secrets/secret_file",
						},
					)
				})

				It("returns a slice of secret objects", func() {
					expected := []*v1.Secret{
						{
							TypeMeta: meta.TypeMeta{
								Kind:       "Secret",
								APIVersion: "v1",
							},
							ObjectMeta: meta.ObjectMeta{
								Name:   secretName,
								Labels: configLabels(secretName),
							},
							Type: v1.SecretTypeOpaque,
							Data: map[string][]byte{
								secretName: {109, 121, 32, 115, 101, 99, 114, 101, 116, 32, 100, 97, 116, 97, 10},
							},
						},
					}

					Expect(k.createSecrets()).To(Equal(expected))
				})
			})

			When("file doesn't exist", func() {
				filePath := "wrong/path"

				BeforeEach(func() {
					secretConfig = composego.SecretConfig(
						composego.FileObjectConfig{
							File: filePath,
						},
					)
				})

				It("returns an error", func() {
					s, err := k.createSecrets()
					Expect(err).To(HaveOccurred())
					Expect(s).To(BeNil())
					Expect(err).To(MatchError(fmt.Sprintf("open %s: no such file or directory", filePath)))
				})
			})
		})
	})

	Describe("createPVC", func() {

		Context("with unspecified or wrong volume size", func() {
			volume := Volumes{
				VolumeName: "some-name",
				PVCSize:    "invalid-amount",
			}

			It("returns an error", func() {
				_, err := k.createPVC(volume)
				Expect(err).To(HaveOccurred())
			})
		})

		When("size is provided", func() {
			pvcSize := "100Mi"

			volume := Volumes{
				VolumeName: "some-name",
				PVCSize:    pvcSize,
			}

			expectedQuantity, _ := resource.ParseQuantity(pvcSize)

			It("creates a PVC object", func() {
				Expect(k.createPVC(volume)).To(Equal(&v1.PersistentVolumeClaim{
					TypeMeta: meta.TypeMeta{
						Kind:       "PersistentVolumeClaim",
						APIVersion: "v1",
					},
					ObjectMeta: meta.ObjectMeta{
						Name:   volume.VolumeName,
						Labels: configLabels(volume.VolumeName),
					},
					Spec: v1.PersistentVolumeClaimSpec{
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: expectedQuantity,
							},
						},
						AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
					},
				}))
			})
		})

		When("volume mode is set to read only", func() {
			volume := Volumes{
				VolumeName: "some-name",
				PVCSize:    "10Gi",
				Mode:       "ro",
			}

			It("sets correct access mode", func() {
				var spec v1.PersistentVolumeClaimSpec

				pvc, err := k.createPVC(volume)
				if pvc != nil {
					spec = pvc.Spec
				}

				Expect(spec.AccessModes).To(Equal([]v1.PersistentVolumeAccessMode{v1.ReadOnlyMany}))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		When("selector value is specified", func() {
			volume := Volumes{
				VolumeName:    "some-name",
				PVCSize:       "10Gi",
				SelectorValue: "some-selector",
			}

			It("sets MatchLabels selector in the spec", func() {
				pvc, _ := k.createPVC(volume)
				Expect(pvc.Spec.Selector).To(Equal(&meta.LabelSelector{
					MatchLabels: configLabels(volume.SelectorValue),
				}))
			})
		})

		When("storage class is specified", func() {
			storageClassName := "ssd"

			volume := Volumes{
				VolumeName:   "some-name",
				PVCSize:      "10Gi",
				StorageClass: storageClassName,
			}

			It("sets StorageClassName in the spec", func() {
				pvc, _ := k.createPVC(volume)
				Expect(pvc.Spec.StorageClassName).To(Equal(&storageClassName))
			})
		})
	})

	Describe("configPorts", func() {

		When("project service has ports defined via ports or expose attributes", func() {
			BeforeEach(func() {
				projectService.Ports = []composego.ServicePortConfig{
					{
						Target:    8080,
						Published: 80,
						HostIP:    "10.10.10.10",
						Protocol:  "tcp",
					},
					{
						Target:    8080,
						Published: 9999,
						HostIP:    "10.10.10.10",
						Protocol:  "tcp",
					},
				}
			})

			It("returns a slice of unique ContainerPort objects", func() {
				p := k.configPorts(projectService)
				Expect(p).To(HaveLen(1))
				Expect(p).To(Equal([]v1.ContainerPort{
					{
						ContainerPort: int32(8080),
						Protocol:      "TCP",
						HostIP:        "10.10.10.10",
					},
				}))
			})
		})
	})

	Describe("configServicePorts", func() {

		When("project service has ports defined via ports or expose attributes", func() {
			BeforeEach(func() {
				projectService.Ports = []composego.ServicePortConfig{
					{
						Target:   8080,
						Protocol: "tcp",
					},
					{
						Target:    8080,
						Published: 9999,
						Protocol:  "tcp",
					},
				}
			})

			It("returns a slice of ServicePort objects", func() {
				p := k.configServicePorts(config.ClusterIPService, projectService)
				Expect(p).To(HaveLen(2))
				Expect(p).To(Equal([]v1.ServicePort{
					{
						Name:     "8080",
						Protocol: "TCP",
						Port:     8080,
						TargetPort: intstr.IntOrString{
							Type:   0,
							IntVal: 8080,
							StrVal: "8080",
						},
						NodePort: 0,
					},
					{
						Name:     "9999",
						Protocol: "TCP",
						Port:     9999,
						TargetPort: intstr.IntOrString{
							Type:   0,
							IntVal: 8080,
							StrVal: "8080",
						},
						NodePort: 0,
					},
				}))
			})

			Context("and nodeport service is in use", func() {
				nodePort := int32(4444)

				BeforeEach(func() {
					projectService.SvcK8sConfig.Service.NodePort = int(nodePort)
				})

				It("specifies that port in the service port spec", func() {
					p := k.configServicePorts(config.NodePortService, projectService)
					Expect(p[0].NodePort).To(Equal(nodePort))
				})
			})
		})
	})

	Describe("configCapabilities", func() {
		When("cap_add capabilities are specified", func() {
			capAdd := "ALL"

			BeforeEach(func() {
				projectService.CapAdd = []string{
					capAdd,
				}
			})

			It("returns capabilities as expected", func() {
				caps := k.configCapabilities(projectService)
				Expect(caps).To(Equal(&v1.Capabilities{
					Add: []v1.Capability{
						v1.Capability(capAdd),
					},
					Drop: make([]v1.Capability, 0),
				}))
			})
		})

		When("cap_drops capabilities are specified", func() {
			capDrop := "NET_ADMIN"

			BeforeEach(func() {
				projectService.CapDrop = []string{
					capDrop,
				}
			})

			It("returns capabilities as expected", func() {
				caps := k.configCapabilities(projectService)
				Expect(caps).To(Equal(&v1.Capabilities{
					Add: make([]v1.Capability, 0),
					Drop: []v1.Capability{
						v1.Capability(capDrop),
					},
				}))
			})
		})
	})

	// @todo
	Describe("configTmpfs", func() {
	})

	// @todo
	Describe("configSecretVolumes", func() {
	})

	// @todo
	Describe("configVolumes", func() {
	})

	Describe("configEmptyVolumeSource", func() {
		When("key passed as `tmpfs`", func() {
			It("returns EmptyDir volume source as expected", func() {
				Expect(k.configEmptyVolumeSource("tmpfs")).To(Equal(&v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{Medium: v1.StorageMediumMemory},
				}))
			})
		})

		When("key is passed with value other than `tmpfs`", func() {
			It("returns EmptyDir volume source as expected", func() {
				Expect(k.configEmptyVolumeSource("")).To(Equal(&v1.VolumeSource{
					EmptyDir: &v1.EmptyDirVolumeSource{},
				}))
			})
		})
	})

	Describe("configConfigMapVolumeSource", func() {
		configMapName := "mymap"
		targetPath := "/mnt/volume"

		When("ConfigMap doesn't use sub-paths", func() {
			configMap := &v1.ConfigMap{}

			It("configures ConfigMapVolumeSource as expected", func() {
				volSrc := k.configConfigMapVolumeSource(configMapName, targetPath, configMap)
				Expect(volSrc).To(Equal(&v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{
							Name: configMapName,
						},
					},
				}))
			})
		})

		When("ConfigMap uses sub-paths", func() {
			configMap := &v1.ConfigMap{
				ObjectMeta: meta.ObjectMeta{
					Annotations: map[string]string{
						"use-subpath": "true",
					},
				},
				Data: map[string]string{
					"key": "some data",
				},
			}

			It("configures ConfigMapVolumeSource as expected", func() {
				volSrc := k.configConfigMapVolumeSource(configMapName, targetPath, configMap)

				_, expectedPath := path.Split(targetPath)

				Expect(volSrc).To(Equal(&v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{
							Name: configMapName,
						},
						Items: []v1.KeyToPath{
							{
								Key:  "key",
								Path: expectedPath,
							},
						},
					},
				}))
			})
		})
	})

	Describe("configHostPathVolumeSource", func() {
		path := "../host/dir"

		JustBeforeEach(func() {
			// path used to generate HostPathVolumeSource
			// is calculated from the base dir determined by the
			// location of the first compose input file, so we need to set it first.
			k.Opt.InputFiles = []string{
				"/path/to/myproject/docker-compose.yaml",
			}
		})

		It("configures HostPathVolumeSource as expected", func() {
			volSrc, err := k.configHostPathVolumeSource(path)
			Expect(err).ToNot(HaveOccurred())
			Expect(volSrc).To(Equal(&v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{Path: "/path/to/host/dir"},
			}))
		})
	})

	Describe("configPVCVolumeSource", func() {
		It("creates PVC volume source as expected", func() {
			claimName := "claimName"
			Expect(k.configPVCVolumeSource(claimName, false)).To(Equal(&v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: claimName,
					ReadOnly:  false,
				},
			}))
		})
	})

	Describe("configEnvs", func() {

		// NOTE: compose-go automatically appends all environment variables defined in env_file (if any)
		// 		 to the list of explicitly defined environment variables for a project service.
		// 		 Values of explicitly defined variables have precedence over the ones coming from env_file.

		Context("with environment variables explicitly defined for project service", func() {
			dummyVal := "123"

			BeforeEach(func() {
				projectService.Environment = composego.MappingWithEquals{
					"ZZZ": &dummyVal,
					"AAA": &dummyVal,
					"FFF": &dummyVal,
				}
			})

			It("sorts project service env vars as expected", func() {
				vars, err := k.configEnvs(projectService)
				Expect(vars).To(HaveLen(3))
				Expect(vars[0].Name).To(Equal("AAA"))
				Expect(vars[1].Name).To(Equal("FFF"))
				Expect(vars[2].Name).To(Equal("ZZZ"))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("for env dependent vars containing double curly braces e.g. {{OTHER_ENV_VAR_NAME}} ", func() {

			secretRef := "postgres://{{USER}}:{{PASS}}@{{HOST}}:{{PORT}}/{{DB}}"

			BeforeEach(func() {
				projectService.Environment = composego.MappingWithEquals{
					"MY_SECRET": &secretRef,
				}
			})

			It("expands that env variable value to reference dependent variables", func() {
				vars, err := k.configEnvs(projectService)

				Expect(vars[0].Value).To(Equal("postgres://$(USER):$(PASS)@$(HOST):$(PORT)/$(DB)"))
				Expect(err).ToNot(HaveOccurred())
			})

		})

		Context("for environment variables values that start with a special case keywords", func() {

			When("env var value starts with a special keyword but doesn't have an expected format", func() {
				secret := "secret.foo"
				config := "config.bar"
				pod := "pod.baz"
				container := "container"

				BeforeEach(func() {
					projectService.Environment = composego.MappingWithEquals{
						"MY_SECRET": &secret,
						"MY_CONFIG": &config,
						"MY_POD":    &pod,
						"MY_CONT":   &container,
					}
				})

				It("treats the value as literal", func() {
					vars, err := k.configEnvs(projectService)

					Expect(vars).To(HaveLen(4))
					Expect(vars).To(ContainElements([]v1.EnvVar{
						{
							Name:  "MY_SECRET",
							Value: secret,
						},
						{
							Name:  "MY_CONFIG",
							Value: config,
						},
						{
							Name:  "MY_POD",
							Value: pod,
						},
						{
							Name:  "MY_CONT",
							Value: container,
						},
					}))

					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("when special case value matches the format ", func() {

				Context("and the symbolic value has insufficient number of elements", func() {
					val := "secret.foo.bar.baz"

					BeforeEach(func() {
						projectService.Environment = composego.MappingWithEquals{
							"MY_SECRET": &val,
						}
					})

					It("returns an error", func() {
						vars, err := k.configEnvs(projectService)

						Expect(vars).To(HaveLen(0))
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(Equal("environment variable MY_SECRET referencing kubernetes secret is invalid: secret.foo.bar.baz"))
					})
				})
			})
		})

		Context("for env vars with symbolic values", func() {

			Context("as secret.secret-name.secret-key", func() {
				secretRef := "secret.my-secret-name.my-secret-key"

				BeforeEach(func() {
					projectService.Environment = composego.MappingWithEquals{
						"MY_SECRET": &secretRef,
					}
				})

				It("expands that env variable to reference secret key", func() {
					vars, err := k.configEnvs(projectService)

					Expect(vars[0].ValueFrom).To(Equal(&v1.EnvVarSource{
						SecretKeyRef: &v1.SecretKeySelector{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "my-secret-name",
							},
							Key: "my-secret-key",
						},
					}))
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("as config.config-name.config-key", func() {
				configRef := "config.my-config-name.my-config-key"

				BeforeEach(func() {
					projectService.Environment = composego.MappingWithEquals{
						"MY_CONFIG": &configRef,
					}
				})

				It("expands that env variable to reference config key", func() {
					vars, err := k.configEnvs(projectService)

					Expect(vars[0].ValueFrom).To(Equal(&v1.EnvVarSource{
						ConfigMapKeyRef: &v1.ConfigMapKeySelector{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "my-config-name",
							},
							Key: "my-config-key",
						},
					}))
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("as pod field path", func() {

				Context("with valid pod field path eg. pod.metadata.namespace", func() {
					configRef := "pod.metadata.namespace"

					BeforeEach(func() {
						projectService.Environment = composego.MappingWithEquals{
							"MY_CONFIG": &configRef,
						}
					})

					It("expands that env variable to reference pod field path", func() {
						vars, err := k.configEnvs(projectService)

						Expect(vars[0].ValueFrom).To(Equal(&v1.EnvVarSource{
							FieldRef: &v1.ObjectFieldSelector{
								FieldPath: "metadata.namespace",
							},
						}))
						Expect(err).ToNot(HaveOccurred())
					})
				})

				Context("with not supported path", func() {
					configRef := "pod.unsupported.path"

					BeforeEach(func() {
						projectService.Environment = composego.MappingWithEquals{
							"MY_CONFIG": &configRef,
						}
					})

					It("doesn't add environment variable with misconfigured reference", func() {
						vars, err := k.configEnvs(projectService)

						Expect(vars).To(HaveLen(0))
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(Equal("environment variable MY_CONFIG references unsupported kubernetes pod field: pod.unsupported.path"))
					})
				})
			})

			Context("as container resource resource field", func() {

				Context("with valid container resource eg. container.{my-container}.limits.cpu", func() {
					configRef := "container.my-container.limits.cpu"

					BeforeEach(func() {
						projectService.Environment = composego.MappingWithEquals{
							"MY_CONFIG": &configRef,
						}
					})

					It("expands that env variable to reference container resource field", func() {
						vars, err := k.configEnvs(projectService)

						Expect(vars[0].ValueFrom).To(Equal(&v1.EnvVarSource{
							ResourceFieldRef: &v1.ResourceFieldSelector{
								ContainerName: "my-container",
								Resource:      "limits.cpu",
							},
						}))
						Expect(err).ToNot(HaveOccurred())
					})
				})

				Context("with not supported resource", func() {
					configRef := "container.my-container.unsupported.resource"

					BeforeEach(func() {
						projectService.Environment = composego.MappingWithEquals{
							"MY_CONFIG": &configRef,
						}
					})

					It("doesn't add environment variable with misconfigured reference", func() {
						vars, err := k.configEnvs(projectService)

						Expect(vars).To(HaveLen(0))
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(Equal("environment variable MY_CONFIG references unsupported kubernetes container resource: container.my-container.unsupported.resource"))
					})
				})
			})
		})
	})

	// @todo
	// covered by partial methods specs
	Describe("createKubernetesObjects", func() {
	})

	Describe("createConfigMapFromComposeConfig", func() {
		configName := "config"

		BeforeEach(func() {
			projectService.Configs = []composego.ServiceConfigObjConfig{
				{
					Source: configName,
					Target: "/some/mount/path",
				},
			}
		})

		Context("for external config", func() {

			JustBeforeEach(func() {
				project.Configs = composego.Configs{
					configName: composego.ConfigObjConfig{
						External: composego.External{
							External: true,
						},
					},
				}
			})

			It("warns and continues", func() {
				var objects []runtime.Object
				newObjs := k.createConfigMapFromComposeConfig(projectService, objects)
				Expect(newObjs).To(HaveLen(0))
			})
		})

		Context("for local config file", func() {
			JustBeforeEach(func() {
				project.Configs = composego.Configs{
					configName: composego.ConfigObjConfig{
						File: "../../testdata/converter/kubernetes/configmaps/config-a",
					},
				}
			})

			It("generates a ConfigMap object and appends it to objects slice", func() {
				var objects []runtime.Object
				newObjs := k.createConfigMapFromComposeConfig(projectService, objects)
				Expect(newObjs).To(HaveLen(1))
			})
		})
	})

	Describe("createNetworkPolicy", func() {
		projectServiceName := "web"
		networkName := "foo"

		It("creates network policy", func() {
			Expect(k.createNetworkPolicy(projectServiceName, networkName)).To(Equal(&networking.NetworkPolicy{
				TypeMeta: meta.TypeMeta{
					Kind:       "NetworkPolicy",
					APIVersion: "networking.k8s.io/v1",
				},
				ObjectMeta: meta.ObjectMeta{
					Name: networkName,
				},
				Spec: networking.NetworkPolicySpec{
					PodSelector: meta.LabelSelector{
						MatchLabels: map[string]string{NetworkLabel + "/" + networkName: "true"},
					},
					Ingress: []networking.NetworkPolicyIngressRule{{
						From: []networking.NetworkPolicyPeer{{
							PodSelector: &meta.LabelSelector{
								MatchLabels: map[string]string{NetworkLabel + "/" + networkName: "true"},
							},
						}},
					}},
				},
			}))
		})
	})

	// @todo
	Describe("updateController", func() {
	})

	Describe("createService", func() {
		BeforeEach(func() {
			projectService.Ports = []composego.ServicePortConfig{
				{
					Target:   8080,
					Protocol: "tcp",
				},
			}
		})

		expectedPorts := []v1.ServicePort{
			{
				Name:     "8080",
				Protocol: "TCP",
				Port:     8080,
				TargetPort: intstr.IntOrString{
					Type:   0,
					IntVal: 8080,
					StrVal: "8080",
				},
				NodePort: 0,
			},
		}

		Context("for headless service type", func() {
			It("creates headless service", func() {
				svc, err := k.createService(config.HeadlessService, projectService)
				Expect(err).NotTo(HaveOccurred())
				Expect(svc.Spec.Type).To(Equal(v1.ServiceTypeClusterIP))
				Expect(svc.Spec.ClusterIP).To(Equal("None"))
				Expect(svc.ObjectMeta.Annotations).To(Equal(configAnnotations(projectService.Labels)))
				Expect(svc.Spec.Ports).To(Equal(expectedPorts))
			})
		})

		Context("for any other service type", func() {
			It("creates a service", func() {
				svc, err := k.createService(config.NodePortService, projectService)
				Expect(err).NotTo(HaveOccurred())
				Expect(svc.Spec.Type).To(Equal(v1.ServiceTypeNodePort))
				Expect(svc.ObjectMeta.Annotations).To(Equal(configAnnotations(projectService.Labels)))
				Expect(svc.Spec.Ports).To(Equal(expectedPorts))
			})
		})
	})

	Describe("createHeadlessService", func() {
		It("creates headless service", func() {
			svc := k.createHeadlessService(projectService)
			Expect(svc.Spec.ClusterIP).To(Equal("None"))
			Expect(svc.ObjectMeta.Annotations).To(Equal(configAnnotations(projectService.Labels)))
			Expect(svc.Spec.Ports).To(Equal([]v1.ServicePort{
				{
					Name:     "headless",
					Protocol: "",
					Port:     55555,
					TargetPort: intstr.IntOrString{
						Type:   0,
						IntVal: 0,
						StrVal: "",
					},
					NodePort: 0,
				},
			}))
		})
	})

	// @todo
	Describe("updateKubernetesObjects", func() {
		var (
			o    *v1apps.Deployment
			objs []runtime.Object
		)

		BeforeEach(func() {
			o = &v1apps.Deployment{
				TypeMeta: meta.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				Spec: v1apps.DeploymentSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: "foo",
								},
							},
						},
					},
				},
			}

			objs = append(objs, o)
		})

		Context("readiness probe", func() {

			When("readiness probe is defined for project service", func() {
				JustBeforeEach(func() {
					svcK8sConfig := config.DefaultSvcK8sConfig()
					svcK8sConfig.Workload.LivenessProbe.Type = config.ProbeTypeNone.String()
					svcK8sConfig.Workload.ReadinessProbe.Type = config.ProbeTypeExec.String()
					svcK8sConfig.Workload.ReadinessProbe.Exec.Command = []string{"hello world"}

					ext, err := svcK8sConfig.Map()
					Expect(err).NotTo(HaveOccurred())
					projectService.Extensions = map[string]interface{}{
						config.K8SExtensionKey: ext,
					}
				})

				It("includes readiness probe definition in the pod spec", func() {
					err := k.updateKubernetesObjects(projectService, &objs)
					Expect(err).ToNot(HaveOccurred())
					Expect(o.Spec.Template.Spec.Containers[0].ReadinessProbe).NotTo(BeNil())
					Expect(o.Spec.Template.Spec.Containers[0].ReadinessProbe.Exec.Command).To(Equal([]string{"hello world"}))
				})
			})

			When("readiness probe is not defined or disabled", func() {
				JustBeforeEach(func() {
					svcK8sConfig := config.SvcK8sConfig{}
					svcK8sConfig.Workload.LivenessProbe.Type = config.ProbeTypeNone.String()
					m, err := svcK8sConfig.Map()

					Expect(err).NotTo(HaveOccurred())

					projectService.Extensions = map[string]interface{}{config.K8SExtensionKey: m}
					projectService, err = NewProjectService(projectService.ServiceConfig)
				})

				It("doesn't include readiness probe definition in the pod spec", func() {
					err := k.updateKubernetesObjects(projectService, &objs)
					Expect(err).ToNot(HaveOccurred())
					Expect(o.Spec.Template.Spec.Containers[0].ReadinessProbe).To(BeNil())
				})
			})
		})
	})

	Describe("sortServicesFirst", func() {
		objs := []runtime.Object{
			&v1beta1.Deployment{
				TypeMeta: meta.TypeMeta{
					Kind: "Deployment",
				},
			},
			&v1.Service{
				TypeMeta: meta.TypeMeta{
					Kind: "Service",
				},
			},
		}

		It("returns objects with services first", func() {
			Expect(objs[0].GetObjectKind().GroupVersionKind().Kind).To(Equal("Deployment"))
			Expect(objs[1].GetObjectKind().GroupVersionKind().Kind).To(Equal("Service"))
			k.sortServicesFirst(&objs)
			Expect(objs[0].GetObjectKind().GroupVersionKind().Kind).To(Equal("Service"))
			Expect(objs[1].GetObjectKind().GroupVersionKind().Kind).To(Equal("Deployment"))
		})
	})

	Describe("removeDupObjects", func() {
		objs := []runtime.Object{
			&v1.ConfigMap{
				TypeMeta: meta.TypeMeta{
					Kind: "ConfigMap",
				},
				ObjectMeta: meta.ObjectMeta{
					Name: "config1",
				},
			},
			&v1.ConfigMap{
				TypeMeta: meta.TypeMeta{
					Kind: "ConfigMap",
				},
				ObjectMeta: meta.ObjectMeta{
					Name: "config1",
				},
			},
		}

		Context("when the same object exists multiple times", func() {
			It("removes duplicates", func() {
				k.removeDupObjects(&objs)
				Expect(objs).To(HaveLen(1))
			})
		})

		Context("with non-duplicate objects", func() {
			objs := append(objs, &v1beta1.Deployment{
				TypeMeta: meta.TypeMeta{
					Kind: "Deployment",
				},
			})

			It("returns them without removing duplicates", func() {
				k.removeDupObjects(&objs)
				Expect(objs).To(HaveLen(2))
				Expect(objs[0].GetObjectKind().GroupVersionKind().Kind).To(Equal("ConfigMap"))
				Expect(objs[1].GetObjectKind().GroupVersionKind().Kind).To(Equal("Deployment"))
			})
		})
	})

	Describe("setPodResources", func() {
		podSpec := &v1.PodTemplateSpec{
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name: "example-container",
					},
				},
			},
		}

		Context("with memory request provided in configuration", func() {
			BeforeEach(func() {
				svcK8sConfig := config.DefaultSvcK8sConfig()
				svcK8sConfig.Workload.Resource.Memory = "10Mi"

				ext, err := svcK8sConfig.Map()
				Expect(err).NotTo(HaveOccurred())
				projectService.Extensions = map[string]interface{}{
					config.K8SExtensionKey: ext,
				}

				projectService, err = NewProjectService(projectService.ServiceConfig)
			})

			It("sets container memory request as expected", func() {
				k.setPodResources(projectService, podSpec)
				Expect(podSpec.Spec.Containers[0].Resources.Requests.Memory().String()).To(Equal("10Mi"))
			})
		})

		Context("with memory limit provided in configuration", func() {
			BeforeEach(func() {
				svcK8sConfig := config.DefaultSvcK8sConfig()
				svcK8sConfig.Workload.Resource.MaxMemory = "10M"

				ext, err := svcK8sConfig.Map()
				Expect(err).NotTo(HaveOccurred())
				projectService.Extensions = map[string]interface{}{
					config.K8SExtensionKey: ext,
				}

				projectService, err = NewProjectService(projectService.ServiceConfig)
			})

			It("sets container memory limit as expected", func() {
				k.setPodResources(projectService, podSpec)
				Expect(podSpec.Spec.Containers[0].Resources.Limits.Memory().String()).To(Equal("10000000"))
			})
		})

		Context("with cpu request provided in configuration", func() {
			BeforeEach(func() {
				svcK8sConfig := config.DefaultSvcK8sConfig()
				svcK8sConfig.Workload.Resource.CPU = "0.1"

				ext, err := svcK8sConfig.Map()
				Expect(err).NotTo(HaveOccurred())
				projectService.Extensions = map[string]interface{}{
					config.K8SExtensionKey: ext,
				}

				projectService, err = NewProjectService(projectService.ServiceConfig)
			})

			It("sets container cpu request as expected", func() {
				k.setPodResources(projectService, podSpec)
				Expect(podSpec.Spec.Containers[0].Resources.Requests.Cpu().String()).To(Equal("100m"))
			})
		})

		Context("with cpu limit provided in configuration", func() {
			BeforeEach(func() {
				svcK8sConfig := config.DefaultSvcK8sConfig()
				svcK8sConfig.Workload.Resource.MaxCPU = "0.5"

				ext, err := svcK8sConfig.Map()
				Expect(err).NotTo(HaveOccurred())
				projectService.Extensions = map[string]interface{}{
					config.K8SExtensionKey: ext,
				}

				projectService, err = NewProjectService(projectService.ServiceConfig)
			})

			It("sets container cpu limit as expected", func() {
				k.setPodResources(projectService, podSpec)
				Expect(podSpec.Spec.Containers[0].Resources.Limits.Cpu().String()).To(Equal("500m"))
			})
		})
	})

	Describe("setPodSecurityContext", func() {
		podSecContext := &v1.PodSecurityContext{}

		When("runAsUser is specified in a k8s extension", func() {
			runAsUser := int64(1000)

			BeforeEach(func() {
				svcK8sConfig := config.DefaultSvcK8sConfig()
				svcK8sConfig.Workload.PodSecurity.RunAsUser = &runAsUser

				m, err := svcK8sConfig.Map()
				Expect(err).NotTo(HaveOccurred())

				projectService.Extensions = map[string]interface{}{config.K8SExtensionKey: m}

				projectService, err = NewProjectService(projectService.ServiceConfig)
				Expect(err).NotTo(HaveOccurred())
			})

			It("adds RunAsUser into pod security context as expected", func() {
				k.setPodSecurityContext(projectService, podSecContext)
				Expect(podSecContext.RunAsUser).To(Equal(&runAsUser))
			})
		})

		When("runAsGroup is specified in a k8s extension", func() {
			runAsGroup := int64(1000)

			BeforeEach(func() {
				svcK8sConfig := config.DefaultSvcK8sConfig()
				svcK8sConfig.Workload.PodSecurity.RunAsGroup = &runAsGroup

				m, err := svcK8sConfig.Map()
				Expect(err).NotTo(HaveOccurred())

				projectService.Extensions = map[string]interface{}{config.K8SExtensionKey: m}

				projectService, err = NewProjectService(projectService.ServiceConfig)
				Expect(err).NotTo(HaveOccurred())
			})

			It("adds RunAsGroup into pod security context as expected", func() {
				k.setPodSecurityContext(projectService, podSecContext)
				Expect(podSecContext.RunAsGroup).To(Equal(&runAsGroup))
			})
		})

		When("fsGroup is specified in a k8s extension", func() {
			fsGroup := int64(1000)

			BeforeEach(func() {
				svcK8sConfig := config.DefaultSvcK8sConfig()
				svcK8sConfig.Workload.PodSecurity.FsGroup = &fsGroup

				m, err := svcK8sConfig.Map()
				Expect(err).NotTo(HaveOccurred())

				projectService.Extensions = map[string]interface{}{config.K8SExtensionKey: m}

				projectService, err = NewProjectService(projectService.ServiceConfig)
				Expect(err).NotTo(HaveOccurred())
			})

			It("adds FSGroup into pod security context as expected", func() {
				k.setPodSecurityContext(projectService, podSecContext)
				Expect(podSecContext.FSGroup).To(Equal(&fsGroup))
			})
		})

		When("group_add is specified in project service spec", func() {

			Context("with numeric value", func() {
				GroupAdd := int64(1000)

				BeforeEach(func() {
					projectService.GroupAdd = []string{strconv.Itoa(int(GroupAdd))}
				})

				It("adds SupplementalGroups into pod security context as expected", func() {
					k.setPodSecurityContext(projectService, podSecContext)
					Expect(podSecContext.SupplementalGroups).To(Equal([]int64{GroupAdd}))
				})
			})

			Context("with non numeric value", func() {
				GroupAdd := "groupname"

				BeforeEach(func() {
					projectService.GroupAdd = []string{GroupAdd}
				})

				It("log a warning and skips that group", func() {
					k.setPodSecurityContext(projectService, podSecContext)
					Expect(podSecContext.SupplementalGroups).To(HaveLen(0))
				})
			})
		})
	})

	Describe("setSecurityContext", func() {
		var (
			secContext *v1.SecurityContext
			caps       *v1.Capabilities
		)

		BeforeEach(func() {
			secContext = &v1.SecurityContext{}
			caps = &v1.Capabilities{}
		})

		When("project service has `privileged` flag set up", func() {
			privileged := true

			BeforeEach(func() {
				projectService.Privileged = privileged
			})

			It("sets Privileged in container security context as expected", func() {
				k.setSecurityContext(projectService, caps, secContext)
				Expect(secContext.Privileged).To(Equal(&privileged))
			})
		})

		When("project service has `user` flag set up", func() {

			Context("as numeric UID", func() {
				user := int64(1000)

				BeforeEach(func() {
					projectService.User = strconv.Itoa(int(user))
				})

				It("sets Privileged in container security context as expected", func() {
					k.setSecurityContext(projectService, caps, secContext)
					Expect(secContext.RunAsUser).To(Equal(&user))
				})
			})

			Context("as non-numeric value", func() {
				BeforeEach(func() {
					projectService.User = "username"
				})

				It("log a warning and doesn't set the user in container security context", func() {
					k.setSecurityContext(projectService, caps, secContext)
					Expect(secContext.RunAsUser).To(BeNil())
				})
			})
		})

		When("capabilities are defined", func() {
			BeforeEach(func() {
				caps.Add = []v1.Capability{
					"ALL",
				}
				caps.Drop = []v1.Capability{
					"NET_ADMIN",
				}
			})

			It("they get set on container security context", func() {
				k.setSecurityContext(projectService, caps, secContext)
				Expect(secContext.Capabilities).To(Equal(caps))
			})
		})
	})
})
