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

// This file includes helper functions necessary to load, process, extract
// and map information contained in Docker Compose file into interim data structure
// which serves as an input to the Kubernetes converter.
// Note: Some functionality below has been extracted from the Kompose project
// and link to original Kompose code have been added for reference.

package kubernetes

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	compose "github.com/appvia/kube-devx/pkg/kev/compose"
	"github.com/appvia/kube-devx/pkg/kev/config"
	composego "github.com/compose-spec/compose-go/types"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	v1 "k8s.io/api/core/v1"
)

const (
	// LabelServiceType defines the type of service to be created
	LabelServiceType = "kompose.service.type"
	// LabelNodePortPort defines the port value for NodePort service
	LabelNodePortPort = "kompose.service.nodeport.port"
	// LabelServiceExpose defines if the service needs to be made accessible from outside the cluster or not
	LabelServiceExpose = "kompose.service.expose"
	// LabelServiceExposeTLSSecret  provides the name of the TLS secret to use with the Kubernetes ingress controller
	LabelServiceExposeTLSSecret = "kompose.service.expose.tls-secret"
	// LabelControllerType defines the type of controller to be created
	LabelControllerType = "kompose.controller.type"
	// LabelImagePullSecret defines a secret name for kubernetes ImagePullSecrets
	LabelImagePullSecret = "kompose.image-pull-secret"
	// LabelImagePullPolicy defines Kubernetes PodSpec imagePullPolicy.
	LabelImagePullPolicy = "kompose.image-pull-policy"
	// LabelVolumeSize defines persistent volume size
	LabelVolumeSize = "kompose.volume.size"
	// LabelVolumeSelector defines persistent volume selector
	LabelVolumeSelector = "kompose.volume.selector"
)

// LoadCompose loads a docker-compose file into KomposeObject
func LoadCompose(file string) (KomposeObject, error) {
	// Load compose project
	project, err := compose.LoadProject([]string{file})
	if err != nil {
		return KomposeObject{}, err
	}

	// parse and map to KomposeObject
	komposeObject, err := dockerComposeToKomposeMapping(project)
	if err != nil {
		return KomposeObject{}, err
	}

	return komposeObject, nil
}

