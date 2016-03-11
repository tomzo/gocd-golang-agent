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
	"github.com/satori/go.uuid"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

var buildSession *BuildSession
var logger *Logger
var config *Config
var UUID string

func LogDebug(format string, v ...interface{}) {
	logger.Debug.Printf(format, v...)
}

func LogInfo(format string, v ...interface{}) {
	logger.Info.Printf(format, v...)
}

func Initialize() {
	config = LoadConfig()
	logger = MakeLogger(config.LogDir, "gocd-golang-agent.log", config.OutputDebugLog)
	LogInfo(">>>>>>> go >>>>>>>")
	if config.WorkDir != "" {
		if err := os.Chdir(config.WorkDir); err != nil {
			logger.Error.Fatal(err)
		}
	}
	if err := os.MkdirAll(config.ConfigDir, 0744); err != nil {
		logger.Error.Fatal(err)
	}

	if _, err := os.Stat(config.UuidFile); err == nil {
		data, err2 := ioutil.ReadFile(config.UuidFile)
		if err2 != nil {
			logger.Error.Printf("failed to read uuid file(%v): %v", config.UuidFile, err2)
		} else {
			UUID = string(data)
		}
	}
	if UUID == "" {
		UUID = uuid.NewV4().String()
		ioutil.WriteFile(config.UuidFile, []byte(UUID), 0600)
	}
}

func Start() error {
	err := Register()
	if err != nil {
		return err
	}

	httpClient, err := GoServerRemoteClient(true)
	if err != nil {
		return err
	}

	conn, err := MakeWebsocketConnection(config.WsServerURL(), config.HttpsServerURL("/"))
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
		logger.Error.Printf("ERROR: unknown message action %v", msg)
	}
	return nil
}

func processBuildCommandMessage(msg *Message, buildSession *BuildSession) {
	defer logger.Debug.Printf("! exit goroutine: process build command message")
	SetState("runtimeStatus", "Building")
	defer SetState("runtimeStatus", "Idle")
	command, _ := msg.Data["data"].(map[string]interface{})
	buildCmd := MakeBuildCommand(command)
	LogInfo("start process build command:")
	LogInfo(buildCmd.dump(2, 2))
	err := buildSession.Process(buildCmd)
	if err != nil {
		LogInfo("Error(%v) when processing message : %v", err, msg)
	}
}

func ping(send chan *Message) {
	var msgType string
	if config.AgentAutoRegisterElasticPluginId == "" {
		msgType = "com.thoughtworks.go.server.service.AgentRuntimeInfo"
	} else {
		msgType = "com.thoughtworks.go.server.service.ElasticAgentRuntimeInfo"
	}
	send <- MakeMessage("ping", msgType, AgentRuntimeInfo())
}

func closeBuildSession() {
	if buildSession != nil {
		buildSession.Close()
		buildSession = nil
	}
}
