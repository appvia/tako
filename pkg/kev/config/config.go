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

import (
	"io/ioutil"
	"path"

	"github.com/goccy/go-yaml"
	yaml3 "gopkg.in/yaml.v3"
)

// New creates and returns an app config
func New() *Config {
	return &Config{
		Name:        "Change me",
		Description: "Change me...",
		Components:  make(map[string]Component),
	}
}

// Bytes representation of application configuration
func (c *Config) Bytes() ([]byte, error) {
	bytes, err := yaml.Marshal(c)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// GetBaseConfig returns the base app config
func GetBaseConfig(root string) (*Config, error) {
	configPath := path.Join(root, "config.yaml")
	baseConfigContent, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	baseConfig := &Config{}
	if err = yaml3.Unmarshal(baseConfigContent, baseConfig); err != nil {
		return nil, err
	}

	return baseConfig, nil
}

// GetEnvConfig returns a deployment env config
func GetEnvConfig(root string, env string) (Config, error) {
	configPath := path.Join(root, env, "config.yaml")
	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}
	envConfig := Config{}
	if err = yaml3.Unmarshal(content, &envConfig); err != nil {
		return Config{}, err
	}
	return envConfig, nil
}
