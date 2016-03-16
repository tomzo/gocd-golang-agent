/*
 * Copyright 2016 ThoughtWorks, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package protocal

import (
	"github.com/satori/go.uuid"
)

type Message struct {
	Action string                 `json:"action"`
	Data   map[string]interface{} `json:"data"`
	AckId  string                 `json:"ackId"`
}

func newMessage(action, dataType string, data interface{}) *Message {
	return &Message{
		Action: action,
		Data:   map[string]interface{}{"type": dataType, "data": data},
		AckId:  uuid.NewV4().String(),
	}
}

func SetCookieMessage(cookie string) *Message {
	return newMessage("setCookie", "java.lang.String", cookie)
}

func AckMessage(ackId string) *Message {
	return newMessage("ack", "java.lang.String", ackId)
}

func CmdMessage(cmd *BuildCommand) *Message {
	return newMessage("cmd", "BuildCommand", cmd)
}

func PingMessage(elasticAgent bool, data map[string]interface{}) *Message {
	var msgType string
	if elasticAgent {
		msgType = "com.thoughtworks.go.server.service.AgentRuntimeInfo"
	} else {
		msgType = "com.thoughtworks.go.server.service.ElasticAgentRuntimeInfo"
	}
	return newMessage("ping", msgType, data)
}

func ReportMessage(t string, report map[string]interface{}) *Message {
	return newMessage(t, "com.thoughtworks.go.websocket.Report", report)
}

func ReregisterMessage() *Message {
	return &Message{Action: "reregister"}
}
