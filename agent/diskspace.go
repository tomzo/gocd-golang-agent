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
	"strconv"
	"syscall"
)

func UsableSpace() string {
	_, free, err := diskSpace("/")
	if err != nil {
		LogInfo("Unknown diskspace, error: %v", err)
		return "-1"
	}
	return strconv.Itoa(free)
}

// Space returns total and free bytes available in a directory, e.g. `/`.
// Think of it as "df" UNIX command.
func diskSpace(path string) (total, free int, err error) {
	s := syscall.Statfs_t{}
	err = syscall.Statfs(path, &s)
	if err != nil {
		return
	}
	total = int(s.Bsize) * int(s.Blocks)
	free = int(s.Bsize) * int(s.Bfree)
	return
}
