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

package agent

import (
	"github.com/bmatcuk/doublestar"
	"github.com/gocd-contrib/gocd-golang-agent/junit"
	"github.com/gocd-contrib/gocd-golang-agent/protocol"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"github.com/gocd-contrib/gocd-golang-agent/nunit"
)

type UnitTestReport struct {
	Tests     int
	Failures  int
	Skipped   int
	Time      float64
	TestCases []*TestCase
}

type TestCase struct {
	Name    string
	Failure *Failure
	Error   *Error
}

type FailureMessage struct {
	StackTrace string
}

type Failure struct {
	FailureMessage
}

type Error struct {
	FailureMessage
}

func (r *UnitTestReport) Merge(another *UnitTestReport) {
	r.Tests += another.Tests
	r.Failures += another.Failures
	r.Skipped += another.Skipped
	r.Time += another.Time
	r.TestCases = append(r.TestCases, another.TestCases...)
}

func CommandGenerateTestReport(s *BuildSession, cmd *protocol.BuildCommand) error {
	srcs, err := cmd.ListArg("srcs")
	if err != nil {
		return err
	}
	if len(srcs) == 0 {
		return nil
	}
	uploadPath := cmd.Args["uploadPath"]

	report := new(UnitTestReport)

	rep, err := generateUnitTestReportFromJunitReport(s, srcs)
	if err != nil {
		return err
	}
	report.Merge(rep)

	req2, err := generateUnitTestReportFromNunitReport(s, srcs)
	if err != nil {
		return err
	}

	report.Merge(req2)

	return uploadUnitTestReportArtifacts(s, uploadPath, report)
}

func uploadUnitTestReportArtifacts(s *BuildSession, uploadPath string, req *UnitTestReport) error {

	template, err := loadTestReportTemplate()
	if err != nil {
		return err
	}

	outputPath := filepath.Join(s.wd, uploadPath, protocol.TestReportFileName)

	err = Mkdirs(filepath.Dir(outputPath))
	if err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}

	err = template.Execute(file, req)

	defer file.Close()

	if err != nil {
		return err
	}
	return uploadArtifacts(s, file.Name(), uploadPath, false)
}

func generateUnitTestReportFromNunitReport(s *BuildSession, srcs []string) (report *UnitTestReport, err error) {

	results := nunit.NewTestResults()
	report = new(UnitTestReport)

	for _, src := range srcs {
		path := filepath.Join(s.wd, src)
		if strings.Contains(path, "*") {
			matches, err1 := doublestar.Glob(path)
			if err1 != nil {
				err = err1
			}
			sort.Strings(matches)
			for _, fpath := range matches {
				generateNUnitTestReport(s, results, fpath)
			}
		} else {
			generateNUnitTestReport(s, results, path)
		}
	}

	s.debugLog("nunit test report: %+v", results)

	report.Tests = results.Total
	report.Skipped = results.Skipped
	report.Failures = results.Failures + results.Errors
	report.Time = results.Time

	report.TestCases = mapNunitTestCaseToTemplate(results.TestCases)

	return

}

func generateUnitTestReportFromJunitReport(s *BuildSession, srcs []string) (report *UnitTestReport, err error) {
	suite := junit.NewTestSuite()
	report = new(UnitTestReport)

	for _, src := range srcs {
		path := filepath.Join(s.wd, src)
		if strings.Contains(path, "*") {
			matches, err1 := doublestar.Glob(path)
			if err1 != nil {
				err = err1
			}
			sort.Strings(matches)
			for _, fpath := range matches {
				generateJunitTestReport(s, suite, fpath)
			}
		} else {
			generateJunitTestReport(s, suite, path)
		}
	}

	s.debugLog("test report: %+v", suite)

	report.Tests = suite.Tests
	report.Skipped = suite.Skipped
	report.Failures = suite.Failures + suite.Errors
	report.TestCases = mapJunitTestCaseToTemplate(suite.TestCases)
	report.Time = suite.Time

	return
}

func generateNUnitTestReport(s *BuildSession, result *nunit.TestResults, path string) {
	err := nunit.GenerateNUnitTestReport(result, path)
	if err != nil {
		s.debugLog("ignore %v for error: %v", path, err)
		return
	}
	return
}

func generateJunitTestReport(s *BuildSession, result *junit.TestSuite, path string) {
	err := junit.GenerateJunitTestReport(result, path)
	if err != nil {
		s.debugLog("ignore %v for error: %v", path, err)
		return
	}
	return
}

func mapJunitTestCaseToTemplate(testCases []*junit.TestCase) (results []*TestCase) {
	for _, item := range testCases {
		t := new(TestCase)
		t.Name = item.Name
		if item.Failure != nil {
			t.Failure = new(Failure)
			t.Failure.StackTrace = item.Failure.StackTrace
		}
		if item.Error != nil {
			t.Error = new(Error)
			t.Error.StackTrace = item.Error.StackTrace
		}
		results = append(results, t)
	}
	return
}

func mapNunitTestCaseToTemplate(testCases []*nunit.TestCase) (results []*TestCase) {
	for _, item := range testCases {
		t := new(TestCase)
		t.Name = item.Name
		if item.Failure != nil {
			t.Failure = new(Failure)
			t.Failure.StackTrace = item.Failure.StackTrace.Content
		}
		results = append(results, t)
	}
	return
}

func loadTestReportTemplate() (*template.Template, error) {
	return template.New("TestReportTemplate").Parse(UNIT_TEST_REPORT_TEMPLATE)
}