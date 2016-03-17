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

package server

import (
	"io/ioutil"
	"net/http"
)

func consoleHandler(s *Server) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		buildId := parseBuildId(req.URL.Path)
		bytes, err := ioutil.ReadAll(req.Body)
		if err != nil {
			s.responseBadRequest(err, w)
			return
		}
		err = s.appendToFile(s.ConsoleLogFile(buildId), bytes)
		if err != nil {
			s.responseInternalError(err, w)
		}
	}
}
