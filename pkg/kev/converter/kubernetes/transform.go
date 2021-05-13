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

// Note: Some functionality below have been extracted from Kompose project
// and updated accordingly to meet new dependencies and requirements of this tool.
// Functions below have link to original Kompose code for reference.

package kubernetes

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/appvia/kev/pkg/kev/config"
	"github.com/appvia/kev/pkg/kev/log"
	kmd "github.com/appvia/komando"
	composego "github.com/compose-spec/compose-go/types"
	"github.com/pkg/errors"

	"github.com/spf13/cast"
	v1apps "k8s.io/api/apps/v1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	v1batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Kubernetes transformer
type Kubernetes struct {
	Opt      ConvertOptions     // user provided options from the command line
	Project  *composego.Project // docker compose project
	Excluded []string           // docker compose service names that should be excluded
	UI       kmd.UI
}

// Transform converts compose project to set of k8s objects
// returns object that are already sorted in the way that Services are first
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L1140
func (k *Kubernetes) Transform() ([]runtime.Object, error) {
	// holds all the converted objects
	var allobjects []runtime.Object
	var renderedNetworkPolicy runtime.Object

	sg := k.UI.StepGroup()
	defer sg.Done()

	// @step iterate over defined secrets and build Secret objects accordingly
	if k.Project.Secrets != nil && len(k.Project.Secrets) > 0 {
		stepSecrets := sg.Add("Converting project secrets")
		secrets, err := k.createSecrets()
		if err != nil {
			msg := "Unable to create Secret resource"
			log.Error(msg)
			stepSecrets.Error()
			return nil, errors.Wrapf(err, "%s, details:\n", msg)
		}
		for _, item := range secrets {
			allobjects = append(allobjects, item)
		}
		stepSecrets.Success("Converted project secrets")
	}

	// @step sort project services by name for consistency
	sortServices(k.Project)

	// @step iterate over sorted service definitions
	for _, pSvc := range k.Project.Services {
		// @step skip service if excluded
		if contains(k.Excluded, pSvc.Name) {
			continue
		}

		stepSvc := sg.Add(fmt.Sprintf("Converting service: %s", pSvc.Name))
		var objects []runtime.Object

		projectService, err := NewProjectService(pSvc)
		if err != nil {
			return nil, err
		}

		// @step skip disabled services
		if !projectService.enabled() {
			continue
		}

		// @step normalise project service name
		if rfc1123dns(projectService.Name) != projectService.Name {
			log.DebugfWithFields(log.Fields{
				"project-service": projectService.Name,
			}, "Compose service name normalised to %q",
				rfc1123dns(projectService.Name))

			projectService.Name = rfc1123dns(projectService.Name)
		}

		// @step we're not concerned about building & publishing images yet,
		// but will validate presence of image key for each service.
		// If there's no "image" key, use the name of the container that's built
		if projectService.Image == "" {
			projectService.Image = projectService.Name
		}
		if projectService.Image == "" {
			stepSvc.Error()
			return nil, fmt.Errorf("image key required within build parameters in order to build and push service '%s'", projectService.Name)
		}

		// @step create kubernetes object (never create a pod in isolation!)
		// https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-lifetime
		objects = k.createKubernetesObjects(projectService)

		// @step create service / ingress
		serviceType, err := projectService.serviceType()
		if err != nil {
			msg := "Could not establish service type. Service hasn't been created!"
			log.Error(msg)
			stepSvc.Error()
			return nil, errors.Wrapf(err, "%s, details:\n", msg)
		}

		if k.portsExist(projectService) && serviceType != config.NoService {
			// Create a k8s service of a type specified by the compose service config,
			// only if ports are defined and service type is different than NoService
			svc := k.createService(serviceType, projectService)
			objects = append(objects, svc)

			// For exposed service also create an ingress (Note only the first port is used for ingress!)
			expose, err := projectService.exposeService()
			if err != nil {
				msg := "Could not expose the service. Ingress hasn't been created!"
				log.Error(msg)
				stepSvc.Error()
				return nil, errors.Wrapf(err, "%s, details:\n", msg)
			}
			if expose != "" {
				objects = append(objects, k.initIngress(projectService, svc.Spec.Ports[0].Port))
			}
		} else if serviceType == config.HeadlessService {
			// No ports defined - creating headless service instead
			svc := k.createHeadlessService(projectService)
			objects = append(objects, svc)
		}

		// @step updating all objects related to a current compose service
		if err = k.updateKubernetesObjects(projectService, &objects); err != nil {
			msg := "Error occurred while transforming Kubernetes objects"
			log.Error(msg)
			stepSvc.Error()
			return nil, errors.Wrapf(err, "%s, details:\n", msg)
		}

		stepSvc.Success(fmt.Sprintf("Converted service: %s", pSvc.Name))
		for _, object := range objects {
			k.UI.Output(
				fmt.Sprintf("rendered %s", object.GetObjectKind().GroupVersionKind().Kind),
				kmd.WithStyle(kmd.LogStyle),
				kmd.WithIndent(3),
				kmd.WithIndentChar(kmd.LogIndentChar),
			)
		}

		// @step create network policies if networks defined
		if len(projectService.Networks) > 0 {
			for name := range projectService.Networks {
				log.DebugWithFields(log.Fields{
					"project-service": projectService.Name,
					"network-name":    name,
				}, "Network detected and will be converted to equivalent NetworkPolicy")

				np, err := k.createNetworkPolicy(projectService.Name, name)
				if err != nil {
					msg := fmt.Sprintf("Unable to create Network Policy for network %v for service %v", name, projectService.Name)
					log.Error(msg)
					stepSvc.Error()
					return nil, err
				}
				objects = append(objects, np)
				renderedNetworkPolicy = np
			}
		}

		allobjects = append(allobjects, objects...)
	}

	if renderedNetworkPolicy != nil {
		sg.Add("Networking").Success()
		k.UI.Output(
			fmt.Sprintf("rendered %s", renderedNetworkPolicy.GetObjectKind().GroupVersionKind().Kind),
			kmd.WithStyle(kmd.LogStyle),
			kmd.WithIndent(3),
			kmd.WithIndentChar(kmd.LogIndentChar),
		)
	}

	// @step sort all object so Services are first and remove duplicates
	k.sortServicesFirst(&allobjects)
	k.removeDupObjects(&allobjects)

	return allobjects, nil
}

// initPodSpec creates the pod specification
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L129
func (k *Kubernetes) initPodSpec(projectService ProjectService) v1.PodSpec {

	// @step determine docker image
	image := projectService.Image
	if image == "" {
		image = projectService.Name
	}

	// @step get image pull secret for the pod
	pullSecret := projectService.imagePullSecret()

	// @step get service account for the pod
	serviceAccount := projectService.serviceAccountName()

	pod := v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:  projectService.Name,
				Image: image,
			},
		},
	}
	if pullSecret != "" {
		pod.ImagePullSecrets = []v1.LocalObjectReference{
			{
				Name: pullSecret,
			},
		}
	}
	if serviceAccount != "" {
		pod.ServiceAccountName = serviceAccount
	}

	return pod
}

