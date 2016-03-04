package libgogoagent

import (
	"github.com/satori/go.uuid"
	"os"
	"time"
)

var (
	_uuid                     = uuid.NewV4().String()
	serverHost                = readEnv("GOCD_SERVER_HOST", "localhost")
	sslPort                   = readEnv("GOCD_SERVER_SSL_PORT", "8154")
	receivedMessageBufferSize = 10
	sendMessageTimeout        = 120 * time.Second

	agentAutoRegisterKey             = readEnv("GOCD_AGENT_AUTO_REGISTER_KEY", "")
	agentAutoRegisterResources       = readEnv("GOCD_AGENT_AUTO_REGISTER_RESOURCES", "")
	agentAutoRegisterEnvironments    = readEnv("GOCD_AGENT_AUTO_REGISTER_ENVIRONMENTS", "")
	agentAutoRegisterHostname        = readEnv("GOCD_AGENT_AUTO_REGISTER_HOSTNAME", "")
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
	return serverHost + ":" + sslPort
}

func ConfigGetHttpsServerURL(path string) string {
	return "https://" + ConfigGetSslHostAndPort() + path
}

func ConfigGetWsServerURL() string {
	return "wss://" + serverHost + ":8154/go/agent-websocket"
}

func ConfigGetAgentUUID() string {
	return _uuid
}
