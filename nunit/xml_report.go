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

	Environment *Environment `xml:"environment"`
	TestSuite *TestSuite `xml:"test-suite"`
}

type Environment struct {
	XMLName xml.Name `xl:"environment"`
	NUnitVersion string `xml:"nunit-version,attr"`
	ClrVersion string `xml:"clr-version,attr"`
	OsVersion string `xml:"os-version,attr"`
	Platform string `xml:"platform,attr"`
	Cwd string `xml:"cwd,attr"`
	MachineName string`xml:"machine-name,attr"`
	User string `xml:"user,attr"`
	UserDomain string `xml:"user-domain,attr"`
}

type TestSuite struct {
	XMLName xml.Name `xml:"test-suite"`

	Categories *Categories `xml:"categories"`
	Properties *Properties `xml:"properties"`
	Failure *Failure `xml:"failure"`
	Reason *Reason `xml:"reason"`

	Results *Results `xml:"results"`

	Name string `xml:"name,attr"`
	Executed bool `xml:"executed,attr"`
	Success bool `xml:"success,attr"`
	Time float64 `xml:"time,attr"`
	Asserts int `xml:"asserts,attr"`
}

type Results struct {
	XMLName xml.Name `xml:"results"`
	TestSuits []*TestSuite `xml:"test-suite"`
	TestCases []*TestCase `xml:"test-case"`
}

type TestCase struct {
	XMLName xml.Name `xml:"test-case"`

	Categories *Categories `xml:"categories"`
	Properties *Properties `xml:"properties"`
	Failure *Failure `xml:"failure"`
	Reason *Reason `xml:"reason"`

	Name string `xml:"name,attr"`
	Description string `xml:"name,attr"`
	Success string `xml:"success,attr"`
	Time float64 `xml:"time.attr"`
	Executed bool `xml:"executed,attr"`
	Asserts int `xml:"asserts,attr"`
}

type Categories struct {
	Categories []*Category `xml:"category"`
}

type Category struct {
	XMLName xml.Name `xml:"category"`
	Name string `xml:"name,attr"`
}

type Properties struct {
	XMLName xml.Name `xml:"properties"`
	Properties []*Property `xml:"property"`
}

type Property struct {
	XMLName xml.Name `xml:"property"`
	Name string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type Failure struct {
	XMLName xml.Name `xml:"failure"`
	Message *Message `xml:"message"`
}

type Reason struct {
	XMLName xml.Name `xml:"reason"`
	Message *Message `xml:"message"`
}

type Message struct {
	XMLName xml.Name `xml:"message"`
	Content string `xml:",chardata"`
}

type StackTrace struct {
	XMLName xml.Name `xml:"stack-trace"`
	Content string `xml:",chardata"`
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

	fmt.Println("results is", results)

	return
}