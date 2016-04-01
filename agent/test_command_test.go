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
	"github.com/gocd-contrib/gocd-golang-agent/protocol"
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
		{"file exist", []string{"-nf", file}, ""},
		{"file not exist", []string{"-nf", file + "no"}, "file not exist\n"},
		{"dir is not file", []string{"-nf", dir}, "dir is not file\n"},

		{"dir exist", []string{"-d", dir}, "dir exist\n"},
		{"dir exist", []string{"-nd", dir}, ""},
		{"dir not exist", []string{"-d", dir + "no"}, ""},
		{"dir not exist", []string{"-nd", dir + "no"}, "dir not exist\n"},
		{"file is not dir", []string{"-d", file}, ""},
		{"file is not dir", []string{"-nd", file}, "file is not dir\n"},

		{"equal", []string{"-eq", "hello", "echo", "hello"}, "equal\n"},
		{"not equal", []string{"-eq", "hello", "echo", "world"}, ""},
		{"equal", []string{"-neq", "hello", "echo", "hello"}, ""},
		{"not equal", []string{"-neq", "hello", "echo", "world"}, "not equal\n"},
	}

	for _, test := range tests {
		testCmd := protocol.TestCommand(test.testArgs...).Setwd(relativePath(wd))
		goServer.SendBuild(AgentId, buildId, echo(test.echo).SetTest(testCmd))
		assert.Equal(t, "agent Building", stateLog.Next())
		assert.Equal(t, "build Passed", stateLog.Next())
		assert.Equal(t, "agent Idle", stateLog.Next())
		log, err := goServer.ConsoleLog(buildId)
		if err != nil {
			t.Errorf("Can't find console log when test: %+v", test)
		}
		actual := trimTimestamp(log)
		if test.expected != actual {
			t.Errorf("test: %+v\nbut was '%v'", test, actual)
		}
		os.Truncate(goServer.ConsoleLogFile(buildId), 0)
	}
}

func TestTestCommandEchoShouldAlsoBeMaskedForSecrets(t *testing.T) {
	setUp(t)
	defer tearDown()

	testCmd := protocol.TestCommand("-eq", "$$$", "echo", "secret")
	goServer.SendBuild(AgentId, buildId,
		protocol.SecretCommand("secret", "$$$"),
		protocol.EchoCommand("hello world").SetTest(testCmd),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	expected := Sprintf("hello world\n")
	assert.Equal(t, expected, trimTimestamp(log))
}
