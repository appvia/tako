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
	kmd "github.com/appvia/komando"
	composego "github.com/compose-spec/compose-go/types"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
)

// getService retrieves the specific service by name from the override's services.
func (o *composeOverride) getService(name string) (ServiceConfig, error) {
	for _, s := range o.Services {
		if s.Name == name {
			return s, nil
		}
	}
	return ServiceConfig{}, fmt.Errorf("no such service: %s", name)
}

// getVolume retrieves a specific volume by name from the override's volumes.
func (o *composeOverride) getVolume(name string) (VolumeConfig, error) {
	for k, v := range o.Volumes {
		if k == name {
			return v, nil
		}
	}
	return VolumeConfig{}, fmt.Errorf("no such volume: %s", name)
}

// toBaseLabels returns a copy of the composeOverride with condensed base labels for services and volumes.
func (o *composeOverride) toBaseLabels() *composeOverride {
	var services Services
	volumes := Volumes{}

	for _, svcConfig := range o.Services {
		services = append(services, svcConfig.condenseLabels(config.BaseServiceLabels))
	}
	for key, volConfig := range o.Volumes {
		volumes[key] = volConfig.condenseLabels(config.BaseVolumeLabels)
	}

	return &composeOverride{Version: o.Version, Services: services, Volumes: volumes}
}

// toLabelsMatching condenses an override's labels to the same label keys found in the provided override.
func (o *composeOverride) toLabelsMatching(other *composeOverride) *composeOverride {
	services := o.servicesWithLabelsMatching(other)
	volumes := o.volumesWithLabelsMatching(other)
	return &composeOverride{Version: o.Version, Services: services, Volumes: volumes, UI: o.UI}
}

func (o *composeOverride) servicesWithLabelsMatching(other *composeOverride) Services {
	var services Services
	for _, svc := range o.Services {
		otherSvc, err := other.getService(svc.Name)
		if err != nil {
			services = append(services, svc)
			continue
		}
		services = append(services, svc.condenseLabels(keys(otherSvc.Labels)))
	}
	return services
}

func (o *composeOverride) volumesWithLabelsMatching(other *composeOverride) Volumes {
	volumes := Volumes{}
	for volKey, volConfig := range o.Volumes {
		otherVol, err := other.getVolume(volKey)
		if err != nil {
			volumes[volKey] = volConfig
			continue
		}
		volumes[volKey] = volConfig.condenseLabels(keys(otherVol.Labels))
	}
	return volumes
}

// expandLabelsFrom returns a copy of the compose override
// filling in gaps in services and volumes labels (keys and values) using the provided override.
func (o *composeOverride) expandLabelsFrom(other *composeOverride) *composeOverride {
	services := o.servicesLabelsExpandedFrom(other)
	volumes := o.volumesLabelsExpandedFrom(other)
	return &composeOverride{Version: o.Version, Services: services, Volumes: volumes}
}

func (o *composeOverride) servicesLabelsExpandedFrom(other *composeOverride) Services {
	var out Services
	for _, otherSvc := range other.Services {
		dstSvc, err := o.getService(otherSvc.Name)
		if err != nil {
			continue
		}
		for key, value := range otherSvc.GetLabels() {
			if _, ok := dstSvc.Labels[key]; !ok {
				dstSvc.Labels[key] = value
			}
		}
		out = append(out, dstSvc)
	}
	return out
}

func (o *composeOverride) volumesLabelsExpandedFrom(other *composeOverride) Volumes {
	out := Volumes{}
	for otherVolKey, otherVolConfig := range other.Volumes {
		dstVol, err := o.getVolume(otherVolKey)
		if err != nil {
			continue
		}

		for key, value := range otherVolConfig.Labels {
			if _, ok := dstVol.Labels[key]; !ok {
				dstVol.Labels[key] = value
			}
		}
		out[otherVolKey] = dstVol
	}
	return out
}

// diffAndPatch detects and patches all changes between a destination override and the current override.
// A change is either a create, update or delete event.
// A change targets an override's version, services or volumes and its properties will depend on the actual target.
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
// - Env vars specified in docker compose overrides modify a project's docker-compose env vars.
// - A changeset will ONLY REMOVE an env var if it is removed from a project's docker-compose env vars.
// - A changeset will NOT update or create env vars in an environment specific docker compose override file.
// - To create useful diffs the project's base docker-compose env vars will be taken into account.
func (o *composeOverride) diffAndPatch(dst *composeOverride) {
	o.detectAndPatchVersionUpdate(dst)
	o.detectAndPatchServicesCreate(dst)
	o.detectAndPatchServiceExtensions(dst)
	o.detectAndPatchServicesDelete(dst)
	o.detectAndPatchServicesEnvironmentDelete(dst)
	o.detectAndPatchVolumesCreate(dst)
	o.detectAndPatchVolumesDelete(dst)
}

