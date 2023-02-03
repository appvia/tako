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

type extensionOptions struct {
	skipValidation bool
}

// K8sExtensionOption will modify parsing behaviour of the k8s extension.
type K8sExtensionOption func(*extensionOptions)

// SkipValidation skips validation when parsing a k8s extension from a service.
func SkipValidation() K8sExtensionOption {
	return func(extOpts *extensionOptions) {
		extOpts.skipValidation = true
	}
}
