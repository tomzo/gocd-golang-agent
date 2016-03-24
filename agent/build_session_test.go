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
package agent_test

import (
	"github.com/bmatcuk/doublestar"
	. "github.com/gocd-contrib/gocd-golang-agent/agent"
	"github.com/gocd-contrib/gocd-golang-agent/protocal"
	"github.com/xli/assert"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestExport(t *testing.T) {
	setUp(t)
	defer tearDown()

	os.Setenv("TEST_EXPORT", "EXPORT_VALUE")
	defer os.Setenv("TEST_EXPORT", "")

	goServer.SendBuild(AgentId, buildId,
		protocal.ExportCommand("env1", "value1", "false"),
		protocal.ExportCommand("env2", "value2", "true"),
		protocal.ExportCommand("env1", "value4", "false"),
		protocal.ExportCommand("env2", "value5", "true"),
		protocal.ExportCommand("env2", "value6", "false"),
		protocal.ExportCommand("env2", "value6", ""),
		protocal.ExportCommand("env2", "", ""),
		protocal.ExportCommand("TEST_EXPORT"),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	expected := `setting environment variable 'env1' to value 'value1'
setting environment variable 'env2' to value '********'
overriding environment variable 'env1' with value 'value4'
overriding environment variable 'env2' with value '********'
overriding environment variable 'env2' with value 'value6'
overriding environment variable 'env2' with value 'value6'
overriding environment variable 'env2' with value ''
setting environment variable 'TEST_EXPORT' to value 'EXPORT_VALUE'
`
	assert.Equal(t, expected, trimTimestamp(log))
}

func TestMkdirCommand(t *testing.T) {
	setUp(t)
	defer tearDown()

	wd := pipelineDir()
	goServer.SendBuild(AgentId, buildId,
		protocal.MkdirsCommand("path/in/pipeline/dir").Setwd(wd),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())
	_, err := os.Stat(filepath.Join(wd, "path/in/pipeline/dir"))
	assert.Nil(t, err)
}

func TestCleandirCommand(t *testing.T) {
	setUp(t)
	defer tearDown()

	wd := createTestProjectInPipelineDir()
	goServer.SendBuild(AgentId, buildId,
		protocal.CleandirCommand("test", "world2").Setwd(wd),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	matches, err := doublestar.Glob(filepath.Join(wd, "**/*.txt"))
	assert.Nil(t, err)
	sort.Strings(matches)
	expected := []string{
		"0.txt",
		"src/1.txt",
		"src/2.txt",
		"src/hello/3.txt",
		"src/hello/4.txt",
		"test/world2/10.txt",
		"test/world2/11.txt",
	}

	for i, f := range matches {
		actual := f[len(wd)+1:]
		assert.Equal(t, expected[i], actual)
	}
}

func TestFailCommand(t *testing.T) {
	setUp(t)
	defer tearDown()

	goServer.SendBuild(AgentId, buildId,
		protocal.FailCommand("something is wrong, please fail"),
		protocal.ReportCompletedCommand(),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Failed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	expected := Sprintf("something is wrong, please fail\n")
	assert.Equal(t, expected, trimTimestamp(log))
}
