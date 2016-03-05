package libgogoagent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
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

type WebsocketConnection struct {
	Conn     *websocket.Conn
	Received chan Message
	Ack      chan int
}

func (wc *WebsocketConnection) Send(msg *Message) error {
	msg.AckId = uuid.NewV4().String()
	LogInfo("--> %v", msg.Action)
	LogDebug("message data: %v", msg.Data)
	err := MessageCodec.Send(wc.Conn, msg)
	if err != nil {
		return err
	}
	timeout := time.NewTimer(sendMessageTimeout)
	defer timeout.Stop()
	select {
	case <-timeout.C:
		return errors.New("send message timeout")
	case _, ok := <-wc.Ack:
		if ok {
			return nil
		} else {
			return errors.New("Ack channel is closed.")
		}
	}
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
	received := make(chan Message, receivedMessageBufferSize)
	ack := make(chan int, 1)
	go startReceiveMessage(ws, received, ack)
	return &WebsocketConnection{Conn: ws, Received: received, Ack: ack}, nil
}

func startReceiveMessage(ws *websocket.Conn, received chan Message, ack chan int) {
	defer close(received)
	defer close(ack)
	for {
		var msg Message
		err := MessageCodec.Receive(ws, &msg)
		if err != nil {
			LogInfo("stop reading message due to error: %v", err)
			return
		}
		LogInfo("<-- %v", msg.Action)
		LogDebug("message data: %v", msg.Data)
		if msg.Action == "ack" {
			ack <- 1
		} else {
			if len(received) == cap(received) {
				LogInfo("Something is wrong, too many received messages are queued up.")
				LogInfo("Received messages buffer size: %v", cap(received))
				return
			} else {
				received <- msg
			}
		}
	}
}