func (o *composeOverride) detectAndPatchServiceExtensions(dst *composeOverride) {
	sg := o.UI.StepGroup()
	defer sg.Done()
	step := sg.Add("Detecting service extension updates")

	cset := changeset{}
	for i, srcSvc := range o.Services {
		dstSvc, err := dst.getService(srcSvc.Name)
		if err != nil {
			continue
		}

		k8sconf, err := config.ParseK8SCfgFromMap(dstSvc.Extensions)
		if err != nil {
			log.Debugf("unable to convert extensions to k8s configuration %s", err.Error())
			continue
		}

		k8sconf, err = k8sconf.Merge(k8sconf)
		if err != nil {
			log.Debugf("unable to merge extensions to k8s configuration %s", err.Error())
			continue
		}

		k8smap, err := k8sconf.ToMap()
		if err != nil {
			log.Debugf("unable to convert k8s configuration to map %s", err.Error())
			continue
		}

		if reflect.DeepEqual(k8smap, dstSvc.Extensions) {
			continue
		}

		cset.services = append(cset.services, change{
			Type:   UPDATE,
			Parent: "extensions",
			Index:  i,
			Value:  k8smap,
		})
	}

	if cset.HasNoPatches() {
		step.Success("No extension updates detected")
		return
	}

	msgs := cset.applyServicesPatchesIfAny(dst)
	step.Success("Applied extension updates")
	for _, msg := range msgs {
		o.UI.Output(msg, kmd.WithStyle(kmd.LogStyle),
			kmd.WithIndentChar(kmd.LogIndentChar),
			kmd.WithIndent(3))
	}
}

func (o *composeOverride) detectAndPatchVersionUpdate(dst *composeOverride) {
	sg := o.UI.StepGroup()
	defer sg.Done()
	step := sg.Add("Detecting version update")

	cset := changeset{}
	if dst.Version != o.Version {
		cset.version = change{Value: o.Version, Type: UPDATE, Target: "version"}
	}

	if cset.HasNoPatches() {
		step.Success("No version update detected")
		return
	}
	msg := cset.applyVersionPatchesIfAny(dst)
	step.Success("Applied version update")
	o.UI.Output(msg, kmd.WithStyle(kmd.LogStyle),
		kmd.WithIndentChar(kmd.LogIndentChar),
		kmd.WithIndent(3))
}

func (o *composeOverride) detectAndPatchServicesCreate(dst *composeOverride) {
	sg := o.UI.StepGroup()
	defer sg.Done()
	step := sg.Add("Detecting service additions")

	cset := changeset{}
	dstSvcSet := dst.Services.Set()
	for _, srcSvc := range o.Services {
		if !dstSvcSet[srcSvc.Name] {
			cset.services = append(cset.services, change{
				Type:  CREATE,
				Value: srcSvc.minusEnvVars(),
			})
		}
	}
	if cset.HasNoPatches() {
		step.Success("No service additions detected")
		return
	}

	msgs := cset.applyServicesPatchesIfAny(dst)
	step.Success("Applied service additions")
	for _, msg := range msgs {
		o.UI.Output(msg, kmd.WithStyle(kmd.LogStyle),
			kmd.WithIndentChar(kmd.LogIndentChar),
			kmd.WithIndent(3))
	}
}

func (o *composeOverride) detectAndPatchServicesDelete(dst *composeOverride) {
	sg := o.UI.StepGroup()
	defer sg.Done()
	step := sg.Add("Detecting service removals")

	cset := changeset{}
	srcSvcSet := o.Services.Set()
	for index, dstSvc := range dst.Services {
		if !srcSvcSet[dstSvc.Name] {
			cset.services = append(cset.services, change{
				Type:  DELETE,
				Index: index,
			})
		}
	}

	if cset.HasNoPatches() {
		step.Success("No service removals detected")
		return
	}

	msgs := cset.applyServicesPatchesIfAny(dst)
	step.Success("Applied service removals")
	for _, msg := range msgs {
		o.UI.Output(msg, kmd.WithStyle(kmd.LogStyle),
			kmd.WithIndentChar(kmd.LogIndentChar),
			kmd.WithIndent(3))
	}
}

