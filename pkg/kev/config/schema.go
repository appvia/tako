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

// VolumesSchema defines validation rules for volume labels
var VolumesSchema = map[string]interface{}{
	"$schema": "http://json-schema.org/draft/2019-09/schema#",
	"$id":     "http://appvia.io/schemas/kev-service-labels-schema.json",
	"type":    "object",
	"properties": map[string]interface{}{
		LabelVolumeSize:         map[string]interface{}{"type": "string", "pattern": `^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$`},
		LabelVolumeStorageClass: map[string]interface{}{"type": "string", "pattern": `^[a-zA-Z0-9._-]+$`},
		LabelVolumeSelector:     map[string]interface{}{"type": "string", "pattern": `^[a-zA-Z0-9._-]+$`},
	},
	"required":             []string{LabelVolumeSize},
	"additionalProperties": false,
}
