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

package protocol_test

import (
	. "github.com/gocd-contrib/gocd-golang-agent/protocol"
	"github.com/xli/assert"
	"testing"
)

func TestListArg(t *testing.T) {
	cmd := NewBuildCommand("foo").AddListArg("lines", []string{"hello", "world", "!"})
	list, err := cmd.ListArg("lines")
	assert.Nil(t, err)
	assert.Equal(t, 3, len(list))
	assert.Equal(t, "hello", list[0])
	assert.Equal(t, "world", list[1])
	assert.Equal(t, "!", list[2])

	assert.Equal(t, `["hello","world","!"]`, cmd.Args["lines"])
}

func TestAddArg(t *testing.T) {
	cmd := NewBuildCommand(CommandCompose)
	cmd.AddArg("hello", "world")
	assert.Equal(t, "world", cmd.Args["hello"])
}

func TestAddCommands(t *testing.T) {
	cmd := NewBuildCommand(CommandCompose)
	assert.Equal(t, 0, len(cmd.SubCommands))
	cmd.AddCommands(NewBuildCommand(CommandEcho))
	assert.Equal(t, 1, len(cmd.SubCommands))
}
