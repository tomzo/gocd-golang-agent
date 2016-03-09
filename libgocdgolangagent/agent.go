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
	"errors"
	"net/http"
	"os"
	"time"
)

var buildSession *BuildSession

func StartAgent() {
	if err := os.Chdir(AgentWorkDir()); err != nil {
		panic(err)
	}

	if err := InitLogger(); err != nil {
		panic(err)
	}

	if err := InitConfig(); err != nil {
		LogInfo("%v", err)
		os.Exit(-1)
	}

	for {
		err := doStartAgent()
		if err != nil {
			LogInfo("something wrong: %v", err.Error())
		}
		LogInfo("sleep 10 seconds and restart")
		time.Sleep(10 * time.Second)
	}
}

func closeBuildSession() {
	if buildSession != nil {
		buildSession.Close()
		buildSession = nil
	}
}

func doStartAgent() error {
	err := Register()
	if err != nil {
		return err
	}

	httpClient, err := GoServerRemoteClient(true)
	if err != nil {
		return err
	}

	conn, err := MakeWebsocketConnection(ConfigGetWsServerURL(), ConfigGetHttpsServerURL("/"))
	if err != nil {
		return err
	}
	defer conn.Close()
	defer closeBuildSession()

	pingTick := time.NewTicker(10 * time.Second)
	ping(conn.Send)
	for {
		select {
		case <-pingTick.C:
			ping(conn.Send)
		case msg, ok := <-conn.Received:
			if !ok {
				return errors.New("Websocket connection is closed")
			}
			err := processMessage(msg, httpClient, conn.Send)
			if err != nil {
				return err
			}
		}
	}
}

func processMessage(msg *Message, httpClient *http.Client, send chan *Message) error {
	switch msg.Action {
	case "setCookie":
		str, _ := msg.Data["data"].(string)
		SetState("cookie", str)
	case "cancelJob":
		closeBuildSession()
	case "reregister":
		CleanRegistration()
		return errors.New("received reregister message")
	case "cmd":
		closeBuildSession()
		buildSession = MakeBuildSession(httpClient, send)
		go processBuildCommandMessage(msg, buildSession)
	default:
		LogInfo("ERROR: unknown message action %v", msg)
	}
	return nil
}

func processBuildCommandMessage(msg *Message, buildSession *BuildSession) {
	defer LogDebug("! exit goroutine: process build command message")
	SetState("runtimeStatus", "Building")
	defer SetState("runtimeStatus", "Idle")
	command, _ := msg.Data["data"].(map[string]interface{})
	LogInfo("start process build command")
	err := buildSession.Process(MakeBuildCommand(command))
	if err != nil {
		LogInfo("Error(%v) when processing message : %v", err, msg)
	}
}

func ping(send chan *Message) {
	var msgType string
	if agentAutoRegisterElasticPluginId == "" {
		msgType = "com.thoughtworks.go.server.service.AgentRuntimeInfo"
	} else {
		msgType = "com.thoughtworks.go.server.service.ElasticAgentRuntimeInfo"
	}
	send <- MakeMessage("ping", msgType, AgentRuntimeInfo())
}
