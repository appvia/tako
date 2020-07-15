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

	"github.com/appvia/kube-devx/pkg/kev/config"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	v1apps "k8s.io/api/apps/v1"
	v1batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Kubernetes transformer
type Kubernetes struct {
	// the user provided options from the command line
	Opt ConvertOptions
}

var customConfig *config.Config

// Transform maps komposeObject to k8s objects
// returns object that are already sorted in the way that Services are first
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L1140
func (k *Kubernetes) Transform(komposeObject KomposeObject, opt ConvertOptions, envConfig *config.Config) ([]runtime.Object, error) {

	// @todo: make use of passed envConfig - this might be used to better control the outcome!
	customConfig = envConfig

	// this will hold all the converted data
	var allobjects []runtime.Object

	// @step Iterate over defined secrets and build Secret objects accordingly
	if komposeObject.Secrets != nil {
		secrets, err := k.CreateSecrets(komposeObject)
		if err != nil {
			return nil, errors.Wrapf(err, "Unable to create Secret resource")
		}
		for _, item := range secrets {
			allobjects = append(allobjects, item)
		}
	}

	// @step iterate over sorted service configs
	sortedKeys := SortedKeys(komposeObject)
	for _, name := range sortedKeys {
		service := komposeObject.ServiceConfigs[name]
		var objects []runtime.Object

		// @todo: We're not concerned about building & publishing images but will validate presence of image key for each service!
		// If there's no "image" key, use the name of the container that's built
		if service.Image == "" {
			service.Image = name
		}
		if service.Image == "" {
			return nil, fmt.Errorf("image key required within build parameters in order to build and push service '%s'", name)
		}

		// @step create kubernetes object (never create a pod in isolation)
		// https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-lifetime
		objects = k.CreateKubernetesObjects(name, service, opt)

		if k.PortsExist(service) {
			// Create a k8s service of a type defined by the service config
			svc := k.CreateService(name, service, objects)
			objects = append(objects, svc)

			// For exposed service also create an ingress
			if service.ExposeService != "" {
				objects = append(objects, k.initIngress(name, service, svc.Spec.Ports[0].Port))
			}
		} else {
			// No ports defined - createing headless service instead
			if service.ServiceType == "Headless" {
				svc := k.CreateHeadlessService(name, service, objects)
				objects = append(objects, svc)
			}
		}

		err := k.UpdateKubernetesObjects(name, service, opt, &objects)
		if err != nil {
			return nil, errors.Wrap(err, "Error transforming Kubernetes objects")
		}

		if len(service.Network) > 0 {

			for _, net := range service.Network {

				fmt.Printf("ℹ️  %v '%s' network detected and will be converted to equivalent NetworkPolicy\n", name, net)
				np, err := k.CreateNetworkPolicy(name, net)

				if err != nil {
					return nil, errors.Wrapf(err, "Unable to create Network Policy for network %v for service %v", net, name)
				}
				objects = append(objects, np)

			}

		}

		allobjects = append(allobjects, objects...)

	}

	// @step sort all object so Services are first, remove duplicates and fix worklaod versions
	k.SortServicesFirst(&allobjects)
	k.RemoveDupObjects(&allobjects)
	k.FixWorkloadVersion(&allobjects)

	return allobjects, nil
}

// InitPodSpec creates the pod specification
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L129
func (k *Kubernetes) InitPodSpec(name string, service ServiceConfig) v1.PodSpec {

	image := service.Image
	if image == "" {
		image = name
	}

	// @todo Prioritising kev config over kompose elements for now!
	pullSecret := ""
	if customConfig.Components[name].Workload.ImagePullSecret != "" {
		pullSecret = customConfig.Components[name].Workload.ImagePullSecret
	} else if customConfig.Workload.ImagePullSecret != "" {
		pullSecret = customConfig.Workload.ImagePullSecret
	} else if service.ImagePullSecret != "" {
		pullSecret = service.ImagePullSecret
	} else {
		pullSecret = config.DefaultImagePullSecret
	}

	pod := v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:  name,
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
	return pod
}

//InitPodSpecWithConfigMap creates the pod specification
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L154
func (k *Kubernetes) InitPodSpecWithConfigMap(name string, image string, service ServiceConfig) v1.PodSpec {
	var volumeMounts []v1.VolumeMount
	var volumes []v1.Volume

	for _, value := range service.Configs {
		cmVolName := FormatFileName(value.Source)
		target := value.Target
		if target == "" {
			// short syntax, = /<source>
			target = "/" + value.Source
		}
		subPath := filepath.Base(target)

		volSource := v1.ConfigMapVolumeSource{}
		volSource.Name = cmVolName
		key, err := service.GetConfigMapKeyFromMeta(value.Source)
		if err != nil {
			fmt.Printf("cannot parse config %s , %s", value.Source, err.Error())
			// mostly it's external
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

	pod := v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:         name,
				Image:        image,
				VolumeMounts: volumeMounts,
			},
		},
		Volumes: volumes,
	}
	return pod
}

// InitRC initializes Kubernetes ReplicationController object
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L216
func (k *Kubernetes) InitRC(name string, service ServiceConfig, replicas int) *v1.ReplicationController {

	repl := int32(replicas)

	rc := &v1.ReplicationController{
		TypeMeta: meta.TypeMeta{
			Kind:       "ReplicationController",
			APIVersion: "v1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:   name,
			Labels: ConfigLabels(name),
		},
		Spec: v1.ReplicationControllerSpec{
			Replicas: &repl,
			Template: &v1.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Labels: ConfigLabels(name),
				},
				Spec: k.InitPodSpec(name, service),
			},
		},
	}
	return rc
}

// InitSvc initializes Kubernetes Service object
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L240
func (k *Kubernetes) InitSvc(name string, service ServiceConfig) *v1.Service {
	svc := &v1.Service{
		TypeMeta: meta.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:   name,
			Labels: ConfigLabels(name),
		},
		Spec: v1.ServiceSpec{
			Selector: ConfigLabels(name),
		},
	}
	return svc
}

