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
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/appvia/kube-devx/pkg/kev/app"
	"github.com/appvia/kube-devx/pkg/kev/version"
	"github.com/docker/go-units"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Selector used as labels and selector
const Selector = "io.kompose.service"

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
		return errors.Wrap(err, "Failed to generate Chart.yaml template, template.New failed")
	}
	var chartData bytes.Buffer
	_ = t.Execute(&chartData, details)

	err = ioutil.WriteFile(dirName+string(os.PathSeparator)+"Chart.yaml", chartData.Bytes(), 0644)
	if err != nil {
		return err
	}

	fmt.Printf("chart created in %q\n", dirName+string(os.PathSeparator))
	return nil
}

// Check if given path is a directory
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L115
func isDir(name string) (bool, error) {

	// Open file to get stat later
	f, err := os.Open(name)
	if err != nil {
		return false, nil
	}
	defer f.Close()

	// Get file attributes and information
	fileStat, err := f.Stat()
	if err != nil {
		return false, errors.Wrap(err, "error retrieving file information, f.Stat failed")
	}

	// Check if given path is a directory
	if fileStat.IsDir() {
		return true, nil
	}
	return false, nil
}

// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L137
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

// PrintList will take the data converted and decide on the commandline attributes given
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L153
// func PrintList(objects []runtime.Object, opt ConvertOptions) error {
func PrintList(objects []runtime.Object, opt ConvertOptions, rendered map[string]app.FileConfig) error {

	var f *os.File
	dirName := getDirName(opt)
	fmt.Printf("Target Dir: %s\n", dirName)

	// Check if output file is a directory
	isDirVal, err := isDir(opt.OutFile)
	if err != nil {
		return errors.Wrap(err, "isDir failed")
	}
	if opt.CreateChart {
		isDirVal = true
	}
	if !isDirVal {
		f, err = CreateOutFile(opt.OutFile)
		if err != nil {
			return errors.Wrap(err, "CreateOutFile failed")
		}
		defer f.Close()
	}

	var files []string

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
		convertedList, err := convertToVersion(list, listVersion)
		if err != nil {
			return err
		}

		data, err := marshal(convertedList, opt.GenerateJSON, opt.YAMLIndent)
		if err != nil {
			return fmt.Errorf("error in marshalling the List: %v", err)
		}

		printVal, err := Print("", dirName, "", data, opt.ToStdout, opt.GenerateJSON, f, opt.Provider)
		if err != nil {
			return errors.Wrap(err, "Print failed")
		}

		files = append(files, printVal)
		rendered[printVal] = app.FileConfig{
			Content: data,
			File:    printVal,
		}
	} else {
		// @step output directory specified - print all objects individually to that directory
		finalDirName := dirName

		// if that's a chart it'll spit things out to "templates" subdir
		if opt.CreateChart {
			finalDirName = path.Join(dirName, "templates")
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
			data, err := marshal(versionedObject, opt.GenerateJSON, opt.YAMLIndent)
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

			file, err = Print(objectMeta.Name, finalDirName, strings.ToLower(typeMeta.Kind), data, opt.ToStdout, opt.GenerateJSON, f, opt.Provider)
			if err != nil {
				return errors.Wrap(err, "Print failed")
			}

			files = append(files, file)
			rendered[file] = app.FileConfig{
				Content: data,
				File:    file,
			}
		}
	}
	// @step for helm output generate chart directory structure
	if opt.CreateChart {
		err = generateHelm(dirName)
		if err != nil {
			return errors.Wrap(err, "generateHelm failed")
		}
	}
	return nil
}

// marshal object runtime.Object and return byte array
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

// Convert JSON to YAML.
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

// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L308
func marshalWithIndent(o interface{}, indent int) ([]byte, error) {
	j, err := json.Marshal(o)
	if err != nil {
		return nil, fmt.Errorf("error marshaling into JSON: %s", err.Error())
	}

	y, err := jsonToYaml(j, indent)
	if err != nil {
		return nil, fmt.Errorf("error converting JSON to YAML: %s", err.Error())
	}

	return y, nil
}

