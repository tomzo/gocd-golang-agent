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
	if err := Mkdirs(config.ConfigDir); err != nil {
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
		ioutil.WriteFile(config.AgentIdFile, []byte(AgentId), 0644)
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
	case protocal.SetCookieAction:
		SetState("cookie", msg.StringData())
	case protocal.CancelJobAction:
		closeBuildSession()
	case protocal.ReregisterAction:
		CleanRegistration()
		return errors.New("received reregister message")
	case protocal.BuildAction:
		closeBuildSession()
		build := msg.Build()
		SetState("buildLocator", build.BuildLocator)
		SetState("buildLocatorForDisplay", build.BuildLocatorForDisplay)
		curl, err := config.MakeFullServerURL(build.ConsoleURI)
		if err != nil {
			return err
		}
		aurl, err := config.MakeFullServerURL(build.ArtifactUploadBaseUrl)
		if err != nil {
			return err
		}

		buildSession = MakeBuildSession(
			build.BuildId,
			build.BuildCommand,
			MakeBuildConsole(httpClient, curl),
			NewUploader(httpClient, aurl),
			send,
		)
		go processBuild(send, buildSession)
	default:
		logger.Error.Printf("ERROR: unknown message action %v", msg)
	}
	return nil
}

func processBuild(send chan *protocal.Message, buildSession *BuildSession) {
	defer func() {
		SetState("runtimeStatus", "Idle")
		ping(send)
		logger.Debug.Printf("! exit goroutine: process build command message")
	}()
	SetState("runtimeStatus", "Building")
	ping(send)
	err := buildSession.Process()
	if err != nil {
		LogInfo("Processing build failed: %v", err)
	}
	LogInfo("done")
}

func ping(send chan *protocal.Message) {
	send <- protocal.PingMessage(GetAgentRuntimeInfo())
}

func closeBuildSession() {
	if buildSession != nil {
		buildSession.Close()
		buildSession = nil
	}
}
