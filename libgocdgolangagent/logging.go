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

package libgocdgolangagent

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var infoLogger, debugLogger *log.Logger

func logOutput() (io.Writer, error) {
	logDir := AgentLogDir()
	if logDir != "" {
		fpath := filepath.Join(AgentLogDir(), "gocd-golang-agent.log")
		file, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		return file, err
	} else {
		return os.Stdout, nil
	}
}

func InitLogger() error {
	writer, err := logOutput()
	if err != nil {
		return err
	}
	debugLogger = log.New(writer, "[DEBUG] ", 0)
	infoLogger = log.New(writer, "[INFO] ", 0)
	if ConfigGetDebug() {
		debugLogger.SetOutput(ioutil.Discard)
	}
	return nil
}

func LogDebug(format string, v ...interface{}) {
	debugLogger.Printf(format, v...)
}

func LogInfo(format string, v ...interface{}) {
	infoLogger.Printf(format, v...)
}