// dockerComposeToKomposeMapping maps docker composego into interim KomposeObject representation
// @todo: As we already extract a bunch of information from the Compose we could potentially
// fall back onto that info instead of processing it again?
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v3.go#L277
func dockerComposeToKomposeMapping(composeObject *composego.Project) (KomposeObject, error) {

	// @step initialise object
	komposeObject := KomposeObject{
		ServiceConfigs: make(map[string]ServiceConfig),
		LoadedFrom:     "composego",
		Secrets:        composeObject.Secrets,
	}

	// @step parse composego object and convert it to KomposeObject
	for _, composeServiceConfig := range composeObject.Services {

		// @step initiate service config
		name := composeServiceConfig.Name

		serviceConfig := ServiceConfig{
			Image:         composeServiceConfig.Image,
			WorkingDir:    composeServiceConfig.WorkingDir,
			Annotations:   map[string]string(composeServiceConfig.Labels),
			CapAdd:        composeServiceConfig.CapAdd,
			CapDrop:       composeServiceConfig.CapDrop,
			Expose:        composeServiceConfig.Expose,
			Privileged:    composeServiceConfig.Privileged,
			User:          composeServiceConfig.User,
			Stdin:         composeServiceConfig.StdinOpen,
			Tty:           composeServiceConfig.Tty,
			TmpFs:         composeServiceConfig.Tmpfs,
			ContainerName: normalizeContainerNames(composeServiceConfig.ContainerName),
			Command:       composeServiceConfig.Entrypoint,
			Args:          composeServiceConfig.Command,
			Labels:        composeServiceConfig.Labels,
			HostName:      composeServiceConfig.Hostname,
			DomainName:    composeServiceConfig.DomainName,
			Secrets:       composeServiceConfig.Secrets,
		}

		// @step network
		parseNetwork(&composeServiceConfig, &serviceConfig, composeObject)

		// @step resources
		if err := parseResources(&composeServiceConfig, &serviceConfig); err != nil {
			return KomposeObject{}, err
		}

		// @step Deploy mode and labels
		if composeServiceConfig.Deploy != nil {
			serviceConfig.DeployMode = composeServiceConfig.Deploy.Mode
			serviceConfig.DeployLabels = composeServiceConfig.Deploy.Labels
		}

		// @step HealthCheck
		if composeServiceConfig.HealthCheck != nil && !composeServiceConfig.HealthCheck.Disable {
			var err error
			serviceConfig.HealthChecks, err = parseHealthCheck(*composeServiceConfig.HealthCheck)
			if err != nil {
				return KomposeObject{}, errors.Wrap(err, "Unable to parse health check")
			}
		}

		// @step restart policy
		// restart-policy: deploy.restart_policy.condition will rewrite restart option
		// see: https://docs.docker.com/compose/compose-file/#restart_policy
		serviceConfig.Restart = composeServiceConfig.Restart
		if composeServiceConfig.Deploy != nil && composeServiceConfig.Deploy.RestartPolicy != nil {
			serviceConfig.Restart = composeServiceConfig.Deploy.RestartPolicy.Condition
		}
		if serviceConfig.Restart == "unless-stopped" {
			fmt.Printf("Restart policy 'unless-stopped' in service %s is not supported, converting it to 'always'", name)
			serviceConfig.Restart = "always"
		}

		// @step replicas
		if composeServiceConfig.Deploy != nil && composeServiceConfig.Deploy.Replicas != nil {
			serviceConfig.Replicas = int(*composeServiceConfig.Deploy.Replicas)
		}

		// @step placement
		if composeServiceConfig.Deploy != nil && composeServiceConfig.Deploy.Placement.Constraints != nil {
			serviceConfig.Placement = loadPlacement(composeServiceConfig.Deploy.Placement.Constraints)
		}

		// @step update strategy
		if composeServiceConfig.Deploy != nil && composeServiceConfig.Deploy.UpdateConfig != nil {
			serviceConfig.DeployUpdateConfig = *composeServiceConfig.Deploy.UpdateConfig
		}

		// TODO: Build is not yet supported, see:
		// https://github.com/docker/cli/blob/master/cli/compose/types/types.go#L9
		// We will have to *manually* add this / parse.
		if composeServiceConfig.Build != nil {
			serviceConfig.Build = composeServiceConfig.Build.Context
			serviceConfig.Dockerfile = composeServiceConfig.Build.Dockerfile
			serviceConfig.BuildArgs = composeServiceConfig.Build.Args
			serviceConfig.BuildLabels = composeServiceConfig.Build.Labels
		}

		// @step environment
		parseEnvironment(&composeServiceConfig, &serviceConfig)

		// @step envrionment from env_file
		serviceConfig.EnvFile = composeServiceConfig.EnvFile

		// @step ports
		// composego v3.2+ uses a new "long syntax" format
		// https://docs.docker.com/compose/compose-file/#ports
		// here we will translate `expose` too, they basically means the same thing in kubernetes
		serviceConfig.Port = loadPorts(composeServiceConfig.Ports, serviceConfig.Expose)

		// @step volumes
		// composego v3 uses "long syntax" format for volumes
		// https://docs.docker.com/compose/compose-file/#long-syntax-3
		serviceConfig.VolList = loadVolumes(composeServiceConfig.Volumes)

		// @step kompose in-cluster-wordpress
		if err := parseKomposeLabels(composeServiceConfig.Labels, &serviceConfig); err != nil {
			return KomposeObject{}, err
		}

		// @step log if the service name has been normalised
		if normalizeServiceNames(name) != name {
			fmt.Printf("Service name in docker-compose has been changed from %q to %q", name, normalizeServiceNames(name))
		}

		// @step configs
		serviceConfig.Configs = composeServiceConfig.Configs
		serviceConfig.ConfigsMetaData = composeObject.Configs

		// @step service type
		if composeServiceConfig.Deploy != nil && composeServiceConfig.Deploy.EndpointMode == "vip" {
			serviceConfig.ServiceType = config.NodePortService
		}

		// @step add service to object
		komposeObject.ServiceConfigs[normalizeServiceNames(name)] = serviceConfig
	}

	// @step handle volumes
	handleVolume(&komposeObject, &composeObject.Volumes)

	return komposeObject, nil
}

