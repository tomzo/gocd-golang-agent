package libgocdgolangagent

import (
	"github.com/satori/go.uuid"
	"net/url"
	"os"
	"time"
)

var (
	_uuid                     = uuid.NewV4().String()
	serverUrl, _              = url.Parse(readEnv("GOCD_SERVER_URL", "https://localhost:8154"))
	serverHostAndPort         = serverUrl.Host
	receivedMessageBufferSize = 10
	sendMessageTimeout        = 120 * time.Second

	agentAutoRegisterKey             = readEnv("GOCD_AGENT_AUTO_REGISTER_KEY", "")
	agentAutoRegisterResources       = readEnv("GOCD_AGENT_AUTO_REGISTER_RESOURCES", "")
	agentAutoRegisterEnvironments    = readEnv("GOCD_AGENT_AUTO_REGISTER_ENVIRONMENTS", "")
	agentAutoRegisterElasticAgentId  = readEnv("GOCD_AGENT_AUTO_REGISTER_ELASTIC_AGENT_ID", "")
	agentAutoRegisterElasticPluginId = readEnv("GOCD_AGENT_AUTO_REGISTER_ELASTIC_PLUGIN_ID", "")
)

func readEnv(varname string, defaultVal string) string {
	val := os.Getenv(varname)
	LogInfo("env %v=%v", varname, val)
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
