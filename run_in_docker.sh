#!/bin/bash
set -x

detectIP() {
    for i in 0 1 2 3 4 5 6 7 8 9
    do
        ip=`ipconfig getifaddr en${i}`
        if [ "${ip}" ]
        then
            echo $ip
            return
        fi
    done
}


CGO_ENABLED=0 GOOS=linux go build -a -tags netgo -ldflags '-w' .
docker build -t gogoagent .

GOCD_SERVER_HOST=${GOCD_SERVER_HOST:-`detectIP`}
GOCD_SERVER_SSL_PORT=${GOCD_SERVER_SSL_PORT:-8154}

docker run -e GOCD_SERVER_URL="https://$GOCD_SERVER_HOST:$GOCD_SERVER_SSL_PORT" -e DEBUG="${DEBUG}" gogoagent /goagent