// normalizeContainerNames normalises container name
// @orig: https://github.com/kubernetes/kompose/blob/1f0a097836fb4e0ae4a802eb7ab543a4f9493727/pkg/loader/compose/utils.go#L123
func normalizeContainerNames(svcName string) string {
	return strings.ToLower(svcName)
}

// normalizeServiceNames normalises service name
// @orig: https://github.com/kubernetes/kompose/blob/1f0a097836fb4e0ae4a802eb7ab543a4f9493727/pkg/loader/compose/utils.go#L127
func normalizeServiceNames(svcName string) string {
	re := regexp.MustCompile("[._]")
	return strings.ToLower(re.ReplaceAllString(svcName, "-"))
}

// normalizeVolumes normalises volume name
// @orig: https://github.com/kubernetes/kompose/blob/1f0a097836fb4e0ae4a802eb7ab543a4f9493727/pkg/loader/compose/utils.go#L132
func normalizeVolumes(svcName string) string {
	return strings.Replace(svcName, "_", "-", -1)
}

// parseNetwork parses composego networks
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v3.go#L407
func parseNetwork(composeServiceConfig *composego.ServiceConfig, serviceConfig *ServiceConfig, composeObject *composego.Project) {
	if len(composeServiceConfig.Networks) == 0 {
		defaultNetwork, _ := composeObject.Networks["default"]
		if defaultNetwork.Name != "" {
			serviceConfig.Network = append(serviceConfig.Network, defaultNetwork.Name)
		}
	} else {
		var alias = ""
		for key := range composeServiceConfig.Networks {
			alias = key
			netName := composeObject.Networks[alias].Name
			// if Network Name Field is empty in the docker-composego definition
			// we will use the alias name defined in service config file
			if netName == "" {
				netName = alias
			}
			serviceConfig.Network = append(serviceConfig.Network, netName)
		}
	}
}

// parseResources parses composego resource requests & limits
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v3.go#L427
func parseResources(composeServiceConfig *composego.ServiceConfig, serviceConfig *ServiceConfig) error {
	deploy := composeServiceConfig.Deploy

	if deploy != nil {

		// memory:
		// TODO: Refactor yaml.MemStringorInt in kobject.go to int64
		// cpu:
		// convert to k8s format, for example: 0.5 = 500m
		// See: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
		// "The expression 0.1 is equivalent to the expression 100m, which can be read as “one hundred millicpu”."

		if deploy.Resources.Limits != nil {
			serviceConfig.MemLimit = MemStringorInt(composeServiceConfig.Deploy.Resources.Limits.MemoryBytes)

			if composeServiceConfig.Deploy.Resources.Limits.NanoCPUs != "" {
				cpuLimit, err := strconv.ParseFloat(composeServiceConfig.Deploy.Resources.Limits.NanoCPUs, 64)
				if err != nil {
					return errors.Wrap(err, "Unable to convert cpu limits resources value")
				}
				serviceConfig.CPULimit = int64(cpuLimit * 1000)
			}
		}
		if deploy.Resources.Reservations != nil {
			serviceConfig.MemReservation = MemStringorInt(composeServiceConfig.Deploy.Resources.Reservations.MemoryBytes)

			if composeServiceConfig.Deploy.Resources.Reservations.NanoCPUs != "" {
				cpuReservation, err := strconv.ParseFloat(composeServiceConfig.Deploy.Resources.Reservations.NanoCPUs, 64)
				if err != nil {
					return errors.Wrap(err, "Unable to convert cpu limits reservation value")
				}
				serviceConfig.CPUReservation = int64(cpuReservation * 1000)
			}
		}
	}
	return nil

}