// InitConfigMapForEnv initializes a ConfigMap object
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L258
func (k *Kubernetes) InitConfigMapForEnv(name string, service ServiceConfig, opt ConvertOptions, envFile string) *v1.ConfigMap {

	envs, err := GetEnvsFromFile(envFile, opt)
	if err != nil {
		fmt.Printf("Unable to retrieve env file: %s", err)
	}

	// Remove root pathing
	// replace all other slashes / periods
	envName := FormatEnvName(envFile)

	// In order to differentiate files, we append to the name and remove '.env' if applicable from the file name
	configMap := &v1.ConfigMap{
		TypeMeta: meta.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:   envName,
			Labels: ConfigLabels(name + "-" + envName),
		},
		Data: envs,
	}

	return configMap
}

// IntiConfigMapFromFileOrDir will create a configmap from dir or file
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L288
func (k *Kubernetes) IntiConfigMapFromFileOrDir(name, cmName, filePath string, service ServiceConfig) (*v1.ConfigMap, error) {
	configMap := &v1.ConfigMap{
		TypeMeta: meta.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:   cmName,
			Labels: ConfigLabels(name),
		},
	}
	dataMap := make(map[string]string)

	fi, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	switch mode := fi.Mode(); {
	case mode.IsDir():
		files, err := ioutil.ReadDir(filePath)
		if err != nil {
			return nil, err
		}

		for _, file := range files {
			if !file.IsDir() {
				fmt.Printf("Read file to ConfigMap: %s", file.Name())
				data, err := GetContentFromFile(filePath + "/" + file.Name())
				if err != nil {
					return nil, err
				}
				dataMap[file.Name()] = data
			}
		}
		configMap.Data = dataMap

	case mode.IsRegular():
		// do file stuff
		configMap = k.InitConfigMapFromFile(name, service, filePath)
		configMap.Name = cmName
		configMap.Annotations = map[string]string{
			"use-subpath": "true",
		}
	}

	return configMap, nil
}

// useSubPathMount check if a configmap should be mounted as subpath
// in this situation, this configmap will only contains 1 key in data
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L339
func useSubPathMount(cm *v1.ConfigMap) bool {
	if cm.Annotations == nil {
		return false
	}
	if cm.Annotations["use-subpath"] != "true" {
		return false
	}
	return true
}

//InitConfigMapFromFile initializes a ConfigMap object
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L350
func (k *Kubernetes) InitConfigMapFromFile(name string, service ServiceConfig, fileName string) *v1.ConfigMap {
	content, err := GetContentFromFile(fileName)
	if err != nil {
		fmt.Printf("Unable to retrieve file: %s", err)
	}

	dataMap := make(map[string]string)
	dataMap[filepath.Base(fileName)] = content

	configMapName := ""
	for key, tmpConfig := range service.ConfigsMetaData {
		if tmpConfig.File == fileName {
			configMapName = key
		}
	}
	configMap := &v1.ConfigMap{
		TypeMeta: meta.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:   FormatFileName(configMapName),
			Labels: ConfigLabels(name),
		},
		Data: dataMap,
	}
	return configMap
}

// InitD initializes Kubernetes Deployment object
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L380
func (k *Kubernetes) InitD(name string, service ServiceConfig, replicas int) *v1beta1.Deployment {

	repl := int32(replicas)

	var podSpec v1.PodSpec
	if len(service.Configs) > 0 {
		podSpec = k.InitPodSpecWithConfigMap(name, service.Image, service)
	} else {
		podSpec = k.InitPodSpec(name, service)
	}

	dc := &v1beta1.Deployment{
		TypeMeta: meta.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:   name,
			Labels: ConfigAllLabels(name, &service),
		},
		Spec: v1beta1.DeploymentSpec{
			Replicas: &repl,
			Selector: &meta.LabelSelector{
				MatchLabels: ConfigLabels(name),
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Annotations: ConfigAnnotations(service),
				},
				Spec: podSpec,
			},
		},
	}
	dc.Spec.Template.Labels = ConfigLabels(name)

	// @step derives service update strategy and adds to the deployment configuration spec
	update := service.GetKubernetesUpdateStrategy()
	if update != nil {
		dc.Spec.Strategy = v1beta1.DeploymentStrategy{
			Type:          v1beta1.RollingUpdateDeploymentStrategyType,
			RollingUpdate: update,
		}
		fmt.Printf("Set deployment '%s' rolling update: MaxSurge: %s, MaxUnavailable: %s", name, update.MaxSurge.String(), update.MaxUnavailable.String())
	}

	return dc
}

// InitDS initializes Kubernetes DaemonSet object
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L427
func (k *Kubernetes) InitDS(name string, service ServiceConfig) *v1beta1.DaemonSet {
	ds := &v1beta1.DaemonSet{
		TypeMeta: meta.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:   name,
			Labels: ConfigAllLabels(name, &service),
		},
		Spec: v1beta1.DaemonSetSpec{
			Template: v1.PodTemplateSpec{
				Spec: k.InitPodSpec(name, service),
			},
		},
	}
	return ds
}

// InitSTS initialises a new StatefulSet
func (k *Kubernetes) InitSTS(name string, service ServiceConfig, replicas int) *v1apps.StatefulSet {

	repl := int32(replicas)

	var podSpec v1.PodSpec
	if len(service.Configs) > 0 {
		podSpec = k.InitPodSpecWithConfigMap(name, service.Image, service)
	} else {
		podSpec = k.InitPodSpec(name, service)
	}

	sts := &v1apps.StatefulSet{
		TypeMeta: meta.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:   name,
			Labels: ConfigAllLabels(name, &service),
		},
		Spec: v1apps.StatefulSetSpec{
			Replicas: &repl,
			Selector: &meta.LabelSelector{
				MatchLabels: ConfigLabels(name),
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Annotations: ConfigAnnotations(service),
				},
				Spec: podSpec,
			},
		},
	}
	sts.Spec.Template.Labels = ConfigLabels(name)

	// @step define service name responsible for governing the StatefulSet
	sts.Spec.ServiceName = name

	// @step derives service update strategy and adds to the deployment configuration spec
	update := &v1apps.RollingUpdateStatefulSetStrategy{}
	sts.Spec.UpdateStrategy = v1apps.StatefulSetUpdateStrategy{
		Type:          v1apps.RollingUpdateStatefulSetStrategyType,
		RollingUpdate: update,
	}

	return sts
}

