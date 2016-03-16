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

package agent

import (
	"errors"
	"github.com/gocd-contrib/gocd-golang-agent/protocal"
	"github.com/satori/go.uuid"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

var buildSession *BuildSession
var logger *Logger
var config *Config
var AgentId string

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

	if _, err := os.Stat(config.AgentIdFile); err == nil {
		data, err2 := ioutil.ReadFile(config.AgentIdFile)
		if err2 != nil {
			logger.Error.Printf("failed to read uuid file(%v): %v", config.AgentIdFile, err2)
		} else {
			AgentId = string(data)
		}
	}
	if AgentId == "" {
		AgentId = uuid.NewV4().String()
		ioutil.WriteFile(config.AgentIdFile, []byte(AgentId), 0600)
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

	conn, err := MakeWebsocketConnection(config.WssServerURL(), config.HttpsServerURL())
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

func processMessage(msg *protocal.Message, httpClient *http.Client, send chan *protocal.Message) error {
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
		buildSession = MakeBuildSession(httpClient, send, config)
		go processBuildCommandMessage(msg, buildSession)
	default:
		logger.Error.Printf("ERROR: unknown message action %v", msg)
	}
	return nil
}

func processBuildCommandMessage(msg *protocal.Message, buildSession *BuildSession) {
	defer func() {
		SetState("runtimeStatus", "Idle")
		ping(buildSession.Send)
		logger.Debug.Printf("! exit goroutine: process build command message")
	}()
	SetState("runtimeStatus", "Building")
	ping(buildSession.Send)
	command, _ := msg.Data["data"].(map[string]interface{})
	buildCmd := protocal.Parse(command)
	LogInfo("start process build command:")
	LogInfo(buildCmd.Dump(2, 2))
	err := buildSession.Process(buildCmd)
	if err != nil {
		LogInfo("Error(%v) when processing message : %v", err, msg)
	} else {
		LogInfo("done")
	}
}

func ping(send chan *protocal.Message) {
	send <- protocal.PingMessage(
		config.IsElasticAgent(),
		AgentRuntimeInfo(),
	)
}

func closeBuildSession() {
	if buildSession != nil {
		buildSession.Close()
		buildSession = nil
	}
}
