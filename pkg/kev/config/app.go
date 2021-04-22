/**
 * Copyright 2021 Appvia Ltd <info@appvia.io>
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
	"fmt"
	"strconv"
	"time"
)

var (
	// AppName is the app's name, can be overridden using ldflags.
	AppName = "kev"

	// Author is the prog author
	Author = "Appvia"

	// Email is the default email
	Email = "info@appvia.io"

	// Compiled in the time it was compiling
	Compiled = "0"

	// GitSHA is the sha this was built off
	GitSHA = "no gitsha provided"

	// GitBranch is the branch program was built off
	GitBranch = "no branch provided"

	// Release is the releasing version
	Release = "latest"

	// Tag is the release tag of the build
	Tag = ""
)

// Version returns the proxy version
func Version() string {
	tm, err := strconv.ParseInt(Compiled, 10, 64)
	if err != nil {
		return "unable to parse compiled time"
	}
	return fmt.Sprintf("%s %s (branch: %s, git+sha: %s, built: %s)",
		AppName,
		Release,
		GitBranch,
		GitSHA,
		time.Unix(tm, 0).Format("02-01-2006"),
	)
}
