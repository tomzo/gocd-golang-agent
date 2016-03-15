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
	"runtime"
	"strings"
	"testing"
	"time"
)

var (
	goServer *server.Server
	stateLog *StateLog
	buildId  = "1"
)

func TestReportStatusAndSetCookieAfterConnected(t *testing.T) {
	done := startAgent(t)
	compose := protocal.ComposeCommand(
		startCmd(),
		protocal.ReportCurrentStatusCommand("Preparing"),
		protocal.ReportCurrentStatusCommand("Building"),
		protocal.ReportCompletingCommand(),
		protocal.ReportCompletedCommand(),
		protocal.EndCommand(),
	)
	goServer.Send(UUID, protocal.CmdMessage(compose))

	waitForNextState(t, "agent Building")
	if GetState("cookie") == "" {
		t.Fatal("cookie is not set")
	}

	waitForNextState(t, "build Preparing")

	waitForNextState(t, "agent Building")
	waitForNextState(t, "build Building")

	waitForNextState(t, "agent Building")
	waitForNextState(t, "build Passed")

	waitForNextState(t, "agent Building")
	waitForNextState(t, "build Passed")

	waitForNextState(t, "agent Idle")

	goServer.Send(UUID, protocal.ReregisterMessage())
	<-done
}

func TestEcho(t *testing.T) {
	done := startAgent(t)

	compose := protocal.ComposeCommand(
		startCmd(),
		protocal.EchoCommand("echo hello world"),
		protocal.EndCommand(),
	)
	goServer.Send(UUID, protocal.CmdMessage(compose))
	waitForNextState(t, "agent Idle")

	log, err := goServer.ConsoleLog(buildId)
	if err != nil {
		t.Fatal("can't get console log: ", err)
	}
	if !strings.Contains(string(log), "echo hello world") {
		t.Fatalf("console log dos not contain echo content: %v", string(log))
	}

	goServer.Send(UUID, protocal.ReregisterMessage())
	<-done
}

func startAgent(t *testing.T) chan bool {
	done := make(chan bool)
	go func() {
		t.Log("start agent")
		err := Start()
		if err.Error() != "received reregister message" {
			t.Error("Unexpected error to quit agent: ", err)
		}
		close(done)
	}()
	waitForNextState(t, "agent Idle")
	return done
}

func waitForNextState(t *testing.T, expected string) {
	state := stateLog.Next()
	if expected != state {
		_, file, line, _ := runtime.Caller(1)
		finfo, _ := os.Stat(file)
		t.Fatalf("expected agent state: %v, but get: %v\n%v:%v:", expected, state, finfo.Name(), line)
	}
}

func startCmd() *protocal.BuildCommand {
	return protocal.StartCommand(map[string]string{
		"buildId":                buildId,
		"buildLocator":           "p/1/s/1/j",
		"buildLocatorForDisplay": "p/1/s/1/j",
		"consoleURI":             goServer.ConsoleUrl(buildId),
		"artifactUploadBaseUrl":  goServer.ArtifactUploadBaseUrl(buildId),
		"propertyBaseUrl":        goServer.PropertyBaseUrl(buildId),
	})
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
	stateLog = &StateLog{states: make(chan string)}
	goServer = server.New(port,
		"https://"+cert.Host+":"+port,
		certFile,
		keyFile,
		workingDir,
		MakeLogger(workingDir, "server.log", true).Info)
	goServer.StateListeners = []server.StateListener{stateLog}

	go func() {
		e := goServer.Start()
		panic(e.Error())
	}()

	println("wait for server started")
	if err := goServer.WaitForStarted(); err != nil {
		panic(err)
	}
	println("server started")
}

type StateLog struct {
	states chan string
}

func (as *StateLog) Notify(class, id, state string) {
	switch class {
	case "agent":
		if id == UUID {
			as.states <- "agent " + state
		}
	case "build":
		if id == buildId {
			as.states <- "build " + state
		}
	}
}

func (as *StateLog) Next() string {
	select {
	case state := <-as.states:
		return state
	case <-time.After(5 * time.Second):
		return "timeout"
	}
}
