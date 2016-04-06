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
	"github.com/gocd-contrib/gocd-golang-agent/protocol"
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
	LogInfo("set %v to %v", key, value)
	state[key] = value
}

func GetState(key string) string {
	lock.Lock()
	defer lock.Unlock()
	return state[key]
}

func GetAgentRuntimeInfo() *protocol.AgentRuntimeInfo {
	info := protocol.AgentRuntimeInfo{
		Identifier: &protocol.AgentIdentifier{
			HostName:  config.Hostname,
			IpAddress: config.IpAddress,
			Uuid:      AgentId,
		},
		BuildingInfo: &protocol.AgentBuildingInfo{
			BuildingInfo: GetState("buildLocatorForDisplay"),
			BuildLocator: GetState("buildLocator"),
		},
		RuntimeStatus:                GetState("runtimeStatus"),
		Location:                     config.WorkingDir,
		UsableSpace:                  UsableSpace(),
		OperatingSystemName:          runtime.GOOS,
		ElasticPluginId:              config.AgentAutoRegisterElasticPluginId,
		ElasticAgentId:               config.AgentAutoRegisterElasticAgentId,
		SupportsBuildCommandProtocol: true,
	}
	if cookie := GetState("cookie"); cookie != "" {
		info.Cookie = cookie
	}
	return &info
}
