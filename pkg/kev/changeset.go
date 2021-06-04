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
	"fmt"
	"reflect"

	"github.com/appvia/kev/pkg/kev/config"
	"github.com/appvia/kev/pkg/kev/log"
)

const (
	CREATE = "create"
	UPDATE = "update"
	DELETE = "delete"
)

// changes returns a flat list of all available changes
func (cset changeset) changes() []change {
	var out []change
	if !reflect.DeepEqual(cset.version, change{}) {
		out = append(out, cset.version)
	}
	out = append(out, cset.services...)
	out = append(out, cset.volumes...)
	return out
}

// HasNoPatches informs if a changeset has any patches to apply.
func (cset changeset) HasNoPatches() bool {
	return len(cset.changes()) <= 0
}

func (cset changeset) applyVersionPatchesIfAny(o *composeOverride) string {
	chg := cset.version
	if reflect.DeepEqual(chg, change{}) {
		return ""
	}
	return chg.patchVersion(o)
}

func (cset changeset) applyServicesPatchesIfAny(o *composeOverride) ([]string, error) {
	var out []string
	for _, change := range cset.services {
		patchDetails, err := change.patchService(o)
		if err != nil {
			return nil, err
		}
		out = append(out, patchDetails)
	}
	return out, nil
}

func (cset changeset) applyVolumesPatchesIfAny(o *composeOverride) ([]string, error) {
	var out []string
	for _, change := range cset.volumes {
		patchDetails, err := change.patchVolume(o)
		if err != nil {
			return nil, err
		}
		out = append(out, patchDetails)
	}
	return out, nil
}

func (chg change) patchVersion(override *composeOverride) string {
	if chg.Type != UPDATE {
		return ""
	}
	pre := override.Version
	newValue := chg.Value.(string)
	override.Version = newValue

	msg := fmt.Sprintf("version %s updated to %s", pre, newValue)
	log.Debugf(msg)
	return msg
}

func (chg change) patchService(override *composeOverride) (string, error) {
	switch chg.Type {
	case CREATE:
		newValue := chg.Value.(ServiceConfig)

		minified, err := config.MinifySvcK8sExtension(newValue.Extensions)
		if err != nil {
			return "", err
		}

		newValue.Extensions[config.K8SExtensionKey] = minified
		override.Services = append(override.Services, newValue)

		msg := fmt.Sprintf("added service: %s", newValue.Name)
		log.Debugf(msg)
		return msg, nil
	case DELETE:
		switch {
		case chg.Parent == "environment":
			delete(override.Services[chg.Index.(int)].Environment, chg.Target)
			msg := fmt.Sprintf("removed env var: %s from service %s", chg.Target, override.Services[chg.Index.(int)].Name)
			log.Debugf(msg)
			return msg, nil
		default:
			deletedSvcName := override.Services[chg.Index.(int)].Name
			override.Services = append(override.Services[:chg.Index.(int)], override.Services[chg.Index.(int)+1:]...)
			msg := fmt.Sprintf("removed service: %s", deletedSvcName)
			log.Debugf(msg)
			return msg, nil
		}
	case UPDATE:
		switch chg.Parent {
		case "extensions":
			svc := override.Services[chg.Index.(int)]
			svcName := svc.Name

			newValue, ok := chg.Value.(map[string]interface{})
			if !ok {
				log.Debugf("unable to update service [%s], invalid value %+v", svcName, newValue)
				return "", nil
			}

			if svc.Extensions == nil {
				svc.Extensions = make(map[string]interface{})
			}

			svc.Extensions[config.K8SExtensionKey] = newValue
			log.Debugf("service [%s] extensions updated to %+v", svcName, newValue)
		}
	}
	return "", nil
}

func (chg change) patchVolume(override *composeOverride) (string, error) {
	switch chg.Type {
	case CREATE:
		newValue := chg.Value.(VolumeConfig)

		minified, err := config.MinifyVolK8sExtension(newValue.Extensions)
		if err != nil {
			return "", err
		}

		newValue.Extensions[config.K8SExtensionKey] = minified
		override.Volumes[chg.Index.(string)] = newValue

		msg := fmt.Sprintf("added volume: %s", chg.Index.(string))
		log.Debugf(msg)
		return msg, nil
	case DELETE:
		delete(override.Volumes, chg.Index.(string))
		msg := fmt.Sprintf("removed volume: %s", chg.Index.(string))
		log.Debugf(msg)
		return msg, nil
	}
	return "", nil
}
