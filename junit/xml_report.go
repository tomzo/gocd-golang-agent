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

package junit

import (
	"encoding/xml"
	"io/ioutil"
	"os"
)

type TestSuites struct {
	XMLName xml.Name `xml:"testsuites"`

	TestSuites []*TestSuite `xml:"testsuite"`
}

type TestSuite struct {
	XMLName xml.Name `xml:"testsuite"`

	Failures int     `xml:"failures,attr"`
	Time     float64 `xml:"time,attr"`
	Errors   int     `xml:"errors,attr"`
	Skipped  int     `xml:"skipped,attr"`
	Tests    int     `xml:"tests,attr"`
	Name     string  `xml:"name,attr"`

	TestCases []*TestCase `xml:"testcase"`
}

func (s *TestSuite) Merge(another *TestSuite) {
	s.Failures += another.Failures
	s.Time += another.Time
	s.Errors += another.Errors
	s.Skipped += another.Skipped
	s.Tests += another.Tests
	s.TestCases = append(s.TestCases, another.TestCases...)
}

type TestCase struct {
	XMLName xml.Name `xml:"testcase"`

	Time      float64 `xml:"time,attr"`
	ClassName string  `xml:"classname,attr"`
	Name      string  `xml:"name,attr"`

	Skipped   *Skipped   `xml:"skipped"`
	Error     *Error     `xml:"error"`
	Failure   *Failure   `xml:"failure"`
	SystemOut *SystemOut `xml:"system-out"`
}

type Skipped struct {
	XMLName xml.Name `xml:"skipped"`
}

type Error struct {
	XMLName xml.Name `xml:"error"`

	Type       string `xml:"type,attr"`
	StackTrace string `xml:",chardata"`
}

type Failure struct {
	XMLName xml.Name `xml:"failure"`

	Message    string `xml:"message,attr"`
	Type       string `xml:"type,attr"`
	StackTrace string `xml:",chardata"`
}

type SystemOut struct {
	XMLName xml.Name `xml:"system-out"`

	Log string `xml:",chardata"`
}

func (t *TestCase) Type() string {
	if t.IsFail() {
		return "Failure"
	}
	if t.IsError() {
		return "Error"
	}
	return ""
}

func (t *TestCase) IsFail() bool {
	return t.Failure != nil
}

func (t *TestCase) IsError() bool {
	return t.Error != nil
}

func (t *TestCase) IsSkip() bool {
	return t.Skipped != nil
}

func NewTestSuite() *TestSuite {
	return &TestSuite{
		TestCases: make([]*TestCase, 0),
	}
}

func Read(f string) (suite *TestSuite, err error) {
	data, err := ioutil.ReadFile(f)
	if err != nil {
		return
	}

	suite = NewTestSuite()
	err = xml.Unmarshal(data, suite)
	if err != nil {
		if isTestSuites(err) {
			suites := &TestSuites{
				TestSuites: make([]*TestSuite, 0),
			}
			err = xml.Unmarshal(data, suites)
			if err != nil {
				return
			}
			for _, s := range suites.TestSuites {
				suite.Merge(s)
			}
		} else {
			return
		}
	}
	return
}

func GenerateJunitTestReport(result *TestSuite, path string) (err error){
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if info.IsDir() {
		return
	}
	suite, err := Read(path)
	if err != nil {
		return
	}
	result.Merge(suite)
	return
}

func isTestSuites(err error) bool {
	return err.Error() == "expected element type <testsuite> but have <testsuites>"
}
