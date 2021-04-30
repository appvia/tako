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

// Note: Functionality below have been extracted from Kompose project and updated accordingly
// to meet new dependencies and requirements of this tool.
// Links to original implementation have been added for reference.

package kubernetes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/appvia/kev/pkg/kev/config"
	"github.com/appvia/kev/pkg/kev/log"
	composego "github.com/compose-spec/compose-go/types"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Selector used as labels and selector
const (
	Selector     = "io.kev.service"
	NetworkLabel = "io.kev.network"
)

// EnvSort struct
type EnvSort []v1.EnvVar

// Len returns the number of elements in the collection.
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/utils.go#L214-L228
func (env EnvSort) Len() int {
	return len(env)
}

// Less returns whether the element with index i should sort before
// the element with index j.
func (env EnvSort) Less(i, j int) bool {
	return env[i].Name < env[j].Name
}

// swaps the elements with indexes i and j.
func (env EnvSort) Swap(i, j int) {
	env[i], env[j] = env[j], env[i]
}

// PrintList prints k8s objects
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L153
func PrintList(objects []runtime.Object, opt ConvertOptions, rendered map[string][]byte) error {

	var f *os.File
	dirName := getDirName(opt)
	log.Debugf("Target Dir: %s", dirName)

	// Check if output file is a directory
	isDirVal, err := isDir(opt.OutFile)
	if err != nil {
		log.Error("Directory check failed")
		return err
	}
	if opt.CreateChart {
		isDirVal = true
	}
	if !isDirVal {
		f, err = createOutFile(opt.OutFile)
		if err != nil {
			log.Error("Error creating output file")
			return err
		}
		defer func(f *os.File) {
			err := f.Close()
			if err != nil {
				log.Error("Error closing output file")
			}
		}(f)
	}

	var files []string
	var indent int

	if opt.YAMLIndent > 0 {
		indent = opt.YAMLIndent
	} else {
		indent = 2
	}

	// @step print to stdout, or to a single file - it will return a list object
	if opt.ToStdout || f != nil {
		list := &v1.List{}
		// convert objects to versioned and add them to list
		for _, object := range objects {
			versionedObject, err := convertToVersion(object, schema.GroupVersion{})
			if err != nil {
				return err
			}

			list.Items = append(list.Items, runtime.RawExtension{Object: versionedObject})
		}
		// version list itself
		listVersion := schema.GroupVersion{Group: "", Version: "v1"}
		list.Kind = "List"
		list.APIVersion = "v1"
		convertedList, err := convertToVersion(list, listVersion)
		if err != nil {
			return err
		}

		data, err := marshal(convertedList, opt.GenerateJSON, indent)
		if err != nil {
			log.Error("Error in marshalling the List")
			return err
		}

		printVal, err := print("", dirName, "", data, opt.ToStdout, opt.GenerateJSON, f)
		if err != nil {
			log.Error("Printing manifests failed")
			return err
		}

		files = append(files, printVal)
		rendered[printVal] = data
	} else {
		// @step output directory specified - print all objects individually to that directory
		finalDirName := dirName

		// if that's a chart it'll spit things out to "templates" subdir
		if opt.CreateChart {
			finalDirName = path.Join(dirName, "templates")
		}

		if err := os.RemoveAll(finalDirName); err != nil {
			return err
		}

		if err := os.MkdirAll(finalDirName, 0755); err != nil {
			return err
		}

		var file string
		// create a separate file for each provider
		for _, v := range objects {
			versionedObject, err := convertToVersion(v, schema.GroupVersion{})
			if err != nil {
				return err
			}
			data, err := marshal(versionedObject, opt.GenerateJSON, indent)
			if err != nil {
				return err
			}

			var typeMeta meta.TypeMeta
			var objectMeta meta.ObjectMeta

			if us, ok := v.(*unstructured.Unstructured); ok {
				typeMeta = meta.TypeMeta{
					Kind:       us.GetKind(),
					APIVersion: us.GetAPIVersion(),
				}
				objectMeta = meta.ObjectMeta{
					Name: us.GetName(),
				}
			} else {
				val := reflect.ValueOf(v).Elem()
				// Use reflect to access TypeMeta struct inside runtime.Object.
				// cast it to correct type - meta.TypeMeta
				typeMeta = val.FieldByName("TypeMeta").Interface().(meta.TypeMeta)

				// Use reflect to access ObjectMeta struct inside runtime.Object.
				// cast it to correct type - meta.ObjectMeta
				objectMeta = val.FieldByName("ObjectMeta").Interface().(meta.ObjectMeta)

			}

			file, err = print(objectMeta.Name, finalDirName, strings.ToLower(typeMeta.Kind), data, opt.ToStdout, opt.GenerateJSON, f)
			if err != nil {
				log.Error("Printing manifests failed")
				return err
			}

			files = append(files, file)
			rendered[file] = data
		}
	}
	// @step for helm output generate chart directory structure
	if opt.CreateChart {
		err = generateHelm(dirName)
		if err != nil {
			log.Error("Couldn't generate HELM chart")
			return err
		}
	}
	return nil
}

