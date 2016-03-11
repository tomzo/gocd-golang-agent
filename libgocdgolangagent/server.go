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
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/satori/go.uuid"
	"golang.org/x/net/websocket"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type AgentStateListener interface {
	Notify(uuid, state string)
}

type Server struct {
	Port                string
	URL                 string
	CertPemFile         string
	KeyPemFile          string
	WorkingDir          string
	Logger              *Logger
	AgentStateListeners []AgentStateListener
	addAgent            chan *RemoteAgent
	delAgent            chan *RemoteAgent
	sendMessage         chan *RemoteAgentMessage
}

func (s *Server) Start() error {
	s.addAgent = make(chan *RemoteAgent)
	s.delAgent = make(chan *RemoteAgent)
	s.sendMessage = make(chan *RemoteAgentMessage)

	go s.manageAgents()

	http.Handle("/go/agent-websocket", s.websocketHandler())
	http.HandleFunc("/go/console", s.consoleHandler())
	http.HandleFunc("/go/admin/agent", s.registorHandler())
	http.HandleFunc("/go/status", s.statusHandler())
	s.Log("listen to %v", s.Port)
	return http.ListenAndServeTLS(":"+s.Port, s.CertPemFile, s.KeyPemFile, nil)
}

func (s *Server) ConsoleUrl(buildId string) string {
	return s.URL + "/go/console?buildId=" + buildId
}

func (s *Server) ArtifactUploadBaseUrl(buildId string) string {
	return s.URL + "/go/artifacts/" + buildId
}
func (s *Server) PropertyBaseUrl(buildId string) string {
	return s.URL + "/go/property/" + buildId
}

func (s *Server) StatusUrl() string {
	return s.URL + "/go/status"
}

func (s *Server) manageAgents() {
	agents := make(map[string]*RemoteAgent)
	messages := make(map[string][]*Message)
	for {
		select {
		case agent := <-s.addAgent:
			agents[agent.UUID] = agent
			for _, msg := range messages[agent.UUID] {
				agent.Send(msg)
			}
			delete(messages, agent.UUID)
		case agent := <-s.delAgent:
			delete(agents, agent.UUID)
		case am := <-s.sendMessage:
			agent := agents[am.UUID]
			if agent != nil {
				agent.Send(am.Msg)
			} else {
				messages[am.UUID] = append(messages[am.UUID], am.Msg)
			}
		}
	}
}

func (s *Server) WaitForStarted() error {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	timeout := time.After(10 * time.Second)
	for {
		select {
		case <-timeout:
			return errors.New("wait for server start timeout")
		default:
			_, err := client.Get(s.StatusUrl())
			if err == nil {
				return nil
			}
		}
	}
}

func (s *Server) statusHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("ok"))
	}
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
			s.Del(agent)
			s.Log("close websocket connection for %v", agent)
			err := agent.Close()
			if err != nil {
				s.Log("error when closing websocket connection for %v: %v", agent, err)
			}
		}()
		s.Log("websocket connection is open for %v", agent)
		agent.Listen(s)
	})
}

func (s *Server) ConsoleLog(buildId string) ([]byte, error) {
	return ioutil.ReadFile(s.consoleLogFile(buildId))
}

func (s *Server) consoleHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		bytes, err := ioutil.ReadAll(req.Body)
		if err != nil {
			s.Log("read request body error: %v", err)
			return
		}
		buildId := req.URL.Query().Get("buildId")
		err = s.appendConsoleLog(s.consoleLogFile(buildId), bytes)
		if err != nil {
			s.Log("append console log error: %v", err)
			return
		}
	}
}

func (s *Server) consoleLogFile(buildId string) string {
	return filepath.Join(s.WorkingDir, buildId, "console.log")
}

func (s *Server) appendConsoleLog(filename string, data []byte) error {
	err := os.MkdirAll(filepath.Dir(filename), 0744)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0744)
	if err != nil {
		return err
	}
	s.Log("append data(%v) to %v", len(data), filename)
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}

func (s *Server) Log(format string, v ...interface{}) {
	s.Logger.Info.Printf(format, v...)
}

func (s *Server) Error(format string, v ...interface{}) {
	s.Logger.Error.Printf(format, v...)
}

func (s *Server) Send(uid string, msg *Message) {
	s.sendMessage <- &RemoteAgentMessage{UUID: uid, Msg: msg}
}

func (s *Server) Add(agent *RemoteAgent) {
	s.addAgent <- agent
}

func (s *Server) Del(agent *RemoteAgent) {
	s.delAgent <- agent
}

func (s *Server) Notify(uuid, state string) {
	for _, listener := range s.AgentStateListeners {
		listener.Notify(uuid, state)
	}
}

type RemoteAgent struct {
	conn *websocket.Conn
	UUID string
}

func (agent *RemoteAgent) Listen(server *Server) {
	for {
		var msg Message
		err := MessageCodec.Receive(agent.conn, &msg)
		if err == io.EOF {
			return
		} else if err != nil {
			server.Error("receive error: %v", err)
		} else {
			server.Log("received message: %v", msg.Action)
			err = agent.Ack(&msg)
			if err != nil {
				server.Error("ack error: %v", err)
			}
			switch msg.Action {
			case "ping":
				if agent.UUID == "" {
					agent.UUID = AgentUUID(msg.Data["data"])
					server.Add(agent)
					agent.SetCookie()
				}
				server.Notify(agent.UUID, AgentRuntimeStatus(msg.Data["data"]))
			case "reportCurrentStatus":
				report := msg.Data["data"].(map[string]interface{})
				state := AgentRuntimeStatus(report["agentRuntimeInfo"])
				server.Notify(agent.UUID, state)
			}
		}
	}
}

func (agent *RemoteAgent) Send(msg *Message) error {
	return MessageCodec.Send(agent.conn, msg)
}

func (agent *RemoteAgent) SetCookie() error {
	return agent.Send(MakeMessage("setCookie", "java.lang.String", uuid.NewV4().String()))
}

func (agent *RemoteAgent) Ack(msg *Message) error {
	if msg.AckId != "" {
		return agent.Send(MakeMessage("ack", "java.lang.String", msg.AckId))
	}
	return nil
}

func (agent *RemoteAgent) String() string {
	return fmt.Sprintf("[agent %v, uuid: %v]",
		agent.conn.RemoteAddr(), agent.UUID)
}

func (agent *RemoteAgent) Close() error {
	return agent.conn.Close()
}

type RemoteAgentMessage struct {
	UUID string
	Msg  *Message
}
