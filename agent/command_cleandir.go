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
	"path/filepath"
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
