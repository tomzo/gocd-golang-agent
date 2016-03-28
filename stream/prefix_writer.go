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
	"bytes"
	"io"
)

type PrefixWriter struct {
	io.Writer
	Prefix func() []byte
	ap     bool
}

func NewPrefixWriter(writer io.Writer, prefix func() []byte) *PrefixWriter {
	return &PrefixWriter{writer, prefix, true}
}

func (w *PrefixWriter) Write(out []byte) (int, error) {
	if len(out) == 0 {
		return 0, nil
	}
	ln := []byte{'\n'}
	lines := bytes.Split(out, ln)
	last := len(lines) - 1
	for i, line := range lines {
		if i == last && len(line) == 0 {
			w.ap = true
			break
		}
		if i > 0 || w.ap {
			if err := w.appendPrefix(); err != nil {
				return -1, err
			}
			w.ap = false
		}
		w.Writer.Write(line)
		if i < last {
			w.Writer.Write(ln)
		}
	}
	return len(out), nil
}

func (w *PrefixWriter) appendPrefix() error {
	_, err := w.Writer.Write(w.Prefix())
	return err
}
