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
	"encoding/json"
	"golang.org/x/net/websocket"
	"io/ioutil"
	"log"
	"net/http"
)

type Server struct {
	Port         string
	CertPemFile  string
	KeyPemFile   string
	Handle       func(*RemoteAgent)
	OnConsoleLog func(data string, err error)
}

func (s *Server) Start() error {
	http.Handle("/go/agent-websocket", s.websocketHandler())
	http.HandleFunc("/console", s.consoleHandler())
	http.HandleFunc("/go/admin/agent", s.registorHandler())

	return http.ListenAndServeTLS(":"+s.Port, s.CertPemFile, s.KeyPemFile, nil)
}

func (s *Server) registorHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		var agentPrivateKey, agentCert, regJson []byte
		var err error
		var reg *Registration

		agentPrivateKey, err = ioutil.ReadFile(s.KeyPemFile)
		if err != nil {
			goto responseError
		}
		agentCert, err = ioutil.ReadFile(s.CertPemFile)
		if err != nil {
			goto responseError
		}
		reg = &Registration{
			AgentPrivateKey:  string(agentPrivateKey),
			AgentCertificate: string(agentCert),
		}
		regJson, err = json.Marshal(reg)
		if err != nil {
			goto responseError
		}
		w.Write(regJson)
		return
	responseError:
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func (s *Server) websocketHandler() websocket.Handler {
	return websocket.Handler(func(ws *websocket.Conn) {
		agent := &RemoteAgent{conn: ws}
		defer func() {
			log.Printf("close websocket connection for %v", agent)
			err := agent.Close()
			if err != nil {
				log.Printf("error when closing websocket connection for %v: %v", agent, err)
			}
		}()
		log.Printf("websocket connection is open for %v", agent)
		s.Handle(agent)
	})
}

func (s *Server) consoleHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		bytes, err := ioutil.ReadAll(req.Body)
		s.OnConsoleLog(string(bytes), err)
	}
}

type RemoteAgent struct {
	conn *websocket.Conn
}

func (agent *RemoteAgent) Send(msg *Message) error {
	return MessageCodec.Send(agent.conn, msg)
}

func (agent *RemoteAgent) Ack(ackId string) error {
	return agent.Send(MakeMessage("ack", "java.lang.String", ackId))
}

func (agent *RemoteAgent) Receive() (*Message, error) {
	var msg Message
	err := MessageCodec.Receive(agent.conn, &msg)
	return &msg, err
}

func (agent *RemoteAgent) String() string {
	return agent.conn.RemoteAddr().String()
}

func (agent *RemoteAgent) Close() error {
	return agent.conn.Close()
}
