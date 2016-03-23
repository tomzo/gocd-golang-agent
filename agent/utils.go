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

package agent

import (
	"bytes"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func Join(sep string, parts ...string) string {
	var buf bytes.Buffer
	for i, s := range parts {
		if len(s) == 0 || s == sep {
			if i < len(parts)-1 {
				buf.WriteString(sep)
			}
			continue
		}
		start := 0
		end := len(s)
		if i > 0 && s[0] == '/' {
			start++
		}
		if i < len(parts) && s[len(s)-1] == '/' {
			end--
		}
		buf.WriteString(s[start:end])
		if i < len(parts)-1 {
			buf.WriteString(sep)
		}
	}
	return buf.String()
}

func BaseDirOfPathWithWildcard(path string) string {
	dir := strings.Split(path, "*")[0]
	if dir == "" {
		return ""
	}
	// clean up filename in the path
	base, _ := filepath.Split(dir)
	if base == "" {
		return ""
	}
	return base[:len(base)-1]
}

func AppendUrlParam(base *url.URL, paramName, paramValue string) *url.URL {
	url, _ := url.Parse(base.String())
	values := url.Query()
	values.Set(paramName, paramValue)
	base.RawQuery = values.Encode()
	return url
}

func AppendUrlPath(base *url.URL, path string) *url.URL {
	url, _ := url.Parse(base.String())
	url.RawPath = Join("/", url.RawPath, path)
	return url
}

func Mkdirs(path string) error {
	return os.MkdirAll(path, 0755)
}