// getConfigMapKeyFromMeta gets configmap from project configs
func (k *Kubernetes) getConfigMapKeyFromMeta(configName string) (string, error) {
	if k.Project.Configs == nil {
		return "", fmt.Errorf("config %s not found", configName)
	}

	if _, ok := k.Project.Configs[configName]; !ok {
		return "", fmt.Errorf("config %s not found", configName)
	}

	config := k.Project.Configs[configName]

	if config.External.External {
		return "", fmt.Errorf("config %s is external", configName)
	}

	return filepath.Base(config.File), nil
}

// initPodSpecWithConfigMap creates the pod specification
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L154
func (k *Kubernetes) initPodSpecWithConfigMap(projectService ProjectService) v1.PodSpec {
	var volumeMounts []v1.VolumeMount
	var volumes []v1.Volume

	for _, value := range projectService.Configs {
		cmVolName := formatFileName(value.Source)
		target := value.Target
		if target == "" {
			// short syntax, = /<source>
			target = "/" + value.Source
		}
		subPath := filepath.Base(target)

		volSource := v1.ConfigMapVolumeSource{}
		volSource.Name = cmVolName

		key, err := k.getConfigMapKeyFromMeta(value.Source)
		if err != nil {
			// config is most likely defined as external
			log.WarnfWithFields(log.Fields{
				"project-service": projectService.Name,
				"config":          value.Source,
			}, "Cannot parse config: %s", err.Error())

			continue
		}

		volSource.Items = []v1.KeyToPath{{
			Key:  key,
			Path: subPath,
		}}

		if value.Mode != nil {
			tmpMode := int32(*value.Mode)
			volSource.DefaultMode = &tmpMode
		}

		cmVol := v1.Volume{
			Name:         cmVolName,
			VolumeSource: v1.VolumeSource{ConfigMap: &volSource},
		}

		volumeMounts = append(volumeMounts,
			v1.VolumeMount{
				Name:      cmVolName,
				MountPath: target,
				SubPath:   subPath,
			})

		volumes = append(volumes, cmVol)
	}

	pod := k.initPodSpec(projectService)
	pod.Containers = []v1.Container{
		{
			Name:         projectService.Name,
			Image:        projectService.Image,
			VolumeMounts: volumeMounts,
		},
	}
	pod.Volumes = volumes

	return pod
}

// initSvc initializes Kubernetes Service object
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L240
func (k *Kubernetes) initSvc(projectService ProjectService) *v1.Service {
	svc := &v1.Service{
		TypeMeta: meta.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:   rfc1123label(projectService.Name),
			Labels: configLabels(projectService.Name),
		},
		Spec: v1.ServiceSpec{
			Selector: configLabels(projectService.Name),
		},
	}
	return svc
}

// initConfigMap initialises ConfigMap object
func (k *Kubernetes) initConfigMap(projectService ProjectService, configMapName string, data map[string]string) *v1.ConfigMap {
	return &v1.ConfigMap{
		TypeMeta: meta.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:   rfc1123dns(configMapName),
			Labels: configLabels(projectService.Name),
		},
		Data: data,
	}
}

// initConfigMapFromFileOrDir will create a configmap from dir or file
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L288
func (k *Kubernetes) initConfigMapFromFileOrDir(projectService ProjectService, configMapName, filePath string) (*v1.ConfigMap, error) {
	configMap := &v1.ConfigMap{}

	fi, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	switch mode := fi.Mode(); {
	case mode.IsDir():
		configMap, err = k.initConfigMapFromDir(projectService, configMapName, filePath)
		if err != nil {
			return nil, fmt.Errorf("Couldn't initiate ConfigMap from directory: %s", err)
		}

	case mode.IsRegular():
		configMap, err = k.initConfigMapFromFile(projectService, filePath)
		if err != nil {
			return nil, fmt.Errorf("Couldn't initiate ConfigMap from file: %s", err)
		}
		configMap.Name = rfc1123dns(configMapName) // always override name with passed value
		configMap.Annotations = map[string]string{
			"use-subpath": "true",
		}
	}

	return configMap, nil
}

// initConfigMapFromDir initialised ConfigMap from a directory
func (k *Kubernetes) initConfigMapFromDir(projectService ProjectService, configMapName, dir string) (*v1.ConfigMap, error) {
	dataMap := make(map[string]string)

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() {
			log.DebugWithFields(log.Fields{
				"project-service": projectService.Name,
				"file":            file.Name(),
			}, "Read file to ConfigMap")

			data, err := getContentFromFile(dir + "/" + file.Name())
			if err != nil {
				return nil, err
			}
			dataMap[file.Name()] = data
		}
	}

	return k.initConfigMap(projectService, configMapName, dataMap), nil
}

// initConfigMapFromFile initializes a ConfigMap object from a single file
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L350
func (k *Kubernetes) initConfigMapFromFile(projectService ProjectService, fileName string) (*v1.ConfigMap, error) {
	content, err := getContentFromFile(fileName)
	if err != nil {
		log.ErrorfWithFields(log.Fields{
			"project-service": projectService.Name,
			"file":            fileName,
		}, "Unable to retrieve file to initialise ConfigMap from: %s", err.Error())

		return nil, err
	}

	dataMap := make(map[string]string)
	dataMap[filepath.Base(fileName)] = content

	configMapName := ""
	for key, tmpConfig := range k.Project.Configs {
		if tmpConfig.File == fileName {
			configMapName = key
		}
	}

	if configMapName == "" {
		return nil, fmt.Errorf("No config found matching the file name")
	}

	return k.initConfigMap(projectService, configMapName, dataMap), nil
}

// initDeployment initializes Kubernetes Deployment object
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L380
func (k *Kubernetes) initDeployment(projectService ProjectService) *v1apps.Deployment {
	var podSpec v1.PodSpec
	if len(projectService.Configs) > 0 {
		podSpec = k.initPodSpecWithConfigMap(projectService)
	} else {
		podSpec = k.initPodSpec(projectService)
	}

	replicas := projectService.replicas()

	dc := &v1apps.Deployment{
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
			Template: v1.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Annotations: configAnnotations(projectService),
					Labels:      configLabels(projectService.Name),
				},
				Spec: podSpec,
			},
		},
	}

	// @step add update strategy if present
	update := projectService.getKubernetesUpdateStrategy()
	if update != nil {
		dc.Spec.Strategy = v1apps.DeploymentStrategy{
			Type:          v1apps.RollingUpdateDeploymentStrategyType,
			RollingUpdate: update,
		}

		log.DebugWithFields(log.Fields{
			"project-service": projectService.Name,
			"max-surge":       update.MaxSurge.String(),
			"max-unavailable": update.MaxUnavailable.String(),
		}, "Set deployment rolling update")
	}

	return dc
}

// initDaemonSet initializes Kubernetes DaemonSet object
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L427
func (k *Kubernetes) initDaemonSet(projectService ProjectService) *v1apps.DaemonSet {
	ds := &v1apps.DaemonSet{
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
	}
	return ds
}

// initStatefulSet initialises a new StatefulSet
func (k *Kubernetes) initStatefulSet(projectService ProjectService) *v1apps.StatefulSet {
	var podSpec v1.PodSpec
	if len(projectService.Configs) > 0 {
		podSpec = k.initPodSpecWithConfigMap(projectService)
	} else {
		podSpec = k.initPodSpec(projectService)
	}

	replicas := projectService.replicas()

	sts := &v1apps.StatefulSet{
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
					Annotations: configAnnotations(projectService),
					Labels:      configLabels(projectService.Name),
				},
				Spec: podSpec,
			},
			ServiceName: projectService.Name,
			UpdateStrategy: v1apps.StatefulSetUpdateStrategy{
				Type:          v1apps.RollingUpdateStatefulSetStrategyType,
				RollingUpdate: &v1apps.RollingUpdateStatefulSetStrategy{},
			},
		},
	}

	return sts
}

