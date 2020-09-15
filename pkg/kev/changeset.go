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
	"io"
	"reflect"

	"github.com/appvia/kev/pkg/kev/config"
)

const (
	CREATE = "create"
	UPDATE = "update"
	DELETE = "delete"
)

// newChangeset detects all changes between a destination overlay and source overlay.
// A change is either a create, update or delete event.
// A change targets an overlay's version, services or volumes and it's properties will depend on the actual target.
// Example: here's a Change that creates a new service:
// {
//    Type: "create",   //string
//    Value: srcSvc,    //interface{} in this case: ServiceConfig
// }
// Example: here's a Change that updates a service's label:
// {
// 		Type:   "update",                 //string
// 		Index:  index,                    // interface{} in this case: int
// 		Parent: "labels",                 // string
// 		Target: config.LabelServiceType,  // string
// 		Value:  srcSvc.GetLabels()[config.LabelServiceType], // interface{} in this case: string
// }
//
// ENV VARS NOTE:
// The changeset deals with the docker-compose `environment` attribute as a special case:
// - Env vars in overlays override a project's docker-compose env vars.
// - A changeset will ONLY REMOVE an env var if it is removed from a project's docker-compose env vars.
// - A changeset will NOT update or create env vars in deployment environments.
// - To create useful diffs a project's docker-compose env vars will be taken into account.
func newChangeset(dst *composeOverlay, src *composeOverlay) changeset {
	cset := changeset{}
	detectVersionUpdate(dst, src, &cset)
	detectServicesCreate(dst, src, &cset)
	detectServicesDelete(dst, src, &cset)
	detectServicesEnvironmentDelete(dst, src, &cset)
	detectServicesUpdate(dst, src, &cset)
	detectVolumesCreate(dst, src, &cset)
	detectVolumesDelete(dst, src, &cset)
	return cset
}

func detectVersionUpdate(dst *composeOverlay, src *composeOverlay, cset *changeset) {
	if dst.Version != src.Version {
		cset.version = change{Value: src.Version, Type: UPDATE, Target: "version"}
	}
}

func detectServicesCreate(dst *composeOverlay, src *composeOverlay, cset *changeset) {
	dstSvcSet := dst.Services.Set()
	for _, srcSvc := range src.Services {
		if !dstSvcSet[srcSvc.Name] {
			cset.services = append(cset.services, change{
				Type:  CREATE,
				Value: srcSvc.minusEnvVars(),
			})
		}
	}
}

func detectServicesDelete(dst *composeOverlay, src *composeOverlay, cset *changeset) {
	srcSvcSet := src.Services.Set()
	for index, dstSvc := range dst.Services {
		if !srcSvcSet[dstSvc.Name] {
			cset.services = append(cset.services, change{
				Type:  DELETE,
				Index: index,
			})
		}
	}
}

func detectServicesEnvironmentDelete(dst *composeOverlay, src *composeOverlay, cset *changeset) {
	srcSvcMapping := src.Services.Map()
	for index, dstSvc := range dst.Services {
		srcSvc, ok := srcSvcMapping[dstSvc.Name]
		if !ok {
			continue
		}
		for envVarKey := range dstSvc.Environment {
			if _, ok := srcSvc.Environment[envVarKey]; !ok {
				cset.services = append(cset.services, change{
					Type:   DELETE,
					Index:  index,
					Parent: "environment",
					Target: envVarKey,
				})
			}
		}
	}
}

func detectServicesUpdate(dst *composeOverlay, src *composeOverlay, cset *changeset) {
	srcSvcMapping := src.Services.Map()
	for index, dstSvc := range dst.Services {
		srcSvc, ok := srcSvcMapping[dstSvc.Name]
		if !ok {
			continue
		}

		if srcSvc.GetLabels()[config.LabelServiceType] != dstSvc.GetLabels()[config.LabelServiceType] {
			cset.services = append(cset.services, change{
				Type:   UPDATE,
				Index:  index,
				Parent: "labels",
				Target: config.LabelServiceType,
				Value:  srcSvc.GetLabels()[config.LabelServiceType],
			})
		}
	}
}

