#!/bin/bash
set -e

SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"
PROJECT_DIR="$SCRIPT_DIR/../../"
BINARY_FILE="$SCRIPT_DIR/package/usr/bin/gocd-agent"
DEB_FILE="gocd-agent_0.1-1_all.deb"

echo "############################"
echo "Cross compiling for linux..."
echo "############################"

cd $PROJECT_DIR
CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' .
mkdir -p `dirname $BINARY_FILE`
cp gogoagent $BINARY_FILE
chmod 0755 $BINARY_FILE

echo "############################"
echo "Packaging to deb file..."
echo "############################"

cd $SCRIPT_DIR
docker build -t gocd/deb-maker .
docker run -v ${PWD}:/build gocd/deb-maker /bin/bash -c "cd /build/package && fakeroot dpkg-deb --build . ../$DEB_FILE"

echo "############################"
echo "Making sure installer can be installed..."
echo "############################"

docker run -v ${PWD}:/build gocd/deb-maker /bin/bash -c "cd /build && dpkg -i $DEB_FILE && service gocd-agent start && sleep 3 && ps aux | grep -v 'grep' | grep gocd-agent"

echo "All check passed."