// initJob initialises a new Kubernetes Job
func (k *Kubernetes) initJob(projectService ProjectService, replicas int) *v1batch.Job {
	repl := int32(replicas)

	var podSpec v1.PodSpec
	if len(projectService.Configs) > 0 {
		podSpec = k.initPodSpecWithConfigMap(projectService)
	} else {
		podSpec = k.initPodSpec(projectService)
	}

	j := &v1batch.Job{
		TypeMeta: meta.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:   projectService.Name,
			Labels: configAllLabels(projectService),
		},
		Spec: v1batch.JobSpec{
			Parallelism: &repl,
			Completions: &repl,
			Selector: &meta.LabelSelector{
				MatchLabels: configLabels(projectService.Name),
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Annotations: configAnnotations(projectService),
					Labels:      configLabels(projectService.Name),
				},
				Spec: podSpec,
			},
		},
	}

	return j
}

// initIngress initialises ingress object
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L446
// @todo change to networkingv1 after migration to k8s 0.19
func (k *Kubernetes) initIngress(projectService ProjectService, port int32) *networkingv1beta1.Ingress {
	expose, _ := projectService.exposeService()
	if expose == "" {
		return nil
	}
	hosts := regexp.MustCompile("[ ,]*,[ ,]*").Split(expose, -1)

	ingress := &networkingv1beta1.Ingress{
		TypeMeta: meta.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1beta1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:        projectService.Name,
			Labels:      configLabels(projectService.Name),
			Annotations: configAnnotations(projectService),
		},
		Spec: networkingv1beta1.IngressSpec{
			Rules: make([]networkingv1beta1.IngressRule, len(hosts)),
		},
	}

	for i, host := range hosts {
		host, p := parseIngressPath(host)
		ingress.Spec.Rules[i] = networkingv1beta1.IngressRule{
			IngressRuleValue: networkingv1beta1.IngressRuleValue{
				HTTP: &networkingv1beta1.HTTPIngressRuleValue{
					Paths: []networkingv1beta1.HTTPIngressPath{
						{
							Path: p,
							Backend: networkingv1beta1.IngressBackend{
								ServiceName: projectService.Name,
								ServicePort: intstr.IntOrString{
									IntVal: port,
								},
							},
						},
					},
				},
			},
		}
		if host != "true" {
			ingress.Spec.Rules[i].Host = host
		}
	}

	tlsSecretName := projectService.tlsSecretName()
	if tlsSecretName != "" {
		ingress.Spec.TLS = []networkingv1beta1.IngressTLS{
			{
				Hosts:      hosts,
				SecretName: tlsSecretName,
			},
		}
	}

	return ingress
}

// initHpa intialised horizontal pod autoscaler for a project service
func (k *Kubernetes) initHpa(projectService ProjectService, target runtime.Object) *autoscalingv2beta2.HorizontalPodAutoscaler {
	t := reflect.ValueOf(target).Elem()
	typeMeta := t.FieldByName("TypeMeta").Interface().(meta.TypeMeta)
	if !contains([]string{"Deployment", "StatefulSet"}, typeMeta.Kind) {
		log.WarnWithFields(log.Fields{
			"project-service": projectService.Name,
			"kind":            typeMeta.Kind,
		}, "Unsupported target kind for Horizontal Pod Autoscaler. Skipping ...")

		return nil
	}

	replicas := projectService.replicas()
	maxRepl := projectService.autoscaleMaxReplicas()
	targetCPUUtilization := projectService.autoscaleTargetCPUUtilization()
	targetMemoryUtilization := projectService.autoscaleTargetMemoryUtilization()

	// if replicas set to 0, autobump to at least 1
	if replicas == 0 {
		replicas = 1
	}

	// no HPA without max replicas
	if maxRepl == 0 {
		return nil
	}

	// max replicas should be greater than min replicas!
	if maxRepl > 0 && maxRepl <= replicas {
		log.WarnWithFields(log.Fields{
			"project-service":        projectService.Name,
			"replicas":               replicas,
			"autoscale-max-replicas": maxRepl,
		}, "Max replicas must be greater than initial replicas number for the Horizontal Pod Autoscaler. Skipping ...")

		return nil
	}

	metrics := []autoscalingv2beta2.MetricSpec{}

	if targetCPUUtilization > 0 {
		metrics = append(metrics, autoscalingv2beta2.MetricSpec{
			Type: "Resource",
			Resource: &autoscalingv2beta2.ResourceMetricSource{
				Name: "cpu",
				Target: autoscalingv2beta2.MetricTarget{
					Type:               "Utilization",
					AverageUtilization: &targetCPUUtilization,
				},
			},
		})
	}

	if targetMemoryUtilization > 0 {
		metrics = append(metrics, autoscalingv2beta2.MetricSpec{
			Type: "Resource",
			Resource: &autoscalingv2beta2.ResourceMetricSource{
				Name: "memory",
				Target: autoscalingv2beta2.MetricTarget{
					Type:               "Utilization",
					AverageUtilization: &targetMemoryUtilization,
				},
			},
		})
	}

	return &autoscalingv2beta2.HorizontalPodAutoscaler{
		TypeMeta: meta.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: "autoscaling/v2beta2",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:        projectService.Name,
			Labels:      configLabels(projectService.Name),
			Annotations: configAnnotations(projectService),
		},
		Spec: autoscalingv2beta2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2beta2.CrossVersionObjectReference{
				Kind:       typeMeta.Kind,
				APIVersion: typeMeta.APIVersion,
				Name:       projectService.Name,
			},
			MinReplicas: &replicas,
			MaxReplicas: maxRepl,
			Metrics:     metrics,
		},
		Status: autoscalingv2beta2.HorizontalPodAutoscalerStatus{
			Conditions: []autoscalingv2beta2.HorizontalPodAutoscalerCondition{},
		},
	}
}

// createSecrets create secrets
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L502
func (k *Kubernetes) createSecrets() ([]*v1.Secret, error) {
	var objects []*v1.Secret
	for name, secretConfig := range k.Project.Secrets {
		if secretConfig.File != "" {
			dataString, err := getContentFromFile(secretConfig.File)
			if err != nil {
				log.ErrorWithFields(log.Fields{
					"file": secretConfig.File,
				}, "Unable to read secret(s) from file")

				return nil, err
			}
			data := []byte(dataString)
			secret := &v1.Secret{
				TypeMeta: meta.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: meta.ObjectMeta{
					Name:   name,
					Labels: configLabels(name),
				},
				Type: v1.SecretTypeOpaque,
				Data: map[string][]byte{name: data},
			}
			objects = append(objects, secret)
		} else {
			log.WarnWithFields(log.Fields{
				"secret-name": name,
			}, "Your deployment(s) expects secret to exist in the target K8s cluster namespace.")
			log.Warn("Follow the official guidelines on how to create K8s secrets manually")
			log.Warn("https://kubernetes.io/docs/tasks/inject-data-application/distribute-credentials-secure/")
		}
	}

	return objects, nil
}