// parseHealthCheck extracts healthcheck information to Kubernetes supported format
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v3.go#L234
func parseHealthCheck(composeHealthCheck composego.HealthCheckConfig) (HealthCheck, error) {

	var timeout, interval, retries, startPeriod int32

	// Here we convert the timeout from 1h30s (example) to 36030 seconds.
	if composeHealthCheck.Timeout != nil {
		parse, err := time.ParseDuration(composeHealthCheck.Timeout.String())
		if err != nil {
			return HealthCheck{}, errors.Wrap(err, "unable to parse health check timeout variable")
		}
		timeout = int32(parse.Seconds())
	}

	if composeHealthCheck.Interval != nil {
		parse, err := time.ParseDuration(composeHealthCheck.Interval.String())
		if err != nil {
			return HealthCheck{}, errors.Wrap(err, "unable to parse health check interval variable")
		}
		interval = int32(parse.Seconds())
	}

	if composeHealthCheck.Retries != nil {
		retries = int32(*composeHealthCheck.Retries)
	}

	if composeHealthCheck.StartPeriod != nil {
		parse, err := time.ParseDuration(composeHealthCheck.StartPeriod.String())
		if err != nil {
			return HealthCheck{}, errors.Wrap(err, "unable to parse health check startPeriod variable")
		}
		startPeriod = int32(parse.Seconds())
	}

	// Due to docker/cli adding "CMD-SHELL" to the struct, we remove the first element of composeHealthCheck.Test
	return HealthCheck{
		Test:        composeHealthCheck.Test[1:],
		Timeout:     timeout,
		Interval:    interval,
		Retries:     retries,
		StartPeriod: startPeriod,
	}, nil
}

// loadPlacement parses placement information from composego.
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v3.go#L136
func loadPlacement(constraints []string) map[string]string {
	placement := make(map[string]string)
	errMsg := "constraints in placement is not supported, only 'node.hostname', 'node.role == worker', 'node.role == manager', 'engine.in-cluster-wordpress.operatingsystem' and 'node.in-cluster-wordpress.xxx' (ex: node.in-cluster-wordpress.something == anything) is supported as a constraint"
	for _, j := range constraints {
		p := strings.Split(j, " == ")
		if len(p) < 2 {
			fmt.Println(p[0], errMsg)
			continue
		}
		if p[0] == "node.role" && p[1] == "worker" {
			placement["node-role.kubernetes.io/worker"] = "true"
		} else if p[0] == "node.role" && p[1] == "manager" {
			placement["node-role.kubernetes.io/master"] = "true"
		} else if p[0] == "node.hostname" {
			placement["kubernetes.io/hostname"] = p[1]
		} else if p[0] == "engine.in-cluster-wordpress.operatingsystem" {
			placement["beta.kubernetes.io/os"] = p[1]
		} else if strings.HasPrefix(p[0], "node.in-cluster-wordpress.") {
			label := strings.TrimPrefix(p[0], "node.in-cluster-wordpress.")
			placement[label] = p[1]
		} else {
			fmt.Println(p[0], errMsg)
		}
	}
	return placement
}

// parseEnvironment parses composego service environment
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v3.go#L465
func parseEnvironment(composeServiceConfig *composego.ServiceConfig, serviceConfig *ServiceConfig) {
	// DockerCompose uses map[string]*string while we use []string
	// So let's convert that using this hack
	// Note: unset env pick up the env value on host if exist
	for name, value := range composeServiceConfig.Environment {
		var env EnvVar
		if value != nil {
			env = EnvVar{Name: name, Value: *value}
		} else {
			result, _ := os.LookupEnv(name)
			if result != "" {
				env = EnvVar{Name: name, Value: result}
			} else {
				continue
			}
		}
		serviceConfig.Environment = append(serviceConfig.Environment, env)
	}
}

