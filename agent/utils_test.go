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
	"github.com/xli/assert"
	"testing"
)

func TestBaseDirOfPathWithWildcard(t *testing.T) {
	assert.Equal(t, "", BaseDirOfPathWithWildcard("/*"))
	assert.Equal(t, "", BaseDirOfPathWithWildcard("/**/*"))
	assert.Equal(t, "", BaseDirOfPathWithWildcard("/*.go"))
	assert.Equal(t, "", BaseDirOfPathWithWildcard("/hello/*.go"))

	assert.Equal(t, "/hello", BaseDirOfPathWithWildcard("/hello/world/*.go"))
	assert.Equal(t, "/hello", BaseDirOfPathWithWildcard("/hello/world/**/*.go"))
	assert.Equal(t, "/hello", BaseDirOfPathWithWildcard("/hello/world/f*/*.go"))
}

func TestJoin(t *testing.T) {
	assert.Equal(t, "/", Join("/", "", ""))
	assert.Equal(t, "/", Join("/", "/", "/"))
	assert.Equal(t, "a/b", Join("/", "a", "b"))
	assert.Equal(t, "a/b", Join("/", "a", "/b"))
	assert.Equal(t, "a/b", Join("/", "a/", "/b"))
	assert.Equal(t, "a/", Join("/", "a/", "/"))
	assert.Equal(t, "a/b/", Join("/", "a/b", "/"))
}