// createPVC initializes PersistentVolumeClaim
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L534
func (k *Kubernetes) createPVC(volume Volumes) (*v1.PersistentVolumeClaim, error) {
	// @step get size quantity
	volSize, err := resource.ParseQuantity(volume.PVCSize)
	if err != nil {
		log.Error("Error parsing volume size")
		return nil, err
	}

	pvc := &v1.PersistentVolumeClaim{
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
					v1.ResourceStorage: volSize,
				},
			},
		},
	}

	if len(volume.SelectorValue) > 0 {
		pvc.Spec.Selector = &meta.LabelSelector{
			MatchLabels: configLabels(volume.SelectorValue),
		}
	}

	if len(volume.StorageClass) > 0 {
		pvc.Spec.StorageClassName = &volume.StorageClass
	}

	if volume.Mode == "ro" {
		pvc.Spec.AccessModes = []v1.PersistentVolumeAccessMode{v1.ReadOnlyMany}
	} else {
		pvc.Spec.AccessModes = []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}
	}

	return pvc, nil
}

// configPorts configures the container ports.
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L573
func (k *Kubernetes) configPorts(projectService ProjectService) []v1.ContainerPort {
	ports := []v1.ContainerPort{}
	exist := map[string]bool{}
	for _, port := range projectService.ports() {

		// @step upcase compose-go port protocol
		protocol := strings.ToUpper(port.Protocol)

		// @step skip port if already processed
		if exist[fmt.Sprint(port.Target)+protocol] {
			continue
		}

		ports = append(ports, v1.ContainerPort{
			ContainerPort: int32(port.Target),
			Protocol:      v1.Protocol(protocol),
			HostIP:        port.HostIP,
		})

		exist[fmt.Sprint(port.Target)+protocol] = true
	}

	return ports
}

// configServicePorts configure the container service ports.
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L602
func (k *Kubernetes) configServicePorts(serviceType string, projectService ProjectService) []v1.ServicePort {
	servicePorts := []v1.ServicePort{}
	seenPorts := make(map[int]struct{}, len(projectService.ports()))

	var servicePort v1.ServicePort
	for _, port := range projectService.ports() {
		if port.Published == 0 {
			port.Published = port.Target
		}

		var targetPort intstr.IntOrString
		targetPort.IntVal = int32(port.Target)
		targetPort.StrVal = strconv.Itoa(int(port.Target))

		// @step define port name depending on whether it was seen before
		name := strconv.Itoa(int(port.Published))
		if _, ok := seenPorts[int(port.Published)]; ok {
			// https://github.com/kubernetes/kubernetes/issues/2995
			if strings.EqualFold(serviceType, string(v1.ServiceTypeLoadBalancer)) {
				log.WarnWithFields(log.Fields{
					"project-service": projectService.Name,
					"port":            port.Published,
				}, "LoadBalancer service type cannot use TCP and UDP for the same port")
			}
			name = fmt.Sprintf("%s-%s", name, strings.ToLower(string(port.Protocol)))
		}

		servicePort = v1.ServicePort{
			Name:       name,
			Port:       int32(port.Published),
			TargetPort: targetPort,
			Protocol:   v1.Protocol(strings.ToUpper(port.Protocol)), // compose-go port protocol is lowercase
		}

		// For NodePort service type specify port value
		np := projectService.nodePort()
		if strings.EqualFold(serviceType, string(v1.ServiceTypeNodePort)) && np != 0 {
			servicePort.NodePort = np
		}

		servicePorts = append(servicePorts, servicePort)
		seenPorts[int(port.Published)] = struct{}{}
	}

	return servicePorts
}

// configCapabilities configure POSIX capabilities that can be added or removed to a container
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L648
func (k *Kubernetes) configCapabilities(projectService ProjectService) *v1.Capabilities {
	capsAdd := []v1.Capability{}
	capsDrop := []v1.Capability{}

	for _, capAdd := range projectService.CapAdd {
		capsAdd = append(capsAdd, v1.Capability(capAdd))
	}

	for _, capDrop := range projectService.CapDrop {
		capsDrop = append(capsDrop, v1.Capability(capDrop))
	}

	return &v1.Capabilities{
		Add:  capsAdd,
		Drop: capsDrop,
	}
}

// configTmpfs configure the tmpfs.
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L664
func (k *Kubernetes) configTmpfs(projectService ProjectService) ([]v1.VolumeMount, []v1.Volume) {
	volumeMounts := []v1.VolumeMount{}
	volumes := []v1.Volume{}

	for index, volume := range projectService.Tmpfs {
		// @step naming volumes if multiple tmpfs are provided
		volumeName := fmt.Sprintf("%s-tmpfs%d", projectService.Name, index)
		volume = strings.Split(volume, ":")[0]
		// @step create a new volume mount object and append to list
		volMount := v1.VolumeMount{
			Name:      volumeName,
			MountPath: volume,
		}
		volumeMounts = append(volumeMounts, volMount)

		// @step create tmpfs specific empty volumes
		volSource := k.configEmptyVolumeSource("tmpfs")

		// @step create a new volume object using the volsource and add to list
		vol := v1.Volume{
			Name:         volumeName,
			VolumeSource: *volSource,
		}

		volumes = append(volumes, vol)
	}

	return volumeMounts, volumes
}

// configSecretVolumes config volumes from secret.
// Link: https://docs.docker.com/compose/compose-file/#secrets
// In kubernetes' Secret resource, it has a data structure like a map[string]bytes, every key will act like the file name
// when mount to a container. This is the part that missing in compose. So we will create a single key secret from compose
// config and the key's name will be the secret's name, it's value is the file content.
// compose'secret can only be mounted at `/run/secrets`, so we will hardcoded this.
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L699
func (k *Kubernetes) configSecretVolumes(projectService ProjectService) ([]v1.VolumeMount, []v1.Volume) {
	var volumeMounts []v1.VolumeMount
	var volumes []v1.Volume

	if len(projectService.Secrets) > 0 {
		for _, secretConfig := range projectService.Secrets {
			if secretConfig.UID != "" {
				log.WarnWithFields(log.Fields{
					"project-service": projectService.Name,
				}, "Ignoring `uid` field on compose project service secret")
			}
			if secretConfig.GID != "" {
				log.WarnWithFields(log.Fields{
					"project-service": projectService.Name,
				}, "Ignoring `gid` field on compose project service secret")
			}

			var itemPath string // should be the filename
			var mountPath = ""  // should be the directory

			// short-syntax
			if secretConfig.Target == "" {
				// the secret path (mountPath) should be inside the default directory /run/secrets
				mountPath = "/run/secrets/" + secretConfig.Source
				// the itemPath should be the source itself
				itemPath = secretConfig.Source
			} else {
				// long-syntax, get the last part of the path and consider it the filename
				pathSplitted := strings.Split(secretConfig.Target, "/")
				lastPart := pathSplitted[len(pathSplitted)-1]

				// if the filename (lastPart) and the target is the same
				if lastPart == secretConfig.Target {
					// the secret path should be the source (it need to be inside a directory and only the filename was given)
					mountPath = secretConfig.Source
				} else {
					// should then get the target without the filename (lastPart)
					mountPath = mountPath + strings.TrimSuffix(secretConfig.Target, "/"+lastPart)
				}

				// if the target isn't absolute path
				if !strings.HasPrefix(secretConfig.Target, "/") {
					// concat the default secret directory
					mountPath = "/run/secrets/" + mountPath
				}

				itemPath = lastPart
			}

			volSource := v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: secretConfig.Source,
					Items: []v1.KeyToPath{{
						Key:  secretConfig.Source,
						Path: itemPath,
					}},
				},
			}

			if secretConfig.Mode != nil {
				mode := cast.ToInt32(*secretConfig.Mode)
				volSource.Secret.DefaultMode = &mode
			}

			vol := v1.Volume{
				Name:         secretConfig.Source,
				VolumeSource: volSource,
			}
			volumes = append(volumes, vol)

			volMount := v1.VolumeMount{
				Name:      vol.Name,
				MountPath: mountPath,
			}
			volumeMounts = append(volumeMounts, volMount)
		}
	}

	return volumeMounts, volumes
}

