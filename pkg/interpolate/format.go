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

package interpolate

import (
	"fmt"
	"regexp"
)

// WithQuoteDecimalValue formats a decimal value to '<decimal_value>'
var WithQuoteDecimalValue Formatter = func(value []byte) []byte {
	result := value
	pattern := regexp.MustCompile(`\d+(\.\d{1,2})+`)
	if pattern.Match(value) {
		result = []byte(fmt.Sprintf("\"%s\"", value))
	}
	return result
}
