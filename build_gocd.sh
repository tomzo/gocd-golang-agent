#!/bin/bash
#
# Copyright 2016 ThoughtWorks, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License..
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

if [[ -f var ]]; then
. vars
fi

BUILD_DIR=$(pwd)
export GOPATH=$BUILD_DIR

# Clean my build
if [[ -f gocd-golang-agent ]]; then
  echo "Remove binary : gocd-golang-agent"
  rm -rf gocd-golang-agent
fi

# Pull dependencies
echo "Get dependencies"
go get golang.org/x/net/websocket

# Pull dependencies used by Test
go get golang.org/x/text
go get golang.org/x/crypto/ssh

go build -o $GOPATH/bin/go-junit-report github.com/jstemmer/go-junit-report
go test -test.v github.com/gocd-contrib/gocd-golang-agent | $GOPATH/bin/go-junit-report > testreport.xml

# Go Build !!
echo "Starting   building..."

CGO_ENABLED=0 GOOS=darwin go build -a -o gocd-golang-agent .
