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

package transform

import (
	"fmt"
	"log"

	"github.com/appvia/kube-devx/pkg/kev/defaults"
	"github.com/compose-spec/compose-go/loader"
	compose "github.com/compose-spec/compose-go/types"
	"github.com/goccy/go-yaml"
)

// Transform is a transform func type.
// Documents how a transform func should be created.
// Useful as a function param for functions that accept transforms.
type Transform func(data []byte) ([]byte, error)

// UnmarshallGeneral deserializes a []byte into an map[string]interface{}
func UnmarshallGeneral(data []byte) (map[string]interface{}, error) {
	var out map[string]interface{}
	err := yaml.Unmarshal(data, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UnmarshallComposeConfig deserializes a []]byte into *compose.Config
func UnmarshallComposeConfig(data []byte) (*compose.Config, error) {
	source, err := UnmarshallGeneral(data)
	if err != nil {
		return nil, err
	}

	return loader.Load(compose.ConfigDetails{
		WorkingDir: ".",
		ConfigFiles: []compose.ConfigFile{
			{
				Filename: "temp-file",
				Config:   source,
			},
		},
	})
}

// DeployWithDefaults attaches a deploy block with presets to any service
// missing a deploy block.
func DeployWithDefaults(data []byte) ([]byte, error) {
	log.Println("Transform: DeployWithDefaults")

	x, err := UnmarshallComposeConfig(data)
	if err != nil {
		return []byte{}, err
	}

	var updated compose.Services
	err = x.WithServices(x.ServiceNames(), func(svc compose.ServiceConfig) error {
		if svc.Deploy == nil {
			svc.Deploy = defaults.Deploy()
		}
		updated = append(updated, svc)
		return nil
	})
	if err != nil {
		return []byte{}, err
	}

	x.Services = nil
	x.Services = updated
	return yaml.Marshal(x)
}

// HealthCheckBase attaches a base healthcheck  block with placeholders to be updated by users
// to any service missing a healthcheck block.
func HealthCheckBase(data []byte) ([]byte, error) {
	log.Println("Transform: HealthCheckBase")

	x, err := UnmarshallComposeConfig(data)
	if err != nil {
		return []byte{}, err
	}

	var updated compose.Services
	err = x.WithServices(x.ServiceNames(), func(svc compose.ServiceConfig) error {
		if svc.HealthCheck == nil {
			svc.HealthCheck = defaults.HealthCheck(svc.Name)
		}
		updated = append(updated, svc)
		return nil
	})
	if err != nil {
		return []byte{}, err
	}

	x.Services = nil
	x.Services = updated
	return yaml.Marshal(x)
}

// Echo can be used to view data at different stages of
// a transform pipeline.
func Echo(data []byte) ([]byte, error) {
	log.Println("Transform: Echo")
	x, err := UnmarshallComposeConfig(data)
	if err != nil {
		return []byte{}, err
	}
	fmt.Println(string(data))
	return yaml.Marshal(x)
}
