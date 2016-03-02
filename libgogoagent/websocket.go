package libgogoagent

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"golang.org/x/net/websocket"
	"io"
	"io/ioutil"
	"sync"
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
	Connection       *websocket.Conn
	Config           *websocket.Config
	ConnectionClosed bool
	Lock             sync.Mutex
}

func MakeWebSocketMessageClient(wsLoc string, httpLoc string, tlsconf *tls.Config) (*WebSocketMessageClient, error) {
	config, _ := websocket.NewConfig(wsLoc, httpLoc)
	config.TlsConfig = tlsconf
	ws, err := websocket.DialConfig(config)
	return &WebSocketMessageClient{Connection: ws, Config: config}, err
}

func (client *WebSocketMessageClient) Send(msg *Message) error {
	if err := client.Reconnect(); err != nil {
		return err
	}

	err := MessageCodec.Send(client.getConn(), msg)

	if err == io.EOF {
		client.needReconnect()
	}

	return err
}

func (client *WebSocketMessageClient) Receive() (*Message, error) {
	if err := client.Reconnect(); err != nil {
		return nil, err
	}

	var msg Message
	err := MessageCodec.Receive(client.getConn(), &msg)

	if err == io.EOF {
		client.needReconnect()
	}
	return &msg, err
}

func (client *WebSocketMessageClient) getConn() *websocket.Conn {
	client.Lock.Lock()
	defer client.Lock.Unlock()
	return client.Connection
}

func (client *WebSocketMessageClient) needReconnect() {
	client.Lock.Lock()
	client.ConnectionClosed = true
	client.Lock.Unlock()
}

func (client *WebSocketMessageClient) Reconnect() error {
	client.Lock.Lock()
	defer client.Lock.Unlock()
	if client.ConnectionClosed {
		LogInfo("trying to reconnect websocket connection...")
		ws, err := websocket.DialConfig(client.Config)
		if err == nil {
			client.Connection = ws
		}
		client.ConnectionClosed = err != nil
		if err != nil {
			return err
		}
	}
	return nil
}

// func Start(wsLoc string, httpLoc string, tlsconf *tls.Config) (in, out chan Message) {
// 	in, out := make(chan Message), make(chan Message)
// 	config, _ := websocket.NewConfig(wsLoc, httpLoc)
// 	config.TlsConfig = tlsconf

// 	go func() {
// 		reconnect := make(chan bool)
// 		stopSend := make(chan bool)
// 		reconnecting := false
// 		for {
// 			ws, err := websocket.DialConfig(config)
// 			if err != nil {
// 				time.Sleep(10 * time.Seconds)
// 				continue
// 			}
// 			if reconnecting {
// 				stopSend <- true
// 			}
// 			go startReceive(ws, out, reconnect)
// 			go startSend(ws, in, stopSend)
// 			reconnecting = <-reconnect
// 		}
// 	}()
// 	return
// }

// func startSend(ws *websocket.Conn, out chan Message, quit chan bool) {
// 	for {
// 		select {
// 		case msg := <-in:
// 			error := MessageCodec.Send(ws, msg)
// 			if error != nil {
// 				LogInfo("send message failed: %v", error)
// 			}
// 		case t := <-quit:
// 			return
// 		}
// 	}
// }

// func startReceive(ws *websocket.Conn, out chan Message, reconnect chan bool) {
// 	for {
// 		var msg Message
// 		err := MessageCodec.Receive(ws, &msg)
// 		if err != nil {
// 			LogInfo("receive message failed: %v", error)
// 			reconnect <- true
// 			return
// 		}
// 		out <- msg
// 	}
// }
