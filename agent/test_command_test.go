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
	. "github.com/gocd-contrib/gocd-golang-agent/agent"
	"github.com/gocd-contrib/gocd-golang-agent/protocal"
	"github.com/xli/assert"
	"os"
	"testing"
)

func TestTestCommand(t *testing.T) {
	setUp(t)
	defer tearDown()

	wd := createTestProjectInPipelineDir()
	file := "src/hello/3.txt"
	dir := "src/hello"

	var tests = []struct {
		echo     string
		testArgs []string
		expected string
	}{
		{"file exist", []string{"-f", file}, "file exist\n"},
		{"file not exist", []string{"-f", file + "no"}, ""},
		{"dir is not file", []string{"-f", dir}, ""},

		{"dir exist", []string{"-d", dir}, "dir exist\n"},
		{"dir not exist", []string{"-d", dir + "no"}, ""},
		{"file is not dir", []string{"-d", file}, ""},

		{"equal", []string{"-eq", "hello", "echo", "hello"}, "equal\n"},
		{"not equal", []string{"-eq", "hello", "echo", "world"}, ""},
	}

	for _, test := range tests {
		testCmd := protocal.TestCommand(test.testArgs...).Setwd(wd)
		goServer.SendBuild(AgentId, buildId, protocal.EchoCommand(test.echo).SetTest(testCmd))
		assert.Equal(t, "agent Building", stateLog.Next())
		assert.Equal(t, "build Passed", stateLog.Next())
		assert.Equal(t, "agent Idle", stateLog.Next())
		log, err := goServer.ConsoleLog(buildId)
		assert.Nil(t, err)
		actual := trimTimestamp(log)
		if test.expected != actual {
			t.Errorf("test: %+v\nbut was '%v'", test, actual)
		}
		os.Truncate(goServer.ConsoleLogFile(buildId), 0)
	}
}
