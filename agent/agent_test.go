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
package agent_test

import (
	"bytes"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	. "github.com/gocd-contrib/gocd-golang-agent/agent"
	"github.com/gocd-contrib/gocd-golang-agent/protocal"
	"github.com/gocd-contrib/gocd-golang-agent/server"
	"github.com/xli/assert"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	testFileContentMD5 = "41e43efb30d3fbfcea93542157809ac0"

	goServerUrl  string
	goServer     *server.Server
	stateLog     *StateLog
	buildId      string
	agentStopped chan bool
)

func TestSetCookieAfterConnected(t *testing.T) {
	setUp(t)
	defer tearDown()
	goServer.SendBuild(AgentId, buildId, protocal.EchoCommand("hello"))

	assert.Equal(t, "agent Building", stateLog.Next())
	assert.NotEqual(t, "", GetState("cookie"))
	assert.Equal(t, "agent Idle", stateLog.Next())
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
	os.Setenv("GOCD_SERVER_URL", goServerUrl)
	os.Setenv("GOCD_SERVER_WEB_SOCKET_PATH", server.WebSocketPath)
	os.Setenv("GOCD_SERVER_REGISTRATION_PATH", server.RegistrationPath)
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
	goServerUrl = "https://" + cert.Host + ":" + port
	goServer = server.New(port,
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
	if err := waitForServerStarted(goServerUrl + server.StatusPath); err != nil {
		panic(err)
	}
	println("server started")
}

func waitForServerStarted(url string) error {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	timeout := time.After(5 * time.Second)
	for {
		select {
		case <-timeout:
			return errors.New("wait for server start timeout")
		default:
			_, err := client.Get(url)
			if err == nil {
				return nil
			}
		}
	}
}

type StateLog struct {
	states           chan string
	mu               sync.Mutex
	buildId, agentId string
}

func (log *StateLog) Notify(class, id, state string) {
	log.mu.Lock()
	defer log.mu.Unlock()
	switch class {
	case "agent":
		if id == log.agentId {
			log.states <- "agent " + state
		}
	case "build":
		if id == log.buildId {
			log.states <- "build " + state
		}
	}
}

func (log *StateLog) Next() string {
	select {
	case state := <-log.states:
		return state
	case <-time.After(5 * time.Second):
		return "timeout"
	}
}

func (log *StateLog) Reset(buildId, agentId string) {
	log.mu.Lock()
	defer log.mu.Unlock()
	log.buildId = buildId
	log.agentId = agentId
}

func contains(s1, s2 string) bool {
	return strings.Contains(s1, s2)
}

func split(s1, s2 string) []string {
	return strings.Split(s1, s2)
}

func startWith(s1, s2 string) bool {
	return strings.HasPrefix(s1, s2)
}

func sprintf(f string, args ...interface{}) string {
	return fmt.Sprintf(f, args...)
}

func trimTimestamp(log string) string {
	lines := strings.Split(log, "\n")
	var buf bytes.Buffer
	for _, l := range lines {
		if len(l) > 13 {
			buf.WriteString(l[13:])
			buf.WriteString("\n")
		}
	}
	return buf.String()
}

func createPipelineDir() string {
	dir := pipelineDir()
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		panic(err)
	}
	return dir
}

func createTestProjectInPipelineDir() string {
	root := createPipelineDir()
	err := os.MkdirAll(root+"/src/hello", 0777)
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(root+"/test/world", 0777)
	if err != nil {
		panic(err)
	}
	createTestFile(root, "0.txt")
	createTestFile(root+"/src", "1.txt")
	createTestFile(root+"/src", "2.txt")
	createTestFile(root+"/src/hello", "3.txt")
	createTestFile(root+"/src/hello", "4.txt")
	createTestFile(root+"/test", "5.txt")
	createTestFile(root+"/test", "6.txt")
	createTestFile(root+"/test", "7.txt")
	createTestFile(root+"/test/world", "8.txt")
	createTestFile(root+"/test/world", "9.txt")
	createTestFile(root+"/test/world", "10.txt")
	createTestFile(root+"/test/world", "11.txt")
	createTestFile(root+"/test/world2", "10.txt")
	createTestFile(root+"/test/world2", "11.txt")
	return root
}

func createTestFile(dir, fname string) string {
	err := writeFile(dir, fname, "file created for test")
	if err != nil {
		panic(err)
	}
	return fname
}

func writeFile(dir, fname, content string) error {
	err := os.MkdirAll(dir, 0744)
	if err != nil {
		return err
	}
	fpath := filepath.Join(dir, fname)
	f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE, 0744)
	if err != nil {
		return err
	}
	data := []byte(content)
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		return io.ErrShortWrite
	}
	return f.Close()
}

func pipelineDir() string {
	return os.Getenv("GOCD_AGENT_WORK_DIR") + "/pipelines/" + buildId
}

func startAgent(t *testing.T) chan bool {
	done := make(chan bool)
	go func() {
		err := Start()
		if err.Error() != "received reregister message" {
			t.Error("Unexpected error to quit agent: ", err)
		}
		close(done)
	}()
	assert.Equal(t, "agent Idle", stateLog.Next())
	return done
}

func setUp(t *testing.T) {
	pc, _, _, _ := runtime.Caller(1)
	_func := runtime.FuncForPC(pc)
	parts := strings.Split(_func.Name(), ".")

	buildId = parts[len(parts)-1]
	stateLog.Reset(buildId, AgentId)
	agentStopped = startAgent(t)
}

func tearDown() {
	goServer.Send(AgentId, protocal.ReregisterMessage())
	<-agentStopped
	err := os.RemoveAll(pipelineDir())
	if err != nil {
		println("WARN: clean up pipeline directory failed:", err.Error())
	}
}
