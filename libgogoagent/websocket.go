package libgogoagent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"golang.org/x/net/websocket"
	"io/ioutil"
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

type WebsocketConnection struct {
	Conn     *websocket.Conn
	Received chan *Message
}

func (wc *WebsocketConnection) Send(msg *Message) error {
	return MessageCodec.Send(wc.Conn, msg)
}

func (wc *WebsocketConnection) Close() {
	err := wc.Conn.Close()
	if err != nil {
		LogInfo("Close websocket connection failed: %v", err)
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
	received := make(chan *Message, 1)
	go startReceiveMessage(ws, received)
	return &WebsocketConnection{Conn: ws, Received: received}, nil
}

func startReceiveMessage(ws *websocket.Conn, received chan *Message) {
	defer close(received)
	for {
		var msg Message
		err := MessageCodec.Receive(ws, &msg)
		if err != nil {
			LogInfo("stop reading message due to error: %v", err)
			return
		}
		received <- &msg
	}
}