// Convert object to versioned object
// if groupVersion is  empty (schema.GroupVersion{}), use version from original object (obj)
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L324
func convertToVersion(obj runtime.Object, groupVersion schema.GroupVersion) (runtime.Object, error) {

	// ignore unstruct object
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

// TranslatePodResource config pod resources
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L592
func TranslatePodResource(service *ServiceConfig, template *v1.PodTemplateSpec) {
	// Configure the resource limits
	if service.MemLimit != 0 || service.CPULimit != 0 {
		resourceLimit := v1.ResourceList{}

		if service.MemLimit != 0 {
			resourceLimit[v1.ResourceMemory] = *resource.NewQuantity(int64(service.MemLimit), resource.BinarySI)
		}

		if service.CPULimit != 0 {
			resourceLimit[v1.ResourceCPU] = *resource.NewMilliQuantity(service.CPULimit, resource.DecimalSI)
		}

		template.Spec.Containers[0].Resources.Limits = resourceLimit
	}

	// Configure the resource requests
	if service.MemReservation != 0 || service.CPUReservation != 0 {
		resourceRequests := v1.ResourceList{}

		if service.MemReservation != 0 {
			resourceRequests[v1.ResourceMemory] = *resource.NewQuantity(int64(service.MemReservation), resource.BinarySI)
		}

		if service.CPUReservation != 0 {
			resourceRequests[v1.ResourceCPU] = *resource.NewMilliQuantity(service.CPUReservation, resource.DecimalSI)
		}

		template.Spec.Containers[0].Resources.Requests = resourceRequests
	}

	return

}

// GetImagePullPolicy get image pull settings
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L628
func GetImagePullPolicy(name, policy string) (v1.PullPolicy, error) {
	switch policy {
	case "":
	case "Always":
		return v1.PullAlways, nil
	case "Never":
		return v1.PullNever, nil
	case "IfNotPresent":
		return v1.PullIfNotPresent, nil
	default:
		return "", errors.New("Unknown image-pull-policy " + policy + " for service " + name)
	}
	return "", nil

}

// GetRestartPolicy ...
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L645
// @todo Make RestartPolicy type consistent across the codebase!
func GetRestartPolicy(name, restart string) (v1.RestartPolicy, error) {
	switch restart {
	case "", "always", "any":
		return v1.RestartPolicyAlways, nil
	case "no", "none":
		return v1.RestartPolicyNever, nil
	case "on-failure":
		return v1.RestartPolicyOnFailure, nil
	default:
		return "", errors.New("Unknown restart policy " + restart + " for service " + name)
	}
}

// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L700
func resetWorkloadAPIVersion(d runtime.Object) runtime.Object {
	data, err := json.Marshal(d)
	if err == nil {
		var us unstructured.Unstructured
		if err := json.Unmarshal(data, &us); err == nil {
			us.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    d.GetObjectKind().GroupVersionKind().Kind,
			})
			return &us
		}
	}
	return d
}

// SortedKeys Ensure the kubernetes objects are in a consistent order
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L734
func SortedKeys(komposeObject KomposeObject) []string {
	var sortedKeys []string
	for name := range komposeObject.ServiceConfigs {
		sortedKeys = append(sortedKeys, name)
	}
	sort.Strings(sortedKeys)
	return sortedKeys
}

// DurationStrToSecondsInt converts duration string to *int64 in seconds
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L744
func DurationStrToSecondsInt(s string) (*int64, error) {
	if s == "" {
		return nil, nil
	}
	duration, err := time.ParseDuration(s)
	if err != nil {
		return nil, err
	}
	r := (int64)(duration.Seconds())
	return &r, nil
}

// GetEnvsFromFile get env vars from env_file
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L757
func GetEnvsFromFile(file string, opt ConvertOptions) (map[string]string, error) {
	// Get the correct file context / directory
	composeDir, err := GetComposeFileDir(opt.InputFiles)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to load file context")
	}
	fileLocation := path.Join(composeDir, file)

	// Load environment variables from file
	envLoad, err := godotenv.Read(fileLocation)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to read env_file")
	}

	return envLoad, nil
}

// GetContentFromFile gets the content from the file..
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L775
func GetContentFromFile(file string) (string, error) {
	fileBytes, err := ioutil.ReadFile(file)
	if err != nil {
		return "", errors.Wrap(err, "Unable to read file")
	}
	return string(fileBytes), nil
}

// FormatEnvName format env name
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L784
func FormatEnvName(name string) string {
	envName := strings.Trim(name, "./")
	envName = strings.Replace(envName, ".", "-", -1)
	envName = strings.Replace(envName, "/", "-", -1)
	return envName
}

// FormatFileName format file name
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L792
func FormatFileName(name string) string {
	// Split the filepath name so that we use the
	// file name (after the base) for ConfigMap,
	// it shouldn't matter whether it has special characters or not
	_, file := path.Split(name)

	// Make it DNS-1123 compliant for Kubernetes
	return strings.Replace(file, "_", "-", -1)
}

//FormatContainerName format Container name
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/k8sutils.go#L803
func FormatContainerName(name string) string {
	name = strings.Replace(name, "_", "-", -1)
	return name

}

// ConfigLabels configures label name alone
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/utils.go#L122
func ConfigLabels(name string) map[string]string {
	return map[string]string{Selector: name}
}

