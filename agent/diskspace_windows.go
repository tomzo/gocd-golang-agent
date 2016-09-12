// +build windows

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
	"unsafe"
	"os"
)

func UsableSpace() int64 {
	wd, err := os.Getwd()
	if err != nil {
		LogInfo("Cannot find working directory, error: %v", err)
		return -1
	}
	_, free, err := diskSpace(wd)
	if err != nil {
		LogInfo("Unknown diskspace, error: %v", err)
		return -1
	}
	return free
}

func UsableSpaceString() string {
	return strconv.FormatInt(UsableSpace(), 10)
}


func diskSpace(path string) (total, free int64, err error) {
	var (
		freeBytes int64
	 	totalBytes int64
		availBytes int64
	)

	h := syscall.MustLoadDLL("kernel32.dll")
	c := h.MustFindProc("GetDiskFreeSpaceExW")

	_, _, err = c.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(path))),
		uintptr(unsafe.Pointer(&freeBytes)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&availBytes)))
	total = totalBytes
	free = freeBytes
	return
}
