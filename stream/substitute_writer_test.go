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
package stream_test

import (
	"bytes"
	. "github.com/gocd-contrib/gocd-golang-agent/stream"
	"github.com/xli/assert"
	"testing"
)

func TestSubstituteWriter(t *testing.T) {
	var tests = []struct {
		subs   map[string]interface{}
		inputs []string
		output string
	}{
		{
			map[string]interface{}{
				"${hello}": "world",
			},
			[]string{"hello ${hello}"},
			"hello world",
		},
		{
			map[string]interface{}{
				"${hello}": "world",
				"abcd":     "****",
			},
			[]string{"hello ${hello} ${abcd}", " ${hello}"},
			"hello world ${****} world",
		},
		{
			map[string]interface{}{
				"${hello}": "world",
				"abcd":     "****",
			},
			[]string{"hello ${hello} ${abcd}", " ${hello}"},
			"hello world ${****} world",
		},
		{
			map[string]interface{}{
				"${hello}": func() string { return "world" },
			},
			[]string{"hello ${hello}"},
			"hello world",
		},
	}
	for _, test := range tests {
		var buf bytes.Buffer
		w := &SubstituteWriter{
			Substitutions: test.subs,
			Writer:        &buf,
		}
		for _, d := range test.inputs {
			size, err := w.Write([]byte(d))
			assert.Nil(t, err)
			assert.Equal(t, len(d), size)
		}
		assert.Equal(t, test.output, buf.String())
	}
}
