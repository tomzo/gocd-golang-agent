/*
 * Copyright 2016 ThoughtWorks, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package nunit

import (
	"encoding/xml"
	"io/ioutil"
	"fmt"
)

type TestResults struct {
	XMLName xml.Name `xml:"test-results"`

	Name string `xml:"name,attr"`
	Total int `xml:"total,attr"`
	Errors int `xml:"errors,attr"`
	Failures int `xml:"failures,attr"`
	NotRun int `xml:"not-run,attr"`
	Ignored int `xml:"ignored,attr"`
	Skipped int `xml:"skipped,attr"`
	Invalid int `xml:"invalid,attr"`

}

type TestSuit struct {
	XMLName xml.Name `xml:"test-suite"`

	Name string `xml:"name,attr"`
	Executed bool `xml:"executed,attr"`
	Success bool `xml:"success,attr"`
	Time float64 `xml:"time,attr"`
	Asserts int `xml:"asserts,attr"`
}

func NewTestResults() *TestResults {
	return new(TestResults)
}

func (t *TestResults) Merge(another *TestResults) {
	fmt.Println(t.Total, another.Total)
	t.Total += another.Total
	t.Errors += another.Errors
	t.Failures += another.Failures
	t.NotRun += another.NotRun
	t.Ignored += another.Ignored
	t.Skipped += another.Skipped
	t.Invalid += another.Invalid
}

func Read(f string) (results *TestResults, err error){
	data, err := ioutil.ReadFile(f)
	if err != nil {
		return
	}

	results = NewTestResults()
	xml.Unmarshal(data, results)

	return
}