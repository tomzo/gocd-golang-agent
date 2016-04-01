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

package protocol

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"golang.org/x/net/websocket"
	"io/ioutil"
)

func messageMarshal(v interface{}) ([]byte, byte, error) {
	json, jerr := json.Marshal(v)
	if jerr != nil {
		return []byte{}, websocket.BinaryFrame, jerr
	}
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, err := w.Write(json)
	w.Close()

	return b.Bytes(), websocket.BinaryFrame, err
}

func messageUnmarshal(msg []byte, payloadType byte, v interface{}) (err error) {
	reader, _ := gzip.NewReader(bytes.NewBuffer(msg))
	jsonBytes, _ := ioutil.ReadAll(reader)
	return json.Unmarshal(jsonBytes, v)
}

var messageCodec = websocket.Codec{messageMarshal, messageUnmarshal}

func ReceiveMessage(conn *websocket.Conn) (*Message, error) {
	var msg Message
	err := messageCodec.Receive(conn, &msg)
	return &msg, err
}

func SendMessage(conn *websocket.Conn, msg *Message) error {
	return messageCodec.Send(conn, msg)
}
