package libgogoagent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"golang.org/x/net/websocket"
	"io/ioutil"
	// "log"
)

func messageMarshal(v interface{}) ([]byte, byte, error) {
	json, jerr := json.Marshal(v)
	// log.Println(string(json))
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

type Message struct {
	Action string                 `json:"action"`
	Data   map[string]interface{} `json:"data"`
}