// configVolumes configure the container volumes.
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L774
func (k *Kubernetes) configVolumes(projectService ProjectService) ([]v1.VolumeMount, []v1.Volume, []*v1.PersistentVolumeClaim, []*v1.ConfigMap, error) {
	volumeMounts := []v1.VolumeMount{}
	volumes := []v1.Volume{}
	var PVCs []*v1.PersistentVolumeClaim
	var cms []*v1.ConfigMap
	var volumeName string

	// @step set volumes configuration based on user preference: empty volumes vs PVC vs volume claims
	useEmptyVolumes := k.Opt.EmptyVols
	useHostPath := k.Opt.Volumes == "hostPath"
	useConfigMap := k.Opt.Volumes == "configMap"

	if k.Opt.Volumes == "emptyDir" {
		useEmptyVolumes = true
	}

	// @step config volumes from secret if present
	secretsVolumeMounts, secretsVolumes := k.configSecretVolumes(projectService)
	volumeMounts = append(volumeMounts, secretsVolumeMounts...)
	volumes = append(volumes, secretsVolumes...)

	var count int
	// @step iterate over project service volumes
	projectServiceVolumes, err := projectService.volumes(k.Project)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for _, volume := range projectServiceVolumes {

		// check if ro/rw mode is defined, default rw
		readonly := len(volume.Mode) > 0 && volume.Mode == "ro"

		if volume.VolumeName == "" {
			if useEmptyVolumes {
				volumeName = strings.Replace(volume.PVCName, "claim", "empty", 1)
			} else if useHostPath {
				volumeName = strings.Replace(volume.PVCName, "claim", "hostpath", 1)
			} else if useConfigMap {
				volumeName = strings.Replace(volume.PVCName, "claim", "cm", 1)
			} else {
				volumeName = volume.PVCName
			}
			count++
		} else {
			volumeName = volume.VolumeName
		}

		volMount := v1.VolumeMount{
			Name:      volumeName,
			ReadOnly:  readonly,
			MountPath: volume.Container,
		}

		// @ step get a volume source based on the type of volume we are using
		// For PVC we will also create a PVC object and add to list
		var volsource *v1.VolumeSource

		if useEmptyVolumes {
			log.DebugWithFields(log.Fields{
				"project-service": projectService.Name,
			}, "Use empty volume")

			volsource = k.configEmptyVolumeSource("volume")
		} else if useHostPath {
			log.DebugWithFields(log.Fields{
				"project-service": projectService.Name,
			}, "Use HostPath volume")

			source, err := k.configHostPathVolumeSource(volume.Host)
			if err != nil {
				log.Error("Couldn't create HostPath volume source")
				return nil, nil, nil, nil, err
			}
			volsource = source
		} else if useConfigMap {
			log.DebugWithFields(log.Fields{
				"project-service": projectService.Name,
			}, "Use configmap volume")

			cm, err := k.initConfigMapFromFileOrDir(projectService, volumeName, volume.Host)
			if err != nil {
				log.Error("Couldn't create ConfigMap volume source")
				return nil, nil, nil, nil, err
			}

			cms = append(cms, cm)
			volsource = k.configConfigMapVolumeSource(volumeName, volume.Container, cm)

			if useSubPathMount(cm) {
				volMount.SubPath = volsource.ConfigMap.Items[0].Path
			}

		} else {
			log.DebugWithFields(log.Fields{
				"project-service": projectService.Name,
			}, "Use PVC volume")

			volsource = k.configPVCVolumeSource(volumeName, readonly)

			if volume.VFrom == "" {
				createdPVC, err := k.createPVC(volume)

				if err != nil {
					log.Error("Couldn't create PVC volume source")
					return nil, nil, nil, nil, err
				}

				PVCs = append(PVCs, createdPVC)
			}

		}
		volumeMounts = append(volumeMounts, volMount)

		// @step create a new volume object using the volsource and add to list
		vol := v1.Volume{
			Name:         volumeName,
			VolumeSource: *volsource,
		}
		volumes = append(volumes, vol)

		if len(volume.Host) > 0 && (!useHostPath && !useConfigMap) {
			log.WarnWithFields(log.Fields{
				"project-service": projectService.Name,
				"host":            volume.Host,
			}, "Volume mount on the host isn't supported. Ignoring path on the host")
		}
	}

	return volumeMounts, volumes, PVCs, cms, nil
}

// configEmptyVolumeSource is a helper function to create an EmptyDir v1.VolumeSource
// either for Tmpfs or for emptyvolumes
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L894
func (k *Kubernetes) configEmptyVolumeSource(key string) *v1.VolumeSource {
	if key == "tmpfs" {
		return &v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{Medium: v1.StorageMediumMemory},
		}
	}

	return &v1.VolumeSource{
		EmptyDir: &v1.EmptyDirVolumeSource{},
	}
}

// configConfigMapVolumeSource config a configmap to use as volume source
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L911
func (k *Kubernetes) configConfigMapVolumeSource(cmName string, targetPath string, cm *v1.ConfigMap) *v1.VolumeSource {
	s := v1.ConfigMapVolumeSource{}
	s.Name = cmName

	if useSubPathMount(cm) {
		var keys []string
		for k := range cm.Data {
			keys = append(keys, k)
		}
		key := keys[0]
		_, p := path.Split(targetPath)
		s.Items = []v1.KeyToPath{
			{
				Key:  key,
				Path: p,
			},
		}
	}

	return &v1.VolumeSource{
		ConfigMap: &s,
	}
}

// configHostPathVolumeSource is a helper function to create a HostPath v1.VolumeSource
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L935
func (k *Kubernetes) configHostPathVolumeSource(path string) (*v1.VolumeSource, error) {
	dir, err := getComposeFileDir(k.Opt.InputFiles)
	if err != nil {
		return nil, err
	}

	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(dir, path)
	}

	return &v1.VolumeSource{
		HostPath: &v1.HostPathVolumeSource{Path: absPath},
	}, nil
}

// configPVCVolumeSource is helper function to create an v1.VolumeSource with a PVC
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L951
func (k *Kubernetes) configPVCVolumeSource(name string, readonly bool) *v1.VolumeSource {
	return &v1.VolumeSource{
		PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
			ClaimName: name,
			ReadOnly:  readonly,
		},
	}
}

