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
	"strconv"

	"github.com/r3labs/diff"
)

// newChangeset creates a changeset based on a diff.Changelog.
// A diff.Changelog is an ordered []diff.Changes slice produced from diffing two structs.
// E.g. Here's a diff.Change that updates ["services", 0, "labels", "kev.workload.liveness-probe-command"]:
// {
//    Type: "update",   //string
//    Path: {           // []string
//      "services",
//      "0",
//      "labels",
//      "kev.workload.liveness-probe-command"
//    },
//    From: "[\"CMD\", \"echo\", \"Define healthcheck command for service wordpress\"]",  // string
//    To: "[\"CMD\", \"curl\", \"localhost:80/healthy\"]"                                 // string
// }
//
// ENV VARS NOTE:
// The changeset deals with the docker-compose `environment` attribute as a special case:
// - Env vars in overlays override a project's docker-compose env vars.
// - A changeset will ONLY REMOVE an env var if it is removed from a project's docker-compose env vars.
// - A changeset will NOT update or create env vars in deployment environments.
// - To create useful diffs a project's docker-compose env vars will be taken into account.
//
func newChangeset(clog diff.Changelog) (changeset, error) {
	var verChanges []change
	volChgGroup := make(changeGroup)
	svcChgGroup := make(changeGroup)

	for _, e := range clog {
		switch e.Path[0] {
		case "version":
			change := change{
				update: e.Type == "update",
			}
			if e.To != nil {
				change.value = e.To.(string)
			}
			verChanges = append(verChanges, change)

		case "services":
			svcIndex, err := strconv.Atoi(e.Path[1])
			if err != nil {
				return changeset{}, err
			}

			// Do not append more changes for a service if the service is marked for deletion
			if isServiceAlreadyMarkedForDeletion(svcChgGroup, svcIndex) {
				continue
			}

			change := change{
				parent: e.Path[len(e.Path)-2],
				target: e.Path[len(e.Path)-1],
				index:  svcIndex,
				update: isServiceUpdateChange(e),
				create: isServiceCreateChange(e, svcChgGroup, svcIndex),
				delete: e.Type == "delete",
			}
			if e.To != nil {
				change.value = e.To.(string)
			}
			svcChgGroup[svcIndex] = append(svcChgGroup[svcIndex], change)

		case "volumes":
			volName := e.Path[1]

			// Do not append more changes for a volume if it's marked for deletion
			if isVolumeAlreadyMarkedForDeletion(volChgGroup, volName) {
				continue
			}

			change := change{
				parent: e.Path[len(e.Path)-2],
				target: e.Path[len(e.Path)-1],
				index:  volName,
				update: isVolumeUpdateChange(e),
				create: isVolumeCreateChange(e, volChgGroup, volName),
				delete: e.Type == "delete",
			}
			if e.To != nil {
				change.value = e.To.(string)
			}
			volChgGroup[volName] = append(volChgGroup[volName], change)
		}
	}

	return changeset{version: verChanges, services: svcChgGroup, volumes: volChgGroup}, nil
}

// changes returns a flat list of all available changes
func (cset changeset) changes() []change {
	var out []change
	for _, vc := range cset.version {
		out = append(out, vc)
	}
	for _, vc := range cset.services {
		out = append(out, vc...)
	}
	for _, vc := range cset.volumes {
		out = append(out, vc...)
	}
	return out
}

// HasNoPatches informs if a changeset has any patches to apply.
// A changeset may have changes but these may not result into valid patches.
func (cset changeset) HasNoPatches() bool {
	for _, chg := range cset.changes() {
		if chg.hasPatch() {
			return false
		}
	}
	return true
}

func (cset changeset) applyVersionPatchesIfAny(o *composeOverlay, reporter io.Writer) {
	for _, change := range cset.version {
		if change.hasPatch() {
			change.patchVersion(o, reporter)
		}
	}
}

func (cset changeset) applyServicesPatchesIfAny(o *composeOverlay, reporter io.Writer) {
	for _, group := range cset.services {
		for _, change := range group {
			if change.hasPatch() {
				change.patchService(o, reporter)
			}
		}
	}
}

func (cset changeset) applyVolumesPatchesIfAny(o *composeOverlay, reporter io.Writer) {
	for _, group := range cset.volumes {
		for _, change := range group {
			if change.hasPatch() {
				change.patchVolume(o, reporter)
			}
		}
	}
}

// hasPatch informs whether a change has a valid patch to apply or not
func (chg change) hasPatch() bool {
	return chg.create || chg.update || chg.delete
}

