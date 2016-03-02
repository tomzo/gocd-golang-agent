package libgogoagent

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
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

type WebSocketMessageClient struct {
	Connection *websocket.Conn
}

func MakeWebSocketMessageClient(wsLoc string, httpLoc string, tlsconf *tls.Config) (*WebSocketMessageClient, error) {
	config, _ := websocket.NewConfig(wsLoc, httpLoc)
	config.TlsConfig = tlsconf
	ws, err := websocket.DialConfig(config)
	return &WebSocketMessageClient{ws}, err
}

func (client *WebSocketMessageClient) Send(msg *Message) error {
	return MessageCodec.Send(client.Connection, msg)
}

func (client *WebSocketMessageClient) Receive() (*Message, error) {
	var msg Message
	err := MessageCodec.Receive(client.Connection, &msg)
	return &msg, err
}
