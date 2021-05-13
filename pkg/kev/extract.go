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

package kev

import (
	"strconv"

	"github.com/appvia/kev/pkg/kev/config"

	composego "github.com/compose-spec/compose-go/types"
)

//TODO: Remove once all functions have been moved over.
// extractWorkloadRollingUpdatePolicy extracts deployment's rolling update policy.
func extractWorkloadRollingUpdatePolicy(source composego.ServiceConfig, target *ServiceConfig) {
	if source.Deploy != nil && source.Deploy.UpdateConfig != nil {
		value := strconv.FormatUint(*source.Deploy.UpdateConfig.Parallelism, 10)
		target.Labels.Add(config.LabelWorkloadRollingUpdateMaxSurge, value)
	}
}
