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
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Hostname           string
	SendMessageTimeout time.Duration
	ServerUrl          *url.URL
	ServerHostAndPort  string
	ContextPath        string
	WebSocketPath      string
	RegistrationPath   string
	WorkDir            string
	LogDir             string
	ConfigDir          string
	IpAddress          string

	AgentAutoRegisterKey             string
	AgentAutoRegisterResources       string
	AgentAutoRegisterEnvironments    string
	AgentAutoRegisterElasticAgentId  string
	AgentAutoRegisterElasticPluginId string

	GoServerCAFile      string
	AgentPrivateKeyFile string
	AgentCertFile       string
	AgentIdFile         string
	OutputDebugLog      bool

	workingDir string
}

func LoadConfig() *Config {
	gocdServerURL := readEnv("GOCD_SERVER_URL", "https://localhost:8154/go")
	serverUrl, err := url.Parse(gocdServerURL)
	if err != nil {
		panic(err)
	}
	serverUrl.Scheme = "https"
	hostname, _ := os.Hostname()

	return &Config{
		Hostname:                         hostname,
		SendMessageTimeout:               120 * time.Second,
		ServerUrl:                        serverUrl,
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
		AgentIdFile:                      filepath.Join("config", "agent-id"),
		OutputDebugLog:                   os.Getenv("DEBUG") != "",
		WebSocketPath:                    readEnv("GOCD_SERVER_WEB_SOCKET_PATH", "/agent-websocket"),
		RegistrationPath:                 readEnv("GOCD_SERVER_REGISTRATION_PATH", "/admin/agent"),
		IpAddress:                        lookupIpAddress(),
	}
}

func (c *Config) WorkingDir() string {
	if c.workingDir == "" {
		c.workingDir, _ = os.Getwd()
	}
	return c.workingDir
}

func lookupIpAddress() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

func (c *Config) HttpsServerURL() string {
	return c.ServerUrl.String()
}

func (c *Config) WssServerURL() string {
	u, _ := url.Parse(c.HttpsServerURL())
	u.Scheme = "wss"
	return Join("/", u.String(), c.WebSocketPath)
}

func (c *Config) RegistrationURL() (*url.URL, error) {
	return c.MakeFullServerURL(c.RegistrationPath)
}

func (c *Config) MakeFullServerURL(u string) (*url.URL, error) {
	if strings.HasPrefix(u, "/") {
		return url.Parse(Join("/", c.HttpsServerURL(), u))
	} else {
		return url.Parse(u)
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
