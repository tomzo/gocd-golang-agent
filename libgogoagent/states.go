package libgogoagent

import (
	"os"
	"runtime"
	"sync"
)

var state = map[string]string{
	"runtimeStatus": "Idle",
}

var lock sync.Mutex

func SetState(key, value string) {
	lock.Lock()
	defer lock.Unlock()
	state[key] = value
}

func GetState(key string) string {
	lock.Lock()
	defer lock.Unlock()
	return state[key]
}

func AgentRuntimeInfo() map[string]interface{} {
	hostname, _ := os.Hostname()
	workingDir, _ := os.Getwd()
	data := make(map[string]interface{})
	data["identifier"] = map[string]string{
		"hostName":  hostname,
		"ipAddress": "127.0.0.1",
		"uuid":      ConfigGetAgentUUID()}
	data["runtimeStatus"] = GetState("runtimeStatus")
	data["buildingInfo"] = map[string]string{
		"buildingInfo": GetState("buildingInfo"),
		"buildLocator": GetState("buildLocator")}
	data["location"] = workingDir
	data["usableSpace"] = "12262604800"
	data["operatingSystemName"] = runtime.GOOS
	data["agentLauncherVersion"] = ""

	if cookie := GetState("cookie"); cookie != "" {
		data["cookie"] = cookie
	}
	return data
}
