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
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

type Logger struct {
	Info  *log.Logger
	Debug *log.Logger
	Error *log.Logger
}

func MakeLogger(logDir, file string, debug bool) *Logger {
	var output, debugOutput io.Writer
	if logDir != "" {
		fpath := filepath.Join(logDir, file)
		var err error
		output, err = os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			panic(err)
		}
	} else {
		output = os.Stdout
	}

	if debug {
		debugOutput = output
	} else {
		debugOutput = ioutil.Discard
	}

	debugLogger := log.New(debugOutput, "", 0)
	infoLogger := log.New(output, "", 0)
	errorLogger := log.New(output, "ERROR: ", log.Lshortfile)

	return &Logger{Debug: debugLogger, Info: infoLogger, Error: errorLogger}
}