// print either renders to stdout or to file/s
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/utils.go#L176
func print(name, path string, trailing string, data []byte, toStdout, generateJSON bool, f *os.File) (string, error) {
	file := ""
	if generateJSON {
		file = fmt.Sprintf("%s-%s.json", name, trailing)
	} else {
		file = fmt.Sprintf("%s-%s.yaml", name, trailing)
	}
	if toStdout {
		_, _ = fmt.Fprintf(os.Stdout, "%s\n", string(data))
		return "", nil
	} else if f != nil {
		// Write all content to a single file f
		if _, err := f.WriteString(fmt.Sprintf("%s\n", string(data))); err != nil {
			log.Error("Couldn't write manifests content to a single file")
			return "", err
		}
		_ = f.Sync()
	} else {
		// Write content separately to each file
		file = filepath.Join(path, file)
		if err := ioutil.WriteFile(file, []byte(data), 0644); err != nil {
			log.ErrorWithFields(log.Fields{
				"file": file,
			}, "Failed to write content to a file")
			return "", err
		}
		log.Debugf("%s file %q created", Name, file)
	}
	return file, nil
}

//  Generate Helm Chart configuration
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L54
func generateHelm(dirName string) error {
	type ChartDetails struct {
		Name string
	}

	details := ChartDetails{dirName}
	manifestDir := dirName + string(os.PathSeparator) + "templates"
	dir, err := os.Open(dirName)

	// @step Setup the initial directories/files
	if err == nil {
		_ = dir.Close()
	}

	if err != nil {
		err = os.Mkdir(dirName, 0755)
		if err != nil {
			return err
		}

		err = os.Mkdir(manifestDir, 0755)
		if err != nil {
			return err
		}
	}

	// @step Create the readme file
	readme := "This chart was created by Kompose\n"
	err = ioutil.WriteFile(dirName+string(os.PathSeparator)+"README.md", []byte(readme), 0644)
	if err != nil {
		return err
	}

	// @step Create the Chart.yaml file
	chart := `name: {{.Name}}
description: A generated Helm Chart for {{.Name}} from Skippbox Kompose
version: 0.0.1
apiVersion: v1
keywords:
  - {{.Name}}
sources:
home:
`

	t, err := template.New("ChartTmpl").Parse(chart)
	if err != nil {
		log.Error("Failed to generate Chart template")
		return err
	}
	var chartData bytes.Buffer
	_ = t.Execute(&chartData, details)

	err = ioutil.WriteFile(dirName+string(os.PathSeparator)+"Chart.yaml", chartData.Bytes(), 0644)
	if err != nil {
		return err
	}

	log.Infof("chart created in %q", dirName+string(os.PathSeparator))
	return nil
}

// Check if given path is a directory
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L115
func isDir(name string) (bool, error) {

	f, err := os.Open(name)
	if err != nil {
		return false, nil
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Error("error closing a file.")
		}
	}(f)

	// Get file attributes and information
	fileStat, err := f.Stat()
	if err != nil {
		log.ErrorWithFields(log.Fields{
			"file": name,
		}, "Error retrieving file information. Stat failed.")
		return false, err
	}

	// Check if given path is a directory
	if fileStat.IsDir() {
		return true, nil
	}
	return false, nil
}

// getDirName gets directory name
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L137
// @todo: UNUSED yet but could use it to determine output directory
func getDirName(opt ConvertOptions) string {
	dirName := opt.OutFile
	if dirName == "" {
		// This assumes all the docker-compose files are in the same directory
		if opt.CreateChart {
			filename := opt.InputFiles[0]
			extension := filepath.Ext(filename)
			dirName = filename[0 : len(filename)-len(extension)]
		} else {
			dirName = "."
		}
	}
	return dirName
}

