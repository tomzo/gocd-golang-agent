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

func CommandDownloadArtifact(s *BuildSession, cmd *protocol.BuildCommand) error {
	checksumURL, err := config.MakeFullServerURL(cmd.Args["checksumUrl"])
	if err != nil {
		return err
	}
	absChecksumFile := filepath.Join(s.wd, cmd.Args["checksumFile"])
	err = s.artifacts.DownloadFile(checksumURL, absChecksumFile)
	if err != nil {
		return err
	}

	srcURL, err := config.MakeFullServerURL(cmd.Args["url"])
	if err != nil {
		return err
	}
	srcPath := cmd.Args["src"]
	absDestPath := filepath.Join(s.wd, cmd.Args["dest"])
	if cmd.Name == protocol.CommandDownloadDir {
		_, fname := filepath.Split(srcPath)
		absDestPath = filepath.Join(s.wd, cmd.Args["dest"], fname)
	}
	err = s.artifacts.VerifyChecksum(srcPath, absDestPath, absChecksumFile)
	if err == nil {
		s.ConsoleLog("[%v] exists and matches checksum, does not need dowload it from server.\n", srcPath)
		return nil
	}
	s.debugLog("download %v to %v", srcURL, absDestPath)
	if cmd.Name == protocol.CommandDownloadDir {
		err = s.artifacts.DownloadDir(srcURL, absDestPath)
	} else {
		err = s.artifacts.DownloadFile(srcURL, absDestPath)
	}
	if err != nil {
		return err
	}
	return s.artifacts.VerifyChecksum(srcPath, absDestPath, absChecksumFile)
}
