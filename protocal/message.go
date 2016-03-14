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
	"bytes"
	"compress/gzip"
	"encoding/json"
	"github.com/satori/go.uuid"
	"golang.org/x/net/websocket"
	"io/ioutil"
)

type Message struct {
	Action string                 `json:"action"`
	Data   map[string]interface{} `json:"data"`
	AckId  string                 `json:"ackId"`
}

func NewMessage(action, dataType string, data interface{}) *Message {
	return &Message{
		Action: action,
		Data:   map[string]interface{}{"type": dataType, "data": data},
		AckId:  uuid.NewV4().String(),
	}
}

func SetCookieMessage(cookie string) *Message {
	return NewMessage("setCookie", "java.lang.String", cookie)
}

func AckMessage(ackId string) *Message {
	return NewMessage("ack", "java.lang.String", ackId)
}

func CmdMessage(cmd *BuildCommand) *Message {
	return NewMessage("cmd", "BuildCommand", cmd)
}

func PingMessage(elasticAgent bool, data map[string]interface{}) *Message {
	var msgType string
	if elasticAgent {
		msgType = "com.thoughtworks.go.server.service.AgentRuntimeInfo"
	} else {
		msgType = "com.thoughtworks.go.server.service.ElasticAgentRuntimeInfo"
	}
	return NewMessage("ping", msgType, data)
}

func ReportMessage(t string, report map[string]interface{}) *Message {
	return NewMessage(t, "com.thoughtworks.go.websocket.Report", report)
}

func messageMarshal(v interface{}) ([]byte, byte, error) {
	json, jerr := json.Marshal(v)
	if jerr != nil {
		return []byte{}, websocket.BinaryFrame, jerr
	}
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, err := w.Write([]byte(json))
	w.Close()

	return b.Bytes(), websocket.BinaryFrame, err
}

func messageUnmarshal(msg []byte, payloadType byte, v interface{}) (err error) {
	reader, _ := gzip.NewReader(bytes.NewBuffer(msg))
	jsonBytes, _ := ioutil.ReadAll(reader)
	return json.Unmarshal(jsonBytes, v)
}

var MessageCodec = websocket.Codec{messageMarshal, messageUnmarshal}
