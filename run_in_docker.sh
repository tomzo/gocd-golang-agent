#!/bin/bash
set -x
CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' .
docker build -t gogoagent .

GOCD_SERVER_HOST=${GOCD_SERVER_HOST:-`ipconfig getifaddr en0 || ipconfig getifaddr en1`}
GOCD_SERVER_SSL_PORT=${GOCD_SERVER_SSL_PORT:-8154}

docker run -e GOCD_SERVER_HOST="$GOCD_SERVER_HOST" -e GOCD_SERVER_SSL_PORT="$GOCD_SERVER_SSL_PORT" gogoagent
