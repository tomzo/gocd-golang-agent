#!/bin/bash
#
# Copyright 2016 ThoughtWorks, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#  http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

set +x
set -e

#                   src/github.com/gocd-contrib/gocd-golang-agent
export GOPATH=`pwd`/../../../../
go get golang.org/x/net/websocket
go get github.com/satori/go.uuid
go get github.com/xli/assert
go get github.com/bmatcuk/doublestar
go get github.com/jstemmer/go-junit-report
# go get -u all
go test -test.v ./... | $GOPATH/bin/go-junit-report > testreport.xml