// configEnvs returns a list of sorted kubernetes EnvVar objects mapping all project service environment variables
// NOTE: compose-go library preloads all environment variables defined in env_files (if any), and appends
// 		  them to the list of explicitly provided environment variables.
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L961
func (k *Kubernetes) configEnvs(projectService ProjectService) ([]v1.EnvVar, error) {
	envs := EnvSort{}
	envsWithDeps := []v1.EnvVar{}

	// @step load up the environment variables
	for k, v := range projectService.environment() {
		// @step for nil value we replace it with empty string
		if v == nil {
			temp := "" // *string cannot be initialized in one statement
			v = &temp
		}

		// @step generate EnvVar spec and handle special value reference cases for `secret`, `configmap`, `pod` field or `container` resource
		// e.g. `secret.my-secret-name.my-key`,
		// 		`config.my-config-name.config-key`,
		// 		`pod.metadata.namespace`,
		// 		`container.my-container-name.limits.cpu`,
		// if none of the special cases has been referenced by the env var value then it's going to be treated as literal value
		parts := strings.Split(*v, ".")
		switch parts[0] {
		case "secret":
			envs = append(envs, v1.EnvVar{
				Name: k,
				ValueFrom: &v1.EnvVarSource{
					SecretKeyRef: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{
							Name: parts[1],
						},
						Key: parts[2],
					},
				},
			})
		case "config":
			envs = append(envs, v1.EnvVar{
				Name: k,
				ValueFrom: &v1.EnvVarSource{
					ConfigMapKeyRef: &v1.ConfigMapKeySelector{
						LocalObjectReference: v1.LocalObjectReference{
							Name: parts[1],
						},
						Key: parts[2],
					},
				},
			})
		case "pod":
			// Selects a field of the pod
			// supported paths: metadata.name, metadata.namespace, metadata.labels, metadata.annotations,
			// 					spec.nodeName, spec.serviceAccountName, status.hostIP, status.podIP, status.podIPs.
			paths := []string{
				"metadata.name", "metadata.namespace", "metadata.labels", "metadata.annotations",
				"spec.nodeName", "spec.serviceAccountName", "status.hostIP", "status.podIP", "status.podIPs",
			}

			path := strings.Join(parts[1:], ".")

			if contains(paths, path) {
				envs = append(envs, v1.EnvVar{
					Name: k,
					ValueFrom: &v1.EnvVarSource{
						FieldRef: &v1.ObjectFieldSelector{
							FieldPath: path,
						},
					},
				})
			} else {
				log.WarnfWithFields(log.Fields{
					"project-service": projectService.Name,
					"env-var":         k,
					"path":            path,
				}, "Unsupported Pod field reference: %s", path)
			}
		case "container":
			// Selects a resource of the container. Only resources limits and requests are currently supported:
			// 		limits.cpu, limits.memory, limits.ephemeral-storage,
			//  	requests.cpu, requests.memory and requests.ephemeral-storage
			resources := []string{
				"limits.cpu", "limits.memory", "limits.ephemeral-storage",
				"requests.cpu", "requests.memory", "requests.ephemeral-storage",
			}
			resource := strings.Join(parts[2:], ".")

			if contains(resources, resource) {
				envs = append(envs, v1.EnvVar{
					Name: k,
					ValueFrom: &v1.EnvVarSource{
						ResourceFieldRef: &v1.ResourceFieldSelector{
							ContainerName: parts[1],
							Resource:      resource,
						},
					},
				})
			} else {
				log.WarnfWithFields(log.Fields{
					"project-service": projectService.Name,
					"env-var":         k,
					"container":       parts[1],
					"resource":        resource,
				}, "Unsupported Container resource reference: %s", resource)
			}
		default:
			if strings.Contains(*v, "{{") && strings.Contains(*v, "}}") {
				*v = strings.ReplaceAll(*v, "{{", "$(")
				*v = strings.ReplaceAll(*v, "}}", ")")

				envsWithDeps = append(envsWithDeps, v1.EnvVar{
					Name:  k,
					Value: *v,
				})
			} else {
				envs = append(envs, v1.EnvVar{
					Name:  k,
					Value: *v,
				})
			}
		}
	}

	// Stable sorts data while keeping the original order of equal elements
	// we need this because envs are not populated in any random order
	// this sorting ensures they are populated in a particular order
	sort.Stable(envs)

	// append dependent env variables at the end to ensure proper expansion in K8s
	for _, de := range envsWithDeps {
		envs = append(envs, de)
	}

	return envs, nil
}

// createKubernetesObjects generates a Kubernetes object for each input compose project service
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L1020
func (k *Kubernetes) createKubernetesObjects(projectService ProjectService) []runtime.Object {
	var objects []runtime.Object

	// @step get workload type
	workloadType := projectService.workloadType()

	// @step create ConfigMap objects for compose project service (external are not supported!)
	if len(projectService.Configs) > 0 {
		objects = k.createConfigMapFromComposeConfig(projectService, objects)
	}

	// @step create object based on inferred / manually configured workload controller type
	switch strings.ToLower(workloadType) {
	case strings.ToLower(config.DeploymentWorkload):
		o := k.initDeployment(projectService)
		objects = append(objects, o)
		hpa := k.initHpa(projectService, o)
		if hpa != nil {
			objects = append(objects, hpa)
		}
	case strings.ToLower(config.StatefulsetWorkload):
		o := k.initStatefulSet(projectService)
		objects = append(objects, o)
		hpa := k.initHpa(projectService, o)
		if hpa != nil {
			objects = append(objects, hpa)
		}
	case strings.ToLower(config.DaemonsetWorkload):
		objects = append(objects, k.initDaemonSet(projectService))
	}

	return objects
}

// createConfigMapFromComposeConfig will create ConfigMap objects for each non-external config
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L1078
func (k *Kubernetes) createConfigMapFromComposeConfig(projectService ProjectService, objects []runtime.Object) []runtime.Object {
	for _, config := range projectService.Configs {
		currentConfigName := config.Source
		currentConfigObj := k.Project.Configs[currentConfigName]

		if currentConfigObj.External.External {
			log.WarnWithFields(log.Fields{
				"project-service": projectService.Name,
				"config-name":     currentConfigName,
			}, "Your deployment expects configmap to exist in the target K8s cluster namespace.")
			log.Warn("Follow the official guidelines on how to create K8s configmap manually")
			log.Warn("https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/")

			continue
		}

		currentFileName := currentConfigObj.File
		configMap, err := k.initConfigMapFromFile(projectService, currentFileName)
		if err != nil {
			log.ErrorfWithFields(log.Fields{
				"project-service": projectService.Name,
				"config":          currentFileName,
			}, "Unable to initialise ConfigMap from file: %s", currentFileName)
		} else {
			objects = append(objects, configMap)
		}
	}
	return objects
}

// initPod initializes Kubernetes Pod object
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L1093
// @todo: UNUSED currently
func (k *Kubernetes) initPod(projectService ProjectService) *v1.Pod {
	pod := v1.Pod{
		TypeMeta: meta.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:        projectService.Name,
			Labels:      configLabels(projectService.Name),
			Annotations: configAnnotations(projectService),
		},
		Spec: k.initPodSpec(projectService),
	}

	return &pod
}

