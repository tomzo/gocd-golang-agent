#!/bin/bash
set -x
set -e

SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"
PROJECT_DIR="$SCRIPT_DIR/../../"
cd $PROJECT_DIR
CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' .
mkdir -p $SCRIPT_DIR/DEBIAN/usr/share/bin
cp gogoagent $SCRIPT_DIR/DEBIAN/usr/share/bin/gocd-agent
chmod 0755 $SCRIPT_DIR/DEBIAN/usr/share/bin/gocd-agent
cd $SCRIPT_DIR
docker build -t gocd/deb-maker .
docker run -v ${PWD}:/build gocd/deb-maker /bin/bash -c "cd /build/package && fakeroot dpkg-deb --build . ../gocd-agent_0.1-1_all.deb"
