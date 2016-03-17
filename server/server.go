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

package server

import (
	"encoding/json"
	"github.com/gocd-contrib/gocd-golang-agent/protocal"
	"golang.org/x/net/websocket"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	WebSocketPath    = "/agent-websocket"
	RegistrationPath = "/agent-register"
	StatusPath       = "/status"

	ConsoleLogPath = "/console"
	ArtifactsPath  = "/artifacts"
	PropertiesPath = "/properties"
)

type StateListener interface {
	Notify(class, id, state string)
}

type AgentMessage struct {
	agentId string
	Msg     *protocal.Message
}

type Server struct {
	Port           string
	CertPemFile    string
	KeyPemFile     string
	WorkingDir     string
	Logger         *log.Logger
	StateListeners []StateListener

	addAgent    chan *RemoteAgent
	delAgent    chan *RemoteAgent
	sendMessage chan *AgentMessage
}

func New(port, certFile, keyFile, workingDir string, logger *log.Logger) *Server {
	return &Server{
		Port:        port,
		CertPemFile: certFile,
		KeyPemFile:  keyFile,
		WorkingDir:  workingDir,
		Logger:      logger,
		addAgent:    make(chan *RemoteAgent),
		delAgent:    make(chan *RemoteAgent),
		sendMessage: make(chan *AgentMessage),
	}

}

func (s *Server) Start() error {
	go manageAgents(s)
	http.HandleFunc(RegistrationPath, registorHandler(s))
	http.Handle(WebSocketPath, websocketHandler(s))
	http.HandleFunc(ConsoleLogPath+"/", consoleHandler(s))
	http.HandleFunc(ArtifactsPath+"/", artifactsHandler(s))
	http.HandleFunc(StatusPath, statusHandler())
	s.log("listen to %v", s.Port)
	return http.ListenAndServeTLS(":"+s.Port, s.CertPemFile, s.KeyPemFile, nil)
}

func (s *Server) BuildContext(id string) map[string]string {
	locator := "/builds/" + id
	return map[string]string{
		"buildId":                id,
		"buildLocator":           locator,
		"buildLocatorForDisplay": locator,
		"consoleURI":             ConsoleLogPath + locator,
		"artifactUploadBaseUrl":  ArtifactsPath + locator,
		"propertyBaseUrl":        PropertiesPath + locator,
	}
}

func (s *Server) ConsoleLog(buildId string) (string, error) {
	bytes, err := ioutil.ReadFile(s.ConsoleLogFile(buildId))
	return string(bytes), err
}

func (s *Server) Checksum(buildId string) (string, error) {
	bytes, err := ioutil.ReadFile(s.ChecksumFile(buildId))
	return string(bytes), err
}

func (s *Server) ArtifactFile(buildId, file string) string {
	return filepath.Join(s.WorkingDir, buildId, "artifacts", file)
}

func (s *Server) ChecksumFile(buildId string) string {
	return filepath.Join(s.WorkingDir, buildId, "md5.checksum")
}

func (s *Server) ConsoleLogFile(buildId string) string {
	return filepath.Join(s.WorkingDir, buildId, "console.log")
}

func (s *Server) Send(agentId string, msg *protocal.Message) {
	s.sendMessage <- &AgentMessage{agentId: agentId, Msg: msg}
}

func (s *Server) log(format string, v ...interface{}) {
	s.Logger.Printf(format, v...)
}

func (s *Server) error(format string, v ...interface{}) {
	s.Logger.Printf(format, v...)
}

func (s *Server) add(agent *RemoteAgent) {
	s.addAgent <- agent
}

func (s *Server) del(agent *RemoteAgent) {
	s.delAgent <- agent
}

func (s *Server) notifyAgent(uuid, state string) {
	s.notify("agent", uuid, state)
}

func (s *Server) notifyBuild(uuid, state string) {
	s.notify("build", uuid, state)
}

func (s *Server) notify(class, uuid, state string) {
	for _, listener := range s.StateListeners {
		listener.Notify(class, uuid, state)
	}
}

func (s *Server) appendToFile(filename string, data []byte) error {
	err := os.MkdirAll(filepath.Dir(filename), 0744)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0744)
	if err != nil {
		return err
	}
	s.log("append data(%v) to %v", len(data), filename)
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}

func manageAgents(s *Server) {
	agents := make(map[string]*RemoteAgent)
	for {
		select {
		case agent := <-s.addAgent:
			agents[agent.id] = agent
		case agent := <-s.delAgent:
			delete(agents, agent.id)
		case am := <-s.sendMessage:
			agent := agents[am.agentId]
			if agent != nil {
				agent.Send(am.Msg)
			} else {
				s.log("could not find agent by id %v for sending message: %v", am.agentId, am.Msg.Action)
			}
		}
	}
}

func statusHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("ok"))
	}
}

// todo: does not generate real agent cert and private key yet, just
// use server cert and private key for testing environment.
func registorHandler(s *Server) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		var agentPrivateKey, agentCert, regJson []byte
		var err error
		var reg *protocal.Registration

		agentPrivateKey, err = ioutil.ReadFile(s.KeyPemFile)
		if err != nil {
			s.responseInternalError(err, w)
			return
		}
		agentCert, err = ioutil.ReadFile(s.CertPemFile)
		if err != nil {
			s.responseInternalError(err, w)
			return
		}

		reg = &protocal.Registration{
			AgentPrivateKey:  string(agentPrivateKey),
			AgentCertificate: string(agentCert),
		}
		regJson, err = json.Marshal(reg)
		if err != nil {
			s.responseInternalError(err, w)
			return
		}
		w.Write(regJson)
	}
}

func websocketHandler(s *Server) websocket.Handler {
	return websocket.Handler(func(ws *websocket.Conn) {
		agent := &RemoteAgent{conn: ws}
		s.log("websocket connection is open for %v", agent)
		err := agent.Listen(s)
		s.del(agent)
		if err != io.EOF {
			s.log("close websocket connection for %v", agent)
			err := agent.Close()
			if err != nil {
				s.error("error when closing websocket connection for %v: %v", agent, err)
			}
		}
	})
}

func parseBuildId(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}
