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
set -e
# set -x
SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"
. $SCRIPT_DIR/vars
PROJECT_DIR="${SCRIPT_DIR}/../../"
RPM_BUILD_DIR="${PROJECT_DIR}/tmp_build"
export GOPATH=$PROJECT_DIR

go get golang.org/x/net/websocket
go get github.com/satori/go.uuid
go get github.com/xli/assert
go get github.com/bmatcuk/doublestar
go get github.com/jstemmer/go-junit-report
mkdir -p $PROJECT_DIR/src/github.com/gocd-contrib/gocd-golang-agent/
if [ ! -d $PROJECT_DIR/src/github.com/gocd-contrib/gocd-golang-agent/agent ]; then
  ln -s $PROJECT_DIR/agent $PROJECT_DIR/src/github.com/gocd-contrib/gocd-golang-agent/agent
fi
if [ ! -d $PROJECT_DIR/src/github.com/gocd-contrib/gocd-golang-agent/junit ]; then
  ln -s $PROJECT_DIR/junit $PROJECT_DIR/src/github.com/gocd-contrib/gocd-golang-agent/junit
fi
if [ ! -d $PROJECT_DIR/src/github.com/gocd-contrib/gocd-golang-agent/protocol ]; then
  ln -s $PROJECT_DIR/protocol $PROJECT_DIR/src/github.com/gocd-contrib/gocd-golang-agent/protocol
fi
if [ ! -d $PROJECT_DIR/src/github.com/gocd-contrib/gocd-golang-agent/stream ]; then
  ln -s $PROJECT_DIR/stream $PROJECT_DIR/src/github.com/gocd-contrib/gocd-golang-agent/stream
fi

echo "################################"
echo "Create Temporary Build Structure"
echo "################################"
mkdir -p $RPM_BUILD_DIR/gocd-golang-agent-$GGA_VERSION/opt/gocd-golang-agent/bin
mkdir -p $RPM_BUILD_DIR/gocd-golang-agent-$GGA_VERSION/etc/init.d
mkdir -p $RPM_BUILD_DIR/gocd-golang-agent-$GGA_VERSION/etc/default
mkdir -p $RPM_BUILD_DIR/BUILD
mkdir -p $RPM_BUILD_DIR/BUILDROOT
mkdir -p $RPM_BUILD_DIR/RPMS
mkdir -p $RPM_BUILD_DIR/SOURCES
mkdir -p $RPM_BUILD_DIR/SPECS
mkdir -p $RPM_BUILD_DIR/SRPMS
mkdir -p $PROJECT_DIR/build_installers

echo "############################"
echo "Cross compiling for linux..."
echo "############################"

cd $PROJECT_DIR
CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' .
chmod 0755 gocd-golang-agent
cp gocd-golang-agent $RPM_BUILD_DIR/gocd-golang-agent-$GGA_VERSION/opt/gocd-golang-agent/bin

echo "############################"
echo "Packaging rpm source file..."
echo "############################"
cd $SCRIPT_DIR
install -m 755 agent.sh $RPM_BUILD_DIR/gocd-golang-agent-$GGA_VERSION/opt/gocd-golang-agent/bin
install -m 755 gocd-golang-agent_default $RPM_BUILD_DIR/gocd-golang-agent-$GGA_VERSION/etc/default/gocd-golang-agent
install -m 755 gocd-golang-agent_init.d $RPM_BUILD_DIR/gocd-golang-agent-$GGA_VERSION/etc/init.d/gocd-golang-agent
cat << EOF > $RPM_BUILD_DIR/gocd-golang-agent-$GGA_VERSION/etc/default/gocd-golang-agent
export GOCD_SERVER_URL=https://localhost:8154/go
export GOCD_AGENT_WORKING_DIR=/var/lib/gocd-golang-agent
export GOCD_AGENT_LOG_DIR=/var/log/gocd-golang-agent
EOF
cd $RPM_BUILD_DIR
tar zcvf gocd-golang-agent-$GGA_VERSION.tar.gz gocd-golang-agent-$GGA_VERSION
mv gocd-golang-agent-$GGA_VERSION.tar.gz $RPM_BUILD_DIR/SOURCES
cp $SCRIPT_DIR/gocd-golang-agent.spec $RPM_BUILD_DIR/SPECS
rpmbuild -bb --define '_topdir '$RPM_BUILD_DIR $RPM_BUILD_DIR/SPECS/gocd-golang-agent.spec
mv $RPM_BUILD_DIR/RPMS/`uname -m`/gocd-golang-agent-$GGA_VERSION-$GGA_RPM_VERSION.`uname -m`.rpm $PROJECT_DIR/build_installers
rm -rf $RPM_BUILD_DIR


