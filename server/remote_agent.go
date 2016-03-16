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
	"fmt"
	"github.com/gocd-contrib/gocd-golang-agent/protocal"
	"github.com/satori/go.uuid"
	"golang.org/x/net/websocket"
	"io"
)

type RemoteAgent struct {
	conn *websocket.Conn
	id   string
}

func (agent *RemoteAgent) Listen(server *Server) error {
	for {
		msg, err := protocal.ReceiveMessage(agent.conn)
		if err == io.EOF {
			return err
		} else if err != nil {
			server.error("receive error: %v", err)
		} else {
			agent.processMessage(server, msg)
		}
	}
}

func (agent *RemoteAgent) processMessage(server *Server, msg *protocal.Message) {
	server.log("received message: %v", msg.Action)
	err := agent.Ack(msg)
	if err != nil {
		server.error("ack error: %v", err)
	}
	switch msg.Action {
	case "ping":
		if agent.id == "" {
			agent.id = protocal.AgentId(msg.Data["data"])
			server.add(agent)
			agent.SetCookie()
		}
		agentState := protocal.AgentRuntimeStatus(msg.Data["data"])
		server.notifyAgent(agent.id, agentState)
	case "reportCurrentStatus":
		report := msg.Data["data"].(map[string]interface{})
		agentState := protocal.AgentRuntimeStatus(report["agentRuntimeInfo"])
		server.notifyAgent(agent.id, agentState)
		buildId, _ := report["buildId"].(string)
		jobState, _ := report["jobState"].(string)
		server.notifyBuild(buildId, jobState)
	case "reportCompleting", "reportCompleted":
		report := msg.Data["data"].(map[string]interface{})
		agentState := protocal.AgentRuntimeStatus(report["agentRuntimeInfo"])
		server.notifyAgent(agent.id, agentState)
		buildId, _ := report["buildId"].(string)
		jobResult, _ := report["result"].(string)
		server.notifyBuild(buildId, jobResult)
	}
}

func (agent *RemoteAgent) Send(msg *protocal.Message) error {
	return protocal.SendMessage(agent.conn, msg)
}

func (agent *RemoteAgent) SetCookie() error {
	return agent.Send(protocal.SetCookieMessage(uuid.NewV4().String()))
}

func (agent *RemoteAgent) Ack(msg *protocal.Message) error {
	if msg.AckId != "" {
		return agent.Send(protocal.AckMessage(msg.AckId))
	}
	return nil
}

func (agent *RemoteAgent) String() string {
	return fmt.Sprintf("[agent %v, id: %v]",
		agent.conn.RemoteAddr(), agent.id)
}

func (agent *RemoteAgent) Close() error {
	return agent.conn.Close()
}
