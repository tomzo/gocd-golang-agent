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
echo "================"
echo "Get dependencies"
echo "================"

if [[ -d src/golang.org/x/net/websocket ]]; then
  echo "Remove external library : golang.org/x/net/websocket"
  rm -rf src/golang.org/x/net/websocket
fi
echo "Get golang.org/x/net/websocket"
go get golang.org/x/net/websocket

# Pull dependencies used by Test
if [[ -d src/golang.org/x/text ]]; then
  echo "Remove external library : golang.org/x/text"
  rm -rf src/golang.org/x/text
fi
go get golang.org/x/text

if [[ -d src/golang.org/x/crypto/ssh ]]; then
  echo "Remove external library : golang.org/x/crypto/ssh"
  rm -rf src/golang.org/x/crypto/ssh
fi
go get golang.org/x/crypto/ssh

if [[ -f $GOPATH/bin/go-junit-report ]]; then
  echo "Remove binary : $GOPATH/bin/go-junit-report"
  rm -rf $GOPATH/bin/go-junit-report
fi
go build -o $GOPATH/bin/go-junit-report github.com/jstemmer/go-junit-report
if [[ -f testreport.xml ]]; then
  echo "Remove old test report : testreport.xml"
  /bin/rm -rf testreport.xml
fi
go test -test.v github.com/gocd-contrib/gocd-golang-agent... | $GOPATH/bin/go-junit-report > testreport.xml

# Go Build !!
echo "Starting   building..."
/bin/rm -rf output
/bin/mkdir output
CGO_ENABLED=0 GOOS=linux go build -a -o output/gocd-golang-agent_linux_x86 github.com/gocd-contrib/gocd-golang-agent
CGO_ENABLED=0 GOOS=darwin go build -a -o output/gocd-golang-agent_darwin_x86 github.com/gocd-contrib/gocd-golang-agent
