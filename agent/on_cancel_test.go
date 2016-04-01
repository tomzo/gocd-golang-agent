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
	"testing"
	"time"
)

func TestOnCancel1(t *testing.T) {
	setUp(t)
	defer tearDown()
	goServer.SendBuild(AgentId, buildId,
		protocol.ComposeCommand(
			echo("echo before sleep"),
			protocol.ExecCommand("sleep", "5").SetOnCancel(echo("read on cancel")),
			echo("should not process this echo").RunIf("any"),
		).SetOnCancel(protocol.ExecCommand("echo", "compose on cancel")),
		echo("should not process this echo"),
	)

	assert.Equal(t, "agent Building", stateLog.Next())

	goServer.Send(AgentId, protocol.CancelMessage())

	assert.Equal(t, "build Cancelled", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)

	expected := `echo before sleep
read on cancel
compose on cancel
`
	assert.Equal(t, expected, trimTimestamp(log))
}

func TestOnCancel2(t *testing.T) {
	CancelCommandTimeout = 10 * time.Millisecond
	defer func() {
		CancelCommandTimeout = DefaultCancelCommandTimeout
	}()

	setUp(t)
	defer tearDown()
	cancel := protocol.ExecCommand("sleep", "60")
	goServer.SendBuild(AgentId, buildId,
		protocol.ExecCommand("sleep", "5").SetOnCancel(cancel),
	)

	assert.Equal(t, "agent Building", stateLog.Next())

	goServer.Send(AgentId, protocol.CancelMessage())

	assert.Equal(t, "build Cancelled", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)

	expected := `WARN: Kill cancel task because it did not finish in 10ms.
`
	assert.Equal(t, expected, trimTimestamp(log))
}

func TestOnCancelShouldContinueMaskEchos(t *testing.T) {
	setUp(t)
	defer tearDown()
	goServer.SendBuild(AgentId, buildId,
		protocol.SecretCommand("secret", "$$$"),
		protocol.ExecCommand("sleep", "5").SetOnCancel(echo("secret on cancel: ${agent.location}")),
	)

	assert.Equal(t, "agent Building", stateLog.Next())

	goServer.Send(AgentId, protocol.CancelMessage())

	assert.Equal(t, "build Cancelled", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)

	config := GetConfig()
	expected := Sprintf("$$$ on cancel: %v\n", config.WorkingDir)
	assert.Equal(t, expected, trimTimestamp(log))
}

func TestCancelBuildWhenBuildIsHangingOnTestCommand(t *testing.T) {
	setUp(t)
	defer tearDown()

	testCmd := protocol.ExecCommand("sleep", "5")
	goServer.SendBuild(AgentId, buildId,
		echo("hello before cancel"),
		echo("hello after sleep 5").SetTest(testCmd),
	)
	assert.Equal(t, "agent Building", stateLog.Next())

	goServer.Send(AgentId, protocol.CancelMessage())

	assert.Equal(t, "build Cancelled", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)

	expected := "hello before cancel\n"
	assert.Equal(t, expected, trimTimestamp(log))
}