// marshal marshals a runtime.Object and return byte array
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L269
func marshal(obj runtime.Object, jsonFormat bool, indent int) (data []byte, err error) {
	// convert data to yaml or json
	if jsonFormat {
		data, err = json.MarshalIndent(obj, "", "  ")
	} else {
		data, err = marshalWithIndent(obj, indent)
	}
	if err != nil {
		data = nil
	}
	return
}

// jsonToYaml converts JSON to YAML
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L283
func jsonToYaml(j []byte, spaces int) ([]byte, error) {
	// Convert the JSON to an object.
	var jsonObj interface{}
	// We are using yaml.Unmarshal here (instead of json.Unmarshal) because the
	// Go JSON library doesn't try to pick the right number type (int, float,
	// etc.) when unmarshling to interface{}, it just picks float64
	// universally. go-yaml does go through the effort of picking the right
	// number type, so we can preserve number type throughout this process.
	err := yaml.Unmarshal(j, &jsonObj)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(spaces)
	if err := encoder.Encode(jsonObj); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// marshalWithIndent marshals with specified indentation size
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L308
func marshalWithIndent(o interface{}, indent int) ([]byte, error) {
	j, err := json.Marshal(o)
	if err != nil {
		log.Error("Error marshaling into JSON")
		return nil, err
	}

	y, err := jsonToYaml(j, indent)
	if err != nil {
		log.Error("Error converting JSON to YAML")
		return nil, err
	}

	return y, nil
}

// convertToVersion converts object to a versioned object
// if groupVersion is  empty (schema.GroupVersion{}), use version from original object (obj)
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L324
func convertToVersion(obj runtime.Object, groupVersion schema.GroupVersion) (runtime.Object, error) {

	// ignore unstructured object
	if _, ok := obj.(*unstructured.Unstructured); ok {
		return obj, nil
	}

	var version schema.GroupVersion

	if groupVersion.Empty() {
		objectVersion := obj.GetObjectKind().GroupVersionKind()
		version = schema.GroupVersion{Group: objectVersion.Group, Version: objectVersion.Version}
	} else {
		version = groupVersion
	}

	s := runtime.NewScheme()
	s.AddKnownTypes(version, obj)
	convertedObject, err := s.ConvertToVersion(obj, version)
	if err != nil {
		return nil, err
	}

	return convertedObject, nil
}

// getImagePullPolicy returns image pull policy based on the string input
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L628
func getImagePullPolicy(projectServiceName, policy string) (v1.PullPolicy, error) {
	switch strings.ToLower(policy) {
	case "", "always":
		return v1.PullAlways, nil
	case "never":
		return v1.PullNever, nil
	case "ifnotpresent":
		return v1.PullIfNotPresent, nil
	default:
		return "", fmt.Errorf("Unknown image-pull-policy %s for service %s", policy, projectServiceName)
	}
}

// getRestartPolicy returns K8s RestartPolicy
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L645
func getRestartPolicy(projectServiceName, restart string) (v1.RestartPolicy, error) {
	switch strings.ToLower(restart) {
	case "", "always", "any":
		return v1.RestartPolicyAlways, nil
	case "no", "none", "never":
		return v1.RestartPolicyNever, nil
	case "on-failure", "onfailure":
		return v1.RestartPolicyOnFailure, nil
	default:
		return "", fmt.Errorf("Unknown restart policy %s for service %s", restart, projectServiceName)
	}
}

// getServiceType returns service type based on passed string value
// @orig: https://github.com/kubernetes/kompose/blob/1f0a097836fb4e0ae4a802eb7ab543a4f9493727/pkg/loader/compose/utils.go#L108
func getServiceType(serviceType string) (string, error) {
	switch strings.ToLower(serviceType) {
	case "", "clusterip":
		return string(v1.ServiceTypeClusterIP), nil
	case "nodeport":
		return string(v1.ServiceTypeNodePort), nil
	case "loadbalancer":
		return string(v1.ServiceTypeLoadBalancer), nil
	case "headless":
		return config.HeadlessService, nil
	case "none":
		return config.NoService, nil
	default:
		return "", fmt.Errorf("Unknown value %s, supported values are 'none, nodeport, clusterip, headless or loadbalancer'", serviceType)
	}
}

// sortServices sorts all compose project services by name
func sortServices(project *composego.Project) {
	sort.Slice(project.Services, func(i, j int) bool {
		return project.Services[i].Name < project.Services[j].Name
	})
}

// durationStrToSecondsInt converts duration string to *int32 in seconds
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L744
func durationStrToSecondsInt(s string) (*int32, error) {
	if s == "" {
		return nil, nil
	}
	duration, err := time.ParseDuration(s)
	if err != nil {
		return nil, err
	}
	r := (int32)(duration.Seconds())
	return &r, nil
}

// getContentFromFile gets the content from the file..
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L775
func getContentFromFile(file string) (string, error) {
	fileBytes, err := ioutil.ReadFile(file)
	if err != nil {
		log.ErrorWithFields(log.Fields{
			"file": file,
		}, "Unable to read file")
		return "", err
	}
	return string(fileBytes), nil
}

// rfc1123
// NOTE: only accept alphanumeric chars (specifically excluding dots)
// https://kubernetes.io/docs/concepts/overview/working-with-objects/names/
func rfc1123(s string) string {
	re := regexp.MustCompile("[^A-Za-z0-9]+")
	return strings.Trim(strings.ToLower(re.ReplaceAllString(s, "-")), "-")
}

// rfc1123dns
// https://kubernetes.io/docs/concepts/overview/working-with-objects/names/
func rfc1123dns(s string) string {
	s = rfc1123(s)
	if len(s) > 253 {
		return s[0:253]
	}
	return s
}

// rfc1123label
// https://kubernetes.io/docs/concepts/overview/working-with-objects/names/
func rfc1123label(s string) string {
	s = rfc1123(s)
	if len(s) > 63 {
		return s[0:63]
	}
	return s
}

// formatFileName format file name
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L792
func formatFileName(name string) string {
	// Split the filepath name so that we use the
	// file name (after the base) for ConfigMap,
	// it shouldn't matter whether it has special characters or not
	_, file := path.Split(name)

	// Make it DNS-1123 compliant for Kubernetes
	return rfc1123(file)
}

// configLabels configures selector label for project service passed
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/utils.go#L122
func configLabels(name string) map[string]string {
	return map[string]string{Selector: name}
}

// configAllLabels creates labels with service name and deploy labels
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/utils.go#L140
func configAllLabels(projectService ProjectService) map[string]string {
	base := configLabels(projectService.Name)
	if projectService.Deploy != nil && projectService.Deploy.Labels != nil {
		for k, v := range projectService.Deploy.Labels {
			base[k] = v
		}
	}
	return base
}

// configAnnotations configures annotations - they are effectively compose project service labels,
// but will exclude all Kev configuration labels by default.
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/utils.go#L152
func configAnnotations(projectService ProjectService) map[string]string {
	annotations := map[string]string{}
	for key, value := range projectService.Labels {
		// don't turn kev configuration labels into kubernetes annotations!
		if !strings.HasPrefix(key, config.LabelPrefix) {
			annotations[key] = value
		}
	}
	return annotations
}

// parseIngressPath parses the path for ingress.
// eg. example.com/org -> example.com org
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/utils.go#L109
func parseIngressPath(url string) (string, string) {
	if strings.Contains(url, "/") {
		splits := strings.Split(url, "/")
		return splits[0], "/" + splits[1]
	}
	return url, ""
}

// getComposeFileDir returns compose file directory
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/utils.go#L233
func getComposeFileDir(inputFiles []string) (string, error) {
	// This assumes all the docker-compose files are in the same directory
	inputFile := inputFiles[0]
	if strings.Index(inputFile, "/") != 0 {
		workDir, err := os.Getwd()
		if err != nil {
			return "", err
		}
		inputFile = filepath.Join(workDir, inputFile)
	}
	log.Debugf("Compose file dir: %s", filepath.Dir(inputFile))
	return filepath.Dir(inputFile), nil
}

// createOutFile creates the file to write to if --out is specified
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/utils.go#L45
func createOutFile(out string) (*os.File, error) {
	var f *os.File
	var err error
	if len(out) != 0 {
		f, err = os.Create(out)
		if err != nil {
			log.ErrorWithFields(log.Fields{
				"file": out,
			}, "Unable to create a file.")
			return nil, err
		}
	}
	return f, nil
}

// configLabelsWithNetwork configures label and add Network Information in labels
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/utils.go#L127
func configLabelsWithNetwork(projectService ProjectService) map[string]string {
	labels := map[string]string{}
	labels[Selector] = projectService.Name

	for n := range projectService.Networks {
		labels[NetworkLabel+"/"+n] = "true"
	}

	return labels
}

// findByName selects compose project service by name
func findByName(projectServices composego.Services, name string) *composego.ServiceConfig {
	for _, ps := range projectServices {
		if rfc1123dns(ps.Name) == name {
			return &ps
		}
	}
	return nil
}

// retrieveVolume returns all volumes associated with service.
// If `volumes_from` key is used, we also retrieve volumes used by those services. Hence, recursive function call.
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v1v2.go#L341
func retrieveVolume(projectServiceName string, project *composego.Project) (volume []Volumes, err error) {

	// @step find service by name passed in args
	projectService := findByName(project.Services, projectServiceName)
	if projectService == nil {
		log.ErrorWithFields(log.Fields{
			"project-service": projectServiceName,
		}, "Could not find a project service with name")

		return nil, fmt.Errorf("Could not find a project service with name %s", projectServiceName)
	}

	// @step if volumes-from key is present
	if projectService.VolumesFrom != nil {

		// iterating over services from `volumes-from`
		for _, depSvc := range projectService.VolumesFrom {

			// recursive call for retrieving volumes from `volumes-from` services
			dVols, err := retrieveVolume(depSvc, project)
			if err != nil {
				log.Error("Could not retrieve the volume")
				return nil, errors.New("Could not retrieve the volume")
			}

			var cVols []Volumes
			cVols, err = parseVols(loadVolumes(projectService.Volumes), projectService.Name)
			if err != nil {
				log.Error("Error generating current volumes")
				return nil, errors.New("Error generating current volumes")
			}

			for _, cv := range cVols {
				// check whether volumes of current service is the same (or not) as that of dependent volumes coming from `volumes-from`
				// check is done based on the MountPath
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
				// check is done based on volume PVCName
				if checkVolDependent(dv, volume) {
					// if found, add service name to `VFrom`
					dv.VFrom = dv.SvcName
					volume = append(volume, dv)
				}
			}

		}
	} else {
		// @step if `volumes-from` is not present
		volume, err = parseVols(loadVolumes(projectService.Volumes), projectService.Name)
		if err != nil {
			log.Error("Error generating current volumes")
			return nil, errors.New("Error generating current volumes")
		}
	}

	return
}

// parseVols parses slice of volume strings for a project service and returns slice of Volumes objects
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v1v2.go#L406
func parseVols(volNames []string, svcName string) ([]Volumes, error) {
	var volumes []Volumes
	var err error

	for i, vn := range volNames {
		var v Volumes

		v.VolumeName, v.Host, v.Container, v.Mode, err = parseVolume(vn)
		if err != nil {
			log.ErrorWithFields(log.Fields{
				"volume": vn,
			}, "Could not parse volume")

			return nil, err
		}

		v.VolumeName = rfc1123(v.VolumeName)
		v.SvcName = svcName
		v.MountPath = fmt.Sprintf("%s:%s", v.Host, v.Container)
		v.PVCName = fmt.Sprintf("%s-claim%d", v.SvcName, i)

		volumes = append(volumes, v)
	}

	return volumes, nil
}

// parseVolume parses a given volume, which might be [name:][host:]container[:access_mode]
// @orig: https://github.com/kubernetes/kompose/blob/ca75c31df8257206d4c50d1cca23f78040bb98ca/pkg/transformer/utils.go#L58
func parseVolume(volume string) (name, host, container, mode string, err error) {
	separator := ":"

	// @step Parse based on separator
	volumeStrings := strings.Split(volume, separator)

	// @step For empty volume strings
	if len(volumeStrings) == 0 {
		log.ErrorWithFields(log.Fields{
			"volume": volume,
		}, "Invalid volume name format")

		err = fmt.Errorf("Invalid volume format: %s", volume)
		return
	}

	// @step Set name if existed
	if !isPath(volumeStrings[0]) {
		name = volumeStrings[0]
		volumeStrings = volumeStrings[1:]
	}

	// @step Ensure volume name is not the only thing provided
	if len(volumeStrings) == 0 {
		log.ErrorWithFields(log.Fields{
			"volume": volume,
		}, "Invalid volume name format")

		err = fmt.Errorf("Invalid volume format: %s", volume)
		return
	}

	// @step Get the last ":" passed which is presumably the "access mode"
	possibleAccessMode := volumeStrings[len(volumeStrings)-1]

	// @step Check to see if :Z or :z exists. We do not support SELinux relabeling at the moment.
	// See https://github.com/kubernetes/kompose/issues/176
	// Otherwise, check to see if "rw" or "ro" has been passed
	if possibleAccessMode == "z" || possibleAccessMode == "Z" {
		log.Debugf("Volume mount \"%s\" will be mounted without labeling support. :z or :Z not supported", volume)
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
		log.ErrorWithFields(log.Fields{
			"volume": volume,
		}, "Invalid volume name format")

		err = fmt.Errorf("Invalid volume format: %s", volume)
		return
	}
	return
}

// isPath determines whether supplied strings has a path format
// @orig: https://github.com/kubernetes/kompose/blob/ca75c31df8257206d4c50d1cca23f78040bb98ca/pkg/transformer/utils.go#L117
func isPath(substring string) bool {
	return strings.Contains(substring, "/") || substring == "."
}

// loadVolumes Convert the Docker Compose v3 volumes to []string (the old way)
// TODO: Check to see if it's a "bind" or "volume". Ignore for now.
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

// getVol for dependent volumes, returns true and the respective volume if mountpath are the same
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v1v2.go#L427
func getVol(v Volumes, volumes []Volumes) (bool, Volumes) {
	for _, dv := range volumes {
		if dv.MountPath == v.MountPath {
			return true, dv
		}
	}
	return false, Volumes{}
}

// checkVolDependent checks whether dependent volume is already defined
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v1v2.go#L395
func checkVolDependent(dv Volumes, volumes []Volumes) bool {
	for _, vol := range volumes {
		if vol.PVCName == dv.PVCName {
			return false
		}
	}

	return true
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

// loadPlacement parses placement information from composego
// @orig: https://github.com/kubernetes/kompose/blob/e7f05588bf8bd645000612faa136b1b6aa0d5bb6/pkg/loader/compose/v3.go#L136
func loadPlacement(constraints []string) map[string]string {
	placement := make(map[string]string)

	errMsg := "Constraint in placement is not supported. Only 'node.hostname==...', 'node.role==worker', 'node.role==manager', 'engine.labels.operatingsystem' and 'node.labels.(...)' (ex: node.labels.something==anything) is supported as a constraint"

	for _, c := range constraints {
		p := strings.Split(strings.Replace(c, " ", "", -1), "==")

		if len(p) < 2 {
			log.WarnWithFields(log.Fields{"placement": p[0]}, errMsg)
			continue
		}

		if p[0] == "node.role" && p[1] == "worker" {
			placement["node-role.kubernetes.io/worker"] = "true"
		} else if p[0] == "node.role" && p[1] == "manager" {
			placement["node-role.kubernetes.io/master"] = "true"
		} else if p[0] == "node.hostname" {
			placement["kubernetes.io/hostname"] = p[1]
		} else if p[0] == "engine.labels.operatingsystem" {
			placement["beta.kubernetes.io/os"] = p[1]
		} else if strings.HasPrefix(p[0], "node.labels.") {
			label := strings.TrimPrefix(p[0], "node.labels.")
			placement[label] = p[1]
		} else {
			log.WarnWithFields(log.Fields{"placement": p[0]}, errMsg)
		}
	}

	return placement
}

// contains returns true of slice of strings contains a given string
func contains(strs []string, s string) bool {
	sort.Strings(strs)
	i := sort.SearchStrings(strs, s)
	return i < len(strs) && strs[i] == s
}

// runtimeObjectConvertToTarget converts runtime object into a target
func runtimeObjectConvertToTarget(o runtime.Object, target interface{}) error {
	raw, err := ToUnstructured(o)
	if err != nil {
		return err
	}

	return FromUnstructured(raw, target)
}

// ToUnstructured converts runtime.Object to unstructured map[string]interface{}
func ToUnstructured(o runtime.Object) (map[string]interface{}, error) {
	raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(o)
	if err != nil {
		return nil, err
	}

	return raw, nil
}

// FromUnstructured converts unstructured to target object
func FromUnstructured(unstructured map[string]interface{}, target interface{}) error {
	return runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured, target)
}