// InitJ initialises a new Kubernetes Job
func (k *Kubernetes) InitJ(name string, service ServiceConfig, replicas int) *v1batch.Job {

	repl := int32(replicas)

	var podSpec v1.PodSpec
	if len(service.Configs) > 0 {
		podSpec = k.InitPodSpecWithConfigMap(name, service.Image, service)
	} else {
		podSpec = k.InitPodSpec(name, service)
	}

	j := &v1batch.Job{
		TypeMeta: meta.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:   name,
			Labels: ConfigAllLabels(name, &service),
		},
		Spec: v1batch.JobSpec{
			Parallelism: &repl,
			Completions: &repl,
			Selector: &meta.LabelSelector{
				MatchLabels: ConfigLabels(name),
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Annotations: ConfigAnnotations(service),
				},
				Spec: podSpec,
			},
		},
	}
	j.Spec.Template.Labels = ConfigLabels(name)

	return j
}

// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L446
func (k *Kubernetes) initIngress(name string, service ServiceConfig, port int32) *v1beta1.Ingress {

	hosts := regexp.MustCompile("[ ,]*,[ ,]*").Split(service.ExposeService, -1)

	ingress := &v1beta1.Ingress{
		TypeMeta: meta.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:        name,
			Labels:      ConfigLabels(name),
			Annotations: ConfigAnnotations(service),
		},
		Spec: v1beta1.IngressSpec{
			Rules: make([]v1beta1.IngressRule, len(hosts)),
		},
	}

	for i, host := range hosts {
		host, p := ParseIngressPath(host)
		ingress.Spec.Rules[i] = v1beta1.IngressRule{
			IngressRuleValue: v1beta1.IngressRuleValue{
				HTTP: &v1beta1.HTTPIngressRuleValue{
					Paths: []v1beta1.HTTPIngressPath{
						{
							Path: p,
							Backend: v1beta1.IngressBackend{
								ServiceName: name,
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

	if service.ExposeServiceTLS != "" {
		ingress.Spec.TLS = []v1beta1.IngressTLS{
			{
				Hosts:      hosts,
				SecretName: service.ExposeServiceTLS,
			},
		}
	}

	return ingress
}

// CreateSecrets create secrets
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L502
func (k *Kubernetes) CreateSecrets(komposeObject KomposeObject) ([]*v1.Secret, error) {
	var objects []*v1.Secret
	for name, config := range komposeObject.Secrets {
		if config.File != "" {
			dataString, err := GetContentFromFile(config.File)
			if err != nil {
				fmt.Println("unable to read secret from file: ", config.File)
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
					Labels: ConfigLabels(name),
				},
				Type: v1.SecretTypeOpaque,
				Data: map[string][]byte{name: data},
			}
			objects = append(objects, secret)
		} else {
			fmt.Printf("⚠️  Your deployment(s) expects '%s' secret to exist in the target K8s cluster namespace.\n", name)
			fmt.Println("   Follow the official guidelines on how to create K8s secrets manually")
			fmt.Println("   https://kubernetes.io/docs/tasks/inject-data-application/distribute-credentials-secure/")
		}
	}
	return objects, nil

}

// CreatePVC initializes PersistentVolumeClaim
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L534
func (k *Kubernetes) CreatePVC(name string, mode string, size string, selectorValue string) (*v1.PersistentVolumeClaim, error) {
	volSize, err := resource.ParseQuantity(size)
	if err != nil {
		return nil, errors.Wrap(err, "resource.ParseQuantity failed, Error parsing size")
	}

	pvc := &v1.PersistentVolumeClaim{
		TypeMeta: meta.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:   name,
			Labels: ConfigLabels(name),
		},
		Spec: v1.PersistentVolumeClaimSpec{
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: volSize,
				},
			},
		},
	}

	if len(selectorValue) > 0 {
		pvc.Spec.Selector = &meta.LabelSelector{
			MatchLabels: ConfigLabels(selectorValue),
		}
	}

	if mode == "ro" {
		pvc.Spec.AccessModes = []v1.PersistentVolumeAccessMode{v1.ReadOnlyMany}
	} else {
		pvc.Spec.AccessModes = []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}
	}
	return pvc, nil
}

// ConfigPorts configures the container ports.
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L573
func (k *Kubernetes) ConfigPorts(name string, service ServiceConfig) []v1.ContainerPort {
	ports := []v1.ContainerPort{}
	exist := map[string]bool{}
	for _, port := range service.Port {
		// temp use as an id
		if exist[string(port.ContainerPort)+string(port.Protocol)] {
			continue
		}
		// If the default is already TCP, no need to include it.
		if port.Protocol == v1.ProtocolTCP {
			ports = append(ports, v1.ContainerPort{
				ContainerPort: port.ContainerPort,
				HostIP:        port.HostIP,
			})
		} else {
			ports = append(ports, v1.ContainerPort{
				ContainerPort: port.ContainerPort,
				Protocol:      port.Protocol,
				HostIP:        port.HostIP,
			})
		}
		exist[string(port.ContainerPort)+string(port.Protocol)] = true

	}

	return ports
}

// ConfigServicePorts configure the container service ports.
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L602
func (k *Kubernetes) ConfigServicePorts(name string, service ServiceConfig) []v1.ServicePort {
	servicePorts := []v1.ServicePort{}
	seenPorts := make(map[int]struct{}, len(service.Port))

	var servicePort v1.ServicePort
	for _, port := range service.Port {
		if port.HostPort == 0 {
			port.HostPort = port.ContainerPort
		}

		var targetPort intstr.IntOrString
		targetPort.IntVal = port.ContainerPort
		targetPort.StrVal = strconv.Itoa(int(port.ContainerPort))

		// decide the name based on whether we saw this port before
		name := strconv.Itoa(int(port.HostPort))
		if _, ok := seenPorts[int(port.HostPort)]; ok {
			// https://github.com/kubernetes/kubernetes/issues/2995
			if service.ServiceType == string(v1.ServiceTypeLoadBalancer) {
				fmt.Printf("Service %s of type LoadBalancer cannot use TCP and UDP for the same port", name)
			}
			name = fmt.Sprintf("%s-%s", name, strings.ToLower(string(port.Protocol)))
		}

		servicePort = v1.ServicePort{
			Name:       name,
			Port:       port.HostPort,
			TargetPort: targetPort,
		}

		if service.ServiceType == string(v1.ServiceTypeNodePort) && service.NodePortPort != 0 {
			servicePort.NodePort = service.NodePortPort
		}

		// If the default is already TCP, no need to include it.
		if port.Protocol != v1.ProtocolTCP {
			servicePort.Protocol = port.Protocol
		}

		servicePorts = append(servicePorts, servicePort)
		seenPorts[int(port.HostPort)] = struct{}{}
	}
	return servicePorts
}

//ConfigCapabilities configure POSIX capabilities that can be added or removed to a container
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L648
func (k *Kubernetes) ConfigCapabilities(service ServiceConfig) *v1.Capabilities {
	capsAdd := []v1.Capability{}
	capsDrop := []v1.Capability{}
	for _, capAdd := range service.CapAdd {
		capsAdd = append(capsAdd, v1.Capability(capAdd))
	}
	for _, capDrop := range service.CapDrop {
		capsDrop = append(capsDrop, v1.Capability(capDrop))
	}
	return &v1.Capabilities{
		Add:  capsAdd,
		Drop: capsDrop,
	}
}

// ConfigTmpfs configure the tmpfs.
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L664
func (k *Kubernetes) ConfigTmpfs(name string, service ServiceConfig) ([]v1.VolumeMount, []v1.Volume) {
	//initializing volumemounts and volumes
	volumeMounts := []v1.VolumeMount{}
	volumes := []v1.Volume{}

	for index, volume := range service.TmpFs {
		//naming volumes if multiple tmpfs are provided
		volumeName := fmt.Sprintf("%s-tmpfs%d", name, index)
		volume = strings.Split(volume, ":")[0]
		// create a new volume mount object and append to list
		volMount := v1.VolumeMount{
			Name:      volumeName,
			MountPath: volume,
		}
		volumeMounts = append(volumeMounts, volMount)

		//create tmpfs specific empty volumes
		volSource := k.ConfigEmptyVolumeSource("tmpfs")

		// create a new volume object using the volsource and add to list
		vol := v1.Volume{
			Name:         volumeName,
			VolumeSource: *volSource,
		}
		volumes = append(volumes, vol)
	}
	return volumeMounts, volumes
}

// ConfigSecretVolumes config volumes from secret.
// Link: https://docs.docker.com/compose/compose-file/#secrets
// In kubernetes' Secret resource, it has a data structure like a map[string]bytes, every key will act like the file name
// when mount to a container. This is the part that missing in compose. So we will create a single key secret from compose
// config and the key's name will be the secret's name, it's value is the file content.
// compose'secret can only be mounted at `/run/secrets`, so we will hardcoded this.
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L699
func (k *Kubernetes) ConfigSecretVolumes(name string, service ServiceConfig) ([]v1.VolumeMount, []v1.Volume) {
	var volumeMounts []v1.VolumeMount
	var volumes []v1.Volume
	if len(service.Secrets) > 0 {
		for _, secretConfig := range service.Secrets {
			if secretConfig.UID != "" {
				fmt.Printf("Ignore pid in secrets for service: %s", name)
			}
			if secretConfig.GID != "" {
				fmt.Printf("Ignore gid in secrets for service: %s", name)
			}

			var itemPath string // should be the filename
			var mountPath = ""  // should be the directory
			// if is used the short-syntax
			if secretConfig.Target == "" {
				// the secret path (mountPath) should be inside the default directory /run/secrets
				mountPath = "/run/secrets/" + secretConfig.Source
				// the itemPath should be the source itself
				itemPath = secretConfig.Source
			} else {
				// if is the long-syntax, i should get the last part of path and consider it the filename
				pathSplitted := strings.Split(secretConfig.Target, "/")
				lastPart := pathSplitted[len(pathSplitted)-1]

				// if the filename (lastPart) and the target is the same
				if lastPart == secretConfig.Target {
					// the secret path should be the source (it need to be inside a directory and only the filename was given)
					mountPath = secretConfig.Source
				} else {
					// should then get the target without the filename (lastPart)
					mountPath = mountPath + strings.TrimSuffix(secretConfig.Target, "/"+lastPart) // menos ultima parte
				}

				// if the target isn't absolute path
				if strings.HasPrefix(secretConfig.Target, "/") == false {
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

// ConfigVolumes configure the container volumes.
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L774
func (k *Kubernetes) ConfigVolumes(name string, service ServiceConfig) ([]v1.VolumeMount, []v1.Volume, []*v1.PersistentVolumeClaim, []*v1.ConfigMap, error) {
	volumeMounts := []v1.VolumeMount{}
	volumes := []v1.Volume{}
	var PVCs []*v1.PersistentVolumeClaim
	var cms []*v1.ConfigMap
	var volumeName string

	// Set volumes configuration based on user preference, e.g. to use empty volumes
	// as opposed to persistent volumes and volume claims
	useEmptyVolumes := k.Opt.EmptyVols
	useHostPath := k.Opt.Volumes == "hostPath"
	useConfigMap := k.Opt.Volumes == "configMap"

	if k.Opt.Volumes == "emptyDir" {
		useEmptyVolumes = true
	}

	// config volumes from secret if present
	secretsVolumeMounts, secretsVolumes := k.ConfigSecretVolumes(name, service)
	volumeMounts = append(volumeMounts, secretsVolumeMounts...)
	volumes = append(volumes, secretsVolumes...)

	var count int
	//iterating over array of `Vols` struct as it contains all necessary information about volumes
	for _, volume := range service.Volumes {

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

		// Get a volume source based on the type of volume we are using
		// For PVC we will also create a PVC object and add to list
		var volsource *v1.VolumeSource

		if useEmptyVolumes {
			volsource = k.ConfigEmptyVolumeSource("volume")
		} else if useHostPath {
			source, err := k.ConfigHostPathVolumeSource(volume.Host)
			if err != nil {
				return nil, nil, nil, nil, errors.Wrap(err, "k.ConfigHostPathVolumeSource failed")
			}
			volsource = source
		} else if useConfigMap {
			fmt.Printf("Use configmap volume")

			if cm, err := k.IntiConfigMapFromFileOrDir(name, volumeName, volume.Host, service); err != nil {
				return nil, nil, nil, nil, err
			} else {
				cms = append(cms, cm)
				volsource = k.ConfigConfigMapVolumeSource(volumeName, volume.Container, cm)

				if useSubPathMount(cm) {
					volMount.SubPath = volsource.ConfigMap.Items[0].Path
				}
			}

		} else {
			volsource = k.ConfigPVCVolumeSource(volumeName, readonly)
			if volume.VFrom == "" {
				defaultSize := config.DefaultVolumeSize

				if len(volume.PVCSize) > 0 {
					defaultSize = volume.PVCSize
				} else {
					for key, value := range service.Labels {
						if key == "kompose.volume.size" {
							defaultSize = value
						}
					}
				}

				createdPVC, err := k.CreatePVC(volumeName, volume.Mode, defaultSize, volume.SelectorValue)

				if err != nil {
					return nil, nil, nil, nil, errors.Wrap(err, "k.CreatePVC failed")
				}

				PVCs = append(PVCs, createdPVC)
			}

		}
		volumeMounts = append(volumeMounts, volMount)

		// create a new volume object using the volsource and add to list
		vol := v1.Volume{
			Name:         volumeName,
			VolumeSource: *volsource,
		}
		volumes = append(volumes, vol)

		if len(volume.Host) > 0 && (!useHostPath && !useConfigMap) {
			fmt.Printf("Volume mount on the host %q isn't supported - ignoring path on the host", volume.Host)
		}

	}

	return volumeMounts, volumes, PVCs, cms, nil
}

// ConfigEmptyVolumeSource is helper function to create an EmptyDir v1.VolumeSource
//either for Tmpfs or for emptyvolumes
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L894
func (k *Kubernetes) ConfigEmptyVolumeSource(key string) *v1.VolumeSource {
	//if key is tmpfs
	if key == "tmpfs" {
		return &v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{Medium: v1.StorageMediumMemory},
		}

	}

	//if key is volume
	return &v1.VolumeSource{
		EmptyDir: &v1.EmptyDirVolumeSource{},
	}

}

// ConfigConfigMapVolumeSource config a configmap to use as volume source
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L911
func (k *Kubernetes) ConfigConfigMapVolumeSource(cmName string, targetPath string, cm *v1.ConfigMap) *v1.VolumeSource {
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

// ConfigHostPathVolumeSource is a helper function to create a HostPath v1.VolumeSource
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L935
func (k *Kubernetes) ConfigHostPathVolumeSource(path string) (*v1.VolumeSource, error) {
	dir, err := GetComposeFileDir(k.Opt.InputFiles)
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

// ConfigPVCVolumeSource is helper function to create an v1.VolumeSource with a PVC
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L951
func (k *Kubernetes) ConfigPVCVolumeSource(name string, readonly bool) *v1.VolumeSource {
	return &v1.VolumeSource{
		PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
			ClaimName: name,
			ReadOnly:  readonly,
		},
	}
}

// ConfigEnvs configures the environment variables.
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L961
func (k *Kubernetes) ConfigEnvs(name string, service ServiceConfig, opt ConvertOptions) ([]v1.EnvVar, error) {

	envs := EnvSort{}

	keysFromEnvFile := make(map[string]bool)

	// If there is an env_file, use ConfigMaps and ignore the environment variables
	// already specified

	if len(service.EnvFile) > 0 {

		// Load each env_file

		for _, file := range service.EnvFile {

			envName := FormatEnvName(file)

			// Load environment variables from file
			envLoad, err := GetEnvsFromFile(file, opt)
			if err != nil {
				return envs, errors.Wrap(err, "Unable to read env_file")
			}

			// Add configMapKeyRef to each environment variable
			for k := range envLoad {
				envs = append(envs, v1.EnvVar{
					Name: k,
					ValueFrom: &v1.EnvVarSource{
						ConfigMapKeyRef: &v1.ConfigMapKeySelector{
							LocalObjectReference: v1.LocalObjectReference{
								Name: envName,
							},
							Key: k,
						}},
				})
				keysFromEnvFile[k] = true
			}
		}
	}

	// Load up the environment variables
	for _, v := range service.Environment {
		if !keysFromEnvFile[v.Name] {
			// @step check whether env var value references secret or configmap e.g. `secret.my-secret-name.my-key`, `config.my-config-name.config-key`
			parts := strings.Split(v.Value, ".")
			if len(parts) == 3 {
				switch parts[0] {
				case "secret":
					envs = append(envs, v1.EnvVar{
						Name: v.Name,
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
						Name: v.Name,
						ValueFrom: &v1.EnvVarSource{
							ConfigMapKeyRef: &v1.ConfigMapKeySelector{
								LocalObjectReference: v1.LocalObjectReference{
									Name: parts[1],
								},
								Key: parts[2],
							},
						},
					})
				}
			} else {
				envs = append(envs, v1.EnvVar{
					Name:  v.Name,
					Value: v.Value,
				})
			}
		}

	}

	// Stable sorts data while keeping the original order of equal elements
	// we need this because envs are not populated in any random order
	// this sorting ensures they are populated in a particular order
	sort.Stable(envs)
	return envs, nil
}

// CreateKubernetesObjects generates a Kubernetes artifact for each input type service
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L1020
func (k *Kubernetes) CreateKubernetesObjects(name string, service ServiceConfig, opt ConvertOptions) []runtime.Object {
	var objects []runtime.Object
	var replica int

	// @step get number of replicas for service
	// @todo Prioritise kev config over kompose convert option for now!
	// @todo Remove opt.Replicas from convert options and cleanup
	replica = 1
	if customConfig.Components[name].Workload.Replicas != 0 {
		replica = int(customConfig.Components[name].Workload.Replicas)
	} else if customConfig.Workload.Replicas != 0 {
		replica = int(customConfig.Workload.Replicas)
	} else if opt.Replicas != 0 {
		replica = opt.Replicas
	}

	// @step Check whether compose service deploy mode is set to global (which indicates daemonset should be used!)
	wType := ""
	if customConfig.Components[name].Workload.Type != config.DaemonsetWorkload {
		wType = customConfig.Components[name].Workload.Type
	} else if customConfig.Workload.Type != config.DaemonsetWorkload {
		wType = customConfig.Workload.Type
	}

	if service.DeployMode == "global" && wType != "" {
		// compose service defined as global but kev configuration workload type is different than DaemonSet
		fmt.Printf("Compose service defined as 'global' should map to K8s DaemonSet. User override forces conversion to %s", wType)
	}

	// @step Resolve kompose.controller.type label
	if val, ok := service.Labels[LabelControllerType]; ok {
		if opt.Controller != "" {
			fmt.Printf("Use controller type %s for service %s", val, name)
		}
		opt.Controller = val
	}

	// @step Create ConfigMap objects for service (external are not supported!)
	if len(service.Configs) > 0 {
		objects = k.createConfigMapFromComposeConfig(name, opt, service, objects)
	}

	// @step Create object based on inferred / manually configured workload controller type
	// @todo Prioritise kev config over kompose label configuration for now!
	workloadType := ""
	if customConfig.Components[name].Workload.Type != "" {
		workloadType = customConfig.Components[name].Workload.Type
	} else if customConfig.Workload.Type != "" {
		workloadType = customConfig.Workload.Type
	} else {
		workloadType = opt.Controller
	}

	switch strings.ToLower(workloadType) {
	case strings.ToLower(config.DeploymentWorkload):
		objects = append(objects, k.InitD(name, service, replica))
	case strings.ToLower(config.DaemonsetWorkload):
		objects = append(objects, k.InitDS(name, service))
	case strings.ToLower(config.StatefulsetWorkload):
		objects = append(objects, k.InitSTS(name, service, replica))
	case "replicationcontroller":
		objects = append(objects, k.InitRC(name, service, replica))
	}

	// @step For service referencing Env_file(s) init a new ConfigMap
	if len(service.EnvFile) > 0 {
		for _, envFile := range service.EnvFile {
			configMap := k.InitConfigMapForEnv(name, service, opt, envFile)
			objects = append(objects, configMap)
		}
	}

	return objects
}

// createConfigMapFromComposeConfig will create ConfigMap objects for each non-external config
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L1078
func (k *Kubernetes) createConfigMapFromComposeConfig(name string, opt ConvertOptions, service ServiceConfig, objects []runtime.Object) []runtime.Object {
	for _, config := range service.Configs {
		currentConfigName := config.Source
		currentConfigObj := service.ConfigsMetaData[currentConfigName]
		if currentConfigObj.External.External {
			fmt.Printf("⚠️  Your deployment(s) expects '%s' configmap to exist in the target K8s cluster namespace.\n", name)
			fmt.Println("   Follow the official guidelines on how to create K8s configmap manually")
			fmt.Println("   https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/")
			continue
		}
		currentFileName := currentConfigObj.File
		configMap := k.InitConfigMapFromFile(name, service, currentFileName)
		objects = append(objects, configMap)
	}
	return objects
}

// InitPod initializes Kubernetes Pod object
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L1093
func (k *Kubernetes) InitPod(name string, service ServiceConfig) *v1.Pod {
	pod := v1.Pod{
		TypeMeta: meta.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:   name,
			Labels: ConfigLabels(name),
		},
		Spec: k.InitPodSpec(name, service),
	}
	return &pod
}

// CreateNetworkPolicy initializes Network policy
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L1109
func (k *Kubernetes) CreateNetworkPolicy(name string, networkName string) (*networking.NetworkPolicy, error) {

	str := "true"
	np := &networking.NetworkPolicy{
		TypeMeta: meta.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name: networkName,
			//Labels: ConfigLabels(name)(name),
		},
		Spec: networking.NetworkPolicySpec{
			PodSelector: meta.LabelSelector{
				MatchLabels: map[string]string{"io.kompose.network/" + networkName: str},
			},
			Ingress: []networking.NetworkPolicyIngressRule{{
				From: []networking.NetworkPolicyPeer{{
					PodSelector: &meta.LabelSelector{
						MatchLabels: map[string]string{"io.kompose.network/" + networkName: str},
					},
				}},
			}},
		},
	}

	return np, nil
}

// UpdateController updates the given object with the given pod template update function and ObjectMeta update function
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L1254
func (k *Kubernetes) UpdateController(obj runtime.Object, updateTemplate func(*v1.PodTemplateSpec) error, updateMeta func(meta *meta.ObjectMeta)) (err error) {
	switch t := obj.(type) {
	case *v1.ReplicationController:
		if t.Spec.Template == nil {
			t.Spec.Template = &v1.PodTemplateSpec{}
		}
		err = updateTemplate(t.Spec.Template)
		if err != nil {
			return errors.Wrap(err, "updateTemplate failed")
		}
		updateMeta(&t.ObjectMeta)
	case *v1beta1.Deployment:
		err = updateTemplate(&t.Spec.Template)
		if err != nil {
			return errors.Wrap(err, "updateTemplate failed")
		}
		updateMeta(&t.ObjectMeta)
	case *v1apps.StatefulSet:
		err = updateTemplate(&t.Spec.Template)
		if err != nil {
			return errors.Wrap(err, "updateTemplate failed")
		}
		updateMeta(&t.ObjectMeta)
	case *v1beta1.DaemonSet:
		err = updateTemplate(&t.Spec.Template)
		if err != nil {
			return errors.Wrap(err, "updateTemplate failed")
		}
		updateMeta(&t.ObjectMeta)
	case *v1batch.Job:
		err = updateTemplate(&t.Spec.Template)
		if err != nil {
			return errors.Wrap(err, "updateTemplate failed")
		}
		updateMeta(&t.ObjectMeta)
	case *v1.Pod:
		p := v1.PodTemplateSpec{
			ObjectMeta: t.ObjectMeta,
			Spec:       t.Spec,
		}
		err = updateTemplate(&p)
		if err != nil {
			return errors.Wrap(err, "updateTemplate failed")
		}
		t.Spec = p.Spec
		t.ObjectMeta = p.ObjectMeta
	}
	return nil
}

// PortsExist checks if service has ports defined
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L347
func (k *Kubernetes) PortsExist(service ServiceConfig) bool {
	return len(service.Port) != 0
}

// CreateService creates a k8s service
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L352
func (k *Kubernetes) CreateService(name string, service ServiceConfig, objects []runtime.Object) *v1.Service {
	svc := k.InitSvc(name, service)

	// Configure the service ports.
	servicePorts := k.ConfigServicePorts(name, service)
	svc.Spec.Ports = servicePorts

	// @step Get the actual service type by prioritising kev config over kompose labels
	// @todo Refactor when we figure out the configuration via labels
	component := customConfig.Components[name]
	serviceType := ""
	if component.Service.Type != "" {
		// prioritise app component service type
		serviceType = component.Service.Type
	} else if customConfig.Service.Type != "" {
		// fallback to app default service type
		serviceType = customConfig.Service.Type
	} else {
		// fallback to Kompose derived value
		serviceType = service.ServiceType
	}

	// @step set the service spec
	if serviceType == config.HeadlessService {
		svc.Spec.Type = v1.ServiceTypeClusterIP
		svc.Spec.ClusterIP = "None"
	} else {
		svc.Spec.Type = v1.ServiceType(serviceType)
	}

	// Configure annotations
	annotations := ConfigAnnotations(service)
	svc.ObjectMeta.Annotations = annotations

	return svc
}

// CreateHeadlessService creates a k8s headless service.
// This is used for docker-compose services without ports. For such services we can't create regular Kubernetes Service.
// and without Service Pods can't find each other using DNS names.
// Instead of regular Kubernetes Service we create Headless Service. DNS of such service points directly to Pod IP address.
// You can find more about Headless Services in Kubernetes documentation https://kubernetes.io/docs/user-guide/services/#headless-services
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L378
func (k *Kubernetes) CreateHeadlessService(name string, service ServiceConfig, objects []runtime.Object) *v1.Service {
	svc := k.InitSvc(name, service)

	servicePorts := []v1.ServicePort{}
	// Configure a dummy port: https://github.com/kubernetes/kubernetes/issues/32766.
	servicePorts = append(servicePorts, v1.ServicePort{
		Name: "headless",
		Port: 55555,
	})

	svc.Spec.Ports = servicePorts
	svc.Spec.ClusterIP = "None"

	// Configure annotations
	annotations := ConfigAnnotations(service)
	svc.ObjectMeta.Annotations = annotations

	return svc
}

// UpdateKubernetesObjects loads configurations to k8s objects
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L399
func (k *Kubernetes) UpdateKubernetesObjects(name string, service ServiceConfig, opt ConvertOptions, objects *[]runtime.Object) error {

	// Configure the environment variables.
	envs, err := k.ConfigEnvs(name, service, opt)
	if err != nil {
		return errors.Wrap(err, "Unable to load env variables")
	}

	// Configure the container volumes.
	volumesMount, volumes, pvc, cms, err := k.ConfigVolumes(name, service)
	if err != nil {
		return errors.Wrap(err, "k.ConfigVolumes failed")
	}
	// Configure Tmpfs
	if len(service.TmpFs) > 0 {
		TmpVolumesMount, TmpVolumes := k.ConfigTmpfs(name, service)

		volumes = append(volumes, TmpVolumes...)

		volumesMount = append(volumesMount, TmpVolumesMount...)

	}

	if pvc != nil {
		// Looping on the slice pvc instead of `*objects = append(*objects, pvc...)`
		// because the type of objects and pvc is different, but when doing append
		// one element at a time it gets converted to runtime.Object for objects slice
		for _, p := range pvc {
			*objects = append(*objects, p)
		}
	}

	if cms != nil {
		for _, c := range cms {
			*objects = append(*objects, c)
		}
	}

	// Configure the container ports.
	ports := k.ConfigPorts(name, service)

	// Configure capabilities
	capabilities := k.ConfigCapabilities(service)

	// Configure annotations
	annotations := ConfigAnnotations(service)

	// fillTemplate fills the pod template with the value calculated from config
	fillTemplate := func(template *v1.PodTemplateSpec) error {
		if len(service.ContainerName) > 0 {
			template.Spec.Containers[0].Name = FormatContainerName(service.ContainerName)
		}
		template.Spec.Containers[0].Env = envs
		template.Spec.Containers[0].Command = service.Command
		template.Spec.Containers[0].Args = service.Args
		template.Spec.Containers[0].WorkingDir = service.WorkingDir
		template.Spec.Containers[0].VolumeMounts = append(template.Spec.Containers[0].VolumeMounts, volumesMount...)
		template.Spec.Containers[0].Stdin = service.Stdin
		template.Spec.Containers[0].TTY = service.Tty
		template.Spec.Volumes = append(template.Spec.Volumes, volumes...)
		template.Spec.NodeSelector = service.Placement
		// Configure the HealthCheck
		// We check to see if it's blank
		if !reflect.DeepEqual(service.HealthChecks, HealthCheck{}) {
			probe := v1.Probe{}

			if len(service.HealthChecks.Test) > 0 {
				probe.Handler = v1.Handler{
					Exec: &v1.ExecAction{
						Command: service.HealthChecks.Test,
					},
				}
			} else {
				return errors.New("Health check must contain a command")
			}

			probe.TimeoutSeconds = service.HealthChecks.Timeout
			probe.PeriodSeconds = service.HealthChecks.Interval
			probe.FailureThreshold = service.HealthChecks.Retries

			// See issue: https://github.com/docker/cli/issues/116
			// StartPeriod has been added to docker/cli however, it is not yet added
			// to compose. Once the feature has been implemented, this will automatically work
			probe.InitialDelaySeconds = service.HealthChecks.StartPeriod

			template.Spec.Containers[0].LivenessProbe = &probe
		}

		if service.StopGracePeriod != "" {
			template.Spec.TerminationGracePeriodSeconds, err = DurationStrToSecondsInt(service.StopGracePeriod)
			if err != nil {
				fmt.Printf("Failed to parse duration \"%v\" for service \"%v\"", service.StopGracePeriod, name)
			}
		}

		TranslatePodResource(&service, template)

		// Configure resource reservations
		podSecurityContext := &v1.PodSecurityContext{}

		// @todo: Is it even relevant... Check and cleanup!
		// //set pid namespace mode
		// if service.Pid != "" {
		// 	if service.Pid == "host" {
		// 		podSecurityContext.HostPID = true
		// 	} else {
		// 		fmt.Sprintf("Ignoring PID key for service \"%v\". Invalid value \"%v\".", name, service.Pid)
		// 	}
		// }

		//set supplementalGroups
		if service.GroupAdd != nil {
			podSecurityContext.SupplementalGroups = service.GroupAdd
		}

		// Setup security context
		securityContext := &v1.SecurityContext{}
		if service.Privileged {
			securityContext.Privileged = &service.Privileged
		}
		if service.User != "" {
			uid, err := strconv.ParseInt(service.User, 10, 64)
			if err != nil {
				fmt.Printf("Ignoring user directive. User to be specified as a UID (numeric).")
			} else {
				securityContext.RunAsUser = &uid
			}

		}
		// @todo: Add other elements to podSecurityContext / conteiner securityContext such as RunAsUser, RunAsGroup, FsGroup...!

		//set capabilities if it is not empty
		if len(capabilities.Add) > 0 || len(capabilities.Drop) > 0 {
			securityContext.Capabilities = capabilities
		}

		// update template only if securityContext is not empty
		if *securityContext != (v1.SecurityContext{}) {
			template.Spec.Containers[0].SecurityContext = securityContext
		}
		if !reflect.DeepEqual(*podSecurityContext, v1.PodSecurityContext{}) {
			template.Spec.SecurityContext = podSecurityContext
		}
		template.Spec.Containers[0].Ports = ports
		template.ObjectMeta.Labels = ConfigLabelsWithNetwork(name, service.Network)

		// Configure the image pull policy
		// @todo Prioritise kev configuration over kompose derived value for now!
		ipPol := ""
		if customConfig.Components[name].Workload.ImagePullPolicy != "" {
			ipPol = customConfig.Components[name].Workload.ImagePullPolicy
		} else if customConfig.Workload.ImagePullPolicy != "" {
			ipPol = customConfig.Workload.ImagePullPolicy
		} else {
			policy, err := GetImagePullPolicy(name, service.ImagePullPolicy)
			if err != nil {
				// Value derived by kompose is invalid. Default to "Always".
				ipPol = config.DefaultImagePullPolicy
			} else {
				ipPol = string(policy)
			}
		}
		template.Spec.Containers[0].ImagePullPolicy = v1.PullPolicy(ipPol)

		// Configure the container restart policy.
		// @todo Prioritise kev configuration over kompose derived value for now!
		rPol := ""
		if customConfig.Components[name].Workload.Restart != "" {
			rPol = customConfig.Components[name].Workload.Restart
		} else if customConfig.Workload.Restart != "" {
			rPol = customConfig.Workload.Restart
		} else {
			restart, err := GetRestartPolicy(name, service.Restart)
			if err != nil {
				// Value derived by kompose is invalid. Default to "Always".
				rPol = config.RestartPolicyAlways
			} else {
				rPol = string(restart)
			}
		}
		template.Spec.RestartPolicy = v1.RestartPolicy(rPol)

		// Configure hostname/domain_name settings
		if service.HostName != "" {
			template.Spec.Hostname = service.HostName
		}
		if service.DomainName != "" {
			template.Spec.Subdomain = service.DomainName
		}

		return nil
	}

	// fillObjectMeta fills the metadata with the value calculated from config
	fillObjectMeta := func(meta *meta.ObjectMeta) {
		meta.Annotations = annotations
	}

	// update supported controller
	for _, obj := range *objects {
		err = k.UpdateController(obj, fillTemplate, fillObjectMeta)
		if err != nil {
			return errors.Wrap(err, "k.UpdateController failed")
		}
		if len(service.Volumes) > 0 {
			switch objType := obj.(type) {
			case *v1beta1.Deployment:
				objType.Spec.Strategy.Type = v1beta1.RecreateDeploymentStrategyType
			}
		}
	}
	return nil
}

// SortServicesFirst - the objects that we get can be in any order this keeps services first
// according to best practice kubernetes services should be created first
// http://kubernetes.io/docs/user-guide/config-best-practices/
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L661
func (k *Kubernetes) SortServicesFirst(objs *[]runtime.Object) {
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

// RemoveDupObjects remove objects that are dups...eg. configmaps from env.
// since we know for sure that the duplication can only happends on ConfigMap, so
// this code will looks like this for now.
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L679
func (k *Kubernetes) RemoveDupObjects(objs *[]runtime.Object) {
	var result []runtime.Object
	exist := map[string]bool{}
	for _, obj := range *objs {
		if us, ok := obj.(*v1.ConfigMap); ok {
			k := us.GroupVersionKind().String() + us.GetNamespace() + us.GetName()
			if exist[k] {
				fmt.Printf("Remove duplicate configmap: %s", us.GetName())
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

// FixWorkloadVersion force reset deployment/daemonset's apiversion to apps/v1
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L717
func (k *Kubernetes) FixWorkloadVersion(objs *[]runtime.Object) {
	var result []runtime.Object
	for _, obj := range *objs {
		if d, ok := obj.(*v1beta1.Deployment); ok {
			nd := resetWorkloadAPIVersion(d)
			result = append(result, nd)
		} else if d, ok := obj.(*v1beta1.DaemonSet); ok {
			nd := resetWorkloadAPIVersion(d)
			result = append(result, nd)
		} else {
			result = append(result, obj)
		}
	}
	*objs = result
}
