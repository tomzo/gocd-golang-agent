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
	"testing"
	"path/filepath"
	"runtime"
	"github.com/xli/assert"
)

func TestReadReportForNUnit2XReport(t *testing.T) {
	results, err := Read(filepath.Join(DIR(), "test", "nunit2x_report1.xml"))

	assert.Nil(t, err)
	assert.Equal(t, 8, results.Total)
	assert.Equal(t, 1, results.Errors)
	assert.Equal(t, 1, results.Failures)
	assert.Equal(t, 7, results.NotRun)
	assert.Equal(t, 4, results.Ignored)
	assert.Equal(t, 0, results.Skipped)
}

func TestReadReportMergeNUnit2XReport(t *testing.T) {

	result := NewTestResults()

	results1, err1 := Read(filepath.Join(DIR(), "test", "nunit2x_report1.xml"))
	assert.Nil(t, err1)
	result.Merge(results1)

	results2, err2 := Read(filepath.Join(DIR(), "test", "nunit2x_report2.xml"))
	assert.Nil(t, err2)
	result.Merge(results2)

	assert.Equal(t, 16, result.Total)
	assert.Equal(t, 2, result.Errors)
	assert.Equal(t, 2, result.Failures)
	assert.Equal(t, 14, result.NotRun)
	assert.Equal(t, 8, result.Ignored)
	assert.Equal(t, 0, result.Skipped)

}

func DIR() string {
	_, filename, _, _ := runtime.Caller(1)
	return filepath.Dir(filename)
}