// Convert Docker Compose v3 ports to Ports
// expose ports will be treated as TCP ports
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v3.go#L185
func loadPorts(ports []composego.ServicePortConfig, expose []string) []Ports {
	komposePorts := []Ports{}

	exist := map[string]bool{}

	for _, port := range ports {
		// Convert to a kobject Ports struct
		// NOTE: V3 doesn't use IP (they utilize Swarm instead for host-networking).
		// Thus, IP is blank.
		komposePorts = append(komposePorts, Ports{
			HostPort:      int32(port.Published),
			ContainerPort: int32(port.Target),
			HostIP:        "",
			Protocol:      v1.Protocol(strings.ToUpper(string(port.Protocol))),
		})

		exist[cast.ToString(port.Target)+strings.ToUpper(string(port.Protocol))] = true
	}

	// Service should be exposed
	// @todo: Check how kompose determines that service should be exposed?
	if expose != nil {
		for _, port := range expose {
			portValue := port
			protocol := v1.ProtocolTCP
			if strings.Contains(portValue, "/") {
				splits := strings.Split(port, "/")
				portValue = splits[0]
				protocol = v1.Protocol(strings.ToUpper(splits[1]))
			}

			if exist[portValue+string(protocol)] {
				continue
			}

			komposePorts = append(komposePorts, Ports{
				HostPort:      cast.ToInt32(portValue),
				ContainerPort: cast.ToInt32(portValue),
				HostIP:        "",
				Protocol:      protocol,
			})
		}
	}

	return komposePorts
}

// Convert the Docker Compose v3 volumes to []string (the old way)
// TODO: Check to see if it's a "bind" or "volume". Ignore for now.
// TODO: Refactor it similar to loadV3Ports
// See: https://docs.docker.com/compose/compose-file/#long-syntax-3
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v3.go#L163
func loadVolumes(volumes []composego.ServiceVolumeConfig) []string {

	var volArray []string
	for _, vol := range volumes {
		// There will *always* be Source when parsing
		v := vol.Source

		if vol.Target != "" {
			v = v + ":" + vol.Target
		}

		if vol.ReadOnly {
			v = v + ":ro"
		}

		volArray = append(volArray, v)
	}
	return volArray
}

// parseKomposeLabels parse kompose in-cluster-wordpress, also do some validation
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v3.go#L487
func parseKomposeLabels(labels map[string]string, serviceConfig *ServiceConfig) error {
	// Label handler
	// Labels used to influence conversion of kompose will be handled
	// from here for docker-composego. Each loader will have such handler.

	if serviceConfig.Labels == nil {
		serviceConfig.Labels = make(map[string]string)
	}

	// @todo: See how we could use existing & additional in-cluster-wordpress to better control outcome.
	for key, value := range labels {
		switch key {
		case LabelServiceType:
			serviceType, err := handleServiceType(value)
			if err != nil {
				return errors.Wrap(err, "handleServiceType failed")
			}
			serviceConfig.ServiceType = serviceType
		case LabelServiceExpose:
			serviceConfig.ExposeService = strings.Trim(strings.ToLower(value), " ,")
		case LabelNodePortPort:
			serviceConfig.NodePortPort = cast.ToInt32(value)
		case LabelServiceExposeTLSSecret:
			serviceConfig.ExposeServiceTLS = value
		case LabelImagePullSecret:
			serviceConfig.ImagePullSecret = value
		case LabelImagePullPolicy:
			serviceConfig.ImagePullPolicy = value
		default:
			serviceConfig.Labels[key] = value
		}
	}

	// @step validate service expose in-cluster-wordpress
	if serviceConfig.ExposeService == "" && serviceConfig.ExposeServiceTLS != "" {
		return errors.New("kompose.service.expose.tls-secret was specified without kompose.service.expose")
	}

	// @step validate service type in-cluster-wordpress
	if serviceConfig.ServiceType != string(v1.ServiceTypeNodePort) && serviceConfig.NodePortPort != 0 {
		return errors.New("kompose.service.type must be nodeport when assign node port value")
	}

	// @step validate service port in-cluster-wordpress
	if len(serviceConfig.Port) > 1 && serviceConfig.NodePortPort != 0 {
		return errors.New("cannot set kompose.service.nodeport.port when service has multiple ports")
	}

	return nil
}

