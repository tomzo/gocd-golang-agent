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
)

func TestEcho(t *testing.T) {
	setUp(t)
	defer tearDown()

	goServer.SendBuild(AgentId, buildId,
		protocol.EchoCommand("echo hello world"),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	assert.Equal(t, "echo hello world\n", trimTimestamp(log))
}
