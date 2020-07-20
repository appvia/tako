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

package app

import "path"

// RootDir returns a loaded definition's root directory
func (def *Definition) RootDir() string {
	return baseDir
}

// RootDir returns a loaded definition's work directory
func (def *Definition) WorkDir() string {
	return workDir
}

// RootDir returns a loaded definition's work directory path including root path
func (def *Definition) WorkPath() string {
	return path.Join(def.RootDir(), def.WorkDir())
}

// BuildDir returns a loaded definition's build directory
func (def *Definition) BuildDir() string {
	return buildDir
}

// BuildDir returns a loaded definition's build path including the root & work paths
func (def *Definition) BuildPath() string {
	return path.Join(def.RootDir(), def.WorkDir(), def.BuildDir())
}