func (chg change) patchVersion(overlay *composeOverlay, reporter io.Writer) {
	if !chg.update {
		return
	}
	pre := overlay.Version
	overlay.Version = chg.value
	_, _ = reporter.Write([]byte(fmt.Sprintf(" → version updated, from:[%s] to:[%s]\n", pre, chg.value)))
}

func (chg change) patchService(overlay *composeOverlay, reporter io.Writer) {
	if chg.create {
		overlay.Services = append(overlay.Services, ServiceConfig{
			Labels: map[string]string{},
		})
		_, _ = reporter.Write([]byte(fmt.Sprintf(" → service [%s] added\n", chg.value)))
	}

	if chg.delete {
		switch {
		case chg.parent == "environment":
			delete(overlay.Services[chg.index.(int)].Environment, chg.target)
			_, _ = reporter.Write([]byte(fmt.Sprintf(" → service [%s], env var [%s] deleted\n", overlay.Services[chg.index.(int)].Name, chg.target)))
		default:
			deletedSvcName := overlay.Services[chg.index.(int)].Name
			overlay.Services = append(overlay.Services[:chg.index.(int)], overlay.Services[chg.index.(int)+1:]...)
			_, _ = reporter.Write([]byte(fmt.Sprintf(" → service [%s] deleted\n", deletedSvcName)))
		}
	}

	if chg.update {
		switch {
		case chg.target == "name":
			isUpdate := len(overlay.Services[chg.index.(int)].Name) > 0
			overlay.Services[chg.index.(int)].Name = chg.value
			if isUpdate {
				_, _ = reporter.Write([]byte(fmt.Sprintf(" → service name updated to: [%s]\n", chg.value)))
			}
		case chg.parent == "labels":
			pre, isUpdate := overlay.Services[chg.index.(int)].Labels[chg.target]
			overlay.Services[chg.index.(int)].Labels[chg.target] = chg.value
			if isUpdate {
				_, _ = reporter.Write([]byte(fmt.Sprintf(" → service [%s], label [%s] updated, from:[%s] to:[%s]\n", overlay.Services[chg.index.(int)].Name, chg.target, pre, chg.value)))
			}
		}
	}
}

func (chg change) patchVolume(overlay *composeOverlay, reporter io.Writer) {
	if chg.create {
		overlay.Volumes = Volumes{
			chg.index.(string): VolumeConfig{
				Labels: map[string]string{},
			},
		}
		_, _ = reporter.Write([]byte(fmt.Sprintf(" → volume [%s] added\n", chg.index.(string))))
	}

	if chg.delete {
		delete(overlay.Volumes, chg.index.(string))
		_, _ = reporter.Write([]byte(fmt.Sprintf(" → volume [%s] deleted\n", chg.index.(string))))
	}

	if chg.update {
		switch {
		case chg.parent == "labels":
			pre, isUpdate := overlay.Volumes[chg.index.(string)].Labels[chg.target]
			overlay.Volumes[chg.index.(string)].Labels[chg.target] = chg.value
			if isUpdate {
				_, _ = reporter.Write([]byte(fmt.Sprintf(" → volume [%s], label [%s] updated, from:[%s] to:[%s]\n", chg.index.(string), chg.target, pre, chg.value)))
			}
		}
	}
}

func isServiceAlreadyMarkedForDeletion(chgGroup changeGroup, index int) bool {
	group, ok := chgGroup[index]
	return ok == true && group[0].delete && group[0].target == "name"
}

func isServiceCreateChange(e diff.Change, chgGroup changeGroup, index int) bool {
	_, ok := chgGroup[index]
	parent := e.Path[len(e.Path)-2]

	// environment is a special case, see ENV VARS NOTE
	return e.Type == "create" && ok == false && parent != "environment"
}

func isServiceUpdateChange(e diff.Change) bool {
	parent := e.Path[len(e.Path)-2]

	// environment is a special case, see ENV VARS NOTE
	return (e.Type == "create" || e.Type == "update") && parent != "environment"
}

func isVolumeAlreadyMarkedForDeletion(chgGroup changeGroup, key string) bool {
	group, ok := chgGroup[key]
	return ok == true && group[0].delete
}

func isVolumeCreateChange(e diff.Change, chgGroup changeGroup, key string) bool {
	_, ok := chgGroup[key]
	return e.Type == "create" && ok == false
}

func isVolumeUpdateChange(e diff.Change) bool {
	return e.Type == "create" || e.Type == "update"
}
