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

package tako

import "github.com/pkg/errors"

// String returns string representation of a RunnerEvent.
// Useful for use in error messages.
func (e RunnerEvent) String() string {
	switch e {
	case PreLoadProject:
		return "PreLoadProject"
	case PostLoadProject:
		return "PostLoadProject"
	case PreValidateSources:
		return "PreValidateSources"
	case PostValidateSources:
		return "PostValidateSources"
	case PreVerifySkaffold:
		return "PreVerifySkaffold"
	case PostVerifySkaffold:
		return "PostVerifySkaffold"
	case PreValidateEnvSources:
		return "PreValidateEnvSources"
	case PostValidateEnvSources:
		return "PostValidateEnvSources"
	case PreReconcileEnvs:
		return "PreReconcileEnvs"
	case PostReconcileEnvs:
		return "PostReconcileEnvs"
	case PreRenderFromComposeToK8sManifests:
		return "PreRenderFromComposeToK8sManifests"
	case PostRenderFromComposeToK8sManifests:
		return "PostRenderFromComposeToK8sManifests"
	case PreEnsureFirstInit:
		return "PreEnsureFirstInit"
	case PostEnsureFirstInit:
		return "PostEnsureFirstInit"
	case PreDetectSources:
		return "PreDetectSources"
	case PostDetectSources:
		return "PostDetectSources"
	case PreCreateManifest:
		return "PreCreateManifest"
	case PostCreateManifest:
		return "PostCreateManifest"
	case PreCreateOrUpdateSkaffoldManifest:
		return "PreCreateOrUpdateSkaffoldManifest"
	case PostCreateOrUpdateSkaffoldManifest:
		return "PostCreateOrUpdateSkaffoldManifest"
	case PrePrintSummary:
		return "PrePrintSummary"
	case PostPrintSummary:
		return "PostPrintSummary"
	case SecretsDetected:
		return "SecretsDetected"
	case DevLoopStarting:
		return "DevLoopStarting"
	case DevLoopIterated:
		return "DevLoopIterated"
	default:
		return ""
	}
}

// RunnerEvents
const (
	PreLoadProject RunnerEvent = iota
	PostLoadProject
	PreValidateSources
	PostValidateSources
	PreVerifySkaffold
	PostVerifySkaffold
	PreValidateEnvSources
	PostValidateEnvSources
	PreReconcileEnvs
	PostReconcileEnvs
	PreRenderFromComposeToK8sManifests
	PostRenderFromComposeToK8sManifests
	PreEnsureFirstInit
	PostEnsureFirstInit
	PreDetectSources
	PostDetectSources
	PreCreateManifest
	PostCreateManifest
	PreCreateOrUpdateSkaffoldManifest
	PostCreateOrUpdateSkaffoldManifest
	PrePrintSummary
	PostPrintSummary
	SecretsDetected
	DevLoopStarting
	DevLoopIterated
)

// newEventError returns an event error wrapping the original error
func newEventError(err error, event RunnerEvent) error {
	return errors.Errorf("%s\nwhen handling %s event", err.Error(), event)
}
