package libgogoagent

import (
	"github.com/satori/go.uuid"
	"os"
)

var (
	uuid       = uuid.NewV4().String()
	serverHost = readEnv("GOCD_SERVER_HOST", "localhost")
	sslPort    = readEnv("GOCD_SERVER_SSL_PORT", "8154")
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
	return uuid
}
