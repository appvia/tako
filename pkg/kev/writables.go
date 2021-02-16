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

import "path/filepath"

// Write writes the results
func (results WritableResults) Write() error {
	for _, result := range results {
		if err := result.Write(); err != nil {
			return err
		}
	}
	return nil
}

// Filename returns the filename for the writable result
func (r WritableResult) Filename() string {
	if len(r.FilePath) == 0 {
		return ""
	}
	return filepath.Base(r.FilePath)
}

// Write writes the WriterTo to the filepath
func (r WritableResult) Write() error {
	return WriteTo(r.FilePath, r.WriterTo)
}
