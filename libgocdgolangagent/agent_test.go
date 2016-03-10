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
	agentWorkDir string
)

func TestAgent(t *testing.T) {
	certFile := tmpFile("test-server-cert.pem")
	keyFile := tmpFile("test-server-private.pem")
	cert := MakeCert()
	err := cert.Generate(certFile, keyFile)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{
		Port:        "1234",
		CertPemFile: certFile,
		KeyPemFile:  keyFile,
		Handle: func(agent *RemoteAgent) {
			var err error
			var msg *Message
			for {
				msg, err = agent.Receive()
				if err != nil {
					t.Logf("receive error: %v", err)
					return
				}
				t.Log(msg)
				err = agent.Ack(msg.AckId)
				if err != nil {
					t.Logf("ack error: %v", err)
					return
				}
				err = agent.Send(&Message{Action: "reregister"})
				if err != nil {
					t.Logf("send message error: %v", err)
					return
				}
			}
		},
		OnConsoleLog: func(str string, err error) {
			if err == nil {
				t.Logf("console log: %v", str)
			} else {
				t.Logf("console log error: %v", err)
			}
		},
	}

	go func() {
		err = server.Start()
		if err != nil {
			t.Fatal(err)
		}
	}()

	os.Setenv("GOCD_SERVER_URL", "https://"+cert.Host+":"+server.Port)
	os.Setenv("GOCD_AGENT_WORK_DIR", agentWorkDir)
	os.Setenv("GOCD_AGENT_DIR_DIR", agentWorkDir)

	Initialize()
	err = Start()
	if err.Error() != "received reregister message" {
		t.Error("Unexpected error to quit agent: ", err)
	}
}

func TestMain(m *testing.M) {
	flag.Parse()

	var err error
	agentWorkDir, err = ioutil.TempDir("", "gocdgolangagent")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(agentWorkDir)

	os.Exit(m.Run())
}

func tmpFile(name string) string {
	return filepath.Join(agentWorkDir, name)
}
