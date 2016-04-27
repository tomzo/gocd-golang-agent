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
)

type UnitTestReport struct {
	Tests     int
	Failures  int
	Skipped   int
	Time      float64
	TestCases []*junit.TestCase
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

	suite := junit.NewTestSuite()
	for _, src := range srcs {
		path := filepath.Join(s.wd, src)
		if strings.Contains(path, "*") {
			matches, err := doublestar.Glob(path)
			if err != nil {
				return err
			}
			sort.Strings(matches)
			for _, fpath := range matches {
				generateTestReport(s, suite, fpath)
			}
		} else {
			generateTestReport(s, suite, path)
		}
	}

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
	s.debugLog("test report: %+v", suite)

	var rep UnitTestReport
	rep.Tests = suite.Tests
	rep.Skipped = suite.Skipped
	rep.Failures = suite.Failures + suite.Errors
	rep.TestCases = suite.TestCases
	rep.Time = suite.Time

	err = template.Execute(file, rep)
	file.Close()
	if err != nil {
		return err
	}
	return uploadArtifacts(s, file.Name(), uploadPath, false)
}

func generateTestReport(s *BuildSession, result *junit.TestSuite, path string) {
	info, err := os.Stat(path)
	if err != nil {
		s.debugLog("ignore %v for error: %v", path, err)
		return
	}
	if info.IsDir() {
		return
	}
	suite, err := junit.Read(path)
	if err != nil {
		s.debugLog("ignore %v for error: %v", path, err)
		return
	}
	s.debugLog("suite: %+v", suite)
	result.Merge(suite)
	return
}

func loadTestReportTemplate() (*template.Template, error) {
	return template.New("TestReportTemplate").Parse(UNIT_TEST_REPORT_TEMPLATE)
}
