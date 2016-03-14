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
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	SendMessageTimeout time.Duration
	ServerHostAndPort  string
	WebSocketPath      string
	RegistrationPath   string
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
	UuidFile            string
	OutputDebugLog      bool
}

func LoadConfig() *Config {
	serverUrl, _ := url.Parse(readEnv("GOCD_SERVER_URL", "https://localhost:8154"))
	return &Config{
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
		UuidFile:                         filepath.Join("config", "uuid"),
		OutputDebugLog:                   os.Getenv("DEBUG") != "",
		WebSocketPath:                    readEnv("GOCD_SERVER_WEB_SOCKET_PATH", "/go/agent-websocket"),
		RegistrationPath:                 readEnv("GOCD_SERVER_REGISTRATION_PATH", "/go/admin/agent"),
	}
}

func (c *Config) HttpsServerURL() string {
	return "https://" + c.ServerHostAndPort
}

func (c *Config) WssServerURL() string {
	return "wss://" + c.ServerHostAndPort + c.WebSocketPath
}

func (c *Config) RegistrationURL() string {
	return c.MakeFullServerURL(c.RegistrationPath)
}

func (c *Config) MakeFullServerURL(url string) string {
	if strings.HasPrefix(url, "/") {
		return c.HttpsServerURL() + url
	} else {
		return url
	}
}

func (c *Config) IsElasticAgent() bool {
	return config.AgentAutoRegisterElasticPluginId == ""
}

func readEnv(varname string, defaultVal string) string {
	val := os.Getenv(varname)
	if val == "" {
		return defaultVal
	} else {
		return val
	}
}
