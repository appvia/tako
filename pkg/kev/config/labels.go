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

package config

const (
	LabelPrefix = "kev."

	// LabelWorkloadRollingUpdateMaxSurge max number of nodes updated at once
	LabelWorkloadRollingUpdateMaxSurge = LabelPrefix + "workload.rolling-update-max-surge"

	// LabelWorkloadServiceAccountName defines service account name to be used by the workload
	LabelWorkloadServiceAccountName = LabelPrefix + "workload.service-account-name"

	// LabelServiceNodePortPort defines port number for NodePort k8s service kind
	LabelServiceNodePortPort = LabelPrefix + "service.nodeport.port"

	// LabelServiceExpose informs whether K8s service should be exposed externally. To enable set as "true" or "domain.com,otherdomain.com".
	LabelServiceExpose = LabelPrefix + "service.expose"

	// LabelServiceExposeTLSSecret  provides the name of the TLS secret to use with the Kubernetes ingress controller
	LabelServiceExposeTLSSecret = LabelPrefix + "service.expose.tls-secret"
)

var BaseServiceLabels []string
