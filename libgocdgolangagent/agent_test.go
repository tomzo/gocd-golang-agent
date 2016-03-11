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
	"testing"
)

var (
	server *Server
)

func TestAgent(t *testing.T) {
	Initialize()
	logger.Debug.SetPrefix("[agent]")
	logger.Info.SetPrefix("[agent]")
	logger.Error.SetPrefix("[agent]")
	done := make(chan bool)
	go func() {
		err := Start()
		if err.Error() != "received reregister message" {
			t.Error("Unexpected error to quit agent: ", err)
		}
		done <- true
	}()
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
	os.Setenv("GOCD_SERVER_URL", server.URL)
	os.Setenv("GOCD_AGENT_WORK_DIR", agentWorkingDir)
	os.Setenv("GOCD_AGENT_LOG_DIR", agentWorkingDir)

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
	server := &Server{
		Port:        port,
		URL:         "https://" + cert.Host + ":" + port,
		CertPemFile: certFile,
		KeyPemFile:  keyFile,
		WorkingDir:  workingDir,
		Logger:      MakeLogger(workingDir, "server.log", true),
	}

	go func() { panic(server.Start()) }()
	return server
}
