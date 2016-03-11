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
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var (
	server     *Server
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
		t.Fatal("expected Idle, but get: %v", state)
	}

	start := &BuildCommand{
		Name: "start",
		Args: []interface{}{map[string]interface{}{
			"buildId":                buildId,
			"buildLocator":           "p/1/s/1/j",
			"buildLocatorForDisplay": "p/1/s/1/j",
			"consoleURI":             server.ConsoleUrl(buildId),
			"artifactUploadBaseUrl":  server.ArtifactUploadBaseUrl(buildId),
			"propertyBaseUrl":        server.PropertyBaseUrl(buildId),
		}},
		RunIfConfig: "any",
	}
	reportCurrentStatus := &BuildCommand{
		Name:        "reportCurrentStatus",
		Args:        []interface{}{"Building"},
		RunIfConfig: "passed",
	}
	echo := &BuildCommand{
		Name:        "echo",
		Args:        []interface{}{"echo hello world"},
		RunIfConfig: "passed",
	}
	end := &BuildCommand{Name: "end", RunIfConfig: "any"}

	compose := &BuildCommand{
		Name: "compose",
		SubCommands: []*BuildCommand{
			start,
			reportCurrentStatus,
			echo,
			end},
		RunIfConfig: "any",
	}
	server.Send(UUID, MakeMessage("cmd", "BuildCommand", compose))
	state = agentState.Next()
	if state != "Building" {
		t.Fatal("expected Building, but get: %v", state)
	}
	if GetState("cookie") == "" {
		t.Fatal("cookie is not set")
	}
	state = agentState.Next()
	if state != "Idle" {
		t.Fatal("expected Idle, but get: %v", state)
	}
	log, err := server.ConsoleLog(buildId)
	if err != nil {
		t.Fatal("can't get console log: %v", err)
	}
	if !strings.Contains(string(log), "echo hello world") {
		t.Fatal("echo command is not processed")
	}
	server.Send(UUID, &Message{Action: "reregister"})
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

	server = startServer(serverWorkingDir)
	os.Setenv("DEBUG", "t")
	os.Setenv("GOCD_SERVER_URL", server.URL)
	os.Setenv("GOCD_AGENT_WORK_DIR", agentWorkingDir)
	os.Setenv("GOCD_AGENT_LOG_DIR", agentWorkingDir)

	Initialize()

	os.Exit(m.Run())
}

func startServer(workingDir string) *Server {
	certFile := filepath.Join(workingDir, "cert.pem")
	keyFile := filepath.Join(workingDir, "private.pem")
	cert := MakeCert()
	err := cert.Generate(certFile, keyFile)
	if err != nil {
		panic(err)
	}
	port := "1234"
	agentState = AgentState{
		states: make(chan string),
	}
	server := &Server{
		Port:                port,
		URL:                 "https://" + cert.Host + ":" + port,
		CertPemFile:         certFile,
		KeyPemFile:          keyFile,
		WorkingDir:          workingDir,
		Logger:              MakeLogger(workingDir, "server.log", true),
		AgentStateListeners: []AgentStateListener{agentState},
	}

	go func() { panic(server.Start()) }()

	if err := server.WaitForStarted(); err != nil {
		panic(err)
	}

	return server
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
	return <-as.states
}