func (o *composeOverride) detectAndPatchServicesEnvironmentDelete(dst *composeOverride) {
	sg := o.UI.StepGroup()
	defer sg.Done()
	step := sg.Add("Detecting env var removals")

	cset := changeset{}
	srcSvcMapping := o.Services.Map()
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

	if cset.HasNoPatches() {
		step.Success("No env var removals detected")
		return
	}

	msgs := cset.applyServicesPatchesIfAny(dst)
	step.Success("Applied env var removals")
	for _, msg := range msgs {
		o.UI.Output(msg, kmd.WithStyle(kmd.LogStyle),
			kmd.WithIndentChar(kmd.LogIndentChar),
			kmd.WithIndent(3))
	}
}

func (o *composeOverride) detectAndPatchVolumesCreate(dst *composeOverride) {
	sg := o.UI.StepGroup()
	defer sg.Done()
	step := sg.Add("Detecting volume additions")

	cset := changeset{}
	for srcVolKey, srcVolConfig := range o.Volumes {
		if _, ok := dst.Volumes[srcVolKey]; !ok {
			cset.volumes = append(cset.volumes, change{
				Type:  CREATE,
				Index: srcVolKey,
				Value: srcVolConfig,
			})
		}
	}

	if cset.HasNoPatches() {
		step.Success("No volume additions detected")
		return
	}

	msgs := cset.applyVolumesPatchesIfAny(dst)
	step.Success("Applied volume additions")
	for _, msg := range msgs {
		o.UI.Output(msg, kmd.WithStyle(kmd.LogStyle),
			kmd.WithIndentChar(kmd.LogIndentChar),
			kmd.WithIndent(3))
	}
}

func (o *composeOverride) detectAndPatchVolumesDelete(dst *composeOverride) {
	sg := o.UI.StepGroup()
	defer sg.Done()
	step := sg.Add("Detecting volume removals")

	cset := changeset{}
	for dstVolKey := range dst.Volumes {
		if _, ok := o.Volumes[dstVolKey]; !ok {
			cset.volumes = append(cset.volumes, change{
				Type:  DELETE,
				Index: dstVolKey,
			})
		}
	}

	if cset.HasNoPatches() {
		step.Success("No volume removals detected")
		return
	}

	msgs := cset.applyVolumesPatchesIfAny(dst)
	step.Success("Applied volume removals")
	for _, msg := range msgs {
		o.UI.Output(msg, kmd.WithStyle(kmd.LogStyle),
			kmd.WithIndentChar(kmd.LogIndentChar),
			kmd.WithIndent(3))
	}
}

// mergeInto merges an override onto a compose project.
// For env vars, it enforces the expected docker-compose CLI behaviour.
func (o *composeOverride) mergeInto(p *ComposeProject) error {
	if err := o.mergeServicesInto(p); err != nil {
		return errors.Wrap(err, "cannot merge services into project")
	}
	if err := o.mergeVolumesInto(p); err != nil {
		return errors.Wrap(err, "cannot merge volumes into project")
	}
	return nil
}

func (o *composeOverride) mergeServicesInto(p *ComposeProject) error {
	var overridden composego.Services
	for _, override := range o.Services {
		base, err := p.GetService(override.Name)
		if err != nil {
			return err
		}

		envVarsFromNilToBlankInService(base)

		if err := mergo.Merge(&base.Labels, &override.Labels, mergo.WithOverride); err != nil {
			return errors.Wrapf(err, "cannot merge labels for service %s", override.Name)
		}
		if err := mergo.Merge(&base.Extensions, &override.Extensions, mergo.WithOverride); err != nil {
			return errors.Wrapf(err, "cannot merge extensions for service %s", override.Name)
		}
		if err := mergo.Merge(&base.Environment, &override.Environment, mergo.WithOverride); err != nil {
			return errors.Wrapf(err, "cannot merge env vars for service %s", override.Name)
		}
		overridden = append(overridden, base)
	}
	p.Services = overridden
	return nil
}

func (o *composeOverride) mergeVolumesInto(p *ComposeProject) error {
	for name, override := range o.Volumes {
		base, ok := p.Volumes[name]
		if !ok {
			return fmt.Errorf("could not find volume %s", override.Name)
		}

		if err := mergo.Merge(&base.Labels, &override.Labels, mergo.WithOverwriteWithEmptyValue); err != nil {
			return errors.Wrapf(err, "cannot merge labels for volume %s", name)
		}
		p.Volumes[name] = base
	}
	return nil
}