// handleServiceType returns service type based on passed string value
// @orig: https://github.com/kubernetes/kompose/blob/1f0a097836fb4e0ae4a802eb7ab543a4f9493727/pkg/loader/compose/utils.go#L108
func handleServiceType(ServiceType string) (string, error) {
	switch strings.ToLower(ServiceType) {
	case "", "clusterip":
		return string(v1.ServiceTypeClusterIP), nil
	case "nodeport":
		return string(v1.ServiceTypeNodePort), nil
	case "loadbalancer":
		return string(v1.ServiceTypeLoadBalancer), nil
	case "headless":
		return config.HeadlessService, nil
	default:
		return "", errors.New("Unknown value " + ServiceType + " , supported values are 'nodeport, clusterip, headless or loadbalancer'")
	}
}

// handleVolume iterates through services and sets volumes for each
// respecting volume lables if set in the manifest
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v3.go#L535
func handleVolume(komposeObject *KomposeObject, volumes *composego.Volumes) {
	for name := range komposeObject.ServiceConfigs {
		vols, err := retrieveVolume(name, *komposeObject)
		if err != nil {
			errors.Wrap(err, "could not retrieve volume")
		}
		for volName, vol := range vols {
			size, selector := getVolumeLabels(vol.VolumeName, volumes)
			if len(size) > 0 || len(selector) > 0 {
				// We can't assign value to struct field in map while iterating over it, so temporary variable `temp` is used here
				var temp = vols[volName]
				temp.PVCSize = size
				temp.SelectorValue = selector
				vols[volName] = temp
			}
		}
		// We can't assign value to struct field in map while iterating over it, so temporary variable `temp` is used here
		var temp = komposeObject.ServiceConfigs[name]
		temp.Volumes = vols
		komposeObject.ServiceConfigs[name] = temp
	}
}

// getVolumeLabels returns size and selector if present in named volume in-cluster-wordpress
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v3.go#L559
func getVolumeLabels(name string, volumes *composego.Volumes) (string, string) {
	size, selector := "", ""

	if volume, ok := (*volumes)[name]; ok {
		for key, value := range volume.Labels {
			if key == LabelVolumeSize {
				size = value
			} else if key == LabelVolumeSelector {
				selector = value
			}
		}
	}

	return size, selector
}

// returns all volumes associated with service, if `volumes_from` key is used, we have to retrieve volumes from the services which are mentioned there. Hence, recursive function is used here.
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v1v2.go#L341
func retrieveVolume(svcName string, komposeObject KomposeObject) (volume []Volumes, err error) {
	// if volumes-from key is present
	if komposeObject.ServiceConfigs[svcName].VolumesFrom != nil {
		// iterating over services from `volumes-from`
		for _, depSvc := range komposeObject.ServiceConfigs[svcName].VolumesFrom {
			// recursive call for retrieving volumes of services from `volumes-from`
			dVols, err := retrieveVolume(depSvc, komposeObject)
			if err != nil {
				return nil, errors.Wrapf(err, "could not retrieve the volume")
			}
			var cVols []Volumes
			cVols, err = ParseVols(komposeObject.ServiceConfigs[svcName].VolList, svcName)
			if err != nil {
				return nil, errors.Wrapf(err, "error generating current volumes")
			}

			for _, cv := range cVols {
				// check whether volumes of current service is same or not as that of dependent volumes coming from `volumes-from`
				ok, dv := getVol(cv, dVols)
				if ok {
					// change current volumes service name to dependent service name
					if dv.VFrom == "" {
						cv.VFrom = dv.SvcName
						cv.SvcName = dv.SvcName
					} else {
						cv.VFrom = dv.VFrom
						cv.SvcName = dv.SvcName
					}
					cv.PVCName = dv.PVCName
				}
				volume = append(volume, cv)

			}
			// iterating over dependent volumes
			for _, dv := range dVols {
				// check whether dependent volume is already present or not
				if checkVolDependent(dv, volume) {
					// if found, add service name to `VFrom`
					dv.VFrom = dv.SvcName
					volume = append(volume, dv)
				}
			}
		}
	} else {
		// if `volumes-from` is not present
		volume, err = ParseVols(komposeObject.ServiceConfigs[svcName].VolList, svcName)
		if err != nil {
			return nil, errors.Wrapf(err, "error generating current volumes")
		}
	}
	return
}

