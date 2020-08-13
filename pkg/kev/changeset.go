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

			_, ok := svcChgGroup[svcIndex]
			change := change{
				parent: e.Path[len(e.Path)-2],
				target: e.Path[len(e.Path)-1],
				index:  svcIndex,
				update: e.Type == "create" || e.Type == "update",
				create: e.Type == "create" && ok == false,
				delete: e.Type == "delete",
			}
			if e.To != nil {
				change.value = e.To.(string)
			}
			svcChgGroup[svcIndex] = append(svcChgGroup[svcIndex], change)

		case "volumes":
			volName := e.Path[1]
			_, ok := volChgGroup[volName]

			change := change{
				parent: e.Path[len(e.Path)-2],
				target: e.Path[len(e.Path)-1],
				index:  volName,
				update: e.Type == "create" || e.Type == "update",
				create: e.Type == "create" && ok == false,
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

func isServiceAlreadyMarkedForDeletion(chgGroup changeGroup, index int) bool {
	group, ok := chgGroup[index]
	return ok == true && group[0].delete && group[0].target == "name"
}

func (c changeset) applyVersionChangesIfAny(l *labels) {
	for _, change := range c.version {
		change.applyVersion(l)
	}
}

func (c changeset) applyServicesChangesIfAny(l *labels) {
	for _, group := range c.services {
		for _, change := range group {
			change.applyService(l)
		}
	}
}

func (c changeset) applyVolumesChangesIfAny(l *labels) {
	for _, group := range c.volumes {
		for _, change := range group {
			change.applyVolume(l)
		}
	}
}

func (c change) applyVersion(l *labels) {
	if !c.update {
		return
	}
	l.Version = c.value
}

func (c change) applyService(l *labels) {
	if c.create {
		l.Services = append(l.Services, ServiceConfig{
			Labels: map[string]string{},
		})
	}

	if c.delete {
		switch {
		case c.parent == "environment":
			delete(l.Services[c.index.(int)].Environment, c.target)
		default:
			l.Services = append(l.Services[:c.index.(int)], l.Services[c.index.(int)+1:]...)
		}
	}

	if c.update {
		switch {
		case c.target == "name":
			l.Services[c.index.(int)].Name = c.value
		case c.parent == "labels":
			l.Services[c.index.(int)].Labels[c.target] = c.value
		}
	}
}

func (c change) applyVolume(l *labels) {
	if c.create {
		l.Volumes = Volumes{
			c.index.(string): VolumeConfig{
				Labels: map[string]string{},
			},
		}
	}

	if c.delete {
		delete(l.Volumes, c.index.(string))
	}

	if c.update {
		switch {
		case c.parent == "labels":
			l.Volumes[c.index.(string)].Labels[c.target] = c.value
		}
	}
}
