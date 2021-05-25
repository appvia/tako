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

// MinifySvcExtension creates a minimal service extension configuration using the supplied src.
func MinifySvcExtension(src map[string]interface{}) (map[string]interface{}, error) {
	srcCfg, err := ParseSvcK8sConfigFromMap(src)
	if err != nil {
		return nil, err
	}

	var probeConfig ProbeConfig
	switch srcCfg.Workload.LivenessProbe.Type {
	case ProbeTypeExec.String():
		probeConfig = ProbeConfig{
			Exec: srcCfg.Workload.LivenessProbe.ProbeConfig.Exec,
		}
	case ProbeTypeHTTP.String():
		probeConfig = ProbeConfig{
			HTTP: srcCfg.Workload.LivenessProbe.ProbeConfig.HTTP,
		}
	case ProbeTypeTCP.String():
		probeConfig = ProbeConfig{
			TCP: srcCfg.Workload.LivenessProbe.ProbeConfig.TCP,
		}
	}

	return SvcK8sConfig{
		Workload: Workload{
			Replicas: srcCfg.Workload.Replicas,
			LivenessProbe: LivenessProbe{
				Type:        srcCfg.Workload.LivenessProbe.Type,
				ProbeConfig: probeConfig,
			},
		},
	}.Map()
}

// MinifyVolExtension creates a minimal volume extension configuration using the supplied src.
func MinifyVolExtension(src map[string]interface{}) (map[string]interface{}, error) {
	srcCfg, err := ParseVolK8sConfigFromMap(src)
	if err != nil {
		return nil, err
	}

	return VolK8sConfig{Size: srcCfg.Size}.Map()
}