// createNetworkPolicy initializes Network policy
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L1109
func (k *Kubernetes) createNetworkPolicy(_, networkName string) (*networking.NetworkPolicy, error) {
	str := "true"

	np := &networking.NetworkPolicy{
		TypeMeta: meta.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name: networkName,
			//Labels: ConfigLabels(name),
		},
		Spec: networking.NetworkPolicySpec{
			PodSelector: meta.LabelSelector{
				MatchLabels: map[string]string{NetworkLabel + "/" + networkName: str},
			},
			Ingress: []networking.NetworkPolicyIngressRule{{
				From: []networking.NetworkPolicyPeer{{
					PodSelector: &meta.LabelSelector{
						MatchLabels: map[string]string{NetworkLabel + "/" + networkName: str},
					},
				}},
			}},
		},
	}

	return np, nil
}

// updateController updates the given object with the given pod template update function and ObjectMeta update function
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L1254
func (k *Kubernetes) updateController(obj runtime.Object, updateTemplate func(*v1.PodTemplateSpec) error, updateMeta func(meta *meta.ObjectMeta)) (err error) {
	switch t := obj.(type) {
	case *v1apps.Deployment:
		if err = updateTemplate(&t.Spec.Template); err != nil {
			log.Error("Unable to update Deployment template")
			return err
		}
		updateMeta(&t.ObjectMeta)
	case *v1apps.StatefulSet:
		if err = updateTemplate(&t.Spec.Template); err != nil {
			log.Error("Unable to update StatefulSet template")
			return err
		}
		updateMeta(&t.ObjectMeta)
	case *v1apps.DaemonSet:
		if err = updateTemplate(&t.Spec.Template); err != nil {
			log.Error("Unable to update DaemonSet template")
			return err
		}
		updateMeta(&t.ObjectMeta)
	case *v1batch.Job:
		if err = updateTemplate(&t.Spec.Template); err != nil {
			log.Error("Unable to update Job template")
			return err
		}
		updateMeta(&t.ObjectMeta)
	case *v1.Pod:
		p := v1.PodTemplateSpec{
			ObjectMeta: t.ObjectMeta,
			Spec:       t.Spec,
		}
		if err = updateTemplate(&p); err != nil {
			log.Error("Unable to update Pod template")
			return err
		}
		t.Spec = p.Spec
		t.ObjectMeta = p.ObjectMeta
	}

	return nil
}

// portsExist checks if service has ports defined (including ports defined by `expose`)
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L347
func (k *Kubernetes) portsExist(projectService ProjectService) bool {
	return len(projectService.ports()) != 0
}

// createService creates a k8s service
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L352
func (k *Kubernetes) createService(serviceType string, projectService ProjectService) *v1.Service {
	svc := k.initSvc(projectService)

	// @step configure the service ports.
	servicePorts := k.configServicePorts(serviceType, projectService)
	svc.Spec.Ports = servicePorts

	if strings.EqualFold(serviceType, config.HeadlessService) {
		svc.Spec.Type = v1.ServiceTypeClusterIP
		svc.Spec.ClusterIP = "None"
	} else {
		svc.Spec.Type = v1.ServiceType(serviceType)
	}

	svc.ObjectMeta.Annotations = configAnnotations(projectService)

	return svc
}

// createHeadlessService creates a k8s headless service.
// This is used for docker-compose services without ports. For such services we can't create regular Kubernetes Service.
// and without Service Pods can't find each other using DNS names.
// Instead of regular Kubernetes Service we create Headless Service. DNS of such service points directly to Pod IP address.
// You can find more about Headless Services in Kubernetes documentation https://kubernetes.io/docs/user-guide/services/#headless-services
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L378
func (k *Kubernetes) createHeadlessService(projectService ProjectService) *v1.Service {
	svc := k.initSvc(projectService)

	servicePorts := []v1.ServicePort{}
	// @step configure a dummy port: https://github.com/kubernetes/kubernetes/issues/32766.
	servicePorts = append(servicePorts, v1.ServicePort{
		Name: "headless",
		Port: 55555,
	})

	svc.Spec.Ports = servicePorts
	svc.Spec.ClusterIP = "None"

	svc.ObjectMeta.Annotations = configAnnotations(projectService)

	return svc
}

// updateKubernetesObjects updates k8s objects
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L399
func (k *Kubernetes) updateKubernetesObjects(projectService ProjectService, objects *[]runtime.Object) error {
	// @step configure the environment variables
	envs, err := k.configEnvs(projectService)
	if err != nil {
		log.Error("Unable to load env variables")
		return err
	}

	// @step configure the container volumes
	volumesMounts, volumes, pvcs, cms, err := k.configVolumes(projectService)
	if err != nil {
		log.Error("Unable to configure container volumes")
		return err
	}

	// @step configure Tmpfs
	if len(projectService.Tmpfs) > 0 {
		TmpVolumesMount, TmpVolumes := k.configTmpfs(projectService)
		volumes = append(volumes, TmpVolumes...)
		volumesMounts = append(volumesMounts, TmpVolumesMount...)
	}

	// @step add PVCs to objects
	// Looping on the slice pvcs instead of `*objects = append(*objects, pvcs...)`
	// because the type of objects and pvcs is different, but when doing append
	// one element at a time it gets converted to runtime.Object for objects slice
	for _, p := range pvcs {
		*objects = append(*objects, p)
	}

	// @step add ConfigMaps to objects
	for _, c := range cms {
		*objects = append(*objects, c)
	}

	// @step configure the container ports
	ports := k.configPorts(projectService)

	// @step configure capabilities
	capabilities := k.configCapabilities(projectService)

	// @step configure annotations
	annotations := configAnnotations(projectService)

	// @step fillTemplate function will fill the pod template with the values calculated from config
	fillTemplate := func(template *v1.PodTemplateSpec) error {
		if len(projectService.ContainerName) > 0 {
			template.Spec.Containers[0].Name = rfc1123dns(projectService.ContainerName)
		}
		template.Spec.Containers[0].Env = envs
		template.Spec.Containers[0].Command = projectService.Entrypoint
		template.Spec.Containers[0].Args = projectService.Command
		template.Spec.Containers[0].WorkingDir = projectService.WorkingDir
		template.Spec.Containers[0].VolumeMounts = append(template.Spec.Containers[0].VolumeMounts, volumesMounts...)
		template.Spec.Containers[0].Stdin = projectService.StdinOpen
		template.Spec.Containers[0].TTY = projectService.Tty
		template.Spec.Volumes = append(template.Spec.Volumes, volumes...)
		template.Spec.NodeSelector = projectService.placement()

		// @step configure the HealthCheck
		healthCheck, err := projectService.LivenessProbe()
		if err != nil {
			log.ErrorWithFields(log.Fields{
				"project-service": projectService.Name,
			}, "Healthcheck definition has errors")

			return err
		}
		if healthCheck != nil {
			template.Spec.Containers[0].LivenessProbe = healthCheck
		}

		// @step configure readiness probe
		// Note: This is not covered by the docker compose spec
		readinessProbe, err := projectService.ReadinessProbe()
		if err != nil {
			log.ErrorWithFields(log.Fields{
				"project-service": projectService.Name,
			}, "Readiness probe definition has errors")

			return err
		}
		if readinessProbe != nil {
			template.Spec.Containers[0].ReadinessProbe = readinessProbe
		}

		// @step configure pod termination grace priod
		if projectService.StopGracePeriod != nil && len(projectService.StopGracePeriod.String()) > 0 {
			sgp, err := durationStrToSecondsInt(projectService.StopGracePeriod.String())
			if err != nil {
				log.ErrorWithFields(log.Fields{
					"project-service": projectService.Name,
					"duration":        projectService.StopGracePeriod.String(),
				}, "Failed to parse pod termination grace period duration")

				return err
			}
			gracePeriod := int64(*sgp)
			template.Spec.TerminationGracePeriodSeconds = &gracePeriod
		}

		// @step configure pod resource requests and limits
		k.setPodResources(projectService, template)

		// @step configure pod security context
		podSecurityContext := &v1.PodSecurityContext{}
		k.setPodSecurityContext(projectService, podSecurityContext)

		// @step setup container security context
		securityContext := &v1.SecurityContext{}
		k.setSecurityContext(projectService, capabilities, securityContext)

		// @step update template only if container securityContext is not empty
		if *securityContext != (v1.SecurityContext{}) {
			template.Spec.Containers[0].SecurityContext = securityContext
		}

		// @step update template only if podSecurityContext is not empty
		if !reflect.DeepEqual(*podSecurityContext, v1.PodSecurityContext{}) {
			template.Spec.SecurityContext = podSecurityContext
		}

		// @step update ports
		template.Spec.Containers[0].Ports = ports

		// @step update labels
		template.ObjectMeta.Labels = configLabelsWithNetwork(projectService)

		// @step configure the image pull policy
		template.Spec.Containers[0].ImagePullPolicy = projectService.imagePullPolicy()

		// @step configure the container restart policy.
		template.Spec.RestartPolicy = projectService.restartPolicy()

		// @step configure hostname/domain_name settings
		if projectService.Hostname != "" {
			template.Spec.Hostname = projectService.Hostname
		}
		if projectService.DomainName != "" {
			template.Spec.Subdomain = projectService.DomainName
		}

		return nil
	}

	// @step fillObjectMeta fills the metadata with the value calculated from config
	fillObjectMeta := func(meta *meta.ObjectMeta) {
		meta.Annotations = annotations
	}

	// @step update supported k8s workload objects
	for _, obj := range *objects {
		if err = k.updateController(obj, fillTemplate, fillObjectMeta); err != nil {
			log.ErrorWithFields(log.Fields{
				"project-service": projectService.Name,
			}, "Couldn't update k8s object")

			return err
		}

		projectServiceVolumes, _ := projectService.volumes(k.Project)
		if len(projectServiceVolumes) > 0 {
			switch objType := obj.(type) {
			// @todo Check if applicable to other object types
			case *v1apps.Deployment:
				objType.Spec.Strategy.Type = v1apps.RecreateDeploymentStrategyType
			}
		}
	}

	return nil
}

