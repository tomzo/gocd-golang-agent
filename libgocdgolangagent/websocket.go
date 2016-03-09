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

package libgocdgolangagent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"github.com/satori/go.uuid"
	"golang.org/x/net/websocket"
	"io/ioutil"
	"time"
)

type Message struct {
	Action string                 `json:"action"`
	Data   map[string]interface{} `json:"data"`
	AckId  string                 `json:"ackId"`
}

type WebsocketConnection struct {
	Conn     *websocket.Conn
	Send     chan *Message
	Received chan *Message
}

func (wc *WebsocketConnection) Close() {
	close(wc.Send)
	err := wc.Conn.Close()
	if err != nil {
		logger.Error.Printf("Close websocket connection failed: %v", err)
	}
}

func MakeMessage(action, dataType string, data map[string]interface{}) *Message {
	return &Message{
		Action: action,
		Data:   map[string]interface{}{"type": dataType, "data": data},
		AckId:  uuid.NewV4().String(),
	}
}

func MakeWebsocketConnection(wsLoc, httpLoc string) (*WebsocketConnection, error) {
	tlsConfig, err := GoServerTlsConfig(true)
	if err != nil {
		return nil, err
	}
	wsConfig, err := websocket.NewConfig(wsLoc, httpLoc)
	if err != nil {
		return nil, err
	}
	wsConfig.TlsConfig = tlsConfig
	LogInfo("connect to: %v", wsLoc)
	ws, err := websocket.DialConfig(wsConfig)
	if err != nil {
		return nil, err
	}
	ack := make(chan string)
	send := make(chan *Message)
	received := make(chan *Message)

	go startReceiveMessage(ws, received, ack)
	go startSendMessage(ws, send, ack)
	return &WebsocketConnection{Conn: ws, Send: send, Received: received}, nil
}

func startSendMessage(ws *websocket.Conn, send chan *Message, ack chan string) {
	defer LogDebug("! exit goroutine: send message")
	connClosed := false
loop:
	select {
	case id := <-ack:
		LogInfo("Ignore ack with id: %v", id)
	case msg, ok := <-send:
		if !ok {
			return
		}
		LogInfo("--> %v", msg.Action)
		LogDebug("message data: %v", msg.Data)
		if connClosed {
			logger.Error.Printf("send message failed: connection is closed")
			goto loop
		}
		if err := MessageCodec.Send(ws, msg); err == nil {
			waitForMessageAck(msg.AckId, ack)
			goto loop
		} else {
			logger.Error.Printf("send message failed: %v", err)
			if err := ws.Close(); err == nil {
				connClosed = true
			} else {
				logger.Error.Printf("Close websocket connection failed: %v", err)
			}
		}
	}
	goto loop
}

func waitForMessageAck(ackId string, ack chan string) {
	timeout := time.NewTimer(config.SendMessageTimeout)
	defer timeout.Stop()
	for {
		select {
		case <-timeout.C:
			LogInfo("wait for message ack timeout, id: %v", ackId)
			return
		case id := <-ack:
			if id == ackId {
				return
			} else {
				LogInfo("ignore ack with id: %v, expected: %v", id, ackId)
			}
		}
	}
}

func startReceiveMessage(ws *websocket.Conn, received chan *Message, ack chan string) {
	defer LogDebug("! exit goroutine: receive message")
	defer close(received)
	for {
		var msg Message
		err := MessageCodec.Receive(ws, &msg)
		if err != nil {
			logger.Error.Printf("receive message failed: %v", err)
			return
		}
		LogInfo("<-- %v", msg.Action)
		LogDebug("message data: %v", msg.Data)

		if msg.Action == "ack" {
			ackId, _ := msg.Data["data"].(string)
			ack <- ackId
		} else {
			received <- &msg
		}
	}
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