func detectVolumesCreate(dst *composeOverlay, src *composeOverlay, cset *changeset) {
	for srcVolKey, srcVolConfig := range src.Volumes {
		if _, ok := dst.Volumes[srcVolKey]; !ok {
			cset.volumes = append(cset.volumes, change{
				Type:  CREATE,
				Index: srcVolKey,
				Value: srcVolConfig,
			})
		}
	}
}

func detectVolumesDelete(dst *composeOverlay, src *composeOverlay, cset *changeset) {
	for dstVolKey := range dst.Volumes {
		if _, ok := src.Volumes[dstVolKey]; !ok {
			cset.volumes = append(cset.volumes, change{
				Type:  DELETE,
				Index: dstVolKey,
			})
		}
	}
}

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

func (cset changeset) applyVersionPatchesIfAny(o *composeOverlay, reporter io.Writer) {
	chg := cset.version
	if reflect.DeepEqual(chg, change{}) {
		return
	}
	chg.patchVersion(o, reporter)
}

func (cset changeset) applyServicesPatchesIfAny(o *composeOverlay, reporter io.Writer) {
	for _, change := range cset.services {
		change.patchService(o, reporter)
	}
}

func (cset changeset) applyVolumesPatchesIfAny(o *composeOverlay, reporter io.Writer) {
	for _, change := range cset.volumes {
		change.patchVolume(o, reporter)
	}
}

func (chg change) patchVersion(overlay *composeOverlay, reporter io.Writer) {
	if chg.Type != UPDATE {
		return
	}
	pre := overlay.Version
	newValue := chg.Value.(string)
	overlay.Version = newValue
	_, _ = reporter.Write([]byte(fmt.Sprintf(" → version updated, from:[%s] to:[%s]\n", pre, newValue)))
}

func (chg change) patchService(overlay *composeOverlay, reporter io.Writer) {
	switch chg.Type {
	case CREATE:
		newValue := chg.Value.(ServiceConfig)
		overlay.Services = append(overlay.Services, newValue)
		_, _ = reporter.Write([]byte(fmt.Sprintf(" → service [%s] added\n", newValue.Name)))
	case DELETE:
		switch {
		case chg.Parent == "environment":
			delete(overlay.Services[chg.Index.(int)].Environment, chg.Target)
			_, _ = reporter.Write([]byte(fmt.Sprintf(" → service [%s], env var [%s] deleted\n", overlay.Services[chg.Index.(int)].Name, chg.Target)))
		default:
			deletedSvcName := overlay.Services[chg.Index.(int)].Name
			overlay.Services = append(overlay.Services[:chg.Index.(int)], overlay.Services[chg.Index.(int)+1:]...)
			_, _ = reporter.Write([]byte(fmt.Sprintf(" → service [%s] deleted\n", deletedSvcName)))
		}
	case UPDATE:
		if chg.Parent == "labels" {
			pre, canUpdate := overlay.Services[chg.Index.(int)].Labels[chg.Target]
			newValue := chg.Value.(string)
			overlay.Services[chg.Index.(int)].Labels[chg.Target] = newValue
			if canUpdate {
				_, _ = reporter.Write([]byte(fmt.Sprintf(" → service [%s], label [%s] updated, from:[%s] to:[%s]\n", overlay.Services[chg.Index.(int)].Name, chg.Target, pre, newValue)))
			}
		}
	}
}

func (chg change) patchVolume(overlay *composeOverlay, reporter io.Writer) {
	switch chg.Type {
	case CREATE:
		overlay.Volumes[chg.Index.(string)] = chg.Value.(VolumeConfig)
		_, _ = reporter.Write([]byte(fmt.Sprintf(" → volume [%s] added\n", chg.Index.(string))))
	case DELETE:
		delete(overlay.Volumes, chg.Index.(string))
		_, _ = reporter.Write([]byte(fmt.Sprintf(" → volume [%s] deleted\n", chg.Index.(string))))
	}
}
