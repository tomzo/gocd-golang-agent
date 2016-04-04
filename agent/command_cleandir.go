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
	"github.com/gocd-contrib/gocd-golang-agent/protocol"
	"github.com/gocd-contrib/gocd-golang-agent/stream"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func CommandCleandir(s *BuildSession, cmd *protocol.BuildCommand) (err error) {
	path := cmd.Args["path"]
	allows, err := cmd.ListArg("allowed")
	if err != nil {
		return
	}
	fullPath := filepath.Join(s.wd, path)
	s.debugLog("cleandir %v, excludes: %+v", fullPath, allows)
	return Cleandir(s.console, fullPath, allows...)
}

func Cleandir(log io.Writer, root string, allows ...string) error {
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
	w := stream.NewSubstituteWriter(log)
	w.Substitutions[" "+root+"/"] = " "
	return cleandir(w, root, allows...)
}

func cleandir(log io.Writer, root string, allows ...string) error {
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
				log.Write([]byte(Sprintf("Deleting folder %v\n", fpath)))
				if err := os.RemoveAll(fpath); err != nil {
					return err
				}
			} else if fpath != match {
				if err := cleandir(log, fpath, allows...); err != nil {
					return err
				}
			} else {
				log.Write([]byte(Sprintf("Keeping folder %v\n", fpath)))
			}
		} else {
			log.Write([]byte(Sprintf("Deleting file %v\n", fpath)))
			err := os.Remove(fpath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