// sortServicesFirst - sorts the objects so that services are first
// according to best practice kubernetes services should be created first
// http://kubernetes.io/docs/user-guide/config-best-practices/
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L661
func (k *Kubernetes) sortServicesFirst(objs *[]runtime.Object) {
	var svc, others, ret []runtime.Object

	for _, obj := range *objs {
		if obj.GetObjectKind().GroupVersionKind().Kind == "Service" {
			svc = append(svc, obj)
		} else {
			others = append(others, obj)
		}
	}
	ret = append(ret, svc...)
	ret = append(ret, others...)

	*objs = ret
}

// removeDupObjects removes duplicate objects...
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L679
func (k *Kubernetes) removeDupObjects(objs *[]runtime.Object) {
	var result []runtime.Object
	exist := map[string]bool{}

	for _, obj := range *objs {
		if us, ok := obj.(meta.Object); ok {
			k := obj.GetObjectKind().GroupVersionKind().String() + us.GetNamespace() + us.GetName()
			if exist[k] {
				log.DebugfWithFields(log.Fields{
					"configmap": us.GetName(),
				}, "Remove duplicate resource: %s/%s", obj.GetObjectKind().GroupVersionKind().Kind, us.GetName())

				continue
			} else {
				result = append(result, obj)
				exist[k] = true
			}
		} else {
			result = append(result, obj)
		}
	}

	*objs = result
}

// setPodResources configures pod resources
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L592
func (k *Kubernetes) setPodResources(projectService ProjectService, template *v1.PodTemplateSpec) {
	// @step resource limits
	memLimit, cpuLimit := projectService.resourceLimits()

	if *memLimit > 0 || *cpuLimit > 0 {
		resourceLimits := v1.ResourceList{}

		if *memLimit > 0 {
			resourceLimits[v1.ResourceMemory] = *resource.NewQuantity(*memLimit, resource.BinarySI)
		}

		if *cpuLimit > 0 {
			resourceLimits[v1.ResourceCPU] = *resource.NewMilliQuantity(*cpuLimit, resource.DecimalSI)
		}

		template.Spec.Containers[0].Resources.Limits = resourceLimits
	}

	// @step resource requests
	memRequest, cpuRequest := projectService.resourceRequests()

	if *memRequest > 0 || *cpuRequest > 0 {
		resourceRequests := v1.ResourceList{}

		if *memRequest > 0 {
			resourceRequests[v1.ResourceMemory] = *resource.NewQuantity(*memRequest, resource.BinarySI)
		}

		if *cpuRequest > 0 {
			resourceRequests[v1.ResourceCPU] = *resource.NewMilliQuantity(*cpuRequest, resource.DecimalSI)
		}

		template.Spec.Containers[0].Resources.Requests = resourceRequests
	}
}

// setPodSecurityContext sets a pod security context
func (k *Kubernetes) setPodSecurityContext(projectService ProjectService, podSecurityContext *v1.PodSecurityContext) {
	// @step set RunAsUser
	podSecurityContext.RunAsUser = projectService.runAsUser()

	// @step set RunAsGroup
	podSecurityContext.RunAsGroup = projectService.runAsGroup()

	// @step set FsGroup
	podSecurityContext.FSGroup = projectService.fsGroup()

	// @step set supplementalGroups
	if projectService.GroupAdd != nil {
		var groups []int64
		// map supplemental groups to int64 as this is what's expected in PSC
		for _, g := range projectService.GroupAdd {
			gid, err := strconv.ParseInt(g, 10, 64)
			if err != nil {
				log.WarnWithFields(log.Fields{
					"project-service":    projectService.Name,
					"supplemental-group": g,
				}, "Ignoring supplemental group as it's not numeric. Supplemental groups must be specified as a GID (numeric).")
			} else {
				groups = append(groups, gid)
			}
		}
		podSecurityContext.SupplementalGroups = groups
	}
}

// setSecurityContext sets container security context
func (k *Kubernetes) setSecurityContext(projectService ProjectService, capabilities *v1.Capabilities, securityContext *v1.SecurityContext) {
	// @step set Privileged
	if projectService.Privileged {
		securityContext.Privileged = &projectService.Privileged
	}

	// @step set RunAsUser
	if projectService.User != "" {
		uid, err := strconv.ParseInt(projectService.User, 10, 64)
		if err != nil {
			log.WarnWithFields(log.Fields{
				"project-service": projectService.Name,
				"user":            projectService.User,
			}, "Ignoring `user` directive value. User must be specified as a UID (numeric).")
		} else {
			securityContext.RunAsUser = &uid
		}
	}

	// @step set capabilities if specified
	if len(capabilities.Add) > 0 || len(capabilities.Drop) > 0 {
		securityContext.Capabilities = capabilities
	}
}
