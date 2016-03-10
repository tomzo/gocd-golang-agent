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
SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"
. $SCRIPT_DIR/vars
$SCRIPT_DIR/build.sh

echo "############################"
echo "Publishing package to bintray repos"
echo "############################"

curl -T "$DEB_FILENAME" -ualex-hal9000:${GGA_RELEASE_KEY:?"need set GGA_RELEASE_KEY"} "https://api.bintray.com/content/alex-hal9000/gocd-golang-agent/gocd-golang-agent/${GGA_VERSION}/${DEB_FILENAME};deb_distribution=master;deb_component=main;deb_architecture=amd64;publish=1"

echo
echo "############################"
echo "testing new version of package is accessible"
echo "############################"
echo "Wait 30 seconds to package info get updated in bintray service. You can ctrl-c now if you do not want verify the published package"
sleep 30
docker run gocd/deb-maker /bin/bash -c "echo 'deb https://dl.bintray.com/alex-hal9000/gocd-golang-agent master main' > /etc/apt/sources.list && apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys 379CE192D401AB61 && apt-get update && apt-get -y --force-yes install gocd-golang-agent && dpkg -s gocd-golang-agent"
