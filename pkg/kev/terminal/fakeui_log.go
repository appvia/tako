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

package terminal

import "fmt"

const (
	LogStepSuccess = "Success"
	LogStepWarning = "Warning"
	LogStepError   = "Error"
)

type logOpType uint

const (
	headerOp logOpType = iota
	outputOp
	namedValuesOp
	stepOp
)

type UILog interface {
	NextHeader() map[string][]string
	NextOutput() map[string][]string
	NextStep() map[string][]string
	LastHeader() map[string][]string
	LastOutput() map[string][]string
	LastStep() map[string][]string
	Reset()
}

type fakeUILog struct {
	data           map[logOpType][]map[string][]string
	nextHeader     int
	nextStep       int
	nextOutput     int
	nextNamedValue int
}

func newFakeUILog() *fakeUILog {
	return &fakeUILog{data: make(map[logOpType][]map[string][]string)}
}

func (l *fakeUILog) logStepStop(target string, a []interface{}) {
	configured := []string{}
	if len(a) > 0 {
		configured = append(configured, fmt.Sprint(a))
	}
	msgWithOpts := map[string][]string{target: configured}
	l.logOp(stepOp, msgWithOpts)
}

func (l *fakeUILog) logOp(op logOpType, msgWithOpts map[string][]string) {
	if _, ok := l.data[op]; !ok {
		l.data[op] = []map[string][]string{
			msgWithOpts,
		}
	} else {
		l.data[op] = append(l.data[op], msgWithOpts)
	}
}

func (l *fakeUILog) NextHeader() map[string][]string {
	if _, ok := l.data[headerOp]; !ok {
		return nil
	}
	defer func() {
		if l.nextHeader+1 < (len(l.data[headerOp])) {
			l.nextHeader++
		}
	}()
	return l.data[headerOp][l.nextHeader]
}

func (l *fakeUILog) LastHeader() map[string][]string {
	if _, ok := l.data[headerOp]; !ok {
		return nil
	}
	return l.data[headerOp][len(l.data[headerOp])-1]
}

func (l *fakeUILog) NextOutput() map[string][]string {
	if _, ok := l.data[outputOp]; !ok {
		return nil
	}
	defer func() {
		if l.nextOutput+1 < (len(l.data[outputOp])) {
			l.nextOutput++
		}
	}()
	return l.data[outputOp][l.nextOutput]
}

func (l *fakeUILog) LastOutput() map[string][]string {
	if _, ok := l.data[outputOp]; !ok {
		return nil
	}
	return l.data[outputOp][len(l.data[outputOp])-1]
}

func (l *fakeUILog) NextStep() map[string][]string {
	if _, ok := l.data[stepOp]; !ok {
		return nil
	}
	defer func() {
		if l.nextStep+1 < (len(l.data[stepOp])) {
			l.nextStep++
		}
	}()
	return l.data[stepOp][l.nextStep]
}

func (l *fakeUILog) LastStep() map[string][]string {
	if _, ok := l.data[stepOp]; !ok {
		return nil
	}
	return l.data[stepOp][len(l.data[stepOp])-1]
}

func (l *fakeUILog) Reset() {
	l.data = nil
	l.data = make(map[logOpType][]map[string][]string)
	l.nextHeader = 0
	l.nextStep = 0
	l.nextOutput = 0
	l.nextNamedValue = 0
}
