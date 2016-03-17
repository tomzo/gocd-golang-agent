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

package protocal_test

import (
	. "github.com/gocd-contrib/gocd-golang-agent/protocal"
	"github.com/xli/assert"
	"testing"
)

func TestExtractArgList(t *testing.T) {
	cmd := EchoCommand("hello", "world")
	cmd.Args["2"] = "!"
	cmd.Args["3"] = "not extracted"
	cmd.Args["key"] = "value"
	list := cmd.ExtractArgList(3)
	assert.Equal(t, 3, len(list))
	assert.Equal(t, "hello", list[0])
	assert.Equal(t, "world", list[1])
	assert.Equal(t, "!", list[2])
}
