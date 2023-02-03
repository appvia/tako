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

package tako

import (
	"github.com/appvia/tako/pkg/tako/config"
	"github.com/pkg/errors"
)

func newVolumeConfig(name string, p *ComposeProject) (VolumeConfig, error) {
	ext := p.Volumes[name].Extensions

	_, err := config.ParseVolK8sConfigFromMap(ext)
	if err != nil {
		return VolumeConfig{}, errors.Wrapf(err, "when parsing vol %s extensions", name)
	}

	return VolumeConfig{Extensions: ext}, nil
}