// ConfigAllLabels creates in-cluster-wordpress with service nam and deploy in-cluster-wordpress
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/utils.go#L140
func ConfigAllLabels(name string, service *ServiceConfig) map[string]string {
	base := ConfigLabels(name)
	if service.DeployLabels != nil {
		for k, v := range service.DeployLabels {
			base[k] = v
		}
	}
	return base

}

// ConfigAnnotations configures annotations
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/utils.go#L152
func ConfigAnnotations(service ServiceConfig) map[string]string {

	annotations := map[string]string{}
	for key, value := range service.Annotations {
		annotations[key] = value
	}
	annotations["kompose.cmd"] = strings.Join(os.Args, " ")
	versionCmd := exec.Command("kompose", "version")
	out, err := versionCmd.Output()
	if err != nil {
		errors.Wrap(err, "Failed to get kompose version")

	}
	annotations["kompose.version"] = strings.Trim(string(out), " \n")

	// If the version is blank (couldn't retrieve the kompose version for whatever reason)
	if annotations["kompose.version"] == "" {
		annotations["kompose.version"] = version.Version()
	}

	return annotations
}

// ParseIngressPath parse path for ingress.
// eg. example.com/org -> example.com org
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/utils.go#L109
func ParseIngressPath(url string) (string, string) {
	if strings.Contains(url, "/") {
		splits := strings.Split(url, "/")
		return splits[0], "/" + splits[1]
	}
	return url, ""
}

// GetComposeFileDir returns compose file directory
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/utils.go#L233
func GetComposeFileDir(inputFiles []string) (string, error) {
	// This assumes all the docker-compose files are in the same directory
	inputFile := inputFiles[0]
	if strings.Index(inputFile, "/") != 0 {
		workDir, err := os.Getwd()
		if err != nil {
			return "", err
		}
		inputFile = filepath.Join(workDir, inputFile)
	}
	fmt.Printf("Compose file dir: %s", filepath.Dir(inputFile))
	return filepath.Dir(inputFile), nil
}

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

// Print either prints to stdout or to file/s
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/utils.go#L176
func Print(name, path string, trailing string, data []byte, toStdout, generateJSON bool, f *os.File, provider string) (string, error) {
	file := ""
	if generateJSON {
		file = fmt.Sprintf("%s-%s.json", name, trailing)
	} else {
		file = fmt.Sprintf("%s-%s.yaml", name, trailing)
	}
	if toStdout {
		fmt.Fprintf(os.Stdout, "%s\n", string(data))
		return "", nil
	} else if f != nil {
		// Write all content to a single file f
		if _, err := f.WriteString(fmt.Sprintf("%s\n", string(data))); err != nil {
			return "", errors.Wrap(err, "f.WriteString failed, Failed to write %s to file: "+trailing)
		}
		f.Sync()
	} else {
		// Write content separately to each file
		file = filepath.Join(path, file)
		if err := ioutil.WriteFile(file, []byte(data), 0644); err != nil {
			return "", errors.Wrap(err, "Failed to write %s: "+trailing)
		}
		fmt.Printf("⎈  %s file %q created\n", Name, file)
	}
	return file, nil
}

// CreateOutFile creates the file to write to if --out is specified
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/utils.go#L45
func CreateOutFile(out string) (*os.File, error) {
	var f *os.File
	var err error
	if len(out) != 0 {
		f, err = os.Create(out)
		if err != nil {
			return nil, errors.Wrap(err, "error creating file, os.Create failed")
		}
	}
	return f, nil
}

// ConfigLabelsWithNetwork configures label and add Network Information in in-cluster-wordpress
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/utils.go#L127
func ConfigLabelsWithNetwork(name string, net []string) map[string]string {

	labels := map[string]string{}
	labels[Selector] = name

	for _, n := range net {
		labels["io.kompose.network/"+n] = "true"
	}
	return labels
}

// MemStringorInt represents a string or an integer
// the String supports notations like 10m for ten Megabytes of memory
// NOTE: Extacted from https://github.com/docker/libcompose/blob/master/yaml/types_yaml.go#L38-L62
// 		 as we use github.com/compose-spec/compose-go and want to avoid potential conflicts.
type MemStringorInt int64

// UnmarshalYAML implements the Unmarshaller interface.
func (s *MemStringorInt) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var intType int64
	if err := unmarshal(&intType); err == nil {
		*s = MemStringorInt(intType)
		return nil
	}

	var stringType string
	if err := unmarshal(&stringType); err == nil {
		intType, err := units.RAMInBytes(stringType)

		if err != nil {
			return err
		}
		*s = MemStringorInt(intType)
		return nil
	}

	return errors.New("Failed to unmarshal MemStringorInt")
}
