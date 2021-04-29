/**
 * Copyright 2021 Appvia Ltd <info@appvia.io>
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

package config

// VolumeExtension represents the root of the docker-compose extensions for a volume
type VolumeExtension struct {
	K8S K8sVol `yaml:"x-k8s"`
}

// K8sVol represents the root of the k8s specific fields supported by kev.
type K8sVol struct {
	Size         int    `yaml:"size,omitempty"`
	StorageClass string `yaml:"storageClass,omitempty"`
	Selector     string `yaml:"selector,omitempty"`
}
