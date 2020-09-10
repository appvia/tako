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

	"github.com/appvia/kube-devx/pkg/kev/config"
	composego "github.com/compose-spec/compose-go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ = Describe("Utils", func() {

	Describe("convertToVersion", func() {

		Context("with unstructured object", func() {
			o := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"hello": "world",
				},
			}
			gv := schema.GroupVersion{Group: "", Version: "v1"}

			It("returns original object unchanged", func() {
				versioned, err := convertToVersion(o, gv)
				Expect(o).To(Equal(versioned))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("with structured object", func() {

			Context("when schema group version is passed", func() {
				o := &v1.List{
					TypeMeta: meta.TypeMeta{
						APIVersion: "group/version",
					},
				}
				gv := schema.GroupVersion{Group: "", Version: "v1"}

				It("returns object with that version", func() {
					versioned, err := convertToVersion(o, gv)
					info := versioned.DeepCopyObject().GetObjectKind().GroupVersionKind()
					Expect(info.Kind).To(Equal("List"))
					Expect(info.Version).To(Equal("v1"))
					Expect(info.Group).To(Equal(""))
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("when schema group version is empty", func() {
				o := &v1beta1.Deployment{
					TypeMeta: meta.TypeMeta{
						Kind:       "Deployment",
						APIVersion: "extensions/v1beta1",
					},
				}
				gv := schema.GroupVersion{}

				It("extracts version information from passed object", func() {
					versioned, err := convertToVersion(o, gv)
					Expect(o).To(Equal(versioned))

					info := versioned.DeepCopyObject().GetObjectKind().GroupVersionKind()
					Expect(info.Kind).To(Equal("Deployment"))
					Expect(info.Version).To(Equal("v1beta1"))
					Expect(info.Group).To(Equal("extensions"))
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})
	})

	Describe("getImagePullPolicy", func() {
		s := "db"

		Context("for valid pull policy string", func() {
			It("returns corresponding v1.PullPolicy", func() {
				Expect(getImagePullPolicy(s, "")).To(Equal(v1.PullAlways))
				Expect(getImagePullPolicy(s, "Always")).To(Equal(v1.PullAlways))
				Expect(getImagePullPolicy(s, "Never")).To(Equal(v1.PullNever))
				Expect(getImagePullPolicy(s, "IfNotPresent")).To(Equal(v1.PullIfNotPresent))
			})

			It("image pull policy string is case insensitive", func() {
				Expect(getImagePullPolicy(s, "")).To(Equal(v1.PullAlways))
				Expect(getImagePullPolicy(s, "always")).To(Equal(v1.PullAlways))
				Expect(getImagePullPolicy(s, "NEVER")).To(Equal(v1.PullNever))
				Expect(getImagePullPolicy(s, "IfNOTPresenT")).To(Equal(v1.PullIfNotPresent))
			})
		})

		Context("for invalid pull policy string", func() {
			policy := "INVALID"

			It("returns an error", func() {
				_, err := getImagePullPolicy(s, policy)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(fmt.Sprintf("Unknown image-pull-policy %s for service %s", policy, s)))
			})
		})
	})

	Describe("getRestartPolicy", func() {
		s := "db"

		Context("for valid restart policy string", func() {
			It("returns corresponding v1.RestartPolicy", func() {
				Expect(getRestartPolicy(s, "")).To(Equal(v1.RestartPolicyAlways))
				Expect(getRestartPolicy(s, "Always")).To(Equal(v1.RestartPolicyAlways))
				Expect(getRestartPolicy(s, "Any")).To(Equal(v1.RestartPolicyAlways))
				Expect(getRestartPolicy(s, "No")).To(Equal(v1.RestartPolicyNever))
				Expect(getRestartPolicy(s, "None")).To(Equal(v1.RestartPolicyNever))
				Expect(getRestartPolicy(s, "Never")).To(Equal(v1.RestartPolicyNever))
				Expect(getRestartPolicy(s, "On-Failure")).To(Equal(v1.RestartPolicyOnFailure))
				Expect(getRestartPolicy(s, "OnFailure")).To(Equal(v1.RestartPolicyOnFailure))
			})

			It("restart policy string is case insensitive", func() {
				Expect(getRestartPolicy(s, "ALWAYS")).To(Equal(v1.RestartPolicyAlways))
				Expect(getRestartPolicy(s, "any")).To(Equal(v1.RestartPolicyAlways))
				Expect(getRestartPolicy(s, "nO")).To(Equal(v1.RestartPolicyNever))
				Expect(getRestartPolicy(s, "NoNE")).To(Equal(v1.RestartPolicyNever))
				Expect(getRestartPolicy(s, "NeVer")).To(Equal(v1.RestartPolicyNever))
				Expect(getRestartPolicy(s, "On-FaILure")).To(Equal(v1.RestartPolicyOnFailure))
				Expect(getRestartPolicy(s, "onFAILURE")).To(Equal(v1.RestartPolicyOnFailure))
			})
		})

		Context("for invalid restart policy string", func() {
			policy := "INVALID"

			It("returns an error", func() {
				_, err := getRestartPolicy(s, policy)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(fmt.Sprintf("Unknown restart policy %s for service %s", policy, s)))
			})
		})
	})

	Describe("getServiceType", func() {

		Context("for valid service type string", func() {
			It("returns corresponding v1.ServiceType string", func() {
				Expect(getServiceType("")).To(Equal(string(v1.ServiceTypeClusterIP)))
				Expect(getServiceType("ClusterIP")).To(Equal(string(v1.ServiceTypeClusterIP)))
				Expect(getServiceType("NodePort")).To(Equal(string(v1.ServiceTypeNodePort)))
				Expect(getServiceType("LoadBalancer")).To(Equal(string(v1.ServiceTypeLoadBalancer)))
				Expect(getServiceType("Headless")).To(Equal(config.HeadlessService))
				Expect(getServiceType("None")).To(Equal(config.NoService))
			})

			It("restart service type string is case insensitive", func() {
				Expect(getServiceType("clusterip")).To(Equal(string(v1.ServiceTypeClusterIP)))
				Expect(getServiceType("CLUSTERIP")).To(Equal(string(v1.ServiceTypeClusterIP)))
			})
		})

		Context("for service type string", func() {
			sType := "INVALID"

			It("returns an error", func() {
				_, err := getServiceType(sType)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(fmt.Errorf("Unknown value %s, supported values are 'none, nodeport, clusterip, headless or loadbalancer'", sType)))
			})
		})
	})

	Describe("sortServices", func() {
		s1 := ProjectService{Name: "z"}
		s2 := ProjectService{Name: "a"}
		s3 := ProjectService{Name: "c"}

		services := composego.Services{}

		p := composego.Project{
			Services: append(services,
				composego.ServiceConfig(s1),
				composego.ServiceConfig(s2),
				composego.ServiceConfig(s3),
			),
		}

		It("sorts services by name ascending", func() {
			sortServices(&p)
			Expect(p.Services[0].Name).To(Equal(s2.Name))
			Expect(p.Services[1].Name).To(Equal(s3.Name))
			Expect(p.Services[2].Name).To(Equal(s1.Name))
		})
	})

	Describe("resetWorkloadAPIVersion", func() {
		o := &v1beta1.Deployment{
			TypeMeta: meta.TypeMeta{
				Kind: "Deployment",
			},
		}

		It("sets group, version and kind on a passed in k8s runtime.Object", func() {
			d := resetWorkloadAPIVersion(o)
			Expect(d.GetObjectKind().GroupVersionKind()).To(Equal(schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			}))
		})
	})

	Describe("durationStrToSecondsInt", func() {

		It("parses duration string into number of seconds (int)", func() {
			expected1 := int32(5)
			expected2 := int32(90)

			Expect(durationStrToSecondsInt("5s")).To(Equal(&expected1))
			Expect(durationStrToSecondsInt("1m30s")).To(Equal(&expected2))
		})

		It("returns nil for empty input string", func() {
			Expect(durationStrToSecondsInt("")).To(BeNil())
		})

		It("returns an error for malformed or unsupported input strings", func() {
			_, err1 := durationStrToSecondsInt("2")
			Expect(err1).To(HaveOccurred())

			_, err2 := durationStrToSecondsInt("abc")
			Expect(err2).To(HaveOccurred())
		})
	})

	Describe("configLabelsWithNetwork", func() {
		svcName := "db"
		networkNameA := "mynetA"
		networkNameB := "mynetB"

		projectService := ProjectService{
			Name: svcName,
			Networks: map[string]*composego.ServiceNetworkConfig{
				networkNameA: {},
				networkNameB: {},
			},
		}

		It("prepares template metadata labels with service and network policy selectors", func() {
			l := configLabelsWithNetwork(projectService)
			Expect(l).To(HaveKeyWithValue(NetworkLabel+"/"+networkNameA, "true"))
			Expect(l).To(HaveKeyWithValue(NetworkLabel+"/"+networkNameA, "true"))
			Expect(l).To(HaveKeyWithValue(Selector, svcName))
			Expect(l).To(HaveLen(3))
		})
	})

	Describe("retrieveVolume", func() {
		var project composego.Project

		JustBeforeEach(func() {
			project = composego.Project{}
		})

		Context("when project services don't contain named service", func() {
			It("returns an error", func() {
				unknowSvcName := "UNKNOWN-SVC-NAME"
				_, err := retrieveVolume(unknowSvcName, &project)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(fmt.Sprintf("Could not find a project service with name %s", unknowSvcName)))
			})
		})

		Context("when project services contain named service", func() {
			s := ProjectService{
				Name: "db",
				Volumes: []composego.ServiceVolumeConfig{
					{
						Source:   "vol1",
						Target:   "/target/path",
						ReadOnly: false,
					},
				},
			}

			JustBeforeEach(func() {
				project.Services = append(project.Services, composego.ServiceConfig(s))
			})

			Context("and project service doesn't reference volumes from other project services (no VolumesFrom present)", func() {
				It("returns volumes for named project service only", func() {
					vols, _ := retrieveVolume(s.Name, &project)
					Expect(vols).To(HaveLen(1))
				})
			})

			Context("and project service references volumes from other project services (VolumesFrom non empty)", func() {

				Context("and the mount path for project volume and dependent volume are the same", func() {

					s2 := ProjectService{
						Name: "other",
						Volumes: []composego.ServiceVolumeConfig{
							{
								Source:   "vol2",
								Target:   "/target/path",
								ReadOnly: false,
							},
						},
						VolumesFrom: []string{s.Name},
					}

					JustBeforeEach(func() {
						project.Services = append(project.Services,
							composego.ServiceConfig(s),
							composego.ServiceConfig(s2),
						)
					})

					It("returns volumes with different mount paths only", func() {
						vols, _ := retrieveVolume(s2.Name, &project)
						Expect(vols).To(HaveLen(1))
					})
				})

				Context("and the mount path for project volume and dependent volume are different", func() {

					s2 := ProjectService{
						Name: "other",
						Volumes: []composego.ServiceVolumeConfig{
							{
								Source:   "vol2",
								Target:   "/other/path",
								ReadOnly: false,
							},
						},
						VolumesFrom: []string{s.Name},
					}

					JustBeforeEach(func() {
						project.Services = append(project.Services,
							composego.ServiceConfig(s),
							composego.ServiceConfig(s2),
						)
					})

					It("returns all volumes", func() {
						vols, _ := retrieveVolume(s2.Name, &project)
						Expect(vols).To(HaveLen(2))
					})
				})
			})
		})
	})

	Describe("parseVols", func() {
		projectSvcName := "web"

		Context("with valid volume name string representations", func() {
			volumeNames := []string{
				"vol1:/some/path",
				"vol2:/another/path/:ro",
			}

			It("converts volume string representation to corresponding slice of Volumes objects", func() {
				vols, err := parseVols(volumeNames, projectSvcName)
				Expect(vols).To(HaveLen(2))
				Expect(vols).To(ContainElements([]Volumes{
					{
						SvcName:       projectSvcName,
						MountPath:     ":/some/path",
						VFrom:         "",
						VolumeName:    "vol1",
						Host:          "",
						Container:     "/some/path",
						Mode:          "",
						PVCName:       "web-claim0",
						PVCSize:       "",
						StorageClass:  "",
						SelectorValue: "",
					},
					{
						SvcName:       projectSvcName,
						MountPath:     ":/another/path/",
						VFrom:         "",
						VolumeName:    "vol2",
						Host:          "",
						Container:     "/another/path/",
						Mode:          "ro",
						PVCName:       "web-claim1",
						PVCSize:       "",
						StorageClass:  "",
						SelectorValue: "",
					},
				}))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("with invalid volume name string representation", func() {
			volumeNames := []string{
				"vol1",
			}

			It("returns an error", func() {
				vols, err := parseVols(volumeNames, projectSvcName)
				Expect(vols).To(HaveLen(0))
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(fmt.Sprintf("Invalid volume format: %s", "vol1")))
			})
		})
	})

	Describe("parseVolume", func() {
		Context("when volume string contains z/Z suffix", func() {
			v := "/foo:/bar:Z"

			It("sets the mode as empty string", func() {
				_, _, _, mode, _ := parseVolume(v)
				Expect(mode).To(Equal(""))
			})
		})

		Context("for volume", func() {

			validateVolumeElements := func(v, expName, expHost, expContainer, expMode string) {
				defer GinkgoRecover()
				name, host, container, mode, err := parseVolume(v)
				Expect(name).To(Equal(expName))
				Expect(host).To(Equal(expHost))
				Expect(container).To(Equal(expContainer))
				Expect(mode).To(Equal(expMode))
				Expect(err).ToNot(HaveOccurred())
			}

			name := "vol"
			host1 := "./foo"
			host2 := "~/bar"
			container1 := "/etc/foo"
			container2 := "/etc/bar/"
			mode := "rw"

			Context("with name:host:container:mode format", func() {
				v := fmt.Sprintf("%s:%s:%s:%s", name, host1, container1, mode)
				validateVolumeElements(v, name, host1, container1, mode)
			})

			Context("with host:container:mode format", func() {
				v := fmt.Sprintf("%s:%s:%s", host2, container2, mode)
				validateVolumeElements(v, "", host2, container2, mode)
			})

			Context("with name:container:mode format", func() {
				v := fmt.Sprintf("%s:%s:%s", name, container1, mode)
				validateVolumeElements(v, name, "", container1, mode)
			})

			Context("with name:host:container format", func() {
				v := fmt.Sprintf("%s:%s:%s", name, host1, container1)
				validateVolumeElements(v, name, host1, container1, "")
			})

			Context("with host:container format", func() {
				v := fmt.Sprintf("%s:%s", host1, container1)
				validateVolumeElements(v, "", host1, container1, "")
			})

			Context("with container:mode format", func() {
				v := fmt.Sprintf("%s:%s", container2, mode)
				validateVolumeElements(v, "", "", container2, mode)
			})

			Context("with name:container format", func() {
				v := fmt.Sprintf("%s:%s", name, container1)
				validateVolumeElements(v, name, "", container1, "")
			})

			Context("with container format", func() {
				v := container2
				validateVolumeElements(v, "", "", container2, "")
			})
		})
	})

	Describe("getVolumeLabels", func() {
		vol := composego.VolumeConfig{
			Labels: composego.Labels{},
		}

		Context("when volume size label present", func() {
			vol.Labels.Add(config.LabelVolumeSize, "10Gi")

			It("returns volume size with value from label", func() {
				size, _, _ := getVolumeLabels(vol)
				Expect(size).To(Equal("10Gi"))
			})
		})

		Context("when volume storage class label present", func() {
			vol.Labels.Add(config.LabelVolumeStorageClass, "ssd")

			It("returns volume storage class with value from label", func() {
				_, _, class := getVolumeLabels(vol)
				Expect(class).To(Equal("ssd"))
			})
		})

		Context("when volume selector label present", func() {
			vol.Labels.Add(config.LabelVolumeSelector, "select")

			It("returns volume selector with value from label", func() {
				_, selector, _ := getVolumeLabels(vol)
				Expect(selector).To(Equal("select"))
			})
		})
	})

	Describe("loadPlacement", func() {

		Context("for supported compose placement constraint", func() {

			Context("node.hostname==xyz...", func() {
				It("returns expected kubernetes node selector", func() {
					Expect(loadPlacement([]string{"node.hostname==myhost"})).To(HaveKeyWithValue("kubernetes.io/hostname", "myhost"))
				})
			})

			Context("node.role==worker", func() {
				It("returns expected kubernetes node selector", func() {
					Expect(loadPlacement([]string{"node.role==worker"})).To(HaveKeyWithValue("node-role.kubernetes.io/worker", "true"))
				})
			})

			Context("node.role==manager", func() {
				It("returns expected kubernetes node selector", func() {
					Expect(loadPlacement([]string{"node.role==manager"})).To(HaveKeyWithValue("node-role.kubernetes.io/master", "true"))
				})
			})

			Context("engine.labels.operatingsystem==linux", func() {
				It("returns expected kubernetes node selector", func() {
					Expect(loadPlacement([]string{"engine.labels.operatingsystem==linux"})).To(HaveKeyWithValue("beta.kubernetes.io/os", "linux"))
				})
			})

			Context("node.labels.(...)", func() {
				It("returns expected kubernetes node selector", func() {
					Expect(loadPlacement([]string{"node.labels.key==value"})).To(HaveKeyWithValue("key", "value"))
				})
			})
		})

		Context("for unsupported placement contraints", func() {
			placement := "invalid==value"

			It("warns user and ignores placement constraint", func() {
				Expect(loadPlacement([]string{placement})).To(HaveLen(0))

				assertLog(logrus.WarnLevel,
					"Constraint in placement is not supported. Only 'node.hostname==...', 'node.role==worker', 'node.role==manager', 'engine.labels.operatingsystem' and 'node.labels.(...)' (ex: node.labels.something==anything) is supported as a constraint",
					map[string]string{
						"placement": "invalid",
					},
				)
			})
		})

	})

	Describe("configAllLabels", func() {
		svcName := "db"
		projectService := ProjectService{
			Name: svcName,
		}

		Context("without any labels defined in deploy block", func() {
			It("returns a map of labels containing selector only", func() {
				Expect(configAllLabels(projectService)).To(HaveKeyWithValue(Selector, svcName))
			})
		})

		Context("with labels specified in deploy block", func() {
			projectService.Deploy = &composego.DeployConfig{
				Labels: composego.Labels{
					"FOO": "BAR",
				},
			}

			It("includes deploy labels", func() {
				Expect(configAllLabels(projectService)).To(HaveKeyWithValue("FOO", "BAR"))
			})
		})
	})

	Describe("configAnnotations", func() {

		Context("with project service labels present", func() {

			Context("containing kev specific labels", func() {
				projectService := ProjectService{
					Labels: composego.Labels{
						"FOO":                   "BAR",
						config.LabelServiceType: "value",
					},
				}

				It("returns a map of labels excluding kev control labels", func() {
					Expect(configAnnotations(projectService)).To(HaveLen(1))
					Expect(configAnnotations(projectService)).To(HaveKeyWithValue("FOO", "BAR"))
				})
			})

			Context("not containing kev specific labels", func() {
				projectService := ProjectService{
					Labels: composego.Labels{
						"FOO": "BAR",
						"BAR": "BAZ",
					},
				}

				It("returns a map of labels as expected", func() {
					Expect(configAnnotations(projectService)).To(HaveLen(2))
					Expect(configAnnotations(projectService)).To(HaveKeyWithValue("FOO", "BAR"))
					Expect(configAnnotations(projectService)).To(HaveKeyWithValue("BAR", "BAZ"))
				})
			})
		})
	})
})
