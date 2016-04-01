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
package junit_test

import (
	. "github.com/gocd-contrib/gocd-golang-agent/junit"
	"github.com/xli/assert"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReadReport(t *testing.T) {
	suite, err := Read(filepath.Join(DIR(), "test", "junit_report1.xml"))
	assert.Nil(t, err)
	assert.Equal(t, 1, suite.Tests)
}

func TestReadTestSuitesReport(t *testing.T) {
	suite, err := Read(filepath.Join(DIR(), "test", "junit_report3.xml"))
	assert.Nil(t, err)
	assert.Equal(t, 1, suite.Tests)
}

func TestMergeSuite(t *testing.T) {
	suite := NewTestSuite()
	suite1, err := Read(filepath.Join(DIR(), "test", "junit_report1.xml"))
	assert.Nil(t, err)
	suite.Merge(suite1)

	suite2, err := Read(filepath.Join(DIR(), "test", "junit_report2.xml"))
	assert.Nil(t, err)
	suite.Merge(suite2)

	suite3, err := Read(filepath.Join(DIR(), "test", "junit_report3.xml"))
	assert.Nil(t, err)
	suite.Merge(suite3)

	assert.Equal(t, 4, suite.Tests)
	assert.Equal(t, 1, suite.Failures)
	assert.Equal(t, 4, len(suite.TestCases))
}

func DIR() string {
	_, filename, _, _ := runtime.Caller(1)
	return filepath.Dir(filename)
}
