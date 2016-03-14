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
	"flag"
	"github.com/gocd-contrib/gocd-golang-agent/protocal"
	"github.com/gocd-contrib/gocd-golang-agent/server"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var (
	goServer   *server.Server
	agentState AgentState
)

func Test(t *testing.T) {
	buildId := "1"
	done := make(chan bool)
	go func() {
		t.Log("start agent")
		err := Start()
		if err.Error() != "received reregister message" {
			t.Error("Unexpected error to quit agent: ", err)
		}
		close(done)
	}()

	t.Log("wait for agent register to server")
	state := agentState.Next()
	if state != "Idle" {
		t.Fatal("expected Idle, but get: ", state)
	}

	start := protocal.StartCommand(map[string]string{
		"buildId":                buildId,
		"buildLocator":           "p/1/s/1/j",
		"buildLocatorForDisplay": "p/1/s/1/j",
		"consoleURI":             goServer.ConsoleUrl(buildId),
		"artifactUploadBaseUrl":  goServer.ArtifactUploadBaseUrl(buildId),
		"propertyBaseUrl":        goServer.PropertyBaseUrl(buildId),
	})
	reportCurrentStatus := protocal.ReportCurrentStatusCommand("Building")
	echo := protocal.EchoCommand("echo hello world")
	end := protocal.EndCommand()

	compose := protocal.ComposeCommand(start,
		reportCurrentStatus,
		echo,
		end).RunIf("any")
	goServer.Send(UUID, protocal.CmdMessage(compose))
	state = agentState.Next()
	if state != "Building" {
		t.Fatal("expected Building, but get: ", state)
	}
	if GetState("cookie") == "" {
		t.Fatal("cookie is not set")
	}
	state = agentState.Next()
	if state != "Idle" {
		t.Fatal("expected Idle, but get: ", state)
	}
	log, err := goServer.ConsoleLog(buildId)
	if err != nil {
		t.Fatal("can't get console log: ", err)
	}
	if !strings.Contains(string(log), "echo hello world") {
		t.Fatal("echo command is not processed")
	}
	goServer.Send(UUID, &protocal.Message{Action: "reregister"})
	<-done
}

func TestMain(m *testing.M) {
	flag.Parse()

	workingDir, err := ioutil.TempDir("", "gocd-golang-agent")
	if err != nil {
		panic(err)
	}
	println("working directories:", workingDir)
	serverWorkingDir := filepath.Join(workingDir, "server")
	agentWorkingDir := filepath.Join(workingDir, "agent")

	err = os.MkdirAll(serverWorkingDir, 0777)
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(agentWorkingDir, 0777)
	if err != nil {
		panic(err)
	}

	startServer(serverWorkingDir)
	os.Setenv("DEBUG", "t")
	os.Setenv("GOCD_SERVER_URL", goServer.URL)
	os.Setenv("GOCD_SERVER_WEB_SOCKET_PATH", goServer.WebSocketPath())
	os.Setenv("GOCD_SERVER_REGISTRATION_PATH", goServer.RegistrationPath())
	os.Setenv("GOCD_AGENT_WORK_DIR", agentWorkingDir)
	os.Setenv("GOCD_AGENT_LOG_DIR", agentWorkingDir)

	println("initialize agent")
	Initialize()

	os.Exit(m.Run())
}

func startServer(workingDir string) {
	certFile := filepath.Join(workingDir, "cert.pem")
	keyFile := filepath.Join(workingDir, "private.pem")
	cert := server.NewCert()
	err := cert.Generate(certFile, keyFile)
	if err != nil {
		panic(err)
	}
	port := "1234"
	agentState = AgentState{
		states: make(chan string),
	}
	goServer = server.New(port,
		"https://"+cert.Host+":"+port,
		certFile,
		keyFile,
		workingDir,
		MakeLogger(workingDir, "server.log", true).Info,
		[]server.AgentStateListener{agentState})

	go func() {
		e := goServer.Start()
		panic(e.Error())
	}()

	println("wait for server started")
	if err := goServer.WaitForStarted(); err != nil {
		panic(err)
	}
}

type AgentState struct {
	states chan string
}

func (as AgentState) Notify(agentUuid, state string) {
	if agentUuid == UUID {
		as.states <- state
	}
}

func (as AgentState) Next() string {
	select {
	case state := <-as.states:
		return state
	case <-time.After(5 * time.Second):
		return "timeout"
	}
}