// for dependent volumes, returns true and the respective volume if mountpath are same
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v1v2.go#L427
func getVol(toFind Volumes, Vols []Volumes) (bool, Volumes) {
	for _, dv := range Vols {
		if toFind.MountPath == dv.MountPath {
			return true, dv
		}
	}
	return false, Volumes{}
}

// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v1v2.go#L395
func checkVolDependent(dv Volumes, volume []Volumes) bool {
	for _, vol := range volume {
		if vol.PVCName == dv.PVCName {
			return false
		}
	}
	return true

}

// ParseVols parse volumes
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v1v2.go#L406
func ParseVols(volNames []string, svcName string) ([]Volumes, error) {
	var volumes []Volumes
	var err error

	for i, vn := range volNames {
		var v Volumes
		v.VolumeName, v.Host, v.Container, v.Mode, err = ParseVolume(vn)
		if err != nil {
			return nil, errors.Wrapf(err, "could not parse volume %q: %v", vn, err)
		}
		v.VolumeName = normalizeVolumes(v.VolumeName)
		v.SvcName = svcName
		v.MountPath = fmt.Sprintf("%s:%s", v.Host, v.Container)
		v.PVCName = fmt.Sprintf("%s-claim%d", v.SvcName, i)
		volumes = append(volumes, v)
	}

	return volumes, nil
}

// ParseVolume parses a given volume, which might be [name:][host:]container[:access_mode]
// @orig: https://github.com/kubernetes/kompose/blob/ca75c31df8257206d4c50d1cca23f78040bb98ca/pkg/transformer/utils.go#L58
func ParseVolume(volume string) (name, host, container, mode string, err error) {
	separator := ":"

	// @step Parse based on separator
	volumeStrings := strings.Split(volume, separator)
	if len(volumeStrings) == 0 {
		return
	}

	// @step Set name if existed
	if !isPath(volumeStrings[0]) {
		name = volumeStrings[0]
		volumeStrings = volumeStrings[1:]
	}

	// @step For empty volume strings
	if len(volumeStrings) == 0 {
		err = fmt.Errorf("invalid volume format: %s", volume)
		return
	}

	// @step Get the last ":" passed which is presumably the "access mode"
	possibleAccessMode := volumeStrings[len(volumeStrings)-1]

	// @step Check to see if :Z or :z exists. We do not support SELinux relabeling at the moment.
	// See https://github.com/kubernetes/kompose/issues/176
	// Otherwise, check to see if "rw" or "ro" has been passed
	if possibleAccessMode == "z" || possibleAccessMode == "Z" {
		fmt.Printf("Volume mount \"%s\" will be mounted without labeling support. :z or :Z not supported", volume)
		mode = ""
		volumeStrings = volumeStrings[:len(volumeStrings)-1]
	} else if possibleAccessMode == "rw" || possibleAccessMode == "ro" {
		mode = possibleAccessMode
		volumeStrings = volumeStrings[:len(volumeStrings)-1]
	}

	// @step Check the volume format as well as host
	container = volumeStrings[len(volumeStrings)-1]
	volumeStrings = volumeStrings[:len(volumeStrings)-1]
	if len(volumeStrings) == 1 {
		host = volumeStrings[0]
	}
	if !isPath(container) || (len(host) > 0 && !isPath(host)) || len(volumeStrings) > 1 {
		err = fmt.Errorf("invalid volume format: %s", volume)
		return
	}
	return
}

// @orig: https://github.com/kubernetes/kompose/blob/ca75c31df8257206d4c50d1cca23f78040bb98ca/pkg/transformer/utils.go#L117
func isPath(substring string) bool {
	return strings.Contains(substring, "/") || substring == "."
}
