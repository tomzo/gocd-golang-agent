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

var (
	_uuid                            = uuid.NewV4().String()
	sendMessageTimeout               = 120 * time.Second
	serverUrl, _                     = url.Parse(readEnv("GOCD_SERVER_URL", "https://localhost:8154"))
	serverHostAndPort                = serverUrl.Host
	agentWorkDir                     = os.Getenv("GOCD_AGENT_WORK_DIR")
	agentLogDir                      = os.Getenv("GOCD_AGENT_LOG_DIR")
	agentAutoRegisterKey             = os.Getenv("GOCD_AGENT_AUTO_REGISTER_KEY")
	agentAutoRegisterResources       = os.Getenv("GOCD_AGENT_AUTO_REGISTER_RESOURCES")
	agentAutoRegisterEnvironments    = os.Getenv("GOCD_AGENT_AUTO_REGISTER_ENVIRONMENTS")
	agentAutoRegisterElasticAgentId  = os.Getenv("GOCD_AGENT_AUTO_REGISTER_ELASTIC_AGENT_ID")
	agentAutoRegisterElasticPluginId = os.Getenv("GOCD_AGENT_AUTO_REGISTER_ELASTIC_PLUGIN_ID")
	goServerCAFile                   = ConfigFilePath("go-server-ca.pem")
	agentPrivateKeyFile              = ConfigFilePath("agent-private-key.pem")
	agentCertFile                    = ConfigFilePath("agent-cert.pem")
	outputDebugLog                   = os.Getenv("DEBUG") != ""
)

func readEnv(varname string, defaultVal string) string {
	val := os.Getenv(varname)
	if val == "" {
		return defaultVal
	} else {
		return val
	}
}

func ConfigGetSslHostAndPort() string {
	return serverHostAndPort
}

func ConfigGetHttpsServerURL(path string) string {
	return "https://" + ConfigGetSslHostAndPort() + path
}

func ConfigGetWsServerURL() string {
	return "wss://" + ConfigGetSslHostAndPort() + "/go/agent-websocket"
}

func ConfigGetAgentUUID() string {
	return _uuid
}

func ConfigFilePath(fileName string) string {
	return filepath.Join("config", fileName)
}

func ConfigMakeFullServerURL(url string) string {
	if strings.HasPrefix(url, "/") {
		return ConfigGetHttpsServerURL(url)
	} else {
		return url
	}
}
