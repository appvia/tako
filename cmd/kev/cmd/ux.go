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

package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

func getReporter(onScreen bool) io.Writer {
	out := ioutil.Discard
	if onScreen {
		out = os.Stdout
	}
	return out
}

func displayError(cmdName string, err error) error {
	_, _ = os.Stdout.Write([]byte("⨯ " + cmdName + "\n"))
	return fmt.Errorf(" → Error: %s", err)
}

func displayInitSuccess(w io.Writer, files []skippableFile) {
	_, _ = w.Write([]byte("✓ Init\n"))
	for _, file := range files {
		msg := fmt.Sprintf(" → Creating %s ... Done\n", file.FileName)

		if file.Updated {
			msg = fmt.Sprintf(" → Updating %s ... Done\n", file.FileName)
		}

		if file.Skipped {
			msg = fmt.Sprintf(" → %s already exists ... Skipping\n", file.FileName)
		}
		_, _ = w.Write([]byte(msg))
	}
}
