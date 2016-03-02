package libgogoagent

import (
	"errors"
	"os"
	"runtime"
	"time"
)

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
	send := make(chan *Message)
	go ping(send)
	for {
		err := doStart(send)
		if err != nil {
			LogInfo(err.Error())
		}
		time.Sleep(10 * time.Second)
	}
}

func doStart(send chan *Message) error {
	err := Register(registerData())
	if err != nil {
		return err
	}

	httpClient, err := GoServerRemoteClient(true)
	if err != nil {
		return err
	}
	buildSession := &BuildSession{
		HttpClient: httpClient,
		Send:       send,
	}
	defer buildSession.Close()

	conn, err := MakeWebsocketConnection(ConfigGetWsServerURL(), ConfigGetHttpsServerURL("/"))
	if err != nil {
		return err
	}
	defer conn.Close()
	for {
		select {
		case msg := <-send:
			err := conn.Send(msg)
			if err != nil {
				return err
			}
		case msg := <-conn.Received:
			err := processMessage(msg, buildSession)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func processMessage(msg *Message, buildSession *BuildSession) error {
	switch msg.Action {
	case "setCookie":
		str, _ := msg.Data["data"].(string)
		SetState("cookie", str)
	case "cancelJob":
		buildSession.Cancel()
	case "reregister":
		CleanRegistration()
		return errors.New("received reregister message")
	case "cmd":
		if "Building" == GetState("runtimeStatus") {
			buildSession.Cancel()
		}
		go processBuildCommandMessage(msg, buildSession)
	default:
		LogInfo("ERROR: unknown message action %v", msg)
	}
	return nil
}

func processBuildCommandMessage(msg *Message, buildSession *BuildSession) {
	SetState("runtimeStatus", "Building")
	command, _ := msg.Data["data"].(map[string]interface{})
	err := buildSession.Process(MakeBuildCommand(command))
	SetState("runtimeStatus", "Idle")
	if err != nil {
		LogInfo("Error(%v) when processing message : %v", err, msg)
	}
}

func ping(send chan *Message) {
	for {
		msg := MakeMessage("ping",
			"com.thoughtworks.go.server.service.AgentRuntimeInfo",
			AgentRuntimeInfo())
		send <- msg
		time.Sleep(10 * time.Second)
	}
}
