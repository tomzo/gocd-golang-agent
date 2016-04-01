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
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
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

func Cleandir(root string, allows ...string) error {
	root = filepath.Clean(root)
	for i, allow := range allows {
		allows[i] = filepath.Clean(filepath.Join(root, allow))
		if allows[i] == root {
			return nil
		}
		if strings.HasPrefix(root, allows[i]) {
			return Err("Cannot clean directory. Folder %v is outside the base folder %v", allows[i], root)
		}
	}

	return cleandir(root, allows...)
}

func cleandir(root string, allows ...string) error {
	infos, err := ioutil.ReadDir(root)
	if err != nil {
		return err
	}

	for _, finfo := range infos {
		fpath := filepath.Join(root, finfo.Name())
		if finfo.IsDir() {
			match := ""
			for _, allow := range allows {
				if strings.HasPrefix(allow, fpath) {
					match = allow
					break
				}
			}
			if match == "" {
				if err := os.RemoveAll(fpath); err != nil {
					return err
				}
			} else if fpath != match {
				if err := cleandir(fpath, allows...); err != nil {
					return err
				}
			}
		} else {
			err := os.Remove(fpath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func Sprintf(f string, args ...interface{}) string {
	return fmt.Sprintf(f, args...)
}

func Err(f string, args ...interface{}) error {
	return errors.New(Sprintf(f, args...))
}

func closeAndWait(stop, closed chan bool, timeout time.Duration) error {
	select {
	case _, ok := <-stop:
		if !ok {
			return nil
		}
	default:
	}

	close(stop)

	select {
	case <-closed:
		return nil
	case <-time.After(timeout):
		return Err("Wait for closed timeout")
	}
}

func isClosedChan(ch chan bool) bool {
	select {
	case _, ok := <-ch:
		return !ok
	default:
		return false
	}
}

func ParseChecksum(checksum string) map[string]string {
	ret := make(map[string]string)
	for _, l := range strings.Split(checksum, "\n") {
		if strings.HasPrefix(l, "#") {
			continue
		}
		i := strings.Index(l, "=")
		if i > -1 {
			ret[l[:i]] = l[i+1:]
		}
	}
	return ret
}

func ComputeMd5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	var result []byte
	return Sprintf("%x", hash.Sum(result)), nil
}
