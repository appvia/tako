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

package interpolate

func NewTarget() *Target {
	return &Target{}
}

func (t *Target) Content(c []byte) *Target {
	t.raw = c
	return t
}

func (t *Target) Prepare(transform ...func(data []byte) ([]byte, error)) *Target {
	t.transforms = append(t.transforms, transform...)
	return t
}

func (t *Target) Resolver(resolver Resolver) *Target {
	t.resolver = resolver
	return t
}

func (t *Target) Interpolate(data []byte, f ...Formatter) ([]byte, error) {
	if len(t.prepared) < 1 {
		err := t.prepData()
		if err != nil {
			return nil, err
		}
	}

	return t.resolver.Resolve(data, t.prepared, f...)
}

func (t *Target) prepData() error {
	data, err := runTransforms(t.raw, t.transforms)
	if err != nil {
		return err
	}
	t.prepared = data
	return nil
}

func runTransforms(data []byte, transforms []func(data []byte) ([]byte, error)) ([]byte, error) {
	for _, t := range transforms {
		var err error
		data, err = t(data)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}
