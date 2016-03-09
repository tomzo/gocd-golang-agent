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
	"github.com/satori/go.uuid"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	UUID               string
	SendMessageTimeout time.Duration
	ServerHostAndPort  string
	WorkDir            string
	LogDir             string
	ConfigDir          string

	AgentAutoRegisterKey             string
	AgentAutoRegisterResources       string
	AgentAutoRegisterEnvironments    string
	AgentAutoRegisterElasticAgentId  string
	AgentAutoRegisterElasticPluginId string

	GoServerCAFile      string
	AgentPrivateKeyFile string
	AgentCertFile       string
	OutputDebugLog      bool
}

func LoadConfig() *Config {
	serverUrl, _ := url.Parse(readEnv("GOCD_SERVER_URL", "https://localhost:8154"))
	return &Config{
		UUID:                             uuid.NewV4().String(),
		SendMessageTimeout:               120 * time.Second,
		ServerHostAndPort:                serverUrl.Host,
		WorkDir:                          os.Getenv("GOCD_AGENT_WORK_DIR"),
		LogDir:                           os.Getenv("GOCD_AGENT_LOG_DIR"),
		ConfigDir:                        readEnv("GOCD_AGENT_CONFIG_DIR", "config"),
		AgentAutoRegisterKey:             os.Getenv("GOCD_AGENT_AUTO_REGISTER_KEY"),
		AgentAutoRegisterResources:       os.Getenv("GOCD_AGENT_AUTO_REGISTER_RESOURCES"),
		AgentAutoRegisterEnvironments:    os.Getenv("GOCD_AGENT_AUTO_REGISTER_ENVIRONMENTS"),
		AgentAutoRegisterElasticAgentId:  os.Getenv("GOCD_AGENT_AUTO_REGISTER_ELASTIC_AGENT_ID"),
		AgentAutoRegisterElasticPluginId: os.Getenv("GOCD_AGENT_AUTO_REGISTER_ELASTIC_PLUGIN_ID"),
		GoServerCAFile:                   filepath.Join("config", "go-server-ca.pem"),
		AgentPrivateKeyFile:              filepath.Join("config", "agent-private-key.pem"),
		AgentCertFile:                    filepath.Join("config", "agent-cert.pem"),
		OutputDebugLog:                   os.Getenv("DEBUG") != "",
	}
}

func (c *Config) HttpsServerURL(path string) string {
	return "https://" + c.ServerHostAndPort + path
}

func (c *Config) WsServerURL() string {
	return "wss://" + c.ServerHostAndPort + "/go/agent-websocket"
}

func (c *Config) MakeFullServerURL(url string) string {
	if strings.HasPrefix(url, "/") {
		return c.HttpsServerURL(url)
	} else {
		return url
	}
}

func readEnv(varname string, defaultVal string) string {
	val := os.Getenv(varname)
	if val == "" {
		return defaultVal
	} else {
		return val
	}
}
