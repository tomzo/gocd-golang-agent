package libgogoagent

import (
	"errors"
	"net/http"
	"os"
	"runtime"
	"time"
)

var buildSession *BuildSession

func registerData() map[string]string {
	hostname, _ := os.Hostname()
	workingDir, _ := os.Getwd()

	return map[string]string{
		"hostname":                      hostname,
		"uuid":                          ConfigGetAgentUUID(),
		"location":                      workingDir,
		"operatingSystem":               runtime.GOOS,
		"usablespace":                   "5000000000",
		"agentAutoRegisterKey":          "",
		"agentAutoRegisterResources":    "",
		"agentAutoRegisterEnvironments": "",
		"agentAutoRegisterHostname":     "",
		"elasticAgentId":                "",
		"elasticPluginId":               "",
	}
}

func StartAgent() {
	for {
		err := doStartAgent()
		if err != nil {
			LogInfo("something wrong: %v", err.Error())
		}
		LogInfo("sleep 10 seconds and restart")
		time.Sleep(10 * time.Second)
	}
}

func closeBuildSession() {
	if buildSession != nil {
		buildSession.Close()
		buildSession = nil
	}
}

func doStartAgent() error {
	err := Register(registerData())
	if err != nil {
		return err
	}

	httpClient, err := GoServerRemoteClient(true)
	if err != nil {
		return err
	}
	conn, err := MakeWebsocketConnection(ConfigGetWsServerURL(), ConfigGetHttpsServerURL("/"))
	if err != nil {
		return err
	}
	defer conn.Close()
	defer closeBuildSession()

	send := make(chan *Message, 1)
	defer close(send)
	pingTick := time.NewTicker(10 * time.Second)
	defer pingTick.Stop()
	for {
		select {
		case <-pingTick.C:
			send <- MakeMessage("ping",
				"com.thoughtworks.go.server.service.AgentRuntimeInfo",
				AgentRuntimeInfo())
		case msg := <-send:
			LogInfo("send %v message", msg.Action)
			err := conn.Send(msg)
			if err != nil {
				return err
			}
		case msg := <-conn.Received:
			LogInfo("received %v message", msg.Action)
			err := processMessage(msg, httpClient, send)
			if err != nil {
				return err
			}
		}
	}
}

func processMessage(msg *Message, httpClient *http.Client, send chan *Message) error {
	switch msg.Action {
	case "setCookie":
		str, _ := msg.Data["data"].(string)
		SetState("cookie", str)
	case "cancelJob":
		closeBuildSession()
	case "reregister":
		CleanRegistration()
		return errors.New("registration problem")
	case "cmd":
		closeBuildSession()
		buildSession = MakeBuildSession(httpClient, send)
		go processBuildCommandMessage(msg, buildSession)
	default:
		LogInfo("ERROR: unknown message action %v", msg)
	}
	return nil
}

func processBuildCommandMessage(msg *Message, buildSession *BuildSession) {
	SetState("runtimeStatus", "Building")
	command, _ := msg.Data["data"].(map[string]interface{})
	LogInfo("start process build command")
	err := buildSession.Process(MakeBuildCommand(command))
	SetState("runtimeStatus", "Idle")
	if err != nil {
		LogInfo("Error(%v) when processing message : %v", err, msg)
	}
}
