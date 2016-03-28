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

package stream

import (
	"io"
	"strings"
)

type SubstituteWriter struct {
	io.Writer
	Substitutions map[string]interface{}
}

func NewSubstituteWriter(writer io.Writer) *SubstituteWriter {
	return &SubstituteWriter{writer, make(map[string]interface{})}
}

func (w *SubstituteWriter) Write(out []byte) (int, error) {
	str := string(out)
	for k, v := range w.Substitutions {
		vs, ok := v.(string)
		if !ok {
			f, _ := v.(func() string)
			vs = f()
		}
		str = strings.Replace(str, k, vs, -1)
	}

	_, err := w.Writer.Write([]byte(str))
	return len(out), err
}
