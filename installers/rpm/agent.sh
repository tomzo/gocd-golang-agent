#!/bin/bash
#*************************GO-LICENSE-START********************************
# Copyright 2016 ThoughtWorks, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#*************************GO-LICENSE-END**********************************


SERVICE_NAME=${1:-gocd-golang-agent}

if [ -f /etc/default/${SERVICE_NAME} ]; then
  echo "[`date`] using default settings from /etc/default/${SERVICE_NAME}"
  . /etc/default/${SERVICE_NAME}
fi

CWD=`dirname "$0"`

GOCD_SERVER_URL=${GOCD_SERVER_URL:-"https://127.0.0.1:8154/go"}
VNC=${VNC:-"N"}
GOCD_AGENT_WORK_DIR=${GOCD_AGENT_WORK_DIR:-"/var/lib/gocd-golang-agent"}

if [ ! -d "${GOCD_AGENT_WORK_DIR}" ]; then
    echo Agent working directory ${GOCD_AGENT_WORK_DIR} does not exist
    exit 2
fi

GOCD_AGENT_LOG_DIR=/var/log/${SERVICE_NAME}

STDOUT_LOG_FILE=$GOCD_AGENT_LOG_DIR/${SERVICE_NAME}.log

PID_FILE="$GOCD_AGENT_WORK_DIR/gocd-golang-agent.pid"


if [ "$VNC" == "Y" ]; then
    echo "[`date`] Starting up VNC on :3"
    /usr/bin/vncserver :3
    DISPLAY=:3
    export DISPLAY
fi

export GOCD_AGENT_LOG_DIR
export LOG_FILE

CMD="/opt/${SERVICE_NAME}/bin/${SERVICE_NAME}"

cd "$GOCD_AGENT_WORK_DIR"

if [ "$DAEMON" == "Y" ]; then
    eval exec nohup "$CMD" >>"$STDOUT_LOG_FILE" 2>&1 &
    echo $! >"$PID_FILE"
else
    eval exec "$CMD"
fi

