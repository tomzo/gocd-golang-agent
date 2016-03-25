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
	"testing"
)

func TestRunIfConfig(t *testing.T) {
	setUp(t)
	defer tearDown()

	goServer.SendBuild(AgentId, buildId,
		protocal.EchoCommand("should not echo if failed when passed").RunIf("failed"),
		protocal.EchoCommand("should echo if any when passed").RunIf("any"),
		protocal.EchoCommand("should echo if passed when passed").RunIf("passed"),
		protocal.ExecCommand("cmdnotexist"),
		protocal.EchoCommand("should echo if failed when failed").RunIf("failed"),
		protocal.EchoCommand("should echo if any when failed").RunIf("any"),
		protocal.EchoCommand("should not echo if passed when failed").RunIf("passed"),
	)

	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Failed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)

	expected := `should echo if any when passed
should echo if passed when passed
exec: "cmdnotexist": executable file not found in $PATH
should echo if failed when failed
should echo if any when failed
`
	assert.Equal(t, expected, trimTimestamp(log))
}

func TestComposeCommandWithRunIfConfig(t *testing.T) {
	setUp(t)
	defer tearDown()

	goServer.SendBuild(AgentId, buildId,
		protocal.ComposeCommand(
			protocal.ComposeCommand(
				protocal.EchoCommand("hello world1"),
				protocal.EchoCommand("hello world2"),
			).RunIf("any"),
			protocal.ComposeCommand(
				protocal.EchoCommand("hello world3"),
				protocal.EchoCommand("hello world4"),
			),
		).RunIf("failed"),
		protocal.ComposeCommand(
			protocal.EchoCommand("hello world5").RunIf("failed"),
			protocal.EchoCommand("hello world6"),
		),
	)

	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	assert.Equal(t, "hello world6\n", trimTimestamp(log))
}
