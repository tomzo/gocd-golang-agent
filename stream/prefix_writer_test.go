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
	"strconv"
	"testing"
)

func TestPrefixWriter(t *testing.T) {
	var tests = []struct {
		inputs []string
		output string
	}{
		{[]string{"hello"}, "1 hello"},
		{[]string{"hello", " world"}, "2 hello world"},
		{[]string{"hello\n", "world"}, "3 hello\n4 world"},
		{[]string{"hello\nworld", "!"}, "5 hello\n6 world!"},
		{[]string{"hello\nworld\n", "!"}, "7 hello\n8 world\n9 !"},
		{[]string{"\n", "hello"}, "10 \n11 hello"},
		{[]string{"\n", "\nhello"}, "12 \n13 \n14 hello"},
		{[]string{"...", "...", "...", "\nhello"}, "15 .........\n16 hello"},
		{[]string{"...", "...\n", "hello\n"}, "17 ......\n18 hello\n"},
		{[]string{"hello\n"}, "19 hello\n"},
		{[]string{"hello", "\n", "world"}, "20 hello\n21 world"},
	}
	i := 0
	for _, test := range tests {
		var buf bytes.Buffer
		w := NewPrefixWriter(&buf, func() []byte {
			i++
			return []byte(strconv.Itoa(i) + " ")
		})
		for _, d := range test.inputs {
			w.Write([]byte(d))
		}
		assert.Equal(t, test.output, buf.String())
	}
}
