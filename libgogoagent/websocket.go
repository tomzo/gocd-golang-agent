package libgogoagent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"golang.org/x/net/websocket"
	"io/ioutil"
	"time"
)

type Message struct {
	Action string                 `json:"action"`
	Data   map[string]interface{} `json:"data"`
}

func MakeMessage(action string, dataType string, data map[string]interface{}) *Message {
	return &Message{action, map[string]interface{}{"type": dataType, "data": data}}
}

func messageMarshal(v interface{}) ([]byte, byte, error) {
	LogDebug("--> [ws]: %v", v)
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
	defer LogDebug("<-- [ws]: %v", v)
	return json.Unmarshal(jsonBytes, v)
}

var MessageCodec = websocket.Codec{messageMarshal, messageUnmarshal}

func StartWebsocket(wsLoc string, httpLoc string) (send chan *Message, received chan *Message) {
	send, recieved := make(chan *Message), make(chan *Message)
	go func() {
	connect:
		LogInfo("connect to: %v", wsLoc)
		config, err := websocket.NewConfig(wsLoc, httpLoc)
		if err != nil {
			LogInfo("Cannot create websocket config with [%v, %v]", wsLoc, httpLoc)
			panic(err)
		}
		config.TlsConfig = GoServerTlsConfig(true)
		ws, err := websocket.DialConfig(config)
		if err != nil {
			time.Sleep(10 * time.Second)
			goto connect
		}
		go func() {
		receiveMessage:
			var msg Message
			err := MessageCodec.Receive(ws, &msg)
			if err == nil {
				received <- &msg
				goto receiveMessage
			}
			LogInfo("receive message failed: %v", err)
		}()
	sendMessage:
		msg := <-send
		error := MessageCodec.Send(ws, msg)
		if error != nil {
			LogInfo("send message failed: %v", error)
			if closeErr := ws.Close(); closeErr != nil {
				LogInfo("Close websocket connection failed: %v", closeErr)
			}
			goto connect
		}
		goto sendMessage
	}()
	return send, recieved
}
