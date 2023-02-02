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

package kustomize

import (
	kubernetes "github.com/appvia/kev/pkg/kev/converter/kubernetes"
	kmd "github.com/appvia/komando"
	composego "github.com/compose-spec/compose-go/types"

	"k8s.io/apimachinery/pkg/runtime"
)

const DefaultIngressBackendKeyword = "default"

// Kustomize transformer
type Kustomize struct {
	Opt      ConvertOptions     // user provided options from the command line
	Project  *composego.Project // docker compose project
	Excluded []string           // docker compose service names that should be excluded
	UI       kmd.UI
}

// Transform converts compose project to set of k8s objects
// returns object that are already sorted in the way that Services are first
// @orig: https://github.com/kubernetes/kompose/blob/master/pkg/transformer/kubernetes/kubernetes.go#L1140
func (k *Kustomize) Transform() ([]runtime.Object, error) {

	k8sTransformer := &kubernetes.Kubernetes{Project: k.Project, UI: k.UI}
	allObjects, err := k8sTransformer.Transform()

	return allObjects, err
}
